BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
    p_autenticacion_ref text,
    p_sesion_ref text
)
RETURNS TABLE(
    autenticacion_ref text,
    autenticacion_huella_sha256 text,
    asercion_ref text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision text,
    control_sesion_huella_sha256 text,
    cuenta_ref text,
    cuenta_ordinaria_ref text,
    cuenta_privilegiada boolean,
    superficie text,
    metodo_observado text,
    garantia_observada text,
    politica_garantia_ref text,
    politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz,
    sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz,
    sesion_revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    sesion record;
    estado_bloqueado record;
    cuentas_bloqueadas integer := 0;
    cuentas_esperadas integer;
    ahora timestamptz(6);
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(
           p_autenticacion_ref, 'aut_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           p_sesion_ref, 'ses_'
       ) IS NOT TRUE THEN
        RETURN;
    END IF;

    -- El puntero de control se bloquea antes que las cuentas. Revocacion y
    -- revalidacion siguen asi el mismo orden global y no observan una revision
    -- historica como si fuera la actual.
    SELECT base.autenticacion_ref,
           base.autenticacion_huella_sha256,
           base.asercion_ref,
           base.sesion_ref,
           control.control_sesion_ref,
           control.revision AS control_sesion_revision,
           control.estado AS control_sesion_estado,
           control.huella_sha256 AS control_sesion_huella_sha256,
           base.cuenta_ref,
           base.cuenta_ordinaria_ref,
           base.cuenta_privilegiada,
           base.superficie,
           base.metodo_observado,
           base.garantia_observada,
           base.politica_garantia_ref,
           base.politica_garantia_huella_sha256,
           base.autenticacion_verificada_en,
           base.sesion_emitida_en,
           control.sesion_valida_hasta,
           control.sesion_revalidada_en,
           consumo.cuenta_revision,
           consumo.cuenta_ordinaria_revision
      INTO STRICT sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_identidad_sesiones_v1.consumo_asercion AS consumo
        ON consumo.sesion_ref = base.sesion_ref
       AND consumo.autenticacion_ref = base.autenticacion_ref
       AND consumo.autenticacion_huella_sha256 =
           base.autenticacion_huella_sha256
       AND consumo.asercion_ref = base.asercion_ref
       AND consumo.cuenta_ref = base.cuenta_ref
       AND consumo.cuenta_ordinaria_ref = base.cuenta_ordinaria_ref
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = base.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = actual.sesion_ref
       AND control.control_sesion_ref = actual.control_sesion_ref
       AND control.revision = actual.revision
     WHERE base.autenticacion_ref = p_autenticacion_ref
       AND base.sesion_ref = p_sesion_ref
       AND consumo.control_sesion_ref = control.control_sesion_ref
       AND consumo.control_sesion_revision = control.revision
     FOR UPDATE OF actual;

    IF sesion.control_sesion_estado <> 'activa'
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.asercion_ref, 'ase_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.control_sesion_ref, 'cse_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.cuenta_ordinaria_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(
           sesion.politica_garantia_ref, 'pga_'
       ) IS NOT TRUE
       OR sesion.control_sesion_revision NOT BETWEEN
           1 AND 18446744073709551615
       OR sesion.autenticacion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.autenticacion_huella_sha256 = repeat('0', 64)
       OR sesion.control_sesion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.control_sesion_huella_sha256 = repeat('0', 64)
       OR sesion.politica_garantia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR sesion.politica_garantia_huella_sha256 = repeat('0', 64)
       OR sesion.superficie NOT IN (
           'externa_personal', 'interna_corporativa',
           'administracion_privilegiada'
       )
       OR sesion.metodo_observado NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
       )
       OR sesion.garantia_observada NOT IN (
           'bajo', 'sustancial', 'alto'
       )
       OR (sesion.superficie = 'externa_personal'
           AND sesion.garantia_observada = 'bajo')
       OR (sesion.superficie IN (
               'interna_corporativa', 'administracion_privilegiada'
           ) AND sesion.garantia_observada <> 'alto')
       OR sesion.autenticacion_verificada_en > sesion.sesion_emitida_en
       OR sesion.sesion_revalidada_en < sesion.autenticacion_verificada_en
       OR sesion.sesion_revalidada_en < sesion.sesion_emitida_en
       OR sesion.sesion_valida_hasta <= sesion.sesion_revalidada_en
       OR (sesion.cuenta_privilegiada AND (
           sesion.superficie <> 'administracion_privilegiada'
           OR sesion.cuenta_ref = sesion.cuenta_ordinaria_ref
       ))
       OR (NOT sesion.cuenta_privilegiada AND (
           sesion.superficie = 'administracion_privilegiada'
           OR sesion.cuenta_ref <> sesion.cuenta_ordinaria_ref
       )) THEN
        RETURN;
    END IF;

    cuentas_esperadas := CASE
        WHEN sesion.cuenta_privilegiada THEN 2
        ELSE 1
    END;
    FOR estado_bloqueado IN
        SELECT actual.cuenta_ref, actual.revision, estado.estado
          FROM vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
          JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
            ON estado.cuenta_ref = actual.cuenta_ref
           AND estado.revision = actual.revision
         WHERE actual.cuenta_ref IN (
             sesion.cuenta_ref, sesion.cuenta_ordinaria_ref
         )
         ORDER BY actual.cuenta_ref COLLATE "C"
         FOR UPDATE OF actual
    LOOP
        cuentas_bloqueadas := cuentas_bloqueadas + 1;
        IF estado_bloqueado.estado <> 'activa'
           OR (estado_bloqueado.cuenta_ref = sesion.cuenta_ref
               AND estado_bloqueado.revision <>
                   sesion.cuenta_revision)
           OR (estado_bloqueado.cuenta_ref = sesion.cuenta_ordinaria_ref
               AND estado_bloqueado.revision <>
                   sesion.cuenta_ordinaria_revision) THEN
            RETURN;
        END IF;
    END LOOP;
    IF cuentas_bloqueadas <> cuentas_esperadas THEN
        RETURN;
    END IF;

    -- El reloj se toma solo despues de todos los bloqueos. Los intervalos son
    -- semiabiertos: el instante exacto de expiracion ya no es valido.
    ahora := clock_timestamp();
    IF ahora < sesion.sesion_revalidada_en
       OR ahora >= sesion.sesion_valida_hasta
       OR ahora >= sesion.autenticacion_verificada_en + (
           CASE sesion.superficie
               WHEN 'externa_personal' THEN interval '12 hours'
               WHEN 'interna_corporativa' THEN interval '15 minutes'
               WHEN 'administracion_privilegiada' THEN interval '5 minutes'
               ELSE interval '0 seconds'
           END
       ) THEN
        RETURN;
    END IF;

    RETURN QUERY SELECT
        sesion.autenticacion_ref,
        sesion.autenticacion_huella_sha256,
        sesion.asercion_ref,
        sesion.sesion_ref,
        sesion.control_sesion_ref,
        sesion.control_sesion_revision::text,
        sesion.control_sesion_huella_sha256,
        sesion.cuenta_ref,
        sesion.cuenta_ordinaria_ref,
        sesion.cuenta_privilegiada,
        sesion.superficie,
        sesion.metodo_observado,
        sesion.garantia_observada,
        sesion.politica_garantia_ref,
        sesion.politica_garantia_huella_sha256,
        sesion.autenticacion_verificada_en,
        sesion.sesion_emitida_en,
        sesion.sesion_valida_hasta,
        sesion.sesion_revalidada_en;
EXCEPTION
    WHEN no_data_found OR too_many_rows OR data_exception
        OR invalid_text_representation OR numeric_value_out_of_range
        OR cardinality_violation THEN
        RETURN;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text, text)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text, text)
    TO vec_identidad_sesiones_v1_revalidador;

COMMIT;
