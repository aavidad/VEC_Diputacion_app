package restauracioncopias

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
	d "vec-diputacion-granada/internal/modules/administracion/domain/restauracioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/restauracioncopias"
)

type reloj struct{ n time.Time }

func (r *reloj) Ahora() time.Time { return r.n }

type autoridad struct{ revocada bool }

func (a *autoridad) AutorizarActual(_ context.Context, q p.Acceso) (p.Concesion, error) {
	if a.revocada {
		return p.Concesion{}, ErrAutoridad
	}
	persona := "persona:1"
	if q.SesionRef == "sesion:2" {
		persona = "persona:2"
	}
	return p.Concesion{PersonaRef: persona, Ref: "decision:1", Acceso: q, Caduca: time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC)}, nil
}
func (a *autoridad) RevalidarPersona(context.Context, string, p.Accion, string, string) error {
	if a.revocada {
		return ErrAutoridad
	}
	return nil
}

type observador struct{ o p.Observacion }

func (o *observador) ObservarActual(context.Context, string, string, string) (p.Observacion, error) {
	return o.o, nil
}

type registro struct {
	r      p.Registro
	cercas int
}

func (r *registro) Crear(_ context.Context, s d.Sellada, _ p.Concesion) (p.Registro, error) {
	r.r = p.Registro{Propuesta: s, Version: 1}
	return r.r, nil
}
func (r *registro) Leer(context.Context, string, p.Concesion) (p.Registro, error) { return r.r, nil }
func (r *registro) RevisarCAS(_ context.Context, _ string, v uint64, h string, rev d.Revision, _ p.Concesion) (p.Registro, error) {
	if r.r.Version != v || r.r.Propuesta.SHA256 != h {
		return p.Registro{}, ErrCAS
	}
	r.r.Version++
	r.r.Revision = &rev
	return r.r, nil
}
func (r *registro) Cercar(context.Context, string, uint64, string, string, p.Concesion) error {
	r.cercas++
	return nil
}
func servicioFixture(t *testing.T) (Servicio, *autoridad, *observador, *registro) {
	t.Helper()
	b, e := os.ReadFile("../../../../../cmd/vec-copias-comprobar/testdata/compatible.json")
	if e != nil {
		t.Fatal(e)
	}
	var o p.Observacion
	if e = json.Unmarshal(b, &o); e != nil {
		t.Fatal(e)
	}
	o.PreimagenSHA256 = strings.Repeat("a", 64)
	o.ConjuntoAutenticado = true
	o.PoliticaAutenticada = true
	o.VentanaRef = "ventana:1"
	a := &autoridad{}
	obs := &observador{o}
	reg := &registro{}
	s := Servicio{a, obs, reg, &reloj{time.Date(2026, 10, 1, 18, 0, 0, 0, time.UTC)}, d.Configuracion{Entorno: "operativo"}}
	q := Solicitud{Ref: "propuesta:1", ConjuntoRef: o.Manifiesto.ConjuntoRef, DestinoRef: o.Destino.Ref, MotivoRef: "motivo:1", VentanaRef: o.VentanaRef, VentanaInicio: s.Reloj.Ahora(), VentanaFin: s.Reloj.Ahora().Add(time.Hour), Caduca: s.Reloj.Ahora().Add(time.Hour)}
	if _, e = s.Proponer(context.Background(), "sesion:1", q); e != nil {
		t.Fatal(e)
	}
	return s, a, obs, reg
}
func TestBloqueosAntesDeCercar(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(Servicio, *autoridad, *observador, *registro)
		want error
	}{
		{"revocada", func(_ Servicio, a *autoridad, _ *observador, _ *registro) { a.revocada = true }, ErrAutoridad},
		{"preimagen", func(_ Servicio, _ *autoridad, o *observador, _ *registro) {
			o.o.PreimagenSHA256 = strings.Repeat("b", 64)
		}, ErrPreimagen},
		{"copia_previa_ausente", func(Servicio, *autoridad, *observador, *registro) {}, ErrCopiaPrevia},
		{"alterada", func(_ Servicio, _ *autoridad, _ *observador, r *registro) {
			r.r.Propuesta.Propuesta.MotivoRef = "motivo:2"
		}, d.ErrAlterada},
		{"expirada", func(s Servicio, _ *autoridad, _ *observador, _ *registro) {
			s.Reloj.(*reloj).n = s.Reloj.Ahora().Add(time.Hour)
		}, d.ErrCaducada},
		{"incompatible", func(_ Servicio, _ *autoridad, o *observador, _ *registro) { o.o.Destino.PostgreSQL.Version = "18.5" }, ErrCompatibilidad},
		{"no_comprobable", func(_ Servicio, _ *autoridad, o *observador, _ *registro) { o.o.Destino.Completo = false }, ErrCompatibilidad},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, a, o, r := servicioFixture(t)
			if _, e := s.Revisar(context.Background(), "sesion:2", r.r.Propuesta.Propuesta.Ref, r.r.Propuesta.Propuesta.DestinoRef, r.r.Propuesta.SHA256, 1); e != nil {
				t.Fatal(e)
			}
			tc.mut(s, a, o, r)
			e := s.ComprobarAntesSustitucion(context.Background(), "sesion:2", r.r.Propuesta.Propuesta.Ref, r.r.Propuesta.Propuesta.DestinoRef, r.r.Propuesta.SHA256, 2)
			if !errors.Is(e, tc.want) || r.cercas != 0 {
				t.Fatal(e, r.cercas)
			}
		})
	}
}
func TestMismaPersonaYCas(t *testing.T) {
	s, _, _, r := servicioFixture(t)
	ctx := context.Background()
	if _, e := s.Revisar(ctx, "sesion:1", r.r.Propuesta.Propuesta.Ref, r.r.Propuesta.Propuesta.DestinoRef, r.r.Propuesta.SHA256, 1); !errors.Is(e, d.ErrMismaPersona) {
		t.Fatal(e)
	}
	if _, e := s.Revisar(ctx, "sesion:2", r.r.Propuesta.Propuesta.Ref, r.r.Propuesta.Propuesta.DestinoRef, r.r.Propuesta.SHA256, 2); !errors.Is(e, ErrCAS) {
		t.Fatal(e)
	}
}
func TestCopiaPreviaCompletaYCercado(t *testing.T) {
	s, _, o, r := servicioFixture(t)
	// Un destino dedicado con la misma instalación simplifica el escenario feliz.
	destRef := o.o.Destino.Ref
	o.o.Destino = o.o.Manifiesto.Inventario
	o.o.Destino.Ref = destRef
	o.o.ExclusionRef = "exclusion:1"
	previa := o.o.Manifiesto
	previa.ConjuntoRef = "copia:previa"
	previa.Inicio = s.Reloj.Ahora()
	previa.Fin = previa.Inicio
	previa.Verificacion.Fecha = previa.Inicio
	o.o.CopiaPrevia = &p.CopiaPrevia{Manifiesto: previa, Autenticada: true, DestinoRef: destRef, PreimagenSHA256: o.o.PreimagenSHA256, ExclusionRef: o.o.ExclusionRef}
	ctx := context.Background()
	if _, e := s.Revisar(ctx, "sesion:2", r.r.Propuesta.Propuesta.Ref, destRef, r.r.Propuesta.SHA256, 1); e != nil {
		t.Fatal(e)
	}
	if e := s.ComprobarAntesSustitucion(ctx, "sesion:2", r.r.Propuesta.Propuesta.Ref, destRef, r.r.Propuesta.SHA256, 2); e != nil || r.cercas != 1 {
		t.Fatal(e, r.cercas)
	}
	o.o.CopiaPrevia.ExclusionRef = "exclusion:otra"
	if e := s.ComprobarAntesSustitucion(ctx, "sesion:2", r.r.Propuesta.Propuesta.Ref, destRef, r.r.Propuesta.SHA256, 2); !errors.Is(e, ErrCopiaPrevia) || r.cercas != 1 {
		t.Fatal(e, r.cercas)
	}
}
