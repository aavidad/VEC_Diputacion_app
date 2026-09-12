\set ON_ERROR_STOP on
-- Fachada de lectura histórica. No concede permiso actual ni reconstruye Orden.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000078:lectura_raices:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $pre$
DECLARE r oid:='vec_contratacion_temporal_lector_raices_historicas'::regrole;
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
  AND proname='leer_raices_incorporacion_ejercicio_v2')
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
  WHERE n.oid='vec_contratacion_temporal'::regnamespace AND a.grantee=r)
 THEN RAISE EXCEPTION 'CT78: precondiciones incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['seguimiento_raiz_v2','seguimiento_definicion_v2','seguimiento_estado_v2'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE relnamespace='vec_contratacion_temporal'::regnamespace
   AND relname=n AND relkind='r' AND NOT relispartition AND relowner=o AND relrowsecurity AND relforcerowsecurity)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.relnamespace='vec_contratacion_temporal'::regnamespace AND c.relname=n
    AND (a.grantee<>o OR a.grantor<>o))
  OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_attribute a ON a.attrelid=c.oid
   WHERE c.relnamespace='vec_contratacion_temporal'::regnamespace AND c.relname=n AND a.attnum>0
    AND (a.attisdropped OR a.attacl IS NOT NULL))
  THEN RAISE EXCEPTION 'CT78: fuente propietaria incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH n IN ARRAY ARRAY[
  'estado_seguimiento_canonico_v1(jsonb,jsonb)',
  'seguimiento73_definicion(jsonb)','seguimiento73_nodo(jsonb,text)'] LOOP
  f:=to_regprocedure('vec_contratacion_temporal.'||n);
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner=o AND NOT prosecdef AND provolatile='i')
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee<>o OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'CT78: codec propietario requerido' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(
 p_organizacion_ref text, p_expediente_ref text, p_relacion_ref text
) RETURNS TABLE (
 seguimiento_ref text, organizacion_ref text, expediente_ref text, relacion_ref text,
 version_expediente_observada text, definicion_ref text, definicion_version text,
 definicion_sha256 text, publicacion_json jsonb, raiz_canonica bytea, raiz_sha256 text,
 estado_inicial_json jsonb, estado_inicial_canonico bytea
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET statement_timeout='5s' SET lock_timeout='2s'
AS $raices$
DECLARE
 login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
 membresias integer; funciones oid[]; login record; grupo record;
 z vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;
 f vec_contratacion_temporal.seguimiento_definicion_v2%ROWTYPE;
 a vec_contratacion_temporal.seguimiento_estado_v2%ROWTYPE;
 def jsonb; raiz jsonb; canon bytea; v_selector text;
 cantidad bigint; presupuesto bigint; incompleto boolean; vistos integer:=0;
 inicio timestamptz:=clock_timestamp(); codigo text;
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
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_contratacion_temporal_lector_raices_historicas';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_contratacion_temporal';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[pg_catalog.to_regprocedure('vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(text,text,text)')];
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
           SELECT 1 FROM pg_catalog.pg_roles ct78_catalogo_r
            WHERE ct78_catalogo_r.oid<>login_oid
              AND pg_catalog.pg_has_role(login_oid,ct78_catalogo_r.oid,'MEMBER')
              AND ct78_catalogo_r.oid<>runtime_oid
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles ct78_catalogo_r
            WHERE ct78_catalogo_r.oid<>runtime_oid
              AND pg_catalog.pg_has_role(runtime_oid,ct78_catalogo_r.oid,'MEMBER')
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = runtime_oid
              AND admin_option IS FALSE AND inherit_option IS TRUE AND set_option IS FALSE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting ct78_catalogo_s
            WHERE ct78_catalogo_s.setrole IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl d
           LEFT JOIN LATERAL pg_catalog.aclexplode(
             coalesce(d.defaclacl,'{}'::aclitem[])
           ) ct78_catalogo_a ON true
            WHERE d.defaclrole IN (login_oid,runtime_oid)
               OR ct78_catalogo_a.grantee IN (login_oid,runtime_oid)
               OR ct78_catalogo_a.grantor IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy ct78_catalogo_p
            WHERE login_oid=ANY(ct78_catalogo_p.polroles) OR runtime_oid=ANY(ct78_catalogo_p.polroles)
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
           SELECT count(*)=1 AND bool_and(ct78_catalogo_a.privilege_type='CONNECT' AND NOT ct78_catalogo_a.is_grantable AND ct78_catalogo_a.grantor=b.datdba)
             FROM pg_catalog.pg_database b
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
             ) ct78_catalogo_a
            WHERE b.oid=base_oid AND ct78_catalogo_a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(ct78_catalogo_a.privilege_type='USAGE' AND NOT ct78_catalogo_a.is_grantable AND ct78_catalogo_a.grantor=n.nspowner)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
             ) ct78_catalogo_a
            WHERE n.oid=esquema_oid AND ct78_catalogo_a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND count(DISTINCT ct78_catalogo_p.oid)=1
                  AND bool_and(ct78_catalogo_a.privilege_type='EXECUTE' AND NOT ct78_catalogo_a.is_grantable AND ct78_catalogo_a.grantor=ct78_catalogo_p.proowner)
             FROM pg_catalog.pg_proc ct78_catalogo_p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(ct78_catalogo_p.proacl,pg_catalog.acldefault('f',ct78_catalogo_p.proowner))
            ) ct78_catalogo_a
            WHERE ct78_catalogo_p.oid=ANY(funciones) AND ct78_catalogo_a.grantee=runtime_oid
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
         SELECT 1 FROM pg_catalog.pg_proc ct78_catalogo_p
         JOIN pg_catalog.pg_namespace n ON n.oid=ct78_catalogo_p.pronamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
            AND ct78_catalogo_p.oid <> ALL(funciones)
            AND pg_catalog.has_function_privilege(login_oid,ct78_catalogo_p.oid,'EXECUTE')
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
         SELECT 1 FROM pg_catalog.pg_foreign_data_wrapper ct78_catalogo_f
          WHERE pg_catalog.has_foreign_data_wrapper_privilege(login_oid,ct78_catalogo_f.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_server ct78_catalogo_s
          WHERE pg_catalog.has_server_privilege(login_oid,ct78_catalogo_s.oid,'USAGE')
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
         SELECT 1 FROM pg_catalog.pg_parameter_acl ct78_catalogo_a
          WHERE pg_catalog.has_parameter_privilege(login_oid,ct78_catalogo_a.parname,'SET')
             OR pg_catalog.has_parameter_privilege(login_oid,ct78_catalogo_a.parname,'ALTER SYSTEM')
       )) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN lector historico no acreditado';
    END IF;


 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contratacion_temporal:000078:lectura_raices:v2',0));

 FOREACH v_selector IN ARRAY ARRAY[p_organizacion_ref,p_expediente_ref,p_relacion_ref] LOOP
  IF v_selector IS NULL OR octet_length(v_selector)<>68
   OR v_selector !~ '^ref:[0-9a-f]{64}$' OR v_selector='ref:'||repeat('0',64)
  THEN RAISE EXCEPTION 'CT78: selector invalido' USING ERRCODE='22023'; END IF;
 END LOOP;

 -- No LIMIT ni INNER JOIN: piezas ausentes deben fallar, no ocultar candidatos.
 -- Toda la llamada ve la misma instantanea SERIALIZABLE READ ONLY.
 SELECT count(*) INTO cantidad
 FROM vec_contratacion_temporal.seguimiento_raiz_v2 r
 WHERE r.organizacion_ref=p_organizacion_ref AND r.expediente_ref=p_expediente_ref AND r.relacion_ref=p_relacion_ref;
 IF cantidad>16 THEN RAISE EXCEPTION 'CT78: limite de candidatos' USING ERRCODE='55000'; END IF;
 -- SUMA exacta de los trece campos transportados; bigint antes de sumar.
 SELECT coalesce(sum(
  octet_length(r.seguimiento_ref)::bigint+octet_length(r.organizacion_ref)+octet_length(r.expediente_ref)+octet_length(r.relacion_ref)
  +octet_length(r.version_expediente_observada::text)+octet_length(r.definicion_ref)+octet_length(r.definicion_version::text)
  +octet_length(r.definicion_sha256)+octet_length(d.publicacion_json::text)+octet_length(r.raiz_canonica)+octet_length(r.raiz_sha256)
  +octet_length(e.estado_json::text)+octet_length(e.estado_canonico)),0),
  coalesce(bool_or(d.definicion_ref IS NULL OR e.seguimiento_ref IS NULL),false)
 INTO presupuesto,incompleto
 FROM vec_contratacion_temporal.seguimiento_raiz_v2 r
 LEFT JOIN vec_contratacion_temporal.seguimiento_definicion_v2 d
  ON (d.definicion_ref,d.definicion_version,d.definicion_sha256)=(r.definicion_ref,r.definicion_version,r.definicion_sha256)
 LEFT JOIN vec_contratacion_temporal.seguimiento_estado_v2 e ON e.seguimiento_ref=r.seguimiento_ref AND e.version_seguimiento=0
 WHERE r.organizacion_ref=p_organizacion_ref AND r.expediente_ref=p_expediente_ref AND r.relacion_ref=p_relacion_ref;
 IF incompleto OR presupuesto>33554432
 THEN RAISE EXCEPTION 'CT78: fuente incompleta o excesiva' USING ERRCODE='55000'; END IF;

 FOR z IN SELECT r.* FROM vec_contratacion_temporal.seguimiento_raiz_v2 r
  WHERE r.organizacion_ref=p_organizacion_ref AND r.expediente_ref=p_expediente_ref AND r.relacion_ref=p_relacion_ref
  ORDER BY r.seguimiento_ref COLLATE "C"
 LOOP
  vistos:=vistos+1;
  IF vistos>16 OR clock_timestamp()<inicio OR clock_timestamp()>inicio+interval '5 seconds'
  THEN RAISE EXCEPTION 'CT78: lectura excede limite' USING ERRCODE='55000'; END IF;
  SELECT d.* INTO STRICT f FROM vec_contratacion_temporal.seguimiento_definicion_v2 d
   WHERE (d.definicion_ref,d.definicion_version,d.definicion_sha256)=(z.definicion_ref,z.definicion_version,z.definicion_sha256);
  SELECT e.* INTO STRICT a FROM vec_contratacion_temporal.seguimiento_estado_v2 e
   WHERE e.seguimiento_ref=z.seguimiento_ref AND e.version_seguimiento=0;
  IF z.version_inicial IS DISTINCT FROM 0::numeric
   OR z.version_expediente_observada NOT BETWEEN 1 AND 9007199254740991
   OR z.version_expediente_observada::text !~ '^[1-9][0-9]*$'
   OR z.definicion_version::text !~ '^[1-9][0-9]*$'
   OR a.version_anterior IS NOT NULL OR a.estado_anterior_sha256 IS NOT NULL
   OR (a.organizacion_ref,a.expediente_ref,a.relacion_ref,a.definicion_ref,a.definicion_version,a.definicion_sha256,a.raiz_sha256)
    IS DISTINCT FROM (z.organizacion_ref,z.expediente_ref,z.relacion_ref,z.definicion_ref,z.definicion_version,z.definicion_sha256,z.raiz_sha256)
   OR octet_length(f.publicacion_json::text) NOT BETWEEN 2 AND 8388608
   OR octet_length(f.definicion_canonica) NOT BETWEEN 1 AND 8388608
   OR octet_length(z.raiz_canonica) NOT BETWEEN 1 AND 8388608
   OR octet_length(a.estado_json::text) NOT BETWEEN 2 AND 8388608
   OR octet_length(a.estado_canonico) NOT BETWEEN 1 AND 8388608
  THEN RAISE EXCEPTION 'CT78: fila original incompatible' USING ERRCODE='55000'; END IF;

  -- CT73 restaura la definicion y reproduce toda la semantica de v0.
  -- No se compara su vigencia con el reloj actual.
  BEGIN
   def:=vec_contratacion_temporal.seguimiento73_definicion(f.publicacion_json);
   canon:=vec_contratacion_temporal.estado_seguimiento_canonico_v1(f.publicacion_json,a.estado_json);
   raiz:=jsonb_build_object('referencia',z.seguimiento_ref,'organizacion_ref',z.organizacion_ref,
    'expediente_ref',z.expediente_ref,'relacion_ref',z.relacion_ref,
    'definicion',a.estado_json->'definicion','estado_actual',def->'estado_inicial',
    'periodo_previsto',a.estado_json->'periodo_previsto','creado_en',a.estado_json->'creado_en');
   IF def->>'referencia' IS DISTINCT FROM z.definicion_ref
    OR (def->>'version')::numeric IS DISTINCT FROM z.definicion_version
    OR def->>'huella_sha256' IS DISTINCT FROM z.definicion_sha256
    OR f.definicion_canonica IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_nodo(def,'publicacion')
    OR encode(sha256(f.definicion_canonica),'hex') IS DISTINCT FROM z.definicion_sha256
    OR a.estado_json->'version' IS DISTINCT FROM '0'::jsonb
    OR a.estado_json->>'referencia' IS DISTINCT FROM z.seguimiento_ref
    OR a.estado_json->>'organizacion_ref' IS DISTINCT FROM z.organizacion_ref
    OR a.estado_json->>'expediente_ref' IS DISTINCT FROM z.expediente_ref
    OR a.estado_json->>'relacion_ref' IS DISTINCT FROM z.relacion_ref
    OR a.estado_json->'definicion' IS DISTINCT FROM jsonb_build_object('referencia',z.definicion_ref,'version',z.definicion_version,'huella_sha256',z.definicion_sha256)
    OR canon IS DISTINCT FROM a.estado_canonico
    OR encode(sha256(canon),'hex') IS DISTINCT FROM a.estado_sha256
    OR z.raiz_canonica IS DISTINCT FROM vec_contratacion_temporal.seguimiento73_nodo(raiz,'raiz')
    OR encode(sha256(z.raiz_canonica),'hex') IS DISTINCT FROM z.raiz_sha256
    OR a.estado_json->>'huella_raiz_sha256' IS DISTINCT FROM z.raiz_sha256
   THEN RAISE EXCEPTION 'CT78: canon original incompatible' USING ERRCODE='55000'; END IF;
  EXCEPTION WHEN OTHERS THEN
   RAISE EXCEPTION 'CT78: integridad original no acreditada' USING ERRCODE='55000';
  END;
  -- RETURN NEXT almacena filas en el tuplestore: ningun parcial se entrega
  -- si falla un candidato posterior o el control final de esta funcion.
  seguimiento_ref:=z.seguimiento_ref; organizacion_ref:=z.organizacion_ref;
  expediente_ref:=z.expediente_ref; relacion_ref:=z.relacion_ref;
  version_expediente_observada:=z.version_expediente_observada::text;
  definicion_ref:=z.definicion_ref; definicion_version:=z.definicion_version::text;
  definicion_sha256:=z.definicion_sha256; publicacion_json:=f.publicacion_json;
  raiz_canonica:=z.raiz_canonica; raiz_sha256:=z.raiz_sha256;
  estado_inicial_json:=a.estado_json; estado_inicial_canonico:=a.estado_canonico;
  RETURN NEXT;
 END LOOP;
 IF vistos<>cantidad OR clock_timestamp()<inicio OR clock_timestamp()>inicio+interval '5 seconds'
 THEN RAISE EXCEPTION 'CT78: retorno no disponible' USING ERRCODE='55000'; END IF;
 RETURN;
EXCEPTION WHEN OTHERS THEN
 codigo:=SQLSTATE;
 IF codigo NOT IN('42501','25000','22023','40001','40P01','55P03') THEN codigo:='55000'; END IF;
 RAISE EXCEPTION 'CT78: raices originales no disponibles' USING ERRCODE=codigo;
END $raices$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(text,text,text) FROM PUBLIC;
-- Crear objetos solo con defaults owner-only; no limpiar permisos ajenos silenciosamente.
DO $acl$
DECLARE f oid:='vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(text,text,text)'::regprocedure;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'CT78: defaults incompatibles' USING ERRCODE='55000'; END IF;
END $acl$;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_lector_raices_historicas;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(text,text,text)
 TO vec_contratacion_temporal_lector_raices_historicas;
COMMIT;
