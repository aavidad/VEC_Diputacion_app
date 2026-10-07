package internactproveedores

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestEmisorRenovableConCapturaFallaCerradoSinLector(t *testing.T) {
	var emisor *emisorMaterialRenovable
	_, _, material, captura, err := emisor.EmitirMaterialAutorizacionAtestadaV3ConCaptura(
		context.Background(), domain.SolicitudAutorizacionLigadaV3{}, domain.ResultadoContextoActorRegistradoV2{},
	)
	if !errors.Is(err, ErrProveedoresCTNoDisponibles) || material != nil || captura != nil {
		t.Fatalf("emision renovable sin lector no cerro: %v", err)
	}
}
