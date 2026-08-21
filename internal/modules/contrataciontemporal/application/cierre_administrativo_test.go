package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var instanteCierreAdministrativoPrueba = time.Date(
	2026, time.August, 21, 10, 0, 0, 0, time.UTC,
)

type transaccionCierreAdministrativoPrueba struct {
	preparacion         ports.PreparacionTransaccionCierreAdministrativo
	err                 error
	resultadoInvalido   bool
	replayConfirmado    bool
	antesDeAplicar      func()
	llamadas            int
	aplicaciones        int
	solicitud           ports.SolicitudTransaccionCierreAdministrativo
	solicitudConfirmada ports.SolicitudTransaccionCierreAdministrativo
	siguiente           domain.Seguimiento
	confirmada          bool
}

func (t *transaccionCierreAdministrativoPrueba) EjecutarCierreAdministrativo(
	ctx context.Context,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	aplicar ports.AplicarCierreAdministrativo,
) (ports.ResultadoCierreAdministrativo, error) {
	t.llamadas++
	t.solicitud = solicitud
	if t.err != nil {
		return ports.ResultadoCierreAdministrativo{}, t.err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	if t.replayConfirmado {
		return ports.NuevoResultadoCierreAdministrativo(
			ports.DatosResultadoCierreAdministrativo{
				Solicitud:         t.solicitudConfirmada,
				VersionResultante: t.solicitudConfirmada.VersionEsperada + 1,
				ActuacionRef:      t.preparacion.ActuacionRef,
				ReciboRef:         t.preparacion.ReciboRef,
				Estado:            ports.EstadoResultadoCierreAdministrativoReplayConfirmado,
			},
		)
	}
	if t.antesDeAplicar != nil {
		t.antesDeAplicar()
	}
	t.aplicaciones++
	siguiente, err := aplicar(t.preparacion)
	if err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	t.siguiente = siguiente
	t.confirmada = true
	t.solicitudConfirmada = solicitud
	if t.resultadoInvalido {
		return ports.ResultadoCierreAdministrativo{}, nil
	}
	return ports.NuevoResultadoCierreAdministrativo(
		ports.DatosResultadoCierreAdministrativo{
			Solicitud: solicitud, VersionResultante: siguiente.Version(),
			ActuacionRef: t.preparacion.ActuacionRef,
			ReciboRef:    t.preparacion.ReciboRef,
			Estado:       ports.EstadoResultadoCierreAdministrativoConfirmado,
		},
	)
}

func TestCierreAdministrativoConfirmaSoloSinTareasPendientes(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	transaccion := &transaccionCierreAdministrativoPrueba{
		preparacion: preparacionCierreAdministrativoPrueba(
			t,
			solicitudTransaccionalCierrePrueba(solicitud),
			definicion,
			seguimiento,
			instanteCierreAdministrativoPrueba,
		),
	}
	servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)
	anteriores := seguimiento.Actuaciones()

	resultado, err := servicio.Cerrar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("cerrar administrativamente: %v", err)
	}
	if !transaccion.confirmada || transaccion.llamadas != 1 {
		t.Fatalf("la transacción no confirmó exactamente una vez")
	}
	if resultado.VersionSeguimiento() != seguimiento.Version()+1 ||
		resultado.ReciboRef() != transaccion.preparacion.ReciboRef {
		t.Fatalf("resultado opaco incoherente: version=%d recibo=%q",
			resultado.VersionSeguimiento(), resultado.ReciboRef())
	}
	posteriores := transaccion.siguiente.Actuaciones()
	if len(posteriores) != len(anteriores)+1 ||
		posteriores[0].HuellaActuacionSHA256 != anteriores[0].HuellaActuacionSHA256 {
		t.Fatalf("el cierre no conservó la historia de solo adición")
	}
	ultima := posteriores[len(posteriores)-1]
	if ultima.Clase != domain.TransicionOrdinaria ||
		ultima.MotivoClave != solicitud.MotivoClave ||
		ultima.ActorRef != transaccion.preparacion.ActorRef ||
		ultima.ReciboRef != transaccion.preparacion.ReciboRef ||
		transaccion.siguiente.CeseEfectivo() == nil {
		t.Fatalf("la actuación de cierre no quedó versionada y auditable: %#v", ultima)
	}
	serializado, err := json.Marshal(resultado)
	if err != nil || string(serializado) != "{}" {
		t.Fatalf("el resultado expuso detalles internos: %q, %v", serializado, err)
	}
}

