package application

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

type registroExternoPrueba struct {
	solicitud docports.AltaExterna
	llamadas  int
	respuesta func(docports.AltaExterna) docdomain.Documento
}

var errRegistroExternoPrueba = errors.New("registro detenido por prueba")

func (r *registroExternoPrueba) RegistrarExterno(_ context.Context, a docports.AltaExterna) (docdomain.Documento, error) {
	r.llamadas++
	r.solicitud = a
	if r.respuesta != nil {
		return r.respuesta(a), nil
	}
	return docdomain.Documento{}, errRegistroExternoPrueba
}

func documentoExternoConfirmadoPrueba(a docports.AltaExterna) docdomain.Documento {
	return docdomain.Documento{
		ID: a.ID, NumeroVEC: "VEC-2026-1", ModuloID: a.ModuloID,
		ExpedienteRef: a.ExpedienteRef, TipoRef: a.TipoRef, Version: a.Version,
		MIME: a.MIME, Tamano: a.Tamano, HuellaSHA256: a.Custodia.HuellaSHA256,
		PoliticaRef:          a.SolicitudPolitica.PoliticaRef(),
		VersionPolitica:      a.SolicitudPolitica.VersionPolitica(),
		HuellaPoliticaSHA256: hex.EncodeToString(a.SolicitudPolitica.HuellaPoliticaSHA256()),
		ConservacionHasta:    time.Date(2027, 9, 21, 0, 0, 0, 0, time.UTC),
		Proteccion:           "conservacion", EstadoPolitica: docdomain.EstadoPoliticaProvisional,
		EstadoFirma: docdomain.EstadoFirmaPendienteProveedor,
		CreadoEn:    time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC),
		Custodia:    docdomain.CustodiaExterna, CustodiaExternaRef: a.Custodia,
	}
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

func TestJustificanteComisionCompruebaReciboExternoExacto(t *testing.T) {
	casos := map[string]func(*docdomain.Documento){
		"valido":                   func(*docdomain.Documento) {},
		"otra version":             func(d *docdomain.Documento) { d.Version++ },
		"otro MIME":                func(d *docdomain.Documento) { d.MIME = "application/pdf" },
		"otro tamano":              func(d *docdomain.Documento) { d.Tamano = 1 },
		"otra politica":            func(d *docdomain.Documento) { d.PoliticaRef = "ref:" + strings.Repeat("f", 64) },
		"otra version de politica": func(d *docdomain.Documento) { d.VersionPolitica++ },
		"otra huella de politica":  func(d *docdomain.Documento) { d.HuellaPoliticaSHA256 = strings.Repeat("e", 64) },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			orden, resultado := borradorConJustificantesPrueba(t)
			registro := &registroExternoPrueba{respuesta: func(a docports.AltaExterna) docdomain.Documento {
				d := documentoExternoConfirmadoPrueba(a)
				alterar(&d)
				return d
			}}
			servicio, err := NuevoServicioJustificantesComision(&lectorDocumentoPrueba{resultado: resultado}, registro)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Registrar(context.Background(), orden)
			if nombre == "valido" {
				if err != nil {
					t.Fatalf("recibo valido: %v", err)
				}
			} else if !errors.Is(err, dietasports.ErrDocumentoComisionNoDisponible) {
				t.Fatalf("recibo discordante aceptado: %v", err)
			}
		})
	}
}
