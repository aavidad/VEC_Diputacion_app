package application

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type estadoRegistroGINPIXPrueba uint8

const (
	estadoRegistroGINPIXReintentable estadoRegistroGINPIXPrueba = iota + 1
	estadoRegistroGINPIXAutorizado
	estadoRegistroGINPIXIndeterminado
	estadoRegistroGINPIXConfirmado
)

type registroGINPIXPrueba struct {
	mu                  sync.Mutex
	idempotencia        string
	clave               string
	reservaRef          string
	intento             uint64
	estado              estadoRegistroGINPIXPrueba
	resultado           ports.ResultadoOperacionGINPIX
	reservas            int
	consultas           int
	fallosPreemision    int
	indeterminaciones   int
	confirmaciones      int
	errReserva          error
	errConsulta         error
	errTransicion       error
	errConfirmacion     error
	resultadoConError   bool
	cancelarAlReservar  context.CancelFunc
	cancelarAlConsultar context.CancelFunc
	cancelarAlConfirmar context.CancelFunc
}

func (r *registroGINPIXPrueba) ReservarOperacionGINPIX(
	ctx context.Context,
	s ports.SolicitudOperacionGINPIX,
) (ports.ReservaOperacionGINPIX, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reservas++
	if err := ctx.Err(); err != nil {
		return ports.ReservaOperacionGINPIX{}, err
	}
	datos, err := s.Datos()
	if err != nil {
		return ports.ReservaOperacionGINPIX{}, err
	}
	if r.idempotencia != "" && (r.idempotencia != datos.IdempotenciaRef || r.clave != datos.ClaveOperacionRef) {
		return ports.ReservaOperacionGINPIX{}, ports.ErrColisionOperacionGINPIX
	}
	if r.idempotencia == "" {
		r.idempotencia, r.clave = datos.IdempotenciaRef, datos.ClaveOperacionRef
		r.estado = estadoRegistroGINPIXReintentable
	}
	nuevaAutorizacion := false
	if r.estado == estadoRegistroGINPIXReintentable {
		r.intento++
		r.reservaRef = "reserva-operacion-ginpix-" + strconv.FormatUint(r.intento, 10)
		r.estado = estadoRegistroGINPIXAutorizado
		nuevaAutorizacion = true
	}
	reserva := r.reservaActual()
	if r.estado == estadoRegistroGINPIXAutorizado && !nuevaAutorizacion {
		reserva.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
	}
	if r.cancelarAlReservar != nil {
		r.cancelarAlReservar()
		r.cancelarAlReservar = nil
	}
	if r.errReserva != nil {
		return reserva, r.errReserva
	}
	return reserva, nil
}

func (r *registroGINPIXPrueba) ConsultarReservaOperacionGINPIX(
	ctx context.Context,
	s ports.SolicitudOperacionGINPIX,
) (ports.ReservaOperacionGINPIX, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.consultas++
	if err := ctx.Err(); err != nil {
		return ports.ReservaOperacionGINPIX{}, err
	}
	datos, err := s.Datos()
	if err != nil {
		return ports.ReservaOperacionGINPIX{}, err
	}
	if r.idempotencia == "" {
		return ports.ReservaOperacionGINPIX{}, ports.ErrOperacionGINPIXNoReservada
	}
	if r.idempotencia != datos.IdempotenciaRef || r.clave != datos.ClaveOperacionRef {
		return ports.ReservaOperacionGINPIX{}, ports.ErrColisionOperacionGINPIX
	}
	reserva := r.reservaActual()
	if r.cancelarAlConsultar != nil {
		r.cancelarAlConsultar()
		r.cancelarAlConsultar = nil
	}
	if r.errConsulta != nil {
		return reserva, r.errConsulta
	}
	return reserva, nil
}

func (r *registroGINPIXPrueba) RegistrarFalloPreemisionGINPIX(
	_ context.Context,
	reserva ports.ReservaOperacionGINPIX,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.errTransicion != nil || r.estado != estadoRegistroGINPIXAutorizado ||
		reserva.ReservaRef != r.reservaRef {
		return errors.New("transicion preemision rechazada")
	}
	r.fallosPreemision++
	r.estado = estadoRegistroGINPIXReintentable
	return nil
}

