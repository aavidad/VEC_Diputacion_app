package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type firmanteAtestacionAutorizacionV3Prueba struct {
	firmadaEn    relojFirmaAtestacionV3Prueba
	cancelar     context.CancelFunc
	err          error
	invocaciones int
}

type relojFirmaAtestacionV3Prueba interface {
	Ahora() time.Time
}

func (f *firmanteAtestacionAutorizacionV3Prueba) FirmarAtestacionAutorizacionV3(
	_ context.Context,
	solicitud ports.SolicitudFirmaAtestacionAutorizacionV3,
) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	f.invocaciones++
	if f.cancelar != nil {
		f.cancelar()
	}
	if f.err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, f.err
	}
	return ports.NuevoResultadoFirmaAtestacionAutorizacionV3(
		solicitud,
		[]byte("firma-opaca-vec-ad-3"),
		"evidencia_firma_0123456789abcdef",
		f.firmadaEn.Ahora(),
	)
}

func TestServicioAtestacionesAutorizacionV3FirmaDecisionConcedida(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	decision, _, err := e.servicio.ExigirSolicitudLigadaV3(
		context.Background(),
		e.solicitud,
		e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	firmante := &firmanteAtestacionAutorizacionV3Prueba{
		firmadaEn: &relojAutorizacionServicioPrueba{ahora: e.ahora},
	}
	servicio, err := NuevoServicioAtestacionesAutorizacionV3(
		cabeceraAtestacionAutorizacionV3AplicacionPrueba(),
		firmante,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := servicio.Atestar(
		context.Background(),
		decision,
		datosSolicitud.ReferenciaMotivo,
		e.resultado,
	)
	if err != nil || atestacion.Validar() != nil || firmante.invocaciones != 1 {
		t.Fatalf(
			"atestación V3 inválida: error=%v invocaciones=%d",
			err,
			firmante.invocaciones,
		)
	}
}

func TestServicioAtestacionesAutorizacionV3RespetaCancelacion(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	decision, _, err := e.servicio.ExigirSolicitudLigadaV3(
		context.Background(),
		e.solicitud,
		e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud, _ := e.solicitud.Datos()

	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	firmante := &firmanteAtestacionAutorizacionV3Prueba{
		firmadaEn: &relojAutorizacionServicioPrueba{ahora: e.ahora},
	}
	servicio, _ := NuevoServicioAtestacionesAutorizacionV3(
		cabeceraAtestacionAutorizacionV3AplicacionPrueba(),
		firmante,
	)
	if _, err := servicio.Atestar(
		cancelado,
		decision,
		datosSolicitud.ReferenciaMotivo,
		e.resultado,
	); !errors.Is(err, context.Canceled) || firmante.invocaciones != 0 {
		t.Fatalf("cancelación previa no cerró: %v", err)
	}

	ctx, cancelarDurante := context.WithCancel(context.Background())
	firmante.cancelar = cancelarDurante
	if _, err := servicio.Atestar(
		ctx,
		decision,
		datosSolicitud.ReferenciaMotivo,
		e.resultado,
	); !errors.Is(err, context.Canceled) || firmante.invocaciones != 1 {
		t.Fatalf("cancelación durante firma no cerró: %v", err)
	}
}

func TestNuevoServicioAtestacionesAutorizacionV3RechazaNuloTipado(t *testing.T) {
	var firmante *firmanteAtestacionAutorizacionV3Prueba
	if servicio, err := NuevoServicioAtestacionesAutorizacionV3(
		cabeceraAtestacionAutorizacionV3AplicacionPrueba(),
		firmante,
	); err == nil || servicio != nil {
		t.Fatalf("firmante nulo tipado aceptado: %#v, %v", servicio, err)
	}
}

func cabeceraAtestacionAutorizacionV3AplicacionPrueba() domain.CabeceraAtestacionAutorizacionV3 {
	return domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          "VEC-AD-3-COSE-EDDSA-1",
		ClaveID:        "clave:prueba:vec-ad-3:2026-07",
		Audiencia:      "vec-diputacion/pruebas/contratacion-temporal",
	}
}
