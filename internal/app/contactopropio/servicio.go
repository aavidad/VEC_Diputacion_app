// Package contactopropio compone el registro del correo propio con las
// autoridades de identidad, autorización y persistencia ya existentes.
package contactopropio

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/usuarios"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrContactoPropioInvalido     = errors.New("contacto propio: entrada invalida")
	ErrContactoPropioNoDisponible = errors.New("contacto propio: operacion no disponible")
)

const (
	FinalidadRegistro      = "gestion_contacto_propio"
	AudienciaRegistro      = "vec.contacto_usuario.registro.v1"
	ambitoAuditoriaCentral = "bolsa_registro_accesos_t13"
)

// GeneradorCorrelacion tiene la firma nativa del generador V2; no acepta una
// correlación declarada en la petición ni emite capacidades de autorización.
type GeneradorCorrelacion interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

// Dependencias exige autoridades existentes y un pool del rol escritor
// nominal, nunca owner. PerfilPropioRef es un selector concreto configurado
// por el servidor; el resolutor registrado debe vincularlo a la cuenta de la
// cápsula. AmbitosRecurso procede de la configuración gobernada y el PDP
// comprueba su cobertura. Este constructor no publica roles ni los concede.
type Dependencias struct {
	Identidad            *httpseguridad.ServicioIdentidad
	Revalidador          ports.RevalidadorAutenticacionActorV1
	Resolutor            domain.ResolutorContextoActorRegistradoV2
	FuenteAutorizacion   ports.FuenteAutorizacion
	Emisor               *confianzaatestacion.EmisorMaterialAutorizacionAtestadaV3
	Seudonimizador       ports.SeudonimizadorSujetoAlmacen
	Protector            ports.ProtectorContactoUsuario
	PoolEscritor         *pgxpool.Pool
	Reloj                ports.Reloj
	GeneradorCorrelacion GeneradorCorrelacion
	PerfilPropioRef      string
	Motivo               domain.ReferenciaEntradaCatalogo
	AmbitosRecurso       map[string]string
}

