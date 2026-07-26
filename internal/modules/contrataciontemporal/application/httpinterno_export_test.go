package application

import "testing"

// EscenarioHTTPDecisionPrueba expone servicios reales únicamente a las
// pruebas externas del paquete. No crea resultados, sellos ni recibos.
type EscenarioHTTPDecisionPrueba struct {
	Presentador *ServicioPresentacionPropuestaCobertura
	Decisor     *ServicioConfirmacionDecisionCobertura
	Propuesta   SolicitudProponerCobertura
	Decision    SolicitudDecidirCobertura
}

func NuevoEscenarioHTTPDecisionPrueba(t *testing.T) EscenarioHTTPDecisionPrueba {
	t.Helper()
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	return EscenarioHTTPDecisionPrueba{
		Presentador: escenario.base.servicio,
		Decisor:     escenario.servicio,
		Propuesta:   escenario.base.solicitud,
		Decision:    escenario.solicitud,
	}
}

type EscenarioHTTPRectificacionPrueba struct {
	Presentador   *ServicioPresentacionPropuestaCobertura
	Decisor       *ServicioConfirmacionDecisionCobertura
	Rectificacion SolicitudRectificarCobertura
}

func NuevoEscenarioHTTPRectificacionPrueba(t *testing.T) EscenarioHTTPRectificacionPrueba {
	t.Helper()
	escenario := nuevoEscenarioRectificacionConfirmacionCobertura(t, true)
	return EscenarioHTTPRectificacionPrueba{
		Presentador:   escenario.base.servicio,
		Decisor:       escenario.servicio,
		Rectificacion: escenario.solicitudRectificar,
	}
}
