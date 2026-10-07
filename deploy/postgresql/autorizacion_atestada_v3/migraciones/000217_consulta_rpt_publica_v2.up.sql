\set ON_ERROR_STOP on
-- AD217: consumidor nominal de la publicación RPT v2. Personal39 instala su
-- recibo después de esta fachada. AD214/AD216 conservan su historia y huellas.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000217',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass
       AND attname='transaccion_origen' AND atttypid='xid8'::regtype AND NOT attisdropped)
    OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
       AND attname='transaccion_origen' AND atttypid='xid8'::regtype AND NOT attisdropped)
 THEN RAISE EXCEPTION 'AD217: PARO preimagen causal/TopXID/fachada incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_propietario' AND NOT rolcanlogin
   AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_rpt_v2_lector'
   AND NOT rolcanlogin AND NOT rolinherit AND rolconfig IS NULL
   AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_rpt_v2_app'
   AND rolcanlogin AND rolinherit AND rolconfig IS NULL
   AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member=to_regrole('vec_personal_rpt_v2_lector'))
 OR (SELECT count(*) FROM pg_auth_members WHERE member=to_regrole('vec_personal_rpt_v2_app'))<>1
 OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=to_regrole('vec_personal_rpt_v2_app')
   AND m.roleid=to_regrole('vec_personal_rpt_v2_lector')
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 OR NOT has_database_privilege(to_regrole('vec_personal_rpt_v2_app'),current_database(),'CONNECT')
 OR has_database_privilege(to_regrole('vec_personal_rpt_v2_app'),current_database(),'TEMP')
 OR NOT has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','USAGE')
 OR to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE login_nombre='vec_personal_rpt_v2_app')
 THEN RAISE EXCEPTION 'AD217: PARO roles o ACL Personal incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo_pre$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 definicion text; fuente text; meta jsonb; deps jsonb; compartidas jsonb;
