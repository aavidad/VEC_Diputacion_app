package domain

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var (
	// ErrParseoAtestacionDenegacionAutorizacionV1Invalido identifica un
	// VEC-AD-D-1 no exacto. La proyeccion resultante sigue sin demostrar firma,
	// procedencia ni autoridad para mutar estado.
	ErrParseoAtestacionDenegacionAutorizacionV1Invalido = errors.New("vec: parseo no autoritativo VEC-AD-D-1 invalido")

	// ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
	// bloquea todos los codecs generales sobre la proyeccion negativa.
	ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida = errors.New("vec: serializacion de proyeccion no autoritativa VEC-AD-D-1 prohibida")
)

const representacionRedactadaProyeccionAtestacionDenegacionAutorizacionV1 = "[PROYECCION-VEC-AD-D-1-NOMINAL-NO-AUTORITATIVA-REDACTADA]"

// ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa solo acredita la
// forma canonica de una evidencia negativa. Conserva los datos completos en
// campos privados exclusivamente para validar cruces y nunca los expone.
type ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa struct {
	cabecera CabeceraAtestacionDenegacionAutorizacionV1
	datos    *datosDecisionAtestacionAutorizacionV2NoAutoritativos
	motivo   *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos
}

// ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo exige el
// dominio VEC-AD-D-1, una decision negativa V2 completa y reserializacion exacta.
func ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(
	contenido []byte,
) (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa, error) {
	if len(contenido) == 0 || len(contenido) > TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 ||
		!comprobarEsquemaDecisionAtestacionAutorizacionV2() ||
		!limitesEscritorAtestacionDenegacionAutorizacionV1Compatibles(
			TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
			TamanoMaximoMensajeAtestacionAutorizacionV1,
		) {
		return ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa{}, errorParseoAtestacionDenegacionAutorizacionV1()
	}

	cabeceraCruda, datos, motivo, err := parsearMensajeAtestacionSolicitudLigadaNoAutoritativo(
		contenido,
		EsquemaMensajeAtestacionDenegacionAutorizacionV1,
	)
	cabecera := CabeceraAtestacionDenegacionAutorizacionV1{
		FormatoVersion: cabeceraCruda.formatoVersion,
		Suite:          cabeceraCruda.suite,
		ClaveID:        cabeceraCruda.claveID,
		Audiencia:      cabeceraCruda.audiencia,
	}
	if err != nil || cabecera.Validar() != nil ||
		validarDatosAtestacionSolicitudLigadaNoAutoritativos(datos, motivo, false) != nil {
		return ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa{}, errorParseoAtestacionDenegacionAutorizacionV1()
	}

	canonico, err := serializarMensajeAtestacionSolicitudLigadaNoAutoritativo(
		EsquemaMensajeAtestacionDenegacionAutorizacionV1,
		cabecera.FormatoVersion,
		cabecera.Suite,
		cabecera.ClaveID,
		cabecera.Audiencia,
		datos,
		motivo,
		TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
	)
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa{}, errorParseoAtestacionDenegacionAutorizacionV1()
	}

	copiaDatos := clonarDatosAtestacionSolicitudLigadaNoAutoritativos(datos)
	copiaMotivo := motivo
	return ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa{
		cabecera: cabecera,
		datos:    &copiaDatos,
		motivo:   &copiaMotivo,
	}, nil
}

// Cabecera devuelve configuracion nominal; no selecciona una clave confiable.
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) Cabecera() (
	CabeceraAtestacionDenegacionAutorizacionV1,
	error,
) {
	if p.validar() != nil {
		return CabeceraAtestacionDenegacionAutorizacionV1{}, errorParseoAtestacionDenegacionAutorizacionV1()
	}
	return p.cabecera, nil
}

// DecisionRef devuelve un identificador nominal y no una denegacion firmada.
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) DecisionRef() (string, error) {
	if p.validar() != nil {
		return "", errorParseoAtestacionDenegacionAutorizacionV1()
	}
	return p.datos.DecisionRef, nil
}

// SolicitudHuellaSHA256 devuelve el compromiso nominal de solicitud.
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) SolicitudHuellaSHA256() (string, error) {
	if p.validar() != nil {
		return "", errorParseoAtestacionDenegacionAutorizacionV1()
	}
	return p.datos.SolicitudHuellaSHA256, nil
}

// MotivoHuellaSHA256 devuelve el compromiso nominal sin revelar coordenadas.
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MotivoHuellaSHA256() (string, error) {
	if p.validar() != nil {
		return "", errorParseoAtestacionDenegacionAutorizacionV1()
	}
	return p.datos.MotivoHuellaSHA256, nil
}

func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) validar() error {
	if p.datos == nil || p.motivo == nil || p.cabecera.Validar() != nil ||
		validarDatosAtestacionSolicitudLigadaNoAutoritativos(*p.datos, *p.motivo, false) != nil {
		return errorParseoAtestacionDenegacionAutorizacionV1()
	}
	return nil
}

func errorParseoAtestacionDenegacionAutorizacionV1() error {
	return errors.Join(
		ErrParseoAtestacionDenegacionAutorizacionV1Invalido,
		ErrMensajeAtestacionAutorizacionInvalido,
	)
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) String() string {
	return representacionRedactadaProyeccionAtestacionDenegacionAutorizacionV1
}

func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) GoString() string {
	return p.String()
}

func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalJSON([]byte) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalText([]byte) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalBinary([]byte) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) GobDecode([]byte) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalCBOR([]byte) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalYAML() (any, error) {
	return nil, ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
}
