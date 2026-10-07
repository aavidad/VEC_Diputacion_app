package firmaemisorv2

import (
	"context"
	"maps"
	"slices"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// AutorizarConsultaFirmasR5V2 conserva al operador de la sesión revalidada.
// El candidato es un parámetro del indicador; nunca sustituye al actor.
// CT172 consume la capacidad y audita la lectura antes de proyectar sus datos.
func (e *Emisor) AutorizarConsultaFirmasR5V2(ctx context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadConsultaFirmasR5V2, error) {
	var cero ports.CapacidadConsultaFirmasR5V2
	if e == nil || ctx == nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if ctx.Err() != nil {
		return cero, opaco(ctx, nil)
	}
	recurso, err := firma.RecursoConsultaFirmasR5V2(m)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	base, err := e.contexto(ctx)
	if err != nil {
		return cero, err
	}
	if err := e.cotejarAmbitosConsulta(ctx, base, recurso); err != nil {
		return cero, err
	}
	correlacion, err := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: base.Vinculo, ReferenciaMotivo: e.motivo,
		Accion: ports.AccionConsultarFirmasR5V2, Recurso: recurso,
		Finalidad: ports.FinalidadFirmaDocumento, Correlacion: correlacion,
	})
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	resultado, err := base.Resultado.Clonar()
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	decision, confirmacion, exportador, err := e.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	if nulo(exportador) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	restricciones, err := decision.RestriccionesProyeccionPara(solicitud)
	// AD162 exige esta lista canónica completa y ninguna obligación adicional.
	if err != nil || !slices.Equal(restricciones.CamposPermitidos, ports.CamposConsultaFirmasR5V2()) || len(restricciones.Obligaciones) != 0 {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	if !vp.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, base.Resultado, e.motivo, material, ports.AudienciaConsultaFirmasR5V2) ||
		!base.Vinculo.VigenteEn(e.reloj.Ahora(), base.Resultado) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	ahora, resumen := e.reloj.Ahora(), material.ResumenCapacidad()
	if ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	capacidad := ports.TransportarMaterialConsultaFirmasR5V2(material)
	if firma.ValidarCapacidadConsultaFirmasR5V2(capacidad, m) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if ctx.Err() != nil {
		return cero, opaco(ctx, nil)
	}
	return capacidad, nil
}

var _ ports.AutorizadorConsultaFirmasR5V2 = (*Emisor)(nil)

// cotejarAmbitosConsulta: con la fuente de ámbitos compuesta, el recurso de la
// consulta o la recuperación lleva exactamente los de la asignación vigente
// (organización y, si la tiene, la unidad, que va en UnidadRef). Sin ella
// decide el PDP, que exige lo mismo; AD210 los relee en el consumo.
func (e *Emisor) cotejarAmbitosConsulta(ctx context.Context, base ContextoActorFirmaV2, r vd.RecursoAutorizable) error {
	if nulo(e.autorizacion) {
		return nil
	}
	a, err := e.ambitosAsignacion(ctx, base)
	if err != nil {
		return err
	}
	if !maps.Equal(r.Ambitos, a.Mapa()) {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
