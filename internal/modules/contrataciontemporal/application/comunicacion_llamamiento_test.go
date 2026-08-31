package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type transaccionComunicacionLlamamientoPrueba struct {
	comunicacion     ports.ComunicacionProbatoria
	errorRegistro    error
	resolucion       ports.ResultadoResolucionLlamamiento
	errorResolucion  error
	antesDeRegistrar func()
	antesDeResolver  func()
	registros        int
	resoluciones     int
}

func (t *transaccionComunicacionLlamamientoPrueba) RegistrarComunicacion(
	_ context.Context,
	_ ports.SolicitudRegistrarComunicacionLlamamiento,
) (ports.ComunicacionProbatoria, error) {
	t.registros++
	if t.antesDeRegistrar != nil {
		t.antesDeRegistrar()
	}
	return t.comunicacion, t.errorRegistro
}

func (t *transaccionComunicacionLlamamientoPrueba) ResolverLlamamiento(
	_ context.Context,
	_ ports.SolicitudResolverLlamamiento,
) (ports.ResultadoResolucionLlamamiento, error) {
	t.resoluciones++
	if t.antesDeResolver != nil {
		t.antesDeResolver()
	}
	return t.resolucion, t.errorResolucion
}

func TestNuevoServicioComunicacionLlamamientoFallaCerrado(t *testing.T) {
	if _, err := NuevoServicioComunicacionLlamamiento(nil); !errors.Is(
		err,
		ErrServicioComunicacionLlamamientoInvalido,
	) {
		t.Fatalf("dependencia nula aceptada: %v", err)
	}
	var transaccionNula *transaccionComunicacionLlamamientoPrueba
	if _, err := NuevoServicioComunicacionLlamamiento(transaccionNula); !errors.Is(
		err,
		ErrServicioComunicacionLlamamientoInvalido,
	) {
		t.Fatalf("dependencia nula tipada aceptada: %v", err)
	}
}

func TestServicioComunicacionLlamamientoSoloDependeDeTransaccionLocal(t *testing.T) {
	tipo := reflect.TypeOf(ServicioComunicacionLlamamiento{})
	if tipo.NumField() != 1 || tipo.Field(0).Name != "transaccion" ||
		tipo.Field(0).Type != reflect.TypeOf((*ports.TransaccionComunicacionLlamamiento)(nil)).Elem() {
		t.Fatalf("el servicio incorpora una frontera ajena a la transaccion local: %v", tipo)
	}
}

func TestServicioComunicacionLlamamientoRegistraResultadoLocal(t *testing.T) {
	solicitud := solicitudRegistroComunicacionAplicacionPrueba()
	transaccion := &transaccionComunicacionLlamamientoPrueba{
		comunicacion: comunicacionAplicacionPrueba(solicitud),
	}
	servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)

	resultado, err := servicio.Registrar(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("registro rechazado: %v", err)
	}
	if resultado.ValidarPara(solicitud) != nil || transaccion.registros != 1 ||
		transaccion.resoluciones != 0 {
		t.Fatalf("registro local incoherente: resultado=%+v llamadas=%d/%d", resultado, transaccion.registros, transaccion.resoluciones)
	}
}

func TestServicioComunicacionLlamamientoResuelveSoloConOutboxLocal(t *testing.T) {
	casos := []struct {
		nombre    string
		respuesta ports.RespuestaLlamamiento
		plazo     ports.EstadoPlazoLlamamiento
		outbox    *ports.EstadoOutboxSiguienteCandidato
	}{
		{"aceptacion_sin_outbox", ports.RespuestaLlamamientoAceptada, ports.PlazoLlamamientoVigente, nil},
		{"renuncia_pendiente", ports.RespuestaLlamamientoRenunciada, ports.PlazoLlamamientoVigente, estadoOutboxAplicacion(ports.OutboxSiguienteCandidatoPendiente)},
		{"expiracion_sin_avance", ports.RespuestaLlamamientoExpirada, ports.PlazoLlamamientoExpirado, nil},
		{"expiracion_con_avance", ports.RespuestaLlamamientoExpirada, ports.PlazoLlamamientoExpirado, estadoOutboxAplicacion(ports.OutboxSiguienteCandidatoPendiente)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := solicitudResolverComunicacionAplicacionPrueba(caso.respuesta)
			transaccion := &transaccionComunicacionLlamamientoPrueba{
				resolucion: resolucionAplicacionPrueba(
					solicitud,
					ports.ResultadoComunicacionLlamamientoConfirmado,
					caso.plazo,
					caso.outbox,
				),
			}
			servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
			resultado, err := servicio.Resolver(context.Background(), solicitud)
			if err != nil {
				t.Fatalf("resolucion rechazada: %v", err)
			}
			if resultado.ValidarPara(solicitud) != nil || transaccion.resoluciones != 1 ||
				transaccion.registros != 0 {
				t.Fatalf("resolucion local incoherente: resultado=%+v llamadas=%d/%d", resultado, transaccion.registros, transaccion.resoluciones)
			}
		})
	}
}

