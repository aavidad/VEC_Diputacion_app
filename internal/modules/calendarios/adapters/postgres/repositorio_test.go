package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/calendarios/application"
	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

func TestRepositorioRechazaDependenciasYConsultasInvalidas(t *testing.T) {
	if _, err := NuevoRepositorio(nil); !errors.Is(err, ErrNoDisponible) {
		t.Fatal("sin pool no se construye")
	}
	var pool *pgxpool.Pool
	if _, err := NuevoRepositorio(pool); !errors.Is(err, ErrNoDisponible) {
		t.Fatal("un pool nulo tipado tampoco")
	}
	r := &Repositorio{pool: &pgxpool.Pool{}}
	for _, c := range []ports.ConsultaVersiones{
		{Anio: 2026, ConocidoEn: time.Now()},
		{Anio: 2026, Ambitos: []domain.Ambito{{Tipo: "otro", Ref: "es"}}, ConocidoEn: time.Now()},
		{Anio: 2026, Ambitos: []domain.Ambito{{Tipo: domain.AmbitoNacional, Ref: "es"}}},
	} {
		if _, err := r.VersionesVigentes(context.Background(), c); !errors.Is(err, ErrNoDisponible) {
			t.Fatalf("%+v: %v", c, err)
		}
	}
}

type relojSistema struct{}

func (relojSistema) Ahora() time.Time { return time.Now() }

// Requiere un PostgreSQL desechable con roles, 000001 y 000002 instalados y
// un login miembro de vec_calendarios_lector (ver pruebas_sql).
func TestRepositorioPostgreSQLReal(t *testing.T) {
	dsn := os.Getenv("VEC_CALENDARIOS_TEST_PG_URL")
	if dsn == "" {
		t.Skip("PostgreSQL desechable de Calendarios no configurado")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := NuevoRepositorio(pool)
	if err != nil {
		t.Fatal(err)
	}
	s, err := application.NuevoServicio(repo, relojSistema{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CalendarioCentro(ctx, ports.SolicitudCalendarioCentro{CentroRef: "centro-530", Anio: 2026})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Dias) != 365 || c.Resumen.FestivosOficiales != 14 || c.Resumen.NoLaborablesCentro != 2 || len(c.Versiones) != 4 {
		t.Fatalf("calendario real: %+v", c.Resumen)
	}
	centros, err := s.Centros(ctx, 2026, time.Time{})
	if err != nil || len(centros) != 4 {
		t.Fatalf("centros: %d %v", len(centros), err)
	}
	r, err := s.CalcularPlazo(ctx, ports.SolicitudCalculoPlazo{
		Inicio: mustFecha(t, "2026-10-23"), Unidad: domain.UnidadDiasNaturales, Cantidad: 10, MunicipioSede: "municipio:ine:18087",
	})
	if err != nil || r.Vencimiento.String() != "2026-11-03" || !r.Prorrogado {
		t.Fatalf("plazo real: %+v %v", r, err)
	}
	_, err = s.CalcularPlazo(ctx, ports.SolicitudCalculoPlazo{
		Inicio: mustFecha(t, "2026-12-28"), Unidad: domain.UnidadDiasHabiles, Cantidad: 5, MunicipioSede: "municipio:ine:18087",
	})
	if !errors.Is(err, domain.ErrCalendarioNoCubre) {
		t.Fatalf("2027 sin publicar debe fallar cerrado: %v", err)
	}
}

func mustFecha(t *testing.T, s string) domain.FechaCivil {
	t.Helper()
	f, err := domain.ParsearFechaCivil(s)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
