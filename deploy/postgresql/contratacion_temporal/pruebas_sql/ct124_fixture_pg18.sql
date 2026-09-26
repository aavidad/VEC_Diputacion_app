\set ON_ERROR_STOP on
-- Fixture de CT124 sobre la estructura real restaurada (después de
-- ct115_ct116_fixture_pg18.sql, que crea vec_ct115_runtime, los dobles de
-- AD3-82/83 y la incorporación sintética del expediente A): dobles explícitos
-- de las fachadas AD3-88 (prueban la transacción CT, no la criptografía V3) y
-- la entrega sintética a RRHH de la petición ratificada del centro 520 como
-- expediente B. Base desechable: se confirma.
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.ginpix.confirmar' THEN RAISE EXCEPTION 'doble: GINPIX denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion' NOT IN ('contratacion_temporal.incorporacion.confirmar_centro','contratacion_temporal.incorporacion.consultar_centro') THEN
  RAISE EXCEPTION 'doble: centro denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;

-- Entrega sintética (CT68) de la petición ratificada del centro 520 como el
-- expediente B, sin sus dependencias de alta: solo el vínculo que lee CT124.
SET session_replication_role=replica;
INSERT INTO vec_contratacion_temporal.entrega_peticion_centro_confirmacion(peticion_ref,entrega_ref,expediente_ref,numero_visible,version_alta,
  recibo_ref,auditoria_alta_ref,evento_alta_ref,alta_confirmada_en,ambito_alta_hmac,ambito_alta_raiz_hmac,confirmacion_alta_ref,recibo_alta,
  acceso_confirmacion_ref,registrada_en)
SELECT r.peticion_ref,'entrega:ct124:b',:'exp_b','2026/B-124',1,'recibo:ct124:alta:b','auditoria:ct124:b','evento:ct124:b',
  timestamptz '2026-09-20 10:00:00+00','hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||repeat('b',64),
  'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||repeat('c',64),'confirmacion:ct124:b','{}','acceso:ct124:b',timestamptz '2026-09-20 10:00:00+00'
  FROM vec_contratacion_temporal.peticion_centro_revision r WHERE r.version=2 AND r.estado='ratificada' AND r.centro_ref='centro-520';
RESET session_replication_role;
SELECT 'fixture CT124 OK';
