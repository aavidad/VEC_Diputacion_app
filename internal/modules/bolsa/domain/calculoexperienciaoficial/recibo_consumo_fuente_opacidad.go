package calculoexperienciaoficial

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

const textoReciboConsumoFuenteOculto = "[RECIBO-CONSUMO-AUTORIZACION-FUENTE-OCULTO]"

func (ReciboConsumoAutorizacionFuenteV2) MarshalJSON() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV2) UnmarshalJSON([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV2) MarshalText() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV2) UnmarshalText([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV2) MarshalBinary() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV2) UnmarshalBinary([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV2) GobEncode() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV2) GobDecode([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV2) UnmarshalCBOR([]byte) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV2) MarshalYAML() (any, error) {
	return nil, ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV2) UnmarshalYAML(func(any) error) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV2) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrEntradaNoPermitida
}

func (*ReciboConsumoAutorizacionFuenteV2) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrEntradaNoPermitida
}

func (ReciboConsumoAutorizacionFuenteV2) String() string {
	return textoReciboConsumoFuenteOculto
}

func (r ReciboConsumoAutorizacionFuenteV2) GoString() string { return r.String() }

func (r ReciboConsumoAutorizacionFuenteV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

func (r ReciboConsumoAutorizacionFuenteV2) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
