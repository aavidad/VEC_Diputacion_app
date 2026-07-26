-- O4-04E/6: recurso VEC denegado y prueba durable verificable.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0
  )
);
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=8
 FOR UPDATE;

DO $prevalidacion$
BEGIN
  IF NOT EXISTS (
    SELECT 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema=8
  ) OR pg_catalog.to_regclass(
    'vec_contratacion_temporal.prueba_denegacion_decision_cobertura'
  ) IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estado incompatible para contexto denegado O4-04E';
  END IF;
END
$prevalidacion$;

-- encoding/json escapa HTML y también U+2028/U+2029. to_json cubre el resto
-- de escapes de una cadena UTF-8; estos reemplazos cierran la diferencia Go.
CREATE FUNCTION vec_contratacion_temporal.o404e_texto_json_go_v1(
  p_valor text
)
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

CREATE FUNCTION vec_contratacion_temporal.o404e_mapa_json_go_v1(
  p_mapa jsonb
)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
  SELECT CASE
    WHEN pg_catalog.jsonb_typeof(p_mapa)='object'
     AND NOT EXISTS (
       SELECT 1 FROM pg_catalog.jsonb_each(p_mapa) e
        WHERE pg_catalog.jsonb_typeof(e.value)<>'string'
     )
    THEN '{'||coalesce((
      SELECT pg_catalog.string_agg(
        vec_contratacion_temporal.o404e_texto_json_go_v1(e.key)||':'||
        vec_contratacion_temporal.o404e_texto_json_go_v1(e.value),
        ',' ORDER BY e.key COLLATE "C"
      )
      FROM pg_catalog.jsonb_each_text(p_mapa) e
    ),'')||'}'
  END
$funcion$;

