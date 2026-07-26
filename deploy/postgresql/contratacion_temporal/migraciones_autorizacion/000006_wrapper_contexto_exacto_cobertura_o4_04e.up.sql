-- O4-04E/2: wrapper VEC con contexto RecursoAutorizable exacto.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';

SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0
  )
);
DO $prevalidacion$
BEGIN
  IF pg_catalog.to_regprocedure(
    'vec_autorizacion.o404d_registrar_decision_v3_base_v1(bytea,bytea,numeric,numeric)'
  ) IS NULL OR pg_catalog.to_regprocedure(
    'vec_autorizacion.o404d_registrar_decision_v3_viva_v1(bytea,bytea,numeric,numeric)'
  ) IS NULL OR pg_catalog.to_regclass(
    'vec_autorizacion.enlace_decision_cobertura_ct_o404e'
  ) IS NULL OR pg_catalog.to_regprocedure(
    'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)'
  ) IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estado incompatible para wrapper exacto O4-04E';
  END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.o404e_texto_json_go_v1(p_valor text)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
  SELECT pg_catalog.replace(
    pg_catalog.replace(
      pg_catalog.replace(
        pg_catalog.replace(
          pg_catalog.replace(pg_catalog.to_json(p_valor)::text,
            '&',E'\\u0026'),'<',E'\\u003c'),'>',E'\\u003e'),
      pg_catalog.chr(8232),E'\\u2028'),
    pg_catalog.chr(8233),E'\\u2029')
$funcion$;

