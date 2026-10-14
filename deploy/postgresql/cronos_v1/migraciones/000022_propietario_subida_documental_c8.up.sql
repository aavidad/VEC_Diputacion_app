\set ON_ERROR_STOP on
-- CRN22, subfase de titularidad C8. Documentos revalida el acto de subida
-- contra la solicitud y el último estado de Cronos en su transacción
-- serializable. No crea por sí sola una subida ni un enlace de justificante.
-- La función de cuatro argumentos solo prueba titularidad/versión; Documentos
-- debe acreditar sesión, decisión positiva y auditoría antes de consumirla.
-- Posterior a CRN21 y a la creación privada del LOGIN documental de Cronos.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000022',0));
DO $pre$
DECLARE politicas integer;
BEGIN
 IF current_user<>'vec_cronos_v1_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_cronos_v1.permiso_solicitud') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_estado') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_catalogo') IS NULL
    OR to_regprocedure('vec_cronos_v1.validar_propietario_subida_documental_v1(text,bigint,text,text)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_interno_documentos_cronos_datos'
       AND rolcanlogin AND rolinherit AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreaterole AND NOT rolcreatedb
       AND NOT rolreplication AND rolconfig IS NULL)
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_interno_documentos_cronos_datos'::regrole
       AND m.roleid='vec_documentos_ejecutor'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_interno_documentos_cronos_datos'::regrole
       AND m.roleid<>'vec_documentos_ejecutor'::regrole)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_documentos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_class c WHERE c.oid IN ('vec_cronos_v1.permiso_solicitud'::regclass,
      'vec_cronos_v1.permiso_estado'::regclass,'vec_cronos_v1.permiso_catalogo'::regclass)
      AND (c.relowner<>'vec_cronos_v1_propietario'::regrole OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
 THEN RAISE EXCEPTION 'CRN22: preimagen de titularidad incompatible' USING ERRCODE='55000'; END IF;
 SELECT count(*) INTO politicas FROM pg_policy p JOIN pg_class c ON c.oid=p.polrelid
 JOIN (VALUES
  ('permiso_solicitud','adicion_propia','a','','c97e713cb8239cb746d0d0ad0420530fceec95d03e21b08b24b3eaae519073d2'),
  ('permiso_solicitud','lectura_propia','r','c97e713cb8239cb746d0d0ad0420530fceec95d03e21b08b24b3eaae519073d2',''),
  ('permiso_solicitud','lectura_resolutor','r','5e22177e4ac7cd04cd97be7997b0eea616770cb80803b5455c90097fd299d283',''),
  ('permiso_estado','adicion_propia','a','','c97e713cb8239cb746d0d0ad0420530fceec95d03e21b08b24b3eaae519073d2'),
  ('permiso_estado','adicion_resolutor','a','','90c497225deb35979e9b07d4aa93dc0ca7ca787af36a7b0bac63b5d1faedcd10'),
  ('permiso_estado','lectura_propia','r','c97e713cb8239cb746d0d0ad0420530fceec95d03e21b08b24b3eaae519073d2',''),
  ('permiso_estado','lectura_resolutor','r','5e22177e4ac7cd04cd97be7997b0eea616770cb80803b5455c90097fd299d283','')
 ) esperado(tabla,nombre,orden,lectura,escritura)
   ON c.relname=esperado.tabla AND p.polname=esperado.nombre AND p.polcmd=esperado.orden
 WHERE c.relnamespace='vec_cronos_v1'::regnamespace AND p.polpermissive
   AND p.polroles=ARRAY['vec_cronos_v1_propietario'::regrole::oid]
   AND coalesce(encode(sha256(convert_to(pg_get_expr(p.polqual,p.polrelid),'UTF8')),'hex'),'')=esperado.lectura
   AND coalesce(encode(sha256(convert_to(pg_get_expr(p.polwithcheck,p.polrelid),'UTF8')),'hex'),'')=esperado.escritura;
 IF politicas<>7 OR (SELECT count(*) FROM pg_policy p WHERE p.polrelid IN
   ('vec_cronos_v1.permiso_solicitud'::regclass,'vec_cronos_v1.permiso_estado'::regclass))<>7 THEN
   RAISE EXCEPTION 'CRN22: políticas HZ37 incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

-- Las tablas conservan sus políticas propias. Estas dos lecturas solo son
-- posibles desde la función propietaria y el LOGIN documental exacto.
CREATE POLICY lectura_documental_propia ON vec_cronos_v1.permiso_solicitud
 FOR SELECT TO vec_cronos_v1_propietario USING
 (session_user='vec_interno_documentos_cronos_datos'
  AND actor_ref=nullif(current_setting('vec.cronos.documental_persona_ref',true),''));
CREATE POLICY lectura_documental_propia ON vec_cronos_v1.permiso_estado
 FOR SELECT TO vec_cronos_v1_propietario USING
 (session_user='vec_interno_documentos_cronos_datos'
  AND EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_solicitud s
    WHERE s.solicitud_ref=permiso_estado.solicitud_ref
      AND s.actor_ref=nullif(current_setting('vec.cronos.documental_persona_ref',true),'')));
