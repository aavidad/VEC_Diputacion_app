package bootstrap

import (
	"context"
	"log/slog"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// fuenteTransicionesS6 conserva la asignación y los ámbitos originales,
// pero relee en PostgreSQL cada versión publicada. No usa la semilla B-BACK.
type fuenteTransicionesS6 struct {
	base  puertosvec.FuenteAutorizacion
	ancla dominiovec.InstantaneaAutorizacion
}

func (f fuenteTransicionesS6) ObtenerInstantaneaAutorizacion(ctx context.Context, principal, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
	vacia := dominiovec.InstantaneaAutorizacion{}
	if ctx == nil || ctx.Err() != nil || f.base == nil || principal != f.ancla.AsignacionPerfil.PrincipalID ||
		perfil != f.ancla.AsignacionPerfil.PerfilActivoRef {
		return vacia, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	i, err := f.base.ObtenerInstantaneaAutorizacion(ctx, principal, perfil)
	if err != nil || i.Validar() != nil ||
		i.AsignacionPerfil.AsignacionID != f.ancla.AsignacionPerfil.AsignacionID ||
		i.VersionRol.RolID != f.ancla.VersionRol.RolID ||
		!reflect.DeepEqual(i.AsignacionPerfil.Ambitos, f.ancla.AsignacionPerfil.Ambitos) ||
		i.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" ||
		!concesionesTransicionesS6Exactas(i.VersionRol.Concesiones) {
		return vacia, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return i, nil
}

// politicaTransicionesS6PostgreSQL solo adopta autoridad, motivo y rol ya
// publicados. Registro de candidatas/denegaciones usa el LOGIN común propio.
func politicaTransicionesS6PostgreSQL(ctx context.Context, fuentePool, motivosPool, registroPool *pgxpool.Pool,
	soporte *soporteSesionBorradorBolsaDesarrollo, motivo dominiovec.ReferenciaEntradaCatalogo,
	ahora time.Time) (politicaAutorizacionSolicitudLigadaV3Desarrollo, bool, error) {
	vacia := politicaAutorizacionSolicitudLigadaV3Desarrollo{}
	if ctx == nil || ctx.Err() != nil || fuentePool == nil || motivosPool == nil || registroPool == nil ||
		fuentePool == motivosPool || fuentePool == registroPool || motivosPool == registroPool ||
		soporte == nil || soporte.soporteCanal == nil || motivo.Validar() != nil {
		return vacia, false, nil
	}
	vinculo, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		slog.Warn("bolsa transiciones: política no disponible", "etapa", "vinculo", "causa", causaFalloPostgreSQLCTDesarrollo(err))
		return vacia, false, err
	}
	fuente, err := postgresvec.NuevoAlmacenAutorizacion(fuentePool)
	if err != nil {
		return vacia, false, err
	}
	registro, err := postgresvec.NuevoAlmacenAutorizacion(registroPool)
	if err != nil {
		return vacia, false, err
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool, motivo.CatalogoID)
	if err != nil {
		return vacia, false, err
	}
	if err := validador.ValidarReferenciaMotivoAutorizacionV2(ctx, motivo, ahora); err != nil {
		slog.Warn("bolsa transiciones: política no disponible", "etapa", "motivo", "causa", causaFalloPostgreSQLCTDesarrollo(err))
		return vacia, false, err
	}
	i, err := fuente.ObtenerInstantaneaAutorizacion(ctx, vinculo.PrincipalID, vinculo.PerfilActivoRef)
	if err != nil {
		slog.Warn("bolsa transiciones: política no disponible", "etapa", "instantanea", "causa", causaFalloPostgreSQLCTDesarrollo(err))
		return vacia, false, err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: puertosbolsa.RecursoGobiernoTransicionesSituacion,
		ModuloID: "bolsa", Tipo: puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		Ambitos: map[string]string{"unidad_ref": soporte.unidadRef, "ambito_ref": soporte.ambitoRef}}
	if i.Validar() != nil || !i.AsignacionPerfil.VigenteEn(ahora) ||
		i.AsignacionPerfil.PrincipalID != vinculo.PrincipalID || i.AsignacionPerfil.PerfilActivoRef != vinculo.PerfilActivoRef ||
		!i.AsignacionPerfil.Cubre(recurso) || i.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		!concesionesTransicionesS6Exactas(i.VersionRol.Concesiones) {
		return vacia, false, nil
	}
	p, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(fuenteTransicionesS6{base: fuente, ancla: i},
		registro, registro, validador)
	return p, err == nil, err
}
