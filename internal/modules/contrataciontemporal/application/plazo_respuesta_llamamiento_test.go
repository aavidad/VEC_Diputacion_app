package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type reglasPlazoPrueba struct {
	llamadas int
	inicio   time.Time
	plazo    ports.PlazoRespuestaGobernado
	err      error
}

func (r *reglasPlazoPrueba) PlazoRespuesta(_ context.Context, inicio time.Time) (ports.PlazoRespuestaGobernado, error) {
	r.llamadas++
	r.inicio = inicio
	return r.plazo, r.err
}

type registroPlazoPrueba struct {
	recibido *ports.PlazoRespuestaGobernado
	estado   string
	err      error
}

func (r *registroPlazoPrueba) RegistrarEventoPlazo(_ context.Context, s ports.SolicitudRegistrarEventoPlazoLlamamiento, p *ports.PlazoRespuestaGobernado) (ports.EventoPlazoLlamamientoRegistrado, error) {
	r.recibido = p
	if r.err != nil {
		return ports.EventoPlazoLlamamientoRegistrado{}, r.err
	}
	estado := r.estado
	if estado == "" {
		estado = ports.EstadoEventoPlazoRegistrado
	}
	return ports.EventoPlazoLlamamientoRegistrado{Solicitud: s, Plazo: p, EventoRef: "evento:1", ReciboRef: "recibo:1",
		AuditoriaRef: "auditoria:1", RegistradoEn: s.InstanteEn.Add(time.Minute), Estado: estado}, nil
}

func solicitudPlazoAplicacion(tipo ports.TipoEventoPlazoLlamamiento) ports.SolicitudRegistrarEventoPlazoLlamamiento {
	return ports.SolicitudRegistrarEventoPlazoLlamamiento{
		ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: "organizacion:plazo",
		ExpedienteRef: "expediente:plazo", LlamamientoRef: "llamamiento:plazo", ComunicacionRef: "comunicacion:plazo",
		VersionComunicacionEsperada: 2, Tipo: tipo, InstanteEn: time.Date(2026, 9, 28, 9, 30, 0, 0, time.UTC),
		PruebaRef: "prueba:llamada",
	}
}

func plazoAplicacionPrueba() ports.PlazoRespuestaGobernado {
	ref := func(e string) ports.ReferenciaGobernadaComunicacionLlamamiento {
		return ports.ReferenciaGobernadaComunicacionLlamamiento{Referencia: "vec.bolsa.reglas:1:" + e, Version: 1, HuellaSHA256: strings.Repeat("d", 64)}
	}
	return ports.PlazoRespuestaGobernado{RespuestaHasta: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC), UltimoDia: "2026-09-29",
		Politica: ref("b05.plazo_respuesta"), TratamientoFueraDePlazo: domain.TratamientoFueraDePlazoExigeCausaJustificada,
		ConfirmacionExpiracion: domain.ConfirmacionExpiracionRRHH, CriterioRespuesta: ref("b07.fuera_de_plazo"),
		CriterioExpiracion: ref("b08.sin_respuesta_baja")}
}

func TestServicioPlazoCalculaElVencimientoDesdeElContactoEfectivo(t *testing.T) {
	reglas := &reglasPlazoPrueba{plazo: plazoAplicacionPrueba()}
	registro := &registroPlazoPrueba{}
	servicio, err := NuevoServicioEventosPlazoLlamamiento(reglas, registro)
	if err != nil {
		t.Fatal(err)
	}
	s := solicitudPlazoAplicacion(ports.EventoPlazoContactoEfectivo)
	r, err := servicio.Registrar(context.Background(), s)
	if err != nil || reglas.llamadas != 1 || !reglas.inicio.Equal(s.InstanteEn) || registro.recibido == nil ||
		*registro.recibido != reglas.plazo || r.Plazo == nil {
		t.Fatalf("el vencimiento debe calcularse en el servidor: %+v %v", r, err)
	}
	causa := solicitudPlazoAplicacion(ports.EventoPlazoCausaJustificada)
	if r, err := servicio.Registrar(context.Background(), causa); err != nil || reglas.llamadas != 1 || r.Plazo != nil || registro.recibido != nil {
		t.Fatalf("la causa justificada no recalcula plazos: %+v %v", r, err)
	}
}

func TestServicioPlazoNoSuponePlazoNiFiltraErrores(t *testing.T) {
	s := solicitudPlazoAplicacion(ports.EventoPlazoContactoEfectivo)
	sinReglas := &reglasPlazoPrueba{err: errors.New("catálogo privado caído")}
	registro := &registroPlazoPrueba{}
	servicio, _ := NuevoServicioEventosPlazoLlamamiento(sinReglas, registro)
	if _, err := servicio.Registrar(context.Background(), s); !errors.Is(err, ErrReglasPlazoLlamamientoNoDisponibles) || registro.recibido != nil {
		t.Fatalf("sin reglas no hay plazo supuesto: %v", err)
	}
	invalido := plazoAplicacionPrueba()
	invalido.RespuestaHasta = s.InstanteEn
	servicio, _ = NuevoServicioEventosPlazoLlamamiento(&reglasPlazoPrueba{plazo: invalido}, registro)
	if _, err := servicio.Registrar(context.Background(), s); !errors.Is(err, ErrReglasPlazoLlamamientoNoDisponibles) {
		t.Fatalf("vencimiento incoherente aceptado: %v", err)
	}
	for causa, esperado := range map[error]error{
		ports.ErrClaveEventoPlazoUsada:        ErrClaveEventoPlazoEnColision,
		ports.ErrEventoPlazoEnConflicto:       ErrEventoPlazoEnConflicto,
		ports.ErrOperacionEventoPlazoDenegada: ErrEventoPlazoDenegado,
		errors.New("detalle sql privado"):     ErrEventoPlazoNoDisponible,
	} {
		servicio, _ = NuevoServicioEventosPlazoLlamamiento(&reglasPlazoPrueba{plazo: plazoAplicacionPrueba()}, &registroPlazoPrueba{err: causa})
		if _, err := servicio.Registrar(context.Background(), s); !errors.Is(err, esperado) || strings.Contains(err.Error(), "privado") {
			t.Errorf("%v -> %v", causa, err)
		}
	}
	servicio, _ = NuevoServicioEventosPlazoLlamamiento(&reglasPlazoPrueba{plazo: plazoAplicacionPrueba()}, &registroPlazoPrueba{estado: "otro"})
	if _, err := servicio.Registrar(context.Background(), s); !errors.Is(err, ErrResultadoEventoPlazoNoConfiable) {
		t.Fatalf("recibo no confiable aceptado: %v", err)
	}
	if _, err := NuevoServicioEventosPlazoLlamamiento(nil, registro); !errors.Is(err, ErrServicioEventosPlazoInvalido) {
		t.Fatal("servicio sin reglas compuesto")
	}
}

func TestResolucionFueraDePlazoYAntesDeVencerSeClasifican(t *testing.T) {
	ctx := context.Background()
	if err := clasificarErrorComunicacionLlamamiento(ctx, ports.ErrRespuestaFueraDePlazoLlamamiento); !errors.Is(err, ErrValidacionRespuestaLlamamientoPendiente) {
		t.Fatalf("respuesta tardía: %v", err)
	}
	if err := clasificarErrorComunicacionLlamamiento(ctx, ports.ErrPlazoRespuestaNoVencido); !errors.Is(err, ErrPlazoRespuestaLlamamientoNoVencido) {
		t.Fatalf("expiración anticipada: %v", err)
	}
}
