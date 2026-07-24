package domain

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var (
	ErrSolicitudAutorizacionLigadaV3Invalida               = errors.New("vec: solicitud de autorizacion ligada V3 invalida")
	ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida = errors.New("vec: serializacion de solicitud de autorizacion ligada V3 prohibida")
)

// DatosSolicitudAutorizacionLigadaV3 contiene solo capacidades resueltas por
// el servidor. V3 no admite el vinculo V1: el vinculo de actor V2 es la unica
// fuente de identidad, garantia y procedencia de contexto.
type DatosSolicitudAutorizacionLigadaV3 struct {
	bloqueoSerializacionSolicitudAutorizacionLigadaV3
	VinculoAutenticacionActor VinculoAutenticacionActorV2
	ReferenciaMotivo          ReferenciaEntradaCatalogo
	Accion                    string
	Recurso                   RecursoAutorizable
	Finalidad                 string
	Correlacion               ReferenciaCorrelacionAutorizacionV2
}

type datosSolicitudAutorizacionLigadaV3 struct {
	vinculoAutenticacionActor VinculoAutenticacionActorV2
	referenciaMotivo          ReferenciaEntradaCatalogo
	accion                    string
	recurso                   RecursoAutorizable
	finalidad                 string
	correlacion               ReferenciaCorrelacionAutorizacionV2
}

// SolicitudAutorizacionLigadaV3 es una capacidad nominal opaca. No puede
// convertirse en una solicitud V1/V2 ni reconstruirse desde una entrada.
type SolicitudAutorizacionLigadaV3 struct {
	bloqueoSerializacionSolicitudAutorizacionLigadaV3
	datos *datosSolicitudAutorizacionLigadaV3
}

func NuevaSolicitudAutorizacionLigadaV3(datos DatosSolicitudAutorizacionLigadaV3) (SolicitudAutorizacionLigadaV3, error) {
	if err := prevalidarDatosSolicitudAutorizacionLigadaV3(datos); err != nil {
		return SolicitudAutorizacionLigadaV3{}, errorSolicitudAutorizacionLigadaV3(err)
	}
	clon, err := clonarDatosSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return SolicitudAutorizacionLigadaV3{}, errorSolicitudAutorizacionLigadaV3(err)
	}
	if err := validarDatosSolicitudAutorizacionLigadaV3(clon); err != nil {
		return SolicitudAutorizacionLigadaV3{}, errorSolicitudAutorizacionLigadaV3(err)
	}
	return SolicitudAutorizacionLigadaV3{datos: &datosSolicitudAutorizacionLigadaV3{
		vinculoAutenticacionActor: clon.VinculoAutenticacionActor,
		referenciaMotivo:          clon.ReferenciaMotivo, accion: clon.Accion, recurso: clon.Recurso,
		finalidad: clon.Finalidad, correlacion: clon.Correlacion,
	}}, nil
}

func (s SolicitudAutorizacionLigadaV3) Datos() (DatosSolicitudAutorizacionLigadaV3, error) {
	if s.datos == nil {
		return DatosSolicitudAutorizacionLigadaV3{}, errorSolicitudAutorizacionLigadaV3(nil)
	}
	datos := DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.datos.vinculoAutenticacionActor,
		ReferenciaMotivo:          s.datos.referenciaMotivo, Accion: s.datos.accion, Recurso: s.datos.recurso,
		Finalidad: s.datos.finalidad, Correlacion: s.datos.correlacion,
	}
	clon, err := clonarDatosSolicitudAutorizacionLigadaV3(datos)
	if err != nil || validarDatosSolicitudAutorizacionLigadaV3(clon) != nil {
		return DatosSolicitudAutorizacionLigadaV3{}, errorSolicitudAutorizacionLigadaV3(err)
	}
	return clon, nil
}

func prevalidarDatosSolicitudAutorizacionLigadaV3(datos DatosSolicitudAutorizacionLigadaV3) error {
	if len(datos.Recurso.Ambitos) > maximoElementosAutorizacion ||
		len(datos.Recurso.Atributos) > maximoElementosAutorizacion ||
		datos.Recurso.Validar() != nil || datos.VinculoAutenticacionActor.Validar() != nil ||
		!ReferenciaMotivoAutorizacionV2Valida(datos.ReferenciaMotivo) || datos.Correlacion.Validar() != nil {
		return ErrSolicitudAutorizacionLigadaV3Invalida
	}
	return nil
}

func validarDatosSolicitudAutorizacionLigadaV3(datos DatosSolicitudAutorizacionLigadaV3) error {
	vinculo, errVinculo := datos.VinculoAutenticacionActor.Datos()
	correlacion, errCorrelacion := datos.Correlacion.ValorCanonico()
	if errVinculo != nil || errCorrelacion != nil || !ReferenciaMotivoAutorizacionV2Valida(datos.ReferenciaMotivo) {
		return errorSolicitudAutorizacionLigadaV3(errors.Join(errVinculo, errCorrelacion))
	}
	proyeccion := SolicitudAutorizacion{
		Principal:       Principal{ID: vinculo.PrincipalID, AuthMethod: vinculo.MetodoObservado, AuthAssurance: vinculo.GarantiaObservada},
		PerfilActivoRef: vinculo.PerfilActivoRef, Accion: datos.Accion, Recurso: datos.Recurso,
		Finalidad: datos.Finalidad, CorrelacionRef: correlacion, Motivo: datos.ReferenciaMotivo.EntradaClave,
	}
	if proyeccion.Validar() != nil || !ReferenciaCorrelacionAutorizacionV2Valida(correlacion) {
		return errorSolicitudAutorizacionLigadaV3(nil)
	}
	return nil
}

func clonarDatosSolicitudAutorizacionLigadaV3(datos DatosSolicitudAutorizacionLigadaV3) (DatosSolicitudAutorizacionLigadaV3, error) {
	if err := prevalidarDatosSolicitudAutorizacionLigadaV3(datos); err != nil {
		return DatosSolicitudAutorizacionLigadaV3{}, err
	}
	recurso := datos.Recurso
	recurso.Ambitos = clonarMapaAutorizacion(datos.Recurso.Ambitos)
	recurso.Atributos = clonarMapaAutorizacion(datos.Recurso.Atributos)
	return DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: datos.VinculoAutenticacionActor, ReferenciaMotivo: datos.ReferenciaMotivo,
		Accion: datos.Accion, Recurso: recurso, Finalidad: datos.Finalidad, Correlacion: datos.Correlacion,
	}, nil
}

func errorSolicitudAutorizacionLigadaV3(causa error) error {
	return errors.Join(ErrSolicitudAutorizacionInvalida, ErrSolicitudAutorizacionLigadaV3Invalida, causa)
}

type bloqueoSerializacionSolicitudAutorizacionLigadaV3 struct{}

func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV3) UnmarshalText([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV3) GobDecode([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV3) String() string {
	return "[SOLICITUD-AUTORIZACION-LIGADA-V3-OPACA]"
}
func (b bloqueoSerializacionSolicitudAutorizacionLigadaV3) GoString() string { return b.String() }
func (b bloqueoSerializacionSolicitudAutorizacionLigadaV3) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, b.String())
}
func (b bloqueoSerializacionSolicitudAutorizacionLigadaV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
