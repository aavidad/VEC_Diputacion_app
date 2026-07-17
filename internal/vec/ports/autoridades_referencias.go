package ports

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	PrefijoReferenciaSolicitudFuenteAutoridad = "solicitud:fuente_autoridad:"
	PrefijoReferenciaOperacionFuenteAutoridad = "operacion:fuente_autoridad:"
	longitudMinimaSufijoReferenciaAutoridad   = 22
	longitudMaximaSufijoReferenciaAutoridad   = 128
)

var (
	ErrReferenciaGeneradaFuenteAutoridadInvalida = errors.New("vec: referencia generada de fuente de autoridad invalida")
	ErrGeneracionReferenciaFuenteAutoridad       = errors.New("vec: no se pudo generar una referencia de fuente de autoridad")
	ErrColisionReferenciaFuenteAutoridad         = errors.New("vec: colision de referencia de fuente de autoridad")
	ErrSerializacionReferenciaAutoridad          = errors.New("vec: serializacion de referencia de fuente de autoridad prohibida")
)

type ReferenciaSolicitudFuenteAutoridad struct {
	bloqueoSerializacionReferenciaAutoridad
	valor string
}

func NuevaReferenciaSolicitudFuenteAutoridad(valor string) (ReferenciaSolicitudFuenteAutoridad, error) {
	if !referenciaGeneradaAutoridadValida(valor, PrefijoReferenciaSolicitudFuenteAutoridad) {
		return ReferenciaSolicitudFuenteAutoridad{}, ErrReferenciaGeneradaFuenteAutoridadInvalida
	}
	return ReferenciaSolicitudFuenteAutoridad{valor: valor}, nil
}

func (r ReferenciaSolicitudFuenteAutoridad) Referencia() (string, error) {
	if !referenciaGeneradaAutoridadValida(r.valor, PrefijoReferenciaSolicitudFuenteAutoridad) {
		return "", ErrReferenciaGeneradaFuenteAutoridadInvalida
	}
	return r.valor, nil
}

func (ReferenciaSolicitudFuenteAutoridad) String() string {
	return "[REFERENCIA-SOLICITUD-FUENTE-AUTORIDAD-OPACA]"
}
func (r ReferenciaSolicitudFuenteAutoridad) GoString() string { return r.String() }
func (r ReferenciaSolicitudFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReferenciaSolicitudFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

type ReferenciaOperacionFuenteAutoridad struct {
	bloqueoSerializacionReferenciaAutoridad
	valor string
}

func NuevaReferenciaOperacionFuenteAutoridad(valor string) (ReferenciaOperacionFuenteAutoridad, error) {
	if !referenciaGeneradaAutoridadValida(valor, PrefijoReferenciaOperacionFuenteAutoridad) {
		return ReferenciaOperacionFuenteAutoridad{}, ErrReferenciaGeneradaFuenteAutoridadInvalida
	}
	return ReferenciaOperacionFuenteAutoridad{valor: valor}, nil
}

func (r ReferenciaOperacionFuenteAutoridad) Referencia() (string, error) {
	if !referenciaGeneradaAutoridadValida(r.valor, PrefijoReferenciaOperacionFuenteAutoridad) {
		return "", ErrReferenciaGeneradaFuenteAutoridadInvalida
	}
	return r.valor, nil
}

func (ReferenciaOperacionFuenteAutoridad) String() string {
	return "[REFERENCIA-OPERACION-FUENTE-AUTORIDAD-OPACA]"
}
func (r ReferenciaOperacionFuenteAutoridad) GoString() string { return r.String() }
func (r ReferenciaOperacionFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReferenciaOperacionFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

// GeneradorReferenciasFuentesAutoridad produce namespaces distintos. La forma
// no acredita entropia: el adaptador debe obtener al menos 128 bits aleatorios
// con un CSPRNG y codificarlos, por ejemplo, como 22 caracteres base64url sin
// relleno o 32 hexadecimales. La barrera durable reserva ademas la unicidad y
// puede devolver ErrColisionReferenciaFuenteAutoridad.
type GeneradorReferenciasFuentesAutoridad interface {
	NuevaReferenciaSolicitud(context.Context) (ReferenciaSolicitudFuenteAutoridad, error)
	NuevaReferenciaOperacion(context.Context) (ReferenciaOperacionFuenteAutoridad, error)
}

func referenciaGeneradaAutoridadValida(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) {
		return false
	}
	sufijo := strings.TrimPrefix(valor, prefijo)
	if len(sufijo) < longitudMinimaSufijoReferenciaAutoridad ||
		len(sufijo) > longitudMaximaSufijoReferenciaAutoridad {
		return false
	}
	for indice := 0; indice < len(sufijo); indice++ {
		caracter := sufijo[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '-' || caracter == '_' {
			continue
		}
		return false
	}
	return referenciaPuertoAutoridadValida(valor)
}

type bloqueoSerializacionReferenciaAutoridad struct{}

func (bloqueoSerializacionReferenciaAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReferenciaAutoridad
}
func (*bloqueoSerializacionReferenciaAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionReferenciaAutoridad
}
func (bloqueoSerializacionReferenciaAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReferenciaAutoridad
}
func (*bloqueoSerializacionReferenciaAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionReferenciaAutoridad
}
func (bloqueoSerializacionReferenciaAutoridad) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionReferenciaAutoridad
}
func (*bloqueoSerializacionReferenciaAutoridad) UnmarshalBinary([]byte) error {
	return ErrSerializacionReferenciaAutoridad
}
func (bloqueoSerializacionReferenciaAutoridad) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionReferenciaAutoridad
}
func (*bloqueoSerializacionReferenciaAutoridad) GobDecode([]byte) error {
	return ErrSerializacionReferenciaAutoridad
}
func (bloqueoSerializacionReferenciaAutoridad) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionReferenciaAutoridad
}
func (*bloqueoSerializacionReferenciaAutoridad) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionReferenciaAutoridad
}
