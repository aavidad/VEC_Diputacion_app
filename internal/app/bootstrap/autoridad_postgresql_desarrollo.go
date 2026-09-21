package bootstrap

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const rolPropietarioAutorizacionPostgreSQLDesarrollo = "vec_autorizacion_propietario"

// autoridadPostgreSQLDesarrollo materializa una instantánea contra la misma
// autoridad VEC. El vínculo lo aporta el módulo tras revalidarlo en su frontera.
type autoridadPostgreSQLDesarrollo struct {
	pool           *pgxpool.Pool
	vinculo        dominiovec.VinculoAutenticacionActorV2
	prefijoBloqueo string
	actoControlRol string
	actoAsignacion string
	actoSesion     string
}

func (a autoridadPostgreSQLDesarrollo) validaConfiguracion() bool {
	return a.pool != nil && a.prefijoBloqueo != "" && a.actoControlRol != "" &&
		a.actoAsignacion != "" && a.actoSesion != ""
}

func clonarInstantaneaAutorizacionPostgreSQLDesarrollo(
	instantanea dominiovec.InstantaneaAutorizacion,
) dominiovec.InstantaneaAutorizacion {
	copia := instantanea
	copia.VersionRol.Concesiones = append([]dominiovec.ConcesionRol(nil), instantanea.VersionRol.Concesiones...)
	for i := range copia.VersionRol.Concesiones {
		copia.VersionRol.Concesiones[i].Finalidades = append([]string(nil), instantanea.VersionRol.Concesiones[i].Finalidades...)
	}
	copia.AsignacionPerfil.Ambitos = append([]dominiovec.AmbitoPerfil(nil), instantanea.AsignacionPerfil.Ambitos...)
	for i := range copia.AsignacionPerfil.Ambitos {
		copia.AsignacionPerfil.Ambitos[i].Valores = append([]string(nil), instantanea.AsignacionPerfil.Ambitos[i].Valores...)
	}
	copia.Politicas = append([]dominiovec.PoliticaRestrictiva(nil), instantanea.Politicas...)
	for i := range copia.Politicas {
		origen := instantanea.Politicas[i]
		politica := &copia.Politicas[i]
		politica.Acciones = append([]string(nil), origen.Acciones...)
		politica.Modulos = append([]string(nil), origen.Modulos...)
		politica.TiposRecurso = append([]string(nil), origen.TiposRecurso...)
		politica.FinalidadesPermitidas = append([]string(nil), origen.FinalidadesPermitidas...)
		politica.Restricciones = append([]dominiovec.RestriccionAtributoRecurso(nil), origen.Restricciones...)
		for j := range politica.Restricciones {
			politica.Restricciones[j].ValoresPermitidos = append([]string(nil), origen.Restricciones[j].ValoresPermitidos...)
		}
	}
	return copia
}

type asignacionActualPostgreSQLDesarrollo struct {
	referencia    string
	identificador string
	version       int64
	perfilRef     string
	principalID   string
	versionRolRef string
	huella        string
}

func (a *autoridadPostgreSQLDesarrollo) PrepararInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	return a.prepararInstantanea(ctx, instantanea, false)
}

func (a *autoridadPostgreSQLDesarrollo) PublicarInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	return a.publicarInstantanea(ctx, instantanea)
}

