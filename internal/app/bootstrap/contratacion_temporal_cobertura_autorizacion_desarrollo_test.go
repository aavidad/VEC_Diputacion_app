package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autoridadAsignacionesContratacionTemporalDesarrolloPrueba struct {
	preparadas int
	publicadas int
}

type registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba struct {
	concesiones  int
	denegaciones int
	huella       string
}

func (r *registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	_ context.Context,
	orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	huella, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(datos.Decision)
	if err != nil {
		return time.Time{}, err
	}
	emitidaEn, _, err := datos.Decision.VentanaValidez()
	if err != nil {
		return time.Time{}, err
	}
	r.concesiones++
	r.huella = huella
	return emitidaEn, nil
}

func (r *registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba) RegistrarDenegacionAutorizacionLigadaV3(
	_ context.Context,
	orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if _, err := orden.Datos(); err != nil {
		return err
	}
	r.denegaciones++
	return nil
}

func (a *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) PrepararInstantanea(
	_ context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	a.preparadas++
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(instantanea), nil
}

func (a *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) PublicarInstantanea(
	context.Context,
	dominiovec.InstantaneaAutorizacion,
) error {
	a.publicadas++
	return nil
}

func TestAutorizacionCoberturaDesarrolloSeparaRutasYAmbitos(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	alta, err := soporte.ObtenerInstantaneaAutorizacion(
		contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaAltaSolicitudes),
		vinculo.PrincipalID,
		vinculo.PerfilActivoRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	propuestaCtx := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaPropuestaCobertura,
	)
	coberturaVEC, err := soporte.ObtenerInstantaneaAutorizacion(
		propuestaCtx,
		vinculo.PrincipalID,
		vinculo.PerfilActivoRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(alta.AsignacionPerfil.Ambitos) != 3 ||
		len(coberturaVEC.AsignacionPerfil.Ambitos) != 2 ||
		len(coberturaVEC.VersionRol.Concesiones) != 4 {
		t.Fatalf(
			"instantaneas mezcladas: alta=%+v cobertura=%+v",
			alta.AsignacionPerfil.Ambitos,
			coberturaVEC.AsignacionPerfil.Ambitos,
		)
	}
	tiposPorAccion := make(map[string]string, len(coberturaVEC.VersionRol.Concesiones))
	for _, concesion := range coberturaVEC.VersionRol.Concesiones {
		tiposPorAccion[concesion.Accion] = concesion.TipoRecurso
	}
	for accion, tipoEsperado := range map[string]string{
		accionPropuestaCoberturaDesarrollo:                ports.TipoRecursoExpediente,
		string(domain.AccionDecidirCoberturaGobernada):    "decision_cobertura_gobernada",
		string(domain.AccionRectificarCoberturaGobernada): "decision_cobertura_gobernada",
		string(ports.AccionConsultarResultadoCobertura):   ports.TipoRecursoExpediente,
	} {
		if tiposPorAccion[accion] != tipoEsperado {
			t.Fatalf("tipo de recurso para %s=%q, esperado %q", accion, tiposPorAccion[accion], tipoEsperado)
		}
	}
	claves := map[string]string{}
	for _, ambito := range coberturaVEC.AsignacionPerfil.Ambitos {
		claves[ambito.Clave] = ambito.Valores[0]
	}
	if len(claves) != 2 ||
		claves["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		claves["unidad_ejecutora_ref"] != unidadCoberturaContratacionTemporalDesarrollo {
		t.Fatalf("ambitos de cobertura no exactos: %+v", claves)
	}
	if _, err := soporte.ResolverContextoCanalCobertura(propuestaCtx); err != nil {
		t.Fatalf("contexto de propuesta denegado: %v", err)
	}
	if _, err := soporte.ResolverContextoCanalCobertura(
		contextoRutaCoberturaDesarrolloPrueba(
			soporte,
			principal,
			httpinterno.RutaResultadoCobertura,
		),
	); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatalf("resultado obtuvo autoridad de efecto: %v", err)
	}
}

