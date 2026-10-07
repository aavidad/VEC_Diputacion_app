package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

const consultaHechoPropio = `SELECT vec_meritos.consultar_hecho_propio_v1($1::bytea,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

// Consulta confirma consumo V3, lectura actual, auditoría y recibo en una
// transacción. Cada acceso requiere material nuevo; no recupera una consulta.
type Consulta struct{ pool iniciador }

var _ ports.RepositorioConsultaPropia = (*Consulta)(nil)

func NuevaConsulta(pool *pgxpool.Pool) (*Consulta, error) { return nuevaConsulta(pool) }

func nuevaConsulta(pool iniciador) (*Consulta, error) {
	if nulo(pool) {
		return nil, ports.ErrConsultaNoDisponible
	}
	return &Consulta{pool: pool}, nil
}

func (r *Consulta) ConsultarActual(ctx context.Context, orden ports.OrdenConsultaPropia) (ports.ResultadoConsultaPropia, error) {
	if ctx == nil || r == nil || nulo(r.pool) {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	selector, err := serializarConsultaPropia(orden)
	if err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nulo(tx) {
		return ports.ResultadoConsultaPropia{}, errorConsultaSQL(ctx, err)
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return ports.ResultadoConsultaPropia{}, errorConsultaSQL(ctx, err)
	}
	m := orden.Autorizacion.Material
	var raw []byte
	err = tx.QueryRow(ctx, consultaHechoPropio, selector, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&raw)
	if err != nil {
		return ports.ResultadoConsultaPropia{}, errorConsultaSQL(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	resultado, err := leerResultadoConsultaPropia(raw)
	if err != nil || application.ValidarResultadoConsultaPropia(orden, resultado) != nil {
		return ports.ResultadoConsultaPropia{}, ports.ErrConsultaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ResultadoConsultaPropia{}, errorConsultaSQL(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaPropia{}, err
	}
	return resultado, nil
}

func errorConsultaSQL(ctx context.Context, err error) error {
	traducido := errorSQL(ctx, err)
	if errors.Is(traducido, ports.ErrRegistroNoDisponible) {
		return ports.ErrConsultaNoDisponible
	}
	return traducido
}
