package cobertura

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReciboDenegadoVECOperacionDecisionCoberturaLigaPruebaSinEfectoC2(
	t *testing.T,
) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	reserva := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	base := reciboDenegadoVECOperacionDecisionCoberturaPrueba(t, solicitud)
	if base.ValidarPara(consulta) != nil ||
		base.ValidarParaReservaCongelada(solicitud, reserva) != nil {
		t.Fatal("denegación terminal válida rechazada")
	}
	if _, aplicada := base.ResultadoAplicado(); aplicada {
		t.Fatal("denegación fingió efecto C2")
	}
	if _, denegada := base.ResultadoDenegadoVEC(); !denegada {
		t.Fatal("denegación perdió su discriminante terminal")
	}
	casos := map[string]func(*ReciboOperacionDecisionCobertura){
		"decisión VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.DecisionVECRef = ""
		},
		"huella VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.DecisionVECHuellaSHA256 = strings.Repeat("0", 64)
		},
		"código ausente": func(r *ReciboOperacionDecisionCobertura) {
			r.CodigoProbatorioVEC = ""
		},
		"código de concesión": func(r *ReciboOperacionDecisionCobertura) {
			r.CodigoProbatorioVEC = "concedida"
		},
		"código inventado": func(r *ReciboOperacionDecisionCobertura) {
			r.CodigoProbatorioVEC = "denegada_por_rrhh"
		},
		"concesión contradictoria": func(r *ReciboOperacionDecisionCobertura) {
			r.ConcedidaVEC = true
		},
		"rama ausente": func(r *ReciboOperacionDecisionCobertura) {
			r.DenegadaVEC = nil
		},
		"efecto C2 residual": func(r *ReciboOperacionDecisionCobertura) {
			huella := strings.Repeat("4", 64)
			r.Aplicada = &ResultadoAplicadoOperacionDecisionCobertura{
				DecisionCoberturaRef:    "decision-cobertura:sha256:" + huella,
				DecisionCoberturaHuella: huella,
				VersionResultante:       2,
				EventoRef:               "evento_residual_01",
				ActuacionRef:            "actuacion_residual_01",
			}
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			recibo := clonarReciboOperacionDecisionCobertura(base)
			mutar(&recibo)
			if recibo.ValidarPara(consulta) == nil {
				t.Fatal("denegación contradictoria aceptada")
			}
		})
	}
}

func TestReservaNoExigeResultadosQueSoloNacenTrasEvaluar(t *testing.T) {
	tipo := reflect.TypeOf(DatosReservaPropietariaOperacionDecisionCobertura{})
	for _, campo := range []string{
		"DecisionVECHuellaSHA256",
		"DecisionCoberturaRef",
		"DecisionCoberturaHuella",
		"VersionResultante",
		"CodigoProbatorioVEC",
		"ConcedidaVEC",
	} {
		if _, existe := tipo.FieldByName(campo); existe {
			t.Fatalf("la reserva congeló prematuramente %s", campo)
		}
	}
	if _, existe := tipo.FieldByName("DecisionVECRef"); !existe {
		t.Fatal("la reserva perdió la referencia VEC preasignable")
	}
	for _, campo := range []string{"AnalisisRef", "AnalisisHuellaSHA256"} {
		if _, existe := tipo.FieldByName(campo); !existe {
			t.Fatalf("la reserva propietaria no congeló %s", campo)
		}
	}
}

