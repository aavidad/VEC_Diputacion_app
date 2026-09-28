package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	bolsa, err := leerBolsaCapacidadPoliticaOfertas(r.Body)
	if err != nil {
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

func leerBolsaCapacidadPoliticaOfertas(cuerpo io.Reader) (string, error) {
	d := json.NewDecoder(io.LimitReader(cuerpo, maximoCuerpoCapacidadPoliticaOfertas+1))
	inicio, err := d.Token()
	if err != nil {
		return "", fmt.Errorf("leer inicio de solicitud: %w", err)
	}
	if inicio != json.Delim('{') {
		return "", errors.New("la solicitud no es un objeto")
	}
	clave, err := d.Token()
	if err != nil {
		return "", fmt.Errorf("leer clave de solicitud: %w", err)
	}
	if clave != "bolsa_ref" {
		return "", errors.New("la solicitud no empieza por bolsa_ref")
	}
	var bolsa string
	if err = d.Decode(&bolsa); err != nil {
		return "", fmt.Errorf("leer bolsa_ref: %w", err)
	}
	if bolsa == "" || len(bolsa) > 256 || strings.TrimSpace(bolsa) != bolsa {
		return "", errors.New("bolsa_ref no es canonica")
	}
	fin, err := d.Token()
	if err != nil {
		return "", fmt.Errorf("leer fin de solicitud: %w", err)
	}
	if fin != json.Delim('}') {
		return "", errors.New("la solicitud contiene otro campo")
	}
	_, err = d.Token()
	if err == io.EOF {
		return bolsa, nil
	}
	if err != nil {
		return "", fmt.Errorf("leer resto de solicitud: %w", err)
	}
	return "", errors.New("la solicitud contiene otro valor")
}
