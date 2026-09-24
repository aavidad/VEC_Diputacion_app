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
	return rutasHTTPDesdeManifiestoWeb(contenido)
}

func rutasHTTPDesdeManifiestoWeb(contenido []byte) map[string]struct{} {
	rutas := make(map[string]struct{})
	empaquetados := make(map[string]struct{}, 2)
	for _, linea := range strings.Split(string(contenido), "\n") {
		rutaFuente := strings.TrimSpace(linea)
		if rutaFuente == "" || rutaFuente == "produccion.manifest" {
			continue
		}
		// El ZIP y su indice son recursos del artefacto para la ruta OSM
		// autorizada. No se convierten en rutas estaticas del navegador.
		if rutaFuente == "cartografia/granada-base-20260719-z8-z12.zip" ||
			rutaFuente == "cartografia/granada-base-20260719-z8-z12.json" {
			if _, duplicada := empaquetados[rutaFuente]; duplicada {
				return map[string]struct{}{}
			}
			empaquetados[rutaFuente] = struct{}{}
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
