package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

const rutaEstructuraOrganizativaPublicaPresentacion = "/api/vec/personal/estructura-organizativa-publica"

var ErrConcesionEstructuraOrganizativaPublicaPresentacionInvalida = errors.New("httpapi: concesion de estructura organizativa publica de presentacion invalida")

type ConsultaEstructuraOrganizativaPublica interface {
	Obtener(context.Context) (domain.EstructuraOrganizativaPublica, error)
}

func NewHandlerEstructuraOrganizativaPublicaPresentacion(cfg config.Config, consulta ConsultaEstructuraOrganizativaPublica) (http.Handler, error) {
	if !cfg.Normalize().RRHHPresentationEnabledByDoubleGuard() || dependenciaHTTPNula(consulta) {
		return nil, ErrConcesionEstructuraOrganizativaPublicaPresentacionInvalida
	}
	return handlerEstructuraOrganizativaPublica(consulta), nil
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
