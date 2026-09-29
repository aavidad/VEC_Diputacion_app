package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/aspirantes/ports"
)

func TestErroresSQLNominales(t *testing.T) {
	casos := map[string]error{"42501": ports.ErrProhibido, "P1409": ports.ErrConflicto, "P1411": ports.ErrFichaExistente,
		"P1404": ports.ErrSinFicha, "22023": ports.ErrInvalida, "40001": ports.ErrNoDisponible, "23514": ports.ErrNoDisponible}
	for codigo, esperado := range casos {
		err := errorSeguro(context.Background(), &pgconn.PgError{Code: codigo, Message: "detalle interno con per_secreto"})
		if err != esperado {
			t.Fatalf("%s: %v", codigo, err)
		}
	}
	if errorSeguro(context.Background(), errors.New("dial tcp")) != ports.ErrNoDisponible {
		t.Fatal("error de red")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(errorSeguro(ctx, errors.New("x")), context.Canceled) {
		t.Fatal("cancelación")
	}
}

func TestReciboExigeFormaExacta(t *testing.T) {
	bien := reciboSQL{ReciboRef: "asprec_0123456789abcdef0123456789abcdef", Accion: ports.AccionAlta, Version: 1, FechaUTC: time.Now().UTC()}
	if _, err := bien.recibo(ports.AccionAlta, 1); err != nil {
		t.Fatal(err)
	}
	for _, mal := range []reciboSQL{
		{ReciboRef: "corto", Accion: ports.AccionAlta, Version: 1, FechaUTC: time.Now().UTC()},
		{ReciboRef: bien.ReciboRef, Accion: ports.AccionRectificar, Version: 1, FechaUTC: time.Now().UTC()},
		{ReciboRef: bien.ReciboRef, Accion: ports.AccionAlta, Version: 2, FechaUTC: time.Now().UTC()},
		{ReciboRef: bien.ReciboRef, Accion: ports.AccionAlta, Version: 1},
		{ReciboRef: bien.ReciboRef, Accion: ports.AccionAlta, Version: 1, FechaUTC: time.Now().In(time.FixedZone("CEST", 7200))},
	} {
		if _, err := mal.recibo(ports.AccionAlta, 1); err == nil {
			t.Fatalf("aceptado %+v", mal)
		}
	}
}

func TestDecodificacionEstricta(t *testing.T) {
	var f fichaSQL
	if decodificarEstricto([]byte(`{"estado":"sin_ficha","extra":1}`), &f) == nil {
		t.Fatal("campo desconocido aceptado")
	}
	if decodificarEstricto([]byte(`{"estado":"sin_ficha"} {}`), &f) == nil {
		t.Fatal("dos documentos aceptados")
	}
	if _, err := (sobreSQL{ClaveRef: "k", NonceHex: "00", CifradoHex: "00"}).decodificar(46); err == nil {
		t.Fatal("sobre corto aceptado")
	}
}
