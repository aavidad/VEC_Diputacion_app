package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

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
				d.preparaciones.llamadas != 0 ||
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
		d.preparaciones.llamadas != 0 || d.transaccion.llamadas != 0 {
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
		AmbitosIdempotenciaHMAC: evidencia.AmbitosIdempotenciaHMAC,
		HuellasPeticionHMAC:     huellasCruzadas,
		Preparacion:             evidencia.Preparacion,
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

func TestRegistroSolicitudRechazaEstadosPreparacionInvalidos(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	recibo := escenario.recibo
	casos := map[string]func(*ports.PreparacionAlta){
		"desconocido": func(p *ports.PreparacionAlta) {
			p.Estado = "estado_ajeno"
		},
		"confirmada sin recibo": func(p *ports.PreparacionAlta) {
			p.Estado = ports.PreparacionConfirmada
		},
		"reservada con recibo": func(p *ports.PreparacionAlta) {
			p.ReciboConfirmado = &recibo
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			caso := escenario
			mutar(&caso.preparacion)
			servicio, d := construirServicioRegistro(t, caso)

			_, err := servicio.Registrar(context.Background(), caso.solicitud)
			if !errors.Is(err, ports.ErrPreparacionAltaInvalida) ||
				d.transaccion.llamadas != 0 {
				t.Fatalf("estado %s produjo efecto: %v", nombre, err)
			}
		})
	}
}
