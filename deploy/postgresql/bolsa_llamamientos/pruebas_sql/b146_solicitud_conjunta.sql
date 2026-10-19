\set ON_ERROR_STOP on
-- B146 (F1-41): solicitud conjunta con un solo recibo y reintento, y solicitud
-- separada por categoría. Todo ocurre en una transacción revertida: el LOGIN
-- externo de desarrollo y una fachada del consumo V3 (decisión nueva por
-- llamada) existen solo dentro de ella. No acredita V3 real.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL statement_timeout='60s';
DO $logins$
BEGIN
 IF to_regrole('vec_externo_bolsa_desarrollo') IS NULL THEN
  CREATE ROLE vec_externo_bolsa_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_bolsa_llamamientos_portal_externo TO vec_externo_bolsa_desarrollo WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
 END IF;
END $logins$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_presentacion_inscripcion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
  encode(sha256(p_decision),'hex'),'aud_v3_'||substr(encode(sha256(p_decision),'hex'),1,32),
  clock_timestamp(),true;
END $f$;

-- Dos convocatorias publicadas: conjunta (tres categorías) y separada (dos).
DO $datos$
DECLARE
 cat_doc text:='{"id":"bolsa.categorias.b146","version":1}';
 pol_doc text:='{"id":"bolsa.politica.b146","version":1}';
 cat_sha text; pol_sha text; material bytea; canon jsonb; ref_conv text; clave_gestion text;
 modalidad text; id text; categorias jsonb; categoria text;
