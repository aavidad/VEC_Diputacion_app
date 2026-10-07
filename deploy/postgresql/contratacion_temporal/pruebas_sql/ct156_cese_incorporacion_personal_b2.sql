\set ON_ERROR_STOP on
-- Base sintética desechable tras CT155/156. Fixtures SQL directos prueban
-- publicación y verificación, no acreditan un acto Personal ni autorización V3.
-- Todo se revierte; no ejecutar sobre principal ni sobre datos reales.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
CREATE FUNCTION pg_temp.exigir_ct156(ok boolean,caso text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN IF ok IS DISTINCT FROM true THEN RAISE EXCEPTION 'CT156: %',caso;END IF;END $$;
REVOKE ALL ON FUNCTION pg_temp.exigir_ct156(boolean,text) FROM PUBLIC;

-- Una cuenta por capacidad; no conceder combinaciones CT/Bolsa.
CREATE ROLE vec_ct156_feed_prueba LOGIN INHERIT;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct156_feed_prueba;
CREATE ROLE vec_ct156_relevo_prueba LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_relevo_cese TO vec_ct156_relevo_prueba;
CREATE ROLE vec_ct156_ajeno_prueba LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_ct156_ajeno_prueba;
GRANT EXECUTE ON FUNCTION pg_temp.exigir_ct156(boolean,text)
 TO vec_ct156_feed_prueba,vec_ct156_relevo_prueba,vec_ct156_ajeno_prueba;
CREATE SCHEMA prueba_ct156 AUTHORIZATION vec_bolsa_llamamientos_propietario;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE FUNCTION prueba_ct156.verificar(origen text,huella text,posicion bigint) RETURNS jsonb
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp AS $$
 SELECT to_jsonb(v) FROM vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(origen,huella,posicion) v
$$;
REVOKE ALL ON FUNCTION prueba_ct156.verificar(text,text,bigint) FROM PUBLIC;
GRANT USAGE ON SCHEMA prueba_ct156 TO vec_bolsa_llamamientos_relevo_cese,vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION prueba_ct156.verificar(text,text,bigint) TO vec_bolsa_llamamientos_relevo_cese,vec_bolsa_llamamientos_ejecutor;
RESET ROLE;

CREATE TEMP TABLE ct156_esperado(protocolo text,origen text,incorporacion text,relacion text,inicio text,recibo text);
CREATE TEMP TABLE ct156_feed(evento_ref text,evento jsonb,huella_sha256 text,
 origen_ref text,origen_posicion bigint,origen_creada_en timestamptz);
CREATE TEMP TABLE ct156_ceses(evento_ref text,evento jsonb,huella_sha256 text,
 origen_ref text,origen_posicion bigint,origen_creada_en timestamptz);
GRANT INSERT,SELECT ON ct156_feed,ct156_ceses TO vec_ct156_feed_prueba;
GRANT SELECT ON ct156_feed,ct156_ceses TO vec_ct156_relevo_prueba,vec_ct156_ajeno_prueba;
GRANT SELECT ON ct156_esperado TO vec_ct156_relevo_prueba;
-- B2 se siembra sobre un expediente existente sin incorporación. El agregado
-- y su cadena histórica se conservan; el outbox se escribe con su helper real.
DO $fixture_b2$
DECLARE e record; m jsonb; canon text; ahora timestamptz:='2026-09-30T12:34:56.123456Z';
 ref text:='recibo:incorporacion-personal-b2:ct156-prueba';
BEGIN
 SELECT ea.organizacion_ref,v.expediente_ref,v.version,v.agregado_json INTO STRICT e
 FROM vec_contratacion_temporal.expediente_version_integral v
 JOIN vec_contratacion_temporal.expediente_alta ea USING(expediente_ref)
 WHERE NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2 i WHERE i.expediente_ref=v.expediente_ref)
 AND NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 p WHERE p.expediente_ref=v.expediente_ref)
 AND v.agregado_json#>>'{analisis,modalidad_clave}'~'^[a-z][a-z0-9_.-]{1,79}$'
 AND v.agregado_json#>>'{analisis,causa_clave}'~'^[a-z][a-z0-9_.-]{1,79}$'
 ORDER BY v.expediente_ref,v.version DESC LIMIT 1;
 m:=jsonb_build_object('material',jsonb_build_object('desde','2026-09-01','hasta','2026-12-31','bolsa',jsonb_build_object('llamamiento_ref','llamamiento:ct156:b2')));
 canon:=vec_contratacion_temporal.canon_plan_personal_ct155(m);
 SET LOCAL ROLE vec_contratacion_temporal_propietario;
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155(e.organizacion_ref,e.expediente_ref,e.version,'plan:ct156:b2','evento:ct156:b2:plan','ct.plan-incorporacion-personal.v1','recibo:ct156:b2:plan',ahora);
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155(e.organizacion_ref,e.expediente_ref,e.version,'plan:ct156:b2','evento:ct156:b2:incorporacion','ct.incorporacion-personal.v1',ref,ahora);
 RESET ROLE;
 INSERT INTO vec_contratacion_temporal.plan_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,version_expediente,clave_idempotencia,plan_ref,recibo_ref,intencion_ref,intencion_recibo_ref,solicitud_personal_ref,idempotencia_personal_uuid,material,material_json,material_sha256,actor_ref,perfil_ref,decision_ref,consumo_sha256,auditoria_ref,outbox_ref,registrada_en,contrato_json)
 VALUES(e.organizacion_ref,e.expediente_ref,e.version,gen_random_uuid(),'plan:ct156:b2','recibo:ct156:b2:plan','intencion:ct156:b2','recibo:intencion:ct156:b2','solicitud:ct156:b2',gen_random_uuid(),canon,m,encode(sha256(convert_to(canon,'UTF8')),'hex'),'actor:ct156','perfil:ct156','decision:ct156:b2:plan',repeat('1',64),'auditoria:ct156:b2:plan','evento:ct156:b2:plan',ahora,'{}');
 INSERT INTO vec_contratacion_temporal.origen_incorporacion_personal_b2_v1(organizacion_ref,expediente_ref,plan_ref,recibo_ref,material,material_json,material_sha256,relacion_ref,ocupacion_ref,actor_ref,perfil_ref,decision_ref,consumo_sha256,auditoria_ref,outbox_ref,registrada_en,recibo_json)
 VALUES(e.organizacion_ref,e.expediente_ref,'plan:ct156:b2',ref,canon,m,encode(sha256(convert_to(canon,'UTF8')),'hex'),'rel_ct156abcdefghijklmnopqrstuv','ocu_ct156abcdefghijklmnopqrstuv','actor:ct156','perfil:ct156','decision:ct156:b2:origen',repeat('2',64),'auditoria:ct156:b2:origen','evento:ct156:b2:incorporacion',ahora,'{}');
 INSERT INTO ct156_esperado VALUES('personal_b2_v1','evento:ct156:b2:cese',ref,'rel_ct156abcdefghijklmnopqrstuv','2026-09-01T00:00:00.000000Z','recibo:ct156:b2:cese');
