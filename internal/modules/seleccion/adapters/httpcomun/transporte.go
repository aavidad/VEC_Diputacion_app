// Package httpcomun reúne las reglas de transporte HTTP comunes a las dos
// superficies de Selección (personal e interna de RRHH): cabeceras
// prohibidas, lectura acotada de JSON, respuestas sin caché y traducción de
// errores a códigos estables (claves que la web traduce; nunca texto).
package httpcomun

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/baremacion"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// ErrAutenticacionAusente lo devuelve la frontera de identidad cuando la
// petición no trae una identidad verificada: 401.
var ErrAutenticacionAusente = errors.New("seleccion: autenticacion ausente")

// FormatoInstante de las respuestas (UTC, microsegundos).
const FormatoInstante = "2006-01-02T15:04:05.000000Z"

var claveIdempotencia = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

// Nula informa de una dependencia ausente, también tras una interfaz.
func Nula(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}

// CabeceraProhibida rechaza cookies e identidades de cabecera libre: la
// identidad solo sale de la frontera mTLS.
func CabeceraProhibida(h http.Header) bool {
	for k := range h {
		l := strings.ToLower(k)
		if l == "cookie" || l == "authorization" || l == "proxy-authorization" || strings.HasPrefix(l, "x-vec-") || strings.HasPrefix(l, "x-auth-") ||
			strings.HasPrefix(l, "x-forwarded-") || l == "x-remote-user" || l == "remote-user" {
			return true
		}
	}
	return false
}

// CuerpoAusente admite el GET sin cuerpo de HTTP/1.1 y HTTP/2.
func CuerpoAusente(r *http.Request) bool {
	if r.Body == nil || r.Body == http.NoBody {
		return true
	}
	return r.ProtoMajor >= 2 && r.ContentLength == 0
}

// RutaExacta exige la ruta sin escapes ni variantes.
func RutaExacta(r *http.Request, ruta string) bool {
	return r != nil && r.URL != nil && r.URL.Path == ruta && r.URL.RawPath == "" && r.URL.EscapedPath() == ruta && !r.URL.ForceQuery
}

// LecturaPermitida valida un GET sin cuerpo ni cabeceras prohibidas; query
// solo admite los parámetros dados, una vez cada uno.
func LecturaPermitida(r *http.Request, parametros ...string) (map[string]string, bool) {
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || !CuerpoAusente(r) || CabeceraProhibida(r.Header) {
		return nil, false
	}
	valores := r.URL.Query()
	if len(valores) != len(parametros) {
		return nil, false
	}
	resultado := make(map[string]string, len(parametros))
	for _, p := range parametros {
		v, ok := valores[p]
		if !ok || len(v) != 1 || v[0] == "" || len(v[0]) > 200 {
			return nil, false
		}
		resultado[p] = v[0]
	}
	return resultado, true
}

// LeerEscritura valida cabeceras y lee el JSON de una escritura (sin campos
// desconocidos). Devuelve la clave de idempotencia si se exige.
func LeerEscritura(w http.ResponseWriter, r *http.Request, maximo int64, destino any, conClave bool) (string, bool) {
	sitio := r.Header.Get("Sec-Fetch-Site")
	if CabeceraProhibida(r.Header) || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" ||
		(sitio != "" && sitio != "same-origin") || r.URL.RawQuery != "" || r.ContentLength < 2 || r.ContentLength > maximo ||
		len(r.TransferEncoding) != 0 || r.Body == nil {
		Responder(w, http.StatusBadRequest, Error("peticion_no_permitida"))
		return "", false
	}
	clave := r.Header.Get("Idempotency-Key")
	if conClave && (len(r.Header.Values("Idempotency-Key")) != 1 || !claveIdempotencia.MatchString(clave)) {
		Responder(w, http.StatusBadRequest, Error("peticion_no_permitida"))
		return "", false
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximo))
	if err != nil || int64(len(contenido)) != r.ContentLength {
		Responder(w, http.StatusBadRequest, Error("peticion_no_permitida"))
		return "", false
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(destino) != nil || decodificador.More() {
		Responder(w, http.StatusBadRequest, Error("datos_no_validos"))
		return "", false
	}
	return clave, true
}

