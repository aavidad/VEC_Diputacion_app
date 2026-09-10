\set ON_ERROR_STOP on
-- Fachada de lectura histórica. No concede permiso actual ni reconstruye Orden.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:000006:localizador:v1',0));
SET LOCAL ROLE vec_personal_propietario;
DO $pre$
DECLARE r oid:='vec_personal_localizador_solicitud_alta'::regrole;
 o oid:='vec_personal_propietario'::regrole; n text; f oid;
BEGIN
 IF current_setting('server_encoding')<>'UTF8'
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_personal' AND nspowner=o)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=o AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
  AND rolconfig IS NULL AND rolconnlimit=-1 AND rolvaliduntil IS NULL)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r)
 OR EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_personal'::regnamespace
  AND proname='localizar_solicitud_alta_ejercicio_v1')
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
  WHERE n.oid='vec_personal'::regnamespace AND a.grantee=r)
 THEN RAISE EXCEPTION 'Personal6: precondiciones incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['registro_alta_ejercicio','relacion_alta_ejercicio','ocupacion_alta_ejercicio','auditoria_alta_ejercicio','outbox_alta_ejercicio'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE relnamespace='vec_personal'::regnamespace
   AND relname=n AND relkind='r' AND NOT relispartition AND relowner=o AND relrowsecurity AND relforcerowsecurity)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.relnamespace='vec_personal'::regnamespace AND c.relname=n
    AND (a.grantee<>o OR a.grantor<>o))
  OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_attribute a ON a.attrelid=c.oid
   WHERE c.relnamespace='vec_personal'::regnamespace AND c.relname=n AND a.attnum>0
    AND (a.attisdropped OR a.attacl IS NOT NULL))
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_personal'::regnamespace
  AND c.relname=n AND c.relowner='vec_personal_propietario'::regrole AND c.relkind='r' AND NOT c.relispartition
  AND c.relrowsecurity AND c.relforcerowsecurity
  AND (SELECT count(*) FROM pg_policy p WHERE p.polrelid=c.oid)=1
  AND EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=c.oid AND p.polname='propietario'
   AND p.polcmd='*' AND p.polpermissive AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid]
   AND pg_get_expr(p.polqual,p.polrelid)='true' AND pg_get_expr(p.polwithcheck,p.polrelid)='true'))
  THEN RAISE EXCEPTION 'Personal6: fuente propietaria incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH n IN ARRAY ARRAY[
  'material_alta_ejercicio_canonico_v1(jsonb)',
  'solicitud_alta_ejercicio_canonica_v1(jsonb)',
  'contexto_alta_ejercicio_canonico_v1(jsonb)',
  'material_registro_alta_valido_v1(jsonb)',
  'fecha_registro_alta_pg_v1(text)',
  'referencia_alta_ejercicio_valida_v1(text)'] LOOP
  f:=to_regprocedure('vec_personal.'||n);
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner=o AND NOT prosecdef AND provolatile='i')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee<>o OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'Personal6: codec propietario requerido' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_personal.localizar_solicitud_alta_ejercicio_v1(
 p_organizacion_ref text, p_expediente_ref text, p_solicitud_ref text
) RETURNS TABLE (solicitud_json jsonb, resultado_ref text, recibo_ref text, relacion_ref text, ocupacion_ref text, material_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET statement_timeout='5s' SET lock_timeout='2s'
AS $localizador$
DECLARE
 login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
 membresias integer; funciones oid[]; login record; grupo record;
 r vec_personal.registro_alta_ejercicio%ROWTYPE;
 rel vec_personal.relacion_alta_ejercicio%ROWTYPE; ocu vec_personal.ocupacion_alta_ejercicio%ROWTYPE;
 aud vec_personal.auditoria_alta_ejercicio%ROWTYPE; ob vec_personal.outbox_alta_ejercicio%ROWTYPE;
 w jsonb; s jsonb; x jsonb; recibo jsonb; evento jsonb;
 mc bytea; sc bytea; rc bytea; hm text; hs text; hr text;
 inicio timestamptz:=clock_timestamp(); codigo text; cantidad bigint; selector text; fuente text;
 conflicto boolean:=false;
BEGIN
     IF current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'on'
       OR current_setting('TimeZone')<>'UTC' THEN
        RAISE EXCEPTION 'lector historico requiere SERIALIZABLE READ ONLY UTC' USING ERRCODE='25000';
    END IF;

    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    -- Rechazo secuencial, previo al predicado con subconsultas de privilegios.
    -- No depender del orden de evaluacion SQL de OR/InitPlans para negar DBA.
    IF login.oid IS NULL OR login.rolcanlogin IS NOT TRUE
       OR login.rolsuper IS DISTINCT FROM false OR login.rolinherit IS DISTINCT FROM true
       OR login.rolcreaterole IS DISTINCT FROM false OR login.rolcreatedb IS DISTINCT FROM false
       OR login.rolreplication IS DISTINCT FROM false OR login.rolbypassrls IS DISTINCT FROM false
       OR login.rolconfig IS NOT NULL OR current_setting('role') <> 'none' THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN lector historico no acreditado';
    END IF;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_personal_localizador_solicitud_alta';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_personal';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[pg_catalog.to_regprocedure('vec_personal.localizar_solicitud_alta_ejercicio_v1(text,text,text)')];
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
           SELECT 1 FROM pg_catalog.pg_roles localizador_catalogo_r
            WHERE localizador_catalogo_r.oid<>login_oid
              AND pg_catalog.pg_has_role(login_oid,localizador_catalogo_r.oid,'MEMBER')
              AND localizador_catalogo_r.oid<>runtime_oid
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles localizador_catalogo_r
            WHERE localizador_catalogo_r.oid<>runtime_oid
              AND pg_catalog.pg_has_role(runtime_oid,localizador_catalogo_r.oid,'MEMBER')
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = runtime_oid
              AND admin_option IS FALSE AND inherit_option IS TRUE AND set_option IS FALSE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting localizador_catalogo_s
            WHERE localizador_catalogo_s.setrole IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl d
           LEFT JOIN LATERAL pg_catalog.aclexplode(
             coalesce(d.defaclacl,'{}'::aclitem[])
           ) localizador_catalogo_a ON true
            WHERE d.defaclrole IN (login_oid,runtime_oid)
               OR localizador_catalogo_a.grantee IN (login_oid,runtime_oid)
               OR localizador_catalogo_a.grantor IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy localizador_catalogo_p
            WHERE login_oid=ANY(localizador_catalogo_p.polroles) OR runtime_oid=ANY(localizador_catalogo_p.polroles)
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
           SELECT count(*)=1 AND bool_and(localizador_catalogo_a.privilege_type='CONNECT' AND NOT localizador_catalogo_a.is_grantable AND localizador_catalogo_a.grantor=b.datdba)
             FROM pg_catalog.pg_database b
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
             ) localizador_catalogo_a
            WHERE b.oid=base_oid AND localizador_catalogo_a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(localizador_catalogo_a.privilege_type='USAGE' AND NOT localizador_catalogo_a.is_grantable AND localizador_catalogo_a.grantor=n.nspowner)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
             ) localizador_catalogo_a
            WHERE n.oid=esquema_oid AND localizador_catalogo_a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND count(DISTINCT localizador_catalogo_p.oid)=1
                  AND bool_and(localizador_catalogo_a.privilege_type='EXECUTE' AND NOT localizador_catalogo_a.is_grantable AND localizador_catalogo_a.grantor=localizador_catalogo_p.proowner)
             FROM pg_catalog.pg_proc localizador_catalogo_p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(localizador_catalogo_p.proacl,pg_catalog.acldefault('f',localizador_catalogo_p.proowner))
            ) localizador_catalogo_a
            WHERE localizador_catalogo_p.oid=ANY(funciones) AND localizador_catalogo_a.grantee=runtime_oid
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
         SELECT 1 FROM pg_catalog.pg_proc localizador_catalogo_p
         JOIN pg_catalog.pg_namespace n ON n.oid=localizador_catalogo_p.pronamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
            AND localizador_catalogo_p.oid <> ALL(funciones)
            AND pg_catalog.has_function_privilege(login_oid,localizador_catalogo_p.oid,'EXECUTE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_type t
         JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND t.typtype IN ('c','d','e','m','r')
            AND pg_catalog.has_type_privilege(login_oid,t.oid,'USAGE')
            -- PostgreSQL atribuye USAGE implícito al tipo fila de una tabla.
            -- Sin ACL propia, el tipo fila no concede acceso a datos;
            -- todas las capacidades de tabla/columna se rechazan arriba.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
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
         SELECT 1 FROM pg_catalog.pg_foreign_data_wrapper localizador_catalogo_f
          WHERE pg_catalog.has_foreign_data_wrapper_privilege(login_oid,localizador_catalogo_f.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_server localizador_catalogo_s
          WHERE pg_catalog.has_server_privilege(login_oid,localizador_catalogo_s.oid,'USAGE')
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
         SELECT 1 FROM pg_catalog.pg_parameter_acl localizador_catalogo_a
          WHERE pg_catalog.has_parameter_privilege(login_oid,localizador_catalogo_a.parname,'SET')
             OR pg_catalog.has_parameter_privilege(login_oid,localizador_catalogo_a.parname,'ALTER SYSTEM')
       )) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN lector historico no acreditado';
    END IF;


 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_personal:000006:localizador:v1',0));


 FOREACH selector IN ARRAY ARRAY[p_organizacion_ref,p_expediente_ref,p_solicitud_ref] LOOP
  IF selector IS NULL OR octet_length(selector) NOT BETWEEN 3 AND 160
   OR selector COLLATE "C" !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  THEN RAISE EXCEPTION 'localizador: selector invalido' USING ERRCODE='22023'; END IF;
 END LOOP;

 -- Nunca convertir una politica alterada en ausencia de negocio.
 FOREACH fuente IN ARRAY ARRAY['registro_alta_ejercicio','relacion_alta_ejercicio','ocupacion_alta_ejercicio','auditoria_alta_ejercicio','outbox_alta_ejercicio'] LOOP
  IF false
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_personal'::regnamespace
  AND c.relname=fuente AND c.relowner='vec_personal_propietario'::regrole AND c.relkind='r' AND NOT c.relispartition
  AND c.relrowsecurity AND c.relforcerowsecurity
  AND (SELECT count(*) FROM pg_policy p WHERE p.polrelid=c.oid)=1
  AND EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=c.oid AND p.polname='propietario'
   AND p.polcmd='*' AND p.polpermissive AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid]
   AND pg_get_expr(p.polqual,p.polrelid)='true' AND pg_get_expr(p.polwithcheck,p.polrelid)='true'))
  THEN RAISE EXCEPTION 'localizador: fuente propietaria alterada' USING ERRCODE='55000'; END IF;
 END LOOP;
 SELECT count(*) INTO cantidad FROM vec_personal.registro_alta_ejercicio q
 WHERE (q.organizacion_ref=p_organizacion_ref AND q.expediente_ref=p_expediente_ref AND q.solicitud_ref=p_solicitud_ref)
 OR (q.material_json->>'OrganizacionRef'=p_organizacion_ref
  AND q.material_json#>>'{Preparacion,Solicitud,expediente_ref}'=p_expediente_ref
  AND q.material_json#>>'{Preparacion,Solicitud,solicitud_ref}'=p_solicitud_ref);
 IF clock_timestamp()<inicio OR clock_timestamp()>inicio+interval '5 seconds' THEN
  RAISE EXCEPTION 'localizador: plazo excedido' USING ERRCODE='57014'; END IF;
 IF cantidad=0 THEN RETURN; END IF;
 IF cantidad<>1 THEN RAISE EXCEPTION 'Personal6: unicidad incompatible' USING ERRCODE='55000'; END IF;
 SELECT q.* INTO STRICT r FROM vec_personal.registro_alta_ejercicio q
 WHERE (q.organizacion_ref=p_organizacion_ref AND q.expediente_ref=p_expediente_ref AND q.solicitud_ref=p_solicitud_ref)
 OR (q.material_json->>'OrganizacionRef'=p_organizacion_ref
  AND q.material_json#>>'{Preparacion,Solicitud,expediente_ref}'=p_expediente_ref
  AND q.material_json#>>'{Preparacion,Solicitud,solicitud_ref}'=p_solicitud_ref);
 IF r.organizacion_ref IS DISTINCT FROM p_organizacion_ref OR r.expediente_ref IS DISTINCT FROM p_expediente_ref
 OR r.solicitud_ref IS DISTINCT FROM p_solicitud_ref
 OR octet_length(r.material_json::text)>65536 OR octet_length(r.recibo_json::text)>65536
 OR octet_length(r.material_canonico)>65536 OR octet_length(r.solicitud_canonica)>65536 OR octet_length(r.contexto_canonico)>65536
 THEN RAISE EXCEPTION 'Personal6: registro cruzado o excesivo' USING ERRCODE='55000'; END IF;
 BEGIN
    IF vec_personal.material_registro_alta_valido_v1(r.material_json) IS NOT TRUE THEN
        RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000';
    END IF;
    s:=r.material_json#>'{Preparacion,Solicitud}'; x:=r.material_json#>'{Preparacion,Vinculo}';
    w:=jsonb_build_object('Esquema','vec.personal.alta-ejercicio.material.v1','Material',r.material_json);
    mc:=vec_personal.material_alta_ejercicio_canonico_v1(w);
    sc:=vec_personal.solicitud_alta_ejercicio_canonica_v1(s);
    rc:=vec_personal.contexto_alta_ejercicio_canonico_v1(w);
    hm:=encode(sha256(mc),'hex'); hs:=encode(sha256(sc),'hex'); hr:=encode(sha256(rc),'hex');
    SELECT q.* INTO rel FROM vec_personal.relacion_alta_ejercicio q WHERE q.relacion_ref=r.relacion_ref;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    SELECT q.* INTO ocu FROM vec_personal.ocupacion_alta_ejercicio q WHERE q.ocupacion_ref=r.ocupacion_ref;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    SELECT q.* INTO aud FROM vec_personal.auditoria_alta_ejercicio q WHERE q.auditoria_ref=r.auditoria_ref;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    SELECT q.* INTO ob FROM vec_personal.outbox_alta_ejercicio q WHERE q.outbox_ref=r.outbox_ref;
    IF NOT FOUND THEN RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000'; END IF;
    IF octet_length(ocu.fuente_rpt::text)>65536 OR octet_length(ob.payload::text)>65536 THEN
        RAISE EXCEPTION 'Personal6: historia excesiva' USING ERRCODE='55000'; END IF;
    recibo:=jsonb_build_object(
        'resultado',jsonb_build_object('esquema','vec.contratacion-temporal.personal-rpt.alta.v1','contrato_version',1,
            'resultado_ref',r.resultado_ref,'recibo_ref',r.recibo_ref,'solicitud_ref',r.solicitud_ref,
            'correlacion_ref',s->>'correlacion_ref','idempotencia_ref',r.idempotencia_ref,
            'huella_solicitud_sha256',hs,'estado','confirmada','relacion_ref',r.relacion_ref,'ocupacion_ref',r.ocupacion_ref),
        'material_sha256',hm,'registrado_en',to_char(r.registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'decision_original_ref',r.decision_original_ref,'auditoria_ref',r.auditoria_ref,'outbox_ref',r.outbox_ref,
        'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false,
        'replay',false,'decision_consumida_ref',r.decision_original_ref);
    evento:=jsonb_build_object('esquema','vec.personal.alta-ejercicio.registrada.v1',
        'resultado_ref',r.resultado_ref,'recibo_ref',r.recibo_ref,'relacion_ref',r.relacion_ref,
        'ocupacion_ref',r.ocupacion_ref,'material_sha256',hm,
        'ejercicio_sintetico',true,'firma_oficial',false,'eficacia_administrativa',false);
    IF mc IS NULL OR sc IS NULL OR rc IS NULL OR octet_length(mc) NOT BETWEEN 1 AND 65536
       OR r.material_canonico IS DISTINCT FROM mc OR r.solicitud_canonica IS DISTINCT FROM sc
       OR r.contexto_canonico IS DISTINCT FROM rc OR r.material_sha256 IS DISTINCT FROM hm
       OR r.solicitud_sha256 IS DISTINCT FROM hs OR r.contexto_sha256 IS DISTINCT FROM hr
       OR r.organizacion_ref IS DISTINCT FROM r.material_json->>'OrganizacionRef'
       OR r.actor_ref IS DISTINCT FROM r.material_json->>'ActorRef'
       OR r.perfil_ref IS DISTINCT FROM r.material_json->>'PerfilRef'
       OR r.solicitud_ref IS DISTINCT FROM s->>'solicitud_ref' OR r.expediente_ref IS DISTINCT FROM s->>'expediente_ref'
       OR r.idempotencia_ref IS DISTINCT FROM s->>'idempotencia_ref' OR r.recibo_json IS DISTINCT FROM recibo
       OR r.ejercicio_sintetico IS DISTINCT FROM true OR r.firma_oficial IS DISTINCT FROM false
       OR r.eficacia_administrativa IS DISTINCT FROM false OR NOT isfinite(r.registrado_en) OR r.registrado_en>inicio
       OR rel.organizacion_ref IS DISTINCT FROM r.organizacion_ref
       OR rel.persona_sintetica_ref IS DISTINCT FROM x->>'PersonaSinteticaRef'
       OR rel.desde IS DISTINCT FROM vec_personal.fecha_registro_alta_pg_v1(x->>'Desde')
       OR rel.hasta IS DISTINCT FROM vec_personal.fecha_registro_alta_pg_v1(x->>'Hasta')
       OR rel.registrada_en IS DISTINCT FROM r.registrado_en
       OR rel.ejercicio_sintetico IS DISTINCT FROM true OR rel.firma_oficial IS DISTINCT FROM false
       OR rel.eficacia_administrativa IS DISTINCT FROM false
       OR ocu.relacion_ref IS DISTINCT FROM r.relacion_ref OR ocu.centro_ref IS DISTINCT FROM x->>'CentroRef'
       OR ocu.puesto_ref IS DISTINCT FROM x->>'PuestoRef' OR ocu.plaza_ref IS DISTINCT FROM x->>'PlazaRef'
       OR ocu.fuente_rpt IS DISTINCT FROM x->'FuenteRPT'
       OR aud.resultado_ref IS DISTINCT FROM r.resultado_ref OR aud.decision_ref IS DISTINCT FROM r.decision_original_ref
       OR aud.material_sha256 IS DISTINCT FROM hm OR aud.contexto_sha256 IS DISTINCT FROM hr
       OR aud.actor_ref IS DISTINCT FROM r.actor_ref OR aud.perfil_ref IS DISTINCT FROM r.perfil_ref
       OR aud.recuperacion IS DISTINCT FROM false OR aud.registrada_en IS DISTINCT FROM r.registrado_en
       OR NOT isfinite(aud.consumida_en) OR aud.consumida_en>aud.registrada_en
       OR aud.consumo_huella_sha256 IS NULL OR aud.consumo_huella_sha256 !~ '^[a-f0-9]{64}$'
       OR vec_personal.referencia_alta_ejercicio_valida_v1(aud.auditoria_v3_ref) IS NOT TRUE
       OR ob.resultado_ref IS DISTINCT FROM r.resultado_ref OR ob.payload IS DISTINCT FROM evento
       OR ob.tipo_evento IS DISTINCT FROM 'personal.alta_ejercicio.registrada.v1'
       OR ob.payload_sha256 IS DISTINCT FROM encode(sha256(convert_to(evento::text,'UTF8')),'hex')
       OR ob.registrada_en IS DISTINCT FROM r.registrado_en THEN
        RAISE EXCEPTION 'registro Personal inconsistente' USING ERRCODE='55000';
    END IF;

 FOREACH selector IN ARRAY ARRAY[r.resultado_ref,r.recibo_ref,r.relacion_ref,r.ocupacion_ref,r.decision_original_ref,r.auditoria_ref,r.outbox_ref] LOOP
  IF vec_personal.referencia_alta_ejercicio_valida_v1(selector) IS NOT TRUE THEN
   RAISE EXCEPTION 'Personal6: referencia original incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 EXCEPTION WHEN SQLSTATE '22023' OR data_exception THEN
  RAISE EXCEPTION 'localizador: canon propietario incompatible' USING ERRCODE='55000';
 END;
 solicitud_json:=s; resultado_ref:=r.resultado_ref; recibo_ref:=r.recibo_ref; relacion_ref:=r.relacion_ref;
 ocupacion_ref:=r.ocupacion_ref; material_sha256:=r.material_sha256;
 IF clock_timestamp()<inicio OR clock_timestamp()>inicio+interval '5 seconds' THEN
  RAISE EXCEPTION 'localizador: plazo excedido' USING ERRCODE='57014'; END IF;
 RETURN NEXT;
 RETURN;
EXCEPTION WHEN OTHERS THEN
 codigo:=SQLSTATE;
 IF codigo='P1102' AND NOT conflicto THEN codigo:='55000'; END IF;
 IF codigo NOT IN('P1102','42501','25000','22023','40001','40P01','55P03','57014') THEN codigo:='55000'; END IF;
 RAISE EXCEPTION 'localizador: original no disponible' USING ERRCODE=codigo;
END $localizador$;
REVOKE ALL ON FUNCTION vec_personal.localizar_solicitud_alta_ejercicio_v1(text,text,text) FROM PUBLIC;
-- Crear objetos solo con defaults owner-only; no limpiar permisos ajenos silenciosamente.
DO $acl$
DECLARE f oid:='vec_personal.localizar_solicitud_alta_ejercicio_v1(text,text,text)'::regprocedure;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'Personal6: defaults incompatibles' USING ERRCODE='55000'; END IF;
END $acl$;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_localizador_solicitud_alta;
GRANT EXECUTE ON FUNCTION vec_personal.localizar_solicitud_alta_ejercicio_v1(text,text,text)
 TO vec_personal_localizador_solicitud_alta;
COMMIT;
