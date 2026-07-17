package baremacion

import (
	"errors"
	"math/big"
	"testing"
)

func TestMultiplicarRedondeadoAplicaCadaPoliticaExplicita(t *testing.T) {
	base, _ := PuntosDesdeMicropuntos(5)
	mitad, _ := NuevoRacional(1, 2)

	casos := []struct {
		nombre string
		modo   ModoRedondeo
		quiere int64
	}{
		{nombre: "truncar", modo: RedondeoTruncar, quiere: 2},
		{nombre: "hacia_arriba", modo: RedondeoHaciaArriba, quiere: 3},
		{nombre: "mitad_arriba", modo: RedondeoMitadArriba, quiere: 3},
		{nombre: "mitad_al_par", modo: RedondeoMitadAlPar, quiere: 2},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := base.MultiplicarRedondeado(mitad, caso.modo)
			if err != nil {
				t.Fatalf("multiplicar: %v", err)
			}
			if resultado.Micropuntos() != caso.quiere {
				t.Fatalf("resultado = %d; quiere %d", resultado.Micropuntos(), caso.quiere)
			}
		})
	}
}

func TestMultiplicarRedondeadoMitadAlParElevaUnImpar(t *testing.T) {
	base, _ := PuntosDesdeMicropuntos(7)
	mitad, _ := NuevoRacional(1, 2)
	resultado, err := base.MultiplicarRedondeado(mitad, RedondeoMitadAlPar)
	if err != nil {
		t.Fatalf("multiplicar: %v", err)
	}
	if resultado.Micropuntos() != 4 {
		t.Fatalf("resultado = %d; quiere 4", resultado.Micropuntos())
	}
}

func TestMultiplicarRedondeadoExactoNoPierdeFracciones(t *testing.T) {
	base, _ := PuntosDesdeMicropuntos(5)
	mitad, _ := NuevoRacional(1, 2)
	if _, err := base.MultiplicarRedondeado(mitad, RedondeoExacto); !errors.Is(err, ErrResultadoNoExacto) {
		t.Fatalf("error = %v; quiere resultado no exacto", err)
	}
	par, _ := PuntosDesdeMicropuntos(6)
	resultado, err := par.MultiplicarRedondeado(mitad, RedondeoExacto)
	if err != nil || resultado.Micropuntos() != 3 {
		t.Fatalf("resultado exacto = %d, %v", resultado.Micropuntos(), err)
	}
}

func TestMultiplicarRedondeadoExactoEquivaleAMultiplicarExacto(t *testing.T) {
	mitad, _ := NuevoRacional(1, 2)
	dos, _ := NuevoRacional(2, 1)
	negativo, _ := NuevoRacional(-1, 2)
	bordeFraccionario, _ := NuevoRacional(1_000_000_000, 999_999_999)

	casos := []struct {
		nombre string
		base   Puntos
		factor Racional
	}{
		{nombre: "cero", base: Puntos{}, factor: mitad},
		{nombre: "exacto", base: Puntos{micropuntos: 6}, factor: mitad},
		{nombre: "fraccionario", base: Puntos{micropuntos: 5}, factor: mitad},
		{nombre: "desbordamiento_entero", base: Puntos{micropuntos: MaximoMicropuntos}, factor: dos},
		{
			nombre: "fraccionario_y_fuera_de_limite",
			base:   Puntos{micropuntos: MaximoMicropuntos},
			factor: bordeFraccionario,
		},
		{nombre: "factor_negativo", base: Puntos{micropuntos: 1}, factor: negativo},
		{nombre: "puntos_invalidos", base: Puntos{micropuntos: -1}, factor: mitad},
		{nombre: "racional_invalido", base: Puntos{micropuntos: 1}, factor: Racional{}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			redondeado, errRedondeado := caso.base.MultiplicarRedondeado(caso.factor, RedondeoExacto)
			exacto, errExacto := caso.base.MultiplicarExacto(caso.factor)
			if redondeado != exacto {
				t.Fatalf("resultado redondeado = %v; exacto = %v", redondeado, exacto)
			}
			compararErroresDeValor(t, errRedondeado, errExacto)
		})
	}
}

