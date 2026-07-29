package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const organizacionConsultaRRHHPrueba = "organizacion:diputacion-granada"

type generadorCorrelacionConsultaRRHHPrueba struct {
	mu       sync.Mutex
	valor    string
	err      error
	cancelar context.CancelFunc
	llamadas int
}

func (g *generadorCorrelacionConsultaRRHHPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.llamadas++
	if g.cancelar != nil {
		g.cancelar()
	}
	return g.valor, g.err
}

type resolutorMotivoConsultaRRHHPrueba struct {
	mu              sync.Mutex
	motivoCuadro    dominiovec.ReferenciaEntradaCatalogo
	motivoDetalle   dominiovec.ReferenciaEntradaCatalogo
	errCuadro       error
	errDetalle      error
	cancelarCuadro  context.CancelFunc
	cancelarDetalle context.CancelFunc
	llamadasCuadro  int
	llamadasDetalle int
}

func (r *resolutorMotivoConsultaRRHHPrueba) ResolverMotivoCuadroRRHH(
	context.Context,
	time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadasCuadro++
	if r.cancelarCuadro != nil {
		r.cancelarCuadro()
	}
	return r.motivoCuadro, r.errCuadro
}

func (r *resolutorMotivoConsultaRRHHPrueba) ResolverMotivoDetalleRRHH(
	context.Context,
	time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadasDetalle++
	if r.cancelarDetalle != nil {
		r.cancelarDetalle()
	}
	return r.motivoDetalle, r.errDetalle
}

type relojEmisorConsultaRRHHPrueba struct {
	mu        sync.Mutex
	instantes []time.Time
	llamadas  int
}

func (r *relojEmisorConsultaRRHHPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	if len(r.instantes) == 0 {
		return time.Time{}
	}
	instante := r.instantes[0]
	if len(r.instantes) > 1 {
		r.instantes = r.instantes[1:]
	}
	return instante
}

type emisorAtestadoConsultaRRHHPrueba struct {
	mu              sync.Mutex
	t               *testing.T
	instante        time.Time
	err             error
	cancelar        context.CancelFunc
	resultadoAjeno  *dominiovec.ResultadoContextoActorRegistradoV2
	llamadas        int
	ultimaSolicitud dominiovec.SolicitudAutorizacionLigadaV3
	ultimoResultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (e *emisorAtestadoConsultaRRHHPrueba) EmitirMaterialAutorizacionAtestadaV3(
	_ context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.llamadas++
	e.ultimaSolicitud = solicitud
	e.ultimoResultado = resultado
	resultadoExportado := resultado
	if e.resultadoAjeno != nil {
		resultadoExportado = *e.resultadoAjeno
	}
	datos, err := solicitud.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		e.t, solicitud, resultado, datos.ReferenciaMotivo, e.instante,
		"decision:rrhh:v3:prueba", true,
	)
	if err != nil {
		e.t.Fatal(err)
	}
	exportacion := exportacionMaterialConsumoConsultaRRHHPrueba(
		e.t, solicitud, decision, datos.ReferenciaMotivo, resultadoExportado,
		datos.Recurso, datos.Accion, e.instante,
	)
	if e.cancelar != nil {
		e.cancelar()
	}
	return decision, confirmacion,
		&exportadorMaterialConsumoConsultaRRHHPrueba{exportacion: exportacion},
		e.err
}

type entornoEmisorConsultaRRHHPrueba struct {
	emisor        *ports.EmisorMaterialConsultaRRHH
	motivos       *resolutorMotivoConsultaRRHHPrueba
	correlaciones *generadorCorrelacionConsultaRRHHPrueba
	reloj         *relojEmisorConsultaRRHHPrueba
	cuadro        *emisorAtestadoConsultaRRHHPrueba
	detalle       *emisorAtestadoConsultaRRHHPrueba
}

type exportadorMaterialConsumoConsultaRRHHPrueba struct {
	exportacion puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (exportadorMaterialConsumoConsultaRRHHPrueba) String() string {
	return "[EXPORTADOR-MATERIAL-CONSUMO-CONSULTA-RRHH-PRUEBA]"
}

func (e exportadorMaterialConsumoConsultaRRHHPrueba) LogValue() slog.Value {
	return slog.StringValue(e.String())
}

func (e exportadorMaterialConsumoConsultaRRHHPrueba) ExportarMaterialParaConsumidor() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	return e.exportacion, nil
}

func contextoConsultaRRHHV3Prueba(
	t *testing.T,
	ahora time.Time,
) ports.ContextoConsultaRRHH {
	t.Helper()
	autoridad := contextoAutorizacionAltaV3Prueba(t, ahora)
	contexto, err := ports.NuevoContextoConsultaRRHH(
		autoridad,
		organizacionConsultaRRHHPrueba,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear contexto de consulta RRHH V3: %v", err)
	}
	return contexto
}

