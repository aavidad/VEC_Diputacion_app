package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
	"vec-diputacion-granada/internal/shared/i18n"
	textos "vec-diputacion-granada/web"
)

// Consulta pública de bolsas de trabajo (B10): relación de bolsas en vigor y
// lista ordenada de aspirantes de cada bolsa con documento enmascarado y
// situación. Anónima; nunca nombres, contactos ni referencias internas de
// participación. Contrato: web/static/bolsa/contrato-publico-bolsas.js.
const (
	RutaBolsasPublicas   = "/api/publico/bolsa/bolsas"
	EsquemaBolsasPublico = "vec.bolsa.publico.bolsas.v1"
	EsquemaListaPublico  = "vec.bolsa.publico.lista.v1"

	limitePorDefectoListaPublica = 50
	limiteMaximoListaPublica     = 100
	maximoPosicionesBolsaPublica = 100000
)

var (
	ErrFuenteBolsasPublicasRequerida  = errors.New("bolsa publico http: fuente de bolsas requerida")
	ErrBolsaPublicaNoEncontrada       = errors.New("bolsa publico: bolsa no encontrada")
	ErrConsultaBolsasPublicasInvalida = errors.New("bolsa publico: consulta invalida")

	patronDocumentoEnmascaradoPublico = regexp.MustCompile(`^\*{3}[0-9]{4}\*{2}$`)
	patronReferenciaBolsaPublica      = regexp.MustCompile(`^[a-z0-9][a-z0-9:._-]{2,159}$`)
	situacionesPublicas               = map[string]struct{}{"disponible": {}, "ocupado": {}, "no_disponible": {}, "excluido": {}, "renuncia_pendiente": {}}
)

type BolsaPublica struct {
	BolsaRef     string
	Categoria    string
	Grupos       []string
	TipoLista    string
	VigenteDesde time.Time
	VigenteHasta *time.Time
	Total        int
}

type PosicionPublica struct {
	Orden                int
	DocumentoEnmascarado string
	EstadoClave          string
}

// FuenteBolsasPublicas entrega las bolsas en vigor y sus posiciones ya
// minimizadas. El instante es el de generación del dato servido.
type FuenteBolsasPublicas interface {
	BolsasPublicas(context.Context) ([]BolsaPublica, time.Time, error)
	ListaPublica(context.Context, string) (BolsaPublica, []PosicionPublica, time.Time, error)
}

// FuentePaginaBolsasPublicas es opcional: entrega solo el tramo pedido de la
// lista (como mucho cantidad posiciones desde la indicada), ya cotejado con la
// lista completa por la fuente. Evita leer miles de posiciones para servir
// cincuenta. La búsqueda por documento sigue usando ListaPublica.
type FuentePaginaBolsasPublicas interface {
	PaginaListaPublica(ctx context.Context, bolsaRef string, desde, cantidad int) (BolsaPublica, []PosicionPublica, time.Time, error)
}

type manejadorBolsasPublicas struct {
	fuente   FuenteBolsasPublicas
	catalogo *i18n.Catalog
	idiomas  []string
	selector language.Matcher
}

func NuevoManejadorBolsasPublicas(fuente FuenteBolsasPublicas) (http.Handler, error) {
	if fuente == nil {
		return nil, ErrFuenteBolsasPublicasRequerida
	}
	manejador, err := nuevoManejadorBolsasPublicasI18n()
	if err != nil {
		return nil, err
	}
	manejador.fuente = fuente
	return manejador, nil
}

func nuevoManejadorBolsasPublicasI18n() (*manejadorBolsasPublicas, error) {
	catalogo, err := textos.CatalogoBolsasPublicas()
	if err != nil {
		return nil, err
	}
	idiomas := []string{catalogo.DefaultLocale()}
	for _, idioma := range catalogo.Locales() {
		if idioma != catalogo.DefaultLocale() {
			idiomas = append(idiomas, idioma)
		}
	}
	etiquetas := make([]language.Tag, 0, len(idiomas))
	for _, idioma := range idiomas {
		etiquetas = append(etiquetas, language.Make(idioma))
	}
	return &manejadorBolsasPublicas{catalogo: catalogo, idiomas: idiomas, selector: language.NewMatcher(etiquetas)}, nil
}

