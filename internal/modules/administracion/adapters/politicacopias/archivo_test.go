package politicacopias

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
)

func politica() d.Politica {
	return d.Politica{Formato: 1, Referencia: "politica:test", Destino: "destino:test", ZonaHoraria: "Europe/Madrid", FechaInicial: "2026-01-01", CadaDias: 1, Ventana: d.Ventana{Inicio: "02:30", Fin: "04:00"}, Retencion: d.Retencion{ConservarMinimo: 1, EdadMaximaDias: 30}}
}
func attribution() p.Atribucion {
	return p.Atribucion{Actor: "actor:test", Revisor: "actor:revisor", Correlacion: "correlacion:test"}
}
func accept(context.Context) error { return nil }
func TestJournalRestartCASAndHistory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "control")
	a, e := Abrir(dir)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	first, e := a.Guardar(ctx, 0, politica(), attribution(), now, accept)
	if e != nil {
		t.Fatal(e)
	}
	_ = a.Cerrar()
	a, e = Abrir(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = a.Cerrar() }()
	got, e := a.Actual(ctx)
	if e != nil || got.RegistroSHA256 != first.RegistroSHA256 || got.Version != 1 {
		t.Fatalf("restart %v %v", got, e)
	}
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	success, conflict := 0, 0
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q := politica()
			q.CadaDias = 2
			_, e := a.Guardar(ctx, 1, q, attribution(), now.Add(time.Minute), accept)
			mu.Lock()
			defer mu.Unlock()
			if e == nil {
				success++
			} else if errors.Is(e, d.ErrVersion) {
				conflict++
			} else {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if success != 1 || conflict != 15 {
		t.Fatalf("success %d conflict %d", success, conflict)
	}
	history, e := a.Historia(ctx)
	if e != nil || len(history) != 2 || history[0].RegistroSHA256 != first.RegistroSHA256 || history[1].AnteriorSHA256 != first.RegistroSHA256 {
		t.Fatalf("history %v %v", history, e)
	}
}
func TestDeniedRecheckNoPartialAppendAndCorruption(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "control")
	a, e := Abrir(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = a.Cerrar() }()
	ctx := context.Background()
	now := time.Now().UTC()
	_, e = a.Guardar(ctx, 0, politica(), attribution(), now, func(context.Context) error { return d.ErrDenegado })
	if !errors.Is(e, d.ErrDenegado) {
		t.Fatal(e)
	}
	h, e := a.Historia(ctx)
	if e != nil || len(h) != 0 {
		t.Fatal("denied update persisted")
	}
	actor := attribution()
	actor.Revisor = actor.Actor
	if _, e = a.Guardar(ctx, 0, politica(), actor, now, accept); !errors.Is(e, d.ErrDenegado) {
		t.Fatal("same person review accepted")
	}
	_, e = a.Guardar(ctx, 0, politica(), attribution(), now, accept)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "00000000000000000001.json"), []byte("{}"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Historia(ctx); !errors.Is(e, d.ErrHistoria) {
		t.Fatalf("corruption %v", e)
	}
}
func TestJournalLockCancellationSymlinksAndPending(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "control")
	a, e := Abrir(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = a.Cerrar() }()
	if e = os.WriteFile(filepath.Join(dir, "pendiente"), []byte("interrupted"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Guardar(context.Background(), 0, politica(), attribution(), time.Now().UTC(), accept); e != nil {
		t.Fatal(e)
	}
	unlock, e := a.bloqueo(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, e = a.Actual(ctx)
	unlock()
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("lock %v", e)
	}
	other := t.TempDir()
	link := filepath.Join(other, "link")
	if e = os.Symlink(dir, link); e != nil {
		t.Fatal(e)
	}
	if _, e = Abrir(link); e == nil {
		t.Fatal("symlink root accepted")
	}
	file := filepath.Join(dir, "00000000000000000001.json")
	if e = os.Remove(file); e != nil {
		t.Fatal(e)
	}
	if e = os.Symlink(filepath.Join(dir, "control.lock"), file); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Historia(context.Background()); e == nil {
		t.Fatal("symlink record accepted")
	}
}
func TestStrictExternalJSON(t *testing.T) {
	for _, in := range []string{`{"formato":1,"formato":2}`, `{"Formato":1}`, `{"unknown":2}`, `{} {}`, strings.Repeat("[", 34) + strings.Repeat("]", 34)} {
		var target d.Politica
		if Decodificar(strings.NewReader(in), &target) == nil {
			t.Fatalf("accepted %s", in)
		}
	}
}

func TestDeletionOfCompleteLastVersionBlocksJournal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "control")
	a, e := Abrir(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = a.Cerrar() }()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, e = a.Guardar(ctx, 0, politica(), attribution(), now, accept); e != nil {
		t.Fatal(e)
	}
	q := politica()
	q.CadaDias = 2
	if _, e = a.Guardar(ctx, 1, q, attribution(), now.Add(time.Second), accept); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(filepath.Join(dir, "00000000000000000002.json")); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Historia(ctx); !errors.Is(e, d.ErrHistoria) {
		t.Fatal("shortened history accepted", e)
	}
	if _, e = a.Guardar(ctx, 1, q, attribution(), now.Add(time.Minute), accept); !errors.Is(e, d.ErrHistoria) {
		t.Fatal("lost history overwritten", e)
	}
}