func TestCierreAdministrativoDevuelveReplayConfirmadoSinRepetirCallback(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	transaccion := &transaccionCierreAdministrativoPrueba{
		preparacion: preparacionCierreAdministrativoPrueba(
			t, solicitudTransaccionalCierrePrueba(solicitud), definicion,
			seguimiento, instanteCierreAdministrativoPrueba,
		),
	}
	servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)
	primero, err := servicio.Cerrar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primer cierre: %v", err)
	}
	transaccion.replayConfirmado = true
	segundo, err := servicio.Cerrar(context.Background(), solicitud)
	if err != nil || !segundo.EsReplayConfirmado() ||
		segundo.ReciboRef() != primero.ReciboRef() ||
		segundo.VersionSeguimiento() != primero.VersionSeguimiento() ||
		transaccion.aplicaciones != 1 || transaccion.llamadas != 2 {
		t.Fatalf("replay no exacto: err=%v aplicaciones=%d llamadas=%d",
			err, transaccion.aplicaciones, transaccion.llamadas)
	}
	colision := solicitud
	colision.MotivoClave = "subsanacion_excepcional"
	_, err = servicio.Cerrar(context.Background(), colision)
	if !errors.Is(err, ErrResultadoCierreAdministrativoInvalido) ||
		transaccion.aplicaciones != 1 {
		t.Fatalf("el replay aceptó la misma clave con otra identidad: %v", err)
	}
}

func TestCierreAdministrativoCancelaDuranteFronteraAntesDelEfecto(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	ctx, cancelar := context.WithCancel(context.Background())
	transaccion := &transaccionCierreAdministrativoPrueba{
		preparacion: preparacionCierreAdministrativoPrueba(
			t, solicitudTransaccionalCierrePrueba(solicitud), definicion,
			seguimiento, instanteCierreAdministrativoPrueba,
		),
		antesDeAplicar: cancelar,
	}
	_, err := nuevoServicioCierreAdministrativoPrueba(t, transaccion).
		Cerrar(ctx, solicitud)
	if !errors.Is(err, context.Canceled) || transaccion.confirmada ||
		transaccion.aplicaciones != 1 || transaccion.siguiente.Version() != 0 {
		t.Fatalf("cancelación durante frontera no revirtió: err=%v", err)
	}
}

func TestCierreAdministrativoRechazaInventarioYAutorizacionCruzados(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	orden := solicitudTransaccionalCierrePrueba(solicitud)
	for _, caso := range []struct {
		nombre    string
		modificar func(*ports.PreparacionTransaccionCierreAdministrativo)
	}{
		{"inventario vacio de otro expediente", func(p *ports.PreparacionTransaccionCierreAdministrativo) {
			p.Inventario.Total, p.Inventario.Pendientes = 0, 0
			p.Inventario.ExpedienteRef = referenciaCierreAdministrativoPrueba("expediente_ajeno_01")
		}},
		{"autorizacion V3 de otro expediente", func(p *ports.PreparacionTransaccionCierreAdministrativo) {
			ajena := orden
			ajena.ExpedienteRef = referenciaCierreAdministrativoPrueba("expediente_ajeno_01")
			prepararAutorizacionCierreAdministrativoPrueba(
				t, p, ajena, instanteCierreAdministrativoPrueba,
			)
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			preparacion := preparacionCierreAdministrativoPrueba(
				t, orden, definicion, seguimiento, instanteCierreAdministrativoPrueba,
			)
			caso.modificar(&preparacion)
			transaccion := &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
			_, err := nuevoServicioCierreAdministrativoPrueba(t, transaccion).
				Cerrar(context.Background(), solicitud)
			if !errors.Is(err, ErrCierreAdministrativoNoPermitido) || transaccion.confirmada {
				t.Fatalf("cruce aceptado: err=%v", err)
			}
		})
	}
}

