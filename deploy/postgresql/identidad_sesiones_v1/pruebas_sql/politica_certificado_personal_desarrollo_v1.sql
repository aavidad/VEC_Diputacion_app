-- Prueba focal sobre PostgreSQL 18 con roles y migraciones 000001–000006.
-- Datos sintéticos; la política se inserta solo en esta base desechable.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE
    ref_politica constant text := 'pga_sintetica_personal_desarrollo_20260924';
    huella constant text := repeat('a', 64);
    verificada timestamptz(6);
    retirada timestamptz(6);
    sesion record;
    cuenta text;
    filas integer;
BEGIN
    IF vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
        ref_politica, huella, clock_timestamp()) THEN
        RAISE EXCEPTION 'se aceptó una política ausente';
    END IF;
    IF has_table_privilege(
        'vec_personal_propietario',
        'vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1',
        'SELECT,INSERT,UPDATE,DELETE'
    ) THEN
        RAISE EXCEPTION 'Personal recibió acceso directo a la política';
    END IF;
    INSERT INTO vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
        (singleton, politica_ref, huella_sha256, retirar_en, activa)
    VALUES (true, ref_politica, huella, clock_timestamp() + interval '1 day', true);
    SELECT retirar_en INTO STRICT retirada
      FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
     WHERE singleton;
    IF NOT vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(
        ref_politica, huella, retirada)
       OR vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(
        ref_politica, huella, retirada + interval '1 microsecond')
       OR vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(
        ref_politica, repeat('b', 64), retirada) THEN
        RAISE EXCEPTION 'el preflight aceptó política divergente';
    END IF;

    verificada := clock_timestamp();
    IF NOT vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
        ref_politica, huella, verificada)
       OR vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
        'pga_sintetica_personal_desarrollo_distinta', huella, verificada)
       OR vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
        ref_politica, repeat('b', 64), verificada) THEN
        RAISE EXCEPTION 'cotejo de ref/huella falló';
    END IF;

    SELECT p.cuenta_ref INTO STRICT cuenta
      FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
          'opr_policy_provision_00000001',
          'vec.identidad.hmac-sha256.v1',
          'idh_policy_personal_00000001', 'clave-sintetica', 1,
          decode(repeat('11', 32), 'hex'),
          decode(repeat('22', 32), 'hex'), false, NULL
      ) AS p;
    SELECT s.* INTO STRICT sesion
      FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
          'opr_policy_session_000000001',
          'vec.identidad.hmac-sha256.v1',
          'idh_policy_personal_00000001', 'clave-sintetica', 1,
          decode(repeat('33', 32), 'hex'),
          decode(repeat('44', 32), 'hex'),
          decode(repeat('22', 32), 'hex'),
          decode(repeat('11', 32), 'hex'), NULL,
          false, 'interna_corporativa', 'certificado', 'sustancial',
          repeat('c', 64), verificada, verificada,
          verificada + interval '4 minutes', ref_politica, huella
      ) AS s;
    IF sesion.cuenta_ref <> cuenta THEN
        RAISE EXCEPTION 'la sesión no resolvió la cuenta';
    END IF;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
          sesion.autenticacion_ref, sesion.sesion_ref
      );
    IF filas <> 1 THEN
        RAISE EXCEPTION 'sesión sintética vigente no revalidada';
    END IF;

    UPDATE vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
       SET activa = false WHERE singleton;
    SELECT count(*) INTO filas
      FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
          sesion.autenticacion_ref, sesion.sesion_ref
      );
    IF filas <> 0 OR
       vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(
           ref_politica, huella, retirada) OR
       vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
           ref_politica, huella, verificada) THEN
        RAISE EXCEPTION 'la retirada no cerró la sesión';
    END IF;
END $prueba$;
COMMIT;
