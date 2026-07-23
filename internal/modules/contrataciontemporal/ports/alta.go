package ports

import (
	"context"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	ErrPreparacionAltaInvalida  = errors.New("contratacion temporal: preparacion de alta invalida")
	ErrOrdenAltaInvalida        = errors.New("contratacion temporal: orden de alta invalida")
	ErrPersistenciaNoDisponible = errors.New("contratacion temporal: persistencia no disponible")
	ErrClaveIdempotenciaUsada   = errors.New("contratacion temporal: clave de idempotencia usada con otros datos")
)

// La clave debe generarla cada cliente con CSPRNG y conservarse solo durante
// el reintento. El formato UUIDv4 canónico descarta etiquetas humanas, formas
// no canónicas y el centinela nulo; la sintaxis no prueba por sí sola la
// calidad del generador, que se exige en cada adaptador de entrada.
var patronClaveIdempotencia = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

func claveIdempotenciaValida(valor string) bool {
	return patronClaveIdempotencia.MatchString(valor) &&
		valor != "00000000-0000-4000-8000-000000000000"
}

type ReferenciasAlta struct {
	ExpedienteRef string
	NumeroVisible string
	ReciboRef     string
}

func (r ReferenciasAlta) Validar() error {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.NumeroExpedienteValido(r.NumeroVisible) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

type MaterialHuellaAlta struct {
	OrganizacionRef string
	ActorRef        string
	PerfilRef       string
	Flujo           domain.ReferenciaFlujo
	Solicitud       domain.SolicitudCentro
}

func (m MaterialHuellaAlta) Validar() error {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(m.ActorRef) ||
		!domain.ReferenciaOpacaValida(m.PerfilRef) ||
		m.Flujo.Validar() != nil || m.Solicitud.Validar() != nil {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// DerivadorHuellaAlta usa una clave gestionada fuera del proceso. Nunca
// persiste el material en claro como sustituto de la solicitud.
type DerivadorHuellaAlta interface {
	DerivarHuellaAlta(
		context.Context,
		MaterialHuellaAlta,
	) (ColeccionSellosHMAC, error)
}

type SolicitudPrepararAlta struct {
	ClaveIdempotencia   string
	HuellasPeticionHMAC ColeccionSellosHMAC
	OrganizacionRef     string
	ActorRef            string
	PerfilRef           string
}

func (s SolicitudPrepararAlta) Validar() error {
	if !claveIdempotenciaValida(s.ClaveIdempotencia) ||
		s.HuellasPeticionHMAC.ValidarDominio(
			"vec.contratacion-temporal.huella-peticion",
		) != nil ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

type EstadoPreparacionAlta string

const (
	PreparacionReservada  EstadoPreparacionAlta = "reservada"
	PreparacionConfirmada EstadoPreparacionAlta = "confirmada"
)

type PreparacionAlta struct {
	ReservaRef             string
	Referencias            ReferenciasAlta
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	OrganizacionRef        string
	ActorRef               string
	PerfilRef              string
	Estado                 EstadoPreparacionAlta
	ReciboConfirmado       *ReciboAlta
}

func (p PreparacionAlta) ValidarPara(solicitud SolicitudPrepararAlta) error {
	if solicitud.Validar() != nil || !domain.ReferenciaOpacaValida(p.ReservaRef) ||
		p.Referencias.Validar() != nil ||
		!SelloHMACSHA256Valido(p.AmbitoIdempotenciaHMAC) ||
		!solicitud.HuellasPeticionHMAC.Contiene(p.HuellaPeticionHMAC) ||
		p.OrganizacionRef != solicitud.OrganizacionRef ||
		p.ActorRef != solicitud.ActorRef || p.PerfilRef != solicitud.PerfilRef ||
		(p.Estado != PreparacionReservada && p.Estado != PreparacionConfirmada) {
		return ErrPreparacionAltaInvalida
	}
	if p.Estado == PreparacionReservada && p.ReciboConfirmado != nil {
		return ErrPreparacionAltaInvalida
	}
	if p.Estado == PreparacionConfirmada {
		if p.ReciboConfirmado == nil ||
			p.ReciboConfirmado.ValidarEstructura() != nil ||
			p.ReciboConfirmado.ExpedienteRef != p.Referencias.ExpedienteRef ||
			p.ReciboConfirmado.NumeroVisible != p.Referencias.NumeroVisible ||
			p.ReciboConfirmado.ReciboRef != p.Referencias.ReciboRef {
			return ErrPreparacionAltaInvalida
		}
	}
	return nil
}

type PreparadorAltaIdempotente interface {
	PrepararAlta(context.Context, SolicitudPrepararAlta) (PreparacionAlta, error)
}

type Reloj interface {
	Ahora() time.Time
}

type OrdenConfirmarAlta struct {
	datos *datosOrdenConfirmarAlta
}

type datosOrdenConfirmarAlta struct {
	expediente     domain.Expediente
	identidad      IdentidadOperacion
	autorizacion   AutorizacionEfecto
	preparacion    PreparacionAlta
	correlacionRef string
}

type DatosOrdenConfirmarAlta struct {
	Expediente     domain.Expediente
	Identidad      IdentidadOperacion
	Autorizacion   AutorizacionEfecto
	Preparacion    PreparacionAlta
	CorrelacionRef string
}

func NuevaOrdenConfirmarAlta(datos DatosOrdenConfirmarAlta) (OrdenConfirmarAlta, error) {
	if datos.Expediente.Validar() != nil ||
		!domain.ReferenciaOpacaValida(datos.CorrelacionRef) {
		return OrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	identidad, err := datos.Identidad.Datos()
	autorizacion, errAutorizacion := datos.Autorizacion.Datos()
	if err != nil || errAutorizacion != nil ||
		datos.Preparacion.Estado != PreparacionReservada ||
		!domain.ReferenciaOpacaValida(datos.Preparacion.ReservaRef) ||
		datos.Preparacion.Referencias.Validar() != nil ||
		!SelloHMACSHA256Valido(datos.Preparacion.AmbitoIdempotenciaHMAC) ||
		!SelloHMACSHA256Valido(datos.Preparacion.HuellaPeticionHMAC) ||
		datos.Preparacion.OrganizacionRef != datos.Expediente.OrganizacionRef ||
		datos.Preparacion.ActorRef != identidad.ActorRef ||
		datos.Preparacion.PerfilRef != identidad.PerfilRef ||
		datos.Preparacion.ReciboConfirmado != nil ||
		autorizacion.RecursoRef != datos.Expediente.Referencia ||
		datos.Preparacion.Referencias.ExpedienteRef != datos.Expediente.Referencia ||
		datos.Preparacion.Referencias.NumeroVisible != datos.Expediente.NumeroVisible ||
		datos.Preparacion.Referencias.ReciboRef != datos.Expediente.Actuaciones[0].ReciboRef ||
		autorizacion.ActorRef != identidad.ActorRef ||
		autorizacion.PerfilRef != identidad.PerfilRef ||
		!datos.Identidad.VigenteEn(datos.Expediente.CreadoEn) ||
		datos.Expediente.CreadoEn.Before(autorizacion.EmitidaEn) ||
		!datos.Expediente.CreadoEn.Before(autorizacion.ValidaHasta) {
		return OrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	return OrdenConfirmarAlta{datos: &datosOrdenConfirmarAlta{
		expediente: datos.Expediente.Clonar(), identidad: datos.Identidad,
		autorizacion: datos.Autorizacion, preparacion: datos.Preparacion,
		correlacionRef: datos.CorrelacionRef,
	}}, nil
}

func (o OrdenConfirmarAlta) Datos() (DatosOrdenConfirmarAlta, error) {
	if o.datos == nil {
		return DatosOrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	datos := DatosOrdenConfirmarAlta{
		Expediente: o.datos.expediente.Clonar(), Identidad: o.datos.identidad,
		Autorizacion: o.datos.autorizacion, Preparacion: o.datos.preparacion,
		CorrelacionRef: o.datos.correlacionRef,
	}
	if _, err := NuevaOrdenConfirmarAlta(datos); err != nil {
		return DatosOrdenConfirmarAlta{}, err
	}
	return datos, nil
}

type ReciboAlta struct {
	ExpedienteRef string    `json:"expediente_ref"`
	NumeroVisible string    `json:"numero_visible"`
	Version       uint64    `json:"version"`
	ReciboRef     string    `json:"recibo_ref"`
	AuditoriaRef  string    `json:"auditoria_ref"`
	EventoRef     string    `json:"evento_ref"`
	ConfirmadaEn  time.Time `json:"confirmada_en"`
}

func (r ReciboAlta) ValidarEstructura() error {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.NumeroExpedienteValido(r.NumeroVisible) || r.Version == 0 ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) {
		return ErrPersistenciaNoDisponible
	}
	return nil
}

func (r ReciboAlta) ValidarPara(expediente domain.Expediente) error {
	if expediente.Validar() != nil || r.ExpedienteRef != expediente.Referencia ||
		r.NumeroVisible != expediente.NumeroVisible || r.Version != expediente.Version ||
		r.ValidarEstructura() != nil ||
		r.ConfirmadaEn.Before(expediente.ActualizadoEn) ||
		len(expediente.Actuaciones) == 0 ||
		r.ReciboRef != expediente.Actuaciones[0].ReciboRef {
		return ErrPersistenciaNoDisponible
	}
	return nil
}

// TransaccionAltas debe cotejar y consumir la autorización, confirmar la
// reserva, el expediente, la auditoría y el outbox en un único COMMIT.
type TransaccionAltas interface {
	ConfirmarAlta(context.Context, OrdenConfirmarAlta) (ReciboAlta, error)
}
