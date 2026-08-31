package application

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	clavePeticionSeleccionPrueba  = "vec.contratacion-temporal.integracion-bolsa-peticion/v1"
	claveRespuestaSeleccionPrueba = "vec.contratacion-temporal.integracion-bolsa-respuesta/v1"
	claveIdempotenciaSeleccion    = "018f47a2-6b31-4c80-8a95-4d2e707c5a11"
)

func TestSeleccionLlamamientoOrquestaPrimeraPosicionEvaluable(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	recibo, err := e.ejecutar(context.Background())
	if err != nil {
		t.Fatalf("seleccionar y llamar: %v", err)
	}
	if !recibo.PropuestaGenerada || recibo.OrdenSeleccionado != 2 ||
		recibo.Necesidad != e.preparador.necesidad ||
		recibo.Bolsa != e.preparador.bolsa || recibo.Politica != e.preparador.politica ||
		recibo.LlamamientoRef != "llamamiento:seleccion:001" {
		t.Fatalf("recibo minimizado inesperado: %#v", recibo)
	}
	datos, err := e.llamamientos.ultimo.DatosEn(e.instante)
	if err != nil || datos.MaximaPosicionEvaluable != e.ordenes.total ||
		datos.TotalPosicionesOrden != e.ordenes.total {
		t.Fatalf("Bolsa no recibió el orden completo como límite evaluable: %+v err=%v", datos, err)
	}
	if e.disponibilidad.llamadas != 1 || e.ordenes.llamadas != 1 ||
		e.llamamientos.llamadas != 1 || e.llamamientos.creaciones != 1 {
		t.Fatalf("fronteras invocadas en orden incorrecto: disponibilidad=%d orden=%d llamamiento=%d creaciones=%d",
			e.disponibilidad.llamadas, e.ordenes.llamadas,
			e.llamamientos.llamadas, e.llamamientos.creaciones)
	}
	tipo := reflect.TypeOf(SolicitudSeleccionLlamamiento{})
	if tipo.NumField() != 1 || tipo.Field(0).Name != "ClaveIdempotencia" {
		t.Fatalf("la entrada permite elegir bolsa/candidato/posición: %v", tipo)
	}
}
func TestSeleccionLlamamientoFallaCerradoSinBolsaODisponibilidad(t *testing.T) {
	casos := []struct {
		nombre          string
		bolsaEncontrada bool
		disponible      bool
		cantidad        uint32
		cantidadExacta  bool
	}{
		{"sin bolsa", false, false, 0, true},
		{"sin disponibilidad", true, false, 0, true},
		{"cantidad inexacta", true, true, 1, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			e.disponibilidad.bolsaEncontrada = caso.bolsaEncontrada
			e.disponibilidad.disponible = caso.disponible
			e.disponibilidad.cantidad = caso.cantidad
			e.disponibilidad.cantidadExacta = caso.cantidadExacta
			_, err := e.ejecutar(context.Background())
			if !errors.Is(err, ErrSeleccionLlamamientoNoDisponible) ||
				e.ordenes.llamadas != 0 || e.llamamientos.llamadas != 0 {
				t.Fatalf("la ausencia avanzó al efecto: orden=%d llamamiento=%d err=%v",
					e.ordenes.llamadas, e.llamamientos.llamadas, err)
			}
		})
	}
}
func TestSeleccionLlamamientoRechazaOrdenAusenteOIncompleto(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		generada bool
		completa bool
		total    uint32
	}{
		{"ausente", false, false, 0},
		{"incompleto", true, false, 3},
		{"sin posiciones", true, true, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			e.ordenes.generada, e.ordenes.completa, e.ordenes.total =
				caso.generada, caso.completa, caso.total
			_, err := e.ejecutar(context.Background())
			if !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) ||
				e.llamamientos.llamadas != 0 {
				t.Fatalf("orden no completa produjo llamamiento: llamadas=%d err=%v",
					e.llamamientos.llamadas, err)
			}
		})
	}
}
func TestSeleccionLlamamientoRechazaEvidenciaCruzada(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.disponibilidad.sello = selloSeleccionPrueba('2')
	_, err := e.ejecutar(context.Background())
	if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) ||
		e.ordenes.llamadas != 0 || e.llamamientos.llamadas != 0 {
		t.Fatalf("evidencia de orden cruzada con disponibilidad avanzó: err=%v", err)
	}
}
func TestSeleccionLlamamientoReplayExactoUsaIdempotenciaDelPuerto(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	solicitud := SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion}
	primero, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primer intento: %v", err)
	}
	e.disponibilidad.cantidad = 1
	e.preparador.alternarPolitica = true
	segundo, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if primero != segundo || e.preparador.consultasPreparadas != 1 ||
		e.disponibilidad.llamadas != 1 || e.llamamientos.llamadas != 1 || e.ordenes.llamadas != 1 ||
		e.llamamientos.creaciones != 1 {
		t.Fatalf("replay consultó o repitió efectos: igual=%v preparadas=%d disponibilidad=%d ordenes=%d llamadas=%d creaciones=%d",
			primero == segundo, e.preparador.consultasPreparadas, e.disponibilidad.llamadas,
			e.ordenes.llamadas, e.llamamientos.llamadas, e.llamamientos.creaciones)
	}
}
func TestSeleccionLlamamientoColisionFallaSinSegundoEfecto(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.ordenes.iniciada, e.ordenes.continuar = make(chan struct{}), make(chan struct{})
	solicitud := SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion}
	terminada := make(chan error, 1)
	go func() { _, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud); terminada <- err }()
	<-e.ordenes.iniciada
	e.preparador.alternarPolitica = true
	_, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
	if !errors.Is(err, ErrClaveSeleccionLlamamientoEnColision) ||
		e.llamamientos.creaciones != 0 || strings.Contains(err.Error(), "detalle privado") {
		t.Fatalf("colisión no quedó cerrada y opaca: creaciones=%d err=%v",
			e.llamamientos.creaciones, err)
	}
	close(e.ordenes.continuar)
	if err := <-terminada; err != nil {
		t.Fatalf("la ejecución original falló: %v", err)
	}
}
func TestSeleccionLlamamientoTerminalesHostilesFallanCerrados(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*ports.EstadoEjecucionSeleccionLlamamiento)
	}{
		{"referencias cruzadas", func(e *ports.EstadoEjecucionSeleccionLlamamiento) {
			e.ReciboConfirmado.Bolsa = referenciaSeleccionPrueba("bolsa:ajena:001", '1')
		}},
		{"seudonimo fabricado", func(e *ports.EstadoEjecucionSeleccionLlamamiento) {
			e.ReciboConfirmado.SeleccionRef = ports.SeudonimoSeleccionBolsa{}
		}},
		{"limites alterados", func(e *ports.EstadoEjecucionSeleccionLlamamiento) {
			e.ArtefactoConfirmado.Comando.MaximaPosicionEvaluable = 1
		}},
		{"evidencia cruzada", func(e *ports.EstadoEjecucionSeleccionLlamamiento) {
			e.ArtefactoConfirmado.Evidencia.HuellaRespuestaSHA256 = strings.Repeat("e", 64)
		}},
		{"efecto terminal falso", func(e *ports.EstadoEjecucionSeleccionLlamamiento) {
			e.EfectoPosible = ports.EfectoPrepararOrdenSeleccionLlamamiento
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			if _, err := e.ejecutar(context.Background()); err != nil {
				t.Fatalf("confirmar escenario: %v", err)
			}
			estado := e.ejecuciones.estado(false)
			caso.mutar(&estado)
			e.ejecuciones.terminalForzado = &estado
			antes := [4]int{e.preparador.consultasPreparadas, e.disponibilidad.llamadas, e.ordenes.llamadas, e.llamamientos.llamadas}
			_, err := e.ejecutar(context.Background())
			despues := [4]int{e.preparador.consultasPreparadas, e.disponibilidad.llamadas, e.ordenes.llamadas, e.llamamientos.llamadas}
			if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) || antes != despues {
				t.Fatalf("terminal hostil avanzó: antes=%v después=%v err=%v", antes, despues, err)
			}
		})
	}
}
func TestSeleccionLlamamientoLigaCantidadExactaAntesDelSiguienteEfecto(t *testing.T) {
	casos := []struct {
		nombre           string
		configurar       func(*escenarioSeleccionLlamamientoPrueba)
		errorEsperado    error
		ordenesEsperadas int
	}{
		{"maximo no cubre disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba) {
			e.preparador.maximoResultados, e.preparador.maximoPosiciones = 4, 2
			e.disponibilidad.cantidad = 3
		}, ErrResultadoSeleccionLlamamientoNoConfiable, 0},
		{"recibo no cubre disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba) {
			e.ordenes.total = 2
		}, ErrEjecucionSeleccionLlamamientoIndeterminada, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			caso.configurar(&e)
			_, err := e.ejecutar(context.Background())
			if !errors.Is(err, caso.errorEsperado) ||
				e.ordenes.llamadas != caso.ordenesEsperadas || e.llamamientos.llamadas != 0 {
				t.Fatalf("cantidad insuficiente avanzó: orden=%d llamada=%d err=%v",
					e.ordenes.llamadas, e.llamamientos.llamadas, err)
			}
		})
	}
}
func TestSeleccionLlamamientoConcurrenciaFallaDiferenciada(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.ordenes.iniciada, e.ordenes.continuar = make(chan struct{}), make(chan struct{})
	solicitud := SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion}
	terminada := make(chan error, 1)
	go func() {
		_, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
		terminada <- err
	}()
	<-e.ordenes.iniciada
	_, err := e.servicio.SeleccionarYLlamar(context.Background(), solicitud)
	if !errors.Is(err, ErrEjecucionSeleccionLlamamientoConcurrente) ||
		e.ordenes.llamadas != 1 || e.llamamientos.llamadas != 0 {
		t.Fatalf("concurrencia no quedó ocupada: orden=%d llamada=%d err=%v",
			e.ordenes.llamadas, e.llamamientos.llamadas, err)
	}
	close(e.ordenes.continuar)
	if err := <-terminada; err != nil {
		t.Fatalf("la ejecución propietaria falló: %v", err)
	}
}
func TestSeleccionLlamamientoRechazaLimiteDependenciasYEntrada(t *testing.T) {
	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.preparador.maximoResultados = ports.MaximoElementosIntegracionBolsa + 1
	_, err := e.ejecutar(context.Background())
	if !errors.Is(err, ErrResultadoSeleccionLlamamientoNoConfiable) ||
		e.disponibilidad.llamadas != 0 || e.llamamientos.llamadas != 0 {
		t.Fatalf("límite inválido alcanzó conectores: %v", err)
	}
	if _, err := NuevoServicioSeleccionLlamamiento(e.preparador, (*ejecucionesSeleccionLlamamientoPrueba)(nil), e.disponibilidad, e.ordenes, e.llamamientos, e.servicio.verificador, e.servicio.reloj); !errors.Is(err, ErrServicioSeleccionLlamamientoInvalido) {
		t.Fatalf("registro de ejecución inválido aceptado: %v", err)
	}
	_, err = e.servicio.SeleccionarYLlamar(context.Background(), SolicitudSeleccionLlamamiento{ClaveIdempotencia: "candidato:42"})
	if !errors.Is(err, ErrSolicitudSeleccionLlamamientoInvalida) {
		t.Fatalf("UUID no canónica aceptada: %v", err)
	}
}
func TestSeleccionLlamamientoCancelaAntesDeCadaFronteraSiguiente(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*escenarioSeleccionLlamamientoPrueba, context.CancelFunc)
		llamadas [3]int
	}{
		{"tras preparar disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) {
			e.preparador.cancelarEn, e.preparador.cancelar = "consulta", c
		}, [3]int{}},
		{"tras consultar disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) { e.disponibilidad.cancelar = c }, [3]int{1, 0, 0}},
		{"tras preparar orden", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) {
			e.preparador.cancelarEn, e.preparador.cancelar = "orden", c
		}, [3]int{1, 0, 0}},
		{"tras recibir orden", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) { e.ordenes.cancelar = c }, [3]int{1, 1, 0}},
		{"tras preparar llamamiento", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) {
			e.preparador.cancelarEn, e.preparador.cancelar = "llamamiento", c
		}, [3]int{1, 1, 0}},
		{"tras solicitar llamamiento", func(e *escenarioSeleccionLlamamientoPrueba, c context.CancelFunc) { e.llamamientos.cancelar = c }, [3]int{1, 1, 1}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			ctx, cancelar := context.WithCancel(context.Background())
			caso.preparar(&e, cancelar)
			_, err := e.ejecutar(ctx)
			obtenidas := [3]int{e.disponibilidad.llamadas, e.ordenes.llamadas, e.llamamientos.llamadas}
			if !errors.Is(err, context.Canceled) || obtenidas != caso.llamadas ||
				errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) != (caso.llamadas[1] > 0) {
				t.Fatalf("cancelación avanzó: llamadas=%v err=%v", obtenidas, err)
			}
		})
	}

	e := nuevoEscenarioSeleccionLlamamiento(t)
	e.disponibilidad.err = context.DeadlineExceeded
	if _, err := e.ejecutar(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("la frontera perdió la causa de cancelación: %v", err)
	}
}
func TestSeleccionLlamamientoCierraFallosDeTodasLasFronteras(t *testing.T) {
	casos := []struct {
		nombre     string
		configurar func(*escenarioSeleccionLlamamientoPrueba)
		llamadas   [3]int
		esperado   error
	}{
		{"preparar disponibilidad", func(e *escenarioSeleccionLlamamientoPrueba) { e.preparador.fallarEn = "consulta" }, [3]int{}, ErrSeleccionLlamamientoNoDisponible},
		{"preparar orden", func(e *escenarioSeleccionLlamamientoPrueba) { e.preparador.fallarEn = "orden" }, [3]int{1, 0, 0}, ErrSeleccionLlamamientoNoDisponible},
		{"orden", func(e *escenarioSeleccionLlamamientoPrueba) { e.ordenes.err = errors.New("detalle privado") }, [3]int{1, 1, 0}, ErrEjecucionSeleccionLlamamientoIndeterminada},
		{"preparar llamamiento", func(e *escenarioSeleccionLlamamientoPrueba) { e.preparador.fallarEn = "llamamiento" }, [3]int{1, 1, 0}, ErrEjecucionSeleccionLlamamientoIndeterminada},
		{"llamamiento", func(e *escenarioSeleccionLlamamientoPrueba) { e.llamamientos.err = errors.New("detalle privado") }, [3]int{1, 1, 1}, ErrEjecucionSeleccionLlamamientoIndeterminada},
		{"recibo cruzado", func(e *escenarioSeleccionLlamamientoPrueba) { e.llamamientos.cruzarRecibo = true }, [3]int{1, 1, 1}, ErrEjecucionSeleccionLlamamientoIndeterminada},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			caso.configurar(&e)
			_, err := e.ejecutar(context.Background())
			obtenidas := [3]int{e.disponibilidad.llamadas, e.ordenes.llamadas, e.llamamientos.llamadas}
			if !errors.Is(err, caso.esperado) || obtenidas != caso.llamadas || strings.Contains(err.Error(), "detalle") {
				t.Fatalf("fallo no quedó cerrado: llamadas=%v err=%v", obtenidas, err)
			}
			if errors.Is(caso.esperado, ErrEjecucionSeleccionLlamamientoIndeterminada) {
				_, repetido := e.ejecutar(context.Background())
				esperadas := [3]int{caso.llamadas[0] + 1, caso.llamadas[1], caso.llamadas[2]}
				if actuales := [3]int{e.disponibilidad.llamadas, e.ordenes.llamadas, e.llamamientos.llamadas}; !errors.Is(repetido, ErrEjecucionSeleccionLlamamientoIndeterminada) || actuales != esperadas {
					t.Fatalf("estado indeterminado reejecutado: llamadas=%v err=%v", actuales, repetido)
				}
			}
		})
	}
}
func TestSeleccionLlamamientoRechazaContextosDesligadosYRelojRetrogrado(t *testing.T) {
	casos := []struct {
		nombre, operacion string
		mutar             func(*ports.DatosContextoPeticionIntegracionBolsa)
		ordenes           int
	}{
		{"organizacion de otra orden", "operacion:orden:001", func(d *ports.DatosContextoPeticionIntegracionBolsa) { d.OrganizacionRef = "organizacion:ajena" }, 0},
		{"correlacion de otro llamamiento", "operacion:llamamiento:001", func(d *ports.DatosContextoPeticionIntegracionBolsa) { d.CorrelacionRef = "correlacion:ajena" }, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeleccionLlamamiento(t)
			e.preparador.mutarContexto = func(operacion string, datos *ports.DatosContextoPeticionIntegracionBolsa) {
				if operacion == caso.operacion {
					caso.mutar(datos)
				}
			}
			_, err := e.ejecutar(context.Background())
			esperado := ErrResultadoSeleccionLlamamientoNoConfiable
			if caso.ordenes != 0 {
				esperado = ErrEjecucionSeleccionLlamamientoIndeterminada
			}
			if !errors.Is(err, esperado) ||
				e.ordenes.llamadas != caso.ordenes || e.llamamientos.llamadas != 0 {
				t.Fatalf("contextos desligados avanzaron: orden=%d llamada=%d err=%v", e.ordenes.llamadas, e.llamamientos.llamadas, err)
			}
		})
	}

	base := time.Date(2026, 8, 22, 10, 3, 0, 0, time.UTC)
	for _, secuencia := range [][]time.Time{
		{base, base.Add(-time.Minute)},
		{base, base, base.Add(time.Minute), base},
		{base, base, base, base, base.Add(time.Minute), base},
	} {
		e := nuevoEscenarioSeleccionLlamamiento(t)
		e.servicio.reloj = &relojSecuenciaSeleccionPrueba{instantes: secuencia}
		_, err := e.ejecutar(context.Background())
		esperado := ErrResultadoSeleccionLlamamientoNoConfiable
		if len(secuencia) >= 4 {
			esperado = ErrEjecucionSeleccionLlamamientoIndeterminada
		}
		if !errors.Is(err, esperado) {
			t.Fatalf("reloj retrógrado aceptado (%d lecturas): %v", len(secuencia), err)
		}
	}
}

