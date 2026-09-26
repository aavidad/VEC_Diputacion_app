\set ON_ERROR_STOP on
-- Fixture de la cadena «aceptación → no incorporación → baja en Bolsa →
-- siguiente llamamiento» (CT124 + Bolsa 000042) sobre la estructura real
-- restaurada, después de ct115_ct116_fixture_pg18.sql cargado con la
-- incorporación sintética en el expediente B (el A, con su aceptación
-- confirmada y su propuesta de nombramiento, queda sin incorporación).
-- Dobles explícitos de las fachadas AD3 (prueban las transacciones CT y Bolsa,
-- no la criptografía V3), rol de ejecución de Bolsa y constitución sintética
-- de la bolsa del llamamiento del expediente A con la persona aceptada.
-- Base desechable: se confirma.
BEGIN;
CREATE ROLE vec_b42_runtime LOGIN INHERIT IN ROLE vec_bolsa_llamamientos_ejecutor;
GRANT CONNECT ON DATABASE postgres TO vec_b42_runtime;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_no_incorporacion_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.incorporacion.no_incorporacion' THEN RAISE EXCEPTION 'doble: no incorporación denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_continuacion_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb; v text:=gen_random_uuid()::text;
BEGIN
 RETURN QUERY SELECT 'decision:'||v, d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(convert_to(v,'UTF8')),'hex'),
   'aud_v3_'||md5(v), clock_timestamp(), true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; v text:=gen_random_uuid()::text;
BEGIN
 RETURN QUERY SELECT 'decision:'||v, c->>'efecto_ref', c->>'huella_efecto_sha256', encode(sha256(convert_to(v,'UTF8')),'hex'),
   'auditoria:'||v, clock_timestamp(), true;
END $f$;

-- Constitución sintética de la bolsa del llamamiento del expediente A (sin
-- sus dependencias de importación): la persona aceptada queda disponible.
SELECT l.bolsa_ref AS bolsa_a, convert_from(i.registro_canonico,'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}' AS participacion_a
  FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
  JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.operacion_ref=l.operacion_ref
 WHERE l.llamamiento_ref=(SELECT llamamiento_ref FROM vec_contratacion_temporal.propuesta_formalizacion
                           WHERE expediente_ref='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001') \gset
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.constitucion(acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,
  huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
VALUES ('acta:prueba:b42',:'bolsa_a',1,repeat('1',64),'instantanea:prueba:b42',1,repeat('2',64),'categoria:desarrollo:c2','actor:rrhh:sintetico',
  timestamptz '2026-09-01 08:00:00+00',timestamptz '2026-09-01 08:00:00+00');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
VALUES ('instantanea:prueba:b42',1,2,:'participacion_a',2);
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,
  registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version)
VALUES (:'participacion_a','disponible',timestamptz '2026-09-01 08:00:00+00',NULL,NULL,'Constitución de bolsa','sistema:constitucion',
  timestamptz '2026-09-01 08:00:00+00','constitucion:'||:'participacion_a','recibo:situacion:constitucion:'||:'participacion_a',NULL);
SET LOCAL session_replication_role=origin;
SELECT CASE WHEN (SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=:'participacion_a')='disponible'
  THEN 'fixture no incorporación OK' ELSE 'fixture no incorporación FALLO' END;
COMMIT;
