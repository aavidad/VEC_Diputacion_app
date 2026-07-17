package seguridad

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

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

func nuevaReferenciaFuenteAutoridad(
	ctx context.Context,
	lector io.Reader,
	prefijo string,
) (string, error) {
	if ctx == nil || lector == nil {
		return "", ports.ErrGeneracionReferenciaFuenteAutoridad
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(ports.ErrGeneracionReferenciaFuenteAutoridad, err)
	}
	aleatorio := make([]byte, 16)
	if _, err := io.ReadFull(lector, aleatorio); err != nil {
		return "", errors.Join(ports.ErrGeneracionReferenciaFuenteAutoridad, err)
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(ports.ErrGeneracionReferenciaFuenteAutoridad, err)
	}
	return prefijo + hex.EncodeToString(aleatorio), nil
}

var (
	_ ports.GeneradorReferenciaDecisionAutorizacion = GeneradorReferenciasCriptograficas{}
	_ ports.GeneradorReferenciasFuentesAutoridad    = GeneradorReferenciasCriptograficas{}
)
