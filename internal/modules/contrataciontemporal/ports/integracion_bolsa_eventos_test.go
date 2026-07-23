package ports

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestEventoExigeEnlaceGobernadoExactoYAutenticacion(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	evento, enlace := eventoYEnlaceBolsaPrueba(t, base)
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)

	comprobante, evidencia, err := verificador.VerificarEvento(
		context.Background(), evento, enlace, ahora,
	)
	if err != nil || comprobante.datos == nil || evidencia.Validar() != nil {
		t.Fatalf("evento auténtico no promovido: %v", err)
	}
	comando, err := NuevoComandoRegistrarEventoBolsa(evento, enlace, comprobante, ahora)
	if err != nil {
		t.Fatalf("comando verificado rechazado: %v", err)
	}
	recuperado, prueba, err := comando.Datos()
	if err != nil || recuperado != evento || prueba.datos == nil {
		t.Fatalf("comando no conserva evento y prueba: %v", err)
	}
	comandoSinPrueba := comandoLlamamientoPrueba(
		t, base, selladorRespuestaBolsaPrueba(),
	)
	reciboSinPrueba := reciboLlamamientoPrueba(t, comandoSinPrueba, base)
	if _, err := NuevoEnlaceEventoLlamamientoBolsa(
		PreparacionEnlaceEventoLlamamientoBolsa{
			Comando: comandoSinPrueba, Recibo: reciboSinPrueba,
			Comprobante: ComprobanteEvidenciaIntegracionBolsa{},
		},
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("recibo sin autenticar produjo enlace: %v", err)
	}
	if _, err := NuevoComandoRegistrarEventoBolsa(
		evento,
		enlace,
		ComprobanteEvidenciaIntegracionBolsa{},
		ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("evento sin comprobante obtuvo comando: %v", err)
	}

	mutada := evento
	mutada.Estado = referenciaBolsaPrueba("estado:otro", "4")
	mutada.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(mutada))
	if err := mutada.ValidarParaEn(enlace, ahora); err != nil {
		t.Fatalf("mutación nominal debía llegar al HMAC: %v", err)
	}
	if _, _, err := verificador.VerificarEvento(
		context.Background(), mutada, enlace, ahora,
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("evento alterado conservó autenticidad: %v", err)
	}

	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, _, err := verificador.VerificarEvento(
		cancelado, evento, enlace, ahora,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("se perdió cancelación: %v", err)
	}
}

