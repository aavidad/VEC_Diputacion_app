package catalogopresencia

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func huellaTest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func TestSnapshotSHAEstrictoSinCamposSensibles(t *testing.T) {
	b, err := os.ReadFile("../../../../../data/demo/cronos/presencia-equipo.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := CargarSnapshot(b, huellaTest(b))
	if err != nil || len(s.Personas) != 6 || s.SHA256 != huellaTest(b) {
		t.Fatal(err)
	}
	for _, entrada := range []string{
		strings.Replace(string(b), `"demostracion": true`, `"demostracion": true, "demostracion": true`, 1),
		strings.Replace(string(b), `"version_esquema": 1`, `"Version_esquema": 1`, 1),
		strings.Replace(string(b), `"version_esquema": 1`, `"version_esquema": 1, "motivo_permiso": "x"`, 1),
		strings.Replace(string(b), `"version_esquema": 1`, `"version_esquema": 1, "documento_ref": "x"`, 1),
		strings.Replace(string(b), `"version_esquema": 1`, `"version_esquema": 1, "gps": []`, 1),
		string(b) + `{}`, strings.Repeat(" ", LimiteJSON+1),
	} {
		if _, err := CargarSnapshot([]byte(entrada), huellaTest([]byte(entrada))); err == nil {
			t.Fatal("incompatible input accepted")
		}
	}
	if _, err := CargarSnapshot(b, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong SHA accepted")
	}
}
func TestCatalogosRealesPorIdioma(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		b, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/cronos-presencia-ensayo.json")
		if err != nil {
			t.Fatal(err)
		}
		c, err := CargarTextos(b, huellaTest(b), idioma)
		if err != nil || c.Textos["titulo"] == "" || c.Textos["cobertura_incompleta"] == "" || c.Textos["sin_marcajes"] == "" || c.Textos["secuencia_ambigua"] == "" {
			t.Fatal(err)
		}
		if _, err := CargarTextos(b, huellaTest(b), "otro"); err == nil {
			t.Fatal("wrong locale")
		}
	}
}

func TestCatalogoMotivosExigeClavesExactas(t *testing.T) {
	b, err := os.ReadFile("../../../../../web/static/textos/es/cronos-presencia-ensayo.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		nombre string
		mutar  func(map[string]string)
	}{
		{"falta motivo", func(m map[string]string) { delete(m, "sin_marcajes") }},
		{"clave sustituida", func(m map[string]string) { m["otra_clave"] = m["sin_marcajes"]; delete(m, "sin_marcajes") }},
		{"clave adicional", func(m map[string]string) { m["otra_clave"] = "x" }},
		{"motivo vacio", func(m map[string]string) { m["sin_marcajes"] = "" }},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			var c CatalogoTextos
			if err := json.Unmarshal(b, &c); err != nil {
				t.Fatal(err)
			}
			tc.mutar(c.Textos)
			entrada, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = CargarTextos(entrada, huellaTest(entrada), "es"); err == nil {
				t.Fatal("catalogue with incompatible keys accepted")
			}
		})
	}
}
