package cobertura

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const redaccionEvidenciaConsulta = "[EVIDENCIA-CONSULTA-COBERTURA-REDACTADA]"

// EvidenciaConsultaCobertura conserva una orden de consumo pendiente sin
// exponer su atestación ni producir un efecto durable por sí misma.
type EvidenciaConsultaCobertura struct {
	orden        ports.OrdenConsumoCobertura
	resumen      ports.ResumenOrdenConsumoCobertura
	comprobadaEn time.Time
}

func NuevaEvidenciaConsultaCobertura(
	orden ports.OrdenConsumoCobertura,
	comprobadaEn time.Time,
) (EvidenciaConsultaCobertura, error) {
	resumen, err := orden.ResumenPendienteEn(comprobadaEn)
	if err != nil {
		return EvidenciaConsultaCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return EvidenciaConsultaCobertura{
		orden: orden, resumen: resumen, comprobadaEn: comprobadaEn,
	}, nil
}

func (e EvidenciaConsultaCobertura) ValidarEn(comprobadaEn time.Time) error {
	if !domain.InstanteUTCCanonico(e.comprobadaEn) ||
		!domain.InstanteUTCCanonico(comprobadaEn) ||
		comprobadaEn.Before(e.comprobadaEn) {
		return ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	actual, err := e.orden.ResumenPendienteEn(comprobadaEn)
	if err != nil || actual != e.resumen {
		return ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (e EvidenciaConsultaCobertura) Comprobacion() (
	domain.ComprobacionCobertura,
	error,
) {
	if e.resumen.Comprobacion.Validar() != nil ||
		e.resumen.Comprobacion.Detalle != "" {
		return domain.ComprobacionCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return e.resumen.Comprobacion, nil
}

func (e EvidenciaConsultaCobertura) Resumen() (
	ports.ResumenOrdenConsumoCobertura,
	error,
) {
	if e.resumen.Comprobacion.Validar() != nil ||
		e.resumen.Comprobacion.Detalle != "" {
		return ports.ResumenOrdenConsumoCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return e.resumen, nil
}

func (e EvidenciaConsultaCobertura) OrdenPendienteEn(
	comprobadaEn time.Time,
) (ports.OrdenConsumoCobertura, error) {
	if e.ValidarEn(comprobadaEn) != nil {
		return ports.OrdenConsumoCobertura{},
			ports.ErrResultadoFuenteCoberturaNoConfiable
	}
	return e.orden, nil
}

func (e EvidenciaConsultaCobertura) sueloTemporal() time.Time {
	return e.comprobadaEn
}

func (EvidenciaConsultaCobertura) String() string {
	return redaccionEvidenciaConsulta
}

func (e EvidenciaConsultaCobertura) GoString() string { return e.String() }

func (e EvidenciaConsultaCobertura) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}

func (e EvidenciaConsultaCobertura) LogValue() slog.Value {
	return slog.StringValue(e.String())
}

func (e EvidenciaConsultaCobertura) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e EvidenciaConsultaCobertura) MarshalJSON() ([]byte, error) {
	return []byte(`"` + e.String() + `"`), nil
}
