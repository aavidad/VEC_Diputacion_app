package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	instanteConfirmacionIncorporacionPrueba = time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	errTransaccionIncorporacionPrueba       = errors.New("fallo transaccional opaco")
)

type resolutorContextoIncorporacionPrueba struct {
	mu       sync.Mutex
	contexto ports.ContextoConfirmacionIncorporacion
	err      error
	llamadas int
}

func (r *resolutorContextoIncorporacionPrueba) ResolverContextoConfirmacionIncorporacion(
	ctx context.Context,
) (ports.ContextoConfirmacionIncorporacion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if err := ctx.Err(); err != nil {
		return ports.ContextoConfirmacionIncorporacion{}, err
	}
	return r.contexto, r.err
}

func (r *resolutorContextoIncorporacionPrueba) numeroLlamadas() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas
}

type transaccionIncorporacionPrueba struct {
	mu                sync.Mutex
	definicion        domain.DefinicionSeguimiento
	seguimiento       domain.Seguimiento
	versionExpediente uint64
	confirmadaEn      time.Time
	referencias       ports.ReferenciasDurablesConfirmacionIncorporacion
	fallo             error
	adulterar         func(*ports.ReciboConfirmacionIncorporacion)
	llamadas          int
	efectos           int
	ultima            ports.EvidenciaOrdenConfirmarIncorporacion
}

func (t *transaccionIncorporacionPrueba) ConfirmarIncorporacion(
	ctx context.Context,
	orden ports.OrdenConfirmarIncorporacion,
) (ports.ReciboConfirmacionIncorporacion, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.llamadas++
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	if t.fallo != nil {
		return ports.ReciboConfirmacionIncorporacion{}, t.fallo
	}
	evidencia, err := orden.Datos()
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	if err := orden.ValidarDentroDeTransaccion(t.confirmadaEn); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	t.ultima = evidencia
	datos := evidencia.Confirmacion
	solicitudV3, err := evidencia.Contexto.SolicitudAutorizacionV3.Datos()
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	confirmacionV3, err := evidencia.Contexto.ConfirmacionRegistroV3.Datos()
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	estado := t.seguimiento.Estado()
	if estado.OrganizacionRef != solicitudV3.Recurso.Ambitos["organizacion_ref"] ||
		estado.ExpedienteRef != datos.SolicitudPersonal.ExpedienteRef ||
		estado.RelacionRef != datos.ResultadoPersonal.RelacionRef ||
		datos.SolicitudPersonal.VersionExpediente != t.versionExpediente {
		return ports.ReciboConfirmacionIncorporacion{},
			errTransaccionIncorporacionPrueba
	}
	transicion, err := orden.DatosTransicionSeguimiento(
		t.confirmadaEn,
		t.referencias,
	)
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	versionAnterior := t.seguimiento.Version()
	siguiente, err := t.seguimiento.Aplicar(
		t.definicion,
		datos.VersionSeguimientoEsperada,
		transicion,
	)
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	if siguiente.Version() != versionAnterior {
		t.efectos++
	}
	t.seguimiento = siguiente
	recibo := ports.ReciboConfirmacionIncorporacion{
		ReciboRef:               t.referencias.ReciboRef,
		ActuacionRef:            t.referencias.ActuacionRef,
		CorrelacionRef:          t.referencias.CorrelacionRef,
		ActorRef:                t.referencias.ActorRef,
		DecisionAutorizacionRef: confirmacionV3.DecisionRef,
		ExpedienteRef:           datos.SolicitudPersonal.ExpedienteRef,
		ResultadoPersonalRef:    datos.ResultadoPersonal.ResultadoRef,
		ReciboPersonalRef:       datos.ResultadoPersonal.ReciboRef,
		RelacionRef:             datos.ResultadoPersonal.RelacionRef,
		OcupacionRef:            datos.ResultadoPersonal.OcupacionRef,
		TransicionClave:         ports.TransicionConfirmarIncorporacion,
		VersionAnterior:         datos.VersionSeguimientoEsperada,
		VersionResultante:       datos.VersionSeguimientoEsperada + 1,
		FechaIncorporacion:      datos.PeriodoIncorporacion.Desde,
		ConfirmadaEn:            t.confirmadaEn,
		Documentos: append(
			[]domain.DocumentoSeguimiento(nil),
			datos.Documentos...,
		),
	}
	if t.adulterar != nil {
		t.adulterar(&recibo)
	}
	return recibo, nil
}

