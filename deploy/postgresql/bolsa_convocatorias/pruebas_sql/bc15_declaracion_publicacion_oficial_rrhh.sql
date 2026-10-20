\set ON_ERROR_STOP on
-- Vectores BC15 en una transacción revertida, con datos ficticios. Sin el
-- consumidor del núcleo la declaración web falla cerrada (55000); con él, un
-- material que no es una decisión V3 válida se deniega (42501). Nunca escribe.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
SET LOCAL timezone='UTC';
DO $preparar$
DECLARE id text:='convocatoria:bc15:vector'; id2 text:='convocatoria:bc15:sin-ambito';
 contenido jsonb; bytes bytea; pol_doc text; pol_sha text; ref text;
 f_declarar regprocedure:='vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 f_consultar regprocedure:='vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(text,integer)'::regprocedure;
BEGIN
 -- Concesiones: solo el ejecutor de Bolsa, solo estas dos funciones.
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_declarar,'EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f_consultar,'EXECUTE')
 OR NOT has_schema_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_convocatorias','USAGE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb)','EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(text)','EXECUTE')
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_convocatorias.publicacion_oficial_bases','SELECT')
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion','SELECT')
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_convocatorias.version_convocatoria','SELECT')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid IN(f_declarar,f_consultar) AND a.grantee=0)
 THEN RAISE EXCEPTION 'BC15: concesiones abiertas o ausentes'; END IF;
 -- Solo la primera versión tiene ámbito de gestión publicado.
 pol_doc:=jsonb_build_object('id','bolsa.politica.bc15','version',1,'revision',1,'modulo_id','bolsa',
  'nombre','Política sintética','fuente_ref','fuente:sintetica','motivo_creacion','Prueba BC15',
  'entradas',jsonb_build_array(jsonb_build_object(
   'clave','ambito.gestion.'||encode(sha256(convert_to(id,'UTF8')),'hex')||'.v1',
   'etiqueta','Ámbito RRHH','orden',1,'vigente_desde','2026-10-01T00:00:00Z',
   'atributos',jsonb_build_object('convocatoria_ref','cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1',
    'unidad_ref','unidad:bc15','ambito_ref','ambito:bc15','canal','interna_corporativa'))),
  'estado','publicado','creado_por','persona:sintetica-1','creado_en','2026-10-01T00:00:00Z',
  'publicado_por','persona:sintetica-2','publicado_en','2026-10-01T00:00:00Z',
  'aprobacion_ref','aprobacion:sintetica','motivo_publicacion','Prueba')::text;
 pol_sha:=encode(sha256(convert_to(pol_doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.politica.bc15',1,pol_sha,pol_doc,'{}'::jsonb,
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:bc15:a','aprobacion:bc15:b','actor:bc15',
  'decision:bc15','recibo:bc15','motivos_bolsa:1:motivo_demo');
 FOREACH ref IN ARRAY ARRAY[id,id2] LOOP
  contenido:=jsonb_build_object('id',ref,'secuencia',1,'estado_gobierno','publicada',
   'publicada_en',statement_timestamp()-interval '3 days',
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',ref||'#1'),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',ref||'#1'),
   'contenido',jsonb_build_object('identificador_publico','BC15-'||right(ref,6),'titulo','Convocatoria BC15',
    'resumen','Vector','tipo','bolsa','categorias',jsonb_build_array('cat.alpha'),
    'plazos',jsonb_build_array(jsonb_build_object('tipo','inscripcion','referencia','plazo:bc15',
      'abre_en',statement_timestamp()+interval '2 days','cierra_en',statement_timestamp()+interval '30 days')),
    'requisitos','[]'::jsonb,
    'catalogo_categorias',jsonb_build_object('catalogo_id','catalogo:bc15:categorias',
      'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64))),
   'configuracion',jsonb_build_object(
    'catalogos',jsonb_build_object('id','bolsa.politica.bc15','version',1,'huella_contenido_sha256',pol_sha),
    'flujo_solicitud',jsonb_build_object('id','flujo:bc15','version',1,'huella_contenido_sha256',repeat('c',64)),
    'documentos',jsonb_build_array(jsonb_build_object('rol','bases','publicacion_ref','publicacion:bc15'))));
  bytes:=convert_to(contenido::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria(convocatoria_id,secuencia,referencia,estado,
   version_canonica,huella_version_sha256,instancia_flujo_ref,registrada_en)
  VALUES(ref,1,ref||'#1','publicada',bytes,encode(sha256(bytes),'hex'),NULL,statement_timestamp());
 END LOOP;
 PERFORM set_config('vec.bc15.ref','cv1_'||encode(sha256(convert_to(id,'UTF8')),'hex')||'_v1',true);
 PERFORM set_config('vec.bc15.ref2','cv1_'||encode(sha256(convert_to(id2,'UTF8')),'hex')||'_v1',true);
 -- Fuera del pool de Bolsa, ni lectura ni declaración.
 BEGIN
  PERFORM vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(NULL,10);
  RAISE EXCEPTION 'BC15: consulta sin sesión de Bolsa';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $preparar$;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $sesion$
DECLARE ref text:=current_setting('vec.bc15.ref'); ref2 text:=current_setting('vec.bc15.ref2');
 lista jsonb; una jsonb; base jsonb; fallo text; esperado text; vacio bytea:='\x7b7d';
BEGIN
 lista:=vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(NULL,100);
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(lista->'items') x
   WHERE x->>'convocatoria_ref'=ref AND x->>'ambito_ref'='ambito:bc15' AND x->>'unidad_ref'='unidad:bc15'
    AND x->'publicacion'='null'::jsonb AND NOT x ? 'convocatoria_id')
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(lista->'items') x WHERE x->>'convocatoria_ref'=ref2)
 THEN RAISE EXCEPTION 'BC15: lista incorrecta %',lista; END IF;
 una:=vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(ref,1);
 IF jsonb_array_length(una->'items')<>1 OR una#>>'{items,0,convocatoria_ref}'<>ref
 OR una#>'{items,0}' ? 'convocatoria_id' THEN RAISE EXCEPTION 'BC15: consulta de una versión incorrecta'; END IF;
 FOREACH fallo IN ARRAY ARRAY['cv1_x_v1',ref2] LOOP
  BEGIN
   PERFORM vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(fallo,1);
   RAISE EXCEPTION 'BC15: versión % admitida',fallo;
  EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'B9601' THEN NULL; END;
 END LOOP;
 base:=jsonb_build_object('convocatoria_ref',ref,'huella_version_sha256',una#>>'{items,0,huella_version_sha256}',
  'medio','bop_granada','boletin_numero',201,'anuncio_ref','ENSAYO-BOP-2026-000201',
  'fecha_publicacion',to_char((clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date,'YYYY-MM-DD'),
  'direccion',NULL,'sustituye_ref',NULL,'clave_idempotencia','bc15-prueba-0001');
 BEGIN
  PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(base||'{"declarada_por":"x"}',
   vacio,vacio,vacio,vacio,1,1,vacio,vacio,vacio,vacio);
  RAISE EXCEPTION 'BC15: clave del declarante admitida en el cuerpo';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 -- Sin consumidor: 55000. Con consumidor (AD301): un material que no es una
 -- decisión V3 se deniega. En ningún caso se escribe.
 esperado:=CASE WHEN to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  THEN '55000' ELSE '42501' END;
 BEGIN
  PERFORM vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(base,
   vacio,vacio,vacio,vacio,1,1,vacio,vacio,vacio,vacio);
  RAISE EXCEPTION 'BC15: declaración sin decisión admitida';
 EXCEPTION WHEN OTHERS THEN
  IF SQLSTATE<>esperado THEN RAISE; END IF;
 END;
END $sesion$;
RESET SESSION AUTHORIZATION;
DO $despues$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases WHERE clave_idempotencia LIKE 'bc15-%')
 OR EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion)
 THEN RAISE EXCEPTION 'BC15: la denegación escribió'; END IF;
END $despues$;
ROLLBACK;
