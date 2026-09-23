\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
-- DOWN sólo sobre una instalación vacía, nunca sobre historia conservada.
LOCK TABLE vec_cronos_v1.marcaje_original,vec_cronos_v1.marcaje_historia,vec_cronos_v1.marcaje_outbox,vec_cronos_v1.marcaje_acceso IN ACCESS EXCLUSIVE MODE;
DO $pre$
DECLARE tabla text; existente boolean;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
        RAISE EXCEPTION 'reversión Cronos requiere DBA' USING ERRCODE='42501';
    END IF;
    FOREACH tabla IN ARRAY ARRAY['marcaje_original','marcaje_historia','marcaje_outbox','marcaje_acceso'] LOOP
        -- El DBA comprueba todas las filas, incluidas las protegidas por RLS.
        EXECUTE format('SELECT EXISTS (SELECT 1 FROM vec_cronos_v1.%I)',tabla) INTO existente;
        IF existente THEN RAISE EXCEPTION 'Cronos conserva historia' USING ERRCODE='55000'; END IF;
    END LOOP;
END
$pre$;
-- La función de 000002 y su dependencia AD3 deben retirarse previamente.
DROP TABLE vec_cronos_v1.marcaje_acceso,vec_cronos_v1.marcaje_outbox,vec_cronos_v1.marcaje_historia,vec_cronos_v1.marcaje_original;
DROP FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
DROP SCHEMA vec_cronos_v1;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_cronos_v1_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
DROP ROLE vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_propietario;
COMMIT;