func (t *transaccionIncorporacionPrueba) estado() (int, int, domain.Seguimiento) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.llamadas, t.efectos, t.seguimiento
}

func TestCTLITEO702AConfirmaIncorporacionYReproduceReplayExacto(t *testing.T) {
	servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)

	recibo, err := servicio.Confirmar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("confirmar incorporacion: %v", err)
	}
	repetido, err := servicio.Confirmar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("repetir confirmacion exacta: %v", err)
	}
	if !reflect.DeepEqual(recibo, repetido) {
		t.Fatalf("el replay no devolvio el mismo recibo: %#v / %#v", recibo, repetido)
	}
	llamadas, efectos, seguimiento := transaccion.estado()
	if llamadas != 2 || efectos != 1 || seguimiento.Version() != 1 ||
		len(seguimiento.Actuaciones()) != 1 {
		t.Fatalf(
			"replay no exacto: llamadas=%d efectos=%d version=%d actuaciones=%d",
			llamadas, efectos, seguimiento.Version(), len(seguimiento.Actuaciones()),
		)
	}
	actuacion := seguimiento.Actuaciones()[0]
	if actuacion.TransicionClave != ports.TransicionConfirmarIncorporacion ||
		actuacion.ActorRef != referenciaIncorporacionPrueba("actor_rrhh_0001") ||
		actuacion.UnidadRef != referenciaIncorporacionPrueba("unidad_rrhh_0001") ||
		!actuacion.EfectivoEn.Equal(solicitud.FechaIncorporacion) ||
		actuacion.ReciboRef != recibo.ReciboRef ||
		recibo.RelacionRef != solicitud.ResultadoPersonal.RelacionRef ||
		recibo.OcupacionRef != solicitud.ResultadoPersonal.OcupacionRef ||
		len(recibo.Documentos) != 2 || contextos.numeroLlamadas() != 2 {
		t.Fatalf("confirmacion incompleta: actuacion=%#v recibo=%#v", actuacion, recibo)
	}

	divergente := solicitud
	divergente.Documentos = append(
		[]domain.DocumentoSeguimiento(nil),
		solicitud.Documentos...,
	)
	divergente.Documentos[0].Referencia = referenciaIncorporacionPrueba("documento_incorporacion_divergente")
	if _, err := servicio.Confirmar(context.Background(), divergente); !errors.Is(
		err,
		domain.ErrActuacionSeguimientoEnConflicto,
	) {
		t.Fatalf("replay divergente aceptado: %v", err)
	}
	_, efectos, seguimiento = transaccion.estado()
	if efectos != 1 || seguimiento.Version() != 1 || len(seguimiento.Actuaciones()) != 1 {
		t.Fatal("la colision semantica produjo un segundo efecto")
	}
}

func TestCTLITEO702ARechazaEntradaAntesDeResolverAutoridad(t *testing.T) {
	_, contextoBase, transaccionBase, solicitudBase := escenarioConfirmacionIncorporacion(t)
	casos := map[string]func(*SolicitudConfirmarIncorporacion){
		"resultado rechazado": func(s *SolicitudConfirmarIncorporacion) {
			s.ResultadoPersonal.Estado = ports.AltaPersonalRPTRechazada
		},
		"resultado discordante": func(s *SolicitudConfirmarIncorporacion) {
			s.ResultadoPersonal.CorrelacionRef = "correlacion:personal:otra"
		},
		"sin ocupacion": func(s *SolicitudConfirmarIncorporacion) {
			s.ResultadoPersonal.OcupacionRef = ""
		},
		"periodo invalido": func(s *SolicitudConfirmarIncorporacion) {
			s.FechaFinPrevista = s.FechaIncorporacion
		},
		"sin documentos": func(s *SolicitudConfirmarIncorporacion) {
			s.Documentos = nil
		},
		"version agotada": func(s *SolicitudConfirmarIncorporacion) {
			s.VersionSeguimientoEsperada = ports.MaximoEnteroSeguroOperacionAnalisis
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			contextos := &resolutorContextoIncorporacionPrueba{contexto: contextoBase.contexto}
			transaccion := nuevaTransaccionIncorporacionPrueba(
				t,
				transaccionBase.definicion,
				solicitudBase.ResultadoPersonal.RelacionRef,
			)
			servicio, err := NuevoServicioConfirmacionIncorporacion(
				contextos,
				&relojMutable{instante: instanteConfirmacionIncorporacionPrueba},
				transaccion,
			)
			if err != nil {
				t.Fatalf("crear servicio: %v", err)
			}
			solicitud := solicitudBase
			solicitud.Documentos = append(
				[]domain.DocumentoSeguimiento(nil),
				solicitudBase.Documentos...,
			)
			mutar(&solicitud)
			recibo, err := servicio.Confirmar(context.Background(), solicitud)
			if !errors.Is(err, ErrSolicitudConfirmacionIncorporacionInvalida) ||
				!reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
				t.Fatalf("entrada invalida aceptada: recibo=%#v error=%v", recibo, err)
			}
			llamadas, efectos, _ := transaccion.estado()
			if contextos.numeroLlamadas() != 0 || llamadas != 0 || efectos != 0 {
				t.Fatal("una entrada invalida alcanzo autoridad o transaccion")
			}
		})
	}
}

