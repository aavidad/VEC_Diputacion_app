package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaAcreditacionPrueba struct{ valido bool }

func (f filaAcreditacionPrueba) Scan(destinos ...any) error {
	*destinos[0].(*bool) = f.valido
	return nil
}

type transaccionAjustesPrueba struct {
	llamadas []string
	fallar   bool
	denegar  bool
}

func (tx *transaccionAjustesPrueba) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	tx.llamadas = append(tx.llamadas, "ajustes")
	if sql != ajustesTransaccionSQL || tx.fallar {
		return pgconn.CommandTag{}, errors.New("fallo al ajustar la transacción")
	}
	return pgconn.CommandTag{}, nil
}

func (tx *transaccionAjustesPrueba) QueryRow(_ context.Context, sql string, args ...any) fila {
	tx.llamadas = append(tx.llamadas, "acreditacion")
	return filaAcreditacionPrueba{valido: !tx.denegar && sql == acreditarEjecutorSQL && len(args) == 1 && args[0] == RolEjecutor}
}

func (tx *transaccionAjustesPrueba) Commit(context.Context) error {
	tx.llamadas = append(tx.llamadas, "commit")
	return nil
}

func (tx *transaccionAjustesPrueba) Rollback(context.Context) error {
	tx.llamadas = append(tx.llamadas, "rollback")
	return nil
}

func TestAbrirAgrupaAjustesAntesDeAcreditar(t *testing.T) {
	tx := &transaccionAjustesPrueba{}
	r := &RegistroFichasPostgreSQL{iniciar: func(context.Context) (transaccion, error) { return tx, nil }}
	obtenida, err := r.abrir(context.Background())
	if err != nil || obtenida != tx || len(tx.llamadas) != 2 || tx.llamadas[0] != "ajustes" || tx.llamadas[1] != "acreditacion" {
		t.Fatalf("apertura: tx=%v err=%v llamadas=%v", obtenida, err, tx.llamadas)
	}
}

func TestAbrirFalloAjustesRevierteSinAcreditarNiLeer(t *testing.T) {
	tx := &transaccionAjustesPrueba{fallar: true}
	r := &RegistroFichasPostgreSQL{iniciar: func(context.Context) (transaccion, error) { return tx, nil }}
	if obtenida, err := r.abrir(context.Background()); obtenida != nil || !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatalf("apertura tras fallo: tx=%v err=%v", obtenida, err)
	}
	if len(tx.llamadas) != 2 || tx.llamadas[0] != "ajustes" || tx.llamadas[1] != "rollback" {
		t.Fatalf("fallo antes de datos: %v", tx.llamadas)
	}
}

func TestAbrirAcreditacionDenegadaRevierteSinLeer(t *testing.T) {
	tx := &transaccionAjustesPrueba{denegar: true}
	r := &RegistroFichasPostgreSQL{iniciar: func(context.Context) (transaccion, error) { return tx, nil }}
	if obtenida, err := r.abrir(context.Background()); obtenida != nil || !errors.Is(err, ports.ErrNoDisponible) {
		t.Fatalf("acreditación denegada: tx=%v err=%v", obtenida, err)
	}
	if got := tx.llamadas; len(got) != 3 || got[0] != "ajustes" || got[1] != "acreditacion" || got[2] != "rollback" {
		t.Fatalf("denegación sin lectura ni commit: %v", got)
	}
}

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

func TestDenegacionSoloDeAspirantesYSinPersona(t *testing.T) {
	r := &RegistroFichasPostgreSQL{}
	for _, o := range []vecports.OrdenAuditoriaFronteraRutaExacta{
		{CorrelacionRef: "corr_no_disponible", Motivo: vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes, Ruta: "/api/vec/aspirantes/area-personal/mi-ficha", ActorRef: "per_AAAAAAAAAAAAAAAAAAAAAA"},
		{CorrelacionRef: "corr_no_disponible", Motivo: vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias, Ruta: "/api/vec/usuarios/area-personal/mis-preferencias"},
		{CorrelacionRef: "corr_no_disponible", Motivo: vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes, Ruta: "/api/vec/aspirantes/otra"},
	} {
		if err := r.RegistrarAuditoriaFronteraRutaExacta(context.Background(), o); !errors.Is(err, ports.ErrInvalida) {
			t.Fatalf("%+v: %v", o, err)
		}
	}
}
