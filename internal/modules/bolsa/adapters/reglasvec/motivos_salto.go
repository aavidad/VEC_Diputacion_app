package reglasvec

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

var _ ports.CatalogoMotivosSaltoOrden = (*CatalogoMotivosSalto)(nil)

// claveMotivoSalto es el mismo formato que exige la reserva B147 en SQL.
var claveMotivoSalto = regexp.MustCompile(`^b01\.salto\.[a-z0-9_]{1,64}$`)

// CatalogoMotivosSalto lee en cada consulta las entradas «b01.salto.*» del
// catálogo de reglas de Bolsa: cada una es un motivo que permite llamar
// saltando el orden estricto. Hoy el catálogo no tiene ninguna.
type CatalogoMotivosSalto struct {
	resolutor *reglas.Resolutor
}

// NuevoCatalogoMotivosSalto devuelve nil sin resolutor: el servicio lo trata
// como un catálogo sin motivos.
func NuevoCatalogoMotivosSalto(resolutor *reglas.Resolutor) *CatalogoMotivosSalto {
	if resolutor == nil {
		return nil
	}
	return &CatalogoMotivosSalto{resolutor: resolutor}
}

func (c *CatalogoMotivosSalto) MotivosSaltoOrden(ctx context.Context) ([]ports.MotivoSaltoOrden, error) {
	if c == nil || ctx == nil {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	todas, err := c.resolutor.Reglas(ctx)
	if err != nil {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	motivos := []ports.MotivoSaltoOrden{}
	for _, regla := range todas {
		if strings.HasPrefix(regla.Clave, reglas.BolsaPrefijoMotivosSalto) && claveMotivoSalto.MatchString(regla.Clave) {
			motivos = append(motivos, ports.MotivoSaltoOrden{Clave: regla.Clave, Etiqueta: regla.Etiqueta})
		}
	}
	return motivos, nil
}

// ResolverMotivoSaltoOrden devuelve la referencia versionada y la huella del
// catálogo de la entrada vigente; una clave ajena o ausente no se admite.
func (c *CatalogoMotivosSalto) ResolverMotivoSaltoOrden(ctx context.Context, clave string) (ports.SaltoOrdenLlamamiento, error) {
	if c == nil || ctx == nil {
		return ports.SaltoOrdenLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	if !claveMotivoSalto.MatchString(clave) {
		return ports.SaltoOrdenLlamamiento{}, ports.ErrMotivoSaltoNoAdmitido
	}
	regla, err := c.resolutor.Regla(ctx, clave)
	if errors.Is(err, reglas.ErrReglaNoEncontrada) {
		return ports.SaltoOrdenLlamamiento{}, ports.ErrMotivoSaltoNoAdmitido
	}
	if err != nil || regla.Clave != clave || regla.Referencia == "" || regla.HuellaCatalogo == "" {
		return ports.SaltoOrdenLlamamiento{}, ports.ErrEmisionLlamamientoNoDisponible
	}
	return ports.SaltoOrdenLlamamiento{Clave: clave, ReglaRef: regla.Referencia, HuellaCatalogo: regla.HuellaCatalogo}, nil
}
