\set ON_ERROR_STOP on
-- AD304: añade el conjunto 7 para proponer y cerrar el gobierno de un rol nuevo.
-- Requiere AD198, AD248 y la audiencia de AD220/AUT60 ya admitida por el CHECK.
-- No publica claves ni realiza actos de gobierno. Una sola aplicación; sin DOWN.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000304',0));
DO $base$ BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD304: PARO clave=dependencias esperado=PG18_AD198_AD220' USING ERRCODE='55000'; END IF;
END $base$;
LOCK TABLE vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 IN SHARE ROW EXCLUSIVE MODE;
DO $pre$
DECLARE seis record; c record; def text;
BEGIN
 SELECT * INTO seis FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=6;
 SELECT * INTO c FROM pg_class WHERE oid='vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1'::regclass;
 SELECT pg_get_constraintdef(k.oid,false) INTO def FROM pg_constraint k
 WHERE k.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND k.conname='clave_capacidad_version_audiencia_consumo_check' AND k.contype='c' AND k.convalidated;
 IF seis.audiencias IS DISTINCT FROM ARRAY[
  'vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1',
  'vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
  'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1',
  'vec_personal.cargo_competencial.publicar.v1',
  'vec_contexto_actor.certificado_nominal.publicar.v1',
  'vec_autorizacion.versionar_rol_bolsa.propuesta.v1',
  'vec_autorizacion.versionar_rol_bolsa.cierre.v1',
  'vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1',
  'vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1']::text[]
 OR seis.segmentos IS DISTINCT FROM ARRAY[
  'usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma',
  'personal:cargo-competencial','certificados:nominal',
  'perfiles:version-bolsa:propuesta','perfiles:version-bolsa:cierre',
  'perfiles:version-bolsa-externa:propuesta','perfiles:version-bolsa-externa:cierre']::text[]
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)<>6
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version>=7)
 OR c.relowner IS DISTINCT FROM to_regrole('vec_autorizacion_atestada_v3_propietario')
 OR c.relrowsecurity OR c.relforcerowsecurity
 OR EXISTS(SELECT 1 FROM aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE a.grantee<>c.relowner)
 OR (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid=c.oid AND NOT t.tgisinternal)<>2
 OR (SELECT count(*) FROM pg_trigger t WHERE t.tgrelid=c.oid AND t.tgenabled='O'
     AND t.tgname IN('inmutable','no_truncar'))<>2
 OR def IS NULL OR strpos(def,'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1')=0
 OR strpos(def,'vec_autorizacion.gobierno_rol_nuevo.cierre.v1')=0
 THEN RAISE EXCEPTION 'AD304: PARO clave=preimagen esperado=conjunto_6_cerrado' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec.ad304.filas',
  (SELECT encode(sha256(convert_to(jsonb_agg(to_jsonb(s) ORDER BY s.version)::text,'UTF8')),'hex')
   FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 s),true);
 PERFORM set_config('vec.ad304.meta',
  (SELECT jsonb_build_object('owner',x.relowner,'acl',x.relacl,'rls',x.relrowsecurity,
   'force_rls',x.relforcerowsecurity,'kind',x.relkind,'persistence',x.relpersistence,
   'options',x.reloptions,'triggers',(SELECT jsonb_agg(jsonb_build_object(
    'name',t.tgname,'enabled',t.tgenabled,'definition',pg_get_triggerdef(t.oid)) ORDER BY t.tgname)
    FROM pg_trigger t WHERE t.tgrelid=x.oid AND NOT t.tgisinternal))::text
   FROM pg_class x WHERE x.oid=c.oid),true);
 PERFORM set_config('vec.ad304.check',def,true);
END $pre$;
INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos)
SELECT 7,audiencias||ARRAY[
 'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1',
 'vec_autorizacion.gobierno_rol_nuevo.cierre.v1']::text[],
 segmentos||ARRAY['perfiles:gobierno-rol-nuevo:propuesta',
                  'perfiles:gobierno-rol-nuevo:cierre']::text[]
FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=6;
DO $post$
DECLARE siete record; seis record; tabla oid:='vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1'::regclass;
BEGIN
 SELECT * INTO seis FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=6;
 SELECT * INTO siete FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=7;
 IF NOT FOUND OR cardinality(siete.audiencias)<>12 OR cardinality(siete.segmentos)<>12
 OR siete.audiencias IS DISTINCT FROM seis.audiencias||ARRAY[
  'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1','vec_autorizacion.gobierno_rol_nuevo.cierre.v1']::text[]
 OR siete.segmentos IS DISTINCT FROM seis.segmentos||ARRAY[
  'perfiles:gobierno-rol-nuevo:propuesta','perfiles:gobierno-rol-nuevo:cierre']::text[]
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)<>7
 OR current_setting('vec.ad304.filas') IS DISTINCT FROM
  (SELECT encode(sha256(convert_to(jsonb_agg(to_jsonb(s) ORDER BY s.version)::text,'UTF8')),'hex')
   FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 s WHERE s.version<=6)
 OR current_setting('vec.ad304.meta')::jsonb IS DISTINCT FROM
  (SELECT jsonb_build_object('owner',x.relowner,'acl',x.relacl,'rls',x.relrowsecurity,
   'force_rls',x.relforcerowsecurity,'kind',x.relkind,'persistence',x.relpersistence,
   'options',x.reloptions,'triggers',(SELECT jsonb_agg(jsonb_build_object(
    'name',t.tgname,'enabled',t.tgenabled,'definition',pg_get_triggerdef(t.oid)) ORDER BY t.tgname)
    FROM pg_trigger t WHERE t.tgrelid=x.oid AND NOT t.tgisinternal))
   FROM pg_class x WHERE x.oid=tabla)
 OR current_setting('vec.ad304.check') IS DISTINCT FROM
  (SELECT pg_get_constraintdef(k.oid,false) FROM pg_constraint k
   WHERE k.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND k.conname='clave_capacidad_version_audiencia_consumo_check')
 THEN RAISE EXCEPTION 'AD304: PARO clave=postimagen esperado=solo_fila_7' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
