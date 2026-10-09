package administracion

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	postgres "vec-diputacion-granada/internal/vec/adapters/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const FinalidadGobiernoInscripcion = "gobierno_definiciones_perfiles"

func NuevaConfianzaGobiernoInscripcionV3(cfg ConfiguracionConfianzaPerfilesV3,
	deps DependenciasConfianzaPerfilesV3) (ConfianzaPerfilesV3, error) {
	return nuevaConfianzaConAudienciasV3(cfg, deps, []string{
		postgres.AudienciaGobiernoInscripcionProponer, postgres.AudienciaGobiernoInscripcionAprobar,
	})
}

type EmisorGobiernoInscripcionV3 struct {
	emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3
	motivos  map[string]domain.ReferenciaEntradaCatalogo
	reloj    ports.Reloj
}

var _ postgres.EmisorGobiernoInscripcionV3 = (*EmisorGobiernoInscripcionV3)(nil)

func NuevoEmisorGobiernoInscripcionV3(
	emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3,
	motivos map[string]domain.ReferenciaEntradaCatalogo, reloj ports.Reloj,
) (*EmisorGobiernoInscripcionV3, error) {
	if len(emisores) != 2 || len(motivos) != 2 || dependenciaConfianzaPerfilesNula(reloj) {
		return nil, ErrConfiguracion
	}
	for _, audiencia := range []string{postgres.AudienciaGobiernoInscripcionProponer, postgres.AudienciaGobiernoInscripcionAprobar} {
		if emisores[audiencia] == nil || !domain.ReferenciaMotivoAutorizacionV2Valida(motivos[audiencia]) {
			return nil, ErrConfiguracion
		}
	}
	return &EmisorGobiernoInscripcionV3{emisores: maps.Clone(emisores), motivos: maps.Clone(motivos), reloj: reloj}, nil
}

func concesionGobiernoInscripcion(s domain.InstantaneaAutorizacion, accion, tipo string) bool {
	coincidencias := 0
	for _, c := range s.VersionRol.Concesiones {
		if c.Accion != accion {
			continue
		}
		if c.ModuloID != "administracion" || c.TipoRecurso != tipo ||
			!slices.Equal(c.Finalidades, []string{FinalidadGobiernoInscripcion}) ||
			c.GarantiaMinima != domain.AuthAssuranceHigh ||
			len(c.CamposPermitidos) != 0 || len(c.Obligaciones) != 0 {
			return false
		}
		coincidencias++
	}
	return coincidencias == 1
}

func snapshotGobiernoInscripcion(s domain.InstantaneaAutorizacion, actor domain.ContextoActor,
	recurso domain.RecursoAutorizable, accion string, ahora time.Time) bool {
	return s.Validar() == nil && s.VersionRol.RolID == "administracion_perfiles" &&
		domain.VersionRolAplicacionAdmitida(s.VersionRol.Referencia()) &&
		s.VersionRol.Estado == domain.EstadoVersionRolPublicada &&
		s.ControlVigenciaVersionRol.Estado == domain.EstadoControlVigenciaVersionRolHabilitada &&
		!ahora.Before(s.VersionRol.PublicadaEn) && !ahora.Before(s.ControlVigenciaVersionRol.ActualizadoEn) &&
		s.AsignacionPerfil.PrincipalID == actor.PersonaRef &&
		s.AsignacionPerfil.PerfilActivoRef == actor.PerfilActivoRef &&
		s.AsignacionPerfil.VigenteEn(ahora) && s.AsignacionPerfil.Cubre(recurso) &&
		concesionGobiernoInscripcion(s, accion, recurso.Tipo)
}

func (e *EmisorGobiernoInscripcionV3) EmitirGobiernoInscripcion(ctx context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, snapshot domain.InstantaneaAutorizacion,
	efecto postgres.EfectoGobiernoInscripcion) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacia ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	errNoDisponible := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if e == nil || ctx == nil || ctx.Err() != nil || dependenciaConfianzaPerfilesNula(e.reloj) ||
		e.emisores[efecto.Audiencia] == nil {
		return vacia, errNoDisponible
	}
	ahora := e.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return vacia, errNoDisponible
	}
	recurso, err := postgres.RecursoGobiernoInscripcion(efecto, snapshot.AsignacionPerfil)
	if err != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil || !vinculo.CuentaPrivilegiada ||
		vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
		vinculo.GarantiaObservada != domain.AuthAssuranceHigh ||
		!snapshotGobiernoInscripcion(snapshot, actor, recurso, efecto.Accion, ahora) {
		return vacia, errNoDisponible
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, errNoDisponible
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil || valor != efecto.CorrelacionRef {
		return vacia, errNoDisponible
	}
	resultado, err := evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacia, errNoDisponible
	}
	motivo := e.motivos[efecto.Audiencia]
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: evidencia.Vinculo, ReferenciaMotivo: motivo, Accion: efecto.Accion,
		Recurso: recurso, Finalidad: FinalidadGobiernoInscripcion, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, errNoDisponible
	}
	decision, confirmacion, exportador, err := e.emisores[efecto.Audiencia].EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		if ctx.Err() == nil && errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacia, domain.ErrAutorizacionDenegada
		}
		return vacia, errNoDisponible
	}
	ahora = e.reloj.Ahora()
	if ctx.Err() != nil || dependenciaConfianzaPerfilesNula(exportador) || confirmacion.Validar() != nil ||
		validarDecisionGobiernoRolNuevo(decision, confirmacion, solicitud, motivo, resultado, ahora) != nil ||
		!evidencia.Vinculo.VigenteEn(ahora, resultado) {
		return vacia, errNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacia, errNoDisponible
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, errNoDisponible
	}
	r := material.ResumenCapacidad()
	if material.ValidarEstructura() != nil || ctx.Err() != nil || r.Operacion() != efecto.Accion ||
		r.AudienciaConsumo() != efecto.Audiencia || r.EfectoRef() != recurso.Referencia ||
		r.EfectoHuellaSHA256() != h || r.ContextoRef() != resultado.RegistroContextoRef ||
		r.ContextoHuellaSHA256() != resultado.HuellaSHA256 ||
		!bytes.Equal(material.ContextoActorCanonico(), resultado.RepresentacionCanonica) ||
		material.PersonaVersion() != resultado.Contexto.Instantanea.PersonaVersion ||
		material.PerfilVersion() != resultado.Contexto.Instantanea.PerfilVersion {
		return vacia, errNoDisponible
	}
	return material, nil
}
