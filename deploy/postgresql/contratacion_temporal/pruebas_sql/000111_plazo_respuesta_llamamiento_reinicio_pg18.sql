\set ON_ERROR_STOP on
-- Tras reiniciar PostgreSQL: los recibos, vencimientos y la expiración se
-- recuperan idénticos y un replay no añade eventos ni resoluciones.
SET SESSION AUTHORIZATION vec_ct111_runtime;
CREATE TEMP TABLE tras_reinicio (caso text PRIMARY KEY, valor jsonb NOT NULL);
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO tras_reinicio SELECT 'contacto1', prueba_ct111.plazo((SELECT
    format('{"Solicitud":%s,"Plazo":%s}',valor->'Solicitud',valor->'Plazo')
    FROM prueba_ct111.antes_reinicio WHERE caso='contacto1'));
INSERT INTO tras_reinicio SELECT 'expiracion1', prueba_ct111.resolver(prueba_ct111.material_resolucion(1,
    '22222222-2222-4222-8222-000000000001','expiracion_gobernada',prueba_ct111.referencia('b08.sin_respuesta_baja')));
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT prueba_ct111.exigir((SELECT t.valor-'Estado'=a.valor-'Estado' AND t.valor->>'Estado'='replay_registrado'
    FROM tras_reinicio t, prueba_ct111.antes_reinicio a WHERE t.caso='contacto1' AND a.caso='contacto1'),
    'contacto efectivo y vencimiento idénticos tras reinicio');
SELECT prueba_ct111.exigir((SELECT t.valor-'Estado'=a.valor-'Estado' AND t.valor->>'Estado'='replay_confirmado'
    FROM tras_reinicio t, prueba_ct111.antes_reinicio a WHERE t.caso='expiracion1' AND a.caso='expiracion1'),
    'expiración confirmada idéntica tras reinicio');
SELECT prueba_ct111.exigir((SELECT count(*) FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh)=7
    AND (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh)=5
    AND (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
          WHERE estado_plazo='expirado' AND justificante_ref IS NULL AND contacto_ref IS NOT NULL)=1,
    'sin duplicados tras reinicio');
