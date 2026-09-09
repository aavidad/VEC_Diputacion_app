package ports

import (
	"strings"
	"testing"
	"time"
)

func solicitudResolucionPrueba() SolicitudResolucionFormalizacion {
	return SolicitudResolucionFormalizacion{ExpedienteRef: "expediente:prueba", VersionEsperada: 7, PropuestaRef: "propuesta:prueba",
		ClaveIdempotencia: "31111111-1111-4111-8111-111111111111", NumeroResolucion: "EJ-2026/1", FechaResolucion: "2026-09-06",
		Motivo: "Revisión manual del ejercicio.", ConfirmaRevisionPropuesta: true, ConfirmaEjercicioManual: true}
}
func reciboResolucionPrueba(s SolicitudResolucionFormalizacion) ResultadoResolucionFormalizacion {
	return ResultadoResolucionFormalizacion{Solicitud: s, Estado: "registrada", ResolucionRef: "resolucion:prueba",
		DocumentoRef: ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoSHA256: strings.Repeat("a", 64), DocumentoVersion: 7,
		ActuacionRef: "resolucion:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "evento:prueba", ReciboRef: "recibo:prueba",
		VersionResultante: 8, RegistradaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
}

func TestResolucionFormalizacionPreparacionV7V8Cerrada(t *testing.T) {
	s := solicitudResolucionPrueba()
	r := reciboResolucionPrueba(s)
	p := PreparacionResolucionFormalizacion{s.ExpedienteRef, s.PropuestaRef, 7, 7, nil}
	if p.ValidarPara(s.ExpedienteRef) != nil {
		t.Fatal("v7 rechazada")
	}
	p.VersionActual = 8
	p.Recibo = &r
	if p.ValidarPara(s.ExpedienteRef) != nil {
		t.Fatal("v8 rechazada")
	}
	c := p.Clonar()
	c.Recibo.ReciboRef = "recibo:otro"
	if p.Recibo.ReciboRef == c.Recibo.ReciboRef {
		t.Fatal("clon compartido")
	}
	for _, mutar := range []func(*PreparacionResolucionFormalizacion){
		func(p *PreparacionResolucionFormalizacion) { p.VersionActual = 7 },
		func(p *PreparacionResolucionFormalizacion) { p.Recibo = nil },
		func(p *PreparacionResolucionFormalizacion) { p.VersionActual = 9 },
		func(p *PreparacionResolucionFormalizacion) { p.VersionEsperada = 8 },
		func(p *PreparacionResolucionFormalizacion) { p.PropuestaRef = "propuesta:otra" },
		func(p *PreparacionResolucionFormalizacion) { p.Recibo.Estado = "replay_registrada" },
	} {
		c := p.Clonar()
		mutar(&c)
		if c.ValidarPara(s.ExpedienteRef) == nil {
			t.Fatal("preparación divergente")
		}
	}
}

func TestResolucionFormalizacionSolicitudYDocumentoCerrados(t *testing.T) {
	s := solicitudResolucionPrueba()
	if s.Validar() != nil {
		t.Fatal("fixture inválido")
	}
	for _, mutar := range []func(*SolicitudResolucionFormalizacion){
		func(s *SolicitudResolucionFormalizacion) { s.ConfirmaRevisionPropuesta = false },
		func(s *SolicitudResolucionFormalizacion) { s.ConfirmaEjercicioManual = false },
		func(s *SolicitudResolucionFormalizacion) { s.VersionEsperada = 8 },
		func(s *SolicitudResolucionFormalizacion) { s.FechaResolucion = "2026-02-30" },
		func(s *SolicitudResolucionFormalizacion) { s.ClaveIdempotencia = "identidad:no" },
		func(s *SolicitudResolucionFormalizacion) { s.Motivo = strings.Repeat("ñ", 1001) },
		func(s *SolicitudResolucionFormalizacion) { s.NumeroResolucion = "uno\nDos" },
		func(s *SolicitudResolucionFormalizacion) { s.ExpedienteRef = "https://no permitido" },
	} {
		v := s
		mutar(&v)
		if v.Validar() == nil {
			t.Fatal("entrada inválida admitida", v)
		}
	}
	r := reciboResolucionPrueba(s)
	if r.ValidarPara(s) != nil {
		t.Fatal("recibo válido rechazado")
	}
	r.Estado = "replay_registrada"
	if r.ValidarPara(s) != nil {
		t.Fatal("replay rechazado")
	}
	for _, mutar := range []func(*ResultadoResolucionFormalizacion){
		func(r *ResultadoResolucionFormalizacion) { r.DocumentoSHA256 = strings.Repeat("0", 64) },
		func(r *ResultadoResolucionFormalizacion) { r.DocumentoRef = "documento:ajeno" },
		func(r *ResultadoResolucionFormalizacion) { r.DocumentoVersion = 8 },
		func(r *ResultadoResolucionFormalizacion) { r.Estado = "firmada" },
		func(r *ResultadoResolucionFormalizacion) { r.AuditoriaRef = "" },
		func(r *ResultadoResolucionFormalizacion) { r.RegistradaEn = r.RegistradaEn.Add(time.Nanosecond) },
	} {
		v := r
		mutar(&v)
		if v.ValidarPara(s) == nil {
			t.Fatal("recibo inválido admitido")
		}
	}
}
