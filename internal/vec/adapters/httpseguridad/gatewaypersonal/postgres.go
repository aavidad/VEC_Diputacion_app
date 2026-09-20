// Package gatewaypersonal adapta exclusivamente las tres funciones del rol
// ejecutor; no consulta tablas del esquema de sesiones.
package gatewaypersonal

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

var ErrAlmacen = errors.New("gateway personal postgres: operacion no disponible")

type AlmacenPostgreSQL struct{ pool *pgxpool.Pool }

func Abrir(ctx context.Context, dsn string) (*AlmacenPostgreSQL, error) {
	if ctx == nil || dsn == "" {
		return nil, ErrAlmacen
	}
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, ErrAlmacen
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, ErrAlmacen
	}
	return &AlmacenPostgreSQL{pool: p}, nil
}
func (a *AlmacenPostgreSQL) CerrarPool() {
	if a != nil && a.pool != nil {
		a.pool.Close()
	}
}
func (a *AlmacenPostgreSQL) Abrir(ctx context.Context, hash []byte, cuenta, huella string, autenticado, expira time.Time) error {
	if a == nil || a.pool == nil || ctx == nil {
		return ErrAlmacen
	}
	var activa bool
	var expiraReal *time.Time
	var cuentaReal *string
	err := a.pool.QueryRow(ctx, `SELECT * FROM vec_gateway_personal.abrir_sesion_v1($1,$2,$3,$4,$5)`, hex.EncodeToString(hash), cuenta, huella, autenticado, expira).Scan(&activa, &expiraReal, &cuentaReal)
	if err != nil || !activa || expiraReal == nil || cuentaReal == nil || *cuentaReal != cuenta || !expiraReal.UTC().Equal(expira.UTC()) {
		return ErrAlmacen
	}
	return nil
}
func (a *AlmacenPostgreSQL) Activa(ctx context.Context, hash []byte, ahora time.Time) (bool, time.Time, error) {
	if a == nil || a.pool == nil || ctx == nil {
		return false, time.Time{}, ErrAlmacen
	}
	var activa bool
	var expira *time.Time
	var cuenta *string
	if err := a.pool.QueryRow(ctx, `SELECT * FROM vec_gateway_personal.consultar_sesion_v1($1,$2)`, hex.EncodeToString(hash), ahora).Scan(&activa, &expira, &cuenta); err != nil {
		return false, time.Time{}, ErrAlmacen
	}
	if !activa || expira == nil || cuenta == nil {
		return false, time.Time{}, nil
	}
	return true, expira.UTC(), nil
}
func (a *AlmacenPostgreSQL) Cerrar(ctx context.Context, hash []byte, ahora time.Time) error {
	if a == nil || a.pool == nil || ctx == nil {
		return ErrAlmacen
	}
	_, err := a.pool.Exec(ctx, `SELECT vec_gateway_personal.cerrar_sesion_v1($1,$2)`, hex.EncodeToString(hash), ahora)
	if err != nil {
		return ErrAlmacen
	}
	return nil
}
