package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"strings"
	"time"
)

var syntheticLogin = regexp.MustCompile(`^vec_rum03_[a-z0-9_]{1,40}$`)

func pool(ctx context.Context, login, password string) (*pgxpool.Pool, error) {
	// Network is disabled by the execution sandbox. This additionally closes
	// configuration to synthetic logins on the clone's private Unix socket.
	if !syntheticLogin.MatchString(login) {
		return nil, errors.New("login_scope")
	}
	if password == "" || len(password) > 1024 || strings.ContainsAny(password, "\x00\r\n") {
		return nil, errors.New("credential_not_available")
	}
	cfg, err := pgxpool.ParseConfig("host=/socket port=5432 dbname=postgres sslmode=disable user=vec_rum03_runtime_interno password=fixture_socket_only")
	if err != nil {
		return nil, err
	}
	// Sólo el socket y las cuentas sintéticas nominales; no hereda una DSN,
	// contraseña, host alternativo, parámetros de sesión ni fallback del entorno.
	cfg.ConnConfig.Host, cfg.ConnConfig.Port, cfg.ConnConfig.Database, cfg.ConnConfig.User = "/socket", 5432, "postgres", login
	cfg.ConnConfig.Password, cfg.ConnConfig.TLSConfig, cfg.ConnConfig.Fallbacks = password, nil, nil
	cfg.ConnConfig.RuntimeParams = map[string]string{}
	cfg.MaxConns = 1
	cfg.ConnConfig.ConnectTimeout = 3 * time.Second
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}
