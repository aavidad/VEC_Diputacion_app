package bootstrap

import (
	"context"
	"time"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Las claves sólo enlazan la solicitud ya comprobada con el predicado del
// soporte V3. No contienen una concesión ni datos procedentes de HTTP.
type claveSolicitudDespachoCorreoLlamamientoDesarrollo struct{}
type claveSolicitudResultadoCorreoLlamamientoDesarrollo struct{}

type solicitudResultadoCorreoLlamamientoDesarrollo struct {
	solicitud ports.SolicitudRegistrarResultadoCorreoLlamamiento
	auditoria ports.AuditoriaResultadoCorreoLlamamiento
}

// autorizadorCorreoLlamamientoDesarrollo sólo adapta las dos autoridades V3
// nominales del bootstrap al puerto del caso de uso. La emisión, la
// revalidación del contexto, el PDP, la confirmación, la atestación y la
// revocación siguen perteneciendo a autorizadorLlamamientoDesarrollo y al
// proveedor de material de bootstrap.
type autorizadorCorreoLlamamientoDesarrollo struct {
	despacho  *autorizadorLlamamientoDesarrollo
	resultado *autorizadorLlamamientoDesarrollo
	reloj     ports.Reloj
}

var (
	_ ports.AutorizadorDespachoCorreoLlamamiento  = (*autorizadorCorreoLlamamientoDesarrollo)(nil)
	_ ports.AutorizadorResultadoCorreoLlamamiento = (*autorizadorCorreoLlamamientoDesarrollo)(nil)
)

// nuevosAutorizadoresCorreoLlamamientoDesarrollo recibe exclusivamente las
// autoridades nominales ya compuestas. Sin PDP, soporte, reloj o proveedores
// V3 de audiencia propia no se construye ningún autorizador.
func nuevosAutorizadoresCorreoLlamamientoDesarrollo(
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	reloj ports.Reloj,
) (ports.AutorizadorDespachoCorreoLlamamiento, ports.AutorizadorResultadoCorreoLlamamiento, error) {
	if alta == nil || alta.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(alta.autorizador) ||
		alta.postgresql.proveedorMaterialDespachoCorreo == nil ||
		alta.postgresql.proveedorMaterialResultadoCorreo == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, nil, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	a := &autorizadorCorreoLlamamientoDesarrollo{
		despacho: &autorizadorLlamamientoDesarrollo{
			alta: alta, material: alta.postgresql.proveedorMaterialDespachoCorreo,
			despachoCorreo: true,
		},
		resultado: &autorizadorLlamamientoDesarrollo{
			alta: alta, material: alta.postgresql.proveedorMaterialResultadoCorreo,
			resultadoCorreo: true,
		},
		reloj: reloj,
	}
	return a, a, nil
}

func (a *autorizadorCorreoLlamamientoDesarrollo) AutorizarDespachoCorreoLlamamiento(
	ctx context.Context,
	solicitud ports.SolicitudDespacharCorreoLlamamiento,
) (ports.CapacidadDespachoCorreoLlamamiento, error) {
	vacia := ports.CapacidadDespachoCorreoLlamamiento{}
	if ctx == nil || a == nil || a.despacho == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.reloj) || solicitud.Validar() != nil {
		return vacia, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	recurso, err := ctapplication.RecursoDespachoCorreoLlamamiento(solicitud)
	if err != nil {
		return vacia, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	ctx = context.WithValue(ctx, claveSolicitudDespachoCorreoLlamamientoDesarrollo{}, solicitud)
	material, err := a.despacho.AutorizarOperacion(ctx, ctapplication.AccionDespacharCorreoLlamamiento, recurso)
	if err := errorAutorizacionCorreoLlamamientoDesarrollo(ctx, err); err != nil {
		return vacia, err
	}
	capacidad, err := ctapplication.NuevaCapacidadDespachoCorreoLlamamiento(solicitud, material, a.reloj.Ahora())
	if err != nil {
		return vacia, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	return capacidad, nil
}

func (a *autorizadorCorreoLlamamientoDesarrollo) AutorizarResultadoCorreoLlamamiento(
	ctx context.Context,
	solicitud ports.SolicitudRegistrarResultadoCorreoLlamamiento,
	auditoria ports.AuditoriaResultadoCorreoLlamamiento,
) (ports.CapacidadResultadoCorreoLlamamiento, error) {
	vacia := ports.CapacidadResultadoCorreoLlamamiento{}
	if ctx == nil || a == nil || a.resultado == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.reloj) ||
		solicitud.Validar() != nil {
		return vacia, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	if _, err := auditoria.HuellaSHA256(); err != nil {
		return vacia, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	recurso, err := ctapplication.RecursoResultadoCorreoLlamamiento(solicitud, auditoria)
	if err != nil {
		return vacia, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	ctx = context.WithValue(ctx, claveSolicitudResultadoCorreoLlamamientoDesarrollo{}, solicitudResultadoCorreoLlamamientoDesarrollo{
		solicitud: solicitud, auditoria: auditoria,
	})
	material, err := a.resultado.AutorizarOperacion(ctx, ctapplication.AccionRegistrarResultadoCorreoLlamamiento, recurso)
	if err := errorAutorizacionCorreoLlamamientoDesarrollo(ctx, err); err != nil {
		return vacia, err
	}
	capacidad, err := ctapplication.NuevaCapacidadResultadoCorreoLlamamiento(solicitud, auditoria, material, a.reloj.Ahora())
	if err != nil {
		return vacia, ports.ErrDespachoCorreoLlamamientoDenegado
	}
	return capacidad, nil
}

func errorAutorizacionCorreoLlamamientoDesarrollo(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if causa == nil {
		return nil
	}
	return ports.ErrDespachoCorreoLlamamientoDenegado
}

func accionCorreoLlamamientoDesarrollo(accion string) bool {
	return accion == ctapplication.AccionDespacharCorreoLlamamiento || accion == ctapplication.AccionRegistrarResultadoCorreoLlamamiento
}

// Un rol nominal nuevo contiene dos concesiones exactas. Su versión es común
// al despacho y al resultado; no se amplía el rol histórico de comunicación.
func nuevaInstantaneaCorreoLlamamientoDesarrollo(principal, perfil string, ahora time.Time) (dominiovec.InstantaneaAutorizacion, error) {
	concesion := func(accion, tipo string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{Accion: accion, ModuloID: ports.ModuloContratacion, TipoRecurso: tipo,
			Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh}
	}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principal, perfil, ahora,
		"correo_llamamiento_ct_desarrollo", "Correo de llamamiento de desarrollo", "correo-llamamiento-ct-desarrollo",
		[]dominiovec.ConcesionRol{
			concesion(ctapplication.AccionDespacharCorreoLlamamiento, ctapplication.TipoRecursoDespachoCorreoLlamamiento),
			concesion(ctapplication.AccionRegistrarResultadoCorreoLlamamiento, ctapplication.TipoRecursoResultadoCorreoLlamamiento),
		}, []dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}