func TestPreparacionDecisionCoberturaPublicaLaAsignacionQueReferencia(t *testing.T) {
	soporte, consultas, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	delegado, ok := consultas.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)
	if !ok {
		t.Fatal("el autorizador de cobertura no expone la preparacion compuesta")
	}
	autorizador := &autorizadorAnalisisContratacionTemporalDesarrollo{
		delegado: delegado,
		soporte:  soporte,
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaDecisionCobertura,
	)
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: soporte.contexto.Vinculo,
			ReferenciaMotivo:          soporte.motivoDecisionCobertura,
			Accion:                    string(domain.AccionDecidirCoberturaGobernada),
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: "reserva:ct:desarrollo:cobertura:0001",
				ModuloID:   ports.ModuloContratacion,
				Tipo:       tipoRecursoDecisionCoberturaDesarrollo,
				Ambitos: map[string]string{
					"organizacion_ref":     organizacionAltaContratacionTemporalDesarrollo,
					"unidad_ejecutora_ref": unidadCoberturaContratacionTemporalDesarrollo,
				},
				Atributos: map[string]string{},
			},
			Finalidad:   finalidadDecisionCoberturaDesarrollo,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, candidata, err := autorizador.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctx,
		solicitud,
		soporte.contexto.Resultado,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	)
	if _, errResumen := candidata.Resumen(); err != nil || errResumen != nil {
		t.Fatalf("candidata de cobertura no preparada: %v", err)
	}
	autoridad, ok := soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if !ok {
		t.Fatal("autoridad de asignaciones de prueba inesperada")
	}
	if autoridad.preparadas != 1 || autoridad.publicadas != 1 {
		t.Fatalf("asignacion de cobertura no publicada antes de la candidata: %+v", autoridad)
	}
}

