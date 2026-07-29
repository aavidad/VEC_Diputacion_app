package ports_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type relojGuardianConsultaRRHHPrueba struct {
	mu        sync.Mutex
	instantes []time.Time
	llamadas  int
}

func (r *relojGuardianConsultaRRHHPrueba) Ahora() time.Time {
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

type resolutorMotivoGuardianConsultaRRHHPrueba struct {
	mu              sync.Mutex
	motivoCuadro    dominiovec.ReferenciaEntradaCatalogo
	motivoDetalle   dominiovec.ReferenciaEntradaCatalogo
	errCuadro       error
	errDetalle      error
	cancelarCuadro  context.CancelFunc
	llamadasCuadro  int
	llamadasDetalle int
}

func (r *resolutorMotivoGuardianConsultaRRHHPrueba) ResolverMotivoCuadroRRHH(
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

func (r *resolutorMotivoGuardianConsultaRRHHPrueba) ResolverMotivoDetalleRRHH(
	context.Context,
	time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadasDetalle++
	return r.motivoDetalle, r.errDetalle
}

type generadorCorrelacionGuardianConsultaRRHHPrueba struct {
	mu         sync.Mutex
	referencia string
	err        error
	cancelar   context.CancelFunc
	llamadas   int
}

func (g *generadorCorrelacionGuardianConsultaRRHHPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.llamadas++
	if g.cancelar != nil {
		g.cancelar()
	}
	return g.referencia, g.err
}

type emisorGuardianConsultaRRHHPrueba struct {
	mu                sync.Mutex
	t                 *testing.T
	audiencia         string
	instanteMaterial  time.Time
	err               error
	cancelar          context.CancelFunc
	exportadorNulo    bool
	exportadorCancela context.CancelFunc
	llamadas          int
	ultimaSolicitud   dominiovec.SolicitudAutorizacionLigadaV3
	ultimoResultado   dominiovec.ResultadoContextoActorRegistradoV2
}

func (e *emisorGuardianConsultaRRHHPrueba) EmitirMaterialAutorizacionAtestadaV3(
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
	decision, confirmacion, exportacion := materialGuardianConsultaRRHHPrueba(
		e.t, solicitud, resultado, e.audiencia, e.instanteMaterial,
	)
	if e.cancelar != nil {
		e.cancelar()
	}
	if e.err != nil {
		return decision, confirmacion, nil, e.err
	}
	exportador := &exportadorGuardianConsultaRRHHPrueba{
		exportacion: exportacion,
		cancelar:    e.exportadorCancela,
	}
	if e.exportadorNulo {
		exportador = nil
	}
	return decision, confirmacion, exportador, nil
}

type exportadorGuardianConsultaRRHHPrueba struct {
	exportacion puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	cancelar    context.CancelFunc
}

func (*exportadorGuardianConsultaRRHHPrueba) String() string {
	return "[MATERIAL-GUARDIAN-RRHH-PRUEBA-OPACO]"
}

func (*exportadorGuardianConsultaRRHHPrueba) LogValue() slog.Value {
	return slog.StringValue("[MATERIAL-GUARDIAN-RRHH-PRUEBA-OPACO]")
}

func (e *exportadorGuardianConsultaRRHHPrueba) ExportarMaterialParaConsumidor() (
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	if e.cancelar != nil {
		e.cancelar()
	}
	return e.exportacion, nil
}

func materialGuardianConsultaRRHHPrueba(
	t *testing.T,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	audiencia string,
	instante time.Time,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	decision, confirmacion := concesionConsultaRRHHPrueba(
		t, solicitud, resultado, datos.ReferenciaMotivo,
		instante.Add(-2*time.Millisecond),
	)
	decisionHuella, err :=
		dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	motivoHuella, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(
		datos.ReferenciaMotivo,
	)
	if err != nil {
		t.Fatal(err)
	}
	recursoHuella, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	datosConfirmacion, err := confirmacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	emitidaEn := instante.Add(-time.Millisecond)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		datosConfirmacion.DecisionRef, decisionHuella, motivoHuella,
		resultado.RegistroContextoRef, resultado.HuellaSHA256,
		datos.Accion, datos.Recurso.Referencia, recursoHuella,
		audiencia, emitidaEn, emitidaEn.Add(5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decisionCanonica, err :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	motivoCanonico, err :=
		dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
			datos.ReferenciaMotivo,
		)
	if err != nil {
		t.Fatal(err)
	}
	semilla := [ed25519.SeedSize]byte{7}
	privada := ed25519.NewKeyFromSeed(semilla[:])
	raizSPKI, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err :=
		puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
			bytes.Repeat([]byte{0xa5}, puertosvec.TamanoMinimoCapacidadCanonicaV3),
			resumen, decisionCanonica, motivoCanonico,
			resultado.RepresentacionCanonica,
			resultado.Contexto.Instantanea.PersonaVersion,
			resultado.Contexto.Instantanea.PerfilVersion,
			[]byte("payload-vec-ad-3-guardian-prueba"),
			[]byte("sobre-cose-sign1-guardian-prueba"),
			[]byte("evidencia-verificacion-guardian-prueba"), raizSPKI,
		)
	if err != nil {
		t.Fatal(err)
	}
	return decision, confirmacion, exportacion
}

func motivoGuardianConsultaRRHHPrueba(
	marca string,
) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      2,
		CatalogoHuellaSHA256: strings.Repeat(marca, 64),
		EntradaClave:         "motivo_" + strings.Repeat(marca, 32),
	}
}

func nuevoGuardianConsultaRRHHPrueba(
	t *testing.T,
	instantes []time.Time,
) (
	*ports.EmisorMaterialConsultaRRHH,
	*resolutorMotivoGuardianConsultaRRHHPrueba,
	*generadorCorrelacionGuardianConsultaRRHHPrueba,
	*relojGuardianConsultaRRHHPrueba,
	*emisorGuardianConsultaRRHHPrueba,
	*emisorGuardianConsultaRRHHPrueba,
) {
	t.Helper()
	motivos := &resolutorMotivoGuardianConsultaRRHHPrueba{
		motivoCuadro:  motivoGuardianConsultaRRHHPrueba("6"),
		motivoDetalle: motivoGuardianConsultaRRHHPrueba("7"),
	}
	correlaciones := &generadorCorrelacionGuardianConsultaRRHHPrueba{
		referencia: "correlacion_" + strings.Repeat("8", 32),
	}
	reloj := &relojGuardianConsultaRRHHPrueba{instantes: instantes}
	final := instantes[len(instantes)-1]
	cuadro := &emisorGuardianConsultaRRHHPrueba{
		t: t, audiencia: ports.AudienciaConsumoConsultaCuadroRRHHV3,
		instanteMaterial: final,
	}
	detalle := &emisorGuardianConsultaRRHHPrueba{
		t: t, audiencia: ports.AudienciaConsumoConsultaDetalleRRHHV3,
		instanteMaterial: final,
	}
	guardian, err := ports.NuevoEmisorMaterialConsultaRRHH(
		motivos, correlaciones, reloj, cuadro, detalle,
	)
	if err != nil {
		t.Fatalf("crear guardián RRHH: %v", err)
	}
	return guardian, motivos, correlaciones, reloj, cuadro, detalle
}

func errorPrivadoGuardianConsultaRRHHPrueba() error {
	return errors.New("SECRETO-PRIVADO-GUARDIAN-RRHH")
}
