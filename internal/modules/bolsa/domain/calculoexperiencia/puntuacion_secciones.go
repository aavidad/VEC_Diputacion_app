package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func calcularSeccionesPuntuacionV1(
	secciones []reglasbaremo.SeccionBaremo,
	reglas []reglaCalculadaPuntuacionV1,
	contador *contadorOperaciones,
	cero racionalExacto,
) ([]SubtotalSeccionResultadoExperienciaV1, baremacion.Puntos, error) {
	posiciones := make(map[string]int, len(secciones))
	sumas := make([]racionalExacto, len(secciones))
	for indice, seccion := range secciones {
		posiciones[seccion.Clave()] = indice
		sumas[indice] = cero
	}
	for _, regla := range reglas {
		posicion, existe := posiciones[regla.material.seccionClave]
		if !existe {
			return nil, baremacion.Puntos{}, errorContextoPuntuacionV1()
		}
		var err error
		sumas[posicion], err = sumas[posicion].sumar(regla.final)
		if err != nil {
			return nil, baremacion.Puntos{}, err
		}
	}
	resultado := make([]SubtotalSeccionResultadoExperienciaV1, len(secciones))
	totalExacto := cero
	for indice, seccion := range secciones {
		limite, err := nuevoRacionalExactoDesdeEntero(
			contador, seccion.PuntosMaximos().Micropuntos(),
		)
		if err != nil {
			return nil, baremacion.Puntos{}, err
		}
		tope, final, err := aplicarTopeRacionalPuntuacionV1(sumas[indice], limite)
		if err != nil {
			return nil, baremacion.Puntos{}, err
		}
		puntos, err := final.convertirAPuntos()
		if err != nil {
			return nil, baremacion.Puntos{}, err
		}
		antes, err := nuevoExactoResultadoDesdeRacionalV1(sumas[indice])
		if err != nil {
			return nil, baremacion.Puntos{}, err
		}
		resultado[indice] = SubtotalSeccionResultadoExperienciaV1{
			seccionClave: seccion.Clave(), antesTope: antes,
			tope: tope, puntosFinales: puntos,
		}
		totalExacto, err = totalExacto.sumar(final)
		if err != nil {
			return nil, baremacion.Puntos{}, err
		}
	}
	total, err := totalExacto.convertirAPuntos()
	if err != nil {
		return nil, baremacion.Puntos{}, err
	}
	return resultado, total, nil
}
