package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
)

const (
	AccionConfirmarAceptacionDirecta      = "bolsa.respuesta_revision.resolver"
	FinalidadConfirmarAceptacionDirecta   = "resolver_respuestas_bolsa_rrhh"
	AudienciaConfirmarAceptacionDirecta   = "vec_bolsa_llamamientos.respuesta_revision.resolver.v1"
	TipoRecursoConfirmarAceptacionDirecta = "respuesta_revision_bolsa"
)

var ErrConfirmacionAceptacionDirectaConflicto = errors.New("bolsa: aceptacion o version cambiada")
var referenciaRespuestaDirecta = regexp.MustCompile(`^respuesta-portal:[0-9a-f]{64}$`)
var huellaConfirmacionDirecta = regexp.MustCompile(`^[0-9a-f]{64}$`)
var claveConfirmacionDirecta = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,256}$`)

// La referencia de respuesta pertenece a Bolsa y se obtiene dentro de la
// transacción; el cliente solo identifica la ficha y el llamamiento.
type SolicitudConfirmarAceptacionDirecta struct {
	SolicitudCambiarSituacionParticipacion
	LlamamientoRef         string
	SituacionEsperadaDesde time.Time
}

func (s SolicitudConfirmarAceptacionDirecta) Validar() error {
	if s.SolicitudCambiarSituacionParticipacion.Validar() != nil ||
		!ReferenciaOpacaLlamamientoValida(s.BolsaRef) ||
		!ReferenciaOpacaLlamamientoValida(s.ParticipacionRef) ||
		!ReferenciaOpacaLlamamientoValida(s.LlamamientoRef) ||
		!claveConfirmacionDirecta.MatchString(s.ClaveIdempotencia) ||
		!utf8.ValidString(s.Motivo) || len(s.Motivo) > 1000 ||
		strings.TrimSpace(s.Motivo) != s.Motivo ||
		s.Destino != domain.SituacionPendienteIncorporacion || s.FechaDisponible != nil ||
		s.SituacionEsperadaDesde.IsZero() || s.SituacionEsperadaDesde.Location() != time.UTC ||
		s.SituacionEsperadaDesde.Nanosecond()%1_000 != 0 {
		return ErrSituacionParticipacionNoDisponible
	}
	return nil
}

// El orden de los campos fija los bytes que SQL coteja con el atributo V3.
func CanonConfirmacionAceptacionDirecta(s SolicitudConfirmarAceptacionDirecta) (string, string, error) {
	if s.Validar() != nil {
		return "", "", ErrSituacionParticipacionNoDisponible
	}
	b, err := json.Marshal(struct {
		Esquema                string `json:"esquema"`
		BolsaRef               string `json:"bolsa_ref"`
		ParticipacionRef       string `json:"participacion_ref"`
		LlamamientoRef         string `json:"llamamiento_ref"`
		SituacionEsperadaDesde string `json:"situacion_desde_esperada"`
		Motivo                 string `json:"motivo"`
		ClaveIdempotencia      string `json:"clave_idempotencia"`
		ActorRef               string `json:"actor_ref"`
		Destino                string `json:"destino"`
	}{"vec.bolsa.respuesta-revision.confirmar-aceptacion.v1", s.BolsaRef, s.ParticipacionRef,
		s.LlamamientoRef, s.SituacionEsperadaDesde.Format("2006-01-02T15:04:05.000000Z"),
		s.Motivo, s.ClaveIdempotencia, s.ResultadoContexto.Contexto.PersonaRef, s.Destino})
	if err != nil || len(b) > 16*1024 {
		return "", "", ErrSituacionParticipacionNoDisponible
	}
	h := sha256.Sum256(b)
	return string(b), hex.EncodeToString(h[:]), nil
}

type ComandoConfirmarAceptacionDirecta struct {
	ComandoCambiarSituacionParticipacion
	LlamamientoRef          string
	SituacionEsperadaDesde  time.Time
	Canon                   string
	CanonSHA256             string
	ContextoRecursoCanonico []byte
}

type RegistroConfirmacionAceptacionDirecta struct {
	RegistroSituacionParticipacion
	BolsaRef               string
	LlamamientoRef         string
	RespuestaRef           string
	ResolucionRef          string
	SituacionEsperadaDesde time.Time
	CanonSHA256            string
}

// El adaptador comprueba este contrato con el recibo de SQL antes del COMMIT.
func (r RegistroConfirmacionAceptacionDirecta) ValidarPara(c ComandoConfirmarAceptacionDirecta) error {
	if !referenciaRespuestaDirecta.MatchString(r.RespuestaRef) ||
		!ReferenciaOpacaLlamamientoValida(r.ResolucionRef) ||
		r.ReciboRef != c.ReciboRef || !ReferenciaOpacaLlamamientoValida(r.ReciboRef) ||
		r.BolsaRef != c.BolsaRef || r.ParticipacionRef != c.Cambio.ParticipacionRef ||
		r.LlamamientoRef != c.LlamamientoRef || r.Motivo != c.Cambio.Motivo ||
		r.Situacion != domain.SituacionPendienteIncorporacion ||
		r.FechaDisponible != nil ||
		!r.SituacionEsperadaDesde.Equal(c.SituacionEsperadaDesde) ||
		r.SituacionEsperadaDesde.Location() != time.UTC || r.SituacionEsperadaDesde.Nanosecond()%1_000 != 0 ||
		r.CanonSHA256 != c.CanonSHA256 || !huellaConfirmacionDirecta.MatchString(r.CanonSHA256) ||
		r.ResueltaEn == nil || r.ResueltaEn.IsZero() || r.ResueltaEn.Location() != time.UTC ||
		r.ResueltaEn.Nanosecond()%1_000 != 0 || r.Desde.IsZero() || r.Desde.Location() != time.UTC ||
		r.Desde.Nanosecond()%1_000 != 0 {
		return ErrSituacionParticipacionNoDisponible
	}
	return nil
}

type RepositorioConfirmacionAceptacionDirecta interface {
	ConfirmarAceptacionDirecta(context.Context, ComandoConfirmarAceptacionDirecta) (RegistroConfirmacionAceptacionDirecta, error)
}
