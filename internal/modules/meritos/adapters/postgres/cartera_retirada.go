package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
)

const retirarHechoCartera = `SELECT vec_meritos.retirar_hecho_propio_v1(
$1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,
$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

type RetiradaCartera struct{ pool iniciador }

var _ ports.RepositorioRetiradaCartera = (*RetiradaCartera)(nil)

func NuevaRetiradaCartera(pool *pgxpool.Pool) (*RetiradaCartera, error) {
	if nulo(pool) {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &RetiradaCartera{pool: pool}, nil
}

func (r *RetiradaCartera) RetirarCartera(ctx context.Context,
	orden ports.OrdenRetiradaCartera) (ports.ResultadoRetiradaCartera, error) {
	vacio := ports.ResultadoRetiradaCartera{}
	if ctx == nil || r == nil || nulo(r.pool) || !domain.ReferenciaValida(orden.PersonaRef) ||
		!domain.ReferenciaValida(orden.HechoRef) || orden.HechoVersionEsperada < 1 ||
		orden.ClaveIdempotencia == "" || !huellaSHA256CarteraSQL(orden.HuellaComandoSHA256) ||
		len(orden.ComandoCanonico) == 0 || len(orden.ComandoCanonico) > 65536 ||
		orden.Material.ValidarEstructura() != nil {
		return vacio, ports.ErrCarteraNoDisponible
	}
	var resultado ports.ResultadoRetiradaCartera
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var fallo error
		resultado, fallo = r.retirarIntento(ctx, orden)
		return fallo
	})
	if err != nil {
		return vacio, errorCarteraSQL(ctx, err)
	}
	return resultado, nil
}

func (r *RetiradaCartera) retirarIntento(ctx context.Context,
	o ports.OrdenRetiradaCartera) (ports.ResultadoRetiradaCartera, error) {
	vacia := ports.ResultadoRetiradaCartera{}
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
	m := o.Material
	var raw []byte
	err = tx.QueryRow(ctx, retirarHechoCartera, o.ComandoCanonico,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(),
		m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(),
		m.RaizPublicaSPKI()).Scan(&raw)
	if err != nil {
		return vacia, err
	}
	defer borrarCarteraRaw(raw)
	resultado, err := decodificarRetiradaCartera(raw)
	if err != nil || !resultadoRetiradaCarteraValido(o, resultado) {
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

func decodificarRetiradaCartera(raw []byte) (ports.ResultadoRetiradaCartera, error) {
	vacio := ports.ResultadoRetiradaCartera{}
	if len(raw) == 0 || len(raw) > 65536 {
		return vacio, ports.ErrCarteraNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var resultado ports.ResultadoRetiradaCartera
	if err := d.Decode(&resultado); err != nil {
		return vacio, errors.Join(ports.ErrCarteraNoDisponible, err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		if err != nil {
			return vacio, errors.Join(ports.ErrCarteraNoDisponible, err)
		}
		return vacio, ports.ErrCarteraNoDisponible
	}
	return resultado, nil
}

func resultadoRetiradaCarteraValido(o ports.OrdenRetiradaCartera,
	r ports.ResultadoRetiradaCartera) bool {
	if !domain.ReferenciaValida(r.AuditoriaRef) {
		return false
	}
	if r.Codigo == "conflicto" {
		return r.Recibo == nil
	}
	if r.Codigo != "confirmada" && r.Codigo != "repetida" || r.Recibo == nil {
		return false
	}
	v := r.Recibo
	return domain.ReferenciaValida(v.Referencia) && v.HechoRef == o.HechoRef &&
		v.VersionHecho == o.HechoVersionEsperada &&
		v.Disponibilidad.Estado == "retirado" &&
		v.Disponibilidad.Version == o.DisponibilidadVersionEsperada+1 &&
		!v.RetiradoEn.IsZero() && domain.ReferenciaValida(v.AuditoriaRef) &&
		domain.ReferenciaValida(v.EventoRef) &&
		v.ClaveIdempotencia == o.ClaveIdempotencia &&
		v.HuellaComandoSHA256 == o.HuellaComandoSHA256
}
