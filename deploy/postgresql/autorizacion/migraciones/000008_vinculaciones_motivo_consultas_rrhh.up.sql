-- Fundamento privado de las vinculaciones nominales de motivo para las
-- consultas RRHH. Esta migracion no publica ni resuelve vinculaciones.
BEGIN;

SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008', 0
    )
);

DO $prevalidacion$
DECLARE
    historia regclass := pg_catalog.to_regclass(
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'
    );
    checkpoint regclass := pg_catalog.to_regclass(
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'
    );
    funcion regprocedure;
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace
         WHERE nspname = 'vec_autorizacion'
           AND nspowner = 'vec_autorizacion_propietario'::regrole
    )
       OR pg_catalog.to_regclass(
           'vec_autorizacion.motivo_v2_catalogo_publicado'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.motivo_v2_entrada'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 requiere la proyeccion de motivos V2';
    END IF;

    IF (historia IS NULL) IS DISTINCT FROM (checkpoint IS NULL) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta una instalacion parcial';
    END IF;
    IF historia IS NOT NULL AND (
        NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class
             WHERE oid = historia
               AND relkind = 'r'
               AND relowner = 'vec_autorizacion_propietario'::regrole
               AND pg_catalog.obj_description(oid, 'pg_class') =
                   'vec_autorizacion:vinculacion-motivo-consulta-rrhh:v1:000008'
        )
        OR NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class
             WHERE oid = checkpoint
               AND relkind = 'r'
               AND relowner = 'vec_autorizacion_propietario'::regrole
               AND pg_catalog.obj_description(oid, 'pg_class') =
                   'vec_autorizacion:vinculacion-motivo-consulta-rrhh:checkpoint-v1:000008'
        )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta tablas preexistentes';
    END IF;

    FOREACH funcion IN ARRAY ARRAY[
        pg_catalog.to_regprocedure(
            'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'
        ),
        pg_catalog.to_regprocedure(
            'vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()'
        )
    ] LOOP
        IF funcion IS NOT NULL AND NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_proc
             WHERE oid = funcion
               AND proowner = 'vec_autorizacion_propietario'::regrole
               AND pg_catalog.obj_description(oid, 'pg_proc') =
                   'vec_autorizacion:vinculacion-motivo-consulta-rrhh:v1:000008'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = '000008 no adopta funciones preexistentes';
        END IF;
    END LOOP;
END
$prevalidacion$;

SET LOCAL ROLE vec_autorizacion_propietario;

DO $referencia_completa$
DECLARE
    existente record;
BEGIN
    SELECT c.oid, pg_catalog.pg_get_constraintdef(c.oid, true) AS definicion
      INTO existente
      FROM pg_catalog.pg_constraint AS c
     WHERE c.conrelid =
           'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass
       AND c.conname = 'motivo_v2_catalogo_referencia_completa_unica';
    IF FOUND THEN
        IF existente.definicion IS DISTINCT FROM
           'UNIQUE (catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)' THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = '000008 no adopta una restriccion homonima';
        END IF;
    ELSE
        ALTER TABLE vec_autorizacion.motivo_v2_catalogo_publicado
            ADD CONSTRAINT motivo_v2_catalogo_referencia_completa_unica
            UNIQUE (
                catalogo_id,
                catalogo_version,
                catalogo_huella_publicada_sha256
            );
    END IF;
END
$referencia_completa$;

