\set ON_ERROR_STOP on
-- Sólo reversible en una instalación vacía; nunca elimina historia conservada.
-- Retirar antes el consumidor nominal AD3 y sus ACL. Sin CASCADE ni DROP OWNED.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas_v1:000001',0));
DO $precondiciones$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_borrador_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'retirada de Dietas no permitida' USING ERRCODE='55000';
    END IF;
    LOCK TABLE vec_dietas_v1.borrador_revision,vec_dietas_v1.borrador_outbox,
        vec_dietas_v1.borrador_acceso IN ACCESS EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM vec_dietas_v1.borrador_revision)
       OR EXISTS (SELECT 1 FROM vec_dietas_v1.borrador_outbox)
       OR EXISTS (SELECT 1 FROM vec_dietas_v1.borrador_acceso) THEN
        RAISE EXCEPTION 'Dietas conserva historia; retirada rechazada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid IN
        ('vec_dietas_v1_propietario'::regrole,'vec_dietas_v1_ejecutor'::regrole)
        OR member IN ('vec_dietas_v1_propietario'::regrole,'vec_dietas_v1_ejecutor'::regrole)) THEN
        RAISE EXCEPTION 'Dietas conserva asignaciones técnicas' USING ERRCODE='55000';
    END IF;
END
$precondiciones$;
DROP FUNCTION vec_dietas_v1.crear_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_dietas_v1.listar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_dietas_v1.recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_dietas_v1.operar_borrador_propio_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_dietas_v1.borrador_acceso;
DROP TABLE vec_dietas_v1.borrador_outbox;
DROP TABLE vec_dietas_v1.borrador_revision;
DROP FUNCTION vec_dietas_v1.rechazar_mutacion_historia_v1();
DROP FUNCTION vec_dietas_v1.validar_borrador_completo_v1(jsonb);
DROP FUNCTION vec_dietas_v1.instante_civil_unico_v1(timestamptz,text,text,text);
DROP FUNCTION vec_dietas_v1.objeto_exacto_v1(jsonb,text[]);
DROP SCHEMA vec_dietas_v1;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_dietas_v1_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
DROP ROLE vec_dietas_v1_ejecutor;
DROP ROLE vec_dietas_v1_propietario;
COMMIT;
