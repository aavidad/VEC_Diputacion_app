// Package composicion conecta las autoridades comunes ya compuestas
// (identidad, ContextoActor con alcance {empleado} y emisor V3) con los
// puertos de Cronos. No crea identidades, concesiones ni conexiones.
package composicion

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionCronosNoDisponible = errors.New("cronos: composición de autoridades no disponible")

// IdentidadRegistradaCronos procede exclusivamente de la frontera de sesión:
// resultado de ContextoActor registrado con alcance {empleado} y el vínculo de
// autenticación revalidado de la misma petición.
type IdentidadRegistradaCronos struct {
	Contexto vecdomain.ResultadoContextoActorRegistradoV2
	Vinculo  vecdomain.VinculoAutenticacionActorV2
}

type ResolutorIdentidadRegistradaCronos interface {
	ResolverIdentidadRegistradaCronos(context.Context) (IdentidadRegistradaCronos, error)
}

// EmisorMaterialCronosV3 es la autoridad común V3 compuesta con PDP,
// firmante y verificador. Cronos sólo recibe el material exportable nominal.
type EmisorMaterialCronosV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// MotivosCronos fija la entrada del catálogo de motivos de cada acción.
// Los cuatro de la resolución de permisos y los avisos (AD3-57) van juntos:
// o están todos, y la capacidad se compone, o no está ninguno.
type MotivosCronos struct {
	Saldo, Marcaje, Disponibilidad, Recuperacion vecdomain.ReferenciaEntradaCatalogo
	Movimientos, Correccion, Permisos, Permiso   vecdomain.ReferenciaEntradaCatalogo
	Bandeja, Resolucion, Avisos, ArchivoAviso    vecdomain.ReferenciaEntradaCatalogo
}

func (m MotivosCronos) validar() error {
	for _, r := range []vecdomain.ReferenciaEntradaCatalogo{m.Saldo, m.Marcaje, m.Disponibilidad, m.Recuperacion, m.Movimientos, m.Correccion, m.Permisos, m.Permiso} {
		if !vecdomain.ReferenciaMotivoAutorizacionV2Valida(r) {
			return ErrComposicionCronosNoDisponible
		}
	}
	if _, err := m.resolucionConfigurada(); err != nil {
		return err
	}
	return nil
}

// resolucionConfigurada distingue la capacidad apagada (ningún motivo) de
// una configuración incompleta, que impide componer.
func (m MotivosCronos) resolucionConfigurada() (bool, error) {
	cero := vecdomain.ReferenciaEntradaCatalogo{}
	validos, vacios := 0, 0
	for _, r := range []vecdomain.ReferenciaEntradaCatalogo{m.Bandeja, m.Resolucion, m.Avisos, m.ArchivoAviso} {
		switch {
		case r == cero:
			vacios++
		case vecdomain.ReferenciaMotivoAutorizacionV2Valida(r):
			validos++
		}
	}
	switch {
	case validos == 4:
		return true, nil
	case vacios == 4:
		return false, nil
	}
	return false, ErrComposicionCronosNoDisponible
}

// ResolucionConfigurada indica si el autorizador puede emitir las decisiones
// de la resolución de permisos y de los avisos.
func (a *AutorizadorCronos) ResolucionConfigurada() bool {
	if a == nil {
		return false
	}
	ok, err := a.motivos.resolucionConfigurada()
	return ok && err == nil
}

// AutorizadorCronos emite una decisión V3 nueva por acción y recurso exacto.
type AutorizadorCronos struct {
	emisor  EmisorMaterialCronosV3
	motivos MotivosCronos
}

func NuevoAutorizadorCronos(emisor EmisorMaterialCronosV3, motivos MotivosCronos) (*AutorizadorCronos, error) {
	if nulo(emisor) || motivos.validar() != nil {
		return nil, ErrComposicionCronosNoDisponible
	}
	return &AutorizadorCronos{emisor: emisor, motivos: motivos}, nil
}

type contratoV3 struct {
	accion, finalidad, audiencia string
	motivo                       vecdomain.ReferenciaEntradaCatalogo
}

func (a *AutorizadorCronos) emitir(ctx context.Context, id IdentidadRegistradaCronos, c contratoV3, recurso vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if a == nil || nulo(a.emisor) || ctx == nil || ctx.Err() != nil || id.Contexto.Validar() != nil || id.Vinculo.ValidarPara(id.Contexto) != nil || recurso.Validar() != nil {
		return vacio, ports.ErrDependenciaNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, ports.ErrDependenciaNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: id.Vinculo, ReferenciaMotivo: c.motivo,
		Accion: c.accion, Recurso: recurso, Finalidad: c.finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return vacio, ports.ErrDependenciaNoDisponible
	}
	decision, confirmacion, exportador, err := a.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, id.Contexto)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacio, vecdomain.ErrPermissionDenied
		}
		return vacio, ports.ErrDependenciaNoDisponible
	}
	if decision.ValidarPara(solicitud) != nil || nulo(exportador) {
		return vacio, ports.ErrDependenciaNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, id.Contexto, c.motivo, material, c.audiencia) ||
		material.PersonaVersion() != id.Contexto.Contexto.Instantanea.PersonaVersion || material.PerfilVersion() != id.Contexto.Contexto.Instantanea.PerfilVersion {
		return vacio, ports.ErrDependenciaNoDisponible
	}
	return material, nil
}

