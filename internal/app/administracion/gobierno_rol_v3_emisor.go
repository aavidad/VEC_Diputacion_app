package administracion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"time"

	gobierno "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ConfianzaGobiernoRolNuevoV3 sólo admite las dos audiencias del corte.
// Reutiliza PDP, atestación, verificador y claves publicadas del ensamblaje
// ADMIN; no genera claves ni concesiones durante el arranque.
func NuevaConfianzaGobiernoRolNuevoV3(cfg ConfiguracionConfianzaPerfilesV3,
	deps DependenciasConfianzaPerfilesV3) (ConfianzaPerfilesV3, error) {
	return nuevaConfianzaConAudienciasV3(cfg, deps, []string{
		gobierno.AudienciaGobiernoRolProponer, gobierno.AudienciaGobiernoRolAprobar,
	})
}

type EmisorGobiernoRolNuevoV3 struct {
	emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3
	motivos  map[string]domain.ReferenciaEntradaCatalogo
	reloj    ports.Reloj
}

var _ gobierno.EmisorGobiernoRolNuevo = (*EmisorGobiernoRolNuevoV3)(nil)

func NuevoEmisorGobiernoRolNuevoV3(
	emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3,
	motivos map[string]domain.ReferenciaEntradaCatalogo, reloj ports.Reloj,
) (*EmisorGobiernoRolNuevoV3, error) {
	if len(emisores) != 2 || len(motivos) != 2 || dependenciaConfianzaPerfilesNula(reloj) {
		return nil, ErrConfiguracion
	}
	for _, audiencia := range []string{gobierno.AudienciaGobiernoRolProponer, gobierno.AudienciaGobiernoRolAprobar} {
		if emisores[audiencia] == nil || !domain.ReferenciaMotivoAutorizacionV2Valida(motivos[audiencia]) {
			return nil, ErrConfiguracion
		}
	}
	return &EmisorGobiernoRolNuevoV3{emisores: maps.Clone(emisores), motivos: maps.Clone(motivos), reloj: reloj}, nil
}

func concesionGobiernoRolNuevo(s domain.InstantaneaAutorizacion, accion, tipo string) bool {
	for _, c := range s.VersionRol.Concesiones {
		if c.Accion == accion && c.ModuloID == "administracion" && c.TipoRecurso == tipo {
			return slices.Equal(c.Finalidades, []string{gobierno.FinalidadGobiernoRol}) &&
				c.GarantiaMinima == domain.AuthAssuranceHigh && len(c.CamposPermitidos) == 0 && len(c.Obligaciones) == 0
		}
	}
	return false
}

func snapshotGobiernoRolNuevo(s domain.InstantaneaAutorizacion, actor domain.ContextoActor,
	recurso domain.RecursoAutorizable, accion string, ahora time.Time) bool {
	return s.Validar() == nil && domain.VersionRolAplicacionAdmitida(s.VersionRol.Referencia()) &&
		s.VersionRol.Estado == domain.EstadoVersionRolPublicada &&
		s.ControlVigenciaVersionRol.Estado == domain.EstadoControlVigenciaVersionRolHabilitada &&
		!ahora.Before(s.VersionRol.PublicadaEn) && !ahora.Before(s.ControlVigenciaVersionRol.ActualizadoEn) &&
		s.AsignacionPerfil.PrincipalID == actor.PersonaRef && s.AsignacionPerfil.PerfilActivoRef == actor.PerfilActivoRef &&
		s.AsignacionPerfil.VigenteEn(ahora) && s.AsignacionPerfil.Cubre(recurso) &&
		concesionGobiernoRolNuevo(s, accion, recurso.Tipo)
}

func (e *EmisorGobiernoRolNuevoV3) EmitirGobiernoRolNuevo(ctx context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, snapshot domain.InstantaneaAutorizacion,
	efecto gobierno.Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacia ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	fallo := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if e == nil || ctx == nil || ctx.Err() != nil || dependenciaConfianzaPerfilesNula(e.reloj) ||
		e.emisores[efecto.Audiencia] == nil {
		return vacia, fallo
	}
	ahora := e.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return vacia, fallo
	}
	recurso, err := gobierno.RecursoGobiernoRolNuevo(efecto, snapshot.AsignacionPerfil)
	if err != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil {
		return vacia, fallo
	}
	if !vinculo.CuentaPrivilegiada || vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
		vinculo.GarantiaObservada != domain.AuthAssuranceHigh ||
		!snapshotGobiernoRolNuevo(snapshot, actor, recurso, efecto.Accion, ahora) {
		return vacia, fallo
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, fallo
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil || valor != efecto.CorrelacionAccesoRef {
		return vacia, fallo
	}
	resultado, err := evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacia, fallo
	}
	motivo := e.motivos[efecto.Audiencia]
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: evidencia.Vinculo, ReferenciaMotivo: motivo, Accion: efecto.Accion,
		Recurso: recurso, Finalidad: gobierno.FinalidadGobiernoRol, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, fallo
	}
	decision, confirmacion, exportador, err := e.emisores[efecto.Audiencia].EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		if ctx.Err() == nil && errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacia, domain.ErrAutorizacionDenegada
		}
		return vacia, fallo
	}
	ahora = e.reloj.Ahora()
	if ctx.Err() != nil || dependenciaConfianzaPerfilesNula(exportador) || confirmacion.Validar() != nil ||
		validarDecisionGobiernoRolNuevo(decision, confirmacion, solicitud, motivo, resultado, ahora) != nil ||
		!evidencia.Vinculo.VigenteEn(ahora, resultado) {
		return vacia, fallo
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacia, fallo
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, fallo
	}
	r := material.ResumenCapacidad()
	if material.ValidarEstructura() != nil || ctx.Err() != nil || r.Operacion() != efecto.Accion ||
		r.AudienciaConsumo() != efecto.Audiencia || r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != h ||
		r.ContextoRef() != resultado.RegistroContextoRef || r.ContextoHuellaSHA256() != resultado.HuellaSHA256 ||
		!bytes.Equal(material.ContextoActorCanonico(), resultado.RepresentacionCanonica) ||
		material.PersonaVersion() != resultado.Contexto.Instantanea.PersonaVersion ||
		material.PerfilVersion() != resultado.Contexto.Instantanea.PerfilVersion {
		return vacia, fallo
	}
	return material, nil
}

func validarDecisionGobiernoRolNuevo(d domain.DecisionAutorizacionLigadaV3,
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	solicitud domain.SolicitudAutorizacionLigadaV3, motivo domain.ReferenciaEntradaCatalogo,
	resultado domain.ResultadoContextoActorRegistradoV2, ahora time.Time) error {
	concedida, _, err := d.Resultado()
	if err != nil || !concedida || d.ValidarPara(solicitud) != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, d, motivo, resultado)
	if err != nil || confirmacion.ValidarPara(orden) != nil ||
		!confirmacion.DentroDeVentanaEn(ahora.UTC().Truncate(time.Microsecond)) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	b, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	if err != nil {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var datos struct {
		VersionRol   string               `json:"version_rol_ref"`
		Garantia     domain.AuthAssurance `json:"garantia_minima"`
		Campos       []string             `json:"campos_permitidos"`
		Obligaciones []string             `json:"obligaciones"`
	}
	if json.Unmarshal(b, &datos) != nil || !domain.VersionRolAplicacionAdmitida(datos.VersionRol) ||
		datos.Garantia != domain.AuthAssuranceHigh || len(datos.Campos) != 0 || len(datos.Obligaciones) != 0 {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}
