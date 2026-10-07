package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// LectorFlujosVisualesRRHH conserva la presentación publicada de cada triple.
// Un expediente antiguo mantiene su rail original y uno nuevo obtiene el suyo
// solo si existe una definición de presentación para su flujo exacto.
type LectorFlujosVisualesRRHH struct {
	porFlujo map[domain.ReferenciaFlujo]*LectorFlujoVisualRRHH
}

func NuevoLectorFlujosVisualesRRHH(lectores ...*LectorFlujoVisualRRHH) (*LectorFlujosVisualesRRHH, error) {
	if len(lectores) == 0 {
		return nil, ErrManifestFlujoVisualRRHHInvalido
	}
	resultado := &LectorFlujosVisualesRRHH{porFlujo: make(map[domain.ReferenciaFlujo]*LectorFlujoVisualRRHH, len(lectores))}
	for _, lector := range lectores {
		if lector == nil || lector.origen.Validar() != nil || lector.base.Validar() != nil {
			return nil, ErrManifestFlujoVisualRRHHInvalido
		}
		if _, repetido := resultado.porFlujo[lector.origen]; repetido {
			return nil, ErrManifestFlujoVisualRRHHInvalido
		}
		resultado.porFlujo[lector.origen] = lector
	}
	return resultado, nil
}

func (l *LectorFlujosVisualesRRHH) Resolver(ctx context.Context, flujo domain.ReferenciaFlujo, faseActual domain.ClaveFase) (ports.PresentacionFlujoRRHH, error) {
	if l == nil || flujo.Validar() != nil {
		return ports.PresentacionFlujoRRHH{}, ErrManifestFlujoVisualRRHHInvalido
	}
	lector := l.porFlujo[flujo]
	if lector == nil {
		return ports.PresentacionFlujoRRHH{}, ErrManifestFlujoVisualRRHHInvalido
	}
	return lector.Resolver(ctx, flujo, faseActual)
}