func TestCTLITEO702ADeniegaCrucesDeAutorizacionNominalV3(t *testing.T) {
	casos := map[string]func(
		*testing.T,
		*ports.ContextoConfirmacionIncorporacion,
		SolicitudConfirmarIncorporacion,
	){
		"accion": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, s SolicitudConfirmarIncorporacion) {
			*c = contextoConfirmacionIncorporacionV3Prueba(
				t, s.datos(), instanteConfirmacionIncorporacionPrueba,
				func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
					d.Accion = "contratacion_temporal.incorporacion.consultar"
				},
			)
		},
		"recurso": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, s SolicitudConfirmarIncorporacion) {
			*c = contextoConfirmacionIncorporacionV3Prueba(
				t, s.datos(), instanteConfirmacionIncorporacionPrueba,
				func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
					d.Recurso.Tipo = ports.TipoRecursoExpediente
				},
			)
		},
		"finalidad": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, s SolicitudConfirmarIncorporacion) {
			*c = contextoConfirmacionIncorporacionV3Prueba(
				t, s.datos(), instanteConfirmacionIncorporacionPrueba,
				func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
					d.Finalidad = "consultar_incorporacion"
				},
			)
		},
		"perfil": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.SolicitudContexto.PerfilRef = "prf_0123456789abcdefghijklmnb"
		},
		"vinculo": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.ContextoAutorizacion = contextoAutorizacionAltaV3PruebaConMarcas(
				t, instanteConfirmacionIncorporacionPrueba, "b", "b",
			)
		},
		"concesion ausente": func(_ *testing.T, c *ports.ContextoConfirmacionIncorporacion, _ SolicitudConfirmarIncorporacion) {
			c.DecisionAutorizacionV3 = dominiovec.DecisionAutorizacionLigadaV3{}
			c.ConfirmacionRegistroV3 = puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
		},
		"correlacion": func(t *testing.T, c *ports.ContextoConfirmacionIncorporacion, s SolicitudConfirmarIncorporacion) {
			*c = contextoConfirmacionIncorporacionV3Prueba(
				t, s.datos(), instanteConfirmacionIncorporacionPrueba,
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
			if _, err := servicio.Confirmar(context.Background(), solicitud); !errors.Is(
				err, ErrConfirmacionIncorporacionDenegada,
			) {
				t.Fatalf("cruce V3 aceptado: %v", err)
			}
			llamadas, efectos, _ := transaccion.estado()
			if llamadas != 0 || efectos != 0 {
				t.Fatal("el cruce V3 alcanzo la transaccion")
			}
		})
	}

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
		if _, err := servicio.Confirmar(context.Background(), solicitud); !errors.Is(
			err, ErrConfirmacionIncorporacionDenegada,
		) {
			t.Fatalf("autorizacion V3 vencida aceptada: %v", err)
		}
		llamadas, efectos, _ := transaccion.estado()
		if llamadas != 0 || efectos != 0 {
			t.Fatal("la autorizacion vencida alcanzo la transaccion")
		}
	})
}

