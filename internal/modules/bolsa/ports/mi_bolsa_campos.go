package ports

import (
	"context"
	"errors"
)

// Datos de «Mi bolsa» que un catálogo puede mostrar u ocultar. La bolsa
// identifica la participación y siempre es visible.
const (
	CampoPortalBolsa             = "bolsa"
	CampoPortalPosicion          = "posicion"
	CampoPortalEstado            = "estado"
	CampoPortalUltimoLlamamiento = "ultimo_llamamiento"
	CampoPortalContratos         = "contratos"
	CampoPortalFechaDisponible   = "fecha_disponible"
)

var (
	ErrCamposPortalMiBolsaInvalidos     = errors.New("bolsa: campos del portal de mi bolsa no validos")
	ErrCamposPortalMiBolsaNoDisponibles = errors.New("bolsa: campos del portal de mi bolsa no disponibles")
)

// CamposPortalMiBolsaTodos es la conducta sin catálogo: todos los datos.
func CamposPortalMiBolsaTodos() []string {
	return []string{
		CampoPortalBolsa, CampoPortalPosicion, CampoPortalEstado,
		CampoPortalUltimoLlamamiento, CampoPortalContratos, CampoPortalFechaDisponible,
	}
}

// CamposPortalMiBolsa devuelve la lista vigente de datos visibles. Un error
// no se interpreta como «mostrar todo»: la consulta se rechaza.
type CamposPortalMiBolsa interface {
	CamposVisiblesMiBolsa(context.Context) ([]string, error)
}

// ValidarCamposPortalMiBolsa exige nombres conocidos, sin repetir, y que
// incluyan la bolsa. Devuelve una copia en el orden recibido.
func ValidarCamposPortalMiBolsa(campos []string) ([]string, error) {
	conocidos := make(map[string]bool, 6)
	for _, campo := range CamposPortalMiBolsaTodos() {
		conocidos[campo] = true
	}
	vistos := make(map[string]bool, len(campos))
	copia := make([]string, 0, len(campos))
	for _, campo := range campos {
		if !conocidos[campo] || vistos[campo] {
			return nil, ErrCamposPortalMiBolsaInvalidos
		}
		vistos[campo] = true
		copia = append(copia, campo)
	}
	if !vistos[CampoPortalBolsa] {
		return nil, ErrCamposPortalMiBolsaInvalidos
	}
	return copia, nil
}
