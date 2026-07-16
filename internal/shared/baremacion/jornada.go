package baremacion

import "encoding/json"

// FraccionJornada representa de forma exacta una jornada mayor que cero y no
// superior a la completa. Un tercio es 1/3, nunca una aproximacion decimal.
type FraccionJornada struct {
	valor Racional
}

// NuevaFraccionJornada reduce la fraccion y exige 0 < valor <= 1.
func NuevaFraccionJornada(numerador, denominador int64) (FraccionJornada, error) {
	valor, err := NuevoRacional(numerador, denominador)
	if err != nil {
		return FraccionJornada{}, remapearTipoError("fraccion_jornada", err)
	}
	uno, _ := NuevoRacional(1, 1)
	comparacion, _ := valor.Comparar(uno)
	if valor.numerador <= 0 || comparacion > 0 {
		return FraccionJornada{}, nuevoError("fraccion_jornada", CodigoFueraDeLimites)
	}
	return FraccionJornada{valor: valor}, nil
}

// JornadaCompleta devuelve la fraccion canonica 1/1.
func JornadaCompleta() FraccionJornada {
	return FraccionJornada{valor: Racional{numerador: 1, denominador: 1}}
}

// EsValida comprueba que la fraccion canonica esta en el intervalo (0,1].
func (f FraccionJornada) EsValida() bool {
	if !f.valor.EsValido() || f.valor.numerador <= 0 {
		return false
	}
	return f.valor.numerador <= f.valor.denominador
}

// Racional devuelve el valor exacto para operaciones del motor.
func (f FraccionJornada) Racional() Racional { return f.valor }

// Numerador devuelve el numerador canonico.
func (f FraccionJornada) Numerador() int64 { return f.valor.numerador }

// Denominador devuelve el denominador canonico.
func (f FraccionJornada) Denominador() int64 { return f.valor.denominador }

// EsCompleta indica si la jornada es exactamente 1/1.
func (f FraccionJornada) EsCompleta() bool {
	return f.EsValida() && f.valor.numerador == 1 && f.valor.denominador == 1
}

func (f FraccionJornada) String() string {
	if !f.EsValida() {
		return ""
	}
	return f.valor.String()
}

func (f FraccionJornada) MarshalJSON() ([]byte, error) {
	if !f.EsValida() {
		return nil, nuevoError("fraccion_jornada", CodigoValorInvalido)
	}
	return json.Marshal(f.String())
}

func (f *FraccionJornada) UnmarshalJSON(datos []byte) error {
	if f == nil {
		return nuevoError("fraccion_jornada", CodigoValorInvalido)
	}
	var valor Racional
	if err := valor.UnmarshalJSON(datos); err != nil {
		return remapearTipoError("fraccion_jornada", err)
	}
	construida, err := NuevaFraccionJornada(valor.numerador, valor.denominador)
	if err != nil {
		return err
	}
	*f = construida
	return nil
}
