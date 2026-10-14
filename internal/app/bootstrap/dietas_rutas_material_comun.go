package bootstrap

import (
	"context"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Rutas recibe una capacidad nominal del publicador común. La autorización
// se evalúa una sola vez en el servicio de aplicación; este adaptador usa la
// decisión y su confirmación para emitir material, sin decidir de nuevo.
type proveedorMaterialRutasCTDesarrollo struct {
	comun *proveedorMaterialAltaContratacionTemporalDesarrollo
}

var _ dietasports.ProveedorMaterialAccesoRutas = (*proveedorMaterialRutasCTDesarrollo)(nil)

func descriptorMaterialRutasDietasDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        audienciaConsumoRutasDietasDesarrollo,
		Dominio:          "vec.dietas.rutas.acceso.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:dietas-rutas:",
		ProveedorNominal: "proveedor-material-dietas-rutas",
	}
}

func (p *proveedorMaterialRutasCTDesarrollo) ProveerMaterialAccesoRutas(
	ctx context.Context,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	decision vecdomain.DecisionAutorizacionLigadaV3,
	confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	contexto vecdomain.ResultadoContextoActorRegistradoV2,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || p.comun == nil || ctx == nil || ctx.Err() != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	datos, err := solicitud.Datos()
	if err != nil || datos.Recurso.ModuloID != "dietas" ||
		datos.Finalidad != dietasports.FinalidadConsultarItinerarioDietas ||
		!((datos.Accion == dietasports.AccionConsultarCatalogoRutasDietas && datos.Recurso.Tipo == dietasports.TipoCatalogoRutasDietas) ||
			(datos.Accion == dietasports.AccionSolicitarCalculoRutasDietas && datos.Recurso.Tipo == dietasports.TipoCalculoRutasDietas)) {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, datos.ReferenciaMotivo, contexto)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	registro, err := confirmacion.Datos()
	if err != nil || !confirmacion.DentroDeVentanaEn(registro.RegistradaEn) {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	material, err := p.comun.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, datos.ReferenciaMotivo, contexto)
	if err != nil || material.ValidarEstructura() != nil || material.ResumenCapacidad().AudienciaConsumo() != audienciaConsumoRutasDietasDesarrollo {
		return vacio, errMaterialRutasDietasDesarrollo
	}
	return material, nil
}
