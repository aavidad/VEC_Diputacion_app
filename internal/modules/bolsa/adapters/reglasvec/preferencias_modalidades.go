package reglasvec

import (
	"context"
	"errors"
	"slices"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// CatalogoPreferenciasModalidades traduce una lectura del catálogo común. La
// composición elige su fuente y aplica las guardas del perfil de ejecución.
type CatalogoPreferenciasModalidades struct {
	resolutor *reglas.Resolutor
}

var _ ports.CatalogoPreferenciasModalidades = (*CatalogoPreferenciasModalidades)(nil)

func NuevoCatalogoPreferenciasModalidades(resolutor *reglas.Resolutor) *CatalogoPreferenciasModalidades {
	return &CatalogoPreferenciasModalidades{resolutor: resolutor}
}

func (c *CatalogoPreferenciasModalidades) ConsultarModalidadesDisponibles(ctx context.Context) ([]ports.ModalidadPreferencia, error) {
	if ctx == nil || c == nil || c.resolutor == nil {
		return nil, ports.ErrPreferenciasNoDisponibles
	}
	lectura, err := c.resolutor.LeerReglas(ctx)
	if err != nil {
		return nil, errors.Join(ports.ErrPreferenciasNoDisponibles, err)
	}
	opciones := make([]ports.ModalidadPreferencia, 0)
	for _, regla := range lectura.Reglas() {
		clave, modalidad := strings.CutPrefix(regla.Clave, reglas.CTPrefijoModalidad)
		if !modalidad {
			continue
		}
		fuente := regla.ReferenciaEntrada
		if clave == "" || regla.AjusteNoAplicable || fuente.Validar() != nil ||
			fuente.CatalogoID != reglas.CatalogoContratacionTemporal ||
			fuente.EntradaClave != regla.Clave || fuente.CatalogoHuellaSHA256 != regla.HuellaCatalogo {
			return nil, ports.ErrPreferenciasNoDisponibles
		}
		opciones = append(opciones, ports.ModalidadPreferencia{Clave: clave, Fuente: fuente})
	}
	if len(opciones) == 0 || len(opciones) > 256 {
		return nil, ports.ErrPreferenciasNoDisponibles
	}
	slices.SortFunc(opciones, func(a, b ports.ModalidadPreferencia) int { return strings.Compare(a.Clave, b.Clave) })
	for i := 1; i < len(opciones); i++ {
		if opciones[i-1].Clave == opciones[i].Clave {
			return nil, ports.ErrPreferenciasNoDisponibles
		}
	}
	return opciones, nil
}
