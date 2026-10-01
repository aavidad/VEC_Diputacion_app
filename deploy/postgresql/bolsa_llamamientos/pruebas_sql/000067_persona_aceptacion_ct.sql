\set ON_ERROR_STOP on
-- PG18 desechable, AD3-128/CTX18/B67 reales instaladas. Solo prueba.
-- El núcleo de consumo y CTX18 se sustituyen por dobles en ROLLBACK;
-- no acredita COSE, persona corporativa ni aceptación legal. Los canónicos
-- de la constitución B7 preexistente se reutilizan SIN inventar sus huellas.
BEGIN ISOLATION LEVEL READ COMMITTED;
CREATE ROLE vec_b67_guarda_ejecutor LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b67_guarda_ejecutor;
SET SESSION AUTHORIZATION vec_b67_guarda_ejecutor;
DO $g$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'FALLO aislamiento';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK READ COMMITTED denegado'; END;
END $g$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
CREATE ROLE vec_b67_guarda_ejecutor LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b67_guarda_ejecutor;
SET SESSION AUTHORIZATION vec_b67_guarda_ejecutor;
SET LOCAL transaction_read_only='on';
DO $g$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'FALLO READ ONLY';
 EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'OK READ ONLY denegado'; END;
END $g$;
RESET SESSION AUTHORIZATION;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='3s';
CREATE ROLE vec_b67_prueba_ejecutor LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b67_prueba_ejecutor;
CREATE TEMP TABLE prueba_b67_consumos (capacidad bytea PRIMARY KEY,auditoria_ref text NOT NULL);
GRANT SELECT,INSERT ON pg_temp.prueba_b67_consumos TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,p_contexto_actor_canonico bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload_vec_ad_3 bytea,p_sobre_cose_sign1 bytea,p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $stub$
DECLARE c jsonb:=convert_from(p_capacidad_canonica,'UTF8')::jsonb; nueva boolean; a text:='auditoria-prueba:'||md5(p_capacidad_canonica::text);
BEGIN
 IF p_perfil_mutacion IS DISTINCT FROM 'consulta_persona_aceptacion_ct_bolsa' THEN RAISE EXCEPTION 'perfil de prueba incorrecto'; END IF;
 nueva:=NOT EXISTS (SELECT 1 FROM pg_temp.prueba_b67_consumos WHERE capacidad=p_capacidad_canonica);
 IF nueva THEN INSERT INTO pg_temp.prueba_b67_consumos VALUES(p_capacidad_canonica,a); END IF;
 RETURN QUERY SELECT 'decision-prueba:'||md5(p_capacidad_canonica::text),c->>'efecto_ref',c->>'huella_efecto_sha256',encode(sha256(p_capacidad_canonica),'hex'),a,clock_timestamp(),nueva;
