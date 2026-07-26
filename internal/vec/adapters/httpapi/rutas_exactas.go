package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
)

const (
	prefijoRutaExactaVEC = "/api/vec/"
	maximoRutaExactaVEC  = 512
)

var ErrRutaExactaInvalida = errors.New(
	"vec http: ruta exacta adicional invalida",
)

var (
	errRutaExactaNoEncontrada = errors.New(
		"vec http: ruta exacta no encontrada",
	)
	ErrAutenticacionRutaExactaRequerida = errors.New(
		"vec http: autenticacion de ruta exacta requerida",
	)
	ErrAccesoRutaExactaDenegado = errors.New(
		"vec http: acceso a ruta exacta denegado",
	)
	ErrAutoridadRutaExactaNoDisponible = errors.New(
		"vec http: autoridad de ruta exacta no disponible",
	)
)

// AutoridadRutasExactas comprueba la capacidad opaca que la frontera
// corporativa incorpora al contexto. Nunca debe deducir autoridad desde
// cabeceras, URL, cuerpo, cookies o cadenas aportadas por el cliente.
type AutoridadRutasExactas interface {
	AutorizarRutaExacta(context.Context, string) error
}

// RutaExacta permite que la raíz de composición incorpore adaptadores de
// módulos sin convertir este dispatcher en una fábrica de infraestructura.
// La ruta completa llega intacta al manejador.
type RutaExacta struct {
	Ruta      string
	Manejador http.Handler
}

func prepararRutasExactas(
	declaradas []RutaExacta,
	autoridad AutoridadRutasExactas,
) (map[string]http.Handler, error) {
	if len(declaradas) == 0 {
		if !dependenciaRutaExactaNula(autoridad) {
			return nil, ErrRutaExactaInvalida
		}
		return nil, nil
	}
	if dependenciaRutaExactaNula(autoridad) {
		return nil, ErrRutaExactaInvalida
	}
	rutas := make(map[string]http.Handler, len(declaradas))
	for _, declarada := range declaradas {
		if !rutaExactaAdicionalValida(declarada.Ruta) ||
			manejadorRutaExactaInvalido(declarada.Manejador) {
			return nil, ErrRutaExactaInvalida
		}
		if _, repetida := rutas[declarada.Ruta]; repetida {
			return nil, ErrRutaExactaInvalida
		}
		rutas[declarada.Ruta] = declarada.Manejador
	}
	return rutas, nil
}

func rutaExactaAdicionalValida(ruta string) bool {
	if len(ruta) <= len(prefijoRutaExactaVEC) ||
		len(ruta) > maximoRutaExactaVEC ||
		!strings.HasPrefix(ruta, prefijoRutaExactaVEC) ||
		strings.HasSuffix(ruta, "/") ||
		rutaColisionaConShellVEC(ruta) {
		return false
	}
	for _, segmento := range strings.Split(
		strings.TrimPrefix(ruta, prefijoRutaExactaVEC),
		"/",
	) {
		if !segmentoRutaExactaValido(segmento) {
			return false
		}
	}
	return true
}

func segmentoRutaExactaValido(segmento string) bool {
	if segmento == "" || len(segmento) > 64 {
		return false
	}
	for _, caracter := range segmento {
		if (caracter < 'a' || caracter > 'z') &&
			(caracter < '0' || caracter > '9') &&
			caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}

func rutaColisionaConShellVEC(ruta string) bool {
	for _, reservada := range rutasBaseVEC() {
		if !strings.Contains(reservada, "{") && ruta == reservada {
			return true
		}
	}
	return strings.HasPrefix(ruta, "/api/vec/personal/rpt/positions/") ||
		strings.HasPrefix(ruta, "/api/vec/personal/categories/") ||
		(strings.HasPrefix(ruta, "/api/vec/modules/") &&
			strings.HasSuffix(ruta, "/action"))
}

func manejadorRutaExactaInvalido(manejador http.Handler) bool {
	if dependenciaRutaExactaNula(manejador) {
		return true
	}
	_, esMux := manejador.(*http.ServeMux)
	return esMux
}

func dependenciaRutaExactaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		if valor.IsNil() {
			return true
		}
	}
	return false
}

func peticionRutaExactaCanonica(peticion *http.Request) bool {
	if peticion == nil || peticion.URL == nil ||
		peticion.URL.RawPath != "" || peticion.URL.Opaque != "" ||
		peticion.URL.Fragment != "" || peticion.URL.RawFragment != "" ||
		peticion.URL.ForceQuery || peticion.URL.Scheme != "" ||
		peticion.URL.Host != "" || peticion.URL.User != nil {
		return false
	}
	escapada := peticion.URL.EscapedPath()
	return escapada == peticion.URL.Path && !strings.Contains(escapada, "%")
}

func responderAutorizacionRutaExacta(
	respuesta http.ResponseWriter,
	err error,
) {
	estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
	switch {
	case errors.Is(err, errRutaExactaNoEncontrada):
		estado, codigo = http.StatusNotFound, "recurso_no_encontrado"
	case errors.Is(err, ErrAutenticacionRutaExactaRequerida):
		estado, codigo = http.StatusUnauthorized, "autenticacion_requerida"
	case errors.Is(err, ErrAccesoRutaExactaDenegado):
		estado, codigo = http.StatusForbidden, "acceso_denegado"
	}
	contenido, _ := json.Marshal(map[string]any{
		"error": map[string]string{
			"codigo":          codigo,
			"clave_i18n":      "api.vec.ruta_exacta.error." + codigo,
			"correlacion_ref": nuevaCorrelacionRutaExacta(),
		},
	})
	for _, cabecera := range []string{
		"Set-Cookie",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Expose-Headers",
		"Content-Encoding",
		"Location",
		"Retry-After",
	} {
		respuesta.Header().Del(cabecera)
	}
	respuesta.Header().Set("Content-Type", "application/json; charset=utf-8")
	respuesta.Header().Set("Cache-Control", "no-store, no-transform")
	respuesta.Header().Set("X-Content-Type-Options", "nosniff")
	respuesta.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; base-uri 'none'; frame-ancestors 'none'",
	)
	respuesta.WriteHeader(estado)
	_, _ = respuesta.Write(contenido)
}

func nuevaCorrelacionRutaExacta() string {
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		return "corr_no_disponible"
	}
	return "corr_" + hex.EncodeToString(aleatorio)
}

func vecRoutes() []string {
	return rutasBaseVEC()
}

func rutasBaseVEC() []string {
	return []string{
		"/api/vec/session",
		"/api/vec/modules",
		"/api/vec/workspace",
		"/api/vec/menu",
		"/api/vec/audit",
		"/api/vec/cronos/timecards",
		"/api/vec/cronos/leave-requests",
		"/api/vec/personal/rpt/positions",
		"/api/vec/personal/rpt/positions/{code}",
		"/api/vec/personal/rpt/imports",
		"/api/vec/personal/rpt/stats",
		"/api/vec/personal/categories",
		"/api/vec/personal/categories/{slug}",
		"/api/vec/personal/catalogs",
		"/api/vec/dietas/road-route",
		"/api/vec/modules/cronos/action",
		"/api/vec/modules/horarios/action",
		"/api/vec/modules/permisos/action",
		"/api/vec/modules/dietas/action",
		"/api/vec/modules/rutas/action",
		"/api/vec/modules/bolsa/action",
		"/api/vec/modules/administracion/action",
		"/api/vec/modules/personal/action",
		"/api/vec/modules/nominas/action",
	}
}
