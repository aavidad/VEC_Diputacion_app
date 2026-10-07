\set ON_ERROR_STOP on
-- AD3-118. Usuarios externo reutiliza la historia EXTERNA de AD3-116, pero
-- consume decisiones y revalida motivos exclusivamente mediante AUT-17.
-- Requiere AD3-116, AUT-17 y la cuenta técnica de Usuarios externo.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000118',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $preimagen$
DECLARE
 f regprocedure := pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR f IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc WHERE oid=f
       AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.atestacion_decision_v3_externa') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3_externa') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.decision_concedida_contexto_actor_v3_externa') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
       'vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
       'vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(bytea,bytea,numeric,numeric)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_usuarios_ejecutor_externo'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-118: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $carril$
DECLARE
 f regprocedure := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 nucleo regprocedure := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; usuarios text; publicado text; ampliado text;
 inicio integer; fin integer; marca text;
 guardia text := $g$BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR session_user <> 'vec_externo_usuarios_desarrollo'
       OR p_perfil_mutacion IS NULL
       OR p_perfil_mutacion <> ALL (ARRAY[
          'preferencias_consulta_usuarios','preferencias_actualizacion_usuarios',
          'imagen_consultar_usuarios','imagen_actualizar_usuarios',
          'correos_consultar_usuarios','correos_anadir_usuarios',
          'correos_reenviar_usuarios','correos_verificar_usuarios',
          'correos_activar_usuarios','correos_retirar_usuarios'])
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user
          AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreatedb
          AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
           WHERE m.member=session_user::regrole) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
          WHERE m.member=session_user::regrole
            AND m.roleid='vec_usuarios_ejecutor_externo'::regrole
            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
       OR EXISTS (WITH RECURSIVE roles(rol_id) AS (
            SELECT m.roleid FROM pg_catalog.pg_auth_members m
             WHERE m.member=session_user::regrole
            UNION
            SELECT m.roleid FROM pg_catalog.pg_auth_members m
             JOIN roles r ON r.rol_id=m.member)
           SELECT 1 FROM roles WHERE rol_id<>'vec_usuarios_ejecutor_externo'::regrole)
    THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='consumo Usuarios externo VEC-AD-3 rechazado';
    END IF;
$g$;
 despacho text := $d$BEGIN
    IF session_user = 'vec_externo_usuarios_desarrollo' THEN
        IF p_perfil_mutacion IS NULL OR p_perfil_mutacion <> ALL (ARRAY[
          'preferencias_consulta_usuarios','preferencias_actualizacion_usuarios',
          'imagen_consultar_usuarios','imagen_actualizar_usuarios',
          'correos_consultar_usuarios','correos_anadir_usuarios',
          'correos_reenviar_usuarios','correos_verificar_usuarios',
          'correos_activar_usuarios','correos_retirar_usuarios']) THEN
            RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='perfil Usuarios externo VEC-AD-3 rechazado';
        END IF;
        RETURN QUERY SELECT q.* FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(
            p_perfil_mutacion,p_capacidad_canonica,p_decision_canonica,p_motivo_canonico,
            p_contexto_actor_canonico,p_persona_version,p_perfil_version,p_payload_vec_ad_3,
            p_sobre_cose_sign1,p_evidencia_verificacion,p_raiz_publica_spki) q;
        RETURN;
    END IF;
$d$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT original;
 marca := E'BEGIN\n    IF pg_catalog.current_setting(''transaction_isolation'')';
 inicio := pg_catalog.strpos(original,marca);
 fin := pg_catalog.strpos(original,'    SELECT setting::numeric INTO v_statement');
 IF inicio=0 OR fin<=inicio
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,'')))<>pg_catalog.length(marca)
    OR pg_catalog.strpos(original,'CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(')<>1
    OR pg_catalog.strpos(original,'registrar_y_revalidar_decision_contexto_actor_externa_v3(')=0
    OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,
         'revalidar_decision_contexto_actor_externa_v3_viva(',''))) /
       pg_catalog.length('revalidar_decision_contexto_actor_externa_v3_viva(') <> 2
    OR pg_catalog.strpos(original,'vec_autorizacion_atestada_v3.atestacion_decision_v3_externa')=0
 THEN RAISE EXCEPTION 'AD3-118: carril externo incompatible' USING ERRCODE='55000'; END IF;
 usuarios := pg_catalog.replace(original,
   'CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(',
   'CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(');
 inicio := pg_catalog.strpos(usuarios,marca);
 fin := pg_catalog.strpos(usuarios,'    SELECT setting::numeric INTO v_statement');
 usuarios := pg_catalog.substr(usuarios,1,inicio-1)||guardia||pg_catalog.substr(usuarios,fin);
 usuarios := pg_catalog.replace(usuarios,
   'registrar_y_revalidar_decision_contexto_actor_externa_v3(',
   'registrar_y_revalidar_decision_usuarios_externo_v3(');
 usuarios := pg_catalog.replace(usuarios,
   'revalidar_decision_contexto_actor_externa_v3_viva(',
   'revalidar_decision_usuarios_externo_v3_viva(');
 EXECUTE usuarios;
 SELECT pg_catalog.pg_get_functiondef(nucleo) INTO STRICT publicado;
 marca := E'BEGIN\n    IF session_user = ''vec_externo_bolsa_desarrollo''';
 IF pg_catalog.strpos(publicado,marca)=0
    OR (pg_catalog.length(publicado)-pg_catalog.length(pg_catalog.replace(publicado,marca,'')))<>pg_catalog.length(marca)
 THEN RAISE EXCEPTION 'AD3-118: despacho incompatible' USING ERRCODE='55000'; END IF;
 ampliado := pg_catalog.replace(publicado,marca,despacho||'    IF session_user = ''vec_externo_bolsa_desarrollo''');
 EXECUTE ampliado;
 IF pg_catalog.pg_get_functiondef(nucleo) IS DISTINCT FROM ampliado
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-118: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $carril$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_usuarios_externa_interna(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor;
COMMIT;
