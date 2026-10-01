package politicacopias

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
	a "vec-diputacion-granada/internal/modules/administracion/adapters/politicacopias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
)

type testClock struct{ now time.Time }

func (c *testClock) Ahora() time.Time { return c.now }

type testAuthority struct {
	deny, revoked bool
	reviewer      string
	requests      []p.Intencion
}

func (a *testAuthority) Autorizar(_ context.Context, i p.Intencion) (p.Atribucion, error) {
	a.requests = append(a.requests, i)
	if a.deny {
		return p.Atribucion{}, d.ErrDenegado
	}
	return p.Atribucion{Actor: "actor:test", Revisor: a.reviewer, Correlacion: "correlacion:test"}, nil
}
func (a *testAuthority) Revalidar(_ context.Context, _ p.Intencion, _ p.Atribucion) error {
	if a.revoked {
		return d.ErrDenegado
	}
	return nil
}

type testExecutor struct {
	seen  map[string]string
	calls int
	fail  bool
}

func (r *testExecutor) Reservar(_ context.Context, v p.Reserva, _ p.Atribucion) (p.ReciboReserva, error) {
	op, exists := r.seen[v.Clave]
	if !exists {
		op = "operacion:test"
		r.seen[v.Clave] = op
	}
	return p.ReciboReserva{Operacion: op, Replay: exists}, nil
}
func (r *testExecutor) Ejecutar(_ context.Context, _ p.Reserva, _ p.ReciboReserva, _ p.Atribucion) error {
	r.calls++
	if r.fail {
		return d.ErrDependencia
	}
	return nil
}
func setup(t *testing.T) (Servicio, *testAuthority, *testClock, *testExecutor) {
	t.Helper()
	store, e := a.Abrir(filepath.Join(t.TempDir(), "control"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = store.Cerrar() })
	authority := &testAuthority{reviewer: "actor:revisor"}
	clock := &testClock{time.Date(2026, 10, 1, 0, 40, 0, 0, time.UTC)}
	executor := &testExecutor{seen: map[string]string{}}
	return Servicio{Repositorio: store, Autoridad: authority, Reloj: clock, Reservador: executor, Ejecutor: executor}, authority, clock, executor
}
func config() d.Politica {
	return d.Politica{Formato: 1, Referencia: "politica:test", Destino: "destino:test", ZonaHoraria: "Europe/Madrid", FechaInicial: "2026-01-01", CadaDias: 1, Ventana: d.Ventana{Inicio: "02:30", Fin: "04:00"}, Retencion: d.Retencion{ConservarMinimo: 1, EdadMaximaDias: 30}}
}
func TestFailClosedAndCurrentRevocation(t *testing.T) {
	s, authority, _, _ := setup(t)
	s.Autoridad = nil
	if _, e := s.Configurar(context.Background(), 0, config()); !errors.Is(e, d.ErrDenegado) {
		t.Fatal(e)
	}
	s.Autoridad = authority
	authority.revoked = true
	if _, e := s.Configurar(context.Background(), 0, config()); !errors.Is(e, d.ErrDenegado) {
		t.Fatal(e)
	}
	h, e := s.Repositorio.Historia(context.Background())
	if e != nil || len(h) != 0 {
		t.Fatal("revoked persisted")
	}
	authority.revoked = false
	if _, e = s.Configurar(context.Background(), 0, config()); e != nil {
		t.Fatal(e)
	}
	authority.deny = true
	if _, e = s.Consultar(context.Background()); !errors.Is(e, d.ErrDenegado) {
		t.Fatal("denied read")
	}
}
func TestDoubleControlCannotBeLoweredUnilaterally(t *testing.T) {
	s, authority, _, _ := setup(t)
	ctx := context.Background()
	if _, e := s.Configurar(ctx, 0, config()); e != nil {
		t.Fatal(e)
	}
	policy := config()
	b := false
	policy.Retencion.DobleControl = &b
	authority.reviewer = ""
	if _, e := s.Configurar(ctx, 1, policy); !errors.Is(e, d.ErrDenegado) {
		t.Fatal("old double control bypass", e)
	}
	authority.reviewer = "actor:revisor"
	if _, e := s.Configurar(ctx, 1, policy); e != nil {
		t.Fatal(e)
	}
	authority.reviewer = ""
	policy.CadaDias = 2
	if _, e := s.Configurar(ctx, 2, policy); e != nil {
		t.Fatal("explicit single-control policy unavailable", e)
	}
}
func TestScheduleReservationReplayAndWindow(t *testing.T) {
	s, authority, clock, executor := setup(t)
	ctx := context.Background()
	if _, e := s.Configurar(ctx, 0, config()); e != nil {
		t.Fatal(e)
	}
	when := time.Date(2026, 10, 1, 0, 30, 0, 0, time.UTC)
	first, e := s.Ejecutar(ctx, 1, when)
	if e != nil || first.Replay {
		t.Fatal(first, e)
	}
	second, e := s.Ejecutar(ctx, 1, when)
	if e != nil || !second.Replay || first.Operacion != second.Operacion || len(executor.seen) != 1 {
		t.Fatal(second, e)
	}
	// Replays still revalidate authority; the test executor only records invocations,
	// it does not assert actual capture idempotence or backup validity.
	authority.revoked = true
	if _, e = s.Ejecutar(ctx, 1, when); !errors.Is(e, d.ErrDenegado) || executor.calls != 2 {
		t.Fatal("revoked schedule executed", e)
	}
	authority.revoked = false
	clock.now = when.Add(3 * time.Hour)
	if _, e = s.Ejecutar(ctx, 1, when); !errors.Is(e, d.ErrVentana) {
		t.Fatal(e)
	}
	clock.now = when
	if _, e = s.Ejecutar(ctx, 2, when); !errors.Is(e, d.ErrVersion) {
		t.Fatal(e)
	}
	if _, e = s.Ejecutar(ctx, 1, when.Add(time.Minute)); !errors.Is(e, d.ErrVentana) {
		t.Fatal("arbitrary event accepted", e)
	}
}
func TestReadAuthorizationBindsEveryPolicyVersion(t *testing.T) {
	s, authority, _, _ := setup(t)
	ctx := context.Background()
	if _, e := s.Configurar(ctx, 0, config()); e != nil {
		t.Fatal(e)
	}
	q := config()
	q.Destino = "destino:otro"
	if _, e := s.Configurar(ctx, 1, q); e != nil {
		t.Fatal(e)
	}
	authority.requests = nil
	h, e := s.Consultar(ctx)
	if e != nil || len(h) != 2 || len(authority.requests) != 2 {
		t.Fatal(h, e)
	}
	for i, intent := range authority.requests {
		if intent.Politica != h[i].Politica.Referencia || intent.Destino != h[i].Politica.Destino || intent.PoliticaSHA256 != h[i].SHA256 || intent.VersionEsperada != h[i].Version {
			t.Fatal("unscoped read", intent)
		}
	}
}

