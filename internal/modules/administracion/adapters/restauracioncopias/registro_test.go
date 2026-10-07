package restauracioncopias

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	d "vec-diputacion-granada/internal/modules/administracion/domain/restauracioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/restauracioncopias"
)

func fixture(t *testing.T) (string, d.Sellada, p.Concesion) {
	t.Helper()
	dir := t.TempDir()
	if os.Chmod(dir, 0700) != nil {
		t.Fatal("chmod")
	}
	n := time.Date(2026, 10, 1, 18, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	s, e := d.Sellar(d.Propuesta{FormatoVersion: 1, Ref: "propuesta:1", ConjuntoRef: "copia:1", ConjuntoSHA256: h, DestinoRef: "destino:1", PreimagenSHA256: h, MotivoRef: "motivo:1", VentanaRef: "ventana:1", VentanaInicio: n, VentanaFin: n.Add(time.Hour), PoliticaRef: "politica:1", PoliticaSHA256: h, ProponentePersonaRef: "persona:1", Creada: n, Caduca: n.Add(time.Hour), DobleControl: true, Entorno: "sintetico_offline"})
	if e != nil {
		t.Fatal(e)
	}
	c := p.Concesion{PersonaRef: "persona:1", Ref: "decision:1", Acceso: p.Acceso{Accion: p.Proponer, DestinoRef: "destino:1", PropuestaSHA256: s.SHA256}, Caduca: s.Propuesta.Caduca}
	return dir, s, c
}
func TestDurableReinicioCASConcurrencia(t *testing.T) {
	dir, s, c := fixture(t)
	ctx := context.Background()
	r, e := Abrir(dir, func(context.Context, p.Concesion) error { return nil })
	if e != nil {
		t.Fatal(e)
	}
	first, e := r.Crear(ctx, s, c)
	if e != nil {
		t.Fatal(e)
	}
	again, e := r.Crear(ctx, s, c)
	if e != nil || again != first {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	c.Acceso.Accion = p.Revisar
	c.PersonaRef = "persona:2"
	rev, e := s.Revisar(c.PersonaRef, s.Propuesta.Creada)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := Abrir(dir, func(context.Context, p.Concesion) error { return nil })
			if e != nil {
				results <- e
				return
			}
			_, e = r.RevisarCAS(ctx, s.Propuesta.Ref, 1, s.SHA256, rev, c)
			results <- e
			if e = r.Close(); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for e := range results {
		if e == nil {
			successes++
		} else if !errors.Is(e, ErrConflicto) {
			t.Fatal(e)
		}
	}
	if successes != 1 {
		t.Fatal(successes)
	}
	r, e = Abrir(dir, func(context.Context, p.Concesion) error { return nil })
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := r.Close(); e != nil {
			t.Error(e)
		}
	}()
	read, e := r.Leer(ctx, s.Propuesta.Ref, c)
	if e != nil || read.Version != 2 || read.Revision == nil {
		t.Fatal(read, e)
	}
	if e = r.Cercar(ctx, s.Propuesta.Ref, 2, s.SHA256, "exclusion:1", c); e == nil {
		t.Fatal("offline no cerca")
	}
}
func TestRevocacionYJournalAlterado(t *testing.T) {
	dir, s, c := fixture(t)
	denied := errors.New("revocada")
	r, e := Abrir(dir, func(context.Context, p.Concesion) error { return denied })
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.Crear(context.Background(), s, c); !errors.Is(e, denied) {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(dir+"/propuestas.v1.jsonl", []byte("{}"), 0600); e != nil {
		t.Fatal(e)
	}
	r, e = Abrir(dir, func(context.Context, p.Concesion) error { return nil })
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := r.Close(); e != nil {
			t.Error(e)
		}
	}()
	if _, e = r.Crear(context.Background(), s, c); !errors.Is(e, ErrRegistro) {
		t.Fatal(e)
	}
}
