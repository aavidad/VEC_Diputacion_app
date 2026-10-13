\set ON_ERROR_STOP on
-- Sólo en el clon causal PG18 de dirección; todas las fixtures hacen ROLLBACK.
-- No sustituye autoridades: publicador, validador Usuarios y provisión CA15 reales.
-- Los roles/control son fixtures del registro AUT, no una aprobación ADMIN/V3.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $estructura$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)');
 IF p.proowner IS DISTINCT FROM to_regrole('vec_autorizacion_propietario')
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'::"char"
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')<>
  'd628be56a728fe4ff39f61764eb7c24e543a39a75ea28678ff3cbfe7e9ec2f80'
 OR (SELECT count(*) FROM aclexplode(p.proacl))<>2
 OR EXISTS(SELECT 1 FROM aclexplode(p.proacl) a WHERE
  a.grantor<>p.proowner OR a.grantee NOT IN(p.proowner,to_regrole('vec_autorizacion_publicador_usuarios_externo'))
  OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=
  to_regprocedure('vec_autorizacion.rol_usuarios_externo_acotado_v1(jsonb)')) IS DISTINCT FROM
  '6a32d06608d62cb3caaaba1c15e3600b2ab998602154939821c29279a89d2fc8'
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=
  to_regprocedure('vec_autorizacion.publicador_usuarios_externo_interno_valido_v1()')) IS DISTINCT FROM
  '0180a5752c84875defdc49235e5d4a9cca4d94fc702a9c301214b9cbdcd386a0'
 THEN RAISE EXCEPTION 'AUT81: estructura/ACL divergente';END IF;
END $estructura$;
CREATE ROLE prueba_aut81_publicador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_publicador_usuarios_externo TO prueba_aut81_publicador WITH INHERIT TRUE,SET FALSE;
CREATE TEMP TABLE aut81_roles(caso text PRIMARY KEY,rol_ref text,documento jsonb);
CREATE TEMP TABLE aut81_antes AS SELECT
 (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_externa) AS asignaciones,
 (SELECT count(*) FROM vec_autorizacion.publicacion_candidato_externo_evento) AS eventos,
 (SELECT count(*) FROM vec_autorizacion.asignacion_perfil) AS internas;
GRANT SELECT,INSERT ON TABLE pg_temp.aut81_roles TO vec_autorizacion_propietario;
GRANT SELECT ON TABLE pg_temp.aut81_roles TO prueba_aut81_publicador;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $provision$
DECLARE s jsonb;c jsonb;k text;prefijo text;caso integer;
BEGIN
 FOR caso IN 1..2 LOOP
  s:=jsonb_build_object('provision_ref','pue_aut81_sintetico_0000000000'||caso,
   'poblacion','usuarios','estado','activo','vinculo_candidato',null);
  FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto'] LOOP
   prefijo:=CASE k WHEN 'cuenta' THEN 'cta_' WHEN 'persona' THEN 'per_' WHEN 'perfil' THEN 'prf_' ELSE 'vca_' END;
   c:=jsonb_build_object('referencia',prefijo||'aut81_sintetico_0000000000'||CASE WHEN k='persona' THEN 1 ELSE caso END,
    'version',1,'procedencia_ref','prc_aut81_sintetico_000000000001','procedencia_version',1,
    'procedencia_huella_sha256',repeat('a',64),'procedencia_autoridad','autoridad_maestra_acreditada',
    'estado','activo','vigente_desde','2020-01-01T00:00:00.000000Z','vigente_hasta','2099-01-01T00:00:00.000000Z');
   s:=s||jsonb_build_object(k,c);
  END LOOP;
  PERFORM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(s,0,null,
   vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(s,1));
 END LOOP;
 -- CA15 admite la misma persona con dos cuentas/perfiles/contextos distintos.
 IF NOT vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
  'cta_aut81_sintetico_00000000001','per_aut81_sintetico_00000000001','prf_aut81_sintetico_00000000001')
 OR NOT vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
  'cta_aut81_sintetico_00000000002','per_aut81_sintetico_00000000001','prf_aut81_sintetico_00000000002')
 OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
  'cta_aut81_sintetico_00000000001','per_aut81_sintetico_00000000001','prf_aut81_sintetico_00000000002')
 THEN RAISE EXCEPTION 'AUT81: segregación de provisión divergente';END IF;
