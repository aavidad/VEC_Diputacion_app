\set ON_ERROR_STOP on
DO $acl$
BEGIN
 IF has_table_privilege('vec_personal_ejecutor','vec_personal.importacion_organizacion_revision','SELECT')
    OR has_table_privilege('vec_personal_ejecutor','vec_personal.importacion_organizacion_recibo','INSERT')
    OR NOT has_function_privilege('vec_personal_ejecutor','vec_personal.ejecutar_importacion_organizacion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_personal.importacion_organizacion_revision'::regclass) THEN
  RAISE EXCEPTION 'ACL importación B3 incorrecta';
 END IF;
END $acl$;
SET SESSION AUTHORIZATION vec_prueba_personal;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
DO $casos$
DECLARE mf jsonb; hechos jsonb; m text; d jsonb; c jsonb; h text; canon text; r jsonb; lote text; v_ref text; v_fase text; denegada boolean:=false;
BEGIN
 mf:=jsonb_build_object('organismo_ref','org:prueba','tipo','rpt',
  'version_ref','22222222-2222-4222-8222-222222222222','version_revision',1,
  'fuente_ref','fuente:prueba','fuente_version','v1','fuente_huella_sha256',repeat('a',64),
  'catalogo_unidades',jsonb_build_object('id','estructura-organizativa-dipgra','version',1,'revision',1,'huella_sha256',repeat('b',64)),
  'catalogo_clasificaciones',jsonb_build_object('id','clasificacion:prueba','version',1,'revision',1,'huella_sha256',repeat('c',64)));
 hechos:=jsonb_build_array(jsonb_build_object('clase','nodo','hecho_ref','33333333-3333-4333-8333-333333333333',
  'revision',1,'fila_fuente_ref','fila:uno','organismo_ref','org:prueba','unidad_ref','unidad:uno',
  'vigente_desde','2024-01-01','catalogo_entrada_clave','unidad:uno','tipo_unidad','centro',
  'denominacion','Centro sintético'));
 v_fase:='preparar'; v_ref:='org:prueba';
 m:=jsonb_build_object('esquema','vec.personal.importacion-organizacion.v1','fase',v_fase,
  'lote_ref','','revision_esperada',0,'clave_idempotencia','aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  'correlacion_ref','corr:uno','manifiesto',mf,'hechos',hechos,'decisiones','[]'::jsonb,
  'revisor_actor_ref','','actor_ref','actor:uno','contexto_ref','contexto:uno',
  'perfil_ref','perfil:uno','persona_version',1,'perfil_version',1)::text;
 canon:='{"ambitos":{"organismo_ref":"org:prueba"},"atributos":{"fase":"'||v_fase||'","fuente_ref":"fuente:prueba","lote_ref":"'||v_ref||'","material_sha256":"'||encode(sha256(convert_to(m,'UTF8')),'hex')||'"}}';
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 d:=jsonb_build_object('decision_ref','decision:preparar','principal_id','actor:uno','perfil_activo_ref','perfil:uno',
  'concedida',true,'valida_hasta','2099-01-01T00:00:00.000000Z','accion','personal.organizacion_historica.preparar','modulo_id','personal',
  'tipo_recurso','importacion_organizacion_historica','finalidad','preparar_organizacion_historica',
  'recurso_ref',v_ref,'contexto_recurso_huella_sha256',h,
  'campos_permitidos','["hechos","manifiesto","recibo"]'::jsonb,'obligaciones','[]'::jsonb);
 c:=jsonb_build_object('audiencia_consumo','vec_personal.organizacion_historica.importar.v1',
  'operacion','personal.organizacion_historica.preparar','efecto_ref',v_ref,'huella_efecto_sha256',h);
 r:=vec_personal.ejecutar_importacion_organizacion_v1(m,NULL,convert_to(c::text,'UTF8'),
  convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF r->>'estado'<>'preparacion_no_autoritativa' OR r->>'revision_nueva'<>'1' OR r->>'replay'<>'false' THEN
  RAISE EXCEPTION 'preparación inesperada: %',r;
 END IF;
 lote:=r->>'lote_ref';
 r:=vec_personal.ejecutar_importacion_organizacion_v1(m,NULL,convert_to(c::text,'UTF8'),
  convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF r->>'replay'<>'true' OR r->>'lote_ref'<>lote THEN RAISE EXCEPTION 'replay divergente'; END IF;
 v_fase:='conciliar'; v_ref:=lote;
 m:=jsonb_build_object('esquema','vec.personal.importacion-organizacion.v1','fase',v_fase,
  'lote_ref',lote,'revision_esperada',1,'clave_idempotencia','bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',
  'correlacion_ref','corr:dos','manifiesto',mf,'hechos','[]'::jsonb,
  'decisiones',jsonb_build_array(jsonb_build_object('fila_fuente_ref','fila:uno','clase','unidad',
   'destino_ref','unidad:uno','resultado','vinculada','motivo','cotejo sintético','evidencia_ref','evidencia:uno')),
  'revisor_actor_ref','','actor_ref','actor:uno','contexto_ref','contexto:uno',
  'perfil_ref','perfil:uno','persona_version',1,'perfil_version',1)::text;
 canon:='{"ambitos":{"organismo_ref":"org:prueba"},"atributos":{"fase":"'||v_fase||'","fuente_ref":"fuente:prueba","lote_ref":"'||v_ref||'","material_sha256":"'||encode(sha256(convert_to(m,'UTF8')),'hex')||'"}}';
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 d:=jsonb_set(d,'{decision_ref}','"decision:conciliar"'::jsonb);
 d:=jsonb_set(d,'{accion}','"personal.organizacion_historica.conciliar"'::jsonb);
 d:=jsonb_set(d,'{finalidad}','"conciliar_organizacion_historica"'::jsonb);
 d:=jsonb_set(d,'{recurso_ref}',to_jsonb(v_ref));
 d:=jsonb_set(d,'{contexto_recurso_huella_sha256}',to_jsonb(h));
 d:=jsonb_set(d,'{campos_permitidos}','["decisiones","recibo"]'::jsonb);
 c:=jsonb_set(c,'{operacion}','"personal.organizacion_historica.conciliar"'::jsonb);
 c:=jsonb_set(c,'{efecto_ref}',to_jsonb(v_ref));
 c:=jsonb_set(c,'{huella_efecto_sha256}',to_jsonb(h));
 r:=vec_personal.ejecutar_importacion_organizacion_v1(m,NULL,convert_to(c::text,'UTF8'),
  convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF r->>'revision_nueva'<>'2' OR r->>'estado'<>'conciliada' THEN
  RAISE EXCEPTION 'conciliación inesperada: %',r;
 END IF;
 d:=jsonb_set(d,'{accion}','"personal.organizacion_historica.publicar"'::jsonb);
 d:=jsonb_set(d,'{finalidad}','"publicar_organizacion_historica"'::jsonb);
 BEGIN
  PERFORM vec_personal.ejecutar_importacion_organizacion_v1(
   replace(m,'"conciliar"','"publicar"'),jsonb_build_object('acreditacion_ref','falsa'),
   convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,
   'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true;
 END;
 IF NOT denegada THEN RAISE EXCEPTION 'publicación sin autoridad no denegada'; END IF;
END $casos$;
COMMIT;
RESET SESSION AUTHORIZATION;
DO $post$
BEGIN
 IF (SELECT count(*) FROM vec_personal.importacion_organizacion_revision)<>2
    OR (SELECT count(*) FROM vec_personal.importacion_organizacion_recibo)<>2
    OR (SELECT count(*) FROM vec_personal.version_rpt_historia WHERE organismo_ref='org:prueba')<>0 THEN
  RAISE EXCEPTION 'importación sintética alteró historia publicada';
 END IF;
END $post$;
