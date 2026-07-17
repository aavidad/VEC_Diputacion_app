package calculoexperiencia

import (
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestRacionalExactoConstruyeNormalizaYNoAdmiteNegativos(t *testing.T) {
	contador := nuevoContadorOperaciones()
	cero, err := nuevoRacionalExactoDesdeEntero(contador, 0)
	comprobarRepresentacion(t, cero, "0/1", err)

	fraccionComun, _ := baremacion.NuevoRacional(6, 8)
	fraccion, err := nuevoRacionalExactoDesdeRacional(contador, fraccionComun)
	comprobarRepresentacion(t, fraccion, "3/4", err)

	if _, err := nuevoRacionalExactoDesdeEntero(contador, -1); !errors.Is(err, ErrResultadoNegativo) {
		t.Fatalf("entero negativo: %v", err)
	}
	negativo, _ := baremacion.NuevoRacional(-1, 3)
	if _, err := nuevoRacionalExactoDesdeRacional(contador, negativo); !errors.Is(err, ErrResultadoNegativo) {
		t.Fatalf("racional negativo: %v", err)
	}
	if _, err := nuevoRacionalExactoDesdeRacional(contador, baremacion.Racional{}); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("racional invalido: %v", err)
	}
	if _, err := nuevoRacionalExactoDesdeEntero(nil, 1); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("contador ausente: %v", err)
	}
}

func TestRacionalExactoOperaSinMutarOperandos(t *testing.T) {
	contador := nuevoContadorOperaciones()
	unTercio := exactoDesdeRacional(t, contador, 1, 3)
	unSexto := exactoDesdeRacional(t, contador, 1, 6)

	suma, err := unTercio.sumar(unSexto)
	comprobarRepresentacion(t, suma, "1/2", err)
	resta, err := suma.restar(unSexto)
	comprobarRepresentacion(t, resta, "1/3", err)
	producto, err := unTercio.multiplicar(suma)
	comprobarRepresentacion(t, producto, "1/6", err)
	cociente, err := unTercio.dividirPositivo(suma)
	comprobarRepresentacion(t, cociente, "2/3", err)
	comparacion, err := unSexto.comparar(unTercio)
	if err != nil || comparacion >= 0 {
		t.Fatalf("comparar = %d, %v", comparacion, err)
	}

	comprobarRepresentacion(t, unTercio, "1/3", nil)
	comprobarRepresentacion(t, unSexto, "1/6", nil)

	cero, _ := nuevoRacionalExactoDesdeEntero(contador, 0)
	if _, err := unTercio.dividirPositivo(cero); !errors.Is(err, ErrDivisionPorCero) {
		t.Fatalf("division por cero: %v", err)
	}
}

func TestRacionalExactoRestaFallaCerradoYMinimoCopia(t *testing.T) {
	contador := nuevoContadorOperaciones()
	unTercio := exactoDesdeRacional(t, contador, 1, 3)
	dosTercios := exactoDesdeRacional(t, contador, 2, 3)
	if _, err := unTercio.restar(dosTercios); !errors.Is(err, ErrResultadoNegativo) {
		t.Fatalf("resta negativa: %v", err)
	}

	menor, err := dosTercios.minimo(unTercio)
	comprobarRepresentacion(t, menor, "1/3", err)
	copia, err := menor.copiar()
	comprobarRepresentacion(t, copia, "1/3", err)

	// El test pertenece al paquete y puede forzar una mutacion que ningun
	// consumidor externo puede realizar. Demuestra que las palabras de big.Int
	// de la copia no comparten almacenamiento con el original.
	copia.numerador.SetInt64(99)
	comprobarRepresentacion(t, menor, "1/3", nil)
	comprobarRepresentacion(t, unTercio, "1/3", nil)
}

func TestRacionalExactoTruncaYConservaResto(t *testing.T) {
	contador := nuevoContadorOperaciones()
	valor := exactoDesdeRacional(t, contador, 61, 30)
	truncado, err := valor.truncar()
	comprobarRepresentacion(t, truncado, "2/1", err)
	resto, err := valor.resto()
	comprobarRepresentacion(t, resto, "1/30", err)
	comprobarRepresentacion(t, valor, "61/30", nil)
}

