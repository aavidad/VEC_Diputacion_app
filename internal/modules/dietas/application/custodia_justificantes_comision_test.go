package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

type registroExternoPrueba struct {
	solicitud docports.AltaExterna
	llamadas  int
}

var errRegistroExternoPrueba = errors.New("registro detenido por prueba")

func (r *registroExternoPrueba) RegistrarExterno(_ context.Context, a docports.AltaExterna) (docdomain.Documento, error) {
	r.llamadas++
	r.solicitud = a
	return docdomain.Documento{}, errRegistroExternoPrueba
}

func borradorConJustificantesPrueba(t *testing.T) (OrdenJustificanteComision, dietasports.ResultadoBorradorComision) {
	t.Helper()
	orden, resultado := ordenDocumentoPrueba(t)
	ref, huella := "justificante:taxi:0001", strings.Repeat("d", 64)
	resultado.Comision.Documento = &domain.DocumentoComision{Lineas: []domain.LineaDocumentoComision{
		{Tipo: "manutencion", ImporteCentimos: 1000},
		{Tipo: "otro", Concepto: "Taxi", ImporteCentimos: 1500, JustificanteRef: &ref, JustificanteSHA256: &huella},
		{Tipo: "otro", Concepto: "Sin justificante", ImporteCentimos: 200},
	}}
	return OrdenJustificanteComision{
		ReferenciaComision: orden.ReferenciaComision, VersionEsperada: orden.VersionEsperada, IndiceLinea: 1,
		DocumentoID: "ref:" + strings.Repeat("8", 64), ClaveIdempotencia: "ref:" + strings.Repeat("9", 64),
		TipoDocumentalRef: orden.TipoDocumentalRef, SolicitudPolitica: orden.SolicitudPolitica,
		Autorizacion: docports.AutorizacionV3{Accion: docports.AccionRegistrarExterno},
	}, resultado
}

func TestJustificantesComisionSoloEnumeraLineasConReferenciaYHuella(t *testing.T) {
	_, resultado := borradorConJustificantesPrueba(t)
	j := JustificantesComision(resultado)
	if len(j) != 1 || j[0].IndiceLinea != 1 || j[0].Custodia.CustodioID != CustodioJustificantesDietas ||
		j[0].Custodia.Referencia != "justificante:taxi:0001" || j[0].Custodia.Validar() != nil {
		t.Fatalf("justificantes: %+v", j)
	}
	resultado.Comision.Documento = nil
	if JustificantesComision(resultado) != nil {
		t.Fatal("borrador sin documento con justificantes")
	}
}

func TestJustificanteComisionLlegaAlPuertoComunSinContenido(t *testing.T) {
	orden, resultado := borradorConJustificantesPrueba(t)
	registro := &registroExternoPrueba{}
	servicio, err := NuevoServicioJustificantesComision(&lectorDocumentoPrueba{resultado: resultado}, registro)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Registrar(context.Background(), orden); !errors.Is(err, errRegistroExternoPrueba) || registro.llamadas != 1 {
		t.Fatalf("camino al puerto comun: %v llamadas=%d", err, registro.llamadas)
	}
	agrupacion, _ := ReferenciaAgrupacionDocumentoComision(orden.ReferenciaComision, orden.VersionEsperada)
	s := registro.solicitud
	if s.ExpedienteRef != agrupacion || s.ModuloID != "dietas" || s.Version != VersionJustificanteDietas ||
		s.Custodia.Referencia != "justificante:taxi:0001" || s.Custodia.HuellaSHA256 != strings.Repeat("d", 64) ||
		s.MIME != "" || s.Tamano != 0 {
		t.Fatalf("alta externa discordante: %+v", s)
	}
}

func TestJustificanteComisionDeniegaLineaSinCustodiaOVersionAjena(t *testing.T) {
	casos := map[string]func(*OrdenJustificanteComision, *dietasports.ResultadoBorradorComision){
		"linea sin justificante": func(o *OrdenJustificanteComision, _ *dietasports.ResultadoBorradorComision) { o.IndiceLinea = 2 },
		"linea inexistente":      func(o *OrdenJustificanteComision, _ *dietasports.ResultadoBorradorComision) { o.IndiceLinea = 9 },
		"version superada": func(_ *OrdenJustificanteComision, r *dietasports.ResultadoBorradorComision) {
			r.Recibo.Version = 3
		},
		"huella nula": func(_ *OrdenJustificanteComision, r *dietasports.ResultadoBorradorComision) {
			cero := strings.Repeat("0", 64)
			r.Comision.Documento.Lineas[1].JustificanteSHA256 = &cero
		},
	}
	for nombre, alterar := range casos {
		orden, resultado := borradorConJustificantesPrueba(t)
		alterar(&orden, &resultado)
		registro := &registroExternoPrueba{}
		servicio, _ := NuevoServicioJustificantesComision(&lectorDocumentoPrueba{resultado: resultado}, registro)
		if _, err := servicio.Registrar(context.Background(), orden); !errors.Is(err, dietasports.ErrInstantaneaDocumentoComisionInvalida) || registro.llamadas != 0 {
			t.Errorf("%s: err=%v llamadas=%d", nombre, err, registro.llamadas)
		}
	}
	orden, resultado := borradorConJustificantesPrueba(t)
	orden.Autorizacion.Accion = docports.AccionAlta
	registro := &registroExternoPrueba{}
	servicio, _ := NuevoServicioJustificantesComision(&lectorDocumentoPrueba{resultado: resultado}, registro)
	if _, err := servicio.Registrar(context.Background(), orden); !errors.Is(err, dietasports.ErrDocumentoComisionNoDisponible) || registro.llamadas != 0 {
		t.Fatalf("accion de alta aceptada: %v", err)
	}
}
