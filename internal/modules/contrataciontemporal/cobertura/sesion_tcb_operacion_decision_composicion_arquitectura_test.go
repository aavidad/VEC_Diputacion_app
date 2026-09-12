package cobertura

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"testing"
)

const rutaComposicionPrivadaSesionTCB = "vec-diputacion-granada/internal/app/bootstrap"
const ficheroComposicionPrivadaSesionTCB = "contratacion_temporal_cobertura_composicion_desarrollo.go"
const funcionComposicionPrivadaSesionTCB = "nuevasDependenciasCoberturaContratacionTemporalDesarrollo"

// Los constructores reservados a composición se invocan aquí desde 28b14bba.
// La excepción identifica llamadas concretas; no permite reexportar el
// constructor ni habilita otros símbolos TCB en el paquete bootstrap.
func constructorEnComposicionPrivadaSesionTCB(
	analisis analisisArquitecturaSesionTCB,
	paquete paqueteAnalizadoSesionTCB,
	identificador *ast.Ident,
	objeto types.Object,
) bool {
	if paquete.metadatos.ImportPath != rutaComposicionPrivadaSesionTCB ||
		!objetoEsSimboloTCB(objeto, analisis.simbolosTCB) {
		return false
	}
	switch objeto.Name() {
	case "NuevaTransaccionOperacionDecisionCoberturaTCB",
		"NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB",
		"NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB":
	default:
		return false
	}
	rutaEsperada := filepath.Join(paquete.metadatos.Dir, ficheroComposicionPrivadaSesionTCB)
	for _, archivo := range paquete.ficheros {
		origen := analisis.conjunto.File(archivo.Pos())
		if origen == nil || filepath.Clean(origen.Name()) != rutaEsperada {
			continue
		}
		for _, declaracion := range archivo.Decls {
			funcion, ok := declaracion.(*ast.FuncDecl)
			if !ok || funcion.Recv != nil || funcion.Body == nil ||
				funcion.Name.Name != funcionComposicionPrivadaSesionTCB {
				continue
			}
			permitida := false
			ast.Inspect(funcion.Body, func(nodo ast.Node) bool {
				if _, cierre := nodo.(*ast.FuncLit); cierre {
					return false
				}
				llamada, ok := nodo.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := llamada.Fun.(*ast.SelectorExpr)
				if ok && selector.Sel == identificador {
					permitida = true
				}
				return true
			})
			if permitida {
				return true
			}
		}
	}
	return false
}

