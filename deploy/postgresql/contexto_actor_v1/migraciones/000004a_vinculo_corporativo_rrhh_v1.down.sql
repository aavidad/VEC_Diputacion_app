-- Retirada de la correccion 000004a del vinculo corporativo RRHH.
-- No se admite: 000004a solo se instala en bases con historia (o en lugar de
-- 000004) y su postimagen incluye consumidores posteriores (000006, 000007,
-- Identidad 000004). La retirada exige un procedimiento revisado propio.
\set ON_ERROR_STOP on
DO $f$ BEGIN
    RAISE EXCEPTION 'ContextoActor 000004a: DOWN no admitido con historia' USING ERRCODE = '55000';
END $f$;
