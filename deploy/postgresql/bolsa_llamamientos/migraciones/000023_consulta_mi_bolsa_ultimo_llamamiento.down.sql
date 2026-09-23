\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000023:down', 0));

-- Revierte exclusivamente la proyeccion B11 de llamamiento al contrato exacto de 000022.
-- No ejecutar sobre bases con historia; usar solo en ensayo aislado.
-- No expone motivo libre, actor, recibo ni referencia interna de participacion.
DO $precondicion$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL THEN
   RAISE EXCEPTION 'estado incompatible para consulta Mi bolsa con situacion' USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_v1(
 p_candidato_ref text,p_consultada_en timestamptz,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE c jsonb; d jsonb; x jsonb; v_candidatos integer;
BEGIN
 IF p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$' OR p_consultada_en IS NULL
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'consulta Mi bolsa inválida' USING ERRCODE='22023';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN
   RAISE EXCEPTION 'consulta Mi bolsa inválida' USING ERRCODE='22023';
 END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR d->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
   RAISE EXCEPTION 'consulta Mi bolsa denegada' USING ERRCODE='42501';
 END IF;
 SELECT count(*) INTO v_candidatos FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_candidatos<>1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
   RAISE EXCEPTION 'consulta Mi bolsa denegada' USING ERRCODE='42501';
 END IF;
 PERFORM 1 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
    'bolsa',participacion.bolsa_ref,'categoria',participacion.categoria_ref,'version',participacion.version_bolsa,'orden_inicial',participacion.orden,'total_instantanea',participacion.total_participaciones,'estado_bolsa',participacion.estado,'vigente_desde',to_char(participacion.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_hasta',CASE WHEN participacion.vigente_hasta IS NULL THEN NULL ELSE to_char(participacion.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
    'situacion_actual',CASE WHEN situacion.participacion_ref IS NULL THEN NULL ELSE jsonb_build_object(
      'estado',situacion.situacion,
      'desde',to_char(situacion.desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'hasta',CASE WHEN situacion.hasta IS NULL THEN NULL ELSE to_char(situacion.hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
      'fecha_disponible',CASE WHEN situacion.fecha_disponible IS NULL THEN NULL ELSE to_char(situacion.fecha_disponible AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END
    ) END
  ) ORDER BY participacion.confirmada_en DESC,participacion.categoria_ref),'[]'::jsonb) INTO x
 FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref) participacion
 LEFT JOIN LATERAL (
   SELECT s.participacion_ref,s.situacion,s.desde,s.hasta,s.fecha_disponible
     FROM vec_bolsa_llamamientos.situacion_participacion s
    WHERE s.participacion_ref=participacion.participacion_ref AND s.desde<=p_consultada_en
    ORDER BY s.desde DESC LIMIT 1
 ) situacion ON true;
 RETURN jsonb_build_object('consultada_en',to_char(p_consultada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'participaciones',x);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
COMMIT;
