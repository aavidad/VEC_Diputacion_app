\set ON_ERROR_STOP on

SELECT jsonb_build_object(
    'actor_id', 'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
    'actor_profile', 'auditoria-interna',
    'actor_roles', '["auditor","supervisor"]'::jsonb,
    'represented_subject_id', '', 'auth_method', 'sso',
    'auth_assurance', 'alto', 'authorization_ref', '',
    'purpose', 'tramitacion', 'action', 'expediente.leer',
    'module_id', 'vec.module.bolsa',
    'subject_ref', 'expediente:sha256:' || repeat('b', 64),
    'object_version', 7,
    'expediente_ref', 'expediente:sha256:' || repeat('c', 64),
    'document_ref', '', 'rule_ref', '', 'reason', '',
    'result', 'permitido', 'before_hash', '', 'after_hash', '',
    'correlation_ref', 'registro-prueba-' || repeat('d', 32),
    'metadata', '{"canal":"interno"}'::jsonb,
    'occurred_at', to_char(
        clock_timestamp() AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    )
)::text AS entrada \gset

-- La mecánica interna solo se prueba como propietario. Ningún rol runtime
-- puede ejecutar este append sin un wrapper VEC ligado a su efecto.
SET SESSION AUTHORIZATION vec_bolsa_accesos_propietario;
PREPARE registrar_t13(jsonb) AS
    SELECT vec_bolsa_registro_accesos.registrar_interno_v1($1);
\set ON_ERROR_STOP off
BEGIN;
EXECUTE registrar_t13(:'entrada'::jsonb);
\set estado_aislamiento :SQLSTATE
ROLLBACK;
\set ON_ERROR_STOP on
SELECT :'estado_aislamiento' = '22023' AS aislamiento_denegado \gset
\if :aislamiento_denegado
\else
    \echo 'el append interno aceptó aislamiento inferior a serializable'
    SELECT 1 / 0;
\endif
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
EXECUTE registrar_t13(:'entrada'::jsonb) \gset primera_
COMMIT;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
EXECUTE registrar_t13(:'entrada'::jsonb) \gset repetida_
COMMIT;
DEALLOCATE registrar_t13;
RESET SESSION AUTHORIZATION;

SELECT :'primera_registrar_interno_v1'::jsonb =
       :'repetida_registrar_interno_v1'::jsonb AS replay_idempotente \gset
\if :replay_idempotente
\else
    \echo 'el reintento exacto del append no devolvió el mismo registro'
    SELECT 1 / 0;
\endif

SELECT jsonb_set(
    :'entrada'::jsonb,
    '{correlation_ref}',
    to_jsonb('registro-rollback-' || repeat('e', 32))
)::text AS entrada_rollback \gset
SET SESSION AUTHORIZATION vec_bolsa_accesos_propietario;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.registrar_interno_v1(
    :'entrada_rollback'::jsonb
);
ROLLBACK;
RESET SESSION AUTHORIZATION;

DO $comprobar$
DECLARE
    errores integer;
BEGIN
    IF (SELECT count(*)
          FROM vec_bolsa_registro_accesos.registro_acceso) <> 1
       OR EXISTS (
           SELECT 1
             FROM vec_bolsa_registro_accesos.registro_acceso
            WHERE correlation_ref = 'registro-rollback-' || repeat('e', 32)
       ) THEN
        RAISE EXCEPTION 'rollback o idempotencia T13 incorrectos';
    END IF;
    SELECT count(*) INTO errores
      FROM (
          SELECT secuencia, firma_anterior,
                 lag(firma) OVER (ORDER BY secuencia) AS anterior,
                 firma,
                 encode(sha256(
                     decode(firma_anterior, 'hex') || registro_canonico
                 ), 'hex') AS recalculada
            FROM vec_bolsa_registro_accesos.registro_acceso
      ) AS cadena
     WHERE firma <> recalculada
        OR (secuencia = 1 AND firma_anterior <> repeat('0', 64))
        OR (secuencia > 1 AND firma_anterior <> anterior);
    IF errores <> 0 THEN
        RAISE EXCEPTION 'cadena T13 discontinua';
    END IF;
    BEGIN
        UPDATE vec_bolsa_registro_accesos.registro_acceso
           SET result = 'error' WHERE secuencia = 1;
        RAISE EXCEPTION 'adulteracion aceptada';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
END
$comprobar$;