func TestRacionalExactoAplicaTodosLosModosDeRedondeo(t *testing.T) {
	casos := []struct {
		nombre      string
		numerador   int64
		denominador int64
		modo        baremacion.ModoRedondeo
		esperado    int64
		error       error
	}{
		{nombre: "exacto", numerador: 4, denominador: 2, modo: baremacion.RedondeoExacto, esperado: 2},
		{nombre: "exacto_rechaza_resto", numerador: 5, denominador: 2, modo: baremacion.RedondeoExacto, error: ErrResultadoNoExacto},
		{nombre: "truncar", numerador: 5, denominador: 2, modo: baremacion.RedondeoTruncar, esperado: 2},
		{nombre: "arriba", numerador: 5, denominador: 2, modo: baremacion.RedondeoHaciaArriba, esperado: 3},
		{nombre: "mitad_arriba", numerador: 5, denominador: 2, modo: baremacion.RedondeoMitadArriba, esperado: 3},
		{nombre: "mitad_par_baja_par", numerador: 5, denominador: 2, modo: baremacion.RedondeoMitadAlPar, esperado: 2},
		{nombre: "mitad_par_sube_impar", numerador: 7, denominador: 2, modo: baremacion.RedondeoMitadAlPar, esperado: 4},
		{nombre: "mitad_par_por_encima", numerador: 8, denominador: 3, modo: baremacion.RedondeoMitadAlPar, esperado: 3},
		{nombre: "cero_arriba", numerador: 0, denominador: 1, modo: baremacion.RedondeoHaciaArriba, esperado: 0},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			contador := nuevoContadorOperaciones()
			valor := exactoDesdeRacional(t, contador, caso.numerador, caso.denominador)
			resultado, err := valor.redondearAPuntos(caso.modo)
			if caso.error != nil {
				if !errors.Is(err, caso.error) {
					t.Fatalf("error = %v; quiere %v", err, caso.error)
				}
				return
			}
			if err != nil || resultado.Micropuntos() != caso.esperado {
				t.Fatalf("resultado = %d, %v; quiere %d", resultado.Micropuntos(), err, caso.esperado)
			}
		})
	}

	contador := nuevoContadorOperaciones()
	valor := exactoDesdeRacional(t, contador, 1, 2)
	if _, err := valor.redondearAPuntos(baremacion.ModoRedondeo("inventado")); !errors.Is(err, ErrModoRedondeoInvalido) {
		t.Fatalf("modo invalido: %v", err)
	}
}

func TestRacionalExactoRechazaPuntosFueraDelLimite(t *testing.T) {
	contador := nuevoContadorOperaciones()
	maximo, _ := nuevoRacionalExactoDesdeEntero(contador, baremacion.MaximoMicropuntos)
	uno, _ := nuevoRacionalExactoDesdeEntero(contador, 1)
	exceso, err := maximo.sumar(uno)
	if err != nil {
		t.Fatalf("construir exceso representable: %v", err)
	}
	if _, err := exceso.redondearAPuntos(baremacion.RedondeoExacto); !errors.Is(err, ErrDesbordamiento) {
		t.Fatalf("desbordamiento de puntos: %v", err)
	}
}

func TestRacionalExactoLimitaCadaComponenteTrasNormalizar(t *testing.T) {
	contador := nuevoContadorOperaciones()
	base := exactoDesdeRacional(t, contador, 1_000_000_000, 1)
	valor := base
	for paso := 0; paso < 7; paso++ {
		var err error
		valor, err = valor.multiplicar(valor)
		if err != nil {
			t.Fatalf("paso %d inesperado: %v", paso, err)
		}
	}
	if _, err := valor.multiplicar(valor); !errors.Is(err, ErrDesbordamiento) {
		t.Fatalf("limite de numerador: %v", err)
	}

	contadorDenominador := nuevoContadorOperaciones()
	baseDenominador := exactoDesdeRacional(t, contadorDenominador, 1, 1_000_000_000)
	valorDenominador := baseDenominador
	for paso := 0; paso < 7; paso++ {
		var err error
		valorDenominador, err = valorDenominador.multiplicar(valorDenominador)
		if err != nil {
			t.Fatalf("denominador paso %d inesperado: %v", paso, err)
		}
	}
	if _, err := valorDenominador.multiplicar(valorDenominador); !errors.Is(err, ErrDesbordamiento) {
		t.Fatalf("limite de denominador: %v", err)
	}
}

