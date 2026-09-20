package gatewaypersonal

import (
	"bytes"
	"context"
	"testing"
	"time"
)

type relojPrueba struct{ t time.Time }

func (r relojPrueba) Ahora() time.Time { return r.t }

type almacenPrueba struct {
	hash           []byte
	cuenta, huella string
	expira         time.Time
	cerrada        bool
}

func (a *almacenPrueba) Abrir(_ context.Context, h []byte, c, f string, _ time.Time, e time.Time) error {
	a.hash = append([]byte(nil), h...)
	a.cuenta = c
	a.huella = f
	a.expira = e
	return nil
}
func (a *almacenPrueba) Activa(_ context.Context, h []byte, _ time.Time) (bool, time.Time, error) {
	return !a.cerrada && bytes.Equal(h, a.hash), a.expira, nil
}
func (a *almacenPrueba) Cerrar(_ context.Context, h []byte, _ time.Time) error {
	if bytes.Equal(h, a.hash) {
		a.cerrada = true
	}
	return nil
}
func TestAbrirHashNoConservaCookieYRevoca(t *testing.T) {
	a := new(almacenPrueba)
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	s, e := Nuevo(a, bytes.Repeat([]byte("k"), 32), bytes.NewReader(bytes.Repeat([]byte{1}, 32)), relojPrueba{ahora})
	if e != nil {
		t.Fatal(e)
	}
	token, exp, e := s.Abrir(context.Background(), "cuenta:1", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if e != nil || len(token) != 64 || bytes.Equal(a.hash, []byte(token)) || !exp.Equal(ahora.Add(VidaSesion)) {
		t.Fatalf("alta insegura: %q %x %v %v", token, a.hash, exp, e)
	}
	ok, got, e := s.Activa(context.Background(), token)
	if e != nil || !ok || !got.Equal(exp) {
		t.Fatalf("sesion no activa: %v %v %v", ok, got, e)
	}
	if e = s.Cerrar(context.Background(), token); e != nil {
		t.Fatal(e)
	}
	if ok, _, _ = s.Activa(context.Background(), token); ok {
		t.Fatal("sesion no revocada")
	}
}
