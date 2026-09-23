\set ON_ERROR_STOP on
-- Sólo el entorno efímero: la fachada AD3 TEST-ONLY se elimina antes de retirar
-- Dietas. No prueba ni sustituye criptografía, firma ni AD3 publicados.
BEGIN;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_autorizacion_atestada_v3.stub_consumo;
DROP SCHEMA vec_autorizacion_atestada_v3;
COMMIT;
