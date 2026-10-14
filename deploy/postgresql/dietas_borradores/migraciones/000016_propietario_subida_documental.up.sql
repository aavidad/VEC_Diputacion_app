\set ON_ERROR_STOP on
-- Dietas 16: Documentos puede comprobar que el justificante pertenece a la
-- persona titular de una comisión en su última versión editable. Documentos
-- acredita la sesión, el contexto y la decisión V3 antes de llamar aquí;
-- Dietas solo decide titularidad, versión y estado de su propia comisión.
-- Depende del contrato de subida común Documentos 000020.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000016:propietario-subida',0));

DO $pre$
DECLARE cantidad integer;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_dietas_propietario'
    OR to_regprocedure('vec_dietas.validar_propietario_subida_documental_v1(text,bigint,text,text)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_documentos'
         AND p.proname='validar_propietario_subida_local_v1'
         AND oidvectortypes(p.proargtypes)='text, text, bigint, text, text, numeric'
         AND p.proowner='vec_documentos_propietario'::regrole AND p.prosecdef)
    OR to_regclass('vec_dietas.borrador_comision') IS NULL
    OR to_regclass('vec_dietas.comision_revision') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_documentos_propietario' AND NOT rolcanlogin)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_documentos_ejecutor' AND NOT rolcanlogin)
 THEN RAISE EXCEPTION 'Dietas16: dependencia o entorno incompatible' USING ERRCODE='55000'; END IF;

 SELECT count(*) INTO cantidad FROM pg_class c
 WHERE c.oid IN ('vec_dietas.borrador_comision'::regclass,
                 'vec_dietas.comision_revision'::regclass)
   AND c.relowner='vec_dietas_propietario'::regrole
   AND c.relkind='r' AND c.relrowsecurity AND c.relforcerowsecurity;
 IF cantidad<>2 THEN RAISE EXCEPTION 'Dietas16: tablas o RLS incompatibles' USING ERRCODE='55000'; END IF;

 SELECT count(*) INTO cantidad FROM pg_policy p
 WHERE p.polrelid IN ('vec_dietas.borrador_comision'::regclass,
                      'vec_dietas.comision_revision'::regclass)
   AND p.polname='persona_contextual' AND p.polcmd='*' AND p.polpermissive
   AND p.polroles=ARRAY['vec_dietas_propietario'::regrole::oid]
   AND encode(sha256(convert_to(pg_get_expr(p.polqual,p.polrelid),'UTF8')),'hex') =
     CASE p.polrelid
      WHEN 'vec_dietas.borrador_comision'::regclass
       THEN '034e226e46d588c1f95cfb03f1a924c105388a9921c64d3750cba535ea4f1965'
      ELSE '8b61b52ed6efe5e77f4b196091ac482feec1632e6e9d14ba5db4a6f3ea1e8bc3' END
   AND pg_get_expr(p.polwithcheck,p.polrelid)=pg_get_expr(p.polqual,p.polrelid);
 IF cantidad<>2 THEN RAISE EXCEPTION 'Dietas16: política propia incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

GRANT USAGE ON SCHEMA vec_dietas TO vec_documentos_propietario;

CREATE FUNCTION vec_dietas.validar_propietario_subida_documental_v1(
 p_comision_ref text,p_version bigint,p_persona_ref text,p_contexto_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE anterior_persona text; cabecera vec_dietas.borrador_comision%ROWTYPE;
 revision vec_dietas.comision_revision%ROWTYPE; valido boolean:=false;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_documentos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Dietas16: consumidor documental incompatible' USING ERRCODE='42501'; END IF;
 IF p_comision_ref IS NULL OR p_comision_ref !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR p_version IS NULL OR p_version<1
    OR p_persona_ref IS NULL OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_contexto_ref IS NULL OR p_contexto_ref !~ '^rca_[A-Za-z0-9_-]{22,128}$'
 THEN RETURN false; END IF;

 anterior_persona:=current_setting('vec.dietas.persona_ref',true);
 PERFORM set_config('vec.dietas.persona_ref',p_persona_ref,true);
 -- El bloqueo de cabecera coincide con el de editar/enviar. FOR SHARE de la
 -- última revisión coincide con la guarda del circuito. No hay ventana entre
 -- validar y la reserva/vinculación documental en la transacción Documentos.
 SELECT * INTO cabecera FROM vec_dietas.borrador_comision
  WHERE referencia=p_comision_ref AND persona_ref=p_persona_ref FOR UPDATE;
 IF FOUND THEN
  SELECT * INTO revision FROM vec_dietas.comision_revision
   WHERE comision_ref=p_comision_ref ORDER BY version DESC LIMIT 1 FOR SHARE;
  IF FOUND THEN
   valido:=revision.version=p_version AND revision.estado IN ('borrador','devuelta');
  ELSE
   valido:=p_version=1 AND cabecera.version=1;
  END IF;
 END IF;
 PERFORM set_config('vec.dietas.persona_ref',coalesce(anterior_persona,''),true);
 RETURN valido;
EXCEPTION WHEN OTHERS THEN
 PERFORM set_config('vec.dietas.persona_ref',coalesce(anterior_persona,''),true);
 RAISE;
END $funcion$;
ALTER FUNCTION vec_dietas.validar_propietario_subida_documental_v1(text,bigint,text,text)
 OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.validar_propietario_subida_documental_v1(text,bigint,text,text)
 FROM PUBLIC,vec_dietas_ejecutor;
GRANT EXECUTE ON FUNCTION vec_dietas.validar_propietario_subida_documental_v1(text,bigint,text,text)
 TO vec_documentos_propietario;

DO $post$
DECLARE f pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO f FROM pg_proc
  WHERE oid=to_regprocedure('vec_dietas.validar_propietario_subida_documental_v1(text,bigint,text,text)');
 IF NOT FOUND OR f.proowner IS DISTINCT FROM 'vec_dietas_propietario'::regrole
    OR NOT f.prosecdef OR f.provolatile<>'v'
    OR f.proconfig::text IS DISTINCT FROM
       '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,lock_timeout=2s}'
    OR f.proacl::text IS DISTINCT FROM
       '{vec_dietas_propietario=X/vec_dietas_propietario,vec_documentos_propietario=X/vec_dietas_propietario}'
    OR NOT has_schema_privilege('vec_documentos_propietario','vec_dietas','USAGE')
    OR has_function_privilege('vec_dietas_ejecutor',f.oid,'EXECUTE')
 THEN RAISE EXCEPTION 'Dietas16: ACL o función incompatibles' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