func TestServicioComunicacionLlamamientoAceptaReplayOutboxPosterior(t *testing.T) {
	solicitud := solicitudResolverComunicacionAplicacionPrueba(
		ports.RespuestaLlamamientoRenunciada,
	)
	for _, estado := range []ports.EstadoOutboxSiguienteCandidato{
		ports.OutboxSiguienteCandidatoDespachada,
		ports.OutboxSiguienteCandidatoIndeterminada,
	} {
		t.Run(string(estado), func(t *testing.T) {
			transaccion := &transaccionComunicacionLlamamientoPrueba{
				resolucion: resolucionAplicacionPrueba(
					solicitud,
					ports.ResultadoComunicacionLlamamientoReplay,
					ports.PlazoLlamamientoVigente,
					estadoOutboxAplicacion(estado),
				),
			}
			servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
			resultado, err := servicio.Resolver(context.Background(), solicitud)
			if err != nil || !resultado.EsReplayConfirmado() {
				t.Fatalf("replay %s rechazado: resultado=%+v err=%v", estado, resultado, err)
			}
		})
	}
}

func TestServicioComunicacionLlamamientoNoInvocaPuertoConSolicitudInvalida(t *testing.T) {
	transaccion := &transaccionComunicacionLlamamientoPrueba{}
	servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)

	registro := solicitudRegistroComunicacionAplicacionPrueba()
	registro.VersionEsperada = 0
	if _, err := servicio.Registrar(context.Background(), registro); !errors.Is(
		err,
		ErrSolicitudComunicacionLlamamientoInvalida,
	) {
		t.Fatalf("registro invalido no rechazado: %v", err)
	}
	resolucion := solicitudResolverComunicacionAplicacionPrueba(
		ports.RespuestaLlamamientoAceptada,
	)
	resolucion.PruebaRespuestaRef = ""
	if _, err := servicio.Resolver(context.Background(), resolucion); !errors.Is(
		err,
		ErrSolicitudComunicacionLlamamientoInvalida,
	) {
		t.Fatalf("resolucion invalida no rechazada: %v", err)
	}
	if transaccion.registros != 0 || transaccion.resoluciones != 0 {
		t.Fatalf("se invoco el puerto local: %d/%d", transaccion.registros, transaccion.resoluciones)
	}
}

func TestServicioComunicacionLlamamientoClasificaFallosLocales(t *testing.T) {
	casos := []struct {
		nombre   string
		puerto   error
		esperado error
	}{
		{"denegacion", ports.ErrOperacionComunicacionLlamamientoDenegada, ErrComunicacionLlamamientoDenegada},
		{"version", ports.ErrVersionComunicacionLlamamientoEnConflicto, ErrVersionComunicacionLlamamientoEnConflicto},
		{"idempotencia", ports.ErrClaveComunicacionLlamamientoUsada, ErrClaveComunicacionLlamamientoEnColision},
		{"indisponibilidad", errors.New("detalle privado"), ErrComunicacionLlamamientoNoDisponible},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			transaccion := &transaccionComunicacionLlamamientoPrueba{errorRegistro: caso.puerto}
			servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
			resultado, err := servicio.Registrar(
				context.Background(),
				solicitudRegistroComunicacionAplicacionPrueba(),
			)
			if resultado != (ports.ComunicacionProbatoria{}) || !errors.Is(err, caso.esperado) {
				t.Fatalf("clasificacion inesperada: resultado=%+v err=%v", resultado, err)
			}
		})
	}
}

