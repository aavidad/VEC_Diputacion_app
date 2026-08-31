package application

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSeleccionLlamamientoReplayExactoUsaIdempotenciaDelPuerto(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	solicitud := SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion}
	primero, err := e.servicio.SeleccionarYLlamarParaAdaptador(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primer intento: %v", err)
	}
	e.disponibilidad.cantidad = 1
	e.preparador.alternarPolitica = true
	segundo, err := e.servicio.SeleccionarYLlamarParaAdaptador(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if primero.ReciboRef == "" || primero.ConfirmadaEn.IsZero() || primero != segundo ||
		e.preparador.consultasPreparadas != 2 || e.preparador.ordenesPreparadas != 1 ||
		e.ejecuciones.resoluciones != 2 ||
		e.disponibilidad.llamadas != 1 || e.llamamientos.llamadas != 1 || e.ordenes.llamadas != 1 ||
		e.llamamientos.creaciones != 1 {
		t.Fatalf("replay no revalidó autoridad o repitió efectos: igual=%v preparadas=%d resoluciones=%d disponibilidad=%d ordenes=%d llamadas=%d creaciones=%d",
			primero == segundo, e.preparador.consultasPreparadas, e.ejecuciones.resoluciones, e.disponibilidad.llamadas,
			e.ordenes.llamadas, e.llamamientos.llamadas, e.llamamientos.creaciones)
	}
}

func TestSeleccionLlamamientoAutoridadAusenteCaducadaOCruzadaNoConsultaNiActua(t *testing.T) {
	for _, dimension := range []string{"organización", "expediente", "versión", "correlación", "autoridad solicitante",
		"autorización", "acción", "recurso", "finalidad", "ausente", "caducada"} {
		t.Run(dimension, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			if dimension != "ausente" && dimension != "caducada" {
				_, err := e.ejecutar(context.Background())
				if err != nil {
					t.Fatal(err)
				}
			}
			antes := e.conteosFrontera()
			e.preparador.mutarContexto = func(operacion string, datos *ports.DatosContextoPeticionIntegracionBolsa) {
				if operacion != "operacion:disponibilidad:001" {
					return
				}
				switch dimension {
				case "organización":
					datos.OrganizacionRef = "organizacion:ajena"
				case "expediente":
					datos.ExpedienteRef = "expediente:ajeno:001"
				case "versión":
					datos.VersionExpediente++
				case "correlación":
					datos.CorrelacionRef = "correlacion:ajena"
				case "autoridad solicitante", "ausente":
					datos.AutoridadSolicitante = map[bool]string{true: "", false: "autoridad:ajena"}[dimension == "ausente"]
				case "autorización":
					datos.Autorizacion = referenciaSeleccionPrueba("autorizacion:ajena:001", '1')
				case "acción":
					datos.Accion = referenciaSeleccionPrueba("accion:ajena:001", '2')
				case "recurso":
					datos.Recurso = referenciaSeleccionPrueba("necesidad:ajena:001", '3')
				case "finalidad":
					datos.Finalidad = referenciaSeleccionPrueba("finalidad:ajena:001", '4')
				case "caducada":
					datos.ValidaHasta = datos.SolicitadaEn.Add(time.Minute)
				}
			}
			_, err := e.ejecutar(context.Background())
			if despues := e.conteosFrontera(); err == nil || antes != despues {
				t.Fatalf("autoridad inválida consultó o actuó: antes=%v después=%v err=%v", antes, despues, err)
			}
		})
	}
}

func (e *escenarioSeleccionLlamamientoPrueba) conteosFrontera() [5]int {
	return [5]int{e.ejecuciones.resoluciones, e.disponibilidad.llamadas,
		e.ordenes.llamadas, e.llamamientos.llamadas, e.llamamientos.creaciones}
}
