package ports

import (
	"encoding/xml"
	"errors"
)

var ErrSerializacionConsultaResultadoCoberturaProhibida = errors.New(
	"contratacion temporal: serializacion de capacidad de consulta de resultado de cobertura prohibida",
)

type bloqueoSerializacionConsultaResultadoCobertura struct{}

func (bloqueoSerializacionConsultaResultadoCobertura) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (*bloqueoSerializacionConsultaResultadoCobertura) UnmarshalJSON([]byte) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (bloqueoSerializacionConsultaResultadoCobertura) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (*bloqueoSerializacionConsultaResultadoCobertura) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (bloqueoSerializacionConsultaResultadoCobertura) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (*bloqueoSerializacionConsultaResultadoCobertura) UnmarshalText([]byte) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (bloqueoSerializacionConsultaResultadoCobertura) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (*bloqueoSerializacionConsultaResultadoCobertura) UnmarshalBinary([]byte) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (bloqueoSerializacionConsultaResultadoCobertura) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (*bloqueoSerializacionConsultaResultadoCobertura) GobDecode([]byte) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (bloqueoSerializacionConsultaResultadoCobertura) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (*bloqueoSerializacionConsultaResultadoCobertura) UnmarshalCBOR([]byte) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (bloqueoSerializacionConsultaResultadoCobertura) MarshalYAML() (any, error) {
	return nil, ErrSerializacionConsultaResultadoCoberturaProhibida
}

func (*bloqueoSerializacionConsultaResultadoCobertura) UnmarshalYAML(
	func(any) error,
) error {
	return ErrSerializacionConsultaResultadoCoberturaProhibida
}
