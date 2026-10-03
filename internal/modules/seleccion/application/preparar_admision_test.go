package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	meritos "vec-diputacion-granada/internal/modules/meritos/domain"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func materialAdmision(t *testing.T) ports.MaterialAdmisionPreparacion {
	t.Helper()
	raw, err := os.ReadFile("../../../../cmd/vec-selectivos-preparar-admision/testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	var m ports.MaterialAdmisionPreparacion
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPrepararAdmisionConservaPendientesYReferenciasSinDecidir(t *testing.T) {
	m := materialAdmision(t)
	r, err := application.PrepararAdmision(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	if r.Persistido || r.AdmisionOficial || r.Estado != "pendiente_revision_competente" || r.UniversoRequisitos != "propuesto_no_cotejado" ||
		r.SolicitudContexto.Estado != "sin_presentar" || r.Bases != m.Bases || r.Revision != m.Revision || len(r.Pendientes) != 6 {
		t.Fatalf("incorrect boundary: %+v", r)
	}
	for _, req := range r.Requisitos {
		if req.Estado != "pendiente" || len(req.Causas) < 2 || req.AccionPropuesta != "seleccion.admision.accion.preparar_aportacion" {
			t.Fatalf("incorrect review: %+v", req)
		}
	}
	if r.Requisitos[0].Soportes[0].Referencia != "hecho:titulacion" || r.Requisitos[0].Soportes[0].Version != 1 || r.Requisitos[0].Requisito.TituloPropuesto != m.Requisitos[0].TituloPropuesto {
		t.Fatal("lost exact proposed references")
	}
	m.SolicitudContexto.Version = "2"
	m.Requisitos[0].HechosEsperados[0].Version = 2
	if r.SolicitudContexto.Version != "1" || r.Requisitos[0].Requisito.HechosEsperados[0].Version != 1 {
		t.Fatal("aliased input")
	}
}

func TestPrepararAdmisionSinRequisitosNiSolicitudConservaListaIncompleta(t *testing.T) {
	m := materialAdmision(t)
	m.Requisitos, m.Hechos, m.SolicitudContexto = nil, nil, nil
	r, err := application.PrepararAdmision(context.Background(), m)
	if err != nil || len(r.Requisitos) != 0 || r.Requisitos == nil || r.AdmisionOficial || r.Estado != "pendiente_revision_competente" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestPrepararAdmisionRechazoOVigenciaNoExcluyen(t *testing.T) {
	for _, estado := range []string{"acreditado", "rechazado"} {
		t.Run(estado, func(t *testing.T) {
			m := materialAdmision(t)
			m.Hechos.Hechos[0].Estado = meritos.Estado(estado)
			m.Hechos.Hechos[0].Vigencia.Hasta = "2026-09-30"
			r, err := application.PrepararAdmision(context.Background(), m)
			if err != nil || r.Requisitos[0].Estado != "pendiente" || r.AdmisionOficial {
				t.Fatalf("%+v %v", r, err)
			}
			found := false
			for _, causa := range r.Requisitos[0].Causas {
				if causa == "seleccion.admision.causa.vigencia_aplicabilidad_pendiente" {
					found = true
				}
			}
			if !found {
				t.Fatal("lost applicability warning")
			}
		})
	}
}

func TestPrepararAdmisionInvalidaNoDevuelveResultado(t *testing.T) {
	casos := map[string]func(*ports.MaterialAdmisionPreparacion){
		"real_scope":             func(m *ports.MaterialAdmisionPreparacion) { m.Alcance = "real" },
		"registered_claim":       func(m *ports.MaterialAdmisionPreparacion) { m.SolicitudContexto.Estado = "presentada" },
		"mismatched_bases":       func(m *ports.MaterialAdmisionPreparacion) { m.SolicitudContexto.Version = "2" },
		"repeated_requirement":   func(m *ports.MaterialAdmisionPreparacion) { m.Requisitos = append(m.Requisitos, m.Requisitos[0]) },
		"different_fact_version": func(m *ports.MaterialAdmisionPreparacion) { m.Hechos.Hechos[0].Version = 2 },
		"invalid_fact_state":     func(m *ports.MaterialAdmisionPreparacion) { m.Hechos.Hechos[0].Estado = "cumple" },
		"invalid_date":           func(m *ports.MaterialAdmisionPreparacion) { m.Hechos.Hechos[0].Vigencia.Hasta = "2026-02-30" },
		"text_rule":              func(m *ports.MaterialAdmisionPreparacion) { m.Requisitos[0].Representacion = "texto_libre" },
		"unversioned_rule":       func(m *ports.MaterialAdmisionPreparacion) { m.Requisitos[0].ReglaVersion = "" },
		"controlled_title":       func(m *ports.MaterialAdmisionPreparacion) { m.Requisitos[0].TituloPropuesto = "x\n" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			m := materialAdmision(t)
			cambiar(&m)
			r, err := application.PrepararAdmision(context.Background(), m)
			if !errors.Is(err, domain.ErrAdmisionPreparacion) || r.Esquema != "" || r.Requisitos != nil {
				t.Fatalf("%+v %v", r, err)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	r, err := application.PrepararAdmision(ctx, materialAdmision(t))
	if !errors.Is(err, context.Canceled) || r.Esquema != "" {
		t.Fatalf("%+v %v", r, err)
	}
}
