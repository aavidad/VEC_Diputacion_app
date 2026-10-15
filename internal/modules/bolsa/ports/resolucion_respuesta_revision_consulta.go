package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaConsultaRespuestasRevision = "vec.bolsa.respuesta-revision.consultar.v1"

// La bolsa es filtro de la ruta RRHH; la base verifica de nuevo que cada
// respuesta pertenezca al ámbito firmado antes de devolverla.
type SolicitudConsultarRespuestasRevision struct {
	BolsaRef           string
	Limite             int
	Cursor             string
	UnidadRef          string
	AmbitoRef          string
	MotivoAutorizacion core.ReferenciaEntradaCatalogo
	Vinculo            core.VinculoAutenticacionActorV2
	ResultadoContexto  core.ResultadoContextoActorRegistradoV2
	Correlacion        core.ReferenciaCorrelacionAutorizacionV2
}

func (s SolicitudConsultarRespuestasRevision) Validar() error {
	if !referenciaBolsaRevision.MatchString(s.BolsaRef) || s.Limite < 1 || s.Limite > 50 ||
		len(s.Cursor) > 256 || strings.IndexFunc(s.Cursor, func(r rune) bool {
		return r < 33 || r > 126
	}) >= 0 || !referenciaBolsaRevision.MatchString(s.UnidadRef) ||
		!referenciaBolsaRevision.MatchString(s.AmbitoRef) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) ||
		s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.Correlacion.Validar() != nil {
		return ErrResolucionRespuestaRevisionInvalida
	}
	return nil
}

func CanonConsultaRespuestasRevision(s SolicitudConsultarRespuestasRevision) (string, error) {
	if s.Validar() != nil {
		return "", ErrResolucionRespuestaRevisionInvalida
	}
	b, err := json.Marshal(struct {
		Esquema   string `json:"esquema"`
		BolsaRef  string `json:"bolsa_ref"`
		Cursor    string `json:"cursor"`
		ActorRef  string `json:"actor_ref"`
		PerfilRef string `json:"perfil_ref"`
		UnidadRef string `json:"unidad_ref"`
		AmbitoRef string `json:"ambito_ref"`
		Limite    int    `json:"limite"`
	}{EsquemaConsultaRespuestasRevision, s.BolsaRef, s.Cursor,
		s.ResultadoContexto.Contexto.PersonaRef, s.ResultadoContexto.Contexto.PerfilActivoRef,
		s.UnidadRef, s.AmbitoRef, s.Limite})
	if err != nil || len(b) > 65536 {
		return "", ErrResolucionRespuestaRevisionInvalida
	}
	return string(b), nil
}