END $provision$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $roles$
DECLARE d jsonb;u jsonb;x jsonb;cs jsonb:='[{"accion":"meritos.cartera.consultar_propia","modulo_id":"meritos","tipo_recurso":"cartera_personal","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"],"obligaciones":["auditar"]},{"accion":"meritos.cartera.retirar_propio","modulo_id":"meritos","tipo_recurso":"hecho","finalidades":["gestion_cartera_propia"],"garantia_minima":"alto","campos_permitidos":["disponibilidad","recibo"],"obligaciones":["auditar"]}]'::jsonb;
 accion text;tipo text;finalidad text;campos jsonb;caso integer;ref text;c jsonb;item record;
BEGIN
 -- El RolID M4 conserva su nombre de catálogo, y su autor es distinto del
 -- publicador de asignaciones. Las negativas alteran una dimensión cada vez.
 d:=jsonb_build_object('rol_id','persona_meritos_cartera','version',810000001,
  'nombre','areaPersonal.meritos.rolCartera','estado','publicada','concesiones',cs,
  'publicada_por','admin:aut81:fixture','publicada_en','2026-01-01T00:00:00Z',
  'retirada_en','0001-01-01T00:00:00Z');
 INSERT INTO pg_temp.aut81_roles VALUES('m4','rol:persona_meritos_cartera:v810000001',d);
 FOR caso IN 2..12 LOOP
  x:=jsonb_set(d,'{version}',to_jsonb(810000000+caso));
  x:=CASE caso
   WHEN 2 THEN jsonb_set(x,'{concesiones,0,campos_permitidos}','["denominacion"]'::jsonb)
   WHEN 3 THEN jsonb_set(x,'{concesiones,0,obligaciones}','[]'::jsonb)
   WHEN 4 THEN jsonb_set(x,'{concesiones,0,finalidades}','["finalidad_ajena"]'::jsonb)
   WHEN 5 THEN jsonb_set(x,'{concesiones,0,garantia_minima}','"bajo"'::jsonb)
   WHEN 6 THEN jsonb_set(x,'{concesiones,1,campos_permitidos}','["disponibilidad","recibo","documento"]'::jsonb)
   WHEN 7 THEN jsonb_set(x,'{concesiones}',jsonb_build_array(cs->0))
   WHEN 8 THEN jsonb_set(x,'{concesiones,1,modulo_id}','"usuarios"'::jsonb)
   WHEN 9 THEN jsonb_set(x,'{concesiones,1,tipo_recurso}','"cartera_personal"'::jsonb)
   WHEN 10 THEN jsonb_set(x,'{concesiones,1,obligaciones}','[]'::jsonb)
   WHEN 11 THEN jsonb_set(x,'{concesiones,1,finalidades}','["finalidad_ajena"]'::jsonb)
   WHEN 12 THEN jsonb_set(x,'{concesiones,1,garantia_minima}','"sustancial"'::jsonb) END;
  INSERT INTO pg_temp.aut81_roles VALUES('negativa_'||caso,
   'rol:persona_meritos_cartera:v'||(810000000+caso),x);
 END LOOP;
 u:=jsonb_build_object('rol_id','candidato_usuarios_propios_desarrollo','version',810000001,
  'nombre','areaPersonal.usuarios.rolPropio','estado','publicada','concesiones','[]'::jsonb,
  'publicada_por','seguridad:desarrollo:no-autoritativa','publicada_en','2026-01-01T00:00:00Z',
  'retirada_en','0001-01-01T00:00:00Z');
 FOREACH accion IN ARRAY ARRAY['vec.preferencias.consultar','vec.preferencias.actualizar',
  'vec.imagen.consultar','vec.imagen.actualizar','vec.correos.consultar','vec.correos.anadir',
  'vec.correos.reenviar','vec.correos.verificar','vec.correos.activar','vec.correos.retirar'] LOOP
  tipo:=CASE WHEN accion LIKE 'vec.preferencias.%' THEN 'preferencias_persona'
   WHEN accion LIKE 'vec.imagen.%' THEN 'imagen_persona' ELSE 'correos_persona' END;
  finalidad:=CASE tipo WHEN 'preferencias_persona' THEN 'finalidad:usuarios:preferencias-propias:v1'
   WHEN 'imagen_persona' THEN 'finalidad:usuarios:imagen-propia:v1' ELSE 'finalidad:usuarios:correos-propios:v1' END;
  campos:=CASE accion WHEN 'vec.preferencias.consultar' THEN '["catalogo","valores","version"]'::jsonb
   WHEN 'vec.preferencias.actualizar' THEN '["valores","version"]'::jsonb
   WHEN 'vec.imagen.consultar' THEN '["foto","icono","modo","paleta","version"]'::jsonb
   WHEN 'vec.imagen.actualizar' THEN '["foto","icono","modo","paleta","version"]'::jsonb
   WHEN 'vec.correos.consultar' THEN '["activo","correo_ref","direccion","estado","version"]'::jsonb
   WHEN 'vec.correos.anadir' THEN '["correo_ref","direccion","estado","version"]'::jsonb
   WHEN 'vec.correos.reenviar' THEN '["correo_ref","estado","version"]'::jsonb
   WHEN 'vec.correos.verificar' THEN '["correo_ref","estado","version"]'::jsonb
   WHEN 'vec.correos.activar' THEN '["activo","correo_ref","version"]'::jsonb
   ELSE '["activo","correo_ref","estado","version"]'::jsonb END;
  u:=jsonb_set(u,'{concesiones}',(u->'concesiones')||jsonb_build_array(jsonb_build_object(
   'accion',accion,'modulo_id','usuarios','tipo_recurso',tipo,'finalidades',jsonb_build_array(finalidad),
   'campos_permitidos',campos,'garantia_minima','alto')));
 END LOOP;
 IF vec_autorizacion.rol_usuarios_externo_acotado_v1(u) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT81: fixture Usuarios inválida';END IF;
 INSERT INTO pg_temp.aut81_roles VALUES('usuarios','rol:candidato_usuarios_propios_desarrollo:v810000001',u);
 x:=jsonb_set(jsonb_set(u,'{version}','810000002'::jsonb),'{concesiones}',cs);
 INSERT INTO pg_temp.aut81_roles VALUES('usuarios_con_m4','rol:candidato_usuarios_propios_desarrollo:v810000002',x);
 x:=jsonb_set(d,'{rol_id}','"otro_rol_ordinario"'::jsonb);
 INSERT INTO pg_temp.aut81_roles VALUES('rol_ajeno','rol:otro_rol_ordinario:v810000001',x);
 FOR item IN SELECT * FROM pg_temp.aut81_roles LOOP
  ref:=item.rol_ref;
  INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
  VALUES(ref,item.documento->>'rol_id',(item.documento->>'version')::bigint,
   encode(sha256(convert_to(item.documento::text,'UTF8')),'hex'),'2026-01-01T00:00:00Z',item.documento);
  c:=jsonb_build_object('version_rol_ref',ref,'revision',1,'estado','habilitada',
   'actualizado_por','admin:aut81:fixture','actualizado_en','2026-01-01T00:00:00Z');
  INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
  VALUES(ref,1,'habilitada',encode(sha256(convert_to(c::text,'UTF8')),'hex'),'2026-01-01T00:00:00Z',c);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES(ref,1,clock_timestamp(),'admin:aut81:fixture','acto:aut81:rol:'||item.caso);
 END LOOP;