func TestCierreAdministrativoExigeEfectoTemporalCompatible(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoConEfectoCierre(
		t, domain.EfectoPeriodoNinguno,
	)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	transaccion := &transaccionCierreAdministrativoPrueba{
		preparacion: preparacionCierreAdministrativoPrueba(
			t, solicitudTransaccionalCierrePrueba(solicitud), definicion,
			seguimiento, instanteCierreAdministrativoPrueba,
		),
	}
	_, err := nuevoServicioCierreAdministrativoPrueba(t, transaccion).
		Cerrar(context.Background(), solicitud)
	if !errors.Is(err, ErrCierreAdministrativoNoPermitido) || transaccion.confirmada {
		t.Fatalf("se aceptó cierre sin EfectoPeriodoCerrar: %v", err)
	}
}

func TestCierreAdministrativoDeniegaInventarioPendienteOIncompleto(t *testing.T) {
	for _, caso := range []struct {
		nombre     string
		inventario ports.InventarioTareasCierreAdministrativo
	}{
		{
			nombre: "tarea pendiente",
			inventario: ports.InventarioTareasCierreAdministrativo{
				Referencia: referenciaCierreAdministrativoPrueba("inventario_tareas_01"), Version: 4,
				Total: 7, Pendientes: 1, Completo: true,
			},
		},
		{
			nombre: "lectura incompleta",
			inventario: ports.InventarioTareasCierreAdministrativo{
				Referencia: referenciaCierreAdministrativoPrueba("inventario_tareas_01"), Version: 4,
				Total: 7, Completo: false,
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			definicion, seguimiento := escenarioSeguimientoVigente(t)
			solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
			preparacion := preparacionCierreAdministrativoPrueba(
				t,
				solicitudTransaccionalCierrePrueba(solicitud),
				definicion,
				seguimiento,
				instanteCierreAdministrativoPrueba,
			)
			caso.inventario.OrganizacionRef = solicitud.OrganizacionRef
			caso.inventario.ExpedienteRef = solicitud.ExpedienteRef
			caso.inventario.SeguimientoRef = solicitud.SeguimientoRef
			preparacion.Inventario = caso.inventario
			transaccion := &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
			servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)

			_, err := servicio.Cerrar(context.Background(), solicitud)
			if !errors.Is(err, ErrCierreAdministrativoNoPermitido) {
				t.Fatalf("error = %v; se esperaba denegación opaca", err)
			}
			if transaccion.confirmada {
				t.Fatalf("se confirmó un cierre sin inventario completo y libre")
			}
		})
	}
}