type escenarioSeleccionLlamamientoPrueba struct {
	servicio       *ServicioSeleccionLlamamiento
	preparador     *preparadorSeleccionLlamamientoPrueba
	ejecuciones    *ejecucionesSeleccionLlamamientoPrueba
	disponibilidad *disponibilidadSeleccionPrueba
	ordenes        *ordenSeleccionPrueba
	llamamientos   *llamamientoSeleccionPrueba
	instante       time.Time
}

func (e *escenarioSeleccionLlamamientoPrueba) ejecutar(ctx context.Context) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	return e.servicio.SeleccionarYLlamar(ctx, SolicitudSeleccionLlamamiento{ClaveIdempotencia: claveIdempotenciaSeleccion})
}

type ejecucionesSeleccionLlamamientoPrueba struct {
	sync.Mutex
	solicitud       ports.SolicitudReservaEjecucionSeleccionLlamamiento
	reserva         string
	efecto          ports.EfectoSeleccionLlamamiento
	situacion       ports.SituacionEjecucionSeleccionLlamamiento
	recibo          ports.ReciboSolicitudLlamamientoBolsa
	artefacto       ports.ArtefactoProbatorioLlamamientoBolsa
	terminalForzado *ports.EstadoEjecucionSeleccionLlamamiento
}

