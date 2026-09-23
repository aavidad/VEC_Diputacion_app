package ports

import "testing"

func TestEfectoAutorizacionEsDTOSinDecision(t *testing.T) {
	e := EfectoAutorizacionBorrador{Material: []byte("canon")}
	if string(e.Material) != "canon" || e.Recurso.Referencia != "" {
		t.Fatal("dto alterado")
	}
}
