BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;

INSERT INTO vec_bolsa_llamamientos.bolsa_autoritativa VALUES (
    'bolsa-concurrencia', 1,
    encode(sha256(convert_to('{}', 'UTF8')), 'hex'),
    convert_to('{}', 'UTF8'), 'categoria-concurrencia',
    clock_timestamp() - interval '1 day', NULL, 'vigente', clock_timestamp()
);
INSERT INTO vec_bolsa_llamamientos.necesidad_autoritativa VALUES (
    'necesidad-concurrencia', 1,
    encode(sha256(convert_to('{"n":1}', 'UTF8')), 'hex'),
    convert_to('{"n":1}', 'UTF8'), 'bolsa-concurrencia', 1,
    encode(sha256(convert_to('{}', 'UTF8')), 'hex'),
    'categoria-concurrencia', 'unidad-concurrencia',
    clock_timestamp() + interval '1 day', clock_timestamp()
);
INSERT INTO vec_bolsa_llamamientos.necesidad_actual VALUES (
    'necesidad-concurrencia', 1,
    encode(sha256(convert_to('{"n":1}', 'UTF8')), 'hex'),
    'abierta', clock_timestamp()
);
INSERT INTO vec_bolsa_llamamientos.instantanea_autoritativa VALUES
    ('instantanea-concurrencia-1', 1,
     encode(sha256(convert_to('{"i":1}', 'UTF8')), 'hex'),
     convert_to('{"i":1}', 'UTF8'), 'bolsa-concurrencia', 1,
     encode(sha256(convert_to('{}', 'UTF8')), 'hex'), 1,
     clock_timestamp(), clock_timestamp(), clock_timestamp()),
    ('instantanea-concurrencia-2', 1,
     encode(sha256(convert_to('{"i":2}', 'UTF8')), 'hex'),
     convert_to('{"i":2}', 'UTF8'), 'bolsa-concurrencia', 1,
     encode(sha256(convert_to('{}', 'UTF8')), 'hex'), 1,
     clock_timestamp(), clock_timestamp(), clock_timestamp());
COMMIT;
