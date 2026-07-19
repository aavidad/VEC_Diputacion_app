package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"

	"vec-diputacion-granada/config"
)

type healthResponse struct {
	Status string `json:"status"`
}

func NewHTTPServer(cfg config.Config, api http.Handler) (*http.Server, error) {
	return newHTTPServer(cfg, api, NewHandlerWithConfig)
}

// NewHTTPServerPublico construye el listener exclusivo para contenido anonimo
// y API publica. Su tabla de rutas no incluye ninguna superficie de empleado o
// administracion.
func NewHTTPServerPublico(cfg config.Config, api http.Handler) (*http.Server, error) {
	return newHTTPServer(cfg, api, NewHandlerPublicoWithConfig)
}

// NewHTTPServerInterno construye el listener exclusivo para el Portal del
// Empleado y la API VEC. Este listener no expone contenido publico ni la SPA
// historica que mezclaba ambas superficies.
func NewHTTPServerInterno(cfg config.Config, api http.Handler) (*http.Server, error) {
	return newHTTPServer(cfg, api, NewHandlerInternoWithConfig)
}

// NewHTTPServerPresentacion construye el unico listener que puede servir los
// adaptadores sinteticos. La raiz de composicion valida las guardas y este
// limite vuelve a exigirlas, una direccion IP local literal y redes locales.
func NewHTTPServerPresentacion(cfg config.Config, apiPublica http.Handler) (*http.Server, error) {
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
	return newHTTPServer(cfg, apiPublica, NewHandlerPresentacionWithConfig)
}

type constructorHandlerConConfig func(config.Config, http.Handler) http.Handler

func newHTTPServer(cfg config.Config, api http.Handler, constructor constructorHandlerConConfig) (*http.Server, error) {
	if api == nil {
		return nil, errors.New("server: api handler is required")
	}
	cfg = cfg.Normalize()
	redesPermitidas, err := prepararRedesPermitidas(cfg.HTTPAllowedCIDRs)
	if err != nil {
		return nil, err
	}
	if cfg.AuthMode == config.AuthModeFake && !redesExclusivamenteLocales(redesPermitidas) {
		return nil, errors.New("server: fake authentication requires loopback-only networks")
	}
	if cfg.AuthMode == config.AuthModeFake && !direccionEscuchaLoopback(cfg.Address) {
		return nil, errors.New("server: fake authentication requires a literal loopback listener")
	}
	return &http.Server{
		Addr:              cfg.Address,
		Handler:           constructor(cfg, api),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}, nil
}

// NewHandlerPublicoWithConfig expone unicamente la consulta anonima de Bolsa,
// sus recursos imprescindibles y la API publica. La lista positiva evita que
// una nueva carpeta estatica o ruta interna se publique por accidente. Al ser
// anonima tampoco acepta cookies ni permite que una API emita Set-Cookie.
func NewHandlerPublicoWithConfig(cfg config.Config, api http.Handler) http.Handler {
	cfg = cfg.Normalize()
	if api == nil {
		api = http.NotFoundHandler()
	}
	api = limitRequestBody(api, cfg.MaxRequestBodyBytes)
	estaticos := staticHandler(false)

	mux := http.NewServeMux()
	mux.Handle("/healthz", soloLecturaHTTP(http.HandlerFunc(handleHealthz)))
	mux.Handle("/bolsa", soloLecturaHTTP(redireccionDirectorio("bolsa/")))
	mux.Handle("/bolsa/", soloLecturaHTTP(estaticos))
	mux.Handle("/verificar", soloLecturaHTTP(redireccionDirectorio("verificar/")))
	mux.Handle("/verificar/", soloLecturaHTTP(estaticos))
	mux.Handle("/styles.css", soloLecturaHTTP(estaticos))
	mux.Handle("/portal-empleado/assets/logo-diputacion-granada.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/api/publico", api)
	mux.Handle("/api/publico/", api)

	handler := rechazarRutasNoCanonicas(mux)
	handler = rechazarSelectorPresentacionFueraDePresentacion(handler)
	handler = prohibirCookiesYAutorizacionProxyConLimite(handler, cfg.MaxRequestBodyBytes)
	handler = prohibirAutorizacionSuperficieAnonima(handler)
	return protegerSuperficie(cfg, handler)
}

