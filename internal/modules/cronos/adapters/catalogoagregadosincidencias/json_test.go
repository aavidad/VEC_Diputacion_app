package catalogoagregadosincidencias

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func huellaTest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func fixtureTest(t *testing.T, ruta string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../../../" + ruta)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSnapshotAgregadosJSONCerrado(t *testing.T) {
	b := fixtureTest(t, "data/demo/cronos/agregados-incidencias-periodo.json")
	s, err := CargarSnapshot(b, huellaTest(b))
	if err != nil || s.SHA256 != huellaTest(b) {
		t.Fatal(err)
	}
	casos := map[string][]byte{
		"duplicate":        bytes.Replace(b, []byte(`"demo": true`), []byte(`"demo": true, "demo": true`), 1),
		"nested_duplicate": bytes.Replace(b, []byte(`"fuente_version": 3`), []byte(`"fuente_version": 3, "fuente_version": 3`), 1),
		"upper_key":        bytes.Replace(b, []byte(`"demo"`), []byte(`"Demo"`), 1),
		"unknown":          bytes.Replace(b, []byte(`"demo": true`), []byte(`"demo": true, "nombre": "demo"`), 1),
		"sensitive_field":  bytes.Replace(b, []byte(`"codigo": "registro_incompleto"`), []byte(`"codigo": "registro_incompleto", "documento": "demo"`), 1),
		"wrong_type":       bytes.Replace(b, []byte(`"demo": true`), []byte(`"demo": "true"`), 1),
		"schema":           bytes.Replace(b, []byte(`"version_esquema": 1`), []byte(`"version_esquema": 2`), 1),
		"schema_string":    bytes.Replace(b, []byte(`"version_esquema": 1`), []byte(`"version_esquema": "1"`), 1),
		"false":            bytes.Replace(b, []byte(`"demo": true`), []byte(`"demo": false`), 1),
		"null":             []byte("null"), "array": []byte("[]"), "empty": nil,
		"trailing": append(append([]byte{}, b...), []byte(" {}")...),
		"oversize": bytes.Repeat([]byte(" "), LimiteJSON+1),
		"utf8":     bytes.Replace(b, []byte("demo:snapshot"), []byte{'d', 'e', 'm', 'o', ':', 0xff}, 1),
	}
	for nombre, v := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := CargarSnapshot(v, huellaTest(v)); err == nil {
				t.Fatal("accepted invalid JSON")
			}
		})
	}
	if _, err := CargarSnapshot(b, huellaTest([]byte("other"))); err == nil {
		t.Fatal("accepted wrong digest")
	}
}

func TestTextosAgregadosESEN(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		b := fixtureTest(t, "web/static/textos/"+lang+"/cronos-agregados-incidencias-ensayo.json")
		c, err := CargarTextos(b, huellaTest(b), lang)
		if err != nil || len(c.Textos) != 12 || c.Textos["entrada_invalida"] == "" {
			t.Fatal(err)
		}
		if _, err := CargarTextos(b, huellaTest(b), "other"); err == nil {
			t.Fatal("accepted mismatched language")
		}
		for _, v := range [][]byte{
			bytes.Replace(b, []byte(`"titulo"`), []byte(`"otra"`), 1),
			bytes.Replace(b, []byte(`"version_esquema": "1"`), []byte(`"version_esquema": 1`), 1),
			bytes.Replace(b, []byte(`"textos": {`), []byte(`"textos": { "extra": "demo",`), 1),
			bytes.Replace(b, []byte(`"titulo": "`), []byte(`"titulo": "\n`), 1),
		} {
			if _, err := CargarTextos(v, huellaTest(v), lang); err == nil {
				t.Fatal("accepted open text catalogue")
			}
		}
	}
}
