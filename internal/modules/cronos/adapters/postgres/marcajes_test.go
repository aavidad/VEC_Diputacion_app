package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestReciboConservaOriginalYRechazaCruces(t *testing.T) {
	ahora := time.Date(2026, 9, 19, 12, 0, 0, 123456000, time.UTC)
	original := domain.MarcajeOriginal{ClaveOperacion: "op-cronos-0001", InstanteUTC: ahora}
	no, si := false, true
	base := reciboSQL{Referencia: "recibo:cronos:00000000-0000-4000-8000-000000000001", MarcajeOriginalRef: "marcaje:cronos:op-cronos-0001", InstanteUTC: ahora, Replay: &no}
	if !reciboValido(base, original) {
		t.Fatal("recibo de primera escritura rechazado")
	}
	casos := []struct {
		nombre  string
		cambiar func(*reciboSQL)
	}{
		{"recibo confundido con marcaje", func(r *reciboSQL) { r.Referencia = r.MarcajeOriginalRef }},
		{"original ajeno", func(r *reciboSQL) { r.MarcajeOriginalRef = "marcaje:cronos:otra-operacion" }},
		{"hora cambiada primera escritura", func(r *reciboSQL) { r.InstanteUTC = ahora.Add(-time.Second) }},
		{"nanosegundos no durables", func(r *reciboSQL) { r.InstanteUTC = ahora.Add(time.Nanosecond) }},
		{"replay ausente", func(r *reciboSQL) { r.Replay = nil }},
		{"replay futuro", func(r *reciboSQL) { r.Replay = &si; r.InstanteUTC = ahora.Add(time.Second) }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := base
			c.cambiar(&r)
			if reciboValido(r, original) {
				t.Fatal("acepta recibo divergente")
			}
		})
	}
	base.Replay = &si
	base.InstanteUTC = ahora.Add(-time.Hour)
	if !reciboValido(base, original) {
		t.Fatal("replay no conserva hora original")
	}
}
func TestDecodificarReciboRechazaCamposAjenosYTamano(t *testing.T) {
	var r reciboSQL
	if decodificarRecibo([]byte(`{"dato_privado":"no"}`), &r) == nil {
		t.Fatal("campo desconocido")
	}
	if decodificarRecibo(make([]byte, 4097), &r) == nil {
		t.Fatal("recibo excesivo")
	}
}
func TestErrorSQLNoExponeDetalles(t *testing.T) {
	for _, c := range []struct {
		code     string
		esperado error
	}{{"PC002", ports.ErrClaveOperacionEnConflicto}, {"PC003", ports.ErrDependenciaNoDisponible}, {"42501", ports.ErrDependenciaNoDisponible}, {"40001", ports.ErrDependenciaNoDisponible}} {
		err := errorSeguro(context.Background(), &pgconn.PgError{Code: c.code, Message: "detalle_sql_privado"})
		if !errors.Is(err, c.esperado) {
			t.Fatalf("código %s no cerrado", c.code)
		}
	}
}
