package adminselector

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

const maxCuerpo = 1024
const maxRevisionJSON = 1<<53 - 1

type Handler struct {
	// host es la autoridad exacta de la cabecera Host (con puerto si no es
	// 443); nombre, sin puerto, es lo que la frontera ADMIN observa y contrasta
	// con host_admin de la política de certificado.
	origen, host, nombre, audiencia string
	observador                      FuenteObservacion
	seleccionador                   Seleccionador
	auditor                         api.AuditorFrontera
	reloj                           httpseguridad.Reloj
}

func NuevoHandler(origen, audiencia string, observador FuenteObservacion, seleccionador Seleccionador,
	auditor api.AuditorFrontera, reloj httpseguridad.Reloj) (*Handler, error) {
	u, err := url.Parse(origen)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.Hostname() == "" || audiencia == "" ||
		nulo(observador) || nulo(seleccionador) || nulo(auditor) || nulo(reloj) {
		return nil, api.ErrConfiguracionIncompleta
	}
	return &Handler{origen: origen, host: u.Host, nombre: u.Hostname(), audiencia: audiencia, observador: observador,
		seleccionador: seleccionador, auditor: auditor, reloj: reloj}, nil
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || nulo(h.observador) || nulo(h.seleccionador) || nulo(h.auditor) || nulo(h.reloj) {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r == nil || r.URL == nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Host != h.host || cabecerasLibres(r) {
		h.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida")
		return
	}
	if !h.navegacionValida(r) {
		h.denegar(w, r, http.StatusForbidden, "acceso_denegado")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.EscapedPath() != r.URL.Path {
		h.denegar(w, r, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.denegar(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Method == http.MethodGet && r.URL.Path != RutaPropios || r.Method == http.MethodPost && r.URL.Path != RutaSeleccion {
		h.denegar(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	o, err := h.observador.ObservarADMIN(r.Context(), r)
	if err != nil {
		h.errorObservacion(w, r, err)
		return
	}
	if o.Host != h.nombre || o.Audiencia != h.audiencia || !o.Valida(h.reloj.Ahora().UTC()) {
		h.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida")
		return
	}
	if r.Method == http.MethodGet {
		if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || tieneCuerpo(r) {
			h.denegar(w, r, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		propios, err := h.seleccionador.ListarPropiosADMIN(r.Context(), o)
		if err != nil {
			h.falloSeleccionador(w, r, err)
			return
		}
		lista := PerfilesPropios{Revision: propios.Revision, PerfilActivoRef: propios.PerfilActivoRef}
		for _, p := range propios.Perfiles {
			lista.Perfiles = append(lista.Perfiles, PerfilPropio{PerfilRef: p.PerfilRef, RolVersionRef: p.RolVersionRef,
				ClaveI18N: p.ClaveI18N, CategoriaADMIN: p.CategoriaADMIN})
		}
		if !lista.valida() {
			fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		if lista.Perfiles == nil {
			lista.Perfiles = []PerfilPropio{}
		}
		respuesta(w, http.StatusOK, lista)
		return
	}
	media, parametros, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if len(r.Header.Values("Content-Type")) != 1 || err != nil || media != "application/json" ||
		len(parametros) > 1 || len(parametros) == 1 && !strings.EqualFold(parametros["charset"], "utf-8") ||
		len(r.Header.Values("Content-Encoding")) != 0 {
		h.denegar(w, r, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	perfil, revision, estado := decodificar(w, r)
	if estado != 0 {
		h.denegar(w, r, estado, "solicitud_invalida")
		return
	}
	seleccion, err := h.seleccionador.SeleccionarPerfilADMIN(r.Context(), o, perfil, revision)
	if err != nil {
		h.falloSeleccionador(w, r, err)
		return
	}
	if seleccion.PerfilActivoRef != perfil || seleccion.Revision == 0 || seleccion.Revision > maxRevisionJSON ||
		seleccion.Revision < revision || seleccion.Revision > revision+1 ||
		!auditoriaValida(seleccion.AuditoriaRef) || seleccion.SeleccionadaEn.IsZero() || seleccion.SeleccionadaEn.After(h.reloj.Ahora()) {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	respuesta(w, http.StatusOK, Seleccion{PerfilActivoRef: seleccion.PerfilActivoRef, Revision: seleccion.Revision,
		AuditoriaRef: seleccion.AuditoriaRef, SeleccionadaEn: seleccion.SeleccionadaEn})
}

func cabecerasLibres(r *http.Request) bool {
	for nombre := range r.Header {
		n := strings.ToLower(nombre)
		if n == "authorization" || n == "proxy-authorization" || n == "cookie" || n == "cookie2" ||
			n == "forwarded" || strings.HasPrefix(n, "x-forwarded-") || strings.HasPrefix(n, "x-ssl-") ||
			strings.HasPrefix(n, "x-client-") || strings.HasPrefix(n, "x-remote-") || strings.HasPrefix(n, "x-auth-") ||
			n == "x-perfil-activo" || n == "x-actor-persona" || n == "x-user" || n == "x-role" {
			return true
		}
	}
	return false
}

func (h *Handler) navegacionValida(r *http.Request) bool {
	origenes := r.Header.Values("Origin")
	return len(origenes) <= 1 && (len(origenes) == 0 && r.Method == http.MethodGet || len(origenes) == 1 && origenes[0] == h.origen) &&
		len(r.Header.Values("Sec-Fetch-Site")) == 1 && r.Header.Get("Sec-Fetch-Site") == "same-origin" &&
		len(r.Header.Values("Sec-Fetch-Mode")) == 1 && (r.Header.Get("Sec-Fetch-Mode") == "cors" || r.Header.Get("Sec-Fetch-Mode") == "same-origin") &&
		len(r.Header.Values("Sec-Fetch-Dest")) == 1 && r.Header.Get("Sec-Fetch-Dest") == "empty"
}

func tieneCuerpo(r *http.Request) bool {
	if r.Body == nil {
		return false
	}
	var b [1]byte
	n, err := io.ReadFull(r.Body, b[:])
	return n != 0 || err != io.EOF
}

// Tokenizar los dos campos evita la regla last-wins de encoding/json.
func decodificar(w http.ResponseWriter, r *http.Request) (string, uint64, int) {
	if r.ContentLength > maxCuerpo {
		return "", 0, http.StatusRequestEntityTooLarge
	}
	if r.Body == nil {
		return "", 0, http.StatusBadRequest
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCuerpo))
	dec.UseNumber()
	invalid := func(err error) (string, uint64, int) {
		var exceso *http.MaxBytesError
		if errors.As(err, &exceso) {
			return "", 0, http.StatusRequestEntityTooLarge
		}
		return "", 0, http.StatusBadRequest
	}
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return invalid(err)
	}
	var perfil string
	var revision uint64
	vistos := make(map[string]bool, 2)
	for dec.More() {
		tok, err = dec.Token()
		if err != nil {
			return invalid(err)
		}
		clave, ok := tok.(string)
		if !ok || vistos[clave] {
			return invalid(nil)
		}
		vistos[clave] = true
		switch clave {
		case "perfil_ref":
			err = dec.Decode(&perfil)
		case "revision_esperada":
			var valorToken any
			valorToken, err = dec.Token()
			if err == nil {
				numero, ok := valorToken.(json.Number)
				if !ok {
					return invalid(nil)
				}
				var valor int64
				valor, err = numero.Int64()
				if valor < 0 || valor > maxRevisionJSON {
					return invalid(nil)
				}
				revision = uint64(valor)
			}
		default:
			return invalid(nil)
		}
		if err != nil {
			return invalid(err)
		}
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return invalid(err)
	}
	if _, err = dec.Token(); err != io.EOF {
		return invalid(err)
	}
	if len(vistos) != 2 || !referencia(perfil, "prf_") {
		return invalid(nil)
	}
	return perfil, revision, 0
}

func referencia(v, prefijo string) bool {
	if !strings.HasPrefix(v, prefijo) || len(v) < len(prefijo)+22 || len(v) > len(prefijo)+128 {
		return false
	}
	for _, c := range v[len(prefijo):] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func codigoOpaco(v string) bool {
	if v == "" || len(v) > 512 {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == ':' || c == '.') {
			return false
		}
	}
	return true
}

// auditoriaValida admite las dos formas que entrega el selector: la referencia
// de la auditoría común (IS14/AD171, «aud_v3_p_» y 32 hex) y la heredada
// propia del selector («auditoria_seleccion_admin:» y 64 hex).
func auditoriaValida(v string) bool {
	return hexMinusculaConPrefijo(v, "aud_v3_p_", 32) || hexMinusculaConPrefijo(v, "auditoria_seleccion_admin:", 64)
}

func hexMinusculaConPrefijo(v, prefijo string, n int) bool {
	if !strings.HasPrefix(v, prefijo) || len(v) != len(prefijo)+n {
		return false
	}
	for _, c := range v[len(prefijo):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (p PerfilesPropios) valida() bool {
	if p.Revision > maxRevisionJSON || len(p.Perfiles) > 16 || p.PerfilActivoRef != "" && p.Revision == 0 {
		return false
	}
	vistos := map[string]bool{}
	activo := p.PerfilActivoRef == ""
	for _, perfil := range p.Perfiles {
		if !referencia(perfil.PerfilRef, "prf_") || vistos[perfil.PerfilRef] || !codigoOpaco(perfil.RolVersionRef) ||
			!codigoOpaco(perfil.ClaveI18N) || len(perfil.ClaveI18N) > 256 ||
			(perfil.CategoriaADMIN != "aplicacion" && perfil.CategoriaADMIN != "sistemas") {
			return false
		}
		vistos[perfil.PerfilRef] = true
		activo = activo || perfil.PerfilRef == p.PerfilActivoRef
	}
	return activo
}

func (h *Handler) denegar(w http.ResponseWriter, r *http.Request, estado int, codigo string) {
	accion := "seleccionar_perfil_propio"
	recurso := RutaSeleccion
	if r.Method == http.MethodGet {
		accion = "consultar_perfiles_propios"
		recurso = RutaPropios
	}
	if h.auditor.RegistrarDenegacionADMIN(r.Context(), api.DenegacionADMIN{
		Codigo: codigo, Accion: accion, RecursoRef: recurso,
	}) != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	fallo(w, estado, codigo)
}

func (h *Handler) errorObservacion(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, api.ErrAutenticacionRequerida) {
		h.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida")
	} else if errors.Is(err, api.ErrAccesoDenegado) || errors.Is(err, domain.ErrAutorizacionDenegada) {
		h.denegar(w, r, http.StatusForbidden, "acceso_denegado")
	} else {
		h.denegar(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
}

func (h *Handler) falloSeleccionador(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, api.ErrAutenticacionRequerida):
		h.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida")
	case errors.Is(err, api.ErrAccesoDenegado), errors.Is(err, domain.ErrAutorizacionDenegada):
		h.denegar(w, r, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, api.ErrConflictoEstado):
		h.denegar(w, r, http.StatusConflict, "conflicto_estado")
	default:
		h.denegar(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
}

func fallo(w http.ResponseWriter, estado int, codigo string) {
	respuesta(w, estado, struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}{
		Error: struct {
			Codigo string `json:"codigo"`
		}{codigo},
	})
}

func respuesta(w http.ResponseWriter, estado int, v any) {
	for _, cabecera := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Access-Control-Allow-Headers", "Access-Control-Allow-Methods", "Access-Control-Expose-Headers", "Location"} {
		w.Header().Del(cabecera)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}