CREATE FUNCTION vec_autorizacion.o404e_mapa_json_go_v1(p_mapa jsonb)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
  SELECT CASE
    WHEN pg_catalog.jsonb_typeof(p_mapa)='object'
     AND (SELECT pg_catalog.count(*)
            FROM pg_catalog.jsonb_object_keys(p_mapa))<=512
     AND NOT EXISTS (
       SELECT 1 FROM pg_catalog.jsonb_each(p_mapa) e
        WHERE pg_catalog.jsonb_typeof(e.value)<>'string'
           OR e.key !~ '^[!-~]{1,128}$'
           OR e.key~'[[:space:]]'
           OR e.value#>>'{}'=''
           OR pg_catalog.octet_length(e.value#>>'{}')>512
           OR e.value#>>'{}'<>pg_catalog.btrim(e.value#>>'{}')
           OR e.value#>>'{}'~'[[:cntrl:]]'
     )
    THEN '{'||coalesce((
      SELECT pg_catalog.string_agg(
        vec_autorizacion.o404e_texto_json_go_v1(e.key)||':'||
        vec_autorizacion.o404e_texto_json_go_v1(e.value),
        ',' ORDER BY e.key COLLATE "C")
      FROM pg_catalog.jsonb_each_text(p_mapa)e
    ),'')||'}'
  END
$funcion$;

CREATE FUNCTION vec_autorizacion.o404e_claves_exactas_v1(
  p_valor jsonb,p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
  SELECT pg_catalog.jsonb_typeof(p_valor)='object'
    AND (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C")
           FROM pg_catalog.jsonb_object_keys(p_valor) k)=p_claves
$funcion$;

CREATE FUNCTION
vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(
  p_decision_canonica bytea,
  p_motivo_canonico bytea,
  p_persona_version numeric,
  p_perfil_version numeric,
  p_vinculo jsonb
)
RETURNS TABLE(
  rama text,concedida boolean,codigo text,decision_ref text,
  correlacion_ref text,organizacion_ref text,expediente_ref text,
  version_expediente numeric,reserva_ref text,
  contexto_recurso_huella_sha256 text,decision_huella_sha256 text,
  huella_orden_sha256 text,lote_huella_sha256 text,
  prueba_vinculo_sha256 text,registrada_en timestamptz,
  revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
SET row_security='on'
SET timezone='UTC'
SET lock_timeout='2s'
AS $funcion$
DECLARE
  d jsonb;
  v_registro record;
  v_resultado record;
  v_contexto bytea;
  v_contexto_huella text;
  v_prueba text;
  v_vinculada timestamptz(6);
  v_revalidada_en timestamptz(6);
  v_statement numeric;
  v_idle numeric;
BEGIN
  SELECT CASE WHEN unit='ms' AND setting~'^[0-9]{1,18}$'
              THEN setting::numeric END INTO v_statement
    FROM pg_catalog.pg_settings WHERE name='statement_timeout';
  SELECT CASE WHEN unit='ms' AND setting~'^[0-9]{1,18}$'
              THEN setting::numeric END INTO v_idle
    FROM pg_catalog.pg_settings
   WHERE name='idle_in_transaction_session_timeout';
  IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
     OR pg_catalog.current_setting('transaction_read_only')<>'off'
     OR v_statement NOT BETWEEN 1 AND 15000
     OR v_idle NOT BETWEEN 1 AND 20000
     OR p_decision_canonica IS NULL
     OR pg_catalog.octet_length(p_decision_canonica)
        NOT BETWEEN 128 AND 524288
     OR p_motivo_canonico IS NULL
     OR pg_catalog.octet_length(p_motivo_canonico)
        NOT BETWEEN 32 AND 65536
     OR p_persona_version IS NULL OR p_persona_version<>pg_catalog.trunc(
          p_persona_version)
     OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
     OR p_perfil_version IS NULL OR p_perfil_version<>pg_catalog.trunc(
          p_perfil_version)
     OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
     OR pg_catalog.jsonb_typeof(p_vinculo) IS DISTINCT FROM 'object'
     OR pg_catalog.jsonb_typeof(p_vinculo->'ambitos')
        IS DISTINCT FROM 'object'
     OR pg_catalog.jsonb_typeof(p_vinculo->'atributos')
        IS DISTINCT FROM 'object'
     OR NOT vec_autorizacion.o404e_claves_exactas_v1(
       p_vinculo,CASE p_vinculo->>'rama'
       WHEN 'concedida' THEN ARRAY[
         'accion','ambitos','atributos',
         'contexto_recurso_huella_sha256','correlacion_ref',
         'decision_ref','expediente_ref','finalidad',
         'huella_orden_sha256','lote_huella_sha256',
         'organizacion_ref','rama','recurso_modulo','recurso_ref',
         'recurso_tipo','reserva_ref','version_expediente']::text[]
       WHEN 'denegada' THEN ARRAY[
         'accion','ambitos','atributos',
         'contexto_recurso_huella_sha256','correlacion_ref',
         'decision_ref','expediente_ref','finalidad',
         'huella_orden_sha256','organizacion_ref','rama',
         'recurso_modulo','recurso_ref','recurso_tipo','reserva_ref',
         'version_expediente']::text[]
       ELSE ARRAY[]::text[] END)
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_each(p_vinculo) e
        WHERE e.key=ANY(ARRAY[
          'accion','contexto_recurso_huella_sha256','correlacion_ref',
          'decision_ref','expediente_ref','finalidad',
          'huella_orden_sha256','organizacion_ref','rama',
          'recurso_modulo','recurso_ref','recurso_tipo','reserva_ref']::text[])
          AND pg_catalog.jsonb_typeof(e.value)<>'string'
     )
     OR pg_catalog.jsonb_typeof(
          p_vinculo->'version_expediente')<>'number'
     OR (p_vinculo->>'rama'='concedida' AND pg_catalog.jsonb_typeof(
          p_vinculo->'lote_huella_sha256')<>'string')
     OR p_vinculo->>'recurso_modulo' IS DISTINCT FROM
        'contratacion_temporal'
     OR p_vinculo->>'recurso_tipo' IS DISTINCT FROM
        'decision_cobertura_gobernada'
     OR p_vinculo->>'recurso_ref' IS DISTINCT FROM
        p_vinculo->>'reserva_ref'
     OR p_vinculo->>'accion' NOT IN(
       'contratacion_temporal.cobertura.decidir',
       'contratacion_temporal.cobertura.rectificar')
     OR p_vinculo->>'organizacion_ref' !~
       '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR p_vinculo->>'expediente_ref' !~
       '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR p_vinculo->>'reserva_ref' !~
       '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR p_vinculo->>'decision_ref' !~
       '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR p_vinculo->>'correlacion_ref' !~
       '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR p_vinculo->>'version_expediente' !~
        '^([2-9]|[1-9][0-9]+)$'
     OR (p_vinculo->>'version_expediente')::numeric>
        9007199254740990::numeric
     OR p_vinculo->>'huella_orden_sha256' !~ '^[a-f0-9]{64}$'
     OR (p_vinculo->>'rama'='concedida'
       AND p_vinculo->>'lote_huella_sha256' !~ '^[a-f0-9]{64}$')
     OR EXISTS(
       SELECT 1
         FROM pg_catalog.jsonb_each_text(p_vinculo->'ambitos') e
        WHERE pg_catalog.strpos(e.value,'*')>0
     ) THEN
    RETURN;
  END IF;
  v_contexto:=pg_catalog.convert_to(
    '{"ambitos":'||
      vec_autorizacion.o404e_mapa_json_go_v1(p_vinculo->'ambitos')||
    ',"atributos":'||
      vec_autorizacion.o404e_mapa_json_go_v1(p_vinculo->'atributos')||'}',
    'UTF8');
  IF v_contexto IS NULL THEN RETURN; END IF;
  v_contexto_huella:=pg_catalog.encode(
    pg_catalog.sha256(v_contexto),'hex');
  BEGIN
    d:=pg_catalog.convert_from(p_decision_canonica,'UTF8')::jsonb;
  EXCEPTION WHEN data_exception OR invalid_text_representation
    OR character_not_in_repertoire OR untranslatable_character THEN
    RETURN;
  END;
  IF vec_autorizacion.o404d_decision_cobertura_v3_exacta_v1(
       d,p_decision_canonica) IS NOT TRUE
     OR d->>'modulo_id' IS DISTINCT FROM p_vinculo->>'recurso_modulo'
     OR d->>'tipo_recurso' IS DISTINCT FROM p_vinculo->>'recurso_tipo'
     OR d->>'accion' IS DISTINCT FROM p_vinculo->>'accion'
     OR d->>'recurso_ref' IS DISTINCT FROM p_vinculo->>'recurso_ref'
     OR d->>'decision_ref' IS DISTINCT FROM p_vinculo->>'decision_ref'
     OR d->>'correlacion_ref' IS DISTINCT FROM
        p_vinculo->>'correlacion_ref'
     OR d->>'finalidad' IS DISTINCT FROM p_vinculo->>'finalidad'
     OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM
        v_contexto_huella
     OR p_vinculo->>'contexto_recurso_huella_sha256' IS DISTINCT FROM
        v_contexto_huella
     OR (p_vinculo->>'rama'='concedida' AND
       (d->>'concedida'<>'true' OR d->>'codigo'<>'concedida'))
     OR (p_vinculo->>'rama'='denegada' AND d->>'concedida'<>'false')
     THEN RETURN;
  END IF;
  IF p_vinculo->>'rama'='concedida' THEN
    SELECT * INTO v_registro
      FROM vec_autorizacion.o404d_registrar_decision_v3_viva_v1(
        p_decision_canonica,p_motivo_canonico,
        p_persona_version,p_perfil_version);
    IF NOT FOUND OR v_registro.concedida IS DISTINCT FROM true
       OR v_registro.codigo IS DISTINCT FROM 'concedida' THEN RETURN; END IF;
    v_revalidada_en:=v_registro.revalidada_en;
    v_prueba:=pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.decode(v_registro.decision_huella_sha256,'hex')||
      pg_catalog.decode(v_contexto_huella,'hex')||
      pg_catalog.decode(p_vinculo->>'huella_orden_sha256','hex')||
      pg_catalog.decode(p_vinculo->>'lote_huella_sha256','hex')||
      pg_catalog.int4send(pg_catalog.octet_length(d->>'decision_ref'))||
      pg_catalog.convert_to(d->>'decision_ref','UTF8')||
      pg_catalog.int4send(pg_catalog.octet_length(d->>'correlacion_ref'))||
      pg_catalog.convert_to(d->>'correlacion_ref','UTF8')||
      pg_catalog.int8send((extract(epoch FROM
        v_registro.registrada_en)::numeric*1000000)::bigint)||
      pg_catalog.int8send((extract(epoch FROM
        v_revalidada_en)::numeric*1000000)::bigint)
    ),'hex');
  ELSE
    SELECT * INTO v_registro
      FROM vec_autorizacion.o404d_registrar_decision_v3_base_v1(
        p_decision_canonica,p_motivo_canonico,
        p_persona_version,p_perfil_version);
    IF NOT FOUND OR v_registro.concedida IS DISTINCT FROM false
       OR v_registro.codigo IS DISTINCT FROM d->>'codigo' THEN RETURN; END IF;
    v_revalidada_en:=NULL;
    v_prueba:=pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.decode(v_registro.decision_huella_sha256,'hex')||
      pg_catalog.decode(v_contexto_huella,'hex')||
      pg_catalog.decode(p_vinculo->>'huella_orden_sha256','hex')||
      pg_catalog.int4send(pg_catalog.octet_length(d->>'decision_ref'))||
      pg_catalog.convert_to(d->>'decision_ref','UTF8')||
      pg_catalog.int4send(pg_catalog.octet_length(d->>'correlacion_ref'))||
      pg_catalog.convert_to(d->>'correlacion_ref','UTF8')||
      pg_catalog.int8send((extract(epoch FROM
        v_registro.registrada_en)::numeric*1000000)::bigint)||
      pg_catalog.int8send(0)
    ),'hex');
  END IF;
  v_vinculada:=pg_catalog.date_trunc(
    'microseconds',pg_catalog.clock_timestamp());
  INSERT INTO vec_autorizacion.enlace_decision_cobertura_ct_o404e(
    decision_ref,rama,concedida,codigo,accion,decision_huella_sha256,
    decision_concedida_ref,decision_denegada_ref,correlacion_ref,
    organizacion_ref,expediente_ref,version_expediente,reserva_ref,
    contexto_recurso_huella_sha256,huella_orden_sha256,
    lote_huella_sha256,prueba_vinculo_sha256,registrada_en,
    revalidada_en,vinculada_en
  ) VALUES(
    d->>'decision_ref',p_vinculo->>'rama',v_registro.concedida,
    v_registro.codigo,p_vinculo->>'accion',
    v_registro.decision_huella_sha256,
    CASE WHEN v_registro.concedida THEN d->>'decision_ref' END,
    CASE WHEN NOT v_registro.concedida THEN d->>'decision_ref' END,
    d->>'correlacion_ref',p_vinculo->>'organizacion_ref',
    p_vinculo->>'expediente_ref',
    (p_vinculo->>'version_expediente')::numeric,
    p_vinculo->>'reserva_ref',v_contexto_huella,
    p_vinculo->>'huella_orden_sha256',
    p_vinculo->>'lote_huella_sha256',v_prueba,
    v_registro.registrada_en,v_revalidada_en,v_vinculada
  ) ON CONFLICT ON CONSTRAINT enlace_decision_cobertura_ct_o404e_pkey
    DO NOTHING;
  SELECT e.* INTO v_resultado
    FROM vec_autorizacion.enlace_decision_cobertura_ct_o404e e
   WHERE e.decision_ref=d->>'decision_ref'
     AND e.rama=p_vinculo->>'rama'
     AND e.concedida=v_registro.concedida
     AND e.codigo=v_registro.codigo
     AND e.accion=p_vinculo->>'accion'
     AND e.decision_huella_sha256=v_registro.decision_huella_sha256
     AND e.correlacion_ref=d->>'correlacion_ref'
     AND e.organizacion_ref=p_vinculo->>'organizacion_ref'
     AND e.expediente_ref=p_vinculo->>'expediente_ref'
     AND e.version_expediente=
        (p_vinculo->>'version_expediente')::numeric
     AND e.reserva_ref=p_vinculo->>'reserva_ref'
     AND e.contexto_recurso_huella_sha256=v_contexto_huella
     AND e.huella_orden_sha256=p_vinculo->>'huella_orden_sha256'
     AND e.lote_huella_sha256 IS NOT DISTINCT FROM
        p_vinculo->>'lote_huella_sha256'
     AND e.prueba_vinculo_sha256=v_prueba
     AND e.registrada_en=v_registro.registrada_en
     AND e.revalidada_en IS NOT DISTINCT FROM v_revalidada_en;
  IF NOT FOUND THEN
    RAISE EXCEPTION USING ERRCODE='23505',
      MESSAGE='colisión de enlace exacto VEC-CT O4-04E';
  END IF;
  RETURN QUERY SELECT
    v_resultado.rama,v_resultado.concedida,v_resultado.codigo,
    v_resultado.decision_ref,v_resultado.correlacion_ref,
    v_resultado.organizacion_ref,v_resultado.expediente_ref,
    v_resultado.version_expediente,v_resultado.reserva_ref,
    v_resultado.contexto_recurso_huella_sha256,
    v_resultado.decision_huella_sha256,
    v_resultado.huella_orden_sha256,v_resultado.lote_huella_sha256,
    v_resultado.prueba_vinculo_sha256,v_resultado.registrada_en,
    v_resultado.revalidada_en;
EXCEPTION WHEN data_exception OR invalid_text_representation
  OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
  RETURN;
END
$funcion$;

REVOKE ALL ON FUNCTION
  vec_autorizacion.o404e_texto_json_go_v1(text),
  vec_autorizacion.o404e_mapa_json_go_v1(jsonb),
  vec_autorizacion.o404e_claves_exactas_v1(jsonb,text[]),
  vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(
    bytea,bytea,numeric,numeric,jsonb)
FROM PUBLIC,vec_autorizacion_registro,vec_autorizacion_fuente,
  vec_autorizacion_motivos_proyector,
  vec_autorizacion_motivos_evaluador,
  vec_contratacion_temporal_ejecutor,
  vec_contratacion_temporal_migrador,
  vec_contratacion_temporal_gobernador;
GRANT EXECUTE ON FUNCTION
  vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(
    bytea,bytea,numeric,numeric,jsonb)
TO vec_contratacion_temporal_propietario;
COMMENT ON FUNCTION
  vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(
    bytea,bytea,numeric,numeric,jsonb)
IS 'Wrapper O4-04E: recalcula el contexto RecursoAutorizable y persiste su enlace exacto; el lote concedido se liga separadamente.';
COMMIT;
