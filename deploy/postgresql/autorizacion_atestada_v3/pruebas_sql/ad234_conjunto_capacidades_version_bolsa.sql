\set ON_ERROR_STOP on
-- Pruebas de AD234 (conjunto 5 de capacidades ADMIN) sobre una base con
-- AD198, AD205, AD227 y AD234 instaladas. Todo dentro de una transacción que
-- termina en ROLLBACK: no deja ningún efecto. Ejecutar como superusuario.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE cinco record;cuatro record;e text;
BEGIN
 SELECT * INTO STRICT cinco FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=5;
 SELECT * INTO STRICT cuatro FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=4;
 -- 1. El conjunto 5 es el 4, en el mismo orden, más las dos audiencias B1.
 IF cinco.audiencias[1:6] IS DISTINCT FROM cuatro.audiencias OR cinco.segmentos[1:6] IS DISTINCT FROM cuatro.segmentos
 OR cinco.audiencias[7:8] IS DISTINCT FROM ARRAY['vec_autorizacion.versionar_rol_bolsa.propuesta.v1','vec_autorizacion.versionar_rol_bolsa.cierre.v1']
 OR cinco.segmentos[7:8] IS DISTINCT FROM ARRAY['perfiles:version-bolsa:propuesta','perfiles:version-bolsa:cierre']
 THEN RAISE EXCEPTION 'AD234-PRUEBA 1: conjunto 5 distinto'; END IF;
 -- 2. Cada audiencia del conjunto cabe en la tabla de claves (CHECK de AD227)
 --    y cada tramo produce un clave_id con el formato que exige AD198.
 FOR i IN 1..8 LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
    AND c.conname='clave_capacidad_version_audiencia_consumo_check'
    AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,false),''''||cinco.audiencias[i]||'''::text')>0)
  OR ('clave:capacidad:admin:'||cinco.segmentos[i]||':s1:x') !~ '^clave:capacidad:admin:[a-z0-9:._-]{1,160}$'
  THEN RAISE EXCEPTION 'AD234-PRUEBA 2: audiencia % no admitida', cinco.audiencias[i]; END IF;
 END LOOP;
 -- 3. El catálogo de conjuntos sigue cerrado: sin permisos fuera del
 --    propietario y el grupo operador no lo lee directamente.
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
   WHERE c.oid='vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1'::regclass AND a.grantee<>c.relowner)
 OR pg_catalog.has_table_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1','SELECT')
 THEN RAISE EXCEPTION 'AD234-PRUEBA 3: ACL ampliada'; END IF;
 -- 4. La fila es inmutable: ni el propietario puede cambiarla ni borrarla.
 BEGIN
  SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
  UPDATE vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 SET segmentos=segmentos WHERE version=5;
  RAISE EXCEPTION 'AD234-PRUEBA 4: fila mutable';
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS e=MESSAGE_TEXT;
  IF e LIKE 'AD234-PRUEBA%' THEN RAISE; END IF;
 END;
 RESET ROLE;
 BEGIN
  SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
  DELETE FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=5;
  RAISE EXCEPTION 'AD234-PRUEBA 4: fila borrable';
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS e=MESSAGE_TEXT;
  IF e LIKE 'AD234-PRUEBA%' THEN RAISE; END IF;
 END;
 RESET ROLE;
 -- 5. La preimagen de AD198 del conjunto 5 se calcula y lista sus 8 audiencias.
 IF pg_catalog.jsonb_array_length(vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(5)->'audiencias')<>8
 THEN RAISE EXCEPTION 'AD234-PRUEBA 5: preimagen sin las 8 audiencias'; END IF;
 RAISE NOTICE 'AD234-PRUEBAS-OK';
END $prueba$;
ROLLBACK;
