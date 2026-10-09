package ports

import (
	"errors"
	"time"
)

var ErrResultadoBolsaRRHHNoConfiable = errors.New("contratacion temporal: resultado Bolsa no confiable")

// Respuesta y contacto están ligados al llamamiento y a la participación.
// SituacionActual es global de Bolsa y nunca equivale a una respuesta firme.
type ParticipacionResultadoBolsaRRHH struct {
	ParticipacionRef   string     `json:"participacion_ref"`
	Respuesta          *string    `json:"respuesta"`
	Modo               *string    `json:"modo"`
	ReciboRespuestaRef *string    `json:"recibo_respuesta_ref"`
	RespondidaEn       *time.Time `json:"respondida_en"`
	JustificanteRef    *string    `json:"justificante_ref"`
	ContactoResultado  *string    `json:"contacto_resultado"`
	ReciboContactoRef  *string    `json:"recibo_contacto_ref"`
	ContactoEn         *time.Time `json:"contacto_en"`
	SituacionActual    *string    `json:"situacion_actual"`
	ReciboSituacionRef *string    `json:"recibo_situacion_ref"`
	SituacionDesde     *time.Time `json:"situacion_desde"`
}

type VinculoResultadoBolsaRRHH struct {
	BolsaRef         string                            `json:"bolsa_ref"`
	LlamamientoRef   string                            `json:"llamamiento_ref"`
	ReciboEmisionRef string                            `json:"recibo_emision_ref"`
	ReciboVinculoRef string                            `json:"recibo_vinculo_ref"`
	VinculadoEn      time.Time                         `json:"vinculado_en"`
	EmitidoEn        time.Time                         `json:"emitido_en"`
	Participaciones  []ParticipacionResultadoBolsaRRHH `json:"participaciones"`
}

type EmisionVinculableBolsaRRHH struct {
	BolsaRef          string    `json:"bolsa_ref"`
	LlamamientoRef    string    `json:"llamamiento_ref"`
	ReciboEmisionRef  string    `json:"recibo_emision_ref"`
	ReferenciaVisible string    `json:"referencia_visible"`
	EmitidoEn         time.Time `json:"emitido_en"`
}

type ResultadoBolsaRRHH struct {
	Vinculos             []VinculoResultadoBolsaRRHH  `json:"vinculos"`
	TotalVinculos        uint64                       `json:"total_vinculos"`
	PersonasSolicitadas  *uint32                      `json:"personas_solicitadas"`
	AceptacionesFirmes   uint32                       `json:"aceptaciones_firmes"`
	EmisionesVinculables []EmisionVinculableBolsaRRHH `json:"emisiones_vinculables"`
	SiguienteCursor      *string                      `json:"siguiente_cursor"`
}
