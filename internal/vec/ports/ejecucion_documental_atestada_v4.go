package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrEjecucionDocumentalAtestadaV4NoDisponible = errors.New(
	"vec: ejecucion documental atestada v4 no disponible",
)

const estadoOrdenDocumentalAtestadaV4Pendiente = "pendiente_generacion"

// ConectorEjecucionDocumentalAtestadaV4 es el puerto de salida del nucleo.
// PostgreSQL, Oracle u otro conector homologado implementan el consumo
// atomico, pero el caso de uso no conoce su motor, transporte ni credenciales.
//
// La solicitud vinculada y el sobre son valores opacos y de valor cero
// invalido. El conector debe volver a verificar la autorizacion y confirmar el
// efecto, la auditoria y el outbox en una unica frontera transaccional.
type ConectorEjecucionDocumentalAtestadaV4 interface {
	EjecutarDocumentalAtestadoV4(
		context.Context,
		SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
		domain.CabeceraAtestacionAutorizacionV1,
		SobreCriptograficoDocumentalCrudoV4,
	) (ResultadoConectorEjecucionDocumentalAtestadaV4, error)
}

// ResultadoConectorEjecucionDocumentalAtestadaV4 es una confirmacion opaca y
// no una capacidad reutilizable. Solo conserva referencias operativas; nunca
// transporta COSE, payload, identidad personal, secreto ni credencial.
type ResultadoConectorEjecucionDocumentalAtestadaV4 struct {
	ordenRef        string
	estado          string
	auditoriaRef    string
	eventoOutboxRef string
	registradaEn    time.Time
}

func NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
	ordenRef, estado, auditoriaRef, eventoOutboxRef string,
	registradaEn time.Time,
) (ResultadoConectorEjecucionDocumentalAtestadaV4, error) {
	resultado := ResultadoConectorEjecucionDocumentalAtestadaV4{
		ordenRef: ordenRef, estado: estado, auditoriaRef: auditoriaRef,
		eventoOutboxRef: eventoOutboxRef, registradaEn: registradaEn,
	}
	if resultado.Validar() != nil {
		return ResultadoConectorEjecucionDocumentalAtestadaV4{},
			ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return resultado, nil
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) Validar() error {
	if !referenciaEjecucionDocumentalV3Valida(r.ordenRef) ||
		r.estado != estadoOrdenDocumentalAtestadaV4Pendiente ||
		!referenciaResultadoEjecucionDocumentalAtestadaV4Valida(
			r.auditoriaRef, "auditoria:documental:v4:",
		) || !referenciaResultadoEjecucionDocumentalAtestadaV4Valida(
		r.eventoOutboxRef, "evento:documental:v4:",
	) || r.auditoriaRef == r.eventoOutboxRef ||
		!instanteEjecucionDocumentalV3Valido(r.registradaEn) {
		return ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return nil
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) OrdenRef() (string, error) {
	if r.Validar() != nil {
		return "", ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return r.ordenRef, nil
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) Estado() (string, error) {
	if r.Validar() != nil {
		return "", ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return r.estado, nil
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) AuditoriaRef() (string, error) {
	if r.Validar() != nil {
		return "", ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return r.auditoriaRef, nil
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) EventoOutboxRef() (string, error) {
	if r.Validar() != nil {
		return "", ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return r.eventoOutboxRef, nil
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) RegistradaEn() (time.Time, error) {
	if r.Validar() != nil {
		return time.Time{}, ErrEjecucionDocumentalAtestadaV4NoDisponible
	}
	return r.registradaEn, nil
}

func (ResultadoConectorEjecucionDocumentalAtestadaV4) String() string {
	return "[RESULTADO-CONECTOR-EJECUCION-DOCUMENTAL-ATESTADA-V4-REDACTADO]"
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) GoString() string {
	return r.String()
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (r ResultadoConectorEjecucionDocumentalAtestadaV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

func (ResultadoConectorEjecucionDocumentalAtestadaV4) MarshalJSON() ([]byte, error) {
	return nil, ErrEjecucionDocumentalAtestadaV4NoDisponible
}

func (*ResultadoConectorEjecucionDocumentalAtestadaV4) UnmarshalJSON([]byte) error {
	return ErrEjecucionDocumentalAtestadaV4NoDisponible
}

func (ResultadoConectorEjecucionDocumentalAtestadaV4) MarshalText() ([]byte, error) {
	return nil, ErrEjecucionDocumentalAtestadaV4NoDisponible
}

func (*ResultadoConectorEjecucionDocumentalAtestadaV4) UnmarshalText([]byte) error {
	return ErrEjecucionDocumentalAtestadaV4NoDisponible
}

func (ResultadoConectorEjecucionDocumentalAtestadaV4) MarshalBinary() ([]byte, error) {
	return nil, ErrEjecucionDocumentalAtestadaV4NoDisponible
}

func (*ResultadoConectorEjecucionDocumentalAtestadaV4) UnmarshalBinary([]byte) error {
	return ErrEjecucionDocumentalAtestadaV4NoDisponible
}

func referenciaResultadoEjecucionDocumentalAtestadaV4Valida(
	valor, prefijo string,
) bool {
	return strings.HasPrefix(valor, prefijo) &&
		esSHA256Hexadecimal(strings.TrimPrefix(valor, prefijo)) &&
		referenciaEjecucionDocumentalV3Valida(valor)
}