func TestAutorizacionAnalisisDesarrolloLigaRutaAccionRecursoFinalidadYUnidad(
	t *testing.T,
) {
	soporte, autorizadorCobertura, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	delegado, valido := autorizadorCobertura.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)
	if !valido {
		t.Fatal("el autorizador V3 no conserva sus dos contratos")
	}
	autorizador := &autorizadorAnalisisContratacionTemporalDesarrollo{delegado: delegado}
	for _, caso := range []struct {
		ruta   string
		accion string
		motivo dominiovec.ReferenciaEntradaCatalogo
	}{
		{
			httpinterno.RutaRegistroAnalisisRRHH,
			ports.AccionRegistrarAnalisis,
			soporte.motivoRegistroAnalisis,
		},
	} {
		t.Run(caso.accion, func(t *testing.T) {
			ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, caso.ruta)
			canal, err := soporte.ResolverContextoCanalAnalisisRRHH(ctx)
			if err != nil || canal.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
				t.Fatalf("contexto de analisis no resuelto: %+v %v", canal, err)
			}
			correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
				ctx,
				seguridadvec.GeneradorReferenciasCriptograficas{},
			)
			if err != nil {
				t.Fatal(err)
			}
			solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
				dominiovec.DatosSolicitudAutorizacionLigadaV3{
					VinculoAutenticacionActor: soporte.contexto.Vinculo,
					ReferenciaMotivo:          caso.motivo,
					Accion:                    caso.accion,
					Recurso: dominiovec.RecursoAutorizable{
						Referencia: "expediente:analisis:desarrollo:001",
						ModuloID:   ports.ModuloContratacion,
						Tipo:       ports.TipoRecursoAnalisis,
						Ambitos: map[string]string{
							"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
							"expediente_ref":   "expediente:analisis:desarrollo:001",
							"fase_previa":      "solicitud",
							"estado_previo":    "en_curso",
						},
						Atributos: map[string]string{
							ports.AtributoUnidadPoliticaRef: unidadCoberturaContratacionTemporalDesarrollo,
						},
					},
					Finalidad:   finalidadAnalisisContratacionTemporalDesarrollo,
					Correlacion: correlacion,
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			decision, _, err := autorizador.ExigirSolicitudLigadaV3(
				ctx,
				solicitud,
				soporte.contexto.Resultado,
			)
			concedida, _, errResultado := decision.Resultado()
			if err != nil || errResultado != nil || !concedida {
				t.Fatalf("autorizacion de analisis no concedida: %v %v", err, errResultado)
			}
		})
	}
	registro, valido := soporte.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba)
	if !valido || registro.concesiones != 1 || registro.huella == "" {
		t.Fatalf("registro durable de analisis no usado: %+v", registro)
	}
	soporte.mu.Lock()
	totalConcesionesEfimeras := len(soporte.concesiones)
	soporte.mu.Unlock()
	if totalConcesionesEfimeras != 0 {
		t.Fatalf("analisis duplico la concesion durable en memoria: %d", totalConcesionesEfimeras)
	}
	ctxAlta := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaAltaSolicitudes,
	)
	correlacionAlta, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctxAlta,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudAlta, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: soporte.contexto.Vinculo,
			ReferenciaMotivo:          soporte.motivo,
			Accion:                    ports.AccionCrearSolicitud,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: "efecto:alta:posterior-analisis",
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoExpediente,
				Ambitos: map[string]string{
					"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
					"centro_ref":       centroAltaContratacionTemporalDesarrollo,
					"categoria_ref":    categoriaAltaContratacionTemporalDesarrollo,
				},
			},
			Finalidad:   ports.FinalidadCrearSolicitud,
			Correlacion: correlacionAlta,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	decisionAlta, _, err := autorizador.ExigirSolicitudLigadaV3(
		ctxAlta,
		solicitudAlta,
		soporte.contexto.Resultado,
	)
	concedidaAlta, _, errResultadoAlta := decisionAlta.Resultado()
	if err != nil || errResultadoAlta != nil || !concedidaAlta {
		t.Fatalf("alta posterior a analisis no concedida: %v %v", err, errResultadoAlta)
	}
	autoridad, valida := soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if !valida || autoridad.preparadas != 2 || autoridad.publicadas != 2 {
		t.Fatalf("asignaciones de analisis y alta no publicadas: %+v", autoridad)
	}
	soporte.mu.Lock()
	totalConcesionesEfimeras = len(soporte.concesiones)
	soporte.mu.Unlock()
	if totalConcesionesEfimeras != 1 {
		t.Fatalf("alta posterior no registro su concesion efimera: %d", totalConcesionesEfimeras)
	}
	instantaneaSinConcesionAnalisis := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		soporte.instantaneaAnalisis,
	)
	instantaneaSinConcesionAnalisis.VersionRol.Concesiones[0].Accion =
		ports.AccionRectificarAnalisis
	if err := instantaneaSinConcesionAnalisis.Validar(); err != nil {
		t.Fatal(err)
	}
	soporte.mu.Lock()
	soporte.instantaneaAnalisis = instantaneaSinConcesionAnalisis
	soporte.mu.Unlock()
	ctxDenegacion := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaRegistroAnalisisRRHH,
	)
	correlacionDenegacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctxDenegacion,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudDenegada, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: soporte.contexto.Vinculo,
			ReferenciaMotivo:          soporte.motivoRegistroAnalisis,
			Accion:                    ports.AccionRegistrarAnalisis,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: "expediente:analisis:denegado",
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoAnalisis,
				Ambitos: map[string]string{
					"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
					"expediente_ref":   "expediente:analisis:denegado",
					"fase_previa":      "solicitud",
					"estado_previo":    "en_curso",
				},
				Atributos: map[string]string{
					ports.AtributoUnidadPoliticaRef: unidadCoberturaContratacionTemporalDesarrollo,
				},
			},
			Finalidad:   finalidadAnalisisContratacionTemporalDesarrollo,
			Correlacion: correlacionDenegacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _, errDenegacion := autorizador.ExigirSolicitudLigadaV3(
		ctxDenegacion,
		solicitudDenegada,
		soporte.contexto.Resultado,
	)
	if !errors.Is(errDenegacion, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("analisis sin concesion no fue denegado: %v", errDenegacion)
	}
	if registro.denegaciones != 1 {
		t.Fatalf(
			"denegacion de analisis no registrada de forma durable: %+v; error=%v; autoridad=%+v",
			registro,
			errDenegacion,
			autoridad,
		)
	}
	if autoridad.preparadas != 3 || autoridad.publicadas != 3 {
		t.Fatalf("asignacion de la denegacion no publicada: %+v", autoridad)
	}

	if _, err := soporte.ResolverContextoCanalAnalisisRRHH(
		contextoRutaCoberturaDesarrolloPrueba(
			soporte,
			principal,
			httpinterno.RutaRectificacionAnalisisRRHH,
		),
	); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatalf("rectificacion obtuvo contexto de analisis: %v", err)
	}

	if _, err := soporte.ResolverContextoCanalAnalisisRRHH(
		contextoRutaCoberturaDesarrolloPrueba(
			soporte,
			principal,
			httpinterno.RutaAltaSolicitudes,
		),
	); !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatalf("alta obtuvo contexto de analisis: %v", err)
	}
}