func TestEventoNoPuedeAutovalidarFinalidadAccionRecursoNiReferencias(t *testing.T) {
	base := instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	original, enlace := eventoYEnlaceBolsaPrueba(t, base)
	sellador := selladorRespuestaBolsaPrueba()
	pruebas := []struct {
		nombre  string
		cambiar func(*EventoLlamamientoBolsa)
	}{
		{"organizacion", func(e *EventoLlamamientoBolsa) {
			e.OrganizacionRef = "organizacion:otra"
		}},
		{"expediente", func(e *EventoLlamamientoBolsa) {
			e.ExpedienteRef = "expediente:otro"
		}},
		{"version", func(e *EventoLlamamientoBolsa) {
			e.VersionExpedienteEsperada++
		}},
		{"correlacion", func(e *EventoLlamamientoBolsa) {
			e.CorrelacionRef = "correlacion:otra"
		}},
		{"finalidad", func(e *EventoLlamamientoBolsa) {
			e.Finalidad = referenciaBolsaPrueba("finalidad:otra", "2")
		}},
		{"accion", func(e *EventoLlamamientoBolsa) {
			e.Accion = referenciaBolsaPrueba("accion:otra", "2")
		}},
		{"recurso", func(e *EventoLlamamientoBolsa) {
			e.Recurso = referenciaBolsaPrueba("propuesta:otra", "2")
			e.Propuesta = e.Recurso
		}},
		{"peticion_ref", func(e *EventoLlamamientoBolsa) {
			e.PeticionRef = "operacion:otra"
		}},
		{"peticion_digest", func(e *EventoLlamamientoBolsa) {
			e.HuellaPeticionSHA256 = referenciaBolsaPrueba("huella:otra", "2").HuellaSHA256
		}},
		{"recibo_ref", func(e *EventoLlamamientoBolsa) {
			e.ReciboRef = "recibo:otro"
		}},
		{"recibo_digest", func(e *EventoLlamamientoBolsa) {
			e.HuellaReciboSHA256 = referenciaBolsaPrueba("huella:otra", "3").HuellaSHA256
		}},
		{"necesidad", func(e *EventoLlamamientoBolsa) {
			e.Necesidad = referenciaBolsaPrueba("necesidad:otra", "3")
		}},
		{"bolsa", func(e *EventoLlamamientoBolsa) {
			e.Bolsa = referenciaBolsaPrueba("bolsa:otra", "3")
		}},
		{"orden", func(e *EventoLlamamientoBolsa) {
			e.Orden = referenciaBolsaPrueba("orden:otra", "3")
		}},
		{"politica", func(e *EventoLlamamientoBolsa) {
			e.Politica = referenciaBolsaPrueba("politica:otra", "3")
		}},
		{"llamamiento", func(e *EventoLlamamientoBolsa) {
			e.LlamamientoRef = "llamamiento:otro"
		}},
		{"seleccion", func(e *EventoLlamamientoBolsa) {
			e.SeleccionRef = "seleccion:otra"
		}},
		{"retencion", func(e *EventoLlamamientoBolsa) {
			e.RetencionSeleccion = referenciaBolsaPrueba("retencion:otra", "3")
		}},
		{"peticion_solicitada", func(e *EventoLlamamientoBolsa) {
			e.PeticionSolicitadaEn = e.PeticionSolicitadaEn.Add(time.Microsecond)
		}},
		{"peticion_valida", func(e *EventoLlamamientoBolsa) {
			e.PeticionValidaHasta = e.PeticionValidaHasta.Add(-time.Microsecond)
		}},
		{"recibo_confirmado", func(e *EventoLlamamientoBolsa) {
			e.ReciboConfirmadaEn = e.ReciboConfirmadaEn.Add(-time.Microsecond)
		}},
		{"recibo_evidencia_emitida", func(e *EventoLlamamientoBolsa) {
			e.ReciboEvidenciaEmitidaEn = e.ReciboEvidenciaEmitidaEn.Add(time.Microsecond)
		}},
		{"recibo_evidencia_valida", func(e *EventoLlamamientoBolsa) {
			e.ReciboEvidenciaValidaHasta =
				e.ReciboEvidenciaValidaHasta.Add(-time.Microsecond)
		}},
		{"recibo_retener_hasta", func(e *EventoLlamamientoBolsa) {
			e.ReciboRetenerHasta = e.ReciboRetenerHasta.Add(-time.Microsecond)
			e.Procedencia.Evidencia.RetenerHasta = e.ReciboRetenerHasta
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			evento := original
			prueba.cambiar(&evento)
			evento.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(evento))
			firmarEventoBolsaPrueba(t, sellador, enlace, &evento)
			if err := evento.ValidarParaEn(enlace, ahora); !errors.Is(
				err,
				ErrEventoBolsaInvalido,
			) {
				t.Fatalf("relación alterada aceptada: %v", err)
			}
		})
	}
}

func TestEvidenciaEventoDurableSeReautenticaTrasReinicio(t *testing.T) {
	base := instanteBolsaPrueba()
	evento, enlace := eventoYEnlaceBolsaPrueba(t, base)
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	_, evidencia, err := verificador.VerificarEvento(
		context.Background(),
		evento,
		enlace,
		base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("autenticar evento: %v", err)
	}
	contenido, err := json.Marshal(evidencia)
	if err != nil {
		t.Fatalf("serializar evidencia: %v", err)
	}
	var recuperada EvidenciaDurableIntegracionBolsa
	if err := json.Unmarshal(contenido, &recuperada); err != nil {
		t.Fatalf("recuperar evidencia: %v", err)
	}
	if _, err := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV1Prueba,
	).reautenticarEvento(
		context.Background(),
		evento,
		enlace,
		recuperada,
		base.Add(24*time.Hour),
	); err != nil {
		t.Fatalf("evento durable no reautenticado tras reinicio: %v", err)
	}
	alterada := recuperada
	alterada.RespuestaRef = "respuesta:otra"
	if _, err := verificador.reautenticarEvento(
		context.Background(),
		evento,
		enlace,
		alterada,
		base.Add(24*time.Hour),
	); !errors.Is(err, ErrEvidenciaBolsaNoAutenticada) {
		t.Fatalf("evidencia de evento alterada aceptada: %v", err)
	}
}

