BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_identidad_sesiones_v1') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de capacidad rechazado: identidad sigue instalada';
    END IF;
END
$prevalidacion$;

DROP POLICY acceso_identidad_sesiones_v1_actual_avance
    ON vec_autorizacion.control_sesion_actual_v1;
DROP POLICY acceso_identidad_sesiones_v1_actual_alta
    ON vec_autorizacion.control_sesion_actual_v1;
DROP POLICY acceso_identidad_sesiones_v1_actual_lectura
    ON vec_autorizacion.control_sesion_actual_v1;
DROP POLICY acceso_identidad_sesiones_v1_control_alta
    ON vec_autorizacion.control_sesion_v1;
DROP POLICY acceso_identidad_sesiones_v1_control_lectura
    ON vec_autorizacion.control_sesion_v1;
DROP POLICY acceso_identidad_sesiones_v1_alta
    ON vec_autorizacion.sesion_autenticacion_v1;
DROP POLICY acceso_identidad_sesiones_v1_lectura
    ON vec_autorizacion.sesion_autenticacion_v1;

ALTER TABLE vec_autorizacion.sesion_autenticacion_v1
    DROP CONSTRAINT sesion_autenticacion_identidad_compuesta_v1;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.texto_positivo_valido(text, integer)
    FROM vec_identidad_sesiones_v1_propietario;

REVOKE ALL ON vec_autorizacion.control_sesion_actual_v1
    FROM vec_identidad_sesiones_v1_propietario;
REVOKE ALL ON vec_autorizacion.control_sesion_v1
    FROM vec_identidad_sesiones_v1_propietario;
REVOKE ALL ON vec_autorizacion.sesion_autenticacion_v1
    FROM vec_identidad_sesiones_v1_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_identidad_sesiones_v1_propietario;
COMMIT;