func (r *registroGINPIXPrueba) MarcarOperacionGINPIXIndeterminada(
	_ context.Context,
	reserva ports.ReservaOperacionGINPIX,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.errTransicion != nil || r.estado != estadoRegistroGINPIXAutorizado ||
		reserva.ReservaRef != r.reservaRef {
		return errors.New("transicion indeterminada rechazada")
	}
	r.indeterminaciones++
	r.estado = estadoRegistroGINPIXIndeterminado
	return nil
}

func (r *registroGINPIXPrueba) ConfirmarOperacionGINPIX(
	_ context.Context,
	reserva ports.ReservaOperacionGINPIX,
	recibo ports.ReciboExternoOperacionGINPIX,
) (ports.ResultadoOperacionGINPIX, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.estado != estadoRegistroGINPIXAutorizado && r.estado != estadoRegistroGINPIXIndeterminado {
		return ports.ResultadoOperacionGINPIX{}, errors.New("estado no confirmable")
	}
	r.confirmaciones++
	if r.cancelarAlConfirmar != nil {
		r.cancelarAlConfirmar()
		r.cancelarAlConfirmar = nil
	}
	resultado := ports.ResultadoOperacionGINPIX{
		ConfirmacionLocalRef: "confirmacion-local-ginpix-0001",
		ClaveOperacionRef:    r.clave,
		ReciboExterno:        recibo,
	}
	if r.errConfirmacion != nil {
		if r.resultadoConError {
			return resultado, r.errConfirmacion
		}
		return ports.ResultadoOperacionGINPIX{}, r.errConfirmacion
	}
	r.resultado = resultado
	r.estado = estadoRegistroGINPIXConfirmado
	return resultado, nil
}

func (r *registroGINPIXPrueba) reservaActual() ports.ReservaOperacionGINPIX {
	situacion := ports.ReservaOperacionGINPIXEmisionAutorizada
	switch r.estado {
	case estadoRegistroGINPIXIndeterminado:
		situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
	case estadoRegistroGINPIXConfirmado:
		situacion = ports.ReservaOperacionGINPIXConfirmada
	}
	return ports.ReservaOperacionGINPIX{
		ReservaRef: r.reservaRef, ClaveOperacionRef: r.clave, Intento: r.intento,
		Situacion: situacion, Resultado: r.resultado,
	}
}

type conectorGINPIXPrueba struct {
	mu               sync.Mutex
	emisiones        int
	consultas        int
	reciboEmision    ports.ReciboExternoOperacionGINPIX
	reciboConsulta   ports.ReciboExternoOperacionGINPIX
	errEmision       error
	errConsulta      error
	cancelarEmision  context.CancelFunc
	cancelarConsulta context.CancelFunc
	emisionIniciada  chan struct{}
	continuarEmision chan struct{}
}

func (c *conectorGINPIXPrueba) EmitirOperacionGINPIX(
	_ context.Context,
	_ ports.SolicitudOperacionGINPIX,
	_ ports.ReservaOperacionGINPIX,
) (ports.ReciboExternoOperacionGINPIX, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.emisiones++
	if c.emisionIniciada != nil {
		close(c.emisionIniciada)
		continuar := c.continuarEmision
		c.mu.Unlock()
		<-continuar
		c.mu.Lock()
	}
	if c.cancelarEmision != nil {
		c.cancelarEmision()
		c.cancelarEmision = nil
	}
	return c.reciboEmision, c.errEmision
}

func (c *conectorGINPIXPrueba) ConsultarOperacionGINPIX(
	_ context.Context,
	_ ports.SolicitudOperacionGINPIX,
	_ ports.ReservaOperacionGINPIX,
) (ports.ReciboExternoOperacionGINPIX, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consultas++
	if c.cancelarConsulta != nil {
		c.cancelarConsulta()
		c.cancelarConsulta = nil
	}
	return c.reciboConsulta, c.errConsulta
}

