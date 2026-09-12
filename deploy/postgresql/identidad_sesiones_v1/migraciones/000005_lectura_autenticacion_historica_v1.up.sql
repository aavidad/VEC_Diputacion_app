-- Lector de autenticacion ORIGINAL: no renovacion, no asercion ni permiso actual.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:base:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:lectura_historica:v1',0));
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL timezone='UTC';

DO $pre$
DECLARE r oid:='vec_identidad_sesiones_v1_lector_historico'::regrole;
 o oid:='vec_identidad_sesiones_v1_propietario'::regrole;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_identidad_sesiones_v1'::regnamespace
   AND proname='leer_autenticacion_original_v1')
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE oid='vec_identidad_sesiones_v1'::regnamespace AND nspowner=o)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=o AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
   AND rolconfig IS NULL AND rolconnlimit=-1 AND rolvaliduntil IS NULL)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_identidad_sesiones_v1'::regnamespace AND a.grantee=r)
 OR (SELECT count(*) FROM pg_class WHERE relnamespace='vec_identidad_sesiones_v1'::regnamespace
   AND relkind='r' AND relowner=o AND relname IN('consumo_asercion','estado_cuenta'))<>2
 OR to_regprocedure('vec_identidad_sesiones_v1.huella_control_sesion_v1(text,numeric,text,text,timestamptz,timestamptz,text)') IS NULL
 OR to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)') IS NULL
 OR NOT has_schema_privilege(o,'vec_autorizacion','USAGE')
 OR NOT has_table_privilege(o,'vec_autorizacion.sesion_autenticacion_v1','SELECT')
 OR NOT has_table_privilege(o,'vec_autorizacion.control_sesion_v1','SELECT')
 THEN RAISE EXCEPTION 'precondiciones historicas de Identidad no satisfechas' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE FUNCTION vec_identidad_sesiones_v1.leer_autenticacion_original_v1(
    p_autenticacion_ref text, p_sesion_ref text, p_autenticacion_sha256 text
)
RETURNS TABLE(
    autenticacion_ref text,
    autenticacion_huella_sha256 text,
    asercion_ref text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision text,
    control_sesion_huella_sha256 text,
    cuenta_ref text,
    cuenta_ordinaria_ref text,
    cuenta_privilegiada boolean,
    superficie text,
    metodo_observado text,
    garantia_observada text,
    politica_garantia_ref text,
    politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz,
    sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz,
    sesion_revalidada_en timestamptz
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog
SET row_security=on
SET statement_timeout='5s'
SET lock_timeout='2s'
AS $historia$
DECLARE

    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;

    sesion record; estado record; instante timestamptz;
    cuentas integer:=0; esperadas integer;
    inicio timestamptz:=clock_timestamp();
BEGIN
    IF current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'on'
       OR current_setting('TimeZone')<>'UTC' THEN
        RAISE EXCEPTION 'lector historico requiere SERIALIZABLE READ ONLY UTC' USING ERRCODE='25000';
    END IF;

    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_identidad_sesiones_v1_lector_historico';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_identidad_sesiones_v1';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)')];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 1 OR array_position(funciones,NULL) IS NOT NULL
       OR login.rolcanlogin IS NOT TRUE
       OR login.rolsuper OR NOT login.rolinherit OR login.rolcreaterole OR login.rolcreatedb
       OR login.rolreplication OR login.rolbypassrls OR login.rolconfig IS NOT NULL
       OR grupo.rolcanlogin OR grupo.rolsuper OR grupo.rolinherit
       OR grupo.rolcreaterole OR grupo.rolcreatedb OR grupo.rolreplication
       OR grupo.rolbypassrls OR grupo.rolconfig IS NOT NULL
       OR current_setting('role') <> 'none' OR membresias <> 1
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>login_oid
              AND pg_catalog.pg_has_role(login_oid,r.oid,'MEMBER')
              AND r.oid<>runtime_oid
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>runtime_oid
              AND pg_catalog.pg_has_role(runtime_oid,r.oid,'MEMBER')
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = runtime_oid
              AND admin_option IS FALSE AND inherit_option IS TRUE AND set_option IS FALSE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting s
            WHERE s.setrole IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl d
           LEFT JOIN LATERAL pg_catalog.aclexplode(
             coalesce(d.defaclacl,'{}'::aclitem[])
           ) a ON true
            WHERE d.defaclrole IN (login_oid,runtime_oid)
               OR a.grantee IN (login_oid,runtime_oid)
               OR a.grantor IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy p
            WHERE login_oid=ANY(p.polroles) OR runtime_oid=ANY(p.polroles)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=login_oid
       )
       OR NOT COALESCE((
           SELECT count(*)=3 AND bool_and(
             d.deptype='a' AND d.objsubid=0 AND (
               (d.classid='pg_catalog.pg_database'::regclass AND d.objid=base_oid) OR
               (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=esquema_oid) OR
               (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(funciones))
             ))
             FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
             FROM pg_catalog.pg_database b
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
             ) a
            WHERE b.oid=base_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
             ) a
            WHERE n.oid=esquema_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND count(DISTINCT p.oid)=1
                  AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))
            ) a
            WHERE p.oid=ANY(funciones) AND a.grantee=runtime_oid
       ),false)
       OR (pg_catalog.has_database_privilege(login_oid,base_oid,'CONNECT')
       AND NOT pg_catalog.has_database_privilege(login_oid,base_oid,'CREATE')
       AND NOT pg_catalog.has_database_privilege(login_oid,base_oid,'TEMPORARY')
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_namespace n
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND ((n.oid <> esquema_oid
                  AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE'))
                 OR pg_catalog.has_schema_privilege(login_oid,n.oid,'CREATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind IN ('r','p','v','m','f') AND (
              pg_catalog.has_table_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'INSERT') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'DELETE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'TRUNCATE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'REFERENCES') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'TRIGGER') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'MAINTAIN') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'INSERT') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'REFERENCES'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind='S' AND (
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'USAGE') OR
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'UPDATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_proc p
         JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
            AND p.oid <> ALL(funciones)
            AND pg_catalog.has_function_privilege(login_oid,p.oid,'EXECUTE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_type t
         JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND t.typtype IN ('c','d','e','m','r')
            AND pg_catalog.has_type_privilege(login_oid,t.oid,'USAGE')
            -- PostgreSQL atribuye USAGE implícito al tipo fila de una tabla.
            -- Sin ACL propia ni acceso a su esquema, no concede acceso a datos.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
              AND NOT pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
              AND EXISTS (
                SELECT 1 FROM pg_catalog.pg_class relacion
                 WHERE relacion.oid=t.typrelid
                   AND relacion.relkind IN ('r','p','v','m','f')
              )
            )
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_largeobject_metadata l
          WHERE pg_catalog.has_largeobject_privilege(login_oid,l.oid,'SELECT')
             OR pg_catalog.has_largeobject_privilege(login_oid,l.oid,'UPDATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_data_wrapper f
          WHERE pg_catalog.has_foreign_data_wrapper_privilege(login_oid,f.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_server s
          WHERE pg_catalog.has_server_privilege(login_oid,s.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_language l
          WHERE l.oid >= 16384
            AND pg_catalog.has_language_privilege(login_oid,l.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_tablespace t
          WHERE t.oid >= 16384
            AND pg_catalog.has_tablespace_privilege(login_oid,t.oid,'CREATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_parameter_acl a
          WHERE pg_catalog.has_parameter_privilege(login_oid,a.parname,'SET')
             OR pg_catalog.has_parameter_privilege(login_oid,a.parname,'ALTER SYSTEM')
       )) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN lector historico no acreditado';
    END IF;

    IF vec_identidad_sesiones_v1.referencia_valida(p_autenticacion_ref,'aut_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(p_sesion_ref,'ses_') IS NOT TRUE
       OR p_autenticacion_sha256 IS NULL OR p_autenticacion_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autenticacion_sha256=repeat('0',64) THEN
        RAISE EXCEPTION 'selector historico invalido' USING ERRCODE='22023';
    END IF;
    PERFORM pg_advisory_xact_lock_shared(hashtextextended(
      'vec_identidad_sesiones_v1:migracion:lectura_historica:v1',0));
    -- Selección explícita: nunca leer ni devolver columnas HMAC de consumo.
    -- La revisión es la fijada por el consumo original; no punteros actuales.
    SELECT base.autenticacion_ref,
           base.autenticacion_huella_sha256,
           base.asercion_ref,
           base.sesion_ref,
           control.control_sesion_ref,
           control.revision AS control_sesion_revision,
           control.estado AS control_sesion_estado,
           control.huella_sha256 AS control_sesion_huella_sha256,
           base.cuenta_ref,
           base.cuenta_ordinaria_ref,
           base.cuenta_privilegiada,
           base.superficie,
           base.metodo_observado,
           base.garantia_observada,
           base.politica_garantia_ref,
           base.politica_garantia_huella_sha256,
           base.autenticacion_verificada_en,
           base.sesion_emitida_en,
           control.sesion_valida_hasta,
           control.sesion_revalidada_en,
           consumo.cuenta_revision,
           consumo.cuenta_ordinaria_revision,
           consumo.operacion_ref, consumo.consumida_en
      INTO STRICT sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_identidad_sesiones_v1.consumo_asercion AS consumo
        ON consumo.sesion_ref = base.sesion_ref
       AND consumo.autenticacion_ref = base.autenticacion_ref
       AND consumo.autenticacion_huella_sha256 =
           base.autenticacion_huella_sha256
       AND consumo.asercion_ref = base.asercion_ref
       AND consumo.cuenta_ref = base.cuenta_ref
       AND consumo.cuenta_ordinaria_ref = base.cuenta_ordinaria_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = consumo.sesion_ref
       AND control.control_sesion_ref = consumo.control_sesion_ref
       AND control.revision = consumo.control_sesion_revision
     WHERE base.autenticacion_ref = p_autenticacion_ref
       AND base.sesion_ref = p_sesion_ref
       AND consumo.control_sesion_ref = control.control_sesion_ref
       AND consumo.control_sesion_revision = control.revision
       AND consumo.autenticacion_huella_sha256 = p_autenticacion_sha256;


    IF sesion.control_sesion_estado <> 'activa'
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.asercion_ref, 'ase_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.control_sesion_ref, 'cse_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.cuenta_ordinaria_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.politica_garantia_ref, 'pga_'
       ) IS NOT TRUE
       OR sesion.control_sesion_revision NOT BETWEEN
           1 AND 18446744073709551615
       OR sesion.autenticacion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.autenticacion_huella_sha256 = repeat('0', 64)
       OR sesion.control_sesion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.control_sesion_huella_sha256 = repeat('0', 64)
       OR sesion.politica_garantia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.politica_garantia_huella_sha256 = repeat('0', 64)
       OR sesion.superficie NOT IN (
           'externa_personal', 'interna_corporativa',
           'administracion_privilegiada'
       )
       OR sesion.metodo_observado NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
       )
       OR sesion.garantia_observada NOT IN (
           'bajo', 'sustancial', 'alto'
       )
       OR (sesion.superficie = 'externa_personal'
           AND sesion.garantia_observada = 'bajo')
       OR (sesion.superficie IN (
               'interna_corporativa', 'administracion_privilegiada'
           ) AND sesion.garantia_observada <> 'alto')
       OR sesion.autenticacion_verificada_en > sesion.sesion_emitida_en
       OR sesion.sesion_revalidada_en < sesion.autenticacion_verificada_en
       OR sesion.sesion_revalidada_en < sesion.sesion_emitida_en
       OR sesion.sesion_valida_hasta <= sesion.sesion_revalidada_en
       OR (sesion.cuenta_privilegiada AND (
           sesion.superficie <> 'administracion_privilegiada'
           OR sesion.cuenta_ref = sesion.cuenta_ordinaria_ref
       ))
       OR (NOT sesion.cuenta_privilegiada AND (
           sesion.superficie = 'administracion_privilegiada'
           OR sesion.cuenta_ref <> sesion.cuenta_ordinaria_ref
       )) THEN
        RAISE EXCEPTION 'autenticacion historica incoherente' USING ERRCODE='22023';
    END IF;


    IF sesion.autenticacion_ref IS DISTINCT FROM p_autenticacion_ref
       OR sesion.sesion_ref IS DISTINCT FROM p_sesion_ref
       OR sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_autenticacion_sha256
       OR sesion.consumida_en IS DISTINCT FROM sesion.sesion_revalidada_en
       OR sesion.control_sesion_huella_sha256 IS DISTINCT FROM
         vec_identidad_sesiones_v1.huella_control_sesion_v1(
           sesion.control_sesion_ref,sesion.control_sesion_revision,sesion.sesion_ref,
           sesion.control_sesion_estado,sesion.sesion_revalidada_en,sesion.sesion_valida_hasta,
           sesion.operacion_ref)
       OR sesion.sesion_valida_hasta > sesion.sesion_emitida_en + interval '5 minutes'
       OR sesion.sesion_revalidada_en >= sesion.autenticacion_verificada_en + (CASE sesion.superficie
           WHEN 'externa_personal' THEN interval '12 hours'
           WHEN 'interna_corporativa' THEN interval '15 minutes'
           WHEN 'administracion_privilegiada' THEN interval '5 minutes'
           ELSE interval '0 seconds' END) THEN
        RAISE EXCEPTION 'origen historico divergente' USING ERRCODE='22023';
    END IF;
    FOREACH instante IN ARRAY ARRAY[sesion.autenticacion_verificada_en,sesion.sesion_emitida_en,
       sesion.sesion_valida_hasta,sesion.sesion_revalidada_en,sesion.consumida_en] LOOP
        IF instante IS NULL OR NOT isfinite(instante)
           OR instante < timestamptz '0001-01-01 00:00:00+00'
           OR instante >= timestamptz '10000-01-01 00:00:00+00' THEN
            RAISE EXCEPTION 'fecha historica invalida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    esperadas:=CASE WHEN sesion.cuenta_privilegiada THEN 2 ELSE 1 END;
    FOR estado IN
      SELECT e.cuenta_ref,e.revision,e.estado,e.registrada_en
      FROM vec_identidad_sesiones_v1.estado_cuenta e
      WHERE (e.cuenta_ref=sesion.cuenta_ref AND e.revision=sesion.cuenta_revision)
         OR (e.cuenta_ref=sesion.cuenta_ordinaria_ref AND e.revision=sesion.cuenta_ordinaria_revision)
    LOOP
        cuentas:=cuentas+1;
        IF estado.estado IS DISTINCT FROM 'activa'
           OR NOT isfinite(estado.registrada_en)
           OR estado.registrada_en > sesion.consumida_en
           OR (estado.cuenta_ref=sesion.cuenta_ref AND estado.revision<>sesion.cuenta_revision)
           OR (estado.cuenta_ref=sesion.cuenta_ordinaria_ref AND estado.revision<>sesion.cuenta_ordinaria_revision) THEN
            RAISE EXCEPTION 'cuenta historica divergente' USING ERRCODE='22023';
        END IF;
    END LOOP;
    IF cuentas<>esperadas THEN
        RAISE EXCEPTION 'cuenta historica ausente' USING ERRCODE='P0002';
    END IF;
    -- El reloj actual sólo limita la ejecución. No exige permiso actual.
    IF clock_timestamp()>inicio+interval '5 seconds' THEN
        RAISE EXCEPTION 'lectura historica fuera de ventana' USING ERRCODE='57014';
    END IF;
    RETURN QUERY SELECT
        sesion.autenticacion_ref,
        sesion.autenticacion_huella_sha256,
        sesion.asercion_ref,
        sesion.sesion_ref,
        sesion.control_sesion_ref,
        sesion.control_sesion_revision::text,
        sesion.control_sesion_huella_sha256,
        sesion.cuenta_ref,
        sesion.cuenta_ordinaria_ref,
        sesion.cuenta_privilegiada,
        sesion.superficie,
        sesion.metodo_observado,
        sesion.garantia_observada,
        sesion.politica_garantia_ref,
        sesion.politica_garantia_huella_sha256,
        sesion.autenticacion_verificada_en,
        sesion.sesion_emitida_en,
        sesion.sesion_valida_hasta,
        sesion.sesion_revalidada_en;

EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'autenticacion historica no disponible' USING ERRCODE='P0002';
    WHEN data_exception THEN
        RAISE EXCEPTION 'autenticacion historica invalida' USING ERRCODE='22023';
END
$historia$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_identidad_sesiones_v1_lector_historico;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)
 TO vec_identidad_sesiones_v1_lector_historico;
DO $acl$
DECLARE f oid:='vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)'::regprocedure;
 o oid:='vec_identidad_sesiones_v1_propietario'::regrole;
 r oid:='vec_identidad_sesiones_v1_lector_historico'::regrole;
BEGIN
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
 AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'ACL historica de creacion inesperada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
