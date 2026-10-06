package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

type ConsultaEstructuraOrganizativaPublica interface {
	Obtener(context.Context) (domain.EstructuraOrganizativaPublica, error)
}

func escribirEstructuraOrganizativaPublicaJSON(w http.ResponseWriter, estado int, datos any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": datos})
}
func escribirEstructuraOrganizativaPublicaError(w http.ResponseWriter, estado int, mensaje string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": mensaje})
}
