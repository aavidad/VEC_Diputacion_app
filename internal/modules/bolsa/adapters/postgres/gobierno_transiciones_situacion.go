package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type RepositorioGobiernoTransicionesSituacionPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoRepositorioGobiernoTransicionesSituacionPostgreSQL(pool *pgxpool.Pool) (*RepositorioGobiernoTransicionesSituacionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrGobiernoTransicionesNoDisponible
	}
	return &RepositorioGobiernoTransicionesSituacionPostgreSQL{pool: pool}, nil
}

var _ ports.RepositorioGobiernoTransicionesSituacion = (*RepositorioGobiernoTransicionesSituacionPostgreSQL)(nil)

func (r *RepositorioGobiernoTransicionesSituacionPostgreSQL) Leer(ctx context.Context,
	version int, sha string) (ports.GobiernoTransicionesSituacion, error) {
	if r == nil || r.pool == nil || ctx == nil || version < 1 || len(sha) != 64 {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	var datos []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1($1,$2)`,
		version, sha).Scan(&datos); err != nil {
		return ports.GobiernoTransicionesSituacion{}, errorGobiernoTransicionesSituacion(err)
	}
	return decodificarGobiernoTransicionesSituacion(datos, false)
}

func (r *RepositorioGobiernoTransicionesSituacionPostgreSQL) Publicar(ctx context.Context,
	c ports.ComandoGobiernoTransicionesSituacion) (ports.GobiernoTransicionesSituacion, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.GobiernoTransicionesSituacion{}, errorGobiernoTransicionesSituacion(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
	 set_config('row_security','on',true),set_config('timezone','UTC',true),
	 set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
	 set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return ports.GobiernoTransicionesSituacion{}, errorGobiernoTransicionesSituacion(err)
	}
	m := c.Material
	var datos []byte
	var reutilizada bool
	err = tx.QueryRow(ctx, `SELECT politica,reutilizada FROM vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(
	 $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::numeric,$14::numeric,$15,$16,$17,$18)`,
		c.VersionEsperada, c.Transiciones, c.Motivo, c.ClaveIdempotencia, c.ReciboRef, c.ActorRef,
		c.CatalogoVersion, c.CatalogoSHA256,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&datos, &reutilizada)
	if err != nil {
		return ports.GobiernoTransicionesSituacion{}, errorGobiernoTransicionesSituacion(err)
	}
	v, err := decodificarGobiernoTransicionesSituacion(datos, reutilizada)
	if err != nil {
		return v, err
	}
	if v.ReciboRef != c.ReciboRef || !slices.Equal(v.TransicionesGuardadas, c.Transiciones) {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.GobiernoTransicionesSituacion{}, errorGobiernoTransicionesSituacion(err)
	}
	return v, nil
}

func decodificarGobiernoTransicionesSituacion(datos []byte, reutilizada bool) (ports.GobiernoTransicionesSituacion, error) {
	var v ports.GobiernoTransicionesSituacion
	if json.Unmarshal(datos, &v) != nil || v.Version < 1 || v.Limites.CatalogoVersion < 1 ||
		len(v.Limites.CatalogoSHA256) != 64 || len(v.TransicionesGuardadas) == 0 || len(v.Limites.Transiciones) == 0 {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesNoDisponible
	}
	v.Reutilizada = reutilizada
	return v, nil
}

func errorGobiernoTransicionesSituacion(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "VBT01":
			return ports.ErrGobiernoTransicionesConflicto
		case "22023":
			return ports.ErrGobiernoTransicionesInvalido
		}
	}
	return ports.ErrGobiernoTransicionesNoDisponible
}
