package calculoexperiencia

import (
	"math/big"

	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	maximoBitsComponenteExacto = 4096
	maximoOperacionesExactas   = 1_000_000
)

// contadorOperaciones limita conjuntamente el trabajo aritmetico de un
// calculo. Es deliberadamente privado y debe crearse uno por ejecucion.
type contadorOperaciones struct {
	realizadas uint64
	limite     uint64
}

func nuevoContadorOperaciones() *contadorOperaciones {
	return &contadorOperaciones{limite: maximoOperacionesExactas}
}

func nuevoContadorOperacionesConLimite(limite uint64) (*contadorOperaciones, error) {
	if limite == 0 || limite > maximoOperacionesExactas {
		return nil, nuevoError("limite_operaciones", CodigoValorInvalido)
	}
	return &contadorOperaciones{limite: limite}, nil
}

func (c *contadorOperaciones) consumir() error {
	if c == nil || c.limite == 0 || c.limite > maximoOperacionesExactas {
		return nuevoError("contador_operaciones", CodigoValorInvalido)
	}
	if c.realizadas >= c.limite {
		return nuevoError("contador_operaciones", CodigoLimiteOperaciones)
	}
	c.realizadas++
	return nil
}

// racionalExacto es una fraccion canonica, no negativa y privada. big.Int se
// almacena por valor y ninguna operacion devuelve sus componentes mutables.
type racionalExacto struct {
	numerador   big.Int
	denominador big.Int
	contador    *contadorOperaciones
}

func nuevoRacionalExactoDesdeEntero(
	contador *contadorOperaciones,
	valor int64,
) (racionalExacto, error) {
	if err := consumirConstruccion(contador); err != nil {
		return racionalExacto{}, err
	}
	if valor < 0 {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoResultadoNegativo)
	}
	return construirRacionalExacto(contador, big.NewInt(valor), big.NewInt(1))
}

func nuevoRacionalExactoDesdeRacional(
	contador *contadorOperaciones,
	valor baremacion.Racional,
) (racionalExacto, error) {
	if err := consumirConstruccion(contador); err != nil {
		return racionalExacto{}, err
	}
	if !valor.EsValido() {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoValorInvalido)
	}
	if valor.Numerador() < 0 {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoResultadoNegativo)
	}
	return construirRacionalExacto(
		contador,
		big.NewInt(valor.Numerador()),
		big.NewInt(valor.Denominador()),
	)
}

func consumirConstruccion(contador *contadorOperaciones) error {
	if contador == nil {
		return nuevoError("contador_operaciones", CodigoValorInvalido)
	}
	return contador.consumir()
}

// construirRacionalExacto toma posesion logica de copias de los componentes,
// normaliza y aplica el limite despues de cancelar factores comunes.
func construirRacionalExacto(
	contador *contadorOperaciones,
	numerador *big.Int,
	denominador *big.Int,
) (racionalExacto, error) {
	if contador == nil || numerador == nil || denominador == nil {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoValorInvalido)
	}
	n := new(big.Int).Set(numerador)
	d := new(big.Int).Set(denominador)
	if d.Sign() == 0 {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoDivisionPorCero)
	}
	if n.Sign() < 0 || d.Sign() < 0 {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoResultadoNegativo)
	}
	if n.Sign() == 0 {
		d.SetInt64(1)
	} else {
		comun := new(big.Int).GCD(nil, nil, n, d)
		n.Quo(n, comun)
		d.Quo(d, comun)
	}
	if n.BitLen() > maximoBitsComponenteExacto || d.BitLen() > maximoBitsComponenteExacto {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoDesbordamiento)
	}
	var resultado racionalExacto
	resultado.numerador.Set(n)
	resultado.denominador.Set(d)
	resultado.contador = contador
	return resultado, nil
}

func (r racionalExacto) esValido() bool {
	if r.contador == nil || r.numerador.Sign() < 0 || r.denominador.Sign() <= 0 ||
		r.numerador.BitLen() > maximoBitsComponenteExacto ||
		r.denominador.BitLen() > maximoBitsComponenteExacto {
		return false
	}
	if r.numerador.Sign() == 0 {
		return r.denominador.Cmp(big.NewInt(1)) == 0
	}
	comun := new(big.Int).GCD(nil, nil, &r.numerador, &r.denominador)
	return comun.Cmp(big.NewInt(1)) == 0
}

func (r racionalExacto) representacionCanonica() (string, error) {
	if !r.esValido() {
		return "", nuevoError("racional_exacto", CodigoValorInvalido)
	}
	return r.numerador.String() + "/" + r.denominador.String(), nil
}

