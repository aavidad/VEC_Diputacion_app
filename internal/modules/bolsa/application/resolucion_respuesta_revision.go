package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ServicioResolucionRespuestaRevision no lee B30 para decidir autoridad. La
// consulta y el efecto entran al repositorio ya ligados a material V3 exacto.
type ServicioResolucionRespuestaRevision struct {
	autorizador ports.AutorizadorBorradorLlamamientoV3
	registro    ports.RegistroResolucionRespuestaRevision
	lector      ports.LectorRespuestasRevision
}

func NuevoServicioResolucionRespuestaRevision(a ports.AutorizadorBorradorLlamamientoV3,
	r ports.RegistroResolucionRespuestaRevision, l ports.LectorRespuestasRevision,
) (*ServicioResolucionRespuestaRevision, error) {
	if nulo(a) || nulo(r) || nulo(l) {
		return nil, ports.ErrResolucionRespuestaRevisionNoDisponible
	}
	return &ServicioResolucionRespuestaRevision{autorizador: a, registro: r, lector: l}, nil
}

func (s *ServicioResolucionRespuestaRevision) Resolver(ctx context.Context,
	q ports.SolicitudResolverRespuestaRevision,
) (ports.ReciboResolucionRespuestaRevision, error) {
	vacio := ports.ReciboResolucionRespuestaRevision{}
	if s == nil || ctx == nil || ctx.Err() != nil || q.Validar() != nil {
		return vacio, ports.ErrResolucionRespuestaRevisionInvalida
	}
	canon, semantica, err := ports.CanonResolucionRespuestaRevision(q)
	if err != nil {
		return vacio, err
	}
	recurso := core.RecursoAutorizable{Referencia: q.RespuestaRef, ModuloID: "bolsa",
		Tipo:    ports.TipoRecursoRespuestaRevision,
		Ambitos: map[string]string{"unidad_ref": q.UnidadRef, "ambito_ref": q.AmbitoRef},
		Atributos: map[string]string{"material_sha256": ports.HuellaCanonRespuestasRevision(canon),
			"bolsa_ref": q.BolsaRef}}
	material, err := s.emitir(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion,
		q.MotivoAutorizacion, ports.AccionResolverRespuestaRevision,
		ports.FinalidadRespuestaRevision, ports.AudienciaResolverRespuestaRevision, recurso)
	if err != nil {
		return vacio, err
	}
	orden := ports.OrdenResolucionRespuestaRevision{Solicitud: q, Canon: canon,
		HuellaSemanticaSHA256: semantica, Material: material}
	recibo, err := s.registro.ResolverRespuestaRevision(ctx, orden)
	if err != nil {
		return vacio, errorResolucionRespuestaRevision(err)
	}
	if recibo.ValidarPara(orden) != nil {
		return vacio, ports.ErrResolucionRespuestaRevisionNoDisponible
	}
	return recibo, nil
}

func (s *ServicioResolucionRespuestaRevision) Consultar(ctx context.Context,
	q ports.SolicitudConsultarRespuestasRevision,
) (ports.PaginaRespuestasRevision, error) {
	vacia := ports.PaginaRespuestasRevision{}
	if s == nil || ctx == nil || ctx.Err() != nil || q.Validar() != nil {
		return vacia, ports.ErrResolucionRespuestaRevisionInvalida
	}
	canon, err := ports.CanonConsultaRespuestasRevision(q)
	if err != nil {
		return vacia, err
	}
	recurso := core.RecursoAutorizable{Referencia: q.BolsaRef, ModuloID: "bolsa",
		Tipo:      ports.TipoRecursoBandejaRevision,
		Ambitos:   map[string]string{"unidad_ref": q.UnidadRef, "ambito_ref": q.AmbitoRef},
		Atributos: map[string]string{"material_sha256": ports.HuellaCanonRespuestasRevision(canon)}}
	material, err := s.emitir(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion,
		q.MotivoAutorizacion, ports.AccionConsultarRespuestasRevision,
		ports.FinalidadRespuestaRevision, ports.AudienciaConsultarRespuestasRevision, recurso)
	if err != nil {
		return vacia, err
	}
	orden := ports.OrdenConsultaRespuestasRevision{Solicitud: q, Canon: canon, Material: material}
	pagina, err := s.lector.ConsultarRespuestasRevision(ctx, orden)
	if err != nil {
		return vacia, errorResolucionRespuestaRevision(err)
	}
	if pagina.ValidarPara(orden) != nil {
		return vacia, ports.ErrResolucionRespuestaRevisionNoDisponible
	}
	return pagina, nil
}

func (s *ServicioResolucionRespuestaRevision) emitir(ctx context.Context,
	vinculo core.VinculoAutenticacionActorV2, resultado core.ResultadoContextoActorRegistradoV2,
	correlacion core.ReferenciaCorrelacionAutorizacionV2, motivo core.ReferenciaEntradaCatalogo,
	accion, finalidad, audiencia string, recurso core.RecursoAutorizable,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo, Accion: accion,
		Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion})
	if err != nil {
		return vacio, core.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		return vacio, errorResolucionRespuestaRevision(err)
	}
	if nulo(exportador) {
		return vacio, ports.ErrResolucionRespuestaRevisionNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(solicitud, decision,
		confirmacion, resultado, motivo, material, audiencia) {
		return vacio, ports.ErrResolucionRespuestaRevisionNoDisponible
	}
	return material, nil
}

func errorResolucionRespuestaRevision(err error) error {
	if errors.Is(err, core.ErrAutorizacionDenegada) || errors.Is(err, core.ErrPermissionDenied) ||
		errors.Is(err, ports.ErrResolucionRespuestaRevisionConflicto) ||
		errors.Is(err, ports.ErrResolucionRespuestaRevisionInvalida) {
		return err
	}
	return ports.ErrResolucionRespuestaRevisionNoDisponible
}
