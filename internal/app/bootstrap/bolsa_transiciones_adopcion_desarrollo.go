package bootstrap

import (
	"log/slog"
	"reflect"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Una versión gobernada puede sumar la lectura S6 y, si existe, su edición.
// Se adopta intacta; esta comprobación no construye ni publica concesiones.
func instantaneaBolsaS6Compatible(publicada dominiovec.InstantaneaAutorizacion,
	vinculo dominiovec.DatosVinculoAutenticacionActorV2,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, versionBase int) bool {
	if soporte == nil || publicada.Validar() != nil || !publicada.AsignacionPerfil.VigenteEn(ahora) ||
		publicada.AsignacionPerfil.PrincipalID != vinculo.PrincipalID ||
		publicada.AsignacionPerfil.PerfilActivoRef != vinculo.PerfilActivoRef ||
		publicada.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		publicada.VersionRol.Estado != dominiovec.EstadoVersionRolPublicada ||
		publicada.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		publicada.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada ||
		publicada.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return false
	}
	for _, baseVersion := range []int{versionBase, versionBase + saltoProvisionDocumentalBolsa, versionBase + saltoProvisionDatosContactoBolsa} {
		base, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			vinculo.PrincipalID, vinculo.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, baseVersion)
		if err != nil {
			slog.Warn("bolsa transiciones: instantánea base no disponible", "causa", causaFalloPostgreSQLCTDesarrollo(err))
			continue
		}
		if publicada.VersionRol.Version <= baseVersion ||
			publicada.VersionRol.RolID != base.VersionRol.RolID ||
			publicada.AsignacionPerfil.AsignacionID != base.AsignacionPerfil.AsignacionID ||
			!reflect.DeepEqual(publicada.AsignacionPerfil.Ambitos, base.AsignacionPerfil.Ambitos) {
			continue
		}
		restantes := append([]dominiovec.ConcesionRol(nil), publicada.VersionRol.Concesiones...)
		completa := true
		for _, requerida := range base.VersionRol.Concesiones {
			indice := -1
			for i, actual := range restantes {
				if reflect.DeepEqual(actual, requerida) {
					indice = i
					break
				}
			}
			if indice < 0 {
				completa = false
				break
			}
			restantes = append(restantes[:indice], restantes[indice+1:]...)
		}
		if len(restantes) < 1 || len(restantes) > 2 {
			continue
		}
		for _, restante := range restantes {
			if restante.Accion != puertosbolsa.AccionConsultarTransicionesSituacion &&
				restante.Accion != puertosbolsa.AccionPublicarTransicionesSituacion {
				completa = false
			}
		}
		if completa && concesionesTransicionesS6Exactas(restantes) {
			return true
		}
	}
	return false
}
