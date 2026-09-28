\set ON_ERROR_STOP on
-- C3: cierra únicamente los tipos fila de Bolsa B1 y autorización AD4 que
-- existen antes del alta del selector RRHH. No restaura permisos con DOWN.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
-- Este delta pertenece a la cadena nueva AD4+B1. La base principal puede
-- carecer por completo de B1: ese estado no acredita ACL cerrada y no se
-- modifica. Un B1 parcial o derivado sí se rechaza antes de cualquier REVOKE.
DO $alcance$
DECLARE
    tablas integer;
    tipos integer;
BEGIN
    WITH objetivos(nombre) AS (
        SELECT pg_catalog.unnest(ARRAY[
            'bolsa_autoritativa', 'necesidad_autoritativa', 'necesidad_actual',
            'politica_autoritativa', 'instantanea_autoritativa',
            'evaluacion_autoritativa', 'atestacion_autorizacion_version',
            'atestacion_autorizacion_actual', 'propuesta', 'referencia_consumida',
            'uso_decision', 'auditoria', 'auditoria_actual', 'outbox'
        ])
    )
    SELECT count(c.oid), count(t.oid) INTO tablas, tipos
      FROM objetivos o
      LEFT JOIN pg_catalog.pg_namespace n
        ON n.nspname = 'vec_bolsa_llamamientos'
      LEFT JOIN pg_catalog.pg_class c
        ON c.relnamespace = n.oid AND c.relname = o.nombre
      LEFT JOIN pg_catalog.pg_type t
        ON t.typnamespace = n.oid AND t.typname = o.nombre;
    IF tablas = 0 AND tipos = 0 THEN
        PERFORM pg_catalog.set_config('vec.c3_acl_aplicar', 'false', true);
        RETURN;
    END IF;
    IF tablas <> 14 OR tipos <> 14 THEN
        RAISE EXCEPTION 'C3 ACL tipos: B1 parcial: %/14 tablas, %/14 tipos',
            tablas, tipos USING ERRCODE = '55000';
    END IF;
    PERFORM pg_catalog.set_config('vec.c3_acl_aplicar', 'true', true);
END $alcance$;
SELECT pg_catalog.current_setting('vec.c3_acl_aplicar') AS c3_acl_aplicar \gset
\if :c3_acl_aplicar
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contexto_actor_v1:acl-tipos-preselector:c3:v1', 0)
);

DO $preimagen$
DECLARE
    nombre text;
    esquema text;
    dueno name;
    tipo pg_catalog.pg_type%ROWTYPE;
    relacion pg_catalog.pg_class%ROWTYPE;
    cantidad integer := 0;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper) THEN
        RAISE EXCEPTION 'C3 ACL tipos: requiere DBA superusuario' USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.current_setting('server_version_num')::integer < 180000 THEN
        RAISE EXCEPTION 'C3 ACL tipos: requiere PostgreSQL 18' USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NOT NULL THEN
        RAISE EXCEPTION 'C3 ACL tipos: selector ya existe' USING ERRCODE = '55000';
    END IF;
    FOREACH nombre IN ARRAY ARRAY[
        'bolsa_autoritativa', 'necesidad_autoritativa', 'necesidad_actual',
        'politica_autoritativa', 'instantanea_autoritativa',
        'evaluacion_autoritativa', 'atestacion_autorizacion_version',
        'atestacion_autorizacion_actual', 'propuesta', 'referencia_consumida',
        'uso_decision', 'auditoria', 'auditoria_actual', 'outbox',
        'decision_autorizacion_solicitud_ligada_v2'
    ] LOOP
        cantidad := cantidad + 1;
        IF cantidad <= 14 THEN
            esquema := 'vec_bolsa_llamamientos';
            dueno := 'vec_bolsa_llamamientos_propietario';
        ELSE
            esquema := 'vec_autorizacion';
            dueno := 'vec_autorizacion_propietario';
        END IF;
        SELECT t.* INTO tipo FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE n.nspname = esquema AND t.typname = nombre;
        IF NOT FOUND OR pg_catalog.to_regrole(dueno) IS NULL
           OR tipo.typtype <> 'c' OR tipo.typrelid = 0
           OR tipo.typowner <> pg_catalog.to_regrole(dueno)::oid
           OR tipo.typisdefined IS NOT TRUE
           OR (SELECT n.nspowner FROM pg_catalog.pg_namespace AS n
                WHERE n.oid = tipo.typnamespace) <> pg_catalog.to_regrole(dueno)::oid THEN
            RAISE EXCEPTION 'C3 ACL tipos: tipo u owner incompatible: %.%', esquema, nombre
                USING ERRCODE = '55000';
        END IF;
        SELECT c.* INTO relacion FROM pg_catalog.pg_class AS c WHERE c.oid = tipo.typrelid;
        IF NOT FOUND OR relacion.relkind <> 'r' OR relacion.reltype <> tipo.oid
           OR relacion.relowner <> tipo.typowner OR relacion.relnamespace <> tipo.typnamespace
           OR relacion.relname <> tipo.typname THEN
            RAISE EXCEPTION 'C3 ACL tipos: fila no ligada a su tabla: %.%', esquema, nombre
                USING ERRCODE = '55000';
        END IF;
        -- La preimagen B1 debe conservar el USAGE implícito que se corrige.
        -- AD4 puede estar ya cerrada por AD5 en cadenas posteriores.
        IF (cantidad <= 14 AND tipo.typacl IS NOT NULL)
           OR NOT EXISTS (
                SELECT 1 FROM pg_catalog.aclexplode(coalesce(
                    tipo.typacl, pg_catalog.acldefault('T', tipo.typowner))) AS a
                 WHERE a.grantee = tipo.typowner AND a.grantor = tipo.typowner
                   AND a.privilege_type = 'USAGE' AND NOT a.is_grantable)
           OR EXISTS (
                SELECT 1 FROM pg_catalog.aclexplode(coalesce(
                    tipo.typacl, pg_catalog.acldefault('T', tipo.typowner))) AS a
                 WHERE NOT (a.grantee = tipo.typowner AND a.grantor = tipo.typowner
                            AND a.privilege_type = 'USAGE' AND NOT a.is_grantable)
                   AND NOT (a.grantee = 0 AND a.grantor = tipo.typowner
                            AND a.privilege_type = 'USAGE' AND NOT a.is_grantable))
           OR (cantidad <= 14 AND NOT EXISTS (
                SELECT 1 FROM pg_catalog.aclexplode(coalesce(
                    tipo.typacl, pg_catalog.acldefault('T', tipo.typowner))) AS a
                 WHERE a.grantee = 0 AND a.privilege_type = 'USAGE')) THEN
            RAISE EXCEPTION 'C3 ACL tipos: ACL previa incompatible: %.%', esquema, nombre
                USING ERRCODE = '55000';
        END IF;
    END LOOP;
    IF cantidad <> 15 THEN
        RAISE EXCEPTION 'C3 ACL tipos: lista incompleta' USING ERRCODE = '55000';
    END IF;
