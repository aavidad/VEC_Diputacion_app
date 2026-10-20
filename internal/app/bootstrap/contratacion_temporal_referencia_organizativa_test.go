package bootstrap

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const referenciaOrganizativaCTPrueba = `{"esquema":"vec.contratacion-temporal.referencia-organizativa.v1","organizacion_ref":"org_1234567890abcdef","unidad_ref":"unidad:rrhh:prueba","fuente_unidad":{"referencia":"fuente:unidad:rrhh","version":2,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`

func escribirReferenciaOrganizativaCTPrueba(t *testing.T, contenido string) string {
	t.Helper()
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(directorio, "referencia-organizativa.json")
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestLeerReferenciaOrganizativaCTAceptaCoordenadasYConservaSelector(t *testing.T) {
	ruta := escribirReferenciaOrganizativaCTPrueba(t, referenciaOrganizativaCTPrueba)
	referencia, err := leerReferenciaOrganizativaCT(ruta)
	if err != nil || referencia.organizacionRef != "org_1234567890abcdef" || referencia.unidadRef != "unidad:rrhh:prueba" {
		t.Fatalf("referencia válida rechazada: %+v, %v", referencia, err)
	}
	var selector struct {
		Dimension string   `json:"dimension"`
		Valores   []string `json:"valores"`
		Fuente    struct {
			Referencia   string `json:"referencia"`
			Version      int    `json:"version"`
			HuellaSHA256 string `json:"huella_sha256"`
		} `json:"fuente"`
	}
	ambito, err := referencia.ambitoUnidad()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(ambito, &selector); err != nil || selector.Dimension != "unidad_ref" ||
		len(selector.Valores) != 1 || selector.Valores[0] != "unidad:rrhh:prueba" ||
		selector.Fuente.Referencia != "fuente:unidad:rrhh" || selector.Fuente.Version != 2 ||
		selector.Fuente.HuellaSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("selector inesperado: %+v, %v", selector, err)
	}
}

func TestLeerReferenciaOrganizativaCTRechazaMaterialInvalido(t *testing.T) {
	casos := map[string]string{
		"organización legada": strings.Replace(referenciaOrganizativaCTPrueba, "org_1234567890abcdef", "organizacion:dipgra", 1),
		"organización vacía":  strings.Replace(referenciaOrganizativaCTPrueba, "org_1234567890abcdef", "", 1),
		"unidad vacía":        strings.Replace(referenciaOrganizativaCTPrueba, "unidad:rrhh:prueba", "", 1),
		"unidad sin forma":    strings.Replace(referenciaOrganizativaCTPrueba, "unidad:rrhh:prueba", "desconocida", 1),
		"campo desconocido":   strings.Replace(referenciaOrganizativaCTPrueba, `"esquema":`, `"otro":true,"esquema":`, 1),
		"campo duplicado":     strings.Replace(referenciaOrganizativaCTPrueba, `"unidad_ref":`, `"unidad_ref":"unidad:otra","unidad_ref":`, 1),
		"JSON múltiple":       referenciaOrganizativaCTPrueba + `{}`,
		"versión nula":        strings.Replace(referenciaOrganizativaCTPrueba, `"version":2`, `"version":null`, 1),
		"versión cero":        strings.Replace(referenciaOrganizativaCTPrueba, `"version":2`, `"version":0`, 1),
		"versión excesiva":    strings.Replace(referenciaOrganizativaCTPrueba, `"version":2`, `"version":2147483648`, 1),
		"huella alterada":     strings.Replace(referenciaOrganizativaCTPrueba, strings.Repeat("a", 64), strings.Repeat("Z", 64), 1),
		"huella nula":         strings.Replace(referenciaOrganizativaCTPrueba, strings.Repeat("a", 64), strings.Repeat("0", 64), 1),
		"fuente vacía":        strings.Replace(referenciaOrganizativaCTPrueba, "fuente:unidad:rrhh", "", 1),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			ruta := escribirReferenciaOrganizativaCTPrueba(t, contenido)
			_, err := leerReferenciaOrganizativaCT(ruta)
			if !errors.Is(err, errReferenciaOrganizativaCT) || strings.Contains(err.Error(), ruta) || strings.Contains(err.Error(), contenido) {
				t.Fatalf("error nominal esperado, obtenido %v", err)
			}
		})
	}
}

func TestLeerReferenciaOrganizativaCTRechazaRutaAusenteYPermisosAbiertos(t *testing.T) {
	if _, err := leerReferenciaOrganizativaCT(""); !errors.Is(err, errReferenciaOrganizativaCT) {
		t.Fatalf("ruta vacía: %v", err)
	}
	ruta := escribirReferenciaOrganizativaCTPrueba(t, referenciaOrganizativaCTPrueba)
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerReferenciaOrganizativaCT(ruta); !errors.Is(err, errReferenciaOrganizativaCT) {
		t.Fatalf("archivo compartido: %v", err)
	}
}
