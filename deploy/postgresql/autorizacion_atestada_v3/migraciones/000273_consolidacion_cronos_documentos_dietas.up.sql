\set ON_ERROR_STOP on
-- AD273: consolidación nominal desde el núcleo HZ37. Conserva autoridad,
-- historia y cuerpo común; añade sólo contratos explícitos de Cronos,
-- Personal, Documentos y Dietas en una transacción.
-- AD263 precede a AD273. AUT95 depende del rol documental que crea AD273;
-- antes de AUT95 no se consume ningún acto de subida local. D20 y sus actos
-- documentales se activan después de AUT95.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000273',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regprocedure('vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(jsonb)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_bandeja_correcciones_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_anexo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_revision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion.cotejar_lectura_justificacion_cronos_actual_v1(jsonb,text,text,text,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.login_lectura_justificacion_cronos_valido_v1(text)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.acreditar_sesion_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD273: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DO $roles$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR
  EXISTS(SELECT 1 FROM pg_roles WHERE rolname IN ('vec_documentos_subida_datos','vec_documentos_frontera_externa'))
 THEN RAISE EXCEPTION 'AD270: roles preexistentes' USING ERRCODE='55000'; END IF;
END $roles$;
CREATE ROLE vec_documentos_subida_datos NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_documentos_frontera_externa NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $db$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_documentos_subida_datos,vec_documentos_frontera_externa',current_database());
END $db$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.login_liquidacion_dietas_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
  SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
  JOIN pg_roles g ON g.oid=m.roleid
  WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL AND g.rolname='vec_dietas_ejecutor'
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
           OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
   AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)));
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_liquidacion_dietas_valido_v1() FROM PUBLIC;
DO $nucleo_pre$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; p record; original text; deps jsonb;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 original:=pg_get_functiondef(f);
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'3e3f87faab57f26dce85d85fb71d2db45692f8bcdde8d4f273f1aa6ee1118986'
    OR p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD273: nucleo preimagen incompatible' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec_ad273.nucleo_pre',jsonb_build_object('meta',to_jsonb(p)-'prosrc','deps',deps,
   'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
    FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))::text,true);
