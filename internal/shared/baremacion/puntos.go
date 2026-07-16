package baremacion

import (
	"encoding/json"
	"strconv"
)

const (
	// MicropuntosPorPunto fija seis decimales exactos por punto.
	MicropuntosPorPunto int64 = 1_000_000
	// MaximoMicropuntos es un limite tecnico defensivo, no un tope de baremo.
	MaximoMicropuntos int64 = 9_000_000_000_000_000
)

// Puntos representa una puntuacion no negativa en micropuntos. Su campo es
// privado para que no pueda existir un valor fuera de limites.
type Puntos struct {
	micropuntos int64
}

// PuntosDesdeMicropuntos construye una puntuacion exacta.
func PuntosDesdeMicropuntos(micropuntos int64) (Puntos, error) {
	if micropuntos < 0 || micropuntos > MaximoMicropuntos {
		return Puntos{}, nuevoError("puntos", CodigoFueraDeLimites)
	}
	return Puntos{micropuntos: micropuntos}, nil
}

// Micropuntos devuelve la representacion entera exacta.
func (p Puntos) Micropuntos() int64 { return p.micropuntos }

// EsValido comprueba tambien valores obtenidos por una decodificacion hostil.
func (p Puntos) EsValido() bool {
	return p.micropuntos >= 0 && p.micropuntos <= MaximoMicropuntos
}

// Comparar devuelve -1, 0 o 1.
func (p Puntos) Comparar(otros Puntos) (int, error) {
	if !p.EsValido() || !otros.EsValido() {
		return 0, nuevoError("puntos", CodigoValorInvalido)
	}
	if p.micropuntos < otros.micropuntos {
		return -1, nil
	}
	if p.micropuntos > otros.micropuntos {
		return 1, nil
	}
	return 0, nil
}

// Sumar falla de forma cerrada antes de superar el limite del tipo.
func (p Puntos) Sumar(otros Puntos) (Puntos, error) {
	if !p.EsValido() || !otros.EsValido() {
		return Puntos{}, nuevoError("puntos", CodigoValorInvalido)
	}
	if otros.micropuntos > MaximoMicropuntos-p.micropuntos {
		return Puntos{}, nuevoError("puntos", CodigoDesbordamiento)
	}
	return Puntos{micropuntos: p.micropuntos + otros.micropuntos}, nil
}

// Restar rechaza las puntuaciones negativas.
func (p Puntos) Restar(otros Puntos) (Puntos, error) {
	if !p.EsValido() || !otros.EsValido() {
		return Puntos{}, nuevoError("puntos", CodigoValorInvalido)
	}
	if otros.micropuntos > p.micropuntos {
		return Puntos{}, nuevoError("puntos", CodigoResultadoNegativo)
	}
	return Puntos{micropuntos: p.micropuntos - otros.micropuntos}, nil
}

// MultiplicarExacto aplica un factor racional sin introducir una politica de
// redondeo. Si el resultado no cabe en un micropunto, la regla llamante debe
// elegir de forma explicita su politica y no puede perder la fraccion aqui.
func (p Puntos) MultiplicarExacto(factor Racional) (Puntos, error) {
	if !p.EsValido() || !factor.EsValido() {
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
	productoResto := resto * factor.numerador // ambos son como maximo 1e9
	if productoResto%factor.denominador != 0 {
		return Puntos{}, nuevoError("puntos", CodigoResultadoNoExacto)
	}
	parteResto := productoResto / factor.denominador
	if parteResto > MaximoMicropuntos-parteEntera {
		return Puntos{}, nuevoError("puntos", CodigoDesbordamiento)
	}
	return Puntos{micropuntos: parteEntera + parteResto}, nil
}

func (p Puntos) String() string {
	if !p.EsValido() {
		return ""
	}
	return strconv.FormatInt(p.micropuntos, 10)
}

// MarshalJSON usa una cadena decimal: preserva enteros superiores al rango
// exacto de Number en JavaScript y produce una unica representacion.
func (p Puntos) MarshalJSON() ([]byte, error) {
	if !p.EsValido() {
		return nil, nuevoError("puntos", CodigoValorInvalido)
	}
	return json.Marshal(p.String())
}

// UnmarshalJSON acepta exclusivamente la cadena decimal canonica.
func (p *Puntos) UnmarshalJSON(datos []byte) error {
	if p == nil {
		return nuevoError("puntos", CodigoValorInvalido)
	}
	var texto string
	if err := json.Unmarshal(datos, &texto); err != nil {
		return nuevoError("puntos", CodigoValorNoCanonico)
	}
	valor, err := strconv.ParseInt(texto, 10, 64)
	if err != nil || strconv.FormatInt(valor, 10) != texto {
		return nuevoError("puntos", CodigoValorNoCanonico)
	}
	construidos, err := PuntosDesdeMicropuntos(valor)
	if err != nil {
		return err
	}
	*p = construidos
	return nil
}