func TestCTLITEO706AReservaEmiteUnaVezReplayExactoYColisionaVariantes(t *testing.T) {
	servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
	resultado, err := servicio.Ejecutar(context.Background(), solicitud)
	if err != nil || resultado.ValidarPara(solicitud.puertoPrueba(t)) != nil {
		t.Fatalf("ejecutar operacion: resultado=%#v error=%v", resultado, err)
	}
	replay, err := servicio.Ejecutar(context.Background(), solicitud)
	if err != nil || !reflect.DeepEqual(replay, resultado) {
		t.Fatalf("replay divergente: %#v / %v", replay, err)
	}
	if conector.emisiones != 1 || registro.confirmaciones != 1 || registro.reservas != 2 {
		t.Fatalf("efectos duplicados: emisiones=%d confirmaciones=%d reservas=%d",
			conector.emisiones, registro.confirmaciones, registro.reservas)
	}

	variante := solicitud
	variante.Mapeo = mapeoOperacionGINPIXVariantePrueba(t, solicitud.Mapeo)
	if _, err := servicio.Ejecutar(context.Background(), variante); !errors.Is(
		err,
		ErrColisionOperacionGINPIXRecuperable,
	) {
		t.Fatalf("variante no colisiono: %v", err)
	}
	if conector.emisiones != 1 {
		t.Fatal("la colision alcanzo el emisor")
	}
}

func TestCTLITEO706AReservaConcurrenteConcedeUnaSolaEmision(t *testing.T) {
	servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
	conector.emisionIniciada = make(chan struct{})
	conector.continuarEmision = make(chan struct{})
	primero := make(chan error, 1)
	go func() {
		_, err := servicio.Ejecutar(context.Background(), solicitud)
		primero <- err
	}()
	<-conector.emisionIniciada

	const concurrentes = 16
	errores := make(chan error, concurrentes)
	var grupo sync.WaitGroup
	for indice := 0; indice < concurrentes; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultado, err := servicio.Ejecutar(context.Background(), solicitud)
			if resultado != (ports.ResultadoOperacionGINPIX{}) ||
				!errors.Is(err, ErrOperacionGINPIXIndeterminada) {
				errores <- errors.New("una reserva concurrente obtuvo emision")
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
	close(conector.continuarEmision)
	if err := <-primero; err != nil {
		t.Fatalf("completar emision unica: %v", err)
	}
	if conector.emisiones != 1 || registro.confirmaciones != 1 ||
		registro.reservas != concurrentes+1 {
		t.Fatalf("reserva concurrente duplicada: emisiones=%d reservas=%d confirmaciones=%d",
			conector.emisiones, registro.reservas, registro.confirmaciones)
	}
}

func TestCTLITEO706AFalloPreemisionAutorizaNuevaReserva(t *testing.T) {
	servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
	conector.errEmision = errors.Join(
		ports.ErrEmisionOperacionGINPIXNoIniciada,
		errors.New("detalle privado preemision"),
	)
	conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
	if resultado, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
		err,
		ErrOperacionGINPIXNoDisponible,
	) || resultado != (ports.ResultadoOperacionGINPIX{}) {
		t.Fatalf("fallo preemision mal clasificado: %#v / %v", resultado, err)
	}
	if registro.fallosPreemision != 1 || registro.indeterminaciones != 0 ||
		registro.estado != estadoRegistroGINPIXReintentable {
		t.Fatal("el intento preemision no libero una nueva reserva")
	}
	conector.errEmision = nil
	conector.reciboEmision = reciboExternoOperacionGINPIXPrueba(t, solicitud.puertoPrueba(t))
	if _, err := servicio.Ejecutar(context.Background(), solicitud); err != nil {
		t.Fatalf("nuevo intento autorizado: %v", err)
	}
	if conector.emisiones != 2 || registro.intento != 2 || registro.confirmaciones != 1 {
		t.Fatalf("reintento incorrecto: emisiones=%d intento=%d", conector.emisiones, registro.intento)
	}
}

func TestCTLITEO706CRecuperaEmisionAutorizadaAbandonadaSinEmitir(t *testing.T) {
	servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
	puerto := solicitud.puertoPrueba(t)
	reserva, err := registro.ReservarOperacionGINPIX(context.Background(), puerto)
	if err != nil || reserva.Situacion != ports.ReservaOperacionGINPIXEmisionAutorizada {
		t.Fatalf("preparar autorizacion abandonada: %#v / %v", reserva, err)
	}
	conector.reciboConsulta = reciboExternoOperacionGINPIXPrueba(t, puerto)
	resultado, err := servicio.Recuperar(context.Background(), solicitud)
	if err != nil || resultado.ValidarPara(puerto) != nil || conector.emisiones != 0 ||
		conector.consultas != 1 || registro.indeterminaciones != 1 ||
		registro.confirmaciones != 1 || registro.estado != estadoRegistroGINPIXConfirmado {
		t.Fatalf("la autorizacion abandonada no se reconcilio: %#v / %v", resultado, err)
	}
}

