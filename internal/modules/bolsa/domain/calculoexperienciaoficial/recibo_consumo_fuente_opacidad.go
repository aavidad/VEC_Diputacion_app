package calculoexperienciaoficial

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

const textoReciboConsumoFuenteOculto = "[RECIBO-CONSUMO-AUTORIZACION-FUENTE-OCULTO]"

func (ReciboConsumoAutorizacionFuenteV1) MarshalJSON() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV1) UnmarshalJSON([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV1) MarshalText() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV1) UnmarshalText([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV1) MarshalBinary() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV1) UnmarshalBinary([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV1) GobEncode() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV1) GobDecode([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV1) MarshalCBOR() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV1) UnmarshalCBOR([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV1) MarshalYAML() (any, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV1) UnmarshalYAML(func(any) error) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV1) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV1) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV1) String() string {
	return textoReciboConsumoFuenteOculto
}

func (r ReciboConsumoAutorizacionFuenteV1) GoString() string { return r.String() }

func (r ReciboConsumoAutorizacionFuenteV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

func (r ReciboConsumoAutorizacionFuenteV1) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
