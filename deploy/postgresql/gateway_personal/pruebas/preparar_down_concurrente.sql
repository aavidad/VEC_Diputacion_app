-- Sesion A: mantener abierta tras insertar una sesion. El DOWN de la sesion B
-- debe esperar al bloqueo de esta transaccion y despues rechazar la historia.
BEGIN;
SET LOCAL ROLE vec_gateway_personal_ejecutor;
SELECT * FROM vec_gateway_personal.abrir_sesion_v1(
    repeat('d', 64), 'cta_qwertyuiopasdfghjklzxc', repeat('e', 64),
    pg_catalog.statement_timestamp(),
    pg_catalog.statement_timestamp() + interval '10 minutes'
);
-- Ejecutar COMMIT solo despues de iniciar 000001_sesion_gateway.down.sql en B.