func TestAutorizadorConsultasCoberturaUsaServicioV3Real(t *testing.T) {
	soporte, autorizador, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctxPropuesta := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaPropuestaCobertura,
	)
	canal, err := soporte.ResolverContextoCanalCobertura(ctxPropuesta)
	if err != nil {
		t.Fatal(err)
	}
	solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: canal.AutenticacionRef,
		SesionRef:        canal.SesionRef,
		PerfilRef:        canal.PerfilRef,
	}
	contexto, err := soporte.ResolverContextoAutorizacionAltaV3(
		ctxPropuesta,
		solicitudContexto,
	)
	if err != nil {
		t.Fatal(err)
	}
	analisis, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		canal.OrganizacionRef,
		"expediente_temporal_desarrollo_0001",
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := autorizador.AutorizarPresentacionPropuestaCobertura(
		ctxPropuesta,
		solicitudContexto,
		contexto,
		analisis,
		soporte.reloj.Ahora(),
	); err != nil {
		t.Fatalf("propuesta no autorizada por V3: %v", err)
	}

	ctxResultado := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaResultadoCobertura,
	)
	contextoRecuperacion, err :=
		soporte.ResolverContextoRecuperacionResultadoCobertura(ctxResultado)
	if err != nil {
		t.Fatal(err)
	}
	solicitudLectura, err := ports.NuevaSolicitudLecturaResultadoCobertura(
		contextoRecuperacion,
		"expediente_temporal_desarrollo_0001",
		soporte.reloj.Ahora(),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := autorizador.AutorizarLecturaResultadoCobertura(
		ctxResultado,
		solicitudLectura,
	)
	if err != nil || resultado != ports.AutorizacionLecturaResultadoCoberturaConcedida {
		t.Fatalf("lectura no autorizada por V3: %v %v", resultado, err)
	}
	soporte.mu.Lock()
	totalConcesiones := len(soporte.concesiones)
	soporte.mu.Unlock()
	if totalConcesiones != 2 {
		t.Fatalf("concesiones confirmadas = %d, se esperaban 2", totalConcesiones)
	}
}

func TestAutorizacionCoberturaDesarrolloFallaCerrado(t *testing.T) {
	soporte, autorizador, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	if _, err := nuevoAutorizadorConsultasCoberturaDesarrollo(
		soporte,
		nil,
		seguridadvec.GeneradorReferenciasCriptograficas{},
	); !errors.Is(err, errAutorizacionCoberturaDesarrolloNoDisponible) {
		t.Fatalf("autorizador nulo aceptado: %v", err)
	}
	ctxAlta := contextoRutaCoberturaDesarrolloPrueba(
		soporte,
		principal,
		httpinterno.RutaAltaSolicitudes,
	)
	vinculo, _ := soporte.contexto.Vinculo.Datos()
	solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
	}
	analisis, _ := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		organizacionAltaContratacionTemporalDesarrollo,
		"expediente_temporal_desarrollo_0001",
		2,
	)
	if err := autorizador.AutorizarPresentacionPropuestaCobertura(
		ctxAlta,
		solicitudContexto,
		soporte.contexto,
		analisis,
		soporte.reloj.Ahora(),
	); !errors.Is(err, application.ErrPresentacionPropuestaCoberturaDenegada) {
		t.Fatalf("propuesta autorizada bajo ruta de alta: %v", err)
	}
	ctxCancelado, cancelar := context.WithCancel(ctxAlta)
	cancelar()
	if err := autorizador.AutorizarPresentacionPropuestaCobertura(
		ctxCancelado,
		solicitudContexto,
		soporte.contexto,
		analisis,
		soporte.reloj.Ahora(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no propagada: %v", err)
	}
}

