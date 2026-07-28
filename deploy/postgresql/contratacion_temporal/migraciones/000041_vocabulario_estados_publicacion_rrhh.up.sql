-- O4-05/CT-000041A: vocabulario técnico completo de la publicación RRHH.
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
 WHERE control AND version_esquema = 20
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 4
 FOR UPDATE;
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh
IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    v_rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 20
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 4
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.publicacion_version_rrhh'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para ampliar estados RRHH';
    END IF;

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
    ) <> 1 OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.publicacion_version_rrhh
         WHERE estado_clave NOT IN (
             'en_curso', 'completado', 'cancelado'
         )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'catálogo heredado de estados RRHH incompatible';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class tabla
          JOIN pg_catalog.pg_roles propietario
            ON propietario.oid = tabla.relowner
         WHERE tabla.oid =
               'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
           AND tabla.relkind = 'r'
           AND tabla.relpersistence = 'p'
           AND NOT tabla.relispartition
           AND tabla.relrowsecurity
           AND tabla.relforcerowsecurity
           AND propietario.rolname =
               'vec_contratacion_temporal_propietario'
    ) OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid =
               'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
           AND NOT disparador.tgisinternal
           AND disparador.tgenabled = 'O'
           AND disparador.tgname = ANY(ARRAY[
               'publicacion_version_rrhh_inmutable',
               'publicacion_version_rrhh_no_truncar'
           ]::name[])
    ) <> 2 OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger disparador
         WHERE disparador.tgrelid =
               'vec_contratacion_temporal.expediente_version_integral'::regclass
           AND NOT disparador.tgisinternal
           AND disparador.tgenabled = 'O'
           AND disparador.tgname =
               'expediente_version_integral_publicar_rrhh'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia o catálogo RRHH incompatible';
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
            'vec_contratacion_temporal.publicacion_version_rrhh',
            'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'ACL incompatible para ampliar estados RRHH';
        END IF;
    END LOOP;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
DROP CONSTRAINT publicacion_version_rrhh_estado_clave_check;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
ADD CONSTRAINT publicacion_version_rrhh_estado_clave_valido
CHECK (estado_clave IN (
    'pendiente',
    'en_curso',
    'espera_externa',
    'completado',
    'incidencia',
    'cancelado'
)) NOT VALID;
ALTER TABLE vec_contratacion_temporal.publicacion_version_rrhh
VALIDATE CONSTRAINT publicacion_version_rrhh_estado_clave_valido;

CREATE TABLE
vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1 (
    control boolean DEFAULT true NOT NULL,
    version_esquema integer NOT NULL,
    restriccion_nombre text NOT NULL,
    restriccion_definicion text NOT NULL,
    restriccion_validada boolean NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    CONSTRAINT control_vocabulario_estados_rrhh_pk PRIMARY KEY (control),
    CONSTRAINT control_vocabulario_estados_rrhh_control_check
        CHECK (control),
    CONSTRAINT control_vocabulario_estados_rrhh_version_check
        CHECK (version_esquema = 1),
    CONSTRAINT control_vocabulario_estados_rrhh_nombre_check
        CHECK (
            restriccion_nombre =
                'publicacion_version_rrhh_estado_clave_valido'
        ),
    CONSTRAINT control_vocabulario_estados_rrhh_definicion_check
        CHECK (
            restriccion_definicion =
            $def$CHECK ((estado_clave = ANY (ARRAY['pendiente'::text, 'en_curso'::text, 'espera_externa'::text, 'completado'::text, 'incidencia'::text, 'cancelado'::text])))$def$
        ),
    CONSTRAINT control_vocabulario_estados_rrhh_validada_check
        CHECK (restriccion_validada),
    CONSTRAINT control_vocabulario_estados_rrhh_creada_check
        CHECK (
            creada_en =
                pg_catalog.date_trunc('microseconds', creada_en)
        )
);

INSERT INTO
vec_contratacion_temporal.control_vocabulario_estados_publicacion_rrhh_v1 (
    control,
    version_esquema,
    restriccion_nombre,
    restriccion_definicion,
    restriccion_validada,
    creada_en
)
SELECT true,
       1,
       restriccion.conname,
       pg_catalog.pg_get_constraintdef(restriccion.oid, false),
       restriccion.convalidated,
       pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
  FROM pg_catalog.pg_constraint restriccion
 WHERE restriccion.conrelid =
       'vec_contratacion_temporal.publicacion_version_rrhh'::regclass
   AND restriccion.conname =
       'publicacion_version_rrhh_estado_clave_valido';

CREATE TRIGGER control_vocabulario_estados_rrhh_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1
FOR EACH ROW
EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER control_vocabulario_estados_rrhh_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1
FOR EACH STATEMENT
EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

ALTER TABLE vec_contratacion_temporal
    .control_vocabulario_estados_publicacion_rrhh_v1
ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal
    .control_vocabulario_estados_publicacion_rrhh_v1
FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
ON vec_contratacion_temporal
   .control_vocabulario_estados_publicacion_rrhh_v1
TO vec_contratacion_temporal_propietario
USING (true)
WITH CHECK (true);

REVOKE ALL ON TABLE
    vec_contratacion_temporal
        .control_vocabulario_estados_publicacion_rrhh_v1
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

DO $manifiesto$
DECLARE
    v_rol text;
BEGIN
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
    ) <> 1 OR (
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
            MESSAGE = 'manifiesto de estados RRHH incompleto';
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
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'ACL excesiva en manifiesto de estados RRHH';
        END IF;
    END LOOP;
END
$manifiesto$;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 5,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 4;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 21,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 20;
COMMIT;
