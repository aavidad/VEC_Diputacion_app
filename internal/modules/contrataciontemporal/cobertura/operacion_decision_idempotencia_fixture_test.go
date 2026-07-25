package cobertura

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var instanteOperacionDecisionCoberturaPrueba = time.Date(
	2026, 7, 25, 8, 0, 0, 0, time.UTC,
)

type revalidadorVinculoOperacionDecisionCoberturaPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (d revalidadorVinculoOperacionDecisionCoberturaPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.resultado, nil
}

type resolutorVinculoOperacionDecisionCoberturaPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d resolutorVinculoOperacionDecisionCoberturaPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

type relojVinculoOperacionDecisionCoberturaPrueba struct{ instante time.Time }

func (d relojVinculoOperacionDecisionCoberturaPrueba) Ahora() time.Time {
	return d.instante
}

func contextoAutorizacionOperacionDecisionCoberturaPrueba(
	t *testing.T,
) (
	ports.ContextoAutorizacionAltaV3,
	ports.SolicitudResolverContextoAutorizacionAltaV3,
) {
	t.Helper()
	ahora := instanteOperacionDecisionCoberturaPrueba
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdefghijkl",
		VinculoVersion:  3,
		CuentaRef:       cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      "per_0123456789abcdefghijkl",
		PersonaVersion:  2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl",
		PerfilVersion:   5,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta,
		instantanea,
		ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:     "prc_0123456789abcdefghijkl",
		ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat(
			"4",
			64,
		),
		ProcedenciaAutoridad: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema: dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: instantanea.CuentaRef,
			Version:   instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef,
			Version:    instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef,
			Version:   instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef,
			Version:    instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoHuella, err :=
		dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
			manifiestoCanon,
		)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef:               "rca_0123456789abcdefghijklmn",
		Contexto:                          actor,
		RepresentacionCanonica:            canon,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: actor.ResueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl",
		SesionRef:                 "ses_0123456789abcdefghijkl",
		ControlSesionRef:          "cse_0123456789abcdefghijkl",
		ControlSesionRevision:     2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef,
		CuentaOrdinariaRef:        cuenta.CuentaRef,
		Superficie: dominiovec.
			SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:     cuenta.Metodo,
		GarantiaObservada:   cuenta.Garantia,
		PoliticaGarantiaRef: "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat(
			"3",
			64,
		),
		AutenticacionVerificadaEn: ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:           ahora.Add(-9 * time.Minute),
		SesionValidaHasta:         ahora.Add(20 * time.Minute),
		SesionRevalidadaEn:        ahora.Add(-3 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorVinculoOperacionDecisionCoberturaPrueba{
			resultado: autenticacion,
		},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorVinculoOperacionDecisionCoberturaPrueba{
			resultado: resultado,
		},
		dominiovec.SolicitudContextoActor{
			Cuenta:          cuenta,
			PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		relojVinculoOperacionDecisionCoberturaPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{
			Vinculo: vinculo, Resultado: resultado,
		},
		ports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
			PerfilRef:        instantanea.PerfilActivoRef,
		}
}

func identidadOperacionDecisionCoberturaPrueba() DatosIdentidadOperacionDecisionCobertura {
	huella := strings.Repeat("a", 64)
	return DatosIdentidadOperacionDecisionCobertura{
		claveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		tipo:              domain.DecisionCoberturaInicial,
		organizacionRef:   "organizacion_diputacion_granada",
		expedienteRef:     "expediente_temporal_2026_5487",
		versionExpediente: 1,
		actorRef:          "actor_rrhh_opaco_01",
		perfilRef:         "perfil_rrhh_decisor_01",
		accion:            domain.AccionDecidirCoberturaGobernada,
		viaElegida:        "bolsa_vigente",
		identidadSemantica: domain.IdentidadSemanticaPropuestaDecisionCobertura{
			Referencia:   "propuesta-cobertura-semantica:sha256:" + huella,
			HuellaSHA256: huella,
			Canon: domain.
				CanonHuellaSemanticaPropuestaDecisionCoberturaV1(),
		},
	}
}

func motivoOperacionDecisionCoberturaPrueba(
	caracter string,
) domain.MotivoGobernadoDecisionCobertura {
	return domain.MotivoGobernadoDecisionCobertura{
		ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_decision_cobertura",
			CatalogoVersion:      3,
			CatalogoHuellaSHA256: strings.Repeat(caracter, 64),
			EntradaClave:         "motivo_funcional_cobertura",
		},
		ClaveI18n: "contratacion_temporal.cobertura.motivo_funcional",
	}
}

