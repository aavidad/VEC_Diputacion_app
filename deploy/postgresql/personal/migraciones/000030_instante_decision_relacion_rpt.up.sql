\set ON_ERROR_STOP on
-- Personal30: comparar instantes equivalentes sin reescribir los bytes firmados.
-- Requiere Personal27 instalada; no reaplicarla ni ejecutar DOWN.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000030:instante-decision-rpt',0));
DO $corregir$
DECLARE
 firma constant text:='vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
 f oid:=to_regprocedure(firma);
 esperada constant text:='9e4733bcb994a3234d8261c107aeb4b49a9d04f9544302ac34f39c77c604a560';
 posterior constant text:='dd0b06eb3ecb2a8317054739ad01f6c363c725f477d7f98d345f01d83487a78f';
 original text; nuevo text; fuente_sha text; metadata jsonb; deps jsonb; compartidas jsonb;
 a text; b text; i integer; ocurrencias integer;
 antes1 constant text:=$antes1$ ahora timestamptz(6); cap_desde timestamptz; cap_hasta timestamptz; dec_hasta timestamptz;$antes1$;
 despues1 constant text:=$despues1$ ahora timestamptz(6); cap_desde timestamptz; cap_hasta timestamptz; dec_hasta timestamptz; cap_dec_hasta timestamptz;$despues1$;
 antes2 constant text:=$antes2$  dec_hasta:=(d->>'valida_hasta')::timestamptz;$antes2$;
 despues2 constant text:=$despues2$  dec_hasta:=(d->>'valida_hasta')::timestamptz;
  cap_dec_hasta:=(c->>'decision_valida_hasta')::timestamptz;$despues2$;
 antes3 constant text:=$antes3$ OR dec_hasta IS NULL OR NOT isfinite(dec_hasta)$antes3$;
 despues3 constant text:=$despues3$ OR dec_hasta IS NULL OR NOT isfinite(dec_hasta) OR cap_dec_hasta IS NULL OR NOT isfinite(cap_dec_hasta)$despues3$;
 antes4 constant text:=$antes4$ OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta'$antes4$;
 despues4 constant text:=$despues4$ OR cap_dec_hasta IS DISTINCT FROM dec_hasta$despues4$;
BEGIN
 IF current_user<>'vec_personal_propietario' OR f IS NULL THEN
  RAISE EXCEPTION 'PARO clave=Personal30.funcion, actual=%/%, esperado=true/true',
   current_user='vec_personal_propietario',f IS NOT NULL USING ERRCODE='55000';
 END IF;
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex'),to_jsonb(p)-'prosrc',pg_get_functiondef(p.oid)
 INTO STRICT fuente_sha,metadata,original FROM pg_proc p WHERE p.oid=f;
 IF fuente_sha IS DISTINCT FROM esperada
 OR metadata->>'proowner' IS DISTINCT FROM ('vec_personal_propietario'::regrole::oid)::text
 OR metadata->>'pronargs' IS DISTINCT FROM '11'
 OR metadata->>'prorettype' IS DISTINCT FROM ('jsonb'::regtype::oid)::text
 OR metadata->>'prosecdef' IS DISTINCT FROM 'true'
 OR metadata->>'provolatile' IS DISTINCT FROM 'v'
 OR metadata->>'proparallel' IS DISTINCT FROM 'u'
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proconfig=ARRAY[
  'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=5s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f
  AND (x.grantee NOT IN(p.proowner,'vec_personal_ejecutor'::regrole) OR x.grantor<>p.proowner OR x.privilege_type<>'EXECUTE' OR x.is_grantable)) THEN
  RAISE EXCEPTION 'PARO clave=Personal30.preimagen, actual=%, esperado=%',
   jsonb_build_object('cuerpo_sha256',fuente_sha,
    'propietario',metadata->>'proowner' IS NOT DISTINCT FROM ('vec_personal_propietario'::regrole::oid)::text,
    'firma',metadata->>'pronargs' IS NOT DISTINCT FROM '11' AND metadata->>'prorettype' IS NOT DISTINCT FROM ('jsonb'::regtype::oid)::text,
    'atributos',metadata->>'prosecdef' IS NOT DISTINCT FROM 'true' AND metadata->>'provolatile' IS NOT DISTINCT FROM 'v' AND metadata->>'proparallel' IS NOT DISTINCT FROM 'u',
    'configuracion',EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=5s']),
    'acl',(SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f)=2 AND NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f AND (x.grantee NOT IN(p.proowner,'vec_personal_ejecutor'::regrole) OR x.grantor<>p.proowner OR x.privilege_type<>'EXECUTE' OR x.is_grantable))),
   jsonb_build_object('cuerpo_sha256',esperada,'propietario',true,'firma',true,'atributos',true,'configuracion',true,'acl',true)
   USING ERRCODE='55000';
 END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f;
 nuevo:=original;
 FOR i IN 1..4 LOOP
  a:=(ARRAY[antes1,antes2,antes3,antes4])[i];b:=(ARRAY[despues1,despues2,despues3,despues4])[i];
  ocurrencias:=(length(nuevo)-length(replace(nuevo,a,'')))/length(a);
  IF ocurrencias<>1 THEN
   RAISE EXCEPTION 'PARO clave=Personal30.marca_%, actual=%, esperado=1',i,ocurrencias USING ERRCODE='55000';
  END IF;
  nuevo:=replace(nuevo,a,b);
 END LOOP;
 EXECUTE nuevo;
 SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') INTO STRICT fuente_sha FROM pg_proc p WHERE p.oid=f;
 IF to_regprocedure(firma) IS DISTINCT FROM f OR fuente_sha IS DISTINCT FROM posterior
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas THEN
  RAISE EXCEPTION 'PARO clave=Personal30.postimagen, actual=%, esperado=%',
   jsonb_build_object('cuerpo_sha256',fuente_sha,'oid',to_regprocedure(firma) IS NOT DISTINCT FROM f,
    'metadata',(SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS NOT DISTINCT FROM metadata,
    'dependencias',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS NOT DISTINCT FROM deps,
    'dependencias_compartidas',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS NOT DISTINCT FROM compartidas),
   jsonb_build_object('cuerpo_sha256',posterior,'oid',true,'metadata',true,'dependencias',true,'dependencias_compartidas',true)
   USING ERRCODE='55000';
 END IF;
END $corregir$;
COMMIT;