func (a autoridadPostgreSQLDesarrollo) prepararInstantanea(
	ctx context.Context,
	solicitada dominiovec.InstantaneaAutorizacion,
	permitirInicial bool,
) (dominiovec.InstantaneaAutorizacion, error) {
	vacia := dominiovec.InstantaneaAutorizacion{}
	if !a.validaConfiguracion() || ctx == nil ||
		ctx.Err() != nil || solicitada.Validar() != nil || len(solicitada.Politicas) != 0 {
		return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+
		rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
		return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	perfilRef := solicitada.AsignacionPerfil.PerfilActivoRef
	if _, err = tx.Exec(ctx, `
		SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		a.prefijoBloqueo+perfilRef,
	); err != nil {
		return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	actual, encontrada, err := leerAsignacionActualPostgreSQLDesarrollo(
		ctx, tx, perfilRef,
	)
	if err != nil || (!encontrada && !permitirInicial) {
		return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	preparada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(solicitada)
	if encontrada {
		if actual.perfilRef != perfilRef ||
			actual.principalID != solicitada.AsignacionPerfil.PrincipalID ||
			actual.version <= 0 || actual.version == int64(1<<63-1) {
			return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
		preparada.AsignacionPerfil.AsignacionID = actual.identificador
		preparada.AsignacionPerfil.Version = int(actual.version)
		huellaActual, errHuella := preparada.AsignacionPerfil.HuellaSHA256()
		if errHuella != nil || preparada.AsignacionPerfil.Referencia() != actual.referencia ||
			huellaActual != actual.huella || preparada.VersionRol.Referencia() != actual.versionRolRef {
			preparada.AsignacionPerfil.Version = int(actual.version + 1)
		}
	} else if preparada.AsignacionPerfil.Version != 1 {
		return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if preparada.Validar() != nil || tx.Commit(ctx) != nil {
		return vacia, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return preparada, nil
}

func leerAsignacionActualPostgreSQLDesarrollo(
	ctx context.Context,
	consultador interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	perfilRef string,
) (asignacionActualPostgreSQLDesarrollo, bool, error) {
	var actual asignacionActualPostgreSQLDesarrollo
	err := consultador.QueryRow(ctx, `
		SELECT asignacion.asignacion_ref, asignacion.asignacion_id,
		       asignacion.version, asignacion.perfil_activo_ref,
		       asignacion.principal_id, asignacion.version_rol_ref,
		       asignacion.huella_sha256
		  FROM vec_autorizacion.asignacion_perfil_actual AS vigente
		  JOIN vec_autorizacion.asignacion_perfil AS asignacion
		    ON asignacion.perfil_activo_ref=vigente.perfil_activo_ref
		   AND asignacion.asignacion_ref=vigente.asignacion_ref
		 WHERE vigente.perfil_activo_ref=$1
		 FOR UPDATE OF vigente`, perfilRef).Scan(
		&actual.referencia, &actual.identificador, &actual.version,
		&actual.perfilRef, &actual.principalID, &actual.versionRolRef,
		&actual.huella,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return asignacionActualPostgreSQLDesarrollo{}, false, nil
	}
	if err != nil {
		return asignacionActualPostgreSQLDesarrollo{}, false, err
	}
	return actual, true, nil
}

func (a autoridadPostgreSQLDesarrollo) publicarInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	if !a.validaConfiguracion() || ctx == nil ||
		ctx.Err() != nil || instantanea.Validar() != nil || len(instantanea.Politicas) != 0 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	pool := a.pool
	datosVinculo, err := a.vinculo.Datos()
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if instantanea.AsignacionPerfil.PerfilActivoRef != datosVinculo.PerfilActivoRef ||
		instantanea.AsignacionPerfil.PrincipalID != datosVinculo.PrincipalID {
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
		rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err = tx.Exec(ctx, `
		SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		a.prefijoBloqueo+datosVinculo.PerfilActivoRef,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	actual, encontrada, err := leerAsignacionActualPostgreSQLDesarrollo(
		ctx, tx, datosVinculo.PerfilActivoRef,
	)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if encontrada {
		yaPublicada := actual.referencia == asignacionRef && actual.huella == huellaAsignacion
		siguienteExacta := actual.identificador == instantanea.AsignacionPerfil.AsignacionID &&
			actual.perfilRef == datosVinculo.PerfilActivoRef &&
			actual.principalID == datosVinculo.PrincipalID &&
			actual.version > 0 && actual.version < int64(1<<63-1) &&
			instantanea.AsignacionPerfil.Version == int(actual.version+1)
		if !yaPublicada && !siguienteExacta {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
	} else if instantanea.AsignacionPerfil.Version != 1 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	var revisionCatalogo uint64
	var huellaCatalogo string
	if err = tx.QueryRow(ctx, `
		SELECT revision, huella_sha256
		  FROM vec_autorizacion.control_catalogo_politicas
		 WHERE control_id=true
		 FOR UPDATE`).Scan(&revisionCatalogo, &huellaCatalogo); err != nil ||
		revisionCatalogo != instantanea.RevisionCatalogoPoliticas ||
		huellaCatalogo != instantanea.CatalogoPoliticasHuellaSHA256 {
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
				a.actoControlRol}},
		{`INSERT INTO vec_autorizacion.asignacion_perfil
		  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
		   version_rol_ref,huella_sha256,emitida_en,documento)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb WHERE NOT EXISTS (
		  SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=$1)`,
			[]any{asignacionRef, instantanea.AsignacionPerfil.AsignacionID,
				instantanea.AsignacionPerfil.Version, datosVinculo.PerfilActivoRef,
				datosVinculo.PrincipalID, rolRef, huellaAsignacion,
				instantanea.AsignacionPerfil.EmitidaEn, documentoAsignacion}},
		{`INSERT INTO vec_autorizacion.asignacion_perfil_actual AS vigente
		  (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (perfil_activo_ref) DO UPDATE SET
		  asignacion_ref=EXCLUDED.asignacion_ref,
		  actualizada_en=EXCLUDED.actualizada_en,
		  actualizada_por=EXCLUDED.actualizada_por,
		  acto_ref=EXCLUDED.acto_ref
		 WHERE vigente.asignacion_ref IS DISTINCT FROM EXCLUDED.asignacion_ref`,
			[]any{datosVinculo.PerfilActivoRef, asignacionRef,
				instantanea.AsignacionPerfil.EmitidaEn,
				instantanea.AsignacionPerfil.EmitidaPor,
				a.actoAsignacion}},
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
				a.actoSesion}},
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
			    AND acto_ref=$41)
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
			    AND acto_ref=$42)
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
			    AND acto_ref=$43)`,
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
		a.actoControlRol, a.actoAsignacion, a.actoSesion,
	).Scan(&coincide)
	if err != nil || !coincide {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if tx.Commit(ctx) != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}
