\set ON_ERROR_STOP on
-- Requiere CC1 real, CC11 y roles de sus consumidores en PostgreSQL 18
-- desechable. Publica por el contrato CC1, con entradas de forma Go ADMIN.
-- Todo dato y control creados se revierten al final.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='2s';
DO $publicar$
DECLARE vacia_sha text:=encode(sha256(convert_to('{}','UTF8')),'hex');
 doc text; huella text; entradas jsonb; referencia text;
 clave_gestion text; desde text:='2026-10-09T00:00:00Z';
BEGIN
 referencia:='cv1_'||encode(sha256(convert_to('proceso:bolsa:real-2026','UTF8')),'hex')||'_v1';
 clave_gestion:='ambito.gestion.'||substring(referencia from '^cv1_([0-9a-f]{64})_v1$')||'.v1';
 entradas:=jsonb_build_array(
  jsonb_build_object('clave','cat.alpha','etiqueta','Alfa ES','orden',1,
   'vigente_desde',desde,'atributos',jsonb_build_object('etiqueta_en','Alpha EN')));
 doc:=jsonb_build_object('id','bolsa.categorias.real','version',1,'revision',1,
  'modulo_id','bolsa','nombre','Categorías sintéticas','fuente_ref','fuente:sintetica',
  'motivo_creacion','Prueba de catálogo','entradas',entradas,'estado','publicado',
  'creado_por','persona:sintetica-1','creado_en',desde,
  'publicado_por','persona:sintetica-2','publicado_en',desde,
  'aprobacion_ref','aprobacion:sintetica','motivo_publicacion','Prueba')::text;
 huella:=encode(sha256(convert_to(doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.categorias.real',1,huella,doc,
  '{}'::jsonb,vacia_sha,'aprobacion:cat:a','aprobacion:cat:b',
  'actor:cc11','decision:cat','recibo:cc11:cat','motivos_bolsa:1:motivo_demo');

 entradas:=jsonb_build_array(
  jsonb_build_object('clave','presentacion.con.pendientes','etiqueta','Presentación externa',
   'orden',1,'vigente_desde',desde,
   'atributos',jsonb_build_object('valor','true','canal','externa_personal')),
  jsonb_build_object('clave','presentacion.con.pendientes.empleado','etiqueta','Presentación empleado',
   'orden',2,'vigente_desde',desde,
   'atributos',jsonb_build_object('valor','true','canal','interna_corporativa')),
  jsonb_build_object('clave','requisito.pendiente','etiqueta','Pendiente ES',
   'orden',3,'vigente_desde',desde,'atributos',jsonb_build_object('etiqueta_en','Pending EN')),
  jsonb_build_object('clave','requisito.cumple','etiqueta','Cumple ES',
   'orden',4,'vigente_desde',desde,'atributos',jsonb_build_object('etiqueta_en','Meets EN')),
  jsonb_build_object('clave','solicitud.existente','etiqueta','Ya solicitada ES',
   'orden',5,'vigente_desde',desde,'atributos',jsonb_build_object('etiqueta_en','Already applied EN')),
  jsonb_build_object('clave','presentacion.no.disponible','etiqueta','No disponible ES',
   'orden',6,'vigente_desde',desde,'atributos',jsonb_build_object('etiqueta_en','Unavailable EN')),
  jsonb_build_object('clave','asociacion.manual.acta','etiqueta','Asociación manual',
   'orden',7,'vigente_desde',desde,'atributos',jsonb_build_object(
    'valor','true','canal','interna_corporativa',
    'motivo_ref','motivo:asociacion.manual','alcance_ref',referencia)),
  jsonb_build_object('clave',clave_gestion,'etiqueta','Ámbito RRHH',
   'orden',8,'vigente_desde',desde,'atributos',jsonb_build_object(
    'convocatoria_ref',referencia,'unidad_ref','unidad:rrhh-1',
    'ambito_ref','ambito:gestion-1','canal','interna_corporativa')));
 doc:=jsonb_build_object('id','bolsa.politica.real','version',1,'revision',1,
  'modulo_id','bolsa','nombre','Política sintética','fuente_ref','fuente:sintetica',
  'motivo_creacion','Prueba de política','entradas',entradas,'estado','publicado',
  'creado_por','persona:sintetica-1','creado_en',desde,
  'publicado_por','persona:sintetica-2','publicado_en',desde,
  'aprobacion_ref','aprobacion:sintetica','motivo_publicacion','Prueba')::text;
 huella:=encode(sha256(convert_to(doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.politica.real',1,huella,doc,
  '{}'::jsonb,vacia_sha,'aprobacion:pol:a','aprobacion:pol:b',
  'actor:cc11','decision:pol','recibo:cc11:pol','motivos_bolsa:1:motivo_demo');

 entradas:=jsonb_build_array(jsonb_build_object('clave','motivo.manual',
  'etiqueta','Motivo manual ES','orden',1,'vigente_desde',desde,
  'atributos',jsonb_build_object('etiqueta_en','Manual reason EN')));
 doc:=jsonb_build_object('id','bolsa.inscripcion.motivos','version',1,'revision',1,
  'modulo_id','bolsa','nombre','Motivos sintéticos','fuente_ref','fuente:sintetica',
  'motivo_creacion','Prueba de motivos','entradas',entradas,'estado','publicado',
  'creado_por','persona:sintetica-1','creado_en',desde,
  'publicado_por','persona:sintetica-2','publicado_en',desde,
  'aprobacion_ref','aprobacion:sintetica','motivo_publicacion','Prueba')::text;
 huella:=encode(sha256(convert_to(doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.inscripcion.motivos',1,huella,doc,
  '{}'::jsonb,vacia_sha,'aprobacion:mot:a','aprobacion:mot:b',
  'actor:cc11','decision:mot','recibo:cc11:mot','motivos_bolsa:1:motivo_demo');

 entradas:=jsonb_build_array(jsonb_build_object(
  'clave','inscripcion.gestion.rrhh.conjunto','etiqueta','Conjunto RRHH',
  'orden',1,'vigente_desde',desde,'atributos',jsonb_build_object(
   'conjunto_ref','conjunto:rrhh-1','unidad_ref','unidad:rrhh-1',
   'ambito_ref','ambito:gestion-1','canal','interna_corporativa')));
 doc:=jsonb_build_object('id','bolsa.gestion.rrhh.real','version',1,'revision',1,
  'modulo_id','bolsa','nombre','Conjunto RRHH sintético',
  'fuente_ref','fuente:sintetica','motivo_creacion','Prueba de conjunto',
  'entradas',entradas,'estado','publicado',
  'creado_por','persona:sintetica-1','creado_en',desde,
  'publicado_por','persona:sintetica-2','publicado_en',desde,
  'aprobacion_ref','aprobacion:sintetica','motivo_publicacion','Prueba')::text;
 huella:=encode(sha256(convert_to(doc,'UTF8')),'hex');
 PERFORM vec_catalogos_configurables.publicar('bolsa.gestion.rrhh.real',1,huella,doc,
  '{}'::jsonb,vacia_sha,'aprobacion:conjunto:a','aprobacion:conjunto:b',
  'actor:cc11','decision:conjunto','recibo:cc11:conjunto','motivos_bolsa:1:motivo_demo');
END $publicar$;
RESET ROLE;

DO $comprobar$
DECLARE ref text:='cv1_'||encode(sha256(convert_to('proceso:bolsa:real-2026','UTF8')),'hex')||'_v1';
 h_cat text; h_pol text; etiquetas jsonb; categorias jsonb; gestion jsonb; ambitos jsonb;
 motivos jsonb; conjunto jsonb;
BEGIN
 SELECT huella_sha256 INTO h_cat FROM vec_catalogos_configurables.publicacion
  WHERE catalogo_id='bolsa.categorias.real' AND version=1;
 SELECT huella_sha256 INTO h_pol FROM vec_catalogos_configurables.publicacion
  WHERE catalogo_id='bolsa.politica.real' AND version=1;
 IF h_cat IS NULL OR h_pol IS NULL
 OR (SELECT count(*) FROM vec_catalogos_configurables.entrada_publicada
   WHERE catalogo_id='bolsa.politica.real' AND version=1)<>8
 THEN RAISE EXCEPTION 'CC11: publicación CC1 real incompleta'; END IF;
 IF NOT vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
  'bolsa.politica.real',1,h_pol,'externa_personal')
 OR NOT vec_catalogos_configurables.comprobar_politica_presentacion_inscripcion_v1(
  'bolsa.politica.real',1,h_pol,'interna_corporativa')
 THEN RAISE EXCEPTION 'CC11: política Go no legible'; END IF;
 etiquetas:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.real',
   'catalogo_version',1,'catalogo_sha256',h_cat,'categoria_ref','cat.alpha',
   'politica_catalogo_ref','bolsa.politica.real',
   'politica_catalogo_version',1,'politica_catalogo_sha256',h_pol)),
  'en','externa_personal');
 IF etiquetas#>>'{0,categoria}'<>'Alpha EN'
 OR etiquetas#>>'{0,motivo_etiqueta_pendiente}'<>'Pending EN'
 OR etiquetas#>>'{0,politica_valida}'<>'true'
 THEN RAISE EXCEPTION 'CC11: etiquetas/política Go incompatibles: %',etiquetas; END IF;
 etiquetas:=vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.real',
   'catalogo_version',1,'catalogo_sha256',h_cat,'categoria_ref','cat.alpha',
   'politica_catalogo_ref','bolsa.politica.real',
   'politica_catalogo_version',1,'politica_catalogo_sha256',h_pol)),
  'es','interna_corporativa');
 IF etiquetas#>>'{0,categoria}'<>'Alfa ES' OR etiquetas#>>'{0,politica_valida}'<>'true'
 THEN RAISE EXCEPTION 'CC11: canal empleado Go incompatible'; END IF;
 categorias:=vec_catalogos_configurables.comprobar_categorias_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('catalogo_ref','bolsa.categorias.real',
   'catalogo_version',1,'catalogo_sha256',h_cat,
   'categorias_refs',jsonb_build_array('cat.alpha'))),'en');
 IF categorias#>>'{0,catalogo_completo}'<>'true'
 THEN RAISE EXCEPTION 'CC11: categoría Go no verificable'; END IF;
 gestion:=vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
  'bolsa.politica.real',1,h_pol,ref);
 IF gestion->>'unidad_ref'<>'unidad:rrhh-1' OR gestion->>'ambito_ref'<>'ambito:gestion-1'
 THEN RAISE EXCEPTION 'CC11: ámbito Go no legible'; END IF;
 ambitos:=vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(
  jsonb_build_array(jsonb_build_object('politica_catalogo_ref','bolsa.politica.real',
   'politica_catalogo_version',1,'politica_catalogo_sha256',h_pol,
   'convocatoria_ref',ref)));
 IF ambitos#>>'{0,ambito_gestion_disponible}'<>'true' OR ambitos->0 ? 'unidad_ref'
 THEN RAISE EXCEPTION 'CC11: ámbito público Go expuso scope'; END IF;
 IF vec_catalogos_configurables.comprobar_politica_asociacion_inscripcion_v1(
  'bolsa.politica.real',1,h_pol)#>>'{alcance_ref}'<>ref
 THEN RAISE EXCEPTION 'CC11: asociación Go incompatible'; END IF;
 motivos:=vec_catalogos_configurables.listar_motivos_inscripcion_v1('en');
 IF motivos#>>'{motivos,0,motivo_etiqueta}'<>'Manual reason EN'
 OR vec_catalogos_configurables.comprobar_motivo_inscripcion_v1(
   'motivo.manual',1,motivos->>'catalogo_sha256','en')->>'motivo_etiqueta'<>'Manual reason EN'
 THEN RAISE EXCEPTION 'CC11: motivo Go incompatible'; END IF;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()','EXECUTE')
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()','EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_propietario',
   'vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1()','EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_propietario',
   'vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()','EXECUTE')
 THEN RAISE EXCEPTION 'CC11: ACL conjunto RRHH incorrecta'; END IF;
 conjunto:=vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1();
 IF conjunto->>'conjunto_ref'<>'conjunto:rrhh-1'
 OR conjunto->>'unidad_ref'<>'unidad:rrhh-1'
 OR conjunto->>'ambito_ref'<>'ambito:gestion-1'
 OR conjunto ? 'catalogo_ref' OR (SELECT count(*) FROM jsonb_object_keys(conjunto))<>6
 OR conjunto->>'fuente_ref'<>'inscripcion.gestion.rrhh.conjunto'
 OR conjunto->>'fuente_version'<>'1'
 OR conjunto->>'fuente_sha256' IS NULL
 THEN RAISE EXCEPTION 'CC11: conjunto RRHH Go incompleto: %',conjunto; END IF;
 IF vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1()
  IS DISTINCT FROM conjunto
 THEN RAISE EXCEPTION 'CC11: cotejo de efecto divergente'; END IF;
 PERFORM vec_catalogos_configurables.cambiar_proyeccion(
  'inscripcion.gestion.rrhh.conjunto',1,'deshabilitar',NULL,NULL,
  'actor:cc11','decision:desconjunto','recibo:cc11:desconjunto',
  'motivos_bolsa:1:motivo_demo');
 BEGIN
  PERFORM vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1();
  RAISE EXCEPTION 'CC11: conjunto deshabilitado aceptado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
 BEGIN
  PERFORM vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1();
  RAISE EXCEPTION 'CC11: recheck aceptó conjunto deshabilitado';
 EXCEPTION WHEN SQLSTATE 'B9601' THEN NULL; END;
END $comprobar$;
ROLLBACK;
BEGIN;
DO $aislamiento$
BEGIN
 BEGIN
  PERFORM vec_catalogos_configurables.cotejar_conjunto_gestion_rrhh_inscripcion_actual_v1();
  RAISE EXCEPTION 'CC11: recheck sin SERIALIZABLE aceptado';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $aislamiento$;
ROLLBACK;
