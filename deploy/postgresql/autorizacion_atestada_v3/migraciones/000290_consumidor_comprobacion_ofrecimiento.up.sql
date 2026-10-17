\set ON_ERROR_STOP on
-- AD290 / F1-09: fachada de escritura, exclusiva del propietario de Bolsa.
-- El perfil se incorpora a la reconstrucción central coordinada por dirección.
-- Sin ese perfil el núcleo deniega; ninguna concesión de consulta sirve aquí.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000290',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD290: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;d jsonb;
BEGIN
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 524288
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 THEN RAISE EXCEPTION 'AD290: material inválido' USING ERRCODE='42501'; END IF;
 c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR c->>'operacion' IS DISTINCT FROM 'bolsa.comprobacion_ofrecimiento.registrar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.comprobacion_ofrecimiento.v1'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
 OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS NULL
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR jsonb_typeof(d->'ambitos') IS DISTINCT FROM 'object'
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(d->'ambitos') k)
    IS DISTINCT FROM ARRAY['ambito_ref','unidad_ref']
 THEN RAISE EXCEPTION 'AD290: comprobación denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.* FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'comprobacion_ofrecimiento_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz) x;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_comprobacion_ofrecimiento_v1(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
COMMIT;
