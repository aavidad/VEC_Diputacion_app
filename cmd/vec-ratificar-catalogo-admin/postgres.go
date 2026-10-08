package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"
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
	Preparar(context.Context) (string, error)
	CanonYDescriptores(context.Context, string) (string, string, error)
	Ratificar(context.Context, string, string) ([]byte, error)
	Confirmar(context.Context) error
	Cerrar(context.Context)
}

type abrirTransaccion func(context.Context, conexionPrivada) (transaccion, error)

var errOperacion = errors.New("ratificacion_no_confirmada")
var errCommit = errors.New("commit_indeterminado")

type dsnNoValido struct{ causa error }

func (e dsnNoValido) Error() string { return errOperacion.Error() }
func (e dsnNoValido) Unwrap() error { return e.causa }

func ejecutarOperacion(ctx context.Context, cfg conexionPrivada, plan []byte, sha string, p documentoPlan, a aprobacionPrivada, abrir abrirTransaccion) (resultadoRatificacion, error) {
	tx, err := abrir(ctx, cfg)
	if err != nil {
		return resultadoRatificacion{}, errOperacion
	}
	defer func() {
		cierre, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		tx.Cerrar(cierre)
	}()
	operador, err := tx.Preparar(ctx)
	if err != nil || operador == "" {
		return resultadoRatificacion{}, errOperacion
	}
	canon, descriptoresSHA, err := tx.CanonYDescriptores(ctx, string(plan))
	if err != nil || canon != string(plan) || !hashValido(descriptoresSHA) {
		return resultadoRatificacion{}, errOperacion
	}
	b, err := tx.Ratificar(ctx, string(plan), sha)
	if err != nil {
		return resultadoRatificacion{}, errOperacion
	}
	resultado, err := validarRespuesta(b, p, sha, descriptoresSHA, a, operador)
	if err != nil {
		return resultadoRatificacion{}, errOperacion
	}
	if tx.Confirmar(ctx) != nil {
		return resultadoRatificacion{}, errCommit
	}
	return resultado, nil
}

type transaccionPG struct {
	tx       pgx.Tx
	conexion *pgx.Conn
}

func (t *transaccionPG) Preparar(ctx context.Context) (string, error) {
	if _, err := t.tx.Exec(ctx, "SET LOCAL TIME ZONE 'UTC'"); err != nil {
		return "", err
	}
	if _, err := t.tx.Exec(ctx, "SET LOCAL lock_timeout = '2s'"); err != nil {
		return "", err
	}
	if _, err := t.tx.Exec(ctx, "SET LOCAL statement_timeout = '15s'"); err != nil {
		return "", err
	}
	var login string
	var acreditado bool
	err := t.tx.QueryRow(ctx, `SELECT session_user::text, current_user=session_user AND current_setting('role')='none'
		AND pg_has_role(session_user,'vec_admin_ratificacion_catalogo_admin_ejecutor','MEMBER')
		AND NOT (SELECT rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls
			FROM pg_roles WHERE rolname=session_user)`).Scan(&login, &acreditado)
	if err != nil || !acreditado {
		return "", errOperacion
	}
	return login, nil
}

func (t *transaccionPG) CanonYDescriptores(ctx context.Context, plan string) (string, string, error) {
	var canon, hash string
	err := t.tx.QueryRow(ctx, `SELECT $1::jsonb::text,
		encode(sha256(convert_to(($1::jsonb->'descriptores')::text,'UTF8')),'hex')`, plan).Scan(&canon, &hash)
	return canon, hash, err
}

func (t *transaccionPG) Ratificar(ctx context.Context, plan, sha string) ([]byte, error) {
	var b []byte
	err := t.tx.QueryRow(ctx, `SELECT vec_autorizacion.ratificar_catalogo_admin_v7($1::text,$2::text)`, plan, sha).Scan(&b)
	return b, err
}

func (t *transaccionPG) Confirmar(ctx context.Context) error { return t.tx.Commit(ctx) }
func (t *transaccionPG) Cerrar(ctx context.Context) {
	_ = t.tx.Rollback(ctx)
	_ = t.conexion.Close(ctx)
}

func nuevaTransaccionPG(ctx context.Context, cfg conexionPrivada) (transaccion, error) {
	if err := dsnDeclaraIdentidad(cfg.DSN); err != nil {
		return nil, err
	}
	pc, err := pgx.ParseConfig(cfg.DSN)
	if err != nil || !canalValido(&pc.Config, cfg.PermitirSocketDesarrollo) {
		return nil, errOperacion
	}
	for k := range pc.RuntimeParams {
		if k != "application_name" {
			return nil, errOperacion
		}
	}
	pc.ConnectTimeout = 2 * time.Second
	conn, err := pgx.ConnectConfig(ctx, pc)
	if err != nil {
		return nil, errOperacion
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		_ = conn.Close(ctx)
		return nil, errOperacion
	}
	return &transaccionPG{tx: tx, conexion: conn}, nil
}

// Exigir los tres campos en el archivo privado impide que PGHOST, PGUSER o
// PGDATABASE del entorno seleccionen un destino o principal distintos.
func dsnDeclaraIdentidad(dsn string) error {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return dsnNoValido{causa: err}
		}
		if u.User == nil || u.User.Username() == "" || len(u.Path) < 2 ||
			(u.Hostname() == "" && u.Query().Get("host") == "") {
			return errOperacion
		}
		return nil
	}
	campos := map[string]bool{"host": false, "user": false, "dbname": false}
	for i := 0; i < len(dsn); {
		for i < len(dsn) && (dsn[i] == ' ' || dsn[i] == '\t') {
			i++
		}
		if i == len(dsn) {
			break
		}
		inicio := i
		for i < len(dsn) && dsn[i] != '=' && dsn[i] != ' ' && dsn[i] != '\t' {
			i++
		}
		if i == inicio || i == len(dsn) || dsn[i] != '=' {
			return errOperacion
		}
		clave := dsn[inicio:i]
		i++
		valor := false
		if i < len(dsn) && dsn[i] == '\'' {
			i++
			cerrada := false
			for i < len(dsn) {
				if dsn[i] == '\\' && i+1 < len(dsn) {
					valor = true
					i += 2
					continue
				}
				if dsn[i] == '\'' {
					cerrada = true
					i++
					break
				}
				valor = true
				i++
			}
			if !cerrada || i < len(dsn) && dsn[i] != ' ' && dsn[i] != '\t' {
				return errOperacion
			}
		} else {
			for i < len(dsn) && dsn[i] != ' ' && dsn[i] != '\t' {
				valor = true
				i++
			}
		}
		if _, esperado := campos[clave]; esperado && valor {
			campos[clave] = true
		}
	}
	if !campos["host"] || !campos["user"] || !campos["dbname"] {
		return errOperacion
	}
	return nil
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
