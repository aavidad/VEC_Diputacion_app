package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/modules/seleccion/application"
)

func ejemplo(t *testing.T) entrada {
	t.Helper()
	b, err := os.ReadFile("../../data/ejemplos/meritos/preparacion.json")
	if err != nil {
		t.Fatal(err)
	}
	var p meritos.Paquete
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	return entrada{Paquete: p, Solicitud: application.SolicitudHechosPreparacion{
		Alcance: meritos.AlcanceSintetico,
		Contexto: application.ContextoHechosPreparacion{ConvocatoriaRef: "convocatoria:ensayo", BasesRef: "bases:ensayo", BasesVersion: 1,
			SolicitudRef: "solicitud:ensayo", HitoRef: "hito:ensayo", Uso: "merito"},
		Selector: ports.SelectorHechosPreparacion{PersonaRef: "persona:ensayo:01", FechaCorte: p.FechaCorte,
			Hechos: []ports.ReferenciaHechoPreparacion{{Referencia: "hecho:ensayo:01", VersionEsperada: 2}, {Referencia: "hecho:ensayo:02", VersionEsperada: 1}}}}}
}

func serializar(t *testing.T, e entrada) []byte {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCLIConservaHechosYUsoSinExportarIdentidades(t *testing.T) {
	for _, uso := range []string{"requisito", "merito"} {
		e := ejemplo(t)
		e.Solicitud.Contexto.Uso = uso
		var out, errOut bytes.Buffer
		if run(bytes.NewReader(serializar(t, e)), &out, &errOut) != 0 {
			t.Fatal(errOut.String())
		}
		var p application.PreparacionHechosProceso
		if err := json.Unmarshal(out.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		if p.Contexto != e.Solicitud.Contexto || len(p.Hechos) != 2 || p.LecturaAutorizada || p.DecisionReal || p.Alcance != meritos.AlcanceSintetico || len(p.Pendientes) != 3 {
			t.Fatal("contexto o límites de preparación perdidos")
		}
		if p.Hechos[0].Version != 2 || p.Hechos[0].Estado != meritos.Pendiente || p.Hechos[1].Estado != meritos.Acreditado || p.Hechos[1].Horas == nil || *p.Hechos[1].Horas != 20 || len(p.Hechos[1].Pendientes) < 4 {
			t.Fatal("estado, versión o fuente sustituidos")
		}
		for _, privado := range []string{"persona_ref", "nombre", "denominacion", "actor_ref", "actor:ensayo:revisor", "persona:ensayo:02", "hecho:ensayo:03", "puntos", "cumple"} {
			if strings.Contains(out.String(), privado) {
				t.Fatalf("proyección revela %s", privado)
			}
		}
	}
}

func TestCLIRechazaSelectoresYFuentesSinResultadoParcial(t *testing.T) {
	cambios := []func(*entrada){
		func(e *entrada) { e.Solicitud.Alcance = "real" },
		func(e *entrada) { e.Solicitud.Selector.PersonaRef = "persona:ensayo:02" },
		func(e *entrada) { e.Solicitud.Selector.Hechos[0].VersionEsperada = 1 },
		func(e *entrada) { e.Solicitud.Selector.Hechos[1].Referencia = "hecho:ausente" },
		func(e *entrada) {
			e.Solicitud.Selector.Hechos = append(e.Solicitud.Selector.Hechos, e.Solicitud.Selector.Hechos[0])
		},
		func(e *entrada) { e.Solicitud.Selector.FechaCorte = "2026-10-02" },
		func(e *entrada) { e.Solicitud.Contexto.BasesVersion = 0 },
		func(e *entrada) { e.Solicitud.Contexto.HitoRef = "" },
		func(e *entrada) { e.Paquete.Hechos = append(e.Paquete.Hechos, e.Paquete.Hechos[0]) },
		func(e *entrada) {
			h := e.Paquete.Hechos[0]
			h.Referencia = "hecho:duplicado"
			e.Paquete.Hechos = append(e.Paquete.Hechos, h)
		},
	}
	for i, cambiar := range cambios {
		e := ejemplo(t)
		cambiar(&e)
		var out, errOut bytes.Buffer
		if run(bytes.NewReader(serializar(t, e)), &out, &errOut) != 1 || out.Len() != 0 || errOut.String() != "{\"error_clave\":\"meritos.error.hechos_preparacion\"}\n" {
			t.Fatalf("caso %d: resultado parcial o detalles de fuente", i)
		}
	}
}

func TestCLILimitaEntradaYRechazaJSONAmbiguo(t *testing.T) {
	for _, s := range []string{`{"solicitud":{},"solicitud":{}}`, `{"solicitud":{},"\u0073olicitud":{}}`, `{"Solicitud":{}}`, `{"secreto":"reservado"}`, string(serializar(t, ejemplo(t))) + `{}`, strings.Repeat(" ", maxBytes+1)} {
		var out, errOut bytes.Buffer
		if run(strings.NewReader(s), &out, &errOut) != 1 || out.Len() != 0 || strings.Contains(errOut.String(), "reservado") {
			t.Fatal("JSON ambiguo admitido o expuesto")
		}
	}
}
