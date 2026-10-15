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

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionResolverRespuestaRevision      = "bolsa.respuesta_revision.resolver"
	AccionConsultarRespuestasRevision    = "bolsa.respuesta_revision.consultar"
	FinalidadRespuestaRevision           = "resolver_respuestas_bolsa_rrhh"
	TipoRecursoRespuestaRevision         = "respuesta_revision_bolsa"
	TipoRecursoBandejaRevision           = "respuestas_revision_bolsa"
	AudienciaResolverRespuestaRevision   = "vec_bolsa_llamamientos.respuesta_revision.resolver.v1"
	AudienciaConsultarRespuestasRevision = "vec_bolsa_llamamientos.respuesta_revision.consultar.v1"
	EsquemaResolucionRespuestaRevision   = "vec.bolsa.respuesta-revision.resolver.v1"
)

var (
	ErrResolucionRespuestaRevisionNoDisponible = errors.New("bolsa: resolucion de respuesta no disponible")
	ErrResolucionRespuestaRevisionInvalida     = errors.New("bolsa: resolucion de respuesta invalida")
	ErrResolucionRespuestaRevisionConflicto    = errors.New("bolsa: respuesta o version cambiada")
)

var (
	referenciaRespuestaRevision = regexp.MustCompile(`^respuesta-portal:[0-9a-f]{64}$`)
	claveRespuestaRevision      = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,256}$`)
	referenciaBolsaRevision     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$`)
	huellaRespuestaRevision     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// El preparador confía ámbitos e identidad. El cuerpo HTTP aporta la referencia
// completa del motivo, que SQL coteja con la publicación vigente para un alta
// y con el acto original para un replay, además de respuesta, CAS y clave.
type SolicitudResolverRespuestaRevision struct {
	RespuestaRef           string
	Resultado              string
	MotivoRef              core.ReferenciaEntradaCatalogo
	Motivo                 string
	SituacionEsperadaDesde time.Time
	ClaveIdempotencia      string
	BolsaRef               string
	UnidadRef              string
	AmbitoRef              string
	MotivoAutorizacion     core.ReferenciaEntradaCatalogo
	Vinculo                core.VinculoAutenticacionActorV2
	ResultadoContexto      core.ResultadoContextoActorRegistradoV2
	Correlacion            core.ReferenciaCorrelacionAutorizacionV2
}

func (s SolicitudResolverRespuestaRevision) Validar() error {
	if !referenciaRespuestaRevision.MatchString(s.RespuestaRef) ||
		!claveRespuestaRevision.MatchString(s.ClaveIdempotencia) ||
		!referenciaBolsaRevision.MatchString(s.BolsaRef) ||
		!referenciaBolsaRevision.MatchString(s.UnidadRef) ||
		!referenciaBolsaRevision.MatchString(s.AmbitoRef) ||
		s.MotivoRef.Validar() != nil ||
		!utf8.ValidString(s.Motivo) || len(s.Motivo) < 1 || len(s.Motivo) > 1000 ||
		strings.TrimSpace(s.Motivo) != s.Motivo ||
		!core.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) ||
		s.Resultado != dominiobolsa.ResultadoConfirmarAceptacion &&
			s.Resultado != dominiobolsa.ResultadoAdmitirJustificacion &&
			s.Resultado != dominiobolsa.ResultadoRechazarJustificacion ||
		s.SituacionEsperadaDesde.IsZero() || s.SituacionEsperadaDesde.Location() != time.UTC ||
		s.SituacionEsperadaDesde.Nanosecond()%1000 != 0 ||
		s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.Correlacion.Validar() != nil {
		return ErrResolucionRespuestaRevisionInvalida
	}
	return nil
}

// El canon incluye la referencia normativa completa, que SQL verifica antes
// del efecto. La huella semántica permite cotejar el cuerpo de un replay.
func CanonResolucionRespuestaRevision(s SolicitudResolverRespuestaRevision) (string, string, error) {
	if s.Validar() != nil {
		return "", "", ErrResolucionRespuestaRevisionInvalida
	}
	actor := s.ResultadoContexto.Contexto.PersonaRef
	estable := struct {
		Esquema                string                         `json:"esquema"`
		RespuestaRef           string                         `json:"respuesta_ref"`
		Resultado              string                         `json:"resultado"`
		MotivoRef              core.ReferenciaEntradaCatalogo `json:"motivo_ref"`
		Motivo                 string                         `json:"motivo"`
		SituacionEsperadaDesde string                         `json:"situacion_desde_esperada"`
		ClaveIdempotencia      string                         `json:"clave_idempotencia"`
		BolsaRef               string                         `json:"bolsa_ref"`
		ActorRef               string                         `json:"actor_ref"`
	}{EsquemaResolucionRespuestaRevision, s.RespuestaRef, s.Resultado, s.MotivoRef, s.Motivo,
		s.SituacionEsperadaDesde.Format("2006-01-02T15:04:05.000000Z"), s.ClaveIdempotencia, s.BolsaRef, actor}
	b, err := json.Marshal(estable)
	if err != nil {
		return "", "", ErrResolucionRespuestaRevisionInvalida
	}
	huella := sha256.Sum256(b)
	canon, err := json.Marshal(struct {
		Cuerpo    json.RawMessage `json:"cuerpo"`
		UnidadRef string          `json:"unidad_ref"`
		AmbitoRef string          `json:"ambito_ref"`
	}{b, s.UnidadRef, s.AmbitoRef})
	if err != nil || len(canon) > 65536 {
		return "", "", ErrResolucionRespuestaRevisionInvalida
	}
	return string(canon), hex.EncodeToString(huella[:]), nil
}

