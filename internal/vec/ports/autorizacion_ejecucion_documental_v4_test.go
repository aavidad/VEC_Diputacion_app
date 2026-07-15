package ports

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestSolicitudVinculadaAutorizacionEjecucionDocumentalV4LigaAlcanceExactoYNoExponeAutoridad(t *testing.T) {
	escenario := nuevoEscenarioAutorizacionEjecucionDocumentalV4(t)
	vinculo, err := NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		escenario.evidencia,
		escenario.expectativa,
		escenario.vinculadaEn,
	)
	if err != nil {
		t.Fatalf("crear solicitud vinculada exacta: %v", err)
	}
	if err := vinculo.ValidarEn(escenario.vinculadaEn); err != nil {
		t.Fatalf("solicitud vinculada valida rechazada: %v", err)
	}
	huellaVinculo, err := vinculo.HuellaSHA256()
	if err != nil || !esSHA256Hexadecimal(huellaVinculo) {
		t.Fatalf("huella de solicitud vinculada invalida: %q, %v", huellaVinculo, err)
	}

	solicitadaEn := escenario.vinculadaEn.Add(time.Microsecond)
	solicitud, err := vinculo.PrepararSolicitudAplicacionEn(solicitadaEn)
	if err != nil {
		t.Fatalf("preparar solicitud de aplicacion: %v", err)
	}
	if err := solicitud.ValidarContraEn(
		escenario.decision.DecisionRef,
		escenario.expectativa.HuellaPlanSHA256,
		escenario.expectativa.EfectoRef,
		solicitadaEn,
	); err != nil {
		t.Fatalf("solicitud exacta rechazada: %v", err)
	}
	datos, err := solicitud.ProyeccionParaTransaccion()
	if err != nil {
		t.Fatalf("proyectar solicitud: %v", err)
	}
	claveEsperada := ClaveAplicacionAutorizacionEjecucionDocumentalV4{
		DecisionRef:      escenario.decision.DecisionRef,
		HuellaPlanSHA256: escenario.expectativa.HuellaPlanSHA256,
		EfectoRef:        escenario.expectativa.EfectoRef,
	}
	if datos.Esquema != EsquemaSolicitudAplicacionAutorizacionEjecucionDocumentalV4 ||
		datos.Clave != claveEsperada || datos.Accion != AccionEjecutarPlanDocumentalV4 ||
		datos.HuellaSolicitudVinculadaSHA256 != huellaVinculo ||
		!esSHA256Hexadecimal(datos.HuellaSolicitudAplicacionSHA256) ||
		!datos.SolicitadaEn.Equal(solicitadaEn) {
		t.Fatalf("proyeccion transaccional inesperada: %+v", datos)
	}
	evidencia, err := solicitud.EvidenciaEstructural()
	if err != nil || evidencia.ValidarEn(solicitadaEn) != nil {
		t.Fatalf("la solicitud no conservo evidencia vigente: %v", err)
	}

	// Dos solicitudes conservan la misma clave UNIQUE. El adaptador debe
	// insertar DecisionRef junto con la activacion; nunca consumirla aparte.
	segundo, err := vinculo.PrepararSolicitudAplicacionEn(solicitadaEn.Add(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	datosSegundo, _ := segundo.ProyeccionParaTransaccion()
	if datosSegundo.Clave != datos.Clave ||
		datosSegundo.HuellaSolicitudAplicacionSHA256 == datos.HuellaSolicitudAplicacionSHA256 {
		t.Fatal("la clave atomica o el reto temporal de aplicacion no quedaron separados")
	}

	// El constructor debe haber copiado mapas y porciones aportados por el
	// llamador. Mutarlos despues no puede alterar la solicitud ya vinculada.
	escenario.expectativa.Recurso.Ambitos["organizacion"] = "otra"
	escenario.expectativa.Recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef] = "efecto:otro"
	escenario.expectativa.CamposPermitidosEsperados[0] = "campo.ampliado"
	escenario.decision.CamposPermitidos[0] = "campo.mutado"
	if err := vinculo.ValidarEn(solicitadaEn); err != nil {
		t.Fatalf("un alias del llamador altero la solicitud vinculada: %v", err)
	}

	for nombre, valor := range map[string]any{
		"vinculo":   vinculo,
		"solicitud": solicitud,
		"datos":     datos,
	} {
		t.Run("serializacion_"+nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); !errors.Is(
				err,
				ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida,
			) {
				t.Fatalf("JSON no fue denegado: %v", err)
			}
			texto := fmt.Sprintf("%v %+v %#v", valor, valor, valor)
			if strings.Contains(texto, escenario.decision.DecisionRef) ||
				strings.Contains(texto, escenario.decision.PrincipalID) ||
				strings.Contains(texto, escenario.expectativa.HuellaPlanSHA256) {
				t.Fatalf("el formato expuso autorizacion: %s", texto)
			}
		})
	}
}

