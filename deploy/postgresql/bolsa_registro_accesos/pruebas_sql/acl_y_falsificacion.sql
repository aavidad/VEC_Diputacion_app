\set ON_ERROR_STOP on

DO $roles$
DECLARE
    rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_accesos_propietario', 'vec_bolsa_accesos_migrador',
        'vec_bolsa_accesos_registrador', 'vec_bolsa_accesos_consultor',
        'vec_bolsa_accesos_gobernador'
    ] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_roles
             WHERE rolname = rol AND NOT rolcanlogin AND NOT rolsuper
               AND NOT rolcreatedb AND NOT rolcreaterole
               AND NOT rolreplication AND NOT rolbypassrls
        ) THEN
            RAISE EXCEPTION 'rol T13 inseguro: %', rol;
        END IF;
    END LOOP;
    IF pg_catalog.has_database_privilege(
           'vec_bolsa_accesos_propietario', pg_catalog.current_database(),
           'CREATE'
       )
       OR pg_catalog.has_schema_privilege(
           'public', 'vec_bolsa_registro_accesos', 'USAGE'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname IN (
                'vec_bolsa_registro_accesos', 'vec_autorizacion'
            )
              AND p.proname IN (
                  'consultar_accesos_administrativos_v1',
                  'registrar_acceso_v1',
                  'revalidar_decision_registro_accesos_bolsa_v2'
              )
              AND (
                  p.proconfig IS NULL
                  OR NOT ('search_path=pg_catalog' = ANY (p.proconfig))
                  OR EXISTS (
                      SELECT 1 FROM pg_catalog.unnest(p.proconfig) AS ajuste
                       WHERE ajuste LIKE '%pg_temp%'
                  )
              )
       ) THEN
        RAISE EXCEPTION 'ACL/search_path T13 no cerrados';
    END IF;
END
$roles$;

-- La prueba de la decisión tiene cinco strings obligatorios. En particular,
-- JSON null no puede alcanzar el cast de verificada_en ni la lógica temporal.
DO $prueba_vec_tipada$
DECLARE
    origen text;
    prueba_nula jsonb := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'decision_ref', 'decision_tipo_t13',
        'huella_decision_sha256',
            encode(sha256(convert_to('{}', 'UTF8')), 'hex'),
        'verificada_en', NULL,
        'principal_ref', 'principal_tipo_t13'
    );
BEGIN
    SELECT p.prosrc INTO STRICT origen
      FROM pg_catalog.pg_proc AS p
     WHERE p.oid =
           'vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)'::regprocedure;
    IF origen NOT LIKE
           '%jsonb_typeof(valor) IS DISTINCT FROM ''string''%'
       OR vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(
           prueba_nula,
           convert_to('{}', 'UTF8'),
           convert_to('{}', 'UTF8'),
           'correlacion_00000000000000000000000000000000',
           'consulta-accesos:sha256:' || repeat('1', 64),
           'control-interno',
           '[
               "accion", "actor_seudonimizado", "expediente_ref",
               "finalidad", "modulo_id", "ocurrido_en", "recurso_ref",
               "resultado", "version_esquema", "version_objeto"
           ]'::jsonb,
           'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
           'sso',
           'alto'
       ) IS DISTINCT FROM false THEN
        RAISE EXCEPTION 'p_prueba T13 aceptó verificada_en JSON null';
    END IF;
END
$prueba_vec_tipada$;

CREATE ROLE vec_bolsa_accesos_registrador_prueba LOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_bolsa_accesos_consultor_prueba LOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_accesos_registrador
    TO vec_bolsa_accesos_registrador_prueba;
GRANT vec_bolsa_accesos_consultor
    TO vec_bolsa_accesos_consultor_prueba;