func TestReaperturaExcepcionalEsMotivadaVersionadaYAuditable(t *testing.T) {
	definicion, vigente := escenarioSeguimientoVigente(t)
	cerrado, err := vigente.Aplicar(
		definicion,
		vigente.Version(),
		domain.DatosTransicionSeguimiento{
			ActuacionRef:    referenciaCierreAdministrativoPrueba("actuacion_cierre_previo_01"),
			TransicionClave: "cerrar_administrativamente",
			MotivoClave:     "fin_expediente",
			ActorRef:        referenciaCierreAdministrativoPrueba("actor_cierre_previo_01"),
			UnidadRef:       referenciaCierreAdministrativoPrueba("unidad_rrhh_01"),
			EfectivoEn:      instanteCierreAdministrativoPrueba,
			RegistradaEn:    instanteCierreAdministrativoPrueba,
			ReciboRef:       referenciaCierreAdministrativoPrueba("recibo_cierre_previo_01"),
			CorrelacionRef:  referenciaCierreAdministrativoPrueba("correlacion_cierre_previo_01"),
		},
	)
	if err != nil {
		t.Fatalf("preparar seguimiento cerrado: %v", err)
	}
	solicitud := SolicitudReabrirExcepcionalmente{
		OrganizacionRef:   referenciaCierreAdministrativoPrueba("organizacion_publica_01"),
		ExpedienteRef:     referenciaCierreAdministrativoPrueba("expediente_temporal_01"),
		SeguimientoRef:    referenciaCierreAdministrativoPrueba("seguimiento_laboral_01"),
		VersionEsperada:   cerrado.Version(),
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5c",
		TransicionClave:   "reabrir_excepcionalmente",
		MotivoClave:       "subsanacion_excepcional",
	}
	orden := ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         ports.OperacionReabrirExcepcionalmente,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		SeguimientoRef:    solicitud.SeguimientoRef,
		VersionEsperada:   solicitud.VersionEsperada,
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		TransicionClave:   solicitud.TransicionClave,
		MotivoClave:       solicitud.MotivoClave,
	}
	preparacion := preparacionCierreAdministrativoPrueba(
		t,
		orden,
		definicion,
		cerrado,
		instanteCierreAdministrativoPrueba.Add(time.Hour),
	)
	preparacion.ActuacionRef = referenciaCierreAdministrativoPrueba("actuacion_reapertura_01")
	preparacion.ReciboRef = referenciaCierreAdministrativoPrueba("recibo_reapertura_01")
	preparacion.CorrelacionRef = referenciaCierreAdministrativoPrueba("correlacion_reapertura_01")
	transaccion := &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
	servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)
	anteriores := cerrado.Actuaciones()

	resultado, err := servicio.ReabrirExcepcionalmente(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatalf("reabrir excepcionalmente: %v", err)
	}
	posteriores := transaccion.siguiente.Actuaciones()
	ultima := posteriores[len(posteriores)-1]
	if resultado.VersionSeguimiento() != cerrado.Version()+1 ||
		len(posteriores) != len(anteriores)+1 ||
		posteriores[len(anteriores)-1].HuellaActuacionSHA256 !=
			anteriores[len(anteriores)-1].HuellaActuacionSHA256 ||
		ultima.Clase != domain.TransicionReapertura ||
		ultima.MotivoClave != "subsanacion_excepcional" ||
		ultima.ActorRef != preparacion.ActorRef ||
		transaccion.siguiente.CeseEfectivo() != nil {
		t.Fatalf("reapertura excepcional no auditable: %#v", ultima)
	}
}

func TestCierreAdministrativoDeniegaAutoridadOMotivoNoGobernado(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	base := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	for _, caso := range []struct {
		nombre    string
		modificar func(*SolicitudCerrarAdministrativamente, *ports.PreparacionTransaccionCierreAdministrativo)
	}{
		{
			nombre: "sin autoridad",
			modificar: func(_ *SolicitudCerrarAdministrativamente, p *ports.PreparacionTransaccionCierreAdministrativo) {
				p.DecisionAutorizacionV3 = dominiovec.DecisionAutorizacionLigadaV3{}
			},
		},
		{
			nombre: "motivo ajeno al catalogo",
			modificar: func(s *SolicitudCerrarAdministrativamente, _ *ports.PreparacionTransaccionCierreAdministrativo) {
				s.MotivoClave = "subsanacion_excepcional"
			},
		},
		{
			nombre: "transicion no terminal",
			modificar: func(s *SolicitudCerrarAdministrativamente, _ *ports.PreparacionTransaccionCierreAdministrativo) {
				s.TransicionClave = "registrar_incidencia"
				s.MotivoClave = "incidencia_catalogada"
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := base
			preparacion := preparacionCierreAdministrativoPrueba(
				t,
				solicitudTransaccionalCierrePrueba(solicitud),
				definicion,
				seguimiento,
				instanteCierreAdministrativoPrueba,
			)
			caso.modificar(&solicitud, &preparacion)
			preparacion.Solicitud = solicitudTransaccionalCierrePrueba(solicitud)
			transaccion := &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
			servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)

			_, err := servicio.Cerrar(context.Background(), solicitud)
			if !errors.Is(err, ErrCierreAdministrativoNoPermitido) {
				t.Fatalf("error = %v; se esperaba denegación", err)
			}
			if transaccion.confirmada {
				t.Fatalf("se confirmó una decisión no autorizada o no gobernada")
			}
		})
	}
}

