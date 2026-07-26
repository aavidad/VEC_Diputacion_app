\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL TRANSACTION ISOLATION LEVEL SERIALIZABLE, READ ONLY;

DO $prueba$
DECLARE
    v_ambito text :=
        'hmac-sha256:vec.contratacion-temporal.'
        || 'ambito-idempotencia/v1:'
        || repeat('e', 64);
    v_identidades bigint;
    v_versiones bigint;
    v_actuales bigint;
    v_identidad record;
BEGIN
    SELECT count(*)
     INTO STRICT v_identidades
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_ambito;
    SELECT count(*)
      INTO STRICT v_versiones
      FROM vec_contratacion_temporal.reserva_alta_version
     WHERE ambito_hmac = v_ambito;
    SELECT count(*)
      INTO STRICT v_actuales
      FROM vec_contratacion_temporal.reserva_alta_actual
     WHERE ambito_hmac = v_ambito;

    IF v_identidades <> 1 OR v_versiones <> 1 OR v_actuales <> 1 THEN
        RAISE EXCEPTION
            'concurrencia incoherente: identidades %, versiones %, actuales %',
            v_identidades,
            v_versiones,
            v_actuales;
    END IF;

    SELECT reserva_ref, expediente_ref, numero_visible, recibo_ref
      INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_ambito;

    IF v_identidad.reserva_ref
           !~ '^reserva:concurrencia-[0-7]$'
       OR v_identidad.expediente_ref
           !~ '^expediente:concurrencia-[0-7]$'
       OR v_identidad.numero_visible
           !~ '^2026/CONC-[0-7]$'
       OR v_identidad.recibo_ref
           !~ '^recibo:concurrencia-[0-7]$' THEN
        RAISE EXCEPTION
            'referencias ganadoras incoherentes: %',
            row_to_json(v_identidad);
    END IF;
END
$prueba$;

COMMIT;
