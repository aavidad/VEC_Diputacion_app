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
)

const (
	TiempoMaximoConsultaResultadoCobertura = 5 * time.Second
	redaccionSolicitudConsultaCobertura    = "" +
		"[SOLICITUD-CONSULTA-COBERTURA-REDACTADA]"
	redaccionResultadoConsultaCobertura = "" +
		"[RESULTADO-CONSULTA-COBERTURA-REDACTADO]"
)

var (
	ErrServicioConsultaResultadoCoberturaInvalido = errors.New(
		"contratacion temporal: servicio de consulta de resultado de cobertura invalido",
	)
	ErrSolicitudConsultaResultadoCoberturaInvalida = errors.New(
		"contratacion temporal: solicitud de consulta de resultado de cobertura invalida",
	)
	ErrConsultaResultadoCoberturaDenegada = errors.New(
		"contratacion temporal: consulta de resultado de cobertura denegada",
	)
	ErrConsultaResultadoCoberturaNoDisponible = errors.New(
		"contratacion temporal: consulta de resultado de cobertura no disponible",
	)
	ErrConsultaResultadoCoberturaNoConfiable = errors.New(
		"contratacion temporal: consulta de resultado de cobertura no confiable",
	)
	ErrConsultaResultadoCoberturaConflicto = errors.New(
		"contratacion temporal: historia de resultado de cobertura divergente",
	)
)

// SolicitudConsultaResultadoCobertura contiene toda la entrada funcional del
// cliente. Organización, actor y perfil proceden de la frontera confiable.
type SolicitudConsultaResultadoCobertura struct {
	ClaveIdempotencia string
	ExpedienteRef     string
}

func (SolicitudConsultaResultadoCobertura) String() string {
	return redaccionSolicitudConsultaCobertura
}
func (s SolicitudConsultaResultadoCobertura) GoString() string {
	return s.String()
}
func (s SolicitudConsultaResultadoCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudConsultaResultadoCobertura) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func (s SolicitudConsultaResultadoCobertura) validar() error {
	if !ports.ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) {
		return ErrSolicitudConsultaResultadoCoberturaInvalida
	}
	return nil
}

type EstadoConsultaResultadoCobertura string

const (
	ResultadoCoberturaConfirmado   EstadoConsultaResultadoCobertura = "confirmado"
	ResultadoCoberturaNoObservable EstadoConsultaResultadoCobertura = "no_observable"
)

// DatosConsultaResultadoCoberturaParaAdaptador es la unión DTO mínima.
type DatosConsultaResultadoCoberturaParaAdaptador struct {
	Estado EstadoConsultaResultadoCobertura
	Recibo *DatosReciboDecisionCoberturaParaAdaptador
}

// ResultadoConsultaResultadoCobertura es sellado: adaptadores solo observan.
type ResultadoConsultaResultadoCobertura struct {
	confirmado   *ResultadoDecisionCoberturaParaAdaptador
	noObservable bool
	sello        string
}

func nuevoResultadoConsultaCoberturaConfirmado(
	recibo cobertura.ReciboOperacionDecisionCobertura,
) (ResultadoConsultaResultadoCobertura, error) {
	proyeccion, err := nuevoResultadoDecisionCoberturaParaAdaptador(recibo)
	if err != nil {
		return ResultadoConsultaResultadoCobertura{},
			ErrConsultaResultadoCoberturaNoConfiable
	}
	datos, valida := proyeccion.DatosParaAdaptador()
	if !valida {
		return ResultadoConsultaResultadoCobertura{},
			ErrConsultaResultadoCoberturaNoConfiable
	}
	return ResultadoConsultaResultadoCobertura{
		confirmado: &proyeccion,
		sello:      datos.ReciboRef,
	}, nil
}

func nuevoResultadoConsultaCoberturaNoObservable() ResultadoConsultaResultadoCobertura {
	return ResultadoConsultaResultadoCobertura{
		noObservable: true,
		sello:        string(ResultadoCoberturaNoObservable),
	}
}

func (r ResultadoConsultaResultadoCobertura) DatosParaAdaptador() (
	DatosConsultaResultadoCoberturaParaAdaptador,
	bool,
) {
	if r.noObservable {
		if r.confirmado != nil ||
			r.sello != string(ResultadoCoberturaNoObservable) {
			return DatosConsultaResultadoCoberturaParaAdaptador{}, false
		}
		return DatosConsultaResultadoCoberturaParaAdaptador{
			Estado: ResultadoCoberturaNoObservable,
		}, true
	}
	if r.confirmado == nil || r.sello == "" {
		return DatosConsultaResultadoCoberturaParaAdaptador{}, false
	}
	recibo, valida := r.confirmado.DatosParaAdaptador()
	if !valida || recibo.ReciboRef != r.sello {
		return DatosConsultaResultadoCoberturaParaAdaptador{}, false
	}
	return DatosConsultaResultadoCoberturaParaAdaptador{
		Estado: ResultadoCoberturaConfirmado,
		Recibo: &recibo,
	}, true
}

func (ResultadoConsultaResultadoCobertura) String() string {
	return redaccionResultadoConsultaCobertura
}
func (r ResultadoConsultaResultadoCobertura) GoString() string {
	return r.String()
}
func (r ResultadoConsultaResultadoCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ResultadoConsultaResultadoCobertura) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

type ServicioConsultaResultadoCobertura struct {
	contextos ports.ResolutorContextoRecuperacionResultadoCobertura
	accesos   ports.AutorizadorLecturaResultadoCobertura
	sellador  cobertura.SelladorAmbitoOperacionDecisionCobertura
	reloj     cobertura.RelojGobiernoOperacionCobertura
	lector    cobertura.LectorResultadoHistoricoOperacionDecisionCobertura
}

func NuevoServicioConsultaResultadoCobertura(
	contextos ports.ResolutorContextoRecuperacionResultadoCobertura,
	accesos ports.AutorizadorLecturaResultadoCobertura,
	sellador cobertura.SelladorAmbitoOperacionDecisionCobertura,
	reloj cobertura.RelojGobiernoOperacionCobertura,
	lector cobertura.LectorResultadoHistoricoOperacionDecisionCobertura,
) (*ServicioConsultaResultadoCobertura, error) {
	if dependenciaNula(contextos) || dependenciaNula(accesos) ||
		dependenciaNula(sellador) || dependenciaNula(reloj) ||
		dependenciaNula(lector) {
		return nil, ErrServicioConsultaResultadoCoberturaInvalido
	}
	return &ServicioConsultaResultadoCobertura{
		contextos: contextos,
		accesos:   accesos,
		sellador:  sellador,
		reloj:     reloj,
		lector:    lector,
	}, nil
}

func (s *ServicioConsultaResultadoCobertura) Consultar(
	ctx context.Context,
	solicitud SolicitudConsultaResultadoCobertura,
) (ResultadoConsultaResultadoCobertura, error) {
	if s == nil || ctx == nil || !s.dependenciasValidas() {
		return ResultadoConsultaResultadoCobertura{},
			ErrServicioConsultaResultadoCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoConsultaResultadoCobertura{}, err
	}
	if solicitud.validar() != nil {
		return ResultadoConsultaResultadoCobertura{},
			ErrSolicitudConsultaResultadoCoberturaInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoConsultaResultadoCobertura,
	)
	defer cancelar()
	return s.consultarValidada(operacion, solicitud)
}
