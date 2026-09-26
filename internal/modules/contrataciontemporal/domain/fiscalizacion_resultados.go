package domain

import "errors"

// Efectos de cada resultado de fiscalización (duda 5 de RRHH). Qué resultados
// se admiten y qué efecto tiene cada uno lo fija el catálogo de reglas; aquí
// solo se fija el vocabulario y el efecto que la persistencia sabe aplicar a
// cada resultado, que es una invariante técnica: CT52, CT93 y CT120 abren el
// retorno a la unidad gestora solo con el desfavorable.

// EfectoResultadoFiscalizacion es lo que ocurre con el expediente.
type EfectoResultadoFiscalizacion string

const (
	// EfectoFiscalizacionContinua: el expediente sigue su tramitación.
	EfectoFiscalizacionContinua EfectoResultadoFiscalizacion = "continua"
	// EfectoFiscalizacionVuelveUnidadGestora: vuelve a la unidad gestora
	// para subsanar; tras subsanar se fiscaliza de nuevo.
	EfectoFiscalizacionVuelveUnidadGestora EfectoResultadoFiscalizacion = "vuelve_unidad_gestora"
)

// ErrPoliticaResultadosFiscalizacionInvalida: la política no forma un conjunto
// de resultados conocidos, sin repetir y con efectos que se sepan aplicar.
var ErrPoliticaResultadosFiscalizacionInvalida = errors.New(
	"contratacion temporal: politica de resultados de fiscalizacion invalida",
)

// PoliticaResultadosFiscalizacion es la lista ordenada de resultados que
// Intervención puede registrar con el efecto de cada uno.
type PoliticaResultadosFiscalizacion struct {
	resultados []ResultadoFiscalizacion
	efectos    map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion
}

// efectoAplicable es el único efecto que la persistencia sabe aplicar a cada
// resultado.
func efectoAplicable(r ResultadoFiscalizacion) EfectoResultadoFiscalizacion {
	if r == FiscalizacionDesfavorable {
		return EfectoFiscalizacionVuelveUnidadGestora
	}
	return EfectoFiscalizacionContinua
}

// PoliticaResultadosFiscalizacionPredeterminada es la conducta sin catálogo:
// los tres resultados, favorable y favorable con observaciones continúan y el
// desfavorable vuelve a la unidad gestora.
func PoliticaResultadosFiscalizacionPredeterminada() PoliticaResultadosFiscalizacion {
	p, _ := NuevaPoliticaResultadosFiscalizacion(
		[]ResultadoFiscalizacion{FiscalizacionFavorable, FiscalizacionFavorableConObservaciones, FiscalizacionDesfavorable},
		map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion{
			FiscalizacionFavorable:                 EfectoFiscalizacionContinua,
			FiscalizacionFavorableConObservaciones: EfectoFiscalizacionContinua,
			FiscalizacionDesfavorable:              EfectoFiscalizacionVuelveUnidadGestora,
		},
	)
	return p
}

// NuevaPoliticaResultadosFiscalizacion valida la lista del catálogo: cada
// resultado conocido, una sola vez, con efecto declarado y aplicable, y al
// menos uno que permita continuar.
func NuevaPoliticaResultadosFiscalizacion(
	resultados []ResultadoFiscalizacion,
	efectos map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion,
) (PoliticaResultadosFiscalizacion, error) {
	p := PoliticaResultadosFiscalizacion{efectos: map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion{}}
	continua := false
	for _, r := range resultados {
		efecto, declarado := efectos[r]
		if !r.Valido() || !declarado || efecto != efectoAplicable(r) {
			return PoliticaResultadosFiscalizacion{}, ErrPoliticaResultadosFiscalizacionInvalida
		}
		if _, repetido := p.efectos[r]; repetido {
			return PoliticaResultadosFiscalizacion{}, ErrPoliticaResultadosFiscalizacionInvalida
		}
		p.resultados = append(p.resultados, r)
		p.efectos[r] = efecto
		continua = continua || efecto == EfectoFiscalizacionContinua
	}
	if !continua || len(efectos) != len(p.efectos) {
		return PoliticaResultadosFiscalizacion{}, ErrPoliticaResultadosFiscalizacionInvalida
	}
	return p, nil
}

// Admite indica si el catálogo permite registrar el resultado.
func (p PoliticaResultadosFiscalizacion) Admite(r ResultadoFiscalizacion) bool {
	_, ok := p.efectos[r]
	return ok
}

// Efecto devuelve el efecto del resultado admitido.
func (p PoliticaResultadosFiscalizacion) Efecto(r ResultadoFiscalizacion) (EfectoResultadoFiscalizacion, bool) {
	e, ok := p.efectos[r]
	return e, ok
}

// Resultados devuelve una copia de la lista admitida, en su orden.
func (p PoliticaResultadosFiscalizacion) Resultados() []ResultadoFiscalizacion {
	return append([]ResultadoFiscalizacion(nil), p.resultados...)
}
