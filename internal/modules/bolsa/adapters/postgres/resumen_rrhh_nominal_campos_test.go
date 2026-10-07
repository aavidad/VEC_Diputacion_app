package postgres

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestResumenNominalRechazaFilasPersonalesFueraDeCampos(t *testing.T) {
	const generado = `"generado_en":"2026-10-07T12:00:00Z"`
	bolsas := []byte(`{` + generado + `,"bolsas":[]}`)
	if _, err := decodificarResumenNominal(bolsas, ports.AccionRRHHBolsasConsultar); err != nil {
		t.Fatalf("agregado vacío válido: %v", err)
	}
	if _, err := decodificarResumenNominal([]byte(`{`+generado+`,"bolsas":[],"situaciones":[{"participacion_ref":"participacion:ajena"}]}`), ports.AccionRRHHBolsasConsultar); err == nil {
		t.Fatal("la acción bolsas aceptó filas de participaciones")
	}
	estadisticas := []byte(`{` + generado + `,"estadisticas":{"bolsas":{"total":0,"vigentes":0,"sustituidas":0},"personas":{"total":0,"por_estado":{}},"llamamientos":{"en_curso":0,"historico_disponible":false,"total":null,"por_canal":null,"por_resultado":null},"por_bolsa":[]}}`)
	if _, err := decodificarResumenNominal(estadisticas, ports.AccionRRHHEstadisticasConsultar); err != nil {
		t.Fatalf("estadísticas agregadas válidas: %v", err)
	}
	if _, err := decodificarResumenNominal([]byte(`{`+generado+`,"estadisticas":{"bolsas":{"total":0,"vigentes":0,"sustituidas":0},"personas":{"total":0,"por_estado":{}},"llamamientos":{"en_curso":0,"historico_disponible":false,"total":null,"por_canal":null,"por_resultado":null},"por_bolsa":[],"participacion_ref":"participacion:ajena"}}`), ports.AccionRRHHEstadisticasConsultar); err == nil {
		t.Fatal("la acción estadísticas aceptó un campo personal")
	}
}
