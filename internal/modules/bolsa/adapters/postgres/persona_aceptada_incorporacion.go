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

const funcionConsultaPersonaAceptacionCT = `SELECT vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

const ajustesConsultaPersonaAceptacionCT = `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`

type iniciadorPersonaAceptacionCT interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type RepositorioConsultaPersonaAceptacionCTPostgreSQL struct {
	pool  iniciadorPersonaAceptacionCT
	ahora func() time.Time
}

func NuevoRepositorioConsultaPersonaAceptacionCTPostgreSQL(pool *pgxpool.Pool, ahora func() time.Time) (*RepositorioConsultaPersonaAceptacionCTPostgreSQL, error) {
	if pool == nil || ahora == nil {
		return nil, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	return &RepositorioConsultaPersonaAceptacionCTPostgreSQL{pool: pool, ahora: ahora}, nil
}

func (r *RepositorioConsultaPersonaAceptacionCTPostgreSQL) ConsultarPersonaAceptacionCT(ctx context.Context, o ports.OrdenConsultaPersonaAceptacionCT) (ports.ResultadoConsultaPersonaAceptacionCT, error) {
	var cero ports.ResultadoConsultaPersonaAceptacionCT
	if ctx == nil || r == nil || r.pool == nil || r.ahora == nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	for intento := 0; intento < 3; intento++ {
		if err := ctx.Err(); err != nil {
			return cero, err
		}
		if err := application.ValidarOrdenConsultaPersonaAceptacionCT(o, r.ahora().UTC().Truncate(time.Microsecond)); err != nil {
			return cero, err
		}
		p, err := application.PrepararConsultaPersonaAceptacionCT(o.Solicitud)
		if err != nil {
			return cero, err
		}
		resultado, err := r.consultar(ctx, o, p.MaterialCanonico)
		if err == nil {
			return resultado, nil
		}
		if !serializacionPersonaAceptacionCT(err) {
			return cero, errorPersonaAceptacionCT(ctx, err)
		}
	}
	return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
}

func (r *RepositorioConsultaPersonaAceptacionCTPostgreSQL) consultar(ctx context.Context, o ports.OrdenConsultaPersonaAceptacionCT, canon []byte) (ports.ResultadoConsultaPersonaAceptacionCT, error) {
	var cero ports.ResultadoConsultaPersonaAceptacionCT
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, err
	}
	defer func() {
		fin, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(fin)
	}()
	if _, err = tx.Exec(ctx, ajustesConsultaPersonaAceptacionCT); err != nil {
		return cero, err
	}
	m := o.Material
	var b []byte
	err = tx.QueryRow(ctx, funcionConsultaPersonaAceptacionCT, string(canon), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&b)
	if err != nil {
		return cero, err
	}
	if len(b) == 0 || len(b) > 16<<10 {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var resultado ports.ResultadoConsultaPersonaAceptacionCT
	if d.Decode(&resultado) != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	var extra any
	if !errors.Is(d.Decode(&extra), io.EOF) || application.ValidarResultadoConsultaPersonaAceptacionCT(o, resultado, r.ahora().UTC().Truncate(time.Microsecond)) != nil {
		return cero, ports.ErrConsultaPersonaAceptacionCTNoDisponible
	}
	// Pendiente y ausencia autorizada conservan el consumo/auditoría de lectura.
	if err = tx.Commit(ctx); err != nil {
		return cero, err
	}
	return resultado, nil
}

func serializacionPersonaAceptacionCT(err error) bool {
	var p *pgconn.PgError
	return errors.As(err, &p) && p.Code == "40001" && p.Routine != "exec_stmt_raise" && !strings.Contains(p.Where, "RAISE")
}
func errorPersonaAceptacionCT(ctx context.Context, err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "42501" {
		return application.ErrorConsultaPersonaAceptacionCT(ctx, ports.ErrConsultaPersonaAceptacionCTDenegada)
	}
	return application.ErrorConsultaPersonaAceptacionCT(ctx, err)
}

var _ ports.RepositorioConsultaPersonaAceptacionCT = (*RepositorioConsultaPersonaAceptacionCTPostgreSQL)(nil)
