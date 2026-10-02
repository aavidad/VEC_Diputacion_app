package ports

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	AccionConsultarPeticionesRRHH    = "contratacion_temporal.peticion_centro.rrhh.consultar"
	AccionEntregarPeticionRRHH       = "contratacion_temporal.peticion_centro.rrhh.entregar"
	TipoRecursoEntregaPeticionCentro = "entrega_peticion_centro"
	FinalidadEntregaPeticionCentro   = "tramitar_peticion_centro_rrhh"
)

var ErrEntregaPeticionEnConflicto = errors.New("contratacion temporal: entrega de peticion en conflicto")
var ErrNumeroMOADAusente = errors.New("contratacion temporal: numero MOAD ausente para entrega nueva")

// La referencia de petición identifica la operación. El servidor reserva una
// sola clave de alta; el navegador nunca elige una clave, un actor o los datos.
type ComandoEntregarPeticionCentro struct {
	NumeroExpedienteMOAD string `json:"numero_expediente_moad,omitempty"`
	PeticionRef          string `json:"peticion_ref"`
	VersionEsperada      uint64 `json:"version_esperada"`
}

func (c ComandoEntregarPeticionCentro) Validar() error {
	if c.NumeroExpedienteMOAD != "" && !domain.NumeroExpedienteValido(c.NumeroExpedienteMOAD) {
		return domain.ErrPeticionCentroInvalida
	}
	if !domain.ReferenciaOpacaValida(c.PeticionRef) || c.VersionEsperada != 2 {
		return domain.ErrPeticionCentroInvalida
	}
	return nil
}

type EntregaPeticionCentro struct {
	Peticion       domain.DatosPeticionCentro `json:"peticion"`
	EstadoEntrega  string                     `json:"estado_entrega"`
	ClaveAlta      string                     `json:"clave_alta,omitempty"`
	AmbitoAltaHMAC string                     `json:"ambito_alta_hmac,omitempty"`
	ActorRef       string                     `json:"actor_ref,omitempty"`
	PerfilRef      string                     `json:"perfil_ref,omitempty"`
	ReciboAlta     *ReciboAlta                `json:"recibo_alta,omitempty"`
	// AltaAnterior sólo la fija el adaptador tras contrastar el original en la
	// misma transacción de entrega. Nunca sale en JSON ni procede del cliente.
	AltaAnterior *AltaDePeticionCentro `json:"-"`
	// Estas señales describen las inserciones de esta solicitud HTTP. No forman
	// parte de la petición, la reserva ni el recibo conservado.
	ReservaCreadaAhora bool `json:"-"`
	ConfirmadaAhora    bool `json:"-"`
}

func (e EntregaPeticionCentro) Validar() error {
	if _, err := domain.RehidratarPeticionCentro(e.Peticion); err != nil || e.Peticion.Version != 2 || e.Peticion.Estado != "ratificada" {
		return domain.ErrPeticionCentroInvalida
	}
	if (e.ConfirmadaAhora && e.EstadoEntrega != "confirmada") ||
		(e.ReservaCreadaAhora && e.EstadoEntrega == "pendiente") {
		return ErrReciboPeticionCentroNoConfiable
	}
	switch e.EstadoEntrega {
	case "pendiente", "preparada":
		if e.ReciboAlta != nil {
			return ErrReciboPeticionCentroNoConfiable
		}
	case "confirmada":
		if e.ReciboAlta == nil || e.ReciboAlta.ValidarEstructura() != nil || e.ReciboAlta.Version != 1 {
			return ErrReciboPeticionCentroNoConfiable
		}
	default:
		return ErrReciboPeticionCentroNoConfiable
	}
	if e.AltaAnterior != nil && (e.AltaAnterior.Recibo.ValidarEstructura() != nil ||
		e.AltaAnterior.AmbitoHMAC != e.AmbitoAltaHMAC ||
		e.EstadoEntrega == "confirmada" && !reflect.DeepEqual(e.ReciboAlta, &e.AltaAnterior.Recibo)) {
		return ErrReciboPeticionCentroNoConfiable
	}
	return nil
}

// OriginalAltaEntrega contiene sólo las coordenadas que permiten cotejar el
// canon previo a MOAD con la historia confirmada, sin exponerla por HTTP.
type OriginalAltaEntrega struct {
	Esquema            string                 `json:"esquema"`
	OrganizacionRef    string                 `json:"organizacion_ref"`
	ActorRef           string                 `json:"actor_ref"`
	PerfilRef          string                 `json:"perfil_ref"`
	Flujo              domain.ReferenciaFlujo `json:"flujo"`
	PoliticaFin        *domain.PoliticaFin    `json:"politica_fin"`
	AmbitoHMAC         string                 `json:"ambito_hmac"`
	HuellaPeticionHMAC string                 `json:"huella_peticion_hmac"`
	ReciboAlta         ReciboAlta             `json:"recibo_alta"`
}

