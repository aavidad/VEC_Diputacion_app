package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const consultaActoSubidaOriginalC8 = `SELECT vec_cronos_v1.resolver_acto_subida_documental_c8_v1($1,$2,$3,$4,$5)`

// Este pool usa exclusivamente el LOGIN vec_interno_documentos_cronos_datos;
// CRN22 comprueba session_user y la membresía en cada consulta. El adaptador
// no cambia de ROLE ni comparte el pool de peticiones Cronos.
type FuenteActoSubidaOriginalC8 struct{ db *pgxpool.Pool }

func NuevaFuenteActoSubidaOriginalC8(db *pgxpool.Pool) (*FuenteActoSubidaOriginalC8, error) {
	if db == nil {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &FuenteActoSubidaOriginalC8{db: db}, nil
}

func (f *FuenteActoSubidaOriginalC8) ConsultarActoSubidaOriginal(ctx context.Context,
	selector ports.SelectorSubidaOriginalC8, captura ports.CapturaLecturaOriginalC8) (ports.EstadoPropietarioSubidaOriginalC8, error) {
	var cero ports.EstadoPropietarioSubidaOriginalC8
	if f == nil || f.db == nil || ctx == nil || ctx.Err() != nil ||
		!domain.SolicitudPermisoRefValida(selector.SolicitudRef) || selector.VersionEsperada < 1 ||
		selector.VersionEsperada > 1000 || len(captura.ContextoCanonico) == 0 ||
		len(captura.ContextoCanonico) > 65536 || len(captura.VinculoCanonico) == 0 ||
		len(captura.VinculoCanonico) > 16384 || len(captura.Orden) == 0 || len(captura.Orden) > 8192 {
		return cero, ports.ErrJustificacionNoDisponible
	}
	correlacion, ok := capturaOriginalC8Valida(captura,
		"cronos.justificacion.subida.preparar", selector.SolicitudRef)
	if !ok {
		return cero, ports.ErrJustificacionNoDisponible
	}
	tx, err := f.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, ports.ErrJustificacionNoDisponible
	}
	defer func() {
		rollback, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollback)
	}()
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('timezone','UTC',true),
    set_config('row_security','on',true),set_config('lock_timeout','2s',true),
    set_config('statement_timeout','10s',true),set_config('idle_in_transaction_session_timeout','15s',true)`); err != nil {
		return cero, ports.ErrJustificacionNoDisponible
	}
	var raw []byte
	if err := tx.QueryRow(ctx, consultaActoSubidaOriginalC8, selector.SolicitudRef,
		selector.VersionEsperada, captura.ContextoCanonico, captura.VinculoCanonico, string(captura.Orden)).Scan(&raw); err != nil {
		return cero, ports.ErrJustificacionNoDisponible
	}
	defer clear(raw)
	if len(raw) == 0 || len(raw) > 8192 {
		return cero, ports.ErrJustificacionNoDisponible
	}
	estado, err := decodificarActoSubidaOriginalC8(raw, selector)
	if err != nil {
		return cero, err
	}
	if !auditoriaOriginalC8Valida(estado.Auditoria, correlacion) {
		return cero, ports.ErrJustificacionNoDisponible
	}
	if tx.Commit(ctx) != nil {
		return cero, ports.ErrJustificacionNoDisponible
	}
	return estado, nil
}

func decodificarActoSubidaOriginalC8(raw []byte, selector ports.SelectorSubidaOriginalC8) (ports.EstadoPropietarioSubidaOriginalC8, error) {
	var cero ports.EstadoPropietarioSubidaOriginalC8
	var campos map[string]json.RawMessage
	var estado ports.EstadoPropietarioSubidaOriginalC8
	if json.Unmarshal(raw, &campos) != nil || len(campos) != 5 ||
		json.Unmarshal(raw, &estado) != nil || estado.SolicitudRef != selector.SolicitudRef ||
		estado.Version != selector.VersionEsperada || estado.PersonaRef == "" ||
		estado.EmpleadoRef == "" || estado.Auditoria.Referencia == "" ||
		estado.Auditoria.HuellaSHA256 == "" || estado.Auditoria.CorrelacionRef == "" ||
		estado.Auditoria.RegistradaEn == "" {
		return cero, ports.ErrJustificacionNoDisponible
	}
	for _, clave := range []string{"solicitud_ref", "version", "persona_ref", "empleado_ref", "auditoria"} {
		if _, ok := campos[clave]; !ok {
			return cero, ports.ErrJustificacionNoDisponible
		}
	}
	return estado, nil
}

var _ ports.FuenteActoSubidaOriginalC8 = (*FuenteActoSubidaOriginalC8)(nil)