END $fixture_b2$;
-- El CT75 real de esta base se usa tal como está; no se altera ni normaliza
-- su material original, siquiera para generar el cese de prueba.
INSERT INTO ct156_esperado
 SELECT 'ejercicio_v2','evento:ct156:ct75:cese',i.recibo_ref,
 i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}',
 vec_contratacion_temporal.instante_contrato_bolsa_v1((i.material_json#>>'{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),
 'recibo:ct156:ct75:cese' FROM vec_contratacion_temporal.incorporacion_registro_v2 i
 WHERE NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.cese_nombramiento_v1 c WHERE c.expediente_ref=i.expediente_ref)
 ORDER BY i.registrada_en LIMIT 1;
SELECT pg_temp.exigir_ct156((SELECT count(*) FROM ct156_esperado)=2,'fixtures CT75 real y B2 obligatorios');
DO $ceses$
DECLARE x record; i record; e record; payload bytea; anterior text;sec bigint; ahora timestamptz:='2026-09-30T12:34:57.654321Z';
 fecha date;recibojson jsonb; huella text;
BEGIN
 FOR x IN SELECT * FROM ct156_esperado LOOP
 SELECT * INTO STRICT i FROM vec_contratacion_temporal.origen_publicacion_bolsa_ct156(
 (SELECT organizacion_ref FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE recibo_ref=x.incorporacion UNION ALL SELECT organizacion_ref FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 WHERE recibo_ref=x.incorporacion),
 (SELECT expediente_ref FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE recibo_ref=x.incorporacion UNION ALL SELECT expediente_ref FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 WHERE recibo_ref=x.incorporacion),x.incorporacion,x.protocolo);
 SELECT * INTO STRICT e FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=i.expediente_ref ORDER BY version DESC LIMIT 1;
 fecha:=greatest('2026-09-30'::date,(i.inicio_instante AT TIME ZONE 'Europe/Madrid')::date);
 recibojson:=jsonb_build_object('registrada_en',vec_contratacion_temporal.instante_contrato_bolsa_v1(ahora));
 payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.cese.v1','organizacion_ref',i.organizacion_ref,'expediente_ref',i.expediente_ref,'version_resultante',e.version+1,'incorporacion_ref',x.incorporacion,'llamamiento_ref','llamamiento:ct156:'||x.protocolo,'causa_clave','fin_sustitucion','fecha_efecto',to_char(fecha,'YYYY-MM-DD'),'recibo_ref',x.recibo,'registrada_en',recibojson->'registrada_en')::text,'UTF8');
 SELECT max(secuencia)+1 INTO sec FROM vec_contratacion_temporal.outbox_expediente_integral;
 anterior:=repeat('0',64);
 -- La FK de outbox exige versión existente; para no inventar versión de
 -- negocio el fixture usa una versión anterior existente como esperada.
 IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=i.expediente_ref AND version=e.version-1) THEN RAISE EXCEPTION 'CT156: fixture requiere dos versiones del expediente'; END IF;
 payload:=convert_to((convert_from(payload,'UTF8')::jsonb||jsonb_build_object('version_resultante',e.version))::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.outbox_expediente_integral VALUES(x.origen,sec,'operacion:'||x.origen,i.expediente_ref,e.version,'ct.cese.v1',payload,encode(sha256(payload),'hex'),anterior,encode(sha256(anterior::bytea||payload),'hex'),ahora);
 huella:=encode(sha256(convert_to(x.origen,'UTF8')),'hex');
 INSERT INTO vec_contratacion_temporal.cese_nombramiento_v1(ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,causa_clave,fecha_efecto,justificante_tipo,justificante_ref,justificante_sha256,observaciones,incorporacion_ref,inicio_incorporacion,llamamiento_ref,estado,reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,recibo_json,decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,registrada_en,confirmada_en)
 VALUES('hmac-sha256:vec.contratacion-temporal.cese.ambito/v1:'||huella,'hmac-sha256:vec.contratacion-temporal.cese.peticion/v1:'||huella,i.organizacion_ref,i.expediente_ref,e.version-1,'actor:ct156','perfil:ct156','fin_sustitucion',fecha,'comunicacion_reincorporacion','documento:ct156:'||x.protocolo,repeat('1',64),'',x.incorporacion,(i.inicio_instante AT TIME ZONE 'Europe/Madrid')::date,'llamamiento:ct156:'||x.protocolo,'confirmada','reserva:'||x.origen,x.recibo,x.origen,e.agregado_json,e.agregado_json,recibojson,'decision:'||x.origen,huella,huella,'aud_v3_'||substr(huella,1,32),'politica:ct156',1,repeat('1',64),ahora,ahora);
 END LOOP;
END $ceses$;

SET SESSION AUTHORIZATION vec_ct156_feed_prueba;
INSERT INTO ct156_feed SELECT * FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL,NULL,100);
INSERT INTO ct156_ceses SELECT * FROM vec_contratacion_temporal.leer_ceses_bolsa_v1(NULL,NULL,100);
SELECT pg_temp.exigir_ct156((SELECT count(*) FROM ct156_ceses WHERE origen_ref LIKE 'evento:ct156:%')=2,'ambos ceses visibles');
SELECT pg_temp.exigir_ct156((SELECT count(*) FROM ct156_feed WHERE origen_ref='evento:ct156:b2:incorporacion' AND evento->>'tipo'='incorporacion' AND evento->>'inicio'='2026-09-01T00:00:00.000000Z')=1,'incorporación B2 inicial');
SELECT pg_temp.exigir_ct156(NOT EXISTS(SELECT 1 FROM ct156_ceses c LEFT JOIN ct156_feed f USING(origen_ref) WHERE c.origen_ref LIKE 'evento:ct156:%' AND (f.evento IS DISTINCT FROM c.evento OR f.huella_sha256 IS DISTINCT FROM c.huella_sha256 OR f.origen_posicion IS DISTINCT FROM c.origen_posicion)),'feeds mixto y exclusivo idénticos');
SELECT pg_temp.exigir_ct156(coalesce(current_setting('vec.ct115.publicacion_bolsa',true),'')='','marca RLS restaurada');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir_ct156(NOT EXISTS(SELECT 1 FROM ct156_ceses c JOIN ct156_esperado x ON x.origen=c.origen_ref WHERE c.evento->>'inicio' IS DISTINCT FROM x.inicio),'instante CT75 original y B2 UTC');
-- Reconstrucción independiente del cuerpo histórico CT75 y de su huella.
SELECT pg_temp.exigir_ct156(NOT EXISTS(
 SELECT 1 FROM ct156_ceses f JOIN ct156_esperado x ON x.origen=f.origen_ref
 JOIN vec_contratacion_temporal.cese_nombramiento_v1 c ON c.evento_ref=x.origen
 WHERE x.protocolo='ejercicio_v2' AND f.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(jsonb_build_object(
 'esquema','vec.contratacion-temporal.contrato-bolsa.v1','tipo','cese','evento_ref',f.evento_ref,'origen_ref',x.origen,'organizacion_ref',c.organizacion_ref,'expediente_ref',c.expediente_ref,'llamamiento_ref',c.llamamiento_ref,'inicio',x.inicio,'fin_previsto',vec_contratacion_temporal.instante_contrato_bolsa_v1(c.fecha_efecto::timestamp AT TIME ZONE 'UTC'),'modalidad_clave',c.expediente_siguiente_json#>>'{analisis,modalidad_clave}','categoria_ref',c.expediente_siguiente_json#>>'{analisis,categoria_ref}','causa_clave',c.causa_clave,'ocurrido_en',vec_contratacion_temporal.instante_contrato_bolsa_v1(c.registrada_en))::text,'UTF8')),'hex')),'JSON/huella CT75 byte a byte');
-- Incluye hora, zona y microsegundos diferentes de medianoche, aunque la
-- fila histórica de la copia tenga medianoche: el auxiliar conserva el tipo
-- timestamptz original y el formato UTC sin convertirlo a fecha civil.
SELECT pg_temp.exigir_ct156(vec_contratacion_temporal.instante_contrato_bolsa_v1('2026-09-01T23:41:12.987654+02'::timestamptz)='2026-09-01T21:41:12.987654Z'
 AND strpos(pg_get_functiondef('vec_contratacion_temporal.origen_publicacion_bolsa_ct156(text,text,text,text)'::regprocedure),$marca$(i.material_json#>>'{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz$marca$)>0,'instante CT75 con hora/zona/microsegundos');
SET SESSION AUTHORIZATION vec_ct156_relevo_prueba;
SELECT pg_temp.exigir_ct156(NOT EXISTS(SELECT 1 FROM ct156_ceses f JOIN ct156_esperado x ON x.origen=f.origen_ref WHERE prueba_ct156.verificar(f.origen_ref,f.huella_sha256,f.origen_posicion)->>'relacion_ref' IS DISTINCT FROM x.relacion OR prueba_ct156.verificar(f.origen_ref,f.huella_sha256,f.origen_posicion)->>'incorporacion_ref' IS DISTINCT FROM x.incorporacion),'origen/relación nativos de ambos protocolos');
SELECT pg_temp.exigir_ct156(NOT EXISTS(SELECT 1 FROM ct156_ceses f WHERE prueba_ct156.verificar(f.origen_ref,repeat('0',64),f.origen_posicion) IS NOT NULL OR prueba_ct156.verificar(f.origen_ref,f.huella_sha256,f.origen_posicion+1) IS NOT NULL),'huella y posición forjadas denegadas');
SELECT pg_temp.exigir_ct156(coalesce(current_setting('vec.ct129.origen_ref',true),'')='','marca verificador restaurada');
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ct156_ajeno_prueba;
SELECT pg_temp.exigir_ct156(NOT EXISTS(SELECT 1 FROM ct156_ceses f WHERE prueba_ct156.verificar(f.origen_ref,f.huella_sha256,f.origen_posicion) IS NOT NULL),'runtime Bolsa ajeno denegado');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir_ct156(NOT has_function_privilege('public','vec_contratacion_temporal.origen_publicacion_bolsa_ct156(text,text,text,text)','EXECUTE') AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.origen_publicacion_bolsa_ct156(text,text,text,text)','EXECUTE'),'auxiliar cerrado');
SELECT pg_temp.exigir_ct156((SELECT proconfig FROM pg_proc WHERE oid='prueba_ct156.verificar(text,text,bigint)'::regprocedure) IS NOT DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp'],'helper SECURITY DEFINER con entorno fijo');
SELECT pg_temp.exigir_ct156(NOT EXISTS (
 SELECT 1 FROM (VALUES
  ('leer_contratos_bolsa_v1(bigint,text,integer)',ARRAY['search_path=pg_catalog, pg_temp','timezone=utc']),
  ('leer_ceses_bolsa_v1(bigint,text,integer)',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','timezone=utc']),
  ('verificar_cese_publicado_bolsa_v1(text,text,bigint)',ARRAY['search_path=pg_catalog, pg_temp','row_security=on','timezone=utc']),
  ('registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)',
   ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=5s'])
 ) v(firma,config) LEFT JOIN pg_proc p ON p.oid=to_regprocedure('vec_contratacion_temporal.'||v.firma)
 WHERE p.oid IS NULL OR NOT p.prosecdef OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR ARRAY(SELECT lower(c) FROM unnest(p.proconfig) WITH ORDINALITY AS u(c,n) ORDER BY n)
       IS DISTINCT FROM v.config
),'funciones heredadas con entorno exacto y propietario conservado');
ROLLBACK;
SELECT 'CT156 OK';
