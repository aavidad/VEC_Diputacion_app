package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"vec-diputacion-granada/config"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
)

const rutaRPTPublicaPresentacion = "/api/vec/personal/rpt-publica"

var ErrConcesionRPTPublicaPresentacionInvalida = errors.New("httpapi: concesion RPT publica de presentacion invalida")

type ConsultaRPTPublica interface {
	Listar(context.Context) (personaldomain.CatalogoRPTPublica, error)
}

// NewHandlerRPTPublicaPresentacion es una concesion de lectura exacta. La
// raiz aislada decide montarla; el handler no acepta identidad ni rutas hijas.
func NewHandlerRPTPublicaPresentacion(cfg config.Config, consulta ConsultaRPTPublica) (http.Handler, error) {
	if !cfg.Normalize().RRHHPresentationEnabledByDoubleGuard() || dependenciaHTTPNula(consulta) {
		return nil, ErrConcesionRPTPublicaPresentacionInvalida
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			escribirRPTPublicaError(w, http.StatusServiceUnavailable, "rpt_publica_no_disponible")
			return
		}
		if r.URL.Path != rutaRPTPublicaPresentacion || r.URL.RawPath != "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			escribirRPTPublicaError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		servirRPTPublica(w, r, consulta)
	}), nil
}

type filtroRPTPublica struct {
	q             string
	limit, offset int
}

func filtroRPTPublicaDesdePeticion(r *http.Request) (filtroRPTPublica, error) {
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return filtroRPTPublica{}, err
	}
	for clave, valoresClave := range valores {
		if (clave != "q" && clave != "limit" && clave != "offset") || len(valoresClave) != 1 {
			return filtroRPTPublica{}, errors.New("filtro")
		}
	}
	q := strings.TrimSpace(valores.Get("q"))
	if !utf8.ValidString(q) || utf8.RuneCountInString(q) > 100 {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	limit, err := enteroRPTPublica(valores.Get("limit"))
	if err != nil {
		return filtroRPTPublica{}, err
	}
	if limit < 1 || limit > 100 {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	offset, err := enteroRPTPublica(valores.Get("offset"))
	if err != nil || offset < 0 {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	return filtroRPTPublica{q: q, limit: limit, offset: offset}, nil
}
func enteroRPTPublica(valor string) (int, error) {
	if valor == "" {
		return 0, errors.New("filtro")
	}
	if strings.TrimSpace(valor) != valor || (len(valor) > 1 && valor[0] == '0') {
		return 0, errors.New("filtro")
	}
	return strconv.Atoi(valor)
}
func servirRPTPublica(w http.ResponseWriter, r *http.Request, consulta ConsultaRPTPublica) {
	w.Header().Set("Cache-Control", "no-store")
	filtro, err := filtroRPTPublicaDesdePeticion(r)
	if err != nil {
		escribirRPTPublicaError(w, http.StatusBadRequest, "filtro_rpt_publica_invalido")
		return
	}
	catalogo, err := consulta.Listar(r.Context())
	if err != nil || catalogo.Validar() != nil {
		escribirRPTPublicaError(w, http.StatusServiceUnavailable, "rpt_publica_no_disponible")
		return
	}
	coinciden := make([]personaldomain.CategoriaRPTPublica, 0, len(catalogo.Categorias))
	consultaNormalizada := textoRPTPublica(filtro.q)
	for _, categoria := range catalogo.Categorias {
		if consultaNormalizada == "" || strings.Contains(textoRPTPublica(categoria.Clave+" "+categoria.Denominacion+" "+strings.Join(categoria.Grupos, " ")+" "+strings.Join(categoria.Escalas, " ")), consultaNormalizada) {
			coinciden = append(coinciden, categoria.Clonar())
		}
	}
	inicio := filtro.offset
	if inicio > len(coinciden) {
		inicio = len(coinciden)
	}
	fin := inicio + filtro.limit
	if fin > len(coinciden) {
		fin = len(coinciden)
	}
	escribirRPTPublicaJSON(w, http.StatusOK, map[string]any{"rpt": map[string]any{"items": coinciden[inicio:fin], "total": len(coinciden), "limit": filtro.limit, "offset": filtro.offset, "esquema": catalogo.Esquema, "fuente": catalogo.Fuente}})
}
func textoRPTPublica(valor string) string {
	valor = norm.NFD.String(strings.ToLower(strings.TrimSpace(valor)))
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, valor)
}
func escribirRPTPublicaJSON(w http.ResponseWriter, estado int, datos any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": datos})
}
func escribirRPTPublicaError(w http.ResponseWriter, estado int, mensaje string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": mensaje})
}
