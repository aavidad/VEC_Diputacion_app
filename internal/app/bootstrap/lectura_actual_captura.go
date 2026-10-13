package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type capturaLecturaActual struct {
	contexto, vinculo []byte
	orden             json.RawMessage
	hasta             time.Time
}

// Comparte la evidencia de sesión y de permiso actual de las lecturas RRHH.
// Cada propietario añade exclusivamente sus filtros y coteja el contrato en SQL.
func componerCapturaLecturaActual(ctx context.Context, vinculo vecdomain.VinculoAutenticacionActorV2,
	resultado vecdomain.ResultadoContextoActorRegistradoV2, evaluacion vecdomain.ResultadoLecturaActualAutorizacionV3,
	accion, modulo, finalidad, motivo, recurso string, adicionales map[string]any,
) (capturaLecturaActual, error) {
	vacia := capturaLecturaActual{}
	datos, err := vinculo.Datos()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if err != nil || ctx == nil || ctx.Err() != nil || !evaluacion.Permitido ||
		!vinculo.VigenteEn(ahora, resultado) || !evaluacion.ValidaHasta.After(ahora) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, err
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, err
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return vacia, err
	}
	contexto := bytes.Clone(resultado.RepresentacionCanonica)
	if len(contexto) == 0 {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	vinculoCanonico, err := json.Marshal(vinculo)
	if err != nil {
		return vacia, err
	}
	huella := sha256.Sum256(contexto)
	limite := ahora.Add(30 * time.Second)
	if evaluacion.ValidaHasta.Before(limite) {
		limite = evaluacion.ValidaHasta
	}
	if datos.SesionValidaHasta.Before(limite) {
		limite = datos.SesionValidaHasta
	}
	orden := map[string]any{
		"intento_ref":           "lectura_" + hex.EncodeToString(aleatorio[:]),
		"registro_contexto_ref": datos.RegistroContextoRef,
		"contexto_sha256":       hex.EncodeToString(huella[:]), "procedencia_sha256": datos.ManifiestoProcedenciaHuellaSHA256,
		"autenticacion_ref": datos.AutenticacionRef, "sesion_ref": datos.SesionRef,
		"autenticacion_sha256": datos.AutenticacionHuellaSHA256,
		"accion":               accion, "modulo_id": modulo, "recurso_ref": recurso, "finalidad_ref": finalidad,
		"campos_permitidos": append([]string{}, evaluacion.CamposPermitidos...), "resultado": "obtenida", "motivo_ref": motivo,
		"proceso": "vec-server", "canal": "interna_corporativa", "correlacion_ref": correlacionRef,
		"lectura_revision_permisos":  strconv.FormatUint(evaluacion.RevisionPermisos, 10),
		"lectura_instantanea_sha256": evaluacion.HuellaInstantaneaSHA256,
		"asignacion_ref":             evaluacion.AsignacionRef, "asignacion_version": evaluacion.AsignacionVersion,
		"asignacion_huella_sha256": evaluacion.AsignacionHuellaSHA256,
		"version_rol_ref":          evaluacion.VersionRolRef, "version_rol_version": evaluacion.VersionRolVersion,
		"version_rol_huella_sha256":         evaluacion.VersionRolHuellaSHA256,
		"control_version_rol_ref":           evaluacion.ControlVersionRolRef,
		"revision_control_version_rol":      evaluacion.RevisionControlVersionRol,
		"control_version_rol_huella_sha256": evaluacion.ControlVersionRolHuellaSHA256,
		"revision_catalogo_politicas":       evaluacion.RevisionCatalogoPoliticas,
		"catalogo_politicas_huella_sha256":  evaluacion.CatalogoPoliticasHuellaSHA256,
		"politicas_evaluadas":               append([]vecdomain.ReferenciaHuellaPoliticaLecturaActualV3{}, evaluacion.PoliticasEvaluadas...),
		"emitida_en":                        evaluacion.EvaluadaEn, "valida_hasta": limite,
	}
	for k, v := range adicionales {
		if _, existe := orden[k]; existe {
			return vacia, vecdomain.ErrAutorizacionDenegada
		}
		orden[k] = v
	}
	b, err := json.Marshal(orden)
	if err != nil {
		return vacia, vecports.ErrFuenteAutorizacionNoDisponible
	}
	return capturaLecturaActual{contexto: contexto, vinculo: vinculoCanonico, orden: b, hasta: limite}, nil
}