type testNotifier struct {
	fail  bool
	calls int
}

func (n *testNotifier) NotificarFallo(context.Context, p.FalloAgenda) error {
	n.calls++
	if n.fail {
		return d.ErrAviso
	}
	return nil
}
func TestExecutionAndNotificationFailureRemainFailure(t *testing.T) {
	s, _, _, executor := setup(t)
	ctx := context.Background()
	if _, e := s.Configurar(ctx, 0, config()); e != nil {
		t.Fatal(e)
	}
	notifier := &testNotifier{fail: true}
	s.Notificador = notifier
	executor.fail = true
	receipt, e := s.Ejecutar(ctx, 1, time.Date(2026, 10, 1, 0, 30, 0, 0, time.UTC))
	if !errors.Is(e, d.ErrDependencia) || !errors.Is(e, d.ErrAviso) || receipt.AvisoFallo != "fallido" || notifier.calls != 1 {
		t.Fatalf("%+v %v", receipt, e)
	}
	notifier.fail = false
	receipt, e = s.Ejecutar(ctx, 1, time.Date(2026, 10, 1, 0, 30, 0, 0, time.UTC))
	if !errors.Is(e, d.ErrDependencia) || receipt.AvisoFallo != "emitido_sin_acuse" || !receipt.Replay {
		t.Fatalf("%+v %v", receipt, e)
	}
}