// NewHandlerInternoWithConfig expone unicamente el Portal del Empleado y la
// API VEC. No acepta estado de sesion del navegador ni credenciales de proxy;
// la identidad interna debe llegar por el canal autenticado que componga el
// listener, nunca mediante cookies.
func NewHandlerInternoWithConfig(cfg config.Config, api http.Handler) http.Handler {
	cfg = cfg.Normalize()
	if api == nil {
		api = http.NotFoundHandler()
	}
	api = limitRequestBody(api, cfg.MaxRequestBodyBytes)
	estaticos := staticHandler(false)

	mux := http.NewServeMux()
	mux.Handle("/healthz", soloLecturaHTTP(http.HandlerFunc(handleHealthz)))
	mux.Handle("/portal-empleado", soloLecturaHTTP(redireccionDirectorio("portal-empleado/")))
	mux.Handle("/portal-empleado/", soloLecturaHTTP(estaticos))
	mux.Handle("/locales/", soloLecturaHTTP(localeHandler()))
	mux.Handle("/api/vec", api)
	mux.Handle("/api/vec/", api)

	handler := rechazarRutasNoCanonicas(mux)
	handler = rechazarSelectorPresentacionFueraDePresentacion(handler)
	handler = prohibirCookiesYAutorizacionProxyConLimite(handler, cfg.MaxRequestBodyBytes)
	return protegerSuperficie(cfg, handler)
}

// NewHandlerPresentacionWithConfig usa una lista positiva. No publica la SPA
// historica, ficheros de datos, documentacion ni una API interna. La consulta
// publica de Bolsa es la unica API admitida y permanece en solo lectura.
func NewHandlerPresentacionWithConfig(cfg config.Config, apiPublica http.Handler) http.Handler {
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
		http.Redirect(w, r, "presentacion/", http.StatusMovedPermanently)
	})))
	mux.Handle("/healthz", soloLecturaHTTP(http.HandlerFunc(handleHealthz)))
	registrarDirectorioPresentacion(mux, estaticos, "presentacion")
	registrarDirectorioPresentacion(mux, estaticos, "area-personal")
	registrarDirectorioPresentacion(mux, estaticos, "portal-empleado")
	registrarDirectorioPresentacion(mux, estaticos, "bolsa")
	registrarDirectorioPresentacion(mux, estaticos, "verificar")
	mux.Handle("/styles.css", soloLecturaHTTP(estaticos))
	mux.Handle("/favicon.svg", soloLecturaHTTP(estaticos))
	mux.Handle("/api/publico", soloLecturaHTTP(apiPublica))
	mux.Handle("/api/publico/", soloLecturaHTTP(apiPublica))

	handler := rechazarRutasNoCanonicas(mux)
	handler = prohibirCookiesYAutorizacionProxyConLimite(handler, cfg.MaxRequestBodyBytes)
	handler = prohibirAutorizacionSuperficieAnonima(handler)
	handler = marcarModoPresentacionAislada(handler)
	return protegerSuperficie(cfg, handler)
}

func registrarDirectorioPresentacion(mux *http.ServeMux, estaticos http.Handler, directorio string) {
	ruta := "/" + directorio
	mux.Handle(ruta, soloLecturaHTTP(redireccionDirectorio(directorio+"/")))
	mux.Handle(ruta+"/", soloLecturaHTTP(estaticos))
}

func protegerSuperficie(cfg config.Config, handler http.Handler) http.Handler {
	handler = restrictRemoteAddrs(handler, cfg.HTTPAllowedCIDRs)
	if cfg.AuthMode == config.AuthModeFake {
		handler = rechazarCabecerasProxyFake(handler)
	}
	return suprimirCuerpoHEAD(securityHeaders(handler))
}

// rechazarSelectorPresentacionFueraDePresentacion impide activar ramas de UI
// no autoritativas mediante una URL copiada del recorrido de demostracion. El
// valor es irrelevante: la mera presencia de la clave se rechaza.
func rechazarSelectorPresentacionFueraDePresentacion(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for clave := range r.URL.Query() {
			if strings.EqualFold(strings.TrimSpace(clave), "presentacion") {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
		}
		siguiente.ServeHTTP(w, r)
	})
}

func soloLecturaHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func redireccionDirectorio(destino string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destino, http.StatusMovedPermanently)
	})
}

// suprimirCuerpoHEAD mantiene el contrato de los handlers tambien cuando se
// prueban o componen sin pasar por la implementacion de transporte de net/http.
// Esta ultima ya omite el cuerpo de HEAD, pero no todos los ResponseWriter de
// adaptadores y pruebas ofrecen esa garantia.
func suprimirCuerpoHEAD(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w = &escritorRespuestaHEAD{ResponseWriter: w}
		}
		next.ServeHTTP(w, r)
	})
}

type escritorRespuestaHEAD struct {
	http.ResponseWriter
	escrito bool
}

