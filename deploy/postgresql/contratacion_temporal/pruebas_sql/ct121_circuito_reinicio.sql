\set ON_ERROR_STOP on
-- Tras reiniciar PostgreSQL: los siete escritores del circuito tras la
-- expiración (contacto, expiración, continuación, aviso, respuesta,
-- resolución y propuesta) repiten su material y devuelven el mismo recibo
-- sin efecto nuevo; la consulta del justificante devuelve la misma
-- continuación; la historia no cambia.
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
CREATE TEMP TABLE conteo_antes AS
SELECT (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh) AS resoluciones,
       (SELECT count(*) FROM vec_contratacion_temporal.comunicacion_llamamiento_local) AS avisos,
       (SELECT count(*) FROM vec_contratacion_temporal.respuesta_recibida_rrhh) AS respuestas,
       (SELECT count(*) FROM vec_contratacion_temporal.propuesta_formalizacion) AS propuestas,
       (SELECT count(*) FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh) AS plazos,
       (SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral) AS outbox;
GRANT SELECT ON conteo_antes TO vec_ct121c_runtime;
SET SESSION AUTHORIZATION vec_ct121c_runtime;
CREATE TEMP TABLE replay (caso text PRIMARY KEY, valor jsonb NOT NULL);
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO replay SELECT caso, prueba_ct121c.repetir(caso) FROM prueba_ct121c.caso
 WHERE caso IN ('contacto','expiracion','continuacion','aviso','respuesta','justificante','resolucion','propuesta') ORDER BY caso;
COMMIT;
SELECT prueba_ct121c.exigir(count(*)=7 AND bool_and(r.valor->>'Estado' LIKE 'replay%'
        AND r.valor-'Estado'=c.resultado-'Estado'),'tras el reinicio, siete replays con el recibo original')
  FROM replay r JOIN prueba_ct121c.caso c USING (caso) WHERE caso<>'justificante';
SELECT prueba_ct121c.exigir(r.valor=c.resultado,'tras el reinicio, el justificante devuelve la misma continuación')
  FROM replay r JOIN prueba_ct121c.caso c USING (caso) WHERE caso='justificante';
RESET SESSION AUTHORIZATION;
SELECT prueba_ct121c.exigir(
    a.resoluciones=(SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh)
    AND a.avisos=(SELECT count(*) FROM vec_contratacion_temporal.comunicacion_llamamiento_local)
    AND a.respuestas=(SELECT count(*) FROM vec_contratacion_temporal.respuesta_recibida_rrhh)
    AND a.propuestas=(SELECT count(*) FROM vec_contratacion_temporal.propuesta_formalizacion)
    AND a.plazos=(SELECT count(*) FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh)
    AND a.outbox=(SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral)
    AND (SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=:'exp_b')=7,
    'tras el reinicio, sin filas nuevas') FROM conteo_antes a;
SELECT 'CT121 reinicio OK' AS resultado;
