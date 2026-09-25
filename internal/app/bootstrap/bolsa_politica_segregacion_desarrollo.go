package bootstrap

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/config"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

var errPoliticaSegregacionEjemploNoValida = errors.New("bootstrap: catalogo de separacion de funciones de Bolsa no valido")

// publicarPoliticaSegregacionDesarrollo traslada a Bolsa la entrada vigente
// del catálogo de separación de funciones (duda 6 de RRHH). Sin catálogo no
// hace nada y rige la política durable ya publicada; con catálogo, una
// entrada inválida o una base que no la acepta impiden arrancar en lugar de
// ignorar la configuración.
func publicarPoliticaSegregacionDesarrollo(
	ctx context.Context,
	cfg config.Config,
	publicador puertosbolsa.PublicadorPoliticaSegregacion,
	reloj reglas.Reloj,
) error {
	rutas, activas, err := cfg.ReglasEjemploDesarrollo()
	if err != nil || !activas || rutas.BolsaRolesSegregacionSourcePath == "" {
		return err
	}
	if ctx == nil || publicador == nil || reloj == nil {
		return errPoliticaSegregacionEjemploNoValida
	}
	resolutor, err := nuevoResolutorReglasEjemplo(rutas.BolsaRolesSegregacionSourcePath,
		reglas.CatalogoBolsaRolesSegregacion, reglas.ModuloBolsa, nil, reloj)
	if err != nil {
		return err
	}
	ctx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	regla, err := resolutor.Regla(ctx, reglas.BolsaSegundaPersona)
	if err != nil || regla.Unidad != reglas.UnidadLista {
		return errors.Join(errPoliticaSegregacionEjemploNoValida, err)
	}
	politica, err := dominiobolsa.NuevaPoliticaSegregacion(regla.Elementos())
	if err != nil {
		return errors.Join(errPoliticaSegregacionEjemploNoValida, err)
	}
	if _, err := publicador.PublicarPoliticaSegregacion(ctx, puertosbolsa.PublicacionPoliticaSegregacion{
		CatalogoRef: regla.Referencia, CatalogoSHA256: regla.HuellaCatalogo, Politica: politica,
	}); err != nil {
		return errors.Join(errPoliticaSegregacionEjemploNoValida, err)
	}
	return nil
}
