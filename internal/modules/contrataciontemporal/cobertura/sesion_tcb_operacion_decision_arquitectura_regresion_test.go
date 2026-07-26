package cobertura

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

type importadorFronteraNominalPrueba struct {
	cobertura *types.Package
}

func (i importadorFronteraNominalPrueba) Import(
	ruta string,
) (*types.Package, error) {
	if ruta == rutaPaqueteCoberturaSesionTCB {
		return i.cobertura, nil
	}
	return nil, fmt.Errorf("importación inesperada en regresión: %s", ruta)
}

func analizarBypassFronteraNominalPrueba(
	fuente string,
	coberturaImportada *types.Package,
) (
	paqueteAnalizadoSesionTCB,
	*token.FileSet,
	error,
) {
	conjunto := token.NewFileSet()
	archivo, err := parser.ParseFile(
		conjunto,
		"bypass_frontera_nominal.go",
		fuente,
		0,
	)
	if err != nil {
		return paqueteAnalizadoSesionTCB{}, nil, err
	}
	info := &types.Info{
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Types:      make(map[ast.Expr]types.TypeAndValue),
	}
	const ruta = "vec-diputacion-granada/pruebas/bypass-frontera"
	paquete, err := (&types.Config{
		Importer: importadorFronteraNominalPrueba{
			cobertura: coberturaImportada,
		},
	}).Check(ruta, conjunto, []*ast.File{archivo}, info)
	if err != nil {
		return paqueteAnalizadoSesionTCB{}, nil, err
	}
	return paqueteAnalizadoSesionTCB{
		metadatos: paqueteProductivoSesionTCB{ImportPath: ruta},
		tipos:     paquete,
		info:      info,
		ficheros:  []*ast.File{archivo},
	}, conjunto, nil
}

