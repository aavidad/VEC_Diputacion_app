package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	AccionConsultarPeticionesRRHH    = "contratacion_temporal.peticion_centro.rrhh.consultar"
	AccionEntregarPeticionRRHH       = "contratacion_temporal.peticion_centro.rrhh.entregar"
	TipoRecursoEntregaPeticionCentro = "entrega_peticion_centro"
	FinalidadEntregaPeticionCentro   = "tramitar_peticion_centro_rrhh"
)

var ErrEntregaPeticionEnConflicto = errors.New("contratacion temporal: entrega de peticion en conflicto")

// La referencia de petición identifica la operación. El servidor reserva una
// sola clave de alta; el navegador nunca elige una clave, un actor o los datos.
type ComandoEntregarPeticionCentro struct {
	PeticionRef     string `json:"peticion_ref"`
	VersionEsperada uint64 `json:"version_esperada"`
}

func (c ComandoEntregarPeticionCentro) Validar() error {
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
}

func (e EntregaPeticionCentro) Validar() error {
	if _, err := domain.RehidratarPeticionCentro(e.Peticion); err != nil || e.Peticion.Version != 2 || e.Peticion.Estado != "ratificada" {
		return domain.ErrPeticionCentroInvalida
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
	VersionEsperada    uint64      `json:"version_esperada,omitempty"`
	ReciboAlta         *ReciboAlta `json:"recibo_alta,omitempty"`
	AmbitoAltaHMAC     string      `json:"ambito_alta_hmac,omitempty"`
}

func (m MaterialEntregaPeticionCentro) Validar() error {
	if !domain.ReferenciaOpacaValida(m.ActorRef) || !domain.ReferenciaOpacaValida(m.PerfilRef) {
		return ErrAutorizacionDenegada
	}
	if m.Modo == "bandeja" {
		if m.PeticionRef != "" || m.VersionEsperada != 0 || m.ReciboAlta != nil || m.AmbitoAltaHMAC != "" || m.ClaveAltaCandidata != "" {
			return domain.ErrPeticionCentroInvalida
		}
		return nil
	}
	if (ComandoEntregarPeticionCentro{m.PeticionRef, m.VersionEsperada}).Validar() != nil {
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
	RegistrarExpedientePeticion(context.Context, EntregaPeticionCentro) (AltaDePeticionCentro, error)
}
