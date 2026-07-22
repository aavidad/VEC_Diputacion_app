package server

import "net/http"

// registrarActivosCompartidos mantiene una única lista positiva para la
// identidad institucional y el tema heredado por todas las superficies. No
// abre el directorio /assets completo: únicamente publica el logotipo fijado.
func registrarActivosCompartidos(mux *http.ServeMux, estaticos http.Handler) {
	mux.Handle("/styles.css", soloLecturaHTTP(estaticos))
	mux.Handle("/favicon.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/assets/logo-diputacion-granada.svg", soloLecturaHTTP(estaticos))
}
