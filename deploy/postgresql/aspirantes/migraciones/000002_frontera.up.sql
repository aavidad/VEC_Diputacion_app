\set ON_ERROR_STOP on
-- Aspirantes 000002: registro de denegaciones en la frontera del portal
-- externo (401/403 de la ruta de la ficha propia). Solo guarda motivo,
-- correlación e instante: ni persona, ni ruta libre, ni detalle técnico.
-- Aplicar tras 000001.
BEGIN;
SET LOCAL ROLE vec_aspirantes_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_aspirantes:migracion:000002',0));
DO $pre$ BEGIN
 IF current_user<>'vec_aspirantes_propietario'
    OR to_regclass('vec_aspirantes.ficha') IS NULL
    OR to_regprocedure('vec_aspirantes.sesion_valida()') IS NULL
    OR to_regclass('vec_aspirantes.denegacion_frontera') IS NOT NULL
 THEN RAISE EXCEPTION 'Aspirantes 000002: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TABLE vec_aspirantes.denegacion_frontera (
 denegacion_ref text PRIMARY KEY CHECK(denegacion_ref ~ '^aspden_[0-9a-f]{32}$'),
 correlacion_ref text NOT NULL CHECK(correlacion_ref ~ '^corr_([0-9a-f]{32}|no_disponible)$'),
 motivo text NOT NULL CHECK(motivo IN ('autenticacion_requerida','acceso_denegado')),
 ocurrido_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_aspirantes.denegacion_frontera ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_aspirantes.denegacion_frontera FORCE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE vec_aspirantes.denegacion_frontera FROM PUBLIC,vec_aspirantes_ejecutor_externo;
CREATE POLICY denegacion_alta ON vec_aspirantes.denegacion_frontera FOR INSERT TO vec_aspirantes_propietario
 WITH CHECK (vec_aspirantes.sesion_valida());
CREATE TRIGGER denegacion_frontera_inmutable BEFORE UPDATE OR DELETE ON vec_aspirantes.denegacion_frontera
 FOR EACH ROW EXECUTE FUNCTION vec_aspirantes.rechazar_cambio_inmutable();
CREATE TRIGGER denegacion_frontera_no_truncar BEFORE TRUNCATE ON vec_aspirantes.denegacion_frontera
 FOR EACH STATEMENT EXECUTE FUNCTION vec_aspirantes.rechazar_cambio_inmutable();

CREATE FUNCTION vec_aspirantes.registrar_denegacion_frontera_v1(p_motivo text,p_correlacion text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE ref text;
BEGIN
 IF NOT vec_aspirantes.sesion_valida() OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Aspirantes: registro de frontera denegado' USING ERRCODE='42501'; END IF;
 IF p_motivo IS NULL OR p_motivo NOT IN ('autenticacion_requerida','acceso_denegado')
    OR p_correlacion IS NULL OR p_correlacion !~ '^corr_([0-9a-f]{32}|no_disponible)$'
 THEN RAISE EXCEPTION 'Aspirantes: denegación inválida' USING ERRCODE='22023'; END IF;
 ref:='aspden_'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_aspirantes.denegacion_frontera(denegacion_ref,correlacion_ref,motivo,ocurrido_en)
 VALUES(ref,p_correlacion,p_motivo,date_trunc('microseconds',clock_timestamp()));
 RETURN ref;
END $f$;
REVOKE ALL ON FUNCTION vec_aspirantes.registrar_denegacion_frontera_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_aspirantes.registrar_denegacion_frontera_v1(text,text) TO vec_aspirantes_ejecutor_externo;

DO $acl$ BEGIN
 IF (SELECT count(*) FROM pg_proc p WHERE p.pronamespace='vec_aspirantes'::regnamespace
     AND has_function_privilege('vec_aspirantes_ejecutor_externo',p.oid,'EXECUTE'))<>5
    OR has_table_privilege('vec_aspirantes_ejecutor_externo','vec_aspirantes.denegacion_frontera','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_aspirantes' AND roles<>ARRAY['vec_aspirantes_propietario']::name[])
 THEN RAISE EXCEPTION 'Aspirantes 000002: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
