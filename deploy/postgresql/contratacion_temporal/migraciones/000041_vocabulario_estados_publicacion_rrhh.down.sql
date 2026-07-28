-- Reversión segura CT-000041A: nunca descarta estados ya publicados.
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
 WHERE control AND version_esquema = 21
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 5
 FOR UPDATE;
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh
IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal
    .control_vocabulario_estados_publicacion_rrhh_v1
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    v_rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 21
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 5
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal
               .control_vocabulario_estados_publicacion_rrhh_v1
         WHERE control
           AND version_esquema = 1
           AND restriccion_nombre =
               'publicacion_version_rrhh_estado_clave_valido'
           AND restriccion_definicion =
               $def$CHECK ((estado_clave = ANY (ARRAY['pendiente'::text, 'en_curso'::text, 'espera_externa'::text, 'completado'::text, 'incidencia'::text, 'cancelado'::text])))$def$
           AND restriccion_validada
    ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revertir estados RRHH';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
           AND restriccion.conname =
               'publicacion_version_rrhh_estado_clave_valido'
           AND restriccion.contype = 'c'
           AND restriccion.convalidated
           AND restriccion.conenforced
           AND restriccion.conislocal
           AND NOT restriccion.connoinherit
           AND NOT restriccion.condeferrable
           AND NOT restriccion.condeferred
           AND pg_catalog.pg_get_constraintdef(
                   restriccion.oid, false
               ) = $def$CHECK ((estado_clave = ANY (ARRAY['pendiente'::text, 'en_curso'::text, 'espera_externa'::text, 'completado'::text, 'incidencia'::text, 'cancelado'::text])))$def$
    ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'deriva del catálogo de estados RRHH';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class tabla
          JOIN pg_catalog.pg_roles propietario
            ON propietario.oid = tabla.relowner
         WHERE tabla.oid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
           AND tabla.relkind = 'r'
           AND tabla.relpersistence = 'p'
           AND NOT tabla.relispartition
           AND tabla.relrowsecurity
           AND tabla.relforcerowsecurity
           AND propietario.rolname =
               'vec_contratacion_temporal_propietario'
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
    ) <> 13 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_index indice
         WHERE indice.indrelid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
           AND indice.indisvalid
           AND indice.indisready
           AND indice.indislive
    ) <> 1 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
           AND NOT disparador.tgisinternal
           AND disparador.tgenabled = 'O'
           AND disparador.tgname = ANY(ARRAY[
               'control_vocabulario_estados_rrhh_inmutable',
               'control_vocabulario_estados_rrhh_no_truncar'
           ]::name[])
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_policy politica
         WHERE politica.polrelid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
           AND politica.polname = 'propietario_total'
           AND politica.polcmd = '*'
           AND politica.polpermissive
           AND politica.polroles = ARRAY[
               'vec_contratacion_temporal_propietario'::regrole::oid
           ]::oid[]
           AND pg_catalog.pg_get_expr(
                   politica.polqual, politica.polrelid, false
               ) = 'true'
           AND pg_catalog.pg_get_expr(
                   politica.polwithcheck, politica.polrelid, false
               ) = 'true'
    ) <> 1 OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_inherits herencia
         WHERE herencia.inhrelid =
                   'vec_contratacion_temporal.'
                   'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
            OR herencia.inhparent =
                   'vec_contratacion_temporal.'
                   'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_publication_rel pertenencia
         WHERE pertenencia.prrelid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_statistic_ext estadistica
         WHERE estadistica.stxrelid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_seclabel etiqueta
         WHERE etiqueta.classoid = 'pg_catalog.pg_class'::regclass
           AND etiqueta.objoid =
               'vec_contratacion_temporal.'
               'control_vocabulario_estados_publicacion_rrhh_v1'::regclass
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'catálogo derivado impide revertir estados RRHH';
    END IF;

    FOREACH v_rol IN ARRAY ARRAY[
        'public',
        'vec_contratacion_temporal_migrador',
        'vec_contratacion_temporal_ejecutor',
        'vec_contratacion_temporal_gobernador',
        'vec_contratacion_temporal_confirmador_cobertura',
        'vec_contratacion_temporal_lector_resultado_cobertura',
        'vec_contratacion_temporal_consultor_rrhh'
    ]::text[] LOOP
        IF pg_catalog.has_table_privilege(
            v_rol,
            'vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1',
            'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) OR pg_catalog.has_table_privilege(
            v_rol,
            'vec_contratacion_temporal.publicacion_version_rrhh',
            'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'ACL derivada impide revertir estados RRHH';
        END IF;
    END LOOP;

    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE estado_clave IN (
             'pendiente', 'espera_externa', 'incidencia'
         )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'estados nuevos impiden revertir estados RRHH';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
DROP CONSTRAINT publicacion_version_rrhh_estado_clave_valido;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
ADD CONSTRAINT publicacion_version_rrhh_estado_clave_check
CHECK (estado_clave IN (
    'en_curso',
    'completado',
    'cancelado'
)) NOT VALID;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
VALIDATE CONSTRAINT publicacion_version_rrhh_estado_clave_check;

DROP TRIGGER control_vocabulario_estados_rrhh_no_truncar
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1;
DROP TRIGGER control_vocabulario_estados_rrhh_inmutable
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1;
DROP POLICY propietario_total
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1;
DROP TABLE vec_contratacion_temporal
    .control_vocabulario_estados_publicacion_rrhh_v1
RESTRICT;

DO $manifiesto_retirado$
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint restriccion
         WHERE restriccion.conrelid =
               'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
           AND restriccion.conname =
               'publicacion_version_rrhh_estado_clave_check'
           AND restriccion.contype = 'c'
           AND restriccion.convalidated
           AND restriccion.conenforced
           AND restriccion.conislocal
           AND NOT restriccion.connoinherit
           AND NOT restriccion.condeferrable
           AND NOT restriccion.condeferred
           AND pg_catalog.pg_get_constraintdef(
                   restriccion.oid, false
               ) = $def$CHECK ((estado_clave = ANY (ARRAY['en_curso'::text, 'completado'::text, 'cancelado'::text])))$def$
    ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'restauración de estados RRHH incompleta';
    END IF;
END
$manifiesto_retirado$;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 4,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 5;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 20,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 21;
COMMIT;
