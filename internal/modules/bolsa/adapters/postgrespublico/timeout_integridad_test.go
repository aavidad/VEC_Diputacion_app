package postgrespublico

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type transaccionTimeoutEspia struct {
	pgx.Tx
	t         *testing.T
	contexto  context.Context
	efectivo  int64
	consultas int
}

func (tx *transaccionTimeoutEspia) QueryRow(ctx context.Context, sql string, argumentos ...any) pgx.Row {
	tx.t.Helper()
	tx.contexto = ctx
	tx.consultas++
	switch tx.consultas {
	case 1:
		if !strings.Contains(sql, "set_config('statement_timeout', $1, true)") {
			tx.t.Fatalf("sentencia de configuracion inesperada: %q", sql)
		}
		if len(argumentos) != 1 || argumentos[0] != "30s" {
			tx.t.Fatalf("timeout solicitado = %v", argumentos)
		}
		return filaTimeoutEspia{valor: "30s"}
	case 2:
		if !strings.Contains(sql, "current_setting('statement_timeout')") || len(argumentos) != 0 {
			tx.t.Fatalf("sentencia de verificacion inesperada: %q argumentos=%v", sql, argumentos)
		}
		efectivo := tx.efectivo
		if efectivo == 0 {
			efectivo = int64((30 * time.Second) / time.Millisecond)
		}
		return filaTimeoutEspia{valor: efectivo}
	default:
		tx.t.Fatalf("consulta de timeout adicional: %q", sql)
		return filaTimeoutEspia{}
	}
}

type filaTimeoutEspia struct {
	valor any
}

func (fila filaTimeoutEspia) Scan(destinos ...any) error {
	if len(destinos) != 1 {
		return errors.New("numero de destinos inesperado")
	}
	switch destino := destinos[0].(type) {
	case *string:
		valor, ok := fila.valor.(string)
		if !ok {
			return errors.New("valor de configuracion inesperado")
		}
		*destino = valor
	case *int64:
		valor, ok := fila.valor.(int64)
		if !ok {
			return errors.New("valor efectivo inesperado")
		}
		*destino = valor
	default:
		return errors.New("destino de timeout inesperado")
	}
	return nil
}

func TestIntegridadConfiguraStatementTimeoutEfectivoDeTreintaSegundos(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), duracionIntegridadDisponibilidad)
	defer cancelar()
	tx := &transaccionTimeoutEspia{t: t}
	if err := establecerStatementTimeout(ctx, tx, duracionIntegridadDisponibilidad); err != nil {
		t.Fatalf("configurar timeout integral: %v", err)
	}
	limite, existe := tx.contexto.Deadline()
	if !existe || time.Until(limite) <= 0 ||
		time.Until(limite) > duracionIntegridadDisponibilidad+100*time.Millisecond {
		t.Fatalf("se perdio el limite del contexto integral: %v", limite)
	}
}

func TestStatementTimeoutRechazaValorEfectivoDistinto(t *testing.T) {
	tx := &transaccionTimeoutEspia{t: t, efectivo: int64((10 * time.Second) / time.Millisecond)}
	if err := establecerStatementTimeout(
		context.Background(), tx, duracionIntegridadDisponibilidad,
	); !errors.Is(err, ErrConfiguracionPostgreSQLPublicaInvalida) {
		t.Fatalf("valor efectivo distinto aceptado: %v", err)
	}
}
