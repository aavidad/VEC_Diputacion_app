package server

import "net/http"

// registrarActivosCompartidos mantiene una única lista positiva para la
// identidad institucional y el tema común. No abre directorios completos:
// únicamente registra los recursos fijados; staticHandler también exige su
// inclusión en el manifiesto productivo de la superficie correspondiente.
// `/textos/` agrupa los catálogos de textos por idioma (datos i18n): cada
// fichero servido debe figurar también en ese manifiesto.
func registrarActivosCompartidos(mux *http.ServeMux, estaticos http.Handler) {
	mux.Handle("/styles.css", soloLecturaHTTP(estaticos))
	mux.Handle("/favicon.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/assets/logo-diputacion-granada.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/tema-vec.css", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/tema-vec.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/iconos-vec.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/idioma.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/textos.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/http.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/registro-errores.js", soloLecturaHTTP(estaticos))
	// «Mis correos» (5.08b), común a RRHH y al Área personal.
	mux.Handle("/comun/correos-propios.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/correos-propios.css", soloLecturaHTTP(estaticos))
	// «Mi imagen» (5.08c), común a RRHH y al Área personal.
	mux.Handle("/comun/imagen-propia.js", soloLecturaHTTP(estaticos))
	mux.Handle("/comun/imagen-propia.css", soloLecturaHTTP(estaticos))
	mux.Handle("/textos/", soloLecturaHTTP(estaticos))
	// Cada fichero PWA requiere además su entrada exacta en el manifiesto.
	mux.Handle("/pwa/", soloLecturaHTTP(estaticos))
}
