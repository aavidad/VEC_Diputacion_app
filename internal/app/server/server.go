package server

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"vec-diputacion-granada/config"
)

type healthResponse struct {
	Status string `json:"status"`
}

func NewHTTPServer(cfg config.Config, api http.Handler) (*http.Server, error) {
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
		Handler:           NewHandlerWithConfig(cfg, api),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}, nil
}

func NewHandler(api http.Handler) http.Handler {
	return NewHandlerWithConfig(config.Config{}, api)
}

func NewHandlerWithConfig(cfg config.Config, api http.Handler) http.Handler {
	cfg = cfg.Normalize()
	api = limitRequestBody(api, cfg.MaxRequestBodyBytes)
	mux := http.NewServeMux()
	mux.Handle("/locales/", localeHandler())
	mux.Handle("/", staticHandler(cfg.RRHHPresentationEnabled))
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle(cfg.APIBasePath, api)
	mux.Handle(cfg.APIBasePath+"/", api)
	mux.Handle("/candidates", api)
	mux.Handle("/candidates/", api)
	handler := restrictRemoteAddrs(mux, cfg.HTTPAllowedCIDRs)
	if cfg.AuthMode == config.AuthModeFake {
		handler = rechazarCabecerasProxyFake(handler)
	}
	return securityHeaders(handler)
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !presentacionRRHHHabilitada && strings.HasSuffix(
			strings.TrimSuffix(r.URL.Path, "/"), "/datos-presentacion.js",
		) {
			http.NotFound(w, r)
			return
		}
		setNoStoreForStatic(w, r)
		staticFileServer().ServeHTTP(w, r)
	})
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
