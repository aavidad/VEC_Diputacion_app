package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	rolPropietarioContextoContratacionTemporalDesarrollo     = "vec_contexto_actor_v1_propietario"
	rolPropietarioAutorizacionContratacionTemporalDesarrollo = "vec_autorizacion_propietario"
	rolProyectorMotivosContratacionTemporalDesarrollo        = "vec_autorizacion_motivos_proyector"
)

func publicarAutoridadPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if ctx == nil || pool == nil || soporte == nil ||
		soporte.contexto.Resultado.Validar() != nil ||
		soporte.instantanea.Validar() != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return err
	}
	if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := publicarMotivosPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func publicarContextoPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	resultado := soporte.contexto.Resultado
	actor := resultado.Contexto
	instantanea := actor.Instantanea
	manifiesto, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(
		resultado.ManifiestoProcedenciaCanonico,
	)
	if err != nil || manifiesto.ValidarParaContexto(actor) != nil ||
		len(instantanea.Vinculos) != 0 || len(manifiesto.Vinculos) != 0 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	procedencia := manifiesto.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1
	if manifiesto.Persona.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia ||
		manifiesto.Perfil.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia ||
		manifiesto.Contexto.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	base := soporte.principalID + "\x00" + soporte.certificadoSHA256
	operacionRef := referenciaAltaContratacionTemporalDesarrollo(
		"oca_", base+"\x00registro-contexto",
	)
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+
		rolPropietarioContextoContratacionTemporalDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err = tx.Exec(ctx, `
		SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		"vec:ct:desarrollo:autoridad:"+procedencia.ProcedenciaRef,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	consultas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_contexto_actor_v1.procedencias
		  (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
		 SELECT $1,$2,$3,$4 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.procedencias
		   WHERE procedencia_ref=$1 AND procedencia_version=$2)`,
			[]any{procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad)}},
		{`INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
		  (cuenta_ref,version,procedencia_ref,procedencia_version,
		   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
		   WHERE cuenta_ref=$1 AND version=$2)`,
			[]any{instantanea.CuentaRef, instantanea.CuentaVersion,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual(cuenta_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual WHERE cuenta_ref=$1)`,
			[]any{instantanea.CuentaRef, instantanea.CuentaVersion}},
		{`INSERT INTO vec_contexto_actor_v1.persona_versiones
		  (persona_ref,version,procedencia_ref,procedencia_version,
		   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.persona_versiones
		   WHERE persona_ref=$1 AND version=$2)`,
			[]any{actor.PersonaRef, instantanea.PersonaVersion,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.persona_actual(persona_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref=$1)`,
			[]any{actor.PersonaRef, instantanea.PersonaVersion}},
		{`INSERT INTO vec_contexto_actor_v1.perfil_versiones
		  (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,
		   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones
		   WHERE perfil_ref=$1 AND version=$2)`,
			[]any{actor.PerfilActivoRef, instantanea.PerfilVersion, actor.PersonaRef,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.perfil_actual(perfil_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=$1)`,
			[]any{actor.PerfilActivoRef, instantanea.PerfilVersion}},
		{`INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
		  (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,
		   procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
		   estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones
		   WHERE vinculo_ref=$1 AND version=$2)`,
			[]any{instantanea.VinculoRef, instantanea.VinculoVersion,
				instantanea.CuentaRef, actor.PerfilActivoRef, actor.PersonaRef,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual(vinculo_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=$1)`,
			[]any{instantanea.VinculoRef, instantanea.VinculoVersion}},
		{`INSERT INTO vec_contexto_actor_v1.registros_contexto
		  (operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,
		   solicitado_en,resuelto_en,representacion_canonica,huella_sha256,
		   manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,
		   autoridad_efectiva)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.registros_contexto
		   WHERE operacion_ref=$1 OR registro_contexto_ref=$2)`,
			[]any{operacionRef, resultado.RegistroContextoRef, instantanea.CuentaRef,
				actor.PerfilActivoRef, string(actor.Principal.AuthMethod),
				string(actor.Principal.AuthAssurance), resultado.ResueltoEnAutoritativo,
				resultado.ResueltoEnAutoritativo, resultado.RepresentacionCanonica,
				resultado.HuellaSHA256, resultado.ManifiestoProcedenciaCanonico,
				resultado.ManifiestoProcedenciaHuellaSHA256,
				string(resultado.AutoridadEfectiva)}},
	}
	for _, consulta := range consultas {
		if _, err = tx.Exec(ctx, consulta.sql, consulta.args...); err != nil {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
	}
	var coincide bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.registros_contexto
		  WHERE operacion_ref=$1 AND registro_contexto_ref=$2 AND cuenta_ref=$3
		    AND perfil_ref=$4 AND metodo=$5 AND garantia=$6
		    AND solicitado_en=$7 AND resuelto_en=$7
		    AND representacion_canonica=$8 AND huella_sha256=$9
		    AND manifiesto_procedencia_canonico=$10
			 AND manifiesto_procedencia_huella_sha256=$11
			 AND autoridad_efectiva=$12)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.procedencias
		  WHERE procedencia_ref=$19 AND procedencia_version=$20
		    AND procedencia_huella_sha256=$21 AND procedencia_autoridad=$22)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
		  WHERE cuenta_ref=$3 AND version=$13 AND procedencia_ref=$19
		    AND procedencia_version=$20 AND procedencia_huella_sha256=$21
		    AND procedencia_autoridad=$22 AND estado=$23
		    AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.persona_versiones
		  WHERE persona_ref=$14 AND version=$15 AND procedencia_ref=$19
		    AND procedencia_version=$20 AND procedencia_huella_sha256=$21
		    AND procedencia_autoridad=$22 AND estado=$23
		    AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones
		  WHERE perfil_ref=$4 AND version=$16 AND persona_ref=$14
		    AND procedencia_ref=$19 AND procedencia_version=$20
		    AND procedencia_huella_sha256=$21 AND procedencia_autoridad=$22
		    AND estado=$23 AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones
		  WHERE vinculo_ref=$17 AND version=$18 AND cuenta_ref=$3
		    AND perfil_ref=$4 AND persona_ref=$14 AND procedencia_ref=$19
		    AND procedencia_version=$20 AND procedencia_huella_sha256=$21
		    AND procedencia_autoridad=$22 AND estado=$23
		    AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual
		  WHERE cuenta_ref=$3 AND version=$13)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.persona_actual
		  WHERE persona_ref=$14 AND version=$15)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.perfil_actual
		  WHERE perfil_ref=$4 AND version=$16)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual
		  WHERE vinculo_ref=$17 AND version=$18)`,
		operacionRef, resultado.RegistroContextoRef, instantanea.CuentaRef,
		actor.PerfilActivoRef, string(actor.Principal.AuthMethod),
		string(actor.Principal.AuthAssurance), resultado.ResueltoEnAutoritativo,
		resultado.RepresentacionCanonica, resultado.HuellaSHA256,
		resultado.ManifiestoProcedenciaCanonico,
		resultado.ManifiestoProcedenciaHuellaSHA256,
		string(resultado.AutoridadEfectiva), instantanea.CuentaVersion,
		actor.PersonaRef, instantanea.PersonaVersion, instantanea.PerfilVersion,
		instantanea.VinculoRef, instantanea.VinculoVersion,
		procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
		procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
		string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta,
	).Scan(&coincide)
	if err != nil || !coincide {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	instantanea := soporte.instantanea
	datosVinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	documentoRol, err := json.Marshal(instantanea.VersionRol)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	documentoControl, err := json.Marshal(instantanea.ControlVigenciaVersionRol)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	documentoAsignacion, err := json.Marshal(instantanea.AsignacionPerfil)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	huellaRol, err := instantanea.VersionRol.HuellaSHA256()
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	huellaControl, err := instantanea.ControlVigenciaVersionRol.HuellaSHA256()
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	huellaAsignacion, err := instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	rolRef := instantanea.VersionRol.Referencia()
	asignacionRef := instantanea.AsignacionPerfil.Referencia()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+
		rolPropietarioAutorizacionContratacionTemporalDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err = tx.Exec(ctx, `
		SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		"vec:ct:desarrollo:autorizacion:"+datosVinculo.PerfilActivoRef,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	consultas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_autorizacion.version_rol
		  (version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
		 SELECT $1,$2,$3,$4,$5,$6::jsonb WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref=$1)`,
			[]any{rolRef, instantanea.VersionRol.RolID, instantanea.VersionRol.Version,
				huellaRol, instantanea.VersionRol.PublicadaEn, documentoRol}},
		{`INSERT INTO vec_autorizacion.control_vigencia_version_rol
		  (version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
		 SELECT $1,$2,$3,$4,$5,$6::jsonb WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol
		   WHERE version_rol_ref=$1 AND revision=$2)`,
			[]any{rolRef, instantanea.ControlVigenciaVersionRol.Revision,
				string(instantanea.ControlVigenciaVersionRol.Estado), huellaControl,
				instantanea.ControlVigenciaVersionRol.ActualizadoEn, documentoControl}},
		{`INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
		  (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
		 SELECT $1,$2,$3,$4,$5 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual
		   WHERE version_rol_ref=$1)`,
			[]any{rolRef, instantanea.ControlVigenciaVersionRol.Revision,
				instantanea.ControlVigenciaVersionRol.ActualizadoEn,
				instantanea.ControlVigenciaVersionRol.ActualizadoPor,
				"acto:ct:desarrollo:control-rol:v1"}},
		{`INSERT INTO vec_autorizacion.asignacion_perfil
		  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
		   version_rol_ref,huella_sha256,emitida_en,documento)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=$1)`,
			[]any{asignacionRef, instantanea.AsignacionPerfil.AsignacionID,
				instantanea.AsignacionPerfil.Version, datosVinculo.PerfilActivoRef,
				datosVinculo.PrincipalID, rolRef, huellaAsignacion,
				instantanea.AsignacionPerfil.EmitidaEn, documentoAsignacion}},
		{`INSERT INTO vec_autorizacion.asignacion_perfil_actual
		  (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
		 SELECT $1,$2,$3,$4,$5 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual
		   WHERE perfil_activo_ref=$1)`,
			[]any{datosVinculo.PerfilActivoRef, asignacionRef,
				instantanea.AsignacionPerfil.EmitidaEn,
				instantanea.AsignacionPerfil.EmitidaPor,
				"acto:ct:desarrollo:asignacion:v1"}},
		{`INSERT INTO vec_autorizacion.sesion_autenticacion_v1
		  (sesion_ref,autenticacion_ref,autenticacion_huella_sha256,asercion_ref,
		   cuenta_ref,cuenta_ordinaria_ref,cuenta_privilegiada,superficie,
		   metodo_observado,garantia_observada,politica_garantia_ref,
		   politica_garantia_huella_sha256,autenticacion_verificada_en,sesion_emitida_en)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.sesion_autenticacion_v1 WHERE sesion_ref=$1)`,
			[]any{datosVinculo.SesionRef, datosVinculo.AutenticacionRef,
				datosVinculo.AutenticacionHuellaSHA256, datosVinculo.AsercionRef,
				datosVinculo.CuentaRef, datosVinculo.CuentaOrdinariaRef,
				datosVinculo.CuentaPrivilegiada, string(datosVinculo.Superficie),
				string(datosVinculo.MetodoObservado), string(datosVinculo.GarantiaObservada),
				datosVinculo.PoliticaGarantiaRef,
				datosVinculo.PoliticaGarantiaHuellaSHA256,
				datosVinculo.AutenticacionVerificadaEn, datosVinculo.SesionEmitidaEn}},
		{`INSERT INTO vec_autorizacion.control_sesion_v1
		  (control_sesion_ref,revision,sesion_ref,estado,huella_sha256,
		   sesion_revalidada_en,sesion_valida_hasta)
		 SELECT $1,$2,$3,'activa',$4,$5,$6 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.control_sesion_v1
		   WHERE control_sesion_ref=$1 AND revision=$2)`,
			[]any{datosVinculo.ControlSesionRef, datosVinculo.ControlSesionRevision,
				datosVinculo.SesionRef, datosVinculo.ControlSesionHuellaSHA256,
				datosVinculo.SesionRevalidadaEn, datosVinculo.SesionValidaHasta}},
		{`INSERT INTO vec_autorizacion.control_sesion_actual_v1
		  (sesion_ref,control_sesion_ref,revision,actualizada_en,acto_ref)
		 SELECT $1,$2,$3,$4,$5 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.control_sesion_actual_v1 WHERE sesion_ref=$1)`,
			[]any{datosVinculo.SesionRef, datosVinculo.ControlSesionRef,
				datosVinculo.ControlSesionRevision, datosVinculo.SesionRevalidadaEn,
				"acto:ct:desarrollo:sesion:v1"}},
	}
	for _, consulta := range consultas {
		if _, err = tx.Exec(ctx, consulta.sql, consulta.args...); err != nil {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
	}
	var coincide bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1 FROM vec_autorizacion.version_rol
		  WHERE version_rol_ref=$1 AND rol_id=$16 AND version=$17
		    AND huella_sha256=$2 AND publicada_en=$18 AND documento=$3::jsonb)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol
		  WHERE version_rol_ref=$1 AND revision=$4 AND estado=$19
		    AND huella_sha256=$20 AND actualizado_en=$21
		    AND documento=$22::jsonb)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual
		  WHERE version_rol_ref=$1 AND revision=$4 AND actualizada_en=$21
		    AND actualizada_por=$23
		    AND acto_ref='acto:ct:desarrollo:control-rol:v1')
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.asignacion_perfil
		  WHERE asignacion_ref=$5 AND asignacion_id=$24 AND version=$25
		    AND perfil_activo_ref=$6 AND principal_id=$7
		    AND version_rol_ref=$1 AND huella_sha256=$8
		    AND emitida_en=$26 AND documento=$9::jsonb)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual
		  WHERE perfil_activo_ref=$6 AND asignacion_ref=$5
		    AND actualizada_en=$26 AND actualizada_por=$27
		    AND acto_ref='acto:ct:desarrollo:asignacion:v1')
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.sesion_autenticacion_v1
		  WHERE sesion_ref=$10 AND autenticacion_ref=$11
		    AND autenticacion_huella_sha256=$12 AND asercion_ref=$28
		    AND cuenta_ref=$13 AND cuenta_ordinaria_ref=$29
		    AND cuenta_privilegiada=$30 AND superficie=$31
		    AND metodo_observado=$32 AND garantia_observada=$33
		    AND politica_garantia_ref=$34
		    AND politica_garantia_huella_sha256=$35
		    AND autenticacion_verificada_en=$36 AND sesion_emitida_en=$37)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.control_sesion_v1
		  WHERE control_sesion_ref=$14 AND revision=$15 AND sesion_ref=$10
		    AND estado='activa' AND huella_sha256=$38
		    AND sesion_revalidada_en=$39 AND sesion_valida_hasta=$40)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.control_sesion_actual_v1
		  WHERE sesion_ref=$10 AND control_sesion_ref=$14 AND revision=$15
		    AND actualizada_en=$39
		    AND acto_ref='acto:ct:desarrollo:sesion:v1')`,
		rolRef, huellaRol, documentoRol,
		instantanea.ControlVigenciaVersionRol.Revision,
		asignacionRef, datosVinculo.PerfilActivoRef, datosVinculo.PrincipalID,
		huellaAsignacion, documentoAsignacion, datosVinculo.SesionRef,
		datosVinculo.AutenticacionRef, datosVinculo.AutenticacionHuellaSHA256,
		datosVinculo.CuentaRef, datosVinculo.ControlSesionRef,
		datosVinculo.ControlSesionRevision,
		instantanea.VersionRol.RolID, instantanea.VersionRol.Version,
		instantanea.VersionRol.PublicadaEn,
		string(instantanea.ControlVigenciaVersionRol.Estado), huellaControl,
		instantanea.ControlVigenciaVersionRol.ActualizadoEn, documentoControl,
		instantanea.ControlVigenciaVersionRol.ActualizadoPor,
		instantanea.AsignacionPerfil.AsignacionID,
		instantanea.AsignacionPerfil.Version,
		instantanea.AsignacionPerfil.EmitidaEn,
		instantanea.AsignacionPerfil.EmitidaPor,
		datosVinculo.AsercionRef, datosVinculo.CuentaOrdinariaRef,
		datosVinculo.CuentaPrivilegiada, string(datosVinculo.Superficie),
		string(datosVinculo.MetodoObservado), string(datosVinculo.GarantiaObservada),
		datosVinculo.PoliticaGarantiaRef,
		datosVinculo.PoliticaGarantiaHuellaSHA256,
		datosVinculo.AutenticacionVerificadaEn, datosVinculo.SesionEmitidaEn,
		datosVinculo.ControlSesionHuellaSHA256,
		datosVinculo.SesionRevalidadaEn, datosVinculo.SesionValidaHasta,
	).Scan(&coincide)
	if err != nil || !coincide || tx.Commit(ctx) != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

