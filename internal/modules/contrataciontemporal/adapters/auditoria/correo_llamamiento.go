// Package auditoria prepara la entrada canónica AuditEntry para el resultado
// durable del correo. No autoriza: la decisión final V3 continúa perteneciendo
// al autorizador y PostgreSQL la vuelve a comprobar en la misma transacción.
package auditoria

import (
	"context"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const ambitoSeudonimizacionAuditoriaResultadoCorreoLlamamiento = "contratacion_temporal.resultado_correo_llamamiento.audit.v1"

var ErrPreparacionAuditoriaResultadoCorreoLlamamiento = errors.New("contratacion temporal: auditoria de resultado de correo no confiable")

// PreparadorResultadoCorreoLlamamiento deriva la identidad solo de la
// capacidad de despacho que ya cruzó la frontera atestada. El seudonimizador
// común es el único que conoce la clave HMAC.
type PreparadorResultadoCorreoLlamamiento struct {
	seudonimizador vecports.SeudonimizadorSujetoAlmacen
	generador      vecports.GeneradorReferenciasAutorizacionV2
	lector         lectorAtestacionDespacho
}

type lectorAtestacionDespacho interface {
	Referencias([]byte) (versionRol, correlacion string, err error)
}

type lectorAtestacionV3 struct{}

func (lectorAtestacionV3) Referencias(contenido []byte) (string, string, error) {
	proyeccion, err := vecdomain.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(contenido)
	if err != nil {
		return "", "", err
	}
	versionRol, errRol := proyeccion.VersionRolRef()
	correlacion, errCorrelacion := proyeccion.CorrelacionRef()
	if errRol != nil || errCorrelacion != nil {
		return "", "", ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	return versionRol, correlacion, nil
}

var _ ports.PreparadorAuditoriaResultadoCorreoLlamamiento = (*PreparadorResultadoCorreoLlamamiento)(nil)

func NuevoPreparadorResultadoCorreoLlamamiento(seudonimizador vecports.SeudonimizadorSujetoAlmacen, generador vecports.GeneradorReferenciasAutorizacionV2) (*PreparadorResultadoCorreoLlamamiento, error) {
	if dependenciaNula(seudonimizador) || dependenciaNula(generador) {
		return nil, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	return nuevoPreparadorResultadoCorreoLlamamiento(seudonimizador, generador, lectorAtestacionV3{})
}

func nuevoPreparadorResultadoCorreoLlamamiento(seudonimizador vecports.SeudonimizadorSujetoAlmacen, generador vecports.GeneradorReferenciasAutorizacionV2, lector lectorAtestacionDespacho) (*PreparadorResultadoCorreoLlamamiento, error) {
	if dependenciaNula(seudonimizador) || dependenciaNula(generador) || dependenciaNula(lector) {
		return nil, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	return &PreparadorResultadoCorreoLlamamiento{seudonimizador: seudonimizador, generador: generador, lector: lector}, nil
}

func (p *PreparadorResultadoCorreoLlamamiento) PrepararAuditoriaResultadoCorreoLlamamiento(ctx context.Context, solicitud ports.SolicitudRegistrarResultadoCorreoLlamamiento, capacidad ports.CapacidadDespachoCorreoLlamamiento, ocurridoEn time.Time) (ports.AuditoriaResultadoCorreoLlamamiento, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil || !ctdomain.InstanteUTCCanonico(ocurridoEn) || dependenciaNula(p.seudonimizador) || dependenciaNula(p.generador) || dependenciaNula(p.lector) {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	material := capacidad.ExportarMaterialParaConsumidor()
	if material.ValidarEstructura() != nil {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	// La proyección no concede autorización. Aquí solo se usan las referencias
	// ya comprometidas para impedir que el AuditEntry copie la correlación del
	// despacho o introduzca una versión de rol declarada por la aplicación.
	versionRol, correlacionDespacho, errProyeccion := p.lector.Referencias(material.PayloadVECAD3())
	actor, errActor := vecdomain.RehidratarContextoActorVinculadoV2(material.ContextoActorCanonico())
	if errProyeccion != nil || errActor != nil || actor.Validar() != nil ||
		actor.Instantanea.PersonaVersion != material.PersonaVersion() ||
		actor.Instantanea.PerfilVersion != material.PerfilVersion() {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	solicitudSeudonimo, err := vecports.NuevaSolicitudSeudonimizarSujetoAlmacen(actor.Principal.ID, ambitoSeudonimizacionAuditoriaResultadoCorreoLlamamiento)
	if err != nil {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	actorID, err := p.seudonimizador.SeudonimizarSujetoAlmacen(ctx, solicitudSeudonimo)
	if err != nil || !seudonimoHMACValido(actorID) {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generador)
	if err != nil {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	correlacionNueva, err := correlacion.ValorCanonico()
	if err != nil || correlacionNueva == correlacionDespacho {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	auditoria, err := ctdomain.NuevaAuditoriaResultadoCorreoLlamamiento(ctdomain.DatosAuditoriaResultadoCorreoLlamamiento{
		ActorID: actorID, ActorProfile: actor.PerfilActivoRef, VersionRolRef: versionRol,
		AuthMethod: string(actor.Principal.AuthMethod), AuthAssurance: string(actor.Principal.AuthAssurance),
		CorrelationRef: correlacionNueva, Solicitud: solicitud, OcurridoEn: ocurridoEn,
	})
	if err != nil {
		return ports.AuditoriaResultadoCorreoLlamamiento{}, ErrPreparacionAuditoriaResultadoCorreoLlamamiento
	}
	return auditoria, nil
}

func seudonimoHMACValido(valor string) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || partes[1] == "" || len(partes[2]) != 64 {
		return false
	}
	for _, caracter := range partes[1] {
		if !(caracter >= 'a' && caracter <= 'z' || caracter >= '0' && caracter <= '9' || caracter == '_' || caracter == '-') {
			return false
		}
	}
	bytes, err := hex.DecodeString(partes[2])
	return err == nil && len(bytes) == 32 && strings.ToLower(partes[2]) == partes[2]
}

func dependenciaNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
