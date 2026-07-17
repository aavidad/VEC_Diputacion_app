package calculoexperiencia

import (
	"math/big"
	"strings"

	"vec-diputacion-granada/internal/shared/baremacion"
)

const maximoCaracteresExactoResultadoV1 = 2_500

// exactoResultadoV1 conserva un racional no negativo sin exponer math/big ni
// el contador mutable del motor. El formato unico es numerador/denominador.
type exactoResultadoV1 struct {
	canonico string
}

func nuevoExactoResultadoV1(canonico string) (exactoResultadoV1, error) {
	if len(canonico) == 0 || len(canonico) > maximoCaracteresExactoResultadoV1 {
		return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoFueraDeLimites)
	}
	numeradorTexto, denominadorTexto, encontrado := strings.Cut(canonico, "/")
	if !encontrado || numeradorTexto == "" || denominadorTexto == "" ||
		strings.Contains(denominadorTexto, "/") ||
		!enteroDecimalCanonicoResultado(numeradorTexto) ||
		!enteroDecimalCanonicoResultado(denominadorTexto) {
		return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoValorNoCanonico)
	}
	numerador, correcto := new(big.Int).SetString(numeradorTexto, 10)
	if !correcto {
		return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoValorNoCanonico)
	}
	denominador, correcto := new(big.Int).SetString(denominadorTexto, 10)
	if !correcto || numerador.Sign() < 0 || denominador.Sign() <= 0 {
		return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoValorNoCanonico)
	}
	if numerador.BitLen() > maximoBitsComponenteExacto ||
		denominador.BitLen() > maximoBitsComponenteExacto {
		return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoFueraDeLimites)
	}
	comun := new(big.Int).GCD(nil, nil, numerador, denominador)
	if comun.Cmp(big.NewInt(1)) != 0 ||
		(numerador.Sign() == 0 && denominador.Cmp(big.NewInt(1)) != 0) {
		return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoValorNoCanonico)
	}
	return exactoResultadoV1{canonico: canonico}, nil
}

func nuevoExactoResultadoDesdeRacionalV1(valor racionalExacto) (exactoResultadoV1, error) {
	canonico, err := valor.representacionCanonica()
	if err != nil {
		return exactoResultadoV1{}, err
	}
	return nuevoExactoResultadoV1(canonico)
}

func enteroDecimalCanonicoResultado(valor string) bool {
	if valor == "0" {
		return true
	}
	if len(valor) == 0 || valor[0] == '0' {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] < '0' || valor[indice] > '9' {
			return false
		}
	}
	return true
}

func (e exactoResultadoV1) validar() error {
	reconstruido, err := nuevoExactoResultadoV1(e.canonico)
	if err != nil || reconstruido.canonico != e.canonico {
		return nuevoError("resultado.exacto", CodigoValorNoCanonico)
	}
	return nil
}

func (e exactoResultadoV1) texto() string { return e.canonico }

func (e exactoResultadoV1) esEntero() bool {
	return e.validar() == nil && strings.HasSuffix(e.canonico, "/1")
}

func compararExactosResultadoV1(izquierda, derecha exactoResultadoV1) (int, error) {
	izquierdaRacional, err := racionalGrandeResultadoV1(izquierda)
	if err != nil {
		return 0, err
	}
	derechaRacional, err := racionalGrandeResultadoV1(derecha)
	if err != nil {
		return 0, err
	}
	return izquierdaRacional.Cmp(derechaRacional), nil
}

func sumaExactosResultadoV1Coincide(
	izquierda exactoResultadoV1,
	derecha exactoResultadoV1,
	esperada exactoResultadoV1,
) bool {
	primera, err := racionalGrandeResultadoV1(izquierda)
	if err != nil {
		return false
	}
	segunda, err := racionalGrandeResultadoV1(derecha)
	if err != nil {
		return false
	}
	objetivo, err := racionalGrandeResultadoV1(esperada)
	if err != nil {
		return false
	}
	return new(big.Rat).Add(primera, segunda).Cmp(objetivo) == 0
}

