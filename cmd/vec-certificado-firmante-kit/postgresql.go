package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func destinoJSON(b []byte) (destino, error) {
	var d destino
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 6 {
		return d, errors.New("destino")
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if dec.Decode(&d) != nil || dec.Decode(new(any)) != io.EOF || !referenciaValida(d.CuentaRef, "cta_") ||
		!referenciaValida(d.PersonaRef, "per_") || !referenciaValida(d.VinculoRef, "vca_") ||
		d.CuentaVersion == 0 || d.PersonaVersion == 0 || d.VinculoVersion == 0 {
		return destino{}, errors.New("destino")
	}
	return d, nil
}

func hostLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func consultarDestino(registro string) (destino, error) {
	var vacio destino
	dsn := os.Getenv("VEC_KIT_CERTIFICADO_FIRMANTE_DSN")
	if dsn == "" {
		return vacio, errors.New("conexion")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || !hostLoopback(cfg.Host) || len(cfg.Fallbacks) != 0 ||
		(cfg.TLSConfig != nil && cfg.TLSConfig.InsecureSkipVerify) || cfg.User == "" || cfg.Database == "" {
		return vacio, errors.New("conexion")
	}
	if cfg.ConnectTimeout == 0 || cfg.ConnectTimeout > 5*time.Second {
		cfg.ConnectTimeout = 5 * time.Second
	}
	// localhost siempre se dirige a 127.0.0.1: no se resuelve por DNS.
	cfg.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil || !hostLoopback(host) || (network != "tcp" && network != "tcp4" && network != "tcp6") {
			return nil, errors.New("conexion")
		}
		if host == "localhost" {
			host = "127.0.0.1"
			network = "tcp4"
		}
		return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(host, port))
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = map[string]string{}
	}
	cfg.RuntimeParams["application_name"] = "vec-certificado-firmante-kit"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	con, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return vacio, errors.New("conexion")
	}
	defer con.Close(context.Background())
	tx, err := con.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errors.New("consulta")
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SET LOCAL TIME ZONE 'UTC'"); err != nil {
		return vacio, errors.New("consulta")
	}
	var login string
	var acreditada bool
	err = tx.QueryRow(ctx, "SELECT identidad_login,acreditada FROM vec_contexto_actor_v1.acreditar_runtime_certificado_firmante_ct_v1()").Scan(&login, &acreditada)
	if err != nil || !acreditada || login == "" || login != cfg.User {
		return vacio, errors.New("consulta")
	}
	var b []byte
	err = tx.QueryRow(ctx, "SELECT vec_contexto_actor_v1.exportar_destino_vinculo_certificado_ct_v1($1)::text", registro).Scan(&b)
	if err != nil {
		return vacio, errors.New("consulta")
	}
	d, err := destinoJSON(b)
	if err != nil {
		return vacio, errors.New("consulta")
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, errors.New("consulta")
	}
	return d, nil
}
