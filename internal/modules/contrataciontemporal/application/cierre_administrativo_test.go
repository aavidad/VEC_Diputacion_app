package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

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
	if !errors.Is(err, ports.ErrClaveIdempotenciaCierreAdministrativoUsada) ||
		transaccion.aplicaciones != 1 {
		t.Fatalf("el replay aceptó la misma clave con otra identidad: %v", err)
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

func TestCierreAdministrativoSinCeseConservaRelacionYPrefijo(t *testing.T) {
	original, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamenteSinCesePrueba(seguimiento.Version())
	preparacion := preparacionCierreAdministrativoPrueba(
		t, solicitud, original, seguimiento, instanteCierreAdministrativoPrueba,
	)
	sucesora := definicionSucesoraCierreSinCesePrueba(t, original)
	continuacion, err := domain.NuevaContinuacionSeguimiento(
		original, sucesora, seguimiento, datosCierreAdministrativoPrueba(preparacion, solicitud),
	)
	if err != nil {
		t.Fatalf("crear continuidad: %v", err)
	}
	preparacion.DefinicionSucesora = &sucesora
	preparacion.Continuacion = &continuacion
	if err := preparacion.ValidarPara(solicitud); err != nil {
		t.Fatalf("preparación continuada inválida: %v", err)
	}
	transaccion := &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
	resultado, err := nuevoServicioCierreAdministrativoPrueba(t, transaccion).
		CerrarSinCese(context.Background(), solicitudAplicacionCierreSinCesePrueba(solicitud))
	if err != nil {
		t.Fatalf("cerrar sin cese: %v", err)
	}
	if !transaccion.confirmada || resultado.VersionSeguimiento() != seguimiento.Version()+1 ||
		transaccion.siguiente.EstadoActual() != domain.EstadoCerradoAdministrativamenteSeguimiento ||
		transaccion.siguiente.CeseEfectivo() != nil || seguimiento.CeseEfectivo() != nil ||
		!reflect.DeepEqual(seguimiento.PeriodosResultantes(), transaccion.siguiente.PeriodosResultantes()) {
		t.Fatalf("el cierre sin cese alteró la relación: resultado=%#v", transaccion.siguiente.Estado())
	}
	antes, despues := seguimiento.Actuaciones(), transaccion.siguiente.Actuaciones()
	if len(despues) != len(antes)+1 ||
		despues[len(despues)-1].Definicion != sucesora.Referencia() ||
		despues[len(despues)-1].TransicionClave != domain.TransicionCerrarAdministrativamenteSinCese ||
		antes[0].HuellaActuacionSHA256 != despues[0].HuellaActuacionSHA256 {
		t.Fatalf("el cierre sin cese no preservó el prefijo original")
	}
}

func TestCierreAdministrativoSinCeseDeniegaInventarioPendienteYReapertura(t *testing.T) {
	original, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamenteSinCesePrueba(seguimiento.Version())
	preparacion := preparacionCierreAdministrativoPrueba(
		t, solicitud, original, seguimiento, instanteCierreAdministrativoPrueba,
	)
	sucesora := definicionSucesoraCierreSinCesePrueba(t, original)
	continuacion, err := domain.NuevaContinuacionSeguimiento(
		original, sucesora, seguimiento, datosCierreAdministrativoPrueba(preparacion, solicitud),
	)
	if err != nil {
		t.Fatalf("crear continuidad: %v", err)
	}
	preparacion.DefinicionSucesora = &sucesora
	preparacion.Continuacion = &continuacion
	preparacion.Inventario.Pendientes = 1
	transaccion := &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
	servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)
	if _, err := servicio.CerrarSinCese(context.Background(), solicitudAplicacionCierreSinCesePrueba(solicitud)); !errors.Is(err, ErrCierreAdministrativoNoPermitido) || transaccion.confirmada {
		t.Fatalf("se aceptó continuidad con inventario pendiente: %v", err)
	}

	preparacion.Inventario.Pendientes = 0
	transaccion = &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
	servicio = nuevoServicioCierreAdministrativoPrueba(t, transaccion)
	reapertura := SolicitudReabrirExcepcionalmente{
		OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef,
		SeguimientoRef: solicitud.SeguimientoRef, VersionEsperada: solicitud.VersionEsperada,
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5c",
		TransicionClave:   "reabrir_excepcionalmente", MotivoClave: "subsanacion_excepcional",
	}
	if _, err := servicio.ReabrirExcepcionalmente(context.Background(), reapertura); !errors.Is(err, ErrCierreAdministrativoNoPermitido) || transaccion.confirmada {
		t.Fatalf("la reapertura aceptó una continuidad sin cese: %v", err)
	}
}