func TestCTLITEO706CTransicionAbandonadaFallaCerradaSinEfectos(t *testing.T) {
	servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
	puerto := solicitud.puertoPrueba(t)
	if _, err := registro.ReservarOperacionGINPIX(context.Background(), puerto); err != nil {
		t.Fatal(err)
	}
	registro.errTransicion = errors.New("fallo durable sintetico")
	resultado, err := servicio.Recuperar(context.Background(), solicitud)
	if resultado != (ports.ResultadoOperacionGINPIX{}) ||
		!errors.Is(err, ErrOperacionGINPIXIndeterminada) || conector.emisiones != 0 ||
		conector.consultas != 0 || registro.confirmaciones != 0 ||
		registro.estado != estadoRegistroGINPIXAutorizado {
		t.Fatalf("el fallo durable alcanzo un efecto: %#v / %v", resultado, err)
	}
}

func TestCTLITEO706APostemisionSoloConsultaYConcilia(t *testing.T) {
	servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
	conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
	conector.errEmision = ports.ErrEmisionOperacionGINPIXIndeterminada
	if _, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
		err,
		ErrOperacionGINPIXIndeterminada,
	) {
		t.Fatalf("postemision no quedo indeterminada: %v", err)
	}
	if registro.estado != estadoRegistroGINPIXIndeterminado || registro.indeterminaciones != 1 {
		t.Fatal("no se persistio el estado indeterminado")
	}
	if _, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
		err,
		ErrOperacionGINPIXIndeterminada,
	) || conector.emisiones != 1 {
		t.Fatalf("el replay reenvio una operacion pendiente: %v", err)
	}
	conector.reciboConsulta = reciboExternoOperacionGINPIXPrueba(t, solicitud.puertoPrueba(t))
	resultado, err := servicio.Recuperar(context.Background(), solicitud)
	if err != nil || resultado.ValidarPara(solicitud.puertoPrueba(t)) != nil {
		t.Fatalf("conciliar operacion: %#v / %v", resultado, err)
	}
	replay, err := servicio.Recuperar(context.Background(), solicitud)
	if err != nil || !reflect.DeepEqual(resultado, replay) || conector.consultas != 1 ||
		conector.emisiones != 1 || registro.confirmaciones != 1 {
		t.Fatalf("recuperacion no idempotente: %#v / %v", replay, err)
	}
}

func TestCTLITEO706ACancelacionEnCadaFronteraFallaCerrada(t *testing.T) {
	t.Run("antes de reservar", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		if _, err := servicio.Ejecutar(ctx, solicitud); !errors.Is(err, context.Canceled) ||
			registro.reservas != 0 || conector.emisiones != 0 {
			t.Fatalf("cancelacion previa produjo trabajo: %v", err)
		}
	})

	t.Run("tras reservar antes de emitir", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		ctx, cancelar := context.WithCancel(context.Background())
		registro.cancelarAlReservar = cancelar
		if _, err := servicio.Ejecutar(ctx, solicitud); !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ErrOperacionGINPIXIndeterminada) || conector.emisiones != 0 ||
			registro.fallosPreemision != 0 || registro.indeterminaciones != 1 ||
			registro.estado != estadoRegistroGINPIXIndeterminado {
			t.Fatalf("cancelacion preemision mal cerrada: %v", err)
		}
		if _, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXIndeterminada,
		) || conector.emisiones != 0 || registro.intento != 1 {
			t.Fatalf("el replay tras cancelacion volvio a emitir: %v", err)
		}
	})

	t.Run("durante emision", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		ctx, cancelar := context.WithCancel(context.Background())
		conector.cancelarEmision = cancelar
		conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
		conector.errEmision = errors.Join(
			ports.ErrEmisionOperacionGINPIXNoIniciada,
			context.Canceled,
		)
		if resultado, err := servicio.Ejecutar(ctx, solicitud); !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ErrOperacionGINPIXIndeterminada) ||
			resultado != (ports.ResultadoOperacionGINPIX{}) ||
			registro.estado != estadoRegistroGINPIXIndeterminado {
			t.Fatalf("cancelacion postemision mal cerrada: %#v / %v", resultado, err)
		}
	})

	t.Run("durante consulta", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
		conector.errEmision = ports.ErrEmisionOperacionGINPIXIndeterminada
		_, _ = servicio.Ejecutar(context.Background(), solicitud)
		ctx, cancelar := context.WithCancel(context.Background())
		conector.cancelarConsulta = cancelar
		conector.reciboConsulta = reciboExternoOperacionGINPIXPrueba(t, solicitud.puertoPrueba(t))
		if resultado, err := servicio.Recuperar(ctx, solicitud); !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ErrOperacionGINPIXIndeterminada) ||
			resultado != (ports.ResultadoOperacionGINPIX{}) || registro.confirmaciones != 0 {
			t.Fatalf("cancelacion en consulta confirmo: %#v / %v", resultado, err)
		}
	})

	t.Run("tras consulta local antes del conector", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
		conector.errEmision = ports.ErrEmisionOperacionGINPIXIndeterminada
		_, _ = servicio.Ejecutar(context.Background(), solicitud)
		ctx, cancelar := context.WithCancel(context.Background())
		registro.cancelarAlConsultar = cancelar
		if resultado, err := servicio.Recuperar(ctx, solicitud); !errors.Is(err, context.Canceled) ||
			resultado != (ports.ResultadoOperacionGINPIX{}) || conector.consultas != 0 {
			t.Fatalf("cancelacion tras consulta local alcanzo el conector: %#v / %v", resultado, err)
		}
	})

	t.Run("durante confirmacion atomica", func(t *testing.T) {
		servicio, registro, _, solicitud := escenarioOperacionGINPIXRecuperable(t)
		ctx, cancelar := context.WithCancel(context.Background())
		registro.cancelarAlConfirmar = cancelar
		if resultado, err := servicio.Ejecutar(ctx, solicitud); !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ErrOperacionGINPIXIndeterminada) ||
			resultado != (ports.ResultadoOperacionGINPIX{}) || registro.confirmaciones != 1 {
			t.Fatalf("cancelacion en confirmacion publico resultado: %#v / %v", resultado, err)
		}
	})
}

