package ports_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	metodoConsumidorNominalReservado   = "ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion"
	interfazConsumidorNominalReservada = "ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion"
	fabricaProductoNominalReservada    = "ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion"
	constructorConsumidorReservado     = "nuevoConsumidorCASIdempotenciaBaremacion"
	rutaConsumidorNominalPermitida     = "internal/modules/bolsa/adapters/postgres/confianzaidempotencia/consumidor_cas.go"
	rutaServicioNominalPermitida       = "internal/modules/bolsa/adapters/postgres/confianzaidempotencia/servicio.go"
	rutaContratoNominalPermitida       = "internal/modules/bolsa/ports/idempotencia_semantica_baremacion.go"
	rutaBootstrapNominalPermitida      = "internal/app/bootstrap/idempotencia_baremacion.go"
	paqueteConsumidorNominalPermitido  = "confianzaidempotencia"
	paqueteBootstrapNominalPermitido   = "bootstrap"
	receptorConsumidorNominalPermitido = "consumidorCASIdempotenciaBaremacion"
	importacionTCBIdempotencia         = "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres/confianzaidempotencia"
	importacionPuertosBolsa            = "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestConsumidorNominalSoloPuedeVivirEnTCBPrivadaExacta(t *testing.T) {
	raiz := localizarRaizRepositorioIdempotencia(t)
	implementacionesPermitidas := 0
	asercionesCompilacionPermitidas := 0

	err := filepath.WalkDir(raiz, func(ruta string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() {
			if omitirDirectorioAnalisisIdempotencia(raiz, ruta, entrada.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(ruta) != ".go" || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}

		unidad, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		relativa, err := rutaRelativaCanonicaIdempotencia(raiz, ruta)
		if err != nil {
			return err
		}

		for _, declaracion := range unidad.Decls {
			funcion, ok := declaracion.(*ast.FuncDecl)
			if !ok || funcion.Recv == nil || funcion.Name.Name != metodoConsumidorNominalReservado {
				continue
			}
			receptor, esPuntero := receptorMetodoIdempotencia(funcion)
			if !implementacionConsumidorNominalPermitida(
				relativa, unidad.Name.Name, receptor, esPuntero,
			) {
				t.Errorf("implementacion nominal fuera de la TCB privada exacta: %s", relativa)
				continue
			}
			implementacionesPermitidas++
		}

		ast.Inspect(unidad, func(nodo ast.Node) bool {
			identificador, ok := nodo.(*ast.Ident)
			if !ok || !esNombreNominalReservado(identificador.Name) {
				return true
			}
			if !referenciaNominalPermitida(relativa, identificador.Name) {
				t.Errorf("referencia nominal reservada %s fuera de su frontera: %s", identificador.Name, relativa)
			}
			return true
		})

		for _, importacion := range unidad.Imports {
			rutaImportada, err := strconv.Unquote(importacion.Path.Value)
			if err != nil {
				return err
			}
			if rutaImportada == importacionTCBIdempotencia &&
				!importacionTCBPermitida(relativa, unidad.Name.Name) {
				t.Errorf("importacion de la TCB de idempotencia fuera del bootstrap exacto: %s", relativa)
			}
		}

		if relativa == rutaConsumidorNominalPermitida {
			if tieneRestriccionCompilacionIdempotencia(unidad) {
				t.Errorf("el consumidor CAS exacto no puede quedar condicionado por etiquetas de compilacion: %s", relativa)
			}
			asercionesCompilacionPermitidas += contarAsercionesConsumidorNominal(unidad)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	rutaEsperada := filepath.Join(raiz, filepath.FromSlash(rutaConsumidorNominalPermitida))
	_, err = os.Stat(rutaEsperada)
	switch {
	case err == nil && implementacionesPermitidas != 1:
		t.Fatalf("la TCB privada debe contener una implementacion exacta: %d", implementacionesPermitidas)
	case err == nil && asercionesCompilacionPermitidas != 1:
		t.Fatalf("la TCB privada debe acreditar una asercion compilable exacta: %d", asercionesCompilacionPermitidas)
	case err != nil && !os.IsNotExist(err):
		t.Fatal(err)
	case os.IsNotExist(err) && implementacionesPermitidas != 0:
		t.Fatalf("se encontro una implementacion sin fichero TCB: %d", implementacionesPermitidas)
	case os.IsNotExist(err) && asercionesCompilacionPermitidas != 0:
		t.Fatalf("se encontro una asercion sin fichero TCB: %d", asercionesCompilacionPermitidas)
	}
}

func TestListaPositivaConsumidorNominalRechazaUbicacionesYReceptoresAlternativos(t *testing.T) {
	casos := []struct {
		nombre    string
		ruta      string
		paquete   string
		receptor  string
		esPuntero bool
		permitir  bool
	}{
		{"TCB exacta por puntero", rutaConsumidorNominalPermitida, paqueteConsumidorNominalPermitido, receptorConsumidorNominalPermitido, true, true},
		{"receptor por valor", rutaConsumidorNominalPermitida, paqueteConsumidorNominalPermitido, receptorConsumidorNominalPermitido, false, false},
		{"otro fichero TCB", "internal/modules/bolsa/adapters/postgres/confianzaidempotencia/otro.go", paqueteConsumidorNominalPermitido, receptorConsumidorNominalPermitido, true, false},
		{"receptor exportado", rutaConsumidorNominalPermitida, paqueteConsumidorNominalPermitido, "ConsumidorCASIdempotenciaBaremacion", true, false},
		{"otro paquete", rutaConsumidorNominalPermitida, "application", receptorConsumidorNominalPermitido, true, false},
		{"handler interno", "internal/vec/adapters/httpapi/handler.go", "httpapi", receptorConsumidorNominalPermitido, true, false},
		{"ruta no canonica", "interno/../" + rutaConsumidorNominalPermitida, paqueteConsumidorNominalPermitido, receptorConsumidorNominalPermitido, true, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			obtenido := implementacionConsumidorNominalPermitida(
				caso.ruta, caso.paquete, caso.receptor, caso.esPuntero,
			)
			if obtenido != caso.permitir {
				t.Fatalf("permiso inesperado: obtenido=%t esperado=%t", obtenido, caso.permitir)
			}
		})
	}
}

func TestReferenciasNominalesQuedanLimitadasAlContratoYLaTCBExactos(t *testing.T) {
	casos := []struct {
		nombre   string
		ruta     string
		simbolo  string
		permitir bool
	}{
		{"interfaz en contrato", rutaContratoNominalPermitida, interfazConsumidorNominalReservada, true},
		{"metodo en contrato", rutaContratoNominalPermitida, metodoConsumidorNominalReservado, true},
		{"fabrica en contrato", rutaContratoNominalPermitida, fabricaProductoNominalReservada, true},
		{"interfaz en consumidor", rutaConsumidorNominalPermitida, interfazConsumidorNominalReservada, true},
		{"metodo en consumidor", rutaConsumidorNominalPermitida, metodoConsumidorNominalReservado, true},
		{"fabrica en servicio", rutaServicioNominalPermitida, fabricaProductoNominalReservada, true},
		{"constructor en consumidor", rutaConsumidorNominalPermitida, constructorConsumidorReservado, true},
		{"constructor en servicio", rutaServicioNominalPermitida, constructorConsumidorReservado, true},
		{"receptor en consumidor", rutaConsumidorNominalPermitida, receptorConsumidorNominalPermitido, true},
		{"fabrica en handler", "internal/vec/adapters/httpapi/handler.go", fabricaProductoNominalReservada, false},
		{"interfaz en handler", "internal/vec/adapters/httpapi/handler.go", interfazConsumidorNominalReservada, false},
		{"metodo en handler", "internal/vec/adapters/httpapi/handler.go", metodoConsumidorNominalReservado, false},
		{"constructor en handler", "internal/vec/adapters/httpapi/handler.go", constructorConsumidorReservado, false},
		{"receptor embebido en otro fichero TCB", "internal/modules/bolsa/adapters/postgres/confianzaidempotencia/puente.go", receptorConsumidorNominalPermitido, false},
		{"interfaz en otro fichero TCB", "internal/modules/bolsa/adapters/postgres/confianzaidempotencia/puente.go", interfazConsumidorNominalReservada, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenido := referenciaNominalPermitida(caso.ruta, caso.simbolo); obtenido != caso.permitir {
				t.Fatalf("permiso inesperado: obtenido=%t esperado=%t", obtenido, caso.permitir)
			}
		})
	}
}

func TestImportacionTCBSoloSePermiteEnBootstrapExacto(t *testing.T) {
	casos := []struct {
		nombre   string
		ruta     string
		paquete  string
		permitir bool
	}{
		{"bootstrap exacto", rutaBootstrapNominalPermitida, paqueteBootstrapNominalPermitido, true},
		{"otro fichero bootstrap", "internal/app/bootstrap/otro.go", paqueteBootstrapNominalPermitido, false},
		{"paquete falso", rutaBootstrapNominalPermitida, "httpapi", false},
		{"handler", "internal/vec/adapters/httpapi/handler.go", "httpapi", false},
		{"programa", "cmd/vec-api/main.go", "main", false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenido := importacionTCBPermitida(caso.ruta, caso.paquete); obtenido != caso.permitir {
				t.Fatalf("permiso inesperado: obtenido=%t esperado=%t", obtenido, caso.permitir)
			}
		})
	}
}

