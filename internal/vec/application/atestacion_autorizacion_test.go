package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type firmanteAtestacionAplicacionPrueba struct {
	err            error
	invocaciones   int
	firmadaEn      time.Time
	solicitudAjena *ports.SolicitudFirmaAtestacionAutorizacionV1
	resultadoVacio bool
}

func (f *firmanteAtestacionAplicacionPrueba) FirmarAtestacionAutorizacionV1(
	_ context.Context,
	solicitud ports.SolicitudFirmaAtestacionAutorizacionV1,
) (ports.ResultadoFirmaAtestacionAutorizacionV1, error) {
	f.invocaciones++
	if f.err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV1{}, f.err
	}
	if f.resultadoVacio {
		return ports.ResultadoFirmaAtestacionAutorizacionV1{}, nil
	}
	base := solicitud
	if f.solicitudAjena != nil {
		base = *f.solicitudAjena
	}
	return ports.NuevoResultadoFirmaAtestacionAutorizacionV1(
		base, []byte{1, 2, 3, 4}, "evidencia:firma:aplicacion", f.firmadaEn,
	)
}

func TestServicioAtestacionesUsaCabeceraPreconfiguradaYResultadoLigado(t *testing.T) {
	decision := decisionAtestacionAplicacionPrueba(t, "decision:atestacion:aplicacion")
	cabecera := cabeceraAtestacionAplicacionPrueba()
	firmante := &firmanteAtestacionAplicacionPrueba{
		firmadaEn: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
	}
	servicio, err := NuevoServicioAtestacionesAutorizacionV1(cabecera, firmante)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	atestacion, err := servicio.Atestar(context.Background(), decision)
	if err != nil || firmante.invocaciones != 1 {
		t.Fatalf("atestar: invocaciones=%d err=%v", firmante.invocaciones, err)
	}
	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV1(cabecera, decision)
	if err != nil || atestacion.ValidarPara(solicitud) != nil {
		t.Fatalf("resultado no ligado: %v", err)
	}
	recuperada, err := atestacion.Cabecera()
	if err != nil || recuperada != cabecera {
		t.Fatalf("cabecera sustituida: %+v err=%v", recuperada, err)
	}
}

func TestServicioAtestacionesFallaAntesDelFirmanteConDecisionNoConcedida(t *testing.T) {
	decision := decisionAtestacionAplicacionPrueba(t, "decision:atestacion:denegada")
	decision.Concedida = false
	decision.Codigo = "accion_no_concedida"
	firmante := &firmanteAtestacionAplicacionPrueba{firmadaEn: time.Now().UTC().Truncate(time.Microsecond)}
	servicio, err := NuevoServicioAtestacionesAutorizacionV1(cabeceraAtestacionAplicacionPrueba(), firmante)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	if _, err := servicio.Atestar(context.Background(), decision); !errors.Is(err, ports.ErrFirmaAtestacionNoDisponible) {
		t.Fatalf("decision negativa no fallo cerrada: %v", err)
	}
	if firmante.invocaciones != 0 {
		t.Fatalf("una denegacion alcanzo el firmante: %d", firmante.invocaciones)
	}
}

func TestServicioAtestacionesRechazaResultadoDeOtraDecision(t *testing.T) {
	cabecera := cabeceraAtestacionAplicacionPrueba()
	ajena, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV1(
		cabecera, decisionAtestacionAplicacionPrueba(t, "decision:atestacion:ajena"),
	)
	if err != nil {
		t.Fatalf("solicitud ajena: %v", err)
	}
	firmante := &firmanteAtestacionAplicacionPrueba{
		firmadaEn:      time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
		solicitudAjena: &ajena,
	}
	servicio, err := NuevoServicioAtestacionesAutorizacionV1(cabecera, firmante)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	_, err = servicio.Atestar(
		context.Background(), decisionAtestacionAplicacionPrueba(t, "decision:atestacion:propia"),
	)
	if !errors.Is(err, ports.ErrFirmaAtestacionNoDisponible) {
		t.Fatalf("resultado cruzado aceptado: %v", err)
	}
}

func TestServicioAtestacionesNoFiltraErrorDelProveedorYRespetaCancelacion(t *testing.T) {
	firmante := &firmanteAtestacionAplicacionPrueba{err: errors.New("hsm token=secreto clave_privada")}
	servicio, err := NuevoServicioAtestacionesAutorizacionV1(cabeceraAtestacionAplicacionPrueba(), firmante)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	_, err = servicio.Atestar(
		context.Background(), decisionAtestacionAplicacionPrueba(t, "decision:atestacion:error"),
	)
	if !errors.Is(err, ports.ErrFirmaAtestacionNoDisponible) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("error inseguro: %v", err)
	}

	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	antes := firmante.invocaciones
	_, err = servicio.Atestar(ctx, decisionAtestacionAplicacionPrueba(t, "decision:atestacion:cancelada"))
	if !errors.Is(err, context.Canceled) || firmante.invocaciones != antes {
		t.Fatalf("cancelacion no respetada: invocaciones=%d err=%v", firmante.invocaciones, err)
	}
}

func TestNuevoServicioAtestacionesRechazaFirmanteNuloTipado(t *testing.T) {
	var firmante *firmanteAtestacionAplicacionPrueba
	servicio, err := NuevoServicioAtestacionesAutorizacionV1(cabeceraAtestacionAplicacionPrueba(), firmante)
	if servicio != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("firmante nulo aceptado: servicio=%v err=%v", servicio, err)
	}
}

func cabeceraAtestacionAplicacionPrueba() domain.CabeceraAtestacionAutorizacionV1 {
	return domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
		Suite:          "VEC-AD-PRUEBA-1",
		ClaveID:        "clave:prueba:aplicacion",
		Audiencia:      "vec/pruebas/aplicacion",
	}
}

func decisionAtestacionAplicacionPrueba(t *testing.T, referencia string) domain.DecisionAutorizacion {
	t.Helper()
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantaneaAutorizacionServicioPrueba(t)}
	registro := &registroAutorizacionServicioPrueba{}
	servicio := nuevoServicioAutorizacionServicioPrueba(
		t, fuente, registro,
		&generadorAutorizacionServicioPrueba{referencia: referencia},
		time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC),
	)
	decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if err != nil {
		t.Fatalf("crear decision concedida: %v", err)
	}
	return decision
}
