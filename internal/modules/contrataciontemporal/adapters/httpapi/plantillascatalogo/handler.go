package plantillascatalogo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	pg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres/plantillascatalogo"
	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	RutaCatalogo = "/api/vec/contratacion-temporal/plantillas"
	RutaEntradas = RutaCatalogo + "/entradas"
	RutaPublicar = RutaCatalogo + "/publicar"
	maxCuerpo    = 256 << 10
)

// La composición registra estas rutas en la superficie funcional interna RRHH.
type ResolverActor interface {
	ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error)
}
type Servicio interface {
	Consultar(context.Context, vecdomain.ContextoActor) (app.Lectura, error)
	Editar(context.Context, vecdomain.ContextoActor, app.SolicitudEditar) (app.ResultadoCambio, error)
	Publicar(context.Context, vecdomain.ContextoActor, app.SolicitudPublicar) (app.ResultadoCambio, error)
}

type Manejador struct {
	actor    ResolverActor
	servicio Servicio
}

func NuevoManejador(actor ResolverActor, servicio Servicio) (*Manejador, error) {
	if actor == nil || servicio == nil {
		return nil, app.ErrNoDisponible
	}
	return &Manejador{actor: actor, servicio: servicio}, nil
}

func (h *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	preparar(w)
	if h == nil || h.actor == nil || h.servicio == nil || r == nil || r.URL == nil {
		fallo(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if !rutaValida(r) {
		fallo(w, http.StatusNotFound, "no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", map[bool]string{true: "GET"}[r.URL.Path == RutaCatalogo])
		if r.URL.Path != RutaCatalogo {
			w.Header().Set("Allow", "POST")
		}
		fallo(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.URL.Path == RutaCatalogo && r.Method != http.MethodGet || r.URL.Path != RutaCatalogo && r.Method != http.MethodPost {
		fallo(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !origenPermitido(r) {
		fallo(w, http.StatusForbidden, "denegado")
		return
	}
	if !aceptaJSON(r.Header.Get("Accept")) {
		fallo(w, http.StatusNotAcceptable, "representacion_no_aceptable")
		return
	}
	if r.Method == http.MethodGet && (r.ContentLength > 0 || r.Body != nil && r.Body != http.NoBody) {
		fallo(w, http.StatusBadRequest, "entrada_invalida")
		return
	}
	var editar app.SolicitudEditar
	var publicar app.SolicitudPublicar
	if r.Method == http.MethodPost {
		if r.Header.Get("Content-Type") != "application/json" || len(r.Trailer) != 0 || r.ContentLength > maxCuerpo || r.Body == nil {
			fallo(w, http.StatusBadRequest, "entrada_invalida")
			return
		}
		contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxCuerpo+1))
		if err != nil || len(contenido) == 0 || len(contenido) > maxCuerpo || !utf8.Valid(contenido) || !jsonSinDuplicados(contenido) {
			fallo(w, http.StatusBadRequest, "entrada_invalida")
			return
		}
		var destino any = &editar
		if r.URL.Path == RutaPublicar {
			destino = &publicar
		}
		dec := json.NewDecoder(bytes.NewReader(contenido))
		dec.DisallowUnknownFields()
		if dec.Decode(destino) != nil || dec.Decode(&struct{}{}) != io.EOF {
			fallo(w, http.StatusBadRequest, "entrada_invalida")
			return
		}
	}
	actor, err := h.actor.ResolverContextoActor(r.Context())
	if err != nil || actor.Validar() != nil {
		fallo(w, http.StatusUnauthorized, "no_identificado")
		return
	}
	if r.Method == http.MethodGet {
		l, err := h.servicio.Consultar(r.Context(), actor)
		if err != nil {
			falloServicio(w, err)
			return
		}
		responder(w, http.StatusOK, lecturaDTO(l))
		return
	}
	var resultado app.ResultadoCambio
	if r.URL.Path == RutaEntradas {
		resultado, err = h.servicio.Editar(r.Context(), actor, editar)
	} else {
		resultado, err = h.servicio.Publicar(r.Context(), actor, publicar)
	}
	if err != nil {
		falloServicio(w, err)
		return
	}
	var codigo int
	switch resultado.Recibo.EstadoReplay {
	case "registrado":
		codigo = http.StatusCreated
	case "replay":
		codigo = http.StatusOK
	default:
		fallo(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	responder(w, codigo, cambioDTO{Catalogo: convertirCatalogoDTO(resultado.Catalogo), Recibo: resultado.Recibo})
}

// origenPermitido admite solo peticiones del propio portal. Detrás del proxy
// de entrada el servidor recibe el Host interno (por ejemplo "localhost") y no
// el público, de modo que el Origin del navegador nunca coincide con Host. Si
// el navegador declara Sec-Fetch-Site, decide esa cabecera: solo pasa
// "same-origin". Sin ella se conserva la comparación con Host.
func origenPermitido(r *http.Request) bool {
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("Sec-Fetch-Site")) > 1 {
		return false
	}
	if sitio := r.Header.Get("Sec-Fetch-Site"); sitio != "" {
		return sitio == "same-origin"
	}
	origen := r.Header.Get("Origin")
	if origen == "" || origen == "https://"+r.Host {
		return true
	}
	// El canal HTTP se reserva para pruebas locales. Un host remoto con el
	// mismo nombre pero esquema distinto no constituye el mismo origen.
	return origen == "http://"+r.Host && (r.Host == "localhost" || strings.HasPrefix(r.Host, "localhost:") || r.Host == "127.0.0.1" || strings.HasPrefix(r.Host, "127.0.0.1:"))
}

func rutaValida(r *http.Request) bool {
	u := r.URL
	return (u.Path == RutaCatalogo || u.Path == RutaEntradas || u.Path == RutaPublicar) && u.RawQuery == "" && !u.ForceQuery && u.RawPath == "" && u.Scheme == "" && u.Host == "" && u.User == nil && u.Opaque == "" && u.Fragment == "" && u.RawFragment == "" && u.EscapedPath() == u.Path && !strings.Contains(u.Path, "%")
}

type catalogoDTO struct {
	vecdomain.CatalogoConfigurable
	HuellaSHA256 string `json:"huella_sha256"`
}
type lecturaCatalogoDTO struct {
	Borrador      *catalogoDTO `json:"borrador"`
	Publicado     *catalogoDTO `json:"publicado"`
	PuedeEditar   bool         `json:"puede_editar"`
	PuedePublicar bool         `json:"puede_publicar"`
}
type cambioDTO struct {
	Catalogo catalogoDTO `json:"catalogo"`
	Recibo   app.Recibo  `json:"recibo"`
}

func convertirCatalogoDTO(c vecdomain.CatalogoConfigurable) catalogoDTO {
	h, _ := c.HuellaSHA256()
	return catalogoDTO{c, h}
}
func lecturaDTO(l app.Lectura) lecturaCatalogoDTO {
	z := lecturaCatalogoDTO{PuedeEditar: l.PuedeEditar, PuedePublicar: l.PuedePublicar}
	if l.Borrador != nil {
		c := convertirCatalogoDTO(*l.Borrador)
		z.Borrador = &c
	}
	if l.Publicado != nil {
		c := convertirCatalogoDTO(*l.Publicado)
		z.Publicado = &c
	}
	return z
}

func preparar(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func responder(w http.ResponseWriter, codigo int, valor any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(valor)
}
func fallo(w http.ResponseWriter, codigo int, clave string) {
	responder(w, codigo, map[string]any{"error": map[string]string{"codigo": clave, "clave_i18n": "api.contratacion_temporal.plantillas." + clave}})
}
func falloServicio(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrEntradaInvalida):
		fallo(w, 400, "entrada_invalida")
	case errors.Is(err, app.ErrConflicto):
		fallo(w, 409, "conflicto")
	case errors.Is(err, pg.ErrDenegado), errors.Is(err, vecdomain.ErrAutorizacionDenegada):
		fallo(w, 403, "denegado")
	default:
		fallo(w, 503, "no_disponible")
	}
}
func aceptaJSON(v string) bool {
	return v == "" || v == "*/*" || v == "application/json" || strings.Contains(v, "application/json")
}

func jsonSinDuplicados(b []byte) bool {
	d := json.NewDecoder(bytes.NewReader(b))
	if recorrer(d, 0) != nil {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}
func recorrer(d *json.Decoder, n int) error {
	if n > 8 {
		return app.ErrEntradaInvalida
	}
	t, e := d.Token()
	if e != nil {
		return e
	}
	x, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch x {
	case '{':
		vistas := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			s, ok := k.(string)
			if e != nil || !ok || vistas[s] {
				return app.ErrEntradaInvalida
			}
			vistas[s] = true
			if recorrer(d, n+1) != nil {
				return app.ErrEntradaInvalida
			}
		}
		_, e = d.Token()
		return e
	case '[':
		for d.More() {
			if recorrer(d, n+1) != nil {
				return app.ErrEntradaInvalida
			}
		}
		_, e = d.Token()
		return e
	default:
		return app.ErrEntradaInvalida
	}
}
