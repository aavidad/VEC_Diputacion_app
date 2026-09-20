-- Sesion B: debe esperar a que A termine, y luego deja una unica sesion actual.
BEGIN;
SET LOCAL ROLE vec_gateway_personal_ejecutor;
SELECT * FROM vec_gateway_personal.abrir_sesion_v1(
    repeat('1', 64), 'cta_zyxwvutsrqponmlkjihgfe', repeat('2', 64),
    pg_catalog.statement_timestamp(),
    pg_catalog.statement_timestamp() + interval '10 minutes'
);
COMMIT;

DO $verificar$
BEGIN
    IF (SELECT count(*) FROM vec_gateway_personal.sesion
         WHERE cuenta_ref = 'cta_zyxwvutsrqponmlkjihgfe'
           AND revocada_en IS NULL) <> 1 THEN
        RAISE EXCEPTION 'concurrencia dejo mas de una sesion actual';
    END IF;
END
$verificar$;
