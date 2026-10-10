\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE r jsonb; m jsonb; o jsonb; x jsonb; codigo text;
BEGIN
 r:=vec_autorizacion.resolver_rol_administrable_v1('rol:administracion_perfiles:v5');
 o:=jsonb_build_object(
  'cuenta_ref','cta_'||repeat('a',22),'cuenta_version',1,
  'persona_ref','per_'||repeat('a',22),'persona_version',1,
  'perfil_ref','prf_'||repeat('a',22),'perfil_version',0,
  'vinculo_ref','vca_'||repeat('a',22),'vinculo_version',0,
  'huella_sha256',repeat('b',64),'revision_continuidad',0,
  'procedencia_ref','procedencia:prueba','procedencia_version',1,
  'procedencia_huella_sha256',repeat('c',64),
  'vigente_desde','2026-10-10T00:00:00Z','vigente_hasta','2027-01-01T00:00:00Z');
 m:=jsonb_build_object(
  'esquema','administracion_perfiles_acto_v2',
  'operacion_ref','propuesta_admin:'||repeat('d',32),
  'operacion','otorgar','rol_version_ref',r->>'version_ref',
  'rol_huella_sha256',r->>'huella_sha256',
  'actor_persona_ref','per_'||repeat('e',22),
  'actor_perfil_ref','prf_'||repeat('e',22),
  'asignacion_ref','asignacion:admin-prueba:v1',
  'objetivo',o,'unidad_ref','unidad:prueba',
  'motivo',jsonb_build_object('catalogo_id','motivos_admin','catalogo_version',1,
    'catalogo_huella_sha256',repeat('f',64),'entrada_clave','motivo_prueba'));
 x:=vec_autorizacion.validar_material_acto_admin_v1(m::text,true);
 IF x IS DISTINCT FROM m THEN RAISE EXCEPTION 'AUT74: material Go v2 alterado'; END IF;
 x:=vec_autorizacion.validar_material_acto_admin_v1(jsonb_set(m,'{objetivo}',o||jsonb_build_object('centro_ref','centro:prueba'))::text,true);
 IF x#>>'{objetivo,centro_ref}' IS DISTINCT FROM 'centro:prueba' THEN RAISE EXCEPTION 'AUT74: centro opcional perdido'; END IF;
 BEGIN
  PERFORM vec_autorizacion.validar_material_acto_admin_v1((m||jsonb_build_object('asignacion_ref',''))::text,true);
  RAISE EXCEPTION 'AUT74: asignación vacía admitida';
 EXCEPTION WHEN invalid_parameter_value THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  IF codigo<>'22023' THEN RAISE; END IF;
 END;
 m:=jsonb_build_object(
  'esquema','administracion_perfiles_cierre_v2',
  'operacion_ref','cierre_admin:'||repeat('a',32),
  'propuesta_ref','propuesta_admin:'||repeat('d',32),
  'propuesta_huella_sha256',repeat('b',64),
  'decision','rechazada','actor_persona_ref','per_'||repeat('e',22),
  'actor_perfil_ref','prf_'||repeat('e',22),
  'asignacion_ref','asignacion:admin-prueba:v1',
  'motivo',jsonb_build_object('catalogo_id','motivos_admin','catalogo_version',1,
    'catalogo_huella_sha256',repeat('f',64),'entrada_clave','motivo_prueba'));
 BEGIN
  PERFORM vec_autorizacion.cerrar_propuesta_admin_v1(m::text,
    convert_to('{}','UTF8'),convert_to('{"accion":"administracion.perfiles.rechazar"}','UTF8'),
    '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'AUT74: material sin sello admitido';
 EXCEPTION WHEN insufficient_privilege THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  IF codigo<>'42501' THEN RAISE; END IF;
 END;
END $prueba$;
ROLLBACK;
