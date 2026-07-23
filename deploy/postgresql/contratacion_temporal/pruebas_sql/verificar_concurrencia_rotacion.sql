\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE, READ ONLY;

DO $prueba$
DECLARE
    v_raiz text;
    v_identidades bigint;
    v_alias_ambito bigint;
    v_alias_huella bigint;
BEGIN
    SELECT ambito_raiz_hmac
      INTO STRICT v_raiz
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE alias_hmac =
        'hmac-sha256:vec.contratacion-temporal.'
        || 'ambito-idempotencia/v2:' || repeat('7', 64);
    SELECT count(*) INTO v_identidades
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_raiz;
    SELECT count(*) INTO v_alias_ambito
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE ambito_raiz_hmac = v_raiz;
    SELECT count(*) INTO v_alias_huella
      FROM vec_contratacion_temporal.alias_huella_alta
     WHERE ambito_raiz_hmac = v_raiz;
    IF v_identidades <> 1
       OR v_alias_ambito <> 2
       OR v_alias_huella <> 2 THEN
        RAISE EXCEPTION
            'rotación concurrente incoherente: identidades %, ámbitos %, huellas %',
            v_identidades,
            v_alias_ambito,
            v_alias_huella;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM vec_contratacion_temporal.alias_ambito_alta
        WHERE ambito_raiz_hmac =
            'hmac-sha256:vec.contratacion-temporal.'
            || 'ambito-idempotencia/v1:' || repeat('d', 64)
          AND generacion = 2
          AND alias_hmac =
            'hmac-sha256:vec.contratacion-temporal.'
            || 'ambito-idempotencia/v2:' || repeat('a', 64)
    ) OR NOT EXISTS (
        SELECT 1
        FROM vec_contratacion_temporal.alias_huella_alta
        WHERE ambito_raiz_hmac =
            'hmac-sha256:vec.contratacion-temporal.'
            || 'ambito-idempotencia/v1:' || repeat('d', 64)
          AND generacion = 2
          AND alias_hmac =
            'hmac-sha256:vec.contratacion-temporal.'
            || 'huella-peticion/v2:' || repeat('c', 64)
    ) THEN
        RAISE EXCEPTION 'no se añadieron los alias v2 al canónico v1';
    END IF;
END
$prueba$;

COMMIT;
