package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestCTLITEO702ALigaParesActorYCorrelacionSinTraducirlos(t *testing.T) {
	servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
	datosV3, err := contextos.contexto.SolicitudAutorizacionV3.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datosV3.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacionV3, err := datosV3.Correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	preparacion := contextos.contexto.PreparacionSeguimiento
	atributos := datosV3.Recurso.Atributos
	if vinculo.PrincipalID == preparacion.ActorRef ||
		correlacionV3 == preparacion.CorrelacionRef ||
		atributos["principal_v3_ref"] != vinculo.PrincipalID ||
		atributos["actor_seguimiento_ref"] != preparacion.ActorRef ||
		atributos["correlacion_v3_ref"] != correlacionV3 ||
		atributos["correlacion_seguimiento_ref"] != preparacion.CorrelacionRef {
		t.Fatalf("pares V3/Seguimiento no ligados exactamente: %#v", atributos)
	}
	recibo, err := servicio.Confirmar(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if recibo.ActorRef != preparacion.ActorRef ||
		recibo.CorrelacionRef != preparacion.CorrelacionRef {
		t.Fatalf("el efecto no uso la preparacion local: %#v", recibo)
	}
	llamadas, efectos, _ := transaccion.estado()
	if llamadas != 1 || efectos != 1 {
		t.Fatalf("confirmacion nominal inesperada: llamadas=%d efectos=%d", llamadas, efectos)
	}
}

func TestCTLITEO702ADeniegaCadaEjeDelRecursoAntesDelCallback(t *testing.T) {
	otra := referenciaIncorporacionPrueba("otro_eje_autorizacion")
	casos := map[string]func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		"accion": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Accion = "contratacion_temporal.incorporacion.consultar"
		},
		"finalidad": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Finalidad = "consultar_incorporacion"
		},
		"referencia": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Referencia = otra
		},
		"modulo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.ModuloID = "contratacion_temporal_cruzada"
		},
		"tipo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Tipo = ports.TipoRecursoExpediente
		},
		"organizacion": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos["organizacion_ref"] = otra
		},
		"unidad": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos["unidad_ref"] = otra
		},
		"resultado": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["resultado_personal_ref"] = otra
		},
		"relacion": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["relacion_ref"] = otra
		},
		"ocupacion": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["ocupacion_ref"] = otra
		},
		"version expediente": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["version_expediente_esperada"] = "8"
		},
		"version seguimiento": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["version_seguimiento_esperada"] = "1"
		},
		"principal V3": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["principal_v3_ref"] = "per_0123456789abcdefghijklz"
		},
		"actor seguimiento": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["actor_seguimiento_ref"] = otra
		},
		"correlacion V3": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["correlacion_v3_ref"] =
				"correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		},
		"correlacion seguimiento": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["correlacion_seguimiento_ref"] = otra
		},
		"motivo V3": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = dominiovec.ReferenciaEntradaCatalogo{
				CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
				CatalogoHuellaSHA256: strings.Repeat("d", 64),
				EntradaClave:         "motivo_fedcba9876543210fedcba9876543210",
			}
		},
		"ambito extra": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos["ambito_extra"] = otra
		},
		"atributo extra": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos["atributo_extra"] = otra
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
			contextos.contexto = contextoConfirmacionIncorporacionV3Prueba(
				t, solicitud.datos(), instanteConfirmacionIncorporacionPrueba, mutar,
			)
			confirmacionDenegadaAntesDelCallback(t, servicio, transaccion, solicitud)
		})
	}
}