func TestCierreAdministrativoAplicaCASYValidaResultado(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	t.Run("version esperada", func(t *testing.T) {
		solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version() - 1)
		transaccion := &transaccionCierreAdministrativoPrueba{
			preparacion: preparacionCierreAdministrativoPrueba(
				t,
				solicitudTransaccionalCierrePrueba(solicitud),
				definicion,
				seguimiento,
				instanteCierreAdministrativoPrueba,
			),
		}
		servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)

		_, err := servicio.Cerrar(context.Background(), solicitud)
		if !errors.Is(err, ErrVersionCierreAdministrativoEnConflicto) {
			t.Fatalf("error = %v; se esperaba conflicto de versión", err)
		}
		if transaccion.confirmada {
			t.Fatalf("se confirmó con una versión obsoleta")
		}
	})

	t.Run("resultado adulterado", func(t *testing.T) {
		solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
		transaccion := &transaccionCierreAdministrativoPrueba{
			preparacion: preparacionCierreAdministrativoPrueba(
				t,
				solicitudTransaccionalCierrePrueba(solicitud),
				definicion,
				seguimiento,
				instanteCierreAdministrativoPrueba,
			),
			resultadoInvalido: true,
		}
		servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)

		_, err := servicio.Cerrar(context.Background(), solicitud)
		if !errors.Is(err, ErrResultadoCierreAdministrativoInvalido) {
			t.Fatalf("error = %v; se esperaba resultado inválido", err)
		}
	})
}

func TestCierreAdministrativoFallaCerradoEnFronteras(t *testing.T) {
	var nulaTipada *transaccionCierreAdministrativoPrueba
	if _, err := NuevoServicioCierreAdministrativo(nulaTipada); !errors.Is(
		err,
		ErrServicioCierreAdministrativoInvalido,
	) {
		t.Fatalf("constructor con nulo tipado: %v", err)
	}

	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	for _, caso := range []struct {
		nombre    string
		errPuerto error
		esperado  error
	}{
		{"denegacion", ports.ErrCierreAdministrativoDenegado, ErrCierreAdministrativoNoPermitido},
		{"indisponibilidad", errors.New("detalle privado"), ErrCierreAdministrativoNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			transaccion := &transaccionCierreAdministrativoPrueba{
				preparacion: preparacionCierreAdministrativoPrueba(
					t,
					solicitudTransaccionalCierrePrueba(solicitud),
					definicion,
					seguimiento,
					instanteCierreAdministrativoPrueba,
				),
				err: caso.errPuerto,
			}
			servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)
			_, err := servicio.Cerrar(context.Background(), solicitud)
			if !errors.Is(err, caso.esperado) || errors.Is(err, caso.errPuerto) {
				t.Fatalf("error público no opaco: %v", err)
			}
		})
	}

	transaccion := &transaccionCierreAdministrativoPrueba{}
	servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err := servicio.Cerrar(ctx, solicitud)
	if !errors.Is(err, context.Canceled) || transaccion.llamadas != 0 {
		t.Fatalf("cancelación no prioritaria: error=%v llamadas=%d", err, transaccion.llamadas)
	}
}

