package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrResultadoEmisionCTNoDisponible = errors.New("bolsa: resultado de emisión CT no disponible")
	ErrResultadoEmisionCTNoEncontrado = errors.New("bolsa: emisión CT no encontrada")
	ErrResultadoEmisionCTDivergente   = errors.New("bolsa: resultado de emisión CT divergente")
	ErrResultadoEmisionCTDenegado     = errors.New("bolsa: lectura de resultado CT denegada")
)

// ResultadoParticipacionEmisionCT conserva hechos nativos B98. Una respuesta
// vacía significa que aún no hay respuesta registrada, sin inferir un plazo ni
// una resolución administrativa. modo=firme es dato propio de Bolsa.
type ResultadoParticipacionEmisionCT struct {
	ParticipacionRef   string     `json:"participacion_ref"`
	Respuesta          string     `json:"respuesta"`
	Modo               string     `json:"modo"`
	ReciboRespuestaRef string     `json:"recibo_respuesta_ref"`
	RespondidaEn       *time.Time `json:"respondida_en"`
	JustificanteRef    string     `json:"justificante_ref"`
	ContactoResultado  string     `json:"contacto_resultado"`
	ReciboContactoRef  string     `json:"recibo_contacto_ref"`
	ContactoEn         *time.Time `json:"contacto_en"`
	SituacionActual    string     `json:"situacion_actual"`
	ReciboSituacionRef string     `json:"recibo_situacion_ref"`
	SituacionDesde     *time.Time `json:"situacion_desde"`
}

type ResultadoEmisionCT struct {
	BolsaRef         string                            `json:"bolsa_ref"`
	LlamamientoRef   string                            `json:"llamamiento_ref"`
	ReciboEmisionRef string                            `json:"recibo_emision_ref"`
	EmitidoEn        time.Time                         `json:"emitido_en"`
	Participaciones  []ResultadoParticipacionEmisionCT `json:"participaciones"`
}

// LectorResultadoEmisionCT lo implementa composición CT sobre su transacción
// de detalle ya autorizada y auditada. El adaptador Bolsa ofrece lectura EnTx.
type LectorResultadoEmisionCT interface {
	LeerResultadoEmisionCT(context.Context, string, string, string) (ResultadoEmisionCT, error)
}
