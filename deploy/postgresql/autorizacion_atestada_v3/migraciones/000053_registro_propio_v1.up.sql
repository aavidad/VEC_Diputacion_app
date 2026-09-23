\set ON_ERROR_STOP on
-- AD3-53. Perfil nominal de un solo uso para registro propio VEC.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000053',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_version_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_identidad_sesiones_v1_provisionador' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-53: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registro_propio_sesion_nominal_v1()
RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT COALESCE(
  current_setting('transaction_isolation')='serializable'
  AND current_setting('transaction_read_only')='off' AND current_setting('TimeZone')='UTC'
  AND current_user='vec_autorizacion_atestada_v3_propietario' AND session_user<>current_user
  AND EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin
   AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
  AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1
  AND EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole
   AND roleid='vec_identidad_sesiones_v1_provisionador'::regrole
   AND NOT admin_option AND inherit_option AND NOT set_option)
  AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_identidad_sesiones_v1_provisionador'::regrole),false)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registro_propio_sesion_nominal_v1() FROM PUBLIC;

DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; meta jsonb; deps jsonb; acl aclitem[];
 guarda text:=$g$/* AD3-51 GUARDA INICIO */ (CASE WHEN p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$g$;
 guarda_nueva text:=$g$/* AD3-51 GUARDA INICIO */ (CASE WHEN p_perfil_mutacion='registro_propio' THEN vec_autorizacion_atestada_v3.registro_propio_sesion_nominal_v1() IS NOT TRUE WHEN p_perfil_mutacion IN ('contacto_usuario_alta','contacto_usuario_actualizar','contacto_usuario_consultar','contacto_usuario_recibo','contacto_usuario_version_propia','contacto_usuario_version_llamamiento')$g$;
 excl text:=$e$               AND p_perfil_mutacion IS DISTINCT FROM 'acceso_rutas_dietas'$e$;
 excl_nueva text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''registro_propio''';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_propio'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.registro_propio.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.registro_propio.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'vec.registro_propio.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'registro_propio'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'alta_vec_propia'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc',p.proacl INTO STRICT original,meta,acl
 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
  AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
  AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='41a3b44472f819cd16bb6519c4866190d3ea51c6bb6da233fbd56672c38b1c74';
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
 OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
 IF length(original)-length(replace(original,guarda,''))<>length(guarda)
 OR length(original)-length(replace(original,excl,''))<>length(excl)
 OR length(original)-length(replace(original,marca,''))<>length(marca)
 OR strpos(original,'registro_propio')<>0
 THEN RAISE EXCEPTION 'AD3-53: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(replace(original,guarda,guarda_nueva),excl,excl_nueva),marca,extension||marca);
 EXECUTE nuevo;
 IF pg_get_functiondef(f) IS DISTINCT FROM nuevo
 OR replace(replace(replace(pg_get_functiondef(f),extension||marca,marca),excl_nueva,excl),guarda_nueva,guarda) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
  OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-53: núcleo divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE def text; valores text[]; canon text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(oid,true),'\s+',' ','g') INTO STRICT def FROM pg_constraint
 WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND conname='clave_capacidad_version_audiencia_consumo_check' AND contype='c' AND convalidated AND conkey=ARRAY[8]::smallint[];
 SELECT array_agg(m[1] ORDER BY n) INTO valores FROM regexp_matches(def,'''([a-zA-Z0-9_.-]+)''::text','g') WITH ORDINALITY x(m,n);
 canon:='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(
 SELECT quote_literal(v)||'::text' FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||']))';
 IF def IS DISTINCT FROM canon OR cardinality(valores) NOT BETWEEN 15 AND 67
 OR NOT ARRAY['vec.contacto_usuario.recibo.v1','vec.contacto_usuario.version_propia.v1','vec.contacto_usuario.version_llamamiento.v1']<@valores
 OR 'vec.registro_propio.v1'=ANY(valores)
 THEN RAISE EXCEPTION 'AD3-53: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 valores:=valores||ARRAY['vec.registro_propio.v1'];
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||
 array_to_string(ARRAY(SELECT quote_literal(v) FROM unnest(valores) WITH ORDINALITY x(v,n) ORDER BY n),', ')||'))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF vec_autorizacion_atestada_v3.registro_propio_sesion_nominal_v1() IS NOT TRUE
 THEN RAISE EXCEPTION 'AD3-53: sesión denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-53: material inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec.registro_propio.v1'
 OR c->>'operacion' IS DISTINCT FROM 'vec.registro_propio.crear'
 OR d->>'accion' IS DISTINCT FROM 'vec.registro_propio.crear'
 OR d->>'modulo_id' IS DISTINCT FROM 'vec' OR d->>'tipo_recurso' IS DISTINCT FROM 'registro_propio'
 OR d->>'finalidad' IS DISTINCT FROM 'alta_vec_propia'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-53: capacidad divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 'registro_propio',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.auditoria_ref IS NULL OR x.efecto_ref IS DISTINCT FROM d->>'recurso_ref'
 THEN RAISE EXCEPTION 'AD3-53: consumo denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_identidad_sesiones_v1_propietario;
COMMIT;
