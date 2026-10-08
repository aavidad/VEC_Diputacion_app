package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/telemetria"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type emisorPreferenciasUsuarios interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type proveedorPreferenciasUsuarios struct {
	autoridad           *autoridadPreferenciasUsuariosDesarrollo
	consulta            emisorPreferenciasUsuarios
	actualizacion       emisorPreferenciasUsuarios
	motivoConsulta      core.ReferenciaEntradaCatalogo
	motivoActualizacion core.ReferenciaEntradaCatalogo
}

func recursoPreferenciasUsuarios(m usuariosports.MaterialPreferencias) (core.RecursoAutorizable, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return core.RecursoAutorizable{}, usuariosports.ErrPeticionInvalida
	}
	huella := sha256.Sum256(b)
	recurso := core.RecursoAutorizable{Referencia: m.PersonaRef, ModuloID: "usuarios", Tipo: usuariosports.TipoRecursoPreferencias,
		Ambitos:   map[string]string{"persona_ref": m.PersonaRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(huella[:])}}
	if recurso.Validar() != nil {
		return core.RecursoAutorizable{}, usuariosports.ErrPeticionInvalida
	}
	return recurso, nil
}

func (p *proveedorPreferenciasUsuarios) ProveerMaterialPreferencias(ctx context.Context, vinculo core.VinculoAutenticacionActorV2, m usuariosports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || p.autoridad == nil || ctx == nil || ctx.Err() != nil {
		return vacia, usuariosports.ErrNoDisponible
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	datosEntrada, errEntrada := vinculo.Datos()
	datosContexto, errContexto := c.vinculo.Datos()
	if !ok || c.autoridad != p.autoridad || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil ||
		!c.vinculo.VigenteEn(p.autoridad.reloj.Ahora(), c.resultado) || c.resultado.Contexto.PersonaRef != m.PersonaRef ||
		c.resultado.Contexto.PerfilActivoRef != m.PerfilRef || m.FinalidadRef != usuariosports.FinalidadPreferenciasPropias ||
		m.Superficie != p.autoridad.superficie || errEntrada != nil || errContexto != nil || !reflect.DeepEqual(datosEntrada, datosContexto) {
		return vacia, usuariosports.ErrProhibido
	}
	emisor, motivo := p.consulta, p.motivoConsulta
	if m.Accion == usuariosports.AccionActualizarPreferencias {
		emisor, motivo = p.actualizacion, p.motivoActualizacion
	} else if m.Accion != usuariosports.AccionConsultarPreferencias {
		return vacia, usuariosports.ErrPeticionInvalida
	}
	audiencia, err := usuariosports.AudienciaPreferencias(m.Accion, p.autoridad.superficie)
	if err != nil {
		return vacia, usuariosports.ErrProhibido
	}
	if emisor == nil || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return vacia, usuariosports.ErrNoDisponible
	}
	recurso, err := recursoPreferenciasUsuarios(m)
	if err != nil {
		return vacia, err
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, usuariosports.ErrNoDisponible
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: c.vinculo, ReferenciaMotivo: motivo, Accion: m.Accion,
		Recurso: recurso, Finalidad: m.FinalidadRef, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, usuariosports.ErrProhibido
	}
	inicioV3 := time.Now()
	decision, confirmacion, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, c.resultado)
	telemetria.RegistrarFase(ctx, telemetria.FaseV3, time.Since(inicioV3), err)
	if errors.Is(err, core.ErrAutorizacionDenegada) {
		return vacia, usuariosports.ErrProhibido
	}
	if err != nil {
		return vacia, usuariosports.ErrNoDisponible
	}
	if decision.ValidarPara(solicitud) != nil || exportador == nil {
		return vacia, usuariosports.ErrProhibido
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, c.resultado, motivo, material, audiencia) {
		return vacia, usuariosports.ErrProhibido
	}
	return material, nil
}