func TestCTLITEO702ADenegacionCancelacionCASYFalloAtomico(t *testing.T) {
	t.Run("autoridad ausente", func(t *testing.T) {
		servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		contextos.err = errors.New("detalle privado de identidad")
		recibo, err := servicio.Confirmar(context.Background(), solicitud)
		if !errors.Is(err, ErrConfirmacionIncorporacionDenegada) ||
			strings.Contains(err.Error(), "identidad") ||
			!reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
			t.Fatalf("denegacion no opaca: recibo=%#v error=%v", recibo, err)
		}
		llamadas, _, _ := transaccion.estado()
		if llamadas != 0 {
			t.Fatal("la denegacion alcanzo la transaccion")
		}
	})

	t.Run("contexto para otro expediente", func(t *testing.T) {
		servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		contextos.contexto = contextoConfirmacionIncorporacionV3Prueba(
			t, solicitud.datos(), instanteConfirmacionIncorporacionPrueba,
			func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
				d.Recurso.Referencia = referenciaIncorporacionPrueba("expediente_temporal_otro")
			},
		)
		if _, err := servicio.Confirmar(context.Background(), solicitud); !errors.Is(
			err,
			ErrConfirmacionIncorporacionDenegada,
		) {
			t.Fatalf("contexto cruzado aceptado: %v", err)
		}
		llamadas, _, _ := transaccion.estado()
		if llamadas != 0 {
			t.Fatal("el contexto cruzado alcanzo la transaccion")
		}
	})

	t.Run("cancelacion previa", func(t *testing.T) {
		servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		if _, err := servicio.Confirmar(ctx, solicitud); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion no preservada: %v", err)
		}
		llamadas, efectos, _ := transaccion.estado()
		if contextos.numeroLlamadas() != 0 || llamadas != 0 || efectos != 0 {
			t.Fatal("la cancelacion previa produjo trabajo")
		}
	})

	t.Run("version en conflicto", func(t *testing.T) {
		servicio, contextos, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		solicitud.VersionSeguimientoEsperada = 1
		contextos.contexto = contextoConfirmacionIncorporacionV3Prueba(
			t, solicitud.datos(), instanteConfirmacionIncorporacionPrueba, nil,
		)
		if _, err := servicio.Confirmar(context.Background(), solicitud); !errors.Is(
			err,
			domain.ErrVersionEnConflicto,
		) {
			t.Fatalf("CAS incorrecto aceptado: %v", err)
		}
		_, efectos, seguimiento := transaccion.estado()
		if efectos != 0 || seguimiento.Version() != 0 {
			t.Fatal("el conflicto CAS muto el seguimiento")
		}
	})

	t.Run("rollback", func(t *testing.T) {
		servicio, _, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		transaccion.fallo = errTransaccionIncorporacionPrueba
		recibo, err := servicio.Confirmar(context.Background(), solicitud)
		if !errors.Is(err, errTransaccionIncorporacionPrueba) ||
			!reflect.DeepEqual(recibo, ports.ReciboConfirmacionIncorporacion{}) {
			t.Fatalf("fallo transaccional produjo exito: recibo=%#v error=%v", recibo, err)
		}
		_, efectos, seguimiento := transaccion.estado()
		if efectos != 0 || seguimiento.Version() != 0 || len(seguimiento.Actuaciones()) != 0 {
			t.Fatal("el fallo transaccional dejo un efecto parcial")
		}
	})

	t.Run("recibo adulterado", func(t *testing.T) {
		servicio, _, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
		transaccion.adulterar = func(r *ports.ReciboConfirmacionIncorporacion) {
			r.ConfirmadaEn = instanteConfirmacionIncorporacionPrueba.Add(91 * time.Second)
		}
		if _, err := servicio.Confirmar(context.Background(), solicitud); !errors.Is(
			err,
			ErrResultadoConfirmacionIncorporacionNoConfiable,
		) {
			t.Fatalf("recibo adulterado publicado: %v", err)
		}
	})
}

