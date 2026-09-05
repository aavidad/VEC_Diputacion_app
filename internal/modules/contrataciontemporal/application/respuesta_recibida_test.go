package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type registroRespuestasRecibidasPrueba func(context.Context, ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error)

func (r registroRespuestasRecibidasPrueba) RegistrarRespuestaRecibida(ctx context.Context, solicitud ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
	return r(ctx, solicitud)
}

func solicitudRespuestaRecibidaServicioPrueba() ports.SolicitudRegistrarRespuestaRecibida {
	return ports.SolicitudRegistrarRespuestaRecibida{
		ClaveIdempotencia:           "e53cb792-4c62-4daf-8c80-d5d18521748a",
		OrganizacionRef:             "org:desarrollo",
		ExpedienteRef:               "expediente:sintetico",
		LlamamientoRef:              "llamamiento:sintetico",
		ComunicacionRef:             "comunicacion:sintetica",
		VersionComunicacionEsperada: 2,
		Respuesta:                   ports.RespuestaLlamamientoAceptada,
		CorreoRef:                   "correo:declarado-sintetico",
		CorreoSHA256:                strings.Repeat("a", 64),
		RecibidaEn:                  time.Date(2026, 9, 5, 12, 0, 0, 123456000, time.UTC),
	}
}

func resultadoRespuestaRecibidaServicioPrueba(solicitud ports.SolicitudRegistrarRespuestaRecibida) ports.RespuestaRecibidaRegistrada {
	return ports.RespuestaRecibidaRegistrada{
		Solicitud:       solicitud,
		JustificanteRef: "justificante:sintetico",
		ReciboRef:       "recibo:sintetico",
		AuditoriaRef:    "auditoria:sintetica",
		RegistradaEn:    solicitud.RecibidaEn.Add(time.Minute),
		Estado:          ports.EstadoRespuestaRecibidaRegistrada,
	}
}

func TestServicioRespuestasRecibidasRegistroYReplayDeleganCadaPeticion(t *testing.T) {
	ctx := context.Background()
	solicitud := solicitudRespuestaRecibidaServicioPrueba()
	esperado := resultadoRespuestaRecibidaServicioPrueba(solicitud)
	llamadas := 0
	registro := registroRespuestasRecibidasPrueba(func(actual context.Context, recibida ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
		llamadas++
		if actual != ctx || recibida != solicitud {
			t.Fatal("la petición o el contexto de autoridad se sustituyeron")
		}
		resultado := esperado
		if llamadas == 2 {
			resultado.Estado = ports.EstadoRespuestaRecibidaReplay
		}
		return resultado, nil
	})
	servicio, err := NuevoServicioRespuestasRecibidas(registro)
	if err != nil {
		t.Fatal(err)
	}
	for intento := 1; intento <= 2; intento++ {
		resultado, err := servicio.Registrar(ctx, solicitud)
		if intento == 2 {
			esperado.Estado = ports.EstadoRespuestaRecibidaReplay
		}
		if err != nil || resultado != esperado || llamadas != intento {
			t.Fatalf("registro/replay no conservado: llamadas=%d err=%v", llamadas, err)
		}
	}
}

func TestServicioRespuestasRecibidasRechazaEntradaAntesDeRegistro(t *testing.T) {
	var registroNulo registroRespuestasRecibidasPrueba
	for _, registro := range []ports.RegistroRespuestasRecibidas{nil, registroNulo} {
		if servicio, err := NuevoServicioRespuestasRecibidas(registro); servicio != nil || !errors.Is(err, ErrServicioRespuestasRecibidasInvalido) {
			t.Fatal("constructor aceptó dependencia nula")
		}
	}
	registro := registroRespuestasRecibidasPrueba(func(context.Context, ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
		t.Fatal("entrada inválida llegó a persistencia")
		return ports.RespuestaRecibidaRegistrada{}, nil
	})
	servicio, err := NuevoServicioRespuestasRecibidas(registro)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudRespuestaRecibidaServicioPrueba()
	invalida := solicitud
	invalida.Respuesta = ports.RespuestaLlamamientoExpirada
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	vencido, liberar := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer liberar()
	for _, caso := range []struct {
		nombre    string
		servicio  *ServicioRespuestasRecibidas
		ctx       context.Context
		solicitud ports.SolicitudRegistrarRespuestaRecibida
		err       error
	}{
		{"servicio_nulo", nil, context.Background(), solicitud, ErrServicioRespuestasRecibidasInvalido},
		{"sin_registro", &ServicioRespuestasRecibidas{}, context.Background(), solicitud, ErrServicioRespuestasRecibidasInvalido},
		{"contexto_nulo", servicio, nil, solicitud, ErrServicioRespuestasRecibidasInvalido},
		{"solicitud_invalida", servicio, context.Background(), invalida, ErrSolicitudRespuestaRecibidaInvalida},
		{"cancelada", servicio, cancelado, solicitud, context.Canceled},
		{"vencida", servicio, vencido, solicitud, context.DeadlineExceeded},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := caso.servicio.Registrar(caso.ctx, caso.solicitud)
			if !errors.Is(err, caso.err) || resultado != (ports.RespuestaRecibidaRegistrada{}) {
				t.Fatalf("esperado %v, recibido %v", caso.err, err)
			}
		})
	}
}

