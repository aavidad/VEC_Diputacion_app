\set ON_ERROR_STOP on

DO $catalogo$
DECLARE
    tabla regclass;
    acl_ajena integer;
    acl_tipo_ajena integer;
    politicas integer;
    funciones_inseguras integer;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
        'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
    ] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_class
             WHERE oid = tabla
               AND relkind = 'r'
               AND relowner = 'vec_autorizacion_propietario'::regrole
               AND relrowsecurity
               AND relforcerowsecurity
        ) THEN
            RAISE EXCEPTION 'tabla 000008 sin propietario o RLS exactos: %',
                tabla;
        END IF;
        SELECT pg_catalog.count(*) INTO politicas
          FROM pg_catalog.pg_policy
         WHERE polrelid = tabla
           AND polname = 'acceso_propietario_exacto'
           AND polcmd = '*'
           AND polroles =
               ARRAY['vec_autorizacion_propietario'::regrole::oid]
           AND pg_catalog.pg_get_expr(polqual, polrelid) =
               '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
           AND pg_catalog.pg_get_expr(polwithcheck, polrelid) =
               '(CURRENT_USER = ''vec_autorizacion_propietario''::name)';
        IF politicas <> 1 OR (
            SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_policy
             WHERE polrelid = tabla
        ) <> 1 THEN
            RAISE EXCEPTION 'politica RLS inesperada en %', tabla;
        END IF;
    END LOOP;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS c
         WHERE c.conrelid =
               'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass
           AND c.conname =
               'motivo_v2_catalogo_referencia_completa_unica'
           AND c.contype = 'u'
           AND c.convalidated
           AND NOT c.condeferrable
           AND NOT c.condeferred
           AND c.connoinherit
           AND pg_catalog.pg_get_constraintdef(c.oid, true) =
               'UNIQUE (catalogo_id, catalogo_version, catalogo_huella_publicada_sha256)'
           AND pg_catalog.obj_description(c.oid, 'pg_constraint') =
               'vec_autorizacion:vinculacion-motivo-consulta-rrhh:referencia-completa:v1:000008'
    ) THEN
        RAISE EXCEPTION 'UNIQUE 000008 sin estructura o procedencia exactas';
    END IF;

    SELECT pg_catalog.count(*) INTO acl_ajena
      FROM pg_catalog.pg_class AS c
      CROSS JOIN LATERAL pg_catalog.aclexplode(
          coalesce(c.relacl, pg_catalog.acldefault('r', c.relowner))
      ) AS acl
     WHERE c.oid IN (
           'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
           'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
       )
       AND acl.grantee <> c.relowner;
    IF acl_ajena <> 0 THEN
        RAISE EXCEPTION '000008 concedio privilegios de tabla ajenos';
    END IF;

    WITH tipos_relacion AS (
        SELECT t.oid, t.typarray, c.relowner
          FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_type AS t ON t.oid = c.reltype
         WHERE c.oid IN (
            'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
            'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
         )
    ), tipos AS (
        SELECT oid, relowner FROM tipos_relacion
        UNION ALL
        SELECT typarray, relowner FROM tipos_relacion
    )
    SELECT pg_catalog.count(*) INTO acl_tipo_ajena
      FROM tipos
     WHERE pg_catalog.has_type_privilege('public', oid, 'USAGE')
        OR pg_catalog.has_type_privilege(
            'vec_autorizacion_motivos_proyector', oid, 'USAGE'
        )
        OR pg_catalog.has_type_privilege(
            'vec_autorizacion_motivos_evaluador', oid, 'USAGE'
        )
        OR NOT pg_catalog.has_type_privilege(relowner, oid, 'USAGE');
    IF acl_tipo_ajena <> 0 THEN
        RAISE EXCEPTION '000008 dejo tipos compuestos o arrays expuestos';
    END IF;

    SELECT pg_catalog.count(*) INTO funciones_inseguras
      FROM pg_catalog.pg_proc AS p
     WHERE p.oid IN (
        'vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()'::regprocedure,
        'vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()'::regprocedure
     )
       AND (
           p.proowner <> 'vec_autorizacion_propietario'::regrole
           OR p.prosecdef
           OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
           OR pg_catalog.has_function_privilege('public', p.oid, 'EXECUTE')
           OR pg_catalog.has_function_privilege(
               'vec_autorizacion_motivos_proyector', p.oid, 'EXECUTE'
           )
           OR pg_catalog.has_function_privilege(
               'vec_autorizacion_motivos_evaluador', p.oid, 'EXECUTE'
           )
       );
    IF funciones_inseguras <> 0 THEN
        RAISE EXCEPTION 'funciones privadas 000008 con superficie insegura';
    END IF;
END
$catalogo$;

DO $checkpoint_inicial$
BEGIN
    IF (SELECT pg_catalog.count(*)
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
         WHERE clase_consulta IN ('cuadro', 'detalle')
           AND ultima_publicacion_version = 0
           AND ultima_publicacion_ref IS NULL
           AND ultima_publicacion_huella_sha256 IS NULL) <> 2
       OR EXISTS (
           SELECT 1
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
       ) THEN
        RAISE EXCEPTION 'estado inicial 000008 inesperado';
    END IF;
END
$checkpoint_inicial$;

