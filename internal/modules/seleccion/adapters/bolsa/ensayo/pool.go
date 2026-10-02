package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"time"
)

var syntheticLogin = regexp.MustCompile(`^vec_s1_[a-z0-9_]{1,40}$`)

func pool(ctx context.Context, login string) (*pgxpool.Pool, error) {
	// Network is disabled by the execution sandbox. This additionally closes
	// configuration to synthetic logins on the clone's private Unix socket.
	if !syntheticLogin.MatchString(login) {
		return nil, errors.New("login_scope")
	}
	cfg, err := pgxpool.ParseConfig("host=/socket dbname=postgres sslmode=disable user=" + login)
	if err != nil {
		return nil, err
	}
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