func (e *ejecucionesSeleccionLlamamientoPrueba) ResolverTerminal(ctx context.Context, clave string) (ports.EstadoEjecucionSeleccionLlamamiento, bool, error) {
	if err := ctx.Err(); err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, false, err
	}
	e.Lock()
	defer e.Unlock()
	if e.terminalForzado != nil {
		return *e.terminalForzado, true, nil
	}
	if e.situacion != ports.EjecucionSeleccionLlamamientoConfirmada ||
		e.solicitud.ClaveIdempotencia != clave {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, false, nil
	}
	return e.estado(false), true, nil
}
func (e *ejecucionesSeleccionLlamamientoPrueba) Reservar(ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	if err := ctx.Err(); err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, err
	}
	e.Lock()
	defer e.Unlock()
	if e.situacion == "" {
		e.solicitud, e.reserva = solicitud, "reserva:seleccion:001"
		e.situacion = ports.EjecucionSeleccionLlamamientoPropietaria
		return e.estado(true), nil
	}
	if e.solicitud != solicitud {
		return ports.EstadoEjecucionSeleccionLlamamiento{
			Solicitud: solicitud, Situacion: ports.EjecucionSeleccionLlamamientoColision,
		}, nil
	}
	return e.estado(false), nil
}
func (e *ejecucionesSeleccionLlamamientoPrueba) estado(propietaria bool) ports.EstadoEjecucionSeleccionLlamamiento {
	estado := ports.EstadoEjecucionSeleccionLlamamiento{Solicitud: e.solicitud, Situacion: e.situacion, EfectoPosible: e.efecto}
	if e.situacion == ports.EjecucionSeleccionLlamamientoConfirmada {
		estado.ReciboConfirmado = e.recibo
		estado.ArtefactoConfirmado = e.artefacto
	}
	if e.situacion == ports.EjecucionSeleccionLlamamientoPropietaria {
		if propietaria {
			estado.ReservaRef = e.reserva
		} else {
			estado.Situacion = ports.EjecucionSeleccionLlamamientoOcupada
		}
	}
	return estado
}
func (e *ejecucionesSeleccionLlamamientoPrueba) AbrirVentanaEfecto(ctx context.Context, reserva ports.ReservaEjecucionSeleccionLlamamiento, efecto ports.EfectoSeleccionLlamamiento) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.Lock()
	defer e.Unlock()
	if e.reserva != reserva.ReservaRef || e.solicitud != reserva.Solicitud ||
		e.situacion != ports.EjecucionSeleccionLlamamientoPropietaria {
		return errors.New("reserva ajena")
	}
	e.efecto = efecto
	return nil
}
func (e *ejecucionesSeleccionLlamamientoPrueba) MarcarIndeterminada(_ context.Context, reserva ports.ReservaEjecucionSeleccionLlamamiento, efecto ports.EfectoSeleccionLlamamiento) error {
	e.Lock()
	defer e.Unlock()
	if e.reserva != reserva.ReservaRef {
		return errors.New("reserva ausente")
	}
	e.situacion, e.efecto = ports.EjecucionSeleccionLlamamientoIndeterminada, efecto
	return nil
}
func (e *ejecucionesSeleccionLlamamientoPrueba) LiberarAntesDeEfectos(_ context.Context, reserva ports.ReservaEjecucionSeleccionLlamamiento) error {
	e.Lock()
	defer e.Unlock()
	if e.reserva != reserva.ReservaRef || e.efecto != "" {
		return errors.New("reserva no liberable")
	}
	e.solicitud, e.reserva, e.situacion = ports.SolicitudReservaEjecucionSeleccionLlamamiento{}, "", ""
	return nil
}
func (e *ejecucionesSeleccionLlamamientoPrueba) Confirmar(_ context.Context, reserva ports.ReservaEjecucionSeleccionLlamamiento, recibo ports.ReciboSolicitudLlamamientoBolsa, artefacto ports.ArtefactoProbatorioLlamamientoBolsa) error {
	e.Lock()
	defer e.Unlock()
	if e.reserva != reserva.ReservaRef ||
		e.efecto != ports.EfectoSolicitarSeleccionLlamamiento {
		return errors.New("reserva no confirmable")
	}
	e.situacion, e.efecto, e.recibo, e.artefacto = ports.EjecucionSeleccionLlamamientoConfirmada, "", recibo, artefacto
	return nil
}
func (e *ejecucionesSeleccionLlamamientoPrueba) ConsultarEstado(ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	if err := ctx.Err(); err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, err
	}
	e.Lock()
	defer e.Unlock()
	if e.situacion == "" || e.solicitud != solicitud {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, errors.New("ejecucion ausente")
	}
	return e.estado(false), nil
}
func nuevoEscenarioSeleccionLlamamiento(t *testing.T) escenarioSeleccionLlamamientoPrueba {
	t.Helper()
	base := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	emisor, err := ports.NuevoEmisorContextoPeticionIntegracionBolsa("autoridad:contratacion-temporal", clavePeticionSeleccionPrueba, selladorContextoSeleccionPrueba{})
	if err != nil {
		t.Fatalf("crear emisor: %v", err)
	}
	preparador := &preparadorSeleccionLlamamientoPrueba{base: base, emisor: emisor, necesidad: referenciaSeleccionPrueba("necesidad:cobertura:001", 'a'), bolsa: referenciaSeleccionPrueba("bolsa:vigente:001", 'b'), politica: referenciaSeleccionPrueba("politica:llamamiento:001", 'c'), maximoResultados: 3, maximoPosiciones: 3}
	disponibilidad := &disponibilidadSeleccionPrueba{
		base: base, bolsa: preparador.bolsa, bolsaEncontrada: true, disponible: true,
		cantidad: 3, cantidadExacta: true, sello: selloSeleccionPrueba('1'),
	}
	ordenes := &ordenSeleccionPrueba{base: base, generada: true, completa: true, total: 3}
	llamamientos := &llamamientoSeleccionPrueba{base: base}
	ejecuciones := &ejecucionesSeleccionLlamamientoPrueba{}
	verificador, err := ports.NuevoVerificadorEvidenciaIntegracionBolsa("autoridad:bolsa", claveRespuestaSeleccionPrueba, nil, verificadorEvidenciaSeleccionPrueba{})
	if err != nil {
		t.Fatalf("crear verificador: %v", err)
	}
	instante := base.Add(3 * time.Minute)
	servicio, err := NuevoServicioSeleccionLlamamiento(preparador, ejecuciones, disponibilidad, ordenes, llamamientos, verificador, relojSeleccionLlamamientoPrueba{instante: instante})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return escenarioSeleccionLlamamientoPrueba{servicio: servicio, preparador: preparador, ejecuciones: ejecuciones, disponibilidad: disponibilidad, ordenes: ordenes, llamamientos: llamamientos, instante: instante}
}

