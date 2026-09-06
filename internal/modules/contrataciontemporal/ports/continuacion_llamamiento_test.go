package ports

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func continuacionPrueba(t *testing.T) (SolicitudContinuarLlamamiento, AntecedenteContinuacionLlamamiento, ReciboBolsaContinuacion) {
	t.Helper()
	sr := solicitudResolverComunicacionPrueba(RespuestaLlamamientoRenunciada)
	sr.VersionEsperada = 2
	sr.RevisionRespuestaRRHH, sr.RevisionPlazoRRHH = true, true
	sr.CriterioValidacionRef = "politica:revision-sintetica"
	r := resultadoResolucionComunicacionPrueba(sr, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoVigente, estadoOutbox(OutboxSiguienteCandidatoPendiente))
	s := SolicitudContinuarLlamamiento{"31111111-1111-4111-8111-111111111111", sr.OrganizacionRef, sr.ExpedienteRef, r.ResolucionRef, r.IntencionSiguiente.IntencionRef}
	a := AntecedenteContinuacionLlamamiento{Resolucion: r, ComandoSiguienteRef: r.IntencionSiguiente.ComandoOpacoRef,
		ComandoSiguiente: ComandoSiguienteLlamamiento{
			Esquema: "vec.contratacion-temporal.siguiente-candidato.intencion.v1", ComandoRef: r.IntencionSiguiente.ComandoOpacoRef,
			IntencionRef: s.IntencionRef, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
			LlamamientoRef: sr.LlamamientoRef, JustificanteRef: sr.PruebaRespuestaRef, SeleccionClave: sr.ClaveIdempotencia,
		}}
	b := ReciboBolsaContinuacion{IntencionRef: s.IntencionRef, TerminalOperacionRef: "operacion:renuncia",
		OperacionRef: "operacion:siguiente", LlamamientoRef: "llamamiento:siguiente", PropuestaRef: "propuesta:siguiente",
		ReciboRef: "recibo:bolsa", AuditoriaRef: "auditoria:bolsa", EventoRef: "evento:bolsa",
		RegistroSHA256: strings.Repeat("a", 64), ConfirmadaEn: r.ResueltaEn.Add(time.Minute)}
	if s.Validar() != nil || a.ValidarPara(s) != nil || b.ValidarPara(s) != nil {
		t.Fatal("fixture incoherente")
	}
	return s, a, b
}

func TestContinuacionLlamamientoContratoYAntecedente(t *testing.T) {
	s, a, b := continuacionPrueba(t)
	if reflect.TypeOf(s).NumField() != 5 {
		t.Fatal("solicitud no mínima")
	}
	for _, clave := range []string{"", strings.ToUpper(s.ClaveIdempotencia), "31111111-1111-5111-8111-111111111111", "00000000-0000-4000-8000-000000000000"} {
		if clave == s.ClaveIdempotencia {
			continue
		}
		otra := s
		otra.ClaveIdempotencia = clave
		if otra.Validar() == nil {
			t.Fatal("clave inválida admitida")
		}
	}
	for _, mutar := range []func(*AntecedenteContinuacionLlamamiento){
		func(a *AntecedenteContinuacionLlamamiento) { a.ComandoSiguiente.IntencionRef = "intencion:ajena" },
		func(a *AntecedenteContinuacionLlamamiento) { a.ComandoSiguienteRef = "comando:ajeno" },
		func(a *AntecedenteContinuacionLlamamiento) { a.ComandoSiguiente.SeleccionClave = "" },
		func(a *AntecedenteContinuacionLlamamiento) { a.Resolucion.Solicitud.RevisionPlazoRRHH = false },
		func(a *AntecedenteContinuacionLlamamiento) {
			a.Resolucion.Estado = ResultadoComunicacionLlamamientoReplay
		},
	} {
		otra := a
		mutar(&otra)
		if otra.ValidarPara(s) == nil {
			t.Fatal("antecedente cruzado o incompleto")
		}
	}
	for _, m := range []MaterialContinuacionLlamamiento{{Etapa: "consulta", Solicitud: s}, {Etapa: "confirmacion", Solicitud: s, ReciboBolsa: &b}} {
		if m.Validar() != nil {
			t.Fatal("material válido rechazado")
		}
		j, _ := json.Marshal(m)
		var campos map[string]json.RawMessage
		if json.Unmarshal(j, &campos) != nil || (m.Etapa == "consulta" && len(campos) != 2) || (m.Etapa == "confirmacion" && len(campos) != 3) {
			t.Fatal("JSON de etapa incorrecto")
		}
	}
	for _, m := range []MaterialContinuacionLlamamiento{{Etapa: "consulta", Solicitud: s, ReciboBolsa: &b}, {Etapa: "confirmacion", Solicitud: s}, {Etapa: "otra", Solicitud: s}} {
		if m.Validar() == nil {
			t.Fatal("etapa o recibo no ligado")
		}
	}
}

func TestContinuacionLlamamientoReciboYReplay(t *testing.T) {
	s, a, b := continuacionPrueba(t)
	r := ResultadoContinuacionLlamamiento{Solicitud: s, LlamamientoAnteriorRef: a.Resolucion.Solicitud.LlamamientoRef,
		ReciboBolsa: b, ReciboRef: "recibo:continuacion", AuditoriaRef: "auditoria:continuacion",
		ConfirmadaEn: b.ConfirmadaEn.Add(time.Second), Estado: "confirmado"}
	if r.ValidarPara(s) != nil {
		t.Fatal("recibo rechazado")
	}
	original := r
	r.Estado = "replay_confirmado"
	if r.ValidarPara(s) != nil {
		t.Fatal("replay rechazado")
	}
	r.Estado = "confirmado"
	if r != original {
		t.Fatal("replay cambia recibo")
	}
	for _, mutar := range []func(*ResultadoContinuacionLlamamiento){
		func(r *ResultadoContinuacionLlamamiento) { r.LlamamientoAnteriorRef = r.ReciboBolsa.LlamamientoRef },
		func(r *ResultadoContinuacionLlamamiento) { r.ReciboBolsa.IntencionRef = "intencion:otra" },
		func(r *ResultadoContinuacionLlamamiento) { r.ReciboBolsa.RegistroSHA256 = strings.Repeat("0", 64) },
		func(r *ResultadoContinuacionLlamamiento) {
			r.ConfirmadaEn = r.ReciboBolsa.ConfirmadaEn.Add(-time.Microsecond)
		},
		func(r *ResultadoContinuacionLlamamiento) { r.ConfirmadaEn = r.ConfirmadaEn.Add(time.Nanosecond) },
		func(r *ResultadoContinuacionLlamamiento) { r.Estado = "enviada" },
	} {
		otro := r
		mutar(&otro)
		if otro.ValidarPara(s) == nil {
			t.Fatal("recibo no confiable admitido")
		}
	}
}
