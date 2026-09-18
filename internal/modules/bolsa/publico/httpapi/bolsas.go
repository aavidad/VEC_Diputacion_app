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

type manejadorBolsasPublicas struct {
	fuente FuenteBolsasPublicas
}

func NuevoManejadorBolsasPublicas(fuente FuenteBolsasPublicas) (http.Handler, error) {
	if fuente == nil {
		return nil, ErrFuenteBolsasPublicasRequerida
	}
	return &manejadorBolsasPublicas{fuente: fuente}, nil
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
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible", "Servicio no disponible.")
		return
	}
	if r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if !metodoLectura(w, r) {
		return
	}
	if r.URL.RawPath != "" || strings.Contains(r.URL.EscapedPath(), "%") || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		responderError(w, http.StatusBadRequest, "ruta_invalida", "La ruta no es válida.")
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), duracionMaximaOperacionPublica)
	defer cancelar()
	if r.URL.Path == RutaBolsasPublicas {
		if r.URL.RawQuery != "" || r.URL.ForceQuery {
			responderError(w, http.StatusBadRequest, "consulta_invalida", "La consulta no es válida.")
			return
		}
		h.listarBolsas(ctx, w, r)
		return
	}
	bolsaRef, ok := referenciaListaPublica(r.URL.Path)
	if !ok {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado", "Recurso no encontrado.")
		return
	}
	consulta, err := consultaListaPublicaDesde(r.URL.RawQuery)
	if err != nil {
		responderError(w, http.StatusBadRequest, "consulta_invalida", "La consulta no es válida.")
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
		responderErrorBolsasPublicas(w, err)
		return
	}
	var respuesta respuestaBolsasPublicas
	respuesta.Data.Esquema = EsquemaBolsasPublico
	respuesta.Data.GeneradoEn = generadoEn.UTC().Format(time.RFC3339)
	respuesta.Data.Bolsas = make([]bolsaPublicaJSON, 0, len(bolsas))
	for _, bolsa := range bolsas {
		salida, ok := proyectarBolsaPublica(bolsa)
		if !ok {
			responderError(w, http.StatusInternalServerError, "error_interno", "No se ha podido completar la consulta.")
			return
		}
		respuesta.Data.Bolsas = append(respuesta.Data.Bolsas, salida)
	}
	responderJSON(w, r, http.StatusOK, respuesta)
}

func (h *manejadorBolsasPublicas) listarPosiciones(ctx context.Context, w http.ResponseWriter, r *http.Request, bolsaRef string, consulta consultaListaPublica) {
	bolsa, posiciones, generadoEn, err := h.fuente.ListaPublica(ctx, bolsaRef)
	if err != nil {
		responderErrorBolsasPublicas(w, err)
		return
	}
	cabecera, ok := proyectarBolsaPublica(bolsa)
	if !ok || cabecera.BolsaRef != bolsaRef {
		responderError(w, http.StatusInternalServerError, "error_interno", "No se ha podido completar la consulta.")
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
			responderError(w, http.StatusInternalServerError, "error_interno", "No se ha podido completar la consulta.")
			return
		}
		if _, conocida := situacionesPublicas[posicion.EstadoClave]; !conocida {
			responderError(w, http.StatusInternalServerError, "error_interno", "No se ha podido completar la consulta.")
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

func responderErrorBolsasPublicas(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		responderError(w, http.StatusGatewayTimeout, "tiempo_operacion_agotado", "La consulta ha superado el tiempo disponible.")
	case errors.Is(err, context.Canceled):
		responderError(w, http.StatusRequestTimeout, "peticion_cancelada", "La petición fue cancelada.")
	case errors.Is(err, ErrBolsaPublicaNoEncontrada):
		responderError(w, http.StatusNotFound, "bolsa_no_encontrada", "Bolsa de trabajo no encontrada.")
	default:
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible", "No se ha podido completar la consulta.")
	}
}
