\set ON_ERROR_STOP on

DO $acl$
DECLARE
  v_fuente text;
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
       AND NOT rolbypassrls
  ) OR EXISTS(
    SELECT 1 FROM pg_catalog.pg_auth_members m
    JOIN pg_catalog.pg_roles r ON r.oid=m.member
     WHERE r.rolname=
       'vec_contratacion_temporal_lector_resultado_cobertura'
  ) OR NOT pg_catalog.has_schema_privilege(
    'vec_contratacion_temporal_lector_resultado_cobertura',
    'vec_contratacion_temporal','USAGE'
  ) OR pg_catalog.has_schema_privilege(
    'vec_contratacion_temporal_lector_resultado_cobertura',
    'vec_contratacion_temporal','CREATE'
  ) OR NOT pg_catalog.has_database_privilege(
    'vec_contratacion_temporal_lector_resultado_cobertura',
    pg_catalog.current_database(),'CONNECT'
  ) OR pg_catalog.has_database_privilege(
    'vec_contratacion_temporal_lector_resultado_cobertura',
    pg_catalog.current_database(),'CREATE'
  ) OR pg_catalog.has_database_privilege(
    'vec_contratacion_temporal_lector_resultado_cobertura',
    pg_catalog.current_database(),'TEMP'
  ) OR EXISTS(
    SELECT 1 FROM pg_catalog.pg_class c
    JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
     WHERE n.nspname='vec_contratacion_temporal'
       AND pg_catalog.has_table_privilege(
         'vec_contratacion_temporal_lector_resultado_cobertura',
         c.oid,
         'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
  ) THEN
    RAISE EXCEPTION 'ACL del lector O4-05 no es mínima';
  END IF;
  IF NOT pg_catalog.has_function_privilege(
       'vec_contratacion_temporal_lector_resultado_cobertura',
       'vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)',
       'EXECUTE'
     ) OR EXISTS(
    SELECT 1
      FROM pg_catalog.pg_roles r
     WHERE r.rolname IN (
       'vec_contratacion_temporal_migrador',
       'vec_contratacion_temporal_gobernador',
       'vec_contratacion_temporal_ejecutor',
       'vec_contratacion_temporal_confirmador_cobertura'
     ) AND pg_catalog.has_function_privilege(
       r.rolname,
       'vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)',
       'EXECUTE')
  ) OR pg_catalog.has_function_privilege(
       'public',
       'vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)',
       'EXECUTE'
     ) THEN
    RAISE EXCEPTION 'EXECUTE exterior O4-05 no está cerrado';
  END IF;
  SELECT p.prosrc INTO STRICT v_fuente
    FROM pg_catalog.pg_proc p
    JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_contratacion_temporal'
     AND p.proname=
       'recuperar_resultado_propio_decision_cobertura_o405_v1'
     AND p.prosecdef AND p.provolatile='s';
  IF v_fuente!~'gobi_o404b_entorno_valido[(]true[)]'
     OR v_fuente!~'o404e_leer_terminal_interno_v1'
     OR v_fuente~'gobi_o404b_catalogo|gobi_o404b_actual' THEN
    RAISE EXCEPTION
      'lector O4-05 no fija entorno, depende del catálogo o no usa lector fuerte';
  END IF;
END
$acl$;

-- Añade un alias posterior al terminal denegado para probar rotación histórica.
INSERT INTO vec_contratacion_temporal
  .alias_operacion_decision_cobertura(
    alias_ambito_hmac,ambito_raiz_hmac,generacion,
    alias_huella_semantica_hmac,registrada_en)
SELECT
  'hmac-sha256:vec.contratacion-temporal.cobertura-decision.ambito/v2:'
    ||pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(a.ambito_raiz_hmac||':o405:ambito','UTF8')
    ),'hex'),
  a.ambito_raiz_hmac,2,
  'hmac-sha256:vec.contratacion-temporal.cobertura-decision.semantica/v2:'
    ||pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(a.ambito_raiz_hmac||':o405:semantica','UTF8')
    ),'hex'),
  pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
FROM vec_contratacion_temporal.alias_operacion_decision_cobertura a
JOIN vec_contratacion_temporal.confirmacion_operacion_decision_cobertura c
  USING(ambito_raiz_hmac)
WHERE c.rama='denegada' AND a.generacion=1
  AND NOT EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .alias_operacion_decision_cobertura x
     WHERE x.ambito_raiz_hmac=a.ambito_raiz_hmac AND x.generacion=2)
