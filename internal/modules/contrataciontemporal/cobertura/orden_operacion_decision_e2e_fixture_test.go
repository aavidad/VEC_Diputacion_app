package cobertura_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	dominioct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type relojGobiernoOrdenC3 struct{ ahora time.Time }

func (r relojGobiernoOrdenC3) AhoraGobiernoOperacionCobertura(
	context.Context,
) (time.Time, error) {
	return r.ahora, nil
}

type resolutorGobiernoOrdenC3 struct {
	publicacion cobertura.PublicacionGobiernoOperacionCobertura
}

func (r resolutorGobiernoOrdenC3) ResolverGobiernoOperacionCobertura(
	context.Context,
	cobertura.SolicitudResolucionGobiernoOperacionCobertura,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	return r.publicacion, nil
}

type lectorAnalisisOrdenC3 struct{ expediente dominioct.Expediente }

func (l lectorAnalisisOrdenC3) LeerExpedienteAnalisisDurableO3(
	context.Context,
	cobertura.SolicitudInstantaneaAnalisisDurableO3,
) (dominioct.Expediente, error) {
	return l.expediente.Clonar(), nil
}

type revalidadorActorOrdenC3 struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorActorOrdenC3) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorActorOrdenC3 struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (r resolutorActorOrdenC3) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojActorOrdenC3 struct{ ahora time.Time }

func (r relojActorOrdenC3) Ahora() time.Time { return r.ahora }

type generadorCorrelacionOrdenC3 struct{ valor string }

func (g generadorCorrelacionOrdenC3) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type escenarioOrdenC3 struct {
	base               time.Time
	preparacion        cobertura.PreparacionOrdenOperacionDecisionCobertura
	candidataConcedida puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
	candidataDenegada  puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
	candidataAjena     puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
	limite             time.Time
}

