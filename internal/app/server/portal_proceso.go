package server

import (
	"net/http"
	"strings"

	"vec-diputacion-granada/config"
)

// rutasPortal es una lista de rutas exactas y de prefijos. Las rutas ya
// llegan canónicas (rechazarRutasNoCanonicas se aplica antes).
type rutasPortal struct {
	exactas  []string
	prefijos []string
}

func (r rutasPortal) contiene(ruta string) bool {
	for _, exacta := range r.exactas {
		if ruta == exacta {
			return true
		}
	}
	for _, prefijo := range r.prefijos {
		if strings.HasPrefix(ruta, prefijo) {
			return true
		}
	}
	return false
}

// rutasSoloExterno son las del Área personal: un proceso interno nunca las
// atiende, así que una sesión de aspirante no llega a su composición.
var rutasSoloExterno = rutasPortal{
	exactas: []string{
		"/area-personal",
		"/api/vec/bolsa/area-personal",
		"/api/vec/usuarios/contacto-propio",
	},
	prefijos: []string{
		"/area-personal/",
		"/api/vec/bolsa/area-personal/",
		"/api/vec/bolsa/mi-",
		"/api/vec/bolsa/mis-",
		"/api/vec/usuarios/area-personal/",
		"/api/vec/usuarios/contacto-propio/",
		"/api/vec/personas/mi-perfil/",
		"/api/vec/personas/mis-preferencias/",
	},
}

// rutasComunesExterno son las que el proceso externo sirve además del Área
// personal: consulta pública, verificación, recursos compartidos y estado.
// Es una lista positiva: una ruta interna nueva queda fuera sin tocar nada.
var rutasComunesExterno = rutasPortal{
	exactas: []string{
		"/bolsa", "/verificar", "/acceso", "/api/publico",
		"/styles.css", "/favicon.svg",
		"/livez", "/readyz", "/healthz",
	},
	prefijos: []string{
		"/bolsa/", "/verificar/", "/acceso/", "/api/publico/",
		"/assets/", "/comun/", "/textos/", "/locales/", "/pwa/",
	},
}

// restringirRutasPortalProceso limita las rutas según el portal del proceso.
// Sin separación deja pasar todo, como siempre. Un valor no válido cierra
// todas las rutas: nunca se interpreta como portal combinado.
func restringirRutasPortalProceso(valor string, siguiente http.Handler) http.Handler {
	switch valor {
	case config.ValorPortalProcesoExterno:
		return filtrarRutas(siguiente, func(ruta string) bool {
			return rutasSoloExterno.contiene(ruta) || rutasComunesExterno.contiene(ruta)
		})
	case config.ValorPortalProcesoInterno:
		return filtrarRutas(siguiente, func(ruta string) bool {
			return !rutasSoloExterno.contiene(ruta)
		})
	case "":
		return siguiente
	default:
		return http.HandlerFunc(http.NotFound)
	}
}

func filtrarRutas(siguiente http.Handler, admitida func(string) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || !admitida(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		siguiente.ServeHTTP(w, r)
	})
}
