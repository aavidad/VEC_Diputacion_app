\set ON_ERROR_STOP on
-- AD213. Tres lecturas RRHH de Bolsa con acciones y audiencias propias.
-- No crea concesiones humanas; sólo habilita el consumo exacto para B84.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000213',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD213: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 -- Preimagen del clon sintético S, PostgreSQL 18.4, posterior a AD207/208.
 esperada text:='536ea653143147e0cfb2d3277948530acb8d4d1643e44f4786462fe5ad1379fe';
 marca text:=$m$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$m$;
 excl text:=$e$               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_politica_cese_bolsa'
$e$;
 excl_nuevo text:=$en$               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_politica_cese_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_rrhh_bolsa'
$en$;
 runtime text:=$r$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_cese_bolsa'
$r$;
 runtime_nuevo text:=$rn$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_cese_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_rrhh_bolsa'
$rn$;
 extension text:=$x$           OR (
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
    AND d->>'recurso_ref' ~ '^bolsa:[A-Za-z0-9:_-]+:filtro:[a-f0-9]{64}$'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_rrhh_candidatos'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["candidatos","contactos","turno"]'::jsonb)))
$x$;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc p WHERE p.oid=f;
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada
    OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
    OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,excl,''))<>pg_catalog.length(excl)
    OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,runtime,''))<>pg_catalog.length(runtime)
    OR pg_catalog.strpos(original,'consulta_rrhh_bolsa')<>0
 THEN RAISE EXCEPTION 'AD213: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,excl,excl_nuevo);
 nuevo:=pg_catalog.replace(nuevo,runtime,runtime_nuevo);
 nuevo:=pg_catalog.replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR pg_catalog.replace(pg_catalog.replace(pg_catalog.replace(actual,extension||marca,marca),runtime_nuevo,runtime),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD213: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; a text;
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(d,'UTF8')),'hex')
      IS DISTINCT FROM 'af58417e64bd6a982745ae540d7836cbb560bce982fe1d4f5a035a15a835802e'
    OR pg_catalog.left(d,7)<>'CHECK (' OR pg_catalog.right(d,1)<>')'
    OR pg_catalog.strpos(d,'audiencia_consumo')=0
 THEN RAISE EXCEPTION 'AD213: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH a IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1',
  'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1',
  'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1'] LOOP
   IF pg_catalog.strpos(d,pg_catalog.quote_literal(a))<>0 THEN
     RAISE EXCEPTION 'AD213: audiencia ya presente' USING ERRCODE='55000';
   END IF;
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
  ||pg_catalog.format('CHECK (%s OR audiencia_consumo = ANY (ARRAY[%L::text,%L::text,%L::text]))',
     pg_catalog.substr(d,8,pg_catalog.length(d)-8),
     'vec_bolsa_llamamientos.rrhh.bolsas.consultar.v1',
     'vec_bolsa_llamamientos.rrhh.estadisticas.consultar.v1',
     'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1');
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(
 p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb; d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD213: material invalido' USING ERRCODE='22023'; END;
 IF p_accion NOT IN ('bolsa.rrhh.bolsas.consultar','bolsa.rrhh.estadisticas.consultar','bolsa.rrhh.candidatos.consultar')
    OR c->>'operacion' IS DISTINCT FROM p_accion
    OR d->>'accion' IS DISTINCT FROM p_accion
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 THEN RAISE EXCEPTION 'AD213: consulta denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'consulta_rrhh_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD213: consulta requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=f AND a.grantee=0)
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD213: ACL incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
