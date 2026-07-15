package confianzadocumental

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

func TestAutoridadInternaEjecucionDocumentalV4LigaTernaExactaYEsOpaca(t *testing.T) {
	escenario := nuevoEscenarioAutoridadInternaEjecucionDocumentalV4(t)
	autoridad, err := emitirAutoridadEscenarioPDPPrueba(t, escenario)
	if err != nil {
		t.Fatalf("emitir autoridad interna: %v", err)
	}
	if err := autoridad.ValidarEn(escenario.emitidaEn); err != nil {
		t.Fatalf("autoridad exacta rechazada: %v", err)
	}

	aplicarEn := escenario.emitidaEn.Add(time.Microsecond)
	solicitud, err := autoridad.PrepararAplicacionExactaEn(
		escenario.decision.DecisionRef,
		escenario.expectativa.HuellaPlanSHA256,
		escenario.expectativa.EfectoRef,
		aplicarEn,
	)
	if err != nil {
		t.Fatalf("preparar aplicacion exacta: %v", err)
	}
	proyeccion, err := solicitud.ProyeccionParaTransaccion()
	if err != nil {
		t.Fatalf("proyectar solicitud: %v", err)
	}
	if proyeccion.Clave != (ports.ClaveAplicacionAutorizacionEjecucionDocumentalV4{
		DecisionRef:      escenario.decision.DecisionRef,
		HuellaPlanSHA256: escenario.expectativa.HuellaPlanSHA256,
		EfectoRef:        escenario.expectativa.EfectoRef,
	}) || !proyeccion.SolicitadaEn.Equal(aplicarEn) {
		t.Fatalf("terna o instante no ligados: %+v", proyeccion)
	}

	if _, err := json.Marshal(autoridad); !errors.Is(
		err,
		ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida,
	) {
		t.Fatalf("JSON de autoridad no fue bloqueado: %v", err)
	}
	texto := fmt.Sprintf("%v %+v %#v", autoridad, autoridad, autoridad)
	if strings.Contains(texto, escenario.decision.DecisionRef) ||
		strings.Contains(texto, escenario.expectativa.HuellaPlanSHA256) ||
		strings.Contains(texto, escenario.expectativa.PrincipalID) {
		t.Fatalf("el formato expuso datos de autoridad: %s", texto)
	}

	var cero AutoridadInternaEjecucionDocumentalV4
	if err := cero.ValidarEn(aplicarEn); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("el valor cero no se denego: %v", err)
	}
	if _, err := cero.PrepararAplicacionExactaEn("decision:x", huellaInternaPrueba('a'), "efecto:x", aplicarEn); !errors.Is(
		err,
		domain.ErrAutorizacionDenegada,
	) {
		t.Fatalf("el valor cero preparo una aplicacion: %v", err)
	}
}

func TestAutoridadInternaEjecucionDocumentalV4DeniegaTodaDiscrepancia(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*escenarioAutoridadInternaEjecucionDocumentalV4)
	}{
		{
			"sin evidencia",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.evidencia = ports.EvidenciaUsoDecisionAutorizacion{}
			},
		},
		{
			"asignacion esperada distinta",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.AsignacionRef = "asignacion:otra:v1"
			},
		},
		{
			"vinculo esperado ausente",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.VinculoAutenticacionActor = domain.VinculoAutenticacionActorV1{}
			},
		},
		{
			"control de rol distinto",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.ControlVigenciaVersionRolRevision++
			},
		},
		{
			"recurso distinto",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.Recurso.Ambitos["organizacion"] = "otra"
			},
		},
		{
			"comodin",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.Finalidad = "tramitar*"
			},
		},
		{
			"obligacion no acreditada",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.expectativa.ObligacionesEsperadas = []string{"doble_control"}
			},
		},
		{
			"caducada",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.emitidaEn = e.decision.ValidaHasta
			},
		},
		{
			"instante no canonico",
			func(e *escenarioAutoridadInternaEjecucionDocumentalV4) {
				e.emitidaEn = e.emitidaEn.Add(time.Nanosecond)
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAutoridadInternaEjecucionDocumentalV4(t)
			caso.mutar(&escenario)
			autoridad, err := emitirAutoridadEscenarioPDPPrueba(t, escenario)
			if err == nil ||
				!errors.Is(err, ErrAutoridadInternaEjecucionDocumentalV4Invalida) ||
				!errors.Is(err, domain.ErrAutorizacionDenegada) ||
				autoridad.marca != "" {
				t.Fatalf("se esperaba denegacion cerrada; autoridad=%+v error=%v", autoridad, err)
			}
		})
	}
}

