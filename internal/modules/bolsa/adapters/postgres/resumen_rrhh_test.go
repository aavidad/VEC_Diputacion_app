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

func TestLectorResumenBolsasFallaCerradoSinBase(t *testing.T) {
	var l *LectorResumenBolsasPostgreSQL
	if _, _, err := l.LeerResumen(context.Background(), time.Now()); !errors.Is(err, ports.ErrResumenBolsasNoDisponible) {
		t.Fatalf("lector nulo: %v", err)
	}
	l = &LectorResumenBolsasPostgreSQL{pool: &pgxpool.Pool{}}
	if _, _, err := l.LeerResumen(context.Background(), time.Time{}); !errors.Is(err, ports.ErrResumenBolsasNoDisponible) {
		t.Fatalf("corte vacío: %v", err)
	}
	if _, err := NuevoLectorResumenBolsasPostgreSQL(nil); err == nil {
		t.Fatal("pool nulo aceptado")
	}
}

// Con Bolsa 000082 instalada (VEC_BOLSA_LOTES_PG_DSN, rol ejecutor), las
// lecturas de conjunto coinciden fila a fila con las individuales.
func TestLectorResumenBolsasCoincideConLecturasIndividualesPostgreSQL(t *testing.T) {
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
	lector, _ := NuevoLectorResumenBolsasPostgreSQL(pool)
	situaciones, _ := NuevoRepositorioSituacionParticipacionPostgreSQL(pool)
	ceses, _ := NuevaConsultaEstadoCesePostgreSQL(pool)
	orden, _ := NuevaConsultaOrdenVigentePostgreSQL(pool)
	corte := time.Now()
	filas, politicas, err := lector.LeerResumen(ctx, corte)
	if err != nil || len(filas) == 0 {
		t.Fatalf("resumen: %d filas, %v", len(filas), err)
	}
	refs := make([]string, 0, len(filas))
	for _, fila := range filas {
		refs = append(refs, fila.ParticipacionRef)
	}
	individuales, err := situaciones.SituacionesVigentes(ctx, refs)
	if err != nil {
		t.Fatal(err)
	}
	cesesIndividuales, err := ceses.ConsultarEstadosCese(ctx, refs, corte)
	if err != nil {
		t.Fatal(err)
	}
	conCese := 0
	for _, fila := range filas {
		esperada := individuales[fila.ParticipacionRef]
		if fila.Situacion == nil || fila.Situacion.Situacion != esperada.Situacion || !fila.Situacion.Desde.Equal(esperada.Desde) ||
			!mismaFechaDisponiblePostgreSQL(fila.Situacion.FechaDisponible, esperada.FechaDisponible) {
			t.Fatalf("situación distinta para %s", fila.ParticipacionRef)
		}
		estado, presente := cesesIndividuales[fila.ParticipacionRef]
		if presente != (fila.Cese != nil) || (presente && (!estado.FechaEfecto.Equal(fila.Cese.FechaEfecto) ||
			!estado.DisponibleDesde.Equal(fila.Cese.DisponibleDesde) || estado.EnRestriccion != fila.Cese.EnRestriccion || estado.TrabajoCesado != fila.Cese.TrabajoCesado)) {
			t.Fatalf("cese distinto para %s", fila.ParticipacionRef)
		}
		if presente {
			conCese++
		}
	}
	if len(politicas) == 0 {
		t.Fatal("sin políticas")
	}
	for bolsa, politica := range politicas {
		vigente, err := orden.ConsultarOrdenVigente(ctx, bolsa)
		if err != nil {
			t.Fatal(err)
		}
		esperada := vigente.Politica
		if esperada.PoliticaRef != politica.PoliticaRef || esperada.Version != politica.Version || esperada.TipoLista != politica.TipoLista ||
			esperada.Reposicion != politica.Reposicion || esperada.Provisional != politica.Provisional || !esperada.VigenteDesde.Equal(politica.VigenteDesde) {
			t.Fatalf("política distinta para %s: %+v frente a %+v", bolsa, politica, esperada)
		}
	}
	t.Logf("%d participaciones (%d con cese), %d políticas", len(filas), conCese, len(politicas))
}
