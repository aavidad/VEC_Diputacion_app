package ports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func propuestaDesarrolloPrueba(t *testing.T) (SolicitudPropuestaFormalizacion, AntecedentePropuestaFormalizacion, EvidenciaAceptacionBolsaPropuesta) {
	t.Helper()
	s, j := justificanteRespuestaRecibidaPrueba(t)
	s.Respuesta, j.Respuesta.Solicitud.Respuesta = RespuestaLlamamientoAceptada, RespuestaLlamamientoAceptada
	s.RevisionRespuestaRRHH, s.RevisionPlazoRRHH = true, true
	s.CriterioValidacionRef = "politica:manual-sintetica"
	r := ResultadoResolucionLlamamiento{Solicitud: s,
		Politica:           ReferenciaGobernadaComunicacionLlamamiento{Referencia: s.CriterioValidacionRef, Version: 1, HuellaSHA256: strings.Repeat("a", 64)},
		EvaluacionPlazoRef: "evaluacion:manual", EstadoPlazo: PlazoLlamamientoVigente,
		ResolucionRef: "resolucion:aceptada", ReciboLocalRef: "recibo:aceptada", AuditoriaRef: "auditoria:aceptada",
		VersionResultante: 3, ResueltaEn: j.Respuesta.RegistradaEn.Add(time.Minute), Estado: ResultadoComunicacionLlamamientoConfirmado}
	p := solicitudPropuestaFormalizacionPrueba()
	p.VersionEsperada, p.Anexos = 6, nil
	p.OrganizacionRef, p.ExpedienteRef, p.LlamamientoRef = s.OrganizacionRef, s.ExpedienteRef, s.LlamamientoRef
	p.ResolucionLlamamientoAceptadaRef, p.ReciboResolucionAceptadaRef = r.ResolucionRef, r.ReciboLocalRef
	p.TipoFormalizacion.Version, p.Plantilla.Version, p.PoliticaFirma.Version, p.PlanFirma.Version = 1, 1, 1, 1
	a := AntecedentePropuestaFormalizacion{Resolucion: r, Justificante: j, SeleccionClave: "21111111-1111-4111-8111-111111111111"}
	e := EvidenciaAceptacionBolsaPropuesta{OperacionRef: "operacion:aceptada", AperturaOperacionRef: j.Seleccion.OperacionRef,
		LlamamientoRef: s.LlamamientoRef, JustificanteRef: s.PruebaRespuestaRef, EvaluacionPlazoRef: r.EvaluacionPlazoRef,
		Politica:       SnapshotGobernadoFormalizacion{Referencia: r.Politica.Referencia, Version: r.Politica.Version, HuellaSHA256: r.Politica.HuellaSHA256},
		RegistroSHA256: strings.Repeat("b", 64), ResueltaEn: r.ResueltaEn.Add(time.Second)}
	if a.ValidarPara(p) != nil || e.ValidarPara(a) != nil {
		t.Fatal("fixture estructural incoherente")
	}
	return p, a, e
}

func TestPropuestaFormalizacionDesarrolloMaterialYAntecedente(t *testing.T) {
	s, a, e := propuestaDesarrolloPrueba(t)
	consulta := MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}
	confirmacion := MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s, AceptacionBolsa: &e}
	for _, m := range []MaterialPropuestaFormalizacion{consulta, confirmacion} {
		if err := m.Validar(); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		var campos map[string]json.RawMessage
		if json.Unmarshal(b, &campos) != nil || len(campos) != map[string]int{"consulta": 2, "confirmacion": 3}[m.Etapa] {
			t.Fatal("envoltorio inesperado")
		}
	}
	clon := confirmacion.Clonar()
	clon.AceptacionBolsa.OperacionRef = "operacion:otro"
	if e.OperacionRef != "operacion:aceptada" {
		t.Fatal("alias de evidencia")
	}
	for nombre, mutar := range map[string]func(*AntecedentePropuestaFormalizacion){
		"resolucion": func(x *AntecedentePropuestaFormalizacion) { x.Resolucion.ResolucionRef = "resolucion:otra" },
		"renuncia": func(x *AntecedentePropuestaFormalizacion) {
			x.Resolucion.Solicitud.Respuesta = RespuestaLlamamientoRenunciada
		},
		"declaracion": func(x *AntecedentePropuestaFormalizacion) {
			x.Justificante.Respuesta.Solicitud.ExpedienteRef = "expediente:otro"
		},
		"seleccion": func(x *AntecedentePropuestaFormalizacion) { x.SeleccionClave = "no-uuid" },
		"politica":  func(x *AntecedentePropuestaFormalizacion) { x.Resolucion.Politica.Referencia = "politica:otra" },
	} {
		t.Run(nombre, func(t *testing.T) {
			x := a
			mutar(&x)
			if x.ValidarPara(s) == nil {
				t.Fatal("antecedente cruzado")
			}
		})
	}
	for _, mutar := range []func(*EvidenciaAceptacionBolsaPropuesta){
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.AperturaOperacionRef = "operacion:otra" },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.JustificanteRef = "justificante:otro" },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.Politica.Version++ },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.ResueltaEn = a.Resolucion.ResueltaEn.Add(-time.Second) },
		func(x *EvidenciaAceptacionBolsaPropuesta) { x.RegistroSHA256 = strings.Repeat("0", 64) },
	} {
		x := e
		mutar(&x)
		if x.ValidarPara(a) == nil {
			t.Fatal("terminal cruzado")
		}
	}
}

func TestPropuestaFormalizacionDesarrolloRechazaEtapaYSuperficie(t *testing.T) {
	s, _, e := propuestaDesarrolloPrueba(t)
	for _, m := range []MaterialPropuestaFormalizacion{
		{Etapa: "consulta", Solicitud: s, AceptacionBolsa: &e},
		{Etapa: "confirmacion", Solicitud: s},
		{Etapa: "otra", Solicitud: s},
	} {
		if m.Validar() == nil {
			t.Fatal("etapa inválida")
		}
	}
	s.VersionEsperada = 7
	if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() == nil {
		t.Fatal("intención nueva v7; replay conserva solicitud v6")
	}
	s.VersionEsperada = 6
	s.Anexos = []AnexoPropuestaFormalizacion{anexoPropuestaFormalizacionPrueba("x", "a", 1)}
	if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() == nil {
		t.Fatal("anexo fuera de este corte")
	}
}