func sumarExactosResultadoV1(valores []exactoResultadoV1) (exactoResultadoV1, error) {
	total := new(big.Rat)
	for _, valor := range valores {
		racional, err := racionalGrandeResultadoV1(valor)
		if err != nil {
			return exactoResultadoV1{}, err
		}
		total.Add(total, racional)
		if total.Num().BitLen() > maximoBitsComponenteExacto ||
			total.Denom().BitLen() > maximoBitsComponenteExacto {
			return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoDesbordamiento)
		}
	}
	return nuevoExactoResultadoV1(total.Num().String() + "/" + total.Denom().String())
}

func racionalGrandeResultadoV1(valor exactoResultadoV1) (*big.Rat, error) {
	if err := valor.validar(); err != nil {
		return nil, err
	}
	numerador, denominador, _ := strings.Cut(valor.canonico, "/")
	n, _ := new(big.Int).SetString(numerador, 10)
	d, _ := new(big.Int).SetString(denominador, 10)
	return new(big.Rat).SetFrac(n, d), nil
}

func exactoResultadoDesdeMicropuntosV1(micropuntos int64) (exactoResultadoV1, error) {
	if micropuntos < 0 {
		return exactoResultadoV1{}, nuevoError("resultado.exacto", CodigoResultadoNegativo)
	}
	return nuevoExactoResultadoV1(new(big.Int).SetInt64(micropuntos).String() + "/1")
}

func exactoResultadoDesdeJornadaV1(
	jornada baremacion.FraccionJornada,
) (exactoResultadoV1, error) {
	if !jornada.EsValida() {
		return exactoResultadoV1{}, nuevoError("resultado.jornada", CodigoValorInvalido)
	}
	numerador := new(big.Int).SetInt64(jornada.Numerador()).String()
	denominador := new(big.Int).SetInt64(jornada.Denominador()).String()
	return nuevoExactoResultadoV1(numerador + "/" + denominador)
}

func productoExactoPorMicropuntosResultadoV1(
	valor exactoResultadoV1,
	micropuntos int64,
	esperado exactoResultadoV1,
) bool {
	if micropuntos < 0 {
		return false
	}
	origen, err := racionalGrandeResultadoV1(valor)
	if err != nil {
		return false
	}
	objetivo, err := racionalGrandeResultadoV1(esperado)
	if err != nil {
		return false
	}
	factor := new(big.Rat).SetInt64(micropuntos)
	return new(big.Rat).Mul(origen, factor).Cmp(objetivo) == 0
}

func redondearExactoResultadoV1(
	entrada exactoResultadoV1,
	modo baremacion.ModoRedondeo,
) (exactoResultadoV1, error) {
	valor, err := racionalGrandeResultadoV1(entrada)
	if err != nil || !modo.EsValido() {
		return exactoResultadoV1{}, nuevoError("resultado.redondeo", CodigoValorInvalido)
	}
	cociente := new(big.Int)
	residuo := new(big.Int)
	cociente.QuoRem(valor.Num(), valor.Denom(), residuo)
	incrementar := false
	switch modo {
	case baremacion.RedondeoExacto:
		if residuo.Sign() != 0 {
			return exactoResultadoV1{}, nuevoError("resultado.redondeo", CodigoResultadoNoExacto)
		}
	case baremacion.RedondeoTruncar:
	case baremacion.RedondeoHaciaArriba:
		incrementar = residuo.Sign() != 0
	case baremacion.RedondeoMitadArriba:
		incrementar = new(big.Int).Lsh(new(big.Int).Set(residuo), 1).Cmp(valor.Denom()) >= 0
	case baremacion.RedondeoMitadAlPar:
		comparacion := new(big.Int).Lsh(new(big.Int).Set(residuo), 1).Cmp(valor.Denom())
		incrementar = comparacion > 0 || (comparacion == 0 && cociente.Bit(0) != 0)
	default:
		return exactoResultadoV1{}, nuevoError("resultado.redondeo", CodigoModoRedondeoInvalido)
	}
	if incrementar {
		cociente.Add(cociente, big.NewInt(1))
	}
	return nuevoExactoResultadoV1(cociente.String() + "/1")
}
