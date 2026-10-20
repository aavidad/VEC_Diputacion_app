\set ON_ERROR_STOP on
-- Ejecutar solo tras instalar AD301 en una copia desechable. Revierte todo:
-- no emite material V3 ni escribe filas de negocio.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog,pg_temp;
CREATE FUNCTION pg_temp.exigir(caso text,ok boolean) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN IF ok IS NOT TRUE THEN RAISE EXCEPTION 'AD301 prueba: %',caso; END IF; END $f$;
-- AD305 conserva el contrato de publicación y corrige otra rama del núcleo.
SELECT pg_temp.exigir('núcleo y ACL', EXISTS(
 SELECT 1 FROM pg_proc p
 WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.prosecdef AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')=
   '58c98915e85aec4ff54d45864df4f733cd013a5a2be5863482cfce96fc0c0c46'));
SELECT pg_temp.exigir('perfil en las tres ramas', (SELECT count(*)=3 FROM regexp_matches(
 pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
 'publicacion_oficial_bases_rrhh','g')));
SELECT pg_temp.exigir('consumidor solo para convocatorias', EXISTS(
 SELECT 1 FROM pg_proc p
 WHERE p.oid='vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.prosecdef AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_convocatorias_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 AND NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_bolsa_llamamientos_propietario',
  'vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'));
SELECT pg_temp.exigir('convocatorias alcanza solo el consumidor', (SELECT count(*)=1
 FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
  AND has_function_privilege('vec_bolsa_convocatorias_propietario',p.oid,'EXECUTE'))
 AND NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_autorizacion_atestada_v3'::regnamespace
  AND has_table_privilege('vec_bolsa_convocatorias_propietario',c.oid,'SELECT,INSERT,UPDATE,DELETE')));
SELECT pg_temp.exigir('audiencia admitida en la restricción', (SELECT
 encode(sha256(convert_to(pg_get_constraintdef(oid,true),'UTF8')),'hex')='b8a3f4abaebb452c245e2e33e191f776cea16a3e393ca11ae6ddf8c1901f10c8'
 AND pg_get_constraintdef(oid,true) LIKE '%''vec.bolsa.publicacion_oficial_bases.v1''::text%'
 AND pg_get_constraintdef(oid,true) LIKE '%''vec.bolsa.comprobacion_ofrecimiento.v1''::text%'
 FROM pg_constraint WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND conname='clave_capacidad_version_audiencia_consumo_check' AND convalidated));
SELECT pg_temp.exigir('origen único del pool de Bolsa', (SELECT count(*)=1
 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE audiencia_consumo='vec.bolsa.publicacion_oficial_bases.v1')
 AND (SELECT count(*)=1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE login_nombre='vec_bolsa_llamamientos_desarrollo'
  AND audiencia_consumo='vec.bolsa.publicacion_oficial_bases.v1'
  AND operacion='bolsa.convocatoria.publicacion_oficial.declarar'
  AND proceso='vec-server' AND canal_permitido='interna_corporativa'));
-- Un material que no es la capacidad de esta audiencia se deniega antes del núcleo.
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $d$ DECLARE vacio bytea:='\x7b7d'; m bytea;
BEGIN
 FOREACH m IN ARRAY ARRAY[vacio,'\xff'::bytea,convert_to('{"operacion":"bolsa.comprobacion_ofrecimiento.registrar","audiencia_consumo":"vec.bolsa.comprobacion_ofrecimiento.v1"}','UTF8')] LOOP
  BEGIN
   PERFORM vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(m,vacio,vacio,vacio,1,1,vacio,vacio,vacio,vacio);
   RAISE EXCEPTION 'AD301 prueba: material ajeno admitido';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
END $d$;
RESET ROLE;
ROLLBACK;
