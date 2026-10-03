package auditoria

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// IdentidadConsultaNominal is injected by the existing trusted ADMIN boundary.
// It receives an already emptied HTTP body and never accepts browser identity.
type IdentidadConsultaNominal interface {
	ResolverContextoConsultaNominal(context.Context, *http.Request) (ContextoConsulta, error)
}

type ManejadorNominal struct {
	servicio  *ServicioNominal
	opciones  ProveedorOpciones
	identidad IdentidadConsultaNominal
}

func NuevoManejadorNominal(s *ServicioNominal, o ProveedorOpciones, i IdentidadConsultaNominal) (*ManejadorNominal, error) {
	if s == nil || dependenciaNula(o) || dependenciaNula(i) {
		return nil, ErrNoDisponible
	}
	return &ManejadorNominal{servicio: s, opciones: o, identidad: i}, nil
}

func (h *ManejadorNominal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || r == nil || r.URL == nil || h.servicio == nil || dependenciaNula(h.opciones) || dependenciaNula(h.identidad) {
		responderErrorNominal(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.URL.Path != RutaConsultaNominal {
		responderErrorNominal(w, http.StatusNotFound, "ruta_no_admitida")
		return
	}
	if r.Method != http.MethodPost {
		responderErrorNominal(w, http.StatusMethodNotAllowed, "metodo_no_admitido")
		return
	}
	if r.URL.RawQuery != "" || r.URL.Fragment != "" || contieneCabeceraCookie(r.Header) {
		responderErrorNominal(w, http.StatusForbidden, "entrada_no_admitida")
		return
	}
	c, err := decodificarCuerpoNominal(w, r)
	if err != nil {
		responderErrorNominal(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	desde, errDesde := parsearInstante(c.Desde)
	hasta, errHasta := parsearInstante(c.Hasta)
	f := FiltroNominal{ActorRef: c.ActorRef, RecursoRef: c.RecursoRef, Accion: c.Accion, Desde: desde, Hasta: hasta,
		Limite: c.Limite, FinalidadRef: c.FinalidadRef, MotivoRef: c.MotivoRef}
	if errDesde != nil || errHasta != nil || f.Validar() != nil {
		responderErrorNominal(w, http.StatusBadRequest, "filtro_invalido")
		return
	}
	contexto, err := h.identidad.ResolverContextoConsultaNominal(r.Context(), r)
	if err != nil {
		responderErrorNominal(w, http.StatusForbidden, "consulta_denegada")
		return
	}
	o, err := h.opciones.Actuales(r.Context())
	if err != nil {
		responderErrorNominal(w, http.StatusServiceUnavailable, "catalogo_no_disponible")
		return
	}
	if o.PermisoRequerido != AccionConsultarNominal || c.FinalidadRef != o.FinalidadRef || c.MotivoRef != o.MotivoRef ||
		contexto.Motivo != o.Motivo {
		responderErrorNominal(w, http.StatusForbidden, "consulta_denegada")
		return
	}
	p, err := h.servicio.Consultar(r.Context(), PeticionNominal{Filtro: f, Cursor: c.Cursor, Contexto: contexto})
	if err != nil {
		if errors.Is(err, ErrDenegada) {
			responderErrorNominal(w, http.StatusForbidden, "consulta_denegada")
		} else {
			responderErrorNominal(w, http.StatusServiceUnavailable, "consulta_no_disponible")
		}
		return
	}
	responderJSON(w, http.StatusOK, p)
}

type cuerpoNominal struct {
	ActorRef     string
	RecursoRef   string
	Accion       string
	Desde        string
	Hasta        string
	Limite       uint16
	Cursor       string
	FinalidadRef string
	MotivoRef    string
}

func decodificarCuerpoNominal(w http.ResponseWriter, r *http.Request) (cuerpoNominal, error) {
	var c cuerpoNominal
	tipo := r.Header.Get("Content-Type")
	if (tipo != "application/json" && tipo != "application/json; charset=utf-8") || r.ContentLength > maximoCuerpoConsulta || r.Body == nil {
		return c, ErrDenegada
	}
	lector := http.MaxBytesReader(w, r.Body, maximoCuerpoConsulta)
	defer lector.Close()
	d := json.NewDecoder(lector)
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return c, ErrDenegada
	}
	vistas := map[string]bool{}
	for d.More() {
		t, err := d.Token()
		clave, ok := t.(string)
		if err != nil || !ok || vistas[clave] {
			return c, ErrDenegada
		}
		vistas[clave] = true
		var destino *string
		switch clave {
		case "actor_ref":
			destino = &c.ActorRef
		case "recurso_ref":
			destino = &c.RecursoRef
		case "accion":
			destino = &c.Accion
		case "desde":
			destino = &c.Desde
		case "hasta":
			destino = &c.Hasta
		case "cursor":
			destino = &c.Cursor
		case "finalidad_ref":
			destino = &c.FinalidadRef
		case "motivo_ref":
			destino = &c.MotivoRef
		case "limite":
			var v *uint16
			if d.Decode(&v) != nil || v == nil {
				return c, ErrDenegada
			}
			c.Limite = *v
			continue
		default:
			return c, ErrDenegada
		}
		var v *string
		if d.Decode(&v) != nil || v == nil {
			return c, ErrDenegada
		}
		*destino = *v
	}
	fin, err := d.Token()
	if err != nil || fin != json.Delim('}') || !vistas["desde"] || !vistas["hasta"] || !vistas["limite"] ||
		!vistas["finalidad_ref"] || !vistas["motivo_ref"] {
		return c, ErrDenegada
	}
	if d.Decode(new(any)) != io.EOF {
		return c, ErrDenegada
	}
	r.Body = http.NoBody
	r.GetBody = nil
	r.ContentLength = 0
	r.TransferEncoding = nil
	return c, nil
}

func responderErrorNominal(w http.ResponseWriter, status int, codigo string) {
	// Protocol codes are translated by the UI catalogue; no human text lives here.
	responderJSONSeguro(w, status, map[string]string{"codigo": codigo})
}
