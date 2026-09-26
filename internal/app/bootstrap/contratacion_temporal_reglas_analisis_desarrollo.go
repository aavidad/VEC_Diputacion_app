package bootstrap

import (
	"context"
	"errors"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/reglas"
)

// minutosJornadaCompletaPredeterminadaDesarrollo es la jornada completa de
// referencia cuando no hay catálogo de reglas o este no publica la regla c07:
// 37 h 30 min semanales. Con catálogo manda la regla c07.jornada_completa.
const minutosJornadaCompletaPredeterminadaDesarrollo = 37*60 + 30

// maximoMinutosJornadaCompletaDesarrollo es una semana entera: una regla
// mayor no es una jornada y se rechaza.
const maximoMinutosJornadaCompletaDesarrollo = 7 * 24 * 60

var errJornadaCompletaNoDisponible = errors.New("bootstrap: jornada completa de referencia no disponible")

// fuentesReglasAnalisisDesarrollo agrupa los catálogos de ejemplo que consume
// el análisis de Contratación temporal: la regla c07 (jornada completa), las
// opciones del análisis (c12 a c15) y el catálogo ct.retribuciones (coste).
// Cada campo nulo significa «sin catálogo».
type fuentesReglasAnalisisDesarrollo struct {
	jornada       fuenteJornadaCompletaDesarrollo
	retribuciones *fuenteRetribucionesDesarrollo
	// opciones son las modalidades, causas, entradas de retención de crédito,
	// duraciones máximas y urgencia del análisis (reglas c12 a c15).
	opciones *opcionesAnalisisCTDesarrollo
}

// nuevasFuentesReglasAnalisisDesarrollo compone solo lo declarado bajo la doble
// llave de desarrollo; fuera de ella un catálogo declarado impide arrancar.
func nuevasFuentesReglasAnalisisDesarrollo(
	cfg config.Config,
	reloj reglas.Reloj,
) (fuentesReglasAnalisisDesarrollo, error) {
	rutas, activas, err := cfg.ReglasEjemploDesarrollo()
	if err != nil || !activas {
		return fuentesReglasAnalisisDesarrollo{}, err
	}
	var fuentes fuentesReglasAnalisisDesarrollo
	if fuentes.jornada.resolutor, err = nuevoResolutorReglasEjemplo(
		rutas.CTSourcePath, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, nil, reloj,
	); err != nil {
		return fuentesReglasAnalisisDesarrollo{}, err
	}
	if _, err = fuentes.jornada.minutos(context.Background()); err != nil {
		return fuentesReglasAnalisisDesarrollo{}, errors.Join(errReglasEjemploNoValidas, err)
	}
	if fuentes.opciones, err = nuevasOpcionesAnalisisCT(context.Background(), fuentes.jornada.resolutor); err != nil {
		return fuentesReglasAnalisisDesarrollo{}, errors.Join(errReglasEjemploNoValidas, err)
	}
	if fuentes.retribuciones, err = nuevaFuenteRetribucionesDesarrollo(
		rutas.CTRetribucionesSourcePath, reloj,
	); err != nil {
		return fuentesReglasAnalisisDesarrollo{}, err
	}
	return fuentes, nil
}

// fuenteJornadaCompletaDesarrollo resuelve la jornada completa en cada
// consulta, para que una nueva versión del catálogo se aplique sin reiniciar.
type fuenteJornadaCompletaDesarrollo struct {
	resolutor *reglas.Resolutor
}

// minutos devuelve los minutos semanales de la jornada completa. Sin catálogo
// o sin la regla c07 se aplica el valor predeterminado; un catálogo declarado
// pero no disponible, o una regla con otra unidad, no se sustituye por él.
func (f fuenteJornadaCompletaDesarrollo) minutos(ctx context.Context) (int, error) {
	regla, err := f.resolutor.Regla(ctx, reglas.CTJornadaCompleta)
	switch {
	case errors.Is(err, reglas.ErrReglasNoConfiguradas), errors.Is(err, reglas.ErrReglaNoEncontrada):
		return minutosJornadaCompletaPredeterminadaDesarrollo, nil
	case err != nil:
		return 0, errJornadaCompletaNoDisponible
	case regla.Unidad != reglas.UnidadMinutosSemanales || regla.Cantidad < 1 ||
		regla.Cantidad > maximoMinutosJornadaCompletaDesarrollo:
		return 0, errJornadaCompletaNoDisponible
	}
	return regla.Cantidad, nil
}
