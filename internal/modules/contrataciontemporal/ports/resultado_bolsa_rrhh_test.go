package ports_test

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/resultadobolsa"
)

func TestResultadoBolsaRRHHConservaCausalidadYUTC(t *testing.T) {
	llamamiento := "llamamiento:" + strings.Repeat("a", 64)
	recibo := "recibo:" + llamamiento
	raw := `{"vinculos":[{"bolsa_ref":"bolsa:prueba","llamamiento_ref":"` + llamamiento + `",` +
		`"recibo_emision_ref":"` + recibo + `","recibo_vinculo_ref":"recibo:ct:bolsa:prueba",` +
		`"vinculado_en":"2026-10-09T11:00:00+00:00","emitido_en":"2026-10-09T10:00:00+00:00",` +
		`"participaciones":[{"participacion_ref":"participacion:prueba","respuesta":null,"modo":null,` +
		`"recibo_respuesta_ref":null,"respondida_en":null,"justificante_ref":null,` +
		`"contacto_resultado":"rechaza","recibo_contacto_ref":"recibo:contacto:prueba",` +
		`"contacto_en":"2026-10-09T10:30:00+00:00","situacion_actual":"renuncia",` +
		`"recibo_situacion_ref":"recibo:situacion:prueba","situacion_desde":"2026-10-09T10:45:00+00:00"}]}],` +
		`"total_vinculos":1,"personas_solicitadas":null,"aceptaciones_firmes":0,` +
		`"emisiones_vinculables":[],"siguiente_cursor":null}`
	r, err := postgres.ResultadoBolsaRRHHDesdeSQL([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	p := r.Vinculos[0].Participaciones[0]
	if p.Respuesta != nil || p.SituacionActual == nil || *p.SituacionActual != "renuncia" ||
		r.Vinculos[0].EmitidoEn.Location() != time.UTC || p.ContactoEn.Location() != time.UTC {
		t.Fatal("se confundió situación global con respuesta o no se normalizó UTC")
	}
	cruzada := strings.Replace(raw, `"recibo_emision_ref":"`+recibo+`"`,
		`"recibo_emision_ref":"recibo:ajeno"`, 1)
	if _, err := postgres.ResultadoBolsaRRHHDesdeSQL([]byte(cruzada)); err == nil {
		t.Fatal("se aceptó una emisión con recibo ajeno")
	}
}

func TestCursorResultadoBolsaRRHHRechazaFechaYReferenciaInvalidas(t *testing.T) {
	valido := "2026-10-09T11:00:00.123456Z#llamamiento:" + strings.Repeat("a", 64)
	if !resultadobolsa.CursorResultadoBolsaRRHHValido(valido) || !resultadobolsa.ResultadoBolsaRRHHSolicitado(resultadobolsa.ConResultadoBolsaRRHHPagina(t.Context(), valido)) {
		t.Fatal("el cursor de la página no llegó a la lectura")
	}
	for _, cursor := range []string{
		"2026-13-09T11:00:00.123456Z#llamamiento:" + strings.Repeat("a", 64),
		"2026-10-09T11:00:00.123456Z#llamamiento:" + strings.Repeat("g", 64),
		"2026-10-09T11:00:00.123456Z#llamamiento:" + strings.Repeat("a", 63),
	} {
		if resultadobolsa.CursorResultadoBolsaRRHHValido(cursor) {
			t.Fatal("cursor inválido admitido")
		}
	}
}
