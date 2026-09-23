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
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrContactoPropioInvalido       = errors.New("contacto propio: entrada invalida")
	ErrContactoPropioConflicto      = errors.New("contacto propio: version o correo en conflicto")
	ErrContactoPropioCommitIncierto = errors.New("contacto propio: confirmacion incierta")
	ErrContactoPropioNoDisponible   = errors.New("contacto propio: operacion no disponible")
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
type SesionContactoPropio interface {
	ResolverContactoPropio(context.Context) (domain.VinculoAutenticacionActorV2, domain.ResultadoContextoActorRegistradoV2, error)
}

type Dependencias struct {
	Sesion               SesionContactoPropio
	Identidad            *httpseguridad.ServicioIdentidad
	Revalidador          ports.RevalidadorAutenticacionActorV1
	Resolutor            domain.ResolutorContextoActorRegistradoV2
	FuenteAutorizacion   ports.FuenteAutorizacion
	Emisor               ports.AutorizadorContactoUsuario
	Seudonimizador       ports.SeudonimizadorSujetoAlmacen
	Protector            ports.ProtectorContactoUsuario
	Huellas              ports.DerivadorHuellasContactoUsuario
	PoolEscritor         *pgxpool.Pool
	RegistroOperaciones  ports.RegistroContactoUsuario // Opt-in tras Contacto3.
	Reloj                ports.Reloj
	GeneradorCorrelacion GeneradorCorrelacion
	PerfilPropioRef      string
	Motivo               domain.ReferenciaEntradaCatalogo
	AmbitosRecurso       map[string]string
}