type OrdenConsultaRespuestasRevision struct {
	Solicitud SolicitudConsultarRespuestasRevision
	Canon     string
	Material  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// Las opciones ya proceden de norma y política publicadas. HTTP no construye
// ni cambia la referencia de motivo o el destino.
type OpcionRespuestaRevision struct {
	Resultado string                         `json:"resultado"`
	MotivoRef core.ReferenciaEntradaCatalogo `json:"motivo_ref"`
	Destino   string                         `json:"destino"`
}

type RespuestaEnRevisionRRHH struct {
	RespuestaRef           string                    `json:"respuesta_ref"`
	BolsaRef               string                    `json:"bolsa_ref"`
	LlamamientoRef         string                    `json:"llamamiento_ref"`
	ParticipacionRef       string                    `json:"participacion_ref"`
	OrdenActa              int64                     `json:"orden_acta"`
	Respuesta              string                    `json:"respuesta"`
	Procedimiento          string                    `json:"procedimiento"`
	Causa                  string                    `json:"causa,omitempty"`
	JustificanteRef        string                    `json:"justificante_ref,omitempty"`
	JustificanteSHA256     string                    `json:"justificante_sha256,omitempty"`
	RespondidaEn           time.Time                 `json:"respondida_en"`
	SituacionEsperadaDesde time.Time                 `json:"situacion_desde_esperada"`
	SituacionActual        string                    `json:"situacion_actual"`
	Opciones               []OpcionRespuestaRevision `json:"opciones"`
}

type PaginaRespuestasRevision struct {
	BolsaRef        string                    `json:"bolsa_ref"`
	Filas           []RespuestaEnRevisionRRHH `json:"filas"`
	CursorSiguiente string                    `json:"cursor_siguiente,omitempty"`
	AuditoriaRef    string                    `json:"auditoria_ref"`
}

func (p PaginaRespuestasRevision) ValidarPara(o OrdenConsultaRespuestasRevision) error {
	if p.BolsaRef != o.Solicitud.BolsaRef || p.Filas == nil || len(p.Filas) > o.Solicitud.Limite ||
		len(p.CursorSiguiente) > 256 || strings.IndexFunc(p.CursorSiguiente, func(r rune) bool {
		return r < 33 || r > 126
	}) >= 0 || p.AuditoriaRef == "" {
		return ErrResolucionRespuestaRevisionNoDisponible
	}
	vistas := make(map[string]bool, len(p.Filas))
	for _, fila := range p.Filas {
		if !referenciaRespuestaRevision.MatchString(fila.RespuestaRef) || vistas[fila.RespuestaRef] ||
			fila.BolsaRef != p.BolsaRef || fila.LlamamientoRef == "" || fila.ParticipacionRef == "" ||
			fila.OrdenActa < 1 ||
			fila.RespondidaEn.IsZero() || fila.SituacionEsperadaDesde.IsZero() ||
			!slices.Contains([]string{"acepta", "renuncia", "renuncia_justificada"}, fila.Respuesta) ||
			fila.Procedimiento != dominiobolsa.ProcedimientoLlamamientoDirecto ||
			!slices.Contains(dominiobolsa.SituacionesParticipacion(), fila.SituacionActual) ||
			fila.Opciones == nil || fila.Respuesta == "renuncia" && len(fila.Opciones) != 0 {
			return ErrResolucionRespuestaRevisionNoDisponible
		}
		vistasOpciones := make(map[struct {
			Resultado string
			MotivoRef core.ReferenciaEntradaCatalogo
		}]bool, len(fila.Opciones))
		for _, opcion := range fila.Opciones {
			clave := struct {
				Resultado string
				MotivoRef core.ReferenciaEntradaCatalogo
			}{opcion.Resultado, opcion.MotivoRef}
			if vistasOpciones[clave] || opcion.MotivoRef.Validar() != nil ||
				!opcionRevisionCompatible(fila.Respuesta, opcion) ||
				fila.SituacionActual == opcion.Destino &&
					opcion.Resultado != dominiobolsa.ResultadoAdmitirJustificacion {
				return ErrResolucionRespuestaRevisionNoDisponible
			}
			vistasOpciones[clave] = true
		}
		vistas[fila.RespuestaRef] = true
	}
	return nil
}

func opcionRevisionCompatible(respuesta string, o OpcionRespuestaRevision) bool {
	switch respuesta {
	case "acepta":
		return o.Resultado == dominiobolsa.ResultadoConfirmarAceptacion && o.Destino == dominiobolsa.SituacionPendienteIncorporacion
	case "renuncia_justificada":
		return o.Resultado == dominiobolsa.ResultadoAdmitirJustificacion && o.Destino == dominiobolsa.SituacionDisponible ||
			o.Resultado == dominiobolsa.ResultadoRechazarJustificacion && o.Destino == dominiobolsa.SituacionExcluido
	default:
		return false
	}
}

// La implementación consume una V3 nominal y deja auditoría antes de leer
// filas B30. No expone datos si hay denegación o lectura parcial.
type LectorRespuestasRevision interface {
	ConsultarRespuestasRevision(context.Context, OrdenConsultaRespuestasRevision) (PaginaRespuestasRevision, error)
}

func HuellaCanonRespuestasRevision(canon string) string {
	h := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(h[:])
}
