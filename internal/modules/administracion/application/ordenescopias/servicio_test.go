package ordenescopias_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
	destino "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias"
	"vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias/sintetico"
	app "vec-diputacion-granada/internal/modules/administracion/application/ordenescopias"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/ordenescopias"
)

func datos() domain.Datos {
	return domain.Datos{Orden: "orden:uno", Operacion: "operacion:uno", Accion: "restaurar_conjunto", SolicitudSHA256: strings.Repeat("a", 64), Conjunto: "conjunto:uno", ManifiestoSHA256: strings.Repeat("b", 64), PreimagenSHA256: strings.Repeat("c", 64), Destino: "destino:uno", ProponentePersona: "persona:uno", AprobadorPersona: "persona:dos", Politica: "politica:uno", PoliticaSHA256: strings.Repeat("d", 64), DecisionV3: "decision:uno", DecisionSHA256: strings.Repeat("e", 64), ConsumoV3: strings.Repeat("f", 64), Auditoria: "auditoria:uno", Outbox: "outbox:uno", Epoca: "epoca:uno", Fence: 1, VersionCAS: 1, EmitidaEn: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), CaducaEn: time.Date(2026, 10, 1, 11, 0, 0, 0, time.UTC)}
}
func entorno(t *testing.T) (*app.Servicio, *sintetico.Entorno, domain.Orden) {
	t.Helper()
	o, err := domain.Nueva(datos())
	if err != nil {
		t.Fatal(err)
	}
	e, err := sintetico.Nuevo(o)
	if err != nil {
		t.Fatal(err)
	}
	s, err := app.Nuevo(e, cripto(t, 1), e, e)
	if err != nil {
		t.Fatal(err)
	}
	return s, e, o
}
func TestAceptacionConcurrenteUnicaYReplay(t *testing.T) {
	s, _, o := entorno(t)
	ctx := context.Background()
	sobre, err := s.Publicar(ctx, o, datos().EmitidaEn)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan puerto.Aceptacion, 32)
	errs := make(chan error, 32)
	for range 32 {
		wg.Add(1)
		go func() { defer wg.Done(); r, e := s.Aceptar(ctx, sobre, datos().EmitidaEn); results <- r; errs <- e }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	nuevas := 0
	recibo := ""
	for r := range results {
		if !r.Replay {
			nuevas++
		}
		if recibo != "" && recibo != r.Recibo {
			t.Fatal("receipt changed")
		}
		recibo = r.Recibo
	}
	if nuevas != 1 {
		t.Fatalf("new acceptances = %d", nuevas)
	}
}
func TestAlteracionRechazadaAntesAceptacion(t *testing.T) {
	s, _, o := entorno(t)
	ctx := context.Background()
	sobre, err := s.Publicar(ctx, o, datos().EmitidaEn)
	if err != nil {
		t.Fatal(err)
	}
	changes := map[string]func(*domain.Datos){"destino": func(d *domain.Datos) { d.Destino = "destino:otro" }, "persona": func(d *domain.Datos) { d.AprobadorPersona = "persona:otra" }, "epoca": func(d *domain.Datos) { d.Epoca = "epoca:otra" }, "fence": func(d *domain.Datos) { d.Fence++ }, "preimagen": func(d *domain.Datos) { d.PreimagenSHA256 = strings.Repeat("f", 64) }, "consumo": func(d *domain.Datos) { d.ConsumoV3 = strings.Repeat("9", 64) }, "solicitud": func(d *domain.Datos) { d.SolicitudSHA256 = strings.Repeat("f", 64) }}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			d := datos()
			change(&d)
			changed, err := domain.Nueva(d)
			if err != nil {
				t.Fatal(err)
			}
			altered := sobre
			altered.Orden = changed
			if _, err := s.Aceptar(ctx, altered, d.EmitidaEn); !errors.Is(err, app.ErrDenegada) {
				t.Fatal(err)
			}
		})
	}
	corrupt := sobre
	corrupt.Firma = append([]byte(nil), sobre.Firma...)
	corrupt.Firma[0] ^= 1
	if s.Verificar(ctx, corrupt, datos().EmitidaEn) == nil {
		t.Fatal("modified signature accepted")
	}
	another := cripto(t, 2)
	firma, err := another.Sellar(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	if cripto(t, 1).Verificar(ctx, o, firma) == nil {
		t.Fatal("other key accepted")
	}
	r, err := s.Aceptar(ctx, sobre, datos().EmitidaEn)
	if err != nil || r.Replay {
		t.Fatalf("invalid orders caused effects: %v %#v", err, r)
	}
}

type lectorOtro struct{ o domain.Orden }

func (l lectorOtro) LeerOrdenComprometida(context.Context, string) (domain.Orden, error) {
	return l.o, nil
}

type anchorRevocado struct{}

func (anchorRevocado) ValidarActual(context.Context, domain.Orden, time.Time) error {
	return errors.New("revoked")
}
func TestCommitDistintoCaducidadRevocacionYNulos(t *testing.T) {
	s, e, o := entorno(t)
	ctx := context.Background()
	sobre, err := s.Publicar(ctx, o, datos().EmitidaEn)
	if err != nil {
		t.Fatal(err)
	}
	for _, instante := range []time.Time{datos().EmitidaEn.Add(-time.Microsecond), datos().CaducaEn, time.Time{}} {
		if _, err := s.Aceptar(ctx, sobre, instante); err == nil {
			t.Fatal("invalid window")
		}
	}
	d := datos()
	d.Outbox = "outbox:otro"
	otro, _ := domain.Nueva(d)
	diferente, _ := app.Nuevo(lectorOtro{otro}, cripto(t, 1), e, e)
	if _, err := diferente.Publicar(ctx, o, d.EmitidaEn); !errors.Is(err, app.ErrCommit) {
		t.Fatal(err)
	}
	revoked, _ := app.Nuevo(e, cripto(t, 1), anchorRevocado{}, e)
	if _, err := revoked.Aceptar(ctx, sobre, d.EmitidaEn); err == nil {
		t.Fatal("revocation ignored")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if s.Verificar(canceled, sobre, d.EmitidaEn) == nil {
		t.Fatal("cancellation ignored")
	}
	if s.Verificar(nil, sobre, d.EmitidaEn) == nil {
		t.Fatal("nil context")
	}
	var typed *sintetico.Entorno
	if _, err := app.Nuevo(typed, cripto(t, 1), e, e); err == nil {
		t.Fatal("typed nil")
	}
}
func TestPersonaCanonicaDuplicadaYCompromisoInmutable(t *testing.T) {
	d := datos()
	d.AprobadorPersona = d.ProponentePersona
	if _, err := domain.Nueva(d); err == nil {
		t.Fatal("same person")
	}
	d = datos()
	o, _ := domain.Nueva(d)
	h, _ := o.SHA256()
	d.Destino = "destino:otra"
	copia, _ := o.Datos()
	copia.Destino = "destino:cambiado"
	b, _ := o.Bytes()
	b[0] ^= 1
	despues, _ := o.SHA256()
	if h != despues {
		t.Fatal("mutable commitment")
	}
}

func cripto(t *testing.T, k byte) *adapter.ProtectorOrden {
	t.Helper()
	p, e := destino.NuevoProtectorJWE([32]byte{k}, "clave:sintetica:orden", "version:1", 8192)
	if e != nil {
		t.Fatal(e)
	}
	c, e := adapter.NuevoProtectorOrden(p)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
