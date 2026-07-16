package httpapi

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// El adaptador HTTP puede depender temporalmente de contratos funcionales de
// modulo mientras T03 termina, pero no puede volver a elegir adaptadores de
// persistencia ni construir infraestructura ajena. Esa decision corresponde
// exclusivamente a la raiz internal/app/bootstrap.
func TestHTTPAPINoComponeAdaptadoresDeOtrosModulos(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entrada := range entradas {
		if entrada.IsDir() || !strings.HasSuffix(entrada.Name(), ".go") || strings.HasSuffix(entrada.Name(), "_test.go") {
			continue
		}
		archivo, err := parser.ParseFile(token.NewFileSet(), entrada.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("analizar %s: %v", entrada.Name(), err)
		}
		for _, importacion := range archivo.Imports {
			ruta, err := strconv.Unquote(importacion.Path.Value)
			if err != nil {
				t.Fatalf("decodificar import de %s: %v", entrada.Name(), err)
			}
			if strings.Contains(ruta, "/internal/modules/") && strings.Contains(ruta, "/adapters/") {
				t.Fatalf("%s compone el adaptador ajeno %s; debe inyectarlo bootstrap", entrada.Name(), ruta)
			}
		}
	}
}