END $preimagen$;

REVOKE USAGE ON TYPE
    vec_bolsa_llamamientos.bolsa_autoritativa,
    vec_bolsa_llamamientos.necesidad_autoritativa,
    vec_bolsa_llamamientos.necesidad_actual,
    vec_bolsa_llamamientos.politica_autoritativa,
    vec_bolsa_llamamientos.instantanea_autoritativa,
    vec_bolsa_llamamientos.evaluacion_autoritativa,
    vec_bolsa_llamamientos.atestacion_autorizacion_version,
    vec_bolsa_llamamientos.atestacion_autorizacion_actual,
    vec_bolsa_llamamientos.propuesta,
    vec_bolsa_llamamientos.referencia_consumida,
    vec_bolsa_llamamientos.uso_decision,
    vec_bolsa_llamamientos.auditoria,
    vec_bolsa_llamamientos.auditoria_actual,
    vec_bolsa_llamamientos.outbox,
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FROM PUBLIC;

DO $postimagen$
DECLARE
    cantidad integer;
BEGIN
    SELECT count(*) INTO cantidad
      FROM pg_catalog.pg_type AS t
      JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
     WHERE (n.nspname = 'vec_bolsa_llamamientos' AND t.typname = ANY (ARRAY[
               'bolsa_autoritativa', 'necesidad_autoritativa', 'necesidad_actual',
               'politica_autoritativa', 'instantanea_autoritativa',
               'evaluacion_autoritativa', 'atestacion_autorizacion_version',
               'atestacion_autorizacion_actual', 'propuesta', 'referencia_consumida',
               'uso_decision', 'auditoria', 'auditoria_actual', 'outbox']))
        OR (n.nspname = 'vec_autorizacion'
            AND t.typname = 'decision_autorizacion_solicitud_ligada_v2');
    IF cantidad <> 15 OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_type AS t
        JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
        CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(
            t.typacl, pg_catalog.acldefault('T', t.typowner))) AS a
        WHERE ((n.nspname = 'vec_bolsa_llamamientos' AND t.typname = ANY (ARRAY[
                   'bolsa_autoritativa', 'necesidad_autoritativa', 'necesidad_actual',
                   'politica_autoritativa', 'instantanea_autoritativa',
                   'evaluacion_autoritativa', 'atestacion_autorizacion_version',
                   'atestacion_autorizacion_actual', 'propuesta', 'referencia_consumida',
                   'uso_decision', 'auditoria', 'auditoria_actual', 'outbox']))
           OR (n.nspname = 'vec_autorizacion'
               AND t.typname = 'decision_autorizacion_solicitud_ligada_v2'))
          AND a.grantee = 0
    ) THEN
        RAISE EXCEPTION 'C3 ACL tipos: postimagen abierta' USING ERRCODE = '55000';
    END IF;
END $postimagen$;
\else
\echo 'NO_APLICA C3 ACL tipos: B1 ausente (0/14 tablas y tipos); cadena desde cero no instalada'
\endif
COMMIT;
