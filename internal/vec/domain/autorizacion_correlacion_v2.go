package domain

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
)

var (
	ErrReferenciaCorrelacionAutorizacionV2Invalida = errors.New(
		"vec: referencia de correlacion de autorizacion V2 invalida",
	)
	ErrGeneracionReferenciaCorrelacionAutorizacionV2 = errors.New(
		"vec: no se pudo generar la referencia de correlacion de autorizacion V2",
	)
	ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida = errors.New(
		"vec: serializacion de referencia de correlacion de autorizacion V2 prohibida",
	)
)

// generadorReferenciaCorrelacionAutorizacionV2 es el puerto minimo que la
// fabrica necesita. Su forma coincide estructuralmente con el puerto de
// composicion sin acoplar el dominio a la capa ports.
type generadorReferenciaCorrelacionAutorizacionV2 interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

// ReferenciaCorrelacionAutorizacionV2 es una capacidad nominal opaca. Su valor
// cero es invalido y no existe un constructor publico que acepte texto.
type ReferenciaCorrelacionAutorizacionV2 struct {
	bloqueoSerializacionReferenciaCorrelacionAutorizacionV2
	valor string
}

// GenerarReferenciaCorrelacionAutorizacionV2 acuna una referencia una sola vez
// mediante el puerto CSPRNG confiable y solo despues encapsula su valor. El
// llamador debe reutilizar la capacidad resultante durante toda la operacion.
func GenerarReferenciaCorrelacionAutorizacionV2(
	ctx context.Context,
	generador generadorReferenciaCorrelacionAutorizacionV2,
) (ReferenciaCorrelacionAutorizacionV2, error) {
	vacia := ReferenciaCorrelacionAutorizacionV2{}
	if ctx == nil || generadorReferenciaCorrelacionAutorizacionV2Nulo(generador) {
		return vacia, errors.Join(
			ErrGeneracionReferenciaCorrelacionAutorizacionV2,
			ErrReferenciaCorrelacionAutorizacionV2Invalida,
		)
	}
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrGeneracionReferenciaCorrelacionAutorizacionV2, err)
	}
	valor, err := generador.NuevaReferenciaCorrelacionAutorizacionV2(ctx)
	if err != nil {
		return vacia, errors.Join(ErrGeneracionReferenciaCorrelacionAutorizacionV2, err)
	}
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrGeneracionReferenciaCorrelacionAutorizacionV2, err)
	}
	if !ReferenciaCorrelacionAutorizacionV2Valida(valor) {
		return vacia, errors.Join(
			ErrGeneracionReferenciaCorrelacionAutorizacionV2,
			ErrReferenciaCorrelacionAutorizacionV2Invalida,
		)
	}
	return ReferenciaCorrelacionAutorizacionV2{valor: valor}, nil
}

// ValorCanonico revela el identificador solo en las fronteras que deben
// comprometerlo en una decision, auditarlo o persistirlo.
func (r ReferenciaCorrelacionAutorizacionV2) ValorCanonico() (string, error) {
	if r.Validar() != nil {
		return "", ErrReferenciaCorrelacionAutorizacionV2Invalida
	}
	return r.valor, nil
}

// Validar permite comprobar la capacidad sin revelar su valor canonico.
func (r ReferenciaCorrelacionAutorizacionV2) Validar() error {
	if !ReferenciaCorrelacionAutorizacionV2Valida(r.valor) {
		return ErrReferenciaCorrelacionAutorizacionV2Invalida
	}
	return nil
}

func generadorReferenciaCorrelacionAutorizacionV2Nulo(
	generador generadorReferenciaCorrelacionAutorizacionV2,
) bool {
	if generador == nil {
		return true
	}
	valor := reflect.ValueOf(generador)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

type bloqueoSerializacionReferenciaCorrelacionAutorizacionV2 struct{}

func (bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (*bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (*bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) UnmarshalText([]byte) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (*bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (*bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) GobDecode([]byte) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (*bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) UnmarshalCBOR([]byte) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (*bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}
func (*bloqueoSerializacionReferenciaCorrelacionAutorizacionV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida
}

func (ReferenciaCorrelacionAutorizacionV2) String() string {
	return "[REFERENCIA-CORRELACION-AUTORIZACION-V2-OPACA]"
}
func (r ReferenciaCorrelacionAutorizacionV2) GoString() string { return r.String() }
func (r ReferenciaCorrelacionAutorizacionV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReferenciaCorrelacionAutorizacionV2) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
