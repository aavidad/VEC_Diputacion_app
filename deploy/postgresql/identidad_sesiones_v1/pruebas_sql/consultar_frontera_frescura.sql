DO $prueba$
DECLARE
    filas integer;
BEGIN
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_ssssssssssssssssssssssss',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('ad', 32), 'hex'),
          decode(repeat('ae', 32), 'hex'),
          decode(repeat('ac', 32), 'hex'),
          decode(repeat('ab', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('3', 64),
          date_trunc(
              'microseconds',
              statement_timestamp() - interval '15 minutes'
                  + interval '50 milliseconds'
          ),
          date_trunc('microseconds', statement_timestamp() - interval '1 second'),
          date_trunc('microseconds', statement_timestamp() + interval '4 minutes'),
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION
            'la frescura se evaluo antes de esperar por el bloqueo';
    END IF;
END
$prueba$;
