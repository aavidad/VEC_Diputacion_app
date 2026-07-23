package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestSolicitudRegistroNoAceptaMaterialHMACAportado(t *testing.T) {
	tipo := reflect.TypeOf(SolicitudRegistrarExpediente{})
	for _, nombre := range []string{
		"IdentidadesHMAC",
		"AmbitosIdempotenciaHMAC",
		"HuellasPeticionHMAC",
		"AmbitoIdempotenciaHMAC",
		"HuellaPeticionHMAC",
		"GeneracionHMAC",
	} {
		if _, existe := tipo.FieldByName(nombre); existe {
			t.Fatalf("la entrada cliente acepta material HMAC mediante %s", nombre)
		}
	}
}

func TestSolicitudRegistroNoAceptaInstanteConfirmacionDelCliente(
	t *testing.T,
) {
	if _, existe := reflect.TypeOf(SolicitudRegistrarExpediente{}).
		FieldByName("InstanteConfirmacion"); existe {
		t.Fatal("el DTO del cliente controla el instante de confirmación")
	}
}

func TestRegistroSolicitudRechazaRetrocesoYBordeExclusivoDeDecision(
	t *testing.T,
) {
	escenario := nuevoEscenarioRegistro(t)
	casos := map[string]time.Time{
		"retroceso": escenario.instante.Add(-time.Microsecond),
		"borde exclusivo": escenario.instante.Add(
			90 * time.Second,
		),
	}
	for nombre, instante := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, d := construirServicioRegistro(t, escenario)
			d.autorizador.antes = func() {
				d.reloj.fijar(instante)
			}
			_, err := servicio.Registrar(
				context.Background(),
				escenario.solicitud,
			)
			if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
				d.transaccion.llamadas != 0 {
				t.Fatalf("instante fuera de ventana confirmó: %v", err)
			}
		})
	}
}

type mutacionLigaduraV3Prueba func(
	*dominiovec.DatosSolicitudAutorizacionLigadaV3,
) (
	dominiovec.ResultadoContextoActorRegistradoV2,
	dominiovec.ReferenciaEntradaCatalogo,
)

func mutacionRecursoV3Prueba(
	escenario escenarioRegistro,
	mutar func(*dominiovec.DatosSolicitudAutorizacionLigadaV3),
) mutacionLigaduraV3Prueba {
	return func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
		dominiovec.ResultadoContextoActorRegistradoV2,
		dominiovec.ReferenciaEntradaCatalogo,
	) {
		mutar(datos)
		return escenario.contexto.Resultado, escenario.motivo
	}
}

func TestRegistroSolicitudCompletaMatrizAdversarialRecursoV3(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	casos := map[string]mutacionLigaduraV3Prueba{
		"modulo": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.ModuloID = "modulo_ajeno"
		}),
		"tipo": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.Tipo = "expediente_ajeno"
		}),
		"centro": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.Ambitos["centro_ref"] = "centro:ajeno"
		}),
		"categoria": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.Ambitos["categoria_ref"] = "categoria:ajena"
		}),
		"version flujo": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.Atributos["flujo_version"] = "2"
		}),
		"huella flujo": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.Atributos["flujo_huella_sha256"] =
				strings.Repeat("f", 64)
		}),
		"huella peticion activa": mutacionRecursoV3Prueba(
			escenario,
			func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				datos.Recurso.Atributos[ports.AtributoHuellaPeticionHMACActiva] = selloHMACRegistroPrueba(
					"vec.contratacion-temporal.huella-peticion/v2",
					"e",
				)
			},
		),
		"huella efecto": mutacionRecursoV3Prueba(
			escenario,
			func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				datos.Recurso.Atributos[ports.AtributoHuellaEfectoAltaSHA256] =
					strings.Repeat("e", 64)
			},
		),
		"sin huella efecto": mutacionRecursoV3Prueba(
			escenario,
			func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				delete(
					datos.Recurso.Atributos,
					ports.AtributoHuellaEfectoAltaSHA256,
				)
			},
		),
		"ambito extra": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.Ambitos["ambito_extra"] = "valor"
		}),
		"atributo extra": mutacionRecursoV3Prueba(escenario, func(
			datos *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			datos.Recurso.Atributos["atributo_extra"] = "valor"
		}),
	}

	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, d := construirServicioRegistro(t, escenario)
			d.autorizador.transformar = func(
				original dominiovec.SolicitudAutorizacionLigadaV3,
				_ dominiovec.ResultadoContextoActorRegistradoV2,
			) (
				dominiovec.SolicitudAutorizacionLigadaV3,
				dominiovec.ResultadoContextoActorRegistradoV2,
				dominiovec.ReferenciaEntradaCatalogo,
			) {
				datos, err := original.Datos()
				if err != nil {
					t.Fatal(err)
				}
				resultado, motivo := mutar(&datos)
				alterada, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
					datos,
				)
				if err != nil {
					t.Fatal(err)
				}
				return alterada, resultado, motivo
			}

			_, err := servicio.Registrar(
				context.Background(),
				escenario.solicitud,
			)
			if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
				d.transaccion.llamadas != 0 {
				t.Fatalf("ligadura %s cruzada produjo efecto: %v", nombre, err)
			}
		})
	}
}

