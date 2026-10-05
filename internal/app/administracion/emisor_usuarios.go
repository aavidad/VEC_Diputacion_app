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
		if emisores[audiencia] == nil || !domain.ReferenciaMotivoAutorizacionV2Valida(motivos[audiencia]) {
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
	if s.Validar() != nil || !versionRolUsuariosEmisorAdmitida(s.VersionRol.Referencia()) || s.VersionRol.Estado != domain.EstadoVersionRolPublicada || s.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada || ahora.Before(s.VersionRol.PublicadaEn) || ahora.Before(s.ControlVigenciaVersionRol.ActualizadoEn) || s.AsignacionPerfil.PrincipalID != actor.PersonaRef || s.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef || !s.AsignacionPerfil.VigenteEn(ahora) || !s.AsignacionPerfil.Cubre(emision.Recurso) {
		return false
	}
	for _, c := range s.VersionRol.Concesiones {
		if c.Accion == emision.Accion && c.ModuloID == "administracion" && c.TipoRecurso == emision.Recurso.Tipo {
			return slices.Equal(c.Finalidades, []string{"gestion_usuarios"}) && c.GarantiaMinima == domain.AuthAssuranceHigh && slices.Equal(c.CamposPermitidos, camposUsuarios(emision.Audiencia)) && slices.Equal(c.Obligaciones, []string{"auditar"})
		}
	}
	return false
}

// Clasifica la causa en un código cerrado. No conserva mensaje SQL, nombre,
// material criptográfico ni el texto del proveedor en el error entregado.
func errorEmisorUsuarios(err error) error {
	if err != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil
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
	if err != nil {
		return vacia, resultadoVacio, entradaVacia, errorEmisorUsuarios(err)
	}
	if !vinculo.CuentaPrivilegiada || vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 || vinculo.GarantiaObservada != domain.AuthAssuranceHigh {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	emision, err := formato.ValidarEmisionUsuariosAdministrables(entrada)
	if err != nil {
		return vacia, resultadoVacio, entradaVacia, errorEmisorUsuarios(err)
	}
	if e.emisores[emision.Audiencia] == nil || !snapshotUsuariosValido(snapshot, actor, emision, ahora) {
		return vacia, resultadoVacio, entradaVacia, fallo
	}
	resultado, err := evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacia, resultadoVacio, entradaVacia, errorEmisorUsuarios(err)
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: evidencia.Vinculo, ReferenciaMotivo: e.motivos[emision.Audiencia], Accion: emision.Accion, Recurso: emision.Recurso, Finalidad: "gestion_usuarios", Correlacion: emision.Correlacion})
	if err != nil {
		return vacia, resultadoVacio, entradaVacia, errorEmisorUsuarios(err)
	}
	return solicitud, resultado, emision, nil
}

// La vigencia se comprueba sobre la confirmación durable del registro, no
// sobre la decisión en memoria: DecisionAutorizacionLigadaV3.VigenteEn falla
// cerrado por diseño hasta que exista un tipo posterior al COMMIT, y ese tipo
// es precisamente la confirmación. Se liga a esta decisión, motivo y contexto
// con la orden de registro antes de mirar su ventana. El consumo SQL vuelve a
// exigir la decisión registrada; esto sólo evita entregar material caducado.
func validarDecisionUsuarios(d domain.DecisionAutorizacionLigadaV3, confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, solicitud domain.SolicitudAutorizacionLigadaV3, motivo domain.ReferenciaEntradaCatalogo, resultado domain.ResultadoContextoActorRegistradoV2, audiencia string, ahora time.Time) error {
	concedida, _, err := d.Resultado()
	if err != nil {
		return errorEmisorUsuarios(err)
	}
	if !concedida || d.ValidarPara(solicitud) != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, d, motivo, resultado)
	if err != nil {
		return errorEmisorUsuarios(err)
	}
	if confirmacion.ValidarPara(orden) != nil || !confirmacion.DentroDeVentanaEn(ahora.UTC().Truncate(time.Microsecond)) {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	b, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	if err != nil {
		return errorEmisorUsuarios(err)
	}
	var datos struct {
		VersionRol   string               `json:"version_rol_ref"`
		Garantia     domain.AuthAssurance `json:"garantia_minima"`
		Campos       []string             `json:"campos_permitidos"`
		Obligaciones []string             `json:"obligaciones"`
	}
	if err := json.Unmarshal(b, &datos); err != nil {
		return errorEmisorUsuarios(err)
	}
	if !versionRolUsuariosEmisorAdmitida(datos.VersionRol) || datos.Garantia != domain.AuthAssuranceHigh || !slices.Equal(datos.Campos, camposUsuarios(audiencia)) || !slices.Equal(datos.Obligaciones, []string{"auditar"}) {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil
}

func (e *EmisorUsuarios) EmitirLecturaUsuariosAdministrables(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, snapshot domain.InstantaneaAutorizacion, entrada ports.EmisionUsuariosAdministrables) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	fallo := ports.ErrLecturaUsuariosAdministrablesNoDisponible
	solicitud, resultado, emision, err := e.solicitud(ctx, actor, evidencia, snapshot, entrada)
	if err != nil {
		return vacia, errorEmisorUsuarios(err)
	}
	defer clear(emision.Material)
	decision, confirmacion, exportador, err := e.emisores[emision.Audiencia].EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		// La cadena común sólo emite este marcador tras comprobar el registro
		// durable de la denegación. No se infiere de 42501 ni de texto de error.
		if ctx.Err() == nil && errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacia, domain.ErrAutorizacionDenegada
		}
		return vacia, errorEmisorUsuarios(err)
	}
	ahora := e.reloj.Ahora()
	if ctx.Err() != nil || dependenciaConfianzaPerfilesNula(exportador) || confirmacion.Validar() != nil || validarDecisionUsuarios(decision, confirmacion, solicitud, e.motivos[emision.Audiencia], resultado, emision.Audiencia, ahora) != nil || !evidencia.Vinculo.VigenteEn(ahora, resultado) {
		return vacia, fallo
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacia, errorEmisorUsuarios(err)
	}
	if material.ValidarEstructura() != nil || ctx.Err() != nil {
		return vacia, fallo
	}
	huella, err := emision.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil {
		return vacia, errorEmisorUsuarios(err)
	}
	if resumen.Operacion() != emision.Accion || resumen.AudienciaConsumo() != emision.Audiencia || resumen.EfectoRef() != emision.Recurso.Referencia || resumen.EfectoHuellaSHA256() != huella || resumen.ContextoRef() != resultado.RegistroContextoRef || resumen.ContextoHuellaSHA256() != resultado.HuellaSHA256 || !bytes.Equal(material.ContextoActorCanonico(), resultado.RepresentacionCanonica) || material.PersonaVersion() != resultado.Contexto.Instantanea.PersonaVersion || material.PerfilVersion() != resultado.Contexto.Instantanea.PerfilVersion {
		return vacia, fallo
	}
	return material, nil
}

// Sólo versiones del rol fijo de Aplicación. La versión no se fija aquí:
// snapshotUsuariosValido exige además la concesión exacta de usuarios en esa
// versión y la autoridad PostgreSQL (AUT48) la vuelve a comprobar.
func versionRolUsuariosEmisorAdmitida(v string) bool {
	return domain.VersionRolAplicacionAdmitida(v)
}
