package catalogoincidencias

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func shaIncidenciasTest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func fixtureIncidenciasTest(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../../../data/demo/cronos/incidencias-periodo.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCatalogoIncidenciasIdentificaBytesExactos(t *testing.T) {
	b := fixtureIncidenciasTest(t)
	s, err := CargarSnapshot(b, shaIncidenciasTest(b))
	if err != nil {
		t.Fatal(err)
	}
	if !s.Demostracion || s.VersionEsquema != 1 || s.SHA256 != shaIncidenciasTest(b) {
		t.Fatal("lost exact fingerprint")
	}
	if _, err = CargarSnapshot(append(b, ' '), s.SHA256); err == nil {
		t.Fatal("accepted different bytes with old hash")
	}
}
func TestCatalogoIncidenciasRechazaJSONAmbiguoYFueraContrato(t *testing.T) {
	b := fixtureIncidenciasTest(t)
	cases := map[string][]byte{
		"duplicate key":     bytes.Replace(b, []byte(`"version_esquema": 1`), []byte(`"version_esquema": 1, "version_esquema": 1`), 1),
		"uppercase key":     bytes.Replace(b, []byte(`"version_esquema"`), []byte(`"VERSION_ESQUEMA"`), 1),
		"unknown field":     bytes.Replace(b, []byte(`"demostracion": true`), []byte(`"demostracion": true, "unknown": 1`), 1),
		"trailing document": append(append([]byte{}, b...), []byte(`{}`)...),
		"bad UTF8":          bytes.Replace(b, []byte(`demo:snapshot`), []byte{0xff}, 1),
		"unknown schema":    bytes.Replace(b, []byte(`"version_esquema": 1`), []byte(`"version_esquema": 2`), 1),
		"not demo":          bytes.Replace(b, []byte(`"demostracion": true`), []byte(`"demostracion": false`), 1),
		"null":              []byte(`null`),
		"too large":         []byte(strings.Repeat(" ", LimiteJSON+1)),
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			if s, err := CargarSnapshot(bad, shaIncidenciasTest(bad)); err == nil || s.Demostracion {
				t.Fatalf("accepted %s", name)
			}
		})
	}
}
func TestCatalogoIncidenciasTextosPorDatosEIdentidadIdioma(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		b, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/cronos-incidencias-ensayo.json")
		if err != nil {
			t.Fatal(err)
		}
		c, err := CargarTextos(b, shaIncidenciasTest(b), idioma)
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Textos) != 13 || c.Idioma != idioma {
			t.Fatal("incomplete texts")
		}
		if _, err = CargarTextos(b, shaIncidenciasTest(b), "different"); err == nil {
			t.Fatal("accepted language mismatch")
		}
		bad := bytes.Replace(b, []byte(`"titulo"`), []byte(`"unknown"`), 1)
		if _, err = CargarTextos(bad, shaIncidenciasTest(bad), idioma); err == nil {
			t.Fatal("accepted missing title")
		}
	}
	// A matching data catalogue can select another locale without code changes.
	b, err := os.ReadFile("../../../../../web/static/textos/en/cronos-incidencias-ensayo.json")
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.Replace(b, []byte(`"idioma": "en"`), []byte(`"idioma": "fr"`), 1)
	if _, err = CargarTextos(b, shaIncidenciasTest(b), "fr"); err != nil {
		t.Fatal("locale compiled into code", err)
	}
}
