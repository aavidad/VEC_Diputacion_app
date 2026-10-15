package domain

import (
	"errors"
	"slices"
	"strings"
)

var ErrResolucionRespuestaRevisionInvalida = errors.New("bolsa: resolucion de respuesta en revision invalida")

const (
	ResultadoConfirmarAceptacion    = "confirmar_aceptacion"
	ResultadoAdmitirJustificacion   = "admitir_justificacion"
	ResultadoRechazarJustificacion  = "rechazar_justificacion"
	ProcedimientoLlamamientoDirecto = "directo"
)

// HechosResolucionRespuestaRevision procede de Bolsa después de consumir V3.
// La marca pendiente B100 no se sustituye por una situación "en_revision".
type HechosResolucionRespuestaRevision struct {
	Respuesta             string
	Procedimiento         string
	SituacionActual       string
	Causa                 string
	JustificanteRef       string
	JustificanteSHA256    string
	RespuestaPendiente    bool
	LlamamientoAcreditado bool
	JustificanteRevisado  bool
}

// ReglaResolucionRespuestaRevision es una entrada publicada, congelada con
// su huella. La política de situación publicada puede cerrar el par real.
type ReglaResolucionRespuestaRevision struct {
	CatalogoID           string
	CatalogoVersion      int
	CatalogoHuellaSHA256 string
	MotivoClave          string
	Respuesta            string
	Resultado            string
	Procedimiento        string
	Destino              string
	Publicada            bool
	PoliticaVersion      int64
	PoliticaHuellaSHA256 string
	PoliticaPublicada    bool
	Politica             PoliticaTransicionesSituacion
}

type EfectoResolucionRespuestaRevision struct {
	Destino         string
	CambiaSituacion bool
}

// DerivarResolucionRespuestaRevision limita el primer corte al directo B17.
// El catálogo publicado decide si existe una regla; esta envolvente impide
// que una entrada mal publicada amplíe los tres resultados autorizados.
func DerivarResolucionRespuestaRevision(h HechosResolucionRespuestaRevision,
	r ReglaResolucionRespuestaRevision, resultado, motivoClave string,
) (EfectoResolucionRespuestaRevision, error) {
	vacio := EfectoResolucionRespuestaRevision{}
	if !h.RespuestaPendiente || !h.LlamamientoAcreditado ||
		h.Procedimiento != ProcedimientoLlamamientoDirecto ||
		r.Procedimiento != h.Procedimiento || !r.Publicada ||
		r.CatalogoID == "" || r.CatalogoVersion < 1 || !huellaRevisionValida(r.CatalogoHuellaSHA256) ||
		r.PoliticaVersion < 1 || !huellaRevisionValida(r.PoliticaHuellaSHA256) || !r.PoliticaPublicada ||
		r.MotivoClave == "" || r.MotivoClave != motivoClave ||
		r.Respuesta != h.Respuesta || r.Resultado != resultado ||
		!slices.Contains(SituacionesParticipacion(), h.SituacionActual) {
		return vacio, ErrResolucionRespuestaRevisionInvalida
	}
	destino := ""
	switch {
	case h.Respuesta == "acepta" && resultado == ResultadoConfirmarAceptacion &&
		h.Causa == "" && h.JustificanteRef == "" && h.JustificanteSHA256 == "":
		destino = SituacionPendienteIncorporacion
	case h.Respuesta == "renuncia_justificada" && resultado == ResultadoAdmitirJustificacion &&
		h.Causa != "" && h.JustificanteRef != "" && huellaRevisionValida(h.JustificanteSHA256) && h.JustificanteRevisado:
		destino = SituacionDisponible
	case h.Respuesta == "renuncia_justificada" && resultado == ResultadoRechazarJustificacion &&
		h.Causa != "" && h.JustificanteRef != "" && huellaRevisionValida(h.JustificanteSHA256) && h.JustificanteRevisado:
		destino = SituacionExcluido
	default:
		return vacio, ErrResolucionRespuestaRevisionInvalida
	}
	if r.Destino != destino {
		return vacio, ErrResolucionRespuestaRevisionInvalida
	}
	if h.SituacionActual == destino {
		// Admitir una renuncia puede mantener "disponible". La resolución
		// durable debe apagar B100/B101; no se inventa un par S6 a sí mismo.
		if resultado != ResultadoAdmitirJustificacion {
			return vacio, ErrResolucionRespuestaRevisionInvalida
		}
		return EfectoResolucionRespuestaRevision{Destino: destino}, nil
	}
	if !r.Politica.Admite(h.SituacionActual, destino) {
		return vacio, ErrResolucionRespuestaRevisionInvalida
	}
	return EfectoResolucionRespuestaRevision{Destino: destino, CambiaSituacion: true}, nil
}

func huellaRevisionValida(s string) bool {
	if len(s) != 64 {
		return false
	}
	return strings.IndexFunc(s, func(c rune) bool { return c < '0' || c > '9' && (c < 'a' || c > 'f') }) == -1
}
