package almacen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	"vec-diputacion-granada/internal/vec/ports"
)

const moduloOriginalFirmableCT = "contratacion_temporal"

// ServicioDocumentosOriginalCT es la capacidad existente de Documentos. El
// adaptador no mantiene otro registro ni accede directamente a objetos.
type ServicioDocumentosOriginalCT interface {
	CustodiarOriginalFirmable(context.Context, docports.OrdenCustodiarOriginalFirmable, docports.AutorizarOriginalFirmable) (docports.IntentoOriginalFirmable, error)
	DescargarOriginalConDocumento(context.Context, docports.ConsultaDocumento) (docports.Original, docdomain.Documento, error)
}

// AutorizacionesDocumentosOriginalCT se monta solo en una frontera confiable.
// Cada método obtiene concesiones V3 de Documentos para el recurso exacto; la
// solicitud CT no transporta un permiso ni crea contexto de almacén.
type AutorizacionesDocumentosOriginalCT interface {
	AutorizarLecturaOriginalCT(context.Context, ports.SolicitudOriginalFirmableCT, string) (docports.ConsultaDocumento, error)
	PrepararCustodiaOriginalCT(context.Context, ports.SolicitudOriginalFirmableCT, ports.PDFOriginalCT, string) (docports.OrdenCustodiarOriginalFirmable, docports.AutorizarOriginalFirmable, error)
}

// MapeadorExpedienteOriginalCT delega al propietario CT la referencia opaca
// del expediente documental. El adaptador común no interpreta sus IDs.
type MapeadorExpedienteOriginalCT interface {
	ReferenciaDocumentalCT(string) (string, error)
}

type FuncionMapeoExpedienteOriginalCT func(string) (string, error)

func (f FuncionMapeoExpedienteOriginalCT) ReferenciaDocumentalCT(expediente string) (string, error) {
	return f(expediente)
}

type CustodiaDocumentosOriginalCT struct {
	servicio       ServicioDocumentosOriginalCT
	autorizaciones AutorizacionesDocumentosOriginalCT
	mapear         MapeadorExpedienteOriginalCT
	tipos          ports.ResolutorTipoOriginalCT
}

func NuevaCustodiaDocumentosOriginalCT(servicio ServicioDocumentosOriginalCT, autorizaciones AutorizacionesDocumentosOriginalCT, mapear MapeadorExpedienteOriginalCT, tipos ports.ResolutorTipoOriginalCT) (*CustodiaDocumentosOriginalCT, error) {
	if servicio == nil || autorizaciones == nil || mapear == nil || tipos == nil {
		return nil, ports.ErrOriginalFirmableCTNoDisponible
	}
	return &CustodiaDocumentosOriginalCT{servicio: servicio, autorizaciones: autorizaciones, mapear: mapear, tipos: tipos}, nil
}

