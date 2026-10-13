package bootstrap

import (
	"context"

	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// proveedorRetiradaCartera liga el material del consumidor externo únicamente
// a la retirada del hecho propio. La criptografía y el inventario V3 son los
// ya publicados para el portal; este adaptador no crea claves ni permisos.
type proveedorRetiradaCartera struct {
	delegado *proveedorMaterialAltaContratacionTemporalDesarrollo
	motivo   vec.ReferenciaEntradaCatalogo
}

var _ meritosports.ProveedorMaterialRetiradaCartera = proveedorRetiradaCartera{}

func (p proveedorRetiradaCartera) EmitirMaterialRetiradaCartera(ctx context.Context,
	solicitud vec.SolicitudAutorizacionLigadaV3,
	resultado vec.ResultadoContextoActorRegistradoV2,
	decision vec.DecisionAutorizacionLigadaV3,
	confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) (vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if ctx == nil || p.delegado == nil || p.motivo.Validar() != nil ||
		p.motivo.CatalogoID != "motivos_autorizacion_meritos_cartera" ||
		resultado.Validar() != nil {
		return nil, meritosports.ErrCarteraNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil || datos.Accion != meritosports.AccionRetirarCartera ||
		datos.ReferenciaMotivo != p.motivo || datos.Finalidad != meritosports.FinalidadCartera ||
		datos.Recurso.ModuloID != meritosports.ModuloCartera ||
		datos.Recurso.Tipo != "hecho" || datos.Recurso.Referencia == "" ||
		datos.Recurso.Ambitos["persona_ref"] != resultado.Contexto.PersonaRef ||
		datos.VinculoAutenticacionActor.ValidarPara(resultado) != nil ||
		decision.ExigirProyeccionPara(solicitud, []string{"disponibilidad", "recibo"}, []string{"auditar"}) != nil {
		return nil, meritosports.ErrCarteraDenegada
	}
	material, err := emitirMaterialNominalMiBolsaDesarrollo(ctx, p.delegado, p.motivo,
		solicitud, resultado, decision, confirmacion)
	if err != nil {
		return nil, meritosports.ErrCarteraNoDisponible
	}
	return material, nil
}
