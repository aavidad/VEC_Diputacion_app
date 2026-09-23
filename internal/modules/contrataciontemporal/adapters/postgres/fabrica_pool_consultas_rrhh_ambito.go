package postgres

import (
	"context"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const rolConsultorRRHHAmbitoPostgreSQL = "vec_contratacion_temporal_consultor_rrhh_ambito"

// PoolConsultasRRHHConAmbitoPostgreSQL es un tipo nominal: no satisface el
// constructor de la sesion legacy y solo sirve las fachadas CT109.
type PoolConsultasRRHHConAmbitoPostgreSQL struct {
	pool      *pgxpool.Pool
	iniciador *iniciadorConsultasRRHHConAmbitoPostgreSQL
	cierre    sync.Once
}

type iniciadorConsultasRRHHConAmbitoPostgreSQL struct {
	dependencia  *PoolConsultasRRHHConAmbitoPostgreSQL
	loginNominal string
}

func NuevoPoolConsultasRRHHConAmbitoPostgreSQL(ctx context.Context, cadenaConexion, loginNominal string) (*PoolConsultasRRHHConAmbitoPostgreSQL, error) {
	if ctx == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	loginNominal = strings.TrimSpace(loginNominal)
	if !patronLoginNominalConsultaRRHH.MatchString(loginNominal) ||
		loginNominal == rolConsultorRRHHAmbitoPostgreSQL ||
		loginNominal == rolConsultorRRHHPostgreSQL {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	configuracion, err := pgxpool.ParseConfig(cadenaConexion)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		configuracion.ConnConfig.User != loginNominal ||
		!endurecerTLSFabricaPoolO405(&configuracion.ConnConfig.Config, modoTLSAcreditacionPoolO405Produccion) ||
		!configuracionPoolAcreditacionO405Valida(configuracion, modoTLSAcreditacionPoolO405Produccion) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil || pool == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	transferido := false
	defer func() {
		if !transferido {
			pool.Close()
		}
	}()
	if err := acreditarLoginNominalAmbitoRRHH(ctx, pool, loginNominal); err != nil {
		return nil, err
	}
	resultado := &PoolConsultasRRHHConAmbitoPostgreSQL{pool: pool}
	resultado.iniciador = &iniciadorConsultasRRHHConAmbitoPostgreSQL{dependencia: resultado, loginNominal: loginNominal}
	transferido = true
	return resultado, nil
}

const consultaLoginNominalAmbitoRRHH = `SELECT session_user::text=$1::text
 AND EXISTS (SELECT 1 FROM pg_roles u WHERE u.rolname=session_user
   AND u.rolcanlogin AND u.rolinherit AND NOT u.rolsuper
   AND NOT u.rolcreatedb AND NOT u.rolcreaterole
   AND NOT u.rolreplication AND NOT u.rolbypassrls)
 AND EXISTS (SELECT 1 FROM pg_roles g WHERE g.rolname='vec_contratacion_temporal_consultor_rrhh_ambito'
   AND NOT g.rolcanlogin AND g.rolinherit AND NOT g.rolsuper
   AND NOT g.rolcreatedb AND NOT g.rolcreaterole
   AND NOT g.rolreplication AND NOT g.rolbypassrls)
 AND (SELECT count(*) FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member
      WHERE u.rolname=session_user)=1
 AND EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member
      JOIN pg_roles g ON g.oid=m.roleid WHERE u.rolname=session_user
      AND g.rolname='vec_contratacion_temporal_consultor_rrhh_ambito'
      AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND NOT EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.member
      WHERE g.rolname='vec_contratacion_temporal_consultor_rrhh_ambito')`

type lectorLoginAmbitoRRHH interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func acreditarLoginNominalAmbitoRRHH(ctx context.Context, lector lectorLoginAmbitoRRHH, login string) error {
	if ctx == nil || dependenciaNula(lector) {
		return ports.ErrConsultaRRHHNoDisponible
	}
	var correcto bool
	if err := lector.QueryRow(ctx, consultaLoginNominalAmbitoRRHH, login).Scan(&correcto); err != nil || !correcto {
		return ports.ErrConsultaRRHHNoDisponible
	}
	return nil
}

func (p *PoolConsultasRRHHConAmbitoPostgreSQL) Cerrar() {
	if p == nil || p.pool == nil {
		return
	}
	p.cierre.Do(p.pool.Close)
}

func (i *iniciadorConsultasRRHHConAmbitoPostgreSQL) BeginTx(ctx context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if ctx == nil || i == nil || i.dependencia == nil || i.dependencia.iniciador != i ||
		i.dependencia.pool == nil || opciones.IsoLevel != pgx.Serializable ||
		opciones.AccessMode != pgx.ReadWrite {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	configuracion := i.dependencia.pool.Config()
	if configuracion == nil || configuracion.ConnConfig == nil ||
		configuracion.ConnConfig.User != i.loginNominal ||
		!configuracionPoolAcreditacionO405Valida(configuracion, modoTLSAcreditacionPoolO405Produccion) {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	tx, err := i.dependencia.pool.BeginTx(ctx, opciones)
	if err != nil || tx == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if err := acreditarLoginNominalAmbitoRRHH(ctx, tx, i.loginNominal); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}
