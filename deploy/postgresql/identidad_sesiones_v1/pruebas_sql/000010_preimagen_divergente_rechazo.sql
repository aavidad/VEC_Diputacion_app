\set ON_ERROR_STOP on
-- Ejecutar sólo en PostgreSQL 18 desechable con IS6 y antes de instalar IS10.
-- Resultado esperado: psql termina con código 3 / IS10: preimagen divergente.
-- La conexión revierte TODO, incluido el cambio de configuración artificial.
-- Otra conexión debe comprobar ausencia de la tabla IS10 y definición intacta.
BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
ALTER FUNCTION vec_identidad_sesiones_v1.registrar_sesion_v1(
 text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,
 timestamptz,timestamptz,timestamptz,text,text) SET search_path=pg_catalog;
RESET ROLE;
\ir ../migraciones/000010_politica_high_rrhh_rpt_sintetica.up.sql
