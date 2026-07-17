\set ON_ERROR_STOP 1

-- La conexion se establece antes del bloqueo de pg_database. El GRANT se
-- inicia despues del preflight y debe reanudarse tras COMMIT con rol inexistente.
SELECT pg_sleep(5);
GRANT vec_autorizacion_motivos_evaluador
    TO vec_autorizacion_migrador_prueba;
