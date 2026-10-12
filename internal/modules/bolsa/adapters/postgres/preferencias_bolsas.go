package postgres

import (
	"context"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const funcionOrdenarPreferenciasBolsas = "vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1"

type EscrituraPreferenciasBolsasPostgreSQL struct{ pool iniciadorTransacciones }

var _ bolsa.EscrituraPreferenciasBolsas = (*EscrituraPreferenciasBolsasPostgreSQL)(nil)

func NuevaEscrituraPreferenciasBolsasPostgreSQL(pool *pgxpool.Pool) (*EscrituraPreferenciasBolsasPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &EscrituraPreferenciasBolsasPostgreSQL{pool: pool}, nil
}

func (a *EscrituraPreferenciasBolsasPostgreSQL) OrdenarPreferenciasBolsas(ctx context.Context,
	s bolsa.OrdenarPreferenciasBolsas,
) (bolsa.ReciboPreferenciasBolsas, error) {
	if ctx == nil || a == nil || valorNulo(a.pool) || s.CandidatoRef == "" ||
		s.ConsultadaEn.IsZero() || s.Material.ValidarEstructura() != nil || len(s.Orden) == 0 ||
		s.VersionEsperada >= 2147483647 {
		return bolsa.ReciboPreferenciasBolsas{}, bolsa.ErrOrdenPreferenciasInvalido
	}
	var recibo bolsa.ReciboPreferenciasBolsas
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var err error
		recibo, err = a.ordenarIntento(ctx, s)
		return err
	})
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, errorPreferenciasBolsas(ctx, err)
	}
	return recibo, nil
}

func (a *EscrituraPreferenciasBolsasPostgreSQL) ordenarIntento(ctx context.Context,
	s bolsa.OrdenarPreferenciasBolsas,
) (bolsa.ReciboPreferenciasBolsas, error) {
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
 set_config('row_security','on',true),set_config('timezone','UTC',true),
 set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
 set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	m := s.Material
	var version int64
	var resultado bolsa.ReciboPreferenciasBolsas
	err = tx.QueryRow(ctx, `SELECT * FROM `+funcionOrdenarPreferenciasBolsas+`(
 $1::text,$2::text[],$3::bigint,$4::text,$5::timestamptz,
 $6::bytea,$7::bytea,$8::bytea,$9::bytea,$10::numeric,$11::numeric,
 $12::bytea,$13::bytea,$14::bytea,$15::bytea)`,
		s.CandidatoRef, s.Orden, int64(s.VersionEsperada), s.ClaveIdempotencia, s.ConsultadaEn.UTC(),
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(
		&resultado.Reutilizado, &resultado.ReciboRef, &version, &resultado.Orden, &resultado.RegistradoEn)
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	if version < 1 || resultado.ReciboRef == "" || resultado.RegistradoEn.IsZero() ||
		!slices.Equal(resultado.Orden, s.Orden) {
		return bolsa.ReciboPreferenciasBolsas{}, bolsa.ErrPreferenciasNoDisponibles
	}
	resultado.Version = uint64(version)
	if err = tx.Commit(ctx); err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	return resultado, nil
}

func errorPreferenciasBolsas(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501":
			return errors.Join(dominiovec.ErrAutorizacionDenegada, bolsa.ErrOrdenPreferenciasInvalido)
		case "VBP01", "VBP03":
			return bolsa.ErrPreferenciasDesactualizadas
		case "VBP02", "22023":
			return bolsa.ErrOrdenPreferenciasInvalido
		case "40001", "40P01", "55P03", "57014":
			return bolsa.ErrPreferenciasNoDisponibles
		}
	}
	return bolsa.ErrPreferenciasNoDisponibles
}
