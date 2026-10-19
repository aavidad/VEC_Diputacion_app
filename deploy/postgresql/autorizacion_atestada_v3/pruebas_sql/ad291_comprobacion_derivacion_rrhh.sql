\set ON_ERROR_STOP on
-- Ejecutar sólo tras instalar AD291 en una copia desechable.
-- No emite material V3 ni altera filas de negocio.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog,pg_temp;
CREATE FUNCTION pg_temp.exigir(caso text,ok boolean) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN IF ok IS NOT TRUE THEN RAISE EXCEPTION 'AD291 prueba: %',caso; END IF; END $f$;
SELECT pg_temp.exigir('núcleo y ACL', EXISTS(
 SELECT 1 FROM pg_proc p
 WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.prosecdef AND p.provolatile='v'
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')=
   '6e07b8c0690439b3071a0d82fbb230d936e5848faddd84b59c6c8acc08292bd0'));
SELECT pg_temp.exigir('sesión documental RRHH', EXISTS(
 SELECT 1 FROM pg_proc p
 WHERE p.oid='vec_autorizacion.revalidar_sesion_documental_bolsa_rrhh_v1(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure
  AND p.prosecdef AND p.provolatile='v'
  AND p.proowner='vec_autorizacion_propietario'::regrole
  AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}'
  AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')=
   '5264f2cdff2dc8a09fad74fa97de58e5c170bf0310391d5a61be58cf1c734ef7'));
SELECT pg_temp.exigir('audiencias cerradas', (SELECT encode(sha256(convert_to(pg_get_constraintdef(oid,true),'UTF8')),'hex')
 FROM pg_constraint WHERE conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND conname='clave_capacidad_version_audiencia_consumo_check' AND convalidated)=
 '411555a7d3a6e423d1c3538236a7c3da20b0b8b7280a1b07151eaf6cffb2efa4');
SELECT pg_temp.exigir('origen F1-09 único', (SELECT count(*)=1
 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE login_nombre='vec_bolsa_llamamientos_desarrollo'
  AND audiencia_consumo='vec.bolsa.comprobacion_ofrecimiento.v1'
  AND operacion='bolsa.comprobacion_ofrecimiento.registrar'
  AND proceso='vec-server' AND canal_permitido='interna_corporativa'));
ROLLBACK;
