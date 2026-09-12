package contactopropio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/ports"
)

const RutaReciboContactoPropio = "/api/vec/usuarios/contacto-propio/recibo"

// EjecutorReciboContactoPropio limita el transporte a una consulta ya
// autorizada por el servicio. No acepta sujeto, perfil ni autoridad del cliente.
type EjecutorReciboContactoPropio interface {
	ConsultarRecibo(context.Context, uint64) (ports.ResultadoConsultaReciboContactoUsuario, error)
}

type manejadorReciboContactoPropio struct {
	ejecutor EjecutorReciboContactoPropio
	catalogo *i18n.Catalog
}

type entradaReciboContactoPropio struct {
	Version uint64 `json:"version"`
}

var _ http.Handler = (*manejadorReciboContactoPropio)(nil)

func NuevoManejadorRecibo(ejecutor EjecutorReciboContactoPropio, catalogo *i18n.Catalog) (http.Handler, error) {
	if dependenciaContactoPropioNula(ejecutor) || catalogo == nil {
		return nil, ErrManejadorContactoPropioInvalido
	}
	return &manejadorReciboContactoPropio{ejecutor: ejecutor, catalogo: catalogo}, nil
}

func (h *manejadorReciboContactoPropio) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaContactoPropioNula(h.ejecutor) || h.catalogo == nil {
		responderContactoPropio(w, nil, http.StatusForbidden, claveErrorAccesoDenegado)
		return
	}
	if !rutaReciboContactoPropioExacta(r) {
		responderContactoPropio(w, h.catalogo, http.StatusNotFound, claveErrorNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderContactoPropio(w, h.catalogo, http.StatusMethodNotAllowed, claveErrorPeticion)
		return
	}
	if !esJSONContactoPropio(r.Header.Get("Content-Type")) {
		responderContactoPropio(w, h.catalogo, http.StatusBadRequest, claveErrorPeticion)
		return
	}
	entrada, err := leerEntradaReciboContactoPropio(r.Body)
	if err != nil {
		responderContactoPropio(w, h.catalogo, http.StatusBadRequest, claveErrorPeticion)
		return
	}
	resultado, err := h.ejecutor.ConsultarRecibo(r.Context(), entrada.Version)
	if err != nil {
		if errors.Is(err, ErrContactoPropioInvalido) {
			responderContactoPropio(w, h.catalogo, http.StatusBadRequest, claveErrorPeticion)
			return
		}
		responderContactoPropio(w, h.catalogo, http.StatusForbidden, claveErrorAccesoDenegado)
		return
	}
	if !resultado.Encontrado {
		responderContactoPropio(w, h.catalogo, http.StatusNotFound, claveErrorNoEncontrado)
		return
	}
	responderJSONContactoPropio(w, http.StatusOK, reciboContactoPropio{
		ReciboRef: resultado.ReciboOriginal.EvidenciaCentral.Referencia,
		Version:   resultado.Version,
	})
}

func rutaReciboContactoPropioExacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaReciboContactoPropio &&
		r.URL.RawQuery == "" && r.URL.EscapedPath() == r.URL.Path
}

func leerEntradaReciboContactoPropio(cuerpo io.Reader) (entradaReciboContactoPropio, error) {
	var entrada entradaReciboContactoPropio
	if cuerpo == nil {
		return entrada, ErrContactoPropioInvalido
	}
	contenido, err := io.ReadAll(io.LimitReader(cuerpo, maximoCuerpoContacto+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCuerpoContacto {
		return entrada, ErrContactoPropioInvalido
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &campos); err != nil || len(campos) != 1 {
		return entrada, ErrContactoPropioInvalido
	}
	version, ok := campos["version"]
	if !ok || bytes.Equal(bytes.TrimSpace(version), []byte("null")) {
		return entrada, ErrContactoPropioInvalido
	}
	if err := validarClavesJSONContactoPropio(contenido); err != nil {
		return entrada, err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil || decodificador.Decode(&struct{}{}) != io.EOF ||
		entrada.Version == 0 || entrada.Version > 1<<53-1 {
		return entradaReciboContactoPropio{}, ErrContactoPropioInvalido
	}
	return entrada, nil
}
