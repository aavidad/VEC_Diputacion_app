\set ON_ERROR_STOP on
-- B113 amplía B67/B68 y puede haber consumos nominales e historia nativa.
-- La reversión exige un procedimiento histórico propio; no se ejecuta DOWN.
DO $denegado$
BEGIN
 RAISE EXCEPTION 'B113: DOWN prohibido; conservar consultas y acuses originales'
  USING ERRCODE='55000';
END $denegado$;
