package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrPermisoInternoNoPublicable = errors.New("permiso interno CT no publicable")

// SolicitudPermisoInternoCT es una orden administrativa previa al arranque. Las
// referencias proceden del contexto nominal privado y deben existir en F1.
// OrganizacionRef es el ámbito CT del permiso (la misma referencia que el
// selector nominal de vec-interno, p. ej. «organizacion:…»); la organización
// corporativa «org_…» es la del vínculo corporativo de ContextoActor. Son
// vocabularios distintos y el aprovisionamiento coteja cada uno en su autoridad.
type SolicitudPermisoInternoCT struct {
	CuentaRef, PrincipalRef, PerfilRef, OrganizacionRef string
	OrganizacionCorporativaRef                          string
	PoliticaRef, PoliticaHuellaSHA256                   string
}

const rolPermisoInternoCT = "rrhh_interno_certificado_seguimiento_ct_"

var retiradaMaximaPermisoInternoCT = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

func plantillaPermisoInternoCT(s SolicitudPermisoInternoCT, ahora, hasta time.Time) (core.InstantaneaAutorizacion, error) {
	var vacia core.InstantaneaAutorizacion
	if s.CuentaRef == "" || s.PrincipalRef == "" || s.PerfilRef == "" || s.OrganizacionRef == "" ||
		!organizacionCorporativaValida(s.OrganizacionCorporativaRef) ||
		strings.HasPrefix(s.OrganizacionRef, "org_") ||
		len(s.PoliticaRef) < 26 || len(s.PoliticaHuellaSHA256) != 64 ||
		!hasta.After(ahora) || hasta.After(retiradaMaximaPermisoInternoCT) ||
		ahora.Location() != time.UTC || hasta.Location() != time.UTC {
		return vacia, ErrPermisoInternoNoPublicable
	}
	identificador := sha256.Sum256([]byte(s.PrincipalRef + "\x00" + s.PerfilRef))
	sufijo := hex.EncodeToString(identificador[:8])
	rol := core.VersionRol{
		RolID: rolPermisoInternoCT + sufijo, Version: 1,
		Nombre: "Consulta de seguimiento CT con certificado personal temporal",
		Estado: core.EstadoVersionRolPublicada,
		Concesiones: []core.ConcesionRol{{
			Accion: ct.AccionConsultarDetalleRRHH, ModuloID: ct.ModuloContratacion,
			TipoRecurso:    ct.TipoRecursoExpediente,
			Finalidades:    []string{ct.FinalidadConsultarDetalleRRHH},
			GarantiaMinima: core.AuthAssuranceSubstantial,
			CamposPermitidos: []string{
				"esquema", "alcance", "expediente_ref", "version_expediente",
				"recibo_incorporacion_ref", "seguimiento_ref", "version_seguimiento",
				"estado_clave", "periodo", "registrado_en", "actuaciones",
				"ejercicio_sintetico", "firma_oficial", "eficacia_administrativa",
			},
			// La lectura durable registra el acceso; no se inventa una
			// obligación V3 que el consumidor no declare como soportada.
		}},
		PublicadaPor: "seguridad:aprovisionamiento:permiso-interno-ct", PublicadaEn: ahora,
	}
	asignacion := core.AsignacionPerfil{
		AsignacionID: "asg_ct_interno_" + sufijo, Version: 1,
		PerfilActivoRef: s.PerfilRef, PrincipalID: s.PrincipalRef,
		VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
		Ambitos: []core.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{s.OrganizacionRef}},
			{Clave: "clase_ambito", Valores: []string{string(ct.AmbitoOrganizacionRRHH)}},
			{Clave: "ambito_ref", Valores: []string{s.OrganizacionRef}},
		},
		VigenteDesde: ahora, VigenteHasta: hasta,
		EmitidaPor: "identidad:aprovisionamiento:permiso-interno-ct", EmitidaEn: ahora,
	}
	huellaCatalogo, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return vacia, ErrPermisoInternoNoPublicable
	}
	instantanea := core.InstantaneaAutorizacion{
		VersionRol: rol, AsignacionPerfil: asignacion,
		ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{
			VersionRolRef: rol.Referencia(), Revision: 1,
			Estado:         core.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: rol.PublicadaPor, ActualizadoEn: ahora,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	if instantanea.Validar() != nil {
		return vacia, ErrPermisoInternoNoPublicable
	}
	return instantanea, nil
}

// PublicarPermisoInternoCT publica una vez el perfil nominal; no se llama
// durante operaciones ni por el servidor. La transacción administrativa exige
// F1 y política temporal actuales. Replay exacto sólo verifica, no reescribe.
// organizacionCorporativaValida repite la gramática de ContextoActor 000003
// (organizacion_ref_valida): «org_» y de 16 a 80 caracteres [a-z0-9].
func organizacionCorporativaValida(valor string) bool {
	if len(valor) < 20 || len(valor) > 84 || !strings.HasPrefix(valor, "org_") {
		return false
	}
	for _, c := range valor[4:] {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func PublicarPermisoInternoCT(ctx context.Context, pool *pgxpool.Pool, s SolicitudPermisoInternoCT) error {
	if ctx == nil || pool == nil || ctx.Err() != nil || !organizacionCorporativaValida(s.OrganizacionCorporativaRef) {
		return ErrPermisoInternoNoPublicable
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ErrPermisoInternoNoPublicable
	}
	defer tx.Rollback(context.Background())
	var admin bool
	if err := tx.QueryRow(ctx, `SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname=current_user AND current_user=session_user`).Scan(&admin); err != nil || !admin {
		return ErrPermisoInternoNoPublicable
	}
	_, err = tx.Exec(ctx, `SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`, "vec:ct:permiso-interno:"+s.PerfilRef)
	if err != nil {
		return ErrPermisoInternoNoPublicable
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	var hasta time.Time
	err = tx.QueryRow(ctx, `SELECT retirar_en FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
 WHERE singleton=true AND activa=true AND registrada_en<=$1 AND retirar_en>$1
 AND politica_ref=$2 AND huella_sha256=$3
 AND retirar_en<='2026-11-01 00:00:00+00'::timestamptz`, ahora, s.PoliticaRef, s.PoliticaHuellaSHA256).Scan(&hasta)
	if err != nil || hasta.After(retiradaMaximaPermisoInternoCT) {
		return ErrPermisoInternoNoPublicable
	}
	hasta = hasta.UTC().Truncate(time.Microsecond)
	var vinculaciones int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v ON (v.vinculo_ref,v.version)=(va.vinculo_ref,va.version)
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c ON (c.cuenta_ref,c.version)=(ca.cuenta_ref,ca.version)
 JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
 JOIN vec_contexto_actor_v1.perfil_versiones p ON (p.perfil_ref,p.version)=(pa.perfil_ref,pa.version)
 JOIN vec_contexto_actor_v1.persona_actual xa ON xa.persona_ref=v.persona_ref
 JOIN vec_contexto_actor_v1.persona_versiones x ON (x.persona_ref,x.version)=(xa.persona_ref,xa.version)
 JOIN vec_contexto_actor_v1.vinculo_corporativo_actual vc ON
   vc.cuenta_ref=v.cuenta_ref AND vc.superficie='interna_corporativa' AND vc.uso='consulta_rrhh'
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones cv ON
   (cv.vinculo_corporativo_ref,cv.version)=(vc.vinculo_corporativo_ref,vc.version)
 JOIN vec_contexto_actor_v1.organizacion_actual oa ON oa.organizacion_ref=cv.organizacion_ref
 JOIN vec_contexto_actor_v1.organizacion_versiones ov ON
   (ov.organizacion_ref,ov.version)=(oa.organizacion_ref,oa.version)
 WHERE v.cuenta_ref=$1 AND v.perfil_ref=$2 AND v.persona_ref=$3 AND p.persona_ref=$3
 AND cv.cuenta_ref=v.cuenta_ref AND cv.perfil_ref=v.perfil_ref AND cv.persona_ref=v.persona_ref
 AND cv.vinculo_contexto_ref=v.vinculo_ref AND cv.vinculo_contexto_version=v.version
 AND cv.cuenta_version=ca.version AND cv.perfil_version=pa.version AND cv.persona_version=xa.version
 AND cv.organizacion_ref=$5 AND cv.organizacion_version=oa.version
 AND cv.organizacion_procedencia_ref=ov.procedencia_ref
 AND cv.organizacion_procedencia_version=ov.procedencia_version
 AND cv.organizacion_procedencia_huella_sha256=ov.procedencia_huella_sha256
 AND cv.organizacion_procedencia_autoridad=ov.procedencia_autoridad
 AND cv.procedencia_autoridad='autoridad_maestra_acreditada'
 AND ov.procedencia_autoridad='autoridad_maestra_acreditada'
 AND cv.superficie='interna_corporativa' AND cv.uso='consulta_rrhh'
 AND cv.estado='activo' AND ov.estado='activo'
 AND v.estado='activo' AND c.estado='activo' AND p.estado='activo' AND x.estado='activo'
 AND $4>=v.vigente_desde AND $4<v.vigente_hasta
 AND $4>=c.vigente_desde AND $4<c.vigente_hasta
 AND $4>=p.vigente_desde AND $4<p.vigente_hasta
 AND $4>=x.vigente_desde AND $4<x.vigente_hasta
 AND $4>=cv.vigente_desde AND $4<cv.vigente_hasta
 AND $4>=ov.vigente_desde AND $4<ov.vigente_hasta`, s.CuentaRef, s.PerfilRef, s.PrincipalRef, ahora, s.OrganizacionCorporativaRef).Scan(&vinculaciones)
	if err != nil || vinculaciones != 1 {
		return ErrPermisoInternoNoPublicable
	}
	// El perfil lector es distinto del perfil RRHH de vec-server que ya tiene
	// su concesión alta. Compartir persona no suma permisos entre perfiles.
	var perfilRRHHSeparado bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual aa
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=aa.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON (c.version_rol_ref,c.revision)=(ca.version_rol_ref,ca.revision)
 WHERE aa.perfil_activo_ref<>$1 AND a.principal_id=$2
 AND a.documento->>'estado'='activa' AND r.documento->>'estado'='publicada'
 AND c.estado='habilitada' AND (a.documento->>'vigente_desde')::timestamptz<=$6
 AND (a.documento->>'vigente_hasta')::timestamptz>$6
 AND EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
   WHERE c->>'accion'=$3 AND c->>'modulo_id'=$4 AND c->>'tipo_recurso'=$5
     AND c->>'garantia_minima'='alto'))`, s.PerfilRef, s.PrincipalRef,
		ct.AccionConsultarDetalleRRHH, ct.ModuloContratacion, ct.TipoRecursoExpediente, ahora).Scan(&perfilRRHHSeparado)
	if err != nil || !perfilRRHHSeparado {
		return ErrPermisoInternoNoPublicable
	}
	var actualAsignacion, actualRol, actualControl []byte
	err = tx.QueryRow(ctx, `SELECT a.documento,r.documento,c.documento
 FROM vec_autorizacion.asignacion_perfil_actual aa
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=aa.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON (c.version_rol_ref,c.revision)=(ca.version_rol_ref,ca.revision)
 WHERE aa.perfil_activo_ref=$1`, s.PerfilRef).Scan(&actualAsignacion, &actualRol, &actualControl)
	if err == nil {
		var a core.AsignacionPerfil
		var r core.VersionRol
		var c core.ControlVigenciaVersionRol
		if json.Unmarshal(actualAsignacion, &a) != nil || json.Unmarshal(actualRol, &r) != nil || json.Unmarshal(actualControl, &c) != nil ||
			!perfilInternoPublicadoExacto(s, a, r, c, hasta, ahora) {
			return ErrPermisoInternoNoPublicable
		}
		if tx.Commit(ctx) != nil {
			return ErrPermisoInternoNoPublicable
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ErrPermisoInternoNoPublicable
	}
	i, err := plantillaPermisoInternoCT(s, ahora, hasta)
	if err != nil {
		return err
	}
	if err = insertarPermisoInternoCT(ctx, tx, i); err != nil {
		return ErrPermisoInternoNoPublicable
	}
	if tx.Commit(ctx) != nil {
		return ErrPermisoInternoNoPublicable
	}
	return nil
}

func perfilInternoPublicadoExacto(s SolicitudPermisoInternoCT, a core.AsignacionPerfil, r core.VersionRol, c core.ControlVigenciaVersionRol, hasta, ahora time.Time) bool {
	if a.Validar() != nil || r.Validar() != nil || c.Validar() != nil || a.Version != 1 || r.Version != 1 ||
		a.PerfilActivoRef != s.PerfilRef || a.PrincipalID != s.PrincipalRef ||
		!a.VigenteHasta.Equal(hasta) || !a.VigenteHasta.After(ahora) ||
		c.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		c.VersionRolRef != r.Referencia() || a.VersionRolRef != r.Referencia() {
		return false
	}
	plantilla, err := plantillaPermisoInternoCT(s, a.EmitidaEn, hasta)
	return err == nil && reflect.DeepEqual(a, plantilla.AsignacionPerfil) &&
		reflect.DeepEqual(r, plantilla.VersionRol) && reflect.DeepEqual(c, plantilla.ControlVigenciaVersionRol)
}

func insertarPermisoInternoCT(ctx context.Context, tx pgx.Tx, i core.InstantaneaAutorizacion) error {
	r, c, a := i.VersionRol, i.ControlVigenciaVersionRol, i.AsignacionPerfil
	rolJSON, _ := json.Marshal(r)
	controlJSON, _ := json.Marshal(c)
	asignacionJSON, _ := json.Marshal(a)
	hRol, e1 := r.HuellaSHA256()
	hControl, e2 := c.HuellaSHA256()
	hAsignacion, e3 := a.HuellaSHA256()
	if e1 != nil || e2 != nil || e3 != nil {
		return ErrPermisoInternoNoPublicable
	}
	consultas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento) VALUES($1,$2,$3,$4,$5,$6)`, []any{r.Referencia(), r.RolID, r.Version, hRol, r.PublicadaEn, rolJSON}},
		{`INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento) VALUES($1,$2,$3,$4,$5,$6)`, []any{c.VersionRolRef, c.Revision, string(c.Estado), hControl, c.ActualizadoEn, controlJSON}},
		{`INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref) VALUES($1,$2,$3,$4,$5)`, []any{c.VersionRolRef, c.Revision, c.ActualizadoEn, c.ActualizadoPor, "acto:ct:interno:control:" + r.RolID}},
		{`INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, []any{a.Referencia(), a.AsignacionID, a.Version, a.PerfilActivoRef, a.PrincipalID, a.VersionRolRef, hAsignacion, a.EmitidaEn, asignacionJSON}},
		{`INSERT INTO vec_autorizacion.asignacion_perfil_actual(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref) VALUES($1,$2,$3,$4,$5)`, []any{a.PerfilActivoRef, a.Referencia(), a.EmitidaEn, a.EmitidaPor, "acto:ct:interno:asignacion:" + a.AsignacionID}},
	}
	for _, q := range consultas {
		if _, err := tx.Exec(ctx, q.sql, q.args...); err != nil {
			return fmt.Errorf("%w: insertar gobierno", ErrPermisoInternoNoPublicable)
		}
	}
	return nil
}
