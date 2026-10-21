package postgres

import (
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func selectorDocumentalPGPrueba() domain.SelectorBolsaPlanB2Nativo {
	return domain.SelectorBolsaPlanB2Nativo{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba",
		ClaveCT:   "41111111-1111-4111-8111-111111111111",
		UnidadRef: "unidad:prueba", CategoriaRef: "categoria:prueba",
		NecesidadRef: "necesidad:prueba", BolsaRef: "bolsa:prueba",
		LlamamientoRef: "llamamiento:prueba", ReciboEmisionRef: "recibo:emision:prueba",
		ReciboRespuestaRef: "recibo:respuesta:prueba",
	}
}

func TestSelectorDocumentalPGSoloDiezReferenciasPropias(t *testing.T) {
	x := selectorDocumentalPGPrueba()
	b, err := json.Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := decodificarSelectorIdentidadDocumentalRRHH(string(b), x.ExpedienteRef, x.OrganizacionRef)
	if err != nil || actual != x {
		t.Fatal("selector nativo perdido", err)
	}
	for _, raw := range []string{
		string(b) + "{}", "null", strings.Repeat("x", 2049),
		strings.Replace(string(b), `"clave_ct":"`+x.ClaveCT+`"`, `"clave_ct":"clave:falsa"`, 1),
		strings.Replace(string(b), `"necesidad_ref":"`+x.NecesidadRef+`"`, `"necesidad_ref":""`, 1),
		strings.TrimSuffix(string(b), "}") + `,"persona_ref":"per_ajena"}`,
	} {
		if _, err := decodificarSelectorIdentidadDocumentalRRHH(raw, x.ExpedienteRef, x.OrganizacionRef); err == nil {
			t.Fatal("selector ajeno o ampliado admitido")
		}
	}
	if _, err := decodificarSelectorIdentidadDocumentalRRHH(string(b), "expediente:ajeno", x.OrganizacionRef); err == nil {
		t.Fatal("otro expediente")
	}
	if _, err := decodificarSelectorIdentidadDocumentalRRHH(string(b), x.ExpedienteRef, "organizacion:ajena"); err == nil {
		t.Fatal("otra organización")
	}
}
