package firmaemisorv2

import (
	"context"
	"slices"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.AutorizadorRecuperacionFirmasV2 = (*Emisor)(nil)

// La fuente nominal común se revalida en cada invocación por e.contexto.
// El candidato de la solicitud no se convierte en actor ni en perfil.
func (e *Emisor) AutorizarRecuperacionFirmasV2(ctx context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadRecuperacionFirmasV2, error) {
	var cero ports.CapacidadRecuperacionFirmasV2
	if ctx == nil || e == nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if err := ctx.Err(); err != nil {
		return cero, opaco(ctx, err)
	}
	canon, err := m.Canonico()
	clear(canon)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	recurso, err := firma.RecursoConsultaFirmasR5V2(m)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	base, err := e.contexto(ctx)
	if err != nil {
		return cero, err
	}
	correlacion, err := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: base.Vinculo, ReferenciaMotivo: e.motivo,
		Accion: ports.AccionRecuperarFirmasR5V2, Recurso: recurso,
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
	if decision.ValidarPara(solicitud) != nil || nulo(exportador) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	restricciones, err := decision.RestriccionesProyeccionPara(solicitud)
	if err != nil || !slices.Equal(restricciones.CamposPermitidos, ports.CamposRecuperacionFirmasV2()) ||
		len(restricciones.Obligaciones) != 0 {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || ctx.Err() != nil {
		return cero, opaco(ctx, err)
	}
	if !vp.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, base.Resultado, e.motivo, material,
		ports.AudienciaRecuperacionFirmasR5V2) || !base.Vinculo.VigenteEn(e.reloj.Ahora(), base.Resultado) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	ahora, resumen := e.reloj.Ahora(), material.ResumenCapacidad()
	if ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	capacidad := ports.TransportarMaterialRecuperacionFirmasV2(material,
		restricciones.CamposPermitidos, restricciones.Obligaciones)
	if firma.ValidarCapacidadRecuperacionFirmasV2(capacidad, m) != nil || ctx.Err() != nil {
		return cero, opaco(ctx, ports.ErrFirmaDocumentoDenegada)
	}
	return capacidad, nil
}