type entradaMotivoPostgreSQLContratacionTemporalDesarrollo struct {
	Clave        string  `json:"clave"`
	VigenteDesde string  `json:"vigente_desde"`
	VigenteHasta *string `json:"vigente_hasta"`
}

func publicarMotivosPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	motivos := []dominiovec.ReferenciaEntradaCatalogo{
		soporte.motivo,
		soporte.motivoPropuestaCobertura,
		soporte.motivoDecisionCobertura,
		soporte.motivoRectificacionCobertura,
		soporte.motivoResultadoCobertura,
	}
	porCatalogo := make(map[string][]dominiovec.ReferenciaEntradaCatalogo)
	for _, motivo := range motivos {
		if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
		clave := motivo.CatalogoID + "\x00" + motivo.CatalogoHuellaSHA256
		porCatalogo[clave] = append(porCatalogo[clave], motivo)
	}
	claves := make([]string, 0, len(porCatalogo))
	for clave := range porCatalogo {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(
		soporte.reloj.Ahora(),
	)
	if !vigente {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	for _, clave := range claves {
		if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(
			ctx, pool, porCatalogo[clave], desde,
		); err != nil {
			return err
		}
	}
	return nil
}

func publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	motivos []dominiovec.ReferenciaEntradaCatalogo,
	publicadoEn time.Time,
) error {
	if len(motivos) == 0 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	sort.Slice(motivos, func(i, j int) bool {
		return motivos[i].EntradaClave < motivos[j].EntradaClave
	})
	primero := motivos[0]
	entradas := make([]entradaMotivoPostgreSQLContratacionTemporalDesarrollo, len(motivos))
	for indice, motivo := range motivos {
		if motivo.CatalogoID != primero.CatalogoID ||
			motivo.CatalogoVersion != primero.CatalogoVersion ||
			motivo.CatalogoHuellaSHA256 != primero.CatalogoHuellaSHA256 {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
		entradas[indice] = entradaMotivoPostgreSQLContratacionTemporalDesarrollo{
			Clave:        motivo.EntradaClave,
			VigenteDesde: publicadoEn.Format("2006-01-02T15:04:05.000000Z"),
		}
	}
	contenido, err := json.Marshal(entradas)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	eventoRef := referenciaAltaContratacionTemporalDesarrollo(
		"evento_", "catalogo-motivos\x00"+primero.CatalogoID,
	)
	huellaEvento := huellaAltaContratacionTemporalDesarrollo(
		"catalogo-motivos\x00" + primero.CatalogoID,
	)
	var secuencia int64
	txConsulta, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer txConsulta.Rollback(context.Background())
	if _, err = txConsulta.Exec(ctx, `SET LOCAL ROLE `+
		rolPropietarioAutorizacionContratacionTemporalDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	err = txConsulta.QueryRow(ctx, `
		SELECT secuencia_origen
		  FROM vec_autorizacion.motivo_v2_catalogo_publicado
		 WHERE catalogo_id=$1 AND catalogo_version=$2`,
		primero.CatalogoID, primero.CatalogoVersion,
	).Scan(&secuencia)
	if errors.Is(err, pgx.ErrNoRows) {
		err = txConsulta.QueryRow(ctx, `
			SELECT ultima_secuencia+1
			  FROM vec_autorizacion.motivo_v2_checkpoint_origen
			 WHERE control_id=true FOR UPDATE`).Scan(&secuencia)
	}
	if err != nil || txConsulta.Commit(ctx) != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+
		rolProyectorMotivosContratacionTemporalDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	var publicada bool
	err = tx.QueryRow(ctx, `
		SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
		 $1,$2,$3,$4,$5,$6,$7,$8::jsonb)`,
		eventoRef, secuencia, huellaEvento, primero.CatalogoID,
		primero.CatalogoVersion, primero.CatalogoHuellaSHA256,
		publicadoEn, contenido,
	).Scan(&publicada)
	if err != nil || !publicada || tx.Commit(ctx) != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}