func TestServicioComunicacionLlamamientoRechazaRegistroValidoConError(t *testing.T) {
	solicitud := solicitudRegistroComunicacionAplicacionPrueba()
	transaccion := &transaccionComunicacionLlamamientoPrueba{
		comunicacion:  comunicacionAplicacionPrueba(solicitud),
		errorRegistro: errors.New("resultado contradictorio"),
	}
	servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
	resultado, err := servicio.Registrar(context.Background(), solicitud)
	if resultado != (ports.ComunicacionProbatoria{}) || !errors.Is(
		err,
		ErrResultadoComunicacionLlamamientoNoConfiable,
	) {
		t.Fatalf("resultado valido con error aceptado: resultado=%+v err=%v", resultado, err)
	}
}

func TestServicioComunicacionLlamamientoRechazaResolucionValidaConError(t *testing.T) {
	solicitud := solicitudResolverComunicacionAplicacionPrueba(
		ports.RespuestaLlamamientoRenunciada,
	)
	transaccion := &transaccionComunicacionLlamamientoPrueba{
		resolucion: resolucionAplicacionPrueba(
			solicitud,
			ports.ResultadoComunicacionLlamamientoConfirmado,
			ports.PlazoLlamamientoVigente,
			estadoOutboxAplicacion(ports.OutboxSiguienteCandidatoPendiente),
		),
		errorResolucion: errors.New("resultado contradictorio"),
	}
	servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
	resultado, err := servicio.Resolver(context.Background(), solicitud)
	if resultado != (ports.ResultadoResolucionLlamamiento{}) || !errors.Is(
		err,
		ErrResultadoComunicacionLlamamientoNoConfiable,
	) {
		t.Fatalf("resolucion valida con error aceptada: resultado=%+v err=%v", resultado, err)
	}
}

func TestServicioComunicacionLlamamientoPreservaCancelacionContradictoria(t *testing.T) {
	t.Run("registrar", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		solicitud := solicitudRegistroComunicacionAplicacionPrueba()
		transaccion := &transaccionComunicacionLlamamientoPrueba{
			comunicacion: comunicacionAplicacionPrueba(solicitud), errorRegistro: errors.New("fallo"),
			antesDeRegistrar: cancelar,
		}
		servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
		resultado, err := servicio.Registrar(ctx, solicitud)
		if resultado != (ports.ComunicacionProbatoria{}) || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion no preservada: resultado=%+v err=%v", resultado, err)
		}
	})
	t.Run("resolver", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		solicitud := solicitudResolverComunicacionAplicacionPrueba(
			ports.RespuestaLlamamientoAceptada,
		)
		transaccion := &transaccionComunicacionLlamamientoPrueba{
			resolucion: resolucionAplicacionPrueba(
				solicitud,
				ports.ResultadoComunicacionLlamamientoConfirmado,
				ports.PlazoLlamamientoVigente,
				nil,
			),
			errorResolucion: errors.New("fallo"), antesDeResolver: cancelar,
		}
		servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
		resultado, err := servicio.Resolver(ctx, solicitud)
		if resultado != (ports.ResultadoResolucionLlamamiento{}) || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion no preservada: resultado=%+v err=%v", resultado, err)
		}
	})
}

func TestServicioComunicacionLlamamientoRechazaResultadoNoLigado(t *testing.T) {
	solicitud := solicitudResolverComunicacionAplicacionPrueba(
		ports.RespuestaLlamamientoAceptada,
	)
	resultado := resolucionAplicacionPrueba(
		solicitud,
		ports.ResultadoComunicacionLlamamientoConfirmado,
		ports.PlazoLlamamientoVigente,
		nil,
	)
	resultado.VersionResultante++
	transaccion := &transaccionComunicacionLlamamientoPrueba{resolucion: resultado}
	servicio := nuevoServicioComunicacionAplicacionPrueba(t, transaccion)
	obtenido, err := servicio.Resolver(context.Background(), solicitud)
	if obtenido != (ports.ResultadoResolucionLlamamiento{}) || !errors.Is(
		err,
		ErrResultadoComunicacionLlamamientoNoConfiable,
	) {
		t.Fatalf("resultado no ligado aceptado: resultado=%+v err=%v", obtenido, err)
	}
}

