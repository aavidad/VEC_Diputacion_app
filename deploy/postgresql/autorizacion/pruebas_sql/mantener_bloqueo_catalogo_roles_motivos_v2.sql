\set ON_ERROR_STOP 1

-- El bloqueo de sesion permite conectar de antemano tanto down como GRANT.
-- Despues se congela pg_database y se libera solo el advisory: down completa
-- su preflight de roles pero espera antes de DROP, mientras GRANT intenta
-- resolver los roles desde otra base.
SELECT pg_advisory_lock(
    hashtextextended('vec_autorizacion:roles_motivos_v2:down:v1', 0)
);
SELECT pg_sleep(4);
BEGIN;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;
SELECT pg_advisory_unlock(
    hashtextextended('vec_autorizacion:roles_motivos_v2:down:v1', 0)
);
SELECT pg_sleep(4);
ROLLBACK;
