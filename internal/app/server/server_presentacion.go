package server

import (
	"errors"
	"net/http"

	"vec-diputacion-granada/config"
)

// NewHTTPServerPresentacion construye el unico listener que puede servir los
// adaptadores sinteticos. La raiz de composicion valida las guardas y este
// limite vuelve a exigirlas, una direccion IP local literal y redes locales.
func NewHTTPServerPresentacion(cfg config.Config, apiPublica http.Handler) (*http.Server, error) {
	return NewHTTPServerPresentacionConComprobadorDisponibilidad(cfg, apiPublica, nil)
}

func NewHTTPServerPresentacionConComprobadorDisponibilidad(cfg config.Config, apiPublica http.Handler, comprobador ComprobadorDisponibilidad) (*http.Server, error) {
	return newHTTPServerPresentacion(cfg, apiPublica, nil, comprobador)
}

// NewHTTPServerPresentacionCategorias compone la unica concesion sintetica
// adicional de presentacion: la coleccion exacta de categorias profesionales.
// No acepta un API VEC general y conserva cerradas las demas rutas privadas.
func NewHTTPServerPresentacionCategorias(cfg config.Config, apiPublica, categorias http.Handler) (*http.Server, error) {
	if categorias == nil {
		return nil, errors.New("server: concesion sintetica de categorias ausente")
	}
	return newHTTPServerPresentacion(cfg, apiPublica, categorias, nil)
}

// NewHTTPServerPresentacionPersonalRPT suma a la concesion de categorias la
// unica lectura RPT publica aprobada. Ninguna otra ruta /api/vec se monta.
func NewHTTPServerPresentacionPersonalRPT(cfg config.Config, apiPublica, categorias, rpt, estructura http.Handler) (*http.Server, error) {
	if categorias == nil || rpt == nil || estructura == nil {
		return nil, errors.New("server: concesion de Personal ausente")
	}
	return newHTTPServerPresentacionConPersonal(cfg, apiPublica, categorias, rpt, estructura, nil)
}

func newHTTPServerPresentacion(cfg config.Config, apiPublica, categorias http.Handler, comprobador ComprobadorDisponibilidad) (*http.Server, error) {
	return newHTTPServerPresentacionConPersonal(cfg, apiPublica, categorias, nil, nil, comprobador)
}

func newHTTPServerPresentacionConPersonal(cfg config.Config, apiPublica, categorias, rpt, estructura http.Handler, comprobador ComprobadorDisponibilidad) (*http.Server, error) {
	cfg = cfg.Normalize()
	if !cfg.RRHHPresentationEnabledByDoubleGuard() {
		return nil, errors.New("server: activacion de presentacion RRHH incompleta")
	}
	if !direccionEscuchaLocalPresentacion(cfg.Address) {
		return nil, errors.New("server: la presentacion RRHH exige una direccion IP local literal")
	}
	redes, err := prepararRedesPermitidas(cfg.HTTPAllowedCIDRs)
	if err != nil || !redesExclusivamenteLocalesPresentacion(redes) {
		return nil, errors.New("server: la presentacion RRHH exige redes locales enumeradas")
	}
	return newHTTPServer(cfg, apiPublica, func(cfg config.Config, api http.Handler) http.Handler {
		return newHandlerPresentacionConPersonal(cfg, api, categorias, rpt, estructura, comprobador)
	})
}

// NewHandlerPresentacionWithConfig usa una lista positiva. No publica la SPA
// historica, ficheros de datos, documentacion ni una API interna. La consulta
// publica de Bolsa es la unica API admitida y permanece en solo lectura.
func NewHandlerPresentacionWithConfig(cfg config.Config, apiPublica http.Handler) http.Handler {
	return NewHandlerPresentacionWithConfigConComprobadorDisponibilidad(cfg, apiPublica, nil)
}

func NewHandlerPresentacionWithConfigConComprobadorDisponibilidad(cfg config.Config, apiPublica http.Handler, comprobador ComprobadorDisponibilidad) http.Handler {
	return newHandlerPresentacionWithConfig(cfg, apiPublica, nil, comprobador)
}

// NewHandlerPresentacionCategoriasWithConfig conserva la lista positiva de la
// presentacion y anade solo /api/vec/personal/categories. Es un constructor
// separado para que los constructores ordinarios nunca puedan publicar la
// ruta privada por accidente.
func NewHandlerPresentacionCategoriasWithConfig(cfg config.Config, apiPublica, categorias http.Handler) http.Handler {
	if categorias == nil {
		return suprimirCuerpoHEAD(securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		})))
	}
	return newHandlerPresentacionWithConfig(cfg, apiPublica, categorias, nil)
}

func newHandlerPresentacionWithConfig(cfg config.Config, apiPublica, categorias http.Handler, comprobador ComprobadorDisponibilidad) http.Handler {
	return newHandlerPresentacionConPersonal(cfg, apiPublica, categorias, nil, nil, comprobador)
}

func newHandlerPresentacionConPersonal(cfg config.Config, apiPublica, categorias, rpt, estructura http.Handler, comprobador ComprobadorDisponibilidad) http.Handler {
	cfg = cfg.Normalize()
	redes, err := prepararRedesPermitidas(cfg.HTTPAllowedCIDRs)
	if !cfg.RRHHPresentationEnabledByDoubleGuard() ||
		!direccionEscuchaLocalPresentacion(cfg.Address) || err != nil ||
		!redesExclusivamenteLocalesPresentacion(redes) {
		return suprimirCuerpoHEAD(securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		})))
	}
	if apiPublica == nil {
		apiPublica = http.NotFoundHandler()
	}
	estaticos := staticHandler(true)
	mux := http.NewServeMux()
	mux.Handle("/", soloLecturaHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "bolsa/", http.StatusMovedPermanently)
	})))
	registrarRutasDisponibilidad(mux, comprobador)
	for _, directorio := range []string{"bolsa", "verificar"} {
		registrarDirectorioPresentacion(mux, estaticos, directorio)
	}
	registrarActivosCompartidos(mux, estaticos)
	mux.Handle("/api/publico", soloLecturaHTTP(apiPublica))
	mux.Handle("/api/publico/", soloLecturaHTTP(apiPublica))
	if categorias != nil {
		mux.Handle("/api/vec/personal/categories", soloLecturaHTTP(categorias))
	}
	if rpt != nil {
		mux.Handle("/api/vec/personal/rpt-publica", soloLecturaHTTP(rpt))
	}
	if estructura != nil {
		mux.Handle("/api/vec/personal/estructura-organizativa-publica", soloLecturaHTTP(estructura))
	}
	handler := rechazarRutasNoCanonicas(mux)
	handler = prohibirCookiesYAutorizacionProxyConLimite(handler, cfg.MaxRequestBodyBytes)
	handler = prohibirAutorizacion(handler)
	handler = marcarModoPresentacionAislada(handler)
	return protegerSuperficie(cfg, handler)
}
