package ports

import (
	"fmt"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// CapturaEvaluacionSolicitudLigadaV3 transporta la instantanea usada en una
// evaluacion PDP. No concede acceso ni representa estado actual; el consumo
// del efecto sigue sujeto a la autoridad durable y su CAS.
type CapturaEvaluacionSolicitudLigadaV3 interface {
	fmt.Stringer
	slog.LogValuer
	LigarMaterial(
		domain.SolicitudAutorizacionLigadaV3,
		domain.ResultadoContextoActorRegistradoV2,
		domain.DecisionAutorizacionLigadaV3,
		ConfirmacionRegistroConcesionAutorizacionLigadaV3,
		ExportacionMaterialConsumoAutorizacionAtestadaV3,
		string,
	) (CapturaEvaluacionSolicitudLigadaV3, error)
	InstantaneaPara(
		domain.SolicitudAutorizacionLigadaV3,
		domain.ResultadoContextoActorRegistradoV2,
		domain.DecisionAutorizacionLigadaV3,
		ConfirmacionRegistroConcesionAutorizacionLigadaV3,
		ExportacionMaterialConsumoAutorizacionAtestadaV3,
		string,
		time.Time,
	) (domain.InstantaneaAutorizacion, error)
}
