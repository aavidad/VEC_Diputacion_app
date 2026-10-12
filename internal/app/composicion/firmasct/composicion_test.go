package firmasct

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaNoOfreceEscrituraDirectaSinPlan(t *testing.T) {
	// Incluso sin un escritor configurado, esta entrada deniega en lugar de
	// intentar el registro directo de CT172. El decorador escribe por CT176.
	consulta := consultaSinRegistroDirecto{}
	recibo, err := consulta.RegistrarFirmaVerificadaV2(t.Context(),
		ports.MaterialFirmaVerificadaV2{}, ports.CapacidadFirmaVerificadaV2{})
	if !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) || recibo != (ports.ReciboFirmaDocumento{}) {
		t.Fatalf("registro directo: recibo=%v, error=%v", recibo, err)
	}
}