// empleadoUnico exige exactamente un empleado canónico en el contexto
// registrado. ContextoActor con alcance {empleado} ya deniega ausencia o
// ambigüedad; esta comprobación conserva el motivo si llegara otro contexto.
func empleadoUnico(id IdentidadRegistradaCronos) (string, error) {
	if id.Contexto.Validar() != nil {
		return "", ports.ErrDependenciaNoDisponible
	}
	empleados, err := id.Contexto.Contexto.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	switch {
	case err != nil:
		return "", ports.ErrDependenciaNoDisponible
	case len(empleados) == 0:
		return "", ports.ErrEmpleadoNoAcreditado
	case len(empleados) > 1:
		return "", ports.ErrEmpleadoAmbiguo
	}
	return empleados[0], nil
}

// mismaPersona comprueba que el material del servidor pertenece a la
// identidad de la petición en curso: persona, perfil y empleado.
func mismaPersona(id IdentidadRegistradaCronos, actor, perfil, empleado string) bool {
	unico, err := empleadoUnico(id)
	return err == nil && unico == empleado && id.Contexto.Contexto.PersonaRef == actor && id.Contexto.Contexto.PerfilActivoRef == perfil
}

// proveedorPeticion emite las decisiones de una única petición autenticada.
type proveedorPeticion struct {
	autorizador *AutorizadorCronos
	identidad   IdentidadRegistradaCronos
}

func (p proveedorPeticion) ProveerMaterialMarcajePropio(ctx context.Context, m domain.MaterialAutorizacionMarcajePropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoMarcajePropio(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionRegistrarMarcajePropio, application.FinalidadRegistrarMarcajePropio, application.AudienciaMarcajePropio, p.autorizador.motivos.Marcaje}, recurso)
}

func (p proveedorPeticion) ProveerMaterialRecuperacionMarcajeRemoto(ctx context.Context, m domain.MaterialRecuperacionMarcajeRemoto) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoRecuperacionMarcajeRemoto(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionRecuperarMarcajeRemoto, application.FinalidadRecuperarMarcajeRemoto, application.AudienciaRecuperacionMarcajeRemoto, p.autorizador.motivos.Recuperacion}, recurso)
}

func (p proveedorPeticion) ProveerMaterialConsultaSaldoPropio(ctx context.Context, m domain.MaterialConsultaSaldoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaSaldoPropio(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionConsultarSaldoPropio, application.FinalidadConsultarSaldoPropio, application.AudienciaConsultaSaldoPropio, p.autorizador.motivos.Saldo}, recurso)
}

// ProveedorDisponibilidadRemota resuelve la identidad de la petición en
// curso desde el contexto y sólo emite para esa misma persona y empleado.
type ProveedorDisponibilidadRemota struct {
	autorizador *AutorizadorCronos
	identidad   ResolutorIdentidadRegistradaCronos
}

func NuevoProveedorDisponibilidadRemota(a *AutorizadorCronos, identidad ResolutorIdentidadRegistradaCronos) (*ProveedorDisponibilidadRemota, error) {
	if a == nil || nulo(identidad) {
		return nil, ErrComposicionCronosNoDisponible
	}
	return &ProveedorDisponibilidadRemota{autorizador: a, identidad: identidad}, nil
}

func (p *ProveedorDisponibilidadRemota) ProveerMaterialDisponibilidadMarcajeRemoto(ctx context.Context, m domain.MaterialDisponibilidadMarcajeRemoto) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p == nil || nulo(p.identidad) || ctx == nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	id, err := p.identidad.ResolverIdentidadRegistradaCronos(ctx)
	if err != nil || !mismaPersona(id, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoDisponibilidadMarcajeRemoto(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, id, contratoV3{application.AccionConsultarDisponibilidadRemota, application.FinalidadConsultarDisponibilidadRemota, application.AudienciaDisponibilidadMarcajeRemoto, p.autorizador.motivos.Disponibilidad}, recurso)
}

var (
	_ ports.ProveedorMaterialMarcajePropio               = proveedorPeticion{}
	_ ports.ProveedorMaterialRecuperacionMarcajeRemoto   = proveedorPeticion{}
	_ ports.ProveedorMaterialConsultaSaldoPropio         = proveedorPeticion{}
	_ ports.ProveedorMaterialDisponibilidadMarcajeRemoto = (*ProveedorDisponibilidadRemota)(nil)
)

func nulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	default:
		return false
	}
}
