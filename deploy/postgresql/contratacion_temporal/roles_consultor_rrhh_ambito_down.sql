-- Solo retiro de un grupo nunca aprovisionado ni consumido.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $preimagen$
DECLARE r oid:=to_regrole('vec_contratacion_temporal_consultor_rrhh_ambito');
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
   OR r IS NULL
   OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member=r OR roleid=r)
   OR to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'grupo CT ambito en uso' USING ERRCODE='55000'; END IF;
END $preimagen$;
DO $retirar$
BEGIN
 EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_contratacion_temporal_consultor_rrhh_ambito',current_database());
END $retirar$;
DROP ROLE vec_contratacion_temporal_consultor_rrhh_ambito;
COMMIT;
