package ports

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida = errors.New(
	"vec: orden de registro de autorizacion ligada a solicitud invalida",
)

// DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2 es la copia
// defensiva que recibe el adaptador durable. La referencia permite releer en
// su propia transaccion el catalogo exacto y cotejar version, huella y entrada.
type DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2 struct {
	bloqueoSerializacionOrdenRegistroAutorizacionV2
	Decision         domain.DecisionAutorizacion
	ReferenciaMotivo domain.ReferenciaEntradaCatalogo
}

type datosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2 struct {
	Decision         domain.DecisionAutorizacion
	ReferenciaMotivo domain.ReferenciaEntradaCatalogo
}

// OrdenRegistroDecisionAutorizacionSolicitudLigadaV2 es una orden opaca y
// nominal. Evita que un registro V2 reciba solo la decision y pierda la
// preimagen catalogada necesaria para la revalidacion durable.
type OrdenRegistroDecisionAutorizacionSolicitudLigadaV2 struct {
	bloqueoSerializacionOrdenRegistroAutorizacionV2
	datos *datosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2
}

func NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) (OrdenRegistroDecisionAutorizacionSolicitudLigadaV2, error) {
	if err := validarOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
		decision,
		referenciaMotivo,
	); err != nil {
		return OrdenRegistroDecisionAutorizacionSolicitudLigadaV2{}, err
	}
	return OrdenRegistroDecisionAutorizacionSolicitudLigadaV2{
		datos: &datosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2{
			Decision:         clonarDecisionAutorizacionCanonica(decision),
			ReferenciaMotivo: referenciaMotivo,
		},
	}, nil
}

func (o OrdenRegistroDecisionAutorizacionSolicitudLigadaV2) Datos() (
	DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
	error,
) {
	if o.datos == nil || validarOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
		o.datos.Decision,
		o.datos.ReferenciaMotivo,
	) != nil {
		return DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2{},
			ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida
	}
	return DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2{
		Decision:         clonarDecisionAutorizacionCanonica(o.datos.Decision),
		ReferenciaMotivo: o.datos.ReferenciaMotivo,
	}, nil
}

func validarOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) error {
	huella, err := domain.HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if err != nil || !domain.ReferenciaMotivoAutorizacionV2Valida(referenciaMotivo) ||
		decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil ||
		decision.EsquemaHuellaMotivo != domain.EsquemaHuellaMotivoAutorizacionV2 ||
		decision.MotivoHuellaSHA256 != huella {
		return ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida
	}
	return nil
}

// bloqueoSerializacionOrdenRegistroAutorizacionV2 impide que orden o datos
// internos se filtren por codecs o formateo. Datos es una entrega deliberada
// al adaptador, no un DTO de transporte.
type bloqueoSerializacionOrdenRegistroAutorizacionV2 struct{}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*bloqueoSerializacionOrdenRegistroAutorizacionV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*bloqueoSerializacionOrdenRegistroAutorizacionV2) UnmarshalText([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*bloqueoSerializacionOrdenRegistroAutorizacionV2) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*bloqueoSerializacionOrdenRegistroAutorizacionV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*bloqueoSerializacionOrdenRegistroAutorizacionV2) GobDecode([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*bloqueoSerializacionOrdenRegistroAutorizacionV2) UnmarshalCBOR([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*bloqueoSerializacionOrdenRegistroAutorizacionV2) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (bloqueoSerializacionOrdenRegistroAutorizacionV2) String() string {
	return "[ORDEN-REGISTRO-AUTORIZACION-SOLICITUD-LIGADA-V2]"
}

func (b bloqueoSerializacionOrdenRegistroAutorizacionV2) GoString() string { return b.String() }

func (b bloqueoSerializacionOrdenRegistroAutorizacionV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}

func (b bloqueoSerializacionOrdenRegistroAutorizacionV2) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
