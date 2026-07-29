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

const (
	organizacionConsultaRRHHPrueba = "organizacion:diputacion-granada"

	ambitoOrganizacionConsultaRRHHPrueba = "organizacion_ref"
	ambitoClaseConsultaRRHHPrueba        = "clase_ambito"
	ambitoReferenciaConsultaRRHHPrueba   = "ambito_ref"
	atributoDominioConsultaRRHHPrueba    = "consulta_dominio"
	atributoHuellaConsultaRRHHPrueba     = "consulta_huella_sha256"
)

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
	audiencia       string
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
		t: t, audiencia: ports.AudienciaConsumoConsultaCuadroRRHHV3,
		instante: ahora,
	}
	detalle := &emisorAtestadoConsultaRRHHPrueba{
		t: t, audiencia: ports.AudienciaConsumoConsultaDetalleRRHHV3,
		instante: ahora,
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
	clase ports.ClaseAmbitoConsultaRRHH,
	ambitoRef string,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	huella, err := solicitud.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatalf("calcular huella de cuadro RRHH: %v", err)
	}
	recurso := recursoConsultaRRHHV3Prueba(
		contexto,
		clase,
		ambitoRef,
		ambitoRef,
		ports.TipoRecursoCuadroRRHH,
		ports.DominioHuellaConsultaCuadroRRHH,
		huella,
	)
	material := materialConsultaRRHHV3Prueba(
		t,
		contexto,
		recurso,
		ports.AccionConsultarCuadroRRHH,
		ports.FinalidadConsultarCuadroRRHH,
		ahora,
	)
	capacidad, err := ports.NuevaCapacidadConsultaCuadroRRHH(
		contexto,
		material,
		solicitud,
		ahora,
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
	clase ports.ClaseAmbitoConsultaRRHH,
	ambitoRef string,
) ports.CapacidadConsultaRRHH {
	t.Helper()
	huella, err := solicitud.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatalf("calcular huella de detalle RRHH: %v", err)
	}
	recurso := recursoConsultaRRHHV3Prueba(
		contexto,
		clase,
		ambitoRef,
		solicitud.ExpedienteRef(),
		ports.TipoRecursoExpediente,
		ports.DominioHuellaConsultaDetalleRRHH,
		huella,
	)
	material := materialConsultaRRHHV3Prueba(
		t,
		contexto,
		recurso,
		ports.AccionConsultarDetalleRRHH,
		ports.FinalidadConsultarDetalleRRHH,
		ahora,
	)
	capacidad, err := ports.NuevaCapacidadConsultaDetalleRRHH(
		contexto,
		material,
		solicitud,
		ahora,
	)
	if err != nil {
		t.Fatalf("crear capacidad de detalle RRHH V3: %v", err)
	}
	return capacidad
}

func recursoConsultaRRHHV3Prueba(
	contexto ports.ContextoConsultaRRHH,
	clase ports.ClaseAmbitoConsultaRRHH,
	ambitoRef, referencia, tipo, dominio, huella string,
) dominiovec.RecursoAutorizable {
	return dominiovec.RecursoAutorizable{
		Referencia: referencia,
		ModuloID:   ports.ModuloContratacion,
		Tipo:       tipo,
		Ambitos: map[string]string{
			ambitoOrganizacionConsultaRRHHPrueba: contexto.OrganizacionRef(),
			ambitoClaseConsultaRRHHPrueba:        string(clase),
			ambitoReferenciaConsultaRRHHPrueba:   ambitoRef,
		},
		Atributos: map[string]string{
			atributoDominioConsultaRRHHPrueba: dominio,
			atributoHuellaConsultaRRHHPrueba:  huella,
		},
	}
}

func materialConsultaRRHHV3Prueba(
	t *testing.T,
	contexto ports.ContextoConsultaRRHH,
	recurso dominiovec.RecursoAutorizable,
	accion, finalidad string,
	ahora time.Time,
) ports.MaterialAutorizacionConsultaRRHH {
	t.Helper()
	autoridad := contextoAutorizacionAltaV3Prueba(t, ahora)
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		&generadorCorrelacionConsultaRRHHPrueba{
			valor: "correlacion_0123456789abcdef0123456789abcdef",
		},
	)
	if err != nil {
		t.Fatalf("generar correlación de consulta RRHH: %v", err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: autoridad.Vinculo,
			ReferenciaMotivo:          motivo,
			Accion:                    accion,
			Recurso:                   recurso,
			Finalidad:                 finalidad,
			Correlacion:               correlacion,
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud de autorización de consulta RRHH: %v", err)
	}
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		t,
		solicitud,
		autoridad.Resultado,
		motivo,
		ahora,
		"decision:rrhh:v3:prueba",
		true,
	)
	if err != nil {
		t.Fatalf("crear concesión de consulta RRHH: %v", err)
	}
	exportacion := exportacionMaterialConsumoConsultaRRHHPrueba(
		t,
		solicitud,
		decision,
		motivo,
		autoridad.Resultado,
		recurso,
		accion,
		ahora,
	)
	material, err := ports.NuevoMaterialAutorizacionConsultaRRHH(
		contexto,
		solicitud,
		decision,
		confirmacion,
		autoridad.Resultado,
		exportadorMaterialConsumoConsultaRRHHPrueba{exportacion: exportacion},
		ahora,
	)
	if err != nil {
		t.Fatalf("crear material probatorio de consulta RRHH: %v", err)
	}
	return material
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
