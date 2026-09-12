\set ON_ERROR_STOP on
-- T13/5 conserva auditoría común; no revierte cadena, retención ni permisos usados.
DO $down$
BEGIN
    RAISE EXCEPTION 'T13/5: reversión automática denegada; conservar auditoría e historial' USING ERRCODE='55000';
END $down$;
