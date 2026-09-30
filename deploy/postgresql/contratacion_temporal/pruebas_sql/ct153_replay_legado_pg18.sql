\set ON_ERROR_STOP on
-- Clon PG18 desechable después de CT153. Siembra una fila CT130 con los
-- sellos y el material anteriores, acredita una lectura CT134 sintética y
-- comprueba que preparar recupera el mismo recibo sin crear otra operación.
-- Todo queda dentro de ROLLBACK; no prueba firma ni consumo criptográfico V3.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
CREATE ROLE vec_ct153_fixture LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;

CREATE TEMP TABLE ct153_fixture ON COMMIT DROP AS
SELECT jsonb_build_object(
 'organizacion_ref',v.agregado_json->>'organizacion_ref',
 'expediente_ref',v.expediente_ref,
 'relacion_ref','relacion:ct153:legado',
 'version_esperada',v.version,
 'actor_ref','actor:ct153:rrhh',
 'perfil_ref','perfil:ct153:rrhh',
 'fecha_efectiva','2026-09-29',
 'documento_ref','documento:ct153:retorno',
 'documento_sha256',repeat('a',64)) AS material,
 'hmac-sha256:vec.contratacion-temporal.reincorporacion-titular.ambito/v1:'||repeat('b',64) AS ambito,
 'hmac-sha256:vec.contratacion-temporal.reincorporacion-titular.peticion/v1:'||repeat('c',64) AS huella,
 'lectura:'||repeat('d',64) AS lectura_ref,
 'aud_v3_'||repeat('e',32) AS auditoria_ref,
 jsonb_build_object('recibo_ref','recibo:ct153:legado','expediente_ref',v.expediente_ref) AS recibo
FROM vec_contratacion_temporal.expediente_version_integral v
WHERE vec_contratacion_temporal.referencia_valida_ct115(v.agregado_json->>'organizacion_ref')
  AND v.version BETWEEN 1 AND 9007199254740990
ORDER BY v.expediente_ref COLLATE "C",v.version DESC LIMIT 1;
DO $pre$ BEGIN
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
       WHERE oid=to_regprocedure('vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
       IS DISTINCT FROM '0e35099bbb4eba74671609d9553948bef0812e9d4d5eaf1b46be550aec3268c6'
    OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
       WHERE oid=to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
       IS DISTINCT FROM '570e9652dc6ff181230b08a708f7535053ccd59316e447bc8610b3d71b0aef88' THEN
  RAISE EXCEPTION 'CT153 TEST: funciones no instaladas' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM ct153_fixture)<>1 THEN
  RAISE EXCEPTION 'CT153 TEST: falta expediente sintético' USING ERRCODE='55000'; END IF;
 IF has_function_privilege('public','vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()','EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()','EXECUTE') THEN
  RAISE EXCEPTION 'CT153 TEST: ACL del marcador incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
GRANT SELECT ON ct153_fixture TO vec_ct153_fixture;

-- Solo este clon de prueba omite las FK al sembrar historia anterior. Las
-- restricciones CHECK y la comprobación de recibo real siguen activas.
SET LOCAL session_replication_role=replica;
INSERT INTO vec_contratacion_temporal.reincorporacion_titular_v1(
 ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,relacion_ref,version_esperada,actor_ref,perfil_ref,
 fecha_efectiva,documento_ref,documento_sha256,cese_evento_ref,cese_recibo_ref,incorporacion_recibo_ref,
 reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,recibo_json,
 decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,
 politica_ref,politica_version,politica_huella_sha256,registrada_en,confirmada_en)
SELECT ambito,huella,material->>'organizacion_ref',material->>'expediente_ref',material->>'relacion_ref',
 (material->>'version_esperada')::numeric,material->>'actor_ref',material->>'perfil_ref',
 (material->>'fecha_efectiva')::date,material->>'documento_ref',material->>'documento_sha256',
 'evento:ct153:cese','recibo:ct153:cese','recibo:ct153:incorporacion',
 'reserva:ct153:legado','recibo:ct153:legado','evento:ct153:legado','{}','{}',recibo,
 'decision:ct153:legado',repeat('1',64),repeat('2',64),'aud_v3_'||repeat('3',32),
 'politica:ct153:legado',1,repeat('4',64),clock_timestamp(),clock_timestamp()
FROM ct153_fixture;
RESET session_replication_role;

INSERT INTO vec_contratacion_temporal.lectura_reincorporacion_titular_v1(
 lectura_ref,organizacion_ref,expediente_ref,actor_ref,perfil_ref,version_esperada,
 peticion_sha256,contexto_sha256,resultado,cese_evento_ref,cese_recibo_ref,decision_ref,
 consumo_huella_sha256,auditoria_ref,registrada_en)
SELECT lectura_ref,material->>'organizacion_ref',material->>'expediente_ref',material->>'actor_ref',
 material->>'perfil_ref',(material->>'version_esperada')::numeric,
 encode(sha256(convert_to(material::text,'UTF8')),'hex'),repeat('5',64),'coincide',
 'evento:ct153:cese','recibo:ct153:cese','decision:ct153:lectura',repeat('6',64),auditoria_ref,
 clock_timestamp() FROM ct153_fixture;

SET SESSION AUTHORIZATION vec_ct153_fixture;
DO $marcador$ BEGIN
 IF vec_contratacion_temporal.perfil_reincorporacion_ct153_v1() IS DISTINCT FROM 'organizacion_ref' THEN
  RAISE EXCEPTION 'CT153 TEST: marcador incompatible' USING ERRCODE='55000'; END IF;
END $marcador$;
CREATE TEMP TABLE ct153_resultado ON COMMIT DROP AS
SELECT vec_contratacion_temporal.preparar_reincorporacion_titular_acreditada_v1(
 jsonb_build_object('esquema','vec.contratacion-temporal.preparar-reincorporacion-titular.v1',
   'operacion','registrar_reincorporacion_titular','material',material,
   'sellos_hmac',jsonb_build_object('activo',jsonb_build_object(
       'ambito_hmac',ambito,'generacion',1,'huella_peticion_hmac',huella),'retenidos','[]'::jsonb),
   'referencias_candidatas',jsonb_build_object('reserva_ref','reserva:ct153:nueva',
       'recibo_ref','recibo:ct153:nuevo','evento_ref','evento:ct153:nuevo')),
 lectura_ref,auditoria_ref) AS respuesta,recibo
FROM ct153_fixture;
RESET SESSION AUTHORIZATION;

DO $verificar$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM ct153_resultado;
 IF r.respuesta->>'resultado' IS DISTINCT FROM 'confirmada'
    OR r.respuesta->'recibo' IS DISTINCT FROM r.recibo
    OR (SELECT count(*) FROM vec_contratacion_temporal.reincorporacion_titular_v1)<>1
    OR (SELECT count(*) FROM vec_contratacion_temporal.uso_lectura_reincorporacion_titular_v1)<>1 THEN
  RAISE EXCEPTION 'CT153 TEST: replay cambió el recibo o creó otra operación' USING ERRCODE='55000'; END IF;
END $verificar$;
ROLLBACK;
