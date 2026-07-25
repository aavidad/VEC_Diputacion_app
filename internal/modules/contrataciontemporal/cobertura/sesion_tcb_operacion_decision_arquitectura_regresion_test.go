package cobertura

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
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
	casos := map[string]string{
		"alias con nombre sin TCB": cabecera + `
type AccesoOperacion = cobertura.TransaccionOperacionDecisionCobertura
`,
		"interfaz definida embebida": cabecera + `
type AccesoOperacion interface {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
		"alias de interfaz embebida": cabecera + `
type AccesoOperacion = interface {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
		"interfaz anónima en valor": cabecera + `
var acceso interface {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
		"struct nombrado con embedding": cabecera + `
type AccesoOperacion struct {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
		"valor de struct anónimo": cabecera + `
var acceso struct {
	cobertura.TransaccionOperacionDecisionCobertura
}
`,
		"literal local de struct anónimo": cabecera + `
func ocultar() {
	_ = struct {
		cobertura.TransaccionOperacionDecisionCobertura
	}{}
}
`,
	}
	for nombre, fuente := range casos {
		t.Run(nombre, func(t *testing.T) {
			paquete, _, err := analizarBypassFronteraNominalPrueba(
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
			if len(hallazgos) == 0 {
				t.Fatal("el bypass no fue detectado")
			}
			for _, hallazgo := range hallazgos {
				if hallazgo.tipo == "invalid type" {
					t.Fatal("el bypass solo produjo ruido de go/types")
				}
			}
		})
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
	casos := map[string]string{
		"alias": `
type AccesoOperacion = TransaccionOperacionDecisionCobertura
`,
		"interfaz embebida": `
type AccesoOperacion interface {
	TransaccionOperacionDecisionCobertura
}
`,
		"struct anónimo": `
var acceso struct {
	TransaccionOperacionDecisionCobertura
}
`,
	}
	for nombre, bypass := range casos {
		t.Run(nombre, func(t *testing.T) {
			paquete, err := analizarBypassEnPropietarioPrueba(
				base + bypass,
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
