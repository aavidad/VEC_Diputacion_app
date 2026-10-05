package auditoria

import (
	"encoding/json"
	"os"
	"testing"
)

func TestUnionConsumosAD173YFuentesAD174(t *testing.T) {
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/union_consumos_fuentes_ad173_ad174.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 9)
	if r.Estado != "verificada" || !r.ConsumosHistoricosSinFechaLigada || !r.FechaConsumoLigadaCotejada ||
		!r.MaterialFuentesRecalculado || !r.MaterialIntentosFuentesRecalculado || r.ActorPerfilContextoCotejados ||
		r.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("unión o alcance: %+v", r)
	}
	for _, indice := range []int{0, 1, 2} {
		var copia DocumentoVerificacionMixta
		if err = json.Unmarshal(b, &copia); err != nil {
			t.Fatal(err)
		}
		copia.Registros[indice].FuentesIniciales = d.Registros[3].FuentesIniciales
		if VerificarCadenaFuentesInicialesV1(copia, copia.Manifiesto, 9).Estado != "rechazada" {
			t.Fatal("consumo admitió campos técnicos cruzados")
		}
	}
	for _, indice := range []int{3, 5} {
		var copia DocumentoVerificacionMixta
		if err = json.Unmarshal(b, &copia); err != nil {
			t.Fatal(err)
		}
		copia.Registros[indice].ConsumoFecha = d.Registros[2].ConsumoFecha
		if VerificarCadenaFuentesInicialesV1(copia, copia.Manifiesto, 9).Estado != "rechazada" {
			t.Fatal("familia técnica admitió consumo nominal cruzado")
		}
	}
}
