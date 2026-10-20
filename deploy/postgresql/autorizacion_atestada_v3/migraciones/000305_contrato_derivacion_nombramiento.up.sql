\set ON_ERROR_STOP on
-- AD305 / F1-19: corrección de la decisión nominal de derivación.
-- Preimágenes extraídas de la copia PG18 con AD301 y AD292 instaladas.
-- Solo retira el predicado `ambitos` de la rama CT ya publicada por AD291/AD301
-- y del wrapper AD292; no crea audiencia, permiso, función ni CHECK nuevos.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS SHARE MODE;
DO $ad305$
DECLARE
 central regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fachada regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 definicion_central text; definicion_fachada text; cuerpo_central text; cuerpo_fachada text;
 metadatos_antes jsonb; metadatos_despues jsonb; check_antes text;
 marca_central text:=$marca$
 AND jsonb_typeof(d->'ambitos') IS NOT DISTINCT FROM 'object'
 AND (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(d->'ambitos') k) IS NOT DISTINCT FROM ARRAY['organizacion_ref','unidad_ref']
$marca$;
 marca_fachada text:=$marca$
 OR jsonb_typeof(d->'ambitos') IS DISTINCT FROM 'object'
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(d->'ambitos') k)
    IS DISTINCT FROM ARRAY['organizacion_ref','unidad_ref']
$marca$;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR central IS NULL OR fachada IS NULL
 THEN RAISE EXCEPTION 'AD305: preimagen PG18/funciones ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(p.oid),p.prosrc INTO definicion_central,cuerpo_central
 FROM pg_proc p WHERE p.oid=central;
 SELECT pg_get_functiondef(p.oid),p.prosrc INTO definicion_fachada,cuerpo_fachada
 FROM pg_proc p WHERE p.oid=fachada;
 SELECT pg_get_constraintdef(k.oid,true) INTO check_antes FROM pg_constraint k
 WHERE k.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND k.conname='clave_capacidad_version_audiencia_consumo_check' AND k.convalidated;
 IF encode(sha256(convert_to(definicion_central,'UTF8')),'hex') IS DISTINCT FROM
   '49d6559e7aa6e5575c630531cf5c2142a67677e5cf7d3d38da3f04cab7253035'
 OR encode(sha256(convert_to(definicion_fachada,'UTF8')),'hex') IS DISTINCT FROM
   '9d1e4fc89863e7fa8a99e1353826be997107571f746c6d3a32b7f29f13b1a1d1'
 OR encode(sha256(convert_to(check_antes,'UTF8')),'hex') IS DISTINCT FROM
   'b8a3f4abaebb452c245e2e33e191f776cea16a3e393ca11ae6ddf8c1901f10c8'
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=central AND p.prosecdef AND p.provolatile='v'
   AND p.proowner=current_user::regrole
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=fachada AND p.prosecdef AND p.provolatile='v'
   AND p.proowner=current_user::regrole
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 OR length(definicion_central)-length(replace(definicion_central,marca_central,''))<>length(marca_central)
 OR length(definicion_fachada)-length(replace(definicion_fachada,marca_fachada,''))<>length(marca_fachada)
 THEN RAISE EXCEPTION 'AD305: preimagen, ACL o marca no acreditadas' USING ERRCODE='55000'; END IF;
 -- Misma identidad de funciones, ACL, parámetros, dependencias y propietarios.
 SELECT jsonb_agg(jsonb_build_object('proc',to_jsonb(p)-'prosrc',
  'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid),
  'shared',(SELECT coalesce(jsonb_agg(to_jsonb(s) ORDER BY s.dbid,s.classid,s.objid,s.objsubid,s.refclassid,s.refobjid,s.deptype),'[]'::jsonb)
   FROM pg_shdepend s WHERE s.classid='pg_proc'::regclass AND s.objid=p.oid
    AND s.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()))) ORDER BY p.oid)
 INTO metadatos_antes FROM pg_proc p WHERE p.oid IN(central,fachada);
 EXECUTE replace(definicion_central,marca_central,E'\n');
 EXECUTE replace(definicion_fachada,marca_fachada,E'\n');
 SELECT jsonb_agg(jsonb_build_object('proc',to_jsonb(p)-'prosrc',
  'deps',(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid),
  'shared',(SELECT coalesce(jsonb_agg(to_jsonb(s) ORDER BY s.dbid,s.classid,s.objid,s.objsubid,s.refclassid,s.refobjid,s.deptype),'[]'::jsonb)
   FROM pg_shdepend s WHERE s.classid='pg_proc'::regclass AND s.objid=p.oid
    AND s.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()))) ORDER BY p.oid)
 INTO metadatos_despues FROM pg_proc p WHERE p.oid IN(central,fachada);
 IF metadatos_despues IS DISTINCT FROM metadatos_antes
 OR (SELECT p.prosrc FROM pg_proc p WHERE p.oid=central) IS DISTINCT FROM replace(cuerpo_central,marca_central,E'\n')
 OR (SELECT p.prosrc FROM pg_proc p WHERE p.oid=fachada) IS DISTINCT FROM replace(cuerpo_fachada,marca_fachada,E'\n')
 OR encode(sha256(convert_to(pg_get_functiondef(central),'UTF8')),'hex') IS DISTINCT FROM
   '58c98915e85aec4ff54d45864df4f733cd013a5a2be5863482cfce96fc0c0c46'
 OR encode(sha256(convert_to(pg_get_functiondef(fachada),'UTF8')),'hex') IS DISTINCT FROM
   '2f00e33d4778828eaf25ef2c8d48eabef7d846de60bc86ccb4c885253188b3b5'
 OR (SELECT pg_get_constraintdef(k.oid,true) FROM pg_constraint k
   WHERE k.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND k.conname='clave_capacidad_version_audiencia_consumo_check') IS DISTINCT FROM check_antes
 THEN RAISE EXCEPTION 'AD305: postimagen distinta de las dos retiradas' USING ERRCODE='55000'; END IF;
END $ad305$;
COMMIT;
