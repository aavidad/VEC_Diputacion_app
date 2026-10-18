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

// RepositorioResolucionIntencionOfertaCTPostgreSQL invoca B121 con V3 fresca.
// Una respuesta incierta nunca se traduce en ausencia ni reintenta DML.
type RepositorioResolucionIntencionOfertaCTPostgreSQL struct{ pool iniciadorTransacciones }

var _ ports.RepositorioResolucionIntencionOfertaCT = (*RepositorioResolucionIntencionOfertaCTPostgreSQL)(nil)

func NuevoRepositorioResolucionIntencionOfertaCTPostgreSQL(pool *pgxpool.Pool) (*RepositorioResolucionIntencionOfertaCTPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrCierreIntencionOfertaCTNoDisponible
	}
	return nuevoRepositorioResolucionIntencionOfertaCTPostgreSQL(pool)
}

func nuevoRepositorioResolucionIntencionOfertaCTPostgreSQL(pool iniciadorTransacciones) (*RepositorioResolucionIntencionOfertaCTPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrCierreIntencionOfertaCTNoDisponible
	}
	return &RepositorioResolucionIntencionOfertaCTPostgreSQL{pool: pool}, nil
}

func (r *RepositorioResolucionIntencionOfertaCTPostgreSQL) Resolver(ctx context.Context,
	c ports.ComandoResolverIntencionOfertaCT,
) (ports.ResultadoResolverIntencionOfertaCT, error) {
	var vacio ports.ResultadoResolverIntencionOfertaCT
	if r == nil || r.pool == nil || ctx == nil || c.Validar() != nil {
		return vacio, ports.ErrCierreIntencionOfertaCTInvalido
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorResolucionIntencionOfertaCT(ctx, err)
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, errorResolucionIntencionOfertaCT(ctx, err)
	}
	m := c.Material
	var situacion string
	var ofertaRaw, cierreRaw []byte
	err = tx.QueryRow(ctx, `SELECT resultado,oferta,cierre FROM vec_bolsa_llamamientos.recuperar_o_cerrar_intencion_oferta_ct_v1(
		$1::jsonb,$2::text,$3::text,$4::text,$5::text,$6::bytea,$7::bytea,$8::bytea,
		$9::bytea,$10::numeric,$11::numeric,$12::bytea,$13::bytea,$14::bytea,$15::bytea)`,
		[]byte(c.Comando), c.ActorRef, c.MaterialCanonico, c.UnidadRef, c.AmbitoRef,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&situacion, &ofertaRaw, &cierreRaw)
	if err != nil {
		return vacio, errorResolucionIntencionOfertaCT(ctx, err)
	}
	resultado, err := decodificarResolucionIntencionOfertaCT(situacion, ofertaRaw, cierreRaw)
	if err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, ports.ErrCierreIntencionOfertaCTIncierta
	}
	return resultado, nil
}

func decodificarResolucionIntencionOfertaCT(situacion string, ofertaRaw, cierreRaw []byte) (ports.ResultadoResolverIntencionOfertaCT, error) {
	var vacio ports.ResultadoResolverIntencionOfertaCT
	switch situacion {
	case ports.ResultadoIntencionOfertaPublicada:
		if len(cierreRaw) != 0 && string(cierreRaw) != "null" {
			return vacio, ports.ErrCierreIntencionOfertaCTNoDisponible
		}
		oferta, err := decodificarOferta(ofertaRaw, true)
		if err != nil {
			return vacio, err
		}
		return ports.ResultadoResolverIntencionOfertaCT{Situacion: situacion, Oferta: &oferta}, nil
	case ports.ResultadoIntencionOfertaCerrada:
		if len(ofertaRaw) != 0 && string(ofertaRaw) != "null" ||
			len(cierreRaw) == 0 || len(cierreRaw) > 65536 || !json.Valid(cierreRaw) {
			return vacio, ports.ErrCierreIntencionOfertaCTNoDisponible
		}
		var cabecera struct {
			Esquema string `json:"esquema"`
			Estado  string `json:"estado"`
		}
		if json.Unmarshal(cierreRaw, &cabecera) != nil ||
			cabecera.Esquema != "vec.bolsa.oferta-ct.cierre.v1" ||
			cabecera.Estado != "cerrada_sin_publicar" {
			return vacio, ports.ErrCierreIntencionOfertaCTNoDisponible
		}
		return ports.ResultadoResolverIntencionOfertaCT{Situacion: situacion,
			Cierre: append(json.RawMessage(nil), cierreRaw...)}, nil
	default:
		return vacio, ports.ErrCierreIntencionOfertaCTNoDisponible
	}
}

func errorResolucionIntencionOfertaCT(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505", "40001", "40P01", "55P03":
			return ports.ErrCierreIntencionOfertaCTIncierta
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "22023":
			return ports.ErrCierreIntencionOfertaCTInvalido
		case "VBO01":
			return ports.ErrOfertaConflicto
		}
	}
	return ports.ErrCierreIntencionOfertaCTNoDisponible
}