func TestSolicitudesCierreAdministrativoNoAceptanAutoridad(t *testing.T) {
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(SolicitudCerrarAdministrativamente{}),
		reflect.TypeOf(SolicitudReabrirExcepcionalmente{}),
	} {
		for _, campo := range []string{
			"ActorRef", "PerfilRef", "UnidadRef", "AutorizacionRef", "Autorizada",
		} {
			if _, existe := tipo.FieldByName(campo); existe {
				t.Fatalf("%s acepta autoridad desde el DTO mediante %s", tipo.Name(), campo)
			}
		}
	}
}

func nuevoServicioCierreAdministrativoPrueba(
	t *testing.T,
	transaccion ports.TransaccionCierreAdministrativo,
) *ServicioCierreAdministrativo {
	t.Helper()
	servicio, err := NuevoServicioCierreAdministrativo(transaccion)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func solicitudCerrarAdministrativamentePrueba(
	version uint64,
) SolicitudCerrarAdministrativamente {
	return SolicitudCerrarAdministrativamente{
		OrganizacionRef:   referenciaCierreAdministrativoPrueba("organizacion_publica_01"),
		ExpedienteRef:     referenciaCierreAdministrativoPrueba("expediente_temporal_01"),
		SeguimientoRef:    referenciaCierreAdministrativoPrueba("seguimiento_laboral_01"),
		VersionEsperada:   version,
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		TransicionClave:   "cerrar_administrativamente",
		MotivoClave:       "fin_expediente",
	}
}

func solicitudTransaccionalCierrePrueba(
	solicitud SolicitudCerrarAdministrativamente,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         ports.OperacionCerrarAdministrativamente,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		SeguimientoRef:    solicitud.SeguimientoRef,
		VersionEsperada:   solicitud.VersionEsperada,
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		TransicionClave:   solicitud.TransicionClave,
		MotivoClave:       solicitud.MotivoClave,
	}
}

func preparacionCierreAdministrativoPrueba(
	t *testing.T,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	definicion domain.DefinicionSeguimiento,
	seguimiento domain.Seguimiento,
	instante time.Time,
) ports.PreparacionTransaccionCierreAdministrativo {
	t.Helper()
	preparacion := ports.PreparacionTransaccionCierreAdministrativo{
		Solicitud: solicitud, Definicion: definicion, Seguimiento: seguimiento,
		Inventario: ports.InventarioTareasCierreAdministrativo{
			Referencia:      referenciaCierreAdministrativoPrueba("inventario_tareas_01"),
			OrganizacionRef: solicitud.OrganizacionRef,
			ExpedienteRef:   solicitud.ExpedienteRef,
			SeguimientoRef:  solicitud.SeguimientoRef, Version: 4,
			Total: 6, Pendientes: 0, Completo: true,
		},
		UnidadRef:      referenciaCierreAdministrativoPrueba("unidad_rrhh_01"),
		ActuacionRef:   referenciaCierreAdministrativoPrueba("actuacion_cierre_01"),
		ReciboRef:      referenciaCierreAdministrativoPrueba("recibo_cierre_01"),
		CorrelacionRef: referenciaCierreAdministrativoPrueba("correlacion_cierre_01"),
		EfectivoEn:     instante, RegistradaEn: instante,
	}
	prepararAutorizacionCierreAdministrativoPrueba(t, &preparacion, solicitud, instante)
	return preparacion
}

type generadorCorrelacionCierreAdministrativoPrueba struct {
	valor string
}