BEGIN
 cat_sha:=encode(sha256(convert_to(cat_doc,'UTF8')),'hex');
 pol_sha:=encode(sha256(convert_to(pol_doc,'UTF8')),'hex');
 INSERT INTO vec_catalogos_configurables.publicacion(
  catalogo_id,version,huella_sha256,documento_canonico,preimagenes_control,
  preimagenes_huella_sha256,aprobacion_a_ref,aprobacion_b_ref,actor_ref,decision_ref,recibo_ref)
 VALUES('bolsa.categorias.b146',1,cat_sha,cat_doc,'{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:b146:cat:a','aprobacion:b146:cat:b',
  'actor:b146','decision:b146:cat','recibo:b146:cat'),
 ('bolsa.politica.b146',1,pol_sha,pol_doc,'{}',
  encode(sha256(convert_to('{}','UTF8')),'hex'),'aprobacion:b146:pol:a','aprobacion:b146:pol:b',
  'actor:b146','decision:b146:pol','recibo:b146:pol');
 FOREACH categoria IN ARRAY ARRAY['cat.alpha','cat.beta','cat.gamma'] LOOP
  INSERT INTO vec_catalogos_configurables.entrada_publicada(
   catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
  VALUES('bolsa.categorias.b146',1,cat_sha,categoria,'Categoría '||categoria,
   jsonb_build_object('clave',categoria,'etiqueta','Categoría '||categoria,'orden',1,
    'vigente_desde',clock_timestamp()-interval '1 day',
    'atributos',jsonb_build_object('etiqueta_en','Category '||categoria)));
 END LOOP;
 INSERT INTO vec_catalogos_configurables.entrada_publicada(
  catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
 VALUES
 ('bolsa.politica.b146',1,pol_sha,'presentacion.con.pendientes','Política',
  '{"valor":true,"canal":"externa_personal"}'),
 ('bolsa.politica.b146',1,pol_sha,'requisito.pendiente','Pendiente',
  '{"etiquetas":{"es":"Pendiente","en":"Pending"}}'),
 ('bolsa.politica.b146',1,pol_sha,'requisito.cumple','Cumple',
  '{"etiquetas":{"es":"Cumple","en":"Meets"}}'),
 ('bolsa.politica.b146',1,pol_sha,'solicitud.existente','Ya solicitada',
  '{"etiquetas":{"es":"Ya solicitada","en":"Already applied"}}'),
 ('bolsa.politica.b146',1,pol_sha,'presentacion.no.disponible','No disponible',
  '{"etiquetas":{"es":"No disponible","en":"Unavailable"}}');
 FOREACH modalidad IN ARRAY ARRAY['conjunta','separada'] LOOP
  id:='proceso:bolsa:inscripcion-b146-'||modalidad;
  ref_conv:='cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1(id)||'_v1';
  -- El LOGIN externo no lee el esquema de convocatorias: la referencia viaja aquí.
  PERFORM set_config('b146.'||modalidad,ref_conv,true);
  clave_gestion:='ambito.gestion.'||substring(ref_conv FROM 5 FOR 64)||'.v1';
  categorias:=CASE modalidad WHEN 'conjunta' THEN '["cat.alpha","cat.beta","cat.gamma"]'::jsonb
   ELSE '["cat.alpha","cat.beta"]'::jsonb END;
  INSERT INTO vec_catalogos_configurables.entrada_publicada(
   catalogo_id,version,huella_sha256,categoria_id,etiqueta,definicion)
  VALUES('bolsa.politica.b146',1,pol_sha,clave_gestion,'Ámbito RRHH',
   jsonb_build_object('clave',clave_gestion,'etiqueta','Ámbito RRHH','orden',1,
    'vigente_desde',clock_timestamp()-interval '1 day',
    'atributos',jsonb_build_object('convocatoria_ref',ref_conv,'unidad_ref','unidad:rrhh-b146',
     'ambito_ref','ambito:gestion-b146','canal','interna_corporativa')));
  INSERT INTO vec_catalogos_configurables.categoria_control(
   categoria_id,catalogo_id,version,huella_sha256,revision,estado)
  VALUES(clave_gestion,'bolsa.politica.b146',1,pol_sha,1,'habilitada');
  canon:=jsonb_build_object('id',id,'secuencia',1,'estado_gobierno','publicada',
   'publicada_en',clock_timestamp()-interval '1 day',
   'aprobacion_publicacion',jsonb_build_object('convocatoria_ref',id||'#1','ref','aprobacion:b146'),
   'comprobacion_dependencias',jsonb_build_object('convocatoria_ref',id||'#1','ref','dependencias:b146'),
   'contenido',jsonb_build_object('identificador_publico','inscripcion-b146-'||modalidad,
    'titulo','Convocatoria B146','tipo','bolsa','resumen','Prueba sintética B146',
    'modalidad_inscripcion',modalidad,'categorias',categorias,
    'catalogo_categorias',jsonb_build_object('catalogo_id','bolsa.categorias.b146',
     'catalogo_version',1,'catalogo_huella_sha256',cat_sha),
    'plazos',jsonb_build_array(jsonb_build_object('referencia','plazo.b146','tipo','inscripcion',
     'abre_en',clock_timestamp()-interval '1 hour','cierra_en',clock_timestamp()+interval '1 hour')),
    'requisitos',jsonb_build_array(jsonb_build_object('referencia','identidad_certificada','orden',1,
     'descripcion','Identidad acreditada por certificado','obligatorio',true))),
   'configuracion',jsonb_build_object(
    'catalogos',jsonb_build_object('id','bolsa.politica.b146','version',1,'huella_contenido_sha256',pol_sha),
    'flujo_solicitud',jsonb_build_object('id','flujo.b146','version',1,'huella_contenido_sha256',repeat('a',64)),
    'documentos',jsonb_build_array(jsonb_build_object('rol','bases','publicacion_ref','bases.b146'))));
  material:=convert_to(canon::text,'UTF8');
  INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
   convocatoria_id,secuencia,referencia,estado,version_canonica,huella_version_sha256,registrada_en)
  VALUES(id,1,id||'#1','publicada',material,encode(sha256(material),'hex'),clock_timestamp());
 END LOOP;
END $datos$;

-- Constructor de las once listas para una presentación conjunta.
CREATE FUNCTION pg_temp.b146_presentar(p_persona text,p_conv text,p_categorias text[],p_clave text,
 p_material_categorias text[] DEFAULT NULL,p_decisiones integer DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE
 material text; material_sha text; recurso text; recurso_sha text; solicitud text; i integer;
 categorias text[]:=coalesce(p_material_categorias,p_categorias);
 candidato text:='can_'||substr(p_persona,5); perfil text:='prf_'||repeat('p',22); cuenta text:='cta_'||repeat('c',22);
 recursos bytea[]:='{}'; caps bytea[]:='{}'; decs bytea[]:='{}'; motivos bytea[]:='{}'; ctxs bytea[]:='{}';
 vers numeric[]:='{}'; payloads bytea[]:='{}'; contexto bytea;
BEGIN
 material:='{"esquema":"vec.bolsa.inscripcion.presentar_conjunta.v1","convocatoria_ref":'||to_json(p_conv)::text
  ||',"categorias_refs":'||to_json(categorias)::text||',"catalogo_version":1,"clave_idempotencia":'
  ||to_json(p_clave)::text||',"declaraciones":[]}';
 material_sha:=encode(sha256(convert_to(material,'UTF8')),'hex');
 contexto:=convert_to(jsonb_build_object('principal_ref',p_persona,'perfil_activo_ref',perfil,
  'cuenta_ref',cuenta,'metodo','certificado','garantia','alto','contexto_actor_ref','vca_'||repeat('v',22),
  'contexto_version',1,'vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato','referencia',candidato)))::text,'UTF8');
 FOR i IN 1..coalesce(p_decisiones,cardinality(p_categorias)) LOOP
  solicitud:='solicitud_inscripcion_'||encode(sha256(convert_to(
   p_persona||chr(31)||p_conv||chr(31)||p_categorias[i],'UTF8')),'hex');
  recurso:='{"ambitos":{"candidato_ref":'||to_json(candidato)::text||'},"atributos":{"convocatoria_ref":'
   ||to_json(p_conv)::text||',"categoria_ref":'||to_json(p_categorias[i])::text
   ||',"catalogo_version":"1","material_sha256":"'||material_sha||'"}}';
  recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
  recursos:=recursos||convert_to(recurso,'UTF8');
  caps:=caps||convert_to(jsonb_build_object('operacion','bolsa.inscripcion.presentar',
   'audiencia_consumo','vec_bolsa_llamamientos.inscripcion.presentar.v1',
   'efecto_ref',solicitud,'huella_efecto_sha256',recurso_sha)::text,'UTF8');
  decs:=decs||convert_to(jsonb_build_object('decision_ref','decision:b146:'||gen_random_uuid(),
   'principal_id',p_persona,'perfil_activo_ref',perfil,'concedida',true,
   'accion','bolsa.inscripcion.presentar','modulo_id','bolsa','tipo_recurso','inscripcion_convocatoria',
   'finalidad','presentar_inscripcion','recurso_ref',solicitud,'contexto_recurso_huella_sha256',recurso_sha,
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','externa_personal'))::text,'UTF8');
  motivos:=motivos||convert_to('{}','UTF8');
  ctxs:=ctxs||contexto;
  vers:=vers||1::numeric;
  payloads:=payloads||convert_to('x','UTF8');
 END LOOP;
 RETURN vec_bolsa_llamamientos.solicitar_inscripcion_conjunta_v1(material,
  jsonb_build_object('persona_ref',p_persona,'perfil_ref',perfil,'cuenta_ref',cuenta,
   'canal','externa_personal','idioma','es'),
  recursos,caps,decs,motivos,ctxs,vers,vers,payloads,payloads,payloads,
  array_fill(convert_to(repeat('a',44),'UTF8'),ARRAY[cardinality(recursos)]));
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.b146_presentar(text,text,text[],text,text[],integer) TO vec_externo_bolsa_desarrollo;

SET SESSION AUTHORIZATION vec_externo_bolsa_desarrollo;
DO $casos$
DECLARE
 conj text:=current_setting('b146.conjunta');
 sep text:=current_setting('b146.separada');
 persona text:='per_'||repeat('a',22); recibo jsonb; repetido jsonb; codigo text;
 caso record;
BEGIN
 recibo:=pg_temp.b146_presentar(persona,conj,ARRAY['cat.alpha','cat.beta'],'ClavePruebaB146001');
 IF recibo->>'repetida'<>'false' OR recibo->>'estado'<>'pendiente'
 OR recibo->>'recibo_ref' !~ '^recibo_inscripcion_conjunta_[0-9a-f]{64}$'
 OR jsonb_array_length(recibo->'categorias')<>2
 OR recibo#>>'{categorias,0,categoria_ref}'<>'cat.alpha' OR recibo#>>'{categorias,1,categoria_ref}'<>'cat.beta'
 OR recibo#>>'{categorias,0,categoria}'<>'Categoría cat.alpha'
 THEN RAISE EXCEPTION 'B146: primer recibo conjunto incompleto %',recibo; END IF;
 -- Reintento con la misma clave y material: mismo recibo, decisiones nuevas.
 repetido:=pg_temp.b146_presentar(persona,conj,ARRAY['cat.alpha','cat.beta'],'ClavePruebaB146001');
 IF repetido->>'repetida'<>'true' OR repetido->>'recibo_ref'<>recibo->>'recibo_ref'
 OR repetido->>'registrada_en'<>recibo->>'registrada_en'
 OR repetido#>'{categorias,0,solicitud_ref}'<>recibo#>'{categorias,0,solicitud_ref}'
 OR repetido#>'{categorias,1,recibo_ref}'<>recibo#>'{categorias,1,recibo_ref}'
 THEN RAISE EXCEPTION 'B146: reintento divergente % %',recibo,repetido; END IF;
 -- Rechazos: código esperado y ningún efecto.
 FOR caso IN SELECT * FROM (VALUES
   ('B9603',persona,conj,ARRAY['cat.alpha','cat.gamma'],'ClavePruebaB146001',NULL::text[],NULL::integer),
   ('B9604',persona,conj,ARRAY['cat.alpha','cat.beta'],'ClavePruebaB146002',NULL,NULL),
   ('B9605','per_'||repeat('b',22),conj,ARRAY['cat.beta','cat.alpha'],'ClavePruebaB146003',NULL,NULL),
   ('B9605','per_'||repeat('b',22),conj,ARRAY['cat.alpha','cat.delta'],'ClavePruebaB146004',NULL,NULL),
   ('B9608','per_'||repeat('b',22),sep,ARRAY['cat.alpha','cat.beta'],'ClavePruebaB146005',NULL,NULL),
   ('42501','per_'||repeat('b',22),conj,ARRAY['cat.alpha','cat.beta'],'ClavePruebaB146006',NULL,1),
   ('42501','per_'||repeat('b',22),conj,ARRAY['cat.beta','cat.alpha'],'ClavePruebaB146007',
     ARRAY['cat.alpha','cat.beta'],NULL)
  ) AS x(esperado,persona,conv,categorias,clave,material,decisiones) LOOP
  codigo:=NULL;
  BEGIN
   PERFORM pg_temp.b146_presentar(caso.persona,caso.conv,caso.categorias,caso.clave,caso.material,caso.decisiones);
  EXCEPTION WHEN OTHERS THEN codigo:=SQLSTATE;
  END;
  IF codigo IS DISTINCT FROM caso.esperado
  THEN RAISE EXCEPTION 'B146: caso % devolvió %',caso,codigo; END IF;
 END LOOP;
END $casos$;

-- Separada (contenido de B105): una solicitud por categoría, sin repetir.
DO $separada$
DECLARE
 sep text:=current_setting('b146.separada');
 persona text:='per_'||repeat('s',22); candidato text:='can_'||repeat('s',22);
 perfil text:='prf_'||repeat('p',22); cuenta text:='cta_'||repeat('c',22);
 categoria text; clave text; material text; material_sha text; recurso text; recurso_sha text; solicitud text;
 recibo jsonb; codigo text; vuelta integer;
BEGIN
 FOR vuelta IN 1..3 LOOP
  categoria:=CASE WHEN vuelta=2 THEN 'cat.beta' ELSE 'cat.alpha' END;
  clave:='ClaveSeparadaB146'||vuelta;
  material:='{"esquema":"vec.bolsa.inscripcion.presentar.v1","convocatoria_ref":'||to_json(sep)::text
   ||',"categoria_ref":'||to_json(categoria)::text||',"catalogo_version":1,"clave_idempotencia":'
   ||to_json(clave)::text||',"declaraciones":[]}';
  material_sha:=encode(sha256(convert_to(material,'UTF8')),'hex');
  solicitud:='solicitud_inscripcion_'||encode(sha256(convert_to(persona||chr(31)||sep||chr(31)||categoria,'UTF8')),'hex');
  recurso:='{"ambitos":{"candidato_ref":'||to_json(candidato)::text||'},"atributos":{"convocatoria_ref":'
   ||to_json(sep)::text||',"categoria_ref":'||to_json(categoria)::text
   ||',"catalogo_version":"1","material_sha256":"'||material_sha||'"}}';
  recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
  codigo:=NULL;
  BEGIN
   recibo:=vec_bolsa_llamamientos.solicitar_inscripcion_v1(material,
    jsonb_build_object('persona_ref',persona,'perfil_ref',perfil,'cuenta_ref',cuenta,'canal','externa_personal','idioma','es'),
    convert_to(recurso,'UTF8'),
    convert_to(jsonb_build_object('operacion','bolsa.inscripcion.presentar',
     'audiencia_consumo','vec_bolsa_llamamientos.inscripcion.presentar.v1',
     'efecto_ref',solicitud,'huella_efecto_sha256',recurso_sha)::text,'UTF8'),
    convert_to(jsonb_build_object('decision_ref','decision:b146:sep:'||vuelta,'principal_id',persona,
     'perfil_activo_ref',perfil,'correlacion_ref','correlacion_'||repeat('a',32),'concedida',true,
     'accion','bolsa.inscripcion.presentar','modulo_id','bolsa','tipo_recurso','inscripcion_convocatoria',
     'finalidad','presentar_inscripcion','recurso_ref',solicitud,'contexto_recurso_huella_sha256',recurso_sha,
     'vinculo_autenticacion_actor',jsonb_build_object('superficie','externa_personal'))::text,'UTF8'),
    convert_to('{}','UTF8'),
    convert_to(jsonb_build_object('principal_ref',persona,'perfil_activo_ref',perfil,'cuenta_ref',cuenta,
     'metodo','certificado','garantia','alto','contexto_actor_ref','vca_'||repeat('v',22),'contexto_version',1,
     'vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato','referencia',candidato)))::text,'UTF8'),
    1,1,convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to(repeat('a',44),'UTF8'));
  EXCEPTION WHEN OTHERS THEN codigo:=SQLSTATE;
  END;
  IF vuelta<3 AND (codigo IS NOT NULL OR recibo->>'solicitud_ref'<>solicitud OR recibo->>'repetida'<>'false')
  THEN RAISE EXCEPTION 'B146: separada % rechazada (%) %',categoria,codigo,recibo; END IF;
  IF vuelta=3 AND codigo IS DISTINCT FROM 'B9604'
  THEN RAISE EXCEPTION 'B146: segunda solicitud de la misma categoría devolvió %',codigo; END IF;
 END LOOP;
