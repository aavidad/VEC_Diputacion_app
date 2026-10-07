// Package registrocopias coordinates the offline journal, without permission decisions.
package registrocopias

import (
	"context"
	"regexp"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	registro "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

var ref = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,95}$`)

type Servicio struct{ Registro registro.Registro }

func ValidarDeclaracion(d registro.Declaracion) error {
	if !ref.MatchString(d.Actor) || !ref.MatchString(d.Correlacion) {
		return registro.ErrEntrada
	}
	return nil
}

func (s Servicio) Reservar(ctx context.Context, d registro.Declaracion, solicitud operacionescopias.Solicitud) (registro.Resultado, error) {
	if err := s.validar(d); err != nil {
		return registro.Resultado{}, err
	}
	return s.Registro.Reservar(ctx, d, solicitud)
}
func (s Servicio) Aplicar(ctx context.Context, d registro.Declaracion, op string, c operacionescopias.Comando) (registro.Resultado, error) {
	if err := s.validar(d); err != nil {
		return registro.Resultado{}, err
	}
	if !ref.MatchString(op) {
		return registro.Resultado{}, registro.ErrEntrada
	}
	return s.Registro.Aplicar(ctx, d, op, c)
}
func (s Servicio) Consultar(ctx context.Context, d registro.Declaracion, op string) (registro.Resultado, error) {
	if err := s.validar(d); err != nil {
		return registro.Resultado{}, err
	}
	if !ref.MatchString(op) {
		return registro.Resultado{}, registro.ErrEntrada
	}
	return s.Registro.Consultar(ctx, d, op)
}
func (s Servicio) validar(d registro.Declaracion) error {
	if s.Registro == nil {
		return registro.ErrConfiguracion
	}
	return ValidarDeclaracion(d)
}

func (s Servicio) Listar(ctx context.Context, d registro.Declaracion, q registro.Consulta) (registro.Resultado, error) {
	if err := s.validar(d); err != nil {
		return registro.Resultado{}, err
	}
	return s.Registro.Listar(ctx, d, q)
}
