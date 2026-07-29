package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type relojCronologiaCuadroPrueba struct {
	mu       sync.Mutex
	instante time.Time
	lecturas int
}

func (r *relojCronologiaCuadroPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lecturas++
	return r.instante
}

func (r *relojCronologiaCuadroPrueba) fijar(instante time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instante = instante
}

func (r *relojCronologiaCuadroPrueba) totalLecturas() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lecturas
}

type autoridadCronologiaCuadroPrueba struct {
	contexto ports.ContextoConsultaRRHH
	antes    func()
	llamadas int
}

func (a *autoridadCronologiaCuadroPrueba) ResolverContextoConsultaRRHH(
	context.Context,
) (ports.ContextoConsultaRRHH, error) {
	a.llamadas++
	if a.antes != nil {
		a.antes()
	}
	return a.contexto, nil
}

type autorizadorCronologiaCuadroPrueba struct {
	capacidad ports.CapacidadConsultaRRHH
	despues   func()
	instantes []time.Time
}

func (a *autorizadorCronologiaCuadroPrueba) AutorizarCuadroRRHH(
	_ context.Context,
	_ ports.ContextoConsultaRRHH,
	_ ports.SolicitudCuadroRRHH,
	instante time.Time,
) (ports.CapacidadConsultaRRHH, error) {
	a.instantes = append(a.instantes, instante)
	if a.despues != nil {
		a.despues()
	}
	return a.capacidad, nil
}

func (*autorizadorCronologiaCuadroPrueba) AutorizarDetalleRRHH(
	context.Context,
	ports.ContextoConsultaRRHH,
	ports.SolicitudDetalleRRHH,
	time.Time,
) (ports.CapacidadConsultaRRHH, error) {
	return ports.CapacidadConsultaRRHH{}, errors.New(
		"prueba de cronologia de cuadro: detalle no permitido",
	)
}

type sesionCronologiaCuadroPrueba struct {
	t           *testing.T
	expedientes []ports.ResumenExpedienteRRHH
	llamadas    int
	instante    time.Time
}

func (s *sesionCronologiaCuadroPrueba) ConsultarCuadroYRegistrar(
	_ context.Context,
	orden ports.OrdenConsultaCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	s.t.Helper()
	s.llamadas++
	s.instante = orden.Instante()
	recibo := reciboConsultaRRHHPrueba(
		s.t,
		orden.Contexto(),
		orden.Capacidad(),
		orden.Instante(),
		"",
		0,
		uint16(len(s.expedientes)),
	)
	return ports.PaginaCuadroRRHH{
		GeneradaEn:  orden.Instante(),
		Expedientes: append([]ports.ResumenExpedienteRRHH(nil), s.expedientes...),
		Lectura:     recibo,
	}, nil
}

func (*sesionCronologiaCuadroPrueba) ConsultarDetalleYRegistrar(
	context.Context,
	ports.OrdenConsultaDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	return ports.DetalleExpedienteRRHH{}, errors.New(
		"prueba de cronologia de cuadro: detalle no permitido",
	)
}

func TestConsultaCuadroRRHHUsaCronologiaPosteriorALasFronteras(
	t *testing.T,
) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	instantePrevio := entorno.ahora.Add(-time.Second)
	instanteAutorizacion := entorno.ahora
	instanteOrden := entorno.ahora.Add(time.Second)
	reloj := &relojCronologiaCuadroPrueba{instante: instantePrevio}
	autoridad := &autoridadCronologiaCuadroPrueba{
		contexto: entorno.contexto,
		antes: func() {
			reloj.fijar(instanteAutorizacion)
		},
	}
	autorizador := &autorizadorCronologiaCuadroPrueba{
		capacidad: entorno.autorizador.capacidadCuadro,
		despues: func() {
			reloj.fijar(instanteOrden)
		},
	}
	sesion := &sesionCronologiaCuadroPrueba{
		t: t, expedientes: entorno.sesion.pagina.Expedientes,
	}
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		autoridad, autorizador, sesion, reloj,
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = servicio.Consultar(
		context.Background(),
		entorno.cuadro,
	); err != nil {
		t.Fatalf("consultar con capacidad posterior al instante previo: %v", err)
	}
	if autoridad.llamadas != 1 || len(autorizador.instantes) != 1 ||
		autorizador.instantes[0] != instanteAutorizacion ||
		sesion.llamadas != 1 || sesion.instante != instanteOrden ||
		reloj.totalLecturas() != 2 {
		t.Fatalf(
			"cronologia incorrecta: autoridad=%d autorizaciones=%v "+
				"sesion=%d instante_orden=%v lecturas=%d",
			autoridad.llamadas,
			autorizador.instantes,
			sesion.llamadas,
			sesion.instante,
			reloj.totalLecturas(),
		)
	}
}

