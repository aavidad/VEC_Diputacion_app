package bootstrap

import (
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// El sucesor parte de una renuncia (o expiración) o, tras una no
// incorporación, de la aceptación anterior; ningún otro terminal.
func TestEstadoTerminalSucesorDesarrollo(t *testing.T) {
	for tipo, esperado := range map[string]dominiobolsa.EstadoLlamamiento{
		"renuncia_rrhh":   dominiobolsa.EstadoLlamamientoRenunciado,
		"aceptacion_rrhh": dominiobolsa.EstadoLlamamientoAceptado,
	} {
		if e, ok := estadoTerminalSucesorDesarrollo(tipo); !ok || e != esperado {
			t.Fatalf("%s: %q %v", tipo, e, ok)
		}
	}
	for _, tipo := range []string{"", "propuesta", "sin_respuesta", "Aceptacion_rrhh", "aceptacion"} {
		if _, ok := estadoTerminalSucesorDesarrollo(tipo); ok {
			t.Fatalf("terminal %q admitido", tipo)
		}
	}
}