func TestMultiplicarRedondeadoCoincideConBigIntEnBordes(t *testing.T) {
	racionales := []Racional{
		debeRacional(t, 0, 1),
		debeRacional(t, 1, 3),
		debeRacional(t, 1, 2),
		debeRacional(t, 2, 3),
		debeRacional(t, 1, 1),
		debeRacional(t, 3, 2),
		debeRacional(t, 999_999_999, 1_000_000_000),
		debeRacional(t, 1_000_000_000, 999_999_999),
		debeRacional(t, 999_999_999, 999_999_998),
	}
	bases := []int64{
		0, 1, 2, 3, 5, 7,
		8_999_999_991_000_000,
		MaximoMicropuntos - 1,
		MaximoMicropuntos,
	}
	modos := []ModoRedondeo{
		RedondeoTruncar,
		RedondeoHaciaArriba,
		RedondeoMitadArriba,
		RedondeoMitadAlPar,
	}

	for _, baseMicropuntos := range bases {
		base, _ := PuntosDesdeMicropuntos(baseMicropuntos)
		for _, factor := range racionales {
			for _, modo := range modos {
				quiere, desborda := redondearConBigInt(baseMicropuntos, factor, modo)
				resultado, err := base.MultiplicarRedondeado(factor, modo)
				if desborda {
					if !errors.Is(err, ErrDesbordamiento) {
						t.Fatalf("%d * %s (%s): error = %v; quiere desbordamiento", baseMicropuntos, factor, modo, err)
					}
					continue
				}
				if err != nil {
					t.Fatalf("%d * %s (%s): %v", baseMicropuntos, factor, modo, err)
				}
				if resultado.Micropuntos() != quiere {
					t.Fatalf("%d * %s (%s) = %d; quiere %d", baseMicropuntos, factor, modo, resultado.Micropuntos(), quiere)
				}
			}
		}
	}
}

func TestMultiplicarRedondeadoFallaCerrado(t *testing.T) {
	base, _ := PuntosDesdeMicropuntos(MaximoMicropuntos)
	dos, _ := NuevoRacional(2, 1)
	if _, err := base.MultiplicarRedondeado(dos, RedondeoTruncar); !errors.Is(err, ErrDesbordamiento) {
		t.Fatalf("desbordamiento = %v", err)
	}
	negativo, _ := NuevoRacional(-1, 2)
	if _, err := base.MultiplicarRedondeado(negativo, RedondeoTruncar); !errors.Is(err, ErrResultadoNegativo) {
		t.Fatalf("factor negativo = %v", err)
	}
	if _, err := base.MultiplicarRedondeado(Racional{}, RedondeoTruncar); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("factor nulo = %v", err)
	}
	if _, err := base.MultiplicarRedondeado(dos, ModoRedondeo("inventado")); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("modo desconocido = %v", err)
	}
}

func debeRacional(t *testing.T, numerador, denominador int64) Racional {
	t.Helper()
	resultado, err := NuevoRacional(numerador, denominador)
	if err != nil {
		t.Fatalf("construir racional %d/%d: %v", numerador, denominador, err)
	}
	return resultado
}

func compararErroresDeValor(t *testing.T, obtenido, esperado error) {
	t.Helper()
	if obtenido == nil || esperado == nil {
		if obtenido != nil || esperado != nil {
			t.Fatalf("error redondeado = %v; exacto = %v", obtenido, esperado)
		}
		return
	}
	var errorObtenido, errorEsperado *ErrorValor
	if !errors.As(obtenido, &errorObtenido) || !errors.As(esperado, &errorEsperado) {
		t.Fatalf("errores no tipados: redondeado = %T; exacto = %T", obtenido, esperado)
	}
	if errorObtenido.Codigo() != errorEsperado.Codigo() || errorObtenido.Tipo() != errorEsperado.Tipo() {
		t.Fatalf("error redondeado = %v; exacto = %v", obtenido, esperado)
	}
}

func redondearConBigInt(base int64, factor Racional, modo ModoRedondeo) (int64, bool) {
	producto := new(big.Int).Mul(big.NewInt(base), big.NewInt(factor.Numerador()))
	denominador := big.NewInt(factor.Denominador())
	cociente := new(big.Int)
	resto := new(big.Int)
	cociente.QuoRem(producto, denominador, resto)

	incrementar := false
	dobleResto := new(big.Int).Lsh(new(big.Int).Set(resto), 1)
	comparacionMitad := dobleResto.Cmp(denominador)
	switch modo {
	case RedondeoHaciaArriba:
		incrementar = resto.Sign() != 0
	case RedondeoMitadArriba:
		incrementar = comparacionMitad >= 0
	case RedondeoMitadAlPar:
		incrementar = comparacionMitad > 0 || (comparacionMitad == 0 && cociente.Bit(0) != 0)
	}
	if incrementar {
		cociente.Add(cociente, big.NewInt(1))
	}
	if cociente.Cmp(big.NewInt(MaximoMicropuntos)) > 0 {
		return 0, true
	}
	return cociente.Int64(), false
}
