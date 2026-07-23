package seguridad

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const bytesAleatoriosReferenciaAlta = 32

var ErrGeneracionReferenciaAlta = errors.New(
	"contratacion temporal: generacion de referencia no disponible",
)

type GeneradorReferenciasAltaCriptografico struct {
	lector io.Reader
	ahora  func() time.Time
}

func NuevoGeneradorReferenciasAltaCriptografico() *GeneradorReferenciasAltaCriptografico {
	return &GeneradorReferenciasAltaCriptografico{
		lector: rand.Reader,
		ahora:  func() time.Time { return time.Now().UTC() },
	}
}

func (g *GeneradorReferenciasAltaCriptografico) GenerarReferenciasAlta(
	ctx context.Context,
) (ports.ReferenciasAlta, error) {
	if !generadorValido(g) || ctx == nil {
		return ports.ReferenciasAlta{}, ErrGeneracionReferenciaAlta
	}
	if err := ctx.Err(); err != nil {
		return ports.ReferenciasAlta{}, err
	}
	expediente, err := g.generar(ctx, "expediente:ct:")
	if err != nil {
		return ports.ReferenciasAlta{}, err
	}
	recibo, err := g.generar(ctx, "recibo:ct-alta:")
	if err != nil {
		return ports.ReferenciasAlta{}, err
	}
	visible, err := g.numeroVisible(ctx)
	if err != nil {
		return ports.ReferenciasAlta{}, err
	}
	referencias := ports.ReferenciasAlta{
		ExpedienteRef: expediente,
		NumeroVisible: visible,
		ReciboRef:     recibo,
	}
	if referencias.Validar() != nil {
		return ports.ReferenciasAlta{}, ErrGeneracionReferenciaAlta
	}
	return referencias, nil
}

func (g *GeneradorReferenciasAltaCriptografico) NuevaReferenciaReservaAlta(
	ctx context.Context,
) (string, error) {
	if !generadorValido(g) || ctx == nil {
		return "", ErrGeneracionReferenciaAlta
	}
	return g.generar(ctx, "reserva:ct-alta:")
}

func (g *GeneradorReferenciasAltaCriptografico) numeroVisible(
	ctx context.Context,
) (string, error) {
	instante := g.ahora().UTC()
	if instante.Year() < 1 || instante.Year() > 9999 {
		return "", ErrGeneracionReferenciaAlta
	}
	aleatorio, err := g.entropia(ctx)
	if err != nil {
		return "", err
	}
	defer borrar(aleatorio)
	return fmt.Sprintf(
		"%04d/CT-%s",
		instante.Year(),
		hex.EncodeToString(aleatorio[:16]),
	), nil
}

func (g *GeneradorReferenciasAltaCriptografico) generar(
	ctx context.Context,
	prefijo string,
) (string, error) {
	aleatorio, err := g.entropia(ctx)
	if err != nil {
		return "", err
	}
	defer borrar(aleatorio)
	return prefijo + hex.EncodeToString(aleatorio), nil
}

func (g *GeneradorReferenciasAltaCriptografico) entropia(
	ctx context.Context,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	aleatorio := make([]byte, bytesAleatoriosReferenciaAlta)
	if _, err := io.ReadFull(g.lector, aleatorio); err != nil {
		borrar(aleatorio)
		return nil, ErrGeneracionReferenciaAlta
	}
	if err := ctx.Err(); err != nil {
		borrar(aleatorio)
		return nil, err
	}
	return aleatorio, nil
}

func generadorValido(g *GeneradorReferenciasAltaCriptografico) bool {
	return g != nil && g.lector != nil && g.ahora != nil
}

var _ ports.GeneradorReferenciasAlta = (*GeneradorReferenciasAltaCriptografico)(nil)
