package ports

import (
	"context"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

// SolicitudLiquidacionEconomica solo circula dentro del servidor. La
// preparación se construye con lecturas nominales; HTTP no la deserializa.
type SolicitudLiquidacionEconomica struct {
	Decision    SolicitudDecisionCircuito
	Preparacion *domain.PreparacionLiquidacionPublicada
}

// Registrar consume V3 para dietas.liquidacion_economica.registrar, audiencia
// vec_dietas.liquidacion_economica.registrar.v1, y confirma en una transacción
// documento/catalogo vigentes, competencia, versión, acto, recibo, auditoría
// y outbox. La repetición devuelve la constancia original tras reautorizar.
type RepositorioLiquidacionEconomica interface {
	Registrar(context.Context, IdentidadEfectivaCircuito, SolicitudLiquidacionEconomica) (domain.InstantaneaLiquidacionEconomica, error)
}

type RelojLiquidacionEconomica interface{ AhoraUTC() time.Time }

// CapturaLecturaLiquidacionEconomica contiene la sesión y evaluación de
// permisos producidas en la frontera común. La orden trae 33 claves; SQL toma
// unidad y titular del acto conservado y las añade antes del asiento común.
type CapturaLecturaLiquidacionEconomica struct {
	ContextoCanonico []byte
	VinculoCanonico  []byte
	Orden            json.RawMessage
	ValidaHasta      time.Time
}

type ConsultaLiquidacionEconomica struct {
	Referencia string
	Version    uint64 // 0 consulta el último acto de esta comisión; >=2 fija la versión.
}

type AuditoriaLecturaLiquidacionEconomica struct {
	Referencia     string `json:"auditoria_ref"`
	Secuencia      uint64 `json:"secuencia"`
	HuellaSHA256   string `json:"huella_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
	RegistradaEn   string `json:"registrada_en"`
}

type ResultadoLecturaLiquidacionEconomica struct {
	Liquidacion domain.InstantaneaLiquidacionEconomica
	Auditoria   AuditoriaLecturaLiquidacionEconomica
}

// Consultar registra en la auditoría común la lectura nominal antes de
// devolver la instantánea histórica; una denegación no entrega datos.
type LectorLiquidacionEconomica interface {
	Consultar(context.Context, CapturaLecturaLiquidacionEconomica, ConsultaLiquidacionEconomica) (ResultadoLecturaLiquidacionEconomica, error)
}
