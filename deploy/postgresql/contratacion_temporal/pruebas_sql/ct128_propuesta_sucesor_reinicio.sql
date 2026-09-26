\set ON_ERROR_STOP on
-- CT128 tras reiniciar PostgreSQL: se repiten con el mismo material el aviso,
-- la respuesta, la resolución y la propuesta de B (recibos originales, estado
-- de repetición) y la confirmación de GINPIX, el cese y el cierre (mismo
-- recibo); la historia del expediente no gana ninguna fila y la consulta del
-- panel devuelve lo mismo. Requiere ct124_utilidades.sql en la misma sesión.
SET SESSION AUTHORIZATION vec_ct115_runtime;
SET timezone = 'UTC';
CREATE TEMP TABLE replay (caso text, valor jsonb);
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO replay SELECT caso, prueba_ct128.repetir(caso) FROM prueba_ct128.caso
 WHERE caso IN ('aviso','respuesta','resolucion','propuesta') ORDER BY caso;
COMMIT;
SELECT prueba_ct128.exigir(count(*)=4 AND bool_and(r.valor->>'Estado' LIKE 'replay%' AND r.valor-'Estado'=c.resultado-'Estado'),
    'cuatro repeticiones devuelven el recibo original')
  FROM replay r JOIN prueba_ct128.caso c USING (caso);
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',prueba_ct128.v('ginpix')::jsonb)::text AS g2 \gset
SELECT pg_temp.confirmar('confirmar_cese_nombramiento_v1',prueba_ct128.v('cese')::jsonb)::text AS c2 \gset
SELECT pg_temp.confirmar('confirmar_cierre_expediente_v1',prueba_ct128.v('cierre')::jsonb)::text AS k2 \gset
COMMIT;
SELECT prueba_ct128.exigir((:'g2'::jsonb)->'recibo'=(prueba_ct128.v('r_g')::jsonb)->'recibo'
    AND (:'c2'::jsonb)->'recibo'=(prueba_ct128.v('r_cese')::jsonb)->'recibo'
    AND (:'k2'::jsonb)->'recibo'=(prueba_ct128.v('r_cierre')::jsonb)->'recibo','GINPIX, cese y cierre de B repetidos con el mismo recibo');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.consultar_propuestas_expediente_v1(prueba_ct128.v('org'),prueba_ct128.v('exp_a'))::text AS consulta \gset
COMMIT;
SELECT prueba_ct128.exigir(:'consulta'::jsonb=prueba_ct128.v('consulta')::jsonb,'la consulta del panel no cambia');
RESET SESSION AUTHORIZATION;
SELECT prueba_ct128.exigir(prueba_ct128.conteo(prueba_ct128.v('exp_a'))=prueba_ct128.v('conteo'),
    'ninguna fila nueva tras el reinicio: '||prueba_ct128.conteo(prueba_ct128.v('exp_a')));
SELECT 'reinicio CT128 OK';
