package main

import (
	"context"
	"crypto/tls"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type conexionPrivada struct {
	DSN                      string `json:"dsn"`
	PermitirSocketDesarrollo bool   `json:"permitir_socket_desarrollo"`
}
type transaccion interface {
	PrepararUTC(context.Context) error
	Provisionar(context.Context, string, string, string) ([]byte, error)
	Confirmar(context.Context) error
	Cerrar(context.Context)
}
type abrirTransaccion func(context.Context, conexionPrivada, time.Duration) (transaccion, error)

var errConexion = errors.New("conexion_no_disponible")
var errCommit = errors.New("commit_no_confirmado")

// ejecutarOperacion confirma también los envelopes denegado/error: su auditoría
// existe únicamente tras COMMIT. Nunca reintenta una operación automáticamente.
func ejecutarOperacion(ctx context.Context, cfg conexionPrivada, limite time.Duration, canon []byte, aprobacion string, fuente []byte, plan documento, abrir abrirTransaccion) ([]byte, envoltura, error) {
	var cero envoltura
	tx, err := abrir(ctx, cfg, limite)
	if err != nil {
		return nil, cero, errConexion
	}
	defer func() {
		cierre, cancelar := context.WithTimeout(context.Background(), limite)
		defer cancelar()
		tx.Cerrar(cierre)
	}()
	if tx.PrepararUTC(ctx) != nil {
		return nil, cero, errConexion
	}
	b, err := tx.Provisionar(ctx, string(canon), aprobacion, string(fuente))
	if err != nil {
		return nil, cero, errConexion
	}
	e, err := validarEnvoltura(b, plan.Plan, plan.HuellaPlanSHA256)
	if err != nil {
		clear(b)
		return nil, cero, errEnvoltura
	}
	if tx.Confirmar(ctx) != nil {
		clear(b)
		return nil, cero, errCommit
	}
	return b, e, nil
}

type transaccionPG struct {
	tx       pgx.Tx
	conexion *pgx.Conn
}

func (t *transaccionPG) PrepararUTC(ctx context.Context) error {
	_, e := t.tx.Exec(ctx, "SET LOCAL TIME ZONE 'UTC'")
	return e
}
func (t *transaccionPG) Provisionar(ctx context.Context, canon, sha, fuente string) ([]byte, error) {
	var b []byte
	err := t.tx.QueryRow(ctx, "SELECT vec_personal.inicializar_unidad_sintetica_admin_v1($1::text,$2::text,$3::text)", canon, sha, fuente).Scan(&b)
	return b, err
}
func (t *transaccionPG) Confirmar(ctx context.Context) error { return t.tx.Commit(ctx) }
func (t *transaccionPG) Cerrar(ctx context.Context) {
	_ = t.tx.Rollback(ctx)
	_ = t.conexion.Close(ctx)
}
func nuevaTransaccionPG(ctx context.Context, cfg conexionPrivada, limite time.Duration) (transaccion, error) {
	pc, err := pgx.ParseConfig(cfg.DSN)
	if err != nil || !canalValido(&pc.Config, cfg.PermitirSocketDesarrollo) {
		return nil, errConexion
	}
	// No hay elevación de rol ni opciones SQL libres en la composición CLI.
	if _, ok := pc.RuntimeParams["role"]; ok {
		return nil, errConexion
	}
	if _, ok := pc.RuntimeParams["options"]; ok {
		return nil, errConexion
	}
	pc.ConnectTimeout = limite
	conn, err := pgx.ConnectConfig(ctx, pc)
	if err != nil {
		return nil, errConexion
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		_ = conn.Close(ctx)
		return nil, errConexion
	}
	return &transaccionPG{tx: tx, conexion: conn}, nil
}
func canalValido(cfg *pgconn.Config, socket bool) bool {
	if cfg == nil || cfg.User == "" || cfg.Database == "" || !destinoValido(cfg.Host, cfg.TLSConfig, socket) {
		return false
	}
	for _, f := range cfg.Fallbacks {
		if f == nil || !destinoValido(f.Host, f.TLSConfig, socket) {
			return false
		}
	}
	return true
}
func destinoValido(host string, tlsCfg *tls.Config, socket bool) bool {
	if strings.HasPrefix(host, "/") {
		return socket && filepath.IsAbs(host) && filepath.Clean(host) == host
	}
	return host != "" && tlsCfg != nil && !tlsCfg.InsecureSkipVerify && tlsCfg.ServerName != ""
}
