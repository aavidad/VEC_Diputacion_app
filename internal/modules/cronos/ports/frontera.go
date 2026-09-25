package ports

import (
	"context"
	"errors"
	"regexp"
)

var ErrDenegacionFronteraNoRegistrada = errors.New("cronos denegacion de frontera no registrada")

// La persona no tiene exactamente un empleado canónico vigente. Deniega con
// motivo; nunca se elige uno ni se deduce de la cuenta o del certificado.
var (
	ErrEmpleadoNoAcreditado = errors.New("cronos persona sin empleado canonico")
	ErrEmpleadoAmbiguo      = errors.New("cronos persona con empleado ambiguo")
)

// Motivos cerrados de una denegación antes de alcanzar el caso de uso. No
// transportan identidades, certificados ni datos de la petición.
const (
	MotivoFronteraAutenticacion   = "autenticacion_requerida"
	MotivoFronteraAccesoDenegado  = "acceso_denegado"
	MotivoFronteraSinEmpleado     = "sin_empleado"
	MotivoFronteraEmpleadoAmbiguo = "empleado_ambiguo"
	MotivoFronteraDependencia     = "dependencia"
)

var (
	correlacionFronteraCronos = regexp.MustCompile(`^corr_([0-9a-f]{32}|no_disponible)$`)
	actorFronteraCronos       = regexp.MustCompile(`^per_[-A-Za-z0-9_]{22,128}$`)
)

// OrdenDenegacionFronteraCronos. Ruta es una de las publicadas u
// "otra"; Metodo es GET, POST u "otro". ActorRef sólo si ya fue acreditado.
type OrdenDenegacionFronteraCronos struct {
	CorrelacionRef, Motivo, Ruta, Metodo, ActorRef string
}

func (o OrdenDenegacionFronteraCronos) Validar() error {
	switch o.Motivo {
	case MotivoFronteraAutenticacion, MotivoFronteraAccesoDenegado, MotivoFronteraSinEmpleado, MotivoFronteraEmpleadoAmbiguo, MotivoFronteraDependencia:
	default:
		return ErrDenegacionFronteraNoRegistrada
	}
	switch o.Ruta {
	case "/api/interna/cronos/saldos/propio", "/api/interna/cronos/marcajes/remoto",
		"/api/interna/cronos/marcajes/remoto/disponibilidad", "/api/interna/cronos/marcajes/remoto/recibo",
		"/api/interna/cronos/movimientos/propio", "/api/interna/cronos/correcciones/propias",
		"/api/interna/cronos/permisos/propio", "/api/interna/cronos/permisos/solicitudes",
		// Resolución y avisos (000009) y notificaciones (000010): sin ellas una
		// denegación en esas rutas no se podría auditar y respondería 503.
		"/api/interna/cronos/permisos/bandeja", "/api/interna/cronos/permisos/resoluciones",
		"/api/interna/cronos/avisos/propio", "/api/interna/cronos/avisos/archivos",
		"/api/interna/cronos/notificaciones/propio", "/api/interna/cronos/notificaciones/envios",
		"/api/interna/cronos/notificaciones/bandeja", "/api/interna/cronos/notificaciones/atenciones", "otra":
	default:
		return ErrDenegacionFronteraNoRegistrada
	}
	if (o.Metodo != "GET" && o.Metodo != "POST" && o.Metodo != "otro") || !correlacionFronteraCronos.MatchString(o.CorrelacionRef) ||
		(o.ActorRef != "" && !actorFronteraCronos.MatchString(o.ActorRef)) {
		return ErrDenegacionFronteraNoRegistrada
	}
	return nil
}

// RegistroDenegacionFronteraCronos confirma sólo tras el COMMIT del auditor.
type RegistroDenegacionFronteraCronos interface {
	RegistrarDenegacionFronteraCronos(context.Context, OrdenDenegacionFronteraCronos) error
}
