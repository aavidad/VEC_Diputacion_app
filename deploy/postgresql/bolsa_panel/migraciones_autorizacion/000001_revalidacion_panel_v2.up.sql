-- Frontera estrecha instalada bajo el propietario de autorizacion. Revalida
-- una decision V2 ya registrada, el motivo vigente y toda la configuracion
-- mutable. No sustituye la atestacion criptografica de procedencia: el almacen
-- del panel la exige adicionalmente y mantiene EXECUTE cerrado por defecto.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:revalidacion_panel_bolsa_v2:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.materializar_documento_comun_decision_v2(jsonb)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.resolver_motivo_autorizacion_v2_actual(text,integer,text,text)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(jsonb,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_bolsa_panel_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       )
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_panel_bolsa_v2(jsonb,bytea,bytea,text,text,text,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar la frontera V2 del panel';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.revalidar_decision_panel_bolsa_v2(
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_correlacion_ref text,
    p_recurso_ref text,
    p_huella_contexto_sha256 text,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    decision record;
    documento_v2 jsonb;
    documento_comun jsonb;
    motivo jsonb;
    referencia_motivo jsonb;
    verificada_en timestamptz;
    vinculo jsonb;
    asignacion_actual record;
    rol_actual record;
    catalogo_actual record;
    manifiesto_actual jsonb;
    referencias_actuales jsonb;
    referencias_decision jsonb;
    concesiones_coincidentes integer;
BEGIN
    IF p_prueba IS NULL OR jsonb_typeof(p_prueba) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_prueba)) <> 4
       OR NOT (p_prueba ?& ARRAY[
           'esquema_huella', 'decision_ref', 'huella_decision_sha256',
           'verificada_en'
       ])
       OR p_prueba ->> 'esquema_huella' IS DISTINCT FROM
          'vec.autorizacion.decision.reforzada.v2.solicitud-ligada'
       OR jsonb_typeof(p_prueba -> 'decision_ref') <> 'string'
       OR octet_length(p_prueba ->> 'decision_ref') NOT BETWEEN 1 AND 512
       OR (p_prueba ->> 'decision_ref') !~ '^[^*[:space:][:cntrl:]]+$'
       OR jsonb_typeof(p_prueba -> 'huella_decision_sha256') <> 'string'
       OR (p_prueba ->> 'huella_decision_sha256') !~ '^[0-9a-f]{64}$'
       OR jsonb_typeof(p_prueba -> 'verificada_en') <> 'string'
       OR (p_prueba ->> 'verificada_en') !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR p_decision_canonica IS NULL
       OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
       OR p_correlacion_ref !~ '^correlacion_[0-9a-f]{32}$'
       OR p_recurso_ref !~
          '^panel:org_[a-z0-9]{16,80}(:uni_[a-z0-9]{16,80})?$'
       OR p_huella_contexto_sha256 !~ '^[0-9a-f]{64}$'
       OR p_instante IS NULL OR isfinite(p_instante) IS NOT TRUE
       OR encode(sha256(p_decision_canonica), 'hex') IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256' THEN
        RETURN false;
    END IF;

    BEGIN
        documento_v2 := convert_from(p_decision_canonica, 'UTF8')::jsonb;
        documento_comun :=
            vec_autorizacion.materializar_documento_comun_decision_v2(
                documento_v2
            );
        motivo := convert_from(p_motivo_canonico, 'UTF8')::jsonb;
        verificada_en := (p_prueba ->> 'verificada_en')::timestamptz;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception
            OR datetime_field_overflow THEN
            RETURN false;
    END;
    IF documento_comun IS NULL
       OR to_char(verificada_en AT TIME ZONE 'UTC',
                  'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') <>
          p_prueba ->> 'verificada_en'
       OR jsonb_typeof(motivo) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(motivo)) <> 2
       OR NOT (motivo ?& ARRAY['esquema', 'referencia'])
       OR motivo ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR jsonb_typeof(motivo -> 'referencia') <> 'object'
       OR (SELECT count(*)
             FROM jsonb_object_keys(motivo -> 'referencia')) <> 4
       OR NOT ((motivo -> 'referencia') ?& ARRAY[
           'catalogo_id', 'catalogo_version',
           'catalogo_huella_sha256', 'entrada_clave'
       ]) THEN
        RETURN false;
    END IF;
    referencia_motivo := motivo -> 'referencia';
    IF jsonb_typeof(referencia_motivo -> 'catalogo_id') <> 'string'
       OR (referencia_motivo ->> 'catalogo_id') !~
          '^[a-z][a-z0-9._-]{0,127}$'
       OR jsonb_typeof(referencia_motivo -> 'catalogo_version') <> 'number'
       OR (referencia_motivo ->> 'catalogo_version') !~ '^[1-9][0-9]{0,9}$'
       OR (referencia_motivo ->> 'catalogo_version')::numeric NOT BETWEEN
          1 AND 2147483647
       OR jsonb_typeof(
              referencia_motivo -> 'catalogo_huella_sha256'
          ) <> 'string'
       OR (referencia_motivo ->> 'catalogo_huella_sha256') !~
          '^[0-9a-f]{64}$'
       OR referencia_motivo ->> 'catalogo_huella_sha256' = repeat('0', 64)
       OR jsonb_typeof(referencia_motivo -> 'entrada_clave') <> 'string'
       OR (referencia_motivo ->> 'entrada_clave') !~
          '^motivo_[0-9a-f]{32}$'
       OR encode(sha256(p_motivo_canonico), 'hex') IS DISTINCT FROM
          documento_v2 ->> 'motivo_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT registro.* INTO decision
      FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
           AS registro
     WHERE registro.decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE OF registro;
    IF NOT FOUND
       OR decision.huella_decision_sha256 IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256'
       OR decision.decision_canonica IS DISTINCT FROM p_decision_canonica
       OR decision.documento_v2 IS DISTINCT FROM documento_v2
       OR decision.documento_comun IS DISTINCT FROM documento_comun
       OR decision.motivo_canonico IS DISTINCT FROM p_motivo_canonico
       OR decision.motivo_huella_sha256 IS DISTINCT FROM
          documento_v2 ->> 'motivo_huella_sha256'
       OR decision.motivo_catalogo_id IS DISTINCT FROM
          referencia_motivo ->> 'catalogo_id'
       OR decision.motivo_catalogo_version IS DISTINCT FROM
          (referencia_motivo ->> 'catalogo_version')::integer
       OR decision.motivo_catalogo_huella_sha256 IS DISTINCT FROM
          referencia_motivo ->> 'catalogo_huella_sha256'
       OR decision.motivo_entrada_clave IS DISTINCT FROM
          referencia_motivo ->> 'entrada_clave'
       OR decision.accion IS DISTINCT FROM 'bolsa.panel_interno.consultar'
       OR decision.modulo_id IS DISTINCT FROM 'bolsa'
       OR decision.tipo_recurso IS DISTINCT FROM 'panel_interno_agregado'
       OR decision.recurso_ref IS DISTINCT FROM p_recurso_ref
       OR decision.contexto_recurso_huella_sha256 IS DISTINCT FROM
          p_huella_contexto_sha256
       OR decision.finalidad IS DISTINCT FROM 'gestion_operativa_bolsa'
       OR decision.correlacion_ref IS DISTINCT FROM p_correlacion_ref
       OR documento_v2 ->> 'correlacion_ref' IS DISTINCT FROM
          p_correlacion_ref
       OR documento_v2 ->> 'concedida' <> 'true'
       OR documento_v2 ->> 'codigo' <> 'concedida'
       OR documento_v2 ->> 'garantia_minima' <> 'alto'
       OR documento_v2 -> 'campos_permitidos' IS DISTINCT FROM
          '["panel_agregado_sin_datos_personales"]'::jsonb
       OR documento_v2 -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR decision.emitida_en > verificada_en
       OR verificada_en >= decision.valida_hasta
       OR p_instante < verificada_en
       OR p_instante - verificada_en > interval '30 seconds'
       OR p_instante < decision.emitida_en
       OR p_instante >= decision.valida_hasta THEN
        RETURN false;
    END IF;

    vinculo := documento_v2 -> 'vinculo_autenticacion_actor';
    IF jsonb_typeof(vinculo) <> 'object'
       OR jsonb_typeof(vinculo -> 'cuenta_privilegiada') <> 'boolean'
       OR vinculo ->> 'garantia_observada' <> 'alto'
       OR vinculo ->> 'metodo_observado' IS NULL
       OR vinculo ->> 'metodo_observado' = 'demo'
       OR NOT COALESCE((
           (vinculo ->> 'superficie' = 'interna_corporativa'
            AND vinculo -> 'cuenta_privilegiada' = 'false'::jsonb)
           OR
           (vinculo ->> 'superficie' = 'administracion_privilegiada'
            AND vinculo -> 'cuenta_privilegiada' = 'true'::jsonb)
       ), false) THEN
        RETURN false;
    END IF;

    IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
           decision.motivo_catalogo_id,
           decision.motivo_catalogo_version,
           decision.motivo_catalogo_huella_sha256,
           decision.motivo_entrada_clave
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;

    SELECT actual.asignacion_ref, asignacion.principal_id,
           asignacion.version_rol_ref, asignacion.huella_sha256
      INTO asignacion_actual
      FROM vec_autorizacion.asignacion_perfil_actual AS actual
      JOIN vec_autorizacion.asignacion_perfil AS asignacion
        ON asignacion.perfil_activo_ref = actual.perfil_activo_ref
       AND asignacion.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = decision.perfil_activo_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion_actual.asignacion_ref IS DISTINCT FROM
          decision.asignacion_ref
       OR asignacion_actual.principal_id IS DISTINCT FROM
          decision.principal_id
       OR asignacion_actual.version_rol_ref IS DISTINCT FROM
          decision.version_rol_ref
       OR asignacion_actual.huella_sha256 IS DISTINCT FROM
          documento_v2 ->> 'asignacion_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT rol.huella_sha256, rol.documento,
           control.version_rol_ref, control.revision, control.estado,
           control.huella_sha256 AS huella_control
      INTO rol_actual
      FROM vec_autorizacion.version_rol AS rol
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = actual.version_rol_ref
       AND control.revision = actual.revision
     WHERE rol.version_rol_ref = decision.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR rol_actual.huella_sha256 IS DISTINCT FROM
          documento_v2 ->> 'version_rol_huella_sha256'
       OR rol_actual.version_rol_ref IS DISTINCT FROM
          decision.control_vigencia_version_rol_ref
       OR rol_actual.revision IS DISTINCT FROM
          decision.control_vigencia_version_rol_revision
       OR rol_actual.estado IS DISTINCT FROM 'habilitada'
       OR rol_actual.huella_control IS DISTINCT FROM
          documento_v2 ->> 'control_vigencia_version_rol_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT count(*) INTO concesiones_coincidentes
      FROM jsonb_array_elements(rol_actual.documento -> 'concesiones') AS c
     WHERE c ->> 'accion' = 'bolsa.panel_interno.consultar'
       AND c ->> 'modulo_id' = 'bolsa'
       AND c ->> 'tipo_recurso' = 'panel_interno_agregado'
       AND COALESCE(c -> 'obligaciones', '[]'::jsonb) = '[]'::jsonb
       AND (SELECT COALESCE(jsonb_agg(
                  valor ORDER BY valor COLLATE "C"
              ), '[]'::jsonb)
              FROM jsonb_array_elements_text(
                  COALESCE(c -> 'campos_permitidos', '[]'::jsonb)
              ) AS valor) =
          '["panel_agregado_sin_datos_personales"]'::jsonb
       AND EXISTS (
           SELECT 1 FROM jsonb_array_elements_text(c -> 'finalidades') AS f
            WHERE f = 'gestion_operativa_bolsa'
       );
    IF concesiones_coincidentes <> 1 THEN
        RETURN false;
    END IF;

    SELECT revision, huella_sha256 INTO catalogo_actual
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND
       OR catalogo_actual.revision IS DISTINCT FROM
          (decision.documento_comun ->>
           'revision_catalogo_politicas')::numeric
       OR catalogo_actual.huella_sha256 IS DISTINCT FROM
          decision.documento_comun ->> 'catalogo_politicas_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT COALESCE(jsonb_object_agg(
               politica.politica_ref, politica.huella_sha256
               ORDER BY politica.politica_ref COLLATE "C"
           ), '{}'::jsonb),
           COALESCE(jsonb_agg(
               politica.politica_ref
               ORDER BY politica.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO manifiesto_actual, referencias_actuales
      FROM vec_autorizacion.politica_restrictiva_actual AS actual
      JOIN vec_autorizacion.politica_restrictiva AS politica
        ON politica.politica_id = actual.politica_id
       AND politica.politica_ref = actual.politica_ref;
    SELECT COALESCE(jsonb_agg(
               referencia ORDER BY referencia COLLATE "C"
           ), '[]'::jsonb)
      INTO referencias_decision
      FROM jsonb_array_elements_text(
          decision.documento_comun -> 'politicas_evaluadas_refs'
      ) AS referencia;
    IF manifiesto_actual IS DISTINCT FROM
          decision.documento_comun ->
              'politicas_evaluadas_huellas_sha256'
       OR referencias_actuales IS DISTINCT FROM referencias_decision
       OR EXISTS (
           SELECT 1
             FROM jsonb_each(
                 decision.documento_comun -> 'politicas_huellas_sha256'
             ) AS aplicada
            WHERE manifiesto_actual -> aplicada.key IS DISTINCT FROM
                  aplicada.value
       )
       OR vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
           vinculo, decision.principal_id, decision.perfil_activo_ref,
           vinculo ->> 'contexto_actor_huella_sha256', decision.emitida_en,
           decision.valida_hasta, p_instante
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_panel_bolsa_v2(
        jsonb, bytea, bytea, text, text, text, timestamptz
    ) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_bolsa_panel_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_panel_bolsa_v2(
        jsonb, bytea, bytea, text, text, text, timestamptz
    ) TO vec_bolsa_panel_propietario;

COMMENT ON FUNCTION
    vec_autorizacion.revalidar_decision_panel_bolsa_v2(
        jsonb, bytea, bytea, text, text, text, timestamptz
    ) IS
    'Revalida decision V2 exacta del panel, motivo catalogado, RBAC/ABAC, garantia alta y sesion interna. No acredita procedencia criptografica.';
COMMIT;
