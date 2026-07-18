DO $prueba$
DECLARE
    cuenta_ref text;
    cuenta_dos_ref text;
    cuenta_priv_ref text;
    sesion record;
    sesion_frontera_externa record;
    sesion_frontera_interna record;
    sesion_frontera_administracion record;
    control record;
    control_frontera record;
    filas integer;
    resultado boolean;
    revision text;
    autenticacion_verificada timestamptz(6);
    sesion_emitida timestamptz(6);
    sesion_expira timestamptz(6);
    limite_frescura timestamptz(6);
BEGIN
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_000000000000000000000000',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('00', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), false, NULL
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'provisionamiento acepto una huella HMAC nula';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_000000000000000000000001',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('22', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), false, NULL
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'provisionamiento acepto propositos HMAC iguales';
    END IF;

    SELECT provisionada.cuenta_ref INTO STRICT cuenta_ref
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_aaaaaaaaaaaaaaaaaaaaaaaa',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('11', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), false, NULL
      ) AS provisionada;
    IF cuenta_ref !~ '^cta_[A-Za-z0-9_-]{22,128}$' THEN
        RAISE EXCEPTION 'referencia de cuenta no CSPRNG';
    END IF;
    IF vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
           'opr_kkkkkkkkkkkkkkkkkkkkkkkk', cuenta_ref,
           'vec.identidad.hmac-sha256.v1',
           'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-rotada', 2,
           decode(repeat('aa', 32), 'hex'),
           decode(repeat('bb', 32), 'hex')
       ) IS DISTINCT FROM cuenta_ref THEN
        RAISE EXCEPTION 'la rotacion HMAC cambio la cta estable';
    END IF;
    IF vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
           'opr_000000000000000000000002', cuenta_ref,
           'vec.identidad.hmac-sha256.v1',
           'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-invalida', 3,
           decode(repeat('00', 32), 'hex'),
           decode(repeat('00', 32), 'hex')
       ) IS NOT NULL THEN
        RAISE EXCEPTION 'alias acepto huellas HMAC nulas o iguales';
    END IF;

    SELECT provisionada.cuenta_ref INTO STRICT cuenta_dos_ref
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_cccccccccccccccccccccccc',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('55', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), false, NULL
      ) AS provisionada;
    SELECT provisionada.cuenta_ref INTO STRICT cuenta_priv_ref
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_dddddddddddddddddddddddd',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('66', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), true,
          decode(repeat('11', 32), 'hex')
      ) AS provisionada;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_dddddddddddddddddddddddd',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('66', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), true,
          decode(repeat('55', 32), 'hex')
      );
    IF filas <> 0 OR cuenta_priv_ref = cuenta_ref
       OR cuenta_dos_ref = cuenta_ref THEN
        RAISE EXCEPTION 'replay privilegiado cambio la cuenta ordinaria';
    END IF;

    autenticacion_verificada :=
        date_trunc('microseconds', clock_timestamp() - interval '2 seconds');
    sesion_emitida :=
        date_trunc('microseconds', clock_timestamp() - interval '1 second');
    sesion_expira :=
        date_trunc('microseconds', clock_timestamp() + interval '4 minutes');

    SELECT * INTO STRICT sesion
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_bbbbbbbbbbbbbbbbbbbbbbbb',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('33', 32), 'hex'),
          decode(repeat('44', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('a', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF sesion.control_sesion_revision_texto <> '1'
       OR sesion.control_sesion_estado <> 'activa'
       OR sesion.cuenta_ref <> cuenta_ref
       OR sesion.cuenta_ordinaria_ref <> cuenta_ref
       OR sesion.sesion_revalidada_en < sesion_emitida
       OR sesion.sesion_valida_hasta <> sesion_expira THEN
        RAISE EXCEPTION 'confirmacion inicial invalida';
    END IF;

    -- Una invocacion nueva nunca recupera la confirmacion anterior por usar
    -- la misma asercion. La operacion distinta recibe cero filas.
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_eeeeeeeeeeeeeeeeeeeeeeee',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('33', 32), 'hex'),
          decode(repeat('77', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('a', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'replay de asercion aceptado';
    END IF;

    -- La huella del documento protegido es unica por dominio y no depende de
    -- kid/version: rotar la clave operacional no reabre la misma asercion.
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_llllllllllllllllllllllll',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-rotada', 2,
          decode(repeat('cc', 32), 'hex'),
          decode(repeat('dd', 32), 'hex'),
          decode(repeat('bb', 32), 'hex'),
          decode(repeat('aa', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('a', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'rotacion HMAC reabrio una asercion exacta';
    END IF;

    -- La barrera SQL no depende de que Go haya comprobado el TTL.
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_ffffffffffffffffffffffff',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('88', 32), 'hex'),
          decode(repeat('99', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('a', 64), autenticacion_verificada, sesion_emitida,
          sesion_emitida + interval '5 minutes 1 microsecond',
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'TTL superior a cinco minutos aceptado';
    END IF;

    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_111111111111111111111111',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('23', 32), 'hex'),
          decode(repeat('24', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('0', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'huella SHA-256 nula de autenticacion aceptada';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_222222222222222222222222',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('25', 32), 'hex'),
          decode(repeat('26', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('3', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('0', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'huella SHA-256 nula de politica aceptada';
    END IF;

    -- El rol registrador no puede degradar la garantia saltandose el contrato
    -- Go: externa >= sustancial; interna y administracion exactamente alto.
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_tttttttttttttttttttttttt',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('0b', 32), 'hex'), decode(repeat('0c', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('11', 32), 'hex'), NULL,
          false, 'externa_personal', 'certificado', 'bajo', repeat('4', 64),
          autenticacion_verificada, sesion_emitida, sesion_expira,
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'garantia baja aceptada en superficie externa';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_uuuuuuuuuuuuuuuuuuuuuuuu',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('0d', 32), 'hex'), decode(repeat('0e', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'sustancial',
          repeat('5', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'garantia sustancial aceptada en superficie interna';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_vvvvvvvvvvvvvvvvvvvvvvvv',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('0f', 32), 'hex'), decode(repeat('10', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('66', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), true,
          'administracion_privilegiada', 'kerberos_ad', 'sustancial',
          repeat('6', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION
            'garantia sustancial aceptada en administracion privilegiada';
    END IF;

    -- Las tres superficies nacen justo antes de su frontera y conservan TTL
    -- de sesion. Tras cruzarla, la revalidacion durable debe rechazarlas por
    -- frescura aunque su control siga activo. La espera activa evita pg_sleep.
    limite_frescura :=
        date_trunc('microseconds', clock_timestamp() + interval '2 seconds');
    SELECT * INTO STRICT sesion_frontera_externa
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_wwwwwwwwwwwwwwwwwwwwwwww',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('13', 32), 'hex'), decode(repeat('14', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('11', 32), 'hex'), NULL,
          false, 'externa_personal', 'certificado', 'sustancial', repeat('7', 64),
          limite_frescura - interval '12 hours', sesion_emitida, sesion_expira,
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    SELECT * INTO STRICT sesion_frontera_interna
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_xxxxxxxxxxxxxxxxxxxxxxxx',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('15', 32), 'hex'), decode(repeat('16', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto', repeat('8', 64),
          limite_frescura - interval '15 minutes', sesion_emitida, sesion_expira,
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    SELECT * INTO STRICT sesion_frontera_administracion
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_yyyyyyyyyyyyyyyyyyyyyyyy',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('17', 32), 'hex'), decode(repeat('18', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('66', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), true,
          'administracion_privilegiada', 'kerberos_ad', 'alto', repeat('9', 64),
          limite_frescura - interval '5 minutes', sesion_emitida, sesion_expira,
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    WHILE clock_timestamp() < limite_frescura LOOP
        PERFORM 1;
    END LOOP;
    filas := 0;
    FOR control_frontera IN
        SELECT base.*, estado.control_sesion_ref,
               estado.revision AS control_revision, estado.estado,
               estado.huella_sha256, estado.sesion_revalidada_en,
               estado.sesion_valida_hasta
          FROM vec_autorizacion.sesion_autenticacion_v1 AS base
          JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
            ON actual.sesion_ref = base.sesion_ref
          JOIN vec_autorizacion.control_sesion_v1 AS estado
            ON estado.sesion_ref = actual.sesion_ref
           AND estado.control_sesion_ref = actual.control_sesion_ref
           AND estado.revision = actual.revision
         WHERE base.sesion_ref IN (
             sesion_frontera_externa.sesion_ref,
             sesion_frontera_interna.sesion_ref,
             sesion_frontera_administracion.sesion_ref
         )
    LOOP
        filas := filas + 1;
        resultado :=
            vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
                control_frontera.autenticacion_ref,
                control_frontera.autenticacion_huella_sha256,
                control_frontera.asercion_ref, control_frontera.sesion_ref,
                control_frontera.cuenta_ref,
                control_frontera.cuenta_ordinaria_ref,
                control_frontera.cuenta_privilegiada,
                control_frontera.superficie,
                control_frontera.metodo_observado,
                control_frontera.garantia_observada,
                control_frontera.politica_garantia_ref,
                control_frontera.politica_garantia_huella_sha256,
                control_frontera.autenticacion_verificada_en,
                control_frontera.sesion_emitida_en,
                control_frontera.control_sesion_ref,
                control_frontera.control_revision::text,
                control_frontera.estado, control_frontera.huella_sha256,
                control_frontera.sesion_revalidada_en,
                control_frontera.sesion_valida_hasta
            );
        IF resultado IS NOT FALSE THEN
            RAISE EXCEPTION
                'revalidacion acepto frescura vencida en %',
                control_frontera.superficie;
        END IF;
    END LOOP;
    IF filas <> 3 THEN
        RAISE EXCEPTION 'no se probaron las tres fronteras de revalidacion';
    END IF;

    -- Frescura reforzada, evaluada con clock_timestamp() tras los bloqueos:
    -- externa 12h, interna 15m y administracion 5m, todas half-open.
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_mmmmmmmmmmmmmmmmmmmmmmmm',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('01', 32), 'hex'), decode(repeat('02', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('11', 32), 'hex'), NULL,
          false, 'externa_personal', 'certificado', 'alto', repeat('d', 64),
          date_trunc('microseconds', clock_timestamp() - interval '11 hours'),
          date_trunc('microseconds', clock_timestamp() - interval '1 second'),
          date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 1 THEN
        RAISE EXCEPTION 'frescura externa valida rechazada';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_nnnnnnnnnnnnnnnnnnnnnnnn',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('03', 32), 'hex'), decode(repeat('04', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('11', 32), 'hex'), NULL,
          false, 'externa_personal', 'certificado', 'alto', repeat('e', 64),
          date_trunc('microseconds', clock_timestamp() - interval '12 hours'),
          date_trunc('microseconds', clock_timestamp() - interval '1 second'),
          date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'frontera half-open externa aceptada';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_oooooooooooooooooooooooo',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('05', 32), 'hex'), decode(repeat('06', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto', repeat('f', 64),
          date_trunc('microseconds', clock_timestamp() - interval '15 minutes'),
          date_trunc('microseconds', clock_timestamp() - interval '1 second'),
          date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'frontera half-open interna aceptada';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_pppppppppppppppppppppppp',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('07', 32), 'hex'), decode(repeat('08', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('66', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), true,
          'administracion_privilegiada', 'kerberos_ad', 'alto', repeat('1', 64),
          date_trunc('microseconds', clock_timestamp() - interval '4 minutes'),
          date_trunc('microseconds', clock_timestamp() - interval '1 second'),
          date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 1 THEN
        RAISE EXCEPTION 'frescura administrativa valida rechazada';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_qqqqqqqqqqqqqqqqqqqqqqqq',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('09', 32), 'hex'), decode(repeat('0a', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), decode(repeat('66', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), true,
          'administracion_privilegiada', 'kerberos_ad', 'alto', repeat('2', 64),
          date_trunc('microseconds', clock_timestamp() - interval '5 minutes'),
          date_trunc('microseconds', clock_timestamp() - interval '1 second'),
          date_trunc('microseconds', clock_timestamp() + interval '4 minutes'),
          'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'frontera half-open administrativa aceptada';
    END IF;

    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
          'opr_bbbbbbbbbbbbbbbbbbbbbbbb',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('33', 32), 'hex'),
          decode(repeat('44', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('a', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 1 THEN
        RAISE EXCEPTION 'operacion comprometida no reconciliada';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
          'opr_bbbbbbbbbbbbbbbbbbbbbbbb',
          'vec.identidad.hmac-sha256.v1',
          'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
          decode(repeat('33', 32), 'hex'),
          decode(repeat('44', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'kerberos_ad', 'alto',
          repeat('c', 64), autenticacion_verificada, sesion_emitida,
          sesion_expira, 'pga_aaaaaaaaaaaaaaaaaaaaaaaa', repeat('b', 64)
      );
    IF filas <> 0 THEN
        RAISE EXCEPTION 'reconciliacion no cotejo todos los campos';
    END IF;

    SELECT base.*, estado.control_sesion_ref,
           estado.revision AS control_revision, estado.estado,
           estado.huella_sha256, estado.sesion_revalidada_en,
           estado.sesion_valida_hasta
      INTO STRICT control
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = base.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS estado
        ON estado.sesion_ref = actual.sesion_ref
       AND estado.control_sesion_ref = actual.control_sesion_ref
       AND estado.revision = actual.revision
     WHERE base.sesion_ref = sesion.sesion_ref;
    resultado := vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
        control.autenticacion_ref, control.autenticacion_huella_sha256,
        control.asercion_ref, control.sesion_ref, control.cuenta_ref,
        control.cuenta_ordinaria_ref, control.cuenta_privilegiada,
        control.superficie, control.metodo_observado,
        control.garantia_observada, control.politica_garantia_ref,
        control.politica_garantia_huella_sha256,
        control.autenticacion_verificada_en, control.sesion_emitida_en,
        control.control_sesion_ref, control.control_revision::text,
        control.estado, control.huella_sha256,
        control.sesion_revalidada_en, control.sesion_valida_hasta
    );
    IF resultado IS NOT TRUE THEN
        RAISE EXCEPTION 'sesion activa no revalidada';
    END IF;

    revision := vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
        cuenta_ref, '1', 'inactiva', 'opr_gggggggggggggggggggggggg'
    );
    IF revision <> '2' THEN
        RAISE EXCEPTION 'cuenta no inactivada';
    END IF;
    revision := vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
        cuenta_ref, '2', 'activa', 'opr_hhhhhhhhhhhhhhhhhhhhhhhh'
    );
    IF revision <> '3' THEN
        RAISE EXCEPTION 'cuenta no reactivada';
    END IF;
    resultado := vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
        control.autenticacion_ref, control.autenticacion_huella_sha256,
        control.asercion_ref, control.sesion_ref, control.cuenta_ref,
        control.cuenta_ordinaria_ref, control.cuenta_privilegiada,
        control.superficie, control.metodo_observado,
        control.garantia_observada, control.politica_garantia_ref,
        control.politica_garantia_huella_sha256,
        control.autenticacion_verificada_en, control.sesion_emitida_en,
        control.control_sesion_ref, control.control_revision::text,
        control.estado, control.huella_sha256,
        control.sesion_revalidada_en, control.sesion_valida_hasta
    );
    IF resultado IS NOT FALSE THEN
        RAISE EXCEPTION 'reactivacion resucito una sesion anterior';
    END IF;

    revision := vec_identidad_sesiones_v1.revocar_sesion_v1(
        control.sesion_ref, control.control_sesion_ref,
        control.control_revision::text, 'opr_iiiiiiiiiiiiiiiiiiiiiiii'
    );
    IF revision <> '2' THEN
        RAISE EXCEPTION 'sesion activa no revocada';
    END IF;
END
$prueba$;
