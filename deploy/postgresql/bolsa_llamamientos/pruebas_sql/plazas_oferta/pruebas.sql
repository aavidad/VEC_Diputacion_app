-- Pruebas funcionales de Bolsa 000058 (ofertas con varias plazas). Parte de
-- los datos de disposicion_oferta/datos.sql: bolsa:disp:1 con cuatro
-- participaciones en orden 1..4; part:disp:3 es el candidato A, part:disp:2 el
-- B y part:disp:4 el D. Los actos se registran con el rol ejecutor real; la
-- autorización AD3 es un doble. Las ofertas de vencimiento corto se siembran
-- como superusuario (sin disparadores), porque la política exige al menos una
-- hora de plazo.
\set ON_ERROR_STOP on
CREATE SCHEMA prueba_plazas;
CREATE FUNCTION prueba_plazas.espera(p_sql text, p_codigo text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'se esperaba % en: %', p_codigo, p_sql;
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE <> p_codigo THEN RAISE EXCEPTION 'código % (%) en lugar de % en: %', SQLSTATE, SQLERRM, p_codigo, p_sql; END IF;
END $f$;

-- Publica una versión de la política de ofertas de bolsa:disp:1.
CREATE FUNCTION prueba_plazas.politica(p_version_esperada bigint, p_politica jsonb, p_clave text) RETURNS jsonb
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT (vec_bolsa_llamamientos.publicar_politica_ofertas_v1('bolsa:disp:1',p_version_esperada,p_politica,'per_actoractoractoractoractor',p_clave,
  'recibo:politica-ofertas:'||encode(sha256(convert_to(p_clave,'UTF8')),'hex'),
  convert_to('{"efecto_ref":"bolsa:disp:1"}','UTF8'),
  convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"bolsa.politica_ofertas.publicar","modulo_id":"bolsa","tipo_recurso":"bolsa_constituida","finalidad":"gobierno_politica_ofertas_bolsa","recurso_ref":"bolsa:disp:1"}','UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00')).politica
$f$;
ALTER FUNCTION prueba_plazas.politica(bigint,jsonb,text) OWNER TO vec_bolsa_llamamientos_ejecutor;

-- Versión vigente de la política, leída como superusuario (B51 cerró la
-- lectura directa al ejecutor).
CREATE FUNCTION prueba_plazas.politica_vigente() RETURNS jsonb LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT vec_bolsa_llamamientos.leer_politica_ofertas_v1('bolsa:disp:1')
$f$;

-- Publica una oferta real (publicar_oferta_v3) con la política vigente, en días
-- hábiles, autorizando p_autorizadas plazas y pidiendo p_numero.
CREATE FUNCTION prueba_plazas.publicar(p_clave text, p_numero integer, p_autorizadas integer)
RETURNS TABLE(oferta jsonb, reutilizada boolean) LANGUAGE plpgsql AS $f$
DECLARE v jsonb; plazo jsonb; material_h text; contexto_h text; decof bytea;
 publicada timestamptz:=clock_timestamp();
 ultimo text:=((clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date+1)::text;
 vence timestamptz:=(((clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date+2)::timestamp AT TIME ZONE 'Europe/Madrid');
 sufijo text:=encode(sha256(convert_to(p_clave,'UTF8')),'hex');
BEGIN
 v:=prueba_plazas.politica_vigente();
 plazo:=jsonb_build_object('regla_ref','politica-ofertas:bolsa:disp:1:'||(v->>'version'),'huella_catalogo',v->>'huella_sha256',
  'unidad','dias_habiles','cantidad',2,'computo','administrativo','municipio_sede','18087',
  'ultimo_dia',ultimo,'ejemplo',true,'politica_version',(v->>'version')::bigint,'calendarios',jsonb_build_array('cal:1'));
 material_h:=encode(sha256(convert_to(array_to_string(ARRAY['bolsa:disp:1',
  to_char(publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  plazo->>'regla_ref',plazo->>'huella_catalogo',plazo->>'unidad',plazo->>'cantidad',
  plazo->>'computo',plazo->>'municipio_sede',plazo->>'ultimo_dia',plazo->>'politica_version','cal:1',p_autorizadas::text],chr(31)),'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 decof:=convert_to((jsonb_build_object('principal_id','per_actoractoractoractoractor','accion','llamamiento.emitir.v1',
  'modulo_id','bolsa','tipo_recurso','bolsa_constituida','finalidad','gestion_llamamientos_bolsa',
  'recurso_ref','bolsa:disp:1','contexto_recurso_huella_sha256',contexto_h))::text,'UTF8');
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.publicar_oferta_v3('oferta:'||sufijo,'recibo:oferta:'||sufijo,
  'bolsa:disp:1','per_actoractoractoractoractor',p_clave,
  '{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","descripcion":"Sustitución por baja"}'::jsonb,
  plazo,publicada,vence,
  convert_to('{"efecto_ref":"bolsa:disp:1","nonce":"'||gen_random_uuid()||'"}','UTF8'),decof,
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00','unidad:rrhh','ambito:bolsa',p_numero);
END $f$;
ALTER FUNCTION prueba_plazas.publicar(text,integer,integer) OWNER TO vec_bolsa_llamamientos_ejecutor;
ALTER FUNCTION prueba_plazas.publicar(text,integer,integer) SECURITY DEFINER;

-- Oferta con vencimiento en segundos (sembrada sin disparadores).
CREATE FUNCTION prueba_plazas.oferta_rapida(p_sufijo text, p_version bigint, p_numero integer, p_segundos numeric) RETURNS text
LANGUAGE plpgsql AS $f$
DECLARE r text:='oferta:'||encode(sha256(convert_to(p_sufijo,'UTF8')),'hex');
BEGIN
 PERFORM set_config('session_replication_role','replica',true);
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada(oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(r,'recibo:oferta:'||substr(r,8),'bolsa:disp:1','per_actoractoractoractoractor','clave-'||p_sufijo,
  '{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-01","descripcion":"Sustitución"}'::jsonb,
  jsonb_build_object('politica_version',p_version,'unidad','horas_naturales','ejemplo',true),
  clock_timestamp(),clock_timestamp()+p_segundos*interval '1 second',repeat('e',64),'decision:siembra:'||p_sufijo);
 IF p_numero IS NOT NULL THEN
  INSERT INTO vec_bolsa_llamamientos.plazas_oferta VALUES(r,'bolsa:disp:1',p_numero,p_version,clock_timestamp());
 END IF;
 PERFORM set_config('session_replication_role','origin',true);
 RETURN r;
END $f$;

CREATE FUNCTION prueba_plazas.disposicion(p_oferta text, p_participacion text) RETURNS void LANGUAGE sql AS $f$
 INSERT INTO vec_bolsa_llamamientos.disposicion_oferta VALUES(p_oferta,p_participacion,clock_timestamp(),'clave-'||p_participacion,
  'recibo:disposicion:'||encode(sha256(convert_to(p_oferta||p_participacion,'UTF8')),'hex'))
$f$;

-- Registra un acto con el rol ejecutor real.
CREATE FUNCTION prueba_plazas.acto(p_oferta text, p_plaza integer, p_tipo text, p_participacion text, p_secuencia integer, p_clave text)
RETURNS jsonb LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT r.oferta||jsonb_build_object('reutilizada',r.reutilizada) FROM vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(
  p_oferta,'recibo:plaza-oferta:'||encode(sha256(convert_to(p_oferta||chr(31)||p_clave,'UTF8')),'hex'),'bolsa:disp:1',
  p_plaza,p_tipo,p_participacion,p_secuencia,'per_actoractoractoractoractor',p_clave,
  convert_to('{"efecto_ref":"bolsa:disp:1","nonce":"'||gen_random_uuid()||'"}','UTF8'),
  convert_to('{"principal_id":"per_actoractoractoractoractor","accion":"llamamiento.emitir.v1","modulo_id":"bolsa","finalidad":"gestion_llamamientos_bolsa","recurso_ref":"bolsa:disp:1","tipo_recurso":"bolsa_constituida"}','UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00') r
$f$;
ALTER FUNCTION prueba_plazas.acto(text,integer,text,text,integer,text) OWNER TO vec_bolsa_llamamientos_ejecutor;

CREATE FUNCTION prueba_plazas.plaza(p_oferta text, p_n integer) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp())->'plazas'->(p_n-1)
$f$;
CREATE FUNCTION prueba_plazas.estado(p_oferta text) RETURNS text LANGUAGE sql AS $f$
 SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp())->>'estado'
$f$;
CREATE FUNCTION prueba_plazas.mi_bolsa(p_candidato text, p_oferta text) RETURNS text LANGUAGE sql AS $f$
 SELECT x->>'estado' FROM jsonb_array_elements(prueba_disp.ofertas(p_candidato)) x WHERE x->>'oferta_ref'=p_oferta
$f$;
GRANT USAGE ON SCHEMA prueba_plazas TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION prueba_plazas.acto(text,integer,text,text,integer,text),prueba_plazas.publicar(text,integer,integer),
 prueba_plazas.politica_vigente(),
 prueba_plazas.politica(bigint,jsonb,text) TO vec_bolsa_llamamientos_ejecutor;

-- 1. Vector Go/SQL del material de publicación con plazas (mismo orden que
-- huellaMaterialPlazoOfertaConPlazas en la aplicación).
DO $vector$
DECLARE h text;
BEGIN
 h:=encode(sha256(convert_to(array_to_string(ARRAY['bolsa:of:1',
  to_char('2026-09-25T10:00:00.123456Z'::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char('2026-09-29T22:00:00Z'::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'politica-ofertas:bolsa:of:1:1',repeat('a',64),'dias_habiles','2','administrativo','18087','2026-09-29','1',
  'cal:1'||chr(30)||'cal:2','3'],chr(31)),'UTF8')),'hex');
 IF h<>'f6aa0fb741632ea4c51c73603ddde739c5c89e8498ff40be5be7f850fff4de34' THEN RAISE EXCEPTION 'vector Go/SQL de material con plazas divergente: %',h; END IF;
 RAISE NOTICE 'vector Go/SQL del material con plazas OK';
END $vector$;

-- 2. Política y publicación ligada al número de plazas.
DO $p$
DECLARE sin_plazas jsonb:='{"plazo":{"unidad":"dias_habiles","cantidad":2,"computo":"administrativo","municipio_sede":"18087"},"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo"},"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"}}';
 con_plazas jsonb; r record; v jsonb;
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 v:=prueba_plazas.politica(0,sin_plazas,'clave-politica-1');
 IF v->>'version'<>'1' THEN RAISE EXCEPTION 'política 1: %',v; END IF;
 PERFORM prueba_plazas.espera($$SELECT * FROM prueba_plazas.publicar('clave-pub-dos',2,2)$$,'VBO08');
 SELECT * INTO STRICT r FROM prueba_plazas.publicar('clave-pub-una',1,1);
 IF r.reutilizada OR r.oferta->>'numero_plazas'<>'1' OR jsonb_array_length(r.oferta->'plazas')<>1 OR r.oferta->>'estado'<>'abierta'
    OR r.oferta->'politica_plazas'<>'null'::jsonb THEN RAISE EXCEPTION 'oferta de una plaza: %',r.oferta; END IF;
 SELECT * INTO STRICT r FROM prueba_plazas.publicar('clave-pub-una',1,1);
 IF NOT r.reutilizada THEN RAISE EXCEPTION 'replay de publicación no reutilizado'; END IF;
 -- Políticas con plazas mal formadas.
 con_plazas:=sin_plazas||'{"plazas":{"llamada":"simultanea","respuesta_horas":1,"tras_renuncia":"siguiente_en_orden"}}';
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x1')$$,con_plazas||'{"plazas":{"llamada":"simultanea","respuesta_horas":0,"tras_renuncia":"siguiente_en_orden"}}'),'22023');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x2')$$,con_plazas||'{"plazas":{"llamada":"simultanea","respuesta_horas":721,"tras_renuncia":"siguiente_en_orden"}}'),'22023');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x3')$$,con_plazas||'{"plazas":{"llamada":"simultanea","respuesta_horas":1.5,"tras_renuncia":"siguiente_en_orden"}}'),'22023');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x4')$$,con_plazas||'{"plazas":{"llamada":"simultanea","respuesta_horas":"24","tras_renuncia":"siguiente_en_orden"}}'),'22023');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x5')$$,con_plazas||'{"plazas":{"llamada":"al_azar","respuesta_horas":24,"tras_renuncia":"siguiente_en_orden"}}'),'22023');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x6')$$,con_plazas||'{"plazas":{"llamada":"simultanea","respuesta_horas":24,"tras_renuncia":"siguiente_en_orden","otra":1}}'),'22023');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x7')$$,sin_plazas||'{"otra":{}}'),'22023');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.politica(1,%L,'clave-politica-x8')$$,con_plazas||'{"otra":{}}'),'22023');
 v:=prueba_plazas.politica(1,con_plazas||'{"plazas":{"llamada":"simultanea","respuesta_horas":720,"tras_renuncia":"siguiente_en_orden"}}','clave-politica-2');
 IF v->>'version'<>'2' OR v#>>'{politica,plazas,respuesta_horas}'<>'720' THEN RAISE EXCEPTION 'política 2: %',v; END IF;
 -- El número de plazas es parte de lo autorizado.
 PERFORM prueba_plazas.espera($$SELECT * FROM prueba_plazas.publicar('clave-pub-tres-x',2,3)$$,'42501');
 SELECT * INTO STRICT r FROM prueba_plazas.publicar('clave-pub-tres',3,3);
 IF r.reutilizada OR r.oferta->>'numero_plazas'<>'3' OR jsonb_array_length(r.oferta->'plazas')<>3
    OR r.oferta#>>'{politica_plazas,llamada}'<>'simultanea' THEN RAISE EXCEPTION 'oferta de tres plazas: %',r.oferta; END IF;
 PERFORM prueba_plazas.espera($$SELECT * FROM prueba_plazas.publicar('clave-pub-tres',2,2)$$,'VBO01');
 -- El ejecutor no alcanza las vías cerradas ni la proyección interna.
 IF has_function_privilege('vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz)','EXECUTE')
    OR has_table_privilege('vec_bolsa_llamamientos.acto_plaza_oferta','SELECT') THEN
  RAISE EXCEPTION 'el ejecutor alcanza vías cerradas';
 END IF;
 RESET ROLE;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.plazas_oferta)<>2
    OR (SELECT numero_plazas FROM vec_bolsa_llamamientos.plazas_oferta WHERE politica_version=2)<>3 THEN
  RAISE EXCEPTION 'filas de plazas';
 END IF;
 RAISE NOTICE 'política con plazas, publicación ligada al número de plazas, replay y ACL OK';