END $roles$;
RESET ROLE;
SET LOCAL SESSION AUTHORIZATION prueba_aut81_publicador;
DO $publicacion$
DECLARE d jsonb;b bytea;h text;h_m4 text;r record;item record;caso integer;
BEGIN
 FOR caso IN 1..2 LOOP
  d:=jsonb_build_object('asignacion_id','aut81_'||caso,'version',1,
   'perfil_activo_ref','prf_aut81_sintetico_0000000000'||caso,
   'principal_id','per_aut81_sintetico_00000000001',
   'version_rol_ref',CASE caso WHEN 1 THEN 'rol:candidato_usuarios_propios_desarrollo:v810000001'
    ELSE 'rol:persona_meritos_cartera:v810000001' END,
   'estado','activa','ambitos',jsonb_build_array(jsonb_build_object('clave','persona_ref',
    'valores',jsonb_build_array('per_aut81_sintetico_00000000001'))),
   'vigente_desde','2020-01-01T00:00:00Z','vigente_hasta','2099-01-01T00:00:00Z',
   'emitida_por','admin:aut81:fixture','emitida_en','2026-01-01T00:00:00Z',
   'revocada_en','0001-01-01T00:00:00Z');
  b:=convert_to(d::text,'UTF8');h:=encode(sha256(b),'hex');
  SELECT * INTO STRICT r FROM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
   b,h,0,null,'admin:aut81:fixture','acto:aut81:asignacion:'||caso,
   'cta_aut81_sintetico_0000000000'||caso);
  IF r.asignacion_ref<>'asignacion:aut81_'||caso||':v1' OR r.version<>1 OR r.huella_sha256<>h
  THEN RAISE EXCEPTION 'AUT81: recibo de asignación divergente';END IF;
  IF caso=2 THEN h_m4:=h;END IF;
 END LOOP;
 -- Ambas ramas llegan a CAS con su rol aprobado. Repetir el CAS 0 deniega.
 BEGIN
  PERFORM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
   b,h,0,null,'admin:aut81:fixture','acto:aut81:cas-obsoleto','cta_aut81_sintetico_00000000002');
  RAISE EXCEPTION 'AUT81: CAS 0 obsoleto aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL;END;
 d:=jsonb_set(d,'{version}','2'::jsonb);
 b:=convert_to(d::text,'UTF8');h:=encode(sha256(b),'hex');
 BEGIN
  PERFORM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
   b,h,1,repeat('0',64),'admin:aut81:fixture','acto:aut81:cas-huella','cta_aut81_sintetico_00000000002');
  RAISE EXCEPTION 'AUT81: CAS con huella ajena aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL;END;
 FOR item IN SELECT * FROM pg_temp.aut81_roles f WHERE f.caso NOT IN('m4','usuarios') LOOP
  b:=convert_to(jsonb_set(d,'{version_rol_ref}',to_jsonb(item.rol_ref))::text,'UTF8');
  h:=encode(sha256(b),'hex');
  BEGIN
   PERFORM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
    b,h,1,h_m4,'admin:aut81:fixture','acto:aut81:negativa:'||item.caso,'cta_aut81_sintetico_00000000002');
   RAISE EXCEPTION 'AUT81: rol no exacto aceptado: %',item.caso;
  EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 END LOOP;
 b:=convert_to(d::text,'UTF8');h:=encode(sha256(b),'hex');
 BEGIN
  PERFORM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
   b,h,1,h_m4,'admin:aut81:fixture','acto:aut81:cuenta-ajena','cta_aut81_sintetico_00000000001');
  RAISE EXCEPTION 'AUT81: cuenta Usuarios asignó perfil M4';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN
  PERFORM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
   b,repeat('0',64),1,h_m4,'admin:aut81:fixture','acto:aut81:bytes-ajenos','cta_aut81_sintetico_00000000002');
  RAISE EXCEPTION 'AUT81: huella ajena aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 SELECT * INTO STRICT r FROM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(
  b,h,1,h_m4,'admin:aut81:fixture','acto:aut81:asignacion:2:v2','cta_aut81_sintetico_00000000002');
 IF r.asignacion_ref<>'asignacion:aut81_2:v2' OR r.version<>2 OR r.huella_sha256<>h
 THEN RAISE EXCEPTION 'AUT81: CAS vigente no avanzó';END IF;
