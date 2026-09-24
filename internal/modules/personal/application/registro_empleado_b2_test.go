package application

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestRegistroB2ObjetivoRRHHIndependienteDelActor(t *testing.T) {
	actor := solicitudP(t).Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	s := domain.SolicitudFichaEmpleadoB2{EmpleadoRef: "emp_" + strings.Repeat("z", 24), Corte: domain.CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)}, Actor: actor}
	m, err := domain.NuevoMaterialFichaEmpleadoB2(s)
	if err != nil || m.Recurso().Referencia != s.EmpleadoRef || m.Recurso().Ambitos["empleado_ref"] != s.EmpleadoRef || bytes.Contains(m.Canonico(), []byte(`"empleado_ref":"emp_`+strings.Repeat("a", 24)+`"`)) {
		t.Fatalf("objetivo RRHH o material: %v", err)
	}
	if _, ok := m.Recurso().Ambitos["organismo_ref"]; ok {
		t.Fatal("ambito vacio")
	}
	// Cambiar el objeto del llamante no puede alterar material ni concesión.
	s.Actor.Instantanea.Vinculos[0].Referencia = "emp_" + strings.Repeat("q", 24)
	if m.Actor().Instantanea.Vinculos[0].Referencia != "emp_"+strings.Repeat("a", 24) {
		t.Fatal("alias de contexto")
	}
}

func TestRegistroB2NoAfirmaVacantesSinCobertura(t *testing.T) {
	actor := solicitudP(t).Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	s := domain.SolicitudVacantesB2{OrganismoRef: "organismo:dipgra", Corte: domain.CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)}, Limite: 20, Actor: actor}
	m, err := domain.NuevoMaterialVacantesB2(s)
	if err != nil {
		t.Fatal(err)
	}
	p := domain.PaginaVacantesB2{OrganismoRef: s.OrganismoRef, Corte: s.Corte, Limite: 20, Cobertura: "parcial"}
	if !errors.Is(p.ValidarPara(m), domain.ErrRegistroEmpleadoB2Invalido) {
		t.Fatal("acepto vacio parcial como cero vacantes")
	}
	p.Cobertura = "completa"
	if err := p.ValidarPara(m); err != nil {
		t.Fatal(err)
	}
}

func TestRegistroB2AltaLigaPersonaObjetivoYRelacionExplicita(t *testing.T) {
	actor := solicitudP(t).Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	procedencia := domain.ProcedenciaActoEmpleadoB2{ActoRef: "acto:alta", FuenteRef: "fuente:rrhh", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), IdempotenciaRef: "11111111-1111-4111-8111-111111111111"}
	s := domain.SolicitudAltaEmpleadoB2{PersonaRef: "per_" + strings.Repeat("z", 24), OrganismoRef: "organismo:dipgra", UnidadRef: "unidad:uno", RegimenRef: "regimen:funcionario", ModalidadRef: "modalidad:interino", VigenteDesde: fecha, Procedencia: procedencia, Actor: actor}
	m, err := domain.NuevoMaterialAltaEmpleadoB2(s)
	if err != nil || m.Recurso().Referencia != s.PersonaRef || !bytes.Contains(m.Canonico(), []byte(`"persona_ref":"`+s.PersonaRef+`"`)) || bytes.Contains(m.Canonico(), []byte("acreditacion_persona_ref")) {
		t.Fatalf("objetivo no ligado o prelectura B1 indebida: %v", err)
	}
	h := domain.SolicitudHechoEmpleadoB2{Tipo: "ocupacion", EmpleadoRef: "emp_" + strings.Repeat("a", 24), RevisionEsperada: 1, UnidadRef: "unidad:uno", ModalidadRef: "modalidad:uno", ClaseRef: "temporal", PlazaRef: "11111111-1111-4111-8111-111111111111", VersionPlazaRef: "plantilla:uno", VigenteDesde: fecha, Procedencia: procedencia, Actor: actor}
	if _, err := domain.NuevoMaterialHechoEmpleadoB2(h); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) {
		t.Fatal("ocupacion sin relacion elegida")
	}
	h.Tipo = "relacion"
	h.PlazaRef = ""
	h.VersionPlazaRef = ""
	h.ClaseRef = ""
	h.RelacionRef = ""
	h.Estado = "vigente"
	h.RegimenRef = "regimen:funcionario"
	if _, err := domain.NuevoMaterialHechoEmpleadoB2(h); err != nil {
		t.Fatalf("relacion nueva: %v", err)
	}
	h.RelacionRef = "rel_" + strings.Repeat("r", 24)
	if _, err := domain.NuevoMaterialHechoEmpleadoB2(h); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) {
		t.Fatal("revision sin CAS de relacion")
	}
	h.RelacionVersionEsperada = 1
	if _, err := domain.NuevoMaterialHechoEmpleadoB2(h); err != nil {
		t.Fatalf("revision explicita: %v", err)
	}
}