END $p$;

-- 3. Llamada simultánea: por orden, respuesta, renuncia y siguiente, y
-- llamamiento directo cuando se agotan las disposiciones.
CREATE TABLE prueba_plazas.ofertas(nombre text PRIMARY KEY, oferta_ref text NOT NULL);
DO $s$
DECLARE oa text; ob text; oc text; oc2 text; od text; oe text; oabierta text; j jsonb; p jsonb; v jsonb;
 base jsonb:='{"plazo":{"unidad":"dias_habiles","cantidad":2,"computo":"administrativo","municipio_sede":"18087"},"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo"},"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"}}';
BEGIN
 -- Versión 3: llamada sucesiva y llamamiento directo tras renuncia.
 v:=prueba_plazas.politica(2,base||'{"plazas":{"llamada":"sucesiva","respuesta_horas":1,"tras_renuncia":"llamamiento_directo"}}','clave-politica-3');
 IF v->>'version'<>'3' THEN RAISE EXCEPTION 'política 3: %',v; END IF;
 v:=prueba_plazas.politica(3,base||'{"plazas":{"llamada":"simultanea","respuesta_horas":1,"tras_renuncia":"siguiente_en_orden"}}','clave-politica-4');
 oa:=prueba_plazas.oferta_rapida('oa',4,2,3);
 ob:=prueba_plazas.oferta_rapida('ob',3,2,3);
 oc:=prueba_plazas.oferta_rapida('oc',1,NULL,3);
 oc2:=prueba_plazas.oferta_rapida('oc2',1,NULL,3);
 od:=prueba_plazas.oferta_rapida('od',1,NULL,3);
 oe:=prueba_plazas.oferta_rapida('oe',4,1,3);
 oabierta:=prueba_plazas.oferta_rapida('oabierta',4,2,3600);
 INSERT INTO prueba_plazas.ofertas VALUES('oa',oa),('ob',ob),('oe',oe);
 PERFORM prueba_plazas.disposicion(oa,x) FROM unnest(ARRAY['part:disp:3','part:disp:1','part:disp:2']) x;
 PERFORM prueba_plazas.disposicion(ob,x) FROM unnest(ARRAY['part:disp:2','part:disp:4']) x;
 PERFORM prueba_plazas.disposicion(oc,'part:disp:1');
 PERFORM prueba_plazas.disposicion(od,'part:disp:2');
 PERFORM prueba_plazas.disposicion(oe,x) FROM unnest(ARRAY['part:disp:1','part:disp:2']) x;
 PERFORM prueba_plazas.disposicion(oabierta,'part:disp:1');
 -- Antes de vencer no hay propuesta ni actos.
 IF prueba_plazas.estado(oa)<>'abierta' OR prueba_plazas.plaza(oa,1)->'propuesta'<>'null'::jsonb THEN RAISE EXCEPTION 'abierta: %',prueba_plazas.plaza(oa,1); END IF;
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'adjudicada','part:disp:1',0,'clave-abierta-1')$$,oabierta),'VBO03');
 RAISE NOTICE 'plazo abierto rechaza actos OK';