END $separada$;
RESET SESSION AUTHORIZATION;

-- Efectos: una fila por categoría, un recibo conjunto y accesos por decisión.
DO $efectos$
DECLARE
 conj text:=current_setting('b146.conjunta');
 sep text:=current_setting('b146.separada');
BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion WHERE convocatoria_ref=conj)<>2
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_version v
     JOIN vec_bolsa_llamamientos.solicitud_inscripcion s USING(solicitud_ref) WHERE s.convocatoria_ref=conj)<>2
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_outbox o
     JOIN vec_bolsa_llamamientos.solicitud_inscripcion s USING(solicitud_ref) WHERE s.convocatoria_ref=conj)<>2
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_conjunta WHERE convocatoria_ref=conj)<>1
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion_conjunta_categoria g
     JOIN vec_bolsa_llamamientos.solicitud_inscripcion_conjunta p USING(presentacion_ref)
     WHERE p.convocatoria_ref=conj)<>2
 OR (SELECT count(*) FILTER(WHERE a.resultado='confirmada')||'/'||count(*) FILTER(WHERE a.resultado='repetida')
     FROM vec_bolsa_llamamientos.solicitud_inscripcion_acceso a
     JOIN vec_bolsa_llamamientos.solicitud_inscripcion s USING(solicitud_ref) WHERE s.convocatoria_ref=conj)<>'2/2'
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.solicitud_inscripcion WHERE convocatoria_ref=sep)<>2
 THEN RAISE EXCEPTION 'B146: efectos divergentes'; END IF;
