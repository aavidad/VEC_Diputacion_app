package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/shared/i18n"
)

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, key string, err error) {
	envelope := responseEnvelope{Message: h.t(key)}
	if err != nil {
		envelope.Error = publicError(status, err)
	}
	h.writeJSON(w, status, envelope)
}

func publicError(status int, err error) string {
	if err == nil || status >= http.StatusInternalServerError {
		return ""
	}
	message := err.Error()
	internalFragments := []string{
		"usecase:",
		"repository:",
		"durable file",
		"save ",
		"load ",
		"read ",
		"write ",
	}
	for _, fragment := range internalFragments {
		if strings.Contains(message, fragment) {
			return ""
		}
	}
	return message
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, response responseEnvelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) t(key string) string {
	message := h.messages.T(i18n.DefaultLocale, key)
	if message != key {
		return message
	}
	return fallbackMessage(key)
}

func fallbackMessage(key string) string {
	switch key {
	case "api.candidate.expediente_exported":
		return "Expediente exportado correctamente"
	case "api.error.method_not_allowed":
		return "Metodo no permitido"
	case "api.procedure.demo_completed":
		return "Flujo administrativo completado"
	case "api.procedure.convocatoria_created":
		return "Convocatoria creada correctamente"
	case "api.procedure.solicitud_registered":
		return "Solicitud registrada correctamente"
	case "api.procedure.listado_published":
		return "Listado publicado correctamente"
	case "api.admin.status_loaded":
		return "Estado operativo cargado"
	case "api.admin.capabilities_loaded":
		return "Capacidades administrativas cargadas"
	default:
		return key
	}
}

func fallbackCatalog() *i18n.Catalog {
	catalog, _ := i18n.New(i18n.DefaultLocale, map[string]map[string]string{
		i18n.DefaultLocale: {
			"api.candidate.created":              "Candidato creado correctamente",
			"api.candidate.merit_added":          "Merito agregado correctamente",
			"api.candidate.baremo_calculated":    "Baremo calculado correctamente",
			"api.candidate.expediente_exported":  "Expediente exportado correctamente",
			"api.error.bad_request":              "Solicitud no valida",
			"api.error.unauthorized":             "Autenticacion requerida",
			"api.error.forbidden":                "Permisos insuficientes",
			"api.error.not_found":                "Recurso no encontrado",
			"api.error.method_not_allowed":       "Metodo no permitido",
			"api.error.internal":                 "Error interno del servidor",
			"api.procedure.demo_completed":       "Flujo administrativo completado",
			"api.procedure.convocatoria_created": "Convocatoria creada correctamente",
			"api.procedure.solicitud_registered": "Solicitud registrada correctamente",
			"api.procedure.listado_published":    "Listado publicado correctamente",
			"api.admin.status_loaded":            "Estado operativo cargado",
			"api.admin.capabilities_loaded":      "Capacidades administrativas cargadas",
		},
	})
	return catalog
}
