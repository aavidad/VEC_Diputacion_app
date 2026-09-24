\set ON_ERROR_STOP on
SET ROLE vec_personal_propietario;
INSERT INTO vec_personal.version_rpt_historia
 (version_ref,revision,organismo_ref,codigo_version_fuente,estado,vigente_desde,conocido_desde,
  fuente_ref,documento_ref,acto_ref,huella_fuente_sha256)
 VALUES('44444444-4444-4444-8444-444444444444',1,'org:otra','rpt-2024','publicada','2024-01-01','2025-01-01+00',
  'fuente:prueba','documento:rpt','acto:rpt',repeat('a',64));
INSERT INTO vec_personal.version_plantilla_historia
 (version_ref,revision,organismo_ref,ejercicio,codigo_version_fuente,estado,vigente_desde,conocido_desde,
  fuente_ref,documento_ref,acto_ref,huella_fuente_sha256)
 VALUES('55555555-5555-4555-8555-555555555555',1,'org:otra',2024,'plantilla-2024','publicada','2024-01-01','2025-01-01+00',
  'fuente:prueba','documento:plantilla','acto:plantilla',repeat('b',64));
INSERT INTO vec_personal.org_nodo_historia
 (nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,catalogo_revision,
  catalogo_entrada_clave,denominacion,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('22222222-2222-4222-8222-222222222222',1,'org:otra','unidad:dos','centro','estructura-organizativa-dipgra',
  1,1,'unidad:dos','Centro 2','2024-01-01','2025-01-01+00','fuente:prueba','acto:centro',repeat('c',64));
INSERT INTO vec_personal.puesto_tipo_historia
 (tipo_ref,revision,organismo_ref,unidad_ref,rpt_version_ref,rpt_revision,codigo_fila_fuente,denominacion,
  clasificacion_ref,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('66666666-6666-4666-8666-666666666666',1,'org:otra','unidad:dos',
  '44444444-4444-4444-8444-444444444444',1,'fila:uno','Administrativo','clasificacion:uno',
  '2024-01-01','2025-01-01+00','fuente:prueba','acto:tipo',repeat('d',64));
INSERT INTO vec_personal.dotacion_rpt_historia
 (dotacion_ref,revision,organismo_ref,unidad_ref,tipo_ref,tipo_revision,cantidad,reconciliacion,
  vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('77777777-7777-4777-8777-777777777777',1,'org:otra','unidad:dos',
  '66666666-6666-4666-8666-666666666666',1,19,'parcial',
  '2024-01-01','2025-01-01+00','fuente:prueba','acto:dotacion',repeat('e',64));
INSERT INTO vec_personal.plaza_plantilla_historia
 (plaza_ref,revision,organismo_ref,unidad_ref,plantilla_version_ref,plantilla_revision,codigo_plaza_fuente,
  clasificacion_ref,estado_estructural,dotacion_presupuestaria,vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('88888888-8888-4888-8888-888888888888',1,'org:otra','unidad:dos',
  '55555555-5555-4555-8555-555555555555',1,'00001','clasificacion:uno','vigente','acreditada',
  '2024-01-01','2025-01-01+00','fuente:prueba','acto:plaza',repeat('f',64));
INSERT INTO vec_personal.puesto_rpt_historia
 (puesto_ref,revision,organismo_ref,unidad_ref,tipo_ref,tipo_revision,codigo_puesto_fuente,estado_estructural,
  vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('99999999-9999-4999-8999-999999999999',1,'org:otra','unidad:dos',
  '66666666-6666-4666-8666-666666666666',1,'0001','vigente',
  '2024-01-01','2025-01-01+00','fuente:prueba','acto:puesto',repeat('a',64));
INSERT INTO vec_personal.vinculo_plaza_puesto_historia
 (vinculo_ref,revision,organismo_ref,unidad_ref,plaza_ref,plaza_revision,puesto_ref,puesto_revision,estado,
  vigente_desde,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',1,'org:otra','unidad:dos',
  '88888888-8888-4888-8888-888888888888',1,'99999999-9999-4999-8999-999999999999',1,'confirmado',
  '2024-01-01','2025-01-01+00','fuente:prueba','acto:vinculo',repeat('b',64));
RESET ROLE;
SET SESSION AUTHORIZATION vec_prueba_personal;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
DO $caso$
DECLARE m text; canon text; h text; d jsonb; c jsonb; r jsonb;
BEGIN
 m:=jsonb_build_object('esquema','vec.personal.organizacion-historica.v1',
  'organismo_ref','org:otra','unidad_clave','unidad:dos','vigente_en','2024-12-31',
  'conocido_en','2025-06-01T00:00:00.000000Z','version_rpt_ref','',
  'version_plantilla_ref','','limite',100,'cursor','','actor_ref','actor:uno',
  'contexto_actor_ref','contexto:uno','contexto_version',1,'persona_version',1,
  'perfil_ref','perfil:uno','perfil_version',1)::text;
 canon:='{"ambitos":{"organismo_ref":"org:otra","unidad_clave":"unidad:dos"},"atributos":'||
  '{"conocido_en":"2025-06-01T00:00:00.000000Z","cursor":"sin_seleccion","limite":"100","material_sha256":"'||
   encode(sha256(convert_to(m,'UTF8')),'hex')||'","version_plantilla_ref":"sin_seleccion","version_rpt_ref":"sin_seleccion","vigente_en":"2024-12-31"}}';
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 d:=jsonb_build_object('decision_ref','decision:completa','principal_id','actor:uno',
  'perfil_activo_ref','perfil:uno','concedida',true,'valida_hasta','2099-01-01T00:00:00.000000Z',
  'accion','personal.organizacion_historica.consultar','modulo_id','personal',
  'tipo_recurso','organizacion_historica','finalidad','consultar_organizacion_historica',
  'recurso_ref','org:otra','campos_permitidos',
  '["dotaciones","plazas","puestos_individuales","puestos_tipo","unidades","vinculos"]'::jsonb,
  'obligaciones','[]'::jsonb,'contexto_recurso_huella_sha256',h);
 c:=jsonb_build_object('audiencia_consumo','vec_personal.organizacion_historica.consultar.v1',
  'operacion','personal.organizacion_historica.consultar','efecto_ref','org:otra','huella_efecto_sha256',h);
 r:=vec_personal.consultar_organizacion_historica_v1(m,convert_to(c::text,'UTF8'),
  convert_to(d::text,'UTF8'),'m'::bytea,'x'::bytea,1,1,'p'::bytea,'s'::bytea,'e'::bytea,'r'::bytea);
 IF (SELECT sum(jsonb_array_length(value)) FROM jsonb_each(r->'pagina') WHERE key IN
     ('unidades','puestos_tipo','dotaciones','plazas','puestos_individuales','vinculos'))<>6
    OR r #>> '{pagina,version_rpt_ref}'<>'rpt:44444444-4444-4444-8444-444444444444'
    OR r #>> '{pagina,version_plantilla_ref}'<>'plantilla:55555555-5555-4555-8555-555555555555'
    OR r #>> '{pagina,dotaciones,0,cantidad}'<>'19'
    OR r #>> '{pagina,plazas,0,codigo_fuente}'<>'00001'
    OR r #>> '{pagina,puestos_individuales,0,codigo_fuente}'<>'0001'
    OR (r->'pagina') ? 'vacantes' THEN
  RAISE EXCEPTION 'proyección de seis clases incorrecta: %',r;
 END IF;
END $caso$;
COMMIT;
RESET SESSION AUTHORIZATION;
