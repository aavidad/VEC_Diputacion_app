// Package ports define los límites del despacho de correo de llamamientos.
package ports

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrSolicitudCorreoLlamamientoInvalida    = errors.New("contratacion temporal: solicitud de correo de llamamiento invalida")
	ErrResultadoCorreoLlamamientoNoConfiable = errors.New("contratacion temporal: resultado de correo de llamamiento no confiable")
	ErrDespachoCorreoLlamamientoDenegado     = errors.New("contratacion temporal: despacho de correo de llamamiento denegado")
)

const PlantillaCorreoLlamamientoV1 = "llamamiento_rrhh_v1"
const AccionDespacharCorreoLlamamiento = "contratacion_temporal.llamamiento.correo.despachar"

// AudienciaDespachoCorreoLlamamientoV3 es candidata para CT88/AD3. SQL aún
// no la consume: su verificación y consumo atómico pertenecen a CT88.
const AudienciaDespachoCorreoLlamamientoV3 = "vec_contratacion_temporal.despacho_correo_llamamiento.v1"
const TipoRecursoDespachoCorreoLlamamiento = "despacho_correo_llamamiento_contratacion_temporal"

// SolicitudDespacharCorreoLlamamiento liga el despacho a la intención local
// que creó CT54. No contiene destino ni contenido personal.
type SolicitudDespacharCorreoLlamamiento struct {
	OrganizacionRef   string
	ExpedienteRef     string
	LlamamientoRef    string
	ComunicacionRef   string
	IntencionEnvioRef string
}

func (s SolicitudDespacharCorreoLlamamiento) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(s.ComunicacionRef) ||
		!domain.ReferenciaOpacaValida(s.IntencionEnvioRef) {
		return ErrSolicitudCorreoLlamamientoInvalida
	}
	return nil
}

// CapacidadDespachoCorreoLlamamiento representa la autorización V3 ya
// verificada por la frontera confiable. La aplicación no la convierte en un
// booleano ni deriva permisos del registro local.
type CapacidadDespachoCorreoLlamamiento struct {
	material        vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	solicitudHuella string
}

func RecursoDespachoCorreoLlamamiento(s SolicitudDespacharCorreoLlamamiento) (vecdomain.RecursoAutorizable, error) {
	if s.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrSolicitudCorreoLlamamientoInvalida
	}
	b, e := json.Marshal(s)
	if e != nil {
		return vecdomain.RecursoAutorizable{}, e
	}
	h := sha256.Sum256(b)
	return vecdomain.RecursoAutorizable{Referencia: s.IntencionEnvioRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoDespachoCorreoLlamamiento, Ambitos: map[string]string{"organizacion_ref": s.OrganizacionRef}, Atributos: map[string]string{"material_sha256": fmt.Sprintf("%x", h[:])}}, nil
}
func NuevaCapacidadDespachoCorreoLlamamiento(s SolicitudDespacharCorreoLlamamiento, m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) (CapacidadDespachoCorreoLlamamiento, error) {
	r, e := RecursoDespachoCorreoLlamamiento(s)
	if e != nil || m.ValidarEstructura() != nil {
		return CapacidadDespachoCorreoLlamamiento{}, ErrDespachoCorreoLlamamientoDenegado
	}
	h, e := r.HuellaContextoAutorizacionSHA256()
	x := m.ResumenCapacidad()
	if e != nil || x.Operacion() != AccionDespacharCorreoLlamamiento || x.EfectoRef() != s.IntencionEnvioRef || x.EfectoHuellaSHA256() != h || x.AudienciaConsumo() != AudienciaDespachoCorreoLlamamientoV3 || ahora.Before(x.EmitidaEn()) || !ahora.Before(x.ExpiraEn()) {
		return CapacidadDespachoCorreoLlamamiento{}, ErrDespachoCorreoLlamamientoDenegado
	}
	b, _ := json.Marshal(s)
	return CapacidadDespachoCorreoLlamamiento{material: m, solicitudHuella: fmt.Sprintf("%x", sha256.Sum256(b))}, nil
}
func (c CapacidadDespachoCorreoLlamamiento) ValidarPara(s SolicitudDespacharCorreoLlamamiento, ahora time.Time) error {
	n, e := NuevaCapacidadDespachoCorreoLlamamiento(s, c.material, ahora)
	if e != nil || n.solicitudHuella != c.solicitudHuella {
		return ErrDespachoCorreoLlamamientoDenegado
	}
	return nil
}
func (c CapacidadDespachoCorreoLlamamiento) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	m, _ := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(c.material.CapacidadCanonica(), c.material.ResumenCapacidad(), c.material.DecisionCanonica(), c.material.MotivoCanonico(), c.material.ContextoActorCanonico(), c.material.PersonaVersion(), c.material.PerfilVersion(), c.material.PayloadVECAD3(), c.material.SobreCOSESign1(), c.material.EvidenciaVerificacion(), c.material.RaizPublicaSPKI())
	return m
}

