package ports

import (
	"context"
	"errors"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrCeseNativoCTInvalido     = errors.New("bolsa: cese nativo CT inválido")
	ErrCeseNativoCTDenegado     = errors.New("bolsa: cese nativo CT denegado")
	ErrCeseNativoCTConflicto    = errors.New("bolsa: cese nativo CT divergente")
	ErrCeseNativoCTNoDisponible = errors.New("bolsa: cese nativo CT no disponible")
)

// Todas las referencias son hechos opacos ya confirmados por CT, Personal y
// Bolsa. La participación procede del anclaje B113 leído por composición;
// B115 la resuelve de nuevo desde el origen B110 antes de escribir.
type SolicitudCeseNativoCT struct {
	Selector           SelectorAceptacionNativaCT
	ParticipacionRef   string
	CeseCTReciboRef    string
	RelacionReciboRef  string
	OcupacionReciboRef string
	Vinculo            core.VinculoAutenticacionActorV2
	ResultadoContexto  core.ResultadoContextoActorRegistradoV2
	Correlacion        core.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion core.ReferenciaEntradaCatalogo
}

type OrdenCeseNativoCT struct {
	Solicitud SolicitudCeseNativoCT
	ActorRef  string
	Material  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// El acuse usa las referencias B13/B45 originales. DisponibleDesde resulta
// de la política de cese propia de Bolsa; B90 decide elegibilidad al corte.
type ReciboCeseNativoCT struct {
	Reutilizada      bool   `json:"reutilizada"`
	ReciboRef        string `json:"recibo_ref"`
	AuditoriaRef     string `json:"auditoria_ref"`
	EventoRef        string `json:"evento_ref"`
	ParticipacionRef string `json:"participacion_ref"`
	BolsaRef         string `json:"bolsa_ref"`
	FechaEfecto      string `json:"fecha_efecto"`
	DisponibleDesde  string `json:"disponible_desde"`
	PoliticaVersion  uint64 `json:"politica_version"`
}

type RepositorioCeseNativoCT interface {
	AplicarCeseNativoCT(context.Context, OrdenCeseNativoCT) (ReciboCeseNativoCT, error)
}

type AplicadorCeseNativoCT interface {
	AplicarCeseNativoCT(context.Context, SolicitudCeseNativoCT) (ReciboCeseNativoCT, error)
}
