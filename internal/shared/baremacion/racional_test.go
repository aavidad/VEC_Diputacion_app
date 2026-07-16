package baremacion

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"testing"
)

func TestRacionalSeReduceYNormaliza(t *testing.T) {
	t.Parallel()

	casos := []struct {
		numerador   int64
		denominador int64
		esperado    string
	}{
		{6, 8, "3/4"},
		{-6, -8, "3/4"},
		{6, -8, "-3/4"},
		{0, -8, "0/1"},
		{math.MaxInt64, math.MaxInt64, "1/1"},
		{math.MinInt64, math.MinInt64, "1/1"},
		{math.MinInt64, math.MinInt64 / 2, "2/1"},
		{math.MinInt64 / 2, math.MinInt64, "1/2"},
		{math.MaxInt64, -math.MaxInt64, "-1/1"},
		{0, math.MinInt64, "0/1"},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.esperado, func(t *testing.T) {
			t.Parallel()
			resultado, err := NuevoRacional(caso.numerador, caso.denominador)
			if err != nil || resultado.String() != caso.esperado {
				t.Fatalf("NuevoRacional(%d,%d) = %q, %v", caso.numerador, caso.denominador, resultado, err)
			}
		})
	}
	fraccion, _ := NuevoRacional(-6, 8)
	if fraccion.Numerador() != -3 || fraccion.Denominador() != 4 {
		t.Fatalf("componentes = %d/%d", fraccion.Numerador(), fraccion.Denominador())
	}

	if _, err := NuevoRacional(1, 0); !errors.Is(err, ErrDenominadorCero) {
		t.Fatalf("denominador cero: error = %v", err)
	}
	if _, err := NuevoRacional(math.MinInt64, 1); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("MinInt64 irreducible: error = %v", err)
	}
	if _, err := NuevoRacional(MaximoComponenteRacional+1, 1); !errors.Is(err, ErrFueraDeLimites) {
		t.Fatalf("componente excesivo: error = %v", err)
	}
}

func TestRacionalOperaSinComaFlotante(t *testing.T) {
	t.Parallel()

	unTercio, _ := NuevoRacional(1, 3)
	unSexto, _ := NuevoRacional(1, 6)
	suma, err := unTercio.Sumar(unSexto)
	if err != nil || suma.String() != "1/2" {
		t.Fatalf("1/3 + 1/6 = %s, %v", suma, err)
	}
	resta, err := suma.Restar(unSexto)
	if err != nil || resta != unTercio {
		t.Fatalf("1/2 - 1/6 = %s, %v", resta, err)
	}
	producto, err := unTercio.Multiplicar(suma)
	if err != nil || producto.String() != "1/6" {
		t.Fatalf("1/3 * 1/2 = %s, %v", producto, err)
	}
	cociente, err := unTercio.Dividir(suma)
	if err != nil || cociente.String() != "2/3" {
		t.Fatalf("1/3 / 1/2 = %s, %v", cociente, err)
	}
	cero, _ := NuevoRacional(0, 1)
	if _, err := unTercio.Dividir(cero); !errors.Is(err, ErrDivisionPorCero) {
		t.Fatalf("division por cero: error = %v", err)
	}
}

func TestRacionalJSONExigeFormaReducida(t *testing.T) {
	t.Parallel()

	original, _ := NuevoRacional(-1, 3)
	datos, err := json.Marshal(original)
	if err != nil || string(datos) != `"-1/3"` {
		t.Fatalf("Marshal = %s, %v", datos, err)
	}
	var recuperado Racional
	if err := json.Unmarshal(datos, &recuperado); err != nil || recuperado != original {
		t.Fatalf("roundtrip = %s, %v", recuperado, err)
	}

	for _, noCanonico := range []string{`"2/6"`, `"1/-3"`, `"01/3"`, `"0/8"`, `0.5`, `"1/0"`} {
		var destino Racional
		err := json.Unmarshal([]byte(noCanonico), &destino)
		if err == nil {
			t.Errorf("Unmarshal(%s) no rechazo", noCanonico)
		}
	}
}

func TestErrorValorEsTipado(t *testing.T) {
	t.Parallel()

	_, err := NuevoRacional(1, 0)
	var errorValor *ErrorValor
	if !errors.As(err, &errorValor) {
		t.Fatalf("errors.As: %T", err)
	}
	if errorValor.Codigo() != CodigoDenominadorCero || errorValor.Tipo() != "racional" {
		t.Fatalf("error tipado = codigo %q, tipo %q", errorValor.Codigo(), errorValor.Tipo())
	}
	if errorValor.Error() != "baremacion: racional: denominador_cero" {
		t.Fatalf("Error() = %q", errorValor.Error())
	}
}

func FuzzRacionalRoundtripJSON(f *testing.F) {
	semillas := [][2]int64{{0, 1}, {1, 3}, {-6, 8}, {1, 0}, {math.MinInt64, -1}, {math.MaxInt64, math.MaxInt64}}
	for _, semilla := range semillas {
		f.Add(semilla[0], semilla[1])
	}
	f.Fuzz(func(t *testing.T, numerador, denominador int64) {
		original, err := NuevoRacional(numerador, denominador)
		if err != nil {
			return
		}
		datos, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal(%s): %v", original, err)
		}
		var recuperado Racional
		if err := json.Unmarshal(datos, &recuperado); err != nil {
			t.Fatalf("Unmarshal(%s): %v", datos, err)
		}
		if recuperado != original {
			t.Fatalf("roundtrip: %v != %v", recuperado, original)
		}
	})
}

func FuzzRacionalCoincideConBigRat(f *testing.F) {
	semillas := [][2]int64{
		{0, 1},
		{1, 3},
		{-6, 8},
		{math.MinInt64, math.MinInt64},
		{math.MinInt64 / 2, math.MinInt64},
		{math.MinInt64, 1},
		{math.MaxInt64, math.MaxInt64},
		{1, 0},
	}
	for _, semilla := range semillas {
		f.Add(semilla[0], semilla[1])
	}
	f.Fuzz(func(t *testing.T, numerador, denominador int64) {
		obtenido, err := NuevoRacional(numerador, denominador)
		if denominador == 0 {
			if !errors.Is(err, ErrDenominadorCero) {
				t.Fatalf("denominador cero: error = %v", err)
			}
			return
		}

		oraculo := new(big.Rat).SetFrac(big.NewInt(numerador), big.NewInt(denominador))
		magnitudNumerador := new(big.Int).Abs(new(big.Int).Set(oraculo.Num()))
		limite := big.NewInt(MaximoComponenteRacional)
		representable := magnitudNumerador.Cmp(limite) <= 0 && oraculo.Denom().Cmp(limite) <= 0
		if !representable {
			if !errors.Is(err, ErrFueraDeLimites) {
				t.Fatalf("%d/%d fuera de limite: resultado=%s, error=%v", numerador, denominador, obtenido, err)
			}
			return
		}
		if err != nil {
			t.Fatalf("%d/%d representable: %v", numerador, denominador, err)
		}
		if obtenido.String() != oraculo.String() {
			t.Fatalf("%d/%d: obtenido=%s, big.Rat=%s", numerador, denominador, obtenido, oraculo)
		}
	})
}
