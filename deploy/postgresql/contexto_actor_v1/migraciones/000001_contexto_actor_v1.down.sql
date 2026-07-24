\if :{?confirmar_destruccion_contexto_actor_v1}
\else
\echo 'falta confirmacion destructiva de contexto actor V1'
\quit 3
\endif
SELECT :'confirmar_destruccion_contexto_actor_v1' = 'DESTRUIR_CONTEXTO_ACTOR_V1' AS confirmacion_valida \gset
\if :confirmacion_valida
\else
\echo 'confirmacion destructiva incorrecta'
\quit 3
\endif
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
DROP SCHEMA vec_contexto_actor_v1 CASCADE;
COMMIT;