LIMIT 1;

WITH terminales AS (
  SELECT b.ambito_raiz_hmac,b.organizacion_ref,b.expediente_ref,
         b.version_expediente,c.rama
    FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura b
    JOIN vec_contratacion_temporal
      .confirmacion_operacion_decision_cobertura c
      USING(ambito_raiz_hmac)
), consultas AS (
  SELECT t.rama,t.organizacion_ref,t.expediente_ref,
         pg_catalog.jsonb_build_object(
           'esquema',
            'vec.contratacion-temporal.consulta-recuperacion-propia-decision-cobertura.o4-05.v1',
           'organizacion_ref',t.organizacion_ref,
           'expediente_ref',t.expediente_ref,
           'ambitos_idempotencia_hmac',(
             SELECT pg_catalog.jsonb_agg(x.alias_ambito_hmac
                                        ORDER BY x.generacion DESC)
               FROM (
                 SELECT a.alias_ambito_hmac,a.generacion
                   FROM vec_contratacion_temporal
                     .alias_operacion_decision_cobertura a
                  WHERE a.ambito_raiz_hmac=t.ambito_raiz_hmac
                  ORDER BY a.generacion DESC LIMIT 4
               ) x)
         ) AS consulta
    FROM terminales t
)
SELECT
  (SELECT consulta::text FROM consultas
    WHERE rama='denegada' LIMIT 1) AS consulta_denegada,
  (SELECT consulta::text FROM consultas
    WHERE rama='concedida' LIMIT 1) AS consulta_concedida,
  (SELECT pg_catalog.jsonb_set(
      consulta,'{expediente_ref}',
      pg_catalog.to_jsonb(
        (SELECT expediente_ref FROM consultas
          WHERE rama='concedida' LIMIT 1)))::text
     FROM consultas WHERE rama='denegada' LIMIT 1) AS consulta_cruzada,
  (SELECT pg_catalog.jsonb_set(
      d.consulta,'{ambitos_idempotencia_hmac}',
      pg_catalog.jsonb_build_array(
        d.consulta#>>'{ambitos_idempotencia_hmac,0}',
        c.consulta#>>'{ambitos_idempotencia_hmac,0}'))::text
     FROM consultas d CROSS JOIN consultas c
    WHERE d.rama='denegada' AND c.rama='concedida' LIMIT 1
  ) AS consulta_divergente
\gset

CREATE FUNCTION pg_temp.o405_exigir_rechazo(
  p_consulta jsonb,p_sqlstate text
)
RETURNS void
LANGUAGE plpgsql
SECURITY INVOKER
SET search_path=pg_catalog
AS $$
BEGIN
  BEGIN
    PERFORM resultado_json
      FROM vec_contratacion_temporal
        .recuperar_resultado_propio_decision_cobertura_o405_v1(
          p_consulta);
  EXCEPTION WHEN OTHERS THEN
    IF SQLSTATE=p_sqlstate THEN RETURN; END IF;
    RAISE;
  END;
  RAISE EXCEPTION 'consulta O4-05 que debía fallar fue aceptada';
END
$$;
REVOKE ALL ON FUNCTION pg_temp.o405_exigir_rechazo(jsonb,text)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION pg_temp.o405_exigir_rechazo(jsonb,text)
 TO vec_contratacion_temporal_lector_resultado_cobertura;

SET SESSION AUTHORIZATION vec_o405_lector;
-- La misma consulta válida solo cruza la fachada en el entorno lector exacto.
BEGIN ISOLATION LEVEL READ COMMITTED READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_temp.o405_exigir_rechazo(:'consulta_denegada'::jsonb,'42501');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_temp.o405_exigir_rechazo(:'consulta_denegada'::jsonb,'42501');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='0';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_temp.o405_exigir_rechazo(:'consulta_denegada'::jsonb,'42501');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15001ms';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_temp.o405_exigir_rechazo(:'consulta_denegada'::jsonb,'42501');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='0';
SELECT pg_temp.o405_exigir_rechazo(:'consulta_denegada'::jsonb,'42501');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20001ms';
SELECT pg_temp.o405_exigir_rechazo(:'consulta_denegada'::jsonb,'42501');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT resultado_json AS resultado_denegado
  FROM vec_contratacion_temporal
    .recuperar_resultado_propio_decision_cobertura_o405_v1(
      :'consulta_denegada'::jsonb)
\gset
SELECT resultado_json AS resultado_concedido
  FROM vec_contratacion_temporal
    .recuperar_resultado_propio_decision_cobertura_o405_v1(
      :'consulta_concedida'::jsonb)
\gset
SELECT resultado_json AS resultado_ausente
  FROM vec_contratacion_temporal
    .recuperar_resultado_propio_decision_cobertura_o405_v1(
      pg_catalog.jsonb_build_object(
        'esquema',
          'vec.contratacion-temporal.consulta-recuperacion-propia-decision-cobertura.o4-05.v1',
        'organizacion_ref','organizacion:o405:ausente',
        'expediente_ref','expediente:o405:ausente',
        'ambitos_idempotencia_hmac',pg_catalog.jsonb_build_array(
          'hmac-sha256:vec.contratacion-temporal.cobertura-decision.ambito/v9:'
          ||pg_catalog.repeat('9',64))))
\gset
WITH claves_exactas AS (
  SELECT ARRAY[
    'actuacion_ref','ambito_idempotencia_hmac','auditoria_ref',
    'correlacion_vec_ref','decision_vec_ref','esquema','estado',
    'evento_ref','expediente_ref','huella_semantica_hmac',
    'observada_en','observada_en_db','organizacion_ref','recibo',
    'recibo_ref','reserva_ref','revision_cercado','version_expediente'
  ]::text[] AS confirmada,
  ARRAY['esquema','estado','observada_en']::text[] AS no_observable
)
SELECT
  (:'resultado_denegado'::jsonb->>'estado'='confirmado'
   AND (:'resultado_denegado'::jsonb->'recibo'->>'denegada_vec')::boolean
   AND (SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
          FROM pg_catalog.jsonb_object_keys(
            :'resultado_denegado'::jsonb) AS k(clave))=claves_exactas.confirmada
   AND :'resultado_concedido'::jsonb->>'estado'='confirmado'
   AND (:'resultado_concedido'::jsonb->'recibo'->>'aplicada')::boolean
   AND (SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
          FROM pg_catalog.jsonb_object_keys(
            :'resultado_concedido'::jsonb) AS k(clave))=claves_exactas.confirmada
   AND :'resultado_ausente'::jsonb->>'estado'='no_observable'
   AND (SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
          FROM pg_catalog.jsonb_object_keys(
            :'resultado_ausente'::jsonb) AS k(clave))=
        claves_exactas.no_observable) AS uniones_exactas
FROM claves_exactas
\gset
\if :uniones_exactas
\else
  SELECT 1/0;
\endif
SELECT resultado_json AS resultado_cruzado
  FROM vec_contratacion_temporal
    .recuperar_resultado_propio_decision_cobertura_o405_v1(
      :'consulta_cruzada'::jsonb)
\gset
SELECT (:'resultado_cruzado'::jsonb->>'estado'='no_observable'
        AND (SELECT count(*) FROM pg_catalog.jsonb_object_keys(
          :'resultado_cruzado'::jsonb))=3) AS cruce_no_observable
\gset
\if :cruce_no_observable
\else
  SELECT 1/0;
\endif
SELECT pg_temp.o405_exigir_rechazo(
  :'consulta_divergente'::jsonb,'23505');
COMMIT;
RESET SESSION AUTHORIZATION;

-- Una huella de recibo corrupta no se degrada a no_observable.
SELECT ambito_raiz_hmac,recibo_huella_sha256 AS huella_original
  FROM vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
 WHERE rama='denegada' LIMIT 1
\gset corrupcion_
BEGIN;
SET LOCAL session_replication_role='replica';
UPDATE vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
   SET recibo_huella_sha256=pg_catalog.repeat('f',64)
 WHERE ambito_raiz_hmac=:'corrupcion_ambito_raiz_hmac';
COMMIT;
SET SESSION AUTHORIZATION vec_o405_lector;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT pg_temp.o405_exigir_rechazo(
  :'consulta_denegada'::jsonb,'55000');
COMMIT;
RESET SESSION AUTHORIZATION;
BEGIN;
SET LOCAL session_replication_role='replica';
UPDATE vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
   SET recibo_huella_sha256=:'corrupcion_huella_original'
 WHERE ambito_raiz_hmac=:'corrupcion_ambito_raiz_hmac';
COMMIT;