-- Un mero registrador no puede inventar actor, decisión, finalidad ni efecto.
SET SESSION AUTHORIZATION vec_bolsa_accesos_registrador_prueba;
\set ON_ERROR_STOP off
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.registrar_acceso_v1(
    jsonb_build_object(
        'actor_id', 'hmac-sha256:bolsa_accesos_v1:' || repeat('f', 64),
        'actor_profile', 'administrador-inventado',
        'actor_roles', '["superadministrador"]'::jsonb,
        'represented_subject_id', '', 'auth_method', 'sso',
        'auth_assurance', 'alto',
        'authorization_ref', 'decision_inventada',
        'purpose', 'finalidad-inventada',
        'action', 'expediente.leer',
        'module_id', 'vec.module.bolsa',
        'subject_ref', 'expediente:sha256:' || repeat('e', 64),
        'object_version', 1, 'expediente_ref', '', 'document_ref', '',
        'rule_ref', '', 'reason', '', 'result', 'permitido',
        'before_hash', '', 'after_hash', '',
        'correlation_ref', 'falsificacion-registrador-' || repeat('d', 32),
        'metadata', '{}'::jsonb,
        'occurred_at', to_char(
            clock_timestamp() AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    )
);
\set estado_registrador_falso :SQLSTATE
ROLLBACK;
\set ON_ERROR_STOP on
RESET SESSION AUTHORIZATION;
SELECT :'estado_registrador_falso' = '42501'
       AS registrador_falso_denegado \gset
\if :registrador_falso_denegado
\else
    \echo 'el rol registrador pudo fabricar un acceso'
    SELECT 1 / 0;
\endif

SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
\set ON_ERROR_STOP off
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(
    jsonb_build_object(
        'version', 1,
        'filtro', jsonb_build_object(
            'version', 1,
            'actor_seudonimizado',
                'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
            'module_id', '', 'accion', '', 'finalidad_acceso', '',
            'recurso_ref', '', 'expediente_ref', '', 'resultado', '',
            'desde_inclusive', to_char(
                (clock_timestamp() - interval '1 hour')
                    AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'hasta_exclusive', to_char(
                (clock_timestamp() + interval '1 hour')
                    AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'version_objeto', 0, 'limite', 10, 'cursor', '',
            'finalidad_consulta', 'control-interno'
        ),
        'auditoria', jsonb_build_object(
            'actor_id', 'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
            'actor_profile', 'auditoria-interna',
            'actor_roles', '["auditor"]'::jsonb,
            'represented_subject_id', '', 'auth_method', 'sso',
            'auth_assurance', 'alto', 'authorization_ref', 'decision_falsa',
            'purpose', 'control-interno',
            'action', 'bolsa.registro_accesos.consultar',
            'module_id', 'vec.module.bolsa',
            'subject_ref', 'consulta-accesos:sha256:' || repeat('b', 64),
            'object_version', 1, 'expediente_ref', '', 'document_ref', '',
            'rule_ref', '', 'reason', '', 'result', 'permitido',
            'before_hash', '', 'after_hash', '',
            'correlation_ref', 'correlacion_' || repeat('c', 32),
            'metadata', '{}'::jsonb,
            'occurred_at', to_char(
                clock_timestamp() AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            )
        ),
        'autorizacion', jsonb_build_object(
            'prueba', jsonb_build_object(
                'esquema_huella',
                    'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
                'decision_ref', 'decision_falsa',
                'huella_decision_sha256',
                    encode(sha256(convert_to('{}', 'UTF8')), 'hex'),
                'verificada_en', to_char(
                    clock_timestamp() AT TIME ZONE 'UTC',
                    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
                ),
                'principal_ref', 'principal_falso'
            ),
            'decision_canonica',
                encode(convert_to('{}', 'UTF8'), 'base64'),
            'recurso_canonico', encode(convert_to(
                jsonb_build_object(
                    'ambitos', jsonb_build_object('version_esquema', '1'),
                    'atributos', jsonb_build_object(
                        'actor_operador_seudonimizado',
                            'hmac-sha256:bolsa_accesos_v1:' ||
                            repeat('a', 64),
                        'finalidad_consulta', 'control-interno'
                    )
                )::text, 'UTF8'
            ), 'base64')
        )
    )
);
\set estado_falsificacion :SQLSTATE
ROLLBACK;
\set ON_ERROR_STOP on
RESET SESSION AUTHORIZATION;
SELECT :'estado_falsificacion' = '42501' AS falsificacion_denegada \gset
\if :falsificacion_denegada
\else
    \echo 'una decisión fabricada por runtime no fue denegada'
    SELECT 1 / 0;
\endif

DO $tablas$
BEGIN
    IF pg_catalog.has_table_privilege(
           'vec_bolsa_accesos_consultor_prueba',
           'vec_bolsa_registro_accesos.registro_acceso', 'SELECT'
       )
       OR pg_catalog.has_table_privilege(
           'vec_bolsa_accesos_registrador_prueba',
           'vec_bolsa_registro_accesos.registro_acceso', 'INSERT'
       )
       OR pg_catalog.has_function_privilege(
           'vec_bolsa_accesos_registrador_prueba',
           'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_bolsa_accesos_gobernador',
           'vec_bolsa_registro_accesos.publicar_politica_retencion_v1(jsonb)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_bolsa_accesos_consultor_prueba',
           'vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'ACL runtime T13 demasiado amplia';
    END IF;
END
$tablas$;