func (w *escritorRespuestaHEAD) WriteHeader(estado int) {
	if w.escrito {
		return
	}
	if !esRespuestaInformativa(estado) {
		w.escrito = true
	}
	w.ResponseWriter.WriteHeader(estado)
}

func (w *escritorRespuestaHEAD) Write(contenido []byte) (int, error) {
	if !w.escrito {
		w.WriteHeader(http.StatusOK)
	}
	return len(contenido), nil
}

func (w *escritorRespuestaHEAD) Flush() {
	_ = w.FlushError()
}

func (w *escritorRespuestaHEAD) FlushError() error {
	if !w.escrito {
		w.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *escritorRespuestaHEAD) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// rechazarRutasNoCanonicas impide que ServeMux convierta una ruta con saltos
// de directorio o barras duplicadas en una redireccion hacia otra superficie.
func rechazarRutasNoCanonicas(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaHTTPCanonica(r) {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rutaHTTPCanonica(r *http.Request) bool {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" {
		return false
	}
	ruta := r.URL.Path
	if ruta == "" || ruta[0] != '/' || strings.ContainsRune(ruta, '\\') {
		return false
	}
	// Las superficies tienen rutas ASCII cerradas. Rechazar cualquier forma
	// escapada evita diferencias de normalizacion entre proxy, ServeMux y app.
	if r.URL.EscapedPath() != ruta {
		return false
	}
	limpia := path.Clean(ruta)
	return ruta == limpia || (limpia != "/" && ruta == limpia+"/")
}

func prohibirCookiesYAutorizacionProxy(next http.Handler) http.Handler {
	return prohibirCookiesYAutorizacionProxyConLimite(next, config.DefaultMaxRequestBodyBytes)
}

var errCuerpoHTTPDemasiadoGrande = errors.New("server: request body too large")

func prohibirCookiesYAutorizacionProxyConLimite(next http.Handler, limite int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if credencialAmbientalPresente(r.Header) ||
			contieneCabecera(r.Header, "Trailer") || len(r.Trailer) != 0 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if peticionPuedeTransportarTrailers(r) {
			if err := materializarCuerpoYTrailers(r, limite); err != nil {
				if errors.Is(err, errCuerpoHTTPDemasiadoGrande) {
					http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
					return
				}
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if len(r.Trailer) != 0 {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
		}
		// Defensa adicional para peticiones construidas fuera de net/http: el
		// handler recibe mapas propios y nunca puede observar un trailer que el
		// transporte materialice sobre la peticion original.
		peticionAislada := r.Clone(r.Context())
		peticionAislada.Header.Del("Trailer")
		peticionAislada.Trailer = make(http.Header)
		peticionAislada.GetBody = nil
		escritor := &escritorSinCookies{destino: w}
		next.ServeHTTP(escritor, peticionAislada)
		eliminarCookiesRespuesta(escritor.Header())
	})
}

func peticionPuedeTransportarTrailers(r *http.Request) bool {
	return len(r.TransferEncoding) != 0 ||
		(r.ProtoMajor >= 2 && r.Body != nil && r.Body != http.NoBody)
}

func materializarCuerpoYTrailers(r *http.Request, limite int64) error {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	if limite <= 0 {
		limite = config.DefaultMaxRequestBodyBytes
	}
	original := r.Body
	contenido, err := io.ReadAll(io.LimitReader(original, limite))
	if err != nil {
		_ = original.Close()
		return err
	}
	if int64(len(contenido)) == limite {
		extra, err := io.ReadAll(io.LimitReader(original, 1))
		if err != nil {
			_ = original.Close()
			return err
		}
		if len(extra) != 0 {
			_ = original.Close()
			return errCuerpoHTTPDemasiadoGrande
		}
	}
	if err := original.Close(); err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(contenido))
	r.ContentLength = int64(len(contenido))
	r.TransferEncoding = nil
	r.GetBody = nil
	return nil
}

func prohibirAutorizacionSuperficieAnonima(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contieneCabecera(r.Header, "Authorization") ||
			contieneCabecera(r.Trailer, "Authorization") {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func credencialAmbientalPresente(cabeceras http.Header) bool {
	return contieneCabecera(cabeceras, "Cookie") ||
		contieneCabecera(cabeceras, "Proxy-Authorization") ||
		contieneCabeceraIdentidadHeredada(cabeceras)
}

func contieneCabecera(cabeceras http.Header, buscada string) bool {
	for nombre := range cabeceras {
		if strings.EqualFold(strings.TrimSpace(nombre), buscada) {
			return true
		}
	}
	return false
}

func contieneCabeceraIdentidadHeredada(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		normalizado := strings.ToLower(strings.TrimSpace(nombre))
		if strings.HasPrefix(normalizado, "x-vec-") ||
			strings.HasPrefix(normalizado, "x-auth-") ||
			strings.HasPrefix(normalizado, "x-forwarded-") ||
			normalizado == "x-remote-user" || normalizado == "remote-user" ||
			normalizado == "forwarded" || normalizado == "via" {
			return true
		}
	}
	return false
}

func eliminarCookiesRespuesta(cabeceras http.Header) {
	declaraciones := make([]string, 0, len(cabeceras))
	for nombre, valores := range cabeceras {
		normalizado := strings.TrimSpace(nombre)
		esCookieTrailer := len(normalizado) >= len(http.TrailerPrefix) &&
			strings.EqualFold(normalizado[:len(http.TrailerPrefix)], http.TrailerPrefix) &&
			strings.EqualFold(strings.TrimSpace(normalizado[len(http.TrailerPrefix):]), "Set-Cookie")
		switch {
		case strings.EqualFold(normalizado, "Set-Cookie"), esCookieTrailer:
			delete(cabeceras, nombre)
		case strings.EqualFold(normalizado, "Trailer"):
			declaraciones = append(declaraciones, valores...)
			delete(cabeceras, nombre)
		}
	}
	if len(declaraciones) == 0 {
		return
	}
	permitidas := make([]string, 0, len(declaraciones))
	for _, declaracion := range declaraciones {
		for _, nombre := range strings.Split(declaracion, ",") {
			nombre = strings.TrimSpace(nombre)
			if nombre != "" && !strings.EqualFold(nombre, "Set-Cookie") {
				permitidas = append(permitidas, nombre)
			}
		}
	}
	for _, permitida := range permitidas {
		cabeceras.Add("Trailer", permitida)
	}
}

type escritorSinCookies struct {
	destino http.ResponseWriter
	escrito bool
}

func (w *escritorSinCookies) Header() http.Header { return w.destino.Header() }

func (w *escritorSinCookies) WriteHeader(estado int) {
	if w.escrito {
		return
	}
	eliminarCookiesRespuesta(w.Header())
	if !esRespuestaInformativa(estado) {
		w.escrito = true
	}
	w.destino.WriteHeader(estado)
}

func (w *escritorSinCookies) Write(contenido []byte) (int, error) {
	if !w.escrito {
		w.WriteHeader(http.StatusOK)
	}
	return w.destino.Write(contenido)
}

func (w *escritorSinCookies) Flush() {
	_ = w.FlushError()
}

func (w *escritorSinCookies) FlushError() error {
	eliminarCookiesRespuesta(w.Header())
	if !w.escrito {
		w.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(w.destino).Flush()
}

func esRespuestaInformativa(estado int) bool {
	return estado >= 100 && estado <= 199 && estado != http.StatusSwitchingProtocols
}

func rechazarCabecerasProxyFake(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for nombre := range r.Header {
			nombre = strings.TrimSpace(nombre)
			if strings.EqualFold(nombre, "Forwarded") || strings.EqualFold(nombre, "Via") ||
				strings.HasPrefix(strings.ToLower(nombre), "x-forwarded-") {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func limitRequestBody(next http.Handler, limit int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		// La respuesta parte como no almacenable. Los recursos estaticos
		// versionados pueden sustituir esta politica de forma deliberada dentro de
		// su manejador; las API y sesiones nunca heredan una cache permisiva.
		headers.Set("Cache-Control", "no-store")
		headers.Set("Pragma", "no-cache")
		headers.Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Cross-Origin-Resource-Policy", "same-origin")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func staticHandler(presentacionRRHHHabilitada bool) http.Handler {
	rutasProduccion := map[string]struct{}(nil)
	if !presentacionRRHHHabilitada {
		rutasProduccion = cargarRutasWebProduccion()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !presentacionRRHHHabilitada && rutaMaterialExclusivoPresentacion(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		if !presentacionRRHHHabilitada {
			if _, permitida := rutasProduccion[r.URL.Path]; !permitida {
				http.NotFound(w, r)
				return
			}
		}
		setNoStoreForStatic(w, r)
		staticFileServer().ServeHTTP(w, r)
	})
}

func rutaMaterialExclusivoPresentacion(ruta string) bool {
	for _, segmento := range strings.Split(strings.ToLower(ruta), "/") {
		if strings.Contains(segmento, "presentacion") || strings.Contains(segmento, "demo") {
			return true
		}
	}
	return false
}

func setNoStoreForStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" || strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".json") {
		w.Header().Set("Cache-Control", "no-store")
		return
	}
	if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
	}
}

func staticFileServer() http.Handler {
	for _, dir := range []string{"web/static", "../../../web/static"} {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			return http.FileServer(http.Dir(dir))
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "static files not found", http.StatusNotFound)
	})
}

func localeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.StripPrefix("/locales/", localeFileServer()).ServeHTTP(w, r)
	})
}

