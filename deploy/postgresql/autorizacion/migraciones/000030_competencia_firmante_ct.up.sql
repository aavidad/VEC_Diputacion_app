\set ON_ERROR_STOP on
-- AUT30: fachada central de competencia del firmante tercero. El material
-- procede de CT170, pero ninguna afirmación del solicitante concede cargo.
-- Primer corte preparatorio: revalida la asignación y el rol actuales bajo
-- bloqueo; la puerta nominal de cargo/perfil publicado permanece cerrada.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:000030', 0));

DO $preimagen$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
       OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
       OR pg_catalog.to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_actual') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion.version_rol') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion.control_vigencia_version_rol') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion.control_vigencia_version_rol_actual') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion.instante_utc_microsegundo_valido(text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_competencia_firmante_ct_v1(text)') IS NOT NULL
    THEN
        RAISE EXCEPTION 'AUT30: preimagen incompatible o migración ya instalada'
            USING ERRCODE = '55000';
    END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.revalidar_competencia_firmante_ct_v1(p_solicitud text)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET row_security = on
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    s jsonb;
    a record;
    r record;
    instante_actual timestamptz(6);
BEGIN
    -- La única llamada admitida procede de CT170 bajo la misma transacción
    -- SERIALIZABLE. El propietario CT no recibe SELECT sobre las tablas.
    IF current_user <> 'vec_autorizacion_propietario'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(session_user,
            'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(session_user,
            'vec_contratacion_temporal_propietario', 'MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'AUT30: consumidor o transacción incompatibles'
            USING ERRCODE = '42501';
    END IF;

    IF p_solicitud IS NULL OR pg_catalog.octet_length(p_solicitud) NOT BETWEEN 2 AND 16384 THEN
        RETURN false;
    END IF;
    s := p_solicitud::jsonb;
    IF pg_catalog.jsonb_typeof(s) IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM pg_catalog.json_each(p_solicitud::json))
          <> (SELECT count(*) FROM pg_catalog.jsonb_each(s))
       OR pg_catalog.jsonb_typeof(s -> 'FirmantePrincipalRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'PerfilFirmanteRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'CargoFirmante') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'OrganizacionRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'UnidadFirmanteRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'PerfilActivoFirmanteRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'AsignacionFirmanteRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'AsignacionFirmanteVersion') IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(s -> 'AsignacionFirmanteHuella') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'VersionRolFirmanteRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'VersionRolFirmanteHuella') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'ControlVigenciaFirmanteRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'ControlVigenciaFirmanteRevision') IS DISTINCT FROM 'number'
       OR pg_catalog.jsonb_typeof(s -> 'ControlVigenciaFirmanteHuella') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'AsignacionVigenteDesde') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'AsignacionVigenteHasta') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'CatalogoRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'CatalogoHuella') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'PasoRef') IS DISTINCT FROM 'string'
       OR pg_catalog.jsonb_typeof(s -> 'PasoOrden') IS DISTINCT FROM 'number'
       OR (s ->> 'AsignacionFirmanteHuella') !~ '^[0-9a-f]{64}$'
       OR (s ->> 'VersionRolFirmanteHuella') !~ '^[0-9a-f]{64}$'
       OR (s ->> 'ControlVigenciaFirmanteHuella') !~ '^[0-9a-f]{64}$'
       OR (s ->> 'CatalogoHuella') !~ '^[0-9a-f]{64}$'
       OR (s ->> 'AsignacionFirmanteVersion') !~ '^[1-9][0-9]{0,15}$'
       OR (s ->> 'ControlVigenciaFirmanteRevision') !~ '^[1-9][0-9]{0,15}$'
       OR (s ->> 'PasoOrden') !~ '^[1-9][0-9]{0,8}$'
       OR vec_autorizacion.instante_utc_microsegundo_valido(
            s ->> 'AsignacionVigenteDesde') IS NOT TRUE
       OR vec_autorizacion.instante_utc_microsegundo_valido(
            s ->> 'AsignacionVigenteHasta') IS NOT TRUE THEN
        RETURN false;
    END IF;

    SELECT actual.asignacion_ref, actual.acto_ref AS acto_actual,
           asignacion.version, asignacion.principal_id,
           asignacion.perfil_activo_ref, asignacion.version_rol_ref,
           asignacion.huella_sha256, asignacion.documento
      INTO a
      FROM vec_autorizacion.asignacion_perfil_actual AS actual
      JOIN vec_autorizacion.asignacion_perfil AS asignacion
        ON asignacion.perfil_activo_ref = actual.perfil_activo_ref
       AND asignacion.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = s ->> 'PerfilActivoFirmanteRef'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR a.asignacion_ref IS DISTINCT FROM s ->> 'AsignacionFirmanteRef'
       OR a.version IS DISTINCT FROM (s ->> 'AsignacionFirmanteVersion')::bigint
       OR a.principal_id IS DISTINCT FROM s ->> 'FirmantePrincipalRef'
       OR a.version_rol_ref IS DISTINCT FROM s ->> 'VersionRolFirmanteRef'
       OR a.huella_sha256 IS DISTINCT FROM s ->> 'AsignacionFirmanteHuella'
       OR a.documento ->> 'estado' IS DISTINCT FROM 'activa'
       OR a.documento ->> 'vigente_desde' IS DISTINCT FROM s ->> 'AsignacionVigenteDesde'
       OR a.documento ->> 'vigente_hasta' IS DISTINCT FROM s ->> 'AsignacionVigenteHasta'
       OR pg_catalog.jsonb_array_length(a.documento -> 'ambitos') <> 2
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(a.documento -> 'ambitos') AS b(valor)
                       WHERE b.valor = pg_catalog.jsonb_build_object(
                           'clave', 'organizacion_ref',
                           'valores', pg_catalog.jsonb_build_array(s ->> 'OrganizacionRef')))
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(a.documento -> 'ambitos') AS b(valor)
                       WHERE b.valor = pg_catalog.jsonb_build_object(
                           'clave', 'unidad_ref',
                           'valores', pg_catalog.jsonb_build_array(s ->> 'UnidadFirmanteRef'))) THEN
        RETURN false;
    END IF;

    SELECT rol.version_rol_ref, rol.huella_sha256 AS rol_huella,
           rol.documento AS rol_documento, control.revision,
           control.estado AS control_estado, control.huella_sha256 AS control_huella
      INTO r
      FROM vec_autorizacion.version_rol AS rol
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = actual.version_rol_ref
       AND control.revision = actual.revision
     WHERE rol.version_rol_ref = a.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR r.version_rol_ref IS DISTINCT FROM s ->> 'ControlVigenciaFirmanteRef'
       OR r.rol_huella IS DISTINCT FROM s ->> 'VersionRolFirmanteHuella'
       OR r.revision IS DISTINCT FROM (s ->> 'ControlVigenciaFirmanteRevision')::numeric
       OR r.control_huella IS DISTINCT FROM s ->> 'ControlVigenciaFirmanteHuella'
       OR r.control_estado IS DISTINCT FROM 'habilitada'
       OR r.rol_documento ->> 'estado' IS DISTINCT FROM 'publicada' THEN
        RETURN false;
    END IF;

    instante_actual := pg_catalog.clock_timestamp();
    IF instante_actual < (a.documento ->> 'vigente_desde')::timestamptz
       OR instante_actual >= (a.documento ->> 'vigente_hasta')::timestamptz THEN
        RETURN false;
    END IF;

    -- Falta la publicación nominal y versionada que enlaza catálogo/paso/
    -- perfil de cargo con persona, asignación, unidad y acto. Una etiqueta,
    -- un rol RBAC, la fecha declarada o un acto enviado por CT no la suplen.
    -- La operación se abre sólo mediante migración posterior y revisión exacta.
    RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_competencia_firmante_ct_v1(text)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_competencia_firmante_ct_v1(text)
    TO vec_contratacion_temporal_propietario;
COMMIT;
