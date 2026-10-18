package postgres

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestFuentePreferenciasModalidadesCanonYAlcance(t *testing.T) {
	leer := func(ruta string) fuentePaqueteCatalogoAccionesV2 {
		t.Helper()
		b, err := os.ReadFile("../../../../../deploy/principal/fuentes/" + ruta)
		if err != nil {
			t.Fatal(err)
		}
		var fuente fuentePaqueteCatalogoAccionesV2
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&fuente); err != nil {
			t.Fatal(err)
		}
		return fuente
	}
	previa := leer("bolsa_preferencias_bolsas_propias_v1.json")
	nueva := leer("bolsa_preferencias_bolsas_propias_v2.json")
	canon, huella, err := canonEntradasFuenteV2(nueva.Entradas)
	if err != nil || string(canon) != nueva.EntradasCanon || huella != nueva.HuellaSHA256 || nueva.Version != 2 || len(nueva.Entradas) != 2 {
		t.Fatalf("fuente sin integridad: %v", err)
	}
	for i, entrada := range nueva.Entradas {
		if entrada.FuenteVersion != 2 || entrada.Version != 2 || entrada.FuenteHuellaSHA256 != huella ||
			entrada.VigenteHasta.Sub(entrada.VigenteDesde) != 90*24*time.Hour {
			t.Fatal("versión o vigencia incorrecta")
		}
		if i == 1 && !reflect.DeepEqual(entrada.Concesion, previa.Entradas[i].Concesion) {
			t.Fatal("cambió la concesión de escritura")
		}
	}
	esperados := append(slices.Clone(previa.Entradas[0].Concesion.CamposPermitidos), "modalidades_guardadas", "modalidades_disponibles")
	if !slices.Equal(nueva.Entradas[0].Concesion.CamposPermitidos, esperados) {
		t.Fatal("campos ajenos en la ampliación")
	}
	consulta := nueva.Entradas[0].Concesion
	consulta.CamposPermitidos = previa.Entradas[0].Concesion.CamposPermitidos
	if !reflect.DeepEqual(consulta, previa.Entradas[0].Concesion) {
		t.Fatal("cambió otro atributo de consulta")
	}
}
