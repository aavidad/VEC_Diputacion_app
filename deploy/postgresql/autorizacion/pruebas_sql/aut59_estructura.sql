\set ON_ERROR_STOP on
-- AUT59, sólo en un clon con AUT51 publicada como Rol7. La migración AUT59
-- debe estar instalada, pero el mantenimiento Rol7→Rol8 aún no ejecutado.
-- Todas las comprobaciones terminan en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $vector$
DECLARE
  f text;
  p record;
  c jsonb;
  esperado jsonb := '[{"accion":"administracion.perfiles.definicion.proponer","modulo_id":"administracion","tipo_recurso":"definicion_rol","finalidades":["gobierno_definiciones_perfiles"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]},{"accion":"administracion.perfiles.definicion.aprobar","modulo_id":"administracion","tipo_recurso":"propuesta_definicion_rol","finalidades":["gobierno_definiciones_perfiles"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]}]'::jsonb;
  grupo oid := to_regrole('vec_admin_mantenimiento_gobierno_definiciones_ejecutor');
BEGIN
  IF grupo IS NULL THEN RAISE EXCEPTION 'AUT59: grupo ejecutor ausente'; END IF;
  IF EXISTS(SELECT 1 FROM pg_roles WHERE oid=grupo AND
    (rolcanlogin OR rolinherit OR rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls OR rolconfig IS NOT NULL))
  THEN RAISE EXCEPTION 'AUT59: grupo ejecutor con atributos excesivos'; END IF;

  c := vec_autorizacion.concesiones_gobierno_definiciones_admin_v1();
  IF c IS DISTINCT FROM esperado THEN
    RAISE EXCEPTION 'AUT59: concesiones fuera del par nominal esperado: %',c;
  END IF;

  -- La instalación de estructura no publica el rol ni mueve los dos punteros.
  IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v8')
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil a
      USING(perfil_activo_ref,asignacion_ref) WHERE a.version_rol_ref='rol:administracion_perfiles:v8')
    OR EXISTS(SELECT 1 FROM vec_autorizacion.config_mantenimiento_gobierno_definiciones_admin_v1)
  THEN RAISE EXCEPTION 'AUT59: estructura produjo efecto o aprobación'; END IF;

  -- Los helpers y las tablas son privados. El único EXECUTE ajeno al
  -- propietario corresponde a la fachada y sólo al grupo ejecutor.
  FOREACH f IN ARRAY ARRAY[
    'vec_autorizacion.concesiones_gobierno_definiciones_admin_v1()',
    'vec_autorizacion.exigir_operador_mantenimiento_gobierno_definiciones_admin_v1()',
    'vec_autorizacion.preimagen_mantenimiento_gobierno_definiciones_admin_v1(jsonb)',
    'vec_autorizacion.documento_asignacion_destino_gobierno_definiciones_v1(jsonb,jsonb,text)',
    'vec_autorizacion.aplicar_mantenimiento_gobierno_definiciones_admin_v1(text,text)',
    'vec_autorizacion.mantener_version_perfil_fijo_gobierno_definiciones_admin_v1(text,text)'
  ] LOOP
    SELECT proowner,prosecdef,proconfig,proacl INTO p FROM pg_proc WHERE oid=to_regprocedure(f);
    IF NOT FOUND OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
      OR p.proconfig IS NULL OR NOT p.proconfig @> ARRAY[CASE WHEN p.prosecdef THEN 'search_path=pg_catalog, pg_temp' ELSE 'search_path=pg_catalog' END]
      OR (f LIKE '%exigir_operador%' OR f LIKE '%preimagen%' OR f LIKE '%aplicar%' OR f LIKE '%mantener%') AND p.prosecdef IS NOT TRUE
    THEN RAISE EXCEPTION 'AUT59: owner, SECURITY DEFINER o search_path de %',f; END IF;
    IF EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE x.grantee<>p.proowner AND
      (x.grantee<>grupo OR f<>'vec_autorizacion.mantener_version_perfil_fijo_gobierno_definiciones_admin_v1(text,text)'
       OR x.privilege_type<>'EXECUTE' OR x.is_grantable))
    THEN RAISE EXCEPTION 'AUT59: EXECUTE no previsto en %',f; END IF;
    IF f='vec_autorizacion.mantener_version_perfil_fijo_gobierno_definiciones_admin_v1(text,text)' AND
      NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
        WHERE x.grantee=grupo AND x.privilege_type='EXECUTE' AND NOT x.is_grantable)
    THEN RAISE EXCEPTION 'AUT59: falta EXECUTE exclusivo de fachada'; END IF;
    IF f<>'vec_autorizacion.mantener_version_perfil_fijo_gobierno_definiciones_admin_v1(text,text)'
      AND has_function_privilege(grupo,to_regprocedure(f),'EXECUTE')
    THEN RAISE EXCEPTION 'AUT59: el operador alcanza helper privado %',f; END IF;
  END LOOP;

  IF EXISTS(SELECT 1 FROM pg_class t CROSS JOIN LATERAL aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) x
      WHERE t.oid IN ('vec_autorizacion.config_mantenimiento_gobierno_definiciones_admin_v1'::regclass,
                      'vec_autorizacion.registro_mantenimiento_gobierno_definiciones_admin_v1'::regclass)
      AND x.grantee<>t.relowner)
    OR EXISTS(SELECT 1 FROM pg_class t WHERE t.oid IN
      ('vec_autorizacion.config_mantenimiento_gobierno_definiciones_admin_v1'::regclass,
       'vec_autorizacion.registro_mantenimiento_gobierno_definiciones_admin_v1'::regclass)
       AND (t.relowner<>'vec_autorizacion_propietario'::regrole OR NOT t.relrowsecurity OR NOT t.relforcerowsecurity))
    OR (SELECT count(*) FROM pg_policy WHERE polrelid IN
      ('vec_autorizacion.config_mantenimiento_gobierno_definiciones_admin_v1'::regclass,
       'vec_autorizacion.registro_mantenimiento_gobierno_definiciones_admin_v1'::regclass)
       AND polroles=ARRAY[('vec_autorizacion_propietario'::regrole)::oid])<>2
    OR (SELECT count(*) FROM pg_policy WHERE polrelid IN
      ('vec_autorizacion.config_mantenimiento_gobierno_definiciones_admin_v1'::regclass,
       'vec_autorizacion.registro_mantenimiento_gobierno_definiciones_admin_v1'::regclass))<>2
    OR has_table_privilege(grupo,'vec_autorizacion.config_mantenimiento_gobierno_definiciones_admin_v1','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege(grupo,'vec_autorizacion.registro_mantenimiento_gobierno_definiciones_admin_v1','SELECT,INSERT,UPDATE,DELETE')
  THEN RAISE EXCEPTION 'AUT59: tabla, RLS o ACL no exclusivos del propietario'; END IF;
END $vector$;
-- Un migrador/superusuario sin LOGIN técnico aprobado no puede atravesar el
-- helper de efecto. Esta llamada se detiene antes de consultar la preimagen.
DO $sin_operador$
DECLARE denegado boolean := false;
BEGIN
 BEGIN
  PERFORM vec_autorizacion.aplicar_mantenimiento_gobierno_definiciones_admin_v1('{}',repeat('0',64));
 EXCEPTION WHEN SQLSTATE '42501' THEN denegado := true;
 END;
 IF NOT denegado THEN RAISE EXCEPTION 'AUT59: aplicar sin operador no devolvió 42501'; END IF;
END $sin_operador$;

-- Vector puro AUT45 adaptado al salto v4→v5: fecha de inicio efectiva igual
-- a la publicación, identidad/ámbitos/Hasta conservados y replay estable.
DO $documento$
DECLARE
 original jsonb := $o${"asignacion_id":"ensayo:aut59:aplicacion","version":4,"perfil_activo_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","principal_id":"per_bbbbbbbbbbbbbbbbbbbbbbbb","version_rol_ref":"rol:administracion_perfiles:v7","estado":"activa","ambitos":[{"clave":"organizacion_ref","valores":["org_cccccccccccccccccccccccccccccccc"]},{"clave":"unidad_ref","valores":["unidad_admin_sintetica"]}],"vigente_desde":"2026-10-08T13:30:00.000000Z","vigente_hasta":"2026-10-08T17:00:00.000000Z","emitida_por":"mantenimiento_operador:ensayo","emitida_en":"2026-10-08T13:30:00.000000Z"}$o$;
 plan jsonb := '{"rol_destino_doc":{"publicada_en":"2026-10-08T14:00:00.000000Z"}}'::jsonb;
 esperado jsonb := $e${"asignacion_id":"ensayo:aut59:aplicacion","version":5,"perfil_activo_ref":"prf_aaaaaaaaaaaaaaaaaaaaaaaa","principal_id":"per_bbbbbbbbbbbbbbbbbbbbbbbb","version_rol_ref":"rol:administracion_perfiles:v8","estado":"activa","ambitos":[{"clave":"organizacion_ref","valores":["org_cccccccccccccccccccccccccccccccc"]},{"clave":"unidad_ref","valores":["unidad_admin_sintetica"]}],"vigente_desde":"2026-10-08T14:00:00.000000Z","vigente_hasta":"2026-10-08T17:00:00.000000Z","emitida_por":"mantenimiento_operador:ensayo","emitida_en":"2026-10-08T14:00:00.000000Z"}$e$;
 destino jsonb;
BEGIN
 destino := vec_autorizacion.documento_asignacion_destino_gobierno_definiciones_v1(original,plan,'ensayo');
 IF destino IS DISTINCT FROM esperado
 OR destino->'principal_id' IS DISTINCT FROM original->'principal_id'
 OR destino->'perfil_activo_ref' IS DISTINCT FROM original->'perfil_activo_ref'
 OR destino->'ambitos' IS DISTINCT FROM original->'ambitos'
 OR destino->'vigente_hasta' IS DISTINCT FROM original->'vigente_hasta'
 OR vec_autorizacion.ajustar_inicio_asignacion_mantenimiento_v1(destino) IS DISTINCT FROM destino
 THEN RAISE EXCEPTION 'AUT59: documento destino alteró identidad, ámbitos, Hasta o inicio'; END IF;
END $documento$;

-- Campos nulos e infinito se rechazan antes de leer las fuentes vivas.
DO $fechas$
DECLARE
 base jsonb := jsonb_build_object('version',4,'operacion_ref','pmf_'||repeat('a',22),
   'preparado_en',to_char(date_trunc('second',clock_timestamp()-interval '1 minute') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
   'caduca_en',to_char(date_trunc('second',clock_timestamp()+interval '10 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
   'rol_origen_sha256',repeat('0',64),'control_revision_esperada',1,
   'control_huella_sha256',repeat('0',64),'catalogo_sha256',repeat('0',64),
   'rol_destino_doc','{}'::jsonb,'asignaciones','[{},{}]'::jsonb);
 caso text;
 prueba jsonb;
 denegado boolean;
 estado text;
BEGIN
 -- La forma base pasa la validación de tipos/fechas; las huellas sintéticas
 -- detienen después el cotejo de la fuente real del clon.
 BEGIN
  PERFORM vec_autorizacion.preimagen_mantenimiento_gobierno_definiciones_admin_v1(base);
  estado := 'aceptado';
 EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS estado = RETURNED_SQLSTATE;
 END;
 IF estado='22023' THEN RAISE EXCEPTION 'AUT59: base sintética falló antes del cotejo de fuente'; END IF;
 FOREACH caso IN ARRAY ARRAY['version_nula','preparado_nulo','caduca_nula','preparado_infinito','caduca_infinita'] LOOP
  prueba := CASE caso
   WHEN 'version_nula' THEN jsonb_set(base,'{version}','null'::jsonb)
   WHEN 'preparado_nulo' THEN jsonb_set(base,'{preparado_en}','null'::jsonb)
   WHEN 'caduca_nula' THEN jsonb_set(base,'{caduca_en}','null'::jsonb)
   WHEN 'preparado_infinito' THEN jsonb_set(base,'{preparado_en}','"infinity"'::jsonb)
   ELSE jsonb_set(base,'{caduca_en}','"infinity"'::jsonb)
  END;
  denegado := false;
  BEGIN
   PERFORM vec_autorizacion.preimagen_mantenimiento_gobierno_definiciones_admin_v1(prueba);
  EXCEPTION WHEN SQLSTATE '22023' THEN denegado := true;
  END;
  IF NOT denegado THEN RAISE EXCEPTION 'AUT59: preimagen aceptó %',caso; END IF;
 END LOOP;
END $fechas$;
ROLLBACK;
SELECT 'AUT59-ESTRUCTURA-OK';
