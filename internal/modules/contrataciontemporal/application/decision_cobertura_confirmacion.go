package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	TiempoMaximoConfirmacionDecisionCobertura = 5 * time.Second
	redaccionSolicitudConfirmacionCobertura   = "[SOLICITUD-CONFIRMACION-DECISION-COBERTURA-REDACTADA]"
)

var (
	ErrServicioConfirmacionDecisionCoberturaInvalido = errors.New(
		"contratacion temporal: servicio de confirmacion de decision de cobertura invalido",
	)
	ErrSolicitudConfirmacionDecisionCoberturaInvalida = errors.New(
		"contratacion temporal: solicitud de confirmacion de decision de cobertura invalida",
	)
	ErrConfirmacionDecisionCoberturaDenegada = errors.New(
		"contratacion temporal: confirmacion de decision de cobertura denegada",
	)
	ErrConfirmacionDecisionCoberturaEnConflicto = errors.New(
		"contratacion temporal: confirmacion de decision de cobertura en conflicto",
	)
	ErrConfirmacionDecisionCoberturaOcupada = errors.New(
		"contratacion temporal: confirmacion de decision de cobertura ocupada",
	)
	ErrConfirmacionDecisionCoberturaNoConfiable = errors.New(
		"contratacion temporal: confirmacion de decision de cobertura no confiable",
	)
	ErrConfirmacionDecisionCoberturaNoDisponible = errors.New(
		"contratacion temporal: confirmacion de decision de cobertura no disponible",
	)
)

// SolicitudDecidirCobertura contiene solo intención y coordenadas de canal.
// Actor, acción, gobierno, políticas, autoridades VEC y evidencias nunca son
// declarables por HTTP, escritorio, CLI o MCP.
type SolicitudDecidirCobertura struct {
	AutenticacionRef   string
	SesionRef          string
	PerfilRef          string
	OrganizacionRef    string
	ExpedienteRef      string
	VersionEsperada    uint64
	ClaveIdempotencia  string
	IdentidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura
	ViaElegida         domain.ClaveCatalogo
	MotivoClave        domain.ClaveCatalogo
}

func (SolicitudDecidirCobertura) String() string {
	return redaccionSolicitudConfirmacionCobertura
}
func (s SolicitudDecidirCobertura) GoString() string { return s.String() }
func (s SolicitudDecidirCobertura) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudDecidirCobertura) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// SolicitudRectificarCobertura añade únicamente la identidad de la decisión
// sustituida. El actor anterior se obtiene del agregado durable.
type SolicitudRectificarCobertura struct {
	AutenticacionRef   string
	SesionRef          string
	PerfilRef          string
	OrganizacionRef    string
	ExpedienteRef      string
	VersionEsperada    uint64
	ClaveIdempotencia  string
	IdentidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura
	ViaElegida         domain.ClaveCatalogo
	MotivoClave        domain.ClaveCatalogo
	PredecesoraRef     string
	PredecesoraHuella  string
}

func (SolicitudRectificarCobertura) String() string {
	return redaccionSolicitudConfirmacionCobertura
}
func (s SolicitudRectificarCobertura) GoString() string { return s.String() }
func (s SolicitudRectificarCobertura) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudRectificarCobertura) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type resolutorClaveMotivoDecisionCobertura interface {
	ResolverClave(
		context.Context,
		domain.ClaveCatalogo,
		time.Time,
	) (cobertura.ResolucionMotivoDecisionCobertura, error)
}

// ServicioConfirmacionDecisionCobertura llega hasta la transacción nominal
// O4-04. No presupone SQL ni permite sustituir el futuro ejecutor TCB por un
// adaptador de memoria productivo.
type ServicioConfirmacionDecisionCobertura struct {
	contextos      ports.ResolutorContextoAutorizacionAltaV3
	motivos        resolutorClaveMotivoDecisionCobertura
	sellador       cobertura.SelladorOperacionDecisionCobertura
	idempotencia   cobertura.PreparadorOperacionDecisionCoberturaIdempotente
	analisis       cobertura.LectorExpedienteAnalisisDurableO3
	reloj          cobertura.RelojGobiernoOperacionCobertura
	gobierno       cobertura.ResolutorGobiernoOperacionCobertura
	coberturas     *PreparadorGlobalCobertura
	autorizaciones puertosvec.PreparadorRegistroCompuestoSolicitudLigadaV3
	transaccion    cobertura.TransaccionOperacionDecisionCobertura
	reconciliador  cobertura.ReconciliadorResultadoAmbiguoOperacionDecisionCobertura
}