func TestAutoridadInternaEjecucionDocumentalV4DeniegaOtraTernaTiempoYManipulacion(t *testing.T) {
	escenario := nuevoEscenarioAutoridadInternaEjecucionDocumentalV4(t)
	autoridad, err := emitirAutoridadEscenarioPDPPrueba(t, escenario)
	if err != nil {
		t.Fatal(err)
	}
	instante := escenario.emitidaEn.Add(time.Microsecond)
	casos := []struct {
		nombre, decisionRef, plan, efecto string
		instante                          time.Time
	}{
		{"otra decision", "decision:otra", escenario.expectativa.HuellaPlanSHA256, escenario.expectativa.EfectoRef, instante},
		{"otro plan", escenario.decision.DecisionRef, huellaInternaPrueba('f'), escenario.expectativa.EfectoRef, instante},
		{"otro efecto", escenario.decision.DecisionRef, escenario.expectativa.HuellaPlanSHA256, "efecto:otro", instante},
		{"retroceso", escenario.decision.DecisionRef, escenario.expectativa.HuellaPlanSHA256, escenario.expectativa.EfectoRef, escenario.emitidaEn.Add(-time.Microsecond)},
		{"caducidad", escenario.decision.DecisionRef, escenario.expectativa.HuellaPlanSHA256, escenario.expectativa.EfectoRef, escenario.decision.ValidaHasta},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := autoridad.PrepararAplicacionExactaEn(
				caso.decisionRef,
				caso.plan,
				caso.efecto,
				caso.instante,
			); !errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("discrepancia no denegada: %v", err)
			}
		})
	}

	manipulada := autoridad
	manipulada.huellaAutoridadSHA256 = huellaInternaPrueba('0')
	if _, err := manipulada.PrepararAplicacionExactaEn(
		escenario.decision.DecisionRef,
		escenario.expectativa.HuellaPlanSHA256,
		escenario.expectativa.EfectoRef,
		instante,
	); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("autoridad manipulada no se denego: %v", err)
	}
}

func TestAutoridadInternaEjecucionDocumentalV4AdmiteLecturasConcurrentes(t *testing.T) {
	escenario := nuevoEscenarioAutoridadInternaEjecucionDocumentalV4(t)
	autoridad, err := emitirAutoridadEscenarioPDPPrueba(t, escenario)
	if err != nil {
		t.Fatal(err)
	}

	const lectores = 16
	var grupo sync.WaitGroup
	errores := make(chan error, lectores)
	for indice := 0; indice < lectores; indice++ {
		grupo.Add(1)
		go func(desplazamiento int) {
			defer grupo.Done()
			instante := escenario.emitidaEn.Add(time.Duration(desplazamiento+1) * time.Microsecond)
			solicitud, err := autoridad.PrepararAplicacionExactaEn(
				escenario.decision.DecisionRef,
				escenario.expectativa.HuellaPlanSHA256,
				escenario.expectativa.EfectoRef,
				instante,
			)
			if err != nil {
				errores <- err
				return
			}
			_, err = solicitud.ProyeccionParaTransaccion()
			if err != nil {
				errores <- err
			}
		}(indice)
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("lectura concurrente: %v", err)
	}
}

type escenarioAutoridadInternaEjecucionDocumentalV4 struct {
	decision    domain.DecisionAutorizacion
	evidencia   ports.EvidenciaUsoDecisionAutorizacion
	expectativa ports.ExpectativaAutorizacionEjecucionDocumentalV4
	emitidaEn   time.Time
}