func (r racionalExacto) comprobarOperacion(otro racionalExacto) error {
	if !r.esValido() || !otro.esValido() {
		return nuevoError("racional_exacto", CodigoValorInvalido)
	}
	if r.contador != otro.contador {
		return nuevoError("racional_exacto", CodigoContextoIncompatible)
	}
	return r.contador.consumir()
}

func (r racionalExacto) comprobarOperacionUnaria() error {
	if !r.esValido() {
		return nuevoError("racional_exacto", CodigoValorInvalido)
	}
	return r.contador.consumir()
}

func (r racionalExacto) sumar(otro racionalExacto) (racionalExacto, error) {
	if err := r.comprobarOperacion(otro); err != nil {
		return racionalExacto{}, err
	}
	izquierda := new(big.Int).Mul(&r.numerador, &otro.denominador)
	derecha := new(big.Int).Mul(&otro.numerador, &r.denominador)
	numerador := new(big.Int).Add(izquierda, derecha)
	denominador := new(big.Int).Mul(&r.denominador, &otro.denominador)
	return construirRacionalExacto(r.contador, numerador, denominador)
}

func (r racionalExacto) restar(otro racionalExacto) (racionalExacto, error) {
	if err := r.comprobarOperacion(otro); err != nil {
		return racionalExacto{}, err
	}
	comun := new(big.Int).GCD(nil, nil, &r.denominador, &otro.denominador)
	factorIzquierdo := new(big.Int).Quo(new(big.Int).Set(&otro.denominador), comun)
	factorDerecho := new(big.Int).Quo(new(big.Int).Set(&r.denominador), comun)
	izquierda := new(big.Int).Mul(&r.numerador, factorIzquierdo)
	derecha := new(big.Int).Mul(&otro.numerador, factorDerecho)
	if izquierda.Cmp(derecha) < 0 {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoResultadoNegativo)
	}
	numerador := new(big.Int).Sub(izquierda, derecha)
	denominador := new(big.Int).Mul(&r.denominador, factorIzquierdo)
	return construirRacionalExacto(r.contador, numerador, denominador)
}

func (r racionalExacto) multiplicar(otro racionalExacto) (racionalExacto, error) {
	if err := r.comprobarOperacion(otro); err != nil {
		return racionalExacto{}, err
	}
	// Cancelar en cruz antes de multiplicar reduce el pico de memoria y permite
	// que un resultado pequeno siga siendo valido aunque los factores sean grandes.
	comunUno := new(big.Int).GCD(nil, nil, &r.numerador, &otro.denominador)
	comunDos := new(big.Int).GCD(nil, nil, &otro.numerador, &r.denominador)
	numeradorUno := new(big.Int).Quo(new(big.Int).Set(&r.numerador), comunUno)
	numeradorDos := new(big.Int).Quo(new(big.Int).Set(&otro.numerador), comunDos)
	denominadorUno := new(big.Int).Quo(new(big.Int).Set(&r.denominador), comunDos)
	denominadorDos := new(big.Int).Quo(new(big.Int).Set(&otro.denominador), comunUno)
	return construirRacionalExacto(
		r.contador,
		new(big.Int).Mul(numeradorUno, numeradorDos),
		new(big.Int).Mul(denominadorUno, denominadorDos),
	)
}

func (r racionalExacto) dividirPositivo(otro racionalExacto) (racionalExacto, error) {
	if err := r.comprobarOperacion(otro); err != nil {
		return racionalExacto{}, err
	}
	if otro.numerador.Sign() == 0 {
		return racionalExacto{}, nuevoError("racional_exacto", CodigoDivisionPorCero)
	}
	comunUno := new(big.Int).GCD(nil, nil, &r.numerador, &otro.numerador)
	comunDos := new(big.Int).GCD(nil, nil, &otro.denominador, &r.denominador)
	numeradorUno := new(big.Int).Quo(new(big.Int).Set(&r.numerador), comunUno)
	denominadorOtro := new(big.Int).Quo(new(big.Int).Set(&otro.denominador), comunDos)
	denominadorUno := new(big.Int).Quo(new(big.Int).Set(&r.denominador), comunDos)
	numeradorOtro := new(big.Int).Quo(new(big.Int).Set(&otro.numerador), comunUno)
	return construirRacionalExacto(
		r.contador,
		new(big.Int).Mul(numeradorUno, denominadorOtro),
		new(big.Int).Mul(denominadorUno, numeradorOtro),
	)
}