-- Replica RecursoAutorizable.HuellaContextoAutorizacionSHA256 y las
-- invariantes de recursoPruebaDenegacionOperacionDecisionCoberturaCoherente.
CREATE FUNCTION
vec_contratacion_temporal.o404e_contexto_recurso_denegacion_v1(
  p_carga jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
DECLARE
  c jsonb:=p_carga->'cabecera';
  d jsonb:=p_carga->'denegacion';
  a jsonb:=d->'atributos';
  m jsonb:=d->'ambitos';
  v_clave text;
  v_json text;
BEGIN
  IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object'
     OR pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object'
     OR pg_catalog.jsonb_typeof(a) IS DISTINCT FROM 'object'
     OR pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object'
     OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
       m,ARRAY['organizacion_ref','unidad_ejecutora_ref']::text[])
     OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
       a,CASE a->>'tipo_operacion'
         WHEN 'inicial' THEN ARRAY[
           'accion','analisis_huella_sha256','analisis_ref',
           'catalogo_huella_sha256','catalogo_ref','catalogo_version',
           'expediente_ref','politica_actuacion_huella_sha256',
           'politica_actuacion_ref','politica_actuacion_version',
           'politica_huella_sha256','politica_ref','politica_version',
           'preparacion_evidencias_huella_sha256',
           'preparacion_evidencias_ref','propuesta_huella_sha256',
           'propuesta_ref','propuesta_semantica_huella_sha256',
           'propuesta_semantica_ref','reserva_ref','revision_cercado',
           'tipo_operacion','version_expediente_esperada','via_elegida'
         ]::text[]
         WHEN 'rectificacion' THEN ARRAY[
           'accion','analisis_huella_sha256','analisis_ref',
           'catalogo_huella_sha256','catalogo_ref','catalogo_version',
           'expediente_ref','politica_actuacion_huella_sha256',
           'politica_actuacion_ref','politica_actuacion_version',
           'politica_huella_sha256','politica_ref','politica_version',
           'predecesora_huella_sha256','predecesora_ref',
           'preparacion_evidencias_huella_sha256',
           'preparacion_evidencias_ref','propuesta_huella_sha256',
           'propuesta_ref','propuesta_semantica_huella_sha256',
           'propuesta_semantica_ref','reserva_ref','revision_cercado',
           'tipo_operacion','version_expediente_esperada','via_elegida'
         ]::text[]
         ELSE ARRAY[]::text[] END)
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_each(m||a) e
        WHERE pg_catalog.jsonb_typeof(e.value)<>'string'
     )
     OR m->>'organizacion_ref'<>c->>'organizacion_ref'
     OR m->>'unidad_ejecutora_ref' !~
        '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
     OR a->>'expediente_ref'<>c->>'expediente_ref'
     OR a->>'version_expediente_esperada'<>c->>'version_expediente'
     OR a->>'reserva_ref'<>c->>'reserva_ref'
     OR a->>'revision_cercado'<>c->>'revision_cercado'
     OR a->>'accion'<>d->>'accion_vec'
     OR a->>'accion'<>(CASE a->>'tipo_operacion'
       WHEN 'inicial' THEN 'contratacion_temporal.cobertura.decidir'
       WHEN 'rectificacion' THEN
         'contratacion_temporal.cobertura.rectificar' END)
     OR a->>'via_elegida' !~ '^[a-z][a-z0-9._-]{1,79}$'
     OR a->>'propuesta_semantica_ref'<>
        'propuesta-cobertura-semantica:sha256:'||
        coalesce(a->>'propuesta_semantica_huella_sha256','') THEN
    RETURN NULL;
  END IF;
  FOREACH v_clave IN ARRAY ARRAY[
    'propuesta_ref','propuesta_semantica_ref',
    'preparacion_evidencias_ref','analisis_ref','catalogo_ref',
    'politica_ref','politica_actuacion_ref'
  ]::text[] LOOP
    IF a->>v_clave !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
      RETURN NULL;
    END IF;
  END LOOP;
  FOREACH v_clave IN ARRAY ARRAY[
    'propuesta_huella_sha256','propuesta_semantica_huella_sha256',
    'preparacion_evidencias_huella_sha256','analisis_huella_sha256',
    'catalogo_huella_sha256','politica_huella_sha256',
    'politica_actuacion_huella_sha256'
  ]::text[] LOOP
    IF a->>v_clave !~ '^[a-f0-9]{64}$'
       OR a->>v_clave=pg_catalog.repeat('0',64) THEN
      RETURN NULL;
    END IF;
  END LOOP;
  FOREACH v_clave IN ARRAY ARRAY[
    'catalogo_version','politica_version','politica_actuacion_version'
  ]::text[] LOOP
    IF a->>v_clave !~ '^[1-9][0-9]*$'
       OR (a->>v_clave)::numeric NOT BETWEEN
          1 AND 9007199254740991::numeric THEN
      RETURN NULL;
    END IF;
  END LOOP;
  IF a->>'tipo_operacion'='rectificacion'
     AND (a->>'predecesora_ref' !~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR a->>'predecesora_huella_sha256' !~ '^[a-f0-9]{64}$'
       OR a->>'predecesora_huella_sha256'=pg_catalog.repeat('0',64)) THEN
    RETURN NULL;
  END IF;
  v_json:='{"ambitos":'||
    vec_contratacion_temporal.o404e_mapa_json_go_v1(m)||
    ',"atributos":'||
    vec_contratacion_temporal.o404e_mapa_json_go_v1(a)||'}';
  RETURN pg_catalog.convert_to(v_json,'UTF8');
