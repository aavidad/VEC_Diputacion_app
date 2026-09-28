package httpinterno

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const RutaCapacidadPoliticaOfertas = RutaPoliticaOfertas + "/capacidad"

const maximoCuerpoCapacidadPoliticaOfertas = 1024

// PreparadorCapacidadPoliticaOfertas consulta la autorización nominal de
// publicación para la bolsa exacta. No emite material consumible: publicar
// vuelve a autorizar la operación antes de cualquier efecto.
type PreparadorCapacidadPoliticaOfertas interface {
	ComprobarCapacidadPublicarPoliticaOfertas(context.Context, string) (bool, error)
}

type HandlerCapacidadPoliticaOfertas struct {
	preparador PreparadorCapacidadPoliticaOfertas
}

func NuevoHandlerCapacidadPoliticaOfertas(p PreparadorCapacidadPoliticaOfertas) (http.Handler, error) {
	if p == nil {
		return nil, ports.ErrPoliticaOfertasNoDisponible
	}
	return &HandlerCapacidadPoliticaOfertas{preparador: p}, nil
}

func (h *HandlerCapacidadPoliticaOfertas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.preparador == nil || r == nil || r.URL == nil {
		responderOferta(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if r.URL.Path != RutaCapacidadPoliticaOfertas || r.URL.RawPath != "" || r.URL.EscapedPath() != RutaCapacidadPoliticaOfertas {
		responderOferta(w, http.StatusNotFound, "no_encontrado", nil)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		responderOferta(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Body == nil || r.Body == http.NoBody ||
		r.ContentLength < 1 || r.ContentLength > maximoCuerpoCapacidadPoliticaOfertas || len(r.TransferEncoding) != 0 ||
		len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" ||
		len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" ||
		cabeceraPresente(r.Header, "Cookie") || cabeceraPresente(r.Header, "Proxy-Authorization") ||
		cabeceraPresente(r.Header, "Authorization") || cabeceraIdentidadHeredadaPresente(r.Header) ||
		cabeceraPresente(r.Header, "Idempotency-Key") {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	bolsa, ok := leerBolsaCapacidadPoliticaOfertas(r.Body)
	if !ok {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	puedePublicar, err := h.preparador.ComprobarCapacidadPublicarPoliticaOfertas(r.Context(), bolsa)
	if err != nil {
		responderErrorPoliticaOfertas(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		PuedePublicar bool `json:"puede_publicar"`
	}{PuedePublicar: puedePublicar})
}

func leerBolsaCapacidadPoliticaOfertas(cuerpo io.Reader) (string, bool) {
	d := json.NewDecoder(io.LimitReader(cuerpo, maximoCuerpoCapacidadPoliticaOfertas+1))
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return "", false
	}
	clave, err := d.Token()
	if err != nil || clave != "bolsa_ref" {
		return "", false
	}
	var bolsa string
	if err = d.Decode(&bolsa); err != nil || bolsa == "" || len(bolsa) > 256 || strings.TrimSpace(bolsa) != bolsa {
		return "", false
	}
	fin, err := d.Token()
	if err != nil || fin != json.Delim('}') {
		return "", false
	}
	_, err = d.Token()
	return bolsa, err == io.EOF
}
