package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strconv"
	"time"

	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	accionLecturaRespuestasLlamamiento    = "bolsa.rrhh.respuestas_llamamiento.consultar"
	finalidadLecturaRespuestasLlamamiento = "seguimiento_contacto_rrhh"
)

var camposLecturaRespuestasLlamamiento = []string{
	"llamamiento_ref", "respuestas[].participacion_ref", "respuestas[].respuesta",
}

func concesionLecturaRespuestasLlamamientoExacta(i vecdomain.InstantaneaAutorizacion) bool {
	coincidencias := 0
	for _, c := range i.VersionRol.Concesiones {
		if c.Accion != accionLecturaRespuestasLlamamiento {
			continue
		}
		campos := slices.Clone(c.CamposPermitidos)
		slices.Sort(campos)
		if c.ModuloID != "bolsa" || c.TipoRecurso != "llamamiento_bolsa" ||
			!slices.Equal(c.Finalidades, []string{finalidadLecturaRespuestasLlamamiento}) ||
			!slices.Equal(campos, camposLecturaRespuestasLlamamiento) ||
			(len(c.Obligaciones) != 0 && !slices.Equal(c.Obligaciones, []string{"auditar"})) {
			return false
		}
		coincidencias++
	}
	return coincidencias == 1
}

// CapturarLecturaRespuestasLlamamiento evalúa una concesión actual para el
// vínculo real de la petición. La lectura de Bolsa vuelve a cotejar la
// captura dentro de su transacción antes de devolver cualquier respuesta.
func CapturarLecturaRespuestasLlamamiento(
	ctx context.Context, fuente vecports.FuenteAutorizacion,
	vinculo vecdomain.VinculoAutenticacionActorV2,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
	bolsaRef, llamamientoRef string,
) (vecports.CapturaLecturaNominalRespuestasLlamamiento, error) {
	vacia := vecports.CapturaLecturaNominalRespuestasLlamamiento{}
	if ctx == nil || ctx.Err() != nil || nuloInscripcionBolsa(fuente) ||
		resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		bolsaRef == "" || llamamientoRef == "" {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	recurso := vecdomain.RecursoAutorizable{
		Referencia: llamamientoRef, ModuloID: "bolsa", Tipo: "llamamiento_bolsa",
		Ambitos:   map[string]string{"bolsa_ref": bolsaRef},
		Atributos: map[string]string{"llamamiento_ref": llamamientoRef},
	}
	if recurso.Validar() != nil {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if !vinculo.VigenteEn(ahora, resultado) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	instantanea, err := fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil || ctx.Err() != nil || instantanea.Validar() != nil ||
		instantanea.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		instantanea.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		instantanea.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" ||
		!concesionLecturaRespuestasLlamamientoExacta(instantanea) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	ahora = time.Now().UTC().Truncate(time.Microsecond)
	evaluacion, err := vecdomain.EvaluarLecturaActualAutorizacionV3(vinculo, resultado,
		instantanea, accionLecturaRespuestasLlamamiento, recurso,
		finalidadLecturaRespuestasLlamamiento, camposLecturaRespuestasLlamamiento, ahora)
	if err != nil || !evaluacion.Permitido ||
		!slices.Equal(evaluacion.CamposPermitidos, camposLecturaRespuestasLlamamiento) ||
		(len(evaluacion.Obligaciones) != 0 && !slices.Equal(evaluacion.Obligaciones, []string{"auditar"})) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx,
		seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, vecports.ErrFuenteAutorizacionNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, vecports.ErrFuenteAutorizacionNoDisponible
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return vacia, vecports.ErrFuenteAutorizacionNoDisponible
	}
	ahora = time.Now().UTC().Truncate(time.Microsecond)
	limite := ahora.Add(30 * time.Second)
	if evaluacion.ValidaHasta.Before(limite) {
		limite = evaluacion.ValidaHasta
	}
	if datos.SesionValidaHasta.Before(limite) {
		limite = datos.SesionValidaHasta
	}
	if ctx.Err() != nil || !vinculo.VigenteEn(ahora, resultado) || !limite.After(ahora) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	contextoCanonico := bytes.Clone(resultado.RepresentacionCanonica)
	vinculoCanonico, err := json.Marshal(vinculo)
	if err != nil || len(contextoCanonico) == 0 {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	contextoHuella := sha256.Sum256(contextoCanonico)
	orden, err := json.Marshal(struct {
		IntentoRef                    string                                              `json:"intento_ref"`
		RegistroContextoRef           string                                              `json:"registro_contexto_ref"`
		ContextoSHA256                string                                              `json:"contexto_sha256"`
		ProcedenciaSHA256             string                                              `json:"procedencia_sha256"`
		AutenticacionRef              string                                              `json:"autenticacion_ref"`
		SesionRef                     string                                              `json:"sesion_ref"`
		AutenticacionSHA256           string                                              `json:"autenticacion_sha256"`
		Accion                        string                                              `json:"accion"`
		ModuloID                      string                                              `json:"modulo_id"`
		RecursoRef                    string                                              `json:"recurso_ref"`
		BolsaRef                      string                                              `json:"bolsa_ref"`
		FinalidadRef                  string                                              `json:"finalidad_ref"`
		CamposPermitidos              []string                                            `json:"campos_permitidos"`
		Resultado                     string                                              `json:"resultado"`
		MotivoRef                     string                                              `json:"motivo_ref"`
		Proceso                       string                                              `json:"proceso"`
		Canal                         string                                              `json:"canal"`
		CorrelacionRef                string                                              `json:"correlacion_ref"`
		LecturaRevisionPermisos       string                                              `json:"lectura_revision_permisos"`
		LecturaInstantaneaSHA256      string                                              `json:"lectura_instantanea_sha256"`
		AsignacionRef                 string                                              `json:"asignacion_ref"`
		AsignacionVersion             int                                                 `json:"asignacion_version"`
		AsignacionHuellaSHA256        string                                              `json:"asignacion_huella_sha256"`
		VersionRolRef                 string                                              `json:"version_rol_ref"`
		VersionRolVersion             int                                                 `json:"version_rol_version"`
		VersionRolHuellaSHA256        string                                              `json:"version_rol_huella_sha256"`
		ControlVersionRolRef          string                                              `json:"control_version_rol_ref"`
		RevisionControlVersionRol     uint64                                              `json:"revision_control_version_rol"`
		ControlVersionRolHuellaSHA256 string                                              `json:"control_version_rol_huella_sha256"`
		RevisionCatalogoPoliticas     uint64                                              `json:"revision_catalogo_politicas"`
		CatalogoPoliticasHuellaSHA256 string                                              `json:"catalogo_politicas_huella_sha256"`
		PoliticasEvaluadas            []vecdomain.ReferenciaHuellaPoliticaLecturaActualV3 `json:"politicas_evaluadas"`
		EmitidaEn                     time.Time                                           `json:"emitida_en"`
		ValidaHasta                   time.Time                                           `json:"valida_hasta"`
	}{
		IntentoRef:          "lectura_" + hex.EncodeToString(aleatorio[:]),
		RegistroContextoRef: datos.RegistroContextoRef,
		ContextoSHA256:      hex.EncodeToString(contextoHuella[:]),
		ProcedenciaSHA256:   datos.ManifiestoProcedenciaHuellaSHA256,
		AutenticacionRef:    datos.AutenticacionRef,
		SesionRef:           datos.SesionRef,
		AutenticacionSHA256: datos.AutenticacionHuellaSHA256,
		Accion:              accionLecturaRespuestasLlamamiento,
		ModuloID:            "bolsa", RecursoRef: llamamientoRef, BolsaRef: bolsaRef,
		FinalidadRef:     finalidadLecturaRespuestasLlamamiento,
		CamposPermitidos: slices.Clone(camposLecturaRespuestasLlamamiento),
		Resultado:        "obtenida", MotivoRef: "lectura_respuestas_llamamiento_correcta",
		Proceso: "vec-server", Canal: "interna_corporativa", CorrelacionRef: correlacionRef,
		LecturaRevisionPermisos:  strconv.FormatUint(evaluacion.RevisionPermisos, 10),
		LecturaInstantaneaSHA256: evaluacion.HuellaInstantaneaSHA256,
		AsignacionRef:            evaluacion.AsignacionRef, AsignacionVersion: evaluacion.AsignacionVersion,
		AsignacionHuellaSHA256: evaluacion.AsignacionHuellaSHA256,
		VersionRolRef:          evaluacion.VersionRolRef, VersionRolVersion: evaluacion.VersionRolVersion,
		VersionRolHuellaSHA256:        evaluacion.VersionRolHuellaSHA256,
		ControlVersionRolRef:          evaluacion.ControlVersionRolRef,
		RevisionControlVersionRol:     evaluacion.RevisionControlVersionRol,
		ControlVersionRolHuellaSHA256: evaluacion.ControlVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:     evaluacion.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256: evaluacion.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:            slices.Clone(evaluacion.PoliticasEvaluadas),
		EmitidaEn:                     evaluacion.EvaluadaEn, ValidaHasta: limite,
	})
	if err != nil {
		return vacia, vecports.ErrFuenteAutorizacionNoDisponible
	}
	return vecports.CapturaLecturaNominalRespuestasLlamamiento{
		ContextoCanonico: contextoCanonico, VinculoCanonico: vinculoCanonico,
		Orden: orden, BolsaRef: bolsaRef, LlamamientoRef: llamamientoRef,
		ValidaHasta: limite,
	}, nil
}
