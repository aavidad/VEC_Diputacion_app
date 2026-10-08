package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
)

type ConsultaRPTPublica interface {
	Listar(context.Context) (personaldomain.CatalogoRPTPublica, error)
}

type vistaRPTPublica string

var patronCodigoPuestoRPTFiltro = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{0,63}$`)

const (
	vistaRPTCategorias vistaRPTPublica = "categorias"
	vistaRPTPuestos    vistaRPTPublica = "puestos"
)

type filtroRPTPublica struct {
	vista         vistaRPTPublica
	q             string
	codigoPuesto  string
	limit, offset int
}

func filtroRPTPublicaDesdePeticion(r *http.Request) (filtroRPTPublica, error) {
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return filtroRPTPublica{}, err
	}
	for clave, valoresClave := range valores {
		if (clave != "vista" && clave != "q" && clave != "limit" && clave != "offset" && clave != "codigo_puesto") || len(valoresClave) != 1 {
			return filtroRPTPublica{}, errors.New("filtro")
		}
	}
	vista := vistaRPTCategorias
	if recibida := valores.Get("vista"); recibida != "" {
		vista = vistaRPTPublica(recibida)
	}
	if vista != vistaRPTCategorias && vista != vistaRPTPuestos {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	codigoPuesto := valores.Get("codigo_puesto")
	if valores.Has("codigo_puesto") && (vista != vistaRPTPuestos || !patronCodigoPuestoRPTFiltro.MatchString(codigoPuesto)) {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	q := strings.TrimSpace(valores.Get("q"))
	if !utf8.ValidString(q) || utf8.RuneCountInString(q) > 100 {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	limit, err := enteroRPTPublica(valores.Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	offset, err := enteroRPTPublica(valores.Get("offset"))
	if err != nil || offset < 0 {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	return filtroRPTPublica{vista: vista, q: q, codigoPuesto: codigoPuesto, limit: limit, offset: offset}, nil
}
func enteroRPTPublica(valor string) (int, error) {
	if valor == "" || strings.TrimSpace(valor) != valor || (len(valor) > 1 && valor[0] == '0') {
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
	q := textoRPTPublica(filtro.q)
	var coinciden any
	total := 0
	if filtro.vista == vistaRPTPuestos {
		filtrados := make([]personaldomain.PuestoRPTPublico, 0, len(catalogo.Puestos))
		for _, puesto := range catalogo.Puestos {
			if filtro.codigoPuesto != "" && puesto.Codigo != filtro.codigoPuesto {
				continue
			}
			if q == "" || strings.Contains(textoRPTPublica(strings.Join([]string{puesto.Codigo, puesto.Denominacion, puesto.CentroCodigo, puesto.Centro, puesto.Delegacion, strings.Join(puesto.Grupos, " "), puesto.Escala, puesto.CategoriaClave, puesto.Tipo, puesto.Provision}, " ")), q) {
				filtrados = append(filtrados, puesto.Clonar())
			}
		}
		total = len(filtrados)
		coinciden = paginaRPT(filtrados, filtro.offset, filtro.limit)
	} else {
		filtradas := make([]personaldomain.CategoriaRPTPublica, 0, len(catalogo.Categorias))
		for _, categoria := range catalogo.Categorias {
			if q == "" || strings.Contains(textoRPTPublica(categoria.Clave+" "+categoria.Denominacion+" "+strings.Join(categoria.Grupos, " ")+" "+strings.Join(categoria.Escalas, " ")), q) {
				filtradas = append(filtradas, categoria.Clonar())
			}
		}
		total = len(filtradas)
		coinciden = paginaRPT(filtradas, filtro.offset, filtro.limit)
	}
	escribirRPTPublicaJSON(w, http.StatusOK, map[string]any{"rpt": map[string]any{"items": coinciden, "total": total, "limit": filtro.limit, "offset": filtro.offset, "vista": filtro.vista, "esquema": catalogo.Esquema, "fuente": catalogo.Fuente, "resumen": catalogo.Resumen}})
}
func paginaRPT[T any](filas []T, offset, limit int) []T {
	inicio := offset
	if inicio > len(filas) {
		inicio = len(filas)
	}
	fin := inicio + limit
	if fin > len(filas) {
		fin = len(filas)
	}
	return filas[inicio:fin]
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
