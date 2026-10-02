package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestResolutorExternoReconciliaCommitInciertoSinDuplicar(t *testing.T) {
	probarConsumidoresContextoExterno(t, func(t *testing.T, usuarios bool) {
		for _, resultado := range []string{"misma", "distinta", "revocada", "ausente", "ausente_dos_veces"} {
			t.Run(resultado, func(t *testing.T) {
				solicitud, fila := solicitudYFilaContextoExterno(t, usuarios)
				primera := &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: errors.New("commit incierto sintético")}
				reconciliacion := &txContextoActorDoble{filas: []pgx.Row{fila}}
				transacciones := []*txContextoActorDoble{primera, reconciliacion}
				esperadas := 2
				exito := resultado == "misma" || resultado == "ausente"
				switch resultado {
				case "distinta":
					valores := append([]any(nil), fila.valores...)
					valores[3] = strings.Repeat("0", 64)
					reconciliacion.filas = []pgx.Row{filaContextoActorDoble{valores: valores}}
				case "revocada":
					reconciliacion.filas = []pgx.Row{filaContextoActorDoble{err: &pgconn.PgError{Code: "42501"}}}
				case "ausente", "ausente_dos_veces":
					reconciliacion.filas = []pgx.Row{filaContextoActorDoble{err: pgx.ErrNoRows}}
					repetida := &txContextoActorDoble{filas: []pgx.Row{fila}}
					transacciones = append(transacciones, repetida)
					esperadas = 3
					if resultado == "ausente_dos_veces" {
						repetida.errCommit = errors.New("segundo commit incierto sintético")
						transacciones = append(transacciones, &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{err: pgx.ErrNoRows}}})
						esperadas = 4
					}
				}
				// Una transacción disponible extra detecta toda escritura adicional.
				transacciones = append(transacciones, &txContextoActorDoble{filas: []pgx.Row{fila}})
				pool := &poolContextoActorDoble{transacciones: transacciones}
				r := nuevoResolutorContextoExternoPrueba(t, pool, usuarios)
				confirmacion, err := r.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
				if (exito && (err != nil || confirmacion.ValidarParaProductiva(solicitud) != nil)) ||
					(!exito && (!errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || confirmacion.RegistroContextoRef != "")) || pool.llamadas != esperadas {
					t.Fatalf("reconciliación inesperada: intentos=%d err=%v", pool.llamadas, err)
				}
				comprobarIntentosContextoExterno(t, pool, usuarios)
				if pool.opciones[1].IsoLevel != pgx.ReadCommitted || !strings.Contains(reconciliacion.consultas[0], "reconciliar_contexto_") {
					t.Fatal("commit incierto repetido antes de reconciliar")
				}
				if exito && (confirmacion.RegistroContextoRef != fila.valores[1] || !confirmacion.ResueltoEnAutoritativo.Equal(fila.valores[7].(time.Time))) {
					t.Fatal("recibo o fecha cambiaron durante la recuperación")
				}
			})
		}
	})
}