func TestRacionalExactoCancelaAntesDeAplicarLimite(t *testing.T) {
	contador := nuevoContadorOperaciones()
	grande := exactoDesdeRacional(t, contador, 1_000_000_000, 1)
	pequeno := exactoDesdeRacional(t, contador, 1, 1_000_000_000)
	valor := grande
	inverso := pequeno
	for paso := 0; paso < 7; paso++ {
		var err error
		valor, err = valor.multiplicar(valor)
		if err != nil {
			t.Fatalf("valor paso %d: %v", paso, err)
		}
		inverso, err = inverso.multiplicar(inverso)
		if err != nil {
			t.Fatalf("inverso paso %d: %v", paso, err)
		}
	}
	uno, err := valor.multiplicar(inverso)
	comprobarRepresentacion(t, uno, "1/1", err)
}

func TestContadorOperacionesFallaCerrado(t *testing.T) {
	if _, err := nuevoContadorOperacionesConLimite(0); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("limite cero: %v", err)
	}
	if _, err := nuevoContadorOperacionesConLimite(maximoOperacionesExactas + 1); !errors.Is(err, ErrValorInvalido) {
		t.Fatalf("limite excesivo: %v", err)
	}
	contador, err := nuevoContadorOperacionesConLimite(2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevoRacionalExactoDesdeEntero(contador, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := nuevoRacionalExactoDesdeEntero(contador, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := nuevoRacionalExactoDesdeEntero(contador, 3); !errors.Is(err, ErrLimiteOperaciones) {
		t.Fatalf("tercera operacion: %v", err)
	}
}

func TestRacionalExactoRechazaContextosMezclados(t *testing.T) {
	uno, _ := nuevoRacionalExactoDesdeEntero(nuevoContadorOperaciones(), 1)
	dos, _ := nuevoRacionalExactoDesdeEntero(nuevoContadorOperaciones(), 2)
	if _, err := uno.sumar(dos); !errors.Is(err, ErrContextoIncompatible) {
		t.Fatalf("contextos incompatibles: %v", err)
	}
}

func TestRacionalExactoCoincideConBigRatComoOraculo(t *testing.T) {
	generador := rand.New(rand.NewSource(20260717))
	for iteracion := 0; iteracion < 2_000; iteracion++ {
		aNum := generador.Int63n(1_000_000)
		aDen := generador.Int63n(999_999) + 1
		bNum := generador.Int63n(1_000_000)
		bDen := generador.Int63n(999_999) + 1
		contador := nuevoContadorOperaciones()
		a := exactoDesdeRacional(t, contador, aNum, aDen)
		b := exactoDesdeRacional(t, contador, bNum, bDen)
		oraculoA := new(big.Rat).SetFrac(big.NewInt(aNum), big.NewInt(aDen))
		oraculoB := new(big.Rat).SetFrac(big.NewInt(bNum), big.NewInt(bDen))

		suma, err := a.sumar(b)
		compararConOraculo(t, suma, new(big.Rat).Add(oraculoA, oraculoB), err)
		producto, err := a.multiplicar(b)
		compararConOraculo(t, producto, new(big.Rat).Mul(oraculoA, oraculoB), err)
		comparacion, err := a.comparar(b)
		if err != nil || comparacion != oraculoA.Cmp(oraculoB) {
			t.Fatalf("iteracion %d: comparar = %d, %v; quiere %d", iteracion, comparacion, err, oraculoA.Cmp(oraculoB))
		}
		if bNum != 0 {
			cociente, err := a.dividirPositivo(b)
			compararConOraculo(t, cociente, new(big.Rat).Quo(oraculoA, oraculoB), err)
		}
	}
}

func TestRedondeoExactoCoincideConFundamentoCompartido(t *testing.T) {
	bases := []int64{0, 1, 5, 7, 1_000_000, baremacion.MaximoMicropuntos}
	factores := [][2]int64{{0, 1}, {1, 3}, {1, 2}, {2, 3}, {1, 1}}
	modos := []baremacion.ModoRedondeo{
		baremacion.RedondeoExacto,
		baremacion.RedondeoTruncar,
		baremacion.RedondeoHaciaArriba,
		baremacion.RedondeoMitadArriba,
		baremacion.RedondeoMitadAlPar,
	}
	for _, base := range bases {
		puntos, _ := baremacion.PuntosDesdeMicropuntos(base)
		for _, componentes := range factores {
			factor, _ := baremacion.NuevoRacional(componentes[0], componentes[1])
			for _, modo := range modos {
				contador := nuevoContadorOperaciones()
				exactoBase, _ := nuevoRacionalExactoDesdeEntero(contador, base)
				exactoFactor, _ := nuevoRacionalExactoDesdeRacional(contador, factor)
				producto, err := exactoBase.multiplicar(exactoFactor)
				if err != nil {
					t.Fatal(err)
				}
				obtenido, errObtenido := producto.redondearAPuntos(modo)
				esperado, errEsperado := puntos.MultiplicarRedondeado(factor, modo)
				if obtenido != esperado || !erroresEquivalentes(errObtenido, errEsperado) {
					t.Fatalf("%d * %s (%s): obtenido=%v,%v esperado=%v,%v", base, factor, modo, obtenido, errObtenido, esperado, errEsperado)
				}
			}
		}
	}
}

func TestErrorCalculoNoFiltraValores(t *testing.T) {
	err := nuevoError("racional_exacto", CodigoValorInvalido)
	if err.Error() != "calculo de experiencia: racional_exacto: valor_invalido" {
		t.Fatalf("texto de error = %q", err)
	}
	var tipado *ErrorCalculo
	if !errors.As(err, &tipado) || tipado.Codigo() != CodigoValorInvalido || tipado.Campo() != "racional_exacto" {
		t.Fatalf("error tipado = %#v", tipado)
	}
	if codigoError(err) != CodigoValorInvalido || codigoError(errors.New("externo")) != CodigoValorInvalido {
		t.Fatal("clasificacion inesperada")
	}
}

func exactoDesdeRacional(
	t *testing.T,
	contador *contadorOperaciones,
	numerador int64,
	denominador int64,
) racionalExacto {
	t.Helper()
	valor, err := baremacion.NuevoRacional(numerador, denominador)
	if err != nil {
		t.Fatalf("racional %d/%d: %v", numerador, denominador, err)
	}
	resultado, err := nuevoRacionalExactoDesdeRacional(contador, valor)
	if err != nil {
		t.Fatalf("exacto %d/%d: %v", numerador, denominador, err)
	}
	return resultado
}

func comprobarRepresentacion(t *testing.T, valor racionalExacto, esperado string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("construir %s: %v", esperado, err)
	}
	representacion, err := valor.representacionCanonica()
	if err != nil || representacion != esperado {
		t.Fatalf("representacion = %q, %v; quiere %q", representacion, err, esperado)
	}
}

func compararConOraculo(t *testing.T, obtenido racionalExacto, esperado *big.Rat, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("operacion exacta: %v", err)
	}
	representacion, err := obtenido.representacionCanonica()
	if err != nil || representacion != esperado.RatString() {
		t.Fatalf("obtenido=%q,%v; oraculo=%q", representacion, err, esperado.RatString())
	}
}

func erroresEquivalentes(obtenido, esperado error) bool {
	if obtenido == nil || esperado == nil {
		return obtenido == nil && esperado == nil
	}
	if errors.Is(obtenido, ErrResultadoNoExacto) {
		return errors.Is(esperado, baremacion.ErrResultadoNoExacto)
	}
	if errors.Is(obtenido, ErrDesbordamiento) {
		return errors.Is(esperado, baremacion.ErrDesbordamiento)
	}
	return false
}
