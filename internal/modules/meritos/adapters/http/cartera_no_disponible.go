package meritoshttp

import "net/http"

// NuevoHandlerCarteraNoDisponible conserva las dos rutas visibles aunque
// falten LOGIN, catálogo o material de autorización. No resuelve identidad.
func NuevoHandlerCarteraNoDisponible() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r == nil || r.URL == nil || r.URL.RawPath != "" ||
			r.URL.Path != RutaCartera && r.URL.Path != RutaRetiradasCartera ||
			r.URL.EscapedPath() != r.URL.Path {
			responderErrorCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
			return
		}
		metodo := http.MethodGet
		if r.URL.Path == RutaRetiradasCartera {
			metodo = http.MethodPost
		}
		if r.Method != metodo {
			w.Header().Set("Allow", metodo)
			responderErrorCartera(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
			return
		}
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
	})
}