END $s$;
SELECT pg_sleep(3.5);
DO $a$
DECLARE oa text:=(SELECT oferta_ref FROM prueba_plazas.ofertas WHERE nombre='oa'); j jsonb; p jsonb;
BEGIN
 -- Tras el plazo, cada plaza propone a la siguiente persona por orden.
 IF prueba_plazas.estado(oa)<>'pendiente_resolucion'
    OR prueba_plazas.plaza(oa,1)->'propuesta'<>'{"tipo":"adjudicar","orden_vigente":1,"participacion_ref":"part:disp:1"}'::jsonb
    OR prueba_plazas.plaza(oa,2)->'propuesta'<>'{"tipo":"adjudicar","orden_vigente":2,"participacion_ref":"part:disp:2"}'::jsonb THEN
  RAISE EXCEPTION 'propuestas iniciales: % / %',prueba_plazas.plaza(oa,1),prueba_plazas.plaza(oa,2);
 END IF;
 -- Se puede confirmar la plaza 2 primero; la 1 conserva su propuesta.
 j:=prueba_plazas.acto(oa,2,'adjudicada','part:disp:2',0,'clave-oa-1');
 IF (j->>'reutilizada')::boolean OR j->'plazas'->1->>'estado'<>'pendiente_respuesta' OR j->>'estado'<>'en_curso'
    OR j->'plazas'->0->'propuesta'->>'participacion_ref'<>'part:disp:1' THEN RAISE EXCEPTION 'adjudicar plaza 2: %',j; END IF;
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'adjudicada','part:disp:3',0,'clave-oa-x1')$$,oa),'VBO04');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'adjudicada','part:disp:1',1,'clave-oa-x2')$$,oa),'VBO04');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'llamamiento_directo',NULL,0,'clave-oa-x3')$$,oa),'VBO04');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,3,'adjudicada','part:disp:1',0,'clave-oa-x4')$$,oa),'22023');
 j:=prueba_plazas.acto(oa,1,'adjudicada','part:disp:1',0,'clave-oa-2');
 p:=j->'plazas'->0;
 IF p->>'estado'<>'pendiente_respuesta' OR p->>'participacion_ref'<>'part:disp:1' OR (p->>'puede_sin_respuesta')::boolean
    OR (p->>'responder_antes_de')::timestamptz NOT BETWEEN clock_timestamp()+interval '59 minutes' AND clock_timestamp()+interval '61 minutes' THEN
  RAISE EXCEPTION 'adjudicar plaza 1: %',p;
 END IF;
 -- Sin respuesta solo al vencer su plazo; la respuesta es de quien ocupa la plaza.
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'sin_respuesta','part:disp:1',1,'clave-oa-x5')$$,oa),'VBO07');
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'aceptada','part:disp:2',1,'clave-oa-x6')$$,oa),'VBO04');
 -- Renuncia: la plaza queda libre y propone a la siguiente persona no llamada.
 j:=prueba_plazas.acto(oa,1,'renuncia','part:disp:1',1,'clave-oa-3');
 p:=j->'plazas'->0;
 IF p->>'estado'<>'vacante' OR p->'propuesta'<>'{"tipo":"adjudicar","orden_vigente":3,"participacion_ref":"part:disp:3"}'::jsonb
    OR jsonb_array_length(p->'historial')<>2 THEN RAISE EXCEPTION 'renuncia: %',p; END IF;
 -- Replay exacto sin efectos; misma clave con otro acto, conflicto.
 j:=prueba_plazas.acto(oa,1,'renuncia','part:disp:1',1,'clave-oa-3');
 IF NOT (j->>'reutilizada')::boolean THEN RAISE EXCEPTION 'replay de renuncia'; END IF;
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'aceptada','part:disp:1',1,'clave-oa-3')$$,oa),'VBO01');
 j:=prueba_plazas.acto(oa,2,'aceptada','part:disp:2',1,'clave-oa-4');
 IF j->'plazas'->1->>'estado'<>'cubierta' THEN RAISE EXCEPTION 'aceptada: %',j; END IF;
 j:=prueba_plazas.acto(oa,1,'adjudicada','part:disp:3',2,'clave-oa-5');
 j:=prueba_plazas.acto(oa,1,'renuncia','part:disp:3',3,'clave-oa-6');
 -- Sin más disposiciones, VEC propone el llamamiento directo para esa plaza.
 IF j->'plazas'->0->'propuesta'<>'{"tipo":"llamamiento_directo"}'::jsonb THEN RAISE EXCEPTION 'agotadas: %',j->'plazas'->0; END IF;
 j:=prueba_plazas.acto(oa,1,'llamamiento_directo',NULL,4,'clave-oa-7');
 IF j->>'estado'<>'cerrada' OR j->'plazas'->0->>'estado'<>'llamamiento_directo' OR jsonb_array_length(j->'plazas'->0->'historial')<>5 THEN
  RAISE EXCEPTION 'cerrada: %',j;
 END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta WHERE oferta_ref=oa)<>7
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.acto_plaza_oferta_outbox WHERE oferta_ref=oa)<>7
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.acto_plaza_oferta WHERE oferta_ref=oa AND (auditoria_ref IS NULL OR decision_ref IS NULL)) THEN
  RAISE EXCEPTION 'actos, auditoría u outbox de la oferta simultánea';
 END IF;
 -- «Mi bolsa»: quien renunció la ve resuelta; quien aceptó, adjudicada a su nombre.
 IF prueba_plazas.mi_bolsa('can_disposicion_candidato_A01',oa)<>'resuelta'
    OR prueba_plazas.mi_bolsa('can_disposicion_candidato_B01',oa)<>'adjudicada_propia'
    OR prueba_plazas.mi_bolsa('can_disposicion_candidato_D01',oa) IS NOT NULL THEN
  RAISE EXCEPTION 'Mi bolsa de la oferta simultánea';
 END IF;
 RAISE NOTICE 'llamada simultánea: orden, respuesta, renuncia y siguiente, llamamiento directo, replay y Mi bolsa OK';
