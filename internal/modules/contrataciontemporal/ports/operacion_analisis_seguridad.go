package ports

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

const (
	// MaximoEnteroSeguroOperacionAnalisis evita que una versión o un entero
	// canónico cambien al cruzar clientes que representan números como IEEE-754.
	MaximoEnteroSeguroOperacionAnalisis uint64 = 1<<53 - 1
	TiempoMaximoOperacionAnalisis              = 15 * time.Second
)

var ErrSerializacionOperacionAnalisisProhibida = errors.New(
	"contratacion temporal: serializacion de operacion de analisis prohibida",
)

func VersionOperacionAnalisisConIncrementoValida(version uint64) bool {
	return version > 0 && version < MaximoEnteroSeguroOperacionAnalisis
}

func VersionOperacionAnalisisValida(version uint64) bool {
	return version > 0 && version <= MaximoEnteroSeguroOperacionAnalisis
}

func enteroCanonicoOperacionAnalisisValido(valor int64) bool {
	if valor >= 0 {
		return uint64(valor) <= MaximoEnteroSeguroOperacionAnalisis
	}
	return valor >= -int64(MaximoEnteroSeguroOperacionAnalisis)
}

type bloqueoSerializacionOperacionAnalisis struct{}

func (bloqueoSerializacionOperacionAnalisis) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOperacionAnalisisProhibida
}
func (*bloqueoSerializacionOperacionAnalisis) UnmarshalJSON([]byte) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (bloqueoSerializacionOperacionAnalisis) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (*bloqueoSerializacionOperacionAnalisis) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (bloqueoSerializacionOperacionAnalisis) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionOperacionAnalisisProhibida
}
func (*bloqueoSerializacionOperacionAnalisis) UnmarshalText([]byte) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (bloqueoSerializacionOperacionAnalisis) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionOperacionAnalisisProhibida
}
func (*bloqueoSerializacionOperacionAnalisis) UnmarshalBinary([]byte) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (bloqueoSerializacionOperacionAnalisis) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionOperacionAnalisisProhibida
}
func (*bloqueoSerializacionOperacionAnalisis) GobDecode([]byte) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (bloqueoSerializacionOperacionAnalisis) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionOperacionAnalisisProhibida
}
func (*bloqueoSerializacionOperacionAnalisis) UnmarshalCBOR([]byte) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (bloqueoSerializacionOperacionAnalisis) MarshalYAML() (any, error) {
	return nil, ErrSerializacionOperacionAnalisisProhibida
}
func (*bloqueoSerializacionOperacionAnalisis) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionOperacionAnalisisProhibida
}
func (bloqueoSerializacionOperacionAnalisis) String() string {
	return "[OPERACION-ANALISIS-OPACA]"
}
func (b bloqueoSerializacionOperacionAnalisis) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionOperacionAnalisis) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionOperacionAnalisis) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
