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

// SelladorCadenaAuditoriaPostgreSQL ejecuta solo la fachada AD207 que calcula
// los eslabones de los asientos pendientes. No recibe ni lee datos de
// personas: la función trabaja con números, referencias y huellas.
type SelladorCadenaAuditoriaPostgreSQL struct{ pool poolSelladoAuditoria }

func NuevoSelladorCadenaAuditoriaPostgreSQL(pool poolSelladoAuditoria) (*SelladorCadenaAuditoriaPostgreSQL, error) {
	if pool == nil {
		return nil, ErrSelladoAuditoriaPostgreSQL
	}
	return &SelladorCadenaAuditoriaPostgreSQL{pool: pool}, nil
}

// Sellar sella un lote de cada cadena. READ COMMITTED: la función lo exige
// para partir de la cabeza que confirmó el sellado anterior.
func (s *SelladorCadenaAuditoriaPostgreSQL) Sellar(ctx context.Context) (interna, externa int, err error) {
	if s == nil || s.pool == nil || ctx == nil {
		return 0, 0, ErrSelladoAuditoriaPostgreSQL
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})
	if err != nil {
		return 0, 0, errors.Join(ErrSelladoAuditoriaPostgreSQL, err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err = tx.QueryRow(ctx, `SELECT interna, externa FROM vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5($1)`,
		LoteSelladoAuditoria).Scan(&interna, &externa); err != nil {
		return 0, 0, errors.Join(ErrSelladoAuditoriaPostgreSQL, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, 0, errors.Join(ErrSelladoAuditoriaPostgreSQL, err)
	}
	return interna, externa, nil
}
