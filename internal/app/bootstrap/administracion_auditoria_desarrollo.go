package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	admin "vec-diputacion-granada/internal/modules/administracion"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	t13 "vec-diputacion-granada/internal/modules/bolsa/application/registroaccesos"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const ambitoSeudonimoAuditoriaAdministracion = "bolsa_registro_accesos_t13"
const maximaVersionAuditoriaAdministracion = uint64(9007199254740991)

var errPreparacionAuditoriaAdministracion = errors.New("bootstrap: auditoria administrativa no disponible")

// Prepara la entrada ligada a la mutación. La autoridad T13 completa el recibo,
// la decisión y el consumo dentro de la transacción; aquí no se persiste nada.
// La composición inyecta un seudonimizador con clave exclusiva de auditoría.
type preparacionAuditoriaAdministracionDesarrollo struct {
	sesion         sesionDurableAdministracionCorreoV3
	seudonimizador vp.SeudonimizadorSujetoAlmacen
	referencias    vp.GeneradorReferenciasAutorizacionV2
	reloj          vp.Reloj
}

var _ adminports.PreparadorAuditoriaConfiguracionCorreo = (*preparacionAuditoriaAdministracionDesarrollo)(nil)

func nuevaPreparacionAuditoriaAdministracionDesarrollo(sesion sesionDurableAdministracionCorreoV3, seudonimizador vp.SeudonimizadorSujetoAlmacen, referencias vp.GeneradorReferenciasAutorizacionV2, reloj vp.Reloj) (*preparacionAuditoriaAdministracionDesarrollo, error) {
	if dependenciaAdministracionNula(sesion) || dependenciaAdministracionNula(seudonimizador) || dependenciaAdministracionNula(referencias) || dependenciaAdministracionNula(reloj) {
		return nil, errPreparacionAuditoriaAdministracion
	}
	return &preparacionAuditoriaAdministracionDesarrollo{sesion: sesion, seudonimizador: seudonimizador, referencias: referencias, reloj: reloj}, nil
}

func (preparacionAuditoriaAdministracionDesarrollo) String() string {
	return "preparacionAuditoriaAdministracion{redactada}"
}
func (preparacionAuditoriaAdministracionDesarrollo) GoString() string {
	return "preparacionAuditoriaAdministracion{redactada}"
}
func (preparacionAuditoriaAdministracionDesarrollo) MarshalJSON() ([]byte, error) {
	return []byte(`{"redactado":true}`), nil
}

