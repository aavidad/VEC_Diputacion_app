\set ON_ERROR_STOP on
-- AD3-127. Autoridad nominal de consulta y registro del vínculo prospectivo CT.
-- AD3-117 conserva íntegra su fachada de publicación histórica RPT.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000127',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR has_function_privilege('vec_contratacion_temporal_propietario',
       'vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario'
                   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-127: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
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
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
    OR strpos(original,'vinculo_categoria_rpt_ct')<>0
 THEN RAISE EXCEPTION 'AD3-127: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-127: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; a text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_catalogos_configurables.lectura_categorias.v1')=0
 THEN RAISE EXCEPTION 'AD3-127: audiencia previa incompatible' USING ERRCODE='55000'; END IF;
 FOREACH a IN ARRAY ARRAY['vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1','vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1'] LOOP
  IF strpos(d,quote_literal(a))<>0 THEN RAISE EXCEPTION 'AD3-127: audiencia repetida' USING ERRCODE='55000'; END IF;
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
  ||left(d,length(d)-3)||', ''vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1''::text, ''vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1''::text]))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; x record; material_h text; contexto_h text; accion text;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
 THEN RAISE EXCEPTION 'AD3-127: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-127: material inválido' USING ERRCODE='22023'; END;
 accion:=c->>'operacion';
 material_h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'organizacion_ref')||'"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF jsonb_typeof(s)<>'object'
    OR s->>'organizacion_ref' IS NULL OR s->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'expediente_ref' IS NULL OR s->>'expediente_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR accion NOT IN ('contratacion_temporal.categoria_rpt.vinculo.consultar','contratacion_temporal.categoria_rpt.vinculo.registrar')
    OR c->>'audiencia_consumo' IS DISTINCT FROM (CASE accion
      WHEN 'contratacion_temporal.categoria_rpt.vinculo.consultar' THEN 'vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1'
      ELSE 'vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1' END)
    OR d->>'accion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_categoria_rpt_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_vinculo_categoria_rpt_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->'campos_permitidos' IS DISTINCT FROM (CASE accion
      WHEN 'contratacion_temporal.categoria_rpt.vinculo.consultar' THEN '["analisis","vinculo"]'::jsonb
      ELSE '["recibo"]'::jsonb END)
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR (accion='contratacion_temporal.categoria_rpt.vinculo.consultar') IS DISTINCT FROM (s->>'esquema'='vec.ct.vinculo-categoria-rpt.consulta.v1')
 THEN RAISE EXCEPTION 'AD3-127: autorización divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'vinculo_categoria_rpt_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-127: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
 WHERE p.oid=f AND x.grantee<>p.proowner AND x.grantee<>'vec_contratacion_temporal_propietario'::regrole LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
 END LOOP;
 IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
              WHERE p.oid=f AND x.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole))
 THEN RAISE EXCEPTION 'AD3-127: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
