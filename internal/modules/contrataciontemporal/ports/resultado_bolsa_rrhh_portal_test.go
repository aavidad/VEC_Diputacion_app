package ports_test

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
)

func TestResultadoBolsaPortalSoloHabilitaOrigenNativoFirme(t *testing.T) {
	llamamiento := "llamamiento:" + strings.Repeat("a", 64)
	base := `{"vinculos":[{"bolsa_ref":"bolsa:portal","llamamiento_ref":"` + llamamiento + `",` +
		`"recibo_emision_ref":"recibo:` + llamamiento + `","recibo_vinculo_ref":"recibo:ct:portal",` +
		`"vinculado_en":"2026-10-20T10:00:00Z","emitido_en":"2026-10-20T09:00:00Z",` +
		`"participaciones":[{"participacion_ref":"participacion:portal","respuesta":"acepta",` +
		`"modo":"firme","recibo_respuesta_ref":"recibo:respuesta:portal",` +
		`"respondida_en":"2026-10-20T09:30:00Z","justificante_ref":null,` +
		`"contacto_resultado":null,"recibo_contacto_ref":null,"contacto_en":null,` +
		`"situacion_actual":null,"recibo_situacion_ref":null,"situacion_desde":null,` +
		`"origen_ct_confirmado":true,"resolucion_portal_disponible":true}]}],"total_vinculos":1,` +
		`"personas_solicitadas":null,"aceptaciones_firmes":1,"emisiones_vinculables":[],"siguiente_cursor":null}`
	antes, err := postgres.ResultadoBolsaRRHHDesdeSQL([]byte(base))
	if err != nil || !antes.Vinculos[0].Participaciones[0].ResolucionPortalDisponible {
		t.Fatal("la respuesta singular acreditada no habilita la revisión", err)
	}
	noDisponible := strings.Replace(base, `"resolucion_portal_disponible":true`, `"resolucion_portal_disponible":false`, 1)
	despues, err := postgres.ResultadoBolsaRRHHDesdeSQL([]byte(noDisponible))
	if err != nil || despues.Vinculos[0].Participaciones[0].ResolucionPortalDisponible {
		t.Fatal("la ficha no pudo mostrar una aceptación sin nueva operación", err)
	}
	for nombre, dato := range map[string]string{
		"sin_origen_nativo":    strings.Replace(base, `"origen_ct_confirmado":true`, `"origen_ct_confirmado":false`, 1),
		"renuncia_no_aceptada": strings.Replace(base, `"respuesta":"acepta"`, `"respuesta":"renuncia"`, 1),
	} {
		if _, err := postgres.ResultadoBolsaRRHHDesdeSQL([]byte(dato)); err == nil {
			t.Fatal("proyección portal no confiable aceptada:", nombre)
		}
	}
}