func TestSolicitudVinculadaAutorizacionEjecucionDocumentalV4DeniegaAusenciaYTodaDiscrepancia(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*escenarioAutorizacionEjecucionDocumentalV4)
	}{
		{
			"accion ajena",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.decision.Accion = "vec.documentos.ejecucion.consultar"
			},
		},
		{
			"actor distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.PrincipalID = "persona:otra"
			},
		},
		{
			"perfil distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.PerfilActivoRef = "perfil:otro"
			},
		},
		{
			"sesion distinta",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.SesionRef = "sesion:otra"
			},
		},
		{
			"contexto actor distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.ContextoActorHuellaSHA256 = huellaPrueba('d')
			},
		},
		{
			"vinculo completo no esperado",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.VinculoAutenticacionActor = domain.VinculoAutenticacionActorV1{}
			},
		},
		{
			"asignacion distinta",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.AsignacionRef = "asignacion:rrhh:otra:v3"
			},
		},
		{
			"version de rol distinta",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.VersionRolHuellaSHA256 = huellaPrueba('d')
			},
		},
		{
			"control de vigencia distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.ControlVigenciaVersionRolRevision++
			},
		},
		{
			"revision de catalogo distinta",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.RevisionCatalogoPoliticas++
			},
		},
		{
			"politicas esperadas ausentes",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.PoliticasRefs = nil
				e.expectativa.DecisionEsperada.PoliticasHuellasSHA256 = nil
			},
		},
		{
			"garantia distinta",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.DecisionEsperada.GarantiaMinima = domain.AuthAssuranceSubstantial
			},
		},
		{
			"recurso distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.Recurso.Referencia = "recurso:documental:otro"
			},
		},
		{
			"ambito distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.Recurso.Ambitos["organizacion"] = "otra"
			},
		},
		{
			"ambito ausente",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.Recurso.Ambitos = nil
			},
		},
		{
			"atributo ambiguo adicional",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.Recurso.Atributos["modo"] = "ampliado"
			},
		},
		{
			"efecto distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.EfectoRef = "efecto:documental:otro"
			},
		},
		{
			"plan distinto",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.HuellaPlanSHA256 = huellaPrueba('e')
			},
		},
		{
			"finalidad distinta",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.Finalidad = "finalidad:otra"
			},
		},
		{
			"correlacion distinta",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.CorrelacionRef = "correlacion:otra"
			},
		},
		{
			"campos no especificados",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.CamposPermitidosEsperados = nil
			},
		},
		{
			"obligacion sin prueba",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.ObligacionesEsperadas = []string{"doble_control"}
			},
		},
		{
			"cumplimiento no exigido",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.CumplimientosObligacionesPorRef = map[string]string{
					"doble_control": "evidencia:doble-control:001",
				}
			},
		},
		{
			"comodin",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.expectativa.PrincipalID = "persona:*"
			},
		},
		{
			"instante anterior a verificacion",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.vinculadaEn = e.verificadaEn.Add(-time.Microsecond)
			},
		},
		{
			"instante no canonico",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.vinculadaEn = e.vinculadaEn.In(time.FixedZone("UTC equivalente", 0))
			},
		},
		{
			"caducada",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.vinculadaEn = e.decision.ValidaHasta
			},
		},
		{
			"decision denegada",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.decision.Concedida = false
				e.decision.Codigo = "denegada"
			},
		},
		{
			"decision con obligacion no acreditable",
			func(e *escenarioAutorizacionEjecucionDocumentalV4) {
				e.decision.Obligaciones = []string{"doble_control"}
				e.expectativa.ObligacionesEsperadas = []string{"doble_control"}
				e.expectativa.CumplimientosObligacionesPorRef = map[string]string{
					"doble_control": "evidencia:doble-control:001",
				}
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAutorizacionEjecucionDocumentalV4SinEvidencia(t)
			caso.mutar(&escenario)
			evidencia, _ := NuevaEvidenciaUsoDecisionAutorizacion(
				escenario.decision,
				escenario.verificadaEn,
			)
			vinculo, err := NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
				evidencia,
				escenario.expectativa,
				escenario.vinculadaEn,
			)
			comprobarDenegacionAutorizacionEjecucionDocumentalV4(t, vinculo, err)
		})
	}

	t.Run("fallo o ausencia de PDP", func(t *testing.T) {
		escenario := nuevoEscenarioAutorizacionEjecucionDocumentalV4SinEvidencia(t)
		vinculo, err := NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
			EvidenciaUsoDecisionAutorizacion{},
			escenario.expectativa,
			escenario.vinculadaEn,
		)
		comprobarDenegacionAutorizacionEjecucionDocumentalV4(t, vinculo, err)
	})

	var cero SolicitudVinculadaAutorizacionEjecucionDocumentalV4
	if err := cero.ValidarEn(time.Now().UTC().Truncate(time.Microsecond)); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("el valor cero no se denego: %v", err)
	}
	if _, err := cero.PrepararSolicitudAplicacionEn(time.Now().UTC().Truncate(time.Microsecond)); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("el valor cero preparo una solicitud: %v", err)
	}
}

