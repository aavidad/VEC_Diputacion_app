\set ON_ERROR_STOP on
-- Semilla de la copia desechable de Cronos. Usa la misma migración revisada.
-- Ejecutar una vez en la copia, después de la proyección de motivos V2.
\ir ../../../postgresql/cronos_v1/migraciones/000018_catalogo_motivos_cronos_v1.up.sql