type relojSeleccionLlamamientoPrueba struct{ instante time.Time }

func (r relojSeleccionLlamamientoPrueba) Ahora() time.Time { return r.instante }

type relojSecuenciaSeleccionPrueba struct {
	instantes []time.Time
	indice    int
}

func (r *relojSecuenciaSeleccionPrueba) Ahora() time.Time {
	if r.indice >= len(r.instantes) {
		return r.instantes[len(r.instantes)-1]
	}
	instante := r.instantes[r.indice]
	r.indice++
	return instante
}

type selladorContextoSeleccionPrueba struct{}

func (selladorContextoSeleccionPrueba) SellarDatos(context.Context, []byte) (string, error) {
	return "hmac-sha256:" + clavePeticionSeleccionPrueba + ":" + strings.Repeat("a", 64), nil
}

type verificadorEvidenciaSeleccionPrueba struct{}

func (verificadorEvidenciaSeleccionPrueba) VerificarDatos(ctx context.Context, clave string, material []byte, sello string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	esperado := byte(0)
	switch {
	case bytes.Contains(material, []byte("disponibilidad_volatil")):
		esperado = '1'
	case bytes.Contains(material, []byte("recibo_orden")):
		esperado = '2'
	case bytes.Contains(material, []byte("recibo_llamamiento")):
		esperado = '3'
	}
	if clave != claveRespuestaSeleccionPrueba || esperado == 0 ||
		sello != selloSeleccionPrueba(esperado) {
		return errors.New("detalle privado: evidencia cruzada")
	}
	return nil
}

