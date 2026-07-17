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

type firmanteAtestacionV2AplicacionPrueba struct {
	err            error
	invocaciones   int
	firmadaEn      time.Time
	solicitudAjena *ports.SolicitudFirmaAtestacionAutorizacionV2
	resultadoVacio bool
}

func (f *firmanteAtestacionV2AplicacionPrueba) FirmarAtestacionAutorizacionV2(
	_ context.Context,
	solicitud ports.SolicitudFirmaAtestacionAutorizacionV2,
) (ports.ResultadoFirmaAtestacionAutorizacionV2, error) {
	f.invocaciones++
	if f.err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV2{}, f.err
	}
	if f.resultadoVacio {
		return ports.ResultadoFirmaAtestacionAutorizacionV2{}, nil
	}
	base := solicitud
	if f.solicitudAjena != nil {
		base = *f.solicitudAjena
	}
	return ports.NuevoResultadoFirmaAtestacionAutorizacionV2(
		base,
		[]byte{1, 2, 3, 4},
		"evidencia:firma:aplicacion:v2",
		f.firmadaEn,
	)
}

func TestServicioAtestacionesV2FirmaMensajeLigadoCompleto(t *testing.T) {
	referenciaMotivo := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	decision := decisionAtestacionV2AplicacionPrueba(t, "decision:atestacion:v2", referenciaMotivo)
	cabecera := cabeceraAtestacionV2AplicacionPrueba()
	firmante := &firmanteAtestacionV2AplicacionPrueba{
		firmadaEn: time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC),
	}
	servicio, err := NuevoServicioAtestacionesAutorizacionV2(cabecera, firmante)
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := servicio.Atestar(context.Background(), decision, referenciaMotivo)
	if err != nil || firmante.invocaciones != 1 {
		t.Fatalf("atestar V2: invocaciones=%d err=%v", firmante.invocaciones, err)
	}
	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil || atestacion.ValidarPara(solicitud) != nil {
		t.Fatalf("atestacion no ligada al mensaje V2: %v", err)
	}
	conservada, err := atestacion.Solicitud()
	if err != nil {
		t.Fatal(err)
	}
	huellaSolicitud, _ := conservada.HuellaSolicitudLigadaSHA256()
	huellaMotivo, _ := conservada.HuellaMotivoCatalogoSHA256()
	if huellaSolicitud != decision.SolicitudHuellaSHA256 ||
		huellaMotivo != decision.MotivoHuellaSHA256 {
		t.Fatal("la atestacion perdio los compromisos de solicitud o motivo")
	}
}

func TestServicioAtestacionesV2FallaAntesDelFirmanteConEntradaIncoherente(t *testing.T) {
	referenciaMotivo := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	decision := decisionAtestacionV2AplicacionPrueba(t, "decision:atestacion:v2:invalida", referenciaMotivo)
	firmante := &firmanteAtestacionV2AplicacionPrueba{
		firmadaEn: time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC),
	}
	servicio, err := NuevoServicioAtestacionesAutorizacionV2(
		cabeceraAtestacionV2AplicacionPrueba(),
		firmante,
	)
	if err != nil {
		t.Fatal(err)
	}

	ajena := referenciaMotivo
	ajena.EntradaClave = claveMotivoAutorizacionV2Alternativa
	if _, err := servicio.Atestar(
		context.Background(),
		decision,
		ajena,
	); !errors.Is(err, ports.ErrFirmaAtestacionNoDisponible) {
		t.Fatalf("motivo ajeno aceptado: %v", err)
	}
	decision.Concedida = false
	decision.Codigo = "accion_no_concedida"
	if _, err := servicio.Atestar(
		context.Background(),
		decision,
		referenciaMotivo,
	); !errors.Is(err, ports.ErrFirmaAtestacionNoDisponible) {
		t.Fatalf("denegacion firmada como concesion: %v", err)
	}
	if firmante.invocaciones != 0 {
		t.Fatalf("una entrada invalida alcanzo el firmante: %d", firmante.invocaciones)
	}
}

