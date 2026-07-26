package postgres

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type selloFabricaPoolO405 struct {
	dependencia              *PoolRecuperacionCoberturaO405PostgreSQL
	modo                     modoTLSAcreditacionPoolO405
	callbacksPredeterminados bool
}

// PoolRecuperacionCoberturaO405PostgreSQL mantiene privada la configuración
// de transporte acreditada. Su propietario explícito debe llamar Cerrar.
type PoolRecuperacionCoberturaO405PostgreSQL struct {
	pool   *pgxpool.Pool
	sello  *selloFabricaPoolO405
	cierre *cierrePoolRecuperacionCoberturaO405
}

type cierrePoolRecuperacionCoberturaO405 struct {
	unaVez sync.Once
	cerrar func()
}

// NuevoPoolRecuperacionCoberturaO405PostgreSQL crea el único tipo de pool
// aceptado por el constructor O4-05. La fábrica conserva en privado la
// configuración usada, impide inyectar callbacks de transporte o protocolo y
// no retiene ownership tras devolver el pool.
func NuevoPoolRecuperacionCoberturaO405PostgreSQL(
	ctx context.Context,
	cadenaConexion string,
) (*PoolRecuperacionCoberturaO405PostgreSQL, error) {
	return nuevoPoolRecuperacionCoberturaO405PostgreSQL(
		ctx,
		cadenaConexion,
		modoTLSAcreditacionPoolO405Produccion,
	)
}

func nuevoPoolRecuperacionCoberturaO405PostgreSQL(
	ctx context.Context,
	cadenaConexion string,
	modo modoTLSAcreditacionPoolO405,
) (
	dependenciaResultado *PoolRecuperacionCoberturaO405PostgreSQL,
	errResultado error,
) {
	var poolCreado *pgxpool.Pool
	transferido := false
	defer func() {
		if recover() != nil {
			dependenciaResultado = nil
			errResultado =
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
		}
		if !transferido && poolCreado != nil {
			poolCreado.Close()
		}
	}()
	if dependenciaNula(ctx) {
		return nil,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	configuracion, err := pgxpool.ParseConfig(cadenaConexion)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		!endurecerTLSFabricaPoolO405(
			&configuracion.ConnConfig.Config,
			modo,
		) ||
		!configuracionPoolAcreditacionO405Valida(configuracion, modo) {
		return nil, errorAcreditacionPoolO405(ctx)
	}
	poolCreado, err = pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil || poolCreado == nil {
		return nil, errorAcreditacionPoolO405(ctx)
	}
	dependencia := &PoolRecuperacionCoberturaO405PostgreSQL{
		pool: poolCreado,
		cierre: &cierrePoolRecuperacionCoberturaO405{
			cerrar: poolCreado.Close,
		},
	}
	dependencia.sello = &selloFabricaPoolO405{
		dependencia:              dependencia,
		modo:                     modo,
		callbacksPredeterminados: true,
	}
	transferido = true
	return dependencia, nil
}

// Cerrar materializa el ownership del llamador. Es idempotente y una copia
// accidental del wrapper no puede cerrar el pool del original.
func (p *PoolRecuperacionCoberturaO405PostgreSQL) Cerrar() {
	if p == nil || p.sello == nil || p.sello.dependencia != p ||
		p.cierre == nil || p.cierre.cerrar == nil {
		return
	}
	p.cierre.unaVez.Do(p.cierre.cerrar)
}

func endurecerTLSFabricaPoolO405(
	configuracion *pgconn.Config,
	modo modoTLSAcreditacionPoolO405,
) bool {
	if configuracion == nil {
		return false
	}
	switch modo {
	case modoTLSAcreditacionPoolO405Produccion:
		if !endurecerTLSFabricaPoolO405Destino(configuracion.TLSConfig) {
			return false
		}
		for _, alternativa := range configuracion.Fallbacks {
			if alternativa == nil ||
				!endurecerTLSFabricaPoolO405Destino(
					alternativa.TLSConfig,
				) {
				return false
			}
		}
		return true
	case modoTLSAcreditacionPoolO405SocketUnixPrueba:
		return configuracionTLSAcreditacionPoolO405Valida(
			configuracion,
			modo,
		)
	default:
		return false
	}
}

func endurecerTLSFabricaPoolO405Destino(configuracion *tls.Config) bool {
	if configuracion == nil {
		return false
	}
	if configuracion.MinVersion == 0 {
		configuracion.MinVersion = tls.VersionTLS12
	}
	if configuracion.MinVersion < tls.VersionTLS13 &&
		len(configuracion.CipherSuites) == 0 {
		configuracion.CipherSuites =
			cipherSuitesTLS12AcreditacionPoolO405()
	}
	return true
}

func selloAcreditacionO405Valido(
	sello *selloFabricaPoolO405,
	modo modoTLSAcreditacionPoolO405,
) bool {
	return sello != nil && sello.modo == modo &&
		sello.callbacksPredeterminados &&
		sello.dependencia != nil &&
		sello.dependencia.sello == sello
}
