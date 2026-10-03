package bootstrap

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El pool exclusivo solo registra intentos. AD169 coteja el LOGIN, su única
// membresía y el proceso/canal DBA en cada conexión y al escribir el asiento.
func nuevoRegistradorIntentosConsulta(ctx context.Context, cfg config.Config, referencia *pgxpool.Pool, reservados ...*pgxpool.Pool) (vecports.RegistradorIntentosAuditoria, auditoria.ConfiguracionIntentosConsulta, func(), error) {
	fallo := func() (vecports.RegistradorIntentosAuditoria, auditoria.ConfiguracionIntentosConsulta, func(), error) {
		return nil, auditoria.ConfiguracionIntentosConsulta{}, nil, auditoria.ErrNoDisponible
	}
	if ctx == nil || ctx.Err() != nil || !cfg.DevelopmentEnabledByDoubleKey() {
		return fallo()
	}
	c, err := leerConfiguracionAuditoriaIntentosDesarrollo(cfg)
	if err != nil {
		return fallo()
	}
	logins := make([]string, 0, len(reservados))
	for _, pool := range reservados {
		if pool == nil {
			return fallo()
		}
		logins = append(logins, pool.Config().ConnConfig.User)
	}
	registrador, proceso, cerrar, err := AbrirRegistradorIntentosAuditoriaDesarrollo(ctx, cfg, referencia, logins)
	if err != nil {
		return fallo()
	}
	if proceso != c.Proceso {
		cerrar()
		return fallo()
	}
	return registrador, auditoria.ConfiguracionIntentosConsulta{Proceso: proceso, Canal: c.Canal, Plazo: time.Duration(c.LimiteSegundos) * time.Second}, cerrar, nil
}
