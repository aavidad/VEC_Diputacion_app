-- Destruccion deliberada del almacen inmutable. Con historia exige una
-- confirmacion operativa explicita y el expediente de retencion aplicable.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL row_security = off;

DO $prevalidacion$
BEGIN
    -- FORCE RLS impide que el propietario vea la historia, por lo que una
    -- retirada destructiva solo puede inventariarla bajo una identidad DBA
    -- que realmente eluda RLS. Nunca se cambia al propietario antes del conteo.
    IF NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = current_user AND rolsuper
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'down rechazado: requiere superusuario para inventariar RLS';
    END IF;
    IF to_regnamespace('vec_bolsa_calculo_experiencia') IS NULL
       OR to_regclass(
           'vec_bolsa_calculo_experiencia.resultado_oficial'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta el almacen esperado';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_bolsa_calculo_experiencia.resultado_oficial
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_calculo_experiencia.intento IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_calculo_experiencia.consumo_autorizaciones
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_calculo_experiencia.recibo IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_calculo_experiencia.auditoria IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_calculo_experiencia.outbox IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_calculo_experiencia.configuracion_tenant
    IN ACCESS EXCLUSIVE MODE;

DO $confirmar_historia$
BEGIN
    IF (
        EXISTS (SELECT 1 FROM vec_bolsa_calculo_experiencia.resultado_oficial)
        OR EXISTS (SELECT 1 FROM vec_bolsa_calculo_experiencia.intento)
        OR EXISTS (
            SELECT 1 FROM vec_bolsa_calculo_experiencia.consumo_autorizaciones
        )
        OR EXISTS (SELECT 1 FROM vec_bolsa_calculo_experiencia.recibo)
        OR EXISTS (SELECT 1 FROM vec_bolsa_calculo_experiencia.auditoria)
        OR EXISTS (SELECT 1 FROM vec_bolsa_calculo_experiencia.outbox)
    ) AND current_setting(
        'vec.confirmar_destruccion_calculo_experiencia', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_CALCULO_EXPERIENCIA_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe historia durable',
            HINT = 'tramite la destruccion formal y use la confirmacion explicita';
    END IF;
END
$confirmar_historia$;

-- Los disparadores de truncado se retiran solo despues de superar la barrera
-- explicita de destruccion y manteniendo bloqueos exclusivos sobre la historia.
DO $retirar_barrera_truncado$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'resultado_oficial', 'intento', 'consumo_autorizaciones',
        'recibo', 'auditoria', 'outbox', 'configuracion_tenant'
    ] LOOP
        EXECUTE format(
            'DROP TRIGGER impedir_truncado ON %I.%I',
            'vec_bolsa_calculo_experiencia', tabla
        );
    END LOOP;
END
$retirar_barrera_truncado$;

DROP TABLE
    vec_bolsa_calculo_experiencia.outbox,
    vec_bolsa_calculo_experiencia.auditoria,
    vec_bolsa_calculo_experiencia.recibo,
    vec_bolsa_calculo_experiencia.consumo_autorizaciones,
    vec_bolsa_calculo_experiencia.intento,
    vec_bolsa_calculo_experiencia.resultado_oficial,
    vec_bolsa_calculo_experiencia.configuracion_tenant;

DROP FUNCTION
    vec_bolsa_calculo_experiencia.validar_autorizaciones_del_intento();
DROP FUNCTION vec_bolsa_calculo_experiencia.validar_predecesor_resultado();
DROP FUNCTION
    vec_bolsa_calculo_experiencia.validar_encadenamiento_auditoria();
DROP FUNCTION vec_bolsa_calculo_experiencia.rechazar_mutacion_inmutable();
DROP DOMAIN vec_bolsa_calculo_experiencia.instante_utc;
DROP FUNCTION vec_bolsa_calculo_experiencia.instante_utc_valido(text);
DROP FUNCTION
    vec_bolsa_calculo_experiencia.indice_hmac_sha256_valido(text);
DROP FUNCTION vec_bolsa_calculo_experiencia.sujeto_hmac_ref_valido(text);
DROP FUNCTION vec_bolsa_calculo_experiencia.huella_sha256_valida(text);
DROP FUNCTION
    vec_bolsa_calculo_experiencia.texto_opaco_valido(text, integer);

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_calculo_experiencia_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_calculo_experiencia_propietario
    GRANT USAGE ON TYPES TO PUBLIC;
DROP SCHEMA vec_bolsa_calculo_experiencia;
COMMIT;
