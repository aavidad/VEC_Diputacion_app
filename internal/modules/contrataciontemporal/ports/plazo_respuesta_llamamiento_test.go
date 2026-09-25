package ports

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func solicitudEventoPlazoPrueba(tipo TipoEventoPlazoLlamamiento) SolicitudRegistrarEventoPlazoLlamamiento {
	return SolicitudRegistrarEventoPlazoLlamamiento{
		ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: "organizacion:plazo",
		ExpedienteRef: "expediente:plazo", LlamamientoRef: "llamamiento:plazo", ComunicacionRef: "comunicacion:plazo",
		VersionComunicacionEsperada: 2, Tipo: tipo, InstanteEn: time.Date(2026, 9, 28, 9, 30, 0, 0, time.UTC),
		PruebaRef: "prueba:llamada",
	}
}

func plazoGobernadoPrueba() PlazoRespuestaGobernado {
	referencia := func(entrada string) ReferenciaGobernadaComunicacionLlamamiento {
		return ReferenciaGobernadaComunicacionLlamamiento{Referencia: "vec.bolsa.reglas:1:" + entrada, Version: 1, HuellaSHA256: strings.Repeat("c", 64)}
	}
	return PlazoRespuestaGobernado{
		RespuestaHasta: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC), UltimoDia: "2026-09-29",
		Politica: referencia("b05.plazo_respuesta"), TratamientoFueraDePlazo: domain.TratamientoFueraDePlazoExigeCausaJustificada,
		ConfirmacionExpiracion: domain.ConfirmacionExpiracionRRHH, CriterioRespuesta: referencia("b07.fuera_de_plazo"),
		CriterioExpiracion: referencia("b08.sin_respuesta_baja"), ReglaEjemplo: true,
	}
}

func TestSolicitudEventoPlazoValidaSoloLaDeclaracion(t *testing.T) {
	s := solicitudEventoPlazoPrueba(EventoPlazoContactoEfectivo)
	if s.Validar() != nil || solicitudEventoPlazoPrueba(EventoPlazoCausaJustificada).Validar() != nil {
		t.Fatal("declaración válida rechazada")
	}
	for nombre, mutar := range map[string]func(*SolicitudRegistrarEventoPlazoLlamamiento){
		"clave":       func(s *SolicitudRegistrarEventoPlazoLlamamiento) { s.ClaveIdempotencia = "no-uuid" },
		"version":     func(s *SolicitudRegistrarEventoPlazoLlamamiento) { s.VersionComunicacionEsperada = 1 },
		"tipo":        func(s *SolicitudRegistrarEventoPlazoLlamamiento) { s.Tipo = "correo" },
		"sin_fecha":   func(s *SolicitudRegistrarEventoPlazoLlamamiento) { s.InstanteEn = time.Time{} },
		"nanosegundo": func(s *SolicitudRegistrarEventoPlazoLlamamiento) { s.InstanteEn = s.InstanteEn.Add(time.Nanosecond) },
		"zona": func(s *SolicitudRegistrarEventoPlazoLlamamiento) {
			s.InstanteEn = s.InstanteEn.In(time.FixedZone("x", 3600))
		},
		"prueba": func(s *SolicitudRegistrarEventoPlazoLlamamiento) { s.PruebaRef = "" },
	} {
		otra := s
		mutar(&otra)
		if otra.Validar() == nil {
			t.Errorf("%s aceptado", nombre)
		}
	}
}

func TestEventoPlazoExigePlazoSoloEnElContacto(t *testing.T) {
	s := solicitudEventoPlazoPrueba(EventoPlazoContactoEfectivo)
	plazo := plazoGobernadoPrueba()
	r := EventoPlazoLlamamientoRegistrado{Solicitud: s, Plazo: &plazo, EventoRef: "evento:1", ReciboRef: "recibo:1",
		AuditoriaRef: "auditoria:1", RegistradoEn: s.InstanteEn.Add(time.Minute), Estado: EstadoEventoPlazoRegistrado}
	if r.ValidarPara(s) != nil {
		t.Fatal("contacto válido rechazado")
	}
	sinPlazo := r
	sinPlazo.Plazo = nil
	if sinPlazo.ValidarPara(s) == nil {
		t.Fatal("contacto sin vencimiento aceptado")
	}
	causa := solicitudEventoPlazoPrueba(EventoPlazoCausaJustificada)
	conPlazo := r
	conPlazo.Solicitud = causa
	if conPlazo.ValidarPara(causa) == nil {
		t.Fatal("causa con vencimiento aceptada")
	}
	conPlazo.Plazo = nil
	if conPlazo.ValidarPara(causa) != nil {
		t.Fatal("causa válida rechazada")
	}
	for nombre, mutar := range map[string]func(*PlazoRespuestaGobernado){
		"vence_antes":  func(p *PlazoRespuestaGobernado) { p.RespuestaHasta = s.InstanteEn },
		"dia":          func(p *PlazoRespuestaGobernado) { p.UltimoDia = "29/09/2026" },
		"politica":     func(p *PlazoRespuestaGobernado) { p.Politica.HuellaSHA256 = "" },
		"tratamiento":  func(p *PlazoRespuestaGobernado) { p.TratamientoFueraDePlazo = "libre" },
		"confirmacion": func(p *PlazoRespuestaGobernado) { p.ConfirmacionExpiracion = "automatica" },
	} {
		otro := plazoGobernadoPrueba()
		mutar(&otro)
		if otro.ValidarDesde(s.InstanteEn) == nil {
			t.Errorf("plazo %s aceptado", nombre)
		}
	}
}

func TestResultadoResolucionConPlazoYCausa(t *testing.T) {
	s := solicitudResolverComunicacionPrueba(RespuestaLlamamientoAceptada)
	r := resultadoResolucionComunicacionPrueba(s, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoVigente, nil)
	r.RespuestaHasta = r.ResueltaEn.Add(-time.Hour)
	r.RespuestaFueraDePlazo, r.CausaJustificadaRef = true, "evento:causa"
	if r.ValidarPara(s) != nil {
		t.Fatal("respuesta tardía admitida con causa rechazada")
	}
	r.RespuestaFueraDePlazo = false
	if r.ValidarPara(s) == nil {
		t.Fatal("causa sin respuesta tardía aceptada")
	}
	r.RespuestaFueraDePlazo, r.RespuestaHasta = true, time.Time{}
	if r.ValidarPara(s) == nil {
		t.Fatal("respuesta tardía sin vencimiento aceptada")
	}
	e := solicitudResolverComunicacionPrueba(RespuestaLlamamientoExpirada)
	expirada := resultadoResolucionComunicacionPrueba(e, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoExpirado,
		estadoOutbox(OutboxSiguienteCandidatoPendiente))
	if expirada.ValidarPara(e) != nil {
		t.Fatal("expiración válida rechazada")
	}
	expirada.RespuestaHasta = expirada.ResueltaEn.Add(time.Second)
	if expirada.ValidarPara(e) == nil {
		t.Fatal("expiración antes del vencimiento aceptada")
	}
}