func identidadRectificacionDecisionCoberturaPrueba() DatosIdentidadOperacionDecisionCobertura {
	datos := identidadOperacionDecisionCoberturaPrueba()
	datos.tipo = domain.DecisionCoberturaRectificacion
	datos.accion = domain.AccionRectificarCoberturaGobernada
	datos.versionExpediente = 2
	datos.viaElegida = "oferta_sae"
	datos.motivo = motivoOperacionDecisionCoberturaPrueba("b")
	datos.predecesoraHuella = strings.Repeat("c", 64)
	datos.predecesoraRef = "decision-cobertura:sha256:" +
		datos.predecesoraHuella
	return datos
}

func selloOperacionDecisionCoberturaPrueba(
	dominio string,
	generacion uint32,
	caracter string,
) string {
	return "hmac-sha256:" + dominio + "/v" +
		strconv.FormatUint(uint64(generacion), 10) + ":" +
		strings.Repeat(caracter, 64)
}

func sellosOperacionDecisionCoberturaPrueba(
	t *testing.T,
) SellosOperacionDecisionCobertura {
	t.Helper()
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		selloOperacionDecisionCoberturaPrueba(
			dominioAmbitoOperacionDecisionCobertura, 2, "a",
		),
		[]string{selloOperacionDecisionCoberturaPrueba(
			dominioAmbitoOperacionDecisionCobertura, 1, "b",
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	semanticas, err := ports.NuevaColeccionSellosHMAC(
		selloOperacionDecisionCoberturaPrueba(
			dominioSemanticaOperacionDecisionCobertura, 2, "c",
		),
		[]string{selloOperacionDecisionCoberturaPrueba(
			dominioSemanticaOperacionDecisionCobertura, 1, "d",
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	return SellosOperacionDecisionCobertura{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasSemanticasHMAC:   semanticas,
	}
}

func solicitudReservaOperacionDecisionCoberturaPrueba(
	t *testing.T,
	identidad DatosIdentidadOperacionDecisionCobertura,
) (
	SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	SolicitudReservarOperacionDecisionCobertura,
) {
	t.Helper()
	consulta, err := NuevaSolicitudConsultarOperacionDecisionCoberturaConfirmada(
		identidad,
		sellosOperacionDecisionCoberturaPrueba(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	token, err := GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := NuevaSolicitudReservarOperacionDecisionCobertura(
		consulta, token,
	)
	if err != nil {
		t.Fatal(err)
	}
	return consulta, solicitud
}

func expedienteOperacionDecisionCoberturaPrueba(t *testing.T) domain.Expediente {
	t.Helper()
	inicio := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	fin := time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente_temporal_2026_5487",
		OrganizacionRef: "organizacion_diputacion_granada",
		NumeroVisible:   "2026/5487",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo_contratacion_temporal",
			Version:       3,
			HuellaSHA256:  strings.Repeat("e", 64),
		},
		FaseInicial: "solicitud",
		Solicitud: domain.SolicitudCentro{
			CentroRef: "centro_social_01", ContactoRef: "contacto_opaco_01",
			CategoriaRef: "categoria_trabajador_social", GrupoSubgrupo: "A2",
			MotivoClave: "sustitucion_it",
			Detalle:     "Necesidad temporal para prueba del contrato.",
			Periodo:     domain.PeriodoPrevisto{Inicio: inicio, Fin: fin},
			RC:          domain.DeclaracionRC{},
		},
		Actuacion: domain.DatosActuacion{
			AccionClave: "solicitud.registrada", ActorRef: "actor_rrhh_opaco_01",
			UnidadRef: "unidad_rrhh_01", ReciboRef: "recibo_alta_01",
			RealizadaEn: instanteOperacionDecisionCoberturaPrueba,
			FaseDestino: "solicitud", EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		t.Fatalf("crear expediente de prueba: %v", err)
	}
	return expediente
}

func datosPropiedadOperacionDecisionCoberturaPrueba(
	t *testing.T,
	solicitud SolicitudReservarOperacionDecisionCobertura,
) DatosReservaPropietariaOperacionDecisionCobertura {
	t.Helper()
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	expediente := expedienteOperacionDecisionCoberturaPrueba(t)
	return DatosReservaPropietariaOperacionDecisionCobertura{
		ReservaRef:              "reserva_decision_cobertura_01",
		ReciboRef:               "recibo_decision_cobertura_01",
		ActuacionRef:            "actuacion_decision_cobertura_01",
		AuditoriaRef:            "auditoria_decision_cobertura_01",
		EventoRef:               "evento_decision_cobertura_01",
		CorrelacionVECRef:       "correlacion_vec_decision_cobertura_01",
		DecisionVECRef:          "decision_vec_autorizacion_01",
		TokenPropietarioSHA256:  datosSolicitud.TokenPropietarioSHA256,
		AmbitoIdempotenciaHMAC:  datosSolicitud.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     datosSolicitud.HuellaSemanticaHMAC,
		AgregadoAnterior:        &expediente,
		RevisionCercadoAnterior: 0,
		RevisionCercado:         1,
		ObservadaEnDB:           instanteOperacionDecisionCoberturaPrueba,
		PropiedadHasta: instanteOperacionDecisionCoberturaPrueba.Add(
			MaximoLeaseOperacionDecisionCobertura,
		),
	}
}

func datosReservaTerminalOperacionDecisionCoberturaPrueba(
	t *testing.T,
	solicitud SolicitudReservarOperacionDecisionCobertura,
) DatosReservaTerminalOperacionDecisionCobertura {
	t.Helper()
	propiedad := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return DatosReservaTerminalOperacionDecisionCobertura{
		OrganizacionRef:        datosSolicitud.OrganizacionRef,
		ExpedienteRef:          datosSolicitud.ExpedienteRef,
		VersionExpediente:      datosSolicitud.VersionExpediente,
		ReservaRef:             propiedad.ReservaRef,
		ReciboRef:              propiedad.ReciboRef,
		ActuacionRef:           propiedad.ActuacionRef,
		AuditoriaRef:           propiedad.AuditoriaRef,
		EventoRef:              propiedad.EventoRef,
		CorrelacionVECRef:      propiedad.CorrelacionVECRef,
		DecisionVECRef:         propiedad.DecisionVECRef,
		AmbitoIdempotenciaHMAC: propiedad.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:    propiedad.HuellaSemanticaHMAC,
		RevisionCercado:        propiedad.RevisionCercado,
		ObservadaEnDB:          propiedad.ObservadaEnDB,
	}
}

func reservaTerminalOperacionDecisionCoberturaPrueba(
	t *testing.T,
	consulta SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	solicitud SolicitudReservarOperacionDecisionCobertura,
) ReservaTerminalOperacionDecisionCobertura {
	t.Helper()
	reserva, err := RehidratarReservaTerminalOperacionDecisionCobertura(
		consulta,
		datosReservaTerminalOperacionDecisionCoberturaPrueba(t, solicitud),
	)
	if err != nil {
		t.Fatal(err)
	}
	return reserva
}

func reciboOperacionDecisionCoberturaPrueba(
	t *testing.T,
	solicitud SolicitudReservarOperacionDecisionCobertura,
) ReciboOperacionDecisionCobertura {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	huella := strings.Repeat("9", 64)
	return ReciboOperacionDecisionCobertura{
		ReciboRef:               "recibo_decision_cobertura_01",
		ReservaRef:              "reserva_decision_cobertura_01",
		AuditoriaRef:            "auditoria_decision_cobertura_01",
		CorrelacionVECRef:       "correlacion_vec_decision_cobertura_01",
		DecisionVECRef:          "decision_vec_autorizacion_01",
		DecisionVECHuellaSHA256: strings.Repeat("f", 64),
		CodigoProbatorioVEC:     "concedida",
		ConcedidaVEC:            true,
		RevisionCercado:         1,
		AmbitoIdempotenciaHMAC:  datos.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     datos.HuellaSemanticaHMAC,
		ConfirmadaEn: instanteOperacionDecisionCoberturaPrueba.Add(
			time.Second,
		),
		Aplicada: &ResultadoAplicadoOperacionDecisionCobertura{
			DecisionCoberturaRef:    "decision-cobertura:sha256:" + huella,
			DecisionCoberturaHuella: huella,
			VersionResultante:       datos.VersionExpediente + 1,
			EventoRef:               "evento_decision_cobertura_01",
			ActuacionRef:            "actuacion_decision_cobertura_01",
		},
	}
}

func reciboDenegadoVECOperacionDecisionCoberturaPrueba(
	t *testing.T,
	solicitud SolicitudReservarOperacionDecisionCobertura,
) ReciboOperacionDecisionCobertura {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return ReciboOperacionDecisionCobertura{
		ReciboRef:               "recibo_decision_cobertura_01",
		ReservaRef:              "reserva_decision_cobertura_01",
		AuditoriaRef:            "auditoria_decision_cobertura_01",
		CorrelacionVECRef:       "correlacion_vec_decision_cobertura_01",
		DecisionVECRef:          "decision_vec_autorizacion_01",
		DecisionVECHuellaSHA256: strings.Repeat("6", 64),
		CodigoProbatorioVEC:     "accion_no_concedida",
		ConcedidaVEC:            false,
		RevisionCercado:         1,
		AmbitoIdempotenciaHMAC:  datos.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     datos.HuellaSemanticaHMAC,
		ConfirmadaEn: instanteOperacionDecisionCoberturaPrueba.Add(
			time.Second,
		),
		DenegadaVEC: &ResultadoDenegadoVECOperacionDecisionCobertura{},
	}
}
