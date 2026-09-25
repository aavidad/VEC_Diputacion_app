\set ON_ERROR_STOP on
-- Tras reiniciar PostgreSQL: el terminal «sin respuesta» se recupera idéntico.
SET SESSION AUTHORIZATION vec_b39_runtime;
SET TimeZone='UTC';
SET statement_timeout='15s';
SET idle_in_transaction_session_timeout='20s';
CREATE TEMP TABLE r39b (recibo text, evento text, confirmada timestamptz);
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r39b SELECT g.recibo_ref,g.evento_ref,g.confirmada_en
  FROM prueba_b39.guardar(prueba_b39.canonico('operacion:expiracion'),'bolsa.llamamiento.renuncia_rrhh.registrar') g;
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT prueba_b39.exigir((SELECT r.recibo=o.recibo_ref AND r.confirmada=o.confirmada_en
    FROM r39b r, vec_bolsa_llamamientos.integracion_desarrollo o WHERE o.operacion_ref='operacion:expiracion')
    AND (SELECT count(*) FROM vec_bolsa_llamamientos.integracion_desarrollo)=4,
    'terminal sin respuesta idéntico tras reinicio, sin duplicados');
