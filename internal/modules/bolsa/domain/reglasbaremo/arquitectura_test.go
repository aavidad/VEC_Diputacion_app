package reglasbaremo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestContratoReglasNoAdmiteConfiguracionDinamica protege el limite de
// confianza del baremo: las bases sólo pueden materializarse con tipos
// declarativos conocidos y aritmetica exacta. Una formula, un callback o una
// consulta introducidos en el modelo convertirian datos administrativos en
// codigo ejecutable.
func TestContratoReglasNoAdmiteConfiguracionDinamica(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("leer paquete: %v", err)
	}

	for _, entrada := range entradas {
		if entrada.IsDir() || filepath.Ext(entrada.Name()) != ".go" ||
			strings.HasSuffix(entrada.Name(), "_test.go") {
			continue
		}
		ruta := entrada.Name()
		archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
		if err != nil {
			t.Fatalf("analizar %s: %v", ruta, err)
		}
		comprobarImportacionesCerradas(t, ruta, archivo)
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			comprobarNodoContratoCerrado(t, ruta, nodo)
			return true
		})
	}
}

func comprobarImportacionesCerradas(t *testing.T, ruta string, archivo *ast.File) {
	t.Helper()
	prohibidas := map[string]struct{}{
		"database/sql": {}, "net/http": {}, "os/exec": {}, "plugin": {},
	}
	for _, importacion := range archivo.Imports {
		nombre, err := strconv.Unquote(importacion.Path.Value)
		if err != nil {
			t.Fatalf("importacion ilegible en %s: %v", ruta, err)
		}
		if _, prohibida := prohibidas[nombre]; prohibida {
			t.Errorf("%s importa %q: el dominio no ejecuta SQL, red ni procesos", ruta, nombre)
		}
	}
}

func comprobarNodoContratoCerrado(t *testing.T, ruta string, nodo ast.Node) {
	t.Helper()
	switch valor := nodo.(type) {
	case *ast.BasicLit:
		if valor.Kind == token.FLOAT {
			t.Errorf("%s contiene un literal decimal: use Puntos o Racional", ruta)
		}
	case *ast.Ident:
		if valor.Name == "float32" || valor.Name == "float64" || valor.Name == "any" {
			t.Errorf("%s usa el tipo abierto o inexacto %q", ruta, valor.Name)
		}
	case *ast.InterfaceType:
		if valor.Methods == nil || len(valor.Methods.List) == 0 {
			t.Errorf("%s usa una interfaz vacia", ruta)
		}
	case *ast.SelectorExpr:
		if valor.Sel.Name == "RawMessage" {
			t.Errorf("%s usa RawMessage en el modelo gobernado", ruta)
		}
	case *ast.StructType:
		for _, campo := range valor.Fields.List {
			comprobarCampoContratoCerrado(t, ruta, campo)
		}
	}
}

func comprobarCampoContratoCerrado(t *testing.T, ruta string, campo *ast.Field) {
	t.Helper()
	switch campo.Type.(type) {
	case *ast.FuncType:
		t.Errorf("%s declara un callback configurable en una estructura", ruta)
	case *ast.MapType:
		t.Errorf("%s declara un mapa en una estructura del contrato", ruta)
	}
	for _, nombre := range campo.Names {
		normalizado := strings.ToLower(nombre.Name)
		for _, fragmento := range []string{"formula", "expresion", "script", "consulta_sql"} {
			if strings.Contains(normalizado, fragmento) {
				t.Errorf("%s declara el campo dinamico %q", ruta, nombre.Name)
			}
		}
	}
}