var perfilPropioCanonico = regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,128}$`)

type Servicio struct {
	dependencias Dependencias
	registro     ports.RegistroContactoUsuario
}

func NuevoServicio(d Dependencias) (*Servicio, error) {
	for _, dependencia := range []any{d.Identidad, d.Revalidador, d.Resolutor, d.FuenteAutorizacion, d.Emisor, d.Seudonimizador, d.Protector, d.PoolEscritor, d.Reloj, d.GeneradorCorrelacion} {
		if dependenciaContactoPropioNula(dependencia) {
			return nil, ErrContactoPropioNoDisponible
		}
	}
	if !perfilPropioCanonico.MatchString(d.PerfilPropioRef) || d.Motivo.Validar() != nil || len(d.AmbitosRecurso) == 0 {
		return nil, ErrContactoPropioNoDisponible
	}
	d.AmbitosRecurso = clonarAmbitos(d.AmbitosRecurso)
	recurso := domain.RecursoAutorizable{Referencia: "contacto", ModuloID: usuarios.ModuleID, Tipo: "contacto_usuario", Ambitos: d.AmbitosRecurso}
	if recurso.Validar() != nil {
		return nil, ErrContactoPropioNoDisponible
	}
	registro, err := postgres.NuevoRegistroContactoUsuarioPostgreSQL(d.PoolEscritor)
	if err != nil {
		return nil, ErrContactoPropioNoDisponible
	}
	return &Servicio{dependencias: d, registro: registro}, nil
}

// Guardar sólo obtiene el sujeto de la identidad registrada. Su recibo no
// contiene el correo. El alta general y su requisito de contacto deben usar
// el estado durable de este registro; esta operación no completa un perfil.
func (s *Servicio) Guardar(ctx context.Context, correo string, versionEsperada uint64) (ports.ReciboContactoUsuario, error) {
	vacio := ports.ReciboContactoUsuario{}
	if len(correo) == 0 || len(correo) > 254 || strings.TrimSpace(correo) != correo || strings.ContainsAny(correo, "\r\n") || versionEsperada >= 1<<53-1 {
		return vacio, ErrContactoPropioInvalido
	}
	if s == nil || ctx == nil || ctx.Err() != nil || s.dependencias.Identidad == nil || dependenciaContactoPropioNula(s.registro) {
		return vacio, ErrContactoPropioNoDisponible
	}
	d := s.dependencias
	cuenta, audit, err := d.Identidad.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil || audit.CuentaPrivilegiada() || audit.Superficie() != httpseguridad.SuperficieExternaPersonal || cuenta.Garantia != domain.AuthAssuranceHigh || (cuenta.Metodo != domain.AuthMethodCertificate && cuenta.Metodo != domain.AuthMethodDNIe) {
		return vacio, ErrContactoPropioNoDisponible
	}
	vinculo, resultado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(ctx, d.Revalidador, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: audit.AutenticacionRef(), SesionRef: audit.SesionRef()}, d.Resolutor, domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: d.PerfilPropioRef}, d.Reloj)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	contacto, err := domain.NuevoContactoUsuario(resultado.Contexto.PersonaRef, correo, versionEsperada+1)
	if err != nil {
		return vacio, ErrContactoPropioInvalido
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, d.GeneradorCorrelacion)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	recurso := domain.RecursoAutorizable{Referencia: resultado.Contexto.PersonaRef, ModuloID: usuarios.ModuleID, Tipo: "contacto_usuario", Ambitos: clonarAmbitos(d.AmbitosRecurso)}
	accion := application.AccionAltaContactoUsuario
	if versionEsperada > 0 {
		accion = application.AccionActualizarContactoUsuario
	}
	preparador := preparadorAuditoria{fuente: d.FuenteAutorizacion, seudonimizador: d.Seudonimizador, reloj: d.Reloj, correlacion: correlacionRef, recurso: recurso}
	servicio, err := application.NuevoServicioContactoUsuario(preparador, d.Protector, d.Emisor, s.registro)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	recibo, err := servicio.Guardar(ctx, ports.SolicitudRegistroContactoUsuario{ContextoActor: resultado.Contexto, Contacto: contacto, VersionEsperada: versionEsperada, FinalidadRef: FinalidadRegistro, Recurso: recurso, Audiencia: AudienciaRegistro, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: d.Motivo, Accion: accion, Recurso: recurso, Finalidad: FinalidadRegistro, Correlacion: correlacion}})
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	return recibo, nil
}

// preparadorAuditoria adapta la fuente central y el HMAC existente. La firma
// y AuthorizationRef finales sólo los produce T13 tras consumir la decisión.
type preparadorAuditoria struct {
	fuente         ports.FuenteAutorizacion
	seudonimizador ports.SeudonimizadorSujetoAlmacen
	reloj          ports.Reloj
	correlacion    string
	recurso        domain.RecursoAutorizable
}

func (p preparadorAuditoria) PrepararAuditoriaContactoUsuario(ctx context.Context, actor domain.ContextoActor, accion, modulo, sujeto string, version uint64) (domain.AuditEntry, error) {
	vacio := domain.AuditEntry{}
	if ctx == nil || ctx.Err() != nil || actor.Validar() != nil || actor.PersonaRef != sujeto || modulo != usuarios.ModuleID || p.recurso.Referencia != sujeto || p.recurso.ModuloID != modulo || (accion != application.AccionAltaContactoUsuario && accion != application.AccionActualizarContactoUsuario) || version == 0 || version > 1<<53-1 || p.correlacion == "" {
		return vacio, ErrContactoPropioNoDisponible
	}
	instantanea, err := p.fuente.ObtenerInstantaneaAutorizacion(ctx, actor.Principal.ID, actor.PerfilActivoRef)
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil || instantanea.Validar() != nil || instantanea.AsignacionPerfil.PrincipalID != actor.Principal.ID || instantanea.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef || !instantanea.AsignacionPerfil.VigenteEn(ahora) || !instantanea.AsignacionPerfil.Cubre(p.recurso) || instantanea.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		return vacio, ErrContactoPropioNoDisponible
	}
	solicitud, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(actor.Principal.ID, ambitoAuditoriaCentral)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	actorHMAC, err := p.seudonimizador.SeudonimizarSujetoAlmacen(ctx, solicitud)
	if err != nil || ctx.Err() != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	return domain.AuditEntry{ActorID: actorHMAC, ActorProfile: actor.PerfilActivoRef, ActorRoles: []string{instantanea.VersionRol.Referencia()}, AuthMethod: actor.Principal.AuthMethod, AuthAssurance: actor.Principal.AuthAssurance, Purpose: FinalidadRegistro, Action: accion, ModuleID: modulo, SubjectRef: sujeto, ObjectVersion: int(version), Result: "accepted", CorrelationRef: p.correlacion, OccurredAt: ahora}, nil
}

func clonarAmbitos(origen map[string]string) map[string]string {
	copia := make(map[string]string, len(origen))
	for clave, valor := range origen {
		copia[clave] = valor
	}
	return copia
}
