-- Fundamento privado de las vinculaciones nominales de motivo para las
-- consultas RRHH. Esta migracion no publica ni resuelve vinculaciones.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL default_table_access_method = heap;
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
    IF historia IS NOT NULL AND (
        EXISTS (
            SELECT 1 FROM pg_catalog.pg_class AS c
             WHERE c.oid IN (historia, checkpoint)
               AND (c.relkind IS DISTINCT FROM 'r'
                   OR c.relpersistence IS DISTINCT FROM 'p'
                   OR c.relam IS DISTINCT FROM (
                       SELECT a.oid FROM pg_catalog.pg_am AS a
                        WHERE a.amname = 'heap' AND a.amtype = 't'
                   ) OR c.relispartition)
        ) OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_inherits AS i
             WHERE i.inhrelid IN (historia, checkpoint)
                OR i.inhparent IN (historia, checkpoint)
        ) OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_rewrite AS r
             WHERE r.ev_class IN (historia, checkpoint)
        )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 exige tablas permanentes heap sin herencia, particion ni reglas';
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

    PERFORM pg_catalog.set_config(
        'vec_autorizacion.migracion_000008_reentrada',
        (historia IS NOT NULL)::text,
        true
    );
END
$prevalidacion$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $referencia_completa$
DECLARE
    existente record;
BEGIN
    SELECT
        c.oid, c.contype, c.convalidated, c.condeferrable,
        c.condeferred, c.connoinherit,
        pg_catalog.pg_get_constraintdef(c.oid, true) AS definicion
      INTO existente
      FROM pg_catalog.pg_constraint AS c
     WHERE c.conrelid =
           'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass
       AND c.conname = 'motivo_v2_catalogo_referencia_completa_unica';
    IF FOUND THEN
        IF existente.contype IS DISTINCT FROM 'u'
           OR NOT existente.convalidated
           OR existente.condeferrable
           OR existente.condeferred
           OR NOT existente.connoinherit
           OR existente.definicion IS DISTINCT FROM
           'UNIQUE (catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)'
           OR pg_catalog.obj_description(
               existente.oid, 'pg_constraint'
           ) IS DISTINCT FROM
              'vec_autorizacion:vinculacion-motivo-consulta-rrhh:referencia-completa:v1:000008' THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = '000008 no adopta una restriccion homonima o sin procedencia';
        END IF;
    ELSE
        ALTER TABLE vec_autorizacion.motivo_v2_catalogo_publicado
            ADD CONSTRAINT motivo_v2_catalogo_referencia_completa_unica
            UNIQUE (
                catalogo_id,
                catalogo_version,
                catalogo_huella_publicada_sha256
            );
        COMMENT ON CONSTRAINT
            motivo_v2_catalogo_referencia_completa_unica
            ON vec_autorizacion.motivo_v2_catalogo_publicado IS
            'vec_autorizacion:vinculacion-motivo-consulta-rrhh:referencia-completa:v1:000008';
    END IF;
