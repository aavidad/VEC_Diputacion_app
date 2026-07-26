package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const redaccionGeneradorCorrelacionVECReservada = "[GENERADOR-CORRELACION-VEC-RESERVADA-REDACTADO]"

var errCorrelacionVECReservadaNoDisponible = errors.New(
	"contratacion temporal: correlacion reservada VEC no disponible",
)

// generadorCorrelacionVECReservada acuña como capacidad nominal la
// correlación preasignada por la reserva durable. Solo puede entregarla una
// vez y nunca deriva su valor de identidad, expediente ni datos de cliente.
type generadorCorrelacionVECReservada struct {
	mu         sync.Mutex
	referencia string
	consumida  bool
}

func nuevoGeneradorCorrelacionVECReservada(
	referencia string,
) (*generadorCorrelacionVECReservada, error) {
	if !dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(referencia) {
		return nil, errCorrelacionVECReservadaNoDisponible
	}
	return &generadorCorrelacionVECReservada{referencia: referencia}, nil
}

func (g *generadorCorrelacionVECReservada) NuevaReferenciaCorrelacionAutorizacionV2(
	ctx context.Context,
) (string, error) {
	if g == nil || ctx == nil {
		return "", errCorrelacionVECReservadaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(errCorrelacionVECReservadaNoDisponible, err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", errors.Join(errCorrelacionVECReservadaNoDisponible, err)
	}
	if g.consumida ||
		!dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(g.referencia) {
		return "", errCorrelacionVECReservadaNoDisponible
	}
	g.consumida = true
	return g.referencia, nil
}

func (*generadorCorrelacionVECReservada) String() string {
	return redaccionGeneradorCorrelacionVECReservada
}

func (g *generadorCorrelacionVECReservada) GoString() string {
	return g.String()
}

func (g *generadorCorrelacionVECReservada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, g.String())
}

func (g *generadorCorrelacionVECReservada) LogValue() slog.Value {
	return slog.StringValue(g.String())
}
