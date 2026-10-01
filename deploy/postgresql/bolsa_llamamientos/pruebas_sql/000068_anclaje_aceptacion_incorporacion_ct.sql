\set ON_ERROR_STOP on
-- PG18 desechable, AD3-131/B67/B68 reales instaladas. Solo prueba.
-- El núcleo de consumo se sustituye por un doble en ROLLBACK; no se llama CTX.
-- No acredita COSE, identidad corporativa ni aceptación legal. Los negativos
-- de actor/perfil verifican la propagación al doble, no la criptografía real.
-- Los canónicos de la constitución B7 preexistente se reutilizan SIN inventar sus huellas.
BEGIN ISOLATION LEVEL READ COMMITTED;
CREATE ROLE vec_b68_guarda_ejecutor LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b68_guarda_ejecutor;
SET SESSION AUTHORIZATION vec_b68_guarda_ejecutor;
DO $g$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'FALLO aislamiento';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK READ COMMITTED denegado'; END;
END $g$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
CREATE ROLE vec_b68_guarda_ejecutor LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b68_guarda_ejecutor;
SET SESSION AUTHORIZATION vec_b68_guarda_ejecutor;
SET LOCAL transaction_read_only='on';
DO $g$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'FALLO READ ONLY';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK READ ONLY denegado'; END;
END $g$;
RESET SESSION AUTHORIZATION;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='3s';
CREATE ROLE vec_b68_prueba_ejecutor LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b68_prueba_ejecutor;
CREATE TEMP TABLE prueba_b68_consumos (capacidad bytea PRIMARY KEY,auditoria_ref text NOT NULL);
GRANT SELECT,INSERT ON pg_temp.prueba_b68_consumos TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,p_contexto_actor_canonico bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload_vec_ad_3 bytea,p_sobre_cose_sign1 bytea,p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $stub$
DECLARE c jsonb:=convert_from(p_capacidad_canonica,'UTF8')::jsonb;
 d jsonb:=convert_from(p_decision_canonica,'UTF8')::jsonb; contexto jsonb:=convert_from(p_contexto_actor_canonico,'UTF8')::jsonb; nueva boolean; a text:='auditoria-prueba:'||md5(p_capacidad_canonica::text);
BEGIN
 IF p_perfil_mutacion IS DISTINCT FROM 'consulta_anclaje_aceptacion_ct_bolsa' THEN RAISE EXCEPTION 'perfil de prueba incorrecto' USING ERRCODE='42501'; END IF;
 IF d->>'actor_ref' IS DISTINCT FROM contexto->>'actor_ref'
    OR d->>'perfil_ref' IS DISTINCT FROM 'perfil:prueba:68'
 THEN RAISE EXCEPTION 'actor/perfil de prueba incorrecto' USING ERRCODE='42501'; END IF;
 nueva:=NOT EXISTS (SELECT 1 FROM pg_temp.prueba_b68_consumos WHERE capacidad=p_capacidad_canonica);
 IF nueva THEN INSERT INTO pg_temp.prueba_b68_consumos VALUES(p_capacidad_canonica,a); END IF;
 RETURN QUERY SELECT 'decision-prueba:'||md5(p_capacidad_canonica::text),c->>'efecto_ref',c->>'huella_efecto_sha256',encode(sha256(p_capacidad_canonica),'hex'),a,clock_timestamp(),nueva;
