package calculoexperiencia

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

func TestArquitecturaExactaCerrada(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("leer paquete: %v", err)
	}
	for _, entrada := range entradas {
		if entrada.IsDir() || filepath.Ext(entrada.Name()) != ".go" ||
			strings.HasSuffix(entrada.Name(), "_test.go") {
			continue
		}
		archivo, err := parser.ParseFile(token.NewFileSet(), entrada.Name(), nil, 0)
		if err != nil {
			t.Fatalf("analizar %s: %v", entrada.Name(), err)
		}
		comprobarImportacionesExactas(t, entrada.Name(), archivo)
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			comprobarNodoExacto(t, entrada.Name(), nodo)
			return true
		})
	}
}

func TestArquitecturaDetectaIdentificadoresPersonalesCompuestos(t *testing.T) {
	for _, nombre := range []string{
		"PersonaID", "DNIHash", "datoSalud", "correo_electronico", "IBANRef",
	} {
		if !nombrePersonalProhibido(nombre) {
			t.Errorf("no se detecto %q", nombre)
		}
	}
	for _, nombre := range []string{"PeriodoServicio", "PuntosPorUnidad", "catalogoRef"} {
		if nombrePersonalProhibido(nombre) {
			t.Errorf("falso positivo para %q", nombre)
		}
	}
}

func comprobarImportacionesExactas(t *testing.T, ruta string, archivo *ast.File) {
	t.Helper()
	permitidas := map[string]struct{}{
		"errors":   {},
		"fmt":      {},
		"math/big": {},
		"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo": {},
		"vec-diputacion-granada/internal/shared/baremacion":                 {},
	}
	porArchivo := map[string][]string{
		"canonico.go": {
			"bytes", "crypto/sha256", "crypto/subtle", "encoding/hex",
			"encoding/json", "io", "unicode/utf8",
		},
		"entrada.go":               {"sort", "strings"},
		"resultado_exacto.go":      {"strings"},
		"resultado_registrador.go": {"sort"},
		"resultado_material.go":    {"bytes", "encoding/json"},
		"resultado_canonico.go": {
			"crypto/sha256", "encoding/hex", "encoding/json",
		},
		"resultado_restauracion.go": {
			"bytes", "crypto/subtle", "encoding/json", "io", "unicode/utf8",
		},
	}
	for _, nombre := range porArchivo[filepath.Base(ruta)] {
		permitidas[nombre] = struct{}{}
	}
	for _, importacion := range archivo.Imports {
		nombre, err := strconv.Unquote(importacion.Path.Value)
		if err != nil {
			t.Fatalf("importacion ilegible en %s: %v", ruta, err)
		}
		if _, permitida := permitidas[nombre]; !permitida {
			t.Errorf("%s importa %q fuera del limite aritmetico cerrado", ruta, nombre)
		}
	}
}

func comprobarNodoExacto(t *testing.T, ruta string, nodo ast.Node) {
	t.Helper()
	switch valor := nodo.(type) {
	case *ast.BasicLit:
		if valor.Kind == token.FLOAT {
			t.Errorf("%s contiene un literal flotante", ruta)
		}
		if valor.Kind == token.STRING {
			texto, err := strconv.Unquote(valor.Value)
			if err == nil && pareceSQL(texto) {
				t.Errorf("%s contiene SQL en una cadena de produccion", ruta)
			}
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
			t.Errorf("%s usa una configuracion JSON abierta", ruta)
		}
	case *ast.StructType:
		for _, campo := range valor.Fields.List {
			if _, ok := campo.Type.(*ast.FuncType); ok {
				t.Errorf("%s declara un callback en una estructura", ruta)
			}
			if _, ok := campo.Type.(*ast.MapType); ok {
				t.Errorf("%s declara un mapa dinamico en una estructura", ruta)
			}
			if campo.Tag != nil {
				etiqueta, err := strconv.Unquote(campo.Tag.Value)
				if err == nil {
					for _, parte := range strings.Fields(etiqueta) {
						if strings.HasPrefix(parte, `json:"`) {
							nombreJSON := strings.Split(strings.TrimPrefix(parte, `json:"`), ",")[0]
							nombreJSON = strings.TrimSuffix(nombreJSON, `"`)
							if nombrePersonalProhibido(nombreJSON) {
								t.Errorf("%s declara la clave JSON personal %q", ruta, nombreJSON)
							}
						}
					}
				}
			}
			for _, nombre := range campo.Names {
				normalizado := strings.ToLower(nombre.Name)
				for _, fragmento := range []string{"formula", "expresion", "script", "consulta", "plantilla"} {
					if strings.Contains(normalizado, fragmento) {
						t.Errorf("%s declara el campo dinamico %q", ruta, nombre.Name)
					}
				}
				if nombrePersonalProhibido(nombre.Name) {
					t.Errorf("%s declara el dato personal %q", ruta, nombre.Name)
				}
			}
		}
	case *ast.FuncDecl:
		if valor.Name.IsExported() && contieneBigMutable(valor.Type) {
			t.Errorf("%s expone math/big mediante %s", ruta, valor.Name.Name)
		}
	}
}

func nombrePersonalProhibido(nombre string) bool {
	normalizado := strings.ToLower(nombre)
	for _, prohibido := range []string{
		"dni", "nif", "nie", "nombre", "apellido", "apellidos", "persona",
		"diagnostico", "causa", "motivo", "direccion", "telefono", "correo",
		"email", "salud", "nacimiento", "iban", "nss", "naf",
	} {
		if strings.Contains(normalizado, prohibido) {
			return true
		}
	}
	return false
}

func contieneBigMutable(tipo *ast.FuncType) bool {
	encontrado := false
	ast.Inspect(tipo, func(nodo ast.Node) bool {
		selector, ok := nodo.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		paquete, ok := selector.X.(*ast.Ident)
		if ok && paquete.Name == "big" && (selector.Sel.Name == "Int" || selector.Sel.Name == "Rat") {
			encontrado = true
		}
		return true
	})
	return encontrado
}

func pareceSQL(texto string) bool {
	campos := strings.Fields(strings.ToLower(strings.TrimSpace(texto)))
	if len(campos) == 0 {
		return false
	}
	primera := strings.TrimFunc(campos[0], func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	switch primera {
	case "select", "insert", "update", "delete", "create", "alter", "drop", "truncate":
		return true
	default:
		return false
	}
}
