package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// FuenteCapacidades entrega únicamente la consulta de capacidades. No abre las
// seis lecturas heredadas ni permite construir una autoridad de actos.
type FuenteCapacidades struct {
	pool                  conexion
	emisor                Emisor
	fuente                ports.FuenteAutorizacion
	catalogo              ports.CatalogoRolesAdministrables
	reloj                 ports.Reloj
	intentos              ports.RegistradorIntentosAuditoria
	configuracionIntentos ConfiguracionIntentosCapacidades
}

var _ api.FuenteLecturas = (*FuenteCapacidades)(nil)

// NuevaFuenteCapacidades permanece cerrada hasta acreditar el preflight del
// registrador común, la composición privada y el LOGIN lector acotado a
// consultar_capacidades_admin_v1, sin permisos de actos. El consumo del puerto
// de intentos no acredita esas dependencias ni su instalación.
func NuevaFuenteCapacidades(ctx context.Context, pool *pgxpool.Pool, emisor Emisor,
	fuente ports.FuenteAutorizacion, catalogo ports.CatalogoRolesAdministrables, reloj ports.Reloj,
) (*FuenteCapacidades, error) {
	if ctx == nil || pool == nil || ausente(emisor) || ausente(fuente) || ausente(catalogo) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

type capacidadesJSON struct {
	api.Capacidades
	PerfilActivoRef     string    `json:"perfil_activo_ref"`
	AsignacionPerfilRef string    `json:"asignacion_perfil_ref"`
	CorrelacionRef      string    `json:"correlacion_ref"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	RegistradaEn        time.Time `json:"registrada_en"`
}

func (f *FuenteCapacidades) consultarCapacidades(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles) (api.Capacidades, error) {
	if ctx == nil || f == nil || ausente(f.pool) || ausente(f.emisor) || ausente(f.fuente) || ausente(f.catalogo) || ausente(f.reloj) {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return api.Capacidades{}, err
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	// Representación V3 de la misma entropía creada por la frontera común.
	// El canal técnico conserva los 32 hex originales; no hay otra generación.
	correlacion = "correlacion_" + correlacion
	ahora := f.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return api.Capacidades{}, domain.ErrAutorizacionDenegada
	}
	instantanea, err := f.fuente.ObtenerInstantaneaAutorizacion(ctx, actor.PersonaRef, actor.PerfilActivoRef)
	if err != nil {
		return api.Capacidades{}, traducirErrorLectura(ctx, err)
	}
	if instantanea.Validar() != nil {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if instantanea.AsignacionPerfil.PrincipalID != actor.PersonaRef || instantanea.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef ||
		!instantanea.AsignacionPerfil.VigenteEn(ahora) || instantanea.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		ahora.Before(instantanea.VersionRol.PublicadaEn) || ahora.Before(instantanea.ControlVigenciaVersionRol.ActualizadoEn) ||
		instantanea.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		return api.Capacidades{}, domain.ErrAutorizacionDenegada
	}
	rol, err := f.catalogo.ResolverRolAdministrable(ctx, instantanea.VersionRol.Referencia())
	if err != nil {
		return api.Capacidades{}, traducirErrorLectura(ctx, err)
	}
	huella, err := instantanea.VersionRol.HuellaSHA256()
	if err != nil || rol.ValidarEn(ahora) != nil || rol.VersionRef != instantanea.VersionRol.Referencia() || rol.HuellaSHA256 != huella ||
		rol.Clase != domain.ClaseControlPerfilAdministrador || rol.CategoriaAdmin != "aplicacion" {
		return api.Capacidades{}, domain.ErrAutorizacionDenegada
	}
	material, err := json.Marshal(struct {
		lecturaJSON
		AsignacionPerfilRef string `json:"asignacion_perfil_ref"`
		CorrelacionRef      string `json:"correlacion_ref"`
	}{lecturaJSON{Esquema: "administracion_perfiles_lectura_v1", Consulta: "capacidades", OperacionRef: actor.PerfilActivoRef,
		ActorPersonaRef: actor.PersonaRef, ActorPerfilRef: actor.PerfilActivoRef, Limite: limiteLecturasADMIN}, instantanea.AsignacionPerfil.Referencia(), correlacion})
	if err != nil {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	efecto := Efecto{Accion: "administracion.perfiles.consultar", Audiencia: "vec_autorizacion.administracion_perfiles.lectura.capacidades.v1",
		Referencia: actor.PerfilActivoRef, Material: material, CorrelacionAccesoRef: correlacion}
	var salida capacidadesJSON
	respuestaValidada := false
	err = ejecutarConsumoADMIN(ctx, f.pool, f.emisor, f.reloj, actor, evidencia, instantanea, efecto, capacidadesLecturaSQL, func(b []byte) error {
		if decodificar(b, &salida) != nil || salida.Version != "1" || salida.ActorPersonaRef != actor.PersonaRef ||
			salida.PerfilActivoRef != actor.PerfilActivoRef || salida.AsignacionPerfilRef != instantanea.AsignacionPerfil.Referencia() ||
			salida.CorrelacionRef != correlacion || !referenciaAuditoriaCapacidades(salida.AuditoriaRef) ||
			!instantePersistible(salida.RegistradaEn) || salida.RegistradaEn.After(f.reloj.Ahora()) ||
			salida.Acciones == nil || len(salida.Acciones) != 0 {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		respuestaValidada = true
		return nil
	})
	if err != nil {
		// Después de validar la respuesta solo quedan cancelación y COMMIT.
		// Un fallo de COMMIT nunca acredita una denegación positiva, aunque
		// el traductor común reconozca su código como rechazo de acceso.
		if respuestaValidada && ctx.Err() == nil {
			return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return api.Capacidades{}, traducirErrorLectura(ctx, err)
	}
	return salida.Capacidades, nil
}

func referenciaAuditoriaCapacidades(ref string) bool {
	if len(ref) != len("aud_v3_")+32 || ref[:len("aud_v3_")] != "aud_v3_" {
		return false
	}
	return domain.EsCorrelacionTecnicaValida(ref[len("aud_v3_"):])
}

func (*FuenteCapacidades) BuscarPersonas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, api.ConsultaPersonas) (api.PaginaPersonas, error) {
	return api.PaginaPersonas{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (*FuenteCapacidades) ConsultarPersona(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (api.FichaPersona, error) {
	return api.FichaPersona{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (*FuenteCapacidades) ListarRoles(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (api.Roles, error) {
	return api.Roles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (*FuenteCapacidades) ListarPropuestas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (api.PaginaPropuestas, error) {
	return api.PaginaPropuestas{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (*FuenteCapacidades) ConsultarPropuesta(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (api.Propuesta, error) {
	return api.Propuesta{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (*FuenteCapacidades) ConsultarRecibo(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (domain.ReciboAdministracionPerfiles, error) {
	return domain.ReciboAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
