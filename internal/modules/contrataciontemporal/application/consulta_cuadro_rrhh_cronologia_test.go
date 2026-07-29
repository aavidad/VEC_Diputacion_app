package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

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
		s.t, orden.Contexto(), orden.Capacidad(), orden.Instante(),
		"", 0, uint16(len(s.expedientes)),
	)
	return ports.PaginaCuadroRRHH{
		GeneradaEn: orden.Instante(),
		Expedientes: append(
			[]ports.ResumenExpedienteRRHH(nil), s.expedientes...,
		),
		Lectura: recibo,
	}, nil
}

func (*sesionCronologiaCuadroPrueba) ConsultarDetalleYRegistrar(
	context.Context,
	ports.OrdenConsultaDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
}

func TestConsultaCuadroRRHHLeeRelojSoloTrasEmitirMaterial(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	reloj := &relojConsultaRRHHPrueba{
		instantes: []time.Time{entorno.ahora, entorno.ahora.Add(time.Second)},
	}
	lecturasAlEmitir := -1
	entorno.emision.cuadro.cancelar = func() {
		lecturasAlEmitir = reloj.llamadas
	}
	sesion := &sesionCronologiaCuadroPrueba{
		t: t, expedientes: entorno.sesion.pagina.Expedientes,
	}
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.emisor, sesion, reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = servicio.Consultar(
		context.Background(), entorno.cuadro,
	); err != nil {
		t.Fatalf("consultar con cronología válida: %v", err)
	}
	if lecturasAlEmitir != 0 || reloj.llamadas != 2 ||
		entorno.emision.cuadro.llamadas != 1 ||
		entorno.emision.detalle.llamadas != 0 ||
		sesion.llamadas != 1 ||
		sesion.instante != entorno.ahora.Add(time.Second) {
		t.Fatalf(
			"fronteras desordenadas: antes_emisión=%d reloj=%d "+
				"cuadro=%d detalle=%d sesión=%d instante=%v",
			lecturasAlEmitir, reloj.llamadas,
			entorno.emision.cuadro.llamadas,
			entorno.emision.detalle.llamadas,
			sesion.llamadas, sesion.instante,
		)
	}
}

func TestConsultaCuadroRRHHRechazaRelojesPosterioresInseguros(
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
			sesion := &sesionCronologiaCuadroPrueba{
				t: t, expedientes: entorno.sesion.pagina.Expedientes,
			}
			servicio, err := NuevoServicioConsultaCuadroRRHH(
				entorno.autoridad, entorno.emisor, sesion, reloj,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Consultar(context.Background(), entorno.cuadro)
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

func TestConsultaCuadroRRHHCanceladaTrasCadaFronteraNoAbreSesion(
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
				entorno.emision.motivos.cancelarCuadro = cancelar
			case "correlación":
				entorno.emision.correlaciones.cancelar = cancelar
			case "emisión":
				entorno.emision.cuadro.cancelar = cancelar
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
			sesion := &sesionCronologiaCuadroPrueba{
				t: t, expedientes: entorno.sesion.pagina.Expedientes,
			}
			servicio, err := NuevoServicioConsultaCuadroRRHH(
				entorno.autoridad, entorno.emisor, sesion, reloj,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Consultar(ctx, entorno.cuadro)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelación no prevalece: %v", err)
			}
			if sesion.llamadas != 0 {
				t.Fatal("la cancelación permitió abrir sesión")
			}
		})
	}
}

func TestConsultaCuadroRRHHRechazaMaterialFallidoSinFiltrarCausa(
	t *testing.T,
) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	privado := errors.New("SECRETO-PRIVADO-EMISION-CUADRO")
	entorno.emision.cuadro.err = privado
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Consultar(context.Background(), entorno.cuadro)
	if !errors.Is(err, ErrConsultaRRHHNoDisponible) ||
		errors.Is(err, privado) ||
		entorno.sesion.llamadasCuadro != 0 ||
		entorno.reloj.llamadas != 0 {
		t.Fatalf("fallo durable no cerrado: error=%v sesión=%d reloj=%d",
			err, entorno.sesion.llamadasCuadro, entorno.reloj.llamadas)
	}
}