END
$referencia_completa$;
CREATE TABLE IF NOT EXISTS
vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (
    clase_consulta text NOT NULL,
    publicacion_version bigint NOT NULL,
    publicacion_ref text NOT NULL,
    publicacion_huella_sha256 text NOT NULL,
    catalogo_id text NOT NULL,
    catalogo_version integer NOT NULL,
    catalogo_huella_sha256 text NOT NULL,
    entrada_clave text NOT NULL,
    publicada_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    registrada_txid bigint NOT NULL DEFAULT pg_catalog.txid_current(),
    CONSTRAINT vinculacion_motivo_rrhh_pk
        PRIMARY KEY (clase_consulta, publicacion_version),
    CONSTRAINT vinculacion_motivo_rrhh_publicacion_ref_unica
        UNIQUE (publicacion_ref),
    CONSTRAINT vinculacion_motivo_rrhh_publicacion_huella_unica
        UNIQUE (publicacion_huella_sha256),
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
    clase_consulta text NOT NULL,
    ultima_publicacion_version bigint NOT NULL,
    ultima_publicacion_ref text,
    ultima_publicacion_huella_sha256 text,
    actualizado_en timestamptz(6) NOT NULL,
    CONSTRAINT vinculacion_motivo_rrhh_checkpoint_pk
        PRIMARY KEY (clase_consulta),
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

DO $estructura_exacta$
BEGIN
    IF EXISTS (
        WITH esperado(
            tabla, posicion, columna, tipo, no_nula, defecto,
            identidad, generada, ausente, eliminada
        ) AS (
            VALUES
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 1, 'clase_consulta', 'text', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 2, 'publicacion_version', 'bigint', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 3, 'publicacion_ref', 'text', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 4, 'publicacion_huella_sha256', 'text', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 5, 'catalogo_id', 'text', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 6, 'catalogo_version', 'integer', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 7, 'catalogo_huella_sha256', 'text', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 8, 'entrada_clave', 'text', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 9, 'publicada_en', 'timestamp(6) with time zone', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 10, 'registrada_en', 'timestamp(6) with time zone', true, 'clock_timestamp()', '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 11, 'registrada_txid', 'bigint', true, 'txid_current()', '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 1, 'clase_consulta', 'text', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 2, 'ultima_publicacion_version', 'bigint', true, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 3, 'ultima_publicacion_ref', 'text', false, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 4, 'ultima_publicacion_huella_sha256', 'text', false, NULL::text, '', '', false, false),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 5, 'actualizado_en', 'timestamp(6) with time zone', true, NULL::text, '', '', false, false)
        ),
        actual AS (
            SELECT
                a.attrelid, a.attnum::integer, a.attname::text,
                pg_catalog.format_type(a.atttypid, a.atttypmod),
                a.attnotnull,
                pg_catalog.pg_get_expr(d.adbin, d.adrelid, true),
                a.attidentity::text, a.attgenerated::text,
                a.atthasmissing, a.attisdropped
              FROM pg_catalog.pg_attribute AS a
              LEFT JOIN pg_catalog.pg_attrdef AS d
                ON d.adrelid = a.attrelid AND d.adnum = a.attnum
             WHERE a.attrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             )
               AND a.attnum > 0
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_attribute AS a
          JOIN pg_catalog.pg_type AS t ON t.oid = a.atttypid
         WHERE a.attrelid IN (
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         ) AND a.attnum > 0 AND a.attcollation IS DISTINCT FROM t.typcollation
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_class AS c CROSS JOIN LATERAL
          pg_catalog.aclexplode(coalesce(
              c.relacl, pg_catalog.acldefault('r', c.relowner)
          )) AS acl
         WHERE c.oid IN (
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         ) AND acl.grantee <> c.relowner
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_attribute AS a
          JOIN pg_catalog.pg_class AS c ON c.oid = a.attrelid
          CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl) AS acl
         WHERE a.attrelid IN (
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         ) AND a.attnum > 0 AND acl.grantee <> c.relowner
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta columnas, colaciones o ACL alteradas';
    END IF;

    IF EXISTS (
        WITH esperado(tabla, nombre, tipo, definicion) AS (
            VALUES
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_pk', 'p', 'PRIMARY KEY (clase_consulta, publicacion_version)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_publicacion_ref_unica', 'u', 'UNIQUE (publicacion_ref)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_publicacion_huella_unica', 'u', 'UNIQUE (publicacion_huella_sha256)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_clase_cerrada', 'c', 'CHECK (clase_consulta = ANY (ARRAY[''cuadro''::text, ''detalle''::text]))'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_version_positiva', 'c', 'CHECK (publicacion_version > 0)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_ref_opaca', 'c', 'CHECK (publicacion_ref ~ ''^publicacion_motivo_rrhh_[0-9a-f]{32}$''::text)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_huellas_validas', 'c', 'CHECK (publicacion_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND publicacion_huella_sha256 <> repeat(''0''::text, 64) AND catalogo_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND catalogo_huella_sha256 <> repeat(''0''::text, 64))'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_entrada_opaca', 'c', 'CHECK (entrada_clave ~ ''^motivo_[0-9a-f]{32}$''::text)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_instantes_validos', 'c', 'CHECK (isfinite(publicada_en) AND isfinite(registrada_en) AND publicada_en <= registrada_en AND EXTRACT(year FROM (publicada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (publicada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (registrada_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_publicacion_completa_unica', 'u', 'UNIQUE (clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_catalogo_completo_fk', 'f', 'FOREIGN KEY (catalogo_id, catalogo_version, catalogo_huella_sha256) REFERENCES vec_autorizacion.motivo_v2_catalogo_publicado(catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_entrada_fk', 'f', 'FOREIGN KEY (catalogo_id, catalogo_version, entrada_clave) REFERENCES vec_autorizacion.motivo_v2_entrada(catalogo_id, catalogo_version, entrada_clave)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_pk', 'p', 'PRIMARY KEY (clase_consulta)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_clase_cerrada', 'c', 'CHECK (clase_consulta = ANY (ARRAY[''cuadro''::text, ''detalle''::text]))'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_completo', 'c', 'CHECK (ultima_publicacion_version = 0 AND ultima_publicacion_ref IS NULL AND ultima_publicacion_huella_sha256 IS NULL OR ultima_publicacion_version > 0 AND ultima_publicacion_ref ~ ''^publicacion_motivo_rrhh_[0-9a-f]{32}$''::text AND ultima_publicacion_huella_sha256 ~ ''^[0-9a-f]{64}$''::text AND ultima_publicacion_huella_sha256 <> repeat(''0''::text, 64))'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_instante_valido', 'c', 'CHECK (isfinite(actualizado_en) AND EXTRACT(year FROM (actualizado_en AT TIME ZONE ''UTC''::text)) >= 1::numeric AND EXTRACT(year FROM (actualizado_en AT TIME ZONE ''UTC''::text)) <= 9999::numeric)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_historia_fk', 'f', 'FOREIGN KEY (clase_consulta, ultima_publicacion_version, ultima_publicacion_ref, ultima_publicacion_huella_sha256) REFERENCES vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1(clase_consulta, publicacion_version, publicacion_ref, publicacion_huella_sha256)')
        ),
        actual AS (
            SELECT
                c.conrelid, c.conname::text, c.contype::text,
                pg_catalog.pg_get_constraintdef(c.oid, true)
              FROM pg_catalog.pg_constraint AS c
             WHERE c.conrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             )
               AND c.contype <> 'n'
               AND c.convalidated
               AND NOT c.condeferrable
               AND NOT c.condeferred
               AND c.connoinherit = (c.contype <> 'c')
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS c
         WHERE c.conrelid IN (
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         )
           AND c.contype <> 'n'
           AND (
               NOT c.convalidated OR c.condeferrable OR c.condeferred
               OR c.connoinherit IS DISTINCT FROM (c.contype <> 'c')
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta restricciones o claves foraneas alteradas';
    END IF;

    IF EXISTS (
        WITH esperado(tabla, indice, primaria, unica, exclusion, valida, lista, viva, inmediata) AS (
            SELECT c.conrelid, c.conindid, c.contype = 'p',
                   true, false, true, true, true, true
              FROM pg_catalog.pg_constraint AS c
             WHERE c.conrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             ) AND c.contype IN ('p', 'u')
        ), actual AS (
            SELECT i.indrelid, i.indexrelid, i.indisprimary, i.indisunique,
                   i.indisexclusion, i.indisvalid, i.indisready, i.indislive,
                   i.indimmediate
              FROM pg_catalog.pg_index AS i
             WHERE i.indrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             ) AND (i.indisprimary OR i.indisunique OR i.indisexclusion)
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta indices semanticos alterados';
    END IF;

    IF EXISTS (
        WITH esperado(tabla, columnas) AS (
            SELECT
                a.attrelid, ARRAY[a.attnum]::smallint[]
              FROM pg_catalog.pg_attribute AS a
             WHERE a.attrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             )
               AND a.attnum > 0
               AND NOT a.attisdropped
               AND a.attnotnull
        ),
        actual AS (
            SELECT c.conrelid, c.conkey
              FROM pg_catalog.pg_constraint AS c
             WHERE c.conrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             )
               AND c.contype = 'n'
               AND c.convalidated
               AND NOT c.condeferrable
               AND NOT c.condeferred
               AND NOT c.connoinherit
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS c
         WHERE c.conrelid IN (
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         )
           AND c.contype = 'n'
           AND (
               NOT c.convalidated OR c.condeferrable OR c.condeferred
               OR c.connoinherit
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta restricciones NOT NULL alteradas';
    END IF;
END
$estructura_exacta$;
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
    reentrada boolean := pg_catalog.current_setting(
        'vec_autorizacion.migracion_000008_reentrada'
    )::boolean;
BEGIN
    IF NOT reentrada THEN
        CREATE TRIGGER vinculacion_motivo_rrhh_inmutable
            BEFORE UPDATE OR DELETE ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
            FOR EACH ROW EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
        CREATE TRIGGER vinculacion_motivo_rrhh_no_truncar
            BEFORE TRUNCATE ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
            FOR EACH STATEMENT EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
        CREATE TRIGGER vinculacion_motivo_rrhh_checkpoint_avance
            BEFORE UPDATE ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            FOR EACH ROW EXECUTE FUNCTION
                vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1();
        CREATE TRIGGER vinculacion_motivo_rrhh_checkpoint_inmutable
            BEFORE INSERT OR DELETE ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            FOR EACH ROW EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
        CREATE TRIGGER vinculacion_motivo_rrhh_checkpoint_no_truncar
            BEFORE TRUNCATE ON vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            FOR EACH STATEMENT EXECUTE FUNCTION
                vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
    END IF;
END
$protecciones$;

DO $disparadores_exactos$
BEGIN
    IF EXISTS (
        WITH esperado(tabla, nombre, evento_orientacion, habilitado, funcion) AS (
            VALUES
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_inmutable', 27::smallint, 'O', 'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'vinculacion_motivo_rrhh_no_truncar', 34::smallint, 'O', 'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_avance', 19::smallint, 'O', 'vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()'::regprocedure),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_inmutable', 15::smallint, 'O', 'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'vinculacion_motivo_rrhh_checkpoint_no_truncar', 34::smallint, 'O', 'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure)
        ),
        actual AS (
            SELECT
                t.tgrelid, t.tgname::text, t.tgtype,
                t.tgenabled::text, t.tgfoid
              FROM pg_catalog.pg_trigger AS t
             WHERE t.tgrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             )
               AND NOT t.tgisinternal
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger AS t
         WHERE t.tgrelid IN (
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         )
           AND NOT t.tgisinternal
           AND ROW(t.tgparentid, t.tgconstrrelid, t.tgconstrindid,
                   t.tgconstraint, t.tgdeferrable, t.tginitdeferred,
                   t.tgnargs, t.tgattr::text, pg_catalog.encode(t.tgargs, 'hex'),
                   t.tgqual IS NULL, t.tgoldtable IS NULL, t.tgnewtable IS NULL)
               IS DISTINCT FROM ROW(0::oid, 0::oid, 0::oid, 0::oid,
                   false, false, 0, '', '', true, true, true)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta disparadores alterados';
    END IF;
    IF EXISTS (
        WITH claves AS (
            SELECT c.oid, c.conrelid, c.confrelid, c.conindid
              FROM pg_catalog.pg_constraint AS c
             WHERE c.conrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             ) AND c.contype = 'f'
        ), esperado AS (
            SELECT c.oid, x.tabla, x.otra, c.conindid, x.tipo,
                   'O', true, x.funcion, true
              FROM claves AS c CROSS JOIN LATERAL (VALUES
                (c.conrelid, c.confrelid, 5::smallint, 'RI_FKey_check_ins'),
                (c.conrelid, c.confrelid, 17::smallint, 'RI_FKey_check_upd'),
                (c.confrelid, c.conrelid, 9::smallint, 'RI_FKey_noaction_del'),
                (c.confrelid, c.conrelid, 17::smallint, 'RI_FKey_noaction_upd')
              ) AS x(tabla, otra, tipo, funcion)
        ), actual AS (
            SELECT t.tgconstraint, t.tgrelid, t.tgconstrrelid,
                   t.tgconstrindid, t.tgtype, t.tgenabled::text,
                   t.tgisinternal, p.proname::text,
                   ROW(t.tgparentid, t.tgdeferrable, t.tginitdeferred,
                       t.tgnargs, t.tgattr::text, pg_catalog.encode(t.tgargs, 'hex'),
                       t.tgqual IS NULL, t.tgoldtable IS NULL, t.tgnewtable IS NULL)
                   = ROW(0::oid, false, false, 0, '', '', true, true, true)
              FROM pg_catalog.pg_trigger AS t
              JOIN pg_catalog.pg_proc AS p ON p.oid = t.tgfoid
             WHERE t.tgconstraint IN (SELECT oid FROM claves)
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta disparadores RI alterados';
    END IF;
END
$disparadores_exactos$;

DO $rls$
DECLARE
    tabla regclass;
    reentrada boolean := pg_catalog.current_setting(
        'vec_autorizacion.migracion_000008_reentrada'
    )::boolean;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
    ] LOOP
        IF NOT reentrada THEN
            EXECUTE pg_catalog.format(
                'ALTER TABLE %s ENABLE ROW LEVEL SECURITY', tabla
            );
            EXECUTE pg_catalog.format(
                'ALTER TABLE %s FORCE ROW LEVEL SECURITY', tabla
            );
        END IF;
        IF NOT reentrada AND NOT EXISTS (
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

DO $rls_exacto$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS c
         WHERE c.oid IN (
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
             'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         )
           AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity)
    ) OR EXISTS (
        WITH esperado(
            tabla, nombre, permisiva, orden, roles, condicion, comprobacion
        ) AS (
            VALUES
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass, 'acceso_propietario_exacto', true, '*', ARRAY['vec_autorizacion_propietario'::regrole::oid], '(CURRENT_USER = ''vec_autorizacion_propietario''::name)', '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'),
              ('vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass, 'acceso_propietario_exacto', true, '*', ARRAY['vec_autorizacion_propietario'::regrole::oid], '(CURRENT_USER = ''vec_autorizacion_propietario''::name)', '(CURRENT_USER = ''vec_autorizacion_propietario''::name)')
        ),
        actual AS (
            SELECT
                p.polrelid, p.polname::text, p.polpermissive,
                p.polcmd::text, p.polroles,
                pg_catalog.pg_get_expr(p.polqual, p.polrelid),
                pg_catalog.pg_get_expr(p.polwithcheck, p.polrelid)
              FROM pg_catalog.pg_policy AS p
             WHERE p.polrelid IN (
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
                 'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
             )
        )
        (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
        UNION ALL
        (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no adopta RLS o politicas alteradas';
    END IF;
END
$rls_exacto$;

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
