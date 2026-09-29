-- Ensayo local transaccional: el mismo recibo real ante LOGIN aprobado e interno.
BEGIN;
SET LOCAL search_path = pg_catalog;
CREATE ROLE vec_test_aut_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_test_aut_extra NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_test_login_externo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_test_login_interno LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_test_aut_externo TO vec_test_login_externo
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;

SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SELECT vec_identidad_externa_v1.huella_material_alta_v1(
    'cta_abcdefghij0123456789kl','vec.identidad.hmac-sha256.v1',
    'idh_abcdefghij0123456789kl','clave-prueba',1,
    decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex')) AS material,
    vec_identidad_externa_v1.huella_ausencia_v1(
    'cta_abcdefghij0123456789kl') AS preimagen \gset
INSERT INTO vec_identidad_externa_v1.aprobacion_alta(
    aprobacion_ref,huella_material_sha256,huella_preimagen_sha256,revision_esperada)
VALUES ('apr_abcdefghij0123456789kl',:'material',:'preimagen',0);
SELECT vec_identidad_externa_v1.confirmar_alta_v1(
    'opr_abcdefghij0123456789aa','apr_abcdefghij0123456789kl',
    'cta_abcdefghij0123456789kl','vec.identidad.hmac-sha256.v1',
    'idh_abcdefghij0123456789kl','clave-prueba',1,
    decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),
    :'material',:'preimagen',0) AS cuenta_confirmada \gset
SELECT clock_timestamp()-interval '1 second' AS emitida,
       clock_timestamp()+interval '2 minutes' AS expira \gset
SELECT * FROM vec_identidad_externa_v1.registrar_sesion_v1(
    'opr_abcdefghij0123456789bb','vec.identidad.hmac-sha256.v1',
    'idh_abcdefghij0123456789kl','clave-prueba',1,
    decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),
    decode(repeat('22',32),'hex'),decode(repeat('11',32),'hex'),
    NULL,false,'externa_personal','certificado','alto',repeat('a',64),
    :'emitida'::timestamptz,:'emitida'::timestamptz,
    :'expira'::timestamptz,'pga_abcdefghij0123456789kl',repeat('b',64)) \gset
INSERT INTO vec_identidad_externa_v1.llamante_autorizacion(
    login,grupo_esperado,aprobacion_ref)
VALUES ('vec_test_login_externo','vec_test_aut_externo',
        'apr_abcdefghij0123456789zz');
RESET ROLE;

CREATE TEMP TABLE vec_capsula_prueba(payload jsonb);
INSERT INTO vec_capsula_prueba VALUES (jsonb_build_object(
    'aut',:'autenticacion_ref','ase',:'asercion_ref','ses',:'sesion_ref',
    'ctl',:'control_sesion_ref','rev',:'control_sesion_revision_texto',
    'estado',:'control_sesion_estado','huella',:'control_sesion_huella_sha256',
    'cuenta',:'cuenta_ref','revalidada',:'sesion_revalidada_en',
    'hasta',:'sesion_valida_hasta','emitida',:'emitida'));
GRANT SELECT ON vec_capsula_prueba TO vec_autorizacion_propietario;

CREATE FUNCTION pg_temp.probar_fachada_externa(p_cuenta text)
RETURNS boolean LANGUAGE sql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
    SELECT vec_identidad_externa_v1.acreditar_sesion_externa_v1(
        p->>'aut',repeat('a',64),p->>'ase',p->>'ses',
        COALESCE(p_cuenta,p->>'cuenta'),p->>'cuenta',false,
        'externa_personal','certificado','alto',
        'pga_abcdefghij0123456789kl',repeat('b',64),
        (p->>'emitida')::timestamptz,(p->>'emitida')::timestamptz,
        p->>'ctl',p->>'rev',p->>'estado',p->>'huella',
        (p->>'revalidada')::timestamptz,(p->>'hasta')::timestamptz)
    FROM pg_temp.vec_capsula_prueba AS c(p)
$f$;
ALTER FUNCTION pg_temp.probar_fachada_externa(text)
    OWNER TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION pg_temp.probar_fachada_externa(text)
    TO vec_test_login_externo,vec_test_login_interno;

SET SESSION AUTHORIZATION vec_test_login_externo;
DO $prueba$ BEGIN
    IF pg_temp.probar_fachada_externa(NULL) IS NOT TRUE THEN
        RAISE EXCEPTION 'LOGIN externo aprobado no acredito sesion real';
    END IF;
    IF pg_temp.probar_fachada_externa('cta_otra123456789abcdefghijkl') IS NOT FALSE THEN
        RAISE EXCEPTION 'material de otra cuenta fue aceptado';
    END IF;
END $prueba$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_test_login_interno;
DO $prueba$ BEGIN
    IF pg_temp.probar_fachada_externa(NULL) IS NOT FALSE THEN
        RAISE EXCEPTION 'LOGIN interno sono sesion externa';
    END IF;
END $prueba$;
RESET SESSION AUTHORIZATION;

GRANT vec_test_aut_extra TO vec_test_aut_externo
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_test_login_externo;
DO $prueba$ BEGIN
    IF pg_temp.probar_fachada_externa(NULL) IS NOT FALSE THEN
        RAISE EXCEPTION 'membresia transitiva amplificada aceptada';
    END IF;
END $prueba$;
RESET SESSION AUTHORIZATION;
REVOKE vec_test_aut_extra FROM vec_test_aut_externo;

GRANT vec_test_aut_externo TO vec_test_login_externo
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
SET SESSION AUTHORIZATION vec_test_login_externo;
DO $prueba$ BEGIN
    IF pg_temp.probar_fachada_externa(NULL) IS NOT FALSE THEN
        RAISE EXCEPTION 'SET ROLE ampliado aceptado';
    END IF;
END $prueba$;
RESET SESSION AUTHORIZATION;
REVOKE vec_test_aut_externo FROM vec_test_login_externo;
GRANT vec_test_aut_externo TO vec_test_login_externo
    WITH ADMIN TRUE, INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION vec_test_login_externo;
DO $prueba$ BEGIN
    IF pg_temp.probar_fachada_externa(NULL) IS NOT FALSE THEN
        RAISE EXCEPTION 'ADMIN ampliado aceptado';
    END IF;
END $prueba$;
RESET SESSION AUTHORIZATION;
REVOKE vec_test_aut_externo FROM vec_test_login_externo;
GRANT vec_test_aut_externo TO vec_test_login_externo
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
ALTER ROLE vec_test_login_externo CREATEROLE;
SET SESSION AUTHORIZATION vec_test_login_externo;
DO $prueba$ BEGIN
    IF pg_temp.probar_fachada_externa(NULL) IS NOT FALSE THEN
        RAISE EXCEPTION 'LOGIN con CREATEROLE aceptado';
    END IF;
END $prueba$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
