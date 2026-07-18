-- Capacidad minima del propietario de identidad sobre la unica fuente de
-- sesion/control existente. No se duplican estas tablas en otro esquema.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_autorizacion.sesion_autenticacion_v1') IS NULL
       OR to_regclass('vec_autorizacion.control_sesion_v1') IS NULL
       OR to_regclass('vec_autorizacion.control_sesion_actual_v1') IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_identidad_sesiones_v1_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       ) OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_policies
            WHERE schemaname = 'vec_autorizacion'
              AND policyname LIKE 'acceso_identidad_sesiones_v1_%'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para capacidad de identidad V1';
    END IF;
END
$prevalidacion$;

GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_identidad_sesiones_v1_propietario;
GRANT SELECT, INSERT, REFERENCES
    ON vec_autorizacion.sesion_autenticacion_v1
    TO vec_identidad_sesiones_v1_propietario;
GRANT SELECT, INSERT, REFERENCES
    ON vec_autorizacion.control_sesion_v1
    TO vec_identidad_sesiones_v1_propietario;
GRANT SELECT, INSERT, UPDATE
    ON vec_autorizacion.control_sesion_actual_v1
    TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.texto_positivo_valido(text, integer)
    TO vec_identidad_sesiones_v1_propietario;

-- La PK de sesion ya hace unico el conjunto. La restriccion nominal permite
-- una FK que demuestre que consumo_asercion no cruza cuentas o autenticacion.
ALTER TABLE vec_autorizacion.sesion_autenticacion_v1
    ADD CONSTRAINT sesion_autenticacion_identidad_compuesta_v1 UNIQUE (
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        cuenta_ref, cuenta_ordinaria_ref
    );

CREATE POLICY acceso_identidad_sesiones_v1_lectura
    ON vec_autorizacion.sesion_autenticacion_v1
    FOR SELECT TO vec_identidad_sesiones_v1_propietario
    USING (current_user = 'vec_identidad_sesiones_v1_propietario');
CREATE POLICY acceso_identidad_sesiones_v1_alta
    ON vec_autorizacion.sesion_autenticacion_v1
    FOR INSERT TO vec_identidad_sesiones_v1_propietario
    WITH CHECK (current_user = 'vec_identidad_sesiones_v1_propietario');

CREATE POLICY acceso_identidad_sesiones_v1_control_lectura
    ON vec_autorizacion.control_sesion_v1
    FOR SELECT TO vec_identidad_sesiones_v1_propietario
    USING (current_user = 'vec_identidad_sesiones_v1_propietario');
CREATE POLICY acceso_identidad_sesiones_v1_control_alta
    ON vec_autorizacion.control_sesion_v1
    FOR INSERT TO vec_identidad_sesiones_v1_propietario
    WITH CHECK (current_user = 'vec_identidad_sesiones_v1_propietario');

CREATE POLICY acceso_identidad_sesiones_v1_actual_lectura
    ON vec_autorizacion.control_sesion_actual_v1
    FOR SELECT TO vec_identidad_sesiones_v1_propietario
    USING (current_user = 'vec_identidad_sesiones_v1_propietario');
CREATE POLICY acceso_identidad_sesiones_v1_actual_alta
    ON vec_autorizacion.control_sesion_actual_v1
    FOR INSERT TO vec_identidad_sesiones_v1_propietario
    WITH CHECK (current_user = 'vec_identidad_sesiones_v1_propietario');
CREATE POLICY acceso_identidad_sesiones_v1_actual_avance
    ON vec_autorizacion.control_sesion_actual_v1
    FOR UPDATE TO vec_identidad_sesiones_v1_propietario
    USING (current_user = 'vec_identidad_sesiones_v1_propietario')
    WITH CHECK (current_user = 'vec_identidad_sesiones_v1_propietario');
COMMIT;