func TestInboxDetectaReplayColisionYSecuenciaCAS(t *testing.T) {
	base := instanteBolsaPrueba()
	evento, _ := eventoYEnlaceBolsaPrueba(t, base)
	if err := ValidarIdentidadEventoBolsa(evento, evento); err != nil {
		t.Fatalf("replay idéntico rechazado: %v", err)
	}
	mutado := evento
	mutado.Estado = referenciaBolsaPrueba("estado:rectificado", "4")
	mutado.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(mutado))
	if err := ValidarIdentidadEventoBolsa(evento, mutado); !errors.Is(
		err,
		ErrColisionEventoBolsa,
	) {
		t.Fatalf("misma identidad con otra carga no fue colisión: %v", err)
	}
	otraAutoridad := evento
	otraAutoridad.Procedencia.AutoridadRef = "autoridad:otra"
	if err := ValidarIdentidadEventoBolsa(evento, otraAutoridad); !errors.Is(
		err,
		ErrEventoBolsaInvalido,
	) {
		t.Fatalf("otra autoridad compartió identidad: %v", err)
	}

	acuse := acuseEventoBolsaPrueba(evento, base.Add(4*time.Minute))
	if err := acuse.ValidarPara(evento); err != nil {
		t.Fatalf("acuse exacto rechazado: %v", err)
	}
	if err := ValidarReplayAcuseEventoBolsa(acuse, acuse, evento); err != nil {
		t.Fatalf("replay no devolvió mismo acuse: %v", err)
	}
	repetidoDistinto := acuse
	repetidoDistinto.InboxRef = "inbox:otro"
	if err := ValidarReplayAcuseEventoBolsa(
		acuse,
		repetidoDistinto,
		evento,
	); !errors.Is(err, ErrAcuseEventoBolsaNoConfiable) {
		t.Fatalf("replay fabricó otro acuse: %v", err)
	}

	alteraciones := []func(*AcuseEventoLlamamientoBolsa){
		func(a *AcuseEventoLlamamientoBolsa) { a.PeticionRef = "operacion:otra" },
		func(a *AcuseEventoLlamamientoBolsa) {
			a.HuellaPeticionSHA256 = referenciaBolsaPrueba("huella:otra", "3").HuellaSHA256
		},
		func(a *AcuseEventoLlamamientoBolsa) { a.ReciboRef = "recibo:otro" },
		func(a *AcuseEventoLlamamientoBolsa) {
			a.HuellaReciboSHA256 = referenciaBolsaPrueba("huella:otra", "4").HuellaSHA256
		},
		func(a *AcuseEventoLlamamientoBolsa) { a.Secuencia++ },
		func(a *AcuseEventoLlamamientoBolsa) { a.VersionAnterior++ },
		func(a *AcuseEventoLlamamientoBolsa) { a.VersionResultante++ },
	}
	for _, cambiar := range alteraciones {
		alterado := acuse
		cambiar(&alterado)
		if err := alterado.ValidarPara(evento); !errors.Is(
			err,
			ErrAcuseEventoBolsaNoConfiable,
		) {
			t.Fatalf("acuse no ligado aceptado: %+v err=%v", alterado, err)
		}
	}

	salto := evento
	salto.Secuencia++
	salto.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(salto))
	if err := salto.ValidarEn(base.Add(3 * time.Minute)); !errors.Is(
		err,
		ErrEventoBolsaInvalido,
	) {
		t.Fatalf("salto de secuencia aceptado: %v", err)
	}
}

func eventoYEnlaceBolsaPrueba(
	t *testing.T,
	instante time.Time,
) (EventoLlamamientoBolsa, EnlaceEventoLlamamientoBolsa) {
	t.Helper()
	sellador := selladorRespuestaBolsaPrueba()
	comando := comandoLlamamientoPrueba(t, instante, sellador)
	recibo := reciboLlamamientoPrueba(t, comando, instante)
	firmarLlamamientoPrueba(t, sellador, comando, &recibo)
	comprobante := autenticarReciboLlamamientoPrueba(
		t,
		comando,
		recibo,
		instante.Add(3*time.Minute),
	)
	enlace, err := NuevoEnlaceEventoLlamamientoBolsa(
		PreparacionEnlaceEventoLlamamientoBolsa{
			Comando: comando, Recibo: recibo, Comprobante: comprobante,
		},
	)
	if err != nil {
		t.Fatalf("crear enlace de evento: %v", err)
	}
	return nuevoEventoParaEnlaceBolsaPrueba(t, instante, enlace), enlace
}

