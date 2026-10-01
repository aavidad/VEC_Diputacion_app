package domain

import (
	"reflect"
	"testing"

	documentos "vec-diputacion-granada/internal/vec/domain"
)

func paquetePrueba() Paquete {
	return Paquete{Alcance: AlcanceSintetico, Version: "ensayo:1", FechaCorte: "2026-10-01",
		Personas: []PersonaSintetica{{"persona:externa", "María Pérez Aguilar"}},
		Hechos: []Hecho{{Referencia: "hecho:01", PersonaRef: "persona:externa", Version: 1,
			Tipo: "titulacion", ConceptoRef: "titulo:01", Denominacion: "Grado universitario",
			Procedencia: Procedencia{"fuente:01", "1", "origen:01", "2026-10-01T09:00:00Z"},
			Vigencia:    Vigencia{Desde: "2025-06-20"}, Estado: Declarado}},
	}
}

func TestPrepararExternoConservaHistoriaSinAcreditar(t *testing.T) {
	p := paquetePrueba()
	v2 := p.Hechos[0]
	v2.Version, v2.Estado = 2, Acreditado
	v2.Evidencias = []documentos.ReferenciaDocumento{{ID: "documento:01", Version: 3}}
	v2.Revision = &Revision{"revision:01", "actor:01", "motivo:01", "2026-10-01T09:00:00Z"}
	p.Hechos = append(p.Hechos, v2)
	out, err := Preparar(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out.Paquete, p) || out.Persistido || out.AcreditacionReal || out.Estado != "pendiente_comprobacion_fuentes" {
		t.Fatal("la preparación no conserva los datos y límites")
	}
	for _, r := range out.Revisiones {
		if len(r.Pendientes) < 4 {
			t.Fatal("una afirmación acreditada pierde comprobaciones pendientes")
		}
	}
	// Cambiar el resultado no puede alterar la instantánea aportada.
	out.Paquete.Hechos[1].Revision.ActorRef = "otro"
	out.Paquete.Hechos[1].Evidencias[0].Version = 4
	if p.Hechos[1].Revision.ActorRef != "actor:01" || p.Hechos[1].Evidencias[0].Version != 3 {
		t.Fatal("alias de entrada")
	}
}

func TestPrepararRechazaConflictosDeIdentidadYVersion(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*Paquete)
	}{
		{"real", func(p *Paquete) { p.Alcance = "real" }},
		{"persona_ajena", func(p *Paquete) { p.Hechos[0].PersonaRef = "persona:otra" }},
		{"version_sin_historia", func(p *Paquete) { p.Hechos[0].Version = 2 }},
		{"version_repetida", func(p *Paquete) { p.Hechos = append(p.Hechos, p.Hechos[0]) }},
		{"origen_duplicado", func(p *Paquete) { h := p.Hechos[0]; h.Referencia = "hecho:02"; p.Hechos = append(p.Hechos, h) }},
		{"origen_cambiado", func(p *Paquete) {
			h := p.Hechos[0]
			h.Version = 2
			h.Procedencia.HechoOrigenRef = "origen:02"
			p.Hechos = append(p.Hechos, h)
		}},
		{"persona_cambiada", func(p *Paquete) {
			p.Personas = append(p.Personas, PersonaSintetica{"persona:02", "Luis Romero Díaz"})
			h := p.Hechos[0]
			h.Version = 2
			h.PersonaRef = "persona:02"
			p.Hechos = append(p.Hechos, h)
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p := paquetePrueba()
			caso.cambiar(&p)
			if _, err := Preparar(p); err == nil {
				t.Fatal("aceptó conflicto")
			}
		})
	}
}

func TestIntegridadDeEstadoEvidenciaYFechas(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*Hecho)
	}{
		{"acreditado_sin_revision", func(h *Hecho) { h.Estado = Acreditado }},
		{"rechazado_sin_motivo", func(h *Hecho) { h.Estado = Rechazado; h.Revision = &Revision{} }},
		{"documento_sin_version", func(h *Hecho) { h.Evidencias = []documentos.ReferenciaDocumento{{ID: "documento:01"}} }},
		{"documento_repetido", func(h *Hecho) {
			ev := documentos.ReferenciaDocumento{ID: "documento:01", Version: 1}
			h.Evidencias = []documentos.ReferenciaDocumento{ev, ev}
		}},
		{"fecha_imposible", func(h *Hecho) { h.Vigencia.Desde = "2026-02-30" }},
		{"periodo_invertido", func(h *Hecho) { h.Vigencia.Hasta = "2024-01-01" }},
		{"horas_negativas", func(h *Hecho) { h.Tipo = "curso_asistencia"; n := -1; h.Horas = &n }},
		{"referencia_url", func(h *Hecho) { h.PersonaRef = "https://example.org/persona" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			h := paquetePrueba().Hechos[0]
			caso.cambiar(&h)
			if h.Validar() == nil {
				t.Fatal("aceptó hecho inválido")
			}
		})
	}
}

func TestVigenciaCivilYHechosDeCursoSeparados(t *testing.T) {
	for _, caso := range []struct{ desde, hasta, esperado string }{
		{"2026-10-01", "2026-10-01", "vigente"},
		{"2026-10-02", "", "no_iniciada"},
		{"2025-01-01", "2026-09-30", "finalizada"},
	} {
		p := paquetePrueba()
		p.Hechos[0].Vigencia = Vigencia{caso.desde, caso.hasta}
		out, err := Preparar(p)
		if err != nil || out.Revisiones[0].VigenciaEnCorte != caso.esperado {
			t.Fatalf("%+v: %v", caso, err)
		}
	}
	p := paquetePrueba()
	p.Hechos[0].Tipo = "curso_asistencia"
	superacion := p.Hechos[0]
	superacion.Referencia, superacion.Tipo = "hecho:02", "curso_superacion"
	p.Hechos = append(p.Hechos, superacion)
	if _, err := Preparar(p); err != nil {
		t.Fatal("confunde asistencia y superación", err)
	}
}