END $stub$;
RESET ROLE;
CREATE FUNCTION pg_temp.debe_fallar(p_sql text,p_codigo text,p_etiqueta text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 BEGIN EXECUTE p_sql;
 EXCEPTION WHEN others THEN
  IF SQLSTATE<>p_codigo THEN RAISE EXCEPTION 'FALLO %: código % esperado %: %',p_etiqueta,SQLSTATE,p_codigo,SQLERRM; END IF;
  RAISE NOTICE 'OK %',p_etiqueta; RETURN;
 END;
 RAISE EXCEPTION 'FALLO %: no se rechazó',p_etiqueta;
END $f$;
REVOKE ALL ON FUNCTION pg_temp.debe_fallar(text,text,text) FROM PUBLIC;
CREATE FUNCTION pg_temp.material(p_s jsonb) RETURNS text LANGUAGE sql AS $m$
 SELECT '{'||string_agg(to_json(k)::text||':'||to_json(p_s->>k)::text,',' ORDER BY n)||'}'
 FROM unnest(ARRAY['esquema','unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','aceptacion_registro_sha256','apertura_operacion_ref','llamamiento_ref','propuesta_ref']) WITH ORDINALITY x(k,n)
$m$;
REVOKE ALL ON FUNCTION pg_temp.material(jsonb) FROM PUBLIC;
CREATE FUNCTION pg_temp.consultar(p_material text,p_nonce text,p_alteracion jsonb DEFAULT '{}'::jsonb,p_cap_alteracion jsonb DEFAULT '{}'::jsonb) RETURNS jsonb LANGUAGE plpgsql AS $c$
DECLARE s jsonb:=p_material::jsonb; d jsonb; c jsonb; h text;
BEGIN
 h:=encode(sha256(convert_to('{"ambitos":{"unidad_ref":'||to_json(s->>'unidad_ref')::text||'},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('actor_ref','actor:prueba:68','perfil_ref','perfil:prueba:68','accion','bolsa.aceptacion_ct.anclaje.consultar','modulo_id','bolsa','tipo_recurso','anclaje_aceptacion_ct','finalidad','preparar_incorporacion_ct','recurso_ref',s->>'aceptacion_operacion_ref','contexto_recurso_huella_sha256',h,'campos_permitidos','["anclaje"]'::jsonb,'obligaciones','[]'::jsonb)||p_alteracion;
 c:=jsonb_build_object('audiencia_consumo','vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1','operacion','bolsa.aceptacion_ct.anclaje.consultar','efecto_ref',s->>'aceptacion_operacion_ref','huella_efecto_sha256',h,'huella_decision_sha256',encode(sha256(convert_to(d::text,'UTF8')),'hex'),'nonce',p_nonce,'relleno',repeat('x',512))||p_cap_alteracion;
 RETURN vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(p_material,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'\x01',convert_to('{"actor_ref":"actor:prueba:68"}','UTF8'),1,1,'\x01','\x01','\x01',decode(repeat('a',88),'hex'));
END $c$;
REVOKE ALL ON FUNCTION pg_temp.consultar(text,text,jsonb,jsonb) FROM PUBLIC;

-- Fixture nueva de aceptación/apertura que conserva literalmente la instantánea
-- de una constitución B7 y su vínculo B8. Solo se inserta en este ROLLBACK.
-- No declara legal ni corporativo el origen de ese acta de ejercicio.
DO $fixture$
DECLARE modelo jsonb; original vec_bolsa_llamamientos.integracion_desarrollo; v record;
 ap jsonb; te jsonb; ip text; apt text; tet text; ordref text;
BEGIN
 SELECT * INTO STRICT original FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE tipo='aceptacion_rrhh' ORDER BY operacion_ref LIMIT 1;
 SELECT convert_from(registro_canonico,'UTF8')::jsonb INTO STRICT modelo FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref=original.apertura_operacion_ref;
 PERFORM set_config('prueba_b68.original_material',pg_temp.material(jsonb_build_object('esquema','vec.bolsa.anclaje-aceptacion-ct.consulta.v1','unidad_ref',modelo->>'unidad_ref','categoria_ref',modelo->>'categoria_ref','necesidad_ref',modelo->>'necesidad_ref','aceptacion_operacion_ref',original.operacion_ref,'aceptacion_registro_sha256',original.registro_huella_sha256,'apertura_operacion_ref',original.apertura_operacion_ref,'llamamiento_ref',modelo#>>'{llamamiento,LlamamientoRef}','propuesta_ref',modelo#>>'{llamamiento,PropuestaRef}')),true);
 SELECT vc.*,ct.categoria_ref,ct.bolsa_ref,ct.version_bolsa,io.instantanea_canonica,e.orden
 INTO STRICT v FROM vec_bolsa_llamamientos.vinculo_candidato vc
 JOIN vec_bolsa_llamamientos.constitucion ct ON ct.acta_ref=vc.acta_ref
 JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=vc.instantanea_ref AND e.version_instantanea=vc.version_instantanea AND e.participacion_ref=vc.participacion_ref
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa io ON io.instantanea_ref=ct.instantanea_ref AND io.version=ct.version_instantanea AND io.huella_instantanea_sha256=ct.huella_instantanea_sha256
 ORDER BY vc.participacion_ref LIMIT 1;
 ip:=convert_from(v.instantanea_canonica,'UTF8');
 PERFORM set_config('prueba_b68.participacion',v.participacion_ref,true);
 IF encode(sha256(v.instantanea_canonica),'hex')=ip::jsonb->>'huella_contenido_sha256' THEN RAISE EXCEPTION 'FALLO fixture no distingue las dos huellas'; END IF;
 ordref:=original.orden_operacion_ref;
 ap:=modelo||jsonb_build_object('operacion_ref','operacion:prueba:67:apertura','necesidad_ref','necesidad:prueba:67','categoria_ref',v.categoria_ref,'unidad_ref','unidad:prueba:67','instantanea','__INSTANTANEA_RAW__');
 ap:=jsonb_set(ap,'{fuente,datos,Necesidad,unidad_ref}',to_jsonb('unidad:prueba:67'::text));
 ap:=jsonb_set(ap,'{fuente,datos,Necesidad,categoria_ref}',to_jsonb(v.categoria_ref));
 ap:=jsonb_set(ap,'{fuente,datos,Bolsa,bolsa_ref}',to_jsonb(v.bolsa_ref));
 ap:=jsonb_set(ap,'{fuente,datos,Bolsa,version}',to_jsonb(v.version_bolsa));
 ap:=jsonb_set(ap,'{fuente,datos,Entradas}',ip::jsonb->'entradas');
 ap:=jsonb_set(ap,'{propuesta}',(modelo->'propuesta')||jsonb_build_object('propuesta_ref','propuesta:prueba:67','necesidad_ref','necesidad:prueba:67','bolsa_ref',v.bolsa_ref,'version_bolsa',v.version_bolsa,'huella_bolsa_sha256',ip::jsonb->>'huella_bolsa_sha256','instantanea_ref',v.instantanea_ref,'version_instantanea',v.version_instantanea,'huella_instantanea_sha256',ip::jsonb->>'huella_contenido_sha256','participacion_seleccionada_ref',v.participacion_ref,'orden_seleccionado',v.orden));
 ap:=jsonb_set(ap,'{llamamiento}',(modelo->'llamamiento')||jsonb_build_object('LlamamientoRef','llamamiento:prueba:67','BolsaRef',v.bolsa_ref,'PropuestaRef','propuesta:prueba:67','NecesidadRef','necesidad:prueba:67','Version',1));
 apt:=replace(ap::text,'"__INSTANTANEA_RAW__"',ip);
 te:=ap||jsonb_build_object('operacion_ref','operacion:prueba:67:aceptacion','tipo','aceptacion_rrhh','estado_llamamiento','aceptacion','resolucion',jsonb_build_object('apertura_operacion_ref','operacion:prueba:67:apertura'));
 te:=jsonb_set(te,'{llamamiento,Version}','2'::jsonb);
 tet:=replace(te::text,'"__INSTANTANEA_RAW__"',ip);
 INSERT INTO vec_bolsa_llamamientos.integracion_desarrollo(operacion_ref,tipo,necesidad_ref,version_necesidad,orden_operacion_ref,registro_canonico,registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en,apertura_operacion_ref)
 VALUES ('operacion:prueba:67:apertura','propuesta','necesidad:prueba:67',1,ordref,convert_to(apt,'UTF8'),encode(sha256(convert_to(apt,'UTF8')),'hex'),repeat('a',64),'decision:prueba:67:apertura','recibo:prueba:67:apertura',clock_timestamp(),NULL),
 ('operacion:prueba:67:aceptacion','aceptacion_rrhh','necesidad:prueba:67',1,ordref,convert_to(tet,'UTF8'),encode(sha256(convert_to(tet,'UTF8')),'hex'),repeat('b',64),'decision:prueba:67:aceptacion','recibo:prueba:67:aceptacion',clock_timestamp(),'operacion:prueba:67:apertura');
 INSERT INTO vec_bolsa_llamamientos.llamamiento_integracion_desarrollo VALUES ('llamamiento:prueba:67','operacion:prueba:67:apertura','propuesta:prueba:67',v.bolsa_ref,'necesidad:prueba:67',1,'abierto',clock_timestamp(),ap->'llamamiento');
 PERFORM set_config('prueba_b68.material',pg_temp.material(jsonb_build_object('esquema','vec.bolsa.anclaje-aceptacion-ct.consulta.v1','unidad_ref','unidad:prueba:67','categoria_ref',v.categoria_ref,'necesidad_ref','necesidad:prueba:67','aceptacion_operacion_ref','operacion:prueba:67:aceptacion','aceptacion_registro_sha256',encode(sha256(convert_to(tet,'UTF8')),'hex'),'apertura_operacion_ref','operacion:prueba:67:apertura','llamamiento_ref','llamamiento:prueba:67','propuesta_ref','propuesta:prueba:67')),true);
END $fixture$;
CREATE TEMP TABLE prueba_b68_historia AS
 SELECT 'integracion'::text AS tabla, count(*) AS filas, encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY operacion_ref)::text,'[]'),'UTF8')),'hex') AS huella
 FROM vec_bolsa_llamamientos.integracion_desarrollo t
 UNION ALL SELECT 'llamamiento',count(*),encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY operacion_ref)::text,'[]'),'UTF8')),'hex') FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo t
 UNION ALL SELECT 'vinculo',count(*),encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY participacion_ref)::text,'[]'),'UTF8')),'hex') FROM vec_bolsa_llamamientos.vinculo_candidato t;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO vec_b68_prueba_ejecutor;
