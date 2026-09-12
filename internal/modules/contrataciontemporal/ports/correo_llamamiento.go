// Package ports define los límites del despacho de correo de llamamientos.
package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrSolicitudCorreoLlamamientoInvalida    = domain.ErrSolicitudCorreoLlamamientoInvalida
	ErrResultadoCorreoLlamamientoNoConfiable = domain.ErrResultadoCorreoLlamamientoNoConfiable
	ErrDespachoCorreoLlamamientoDenegado     = errors.New("contratacion temporal: despacho de correo de llamamiento denegado")
)

const PlantillaCorreoLlamamientoV1 = domain.PlantillaCorreoLlamamientoV1

// SolicitudDespacharCorreoLlamamiento liga el despacho a la intención local
// que creó CT54. No contiene destino ni contenido personal.
type SolicitudDespacharCorreoLlamamiento = domain.SolicitudDespacharCorreoLlamamiento

// CapacidadDespachoCorreoLlamamiento es transporte opaco de material V3. No
// valida, deriva ni concede autorización; esas decisiones pertenecen al caso
// de uso que construye el recurso exacto y a PostgreSQL que lo consume.
type CapacidadDespachoCorreoLlamamiento struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialDespachoCorreoLlamamiento(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadDespachoCorreoLlamamiento {
	return CapacidadDespachoCorreoLlamamiento{material: m}
}
func (c CapacidadDespachoCorreoLlamamiento) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

type AutorizadorDespachoCorreoLlamamiento interface {
	AutorizarDespachoCorreoLlamamiento(context.Context, SolicitudDespacharCorreoLlamamiento) (CapacidadDespachoCorreoLlamamiento, error)
}

type SolicitudRegistrarResultadoCorreoLlamamiento = domain.SolicitudRegistrarResultadoCorreoLlamamiento
type AuditoriaResultadoCorreoLlamamiento = domain.AuditoriaResultadoCorreoLlamamiento

func NuevaSolicitudRegistrarResultadoCorreoLlamamiento(s SolicitudDespacharCorreoLlamamiento, r ReservaIntentoCorreoLlamamiento, estado EstadoCorreoLlamamiento, plantilla string) (SolicitudRegistrarResultadoCorreoLlamamiento, error) {
	return domain.NuevaSolicitudRegistrarResultadoCorreoLlamamiento(s, r, estado, plantilla)
}

// CapacidadResultadoCorreoLlamamiento transporta sólo el material V3 opaco.
// No puede convertirse por sí misma en una decisión ni validar su efecto.
type CapacidadResultadoCorreoLlamamiento struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialResultadoCorreoLlamamiento(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadResultadoCorreoLlamamiento {
	return CapacidadResultadoCorreoLlamamiento{material: m}
}
func (c CapacidadResultadoCorreoLlamamiento) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

type AutorizadorResultadoCorreoLlamamiento interface {
	AutorizarResultadoCorreoLlamamiento(context.Context, SolicitudRegistrarResultadoCorreoLlamamiento, AuditoriaResultadoCorreoLlamamiento) (CapacidadResultadoCorreoLlamamiento, error)
}

// PreparadorAuditoriaResultadoCorreoLlamamiento crea la evidencia AuditEntry
// desde la capacidad atestada de despacho. Es obligatorio: no se sustituye por
// campos declarados por la aplicación ni por la respuesta SMTP.
type PreparadorAuditoriaResultadoCorreoLlamamiento interface {
	PrepararAuditoriaResultadoCorreoLlamamiento(context.Context, SolicitudRegistrarResultadoCorreoLlamamiento, CapacidadDespachoCorreoLlamamiento, time.Time) (AuditoriaResultadoCorreoLlamamiento, error)
}

// MensajeCorreoLlamamiento es efímero y su contenido privado sólo nace en el
// callback autorizado. El wrapper SMTP obtiene sus datos en memoria.
type MensajeCorreoLlamamiento struct {
	destino, asunto, cuerpo string
	MessageID               string
	FechaOrigen             time.Time
}

func NuevoMensajeCorreoLlamamiento(destino, asunto, cuerpo string) (MensajeCorreoLlamamiento, error) {
	if destino == "" || asunto == "" || cuerpo == "" {
		return MensajeCorreoLlamamiento{}, ErrResultadoCorreoLlamamientoNoConfiable
	}
	return MensajeCorreoLlamamiento{destino: destino, asunto: asunto, cuerpo: cuerpo}, nil
}
func (m MensajeCorreoLlamamiento) Datos() (string, string, string) {
	return m.destino, m.asunto, m.cuerpo
}
func (MensajeCorreoLlamamiento) String() string   { return "MensajeCorreoLlamamiento{redactado}" }
func (MensajeCorreoLlamamiento) GoString() string { return "MensajeCorreoLlamamiento{redactado}" }
func (MensajeCorreoLlamamiento) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}

// ResolutorDestinoCorreoLlamamiento no devuelve la dirección al caso de uso:
// la expone sólo durante el callback, después de reservar la intención.
type ResolutorDestinoCorreoLlamamiento interface {
	ConDestinoCorreoLlamamiento(context.Context, SolicitudDespacharCorreoLlamamiento, CapacidadDespachoCorreoLlamamiento, func(DestinoCorreoLlamamiento) error) error
}
type DestinoCorreoLlamamiento struct{ destino string }

func NuevoDestinoCorreoLlamamiento(v string) (DestinoCorreoLlamamiento, error) {
	if v == "" {
		return DestinoCorreoLlamamiento{}, ErrResultadoCorreoLlamamientoNoConfiable
	}
	return DestinoCorreoLlamamiento{v}, nil
}
func (DestinoCorreoLlamamiento) String() string   { return "DestinoCorreoLlamamiento{redactado}" }
func (DestinoCorreoLlamamiento) GoString() string { return "DestinoCorreoLlamamiento{redactado}" }
func (DestinoCorreoLlamamiento) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}
func (d DestinoCorreoLlamamiento) Mensaje(asunto, cuerpo string) (MensajeCorreoLlamamiento, error) {
	return NuevoMensajeCorreoLlamamiento(d.destino, asunto, cuerpo)
}

