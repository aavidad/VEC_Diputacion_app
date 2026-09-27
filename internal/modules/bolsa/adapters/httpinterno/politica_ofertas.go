package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const RutaPoliticaOfertas = "/api/vec/bolsa/politica-ofertas"

type EntradaPublicarPoliticaOfertas struct {
	BolsaRef          string
	VersionEsperada   int64
	ClaveIdempotencia string
	Politica          domain.PoliticaOfertas
}

type ConsultaPoliticaOfertasPreparada struct {
	BolsaRef      string
	PuedePublicar bool // decisión positiva exacta del PDP central, nunca del rol de shell
	Material      ports.ConsultaPoliticaOfertasAutorizada
}

// El preparador deriva actor, ámbito y material V3 de la sesión confiable.
// GET exige la autorización nominal AD3-97 de la bolsa exacta.
type PreparadorPoliticaOfertas interface {
	PrepararConsultaPoliticaOfertas(context.Context, string) (ConsultaPoliticaOfertasPreparada, error)
	PrepararPublicacionPoliticaOfertas(context.Context, EntradaPublicarPoliticaOfertas) (ports.ComandoPublicarPoliticaOfertas, error)
}

type OperadorPoliticaOfertas interface {
	ConsultarAutorizada(context.Context, ports.ConsultaPoliticaOfertasAutorizada) (ports.VersionPoliticaOfertas, error)
	Publicar(context.Context, ports.ComandoPublicarPoliticaOfertas) (ports.VersionPoliticaOfertas, error)
}

type HandlerPoliticaOfertas struct {
	preparador PreparadorPoliticaOfertas
	operador   OperadorPoliticaOfertas
}

func NuevoHandlerPoliticaOfertas(p PreparadorPoliticaOfertas, o OperadorPoliticaOfertas) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, ports.ErrPoliticaOfertasNoDisponible
	}
	return &HandlerPoliticaOfertas{p, o}, nil
}

func (h *HandlerPoliticaOfertas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || r.URL == nil || r.URL.Path != RutaPoliticaOfertas || r.URL.RawPath != "" || r.URL.ForceQuery {
		responderOferta(w, http.StatusNotFound, "no_encontrado", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.consultar(w, r)
	case http.MethodPost:
		h.publicar(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		responderOferta(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
	}
}

func (h *HandlerPoliticaOfertas) consultar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	bolsa := q.Get("bolsa_ref")
	if len(q) != 1 || len(q["bolsa_ref"]) != 1 || bolsa == "" || len(bolsa) > 256 || strings.TrimSpace(bolsa) != bolsa || r.ContentLength != 0 {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	resuelta, err := h.preparador.PrepararConsultaPoliticaOfertas(r.Context(), bolsa)
	if err != nil {
		responderErrorPoliticaOfertas(w, err)
		return
	}
	if resuelta.BolsaRef != bolsa || resuelta.Material.BolsaRef != bolsa {
		responderOferta(w, http.StatusForbidden, "acceso_denegado", nil)
		return
	}
	v, err := h.operador.ConsultarAutorizada(r.Context(), resuelta.Material)
	if err != nil {
		responderErrorPoliticaOfertas(w, err)
		return
	}
	salida := salidaPoliticaOfertas(v)
	salida["puede_publicar"] = resuelta.PuedePublicar
	responderOferta(w, http.StatusOK, "", salida)
}

func (h *HandlerPoliticaOfertas) publicar(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" || r.ContentLength < 1 || r.ContentLength > maximoCuerpoOferta || len(r.TransferEncoding) != 0 ||
		r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	var cuerpo struct {
		BolsaRef          string                 `json:"bolsa_ref"`
		VersionEsperada   *int64                 `json:"version_esperada"`
		ClaveIdempotencia string                 `json:"clave_idempotencia"`
		Politica          domain.PoliticaOfertas `json:"politica"`
	}
	if !decodificarCuerpoOferta(r, &cuerpo) || cuerpo.VersionEsperada == nil ||
		cuerpo.BolsaRef == "" || cuerpo.ClaveIdempotencia == "" || cuerpo.Politica.Validar() != nil {
		responderOferta(w, http.StatusUnprocessableEntity, "politica_invalida", nil)
		return
	}
	c, err := h.preparador.PrepararPublicacionPoliticaOfertas(r.Context(), EntradaPublicarPoliticaOfertas{
		BolsaRef: cuerpo.BolsaRef, VersionEsperada: *cuerpo.VersionEsperada,
		ClaveIdempotencia: cuerpo.ClaveIdempotencia, Politica: cuerpo.Politica,
	})
	if err != nil {
		responderErrorPoliticaOfertas(w, err)
		return
	}
	if c.BolsaRef != cuerpo.BolsaRef || c.VersionEsperada != *cuerpo.VersionEsperada ||
		c.ClaveIdempotencia != cuerpo.ClaveIdempotencia || c.Politica != cuerpo.Politica {
		responderOferta(w, http.StatusForbidden, "acceso_denegado", nil)
		return
	}
	v, err := h.operador.Publicar(r.Context(), c)
	if err != nil {
		responderErrorPoliticaOfertas(w, err)
		return
	}
	estado := http.StatusCreated
	if v.Reutilizada {
		estado = http.StatusOK
	}
	responderOferta(w, estado, "", salidaPoliticaOfertas(v))
}

func salidaPoliticaOfertas(v ports.VersionPoliticaOfertas) map[string]any {
	salida := map[string]any{
		"esquema": ports.EsquemaPoliticaOfertas, "bolsa_ref": v.BolsaRef,
		"version": v.Version,
		"ejemplo": v.Ejemplo, "configurada": v.Configurada,
		"politica": v.Politica,
	}
	if v.HuellaSHA256 != "" {
		salida["huella_sha256"] = v.HuellaSHA256
	}
	if v.ReciboRef != "" {
		salida["recibo_ref"] = v.ReciboRef
	}
	if !v.PublicadaEn.IsZero() {
		salida["publicada_en"] = v.PublicadaEn
	}
	if v.Reutilizada {
		salida["reutilizada"] = true
	}
	return salida
}

func responderErrorPoliticaOfertas(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderOferta(w, http.StatusForbidden, "acceso_denegado", nil)
	case errors.Is(err, ports.ErrPoliticaOfertasConflicto):
		responderOferta(w, http.StatusConflict, "version_o_clave_en_conflicto", nil)
	case errors.Is(err, domain.ErrPoliticaOfertasInvalida):
		responderOferta(w, http.StatusUnprocessableEntity, "politica_invalida", nil)
	default:
		responderOferta(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}
