package ports_test

import (
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
)

func TestOrigenCTConfirmadoEnResultadoBolsaSoloParaRespuestaFirmeSingular(t *testing.T) {
	llamamiento := "llamamiento:" + strings.Repeat("a", 64)
	participacion := `{"participacion_ref":"participacion:original","respuesta":"acepta",` +
		`"modo":"firme","recibo_respuesta_ref":"recibo:respuesta:original",` +
		`"respondida_en":"2026-10-20T09:30:00Z","justificante_ref":null,` +
		`"contacto_resultado":null,"recibo_contacto_ref":null,"contacto_en":null,` +
		`"situacion_actual":null,"recibo_situacion_ref":null,"situacion_desde":null,` +
		`"origen_ct_confirmado":true}`
	base := `{"vinculos":[{"bolsa_ref":"bolsa:original","llamamiento_ref":"` + llamamiento + `",` +
		`"recibo_emision_ref":"recibo:` + llamamiento + `","recibo_vinculo_ref":"recibo:ct:original",` +
		`"vinculado_en":"2026-10-20T10:00:00Z","emitido_en":"2026-10-20T09:00:00Z",` +
		`"participaciones":[` + participacion + `]}],"total_vinculos":1,` +
		`"personas_solicitadas":null,"aceptaciones_firmes":1,"emisiones_vinculables":[],"siguiente_cursor":null}`
	r, err := postgres.ResultadoBolsaRRHHDesdeSQL([]byte(base))
	if err != nil || !r.Vinculos[0].Participaciones[0].OrigenCTConfirmado {
		t.Fatal("origen CT acreditado perdido", err)
	}
	// La función anterior no incluía la marca: debe seguir pudiéndose leer.
	antiguo := strings.Replace(base, `,"origen_ct_confirmado":true`, "", 1)
	r, err = postgres.ResultadoBolsaRRHHDesdeSQL([]byte(antiguo))
	if err != nil || r.Vinculos[0].Participaciones[0].OrigenCTConfirmado {
		t.Fatal("resultado anterior convertido en origen CT", err)
	}
	serializado, _ := json.Marshal(r.Vinculos[0].Participaciones[0])
	if !strings.Contains(string(serializado), `"origen_ct_confirmado":false`) {
		t.Fatal("la ficha no recibió el falso explícito")
	}
	for nombre, dato := range map[string]string{
		"renuncia":  strings.Replace(base, `"respuesta":"acepta"`, `"respuesta":"renuncia"`, 1),
		"sin firme": strings.Replace(base, `"modo":"firme"`, `"modo":"propuesta_rrhh"`, 1),
		"temprana":  strings.Replace(base, `"respondida_en":"2026-10-20T09:30:00Z"`, `"respondida_en":"2026-10-20T08:30:00Z"`, 1),
		"multiple": strings.Replace(base, `"participaciones":[`+participacion+`]`,
			`"participaciones":[`+participacion+`,`+strings.Replace(participacion, "participacion:original", "participacion:otra", 1)+`]`, 1),
	} {
		if _, err := postgres.ResultadoBolsaRRHHDesdeSQL([]byte(dato)); err == nil {
			t.Fatal("origen CT incoherente aceptado:", nombre)
		}
	}
}