func TestAsercionCompilableExigeInterfazDePuertosYPunteroPrivadoExactos(t *testing.T) {
	casos := []struct {
		nombre   string
		codigo   string
		esperado int
	}{
		{
			"importacion ordinaria",
			`package confianzaidempotencia
import "vec-diputacion-granada/internal/modules/bolsa/ports"
type consumidorCASIdempotenciaBaremacion struct{}
var _ ports.ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion = (*consumidorCASIdempotenciaBaremacion)(nil)`,
			1,
		},
		{
			"alias explicito",
			`package confianzaidempotencia
import contratos "vec-diputacion-granada/internal/modules/bolsa/ports"
type consumidorCASIdempotenciaBaremacion struct{}
var _ contratos.ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion = (*consumidorCASIdempotenciaBaremacion)(nil)`,
			1,
		},
		{
			"interfaz local homonima",
			`package confianzaidempotencia
type ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion interface{}
type consumidorCASIdempotenciaBaremacion struct{}
var _ ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion = (*consumidorCASIdempotenciaBaremacion)(nil)`,
			0,
		},
		{
			"constructor en vez de conversion explicita",
			`package confianzaidempotencia
import "vec-diputacion-granada/internal/modules/bolsa/ports"
type consumidorCASIdempotenciaBaremacion struct{}
var _ ports.ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion = new(consumidorCASIdempotenciaBaremacion)`,
			0,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			unidad, err := parser.ParseFile(token.NewFileSet(), "consumidor_cas.go", caso.codigo, 0)
			if err != nil {
				t.Fatal(err)
			}
			if obtenido := contarAsercionesConsumidorNominal(unidad); obtenido != caso.esperado {
				t.Fatalf("aserciones inesperadas: obtenido=%d esperado=%d", obtenido, caso.esperado)
			}
		})
	}
}

