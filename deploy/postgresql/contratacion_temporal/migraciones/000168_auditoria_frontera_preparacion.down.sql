\set ON_ERROR_STOP on
-- CT168 conserva auditoría de solo adición. Recuperar aplicación con un
-- artefacto compatible no exige retirar estas parejas nominales ni sus filas.
-- El ensayo se descarta mediante ROLLBACK; no usar DOWN sobre su historia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $retirada$
BEGIN
 RAISE EXCEPTION 'CT168: DOWN no ejecutable; conservar auditoría y recuperar un artefacto compatible'
 USING ERRCODE='55000';
END $retirada$;
ROLLBACK;
