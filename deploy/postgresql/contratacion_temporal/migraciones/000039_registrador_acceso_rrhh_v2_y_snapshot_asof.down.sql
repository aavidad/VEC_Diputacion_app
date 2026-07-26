-- Reversión segura de C2-D2-C. Conserva toda historia anterior al baseline.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 19
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 3
 FOR UPDATE;

LOCK TABLE
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2,
    vec_contratacion_temporal.control_registrador_acceso_rrhh_v2,
    vec_contratacion_temporal.registro_acceso_rrhh,
    vec_contratacion_temporal.control_cadena_accesos_rrhh,
    vec_contratacion_temporal.publicacion_version_rrhh
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    funcion oid := pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.'
        || 'registrar_acceso_rrhh_interno_v2(jsonb)'
    );
    indice oid := pg_catalog.to_regclass(
        'vec_contratacion_temporal.'
        || 'publicacion_rrhh_organizacion_expediente_corte_desc_idx'
    );
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 19
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 3
    ) OR funcion IS NULL OR indice IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_registrador_acceso_rrhh_v2'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2'
    ) IS NULL
    OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc p
          JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
         WHERE p.oid = funcion
           AND r.rolname = 'vec_contratacion_temporal_propietario'
           AND p.prosecdef
           AND p.provolatile = 'v'
           AND p.proparallel = 'u'
           AND p.proconfig @> ARRAY[
               'search_path=pg_catalog',
               'row_security=on', 'TimeZone=UTC',
               'lock_timeout=1s', 'statement_timeout=4s'
           ]::text[]
    )
    OR pg_catalog.has_function_privilege(
        'public', funcion, 'EXECUTE'
    )
    OR pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_consultor_rrhh',
        funcion, 'EXECUTE'
    )
    OR EXISTS (
        SELECT 1
          FROM pg_catalog.aclexplode((
              SELECT proacl FROM pg_catalog.pg_proc WHERE oid = funcion
          )) a
         WHERE a.privilege_type = 'EXECUTE'
           AND a.grantee NOT IN (
               SELECT oid FROM pg_catalog.pg_roles
                WHERE rolname =
                      'vec_contratacion_temporal_propietario'
           )
    )
    OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_index i
         WHERE i.indexrelid = indice
           AND i.indrelid =
               'vec_contratacion_temporal.'
               'publicacion_version_rrhh'::regclass
           AND i.indisvalid AND i.indisready
           AND pg_catalog.pg_get_indexdef(i.indexrelid) LIKE
               '%USING btree (organizacion_ref COLLATE "C", '
               || 'expediente_ref COLLATE "C", corte_global DESC)%'
    )
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class c
          JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_contratacion_temporal'
           AND c.relname IN (
               'control_registrador_acceso_rrhh_v2',
               'vinculo_identidad_acceso_rrhh_v2'
           )
           AND c.relowner =
               'vec_contratacion_temporal_propietario'::regrole
           AND c.relrowsecurity AND c.relforcerowsecurity
    ) <> 2
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_policies
         WHERE schemaname = 'vec_contratacion_temporal'
           AND tablename IN (
               'control_registrador_acceso_rrhh_v2',
               'vinculo_identidad_acceso_rrhh_v2'
           )
           AND policyname = 'propietario_total'
           AND roles =
               ARRAY['vec_contratacion_temporal_propietario']::name[]
    ) <> 2
    OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger t
         WHERE t.tgrelid IN (
             'vec_contratacion_temporal.'
             'control_registrador_acceso_rrhh_v2'::regclass,
             'vec_contratacion_temporal.'
             'vinculo_identidad_acceso_rrhh_v2'::regclass
         )
           AND NOT t.tgisinternal
           AND t.tgname IN (
               'control_registrador_acceso_rrhh_v2_inmutable',
               'control_registrador_acceso_rrhh_v2_no_truncar',
               'vinculo_identidad_acceso_rrhh_v2_inmutable',
               'vinculo_identidad_acceso_rrhh_v2_no_truncar'
           )
    ) <> 4 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down del registrador RRHH v2 rechazado por deriva';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .vinculo_identidad_acceso_rrhh_v2
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_registrador_acceso_rrhh_v2 base
          JOIN vec_contratacion_temporal
               .control_cadena_accesos_rrhh cadena
            ON cadena.control = base.control
         WHERE base.control
           AND (
               cadena.ultima_secuencia <> base.secuencia_base
               OR cadena.cabeza_sha256 <> base.cabeza_base_sha256
           )
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.registro_acceso_rrhh acceso
          CROSS JOIN vec_contratacion_temporal
               .control_registrador_acceso_rrhh_v2 base
         WHERE base.control
           AND acceso.secuencia > base.secuencia_base
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia v2 impide retirar el registrador RRHH';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;
DROP FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)
    RESTRICT;
DROP TABLE
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
    RESTRICT;
DROP TABLE
    vec_contratacion_temporal.control_registrador_acceso_rrhh_v2
    RESTRICT;
DROP INDEX
    vec_contratacion_temporal.
    publicacion_rrhh_organizacion_expediente_corte_desc_idx
    RESTRICT;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 2,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 3;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 18,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 19;

COMMIT;
