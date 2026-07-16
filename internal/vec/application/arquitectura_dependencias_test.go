package application

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNucleoNoDependeDeModulosNiAdaptadores(t *testing.T) {
	raizNucleo := filepath.Dir(directorioFuentesAplicacionInternoPrueba(t))
	capas := []string{"domain", "ports", "application"}

	for _, capa := range capas {
		raizCapa := filepath.Join(raizNucleo, capa)
		err := filepath.WalkDir(raizCapa, func(ruta string, entrada fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entrada.IsDir() || filepath.Ext(entrada.Name()) != ".go" || strings.HasSuffix(entrada.Name(), "_test.go") {
				return nil
			}
			archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, importacion := range archivo.Imports {
				rutaImportada, err := strconv.Unquote(importacion.Path.Value)
				if err != nil {
					return err
				}
				if strings.Contains(rutaImportada, "/internal/modules/") ||
					strings.Contains(rutaImportada, "/internal/vec/adapters/") {
					relativa, _ := filepath.Rel(raizNucleo, ruta)
					t.Errorf("el núcleo hexagonal %s importa una dependencia exterior: %s", relativa, rutaImportada)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("recorrer capa %s: %v", capa, err)
		}
	}
}
