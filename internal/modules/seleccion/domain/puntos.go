// Package domain contiene las reglas de Selección: convocatoria publicada
// (plazo, turnos, requisitos y baremo), solicitud de participación y
// autobaremación. No conoce puertos, HTTP, SQL ni proveedores.
package domain

import (
	"errors"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/shared/baremacion"
)

// decimalesPuntos es la precisión de las puntuaciones que se muestran; los
// cálculos usan micropuntos exactos.
const decimalesPuntos = 6

// ParsearPuntos convierte una cadena decimal no negativa («1.4», «20») en
// puntos exactos. Admite hasta seis decimales.
func ParsearPuntos(texto string) (baremacion.Puntos, error) {
	numerador, escala, err := decimalCanonico(texto, decimalesPuntos)
	if err != nil {
		return baremacion.Puntos{}, errors.Join(ErrConvocatoriaInvalida, err)
	}
	for escala < decimalesPuntos {
		if numerador > baremacion.MaximoMicropuntos/10 {
			return baremacion.Puntos{}, ErrConvocatoriaInvalida
		}
		numerador *= 10
		escala++
	}
	return baremacion.PuntosDesdeMicropuntos(numerador)
}

// FormatearPuntos devuelve la cadena decimal mínima («1.4», «0», «12.25»).
func FormatearPuntos(p baremacion.Puntos) string {
	m := p.Micropuntos()
	entero := m / baremacion.MicropuntosPorPunto
	resto := m % baremacion.MicropuntosPorPunto
	if resto == 0 {
		return strconv.FormatInt(entero, 10)
	}
	fraccion := strings.TrimRight(strconv.FormatInt(resto+baremacion.MicropuntosPorPunto, 10)[1:], "0")
	return strconv.FormatInt(entero, 10) + "." + fraccion
}

// ParsearCantidad lee la cantidad declarada de un mérito («14», «2.5»): no
// negativa, hasta tres decimales y como máximo un millón de unidades.
func ParsearCantidad(texto string) (baremacion.Racional, error) {
	numerador, escala, err := decimalCanonico(texto, 3)
	if err != nil {
		return baremacion.Racional{}, errors.Join(ErrSolicitudInvalida, err)
	}
	if numerador > 1_000_000_000 {
		return baremacion.Racional{}, ErrSolicitudInvalida
	}
	denominador := int64(1)
	for range escala {
		denominador *= 10
	}
	if numerador/denominador > 1_000_000 {
		return baremacion.Racional{}, ErrSolicitudInvalida
	}
	return baremacion.NuevoRacional(numerador, denominador)
}

// decimalCanonico acepta «123» o «123.45» sin signo, sin ceros a la
// izquierda superfluos ni coma decimal. Devuelve el entero escalado.
func decimalCanonico(texto string, maximoDecimales int) (int64, int, error) {
	if texto == "" || len(texto) > 24 {
		return 0, 0, errDecimalNoCanonico
	}
	entero, fraccion, conPunto := strings.Cut(texto, ".")
	if entero == "" || (len(entero) > 1 && entero[0] == '0') || (conPunto && (fraccion == "" || len(fraccion) > maximoDecimales)) {
		return 0, 0, errDecimalNoCanonico
	}
	for _, r := range entero + fraccion {
		if r < '0' || r > '9' {
			return 0, 0, errDecimalNoCanonico
		}
	}
	valor, err := strconv.ParseInt(entero+fraccion, 10, 64)
	if err != nil {
		return 0, 0, errors.Join(errDecimalNoCanonico, err)
	}
	return valor, len(fraccion), nil
}

var errDecimalNoCanonico = errors.New("seleccion: decimal no canonico")
