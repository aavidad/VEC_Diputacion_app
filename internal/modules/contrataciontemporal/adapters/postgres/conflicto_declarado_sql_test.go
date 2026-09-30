package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Solo el 40001 que lanza la propia función (RAISE) es un conflicto
// declarado; el del servidor sigue siendo una carrera que se repite.
func TestConflictoDeclaradoPorFuncionSQL(t *testing.T) {
	for _, caso := range []struct {
		err       error
		declarado bool
	}{
		{&pgconn.PgError{Code: "40001", Routine: rutinaRaisePLpgSQL}, true},
		{&pgconn.PgError{Code: "40001", Routine: "CheckForSerializationFailure"}, false},
		{&pgconn.PgError{Code: "40P01", Routine: rutinaRaisePLpgSQL}, false},
		{&pgconn.PgError{Code: "P0001", Routine: rutinaRaisePLpgSQL}, false},
		{errors.New("otro"), false},
		{nil, false},
	} {
		if conflictoDeclaradoPorFuncionSQL(caso.err) != caso.declarado {
			t.Fatalf("%v: declarado distinto de %v", caso.err, caso.declarado)
		}
	}
}

// Un conflicto declarado en la preparación de la asignación no se repite y
// responde conflicto de versión; una carrera del servidor se repite.
func TestPreparadorAsignacionConflictoDeclaradoNoSeRepite(t *testing.T) {
	expediente := expedienteAsignacionPostgreSQLPrueba(t)
	solicitud := solicitudAsignacionPostgreSQLPrueba(t, expediente)
	referencias := referenciasAsignacionPostgreSQLPrueba()
	for _, caso := range []struct {
		rutina    string
		inicios   int
		conflicto bool
	}{{rutinaRaisePLpgSQL, 1, true}, {"CheckForSerializationFailure", 2, false}} {
		primera := &transaccionPreparacionPrueba{fila: filaPreparacionPrueba{err: &pgconn.PgError{Code: "40001", Routine: caso.rutina}}}
		segunda := &transaccionPreparacionPrueba{fila: filaAsignacionPostgreSQLPrueba(t, "reservada", solicitud, expediente, referencias)}
		iniciador := &iniciadorPreparacionPrueba{transacciones: []pgx.Tx{primera, segunda}}
		preparador := preparadorAsignacionPostgreSQLPrueba(t, iniciador, referencias)
		_, err := preparador.PrepararAsignacion(context.Background(), solicitud)
		if errors.Is(err, domain.ErrVersionEnConflicto) != caso.conflicto || iniciador.inicios != caso.inicios ||
			(!caso.conflicto && err != nil) || primera.confirmaciones != 0 {
			t.Fatalf("%s: error %v, inicios %d", caso.rutina, err, iniciador.inicios)
		}
	}
}
