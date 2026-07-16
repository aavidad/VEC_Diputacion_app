package confianzadocumental_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Esta prueba externa protege la superficie compilable: ningun paquete de
// application debe encontrar una fabrica libre de AutoridadInterna. La unica
// via publica admisible es el metodo de Servicio que recibe el vinculo opaco,
// la cabecera fijada y el sobre crudo, y verifica COSE por si mismo. No admite
// una prueba o una decision DTO aportadas por el llamador como autoridad.
func TestAPINoPublicaFabricaLibreDeAutoridadInternaEjecucionDocumentalV4(t *testing.T) {
	visitarFuncionesExportadasProduccion(t, func(funcion *ast.FuncDecl, fichero string) {
		if funcion.Type.Results == nil ||
			!contieneIdentificadorAST(funcion.Type.Results, "AutoridadInternaEjecucionDocumentalV4") {
			return
		}
		if funcion.Recv == nil {
			t.Fatalf("fabrica libre exportada prohibida: %s en %s", funcion.Name.Name, fichero)
		}
		if !contieneIdentificadorAST(funcion.Recv, "Servicio") ||
			!contieneIdentificadorAST(funcion.Type.Params, "Context") ||
			!contieneIdentificadorAST(
				funcion.Type.Params,
				"SolicitudVinculadaAutorizacionEjecucionDocumentalV4",
			) ||
			!contieneIdentificadorAST(funcion.Type.Params, "CabeceraAtestacionAutorizacionV1") ||
			!contieneIdentificadorAST(funcion.Type.Params, "SobreCriptograficoDocumentalCrudoV4") ||
			contieneIdentificadorAST(funcion.Type.Params, "PruebaCOSESign1DocumentalVerificada") ||
			contieneIdentificadorAST(funcion.Type.Params, "DecisionAutorizacion") ||
			contieneIdentificadorAST(funcion.Type.Params, "EvidenciaUsoDecisionAutorizacion") ||
			contieneIdentificadorAST(funcion.Type.Params, "Time") {
			t.Fatalf(
				"firma publica de emision de autoridad prohibida: %s en %s",
				funcion.Name.Name,
				fichero,
			)
		}
	})
}

// La evidencia contiene payload y sobre completos. Solo una autoridad interna
// ya emitida puede entregarla junto con la solicitud exacta de aplicacion; no
// existe un constructor libre ni un metodo de Servicio que la fabrique sola.
func TestAPINoPublicaFabricaLibreDeEvidenciaDurableAtestacionPDPV4(t *testing.T) {
	visitarFuncionesExportadasProduccion(t, func(funcion *ast.FuncDecl, fichero string) {
		if funcion.Type.Results == nil || !contieneIdentificadorAST(
			funcion.Type.Results,
			"EvidenciaDurableAtestacionAutorizacionPDPV4",
		) {
			return
		}
		if funcion.Recv == nil ||
			!contieneIdentificadorAST(funcion.Recv, "AutoridadInternaEjecucionDocumentalV4") ||
			funcion.Name.Name != "PrepararAplicacionExactaConEvidenciaEn" ||
			contieneIdentificadorAST(
				funcion.Type.Params,
				"EvidenciaDurableAtestacionAutorizacionPDPV4",
			) {
			t.Fatalf(
				"fabrica publica de evidencia durable prohibida: %s en %s",
				funcion.Name.Name,
				fichero,
			)
		}
	})
}

// La unica operacion durable publica recibe la autoridad COSE opaca y usa el
// repositorio privado fijado en Servicio. El llamador no aporta repositorio,
// pool, DTO, prueba persistible ni instante.
func TestAPIEjecucionAtestadaSoloOcurreDentroDeServicio(t *testing.T) {
	encontrado := false
	visitarFuncionesExportadasProduccion(t, func(funcion *ast.FuncDecl, fichero string) {
		for _, prohibido := range []string{
			"SolicitudRegistroAtestacionPDPDocumentalV4",
			"DatosRegistroAtestacionPDPDocumentalV4",
			"EmisorSolicitudRegistroAtestacionPDPDocumentalV4",
			"VerificadorSolicitudRegistroAtestacionPDPDocumentalV4",
		} {
			if contieneIdentificadorAST(funcion.Type, prohibido) {
				t.Fatalf("API durable expone %s: %s en %s", prohibido, funcion.Name.Name, fichero)
			}
		}
		if funcion.Name.Name == "RegistrarAtestacionPDPDocumentalV4" {
			t.Fatalf("reaparecio el registro separado no atomico en %s", fichero)
		}
		if funcion.Name.Name != "EjecutarPlanDocumentalV4" {
			return
		}
		encontrado = true
		parametros := funcion.Type.Params
		if funcion.Recv == nil || !contieneIdentificadorAST(funcion.Recv, "Servicio") ||
			parametros == nil || !contieneIdentificadorAST(parametros, "Context") ||
			!contieneIdentificadorAST(parametros, "AutoridadInternaEjecucionDocumentalV4") ||
			contieneIdentificadorAST(parametros, "Time") ||
			contieneIdentificadorAST(parametros, "Pool") ||
			contieneIdentificadorAST(parametros, "Repositorio") ||
			contieneIdentificadorAST(parametros, "SolicitudRegistroAtestacionPDPDocumentalV4") ||
			contieneIdentificadorAST(parametros, "DatosRegistroAtestacionPDPDocumentalV4") {
			t.Fatalf(
				"firma publica de registro prohibida: %s en %s",
				funcion.Name.Name,
				fichero,
			)
		}
	})
	if !encontrado {
		t.Fatal("falta el caso de uso publico de ejecucion atestada dentro de Servicio")
	}
}

func visitarFuncionesExportadasProduccion(
	t *testing.T,
	visitar func(*ast.FuncDecl, string),
) {
	t.Helper()
	directorio := directorioFuentesConfianzaPrueba(t)
	entradas, err := os.ReadDir(directorio)
	if err != nil {
		t.Fatalf("leer paquete de confianza: %v", err)
	}

	conjunto := token.NewFileSet()
	for _, entrada := range entradas {
		if entrada.IsDir() || !strings.HasSuffix(entrada.Name(), ".go") ||
			strings.HasSuffix(entrada.Name(), "_test.go") {
			continue
		}
		ruta := filepath.Join(directorio, entrada.Name())
		fichero, err := parser.ParseFile(conjunto, ruta, nil, 0)
		if err != nil {
			t.Fatalf("analizar %s: %v", entrada.Name(), err)
		}
		for _, declaracion := range fichero.Decls {
			funcion, esFuncion := declaracion.(*ast.FuncDecl)
			if !esFuncion || !ast.IsExported(funcion.Name.Name) {
				continue
			}
			visitar(funcion, entrada.Name())
		}
	}
}

func contieneIdentificadorAST(nodo ast.Node, esperado string) bool {
	if nodo == nil {
		return false
	}
	encontrado := false
	ast.Inspect(nodo, func(actual ast.Node) bool {
		identificador, ok := actual.(*ast.Ident)
		if ok && identificador.Name == esperado {
			encontrado = true
			return false
		}
		return !encontrado
	})
	return encontrado
}
