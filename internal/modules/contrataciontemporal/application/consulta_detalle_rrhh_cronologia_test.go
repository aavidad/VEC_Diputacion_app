package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type relojSecuencialDetalleRRHHPrueba struct {
	instantes []time.Time
	lecturas  int
	despuesDe func(int)
}

func (r *relojSecuencialDetalleRRHHPrueba) Ahora() time.Time {
	indice := r.lecturas
	r.lecturas++
	if r.despuesDe != nil {
		defer r.despuesDe(r.lecturas)
	}
	if indice >= len(r.instantes) {
		return time.Time{}
	}
	return r.instantes[indice]
}

type autorizadorCronologiaDetalleRRHHPrueba struct {
	capacidad ports.CapacidadConsultaRRHH
	instantes []time.Time
	cancelar  context.CancelFunc
}

func (a *autorizadorCronologiaDetalleRRHHPrueba) AutorizarCuadroRRHH(
	context.Context,
	ports.ContextoConsultaRRHH,
	ports.SolicitudCuadroRRHH,
	time.Time,
) (ports.CapacidadConsultaRRHH, error) {
	return ports.CapacidadConsultaRRHH{}, ports.ErrConsultaRRHHNoDisponible
}

func (a *autorizadorCronologiaDetalleRRHHPrueba) AutorizarDetalleRRHH(
	_ context.Context,
	_ ports.ContextoConsultaRRHH,
	_ ports.SolicitudDetalleRRHH,
	instante time.Time,
) (ports.CapacidadConsultaRRHH, error) {
	a.instantes = append(a.instantes, instante)
	if a.cancelar != nil {
		a.cancelar()
	}
	return a.capacidad, nil
}

type sesionCronologiaDetalleRRHHPrueba struct {
	detalle  ports.DetalleExpedienteRRHH
	ordenes  []ports.OrdenConsultaDetalleRRHH
	llamadas int
}

func (s *sesionCronologiaDetalleRRHHPrueba) ConsultarCuadroYRegistrar(
	context.Context,
	ports.OrdenConsultaCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
}

func (s *sesionCronologiaDetalleRRHHPrueba) ConsultarDetalleYRegistrar(
	_ context.Context,
	orden ports.OrdenConsultaDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	s.llamadas++
	s.ordenes = append(s.ordenes, orden)
	return s.detalle, nil
}

func TestConsultaDetalleRRHHUsaInstantePosteriorALaAutorizacion(
	t *testing.T,
) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	emitidaEn := entorno.autorizador.capacidadDetalle.ValidaDesde()
	instanteAutorizacion := emitidaEn.Add(-time.Microsecond)
	instanteOrden := emitidaEn
	autorizador := &autorizadorCronologiaDetalleRRHHPrueba{
		capacidad: entorno.autorizador.capacidadDetalle,
	}
	sesion := &sesionCronologiaDetalleRRHHPrueba{
		detalle: entorno.sesion.detalle,
	}
	type observacion struct {
		autoridad   int
		autorizador int
	}
	observaciones := make([]observacion, 0, 2)
	reloj := &relojSecuencialDetalleRRHHPrueba{
		instantes: []time.Time{instanteAutorizacion, instanteOrden},
		despuesDe: func(int) {
			observaciones = append(observaciones, observacion{
				autoridad:   entorno.autoridad.llamadas,
				autorizador: len(autorizador.instantes),
			})
		},
	}
	servicio, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad,
		autorizador,
		sesion,
		reloj,
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = servicio.Consultar(
		context.Background(),
		entorno.detalle,
	); err != nil {
		t.Fatalf("consultar detalle con cronología válida: %v", err)
	}
	if !emitidaEn.After(instanteAutorizacion) ||
		len(autorizador.instantes) != 1 ||
		!autorizador.instantes[0].Equal(instanteAutorizacion) ||
		reloj.lecturas != 2 || sesion.llamadas != 1 ||
		len(sesion.ordenes) != 1 ||
		!sesion.ordenes[0].Instante().Equal(instanteOrden) {
		t.Fatalf(
			"cronología distinta: autorización=%v orden=%v lecturas=%d sesión=%d",
			autorizador.instantes,
			sesion.ordenes,
			reloj.lecturas,
			sesion.llamadas,
		)
	}
	if len(observaciones) != 2 ||
		observaciones[0] != (observacion{autoridad: 1}) ||
		observaciones[1] != (observacion{
			autoridad: 1, autorizador: 1,
		}) {
		t.Fatalf("orden de fronteras distinto: %#v", observaciones)
	}
}

