package server

import (
	"os"
	"path/filepath"
	"strings"
)

// cargarRutasWebProduccion convierte el mismo manifiesto que usa Docker en la
// lista positiva HTTP. Si el manifiesto falta o contiene una ruta no canonica,
// devuelve una lista vacia y la superficie estatica normal falla cerrada. El
// handler de presentacion no usa esta lista porque su artefacto es deliberada y
// fisicamente distinto.
func cargarRutasWebProduccion() map[string]struct{} {
	contenido, err := leerManifiestoWebProduccion()
	if err != nil {
		return map[string]struct{}{}
	}
	rutas := make(map[string]struct{})
	for _, linea := range strings.Split(string(contenido), "\n") {
		rutaFuente := strings.TrimSpace(linea)
		if rutaFuente == "" || rutaFuente == "produccion.manifest" {
			continue
		}
		if filepath.IsAbs(rutaFuente) || strings.Contains(rutaFuente, "..") ||
			!strings.HasPrefix(rutaFuente, "static/") {
			return map[string]struct{}{}
		}
		rutaHTTP := "/" + strings.TrimPrefix(filepath.ToSlash(filepath.Clean(rutaFuente)), "static/")
		if _, duplicada := rutas[rutaHTTP]; duplicada {
			return map[string]struct{}{}
		}
		rutas[rutaHTTP] = struct{}{}
		if strings.HasSuffix(rutaHTTP, "/index.html") {
			rutas[strings.TrimSuffix(rutaHTTP, "index.html")] = struct{}{}
		}
	}
	return rutas
}

func leerManifiestoWebProduccion() ([]byte, error) {
	for _, candidata := range []string{
		"web/produccion.manifest",
		"../../../web/produccion.manifest",
	} {
		contenido, err := os.ReadFile(candidata)
		if err == nil {
			return contenido, nil
		}
	}
	return nil, os.ErrNotExist
}
