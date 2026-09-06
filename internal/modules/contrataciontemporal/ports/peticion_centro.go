package ports

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"
	"unicode"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	OperacionPresentarPeticionCentro = "presentar"
	OperacionRatificarPeticionCentro = "ratificar"
	AccionPresentarPeticionCentro    = "contratacion_temporal.peticion_centro.presentar"
	AccionRatificarPeticionCentro    = "contratacion_temporal.peticion_centro.ratificar"
	AccionConsultarPeticionCentro    = "contratacion_temporal.peticion_centro.consultar"
	TipoRecursoPeticionCentro        = "peticion_centro"
)

var (
	ErrPeticionCentroNoDisponible      = errors.New("contratacion temporal: peticion de centro no disponible")
	ErrClavePeticionCentroUsada        = errors.New("contratacion temporal: clave de peticion de centro usada")
	ErrReciboPeticionCentroNoConfiable = errors.New("contratacion temporal: recibo de peticion de centro no confiable")
)

// ComandoPeticionCentro contiene exclusivamente datos del formulario. Centro y
// contacto de la solicitud no son la identidad ni el cargo de quien actúa.
type ComandoPeticionCentro struct {
	Operacion         string                  `json:"operacion"`
	ClaveIdempotencia string                  `json:"clave_idempotencia"`
	PeticionRef       string                  `json:"peticion_ref,omitempty"`
	VersionEsperada   uint64                  `json:"version_esperada,omitempty"`
	Solicitud         *domain.SolicitudCentro `json:"solicitud,omitempty"`
	Motivo            string                  `json:"motivo,omitempty"`
}

func (s ComandoPeticionCentro) Validar() error {
	if !claveUUIDPeticionCentro(s.ClaveIdempotencia) {
		return domain.ErrPeticionCentroInvalida
	}
	switch s.Operacion {
	case OperacionPresentarPeticionCentro:
		if s.Solicitud == nil || s.Solicitud.Validar() != nil || s.PeticionRef != "" || s.VersionEsperada != 0 || s.Motivo != "" {
			return domain.ErrPeticionCentroInvalida
		}
	case OperacionRatificarPeticionCentro:
		if !domain.ReferenciaOpacaValida(s.PeticionRef) || s.VersionEsperada != 1 || s.Solicitud != nil ||
			s.Motivo == "" || strings.TrimSpace(s.Motivo) != s.Motivo || len(s.Motivo) > 1000 || strings.IndexFunc(s.Motivo, unicode.IsControl) >= 0 {
			return domain.ErrPeticionCentroInvalida
		}
	default:
		return domain.ErrPeticionCentroInvalida
	}
	return nil
}

func (s ComandoPeticionCentro) Clonar() (ComandoPeticionCentro, error) {
	if err := s.Validar(); err != nil {
		return ComandoPeticionCentro{}, err
	}
	if s.Solicitud != nil {
		c, err := s.Solicitud.Clonar()
		if err != nil {
			return ComandoPeticionCentro{}, err
		}
		s.Solicitud = &c
	}
	return s, nil
}

func claveUUIDPeticionCentro(s string) bool {
	if !ClaveIdempotenciaValida(s) || len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' || s[14] != '4' || !strings.ContainsRune("89ab", rune(s[19])) {
		return false
	}
	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// MaterialPeticionCentro solo lo prepara el caso de uso tras resolver la
// adscripción desde una autoridad confiable. No se deserializa desde HTTP.
type MaterialPeticionCentro struct {
	Comando  ComandoPeticionCentro      `json:"comando"`
	Actor    domain.ActorPeticionCentro `json:"actor"`
	Peticion domain.DatosPeticionCentro `json:"peticion"`
}