func solicitudCerrarAdministrativamenteSinCesePrueba(
	version uint64,
) ports.SolicitudTransaccionCierreAdministrativo {
	base := solicitudCerrarAdministrativamentePrueba(version)
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:       ports.OperacionCerrarAdministrativamenteSinCese,
		OrganizacionRef: base.OrganizacionRef, ExpedienteRef: base.ExpedienteRef,
		SeguimientoRef: base.SeguimientoRef, VersionEsperada: base.VersionEsperada,
		ClaveIdempotencia: base.ClaveIdempotencia,
		TransicionClave:   domain.TransicionCerrarAdministrativamenteSinCese,
		MotivoClave:       base.MotivoClave,
	}
}

func solicitudAplicacionCierreSinCesePrueba(
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
) SolicitudCerrarAdministrativamente {
	return SolicitudCerrarAdministrativamente{
		OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef,
		SeguimientoRef: solicitud.SeguimientoRef, VersionEsperada: solicitud.VersionEsperada,
		ClaveIdempotencia: solicitud.ClaveIdempotencia, TransicionClave: solicitud.TransicionClave,
		MotivoClave: solicitud.MotivoClave,
	}
}

func datosCierreAdministrativoPrueba(
	preparacion ports.PreparacionTransaccionCierreAdministrativo,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
) domain.DatosTransicionSeguimiento {
	return domain.DatosTransicionSeguimiento{
		ActuacionRef: preparacion.ActuacionRef, TransicionClave: solicitud.TransicionClave,
		MotivoClave: solicitud.MotivoClave, ActorRef: preparacion.ActorRef,
		UnidadRef: preparacion.UnidadRef, EfectivoEn: preparacion.EfectivoEn,
		RegistradaEn: preparacion.RegistradaEn, Documentos: preparacion.Documentos,
		ReciboRef: preparacion.ReciboRef, CorrelacionRef: preparacion.CorrelacionRef,
	}
}

func definicionSucesoraCierreSinCesePrueba(
	t *testing.T,
	original domain.DefinicionSeguimiento,
) domain.DefinicionSeguimiento {
	t.Helper()
	publicacion := original.Publicacion()
	publicacion.Version++
	publicacion.PublicadoEn = instanteCierreAdministrativoPrueba.Add(-time.Hour)
	publicacion.Vigencia.Desde = publicacion.PublicadoEn
	publicacion.Estados = append(publicacion.Estados, domain.EstadoDefinidoSeguimiento{
		Clave: domain.EstadoCerradoAdministrativamenteSeguimiento, Final: true,
	})
	publicacion.Transiciones = append(publicacion.Transiciones, domain.TransicionDefinidaSeguimiento{
		Clave: domain.TransicionCerrarAdministrativamenteSinCese, Origen: "vigente",
		Destino:           domain.EstadoCerradoAdministrativamenteSeguimiento,
		Clase:             domain.TransicionOrdinaria,
		MotivosPermitidos: []domain.ClaveCatalogo{"fin_expediente"},
		MotivoObligatorio: true, EfectoPeriodo: domain.EfectoPeriodoNinguno,
	})
	sucesora, err := domain.PublicarDefinicionSeguimiento(domain.BorradorDefinicionSeguimiento{
		Referencia: publicacion.Referencia, Version: publicacion.Version,
		PublicadoEn: publicacion.PublicadoEn, Vigencia: publicacion.Vigencia,
		EstadoInicial:            publicacion.EstadoInicial,
		ProhibeCiclosSilenciosos: publicacion.ProhibeCiclosSilenciosos,
		Estados:                  publicacion.Estados, Motivos: publicacion.Motivos,
		Transiciones: publicacion.Transiciones,
	})
	if err != nil {
		t.Fatalf("publicar sucesora de cierre sin cese: %v", err)
	}
	return sucesora
}
