package domain

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var (
	ErrSolicitudAutorizacionLigadaV2Invalida = errors.New(
		"vec: solicitud de autorizacion ligada V2 invalida",
	)
	ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida = errors.New(
		"vec: serializacion de solicitud de autorizacion ligada V2 prohibida",
	)
)

// DatosSolicitudAutorizacionLigadaV2 declara exclusivamente la solicitud
// efectiva V2. Identidad, perfil, metodo y garantia se derivan del vinculo;
// no admite Principal declarado ni un campo de texto Motivo.
type DatosSolicitudAutorizacionLigadaV2 struct {
	bloqueoSerializacionSolicitudAutorizacionLigadaV2
	ContextoActor             ContextoActor
	VinculoAutenticacionActor VinculoAutenticacionActorV1
	ReferenciaMotivo          ReferenciaEntradaCatalogo
	Accion                    string
	Recurso                   RecursoAutorizable
	Finalidad                 string
	CorrelacionRef            string
}

type datosSolicitudAutorizacionLigadaV2 struct {
	contextoActor             ContextoActor
	vinculoAutenticacionActor VinculoAutenticacionActorV1
	referenciaMotivo          ReferenciaEntradaCatalogo
	accion                    string
	recurso                   RecursoAutorizable
	finalidad                 string
	correlacionRef            string
}

// SolicitudAutorizacionLigadaV2 es una capacidad nominal opaca. No puede
// confundirse con SolicitudAutorizacion, que conserva el contrato historico V1.
type SolicitudAutorizacionLigadaV2 struct {
	bloqueoSerializacionSolicitudAutorizacionLigadaV2
	datos *datosSolicitudAutorizacionLigadaV2
}

// NuevaSolicitudAutorizacionLigadaV2 es la unica entrada al contrato V2. Toma
// una copia defensiva y falla cerrado antes de crear la capacidad.
func NuevaSolicitudAutorizacionLigadaV2(
	datos DatosSolicitudAutorizacionLigadaV2,
) (SolicitudAutorizacionLigadaV2, error) {
	clon, err := clonarDatosSolicitudAutorizacionLigadaV2(datos)
	if err != nil {
		return SolicitudAutorizacionLigadaV2{}, errorSolicitudAutorizacionLigadaV2(err)
	}
	if err := validarDatosSolicitudAutorizacionLigadaV2(clon); err != nil {
		return SolicitudAutorizacionLigadaV2{}, errorSolicitudAutorizacionLigadaV2(err)
	}
	return SolicitudAutorizacionLigadaV2{datos: &datosSolicitudAutorizacionLigadaV2{
		contextoActor: clon.ContextoActor, vinculoAutenticacionActor: clon.VinculoAutenticacionActor,
		referenciaMotivo: clon.ReferenciaMotivo, accion: clon.Accion, recurso: clon.Recurso,
		finalidad: clon.Finalidad, correlacionRef: clon.CorrelacionRef,
	}}, nil
}

// Datos entrega una copia defensiva deliberada a la capa de aplicacion. El
// resultado sigue bloqueando codecs y formato para no convertirse en DTO HTTP.
func (s SolicitudAutorizacionLigadaV2) Datos() (
	DatosSolicitudAutorizacionLigadaV2,
	error,
) {
	if s.datos == nil {
		return DatosSolicitudAutorizacionLigadaV2{}, errorSolicitudAutorizacionLigadaV2(nil)
	}
	datos := DatosSolicitudAutorizacionLigadaV2{
		ContextoActor: s.datos.contextoActor, VinculoAutenticacionActor: s.datos.vinculoAutenticacionActor,
		ReferenciaMotivo: s.datos.referenciaMotivo, Accion: s.datos.accion,
		Recurso: s.datos.recurso, Finalidad: s.datos.finalidad, CorrelacionRef: s.datos.correlacionRef,
	}
	clon, err := clonarDatosSolicitudAutorizacionLigadaV2(datos)
	if err != nil {
		return DatosSolicitudAutorizacionLigadaV2{}, errorSolicitudAutorizacionLigadaV2(err)
	}
	if err := validarDatosSolicitudAutorizacionLigadaV2(clon); err != nil {
		return DatosSolicitudAutorizacionLigadaV2{}, errorSolicitudAutorizacionLigadaV2(err)
	}
	return clon, nil
}

func validarDatosSolicitudAutorizacionLigadaV2(
	datos DatosSolicitudAutorizacionLigadaV2,
) error {
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil || !ReferenciaMotivoAutorizacionV2Valida(datos.ReferenciaMotivo) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(datos.CorrelacionRef) {
		return errorSolicitudAutorizacionLigadaV2(err)
	}
	proyeccion := SolicitudAutorizacion{
		Principal: Principal{
			ID: vinculo.PrincipalID, AuthMethod: vinculo.MetodoObservado,
			AuthAssurance: vinculo.GarantiaObservada,
		},
		PerfilActivoRef: vinculo.PerfilActivoRef, Accion: datos.Accion,
		Recurso: datos.Recurso, Finalidad: datos.Finalidad,
		CorrelacionRef: datos.CorrelacionRef, Motivo: datos.ReferenciaMotivo.EntradaClave,
	}
	if proyeccion.Validar() != nil {
		return errorSolicitudAutorizacionLigadaV2(nil)
	}
	if !contextoActorSolicitudAutorizacionAusenteV2(datos.ContextoActor) &&
		(datos.ContextoActor.Validar() != nil ||
			datos.VinculoAutenticacionActor.ValidarPara(datos.ContextoActor) != nil) {
		return errorSolicitudAutorizacionLigadaV2(nil)
	}
	return nil
}

func clonarDatosSolicitudAutorizacionLigadaV2(
	datos DatosSolicitudAutorizacionLigadaV2,
) (DatosSolicitudAutorizacionLigadaV2, error) {
	contexto := ContextoActor{}
	var err error
	if !contextoActorSolicitudAutorizacionAusenteV2(datos.ContextoActor) {
		contexto, err = datos.ContextoActor.Clonar()
		if err != nil {
			return DatosSolicitudAutorizacionLigadaV2{}, err
		}
	}
	recurso := datos.Recurso
	recurso.Ambitos = clonarMapaAutorizacion(datos.Recurso.Ambitos)
	recurso.Atributos = clonarMapaAutorizacion(datos.Recurso.Atributos)
	return DatosSolicitudAutorizacionLigadaV2{
		ContextoActor: contexto, VinculoAutenticacionActor: datos.VinculoAutenticacionActor,
		ReferenciaMotivo: datos.ReferenciaMotivo, Accion: datos.Accion, Recurso: recurso,
		Finalidad: datos.Finalidad, CorrelacionRef: datos.CorrelacionRef,
	}, nil
}

func errorSolicitudAutorizacionLigadaV2(causa error) error {
	return errors.Join(ErrSolicitudAutorizacionInvalida, ErrSolicitudAutorizacionLigadaV2Invalida, causa)
}

type bloqueoSerializacionSolicitudAutorizacionLigadaV2 struct{}

func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV2) UnmarshalText([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV2) GobDecode([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV2) UnmarshalCBOR([]byte) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV2) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (*bloqueoSerializacionSolicitudAutorizacionLigadaV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida
}
func (bloqueoSerializacionSolicitudAutorizacionLigadaV2) String() string {
	return "[SOLICITUD-AUTORIZACION-LIGADA-V2-OPACA]"
}
func (b bloqueoSerializacionSolicitudAutorizacionLigadaV2) GoString() string { return b.String() }
func (b bloqueoSerializacionSolicitudAutorizacionLigadaV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionSolicitudAutorizacionLigadaV2) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
