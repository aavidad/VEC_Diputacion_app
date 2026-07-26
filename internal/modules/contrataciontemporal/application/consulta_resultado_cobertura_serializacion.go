package application

import (
	"encoding/xml"
	"errors"
)

var ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida = errors.New(
	"contratacion temporal: serializacion de solicitud de consulta de resultado de cobertura prohibida",
)

func (SolicitudConsultaResultadoCobertura) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (*SolicitudConsultaResultadoCobertura) UnmarshalJSON([]byte) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (SolicitudConsultaResultadoCobertura) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (*SolicitudConsultaResultadoCobertura) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (SolicitudConsultaResultadoCobertura) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (*SolicitudConsultaResultadoCobertura) UnmarshalText([]byte) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (SolicitudConsultaResultadoCobertura) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (*SolicitudConsultaResultadoCobertura) UnmarshalBinary([]byte) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (SolicitudConsultaResultadoCobertura) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (*SolicitudConsultaResultadoCobertura) GobDecode([]byte) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (SolicitudConsultaResultadoCobertura) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (*SolicitudConsultaResultadoCobertura) UnmarshalCBOR([]byte) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (SolicitudConsultaResultadoCobertura) MarshalYAML() (any, error) {
	return nil, ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}

func (*SolicitudConsultaResultadoCobertura) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionSolicitudConsultaResultadoCoberturaProhibida
}
