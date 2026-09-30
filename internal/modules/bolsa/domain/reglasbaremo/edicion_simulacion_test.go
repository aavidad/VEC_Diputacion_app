package reglasbaremo

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestEdicionConservaContratoHistorico(t *testing.T) {
	b, err := os.ReadFile("../../application/simulacionbaremo/testdata/reglas_a.json")
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	edicion, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarConjuntoReglasBaremo(edicion); err == nil {
		t.Fatal("restauración histórica relajada")
	}
	canonico, err := CanonicalizarConjuntoReglasBaremoJSON(edicion)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonico, b) {
		t.Fatal("edición cambió el conjunto")
	}
	for _, dato := range [][]byte{append(append([]byte{}, b...), b...), bytes.Replace(b, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1), bytes.Replace(b, []byte(`"version":1`), []byte(`"version":1,"Version":2`), 1), bytes.Replace(b, []byte(`"version":1`), []byte(`"Version":1`), 1), bytes.Replace(b, []byte(`"bases":`), []byte(`"ba\u017fes":`), 1), bytes.Replace(b, []byte(`"esquema":`), []byte(`"intruso":true,"esquema":`), 1), []byte(strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34))} {
		if _, err := CanonicalizarConjuntoReglasBaremoJSON(dato); err == nil {
			t.Fatal("edición ambigua admitida")
		}
	}
}