func nuevoEscenarioOrdenC3(t *testing.T) escenarioOrdenC3 {
	t.Helper()
	base := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	expediente := expedienteConAnalisisOrdenC3(t, base)
	solicitudO3, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	instantaneaO3, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		context.Background(),
		lectorAnalisisOrdenC3{expediente: expediente},
		solicitudO3,
	)
	if err != nil {
		t.Fatal(err)
	}
	agregado, analisisRef, analisisHuella, err := instantaneaO3.DesplegarPara(
		solicitudO3,
	)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, politica := catalogoYPoliticaOrdenC3(t, base)
	previsualizacion := prepararC1OrdenC3(
		t,
		agregado,
		analisisRef,
		analisisHuella,
		catalogo,
		politica,
		base.Add(-500*time.Microsecond),
	)
	datosPrevios, err := previsualizacion.DatosCrearPropuestaEn(
		base.Add(-250 * time.Microsecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	propuestaPrevia, err := dominioct.CrearPropuestaDecisionCobertura(
		datosPrevios,
	)
	if err != nil {
		t.Fatal(err)
	}
	semantica, err := propuestaPrevia.IdentidadSemantica()
	if err != nil {
		t.Fatal(err)
	}
	contextoActor, solicitudActor := contextoActorOrdenC3(t, base)
	identidad, err := cobertura.NuevaIdentidadOperacionDecisionCobertura(
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		dominioct.DecisionCoberturaInicial,
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
		contextoActor,
		solicitudActor,
		base,
		dominioct.AccionDecidirCoberturaGobernada,
		"bolsa_vigente",
		semantica,
		dominioct.MotivoGobernadoDecisionCobertura{},
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudReserva, datosReserva := reservaPropietariaOrdenC3(
		t,
		identidad,
		agregado,
		analisisRef,
		analisisHuella,
		base,
	)
	preparacionReserva, err :=
		cobertura.NuevaPreparacionOperacionDecisionCoberturaPropietaria(
			solicitudReserva,
			datosReserva,
		)
	if err != nil {
		t.Fatal(err)
	}
	preparacionC1 := prepararC1OrdenC3(
		t,
		agregado,
		analisisRef,
		analisisHuella,
		catalogo,
		politica,
		base.Add(time.Millisecond),
	)
	datosPropuesta, err := preparacionC1.DatosCrearPropuestaEn(
		base.Add(1500 * time.Microsecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	propuesta, err := dominioct.CrearPropuestaDecisionCobertura(datosPropuesta)
	if err != nil {
		t.Fatal(err)
	}
	solicitudGobierno, gobierno, datosGobierno :=
		gobiernoOrdenC3(t, expediente, catalogo, politica, base)
	preparacion, err := cobertura.PrepararOrdenOperacionDecisionCobertura(
		context.Background(),
		relojGobiernoOrdenC3{ahora: base.Add(3 * time.Millisecond)},
		solicitudReserva,
		preparacionReserva,
		solicitudGobierno,
		gobierno,
		preparacionC1,
		propuesta,
		cobertura.ResolucionMotivoDecisionCobertura{},
	)
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := preparacion.RecursoAutorizableVEC()
	if err != nil {
		t.Fatal(err)
	}
	emitidaEn := base.Add(3500 * time.Microsecond)
	limite := base.Add(5 * time.Second)
	concedida := candidataOrdenC3(
		t,
		contextoActor,
		datosReserva,
		datosGobierno,
		recurso,
		emitidaEn,
		limite,
		true,
	)
	denegada := candidataOrdenC3(
		t,
		contextoActor,
		datosReserva,
		datosGobierno,
		recurso,
		emitidaEn,
		limite,
		false,
	)
	recursoAjeno := recurso
	recursoAjeno.Ambitos = clonarMapaOrdenC3(recurso.Ambitos)
	recursoAjeno.Atributos = clonarMapaOrdenC3(recurso.Atributos)
	recursoAjeno.Atributos["propuesta_semantica_huella_sha256"] =
		strings.Repeat("9", 64)
	ajena := candidataOrdenC3(
		t,
		contextoActor,
		datosReserva,
		datosGobierno,
		recursoAjeno,
		emitidaEn,
		limite,
		false,
	)
	return escenarioOrdenC3{
		base: base, preparacion: preparacion,
		candidataConcedida: concedida, candidataDenegada: denegada,
		candidataAjena: ajena, limite: limite,
	}
}

func clonarMapaOrdenC3(origen map[string]string) map[string]string {
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

func expedienteConAnalisisOrdenC3(
	t *testing.T,
	base time.Time,
) dominioct.Expediente {
	t.Helper()
	periodo := dominioct.PeriodoPrevisto{
		Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
	}
	expediente, err := dominioct.NuevoExpediente(dominioct.AltaExpediente{
		Referencia:      "expediente_temporal_orden_c3_01",
		OrganizacionRef: organizacionOrdenC3,
		NumeroVisible:   "2026/5487",
		Flujo: dominioct.ReferenciaFlujo{
			DefinicionRef: "flujo_contratacion_temporal",
			Version:       3,
			HuellaSHA256:  strings.Repeat("e", 64),
		},
		FaseInicial: "solicitud",
		Solicitud: dominioct.SolicitudCentro{
			CentroRef: "centro_social_01", ContactoRef: "contacto_opaco_01",
			CategoriaRef: "categoria_trabajador_social", GrupoSubgrupo: "A2",
			MotivoClave: "sustitucion_it",
			Detalle:     "Necesidad temporal de prueba.",
			Periodo:     periodo,
			RC:          dominioct.DeclaracionRC{},
		},
		Actuacion: dominioct.DatosActuacion{
			AccionClave: "solicitud.registrada",
			ActorRef:    "actor_rrhh_opaco_01",
			UnidadRef:   "unidad_rrhh_01",
			ReciboRef:   "recibo_alta_orden_c3_01",
			RealizadaEn: base.Add(-2 * time.Minute),
			FaseDestino: "solicitud", EstadoDestino: dominioct.EstadoEnCurso,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	entrada := dominioct.VinculoEntradaRC{
		Referencia:   "entrada_rc_orden_c3_01",
		HuellaSHA256: strings.Repeat("6", 64),
	}
	expediente, err = expediente.RegistrarAnalisis(
		expediente.Version,
		dominioct.AnalisisRRHH{
			ModalidadClave:    "modalidad_interinidad_publicada",
			CategoriaRef:      expediente.Solicitud.CategoriaRef,
			GrupoSubgrupo:     expediente.Solicitud.GrupoSubgrupo,
			CausaClave:        "causa_sustitucion_publicada",
			Periodo:           periodo,
			PorcentajeJornada: dominioct.JornadaCompletaDiezmilesimas,
			EntradaRCEsperada: entrada,
			ValidacionRC: dominioct.ValidacionRC{
				Resultado:           dominioct.RCNoRequerida,
				EntradaRef:          entrada.Referencia,
				HuellaEntradaSHA256: entrada.HuellaSHA256,
				FuenteRef:           "fuente_presupuestaria_orden_c3_01",
				ReciboRef:           "recibo_validacion_rc_orden_c3_01",
				ValidadaEn:          base.Add(-time.Minute - time.Second),
				Motivo:              "Resultado gobernado de prueba.",
			},
		},
		dominioct.DatosActuacion{
			AccionClave: dominioct.ClaveCatalogo(
				puertosct.AccionRegistrarAnalisis,
			),
			ActorRef:      "actor_rrhh_analisis_orden_c3_01",
			UnidadRef:     "unidad_rrhh_analisis_orden_c3_01",
			ReciboRef:     "recibo_confirmacion_analisis_orden_c3_01",
			RealizadaEn:   base.Add(-time.Minute),
			FaseDestino:   "analisis_rrhh",
			EstadoDestino: dominioct.EstadoEnCurso,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}

func reservaPropietariaOrdenC3(
	t *testing.T,
	identidad cobertura.DatosIdentidadOperacionDecisionCobertura,
	agregado dominioct.Expediente,
	analisisRef string,
	analisisHuella string,
	base time.Time,
) (
	cobertura.SolicitudReservarOperacionDecisionCobertura,
	cobertura.DatosReservaPropietariaOperacionDecisionCobertura,
) {
	t.Helper()
	ambito, err := puertosct.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal.cobertura-decision.ambito/v1:"+
			strings.Repeat("a", 64),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	semantica, err := puertosct.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal.cobertura-decision.semantica/v1:"+
			strings.Repeat("b", 64),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta, err :=
		cobertura.NuevaSolicitudConsultarOperacionDecisionCoberturaConfirmada(
			identidad,
			cobertura.SellosOperacionDecisionCobertura{
				AmbitosIdempotenciaHMAC: ambito,
				HuellasSemanticasHMAC:   semantica,
			},
		)
	if err != nil {
		t.Fatal(err)
	}
	token, err := cobertura.GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := cobertura.NuevaSolicitudReservarOperacionDecisionCobertura(
		consulta,
		token,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud, _ := solicitud.Datos()
	datos := cobertura.DatosReservaPropietariaOperacionDecisionCobertura{
		ReservaRef:        "reserva_decision_cobertura_orden_c3_01",
		ReciboRef:         "recibo_decision_cobertura_orden_c3_01",
		ActuacionRef:      "actuacion_decision_cobertura_orden_c3_01",
		AuditoriaRef:      "auditoria_decision_cobertura_orden_c3_01",
		EventoRef:         "evento_decision_cobertura_orden_c3_01",
		CorrelacionVECRef: "correlacion_11111111111111111111111111111111",
		DecisionVECRef:    "dec_11111111111111111111111111111111",
		AnalisisRef:       analisisRef, AnalisisHuellaSHA256: analisisHuella,
		TokenPropietarioSHA256:  datosSolicitud.TokenPropietarioSHA256,
		AmbitoIdempotenciaHMAC:  datosSolicitud.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     datosSolicitud.HuellaSemanticaHMAC,
		AgregadoAnterior:        ptrExpedienteOrdenC3(agregado),
		RevisionCercadoAnterior: 0, RevisionCercado: 1,
		ObservadaEnDB: base, PropiedadHasta: base.Add(5 * time.Second),
	}
	return solicitud, datos
}

func ptrExpedienteOrdenC3(expediente dominioct.Expediente) *dominioct.Expediente {
	clon := expediente.Clonar()
	return &clon
}

func gobiernoOrdenC3(
	t *testing.T,
	expediente dominioct.Expediente,
	catalogo dominioct.CatalogoViasCobertura,
	politica dominioct.PoliticaDecisionCobertura,
	base time.Time,
) (
	cobertura.SolicitudGobiernoOperacionCobertura,
	cobertura.GobiernoOperacionCobertura,
	cobertura.DatosGobiernoOperacionCobertura,
) {
	t.Helper()
	solicitud, err := cobertura.NuevaSolicitudGobiernoDecisionCobertura(
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	finalidadClave, finalidadRef := politica.Finalidad()
	actuacion := cobertura.PublicacionPoliticaActuacionCobertura{
		Referencia: "politica_actuacion_cobertura_orden_c3",
		Version:    1, Canon: cobertura.CanonHuellaPoliticaActuacionCoberturaV1(),
		OrganizacionRef: expediente.OrganizacionRef,
		Accion:          dominioct.AccionDecidirCoberturaGobernada,
		Catalogo:        catalogo.Identidad(), Politica: politica.Identidad(),
		FinalidadContratacionClave: finalidadClave,
		FinalidadContratacionRef:   finalidadRef,
		FinalidadAutorizacionVEC:   "tramitar_cobertura_temporal",
		UnidadEjecutoraRef:         "unidad_rrhh_cobertura_orden_c3",
		FaseDestino:                "asignacion_unidad",
		EstadoDestino:              dominioct.EstadoEnCurso,
		MotivoAutorizacionDecidir:  motivoAutorizacionOrdenC3(),
		MotivoAutorizacionRectificar: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_autorizacion_cobertura",
			CatalogoVersion:      1,
			CatalogoHuellaSHA256: strings.Repeat("c", 64),
			EntradaClave:         "motivo_22222222222222222222222222222222",
		},
		PublicadaEn: base.Add(-time.Hour),
		Vigencia: dominioct.VigenciaCatalogoCobertura{
			Desde: base.Add(-time.Hour), Hasta: base.Add(time.Hour),
		},
	}
	actuacion.HuellaSHA256, err =
		cobertura.CalcularHuellaSHA256PoliticaActuacionCobertura(actuacion)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := cobertura.PublicacionGobiernoOperacionCobertura{
		OrganizacionRef:   expediente.OrganizacionRef,
		ExpedienteRef:     expediente.Referencia,
		VersionExpediente: expediente.Version,
		Catalogo:          catalogo, Politica: politica,
		PoliticaActuacion: actuacion,
	}
	gobierno, err := cobertura.ObtenerGobiernoOperacionCobertura(
		context.Background(),
		relojGobiernoOrdenC3{ahora: base.Add(2 * time.Millisecond)},
		resolutorGobiernoOrdenC3{publicacion: publicacion},
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := gobierno.DesplegarPara(
		context.Background(),
		relojGobiernoOrdenC3{ahora: base.Add(2500 * time.Microsecond)},
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, gobierno, datos
}

func motivoAutorizacionOrdenC3() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_cobertura",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("c", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
}

func contextoActorOrdenC3(
	t *testing.T,
	base time.Time,
) (
	puertosct.ContextoAutorizacionAltaV3,
	puertosct.SolicitudResolverContextoAutorizacionAltaV3,
) {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: base.Add(-time.Hour), VigenteHasta: base.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta,
		instantanea,
		base.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema: dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef, Version: instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef, Version: instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef, Version: instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, _ := manifiesto.RepresentacionCanonicaV1()
	manifiestoHuella, _ :=
		dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
			manifiestoCanon,
		)
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn",
		Contexto:            actor, RepresentacionCanonica: canon, HuellaSHA256: huella,
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
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    base.Add(-10 * time.Minute),
		SesionEmitidaEn:              base.Add(-9 * time.Minute),
		SesionValidaHasta:            base.Add(20 * time.Minute),
		SesionRevalidadaEn:           base.Add(-3 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorActorOrdenC3{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorActorOrdenC3{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		relojActorOrdenC3{ahora: base},
	)
	if err != nil {
		t.Fatal(err)
	}
	return puertosct.ContextoAutorizacionAltaV3{
			Vinculo: vinculo, Resultado: resultado,
		},
		puertosct.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
			PerfilRef:        instantanea.PerfilActivoRef,
		}
}

func candidataOrdenC3(
	t *testing.T,
	contexto puertosct.ContextoAutorizacionAltaV3,
	reserva cobertura.DatosReservaPropietariaOperacionDecisionCobertura,
	gobierno cobertura.DatosGobiernoOperacionCobertura,
	recurso dominiovec.RecursoAutorizable,
	emitidaEn time.Time,
	validaHasta time.Time,
	conceder bool,
) puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3 {
	t.Helper()
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionOrdenC3{valor: reserva.CorrelacionVECRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          gobierno.MotivoAutorizacion,
			Accion:                    string(gobierno.Accion), Recurso: recurso,
			Finalidad:   string(gobierno.FinalidadVEC),
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, _ := contexto.Vinculo.Datos()
	accionConcedida := string(gobierno.Accion)
	if !conceder {
		accionConcedida = "accion_decision_cobertura_ajena"
	}
	version := dominiovec.VersionRol{
		RolID: "tecnico_rrhh_cobertura", Version: 1,
		Nombre: "Técnico de RRHH",
		Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion:   accionConcedida,
			ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
			Finalidades:    []string{string(gobierno.FinalidadVEC)},
			GarantiaMinima: dominiovec.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable_seguridad",
		PublicadaEn:  emitidaEn.Add(-24 * time.Hour),
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(recurso.Ambitos))
	for clave, valor := range recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{
			Clave: clave, Valores: []string{valor},
		})
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: dominiovec.AsignacionPerfil{
			AsignacionID: "asignacion_rrhh_cobertura_orden_c3",
			Version:      1, PerfilActivoRef: datosVinculo.PerfilActivoRef,
			PrincipalID:   datosVinculo.PrincipalID,
			VersionRolRef: version.Referencia(),
			Estado:        dominiovec.EstadoAsignacionPerfilActiva,
			Ambitos:       ambitos,
			VigenteDesde:  emitidaEn.Add(-time.Hour),
			VigenteHasta:  emitidaEn.Add(time.Hour),
			EmitidaPor:    "administrador_identidades",
			EmitidaEn:     emitidaEn.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor,
			ActualizadoEn:  version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		reserva.DecisionVECRef,
		emitidaEn,
		validaHasta,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(
		solicitud,
		evidencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	candidata, err :=
		puertosvec.NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
			solicitud,
			decision,
			gobierno.MotivoAutorizacion,
			contexto.Resultado,
		)
	if err != nil {
		t.Fatal(err)
	}
	return candidata
}
