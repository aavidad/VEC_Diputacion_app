package ports

import (
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCoberturaGobernadaAdmiteLasComprobacionesPersistidas(t *testing.T) {
	cobertura := CoberturaOperativaRRHH{
		ViaClave: "bolsa_vigente", DecisionGobernada: true,
		Comprobaciones: []ComprobacionOperativaRRHH{{
			Clave: "existe_bolsa_vigente", Resultado: domain.ComprobacionAfirmativa,
		}, {
			Clave: "hay_candidaturas_disponibles", Resultado: domain.ComprobacionNegativa,
		}},
	}
	if err := cobertura.validar(); err != nil {
		t.Fatal(err)
	}
	cobertura.Comprobaciones = nil
	if err := cobertura.validar(); err != nil {
		t.Fatalf("la ausencia histórica de comprobaciones sigue siendo legible: %v", err)
	}
}
