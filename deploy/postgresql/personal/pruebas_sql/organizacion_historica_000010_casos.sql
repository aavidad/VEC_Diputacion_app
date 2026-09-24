\set ON_ERROR_STOP on
SET ROLE vec_personal_propietario;
INSERT INTO vec_personal.org_nodo_historia
 (nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,catalogo_revision,
  catalogo_entrada_clave,denominacion,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('11111111-1111-4111-8111-111111111111',1,'org:prueba','unidad:uno','centro',
 'estructura-organizativa-dipgra',1,1,'unidad:uno','Centro sintético','2024-01-01',
 '2025-01-01 00:00:00+00','fuente:prueba','acto:prueba',repeat('a',64));
INSERT INTO vec_personal.org_nodo_historia
 (nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,catalogo_revision,
  catalogo_entrada_clave,denominacion,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('11111111-1111-4111-8111-111111111111',2,'org:prueba','unidad:uno','centro',
 'estructura-organizativa-dipgra',1,2,'unidad:uno','Centro rectificado','2024-01-01',
 '2026-01-01 00:00:00+00','fuente:prueba','acto:rectificacion',repeat('b',64));
RESET ROLE;
DO $acl$
BEGIN
 IF has_table_privilege('vec_personal_ejecutor','vec_personal.org_nodo_historia','SELECT')
    OR has_table_privilege('vec_personal_ejecutor','vec_personal.recibo_consulta_organizacion','INSERT')
    OR NOT has_function_privilege('vec_personal_ejecutor','vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_personal.org_nodo_historia'::regclass) THEN
   RAISE EXCEPTION 'ACL/RLS B3 incorrecta';
 END IF;
END $acl$;
SET SESSION AUTHORIZATION vec_prueba_personal;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
DO $casos$
DECLARE m text; d jsonb; c jsonb; h text; canon text; r jsonb; denegada boolean:=false;
BEGIN
 m:=jsonb_build_object('esquema','vec.personal.organizacion-historica.v1',
  'organismo_ref','org:prueba','unidad_clave','unidad:uno','vigente_en','2024-12-31',
  'conocido_en','2025-06-01T00:00:00.000000Z','version_rpt_ref','',
  'version_plantilla_ref','','limite',10,'cursor','','actor_ref','actor:uno',
  'contexto_actor_ref','contexto:uno','contexto_version',1,'persona_version',1,
  'perfil_ref','perfil:uno','perfil_version',1)::text;
 canon:='{"ambitos":{"organismo_ref":"org:prueba","unidad_clave":"unidad:uno"},"atributos":'||
  '{"conocido_en":"2025-06-01T00:00:00.000000Z","cursor":"sin_seleccion","limite":"10","material_sha256":"'||
   encode(sha256(convert_to(m,'UTF8')),'hex')||'","version_plantilla_ref":"sin_seleccion","version_rpt_ref":"sin_seleccion","vigente_en":"2024-12-31"}}';
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 d:=jsonb_build_object('decision_ref','decision:uno','principal_id','actor:uno',
  'perfil_activo_ref','perfil:uno','concedida',true,'valida_hasta','2099-01-01T00:00:00.000000Z','accion','personal.organizacion_historica.consultar',
  'modulo_id','personal','tipo_recurso','organizacion_historica',
  'finalidad','consultar_organizacion_historica','recurso_ref','org:prueba',
  'campos_permitidos','["dotaciones","plazas","puestos_individuales","puestos_tipo","unidades","vinculos"]'::jsonb,
  'obligaciones','[]'::jsonb,'contexto_recurso_huella_sha256',h);
 c:=jsonb_build_object('audiencia_consumo','vec_personal.organizacion_historica.consultar.v1',
  'operacion','personal.organizacion_historica.consultar','efecto_ref','org:prueba',
  'huella_efecto_sha256',h);
 BEGIN
  PERFORM vec_personal.consultar_organizacion_historica_v1(
   replace(m,'org:prueba','org:ajena'),convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),
   'm'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true;
 END;
 IF NOT denegada THEN RAISE EXCEPTION 'selector divergente no denegado'; END IF;
 -- Una unidad de 141 caracteres cumple el patrón pero supera el límite común
 -- de tablas y recibo (128): se rechaza antes de consumir la autorización.
 denegada:=false;
 DECLARE ml text; cl text; hl text;
 BEGIN
  ml:=replace(m,'"unidad:uno"','"u'||repeat('x',140)||'"');
  cl:=replace(replace(canon,'"unidad:uno"','"u'||repeat('x',140)||'"'),
   encode(sha256(convert_to(m,'UTF8')),'hex'),encode(sha256(convert_to(ml,'UTF8')),'hex'));
  hl:=encode(sha256(convert_to(cl,'UTF8')),'hex');
  PERFORM vec_personal.consultar_organizacion_historica_v1(ml,
   convert_to(jsonb_set(c,'{huella_efecto_sha256}',to_jsonb(hl))::text,'UTF8'),
   convert_to(jsonb_set(d,'{contexto_recurso_huella_sha256}',to_jsonb(hl))::text,'UTF8'),
   'm'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true;
 END;
 IF NOT denegada THEN RAISE EXCEPTION 'unidad fuera del límite común aceptada'; END IF;
 r:=vec_personal.consultar_organizacion_historica_v1(m,convert_to(c::text,'UTF8'),
  convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF jsonb_array_length(r #> '{pagina,unidades}')<>1
    OR r #>> '{pagina,unidades,0,traza,id}'<>'orgunidad:11111111-1111-4111-8111-111111111111'
    OR r #>> '{pagina,version_rpt_ref}'<>''
    OR r #>> '{pagina,cobertura,puestos_tipo}'<>'sin_datos'
    OR r #>> '{evidencia,decision_ref}'<>'decision:uno' THEN
  RAISE EXCEPTION 'proyección B3 inesperada: %',r;
 END IF;
m:=replace(m,'2025-06-01T00:00:00.000000Z','2026-06-01T00:00:00.000000Z');
 canon:=replace(canon,'2025-06-01T00:00:00.000000Z','2026-06-01T00:00:00.000000Z');
 canon:=replace(canon,split_part(split_part(canon,'material_sha256":"',2),'"',1),encode(sha256(convert_to(m,'UTF8')),'hex'));
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 d:=jsonb_set(d,'{contexto_recurso_huella_sha256}',to_jsonb(h));
 c:=jsonb_set(c,'{huella_efecto_sha256}',to_jsonb(h));
 r:=vec_personal.consultar_organizacion_historica_v1(m,convert_to(c::text,'UTF8'),
  convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF r #>> '{pagina,unidades,0,etiqueta}'<>'Centro rectificado'
    OR r #>> '{pagina,unidades,0,traza,version}'<>'2' THEN
  RAISE EXCEPTION 'rectificación tardía no consultable: %',r;
 END IF;
END $casos$;
COMMIT;
RESET SESSION AUTHORIZATION;
