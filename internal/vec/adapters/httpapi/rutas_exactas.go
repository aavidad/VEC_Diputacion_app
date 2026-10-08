package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	prefijoRutaExactaVEC = "/api/vec/"
	maximoRutaExactaVEC  = 512

	plazoMaximoAuditoriaFronteraRutaExacta = 250 * time.Millisecond
)

var ErrRutaExactaInvalida = errors.New(
	"vec http: ruta exacta adicional invalida",
)

var errAuditoriaFronteraUsuariosNoDisponible = errors.New(
	"vec http: auditoria de frontera de usuarios no disponible",
)

var errAuditoriaFronteraBolsaNoDisponible = errors.New(
	"vec http: auditoria de frontera de bolsa no disponible",
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

type claveActorAuditoriaPreferenciasUsuarios struct{}
type claveActorAuditoriaBolsa struct{}

// ConActorVerificadoAuditoriaPreferenciasUsuarios recibe solo el contexto ya
// resuelto por la autoridad de identidad. Conserva la referencia opaca y no
// transporta datos de la petición ni la instantánea completa al registrador.
func ConActorVerificadoAuditoriaPreferenciasUsuarios(
	ctx context.Context,
	actor domain.ContextoActor,
) (context.Context, error) {
	if ctx == nil || actor.Validar() != nil {
		return nil, domain.ErrContextoActorInvalido
	}
	return context.WithValue(ctx, claveActorAuditoriaPreferenciasUsuarios{}, actor.PersonaRef), nil
}

// ConActorVerificadoAuditoriaBolsa recibe solo el actor resuelto por la
// autoridad de identidad exterior. No acepta una referencia de la peticion.
func ConActorVerificadoAuditoriaBolsa(
	ctx context.Context,
	actor domain.ContextoActor,
) (context.Context, error) {
	if ctx == nil || actor.Validar() != nil {
		return nil, domain.ErrContextoActorInvalido
	}
	return context.WithValue(ctx, claveActorAuditoriaBolsa{}, actor.PersonaRef), nil
}

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

// RutaColeccion declara un único prefijo de recurso para rutas cuyo identificador
// opaco forma parte del path. La autorización recibe siempre la ruta completa;
// el adaptador debe validar el resto del path antes de interpretar ese identificador.
type RutaColeccion struct {
	Prefijo   string
	Manejador http.Handler
}

func prepararRutasExactas(
	declaradas []RutaExacta,
	autoridad AutoridadRutasExactas,
) (map[string]http.Handler, error) {
	if len(declaradas) == 0 {
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

func rutaColeccionValida(prefijo string) bool {
	return rutaExactaAdicionalValida(prefijo)
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
	if ruta == "/api/vec/session/start" {
		return true
	}
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
	superficie := superficieAuditoriaFronteraRutaExacta(peticion.URL.Path)
	if (superficie == ports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases ||
		superficie == ports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo) &&
		(peticion.Method != http.MethodPost || peticion.URL.RawQuery != "") {
		return false
	}
	escapada := peticion.URL.EscapedPath()
	return escapada == peticion.URL.Path && !strings.Contains(escapada, "%")
}

func responderAutorizacionRutaExacta(
	respuesta http.ResponseWriter,
	err error,
) {
	responderAutorizacionRutaExactaConCorrelacion(respuesta, err, nuevaCorrelacionRutaExacta())
}

func responderAutorizacionRutaExactaConCorrelacion(
	respuesta http.ResponseWriter,
	err error,
	correlacion string,
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
			"correlacion_ref": correlacion,
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

func (h *Handler) registrarDenegacionRutaExacta(
	ctx context.Context,
	ruta string,
	err error,
	correlacion string,
) error {
	motivo, registrar := motivoAuditoriaDenegacionRutaExacta(err)
	if !registrar {
		return nil
	}
	superficie := superficieAuditoriaFronteraRutaExacta(ruta)
	// Usuarios y Aspirantes fallan cerrado: una denegación sin anotar es 503.
	// Aspirantes nunca anota persona.
	usuarios := superficie == ports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias ||
		superficie == ports.SuperficieAuditoriaFronteraRutaExactaAspirantes
	bolsa := superficie == ports.SuperficieAuditoriaFronteraRutaExactaBolsaCandidato
	preparacion := superficie == ports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases ||
		superficie == ports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo
	if h == nil || dependenciaRutaExactaNula(h.registradorAuditoriaFronteraRutasExactas) {
		if usuarios {
			return errAuditoriaFronteraUsuariosNoDisponible
		}
		if bolsa {
			return errAuditoriaFronteraBolsaNoDisponible
		}
		if preparacion {
			return ErrAutoridadRutaExactaNoDisponible
		}
		return nil
	}
	orden := ports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: correlacion,
		Motivo:         motivo,
		Superficie:     superficie,
		Ruta:           ruta,
	}
	if superficie == ports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias &&
		motivo == ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
		orden.ActorRef, _ = ctx.Value(claveActorAuditoriaPreferenciasUsuarios{}).(string)
	}
	if bolsa &&
		motivo == ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
		orden.ActorRef, _ = ctx.Value(claveActorAuditoriaBolsa{}).(string)
	}
	if orden.Validar() != nil {
		if usuarios {
			return errAuditoriaFronteraUsuariosNoDisponible
		}
		if bolsa {
			return errAuditoriaFronteraBolsaNoDisponible
		}
		if preparacion {
			return ErrAutoridadRutaExactaNoDisponible
		}
		return nil
	}
	ctxAuditoria, cancelar := context.WithTimeout(
		context.WithoutCancel(ctx), plazoMaximoAuditoriaFronteraRutaExacta,
	)
	defer cancelar()
	if err := h.registradorAuditoriaFronteraRutasExactas.RegistrarAuditoriaFronteraRutaExacta(
		ctxAuditoria,
		orden,
	); err != nil {
		log.Printf(
			"vec http: auditoria_frontera_no_registrada correlacion=%s",
			correlacion,
		)
		if usuarios {
			return errAuditoriaFronteraUsuariosNoDisponible
		}
		if bolsa {
			return errAuditoriaFronteraBolsaNoDisponible
		}
		if preparacion {
			return ErrAutoridadRutaExactaNoDisponible
		}
	}
	return nil
}

func superficieAuditoriaFronteraRutaExacta(ruta string) string {
	switch ruta {
	case "/api/vec/auditoria/opciones", "/api/vec/auditoria/consultas":
		return ports.SuperficieAuditoriaFronteraRutaExactaAuditoria
	case "/api/vec/usuarios/mis-preferencias", "/api/vec/usuarios/area-personal/mis-preferencias",
		"/api/vec/usuarios/mis-correos", "/api/vec/usuarios/area-personal/mis-correos",
		"/api/vec/usuarios/mi-imagen", "/api/vec/usuarios/area-personal/mi-imagen":
		return ports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias
	case "/api/vec/bolsa/mi-bolsa", "/api/vec/bolsa/mi-bolsa/historial",
		"/api/vec/bolsa/mi-bolsa/solicitudes", "/api/vec/bolsa/mi-bolsa/respuestas",
		"/api/vec/bolsa/mi-bolsa/solicitudes-documentales",
		"/api/vec/bolsa/mi-bolsa/disposiciones", "/api/vec/bolsa/mi-bolsa/contacto":
		return ports.SuperficieAuditoriaFronteraRutaExactaBolsaCandidato
	case "/api/vec/aspirantes/area-personal/mi-ficha":
		return ports.SuperficieAuditoriaFronteraRutaExactaAspirantes
	case "/api/vec/seleccion/preparacion-bases/guardar", "/api/vec/seleccion/preparacion-bases/consultar":
		return ports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases
	case "/api/vec/bolsa/reglas-baremo/borradores/alta", "/api/vec/bolsa/reglas-baremo/versiones/consultar",
		"/api/vec/bolsa/reglas-baremo/recibos/recuperar":
		return ports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo
	default:
		return ports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
	}
}

func motivoAuditoriaDenegacionRutaExacta(err error) (ports.MotivoAuditoriaFronteraRutaExacta, bool) {
	switch {
	case errors.Is(err, ErrAutenticacionRutaExactaRequerida):
		return ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, true
	case errors.Is(err, ErrAccesoRutaExactaDenegado):
		return ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, true
	default:
		return "", false
	}
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
		"/api/vec/observabilidad/errores-cliente",
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
		"/api/vec/dietas/route-catalog",
		"/api/vec/modules/cronos/action",
		"/api/vec/modules/horarios/action",
		"/api/vec/modules/permisos/action",
		"/api/vec/modules/bolsa/action",
		"/api/vec/modules/administracion/action",
		"/api/vec/modules/personal/action",
		"/api/vec/modules/nominas/action",
	}
}