END $a$;

-- 4. Llamada sucesiva con llamamiento directo tras renuncia o falta de respuesta.
DO $b$
DECLARE ob text:=(SELECT oferta_ref FROM prueba_plazas.ofertas WHERE nombre='ob'); j jsonb;
BEGIN
 IF prueba_plazas.plaza(ob,1)->'propuesta'->>'participacion_ref'<>'part:disp:2' OR prueba_plazas.plaza(ob,2)->'propuesta'<>'null'::jsonb THEN
  RAISE EXCEPTION 'sucesiva inicial: % / %',prueba_plazas.plaza(ob,1),prueba_plazas.plaza(ob,2);
 END IF;
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,2,'adjudicada','part:disp:4',0,'clave-ob-x1')$$,ob),'VBO04');
 -- Adjudicación cuyo plazo de respuesta ya venció (sembrada con su outbox).
 PERFORM set_config('session_replication_role','replica',true);
 INSERT INTO vec_bolsa_llamamientos.acto_plaza_oferta VALUES(ob,1,1,'adjudicada','part:disp:2',2,clock_timestamp()-interval '1 second',
  'recibo:plaza-oferta:'||repeat('b',64),'per_actoractoractoractoractor','clave-ob-siembra',repeat('f',64),'decision:siembra:ob','auditoria:siembra',clock_timestamp()-interval '2 seconds');
 INSERT INTO vec_bolsa_llamamientos.acto_plaza_oferta_outbox VALUES('recibo:plaza-oferta:'||repeat('b',64),ob,'bolsa:disp:1',1,1,'adjudicada',clock_timestamp()-interval '2 seconds',NULL);
 PERFORM set_config('session_replication_role','origin',true);
 IF prueba_plazas.plaza(ob,1)->>'estado'<>'pendiente_respuesta' OR NOT (prueba_plazas.plaza(ob,1)->>'puede_sin_respuesta')::boolean
    OR prueba_plazas.plaza(ob,2)->'propuesta'<>'null'::jsonb THEN RAISE EXCEPTION 'pendiente sucesiva: %',prueba_plazas.plaza(ob,1); END IF;
 j:=prueba_plazas.acto(ob,1,'sin_respuesta','part:disp:2',1,'clave-ob-1');
 -- La política manda llamamiento directo tras la falta de respuesta, aunque
 -- quede una disposición; la plaza 2 espera a que se resuelva la 1.
 IF j->'plazas'->0->'propuesta'<>'{"tipo":"llamamiento_directo"}'::jsonb OR j->'plazas'->1->'propuesta'<>'null'::jsonb THEN
  RAISE EXCEPTION 'tras sin respuesta: %',j->'plazas';
 END IF;
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'adjudicada','part:disp:4',2,'clave-ob-x2')$$,ob),'VBO04');
 j:=prueba_plazas.acto(ob,1,'llamamiento_directo',NULL,2,'clave-ob-2');
 IF j->'plazas'->1->'propuesta'<>'{"tipo":"adjudicar","orden_vigente":4,"participacion_ref":"part:disp:4"}'::jsonb THEN
  RAISE EXCEPTION 'plaza 2 tras directo: %',j->'plazas'->1;
 END IF;
 j:=prueba_plazas.acto(ob,2,'adjudicada','part:disp:4',0,'clave-ob-3');
 j:=prueba_plazas.acto(ob,2,'aceptada','part:disp:4',1,'clave-ob-4');
 IF j->>'estado'<>'cerrada' THEN RAISE EXCEPTION 'sucesiva final: %',j; END IF;
 IF prueba_plazas.mi_bolsa('can_disposicion_candidato_D01',ob)<>'adjudicada_propia'
    OR prueba_plazas.mi_bolsa('can_disposicion_candidato_B01',ob)<>'resuelta' THEN
  RAISE EXCEPTION 'Mi bolsa de la oferta sucesiva';
 END IF;
 RAISE NOTICE 'llamada sucesiva, falta de respuesta y llamamiento directo por política OK';
