package ports

import (
	"context"
	"errors"
)

var (
	ErrOriginalFirmableCTInvalido     = errors.New("vec: original firmable CT invalido")
	ErrOriginalFirmableCTConflicto    = errors.New("vec: original firmable CT en conflicto")
	ErrOriginalFirmableCTNoDisponible = errors.New("vec: original firmable CT no disponible")
	ErrOriginalFirmableCTNoEncontrado = errors.New("vec: original firmable CT no encontrado")
)

const LimiteOriginalFirmableCT = 1 << 20

// OriginalRef es una expectativa, nunca una ruta ni una fuente de autoridad.
// El navegador no aporta bytes ni huella a este contrato.
type SolicitudOriginalFirmableCT struct {
	OrganizacionRef, ExpedienteRef, Documento, OriginalRef string
	OriginalVersion                                        uint64
}

type PDFOriginalCT struct {
	TipoRef   string
	Contenido []byte
}

type FuentePDFOriginalCT interface {
	ObtenerPDFOriginalCT(context.Context, SolicitudOriginalFirmableCT) (PDFOriginalCT, error)
}

// ResolutorTipoOriginalCT usa la versión publicada del catálogo documental.
// La clave Documento de la solicitud identifica el tipo; no transporta su ref.
type ResolutorTipoOriginalCT interface {
	ResolverTipoOriginalCT(context.Context, string) (string, error)
}

type OriginalFirmableCT struct {
	Referencia, TipoRef, HuellaSHA256 string
	Version                           uint64
	Contenido                         []byte
}

// CustodiaOriginalFirmableCT se implementa con el servicio Documentos y su
// registro SQL. Cada lectura consume autorización PDP vigente.
type CustodiaOriginalFirmableCT interface {
	GuardarUnaVez(context.Context, SolicitudOriginalFirmableCT, PDFOriginalCT, string, string) (OriginalFirmableCT, error)
	LeerOriginal(context.Context, SolicitudOriginalFirmableCT, string) (OriginalFirmableCT, error)
}
