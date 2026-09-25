package bootstrap

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/config"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// camposPortalMiBolsaReglas lee de la regla b28.campos_portal qué datos ve
// la persona en «Mi bolsa». Sin la regla vigente se muestran todos, como
// antes de existir el catálogo.
type camposPortalMiBolsaReglas struct {
	resolutor *reglas.Resolutor
}

func (c camposPortalMiBolsaReglas) CamposVisiblesMiBolsa(ctx context.Context) ([]string, error) {
	regla, err := c.resolutor.Regla(ctx, reglas.BolsaCamposPortal)
	if errors.Is(err, reglas.ErrReglaNoEncontrada) {
		return puertosbolsa.CamposPortalMiBolsaTodos(), nil
	}
	if err != nil || regla.Unidad != reglas.UnidadLista {
		return nil, errors.Join(puertosbolsa.ErrCamposPortalMiBolsaNoDisponibles, err)
	}
	return puertosbolsa.ValidarCamposPortalMiBolsa(regla.Elementos())
}

// camposPortalMiBolsaDesarrollo compone la lista solo si hay catálogo de
// reglas de Bolsa. Una regla presente pero no válida impide arrancar.
func camposPortalMiBolsaDesarrollo(cfg config.Config, reloj reglas.Reloj) (puertosbolsa.CamposPortalMiBolsa, error) {
	rutas, activas, err := cfg.ReglasEjemploDesarrollo()
	if err != nil || !activas || rutas.BolsaSourcePath == "" {
		return nil, err
	}
	resolutor, err := nuevoResolutorReglasEjemplo(rutas.BolsaSourcePath, reglas.CatalogoBolsa, reglas.ModuloBolsa, nil, reloj)
	if err != nil {
		return nil, err
	}
	campos := camposPortalMiBolsaReglas{resolutor: resolutor}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if _, err := campos.CamposVisiblesMiBolsa(ctx); err != nil {
		return nil, errors.Join(errReglasEjemploNoValidas, err)
	}
	return campos, nil
}
