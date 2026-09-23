package domain

import (
	"testing"
	"time"
)

func TestAvisoRRHHExigeReferenciaOpacaYDetalle(t *testing.T) {
	aviso := AvisoRRHH{Tipo: AvisoSaltoOrden, BolsaRef: "bolsa:1", Referencia: "aviso:1", Fecha: time.Now(), Detalle: map[string]any{"participacion_ref": "participacion:1"}}
	if err := aviso.Validar(); err != nil {
		t.Fatal(err)
	}
	aviso.Detalle = nil
	if err := aviso.Validar(); err == nil {
		t.Fatal("un aviso sin detalle no debe ser valido")
	}
}
