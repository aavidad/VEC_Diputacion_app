package ports

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

// bloqueoSerializacionGobiernoConvocatoria se embebe en valores internos que
// nunca deben atravesar una frontera de transporte. Sus metodos promovidos
// cierran codificacion y decodificacion por los contratos estandar de Go.
// MaterialIntencionGobiernoConvocatoria no lo usa: su forma canonica es la
// preimagen deliberadamente serializable de autorizacion e idempotencia.
type bloqueoSerializacionGobiernoConvocatoria struct{}

func (bloqueoSerializacionGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}

func (*bloqueoSerializacionGobiernoConvocatoria) UnmarshalJSON([]byte) error {
	return ErrSerializacionGobiernoConvocatoriaProhibida
}

func (bloqueoSerializacionGobiernoConvocatoria) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}

func (*bloqueoSerializacionGobiernoConvocatoria) UnmarshalText([]byte) error {
	return ErrSerializacionGobiernoConvocatoriaProhibida
}

func (bloqueoSerializacionGobiernoConvocatoria) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}

func (*bloqueoSerializacionGobiernoConvocatoria) UnmarshalBinary([]byte) error {
	return ErrSerializacionGobiernoConvocatoriaProhibida
}

func (bloqueoSerializacionGobiernoConvocatoria) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}

func (*bloqueoSerializacionGobiernoConvocatoria) GobDecode([]byte) error {
	return ErrSerializacionGobiernoConvocatoriaProhibida
}

func (bloqueoSerializacionGobiernoConvocatoria) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionGobiernoConvocatoriaProhibida
}

func (*bloqueoSerializacionGobiernoConvocatoria) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionGobiernoConvocatoriaProhibida
}

func (bloqueoSerializacionGobiernoConvocatoria) String() string {
	return "[VALOR-GOBIERNO-CONVOCATORIA-INTERNO]"
}

func (b bloqueoSerializacionGobiernoConvocatoria) GoString() string { return b.String() }

func (b bloqueoSerializacionGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}

func (b bloqueoSerializacionGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

func referenciasGobiernoConvocatoriaDistintas(valores ...string) bool {
	vistas := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		if _, repetida := vistas[valor]; repetida {
			return false
		}
		vistas[valor] = struct{}{}
	}
	return true
}
