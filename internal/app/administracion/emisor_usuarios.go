package administracion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"time"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	formato "vec-diputacion-granada/internal/vec/adapters/usuariosadministrables/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// EmisorUsuarios sólo adapta las dos solicitudes nominales a la cadena común
// real. Los motivos proceden del catálogo privado y no existen en el DTO.
type EmisorUsuarios struct {
	emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3
	motivos  map[string]domain.ReferenciaEntradaCatalogo
	reloj    ports.Reloj
}

var _ ports.EmisorLecturaUsuariosAdministrables = (*EmisorUsuarios)(nil)

func NuevoEmisorUsuarios(emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3, motivos map[string]domain.ReferenciaEntradaCatalogo, reloj ports.Reloj) (*EmisorUsuarios, error) {
	if len(emisores) != 2 || len(motivos) != 2 || dependenciaConfianzaPerfilesNula(reloj) {
		return nil, ErrConfiguracion
	}
	for _, audiencia := range []string{AudienciaUsuariosListarV3, AudienciaUsuariosConsultarV3} {
		if emisores[audiencia] == nil || motivos[audiencia].Validar() != nil {
			return nil, ErrConfiguracion
		}
	}
	return &EmisorUsuarios{emisores: maps.Clone(emisores), motivos: maps.Clone(motivos), reloj: reloj}, nil
}

func camposUsuarios(audiencia string) []string {
	if audiencia == AudienciaUsuariosListarV3 {
		return []string{"denominacion_version", "perfiles", "persona_ref", "siguiente_cursor", "unidad_ref"}
	}
	if audiencia == AudienciaUsuariosConsultarV3 {
		return []string{"denominacion_version", "perfiles", "persona_ref", "unidad_ref"}
	}
	return nil
}

func snapshotUsuariosValido(s domain.InstantaneaAutorizacion, actor domain.ContextoActor, emision ports.EmisionUsuariosAdministrables, ahora time.Time) bool {
	if s.Validar() != nil || s.VersionRol.Referencia() != "rol:administracion_perfiles:v5" || s.VersionRol.Estado != domain.EstadoVersionRolPublicada || s.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada || ahora.Before(s.VersionRol.PublicadaEn) || ahora.Before(s.ControlVigenciaVersionRol.ActualizadoEn) || s.AsignacionPerfil.PrincipalID != actor.PersonaRef || s.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef || !s.AsignacionPerfil.VigenteEn(ahora) || !s.AsignacionPerfil.Cubre(emision.Recurso) {
		return false
	}
	for _, c := range s.VersionRol.Concesiones {
		if c.Accion == emision.Accion && c.ModuloID == "administracion" && c.TipoRecurso == emision.Recurso.Tipo {
			return slices.Equal(c.Finalidades, []string{"gestion_usuarios"}) && c.GarantiaMinima == domain.AuthAssuranceHigh && slices.Equal(c.CamposPermitidos, camposUsuarios(emision.Audiencia)) && slices.Equal(c.Obligaciones, []string{"auditar"})
		}
	}
	return false
}

// La instantánea recibida es una precondición, no la autoridad de permiso: el
// PDP común vuelve a consultar su fuente durable al evaluar la solicitud.
func (e *EmisorUsuarios) solicitud(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, snapshot domain.InstantaneaAutorizacion, entrada ports.EmisionUsuariosAdministrables) (domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2, ports.EmisionUsuariosAdministrables, error) {
	vacia := domain.SolicitudAutorizacionLigadaV3{}
	resultadoVacio := domain.ResultadoContextoActorRegistradoV2{}
	entradaVacia := ports.EmisionUsuariosAdministrables{}
	fallo := ports.ErrLecturaUsuariosAdministrablesNoDisponible
	if e == nil || ctx == nil || ctx.Err() != nil || dependenciaConfianzaPerfilesNula(e.reloj) {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	ahora := e.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil || !vinculo.CuentaPrivilegiada || vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 || vinculo.GarantiaObservada != domain.AuthAssuranceHigh {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	emision, err := formato.ValidarEmisionUsuariosAdministrables(entrada)
	if err != nil || e.emisores[emision.Audiencia] == nil || !snapshotUsuariosValido(snapshot, actor, emision, ahora) {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	resultado, err := evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: evidencia.Vinculo, ReferenciaMotivo: e.motivos[emision.Audiencia], Accion: emision.Accion, Recurso: emision.Recurso, Finalidad: "gestion_usuarios", Correlacion: emision.Correlacion})
	if err != nil {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	return solicitud, resultado, emision, nil
}

func decisionUsuariosValida(d domain.DecisionAutorizacionLigadaV3, solicitud domain.SolicitudAutorizacionLigadaV3, audiencia string, ahora time.Time) bool {
	concedida, _, err := d.Resultado()
	if err != nil || !concedida || d.ValidarPara(solicitud) != nil || !d.VigenteEn(ahora) {
		return false
	}
	b, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	if err != nil {
		return false
	}
	var datos struct {
		VersionRol   string               `json:"version_rol_ref"`
		Garantia     domain.AuthAssurance `json:"garantia_minima"`
		Campos       []string             `json:"campos_permitidos"`
		Obligaciones []string             `json:"obligaciones"`
	}
	if json.Unmarshal(b, &datos) != nil {
		return false
	}
	return datos.VersionRol == "rol:administracion_perfiles:v5" && datos.Garantia == domain.AuthAssuranceHigh && slices.Equal(datos.Campos, camposUsuarios(audiencia)) && slices.Equal(datos.Obligaciones, []string{"auditar"})
}

func (e *EmisorUsuarios) EmitirLecturaUsuariosAdministrables(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, snapshot domain.InstantaneaAutorizacion, entrada ports.EmisionUsuariosAdministrables) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	fallo := ports.ErrLecturaUsuariosAdministrablesNoDisponible
	solicitud, resultado, emision, err := e.solicitud(ctx, actor, evidencia, snapshot, entrada)
	if err != nil {
		return vacia, fallo
	}
	defer clear(emision.Material)
	decision, confirmacion, exportador, err := e.emisores[emision.Audiencia].EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		// La cadena común sólo emite este marcador tras comprobar el registro
		// durable de la denegación. No se infiere de 42501 ni de texto de error.
		if ctx.Err() == nil && errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacia, domain.ErrAutorizacionDenegada
		}
		return vacia, fallo
	}
	ahora := e.reloj.Ahora()
	if ctx.Err() != nil || dependenciaConfianzaPerfilesNula(exportador) || confirmacion.Validar() != nil || !decisionUsuariosValida(decision, solicitud, emision.Audiencia, ahora) || !evidencia.Vinculo.VigenteEn(ahora, resultado) {
		return vacia, fallo
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || material.ValidarEstructura() != nil || ctx.Err() != nil {
		return vacia, fallo
	}
	huella, err := emision.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil || resumen.Operacion() != emision.Accion || resumen.AudienciaConsumo() != emision.Audiencia || resumen.EfectoRef() != emision.Recurso.Referencia || resumen.EfectoHuellaSHA256() != huella || resumen.ContextoRef() != resultado.RegistroContextoRef || resumen.ContextoHuellaSHA256() != resultado.HuellaSHA256 || !bytes.Equal(material.ContextoActorCanonico(), resultado.RepresentacionCanonica) || material.PersonaVersion() != resultado.Contexto.Instantanea.PersonaVersion || material.PerfilVersion() != resultado.Contexto.Instantanea.PerfilVersion {
		return vacia, fallo
	}
	return material, nil
}