END $efectos$;

-- Proyección del aspirante: la conjunta presentada impide otra; la separada
-- con una categoría libre permite seguir. Etiquetas y cotejos fijos.
CREATE OR REPLACE FUNCTION vec_catalogos_configurables.leer_etiquetas_politicas_inscripcion_lote_v1(p_solicitudes jsonb,p_idioma text,p_canal text)
 RETURNS jsonb LANGUAGE sql AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object('categoria_ref',x->>'categoria_ref','categoria','Categoría B146',
  'politica_valida',true,'motivo_etiqueta_pendiente','Pendiente','motivo_etiqueta_cumple','Cumple',
  'impedimento_etiqueta_existente','Ya presentada','impedimento_etiqueta_politica','No disponible')),'[]'::jsonb)
 FROM jsonb_array_elements(p_solicitudes) x $f$;
CREATE OR REPLACE FUNCTION vec_catalogos_configurables.comprobar_ambitos_gestion_inscripcion_lote_v1(p_solicitudes jsonb)
 RETURNS jsonb LANGUAGE sql AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object('ambito_gestion_disponible',true)),'[]'::jsonb)
 FROM jsonb_array_elements(p_solicitudes) x $f$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $proyeccion$
DECLARE
 base jsonb; salida jsonb; conv text; modalidad text; categorias jsonb;