// RegistrarRutasBolsasPublicas aplica la lista positiva de la consulta.
func RegistrarRutasBolsasPublicas(mux *http.ServeMux, manejador http.Handler) {
	if mux == nil || manejador == nil {
		return
	}
	mux.Handle(RutaBolsasPublicas, manejador)
	mux.Handle(RutaBolsasPublicas+"/", manejador)
}

type bolsaPublicaJSON struct {
	BolsaRef     string   `json:"bolsa_ref"`
	Categoria    string   `json:"categoria"`
	Grupos       []string `json:"grupos"`
	TipoLista    string   `json:"tipo_lista"`
	VigenteDesde string   `json:"vigente_desde"`
	VigenteHasta *string  `json:"vigente_hasta"`
	Total        int      `json:"total"`
}

type posicionPublicaJSON struct {
	Orden                int    `json:"orden"`
	DocumentoEnmascarado string `json:"documento_enmascarado"`
	EstadoClave          string `json:"estado_clave"`
}

type respuestaBolsasPublicas struct {
	Data struct {
		Esquema    string             `json:"esquema"`
		GeneradoEn string             `json:"generado_en"`
		Bolsas     []bolsaPublicaJSON `json:"bolsas"`
	} `json:"data"`
}

type respuestaListaPublica struct {
	Data struct {
		Esquema         string                `json:"esquema"`
		GeneradoEn      string                `json:"generado_en"`
		Bolsa           bolsaPublicaJSON      `json:"bolsa"`
		Posiciones      []posicionPublicaJSON `json:"posiciones"`
		HayMas          bool                  `json:"hay_mas"`
		CursorSiguiente *string               `json:"cursor_siguiente"`
	} `json:"data"`
}

type consultaListaPublica struct {
	limite    int
	desde     int
	documento string
}