END $nucleo_pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea) RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET lock_timeout TO '2s'
    AS $_$
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
        IF p_perfil_mutacion NOT IN ('consulta_participaciones_propias_bolsa','portal_candidato_bolsa','presentacion_inscripcion','presentacion_alegacion')
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
                AND p_perfil_mutacion IS DISTINCT FROM 'presentacion_inscripcion'
                AND p_perfil_mutacion IS DISTINCT FROM 'presentacion_alegacion'
                OR EXISTS (WITH RECURSIVE roles(rol_id) AS (
                     SELECT m.roleid FROM pg_catalog.pg_auth_members m
                      WHERE m.member=session_user::pg_catalog.regrole
                     UNION
                     SELECT m.roleid FROM pg_catalog.pg_auth_members m
                      JOIN roles r ON r.rol_id=m.member)
                    SELECT 1 FROM roles
                     WHERE rol_id<>'vec_bolsa_llamamientos_portal_externo'::pg_catalog.regrole)))
       OR (
           (p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_plan_nominal_firma_ct'
            AND vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'version_rol_bolsa'
            AND vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'version_rol_bolsa_externa'
            AND vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'
            AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
             JOIN pg_roles g ON g.oid=m.roleid
             WHERE l.rolname=session_user AND l.rolname='vec_externo_meritos_cartera'
              AND l.rolcanlogin AND l.rolinherit
              AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
              AND l.rolconfig IS NULL AND g.rolname='vec_meritos_cartera_externo'
              AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
               OR g.rolreplication OR g.rolbypassrls)
              AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
            AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1
            AND NOT EXISTS(SELECT 1 FROM pg_auth_members
             WHERE member='vec_meritos_cartera_externo'::regrole))
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_inscripcion'
            AND session_user='vec_bolsa_llamamientos_desarrollo'
            AND current_setting('role')='none'
            AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
              WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
              AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
              AND l.rolconfig IS NULL AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
              AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
              AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
              AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member='vec_bolsa_llamamientos_ejecutor'::regrole)
              AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,m.roleid))))
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'revision_inscripcion'
            AND session_user='vec_bolsa_llamamientos_desarrollo'
            AND current_setting('role')='none'
            AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
              WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
              AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
              AND l.rolconfig IS NULL AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
              AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
              AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
              AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member='vec_bolsa_llamamientos_ejecutor'::regrole)
              AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,m.roleid))))
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_rol_nuevo'
            AND vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1() IS TRUE)
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'actos_admin_perfiles'
            AND vec_autorizacion_atestada_v3.login_actos_admin_perfiles_valido_v1() IS TRUE)
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
               p_perfil_mutacion IS NOT DISTINCT FROM 'carga_convoca_bolsa'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
                 AND r.rolname='vec_bolsa_llamamientos_desarrollo' AND r.rolcanlogin AND r.rolinherit
                 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
                 AND r.rolconfig IS NULL)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
                 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_bolsa_llamamientos_ejecutor'::regrole)
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
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_bolsas_interna'
            AND session_user='vec_interno_bolsa_preferencias_desarrollo'
            AND current_setting('role')='none'
            AND EXISTS (SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
              JOIN pg_roles g ON g.oid=m.roleid
              WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
                AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
                AND l.rolconfig IS NULL AND g.rolname='vec_bolsa_llamamientos_preferencias_interna'
                AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
                AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
                AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
                AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
                AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid))))
           OR (p_perfil_mutacion IS NOT DISTINCT FROM 'registro_economico_dietas'
            AND vec_autorizacion_atestada_v3.login_liquidacion_dietas_valido_v1() IS TRUE)
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
               p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_rechazar')
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                 AND NOT (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_interno'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member IN
                 ('vec_meritos_ejecutor'::regrole,'vec_meritos_interno'::regrole,'vec_meritos_externo'::regrole))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_subida_local_externa'
               AND session_user='vec_externo_documentos_datos'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
                 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_documentos_subida_datos'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_documentos_subida_datos'::regrole)
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_hecho_propio_externo'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                 AND NOT (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member IN
                 ('vec_meritos_ejecutor'::regrole,'vec_meritos_interno'::regrole,'vec_meritos_externo'::regrole))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_consulta_propia_interna'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                 AND NOT (r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls))
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_meritos_consulta_propia_interno'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_meritos_consulta_propia_interno'::regrole)
           )
           OR (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_interno' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_externo' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_rechazar' AND p_perfil_mutacion IS DISTINCT FROM 'meritos_consulta_propia_interna' AND p_perfil_mutacion IS DISTINCT FROM 'consulta_rrhh_bolsa' AND p_perfil_mutacion IS DISTINCT FROM 'carga_convoca_bolsa' AND p_perfil_mutacion IS DISTINCT FROM 'historia_servicios_propia' AND p_perfil_mutacion IS DISTINCT FROM 'exportacion_servicios_propios' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_cargo_competencial' AND p_perfil_mutacion IS DISTINCT FROM 'capacidades_admin' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_publicar' AND p_perfil_mutacion IS DISTINCT FROM 'persona_denominacion_leer' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_listar' AND p_perfil_mutacion IS DISTINCT FROM 'usuarios_admin_consultar' AND p_perfil_mutacion IS DISTINCT FROM 'servicios_certificados_propios' AND p_perfil_mutacion IS DISTINCT FROM 'lote_perfiles_admin' AND p_perfil_mutacion IS DISTINCT FROM 'actos_admin_perfiles' AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_plan_nominal_firma_ct' AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_rol_nuevo' AND p_perfil_mutacion IS DISTINCT FROM 'revision_inscripcion' AND p_perfil_mutacion IS DISTINCT FROM 'incorporacion_inscripcion' AND p_perfil_mutacion IS DISTINCT FROM 'version_rol_bolsa' AND p_perfil_mutacion IS DISTINCT FROM 'version_rol_bolsa_externa'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_cartera_retirada'
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
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_permisos_bandeja'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_permiso_resolver'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_avisos_propio'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_aviso_archivar'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correccion_recibo'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correcciones_bandeja'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correccion_responsable'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correccion_rrhh'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correccion_aplicar'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correccion_historia_propia'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_correccion_historia_equipo'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_notificacion_registrar'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_notificaciones_propio'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_notificaciones_bandeja'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_notificacion_atender'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_presencia_equipo_jefatura'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_presencia_equipo_rrhh'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_incidencias_equipo_jefatura'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_incidencias_equipo_rrhh'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_informe_propio_empleado'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_informe_equipo_jefatura'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_informe_equipo_rrhh'
               AND p_perfil_mutacion IS DISTINCT FROM 'vinculo_propio_historico_crn11'
               AND p_perfil_mutacion IS DISTINCT FROM 'unidad_trabajo_propia_cronos'
               AND p_perfil_mutacion IS DISTINCT FROM 'documentos_subida_local_externa'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_justificacion_anexar'
               AND p_perfil_mutacion IS DISTINCT FROM 'cronos_justificacion_revisar'
               AND p_perfil_mutacion IS DISTINCT FROM 'preferencias_bolsas_interna'
               AND p_perfil_mutacion IS DISTINCT FROM 'registro_economico_dietas'
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
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_propio_historico_crn11'
               AND pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_personal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user AND r.rolname<>'vec_personal_ejecutor' AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'unidad_trabajo_propia_cronos'
               AND current_setting('role')='none'
               AND EXISTS (SELECT 1 FROM pg_roles l WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
                 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                 AND m.roleid='vec_personal_ejecutor'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user
                 AND r.rolname<>'vec_personal_ejecutor' AND pg_has_role(session_user,r.oid,'MEMBER')))
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
               p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar','cronos_permisos_bandeja','cronos_permiso_resolver','cronos_avisos_propio','cronos_aviso_archivar','cronos_correccion_recibo','cronos_correcciones_bandeja','cronos_correccion_responsable','cronos_correccion_rrhh','cronos_correccion_aplicar','cronos_correccion_historia_propia','cronos_correccion_historia_equipo','cronos_notificacion_registrar','cronos_notificaciones_propio','cronos_notificaciones_bandeja','cronos_notificacion_atender','cronos_presencia_equipo_jefatura','cronos_presencia_equipo_rrhh','cronos_incidencias_equipo_jefatura','cronos_incidencias_equipo_rrhh','cronos_informe_propio_empleado','cronos_informe_equipo_jefatura','cronos_informe_equipo_rrhh','cronos_justificacion_anexar','cronos_justificacion_revisar')
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
       ) IS NOT TRUE THEN
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
       OR (
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
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_emision_bolsa_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.vinculo_emision_bolsa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.bolsa.vincular'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_bolsa_expediente'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
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
               p_perfil_mutacion IS NOT DISTINCT FROM 'reanudacion_solicitud_llamamiento'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_solicitud'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.reanudar_solicitud'
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
 p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_propio_historico_crn11'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.vinculo_propio.crn11.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_historico_propio_crn11'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'acreditar_vinculo_historico_propio_crn11'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^emp_[A-Za-z0-9_-]{22,128}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["empleado_ref","fuente_ref","persona_ref","version","vinculo_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'unidad_trabajo_propia_cronos'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.unidad_trabajo.propia.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.unidad_trabajo.propia.cronos.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'unidad_trabajo_propia_cronos'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_unidad_trabajo_propia_cronos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^emp_[A-Za-z0-9_-]{22,128}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["desde","empleado_ref","fuente_ref","fuente_version_ref","hasta_exclusivo","organismo_ref","regimen_ref","relacion_ref","unidad_clave","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IN ('cronos_marcaje_propio','cronos_marcaje_remoto_disponibilidad','cronos_marcaje_remoto_recibo','cronos_saldo_propio','cronos_movimientos_propio','cronos_correccion_solicitar','cronos_permisos_propio','cronos_permiso_solicitar','cronos_permisos_bandeja','cronos_permiso_resolver','cronos_avisos_propio','cronos_aviso_archivar','cronos_correccion_recibo','cronos_correcciones_bandeja','cronos_correccion_responsable','cronos_correccion_rrhh','cronos_correccion_aplicar','cronos_correccion_historia_propia','cronos_correccion_historia_equipo','cronos_notificacion_registrar','cronos_notificaciones_propio','cronos_notificaciones_bandeja','cronos_notificacion_atender','cronos_presencia_equipo_jefatura','cronos_presencia_equipo_rrhh','cronos_incidencias_equipo_jefatura','cronos_incidencias_equipo_rrhh','cronos_informe_propio_empleado','cronos_informe_equipo_jefatura','cronos_informe_equipo_rrhh','cronos_justificacion_anexar','cronos_justificacion_revisar')
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
   OR (p_perfil_mutacion='cronos_permisos_bandeja'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permisos_bandeja.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permisos.bandeja.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bandeja_permisos'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_permisos'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["pendientes"]'::jsonb)
   OR (p_perfil_mutacion='cronos_permiso_resolver'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.permiso.resolver.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.permiso.resolver'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'resolucion_permiso'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'resolver_permiso'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_avisos_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.avisos_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.avisos.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'avisos_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_avisos_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["avisos"]'::jsonb)
   OR (p_perfil_mutacion='cronos_aviso_archivar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.aviso_propio.archivar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.aviso.propio.archivar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'archivo_aviso'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'archivar_aviso_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)   OR (p_perfil_mutacion='cronos_correccion_recibo'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion_propia.recibo.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.recibo.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'recibo_correccion_propia'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_recibo_correccion_propia'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)   OR (p_perfil_mutacion='cronos_correcciones_bandeja'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion.bandeja.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.bandeja.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bandeja_correcciones'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_correcciones'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["pendientes"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_responsable'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion.responsable.decidir.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.responsable.decidir'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'actuacion_correccion'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'decidir_correccion_responsable'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_rrhh'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion.rrhh.resolver.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.rrhh.resolver'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'actuacion_correccion'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'resolver_correccion_rrhh'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_aplicar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion.aplicar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.aplicar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'actuacion_correccion'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'aplicar_correccion'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_historia_propia'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion.historia.propia.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.historia.propia.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'historia_correccion_propia'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_historia_correccion_propia'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["historia"]'::jsonb)
   OR (p_perfil_mutacion='cronos_correccion_historia_equipo'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.correccion.historia.equipo.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.correccion.historia.equipo.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'historia_correccion_equipo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_historia_correccion_equipo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["historia"]'::jsonb)   OR (p_perfil_mutacion='cronos_notificacion_registrar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.notificacion_propia.registrar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.notificacion.propia.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'notificacion_propia'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'comunicar_incidencia_rrhh'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)
   OR (p_perfil_mutacion='cronos_notificaciones_propio'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.notificaciones_propio.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.notificaciones.propio.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'notificaciones_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_notificaciones_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["notificaciones"]'::jsonb)
   OR (p_perfil_mutacion='cronos_notificaciones_bandeja'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.notificaciones_bandeja.consultar.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.notificaciones.bandeja.consultar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bandeja_notificaciones'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_notificaciones'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["notificaciones"]'::jsonb)
   OR (p_perfil_mutacion='cronos_notificacion_atender'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.notificacion.atender.v1'
    AND d->>'accion' IS NOT DISTINCT FROM 'cronos.notificacion.atender'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'atencion_notificacion'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'atender_notificacion'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb)   OR (p_perfil_mutacion='cronos_presencia_equipo_jefatura'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.presencia_equipo.consultar.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.presencia.equipo.consultar'
    AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
    AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'presencia_equipo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_presencia_equipo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["agregado","causa","cobertura","empleado_etiqueta","estado","fecha","persona_ref"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (p_perfil_mutacion='cronos_presencia_equipo_rrhh'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.presencia_equipo.consultar.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.presencia.equipo.consultar'
    AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
    AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'presencia_equipo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_presencia_equipo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["agregado","causa","cobertura","empleado_etiqueta","estado","fecha","persona_ref"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (p_perfil_mutacion='cronos_incidencias_equipo_jefatura'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.incidencias_equipo.consultar.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.incidencias.equipo.consultar'
    AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
    AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incidencias_equipo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_incidencias_equipo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["agregado","cobertura","dias","empleado_etiqueta","periodo","persona_ref"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (p_perfil_mutacion='cronos_incidencias_equipo_rrhh'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.incidencias_equipo.consultar.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.incidencias.equipo.consultar'
    AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
    AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incidencias_equipo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_incidencias_equipo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["agregado","cobertura","dias","empleado_etiqueta","periodo","persona_ref"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (p_perfil_mutacion='cronos_informe_propio_empleado'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.informe_propio.exportar.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.informe.propio.exportar'
    AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
    AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'informe_propio'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'exportar_informe_propio'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["archivo","recibo"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (p_perfil_mutacion='cronos_informe_equipo_jefatura'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.informe_equipo.exportar.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.informe.equipo.exportar'
    AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
    AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'informe_equipo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'exportar_informe_equipo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["archivo","recibo"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
   OR (p_perfil_mutacion='cronos_informe_equipo_rrhh'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.informe_equipo.exportar.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.informe.equipo.exportar'
    AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
    AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'informe_equipo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'exportar_informe_equipo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["archivo","recibo"]'::jsonb
    AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)

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
 p_perfil_mutacion IS NOT DISTINCT FROM 'cronos_justificacion_anexar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.justificacion.anexar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.justificacion.v1'
 AND c->>'efecto_ref' ~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'justificacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'anexar_justificacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'cronos_justificacion_revisar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'cronos.justificacion.revisar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_cronos_v1.justificacion.v1'
 AND c->>'efecto_ref' ~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'cronos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'justificacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_justificacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["recibo"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_bolsas_interna'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.interna.v1'
 AND c->>'efecto_ref' ~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_preferencias_bolsas_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_economico_dietas'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.liquidacion_economica.registrar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.liquidacion_economica.registrar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'liquidacion_economica'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_liquidacion_economica_dietas'
 AND (d->>'recurso_ref' ~ '^dco_[A-Za-z0-9_-]{22,128}$') IS TRUE
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["constancia","economia","esquema","snapshot_sha256"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
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
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["intento","reserva"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.original_firmable.confirmar'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_documentos.operacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_evidencia_meritos'
 AND nullif(d->>'version_rol_ref','') IS NOT NULL
 AND d->>'recurso_ref' ~ '^(ref:[0-9a-f]{64}|[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$'
 AND d->>'recurso_ref' IS DISTINCT FROM ('ref:'||repeat('0',64))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion' IN ('documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva','documentos.ingesta_mayan.abandonar_recurso')
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'reserva_recurso_documental'
       AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["reserva","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.preparar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["plan","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.despachar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["intento","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.resolver'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["evidencia_custodia","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.conciliar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["evidencia_custodia","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.consultar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["plan","estado"]'::jsonb)))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM CASE d->>'accion'
  WHEN 'documentos.carga.preparar_reserva' THEN 'vec_documentos.carga.preparar_reserva.v1'
  WHEN 'documentos.carga.preparar_confirmacion' THEN 'vec_documentos.carga.preparar_confirmacion.v1'
  WHEN 'documentos.carga.preparar_respuesta' THEN 'vec_documentos.carga.preparar_respuesta.v1'
  WHEN 'documentos.carga.consultar_propia' THEN 'vec_documentos.carga.consultar_propia.v1'
  WHEN 'documentos.carga_admitida.consultar_para_custodia' THEN 'vec_documentos.carga_admitida.consultar_para_custodia.v1'
  WHEN 'documentos.carga.conciliar' THEN 'vec_documentos.carga.conciliar.v1'
  WHEN 'documentos.carga.confirmar_intencion' THEN 'vec_documentos.carga.confirmar_intencion.v1'
  WHEN 'documentos.carga.confirmar_resultado' THEN 'vec_documentos.carga.confirmar_resultado.v1'
  WHEN 'vec.documentos.carga.analizar' THEN 'vec_documentos.carga.analizar.v1'
  WHEN 'documentos.carga.promover_intencion' THEN 'vec_documentos.carga.promover_intencion.v1'
  WHEN 'documentos.carga.promover_resultado' THEN 'vec_documentos.carga.promover_resultado.v1' END
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_evidencia_meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'carga_documental'
 AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
 AND d->>'recurso_ref' IS DISTINCT FROM ('ref:'||repeat('0',64))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion'='documentos.carga.preparar_reserva'
        AND d->'campos_permitidos'='["reserva","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.preparar_confirmacion'
        AND d->'campos_permitidos'='["manifiesto","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.preparar_respuesta'
        AND d->'campos_permitidos'='["recibo","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.consultar_propia'
        AND d->'campos_permitidos'='["carga","manifiesto"]'::jsonb)
   OR (d->>'accion'='documentos.carga_admitida.consultar_para_custodia'
        AND d->'campos_permitidos'='["carga_admitida","analisis_seguridad","contenido_admitido"]'::jsonb)
   OR (d->>'accion'='documentos.carga.conciliar'
        AND d->'campos_permitidos'='["evidencia","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.confirmar_intencion'
        AND d->'campos_permitidos'='["intencion","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.confirmar_resultado'
        AND d->'campos_permitidos'='["contenido_cuarentena","estado"]'::jsonb)
   OR (d->>'accion'='vec.documentos.carga.analizar'
        AND d->'campos_permitidos'='["analisis_seguridad","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.promover_intencion'
        AND d->'campos_permitidos'='["intencion","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.promover_resultado'
        AND d->'campos_permitidos'='["contenido_admitido","estado"]'::jsonb)))

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
 p_perfil_mutacion IS NOT DISTINCT FROM 'actos_admin_perfiles'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND (d->'obligaciones' IN ('[]'::jsonb,'["auditar"]'::jsonb)) IS TRUE
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND (
   (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.ordinario.v1'
    AND (c->>'operacion' IN ('administracion.perfiles.otorgar','administracion.perfiles.revocar')) IS TRUE
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'perfil'
    AND (d->>'recurso_ref' ~ '^acto_admin:[0-9a-f]{32}$') IS TRUE)
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.propuesta.v1'
    AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.perfiles.proponer'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'perfil'
    AND (d->>'recurso_ref' ~ '^propuesta_admin:[0-9a-f]{32}$') IS TRUE)
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_autorizacion.administracion_perfiles.cierre.v1'
    AND (c->>'operacion' IN ('administracion.perfiles.aprobar','administracion.perfiles.rechazar')) IS TRUE
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'propuesta_perfil'
    AND (d->>'recurso_ref' ~ '^cierre_admin:[0-9a-f]{32}$') IS TRUE))
 AND vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   c->>'operacion','administracion',d->>'tipo_recurso','gestion_perfiles',
   '[]'::jsonb,d->'vinculo_autenticacion_actor') IS TRUE
 AND vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
    p_decision_canonica,p_motivo_canonico,p_persona_version,p_perfil_version) IS NOT NULL)
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
 p_perfil_mutacion IS NOT DISTINCT FROM 'carga_convoca_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.carga_convoca.confirmar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.carga_convoca.confirmar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'carga_convoca'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'carga_bolsa_convoca'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^acta:importacion-convoca:[0-9a-f]{64}$'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'version_rol_bolsa'
 AND (c->>'operacion' IN ('administracion.perfiles.version_bolsa.proponer','administracion.perfiles.version_bolsa.aprobar')) IS TRUE
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM (
   CASE c->>'operacion'
    WHEN 'administracion.perfiles.version_bolsa.proponer' THEN 'vec_autorizacion.versionar_rol_bolsa.propuesta.v1'
    WHEN 'administracion.perfiles.version_bolsa.aprobar' THEN 'vec_autorizacion.versionar_rol_bolsa.cierre.v1' END)
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM (
   CASE c->>'operacion' WHEN 'administracion.perfiles.version_bolsa.proponer' THEN 'definicion_rol'
    ELSE 'propuesta_definicion_rol' END)
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_definiciones_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL
 AND ((c->>'operacion'='administracion.perfiles.version_bolsa.proponer'
      AND d->>'recurso_ref' ~ '^rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v[1-9][0-9]{0,8}$')
   OR (c->>'operacion'='administracion.perfiles.version_bolsa.aprobar'
      AND (d->>'recurso_ref' ~ '^propuesta_admin:[0-9a-f]{32}$') IS TRUE))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' ~ '^[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_version_rol_bolsa_v1(d) IS TRUE)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'version_rol_bolsa_externa'
 AND (c->>'operacion' IN ('administracion.perfiles.version_bolsa.proponer','administracion.perfiles.version_bolsa.aprobar')) IS TRUE
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM (
   CASE c->>'operacion'
    WHEN 'administracion.perfiles.version_bolsa.proponer' THEN 'vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1'
    WHEN 'administracion.perfiles.version_bolsa.aprobar' THEN 'vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1' END)
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM (
   CASE c->>'operacion' WHEN 'administracion.perfiles.version_bolsa.proponer' THEN 'definicion_rol'
    ELSE 'propuesta_definicion_rol' END)
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_definiciones_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL
 AND ((c->>'operacion'='administracion.perfiles.version_bolsa.proponer'
      AND d->>'recurso_ref' IN ('rol:candidato_bolsa_portal_historial_propio_desarrollo:v2','rol:candidato_bolsa_portal_historial_propio_desarrollo:v3'))
   OR (c->>'operacion'='administracion.perfiles.version_bolsa.aprobar'
      AND (d->>'recurso_ref' ~ '^propuesta_admin:[0-9a-f]{32}$') IS TRUE))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' ~ '^[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_version_rol_bolsa_v1(d) IS TRUE)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_inscripcion'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.incorporar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.inscripcion.rrhh.incorporar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_inscripcion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'incorporar_inscripcion'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'revision_inscripcion'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.inscripcion.revisar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.inscripcion.rrhh.decidir'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_inscripcion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_inscripcion'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_rol_nuevo'
 AND (c->>'operacion' IN ('administracion.perfiles.definicion.proponer','administracion.perfiles.definicion.aprobar')) IS TRUE
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM (
   CASE c->>'operacion'
    WHEN 'administracion.perfiles.definicion.proponer' THEN 'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1'
    WHEN 'administracion.perfiles.definicion.aprobar' THEN 'vec_autorizacion.gobierno_rol_nuevo.cierre.v1' END)
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM (
   CASE c->>'operacion' WHEN 'administracion.perfiles.definicion.proponer' THEN 'definicion_rol'
    ELSE 'propuesta_definicion_rol' END)
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_definiciones_perfiles'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL
 AND ((c->>'operacion'='administracion.perfiles.definicion.proponer'
      AND d->>'recurso_ref' ~ '^rol:[a-z][a-z0-9_]{2,63}:v1$')
   OR (c->>'operacion'='administracion.perfiles.definicion.aprobar'
      AND (d->>'recurso_ref' ~ '^propuesta_admin:[0-9a-f]{32}$') IS TRUE))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT NULL
 AND d->>'contexto_recurso_huella_sha256' ~ '^[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true'
 AND vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS TRUE)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documentos_subida_local_externa'
 AND c->>'operacion' IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio')
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM (CASE c->>'operacion'
  WHEN 'meritos.cartera.comprobante.subir_propio' THEN 'vec_documentos.subida_local.acto_modulo.v1'
  ELSE 'vec_documentos.subida_local.descarga_propia.v1' END)
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_evidencia_meritos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM (CASE c->>'operacion'
  WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
  ELSE '["contenido"]'::jsonb END)
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_propio_externo','meritos_hecho_rechazar')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND (
   (p_perfil_mutacion='meritos_hecho_propio_interno'
    AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
    AND ((c->>'operacion'='meritos.hecho.declarar' AND c->>'audiencia_consumo'='vec_meritos.hecho.declarar.v1' AND d->>'finalidad'='declaracion_hecho_propio')
      OR (c->>'operacion'='meritos.hecho.rectificar' AND c->>'audiencia_consumo'='vec_meritos.hecho.rectificar.v1' AND d->>'finalidad'='rectificacion_hecho_propio')))
   OR (p_perfil_mutacion='meritos_hecho_propio_externo'
    AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
    AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.declarar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.declarar.v1'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'declaracion_hecho_propio')
   OR (p_perfil_mutacion='meritos_hecho_rechazar'
    AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
    AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.rechazar'
    AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.rechazar.v1'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'revision_hecho_merito')
 ))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_consulta_propia_interna'
 AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.consultar_propio'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.consultar_propio.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_hecho_propio'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["hecho_actual","recibo_consulta"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_cartera_retirada'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.cartera.retirar_propio.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.cartera.retirar_propio'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_cartera_propia'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'false'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["disponibilidad","recibo"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
       ) IS NOT TRUE
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
    IF p_perfil_mutacion IN ('meritos_cartera_retirada','documentos_subida_local_externa')
       AND vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
        d->'vinculo_autenticacion_actor',
        (c->>'emitida_en')::timestamptz,(c->>'expira_en')::timestamptz,
        clock_timestamp()) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo cartera Méritos denegado';
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
    IF p_perfil_mutacion IN ('meritos_cartera_retirada','documentos_subida_local_externa') THEN
        SELECT * INTO v_registro
          FROM vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(
            p_decision_canonica,p_motivo_canonico,p_persona_version,p_perfil_version);
    ELSE
        SELECT * INTO v_registro
          FROM vec_autorizacion.
               registrar_y_revalidar_decision_contexto_actor_v3(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;
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
    IF p_perfil_mutacion IN ('meritos_cartera_retirada','documentos_subida_local_externa') THEN
        v_revalidada_en:=vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(
          p_decision_canonica,p_motivo_canonico,p_persona_version,p_perfil_version);
    ELSE
        v_revalidada_en :=
          vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
              p_decision_canonica, p_motivo_canonico,
              p_persona_version, p_perfil_version);
    END IF;
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
$_$;

DO $nucleo_post$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; snapshot jsonb;
BEGIN
 snapshot:=current_setting('vec_ad273.nucleo_pre',true)::jsonb;
 IF snapshot IS NULL
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f)<>'55a75b6e78f63a34f0a633c96eb31b36b5ff3c2fee07c60465a3fe7a1ffc6bf7'
    OR (SELECT jsonb_build_object('meta',to_jsonb(p)-'prosrc',
      'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f),
      'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
       FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM snapshot
 THEN RAISE EXCEPTION 'AD273: nucleo postimagen incompatible' USING ERRCODE='55000'; END IF;
END $nucleo_post$;


LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; post text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex')<>'6d42d1f8ce4be9164e8fd6c1bb76c508be600284bb73398c1f77e132d1d687e5'
   OR left(d,7)<>'CHECK (' OR right(d,1)<>')'
 THEN RAISE EXCEPTION 'AD273: audiencias HZ37 incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-1)||' OR audiencia_consumo = ANY (ARRAY['||
  quote_literal('vec_cronos_v1.permisos_bandeja.consultar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.permiso.resolver.v1')||'::text, '||
  quote_literal('vec_cronos_v1.avisos_propio.consultar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.aviso_propio.archivar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.correccion_propia.recibo.consultar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.correccion.bandeja.v1')||'::text, '||
  quote_literal('vec_cronos_v1.correccion.responsable.decidir.v1')||'::text, '||
  quote_literal('vec_cronos_v1.correccion.rrhh.resolver.v1')||'::text, '||
  quote_literal('vec_cronos_v1.correccion.aplicar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.correccion.historia.propia.v1')||'::text, '||
  quote_literal('vec_cronos_v1.correccion.historia.equipo.v1')||'::text, '||
  quote_literal('vec_cronos_v1.notificacion_propia.registrar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.notificaciones_propio.consultar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.notificaciones_bandeja.consultar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.notificacion.atender.v1')||'::text, '||
  quote_literal('vec_personal.vinculo_propio.crn11.v1')||'::text, '||
  quote_literal('vec_personal.unidad_trabajo.propia.cronos.v1')||'::text, '||
  quote_literal('vec_documentos.subida_local.acto_modulo.v1')||'::text, '||
  quote_literal('vec_documentos.subida_local.descarga_propia.v1')||'::text, '||
  quote_literal('vec_dietas.liquidacion_economica.registrar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.presencia_equipo.consultar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.incidencias_equipo.consultar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.informe_propio.exportar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.informe_equipo.exportar.v1')||'::text, '||
  quote_literal('vec_cronos_v1.justificacion.v1')||'::text, '||
  quote_literal('vec.bolsa.preferencias-bolsas.ordenar.interna.v1')||'::text'||']))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT post FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF post IS DISTINCT FROM nueva THEN RAISE EXCEPTION 'AD273: postimagen de audiencias incompatible' USING ERRCODE='55000'; END IF;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_permisos_bandeja','vec_cronos_v1.permisos_bandeja.consultar.v1',
   'cronos.permisos.bandeja.consultar','bandeja_permisos','consultar_bandeja_permisos','["pendientes"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_permiso_resolver','vec_cronos_v1.permiso.resolver.v1',
   'cronos.permiso.resolver','resolucion_permiso','resolver_permiso','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_avisos_propio','vec_cronos_v1.avisos_propio.consultar.v1',
   'cronos.avisos.propio.consultar','avisos_propio','consultar_avisos_propio','["avisos"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_aviso_archivar','vec_cronos_v1.aviso_propio.archivar.v1',
   'cronos.aviso.propio.archivar','archivo_aviso','archivar_aviso_propio','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_correccion_recibo_propio_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_correccion_recibo','vec_cronos_v1.correccion_propia.recibo.consultar.v1',
  'cronos.correccion.recibo.consultar','recibo_correccion_propia','consultar_recibo_correccion_propia','["recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_correccion_recibo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_correccion_recibo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;

-- La segunda fachada conserva el contrato exacto de Personal26. Su
-- ejecución solo pertenece al propietario Personal; usa otra decisión V3.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
   RAISE EXCEPTION 'consumo CRN11 denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb;
   d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'material CRN11 inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.vinculo_propio.crn11.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_historico_propio_crn11'
    OR d->>'finalidad' IS DISTINCT FROM 'acreditar_vinculo_historico_propio_crn11'
    OR d->'campos_permitidos' IS DISTINCT FROM '["empleado_ref","fuente_ref","persona_ref","version","vinculo_ref"]'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS NULL
    OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
   RAISE EXCEPTION 'consumo CRN11 denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'vinculo_propio_historico_crn11',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
   RAISE EXCEPTION 'consumo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
DO $acl$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_cronos_correccion_recibo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp'])
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_cronos_v1_propietario'::regrole)
                  OR a.privilege_type<>'EXECUTE' OR (a.grantee='vec_cronos_v1_propietario'::regrole AND a.is_grantable)))
 THEN RAISE EXCEPTION 'AD269: ACL o función incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
DO $acl_personal$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
                 AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR NOT has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','USAGE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_personal_propietario'::regrole)
                  OR a.privilege_type<>'EXECUTE' OR (a.grantee='vec_personal_propietario'::regrole AND a.is_grantable)))
 THEN RAISE EXCEPTION 'AD269: ACL Personal incompatible' USING ERRCODE='55000'; END IF;
END $acl_personal$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_unidad_trabajo_propia_cronos_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
   RAISE EXCEPTION 'consumo P52 denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb;
   d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'material P52 inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM 'personal.unidad_trabajo.propia.consultar'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.unidad_trabajo.propia.cronos.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'unidad_trabajo_propia_cronos'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_unidad_trabajo_propia_cronos'
    OR d->'campos_permitidos' IS DISTINCT FROM '["desde","empleado_ref","fuente_ref","fuente_version_ref","hasta_exclusivo","organismo_ref","regimen_ref","relacion_ref","unidad_clave","version"]'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS NULL
    OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
   RAISE EXCEPTION 'consumo P52 denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'unidad_trabajo_propia_cronos',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
   RAISE EXCEPTION 'consumo P52 divergente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_unidad_trabajo_propia_cronos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_unidad_trabajo_propia_cronos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_bandeja_correcciones_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_correcciones_bandeja','vec_cronos_v1.correccion.bandeja.v1',
  'cronos.correccion.bandeja.consultar','bandeja_correcciones','consultar_bandeja_correcciones','["pendientes"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_decision_responsable_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_correccion_responsable','vec_cronos_v1.correccion.responsable.decidir.v1',
  'cronos.correccion.responsable.decidir','actuacion_correccion','decidir_correccion_responsable','["recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_correccion_rrhh','vec_cronos_v1.correccion.rrhh.resolver.v1',
  'cronos.correccion.rrhh.resolver','actuacion_correccion','resolver_correccion_rrhh','["recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_correccion_aplicar','vec_cronos_v1.correccion.aplicar.v1',
  'cronos.correccion.aplicar','actuacion_correccion','aplicar_correccion','["recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_propia_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_correccion_historia_propia','vec_cronos_v1.correccion.historia.propia.v1',
  'cronos.correccion.historia.propia.consultar','historia_correccion_propia','consultar_historia_correccion_propia','["historia"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_equipo_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_correccion_historia_equipo','vec_cronos_v1.correccion.historia.equipo.v1',
  'cronos.correccion.historia.equipo.consultar','historia_correccion_equipo','consultar_historia_correccion_equipo','["historia"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_bandeja_correcciones_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_bandeja_correcciones_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_decision_responsable_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_decision_responsable_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_propia_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_propia_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_equipo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_equipo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_notificacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_notificacion_registrar','vec_cronos_v1.notificacion_propia.registrar.v1',
   'cronos.notificacion.propia.registrar','notificacion_propia','comunicar_incidencia_rrhh','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_propio_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_notificaciones_propio','vec_cronos_v1.notificaciones_propio.consultar.v1',
   'cronos.notificaciones.propio.consultar','notificaciones_propio','consultar_notificaciones_propio','["notificaciones"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_bandeja_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_notificaciones_bandeja','vec_cronos_v1.notificaciones_bandeja.consultar.v1',
   'cronos.notificaciones.bandeja.consultar','bandeja_notificaciones','consultar_bandeja_notificaciones','["notificaciones"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_atencion_notificacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna('cronos_notificacion_atender','vec_cronos_v1.notificacion.atender.v1',
   'cronos.notificacion.atender','atencion_notificacion','atender_notificacion','["recibo"]'::jsonb,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_notificacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_notificacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_bandeja_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_bandeja_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_atencion_notificacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_atencion_notificacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_jefatura_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_presencia_equipo_jefatura','vec_cronos_v1.presencia_equipo.consultar.v1','cronos.presencia.equipo.consultar','presencia_equipo','consultar_presencia_equipo','["agregado","causa","cobertura","empleado_etiqueta","estado","fecha","persona_ref"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_jefatura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_jefatura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_rrhh_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_presencia_equipo_rrhh','vec_cronos_v1.presencia_equipo.consultar.v1','cronos.presencia.equipo.consultar','presencia_equipo','consultar_presencia_equipo','["agregado","causa","cobertura","empleado_etiqueta","estado","fecha","persona_ref"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_jefatura_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_incidencias_equipo_jefatura','vec_cronos_v1.incidencias_equipo.consultar.v1','cronos.incidencias.equipo.consultar','incidencias_equipo','consultar_incidencias_equipo','["agregado","cobertura","dias","empleado_etiqueta","periodo","persona_ref"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_jefatura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_jefatura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_rrhh_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_incidencias_equipo_rrhh','vec_cronos_v1.incidencias_equipo.consultar.v1','cronos.incidencias.equipo.consultar','incidencias_equipo','consultar_incidencias_equipo','["agregado","cobertura","dias","empleado_etiqueta","periodo","persona_ref"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_propio_exportar_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_informe_propio_empleado','vec_cronos_v1.informe_propio.exportar.v1','cronos.informe.propio.exportar','informe_propio','exportar_informe_propio','["archivo","recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_propio_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_propio_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_jefatura_exportar_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_informe_equipo_jefatura','vec_cronos_v1.informe_equipo.exportar.v1','cronos.informe.equipo.exportar','informe_equipo','exportar_informe_equipo','["archivo","recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_jefatura_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_jefatura_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_rrhh_exportar_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_informe_equipo_rrhh','vec_cronos_v1.informe_equipo.exportar.v1','cronos.informe.equipo.exportar','informe_equipo','exportar_informe_equipo','["archivo","recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_rrhh_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_rrhh_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;

-- El publicador externo admite las dos audiencias nuevas junto a las 23
-- anteriores. No modifica claves ni confianza ya publicadas.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_anexo_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_justificacion_anexar','vec_cronos_v1.justificacion.v1','cronos.justificacion.anexar','justificacion','anexar_justificacion','["recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_anexo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_anexo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_revision_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_cronos_nominal_v3_interna(
  'cronos_justificacion_revisar','vec_cronos_v1.justificacion.v1','cronos.justificacion.revisar','justificacion','revisar_justificacion','["recibo"]'::jsonb,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_revision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_revision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR session_user<>'vec_interno_bolsa_preferencias_desarrollo'
    OR current_setting('role')<>'none'
 THEN RAISE EXCEPTION 'AD273: emisor de preferencias internas denegado' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD273: material de preferencias inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.interna.v1'
    OR c->>'efecto_ref' !~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD273: orden de preferencias internas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'preferencias_bolsas_interna',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'AD273: consumo de preferencias internas divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,
  v.auditoria_ref,v.consumida_en,v.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;

DO $publicacion_pre$
DECLARE f oid:='vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb)'::regprocedure; p record; original text; deps jsonb;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 original:=pg_get_functiondef(f);
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'8729e562d090fad7e171950d589cda4d718338fad8eb30b1650755a9780acc38'
    OR p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR p.prosecdef IS DISTINCT FROM false
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD273: publicacion preimagen incompatible' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec_ad273.publicacion_pre',jsonb_build_object('meta',to_jsonb(p)-'prosrc','deps',deps,
   'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
    FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))::text,true);
END $publicacion_pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(m jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE e pg_catalog.jsonb; c pg_catalog.jsonb; r pg_catalog.jsonb; k pg_catalog.jsonb; audiencias pg_catalog.text[]:=ARRAY[
 'vec_usuarios.preferencias.consultar.externa_personal.v1','vec_usuarios.preferencias.actualizar.externa_personal.v1',
 'vec_usuarios.correos.consultar.externa_personal.v1','vec_usuarios.correos.anadir.externa_personal.v1','vec_usuarios.correos.reenviar.externa_personal.v1','vec_usuarios.correos.verificar.externa_personal.v1','vec_usuarios.correos.activar.externa_personal.v1','vec_usuarios.correos.retirar.externa_personal.v1',
 'vec_usuarios.imagen.consultar.externa_personal.v1','vec_usuarios.imagen.actualizar.externa_personal.v1',
 'vec.bolsa.mi-bolsa.v1','vec.bolsa.mi-bolsa.historial.v1',
 'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1','vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1','vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1','vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1','vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1','vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1','vec_bolsa_llamamientos.inscripcion.presentar.v1','vec_bolsa_llamamientos.alegacion.presentar.v1','vec.bolsa.preferencias-bolsas.ordenar.v1','vec_meritos.cartera.retirar_propio.v1','vec_meritos.hecho.declarar.v1','vec_documentos.subida_local.acto_modulo.v1','vec_documentos.subida_local.descarga_propia.v1'];
BEGIN
 e:=vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1();
 IF pg_catalog.octet_length(m::pg_catalog.text)>262144 OR pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(m))<>3
 OR NOT(m ?& ARRAY['configuracion','raiz','claves']) THEN RAISE EXCEPTION 'AD3-140: material rechazado' USING ERRCODE='42501'; END IF;
 c:=m->'configuracion';r:=m->'raiz';
 IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(c))<>5
 OR NOT(c ?& ARRAY['revision','secuencia','huella_configuracion_sha256','publicada_en','expira_en'])
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(c) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key='secuencia' THEN 'number' ELSE 'string' END)
 OR pg_catalog.jsonb_typeof(r) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(r))<>8
 OR NOT(r ?& ARRAY['clave_id','version','clave_publica_spki_hex','huella_spki_sha256','valida_desde','valida_hasta','suite','audiencia_despliegue'])
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(r) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key='version' THEN 'number' ELSE 'string' END)
 OR NOT pg_catalog.starts_with(c->>'revision','confianza:atestacion:externo:')
 OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(c->>'huella_configuracion_sha256')
 OR (c->>'secuencia')!~'^[1-9][0-9]{0,15}$'
 OR (r->>'version')!~'^[1-9][0-9]{0,15}$'
 OR r->>'clave_id' IS DISTINCT FROM 'clave:atestacion:externo:'||(r->>'huella_spki_sha256')
 OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(r->>'huella_spki_sha256')
 OR (r->>'clave_publica_spki_hex')!~'^302a300506032b6570032100[0-9a-f]{64}$'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(r->>'clave_publica_spki_hex','hex')),'hex') IS DISTINCT FROM r->>'huella_spki_sha256'
 OR r->>'suite' IS DISTINCT FROM 'VEC-AD-3-COSE-EDDSA-1'
 OR r->>'audiencia_despliegue' IS DISTINCT FROM 'vec:desarrollo:contratacion-temporal:atestacion:v3'
 OR NOT pg_catalog.isfinite((c->>'publicada_en')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((c->>'expira_en')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((r->>'valida_desde')::pg_catalog.timestamptz)
 OR NOT pg_catalog.isfinite((r->>'valida_hasta')::pg_catalog.timestamptz)
 OR (c->>'publicada_en')::pg_catalog.timestamptz>pg_catalog.clock_timestamp()
 OR (c->>'expira_en')::pg_catalog.timestamptz<=pg_catalog.clock_timestamp()
 OR (r->>'valida_desde')::pg_catalog.timestamptz>(c->>'publicada_en')::pg_catalog.timestamptz
 OR (r->>'valida_hasta')::pg_catalog.timestamptz<(c->>'expira_en')::pg_catalog.timestamptz
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.puntero_configuracion_actual p ON p.configuracion_revision=cr.configuracion_revision WHERE (cr.raiz_clave_id,cr.raiz_version)=(r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version x WHERE x.huella_spki_sha256=r->>'huella_spki_sha256' AND (x.clave_id,x.version) IS DISTINCT FROM (r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz x WHERE (x.raiz_clave_id,x.raiz_version)=(r->>'clave_id',(r->>'version')::pg_catalog.numeric))
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion x WHERE x.configuracion_revision=c->>'revision')
 OR pg_catalog.jsonb_typeof(m->'claves') IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(m->'claves')<>25
 THEN RAISE EXCEPTION 'AD3-140: material rechazado' USING ERRCODE='42501'; END IF;
 IF (SELECT pg_catalog.count(DISTINCT u.item->>'audiencia_consumo') FROM pg_catalog.jsonb_array_elements(m->'claves') AS u(item))<>25
 THEN RAISE EXCEPTION 'AD3-140: audiencias rechazadas' USING ERRCODE='42501'; END IF;
 FOR k IN SELECT x FROM pg_catalog.jsonb_array_elements(m->'claves') x LOOP
  IF pg_catalog.jsonb_typeof(k) IS DISTINCT FROM 'object' OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(k))<>11
  OR NOT(k ?& ARRAY['clave_id','version','revision_gobierno','huella_gobierno_sha256','secreto_hmac_hex','huella_secreto_sha256','emisor_id','audiencia_consumo','valida_desde','valida_hasta','orden'])
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(k) x WHERE pg_catalog.jsonb_typeof(x.value) IS DISTINCT FROM CASE WHEN x.key=ANY(ARRAY['version','revision_gobierno','orden']) THEN 'number' ELSE 'string' END)
  OR k->>'audiencia_consumo'<>ALL(audiencias)
  OR NOT vec_autorizacion_atestada_v3.texto_tecnico_valido(k->>'clave_id',512)
  OR NOT pg_catalog.starts_with(k->>'emisor_id','emisor:externo:')
  OR (k->>'version')!~'^[1-9][0-9]{0,15}$' OR (k->>'revision_gobierno')!~'^[1-9][0-9]{0,15}$' OR (k->>'orden')!~'^[1-9][0-9]{0,15}$'
  OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(k->>'huella_gobierno_sha256')
  OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(k->>'huella_secreto_sha256')
  OR (k->>'secreto_hmac_hex')!~'^[0-9a-f]+$' OR pg_catalog.length(k->>'secreto_hmac_hex') NOT BETWEEN 64 AND 8192 OR pg_catalog.length(k->>'secreto_hmac_hex')%2<>0
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(k->>'secreto_hmac_hex','hex')),'hex') IS DISTINCT FROM k->>'huella_secreto_sha256'
  OR (k->>'valida_desde')::pg_catalog.timestamptz>(c->>'publicada_en')::pg_catalog.timestamptz
  OR (k->>'valida_hasta')::pg_catalog.timestamptz<(c->>'expira_en')::pg_catalog.timestamptz
  OR NOT pg_catalog.isfinite((k->>'valida_desde')::pg_catalog.timestamptz) OR NOT pg_catalog.isfinite((k->>'valida_hasta')::pg_catalog.timestamptz)
  OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad x WHERE (x.clave_id,x.version)=(k->>'clave_id',(k->>'version')::pg_catalog.numeric))
  THEN RAISE EXCEPTION 'AD3-140: clave externa rechazada' USING ERRCODE='42501'; END IF;
 END LOOP;
 RETURN pg_catalog.jsonb_build_object('preimagen_sha256',e->>'preimagen_sha256',
 'huella_aprobacion_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion_atestada_v3.material_publico_externo_v1(m)::pg_catalog.text,'UTF8')),'hex'));
EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'AD3-140: propuesta externa rechazada' USING ERRCODE='42501';
END $function$;

DO $publicacion_post$
DECLARE f oid:='vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1(jsonb)'::regprocedure; snapshot jsonb;
BEGIN
 snapshot:=current_setting('vec_ad273.publicacion_pre',true)::jsonb;
 IF snapshot IS NULL
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f)<>'7f7520aa2169965a9edadb79a2e1b21518d422c0d162d785ec57c375e2047024'
    OR (SELECT jsonb_build_object('meta',to_jsonb(p)-'prosrc',
      'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f),
      'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
       FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM snapshot
 THEN RAISE EXCEPTION 'AD273: publicacion postimagen incompatible' USING ERRCODE='55000'; END IF;
END $publicacion_post$;


-- La lectura vieja de ocho campos y la nueva de nueve conservan el mismo
-- registrador y la misma consulta viva de rol/asignación.
DO $cartera_pre$
DECLARE f oid:='vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(bytea,bytea,jsonb)'::regprocedure; p record; original text; deps jsonb;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 original:=pg_get_functiondef(f);
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'1e85223f3ea32d3b8ad978848675ec8d5af632f9cfe776347f8dc5f26c45515c'
    OR p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=10s']
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_meritos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 THEN RAISE EXCEPTION 'AD273: cartera preimagen incompatible' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec_ad273.cartera_pre',jsonb_build_object('meta',to_jsonb(p)-'prosrc','deps',deps,
   'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
    FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))::text,true);
END $cartera_pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb) RETURNS TABLE(auditoria_ref text, secuencia numeric, huella_sha256 text, correlacion_ref text, registrada_en timestamp with time zone)
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '2s'
    SET statement_timeout TO '10s'
    AS $_$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_acreditada timestamptz;
 v_claves constant text[]:=ARRAY[
  'esquema','persona_ref','limite','filtro','cursor_hecho_ref','corte_en','idioma',
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'tipo_recurso','recurso_ref','finalidad_ref','campos_permitidos','resultado',
  'motivo_ref','proceso','canal','correlacion_ref','lectura_revision_permisos',
  'lectura_instantanea_sha256','perfil_activo_ref','asignacion_ref','asignacion_version',
  'asignacion_huella_sha256','version_rol_ref','version_rol_version',
  'version_rol_huella_sha256','control_version_rol_ref',
  'revision_control_version_rol','control_version_rol_huella_sha256',
  'revision_catalogo_politicas','catalogo_politicas_huella_sha256',
  'politicas_evaluadas','emitida_en','valida_hasta'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
BEGIN
 IF vec_autorizacion_atestada_v3.login_meritos_cartera_valido_v1() IS NOT TRUE
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD262: lectura sin emisor nominal' USING ERRCODE='42501';END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD262: lista de lectura inválida' USING ERRCODE='22023';END IF;
  ELSIF v_clave IN ('limite','asignacion_version','version_rol_version',
    'revision_control_version_rol','revision_catalogo_politicas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
   OR coalesce(p_orden->>v_clave ~ '^[1-9][0-9]{0,19}$',false) IS NOT TRUE
   THEN RAISE EXCEPTION 'AD262: versión de lectura inválida' USING ERRCODE='22023';END IF;
  ELSIF v_clave='cursor_hecho_ref' THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave)>256
   THEN RAISE EXCEPTION 'AD262: cursor inválido' USING ERRCODE='22023';END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD262: orden de lectura inválida' USING ERRCODE='22023';END IF;
 END LOOP;
 IF p_orden->>'esquema' IS DISTINCT FROM 'vec.meritos.cartera.propia.v1'
 OR p_orden->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR (p_orden->>'limite')::integer NOT BETWEEN 1 AND 50
 OR p_orden->>'filtro' NOT IN ('todos','disponibles','retirados')
 OR p_orden->>'idioma' NOT IN ('es','en')
 OR p_orden->>'accion' IS DISTINCT FROM 'meritos.cartera.consultar_propia'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM 'cartera_personal'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM 'cartera:'||(p_orden->>'persona_ref')
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_cartera_propia'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_cartera_meritos_correcta'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->'campos_permitidos' NOT IN (
  '["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb,
  '["denominacion","disponibilidad","evidencias","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb)
 THEN RAISE EXCEPTION 'AD262: contrato positivo inválido' USING ERRCODE='22023';END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD262: evidencia de lectura inválida' USING ERRCODE='22023';END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_contexto->>'principal_ref' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_contexto->>'persona_ref' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM p_orden->>'persona_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR v_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 THEN RAISE EXCEPTION 'AD262: contexto externo divergente' USING ERRCODE='42501';END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_orden->>'registro_contexto_ref',v_contexto->>'esquema',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',v_vinculo->>'autoridad_efectiva',
  v_contexto->>'cuenta_ref',(v_contexto->>'cuenta_version')::numeric,
  v_contexto->>'persona_ref',(v_contexto->>'persona_version')::numeric,
  v_contexto->>'perfil_activo_ref',(v_contexto->>'perfil_version')::numeric,
  v_contexto->>'contexto_actor_ref',(v_contexto->>'contexto_version')::numeric,
  v_contexto->>'metodo',v_contexto->>'garantia',
  (p_orden->>'emitida_en')::timestamptz,(p_orden->>'valida_hasta')::timestamptz);
 IF v_acreditada IS NULL
 OR vec_autorizacion.cotejar_cartera_meritos_actual_v1(p_orden,v_vinculo) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD262: autoridad de lectura revocada' USING ERRCODE='42501';END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_cartera_meritos.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN RAISE EXCEPTION 'AD262: lectura repetida' USING ERRCODE='23505';END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_mc_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_cartera_meritos.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_cartera_meritos',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,
  p_orden->>'lectura_instantanea_sha256');
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $_$;

DO $cartera_post$
DECLARE f oid:='vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(bytea,bytea,jsonb)'::regprocedure; snapshot jsonb;
BEGIN
 snapshot:=current_setting('vec_ad273.cartera_pre',true)::jsonb;
 IF snapshot IS NULL
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f)<>'cee855194ebab7fad3dadc2eaea8bab76368cf55396b89a351c3474c11157940'
    OR (SELECT jsonb_build_object('meta',to_jsonb(p)-'prosrc',
      'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f),
      'shdeps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
       FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f))
      FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM snapshot
 THEN RAISE EXCEPTION 'AD273: cartera postimagen incompatible' USING ERRCODE='55000'; END IF;
END $cartera_post$;


CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record; accion text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR
  session_user<>'vec_externo_documentos_datos' OR
  NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
   AND m.roleid='vec_documentos_subida_datos'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option) OR
  (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
    p.oid=to_regprocedure('vec_autorizacion.acciones_meritos_cartera_v2()')
    AND p.proowner='vec_autorizacion_propietario'::regrole)
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
    p.oid=to_regprocedure('vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)')
    AND p.proowner='vec_autorizacion_propietario'::regrole)
  OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
    p.oid=to_regprocedure('vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)')
    AND p.proowner='vec_autorizacion_propietario'::regrole)
 THEN RAISE EXCEPTION 'AD270: consumidor nominal denegado' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD270: material inválido' USING ERRCODE='22023'; END;
 accion:=d->>'accion';
 IF accion IS NULL OR accion NOT IN ('meritos.cartera.comprobante.subir_propio','meritos.cartera.comprobante.descargar_propio') OR
  c->>'operacion' IS DISTINCT FROM accion OR
  c->>'audiencia_consumo' IS DISTINCT FROM (CASE accion
   WHEN 'meritos.cartera.comprobante.subir_propio' THEN 'vec_documentos.subida_local.acto_modulo.v1'
   ELSE 'vec_documentos.subida_local.descarga_propia.v1' END) OR
  d->>'modulo_id' IS DISTINCT FROM 'meritos' OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos' OR
  d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal' OR
  d->'campos_permitidos' IS DISTINCT FROM (CASE accion
   WHEN 'meritos.cartera.comprobante.subir_propio' THEN '["contenido","estado","recibo"]'::jsonb
   ELSE '["contenido"]'::jsonb END) OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD270: acto no corresponde' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'documentos_subida_local_externa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM TRUE OR v.decision_ref IS DISTINCT FROM c->>'decision_ref' OR
  v.efecto_ref IS DISTINCT FROM c->>'efecto_ref' OR v.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'AD270: consumo incoherente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,
  v.auditoria_ref,v.consumida_en,v.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb;d jsonb;v record;
