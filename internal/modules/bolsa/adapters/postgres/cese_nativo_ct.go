package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const aplicarCeseNativoCTSQL = `SELECT vec_bolsa_llamamientos.aplicar_cese_nativo_ct_v1($1::jsonb,$2::text,$3::text,$4::text,$5::text,$6::text,$7::bytea,$8::bytea,$9::bytea,$10::bytea,$11::numeric,$12::numeric,$13::bytea,$14::bytea,$15::bytea,$16::bytea)`
const ajustesCeseNativoCTSQL = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','20s',true),set_config('idle_in_transaction_session_timeout','25s',true)`

type RepositorioCeseNativoCTPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoRepositorioCeseNativoCTPostgreSQL(pool *pgxpool.Pool) (*RepositorioCeseNativoCTPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrCeseNativoCTNoDisponible
	}
	return &RepositorioCeseNativoCTPostgreSQL{pool: pool}, nil
}

func (r *RepositorioCeseNativoCTPostgreSQL) AplicarCeseNativoCT(ctx context.Context, o ports.OrdenCeseNativoCT) (ports.ReciboCeseNativoCT, error) {
	var cero ports.ReciboCeseNativoCT
	if ctx == nil || r == nil || r.pool == nil || o.Material.ValidarEstructura() != nil {
		return cero, ports.ErrCeseNativoCTNoDisponible
	}
	b, err := json.Marshal(o.Solicitud.Selector)
	if err != nil || len(b) == 0 || len(b) > 4096 {
		return cero, ports.ErrCeseNativoCTInvalido
	}
	for intento := 0; intento < 3; intento++ {
		if err := ctx.Err(); err != nil {
			return cero, err
		}
		recibo, err := r.aplicarUna(ctx, o, b)
		if err == nil {
			return recibo, nil
		}
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "40001" && pg.Routine != "exec_stmt_raise" && !strings.Contains(pg.Where, "RAISE") {
			continue
		}
		return cero, errorCeseNativoCTSQL(err)
	}
	return cero, ports.ErrCeseNativoCTNoDisponible
}

func (r *RepositorioCeseNativoCTPostgreSQL) aplicarUna(ctx context.Context, o ports.OrdenCeseNativoCT, selector []byte) (ports.ReciboCeseNativoCT, error) {
	var cero ports.ReciboCeseNativoCT
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, err
	}
	defer func() {
		fin, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(fin)
	}()
	if _, err = tx.Exec(ctx, ajustesCeseNativoCTSQL); err != nil {
		return cero, err
	}
	m := o.Material
	s := o.Solicitud
	var raw []byte
	err = tx.QueryRow(ctx, aplicarCeseNativoCTSQL, string(selector), s.ParticipacionRef,
		s.CeseCTReciboRef, s.RelacionReciboRef, s.OcupacionReciboRef, o.ActorRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&raw)
	if err != nil {
		return cero, err
	}
	if len(raw) == 0 || len(raw) > 16<<10 {
		return cero, ports.ErrCeseNativoCTNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var recibo ports.ReciboCeseNativoCT
	if d.Decode(&recibo) != nil {
		return cero, ports.ErrCeseNativoCTNoDisponible
	}
	var extra any
	if !errors.Is(d.Decode(&extra), io.EOF) || recibo.ParticipacionRef != s.ParticipacionRef ||
		recibo.BolsaRef != s.Selector.BolsaRef || recibo.ReciboRef == "" || recibo.EventoRef == "" ||
		recibo.AuditoriaRef == "" || recibo.PoliticaVersion == 0 {
		return cero, ports.ErrCeseNativoCTNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, err
	}
	return recibo, nil
}

func errorCeseNativoCTSQL(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrCeseNativoCTDenegado
		case "VBC01", "23505":
			return ports.ErrCeseNativoCTConflicto
		case "22023":
			return ports.ErrCeseNativoCTInvalido
		}
	}
	return ports.ErrCeseNativoCTNoDisponible
}

var _ ports.RepositorioCeseNativoCT = (*RepositorioCeseNativoCTPostgreSQL)(nil)
