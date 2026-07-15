-- Ejecutar solo despues de 000001_autorizacion.down.sql y de retirar todas las
-- membresias LOGIN. DROP ROLE falla cerrado si queda cualquier dependencia.
BEGIN;

DO $bloque$
BEGIN
    EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_registro', current_database());
    EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_fuente', current_database());
    EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_migrador', current_database());
    EXECUTE format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_propietario', current_database());
END
$bloque$;

REVOKE vec_autorizacion_propietario FROM vec_autorizacion_migrador;
DROP ROLE vec_autorizacion_registro;
DROP ROLE vec_autorizacion_fuente;
DROP ROLE vec_autorizacion_migrador;
DROP ROLE vec_autorizacion_propietario;

COMMIT;
