package bootstrap

import (
	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	"vec-diputacion-granada/internal/vec/ports"
)

// El registrador procede del loader dedicado exterior con preflight confirmado.
// Proceso viene de esa misma configuración privada. La composición conecta el
// resultado a los dos manejadores personales; no abre otro pool ni autoridad.
func nuevasLecturasMiBolsaAuditadas(consulta mibolsa.ConsultorLecturaPropia, historial mibolsa.ConsultorHistorialPropio,
	registrador ports.RegistradorIntentosAuditoria, proceso string,
) (*mibolsa.LecturasAuditadas, error) {
	return mibolsa.NuevasLecturasAuditadas(consulta, historial, registrador, proceso)
}
