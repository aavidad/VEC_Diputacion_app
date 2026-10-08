package main

import (
	"encoding/json"
	"errors"
	"io"
)

// Sin catálogo sólo se emite un código de protocolo. Las causas se conservan
// para clasificar el fallo, sin serializar rutas, contenido o mensajes raw.
func informarFalloCatalogo(w io.Writer, err error) int {
	if err == nil {
		return 0
	}
	escrituraErr := json.NewEncoder(w).Encode(struct {
		Codigo string `json:"codigo"`
	}{Codigo: "catalogo_no_disponible"})
	return codigoSalidaFalloCatalogo(errors.Join(err, escrituraErr))
}

func codigoSalidaFalloCatalogo(err error) int {
	if err == nil {
		return 0
	}
	return 2
}
