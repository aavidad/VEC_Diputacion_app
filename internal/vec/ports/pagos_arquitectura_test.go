package ports

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFabricasDeEvidenciaSoloSeInvocanEnAdaptadoresDePasarela(t *testing.T) {
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	raiz := filepath.Clean(filepath.Join(directorio, "../../.."))
	permitido := filepath.ToSlash(filepath.Join("internal", "vec", "adapters", "pagos")) + "/"
	err = filepath.WalkDir(raiz, func(ruta string, entrada fs.DirEntry, errorRecorrido error) error {
		if errorRecorrido != nil {
			return errorRecorrido
		}
		if entrada.IsDir() {
			switch entrada.Name() {
			case ".git", ".worktrees", "vendor", "node_modules":
				return fs.SkipDir
			default:
				return nil
			}
		}
		if filepath.Ext(ruta) != ".go" || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		relativa, errorRelativa := filepath.Rel(raiz, ruta)
		if errorRelativa != nil {
			return errorRelativa
		}
		relativa = filepath.ToSlash(relativa)
		if relativa == "internal/vec/domain/pagos.go" || strings.HasPrefix(relativa, permitido) {
			return nil
		}
		archivo, errorParseo := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
		if errorParseo != nil {
			return errorParseo
		}
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			llamada, esLlamada := nodo.(*ast.CallExpr)
			if !esLlamada {
				return true
			}
			nombre := ""
			switch funcion := llamada.Fun.(type) {
			case *ast.Ident:
				nombre = funcion.Name
			case *ast.SelectorExpr:
				nombre = funcion.Sel.Name
			}
			fabricasReservadas := map[string]struct{}{
				"NuevaEvidenciaInicioOperacionCobroVerificada":     {},
				"NuevaEvidenciaResultadoCobroVerificada":           {},
				"NuevaEvidenciaResultadoDevolucionCobroVerificada": {},
				"NuevaEvidenciaConciliacionCobroVerificada":        {},
			}
			if _, reservada := fabricasReservadas[nombre]; reservada {
				t.Errorf("%s invoca una fabrica reservada al adaptador verificador", relativa)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer arquitectura: %v", err)
	}
}