func TestCTLITEO702ADeniegaPreparacionVinculoYVentanaAntesDelCallback(t *testing.T) {
	casos := map[string]func(*testing.T, *ports.ContextoConfirmacionIncorporacion, SolicitudConfirmarIncorporacion){
		"organizacion preparada": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.PreparacionSeguimiento.OrganizacionRef = referenciaIncorporacionPrueba("organizacion_cruzada")
		},
		"unidad preparada": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.PreparacionSeguimiento.UnidadRef = referenciaIncorporacionPrueba("unidad_cruzada")
		},
		"actor preparado": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.PreparacionSeguimiento.ActorRef = referenciaIncorporacionPrueba("actor_cruzado")
		},
		"correlacion preparada": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.PreparacionSeguimiento.CorrelacionRef = referenciaIncorporacionPrueba("correlacion_cruzada")
		},
		"perfil": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.SolicitudContexto.PerfilRef = "prf_0123456789abcdefghijklmnb"
		},
		"vinculo": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.ContextoAutorizacion = contextoAutorizacionAltaV3PruebaConMarcas(
				t, instanteConfirmacionIncorporacionPrueba, "b", "b",
			)
		},
		"garantia": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, s SolicitudConfirmarIncorporacion) {
			*c = contextoConfirmacionIncorporacionV3ConAutoridadPrueba(
				t, s.datos(), instanteConfirmacionIncorporacionPrueba,
				contextoAutorizacionAltaV3SustancialPrueba(t, instanteConfirmacionIncorporacionPrueba), nil,
			)
		},
	}
	for nombre, cruzar := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
			cruzar(t, &contextos.contexto, solicitud)
			confirmacionDenegadaAntesDelCallback(t, servicio, transaccion, solicitud)
		})
	}

	t.Run("correlacion V3", func(t *testing.T) {
		servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		contextos.contexto = contextoConfirmacionIncorporacionV3Prueba(
			t, solicitud.datos(), instanteConfirmacionIncorporacionPrueba,
			func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				otra, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
					context.Background(),
					&generadorReferenciasDoble{correlacion: "correlacion_fedcba9876543210fedcba9876543210"},
				)
				if err != nil {
					t.Fatal(err)
				}
				d.Correlacion = otra
			},
		)
		confirmacionDenegadaAntesDelCallback(t, servicio, transaccion, solicitud)
	})

	t.Run("vigencia", func(t *testing.T) {
		_, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		servicio, err := NuevoServicioConfirmacionIncorporacion(
			contextos,
			&relojMutable{instante: instanteConfirmacionIncorporacionPrueba.Add(91 * time.Second)},
			transaccion,
		)
		if err != nil {
			t.Fatal(err)
		}
		confirmacionDenegadaAntesDelCallback(t, servicio, transaccion, solicitud)
	})
}

func TestCTLITEO702ADeniegaDecisionYConfirmacionAusentesOCruzadas(t *testing.T) {
	casos := map[string]func(*testing.T, *ports.ContextoConfirmacionIncorporacion, SolicitudConfirmarIncorporacion){
		"decision ausente": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.DecisionAutorizacionV3 = dominiovec.DecisionAutorizacionLigadaV3{}
		},
		"confirmacion ausente": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.ConfirmacionRegistroV3 = puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
		},
		"decision cruzada": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, s SolicitudConfirmarIncorporacion) {
			otro := contextoConfirmacionIncorporacionV3Prueba(
				t, s.datos(), instanteConfirmacionIncorporacionPrueba,
				func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
					d.Recurso.Atributos["version_expediente_esperada"] = strconv.FormatUint(8, 10)
				},
			)
			c.DecisionAutorizacionV3 = otro.DecisionAutorizacionV3
		},
		"confirmacion cruzada": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, s SolicitudConfirmarIncorporacion) {
			otro := contextoConfirmacionIncorporacionV3Prueba(
				t, s.datos(), instanteConfirmacionIncorporacionPrueba,
				func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
					d.Recurso.Atributos["version_expediente_esperada"] = strconv.FormatUint(8, 10)
				},
			)
			c.ConfirmacionRegistroV3 = otro.ConfirmacionRegistroV3
		},
	}
	for nombre, cruzar := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
			cruzar(t, &contextos.contexto, solicitud)
			confirmacionDenegadaAntesDelCallback(t, servicio, transaccion, solicitud)
		})
	}
}

