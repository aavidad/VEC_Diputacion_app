BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
DO $activar$
DECLARE
    solicitada timestamptz(6) :=
        date_trunc('microseconds', clock_timestamp());
    conocida timestamptz(6);
    orden_historico numeric(20, 0);
BEGIN
    INSERT INTO vec_autorizacion_atestada_v2.puntero_clave_capacidad(
        orden, clave_id, version, establecida_en, acto_ref
    ) VALUES (
        2, 'clave-capacidad:prueba:conocimiento:4', 4, solicitada,
        'acto:activar:clave:prueba:conocimiento:4'
    );
    SELECT registrada_en INTO STRICT conocida
      FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad
     WHERE orden = 2;
    SELECT max(orden) INTO orden_historico
      FROM vec_autorizacion_atestada_v2.puntero_clave_capacidad
     WHERE establecida_en <= solicitada + interval '500 milliseconds'
       AND registrada_en <= solicitada + interval '500 milliseconds';
    IF conocida <= solicitada + interval '1 second'
       OR orden_historico IS DISTINCT FROM 1 THEN
        RAISE EXCEPTION
            'el sello post-lock no preservo el orden de conocimiento';
    END IF;
END
$activar$;
ROLLBACK;
