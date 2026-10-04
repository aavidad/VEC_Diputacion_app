package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// FuenteCheckpointPeriodico conserva la captura y su auditoría en la misma
// transacción. Sólo devuelve capturas y acuses después de COMMIT confirmado.
// La recuperación devuelve la captura pendiente original; no la sustituye.
// La confirmación conserva el recibo exacto y audita su efecto conjuntamente.
type FuenteCheckpointPeriodico interface {
	CapturarCheckpointPendiente(context.Context) (CapturaCheckpointPeriodico, error)
	ConfirmarCheckpoint(context.Context, string, domain.ReciboCheckpointDesarrollo) (AcuseCheckpointPeriodico, error)
}

// CapturaCheckpointPeriodico sólo contiene coordenadas y metadatos técnicos.
// La fuente determina la ventana, cobertura y política; el ejecutor no las
// reconstruye ni recibe actores, permisos o registros personales del cliente.
type CapturaCheckpointPeriodico struct {
	Estado               string
	CapturaRef           string
	ConfiguracionVersion uint64
	ConfiguracionSHA256  string
	PinSPKISHA256        string
	Checkpoint           domain.CheckpointDesarrollo
	Acuse                AcuseCheckpointPeriodico
}

// AcuseCheckpointPeriodico prueba la persistencia declarada por la fuente.
// CapturaRef y ReciboHuellaSHA256 sólo se rellenan en la confirmación final.
// La huella del recibo es SHA256 de json.Marshal del recibo completo firmado.
type AcuseCheckpointPeriodico struct {
	AuditoriaRef       string
	Secuencia          uint64
	HuellaSHA256       string
	RegistradaEn       time.Time
	CorrelacionRef     string
	CapturaRef         string
	ReciboHuellaSHA256 string
}

func (CapturaCheckpointPeriodico) String() string {
	return "[CAPTURA-CHECKPOINT-PERIODICO]"
}
func (CapturaCheckpointPeriodico) GoString() string {
	return "ports.CapturaCheckpointPeriodico{[OPACA]}"
}
func (AcuseCheckpointPeriodico) String() string { return "[ACUSE-CHECKPOINT-PERIODICO]" }
func (AcuseCheckpointPeriodico) GoString() string {
	return "ports.AcuseCheckpointPeriodico{[OPACO]}"
}
