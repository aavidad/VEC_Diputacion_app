package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/seleccion/application"
)

func configurarSeleccionLocal(w http.ResponseWriter) {
	e, err := simulacion.Ejemplos()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Ejemplos []simulacion.Ejemplo `json:"ejemplos"`
	}{e}); err != nil {
		slog.Warn("seleccion_respuesta_no_entregada", "operacion", "configuracion_local")
	}
}

func simularSeleccionLocal(w http.ResponseWriter, datos []byte) {
	s, err := simulacion.Decodificar(datos)
	if err != nil {
		responderError(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	e, b, err := simulacion.Preparar(s.EjemploRef)
	if err != nil {
		responderError(w, http.StatusBadRequest, "ejemplo_no_admitido")
		return
	}
	r, err := application.Simular(s.Configuracion, e, b)
	if err != nil {
		responderError(w, http.StatusUnprocessableEntity, "reglas_invalidas")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(r); err != nil {
		slog.Warn("seleccion_respuesta_no_entregada", "operacion", "simulacion_local")
	}
}
