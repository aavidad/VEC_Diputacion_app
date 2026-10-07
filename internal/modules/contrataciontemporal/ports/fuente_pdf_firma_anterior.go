package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// La fuente lee exclusivamente los bytes de una revisión custodiada por
// Documentos, ligados al expediente y a la firma previamente acreditada.
type SolicitudPDFFirmaAnterior struct {
	OrganizacionRef, ExpedienteRef, Documento string
	FirmaRef, ReciboRef, DocumentoRef         string
	DocumentoVersion                          uint64
	DocumentoHuella                           string
}

func (q SolicitudPDFFirmaAnterior) Validar() error {
	if !domain.ReferenciaOpacaValida(q.OrganizacionRef) || !domain.ReferenciaOpacaValida(q.ExpedienteRef) || !domain.ClaveDocumentoFirmaValida(q.Documento) || !domain.ReferenciaOpacaValida(q.FirmaRef) || !domain.ReferenciaOpacaValida(q.ReciboRef) || !domain.ReferenciaOpacaValida(q.DocumentoRef) || q.DocumentoVersion == 0 || q.DocumentoVersion > 9007199254740991 || !domain.HuellaSHA256FirmaValida(q.DocumentoHuella) {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	return nil
}

type PDFFirmaAnterior struct {
	Solicitud SolicitudPDFFirmaAnterior
	Contenido []byte
}

type FuentePDFFirmaAnterior interface {
	ObtenerPDFFirmaAnterior(context.Context, SolicitudPDFFirmaAnterior) (PDFFirmaAnterior, error)
}
