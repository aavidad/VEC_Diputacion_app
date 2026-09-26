package ports

import (
	"context"
	"fmt"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// ErrResultadoFiscalizacionNoAdmitido: el catálogo vigente no admite el
// resultado pedido. Es un contenido no válido, no un fallo del servicio.
var ErrResultadoFiscalizacionNoAdmitido = fmt.Errorf(
	"contratacion temporal: resultado de fiscalizacion no admitido: %w", domain.ErrDatoInvalido,
)

// FuenteResultadosFiscalizacion entrega la política vigente de resultados de
// fiscalización (qué resultados se admiten y su efecto). Un error nunca
// equivale a admitir el resultado.
type FuenteResultadosFiscalizacion interface {
	ResultadosFiscalizacion(context.Context) (domain.PoliticaResultadosFiscalizacion, error)
}
