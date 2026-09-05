package ports

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSolicitudResolverLlamamientoRevisionManualYProyeccionLegacy(t *testing.T) {
	s := solicitudResolverComunicacionPrueba(RespuestaLlamamientoAceptada)
	s.VersionEsperada = 2
	legacy, err := json.Marshal(s)
	if err != nil || s.Validar() != nil || s.RevisionManualConfirmada() {
		t.Fatal("intención legacy alterada", err)
	}
	var ocho map[string]json.RawMessage
	if json.Unmarshal(legacy, &ocho) != nil || len(ocho) != 8 {
		t.Fatal("el material legacy ya no tiene ocho campos")
	}
	manual := s
	manual.RevisionRespuestaRRHH, manual.RevisionPlazoRRHH = true, true
	manual.CriterioValidacionRef = "criterio:rrhh-sintetico"
	if manual.Validar() != nil || !manual.RevisionManualConfirmada() || manual.ParaConsultaJustificante() != s {
		t.Fatal("revisión completa o proyección incorrectas")
	}
	proyectado, _ := json.Marshal(manual.ParaConsultaJustificante())
	if string(proyectado) != string(legacy) || !manual.RevisionManualConfirmada() {
		t.Fatal("proyección cambió bytes legacy o mutó el original")
	}
	b, _ := json.Marshal(manual)
	var once map[string]json.RawMessage
	if json.Unmarshal(b, &once) != nil || len(once) != 11 || string(once["RevisionRespuestaRRHH"]) != "true" ||
		string(once["RevisionPlazoRRHH"]) != "true" || string(once["CriterioValidacionRef"]) != `"criterio:rrhh-sintetico"` {
		t.Fatal("campos manuales no conservan nombres y valores")
	}
	for nombre, mutar := range map[string]func(*SolicitudResolverLlamamiento){
		"sin_respuesta":     func(s *SolicitudResolverLlamamiento) { s.RevisionRespuestaRRHH = false },
		"sin_plazo":         func(s *SolicitudResolverLlamamiento) { s.RevisionPlazoRRHH = false },
		"sin_criterio":      func(s *SolicitudResolverLlamamiento) { s.CriterioValidacionRef = "" },
		"criterio_invalido": func(s *SolicitudResolverLlamamiento) { s.CriterioValidacionRef = "no válido" },
		"renuncia":          func(s *SolicitudResolverLlamamiento) { s.Respuesta = RespuestaLlamamientoRenunciada },
		"otra_version":      func(s *SolicitudResolverLlamamiento) { s.VersionEsperada = 3 },
		"solo_criterio":     func(s *SolicitudResolverLlamamiento) { s.RevisionRespuestaRRHH, s.RevisionPlazoRRHH = false, false },
	} {
		t.Run(nombre, func(t *testing.T) {
			otra := manual
			mutar(&otra)
			if otra.Validar() == nil || otra.RevisionManualConfirmada() {
				t.Fatal("revisión incompleta aceptada")
			}
		})
	}
}

const (
	claveRegistroComunicacionPrueba = "018f47a6-5d2b-4c10-8a11-1234567890ab"
	claveResolverComunicacionPrueba = "118f47a6-5d2b-4c10-9a11-1234567890ab"
)

func TestComunicacionProbatoriaLocalNoAcreditaEntregaNiPlazo(t *testing.T) {
	s := solicitudRegistrarComunicacionPrueba()
	c := comunicacionProbatoriaPrueba(s)
	c.Estado = ResultadoComunicacionLlamamientoLocal
	c.RegistradaEn = c.EntregadaEn
	c.EntregadaEn = time.Time{}
	c.RespuestaHasta = time.Time{}
	c.IntencionEnvioRef = "outbox:local"
	if c.ValidarPara(s) != nil || !c.EsRegistroLocal() || c.EsReplayConfirmado() {
		t.Fatal("registro local rechazado")
	}
	c.Estado = ResultadoComunicacionLlamamientoReplayLocal
	if c.ValidarPara(s) != nil || !c.EsReplayConfirmado() {
		t.Fatal("replay local rechazado")
	}
	for nombre, mutar := range map[string]func(*ComunicacionProbatoria){
		"entrega":           func(c *ComunicacionProbatoria) { c.EntregadaEn = c.RegistradaEn },
		"plazo":             func(c *ComunicacionProbatoria) { c.RespuestaHasta = c.RegistradaEn.Add(time.Hour) },
		"sin_fecha":         func(c *ComunicacionProbatoria) { c.RegistradaEn = time.Time{} },
		"sin_outbox":        func(c *ComunicacionProbatoria) { c.IntencionEnvioRef = "" },
		"estado_probatorio": func(c *ComunicacionProbatoria) { c.Estado = ResultadoComunicacionLlamamientoConfirmado },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := c
			mutar(&alterada)
			if alterada.ValidarPara(s) == nil {
				t.Fatal("registro local falseado aceptado")
			}
		})
	}
}

