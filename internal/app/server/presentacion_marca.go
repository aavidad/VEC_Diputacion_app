package server

import "net/http"

const (
	cabeceraModoPresentacion = "X-VEC-Modo-Presentacion"
	valorModoPresentacion    = "aislada-sintetica-v1"
)

// marcarModoPresentacionAislada permite que las herramientas automáticas
// distingan este listener sintético de cualquier superficie real. Solo se
// compone después de validar las dos guardas, el listener y las redes locales.
func marcarModoPresentacionAislada(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(cabeceraModoPresentacion, valorModoPresentacion)
		next.ServeHTTP(w, r)
	})
}