BEGIN
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD271: material económico inválido' USING ERRCODE='22023';
 END;
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR vec_autorizacion_atestada_v3.login_liquidacion_dietas_valido_v1() IS NOT TRUE
    OR c->>'operacion' IS DISTINCT FROM 'dietas.liquidacion_economica.registrar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.liquidacion_economica.registrar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'liquidacion_economica'
    OR d->>'finalidad' IS DISTINCT FROM 'registrar_liquidacion_economica_dietas'
    OR (d->>'recurso_ref' ~ '^dco_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["constancia","economia","esquema","snapshot_sha256"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD271: operación económica denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'registro_economico_dietas',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'AD271: registro económico requiere consumo nuevo' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,
  v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

-- La autorización de lectura pertenece al PDP central. La procedencia de
-- comisión, unidad y titular la aporta exclusivamente la fachada de Dietas.
-- lectura_instantanea_sha256 identifica la instantánea de permisos, no el
-- contenido económico que devuelve la fachada propietaria.
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.cotejar_lectura_liquidacion_dietas_actual_v1(
 p_orden jsonb,p_principal text,p_perfil text,p_persona text,p_contexto bytea)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE a record;r record;control record;catalogo record;politicas jsonb;contexto jsonb;ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'dietas.liquidacion_economica.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'dietas'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'consultar_liquidacion_economica_dietas'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'unidad_ref' !~ '^[A-Za-z0-9:_-]{3,128}$'
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'unidad_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(p_orden->>'titular_persona_ref',512) IS NOT TRUE
 OR p_orden->'campos_permitidos' IS DISTINCT FROM '["constancia","economia","esquema","snapshot_sha256"]'::jsonb
 OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 contexto:=convert_from(p_contexto,'UTF8')::jsonb;
 IF jsonb_typeof(contexto) IS DISTINCT FROM 'object'
 OR contexto->>'persona_ref' IS DISTINCT FROM p_persona
 OR jsonb_typeof(contexto->'vinculos') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 IF (p_orden->>'emitida_en')::timestamptz>ahora
 OR (p_orden->>'valida_hasta')::timestamptz<=ahora
 OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds'
 THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual ap
 JOIN vec_autorizacion.asignacion_perfil x ON x.perfil_activo_ref=ap.perfil_activo_ref
  AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_perfil FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=a.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
  ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=r.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT COALESCE(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
  ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 RETURN a.principal_id IS NOT DISTINCT FROM p_principal
  AND a.perfil_activo_ref IS NOT DISTINCT FROM p_perfil
  AND a.asignacion_ref IS NOT DISTINCT FROM p_orden->>'asignacion_ref'
  AND a.version IS NOT DISTINCT FROM (p_orden->>'asignacion_version')::numeric
  AND a.huella_sha256 IS NOT DISTINCT FROM p_orden->>'asignacion_huella_sha256'
  AND a.documento->>'estado' IS NOT DISTINCT FROM 'activa'
  AND ahora>=(a.documento->>'vigente_desde')::timestamptz
  AND ahora<(a.documento->>'vigente_hasta')::timestamptz
  AND r.version_rol_ref IS NOT DISTINCT FROM p_orden->>'version_rol_ref'
  AND r.version IS NOT DISTINCT FROM (p_orden->>'version_rol_version')::numeric
  AND r.huella_sha256 IS NOT DISTINCT FROM p_orden->>'version_rol_huella_sha256'
  AND r.documento->>'estado' IS NOT DISTINCT FROM 'publicada'
  AND control.version_rol_ref IS NOT DISTINCT FROM p_orden->>'control_version_rol_ref'
  AND control.revision IS NOT DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
  AND control.huella_sha256 IS NOT DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
  AND control.estado IS NOT DISTINCT FROM 'habilitada'
  AND catalogo.revision IS NOT DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
  AND catalogo.huella_sha256 IS NOT DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
  AND politicas IS NOT DISTINCT FROM p_orden->'politicas_evaluadas'
  -- Esta captura no lleva MAC del resultado del PDP. Hasta que exista un
  -- cotejo SQL equivalente de reglas restrictivas, toda política actual cierra
  -- solo esta lectura nominal, aun si la orden afirma que fue evaluada.
  AND politicas='[]'::jsonb
  AND NOT EXISTS (SELECT 1 FROM vec_autorizacion.politica_restrictiva_actual)
  AND jsonb_typeof(a.documento->'ambitos')='array'
  AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
  AND jsonb_array_length(a.documento->'ambitos') BETWEEN 1 AND 3
  AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') amb(x)
   WHERE amb.x->>'clave' IS NULL
    OR amb.x->>'clave' NOT IN ('persona_ref','unidad_ref','empleado_ref')
    OR jsonb_typeof(amb.x->'valores') IS DISTINCT FROM 'array'
    OR jsonb_array_length(amb.x->'valores')<>1
    OR CASE amb.x->>'clave'
      WHEN 'persona_ref' THEN amb.x->'valores' IS DISTINCT FROM jsonb_build_array(p_persona)
      WHEN 'unidad_ref' THEN amb.x->'valores' IS DISTINCT FROM jsonb_build_array(p_orden->>'unidad_ref')
      WHEN 'empleado_ref' THEN
       (SELECT count(*) FROM jsonb_array_elements(contexto->'vinculos') v(x)
        WHERE v.x->>'tipo'='empleado')<>1
       OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(contexto->'vinculos') v(x)
        WHERE v.x->>'tipo'='empleado' AND v.x->>'estado'='activo'
         AND ahora>=(v.x->>'vigente_desde')::timestamptz
         AND ahora<(v.x->>'vigente_hasta')::timestamptz
         AND amb.x->'valores'=jsonb_build_array(v.x->>'referencia'))
      ELSE true END)
  AND CASE WHEN p_persona=p_orden->>'titular_persona_ref' THEN
   a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object(
    'clave','persona_ref','valores',jsonb_build_array(p_persona)))
   ELSE r.documento->>'rol_id' IN ('dietas_revision_administrativa',
    'dietas_autorizacion','dietas_liquidacion_rrhh','dietas_fiscalizacion')
    AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object(
     'clave','unidad_ref','valores',jsonb_build_array(p_orden->>'unidad_ref')))
    AND a.documento->'ambitos' @> jsonb_build_array(
     jsonb_build_object('clave','persona_ref','valores',jsonb_build_array(p_persona)))
   END
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') con
   WHERE con->>'accion'='dietas.liquidacion_economica.consultar')=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') con
   WHERE con->>'accion'='dietas.liquidacion_economica.consultar'
    AND con->>'modulo_id'='dietas'
    AND con->>'tipo_recurso'='liquidacion_economica'
    AND con->'finalidades'='["consultar_liquidacion_economica_dietas"]'::jsonb
    AND con->'campos_permitidos'=p_orden->'campos_permitidos'
    AND con->>'garantia_minima'='alto'
    AND con->'obligaciones' IN ('[]'::jsonb,'["auditar"]'::jsonb));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_lectura_liquidacion_dietas_actual_v1(jsonb,text,text,text,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_lectura_liquidacion_dietas_actual_v1(jsonb,text,text,text,bytea)
 TO vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion.cotejar_lectura_justificacion_cronos_actual_v1(
 p_orden jsonb,p_principal text,p_perfil text,p_persona text,p_contexto bytea)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE a record;r record;control record;catalogo record;politicas jsonb;contexto jsonb;
 ahora timestamptz(6);accion text;finalidad text;campos jsonb;objetivo text;
 propia boolean;fase_previa boolean;empleado_actual text;empleados_activos integer;
BEGIN
 accion:=p_orden->>'accion';objetivo:=p_orden->>'empleado_objetivo_ref';
 fase_previa:=accion<>'cronos.justificacion.bandeja.consultar'
  AND NOT (p_orden ? 'titular_persona_ref') AND NOT (p_orden ? 'empleado_objetivo_ref');
 finalidad:=CASE accion
  WHEN 'cronos.justificacion.propia.consultar' THEN 'consultar_justificacion_propia'
  WHEN 'cronos.justificacion.rrhh.consultar' THEN 'consultar_justificacion_rrhh'
  WHEN 'cronos.justificacion.bandeja.consultar' THEN 'consultar_bandeja_justificaciones'
  WHEN 'cronos.justificacion.subida.preparar' THEN 'preparar_subida_justificacion' END;
 campos:=CASE accion
  WHEN 'cronos.justificacion.propia.consultar' THEN '["estado_permiso","historia","pendiente_justificar","solicitud_ref","version_permiso"]'::jsonb
  WHEN 'cronos.justificacion.rrhh.consultar' THEN '["estado_permiso","historia","pendiente_justificar","solicitud_ref","version_permiso"]'::jsonb
  WHEN 'cronos.justificacion.bandeja.consultar' THEN '["pendientes"]'::jsonb
  WHEN 'cronos.justificacion.subida.preparar' THEN '["empleado_ref","persona_ref","solicitud_ref","version"]'::jsonb END;
 propia:=accion IN ('cronos.justificacion.propia.consultar','cronos.justificacion.subida.preparar');
 IF current_user<>'vec_autorizacion_propietario'
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object' OR finalidad IS NULL OR campos IS NULL
  OR p_orden->>'modulo_id' IS DISTINCT FROM 'cronos'
  OR p_orden->>'finalidad_ref' IS DISTINCT FROM finalidad
  OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
  OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
  OR p_orden->'campos_permitidos' IS DISTINCT FROM campos
  OR (accion='cronos.justificacion.bandeja.consultar' AND
      (p_orden->>'recurso_ref' IS DISTINCT FROM 'bandeja:cronos:justificaciones'
       OR p_orden ? 'titular_persona_ref' OR p_orden ? 'empleado_objetivo_ref'))
  OR (accion<>'cronos.justificacion.bandeja.consultar' AND
      p_orden->>'recurso_ref' !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
  OR (accion<>'cronos.justificacion.bandeja.consultar' AND NOT fase_previa AND
      (p_orden->>'titular_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
       OR objetivo !~ '^emp_[-A-Za-z0-9_]{22,128}$'))
  OR (accion='cronos.justificacion.subida.preparar' AND NOT fase_previa AND
      (p_orden->>'version_permiso' !~ '^[1-9][0-9]{0,18}$'
       OR p_orden->>'version_esperada' IS DISTINCT FROM p_orden->>'version_permiso'))
  OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 contexto:=convert_from(p_contexto,'UTF8')::jsonb;
 IF jsonb_typeof(contexto) IS DISTINCT FROM 'object'
  OR contexto->>'persona_ref' IS DISTINCT FROM p_persona
  OR jsonb_typeof(contexto->'vinculos') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 SELECT count(*),max(v.x->>'referencia') INTO empleados_activos,empleado_actual
 FROM jsonb_array_elements(contexto->'vinculos') v(x)
 WHERE v.x->>'tipo'='empleado' AND v.x->>'estado'='activo'
  AND ahora>=(v.x->>'vigente_desde')::timestamptz
  AND ahora<(v.x->>'vigente_hasta')::timestamptz;
 IF propia AND empleados_activos<>1 THEN RETURN false; END IF;
 IF fase_previa AND propia THEN objetivo:=empleado_actual; END IF;
 IF (p_orden->>'emitida_en')::timestamptz>ahora
  OR (p_orden->>'valida_hasta')::timestamptz<=ahora
  OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds'
 THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual ap
 JOIN vec_autorizacion.asignacion_perfil x ON x.perfil_activo_ref=ap.perfil_activo_ref
  AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_perfil FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=a.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
  ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=r.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
  ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 RETURN a.principal_id IS NOT DISTINCT FROM p_principal
  AND a.perfil_activo_ref IS NOT DISTINCT FROM p_perfil
  AND a.asignacion_ref IS NOT DISTINCT FROM p_orden->>'asignacion_ref'
  AND a.version IS NOT DISTINCT FROM (p_orden->>'asignacion_version')::numeric
  AND a.huella_sha256 IS NOT DISTINCT FROM p_orden->>'asignacion_huella_sha256'
  AND a.documento->>'estado' IS NOT DISTINCT FROM 'activa'
  AND ahora>=(a.documento->>'vigente_desde')::timestamptz
  AND ahora<(a.documento->>'vigente_hasta')::timestamptz
  AND r.version_rol_ref IS NOT DISTINCT FROM p_orden->>'version_rol_ref'
  AND r.version IS NOT DISTINCT FROM (p_orden->>'version_rol_version')::numeric
  AND r.huella_sha256 IS NOT DISTINCT FROM p_orden->>'version_rol_huella_sha256'
  AND r.documento->>'estado' IS NOT DISTINCT FROM 'publicada'
  AND control.version_rol_ref IS NOT DISTINCT FROM p_orden->>'control_version_rol_ref'
  AND control.revision IS NOT DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
  AND control.huella_sha256 IS NOT DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
  AND control.estado IS NOT DISTINCT FROM 'habilitada'
  AND catalogo.revision IS NOT DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
  AND catalogo.huella_sha256 IS NOT DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
  AND politicas IS NOT DISTINCT FROM p_orden->'politicas_evaluadas'
  AND politicas='[]'::jsonb
  AND NOT EXISTS (SELECT 1 FROM vec_autorizacion.politica_restrictiva_actual)
  AND jsonb_typeof(a.documento->'ambitos')='array'
  AND vec_autorizacion.ambitos_positivos_validos(a.documento) IS TRUE
  AND jsonb_array_length(a.documento->'ambitos') BETWEEN 1 AND 3
  AND a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object(
    'clave','persona_ref','valores',jsonb_build_array(p_persona)))
  AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') amb(x)
   WHERE amb.x->>'clave' IS NULL
    OR amb.x->>'clave' NOT IN ('persona_ref','empleado_ref')
    OR jsonb_typeof(amb.x->'valores') IS DISTINCT FROM 'array'
    OR jsonb_array_length(amb.x->'valores')<>1
    OR (amb.x->>'clave'='persona_ref' AND
        amb.x->'valores' IS DISTINCT FROM jsonb_build_array(p_persona))
    OR (amb.x->>'clave'='empleado_ref' AND
        (accion='cronos.justificacion.bandeja.consultar' OR
         amb.x->'valores' IS DISTINCT FROM jsonb_build_array(objetivo))))
  AND (NOT propia OR
       a.documento->'ambitos' @> jsonb_build_array(jsonb_build_object(
        'clave','empleado_ref','valores',jsonb_build_array(objetivo))))
  AND (accion='cronos.justificacion.bandeja.consultar' OR fase_previa OR
       (propia AND p_persona IS NOT DISTINCT FROM p_orden->>'titular_persona_ref') OR
       (NOT propia AND p_persona IS DISTINCT FROM p_orden->>'titular_persona_ref'))
  AND (NOT propia OR EXISTS (SELECT 1 FROM jsonb_array_elements(contexto->'vinculos') v(x)
       WHERE v.x->>'tipo'='empleado' AND v.x->>'referencia'=objetivo
        AND v.x->>'estado'='activo' AND ahora>=(v.x->>'vigente_desde')::timestamptz
        AND ahora<(v.x->>'vigente_hasta')::timestamptz))
  AND (SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') con
       WHERE con->>'accion'=accion)=1
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') con
       WHERE con->>'accion'=accion AND con->>'modulo_id'='cronos'
        AND con->>'tipo_recurso'='justificacion'
        AND con->'finalidades'=jsonb_build_array(finalidad)
        AND con->'campos_permitidos'=campos
        AND con->>'garantia_minima'='alto'
        AND con->'obligaciones'='["auditar"]'::jsonb)
  AND (accion<>'cronos.justificacion.bandeja.consultar' OR
       ((SELECT count(*) FROM jsonb_array_elements(r.documento->'concesiones') con
         WHERE con->>'accion'='cronos.justificacion.rrhh.consultar')=1
        AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') con
         WHERE con->>'accion'='cronos.justificacion.rrhh.consultar'
          AND con->>'modulo_id'='cronos' AND con->>'tipo_recurso'='justificacion'
          AND con->'finalidades'='["consultar_justificacion_rrhh"]'::jsonb
          AND con->'campos_permitidos'='["estado_permiso","historia","pendiente_justificar","solicitud_ref","version_permiso"]'::jsonb
          AND con->>'garantia_minima'='alto'
          AND con->'obligaciones'='["auditar"]'::jsonb)));
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_lectura_justificacion_cronos_actual_v1(jsonb,text,text,text,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_lectura_justificacion_cronos_actual_v1(jsonb,text,text,text,bytea)
 TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$
DECLARE anterior text;nuevo text;marca text:='''lectura_auditoria_rrhh''::text';h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 IF h IS DISTINCT FROM '939babe4fcb650d87a774a559db7fdd74f0eb64f133a4b6b227821b5b34c468e'
 OR to_regprocedure('vec_autorizacion_atestada_v3.auditoria_confianza_externa_valida_v1(jsonb)') IS NULL
 OR position('auditoria_confianza_externa_valida_v1' IN anterior)=0
 OR (length(anterior)-length(replace(anterior,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AD271: preimagen de familia auditora divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(anterior,marca,marca||', ''lectura_liquidacion_dietas''::text, ''lectura_justificacion_cronos''::text');
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nuevo;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
    encode(sha256(convert_to(nuevo,'UTF8')),'hex') OR anterior IS DISTINCT FROM nuevo
    OR position('auditoria_confianza_externa_valida_v1' IN anterior)=0
 THEN RAISE EXCEPTION 'AD271: postimagen de familia auditora divergente' USING ERRCODE='55000'; END IF;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_liquidacion_dietas_formato_v1 CHECK (
 tipo_registro<>'lectura_liquidacion_dietas' OR (
 intento_ref ~ '^lectura_[0-9a-f]{32}$' AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND contexto_sha256 ~ '^[0-9a-f]{64}$' AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
 AND autenticacion_sha256 ~ '^[0-9a-f]{64}$' AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
 AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
 AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
 AND modulo_id='dietas' AND proceso='vec-server' AND canal='interna_corporativa'
 AND accion='dietas.liquidacion_economica.consultar'
 AND recurso_ref ~ '^dco_[A-Za-z0-9_-]{22,128}$'
 AND finalidad_ref='consultar_liquidacion_economica_dietas' AND resultado='obtenida'
 AND motivo_ref='lectura_liquidacion_dietas_correcta'
 AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'));
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_justificacion_cronos_formato_v1 CHECK (
 tipo_registro<>'lectura_justificacion_cronos' OR (
 intento_ref ~ '^lectura_[0-9a-f]{32}$' AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND contexto_sha256 ~ '^[0-9a-f]{64}$' AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
 AND autenticacion_sha256 ~ '^[0-9a-f]{64}$' AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
 AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
 AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
 AND modulo_id='cronos' AND proceso='vec-server' AND canal='interna_corporativa'
 AND resultado='obtenida' AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 AND ((accion='cronos.justificacion.propia.consultar'
   AND finalidad_ref='consultar_justificacion_propia'
   AND motivo_ref='lectura_justificacion_cronos_propia_correcta'
   AND recurso_ref ~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
  OR (accion='cronos.justificacion.rrhh.consultar'
   AND finalidad_ref='consultar_justificacion_rrhh'
   AND motivo_ref='lectura_justificacion_cronos_rrhh_correcta'
   AND recurso_ref ~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
  OR (accion='cronos.justificacion.bandeja.consultar'
   AND finalidad_ref='consultar_bandeja_justificaciones'
   AND motivo_ref='lectura_bandeja_justificaciones_cronos_correcta'
   AND recurso_ref='bandeja:cronos:justificaciones')
  OR (accion='cronos.justificacion.subida.preparar'
   AND finalidad_ref='preparar_subida_justificacion'
   AND motivo_ref='preparacion_subida_justificacion_cronos_correcta'
   AND recurso_ref ~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'))));
RESET ROLE;


SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_sesion record;
 v_claves constant text[]:=ARRAY[
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'recurso_ref','unidad_ref','titular_persona_ref','finalidad_ref',
  'campos_permitidos','resultado','motivo_ref','proceso','canal','correlacion_ref',
  'lectura_revision_permisos','lectura_instantanea_sha256','asignacion_ref',
  'asignacion_version','asignacion_huella_sha256','version_rol_ref',
  'version_rol_version','version_rol_huella_sha256','control_version_rol_ref',
  'revision_control_version_rol','control_version_rol_huella_sha256',
  'revision_catalogo_politicas','catalogo_politicas_huella_sha256',
  'politicas_evaluadas','emitida_en','valida_hasta'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR vec_autorizacion_atestada_v3.login_liquidacion_dietas_valido_v1() IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD271: lectura sin emisor acreditado' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD271: lista de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF v_clave IN ('asignacion_version','version_rol_version',
                    'revision_control_version_rol','revision_catalogo_politicas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
   OR (p_orden->>v_clave) !~ '^[1-9][0-9]{0,19}$'
   THEN RAISE EXCEPTION 'AD271: versión de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD271: orden de lectura inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'unidad_ref' !~ '^[A-Za-z0-9:_-]{3,128}$'
 OR p_orden->>'titular_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'accion' IS DISTINCT FROM 'dietas.liquidacion_economica.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'dietas'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'consultar_liquidacion_economica_dietas'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_liquidacion_dietas_correcta'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'unidad_ref' ~ '[[:space:][:cntrl:]]'
 OR strpos(p_orden->>'unidad_ref','*')>0
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->'campos_permitidos' IS DISTINCT FROM
    '["constancia","economia","esquema","snapshot_sha256"]'::jsonb
 THEN RAISE EXCEPTION 'AD271: contrato de lectura inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD271: evidencia de lectura inválida' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',p_contexto_canonico) IS NOT TRUE
 OR vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(
  p_contexto_canonico) IS NOT TRUE
 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
  p_vinculo_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD271: vínculo de lectura inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_sesion FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref');
 IF NOT FOUND THEN RAISE EXCEPTION 'AD271: sesión actual ausente' USING ERRCODE='42501'; END IF;
 IF v_sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_sesion.cuenta_ref IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_sesion.cuenta_ordinaria_ref IS DISTINCT FROM v_vinculo->>'cuenta_ordinaria_ref'
 OR v_sesion.control_sesion_ref IS DISTINCT FROM v_vinculo->>'control_sesion_ref'
 OR v_sesion.control_sesion_revision IS DISTINCT FROM v_vinculo->>'control_sesion_revision'
 OR v_sesion.control_sesion_huella_sha256 IS DISTINCT FROM v_vinculo->>'control_sesion_huella_sha256'
 OR v_sesion.superficie IS DISTINCT FROM 'interna_corporativa'
 OR v_sesion.cuenta_privilegiada IS NOT FALSE
 OR v_sesion.garantia_observada IS DISTINCT FROM 'alto'
 OR v_sesion.garantia_observada IS DISTINCT FROM v_vinculo->>'garantia_observada'
 OR v_sesion.sesion_valida_hasta IS DISTINCT FROM (v_vinculo->>'sesion_valida_hasta')::timestamptz
 OR v_sesion.sesion_revalidada_en IS DISTINCT FROM (v_vinculo->>'sesion_revalidada_en')::timestamptz
 OR vec_autorizacion.cotejar_lectura_liquidacion_dietas_actual_v1(
  p_orden,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref',
  v_contexto->>'persona_ref',p_contexto_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD271: autoridad actual de lectura denegada' USING ERRCODE='42501'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_liquidacion_dietas.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN RAISE EXCEPTION 'AD271: lectura ya asentada' USING ERRCODE='23505'; END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_ld_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_liquidacion_dietas.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_liquidacion_dietas',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,
  p_orden->>'lectura_instantanea_sha256');
 RETURN jsonb_build_object('auditoria_ref',v_ref,'secuencia',v_secuencia,
  'huella_sha256',v_huella,'correlacion_ref',p_orden->>'correlacion_ref',
  'registrada_en',v_instante);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(bytea,bytea,jsonb)
 TO vec_dietas_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.login_lectura_justificacion_cronos_valido_v1(p_accion text)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
  SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
  JOIN pg_roles g ON g.oid=m.roleid
  WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL
   AND ((p_accion='cronos.justificacion.subida.preparar'
      AND l.rolname='vec_interno_documentos_cronos_datos' AND g.rolname='vec_documentos_ejecutor')
    OR (p_accion IN ('cronos.justificacion.propia.consultar','cronos.justificacion.rrhh.consultar',
       'cronos.justificacion.bandeja.consultar') AND g.rolname='vec_cronos_v1_ejecutor'))
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
   AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)));
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_lectura_justificacion_cronos_valido_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_sesion_lectura_justificacion_cronos_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_sesion record;v_acreditada timestamptz;
 v_empleado text;v_empleados integer;
 v_claves text[]:=ARRAY[
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'recurso_ref','finalidad_ref','campos_permitidos','resultado','motivo_ref',
  'proceso','canal','correlacion_ref','lectura_revision_permisos',
  'lectura_instantanea_sha256','asignacion_ref','asignacion_version',
  'asignacion_huella_sha256','version_rol_ref','version_rol_version',
  'version_rol_huella_sha256','control_version_rol_ref',
  'revision_control_version_rol','control_version_rol_huella_sha256',
  'revision_catalogo_politicas','catalogo_politicas_huella_sha256',
  'politicas_evaluadas','emitida_en','valida_hasta' ];
 v_clave text;v_accion text;v_finalidad text;v_motivo text;v_campos jsonb;
BEGIN
 v_accion:=p_orden->>'accion';
 v_finalidad:=CASE v_accion
  WHEN 'cronos.justificacion.propia.consultar' THEN 'consultar_justificacion_propia'
  WHEN 'cronos.justificacion.rrhh.consultar' THEN 'consultar_justificacion_rrhh'
  WHEN 'cronos.justificacion.bandeja.consultar' THEN 'consultar_bandeja_justificaciones'
  WHEN 'cronos.justificacion.subida.preparar' THEN 'preparar_subida_justificacion' END;
 v_motivo:=CASE v_accion
  WHEN 'cronos.justificacion.propia.consultar' THEN 'lectura_justificacion_cronos_propia_correcta'
  WHEN 'cronos.justificacion.rrhh.consultar' THEN 'lectura_justificacion_cronos_rrhh_correcta'
  WHEN 'cronos.justificacion.bandeja.consultar' THEN 'lectura_bandeja_justificaciones_cronos_correcta'
  WHEN 'cronos.justificacion.subida.preparar' THEN 'preparacion_subida_justificacion_cronos_correcta' END;
 v_campos:=CASE v_accion
  WHEN 'cronos.justificacion.propia.consultar' THEN '["estado_permiso","historia","pendiente_justificar","solicitud_ref","version_permiso"]'::jsonb
  WHEN 'cronos.justificacion.rrhh.consultar' THEN '["estado_permiso","historia","pendiente_justificar","solicitud_ref","version_permiso"]'::jsonb
  WHEN 'cronos.justificacion.bandeja.consultar' THEN '["pendientes"]'::jsonb
  WHEN 'cronos.justificacion.subida.preparar' THEN '["empleado_ref","persona_ref","solicitud_ref","version"]'::jsonb END;

 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR v_finalidad IS NULL OR v_motivo IS NULL OR v_campos IS NULL
 OR vec_autorizacion_atestada_v3.login_lectura_justificacion_cronos_valido_v1(v_accion) IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD273-C8: lectura sin emisor acreditado' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD273-C8: lista de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF v_clave IN ('asignacion_version','version_rol_version',
    'revision_control_version_rol','revision_catalogo_politicas','version_permiso','version_esperada') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
    OR (p_orden->>v_clave) !~ '^[1-9][0-9]{0,19}$'
   THEN RAISE EXCEPTION 'AD273-C8: versión de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD273-C8: orden de lectura inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'cronos'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM v_finalidad
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM v_motivo
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->'campos_permitidos' IS DISTINCT FROM v_campos
 OR (v_accion='cronos.justificacion.bandeja.consultar' AND
     p_orden->>'recurso_ref' IS DISTINCT FROM 'bandeja:cronos:justificaciones')
 OR (v_accion<>'cronos.justificacion.bandeja.consultar' AND
     p_orden->>'recurso_ref' !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
 THEN RAISE EXCEPTION 'AD273-C8: contrato de lectura inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD273-C8: evidencia de lectura inválida' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',p_contexto_canonico) IS NOT TRUE

 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
  p_vinculo_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD273-C8: vínculo de lectura inválido' USING ERRCODE='42501'; END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_orden->>'registro_contexto_ref',v_contexto->>'esquema',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',v_vinculo->>'autoridad_efectiva',
  v_contexto->>'cuenta_ref',(v_contexto->>'cuenta_version')::numeric,
  v_contexto->>'persona_ref',(v_contexto->>'persona_version')::numeric,
  v_contexto->>'perfil_activo_ref',(v_contexto->>'perfil_version')::numeric,
  v_contexto->>'contexto_actor_ref',(v_contexto->>'contexto_version')::numeric,
  v_contexto->>'metodo',v_contexto->>'garantia',
  (p_orden->>'emitida_en')::timestamptz,(p_orden->>'valida_hasta')::timestamptz);
 IF v_acreditada IS NULL THEN
  RAISE EXCEPTION 'AD273-C8: contexto actual de lectura denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_sesion FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref');
 IF NOT FOUND THEN RAISE EXCEPTION 'AD273-C8: sesión actual ausente' USING ERRCODE='42501'; END IF;
 IF v_sesion.autenticacion_ref IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_sesion.asercion_ref IS DISTINCT FROM v_vinculo->>'asercion_ref'
 OR v_sesion.sesion_ref IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_sesion.metodo_observado IS DISTINCT FROM v_vinculo->>'metodo_observado'
 OR v_sesion.politica_garantia_ref IS DISTINCT FROM v_vinculo->>'politica_garantia_ref'
 OR v_sesion.politica_garantia_huella_sha256 IS DISTINCT FROM v_vinculo->>'politica_garantia_huella_sha256'
 OR v_sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_sesion.cuenta_ref IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_sesion.cuenta_ordinaria_ref IS DISTINCT FROM v_vinculo->>'cuenta_ordinaria_ref'
 OR v_sesion.control_sesion_ref IS DISTINCT FROM v_vinculo->>'control_sesion_ref'
 OR v_sesion.control_sesion_revision IS DISTINCT FROM v_vinculo->>'control_sesion_revision'
 OR v_sesion.control_sesion_huella_sha256 IS DISTINCT FROM v_vinculo->>'control_sesion_huella_sha256'
 OR v_sesion.superficie IS DISTINCT FROM 'interna_corporativa'
 OR v_sesion.cuenta_privilegiada IS NOT FALSE
 OR v_sesion.garantia_observada IS DISTINCT FROM 'alto'
 OR v_sesion.garantia_observada IS DISTINCT FROM v_vinculo->>'garantia_observada'
 OR v_sesion.sesion_valida_hasta IS DISTINCT FROM (v_vinculo->>'sesion_valida_hasta')::timestamptz
 OR v_sesion.sesion_revalidada_en IS DISTINCT FROM (v_vinculo->>'sesion_revalidada_en')::timestamptz
 OR vec_autorizacion.cotejar_lectura_justificacion_cronos_actual_v1(
  p_orden,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref',
  v_contexto->>'persona_ref',p_contexto_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD273-C8: autoridad actual de lectura denegada' USING ERRCODE='42501'; END IF;
 SELECT count(*),max(v.x->>'referencia') INTO v_empleados,v_empleado
 FROM jsonb_array_elements(v_contexto->'vinculos') v(x)
 WHERE v.x->>'tipo'='empleado' AND v.x->>'estado'='activo'
   AND clock_timestamp()>=(v.x->>'vigente_desde')::timestamptz
   AND clock_timestamp()<(v.x->>'vigente_hasta')::timestamptz;
 IF v_empleados<>1 OR v_empleado !~ '^emp_[-A-Za-z0-9_]{22,128}$'
 THEN RAISE EXCEPTION 'AD273-C8: empleado de sesión incompatible' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('persona_ref',v_contexto->>'persona_ref',
  'empleado_ref',v_empleado,'perfil_ref',v_contexto->>'perfil_activo_ref',
  'registro_contexto_ref',p_orden->>'registro_contexto_ref',
  'contexto_sha256',p_orden->>'contexto_sha256',
  'autenticacion_ref',p_orden->>'autenticacion_ref','sesion_ref',p_orden->>'sesion_ref');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.acreditar_sesion_lectura_justificacion_cronos_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.acreditar_sesion_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)
 TO vec_cronos_v1_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_sesion record;v_acreditada timestamptz;
 v_claves text[]:=ARRAY[
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'recurso_ref','finalidad_ref','campos_permitidos','resultado','motivo_ref',
  'proceso','canal','correlacion_ref','lectura_revision_permisos',
  'lectura_instantanea_sha256','asignacion_ref','asignacion_version',
  'asignacion_huella_sha256','version_rol_ref','version_rol_version',
  'version_rol_huella_sha256','control_version_rol_ref',
  'revision_control_version_rol','control_version_rol_huella_sha256',
  'revision_catalogo_politicas','catalogo_politicas_huella_sha256',
  'politicas_evaluadas','emitida_en','valida_hasta' ];
 v_clave text;v_accion text;v_finalidad text;v_motivo text;v_campos jsonb;
 v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;
BEGIN
 v_accion:=p_orden->>'accion';
 v_finalidad:=CASE v_accion
  WHEN 'cronos.justificacion.propia.consultar' THEN 'consultar_justificacion_propia'
  WHEN 'cronos.justificacion.rrhh.consultar' THEN 'consultar_justificacion_rrhh'
  WHEN 'cronos.justificacion.bandeja.consultar' THEN 'consultar_bandeja_justificaciones'
  WHEN 'cronos.justificacion.subida.preparar' THEN 'preparar_subida_justificacion' END;
 v_motivo:=CASE v_accion
  WHEN 'cronos.justificacion.propia.consultar' THEN 'lectura_justificacion_cronos_propia_correcta'
  WHEN 'cronos.justificacion.rrhh.consultar' THEN 'lectura_justificacion_cronos_rrhh_correcta'
  WHEN 'cronos.justificacion.bandeja.consultar' THEN 'lectura_bandeja_justificaciones_cronos_correcta'
  WHEN 'cronos.justificacion.subida.preparar' THEN 'preparacion_subida_justificacion_cronos_correcta' END;
 v_campos:=CASE v_accion
  WHEN 'cronos.justificacion.propia.consultar' THEN '["estado_permiso","historia","pendiente_justificar","solicitud_ref","version_permiso"]'::jsonb
  WHEN 'cronos.justificacion.rrhh.consultar' THEN '["estado_permiso","historia","pendiente_justificar","solicitud_ref","version_permiso"]'::jsonb
  WHEN 'cronos.justificacion.bandeja.consultar' THEN '["pendientes"]'::jsonb
  WHEN 'cronos.justificacion.subida.preparar' THEN '["empleado_ref","persona_ref","solicitud_ref","version"]'::jsonb END;
 IF v_accion IN ('cronos.justificacion.propia.consultar','cronos.justificacion.rrhh.consultar') THEN
  v_claves:=v_claves||ARRAY['titular_persona_ref','empleado_objetivo_ref'];
 ELSIF v_accion='cronos.justificacion.subida.preparar' THEN
  v_claves:=v_claves||ARRAY['titular_persona_ref','empleado_objetivo_ref','version_permiso','version_esperada'];
 END IF;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR v_finalidad IS NULL OR v_motivo IS NULL OR v_campos IS NULL
 OR vec_autorizacion_atestada_v3.login_lectura_justificacion_cronos_valido_v1(v_accion) IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)
 OR (p_orden ?& v_claves) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD273-C8: lectura sin emisor acreditado' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD273-C8: lista de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF v_clave IN ('asignacion_version','version_rol_version',
    'revision_control_version_rol','revision_catalogo_politicas','version_permiso','version_esperada') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
    OR (p_orden->>v_clave) !~ '^[1-9][0-9]{0,19}$'
   THEN RAISE EXCEPTION 'AD273-C8: versión de lectura inválida' USING ERRCODE='22023'; END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD273-C8: orden de lectura inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'cronos'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM v_finalidad
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM v_motivo
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->'campos_permitidos' IS DISTINCT FROM v_campos
 OR (v_accion='cronos.justificacion.bandeja.consultar' AND
     p_orden->>'recurso_ref' IS DISTINCT FROM 'bandeja:cronos:justificaciones')
 OR (v_accion<>'cronos.justificacion.bandeja.consultar' AND
     p_orden->>'recurso_ref' !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
 OR (v_accion IN ('cronos.justificacion.propia.consultar','cronos.justificacion.rrhh.consultar',
     'cronos.justificacion.subida.preparar') AND
     (p_orden->>'titular_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
      OR p_orden->>'empleado_objetivo_ref' !~ '^emp_[-A-Za-z0-9_]{22,128}$'))
 OR (v_accion='cronos.justificacion.subida.preparar' AND
     p_orden->>'version_permiso' IS DISTINCT FROM p_orden->>'version_esperada')
 THEN RAISE EXCEPTION 'AD273-C8: contrato de lectura inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD273-C8: evidencia de lectura inválida' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',p_contexto_canonico) IS NOT TRUE

 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
  p_vinculo_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD273-C8: vínculo de lectura inválido' USING ERRCODE='42501'; END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_orden->>'registro_contexto_ref',v_contexto->>'esquema',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',v_vinculo->>'autoridad_efectiva',
  v_contexto->>'cuenta_ref',(v_contexto->>'cuenta_version')::numeric,
  v_contexto->>'persona_ref',(v_contexto->>'persona_version')::numeric,
  v_contexto->>'perfil_activo_ref',(v_contexto->>'perfil_version')::numeric,
  v_contexto->>'contexto_actor_ref',(v_contexto->>'contexto_version')::numeric,
  v_contexto->>'metodo',v_contexto->>'garantia',
  (p_orden->>'emitida_en')::timestamptz,(p_orden->>'valida_hasta')::timestamptz);
 IF v_acreditada IS NULL THEN
  RAISE EXCEPTION 'AD273-C8: contexto actual de lectura denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_sesion FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref');
 IF NOT FOUND THEN RAISE EXCEPTION 'AD273-C8: sesión actual ausente' USING ERRCODE='42501'; END IF;
 IF v_sesion.autenticacion_ref IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_sesion.asercion_ref IS DISTINCT FROM v_vinculo->>'asercion_ref'
 OR v_sesion.sesion_ref IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_sesion.metodo_observado IS DISTINCT FROM v_vinculo->>'metodo_observado'
 OR v_sesion.politica_garantia_ref IS DISTINCT FROM v_vinculo->>'politica_garantia_ref'
 OR v_sesion.politica_garantia_huella_sha256 IS DISTINCT FROM v_vinculo->>'politica_garantia_huella_sha256'
 OR v_sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_sesion.cuenta_ref IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_sesion.cuenta_ordinaria_ref IS DISTINCT FROM v_vinculo->>'cuenta_ordinaria_ref'
 OR v_sesion.control_sesion_ref IS DISTINCT FROM v_vinculo->>'control_sesion_ref'
 OR v_sesion.control_sesion_revision IS DISTINCT FROM v_vinculo->>'control_sesion_revision'
 OR v_sesion.control_sesion_huella_sha256 IS DISTINCT FROM v_vinculo->>'control_sesion_huella_sha256'
 OR v_sesion.superficie IS DISTINCT FROM 'interna_corporativa'
 OR v_sesion.cuenta_privilegiada IS NOT FALSE
 OR v_sesion.garantia_observada IS DISTINCT FROM 'alto'
 OR v_sesion.garantia_observada IS DISTINCT FROM v_vinculo->>'garantia_observada'
 OR v_sesion.sesion_valida_hasta IS DISTINCT FROM (v_vinculo->>'sesion_valida_hasta')::timestamptz
 OR v_sesion.sesion_revalidada_en IS DISTINCT FROM (v_vinculo->>'sesion_revalidada_en')::timestamptz
 OR vec_autorizacion.cotejar_lectura_justificacion_cronos_actual_v1(
  p_orden,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref',
  v_contexto->>'persona_ref',p_contexto_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD273-C8: autoridad actual de lectura denegada' USING ERRCODE='42501'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_justificacion_cronos.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN RAISE EXCEPTION 'AD273-C8: lectura ya asentada' USING ERRCODE='23505'; END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_jc_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_justificacion_cronos.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_justificacion_cronos',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,
  p_orden->>'lectura_instantanea_sha256');
 RETURN jsonb_build_object('auditoria_ref',v_ref,'secuencia',v_secuencia,
  'huella_sha256',v_huella,'correlacion_ref',p_orden->>'correlacion_ref',
  'registrada_en',v_instante);
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)
 TO vec_cronos_v1_propietario;
RESET ROLE;


SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $preferencias_pre$
DECLARE d text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_class c
   WHERE c.oid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
     AND c.relowner='vec_autorizacion_atestada_v3_propietario'::regrole)
    OR EXISTS (SELECT 1 FROM pg_constraint c
      WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
        AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2')
 THEN RAISE EXCEPTION 'AD273: tabla o v2 de preferencias incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v1' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex')<>'0e538072984f48be856fc99544cbebd8d300f9b407fb4cba295090b70296435a'
 THEN RAISE EXCEPTION 'AD273: lectura de preferencias v1 incompatible' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec_ad273.preferencias_check_pre',d,true);
END $preferencias_pre$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 DROP CONSTRAINT auditoria_lectura_preferencias_bolsas_formato_v1;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_preferencias_bolsas_formato_v2 CHECK (
 tipo_registro<>'lectura_preferencias_bolsas' OR (
 intento_ref ~ '^lectura_[0-9a-f]{32}$' AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND contexto_sha256 ~ '^[0-9a-f]{64}$' AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
 AND autenticacion_sha256 ~ '^[0-9a-f]{64}$' AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
 AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
 AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
 AND modulo_id='bolsa' AND proceso='vec-server' AND canal IN ('externa_personal','interna_corporativa')
 AND accion='bolsa.preferencias_bolsas.consultar_propias'
 AND recurso_ref ~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$'
 AND finalidad_ref='gestion_preferencias_bolsas_propias' AND resultado='obtenida'
 AND motivo_ref='lectura_preferencias_bolsas_correcta'
 AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'));
DO $preferencias_post$
DECLARE d text; expected text;
BEGIN
 expected:=replace(current_setting('vec_ad273.preferencias_check_pre',true),
  'canal = ''externa_personal''::text',
  'canal = ANY (ARRAY[''externa_personal''::text, ''interna_corporativa''::text])');
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2' AND c.contype='c' AND c.convalidated;
 IF d IS DISTINCT FROM expected
    OR position('interna_corporativa' IN d)=0
    OR EXISTS (SELECT 1 FROM pg_constraint c
      WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
        AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v1')
 THEN RAISE EXCEPTION 'AD273: lectura de preferencias v2 incompatible' USING ERRCODE='55000'; END IF;
END $preferencias_post$;
RESET ROLE;

DO $post_ad273$
DECLARE q record; p record; config_actual text[];
BEGIN
 FOR q IN SELECT * FROM (VALUES
 ('vec_autorizacion_atestada_v3.login_liquidacion_dietas_valido_v1()','vec_autorizacion_atestada_v3_propietario','','9bf296512864980f6a4fd3b011a76546fea80779f73d94a75c77cdfde2e437df',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_bandeja_permisos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','6834477da796ea5477933ea10967cf281981c0fc8da65b9a19985221efeba846',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_permiso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','a538aa843afa9b67c0a545ad471cd669990358e5a69607c693968122c641faff',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_avisos_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','43f3b7ace73aff23f59a88879d5bba464b003ac3ef7ce67704890901fc0476fd',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_archivo_aviso_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','35ca759e56539a8a1392745e7ccbddc70a19dd748d22afdbbe3ec859fdf42c37',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_correccion_recibo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','bae4961e15aa0d494c695d9cad46ab91de26314fdd8d3faf8da1adfdc00dbce4',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_personal_propietario','fc1b1ba98e5bd846f03062c67faf65259f47865b6b29d576ad949d0b70d229b7',ARRAY['lock_timeout=2s','search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_unidad_trabajo_propia_cronos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_personal_propietario','d31b62a8360fe3ea74c706db32128cdd373f8e0fe62260706257c28b6c783ac4',ARRAY['lock_timeout=2s','search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_bandeja_correcciones_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','0a2d9b9a7f1a8f21bf99354d82b188c79e6ca7f533905140467cc6b9603864a4',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_decision_responsable_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','a9807682b905bf38fe46bb1a4bcea06d6019d759b8eca4d6992cdb8a33f6a195',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','8a9478a69cb76e820c15954793742da27ef55d67d4c41a7bd9386fada97e52cb',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','3aefe7e4380274f4bbad39a6ddac74231ad40750d5b9c47346d1239693dbaa99',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_propia_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','7d3515761bc3c10562ecb376fe990d53726f9afa0bd9321e7630084011c33623',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_historia_correccion_equipo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','4dd927b03c847fdf2b0a08a7ebe79055a52b4c9745fbdc35476627464eec7626',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_notificacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','ee6f03d7b6710dbe5e315d86d2e697503fe93a38f136426bf209e1ba8075cd82',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','28da20177c7588ee32295ef35946b48087e2d1d1d5b382ed699d6bd37310f654',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_notificaciones_bandeja_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','3f06b2f15b6b1abce3b44926f50a5fb301ad80a58a0cc247d2f5f6c95b8e4c72',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_atencion_notificacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','6929f89f533d1e4c13458b32ef1248eecc059e3b78ae3286e4034a029506820f',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_jefatura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','6d82921b79b4a87167d50ffd30d2470849e062dd37328efe7fe2f616bf0a971a',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_presencia_equipo_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','cb3939a8ba2f18128f374baca5a12751d529130cd578cd6c80189805478fe843',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_jefatura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','2391ff2e8b68e2746dc3b4c3421bdad87cc7f35cbe441ed7daaa5390e01b3c28',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_incidencias_equipo_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','a5eaec37c76581eb44c071151c2cbd0d1ca36e00d1c0e1c9a9f1598a4f5d9702',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_informe_propio_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','8af02280d2022345497dbdfb632319085248c367ada438bc27593b7c790c4313',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_jefatura_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','b4a10c69c837acb4aa5e6900e77a444a0dcbada1b2837a8b36f404954c662339',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_cronos_informe_equipo_rrhh_exportar_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','2a12040f23489509917fbd18917094f9cd8322332966a1c35c7fa63212c8b005',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_documentos_propietario','0206aa1a00f6097ee2b8d3fac7bf1505871b9075c22f7db9092d1c0040a95e03',ARRAY['lock_timeout=2s','search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_liquidacion_economica_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_dietas_propietario','96a6dcdd62f2dcaffb9555f2f2ce0d3dfd2d339ebc97c4bae1d84b8560380a16',ARRAY['lock_timeout=2s','search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_anexo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','796381c559d6a358d85bdd472081996f09f5d77cb9696c4f58d5b6b2051283b1',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_revision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','93c7bd2f7e0d80160426ec696d80435f6a22d025fae16d9f9e9cc82218b93297',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','vec_bolsa_llamamientos_propietario','5736caf5d9d92de023168afe8a7f4e1da4eda5e2a3486a899f954db9f3d6e2c8',ARRAY['lock_timeout=2s','search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion.cotejar_lectura_justificacion_cronos_actual_v1(jsonb,text,text,text,bytea)','vec_autorizacion_propietario','vec_autorizacion_atestada_v3_propietario','666369b4dddd47d8f7a36febb929ddd38cfaa4bd33ebf502a43c0ad8182b9d21',ARRAY['lock_timeout=2s','row_security=on','search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.login_lectura_justificacion_cronos_valido_v1(text)','vec_autorizacion_atestada_v3_propietario','','053b4acb03bb8462bc236d9da9cd3eb29fea6db9876cbe664d99a696d456cc95',ARRAY['search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.acreditar_sesion_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','766df91be21bbc5e6e5418ea5bc9c0eaee23db7c4d7252120bdaaaa18e6d5b5c',ARRAY['lock_timeout=2s','row_security=on','search_path=pg_catalog,pg_temp','statement_timeout=10s']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)','vec_autorizacion_atestada_v3_propietario','vec_cronos_v1_propietario','c9a44827cec9eb744f8d1d43c123ccc70aa0c7e46878560cbf0e163ff560a9e4',ARRAY['lock_timeout=2s','row_security=on','search_path=pg_catalog,pg_temp','statement_timeout=10s']::text[]),
 ('vec_autorizacion.cotejar_lectura_liquidacion_dietas_actual_v1(jsonb,text,text,text,bytea)','vec_autorizacion_propietario','vec_autorizacion_atestada_v3_propietario','001a6b240292def7648bddd584b17ce93e566310942a5ab18b0e7621def7972d',ARRAY['lock_timeout=2s','row_security=on','search_path=pg_catalog,pg_temp']::text[]),
 ('vec_autorizacion_atestada_v3.registrar_lectura_liquidacion_dietas_v1(bytea,bytea,jsonb)','vec_autorizacion_atestada_v3_propietario','vec_dietas_propietario','7c4168bc7600ef8a12ea883ce5a59a3152fa774011d193a0422100df89962bb6',ARRAY['lock_timeout=2s','row_security=on','search_path=pg_catalog,pg_temp','statement_timeout=10s']::text[])
 ) AS t(firma,propietario,destinatario,cuerpo_sha256,configuracion) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(q.firma);
  SELECT array_agg(replace(x,' ','') ORDER BY replace(x,' ','')) INTO config_actual
   FROM unnest(p.proconfig) AS x;
  IF p.proowner IS DISTINCT FROM q.propietario::regrole
    OR p.prosecdef IS DISTINCT FROM true
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM q.cuerpo_sha256
    OR config_actual IS DISTINCT FROM q.configuracion
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE a.privilege_type<>'EXECUTE'
         OR (a.grantee<>p.proowner AND a.grantee IS DISTINCT FROM nullif(q.destinatario,'')::regrole)
         OR (a.grantee<>p.proowner AND a.is_grantable))
    OR (q.destinatario<>'' AND
       (SELECT count(*) FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE a.grantee=nullif(q.destinatario,'')::regrole AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)<>1)
  THEN RAISE EXCEPTION 'AD273: postimagen de función incompatible: %',q.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname IN ('vec_documentos_subida_datos','vec_documentos_frontera_externa')
   AND (r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication
        OR r.rolbypassrls OR r.rolconfig IS NOT NULL
        OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.roleid=r.oid OR m.member=r.oid)))
   OR (SELECT count(*) FROM pg_roles WHERE rolname IN ('vec_documentos_subida_datos','vec_documentos_frontera_externa'))<>2
 THEN RAISE EXCEPTION 'AD273: postimagen de roles incompatible' USING ERRCODE='55000'; END IF;
END $post_ad273$;

COMMIT;
