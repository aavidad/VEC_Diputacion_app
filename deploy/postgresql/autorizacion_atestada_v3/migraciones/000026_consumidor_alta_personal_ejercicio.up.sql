\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
-- Orden común con roles y la futura migración de negocio Personal.
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000026',0));

-- Prerrequisitos: roles Personal propios y AD3-25 sobre la guarda de AD3-11.
-- No se conceden membresías, permisos funcionales, claves ni confianza.
DO $prevalidacion$
DECLARE v_rol text;
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_formalizacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'estado incompatible para AD3-26' USING ERRCODE='55000';
    END IF;
    FOREACH v_rol IN ARRAY ARRAY['vec_personal_propietario','vec_personal_migrador','vec_personal_ejecutor'] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_roles WHERE rolname=v_rol
             AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
             AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
             AND rolinherit=(v_rol='vec_personal_ejecutor')
        ) THEN
            RAISE EXCEPTION 'roles Personal ausentes o incompatibles' USING ERRCODE='55000';
        END IF;
        IF has_function_privilege(v_rol,'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
            RAISE EXCEPTION 'Personal no puede acceder directamente al núcleo' USING ERRCODE='42501';
        END IF;
    END LOOP;
    IF NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_personal'
                   AND nspowner='vec_personal_propietario'::regrole)
       OR has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','USAGE')
       OR has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','CREATE') THEN
        RAISE EXCEPTION 'esquemas o ACL previas incompatibles con AD3-26' USING ERRCODE='55000';
    END IF;
END
$prevalidacion$;

-- Solo se sustituyen la guarda nominal completa conocida y un perfil añadido.
DO $ampliar$
DECLARE
    v_def text; v_nueva text; v_acl aclitem[]; v_owner oid; v_config text[];
    v_runtime_anterior text := $runtime_anterior$       OR NOT (
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
       ) THEN$runtime_anterior$;
    v_runtime_nuevo text := $runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
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
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_emisor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_consumidor', 'MEMBER')
           )
       ) THEN$runtime_nuevo$;
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_personal.alta_ejercicio.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_relacion_ocupacion_sinteticas'
           )
$perfil$;
    v_marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner,p.proconfig
      INTO STRICT v_def,v_acl,v_owner,v_config
      FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_runtime_anterior,''))<>length(v_runtime_anterior)
       OR length(v_def)-length(replace(v_def,v_marca,''))<>length(v_marca)
       OR strpos(v_def,'alta_personal_ejercicio')<>0
       OR strpos(v_def,'p_perfil_mutacion IS NOT DISTINCT FROM ''resolucion_formalizacion_ct''')=0
       OR strpos(v_def,'(d ->> ''valida_hasta'')::timestamptz')=0 THEN
        RAISE EXCEPTION 'núcleo incompatible con extensión nominal Personal' USING ERRCODE='55000';
    END IF;
    v_nueva := replace(replace(v_def,v_runtime_anterior,v_runtime_nuevo),v_marca,v_extension||v_marca);
    EXECUTE v_nueva;
    IF EXISTS (
        SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
         AND (p.proacl IS DISTINCT FROM v_acl OR p.proowner IS DISTINCT FROM v_owner
              OR p.proconfig IS DISTINCT FROM v_config OR NOT p.prosecdef
              OR pg_get_functiondef(p.oid) IS DISTINCT FROM v_nueva)
    ) THEN
        RAISE EXCEPTION 'AD3-26 alteró definición o autoridad ajena del núcleo' USING ERRCODE='55000';
    END IF;
END
$ampliar$;

-- Dos topologías cerradas: AD3-7/11 completa o AD3-11 legacy seguida de 3/4.
-- Solo Personal se añade/retira; no se admiten subconjuntos, reordenaciones ni extras.
DO $audiencias$
DECLARE
    v_def text; v_esperada text; v_legada text;
    v_audiencias_legadas text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
    ];
    v_audiencias text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
        'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1'
    ];
BEGIN
    LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
    SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT v_def FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check'
       AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
    v_esperada := 'CHECK (audiencia_consumo = ANY (ARRAY[' ||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(v_audiencias) WITH ORDINALITY AS u(a,n) ORDER BY n),', ') || ']))';
    v_legada := 'CHECK (audiencia_consumo = ANY (ARRAY[' ||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(v_audiencias_legadas) WITH ORDINALITY AS u(a,n) ORDER BY n),', ') || ']))';
    IF v_def = v_legada THEN
        v_audiencias := v_audiencias_legadas;
    ELSIF v_def IS DISTINCT FROM v_esperada THEN
        RAISE EXCEPTION 'lista de audiencias incompatible con AD3-26' USING ERRCODE='55000';
    END IF;
    v_audiencias := array_append(v_audiencias,'vec_personal.alta_ejercicio.v1');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ' ||
        'ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check ' ||
        'CHECK (audiencia_consumo IN (' ||
        array_to_string(ARRAY(SELECT quote_literal(a)
            FROM unnest(v_audiencias) WITH ORDINALITY AS u(a,n) ORDER BY n),', ') || '))';
END
$audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s'
AS $funcion$
DECLARE v_consumo record;
BEGIN
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'alta_personal_ejercicio',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    -- También el replay del negocio exige autorización fresca: nunca reutilizar
    -- el consumo de una decisión anterior aunque el núcleo conozca ese recibo.
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'alta Personal de ejercicio requiere consumo nuevo' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT v_consumo.decision_ref,v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref,v_consumo.consumida_en,true;
END
$funcion$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_migrador,vec_personal_ejecutor,vec_personal_migrador,
    vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador,
    vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_migrador;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_personal_propietario;

DO $permisos$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
            aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
         WHERE p.oid='vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
           AND (a.grantee<>ALL(ARRAY['vec_autorizacion_atestada_v3_propietario'::regrole::oid,
                                     'vec_personal_propietario'::regrole::oid])
                OR a.privilege_type<>'EXECUTE'
                OR (a.grantee='vec_personal_propietario'::regrole::oid AND a.is_grantable))
    ) OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_alta_personal_ejercicio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'ACL inesperada en consumidor Personal' USING ERRCODE='42501';
    END IF;
END
$permisos$;
COMMIT;
