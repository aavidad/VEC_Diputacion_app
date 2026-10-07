\set ON_ERROR_STOP on
-- AD216: sucesora prospectiva de AD214. Sólo acota bolsa_ref en la rama de
-- consulta de candidatos RRHH; conserva acciones, audiencias, ACL e historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000216',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS SHARE MODE;

DO $cambio$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; fuente text; nuevo text; actual text; meta jsonb; deps jsonb; compartidas jsonb;
 check_actual text; def_sha text; src_sha text; n integer;
 esperado_def_pre constant text:='63afb3d4e6f33d4ee8efce1e54d7e8f92ac9f8cec58506c2af146e88dca4552e';
 esperado_src_pre constant text:='78137d8750422597c0da56797fd3ddff797cd54f18d98b0c1c8f4b7f742079dd';
 esperado_check constant text:='8dae0267b85ad237770d0ccdb7fbf9862afe6d7d022c0c93f4385c182bcbc68e';
 esperado_def_post constant text:='96d92f060ddfd4970bf47167c5b23ee140bc67bd521166363c942f6f83203bc3';
 esperado_src_post constant text:='03621298761d25e5e5decb2a8bee13cb2193f9fc07035a6dd7145ea5e5eec1cd';
 marca text:=$m$d->>'recurso_ref' ~ '^bolsa:[A-Za-z0-9:_-]+:filtro:[a-f0-9]{64}$'$m$;
 sustituta text:=$s$(CASE WHEN octet_length(d->>'recurso_ref') BETWEEN 79 AND 200
    THEN d->>'recurso_ref' ~ '^bolsa:[A-Za-z0-9_-]+(:[A-Za-z0-9_-]+)*:filtro:[a-f0-9]{64}$'
    ELSE false END)$s$;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR f IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_historia_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD216: preimagen causal ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta
   FROM pg_proc p WHERE p.oid=f;
 def_sha:=encode(sha256(convert_to(original,'UTF8')),'hex');
 src_sha:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF def_sha IS NOT DISTINCT FROM esperado_def_post AND src_sha IS NOT DISTINCT FROM esperado_src_post THEN
  RAISE EXCEPTION 'AD216 ya instalada: esperado pre def=%/src=%; actual post def=%/src=%',
    esperado_def_pre,esperado_src_pre,def_sha,src_sha USING ERRCODE='55000';
 END IF;
 IF def_sha IS DISTINCT FROM esperado_def_pre OR src_sha IS DISTINCT FROM esperado_src_pre THEN
  RAISE EXCEPTION 'AD216: preimagen incompatible; esperado def=%/src=% actual def=%/src=%',
    esperado_def_pre,esperado_src_pre,def_sha,src_sha USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT check_actual FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(check_actual,'UTF8')),'hex') IS DISTINCT FROM esperado_check
    OR strpos(check_actual,'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1')=0
 THEN RAISE EXCEPTION 'AD216: CHECK de audiencias incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
      AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
      AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
       aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
       aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner
         AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR strpos(original,'resolver_origen_consumo_v1')=0
    OR strpos(original,'transaccion_origen')=0
    OR strpos(original,'consumo_confirmado_v4')=0
 THEN RAISE EXCEPTION 'AD216: autoridad, ACL u origen V3 divergentes' USING ERRCODE='55000'; END IF;
 n:=(length(original)-length(replace(original,marca,'')))/length(marca);
 IF n<>1 OR (length(fuente)-length(replace(fuente,marca,'')))/length(marca)<>1
    OR strpos(original,sustituta)<>0
 THEN RAISE EXCEPTION 'AD216: marca de candidato ausente, duplicada o ya modificada' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
   INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
    AND d.classid='pg_proc'::regclass AND d.objid=f;
 nuevo:=replace(original,marca,sustituta);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,sustituta,marca) IS DISTINCT FROM original
    OR encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM esperado_def_post
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM esperado_src_post
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
         AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') FROM pg_constraint c
        WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
         AND c.conname='clave_capacidad_version_audiencia_consumo_check') IS DISTINCT FROM esperado_check
 THEN RAISE EXCEPTION 'AD216: postimagen alteró contrato ajeno' USING ERRCODE='55000'; END IF;
END $cambio$;
COMMIT;