func TestComunicacionProbatoriaExigePoliticaCanalPlazoYVersion(t *testing.T) {
	solicitud := solicitudRegistrarComunicacionPrueba()
	comunicacion := comunicacionProbatoriaPrueba(solicitud)
	if err := comunicacion.ValidarPara(solicitud); err != nil {
		t.Fatalf("comunicacion valida rechazada: %v", err)
	}
	replay := comunicacion
	replay.Estado = ResultadoComunicacionLlamamientoReplay
	if err := replay.ValidarPara(solicitud); err != nil || !replay.EsReplayConfirmado() {
		t.Fatalf("replay valido rechazado: replay=%v err=%v", replay.EsReplayConfirmado(), err)
	}

	mutaciones := []struct {
		nombre string
		mutar  func(*ComunicacionProbatoria)
	}{
		{"solicitud", func(c *ComunicacionProbatoria) { c.Solicitud.ExpedienteRef = "expediente:otro" }},
		{"canal", func(c *ComunicacionProbatoria) { c.Canal.Version = 0 }},
		{"politica", func(c *ComunicacionProbatoria) { c.Politica.HuellaSHA256 = strings.Repeat("0", 64) }},
		{"auditoria", func(c *ComunicacionProbatoria) { c.AuditoriaRef = "" }},
		{"version", func(c *ComunicacionProbatoria) { c.VersionResultante++ }},
		{"plazo", func(c *ComunicacionProbatoria) { c.RespuestaHasta = c.EntregadaEn }},
		{"estado", func(c *ComunicacionProbatoria) { c.Estado = "inventado" }},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := comunicacion
			caso.mutar(&alterada)
			if alterada.ValidarPara(solicitud) == nil {
				t.Fatal("resultado alterado aceptado")
			}
		})
	}
}

func TestSolicitudResolverLlamamientoSeparaRespuestaDeExpiracion(t *testing.T) {
	for _, respuesta := range []RespuestaLlamamiento{
		RespuestaLlamamientoAceptada,
		RespuestaLlamamientoRenunciada,
	} {
		solicitud := solicitudResolverComunicacionPrueba(respuesta)
		if err := solicitud.Validar(); err != nil {
			t.Fatalf("respuesta %s rechazada: %v", respuesta, err)
		}
		solicitud.PruebaRespuestaRef = ""
		if solicitud.Validar() == nil {
			t.Fatalf("respuesta %s sin prueba aceptada", respuesta)
		}
	}

	expiracion := solicitudResolverComunicacionPrueba(RespuestaLlamamientoExpirada)
	if err := expiracion.Validar(); err != nil {
		t.Fatalf("expiracion gobernada rechazada: %v", err)
	}
	expiracion.PruebaRespuestaRef = "respuesta:declarada"
	if expiracion.Validar() == nil {
		t.Fatal("expiracion con respuesta personal declarada aceptada")
	}
}

