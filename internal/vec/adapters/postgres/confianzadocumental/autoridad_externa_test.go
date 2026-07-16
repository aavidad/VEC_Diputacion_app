package confianzadocumental_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	confianza "vec-diputacion-granada/internal/vec/adapters/postgres/confianzadocumental"
)

func TestAPIExternaNoPuedeFabricarAutoridadCOSE(t *testing.T) {
	var cero confianza.PruebaCOSESign1DocumentalVerificada
	if !errors.Is(cero.Validar(), confianza.ErrPruebaCOSESign1VerificadaInvalida) {
		t.Fatal("el unico literal construible externamente conservo autoridad")
	}
	tipo := reflect.TypeOf(cero)
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).PkgPath == "" {
			t.Fatalf("campo de autoridad exportado: %s", tipo.Field(indice).Name)
		}
	}
	for nombre, valor := range map[string]any{
		"raiz":          confianza.RaizPublicaFijada{},
		"configuracion": confianza.ConfiguracionConfianzaFijada{},
		"servicio":      confianza.Servicio{},
	} {
		tipo := reflect.TypeOf(valor)
		for indice := 0; indice < tipo.NumField(); indice++ {
			if tipo.Field(indice).PkgPath == "" {
				t.Fatalf("%s expone el campo %s", nombre, tipo.Field(indice).Name)
			}
		}
	}

	directorio := directorioFuentesConfianzaPrueba(t)
	paquetes, err := parser.ParseDir(
		token.NewFileSet(), directorio,
		func(info fs.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") },
		0,
	)
	if err != nil {
		t.Fatalf("analizar superficie publica: %v", err)
	}
	paquete, existe := paquetes["confianzadocumental"]
	if !existe {
		t.Fatal("no se encontro el paquete productivo")
	}
	tiposProtegidos := map[string]struct{}{
		"PruebaCOSESign1DocumentalVerificada": {},
		"RaizPublicaFijada":                   {},
		"ConfiguracionConfianzaFijada":        {},
		"Servicio":                            {},
	}
	for _, archivo := range paquete.Files {
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			funcion, ok := nodo.(*ast.FuncDecl)
			if !ok || !ast.IsExported(funcion.Name.Name) || funcion.Type.Results == nil {
				return true
			}
			for _, resultado := range funcion.Type.Results.List {
				nombreTipo := nombreTipoResultado(resultado.Type)
				if _, protegido := tiposProtegidos[nombreTipo]; protegido {
					permitido := nombreTipo == "PruebaCOSESign1DocumentalVerificada" &&
						funcion.Name.Name == "VerificarCOSESign1" &&
						nombreReceptor(funcion.Recv) == "Servicio"
					if permitido {
						continue
					}
					t.Errorf("fabrica publica de %s detectada: %s", nombreTipo, funcion.Name.Name)
				}
			}
			return true
		})
	}
}

func nombreReceptor(receptor *ast.FieldList) string {
	if receptor == nil || len(receptor.List) != 1 {
		return ""
	}
	return nombreTipoResultado(receptor.List[0].Type)
}

func nombreTipoResultado(expresion ast.Expr) string {
	switch tipo := expresion.(type) {
	case *ast.Ident:
		return tipo.Name
	case *ast.StarExpr:
		return nombreTipoResultado(tipo.X)
	default:
		return ""
	}
}
