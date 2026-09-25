package postgres

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestClasificarErrorSQLSeparaDominioDeIndisponibilidad(t *testing.T) {
	casos := []struct {
		err  error
		want error
	}{
		{&pgconn.PgError{Code: "23505", Message: "documentos: clave idempotente reutilizada"}, ports.ErrConflicto},
		{&pgconn.PgError{Code: "PC003"}, ports.ErrCapacidadNoDisponible},
		{&pgconn.PgError{Code: "PD001"}, ports.ErrCapacidadNoDisponible},
		{&pgconn.PgError{Code: "22023"}, ports.ErrValidacion},
		{&pgconn.PgError{Code: "23514"}, ports.ErrValidacion},
		{&pgconn.PgError{Code: "02000"}, ports.ErrNoEncontrado},
		{&pgconn.PgError{Code: "42501"}, ports.ErrAccesoDenegado},
		{&pgconn.PgError{Code: "40001"}, ports.ErrCapacidadNoDisponible},
		{&pgconn.PgError{Code: "57014"}, ports.ErrCapacidadNoDisponible},
		{&pgconn.PgError{Code: "55000"}, ports.ErrCapacidadNoDisponible},
		{&pgconn.PgError{Code: "08006"}, ports.ErrCapacidadNoDisponible},
		{io.ErrUnexpectedEOF, ports.ErrCapacidadNoDisponible},
		{context.DeadlineExceeded, context.DeadlineExceeded},
		{context.Canceled, context.Canceled},
	}
	for _, c := range casos {
		got := clasificarErrorSQL(c.err)
		if !errors.Is(got, c.want) {
			t.Fatalf("%v: obtenido %v, esperado %v", c.err, got, c.want)
		}
		// El texto del servidor nunca atraviesa el puerto.
		var pg *pgconn.PgError
		if errors.As(got, &pg) {
			t.Fatalf("%v: el error SQL original escapa del adaptador", c.err)
		}
	}
	if !errors.Is(clasificarErrorSQL(context.Canceled), ports.ErrCapacidadNoDisponible) {
		t.Fatal("la cancelación debe seguir siendo indisponibilidad del repositorio")
	}
}

func TestOrdenDenegacionFronteraCerrada(t *testing.T) {
	valida := OrdenDenegacionFrontera{CorrelacionRef: "corr_0123456789abcdef0123456789abcdef", Motivo: MotivoFronteraDenegado,
		Ruta: RutaFronteraConsulta, Metodo: "POST", ActorRef: "per:00000000-0000-4000-8000-000000000001"}
	if valida.Validar() != nil {
		t.Fatal("orden válida rechazada")
	}
	for _, ruta := range []string{RutaFronteraDescarga, RutaFronteraRegistroExterno, RutaFronteraOtra} {
		o := valida
		o.Ruta = ruta
		if o.Validar() != nil {
			t.Fatalf("ruta cerrada %s rechazada", ruta)
		}
	}
	for _, mutar := range []func(*OrdenDenegacionFrontera){
		func(o *OrdenDenegacionFrontera) { o.Motivo = "texto libre" },
		func(o *OrdenDenegacionFrontera) { o.Ruta = "/api/vec/documentos/otra" },
		func(o *OrdenDenegacionFrontera) { o.Metodo = "GET" },
		func(o *OrdenDenegacionFrontera) { o.CorrelacionRef = "corr_x" },
		func(o *OrdenDenegacionFrontera) { o.ActorRef = "persona con espacios" },
	} {
		o := valida
		mutar(&o)
		if o.Validar() == nil {
			t.Fatalf("orden inválida aceptada: %+v", o)
		}
	}
	var r *RegistradorFrontera
	if r.RegistrarDenegacion(context.Background(), valida) == nil || r.Preflight(context.Background()) == nil {
		t.Fatal("registrador nulo aceptado")
	}
}

// TestRegistradorFronteraPG18 usa el LOGIN auditor de probar_integracion_pg18.sh.
func TestRegistradorFronteraPG18(t *testing.T) {
	dsn := os.Getenv("VEC_DOCUMENTOS_PG18_AUDITOR_DSN")
	if dsn == "" {
		t.Skip("sin base PG18 desechable de Documentos")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	r, err := NuevoRegistradorFrontera(pool)
	if err != nil || r.Preflight(ctx) != nil {
		t.Fatalf("preflight del auditor: %v", err)
	}
	orden := OrdenDenegacionFrontera{CorrelacionRef: "corr_no_disponible", Motivo: MotivoFronteraAutenticacion,
		Ruta: RutaFronteraDescarga, Metodo: "POST"}
	if err := r.RegistrarDenegacion(ctx, orden); err != nil {
		t.Fatalf("registro: %v", err)
	}
	externa := orden
	externa.Ruta = RutaFronteraRegistroExterno
	if err := r.RegistrarDenegacion(ctx, externa); err != nil {
		t.Fatalf("registro de la ruta de registro externo: %v", err)
	}
	ejecutor := os.Getenv("VEC_DOCUMENTOS_PG18_DSN")
	if ejecutor == "" {
		return
	}
	poolEjecutor, err := pgxpool.New(ctx, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	defer poolEjecutor.Close()
	otro, _ := NuevoRegistradorFrontera(poolEjecutor)
	if otro.Preflight(ctx) == nil || otro.RegistrarDenegacion(ctx, orden) == nil {
		t.Fatal("el LOGIN ejecutor no debe registrar denegaciones")
	}
}