BEGIN
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT definicion,fuente,meta
 FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(definicion,'UTF8')),'hex')='e314eb6242dd0681dbd8cf5618e4287140831f56ab9faa8fe2222f89f0196773'
 AND encode(sha256(convert_to(fuente,'UTF8')),'hex')='1a8271745e5c0cb028ad91d105e69f31a13dd7e046aa31385405d54e5518c9f7' THEN
  RAISE EXCEPTION 'AD217 ya instalada: esperado pre 96d92f.../036212... actual post e314eb.../1a8271...' USING ERRCODE='55000'; END IF;
 IF encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM '96d92f060ddfd4970bf47167c5b23ee140bc67bd521166363c942f6f83203bc3'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '03621298761d25e5e5decb2a8bee13cb2193f9fc07035a6dd7145ea5e5eec1cd'
 THEN RAISE EXCEPTION 'AD217: PARO preimagen de núcleo; def=% src=%',
  encode(sha256(convert_to(definicion,'UTF8')),'hex'),encode(sha256(convert_to(fuente,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
    AND a.grantee=p.proowner AND a.grantor=p.proowner
    AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR strpos(definicion,'resolver_origen_consumo_v1')=0
 OR strpos(definicion,'transaccion_origen')=0 OR strpos(definicion,'consumo_confirmado_v4')=0
 OR strpos(definicion,'consulta_rpt_publica_v2_personal')<>0
 THEN RAISE EXCEPTION 'AD217: PARO autoridad, ACL u origen V3 divergentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 PERFORM set_config('vec.ad217.pre_meta',meta::text,true);
 PERFORM set_config('vec.ad217.pre_deps',deps::text,true);
 PERFORM set_config('vec.ad217.pre_shared',compartidas::text,true);
END $nucleo_pre$;

-- Definición POST216 íntegra con tres diferencias cotejadas: exclusión del
-- perfil genérico, guarda del LOGIN/grupo RPT y tupla cerrada. No se ejecuta
-- DDL construido desde catálogos en tiempo de instalación.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea)
 RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path = pg_catalog, pg_temp
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    c jsonb;
    d jsonb;
    x jsonb;
    v_clave record;
    v_puntero_clave record;
    v_config record;
    v_raiz record;
    v_registro record;
    v_replay record;
    v_ahora timestamptz(6);
    v_huella_capacidad text;
    v_huella_consumo text;
    v_preimagen_auditoria bytea;
    v_proceso_origen text;
    v_canal_origen text;
    v_actor_nominal text;
    v_perfil_nominal text;
    v_finalidad_nominal text;
    v_fecha_nominal text;
    v_transaccion_origen xid8;
    v_anterior text;
    v_secuencia numeric(20, 0);
    v_auditoria_ref text;
    v_statement numeric;
    v_idle numeric;
    v_revalidada_en timestamptz(6);
BEGIN
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
    IF session_user = 'vec_externo_bolsa_desarrollo' THEN
        IF p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa')
           OR p_perfil_mutacion IS NULL THEN
            RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='perfil externo VEC-AD-3 rechazado';
        END IF;
        RETURN QUERY SELECT q.* FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(
            p_perfil_mutacion,p_capacidad_canonica,p_decision_canonica,p_motivo_canonico,
            p_contexto_actor_canonico,p_persona_version,p_perfil_version,p_payload_vec_ad_3,
            p_sobre_cose_sign1,p_evidencia_verificacion,p_raiz_publica_spki) q;
        RETURN;
    END IF;
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR current_user <>
          'vec_autorizacion_atestada_v3_propietario'
       OR session_user = current_user
       OR (pg_catalog.pg_has_role(session_user, 'vec_bolsa_llamamientos_portal_externo', 'MEMBER')
           AND (p_perfil_mutacion IS DISTINCT FROM 'consulta_participaciones_propias_bolsa'
                AND p_perfil_mutacion IS DISTINCT FROM 'portal_candidato_bolsa'
                OR EXISTS (WITH RECURSIVE roles(rol_id) AS (
                     SELECT m.roleid FROM pg_catalog.pg_auth_members m
                      WHERE m.member=session_user::pg_catalog.regrole
                     UNION
                     SELECT m.roleid FROM pg_catalog.pg_auth_members m
                      JOIN roles r ON r.rol_id=m.member)
                    SELECT 1 FROM roles
                     WHERE rol_id<>'vec_bolsa_llamamientos_portal_externo'::pg_catalog.regrole)))
       OR NOT (
           (p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_plan_nominal_firma_ct'
            AND vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'lote_perfiles_admin'
            AND vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1() IS TRUE)
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'historia_servicios_propia'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb
                  AND NOT r.rolreplication AND NOT r.rolbypassrls AND r.rolconfig IS NULL)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_personal_ejecutor'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_rrhh_bolsa'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
                 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
                 AND r.rolconfig IS NULL)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
                 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_bolsa_llamamientos_ejecutor'::regrole)
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_rpt_publica_v2_personal'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
                 AND r.rolname='vec_personal_rpt_v2_app' AND r.rolcanlogin AND r.rolinherit
                 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
                 AND r.rolconfig IS NULL)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_personal_rpt_v2_lector'::regrole
                 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_rpt_v2_lector'::regrole)
               AND has_database_privilege(session_user,current_database(),'CONNECT')
               AND NOT has_database_privilege(session_user,current_database(),'TEMP')
           )
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'servicios_certificados_propios'
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
             AND m.roleid='vec_personal_ejecutor'::regrole
             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
            AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'exportacion_servicios_propios'
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
             AND m.roleid='vec_personal_ejecutor'::regrole
             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
            AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)
           OR (p_perfil_mutacion IN ('usuarios_admin_listar','usuarios_admin_consultar')

            AND vec_autorizacion_atestada_v3.login_usuarios_admin_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IN ('persona_denominacion_publicar','persona_denominacion_leer')
            AND vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'capacidades_admin'
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
             AND m.roleid='vec_admin_perfiles_lector'::regrole
             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
            AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member='vec_admin_perfiles_lector'::regrole)
            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'publicacion_cargo_competencial'
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
             AND m.roleid='vec_personal_ejecutor'::regrole
             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
            AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'publicacion_certificado_nominal'
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
             AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
            AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
             AND m.roleid='vec_autorizacion_certificado_nominal_ejecutor'::regrole
             AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
            AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole)=1)
           OR (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'consulta_rrhh_bolsa' AND p_perfil_mutacion IS DISTINCT FROM 'consulta_rpt_publica_v2_personal' AND p_perfil_mutacion IS DISTINCT FROM 'historia_servicios_propia' AND p_perfil_mutacion IS DISTINCT FROM 'exportacion_servicios_propios' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_cargo_competencial' AND p_perfil_mutacion IS DISTINCT FROM 'capacidades_admin' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_publicar' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_leer' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_listar' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_consultar' AND p_perfil_mutacion IS DISTINCT FROM 'servicios_certificados_propios' AND p_perfil_mutacion IS DISTINCT FROM 'lote_perfiles_admin' AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_plan_nominal_firma_ct'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_politica_ofertas_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_persona_aceptacion_ct_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_anclaje_aceptacion_ct_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_reincorporacion_titular_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'aspirantes_ficha_consultar'
               AND p_perfil_mutacion IS DISTINCT FROM 'aspirantes_ficha_alta'
               AND p_perfil_mutacion IS DISTINCT FROM 'aspirantes_ficha_rectificar'
               AND p_perfil_mutacion IS DISTINCT FROM 'preferencias_consulta_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'preferencias_actualizacion_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'correos_consultar_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'correos_anadir_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'correos_reenviar_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'correos_verificar_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'correos_activar_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'correos_retirar_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'imagen_consultar_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'imagen_actualizar_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'correo_avisos_llamamiento_usuarios'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_politica_cese_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'politica_ofertas_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_auditoria_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'portal_candidato_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_participaciones_propias_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'situacion_participacion_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'contacto_participacion_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'datos_contacto_participacion_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'emision_llamamiento_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_relacion_propia_dietas_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'crear_borrador_propio_dietas'
               AND p_perfil_mutacion IS DISTINCT FROM 'consultar_borrador_propio_dietas'
               AND p_perfil_mutacion IS DISTINCT FROM 'acceso_rutas_dietas'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_organizacion_historica_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'importacion_organizacion_historica_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'operacion_documentos_comunes'
               AND p_perfil_mutacion IS DISTINCT FROM 'revisor_documento_dietas'
               AND p_perfil_mutacion IS DISTINCT FROM 'competencias_asignacion_dietas_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'rectificacion_dietas_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'documento_dietas_mutacion'
               AND p_perfil_mutacion IS DISTINCT FROM 'documento_dietas_consulta'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_asignacion_dietas_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'correccion_asignacion_dietas_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'alta_inicial_asignacion_dietas_personal'
               AND p_perfil_mutacion IS DISTINCT FROM 'prelectura_dietas'
               AND p_perfil_mutacion IS DISTINCT FROM 'circuito_dietas'
               AND p_perfil_mutacion IS DISTINCT FROM 'bandeja_dietas'
               AND p_perfil_mutacion IS DISTINCT FROM 'registro_empleado_b2'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_marcaje_propio'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_marcaje_remoto_disponibilidad'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_marcaje_remoto_recibo'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_saldo_propio'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_movimientos_propio'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correccion_solicitar'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_permisos_propio'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_permiso_solicitar'
               AND p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio' AND p_perfil_mutacion IS DISTINCT FROM 'lectura_registro_personal_incorporacion' AND p_perfil_mutacion IS DISTINCT FROM 'lectura_registro_personal_incorporacion_v2' AND p_perfil_mutacion IS DISTINCT FROM 'lectura_categorias' AND p_perfil_mutacion IS DISTINCT FROM 'usos_categorias'
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
               (p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_ofertas_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_persona_aceptacion_ct_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_anclaje_aceptacion_ct_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_reincorporacion_titular_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_cese_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'politica_ofertas_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_auditoria_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'datos_contacto_participacion_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'emision_llamamiento_bolsa')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND ((pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
                        AND NOT pg_catalog.pg_has_role(
                           session_user, 'vec_bolsa_llamamientos_portal_externo', 'MEMBER'))
                   OR ((p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
                        OR p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa')
                       AND pg_catalog.pg_has_role(
                           session_user, 'vec_bolsa_llamamientos_portal_externo', 'MEMBER')
                       AND NOT pg_catalog.pg_has_role(
                           session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')))
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
                      OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion'
               AND (
                 (
                   pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
                   AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_personal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
                   AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
                 )
                 OR (
                   pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
                   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_contratacion_temporal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
                   AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_contratacion_temporal_ejecutor'::regrole)
                 )
               )
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_emisor','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_consumidor','MEMBER')
               AND NOT EXISTS (
                 SELECT 1 FROM pg_roles r
                 WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user
                   AND r.rolname NOT IN ('vec_personal_ejecutor','vec_contratacion_temporal_ejecutor')
                   AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER')
               )
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion_v2'
               AND (
                 (
                   pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
                   AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_personal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
                   AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
                 )
                 OR (
                   pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
                   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_contratacion_temporal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
                   AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_contratacion_temporal_ejecutor'::regrole)
                 )
               )
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_emisor','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_consumidor','MEMBER')
               AND NOT EXISTS (
                 SELECT 1 FROM pg_roles r
                 WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user
                   AND r.rolname NOT IN ('vec_personal_ejecutor','vec_contratacion_temporal_ejecutor')
                   AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER')
               )
           )
           OR (
               p_perfil_mutacion IN ('consulta_relacion_propia_dietas_personal','crear_borrador_propio_dietas','consultar_borrador_propio_dietas')
               AND pg_catalog.pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_dietas_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_dietas_ejecutor'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_dietas_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'acceso_rutas_dietas'
               AND pg_catalog.pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_dietas_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_dietas_ejecutor'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_dietas_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_organizacion_historica_personal'
               AND pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_personal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_personal_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'importacion_organizacion_historica_personal'
               AND pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_personal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_personal_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar')
               AND pg_catalog.pg_has_role(session_user,'vec_cronos_v1_ejecutor','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_cronos_v1_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_cronos_v1_ejecutor'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_cronos_v1_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IN ('documento_dietas_mutacion','documento_dietas_consulta','prelectura_dietas','circuito_dietas','bandeja_dietas')
               AND EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.roleid=g.oid WHERE g.rolname='vec_dietas_ejecutor' AND m.member=session_user::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))
               AND NOT EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.member=g.oid WHERE g.rolname='vec_dietas_ejecutor')
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_dietas_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IN ('consulta_asignacion_dietas_personal','correccion_asignacion_dietas_personal','alta_inicial_asignacion_dietas_personal')
               AND EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.roleid=g.oid WHERE g.rolname='vec_personal_d7_ejecutor' AND m.member=session_user::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))
               AND NOT EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.member=g.oid WHERE g.rolname='vec_personal_d7_ejecutor')
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_personal_d7_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IN ('competencias_asignacion_dietas_personal','rectificacion_dietas_personal')
               AND EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.roleid=g.oid WHERE g.rolname='vec_personal_d7_ejecutor' AND m.member=session_user::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))
               AND NOT EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.member=g.oid WHERE g.rolname='vec_personal_d7_ejecutor')
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_personal_d7_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'revisor_documento_dietas'
               AND EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.roleid=g.oid WHERE g.rolname='vec_dietas_ejecutor' AND m.member=session_user::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))
               AND NOT EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.member=g.oid WHERE g.rolname='vec_dietas_ejecutor')
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_dietas_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
               AND EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.roleid=g.oid WHERE g.rolname='vec_personal_ejecutor' AND m.member=session_user::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))
               AND NOT EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.member=g.oid WHERE g.rolname='vec_personal_ejecutor')
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_personal_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
               AND pg_catalog.pg_has_role(session_user,'vec_documentos_ejecutor','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_documentos_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_documentos_ejecutor'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_documentos_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IN ('preferencias_consulta_usuarios','preferencias_actualizacion_usuarios','correos_consultar_usuarios','correos_anadir_usuarios','correos_reenviar_usuarios','correos_verificar_usuarios','correos_activar_usuarios','correos_retirar_usuarios','imagen_consultar_usuarios','imagen_actualizar_usuarios','correo_avisos_llamamiento_usuarios')
               AND pg_catalog.pg_has_role(session_user,'vec_usuarios_ejecutor_interno','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_usuarios_ejecutor_interno'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_usuarios_ejecutor_interno'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_usuarios_ejecutor_interno' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IN ('preferencias_consulta_usuarios','preferencias_actualizacion_usuarios','correos_consultar_usuarios','correos_anadir_usuarios','correos_reenviar_usuarios','correos_verificar_usuarios','correos_activar_usuarios','correos_retirar_usuarios','imagen_consultar_usuarios','imagen_actualizar_usuarios','correo_avisos_llamamiento_usuarios')
               AND pg_catalog.pg_has_role(session_user,'vec_usuarios_ejecutor_externo','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_usuarios_ejecutor_externo'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_usuarios_ejecutor_externo'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_usuarios_ejecutor_externo' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IN ('aspirantes_ficha_consultar','aspirantes_ficha_alta','aspirantes_ficha_rectificar')
               AND pg_catalog.pg_has_role(session_user,'vec_aspirantes_ejecutor_externo','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_aspirantes_ejecutor_externo'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_aspirantes_ejecutor_externo'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_aspirantes_ejecutor_externo' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_categorias'
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
                    WHERE m.member=session_user::pg_catalog.regrole
                      AND g.rolname IN ('vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor')
                      AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
                      AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))=1
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::pg_catalog.regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
                    WHERE m.member IN ('vec_contratacion_temporal_ejecutor'::pg_catalog.regrole,
                                       'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole,
                                       'vec_personal_ejecutor'::pg_catalog.regrole))
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE pg_catalog.left(r.rolname,4)='vec_'
                    AND r.rolname<>session_user
                    AND r.rolname NOT IN ('vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor')
                    AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'usos_categorias'
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
                    WHERE m.member=session_user::pg_catalog.regrole
                      AND g.rolname IN ('vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor')
                      AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
                      AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER'))=1
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::pg_catalog.regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
                    WHERE m.member IN ('vec_contratacion_temporal_ejecutor'::pg_catalog.regrole,
                                       'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole,
                                       'vec_personal_ejecutor'::pg_catalog.regrole))
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE pg_catalog.left(r.rolname,4)='vec_'
                    AND r.rolname<>session_user
                    AND r.rolname NOT IN ('vec_contratacion_temporal_ejecutor','vec_bolsa_llamamientos_ejecutor','vec_personal_ejecutor')
                    AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo VEC-AD-3 rechazado';
    END IF;
    SELECT setting::numeric INTO v_statement
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout' AND unit = 'ms';
    SELECT setting::numeric INTO v_idle
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout' AND unit = 'ms';
    IF v_statement IS NULL OR v_statement NOT BETWEEN 1 AND 15000
       OR v_idle IS NULL OR v_idle NOT BETWEEN 1 AND 20000 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'límites VEC-AD-3 ausentes';
    END IF;
    IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(
           p_capacidad_canonica) IS NOT TRUE
       OR pg_catalog.octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR pg_catalog.octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
       OR pg_catalog.octet_length(p_contexto_actor_canonico)
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.octet_length(p_payload_vec_ad_3)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_sobre_cose_sign1)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_evidencia_verificacion)
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR pg_catalog.scale(p_persona_version) <> 0
       OR pg_catalog.scale(p_perfil_version) <> 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada VEC-AD-3 inválida';
    END IF;
    BEGIN
        c := pg_catalog.convert_from(p_capacidad_canonica, 'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
        x := pg_catalog.convert_from(p_contexto_actor_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'entrada VEC-AD-3 inválida';
    END;
    IF vec_autorizacion_atestada_v3.capacidad_tipos_validos(c) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.capacidad_canonica(c)
          IS DISTINCT FROM p_capacidad_canonica
       OR c ->> 'esquema' <>
          'vec.autorizacion.capacidad-registro-consumo-atestado.v3'
       OR c ->> 'version' <> '3'
       OR NOT (
           (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.solicitud.crear'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.solicitud.crear'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'asignacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.unidad.asignar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.unidad.asignar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM
                   'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'asignacion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'informe_juridico'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.informe_juridico.generar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.informe_juridico.generar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM
                   'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'informe_juridico_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'fiscalizacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.fiscalizacion.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.fiscalizacion.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM
                   'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'fiscalizacion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1'
               AND (
                   c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.orden.preparar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.abrir'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.aceptacion_rrhh.registrar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.renuncia_rrhh.registrar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.siguiente.abrir'
               )
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'bolsa'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'integracion_llamamientos_bolsa'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'comunicacion_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.comunicacion.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.comunicacion.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'comunicacion_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_seleccion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_orden'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'reanudacion_seleccion_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'respuesta_recibida_rrhh'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'respuesta_recibida_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_justificante_respuesta_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.consultar_justificante'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.consultar_justificante'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'justificante_respuesta_recibida_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'resolucion_manual_respuesta_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'resolucion_manual_respuesta_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'continuacion_llamamiento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.siguiente.continuar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.siguiente.continuar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'continuacion_llamamiento_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'propuesta_formalizacion_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.formalizacion.propuesta.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.formalizacion.propuesta.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'propuesta_formalizacion_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'organizacion_preparacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.organizacion.actualizar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'personal.organizacion.actualizar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'estructura_organizativa'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_estructura_organizativa'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'peticion_centro'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.peticion_centro.presentar',
                   'contratacion_temporal.peticion_centro.ratificar',
                   'contratacion_temporal.peticion_centro.consultar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'peticion_centro'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'entrega_peticion_centro'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.peticion_centro.rrhh.consultar',
                   'contratacion_temporal.peticion_centro.rrhh.entregar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'entrega_peticion_centro'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'tramitar_peticion_centro_rrhh'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'resolucion_formalizacion_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'contratacion_temporal.formalizacion.resolucion.manual_ejercicio.registrar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'resolucion_formalizacion_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_personal.alta_ejercicio.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_relacion_ocupacion_sinteticas'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_ejercicio_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.incorporacion_ejercicio.v2'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.incorporacion.confirmar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'confirmacion_incorporacion_ejercicio_v2'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_incorporacion_confirmada_por_personal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.lectura_incorporacion.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'registro_alta_ejercicio_incorporacion'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion_v2'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.lectura_incorporacion.v2'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'registro_alta_ejercicio_incorporacion_v2'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'anotacion_administrativa_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.anotacion_administrativa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.anotacion_administrativa.registrar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'anotacion_administrativa_incorporacion_v1'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_anotacion_administrativa_incorporacion'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'cierre_administrativo_sin_cese_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.cerrar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'seguimiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'cerrar_expediente_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'subsanacion_reparos_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'subsanacion_reparo_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'despacho_correo_llamamiento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.despacho_correo_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.llamamiento.correo.despachar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'despacho_correo_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'resultado_correo_llamamiento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.resultado_correo_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.llamamiento.correo.registrar_resultado'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'resultado_correo_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.mi-bolsa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.consultar'
               AND d ->> 'accion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.consultar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'bolsa'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'consulta_participaciones_propias'
           )
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_borradores_llamamiento_interno')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_situacion_participacion')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.registrar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_contactos_participacion')
 OR (p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_contactos_participacion')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'datos_contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.datos_contacto_participacion.registrar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_datos_contacto_participacion')
 OR (p_perfil_mutacion IS NOT DISTINCT FROM 'datos_contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.datos_contacto_participacion.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.datos_contacto_participacion.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_datos_contacto_participacion'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'emision_llamamiento_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.llamamiento.emitir.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'llamamiento.emitir.v1'
 AND d->>'accion' IS NOT DISTINCT FROM 'llamamiento.emitir.v1'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_llamamientos_bolsa')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_relacion_propia_dietas_personal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.relacion_propia.consultar_dietas.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.relacion.propia.consultar_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM 'personal.relacion.propia.consultar_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'relacion_empleado_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_borrador_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'crear_borrador_propio_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.crear.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'dietas.borrador.propio.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'crear_borrador_propio')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consultar_borrador_propio_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'dietas.borrador.propio.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_borrador_propio')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'acceso_rutas_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas_rutas_v1.acceso.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_itinerario_dietas'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion' IS NOT DISTINCT FROM 'dietas.ruta.catalogo.consultar' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_rutas_dietas')
   OR (d->>'accion' IS NOT DISTINCT FROM 'dietas.ruta.calculo.solicitar' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'calculo_rutas_dietas')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_organizacion_historica_personal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.organizacion_historica.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.organizacion_historica.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'personal.organizacion_historica.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'organizacion_historica'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_organizacion_historica'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["dotaciones","plazas","puestos_individuales","puestos_tipo","unidades","vinculos"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'importacion_organizacion_historica_personal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.organizacion_historica.importar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'importacion_organizacion_historica'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND (
   (d->>'accion' IS NOT DISTINCT FROM 'personal.organizacion_historica.preparar'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_organizacion_historica'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["hechos","manifiesto","recibo"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'personal.organizacion_historica.conciliar'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'conciliar_organizacion_historica'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["decisiones","recibo"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
 ))
           OR (
 p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar')
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND (
   (p_perfil_mutacion='cronos_marcaje_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.marcaje_propio.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.marcaje.propio.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'marcaje_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_marcaje_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_marcaje_remoto_disponibilidad'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.marcaje_remoto_disponibilidad.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.marcaje.remoto.disponibilidad.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'marcaje_remoto_disponibilidad'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_disponibilidad_marcaje_remoto'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["autorizado","continuidad_confirmada","motivo","movimientos_permitidos","periodo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_marcaje_remoto_recibo'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.marcaje_remoto_recibo.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.marcaje.remoto.recibo.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'marcaje_remoto_recibo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'recuperar_recibo_marcaje_remoto'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_saldo_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.saldo_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.saldo.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'saldo_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_saldo_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["detalle","periodo","resumen"]'::jsonb)
   OR (p_perfil_mutacion='cronos_movimientos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.movimientos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.movimientos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'movimientos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_movimientos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["absentismos","calendario","correcciones","periodo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_solicitar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion_propia.solicitar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.solicitar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correccion_marcaje'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_correccion_marcaje'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permisos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permisos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permisos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'permisos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_permisos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","pendientes","resumen"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permiso_solicitar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permiso_propio.solicitar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permiso.solicitar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_permiso'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_permiso_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
 ))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documento_dietas_mutacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.editar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.editar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'editar_borrador_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.borrar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.borrar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'borrar_borrador_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.enviar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.enviar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'enviar_borrador_propio')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documento_dietas_consulta'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.propio.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento_propio.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_documento_propio_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_borrador_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correccion_asignacion_dietas_personal'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.corregir' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.corregir.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'corregir_asignacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.grupo_corregir' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.grupo_corregir.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'corregir_grupo_dieta')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'alta_inicial_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.registrar_inicial'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.registrar_inicial.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_asignacion_dietas_inicial')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'prelectura_dietas'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.circuito.preleer'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.circuito.preleer.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preleer_competencia_circuito_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'circuito_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.revisar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.revisar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.autorizar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.autorizar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'autorizar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.liquidar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.liquidar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'liquidar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.fiscalizar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.fiscalizar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'fiscalizar_documento_dietas')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'bandeja_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bandeja_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.revision.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.revision.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_revision_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.autorizacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.autorizacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_autorizacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.liquidacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.liquidacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_liquidacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.fiscalizacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.fiscalizacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_fiscalizacion_dietas')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'competencias_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.competencias_consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.competencias.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas_competencias'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'tramitar_dietas_asignadas'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'rectificacion_dietas_personal'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.solicitar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.solicitar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_rectificacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.propia.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_rectificacion_dietas_propia')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.resolver' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.resolver.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'resolver_rectificacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.competente.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificaciones_competentes_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_rectificaciones_dietas_competentes')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'revisor_documento_dietas'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.circuito.documento.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.circuito.documento.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_documento_circuito_dietas'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.alta.registrar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.alta.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'alta_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.hecho.registrar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.hecho.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_hecho_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.ficha.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.ficha.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'registro_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_ficha_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","eficacia_administrativa","empleado_ref","evidencia","firma_oficial","ocupaciones","organismo_ref","persona_ref","relaciones","servicios","situaciones","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.vacantes.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.vacantes.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vacantes_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_vacantes'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cobertura","corte","cursor","cursor_siguiente","evidencia","limite","organismo_ref","vacantes"]'::jsonb)))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.catalogo.publicar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.catalogo.publicar.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'entrada_catalogo_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'gobernar_catalogo_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["entrada","recibo"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.catalogo.retirar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.catalogo.retirar.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'entrada_catalogo_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'gobernar_catalogo_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["entrada","recibo"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.catalogo.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.catalogo.consultar.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_catalogo_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cursor_siguiente","entradas","evidencia","organismo_ref"]'::jsonb)))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.empleados.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.empleados.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'empleados_rrhh'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_empleados'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","cursor","cursor_siguiente","empleados","evidencia","limite","organismo_ref"]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'cese_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cese.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.cesar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cese_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_cese_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'cierre_expediente_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cierre_expediente.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.expediente.cerrar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cierre_expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'cerrar_expediente_tras_cese'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'modificacion_tras_nombramiento_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.modificacion_tras_nombramiento.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.expediente.modificar_tras_nombramiento'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'modificacion_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'modificar_expediente_tras_nombramiento'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND ((((c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_pausa'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.solicitar_reactivacion'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1')
     OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.responder_llamamiento'
         AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1'))
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.manifestar_disposicion'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'oferta_bolsa'))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'firma_documento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_documento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'firma_documento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.confirmar_contacto'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_documentos.operacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND (
   (d->>'accion' IS NOT DISTINCT FROM 'documentos.generado.alta'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_generado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'alta_documento_generado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.expediente.listar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_documental'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'listar_documentos_expediente'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["items","siguiente_cursor"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.original.descargar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_original'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'descargar_documento_original'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["contenido","documento"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.notificacion.preparar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'notificacion_preparada'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_notificacion'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["preparacion","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.externo.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_externo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_documento_externo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.firmado.custodiar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_firmado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_documento_firmado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento_firmado.custodia","evidencia_custodia"]'::jsonb)
 ))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'cancelacion_expediente_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.cancelacion_expediente.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.expediente.cancelar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cancelacion_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'cancelar_expediente_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'confirmacion_ginpix_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmacion_ginpix.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.ginpix.confirmar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'confirmacion_ginpix_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'confirmar_ginpix_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_centro_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmar_alta_atestada.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion.confirmar_centro','contratacion_temporal.incorporacion.consultar_centro'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incorporacion_centro_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'no_incorporacion_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.no_incorporacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.incorporacion.no_incorporacion'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'no_incorporacion_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_no_incorporacion_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.historial_propio.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.mi-bolsa.historial.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_historial_propio'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_auditoria_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.auditoria.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'auditoria'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'historial_auditoria'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->>'finalidad' ~ '^[a-z][a-z0-9_]{0,127}$'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'reincorporacion_titular_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.reincorporacion_titular.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.registrar_reincorporacion_titular'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'reincorporacion_titular_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_reincorporacion_titular'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'politica_ofertas_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.politica_ofertas.publicar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_ofertas.publicar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_politica_ofertas_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'catalogo_plantillas_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.plantillas_documentos.consultar','contratacion_temporal.plantillas_documentos.editar','contratacion_temporal.plantillas_documentos.publicar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_plantillas_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_catalogo_plantillas_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='contratacion_temporal.plantillas_documentos.consultar'
      THEN '["borrador","editor_de_esta_version","publicado"]'::jsonb ELSE '["catalogo","recibo"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_cese_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.politica_cese.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_cese.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'politica_cese_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'politica:bolsa:cese:vigente'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_politica_cese_rrhh'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["politica_cese"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'catalogo_plantillas_documental_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas_documental.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.plantillas_documentos.documental_listar','contratacion_temporal.plantillas_documentos.documental_descargar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_plantillas_documental_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_borradores_expediente'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","catalogo_huella_sha256","contenido_json_sha256","procedencia_ref","revision","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_ofertas_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.politica_ofertas.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_ofertas.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_politica_ofertas_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["politica_ofertas"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_reincorporacion_titular_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.lectura_reincorporacion_titular.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.consultar_reincorporacion_titular'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'lectura_reincorporacion_titular_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'verificar_antecedente_reincorporacion_titular'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cese_evento_ref","cese_recibo_ref","documento_ref","documento_sha256","existe_cese","fecha_efectiva","relacion_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_reincorporacion_titular_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.reincorporacion_titular.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_reincorporacion_titular'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["reincorporaciones_titular"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_recibo_respuesta_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.consultar_recibo'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'respuesta_recibida_comunicacion_ct'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM
                   '["auditoria_ref","comunicacion_ref","estado","expediente_ref","justificante_ref","organizacion_ref","recibo_ref","registrada_en","respuesta"]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_comunicaciones_expediente_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.comunicaciones_expediente.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.llamamiento.comunicaciones.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["antecedente_tipo","comunicacion_ref","estado","estado_respuesta","expediente_ref","llamamiento_ref","organizacion_ref","recibo_antecedente_ref","recibo_comunicacion_ref","registrada_en","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_consulta_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.preferencias.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'preferencias_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","valores","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_actualizacion_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.preferencias.actualizar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'preferencias_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["valores","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_consultar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.consultar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.consultar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["activo","correo_ref","direccion","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_anadir_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.anadir.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.anadir.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.anadir'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["correo_ref","direccion","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_reenviar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.reenviar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.reenviar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.reenviar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["correo_ref","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_verificar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.verificar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.verificar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.verificar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["correo_ref","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_activar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.activar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.activar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.activar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["activo","correo_ref","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correos_retirar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.retirar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.retirar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.correos.retirar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'correos_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:correos-propios:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["activo","correo_ref","estado","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'imagen_consultar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.consultar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.consultar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.imagen.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'imagen_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:imagen-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["foto","icono","modo","paleta","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'imagen_actualizar_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.actualizar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.imagen.actualizar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.imagen.actualizar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'imagen_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:imagen-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["foto","icono","modo","paleta","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correo_avisos_llamamiento_usuarios'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'llamamiento.emitir.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bolsa_constituida'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_llamamientos_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'aspirantes_ficha_consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_aspirantes.ficha.consultar.externa_personal.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.aspirantes.ficha.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'aspirantes'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_aspirante_propia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'aspirantes_ficha_alta'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_aspirantes.ficha.alta.externa_personal.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.aspirantes.ficha.alta'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'aspirantes'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_aspirante_propia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'aspirantes_ficha_rectificar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_aspirantes.ficha.rectificar.externa_personal.v1'
 AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.aspirantes.ficha.rectificar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'aspirantes'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_aspirante_propia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["codigo_postal","domicilio","movil","telefono","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'ajustes_reglas_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.ajustes_reglas.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.reglas.consultar_ajustes','contratacion_temporal.reglas.ajustar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_reglas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_reglas_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'vec.contratacion_temporal.reglas'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='contratacion_temporal.reglas.consultar_ajustes'
      THEN '["historial","vigente"]'::jsonb ELSE '["ajustes","recibo"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_firmas_documento_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firmas_documento.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmas.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["CatalogoHuella","CatalogoRef","ClaveIdempotencia","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FirmaRef","FirmadoHuella","OriginalHuella","PasoOrden","PasoRef","ReciboRef","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_categorias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.lectura_categorias.v1'
 AND c->>'operacion' = ANY (ARRAY['vec.catalogos.categorias.listar_habilitadas',
                                   'vec.catalogos.categorias.consultar_historica',
                                   'vec.catalogos.categorias.consultar_uso'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='vec.catalogos.categorias.consultar_uso'
       THEN 'uso_categoria' ELSE 'catalogo_configurable' END
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_categorias_rpt'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE c->>'operacion'
       WHEN 'vec.catalogos.categorias.listar_habilitadas' THEN '["categorias","paginacion","publicaciones"]'::jsonb
       WHEN 'vec.catalogos.categorias.consultar_historica' THEN '["control_actual","entrada","publicacion"]'::jsonb
       ELSE '["uso"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'usos_categorias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.usos_categorias.v1'
 AND c->>'operacion' = ANY (ARRAY['vec.catalogos.categorias.reservar_uso',
                                   'vec.catalogos.categorias.confirmar_uso',
                                   'vec.catalogos.categorias.cancelar_uso'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'uso_categoria'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'vincular_categoria_a_operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo","uso"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_categoria_rpt_ct'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.categoria_rpt.vinculo.consultar','contratacion_temporal.categoria_rpt.vinculo.registrar'])
 AND ((c->>'operacion'='contratacion_temporal.categoria_rpt.vinculo.consultar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1'
       AND d->'campos_permitidos'='["analisis","vinculo"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.categoria_rpt.vinculo.registrar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1'
       AND d->'campos_permitidos'='["recibo"]'::jsonb))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_categoria_rpt_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_vinculo_categoria_rpt_ct'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_persona_aceptacion_ct_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.aceptacion_ct.persona.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.aceptacion_ct.persona.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_aceptacion_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_incorporacion_ct'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["aceptacion","persona","vinculo"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_anclaje_aceptacion_ct_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.aceptacion_ct.anclaje.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'anclaje_aceptacion_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_incorporacion_ct'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["anclaje"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.plan_incorporacion_ct.v1'
 AND c->>'operacion' IN ('personal.plan_incorporacion_ct.preparar','personal.plan_incorporacion_ct.consultar','personal.plan_incorporacion_ct.ejecutar','personal.plan_incorporacion_ct.confirmar','personal.plan_incorporacion_ct.seleccionar','personal.plan_incorporacion_ct.clases_ocupacion')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_incorporacion_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_incorporacion_ct'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM (CASE c->>'operacion' WHEN 'personal.plan_incorporacion_ct.seleccionar' THEN '["evidencia","seleccion"]'::jsonb WHEN 'personal.plan_incorporacion_ct.clases_ocupacion' THEN '["catalogo","evidencia"]'::jsonb ELSE '["ejecucion_huella_sha256","ejecucion_recibo_ref","estado","evidencia","plan","recibo_alta_relacion","recibo_ocupacion"]'::jsonb END)
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_personal_ct'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion_personal.plan.consultar','contratacion_temporal.incorporacion_personal.plan.registrar','contratacion_temporal.incorporacion_personal.origen.confirmar'])
 AND ((c->>'operacion'='contratacion_temporal.incorporacion_personal.plan.consultar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1'
       AND d->'campos_permitidos'='["plan"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.incorporacion_personal.plan.registrar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1'
       AND d->'campos_permitidos'='["recibo"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.incorporacion_personal.origen.confirmar' AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1' AND d->'campos_permitidos'='["recibo"]'::jsonb))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incorporacion_personal_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'incorporar_personal_desde_ct'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.presentar_solicitud_documental'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.solicitudes_documentales.resolver'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.solicitudes_documentales.resolver.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_documental_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_situacion_participacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.solicitudes_documentales.consultar_rrhh'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.solicitudes_documentales.consultar_rrhh.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_situacion_participacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'ct_circuito_consultar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.circuito.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.circuito.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_circuito_rrhh'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["circuito"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'firma_externa_documento_ct'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firma_externa.registrar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_externa.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'firma_externa_documento_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'version_rol_ref' IS NOT DISTINCT FROM 'rol:firma_externa_registro_ct_desarrollo:v1'
 AND d->>'recurso_ref' IS NOT NULL
 AND d->>'recurso_ref' ~ '^operacion-firma-externa-ct:[A-Za-z0-9][A-Za-z0-9._-]{15,63}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_documentos.operacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_original_firmable'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_original_firmable'
 AND nullif(d->>'version_rol_ref','') IS NOT NULL
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
 AND d->>'recurso_ref' IS DISTINCT FROM ('ref:'||repeat('0',64))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion' IS NOT DISTINCT FROM 'documentos.original_firmable.reservar'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["reserva","intento"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.original_firmable.confirmar'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)))

           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'firma_vec_documento_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firma_vec.registrar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_vec.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'firma_vec_documento_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'firma_externa_documento_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firma_externa.registrar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firma_externa.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'firma_externa_documento_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'recuperacion_firmas_r5_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.recuperar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.recuperar.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["ByteRange","CanonNominal","CanonNominalRef","CanonNominalSHA256","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","MaterialRootSHA256","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_plan_nominal_firma_ct'
 AND (c->>'operacion' IN ('vec.catalogos.crear','vec.catalogos.actualizar','vec.catalogos.publicar','vec.catalogos.retirar')) IS TRUE
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_configurable'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND (d->>'recurso_ref' ~ '^[a-z][a-z0-9._-]{2,127}:[1-9][0-9]{0,9}$') IS TRUE
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_firmas_r5_ct_v2'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmas_r5_v2.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT NULL
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["ByteRange","CatalogoHuella","CatalogoRef","CertificadoHuella","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","ContenidoFirmadoHuellaSHA256","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","EntradaDocumentoHuella","EntradaDocumentoLongitud","EntradaDocumentoRef","EntradaDocumentoVersion","EvidenciaFirmasCanonica","EvidenciaFirmasHuellaSHA256","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaAnteriorRef","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","FirmanteRef","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OrdenFirmaPDF","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboAnteriorRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","RevisionHuellaSHA256","RevisionLongitud","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_firmas_r5_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.firmas_r5.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.documento.firmas_r5.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["CatalogoHuella","CatalogoRef","ClaveIdempotencia","CoincideFirmanteCandidato","CoincideFirmanteEnOtroPaso","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FechaPortafirmasDeclarada","FirmaRef","FirmadoHuella","FirmantePrincipalAcreditado","HistoriaHuella","HistoriaRevision","HistoriaSeparacionAcreditada","OriginalHuella","OriginalRef","OriginalVersion","PasoOrden","PasoRef","ReciboRef","ReferenciaPortafirmasDeclarada","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado","Via"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'publicacion_certificado_nominal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contexto_actor.certificado_nominal.publicar.v1'
 AND c->>'operacion' IN ('administracion.certificados.nominal.publicar','administracion.certificados.nominal.retirar')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_certificado_nominal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_certificados_firmantes'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','administracion','vinculo_certificado_nominal','gestionar_certificados_firmantes',
   '[]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'publicacion_cargo_competencial'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.cargo_competencial.publicar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.cargo_competencial.publicar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'cargo_competencial'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'administrar_cargos_competenciales'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cargo","enlace","huella_sha256","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','personal','cargo_competencial','administrar_cargos_competenciales',
   '["cargo","enlace","huella_sha256","recibo","version"]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'capacidades_admin'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lectura.capacidades.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.perfiles.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'perfil'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_perfiles'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','administracion','perfil','gestion_perfiles',
   '["actos_disponibles","perfil_ref","preimagen","version","vinculo_ref"]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'persona_denominacion_publicar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.persona.denominacion.publicar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.persona.denominacion.publicar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_denominacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentacion_persona'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["denominacion"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'persona_denominacion_leer'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.persona.denominacion.leer'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.persona.denominacion.leer.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_denominacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentacion_persona'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["nombre_mostrar"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'usuarios_admin_listar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.usuarios.listar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.admin.usuarios.listar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'conjunto_usuarios'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_usuarios'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^conjunto_admin:[0-9a-f]{32}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["denominacion_version","perfiles","persona_ref","siguiente_cursor","unidad_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'usuarios_admin_consultar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.usuarios.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.admin.usuarios.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_administrable'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_usuarios'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["denominacion_version","perfiles","persona_ref","unidad_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'servicios_certificados_propios'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.servicios_certificados.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.servicios_certificados.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'servicios_certificados'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_servicios_para_certificados'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^emp_[A-Za-z0-9_-]{22,128}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cobertura","corte","evidencia","servicios"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'lote_perfiles_admin'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.perfiles.aplicar_lote_ordinario'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,128}$'
 AND d->>'recurso_ref' IS DISTINCT FROM d->>'principal_id'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles',
   '[]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.ficha_propia.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.ficha_propia.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'ficha_propia_empleado'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_ficha_propia'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","evidencia","relaciones","servicios"]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'exportacion_servicios_propios'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.ficha_propia.servicios.exportar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.ficha_propia.servicios.exportar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'exportacion_servicios_propios'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'exportar_servicios_propios'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^emp_[A-Za-z0-9_-]{22,128}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","evidencia","servicios"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'historia_servicios_propia'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.servicios.historia_propia.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.servicios.historia_propia.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'historia_servicios_propia'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_historia_servicios_propios'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^emp_[A-Za-z0-9_-]{22,128}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cobertura","corte","evidencia","revisiones"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_rrhh_bolsa'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND (
   (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.rrhh.bolsas.consultar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'coleccion_bolsas_rrhh'
    AND d->>'recurso_ref' IS NOT DISTINCT FROM 'coleccion:bolsa:rrhh:bolsas'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_rrhh_bolsas'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["bolsas","conteos"]'::jsonb)
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.rrhh.estadisticas.consultar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'estadisticas_bolsas_rrhh'
    AND d->>'recurso_ref' IS NOT DISTINCT FROM 'coleccion:bolsa:rrhh:estadisticas'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_rrhh_estadisticas'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["estadisticas"]'::jsonb)
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.rrhh.candidatos.consultar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'consulta_candidatos_bolsa'
    AND (CASE WHEN octet_length(d->>'recurso_ref') BETWEEN 79 AND 200
    THEN d->>'recurso_ref' ~ '^bolsa:[A-Za-z0-9_-]+(:[A-Za-z0-9_-]+)*:filtro:[a-f0-9]{64}$'
    ELSE false END)
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_rrhh_candidatos'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["candidatos","contactos","turno"]'::jsonb)))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_rpt_publica_v2_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.rpt_publica.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.rpt_publica.consultar.v2'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rpt_publica_publicacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_organizacion_publicada'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^rpt-publicada:[a-z0-9][a-z0-9:-]{2,127}$'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["categorias_pendientes_grupo","corte","esquema","estado","evidencia","fuente","huella_sha256","items","limit","offset","publicacion_ref","resumen","total","vista"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
       OR c ->> 'nonce' !~ '^[0-9a-f]{64}$'
       OR c ->> 'nonce' = pg_catalog.repeat('0', 64)
       OR c ->> 'clave_version' !~ '^[1-9][0-9]{0,19}$'
       OR c ->> 'revision_gobierno' !~ '^[1-9][0-9]{0,19}$'
       OR c ->> 'configuracion_secuencia' !~ '^[1-9][0-9]{0,19}$'
       OR c ->> 'raiz_version' !~ '^[1-9][0-9]{0,19}$'
       OR (c ->> 'clave_version')::numeric >
          9007199254740991::numeric
       OR (c ->> 'revision_gobierno')::numeric >
          9007199254740991::numeric
       OR (c ->> 'configuracion_secuencia')::numeric >
          9007199254740991::numeric
       OR (c ->> 'raiz_version')::numeric >
          9007199254740991::numeric
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.unnest(ARRAY[
                 c ->> 'huella_gobierno_sha256',
                 c ->> 'huella_decision_sha256',
                 c ->> 'huella_motivo_sha256',
                 c ->> 'huella_payload_vec_ad_3_sha256',
                 c ->> 'huella_sobre_cose_sign1_sha256',
                 c ->> 'huella_prueba_confianza_sha256',
                 c ->> 'huella_contexto_sha256',
                 c ->> 'huella_efecto_sha256',
                 c ->> 'huella_configuracion_sha256',
                 c ->> 'huella_raiz_spki_sha256',
                 c ->> 'mac_sha256'
             ]) AS h(valor)
            WHERE vec_autorizacion_atestada_v3.huella_sha256_valida(
                      h.valor) IS NOT TRUE)
       OR x ->> 'esquema' <> 'vec.contexto-actor.vinculado.v2'
       OR x ->> 'persona_version' <> p_persona_version::text
       OR x ->> 'perfil_version' <> p_perfil_version::text
       OR pg_catalog.encode(
           pg_catalog.sha256(p_contexto_actor_canonico), 'hex') <> c ->> 'huella_contexto_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_decision_canonica), 'hex') <> c ->> 'huella_decision_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_motivo_canonico), 'hex') <> c ->> 'huella_motivo_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_payload_vec_ad_3), 'hex') <> c ->> 'huella_payload_vec_ad_3_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_sobre_cose_sign1), 'hex') <> c ->> 'huella_sobre_cose_sign1_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_evidencia_verificacion), 'hex') <> c ->> 'huella_prueba_confianza_sha256'
       OR pg_catalog.encode(
           pg_catalog.sha256(p_raiz_publica_spki), 'hex') <> c ->> 'huella_raiz_spki_sha256'
       OR d ->> 'decision_ref' <> c ->> 'decision_ref'
       OR d ->> 'motivo_huella_sha256' <> c ->> 'huella_motivo_sha256'
       OR d ->> 'accion' <> c ->> 'operacion'
       OR d ->> 'recurso_ref' <> c ->> 'efecto_ref'
       OR d ->> 'contexto_recurso_huella_sha256' <>
          c ->> 'huella_efecto_sha256'
       OR (d ->> 'valida_hasta')::timestamptz <> (c ->> 'decision_valida_hasta')::timestamptz
       OR d #>> '{vinculo_autenticacion_actor,registro_contexto_ref}' <>
          c ->> 'contexto_ref'
       OR d #>> '{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' <>
          c ->> 'huella_contexto_sha256'
       OR d ->> 'principal_id' <> x ->> 'principal_ref'
       OR d ->> 'perfil_activo_ref' <> x ->> 'perfil_activo_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'ligadura VEC-AD-3 inválida';
    END IF;
    IF p_perfil_mutacion IN ('preferencias_consulta_usuarios','preferencias_actualizacion_usuarios','correos_consultar_usuarios','correos_anadir_usuarios','correos_reenviar_usuarios','correos_verificar_usuarios','correos_activar_usuarios','correos_retirar_usuarios','imagen_consultar_usuarios','imagen_actualizar_usuarios','correo_avisos_llamamiento_usuarios')
       AND NOT (
         (d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
          AND pg_catalog.pg_has_role(session_user,'vec_usuarios_ejecutor_interno','MEMBER')
          AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_usuarios_ejecutor_interno'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
          AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
         OR
         (d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
          AND pg_catalog.pg_has_role(session_user,'vec_usuarios_ejecutor_externo','MEMBER')
          AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_usuarios_ejecutor_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
          AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
       ) THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo Usuarios rechazado';
    END IF;
    IF p_perfil_mutacion IN ('aspirantes_ficha_consultar','aspirantes_ficha_alta','aspirantes_ficha_rectificar')
       AND NOT (
         d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
         AND pg_catalog.pg_has_role(session_user,'vec_aspirantes_ejecutor_externo','MEMBER')
         AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_aspirantes_ejecutor_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
         AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
       ) THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo Aspirantes rechazado';
    END IF;
    v_huella_capacidad := pg_catalog.encode(
        pg_catalog.sha256(p_capacidad_canonica), 'hex');
    PERFORM 1
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'gobierno VEC-AD-3 no disponible';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_autorizacion_atestada_v3:decision:' ||
            (c ->> 'decision_ref'), 0));
    SELECT a.decision_ref, a.efecto_ref, a.huella_efecto_sha256,
           a.consumo_huella_sha256, u.auditoria_ref, a.consumida_en,
           t.capacidad_canonica, t.decision_canonica, t.motivo_canonico,
           t.contexto_actor_canonico, t.payload_vec_ad_3,
           t.sobre_cose_sign1, t.evidencia_verificacion,
           t.raiz_publica_spki
      INTO v_replay
      FROM vec_autorizacion_atestada_v3.consumo_decision_v3 a
      JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t
        USING (decision_ref)
      JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u
        USING (decision_ref)
     WHERE a.decision_ref = c ->> 'decision_ref'
        OR a.nonce = c ->> 'nonce';
    IF FOUND THEN
        IF v_replay.decision_ref <> c ->> 'decision_ref'
           OR v_replay.efecto_ref <> c ->> 'efecto_ref'
           OR v_replay.huella_efecto_sha256 <>
              c ->> 'huella_efecto_sha256'
           OR v_replay.capacidad_canonica <> p_capacidad_canonica
           OR v_replay.decision_canonica <> p_decision_canonica
           OR v_replay.motivo_canonico <> p_motivo_canonico
           OR v_replay.contexto_actor_canonico <>
              p_contexto_actor_canonico
           OR v_replay.payload_vec_ad_3 <> p_payload_vec_ad_3
           OR v_replay.sobre_cose_sign1 <> p_sobre_cose_sign1
           OR v_replay.evidencia_verificacion <>
              p_evidencia_verificacion
           OR v_replay.raiz_publica_spki <> p_raiz_publica_spki THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'conflicto VEC-AD-3';
        END IF;
        RETURN QUERY SELECT
            v_replay.decision_ref, v_replay.efecto_ref,
            v_replay.huella_efecto_sha256,
            v_replay.consumo_huella_sha256, v_replay.auditoria_ref,
            v_replay.consumida_en, false;
        RETURN;
    END IF;
    -- AD172: configuración técnica; no amplía la autorización de negocio.
    -- Se exige sólo al consumo nuevo; la autoridad viva original valida después
    -- el vínculo canónico antes de insertar atestación, consumo y auditoría.
    v_canal_origen := d #>> '{vinculo_autenticacion_actor,superficie}';
    v_proceso_origen := vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(
        c ->> 'audiencia_consumo', c ->> 'operacion', v_canal_origen);
    IF v_proceso_origen IS NULL THEN
        -- AD208: falta de configuración técnica, no denegación. Sin LOGIN.
        RAISE EXCEPTION USING ERRCODE='VA172', MESSAGE='origen de consumo no acreditado',
            DETAIL=pg_catalog.format('audiencia=%s operacion=%s canal=%s',
                c ->> 'audiencia_consumo', c ->> 'operacion', v_canal_origen),
            HINT='falta su fila en configuracion_origen_consumos_v1 o el LOGIN de la sesión no es el de la fila';
    END IF;
    v_ahora := clock_timestamp();
    SELECT k.* INTO v_clave
      FROM vec_autorizacion_atestada_v3.clave_capacidad_version k
     WHERE k.clave_id = c ->> 'clave_id'
       AND k.version = (c ->> 'clave_version')::numeric
     FOR SHARE;
    SELECT p.* INTO v_puntero_clave
      FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
     WHERE p.clave_id = c ->> 'clave_id'
       AND p.version = (c ->> 'clave_version')::numeric
       AND p.establecida_en <= v_ahora
     ORDER BY p.orden DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND
       OR v_puntero_clave.clave_id IS NULL
       OR v_clave.revision_gobierno <>
          (c ->> 'revision_gobierno')::numeric
       OR v_clave.huella_gobierno_sha256 <>
          c ->> 'huella_gobierno_sha256'
       OR v_clave.emisor_id <> c ->> 'emisor_id'
       OR v_clave.audiencia_consumo <> c ->> 'audiencia_consumo'
       OR (c ->> 'emitida_en')::timestamptz < v_clave.valida_desde
       OR (c ->> 'expira_en')::timestamptz > v_clave.valida_hasta
       OR v_ahora < (c ->> 'emitida_en')::timestamptz
       OR v_ahora >= (c ->> 'expira_en')::timestamptz
       OR (c ->> 'expira_en')::timestamptz <=
          (c ->> 'emitida_en')::timestamptz
       OR (c ->> 'expira_en')::timestamptz >
          (c ->> 'emitida_en')::timestamptz + interval '5 seconds'
       OR v_ahora >= (c ->> 'decision_valida_hasta')::timestamptz
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version
              AND r.revocada_en <= v_ahora)
       OR vec_autorizacion_atestada_v3.bytea_igual_constante(
           public.hmac(
               vec_autorizacion_atestada_v3.preimagen_mac(c),
               v_clave.secreto_hmac,
               'sha256'),
           pg_catalog.decode(c ->> 'mac_sha256', 'hex')) IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'capacidad VEC-AD-3 rechazada';
    END IF;
    SELECT cfg.*, cp.configuracion_secuencia_minima,
           cp.raiz_version_minima
      INTO v_config
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
      JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version cfg
        ON cfg.revision = p.configuracion_revision
      JOIN vec_autorizacion_atestada_v3.checkpoint_gobierno cp
        ON cp.control_id
     WHERE p.establecida_en <= v_ahora
     ORDER BY p.orden DESC
     LIMIT 1
     FOR SHARE OF p, cfg;
    SELECT r.* INTO v_raiz
      FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
      JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
        ON r.clave_id = cr.raiz_clave_id
       AND r.version = cr.raiz_version
     WHERE cr.configuracion_revision = v_config.revision
       AND r.clave_id = c ->> 'raiz_clave_id'
       AND r.version = (c ->> 'raiz_version')::numeric
     FOR SHARE OF r;
    IF v_config.revision IS NULL OR v_raiz.clave_id IS NULL
       OR v_config.revision <> c ->> 'revision_confianza'
       OR v_config.secuencia <> (c ->> 'configuracion_secuencia')::numeric
       OR v_config.secuencia < v_config.configuracion_secuencia_minima
       OR v_config.huella_configuracion_sha256 <>
          c ->> 'huella_configuracion_sha256'
       OR v_config.publicada_en <>
          (c ->> 'configuracion_publicada_en')::timestamptz
       OR v_config.expira_en <>
          (c ->> 'configuracion_expira_en')::timestamptz
       OR v_raiz.version < v_config.raiz_version_minima
       OR v_raiz.huella_spki_sha256 <> c ->> 'huella_raiz_spki_sha256'
       OR v_raiz.clave_publica_spki <> p_raiz_publica_spki
       OR v_raiz.valida_desde <> (c ->> 'raiz_valida_desde')::timestamptz
       OR v_raiz.valida_hasta <> (c ->> 'raiz_valida_hasta')::timestamptz
       OR v_raiz.suite <> c ->> 'suite'
       OR v_raiz.audiencia_despliegue <> c ->> 'audiencia_despliegue'
       OR (c ->> 'verificada_en')::timestamptz <
          v_config.publicada_en
       OR (c ->> 'verificada_en')::timestamptz >= v_config.expira_en
       OR (c ->> 'verificada_en')::timestamptz < v_raiz.valida_desde
       OR (c ->> 'verificada_en')::timestamptz >= v_raiz.valida_hasta
       OR v_ahora >= v_config.expira_en OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
            WHERE r.configuracion_revision = v_config.revision
              AND r.revocada_en <= v_ahora)
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion_atestada_v3.revocacion_raiz r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'confianza VEC-AD-3 rechazada';
    END IF;
    SELECT * INTO v_registro
      FROM vec_autorizacion.
           registrar_y_revalidar_decision_contexto_actor_v3(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version);
    IF NOT FOUND OR v_registro.concedida IS NOT TRUE
       OR v_registro.decision_huella_sha256 <>
          c ->> 'huella_decision_sha256' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'decisión VEC-AD-3 rechazada';
    END IF;
    v_ahora := clock_timestamp();
    IF v_ahora >= (c ->> 'expira_en')::timestamptz
       OR v_ahora >= (c ->> 'decision_valida_hasta')::timestamptz
       OR v_ahora >= v_config.expira_en OR v_ahora >= v_raiz.valida_hasta
       OR EXISTS (
           SELECT 1 FROM
             vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
            WHERE r.clave_id = v_clave.clave_id
              AND r.version = v_clave.version AND r.revocada_en <= v_ahora)
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
            WHERE r.configuracion_revision = v_config.revision
              AND r.revocada_en <= v_ahora)
       OR EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r
            WHERE r.raiz_clave_id = v_raiz.clave_id
              AND r.raiz_version = v_raiz.version
              AND r.revocada_en <= v_ahora) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'vigencia VEC-AD-3 agotada';
    END IF;
    v_revalidada_en :=
      vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version);
    IF v_revalidada_en IS NULL
       OR v_revalidada_en < v_registro.revalidada_en THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'revalidación viva VEC-AD-3 rechazada';
    END IF;
    v_huella_consumo := pg_catalog.encode(pg_catalog.sha256(
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(p_capacidad_canonica, 'base64')) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(p_decision_canonica, 'base64')) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(p_contexto_actor_canonico, 'base64')) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(c ->> 'efecto_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'huella_efecto_sha256')), 'hex');
    INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3 (
        decision_ref, huella_decision_sha256, decision_canonica,
        motivo_canonico, contexto_actor_canonico, payload_vec_ad_3,
        sobre_cose_sign1, evidencia_verificacion, raiz_publica_spki,
        capacidad_canonica, huella_capacidad_sha256, efecto_ref,
        huella_efecto_sha256, registrada_en) VALUES (
        c ->> 'decision_ref', c ->> 'huella_decision_sha256',
        p_decision_canonica, p_motivo_canonico,
        p_contexto_actor_canonico, p_payload_vec_ad_3,
        p_sobre_cose_sign1, p_evidencia_verificacion,
        p_raiz_publica_spki, p_capacidad_canonica,
        v_huella_capacidad, c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_ahora);
    v_transaccion_origen := pg_catalog.pg_current_xact_id();
    INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3 (
        decision_ref, huella_decision_sha256, nonce, efecto_ref,
        huella_efecto_sha256, consumo_huella_sha256, consumida_en, transaccion_origen) VALUES (
        c ->> 'decision_ref', c ->> 'huella_decision_sha256',
        c ->> 'nonce', c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_huella_consumo, v_ahora, v_transaccion_origen);
    SELECT asiento_ad207.secuencia_previa, asiento_ad207.anterior_sha256 INTO STRICT v_secuencia, v_anterior FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() asiento_ad207;
    IF v_secuencia >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '22003',
            MESSAGE = 'límite de secuencia VEC-AD-3 alcanzado';
    END IF;
    v_secuencia := v_secuencia + 1;
    v_auditoria_ref := 'aud_v3_' ||
        pg_catalog.substr(v_huella_consumo, 1, 32);
    -- AD173: coordenadas de la decisión/contexto ya autenticados y revalidados.
    -- El instante guardado en ambas tablas procede del mismo v_ahora UTC6.
    v_actor_nominal := d ->> 'principal_id';
    v_perfil_nominal := d ->> 'perfil_activo_ref';
    v_finalidad_nominal := d ->> 'finalidad';
    v_fecha_nominal := pg_catalog.to_char(v_ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    v_preimagen_auditoria :=
        vec_autorizacion_atestada_v3.encuadrar_mac('consumo_confirmado_v4') ||
        vec_autorizacion_atestada_v3.encuadrar_mac('4') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'decision_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(c ->> 'efecto_ref') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            c ->> 'huella_efecto_sha256') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_huella_consumo) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_proceso_origen) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_canal_origen) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_fecha_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_fecha_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_actor_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_perfil_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_finalidad_nominal) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_transaccion_origen::text);
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 (
        auditoria_ref, secuencia, decision_ref, efecto_ref,
        huella_efecto_sha256, anterior_sha256, huella_sha256,
        registrada_en, tipo_registro, version_consumo, proceso, canal,
        actor_ref, perfil_activo_ref, finalidad_ref, transaccion_origen) VALUES (
        v_auditoria_ref, v_secuencia, c ->> 'decision_ref',
        c ->> 'efecto_ref', c ->> 'huella_efecto_sha256',
        v_anterior, pg_catalog.encode(
            pg_catalog.sha256(v_preimagen_auditoria), 'hex'), v_ahora,
        'consumo_confirmado_v4', 4, v_proceso_origen, v_canal_origen,
        v_actor_nominal, v_perfil_nominal, v_finalidad_nominal, v_transaccion_origen);
    NULL; /* AD207: la cabeza la avanza el sellado diferido */
    RETURN QUERY SELECT
        c ->> 'decision_ref', c ->> 'efecto_ref',
        c ->> 'huella_efecto_sha256', v_huella_consumo,
        v_auditoria_ref, v_ahora, true;
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow
      OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada VEC-AD-3 inválida';
END
$function$
;
DO $nucleo_post$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM 'e314eb6242dd0681dbd8cf5618e4287140831f56ab9faa8fe2222f89f0196773'
 OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f)
   IS DISTINCT FROM '1a8271745e5c0cb028ad91d105e69f31a13dd7e046aa31385405d54e5518c9f7'
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f)
   IS DISTINCT FROM current_setting('vec.ad217.pre_meta',true)::jsonb
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
   IS DISTINCT FROM current_setting('vec.ad217.pre_deps',true)::jsonb
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
   FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
    AND d.classid='pg_proc'::regclass AND d.objid=f)
   IS DISTINCT FROM current_setting('vec.ad217.pre_shared',true)::jsonb
 THEN RAISE EXCEPTION 'AD217: PARO postimagen de núcleo, metadatos o dependencias' USING ERRCODE='55000'; END IF;
