package postgres

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type revalidadorActorPreparacionDecisionCoberturaPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorActorPreparacionDecisionCoberturaPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorActorPreparacionDecisionCoberturaPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (r resolutorActorPreparacionDecisionCoberturaPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojActorPreparacionDecisionCoberturaPrueba struct {
	ahora time.Time
}

func (r relojActorPreparacionDecisionCoberturaPrueba) Ahora() time.Time {
	return r.ahora
}

type fixturePreparacionDecisionCoberturaDurable struct {
	base              time.Time
	expediente        domain.Expediente
	contexto          puertosct.ContextoAutorizacionAltaV3
	solicitudContexto puertosct.SolicitudResolverContextoAutorizacionAltaV3
}

func nuevoFixturePreparacionDecisionCoberturaDurable(
	t *testing.T,
) fixturePreparacionDecisionCoberturaDurable {
	t.Helper()
	base := time.Now().UTC().Truncate(time.Microsecond)
	inicio := time.Date(
		base.Year(),
		base.Month(),
		base.Day()+1,
		0,
		0,
		0,
		0,
		time.UTC,
	)
	fin := inicio.Add(30 * 24 * time.Hour)
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente_ct_o404c_integracion_01",
		OrganizacionRef: "organizacion_dipgra",
		NumeroVisible:   "2026/O404C-01",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo_ct_o404c_integracion",
			Version:       3,
			HuellaSHA256:  strings.Repeat("e", 64),
		},
		FaseInicial: "solicitud",
		Solicitud: domain.SolicitudCentro{
			CentroRef: "centro_o404c_01", ContactoRef: "contacto_o404c_01",
			CategoriaRef: "categoria_o404c_01", GrupoSubgrupo: "C2",
			MotivoClave: "sustitucion_o404c",
			Detalle:     "Vector sintético O4-04C sin datos personales.",
			Periodo:     domain.PeriodoPrevisto{Inicio: inicio, Fin: fin},
			RC:          domain.DeclaracionRC{},
		},
		Actuacion: domain.DatosActuacion{
			AccionClave:   "solicitud.registrada",
			ActorRef:      "actor_o404c_01",
			UnidadRef:     "unidad_o404c_01",
			ReciboRef:     "recibo_alta_o404c_01",
			RealizadaEn:   base.Add(-2 * time.Minute),
			FaseDestino:   "solicitud",
			EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	entrada := domain.VinculoEntradaRC{
		Referencia:   "entrada_rc_o404c_01",
		HuellaSHA256: strings.Repeat("6", 64),
	}
	expediente, err = expediente.RegistrarAnalisis(
		expediente.Version,
		domain.AnalisisRRHH{
			ModalidadClave:    "modalidad_interinidad_o404c",
			CategoriaRef:      expediente.Solicitud.CategoriaRef,
			GrupoSubgrupo:     expediente.Solicitud.GrupoSubgrupo,
			CausaClave:        "causa_sustitucion_o404c",
			Periodo:           expediente.Solicitud.Periodo,
			PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
			EntradaRCEsperada: entrada,
			ValidacionRC: domain.ValidacionRC{
				Resultado:           domain.RCNoRequerida,
				EntradaRef:          entrada.Referencia,
				HuellaEntradaSHA256: entrada.HuellaSHA256,
				FuenteRef:           "fuente_presupuestaria_o404c_01",
				ReciboRef:           "recibo_validacion_rc_o404c_01",
				ValidadaEn:          base.Add(-90 * time.Second),
				Motivo:              "contratacion_temporal.rc.no_requerida",
			},
		},
		domain.DatosActuacion{
			AccionClave: domain.ClaveCatalogo(
				puertosct.AccionRegistrarAnalisis,
			),
			ActorRef:      "actor_analisis_o404c_01",
			UnidadRef:     "unidad_analisis_o404c_01",
			ReciboRef:     "recibo_analisis_o404c_01",
			RealizadaEn:   base.Add(-time.Minute),
			FaseDestino:   "analisis_rrhh",
			EstadoDestino: domain.EstadoEnCurso,
		},
	)
	if err != nil {
		t.Fatalf("registrar análisis O3 sintético: %v", err)
	}
	contexto, solicitudContexto :=
		contextoActorPreparacionDecisionCoberturaPrueba(t, base)
	return fixturePreparacionDecisionCoberturaDurable{
		base: base, expediente: expediente,
		contexto: contexto, solicitudContexto: solicitudContexto,
	}
}

func (f fixturePreparacionDecisionCoberturaDurable) identidad(
	t *testing.T,
	claveIdempotencia string,
	organizacionRef string,
) cobertura.DatosIdentidadOperacionDecisionCobertura {
	t.Helper()
	identidad, err := cobertura.NuevaIdentidadOperacionDecisionCobertura(
		claveIdempotencia,
		domain.DecisionCoberturaInicial,
		organizacionRef,
		f.expediente.Referencia,
		f.expediente.Version,
		f.contexto,
		f.solicitudContexto,
		f.base,
		domain.AccionDecidirCoberturaGobernada,
		"bolsa_vigente",
		domain.IdentidadSemanticaPropuestaDecisionCobertura{
			Referencia: "propuesta-cobertura-semantica:sha256:" +
				strings.Repeat("a", 64),
			HuellaSHA256: strings.Repeat("a", 64),
			Canon: domain.
				CanonHuellaSemanticaPropuestaDecisionCoberturaV1(),
		},
		domain.MotivoGobernadoDecisionCobertura{},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("crear identidad O4-04C: %v", err)
	}
	return identidad
}

type parHMACPreparacionDecisionCoberturaPrueba struct {
	generacion uint32
	ambito     string
	semantica  string
}

func parHMACDecisionCoberturaPrueba(
	generacion uint32,
	ambito string,
	semantica string,
) parHMACPreparacionDecisionCoberturaPrueba {
	return parHMACPreparacionDecisionCoberturaPrueba{
		generacion: generacion, ambito: ambito, semantica: semantica,
	}
}

func (p parHMACPreparacionDecisionCoberturaPrueba) ambitoHMAC() string {
	return "hmac-sha256:vec.contratacion-temporal." +
		"cobertura-decision.ambito/v" +
		strconv.FormatUint(uint64(p.generacion), 10) + ":" +
		strings.Repeat(p.ambito, 64)
}

func (p parHMACPreparacionDecisionCoberturaPrueba) semanticaHMAC() string {
	return "hmac-sha256:vec.contratacion-temporal." +
		"cobertura-decision.semantica/v" +
		strconv.FormatUint(uint64(p.generacion), 10) + ":" +
		strings.Repeat(p.semantica, 64)
}

func solicitudPreparacionDecisionCoberturaPrueba(
	t *testing.T,
	identidad cobertura.DatosIdentidadOperacionDecisionCobertura,
	activo parHMACPreparacionDecisionCoberturaPrueba,
	retenidos ...parHMACPreparacionDecisionCoberturaPrueba,
) (
	cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	cobertura.SolicitudReservarOperacionDecisionCobertura,
) {
	t.Helper()
	ambitosRetenidos := make([]string, 0, len(retenidos))
	semanticasRetenidas := make([]string, 0, len(retenidos))
	for _, retenido := range retenidos {
		ambitosRetenidos = append(ambitosRetenidos, retenido.ambitoHMAC())
		semanticasRetenidas = append(
			semanticasRetenidas,
			retenido.semanticaHMAC(),
		)
	}
	ambitos, err := puertosct.NuevaColeccionSellosHMAC(
		activo.ambitoHMAC(),
		ambitosRetenidos,
	)
	if err != nil {
		t.Fatalf("crear colección de ámbitos O4-04C: %v", err)
	}
	semanticas, err := puertosct.NuevaColeccionSellosHMAC(
		activo.semanticaHMAC(),
		semanticasRetenidas,
	)
	if err != nil {
		t.Fatalf("crear colección semántica O4-04C: %v", err)
	}
	consulta, err :=
		cobertura.NuevaSolicitudConsultarOperacionDecisionCoberturaConfirmada(
			identidad,
			cobertura.SellosOperacionDecisionCobertura{
				AmbitosIdempotenciaHMAC: ambitos,
				HuellasSemanticasHMAC:   semanticas,
			},
		)
	if err != nil {
		t.Fatalf("crear consulta O4-04C: %v", err)
	}
	token, err := cobertura.GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		t.Fatalf("generar token O4-04C: %v", err)
	}
	solicitud, err := cobertura.NuevaSolicitudReservarOperacionDecisionCobertura(
		consulta,
		token,
	)
	if err != nil {
		t.Fatalf("crear reserva O4-04C: %v", err)
	}
	return consulta, solicitud
}

func contextoActorPreparacionDecisionCoberturaPrueba(
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
		VigenteDesde: base.Add(-time.Hour),
		VigenteHasta: base.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta,
		instantanea,
		base.Add(-2*time.Minute),
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
			CuentaRef: cuenta.CuentaRef,
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
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    base.Add(-10 * time.Minute),
		SesionEmitidaEn:              base.Add(-9 * time.Minute),
		SesionValidaHasta:            base.Add(20 * time.Minute),
		SesionRevalidadaEn:           base.Add(-3 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorActorPreparacionDecisionCoberturaPrueba{
			resultado: autenticacion,
		},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorActorPreparacionDecisionCoberturaPrueba{
			resultado: resultado,
		},
		dominiovec.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		relojActorPreparacionDecisionCoberturaPrueba{ahora: base},
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
