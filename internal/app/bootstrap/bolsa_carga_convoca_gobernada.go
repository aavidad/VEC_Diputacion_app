package bootstrap

import (
	"context"
	"errors"
	"log"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autoridadesCargaConvocaPostgreSQL struct {
	fuente  *pgxpool.Pool
	motivos *pgxpool.Pool
}

func procedenciaCargaConvocaGobernada(publicada instantaneaPublicadaDesarrollo,
	autoridad autoridadInicialBorradorLlamamientoBolsaDesarrollo) bool {
	propia, ok := autoridad.(*autoridadPostgreSQLDesarrollo)
	if !ok {
		return true // Los dobles de prueba no suministran actos PostgreSQL.
	}
	return publicada.actoAsignacion != "" && publicada.actoControl != "" && publicada.actualizadaPor != "" &&
		publicada.actoAsignacion != propia.actoAsignacion && publicada.actoControl != propia.actoControlRol &&
		publicada.actualizadaPor != "identidad:desarrollo:no-autoritativa"
}

type fuenteCargaConvocaAcotada struct {
	base  puertosvec.FuenteAutorizacion
	ancla dominiovec.InstantaneaAutorizacion
}

func (f fuenteCargaConvocaAcotada) ObtenerInstantaneaAutorizacion(ctx context.Context,
	principal, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
	vacia := dominiovec.InstantaneaAutorizacion{}
	if ctx == nil || ctx.Err() != nil || dependenciaAutorizacionComunDesarrolloNula(f.base) ||
		principal != f.ancla.AsignacionPerfil.PrincipalID ||
		perfil != f.ancla.AsignacionPerfil.PerfilActivoRef {
		return vacia, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	i, err := f.base.ObtenerInstantaneaAutorizacion(ctx, principal, perfil)
	if err != nil {
		return vacia, err
	}
	if i.Validar() != nil || i.AsignacionPerfil.AsignacionID != f.ancla.AsignacionPerfil.AsignacionID ||
		i.AsignacionPerfil.PrincipalID != principal || i.AsignacionPerfil.PerfilActivoRef != perfil ||
		!reflect.DeepEqual(i.AsignacionPerfil.Ambitos, f.ancla.AsignacionPerfil.Ambitos) ||
		i.VersionRol.RolID != f.ancla.VersionRol.RolID ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" ||
		!concesionCargaConvocaExacta(i.VersionRol.Concesiones) {
		return vacia, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	for _, requerida := range f.ancla.VersionRol.Concesiones {
		encontrada := false
		for _, actual := range i.VersionRol.Concesiones {
			if reflect.DeepEqual(actual, requerida) {
				encontrada = true
				break
			}
		}
		if !encontrada {
			return vacia, puertosvec.ErrFuenteAutorizacionNoDisponible
		}
	}
	return i, nil
}

// B1 sólo consume una concesión publicada por el gobierno de autorización.
// El arranque nunca crea una versión de rol ni mueve una asignación para B1.
func concesionCargaConvocaExacta(concesiones []dominiovec.ConcesionRol) bool {
	coincidencias := 0
	for _, c := range concesiones {
		if c.Accion != puertosbolsa.AccionConfirmarCargaConvoca {
			continue
		}
		if c.ModuloID != puertosbolsa.ModuloCargaConvoca || c.TipoRecurso != puertosbolsa.TipoRecursoCargaConvoca ||
			len(c.Finalidades) != 1 || c.Finalidades[0] != puertosbolsa.FinalidadConfirmarCargaConvoca ||
			c.GarantiaMinima != dominiovec.AuthAssuranceHigh || len(c.CamposPermitidos) != 0 || len(c.Obligaciones) != 0 {
			return false
		}
		coincidencias++
	}
	return coincidencias == 1
}

// La compatibilidad comprueba la misma identidad y el conjunto anterior de
// concesiones. Metadatos del acto publicado (autor, fecha, versión y huellas)
// pertenecen a la autoridad central y no se sustituyen por la semilla local.
func instantaneaBolsaCargaConvocaCompatible(i dominiovec.InstantaneaAutorizacion,
	vinculo dominiovec.DatosVinculoAutenticacionActorV2, soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, versionBase int) bool {
	if soporte == nil || i.Validar() != nil || i.VersionRol.Version < versionBase ||
		i.VersionRol.RolID != "tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo" ||
		i.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		!i.AsignacionPerfil.VigenteEn(ahora) ||
		i.AsignacionPerfil.PrincipalID != vinculo.PrincipalID ||
		i.AsignacionPerfil.PerfilActivoRef != vinculo.PerfilActivoRef ||
		!concesionCargaConvocaExacta(i.VersionRol.Concesiones) {
		return false
	}
	base, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		vinculo.PrincipalID, vinculo.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, versionBase)
	if err != nil {
		log.Printf("bolsa.carga_convoca.instantanea_base: %T", err)
		return false
	}
	if i.AsignacionPerfil.AsignacionID != base.AsignacionPerfil.AsignacionID ||
		!reflect.DeepEqual(i.AsignacionPerfil.Ambitos, base.AsignacionPerfil.Ambitos) {
		return false
	}
	// Las capacidades publicadas previamente siguen presentes con sus campos,
	// obligaciones y finalidades exactos. No se exige igualdad de la versión
	// completa porque otras concesiones gobernadas pueden coexistir.
	for _, requerida := range base.VersionRol.Concesiones {
		encontrada := false
		for _, actual := range i.VersionRol.Concesiones {
			if reflect.DeepEqual(actual, requerida) {
				encontrada = true
				break
			}
		}
		if !encontrada {
			return false
		}
	}
	return true
}

// La fuente se consulta al iniciar la ruta y el PDP vuelve a consultarla en
// cada petición. Si desaparece o caduca el grant, la operación queda denegada.
func politicaCargaConvocaGobernada(ctx context.Context, fuente puertosvec.FuenteAutorizacion,
	registro puertosvec.RegistroConcesionesCandidatasAutorizacionLigadaV3,
	denegaciones puertosvec.RegistroDenegacionesAutorizacionLigadaV3,
	motivos puertosvec.ValidadorReferenciaMotivoAutorizacionV2,
	ancla dominiovec.InstantaneaAutorizacion, ahora time.Time) (politicaAutorizacionSolicitudLigadaV3Desarrollo, bool, error) {
	vacia := politicaAutorizacionSolicitudLigadaV3Desarrollo{}
	principal, perfil := ancla.AsignacionPerfil.PrincipalID, ancla.AsignacionPerfil.PerfilActivoRef
	if ctx == nil || ctx.Err() != nil || dependenciaAutorizacionComunDesarrolloNula(fuente) ||
		dependenciaAutorizacionComunDesarrolloNula(registro) || dependenciaAutorizacionComunDesarrolloNula(denegaciones) ||
		dependenciaAutorizacionComunDesarrolloNula(motivos) || principal == "" || perfil == "" ||
		ancla.Validar() != nil || ancla.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		ancla.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		ancla.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		!concesionCargaConvocaExacta(ancla.VersionRol.Concesiones) {
		return vacia, false, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	acotada := fuenteCargaConvocaAcotada{base: fuente, ancla: ancla}
	i, err := acotada.ObtenerInstantaneaAutorizacion(ctx, principal, perfil)
	if err != nil {
		return vacia, false, err
	}
	if i.Validar() != nil || i.AsignacionPerfil.PrincipalID != principal || i.AsignacionPerfil.PerfilActivoRef != perfil ||
		!i.AsignacionPerfil.VigenteEn(ahora) || i.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		i.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		!concesionCargaConvocaExacta(i.VersionRol.Concesiones) {
		return vacia, false, nil
	}
	p, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(acotada, registro, denegaciones, motivos)
	return p, err == nil, err
}

func politicaCargaConvocaPostgreSQL(ctx context.Context, fuentePool, motivosPool, registroPool *pgxpool.Pool,
	ancla dominiovec.InstantaneaAutorizacion, ahora time.Time) (politicaAutorizacionSolicitudLigadaV3Desarrollo, bool, error) {
	vacia := politicaAutorizacionSolicitudLigadaV3Desarrollo{}
	if fuentePool == nil || motivosPool == nil || registroPool == nil || fuentePool == motivosPool ||
		fuentePool == registroPool || motivosPool == registroPool {
		return vacia, false, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	fuente, err := postgresvec.NuevoAlmacenAutorizacion(fuentePool)
	if err != nil {
		return vacia, false, err
	}
	registro, err := postgresvec.NuevoAlmacenAutorizacion(registroPool)
	if err != nil {
		return vacia, false, err
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool,
		motivoConfirmarCargaConvocaBolsaDesarrollo().CatalogoID)
	if err != nil {
		return vacia, false, err
	}
	return politicaCargaConvocaGobernada(ctx, fuente, registro, registro, validador, ancla, ahora)
}

// Esta comprobación permite declarar ambas fronteras sólo cuando la
// asignación ya existe. Una ausencia mantiene B1 fuera del catálogo.
func permiteMontarCargaConvocaPostgreSQL(ctx context.Context, fuentePool *pgxpool.Pool,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, versionBase int) (bool, error) {
	if fuentePool == nil || ctx == nil || ctx.Err() != nil || soporte == nil || soporte.soporteCanal == nil {
		return false, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	vinculo, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		return false, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	principal, perfil := vinculo.PrincipalID, vinculo.PerfilActivoRef
	fuente, err := postgresvec.NuevoAlmacenAutorizacion(fuentePool)
	if err != nil {
		return false, err
	}
	i, err := fuente.ObtenerInstantaneaAutorizacion(ctx, principal, perfil)
	if errors.Is(err, puertosvec.ErrAsignacionPerfilNoEncontrada) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return instantaneaBolsaCargaConvocaCompatible(i, vinculo, soporte, ahora, versionBase), nil
}