func implementacionConsumidorNominalPermitida(ruta, paquete, receptor string, esPuntero bool) bool {
	return rutaCanonicaExactaIdempotencia(ruta, rutaConsumidorNominalPermitida) &&
		paquete == paqueteConsumidorNominalPermitido &&
		receptor == receptorConsumidorNominalPermitido &&
		esPuntero
}

func referenciaNominalPermitida(ruta, simbolo string) bool {
	switch simbolo {
	case interfazConsumidorNominalReservada, metodoConsumidorNominalReservado:
		return rutaCanonicaExactaIdempotencia(ruta, rutaContratoNominalPermitida) ||
			rutaCanonicaExactaIdempotencia(ruta, rutaConsumidorNominalPermitida)
	case fabricaProductoNominalReservada:
		return rutaCanonicaExactaIdempotencia(ruta, rutaContratoNominalPermitida) ||
			rutaCanonicaExactaIdempotencia(ruta, rutaServicioNominalPermitida)
	case constructorConsumidorReservado:
		return rutaCanonicaExactaIdempotencia(ruta, rutaConsumidorNominalPermitida) ||
			rutaCanonicaExactaIdempotencia(ruta, rutaServicioNominalPermitida)
	case receptorConsumidorNominalPermitido:
		return rutaCanonicaExactaIdempotencia(ruta, rutaConsumidorNominalPermitida)
	default:
		return false
	}
}

func importacionTCBPermitida(ruta, paquete string) bool {
	return rutaCanonicaExactaIdempotencia(ruta, rutaBootstrapNominalPermitida) &&
		paquete == paqueteBootstrapNominalPermitido
}

func esNombreNominalReservado(nombre string) bool {
	return nombre == metodoConsumidorNominalReservado ||
		nombre == interfazConsumidorNominalReservada ||
		nombre == fabricaProductoNominalReservada ||
		nombre == constructorConsumidorReservado ||
		nombre == receptorConsumidorNominalPermitido
}

func tieneRestriccionCompilacionIdempotencia(unidad *ast.File) bool {
	for _, grupo := range unidad.Comments {
		if grupo.Pos() >= unidad.Package {
			continue
		}
		for _, comentario := range grupo.List {
			texto := strings.TrimSpace(comentario.Text)
			if strings.HasPrefix(texto, "//go:build") || strings.HasPrefix(texto, "// +build") {
				return true
			}
		}
	}
	return false
}