SET SESSION AUTHORIZATION vec_b68_prueba_ejecutor;
DO $lecturas$
DECLARE m text:=current_setting('prueba_b68.material'); r jsonb; anterior jsonb; k text; nonce integer:=0;
BEGIN
 r:=pg_temp.consultar(m,'positivo');
 IF r->>'estado' IS DISTINCT FROM 'acreditado'
    OR (SELECT array_agg(x.key ORDER BY x.key) FROM jsonb_object_keys(r) x(key)) IS DISTINCT FROM ARRAY['anclaje','estado','evidencia']
    OR (SELECT count(*) FROM jsonb_object_keys(r->'anclaje'))<>10
    OR r#>>'{anclaje,apertura_operacion_ref}' IS DISTINCT FROM 'operacion:prueba:67:apertura'
    OR r#>>'{anclaje,aceptacion_recibo_ref}' IS DISTINCT FROM 'recibo:prueba:67:aceptacion'
    OR (r#>>'{anclaje,apertura_registro_sha256}')!~'^[0-9a-f]{64}$'
    OR r#>>'{evidencia,auditoria_ref}' IS NULL
 THEN RAISE EXCEPTION 'FALLO anclaje acreditado'; END IF;
 anterior:=r;
 r:=pg_temp.consultar(m,'positivo_fresco');
 IF r->'anclaje' IS DISTINCT FROM anterior->'anclaje'
    OR r#>>'{evidencia,auditoria_ref}' IS NOT DISTINCT FROM anterior#>>'{evidencia,auditoria_ref}'
 THEN RAISE EXCEPTION 'FALLO recibo original estable o auditoría fresca'; END IF;
 IF r::text LIKE '%can_%' OR r::text LIKE '%participacion_ref%' OR r::text LIKE '%sujeto_ref%' OR r::text LIKE '%persona%' OR r::text LIKE '%documento_identidad%' THEN RAISE EXCEPTION 'FALLO minimización'; END IF;
 r:=pg_temp.consultar(current_setting('prueba_b68.original_material'),'sin_vinculo');
 IF r->>'estado' IS DISTINCT FROM 'pendiente' OR r->'anclaje' IS DISTINCT FROM 'null'::jsonb THEN RAISE EXCEPTION 'FALLO aceptación histórica sin vínculo original B8'; END IF;
 FOREACH k IN ARRAY ARRAY['unidad_ref','categoria_ref','necesidad_ref','apertura_operacion_ref','llamamiento_ref','propuesta_ref','aceptacion_operacion_ref'] LOOP
  nonce:=nonce+1;
  r:=pg_temp.consultar(pg_temp.material(m::jsonb||jsonb_build_object(k,'ref:prueba:ajena')),'cruce:'||nonce);
  IF r->>'estado' IS DISTINCT FROM 'no_encontrada' OR r->'anclaje' IS DISTINCT FROM 'null'::jsonb OR r#>>'{evidencia,auditoria_ref}' IS NULL THEN RAISE EXCEPTION 'FALLO selector cruzado %',k; END IF;
 END LOOP;
 r:=pg_temp.consultar(pg_temp.material(m::jsonb||jsonb_build_object('aceptacion_registro_sha256',repeat('c',64))),'hash');
 IF r->>'estado' IS DISTINCT FROM 'no_encontrada' OR r->'anclaje' IS DISTINCT FROM 'null'::jsonb THEN RAISE EXCEPTION 'FALLO huella cruzada'; END IF;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',m,'positivo'),'42501','replay no consume otra lectura');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'campos','{"campos_permitidos":["persona"]}'),'42501','campo ajeno');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'obligaciones','{"obligaciones":["otra"]}'),'42501','obligaciones cruzadas');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'recurso','{"recurso_ref":"operacion:prueba:otra"}'),'42501','recurso cruzado');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'accion','{"accion":"bolsa.llamamiento.aceptacion_rrhh.registrar"}'),'42501','escritura no habilita lectura');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'tipo','{"tipo_recurso":"persona_aceptacion_ct"}'),'42501','tipo cruzado');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'finalidad','{"finalidad":"otra"}'),'42501','finalidad cruzada');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'actor','{"actor_ref":"actor:prueba:otro"}'),'42501','actor cruzado rechazado por doble central');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'perfil','{"perfil_ref":"perfil:prueba:otro"}'),'42501','perfil cruzado rechazado por doble central');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb,%L::jsonb)',m,'audiencia','{}','{"audiencia_consumo":"vec_bolsa_llamamientos.aceptacion_ct.persona.v1"}'),'42501','audiencia cruzada');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb,%L::jsonb)',m,'material_hash','{}','{"huella_efecto_sha256":"'||repeat('d',64)||'"}'),'42501','huella del material cruzada');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb,%L::jsonb)',m,'decision_hash','{}','{"huella_decision_sha256":"'||repeat('d',64)||'"}'),'42501','huella de la decisión cruzada');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',replace(m,'{"','{ "'),'canon'),'22023','material no canónico');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',m::jsonb::text,'orden'),'22023','orden JSON divergente');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',left(m,length(m)-1)||',"unidad_ref":"unidad:prueba:67"}','duplicada'),'22023','clave duplicada');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',pg_temp.material(m::jsonb||jsonb_build_object('aceptacion_registro_sha256',repeat('0',64))),'sha_cero'),'22023','SHA nulo');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',pg_temp.material(m::jsonb||jsonb_build_object('esquema','vec.bolsa.persona-aceptacion-ct.consulta.v1')),'esquema'),'22023','esquema ajeno');
 PERFORM pg_temp.debe_fallar('SELECT * FROM vec_bolsa_llamamientos.vinculo_candidato','42501','tabla B8 inaccesible para ejecutor');
 PERFORM pg_temp.debe_fallar('SELECT * FROM vec_autorizacion_atestada_v3.consumir_consulta_anclaje_aceptacion_ct_bolsa_v3_atestada(NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL)','42501','consumidor AD3 inaccesible para ejecutor');
 RAISE NOTICE 'OK anclaje exacto con recibo original, lecturas frescas, pendiente y ausencia';
