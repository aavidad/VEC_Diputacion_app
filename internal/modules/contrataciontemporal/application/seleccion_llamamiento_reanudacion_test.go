package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecucionesReanudablesSeleccionPrueba struct {
	*ejecucionesSeleccionLlamamientoPrueba
	reanudaciones, ventanasOrden int
	denegada                     bool
}

func (e *ejecucionesReanudablesSeleccionPrueba) ReanudarPreparacionOrden(ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	e.reanudaciones++
	if e.denegada || ctx.Err() != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, ports.ErrAutorizacionDenegada
	}
	e.Lock()
	defer e.Unlock()
	if solicitud != e.solicitud || e.situacion != ports.EjecucionSeleccionLlamamientoIndeterminada || e.efecto != ports.EfectoPrepararOrdenSeleccionLlamamiento {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, errors.New("reanudación no aplicable")
	}
	e.reserva = "reserva:seleccion:reanudada"
	e.situacion = ports.EjecucionSeleccionLlamamientoPropietaria
	return e.estado(true), nil
}

func (e *ejecucionesReanudablesSeleccionPrueba) AbrirVentanaEfecto(ctx context.Context, reserva ports.ReservaEjecucionSeleccionLlamamiento, efecto ports.EfectoSeleccionLlamamiento) error {
	if efecto == ports.EfectoPrepararOrdenSeleccionLlamamiento {
		e.ventanasOrden++
	}
	return e.ejecucionesSeleccionLlamamientoPrueba.AbrirVentanaEfecto(ctx, reserva, efecto)
}

func TestSeleccionReanudaOrdenSinReabrirVentanaNiCambiarSolicitud(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	reanudables := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones}
	e.servicio.ejecuciones = reanudables
	e.ordenes.err = errors.New("recibo no recibido")
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) {
		t.Fatalf("fallo inicial: %v", err)
	}
	solicitud := e.ejecuciones.solicitud
	e.ordenes.err = nil
	recibo, err := e.ejecutar(context.Background())
	if err != nil || !recibo.PropuestaGenerada || reanudables.reanudaciones != 1 ||
		reanudables.ventanasOrden != 1 || e.ejecuciones.solicitud != solicitud || e.llamamientos.creaciones != 1 {
		t.Fatalf("no continuó la misma intención sin reabrir ventana: %v", err)
	}
	recuperado, err := e.ejecutar(context.Background())
	if err != nil || recuperado != recibo || reanudables.reanudaciones != 1 || e.llamamientos.creaciones != 1 {
		t.Fatalf("el terminal no recuperó el mismo resultado: %v", err)
	}
}

func TestSeleccionReanudacionDenegadaNoRepiteOrden(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	reanudables := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones, denegada: true}
	e.servicio.ejecuciones = reanudables
	e.ordenes.err = errors.New("recibo no recibido")
	_, _ = e.ejecutar(context.Background())
	e.ordenes.err = nil
	if _, err := e.ejecutar(context.Background()); err == nil || reanudables.reanudaciones != 1 || e.ordenes.llamadas != 1 || e.llamamientos.llamadas != 0 {
		t.Fatalf("reanudación denegada produjo efectos: %v", err)
	}
}

func TestSeleccionNoReanudaLlamamientoIndeterminadoComoOrden(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	reanudables := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones}
	e.servicio.ejecuciones = reanudables
	e.llamamientos.err = errors.New("resultado no recibido")
	_, _ = e.ejecutar(context.Background())
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) ||
		reanudables.reanudaciones != 0 || e.ordenes.llamadas != 1 || e.llamamientos.llamadas != 1 {
		t.Fatalf("abrió otra fase no autorizada: %v", err)
	}
}
