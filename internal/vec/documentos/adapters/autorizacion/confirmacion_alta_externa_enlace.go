package autorizacion

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// IdentidadConfirmacionAltaExternaEnlace resuelve para esta petición el
// contexto registrado y su vínculo autenticado, incluido el perfil nominal
// de Documentos. La composición no debe reconstruirlos desde el documento ni
// desde el DTO Cronos.
type IdentidadConfirmacionAltaExternaEnlace interface {
	ResolverIdentidadConfirmacionAltaExternaEnlace(context.Context) (vecdomain.ResultadoContextoActorRegistradoV2, vecdomain.VinculoAutenticacionActorV2, error)
}

type EmisorConfirmacionAltaExternaEnlace interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type ProveedorConfirmacionAltaExternaEnlaceV3 struct {
	identidad IdentidadConfirmacionAltaExternaEnlace
	emisor    EmisorConfirmacionAltaExternaEnlace
	motivo    vecdomain.ReferenciaEntradaCatalogo
	reloj     vecports.Reloj
}

func dependenciaEnlaceNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return r.IsNil()
	}
	return false
}

func NuevoProveedorConfirmacionAltaExternaEnlaceV3(
	identidad IdentidadConfirmacionAltaExternaEnlace,
	emisor EmisorConfirmacionAltaExternaEnlace,
	motivo vecdomain.ReferenciaEntradaCatalogo,
	reloj vecports.Reloj,
) (*ProveedorConfirmacionAltaExternaEnlaceV3, error) {
	if dependenciaEnlaceNula(identidad) || dependenciaEnlaceNula(emisor) || dependenciaEnlaceNula(reloj) || motivo.Validar() != nil {
		return nil, docports.ErrCapacidadNoDisponible
	}
	return &ProveedorConfirmacionAltaExternaEnlaceV3{identidad: identidad, emisor: emisor, motivo: motivo, reloj: reloj}, nil
}

var _ docports.ProveedorConfirmacionAltaExternaEnlace = (*ProveedorConfirmacionAltaExternaEnlaceV3)(nil)

func (p *ProveedorConfirmacionAltaExternaEnlaceV3) AutorizarConfirmacionAltaExternaEnlace(
	ctx context.Context,
	s docports.SolicitudConfirmacionAltaExternaEnlace,
) (docports.AutorizacionConfirmacionAltaExternaEnlace, error) {
	vacia := docports.AutorizacionConfirmacionAltaExternaEnlace{}
	if p == nil || dependenciaEnlaceNula(p.identidad) || dependenciaEnlaceNula(p.emisor) ||
		dependenciaEnlaceNula(p.reloj) || ctx == nil {
		return vacia, docports.ErrCapacidadNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	recurso, err := s.RecursoV3()
	if err != nil || recurso.Validar() != nil {
		return vacia, docports.ErrSolicitudInvalida
	}
	resultado, vinculo, err := p.identidad.ResolverIdentidadConfirmacionAltaExternaEnlace(ctx)
	if err != nil || resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil {
		return vacia, docports.ErrAccesoDenegado
	}
	datos, err := vinculo.Datos()
	if err != nil {
		return vacia, docports.ErrAccesoDenegado
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, docports.ErrCapacidadNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, docports.ErrCapacidadNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: p.motivo,
		Accion: docports.AccionConfirmarAltaExternaEnlace, Recurso: recurso,
		Finalidad: docports.FinalidadConfirmarAltaExternaEnlace, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, docports.ErrAccesoDenegado
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) ||
			errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
			return vacia, docports.ErrAccesoDenegado
		}
		return vacia, docports.ErrCapacidadNoDisponible
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaEnlaceNula(exportador) {
		return vacia, docports.ErrAccesoDenegado
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion,
		resultado, p.motivo, material, docports.AudienciaConfirmarAltaExternaEnlace) {
		return vacia, docports.ErrAccesoDenegado
	}
	autorizacion := docports.AutorizacionConfirmacionAltaExternaEnlace{
		Material: material, PrincipalID: datos.PrincipalID,
		PerfilActivoRef: datos.PerfilActivoRef, CorrelacionRef: correlacionRef,
	}
	if autorizacion.Validar(s, p.reloj.Ahora()) != nil {
		return vacia, docports.ErrAccesoDenegado
	}
	return autorizacion, nil
}
