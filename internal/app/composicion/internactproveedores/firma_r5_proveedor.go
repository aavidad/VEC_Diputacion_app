package internactproveedores

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrProveedorFirmaR5NoDisponible = errors.New("composicion interna: proveedor original R5 no disponible")

const (
	sqlComprobarMaterialFirmaR5 = `SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('ct_firma_r5',$1::jsonb)`
	sqlLeerConfiguracionFirmaR5 = `SELECT vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct_firma_r5',$1::jsonb)`
)

// ProveedorFirmaR5 usa el PDP, firmante, raíz y F1 ya compuestos por CT.
// Conserva una sola capacidad documental renovable de la publicación común.
type ProveedorFirmaR5 struct {
	fuente *internagobierno.FuenteF1
	motivo core.ReferenciaEntradaCatalogo
	emisor *emisorMaterialRenovable
	reloj  ports.Reloj
}

func (p *ProveedorFirmaR5) Cerrar() {
	if p == nil {
		return
	}
	p.fuente, p.emisor, p.reloj = nil, nil, nil
	p.motivo = core.ReferenciaEntradaCatalogo{}
}

func ConstruirProveedorFirmaR5(ctx context.Context, base *Proveedores, materialCT Material, material MaterialFirmaR5,
	fuente *internagobierno.FuenteF1, reloj ports.Reloj) (*ProveedorFirmaR5, error) {
	if ctx == nil || ctx.Err() != nil || base == nil || base.pdp == nil || base.atestador == nil ||
		base.firmante == nil || base.gobierno == nil || base.gobierno.lector == nil ||
		len(base.pools) < 4 || base.pools[3] == nil || materialCT.raiz == nil ||
		fuente == nil || reloj == nil || material.CatalogoMotivos != base.catalogoMotivos ||
		!core.ReferenciaMotivoAutorizacionV2Valida(material.MotivoOriginal) ||
		material.MotivoOriginal.CatalogoID != base.catalogoMotivos {
		return nil, ErrProveedorFirmaR5NoDisponible
	}
	claves := []claveGobiernoV3{claveGobierno(material.CapacidadDocumentos, docports.AudienciaV3)}
	_, publicacion, err := base.gobierno.lector.Leer(ctx)
	if err != nil {
		return nil, ErrProveedorFirmaR5NoDisponible
	}
	sonda, err := (materialGobiernoV3{Claves: claves, Raiz: base.gobierno.coord}).codificar(publicacion)
	if err != nil {
		return nil, ErrProveedorFirmaR5NoDisponible
	}
	vale, err := comprobacionMaterialGobiernoV3(base.pools[3], sqlComprobarMaterialFirmaR5)(ctx, sonda)
	clear(sonda)
	if err != nil || !vale {
		return nil, ErrProveedorFirmaR5NoDisponible
	}
	lector, err := nuevoLectorGobiernoV3(ctx, publicacion, base.gobierno.raiz, reloj,
		fuenteGobiernoV3(consultaLecturaGobiernoV3(base.pools[3], sqlLeerConfiguracionFirmaR5), claves, base.gobierno.coord))
	if err != nil {
		return nil, ErrProveedorFirmaR5NoDisponible
	}
	capacidad, err := crearCapacidad(materialCT.raiz, material.CapacidadDocumentos, docports.AudienciaV3, reloj)
	if err != nil {
		return nil, ErrProveedorFirmaR5NoDisponible
	}
	return &ProveedorFirmaR5{fuente: fuente, motivo: material.MotivoOriginal, reloj: reloj,
		emisor: &emisorMaterialRenovable{lector: lector, pdp: base.pdp,
			atestador: base.atestador, capacidad: capacidad}}, nil
}

// EmitirMaterialOriginal obtiene la decisión nominal del actor sellado de esta
// petición; ninguna operación recibe un actor o capacidad del transporte.
func (p *ProveedorFirmaR5) EmitirMaterialOriginal(ctx context.Context, accion, finalidad string,
	recurso core.RecursoAutorizable) (core.SolicitudAutorizacionLigadaV3,
	core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var s core.SolicitudAutorizacionLigadaV3
	var d core.DecisionAutorizacionLigadaV3
	var c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	var m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if p == nil || p.fuente == nil || p.emisor == nil || p.reloj == nil ||
		ctx == nil || ctx.Err() != nil || recurso.Validar() != nil || recurso.ModuloID != "documentos" {
		return s, d, c, m, ErrProveedorFirmaR5NoDisponible
	}
	base, err := p.fuente.ContextoVinculadoOriginalCT(ctx)
	if err != nil || base.Resultado.Validar() != nil || base.Vinculo.ValidarPara(base.Resultado) != nil ||
		!base.Vinculo.VigenteEn(p.reloj.Ahora(), base.Resultado) {
		return s, d, c, m, ErrProveedorFirmaR5NoDisponible
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return s, d, c, m, ErrProveedorFirmaR5NoDisponible
	}
	s, err = core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: base.Vinculo, ReferenciaMotivo: p.motivo,
		Accion: accion, Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion})
	if err != nil {
		return s, d, c, m, ErrProveedorFirmaR5NoDisponible
	}
	var exportador vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3
	d, c, exportador, err = p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, base.Resultado)
	if err != nil || d.ValidarPara(s) != nil || interfazNula(exportador) {
		return core.SolicitudAutorizacionLigadaV3{}, core.DecisionAutorizacionLigadaV3{},
			vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, m, ErrProveedorFirmaR5NoDisponible
	}
	m, err = exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(s, d, c, base.Resultado, p.motivo, m, docports.AudienciaV3) || ctx.Err() != nil {
		return core.SolicitudAutorizacionLigadaV3{}, core.DecisionAutorizacionLigadaV3{},
			vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrProveedorFirmaR5NoDisponible
	}
	return s, d, c, m, nil
}