func (o OriginalAltaEntrega) Validar() error {
	if o.Esquema != "vec.contratacion-temporal.original-alta-entrega.v1" ||
		!domain.ReferenciaOpacaValida(o.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(o.ActorRef) || !domain.ReferenciaOpacaValida(o.PerfilRef) ||
		o.Flujo.Validar() != nil || o.ReciboAlta.ValidarEstructura() != nil ||
		!SelloHMACSHA256Valido(o.AmbitoHMAC) || !SelloHMACSHA256Valido(o.HuellaPeticionHMAC) ||
		o.PoliticaFin != nil && o.PoliticaFin.Validar() != nil {
		return ErrReciboPeticionCentroNoConfiable
	}
	return nil
}

func (e EntregaPeticionCentro) ValidarReserva() error {
	if e.Validar() != nil || e.EstadoEntrega == "pendiente" || !ClaveIdempotenciaValida(e.ClaveAlta) ||
		!SelloHMACSHA256Valido(e.AmbitoAltaHMAC) || !domain.ReferenciaOpacaValida(e.ActorRef) || !domain.ReferenciaOpacaValida(e.PerfilRef) {
		return ErrReciboPeticionCentroNoConfiable
	}
	return nil
}

// MaterialEntregaPeticionCentro solo lo construye el servidor. El sello de
// ámbito permite comprobar que el recibo pertenece a la clave reservada.
type MaterialEntregaPeticionCentro struct {
	ClaveAltaCandidata string      `json:"clave_alta_candidata,omitempty"`
	Modo               string      `json:"modo"`
	ActorRef           string      `json:"actor_ref"`
	PerfilRef          string      `json:"perfil_ref"`
	PeticionRef        string      `json:"peticion_ref,omitempty"`
	CentroRef          string      `json:"centro_ref,omitempty"`
	CategoriaRef       string      `json:"categoria_ref,omitempty"`
	VersionEsperada    uint64      `json:"version_esperada,omitempty"`
	ReciboAlta         *ReciboAlta `json:"recibo_alta,omitempty"`
	AmbitoAltaHMAC     string      `json:"ambito_alta_hmac,omitempty"`
}

func (m MaterialEntregaPeticionCentro) Validar() error {
	if !domain.ReferenciaOpacaValida(m.ActorRef) || !domain.ReferenciaOpacaValida(m.PerfilRef) {
		return ErrAutorizacionDenegada
	}
	if m.Modo == "bandeja" {
		if m.PeticionRef != "" || m.CentroRef != "" || m.CategoriaRef != "" || m.VersionEsperada != 0 || m.ReciboAlta != nil || m.AmbitoAltaHMAC != "" || m.ClaveAltaCandidata != "" {
			return domain.ErrPeticionCentroInvalida
		}
		return nil
	}
	if (ComandoEntregarPeticionCentro{PeticionRef: m.PeticionRef, VersionEsperada: m.VersionEsperada}).Validar() != nil {
		return domain.ErrPeticionCentroInvalida
	}
	if !domain.ReferenciaOpacaValida(m.CentroRef) || !domain.ReferenciaOpacaValida(m.CategoriaRef) {
		return domain.ErrPeticionCentroInvalida
	}
	if m.Modo == "preparar" && m.ReciboAlta == nil && ClaveIdempotenciaValida(m.ClaveAltaCandidata) && SelloHMACSHA256Valido(m.AmbitoAltaHMAC) {
		return nil
	}
	if m.Modo == "confirmar" && m.ClaveAltaCandidata == "" && m.ReciboAlta != nil && m.ReciboAlta.ValidarEstructura() == nil && m.ReciboAlta.Version == 1 && SelloHMACSHA256Valido(m.AmbitoAltaHMAC) {
		return nil
	}
	return domain.ErrPeticionCentroInvalida
}

type AltaDePeticionCentro struct {
	Recibo     ReciboAlta
	AmbitoHMAC string
}

type RepositorioEntregasPeticionCentro interface {
	ListarPeticionesRRHH(context.Context) ([]EntregaPeticionCentro, error)
	PrepararEntrega(context.Context, ComandoEntregarPeticionCentro) (EntregaPeticionCentro, error)
	ConfirmarEntrega(context.Context, ComandoEntregarPeticionCentro, AltaDePeticionCentro) (EntregaPeticionCentro, error)
}

// El adaptador de composición llama al registro de solicitudes existente,
// conservando su autorización, cifrado, idempotencia y transacción de alta.
type RegistradorExpedientePeticionCentro interface {
	RegistrarExpedientePeticion(context.Context, EntregaPeticionCentro, string) (AltaDePeticionCentro, error)
}
