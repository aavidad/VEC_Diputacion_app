// Package gatewaypersonal compone el nucleo del acceso personal. No conoce HTTP,
// PostgreSQL ni los ficheros que contienen secretos.
package gatewaypersonal

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
)

var ErrDenegado = errors.New("gateway personal: acceso denegado")

const VidaSesion = 20 * time.Minute

type Reloj interface{ Ahora() time.Time }
type relojSistema struct{}

func (relojSistema) Ahora() time.Time { return time.Now().UTC() }

// AlmacenSesion es el unico puerto durable de este corte. El valor de sesion
// siempre es un HMAC, nunca la cookie opaca.
type AlmacenSesion interface {
	Abrir(context.Context, []byte, string, string, time.Time, time.Time) error
	Activa(context.Context, []byte, time.Time) (bool, time.Time, error)
	Cerrar(context.Context, []byte, time.Time) error
}

type Servicio struct {
	almacen   AlmacenSesion
	reloj     Reloj
	aleatorio io.Reader
	clave     []byte
}

func Nuevo(almacen AlmacenSesion, clave []byte, aleatorio io.Reader, reloj Reloj) (*Servicio, error) {
	if almacen == nil || len(clave) < 32 || aleatorio == nil {
		return nil, ErrDenegado
	}
	if reloj == nil {
		reloj = relojSistema{}
	}
	return &Servicio{almacen: almacen, clave: append([]byte(nil), clave...), aleatorio: aleatorio, reloj: reloj}, nil
}

func (s *Servicio) Abrir(ctx context.Context, cuenta, huella string) (string, time.Time, error) {
	if s == nil || ctx == nil || !referenciaValida(cuenta) || !huellaValida(huella) {
		return "", time.Time{}, ErrDenegado
	}
	bruto := make([]byte, 32)
	if _, err := io.ReadFull(s.aleatorio, bruto); err != nil {
		return "", time.Time{}, ErrDenegado
	}
	token := hex.EncodeToString(bruto)
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	expira := ahora.Add(VidaSesion)
	if err := s.almacen.Abrir(ctx, s.hash(token), cuenta, huella, ahora, expira); err != nil {
		return "", time.Time{}, ErrDenegado
	}
	return token, expira, nil
}

func (s *Servicio) Activa(ctx context.Context, token string) (bool, time.Time, error) {
	if s == nil || ctx == nil || !tokenValido(token) {
		return false, time.Time{}, ErrDenegado
	}
	ok, expira, err := s.almacen.Activa(ctx, s.hash(token), s.reloj.Ahora().UTC().Truncate(time.Microsecond))
	if err != nil || !ok {
		return false, time.Time{}, ErrDenegado
	}
	return true, expira, nil
}

func (s *Servicio) Cerrar(ctx context.Context, token string) error {
	if s == nil || ctx == nil || !tokenValido(token) {
		return ErrDenegado
	}
	if err := s.almacen.Cerrar(ctx, s.hash(token), s.reloj.Ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return ErrDenegado
	}
	return nil
}

func (s *Servicio) hash(token string) []byte {
	m := hmac.New(sha256.New, s.clave)
	_, _ = m.Write([]byte(token))
	return m.Sum(nil)
}
func tokenValido(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil
}
func huellaValida(s string) bool     { return tokenValido(strings.ToLower(s)) }
func referenciaValida(s string) bool { return len(s) > 0 && len(s) <= 200 && strings.TrimSpace(s) == s }
