package bootstrap

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// proveedorImagenUsuarios emite la V3 nominal de cada acción de «Mi imagen»
// con el recurso canónico que PostgreSQL vuelve a calcular. Solo sirve a la
// persona de la sesión de esta misma frontera: nunca a un tercero.
type proveedorImagenUsuarios struct {
	autoridad           *autoridadPreferenciasUsuariosDesarrollo
	emisores            map[string]emisorPreferenciasUsuarios
	motivoConsulta      core.ReferenciaEntradaCatalogo
	motivoActualizacion core.ReferenciaEntradaCatalogo
}

func (p *proveedorImagenUsuarios) ProveerMaterialImagen(ctx context.Context, vinculo core.VinculoAutenticacionActorV2, m usuariosports.MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || p.autoridad == nil || ctx == nil || ctx.Err() != nil {
		return vacia, usuariosports.ErrImagenNoDisponible
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	datosEntrada, errEntrada := vinculo.Datos()
	datosContexto, errContexto := c.vinculo.Datos()
	if !ok || c.autoridad != p.autoridad || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil ||
		!c.vinculo.VigenteEn(p.autoridad.reloj.Ahora(), c.resultado) || c.resultado.Contexto.PersonaRef != m.PersonaRef ||
		c.resultado.Contexto.PerfilActivoRef != m.PerfilRef || m.FinalidadRef != usuariosports.FinalidadImagenPropia ||
		m.Superficie != p.autoridad.superficie || errEntrada != nil || errContexto != nil || !reflect.DeepEqual(datosEntrada, datosContexto) {
		return vacia, usuariosports.ErrImagenProhibido
	}
	emisor := p.emisores[m.Accion]
	motivo := p.motivoActualizacion
	if m.Accion == usuariosports.AccionConsultarImagen {
		motivo = p.motivoConsulta
	}
	audiencia, err := canonico.AudienciaImagen(m.Accion, p.autoridad.superficie)
	if err != nil {
		return vacia, usuariosports.ErrImagenProhibido
	}
	if emisor == nil || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return vacia, usuariosports.ErrImagenNoDisponible
	}
	recurso, err := canonico.RecursoImagen(m)
	if err != nil {
		return vacia, err
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, usuariosports.ErrImagenNoDisponible
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: c.vinculo, ReferenciaMotivo: motivo, Accion: m.Accion,
		Recurso: recurso, Finalidad: m.FinalidadRef, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, usuariosports.ErrImagenProhibido
	}
	decision, confirmacion, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, c.resultado)
	if errors.Is(err, core.ErrAutorizacionDenegada) {
		return vacia, usuariosports.ErrImagenProhibido
	}
	if err != nil {
		return vacia, usuariosports.ErrImagenNoDisponible
	}
	if decision.ValidarPara(solicitud) != nil || exportador == nil {
		return vacia, usuariosports.ErrImagenProhibido
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, c.resultado, motivo, material, audiencia) {
		return vacia, usuariosports.ErrImagenProhibido
	}
	return material, nil
}