func TestSolicitudAplicacionAutorizacionEjecucionDocumentalV4EsExactaTemporalYAntimanipulacion(t *testing.T) {
	escenario := nuevoEscenarioAutorizacionEjecucionDocumentalV4(t)
	vinculo, err := NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		escenario.evidencia,
		escenario.expectativa,
		escenario.vinculadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitadaEn := escenario.vinculadaEn.Add(time.Microsecond)
	solicitud, err := vinculo.PrepararSolicitudAplicacionEn(solicitadaEn)
	if err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre      string
		decisionRef string
		plan        string
		efecto      string
		instante    time.Time
	}{
		{
			"decision distinta", "decision:otra", escenario.expectativa.HuellaPlanSHA256,
			escenario.expectativa.EfectoRef, solicitadaEn,
		},
		{
			"plan distinto", escenario.decision.DecisionRef, huellaPrueba('f'),
			escenario.expectativa.EfectoRef, solicitadaEn,
		},
		{
			"efecto distinto", escenario.decision.DecisionRef, escenario.expectativa.HuellaPlanSHA256,
			"efecto:documental:otro", solicitadaEn,
		},
		{
			"antes de solicitar", escenario.decision.DecisionRef, escenario.expectativa.HuellaPlanSHA256,
			escenario.expectativa.EfectoRef, solicitadaEn.Add(-time.Microsecond),
		},
		{
			"en caducidad exclusiva", escenario.decision.DecisionRef, escenario.expectativa.HuellaPlanSHA256,
			escenario.expectativa.EfectoRef, escenario.decision.ValidaHasta,
		},
		{
			"instante no canonico", escenario.decision.DecisionRef, escenario.expectativa.HuellaPlanSHA256,
			escenario.expectativa.EfectoRef, solicitadaEn.Add(time.Nanosecond),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := solicitud.ValidarContraEn(
				caso.decisionRef,
				caso.plan,
				caso.efecto,
				caso.instante,
			); !errors.Is(err, ErrAutorizacionEjecucionDocumentalV4Invalida) ||
				!errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("solicitud discrepante no denegada: %v", err)
			}
		})
	}

	manipulado := solicitud
	copia := *solicitud.datos
	copia.huella = huellaPrueba('0')
	manipulado.datos = &copia
	if err := manipulado.ValidarContraEn(
		escenario.decision.DecisionRef,
		escenario.expectativa.HuellaPlanSHA256,
		escenario.expectativa.EfectoRef,
		solicitadaEn,
	); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("solicitud manipulada no se denego: %v", err)
	}

	var cero SolicitudAplicacionAutorizacionEjecucionDocumentalV4
	if _, err := cero.ProyeccionParaTransaccion(); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("solicitud cero produjo datos: %v", err)
	}
	if _, err := cero.EvidenciaEstructural(); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("solicitud cero produjo evidencia: %v", err)
	}
}