CREATE TABLE IF NOT EXISTS
vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (
    clase_consulta text NOT NULL,
    publicacion_version bigint NOT NULL,
    publicacion_ref text NOT NULL UNIQUE,
    publicacion_huella_sha256 text NOT NULL UNIQUE,
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    catalogo_huella_sha256 text NOT NULL,
    entrada_clave text NOT NULL,
    publicada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    registrada_txid bigint NOT NULL DEFAULT pg_catalog.txid_current(),
    PRIMARY KEY (clase_consulta, publicacion_version),
    CONSTRAINT vinculacion_motivo_rrhh_clase_cerrada CHECK (
        clase_consulta IN ('cuadro', 'detalle')
    ),
    CONSTRAINT vinculacion_motivo_rrhh_version_positiva CHECK (
        publicacion_version > 0
    ),
    CONSTRAINT vinculacion_motivo_rrhh_ref_opaca CHECK (
        publicacion_ref ~ '^publicacion_motivo_rrhh_[0-9a-f]{32}$'
    ),
    CONSTRAINT vinculacion_motivo_rrhh_huellas_validas CHECK (
        publicacion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND publicacion_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND catalogo_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND catalogo_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CONSTRAINT vinculacion_motivo_rrhh_entrada_opaca CHECK (
        entrada_clave ~ '^motivo_[0-9a-f]{32}$'
    ),
    CONSTRAINT vinculacion_motivo_rrhh_instantes_validos CHECK (
        pg_catalog.isfinite(publicada_en)
        AND pg_catalog.isfinite(registrada_en)
        AND publicada_en <= registrada_en
        AND extract(
            year FROM (publicada_en AT TIME ZONE 'UTC')
        ) BETWEEN 1 AND 9999
        AND extract(
            year FROM (registrada_en AT TIME ZONE 'UTC')
        ) BETWEEN 1 AND 9999
    ),
    CONSTRAINT vinculacion_motivo_rrhh_publicacion_completa_unica
        UNIQUE (
            clase_consulta,
            publicacion_version,
            publicacion_ref,
            publicacion_huella_sha256
        ),
    CONSTRAINT vinculacion_motivo_rrhh_catalogo_completo_fk
        FOREIGN KEY (
            catalogo_id,
            catalogo_version,
            catalogo_huella_sha256
        ) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado (
            catalogo_id,
            catalogo_version,
            catalogo_huella_publicada_sha256
        ),
    CONSTRAINT vinculacion_motivo_rrhh_entrada_fk
        FOREIGN KEY (catalogo_id, catalogo_version, entrada_clave)
        REFERENCES vec_autorizacion.motivo_v2_entrada (
            catalogo_id,
            catalogo_version,
            entrada_clave
        )
);

CREATE TABLE IF NOT EXISTS
vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 (
    clase_consulta text PRIMARY KEY,
    ultima_publicacion_version bigint NOT NULL,
    ultima_publicacion_ref text,
    ultima_publicacion_huella_sha256 text,
    actualizado_en timestamptz(6) NOT NULL,
    CONSTRAINT vinculacion_motivo_rrhh_checkpoint_clase_cerrada CHECK (
        clase_consulta IN ('cuadro', 'detalle')
    ),
    CONSTRAINT vinculacion_motivo_rrhh_checkpoint_completo CHECK (
        (
            ultima_publicacion_version = 0
            AND ultima_publicacion_ref IS NULL
            AND ultima_publicacion_huella_sha256 IS NULL
        )
        OR (
            ultima_publicacion_version > 0
            AND ultima_publicacion_ref ~
                '^publicacion_motivo_rrhh_[0-9a-f]{32}$'
            AND ultima_publicacion_huella_sha256 ~ '^[0-9a-f]{64}$'
            AND ultima_publicacion_huella_sha256 <>
                pg_catalog.repeat('0', 64)
        )
    ),
    CONSTRAINT vinculacion_motivo_rrhh_checkpoint_instante_valido CHECK (
        pg_catalog.isfinite(actualizado_en)
        AND extract(
            year FROM (actualizado_en AT TIME ZONE 'UTC')
        ) BETWEEN 1 AND 9999
    ),
    CONSTRAINT vinculacion_motivo_rrhh_checkpoint_historia_fk
        FOREIGN KEY (
            clase_consulta,
            ultima_publicacion_version,
            ultima_publicacion_ref,
            ultima_publicacion_huella_sha256
        ) REFERENCES
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (
            clase_consulta,
            publicacion_version,
            publicacion_ref,
            publicacion_huella_sha256
        )
);

COMMENT ON TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 IS
    'vec_autorizacion:vinculacion-motivo-consulta-rrhh:v1:000008';
COMMENT ON TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 IS
    'vec_autorizacion:vinculacion-motivo-consulta-rrhh:checkpoint-v1:000008';

DO $inicializar_checkpoint$
DECLARE
    filas integer;
BEGIN
    SELECT pg_catalog.count(*) INTO filas
      FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1;
    IF filas = 0 THEN
        INSERT INTO
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 (
            clase_consulta,
            ultima_publicacion_version,
            actualizado_en
        ) VALUES
            ('cuadro', 0, pg_catalog.clock_timestamp()),
            ('detalle', 0, pg_catalog.clock_timestamp());
    ELSIF filas <> 2 OR NOT EXISTS (
        SELECT 1
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
         WHERE clase_consulta = 'cuadro'
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
         WHERE clase_consulta = 'detalle'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no repara un checkpoint alterado';
    END IF;
END
$inicializar_checkpoint$;

CREATE OR REPLACE FUNCTION
vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'la historia de vinculaciones de motivo RRHH es inmutable';
END
$funcion$;

COMMENT ON FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1() IS
    'vec_autorizacion:vinculacion-motivo-consulta-rrhh:v1:000008';

CREATE OR REPLACE FUNCTION
vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF NEW.clase_consulta IS DISTINCT FROM OLD.clase_consulta
       OR NEW.ultima_publicacion_version IS DISTINCT FROM
          OLD.ultima_publicacion_version + 1
       OR NEW.ultima_publicacion_ref IS NULL
       OR NEW.ultima_publicacion_huella_sha256 IS NULL
       OR NEW.actualizado_en < OLD.actualizado_en
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 AS h
            WHERE h.clase_consulta = NEW.clase_consulta
              AND h.publicacion_version = NEW.ultima_publicacion_version
              AND h.publicacion_ref = NEW.ultima_publicacion_ref
              AND h.publicacion_huella_sha256 =
                  NEW.ultima_publicacion_huella_sha256
              AND h.registrada_txid = pg_catalog.txid_current()
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance no atomico o no monotono de vinculacion RRHH';
    END IF;
    RETURN NEW;
END
$funcion$;

COMMENT ON FUNCTION
    vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1() IS
    'vec_autorizacion:vinculacion-motivo-consulta-rrhh:v1:000008';

DO $protecciones$
DECLARE
    historia regclass :=
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass;
    checkpoint regclass :=
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_trigger
         WHERE tgrelid = historia
           AND tgname = 'vinculacion_motivo_rrhh_inmutable'
           AND NOT tgisinternal
    ) THEN
        CREATE TRIGGER vinculacion_motivo_rrhh_inmutable
            BEFORE UPDATE OR DELETE ON
                vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
            FOR EACH ROW EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_trigger
         WHERE tgrelid = historia
           AND tgname = 'vinculacion_motivo_rrhh_no_truncar'
           AND NOT tgisinternal
    ) THEN
        CREATE TRIGGER vinculacion_motivo_rrhh_no_truncar
            BEFORE TRUNCATE ON
                vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
            FOR EACH STATEMENT EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_trigger
         WHERE tgrelid = checkpoint
           AND tgname = 'vinculacion_motivo_rrhh_checkpoint_avance'
           AND NOT tgisinternal
    ) THEN
        CREATE TRIGGER vinculacion_motivo_rrhh_checkpoint_avance
            BEFORE UPDATE ON
                vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            FOR EACH ROW EXECUTE FUNCTION
                vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1();
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_trigger
         WHERE tgrelid = checkpoint
           AND tgname = 'vinculacion_motivo_rrhh_checkpoint_inmutable'
           AND NOT tgisinternal
    ) THEN
        CREATE TRIGGER vinculacion_motivo_rrhh_checkpoint_inmutable
            BEFORE INSERT OR DELETE ON
                vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            FOR EACH ROW EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_trigger
         WHERE tgrelid = checkpoint
           AND tgname = 'vinculacion_motivo_rrhh_checkpoint_no_truncar'
           AND NOT tgisinternal
    ) THEN
        CREATE TRIGGER vinculacion_motivo_rrhh_checkpoint_no_truncar
            BEFORE TRUNCATE ON
                vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            FOR EACH STATEMENT EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
    END IF;