func TestArquitecturaComposicionPrivadaTCBExcepcionAcotada(t *testing.T) {
	analisisProductivo := obtenerAnalisisArquitecturaSesionTCB(t)
	const constructor = "NuevaTransaccionOperacionDecisionCoberturaTCB"
	funcion := func(nombre, cuerpo string) string {
		return "func " + nombre + "() { " + cuerpo + " }\n"
	}
	llamada := func(nombre string) string {
		return "valor, err := cobertura." + nombre + "(nil); _, _ = valor, err"
	}
	casos := []struct {
		nombre        string
		ruta          string
		fichero       string
		fuente        string
		permitir      bool
		suplantarRuta bool
	}{
		{
			nombre:   "constructor de transaccion en composicion",
			fuente:   funcion(funcionComposicionPrivadaSesionTCB, llamada(constructor)),
			permitir: true,
		},
		{
			nombre: "constructor de reconciliacion en composicion",
			fuente: funcion(funcionComposicionPrivadaSesionTCB,
				llamada("NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB")),
			permitir: true,
		},
		{
			nombre: "constructor de lectura historica en composicion",
			fuente: funcion(funcionComposicionPrivadaSesionTCB,
				llamada("NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB")),
			permitir: true,
		},
		{
			nombre: "otro paquete",
			ruta:   "vec-diputacion-granada/internal/app/puente",
			fuente: funcion(funcionComposicionPrivadaSesionTCB, llamada(constructor)),
		},
		{
			nombre: "canal HTTP",
			ruta:   "vec-diputacion-granada/internal/app/server",
			fuente: funcion(funcionComposicionPrivadaSesionTCB, llamada(constructor)),
		},
		{
			nombre:  "otro archivo de bootstrap",
			fichero: "otro.go",
			fuente:  funcion(funcionComposicionPrivadaSesionTCB, llamada(constructor)),
		},
		{
			nombre:        "directiva line no suplanta el archivo de composicion",
			fichero:       "otro.go",
			fuente:        funcion(funcionComposicionPrivadaSesionTCB, llamada(constructor)),
			suplantarRuta: true,
		},
		{
			nombre: "otra funcion privada",
			fuente: funcion("nuevasDependenciasAlternativas", llamada(constructor)),
		},
		{
			nombre: "funcion exportada",
			fuente: funcion("NuevasDependenciasCoberturaContratacionTemporalDesarrollo", llamada(constructor)),
		},
		{
			nombre: "metodo con el mismo nombre",
			fuente: "type montaje struct{}\nfunc (montaje) " + funcionComposicionPrivadaSesionTCB +
				"() { " + llamada(constructor) + " }\n",
		},
		{
			nombre: "alias de constructor dentro de la composicion",
			fuente: funcion(funcionComposicionPrivadaSesionTCB,
				"constructor := cobertura."+constructor+"; _, _ = constructor(nil)"),
		},
		{
			nombre: "reexportacion global del constructor",
			fuente: "var Constructor = cobertura." + constructor,
		},
		{
			nombre: "otro simbolo TCB en la misma funcion",
			fuente: funcion(funcionComposicionPrivadaSesionTCB,
				"var sesion cobertura.SesionTCBOperacionDecisionCobertura; _ = sesion"),
		},
		{
			nombre: "constructor como argumento de otra llamada",
			fuente: funcion(funcionComposicionPrivadaSesionTCB,
				"func(any) {}(cobertura."+constructor+")"),
		},
		{
			nombre: "constructor dentro de un cierre",
			fuente: funcion(funcionComposicionPrivadaSesionTCB,
				"func() { "+llamada(constructor)+" }()"),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ruta, fichero := caso.ruta, caso.fichero
			if ruta == "" {
				ruta = rutaComposicionPrivadaSesionTCB
			}
			if fichero == "" {
				fichero = ficheroComposicionPrivadaSesionTCB
			}
			directorio := filepath.Join(t.TempDir(), "bootstrap")
			conjunto := token.NewFileSet()
			fuente := "package bootstrap\nimport cobertura \"" +
				rutaPaqueteCoberturaSesionTCB + "\"\n" + caso.fuente
			if caso.suplantarRuta {
				fuente = "//line " + filepath.Join(directorio, ficheroComposicionPrivadaSesionTCB) + ":1\n" + fuente
			}
			archivo, err := parser.ParseFile(conjunto, filepath.Join(directorio, fichero), fuente, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
			tipos, err := (&types.Config{
				Importer: importadorFronteraNominalPrueba{cobertura: analisisProductivo.cobertura},
			}).Check(ruta, conjunto, []*ast.File{archivo}, info)
			if err != nil {
				t.Fatal(err)
			}
			paquete := paqueteAnalizadoSesionTCB{
				metadatos: paqueteProductivoSesionTCB{ImportPath: ruta, Dir: directorio},
				tipos:     tipos, info: info, ficheros: []*ast.File{archivo},
			}
			analisis := analisisProductivo
			analisis.conjunto = conjunto
			comprobados := 0
			for identificador, objeto := range info.Uses {
				if !objetoEsSimboloTCB(objeto, analisis.simbolosTCB) {
					continue
				}
				comprobados++
				if permitido := constructorEnComposicionPrivadaSesionTCB(
					analisis, paquete, identificador, objeto,
				); permitido != caso.permitir {
					t.Errorf("excepcion = %v; esperada = %v", permitido, caso.permitir)
				}
			}
			if comprobados != 1 {
				t.Fatalf("usos TCB comprobados = %d; esperado = 1", comprobados)
			}
		})
	}
}
