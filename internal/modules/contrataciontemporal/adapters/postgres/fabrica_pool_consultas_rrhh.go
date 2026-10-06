package postgres

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const rolConsultorRRHHPostgreSQL = "vec_contratacion_temporal_consultor_rrhh"

var patronLoginNominalConsultaRRHH = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// PoolConsultasRRHHPostgreSQL separa las credenciales con capacidad de
// consulta administrativa de cualquier otro adaptador. Su propietario debe
// llamar Cerrar.
type PoolConsultasRRHHPostgreSQL struct {
	pool      *pgxpool.Pool
	iniciador *iniciadorConsultasRRHHPostgreSQL
	cierre    *cierrePoolConsultasRRHH
}

type iniciadorConsultasRRHHPostgreSQL struct {
	dependencia  *PoolConsultasRRHHPostgreSQL
	loginNominal string
	modoTLS      modoTLSAcreditacionPoolO405
}

type cierrePoolConsultasRRHH struct {
	unaVez sync.Once
	cerrar func()
}

// NuevoPoolConsultasRRHHPostgreSQL crea un pool TLS exclusivo y liga la
// identidad configurada en la cadena al LOGIN nominal aprovisionado por
// Sistemas. El grupo técnico NOLOGIN nunca es una identidad aceptable.
func NuevoPoolConsultasRRHHPostgreSQL(
	ctx context.Context,
	cadenaConexion string,
	loginNominal string,
) (*PoolConsultasRRHHPostgreSQL, error) {
	return nuevoPoolConsultasRRHHPostgreSQL(
		ctx,
		cadenaConexion,
		loginNominal,
		modoTLSAcreditacionPoolO405Produccion,
	)
}

func nuevoPoolConsultasRRHHPostgreSQL(
	ctx context.Context,
	cadenaConexion string,
	loginNominal string,
	modo modoTLSAcreditacionPoolO405,
) (
	dependenciaResultado *PoolConsultasRRHHPostgreSQL,
	errResultado error,
) {
	var poolCreado *pgxpool.Pool
	transferido := false
	defer func() {
		if recover() != nil {
			dependenciaResultado = nil
			errResultado = errorPoolConsultasRRHH(ctx)
		}
		if !transferido && poolCreado != nil {
			poolCreado.Close()
		}
	}()
	if ctx == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	loginNominal = strings.TrimSpace(loginNominal)
	if !loginNominalConsultaRRHHValido(loginNominal) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	configuracion, err := pgxpool.ParseConfig(cadenaConexion)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		configuracion.ConnConfig.User != loginNominal ||
		!endurecerTLSFabricaPoolO405(
			&configuracion.ConnConfig.Config,
			modo,
		) ||
		!configuracionPoolAcreditacionO405Valida(configuracion, modo) {
		return nil, errorPoolConsultasRRHH(ctx)
	}
	poolCreado, err = pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil || poolCreado == nil {
		return nil, errorPoolConsultasRRHH(ctx)
	}
	dependencia := &PoolConsultasRRHHPostgreSQL{
		pool: poolCreado,
		cierre: &cierrePoolConsultasRRHH{
			cerrar: poolCreado.Close,
		},
	}
	dependencia.iniciador = &iniciadorConsultasRRHHPostgreSQL{
		dependencia: dependencia, loginNominal: loginNominal, modoTLS: modo,
	}
	transferido = true
	return dependencia, nil
}

func loginNominalConsultaRRHHValido(login string) bool {
	return login != "" &&
		login != rolConsultorRRHHPostgreSQL &&
		patronLoginNominalConsultaRRHH.MatchString(login) &&
		login == strings.TrimSpace(login)
}

func (p *PoolConsultasRRHHPostgreSQL) Cerrar() {
	if p == nil || p.iniciador == nil ||
		p.iniciador.dependencia != p ||
		p.cierre == nil || p.cierre.cerrar == nil {
		return
	}
	p.cierre.unaVez.Do(p.cierre.cerrar)
}

func (i *iniciadorConsultasRRHHPostgreSQL) BeginTx(
	ctx context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	if ctx == nil || i == nil || i.dependencia == nil ||
		i.dependencia.iniciador != i ||
		i.dependencia.pool == nil ||
		!loginNominalConsultaRRHHValido(i.loginNominal) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	configuracion := i.dependencia.pool.Config()
	if configuracion == nil || configuracion.ConnConfig == nil ||
		configuracion.ConnConfig.User != i.loginNominal ||
		!configuracionPoolAcreditacionO405Valida(configuracion, i.modoTLS) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	tx, err := i.dependencia.pool.BeginTx(ctx, opciones)
	if err != nil || tx == nil {
		return nil, errorPoolConsultasRRHH(ctx)
	}
	return tx, nil
}

func errorPoolConsultasRRHH(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrConsultaRRHHNoDisponible
}