EXCEPTION
  WHEN data_exception OR invalid_text_representation
    OR numeric_value_out_of_range THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_motivo_denegacion_canon_v1(
  p_motivo jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
  SELECT CASE WHEN
    vec_contratacion_temporal.o404e_claves_exactas_v1(
      p_motivo,ARRAY['catalogo_huella_sha256','catalogo_id',
        'catalogo_version','entrada_clave']::text[])
  THEN pg_catalog.convert_to(
    '{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada"'||
    ',"referencia":{"catalogo_id":'||
      vec_contratacion_temporal.o404e_texto_json_go_v1(
        p_motivo->>'catalogo_id')||
    ',"catalogo_version":'||(p_motivo->'catalogo_version')::text||
    ',"catalogo_huella_sha256":'||
      vec_contratacion_temporal.o404e_texto_json_go_v1(
        p_motivo->>'catalogo_huella_sha256')||
    ',"entrada_clave":'||
      vec_contratacion_temporal.o404e_texto_json_go_v1(
        p_motivo->>'entrada_clave')||'}}','UTF8')
  END
$funcion$;

ALTER TABLE
  vec_contratacion_temporal.terminal_operacion_decision_cobertura
ADD CONSTRAINT terminal_ambito_decision_o404e_unico
UNIQUE(ambito_raiz_hmac,decision_vec_ref);

-- Las confirmaciones anteriores pueden existir al actualizar desde O4-04C,
-- por eso la columna es anulable. Todo cierre O4-04E exige y verifica sello.
ALTER TABLE
  vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
ADD COLUMN carga_huella_sha256 text,
ADD CONSTRAINT confirmacion_carga_huella_o404e_valida CHECK (
  carga_huella_sha256 IS NULL
  OR (
    carga_huella_sha256 ~ '^[a-f0-9]{64}$'
    AND carga_huella_sha256<>pg_catalog.repeat('0',64)
  )
);

CREATE TABLE
vec_contratacion_temporal.prueba_denegacion_decision_cobertura(
  ambito_raiz_hmac text PRIMARY KEY,
  decision_vec_ref text NOT NULL UNIQUE,
  prueba_canonica bytea NOT NULL,
  prueba_huella_sha256 text NOT NULL,
  contexto_recurso_canonico bytea NOT NULL,
  contexto_recurso_huella_sha256 text NOT NULL,
  motivo_canonico bytea NOT NULL,
  motivo_huella_sha256 text NOT NULL,
  limite_preparacion timestamptz(6) NOT NULL,
  registrada_en timestamptz(6) NOT NULL,
  FOREIGN KEY(ambito_raiz_hmac) REFERENCES
    vec_contratacion_temporal.terminal_operacion_decision_cobertura
    ON UPDATE RESTRICT ON DELETE RESTRICT
    DEFERRABLE INITIALLY DEFERRED,
  FOREIGN KEY(ambito_raiz_hmac,decision_vec_ref) REFERENCES
    vec_contratacion_temporal.terminal_operacion_decision_cobertura(
      ambito_raiz_hmac,decision_vec_ref)
    ON UPDATE RESTRICT ON DELETE RESTRICT
    DEFERRABLE INITIALLY DEFERRED,
  UNIQUE(ambito_raiz_hmac,decision_vec_ref,prueba_huella_sha256),
  CHECK (decision_vec_ref ~
    '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
  CHECK (pg_catalog.encode(pg_catalog.sha256(prueba_canonica),'hex')=
    prueba_huella_sha256),
  CHECK (pg_catalog.encode(
    pg_catalog.sha256(contexto_recurso_canonico),'hex')=
    contexto_recurso_huella_sha256),
  CHECK (pg_catalog.encode(pg_catalog.sha256(motivo_canonico),'hex')=
    motivo_huella_sha256),
  CHECK (limite_preparacion>registrada_en)
);

CREATE TRIGGER bloquear_mutacion
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.prueba_denegacion_decision_cobertura
FOR EACH ROW EXECUTE FUNCTION
  vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER bloquear_truncado
BEFORE TRUNCATE
ON vec_contratacion_temporal.prueba_denegacion_decision_cobertura
FOR EACH STATEMENT EXECUTE FUNCTION
  vec_contratacion_temporal.rechazar_mutacion_historia_v1();
ALTER TABLE vec_contratacion_temporal.prueba_denegacion_decision_cobertura
  ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.prueba_denegacion_decision_cobertura
  FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
ON vec_contratacion_temporal.prueba_denegacion_decision_cobertura
FOR ALL TO vec_contratacion_temporal_propietario
USING (true) WITH CHECK (true);

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema=9,
       actualizada_en=
         pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=8;

REVOKE ALL ON TABLE
  vec_contratacion_temporal.prueba_denegacion_decision_cobertura
FROM PUBLIC,vec_contratacion_temporal_ejecutor,
  vec_contratacion_temporal_confirmador_cobertura,
  vec_contratacion_temporal_migrador,
  vec_contratacion_temporal_gobernador;
REVOKE ALL ON FUNCTION
  vec_contratacion_temporal.o404e_texto_json_go_v1(text),
  vec_contratacion_temporal.o404e_mapa_json_go_v1(jsonb),
  vec_contratacion_temporal.o404e_contexto_recurso_denegacion_v1(jsonb),
  vec_contratacion_temporal.o404e_motivo_denegacion_canon_v1(jsonb)
FROM PUBLIC,vec_contratacion_temporal_ejecutor,
  vec_contratacion_temporal_confirmador_cobertura,
  vec_contratacion_temporal_migrador,
  vec_contratacion_temporal_gobernador;
COMMIT;
