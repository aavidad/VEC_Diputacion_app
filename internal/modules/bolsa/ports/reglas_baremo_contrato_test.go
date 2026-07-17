package ports

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestContratosReglasBaremoMantienenFronteraHexagonalFina(t *testing.T) {
	archivos, err := filepath.Glob("reglas_baremo_*.go")
	if err != nil {
		t.Fatal(err)
	}
	prohibidas := map[string]struct{}{
		"bytes": {}, "crypto/sha256": {}, "encoding/hex": {}, "encoding/json": {},
		"fmt": {}, "io": {}, "log/slog": {}, "reflect": {}, "sort": {}, "strings": {},
	}
	for _, archivo := range archivos {
		if strings.HasSuffix(archivo, "_test.go") {
			continue
		}
		arbol, errParseo := parser.ParseFile(token.NewFileSet(), archivo, nil, 0)
		if errParseo != nil {
			t.Fatalf("%s: %v", archivo, errParseo)
		}
		for _, importacion := range arbol.Imports {
			ruta, errRuta := strconv.Unquote(importacion.Path.Value)
			if errRuta != nil {
				t.Fatalf("%s: %v", archivo, errRuta)
			}
			if _, prohibida := prohibidas[ruta]; prohibida || strings.HasPrefix(ruta, "crypto/") || strings.HasPrefix(ruta, "encoding/") {
				t.Errorf("%s importa procesamiento prohibido en ports: %s", archivo, ruta)
			}
		}
		for _, declaracion := range arbol.Decls {
			if funcion, encontrada := declaracion.(*ast.FuncDecl); encontrada {
				t.Errorf("%s contiene logica %s; ports solo admite tipos e interfaces", archivo, funcion.Name.Name)
			}
		}
	}
}

func TestPuertosReglasBaremoSonPequenosExactosYUsanAutorizacionCentral(t *testing.T) {
	interfaces := []reflect.Type{
		reflect.TypeOf((*RepositorioGobiernoReglasBaremo)(nil)).Elem(),
		reflect.TypeOf((*ConsultaAutorizadaReglasBaremo)(nil)).Elem(),
		reflect.TypeOf((*FuenteReglasBaremoParaCalculo)(nil)).Elem(),
	}
	for _, contrato := range interfaces {
		if contrato.NumMethod() != 1 {
			t.Fatalf("%s tiene %d operaciones; se esperaba una frontera segregada", contrato, contrato.NumMethod())
		}
		metodo := contrato.Method(0)
		minusculas := strings.ToLower(metodo.Name)
		if strings.Contains(minusculas, "ultima") || strings.Contains(minusculas, "vigente") || strings.Contains(minusculas, "latest") {
			t.Fatalf("%s expone un selector temporal ambiguo", metodo.Name)
		}
		if metodo.Type.NumIn() == 0 || metodo.Type.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
			t.Fatalf("%s no recibe context.Context en primer lugar", metodo.Name)
		}
	}

	tipoAutorizacion := reflect.TypeOf(puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{})
	for _, solicitud := range []reflect.Type{
		reflect.TypeOf(OrdenConfirmacionReglasBaremo{}),
		reflect.TypeOf(SolicitudConsultaExactaReglasBaremo{}),
		reflect.TypeOf(SolicitudFuenteExactaCalculoReglasBaremo{}),
	} {
		campo, existe := solicitud.FieldByName("Autorizacion")
		if !existe || campo.Type != tipoAutorizacion {
			t.Fatalf("%s no reutiliza la autorizacion V2 central", solicitud)
		}
	}
}

func TestPruebaFuenteReglasBaremoEsCompacta(t *testing.T) {
	tipo := reflect.TypeOf(PruebaFuenteExactaCalculoReglasBaremo{})
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		if campo.Type.Kind() == reflect.Slice || campo.Type.Kind() == reflect.Map || campo.Type.Kind() == reflect.Array {
			t.Fatalf("la prueba compacta contiene la coleccion %s", campo.Name)
		}
	}
}

func TestFuenteCalculoExigeReciboTipadoDeConsumoAutorizacion(t *testing.T) {
	tipo := reflect.TypeOf(FuenteExactaCalculoReglasBaremo{})
	campo, existe := tipo.FieldByName("ConsumoAutorizacion")
	if !existe || campo.Type != reflect.TypeOf(oficial.ReciboConsumoAutorizacionFuenteV1{}) {
		t.Fatal("la fuente admite una referencia debil en lugar del recibo durable tipado")
	}
}
