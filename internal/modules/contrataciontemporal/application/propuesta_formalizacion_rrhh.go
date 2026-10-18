package application

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// PropuestaFormalizacionBorrador conserva el antecedente de la propuesta
// cuando la misma historia ya registra su resolución y la anotación
// administrativa posterior. Esas actuaciones no acreditan firma ni eficacia.
func PropuestaFormalizacionBorrador(hitos []ports.HitoExpedienteRRHH) (ports.HitoExpedienteRRHH, bool) {
	indice := len(hitos) - 1
	switch {
	case indice >= 0 && hitos[indice].AccionClave == "registrar_propuesta_formalizacion":
	case indice >= 1 && esResolucionFormalizacionPosterior(hitos[indice]):
		indice--
	case indice >= 2 &&
		esAnotacionAdministrativaPosterior(hitos[indice]) &&
		esResolucionFormalizacionPosterior(hitos[indice-1]):
		indice -= 2
	default:
		return ports.HitoExpedienteRRHH{}, false
	}
	hito := hitos[indice]
	return hito, hito.AccionClave == "registrar_propuesta_formalizacion"
}

func esResolucionFormalizacionPosterior(hito ports.HitoExpedienteRRHH) bool {
	return hito.AccionClave == "registrar_resolucion_formalizacion" &&
		hito.FaseOrigen == "nombramiento" && hito.FaseDestino == "nombramiento" &&
		hito.EstadoOrigen == domain.EstadoEnCurso && hito.EstadoDestino == domain.EstadoEnCurso
}

func esAnotacionAdministrativaPosterior(hito ports.HitoExpedienteRRHH) bool {
	return hito.AccionClave == domain.AccionRegistrarAnotacionAdministrativa &&
		hito.FaseOrigen == "nombramiento" && hito.FaseDestino == "nombramiento" &&
		hito.EstadoOrigen == domain.EstadoEnCurso && hito.EstadoDestino == domain.EstadoEnCurso
}
