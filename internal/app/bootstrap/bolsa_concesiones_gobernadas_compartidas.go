package bootstrap

import (
	"log/slog"
	"reflect"
	"slices"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Solo adopta una autoridad publicada que conserva asignación, ámbitos y
// concesiones base. Cada consumidor decide cuáles son sus extras nominales.
func instantaneaBolsaGobernadaCompatible(publicada dominiovec.InstantaneaAutorizacion,
	vinculo dominiovec.DatosVinculoAutenticacionActorV2,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, versionBase int,
	adicionales func([]dominiovec.ConcesionRol) bool) bool {
	if soporte == nil || adicionales == nil || publicada.Validar() != nil || !publicada.AsignacionPerfil.VigenteEn(ahora) ||
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
		if completa && adicionales(restantes) {
			return true
		}
	}
	return false
}

func concesionesAdicionalesS6Exactas(restantes []dominiovec.ConcesionRol) bool {
	if len(restantes) < 1 || len(restantes) > 2 {
		return false
	}
	for _, c := range restantes {
		if c.Accion != puertosbolsa.AccionConsultarTransicionesSituacion &&
			c.Accion != puertosbolsa.AccionPublicarTransicionesSituacion {
			return false
		}
	}
	return concesionesTransicionesS6Exactas(restantes)
}

func concesionesTransicionesS6Exactas(concesiones []dominiovec.ConcesionRol) bool {
	consultas, publicaciones := 0, 0
	for _, c := range concesiones {
		switch c.Accion {
		case puertosbolsa.AccionConsultarTransicionesSituacion:
			if c.ModuloID != "bolsa" || c.TipoRecurso != puertosbolsa.TipoRecursoGobiernoTransicionesSituacion ||
				!slices.Equal(c.Finalidades, []string{puertosbolsa.FinalidadConsultaTransicionesSituacion}) ||
				!slices.Equal(c.CamposPermitidos, []string{puertosbolsa.CampoConsultaTransicionesSituacion}) ||
				(len(c.Obligaciones) != 0 && !slices.Equal(c.Obligaciones, []string{"auditar"})) ||
				c.GarantiaMinima != dominiovec.AuthAssuranceHigh {
				return false
			}
			consultas++
		case puertosbolsa.AccionPublicarTransicionesSituacion:
			if c.ModuloID != "bolsa" || c.TipoRecurso != puertosbolsa.TipoRecursoGobiernoTransicionesSituacion ||
				!slices.Equal(c.Finalidades, []string{puertosbolsa.FinalidadGobiernoTransicionesSituacion}) ||
				len(c.CamposPermitidos) != 0 || len(c.Obligaciones) != 0 ||
				c.GarantiaMinima != dominiovec.AuthAssuranceHigh {
				return false
			}
			publicaciones++
		}
	}
	return consultas == 1 && publicaciones <= 1
}
