-- Ejercita la identidad LOGIN del pool de verificacion. No hay SELECT directo
-- ni capacidad A/B; una segunda membresia invalida incluso la unica funcion
-- concedida.
SET SESSION AUTHORIZATION vec_convocatorias_verificador_prueba;

DO $minimo_privilegio$
DECLARE
    referencia text := 'recibo-borrador-' || repeat('0', 64);
BEGIN
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.verificar_recibo_borrador_v1(
              referencia, 'transaccion-inexistente', repeat('1', 64)
          );
        RAISE EXCEPTION 'el verificador invento un recibo inexistente';
    EXCEPTION WHEN no_data_found THEN
        IF SQLERRM <> 'recibo durable inexistente o no coincidente' THEN
            RAISE;
        END IF;
    END;
    BEGIN
        PERFORM count(*)
          FROM vec_bolsa_convocatorias.acreditacion_kms_borrador;
        RAISE EXCEPTION 'el verificador leyo la tabla de acreditaciones';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.presupuesto_confirmacion_kms_borrador_v1();
        RAISE EXCEPTION 'el verificador leyo el presupuesto interno';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(
              NULL::jsonb, NULL::jsonb, NULL::jsonb,
              NULL::bytea, NULL::bytea, NULL::bytea, NULL::bytea,
              NULL::bytea, NULL::bytea, NULL::bytea, NULL::bytea
          );
        RAISE EXCEPTION 'el verificador pudo preparar una confirmacion';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.confirmar_borrador_v1(
              referencia, '{}'::jsonb, ''::bytea
          );
        RAISE EXCEPTION 'el verificador pudo ejecutar la fase B';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END
$minimo_privilegio$;

RESET SESSION AUTHORIZATION;
GRANT vec_bolsa_convocatorias_proyector_gobierno
    TO vec_convocatorias_verificador_prueba;
SET SESSION AUTHORIZATION vec_convocatorias_verificador_prueba;

DO $membresia_exclusiva$
DECLARE
    estado text;
    mensaje text;
BEGIN
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.verificar_recibo_borrador_v1(
              'recibo-borrador-' || repeat('0', 64),
              'transaccion-inexistente', repeat('1', 64)
          );
        RAISE EXCEPTION 'una identidad con doble rol verifico recibos';
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS estado = RETURNED_SQLSTATE,
            mensaje = MESSAGE_TEXT;
        IF estado <> '42501'
           OR mensaje <> 'verificacion durable de recibo no autorizada' THEN
            RAISE EXCEPTION
                'rechazo de doble membresia inestable: SQLSTATE=%, mensaje=%',
                estado, mensaje;
        END IF;
    END;
END
$membresia_exclusiva$;

RESET SESSION AUTHORIZATION;
REVOKE vec_bolsa_convocatorias_proyector_gobierno
    FROM vec_convocatorias_verificador_prueba;
