package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecucionesReanudablesSeleccionPrueba struct {
	*ejecucionesSeleccionLlamamientoPrueba
	reanudaciones, ventanasOrden, reanudacionesSolicitud, ventanasSolicitud int
	denegada                                                                bool
	falloReanudacion                                                        error
}

func (e *ejecucionesReanudablesSeleccionPrueba) ReanudarPreparacionOrden(ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	e.reanudaciones++
	if e.denegada || ctx.Err() != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, ports.ErrAutorizacionDenegada
	}
	if e.falloReanudacion != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, e.falloReanudacion
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

func (e *ejecucionesReanudablesSeleccionPrueba) ReanudarSolicitudLlamamiento(ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	e.reanudacionesSolicitud++
	if e.denegada || ctx.Err() != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, ports.ErrAutorizacionDenegada
	}
	if e.falloReanudacion != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, e.falloReanudacion
	}
	e.Lock()
	defer e.Unlock()
	if solicitud != e.solicitud || e.situacion != ports.EjecucionSeleccionLlamamientoIndeterminada || e.efecto != ports.EfectoSolicitarSeleccionLlamamiento {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, errors.New("reanudación no aplicable")
	}
	e.reserva = "reserva:seleccion:solicitud-reanudada"
	e.situacion = ports.EjecucionSeleccionLlamamientoPropietaria
	return e.estado(true), nil
}

func (e *ejecucionesReanudablesSeleccionPrueba) AbrirVentanaEfecto(ctx context.Context, reserva ports.ReservaEjecucionSeleccionLlamamiento, efecto ports.EfectoSeleccionLlamamiento) error {
	if efecto == ports.EfectoPrepararOrdenSeleccionLlamamiento {
		e.ventanasOrden++
	}
	if efecto == ports.EfectoSolicitarSeleccionLlamamiento {
		e.ventanasSolicitud++
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
	reanudables := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones, denegada: true}
	e.servicio.ejecuciones = reanudables
	e.llamamientos.err = errors.New("resultado no recibido")
	_, _ = e.ejecutar(context.Background())
	if _, err := e.ejecutar(context.Background()); err == nil ||
		reanudables.reanudacionesSolicitud != 1 || reanudables.reanudaciones != 0 || e.ordenes.llamadas != 1 || e.llamamientos.llamadas != 1 {
		t.Fatalf("abrió otra fase no autorizada: %v", err)
	}
}

func TestSeleccionReanudacionAunOcupadaConservaClaveYPermiteReintentar(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	reanudables := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones}
	e.servicio.ejecuciones = reanudables
	e.llamamientos.err = errors.New("apertura pendiente")
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) {
		t.Fatalf("primera apertura: %v", err)
	}
	original := e.ejecuciones.solicitud
	e.llamamientos.err = nil
	reanudables.falloReanudacion = errors.New("arrendamiento aun vigente")
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) ||
		e.ejecuciones.solicitud != original || e.llamamientos.llamadas != 1 {
		t.Fatalf("el reintento temprano alteró el efecto: %v", err)
	}
	reanudables.falloReanudacion = nil
	if recibo, err := e.ejecutar(context.Background()); err != nil || !recibo.PropuestaGenerada ||
		e.ejecuciones.solicitud != original || e.llamamientos.creaciones != 1 {
		t.Fatalf("la misma clave no recuperó el llamamiento: %v", err)
	}
}

func TestSeleccionReanudaSolicitudSinReabrirVentanas(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	r := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones}
	e.servicio.ejecuciones = r
	e.llamamientos.err = errors.New("apertura no disponible")
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) {
		t.Fatal(err)
	}
	original := e.ejecuciones.solicitud
	e.llamamientos.err = nil
	recibo, err := e.ejecutar(context.Background())
	if err != nil || !recibo.PropuestaGenerada || r.reanudacionesSolicitud != 1 || r.reanudaciones != 0 || r.ventanasOrden != 1 || r.ventanasSolicitud != 1 || e.ejecuciones.solicitud != original || e.llamamientos.creaciones != 1 {
		t.Fatalf("recuperación distinta: %v", err)
	}
	replay, err := e.ejecutar(context.Background())
	if err != nil || replay != recibo || e.llamamientos.creaciones != 1 || r.reanudacionesSolicitud != 1 {
		t.Fatalf("replay distinto: %v", err)
	}
}

