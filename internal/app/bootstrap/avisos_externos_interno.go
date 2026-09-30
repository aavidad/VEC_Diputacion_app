package bootstrap

import (
	"context"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Un proceso interno separado no compone el lector de direcciones externo.
// La configuración propia declara un productor; el repositorio consume V3 y
// escribe su outbox en la misma transacción que el llamamiento.
func componerAvisosExternosInterno(ctx context.Context, cfg config.Config, s *aplicacionbolsa.ServicioEmisionLlamamiento, r interface {
	puertosbolsa.LectorCandidatoParticipacion
	ActivarFuentesCorreo(context.Context) error
}, cerrar func()) (func(), error) {
	if cfg.PortalProceso != string(separacionportales.PortalInterno) || ctx == nil || s == nil || r == nil {
		return cerrar, errAvisosExternos
	}
	nominal, ok := r.(interface {
		puertosbolsa.FuenteDestinatarioExternoParticipacion
		ActivarAvisosExternos(context.Context) error
	})
	if !ok {
		return cerrar, errAvisosExternos
	}
	c, err := leerConfiguracionAvisosExternos(cfg, "bolsa/avisos-externos.json")
	if err != nil || c.Lote != 0 || c.Intervalo != "" || c.Idioma != "" || c.URLPersonal != "" || nominal.ActivarAvisosExternos(ctx) != nil {
		return cerrar, errAvisosExternos
	}
	if s.EstablecerAvisosExternos(nominal, c.ProductorRef) != nil {
		return cerrar, errAvisosExternos
	}
	return cerrar, nil
}
