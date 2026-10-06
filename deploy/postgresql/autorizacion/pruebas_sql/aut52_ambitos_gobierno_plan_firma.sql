\set ON_ERROR_STOP on
-- AUT52 en un clon con un administrador de Aplicación asignado (Rol7). Sólo
-- lectura, en ROLLBACK. Toma la primera asignación actual del rol
-- administracion_perfiles y comprueba que se acreditan exactamente su persona,
-- su organización y su unidad.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
DO $p$
DECLARE a record; org text; unidad text; ok boolean;
BEGIN
 SELECT x.asignacion_ref,x.version_rol_ref,x.principal_id,x.documento INTO STRICT a
 FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE x.version_rol_ref LIKE 'rol:administracion_perfiles:v%' AND x.documento->>'estado'='activa'
 ORDER BY x.asignacion_ref LIMIT 1;
 SELECT e->'valores'->>0 INTO STRICT org FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='organizacion_ref';
 SELECT e->'valores'->>0 INTO STRICT unidad FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='unidad_ref';
 IF vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(a.version_rol_ref,a.asignacion_ref,a.principal_id,org,unidad) IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT52 prueba: los ámbitos de la asignación no se acreditan'; END IF;
 IF vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(a.version_rol_ref,a.asignacion_ref,a.principal_id,org,'unidad:otra')
 OR vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(a.version_rol_ref,a.asignacion_ref,a.principal_id,'org_0000000000000000ffff',unidad)
 OR vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(a.version_rol_ref,a.asignacion_ref,'per_otra_persona_000000000',org,unidad)
 OR vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1('rol:administracion_perfiles:v1',a.asignacion_ref,a.principal_id,org,unidad)
 OR vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(a.version_rol_ref,'asignacion:ajena',a.principal_id,org,unidad)
 OR vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(a.version_rol_ref,a.asignacion_ref,a.principal_id,org,'u"x') THEN
  RAISE EXCEPTION 'AUT52 prueba: se acreditó un ámbito, persona, versión o asignación ajenos'; END IF;
 -- Fuera de una transacción serializable de escritura no acredita.
 RAISE NOTICE 'AUT52 prueba OK (% · %)',a.version_rol_ref,left(a.asignacion_ref,24);
END $p$;
ROLLBACK;
BEGIN ISOLATION LEVEL READ COMMITTED;
DO $q$ BEGIN
 IF vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1('rol:administracion_perfiles:v7','x','y','org_0123456789abcdef0123','unidad:x') THEN
  RAISE EXCEPTION 'AUT52 prueba: acreditó fuera de SERIALIZABLE'; END IF;
END $q$;
ROLLBACK;
\echo AUT52_PRUEBA_OK