func TestRegistroSolicitudRechazaDecisionV3Denegada(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	d.autorizador.decisionDenegada = true

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("decisión denegada produjo reserva o efecto: %v", err)
	}
}

func TestOrdenAltaCotejaHuellaActivaComprometidaEnV3(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	if _, err := servicio.Registrar(
		context.Background(),
		escenario.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	evidencia, err := d.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosHuellas, err := evidencia.HuellasPeticionHMAC.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosHuellas.Activo.Valor = selloHMACRegistroPrueba(
		"vec.contratacion-temporal.huella-peticion/v2",
		"e",
	)
	retenidas := make([]string, 0, len(datosHuellas.Retenidos))
	for _, retenida := range datosHuellas.Retenidos {
		retenidas = append(retenidas, retenida.Valor)
	}
	huellasCruzadas, err := ports.NuevaColeccionSellosHMAC(
		datosHuellas.Activo.Valor,
		retenidas,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ports.NuevaOrdenConfirmarAlta(ports.DatosOrdenConfirmarAlta{
		Expediente:              evidencia.Expediente,
		SolicitudAutorizacionV3: evidencia.SolicitudAutorizacionV3,
		DecisionAutorizacionV3:  evidencia.DecisionAutorizacionV3,
		ConfirmacionRegistroV3:  evidencia.ConfirmacionRegistroV3,
		ResultadoContextoV2:     evidencia.ResultadoContextoV2,
		AmbitosIdempotenciaHMAC: evidencia.AmbitosIdempotenciaHMAC,
		HuellasPeticionHMAC:     huellasCruzadas,
		Candidatura:             evidencia.Candidatura,
		ProyeccionEfecto:        evidencia.ProyeccionEfecto,
		InstanteConfirmacion:    evidencia.InstanteConfirmacion,
	})
	if !errors.Is(err, ports.ErrOrdenAltaInvalida) {
		t.Fatalf("orden aceptó otra huella activa: %v", err)
	}
}

func TestRegistroSolicitudRechazaReciboNoConfiable(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	d.transaccion.recibo.ExpedienteRef = "expediente:otro-0001"

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoRegistroNoConfiable) {
		t.Fatalf("recibo ajeno aceptado: %v", err)
	}
}

func TestRegistroSolicitudRechazaCandidatosInvalidosAntesDelPDP(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	casos := map[string]func(*generadorAltaDoble){
		"referencias": func(g *generadorAltaDoble) {
			g.referencias.ExpedienteRef = ""
		},
		"reserva": func(g *generadorAltaDoble) {
			g.reservaRef = ""
		},
		"fallo generador": func(g *generadorAltaDoble) {
			g.errReferencias = errors.New("entropía no disponible")
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, d := construirServicioRegistro(t, escenario)
			mutar(d.referencias)

			_, err := servicio.Registrar(
				context.Background(),
				escenario.solicitud,
			)
			if !errors.Is(err, ports.ErrPreparacionAltaInvalida) ||
				d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
				t.Fatalf("candidato %s produjo autoridad o efecto: %v", nombre, err)
			}
		})
	}
}