func (h *manejadorBolsasPublicas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || h == nil || h.fuente == nil {
		h.responderError(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		h.responderError(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.URL.RawPath != "" || strings.Contains(r.URL.EscapedPath(), "%") || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		h.responderError(w, r, http.StatusBadRequest, "ruta_invalida")
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), duracionMaximaOperacionPublica)
	defer cancelar()
	if r.URL.Path == RutaBolsasPublicas {
		if r.URL.RawQuery != "" || r.URL.ForceQuery {
			h.responderError(w, r, http.StatusBadRequest, "consulta_invalida")
			return
		}
		h.listarBolsas(ctx, w, r)
		return
	}
	bolsaRef, ok := referenciaListaPublica(r.URL.Path)
	if !ok {
		h.responderError(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	consulta, err := consultaListaPublicaDesde(r.URL.RawQuery)
	if err != nil {
		h.responderError(w, r, http.StatusBadRequest, "consulta_invalida")
		return
	}
	h.listarPosiciones(ctx, w, r, bolsaRef, consulta)
}

func referenciaListaPublica(ruta string) (string, bool) {
	const prefijo, sufijo = RutaBolsasPublicas + "/", "/lista"
	if !strings.HasPrefix(ruta, prefijo) || !strings.HasSuffix(ruta, sufijo) {
		return "", false
	}
	ref := strings.TrimSuffix(strings.TrimPrefix(ruta, prefijo), sufijo)
	return ref, patronReferenciaBolsaPublica.MatchString(ref)
}

// consultaListaPublicaDesde admite limite (1..100), cursor (posición a partir
// de la que continuar, emitido por esta misma consulta) y documento
// enmascarado exacto. Cualquier otro parámetro o repetición se rechaza.
func consultaListaPublicaDesde(rawQuery string) (consultaListaPublica, error) {
	consulta := consultaListaPublica{limite: limitePorDefectoListaPublica, desde: 1}
	if rawQuery == "" {
		return consulta, nil
	}
	valores, err := url.ParseQuery(rawQuery)
	if err != nil || strings.Contains(rawQuery, ";") {
		return consultaListaPublica{}, ErrConsultaBolsasPublicasInvalida
	}
	for clave, lista := range valores {
		if len(lista) != 1 || lista[0] == "" {
			return consultaListaPublica{}, ErrConsultaBolsasPublicasInvalida
		}
		valor := lista[0]
		switch clave {
		case "limite":
			limite, err := strconv.Atoi(valor)
			if err != nil || limite < 1 || limite > limiteMaximoListaPublica || strconv.Itoa(limite) != valor {
				return consultaListaPublica{}, ErrConsultaBolsasPublicasInvalida
			}
			consulta.limite = limite
		case "cursor":
			desde, err := strconv.Atoi(valor)
			if err != nil || desde < 2 || desde > maximoPosicionesBolsaPublica || strconv.Itoa(desde) != valor {
				return consultaListaPublica{}, ErrConsultaBolsasPublicasInvalida
			}
			consulta.desde = desde
		case "documento":
			if !patronDocumentoEnmascaradoPublico.MatchString(valor) {
				return consultaListaPublica{}, ErrConsultaBolsasPublicasInvalida
			}
			consulta.documento = valor
		default:
			return consultaListaPublica{}, ErrConsultaBolsasPublicasInvalida
		}
	}
	return consulta, nil
}

func (h *manejadorBolsasPublicas) listarBolsas(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	bolsas, generadoEn, err := h.fuente.BolsasPublicas(ctx)
	if err != nil {
		h.responderErrorFuente(w, r, err)
		return
	}
	var respuesta respuestaBolsasPublicas
	respuesta.Data.Esquema = EsquemaBolsasPublico
	respuesta.Data.GeneradoEn = generadoEn.UTC().Format(time.RFC3339)
	respuesta.Data.Bolsas = make([]bolsaPublicaJSON, 0, len(bolsas))
	for _, bolsa := range bolsas {
		salida, ok := proyectarBolsaPublica(bolsa)
		if !ok {
			h.responderError(w, r, http.StatusInternalServerError, "error_interno")
			return
		}
		respuesta.Data.Bolsas = append(respuesta.Data.Bolsas, salida)
	}
	responderJSON(w, r, http.StatusOK, respuesta)
}

func (h *manejadorBolsasPublicas) listarPosiciones(ctx context.Context, w http.ResponseWriter, r *http.Request, bolsaRef string, consulta consultaListaPublica) {
	var (
		bolsa      BolsaPublica
		posiciones []PosicionPublica
		generadoEn time.Time
		err        error
	)
	if paginada, ok := h.fuente.(FuentePaginaBolsasPublicas); ok && consulta.documento == "" {
		bolsa, posiciones, generadoEn, err = paginada.PaginaListaPublica(ctx, bolsaRef, consulta.desde, consulta.limite+1)
	} else {
		bolsa, posiciones, generadoEn, err = h.fuente.ListaPublica(ctx, bolsaRef)
	}
	if err != nil {
		h.responderErrorFuente(w, r, err)
		return
	}
	cabecera, ok := proyectarBolsaPublica(bolsa)
	if !ok || cabecera.BolsaRef != bolsaRef {
		h.responderError(w, r, http.StatusInternalServerError, "error_interno")
		return
	}
	var respuesta respuestaListaPublica
	respuesta.Data.Esquema = EsquemaListaPublico
	respuesta.Data.GeneradoEn = generadoEn.UTC().Format(time.RFC3339)
	respuesta.Data.Bolsa = cabecera
	respuesta.Data.Posiciones = make([]posicionPublicaJSON, 0, consulta.limite)
	ordenAnterior := 0
	for _, posicion := range posiciones {
		if posicion.Orden <= ordenAnterior || posicion.Orden > maximoPosicionesBolsaPublica ||
			!patronDocumentoEnmascaradoPublico.MatchString(posicion.DocumentoEnmascarado) {
			h.responderError(w, r, http.StatusInternalServerError, "error_interno")
			return
		}
		if _, conocida := situacionesPublicas[posicion.EstadoClave]; !conocida {
			h.responderError(w, r, http.StatusInternalServerError, "error_interno")
			return
		}
		ordenAnterior = posicion.Orden
		if posicion.Orden < consulta.desde || (consulta.documento != "" && posicion.DocumentoEnmascarado != consulta.documento) {
			continue
		}
		if len(respuesta.Data.Posiciones) == consulta.limite {
			respuesta.Data.HayMas = true
			cursor := strconv.Itoa(posicion.Orden)
			respuesta.Data.CursorSiguiente = &cursor
			break
		}
		respuesta.Data.Posiciones = append(respuesta.Data.Posiciones, posicionPublicaJSON{
			Orden: posicion.Orden, DocumentoEnmascarado: posicion.DocumentoEnmascarado, EstadoClave: posicion.EstadoClave,
		})
	}
	responderJSON(w, r, http.StatusOK, respuesta)
}

func proyectarBolsaPublica(bolsa BolsaPublica) (bolsaPublicaJSON, bool) {
	if !patronReferenciaBolsaPublica.MatchString(bolsa.BolsaRef) || strings.TrimSpace(bolsa.Categoria) == "" ||
		strings.TrimSpace(bolsa.TipoLista) == "" || bolsa.VigenteDesde.IsZero() || bolsa.Total < 0 {
		return bolsaPublicaJSON{}, false
	}
	grupos := make([]string, 0, len(bolsa.Grupos))
	for _, grupo := range bolsa.Grupos {
		if strings.TrimSpace(grupo) == "" {
			return bolsaPublicaJSON{}, false
		}
		grupos = append(grupos, grupo)
	}
	salida := bolsaPublicaJSON{
		BolsaRef: bolsa.BolsaRef, Categoria: bolsa.Categoria, Grupos: grupos, TipoLista: bolsa.TipoLista,
		VigenteDesde: bolsa.VigenteDesde.UTC().Format(time.RFC3339), Total: bolsa.Total,
	}
	if bolsa.VigenteHasta != nil {
		hasta := bolsa.VigenteHasta.UTC().Format(time.RFC3339)
		salida.VigenteHasta = &hasta
	}
	return salida, true
}

func (h *manejadorBolsasPublicas) responderErrorFuente(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		h.responderError(w, r, http.StatusGatewayTimeout, "tiempo_operacion_agotado")
	case errors.Is(err, context.Canceled):
		h.responderError(w, r, http.StatusRequestTimeout, "peticion_cancelada")
	case errors.Is(err, ErrBolsaPublicaNoEncontrada):
		h.responderError(w, r, http.StatusNotFound, "bolsa_no_encontrada")
	default:
		h.responderError(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
}

func (h *manejadorBolsasPublicas) responderError(w http.ResponseWriter, r *http.Request, estado int, codigo string) {
	catalogo := h
	if catalogo == nil || catalogo.catalogo == nil {
		// También una instancia vacía responde desde los datos i18n compilados.
		manejador, err := nuevoManejadorBolsasPublicasI18n()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		catalogo = manejador
	}
	preferencia := ""
	if r != nil {
		preferencia = r.Header.Get("Accept-Language")
	}
	_, indice := language.MatchStrings(catalogo.selector, preferencia)
	idioma := catalogo.idiomas[indice]
	w.Header().Set("Content-Language", idioma)
	w.Header().Add("Vary", "Accept-Language")
	responderError(w, estado, codigo, catalogo.catalogo.T(idioma, codigo))
}