func confirmacionDenegadaAntesDelCallback(
	t *testing.T,
	servicio *ServicioConfirmacionIncorporacion,
	transaccion *transaccionIncorporacionPrueba,
	solicitud SolicitudConfirmarIncorporacion,
) {
	t.Helper()
	recibo, err := servicio.Confirmar(context.Background(), solicitud)
	if !errors.Is(err, ErrConfirmacionIncorporacionDenegada) ||
		!reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
		t.Fatalf("cruce de autorizacion aceptado: recibo=%#v error=%v", recibo, err)
	}
	llamadas, efectos, _ := transaccion.estado()
	if llamadas != 0 || efectos != 0 {
		t.Fatalf("el cruce alcanzo el callback: llamadas=%d efectos=%d", llamadas, efectos)
	}
}

func TestCTLITEO702AReciboValidaTodosLosEjesDeLaOperacion(t *testing.T) {
	otra := referenciaIncorporacionPrueba("recibo_eje_cruzado")
	casos := map[string]func(*ports.ReciboConfirmacionIncorporacion){
		"correlacion": func(r *ports.ReciboConfirmacionIncorporacion) { r.CorrelacionRef = otra },
		"actor":       func(r *ports.ReciboConfirmacionIncorporacion) { r.ActorRef = otra },
		"organizacion": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.OrganizacionRef = otra
		},
		"unidad": func(r *ports.ReciboConfirmacionIncorporacion) { r.UnidadRef = otra },
		"expediente": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ExpedienteRef = otra
		},
		"solicitud Personal": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.SolicitudPersonalRef = otra
		},
		"decision": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.DecisionAutorizacionRef = "dec_fedcba9876543210fedcba9876543210"
		},
		"resultado Personal": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ResultadoPersonalRef = otra
		},
		"recibo Personal": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ReciboPersonalRef = otra
		},
		"relacion":  func(r *ports.ReciboConfirmacionIncorporacion) { r.RelacionRef = otra },
		"ocupacion": func(r *ports.ReciboConfirmacionIncorporacion) { r.OcupacionRef = otra },
		"transicion": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.TransicionClave = "otra_transicion"
		},
		"motivo": func(r *ports.ReciboConfirmacionIncorporacion) { r.MotivoClave = "otro_motivo" },
		"version expediente": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.VersionExpediente++
		},
		"version anterior": func(r *ports.ReciboConfirmacionIncorporacion) { r.VersionAnterior++ },
		"version resultante": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.VersionResultante++
		},
		"fecha incorporacion": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.FechaIncorporacion = r.FechaIncorporacion.Add(time.Hour)
		},
		"fecha fin prevista": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.FechaFinPrevista = r.FechaFinPrevista.Add(time.Hour)
		},
		"confirmada en": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ConfirmadaEn = r.ConfirmadaEn.Add(91 * time.Second)
		},
		"documentos": func(r *ports.ReciboConfirmacionIncorporacion) {
			r.Documentos[0].Referencia = otra
		},
	}
	for nombre, adulterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, _, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
			transaccion.adulterar = adulterar
			recibo, err := servicio.Confirmar(context.Background(), solicitud)
			if !errors.Is(err, ErrResultadoConfirmacionIncorporacionNoConfiable) ||
				!reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
				t.Fatalf("recibo cruzado publicado: recibo=%#v error=%v", recibo, err)
			}
		})
	}
}

