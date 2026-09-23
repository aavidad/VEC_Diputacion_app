// Package domain contiene las reglas puras de Dietas.
package domain

import (
	"errors"
	"regexp"
	"time"
)

var ErrComisionBorradorInvalida = errors.New("dietas: comision borrador invalida")

var (
	referenciaComision = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)
	codigoRuta         = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,64}$`)
)

// ComisionBorrador es una declaración de comisión. No contiene cuantías,
// kilometraje reconocido, tarifa, validación, liquidación ni pago.
type ComisionBorrador struct {
	Referencia  string           `json:"referencia"`
	Estado      string           `json:"estado"`
	FechaInicio string           `json:"fecha_inicio"`
	FechaFin    string           `json:"fecha_fin"`
	Motivo      string           `json:"motivo"`
	CodigosRuta []string         `json:"codigos_ruta"`
	RelacionRef string           `json:"relacion_ref"`
	Calculo     *CalculoComision `json:"calculo,omitempty"`
}

func (c ComisionBorrador) Validar() error {
	if !referenciaComision.MatchString(c.Referencia) || c.Estado != "borrador" ||
		!fechaCivilValida(c.FechaInicio) || !fechaCivilValida(c.FechaFin) ||
		c.FechaInicio > c.FechaFin || len(c.Motivo) < 3 || len(c.Motivo) > 600 ||
		!textoVisible(c.Motivo) || !referenciaRelacionValida(c.RelacionRef) ||
		len(c.CodigosRuta) > 16 {
		return ErrComisionBorradorInvalida
	}
	vistos := map[string]bool{}
	for _, codigo := range c.CodigosRuta {
		if !codigoRuta.MatchString(codigo) || vistos[codigo] {
			return ErrComisionBorradorInvalida
		}
		vistos[codigo] = true
	}
	if c.Calculo != nil && c.Calculo.Validar(c.CodigosRuta) != nil {
		return ErrComisionBorradorInvalida
	}
	return nil
}

func NuevaComisionBorrador(referencia, inicio, fin, motivo, relacion string, codigos []string) (ComisionBorrador, error) {
	// Una ruta vacía es un dato explícito del contrato, no la ausencia del
	// campo. Así se conserva la misma preimagen y la misma respuesta en una
	// recuperación idempotente.
	copia := append([]string{}, codigos...)
	c := ComisionBorrador{Referencia: referencia, Estado: "borrador", FechaInicio: inicio, FechaFin: fin, Motivo: motivo, CodigosRuta: copia, RelacionRef: relacion}
	if c.Validar() != nil {
		return ComisionBorrador{}, ErrComisionBorradorInvalida
	}
	return c, nil
}

func fechaCivilValida(valor string) bool {
	if len(valor) != len("2006-01-02") {
		return false
	}
	_, err := time.Parse("2006-01-02", valor)
	return err == nil
}

func referenciaRelacionValida(valor string) bool { return referenciaOpacaValida(valor, "rel_") }

func referenciaOpacaValida(valor, prefijo string) bool {
	if len(valor) < len(prefijo)+22 || len(valor) > len(prefijo)+128 || len(valor) <= len(prefijo) || valor[:len(prefijo)] != prefijo {
		return false
	}
	for _, r := range valor[len(prefijo):] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func textoVisible(valor string) bool {
	for _, r := range valor {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