func TestSolicitudVinculadaAutorizacionEjecucionDocumentalV4AdmiteLecturaConcurrenteSinAlias(t *testing.T) {
	escenario := nuevoEscenarioAutorizacionEjecucionDocumentalV4(t)
	vinculo, err := NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		escenario.evidencia,
		escenario.expectativa,
		escenario.vinculadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}

	const lectores = 16
	errores := make(chan error, lectores)
	var grupo sync.WaitGroup
	for indice := 0; indice < lectores; indice++ {
		grupo.Add(1)
		go func(desplazamiento int) {
			defer grupo.Done()
			instante := escenario.vinculadaEn.Add(time.Duration(desplazamiento+1) * time.Microsecond)
			if err := vinculo.ValidarEn(instante); err != nil {
				errores <- err
				return
			}
			solicitud, err := vinculo.PrepararSolicitudAplicacionEn(instante)
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

type escenarioAutorizacionEjecucionDocumentalV4 struct {
	decision     domain.DecisionAutorizacion
	evidencia    EvidenciaUsoDecisionAutorizacion
	expectativa  ExpectativaAutorizacionEjecucionDocumentalV4
	verificadaEn time.Time
	vinculadaEn  time.Time
}

func nuevoEscenarioAutorizacionEjecucionDocumentalV4(
	t *testing.T,
) escenarioAutorizacionEjecucionDocumentalV4 {
	t.Helper()
	escenario := nuevoEscenarioAutorizacionEjecucionDocumentalV4SinEvidencia(t)
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(
		escenario.decision,
		escenario.verificadaEn,
	)
	if err != nil {
		t.Fatalf("crear evidencia raiz: %v", err)
	}
	escenario.evidencia = evidencia
	return escenario
}

func nuevoEscenarioAutorizacionEjecucionDocumentalV4SinEvidencia(
	t *testing.T,
) escenarioAutorizacionEjecucionDocumentalV4 {
	t.Helper()
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	recurso := domain.RecursoAutorizable{
		Referencia: "recurso:ejecucion-documental:001",
		ModuloID:   "bolsa",
		Tipo:       "documento_bolsa",
		Ambitos: map[string]string{
			"organizacion":  "diputacion_granada",
			"procedimiento": "bolsa_auxiliares_2026",
		},
		Atributos: map[string]string{
			AtributoAutorizacionDocumentalEfectoRef:        "efecto:documental:v4:001",
			AtributoAutorizacionDocumentalHuellaPlanSHA256: huellaPrueba('a'),
		},
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("crear huella de recurso: %v", err)
	}
	decision.Accion = AccionEjecutarPlanDocumentalV4
	decision.RecursoRef = recurso.Referencia
	decision.ModuloID = recurso.ModuloID
	decision.TipoRecurso = recurso.Tipo
	decision.ContextoRecursoHuellaSHA256 = huellaRecurso
	decision.Finalidad = "tramitar_bolsa_auxiliares"
	decision.CorrelacionRef = "correlacion:ejecucion-documental:v4:001"
	decision.CamposPermitidos = []string{"documento.generado"}
	decision.Obligaciones = nil
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision documental de prueba invalida: %v", err)
	}
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatalf("extraer vinculo de actor: %v", err)
	}
	expectativa := ExpectativaAutorizacionEjecucionDocumentalV4{
		DecisionEsperada: clonarDecisionAutorizacionCanonica(decision),
		PrincipalID:      decision.PrincipalID, PerfilActivoRef: decision.PerfilActivoRef,
		AutenticacionRef: vinculo.AutenticacionRef, SesionRef: vinculo.SesionRef,
		ControlSesionRef: vinculo.ControlSesionRef, ControlSesionRevision: vinculo.ControlSesionRevision,
		ControlSesionHuellaSHA256: vinculo.ControlSesionHuellaSHA256,
		ContextoActorRef:          vinculo.ContextoActorRef, ContextoActorVersion: vinculo.ContextoActorVersion,
		ContextoActorHuellaSHA256: vinculo.ContextoActorHuellaSHA256,
		Recurso:                   recurso, Finalidad: decision.Finalidad, CorrelacionRef: decision.CorrelacionRef,
		EfectoRef:                       recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef],
		HuellaPlanSHA256:                recurso.Atributos[AtributoAutorizacionDocumentalHuellaPlanSHA256],
		CamposPermitidosEsperados:       append([]string(nil), decision.CamposPermitidos...),
		ObligacionesEsperadas:           nil,
		CumplimientosObligacionesPorRef: nil,
	}
	return escenarioAutorizacionEjecucionDocumentalV4{
		decision: decision, expectativa: expectativa, verificadaEn: verificadaEn,
		vinculadaEn: verificadaEn.Add(time.Microsecond),
	}
}

func comprobarDenegacionAutorizacionEjecucionDocumentalV4(
	t *testing.T,
	vinculo SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	err error,
) {
	t.Helper()
	if err == nil || !errors.Is(err, ErrAutorizacionEjecucionDocumentalV4Invalida) ||
		!errors.Is(err, domain.ErrAutorizacionDenegada) || vinculo.datos != nil {
		t.Fatalf("se esperaba denegacion uniforme; vinculo=%+v, error=%v", vinculo, err)
	}
}
