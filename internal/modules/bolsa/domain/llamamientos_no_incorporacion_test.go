package domain

import "testing"

// Una aceptación solo es antecedente si la persona no llegó a incorporarse.
func TestProponerSiguienteLlamamientoTrasAceptacionSinIncorporacion(t *testing.T) {
	o := ordenSiguienteLlamamientoPrueba(t)
	o.Terminal.estado = EstadoLlamamientoAceptado
	o.Terminal.terminal.Estado = EstadoLlamamientoAceptado
	if _, err := ProponerSiguienteLlamamiento(o); err == nil {
		t.Fatal("aceptación sin no incorporación admitida")
	}
	o.TrasNoIncorporacion = true
	if p, err := ProponerSiguienteLlamamiento(o); err != nil || p.OrdenSeleccionado != 3 {
		t.Fatal("aceptación con no incorporación rechazada", err)
	}
}
