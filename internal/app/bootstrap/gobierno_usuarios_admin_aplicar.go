package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfirmacionGobiernoUsuariosAdmin sólo se entrega después de COMMIT. Un
// fallo/COMMIT indeterminado no autoriza repetir con plan o material nuevos.
type ConfirmacionGobiernoUsuariosAdmin struct {
	Estado           string          `json:"estado"`
	Codigo           string          `json:"codigo"`
	Recibo           json.RawMessage `json:"recibo"`
	AuditoriaIntento json.RawMessage `json:"auditoria_intento"`
}

func AplicarGobiernoUsuariosAdmin(ctx context.Context, pool *pgxpool.Pool, planCanonico, shaAprobado string, material *MaterialUsuariosAdmin) (ConfirmacionGobiernoUsuariosAdmin, error) {
	var vacio ConfirmacionGobiernoUsuariosAdmin
	if ctx == nil || ctx.Err() != nil || pool == nil || material == nil || len(planCanonico) > 16384 || len(shaAprobado) != 64 {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	var secreto bytes.Buffer
	defer func() { borrarBytes(secreto.Bytes()); secreto.Reset() }()
	if material.EscribirMaterialPrivado(&secreto) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	defer func() {
		c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(c)
	}()
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	var raw []byte
	if err = tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1($1::text,$2::text,$3::text)`, planCanonico, shaAprobado, secreto.String()).Scan(&raw); err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	var r ConfirmacionGobiernoUsuariosAdmin
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&r) != nil || len(r.AuditoriaIntento) == 0 || bytes.Equal(r.AuditoriaIntento, []byte("null")) {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	switch r.Estado {
	case "permitido":
		if r.Codigo != "gobierno_usuarios_registrado" && r.Codigo != "gobierno_usuarios_replay" {
			return vacio, ErrGobiernoUsuariosAdmin
		}
		if len(r.Recibo) == 0 || bytes.Equal(r.Recibo, []byte("null")) {
			return vacio, ErrGobiernoUsuariosAdmin
		}
	case "denegado":
		if r.Codigo != "gobierno_usuarios_denegado" {
			return vacio, ErrGobiernoUsuariosAdmin
		}
	case "error":
		if r.Codigo != "gobierno_usuarios_error" {
			return vacio, ErrGobiernoUsuariosAdmin
		}
	default:
		return vacio, ErrGobiernoUsuariosAdmin
	}
	if tx.Commit(ctx) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	return r, nil
}