var perfilPropioCanonico = regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,128}$`)

type Servicio struct {
	dependencias        Dependencias
	registro            ports.RegistroContactoUsuario
	registroOperaciones ports.RegistroContactoUsuario
}

func NuevoServicio(d Dependencias) (*Servicio, error) {
	for _, dependencia := range []any{d.FuenteAutorizacion, d.Emisor, d.Seudonimizador, d.Protector, d.Huellas, d.PoolEscritor, d.Reloj, d.GeneradorCorrelacion} {
		if dependenciaContactoPropioNula(dependencia) {
			return nil, ErrContactoPropioNoDisponible
		}
	}
	if dependenciaContactoPropioNula(d.Sesion) && (d.Identidad == nil || dependenciaContactoPropioNula(d.Revalidador) || dependenciaContactoPropioNula(d.Resolutor)) {
		return nil, ErrContactoPropioNoDisponible
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
	return &Servicio{dependencias: d, registro: registro, registroOperaciones: d.RegistroOperaciones}, nil
}

// Guardar sólo obtiene el sujeto de la identidad registrada. Su recibo no
// contiene el correo. El alta general y su requisito de contacto deben usar
// el estado durable de este registro; esta operación no completa un perfil.
func (s *Servicio) Guardar(ctx context.Context, correo string, versionEsperada uint64) (ports.ReciboContactoUsuario, error) {
	return s.guardar(ctx, correo, versionEsperada, "", "", nil)
}

func (s *Servicio) guardar(ctx context.Context, correo string, versionEsperada uint64, sujetoEsperado, operacionRef string, registro ports.RegistroContactoUsuario) (ports.ReciboContactoUsuario, error) {
	vacio := ports.ReciboContactoUsuario{}
	if len(correo) == 0 || len(correo) > 254 || strings.TrimSpace(correo) != correo || strings.ContainsAny(correo, "\r\n") || versionEsperada >= 1<<53-1 {
		return vacio, ErrContactoPropioInvalido
	}
	if s == nil || ctx == nil || ctx.Err() != nil || (s.dependencias.Identidad == nil && dependenciaContactoPropioNula(s.dependencias.Sesion)) || dependenciaContactoPropioNula(s.registro) {
		return vacio, ErrContactoPropioNoDisponible
	}
	d := s.dependencias
	vinculo, resultado, err := s.resolverContactoPropio(ctx)
	if err != nil || (sujetoEsperado != "" && resultado.Contexto.PersonaRef != sujetoEsperado) {
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
	if operacionRef != "" {
		if !application.ReferenciaOperacionContactoValida(operacionRef) || dependenciaContactoPropioNula(registro) {
			return vacio, ErrContactoPropioNoDisponible
		}
		recurso.Atributos = map[string]string{"contacto_operacion_ref": operacionRef}
	} else {
		registro = s.registro
	}
	accion := application.AccionAltaContactoUsuario
	if versionEsperada > 0 {
		accion = application.AccionActualizarContactoUsuario
	}
	preparador := preparadorAuditoria{fuente: d.FuenteAutorizacion, seudonimizador: d.Seudonimizador, reloj: d.Reloj, correlacion: correlacionRef, recurso: recurso}
	servicio, err := application.NuevoServicioContactoUsuario(preparador, d.Protector, d.Emisor, registro, d.Huellas)
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	recibo, err := servicio.Guardar(ctx, ports.SolicitudRegistroContactoUsuario{ContextoActor: resultado.Contexto, Contacto: contacto, OperacionRef: operacionRef, VersionEsperada: versionEsperada, FinalidadRef: FinalidadRegistro, Recurso: recurso, Audiencia: AudienciaRegistro, ResultadoContexto: resultado, SolicitudBase: domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: d.Motivo, Accion: accion, Recurso: recurso, Finalidad: FinalidadRegistro, Correlacion: correlacion}})
	if errors.Is(err, application.ErrContactoUsuarioConflicto) {
		return vacio, ErrContactoPropioConflicto
	}
	if errors.Is(err, application.ErrContactoUsuarioCommitIncierto) {
		return vacio, ErrContactoPropioCommitIncierto
	}
	if err != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	return recibo, nil
}

func (s *Servicio) resolverContactoPropio(ctx context.Context) (domain.VinculoAutenticacionActorV2, domain.ResultadoContextoActorRegistradoV2, error) {
	var vinculo domain.VinculoAutenticacionActorV2
	var resultado domain.ResultadoContextoActorRegistradoV2
	if s == nil || ctx == nil || ctx.Err() != nil || (s.dependencias.Identidad == nil && dependenciaContactoPropioNula(s.dependencias.Sesion)) {
		return vinculo, resultado, ErrContactoPropioNoDisponible
	}
	d := s.dependencias
	var err error
	if !dependenciaContactoPropioNula(d.Sesion) {
		vinculo, resultado, err = d.Sesion.ResolverContactoPropio(ctx)
	} else {
		cuenta, audit, errorIdentidad := d.Identidad.ExtraerCapsulaIdentidadPeticion(ctx)
		if errorIdentidad != nil || audit.CuentaPrivilegiada() || audit.Superficie() != httpseguridad.SuperficieExternaPersonal || cuenta.Garantia != domain.AuthAssuranceHigh || (cuenta.Metodo != domain.AuthMethodCertificate && cuenta.Metodo != domain.AuthMethodDNIe) {
			return vinculo, resultado, ErrContactoPropioNoDisponible
		}
		vinculo, resultado, err = domain.CrearVinculoAutenticacionActorV2ConResultado(ctx, d.Revalidador, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: audit.AutenticacionRef(), SesionRef: audit.SesionRef()}, d.Resolutor, domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: d.PerfilPropioRef}, d.Reloj)
	}
	datosVinculo, errorDatos := vinculo.Datos()
	if err != nil || errorDatos != nil || vinculo.ValidarPara(resultado) != nil ||
		!vinculo.VigenteEn(d.Reloj.Ahora(), resultado) ||
		datosVinculo.CuentaPrivilegiada || datosVinculo.Superficie != domain.SuperficieAutenticacionExternaPersonalV1 ||
		datosVinculo.CuentaRef != datosVinculo.CuentaOrdinariaRef ||
		resultado.Contexto.PerfilActivoRef != d.PerfilPropioRef || resultado.Contexto.Principal.AuthAssurance != domain.AuthAssuranceHigh || (resultado.Contexto.Principal.AuthMethod != domain.AuthMethodCertificate && resultado.Contexto.Principal.AuthMethod != domain.AuthMethodDNIe) {
		return domain.VinculoAutenticacionActorV2{}, domain.ResultadoContextoActorRegistradoV2{}, ErrContactoPropioNoDisponible
	}
	return vinculo, resultado, nil
}

// CompletarContactoDeAlta enlaza el recibo de contacto con la operación de
// registro propio. Sólo la autoridad del registro puede declarar Confirmado:
// una escritura de contacto, incluso recuperada por replay, no cambia por sí
// misma el estado pendiente_contacto del alta.
func (s *Servicio) CompletarContactoDeAlta(ctx context.Context, alta ports.ReferenciaAltaContactoUsuario, correo string, confirmar ports.ConfirmadorAltaContactoUsuario) (ports.ConfirmacionAltaContactoUsuario, error) {
	vacio := ports.ConfirmacionAltaContactoUsuario{}
	if ctx == nil || ctx.Err() != nil || dependenciaContactoPropioNula(confirmar) || !operacionAltaContactoCanonica.MatchString(alta.OperacionRef) || !domain.ReferenciaSujetoContactoUsuarioValida(alta.PersonaRef) {
		return vacio, ErrContactoPropioNoDisponible
	}
	if s == nil || dependenciaContactoPropioNula(s.registroOperaciones) {
		return vacio, ErrContactoPropioNoDisponible
	}
	recibo, err := s.guardar(ctx, correo, 0, alta.PersonaRef, alta.OperacionRef, s.registroOperaciones)
	if err != nil {
		return vacio, err
	}
	if recibo.SujetoRef != alta.PersonaRef || recibo.Version != 1 || recibo.EvidenciaCentral.Referencia == "" {
		return vacio, ErrContactoPropioNoDisponible
	}
	return confirmarAltaContacto(ctx, alta, recibo, confirmar)
}

func confirmarAltaContacto(ctx context.Context, alta ports.ReferenciaAltaContactoUsuario, recibo ports.ReciboContactoUsuario, confirmar ports.ConfirmadorAltaContactoUsuario) (ports.ConfirmacionAltaContactoUsuario, error) {
	vacio := ports.ConfirmacionAltaContactoUsuario{}
	if ctx == nil || ctx.Err() != nil || dependenciaContactoPropioNula(confirmar) || !operacionAltaContactoCanonica.MatchString(alta.OperacionRef) || !domain.ReferenciaSujetoContactoUsuarioValida(alta.PersonaRef) || recibo.SujetoRef != alta.PersonaRef || recibo.Version != 1 || recibo.EvidenciaCentral.Referencia == "" {
		return vacio, ErrContactoPropioNoDisponible
	}
	resultado, err := confirmar.ConfirmarAltaConContacto(ctx, alta, recibo)
	if err != nil || ctx.Err() != nil || resultado.OperacionRef != alta.OperacionRef || resultado.PersonaRef != alta.PersonaRef || resultado.Version != recibo.Version || resultado.EvidenciaRef != recibo.EvidenciaCentral.Referencia || (resultado.Confirmado && resultado.ReciboRef == "") {
		return vacio, ErrContactoPropioNoDisponible
	}
	return resultado, nil
}

var operacionAltaContactoCanonica = regexp.MustCompile(`^opr_[A-Za-z0-9_-]{22,128}$`)

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
	if ctx == nil || ctx.Err() != nil || actor.Validar() != nil || actor.PersonaRef != sujeto || modulo != usuarios.ModuleID || p.recurso.Referencia != sujeto || p.recurso.ModuloID != modulo || (accion != application.AccionAltaContactoUsuario && accion != application.AccionActualizarContactoUsuario && accion != application.AccionConsultarContactoUsuario && accion != application.AccionVersionContactoPropia && accion != application.AccionPrepararOperacionContacto && accion != application.AccionCancelarOperacionContacto && accion != application.AccionListarOperacionesContacto && accion != application.AccionDetalleOperacionContacto) || version == 0 || version > 1<<53-1 || p.correlacion == "" {
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
