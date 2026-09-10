\set ON_ERROR_STOP on
-- Fachada de lectura histórica. No concede permiso actual ni reconstruye Orden.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000081:localizador:v1',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $pre$
DECLARE r oid:='vec_contratacion_temporal_localizador_incorporacion'::regrole;
 o oid:='vec_contratacion_temporal_propietario'::regrole; n text; f oid;
BEGIN
 IF current_setting('server_encoding')<>'UTF8'
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=o)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=o AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
  AND rolconfig IS NULL AND rolconnlimit=-1 AND rolvaliduntil IS NULL)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r)
 OR EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace
  AND proname='localizar_incorporacion_original_v2')
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
  WHERE n.oid='vec_contratacion_temporal'::regnamespace AND a.grantee=r)
 THEN RAISE EXCEPTION 'CT81: precondiciones incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['incorporacion_registro_v2','incorporacion_auditoria_v2','incorporacion_outbox_v2','seguimiento_raiz_v2','seguimiento_definicion_v2','seguimiento_estado_v2'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE relnamespace='vec_contratacion_temporal'::regnamespace
   AND relname=n AND relkind='r' AND NOT relispartition AND relowner=o AND relrowsecurity AND relforcerowsecurity)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.relnamespace='vec_contratacion_temporal'::regnamespace AND c.relname=n
    AND (a.grantee<>o OR a.grantor<>o))
  OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_attribute a ON a.attrelid=c.oid
   WHERE c.relnamespace='vec_contratacion_temporal'::regnamespace AND c.relname=n AND a.attnum>0
    AND (a.attisdropped OR a.attacl IS NOT NULL))
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_contratacion_temporal'::regnamespace
  AND c.relname=n AND c.relowner='vec_contratacion_temporal_propietario'::regrole AND c.relkind='r' AND NOT c.relispartition
  AND c.relrowsecurity AND c.relforcerowsecurity
  AND (SELECT count(*) FROM pg_policy p WHERE p.polrelid=c.oid)=1
  AND EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=c.oid AND p.polname='propietario'
   AND p.polcmd='*' AND p.polpermissive AND p.polroles=ARRAY['vec_contratacion_temporal_propietario'::regrole::oid]
   AND pg_get_expr(p.polqual,p.polrelid)='true' AND pg_get_expr(p.polwithcheck,p.polrelid)='true'))
  THEN RAISE EXCEPTION 'CT81: fuente propietaria incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH n IN ARRAY ARRAY[  'material_incorporacion_ejercicio_canonico_v2(jsonb)',
  'intencion_registro_incorporacion_canonica_v2(jsonb)',
  'estado_seguimiento_canonico_v1(jsonb,jsonb)',
  'seguimiento73_definicion(jsonb)','seguimiento73_nodo(jsonb,text)',
  'seguimiento73_forma(jsonb,text[],text[])','seguimiento73_micro(jsonb,boolean)',
  'aplicar_incorporacion_seguimiento_v2(jsonb,jsonb,jsonb,text,text,text)',
  'incorporacion75_instante(timestamptz)',
  'incorporacion75_evidencia(jsonb,jsonb,bytea[],numeric,numeric,timestamptz)',
  'incorporacion75_autoridad(jsonb,bytea[],numeric,numeric,timestamptz)',
  'incorporacion75_recibo(jsonb,text,jsonb,jsonb,jsonb,jsonb,jsonb,text,text)'] LOOP
  f:=to_regprocedure('vec_contratacion_temporal.'||n);
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner=o AND NOT prosecdef AND provolatile='i')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee<>o OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'CT81: codec propietario requerido' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.localizar_incorporacion_original_v2(
 p_organizacion_ref text, p_expediente_ref text, p_solicitud_personal_ref text
) RETURNS TABLE (recibo_ref text, material_sha256 text, intencion_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET statement_timeout='5s' SET lock_timeout='2s'
AS $localizador$
DECLARE
 login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
 membresias integer; funciones oid[]; login record; grupo record;
 r vec_contratacion_temporal.incorporacion_registro_v2%ROWTYPE;
 z vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;
 f vec_contratacion_temporal.seguimiento_definicion_v2%ROWTYPE;
 a vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 p vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 aud vec_contratacion_temporal.incorporacion_auditoria_v2%ROWTYPE;
 ob vec_contratacion_temporal.incorporacion_outbox_v2%ROWTYPE;
 pub jsonb; def jsonb; raiz jsonb; aplicado jsonb; recibo jsonb; consumos jsonb; resultado jsonb;
 s jsonb; prep jsonb; k text; he text; mc bytea; ic bytea; ca bytea; cp bytea;
 inicio timestamptz:=clock_timestamp(); ct_en timestamptz; leida_en timestamptz;
 codigo text;
 cantidad bigint; selector text; fuente text; conflicto boolean:=false;
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
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_contratacion_temporal_localizador_incorporacion';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_contratacion_temporal';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[pg_catalog.to_regprocedure('vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text)')];
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
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contratacion_temporal:000081:localizador:v1',0));


 FOREACH selector IN ARRAY ARRAY[p_organizacion_ref,p_expediente_ref,p_solicitud_personal_ref] LOOP
  IF selector IS NULL OR octet_length(selector) NOT BETWEEN 3 AND 160
   OR selector COLLATE "C" !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  THEN RAISE EXCEPTION 'localizador: selector invalido' USING ERRCODE='22023'; END IF;
 END LOOP;

 -- Nunca convertir una politica alterada en ausencia de negocio.
 FOREACH fuente IN ARRAY ARRAY['incorporacion_registro_v2','incorporacion_auditoria_v2','incorporacion_outbox_v2','seguimiento_raiz_v2','seguimiento_definicion_v2','seguimiento_estado_v2'] LOOP
  IF false
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_contratacion_temporal'::regnamespace
  AND c.relname=fuente AND c.relowner='vec_contratacion_temporal_propietario'::regrole AND c.relkind='r' AND NOT c.relispartition
  AND c.relrowsecurity AND c.relforcerowsecurity
  AND (SELECT count(*) FROM pg_policy p WHERE p.polrelid=c.oid)=1
  AND EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=c.oid AND p.polname='propietario'
   AND p.polcmd='*' AND p.polpermissive AND p.polroles=ARRAY['vec_contratacion_temporal_propietario'::regrole::oid]
   AND pg_get_expr(p.polqual,p.polrelid)='true' AND pg_get_expr(p.polwithcheck,p.polrelid)='true'))
  THEN RAISE EXCEPTION 'localizador: fuente propietaria alterada' USING ERRCODE='55000'; END IF;
 END LOOP;
 SELECT count(*) INTO cantidad FROM vec_contratacion_temporal.incorporacion_registro_v2 q
 WHERE (q.organizacion_ref=p_organizacion_ref AND q.expediente_ref=p_expediente_ref AND q.solicitud_ref=p_solicitud_personal_ref)
 OR (q.material_json#>>'{Preparacion,OrganizacionRef}'=p_organizacion_ref
  AND q.material_json#>>'{Confirmacion,SolicitudPersonal,expediente_ref}'=p_expediente_ref
  AND q.material_json#>>'{Confirmacion,SolicitudPersonal,solicitud_ref}'=p_solicitud_personal_ref);
 IF clock_timestamp()<inicio OR clock_timestamp()>inicio+interval '5 seconds' THEN
  RAISE EXCEPTION 'localizador: plazo excedido' USING ERRCODE='57014'; END IF;
 IF cantidad=0 THEN RETURN; END IF;
 IF cantidad>1 THEN conflicto:=true; RAISE EXCEPTION 'CT81: localizacion ambigua' USING ERRCODE='P1102'; END IF;
 SELECT q.* INTO STRICT r FROM vec_contratacion_temporal.incorporacion_registro_v2 q
 WHERE (q.organizacion_ref=p_organizacion_ref AND q.expediente_ref=p_expediente_ref AND q.solicitud_ref=p_solicitud_personal_ref)
 OR (q.material_json#>>'{Preparacion,OrganizacionRef}'=p_organizacion_ref
  AND q.material_json#>>'{Confirmacion,SolicitudPersonal,expediente_ref}'=p_expediente_ref
  AND q.material_json#>>'{Confirmacion,SolicitudPersonal,solicitud_ref}'=p_solicitud_personal_ref);
 IF r.organizacion_ref IS DISTINCT FROM p_organizacion_ref OR r.expediente_ref IS DISTINCT FROM p_expediente_ref
 OR r.solicitud_ref IS DISTINCT FROM p_solicitud_personal_ref
 OR octet_length(r.material_json::text)>16777216
 OR r.recibo_ref IS NULL OR r.recibo_ref !~ '^ref:[0-9a-f]{64}$' OR r.recibo_ref='ref:'||repeat('0',64)
 THEN RAISE EXCEPTION 'CT81: registro cruzado' USING ERRCODE='55000'; END IF;
 BEGIN
 IF octet_length(r.material_canonico) NOT BETWEEN 1 AND 1048576
 OR octet_length(r.intencion_canonica) NOT BETWEEN 1 AND 1048576
 OR octet_length(r.evidencia_orden_json::text)>16777216
 OR octet_length(r.recibo_json::text)>16777216
 OR NOT isfinite(r.registrada_en) OR r.registrada_en>inicio
 THEN RAISE EXCEPTION 'CT81: registro incompatible' USING ERRCODE='55000'; END IF;
 SELECT q.* INTO STRICT z FROM vec_contratacion_temporal.seguimiento_raiz_v2 q WHERE q.seguimiento_ref=r.seguimiento_ref;
 SELECT q.* INTO STRICT f FROM vec_contratacion_temporal.seguimiento_definicion_v2 q
  WHERE q.definicion_ref=z.definicion_ref AND q.definicion_version=z.definicion_version AND q.definicion_sha256=z.definicion_sha256;
 SELECT q.* INTO STRICT a FROM vec_contratacion_temporal.seguimiento_estado_v2 q
  WHERE q.seguimiento_ref=r.seguimiento_ref AND q.version_seguimiento=r.version_anterior AND q.estado_sha256=r.estado_anterior_sha256;
 SELECT q.* INTO STRICT p FROM vec_contratacion_temporal.seguimiento_estado_v2 q
  WHERE q.seguimiento_ref=r.seguimiento_ref AND q.version_seguimiento=r.version_resultante AND q.estado_sha256=r.estado_resultante_sha256;
 SELECT q.* INTO STRICT aud FROM vec_contratacion_temporal.incorporacion_auditoria_v2 q WHERE q.auditoria_ref=r.auditoria_ref;
 SELECT q.* INTO STRICT ob FROM vec_contratacion_temporal.incorporacion_outbox_v2 q WHERE q.outbox_ref=r.outbox_ref;

 IF octet_length(f.publicacion_json::text)>8388608 OR octet_length(f.definicion_canonica)>8388608
 OR octet_length(z.raiz_canonica)>8388608
 OR octet_length(a.estado_json::text)>8388608 OR octet_length(a.estado_canonico)>8388608
 OR octet_length(p.estado_json::text)>8388608 OR octet_length(p.estado_canonico)>8388608
 OR octet_length(ob.evento_json::text)>8388608 OR octet_length(aud.consumos_json::text)>65536
 THEN RAISE EXCEPTION 'CT81: historia excede limite operativo' USING ERRCODE='55000'; END IF;

 pub:=f.publicacion_json; def:=vec_contratacion_temporal.seguimiento73_definicion(pub);
 mc:=vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(r.material_json);
 ic:=vec_contratacion_temporal.intencion_registro_incorporacion_canonica_v2(r.material_json);
 s:=r.material_json#>'{Confirmacion,SolicitudPersonal}'; prep:=r.material_json->'Preparacion';
 IF mc IS DISTINCT FROM r.material_canonico OR encode(sha256(mc),'hex') IS DISTINCT FROM r.material_sha256
 OR ic IS DISTINCT FROM r.intencion_canonica OR encode(sha256(ic),'hex') IS DISTINCT FROM r.intencion_sha256
 OR r.organizacion_ref IS DISTINCT FROM z.organizacion_ref OR r.organizacion_ref IS DISTINCT FROM prep->>'OrganizacionRef'
 OR r.expediente_ref IS DISTINCT FROM z.expediente_ref OR r.expediente_ref IS DISTINCT FROM s->>'expediente_ref'
 OR r.solicitud_ref IS DISTINCT FROM s->>'solicitud_ref' OR r.idempotencia_ref IS DISTINCT FROM s->>'idempotencia_ref'
 OR r.version_expediente IS DISTINCT FROM (r.material_json->>'VersionActualExpediente')::numeric
 OR z.version_expediente_observada>r.version_expediente
 OR z.relacion_ref IS DISTINCT FROM r.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'
 OR r.version_anterior IS DISTINCT FROM (r.material_json#>>'{Confirmacion,VersionSeguimientoEsperada}')::numeric
 OR r.version_resultante IS DISTINCT FROM r.version_anterior+1
 OR f.definicion_canonica IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_nodo(def,'publicacion')
 OR f.definicion_sha256 IS DISTINCT FROM encode(sha256(f.definicion_canonica),'hex')
 THEN RAISE EXCEPTION 'CT81: material o fuente cruzados' USING ERRCODE='55000'; END IF;

 -- Metadatos de ambas filas vinculados a la raiz; JSON y binario se revalidan.
 IF jsonb_build_array(a.organizacion_ref,a.expediente_ref,a.relacion_ref,a.definicion_ref,a.definicion_version,a.definicion_sha256,a.raiz_sha256)
 IS DISTINCT FROM jsonb_build_array(z.organizacion_ref,z.expediente_ref,z.relacion_ref,z.definicion_ref,z.definicion_version,z.definicion_sha256,z.raiz_sha256)
 OR jsonb_build_array(p.organizacion_ref,p.expediente_ref,p.relacion_ref,p.definicion_ref,p.definicion_version,p.definicion_sha256,p.raiz_sha256)
 IS DISTINCT FROM jsonb_build_array(z.organizacion_ref,z.expediente_ref,z.relacion_ref,z.definicion_ref,z.definicion_version,z.definicion_sha256,z.raiz_sha256)
 OR a.estado_json->>'referencia' IS DISTINCT FROM r.seguimiento_ref OR p.estado_json->>'referencia' IS DISTINCT FROM r.seguimiento_ref
 OR (a.estado_json->>'version')::numeric IS DISTINCT FROM a.version_seguimiento
 OR (p.estado_json->>'version')::numeric IS DISTINCT FROM p.version_seguimiento
 OR a.estado_json->>'organizacion_ref' IS DISTINCT FROM z.organizacion_ref
 OR a.estado_json->>'expediente_ref' IS DISTINCT FROM z.expediente_ref
 OR a.estado_json->>'relacion_ref' IS DISTINCT FROM z.relacion_ref
 OR a.estado_json->'definicion' IS DISTINCT FROM jsonb_build_object('referencia',z.definicion_ref,'version',z.definicion_version,'huella_sha256',z.definicion_sha256)
 OR p.version_anterior IS DISTINCT FROM a.version_seguimiento OR p.estado_anterior_sha256 IS DISTINCT FROM a.estado_sha256
 THEN RAISE EXCEPTION 'CT81: estados cruzados' USING ERRCODE='55000'; END IF;
 ca:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,a.estado_json);
 cp:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(pub,p.estado_json);
 raiz:=jsonb_build_object('referencia',z.seguimiento_ref,'organizacion_ref',z.organizacion_ref,'expediente_ref',z.expediente_ref,'relacion_ref',z.relacion_ref,
  'definicion',a.estado_json->'definicion','estado_actual',def->'estado_inicial','periodo_previsto',a.estado_json->'periodo_previsto','creado_en',a.estado_json->'creado_en');
 IF ca IS DISTINCT FROM a.estado_canonico OR encode(sha256(ca),'hex') IS DISTINCT FROM a.estado_sha256
 OR cp IS DISTINCT FROM p.estado_canonico OR encode(sha256(cp),'hex') IS DISTINCT FROM p.estado_sha256
 OR z.raiz_canonica IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_nodo(raiz,'raiz')
 OR z.raiz_sha256 IS DISTINCT FROM encode(sha256(z.raiz_canonica),'hex')
 OR z.raiz_sha256 IS DISTINCT FROM a.estado_json->>'huella_raiz_sha256'
 THEN RAISE EXCEPTION 'CT81: canon historico incompatible' USING ERRCODE='55000'; END IF;

 consumos:=r.recibo_json->'ConsumosOriginales';
 PERFORM vec_contratacion_temporal.seguimiento73_forma(consumos,ARRAY['DecisionCTRef','HuellaExportacionCT','ConsumoCTSHA256','AuditoriaAD3CTRef','ConsumidaCTEn',
  'DecisionLecturaRef','ConsumoLecturaSHA256','AuditoriaAD3LecturaRef','AuditoriaPersonalLecturaRef','LeidaPersonalEn']);
 PERFORM vec_contratacion_temporal.seguimiento73_micro(consumos->'ConsumidaCTEn',false);
 PERFORM vec_contratacion_temporal.seguimiento73_micro(consumos->'LeidaPersonalEn',false);
 ct_en:=(consumos->>'ConsumidaCTEn')::timestamptz; leida_en:=(consumos->>'LeidaPersonalEn')::timestamptz;
 PERFORM vec_contratacion_temporal.incorporacion75_evidencia(r.evidencia_orden_json,r.material_json,r.exportacion_ct,r.persona_version,r.perfil_version,r.registrada_en);
 he:=vec_contratacion_temporal.incorporacion75_autoridad(r.material_json,r.exportacion_ct,r.persona_version,r.perfil_version,ct_en);
 -- Estas funciones cotejan bytes/ventanas ORIGINALES; no consumen ni acreditan RBAC hoy.
 IF he IS DISTINCT FROM consumos->>'HuellaExportacionCT'
 OR consumos->>'DecisionCTRef' IS DISTINCT FROM (convert_from(r.exportacion_ct[1],'UTF8')::jsonb)->>'decision_ref'
 OR (r.evidencia_orden_json->>'EvaluadaEn')::timestamptz>ct_en
 OR ct_en>leida_en OR leida_en>r.registrada_en
 OR consumos->>'DecisionCTRef' IS NOT DISTINCT FROM consumos->>'DecisionLecturaRef'
 OR consumos->>'DecisionLecturaRef' IS NOT DISTINCT FROM r.material_json#>>'{Personal,DecisionOriginalRef}'
 OR consumos->>'DecisionCTRef' IS NOT DISTINCT FROM r.material_json#>>'{Personal,DecisionOriginalRef}'
 OR consumos->>'ConsumoCTSHA256' IS NOT DISTINCT FROM consumos->>'ConsumoLecturaSHA256'
 OR consumos->>'AuditoriaAD3CTRef' IS DISTINCT FROM 'aud_v3_'||substr(consumos->>'ConsumoCTSHA256',1,32)
 OR consumos->>'AuditoriaAD3LecturaRef' IS DISTINCT FROM 'aud_v3_'||substr(consumos->>'ConsumoLecturaSHA256',1,32)
 THEN RAISE EXCEPTION 'CT81: evidencia historica incompatible' USING ERRCODE='55000'; END IF;
 FOREACH k IN ARRAY ARRAY['HuellaExportacionCT','ConsumoCTSHA256','ConsumoLecturaSHA256'] LOOP
  IF jsonb_typeof(consumos->k) IS DISTINCT FROM 'string' OR octet_length(consumos->>k)<>64
   OR consumos->>k !~ '^[0-9a-f]{64}$' OR consumos->>k=repeat('0',64)
  THEN RAISE EXCEPTION 'CT81: consumo historico incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['DecisionCTRef','DecisionLecturaRef','AuditoriaPersonalLecturaRef'] LOOP
  IF jsonb_typeof(consumos->k) IS DISTINCT FROM 'string' OR octet_length(consumos->>k) NOT BETWEEN 3 AND 160
   OR consumos->>k !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  THEN RAISE EXCEPTION 'CT81: referencia historica incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 aplicado:=vec_contratacion_temporal.aplicar_incorporacion_seguimiento_v2(pub,a.estado_json,r.material_json,
  r.recibo_json#>>'{Transicion,actuacion_ref}',r.recibo_ref,vec_contratacion_temporal.incorporacion75_instante(r.registrada_en));
 recibo:=vec_contratacion_temporal.incorporacion75_recibo(r.material_json,z.seguimiento_ref,pub,a.estado_json,p.estado_json,
  aplicado->'evento',consumos,r.auditoria_ref,r.outbox_ref);
 IF decode(aplicado->>'estado_canonico_base64','base64') IS DISTINCT FROM cp
 OR aplicado->>'estado_sha256' IS DISTINCT FROM p.estado_sha256
 OR r.recibo_json IS DISTINCT FROM recibo
 OR aud.recibo_ref IS DISTINCT FROM r.recibo_ref OR aud.recuperado IS DISTINCT FROM false
 OR aud.consumos_json IS DISTINCT FROM consumos OR aud.registrada_en IS DISTINCT FROM r.registrada_en
 OR aud.decision_ct_ref IS DISTINCT FROM consumos->>'DecisionCTRef'
 OR aud.decision_lectura_ref IS DISTINCT FROM consumos->>'DecisionLecturaRef'
 OR aud.consumo_ct_sha256 IS DISTINCT FROM consumos->>'ConsumoCTSHA256'
 OR aud.consumo_lectura_sha256 IS DISTINCT FROM consumos->>'ConsumoLecturaSHA256'
 OR ob.recibo_ref IS DISTINCT FROM r.recibo_ref OR ob.evento_json IS DISTINCT FROM aplicado->'evento'
 OR ob.estado_sha256 IS DISTINCT FROM p.estado_sha256 OR ob.creada_en IS DISTINCT FROM r.registrada_en
 THEN RAISE EXCEPTION 'CT81: registro durable incoherente' USING ERRCODE='55000'; END IF;
 EXCEPTION WHEN SQLSTATE '22023' OR data_exception THEN
  RAISE EXCEPTION 'localizador: canon propietario incompatible' USING ERRCODE='55000';
 END;
 recibo_ref:=r.recibo_ref; material_sha256:=r.material_sha256; intencion_sha256:=r.intencion_sha256;
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
REVOKE ALL ON FUNCTION vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text) FROM PUBLIC;
-- Crear objetos solo con defaults owner-only; no limpiar permisos ajenos silenciosamente.
DO $acl$
DECLARE f oid:='vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text)'::regprocedure;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'CT81: defaults incompatibles' USING ERRCODE='55000'; END IF;
END $acl$;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_localizador_incorporacion;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text)
 TO vec_contratacion_temporal_localizador_incorporacion;
COMMIT;
