package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// pool usa solo el LOGIN ejecutor para GET V3 y publicación. calculo usa un
// LOGIN distinto que hereda exclusivamente el rol NOLOGIN calculador B51.
type RepositorioPoliticaOfertasPostgreSQL struct {
	pool    *pgxpool.Pool
	calculo *pgxpool.Pool
}

func NuevoRepositorioPoliticaOfertasPostgreSQL(pool, calculo *pgxpool.Pool) (*RepositorioPoliticaOfertasPostgreSQL, error) {
	if pool == nil || calculo == nil || pool == calculo {
		return nil, ports.ErrPoliticaOfertasNoDisponible
	}
	return &RepositorioPoliticaOfertasPostgreSQL{pool: pool, calculo: calculo}, nil
}

func (r *RepositorioPoliticaOfertasPostgreSQL) Vigente(ctx context.Context, bolsa string) (ports.VersionPoliticaOfertas, error) {
	if r == nil || r.calculo == nil || ctx == nil || bolsa == "" {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	var salida []byte
	if err := r.calculo.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.leer_politica_ofertas_v1($1)`, bolsa).Scan(&salida); err != nil {
		return ports.VersionPoliticaOfertas{}, errorPoliticaOfertas(err)
	}
	return decodificarPoliticaOfertas(salida, false)
}

func (r *RepositorioPoliticaOfertasPostgreSQL) ConsultarAutorizada(ctx context.Context, c ports.ConsultaPoliticaOfertasAutorizada) (ports.VersionPoliticaOfertas, error) {
	if r == nil || r.pool == nil || ctx == nil || c.BolsaRef == "" || c.Material.ValidarEstructura() != nil {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.VersionPoliticaOfertas{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.VersionPoliticaOfertas{}, errorPoliticaOfertas(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return ports.VersionPoliticaOfertas{}, errorPoliticaOfertas(err)
	}
	m := c.Material
	var salida []byte
	if err := tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.consultar_politica_ofertas_v2(
		$1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`,
		c.BolsaRef, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&salida); err != nil {
		return ports.VersionPoliticaOfertas{}, errorPoliticaOfertas(err)
	}
	v, err := decodificarPoliticaOfertas(salida, false)
	if err != nil {
		return ports.VersionPoliticaOfertas{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.VersionPoliticaOfertas{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.VersionPoliticaOfertas{}, errorPoliticaOfertas(err)
	}
	return v, nil
}

func (r *RepositorioPoliticaOfertasPostgreSQL) Publicar(ctx context.Context, c ports.ComandoPublicarPoliticaOfertas) (ports.VersionPoliticaOfertas, error) {
	if r == nil || r.pool == nil || ctx == nil || c.Material.ValidarEstructura() != nil {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	datos, err := json.Marshal(c.Politica)
	if err != nil {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	m := c.Material
	var salida []byte
	var reutilizada bool
	err = r.pool.QueryRow(ctx, `SELECT politica,reutilizada FROM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
		$1,$2,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11::numeric,$12::numeric,$13,$14,$15,$16)`,
		c.BolsaRef, c.VersionEsperada, datos, c.ActorRef, c.ClaveIdempotencia, c.ReciboRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&salida, &reutilizada)
	if err != nil {
		return ports.VersionPoliticaOfertas{}, errorPoliticaOfertas(err)
	}
	return decodificarPoliticaOfertas(salida, reutilizada)
}

func decodificarPoliticaOfertas(salida []byte, reutilizada bool) (ports.VersionPoliticaOfertas, error) {
	var v ports.VersionPoliticaOfertas
	if json.Unmarshal(salida, &v) != nil || v.BolsaRef == "" ||
		(v.Version > 0 && (!v.Configurada || v.Politica == nil || v.Politica.Validar() != nil)) {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	v.Reutilizada = reutilizada
	return v, nil
}

func errorPoliticaOfertas(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "VBP01":
			return ports.ErrPoliticaOfertasConflicto
		case "22023", "23503":
			return ports.ErrPoliticaOfertasNoDisponible
		}
	}
	return ports.ErrPoliticaOfertasNoDisponible
}
