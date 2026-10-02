package almacen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	"vec-diputacion-granada/internal/vec/ports"
)

const moduloOriginalFirmableCT = "contratacion_temporal"

// ServicioDocumentosOriginalCT es la capacidad existente de Documentos. El
// adaptador no mantiene otro registro ni accede directamente a objetos.
type ServicioDocumentosOriginalCT interface {
	AltaGenerado(context.Context, docports.AltaGenerado) (docdomain.Documento, error)
	DescargarOriginal(context.Context, docports.ConsultaDocumento) (docports.Original, error)
}

// AutorizacionesDocumentosOriginalCT se monta solo en una frontera confiable.
// Cada método obtiene concesiones V3 de Documentos para el recurso exacto; la
// solicitud CT no transporta un permiso ni crea contexto de almacén.
type AutorizacionesDocumentosOriginalCT interface {
	AutorizarLecturaOriginalCT(context.Context, ports.SolicitudOriginalFirmableCT, string) (docports.ConsultaDocumento, error)
	AutorizarAltaOriginalCT(context.Context, ports.SolicitudOriginalFirmableCT, ports.PDFOriginalCT, string) (docports.AltaGenerado, error)
}

type CustodiaDocumentosOriginalCT struct {
	servicio       ServicioDocumentosOriginalCT
	autorizaciones AutorizacionesDocumentosOriginalCT
}

func NuevaCustodiaDocumentosOriginalCT(servicio ServicioDocumentosOriginalCT, autorizaciones AutorizacionesDocumentosOriginalCT) (*CustodiaDocumentosOriginalCT, error) {
	if servicio == nil || autorizaciones == nil {
		return nil, ports.ErrOriginalFirmableCTNoDisponible
	}
	return &CustodiaDocumentosOriginalCT{servicio: servicio, autorizaciones: autorizaciones}, nil
}

func (a *CustodiaDocumentosOriginalCT) LeerOriginal(ctx context.Context, s ports.SolicitudOriginalFirmableCT, ref string) (ports.OriginalFirmableCT, error) {
	if a == nil || a.servicio == nil || a.autorizaciones == nil || ctx == nil || ctx.Err() != nil ||
		s.Validar() != nil || ref != ports.ReferenciaOriginalFirmableCT(s) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	consulta, err := a.autorizaciones.AutorizarLecturaOriginalCT(ctx, s, ref)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if consulta.DocumentoID != ref || consulta.Version != s.OriginalVersion ||
		consulta.Autorizacion.RecursoRef != ref || consulta.Autorizacion.AmbitoRef != s.ExpedienteRef {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	original, err := a.servicio.DescargarOriginal(ctx, consulta)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if original.MIME != "application/pdf" || !huellaContenidoOriginalCT(original.Contenido, original.HuellaSHA256) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	return ports.OriginalFirmableCT{
		Referencia: ref, Version: s.OriginalVersion,
		HuellaSHA256: original.HuellaSHA256, Contenido: bytes.Clone(original.Contenido),
	}, nil
}

func (a *CustodiaDocumentosOriginalCT) GuardarUnaVez(ctx context.Context, s ports.SolicitudOriginalFirmableCT, pdf ports.PDFOriginalCT, ref, huella string) (ports.OriginalFirmableCT, error) {
	if a == nil || a.servicio == nil || a.autorizaciones == nil || ctx == nil || ctx.Err() != nil ||
		s.Validar() != nil || ref != ports.ReferenciaOriginalFirmableCT(s) ||
		!docdomain.ReferenciaOpacaValida(pdf.TipoRef) || !huellaContenidoOriginalCT(pdf.Contenido, huella) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	// Leer primero consume la autorización de Documentos y evita una nueva
	// escritura en un replay. Solo la ausencia nominal permite intentar el alta.
	existente, err := a.LeerOriginal(ctx, s, ref)
	if err == nil {
		return cotejarOriginalCTGuardado(existente, pdf, huella)
	}
	if !errors.Is(err, docports.ErrNoEncontrado) {
		return ports.OriginalFirmableCT{}, err
	}
	alta, err := a.autorizaciones.AutorizarAltaOriginalCT(ctx, s, pdf, ref)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if alta.ID != ref || alta.ClaveIdempotencia != ports.ClaveAltaOriginalFirmableCT(s) ||
		alta.ModuloID != moduloOriginalFirmableCT || alta.ExpedienteRef != s.ExpedienteRef ||
		alta.TipoRef != pdf.TipoRef || alta.Version != s.OriginalVersion ||
		alta.MIME != "application/pdf" || !bytes.Equal(alta.Contenido, pdf.Contenido) ||
		alta.Autorizacion.RecursoRef != ref || alta.Autorizacion.AmbitoRef != s.ExpedienteRef {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	documento, err := a.servicio.AltaGenerado(ctx, alta)
	if err != nil && !errors.Is(err, docports.ErrConflicto) {
		return ports.OriginalFirmableCT{}, err
	}
	if err == nil && (documento.Validar() != nil || documento.ID != ref || documento.ModuloID != moduloOriginalFirmableCT ||
		documento.ExpedienteRef != s.ExpedienteRef || documento.TipoRef != pdf.TipoRef ||
		documento.Version != s.OriginalVersion || documento.HuellaSHA256 != huella ||
		documento.MIME != "application/pdf" || documento.Tamano != int64(len(pdf.Contenido))) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	// En concurrencia el índice único de Documentos decide el vencedor. Una
	// colisión solo es replay si una lectura nueva devuelve los mismos bytes.
	guardado, err := a.LeerOriginal(ctx, s, ref)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	return cotejarOriginalCTGuardado(guardado, pdf, huella)
}

func cotejarOriginalCTGuardado(original ports.OriginalFirmableCT, pdf ports.PDFOriginalCT, huella string) (ports.OriginalFirmableCT, error) {
	if original.HuellaSHA256 != huella || !bytes.Equal(original.Contenido, pdf.Contenido) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTConflicto
	}
	original.TipoRef = pdf.TipoRef
	return original, nil
}

func huellaContenidoOriginalCT(contenido []byte, huella string) bool {
	if len(contenido) < 8 || len(contenido) > ports.LimiteOriginalFirmableCT || !bytes.HasPrefix(contenido, []byte("%PDF-")) ||
		!docdomain.HuellaValida(huella) {
		return false
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]) == huella
}

var _ ports.CustodiaOriginalFirmableCT = (*CustodiaDocumentosOriginalCT)(nil)
