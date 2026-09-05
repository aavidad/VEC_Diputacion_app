\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000011', 0));

-- 000012 debe ampliar la definición resultante, no reinstalar 000010.
-- Se conserva su marca de inserción de perfiles y todo el núcleo criptográfico.
DO $ampliar$
DECLARE
    v_def text;
    v_acl aclitem[];
    v_runtime_anterior text := $runtime_anterior$       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER') THEN$runtime_anterior$;
    v_runtime_nuevo text := $runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
           )
       ) THEN$runtime_nuevo$;
    v_marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1'
               AND (
                   c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.orden.preparar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.abrir'
               )
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'bolsa'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'integracion_llamamientos_bolsa'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$perfil$;
    v_rol text;
BEGIN
    IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'estado incompatible para consumidor de Bolsa'
            USING ERRCODE = '55000';
    END IF;
    FOREACH v_rol IN ARRAY ARRAY[
        'vec_bolsa_llamamientos_propietario',
        'vec_bolsa_llamamientos_ejecutor',
        'vec_bolsa_llamamientos_migrador'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = v_rol AND NOT rolcanlogin AND NOT rolsuper
              AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolbypassrls) THEN
            RAISE EXCEPTION 'rol técnico de Bolsa ausente o incompatible'
                USING ERRCODE = '55000';
        END IF;
    END LOOP;
    SELECT pg_catalog.pg_get_functiondef(p.oid), p.proacl INTO STRICT v_def, v_acl
      FROM pg_catalog.pg_proc p
     WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF pg_catalog.has_function_privilege(
           'vec_bolsa_llamamientos_propietario', 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_bolsa_llamamientos_ejecutor', 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
       OR pg_catalog.strpos(v_def, 'bolsa_llamamiento') <> 0
       OR (pg_catalog.length(v_def) -
           pg_catalog.length(pg_catalog.replace(v_def, v_runtime_anterior, ''))) <>
          pg_catalog.length(v_runtime_anterior)
       OR (pg_catalog.length(v_def) -
           pg_catalog.length(pg_catalog.replace(v_def, v_marca, ''))) <>
          pg_catalog.length(v_marca) THEN
        RAISE EXCEPTION 'núcleo o permisos incompatibles con consumidor Bolsa'
            USING ERRCODE = '55000';
    END IF;
    v_def := pg_catalog.replace(v_def, v_runtime_anterior, v_runtime_nuevo);
    v_def := pg_catalog.replace(v_def, v_marca, v_extension || v_marca);
    EXECUTE v_def;
    IF (SELECT p.proacl FROM pg_catalog.pg_proc p
        WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'la extensión alteró permisos del núcleo'
            USING ERRCODE = '55000';
    END IF;
END
$ampliar$;

-- Dos listas cerradas: CT sola o las siete audiencias del código.
-- Solo se añade/retira Bolsa; cualquier otra definición se rechaza.
DO $audiencias$
DECLARE
    v_def text;
    v_esperada text;
    v_legada text;
    v_audiencias text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
        'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
    ];
BEGIN
    LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        IN ACCESS EXCLUSIVE MODE;
    SELECT pg_catalog.regexp_replace(
        pg_catalog.pg_get_constraintdef(c.oid, true), '\s+', ' ', 'g')
      INTO STRICT v_def
      FROM pg_catalog.pg_constraint c
     WHERE c.conrelid =
           'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname = 'clave_capacidad_version_audiencia_consumo_check'
       AND c.contype = 'c' AND c.convalidated
       AND c.conkey = ARRAY[8]::smallint[];
    v_esperada := 'CHECK (audiencia_consumo = ANY (ARRAY[' ||
        pg_catalog.array_to_string(ARRAY(
            SELECT pg_catalog.quote_literal(a) || '::text'
              FROM pg_catalog.unnest(v_audiencias) a
        ), ', ') || ']))';
    v_legada := 'CHECK (audiencia_consumo = ' ||
        pg_catalog.quote_literal(v_audiencias[1]) || '::text)';
    IF v_def = v_legada THEN
        v_audiencias := v_audiencias[1:1];
    ELSIF v_def IS DISTINCT FROM v_esperada THEN
        RAISE EXCEPTION 'gobierno de audiencias incompatible con 000011'
            USING ERRCODE = '55000';
    END IF;
    v_audiencias := pg_catalog.array_append(v_audiencias,
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ' ||
        'ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check ' ||
        'CHECK (audiencia_consumo IN (' ||
        pg_catalog.array_to_string(ARRAY(
            SELECT pg_catalog.quote_literal(a)
              FROM pg_catalog.unnest(v_audiencias) a
        ), ', ') || '))';
END
$audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS TABLE (
    decision_ref text, efecto_ref text, huella_efecto_sha256 text,
    consumo_huella_sha256 text, auditoria_ref text,
    consumida_en timestamptz, consumo_nuevo boolean
)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
    SELECT * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'bolsa_llamamiento', p_capacidad, p_decision, p_motivo, p_contexto,
        p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz)
$funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC, vec_autorizacion_atestada_v3_consumidor,
      vec_autorizacion_atestada_v3_emisor, vec_contratacion_temporal_propietario,
      vec_bolsa_llamamientos_ejecutor, vec_bolsa_llamamientos_migrador;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_propietario;

DO $permisos$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc p,
            LATERAL pg_catalog.aclexplode(
                coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) a
        WHERE p.oid = 'vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
          AND (a.grantee <> ALL (ARRAY[
                'vec_autorizacion_atestada_v3_propietario'::regrole::oid,
                'vec_bolsa_llamamientos_propietario'::regrole::oid])
               OR (a.grantee = 'vec_bolsa_llamamientos_propietario'::regrole::oid
                   AND a.is_grantable))
    ) THEN
        RAISE EXCEPTION 'permiso inesperado en consumidor nominal de Bolsa'
            USING ERRCODE = '42501';
    END IF;
END
$permisos$;
COMMIT;
