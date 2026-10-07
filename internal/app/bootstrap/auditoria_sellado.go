package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecpostgres "vec-diputacion-granada/internal/vec/adapters/postgres"
)

// rolSelladoAuditoria es el grupo NOLOGIN de AD207 que solo puede ejecutar
// sellar_cadena_auditoria_v5 y, con CT183, sellar_cadena_accesos_rrhh_v1.
// El LOGIN de la conexión pertenece solo a él.
const rolSelladoAuditoria = "vec_auditoria_encadenador"

// pausaSelladoAuditoria: tras un lote pequeño se espera esto; si el lote pasa
// de loteContinuoSelladoAuditoria se repite sin esperar. Con carga alta el
// sellador no para, y con poca no satura la base de transacciones vacías.
const (
	pausaSelladoAuditoria        = 250 * time.Millisecond
	loteContinuoSelladoAuditoria = 1000
)

// iniciarSelladoAuditoria mantiene el sellado diferido de la cadena de
// auditoría (AD207 y CT183) mientras viva el servidor. Sin conexión
// configurada se avisa como error y se arranca igual (las bases sin AD207 no
// la necesitan); con AD207 instalada, a los 10 s sin sellar la base rechaza
// las operaciones auditadas.
func iniciarSelladoAuditoria(ctx context.Context, cfg config.Config) (func(), error) {
	nada := func() {}
	dsn, err := cfg.DSNAuditoriaSelladoSeparado()
	if err != nil {
		return nada, err
	}
	if dsn == "" {
		slog.Error("sellado de la cadena de auditoría sin conexión: los asientos quedan pendientes", "variable", config.EnvAuditoriaSelladoDatabaseURL)
		return nada, nil
	}
	pool, err := abrirPoolRelevoBolsaDesarrollo(ctx, dsn, rolSelladoAuditoria, "vec-auditoria-sellado")
	if err != nil {
		return nada, err
	}
	sellador, err := vecpostgres.NuevoSelladorCadenaAuditoriaPostgreSQL(pool)
	if err != nil {
		pool.Close()
		return nada, err
	}
	// Una pasada antes de atender peticiones: tras una parada larga el latido
	// ha caducado y, sin ella, las primeras operaciones auditadas se rechazarían.
	primera, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(30*time.Second))
	_, err = sellador.Sellar(primera)
	cancelar()
	if err != nil {
		slog.Error("primera pasada del sellado de auditoría fallida; se reintentará", "causa", causaFalloPostgreSQLCTDesarrollo(err))
	}
	detener := mantenerSelladoAuditoria(sellador.Sellar, pausaSelladoAuditoria, esperarTemporizadorCTDesarrollo)
	return func() { detener(); pool.Close() }, nil
}

func mantenerSelladoAuditoria(sellar func(context.Context) (vecpostgres.ResultadoSelladoAuditoria, error), pausa time.Duration, esperar esperaRenovacionCTDesarrollo) func() {
	ctx, cancelar := context.WithCancel(context.Background())
	terminado := make(chan struct{})
	go func() {
		defer close(terminado)
		fallando := false
		for ctx.Err() == nil {
			pasada, cancelarPasada := context.WithTimeout(ctx, plazoarranque.Ampliar(30*time.Second))
			resultado, err := sellar(pasada)
			cancelarPasada()
			switch {
			case err != nil && ctx.Err() == nil:
				// Un error por racha: el texto de pgconn puede llevar host o usuario.
				if !fallando {
					slog.Error("sellado de la cadena de auditoría no disponible; se reintentará", "causa", causaFalloPostgreSQLCTDesarrollo(err))
				}
				fallando = true
			case err == nil:
				if fallando {
					slog.Info("sellado de la cadena de auditoría recuperado")
				}
				fallando = false
				if resultado.Mayor() >= loteContinuoSelladoAuditoria {
					continue
				}
			}
			if esperar(ctx, pausa) != nil {
				return
			}
		}
	}()
	var una sync.Once
	return func() {
		una.Do(func() {
			cancelar()
			<-terminado
		})
	}
}
