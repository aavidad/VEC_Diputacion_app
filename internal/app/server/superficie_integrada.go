package server

import (
	"net/http"

	"vec-diputacion-granada/config"
)

func NewHandler(api http.Handler) http.Handler {
	return NewHandlerWithConfig(config.Config{}, api)
}

// NewHandlerWithConfig conserva la composicion integrada de desarrollo. Aunque
// no constituye una frontera productiva, usa una lista positiva de superficies
// y no sirve la SPA historica situada en la raiz de web/static. Esta raiz se
// incluye en el binario normal y por eso debe permanecer cerrada incluso si un
// empaquetado defectuoso vuelve a copiar index.html o app.js.
func NewHandlerWithConfig(cfg config.Config, api http.Handler) http.Handler {
	cfg = cfg.Normalize()
	api = limitRequestBody(api, cfg.MaxRequestBodyBytes)
	estaticos := staticHandler(false)
	mux := http.NewServeMux()
	mux.Handle("/healthz", soloLecturaHTTP(http.HandlerFunc(handleHealthz)))
	registrarDirectorioAplicacion(mux, estaticos, "bolsa")
	registrarDirectorioAplicacion(mux, estaticos, "area-personal")
	registrarDirectorioAplicacion(mux, estaticos, "portal-empleado")
	registrarDirectorioAplicacion(mux, estaticos, "verificar")
	mux.Handle("/styles.css", soloLecturaHTTP(estaticos))
	mux.Handle("/favicon.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/locales/", soloLecturaHTTP(localeHandler()))
	mux.Handle(cfg.APIBasePath, api)
	mux.Handle(cfg.APIBasePath+"/", api)
	mux.Handle("/candidates", api)
	mux.Handle("/candidates/", api)
	handler := rechazarRutasNoCanonicas(mux)
	handler = rechazarSelectorPresentacionFueraDePresentacion(handler)
	handler = prohibirCookiesYAutorizacionProxyConLimite(handler, cfg.MaxRequestBodyBytes)
	handler = restrictRemoteAddrs(handler, cfg.HTTPAllowedCIDRs)
	if cfg.AuthMode == config.AuthModeFake {
		handler = rechazarCabecerasProxyFake(handler)
	}
	return securityHeaders(handler)
}

func registrarDirectorioAplicacion(mux *http.ServeMux, estaticos http.Handler, directorio string) {
	ruta := "/" + directorio
	mux.Handle(ruta, soloLecturaHTTP(redireccionDirectorio(directorio+"/")))
	mux.Handle(ruta+"/", soloLecturaHTTP(estaticos))
}
