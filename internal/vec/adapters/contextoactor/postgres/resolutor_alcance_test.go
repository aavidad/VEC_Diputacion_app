package postgres

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/ports"
)

func adaptadorAlcancePrueba(t *testing.T, transacciones ...*txContextoActorDoble) (*ResolutorRegistroContextoActorPostgreSQLV2, *poolContextoActorDoble) {
	t.Helper()
	pool := &poolContextoActorDoble{transacciones: transacciones}
	adaptador, err := nuevoResolutorRegistroContextoActorPostgreSQLV2(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, 2*bytesAleatoriosReferenciaContextoActorV2)),
	)
	if err != nil {
		t.Fatal(err)
	}
	return adaptador, pool
}

func TestResolutorContextoActorPostgreSQLEnviaAlcanceCerrado(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		empleado bool
		esperado []any
	}{{"vacio", false, nil}, {"empleado", true, []any{[]string{"empleado"}}}} {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud, fila := solicitudYFilaContextoActorV2Alcance(t, caso.empleado)
			tx := &txContextoActorDoble{filas: []pgx.Row{fila}}
			adaptador, _ := adaptadorAlcancePrueba(t, tx)
			confirmacion, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
			if err != nil || confirmacion.Contexto.AlcanceProyecciones() != solicitud.Proyecciones {
				t.Fatalf("confirmacion con alcance %s rechazada: %v", caso.nombre, err)
			}
			// Alcance vacio: firma heredada exacta de siete argumentos.
			if len(tx.argumentos[0]) != 7+len(caso.esperado) ||
				!reflect.DeepEqual(tx.argumentos[0][7:], caso.esperado) && len(caso.esperado) > 0 ||
				strings.Contains(tx.consultas[0], "$8::text[]") != caso.empleado {
				t.Fatalf("alcance SQL inesperado: %#v", tx.argumentos[0])
			}
		})
	}
}

func TestResolutorContextoActorPostgreSQLRechazaReciboConOtroAlcance(t *testing.T) {
	// SQL devuelve un canon con empleado de Personal a una solicitud vacia, y
	// al reves: el recibo no demuestra el alcance pedido y se rechaza.
	for _, empleadoEnRecibo := range []bool{true, false} {
		solicitud, _ := solicitudYFilaContextoActorV2Alcance(t, !empleadoEnRecibo)
		_, fila := solicitudYFilaContextoActorV2Alcance(t, empleadoEnRecibo)
		adaptador, _ := adaptadorAlcancePrueba(t, &txContextoActorDoble{filas: []pgx.Row{fila}})
		if _, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud); !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
			t.Fatalf("recibo con alcance distinto aceptado (empleado en recibo=%v)", empleadoEnRecibo)
		}
	}
}

func TestResolutorContextoActorPostgreSQLDeniegaConMotivoSinReintentar(t *testing.T) {
	for codigo, motivo := range map[string]error{
		"PCA01": ports.ErrProyeccionEmpleadoContextoActorAusente,
		"PCA02": ports.ErrProyeccionEmpleadoContextoActorAmbigua,
	} {
		solicitud, _ := solicitudYFilaContextoActorV2Alcance(t, true)
		tx := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{
			err: &pgconn.PgError{Code: codigo, Message: "denegado"},
		}}}
		adaptador, pool := adaptadorAlcancePrueba(t, tx, &txContextoActorDoble{})
		_, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
		if !errors.Is(err, motivo) || !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
			t.Fatalf("%s no conservo su motivo: %v", codigo, err)
		}
		if pool.llamadas != 1 || tx.commits != 0 {
			t.Fatalf("%s se reintento o confirmo", codigo)
		}
	}
}

func TestResolutorContextoActorPostgreSQLReconciliaConElMismoAlcance(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2Alcance(t, true)
	escritura := &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: errors.New("commit ambiguo")}
	reconciliacion := &txContextoActorDoble{filas: []pgx.Row{fila}}
	adaptador, _ := adaptadorAlcancePrueba(t, escritura, reconciliacion)
	if _, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud); err != nil {
		t.Fatalf("reconciliacion con alcance rechazada: %v", err)
	}
	if !strings.Contains(reconciliacion.consultas[0], "reconciliar_contexto_actor_v2") ||
		!strings.Contains(reconciliacion.consultas[0], "$8::text[]") ||
		!reflect.DeepEqual(reconciliacion.argumentos[0], escritura.argumentos[0]) {
		t.Fatal("la reconciliacion no repitio la solicitud con el mismo alcance")
	}
}
