-- Selección: retirada DBA de los grupos técnicos. Solo sin esquema ni
-- consumidores AD3-89/AD3-90 instalados. Las membresías concedidas a un LOGIN
-- desaparecen con el rol.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $prevalidacion$
BEGIN
 IF to_regnamespace('vec_seleccion') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retire primero Selección 000001, AD3-90 y AD3-89';
 END IF;
END $prevalidacion$;
DO $revocar$
DECLARE rol text;
BEGIN
 FOREACH rol IN ARRAY ARRAY['vec_seleccion_ejecutor','vec_seleccion_migrador','vec_seleccion_propietario'] LOOP
  EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM %I', current_database(), rol);
 END LOOP;
END $revocar$;
REVOKE vec_seleccion_propietario FROM vec_seleccion_migrador;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_seleccion_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
DROP ROLE vec_seleccion_ejecutor;
DROP ROLE vec_seleccion_migrador;
DROP ROLE vec_seleccion_propietario;
COMMIT;