END $publicacion$;
RESET SESSION AUTHORIZATION;
DO $historia$
DECLARE a record;
BEGIN
 SELECT * INTO STRICT a FROM pg_temp.aut81_antes;
 IF (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_externa)<>a.asignaciones+3
 OR (SELECT count(*) FROM vec_autorizacion.publicacion_candidato_externo_evento)<>a.eventos+3
 OR (SELECT count(*) FROM vec_autorizacion.asignacion_perfil)<>a.internas
 OR (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_id='aut81_2')<>2
 OR (SELECT asignacion_ref FROM vec_autorizacion.asignacion_perfil_actual_externa
  WHERE perfil_activo_ref='prf_aut81_sintetico_00000000002') IS DISTINCT FROM 'asignacion:aut81_2:v2'
 OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_id IN('aut81_1','aut81_2')
  AND documento->>'revocada_en' IS DISTINCT FROM '0001-01-01T00:00:00Z')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.publicacion_candidato_externo_evento
  WHERE acto_ref LIKE 'acto:aut81:%' AND acto_ref NOT IN(
   'acto:aut81:asignacion:1','acto:aut81:asignacion:2','acto:aut81:asignacion:2:v2'))
 OR (SELECT e.huella_previa FROM vec_autorizacion.publicacion_candidato_externo_evento e
  WHERE e.acto_ref='acto:aut81:asignacion:2:v2') IS DISTINCT FROM
  (SELECT e.huella_nueva FROM vec_autorizacion.publicacion_candidato_externo_evento e
   WHERE e.acto_ref='acto:aut81:asignacion:2')
 THEN RAISE EXCEPTION 'AUT81: historia, evento o CAS divergentes';END IF;
END $historia$;
ROLLBACK;
\echo AUT81-ASIGNACION-EXTERNA-OK
