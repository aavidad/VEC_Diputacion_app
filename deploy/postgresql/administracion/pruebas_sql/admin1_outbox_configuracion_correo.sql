\set ON_ERROR_STOP on
-- Contrato estructural candidato de ADMIN1. Ejecutar únicamente en una base
-- desechable donde ADMIN1 ya esté instalado; no inserta configuración, no
-- usa secreto ni acredita rollback funcional.
BEGIN;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    v_columnas text[];
    v_funcion text;
BEGIN
    IF to_regclass('vec_administracion.outbox_configuracion_correo') IS NULL
       OR to_regprocedure('vec_administracion.rechazar_mutacion_outbox_configuracion_correo_v1()') IS NULL
       OR to_regprocedure('vec_administracion.guardar_configuracion_correo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'ADMIN1: outbox o contrato de escritura ausente';
    END IF;

    SELECT array_agg(attname ORDER BY attnum)
      INTO v_columnas
      FROM pg_attribute
     WHERE attrelid = 'vec_administracion.outbox_configuracion_correo'::regclass
       AND attnum > 0 AND NOT attisdropped;
    IF v_columnas IS DISTINCT FROM ARRAY[
        'evento_ref', 'operacion_ref', 'version_configuracion',
        'tipo_evento', 'carga_json', 'creada_en'
    ] THEN
        RAISE EXCEPTION 'ADMIN1: columnas de outbox divergentes: %', v_columnas;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
         WHERE conrelid = 'vec_administracion.outbox_configuracion_correo'::regclass
           AND contype = 'f'
           AND confrelid = 'vec_administracion.auditoria_configuracion_correo'::regclass
           AND conkey = ARRAY[2,3]::smallint[]
           AND confkey = ARRAY[3,2]::smallint[]
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_constraint
         WHERE conrelid = 'vec_administracion.outbox_configuracion_correo'::regclass
           AND contype = 'u' AND conkey = ARRAY[2]::smallint[]
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_constraint
         WHERE conrelid = 'vec_administracion.outbox_configuracion_correo'::regclass
           AND contype = 'c' AND pg_get_constraintdef(oid) LIKE '%administracion.configuracion_correo.actualizada%'
    ) THEN
        RAISE EXCEPTION 'ADMIN1: enlace operación/version o tipo de evento divergente';
    END IF;

    IF NOT (SELECT relrowsecurity AND relforcerowsecurity
              FROM pg_class
             WHERE oid = 'vec_administracion.outbox_configuracion_correo'::regclass)
       OR NOT EXISTS (
            SELECT 1 FROM pg_policy
             WHERE polrelid = 'vec_administracion.outbox_configuracion_correo'::regclass
               AND polname = 'propietario_outbox_configuracion_correo'
       ) OR has_table_privilege('vec_administracion_ejecutor',
              'vec_administracion.outbox_configuracion_correo', 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE') THEN
        RAISE EXCEPTION 'ADMIN1: ACL o RLS de outbox divergentes';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
         WHERE tgrelid = 'vec_administracion.outbox_configuracion_correo'::regclass
           AND tgname = 'outbox_configuracion_correo_inmutable'
           AND NOT tgisinternal
           AND tgfoid = 'vec_administracion.rechazar_mutacion_outbox_configuracion_correo_v1()'::regprocedure
    ) THEN
        RAISE EXCEPTION 'ADMIN1: outbox mutable';
    END IF;

    SELECT pg_get_functiondef(
        'vec_administracion.guardar_configuracion_correo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
    ) INTO v_funcion;
    IF position('v_consumo.consumo_nuevo IS NOT TRUE' IN v_funcion) = 0
       OR position('INSERT INTO vec_administracion.auditoria_configuracion_correo' IN v_funcion) = 0
       OR position('INSERT INTO vec_administracion.outbox_configuracion_correo' IN v_funcion) = 0
       OR position('INSERT INTO vec_administracion.auditoria_configuracion_correo' IN v_funcion)
          > position('INSERT INTO vec_administracion.outbox_configuracion_correo' IN v_funcion)
       OR position('v_consumo.auditoria_ref' IN v_funcion) = 0
       OR position('material_sha256' IN v_funcion) = 0 THEN
        RAISE EXCEPTION 'ADMIN1: replay, enlace o orden atómico divergente';
    END IF;
END
$prueba$;

ROLLBACK;
