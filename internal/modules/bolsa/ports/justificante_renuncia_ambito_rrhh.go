package ports

import (
	"context"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El ámbito procede de la relación de Bolsa y se vuelve a comprobar con la
// sesión y el permiso vigentes antes de emitir cada decisión documental.
type AmbitoJustificanteRenunciaRRHH struct {
	BolsaRef         string `json:"bolsa_ref"`
	ParticipacionRef string `json:"participacion_ref"`
	LlamamientoRef   string `json:"llamamiento_ref"`
	UnidadRef        string `json:"unidad_ref"`
	AmbitoRef        string `json:"ambito_ref"`
	AuditoriaRef     string `json:"auditoria_ref,omitempty"`
}

type ResolutorAmbitoJustificanteRenunciaRRHH interface {
	ResolverAmbitoJustificanteRenunciaRRHH(context.Context, SolicitudConsultaJustificanteRenunciaRRHH) (AmbitoJustificanteRenunciaRRHH, error)
}

type CapturadorAmbitoJustificanteRenunciaRRHH interface {
	CapturarAmbitoJustificanteRenunciaRRHH(context.Context, SolicitudConsultaJustificanteRenunciaRRHH,
		AmbitoJustificanteRenunciaRRHH) (vecports.CapturaLecturaNominalBolsa, error)
}