func TestResultadoResolucionLlamamientoModelaOutboxLocal(t *testing.T) {
	casos := []struct {
		nombre    string
		respuesta RespuestaLlamamiento
		estado    EstadoResultadoComunicacionLlamamiento
		plazo     EstadoPlazoLlamamiento
		outbox    *EstadoOutboxSiguienteCandidato
	}{
		{"aceptacion_nueva", RespuestaLlamamientoAceptada, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoVigente, nil},
		{"aceptacion_replay", RespuestaLlamamientoAceptada, ResultadoComunicacionLlamamientoReplay, PlazoLlamamientoVigente, nil},
		{"renuncia_pendiente", RespuestaLlamamientoRenunciada, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoVigente, estadoOutbox(OutboxSiguienteCandidatoPendiente)},
		{"renuncia_pendiente_en_replay", RespuestaLlamamientoRenunciada, ResultadoComunicacionLlamamientoReplay, PlazoLlamamientoVigente, estadoOutbox(OutboxSiguienteCandidatoPendiente)},
		{"renuncia_despachada_local_en_replay", RespuestaLlamamientoRenunciada, ResultadoComunicacionLlamamientoReplay, PlazoLlamamientoVigente, estadoOutbox(OutboxSiguienteCandidatoDespachada)},
		{"renuncia_indeterminada_en_replay", RespuestaLlamamientoRenunciada, ResultadoComunicacionLlamamientoReplay, PlazoLlamamientoVigente, estadoOutbox(OutboxSiguienteCandidatoIndeterminada)},
		{"expiracion_pendiente", RespuestaLlamamientoExpirada, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoExpirado, estadoOutbox(OutboxSiguienteCandidatoPendiente)},
		{"expiracion_pendiente_en_replay", RespuestaLlamamientoExpirada, ResultadoComunicacionLlamamientoReplay, PlazoLlamamientoExpirado, estadoOutbox(OutboxSiguienteCandidatoPendiente)},
		{"expiracion_despachada_local_en_replay", RespuestaLlamamientoExpirada, ResultadoComunicacionLlamamientoReplay, PlazoLlamamientoExpirado, estadoOutbox(OutboxSiguienteCandidatoDespachada)},
		{"expiracion_indeterminada_en_replay", RespuestaLlamamientoExpirada, ResultadoComunicacionLlamamientoReplay, PlazoLlamamientoExpirado, estadoOutbox(OutboxSiguienteCandidatoIndeterminada)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := solicitudResolverComunicacionPrueba(caso.respuesta)
			resultado := resultadoResolucionComunicacionPrueba(
				solicitud,
				caso.estado,
				caso.plazo,
				caso.outbox,
			)
			if err := resultado.ValidarPara(solicitud); err != nil {
				t.Fatalf("resultado valido rechazado: %v", err)
			}
			if resultado.EsReplayConfirmado() != (caso.estado == ResultadoComunicacionLlamamientoReplay) {
				t.Fatal("estado de replay incoherente")
			}
		})
	}
}

