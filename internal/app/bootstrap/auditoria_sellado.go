package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"vec-diputacion-granada/config"
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
// auditoría (AD207) mientras viva el servidor. Sin conexión configurada los
// asientos se siguen escribiendo, protegidos y pendientes, pero nadie los
// encadena: se avisa como error y no se impide arrancar.
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
	// CT183 es opcional: si está instalada, el mismo LOGIN sella también los
	// accesos RRHH de Contratación temporal.
	var accesosCT bool
	if err = pool.QueryRow(ctx, `SELECT coalesce(has_function_privilege(
		to_regprocedure('vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1(integer)'),'EXECUTE'),false)`).Scan(&accesosCT); err != nil {
		pool.Close()
		return nada, falloPostgreSQLCTDesarrollo(err)
	}
	sellador, err := vecpostgres.NuevoSelladorCadenaAuditoriaPostgreSQL(pool, accesosCT)
	if err != nil {
		pool.Close()
		return nada, err
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
			pasada, cancelarPasada := context.WithTimeout(ctx, 30*time.Second)
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