func nuevoEmisorConsultaRRHHV3Prueba(
	t *testing.T,
	ahora time.Time,
) *entornoEmisorConsultaRRHHPrueba {
	t.Helper()
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	motivos := &resolutorMotivoConsultaRRHHPrueba{
		motivoCuadro: motivo, motivoDetalle: motivo,
	}
	correlaciones := &generadorCorrelacionConsultaRRHHPrueba{
		valor: "correlacion_0123456789abcdef0123456789abcdef",
	}
	reloj := &relojEmisorConsultaRRHHPrueba{
		instantes: []time.Time{ahora, ahora},
	}
	cuadro := &emisorAtestadoConsultaRRHHPrueba{
		t: t, instante: ahora,
	}
	detalle := &emisorAtestadoConsultaRRHHPrueba{
		t: t, instante: ahora,
	}
	emisor, err := ports.NuevoEmisorMaterialConsultaRRHH(
		motivos, correlaciones, reloj, cuadro, detalle,
	)
	if err != nil {
		t.Fatalf("crear emisor real A4.3 para aplicación: %v", err)
	}
	return &entornoEmisorConsultaRRHHPrueba{
		emisor: emisor, motivos: motivos, correlaciones: correlaciones,
		reloj: reloj, cuadro: cuadro, detalle: detalle,
	}
}

func capacidadConsultaCuadroRRHHV3Prueba(
	t *testing.T,
	contexto ports.ContextoConsultaRRHH,
	solicitud ports.SolicitudCuadroRRHH,
	ahora time.Time,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	emision := nuevoEmisorConsultaRRHHV3Prueba(t, ahora)
	material, err := emision.emisor.EmitirMaterialCuadroRRHH(
		context.Background(), contexto, solicitud,
	)
	if err != nil {
		t.Fatalf("emitir material de cuadro RRHH mediante A4.3: %v", err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto, material, solicitud, ahora,
	)
	if err != nil {
		t.Fatalf("crear capacidad de cuadro RRHH V3: %v", err)
	}
	return capacidad
}

func capacidadConsultaDetalleRRHHV3Prueba(
	t *testing.T,
	contexto ports.ContextoConsultaRRHH,
	solicitud ports.SolicitudDetalleRRHH,
	ahora time.Time,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	emision := nuevoEmisorConsultaRRHHV3Prueba(t, ahora)
	material, err := emision.emisor.EmitirMaterialDetalleRRHH(
		context.Background(), contexto, solicitud,
	)
	if err != nil {
		t.Fatalf("emitir material de detalle RRHH mediante A4.3: %v", err)
	}
	capacidad, err := ports.NuevaCapacidadConsultaDetalleRRHH(
		contexto, material, solicitud, ahora,
	)
	if err != nil {
		t.Fatalf("crear capacidad de detalle RRHH V3: %v", err)
	}
	return capacidad
}

func exportacionMaterialConsumoConsultaRRHHPrueba(
	t *testing.T,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	recurso dominiovec.RecursoAutorizable,
	accion string,
	ahora time.Time,
) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	decisionHuella, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatalf("calcular huella de decisión RRHH: %v", err)
	}
	motivoHuella, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatalf("calcular huella de motivo RRHH: %v", err)
	}
	recursoHuella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("calcular huella de recurso RRHH: %v", err)
	}
	decisionCanonica, err := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(
		decision,
	)
	if err != nil {
		t.Fatalf("representar decisión RRHH: %v", err)
	}
	motivoCanonico, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
		motivo,
	)
	if err != nil {
		t.Fatalf("representar motivo RRHH: %v", err)
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil || datosSolicitud.Accion != accion {
		t.Fatalf("recuperar solicitud RRHH ligada: %v", err)
	}
	audiencia := ports.AudienciaConsumoConsultaCuadroRRHHV3
	if accion == ports.AccionConsultarDetalleRRHH {
		audiencia = ports.AudienciaConsumoConsultaDetalleRRHHV3
	}
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:rrhh:v3:prueba",
		decisionHuella,
		motivoHuella,
		resultado.RegistroContextoRef,
		resultado.HuellaSHA256,
		accion,
		recurso.Referencia,
		recursoHuella,
		audiencia,
		ahora,
		ahora.Add(ports.DuracionMaximaCapacidadConsultaRRHH),
	)
	if err != nil {
		t.Fatalf("crear resumen de capacidad RRHH: %v", err)
	}
	raizPublica, err := raizPublicaSPKIConsultaRRHHPrueba()
	if err != nil {
		t.Fatalf("crear raíz pública de prueba RRHH: %v", err)
	}
	exportacion, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte{'c'}, puertosvec.TamanoMinimoCapacidadCanonicaV3),
		resumen,
		decisionCanonica,
		motivoCanonico,
		resultado.RepresentacionCanonica,
		resultado.Contexto.Instantanea.PersonaVersion,
		resultado.Contexto.Instantanea.PerfilVersion,
		[]byte("payload-vec-ad-3-estructural-prueba"),
		[]byte("sobre-cose-sign1-estructural-prueba"),
		[]byte("evidencia-verificacion-estructural-prueba"),
		raizPublica,
	)
	if err != nil {
		t.Fatalf("crear exportación probatoria completa RRHH: %v", err)
	}
	return exportacion
}

func raizPublicaSPKIConsultaRRHHPrueba() ([]byte, error) {
	semilla := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	privada := ed25519.NewKeyFromSeed(semilla)
	publica := privada.Public().(ed25519.PublicKey)
	return x509.MarshalPKIXPublicKey(publica)
}