type preparadorSeleccionLlamamientoPrueba struct {
	base                                   time.Time
	emisor                                 *ports.EmisorContextoPeticionIntegracionBolsa
	necesidad, bolsa, politica             ports.ReferenciaVersionadaIntegracionBolsa
	maximoResultados, maximoPosiciones     uint32
	consultasPreparadas, ordenesPreparadas int
	alternarPolitica                       bool
	cancelarEn, fallarEn                   string
	cancelar                               context.CancelFunc
	mutarContexto                          func(string, *ports.DatosContextoPeticionIntegracionBolsa)
}

func (p *preparadorSeleccionLlamamientoPrueba) PrepararConsultaDisponibilidad(ctx context.Context, _ string) (ports.SolicitudDisponibilidadBolsa, error) {
	p.consultasPreparadas++
	contexto, err := p.contexto(
		ctx, "operacion:disponibilidad:001", p.necesidad,
		referenciaSeleccionPrueba("accion:disponibilidad:001", 'd'),
	)
	if p.cancelarEn == "consulta" && p.cancelar != nil {
		p.cancelar()
	}
	if p.fallarEn == "consulta" {
		err = errors.New("detalle privado")
	}
	return ports.SolicitudDisponibilidadBolsa{
		Contexto: contexto, Necesidad: p.necesidad, CategoriaRef: "categoria:auxiliar",
		MaximoResultados: p.maximoResultados,
	}, err
}
func (p *preparadorSeleccionLlamamientoPrueba) PrepararOrdenCompleto(ctx context.Context, _ string, resultado ports.ResultadoDisponibilidadBolsa) (ports.ComandoPrepararOrdenBolsa, error) {
	p.ordenesPreparadas++
	politica := p.politica
	if p.alternarPolitica && p.ordenesPreparadas > 1 {
		politica = referenciaSeleccionPrueba("politica:llamamiento:002", '9')
	}
	contexto, err := p.contexto(
		ctx, "operacion:orden:001", resultado.Bolsa,
		referenciaSeleccionPrueba("accion:orden:001", 'e'),
	)
	if p.cancelarEn == "orden" && p.cancelar != nil {
		p.cancelar()
	}
	if p.fallarEn == "orden" {
		err = errors.New("detalle privado")
	}
	return ports.ComandoPrepararOrdenBolsa{
		Contexto: contexto, Necesidad: resultado.Necesidad, Bolsa: resultado.Bolsa,
		Politica: politica, MaximoPosiciones: p.maximoPosiciones,
	}, err
}
func (p *preparadorSeleccionLlamamientoPrueba) PrepararContextoLlamamiento(ctx context.Context, _ string, recibo ports.ReciboOrdenBolsa) (ports.ContextoPeticionIntegracionBolsa, error) {
	contexto, err := p.contexto(ctx, "operacion:llamamiento:001", recibo.Orden, recibo.AccionLlamamiento)
	if p.cancelarEn == "llamamiento" && p.cancelar != nil {
		p.cancelar()
	}
	if p.fallarEn == "llamamiento" {
		err = errors.New("detalle privado")
	}
	return contexto, err
}
func (p *preparadorSeleccionLlamamientoPrueba) contexto(ctx context.Context, operacion string, recurso ports.ReferenciaVersionadaIntegracionBolsa, accion ports.ReferenciaVersionadaIntegracionBolsa) (ports.ContextoPeticionIntegracionBolsa, error) {
	datos := ports.DatosContextoPeticionIntegracionBolsa{
		OperacionRef: operacion, OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef: "expediente:temporal:001", VersionExpediente: 7,
		CorrelacionRef:       "correlacion:seleccion:001",
		ContratoVersion:      ports.VersionContratoIntegracionBolsa,
		AutoridadSolicitante: "autoridad:contratacion-temporal",
		Autorizacion:         referenciaSeleccionPrueba("autorizacion:seleccion:001", 'f'),
		Accion:               accion, Recurso: recurso,
		Finalidad:    referenciaSeleccionPrueba("finalidad:seleccion:001", '6'),
		SolicitadaEn: p.base, ValidaHasta: p.base.Add(10 * time.Minute),
	}
	if p.mutarContexto != nil {
		p.mutarContexto(operacion, &datos)
	}
	return p.emisor.Emitir(ctx, datos, p.base)
}