func TestCTLITEO706AResultadoMasErrorYRecibosInvalidosFallanCerrados(t *testing.T) {
	t.Run("reserva resultado mas error", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		registro.errReserva = errors.New("reserva ambigua")
		if resultado, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXNoDisponible,
		) || resultado != (ports.ResultadoOperacionGINPIX{}) || conector.emisiones != 0 {
			t.Fatalf("reserva resultado+error alcanzo el emisor: %#v / %v", resultado, err)
		}
	})

	t.Run("emision resultado mas error", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		conector.errEmision = ports.ErrEmisionOperacionGINPIXNoIniciada
		if resultado, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXIndeterminada,
		) || resultado != (ports.ResultadoOperacionGINPIX{}) ||
			registro.estado != estadoRegistroGINPIXIndeterminado || registro.fallosPreemision != 0 {
			t.Fatalf("resultado+error se trato como preemision: %#v / %v", resultado, err)
		}
	})

	t.Run("emision error generico sin recibo", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
		conector.errEmision = errors.New("fallo de emision incierto")
		if resultado, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXIndeterminada,
		) || resultado != (ports.ResultadoOperacionGINPIX{}) ||
			registro.estado != estadoRegistroGINPIXIndeterminado || registro.fallosPreemision != 0 {
			t.Fatalf("error generico se trato como preemision: %#v / %v", resultado, err)
		}
	})

	t.Run("consulta local resultado mas error", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
		conector.errEmision = ports.ErrEmisionOperacionGINPIXIndeterminada
		_, _ = servicio.Ejecutar(context.Background(), solicitud)
		registro.errConsulta = errors.New("lectura local ambigua")
		if resultado, err := servicio.Recuperar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXNoDisponible,
		) || resultado != (ports.ResultadoOperacionGINPIX{}) || conector.consultas != 0 {
			t.Fatalf("consulta local resultado+error alcanzo el conector: %#v / %v", resultado, err)
		}
	})

	t.Run("consulta resultado mas error", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		conector.reciboEmision = ports.ReciboExternoOperacionGINPIX{}
		conector.errEmision = ports.ErrEmisionOperacionGINPIXIndeterminada
		_, _ = servicio.Ejecutar(context.Background(), solicitud)
		conector.reciboConsulta = reciboExternoOperacionGINPIXPrueba(t, solicitud.puertoPrueba(t))
		conector.errConsulta = errors.New("detalle privado")
		if resultado, err := servicio.Recuperar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXIndeterminada,
		) || resultado != (ports.ResultadoOperacionGINPIX{}) || registro.confirmaciones != 0 {
			t.Fatalf("consulta resultado+error confirmo: %#v / %v", resultado, err)
		}
	})

	t.Run("confirmacion resultado mas error", func(t *testing.T) {
		servicio, registro, _, solicitud := escenarioOperacionGINPIXRecuperable(t)
		registro.errConfirmacion = errors.New("commit ambiguo")
		registro.resultadoConError = true
		if resultado, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXIndeterminada,
		) || resultado != (ports.ResultadoOperacionGINPIX{}) {
			t.Fatalf("confirmacion resultado+error publicada: %#v / %v", resultado, err)
		}
	})

	t.Run("recibo desligado", func(t *testing.T) {
		servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
		conector.reciboEmision.IdempotenciaRef = "idempotencia-variante"
		if resultado, err := servicio.Ejecutar(context.Background(), solicitud); !errors.Is(
			err,
			ErrOperacionGINPIXIndeterminada,
		) || resultado != (ports.ResultadoOperacionGINPIX{}) ||
			registro.estado != estadoRegistroGINPIXIndeterminado {
			t.Fatalf("recibo desligado publicado: %#v / %v", resultado, err)
		}
	})
}

