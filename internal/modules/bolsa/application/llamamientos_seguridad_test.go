package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestServicioLlamamientosSoloAdmiteSuperficiesInternasConGarantiaAlta(t *testing.T) {
	casos := map[string]func(*testing.T, *escenarioAplicacionLlamamiento){
		"superficie_externa": func(_ *testing.T, escenario *escenarioAplicacionLlamamiento) {
			escenario.superficie = dominiovec.SuperficieAutenticacionExternaPersonalV1
		},
		"garantia_insuficiente": func(t *testing.T, escenario *escenarioAplicacionLlamamiento) {
			actor := actorCanonicoAplicacionConAutenticacionPrueba(
				t,
				dominiovec.AuthMethodCertificate,
				dominiovec.AuthAssuranceSubstantial,
			)
			escenario.solicitud.Actor = actor
			escenario.solicitud.PerfilActivoRef = actor.PerfilActivoRef
			escenario.personaEsperadaRef = actor.PersonaRef
		},
		"metodo_demo": func(t *testing.T, escenario *escenarioAplicacionLlamamiento) {
			actor := actorCanonicoAplicacionConAutenticacionPrueba(
				t,
				dominiovec.AuthMethodDemo,
				dominiovec.AuthAssuranceHigh,
			)
			escenario.solicitud.Actor = actor
			escenario.solicitud.PerfilActivoRef = actor.PerfilActivoRef
			escenario.personaEsperadaRef = actor.PersonaRef
		},
	}

	for nombre, preparar := range casos {
		t.Run(nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAplicacionLlamamiento(t)
			preparar(t, escenario)
			_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.persistencias != 0 ||
				!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo"}) {
				t.Fatalf("autenticacion no apta alcanzo el PDP: secuencia=%v error=%v", escenario.secuencia, err)
			}
		})
	}
}

func TestServicioLlamamientosAdmiteSuperficieCorporativaYPrivilegiada(t *testing.T) {
	casos := map[string]func(*escenarioAplicacionLlamamiento){
		"corporativa": func(*escenarioAplicacionLlamamiento) {},
		"administracion_privilegiada": func(escenario *escenarioAplicacionLlamamiento) {
			escenario.superficie = dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1
			escenario.cuentaPrivilegiada = true
		},
	}

	for nombre, preparar := range casos {
		t.Run(nombre, func(t *testing.T) {
			escenario := nuevoEscenarioAplicacionLlamamiento(t)
			preparar(escenario)
			propuesta, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
			if err != nil || propuesta.Validar() != nil || propuesta.OrdenSeleccionado != 2 ||
				escenario.evaluadas != 2 || escenario.persistencias != 1 {
				t.Fatalf("superficie interna valida rechazada: propuesta=%+v error=%v", propuesta, err)
			}
			if propuesta.Evaluaciones[1].Resultado != dominiobolsa.ResultadoElegible {
				t.Fatalf("se altero la seleccion del primer elegible: %+v", propuesta.Evaluaciones)
			}
		})
	}
}

func TestServicioLlamamientosExigeGarantiaMinimaAltaEnDecision(t *testing.T) {
	escenario := nuevoEscenarioAplicacionLlamamiento(t)
	escenario.mutarDecision = func(decision *dominiovec.DecisionAutorizacion) {
		decision.GarantiaMinima = dominiovec.AuthAssuranceSubstantial
	}
	_, err := escenario.servicio(t).ProponerPrimerLlamamiento(context.Background(), escenario.solicitud)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || escenario.persistencias != 0 ||
		!reflect.DeepEqual(escenario.secuencia, []string{"recurso", "vinculo", "autorizar"}) {
		t.Fatalf("decision sin garantia minima alta habilito datos: secuencia=%v error=%v", escenario.secuencia, err)
	}
}
