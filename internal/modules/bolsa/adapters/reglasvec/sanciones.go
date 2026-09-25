// Package reglasvec traduce el catálogo de reglas del núcleo (paquete
// internal/vec/reglas) a los puertos de Bolsa. No fija valores: todo lo que
// decide (consecuencias, efecto, plazos y estados) viene del catálogo.
package reglasvec

import (
	"context"
	"errors"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

var _ ports.CatalogoSancionesParticipacion = (*CatalogoSanciones)(nil)

// CatalogoSanciones lee en cada consulta las entradas «b24.sancion.*», el
// plazo del recurso («b24.consecuencias») y sus estados.
type CatalogoSanciones struct {
	resolutor *reglas.Resolutor
}

// NuevoCatalogoSanciones devuelve nil sin resolutor: el servicio lo trata
// como «sin catálogo» y conserva la conducta anterior.
func NuevoCatalogoSanciones(resolutor *reglas.Resolutor) *CatalogoSanciones {
	if resolutor == nil {
		return nil
	}
	return &CatalogoSanciones{resolutor: resolutor}
}

func (c *CatalogoSanciones) Consecuencias(ctx context.Context) ([]ports.ConsecuenciaSancion, error) {
	if c == nil {
		return nil, ports.ErrSancionesNoConfiguradas
	}
	todas, err := c.resolutor.Reglas(ctx)
	if err != nil {
		return nil, errorCatalogo(err)
	}
	consecuencias := make([]ports.ConsecuenciaSancion, 0, 8)
	for _, regla := range todas {
		if !strings.HasPrefix(regla.Clave, reglas.BolsaPrefijoSanciones) {
			continue
		}
		consecuencia, err := consecuenciaDesdeRegla(regla)
		if err != nil {
			return nil, err
		}
		consecuencias = append(consecuencias, consecuencia)
	}
	return consecuencias, nil
}

func (c *CatalogoSanciones) EstadosRecurso(ctx context.Context) ([]string, error) {
	if c == nil {
		return nil, ports.ErrSancionesNoConfiguradas
	}
	regla, err := c.resolutor.Regla(ctx, reglas.BolsaEstadosRecurso)
	if err != nil {
		return nil, errorCatalogo(err)
	}
	estados := regla.Elementos()
	if len(estados) == 0 {
		return nil, ports.ErrSancionesNoConfiguradas
	}
	for _, estado := range estados {
		if !dominiobolsa.EstadoRecursoValido(estado) {
			return nil, ports.ErrSancionesNoConfiguradas
		}
	}
	return estados, nil
}

// ResolverSancion devuelve la consecuencia, el fin de la suspensión si la
// regla define un plazo, y el vencimiento del recurso de reposición, ambos
// desde la notificación.
func (c *CatalogoSanciones) ResolverSancion(ctx context.Context, clave string, notificadaEn time.Time) (ports.ResolucionCatalogoSancion, error) {
	if c == nil {
		return ports.ResolucionCatalogoSancion{}, ports.ErrSancionesNoConfiguradas
	}
	if !strings.HasPrefix(clave, reglas.BolsaPrefijoSanciones) {
		return ports.ResolucionCatalogoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	regla, err := c.resolutor.Regla(ctx, clave)
	if errors.Is(err, reglas.ErrReglaNoEncontrada) {
		return ports.ResolucionCatalogoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	if err != nil {
		return ports.ResolucionCatalogoSancion{}, errorCatalogo(err)
	}
	consecuencia, err := consecuenciaDesdeRegla(regla)
	if err != nil {
		return ports.ResolucionCatalogoSancion{}, err
	}
	resultado := ports.ResolucionCatalogoSancion{Consecuencia: consecuencia}
	if consecuencia.ConPlazo {
		_, fin, err := c.resolutor.Vencimiento(ctx, clave, notificadaEn, "")
		if err != nil {
			return ports.ResolucionCatalogoSancion{}, errorCatalogo(err)
		}
		resultado.SuspensionHasta = fin.UltimoDia
	}
	recurso, vencimiento, err := c.resolutor.Vencimiento(ctx, reglas.BolsaConsecuencias, notificadaEn, "")
	if err != nil {
		return ports.ResolucionCatalogoSancion{}, errorCatalogo(err)
	}
	resultado.Recurso = ports.PlazoSancion{UltimoDia: vencimiento.UltimoDia, ReglaRef: recurso.Referencia, Huella: recurso.HuellaCatalogo}
	return resultado, nil
}

func consecuenciaDesdeRegla(regla reglas.Regla) (ports.ConsecuenciaSancion, error) {
	efecto := regla.Atributos["efecto"]
	if !dominiobolsa.EfectoSancionValido(efecto) || !dominiobolsa.ClaveConsecuenciaSancionValida(regla.Clave) {
		return ports.ConsecuenciaSancion{}, ports.ErrSancionesNoConfiguradas
	}
	// Solo una suspensión (pausar) admite fin calculado; una baja con plazo
	// sería contradictoria y se rechaza en lugar de interpretarse.
	conPlazo := regla.Unidad.EsPlazo() && regla.Computo != ""
	if conPlazo && efecto != dominiobolsa.OperacionPausar {
		return ports.ConsecuenciaSancion{}, ports.ErrSancionesNoConfiguradas
	}
	return ports.ConsecuenciaSancion{
		Clave: regla.Clave, Etiqueta: regla.Etiqueta, Descripcion: regla.Descripcion, Efecto: efecto,
		Articulo: regla.Articulo, Ejemplo: regla.EsEjemplo(), ConPlazo: conPlazo,
		ReglaRef: regla.Referencia, Huella: regla.HuellaCatalogo,
	}, nil
}

func errorCatalogo(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return errors.Join(ports.ErrSancionesNoConfiguradas, err)
}