END
$protecciones$;

DO $rls$
DECLARE
    tabla regclass;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
    ] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE %s ENABLE ROW LEVEL SECURITY', tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE %s FORCE ROW LEVEL SECURITY', tabla
        );
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_policy
             WHERE polrelid = tabla
               AND polname = 'acceso_propietario_exacto'
        ) THEN
            EXECUTE pg_catalog.format(
                'CREATE POLICY acceso_propietario_exacto ON %s FOR ALL TO vec_autorizacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
                tabla,
                'vec_autorizacion_propietario',
                'vec_autorizacion_propietario'
            );
        END IF;
    END LOOP;
END
$rls$;

REVOKE ALL ON TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
    FROM PUBLIC,
         vec_autorizacion_fuente,
         vec_autorizacion_registro,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;
REVOKE ALL ON FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1(),
    vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()
    FROM PUBLIC,
         vec_autorizacion_fuente,
         vec_autorizacion_registro,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;

DO $cerrar_tipos$
DECLARE
    tipo text;
BEGIN
    FOR tipo IN
        SELECT pg_catalog.format('%I.%I', n.nspname, t.typname)
          FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_type AS t ON t.oid = c.reltype
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE c.oid IN (
            'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
            'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         )
    LOOP
        EXECUTE pg_catalog.format(
            'REVOKE ALL PRIVILEGES ON TYPE %s FROM PUBLIC, vec_autorizacion_fuente, vec_autorizacion_registro, vec_autorizacion_motivos_proyector, vec_autorizacion_motivos_evaluador',
            tipo
        );
    END LOOP;
END
$cerrar_tipos$;

COMMIT;
