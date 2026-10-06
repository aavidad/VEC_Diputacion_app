package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrSelladoAuditoriaPostgreSQL = errors.New("sellado_auditoria_postgresql_no_disponible")

// LoteSelladoAuditoria es el máximo de asientos que sella cada llamada por
// cadena. Unos 0,4 s de trabajo en el laboratorio; mantiene corta la
// transacción del sellador.
const LoteSelladoAuditoria = 10000

type poolSelladoAuditoria interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// ResultadoSelladoAuditoria cuenta los asientos sellados en una pasada.
type ResultadoSelladoAuditoria struct {
	Interna, Externa, AccesosCT int
}

// Mayor devuelve el mayor lote de la pasada.
func (r ResultadoSelladoAuditoria) Mayor() int {
	return max(r.Interna, r.Externa, r.AccesosCT)
}

// SelladorCadenaAuditoriaPostgreSQL ejecuta solo las fachadas de sellado de
// AD207 (cadenas interna y externa) y CT183 (accesos RRHH de Contratación
// temporal). No recibe ni lee datos de personas: las funciones trabajan con
// números, referencias y huellas.
type SelladorCadenaAuditoriaPostgreSQL struct {
	pool      poolSelladoAuditoria
	accesosCT bool
}

// NuevoSelladorCadenaAuditoriaPostgreSQL: accesosCT indica que CT183 está
// instalada y el LOGIN puede sellar también esa cadena.
func NuevoSelladorCadenaAuditoriaPostgreSQL(pool poolSelladoAuditoria, accesosCT bool) (*SelladorCadenaAuditoriaPostgreSQL, error) {
	if pool == nil {
		return nil, ErrSelladoAuditoriaPostgreSQL
	}
	return &SelladorCadenaAuditoriaPostgreSQL{pool: pool, accesosCT: accesosCT}, nil
}

// Sellar sella un lote de cada cadena, cada una en su transacción READ
// COMMITTED: las funciones lo exigen para partir de la cabeza que confirmó el
// sellado anterior.
func (s *SelladorCadenaAuditoriaPostgreSQL) Sellar(ctx context.Context) (ResultadoSelladoAuditoria, error) {
	var r ResultadoSelladoAuditoria
	if s == nil || s.pool == nil || ctx == nil {
		return r, ErrSelladoAuditoriaPostgreSQL
	}
	err := s.enTransaccion(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT interna, externa FROM vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5($1)`,
			LoteSelladoAuditoria).Scan(&r.Interna, &r.Externa)
	})
	if err != nil || !s.accesosCT {
		return r, err
	}
	err = s.enTransaccion(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1($1)`,
			LoteSelladoAuditoria).Scan(&r.AccesosCT)
	})
	return r, err
}

func (s *SelladorCadenaAuditoriaPostgreSQL) enTransaccion(ctx context.Context, f func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})
	if err != nil {
		return errors.Join(ErrSelladoAuditoriaPostgreSQL, err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err = f(tx); err != nil {
		return errors.Join(ErrSelladoAuditoriaPostgreSQL, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return errors.Join(ErrSelladoAuditoriaPostgreSQL, err)
	}
	return nil
}
