package pruebas

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestFabricaAlmacenDePruebaRespetaLaMismaListaPositiva(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	contexto, err := NuevoContextoAlmacen(
		instante, "escritura", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{},
	)
	if err != nil || contexto.ValidarParaEn(ports.AccionAlmacenEscribir, instante) != nil {
		t.Fatalf("capacidad positiva de prueba invalida: %v", err)
	}
	if _, err := NuevoContextoAlmacen(
		instante, "eliminacion", ports.AccionAlmacenEliminar,
		ports.ReferenciaObjetoAlmacen{Referencia: "objeto:1", Version: "v1"},
	); !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, ports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("operacion sin fabrica aceptada: %v", err)
	}
}

func TestFabricasDePruebaNoSeImportanDesdeCodigoProductivo(t *testing.T) {
	raiz, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	const importacion = "vec-diputacion-granada/internal/vec/pruebas"
	err = filepath.WalkDir(raiz, func(ruta string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() {
			nombre := entrada.Name()
			if nombre == ".git" || nombre == "vendor" || nombre == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(ruta) != ".go" || strings.HasSuffix(ruta, "_test.go") ||
			strings.Contains(filepath.ToSlash(ruta), "/internal/vec/pruebas/") {
			return nil
		}
		archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, declaracion := range archivo.Decls {
			importes, ok := declaracion.(*ast.GenDecl)
			if !ok || importes.Tok != token.IMPORT {
				continue
			}
			for _, especificacion := range importes.Specs {
				importacionGo, ok := especificacion.(*ast.ImportSpec)
				if !ok {
					continue
				}
				valor, err := strconv.Unquote(importacionGo.Path.Value)
				if err != nil {
					return err
				}
				if valor == importacion {
					t.Errorf("codigo productivo importa fabricas de prueba: %s", ruta)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