func escenarioAutorizacionCoberturaDesarrolloPrueba(
	t *testing.T,
) (
	*soporteAltaContratacionTemporalDesarrollo,
	*autorizadorConsultasCoberturaDesarrollo,
	dominiovec.Principal,
) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	principal := dominiovec.Principal{
		ID:            "certificado_rrhh_autorizacion_cobertura",
		Roles:         []string{"tecnico_rrhh"},
		AuthMethod:    dominiovec.AuthMethodCertificate,
		AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad":          AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": strings.Repeat("d", 64),
		},
	}
	contexto, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	alta, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		vinculo.PrincipalID,
		vinculo.PerfilActivoRef,
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	coberturaVEC, err :=
		nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
			vinculo.PrincipalID,
			vinculo.PerfilActivoRef,
			ahora,
		)
	if err != nil {
		t.Fatal(err)
	}
	analisisVEC, err :=
		nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(
			vinculo.PrincipalID,
			vinculo.PerfilActivoRef,
			ahora,
		)
	if err != nil {
		t.Fatal(err)
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello:             &selloConsultasContratacionTemporalDesarrollo{},
		principalID:       principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto,
		motivo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_autorizacion",
			CatalogoVersion:      1,
			CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos"),
			EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
				"motivo_",
				"crear-solicitud",
			),
		},
		instantanea:                  alta,
		instantaneaAnalisis:          analisisVEC,
		motivoRegistroAnalisis:       referenciaMotivoAutorizacionAnalisisDesarrollo("registro"),
		instantaneaCobertura:         coberturaVEC,
		motivoPropuestaCobertura:     referenciaMotivoAutorizacionCoberturaDesarrollo("propuesta"),
		motivoDecisionCobertura:      referenciaMotivoAutorizacionCoberturaDesarrollo("decision"),
		motivoRectificacionCobertura: referenciaMotivoAutorizacionCoberturaDesarrollo("rectificacion"),
		motivoResultadoCobertura:     referenciaMotivoAutorizacionCoberturaDesarrollo("resultado"),
		reloj:                        relojContratacionTemporalDesarrollo{},
		concesiones:                  make(map[string]struct{}),
		autoridadAsignaciones:        &autoridadAsignacionesContratacionTemporalDesarrolloPrueba{},
		registroDecisionesAnalisis:   &registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba{},
		instantaneasPorSolicitud:     make(map[string]dominiovec.InstantaneaAutorizacion),
	}
	generador := seguridadvec.GeneradorReferenciasCriptograficas{}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		soporte,
		soporte,
		soporte,
		soporte,
		soporte.reloj,
		generador,
		aplicacionvec.ConfiguracionServicioAutorizacion{
			VigenciaDecision: 90 * time.Second,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	autorizador, err := nuevoAutorizadorConsultasCoberturaDesarrollo(
		soporte,
		servicio,
		generador,
	)
	if err != nil {
		t.Fatal(err)
	}
	return soporte, autorizador, principal
}

func contextoRutaCoberturaDesarrolloPrueba(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	principal dominiovec.Principal,
	ruta string,
) context.Context {
	return context.WithValue(
		context.Background(),
		claveCapacidadConsultasContratacionTemporalDesarrollo{},
		capacidadConsultaContratacionTemporalDesarrollo{
			sello: soporte.sello, ruta: ruta, principal: principal,
		},
	)
}