type OrdenResolucionRespuestaRevision struct {
	Solicitud             SolicitudResolverRespuestaRevision
	Canon                 string
	HuellaSemanticaSHA256 string
	Material              vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// El adaptador consume V3 y verifica el recibo ANTES de COMMIT. Luego deriva
// B30/B17, política y CAS dentro de la misma TX; en replay usa canon original.
type RegistroResolucionRespuestaRevision interface {
	ResolverRespuestaRevision(context.Context, OrdenResolucionRespuestaRevision) (ReciboResolucionRespuestaRevision, error)
}

type ReciboResolucionRespuestaRevision struct {
	RespuestaRef           string                                         `json:"respuesta_ref"`
	BolsaRef               string                                         `json:"bolsa_ref"`
	ActorRef               string                                         `json:"-"`
	Resultado              string                                         `json:"resultado"`
	MotivoRef              core.ReferenciaEntradaCatalogo                 `json:"motivo_ref"`
	Motivo                 string                                         `json:"motivo"`
	SituacionEsperadaDesde time.Time                                      `json:"situacion_desde_esperada"`
	HuellaSemanticaSHA256  string                                         `json:"-"`
	CanonOriginalSHA256    string                                         `json:"-"`
	ReciboRef              string                                         `json:"recibo_ref"`
	ResolucionRef          string                                         `json:"resolucion_ref"`
	AuditoriaRef           string                                         `json:"auditoria_ref"`
	EventoRef              string                                         `json:"evento_ref"`
	Situacion              string                                         `json:"situacion"`
	ResueltaEn             time.Time                                      `json:"resuelta_en"`
	Reutilizada            bool                                           `json:"reutilizada"`
	CambioSituacion        bool                                           `json:"cambio_situacion"`
	Hechos                 dominiobolsa.HechosResolucionRespuestaRevision `json:"-"`
	Regla                  dominiobolsa.ReglaResolucionRespuestaRevision  `json:"-"`
}

func (r ReciboResolucionRespuestaRevision) ValidarPara(o OrdenResolucionRespuestaRevision) error {
	if o.Solicitud.Validar() != nil || r.RespuestaRef != o.Solicitud.RespuestaRef ||
		r.BolsaRef != o.Solicitud.BolsaRef || r.ActorRef != o.Solicitud.ResultadoContexto.Contexto.PersonaRef ||
		r.Resultado != o.Solicitud.Resultado || r.MotivoRef != o.Solicitud.MotivoRef ||
		r.Motivo != o.Solicitud.Motivo ||
		!r.SituacionEsperadaDesde.Equal(o.Solicitud.SituacionEsperadaDesde) ||
		r.HuellaSemanticaSHA256 != o.HuellaSemanticaSHA256 ||
		!huellaRespuestaRevision.MatchString(r.CanonOriginalSHA256) ||
		!referenciaBolsaRevision.MatchString(r.ReciboRef) ||
		!referenciaBolsaRevision.MatchString(r.ResolucionRef) ||
		!referenciaBolsaRevision.MatchString(r.AuditoriaRef) ||
		!referenciaBolsaRevision.MatchString(r.EventoRef) || r.ResueltaEn.IsZero() ||
		r.ResueltaEn.Location() != time.UTC || r.ResueltaEn.Nanosecond()%1000 != 0 {
		return ErrResolucionRespuestaRevisionNoDisponible
	}
	if !r.Reutilizada {
		huella := sha256.Sum256([]byte(o.Canon))
		if r.CanonOriginalSHA256 != hex.EncodeToString(huella[:]) {
			return ErrResolucionRespuestaRevisionNoDisponible
		}
	}
	if r.Regla.CatalogoID != r.MotivoRef.CatalogoID ||
		r.Regla.CatalogoVersion != r.MotivoRef.CatalogoVersion ||
		r.Regla.CatalogoHuellaSHA256 != r.MotivoRef.CatalogoHuellaSHA256 {
		return ErrResolucionRespuestaRevisionNoDisponible
	}
	efecto, err := dominiobolsa.DerivarResolucionRespuestaRevision(r.Hechos, r.Regla, r.Resultado, r.MotivoRef.EntradaClave)
	if err != nil || efecto.Destino != r.Situacion || efecto.CambiaSituacion != r.CambioSituacion {
		return ErrResolucionRespuestaRevisionNoDisponible
	}
	return nil
}
