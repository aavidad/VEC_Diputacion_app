-- CA6 solo puede retirarse sin uso ni dependencias de CT109.
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $preimagen$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CA6: retirada con dependencias' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(jsonb,text,text,text,text,text);
DROP FUNCTION vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(text,numeric,text,numeric,text,numeric,text,numeric);
COMMIT;
