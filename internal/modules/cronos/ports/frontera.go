package ports

import (
	"context"
	"errors"
	"regexp"
)

var ErrDenegacionFronteraNoRegistrada = errors.New("cronos denegacion de frontera no registrada")

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

// OrdenDenegacionFronteraCronos. Ruta es una de las cuatro publicadas u
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
		"/api/interna/cronos/marcajes/remoto/disponibilidad", "/api/interna/cronos/marcajes/remoto/recibo", "otra":
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