func (g generadorCorrelacionCierreAdministrativoPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func prepararAutorizacionCierreAdministrativoPrueba(
	t *testing.T,
	preparacion *ports.PreparacionTransaccionCierreAdministrativo,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	instante time.Time,
) {
	t.Helper()
	contexto := contextoAutorizacionAltaV3Prueba(t, instante)
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatalf("extraer vínculo V3 de cierre: %v", err)
	}
	resumenMotivo := sha256.Sum256([]byte("motivo_autorizacion_cierre"))
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion_cierre", CatalogoVersion: 1,
		CatalogoHuellaSHA256: hex.EncodeToString(resumenMotivo[:]),
		EntradaClave:         "motivo_" + hex.EncodeToString(resumenMotivo[:16]),
	}
	resumenCorrelacion := sha256.Sum256([]byte(
		"correlacion_autorizacion_" + string(solicitud.Operacion) + solicitud.ClaveIdempotencia,
	))
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionCierreAdministrativoPrueba{
			valor: "correlacion_" + hex.EncodeToString(resumenCorrelacion[:16]),
		},
	)
	if err != nil {
		t.Fatalf("generar correlación V3 de cierre: %v", err)
	}
	accion := ports.AccionAutorizacionCerrarAdministrativamente
	finalidad := ports.FinalidadAutorizacionCerrarAdministrativamente
	if solicitud.Operacion == ports.OperacionReabrirExcepcionalmente {
		accion = ports.AccionAutorizacionReabrirExcepcionalmente
		finalidad = ports.FinalidadAutorizacionReabrirExcepcionalmente
	}
	solicitudV3, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          motivo,
			Accion:                    accion,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: solicitud.SeguimientoRef,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoCierreAdministrativo,
				Ambitos: map[string]string{
					"organizacion_ref": solicitud.OrganizacionRef,
					"expediente_ref":   solicitud.ExpedienteRef,
					"seguimiento_ref":  solicitud.SeguimientoRef,
				},
				Atributos: map[string]string{
					"operacion":        string(solicitud.Operacion),
					"version_esperada": strconv.FormatUint(solicitud.VersionEsperada, 10),
					"transicion_clave": string(solicitud.TransicionClave),
					"motivo_clave":     string(solicitud.MotivoClave),
				},
			},
			Finalidad:   finalidad,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud V3 de cierre: %v", err)
	}
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		t,
		solicitudV3,
		contexto.Resultado,
		motivo,
		instante,
		"decision:cierre-administrativo:prueba",
		true,
	)
	if err != nil {
		t.Fatalf("conceder autorización V3 de cierre: %v", err)
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		t.Fatalf("extraer correlación V3 de cierre: %v", err)
	}
	preparacion.ContextoAutorizacionV3 = contexto
	preparacion.SolicitudAutorizacionV3 = solicitudV3
	preparacion.DecisionAutorizacionV3 = decision
	preparacion.ConfirmacionAutorizacionV3 = confirmacion
	preparacion.MotivoAutorizacionV3 = motivo
	preparacion.CorrelacionAutorizacionV3Ref = correlacionRef
	preparacion.ActorRef = referenciaCierreAdministrativoPrueba("actor_rrhh_confiable_01")
	preparacion.PerfilRef = vinculo.PerfilActivoRef
}

func escenarioSeguimientoVigente(
	t *testing.T,
) (domain.DefinicionSeguimiento, domain.Seguimiento) {
	return escenarioSeguimientoConEfectoCierre(t, domain.EfectoPeriodoCerrar)
}

