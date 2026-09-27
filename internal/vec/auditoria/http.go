package auditoria

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const maximoCuerpoConsulta = 8 * 1024

// IdentidadConsulta resuelve únicamente desde la frontera de identidad del
// servidor. El navegador nunca proporciona actor, vínculo, perfil o correlación.
type IdentidadConsulta interface {
	ResolverIdentidadConsulta(context.Context, *http.Request) (IdentidadResuelta, error)
}

type IdentidadResuelta struct {
	Vinculo     vecdomain.VinculoAutenticacionActorV2
	Resultado   vecdomain.ResultadoContextoActorRegistradoV2
	Correlacion vecdomain.ReferenciaCorrelacionAutorizacionV2
}

type Manejador struct {
	servicio  *Servicio
	opciones  ProveedorOpciones
	identidad IdentidadConsulta
	ahora     func() time.Time
}

func NuevoManejador(servicio *Servicio, opciones ProveedorOpciones, identidad IdentidadConsulta) (*Manejador, error) {
	if servicio == nil || dependenciaNula(opciones) || dependenciaNula(identidad) {
		return nil, ErrNoDisponible
	}
	return &Manejador{servicio: servicio, opciones: opciones, identidad: identidad, ahora: time.Now}, nil
}

func (h *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || r == nil || h.servicio == nil || dependenciaNula(h.opciones) || dependenciaNula(h.identidad) {
		responderError(w, http.StatusServiceUnavailable)
		return
	}
	if r.URL.RawQuery != "" || r.URL.Fragment != "" || r.Header.Get("Cookie") != "" {
		responderError(w, http.StatusForbidden)
		return
	}
	switch r.URL.Path {
	case RutaOpciones:
		if r.Method != http.MethodGet {
			responderError(w, http.StatusMethodNotAllowed)
			return
		}
		h.servirOpciones(w, r)
	case RutaConsulta:
		if r.Method != http.MethodPost {
			responderError(w, http.StatusMethodNotAllowed)
			return
		}
		h.servirConsulta(w, r)
	default:
		responderError(w, http.StatusNotFound)
	}
}

func (h *Manejador) resolverIdentidad(r *http.Request) (IdentidadResuelta, error) {
	identidad, err := h.identidad.ResolverIdentidadConsulta(r.Context(), r)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil ||
		!identidad.Vinculo.VigenteEn(h.ahora().UTC().Truncate(time.Microsecond), identidad.Resultado) || identidad.Correlacion.Validar() != nil {
		return IdentidadResuelta{}, ErrDenegada
	}
	return identidad, nil
}

func (h *Manejador) servirOpciones(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolverIdentidad(r); err != nil {
		responderError(w, http.StatusForbidden)
		return
	}
	opciones, err := h.opciones.Actuales(r.Context())
	if err != nil {
		responderError(w, http.StatusServiceUnavailable)
		return
	}
	responderJSON(w, http.StatusOK, opciones)
}

type cuerpoConsulta struct {
	Fuente        string `json:"fuente"`
	ExpedienteRef string `json:"expediente_ref"`
	ActorRef      string `json:"actor_ref"`
	Desde         string `json:"desde"`
	Hasta         string `json:"hasta"`
	Limite        uint16 `json:"limite"`
	Cursor        string `json:"cursor"`
	FinalidadRef  string `json:"finalidad_ref"`
	MotivoRef     string `json:"motivo_ref"`
}

func (h *Manejador) servirConsulta(w http.ResponseWriter, r *http.Request) {
	identidad, err := h.resolverIdentidad(r)
	if err != nil {
		responderError(w, http.StatusForbidden)
		return
	}
	ct := r.Header.Get("Content-Type")
	if ct != "application/json" && ct != "application/json; charset=utf-8" {
		responderError(w, http.StatusBadRequest)
		return
	}
	if r.ContentLength > maximoCuerpoConsulta {
		responderError(w, http.StatusBadRequest)
		return
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximoCuerpoConsulta))
	dec.DisallowUnknownFields()
	var cuerpo cuerpoConsulta
	if dec.Decode(&cuerpo) != nil || dec.Decode(new(any)) != io.EOF {
		responderError(w, http.StatusBadRequest)
		return
	}
	if cuerpo.Limite == 0 {
		cuerpo.Limite = 50
	}
	desde, errDesde := parsearInstante(cuerpo.Desde)
	hasta, errHasta := parsearInstante(cuerpo.Hasta)
	if errDesde != nil || errHasta != nil {
		responderError(w, http.StatusBadRequest)
		return
	}
	opciones, err := h.opciones.Actuales(r.Context())
	if err != nil {
		responderError(w, http.StatusServiceUnavailable)
		return
	}
	if cuerpo.FinalidadRef != opciones.FinalidadRef || cuerpo.MotivoRef != opciones.MotivoRef {
		responderError(w, http.StatusForbidden)
		return
	}
	f := Filtro{Fuente: cuerpo.Fuente, ExpedienteRef: cuerpo.ExpedienteRef, ActorRef: cuerpo.ActorRef,
		Desde: desde, Hasta: hasta, Limite: cuerpo.Limite,
		FinalidadRef: cuerpo.FinalidadRef, MotivoRef: cuerpo.MotivoRef}
	if f.Validar() != nil {
		responderError(w, http.StatusForbidden)
		return
	}
	pagina, err := h.servicio.Consultar(r.Context(), Peticion{Filtro: f, Cursor: cuerpo.Cursor, Contexto: ContextoConsulta{
		Vinculo: identidad.Vinculo, Resultado: identidad.Resultado,
		Motivo: opciones.Motivo, Correlacion: identidad.Correlacion,
	}})
	if err != nil {
		if errors.Is(err, ErrDenegada) {
			responderError(w, http.StatusForbidden)
		} else {
			responderError(w, http.StatusServiceUnavailable)
		}
		return
	}
	responderJSON(w, http.StatusOK, pagina)
}

func parsearInstante(s string) (time.Time, error) {
	if s == "" || !strings.HasSuffix(s, "Z") {
		return time.Time{}, ErrDenegada
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || t.Nanosecond()%1000 != 0 {
		return time.Time{}, ErrDenegada
	}
	return t.UTC(), nil
}

func responderJSON(w http.ResponseWriter, codigo int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		responderError(w, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write(append(b, '\n'))
}

func responderError(w http.ResponseWriter, codigo int) {
	texto := "consulta no disponible"
	if codigo == http.StatusForbidden {
		texto = "consulta denegada"
	}
	if codigo == http.StatusBadRequest {
		texto = "solicitud invalida"
	}
	if codigo == http.StatusNotFound {
		texto = "ruta no encontrada"
	}
	if codigo == http.StatusMethodNotAllowed {
		texto = "metodo no admitido"
	}
	responderJSONSeguro(w, codigo, map[string]string{"error": texto})
}
func responderJSONSeguro(w http.ResponseWriter, codigo int, v any) {
	b := new(bytes.Buffer)
	_ = json.NewEncoder(b).Encode(v)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write(b.Bytes())
}
