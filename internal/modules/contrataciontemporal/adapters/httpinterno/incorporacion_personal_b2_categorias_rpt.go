package httpinterno

import (
	"context"
	"net/http"
	"net/url"
	"regexp"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// RutaCategoriasRPTB2 lista (GET) las categorías RPT publicadas y habilitadas
// del catálogo que fija la configuración B2. Da a la pantalla del vínculo
// CT154 la versión y la huella de publicación que exige su registro.
const RutaCategoriasRPTB2 = "/api/vec/contratacion-temporal/incorporacion-personal-b2/categorias-rpt/v1"

// LimiteCategoriasRPTB2 es el tamaño de página; el lector común admite 100.
const LimiteCategoriasRPTB2 = 100

var patronCursorCategoriasRPTB2 = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)

// LectorCategoriasRPTB2 lo implementa la composición con el lector común de
// categorías RPT y la autoridad nominal B2; el canal sólo trae el cursor.
type LectorCategoriasRPTB2 interface {
	ListarCategoriasRPT(context.Context, string) (PaginaCategoriasRPTB2, error)
}

// CategoriaRPTB2 lleva lo que la pantalla muestra y lo que el registro del
// vínculo copia sin cambiar: versión y huella de la publicación de la que
// procede, fuente y aprobación declaradas en ese documento publicado.
type CategoriaRPTB2 struct {
	CategoriaID          string            `json:"categoria_id"`
	Etiqueta             string            `json:"etiqueta"`
	Atributos            map[string]string `json:"atributos"`
	CatalogoVersion      uint64            `json:"catalogo_version"`
	CatalogoHuellaSHA256 string            `json:"catalogo_huella_sha256"`
	FuenteRef            string            `json:"fuente_ref"`
	AprobacionRef        string            `json:"aprobacion_ref"`
}

func (c CategoriaRPTB2) valida() bool {
	return patronCursorCategoriasRPTB2.MatchString(c.CategoriaID) && c.Etiqueta != "" && c.CatalogoVersion > 0 &&
		patronSHAHTTPB2.MatchString(c.CatalogoHuellaSHA256) && domain.ReferenciaOpacaValida(c.FuenteRef) &&
		domain.ReferenciaOpacaValida(c.AprobacionRef)
}

type PaginaCategoriasRPTB2 struct {
	Categorias      []CategoriaRPTB2 `json:"categorias"`
	HayMas          bool             `json:"hay_mas"`
	SiguienteCursor string           `json:"siguiente_cursor"`
}

// NuevoManejadorCategoriasRPTB2 se monta en la ruta exacta detrás de la
// frontera B2; la identidad y el perfil nominal ya van en el contexto.
func NuevoManejadorCategoriasRPTB2(l LectorCategoriasRPTB2) (http.Handler, error) {
	if dependenciaNula(l) {
		return nil, ErrManejadorIncorporacionPersonalB2
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rutaHTTPCategoriasRPTExacta(r) {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			errorHTTPB2(w, r, 405, "metodo_no_permitido")
			return
		}
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if !cabecerasPropuestaFormalizacionPermitidas(r) || !acceptCompatibleJSON(r.Header) {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		cursor, ok := leerCursorCategoriasRPTB2(r)
		if !ok {
			errorHTTPB2(w, r, 400, "peticion_no_valida")
			return
		}
		pagina, err := l.ListarCategoriasRPT(r.Context(), cursor)
		if r.Context().Err() != nil {
			errorOperacionHTTPB2(w, r, r.Context().Err())
			return
		}
		if err == nil && !paginaCategoriasRPTB2Valida(pagina, cursor) {
			err = ErrManejadorIncorporacionPersonalB2
		}
		if err != nil {
			errorOperacionHTTPB2(w, r, err)
			return
		}
		if pagina.Categorias == nil {
			pagina.Categorias = []CategoriaRPTB2{}
		}
		responderJSONCobertura(w, r, 200, struct {
			Data PaginaCategoriasRPTB2 `json:"data"`
		}{pagina})
	}), nil
}

// leerCursorCategoriasRPTB2 admite sin consulta o sólo «cursor», una vez, con
// la forma de una clave de categoría; sin cuerpo.
func leerCursorCategoriasRPTB2(r *http.Request) (string, bool) {
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || len(r.URL.RawQuery) > 256 {
		return "", false
	}
	if r.URL.RawQuery == "" {
		return "", true
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 1 || len(q["cursor"]) != 1 || !patronCursorCategoriasRPTB2.MatchString(q.Get("cursor")) {
		return "", false
	}
	return q.Get("cursor"), true
}

// Defensa del canal: la página viene del lector común, que ya verificó cada
// publicación; aquí sólo se impide entregar algo incoherente con la consulta.
func paginaCategoriasRPTB2Valida(p PaginaCategoriasRPTB2, cursor string) bool {
	if len(p.Categorias) > LimiteCategoriasRPTB2 {
		return false
	}
	anterior := cursor
	for _, c := range p.Categorias {
		if !c.valida() || c.CategoriaID <= anterior {
			return false
		}
		anterior = c.CategoriaID
	}
	if p.HayMas {
		return len(p.Categorias) > 0 && p.SiguienteCursor == anterior
	}
	return p.SiguienteCursor == ""
}

func rutaHTTPCategoriasRPTExacta(r *http.Request) bool {
	return r != nil && r.URL != nil && r.URL.Path == RutaCategoriasRPTB2 && r.URL.RawPath == "" && r.URL.Scheme == "" && r.URL.Host == "" && r.URL.User == nil && r.URL.Opaque == "" && r.URL.Fragment == "" && r.URL.RawFragment == "" && !r.URL.ForceQuery && r.URL.EscapedPath() == r.URL.Path
}