BEGIN
 FOREACH modalidad IN ARRAY ARRAY['conjunta','separada'] LOOP
  conv:=current_setting('b146.'||modalidad);
  categorias:=CASE modalidad WHEN 'conjunta'
   THEN '[{"categoria_ref":"cat.alpha"},{"categoria_ref":"cat.beta"},{"categoria_ref":"cat.gamma"}]'::jsonb
   ELSE '[{"categoria_ref":"cat.alpha"},{"categoria_ref":"cat.beta"}]'::jsonb END;
  base:=jsonb_build_object('convocatoria_ref',conv,'titulo','Convocatoria B146','resumen','Vector B146',
   'numero_categorias',jsonb_array_length(categorias),'modalidad_inscripcion',modalidad,
   'catalogo_ref','catalogo:b146','catalogo_version',1,'catalogo_sha256',repeat('a',64),
   'politica_catalogo_ref','politica:b146','politica_catalogo_version',1,'politica_catalogo_sha256',repeat('b',64),
   'plazo_abre_en',statement_timestamp()-interval '1 day','plazo_cierra_en',statement_timestamp()+interval '60 days',
   'categorias',categorias,'requisitos','[]'::jsonb);
  -- Conjunta: la persona a ya la presentó. Separada: la persona s tiene las dos.
  salida:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(base,
   CASE modalidad WHEN 'conjunta' THEN 'per_'||repeat('a',22) ELSE 'per_'||repeat('s',22) END,
   'es','certificado','alto',false,'externa_personal');
  IF salida->>'puede_iniciar'<>'false' OR salida->>'impedimento_etiqueta'<>'Ya presentada'
  OR jsonb_array_length(salida->'solicitudes_propias')<>2
  THEN RAISE EXCEPTION 'B146: ficha % con solicitudes propias divergente %',modalidad,salida; END IF;
  -- Otra persona sin solicitudes puede iniciar en ambas.
  salida:=vec_bolsa_llamamientos.proyectar_abiertas_inscripcion_interna_v1(base,'per_'||repeat('z',22),
   'es','certificado','alto',false,'externa_personal');
  IF salida->>'puede_iniciar'<>'true' OR jsonb_array_length(salida->'solicitudes_propias')<>0
  THEN RAISE EXCEPTION 'B146: ficha % libre divergente %',modalidad,salida; END IF;
 END LOOP;
END $proyeccion$;
ROLLBACK;
