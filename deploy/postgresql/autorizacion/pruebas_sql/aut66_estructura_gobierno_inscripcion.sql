\set ON_ERROR_STOP on
-- Sonda estructural de AUT66. Ejecutar tras su UP en copia aislada PG18.
-- No publica definiciones, concesiones, perfiles ni asignaciones.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE f oid;tabla oid;fallo boolean:=false;acciones text[];concesion jsonb;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regclass('vec_autorizacion.propuesta_version_inscripcion_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.cierre_version_inscripcion_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.outbox_version_inscripcion_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.canon_version_inscripcion_v1(jsonb,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.concesion_inscripcion_exacta_v1(jsonb,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.validar_plan_version_inscripcion_v1(jsonb,boolean)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_version_inscripcion_v1(jsonb)') IS NULL
 OR EXISTS(SELECT 1 FROM vec_autorizacion.propuesta_version_inscripcion_v1)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.cierre_version_inscripcion_v1)
 OR EXISTS(SELECT 1 FROM vec_autorizacion.outbox_version_inscripcion_v1)
 THEN RAISE EXCEPTION 'AUT66 prueba: estructura ausente o con efectos';END IF;
 IF vec_autorizacion.acreditar_version_inscripcion_v1('{}'::jsonb) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT66 prueba: acreditador abierto';END IF;
 acciones:=vec_autorizacion.acciones_perfil_inscripcion_v1('empleado');
 IF pg_catalog.cardinality(acciones)<>5 OR 'bolsa.inscripcion.presentar'<>ALL(acciones)
 OR vec_autorizacion.acciones_perfil_inscripcion_v1('desconocido') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT66 prueba: allowlist de empleado divergente';END IF;
 acciones:=vec_autorizacion.acciones_perfil_inscripcion_v1('rrhh');
 IF pg_catalog.cardinality(acciones)<>6
 OR acciones[1] IS DISTINCT FROM 'bolsa.inscripcion.rrhh.convocatorias.listar'
 THEN RAISE EXCEPTION 'AUT66 prueba: selector RRHH ausente o desordenado';END IF;
 concesion:=$j${"accion":"bolsa.inscripcion.presentar","modulo_id":"bolsa",
  "tipo_recurso":"inscripcion_convocatoria","finalidades":["presentar_inscripcion"],
  "garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]}$j$::jsonb;
 IF vec_autorizacion.concesion_inscripcion_exacta_v1(concesion,'externo') IS NOT TRUE
 OR vec_autorizacion.concesion_inscripcion_exacta_v1(concesion,'empleado') IS NOT FALSE
 OR vec_autorizacion.concesion_inscripcion_exacta_v1(
  pg_catalog.jsonb_set(concesion,'{tipo_recurso}',
   '"inscripcion_convocatoria_empleado"'::jsonb),'empleado') IS NOT TRUE
 OR vec_autorizacion.concesion_inscripcion_exacta_v1(
  pg_catalog.jsonb_set(concesion,'{campos_permitidos}',
   '["datos_personales"]'::jsonb),'externo') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT66 prueba: tipo externo/empleado mezclado';END IF;
 concesion:=$j${"accion":"bolsa.inscripcion.convocatorias.listar","modulo_id":"bolsa",
  "tipo_recurso":"convocatoria_inscripcion","finalidades":["consulta_convocatoria_abierta"],
  "garantia_minima":"alto","campos_permitidos":["resumen_inscripcion"],"obligaciones":[]}$j$::jsonb;
 IF vec_autorizacion.concesion_inscripcion_exacta_v1(concesion,'externo') IS NOT TRUE
 OR vec_autorizacion.concesion_inscripcion_exacta_v1(concesion,'empleado') IS NOT FALSE
 OR vec_autorizacion.concesion_inscripcion_exacta_v1(
  pg_catalog.jsonb_set(concesion,'{tipo_recurso}',
   '"convocatoria_inscripcion_empleado"'::jsonb),'empleado') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT66 prueba: lectura externa/empleado mezclada';END IF;
 concesion:=$j${"accion":"bolsa.inscripcion.rrhh.convocatorias.listar","modulo_id":"bolsa",
  "tipo_recurso":"conjunto_gestion_inscripcion",
  "finalidades":["consulta_convocatorias_gestion_rrhh"],"garantia_minima":"alto",
  "campos_permitidos":["convocatorias[].convocatoria_ref","convocatorias[].titulo",
   "convocatorias[].categorias_resumen","convocatorias[].plazo_fin",
   "convocatorias[].estado_publicacion","total","cursor_siguiente"],
  "obligaciones":[]}$j$::jsonb;
 IF vec_autorizacion.concesion_inscripcion_exacta_v1(concesion,'rrhh') IS NOT TRUE
 OR vec_autorizacion.concesion_inscripcion_exacta_v1(concesion,'externo') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT66 prueba: selector RRHH cruzado';END IF;
 BEGIN
  PERFORM vec_autorizacion.canon_version_inscripcion_v1('{}'::jsonb,'plan');
 EXCEPTION WHEN SQLSTATE '22023' THEN fallo:=true;
 END;
 IF NOT fallo THEN RAISE EXCEPTION 'AUT66 prueba: plan vacío admitido';END IF;
 FOR f IN SELECT p.oid FROM pg_catalog.pg_proc p
  WHERE p.pronamespace='vec_autorizacion'::pg_catalog.regnamespace
  AND p.proname IN('canon_version_inscripcion_v1','acciones_perfil_inscripcion_v1',
   'concesion_inscripcion_exacta_v1',
   'validar_plan_version_inscripcion_v1','acreditar_version_inscripcion_v1') LOOP
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,
     pg_catalog.acldefault('f',p.proowner))) acl
    WHERE p.oid=f AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
  THEN RAISE EXCEPTION 'AUT66 prueba: función pública %',f::pg_catalog.regprocedure;END IF;
 END LOOP;
 IF NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
  'vec_autorizacion.acreditar_version_inscripcion_v1(jsonb)','EXECUTE')
 THEN RAISE EXCEPTION 'AUT66 prueba: AD231 sin puerta';END IF;
 FOR tabla IN SELECT pg_catalog.to_regclass('vec_autorizacion.propuesta_version_inscripcion_v1')
  UNION ALL SELECT pg_catalog.to_regclass('vec_autorizacion.cierre_version_inscripcion_v1')
  UNION ALL SELECT pg_catalog.to_regclass('vec_autorizacion.outbox_version_inscripcion_v1') LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=tabla
   AND c.relowner='vec_autorizacion_propietario'::pg_catalog.regrole
   AND c.relrowsecurity AND c.relforcerowsecurity)
   OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_policy p WHERE p.polrelid=tabla
    AND p.polname='propietario_exacto'
    AND p.polroles=ARRAY['vec_autorizacion_propietario'::pg_catalog.regrole]::oid[]
    AND p.polqual IS NOT NULL AND p.polwithcheck IS NOT NULL)
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,
     pg_catalog.acldefault('r',c.relowner))) acl
    WHERE c.oid=tabla AND acl.grantee=0)
   OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger t
       WHERE t.tgrelid=tabla AND NOT t.tgisinternal AND t.tgenabled='O'
       AND t.tgname IN('inmutable','no_truncar'))<>2
  THEN RAISE EXCEPTION 'AUT66 prueba: tabla sin aislamiento %',tabla::pg_catalog.regclass;END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