func TestSeleccionReanudadaConservaEfectoSolicitarEnErrores(t *testing.T) {
	for _, fallo := range []string{"orden", "contexto", "recibo", "cancelacion"} {
		t.Run(fallo, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			r := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones}
			e.servicio.ejecuciones = r
			e.llamamientos.err = errors.New("respuesta perdida")
			_, _ = e.ejecutar(context.Background())
			e.llamamientos.err = nil
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			switch fallo {
			case "orden":
				e.ordenes.err = errors.New("orden no disponible")
			case "contexto":
				e.preparador.fallarEn = "llamamiento"
			case "recibo":
				e.llamamientos.cruzarRecibo = true
			case "cancelacion":
				e.ordenes.cancelar = cancelar
			}
			if _, err := e.ejecutar(ctx); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) || e.ejecuciones.efecto != ports.EfectoSolicitarSeleccionLlamamiento || e.ejecuciones.situacion != ports.EjecucionSeleccionLlamamientoIndeterminada || r.ventanasOrden != 1 || r.ventanasSolicitud != 1 {
				t.Fatalf("perdió efecto solicitar: %v, %s", err, e.ejecuciones.efecto)
			}
		})
	}
}

func TestSeleccionReanudacionSolicitudRechazaCambioClaveOMaterial(t *testing.T) {
	for _, cambio := range []string{"clave", "material"} {
		t.Run(cambio, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			r := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones}
			e.servicio.ejecuciones = r
			e.llamamientos.err = errors.New("respuesta perdida")
			_, _ = e.ejecutar(context.Background())
			e.llamamientos.err = nil
			idOperacion := claveIdempotenciaSeleccion
			if cambio == "clave" {
				idOperacion = "018f47a2-6b31-4c80-8a95-4d2e707c5a22"
			} else {
				e.preparador.alternarPolitica = true
			}
			_, err := e.servicio.SeleccionarYLlamar(context.Background(), SolicitudSeleccionLlamamiento{ClaveIdempotencia: idOperacion})
			if err == nil || r.reanudacionesSolicitud != 0 || e.ordenes.llamadas != 1 || e.llamamientos.llamadas != 1 {
				t.Fatalf("cambio cruzó recuperación: %v", err)
			}
		})
	}
}

type llamamientoConfirmadoRespuestaPerdidaPrueba struct {
	*llamamientoSeleccionPrueba
	persistido ports.ReciboSolicitudLlamamientoBolsa
}

func (l *llamamientoConfirmadoRespuestaPerdidaPrueba) SolicitarLlamamiento(ctx context.Context, c ports.ComandoSolicitarLlamamientoBolsa) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	if l.persistido != (ports.ReciboSolicitudLlamamientoBolsa{}) {
		l.llamadas++
		return l.persistido, nil
	}
	r, err := l.llamamientoSeleccionPrueba.SolicitarLlamamiento(ctx, c)
	if err != nil {
		return r, err
	}
	l.persistido = r
	return ports.ReciboSolicitudLlamamientoBolsa{}, errors.New("respuesta perdida tras confirmar")
}
func TestSeleccionReanudacionRecuperaRespuestaPerdidaSinDuplicarBolsa(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	r := &ejecucionesReanudablesSeleccionPrueba{ejecucionesSeleccionLlamamientoPrueba: e.ejecuciones}
	l := &llamamientoConfirmadoRespuestaPerdidaPrueba{llamamientoSeleccionPrueba: e.llamamientos}
	e.servicio.ejecuciones = r
	e.servicio.llamamientos = l
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) {
		t.Fatal(err)
	}
	recibido, err := e.ejecutar(context.Background())
	if err != nil || recibido != l.persistido || e.llamamientos.creaciones != 1 || r.ventanasOrden != 1 || r.ventanasSolicitud != 1 {
		t.Fatalf("duplicó confirmación: %v", err)
	}
}

func TestSeleccionSinReanudadorSolicitudMantieneRechazoIndeterminada(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.llamamientos.err = errors.New("respuesta perdida")
	_, _ = e.ejecutar(context.Background())
	e.llamamientos.err = nil
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) || e.ordenes.llamadas != 1 || e.llamamientos.llamadas != 1 {
		t.Fatalf("composición no habilitada repitió efecto: %v", err)
	}
}
