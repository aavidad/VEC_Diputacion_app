BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_registro_accesos:migracion:v1', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_registro_accesos') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down T13 rechazado: esquema ausente';
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
    ) OR EXISTS (
        SELECT 1 FROM vec_bolsa_registro_accesos.politica_retencion
         WHERE version <> 1
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down T13 rechazado: hay historia durable';
    END IF;
END
$prevalidacion$;

DROP FUNCTION
    vec_bolsa_registro_accesos.publicar_politica_retencion_v1(jsonb);
DROP FUNCTION
    vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(jsonb);
DROP FUNCTION vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb);
DROP FUNCTION vec_bolsa_registro_accesos.registrar_interno_v1(jsonb);
DROP FUNCTION vec_bolsa_registro_accesos.auditoria_json_v1(bigint);
DROP TABLE vec_bolsa_registro_accesos.consumo_efecto_consulta;
DROP TABLE vec_bolsa_registro_accesos.registro_acceso;
DROP TABLE vec_bolsa_registro_accesos.control_cadena;
DROP TABLE vec_bolsa_registro_accesos.politica_actual;
DROP TABLE vec_bolsa_registro_accesos.politica_retencion;
DROP FUNCTION vec_bolsa_registro_accesos.impedir_mutacion_v1();
DROP FUNCTION vec_bolsa_registro_accesos.huella_valor_filtro_v1(text);
DROP FUNCTION
    vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(jsonb,jsonb);
DROP FUNCTION
    vec_bolsa_registro_accesos.objeto_claves_exactas_v1(jsonb,text[]);
DROP SCHEMA vec_bolsa_registro_accesos;
COMMIT;