func TestReplayConfirmadoNoDependeDelTokenPropietarioOriginal(t *testing.T) {
	consulta, solicitudOriginal := solicitudReservaOperacionDecisionCoberturaPrueba(
		t,
		identidadOperacionDecisionCoberturaPrueba(),
	)
	datosTerminal := datosReservaTerminalOperacionDecisionCoberturaPrueba(
		t,
		solicitudOriginal,
	)
	reservaTerminal, err :=
		RehidratarReservaTerminalOperacionDecisionCobertura(
			consulta,
			datosTerminal,
		)
	if err != nil {
		t.Fatal(err)
	}
	recibo := reciboOperacionDecisionCoberturaPrueba(t, solicitudOriginal)
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaConfirmada(
		consulta,
		reservaTerminal,
		recibo,
	)
	if err != nil {
		t.Fatal(err)
	}

	tokenNuevo, err := GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		t.Fatal(err)
	}
	solicitudNueva, err := NuevaSolicitudReservarOperacionDecisionCobertura(
		consulta,
		tokenNuevo,
	)
	if err != nil {
		t.Fatal(err)
	}
	estado, err := preparacion.EstadoPara(solicitudNueva)
	releido, errRecibo := preparacion.ReciboConfirmadoPara(consulta)
	if err != nil || errRecibo != nil ||
		estado != PreparacionOperacionDecisionCoberturaConfirmada ||
		releido.ReciboRef != recibo.ReciboRef {
		t.Fatalf("replay posterior al reinicio rechazado: %v / %v", err, errRecibo)
	}

	tipo := reflect.TypeOf(datosTerminal)
	for indice := 0; indice < tipo.NumField(); indice++ {
		nombre := strings.ToLower(tipo.Field(indice).Name)
		for _, prohibido := range []string{
			"token", "clave", "actor", "perfil", "motivo", "predecesora",
			"agregado", "propiedadhasta",
		} {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf(
					"fila terminal mínima expuso %s",
					tipo.Field(indice).Name,
				)
			}
		}
	}
	for _, campo := range []string{"AnalisisRef", "AnalisisHuellaSHA256"} {
		if _, existe := tipo.FieldByName(campo); existe {
			t.Fatalf(
				"la terminal expuso %s sin vínculo HMAC terminal versionado",
				campo,
			)
		}
	}

	for _, superficie := range []reflect.Type{
		reflect.TypeOf(DatosSolicitudReservarOperacionDecisionCobertura{}),
		reflect.TypeOf(DatosIdentidadOperacionDecisionCobertura{}),
	} {
		for _, campo := range []string{"AnalisisRef", "AnalisisHuellaSHA256"} {
			if _, existe := superficie.FieldByName(campo); existe {
				t.Fatalf("una entrada cliente expuso autoridad %s", campo)
			}
		}
	}
}

func TestConstructorConfirmadoExigeReservaTerminalNominal(t *testing.T) {
	tipo := reflect.TypeOf(
		NuevaPreparacionOperacionDecisionCoberturaConfirmada,
	)
	tipoTerminal := reflect.TypeOf(ReservaTerminalOperacionDecisionCobertura{})
	if tipo.NumIn() != 3 || tipo.In(1) != tipoTerminal {
		t.Fatalf("superficie confirmada débil: %v", tipo)
	}
}

func TestReplayConfirmadoRechazaFilaTerminalAdulterada(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t,
		identidadOperacionDecisionCoberturaPrueba(),
	)
	base := datosReservaTerminalOperacionDecisionCoberturaPrueba(t, solicitud)
	recibo := reciboOperacionDecisionCoberturaPrueba(t, solicitud)
	casos := map[string]func(*DatosReservaTerminalOperacionDecisionCobertura){
		"organización": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.OrganizacionRef = "organizacion_otra"
		},
		"expediente": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.ExpedienteRef = "expediente_otro"
		},
		"versión": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.VersionExpediente++
		},
		"reserva": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.ReservaRef = "reserva_decision_cobertura_otra"
		},
		"recibo": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.ReciboRef = "recibo_decision_cobertura_otro"
		},
		"actuación": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.ActuacionRef = "actuacion_decision_cobertura_otra"
		},
		"auditoría": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.AuditoriaRef = "auditoria_decision_cobertura_otra"
		},
		"evento": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.EventoRef = "evento_decision_cobertura_otro"
		},
		"correlación VEC": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.CorrelacionVECRef = "correlacion_vec_decision_cobertura_otra"
		},
		"decisión VEC": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.DecisionVECRef = "decision_vec_autorizacion_otra"
		},
		"ámbito HMAC": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.AmbitoIdempotenciaHMAC = strings.Replace(
				d.AmbitoIdempotenciaHMAC,
				strings.Repeat("a", 64),
				strings.Repeat("7", 64),
				1,
			)
		},
		"semántica HMAC": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.HuellaSemanticaHMAC = strings.Replace(
				d.HuellaSemanticaHMAC,
				strings.Repeat("c", 64),
				strings.Repeat("7", 64),
				1,
			)
		},
		"cercado": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.RevisionCercado++
		},
		"cronología": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.ObservadaEnDB = recibo.ConfirmadaEn.Add(time.Microsecond)
		},
		"tiempo no UTC": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.ObservadaEnDB = d.ObservadaEnDB.In(
				time.FixedZone("local", 3600),
			)
		},
		"tiempo submicro": func(d *DatosReservaTerminalOperacionDecisionCobertura) {
			d.ObservadaEnDB = d.ObservadaEnDB.Add(time.Nanosecond)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			mutar(&datos)
			terminal, err :=
				RehidratarReservaTerminalOperacionDecisionCobertura(
					consulta,
					datos,
				)
			if err == nil {
				_, err = NuevaPreparacionOperacionDecisionCoberturaConfirmada(
					consulta,
					terminal,
					recibo,
				)
			}
			if !errors.Is(
				err,
				ErrOperacionDecisionCoberturaIdempotenteInvalida,
			) {
				t.Fatalf("fila terminal adulterada aceptada: %v", err)
			}
		})
	}
}
