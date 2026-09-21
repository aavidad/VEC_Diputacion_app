\set ON_ERROR_STOP on
-- AD3-44 amplía un núcleo compartido posterior a AD3-32/43 y crea historia
-- de consumo. La reversión coherente es rechazar DOWN: retirar sus perfiles o
-- fachadas podría invalidar decisiones y auditoría ya conservadas.
DO $f$
BEGIN
 RAISE EXCEPTION 'AD3-44: DOWN no admitido; conservar AD3-32/43, gobierno, consumo y auditoría' USING ERRCODE='55000';
END $f$;