type disponibilidadSeleccionPrueba struct {
	base                                        time.Time
	bolsa                                       ports.ReferenciaVersionadaIntegracionBolsa
	bolsaEncontrada, disponible, cantidadExacta bool
	cantidad                                    uint32
	sello                                       string
	cancelar                                    context.CancelFunc
	err                                         error
	llamadas                                    int
}

func (d *disponibilidadSeleccionPrueba) ConsultarDisponibilidad(ctx context.Context, solicitud ports.SolicitudDisponibilidadBolsa) (ports.ResultadoDisponibilidadBolsa, error) {
	d.llamadas++
	if d.err != nil {
		return ports.ResultadoDisponibilidadBolsa{}, d.err
	}
	datos, _ := solicitud.Contexto.DatosEn(d.base.Add(3 * time.Minute))
	resultado := ports.ResultadoDisponibilidadBolsa{
		OperacionRef: datos.OperacionRef, OrganizacionRef: datos.OrganizacionRef,
		ExpedienteRef: datos.ExpedienteRef, VersionExpediente: datos.VersionExpediente,
		CorrelacionRef: datos.CorrelacionRef, Necesidad: solicitud.Necesidad,
		CategoriaRef:    solicitud.CategoriaRef,
		Resultado:       referenciaSeleccionPrueba("resultado:disponibilidad:001", '7'),
		BolsaEncontrada: d.bolsaEncontrada, Disponible: d.disponible,
		CantidadDisponible: d.cantidad, CantidadExacta: d.cantidadExacta,
		Procedencia: procedenciaSeleccionPrueba(d.base, "respuesta:disponibilidad:001", "evidencia:disponibilidad:001", d.sello),
	}
	if d.bolsaEncontrada {
		resultado.Bolsa = d.bolsa
	}
	if d.cancelar != nil {
		d.cancelar()
	}
	return resultado, nil
}

