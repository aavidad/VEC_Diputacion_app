package postgres

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"vec-diputacion-granada/internal/vec/ports"
)

const bytesAleatoriosReferenciaContextoActorV2 = 24

// GeneradorOperacionContextoActorV2Criptografico genera operaciones opacas con
// 192 bits del CSPRNG del sistema. No incorpora identificadores del actor.
type GeneradorOperacionContextoActorV2Criptografico struct {
	aleatorio io.Reader
}

func NuevoGeneradorOperacionContextoActorV2Criptografico() GeneradorOperacionContextoActorV2Criptografico {
	return GeneradorOperacionContextoActorV2Criptografico{aleatorio: rand.Reader}
}

func nuevoGeneradorOperacionContextoActorV2(aleatorio io.Reader) GeneradorOperacionContextoActorV2Criptografico {
	return GeneradorOperacionContextoActorV2Criptografico{aleatorio: aleatorio}
}

func (g GeneradorOperacionContextoActorV2Criptografico) NuevaReferenciaOperacionContextoActorV2(
	ctx context.Context,
) (string, error) {
	return nuevaReferenciaContextoActorV2(ctx, g.aleatorio, "oca_",
		ports.ErrGeneradorOperacionContextoActorNoDisponible)
}

func nuevaReferenciaContextoActorV2(
	ctx context.Context,
	aleatorio io.Reader,
	prefijo string,
	errorPuerto error,
) (string, error) {
	if ctx == nil || valorNuloContextoActorPostgreSQL(aleatorio) {
		return "", errorPuerto
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(errorPuerto, err)
	}
	material := make([]byte, bytesAleatoriosReferenciaContextoActorV2)
	if _, err := io.ReadFull(aleatorio, material); err != nil {
		return "", errors.Join(errorPuerto, err)
	}
	if err := ctx.Err(); err != nil {
		return "", errors.Join(errorPuerto, err)
	}
	return prefijo + base64.RawURLEncoding.EncodeToString(material), nil
}

var _ ports.GeneradorOperacionContextoActorV2 = GeneradorOperacionContextoActorV2Criptografico{}
