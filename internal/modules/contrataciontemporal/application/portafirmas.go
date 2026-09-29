package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// EstadoPortafirmas consulta el conector y falla cerrado: sin conector, con
// error o con una respuesta incoherente, el portafirmas cuenta como no
// conectado. La indisponibilidad nunca se presenta como envío ni firma.
func EstadoPortafirmas(ctx context.Context, conector ports.ConectorPortafirmas) ports.EstadoConexionPortafirmas {
	noConectado := ports.EstadoConexionPortafirmas{Motivo: ports.MotivoPortafirmasConexionPendiente}
	if ctx == nil || nula(conector) {
		return noConectado
	}
	estado, err := conector.EstadoConexion(ctx)
	if err != nil || estado.Validar() != nil {
		return noConectado
	}
	return estado
}

// EnviarAPortafirmas pide el envío del borrador exacto. Solo devuelve un
// recibo si el conector acepta el envío del mismo documento; cualquier otra
// respuesta es ErrPortafirmasNoDisponible y no deja constancia de envío.
func EnviarAPortafirmas(ctx context.Context, conector ports.ConectorPortafirmas, sol ports.SolicitudEnvioPortafirmas) (ports.ReciboEnvioPortafirmas, error) {
	var cero ports.ReciboEnvioPortafirmas
	if ctx == nil || !domain.ReferenciaOpacaValida(sol.OrganizacionRef) || !domain.ReferenciaOpacaValida(sol.ExpedienteRef) ||
		sol.VersionExpediente == 0 || !domain.ClaveDocumentoFirmaValida(sol.Documento) || len(sol.Original) == 0 ||
		len(sol.Original) > ports.MaximoDocumentoFirmaBytes || sol.OriginalHuella != huella(sol.Original) ||
		!ports.ClaveIdempotenciaFirmaValida(sol.ClaveIdempotencia) {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	if !EstadoPortafirmas(ctx, conector).Conectado {
		return cero, ports.ErrPortafirmasNoDisponible
	}
	recibo, err := conector.Enviar(ctx, sol)
	if err != nil || !domain.ReferenciaOpacaValida(recibo.EnvioRef) || recibo.OriginalHuella != sol.OriginalHuella {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrPortafirmasNoDisponible
	}
	return recibo, nil
}