func nuevoEscenarioAutoridadInternaEjecucionDocumentalV4(
	t *testing.T,
) escenarioAutoridadInternaEjecucionDocumentalV4 {
	t.Helper()
	decisionEmitidaEn := time.Date(2026, time.July, 15, 10, 0, 0, 123_456_000, time.UTC)
	vinculo, err := pruebasvec.NuevoVinculoGenerico(decisionEmitidaEn)
	if err != nil {
		t.Fatalf("crear vinculo de prueba: %v", err)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		t.Fatalf("leer vinculo de prueba: %v", err)
	}
	referenciasPoliticas := []string{"politica:proteccion:v2", "politica:seguridad:v4"}
	huellasPoliticas := map[string]string{
		referenciasPoliticas[0]: huellaInternaPrueba('1'),
		referenciasPoliticas[1]: huellaInternaPrueba('2'),
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(
		referenciasPoliticas,
		huellasPoliticas,
	)
	if err != nil {
		t.Fatalf("crear catalogo de prueba: %v", err)
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "recurso:ejecucion-documental:001",
		ModuloID:   "bolsa",
		Tipo:       "documento_bolsa",
		Ambitos: map[string]string{
			"organizacion":  "diputacion_granada",
			"procedimiento": "bolsa_auxiliares_2026",
		},
		Atributos: map[string]string{
			ports.AtributoAutorizacionDocumentalEfectoRef:        "efecto:documental:v4:001",
			ports.AtributoAutorizacionDocumentalHuellaPlanSHA256: huellaInternaPrueba('a'),
		},
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("crear huella de recurso: %v", err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:documental:v4:001", Concedida: true, Codigo: "concedida",
		PrincipalID: datosVinculo.PrincipalID, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		Accion:     ports.AccionEjecutarPlanDocumentalV4,
		RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		ContextoRecursoHuellaSHA256:           huellaRecurso,
		Finalidad:                             "tramitar_bolsa_auxiliares",
		CorrelacionRef:                        "correlacion:ejecucion-documental:v4:001",
		VinculoAutenticacionActor:             vinculo,
		AsignacionRef:                         "asignacion:rrhh:001:v3",
		AsignacionHuellaSHA256:                huellaInternaPrueba('3'),
		VersionRolRef:                         "rol:tecnico_rrhh:v7",
		VersionRolHuellaSHA256:                huellaInternaPrueba('4'),
		ControlVigenciaVersionRolRef:          "rol:tecnico_rrhh:v7",
		ControlVigenciaVersionRolRevision:     11,
		ControlVigenciaVersionRolHuellaSHA256: huellaInternaPrueba('5'),
		RevisionCatalogoPoliticas:             19,
		CatalogoPoliticasHuellaSHA256:         huellaCatalogo,
		PoliticasEvaluadasRefs:                append([]string(nil), referenciasPoliticas...),
		PoliticasEvaluadasHuellasSHA256:       clonarMapaInterno(huellasPoliticas),
		PoliticasRefs:                         append([]string(nil), referenciasPoliticas...),
		PoliticasHuellasSHA256:                clonarMapaInterno(huellasPoliticas),
		GarantiaMinima:                        domain.AuthAssuranceHigh,
		CamposPermitidos:                      []string{"documento.generado"},
		EmitidaEn:                             decisionEmitidaEn,
		ValidaHasta:                           decisionEmitidaEn.Add(2 * time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision de prueba invalida: %v", err)
	}
	verificadaEn := decisionEmitidaEn.Add(time.Second)
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		t.Fatalf("crear evidencia: %v", err)
	}
	expectativa := ports.ExpectativaAutorizacionEjecucionDocumentalV4{
		DecisionEsperada:                clonarDecisionInterna(decision),
		PrincipalID:                     decision.PrincipalID,
		PerfilActivoRef:                 decision.PerfilActivoRef,
		AutenticacionRef:                datosVinculo.AutenticacionRef,
		SesionRef:                       datosVinculo.SesionRef,
		ControlSesionRef:                datosVinculo.ControlSesionRef,
		ControlSesionRevision:           datosVinculo.ControlSesionRevision,
		ControlSesionHuellaSHA256:       datosVinculo.ControlSesionHuellaSHA256,
		ContextoActorRef:                datosVinculo.ContextoActorRef,
		ContextoActorVersion:            datosVinculo.ContextoActorVersion,
		ContextoActorHuellaSHA256:       datosVinculo.ContextoActorHuellaSHA256,
		Recurso:                         clonarRecursoInterno(recurso),
		Finalidad:                       decision.Finalidad,
		CorrelacionRef:                  decision.CorrelacionRef,
		EfectoRef:                       recurso.Atributos[ports.AtributoAutorizacionDocumentalEfectoRef],
		HuellaPlanSHA256:                recurso.Atributos[ports.AtributoAutorizacionDocumentalHuellaPlanSHA256],
		CamposPermitidosEsperados:       append([]string(nil), decision.CamposPermitidos...),
		ObligacionesEsperadas:           nil,
		CumplimientosObligacionesPorRef: nil,
	}
	return escenarioAutoridadInternaEjecucionDocumentalV4{
		decision: decision, evidencia: evidencia, expectativa: expectativa,
		emitidaEn: verificadaEn.Add(time.Second),
	}
}

func clonarDecisionInterna(decision domain.DecisionAutorizacion) domain.DecisionAutorizacion {
	decision.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasEvaluadasRefs...)
	decision.PoliticasEvaluadasHuellasSHA256 = clonarMapaInterno(decision.PoliticasEvaluadasHuellasSHA256)
	decision.PoliticasRefs = append([]string(nil), decision.PoliticasRefs...)
	decision.PoliticasHuellasSHA256 = clonarMapaInterno(decision.PoliticasHuellasSHA256)
	decision.CamposPermitidos = append([]string(nil), decision.CamposPermitidos...)
	decision.Obligaciones = append([]string(nil), decision.Obligaciones...)
	return decision
}

func clonarRecursoInterno(recurso domain.RecursoAutorizable) domain.RecursoAutorizable {
	recurso.Ambitos = clonarMapaInterno(recurso.Ambitos)
	recurso.Atributos = clonarMapaInterno(recurso.Atributos)
	return recurso
}

func clonarMapaInterno(origen map[string]string) map[string]string {
	copia := make(map[string]string, len(origen))
	for clave, valor := range origen {
		copia[clave] = valor
	}
	return copia
}

func huellaInternaPrueba(caracter byte) string {
	return strings.Repeat(string(caracter), 64)
}

func emitirAutoridadEscenarioPDPPrueba(
	t *testing.T,
	escenario escenarioAutoridadInternaEjecucionDocumentalV4,
) (AutoridadInternaEjecucionDocumentalV4, error) {
	t.Helper()
	vinculo, _ := ports.NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		escenario.evidencia,
		escenario.expectativa,
		escenario.emitidaEn,
	)

	material := generarMaterialFirmaCOSEPrueba(
		t,
		AlgoritmoCOSEDocumentalEdDSA,
		[]byte("clave:pdp:autoridad-v4"),
	)
	ancla := escenario.emitidaEn.UTC().Truncate(time.Microsecond)
	raiz, err := nuevaRaizPublicaFijadaAtestacionPDP(
		material.claveID,
		material.algoritmoDocumental,
		material.publica,
		suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		audienciaDespliegueAtestacionPDPPrueba,
		EstadoConfianzaClaveDocumentalActiva,
		ancla.Add(-time.Hour),
		ancla.Add(time.Hour),
		time.Time{},
	)
	if err != nil {
		t.Fatalf("crear raiz PDP de prueba: %v", err)
	}
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:pdp:autoridad-v4", ancla.Add(-time.Hour), ancla.Add(time.Hour), raiz,
	)
	if err != nil {
		t.Fatalf("crear configuracion PDP de prueba: %v", err)
	}
	servicio, err := nuevoServicioConReloj(
		configuracion,
		&relojConfianzaDocumentalFijo{ahora: escenario.emitidaEn},
	)
	if err != nil {
		t.Fatalf("crear servicio PDP de prueba: %v", err)
	}

	decision := escenario.decision
	if datos, err := escenario.evidencia.Datos(); err == nil {
		decision = datos.Decision
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
		Suite:          suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		ClaveID:        string(material.claveID),
		Audiencia:      audienciaDespliegueAtestacionPDPPrueba,
	}
	payload, err := domain.SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		t.Fatalf("serializar atestacion PDP de prueba: %v", err)
	}
	solicitudCOSE := nuevaSolicitudCOSEPrueba(
		t,
		payload,
		AudienciaCOSEAtestacionAutorizacionPDP,
	)
	sobre := firmarSobreCOSEPrueba(
		t,
		material,
		payload,
		solicitudCOSE,
		nil,
		nil,
	)
	return servicio.EmitirAutoridadInternaEjecucionDocumentalV4(
		context.Background(),
		vinculo,
		cabecera,
		sobre,
	)
}