func TestResultadoResolucionLlamamientoRechazaAvanceFalsoOParcial(t *testing.T) {
	renuncia := solicitudResolverComunicacionPrueba(RespuestaLlamamientoRenunciada)
	valido := resultadoResolucionComunicacionPrueba(
		renuncia,
		ResultadoComunicacionLlamamientoConfirmado,
		PlazoLlamamientoVigente,
		estadoOutbox(OutboxSiguienteCandidatoPendiente),
	)
	mutaciones := []struct {
		nombre string
		mutar  func(*ResultadoResolucionLlamamiento)
	}{
		{"sin_intencion", func(r *ResultadoResolucionLlamamiento) { r.IntencionSiguiente = IntencionOutboxSiguienteCandidato{} }},
		{"despachada_sin_replay", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.Estado = OutboxSiguienteCandidatoDespachada
		}},
		{"estado_externo_inventado", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.Estado = EstadoOutboxSiguienteCandidato("exito_bolsa")
		}},
		{"indeterminada_sin_replay", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.Estado = OutboxSiguienteCandidatoIndeterminada
		}},
		{"otra_solicitud", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.Solicitud.ExpedienteRef = "expediente:otra-operacion"
		}},
		{"otra_resolucion", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.ResolucionRef = "resolucion:otra-operacion"
		}},
		{"otro_llamamiento", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.LlamamientoRef = "llamamiento:otra-operacion"
		}},
		{"otra_clave", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.ClaveIdempotencia = "418f47a6-5d2b-4c10-ca11-1234567890ab"
		}},
		{"otra_version_esperada", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.VersionEsperada++
		}},
		{"otra_version_resultante", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.VersionResultante++
		}},
		{"comando_vacio", func(r *ResultadoResolucionLlamamiento) { r.IntencionSiguiente.ComandoOpacoRef = "" }},
		{"cronologia", func(r *ResultadoResolucionLlamamiento) {
			r.IntencionSiguiente.ActualizadaEn = r.ResueltaEn.Add(-time.Microsecond)
		}},
		{"recibo_no_local", func(r *ResultadoResolucionLlamamiento) { r.ReciboLocalRef = "" }},
		{"auditoria", func(r *ResultadoResolucionLlamamiento) { r.AuditoriaRef = "" }},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := valido
			caso.mutar(&alterado)
			if alterado.ValidarPara(renuncia) == nil {
				t.Fatal("avance falso o parcial aceptado")
			}
		})
	}

	aceptacion := solicitudResolverComunicacionPrueba(RespuestaLlamamientoAceptada)
	conOutbox := resultadoResolucionComunicacionPrueba(
		aceptacion,
		ResultadoComunicacionLlamamientoConfirmado,
		PlazoLlamamientoVigente,
		estadoOutbox(OutboxSiguienteCandidatoPendiente),
	)
	if conOutbox.ValidarPara(aceptacion) == nil {
		t.Fatal("aceptacion con solicitud de siguiente candidato aceptada")
	}

	expiracion := solicitudResolverComunicacionPrueba(RespuestaLlamamientoExpirada)
	sinIntencion := resultadoResolucionComunicacionPrueba(
		expiracion,
		ResultadoComunicacionLlamamientoConfirmado,
		PlazoLlamamientoExpirado,
		nil,
	)
	if sinIntencion.ValidarPara(expiracion) == nil {
		t.Fatal("expiracion nueva sin intencion pendiente aceptada")
	}

	otraOperacion := renuncia
	otraOperacion.ClaveIdempotencia = "518f47a6-5d2b-4c10-da11-1234567890ab"
	intencionAjena := resultadoResolucionComunicacionPrueba(
		otraOperacion,
		ResultadoComunicacionLlamamientoConfirmado,
		PlazoLlamamientoVigente,
		estadoOutbox(OutboxSiguienteCandidatoPendiente),
	).IntencionSiguiente
	conIntencionAjena := valido
	conIntencionAjena.IntencionSiguiente = intencionAjena
	if conIntencionAjena.ValidarPara(renuncia) == nil {
		t.Fatal("intencion de otra operacion aceptada")
	}
}

func TestContratoComunicacionNoExponeResultadoNiPersonaDeBolsa(t *testing.T) {
	tipo := reflect.TypeOf(ResultadoResolucionLlamamiento{})
	for _, campo := range []string{"ReciboBolsaRef", "EventoBolsaRef", "CandidatoRef", "SeleccionRef"} {
		if _, existe := tipo.FieldByName(campo); existe {
			t.Fatalf("campo de Bolsa o persona expuesto: %s", campo)
		}
	}
	for _, modelo := range []reflect.Type{
		reflect.TypeOf(SolicitudRegistrarComunicacionLlamamiento{}),
		reflect.TypeOf(ComunicacionProbatoria{}),
		reflect.TypeOf(SolicitudResolverLlamamiento{}),
		reflect.TypeOf(ResultadoResolucionLlamamiento{}),
		reflect.TypeOf(IntencionOutboxSiguienteCandidato{}),
	} {
		for indice := 0; indice < modelo.NumField(); indice++ {
			nombre := strings.ToLower(modelo.Field(indice).Name)
			for _, prohibido := range []string{"dni", "nie", "nombre", "correo", "telefono", "direccion", "persona"} {
				if strings.Contains(nombre, prohibido) {
					t.Fatalf("%s expone campo personal %s", modelo.Name(), nombre)
				}
			}
		}
	}
}