func TestCTLITEO706ATypedNilAliasingYRecuperacionSinReserva(t *testing.T) {
	var registroNulo *registroGINPIXPrueba
	var conectorNulo *conectorGINPIXPrueba
	if servicio, err := NuevoServicioOperacionGINPIXRecuperable(registroNulo, &conectorGINPIXPrueba{}, &conectorGINPIXPrueba{}); servicio != nil ||
		!errors.Is(err, ErrServicioOperacionGINPIXRecuperableInvalido) {
		t.Fatalf("registro nil tipado aceptado: %#v / %v", servicio, err)
	}
	if servicio, err := NuevoServicioOperacionGINPIXRecuperable(&registroGINPIXPrueba{}, conectorNulo, &conectorGINPIXPrueba{}); servicio != nil ||
		!errors.Is(err, ErrServicioOperacionGINPIXRecuperableInvalido) {
		t.Fatalf("emisor nil tipado aceptado: %#v / %v", servicio, err)
	}
	if servicio, err := NuevoServicioOperacionGINPIXRecuperable(&registroGINPIXPrueba{}, &conectorGINPIXPrueba{}, conectorNulo); servicio != nil ||
		!errors.Is(err, ErrServicioOperacionGINPIXRecuperableInvalido) {
		t.Fatalf("consultor nil tipado aceptado: %#v / %v", servicio, err)
	}

	servicio, registro, conector, solicitud := escenarioOperacionGINPIXRecuperable(t)
	if _, err := servicio.Recuperar(context.Background(), solicitud); !errors.Is(
		err,
		ErrOperacionGINPIXNoDisponible,
	) || registro.reservas != 0 || conector.emisiones != 0 || conector.consultas != 0 {
		t.Fatalf("recuperacion creo o emitio operacion: %v", err)
	}
	puerto := solicitud.puertoPrueba(t)
	recibo, err := puerto.ReciboIncorporacion()
	if err != nil {
		t.Fatal(err)
	}
	recibo.Documentos[0].Referencia = "documento-mutado"
	posterior, err := puerto.ReciboIncorporacion()
	if err != nil || posterior.Documentos[0].Referencia == "documento-mutado" {
		t.Fatal("la solicitud neutral expuso alias del recibo")
	}
}

func (s SolicitudOperacionGINPIXRecuperable) puertoPrueba(t *testing.T) ports.SolicitudOperacionGINPIX {
	t.Helper()
	puerto, err := ports.NuevaSolicitudOperacionGINPIX(s.Mapeo, s.Orden, s.Incorporacion)
	if err != nil {
		t.Fatalf("crear solicitud neutral: %v", err)
	}
	return puerto
}