SET ROLE vec_autorizacion_motivos_proyector;
SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
    'evento_11111111111111111111111111111111',
    1,
    pg_catalog.repeat('1', 64),
    'motivos_rrhh_prueba',
    1,
    pg_catalog.repeat('2', 64),
    pg_catalog.clock_timestamp() - interval '1 minute',
    pg_catalog.jsonb_build_array(
        pg_catalog.jsonb_build_object(
            'clave', 'motivo_33333333333333333333333333333333',
            'vigente_desde', pg_catalog.to_char(
                pg_catalog.clock_timestamp() - interval '1 hour',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'vigente_hasta', NULL
        ),
        pg_catalog.jsonb_build_object(
            'clave', 'motivo_44444444444444444444444444444444',
            'vigente_desde', pg_catalog.to_char(
                pg_catalog.clock_timestamp() - interval '1 hour',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'vigente_hasta', NULL
        )
    )
) AS catalogo_sintetico_publicado
\gset
\if :catalogo_sintetico_publicado
\else
  \quit 1
\endif
RESET ROLE;

SET ROLE vec_autorizacion_propietario;

DO $fk_inexistente$
BEGIN
    BEGIN
        INSERT INTO vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (
            clase_consulta, publicacion_version, publicacion_ref,
            publicacion_huella_sha256, catalogo_id, catalogo_version,
            catalogo_huella_sha256, entrada_clave, publicada_en
        ) VALUES (
            'cuadro', 1,
            'publicacion_motivo_rrhh_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
            pg_catalog.repeat('a', 64), 'catalogo_inexistente', 1,
            pg_catalog.repeat('b', 64),
            'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
            pg_catalog.clock_timestamp()
        );
        RAISE EXCEPTION 'se admitio una referencia de motivo inexistente';
    EXCEPTION WHEN foreign_key_violation THEN
        NULL;
    END;
END
$fk_inexistente$;

DO $avance_no_atomico$
BEGIN
    BEGIN
        UPDATE
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
           SET ultima_publicacion_version = 1,
               ultima_publicacion_ref =
                 'publicacion_motivo_rrhh_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
               ultima_publicacion_huella_sha256 = pg_catalog.repeat('a', 64),
               actualizado_en = pg_catalog.clock_timestamp()
         WHERE clase_consulta = 'cuadro';
        RAISE EXCEPTION 'se admitio un avance sin historia';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END
$avance_no_atomico$;

BEGIN;
INSERT INTO vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 (
    clase_consulta, publicacion_version, publicacion_ref,
    publicacion_huella_sha256, catalogo_id, catalogo_version,
    catalogo_huella_sha256, entrada_clave, publicada_en
) VALUES
    (
        'cuadro', 1,
        'publicacion_motivo_rrhh_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        pg_catalog.repeat('a', 64), 'motivos_rrhh_prueba', 1,
        pg_catalog.repeat('2', 64),
        'motivo_33333333333333333333333333333333',
        pg_catalog.clock_timestamp()
    ),
    (
        'detalle', 1,
        'publicacion_motivo_rrhh_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        pg_catalog.repeat('b', 64), 'motivos_rrhh_prueba', 1,
        pg_catalog.repeat('2', 64),
        'motivo_44444444444444444444444444444444',
        pg_catalog.clock_timestamp()
    );

UPDATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 AS c
   SET ultima_publicacion_version = h.publicacion_version,
       ultima_publicacion_ref = h.publicacion_ref,
       ultima_publicacion_huella_sha256 =
           h.publicacion_huella_sha256,
       actualizado_en = pg_catalog.clock_timestamp()
  FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 AS h
 WHERE h.clase_consulta = c.clase_consulta
   AND h.publicacion_version = 1;
COMMIT;

DO $dml_hostil$
BEGIN
    BEGIN
        UPDATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
           SET entrada_clave =
               'motivo_55555555555555555555555555555555'
         WHERE clase_consulta = 'cuadro';
        RAISE EXCEPTION 'UPDATE altero historia 000008';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        DELETE FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
         WHERE clase_consulta = 'cuadro';
        RAISE EXCEPTION 'DELETE altero historia 000008';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        TRUNCATE
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1,
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1;
        RAISE EXCEPTION 'TRUNCATE altero historia 000008';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        DELETE FROM
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
         WHERE clase_consulta = 'cuadro';
        RAISE EXCEPTION 'DELETE altero checkpoint 000008';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        INSERT INTO
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1 (
            clase_consulta, ultima_publicacion_version, actualizado_en
        ) VALUES ('otra', 0, pg_catalog.clock_timestamp());
        RAISE EXCEPTION 'INSERT altero checkpoint 000008';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
    BEGIN
        TRUNCATE
          vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1;
        RAISE EXCEPTION 'TRUNCATE altero checkpoint 000008';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
    END;
END
$dml_hostil$;

RESET ROLE;

DO $resultado$
BEGIN
    IF (SELECT pg_catalog.count(*)
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1) <> 2
       OR (SELECT pg_catalog.count(*)
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            WHERE ultima_publicacion_version = 1) <> 2 THEN
        RAISE EXCEPTION 'la evidencia 000008 no quedo integra';
    END IF;
END
$resultado$;
