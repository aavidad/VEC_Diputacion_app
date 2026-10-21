package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaEstadoSeleccionOfertaConflictoSQLNoPareceAusencia(t *testing.T) {
	for _, codigo := range []string{"23505", "55000", "40001", "40P01"} {
		err := errorRegistroSeleccionOferta(context.Background(), &pgconn.PgError{Code: codigo})
		if !errors.Is(err, ports.ErrVinculoOfertaBolsaCTInvalido) {
			t.Fatalf("SQLSTATE %s no conserva conflicto: %v", codigo, err)
		}
	}
}

func TestDecodificarIntencionSingularPendiente(t *testing.T) {
	q := ports.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT{
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba",
		VersionObservada: 6,
	}
	const original = `{"organizacion_ref":"organizacion:prueba","expediente_ref":"expediente:prueba",` +
		`"version_expediente":6,"clave_idempotencia":"018f47a2-6b31-4c80-8a95-4d2e707c5a11"}`
	pendiente, err := decodificarIntencionSingularPendiente("indeterminada", []byte(original),
		[]byte(`{"efecto":"solicitar_llamamiento","ventana_abierta":true}`), q)
	if err != nil || !pendiente.Encontrada || pendiente.Singular == nil ||
		pendiente.Singular.ClaveIdempotencia != "018f47a2-6b31-4c80-8a95-4d2e707c5a11" {
		t.Fatalf("intención pendiente no recuperada: %+v %v", pendiente, err)
	}
	for _, caso := range []struct {
		original string
		estado   string
	}{
		{original, `{"efecto":"solicitar_llamamiento","ventana_abierta":false}`},
		{original, `{"efecto":"otro","ventana_abierta":true}`},
		{`{"organizacion_ref":"organizacion:ajena","expediente_ref":"expediente:prueba",` +
			`"version_expediente":6,"clave_idempotencia":"018f47a2-6b31-4c80-8a95-4d2e707c5a11"}`,
			`{"efecto":"solicitar_llamamiento","ventana_abierta":true}`},
	} {
		if _, err := decodificarIntencionSingularPendiente("indeterminada", []byte(caso.original),
			[]byte(caso.estado), q); err == nil {
			t.Fatal("se aceptó una intención singular divergente")
		}
	}
}
