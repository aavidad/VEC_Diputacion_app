package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

// normalizarPeriodoEfectivo convierte las fechas informadas por la fuente al
// unico formato temporal que usa el calculador: [desde, hasta_exclusivo). El
// corte de convocatoria es siempre inclusivo. El booleano indica si queda al
// menos un dia; un periodo vacio por corte o por extremo exclusivo es valido y
// no se representa mediante un IntervaloCivil invalido.
//
// En periodos cerrados inclusivos se recorta primero el ultimo dia informado y
// solo despues se obtiene Siguiente. Asi nunca se intenta avanzar desde
// 9999-12-31 cuando el corte efectivo es anterior.
func normalizarPeriodoEfectivo(
	periodo PeriodoServicio,
	corteInclusivo baremacion.FechaCivil,
	extremo reglasbaremo.TratamientoExtremoFinal,
) (baremacion.IntervaloCivil, bool, error) {
	if err := periodo.validar(); err != nil {
		return baremacion.IntervaloCivil{}, false,
			nuevoError("periodo_efectivo.periodo", CodigoValorInvalido)
	}
	if !corteInclusivo.EsValida() {
		return baremacion.IntervaloCivil{}, false,
			nuevoError("periodo_efectivo.corte", CodigoValorInvalido)
	}
	if extremo != reglasbaremo.ExtremoFinalInclusivo &&
		extremo != reglasbaremo.ExtremoFinalExclusivo {
		return baremacion.IntervaloCivil{}, false,
			nuevoError("periodo_efectivo.extremo", CodigoValorInvalido)
	}

	corteExclusivo, err := corteInclusivo.Siguiente()
	if err != nil {
		return baremacion.IntervaloCivil{}, false,
			nuevoError("periodo_efectivo.corte", CodigoFueraDeLimites)
	}
	desde := periodo.Desde()
	if comparacion, _ := desde.Comparar(corteExclusivo); comparacion >= 0 {
		return baremacion.IntervaloCivil{}, false, nil
	}

	hastaExclusivo := corteExclusivo
	if finInformado, cerrado := periodo.FinInformado(); cerrado {
		switch extremo {
		case reglasbaremo.ExtremoFinalInclusivo:
			ultimoIncluido := minimoFechaCivil(finInformado, corteInclusivo)
			if comparacion, _ := desde.Comparar(ultimoIncluido); comparacion > 0 {
				return baremacion.IntervaloCivil{}, false, nil
			}
			hastaExclusivo, err = ultimoIncluido.Siguiente()
			if err != nil {
				return baremacion.IntervaloCivil{}, false,
					nuevoError("periodo_efectivo.fin", CodigoFueraDeLimites)
			}
		case reglasbaremo.ExtremoFinalExclusivo:
			hastaExclusivo = minimoFechaCivil(finInformado, corteExclusivo)
		}
	}

	comparacion, err := desde.Comparar(hastaExclusivo)
	if err != nil {
		return baremacion.IntervaloCivil{}, false,
			nuevoError("periodo_efectivo", CodigoValorInvalido)
	}
	if comparacion >= 0 {
		return baremacion.IntervaloCivil{}, false, nil
	}
	intervalo, err := baremacion.NuevoIntervaloCivil(desde, hastaExclusivo)
	if err != nil {
		return baremacion.IntervaloCivil{}, false,
			nuevoError("periodo_efectivo", CodigoValorInvalido)
	}
	return intervalo, true, nil
}

func minimoFechaCivil(
	izquierda baremacion.FechaCivil,
	derecha baremacion.FechaCivil,
) baremacion.FechaCivil {
	comparacion, _ := izquierda.Comparar(derecha)
	if comparacion <= 0 {
		return izquierda
	}
	return derecha
}
