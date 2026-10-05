package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// PDF firmado de un paso anterior (corte 4c-7), perfil de desarrollo.
//
// La firma múltiple R5 verifica cada paso contra el PDF que firmó el paso
// anterior, ya custodiado en Documentos (5.06). Esta fuente lo lee con una
// decisión V3 de documentos.original.descargar emitida por el PDP de CT para
// el documento, la versión y el expediente documental exactos, y la
// concesión de lectura del objeto ligada a esa descarga. Solo devuelve un PDF
// de un tipo reservado a firmados, producido por CT y con la huella que el
// registro de firmas acreditó.

var errPDFFirmaAnteriorCTNoDisponible = errors.New("bootstrap: PDF firmado anterior no disponible")

type fuentePDFFirmaAnteriorCTDesarrollo struct {
	original *originalFirmableCTDesarrollo
	servicio descargaDocumentoCTDesarrollo
}

// descargaDocumentoCTDesarrollo es la descarga de Documentos que usa la
// fuente; en la composición, servicioDocumentosOriginalCTDesarrollo.
type descargaDocumentoCTDesarrollo interface {
	DescargarOriginalConDocumento(context.Context, docports.ConsultaDocumento) (docports.Original, docdomain.Documento, error)
}

var _ ports.FuentePDFFirmaAnterior = (*fuentePDFFirmaAnteriorCTDesarrollo)(nil)

// nuevaFuentePDFFirmaAnteriorCTDesarrollo comparte PDP, catálogo y almacén
// con el original firmable; el servicio de Documentos debe tener como
// fábrica de lectura la del mismo emisor (lectura del objeto con el PDP de CT).
func nuevaFuentePDFFirmaAnteriorCTDesarrollo(o *originalFirmableCTDesarrollo, servicio *docapp.Servicio) (*fuentePDFFirmaAnteriorCTDesarrollo, error) {
	if o == nil || o.politicas == nil || dependenciaEsNulaContratacionTemporalDesarrollo(o.pdp) || servicio == nil {
		return nil, errPDFFirmaAnteriorCTNoDisponible
	}
	return &fuentePDFFirmaAnteriorCTDesarrollo{original: o, servicio: servicioDocumentosOriginalCTDesarrollo{servicio: servicio}}, nil
}

func (f *fuentePDFFirmaAnteriorCTDesarrollo) ObtenerPDFFirmaAnterior(ctx context.Context, q ports.SolicitudPDFFirmaAnterior) (ports.PDFFirmaAnterior, error) {
	var vacio ports.PDFFirmaAnterior
	if f == nil || f.original == nil || f.original.politicas == nil || dependenciaEsNulaContratacionTemporalDesarrollo(f.original.pdp) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(f.servicio) || ctx == nil {
		return vacio, errPDFFirmaAnteriorCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if q.Validar() != nil || q.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacio, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	expediente := ports.ExpedienteDocumentalRef(q.OrganizacionRef, q.ExpedienteRef)
	consulta, err := f.original.autorizarDescarga(ctx, q.DocumentoRef, q.DocumentoVersion, expediente)
	if err != nil {
		return vacio, err
	}
	pdf, documento, err := f.servicio.DescargarOriginalConDocumento(ctx, consulta)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		return vacio, err
	}
	defer clear(pdf.Contenido)
	suma := sha256.Sum256(pdf.Contenido)
	if documento.ID != q.DocumentoRef || documento.Version != q.DocumentoVersion || documento.ModuloID != moduloProductorCustodiaCT ||
		documento.ExpedienteRef != expediente || !f.original.politicas.CustodiaFirmadoReservada(documento.TipoRef) ||
		documento.MIME != "application/pdf" || pdf.MIME != "application/pdf" ||
		documento.HuellaSHA256 != q.DocumentoHuella || pdf.HuellaSHA256 != q.DocumentoHuella ||
		hex.EncodeToString(suma[:]) != q.DocumentoHuella || len(pdf.Contenido) == 0 ||
		len(pdf.Contenido) > ports.MaximoDocumentoFirmaBytes || !bytes.HasPrefix(pdf.Contenido, []byte("%PDF-")) {
		return vacio, ports.ErrAntecedenteFirmaR5NoAcreditado
	}
	return ports.PDFFirmaAnterior{Solicitud: q, Contenido: bytes.Clone(pdf.Contenido)}, nil
}
