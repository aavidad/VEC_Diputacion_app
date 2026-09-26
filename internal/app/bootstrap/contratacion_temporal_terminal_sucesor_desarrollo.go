package bootstrap

import dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"

// estadoTerminalSucesorDesarrollo devuelve el estado que conserva el terminal
// Bolsa del que parte el llamamiento del sucesor: la renuncia (también el
// terminal propio de una expiración) o, tras una no incorporación registrada
// por RRHH (CT124, CT128), la aceptación de la persona que no se incorporó.
// Bolsa solo abre el siguiente desde una aceptación si recibió esa no
// incorporación (Bolsa 000042) y CT solo continúa con su antecedente.
func estadoTerminalSucesorDesarrollo(tipo string) (dominiobolsa.EstadoLlamamiento, bool) {
	switch tipo {
	case "renuncia_rrhh":
		return dominiobolsa.EstadoLlamamientoRenunciado, true
	case "aceptacion_rrhh":
		return dominiobolsa.EstadoLlamamientoAceptado, true
	}
	return "", false
}