func TestServicioRespuestasRecibidasClasificaErroresSinFiltrarProveedor(t *testing.T) {
	casos := []struct {
		origen error
		salida error
	}{
		{ports.ErrSolicitudRespuestaRecibidaInvalida, ErrSolicitudRespuestaRecibidaInvalida},
		{ports.ErrOperacionRespuestaRecibidaDenegada, ErrRespuestaRecibidaDenegada},
		{ports.ErrClaveRespuestaRecibidaUsada, ErrClaveRespuestaRecibidaEnColision},
		{ports.ErrVersionRespuestaRecibidaEnConflicto, ErrVersionRespuestaRecibidaEnConflicto},
		{ports.ErrRespuestaRecibidaNoDisponible, ErrRespuestaRecibidaNoDisponible},
		{ports.ErrResultadoRespuestaRecibidaNoConfiable, ErrResultadoRespuestaRecibidaNoConfiable},
		{context.Canceled, context.Canceled},
		{context.DeadlineExceeded, context.DeadlineExceeded},
		{errors.New("detalle interno sintético no publicable"), ErrRespuestaRecibidaNoDisponible},
	}
	for _, caso := range casos {
		t.Run(caso.salida.Error(), func(t *testing.T) {
			registro := registroRespuestasRecibidasPrueba(func(context.Context, ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
				return ports.RespuestaRecibidaRegistrada{}, fmt.Errorf("proveedor sintético: %w", caso.origen)
			})
			servicio, err := NuevoServicioRespuestasRecibidas(registro)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := servicio.Registrar(context.Background(), solicitudRespuestaRecibidaServicioPrueba())
			if err != caso.salida || resultado != (ports.RespuestaRecibidaRegistrada{}) {
				t.Fatalf("clasificación inesperada: %v", err)
			}
		})
	}
}

func TestServicioRespuestasRecibidasDescartaResultadosNoConfiables(t *testing.T) {
	solicitud := solicitudRespuestaRecibidaServicioPrueba()
	valido := resultadoRespuestaRecibidaServicioPrueba(solicitud)
	ajeno := valido
	ajeno.Solicitud.ExpedienteRef = "expediente:otro"
	for _, caso := range []struct {
		nombre    string
		resultado ports.RespuestaRecibidaRegistrada
		err       error
	}{
		{"vacio_sin_error", ports.RespuestaRecibidaRegistrada{}, nil},
		{"otro_expediente", ajeno, nil},
		{"recibo_con_error", valido, ports.ErrOperacionRespuestaRecibidaDenegada},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			registro := registroRespuestasRecibidasPrueba(func(context.Context, ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
				return caso.resultado, caso.err
			})
			servicio, err := NuevoServicioRespuestasRecibidas(registro)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := servicio.Registrar(context.Background(), solicitud)
			if err != ErrResultadoRespuestaRecibidaNoConfiable || resultado != (ports.RespuestaRecibidaRegistrada{}) {
				t.Fatalf("resultado no confiable entregado: %v", err)
			}
		})
	}
}

func TestServicioRespuestasRecibidasCancelacionPosteriorNoAfirmaExitoNiRollback(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	llamadas := 0
	registro := registroRespuestasRecibidasPrueba(func(_ context.Context, solicitud ports.SolicitudRegistrarRespuestaRecibida) (ports.RespuestaRecibidaRegistrada, error) {
		llamadas++
		cancelar()
		return resultadoRespuestaRecibidaServicioPrueba(solicitud), nil
	})
	servicio, err := NuevoServicioRespuestasRecibidas(registro)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := servicio.Registrar(ctx, solicitudRespuestaRecibidaServicioPrueba())
	if err != context.Canceled || llamadas != 1 || resultado != (ports.RespuestaRecibidaRegistrada{}) {
		t.Fatalf("cancelación posterior incorrecta: %v", err)
	}
}