type EstadoCorreoLlamamiento = domain.EstadoCorreoLlamamiento

const (
	CorreoLlamamientoIniciado              = domain.CorreoLlamamientoIniciado
	CorreoLlamamientoNoAceptadoTransitorio = domain.CorreoLlamamientoNoAceptadoTransitorio
	CorreoLlamamientoNoAceptadoPermanente  = domain.CorreoLlamamientoNoAceptadoPermanente
	CorreoLlamamientoIndeterminado         = domain.CorreoLlamamientoIndeterminado
	CorreoLlamamientoAceptadoPorRelay      = domain.CorreoLlamamientoAceptadoPorRelay
)

type TransportadorCorreoLlamamiento interface {
	Enviar(context.Context, MensajeCorreoLlamamiento) EstadoCorreoLlamamiento
}

type ReservaIntentoCorreoLlamamiento = domain.ReservaIntentoCorreoLlamamiento
type CapacidadFinalizacionIntentoCorreoLlamamiento = domain.CapacidadFinalizacionIntentoCorreoLlamamiento

type RegistroIntentosCorreoLlamamiento interface {
	ReservarIntentoCorreoLlamamiento(context.Context, SolicitudDespacharCorreoLlamamiento, CapacidadDespachoCorreoLlamamiento) (ReservaIntentoCorreoLlamamiento, CapacidadFinalizacionIntentoCorreoLlamamiento, error)
	RegistrarResultadoIntentoCorreoLlamamiento(context.Context, SolicitudRegistrarResultadoCorreoLlamamiento, CapacidadFinalizacionIntentoCorreoLlamamiento, AuditoriaResultadoCorreoLlamamiento, CapacidadResultadoCorreoLlamamiento) error
}
