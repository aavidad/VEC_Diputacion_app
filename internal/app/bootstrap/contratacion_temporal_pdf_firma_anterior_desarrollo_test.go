package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type descargaFirmadoPrueba struct {
	original  docports.Original
	documento docdomain.Documento
	err       error
	consultas []docports.ConsultaDocumento
}

func (d *descargaFirmadoPrueba) DescargarOriginalConDocumento(_ context.Context, c docports.ConsultaDocumento) (docports.Original, docdomain.Documento, error) {
	d.consultas = append(d.consultas, c)
	if d.err != nil {
		return docports.Original{}, docdomain.Documento{}, d.err
	}
	o := d.original
	o.Contenido = bytes.Clone(o.Contenido)
	return o, d.documento, nil
}

func escenarioPDFAnteriorPrueba(t *testing.T) (*fuentePDFFirmaAnteriorCTDesarrollo, *pdpOriginalPrueba, *descargaFirmadoPrueba, ports.SolicitudPDFFirmaAnterior) {
	t.Helper()
	pdp := &pdpOriginalPrueba{actor: dominiovec.DatosVinculoAutenticacionActorV2{PrincipalID: "per_prueba", PerfilActivoRef: "perfil:firma"}}
	o := nuevoOriginalPrueba(t, pdp)
	tipo, err := o.politicas.TipoDocumentalRef("contratacion_temporal.resolucion_firmada.v1")
	if err != nil || !o.politicas.CustodiaFirmadoReservada(tipo) {
		t.Fatalf("tipo de firmado: %v", err)
	}
	contenido := []byte("%PDF-1.7\nfirmado paso 1\n%%EOF")
	huella := sha256HexPrueba(contenido)
	q := ports.SolicitudPDFFirmaAnterior{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:ct:original:001",
		Documento: "resolucion", FirmaRef: "firma:ct:paso1", ReciboRef: "recibo:ct:paso1",
		DocumentoRef: "ref:" + strings.Repeat("d", 64), DocumentoVersion: 1, DocumentoHuella: huella}
	descarga := &descargaFirmadoPrueba{original: docports.Original{Contenido: contenido, MIME: "application/pdf", HuellaSHA256: huella},
		documento: docdomain.Documento{ID: q.DocumentoRef, Version: 1, ModuloID: moduloProductorCustodiaCT,
			ExpedienteRef: ports.ExpedienteDocumentalRef(q.OrganizacionRef, q.ExpedienteRef), TipoRef: tipo,
			MIME: "application/pdf", HuellaSHA256: huella}}
	return &fuentePDFFirmaAnteriorCTDesarrollo{original: o, servicio: descarga}, pdp, descarga, q
}

// La fuente lee con una descarga V3 del documento, la versión y el expediente
// documental exactos y solo devuelve el PDF firmado que CT custodió.
func TestPDFFirmaAnteriorLeeElFirmadoCustodiado(t *testing.T) {
	f, pdp, descarga, q := escenarioPDFAnteriorPrueba(t)
	pdf, err := f.ObtenerPDFFirmaAnterior(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if pdf.Solicitud != q || !bytes.Equal(pdf.Contenido, descarga.original.Contenido) {
		t.Fatal("PDF inesperado")
	}
	c := descarga.consultas[0]
	if c.DocumentoID != q.DocumentoRef || c.Version != 1 || c.Autorizacion.Accion != docports.AccionDescargar ||
		c.Autorizacion.AmbitoRef != ports.ExpedienteDocumentalRef(q.OrganizacionRef, q.ExpedienteRef) || len(pdp.pedidas) != 1 {
		t.Fatalf("consulta inesperada: %+v", c)
	}
}

func TestPDFFirmaAnteriorRechazaOtroDocumentoOAutorizacion(t *testing.T) {
	f, pdp, descarga, q := escenarioPDFAnteriorPrueba(t)
	otra := q
	otra.OrganizacionRef = "organizacion:ajena"
	if _, err := f.ObtenerPDFFirmaAnterior(context.Background(), otra); !errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida) || len(pdp.pedidas) != 0 {
		t.Fatalf("otra organización: %v", err)
	}
	original := descarga.documento
	tipoOriginal, err := f.original.politicas.TipoDocumentalRef("contratacion_temporal.borrador.resolucion.v1")
	if err != nil {
		t.Fatal(err)
	}
	for nombre, alterar := range map[string]func(*docdomain.Documento){
		"otro módulo":     func(d *docdomain.Documento) { d.ModuloID = "dietas" },
		"otro expediente": func(d *docdomain.Documento) { d.ExpedienteRef = "ref:" + strings.Repeat("e", 64) },
		"otra versión":    func(d *docdomain.Documento) { d.Version = 2 },
		"tipo no firmado": func(d *docdomain.Documento) { d.TipoRef = tipoOriginal },
		"otra huella":     func(d *docdomain.Documento) { d.HuellaSHA256 = strings.Repeat("2", 64) },
		"otro documento":  func(d *docdomain.Documento) { d.ID = "ref:" + strings.Repeat("a", 64) },
	} {
		descarga.documento = original
		alterar(&descarga.documento)
		if _, err := f.ObtenerPDFFirmaAnterior(context.Background(), q); !errors.Is(err, ports.ErrAntecedenteFirmaR5NoAcreditado) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	descarga.documento = original
	pdp.err = errOriginalFirmableCTDenegado
	if _, err := f.ObtenerPDFFirmaAnterior(context.Background(), q); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("denegación del PDP: %v", err)
	}
	pdp.err = nil
	descarga.err = docports.ErrCapacidadNoDisponible
	if _, err := f.ObtenerPDFFirmaAnterior(context.Background(), q); !errors.Is(err, docports.ErrCapacidadNoDisponible) {
		t.Fatalf("Documentos caído: %v", err)
	}
	if _, err := nuevaFuentePDFFirmaAnteriorCTDesarrollo(f.original, nil); !errors.Is(err, errPDFFirmaAnteriorCTNoDisponible) {
		t.Fatalf("sin servicio de Documentos: %v", err)
	}
	if _, err := (&fuentePDFFirmaAnteriorCTDesarrollo{}).ObtenerPDFFirmaAnterior(context.Background(), q); !errors.Is(err, errPDFFirmaAnteriorCTNoDisponible) {
		t.Fatalf("sin composición: %v", err)
	}
}
