package seguridad

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// GeneradorReferenciasCriptograficas crea identificadores opacos sin incluir
// DNI, nombre, correo ni ninguna clave de negocio.
type GeneradorReferenciasCriptograficas struct{}

func (GeneradorReferenciasCriptograficas) NuevaReferenciaDecisionAutorizacion() (string, error) {
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		return "", fmt.Errorf("crear referencia de decision: %w", err)
	}
	return "decision:" + hex.EncodeToString(aleatorio), nil
}

func (GeneradorReferenciasCriptograficas) NuevaReferenciaSolicitud(
	ctx context.Context,
) (ports.ReferenciaSolicitudFuenteAutoridad, error) {
	valor, err := nuevaReferenciaFuenteAutoridad(
		ctx,
		rand.Reader,
		ports.PrefijoReferenciaSolicitudFuenteAutoridad,
	)
	if err != nil {
		return ports.ReferenciaSolicitudFuenteAutoridad{}, err
	}
	referencia, err := ports.NuevaReferenciaSolicitudFuenteAutoridad(valor)
	if err != nil {
		return ports.ReferenciaSolicitudFuenteAutoridad{}, errors.Join(
			ports.ErrGeneracionReferenciaFuenteAutoridad,
			err,
		)
	}
	return referencia, nil
}

func (GeneradorReferenciasCriptograficas) NuevaReferenciaOperacion(
	ctx context.Context,
) (ports.ReferenciaOperacionFuenteAutoridad, error) {
	valor, err := nuevaReferenciaFuenteAutoridad(
		ctx,
		rand.Reader,
		ports.PrefijoReferenciaOperacionFuenteAutoridad,
	)
	if err != nil {
		return ports.ReferenciaOperacionFuenteAutoridad{}, err
	}
	referencia, err := ports.NuevaReferenciaOperacionFuenteAutoridad(valor)
	if err != nil {
		return ports.ReferenciaOperacionFuenteAutoridad{}, errors.Join(
			ports.ErrGeneracionReferenciaFuenteAutoridad,
			err,
		)
	}
	return referencia, nil
}

func (GeneradorReferenciasCriptograficas) NuevaReferenciaCorrelacionAutorizacionV2(
	ctx context.Context,
) (string, error) {
	valor, err := nuevaReferenciaOpacaHex128(
		ctx,
		rand.Reader,
		"correlacion_",
		ports.ErrGeneracionReferenciaAutorizacionV2,
	)
	if err != nil || !domain.ReferenciaCorrelacionAutorizacionV2Valida(valor) {
		return "", errors.Join(ports.ErrGeneracionReferenciaAutorizacionV2, err)
	}
	return valor, nil
}

func (GeneradorReferenciasCriptograficas) NuevaClaveMotivoAutorizacionV2(
	ctx context.Context,
) (string, error) {
	valor, err := nuevaReferenciaOpacaHex128(
		ctx,
		rand.Reader,
		"motivo_",
		ports.ErrGeneracionReferenciaAutorizacionV2,
	)
	if err != nil || !domain.ClaveMotivoAutorizacionV2Valida(valor) {
		return "", errors.Join(ports.ErrGeneracionReferenciaAutorizacionV2, err)
	}
	return valor, nil
}

func nuevaReferenciaFuenteAutoridad(
	ctx context.Context,
	lector io.Reader,
	prefijo string,
) (string, error) {
	return nuevaReferenciaOpacaHex128(
		ctx,
		lector,
		prefijo,
		ports.ErrGeneracionReferenciaFuenteAutoridad,
	)
}

func nuevaReferenciaOpacaHex128(
	ctx context.Context,
	lector io.Reader,
	prefijo string,
	errorGeneracion error,
) (string, error) {
	if ctx == nil || lector == nil {
		return "", errorGeneracion
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(errorGeneracion, err)
	}
	aleatorio := make([]byte, 16)
	if _, err := io.ReadFull(lector, aleatorio); err != nil {
		return "", errors.Join(errorGeneracion, err)
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(errorGeneracion, err)
	}
	return prefijo + hex.EncodeToString(aleatorio), nil
}

var (
	_ ports.GeneradorReferenciaDecisionAutorizacion = GeneradorReferenciasCriptograficas{}
	_ ports.GeneradorReferenciasFuentesAutoridad    = GeneradorReferenciasCriptograficas{}
	_ ports.GeneradorReferenciasAutorizacionV2      = GeneradorReferenciasCriptograficas{}
)