func TestArquitecturaInventarioNominalDetectaCadaBypass(
	t *testing.T,
) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	const cabecera = `package bypass
import cobertura "vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
`
	casos := []struct {
		nombre         string
		fuente         string
		origenEsperado string
	}{
		{
			nombre: "TypeName alias con nombre sin TCB",
			fuente: cabecera + `
type AccesoOperacion = cobertura.TransaccionOperacionDecisionCobertura
`,
			origenEsperado: "alias o reexportación nominal AccesoOperacion",
		},
		{
			nombre: "alias compuesto reexporta sin implementar",
			fuente: cabecera + `
type Transporte = []cobertura.TransaccionOperacionDecisionCobertura
`,
			origenEsperado: "alias o reexportación nominal Transporte",
		},
		{
			nombre: "TypeName de interfaz definida embebida",
			fuente: cabecera + `
type AccesoOperacion interface {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "TypeName nominal AccesoOperacion",
		},
		{
			nombre: "interfaz embebida explícita",
			fuente: cabecera + `
type AccesoOperacion interface {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "interfaz embebida",
		},
		{
			nombre: "alias de interfaz embebida",
			fuente: cabecera + `
type AccesoOperacion = interface {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "alias o reexportación nominal AccesoOperacion",
		},
		{
			nombre: "interfaz anónima en variable",
			fuente: cabecera + `
var acceso interface {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "variable acceso",
		},
		{
			nombre: "función con tipos anónimos",
			fuente: cabecera + `
func transportar(
	acceso interface {
		cobertura.TransaccionOperacionDecisionCobertura
	},
) interface {
	cobertura.TransaccionOperacionDecisionCobertura
} {
	return acceso
}
`,
			origenEsperado: "función transportar",
		},
		{
			nombre: "campo con tipo anónimo",
			fuente: cabecera + `
type Servicio struct {
	acceso interface {
		cobertura.TransaccionOperacionDecisionCobertura
	}
}
`,
			origenEsperado: "campo acceso",
		},
		{
			nombre: "struct nombrado con embedding",
			fuente: cabecera + `
type AccesoOperacion struct {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "campo embebido de struct",
		},
		{
			nombre: "variable de struct anónimo",
			fuente: cabecera + `
var acceso struct {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "variable acceso",
		},
		{
			nombre: "literal local de struct anónimo",
			fuente: cabecera + `
func ocultar() {
	_ = struct {
		cobertura.TransaccionOperacionDecisionCobertura
	}{}
}
`,
			origenEsperado: "tipo struct anónimo",
		},
		{
			nombre: "alias local sin TCB",
			fuente: cabecera + `
func ocultar() {
	type Pasarela = cobertura.TransaccionOperacionDecisionCobertura
	var _ Pasarela
}
`,
			origenEsperado: "alias o reexportación nominal Pasarela",
		},
		{
			nombre: "variable de función anónima",
			fuente: cabecera + `
var ejecutar func(interface {
	cobertura.TransaccionOperacionDecisionCobertura
})
`,
			origenEsperado: "variable ejecutar",
		},
		{
			nombre: "parámetro de tipo con restricción anónima",
			fuente: cabecera + `
type Caja[T interface {
	cobertura.TransaccionOperacionDecisionCobertura
}] struct {
	valor T
}
`,
			origenEsperado: "tipo interface anónimo",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			paquete, _, err := analizarBypassFronteraNominalPrueba(
				caso.fuente,
				analisis.cobertura,
			)
			if err != nil {
				t.Fatal(err)
			}
			hallazgos, err := hallarIncorporacionesFronteraNominal(
				paquete,
				analisis.cobertura,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(hallazgos) == 0 {
				t.Fatal("el bypass no fue detectado")
			}
			if !hallazgosContienenOrigen(
				hallazgos,
				caso.origenEsperado,
			) {
				t.Fatalf(
					"no se detectó por %q; hallazgos: %v",
					caso.origenEsperado,
					hallazgos,
				)
			}
			for _, hallazgo := range hallazgos {
				if hallazgo.tipo == "invalid type" {
					t.Fatal("el bypass solo produjo ruido de go/types")
				}
			}
		})
	}
}

func hallazgosContienenOrigen(
	hallazgos []hallazgoFronteraNominal,
	esperado string,
) bool {
	for _, hallazgo := range hallazgos {
		if strings.Contains(hallazgo.origen, esperado) {
			return true
		}
	}
	return false
}

func TestArquitecturaFronteraNominalNoEsForjableFueraDelPaquete(
	t *testing.T,
) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	metodoSellado := analisis.interfazNominal.ExplicitMethod(0)
	firmaSellada, correcta := metodoSellado.Type().(*types.Signature)
	if !correcta {
		t.Fatal("el método sellado dejó de tener firma")
	}

	paqueteAjeno := types.NewPackage(
		"vec-diputacion-granada/pruebas/forja-frontera",
		"forja",
	)
	nombre := types.NewTypeName(
		token.NoPos,
		paqueteAjeno,
		"TransaccionFalsa",
		nil,
	)
	falso := types.NewNamed(nombre, types.NewStruct(nil, nil), nil)
	receptor := types.NewVar(
		token.NoPos,
		paqueteAjeno,
		"",
		types.NewPointer(falso),
	)
	firmaFalsa := types.NewSignatureType(
		receptor,
		nil,
		nil,
		firmaSellada.Params(),
		firmaSellada.Results(),
		firmaSellada.Variadic(),
	)
	falso.AddMethod(types.NewFunc(
		token.NoPos,
		paqueteAjeno,
		metodoSellado.Name(),
		firmaFalsa,
	))

	if types.Implements(falso, analisis.interfazNominal) ||
		types.Implements(types.NewPointer(falso), analisis.interfazNominal) {
		t.Fatal("un paquete ajeno forjó la operación privada de la frontera")
	}
}

func TestArquitecturaInventarioNominalAdmiteDependenciaDirecta(
	t *testing.T,
) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	const fuente = `package bypass
import cobertura "vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"

type Servicio struct {
	transaccion cobertura.TransaccionOperacionDecisionCobertura
}

func nuevoServicio(
	transaccion cobertura.TransaccionOperacionDecisionCobertura,
) *Servicio {
	return &Servicio{transaccion: transaccion}
}
`
	paquete, conjunto, err := analizarBypassFronteraNominalPrueba(
		fuente,
		analisis.cobertura,
	)
	if err != nil {
		t.Fatal(err)
	}
	hallazgos, err := hallarIncorporacionesFronteraNominal(
		paquete,
		analisis.cobertura,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, hallazgo := range hallazgos {
		t.Errorf(
			"dependencia directa clasificada como bypass: %s en %s",
			hallazgo.tipo,
			conjunto.Position(hallazgo.posicion),
		)
	}
}

func TestArquitecturaInventarioGoTypesEnumeraTodasLasCategorias(
	t *testing.T,
) {
	analisis := obtenerAnalisisArquitecturaSesionTCB(t)
	const fuente = `package bypass
import cobertura "vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"

type Escalar = int
type Nominal int
type Base interface {
	base()
}
type Extendida interface {
	Base
}
type Registro struct {
	Nominal
	transaccion cobertura.TransaccionOperacionDecisionCobertura
	anonima interface {
		permitida()
	}
}

const marca Nominal = 1
var permitida cobertura.TransaccionOperacionDecisionCobertura
var lista []int
var indice map[string]int
var cola chan int
var ejecutar func(cobertura.TransaccionOperacionDecisionCobertura)

func usar(
	transaccion cobertura.TransaccionOperacionDecisionCobertura,
) cobertura.TransaccionOperacionDecisionCobertura {
	return transaccion
}
`
	paquete, _, err := analizarBypassFronteraNominalPrueba(
		fuente,
		analisis.cobertura,
	)
	if err != nil {
		t.Fatal(err)
	}
	entradas := inventariarFronteraNominal(paquete)
	esperadas := []string{
		"alias o reexportación nominal Escalar",
		"TypeName nominal Nominal",
		"interfaz embebida",
		"campo embebido Nominal",
		"campo transaccion",
		"constante marca",
		"variable permitida",
		"función usar",
		"tipo struct anónimo",
		"tipo interface anónimo",
		"tipo función anónimo",
		"tipo array o slice anónimo",
		"tipo map anónimo",
		"tipo canal anónimo",
	}
	for _, esperada := range esperadas {
		encontrada := false
		for _, entrada := range entradas {
			if strings.Contains(entrada.origen, esperada) {
				encontrada = true
				break
			}
		}
		if !encontrada {
			t.Errorf(
				"categoría go/types %q ausente del inventario",
				esperada,
			)
		}
	}
}

func analizarBypassEnPropietarioPrueba(
	fuente string,
) (
	paqueteAnalizadoSesionTCB,
	error,
) {
	conjunto := token.NewFileSet()
	archivo, err := parser.ParseFile(
		conjunto,
		"bypass_en_cobertura.go",
		fuente,
		0,
	)
	if err != nil {
		return paqueteAnalizadoSesionTCB{}, err
	}
	info := &types.Info{
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Types:      make(map[ast.Expr]types.TypeAndValue),
	}
	paquete, err := (&types.Config{}).Check(
		rutaPaqueteCoberturaSesionTCB,
		conjunto,
		[]*ast.File{archivo},
		info,
	)
	if err != nil {
		return paqueteAnalizadoSesionTCB{}, err
	}
	return paqueteAnalizadoSesionTCB{
		metadatos: paqueteProductivoSesionTCB{
			ImportPath: rutaPaqueteCoberturaSesionTCB,
		},
		tipos:    paquete,
		info:     info,
		ficheros: []*ast.File{archivo},
	}, nil
}

func TestArquitecturaInventarioNominalDetectaBypassEnPropietario(
	t *testing.T,
) {
	const base = `package cobertura

type TransaccionOperacionDecisionCobertura interface {
	confirmar()
}

type transaccionOperacionDecisionCoberturaTCB struct{}

func (*transaccionOperacionDecisionCoberturaTCB) confirmar() {}
`
	casos := []struct {
		nombre         string
		bypass         string
		origenEsperado string
	}{
		{
			nombre: "alias",
			bypass: `
type AccesoOperacion = TransaccionOperacionDecisionCobertura
`,
			origenEsperado: "alias o reexportación nominal AccesoOperacion",
		},
		{
			nombre: "alias compuesto sin TCB",
			bypass: `
type Transporte = []TransaccionOperacionDecisionCobertura
`,
			origenEsperado: "alias o reexportación nominal Transporte",
		},
		{
			nombre: "interfaz embebida",
			bypass: `
type AccesoOperacion interface {
	TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "interfaz embebida",
		},
		{
			nombre: "variable con interfaz anónima",
			bypass: `
var acceso interface {
	TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "variable acceso",
		},
		{
			nombre: "función con interfaz anónima",
			bypass: `
func transportar(
	acceso interface {
		TransaccionOperacionDecisionCobertura
	},
) {
	_ = acceso
}
`,
			origenEsperado: "función transportar",
		},
		{
			nombre: "campo con interfaz anónima",
			bypass: `
type Servicio struct {
	acceso interface {
		TransaccionOperacionDecisionCobertura
	}
}
`,
			origenEsperado: "campo acceso",
		},
		{
			nombre: "struct anónimo",
			bypass: `
var acceso struct {
	TransaccionOperacionDecisionCobertura
}
`,
			origenEsperado: "tipo struct anónimo",
		},
		{
			nombre: "alias local",
			bypass: `
func ocultar() {
	type Pasarela = TransaccionOperacionDecisionCobertura
	var _ Pasarela
}
`,
			origenEsperado: "alias o reexportación nominal Pasarela",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			paquete, err := analizarBypassEnPropietarioPrueba(
				base + caso.bypass,
			)
			if err != nil {
				t.Fatal(err)
			}
			hallazgos, err := hallarIncorporacionesFronteraNominal(
				paquete,
				paquete.tipos,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(hallazgos) == 0 {
				t.Fatal("el bypass interno no fue detectado")
			}
			if !hallazgosContienenOrigen(
				hallazgos,
				caso.origenEsperado,
			) {
				t.Fatalf(
					"no se detectó por %q; hallazgos: %v",
					caso.origenEsperado,
					hallazgos,
				)
			}
		})
	}

	paquete, err := analizarBypassEnPropietarioPrueba(base)
	if err != nil {
		t.Fatal(err)
	}
	hallazgos, err := hallarIncorporacionesFronteraNominal(
		paquete,
		paquete.tipos,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(hallazgos) != 0 {
		t.Fatalf("la frontera y su implementador legítimo fueron rechazados")
	}
}