func nuevoEventoParaEnlaceBolsaPrueba(
	t *testing.T,
	instante time.Time,
	enlace EnlaceEventoLlamamientoBolsa,
) EventoLlamamientoBolsa {
	t.Helper()
	sellador := selladorRespuestaBolsaPrueba()
	d := enlace.datos
	procedencia := procedenciaBolsaPrueba(instante)
	procedencia.RespuestaRef = "respuesta:evento:bolsa"
	evento := EventoLlamamientoBolsa{
		EventoRef:                  "evento:estado:llamamiento",
		OrganizacionRef:            d.organizacionRef,
		ExpedienteRef:              d.expedienteRef,
		VersionExpedienteEsperada:  d.versionExpediente,
		CorrelacionRef:             d.correlacionRef,
		Finalidad:                  d.finalidad,
		Accion:                     d.accion,
		Recurso:                    d.recurso,
		PeticionRef:                d.peticionRef,
		HuellaPeticionSHA256:       d.huellaPeticion,
		ReciboRef:                  d.reciboRef,
		HuellaReciboSHA256:         d.huellaRecibo,
		PeticionSolicitadaEn:       d.peticionSolicitadaEn,
		PeticionValidaHasta:        d.peticionValidaHasta,
		ReciboConfirmadaEn:         d.reciboConfirmadaEn,
		ReciboEvidenciaEmitidaEn:   d.reciboEvidenciaEmitidaEn,
		ReciboEvidenciaValidaHasta: d.reciboEvidenciaValidaHasta,
		ReciboRetenerHasta:         d.reciboRetenerHasta,
		Necesidad:                  d.necesidad,
		Bolsa:                      d.bolsa,
		Orden:                      d.orden,
		Politica:                   d.politica,
		Propuesta:                  d.recurso,
		LlamamientoRef:             d.llamamientoRef,
		SeleccionRef:               d.seleccionRef,
		RetencionSeleccion:         d.retencionSeleccion,
		Tipo:                       referenciaBolsaPrueba("evento_tipo:estado_llamamiento", "5"),
		Estado:                     referenciaBolsaPrueba("estado:llamamiento", "7"),
		SecuenciaAnterior:          1,
		Secuencia:                  2,
		OcurridoEn:                 instante.Add(time.Minute),
		PublicadoEn:                instante.Add(2 * time.Minute),
		Procedencia:                procedencia,
	}
	evento.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(evento))
	firmarEventoBolsaPrueba(t, sellador, enlace, &evento)
	return evento
}

func firmarEventoBolsaPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	enlace EnlaceEventoLlamamientoBolsa,
	evento *EventoLlamamientoBolsa,
) {
	t.Helper()
	firmarRespuestaBolsaPrueba(
		t,
		sellador,
		"evento_llamamiento",
		evento.PeticionRef,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(*evento),
		&evento.Procedencia,
	)
}

func acuseEventoBolsaPrueba(
	evento EventoLlamamientoBolsa,
	instante time.Time,
) AcuseEventoLlamamientoBolsa {
	return AcuseEventoLlamamientoBolsa{
		AutoridadRef: evento.Procedencia.AutoridadRef, EventoRef: evento.EventoRef,
		OrganizacionRef: evento.OrganizacionRef, ExpedienteRef: evento.ExpedienteRef,
		CorrelacionRef: evento.CorrelacionRef,
		PeticionRef:    evento.PeticionRef, HuellaPeticionSHA256: evento.HuellaPeticionSHA256,
		ReciboRef: evento.ReciboRef, HuellaReciboSHA256: evento.HuellaReciboSHA256,
		HuellaEventoSHA256: evento.HuellaCargaSHA256,
		SecuenciaAnterior:  evento.SecuenciaAnterior, Secuencia: evento.Secuencia,
		VersionAnterior:   evento.VersionExpedienteEsperada,
		VersionResultante: evento.VersionExpedienteEsperada + 1,
		ActuacionRef:      "actuacion:llamamiento", AuditoriaRef: "auditoria:llamamiento",
		InboxRef: "inbox:evento:llamamiento", RegistradoEn: instante,
	}
}
