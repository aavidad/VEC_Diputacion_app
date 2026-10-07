package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"sort"
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

var patronClaveCategoriaRPTFiltro = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

const (
	vistaRPTCategorias vistaRPTPublica = "categorias"
	vistaRPTPuestos    vistaRPTPublica = "puestos"
	vistaRPTCentros    vistaRPTPublica = "centros"
)

type filtroRPTPublica struct {
	vista                        vistaRPTPublica
	q                            string
	categoriaClave, centroCodigo string
	enlaces                      bool
	limit, offset                int
}

func filtroRPTPublicaDesdePeticion(r *http.Request) (filtroRPTPublica, error) {
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return filtroRPTPublica{}, err
	}
	for clave, valoresClave := range valores {
		if (clave != "vista" && clave != "q" && clave != "limit" && clave != "offset" && clave != "enlaces" && clave != "categoria_clave" && clave != "centro_codigo") || len(valoresClave) != 1 {
			return filtroRPTPublica{}, errors.New("filtro")
		}
	}
	vista := vistaRPTCategorias
	if recibida := valores.Get("vista"); recibida != "" {
		vista = vistaRPTPublica(recibida)
	}
	enlaces := valores.Get("enlaces") == "1"
	if valores.Has("enlaces") && !enlaces || vista != vistaRPTCategorias && vista != vistaRPTPuestos && !(enlaces && vista == vistaRPTCentros) {
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
	categoriaClave, centroCodigo := valores.Get("categoria_clave"), valores.Get("centro_codigo")
	if (valores.Has("categoria_clave") || valores.Has("centro_codigo")) && (!enlaces || vista != vistaRPTPuestos) ||
		categoriaClave != "" && (len(categoriaClave) > 64 || !patronClaveCategoriaRPTFiltro.MatchString(categoriaClave)) ||
		centroCodigo != "" && (len(centroCodigo) > 64 || !utf8.ValidString(centroCodigo) || strings.TrimSpace(centroCodigo) != centroCodigo || strings.ContainsFunc(centroCodigo, unicode.IsControl)) {
		return filtroRPTPublica{}, errors.New("filtro")
	}
	return filtroRPTPublica{vista: vista, q: q, limit: limit, offset: offset, categoriaClave: categoriaClave, centroCodigo: centroCodigo, enlaces: enlaces}, nil
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
	if !filtro.enlaces {
		// El contrato anterior conserva sus campos, recuentos y selección exactos.
		servirRPTPublicaLegacy(w, catalogo, filtro, q)
		return
	}
	puestosPorCategoria := make(map[string]recuentoVinculadoRPT)
	centros := make(map[string]centroVinculadoRPT)
	for _, puesto := range catalogo.Puestos {
		if puesto.CategoriaClave != "" {
			r := puestosPorCategoria[puesto.CategoriaClave]
			r.Puestos++
			r.Dotacion += puesto.Dotacion
			puestosPorCategoria[puesto.CategoriaClave] = r
		}
		c := centros[puesto.CentroCodigo]
		if c.Codigo != "" && c.Denominacion != puesto.Centro {
			escribirRPTPublicaError(w, http.StatusServiceUnavailable, "rpt_publica_no_disponible")
			return
		}
		c.Codigo = puesto.CentroCodigo
		c.Denominacion = puesto.Centro
		c.Puestos++
		c.Dotacion += puesto.Dotacion
		centros[puesto.CentroCodigo] = c
	}
	switch filtro.vista {
	case vistaRPTPuestos:
		filtrados := make([]personaldomain.PuestoRPTPublico, 0, len(catalogo.Puestos))
		for _, puesto := range catalogo.Puestos {
			if filtro.categoriaClave != "" && puesto.CategoriaClave != filtro.categoriaClave || filtro.centroCodigo != "" && puesto.CentroCodigo != filtro.centroCodigo {
				continue
			}
			if q == "" || coincidePuestoRPT(puesto, q) {
				filtrados = append(filtrados, puesto.Clonar())
			}
		}
		total = len(filtrados)
		coinciden = paginaRPT(filtrados, filtro.offset, filtro.limit)
	case vistaRPTCategorias:
		filtradas := make([]categoriaVinculadaRPT, 0, len(catalogo.Categorias))
		for _, categoria := range catalogo.Categorias {
			if q != "" && !strings.Contains(textoRPTPublica(categoria.Clave+" "+categoria.Denominacion+" "+strings.Join(categoria.Grupos, " ")+" "+strings.Join(categoria.Escalas, " ")), q) {
				continue
			}
			r := puestosPorCategoria[categoria.Clave]
			filtradas = append(filtradas, categoriaVinculadaRPT{CategoriaRPTPublica: categoria.Clonar(), PuestosVinculados: r.Puestos, DotacionVinculada: r.Dotacion, RecuentoCoincide: r.Puestos == categoria.Puestos && r.Dotacion == categoria.Dotacion})
		}
		total = len(filtradas)
		coinciden = paginaRPT(filtradas, filtro.offset, filtro.limit)
	case vistaRPTCentros:
		filtrados := make([]centroVinculadoRPT, 0, len(centros))
		for _, centro := range centros {
			if q == "" || strings.Contains(textoRPTPublica(centro.Codigo+" "+centro.Denominacion), q) {
				filtrados = append(filtrados, centro)
			}
		}
		sort.Slice(filtrados, func(i, j int) bool {
			if filtrados[i].Denominacion == filtrados[j].Denominacion {
				return filtrados[i].Codigo < filtrados[j].Codigo
			}
			return filtrados[i].Denominacion < filtrados[j].Denominacion
		})
		total = len(filtrados)
		coinciden = paginaRPT(filtrados, filtro.offset, filtro.limit)
	}
	escribirRPTPublicaJSON(w, http.StatusOK, map[string]any{"rpt": map[string]any{"items": coinciden, "total": total, "limit": filtro.limit, "offset": filtro.offset, "vista": filtro.vista, "esquema": catalogo.Esquema, "fuente": catalogo.Fuente, "resumen": catalogo.Resumen, "enlaces": true}})
}

type recuentoVinculadoRPT struct{ Puestos, Dotacion int }
type centroVinculadoRPT struct {
	Codigo       string `json:"codigo"`
	Denominacion string `json:"denominacion"`
	Puestos      int    `json:"puestos"`
	Dotacion     int    `json:"dotacion"`
}
type categoriaVinculadaRPT struct {
	personaldomain.CategoriaRPTPublica
	PuestosVinculados int  `json:"puestos_vinculados"`
	DotacionVinculada int  `json:"dotacion_vinculada"`
	RecuentoCoincide  bool `json:"recuento_coincide"`
}

func coincidePuestoRPT(puesto personaldomain.PuestoRPTPublico, q string) bool {
	return strings.Contains(textoRPTPublica(strings.Join([]string{puesto.Codigo, puesto.Denominacion, puesto.CentroCodigo, puesto.Centro, puesto.Delegacion, strings.Join(puesto.Grupos, " "), puesto.Escala, puesto.CategoriaClave, puesto.Tipo, puesto.Provision}, " ")), q)
}
func servirRPTPublicaLegacy(w http.ResponseWriter, catalogo personaldomain.CatalogoRPTPublica, filtro filtroRPTPublica, q string) {
	var coinciden any
	total := 0
	if filtro.vista == vistaRPTPuestos {
		filtrados := make([]personaldomain.PuestoRPTPublico, 0, len(catalogo.Puestos))
		for _, puesto := range catalogo.Puestos {
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