func TestConsultaCuadroRRHHRechazaCronologiaNoConfiableAntesDeSesion(
	t *testing.T,
) {
	t.Parallel()
	for _, caso := range []struct {
		nombre               string
		instanteAutorizacion func(time.Time) time.Time
		instanteOrden        func(time.Time) time.Time
		autorizaciones       int
		lecturas             int
	}{
		{
			nombre: "retroceso",
			instanteAutorizacion: func(base time.Time) time.Time {
				return base
			},
			instanteOrden: func(base time.Time) time.Time {
				return base.Add(-time.Microsecond)
			},
			autorizaciones: 1,
			lecturas:       2,
		},
		{
			nombre: "capacidad_vencida",
			instanteAutorizacion: func(base time.Time) time.Time {
				return base
			},
			instanteOrden: func(base time.Time) time.Time {
				return base.Add(ports.DuracionMaximaCapacidadConsultaRRHH)
			},
			autorizaciones: 1,
			lecturas:       2,
		},
		{
			nombre: "autorizacion_cero",
			instanteAutorizacion: func(time.Time) time.Time {
				return time.Time{}
			},
			instanteOrden:  func(base time.Time) time.Time { return base },
			autorizaciones: 0,
			lecturas:       1,
		},
		{
			nombre: "autorizacion_no_canonica",
			instanteAutorizacion: func(base time.Time) time.Time {
				return base.Add(time.Nanosecond)
			},
			instanteOrden:  func(base time.Time) time.Time { return base },
			autorizaciones: 0,
			lecturas:       1,
		},
		{
			nombre: "orden_cero",
			instanteAutorizacion: func(base time.Time) time.Time {
				return base
			},
			instanteOrden: func(time.Time) time.Time {
				return time.Time{}
			},
			autorizaciones: 1,
			lecturas:       2,
		},
		{
			nombre: "orden_no_canonica",
			instanteAutorizacion: func(base time.Time) time.Time {
				return base
			},
			instanteOrden: func(base time.Time) time.Time {
				return base.Add(time.Nanosecond)
			},
			autorizaciones: 1,
			lecturas:       2,
		},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			reloj := &relojCronologiaCuadroPrueba{}
			autoridad := &autoridadCronologiaCuadroPrueba{
				contexto: entorno.contexto,
				antes: func() {
					reloj.fijar(caso.instanteAutorizacion(entorno.ahora))
				},
			}
			autorizador := &autorizadorCronologiaCuadroPrueba{
				capacidad: entorno.autorizador.capacidadCuadro,
				despues: func() {
					reloj.fijar(caso.instanteOrden(entorno.ahora))
				},
			}
			sesion := &sesionCronologiaCuadroPrueba{
				t: t, expedientes: entorno.sesion.pagina.Expedientes,
			}
			servicio, err := NuevoServicioConsultaCuadroRRHH(
				autoridad, autorizador, sesion, reloj,
			)
			if err != nil {
				t.Fatal(err)
			}

			_, err = servicio.Consultar(context.Background(), entorno.cuadro)
			if !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("cronologia no confiable aceptada: %v", err)
			}
			if len(autorizador.instantes) != caso.autorizaciones ||
				sesion.llamadas != 0 ||
				reloj.totalLecturas() != caso.lecturas {
				t.Fatalf(
					"hubo actividad posterior: autorizaciones=%d sesion=%d "+
						"lecturas=%d",
					len(autorizador.instantes),
					sesion.llamadas,
					reloj.totalLecturas(),
				)
			}
		})
	}
}

func TestConsultaCuadroRRHHCanceladaTrasAutorizarNoInvocaSesion(
	t *testing.T,
) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	ctx, cancelar := context.WithCancel(context.Background())
	reloj := &relojCronologiaCuadroPrueba{instante: entorno.ahora}
	autoridad := &autoridadCronologiaCuadroPrueba{contexto: entorno.contexto}
	autorizador := &autorizadorCronologiaCuadroPrueba{
		capacidad: entorno.autorizador.capacidadCuadro,
		despues:   cancelar,
	}
	sesion := &sesionCronologiaCuadroPrueba{
		t: t, expedientes: entorno.sesion.pagina.Expedientes,
	}
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		autoridad, autorizador, sesion, reloj,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = servicio.Consultar(ctx, entorno.cuadro)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no preservada: %v", err)
	}
	if sesion.llamadas != 0 || reloj.totalLecturas() != 1 {
		t.Fatalf(
			"cancelacion alcanzo actividad posterior: sesion=%d lecturas=%d",
			sesion.llamadas,
			reloj.totalLecturas(),
		)
	}
}
