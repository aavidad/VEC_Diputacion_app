package consultafirmasv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// SolicitudDescarga sólo acepta la selección visible en la ficha. La
// referencia y la huella del objeto custodiado proceden de la lectura R5.
type SolicitudDescarga struct {
	ExpedienteRef    string
	VersionObservada uint64
	Documento        string
	FirmaRef         string
}

func (s SolicitudDescarga) Validar() error {
	if !domain.ReferenciaOpacaValida(s.ExpedienteRef) || s.VersionObservada == 0 ||
		s.VersionObservada > 9007199254740991 || !domain.ClaveDocumentoFirmaValida(s.Documento) ||
		!domain.ReferenciaOpacaValida(s.FirmaRef) {
		return ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return nil
}

// SeleccionDescarga es una selección de consulta obtenida del perfil nominal
// y del plan vigente. No se obtiene del cuerpo HTTP ni de una lectura libre.
type SeleccionDescarga struct {
	OrganizacionRef string
	Consulta        Solicitud
}

type SelectorDescarga interface {
	SeleccionarConsultaDescarga(context.Context, SolicitudDescarga) (SeleccionDescarga, error)
}

type ConsultaAutorizadaDescarga interface {
	Consultar(context.Context, Solicitud) (Resultado, error)
}

type ServicioDescarga struct {
	selector SelectorDescarga
	consulta ConsultaAutorizadaDescarga
	pdf      ports.FuentePDFFirmaAnterior
}

func NuevaDescarga(selector SelectorDescarga, consulta ConsultaAutorizadaDescarga,
	pdf ports.FuentePDFFirmaAnterior) (*ServicioDescarga, error) {
	if nulo(selector) || nulo(consulta) || nulo(pdf) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &ServicioDescarga{selector: selector, consulta: consulta, pdf: pdf}, nil
}

type ResultadoDescarga struct {
	ExpedienteRef    string
	VersionObservada uint64
	Documento        string
	FirmaRef         string
	ReciboRef        string
	DocumentoRef     string
	DocumentoVersion uint64
	HuellaSHA256     string
	PDF              []byte
}

// Descargar hace una consulta R5 nueva y auditada en esta misma petición.
// Después descarga por V3 de Documentos exactamente la revisión que figura
// en esa historia. Ningún dato de custodia viene de la persona solicitante.
func (s *ServicioDescarga) Descargar(ctx context.Context, q SolicitudDescarga) (ResultadoDescarga, error) {
	var cero ResultadoDescarga
	if s == nil || ctx == nil || nulo(s.selector) || nulo(s.consulta) || nulo(s.pdf) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := q.Validar(); err != nil {
		return cero, err
	}
	seleccion, err := s.selector.SeleccionarConsultaDescarga(ctx, q)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err != nil {
		return cero, err
	}
	if !domain.ReferenciaOpacaValida(seleccion.OrganizacionRef) || seleccion.Consulta.Validar() != nil ||
		seleccion.Consulta.ExpedienteRef != q.ExpedienteRef ||
		seleccion.Consulta.VersionExpediente != q.VersionObservada || seleccion.Consulta.Documento != q.Documento {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	historia, err := s.consulta.Consultar(ctx, seleccion.Consulta)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if historia.ExpedienteRef != q.ExpedienteRef || historia.VersionExpediente != q.VersionObservada ||
		historia.Documento != q.Documento || historia.ExpedienteDocumentalRef !=
		ports.ExpedienteDocumentalRef(seleccion.OrganizacionRef, q.ExpedienteRef) || len(historia.Firmas) > MaximoFilas {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	var elegida *Firma
	for i := range historia.Firmas {
		if historia.Firmas[i].FirmaRef == q.FirmaRef {
			if elegida != nil {
				return cero, ports.ErrResultadoFirmaDocumentoInvalido
			}
			elegida = &historia.Firmas[i]
		}
	}
	if elegida == nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	f := *elegida
	if f.Documento != q.Documento || f.VersionExpediente == 0 || f.VersionExpediente > q.VersionObservada ||
		f.Resultado != domain.ResultadoFirmaFirmado || !domain.ReferenciaOpacaValida(f.ReciboRef) ||
		f.Custodiado == nil || f.RevisionPDF == nil || !domain.ReferenciaOpacaValida(f.Custodiado.Ref) ||
		f.Custodiado.Version == 0 || !domain.HuellaSHA256FirmaValida(f.Custodiado.SHA256) ||
		f.RevisionPDF.RevisionSHA256 != f.Custodiado.SHA256 || f.RevisionPDF.RevisionLongitud == 0 ||
		f.RevisionPDF.RevisionLongitud > ports.MaximoDocumentoFirmaBytes {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	pedida := ports.SolicitudPDFFirmaAnterior{OrganizacionRef: seleccion.OrganizacionRef,
		ExpedienteRef: q.ExpedienteRef, Documento: q.Documento, FirmaRef: f.FirmaRef,
		ReciboRef: f.ReciboRef, DocumentoRef: f.Custodiado.Ref,
		DocumentoVersion: f.Custodiado.Version, DocumentoHuella: f.Custodiado.SHA256}
	if pedida.Validar() != nil {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	archivo, err := s.pdf.ObtenerPDFFirmaAnterior(ctx, pedida)
	if err != nil {
		clear(archivo.Contenido)
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		clear(archivo.Contenido)
		return cero, err
	}
	huella := sha256.Sum256(archivo.Contenido)
	if archivo.Solicitud != pedida || len(archivo.Contenido) != int(f.RevisionPDF.RevisionLongitud) ||
		!bytes.HasPrefix(archivo.Contenido, []byte("%PDF-")) ||
		hex.EncodeToString(huella[:]) != f.Custodiado.SHA256 {
		clear(archivo.Contenido)
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return ResultadoDescarga{ExpedienteRef: q.ExpedienteRef, VersionObservada: q.VersionObservada,
		Documento: q.Documento, FirmaRef: f.FirmaRef, ReciboRef: f.ReciboRef,
		DocumentoRef: f.Custodiado.Ref, DocumentoVersion: f.Custodiado.Version,
		HuellaSHA256: f.Custodiado.SHA256, PDF: archivo.Contenido}, nil
}
