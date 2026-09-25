\set ON_ERROR_STOP on
-- Fixture común de CT115/CT116 sobre la estructura real restaurada: rol de
-- ejecución de pruebas, dobles explícitos de las fachadas AD3-82/83 (prueban
-- la transacción CT, no la criptografía V3) e incorporación sintética del
-- expediente A. Base desechable: se confirma.
\if :{?exp_a}
\else
\set exp_a 'expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
\endif
CREATE ROLE vec_ct115_runtime LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
GRANT CONNECT ON DATABASE postgres TO vec_ct115_runtime;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cese_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.seguimiento.cesar' THEN RAISE EXCEPTION 'doble: cese denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cierre_expediente_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.expediente.cerrar' THEN RAISE EXCEPTION 'doble: cierre denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;

-- Incorporación sintética del expediente A (CT75), sin sus dependencias de
-- seguimiento: solo aporta el periodo confirmado que el cese necesita.
SET session_replication_role=replica;
INSERT INTO vec_contratacion_temporal.incorporacion_registro_v2(recibo_ref,seguimiento_ref,organizacion_ref,idempotencia_ref,solicitud_ref,expediente_ref,
  version_expediente,version_anterior,version_resultante,estado_anterior_sha256,estado_resultante_sha256,material_json,material_canonico,material_sha256,
  intencion_canonica,intencion_sha256,exportacion_ct,persona_version,perfil_version,recibo_json,auditoria_ref,outbox_ref,registrada_en,evidencia_orden_json)
SELECT 'ref:'||repeat('a',64),'seguimiento:ct115:a',v.agregado_json->>'organizacion_ref','idem:ct115:a','solicitud:ct115:a',:'exp_a',7,1,2,repeat('1',64),repeat('2',64),
  '{"Confirmacion":{"PeriodoIncorporacion":{"desde":"2027-01-01T00:00:00Z","hasta":"2027-03-31T00:00:00Z"}}}','\x01',encode(sha256('\x01'::bytea),'hex'),
  '\x02',encode(sha256('\x02'::bytea),'hex'),ARRAY['\x01','\x01','\x01','\x01','\x01','\x01','\x01','\x01']::bytea[],1,1,
  jsonb_build_object('MaterialOriginalSHA256',encode(sha256('\x01'::bytea),'hex'),'IntencionSHA256',encode(sha256('\x02'::bytea),'hex'),
    'SeguimientoRef','seguimiento:ct115:a','AuditoriaCTRef','auditoria:ct115:a','OutboxCTRef','outbox:ct115:a','Transicion',jsonb_build_object('recibo_ref','ref:'||repeat('a',64)),
    'EjercicioSintetico',true,'FirmaOficial',false,'EficaciaAdministrativa',false),'auditoria:ct115:a','outbox:ct115:a',timestamptz '2026-09-24 10:00:00+00','{}'
FROM vec_contratacion_temporal.expediente_version_integral v WHERE v.expediente_ref=:'exp_a' AND v.version=7;
RESET session_replication_role;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_modificacion_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.expediente.modificar_tras_nombramiento' THEN RAISE EXCEPTION 'doble: denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;
