package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrBandejaOriginalC8DemasiadoGrande = errors.New("cronos: bandeja de justificantes demasiado grande")

// FuenteSolicitudOriginalC8 realiza una lectura nominal auditada del permiso
// y devuelve su titular y versión actual. El cuerpo HTTP solo aporta la
// referencia; ni el empleado objetivo ni el titular proceden del cliente.
type FuenteSolicitudOriginalC8 interface {
	PrepararOriginal(context.Context, OrdenOriginalC8, string, ModoLecturaOriginalC8) (SolicitudOriginalC8, error)
}

type ModoLecturaOriginalC8 string

const (
	LecturaOriginalPropia ModoLecturaOriginalC8 = "propio"
	LecturaOriginalRRHH   ModoLecturaOriginalC8 = "rrhh"
)

type SolicitudOriginalC8 struct {
	SolicitudRef        string
	EmpleadoRef         string
	PersonaTitularRef   string
	EstadoPermiso       domain.EstadoSolicitudPermiso
	VersionPermiso      int64
	PendienteJustificar bool
	JustificanteExigido bool
	ContextoRef         string
	Actual              *domain.JustificacionOriginalC8V2
}

// Cada método exige su autoridad documental nominal. La consulta propia de
// Documentos no habilita por sí sola la lectura del original ajeno para RRHH.
type FuenteOriginalAdmitidoC8 interface {
	AcreditarOriginalPropio(context.Context, OrdenOriginalC8, SolicitudOriginalC8, string, uint64) (domain.OriginalJustificacionC8V2, error)
	AcreditarOriginalRRHH(context.Context, OrdenOriginalC8, SolicitudOriginalC8, string, uint64) (domain.OriginalJustificacionC8V2, error)
}

type ProveedorMaterialOriginalC8 interface {
	ProveerMaterialOriginal(context.Context, domain.MaterialJustificacionOriginalC8) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenOriginalC8 struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorMaterialOriginalC8
}

func NuevaOrdenOriginalC8(actor vecdomain.ContextoActor, proveedor ProveedorMaterialOriginalC8) (OrdenOriginalC8, error) {
	if actor.Validar() != nil || dependenciaRemotaNula(proveedor) {
		return OrdenOriginalC8{}, ErrJustificacionNoDisponible
	}
	c, err := actor.Clonar()
	if err != nil {
		return OrdenOriginalC8{}, ErrJustificacionNoDisponible
	}
	return OrdenOriginalC8{actor: c, proveedor: proveedor}, nil
}

func (o OrdenOriginalC8) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.actor.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrJustificacionNoDisponible
	}
	return o.actor.Clonar()
}

func (o OrdenOriginalC8) ProveedorMaterial() ProveedorMaterialOriginalC8 { return o.proveedor }

type ReciboOriginalC8 struct {
	SolicitudRef     string
	DocumentoID      string
	DocumentoVersion uint64
	SHA256           string
	Estado           domain.EstadoJustificacion
	Version          int64
	ReciboRef        string
	InstanteUTC      time.Time
	Replay           bool
}

// La transacción SQL de CRN24 consume V3, vuelve a consultar Documentos,
// compara el estado del permiso y escribe asiento, auditoría, outbox y recibo.
type RepositorioOriginalC8 interface {
	ConfirmarOriginal(context.Context, OrdenOriginalC8, domain.MaterialJustificacionOriginalC8,
		vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboOriginalC8, error)
}

type PeticionAnexoOriginalC8 struct {
	SolicitudRef, ClaveOperacion, DocumentoID string
	DocumentoVersion                          uint64
	VersionEsperada                           int64
}

type PeticionRevisionOriginalC8V2 struct {
	SolicitudRef, ClaveOperacion, MotivoRef string
	VersionEsperada                         int64
	Decision                                domain.EstadoJustificacion
}

// CapturaLecturaOriginalC8 procede de identidad y evaluación de permisos
// actuales del servidor. SQL registra la lectura en la auditoría común antes
// de devolver el estado o los recibos históricos.
type CapturaLecturaOriginalC8 struct {
	ContextoCanonico []byte
	VinculoCanonico  []byte
	Orden            json.RawMessage
	ValidaHasta      time.Time
}

type AuditoriaLecturaOriginalC8 struct {
	Referencia     string `json:"auditoria_ref"`
	HuellaSHA256   string `json:"huella_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
	RegistradaEn   string `json:"registrada_en"`
}

type HechoOriginalC8 struct {
	Version            int64                      `json:"version"`
	Estado             domain.EstadoJustificacion `json:"estado"`
	MotivoRef          string                     `json:"motivo_ref"`
	DocumentoID        string                     `json:"documento_id"`
	DocumentoVersion   uint64                     `json:"documento_version"`
	SHA256             string                     `json:"sha256"`
	ReciboDocumentoRef string                     `json:"recibo_documento_ref"`
	CustodioID         string                     `json:"custodio_id"`
	ReciboRef          string                     `json:"recibo_ref"`
	InstanteUTC        time.Time                  `json:"instante_utc"`
}

type LecturaOriginalC8 struct {
	SolicitudRef        string
	EmpleadoRef         string
	PersonaTitularRef   string
	EstadoPermiso       domain.EstadoSolicitudPermiso
	VersionPermiso      int64
	PendienteJustificar bool
	JustificanteExigido bool
	Historia            []HechoOriginalC8
	Auditoria           AuditoriaLecturaOriginalC8
}

type PendienteRevisionOriginalC8 struct {
	SolicitudRef     string                     `json:"solicitud_ref"`
	EmpleadoEtiqueta string                     `json:"empleado_etiqueta"`
	DocumentoID      string                     `json:"documento_id"`
	DocumentoVersion uint64                     `json:"documento_version"`
	SHA256           string                     `json:"sha256"`
	Estado           domain.EstadoJustificacion `json:"estado"`
	Version          int64                      `json:"version"`
	RegistradaEn     time.Time                  `json:"registrada_en"`
}

type BandejaOriginalC8 struct {
	Pendientes []PendienteRevisionOriginalC8
	Auditoria  AuditoriaLecturaOriginalC8
}

type LectorOriginalC8 interface {
	ConsultarOriginal(context.Context, CapturaLecturaOriginalC8, string) (LecturaOriginalC8, error)
	ConsultarBandejaOriginal(context.Context, CapturaLecturaOriginalC8) (BandejaOriginalC8, error)
}

// CapturadorLecturaOriginalC8 usa la sesión certificada y la evaluación
// actual de permisos; ninguna captura procede del cuerpo HTTP.
type CapturadorLecturaOriginalC8 interface {
	CapturarLecturaOriginalPropia(context.Context, OrdenOriginalC8, string) (CapturaLecturaOriginalC8, error)
	CapturarLecturaOriginalRRHH(context.Context, OrdenOriginalC8, string) (CapturaLecturaOriginalC8, error)
	CapturarBandejaOriginal(context.Context, OrdenOriginalC8) (CapturaLecturaOriginalC8, error)
}
