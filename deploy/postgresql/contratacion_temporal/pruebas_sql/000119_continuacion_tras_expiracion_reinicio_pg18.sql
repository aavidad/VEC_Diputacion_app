\set ON_ERROR_STOP on
-- Tras reiniciar PostgreSQL: la continuación tras la expiración se recupera
-- idéntica y el replay no añade confirmaciones.
SET SESSION AUTHORIZATION vec_ct111_runtime;
CREATE TEMP TABLE tras_reinicio119 (caso text PRIMARY KEY, valor jsonb NOT NULL);
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO tras_reinicio119 SELECT 'continuacion1', prueba_ct119.v2(material)
  FROM prueba_ct119.antes_reinicio WHERE caso='continuacion1';
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT prueba_ct111.exigir((SELECT t.valor-'Estado'=a.valor-'Estado' AND t.valor->>'Estado'='replay_confirmado'
    FROM tras_reinicio119 t, prueba_ct119.antes_reinicio a WHERE t.caso='continuacion1' AND a.caso='continuacion1'),
    'continuación tras expiración idéntica tras reinicio');
SELECT prueba_ct111.exigir((SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    WHERE continuacion_clave IS NOT NULL)=2,'sin confirmaciones duplicadas tras reinicio');