type AutorizadorDespachoCorreoLlamamiento interface {
	AutorizarDespachoCorreoLlamamiento(context.Context, SolicitudDespacharCorreoLlamamiento) (CapacidadDespachoCorreoLlamamiento, error)
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
func (d DestinoCorreoLlamamiento) MensajePlantilla() MensajeCorreoLlamamiento {
	return MensajeCorreoLlamamiento{destino: d.destino, asunto: "Llamamiento de contratación temporal", cuerpo: "Tiene disponible una comunicación de llamamiento de contratación temporal. Consulte su contenido a través del canal indicado por Recursos Humanos."}
}

type EstadoCorreoLlamamiento uint8

const (
	CorreoLlamamientoIniciado EstadoCorreoLlamamiento = iota + 1
	CorreoLlamamientoNoAceptadoTransitorio
	CorreoLlamamientoNoAceptadoPermanente
	CorreoLlamamientoIndeterminado
	CorreoLlamamientoAceptadoPorRelay
)

func (e EstadoCorreoLlamamiento) Valido() bool {
	return e >= CorreoLlamamientoIniciado && e <= CorreoLlamamientoAceptadoPorRelay
}
func (e EstadoCorreoLlamamiento) EsResultado() bool {
	return e >= CorreoLlamamientoNoAceptadoTransitorio && e <= CorreoLlamamientoAceptadoPorRelay
}

type TransportadorCorreoLlamamiento interface {
	Enviar(context.Context, MensajeCorreoLlamamiento) EstadoCorreoLlamamiento
}

type ReservaIntentoCorreoLlamamiento struct {
	IntentoRef      string
	MessageID       string
	FechaOrigen     time.Time
	YaReservado     bool
	SolicitudHuella string
	Estado          EstadoCorreoLlamamiento
}

func (r ReservaIntentoCorreoLlamamiento) ValidarPara(s SolicitudDespacharCorreoLlamamiento) error {
	b, e := json.Marshal(s)
	if e != nil || r.Validar() != nil || r.SolicitudHuella != fmt.Sprintf("%x", sha256.Sum256(b)) || !r.Estado.Valido() || (!r.YaReservado && r.Estado != CorreoLlamamientoIniciado) {
		return ErrResultadoCorreoLlamamientoNoConfiable
	}
	return nil
}

func (r ReservaIntentoCorreoLlamamiento) Validar() error {
	if !domain.ReferenciaOpacaValida(r.IntentoRef) || r.MessageID == "" || !domain.InstanteUTCCanonico(r.FechaOrigen) {
		return ErrResultadoCorreoLlamamientoNoConfiable
	}
	return nil
}

type RegistroIntentosCorreoLlamamiento interface {
	ReservarIntentoCorreoLlamamiento(context.Context, SolicitudDespacharCorreoLlamamiento, CapacidadDespachoCorreoLlamamiento) (ReservaIntentoCorreoLlamamiento, error)
	RegistrarResultadoIntentoCorreoLlamamiento(context.Context, ReservaIntentoCorreoLlamamiento, EstadoCorreoLlamamiento, string) error
	ConsultarIntentoCorreoLlamamiento(context.Context, SolicitudDespacharCorreoLlamamiento, CapacidadDespachoCorreoLlamamiento) (ReservaIntentoCorreoLlamamiento, EstadoCorreoLlamamiento, error)
}
