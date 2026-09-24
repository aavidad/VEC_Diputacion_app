package server

import "net/http"

// registrarActivosCompartidos mantiene una única lista positiva para la
// identidad institucional y el tema común. No abre directorios completos:
// únicamente registra los recursos fijados; staticHandler también exige su
// inclusión en el manifiesto productivo de la superficie correspondiente.
func registrarActivosCompartidos(mux *http.ServeMux, estaticos http.Handler) {
	mux.Handle("/styles.css", soloLecturaHTTP(estaticos))
	mux.Handle("/favicon.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/assets/logo-diputacion-granada.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/tema-vec.css", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/tema-vec.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/iconos-vec.js", soloLecturaHTTP(estaticos))
}