END $lecturas$;
RESET SESSION AUTHORIZATION;
DO $auditoria$
DECLARE actual jsonb; previo jsonb;
BEGIN
 IF (SELECT count(*) FROM pg_temp.prueba_b68_consumos)<>11 THEN RAISE EXCEPTION 'FALLO consumo/auditoría'; END IF;
 SELECT jsonb_agg(to_jsonb(t) ORDER BY tabla) INTO previo FROM pg_temp.prueba_b68_historia t;
 SELECT jsonb_agg(to_jsonb(t) ORDER BY tabla) INTO actual FROM (
  SELECT 'integracion'::text AS tabla,count(*) AS filas,encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY operacion_ref)::text,'[]'),'UTF8')),'hex') AS huella FROM vec_bolsa_llamamientos.integracion_desarrollo t
  UNION ALL SELECT 'llamamiento',count(*),encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY operacion_ref)::text,'[]'),'UTF8')),'hex') FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo t
  UNION ALL SELECT 'vinculo',count(*),encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY participacion_ref)::text,'[]'),'UTF8')),'hex') FROM vec_bolsa_llamamientos.vinculo_candidato t
 ) t;
 IF actual IS DISTINCT FROM previo THEN RAISE EXCEPTION 'FALLO consulta modificó negocio'; END IF;
 RAISE NOTICE 'OK once lecturas auditadas; historia de negocio idéntica';
END $auditoria$;
ROLLBACK;