GRANT USAGE ON SCHEMA vec_cronos_v1 TO vec_documentos_propietario;

CREATE FUNCTION vec_cronos_v1.validar_propietario_subida_documental_v1(
 p_solicitud_ref text,p_version bigint,p_persona_ref text,p_contexto_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $f$
DECLARE anterior text; solicitud vec_cronos_v1.permiso_solicitud%ROWTYPE;
 estado vec_cronos_v1.permiso_estado%ROWTYPE; exigido boolean;
BEGIN
 anterior:=current_setting('vec.cronos.documental_persona_ref',true);
 IF current_user<>'vec_cronos_v1_propietario' OR session_user<>'vec_interno_documentos_cronos_datos'
    OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
      AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolbypassrls AND NOT r.rolcreaterole
      AND NOT r.rolcreatedb AND NOT r.rolreplication AND r.rolconfig IS NULL)
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
      AND m.roleid='vec_documentos_ejecutor'::regrole AND m.inherit_option
      AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR pg_has_role(session_user,'vec_cronos_v1_propietario','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CRN22: consumidor documental incompatible' USING ERRCODE='42501'; END IF;
 IF p_solicitud_ref IS NULL OR p_solicitud_ref !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR p_version IS NULL OR p_version NOT BETWEEN 1 AND 1000
    OR p_persona_ref IS NULL OR p_persona_ref !~ '^per_[-A-Za-z0-9_]{22,128}$'
    OR p_contexto_ref IS NULL OR p_contexto_ref !~ '^rca_[-A-Za-z0-9_]{22,128}$'
 THEN RETURN false; END IF;
 PERFORM set_config('vec.cronos.documental_persona_ref',p_persona_ref,true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:solicitud:'||p_solicitud_ref,0));
 SELECT * INTO solicitud FROM vec_cronos_v1.permiso_solicitud s
  WHERE s.solicitud_ref=p_solicitud_ref AND s.actor_ref=p_persona_ref FOR SHARE;
 IF NOT FOUND THEN
   PERFORM set_config('vec.cronos.documental_persona_ref',coalesce(anterior,''),true);
   RETURN false;
 END IF;
 SELECT e.* INTO estado FROM vec_cronos_v1.permiso_estado e
  WHERE e.solicitud_ref=solicitud.solicitud_ref AND e.empleado_ref=solicitud.empleado_ref
  ORDER BY e.version DESC LIMIT 1 FOR SHARE;
 SELECT c.justificante_exigido INTO exigido FROM vec_cronos_v1.permiso_catalogo c
  WHERE c.version_ref=solicitud.catalogo_version_ref;
 PERFORM set_config('vec.cronos.documental_persona_ref',coalesce(anterior,''),true);
 RETURN coalesce(estado.version=p_version AND estado.estado='concedido'
   AND estado.pendiente_justificar AND exigido IS TRUE,false);
EXCEPTION WHEN OTHERS THEN
 PERFORM set_config('vec.cronos.documental_persona_ref',coalesce(anterior,''),true);
 RAISE;
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.validar_propietario_subida_documental_v1(text,bigint,text,text)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.validar_propietario_subida_documental_v1(text,bigint,text,text)
 TO vec_documentos_propietario;

DO $post$
DECLARE f pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO f FROM pg_proc WHERE oid=to_regprocedure('vec_cronos_v1.validar_propietario_subida_documental_v1(text,bigint,text,text)');
 IF NOT FOUND OR f.proowner IS DISTINCT FROM 'vec_cronos_v1_propietario'::regrole
    OR NOT f.prosecdef OR f.provolatile<>'v'
    OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
    OR f.proacl::text IS DISTINCT FROM
      '{vec_cronos_v1_propietario=X/vec_cronos_v1_propietario,vec_documentos_propietario=X/vec_cronos_v1_propietario}'
    OR NOT has_schema_privilege('vec_documentos_propietario','vec_cronos_v1','USAGE')
 THEN RAISE EXCEPTION 'CRN22: postimagen de función incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