func (p *preparacionAuditoriaAdministracionDesarrollo) PrepararAuditoriaConfiguracionCorreo(ctx context.Context, principal core.Principal, versionNueva uint64) (core.AuditEntry, error) {
	fallo := func() (core.AuditEntry, error) { return core.AuditEntry{}, errPreparacionAuditoriaAdministracion }
	if p == nil || ctx == nil || ctx.Err() != nil || dependenciaAdministracionNula(p.sesion) || dependenciaAdministracionNula(p.seudonimizador) || dependenciaAdministracionNula(p.referencias) || dependenciaAdministracionNula(p.reloj) || versionNueva == 0 || versionNueva > maximaVersionAuditoriaAdministracion || principal.Validate() != nil || !principal.HasPermission(admin.PermissionIntegrationsManage) {
		return fallo()
	}
	capacidad, ok := capacidadAdministracionDesdeContexto(ctx)
	if !ok || capacidad.metodo != "PUT" {
		return fallo()
	}
	sesion, err := p.sesion.ResolverSesionAdministracionCorreoV3(ctx)
	if err != nil || !p.sesionValida(ctx, sesion) || principal.ID != sesion.Principal.ID || principal.AuthMethod != sesion.Principal.AuthMethod || principal.AuthAssurance != sesion.Principal.AuthAssurance {
		return fallo()
	}
	roles := append([]string(nil), sesion.Principal.Roles...)
	rolesPrincipal := append([]string(nil), principal.Roles...)
	sort.Strings(roles)
	sort.Strings(rolesPrincipal)
	if !reflect.DeepEqual(roles, rolesPrincipal) {
		return fallo()
	}
	seudonimo, err := p.seudonimoParaSesion(ctx, sesion)
	if err != nil {
		return fallo()
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.referencias)
	if err != nil {
		return fallo()
	}
	referencia, err := correlacion.ValorCanonico()
	if err != nil || !p.sesionValida(ctx, sesion) {
		return fallo()
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !p.sesionValida(ctx, sesion) {
		return fallo()
	}
	a := entradaAuditoriaAdministracion(sesion, seudonimo, roles, versionNueva, referencia, ahora)
	return a, nil
}

// El proveedor V3 llama este método con la misma sesión que entregará al
// emisor. El HMAC se recalcula: ActorID no se acepta por su forma ni por ser
// distinto del identificador nominal, y no se vuelve a resolver otra sesión.
func (p *preparacionAuditoriaAdministracionDesarrollo) ValidarAuditoriaParaSesion(ctx context.Context, auditoria core.AuditEntry, sesion contextoSesionAdministracionCorreoV3, versionNueva uint64) error {
	if p == nil || ctx == nil || ctx.Err() != nil || dependenciaAdministracionNula(p.seudonimizador) || dependenciaAdministracionNula(p.reloj) || versionNueva == 0 || versionNueva > maximaVersionAuditoriaAdministracion || !p.sesionValida(ctx, sesion) || !core.ReferenciaCorrelacionAutorizacionV2Valida(auditoria.CorrelationRef) || auditoria.OccurredAt.IsZero() || auditoria.OccurredAt != auditoria.OccurredAt.UTC().Truncate(time.Microsecond) {
		return errPreparacionAuditoriaAdministracion
	}
	capacidad, ok := capacidadAdministracionDesdeContexto(ctx)
	if !ok || capacidad.metodo != "PUT" || auditoria.OccurredAt.Before(capacidad.certificadoVerificadoEn.UTC().Truncate(time.Microsecond)) || auditoria.OccurredAt.After(p.reloj.Ahora()) {
		return errPreparacionAuditoriaAdministracion
	}
	seudonimo, err := p.seudonimoParaSesion(ctx, sesion)
	if err != nil {
		return errPreparacionAuditoriaAdministracion
	}
	roles := append([]string(nil), sesion.Principal.Roles...)
	sort.Strings(roles)
	esperada := entradaAuditoriaAdministracion(sesion, seudonimo, roles, versionNueva, auditoria.CorrelationRef, auditoria.OccurredAt)
	if !reflect.DeepEqual(auditoria, esperada) || !p.sesionValida(ctx, sesion) {
		return errPreparacionAuditoriaAdministracion
	}
	return nil
}

func (p *preparacionAuditoriaAdministracionDesarrollo) sesionValida(ctx context.Context, sesion contextoSesionAdministracionCorreoV3) bool {
	if p == nil || ctx == nil || ctx.Err() != nil || dependenciaAdministracionNula(p.reloj) {
		return false
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil || !contextoSesionAdministracionCorreoV3Valido(sesion, ahora) || !contextoPermisoAdministracionGobernadoValido(ctx, sesion.Vinculo, sesion.Resultado, ahora) || len(sesion.Principal.Roles) == 0 || len(sesion.Principal.Roles) > 16 {
		return false
	}
	for _, rol := range sesion.Principal.Roles {
		if strings.ContainsAny(rol, "*?") {
			return false
		}
	}
	capacidad, ok := capacidadAdministracionDesdeContexto(ctx)
	return ok && capacidad.metodo == "PUT" && sesion.Principal.ID == sesion.Resultado.Contexto.PersonaRef && sesion.Principal.AuthMethod == sesion.Resultado.Contexto.Principal.AuthMethod && sesion.Principal.AuthAssurance == sesion.Resultado.Contexto.Principal.AuthAssurance
}

func (p *preparacionAuditoriaAdministracionDesarrollo) seudonimoParaSesion(ctx context.Context, sesion contextoSesionAdministracionCorreoV3) (string, error) {
	if !p.sesionValida(ctx, sesion) {
		return "", errPreparacionAuditoriaAdministracion
	}
	solicitud, err := vp.NuevaSolicitudSeudonimizarSujetoAlmacen(sesion.Principal.ID, ambitoSeudonimoAuditoriaAdministracion)
	if err != nil {
		return "", errPreparacionAuditoriaAdministracion
	}
	seudonimo, err := p.seudonimizador.SeudonimizarSujetoAlmacen(ctx, solicitud)
	if err != nil || !t13.ActorSeudonimizadoValido(seudonimo) || !p.sesionValida(ctx, sesion) {
		return "", errPreparacionAuditoriaAdministracion
	}
	return seudonimo, nil
}

func entradaAuditoriaAdministracion(sesion contextoSesionAdministracionCorreoV3, seudonimo string, roles []string, versionNueva uint64, correlacion string, ahora time.Time) core.AuditEntry {
	return core.AuditEntry{ActorID: seudonimo, ActorProfile: sesion.Resultado.Contexto.PerfilActivoRef, ActorRoles: roles, AuthMethod: sesion.Principal.AuthMethod, AuthAssurance: sesion.Principal.AuthAssurance, Purpose: finalidadConfiguracionCorreoAdministracionV3, Action: accionConfiguracionCorreoAdministracionV3, ModuleID: admin.ModuleID, SubjectRef: referenciaConfiguracionCorreoAdministracionV3, ObjectVersion: int(versionNueva), Result: "accepted", CorrelationRef: correlacion, OccurredAt: ahora}
}
