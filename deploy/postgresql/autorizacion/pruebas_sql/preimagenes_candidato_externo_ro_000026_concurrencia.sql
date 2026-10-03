\set ON_ERROR_STOP on
-- Ejecutar mientras otro cliente del clon mantiene FOR UPDATE sobre checkpoint.
-- La lectura debe terminar antes de 2 s y nunca adquirir FOR SHARE.
SET SESSION AUTHORIZATION vec_externo_v3_fuente_autorizacion_desarrollo;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL lock_timeout = '250ms';
SET LOCAL statement_timeout = '2s';
DO $lectura$
DECLARE d jsonb; m jsonb;
BEGIN
    m := vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1();
    d := vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(
        'candidato_bolsa_historial_propio_desarrollo','rol:candidato_bolsa_historial_propio_desarrollo:v1',
        'principal:aut26:sintetico','perfil:aut26:activa');
    IF m #>> '{checkpoint,coherente}' IS DISTINCT FROM 'true'
       OR d #>> '{asignacion,estado_observacion}' IS DISTINCT FROM 'activa'
    THEN RAISE EXCEPTION 'observación incoherente durante escritor concurrente'; END IF;
END $lectura$;
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT 'AUT26_CONCURRENCIA_RO_OK' AS resultado;
