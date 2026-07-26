package registroaccesos

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const formatoInstanteFiltroConsultaAccesos = "2006-01-02T15:04:05.000000Z"

func formatearInstanteFiltroConsultaAccesos(instante time.Time) string {
	return instante.UTC().Format(formatoInstanteFiltroConsultaAccesos)
}

// atributosAutorizacionFiltroConsultaAccesos materializa el filtro completo
// dentro del contexto de recurso evaluado por VEC. No es una huella aportada
// por cliente: la decisión durable liga el canónico exacto de estos atributos.
func atributosAutorizacionFiltroConsultaAccesos(
	filtro FiltroConsultaAdministrativaAccesos,
	actorOperador string,
) map[string]string {
	return map[string]string{
		"actor_operador_seudonimizado": actorOperador,
		"finalidad_consulta":           filtro.FinalidadDeLaConsulta,
		"filtro_version_sha256": huellaValorFiltroConsultaAccesos(
			strconv.FormatUint(uint64(filtro.Version), 10),
		),
		"filtro_actor_seudonimizado_sha256": huellaValorFiltroConsultaAccesos(
			filtro.ActorSeudonimizado,
		),
		"filtro_modulo_id_sha256": huellaValorFiltroConsultaAccesos(
			filtro.ModuloID,
		),
		"filtro_accion_sha256": huellaValorFiltroConsultaAccesos(filtro.Accion),
		"filtro_finalidad_acceso_sha256": huellaValorFiltroConsultaAccesos(
			filtro.FinalidadAcceso,
		),
		"filtro_recurso_ref_sha256": huellaValorFiltroConsultaAccesos(
			filtro.RecursoRef,
		),
		"filtro_expediente_ref_sha256": huellaValorFiltroConsultaAccesos(
			filtro.ExpedienteRef,
		),
		"filtro_resultado_sha256": huellaValorFiltroConsultaAccesos(
			filtro.Resultado,
		),
		"filtro_desde_inclusive_sha256": huellaValorFiltroConsultaAccesos(
			formatearInstanteFiltroConsultaAccesos(filtro.DesdeInclusive),
		),
		"filtro_hasta_exclusive_sha256": huellaValorFiltroConsultaAccesos(
			formatearInstanteFiltroConsultaAccesos(filtro.HastaExclusive),
		),
		"filtro_version_objeto_sha256": huellaValorFiltroConsultaAccesos(
			strconv.FormatUint(filtro.VersionObjeto, 10),
		),
		"filtro_limite_sha256": huellaValorFiltroConsultaAccesos(
			strconv.FormatUint(uint64(filtro.Limite), 10),
		),
		"filtro_cursor_sha256": huellaValorFiltroConsultaAccesos(
			filtro.Cursor.Valor(),
		),
		"filtro_finalidad_consulta_sha256": huellaValorFiltroConsultaAccesos(
			filtro.FinalidadDeLaConsulta,
		),
	}
}

func huellaValorFiltroConsultaAccesos(valor string) string {
	suma := sha256.Sum256([]byte(valor))
	return hex.EncodeToString(suma[:])
}

func auditoriaConsultaAccesosExacta(
	filtro FiltroConsultaAdministrativaAccesos,
	huella string,
	auditoria vecdomain.AuditEntry,
) bool {
	return ValidarEntradaRegistroAcceso(auditoria) == nil &&
		auditoria.ModuleID == ModuloAuditoriaConsultaAccesos &&
		auditoria.Action == AccionAuditoriaConsultaAccesos &&
		auditoria.Purpose == filtro.FinalidadDeLaConsulta &&
		auditoria.SubjectRef == "consulta-accesos:sha256:"+huella &&
		auditoria.ObjectVersion == int(filtro.Version) &&
		auditoria.Result == "permitido" &&
		auditoria.ActorProfile != "" &&
		len(auditoria.ActorRoles) > 0 &&
		auditoria.RepresentedSubjectID == "" &&
		auditoria.AuthMethod != "" &&
		auditoria.AuthAssurance == vecdomain.AuthAssuranceHigh &&
		auditoria.AuthorizationRef != "" &&
		auditoria.ExpedienteRef == "" &&
		auditoria.DocumentRef == "" &&
		auditoria.RuleRef == "" &&
		auditoria.Reason == "" &&
		auditoria.BeforeHash == "" &&
		auditoria.AfterHash == "" &&
		len(auditoria.Metadata) == 0
}

func registroCumpleFiltroConsultaAccesos(
	registro ResumenAccesoAdministrativo,
	filtro FiltroConsultaAdministrativaAccesos,
) bool {
	return registro.VersionEsquema == filtro.Version &&
		!registro.OcurridoEn.Before(filtro.DesdeInclusive) &&
		registro.OcurridoEn.Before(filtro.HastaExclusive) &&
		(filtro.ActorSeudonimizado == "" ||
			registro.ActorSeudonimizado == filtro.ActorSeudonimizado) &&
		(filtro.ModuloID == "" || registro.ModuloID == filtro.ModuloID) &&
		(filtro.Accion == "" || registro.Accion == filtro.Accion) &&
		(filtro.FinalidadAcceso == "" ||
			registro.Finalidad == filtro.FinalidadAcceso) &&
		(filtro.RecursoRef == "" ||
			registro.RecursoRef == filtro.RecursoRef) &&
		(filtro.ExpedienteRef == "" ||
			registro.ExpedienteRef == filtro.ExpedienteRef) &&
		(filtro.Resultado == "" || registro.Resultado == filtro.Resultado) &&
		(filtro.VersionObjeto == 0 ||
			registro.VersionObjeto == filtro.VersionObjeto)
}

func esHexadecimalMinusculo(valor string) bool {
	if valor == "" {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') &&
			(caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}
