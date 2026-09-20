-- Sesion B: ejecutar la migracion DOWN; debe bloquear hasta que A confirme y
-- finalizar con SQLSTATE 55000 porque la historia ya es visible.
\set ON_ERROR_STOP on
\i deploy/postgresql/gateway_personal/migraciones/000001_sesion_gateway.down.sql