func localeFileServer() http.Handler {
	for _, dir := range []string{"locales", "../../../locales"} {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			return http.FileServer(http.Dir(dir))
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "locales not found", http.StatusNotFound)
	})
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func restrictRemoteAddrs(next http.Handler, allowedCIDRs []string) http.Handler {
	if len(allowedCIDRs) == 0 {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		})
	}
	allowed, err := prepararRedesPermitidas(allowedCIDRs)
	if err != nil {
		log.Printf("invalid VEC_HTTP_ALLOWED_CIDRS configuration; denying all requests: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := remoteIP(r.RemoteAddr)
		if ip == nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if ipAllowed(ip, allowed) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	})
}

func prepararRedesPermitidas(valores []string) ([]*net.IPNet, error) {
	redes := make([]*net.IPNet, 0, len(valores))
	for _, valor := range valores {
		red, err := parseAllowedNetwork(valor)
		if err != nil {
			return nil, errors.New("server: invalid allowed network configuration")
		}
		redes = append(redes, red)
	}
	return redes, nil
}

func parseAllowedNetwork(raw string) (*net.IPNet, error) {
	value := strings.TrimSpace(raw)
	if strings.Contains(value, "/") {
		_, network, err := net.ParseCIDR(value)
		return network, err
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil, errors.New("invalid IP or CIDR")
	}
	bits := 128
	if ip.To4() != nil {
		bits = 32
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
}

func remoteIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(strings.Trim(host, "[]"))
}