func NuevoServicioConfirmacionDecisionCobertura(
	contextos ports.ResolutorContextoAutorizacionAltaV3,
	motivos resolutorClaveMotivoDecisionCobertura,
	sellador cobertura.SelladorOperacionDecisionCobertura,
	idempotencia cobertura.PreparadorOperacionDecisionCoberturaIdempotente,
	analisis cobertura.LectorExpedienteAnalisisDurableO3,
	reloj cobertura.RelojGobiernoOperacionCobertura,
	gobierno cobertura.ResolutorGobiernoOperacionCobertura,
	coberturas *PreparadorGlobalCobertura,
	autorizaciones puertosvec.PreparadorRegistroCompuestoSolicitudLigadaV3,
	transaccion cobertura.TransaccionOperacionDecisionCobertura,
	reconciliador cobertura.ReconciliadorResultadoAmbiguoOperacionDecisionCobertura,
) (*ServicioConfirmacionDecisionCobertura, error) {
	if dependenciaNula(contextos) || dependenciaNula(motivos) ||
		dependenciaNula(sellador) || dependenciaNula(idempotencia) ||
		dependenciaNula(analisis) || dependenciaNula(reloj) ||
		dependenciaNula(gobierno) || dependenciaNula(coberturas) ||
		dependenciaNula(autorizaciones) || dependenciaNula(transaccion) ||
		dependenciaNula(reconciliador) {
		return nil, ErrServicioConfirmacionDecisionCoberturaInvalido
	}
	return &ServicioConfirmacionDecisionCobertura{
		contextos: contextos, motivos: motivos, sellador: sellador,
		idempotencia: idempotencia, analisis: analisis, reloj: reloj,
		gobierno: gobierno, coberturas: coberturas,
		autorizaciones: autorizaciones, transaccion: transaccion,
		reconciliador: reconciliador,
	}, nil
}

func (s *ServicioConfirmacionDecisionCobertura) Decidir(
	ctx context.Context,
	solicitud SolicitudDecidirCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	return s.ejecutar(ctx, datosSolicitudConfirmacionDecisionCobertura{
		tipo:             domain.DecisionCoberturaInicial,
		autenticacionRef: solicitud.AutenticacionRef,
		sesionRef:        solicitud.SesionRef, perfilRef: solicitud.PerfilRef,
		organizacionRef:    solicitud.OrganizacionRef,
		expedienteRef:      solicitud.ExpedienteRef,
		versionEsperada:    solicitud.VersionEsperada,
		claveIdempotencia:  solicitud.ClaveIdempotencia,
		identidadSemantica: solicitud.IdentidadSemantica,
		viaElegida:         solicitud.ViaElegida, motivoClave: solicitud.MotivoClave,
	})
}

func (s *ServicioConfirmacionDecisionCobertura) Rectificar(
	ctx context.Context,
	solicitud SolicitudRectificarCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	return s.ejecutar(ctx, datosSolicitudConfirmacionDecisionCobertura{
		tipo:             domain.DecisionCoberturaRectificacion,
		autenticacionRef: solicitud.AutenticacionRef,
		sesionRef:        solicitud.SesionRef, perfilRef: solicitud.PerfilRef,
		organizacionRef:    solicitud.OrganizacionRef,
		expedienteRef:      solicitud.ExpedienteRef,
		versionEsperada:    solicitud.VersionEsperada,
		claveIdempotencia:  solicitud.ClaveIdempotencia,
		identidadSemantica: solicitud.IdentidadSemantica,
		viaElegida:         solicitud.ViaElegida, motivoClave: solicitud.MotivoClave,
		predecesoraRef:    solicitud.PredecesoraRef,
		predecesoraHuella: solicitud.PredecesoraHuella,
	})
}

type datosSolicitudConfirmacionDecisionCobertura struct {
	tipo               domain.TipoDecisionCoberturaGobernada
	autenticacionRef   string
	sesionRef          string
	perfilRef          string
	organizacionRef    string
	expedienteRef      string
	versionEsperada    uint64
	claveIdempotencia  string
	identidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura
	viaElegida         domain.ClaveCatalogo
	motivoClave        domain.ClaveCatalogo
	predecesoraRef     string
	predecesoraHuella  string
}

func (s *ServicioConfirmacionDecisionCobertura) ejecutar(
	ctx context.Context,
	solicitud datosSolicitudConfirmacionDecisionCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	if s == nil || ctx == nil || !s.dependenciasValidas() {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrServicioConfirmacionDecisionCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	if solicitud.validar() != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrSolicitudConfirmacionDecisionCoberturaInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoConfirmacionDecisionCobertura,
	)
	defer cancelar()
	return s.ejecutarValidada(operacion, solicitud)
}
