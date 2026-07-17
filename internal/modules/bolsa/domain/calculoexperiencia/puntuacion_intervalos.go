package calculoexperiencia

// materializarIntervalosPuntuacionV1 conserva la particion completa en el
// orden de seleccion. Es deliberadamente independiente de la aritmetica y se
// puede usar cuando la fase temporal ha detectado solapes: el orquestador debe
// registrar primero todos estos intervalos y solo despues sus bloqueos.
func materializarIntervalosPuntuacionV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
	temporales resultadoAplicacionesTemporales,
) ([]IntervaloAplicacionResultadoExperienciaV1, error) {
	contexto, err := prepararContextoComunPuntuacionV1(plan, entrada, seleccion, temporales)
	if err != nil {
		return nil, err
	}
	return intervalosDesdeContextoPuntuacionV1(contexto), nil
}

func intervalosDesdeContextoPuntuacionV1(
	contexto contextoPuntuacionV1,
) []IntervaloAplicacionResultadoExperienciaV1 {
	resultado := make([]IntervaloAplicacionResultadoExperienciaV1, len(contexto.orden))
	for indice, entrada := range contexto.orden {
		intervalo := IntervaloAplicacionResultadoExperienciaV1{
			tramo: entrada.tramo.referencia, reglaClave: entrada.regla.Clave(),
			periodo: entrada.tramo.periodo,
			extremo: entrada.regla.UnidadTemporal().ExtremoFinal(),
		}
		if entrada.efectiva {
			intervalo.efectivo = entrada.temporal.intervalo
			intervalo.tieneEfectivo = true
			intervalo.dias = uint64(entrada.temporal.dias)
		} else {
			switch entrada.exclusion.razon {
			case exclusionTemporalFueraDeCorte:
				intervalo.razon = RazonPosteriorCorte
			case exclusionTemporalIntervaloVacio:
				intervalo.razon = RazonIntervaloVacio
			}
		}
		resultado[indice] = intervalo
	}
	return resultado
}
