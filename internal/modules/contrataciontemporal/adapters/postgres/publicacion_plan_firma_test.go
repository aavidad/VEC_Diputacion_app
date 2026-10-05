package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
)

var _ plannominal.PublicacionAutorizada = (*PublicacionPlanFirmaPostgreSQL)(nil)

type filaPublicacionPrueba struct {
	bruto string
	err   error
}

func (f filaPublicacionPrueba) Scan(destino ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destino[0].(*[]byte)) = []byte(f.bruto)
	return nil
}

type poolPublicacionPrueba struct {
	fila       filaPublicacionPrueba
	llamadas   int
	argumentos []any
	sql        string
}

func (p *poolPublicacionPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	p.llamadas++
	p.sql, p.argumentos = sql, args
	return p.fila
}

func catalogoPublicadoPlanPrueba() vd.CatalogoConfigurable {
	return vd.CatalogoConfigurable{ID: "ct.plan.firma.sintetico", Version: 1, Revision: 2, ModuloID: "contratacion_temporal",
		Estado: vd.EstadoCatalogoPublicado, PublicadoEn: time.Date(2026, 10, 3, 11, 0, 0, 0, time.UTC)}
}

// La respuesta de CT178 debe coincidir con la publicación del fichero: fecha y
// revisión. Cualquier divergencia, rechazo o caída deniega el plan.
func TestPublicacionPlanFirmaCotejaFechaYRevision(t *testing.T) {
	c := catalogoPublicadoPlanPrueba()
	sha := strings.Repeat("a", 64)
	en := c.PublicadoEn.Add(time.Hour)
	ok := `{"publicada_en":"2026-10-03T11:00:00.000000Z","revision":2}`
	pool := &poolPublicacionPrueba{fila: filaPublicacionPrueba{bruto: ok}}
	p, err := NuevaPublicacionPlanFirmaPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.ComprobarPublicacionPlanFirmaV2(t.Context(), c, sha, en); err != nil {
		t.Fatalf("publicación vigente rechazada: %v", err)
	}
	if pool.sql != comprobarPlanFirmaPublicadoSQL178 || len(pool.argumentos) != 3 || pool.argumentos[0] != c.ID ||
		pool.argumentos[1] != int64(1) || pool.argumentos[2] != sha {
		t.Fatalf("argumentos inesperados: %v", pool.argumentos)
	}
	for nombre, bruto := range map[string]string{
		"otra fecha":     `{"publicada_en":"2026-10-03T11:00:01.000000Z","revision":2}`,
		"otra revisión":  `{"publicada_en":"2026-10-03T11:00:00.000000Z","revision":3}`,
		"campo de más":   `{"publicada_en":"2026-10-03T11:00:00.000000Z","revision":2,"x":1}`,
		"fecha sin zona": `{"publicada_en":"2026-10-03T11:00:00","revision":2}`,
	} {
		pool.fila = filaPublicacionPrueba{bruto: bruto}
		if err := p.ComprobarPublicacionPlanFirmaV2(t.Context(), c, sha, en); !errors.Is(err, ct.ErrPlanCompetenciaFirmaV2) {
			t.Fatalf("%s aceptada: %v", nombre, err)
		}
	}
	pool.fila = filaPublicacionPrueba{err: errors.New("42501")}
	if err := p.ComprobarPublicacionPlanFirmaV2(t.Context(), c, sha, en); !errors.Is(err, ct.ErrPlanCompetenciaFirmaV2) {
		t.Fatalf("rechazo SQL aceptado: %v", err)
	}
	// Entradas imposibles no llegan a la base.
	pool.fila, pool.llamadas = filaPublicacionPrueba{bruto: ok}, 0
	malos := map[string]func(*vd.CatalogoConfigurable, *string, *time.Time){
		"sha corto":         func(_ *vd.CatalogoConfigurable, s *string, _ *time.Time) { *s = "abc" },
		"id mayúsculas":     func(c *vd.CatalogoConfigurable, _ *string, _ *time.Time) { c.ID = "CT.plan" },
		"otro módulo":       func(c *vd.CatalogoConfigurable, _ *string, _ *time.Time) { c.ModuloID = "bolsa" },
		"publicado después": func(_ *vd.CatalogoConfigurable, _ *string, e *time.Time) { *e = c.PublicadoEn.Add(-time.Second) },
		"sin publicación":   func(c *vd.CatalogoConfigurable, _ *string, _ *time.Time) { c.PublicadoEn = time.Time{} },
		"submicrosegundo": func(c *vd.CatalogoConfigurable, _ *string, _ *time.Time) {
			c.PublicadoEn = c.PublicadoEn.Add(123456700 * time.Nanosecond)
		},
	}
	for nombre, mutar := range malos {
		cc, s, e := c, sha, en
		mutar(&cc, &s, &e)
		if err := p.ComprobarPublicacionPlanFirmaV2(t.Context(), cc, s, e); !errors.Is(err, ct.ErrPlanCompetenciaFirmaV2) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
	if pool.llamadas != 0 {
		t.Fatalf("entradas imposibles consultaron la base %d veces", pool.llamadas)
	}
	cancelado, cancelar := context.WithCancel(t.Context())
	cancelar()
	if err := p.ComprobarPublicacionPlanFirmaV2(cancelado, c, sha, en); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado: %v", err)
	}
	if _, err := NuevaPublicacionPlanFirmaPostgreSQL(nil); !errors.Is(err, ct.ErrPlanCompetenciaFirmaV2) {
		t.Fatal("pool nulo admitido")
	}
}
