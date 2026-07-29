package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type sesionCronologiaDetalleRRHHPrueba struct {
	t        *testing.T
	detalle  ports.DetalleExpedienteRRHH
	ordenes  []ports.OrdenConsultaDetalleRRHH
	llamadas int
}

func (*sesionCronologiaDetalleRRHHPrueba) ConsultarCuadroYRegistrar(
	context.Context,
	ports.OrdenConsultaCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	return ports.PaginaCuadroRRHH{}, ports.ErrConsultaRRHHNoDisponible
}

func (s *sesionCronologiaDetalleRRHHPrueba) ConsultarDetalleYRegistrar(
	_ context.Context,
	orden ports.OrdenConsultaDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	s.t.Helper()
	s.llamadas++
	s.ordenes = append(s.ordenes, orden)
	detalle := s.detalle.Clonar()
	detalle.Lectura = reciboConsultaRRHHPrueba(
		s.t, orden.Contexto(), orden.Capacidad(), orden.Instante(),
		orden.Solicitud().ExpedienteRef(), detalle.Resumen.Version, 1,
	)
	return detalle, nil
}

func TestConsultaDetalleRRHHLeeRelojSoloTrasEmitirMaterial(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	reloj := &relojConsultaRRHHPrueba{
		instantes: []time.Time{entorno.ahora, entorno.ahora.Add(time.Second)},
	}
	lecturasAlEmitir := -1
	entorno.emision.detalle.cancelar = func() {
		lecturasAlEmitir = reloj.llamadas
	}
	sesion := &sesionCronologiaDetalleRRHHPrueba{
		t: t, detalle: entorno.sesion.detalle,
	}
	servicio, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad, entorno.emisor, sesion, reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = servicio.Consultar(
		context.Background(), entorno.detalle,
	); err != nil {
		t.Fatalf("consultar detalle con cronología válida: %v", err)
	}
	if lecturasAlEmitir != 0 || reloj.llamadas != 2 ||
		entorno.emision.detalle.llamadas != 1 ||
		entorno.emision.cuadro.llamadas != 0 ||
		sesion.llamadas != 1 || len(sesion.ordenes) != 1 ||
		sesion.ordenes[0].Instante() != entorno.ahora.Add(time.Second) {
		t.Fatalf(
			"fronteras desordenadas: antes_emisión=%d reloj=%d "+
				"detalle=%d cuadro=%d sesión=%d",
			lecturasAlEmitir, reloj.llamadas,
			entorno.emision.detalle.llamadas,
			entorno.emision.cuadro.llamadas, sesion.llamadas,
		)
	}
}

func TestConsultaDetalleRRHHRechazaRelojesPosterioresInseguros(
	t *testing.T,
) {
	t.Parallel()
	zona := time.FixedZone("no-utc", 60*60)
	for _, caso := range []struct {
		nombre    string
		instantes func(time.Time) []time.Time
		lecturas  int
	}{
		{
			nombre: "capacidad_cero",
			instantes: func(time.Time) []time.Time {
				return []time.Time{{}}
			},
			lecturas: 1,
		},
		{
			nombre: "capacidad_no_utc",
			instantes: func(base time.Time) []time.Time {
				return []time.Time{base.In(zona)}
			},
			lecturas: 1,
		},
		{
			nombre: "capacidad_retrocede",
			instantes: func(base time.Time) []time.Time {
				return []time.Time{base.Add(-time.Microsecond)}
			},
			lecturas: 1,
		},
		{
			nombre: "capacidad_tras_expiracion",
			instantes: func(base time.Time) []time.Time {
				return []time.Time{
					base.Add(ports.DuracionMaximaCapacidadConsultaRRHH),
				}
			},
			lecturas: 1,
		},
		{
			nombre: "orden_retrocede",
			instantes: func(base time.Time) []time.Time {
				return []time.Time{base, base.Add(-time.Microsecond)}
			},
			lecturas: 2,
		},
		{
			nombre: "orden_no_canonica",
			instantes: func(base time.Time) []time.Time {
				return []time.Time{base, base.Add(time.Nanosecond)}
			},
			lecturas: 2,
		},
		{
			nombre: "capacidad_vencida",
			instantes: func(base time.Time) []time.Time {
				return []time.Time{
					base, base.Add(ports.DuracionMaximaCapacidadConsultaRRHH),
				}
			},
			lecturas: 2,
		},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			reloj := &relojConsultaRRHHPrueba{
				instantes: caso.instantes(entorno.ahora),
			}
			sesion := &sesionCronologiaDetalleRRHHPrueba{
				t: t, detalle: entorno.sesion.detalle,
			}
			servicio, err := NuevoServicioConsultaDetalleRRHH(
				entorno.autoridad, entorno.emisor, sesion, reloj,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Consultar(context.Background(), entorno.detalle)
			if !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("cronología insegura aceptada: %v", err)
			}
			if reloj.llamadas != caso.lecturas || sesion.llamadas != 0 {
				t.Fatalf(
					"actividad posterior: reloj=%d sesión=%d",
					reloj.llamadas, sesion.llamadas,
				)
			}
		})
	}
}

func TestConsultaDetalleRRHHCanceladaTrasCadaFronteraNoAbreSesion(
	t *testing.T,
) {
	t.Parallel()
	for _, frontera := range []string{
		"autoridad", "motivo", "correlación", "emisión", "capacidad", "orden",
	} {
		frontera := frontera
		t.Run(frontera, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			reloj := &relojConsultaRRHHPrueba{
				instantes: []time.Time{entorno.ahora, entorno.ahora},
			}
			switch frontera {
			case "autoridad":
				entorno.autoridad.cancelar = cancelar
			case "motivo":
				entorno.emision.motivos.cancelarDetalle = cancelar
			case "correlación":
				entorno.emision.correlaciones.cancelar = cancelar
			case "emisión":
				entorno.emision.detalle.cancelar = cancelar
			case "capacidad":
				reloj.despuesDe = func(lectura int) {
					if lectura == 1 {
						cancelar()
					}
				}
			case "orden":
				reloj.despuesDe = func(lectura int) {
					if lectura == 2 {
						cancelar()
					}
				}
			}
			sesion := &sesionCronologiaDetalleRRHHPrueba{
				t: t, detalle: entorno.sesion.detalle,
			}
			servicio, err := NuevoServicioConsultaDetalleRRHH(
				entorno.autoridad, entorno.emisor, sesion, reloj,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Consultar(ctx, entorno.detalle)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelación no prevalece: %v", err)
			}
			if sesion.llamadas != 0 {
				t.Fatal("la cancelación permitió abrir sesión")
			}
		})
	}
}

func TestConsultaDetalleRRHHRechazaMaterialFallidoSinFiltrarCausa(
	t *testing.T,
) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	privado := errors.New("SECRETO-PRIVADO-EMISION-DETALLE")
	entorno.emision.detalle.err = privado
	servicio, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Consultar(context.Background(), entorno.detalle)
	if !errors.Is(err, ErrConsultaRRHHNoDisponible) ||
		errors.Is(err, privado) ||
		entorno.sesion.llamadasDetalle != 0 ||
		entorno.reloj.llamadas != 0 {
		t.Fatalf("fallo durable no cerrado: error=%v sesión=%d reloj=%d",
			err, entorno.sesion.llamadasDetalle, entorno.reloj.llamadas)
	}
}