END $nucleo_post$;

DO $audiencias_pre$
DECLARE anterior text;
BEGIN
 -- HZ AD216: esta sucesora exige explícitamente convalidated antes del cambio.
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
   '8dae0267b85ad237770d0ccdb7fbf9862afe6d7d022c0c93f4385c182bcbc68e'
 OR strpos(anterior,'vec_personal.rpt_publica.consultar.v2')<>0
 THEN RAISE EXCEPTION 'AD217: PARO CHECK preimagen/no validado' USING ERRCODE='55000'; END IF;
END $audiencias_pre$;
-- CHECK POST216 íntegro más la única audiencia RPT. Sin DDL generado.
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
 DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
 ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check
 CHECK ((((audiencia_consumo = ANY (ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1'::text, 'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1'::text, 'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'::text, 'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'::text, 'vec_personal.alta_ejercicio.v1'::text, 'vec_contratacion_temporal.incorporacion_ejercicio.v2'::text, 'vec_personal.lectura_incorporacion.v1'::text, 'vec_personal.lectura_incorporacion.v2'::text, 'vec_contratacion_temporal.anotacion_administrativa.v1'::text, 'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1'::text, 'vec_contratacion_temporal.despacho_correo_llamamiento.v1'::text, 'vec_contratacion_temporal.resultado_correo_llamamiento.v1'::text, 'vec.bolsa.mi-bolsa.v1'::text, 'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1'::text, 'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1'::text, 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1'::text, 'vec_bolsa_llamamientos.contacto_participacion.registrar.v1'::text, 'vec_bolsa_llamamientos.contacto_participacion.consultar.v1'::text, 'vec_bolsa_llamamientos.datos_contacto_participacion.registrar.v1'::text, 'vec_bolsa_llamamientos.llamamiento.emitir.v1'::text, 'vec_personal.relacion_propia.consultar_dietas.v1'::text, 'vec_dietas.borrador_propio.crear.v1'::text, 'vec_dietas.borrador_propio.consultar.v1'::text, 'vec_dietas_rutas_v1.acceso.v1'::text, 'vec_personal.organizacion_historica.consultar.v1'::text, 'vec_personal.organizacion_historica.importar.v1'::text, 'vec_cronos_v1.marcaje_propio.v1'::text, 'vec_cronos_v1.marcaje_remoto_disponibilidad.v1'::text, 'vec_cronos_v1.marcaje_remoto_recibo.v1'::text, 'vec_cronos_v1.saldo_propio.consultar.v1'::text, 'vec_dietas.borrador_propio.editar.v1'::text, 'vec_dietas.borrador_propio.borrar.v1'::text, 'vec_dietas.borrador_propio.enviar.v1'::text, 'vec_dietas.documento_propio.consultar.v1'::text, 'vec_personal.asignacion_dietas.consultar.v1'::text, 'vec_personal.asignacion_dietas.corregir.v1'::text, 'vec_personal.asignacion_dietas.grupo_corregir.v1'::text, 'vec_personal.asignacion_dietas.registrar_inicial.v1'::text, 'vec_dietas.documento.revisar.v1'::text, 'vec_dietas.documento.autorizar.v1'::text, 'vec_dietas.circuito.preleer.v1'::text, 'vec_dietas.documento.liquidar.v1'::text, 'vec_dietas.documento.fiscalizar.v1'::text, 'vec_dietas.bandeja.revision.consultar.v1'::text, 'vec_dietas.bandeja.autorizacion.consultar.v1'::text, 'vec_dietas.bandeja.liquidacion.consultar.v1'::text, 'vec_dietas.bandeja.fiscalizacion.consultar.v1'::text, 'vec_personal.asignacion_dietas.competencias.v1'::text, 'vec_personal.asignacion_dietas.rectificacion.solicitar.v1'::text, 'vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1'::text, 'vec_personal.asignacion_dietas.rectificacion.resolver.v1'::text, 'vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1'::text, 'vec_dietas.circuito.documento.consultar.v1'::text, 'vec_cronos_v1.movimientos_propio.consultar.v1'::text, 'vec_cronos_v1.correccion_propia.solicitar.v1'::text, 'vec_cronos_v1.permisos_propio.consultar.v1'::text, 'vec_cronos_v1.permiso_propio.solicitar.v1'::text, 'vec_personal.registro_empleado.alta.v1'::text, 'vec_personal.registro_empleado.hecho.v1'::text, 'vec_personal.registro_empleado.ficha.v1'::text, 'vec_personal.registro_empleado.vacantes.v1'::text, 'vec_personal.registro_empleado.catalogo.publicar.v1'::text, 'vec_personal.registro_empleado.catalogo.retirar.v1'::text, 'vec_personal.registro_empleado.catalogo.consultar.v1'::text, 'vec_personal.registro_empleado.empleados.v1'::text, 'vec_contratacion_temporal.cese.v1'::text, 'vec_contratacion_temporal.cierre_expediente.v1'::text, 'vec_contratacion_temporal.modificacion_tras_nombramiento.v1'::text, 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1'::text, 'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1'::text, 'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1'::text, 'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1'::text, 'vec_contratacion_temporal.firma_documento.v1'::text, 'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1'::text, 'vec_documentos.operacion.v1'::text, 'vec_contratacion_temporal.cancelacion_expediente.v1'::text, 'vec_contratacion_temporal.confirmacion_ginpix.v1'::text, 'vec_contratacion_temporal.no_incorporacion.v1'::text, 'vec.bolsa.mi-bolsa.historial.v1'::text, 'vec_auditoria.consulta_rrhh.v1'::text, 'vec_contratacion_temporal.reincorporacion_titular.v1'::text, 'vec_bolsa_llamamientos.politica_ofertas.publicar.v1'::text, 'vec_contratacion_temporal.catalogo_plantillas.v1'::text, 'vec_bolsa_llamamientos.politica_cese.consultar.v1'::text, 'vec_contratacion_temporal.catalogo_plantillas_documental.v1'::text, 'vec_bolsa_llamamientos.politica_ofertas.consultar.v1'::text, 'vec_contratacion_temporal.lectura_reincorporacion_titular.v1'::text, 'vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1'::text, 'vec_contratacion_temporal.comunicaciones_expediente.consultar.v1'::text, 'vec_usuarios.preferencias.consultar.interna_corporativa.v1'::text, 'vec_usuarios.preferencias.actualizar.interna_corporativa.v1'::text, 'vec_usuarios.preferencias.consultar.externa_personal.v1'::text, 'vec_usuarios.preferencias.actualizar.externa_personal.v1'::text, 'vec_usuarios.correos.consultar.interna_corporativa.v1'::text, 'vec_usuarios.correos.consultar.externa_personal.v1'::text, 'vec_usuarios.correos.anadir.interna_corporativa.v1'::text, 'vec_usuarios.correos.anadir.externa_personal.v1'::text, 'vec_usuarios.correos.reenviar.interna_corporativa.v1'::text, 'vec_usuarios.correos.reenviar.externa_personal.v1'::text, 'vec_usuarios.correos.verificar.interna_corporativa.v1'::text, 'vec_usuarios.correos.verificar.externa_personal.v1'::text, 'vec_usuarios.correos.activar.interna_corporativa.v1'::text, 'vec_usuarios.correos.activar.externa_personal.v1'::text, 'vec_usuarios.correos.retirar.interna_corporativa.v1'::text, 'vec_usuarios.correos.retirar.externa_personal.v1'::text, 'vec_usuarios.imagen.consultar.interna_corporativa.v1'::text, 'vec_usuarios.imagen.consultar.externa_personal.v1'::text, 'vec_usuarios.imagen.actualizar.interna_corporativa.v1'::text, 'vec_usuarios.imagen.actualizar.externa_personal.v1'::text, 'vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1'::text, 'vec_aspirantes.ficha.consultar.externa_personal.v1'::text, 'vec_aspirantes.ficha.alta.externa_personal.v1'::text, 'vec_aspirantes.ficha.rectificar.externa_personal.v1'::text, 'vec_contratacion_temporal.ajustes_reglas.v1'::text, 'vec_contratacion_temporal.firmas_documento.consultar.v1'::text, 'vec_catalogos_configurables.lectura_categorias.v1'::text, 'vec_catalogos_configurables.usos_categorias.v1'::text, 'vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1'::text, 'vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1'::text, 'vec_bolsa_llamamientos.aceptacion_ct.persona.v1'::text, 'vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1'::text, 'vec_personal.plan_incorporacion_ct.v1'::text, 'vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1'::text, 'vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1'::text, 'vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1'::text, 'vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1'::text, 'vec_bolsa_llamamientos.solicitudes_documentales.resolver.v1'::text, 'vec_bolsa_llamamientos.solicitudes_documentales.consultar_rrhh.v1'::text, 'vec_contratacion_temporal.circuito.consultar.v1'::text, 'vec_contratacion_temporal.firma_externa.v1'::text, 'vec_contratacion_temporal.firma_vec.v2'::text, 'vec_contratacion_temporal.firma_externa.v2'::text, 'vec_contratacion_temporal.firmas_r5.consultar.v2'::text, 'vec_contratacion_temporal.firmas_r5.consultar.v1'::text, 'vec_contexto_actor.certificado_nominal.publicar.v1'::text, 'vec_personal.cargo_competencial.publicar.v1'::text, 'vec_autorizacion.administracion_perfiles.lectura.capacidades.v1'::text])) OR (audiencia_consumo = ANY (ARRAY['vec.persona.denominacion.publicar.v1'::text, 'vec.persona.denominacion.leer.v1'::text])) OR (audiencia_consumo = ANY (ARRAY['vec.admin.usuarios.listar.v1'::text, 'vec.admin.usuarios.consultar.v1'::text])) OR (audiencia_consumo = 'vec_personal.servicios_certificados.v1'::text) OR (audiencia_consumo = ANY (ARRAY['vec_contratacion_temporal.firmas_r5.recuperar.v2'::text, 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'::text])) OR (audiencia_consumo = 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1'::text) OR (audiencia_consumo = 'vec_bolsa_llamamientos.datos_contacto_participacion.consultar.v1'::text) OR (audiencia_consumo = 'vec_contratacion_temporal.borrador_rrhh.descargar.v1'::text) OR (audiencia_consumo = 'vec_personal.registro_empleado.ficha_propia.v1'::text) OR (audiencia_consumo = 'vec_personal.registro_empleado.ficha_propia.servicios.exportar.v1'::text) OR (audiencia_consumo = 'vec_personal.registro_empleado.servicios.historia_propia.v1'::text) OR (audiencia_consumo = ANY (ARRAY['vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1'::text, 'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1'::text, 'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1'::text])))) OR audiencia_consumo = 'vec_personal.rpt_publica.consultar.v2');
DO $audiencias_post$
DECLARE actual text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT actual FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(actual,'vec_personal.rpt_publica.consultar.v2')=0
 OR strpos(actual,'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1')=0
 OR strpos(actual,'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1')=0
 OR strpos(actual,'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1')=0
 THEN RAISE EXCEPTION 'AD217: PARO CHECK postimagen/no validado' USING ERRCODE='55000'; END IF;
END $audiencias_post$;

-- AD172 conserva sólo el vínculo técnico objetivo. No crea el LOGIN, secretos,
-- perfiles humanos ni autorizaciones: su provisión corresponde al DBA.
INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 (login_nombre,audiencia_consumo,operacion,proceso,canal_permitido)
VALUES ('vec_personal_rpt_v2_app','vec_personal.rpt_publica.consultar.v2',
 'personal.rpt_publica.consultar','vec-server','interna_corporativa');
DO $origen$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE login_nombre='vec_personal_rpt_v2_app')<>1
 OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE login_nombre='vec_personal_rpt_v2_app'
   AND audiencia_consumo='vec_personal.rpt_publica.consultar.v2'
   AND operacion='personal.rpt_publica.consultar'
   AND proceso='vec-server' AND canal_permitido='interna_corporativa')
 THEN RAISE EXCEPTION 'AD217: PARO origen RPT no conservado' USING ERRCODE='55000'; END IF;
END $origen$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
 campos constant jsonb:='["categorias_pendientes_grupo","corte","esquema","estado","evidencia","fuente","huella_sha256","items","limit","offset","publicacion_ref","resumen","total","vista"]';
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_capacidad IS NULL OR vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
 OR p_persona_version IS NULL OR p_perfil_version IS NULL
 OR p_payload IS NULL OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
 OR p_sobre IS NULL OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
 OR p_evidencia IS NULL OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
 OR p_raiz IS NULL OR octet_length(p_raiz)<>44
 THEN RAISE EXCEPTION 'AD217: consumo RPT denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'AD217: material RPT inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
 OR c->>'operacion' IS DISTINCT FROM 'personal.rpt_publica.consultar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.rpt_publica.consultar.v2'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'personal'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'rpt_publica_publicacion'
 OR d->>'finalidad' IS DISTINCT FROM 'consultar_organizacion_publicada'
 OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' !~ '^rpt-publicada:[a-z0-9][a-z0-9:-]{2,127}$'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS NULL
 OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM campos
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 THEN RAISE EXCEPTION 'AD217: concesión RPT divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'consulta_rpt_publica_v2_personal',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
 OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR x.consumo_huella_sha256 IS NULL OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
 OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
 OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en)
 THEN RAISE EXCEPTION 'AD217: consumo RPT divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
 x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;

DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR NOT EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
   AND a.grantee='vec_personal_propietario'::regrole AND a.grantor=p.proowner
   AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR has_function_privilege('vec_personal_rpt_v2_lector',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD217: ACL o ABI de fachada RPT incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
