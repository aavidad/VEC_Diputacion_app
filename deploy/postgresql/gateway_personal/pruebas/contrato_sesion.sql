-- Ejecutar despues de roles_up.sql y 000001_sesion_gateway.up.sql en una base
-- desechable. Usa solo referencias y huellas sinteticas.
BEGIN;
SET LOCAL ROLE vec_gateway_personal_propietario;
SET LOCAL search_path = pg_catalog;

DO $prueba_acl$
BEGIN
    IF pg_catalog.has_table_privilege(
           'vec_gateway_personal_ejecutor',
           'vec_gateway_personal.sesion', 'SELECT,INSERT,UPDATE,DELETE'
       ) OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS clase
           CROSS JOIN LATERAL pg_catalog.aclexplode(
               COALESCE(clase.relacl, pg_catalog.acldefault('r', clase.relowner))
           ) AS acl
           WHERE clase.oid = 'vec_gateway_personal.sesion'::regclass
             AND acl.grantee = 0
       ) OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS funcion
           CROSS JOIN LATERAL pg_catalog.aclexplode(
               COALESCE(funcion.proacl, pg_catalog.acldefault('f', funcion.proowner))
           ) AS acl
           WHERE funcion.oid = 'vec_gateway_personal.abrir_sesion_v1(text,text,text,timestamptz,timestamptz)'::regprocedure
             AND acl.grantee = 0 AND acl.privilege_type = 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'ACL de sesion gateway abierta';
    END IF;
    IF NOT pg_catalog.has_function_privilege(
        'vec_gateway_personal_ejecutor',
        'vec_gateway_personal.abrir_sesion_v1(text,text,text,timestamptz,timestamptz)',
        'EXECUTE'
    ) OR NOT pg_catalog.has_function_privilege(
        'vec_gateway_personal_ejecutor',
        'vec_gateway_personal.consultar_sesion_v1(text,timestamptz)', 'EXECUTE'
    ) OR NOT pg_catalog.has_function_privilege(
        'vec_gateway_personal_ejecutor',
        'vec_gateway_personal.cerrar_sesion_v1(text,timestamptz)', 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'ejecutor no tiene contrato nominal';
    END IF;
END
$prueba_acl$;

SET LOCAL ROLE vec_gateway_personal_ejecutor;

DO $prueba_sesion$
DECLARE
    t0 timestamptz := pg_catalog.statement_timestamp();
    estado record;
BEGIN
    SELECT * INTO estado FROM vec_gateway_personal.consultar_sesion_v1(
        repeat('9', 64), t0
    );
    IF estado.autenticada IS TRUE OR estado.expira_hasta IS NOT NULL
       OR estado.cuenta_actual_ref IS NOT NULL THEN
        RAISE EXCEPTION 'sesion ausente no falla cerrada';
    END IF;

    BEGIN
        PERFORM vec_gateway_personal.abrir_sesion_v1(
            repeat('3', 64), 'cta_abcdefghijklmnopqrstuv', repeat('4', 64),
            t0, t0 + interval '30 minutes 1 microsecond'
        );
        RAISE EXCEPTION 'apertura con expiracion excesiva aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    BEGIN
        PERFORM vec_gateway_personal.abrir_sesion_v1(
            repeat('5', 64), 'cta_abcdefghijklmnopqrstuv', repeat('6', 64),
            t0 - interval '5 minutes 1 microsecond', t0 + interval '1 minute'
        );
        RAISE EXCEPTION 'apertura con reloj pasado aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
    BEGIN
        PERFORM vec_gateway_personal.abrir_sesion_v1(
            repeat('7', 64), 'cta_abcdefghijklmnopqrstuv', repeat('8', 64),
            t0 + interval '5 minutes 1 microsecond', t0 + interval '6 minutes'
        );
        RAISE EXCEPTION 'apertura con reloj futuro aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;

    PERFORM vec_gateway_personal.abrir_sesion_v1(
        repeat('e', 64), 'cta_zyxwvutsrqponmlkjihgfe', repeat('f', 64),
        t0, t0 + interval '1 microsecond'
    );
    SELECT * INTO estado FROM vec_gateway_personal.consultar_sesion_v1(
        repeat('e', 64), t0 + interval '1 second'
    );
    IF estado.autenticada IS TRUE THEN
        RAISE EXCEPTION 'expiracion no se aplica';
    END IF;

    SELECT * INTO estado FROM vec_gateway_personal.abrir_sesion_v1(
        repeat('a', 64), 'cta_abcdefghijklmnopqrstuv', repeat('b', 64),
        t0, t0 + interval '10 minutes'
    );
    IF estado.autenticada IS NOT TRUE
       OR estado.expira_hasta <> t0 + interval '10 minutes'
       OR estado.cuenta_actual_ref <> 'cta_abcdefghijklmnopqrstuv' THEN
        RAISE EXCEPTION 'apertura invalida';
    END IF;

    SELECT * INTO estado FROM vec_gateway_personal.consultar_sesion_v1(
        repeat('a', 64), t0 + interval '1 minute'
    );
    IF estado.autenticada IS NOT TRUE THEN
        RAISE EXCEPTION 'sesion vigente no autenticada';
    END IF;

    SELECT * INTO estado FROM vec_gateway_personal.abrir_sesion_v1(
        repeat('c', 64), 'cta_abcdefghijklmnopqrstuv', repeat('d', 64),
        t0 + interval '2 minutes', t0 + interval '12 minutes'
    );
    IF estado.autenticada IS NOT TRUE THEN
        RAISE EXCEPTION 'rotacion no abrio la nueva sesion';
    END IF;
    SELECT * INTO estado FROM vec_gateway_personal.consultar_sesion_v1(
        repeat('a', 64), t0 + interval '2 minutes'
    );
    IF estado.autenticada IS TRUE THEN
        RAISE EXCEPTION 'rotacion no cerro la sesion anterior';
    END IF;

    SELECT * INTO estado FROM vec_gateway_personal.consultar_sesion_v1(
        repeat('c', 64), t0 + interval '12 minutes'
    );
    IF estado.autenticada IS TRUE THEN
        RAISE EXCEPTION 'expiracion no se aplica';
    END IF;

    SELECT * INTO estado FROM vec_gateway_personal.cerrar_sesion_v1(
        repeat('c', 64), t0 + interval '3 minutes'
    );
    IF estado.autenticada IS TRUE THEN
        RAISE EXCEPTION 'cierre no revoco';
    END IF;
    SELECT * INTO estado FROM vec_gateway_personal.cerrar_sesion_v1(
        repeat('c', 64), t0 + interval '4 minutes'
    );
    IF estado.autenticada IS TRUE THEN
        RAISE EXCEPTION 'cierre idempotente invalido';
    END IF;
    SELECT * INTO estado FROM vec_gateway_personal.consultar_sesion_v1(
        repeat('c', 64), t0 - interval '5 minutes 1 microsecond'
    );
    IF estado.autenticada IS TRUE OR estado.expira_hasta IS NOT NULL
       OR estado.cuenta_actual_ref IS NOT NULL THEN
        RAISE EXCEPTION 'consulta con reloj pasado no falla cerrada';
    END IF;
END
$prueba_sesion$;
ROLLBACK;
