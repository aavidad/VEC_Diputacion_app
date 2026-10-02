package catalogoefectos

import (
	"os"
	"strings"
	"testing"
)

func TestPropuestaC3Estrica(t *testing.T) {
	b, e := os.ReadFile("../../../../../data/demo/reglas/cronos-efectos-permisos.configurable.json")
	if e != nil {
		t.Fatal(e)
	}
	p, e := ValidarPropuestaC3(b)
	if e != nil || len(p.Reglas) != 3 || len(p.SHA256) != 64 {
		t.Fatalf("propuesta: %v", e)
	}
	for _, s := range []string{
		strings.Replace(string(b), `"version_esquema": 1`, `"version_esquema": 1, "version_esquema": 1`, 1),
		strings.Replace(string(b), `"efecto": "credito_jornada_completa"`, `"efecto": "ninguno"`, 1),
		strings.Replace(string(b), `"textos": {`, `"desconocido": true, "textos": {`, 1),
		strings.Replace(string(b), `"textos": {`, `"textos": {"INVALIDO":{"aviso":"x"},`, 1),
	} {
		if _, e := ValidarPropuestaC3([]byte(s)); e == nil {
			t.Fatal("aceptó propuesta alterada")
		}
	}
	if _, e := ValidarPropuestaC3(append(append([]byte(nil), b...), 0xff)); e == nil {
		t.Fatal("aceptó UTF-8 inválido")
	}
}
