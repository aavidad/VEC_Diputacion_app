\if :{?confirmar_destruccion_contexto_actor_v1}
\else
\echo 'falta -v confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1'
\quit 3
\endif
SELECT :'confirmar_destruccion_contexto_actor_v1' = 'DESTRUIR_CONTEXTO_ACTOR_V1' AS confirmacion_valida \gset
\if :confirmacion_valida
\else
\echo 'confirmacion de destruccion incorrecta'
\quit 3
\endif

BEGIN;
SET LOCAL search_path = pg_catalog;
DO $base$
BEGIN
    EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_contexto_actor_v1_runtime, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_propietario', current_database());
END
$base$;
REVOKE vec_contexto_actor_v1_propietario FROM vec_contexto_actor_v1_migrador;
DROP ROLE vec_contexto_actor_v1_runtime;
DROP ROLE vec_contexto_actor_v1_migrador;
DROP ROLE vec_contexto_actor_v1_propietario;
COMMIT;
