\set ON_ERROR_STOP on
-- AD3-103: consumidor nominal de consulta del catálogo de causas de participación.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000103',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolbypassrls)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_ejecutor' AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-103: preimagen incompatible (AD3-102 requerida)' USING ERRCODE='55000'; END IF;
END $pre$;
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''catalogo_causas_participacion''\n';
 excl_nuevo text:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_catalogo_causas_participacion''\n';
 runtime text:=E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''catalogo_causas_participacion''\n';
 runtime_nuevo text:=runtime||E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''consulta_catalogo_causas_participacion''\n';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_catalogo_causas_participacion'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.causas_participacion.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_causas_participacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'seleccionar_causa_participacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'vec.bolsa.causas_participacion'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["aplica_contacto","aplica_situacion","codigo","etiqueta","huella_sha256","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
 OR length(original)-length(replace(original,marca,''))<>length(marca) OR length(original)-length(replace(original,excl,''))<>length(excl) OR length(original)-length(replace(original,runtime,''))<>length(runtime)
 OR strpos(original,'''catalogo_causas_participacion''')=0 OR strpos(original,'vec_bolsa_llamamientos_ejecutor')=0 OR strpos(original,'consulta_catalogo_causas_participacion')<>0 OR strpos(original,'bolsa.causas_participacion.consultar')<>0
 THEN RAISE EXCEPTION 'AD3-103: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(replace(original,excl,excl_nuevo),runtime,runtime_nuevo),marca,extension||marca); EXECUTE nuevo; SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(replace(replace(actual,extension||marca,marca),runtime_nuevo,runtime),excl_nuevo,excl) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-103: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$ DECLARE d text; audiencia text:='vec_bolsa_llamamientos.causas_participacion.consultar.v1'; BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))' OR strpos(d,'vec_bolsa_llamamientos.causas_participacion.publicar.v1')=0 OR strpos(d,quote_literal(audiencia))<>0 THEN RAISE EXCEPTION 'AD3-103: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||left(d,length(d)-3)||', '||quote_literal(audiencia)||'::text]))';
END $audiencias$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-103: material de consulta de catálogo inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.consultar' OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.consultar.v1' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_causas_participacion' OR d->>'finalidad' IS DISTINCT FROM 'seleccionar_causa_participacion' OR d->>'recurso_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion' OR c->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' OR d->'campos_permitidos' IS DISTINCT FROM '["aplica_contacto","aplica_situacion","codigo","etiqueta","huella_sha256","version"]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'AD3-103: consulta de catálogo denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('consulta_catalogo_causas_participacion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-103: la consulta requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
DO $acl$ DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; permitido oid:='vec_bolsa_llamamientos_propietario'::regrole::oid; x record; BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner LOOP EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END); END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s'] OR NOT has_schema_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3','USAGE') THEN RAISE EXCEPTION 'AD3-103: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE' OR (x.grantee=permitido AND x.is_grantable) THEN RAISE EXCEPTION 'AD3-103: ACL de fachada abierta' USING ERRCODE='55000'; END IF; END LOOP;
END $acl$;

-- B57 DOWN necesita una guardia de historia sin recibir SELECT sobre el
-- registro V3. Los locks permanecen hasta el fin de su transacción y cierran
-- la carrera con una nueva atestación mientras se retiran las tablas Bolsa.
CREATE FUNCTION vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1()
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_historia boolean;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' THEN
  RAISE EXCEPTION 'AD3-103: consulta de historia denegada' USING ERRCODE='42501';
 END IF;
 LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN SHARE MODE;
 LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
 SELECT EXISTS (
   SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
   WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'operacion' IN
     ('bolsa.causas_participacion.publicar','bolsa.causas_participacion.consultar',
      'bolsa.causas_participacion.proponer','bolsa.causas_participacion.propuesta.consultar')
 ) OR EXISTS (
   SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version c
   WHERE c.audiencia_consumo IN
     ('vec_bolsa_llamamientos.causas_participacion.publicar.v1',
      'vec_bolsa_llamamientos.causas_participacion.consultar.v1',
      'vec_bolsa_llamamientos.causas_participacion.proponer.v1',
      'vec_bolsa_llamamientos.causas_participacion.propuesta.consultar.v1')
 ) INTO v_historia;
 RETURN v_historia;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.existe_historia_catalogo_causas_bolsa_v1() TO vec_bolsa_llamamientos_propietario;
COMMIT;