func TestCTLITEO702AReplayConcurrenteConservaUnSoloEfecto(t *testing.T) {
	servicio, _, transaccion, solicitud := escenarioConfirmacionIncorporacion(t)
	const trabajadores = 32
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	grupo.Add(trabajadores)
	for i := 0; i < trabajadores; i++ {
		go func() {
			defer grupo.Done()
			recibo, err := servicio.Confirmar(context.Background(), solicitud)
			if err != nil {
				errores <- err
				return
			}
			if recibo.ReciboRef != referenciaIncorporacionPrueba("recibo_incorporacion_0001") {
				errores <- ports.ErrReciboConfirmacionIncorporacionInvalido
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("replay concurrente: %v", err)
	}
	llamadas, efectos, seguimiento := transaccion.estado()
	if llamadas != trabajadores || efectos != 1 || seguimiento.Version() != 1 ||
		len(seguimiento.Actuaciones()) != 1 {
		t.Fatalf(
			"replay concurrente divergente: llamadas=%d efectos=%d version=%d",
			llamadas, efectos, seguimiento.Version(),
		)
	}
}

func escenarioConfirmacionIncorporacion(t *testing.T) (
	*ServicioConfirmacionIncorporacion,
	*resolutorContextoIncorporacionPrueba,
	*transaccionIncorporacionPrueba,
	SolicitudConfirmarIncorporacion,
) {
	t.Helper()
	solicitudPersonal := solicitudPersonalIncorporacionPrueba()
	resultadoPersonal := resultadoPersonalIncorporacionPrueba(t, solicitudPersonal)
	definicion := definicionIncorporacionPrueba(t)
	transaccion := nuevaTransaccionIncorporacionPrueba(
		t,
		definicion,
		resultadoPersonal.RelacionRef,
	)
	solicitud := SolicitudConfirmarIncorporacion{
		SolicitudPersonal:          solicitudPersonal,
		ResultadoPersonal:          resultadoPersonal,
		VersionSeguimientoEsperada: 0,
		FechaIncorporacion:         instanteConfirmacionIncorporacionPrueba.Add(24 * time.Hour),
		FechaFinPrevista:           instanteConfirmacionIncorporacionPrueba.AddDate(0, 1, 0),
		MotivoClave:                "necesidad_servicio",
		Documentos: []domain.DocumentoSeguimiento{
			{TipoClave: "resolucion_incorporacion", Referencia: referenciaIncorporacionPrueba("documento_incorporacion_0001")},
			{TipoClave: "anexo_incorporacion", Referencia: referenciaIncorporacionPrueba("documento_incorporacion_0002")},
		},
	}
	contextos := &resolutorContextoIncorporacionPrueba{
		contexto: contextoConfirmacionIncorporacionV3Prueba(
			t, solicitud.datos(), instanteConfirmacionIncorporacionPrueba, nil,
		),
	}
	servicio, err := NuevoServicioConfirmacionIncorporacion(
		contextos,
		&relojMutable{instante: instanteConfirmacionIncorporacionPrueba},
		transaccion,
	)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio, contextos, transaccion, solicitud
}

func contextoConfirmacionIncorporacionV3Prueba(
	t *testing.T,
	datos ports.DatosConfirmacionIncorporacion,
	ahora time.Time,
	mutar func(*dominiovec.DatosSolicitudAutorizacionLigadaV3),
) ports.ContextoConfirmacionIncorporacion {
	t.Helper()
	autoridad := contextoAutorizacionAltaV3Prueba(t, ahora)
	return contextoConfirmacionIncorporacionV3ConAutoridadPrueba(
		t, datos, ahora, autoridad, mutar,
	)
}

func contextoConfirmacionIncorporacionV3ConAutoridadPrueba(
	t *testing.T,
	datos ports.DatosConfirmacionIncorporacion,
	ahora time.Time,
	autoridad ports.ContextoAutorizacionAltaV3,
	mutar func(*dominiovec.DatosSolicitudAutorizacionLigadaV3),
) ports.ContextoConfirmacionIncorporacion {
	t.Helper()
	vinculo, err := autoridad.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		&generadorReferenciasDoble{correlacion: datos.ResultadoPersonal.CorrelacionRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("c", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	solicitudDatos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: autoridad.Vinculo,
		ReferenciaMotivo:          motivo,
		Accion:                    ports.AccionConfirmarIncorporacion,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: datos.SolicitudPersonal.ExpedienteRef,
			ModuloID:   ports.ModuloContratacion,
			Tipo:       ports.TipoRecursoConfirmacionIncorporacion,
			Ambitos: map[string]string{
				"organizacion_ref": referenciaIncorporacionPrueba("organizacion_publica_0001"),
				"unidad_ref":       referenciaIncorporacionPrueba("unidad_rrhh_0001"),
			},
			Atributos: map[string]string{
				"resultado_personal_ref":       datos.ResultadoPersonal.ResultadoRef,
				"relacion_ref":                 datos.ResultadoPersonal.RelacionRef,
				"ocupacion_ref":                datos.ResultadoPersonal.OcupacionRef,
				"version_expediente_esperada":  strconv.FormatUint(datos.SolicitudPersonal.VersionExpediente, 10),
				"version_seguimiento_esperada": strconv.FormatUint(datos.VersionSeguimientoEsperada, 10),
			},
		},
		Finalidad:   ports.FinalidadConfirmarIncorporacion,
		Correlacion: correlacion,
	}
	if mutar != nil {
		mutar(&solicitudDatos)
	}
	solicitudV3, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(solicitudDatos)
	if err != nil {
		t.Fatal(err)
	}
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		t, solicitudV3, autoridad.Resultado, motivo, ahora,
		"dec_0123456789abcdef0123456789abcdef", true,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoConfirmacionIncorporacion{
		SolicitudContexto: ports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: vinculo.AutenticacionRef,
			SesionRef:        vinculo.SesionRef,
			PerfilRef:        vinculo.PerfilActivoRef,
		},
		ContextoAutorizacion:    autoridad,
		SolicitudAutorizacionV3: solicitudV3,
		DecisionAutorizacionV3:  decision,
		ConfirmacionRegistroV3:  confirmacion,
	}
}

func contextoAutorizacionAltaV3SustancialPrueba(
	t *testing.T,
	ahora time.Time,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	base := contextoAutorizacionAltaV3Prueba(t, ahora)
	datosVinculo, err := base.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: base.Resultado.Contexto.Instantanea.CuentaRef,
		Metodo:    datosVinculo.MetodoObservado,
		Garantia:  dominiovec.AuthAssuranceSubstantial,
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta, base.Resultado.Contexto.Instantanea, base.Resultado.Contexto.ResueltoEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := base.Resultado
	resultado.Contexto = actor
	resultado.RepresentacionCanonica, err = actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	resultado.HuellaSHA256, err = actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := datosVinculo.Autenticacion()
	autenticacion.GarantiaObservada = dominiovec.AuthAssuranceSubstantial
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(), revalidadorVinculoPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		resolutorResultadoVinculoPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		},
		relojVinculoPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
}