END $b$;

-- 5. Ofertas con política sin plazas y resoluciones anteriores.
DO $c$
DECLARE oc text:='oferta:'||encode(sha256(convert_to('oc','UTF8')),'hex'); oc2 text:='oferta:'||encode(sha256(convert_to('oc2','UTF8')),'hex');
 od text:='oferta:'||encode(sha256(convert_to('od','UTF8')),'hex'); j jsonb;
BEGIN
 j:=prueba_plazas.acto(oc,1,'adjudicada','part:disp:1',0,'clave-oc-1');
 IF j->>'estado'<>'adjudicada' OR j->'plazas'->0->>'estado'<>'cubierta' OR j->'plazas'->0->'responder_antes_de'<>'null'::jsonb THEN
  RAISE EXCEPTION 'sin plazas: %',j;
 END IF;
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'aceptada','part:disp:1',1,'clave-oc-x1')$$,oc),'VBO04');
 IF prueba_plazas.plaza(oc2,1)->'propuesta'<>'{"tipo":"llamamiento_directo"}'::jsonb THEN RAISE EXCEPTION 'sin disposiciones: %',prueba_plazas.plaza(oc2,1); END IF;
 j:=prueba_plazas.acto(oc2,1,'llamamiento_directo',NULL,0,'clave-oc2-1');
 IF j->>'estado'<>'llamamiento_directo' THEN RAISE EXCEPTION 'directo: %',j; END IF;
 -- Resolución única anterior (B28): se muestra como su plaza y no admite actos.
 PERFORM set_config('session_replication_role','replica',true);
 INSERT INTO vec_bolsa_llamamientos.resolucion_oferta VALUES(od,'recibo:resolucion-oferta:'||repeat('d',64),'adjudicada','part:disp:2',2,1,
  'per_actoractoractoractoractor','clave-od-antigua',clock_timestamp(),'decision:siembra:od');
 PERFORM set_config('session_replication_role','origin',true);
 IF prueba_plazas.estado(od)<>'adjudicada' OR prueba_plazas.plaza(od,1)->>'estado'<>'cubierta' THEN RAISE EXCEPTION 'resolución anterior'; END IF;
 PERFORM prueba_plazas.espera(format($$SELECT prueba_plazas.acto(%L,1,'aceptada','part:disp:2',0,'clave-od-1')$$,od),'VBO02');
 RAISE NOTICE 'política sin plazas, llamamiento directo sin disposiciones y resolución anterior OK';
