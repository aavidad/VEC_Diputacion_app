package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

// El transporte web admite reglas; la entrada permanece ligada a un ejemplo
// público embebido. No admite ficheros de personas ni modifica Personal/RUM.
func simularConcursos(w http.ResponseWriter, datos []byte) {
	var solicitud struct {
		EjemploRef    string               `json:"ejemplo_ref"`
		Configuracion domain.Configuracion `json:"configuracion"`
	}
	if err := simulacion.Decodificar(bytes.NewReader(datos), &solicitud); err != nil {
		responderError(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	ejemplos, err := simulacion.Ejemplos()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	for _, ejemplo := range ejemplos {
		if ejemplo.Referencia != solicitud.EjemploRef {
			continue
		}
		resultado, err := application.Simular(solicitud.Configuracion, ejemplo.Entrada)
		if err != nil {
			responderError(w, http.StatusUnprocessableEntity, "reglas_invalidas")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(simulacion.Sobre(resultado))
		return
	}
	responderError(w, http.StatusBadRequest, "ejemplo_no_admitido")
}
