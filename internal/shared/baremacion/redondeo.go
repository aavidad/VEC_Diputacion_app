package baremacion

// ModoRedondeo fija como se convierte un resultado racional positivo a
// micropuntos. La regla publicada debe elegirlo de forma expresa; el motor no
// aplica una opcion implicita.
type ModoRedondeo string

const (
	RedondeoExacto      ModoRedondeo = "exacto"
	RedondeoTruncar     ModoRedondeo = "truncar"
	RedondeoHaciaArriba ModoRedondeo = "hacia_arriba"
	RedondeoMitadArriba ModoRedondeo = "mitad_arriba"
	RedondeoMitadAlPar  ModoRedondeo = "mitad_al_par"
)

// EsValido impide que una cadena configurada desde administracion seleccione
// un comportamiento no revisado por el dominio.
func (m ModoRedondeo) EsValido() bool {
	switch m {
	case RedondeoExacto, RedondeoTruncar, RedondeoHaciaArriba,
		RedondeoMitadArriba, RedondeoMitadAlPar:
		return true
	default:
		return false
	}
}

// MultiplicarRedondeado aplica un factor racional no negativo y redondea una
// sola vez a micropuntos. La descomposicion evita multiplicar directamente un
// valor de hasta 9e15 por el numerador del factor.
func (p Puntos) MultiplicarRedondeado(factor Racional, modo ModoRedondeo) (Puntos, error) {
	// El modo exacto conserva deliberadamente el mismo contrato, incluidos la
	// clasificacion y el orden de precedencia de los errores, que la operacion
	// exacta original.
	if modo == RedondeoExacto {
		return p.MultiplicarExacto(factor)
	}
	if !p.EsValido() || !factor.EsValido() || !modo.EsValido() {
		return Puntos{}, nuevoError("puntos", CodigoValorInvalido)
	}
	if factor.numerador < 0 {
		return Puntos{}, nuevoError("puntos", CodigoResultadoNegativo)
	}
	if factor.numerador == 0 || p.micropuntos == 0 {
		return Puntos{}, nil
	}

	cociente := p.micropuntos / factor.denominador
	resto := p.micropuntos % factor.denominador
	if cociente > MaximoMicropuntos/factor.numerador {
		return Puntos{}, nuevoError("puntos", CodigoDesbordamiento)
	}
	parteEntera := cociente * factor.numerador
	productoResto := resto * factor.numerador
	parteResto := productoResto / factor.denominador
	residuo := productoResto % factor.denominador
	if parteResto > MaximoMicropuntos-parteEntera {
		return Puntos{}, nuevoError("puntos", CodigoDesbordamiento)
	}
	resultado := parteEntera + parteResto

	incrementar := false
	switch modo {
	case RedondeoTruncar:
	case RedondeoHaciaArriba:
		incrementar = residuo != 0
	case RedondeoMitadArriba:
		incrementar = residuo*2 >= factor.denominador
	case RedondeoMitadAlPar:
		dobleResiduo := residuo * 2
		incrementar = dobleResiduo > factor.denominador ||
			(dobleResiduo == factor.denominador && resultado%2 != 0)
	}
	if incrementar {
		if resultado == MaximoMicropuntos {
			return Puntos{}, nuevoError("puntos", CodigoDesbordamiento)
		}
		resultado++
	}
	return Puntos{micropuntos: resultado}, nil
}
