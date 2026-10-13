package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
)

const consultaCarteraPropia = `SELECT vec_meritos.listar_cartera_propia_v1($1::bytea,$2::bytea,$3::jsonb)`

type LecturaCartera struct{ pool iniciador }

var _ ports.LectorCartera = (*LecturaCartera)(nil)

func NuevaLecturaCartera(pool *pgxpool.Pool) (*LecturaCartera, error) {
	if nulo(pool) {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &LecturaCartera{pool: pool}, nil
}

func (r *LecturaCartera) LeerCartera(ctx context.Context,
	captura ports.CapturaLecturaCartera) (ports.PaginaCarteraInterna, error) {
	vacia := ports.PaginaCarteraInterna{}
	if ctx == nil || r == nil || nulo(r.pool) || captura.PersonaRef() == "" ||
		!captura.ValidaHasta().After(time.Now().UTC()) {
		return vacia, ports.ErrCarteraNoDisponible
	}
	orden, err := selectorCartera(captura.OrdenJSON())
	if err != nil || orden.PersonaRef != captura.PersonaRef() ||
		orden.CorrelacionRef != captura.CorrelacionRef() {
		return vacia, ports.ErrCarteraNoDisponible
	}
	var resultado ports.PaginaCarteraInterna
	err = postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var fallo error
		resultado, fallo = r.leerIntento(ctx, captura, orden)
		return fallo
	})
	if err != nil {
		return vacia, errorCarteraSQL(ctx, err)
	}
	return resultado, nil
}

type selectorCarteraJSON struct {
	PersonaRef     string    `json:"persona_ref"`
	CorrelacionRef string    `json:"correlacion_ref"`
	Limite         int       `json:"limite"`
	Filtro         string    `json:"filtro"`
	Idioma         string    `json:"idioma"`
	CursorHechoRef string    `json:"cursor_hecho_ref"`
	CorteEn        time.Time `json:"corte_en"`
}

func selectorCartera(raw []byte) (selectorCarteraJSON, error) {
	var p selectorCarteraJSON
	if len(raw) == 0 || len(raw) > 16384 {
		return p, ports.ErrCarteraNoDisponible
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, errors.Join(ports.ErrCarteraNoDisponible, err)
	}
	if p.PersonaRef == "" || p.CorrelacionRef == "" || p.Limite < 1 || p.Limite > 50 ||
		p.CorteEn.IsZero() || p.Filtro != "todos" && p.Filtro != "disponibles" && p.Filtro != "retirados" ||
		p.Idioma != "es" && p.Idioma != "en" {
		return p, ports.ErrCarteraNoDisponible
	}
	return p, nil
}

func (r *LecturaCartera) leerIntento(ctx context.Context, c ports.CapturaLecturaCartera,
	orden selectorCarteraJSON) (ports.PaginaCarteraInterna, error) {
	vacia := ports.PaginaCarteraInterna{}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, err
	}
	if nulo(tx) {
		return vacia, ports.ErrCarteraNoDisponible
	}
	defer revertir(tx)
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
set_config('row_security','on',true),set_config('timezone','UTC',true),
set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacia, err
	}
	var raw []byte
	err = tx.QueryRow(ctx, consultaCarteraPropia, c.ContextoCanonico(), c.VinculoCanonico(), c.OrdenJSON()).Scan(&raw)
	if err != nil {
		return vacia, err
	}
	defer borrarCarteraRaw(raw)
	resultado, err := decodificarPaginaCartera(raw)
	if err != nil || resultado.Validar(orden.CorteEn, orden.CorrelacionRef, orden.Limite,
		orden.Filtro, orden.CursorHechoRef) != nil {
		return vacia, ports.ErrCarteraNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacia, err
	}
	return resultado, nil
}

func decodificarPaginaCartera(raw []byte) (ports.PaginaCarteraInterna, error) {
	vacia := ports.PaginaCarteraInterna{}
	if len(raw) == 0 || len(raw) > 262144 {
		return vacia, ports.ErrCarteraNoDisponible
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var pagina ports.PaginaCarteraInterna
	if err := decoder.Decode(&pagina); err != nil {
		return vacia, errors.Join(ports.ErrCarteraNoDisponible, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err != nil {
			return vacia, errors.Join(ports.ErrCarteraNoDisponible, err)
		}
		return vacia, ports.ErrCarteraNoDisponible
	}
	return pagina, nil
}

func borrarCarteraRaw(raw []byte) {
	for i := range raw {
		raw[i] = 0
	}
}

type falloCarteraSQL struct{ causa error }

func (falloCarteraSQL) Error() string { return ports.ErrCarteraNoDisponible.Error() }
func (e falloCarteraSQL) Unwrap() []error {
	return []error{ports.ErrCarteraNoDisponible, e.causa}
}

func errorCarteraSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return ports.ErrCarteraDenegada
	}
	if err == nil {
		return ports.ErrCarteraNoDisponible
	}
	return falloCarteraSQL{causa: err}
}
