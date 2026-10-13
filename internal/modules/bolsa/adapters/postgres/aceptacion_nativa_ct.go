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
	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const funcionConsultaAceptacionNativaCT = `SELECT vec_bolsa_llamamientos.consultar_aceptacion_nativa_ct_v1($1::text,$2::text,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
const ajustesConsultaAceptacionNativaCT = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

type RepositorioConsultaAceptacionNativaCTPostgreSQL struct {
	pool  iniciadorPersonaAceptacionCT
	ahora func() time.Time
}

func NuevoRepositorioConsultaAceptacionNativaCTPostgreSQL(pool *pgxpool.Pool, ahora func() time.Time) (*RepositorioConsultaAceptacionNativaCTPostgreSQL, error) {
	if pool == nil || ahora == nil {
		return nil, ports.ErrAceptacionNativaCTNoDisponible
	}
	return &RepositorioConsultaAceptacionNativaCTPostgreSQL{pool, ahora}, nil
}

func (r *RepositorioConsultaAceptacionNativaCTPostgreSQL) consultar(ctx context.Context, o ports.OrdenConsultaAceptacionNativaCT, persona bool, validar func([]byte) error) error {
	if ctx == nil || r == nil || r.pool == nil || r.ahora == nil {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	for intento := 0; intento < 3; intento++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := application.ValidarOrdenConsultaAceptacionNativaCT(o, persona, r.ahora().UTC().Truncate(time.Microsecond)); err != nil {
			return err
		}
		p, err := application.PrepararConsultaAceptacionNativaCT(o.Solicitud, persona)
		if err != nil {
			return err
		}
		err = r.consultarUna(ctx, o, p.MaterialCanonico, persona, validar)
		if err == nil {
			return nil
		}
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "40001" || pg.Routine == "exec_stmt_raise" || strings.Contains(pg.Where, "RAISE") {
			if errors.As(err, &pg) && pg.Code == "42501" {
				return ports.ErrAceptacionNativaCTDenegada
			}
			return err
		}
	}
	return ports.ErrAceptacionNativaCTNoDisponible
}

func (r *RepositorioConsultaAceptacionNativaCTPostgreSQL) consultarUna(ctx context.Context, o ports.OrdenConsultaAceptacionNativaCT, canon []byte, persona bool, validar func([]byte) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	defer func() {
		fin, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(fin)
	}()
	if _, err = tx.Exec(ctx, ajustesConsultaAceptacionNativaCT); err != nil {
		return err
	}
	tipo := "anclaje"
	if persona {
		tipo = "persona"
	}
	m := o.Material
	var b []byte
	err = tx.QueryRow(ctx, funcionConsultaAceptacionNativaCT, string(canon), tipo, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&b)
	if err != nil {
		return err
	}
	if len(b) == 0 || len(b) > 16<<10 {
		return ports.ErrAceptacionNativaCTNoDisponible
	}
	if err = validar(b); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *RepositorioConsultaAceptacionNativaCTPostgreSQL) ConsultarAnclajeAceptacionNativaCT(ctx context.Context, o ports.OrdenConsultaAceptacionNativaCT) (ports.ResultadoAnclajeAceptacionNativaCT, error) {
	var cero ports.ResultadoAnclajeAceptacionNativaCT
	var x ports.ResultadoAnclajeAceptacionNativaCT
	err := r.consultar(ctx, o, false, func(b []byte) error {
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		if d.Decode(&x) != nil {
			return ports.ErrAceptacionNativaCTNoDisponible
		}
		var extra any
		if !errors.Is(d.Decode(&extra), io.EOF) || application.ValidarResultadoAnclajeAceptacionNativaCT(o, x, r.ahora().UTC().Truncate(time.Microsecond)) != nil {
			return ports.ErrAceptacionNativaCTNoDisponible
		}
		return nil
	})
	if err != nil {
		return cero, err
	}
	return x, nil
}

func (r *RepositorioConsultaAceptacionNativaCTPostgreSQL) ConsultarPersonaAceptacionNativaCT(ctx context.Context, o ports.OrdenConsultaAceptacionNativaCT) (ports.ResultadoPersonaAceptacionNativaCT, error) {
	var cero ports.ResultadoPersonaAceptacionNativaCT
	var x ports.ResultadoPersonaAceptacionNativaCT
	err := r.consultar(ctx, o, true, func(b []byte) error {
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		if d.Decode(&x) != nil {
			return ports.ErrAceptacionNativaCTNoDisponible
		}
		var extra any
		if !errors.Is(d.Decode(&extra), io.EOF) || application.ValidarResultadoPersonaAceptacionNativaCT(o, x, r.ahora().UTC().Truncate(time.Microsecond)) != nil {
			return ports.ErrAceptacionNativaCTNoDisponible
		}
		return nil
	})
	if err != nil {
		return cero, err
	}
	return x, nil
}

var _ ports.RepositorioConsultaAceptacionNativaCT = (*RepositorioConsultaAceptacionNativaCTPostgreSQL)(nil)