type ordenSeleccionPrueba struct {
	base                time.Time
	generada, completa  bool
	total               uint32
	cancelar            context.CancelFunc
	err                 error
	llamadas            int
	iniciada, continuar chan struct{}
}

func (o *ordenSeleccionPrueba) PrepararOrden(ctx context.Context, comando ports.ComandoPrepararOrdenBolsa) (ports.ReciboOrdenBolsa, error) {
	o.llamadas++
	if o.iniciada != nil {
		close(o.iniciada)
		select {
		case <-o.continuar:
		case <-ctx.Done():
			return ports.ReciboOrdenBolsa{}, ctx.Err()
		}
	}
	if o.err != nil {
		return ports.ReciboOrdenBolsa{}, o.err
	}
	datos, _ := comando.Contexto.DatosEn(o.base.Add(3 * time.Minute))
	recibo := ports.ReciboOrdenBolsa{
		OperacionRef: datos.OperacionRef, OrganizacionRef: datos.OrganizacionRef,
		ExpedienteRef: datos.ExpedienteRef, VersionExpediente: datos.VersionExpediente,
		CorrelacionRef: datos.CorrelacionRef, Necesidad: comando.Necesidad,
		Bolsa: comando.Bolsa, Politica: comando.Politica,
		Resultado:     referenciaSeleccionPrueba("resultado:orden:001", '7'),
		OrdenGenerada: o.generada, OrdenCompleta: o.completa,
		ReciboRef: "recibo:orden:001", AuditoriaRef: "auditoria:orden:001",
		EventoRef: "evento:orden:001", ConfirmadaEn: o.base.Add(time.Minute),
		Procedencia: procedenciaSeleccionPrueba(o.base, "respuesta:orden:001", "evidencia:orden:001", selloSeleccionPrueba('2')),
	}
	if o.generada {
		recibo.Orden = referenciaSeleccionPrueba("orden:bolsa:001", '8')
		recibo.AccionLlamamiento = referenciaSeleccionPrueba("accion:llamamiento:001", '5')
		recibo.TotalPosiciones = o.total
	}
	if o.cancelar != nil {
		o.cancelar()
	}
	return recibo, nil
}