func (a *CustodiaDocumentosOriginalCT) LeerOriginal(ctx context.Context, s ports.SolicitudOriginalFirmableCT, ref string) (ports.OriginalFirmableCT, error) {
	identidad := identidadOriginalCT(s)
	if a == nil || a.servicio == nil || a.autorizaciones == nil || a.mapear == nil || a.tipos == nil || ctx == nil || ctx.Err() != nil ||
		!identidad.Valida() || (s.OriginalRef != "" && s.OriginalRef != identidad.Referencia()) || ref != identidad.Referencia() {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	expedienteDocumental, err := a.mapear.ReferenciaDocumentalCT(s.ExpedienteRef)
	if err != nil || !docdomain.ReferenciaOpacaValida(expedienteDocumental) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	consulta, err := a.autorizaciones.AutorizarLecturaOriginalCT(ctx, s, ref)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if consulta.DocumentoID != ref || consulta.Version != s.OriginalVersion ||
		consulta.Autorizacion.RecursoRef != ref || consulta.Autorizacion.AmbitoRef != expedienteDocumental {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	tipoEsperado, err := a.tipos.ResolverTipoOriginalCT(ctx, s.Documento)
	if err != nil || !almacencanonico.ReferenciaDocumentoValida(tipoEsperado) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	original, documento, err := a.servicio.DescargarOriginalConDocumento(ctx, consulta)
	if err != nil {
		if errors.Is(err, docports.ErrNoEncontrado) {
			return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoEncontrado
		}
		return ports.OriginalFirmableCT{}, err
	}
	if original.MIME != "application/pdf" || !huellaContenidoOriginalCT(original.Contenido, original.HuellaSHA256) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	if documento.Validar() != nil || documento.ID != ref || documento.Version != s.OriginalVersion ||
		documento.ModuloID != moduloOriginalFirmableCT || documento.ExpedienteRef != expedienteDocumental ||
		documento.TipoRef != tipoEsperado ||
		!documento.Descargable() || documento.MIME != original.MIME ||
		documento.HuellaSHA256 != original.HuellaSHA256 || documento.Tamano != int64(len(original.Contenido)) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	return ports.OriginalFirmableCT{
		Referencia: ref, Version: s.OriginalVersion,
		TipoRef:      documento.TipoRef,
		HuellaSHA256: original.HuellaSHA256, Contenido: bytes.Clone(original.Contenido),
	}, nil
}

func (a *CustodiaDocumentosOriginalCT) GuardarUnaVez(ctx context.Context, s ports.SolicitudOriginalFirmableCT, pdf ports.PDFOriginalCT, ref, huella string) (ports.OriginalFirmableCT, error) {
	identidad := identidadOriginalCT(s)
	if a == nil || a.servicio == nil || a.autorizaciones == nil || a.mapear == nil || a.tipos == nil || ctx == nil || ctx.Err() != nil ||
		!identidad.Valida() || (s.OriginalRef != "" && s.OriginalRef != identidad.Referencia()) || ref != identidad.Referencia() ||
		!docdomain.ReferenciaOpacaValida(pdf.TipoRef) || !huellaContenidoOriginalCT(pdf.Contenido, huella) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	expedienteDocumental, err := a.mapear.ReferenciaDocumentalCT(s.ExpedienteRef)
	if err != nil || !docdomain.ReferenciaOpacaValida(expedienteDocumental) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTInvalido
	}
	tipoEsperado, err := a.tipos.ResolverTipoOriginalCT(ctx, s.Documento)
	if err != nil || !almacencanonico.ReferenciaDocumentoValida(tipoEsperado) || pdf.TipoRef != tipoEsperado {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	// Leer primero consume la autorización de Documentos y evita una nueva
	// escritura en un replay. Solo la ausencia nominal permite intentar el alta.
	existente, err := a.LeerOriginal(ctx, s, ref)
	if err == nil {
		return cotejarOriginalCTGuardado(existente, pdf, huella)
	}
	if !errors.Is(err, ports.ErrOriginalFirmableCTNoEncontrado) {
		return ports.OriginalFirmableCT{}, err
	}
	orden, autoridad, err := a.autorizaciones.PrepararCustodiaOriginalCT(ctx, s, pdf, ref)
	if err != nil {
		return ports.OriginalFirmableCT{}, err
	}
	if autoridad == nil || orden.ID != ref || orden.ClaveIdempotencia != identidad.ClaveLogica() ||
		orden.ModuloID != moduloOriginalFirmableCT || orden.ExpedienteRef != expedienteDocumental ||
		orden.TipoRef != pdf.TipoRef || orden.Version != s.OriginalVersion ||
		orden.MIME != "application/pdf" || !bytes.Equal(orden.Contenido, pdf.Contenido) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTNoDisponible
	}
	intento, err := a.servicio.CustodiarOriginalFirmable(ctx, orden, autoridad)
	if err != nil && !errors.Is(err, docports.ErrConflicto) {
		return ports.OriginalFirmableCT{}, err
	}
	if err == nil && (intento.DocumentoID != ref || intento.HuellaSHA256 != huella || intento.Estado != "confirmado") {
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
	if original.TipoRef != pdf.TipoRef || original.HuellaSHA256 != huella || !bytes.Equal(original.Contenido, pdf.Contenido) {
		return ports.OriginalFirmableCT{}, ports.ErrOriginalFirmableCTConflicto
	}
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

func identidadOriginalCT(s ports.SolicitudOriginalFirmableCT) almacencanonico.IdentidadOriginalCT {
	return almacencanonico.IdentidadOriginalCT{
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		Documento: s.Documento, Version: s.OriginalVersion,
	}
}

var _ ports.CustodiaOriginalFirmableCT = (*CustodiaDocumentosOriginalCT)(nil)