func escenarioOperacionGINPIXRecuperable(t *testing.T) (
	*ServicioOperacionGINPIXRecuperable,
	*registroGINPIXPrueba,
	*conectorGINPIXPrueba,
	SolicitudOperacionGINPIXRecuperable,
) {
	t.Helper()
	_, contextos, transaccion, solicitudIncorporacion := escenarioConfirmacionIncorporacion(t)
	orden, err := ports.NuevaOrdenConfirmarIncorporacion(
		contextos.contexto,
		solicitudIncorporacion.datos(),
		instanteConfirmacionIncorporacionPrueba,
	)
	if err != nil {
		t.Fatalf("crear orden O7-02: %v", err)
	}
	recibo, err := transaccion.ConfirmarIncorporacion(context.Background(), orden)
	if err != nil {
		t.Fatalf("crear recibo O7-02: %v", err)
	}
	campo, err := domain.CampoValorGINPIX("VALOR-SINTETICO-GINPIX-06A")
	if err != nil {
		t.Fatal(err)
	}
	modelo, err := domain.NuevoModeloCanonicoGINPIX(domain.BorradorModeloCanonicoGINPIX{
		Esquema: domain.EsquemaModeloCanonicoGINPIXV1, VersionExpediente: recibo.VersionExpediente,
		ExpedienteRef: recibo.ExpedienteRef, IncorporacionRef: recibo.ActuacionRef,
		ProcedenciaRef: "procedencia-modelo-ginpix-06a", CorrelacionRef: "correlacion-ginpix-06a",
		IdempotenciaRef: "idempotencia-ginpix-06a",
		Datos:           []domain.DatoCanonicoGINPIX{{Clave: "codigo_puesto", Campo: campo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mapeo, err := domain.PublicarMapeoVersionadoGINPIX(domain.BorradorMapeoVersionadoGINPIX{
		Esquema: domain.EsquemaMapeoGINPIXV1, Referencia: "mapeo-ginpix-06a", Version: 1,
		ProcedenciaRef: "procedencia-mapeo-ginpix-06a",
		Reglas: []domain.ReglaMapeoGINPIX{{
			CampoCanonico: "codigo_puesto", CampoDestino: "puesto", Obligatorio: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	solicitudMapeo, err := ports.NuevaSolicitudMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudOperacionGINPIXRecuperable{
		Mapeo: solicitudMapeo, Orden: orden, Incorporacion: recibo,
	}
	registro := &registroGINPIXPrueba{}
	conector := &conectorGINPIXPrueba{}
	conector.reciboEmision = reciboExternoOperacionGINPIXPrueba(t, solicitud.puertoPrueba(t))
	servicio, err := NuevoServicioOperacionGINPIXRecuperable(registro, conector, conector)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio, registro, conector, solicitud
}

func reciboExternoOperacionGINPIXPrueba(
	t *testing.T,
	s ports.SolicitudOperacionGINPIX,
) ports.ReciboExternoOperacionGINPIX {
	t.Helper()
	d, err := s.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return ports.ReciboExternoOperacionGINPIX{
		ReciboExternoRef: "recibo-externo-ginpix-06a", EvidenciaExternaRef: "evidencia-externa-ginpix-06a",
		EvidenciaExternaHuellaSHA256: strings.Repeat("a", 64), ClaveOperacionRef: d.ClaveOperacionRef,
		VersionExpediente: d.VersionExpediente, ExpedienteRef: d.ExpedienteRef,
		IncorporacionRef: d.IncorporacionRef, ReciboIncorporacionRef: d.ReciboIncorporacionRef,
		ResultadoPersonalRef: d.ResultadoPersonalRef, ReciboPersonalRef: d.ReciboPersonalRef,
		CorrelacionRef: d.CorrelacionRef, IdempotenciaRef: d.IdempotenciaRef,
		ModeloHuellaSHA256: d.ModeloHuellaSHA256, MapeoRef: d.MapeoRef,
		MapeoVersion: d.MapeoVersion, MapeoHuellaSHA256: d.MapeoHuellaSHA256,
		CargaHuellaSHA256: d.CargaHuellaSHA256,
	}
}

func mapeoOperacionGINPIXVariantePrueba(
	t *testing.T,
	original ports.SolicitudMapeoGINPIX,
) ports.SolicitudMapeoGINPIX {
	t.Helper()
	modelo, err := original.Modelo()
	if err != nil {
		t.Fatal(err)
	}
	publicacion := modelo.Publicacion()
	publicacion.Datos[0].Campo, err = domain.CampoValorGINPIX("VALOR-VARIANTE-GINPIX-06A")
	if err != nil {
		t.Fatal(err)
	}
	modelo, err = domain.NuevoModeloCanonicoGINPIX(publicacion.BorradorModeloCanonicoGINPIX)
	if err != nil {
		t.Fatal(err)
	}
	mapeo, err := original.Mapeo()
	if err != nil {
		t.Fatal(err)
	}
	variante, err := ports.NuevaSolicitudMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatal(err)
	}
	return variante
}
