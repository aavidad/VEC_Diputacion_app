package composicion

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

func TestRegistroLiquidacionCierraSinCompetenciaAntesDeEmitir(t *testing.T) {
	base, _ := identidadYMaterialR15(t)
	emisor := &emisorCircuitoPrueba{}
	r, err := NuevoResolutorIdentidadEfectivaCircuito(&fuenteContextoR15{identidad: base}, FuenteCompetenciaCircuitoSinCatalogo{}, emisor, motivoCircuitoPrueba())
	if err != nil {
		t.Fatal(err)
	}
	s := dietasports.SolicitudLiquidacionEconomica{Decision: dietasports.SolicitudDecisionCircuito{
		Referencia: "dco_" + strings.Repeat("a", 22), Etapa: domain.EtapaLiquidacion,
		Decision: domain.DecisionAprobar, VersionEsperada: 2, ClaveIdempotencia: "clave_0123456789abcdef"}}
	if _, err := r.ResolverRegistroLiquidacionEconomica(context.Background(), s); !errors.Is(err, dietasports.ErrCompetenciaCircuitoSinFuente) || emisor.llamadas != 0 {
		t.Fatalf("sin competencia: %v, emisiones %d", err, emisor.llamadas)
	}
}
