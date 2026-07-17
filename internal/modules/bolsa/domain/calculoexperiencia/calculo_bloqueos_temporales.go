package calculoexperiencia

import "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"

func registrarBloqueoTemporalResultadoV1(
	registrador *registradorResultadoExperienciaV1,
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
	temporales resultadoAplicacionesTemporales,
) error {
	if registrador == nil || len(registrador.bloqueos) != 0 || !temporales.bloqueada() {
		return nuevoError("calculo.bloqueo_temporal", CodigoContextoIncompatible)
	}
	intervalos, err := materializarIntervalosPuntuacionV1(
		plan, entrada, seleccion, temporales,
	)
	if err != nil {
		return err
	}
	bloqueos, err := materializarBloqueosTemporalesResultadoV1(temporales)
	if err != nil {
		return err
	}
	for _, intervalo := range intervalos {
		if err := registrador.registrarIntervalo(intervalo); err != nil {
			return err
		}
	}
	for _, bloqueo := range bloqueos {
		if err := registrador.registrarBloqueo(bloqueo); err != nil {
			return err
		}
	}
	return nil
}

func materializarBloqueosTemporalesResultadoV1(
	temporales resultadoAplicacionesTemporales,
) ([]BloqueoResultadoExperienciaV1, error) {
	if len(temporales.bloqueos) == 0 || len(temporales.bloqueos) > maximoBloqueosResultadoV1 {
		return nil, nuevoError("calculo.bloqueos_temporales", CodigoContextoIncompatible)
	}
	resultado := make([]BloqueoResultadoExperienciaV1, len(temporales.bloqueos))
	for indice, bloqueo := range temporales.bloqueos {
		if bloqueo.codigo != bloqueoTemporalSolapeRechazado {
			return nil, nuevoError("calculo.bloqueo_temporal", CodigoContextoIncompatible)
		}
		tramos, err := ordenarParTramosBloqueoTemporalV1(
			bloqueo.tramoPrimero, bloqueo.tramoSegundo,
		)
		if err != nil {
			return nil, err
		}
		material := BloqueoResultadoExperienciaV1{
			codigo: BloqueoResultadoSolape, tramos: tramos,
			grupoClave: bloqueo.grupoClave,
		}
		if err := validarBloqueoResultadoV1(material); err != nil {
			return nil, err
		}
		resultado[indice] = material
	}
	return resultado, nil
}

func ordenarParTramosBloqueoTemporalV1(
	primero reglasbaremo.ReferenciaVersionada,
	segundo reglasbaremo.ReferenciaVersionada,
) ([]reglasbaremo.ReferenciaVersionada, error) {
	comparacion := compararReferenciasSeleccion(primero, segundo)
	if comparacion == 0 {
		return nil, nuevoError("calculo.bloqueo_temporal.tramos", CodigoContextoIncompatible)
	}
	if comparacion < 0 {
		return []reglasbaremo.ReferenciaVersionada{primero, segundo}, nil
	}
	return []reglasbaremo.ReferenciaVersionada{segundo, primero}, nil
}
