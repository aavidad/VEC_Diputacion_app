package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"vec-diputacion-granada/config"
)

const rutaConfiguracionCorreoAdministracion = "/api/vec/administracion/configuracion-correo"

var errTransporteAdministracionNoDisponible = errors.New("server: transporte de administracion no disponible")

// NewHTTPServerAdministracion crea el listener ADMIN segregado. El material
// TLS pertenece exclusivamente a su bloque de transporte y requiere mTLS TLS
// 1.3 antes de que la autoridad administrativa pueda ligarlo al servidor.
func NewHTTPServerAdministracion(cfg config.Config, api http.Handler) (*http.Server, error) {
	proyeccion, err := cfg.TransporteAdministracion.ConfiguracionServidor(cfg)
	if err != nil || api == nil {
		return nil, errTransporteAdministracionNoDisponible
	}
	tlsConfig, err := configuracionTLSAdministracion(cfg.TransporteAdministracion)
	if err != nil {
		return nil, errTransporteAdministracionNoDisponible
	}
	redes, err := prepararRedesPermitidas(proyeccion.HTTPAllowedCIDRs)
	if err != nil || len(redes) == 0 {
		return nil, errTransporteAdministracionNoDisponible
	}
	return &http.Server{
		Addr:              proyeccion.Address,
		Handler:           newHandlerAdministracion(proyeccion, api),
		ReadHeaderTimeout: proyeccion.ReadHeaderTimeout,
		ReadTimeout:       proyeccion.ReadTimeout,
		WriteTimeout:      proyeccion.WriteTimeout,
		IdleTimeout:       proyeccion.IdleTimeout,
		MaxHeaderBytes:    proyeccion.MaxHeaderBytes,
		TLSConfig:         tlsConfig,
	}, nil
}

// NewHandlerAdministracionWithConfig permite pruebas efímeras de la tabla de
// rutas. Una configuración ADMIN ausente, incompleta o mezclada con RRHH no
// expone una ruta alternativa: responde servicio no disponible.
func NewHandlerAdministracionWithConfig(cfg config.Config, api http.Handler) http.Handler {
	proyeccion, err := cfg.TransporteAdministracion.ConfiguracionServidor(cfg)
	if err != nil || api == nil {
		return handlerAdministracionNoDisponible()
	}
	return newHandlerAdministracion(proyeccion, api)
}

func newHandlerAdministracion(cfg config.Config, api http.Handler) http.Handler {
	api = limitRequestBody(api, cfg.MaxRequestBodyBytes)
	activos := activosAdministracion()
	mux := http.NewServeMux()
	mux.Handle("/administracion", soloLecturaHTTP(redireccionDirectorio("administracion/")))
	mux.Handle("/administracion/", soloLecturaHTTP(activos))
	mux.Handle("/portal-empleado/portal-i18n.js", soloLecturaHTTP(activos))
	mux.Handle(rutaConfiguracionCorreoAdministracion, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !peticionAdministracionExacta(r, rutaConfiguracionCorreoAdministracion) {
			http.NotFound(w, r)
			return
		}
		api.ServeHTTP(w, r)
	}))
	handler := rechazarRutasNoCanonicas(mux)
	handler = prohibirCookiesYAutorizacionProxyConLimite(handler, cfg.MaxRequestBodyBytes)
	handler = prohibirAutorizacion(handler)
	handler = exigirOrigenAdministracion(cfg.HTTPAllowedOrigins, handler)
	return protegerSuperficie(cfg, handler)
}

func handlerAdministracionNoDisponible() http.Handler {
	return suprimirCuerpoHEAD(securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	})))
}

func configuracionTLSAdministracion(c config.ConfiguracionTransporteAdministracion) (*tls.Config, error) {
	certificado, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile)
	if err != nil {
		return nil, errTransporteAdministracionNoDisponible
	}
	pemCA, err := os.ReadFile(c.TLSClientCAFile)
	if err != nil {
		return nil, errTransporteAdministracionNoDisponible
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(pemCA) {
		return nil, errTransporteAdministracionNoDisponible
	}
	return &tls.Config{
		Certificates: []tls.Certificate{certificado},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    raices,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	}, nil
}

func activosAdministracion() http.Handler {
	permitidos := map[string]struct{}{
		"/administracion/":                         {},
		"/administracion/administracion.js":        {},
		"/administracion/configuracion-correo.js":  {},
		"/administracion/configuracion-correo.css": {},
		"/administracion/tema.css":                 {},
		"/portal-empleado/portal-i18n.js":          {},
	}
	estaticos := staticFileServer()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !peticionAdministracionExacta(r, r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		if _, ok := permitidos[r.URL.Path]; !ok {
			http.NotFound(w, r)
			return
		}
		setNoStoreForStatic(w, r)
		switch r.URL.Path {
		case "/administracion/tema.css":
			r = r.Clone(r.Context())
			r.URL.Path = "/portal-empleado/portal.css"
		}
		estaticos.ServeHTTP(w, r)
	})
}

func exigirOrigenAdministracion(origenes []string, siguiente http.Handler) http.Handler {
	permitidos := make(map[string]struct{}, len(origenes))
	for _, origen := range origenes {
		permitidos[strings.ToLower(origen)] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origen := strings.TrimSpace(r.Header.Get("Origin"))
		if origen == "" {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				siguiente.ServeHTTP(w, r)
				return
			}
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		u, err := url.Parse(origen)
		if err != nil || strings.ToLower(u.String()) != strings.ToLower(origen) || u.Scheme != "https" || u.Host == "" || !hostAdministracionCoincide(r.Host, u.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := permitidos[strings.ToLower(origen)]; !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		siguiente.ServeHTTP(w, r)
	})
}

func hostAdministracionCoincide(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func peticionAdministracionExacta(r *http.Request, ruta string) bool {
	return r != nil && r.URL != nil && r.URL.Path == ruta && r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == "" && r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && r.URL.EscapedPath() == r.URL.Path && !strings.Contains(r.URL.Path, "%")
}
