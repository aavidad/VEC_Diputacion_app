package auditoria

import (
	"encoding/json"
	"os"
	"testing"
)

func TestUnionConsumosAD173YCuatroFamiliasTecnicas(t *testing.T) {
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/union_consumos_tecnicos_ad173_ad174_ad176.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	r := VerificarCadenaUnidadInicialV1(d, d.Manifiesto, 15)
	if r.Estado != "verificada" || !r.ConsumosHistoricosSinFechaLigada || !r.FechaConsumoLigadaCotejada ||
		!r.MaterialFuentesRecalculado || !r.MaterialIntentosFuentesRecalculado || !r.MaterialUnidadRecalculado || !r.MaterialIntentosUnidadRecalculado ||
		r.ActorPerfilContextoCotejados || r.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("unión o alcance: %+v", r)
	}
	for _, indice := range []int{0, 1, 2, 3, 5} {
		var copia DocumentoVerificacionMixta
		if err = json.Unmarshal(b, &copia); err != nil {
			t.Fatal(err)
		}
		copia.Registros[indice].UnidadInicial = d.Registros[9].UnidadInicial
		if VerificarCadenaUnidadInicialV1(copia, copia.Manifiesto, 15).Estado != "rechazada" {
			t.Fatal("registro previo admitió campos de unidad cruzados")
		}
	}
	for _, indice := range []int{9, 11} {
		var copia DocumentoVerificacionMixta
		if err = json.Unmarshal(b, &copia); err != nil {
			t.Fatal(err)
		}
		copia.Registros[indice].ConsumoFecha = d.Registros[2].ConsumoFecha
		if VerificarCadenaUnidadInicialV1(copia, copia.Manifiesto, 15).Estado != "rechazada" {
			t.Fatal("unidad admitió consumo nominal cruzado")
		}
	}
}