type llamamientoSeleccionPrueba struct {
	base                 time.Time
	llamadas, creaciones int
	cruzarRecibo         bool
	cancelar             context.CancelFunc
	err                  error
	ultimo               ports.ComandoSolicitarLlamamientoBolsa
}

func (l *llamamientoSeleccionPrueba) SolicitarLlamamiento(_ context.Context, comando ports.ComandoSolicitarLlamamientoBolsa) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	l.llamadas++
	l.ultimo = comando
	if l.err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, l.err
	}
	datos, err := comando.DatosEn(l.base.Add(3 * time.Minute))
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	contexto, _ := datos.Contexto.DatosEn(l.base.Add(3 * time.Minute))
	seudonimo, _ := ports.NuevoSeudonimoSeleccionBolsa(
		"hmac-sha256:vec.contratacion-temporal.seleccion/v1:" + strings.Repeat("9", 64),
	)
	recibo := ports.ReciboSolicitudLlamamientoBolsa{
		OperacionRef: contexto.OperacionRef, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: contexto.ExpedienteRef, VersionExpediente: contexto.VersionExpediente,
		CorrelacionRef: contexto.CorrelacionRef, Necesidad: datos.Necesidad,
		Bolsa: datos.Bolsa, Orden: datos.Orden, Politica: datos.Politica,
		Resultado:         referenciaSeleccionPrueba("resultado:llamamiento:001", '7'),
		PropuestaGenerada: true,
		Propuesta:         referenciaSeleccionPrueba("propuesta:llamamiento:001", '4'),
		AccionEvento:      referenciaSeleccionPrueba("accion:evento:001", '3'),
		LlamamientoRef:    "llamamiento:seleccion:001", SeleccionRef: seudonimo,
		RetencionSeleccion: referenciaSeleccionPrueba("retencion:seleccion:001", '2'),
		OrdenSeleccionado:  2, ReciboRef: "recibo:llamamiento:001",
		AuditoriaRef: "auditoria:llamamiento:001", EventoRef: "evento:llamamiento:001",
		ConfirmadaEn: l.base.Add(time.Minute),
		Procedencia:  procedenciaSeleccionPrueba(l.base, "respuesta:llamamiento:001", "evidencia:llamamiento:001", selloSeleccionPrueba('3')),
	}
	if l.cruzarRecibo {
		recibo.Orden = referenciaSeleccionPrueba("orden:bolsa:ajena", '1')
	}
	if l.cancelar != nil {
		l.cancelar()
	}
	l.creaciones++
	return recibo, nil
}
func referenciaSeleccionPrueba(referencia string, caracter byte) ports.ReferenciaVersionadaIntegracionBolsa {
	return ports.ReferenciaVersionadaIntegracionBolsa{
		Referencia: referencia, Version: 1, HuellaSHA256: strings.Repeat(string(caracter), 64),
	}
}
func procedenciaSeleccionPrueba(base time.Time, respuestaRef string, evidenciaRef string, sello string) ports.ProcedenciaIntegracionBolsa {
	return ports.ProcedenciaIntegracionBolsa{
		AutoridadRef: "autoridad:bolsa", RespuestaRef: respuestaRef,
		ContratoVersion: ports.VersionContratoIntegracionBolsa,
		Fuente:          referenciaSeleccionPrueba("fuente:bolsa:001", 'd'),
		Evidencia: ports.EvidenciaNominalIntegracionBolsa{
			EvidenciaRef: evidenciaRef, ClaveVerificacionRef: claveRespuestaSeleccionPrueba,
			SelloHMAC: sello, EmitidaEn: base.Add(2 * time.Minute),
			ValidaHasta: base.Add(8 * time.Minute), RetenerHasta: base.Add(24 * time.Hour),
		},
	}
}
func selloSeleccionPrueba(caracter byte) string {
	return "hmac-sha256:" + claveRespuestaSeleccionPrueba + ":" + strings.Repeat(string(caracter), 64)
}
