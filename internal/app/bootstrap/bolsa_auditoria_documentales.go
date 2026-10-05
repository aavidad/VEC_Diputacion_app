package bootstrap

import (
	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// El registrador común interno y su proceso vienen del loader dedicado con
// preflight confirmado. El llamador conserva su cierre y monta esta consulta
// en el manejador real de solicitudes documentales RRHH.
func nuevaConsultaDocumentalesBolsaAuditada(consulta application.ConsultorSolicitudesDocumentalesRRHH,
	registrador ports.RegistradorIntentosAuditoria, proceso string,
) (*application.ConsultaSolicitudesDocumentalesAuditada, error) {
	return application.NuevaConsultaSolicitudesDocumentalesAuditada(consulta, registrador, proceso)
}