END $c$;

-- 6. Historia inmutable.
SET ROLE vec_bolsa_llamamientos_propietario;
DO $i$ BEGIN
 BEGIN UPDATE vec_bolsa_llamamientos.acto_plaza_oferta SET tipo='aceptada'; RAISE EXCEPTION 'debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN DELETE FROM vec_bolsa_llamamientos.acto_plaza_oferta; RAISE EXCEPTION 'debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN UPDATE vec_bolsa_llamamientos.plazas_oferta SET numero_plazas=9; RAISE EXCEPTION 'debe ser inmutable'; EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 RAISE NOTICE 'historia de plazas inmutable OK';
END $i$;
RESET ROLE;

-- Funciones para la concurrencia y el reinicio que lanza el guion.
CREATE FUNCTION prueba_plazas.acto_conc(p_clave text) RETURNS text LANGUAGE sql AS $f$
 SELECT prueba_plazas.acto((SELECT oferta_ref FROM prueba_plazas.ofertas WHERE nombre='oe'),1,'adjudicada','part:disp:1',0,p_clave)->>'estado'
$f$;
CREATE FUNCTION prueba_plazas.replay_recibo() RETURNS text LANGUAGE sql AS $f$
 SELECT (j->'plazas'->0->'historial'->1->>'recibo_ref')||'|'||(j->>'reutilizada')
   FROM prueba_plazas.acto((SELECT oferta_ref FROM prueba_plazas.ofertas WHERE nombre='oa'),1,'renuncia','part:disp:1',1,'clave-oa-3') j
$f$;
GRANT SELECT ON prueba_plazas.ofertas TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION prueba_plazas.acto_conc(text),prueba_plazas.replay_recibo() TO vec_bolsa_llamamientos_ejecutor;
