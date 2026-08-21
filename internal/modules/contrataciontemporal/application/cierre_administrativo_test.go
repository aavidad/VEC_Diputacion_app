package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var instanteCierreAdministrativoPrueba = time.Date(
	2026, time.August, 21, 10, 0, 0, 0, time.UTC,
)

type transaccionCierreAdministrativoPrueba struct {
	preparacion       ports.PreparacionTransaccionCierreAdministrativo
	err               error
	resultadoInvalido bool
	llamadas          int
	solicitud         ports.SolicitudTransaccionCierreAdministrativo
	siguiente         domain.Seguimiento
	confirmada        bool
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
	siguiente, err := aplicar(t.preparacion)
	if err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	t.siguiente = siguiente
	t.confirmada = true
	if t.resultadoInvalido {
		return ports.ResultadoCierreAdministrativo{}, nil
	}
	return ports.NuevoResultadoCierreAdministrativo(
		ports.DatosResultadoCierreAdministrativo{
			Solicitud: solicitud, VersionResultante: siguiente.Version(),
			ActuacionRef: t.preparacion.ActuacionRef,
			ReciboRef:    t.preparacion.ReciboRef,
		},
	)
}

func TestCierreAdministrativoConfirmaSoloSinTareasPendientes(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	transaccion := &transaccionCierreAdministrativoPrueba{
		preparacion: preparacionCierreAdministrativoPrueba(
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
				solicitudTransaccionalCierrePrueba(solicitud),
				definicion,
				seguimiento,
				instanteCierreAdministrativoPrueba,
			)
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
		OrganizacionRef: referenciaCierreAdministrativoPrueba("organizacion_publica_01"),
		ExpedienteRef:   referenciaCierreAdministrativoPrueba("expediente_temporal_01"),
		SeguimientoRef:  referenciaCierreAdministrativoPrueba("seguimiento_laboral_01"),
		VersionEsperada: cerrado.Version(),
		TransicionClave: "reabrir_excepcionalmente",
		MotivoClave:     "subsanacion_excepcional",
	}
	orden := ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:       ports.OperacionReabrirExcepcionalmente,
		OrganizacionRef: solicitud.OrganizacionRef,
		ExpedienteRef:   solicitud.ExpedienteRef,
		SeguimientoRef:  solicitud.SeguimientoRef,
		VersionEsperada: solicitud.VersionEsperada,
		TransicionClave: solicitud.TransicionClave,
		MotivoClave:     solicitud.MotivoClave,
	}
	preparacion := preparacionCierreAdministrativoPrueba(
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
				p.AutorizacionConcedida = false
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
		OrganizacionRef: referenciaCierreAdministrativoPrueba("organizacion_publica_01"),
		ExpedienteRef:   referenciaCierreAdministrativoPrueba("expediente_temporal_01"),
		SeguimientoRef:  referenciaCierreAdministrativoPrueba("seguimiento_laboral_01"),
		VersionEsperada: version,
		TransicionClave: "cerrar_administrativamente",
		MotivoClave:     "fin_expediente",
	}
}

func solicitudTransaccionalCierrePrueba(
	solicitud SolicitudCerrarAdministrativamente,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:       ports.OperacionCerrarAdministrativamente,
		OrganizacionRef: solicitud.OrganizacionRef,
		ExpedienteRef:   solicitud.ExpedienteRef,
		SeguimientoRef:  solicitud.SeguimientoRef,
		VersionEsperada: solicitud.VersionEsperada,
		TransicionClave: solicitud.TransicionClave,
		MotivoClave:     solicitud.MotivoClave,
	}
}

func preparacionCierreAdministrativoPrueba(
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	definicion domain.DefinicionSeguimiento,
	seguimiento domain.Seguimiento,
	instante time.Time,
) ports.PreparacionTransaccionCierreAdministrativo {
	return ports.PreparacionTransaccionCierreAdministrativo{
		Solicitud: solicitud, Definicion: definicion, Seguimiento: seguimiento,
		Inventario: ports.InventarioTareasCierreAdministrativo{
			Referencia: referenciaCierreAdministrativoPrueba("inventario_tareas_01"), Version: 4,
			Total: 6, Pendientes: 0, Completo: true,
		},
		AutorizacionRef:       referenciaCierreAdministrativoPrueba("autorizacion_cierre_01"),
		AutorizacionConcedida: true,
		ActorRef:              referenciaCierreAdministrativoPrueba("actor_rrhh_confiable_01"),
		UnidadRef:             referenciaCierreAdministrativoPrueba("unidad_rrhh_01"),
		ActuacionRef:          referenciaCierreAdministrativoPrueba("actuacion_cierre_01"),
		ReciboRef:             referenciaCierreAdministrativoPrueba("recibo_cierre_01"),
		CorrelacionRef:        referenciaCierreAdministrativoPrueba("correlacion_cierre_01"),
		EfectivoEn:            instante, RegistradaEn: instante,
	}
}

func escenarioSeguimientoVigente(
	t *testing.T,
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
					MotivoObligatorio: true, EfectoPeriodo: domain.EfectoPeriodoCerrar,
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
