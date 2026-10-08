\set ON_ERROR_STOP on
-- Ensayo sólo en clon PostgreSQL 18 con B87 instalada. Todo se revierte.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='20s';

DO $test$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1(text,text,text,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[],text,integer,integer,boolean,text,date,boolean)');
 v_ref text:='contacto:'||pg_catalog.repeat('e',64);
 v_instante timestamptz:='2026-10-08 10:00:00+00';
BEGIN
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
      AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
      AND p.prosecdef AND p.provolatile='v'
      AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
      CROSS JOIN LATERAL pg_catalog.aclexplode(pg_catalog.coalesce(p.proacl,
       pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=0)
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a
      WHERE a.attrelid='vec_bolsa_llamamientos.contacto_participacion'::pg_catalog.regclass
      AND a.attname='instante_servidor' AND a.atttypid='boolean'::pg_catalog.regtype
      AND a.attnotnull AND NOT a.attisdropped) THEN
  RAISE EXCEPTION 'B87: función, ACL o columna no instalada' USING ERRCODE='55000';
 END IF;

 -- La restricción conserva la nota obligatoria de registros anteriores.
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
   contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,
   resultado,anotacion,clave_idempotencia,recibo_ref)
  VALUES(v_ref,'bolsa:b87','participacion:b87','llamamiento:b87','telefono',v_instante,
   'per_0123456789abcdefghijkl','contactado','','b87-legacy-vacio','recibo:'||v_ref);
  RAISE EXCEPTION 'B87: nota legacy vacía admitida' USING ERRCODE='55000';
 EXCEPTION WHEN check_violation THEN NULL;
 END;

 -- El modo nuevo permite sólo su propia nota vacía y resultado comunica.
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
  contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,
  resultado,anotacion,clave_idempotencia,recibo_ref,instante_servidor)
 VALUES(v_ref,'bolsa:b87','participacion:b87','llamamiento:b87','telefono',v_instante,
  'per_0123456789abcdefghijkl','comunica','','b87-servidor','recibo:'||v_ref,true);

 BEGIN
  INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
   contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,
   resultado,anotacion,clave_idempotencia,recibo_ref)
  VALUES('contacto:'||pg_catalog.repeat('f',64),'bolsa:b87','participacion:b87',
   'llamamiento:b87','telefono',v_instante,'per_0123456789abcdefghijkl',
   'comunica','legacy','b87-legacy-comunica',
   'recibo:contacto:'||pg_catalog.repeat('f',64));
  RAISE EXCEPTION 'B87: comunica legacy admitido' USING ERRCODE='55000';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
END $test$;
ROLLBACK;
