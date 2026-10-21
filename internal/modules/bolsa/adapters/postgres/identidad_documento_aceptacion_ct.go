package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const funcionIdentidadDocumentoAceptacionCT = `SELECT vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const ajustesIdentidadDocumentoAceptacionCT = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

var errRollbackIdentidadDocumentoAceptacionCT = errors.New("bolsa: rollback de identidad documental fallido")

type RepositorioIdentidadDocumentoAceptacionCTPostgreSQL struct {
	pool  iniciadorPersonaAceptacionCT
	ahora func() time.Time
}

func NuevoRepositorioIdentidadDocumentoAceptacionCTPostgreSQL(pool *pgxpool.Pool, ahora func() time.Time) (*RepositorioIdentidadDocumentoAceptacionCTPostgreSQL, error) {
	if pool == nil || ahora == nil {
		return nil, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	return &RepositorioIdentidadDocumentoAceptacionCTPostgreSQL{pool, ahora}, nil
}

func (r *RepositorioIdentidadDocumentoAceptacionCTPostgreSQL) ConsultarFuenteIdentidadDocumentoAceptacionCT(ctx context.Context, o ports.OrdenIdentidadDocumentoAceptacionCT) (ports.LecturaFuenteIdentidadDocumentoAceptacionCT, error) {
	var cero ports.LecturaFuenteIdentidadDocumentoAceptacionCT
	if ctx == nil || r == nil || r.pool == nil || r.ahora == nil {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	for intento := 0; intento < 3; intento++ {
		if err := ctx.Err(); err != nil {
			return cero, err
		}
		if err := application.ValidarOrdenIdentidadDocumentoAceptacionCT(o, r.ahora().UTC().Truncate(time.Microsecond)); err != nil {
			return cero, err
		}
		p, err := application.PrepararIdentidadDocumentoAceptacionCT(o.Solicitud)
		if err != nil {
			return cero, err
		}
		lectura, err := r.consultar(ctx, o, p.MaterialCanonico)
		if err == nil {
			return lectura, nil
		}
		if errors.Is(err, errRollbackIdentidadDocumentoAceptacionCT) {
			return cero, err
		}
		if !serializacionPersonaAceptacionCT(err) {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "42501" {
				return cero, ports.ErrIdentidadDocumentoAceptacionCTDenegada
			}
			return cero, err
		}
	}
	return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
}

func (r *RepositorioIdentidadDocumentoAceptacionCTPostgreSQL) consultar(ctx context.Context, o ports.OrdenIdentidadDocumentoAceptacionCT, canon []byte) (out ports.LecturaFuenteIdentidadDocumentoAceptacionCT, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return out, err
	}
	confirmado := false
	defer func() {
		if confirmado {
			return
		}
		fin, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		if fallo := tx.Rollback(fin); fallo != nil && !errors.Is(fallo, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("%w: %w", errRollbackIdentidadDocumentoAceptacionCT, fallo))
		}
	}()
	if _, err = tx.Exec(ctx, ajustesIdentidadDocumentoAceptacionCT); err != nil {
		return out, err
	}
	m := o.Material
	var b []byte
	if err = tx.QueryRow(ctx, funcionIdentidadDocumentoAceptacionCT, string(canon), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&b); err != nil {
		return out, err
	}
	if len(b) == 0 || len(b) > 8<<10 {
		return out, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var lectura ports.LecturaFuenteIdentidadDocumentoAceptacionCT
	if d.Decode(&lectura) != nil {
		return out, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	var extra any
	if !errors.Is(d.Decode(&extra), io.EOF) || application.ValidarLecturaFuenteIdentidadDocumentoAceptacionCT(o, lectura, r.ahora().UTC().Truncate(time.Microsecond)) != nil {
		return out, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	confirmado = true
	return lectura, nil
}

var _ ports.RepositorioIdentidadDocumentoAceptacionCT = (*RepositorioIdentidadDocumentoAceptacionCTPostgreSQL)(nil)