func TestServicioAtestacionesV2RechazaResultadoDeOtraDecision(t *testing.T) {
	referenciaMotivo := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	cabecera := cabeceraAtestacionV2AplicacionPrueba()
	ajena, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabecera,
		decisionAtestacionV2AplicacionPrueba(t, "decision:atestacion:v2:ajena", referenciaMotivo),
		referenciaMotivo,
	)
	if err != nil {
		t.Fatal(err)
	}
	firmante := &firmanteAtestacionV2AplicacionPrueba{
		firmadaEn:      time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC),
		solicitudAjena: &ajena,
	}
	servicio, err := NuevoServicioAtestacionesAutorizacionV2(cabecera, firmante)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Atestar(
		context.Background(),
		decisionAtestacionV2AplicacionPrueba(t, "decision:atestacion:v2:propia", referenciaMotivo),
		referenciaMotivo,
	)
	if !errors.Is(err, ports.ErrFirmaAtestacionNoDisponible) {
		t.Fatalf("resultado cruzado aceptado: %v", err)
	}
}

func TestServicioAtestacionesV2OcultaProveedorYRespetaCancelacion(t *testing.T) {
	referenciaMotivo := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	firmante := &firmanteAtestacionV2AplicacionPrueba{
		err: errors.New("hsm token=secreto clave_privada"),
	}
	servicio, err := NuevoServicioAtestacionesAutorizacionV2(
		cabeceraAtestacionV2AplicacionPrueba(),
		firmante,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Atestar(
		context.Background(),
		decisionAtestacionV2AplicacionPrueba(t, "decision:atestacion:v2:error", referenciaMotivo),
		referenciaMotivo,
	)
	if !errors.Is(err, ports.ErrFirmaAtestacionNoDisponible) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("error inseguro: %v", err)
	}

	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	antes := firmante.invocaciones
	_, err = servicio.Atestar(
		ctx,
		decisionAtestacionV2AplicacionPrueba(t, "decision:atestacion:v2:cancelada", referenciaMotivo),
		referenciaMotivo,
	)
	if !errors.Is(err, context.Canceled) || firmante.invocaciones != antes {
		t.Fatalf("cancelacion ignorada: invocaciones=%d err=%v", firmante.invocaciones, err)
	}
}

func TestNuevoServicioAtestacionesV2RechazaFirmanteNuloTipado(t *testing.T) {
	var firmante *firmanteAtestacionV2AplicacionPrueba
	servicio, err := NuevoServicioAtestacionesAutorizacionV2(
		cabeceraAtestacionV2AplicacionPrueba(),
		firmante,
	)
	if servicio != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("firmante nulo aceptado: servicio=%v err=%v", servicio, err)
	}
}

func cabeceraAtestacionV2AplicacionPrueba() domain.CabeceraAtestacionAutorizacionV2 {
	return domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          "VEC-AD-PRUEBA-2",
		ClaveID:        "clave:prueba:aplicacion:v2",
		Audiencia:      "vec/pruebas/aplicacion/autorizacion-v2",
	}
}

func decisionAtestacionV2AplicacionPrueba(
	t *testing.T,
	referenciaDecision string,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) domain.DecisionAutorizacion {
	t.Helper()
	decision := decisionAtestacionAplicacionPrueba(t, referenciaDecision)
	decision.CorrelacionRef = referenciaCorrelacionAutorizacionV2Prueba
	decision.EsquemaHuellaSolicitud = domain.EsquemaHuellaSolicitudAutorizacionV2
	decision.SolicitudHuellaSHA256 = strings.Repeat("7", 64)
	decision.EsquemaHuellaMotivo = domain.EsquemaHuellaMotivoAutorizacionV2
	var err error
	decision.MotivoHuellaSHA256, err = domain.HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	if err := decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
		t.Fatalf("decision V2 de prueba invalida: %v", err)
	}
	return decision
}
