package bootstrap

import (
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Las versiones del rol RRHH de Bolsa van de cuatro en cuatro desde la 5:
// base (5-8), con la consulta documental (9-12) y, además, con la consulta de
// los datos de contacto completos (13-16). Dentro de cada grupo, la posición
// indica la política de ofertas (+1) y la reincorporación de titulares (+2).
// Las de los grupos segundo y tercero solo se publican por provisión aprobada.
const (
	saltoProvisionDocumentalBolsa    = 4
	saltoProvisionDatosContactoBolsa = 8
)

func versionRolBolsaConPoliticaOfertas(v int) bool {
	return v >= 5 && v <= 16 && (v-5)%2 == 1
}

func versionRolBolsaConReincorporacion(v int) bool {
	return v >= 5 && v <= 16 && ((v-5)/2)%2 == 1
}

// objetivoProvisionRRHHBolsa es la asignación que la provisión publica sobre
// la ya publicada: misma asignación, versión siguiente y rol objetivo.
func objetivoProvisionRRHHBolsa(publicada dominiovec.InstantaneaAutorizacion, datos dominiovec.DatosVinculoAutenticacionActorV2,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, version int,
) (dominiovec.InstantaneaAutorizacion, error) {
	if soporte == nil {
		return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	objetivo, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, ahora, version)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	objetivo.AsignacionPerfil.AsignacionID = publicada.AsignacionPerfil.AsignacionID
	objetivo.AsignacionPerfil.Version = publicada.AsignacionPerfil.Version + 1
	return objetivo, nil
}

// huellasProvisionRRHHBolsa son las dos huellas que la aprobación del DBA
// debe repetir: la asignación publicada y la que la sustituirá.
func huellasProvisionRRHHBolsa(publicada dominiovec.InstantaneaAutorizacion, datos dominiovec.DatosVinculoAutenticacionActorV2,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, version int,
) (string, string, error) {
	preimagen, err := publicada.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return "", "", err
	}
	objetivo, err := objetivoProvisionRRHHBolsa(publicada, datos, soporte, ahora, version)
	if err != nil {
		return "", "", err
	}
	huella, err := objetivo.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return "", "", err
	}
	return preimagen, huella, nil
}
