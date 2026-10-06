package postgres

import (
	"context"
	"errors"
	"sync/atomic"

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
	accesosCT atomic.Bool
}

func NuevoSelladorCadenaAuditoriaPostgreSQL(pool poolSelladoAuditoria) (*SelladorCadenaAuditoriaPostgreSQL, error) {
	if pool == nil {
		return nil, ErrSelladoAuditoriaPostgreSQL
	}
	return &SelladorCadenaAuditoriaPostgreSQL{pool: pool}, nil
}

// Sellar sella un lote de cada cadena, cada una en su transacción READ
// COMMITTED: las funciones lo exigen para partir de la cabeza que confirmó el
// sellado anterior.
func (s *SelladorCadenaAuditoriaPostgreSQL) Sellar(ctx context.Context) (ResultadoSelladoAuditoria, error) {
	var r ResultadoSelladoAuditoria
	if s == nil || s.pool == nil || ctx == nil {
		return r, ErrSelladoAuditoriaPostgreSQL
	}
	// Cada cadena en su transacción y sin depender de la otra: un fallo en V3
	// no deja caducar el plazo de los accesos de CT, ni al revés.
	errV3 := s.enTransaccion(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT interna, externa FROM vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5($1)`,
			LoteSelladoAuditoria).Scan(&r.Interna, &r.Externa)
	})
	return r, errors.Join(errV3, s.sellarAccesosCT(ctx, &r))
}

func (s *SelladorCadenaAuditoriaPostgreSQL) sellarAccesosCT(ctx context.Context, r *ResultadoSelladoAuditoria) error {
	var err error
	// CT183 puede instalarse con el servidor en marcha: mientras no esté, se
	// comprueba en cada pasada; una vez vista, se sella siempre.
	if !s.accesosCT.Load() {
		var instalada bool
		if err = s.enTransaccion(ctx, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT coalesce(has_function_privilege(
				to_regprocedure('vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1(integer)'),'EXECUTE'),false)`).Scan(&instalada)
		}); err != nil || !instalada {
			return err
		}
		s.accesosCT.Store(true)
	}
	return s.enTransaccion(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1($1)`,
			LoteSelladoAuditoria).Scan(&r.AccesosCT)
	})
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
