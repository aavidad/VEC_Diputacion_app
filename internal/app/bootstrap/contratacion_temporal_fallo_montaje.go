package bootstrap

import (
	"log/slog"

	"vec-diputacion-granada/internal/shared/telemetria"
)

// Las etiquetas de las guardas proceden del código; la causa conserva su
// identidad para el llamador, pero su texto libre nunca llega al registro.
func registrarFalloMontajeCTDesarrollo(componente string, causa error) error {
	if causa == nil {
		return nil
	}
	fallo := marcarFalloComponenteArranque(componente, causa)
	registrarAvisoFalloMontajeCTDesarrollo(fallo)
	return fallo
}

func registrarAvisoFalloMontajeCTDesarrollo(fallo error) {
	slog.Warn("contratacion_temporal_no_disponible",
		"componente", ComponenteFalloArranque(fallo),
		"causa", telemetria.ClaseErrorArranque(fallo),
		"mensaje", telemetria.MensajeErrorArranque(fallo))
}
