-- Sesion A: ejecutar en una transaccion sin cerrar hasta que B haya intentado
-- abrir. La cerradura por cuenta impide dos sesiones vigentes para la misma
-- referencia opaca.
BEGIN;
SET LOCAL ROLE vec_gateway_personal_ejecutor;
SELECT * FROM vec_gateway_personal.abrir_sesion_v1(
    repeat('e', 64), 'cta_zyxwvutsrqponmlkjihgfe', repeat('f', 64),
    pg_catalog.statement_timestamp(),
    pg_catalog.statement_timestamp() + interval '10 minutes'
);
-- Mantener abierta esta transaccion; confirmar solo despues de ejecutar B.
