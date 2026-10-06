package httpinterno

import (
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// El error proviene del consumidor que cotejó el acuse con su orden. El
// transporte comprueba además su forma pública antes de escribir la cabecera.
func adjuntarAcuseAuditoriaLectura(w http.ResponseWriter, err error) {
	var auditado ports.FalloLecturaAuditado
	if !errors.As(err, &auditado) {
		return
	}
	a := auditado.AcuseLecturaAuditada()
	if len(a.AuditoriaRef) == 0 || len(a.AuditoriaRef) > 128 || a.Secuencia < 1 || a.Secuencia > 9007199254740991 ||
		len(a.HuellaSHA256) != 64 || !core.ReferenciaCorrelacionAutorizacionV2Valida(a.CorrelacionRef) ||
		a.RegistradaEn.IsZero() || a.RegistradaEn.Location() != time.UTC {
		return
	}
	for _, r := range a.AuditoriaRef {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return
		}
	}
	for _, r := range a.HuellaSHA256 {
		if (r < 'a' || r > 'f') && (r < '0' || r > '9') {
			return
		}
	}
	w.Header().Set("X-Audit-Ref", a.AuditoriaRef)
	w.Header().Set("X-Correlation-Ref", a.CorrelacionRef)
}