END $stub$;
RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(p_candidato_ref text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog, pg_temp AS $stub$
BEGIN
 IF current_setting('prueba_b67.ctx_estado',true)='pendiente' THEN RETURN jsonb_build_object('estado','pendiente','persona',NULL,'vinculo',NULL); END IF;
 RETURN jsonb_build_object('estado','acreditado','persona',jsonb_build_object('ref','per_'||repeat('a',22),'version',1),
  'vinculo',jsonb_build_object('ref','vinculo:prueba:67','version',1,'procedencia_ref','procedencia:prueba:67','procedencia_version',1,'procedencia_sha256',repeat('a',64),'poblacion','interna','vigente_hasta','2099-01-01T00:00:00.000000Z'));
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
CREATE FUNCTION pg_temp.material(p_s jsonb) RETURNS text LANGUAGE sql AS $m$
 SELECT '{'||string_agg(to_json(k)::text||':'||to_json(p_s->>k)::text,',' ORDER BY n)||'}'
 FROM unnest(ARRAY['esquema','unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','aceptacion_registro_sha256','apertura_operacion_ref','apertura_registro_sha256','llamamiento_ref','propuesta_ref']) WITH ORDINALITY x(k,n)
$m$;
CREATE FUNCTION pg_temp.consultar(p_material text,p_nonce text,p_alteracion jsonb DEFAULT '{}'::jsonb) RETURNS jsonb LANGUAGE plpgsql AS $c$
DECLARE s jsonb:=p_material::jsonb; d jsonb; c jsonb; h text;
BEGIN
 h:=encode(sha256(convert_to('{"ambitos":{"unidad_ref":'||to_json(s->>'unidad_ref')::text||'},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 d:=jsonb_build_object('accion','bolsa.aceptacion_ct.persona.consultar','modulo_id','bolsa','tipo_recurso','persona_aceptacion_ct','finalidad','preparar_incorporacion_ct','recurso_ref',s->>'aceptacion_operacion_ref','contexto_recurso_huella_sha256',h,'campos_permitidos','["aceptacion","persona","vinculo"]'::jsonb,'obligaciones','[]'::jsonb)||p_alteracion;
 c:=jsonb_build_object('audiencia_consumo','vec_bolsa_llamamientos.aceptacion_ct.persona.v1','operacion','bolsa.aceptacion_ct.persona.consultar','efecto_ref',s->>'aceptacion_operacion_ref','huella_efecto_sha256',h,'huella_decision_sha256',encode(sha256(convert_to(d::text,'UTF8')),'hex'),'nonce',p_nonce,'relleno',repeat('x',512));
 RETURN vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(p_material,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),'\x01','\x01',1,1,'\x01','\x01','\x01',decode(repeat('a',88),'hex'));
END $c$;

-- Fixture nueva de aceptación/apertura que conserva literalmente la instantánea
-- de una constitución B7 y su vínculo B8. Solo se inserta en este ROLLBACK.
-- No declara legal ni corporativo el origen de ese acta de ejercicio.
DO $fixture$
DECLARE modelo jsonb; original vec_bolsa_llamamientos.integracion_desarrollo; v record;
 ap jsonb; te jsonb; ip text; apt text; tet text; ordref text;
BEGIN
 SELECT * INTO STRICT original FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE tipo='aceptacion_rrhh' ORDER BY operacion_ref LIMIT 1;
 SELECT convert_from(registro_canonico,'UTF8')::jsonb INTO STRICT modelo FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref=original.apertura_operacion_ref;
 PERFORM set_config('prueba_b67.original_material',pg_temp.material(jsonb_build_object('esquema','vec.bolsa.persona-aceptacion-ct.consulta.v1','unidad_ref',modelo->>'unidad_ref','categoria_ref',modelo->>'categoria_ref','necesidad_ref',modelo->>'necesidad_ref','aceptacion_operacion_ref',original.operacion_ref,'aceptacion_registro_sha256',original.registro_huella_sha256,'apertura_operacion_ref',original.apertura_operacion_ref,'apertura_registro_sha256',(SELECT registro_huella_sha256 FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref=original.apertura_operacion_ref),'llamamiento_ref',modelo#>>'{llamamiento,LlamamientoRef}','propuesta_ref',modelo#>>'{llamamiento,PropuestaRef}')),true);
 SELECT vc.*,ct.categoria_ref,ct.bolsa_ref,ct.version_bolsa,io.instantanea_canonica,e.orden
 INTO STRICT v FROM vec_bolsa_llamamientos.vinculo_candidato vc
 JOIN vec_bolsa_llamamientos.constitucion ct ON ct.acta_ref=vc.acta_ref
 JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=vc.instantanea_ref AND e.version_instantanea=vc.version_instantanea AND e.participacion_ref=vc.participacion_ref
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa io ON io.instantanea_ref=ct.instantanea_ref AND io.version=ct.version_instantanea AND io.huella_instantanea_sha256=ct.huella_instantanea_sha256
 ORDER BY vc.participacion_ref LIMIT 1;
 ip:=convert_from(v.instantanea_canonica,'UTF8');
 PERFORM set_config('prueba_b67.participacion',v.participacion_ref,true);
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
 PERFORM set_config('prueba_b67.material',pg_temp.material(jsonb_build_object('esquema','vec.bolsa.persona-aceptacion-ct.consulta.v1','unidad_ref','unidad:prueba:67','categoria_ref',v.categoria_ref,'necesidad_ref','necesidad:prueba:67','aceptacion_operacion_ref','operacion:prueba:67:aceptacion','aceptacion_registro_sha256',encode(sha256(convert_to(tet,'UTF8')),'hex'),'apertura_operacion_ref','operacion:prueba:67:apertura','apertura_registro_sha256',encode(sha256(convert_to(apt,'UTF8')),'hex'),'llamamiento_ref','llamamiento:prueba:67','propuesta_ref','propuesta:prueba:67')),true);
END $fixture$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO PUBLIC;
SET SESSION AUTHORIZATION vec_b67_prueba_ejecutor;
DO $lecturas$
DECLARE m text:=current_setting('prueba_b67.material'); r jsonb;
BEGIN
 r:=pg_temp.consultar(m,'positivo');
 IF r->>'estado'<>'acreditado' OR r#>>'{persona,ref}'<>'per_'||repeat('a',22)
    OR r#>>'{aceptacion,apertura_operacion_ref}'<>'operacion:prueba:67:apertura'
    OR r#>>'{vinculo,procedencia_ref}'<>'procedencia:prueba:67'
    OR r#>>'{evidencia,auditoria_ref}' IS NULL
 THEN RAISE EXCEPTION 'FALLO persona acreditada: %',r; END IF;
 IF r::text LIKE '%can_%' OR r::text LIKE '%participacion_ref%' OR r::text LIKE '%sujeto_ref%' OR r::text LIKE '%documento_identidad%' THEN RAISE EXCEPTION 'FALLO minimización'; END IF;
 r:=pg_temp.consultar(current_setting('prueba_b67.original_material'),'sin_vinculo');
 IF r->>'estado'<>'pendiente' OR r->'persona'<>'null'::jsonb OR r->'vinculo'<>'null'::jsonb THEN RAISE EXCEPTION 'FALLO aceptación histórica sin vínculo original B8: %',r; END IF;
 PERFORM set_config('prueba_b67.ctx_estado','pendiente',true);
 r:=pg_temp.consultar(m,'pendiente');
 IF r->>'estado'<>'pendiente' OR r->'aceptacion'='null'::jsonb OR r->'persona'<>'null'::jsonb OR r->'vinculo'<>'null'::jsonb THEN RAISE EXCEPTION 'FALLO pendiente'; END IF;
 r:=pg_temp.consultar(pg_temp.material(m::jsonb||jsonb_build_object('unidad_ref','unidad:prueba:ajena')),'unidad');
 IF r->>'estado'<>'no_encontrada' OR r->'aceptacion'<>'null'::jsonb OR r->'persona'<>'null'::jsonb THEN RAISE EXCEPTION 'FALLO unidad cruzada'; END IF;
 r:=pg_temp.consultar(pg_temp.material(m::jsonb||jsonb_build_object('categoria_ref','categoria:prueba:otra')),'categoria');
 IF r->>'estado'<>'no_encontrada' THEN RAISE EXCEPTION 'FALLO categoría cruzada'; END IF;
 r:=pg_temp.consultar(pg_temp.material(m::jsonb||jsonb_build_object('aceptacion_registro_sha256',repeat('c',64))),'hash');
 IF r->>'estado'<>'no_encontrada' THEN RAISE EXCEPTION 'FALLO huella cruzada'; END IF;
 r:=pg_temp.consultar(pg_temp.material(m::jsonb||jsonb_build_object('apertura_operacion_ref','operacion:prueba:otra')),'apertura');
 IF r->>'estado'<>'no_encontrada' THEN RAISE EXCEPTION 'FALLO apertura cruzada'; END IF;
 r:=pg_temp.consultar(pg_temp.material(m::jsonb||jsonb_build_object('aceptacion_operacion_ref','operacion:prueba:ausente')),'ausente');
 IF r->>'estado'<>'no_encontrada' OR r->'aceptacion'<>'null'::jsonb THEN RAISE EXCEPTION 'FALLO ausencia'; END IF;
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',m,'positivo'),'42501','replay no consume otra lectura');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'campos','{"campos_permitidos":["candidato_ref"]}'),'42501','campo candidato no autorizado');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'obligaciones','{"obligaciones":["otra"]}'),'42501','obligaciones cruzadas');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'recurso','{"recurso_ref":"operacion:prueba:otra"}'),'42501','recurso cruzado');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L,%L::jsonb)',m,'accion','{"accion":"bolsa.llamamiento.aceptacion_rrhh.registrar"}'),'42501','acción de escritura no habilita lectura');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',replace(m,'{"','{ "'),'canon'),'22023','material no canónico');
 PERFORM pg_temp.debe_fallar(format('SELECT pg_temp.consultar(%L,%L)',m::jsonb::text,'orden'),'22023','orden JSON divergente');
 PERFORM pg_temp.debe_fallar('SELECT vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(''can_'||repeat('a',22)||''')','42501','CTX18 inaccesible para ejecutor');
 PERFORM pg_temp.debe_fallar('SELECT * FROM vec_bolsa_llamamientos.vinculo_candidato','42501','tabla B8 inaccesible para ejecutor');
 PERFORM pg_temp.debe_fallar('SELECT * FROM vec_autorizacion_atestada_v3.consumir_consulta_persona_aceptacion_ct_bolsa_v3_atestada(NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL)','42501','consumidor AD3 inaccesible para ejecutor');
 RAISE NOTICE 'OK positivo B7 con dos huellas distintas, pendiente y ausencia autorizados';
END $lecturas$;
RESET SESSION AUTHORIZATION;
DO $auditoria$ BEGIN
 IF (SELECT count(*) FROM pg_temp.prueba_b67_consumos)<>8 THEN RAISE EXCEPTION 'FALLO consumo/auditoría'; END IF;
 RAISE NOTICE 'OK ocho lecturas auditadas; denegaciones sin efectos';
END $auditoria$;
ROLLBACK;