func solicitudRegistrarComunicacionPrueba() SolicitudRegistrarComunicacionLlamamiento {
	return SolicitudRegistrarComunicacionLlamamiento{
		ClaveIdempotencia: claveRegistroComunicacionPrueba,
		OrganizacionRef:   "organizacion:ct-comunicacion",
		ExpedienteRef:     "expediente:ct-comunicacion",
		LlamamientoRef:    "llamamiento:ct-comunicacion",
		VersionEsperada:   7,
		PruebaEntregaRef:  "entrega:probatoria",
	}
}

func comunicacionProbatoriaPrueba(
	solicitud SolicitudRegistrarComunicacionLlamamiento,
) ComunicacionProbatoria {
	entregada := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	respuestaHastaGobernada := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	return ComunicacionProbatoria{
		Solicitud: solicitud, ComunicacionRef: "comunicacion:probatoria",
		Canal:     referenciaGobernadaComunicacionPrueba("canal", "a"),
		Politica:  referenciaGobernadaComunicacionPrueba("politica", "b"),
		ReciboRef: "recibo:comunicacion-local", AuditoriaRef: "auditoria:comunicacion-local",
		VersionResultante: solicitud.VersionEsperada + 1,
		EntregadaEn:       entregada, RespuestaHasta: respuestaHastaGobernada,
		Estado: ResultadoComunicacionLlamamientoConfirmado,
	}
}

func solicitudResolverComunicacionPrueba(
	respuesta RespuestaLlamamiento,
) SolicitudResolverLlamamiento {
	prueba := "respuesta:probatoria"
	if respuesta == RespuestaLlamamientoExpirada {
		prueba = ""
	}
	return SolicitudResolverLlamamiento{
		ClaveIdempotencia: claveResolverComunicacionPrueba,
		OrganizacionRef:   "organizacion:ct-comunicacion", ExpedienteRef: "expediente:ct-comunicacion",
		LlamamientoRef: "llamamiento:ct-comunicacion", ComunicacionRef: "comunicacion:probatoria",
		VersionEsperada: 8, Respuesta: respuesta, PruebaRespuestaRef: prueba,
	}
}

func resultadoResolucionComunicacionPrueba(
	solicitud SolicitudResolverLlamamiento,
	estado EstadoResultadoComunicacionLlamamiento,
	plazo EstadoPlazoLlamamiento,
	estadoOutbox *EstadoOutboxSiguienteCandidato,
) ResultadoResolucionLlamamiento {
	resuelta := time.Date(2026, 8, 31, 11, 0, 0, 0, time.UTC)
	resultado := ResultadoResolucionLlamamiento{
		Solicitud: solicitud, Politica: referenciaGobernadaComunicacionPrueba("politica", "b"),
		EvaluacionPlazoRef: "evaluacion:plazo-gobernado", EstadoPlazo: plazo,
		ResolucionRef: "resolucion:llamamiento-local", ReciboLocalRef: "recibo:resolucion-local",
		AuditoriaRef:      "auditoria:resolucion-local",
		VersionResultante: solicitud.VersionEsperada + 1, ResueltaEn: resuelta, Estado: estado,
	}
	if estadoOutbox != nil {
		resultado.IntencionSiguiente = IntencionOutboxSiguienteCandidato{
			Solicitud: solicitud, ResolucionRef: resultado.ResolucionRef,
			LlamamientoRef: solicitud.LlamamientoRef, ClaveIdempotencia: solicitud.ClaveIdempotencia,
			VersionEsperada: solicitud.VersionEsperada, VersionResultante: resultado.VersionResultante,
			IntencionRef: "outbox:siguiente-candidato", ComandoOpacoRef: "comando:siguiente-candidato",
			Estado: *estadoOutbox, ActualizadaEn: resuelta,
		}
	}
	return resultado
}

func referenciaGobernadaComunicacionPrueba(
	referencia string,
	digito string,
) ReferenciaGobernadaComunicacionLlamamiento {
	return ReferenciaGobernadaComunicacionLlamamiento{
		Referencia: "catalogo:" + referencia, Version: 3, HuellaSHA256: strings.Repeat(digito, 64),
	}
}

func estadoOutbox(estado EstadoOutboxSiguienteCandidato) *EstadoOutboxSiguienteCandidato {
	return &estado
}