func TestCTLITEO702AClonaDocumentosDelReciboAntesDePublicarlo(t *testing.T) {
	servicio, _, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
	var documentosPuerto []domain.DocumentoSeguimiento
	transaccion.adulterar = func(r *ports.ReciboConfirmacionIncorporacion) {
		documentosPuerto = r.Documentos
	}
	recibo, err := servicio.Confirmar(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	esperado := recibo.Documentos[0]
	documentosPuerto[0] = domain.DocumentoSeguimiento{
		TipoClave: "documento_cruzado", Referencia: referenciaIncorporacionPrueba("documento_cruzado"),
	}
	if recibo.Documentos[0] != esperado {
		t.Fatalf("el recibo comparte backing array con el puerto: %#v", recibo.Documentos)
	}
}

type transaccionCanceladaDuranteCallback struct {
	iniciada chan struct{}
}

func (t *transaccionCanceladaDuranteCallback) ConfirmarIncorporacion(
	ctx context.Context,
	_ ports.OrdenConfirmarIncorporacion,
) (ports.ReciboConfirmacionIncorporacion, error) {
	close(t.iniciada)
	<-ctx.Done()
	return ports.ReciboConfirmacionIncorporacion{},
		fmt.Errorf("sql privado dsn=postgres://usuario:secreto@db: %w", ctx.Err())
}

type transaccionEfectoParcialSinRecibo struct {
	llamadas         int
	efectosParciales int
}

func (t *transaccionEfectoParcialSinRecibo) ConfirmarIncorporacion(
	context.Context,
	ports.OrdenConfirmarIncorporacion,
) (ports.ReciboConfirmacionIncorporacion, error) {
	t.llamadas++
	t.efectosParciales++
	return ports.ReciboConfirmacionIncorporacion{},
		errors.New("pq: relation rrhh_privada failed; dsn=postgres://secreto")
}

func TestCTLITEO702ARedactaFallosDelPuertoYPreservaCancelacion(t *testing.T) {
	t.Run("SQL y DSN", func(t *testing.T) {
		servicio, _, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		transaccion.fallo = errors.New(
			"pq: SELECT * FROM personal_privado; dsn=postgres://usuario:secreto@db/rrhh",
		)
		recibo, err := servicio.Confirmar(context.Background(), solicitud)
		if !errors.Is(err, ErrConfirmacionIncorporacionNoDisponible) ||
			strings.Contains(err.Error(), "SELECT") || strings.Contains(err.Error(), "postgres://") ||
			!reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
			t.Fatalf("fallo privado observable: recibo=%#v error=%v", recibo, err)
		}
	})

	t.Run("cancelacion durante callback", func(t *testing.T) {
		_, contextos, _, solicitud := escenarioConfirmacionIncorporacion(t)
		transaccion := &transaccionCanceladaDuranteCallback{iniciada: make(chan struct{})}
		servicio, err := NuevoServicioConfirmacionIncorporacion(
			contextos, &relojMutable{instante: instanteConfirmacionIncorporacionPrueba}, transaccion,
		)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancelar := context.WithCancel(context.Background())
		resultado := make(chan error, 1)
		go func() {
			recibo, err := servicio.Confirmar(ctx, solicitud)
			if !reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
				resultado <- fmt.Errorf("cancelacion publico recibo: %#v", recibo)
				return
			}
			resultado <- err
		}()
		<-transaccion.iniciada
		cancelar()
		err = <-resultado
		if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "postgres://") {
			t.Fatalf("cancelacion durante callback no saneada: %v", err)
		}
	})

	t.Run("deadline del puerto", func(t *testing.T) {
		servicio, _, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		transaccion.fallo = fmt.Errorf("dsn privado: %w", context.DeadlineExceeded)
		_, err := servicio.Confirmar(context.Background(), solicitud)
		if !errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "dsn") {
			t.Fatalf("deadline no preservado de forma opaca: %v", err)
		}
	})
}

func TestCTLITEO702AErrorTrasEfectoParcialNoConfirmaReciboNiReplay(t *testing.T) {
	_, contextos, _, solicitud := escenarioConfirmacionIncorporacion(t)
	transaccion := &transaccionEfectoParcialSinRecibo{}
	servicio, err := NuevoServicioConfirmacionIncorporacion(
		contextos, &relojMutable{instante: instanteConfirmacionIncorporacionPrueba}, transaccion,
	)
	if err != nil {
		t.Fatal(err)
	}
	for intento := 0; intento < 2; intento++ {
		recibo, err := servicio.Confirmar(context.Background(), solicitud)
		if !errors.Is(err, ErrConfirmacionIncorporacionNoDisponible) ||
			strings.Contains(err.Error(), "rrhh_privada") ||
			!reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
			t.Fatalf("fallo parcial publicado en intento %d: recibo=%#v error=%v", intento, recibo, err)
		}
	}
	if transaccion.llamadas != 2 || transaccion.efectosParciales != 2 {
		t.Fatalf("se confirmo recibo/replay tras efecto parcial: %#v", transaccion)
	}
}
