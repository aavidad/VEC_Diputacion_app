package ports

import (
	"context"
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarSolicitudesDocumentalesRRHH    = "bolsa.solicitudes_documentales.consultar_rrhh"
	AudienciaConsultarSolicitudesDocumentalesRRHH = "vec_bolsa_llamamientos.solicitudes_documentales.consultar_rrhh.v1"
)

// SolicitudDocumentalPendienteRRHH contiene solo lo necesario para decidir
// sobre una regularización; la referencia y huella son datos declarados.
type SolicitudDocumentalPendienteRRHH struct {
	SolicitudRef, ContenidoSHA256, DocumentoRef, DocumentoSHA256 string
	FechaFinCausa, Estado, ReciboRef                             string
	Version                                                      int64
	RegistradaEn                                                 time.Time
}

type ConsultaSolicitudesDocumentalesRRHH interface {
	ListarSolicitudesDocumentalesPendientes(context.Context, string, string, string, time.Time, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]SolicitudDocumentalPendienteRRHH, error)
}