func escenarioSeguimientoConEfectoCierre(
	t *testing.T,
	efectoCierre domain.EfectoPeriodoSeguimiento,
) (domain.DefinicionSeguimiento, domain.Seguimiento) {
	t.Helper()
	definicion, err := domain.PublicarDefinicionSeguimiento(
		domain.BorradorDefinicionSeguimiento{
			Referencia:  referenciaCierreAdministrativoPrueba("definicion_cierre_administrativo_01"),
			Version:     1,
			PublicadoEn: instanteCierreAdministrativoPrueba.Add(-96 * time.Hour),
			Vigencia: domain.VigenciaSeguimiento{
				Desde: instanteCierreAdministrativoPrueba.Add(-72 * time.Hour),
				Hasta: instanteCierreAdministrativoPrueba.AddDate(1, 0, 0),
			},
			EstadoInicial: "pendiente_incorporacion", ProhibeCiclosSilenciosos: true,
			Estados: []domain.EstadoDefinidoSeguimiento{
				{Clave: "pendiente_incorporacion"},
				{Clave: "vigente"},
				{Clave: "cesada", Final: true},
			},
			Motivos: []domain.ClaveCatalogo{
				"incorporacion_confirmada", "fin_expediente",
				"subsanacion_excepcional", "incidencia_catalogada",
			},
			Transiciones: []domain.TransicionDefinidaSeguimiento{
				{
					Clave: "confirmar_incorporacion", Origen: "pendiente_incorporacion",
					Destino: "vigente", Clase: domain.TransicionOrdinaria,
					MotivosPermitidos: []domain.ClaveCatalogo{"incorporacion_confirmada"},
					MotivoObligatorio: true, RequierePeriodo: true,
					EfectoPeriodo: domain.EfectoPeriodoAbrir,
				},
				{
					Clave: "registrar_incidencia", Origen: "vigente", Destino: "vigente",
					Clase:             domain.TransicionOrdinaria,
					MotivosPermitidos: []domain.ClaveCatalogo{"incidencia_catalogada"},
					MotivoObligatorio: true, EfectoPeriodo: domain.EfectoPeriodoNinguno,
				},
				{
					Clave: "cerrar_administrativamente", Origen: "vigente", Destino: "cesada",
					Clase:             domain.TransicionOrdinaria,
					MotivosPermitidos: []domain.ClaveCatalogo{"fin_expediente"},
					MotivoObligatorio: true, EfectoPeriodo: efectoCierre,
				},
				{
					Clave: "reabrir_excepcionalmente", Origen: "cesada", Destino: "vigente",
					Clase:             domain.TransicionReapertura,
					MotivosPermitidos: []domain.ClaveCatalogo{"subsanacion_excepcional"},
					MotivoObligatorio: true, EfectoPeriodo: domain.EfectoPeriodoReabrir,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("publicar definición de cierre: %v", err)
	}
	periodo := domain.IntervaloSeguimiento{
		Desde: instanteCierreAdministrativoPrueba.Add(-24 * time.Hour),
		Hasta: instanteCierreAdministrativoPrueba.Add(30 * 24 * time.Hour),
	}
	seguimiento, err := domain.NuevoSeguimiento(
		definicion,
		domain.AltaSeguimiento{
			Referencia:      referenciaCierreAdministrativoPrueba("seguimiento_laboral_01"),
			OrganizacionRef: referenciaCierreAdministrativoPrueba("organizacion_publica_01"),
			ExpedienteRef:   referenciaCierreAdministrativoPrueba("expediente_temporal_01"),
			RelacionRef:     referenciaCierreAdministrativoPrueba("relacion_laboral_opaca_01"),
			PeriodoPrevisto: periodo,
			CreadoEn:        instanteCierreAdministrativoPrueba.Add(-48 * time.Hour),
		},
	)
	if err != nil {
		t.Fatalf("crear seguimiento: %v", err)
	}
	seguimiento, err = seguimiento.Aplicar(
		definicion,
		seguimiento.Version(),
		domain.DatosTransicionSeguimiento{
			ActuacionRef:    referenciaCierreAdministrativoPrueba("actuacion_incorporacion_01"),
			TransicionClave: "confirmar_incorporacion",
			MotivoClave:     "incorporacion_confirmada",
			ActorRef:        referenciaCierreAdministrativoPrueba("actor_personal_01"),
			UnidadRef:       referenciaCierreAdministrativoPrueba("unidad_personal_01"),
			EfectivoEn:      periodo.Desde, RegistradaEn: periodo.Desde,
			Periodo:        &periodo,
			ReciboRef:      referenciaCierreAdministrativoPrueba("recibo_incorporacion_01"),
			CorrelacionRef: referenciaCierreAdministrativoPrueba("correlacion_incorporacion_01"),
		},
	)
	if err != nil {
		t.Fatalf("confirmar incorporación: %v", err)
	}
	return definicion, seguimiento
}

func referenciaCierreAdministrativoPrueba(etiqueta string) string {
	resumen := sha256.Sum256([]byte(etiqueta))
	return "ref:" + hex.EncodeToString(resumen[:])
}
