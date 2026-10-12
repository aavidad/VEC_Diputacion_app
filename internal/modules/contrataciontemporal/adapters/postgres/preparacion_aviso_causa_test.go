package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestLecturaAvisoConservaSQLStateSinPublicarMensaje(t *testing.T) {
	causa := &pgconn.PgError{Code: "42501", Message: "dato privado de la fila", Detail: "DSN privado"}
	eventos := []string{}
	fila := &filaConsultaRRHHPrueba{err: causa, eventos: &eventos}
	tx := &transaccionConsultaRRHHPrueba{fila: fila, eventos: &eventos}
	pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
	lector := &LectorExpedienteSeleccionLlamamientoPostgreSQL{pool: pool}
	_, err := lector.LeerExpedienteParaAvisoConfirmado(context.Background(), "organizacion:ct", "expediente:ct", "llamamiento:ct")
	var pg *pgconn.PgError
	if !errors.Is(err, ports.ErrIntegracionBolsaNoDisponible) || !errors.As(err, &pg) || pg != causa {
		t.Fatalf("se perdió la causa o el centinela: %T", err)
	}
	if err.Error() != ports.ErrIntegracionBolsaNoDisponible.Error() {
		t.Fatal("el mensaje público contiene la causa privada")
	}
	if tx.confirmaciones != 0 || tx.reversiones != 1 || tx.consultas != 1 {
		t.Fatalf("lectura fallida: commits=%d rollback=%d consultas=%d", tx.confirmaciones, tx.reversiones, tx.consultas)
	}
}

func TestLecturaAvisoConservaCausaDeApertura(t *testing.T) {
	causa := errors.New("detalle privado de conexión")
	lector := &LectorExpedienteSeleccionLlamamientoPostgreSQL{pool: &iniciadorEjecucionSeleccionO6Prueba{err: causa}}
	_, err := lector.LeerExpedienteParaAvisoConfirmado(context.Background(), "organizacion:ct", "expediente:ct", "llamamiento:ct")
	if !errors.Is(err, causa) || !errors.Is(err, ports.ErrIntegracionBolsaNoDisponible) || err.Error() != ports.ErrIntegracionBolsaNoDisponible.Error() {
		t.Fatal("causa de apertura perdida o publicada")
	}
}

func TestErrorLecturaSeleccionPriorizaCancelacion(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := errorLecturaSeleccion(ctx, errors.New("causa secundaria")); err != context.Canceled {
		t.Fatalf("no se conserva la cancelación: %T", err)
	}
	if errorLecturaSeleccion(context.Background()) != ports.ErrIntegracionBolsaNoDisponible {
		t.Fatal("cambió el rechazo sin causa")
	}
}