func ipAllowed(ip net.IP, allowed []*net.IPNet) bool {
	for _, network := range allowed {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func redesExclusivamenteLocales(redes []*net.IPNet) bool {
	if len(redes) == 0 {
		return false
	}
	for _, red := range redes {
		if red == nil {
			return false
		}
		unos, bits := red.Mask.Size()
		ip := red.IP
		if ipv4 := ip.To4(); ipv4 != nil {
			if bits != net.IPv4len*8 || unos < 8 || ipv4[0] != 127 {
				return false
			}
			continue
		}
		if bits != net.IPv6len*8 || unos != net.IPv6len*8 || !ip.Equal(net.IPv6loopback) {
			return false
		}
	}
	return true
}

func direccionEscuchaLoopback(direccion string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(direccion))
	if err != nil || strings.TrimSpace(host) == "" {
		return false
	}
	ip := net.ParseIP(strings.Trim(strings.TrimSpace(host), "[]"))
	return ip != nil && ip.IsLoopback()
}

func direccionEscuchaLocalPresentacion(direccion string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(direccion))
	if err != nil || strings.TrimSpace(host) == "" {
		return false
	}
	ip := net.ParseIP(strings.Trim(strings.TrimSpace(host), "[]"))
	return ip != nil && !ip.IsUnspecified() &&
		(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

func redesExclusivamenteLocalesPresentacion(redes []*net.IPNet) bool {
	if len(redes) == 0 {
		return false
	}
	permitidas := []*net.IPNet{
		debeParsearCIDR("127.0.0.0/8"),
		debeParsearCIDR("10.0.0.0/8"),
		debeParsearCIDR("172.16.0.0/12"),
		debeParsearCIDR("192.168.0.0/16"),
		debeParsearCIDR("169.254.0.0/16"),
		debeParsearCIDR("::1/128"),
		debeParsearCIDR("fc00::/7"),
		debeParsearCIDR("fe80::/10"),
	}
	for _, red := range redes {
		if red == nil || !redContenidaEnAlguna(red, permitidas) {
			return false
		}
	}
	return true
}

func redContenidaEnAlguna(red *net.IPNet, permitidas []*net.IPNet) bool {
	unos, bits := red.Mask.Size()
	if unos < 0 {
		return false
	}
	for _, permitida := range permitidas {
		unosPermitidos, bitsPermitidos := permitida.Mask.Size()
		if bits == bitsPermitidos && unos >= unosPermitidos && permitida.Contains(red.IP) {
			return true
		}
	}
	return false
}

func debeParsearCIDR(valor string) *net.IPNet {
	_, red, err := net.ParseCIDR(valor)
	if err != nil {
		panic(err)
	}
	return red
}
