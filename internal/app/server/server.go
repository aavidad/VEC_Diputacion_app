package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

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
	return &http.Server{
		Addr:              cfg.Address,
		Handler:           NewHandlerWithConfig(cfg, api),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}, nil
}

func NewHandler(api http.Handler) http.Handler {
	return NewHandlerWithConfig(config.Config{}, api)
}

func NewHandlerWithConfig(cfg config.Config, api http.Handler) http.Handler {
	cfg = cfg.Normalize()
	mux := http.NewServeMux()
	mux.Handle("/locales/", localeHandler())
	mux.Handle("/", staticHandler())
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle(cfg.APIBasePath, api)
	mux.Handle(cfg.APIBasePath+"/", api)
	mux.Handle("/candidates", api)
	mux.Handle("/candidates/", api)
	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func staticHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		staticFileServer().ServeHTTP(w, r)
	})
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
