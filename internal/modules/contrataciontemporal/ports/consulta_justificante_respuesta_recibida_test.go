package ports

import (
	"errors"
	"testing"
	"time"
)

func justificanteRespuestaRecibidaPrueba(t *testing.T) (SolicitudResolverLlamamiento, JustificanteRespuestaRecibida) {
	t.Helper()
	r := resultadoRespuestaRecibidaPrueba()
	base := r.Solicitud.RecibidaEn.Add(-time.Hour)
	comando := comandoLlamamientoPrueba(t, base, selladorRespuestaBolsaPrueba())
	seleccion := reciboLlamamientoPrueba(t, comando, base)
	// Fixture estructural: la procedencia SQL nominal se comprueba en su
	// adaptador, no se simula una nueva autenticación de este recibo histórico.
	seleccion.VersionExpediente = 6
	r.Solicitud.OrganizacionRef = seleccion.OrganizacionRef
	r.Solicitud.ExpedienteRef = seleccion.ExpedienteRef
	r.Solicitud.LlamamientoRef = seleccion.LlamamientoRef
	s := SolicitudResolverLlamamiento{
		ClaveIdempotencia: "11111111-1111-4111-8111-111111111111",
		OrganizacionRef:   r.Solicitud.OrganizacionRef, ExpedienteRef: r.Solicitud.ExpedienteRef,
		LlamamientoRef: r.Solicitud.LlamamientoRef, ComunicacionRef: r.Solicitud.ComunicacionRef,
		VersionEsperada: 2, Respuesta: r.Solicitud.Respuesta, PruebaRespuestaRef: r.JustificanteRef,
	}
	return s, JustificanteRespuestaRecibida{Respuesta: r, Seleccion: seleccion}
}

func TestConsultaJustificanteRespuestaRecibidaOriginalSinEvaluarPlazo(t *testing.T) {
	s, j := justificanteRespuestaRecibidaPrueba(t)
	for _, respuesta := range []RespuestaLlamamiento{RespuestaLlamamientoAceptada, RespuestaLlamamientoRenunciada} {
		s.Respuesta, j.Respuesta.Solicitud.Respuesta = respuesta, respuesta
		if err := j.ValidarPara(s); err != nil {
			t.Fatal(err)
		}
	}
	// Recuperar el antecedente no depende de la vigencia efímera antigua.
	j.Respuesta.RegistradaEn = j.Seleccion.Procedencia.Evidencia.RetenerHasta.Add(time.Hour)
	if err := j.ValidarPara(s); err != nil {
		t.Fatal("la consulta convirtió evidencia histórica en una sesión vigente", err)
	}
	if s.ClaveIdempotencia == j.Respuesta.Solicitud.ClaveIdempotencia {
		t.Fatal("fixture confundió intento de resolución con registro original")
	}
}

func TestConsultaJustificanteRespuestaRecibidaRechazaAntecedentesDesligados(t *testing.T) {
	s, original := justificanteRespuestaRecibidaPrueba(t)
	for nombre, cambiar := range map[string]func(*JustificanteRespuestaRecibida){
		"justificante": func(j *JustificanteRespuestaRecibida) { j.Respuesta.JustificanteRef += "otro" },
		"respuesta": func(j *JustificanteRespuestaRecibida) {
			j.Respuesta.Solicitud.Respuesta = RespuestaLlamamientoRenunciada
		},
		"organizacion":               func(j *JustificanteRespuestaRecibida) { j.Respuesta.Solicitud.OrganizacionRef += "otra" },
		"expediente":                 func(j *JustificanteRespuestaRecibida) { j.Respuesta.Solicitud.ExpedienteRef += "otro" },
		"llamamiento":                func(j *JustificanteRespuestaRecibida) { j.Respuesta.Solicitud.LlamamientoRef += "otro" },
		"comunicacion":               func(j *JustificanteRespuestaRecibida) { j.Respuesta.Solicitud.ComunicacionRef += "otra" },
		"recibo_respuesta":           func(j *JustificanteRespuestaRecibida) { j.Respuesta.ReciboRef = "" },
		"replay_no_es_original":      func(j *JustificanteRespuestaRecibida) { j.Respuesta.Estado = EstadoRespuestaRecibidaReplay },
		"sin_propuesta":              func(j *JustificanteRespuestaRecibida) { j.Seleccion.PropuestaGenerada = false },
		"version_seleccion":          func(j *JustificanteRespuestaRecibida) { j.Seleccion.VersionExpediente = 7 },
		"seleccion_otra_org":         func(j *JustificanteRespuestaRecibida) { j.Seleccion.OrganizacionRef += "otra" },
		"seleccion_otro_exp":         func(j *JustificanteRespuestaRecibida) { j.Seleccion.ExpedienteRef += "otro" },
		"seleccion_otro_llamamiento": func(j *JustificanteRespuestaRecibida) { j.Seleccion.LlamamientoRef += "otro" },
		"operacion":                  func(j *JustificanteRespuestaRecibida) { j.Seleccion.OperacionRef = "" },
		"referencia_versionada":      func(j *JustificanteRespuestaRecibida) { j.Seleccion.Propuesta.Version = 0 },
		"sin_confirmacion":           func(j *JustificanteRespuestaRecibida) { j.Seleccion.ConfirmadaEn = time.Time{} },
		"no_utc": func(j *JustificanteRespuestaRecibida) {
			j.Seleccion.ConfirmadaEn = j.Seleccion.ConfirmadaEn.In(time.FixedZone("Local", 0))
		},
		"submicro": func(j *JustificanteRespuestaRecibida) {
			j.Seleccion.ConfirmadaEn = j.Seleccion.ConfirmadaEn.Add(time.Nanosecond)
		},
		"sin_procedencia": func(j *JustificanteRespuestaRecibida) { j.Seleccion.Procedencia.AutoridadRef = "" },
	} {
		t.Run(nombre, func(t *testing.T) {
			j := original
			cambiar(&j)
			if err := j.ValidarPara(s); !errors.Is(err, ErrResultadoRespuestaRecibidaNoConfiable) {
				t.Fatal("antecedente inválido admitido", err)
			}
		})
	}
	s.VersionEsperada = 3
	if original.ValidarPara(s) == nil {
		t.Fatal("versión terminal admitida")
	}
	s.VersionEsperada, s.Respuesta, s.PruebaRespuestaRef = 2, RespuestaLlamamientoExpirada, ""
	if original.ValidarPara(s) == nil {
		t.Fatal("expiración convertida en respuesta recibida")
	}
}
