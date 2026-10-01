package catalogoefectos_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
)

func huella(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func TestJSONEstrictoNoAceptaCambiosAmbiguos(t *testing.T) {
	for _, texto := range []string{`{"version":1,"version":2}`, `{"version":1,"v\u0065rsion":2}`, `{"Version":1}`, `{"version":1,"otro":1}`, `{"version":1} {}`, `{"version":1.2}`, `null`, strings.Repeat("[", 18) + "0" + strings.Repeat("]", 18), strings.Repeat(" ", catalogoefectos.LimiteJSON+1)} {
		var destino struct {
			Version int `json:"version"`
		}
		if catalogoefectos.DecodificarEstricto([]byte(texto), huella([]byte(texto)), &destino) == nil {
			t.Fatalf("admitido: %s", texto)
		}
	}
	b, err := os.ReadFile("../../../../../data/demo/reglas/cronos-efectos-permisos.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := catalogoefectos.Cargar(b, huella(b))
	if err != nil || c.Politica.SHA256 != huella(b) {
		t.Fatal("catálogo válido rechazado")
	}
	if _, err = catalogoefectos.Cargar(append(b, ' '), huella(b)); err == nil {
		t.Fatal("cambio de bytes sin otra huella admitido")
	}
	for _, cambio := range []string{strings.Replace(string(b), `"demostracion": true`, `"demostracion": false`, 1), strings.Replace(string(b), `"paquete:ejemplo:vec:v1"`, `"paquete:produccion"`, 1)} {
		if _, err := catalogoefectos.Cargar([]byte(cambio), huella([]byte(cambio))); err == nil {
			t.Fatal("paquete sin marca demo admitido")
		}
	}
}