func rutaCanonicaExactaIdempotencia(ruta, esperada string) bool {
	rutaConBarras := filepath.ToSlash(ruta)
	rutaLimpia := filepath.ToSlash(filepath.Clean(ruta))
	return rutaConBarras == rutaLimpia && rutaLimpia == esperada
}

func receptorMetodoIdempotencia(funcion *ast.FuncDecl) (string, bool) {
	if funcion == nil || funcion.Recv == nil || len(funcion.Recv.List) != 1 {
		return "", false
	}
	tipo := funcion.Recv.List[0].Type
	puntero, esPuntero := tipo.(*ast.StarExpr)
	if esPuntero {
		tipo = puntero.X
	}
	identificador, _ := tipo.(*ast.Ident)
	if identificador == nil {
		return "", esPuntero
	}
	return identificador.Name, esPuntero
}

func contarAsercionesConsumidorNominal(unidad *ast.File) int {
	aliasPuertos := aliasImportacion(unidad, importacionPuertosBolsa)
	if aliasPuertos == "" {
		return 0
	}
	contador := 0
	for _, declaracion := range unidad.Decls {
		grupo, ok := declaracion.(*ast.GenDecl)
		if !ok || grupo.Tok != token.VAR {
			continue
		}
		for _, especificacion := range grupo.Specs {
			valor, ok := especificacion.(*ast.ValueSpec)
			if !ok || len(valor.Names) != 1 || valor.Names[0].Name != "_" || len(valor.Values) != 1 ||
				!esInterfazConsumidoraPuertos(valor.Type, aliasPuertos) ||
				!esConversionPunteroConsumidorNil(valor.Values[0]) {
				continue
			}
			contador++
		}
	}
	return contador
}

func aliasImportacion(unidad *ast.File, rutaEsperada string) string {
	for _, importacion := range unidad.Imports {
		ruta, err := strconv.Unquote(importacion.Path.Value)
		if err != nil || ruta != rutaEsperada {
			continue
		}
		if importacion.Name == nil {
			return filepath.Base(rutaEsperada)
		}
		if importacion.Name.Name != "_" && importacion.Name.Name != "." {
			return importacion.Name.Name
		}
	}
	return ""
}

func esInterfazConsumidoraPuertos(expresion ast.Expr, aliasPuertos string) bool {
	selector, ok := expresion.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != interfazConsumidorNominalReservada {
		return false
	}
	paquete, ok := selector.X.(*ast.Ident)
	return ok && paquete.Name == aliasPuertos
}

func esConversionPunteroConsumidorNil(expresion ast.Expr) bool {
	llamada, ok := expresion.(*ast.CallExpr)
	if !ok || len(llamada.Args) != 1 {
		return false
	}
	argumento, ok := llamada.Args[0].(*ast.Ident)
	if !ok || argumento.Name != "nil" {
		return false
	}
	tipo := quitarParentesisIdempotencia(llamada.Fun)
	puntero, ok := tipo.(*ast.StarExpr)
	if !ok {
		return false
	}
	receptor, ok := puntero.X.(*ast.Ident)
	return ok && receptor.Name == receptorConsumidorNominalPermitido
}

func quitarParentesisIdempotencia(expresion ast.Expr) ast.Expr {
	for {
		parentesis, ok := expresion.(*ast.ParenExpr)
		if !ok {
			return expresion
		}
		expresion = parentesis.X
	}
}

func localizarRaizRepositorioIdempotencia(t *testing.T) string {
	t.Helper()
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	directorio, err = filepath.Abs(directorio)
	if err != nil {
		t.Fatal(err)
	}
	for {
		informacion, err := os.Stat(filepath.Join(directorio, "go.mod"))
		if err == nil && !informacion.IsDir() {
			return directorio
		}
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		padre := filepath.Dir(directorio)
		if padre == directorio {
			t.Fatal("no se encontro go.mod al ascender desde el directorio de pruebas")
		}
		directorio = padre
	}
}

func rutaRelativaCanonicaIdempotencia(raiz, ruta string) (string, error) {
	relativa, err := filepath.Rel(raiz, ruta)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relativa), nil
}

func omitirDirectorioAnalisisIdempotencia(raiz, ruta, nombre string) bool {
	if ruta == raiz {
		return false
	}
	if nombre == ".git" || nombre == ".worktrees" || nombre == "vendor" ||
		nombre == "node_modules" || nombre == "var" {
		return true
	}
	if _, err := os.Stat(filepath.Join(ruta, "go.mod")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(ruta, ".git")); err == nil {
		return true
	}
	return false
}