// Metodo responde 405 si el método no es el esperado.
func Metodo(w http.ResponseWriter, r *http.Request, metodos ...string) bool {
	for _, m := range metodos {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(metodos, ", "))
	Responder(w, http.StatusMethodNotAllowed, Error("metodo_no_permitido"))
	return false
}

type cuerpoError struct {
	Error struct {
		Codigo string `json:"codigo"`
	} `json:"error"`
}

// Error construye el cuerpo de error con su código estable.
func Error(codigo string) any {
	var e cuerpoError
	e.Error.Codigo = codigo
	return e
}

// Datos envuelve una respuesta correcta.
func Datos(v any) any {
	return struct {
		Data any `json:"data"`
	}{v}
}

// Responder escribe JSON sin caché ni rastreo.
func Responder(w http.ResponseWriter, estado int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		estado = http.StatusInternalServerError
		b = []byte(`{"error":{"codigo":"error_interno"}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(estado)
	_, _ = w.Write(b)
}

// ResponderError traduce los errores del caso de uso a códigos estables.
func ResponderError(w http.ResponseWriter, err error) {
	for _, c := range []struct {
		err    error
		estado int
		codigo string
	}{
		{ErrAutenticacionAusente, http.StatusUnauthorized, "autenticacion_requerida"},
		{ports.ErrNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{dominiovec.ErrAutorizacionDenegada, http.StatusForbidden, "acceso_denegado"},
		{dominiovec.ErrPermissionDenied, http.StatusForbidden, "acceso_denegado"},
		{ports.ErrFueraDePlazo, http.StatusConflict, "fuera_de_plazo"},
		{ports.ErrClaveReutilizada, http.StatusConflict, "clave_reutilizada"},
		{ports.ErrVersionObsoleta, http.StatusConflict, "version_obsoleta"},
		{ports.ErrYaPresentada, http.StatusConflict, "ya_presentada"},
		{ports.ErrConvocatoriaActualizada, http.StatusConflict, "convocatoria_actualizada"},
		{ports.ErrRequisitoNoCumplido, http.StatusUnprocessableEntity, "requisito_no_cumplido"},
		{ports.ErrDatosIncompletos, http.StatusUnprocessableEntity, "datos_incompletos"},
		{ports.ErrConvocatoriaNoDisponible, http.StatusNotFound, "convocatoria_no_disponible"},
		{ports.ErrSolicitudNoEncontrada, http.StatusNotFound, "recurso_no_encontrado"},
		{ports.ErrSinBorrador, http.StatusNotFound, "sin_borrador"},
		{ports.ErrDeclaracionRequerida, http.StatusBadRequest, "declaracion_requerida"},
		{ports.ErrDatosNoValidos, http.StatusBadRequest, "datos_no_validos"},
		{domain.ErrSolicitudInvalida, http.StatusBadRequest, "datos_no_validos"},
	} {
		if errors.Is(err, c.err) {
			Responder(w, c.estado, Error(c.codigo))
			return
		}
	}
	Responder(w, http.StatusInternalServerError, Error("error_interno"))
}

// Instante formatea un instante en UTC.
func Instante(t time.Time) string {
	return t.UTC().Format(FormatoInstante)
}

// Puntos formatea una puntuación como cadena decimal.
func Puntos(p baremacion.Puntos) string {
	return domain.FormatearPuntos(p)
}

// ConvocatoriaResumen es la fila de la lista de convocatorias.
type ConvocatoriaResumen struct {
	ConvocatoriaRef string `json:"convocatoria_ref"`
	Titulo          string `json:"titulo"`
	AbreEn          string `json:"abre_en"`
	CierraEn        string `json:"cierra_en"`
	Abierta         bool   `json:"abierta"`
}

// ResumenConvocatorias traduce las convocatorias publicadas.
func ResumenConvocatorias(c []domain.ConvocatoriaPublicada) any {
	filas := make([]ConvocatoriaResumen, 0, len(c))
	for _, x := range c {
		filas = append(filas, ConvocatoriaResumen{ConvocatoriaRef: x.Ref, Titulo: x.Titulo, AbreEn: Instante(x.AbreEn), CierraEn: Instante(x.CierraEn), Abierta: x.Abierta})
	}
	return struct {
		Convocatorias []ConvocatoriaResumen `json:"convocatorias"`
	}{filas}
}
