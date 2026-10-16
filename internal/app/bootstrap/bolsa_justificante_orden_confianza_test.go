package bootstrap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPreflightJustificanteConservaClavesYOrdenSQL(t *testing.T) {
	originales := []claveInventarioV3PortalExterno{
		{Audiencia: audienciaJustificanteBolsaSubir, ClaveID: "clave:subir", Version: 10},
		{Audiencia: audienciaJustificanteBolsaDescargar, ClaveID: "clave:descargar", Version: 11},
		{Audiencia: audienciaJustificanteBolsaPreparar, ClaveID: "clave:preparar", Version: 12},
	}
	inv := inventarioV3PortalExterno{Consumidores: map[string][]claveInventarioV3PortalExterno{consumidorJustificantesBolsaExterno: originales}}
	b, err := materialJSONV3PortalExterno(inv, consumidorJustificantesBolsaExterno, configuracionInventarioV3PortalExterno{})
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Claves []struct {
			Audiencia string `json:"audiencia_consumo"`
			ClaveID   string `json:"clave_id"`
			Version   uint64 `json:"version"`
		} `json:"claves"`
	}
	if json.Unmarshal(b, &d) != nil || len(d.Claves) != 3 {
		t.Fatal("material inválido")
	}
	for i, indice := range []int{2, 0, 1} {
		esperado := originales[indice]
		c := d.Claves[i]
		if c.Audiencia != esperado.Audiencia || c.ClaveID != esperado.ClaveID || c.Version != esperado.Version {
			t.Fatal("cambió una clave al ordenar el preflight")
		}
	}
	if !reflect.DeepEqual(inv.Consumidores[consumidorJustificantesBolsaExterno], originales) {
		t.Fatal("se modificó el inventario ya publicado")
	}
	for _, malas := range [][]claveInventarioV3PortalExterno{originales[:2], {originales[0], originales[0], originales[2]}, {originales[0], originales[1], {Audiencia: "ajena"}}} {
		inv.Consumidores[consumidorJustificantesBolsaExterno] = malas
		if _, err := materialJSONV3PortalExterno(inv, consumidorJustificantesBolsaExterno, configuracionInventarioV3PortalExterno{}); err == nil {
			t.Fatal("se aceptó un conjunto documental distinto")
		}
	}
}
