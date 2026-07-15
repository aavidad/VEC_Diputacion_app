-- Reversion destructiva. Requiere copia verificada y aprobacion operativa.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
DROP SCHEMA vec_autorizacion CASCADE;
COMMIT;
