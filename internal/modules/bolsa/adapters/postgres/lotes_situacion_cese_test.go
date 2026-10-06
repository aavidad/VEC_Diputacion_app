package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Los lotes rechazan entradas inválidas o desmesuradas antes de tocar la base.
func TestLotesSituacionYCeseFallanCerradoSinEnviar(t *testing.T) {
	ctx := context.Background()
	var situaciones *RepositorioSituacionParticipacionPostgreSQL
	if _, err := situaciones.SituacionesVigentes(ctx, []string{"participacion:1"}); !errors.Is(err, ports.ErrSituacionParticipacionNoDisponible) {
		t.Fatalf("repositorio nulo: %v", err)
	}
	situaciones = &RepositorioSituacionParticipacionPostgreSQL{pool: &pgxpool.Pool{}}
	if _, err := situaciones.SituacionesVigentes(ctx, []string{"participacion:1", ""}); !errors.Is(err, ports.ErrSituacionParticipacionNoDisponible) {
		t.Fatalf("referencia vacía: %v", err)
	}
	if _, err := situaciones.SituacionesVigentes(ctx, make([]string, maximoLoteSituaciones+1)); !errors.Is(err, ports.ErrSituacionParticipacionNoDisponible) {
		t.Fatalf("lote desmesurado: %v", err)
	}
	if vacio, err := situaciones.SituacionesVigentes(ctx, nil); err != nil || len(vacio) != 0 {
		t.Fatalf("lote vacío: %v %v", vacio, err)
	}
	ceses := &ConsultaEstadoCesePostgreSQL{pool: &pgxpool.Pool{}}
	corte := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	for _, refs := range [][]string{{" participacion:1"}, {""}, make([]string, maximoLoteEstadosCese+1)} {
		if _, err := ceses.ConsultarEstadosCese(ctx, refs, corte); !errors.Is(err, ports.ErrConsultaEstadoCeseNoDisponible) {
			t.Fatalf("lote de ceses inválido aceptado: %v", err)
		}
	}
	if _, err := ceses.ConsultarEstadosCese(ctx, []string{"participacion:1"}, time.Time{}); !errors.Is(err, ports.ErrConsultaEstadoCeseNoDisponible) {
		t.Fatalf("corte vacío: %v", err)
	}
}

// Con una base real (VEC_BOLSA_LOTES_PG_DSN, rol ejecutor de Bolsa), el lote
// devuelve exactamente lo mismo que la lectura individual.
func TestLotesSituacionYCeseCoincidenConLecturaIndividualPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_BOLSA_LOTES_PG_DSN")
	if dsn == "" {
		t.Skip("sin VEC_BOLSA_LOTES_PG_DSN")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var refs []string
	filas, err := pool.Query(ctx, `SELECT participacion_ref FROM vec_bolsa_llamamientos.listar_constituciones_v1() c CROSS JOIN LATERAL vec_bolsa_llamamientos.listar_entradas_constitucion_v1(c.instantanea_ref, c.version_instantanea) e LIMIT 20000`)
	if err != nil {
		t.Fatal(err)
	}
	for filas.Next() {
		var ref string
		if err := filas.Scan(&ref); err != nil {
			t.Fatal(err)
		}
		refs = append(refs, ref)
	}
	filas.Close()
	if len(refs) == 0 {
		t.Skip("sin participaciones constituidas")
	}
	situaciones, _ := NuevoRepositorioSituacionParticipacionPostgreSQL(pool)
	ceses, _ := NuevaConsultaEstadoCesePostgreSQL(pool)
	corte := time.Now()
	loteSituaciones, err := situaciones.SituacionesVigentes(ctx, refs)
	if err != nil {
		t.Fatal(err)
	}
	loteCeses, err := ceses.ConsultarEstadosCese(ctx, refs, corte)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d participaciones, %d con cese presente", len(refs), len(loteCeses))
	for _, ref := range refs {
		individual, err := situaciones.SituacionVigente(ctx, ref)
		if err != nil {
			t.Fatal(err)
		}
		enLote := loteSituaciones[ref]
		if enLote.Situacion != individual.Situacion || !enLote.Desde.Equal(individual.Desde) || !mismaFechaDisponiblePostgreSQL(enLote.FechaDisponible, individual.FechaDisponible) {
			t.Fatalf("situación distinta para %s: %+v frente a %+v", ref, enLote, individual)
		}
		estado, presente, err := ceses.ConsultarEstadoCese(ctx, ref, corte)
		if err != nil {
			t.Fatal(err)
		}
		estadoLote, presenteLote := loteCeses[ref]
		if presente != presenteLote || !estado.FechaEfecto.Equal(estadoLote.FechaEfecto) || !estado.DisponibleDesde.Equal(estadoLote.DisponibleDesde) ||
			estado.EnRestriccion != estadoLote.EnRestriccion || estado.TrabajoCesado != estadoLote.TrabajoCesado {
			t.Fatalf("cese distinto para %s", ref)
		}
	}
}
