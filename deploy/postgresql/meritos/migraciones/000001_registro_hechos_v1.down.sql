\set ON_ERROR_STOP on
-- Sólo para una base desechable sin historia. Nunca sobre instalación con uso.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:migracion:000001',0));
DO $pre$ DECLARE t text; usada boolean; BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regclass('vec_meritos.hecho_identidad') IS NULL
 THEN RAISE EXCEPTION 'meritos.error.down_incompatible' USING ERRCODE='55000'; END IF;
 -- El superusuario comprueba la historia completa sin el filtro nominal RLS.
 FOREACH t IN ARRAY ARRAY['hecho_identidad','hecho_version','operacion','auditoria_operacion','outbox','acceso_operacion'] LOOP
 EXECUTE format('LOCK TABLE vec_meritos.%I IN ACCESS EXCLUSIVE MODE',t);
 EXECUTE format('SELECT EXISTS (SELECT 1 FROM vec_meritos.%I)',t) INTO usada;
 IF usada THEN RAISE EXCEPTION 'meritos.error.down_con_historia' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_meritos_propietario;
DROP FUNCTION vec_meritos.operar_hecho_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_meritos.registrar_auditoria_v1(jsonb,jsonb,text,timestamptz,text);
DROP FUNCTION vec_meritos.cotejar_autorizacion_v1(jsonb,bytea,bytea,bytea);
DROP FUNCTION vec_meritos.validar_comando_v1(bytea);
DROP FUNCTION vec_meritos.validar_hecho_comando_v1(jsonb);
DROP TABLE vec_meritos.acceso_operacion,vec_meritos.outbox,vec_meritos.auditoria_operacion;
DROP TABLE vec_meritos.operacion;
DROP TABLE vec_meritos.hecho_version;
DROP TABLE vec_meritos.hecho_identidad;
DROP FUNCTION vec_meritos.historia_inmutable_v1();
DROP FUNCTION vec_meritos.claves_exactas_v1(jsonb,text[],text[]);
DROP FUNCTION vec_meritos.referencia_valida_v1(text);
COMMIT;
