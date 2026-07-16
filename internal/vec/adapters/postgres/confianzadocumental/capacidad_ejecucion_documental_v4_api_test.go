package confianzadocumental_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"

	confianza "vec-diputacion-granada/internal/vec/adapters/postgres/confianzadocumental"
)

// Protege el cierre V4: un llamador ordinario no puede construir
// material HMAC, un paquete de artefactos ni sustituir el verificador por una
// interfaz propia. Solo ve el ensamblado concreto por socket Unix y el
// resultado redactado.
func TestAPICapacidadDocumentalV4NoExponePruebaDTOOEmisorInyectable(t *testing.T) {
	tipoEjecutor := reflect.TypeOf(confianza.EjecutorDocumentalPostgreSQLV4{})
	for indice := 0; indice < tipoEjecutor.NumField(); indice++ {
		campo := tipoEjecutor.Field(indice)
		if campo.PkgPath == "" {
			t.Fatalf("el ejecutor expone el campo fabricable %s", campo.Name)
		}
		if campo.Type.Kind() == reflect.Interface {
			t.Fatalf("el ejecutor admite sustituir %s mediante interfaz", campo.Name)
		}
	}
	if _, err := json.Marshal(&confianza.EjecutorDocumentalPostgreSQLV4{}); err == nil {
		t.Fatal("el valor cero del ejecutor pudo serializarse como credencial")
	}

	paquete := analizarPaqueteConfianzaProduccionV4(t)
	tiposProhibidos := []string{
		"CapacidadEjecucionDocumentalV4",
		"ArtefactosEjecucionDocumentalV4",
		"PaqueteEjecucionDocumentalV4",
		"MaterialEmisorCapacidadDocumentalV4",
		"ClienteEmisorCapacidadUnixV4",
		"RepositorioPostgreSQLEjecucionDocumentalV4",
		"EmisorCapacidadDocumentalV4",
		"VerificadorCapacidadDocumentalV4",
	}
	for nombreFichero, fichero := range paquete.Files {
		for _, declaracion := range fichero.Decls {
			switch declaracion := declaracion.(type) {
			case *ast.GenDecl:
				for _, especificacion := range declaracion.Specs {
					tipo, ok := especificacion.(*ast.TypeSpec)
					if !ok || !ast.IsExported(tipo.Name.Name) {
						continue
					}
					for _, prohibido := range tiposProhibidos {
						if strings.Contains(tipo.Name.Name, prohibido) {
							t.Fatalf("tipo de capacidad exportado %s en %s", tipo.Name.Name, nombreFichero)
						}
					}
				}
			case *ast.FuncDecl:
				if !ast.IsExported(declaracion.Name.Name) {
					continue
				}
				for _, prohibido := range []string{
					"capacidadEjecucionDocumentalV4JSON",
					"artefactosEjecucionDocumentalV4",
					"paqueteEjecucionDocumentalV4JSON",
					"materialEmisorCapacidadDocumentalV4",
					"clienteEmisorCapacidadUnixV4",
					"repositorioPostgreSQLEjecucionDocumentalV4",
				} {
					if contieneIdentificadorAST(declaracion.Type, prohibido) {
						t.Fatalf("funcion exportada %s expone %s en %s",
							declaracion.Name.Name, prohibido, nombreFichero)
					}
				}
			}
		}
	}
}

func TestAPICapacidadDocumentalV4MantieneFronterasConcretasMinimas(t *testing.T) {
	encontrados := map[string]bool{
		"NuevoManejadorHTTPEmisorCapacidadDocumentalV4": false,
		"NuevoEjecutorDocumentalPostgreSQLV4":           false,
		"EjecutarDocumentalAtestadoV4":                  false,
	}
	visitarFuncionesExportadasProduccion(t, func(funcion *ast.FuncDecl, fichero string) {
		switch funcion.Name.Name {
		case "NuevoManejadorHTTPEmisorCapacidadDocumentalV4":
			encontrados[funcion.Name.Name] = true
			if funcion.Recv != nil || !contieneIdentificadorAST(funcion.Type.Params, "Context") ||
				!contieneIdentificadorAST(funcion.Type.Params, "Pool") ||
				!contieneIdentificadorAST(funcion.Type.Results, "Handler") ||
				contieneIdentificadorAST(funcion.Type, "EjecutorDocumentalPostgreSQLV4") {
				t.Fatalf("frontera emisora ampliada en %s", fichero)
			}
		case "NuevoEjecutorDocumentalPostgreSQLV4":
			encontrados[funcion.Name.Name] = true
			if funcion.Recv != nil || !contieneIdentificadorAST(funcion.Type.Params, "Context") ||
				!contieneIdentificadorAST(funcion.Type.Params, "Pool") ||
				!contieneIdentificadorAST(funcion.Type.Params, "string") ||
				!contieneIdentificadorAST(funcion.Type.Results, "EjecutorDocumentalPostgreSQLV4") ||
				contieneIdentificadorAST(funcion.Type.Params, "Handler") ||
				contieneIdentificadorAST(funcion.Type.Params, "Servicio") ||
				contieneIdentificadorAST(funcion.Type.Params, "ConfiguracionConfianzaFijada") {
				t.Fatalf("frontera ejecutora ampliada en %s", fichero)
			}
		case "EjecutarDocumentalAtestadoV4":
			if !contieneIdentificadorAST(funcion.Recv, "EjecutorDocumentalPostgreSQLV4") {
				return
			}
			encontrados[funcion.Name.Name] = true
			for _, obligatorio := range []string{
				"Context", "SolicitudVinculadaAutorizacionEjecucionDocumentalV4",
				"CabeceraAtestacionAutorizacionV1", "SobreCriptograficoDocumentalCrudoV4",
			} {
				if !contieneIdentificadorAST(funcion.Type.Params, obligatorio) {
					t.Fatalf("Ejecutar carece de %s en %s", obligatorio, fichero)
				}
			}
			for _, prohibido := range []string{
				"Time", "Handler", "Pool", "Servicio", "DecisionAutorizacion",
				"PruebaCOSESign1DocumentalVerificada", "ResultadoEjecucionPlanDocumentalV4",
			} {
				if contieneIdentificadorAST(funcion.Type.Params, prohibido) {
					t.Fatalf("el puerto admite %s aportado por el llamador en %s", prohibido, fichero)
				}
			}
			if !contieneIdentificadorAST(
				funcion.Type.Results,
				"ResultadoConectorEjecucionDocumentalAtestadaV4",
			) {
				t.Fatalf("el adaptador no devuelve el resultado neutral del puerto en %s", fichero)
			}
		}
	})
	for nombre, encontrado := range encontrados {
		if !encontrado {
			t.Fatalf("falta la frontera sellada %s", nombre)
		}
	}
}

func analizarPaqueteConfianzaProduccionV4(t *testing.T) *ast.Package {
	t.Helper()
	directorio := directorioFuentesConfianzaPrueba(t)
	paquetes, err := parser.ParseDir(
		token.NewFileSet(), directorio,
		func(info os.FileInfo) bool {
			return strings.HasSuffix(info.Name(), ".go") &&
				!strings.HasSuffix(info.Name(), "_test.go")
		},
		0,
	)
	if err != nil {
		t.Fatalf("analizar paquete de confianza: %v", err)
	}
	paquete, existe := paquetes["confianzadocumental"]
	if !existe {
		t.Fatal("no se encontro el paquete productivo de confianza")
	}
	return paquete
}