func (m MaterialPeticionCentro) Validar() error {
	if m.Comando.Validar() != nil {
		return domain.ErrPeticionCentroInvalida
	}
	if _, err := domain.RehidratarPeticionCentro(m.Peticion); err != nil {
		return err
	}
	d := m.Peticion
	if m.Comando.Operacion == OperacionPresentarPeticionCentro {
		if d.Referencia != "peticion:centro:"+m.Comando.ClaveIdempotencia || d.Version != 1 ||
			m.Actor != d.Configuracion.Solicitante || !reflect.DeepEqual(*m.Comando.Solicitud, d.Solicitud) {
			return domain.ErrPeticionCentroInvalida
		}
	} else if d.Referencia != m.Comando.PeticionRef || d.Version != m.Comando.VersionEsperada+1 ||
		m.Actor != d.Configuracion.Ratificador || d.MotivoRatificacion != m.Comando.Motivo {
		return domain.ErrPeticionCentroInvalida
	}
	return nil
}

func (m MaterialPeticionCentro) MismaPeticion(actor domain.ActorPeticionCentro, s ComandoPeticionCentro) bool {
	return m.Validar() == nil && s.Validar() == nil && m.Actor == actor && reflect.DeepEqual(m.Comando, s)
}

type ReciboPeticionCentro struct {
	ReciboRef    string    `json:"recibo_ref"`
	PeticionRef  string    `json:"peticion_ref"`
	Version      uint64    `json:"version"`
	Estado       string    `json:"estado"`
	ActorRef     string    `json:"actor_ref"`
	RegistradoEn time.Time `json:"registrado_en"`
	EstadoLocal  string    `json:"estado_local"`
}

func (r ReciboPeticionCentro) ValidarPara(m MaterialPeticionCentro) error {
	if m.Validar() != nil || !domain.ReferenciaOpacaValida(r.ReciboRef) || r.PeticionRef != m.Peticion.Referencia ||
		r.Version != m.Peticion.Version || r.Estado != m.Peticion.Estado || r.ActorRef != m.Actor.ActorRef ||
		!domain.InstanteUTCCanonico(r.RegistradoEn) || r.RegistradoEn != r.RegistradoEn.Truncate(time.Microsecond) ||
		r.RegistradoEn.Before(m.Peticion.CreadaEn) || r.RegistradoEn.Before(m.Peticion.RatificadaEn) ||
		(r.EstadoLocal != "registrado" && r.EstadoLocal != "replay_confirmado") {
		return ErrReciboPeticionCentroNoConfiable
	}
	return nil
}

// La autoridad resuelve identidad y adscripción, no las toma del comando. La
// configuración puede señalar cualquier cargo realmente habilitado: no existe
// una cadena de niveles de mando compilada en Contratación.
type AutoridadPeticionCentro interface {
	ActorPeticionCentro(context.Context) (domain.ActorPeticionCentro, error)
	ConfiguracionPeticionCentro(context.Context, domain.ActorPeticionCentro) (domain.ConfiguracionPeticionCentro, error)
}

// Toda lectura está autorizada para actor/centro; ausencia se representa por nil.
// Confirmar consume una autorización NUEVA aun al recuperar un recibo anterior,
// y confirma conjuntamente estado, recibo e historial. No hay éxito en memoria.
type RepositorioPeticionesCentro interface {
	ConsultarOperacion(context.Context, domain.ActorPeticionCentro, string) (*MaterialPeticionCentro, error)
	ObtenerPeticion(context.Context, domain.ActorPeticionCentro, string) (domain.DatosPeticionCentro, error)
	ConfirmarPeticion(context.Context, MaterialPeticionCentro) (ReciboPeticionCentro, error)
}

type ConsultaPeticionCentro struct {
	Modo       string                     `json:"modo"`
	Actor      domain.ActorPeticionCentro `json:"actor"`
	Referencia string                     `json:"referencia"`
}

func (s ConsultaPeticionCentro) Validar() error {
	if s.Actor.Validar() != nil {
		return domain.ErrRatificacionCentroDenegada
	}
	switch s.Modo {
	case "operacion":
		if claveUUIDPeticionCentro(s.Referencia) {
			return nil
		}
	case "peticion":
		if domain.ReferenciaOpacaValida(s.Referencia) {
			return nil
		}
	case "bandeja":
		if s.Referencia == s.Actor.CentroRef {
			return nil
		}
	}
	return domain.ErrPeticionCentroInvalida
}

type ConsultaBandejaPeticionesCentro interface {
	ListarPeticiones(context.Context, domain.ActorPeticionCentro) ([]domain.DatosPeticionCentro, error)
}
