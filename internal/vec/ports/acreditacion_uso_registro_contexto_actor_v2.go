package ports

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida = errors.New(
		"vec: orden de acreditacion de uso de registro de contexto de actor v2 invalida",
	)
	ErrAcreditacionUsoRegistroContextoActorV2Denegada = errors.New(
		"vec: acreditacion de uso de registro de contexto de actor v2 denegada",
	)
	ErrAcreditadorUsoRegistroContextoActorV2NoDisponible = errors.New(
		"vec: acreditador de uso de registro de contexto de actor v2 no disponible",
	)
	ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida = errors.New(
		"vec: serializacion de acreditacion de uso de registro de contexto de actor v2 prohibida",
	)
)

// DatosOrdenAcreditacionUsoRegistroContextoActorV2 es la entrega defensiva que
// necesita el adaptador transaccional. Incluye el recibo durable completo porque
// persona/perfil, sus versiones y los vinculos no caben en la proyeccion
// minimizada del vinculo de autenticacion-actor V2.
type DatosOrdenAcreditacionUsoRegistroContextoActorV2 struct {
	bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2
	Resultado   domain.ResultadoContextoActorRegistradoV2
	EmitidaEn   time.Time
	ValidaHasta time.Time
}

type datosOrdenAcreditacionUsoRegistroContextoActorV2 struct {
	resultado   domain.ResultadoContextoActorRegistradoV2
	emitidaEn   time.Time
	validaHasta time.Time
}

// OrdenAcreditacionUsoRegistroContextoActorV2 es una capacidad nominal de
// entrada. No acredita por si misma la existencia ni la vigencia del recibo: la
// respuesta positiva solo puede proceder del puerto transaccional.
type OrdenAcreditacionUsoRegistroContextoActorV2 struct {
	bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2
	datos *datosOrdenAcreditacionUsoRegistroContextoActorV2
}

func NuevaOrdenAcreditacionUsoRegistroContextoActorV2(
	resultado domain.ResultadoContextoActorRegistradoV2,
	emitidaEn time.Time,
	validaHasta time.Time,
) (OrdenAcreditacionUsoRegistroContextoActorV2, error) {
	datos := DatosOrdenAcreditacionUsoRegistroContextoActorV2{
		Resultado: resultado, EmitidaEn: emitidaEn, ValidaHasta: validaHasta,
	}
	clon, err := clonarDatosOrdenAcreditacionUsoRegistroContextoActorV2(datos)
	if err != nil {
		return OrdenAcreditacionUsoRegistroContextoActorV2{}, err
	}
	return OrdenAcreditacionUsoRegistroContextoActorV2{
		datos: &datosOrdenAcreditacionUsoRegistroContextoActorV2{
			resultado: clon.Resultado, emitidaEn: clon.EmitidaEn, validaHasta: clon.ValidaHasta,
		},
	}, nil
}

func (o OrdenAcreditacionUsoRegistroContextoActorV2) Datos() (
	DatosOrdenAcreditacionUsoRegistroContextoActorV2,
	error,
) {
	if o.datos == nil {
		return DatosOrdenAcreditacionUsoRegistroContextoActorV2{},
			ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida
	}
	return clonarDatosOrdenAcreditacionUsoRegistroContextoActorV2(
		DatosOrdenAcreditacionUsoRegistroContextoActorV2{
			Resultado: o.datos.resultado, EmitidaEn: o.datos.emitidaEn, ValidaHasta: o.datos.validaHasta,
		},
	)
}

func clonarDatosOrdenAcreditacionUsoRegistroContextoActorV2(
	datos DatosOrdenAcreditacionUsoRegistroContextoActorV2,
) (DatosOrdenAcreditacionUsoRegistroContextoActorV2, error) {
	if validarDatosOrdenAcreditacionUsoRegistroContextoActorV2(datos) != nil {
		return DatosOrdenAcreditacionUsoRegistroContextoActorV2{},
			ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida
	}
	resultado, err := datos.Resultado.Clonar()
	if err != nil {
		return DatosOrdenAcreditacionUsoRegistroContextoActorV2{},
			ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida
	}
	datos.Resultado = resultado
	return datos, nil
}

func validarDatosOrdenAcreditacionUsoRegistroContextoActorV2(
	datos DatosOrdenAcreditacionUsoRegistroContextoActorV2,
) error {
	if datos.Resultado.Validar() != nil ||
		!instanteAcreditacionUsoRegistroContextoActorV2Canonico(datos.EmitidaEn) ||
		!instanteAcreditacionUsoRegistroContextoActorV2Canonico(datos.ValidaHasta) ||
		!datos.ValidaHasta.After(datos.EmitidaEn) ||
		datos.EmitidaEn.Before(datos.Resultado.ResueltoEnAutoritativo) {
		return ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida
	}
	actor := datos.Resultado.Contexto.Instantanea
	if actor.Estado != domain.EstadoVinculoContextoActorActivo ||
		datos.EmitidaEn.Before(actor.VigenteDesde) || datos.ValidaHasta.After(actor.VigenteHasta) {
		return ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida
	}
	for _, vinculo := range actor.Vinculos {
		if vinculo.Estado != domain.EstadoVinculoContextoActorActivo ||
			datos.EmitidaEn.Before(vinculo.VigenteDesde) || datos.ValidaHasta.After(vinculo.VigenteHasta) {
			return ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida
		}
	}
	return nil
}

func instanteAcreditacionUsoRegistroContextoActorV2Canonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

// AcreditadorUsoRegistroContextoActorV2 es un puerto ligado a la transaccion
// del consumidor. La implementacion productiva no puede abrir, confirmar ni
// revertir una transaccion. El consumidor debe usar una unica transaccion
// SERIALIZABLE de escritura y puede invocar el puerto antes y despues de tomar
// sus propios locks; el segundo instante es la observacion autoritativa final.
type AcreditadorUsoRegistroContextoActorV2 interface {
	AcreditarUsoRegistroContextoActorV2(
		context.Context,
		OrdenAcreditacionUsoRegistroContextoActorV2,
	) (time.Time, error)
}

type bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2 struct{}

func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (*bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (*bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) UnmarshalText([]byte) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (*bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (*bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) GobDecode([]byte) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (*bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) UnmarshalCBOR([]byte) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (*bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (*bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida
}
func (bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) String() string {
	return "[ACREDITACION-USO-REGISTRO-CONTEXTO-ACTOR-V2-OPACA]"
}
func (b bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionAcreditacionUsoRegistroContextoActorV2) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
