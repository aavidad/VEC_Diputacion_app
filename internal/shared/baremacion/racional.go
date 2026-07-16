package baremacion

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// MaximoComponenteRacional limita numerador y denominador ya reducidos. Hace
// que los productos cruzados quepan en int64 sin depender de math/big.
const MaximoComponenteRacional int64 = 1_000_000_000

// Racional es una fraccion canonica: denominador positivo y componentes
// coprimos. El cero se representa exclusivamente como 0/1.
type Racional struct {
	numerador   int64
	denominador int64
}

// NuevoRacional normaliza el signo y reduce la fraccion.
func NuevoRacional(numerador, denominador int64) (Racional, error) {
	if denominador == 0 {
		return Racional{}, nuevoError("racional", CodigoDenominadorCero)
	}
	if numerador == 0 {
		return Racional{denominador: 1}, nil
	}

	// La magnitud se calcula en uint64 para poder representar abs(MinInt64).
	// El signo se aplica solo despues de reducir y comprobar que el componente
	// canonico cabe holgadamente en int64.
	magnitudNumerador := magnitudEntero(numerador)
	magnitudDenominador := magnitudEntero(denominador)
	divisor := maximoComunDivisor(magnitudNumerador, magnitudDenominador)
	magnitudNumerador /= divisor
	magnitudDenominador /= divisor
	limite := uint64(MaximoComponenteRacional)
	if magnitudNumerador > limite || magnitudDenominador > limite {
		return Racional{}, nuevoError("racional", CodigoFueraDeLimites)
	}

	numeradorCanonico := int64(magnitudNumerador)
	if (numerador < 0) != (denominador < 0) {
		numeradorCanonico = -numeradorCanonico
	}
	return Racional{
		numerador:   numeradorCanonico,
		denominador: int64(magnitudDenominador),
	}, nil
}

// Numerador devuelve el numerador canonico con signo.
func (r Racional) Numerador() int64 { return r.numerador }

// Denominador devuelve el denominador canonico, siempre positivo.
func (r Racional) Denominador() int64 { return r.denominador }

// EsValido verifica reduccion, signo y limites defensivos.
func (r Racional) EsValido() bool {
	if r.denominador <= 0 || magnitudEntero(r.numerador) > uint64(MaximoComponenteRacional) ||
		r.denominador > MaximoComponenteRacional {
		return false
	}
	if r.numerador == 0 {
		return r.denominador == 1
	}
	return maximoComunDivisor(magnitudEntero(r.numerador), uint64(r.denominador)) == 1
}

// Comparar devuelve -1, 0 o 1 mediante productos cruzados seguros.
func (r Racional) Comparar(otro Racional) (int, error) {
	if !r.EsValido() || !otro.EsValido() {
		return 0, nuevoError("racional", CodigoValorInvalido)
	}
	izquierda := r.numerador * otro.denominador
	derecha := otro.numerador * r.denominador
	if izquierda < derecha {
		return -1, nil
	}
	if izquierda > derecha {
		return 1, nil
	}
	return 0, nil
}

// Sumar conserva exactitud y rechaza resultados fuera del limite defensivo.
func (r Racional) Sumar(otro Racional) (Racional, error) {
	if !r.EsValido() || !otro.EsValido() {
		return Racional{}, nuevoError("racional", CodigoValorInvalido)
	}
	comun := int64(maximoComunDivisor(uint64(r.denominador), uint64(otro.denominador)))
	factorIzquierdo := otro.denominador / comun
	factorDerecho := r.denominador / comun
	izquierda := r.numerador * factorIzquierdo
	derecha := otro.numerador * factorDerecho
	if (derecha > 0 && izquierda > math.MaxInt64-derecha) ||
		(derecha < 0 && izquierda < math.MinInt64-derecha) {
		return Racional{}, nuevoError("racional", CodigoDesbordamiento)
	}
	denominador := r.denominador * factorIzquierdo
	return NuevoRacional(izquierda+derecha, denominador)
}

// Restar conserva exactitud y rechaza resultados fuera del limite defensivo.
func (r Racional) Restar(otro Racional) (Racional, error) {
	if !otro.EsValido() {
		return Racional{}, nuevoError("racional", CodigoValorInvalido)
	}
	inversoAditivo := Racional{numerador: -otro.numerador, denominador: otro.denominador}
	return r.Sumar(inversoAditivo)
}

// Multiplicar cancela factores antes del producto para reducir el riesgo de
// desbordamiento y aplica despues el limite canonico.
func (r Racional) Multiplicar(otro Racional) (Racional, error) {
	if !r.EsValido() || !otro.EsValido() {
		return Racional{}, nuevoError("racional", CodigoValorInvalido)
	}
	comunUno := int64(maximoComunDivisor(magnitudEntero(r.numerador), uint64(otro.denominador)))
	comunDos := int64(maximoComunDivisor(magnitudEntero(otro.numerador), uint64(r.denominador)))
	numeradorUno := r.numerador / comunUno
	numeradorDos := otro.numerador / comunDos
	denominadorUno := r.denominador / comunDos
	denominadorDos := otro.denominador / comunUno
	return NuevoRacional(numeradorUno*numeradorDos, denominadorUno*denominadorDos)
}

// Dividir conserva exactitud y rechaza de forma tipada la division por cero.
func (r Racional) Dividir(otro Racional) (Racional, error) {
	if !r.EsValido() || !otro.EsValido() {
		return Racional{}, nuevoError("racional", CodigoValorInvalido)
	}
	if otro.numerador == 0 {
		return Racional{}, nuevoError("racional", CodigoDivisionPorCero)
	}
	inverso, err := NuevoRacional(otro.denominador, otro.numerador)
	if err != nil {
		return Racional{}, err
	}
	return r.Multiplicar(inverso)
}

func (r Racional) String() string {
	if !r.EsValido() {
		return ""
	}
	return strconv.FormatInt(r.numerador, 10) + "/" + strconv.FormatInt(r.denominador, 10)
}

func (r Racional) MarshalJSON() ([]byte, error) {
	if !r.EsValido() {
		return nil, nuevoError("racional", CodigoValorInvalido)
	}
	return json.Marshal(r.String())
}

func (r *Racional) UnmarshalJSON(datos []byte) error {
	if r == nil {
		return nuevoError("racional", CodigoValorInvalido)
	}
	var texto string
	if err := json.Unmarshal(datos, &texto); err != nil {
		return nuevoError("racional", CodigoValorNoCanonico)
	}
	partes := strings.Split(texto, "/")
	if len(partes) != 2 {
		return nuevoError("racional", CodigoValorNoCanonico)
	}
	numerador, errNumerador := strconv.ParseInt(partes[0], 10, 64)
	denominador, errDenominador := strconv.ParseInt(partes[1], 10, 64)
	if errNumerador != nil || errDenominador != nil {
		return nuevoError("racional", CodigoValorNoCanonico)
	}
	construido, err := NuevoRacional(numerador, denominador)
	if err != nil {
		return err
	}
	if construido.String() != texto {
		return nuevoError("racional", CodigoValorNoCanonico)
	}
	*r = construido
	return nil
}

func maximoComunDivisor(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// magnitudEntero devuelve |valor| incluso para MinInt64, cuya magnitud no cabe
// en un int64 con signo.
func magnitudEntero(valor int64) uint64 {
	if valor >= 0 {
		return uint64(valor)
	}
	return uint64(-(valor + 1)) + 1
}