func TestConsultaDetalleRRHHRechazaCronologiaInseguraSinSesion(
	t *testing.T,
) {
	t.Parallel()
	zona := time.FixedZone("no-utc", 60*60)
	for _, caso := range []struct {
		nombre         string
		instantes      func(ports.CapacidadConsultaRRHH) []time.Time
		autorizaciones int
		lecturas       int
	}{
		{
			nombre: "retroceso",
			instantes: func(c ports.CapacidadConsultaRRHH) []time.Time {
				return []time.Time{
					c.ValidaDesde(),
					c.ValidaDesde().Add(-time.Microsecond),
				}
			},
			autorizaciones: 1,
			lecturas:       2,
		},
		{
			nombre: "capacidad vencida",
			instantes: func(c ports.CapacidadConsultaRRHH) []time.Time {
				return []time.Time{c.ValidaDesde(), c.ValidaHasta()}
			},
			autorizaciones: 1,
			lecturas:       2,
		},
		{
			nombre: "primer instante cero",
			instantes: func(ports.CapacidadConsultaRRHH) []time.Time {
				return []time.Time{{}}
			},
			lecturas: 1,
		},
		{
			nombre: "primer instante no UTC",
			instantes: func(c ports.CapacidadConsultaRRHH) []time.Time {
				return []time.Time{c.ValidaDesde().In(zona)}
			},
			lecturas: 1,
		},
		{
			nombre: "segundo instante cero",
			instantes: func(c ports.CapacidadConsultaRRHH) []time.Time {
				return []time.Time{c.ValidaDesde(), {}}
			},
			autorizaciones: 1,
			lecturas:       2,
		},
		{
			nombre: "segundo instante no canónico",
			instantes: func(c ports.CapacidadConsultaRRHH) []time.Time {
				return []time.Time{
					c.ValidaDesde(),
					c.ValidaDesde().Add(time.Nanosecond),
				}
			},
			autorizaciones: 1,
			lecturas:       2,
		},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			autorizador := &autorizadorCronologiaDetalleRRHHPrueba{
				capacidad: entorno.autorizador.capacidadDetalle,
			}
			sesion := &sesionCronologiaDetalleRRHHPrueba{
				detalle: entorno.sesion.detalle,
			}
			reloj := &relojSecuencialDetalleRRHHPrueba{
				instantes: caso.instantes(autorizador.capacidad),
			}
			servicio, err := NuevoServicioConsultaDetalleRRHH(
				entorno.autoridad,
				autorizador,
				sesion,
				reloj,
			)
			if err != nil {
				t.Fatal(err)
			}

			_, err = servicio.Consultar(context.Background(), entorno.detalle)
			if !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("cronología insegura no cerrada: %v", err)
			}
			if len(autorizador.instantes) != caso.autorizaciones ||
				reloj.lecturas != caso.lecturas || sesion.llamadas != 0 {
				t.Fatalf(
					"se alcanzó una frontera posterior: autorización=%d reloj=%d sesión=%d",
					len(autorizador.instantes),
					reloj.lecturas,
					sesion.llamadas,
				)
			}
		})
	}
}

func TestConsultaDetalleRRHHCancelacionCronologicaNoInvocaSesion(
	t *testing.T,
) {
	t.Parallel()
	for _, frontera := range []string{
		"resolución",
		"autorización",
		"segundo reloj",
	} {
		frontera := frontera
		t.Run(frontera, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			autorizador := &autorizadorCronologiaDetalleRRHHPrueba{
				capacidad: entorno.autorizador.capacidadDetalle,
			}
			sesion := &sesionCronologiaDetalleRRHHPrueba{
				detalle: entorno.sesion.detalle,
			}
			reloj := &relojSecuencialDetalleRRHHPrueba{
				instantes: []time.Time{
					autorizador.capacidad.ValidaDesde(),
					autorizador.capacidad.ValidaDesde(),
				},
			}
			switch frontera {
			case "resolución":
				entorno.autoridad.cancelar = cancelar
			case "autorización":
				autorizador.cancelar = cancelar
			case "segundo reloj":
				reloj.despuesDe = func(lectura int) {
					if lectura == 2 {
						cancelar()
					}
				}
			}
			servicio, err := NuevoServicioConsultaDetalleRRHH(
				entorno.autoridad,
				autorizador,
				sesion,
				reloj,
			)
			if err != nil {
				t.Fatal(err)
			}

			_, err = servicio.Consultar(ctx, entorno.detalle)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelación no prevalece: %v", err)
			}
			if sesion.llamadas != 0 {
				t.Fatal("la cancelación permitió invocar la sesión")
			}
		})
	}
}