func (r racionalExacto) comparar(otro racionalExacto) (int, error) {
	if err := r.comprobarOperacion(otro); err != nil {
		return 0, err
	}
	izquierda := new(big.Int).Mul(&r.numerador, &otro.denominador)
	derecha := new(big.Int).Mul(&otro.numerador, &r.denominador)
	return izquierda.Cmp(derecha), nil
}

// minimo devuelve una copia profunda del menor operando. En caso de igualdad
// copia el receptor; no devuelve una vista sobre sus palabras internas.
func (r racionalExacto) minimo(otro racionalExacto) (racionalExacto, error) {
	if err := r.comprobarOperacion(otro); err != nil {
		return racionalExacto{}, err
	}
	izquierda := new(big.Int).Mul(&r.numerador, &otro.denominador)
	derecha := new(big.Int).Mul(&otro.numerador, &r.denominador)
	if izquierda.Cmp(derecha) <= 0 {
		return r.copiarSinConsumir(), nil
	}
	return otro.copiarSinConsumir(), nil
}

// copiar crea componentes independientes y conserva el presupuesto compartido
// del calculo. La copia tampoco expone big.Int fuera del paquete.
func (r racionalExacto) copiar() (racionalExacto, error) {
	if err := r.comprobarOperacionUnaria(); err != nil {
		return racionalExacto{}, err
	}
	return r.copiarSinConsumir(), nil
}

func (r racionalExacto) copiarSinConsumir() racionalExacto {
	var copia racionalExacto
	copia.numerador.Set(&r.numerador)
	copia.denominador.Set(&r.denominador)
	copia.contador = r.contador
	return copia
}

func (r racionalExacto) truncar() (racionalExacto, error) {
	if err := r.comprobarOperacionUnaria(); err != nil {
		return racionalExacto{}, err
	}
	cociente := new(big.Int).Quo(&r.numerador, &r.denominador)
	return construirRacionalExacto(r.contador, cociente, big.NewInt(1))
}

func (r racionalExacto) resto() (racionalExacto, error) {
	if err := r.comprobarOperacionUnaria(); err != nil {
		return racionalExacto{}, err
	}
	residuo := new(big.Int).Rem(&r.numerador, &r.denominador)
	return construirRacionalExacto(r.contador, residuo, &r.denominador)
}

// redondearAPuntos interpreta el racional como una cantidad exacta de
// micropuntos, no de puntos, y la convierte al tipo administrativo acotado.
func (r racionalExacto) redondearAPuntos(modo baremacion.ModoRedondeo) (baremacion.Puntos, error) {
	if err := r.comprobarOperacionUnaria(); err != nil {
		return baremacion.Puntos{}, err
	}
	if !modo.EsValido() {
		return baremacion.Puntos{}, nuevoError("modo_redondeo", CodigoModoRedondeoInvalido)
	}
	cociente := new(big.Int)
	residuo := new(big.Int)
	cociente.QuoRem(&r.numerador, &r.denominador, residuo)

	incrementar := false
	switch modo {
	case baremacion.RedondeoExacto:
		if residuo.Sign() != 0 {
			return baremacion.Puntos{}, nuevoError("puntos", CodigoResultadoNoExacto)
		}
	case baremacion.RedondeoTruncar:
	case baremacion.RedondeoHaciaArriba:
		incrementar = residuo.Sign() != 0
	case baremacion.RedondeoMitadArriba:
		incrementar = new(big.Int).Lsh(new(big.Int).Set(residuo), 1).Cmp(&r.denominador) >= 0
	case baremacion.RedondeoMitadAlPar:
		comparacion := new(big.Int).Lsh(new(big.Int).Set(residuo), 1).Cmp(&r.denominador)
		incrementar = comparacion > 0 || (comparacion == 0 && cociente.Bit(0) != 0)
	}
	if incrementar {
		cociente.Add(cociente, big.NewInt(1))
	}
	if !cociente.IsInt64() || cociente.Sign() < 0 ||
		cociente.Cmp(big.NewInt(baremacion.MaximoMicropuntos)) > 0 {
		return baremacion.Puntos{}, nuevoError("puntos", CodigoDesbordamiento)
	}
	resultado, err := baremacion.PuntosDesdeMicropuntos(cociente.Int64())
	if err != nil {
		return baremacion.Puntos{}, nuevoError("puntos", CodigoDesbordamiento)
	}
	return resultado, nil
}