func nuevoServicioComunicacionAplicacionPrueba(
	t *testing.T,
	transaccion ports.TransaccionComunicacionLlamamiento,
) *ServicioComunicacionLlamamiento {
	t.Helper()
	servicio, err := NuevoServicioComunicacionLlamamiento(transaccion)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func solicitudRegistroComunicacionAplicacionPrueba() ports.SolicitudRegistrarComunicacionLlamamiento {
	return ports.SolicitudRegistrarComunicacionLlamamiento{
		ClaveIdempotencia: "218f47a6-5d2b-4c10-aa11-1234567890ab",
		OrganizacionRef:   "organizacion:aplicacion-comunicacion", ExpedienteRef: "expediente:aplicacion-comunicacion",
		LlamamientoRef: "llamamiento:aplicacion-comunicacion", VersionEsperada: 11,
		PruebaEntregaRef: "entrega:aplicacion-probatoria",
	}
}

func comunicacionAplicacionPrueba(
	solicitud ports.SolicitudRegistrarComunicacionLlamamiento,
) ports.ComunicacionProbatoria {
	entregada := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	respuestaHastaGobernada := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	return ports.ComunicacionProbatoria{
		Solicitud: solicitud, ComunicacionRef: "comunicacion:aplicacion-probatoria",
		Canal:     referenciaComunicacionAplicacionPrueba("canal", "c"),
		Politica:  referenciaComunicacionAplicacionPrueba("politica", "d"),
		ReciboRef: "recibo:aplicacion-comunicacion", VersionResultante: solicitud.VersionEsperada + 1,
		EntregadaEn: entregada, RespuestaHasta: respuestaHastaGobernada,
		Estado: ports.ResultadoComunicacionLlamamientoConfirmado,
	}
}

func solicitudResolverComunicacionAplicacionPrueba(
	respuesta ports.RespuestaLlamamiento,
) ports.SolicitudResolverLlamamiento {
	prueba := "respuesta:aplicacion-probatoria"
	if respuesta == ports.RespuestaLlamamientoExpirada {
		prueba = ""
	}
	return ports.SolicitudResolverLlamamiento{
		ClaveIdempotencia: "318f47a6-5d2b-4c10-ba11-1234567890ab",
		OrganizacionRef:   "organizacion:aplicacion-comunicacion", ExpedienteRef: "expediente:aplicacion-comunicacion",
		LlamamientoRef: "llamamiento:aplicacion-comunicacion", ComunicacionRef: "comunicacion:aplicacion-probatoria",
		VersionEsperada: 12, Respuesta: respuesta, PruebaRespuestaRef: prueba,
	}
}

func resolucionAplicacionPrueba(
	solicitud ports.SolicitudResolverLlamamiento,
	estado ports.EstadoResultadoComunicacionLlamamiento,
	plazo ports.EstadoPlazoLlamamiento,
	estadoOutbox *ports.EstadoOutboxSiguienteCandidato,
) ports.ResultadoResolucionLlamamiento {
	resuelta := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	resultado := ports.ResultadoResolucionLlamamiento{
		Solicitud: solicitud, Politica: referenciaComunicacionAplicacionPrueba("politica", "d"),
		EvaluacionPlazoRef: "evaluacion:aplicacion-plazo", EstadoPlazo: plazo,
		ResolucionRef: "resolucion:aplicacion-local", ReciboLocalRef: "recibo:aplicacion-local",
		VersionResultante: solicitud.VersionEsperada + 1, ResueltaEn: resuelta, Estado: estado,
	}
	if estadoOutbox != nil {
		resultado.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{
			IntencionRef: "outbox:aplicacion-siguiente", ComandoOpacoRef: "comando:aplicacion-siguiente",
			Estado: *estadoOutbox, ActualizadaEn: resuelta,
		}
	}
	return resultado
}

func referenciaComunicacionAplicacionPrueba(
	referencia string,
	digito string,
) ports.ReferenciaGobernadaComunicacionLlamamiento {
	return ports.ReferenciaGobernadaComunicacionLlamamiento{
		Referencia: "catalogo:" + referencia, Version: 5, HuellaSHA256: strings.Repeat(digito, 64),
	}
}

func estadoOutboxAplicacion(
	estado ports.EstadoOutboxSiguienteCandidato,
) *ports.EstadoOutboxSiguienteCandidato {
	return &estado
}
