package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPreparadorPostgreSQLRechazaParCruzadoEntreGeneraciones(t *testing.T) {
	fila := filaReservadaPreparacionPrueba("reutilizada").(filaPreparacionPrueba)
	fila.valores[5] = selloHMACPrueba(claveAmbitoAltaPruebaV2, "e")
	// La huella permanece en v1. Ambos sellos pertenecen a la petición, pero
	// nunca formaron un par en la misma generación.
	tx := &transaccionPreparacionPrueba{fila: fila}
	preparador, err := nuevoPreparadorAltaPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
		&selladorAmbitoPrueba{
			huella: selloHMACPrueba(claveAmbitoAltaPruebaV2, "e"),
			retenidas: []string{
				selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
			},
		},
		&generadorReferenciasPrueba{
			referencias: referenciasPreparacionPrueba(),
			reservaRef:  "reserva:alta-candidata-001",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudPreparacionPrueba()
	solicitud.HuellasPeticionHMAC = coleccionPostgreSQLPrueba(
		selloHMACPrueba(clavePeticionAltaPruebaV2, "c"),
		selloHMACPrueba(clavePeticionAltaPrueba, "b"),
	)
	_, err = preparador.PrepararAlta(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		tx.confirmaciones != 0 {
		t.Fatalf("par cruzado no rechazado: err=%v tx=%#v", err, tx)
	}
}

func TestPreparadorPostgreSQLReintentaSoloConflictosTransitorios(t *testing.T) {
	t.Run("cero reintentos", func(t *testing.T) {
		tx := &transaccionPreparacionPrueba{
			fila: filaReservadaPreparacionPrueba("reservada"),
		}
		iniciador := &iniciadorPreparacionPrueba{tx: tx}
		preparador := nuevoPreparadorConIniciadorPrueba(t, iniciador)
		if _, err := preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		); err != nil {
			t.Fatal(err)
		}
		if iniciador.inicios != 1 || tx.confirmaciones != 1 {
			t.Fatalf("intentos inesperados: inicio=%d tx=%#v", iniciador.inicios, tx)
		}
	})

	t.Run("un reintento tras commit serializable", func(t *testing.T) {
		primera := &transaccionPreparacionPrueba{
			fila: filaReservadaPreparacionPrueba("reservada"),
			errConfirmar: &pgconn.PgError{
				Code: "40001",
			},
		}
		segunda := &transaccionPreparacionPrueba{
			fila: filaReservadaPreparacionPrueba("reservada"),
		}
		iniciador := &iniciadorPreparacionPrueba{
			transacciones: []pgx.Tx{primera, segunda},
		}
		preparador := nuevoPreparadorConIniciadorPrueba(t, iniciador)
		if _, err := preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		); err != nil {
			t.Fatal(err)
		}
		if iniciador.inicios != 2 ||
			primera.confirmaciones != 1 ||
			segunda.confirmaciones != 1 {
			t.Fatalf(
				"no se abrió una transacción fresca: inicio=%d primera=%#v segunda=%#v",
				iniciador.inicios,
				primera,
				segunda,
			)
		}
	})

	t.Run("commit confirmado no se convierte en cancelación tardía", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := &transaccionPreparacionPrueba{
			fila:        filaReservadaPreparacionPrueba("reservada"),
			alConfirmar: cancelar,
		}
		iniciador := &iniciadorPreparacionPrueba{tx: tx}
		preparador := nuevoPreparadorConIniciadorPrueba(t, iniciador)
		preparacion, err := preparador.PrepararAlta(
			ctx,
			solicitudPreparacionPrueba(),
		)
		if err != nil || preparacion.Estado != ports.PreparacionReservada ||
			iniciador.inicios != 1 {
			t.Fatalf(
				"commit confirmado convertido en fallo: preparacion=%#v err=%v",
				preparacion,
				err,
			)
		}
	})

	t.Run("agotamiento técnico", func(t *testing.T) {
		transacciones := make([]pgx.Tx, maximoIntentosPrepararAlta)
		for indice := range transacciones {
			transacciones[indice] = &transaccionPreparacionPrueba{
				fila: filaPreparacionPrueba{
					err: &pgconn.PgError{Code: "40P01"},
				},
			}
		}
		iniciador := &iniciadorPreparacionPrueba{
			transacciones: transacciones,
		}
		preparador := nuevoPreparadorConIniciadorPrueba(t, iniciador)
		_, err := preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		)
		if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
			iniciador.inicios != maximoIntentosPrepararAlta {
			t.Fatalf("agotamiento incorrecto: err=%v intentos=%d", err, iniciador.inicios)
		}
	})

	t.Run("cancelación corta reintento", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		primera := &transaccionPreparacionPrueba{
			fila: filaPreparacionPrueba{
				err:        &pgconn.PgError{Code: "40001"},
				alEscanear: cancelar,
			},
		}
		iniciador := &iniciadorPreparacionPrueba{
			transacciones: []pgx.Tx{primera},
		}
		preparador := nuevoPreparadorConIniciadorPrueba(t, iniciador)
		_, err := preparador.PrepararAlta(ctx, solicitudPreparacionPrueba())
		if !errors.Is(err, context.Canceled) || iniciador.inicios != 1 {
			t.Fatalf("cancelación perdida: err=%v intentos=%d", err, iniciador.inicios)
		}
	})

	t.Run("sqlstate no reintentable", func(t *testing.T) {
		primera := &transaccionPreparacionPrueba{
			fila: filaPreparacionPrueba{
				err: &pgconn.PgError{Code: "23505"},
			},
		}
		iniciador := &iniciadorPreparacionPrueba{
			transacciones: []pgx.Tx{primera},
		}
		preparador := nuevoPreparadorConIniciadorPrueba(t, iniciador)
		_, err := preparador.PrepararAlta(
			context.Background(),
			solicitudPreparacionPrueba(),
		)
		if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
			iniciador.inicios != 1 {
			t.Fatalf("error permanente reintentado: err=%v intentos=%d", err, iniciador.inicios)
		}
	})
}

func nuevoPreparadorConIniciadorPrueba(
	t *testing.T,
	iniciador iniciadorTransacciones,
) *PreparadorAltaPostgreSQL {
	t.Helper()
	preparador, err := nuevoPreparadorAltaPostgreSQL(
		iniciador,
		&selladorAmbitoPrueba{
			huella: selloHMACPrueba(claveAmbitoAltaPrueba, "d"),
		},
		&generadorReferenciasPrueba{
			referencias: referenciasPreparacionPrueba(),
			reservaRef:  "reserva:alta-candidata-001",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return preparador
}
