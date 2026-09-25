// Package httpinterno traduce la API interna de solo lectura de Calendarios.
// La autenticación y la autorización de la ruta las decide la frontera común
// antes de llegar aquí; este adaptador valida la forma exacta de la petición
// y nunca deduce identidad ni permisos de ella.
package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/calendarios/application"
	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

const (
	RutaCentros          = "/api/vec/calendarios/centros"
	RutaCalendarioCentro = "/api/vec/calendarios/centro"
	RutaPlazo            = "/api/vec/calendarios/plazo"

	maximoConsulta  = 1024
	maximoRespuesta = 1 << 20
	plazoConsulta   = 10 * time.Second
)

// Rutas devuelve las tres rutas exactas en orden estable.
func Rutas() []string { return []string{RutaCentros, RutaCalendarioCentro, RutaPlazo} }

// Manejador sirve las tres rutas. Con consulta nula responde 503 en todas:
// la ausencia de la base nunca se convierte en un calendario supuesto.
type Manejador struct {
	consulta ports.ConsultaCalendarios
}

func NuevoManejador(consulta ports.ConsultaCalendarios) *Manejador {
	return &Manejador{consulta: consulta}
}

func (m *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasSeguras(w)
	if r == nil || r.URL == nil {
		responderError(w, r, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	ruta := r.URL.Path
	if ruta != RutaCentros && ruta != RutaCalendarioCentro && ruta != RutaPlazo {
		responderError(w, r, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderError(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.URL.RawQuery) > maximoConsulta || cabeceraProhibida(r.Header) {
		responderError(w, r, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	if m == nil || m.consulta == nil {
		responderError(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), plazoConsulta)
	defer cancelar()
	var (
		datos any
		err   error
	)
	switch ruta {
	case RutaCentros:
		datos, err = m.centros(ctx, r.URL.RawQuery)
	case RutaCalendarioCentro:
		datos, err = m.calendario(ctx, r.URL.RawQuery)
	default:
		datos, err = m.plazo(ctx, r.URL.RawQuery)
	}
	if err != nil {
		responderFallo(w, r, err)
		return
	}
	responder(w, r, http.StatusOK, map[string]any{"data": datos})
}

var errConsulta = errors.New("calendarios http: consulta invalida")

func (m *Manejador) centros(ctx context.Context, crudo string) (any, error) {
	q, err := parametros(crudo, []string{"anio"}, "conocido_en")
	if err != nil {
		return nil, err
	}
	anio, err := entero(q["anio"])
	if err != nil {
		return nil, err
	}
	conocido, err := instanteOpcional(q["conocido_en"])
	if err != nil {
		return nil, err
	}
	centros, err := m.consulta.Centros(ctx, anio, conocido)
	if err != nil {
		return nil, err
	}
	if centros == nil {
		centros = []ports.CentroConCalendario{}
	}
	return map[string]any{"anio": anio, "centros": centros}, nil
}

func (m *Manejador) calendario(ctx context.Context, crudo string) (any, error) {
	q, err := parametros(crudo, []string{"centro", "anio"}, "conocido_en")
	if err != nil {
		return nil, err
	}
	anio, err := entero(q["anio"])
	if err != nil {
		return nil, err
	}
	conocido, err := instanteOpcional(q["conocido_en"])
	if err != nil || !domain.ReferenciaValida(q["centro"]) {
		return nil, errConsulta
	}
	return m.consulta.CalendarioCentro(ctx, ports.SolicitudCalendarioCentro{CentroRef: q["centro"], Anio: anio, ConocidoEn: conocido})
}

func (m *Manejador) plazo(ctx context.Context, crudo string) (any, error) {
	q, err := parametros(crudo, []string{"unidad", "cantidad", "sede"}, "inicio", "notificado_en", "residencia", "conocido_en")
	if err != nil {
		return nil, err
	}
	sol := ports.SolicitudCalculoPlazo{Unidad: domain.UnidadPlazo(q["unidad"]), MunicipioSede: q["sede"], MunicipioResidencia: q["residencia"]}
	if sol.Cantidad, err = entero(q["cantidad"]); err != nil || !sol.Unidad.Valida() {
		return nil, errConsulta
	}
	if (q["inicio"] == "") == (q["notificado_en"] == "") {
		return nil, errConsulta
	}
	if q["inicio"] != "" {
		if sol.Inicio, err = domain.ParsearFechaCivil(q["inicio"]); err != nil {
			return nil, errConsulta
		}
	} else if sol.NotificadoEn, err = instanteOpcional(q["notificado_en"]); err != nil {
		return nil, errConsulta
	}
	if sol.ConocidoEn, err = instanteOpcional(q["conocido_en"]); err != nil {
		return nil, errConsulta
	}
	return m.consulta.CalcularPlazo(ctx, sol)
}

// parametros exige claves conocidas, sin repetición, obligatorias presentes y
// sin valores vacíos. Los parámetros opcionales no deben aparecer vacíos.
func parametros(crudo string, obligatorios []string, opcionales ...string) (map[string]string, error) {
	valores, err := url.ParseQuery(crudo)
	if err != nil {
		return nil, errConsulta
	}
	admitidos := map[string]bool{}
	for _, k := range obligatorios {
		admitidos[k] = true
	}
	for _, k := range opcionales {
		admitidos[k] = false
	}
	r := map[string]string{}
	for k, v := range valores {
		if _, ok := admitidos[k]; !ok || len(v) != 1 || v[0] == "" || len(v[0]) > 160 {
			return nil, errConsulta
		}
		r[k] = v[0]
	}
	for _, k := range obligatorios {
		if r[k] == "" {
			return nil, errConsulta
		}
	}
	return r, nil
}

func entero(v string) (int, error) {
	if v == "" || len(v) > 6 || strings.TrimLeft(v, "0123456789") != "" || (len(v) > 1 && v[0] == '0') {
		return 0, errConsulta
	}
	return strconv.Atoi(v)
}

// instanteOpcional admite RFC 3339 con zona explícita; vacío es «ahora».
func instanteOpcional(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}, errConsulta
	}
	return t.UTC(), nil
}

func cabeceraProhibida(h http.Header) bool {
	for nombre := range h {
		n := strings.ToLower(nombre)
		switch {
		case n == "cookie", n == "authorization", n == "proxy-authorization", n == "forwarded", n == "remote-user",
			n == "x-remote-user", n == "x-forwarded-user", n == "x-http-method-override", n == "content-encoding",
			strings.HasPrefix(n, "x-auth-"), strings.HasPrefix(n, "x-vec-"), strings.HasPrefix(n, "x-forwarded-"),
			strings.Contains(n, "role"):
			return true
		}
	}
	return false
}

func cabecerasSeguras(w http.ResponseWriter) {
	for _, c := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location"} {
		w.Header().Del(c)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
}

type ambitoFaltante struct {
	Tipo string `json:"tipo"`
	Ref  string `json:"ref"`
}

func responderFallo(w http.ResponseWriter, r *http.Request, err error) {
	var cobertura *domain.ErrorCobertura
	switch {
	case errors.Is(err, errConsulta), errors.Is(err, application.ErrSolicitudInvalida):
		responderError(w, r, http.StatusBadRequest, "solicitud_invalida", nil)
	case errors.As(err, &cobertura):
		faltan := make([]ambitoFaltante, 0, len(cobertura.Faltan))
		for _, a := range cobertura.Faltan {
			faltan = append(faltan, ambitoFaltante{Tipo: string(a.Tipo), Ref: a.Ref})
		}
		responderError(w, r, http.StatusUnprocessableEntity, "calendario_no_publicado", map[string]any{"anio": cobertura.Anio, "faltan": faltan})
	case errors.Is(err, domain.ErrCalculoNoDeterminado):
		// El plazo pedido no puede fijarse con los calendarios publicados: es un
		// resultado de la consulta, no una caída del servicio.
		responderError(w, r, http.StatusUnprocessableEntity, "plazo_no_determinado", nil)
	default:
		responderError(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}

func responderError(w http.ResponseWriter, r *http.Request, estado int, codigo string, detalle map[string]any) {
	cuerpo := map[string]any{"codigo": codigo, "clave_i18n": "api.calendarios.error." + codigo}
	if detalle != nil {
		cuerpo["detalle"] = detalle
	}
	responder(w, r, estado, map[string]any{"error": cuerpo})
}

func responder(w http.ResponseWriter, r *http.Request, estado int, valor any) {
	contenido, err := json.Marshal(valor)
	if err != nil || len(contenido) > maximoRespuesta {
		estado = http.StatusServiceUnavailable
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible","clave_i18n":"api.calendarios.error.servicio_no_disponible"}}`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	if r == nil || r.Method != http.MethodHead {
		_, _ = w.Write(contenido)
	}
}