func nuevaTransaccionIncorporacionPrueba(
	t *testing.T,
	definicion domain.DefinicionSeguimiento,
	relacionRef string,
) *transaccionIncorporacionPrueba {
	t.Helper()
	seguimiento, err := domain.NuevoSeguimiento(definicion, domain.AltaSeguimiento{
		Referencia:      referenciaIncorporacionPrueba("seguimiento_incorporacion_0001"),
		OrganizacionRef: referenciaIncorporacionPrueba("organizacion_publica_0001"),
		ExpedienteRef:   referenciaIncorporacionPrueba("expediente_temporal_0001"),
		RelacionRef:     relacionRef,
		PeriodoPrevisto: domain.IntervaloSeguimiento{
			Desde: instanteConfirmacionIncorporacionPrueba.Add(24 * time.Hour),
			Hasta: instanteConfirmacionIncorporacionPrueba.AddDate(0, 1, 0),
		},
		CreadoEn: instanteConfirmacionIncorporacionPrueba.Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("crear seguimiento pendiente: %v", err)
	}
	return &transaccionIncorporacionPrueba{
		definicion:        definicion,
		seguimiento:       seguimiento,
		versionExpediente: 7,
		confirmadaEn:      instanteConfirmacionIncorporacionPrueba,
		referencias: ports.ReferenciasDurablesConfirmacionIncorporacion{
			ActuacionRef:   referenciaIncorporacionPrueba("actuacion_incorporacion_0001"),
			ReciboRef:      referenciaIncorporacionPrueba("recibo_incorporacion_0001"),
			CorrelacionRef: referenciaIncorporacionPrueba("correlacion_incorporacion_0001"),
			ActorRef:       referenciaIncorporacionPrueba("actor_rrhh_0001"),
		},
	}
}

func definicionIncorporacionPrueba(t *testing.T) domain.DefinicionSeguimiento {
	t.Helper()
	definicion, err := domain.PublicarDefinicionSeguimiento(
		domain.BorradorDefinicionSeguimiento{
			Referencia:  referenciaIncorporacionPrueba("definicion_seguimiento_incorporacion_0001"),
			Version:     1,
			PublicadoEn: instanteConfirmacionIncorporacionPrueba.Add(-48 * time.Hour),
			Vigencia: domain.VigenciaSeguimiento{
				Desde: instanteConfirmacionIncorporacionPrueba.Add(-24 * time.Hour),
				Hasta: instanteConfirmacionIncorporacionPrueba.AddDate(1, 0, 0),
			},
			EstadoInicial:            "pendiente_incorporacion",
			ProhibeCiclosSilenciosos: true,
			Estados: []domain.EstadoDefinidoSeguimiento{
				{Clave: "pendiente_incorporacion"},
				{Clave: "vigente", Final: true},
			},
			Motivos: []domain.ClaveCatalogo{"necesidad_servicio"},
			Transiciones: []domain.TransicionDefinidaSeguimiento{{
				Clave: "confirmar_incorporacion", Origen: "pendiente_incorporacion",
				Destino: "vigente", Clase: domain.TransicionOrdinaria,
				MotivosPermitidos: []domain.ClaveCatalogo{"necesidad_servicio"},
				MotivoObligatorio: true,
				Documentos: []domain.RequisitoDocumentoSeguimiento{
					{TipoClave: "resolucion_incorporacion", Obligatorio: true},
					{TipoClave: "anexo_incorporacion"},
				},
				RequierePeriodo: true,
				EfectoPeriodo:   domain.EfectoPeriodoAbrir,
			}},
		},
	)
	if err != nil {
		t.Fatalf("publicar definicion de incorporacion: %v", err)
	}
	return definicion
}

func solicitudPersonalIncorporacionPrueba() ports.SolicitudAltaPersonalRPT {
	return ports.SolicitudAltaPersonalRPT{
		Esquema:           ports.EsquemaAltaPersonalRPT,
		ContratoVersion:   ports.VersionContratoAltaPersonalRPT,
		SolicitudRef:      referenciaIncorporacionPrueba("solicitud_personal_0001"),
		ExpedienteRef:     referenciaIncorporacionPrueba("expediente_temporal_0001"),
		VersionExpediente: 7,
		CapacidadRef:      referenciaIncorporacionPrueba("capacidad_personal_0001"),
		CorrelacionRef:    "correlacion_0123456789abcdef0123456789abcdef",
		IdempotenciaRef:   referenciaIncorporacionPrueba("idempotencia_personal_0001"),
		FuenteRPT: ports.ReferenciaVersionadaPersonalRPT{
			Referencia:   referenciaIncorporacionPrueba("rpt_publicada_0001"),
			Version:      3,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		PuestoRef: referenciaIncorporacionPrueba("puesto_rpt_0001"),
		PlazaRef:  referenciaIncorporacionPrueba("plaza_rpt_0001"),
	}
}

func resultadoPersonalIncorporacionPrueba(
	t *testing.T,
	solicitud ports.SolicitudAltaPersonalRPT,
) ports.ResultadoAltaPersonalRPT {
	t.Helper()
	huella, err := solicitud.HuellaSHA256()
	if err != nil {
		t.Fatalf("calcular huella de solicitud Personal: %v", err)
	}
	return ports.ResultadoAltaPersonalRPT{
		Esquema:               ports.EsquemaAltaPersonalRPT,
		ContratoVersion:       ports.VersionContratoAltaPersonalRPT,
		ResultadoRef:          referenciaIncorporacionPrueba("resultado_personal_0001"),
		ReciboRef:             referenciaIncorporacionPrueba("recibo_personal_0001"),
		SolicitudRef:          solicitud.SolicitudRef,
		CorrelacionRef:        solicitud.CorrelacionRef,
		IdempotenciaRef:       solicitud.IdempotenciaRef,
		HuellaSolicitudSHA256: huella,
		Estado:                ports.AltaPersonalRPTConfirmada,
		RelacionRef:           referenciaIncorporacionPrueba("relacion_juridica_0001"),
		OcupacionRef:          referenciaIncorporacionPrueba("ocupacion_personal_0001"),
	}
}

func referenciaIncorporacionPrueba(etiqueta string) string {
	return fmt.Sprintf("ref:%x", sha256.Sum256([]byte(etiqueta)))
}
