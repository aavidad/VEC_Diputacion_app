-- Politica temporal de desarrollo: unica autoridad, sin fila sembrada en Git.
-- El instalador privado fija ref, huella y retirada; ausencia o divergencia deniega.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

DO $pre$
BEGIN
    IF pg_catalog.to_regclass(
        'vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1'
    ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)'
    ) IS NOT NULL
       OR pg_catalog.to_regrole('vec_personal_propietario') IS NULL
       OR pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)'
    ) IS NULL THEN
        RAISE EXCEPTION 'precondiciones de politica personal de desarrollo incumplidas' USING ERRCODE='55000';
    END IF;
END $pre$;

-- Preimagen exacta de las cinco funciones instaladas que se van a sustituir.
-- Un cuerpo, propietario, configuración o ACL ajenos detienen la migración.
DO $preimagen$
DECLARE
    validas integer;
BEGIN
    WITH esperadas(firma, huella, bytes, configuracion, conjunto, consumidor,
                   cuerpo_sql) AS (
        VALUES
        ('vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)',
         'c8d2598ba4e08ccd427826fa335f05354f4e4f0b56e6c31abb2ea0c917b2c068',
         11566, ARRAY['search_path=pg_catalog, pg_temp']::text[], true,
         'vec_identidad_sesiones_v1_registrador', false),
        ('vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)',
         '28b5c23be1737918027342dc045132f6eba187130c51f9d48516bac6e48a043d',
         4739, ARRAY['search_path=pg_catalog, pg_temp']::text[], false,
         'vec_identidad_sesiones_v1_revalidador', false),
        ('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)',
         '277ab207e23d261522c6397578ebba9412f5111d25b985896756fe62677c3d46',
         7783, ARRAY['search_path=pg_catalog, pg_temp']::text[], true,
         'vec_identidad_sesiones_v1_revalidador', false),
        ('vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)',
         'de5d267d2cbf354962429e943686d00cc2e57e6e223b2d4dda525259698869eb',
         19458, ARRAY['search_path=pg_catalog', 'row_security=on',
                      'statement_timeout=5s', 'lock_timeout=2s']::text[],
         true, 'vec_identidad_sesiones_v1_lector_historico', false),
        ('vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)',
         'e6b45360b65a0d5e58289a2ca4e63044a650fe0753d00b8beb0b8116fb56888f',
         17789, ARRAY['search_path=pg_catalog', 'lock_timeout=1s']::text[],
         true, 'vec_contexto_actor_v1_propietario', true)
    ), manifestos AS (
        SELECT e.*, p.oid, p.prosrc, p.prosqlbody, p.proowner,
               p.proconfig, p.prosecdef, p.provolatile, p.proparallel,
               p.prokind, p.proleakproof, p.proisstrict, p.proretset,
               p.pronargs, p.proacl, l.lanname
          FROM esperadas AS e
          LEFT JOIN pg_catalog.pg_proc AS p
            ON p.oid = pg_catalog.to_regprocedure(e.firma)
          LEFT JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
    )
    SELECT count(*) INTO validas FROM manifestos AS m
     WHERE m.oid IS NOT NULL
       AND m.proowner = 'vec_identidad_sesiones_v1_propietario'::regrole
       AND m.proconfig = m.configuracion
       AND m.prosecdef AND m.provolatile = 'v' AND m.proparallel = 'u'
       AND m.prokind = 'f' AND NOT m.proleakproof AND NOT m.proisstrict
       AND m.proretset = m.conjunto
       AND m.pronargs = CASE WHEN m.cuerpo_sql THEN 2
                            WHEN m.firma LIKE '%leer_autenticacion_original%' THEN 3
                            WHEN m.firma LIKE '%revalidar_autenticacion_actor_v1%' THEN 2
                            ELSE 20 END
       AND m.lanname = CASE WHEN m.cuerpo_sql THEN 'sql' ELSE 'plpgsql' END
       AND (m.prosqlbody IS NOT NULL) = m.cuerpo_sql
       AND pg_catalog.octet_length(
           CASE WHEN m.cuerpo_sql THEN pg_catalog.pg_get_functiondef(m.oid)
                ELSE m.prosrc END
       ) = m.bytes
       AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           CASE WHEN m.cuerpo_sql THEN pg_catalog.pg_get_functiondef(m.oid)
                ELSE m.prosrc END, 'UTF8')), 'hex') = m.huella
       AND (
           SELECT count(*) = 2
              AND count(DISTINCT a.grantee) = 2
              AND bool_and(
                  a.grantor = m.proowner
                  AND a.grantee IN (
                      m.proowner, pg_catalog.to_regrole(m.consumidor)
                  )
                  AND a.privilege_type = 'EXECUTE'
                  AND NOT a.is_grantable
              )
             FROM pg_catalog.aclexplode(m.proacl) AS a
       );
    IF validas <> 5 THEN
        RAISE EXCEPTION 'preimagen de identidad divergente' USING ERRCODE='55000';
    END IF;
END $preimagen$;

CREATE TABLE vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1 (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    politica_ref text NOT NULL CHECK (
        politica_ref ~ '^pga_[A-Za-z0-9_-]{22,128}$'
    ),
    huella_sha256 text NOT NULL CHECK (
        huella_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_sha256 <> pg_catalog.repeat('0',64)
    ),
    retirar_en timestamptz(6) NOT NULL CHECK (
        pg_catalog.isfinite(retirar_en)
    ),
    activa boolean NOT NULL DEFAULT true,
    registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    CHECK (pg_catalog.isfinite(registrada_en) AND retirar_en > registrada_en)
);
REVOKE ALL ON TABLE
    vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
    FROM PUBLIC;
REVOKE ALL ON TYPE
    vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
    FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.inmutabilidad_politica_desarrollo_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog
AS $inmutable$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.activa AND NOT NEW.activa
       AND NEW.singleton IS NOT DISTINCT FROM OLD.singleton
       AND NEW.politica_ref IS NOT DISTINCT FROM OLD.politica_ref
       AND NEW.huella_sha256 IS NOT DISTINCT FROM OLD.huella_sha256
       AND NEW.retirar_en IS NOT DISTINCT FROM OLD.retirar_en
       AND NEW.registrada_en IS NOT DISTINCT FROM OLD.registrada_en THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'politica de desarrollo inmutable o ya retirada' USING ERRCODE='55000';
END $inmutable$;
REVOKE ALL ON FUNCTION
    vec_identidad_sesiones_v1.inmutabilidad_politica_desarrollo_v1()
    FROM PUBLIC;
CREATE TRIGGER impedir_cambio_politica_desarrollo
BEFORE UPDATE OR DELETE ON
    vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_identidad_sesiones_v1.inmutabilidad_politica_desarrollo_v1();

CREATE FUNCTION vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
    p_ref text, p_huella text, p_instante timestamptz
)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = on
AS $admite$
DECLARE
    politica record;
    ahora timestamptz(6);
BEGIN
    IF p_ref IS NULL OR p_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
       OR p_huella = pg_catalog.repeat('0',64)
       OR p_instante IS NULL OR NOT pg_catalog.isfinite(p_instante) THEN
        RETURN false;
    END IF;
    -- La revocación debe esperar a cualquier acreditación en curso. Una
    -- transacción de solo lectura no puede retener este bloqueo: se deniega.
    IF pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RETURN false;
    END IF;
    SELECT politica_ref, huella_sha256, retirar_en, activa, registrada_en
      INTO politica
      FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
     WHERE singleton FOR SHARE;
    ahora := pg_catalog.clock_timestamp();
    RETURN FOUND AND politica.activa
       AND politica.politica_ref = p_ref
       AND politica.huella_sha256 = p_huella
       AND p_instante >= politica.registrada_en
       AND p_instante <= ahora
       AND p_instante < politica.retirar_en
       AND ahora < politica.retirar_en;
EXCEPTION WHEN data_exception THEN
    RETURN false;
END $admite$;
REVOKE ALL ON FUNCTION
    vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1
    TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)
    TO vec_personal_propietario;

-- La historia demuestra qué política estaba vigente al verificarse la
-- autenticación original. Su revocación presente no reescribe ese hecho.
CREATE FUNCTION vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(
    p_ref text, p_huella text, p_instante timestamptz
)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = on
AS $historica$
DECLARE
    politica record;
BEGIN
    IF p_ref IS NULL OR p_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
       OR p_instante IS NULL OR NOT pg_catalog.isfinite(p_instante) THEN
        RETURN false;
    END IF;
    SELECT politica_ref, huella_sha256, retirar_en, registrada_en
      INTO politica
      FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
     WHERE singleton;
    RETURN FOUND AND politica.politica_ref = p_ref
       AND politica.huella_sha256 = p_huella
       AND p_instante >= politica.registrada_en
       AND p_instante < politica.retirar_en;
EXCEPTION WHEN data_exception THEN
    RETURN false;
END $historica$;
REVOKE ALL ON FUNCTION
    vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)
    FROM PUBLIC;

-- Sonda nominal de arranque: el proceso y PostgreSQL deben compartir también
-- la fecha de retirada, además de referencia y huella de la misma política.
CREATE FUNCTION vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(
    p_ref text, p_huella text, p_retirada_esperada timestamptz
)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = on
AS $coincide$
DECLARE
    politica record;
BEGIN
    IF p_ref IS NULL OR p_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$'
       OR p_retirada_esperada IS NULL
       OR NOT pg_catalog.isfinite(p_retirada_esperada) THEN
        RETURN false;
    END IF;
    SELECT politica_ref, huella_sha256, retirar_en, activa
      INTO politica
      FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
     WHERE singleton FOR SHARE;
    RETURN FOUND AND politica.activa
       AND politica.politica_ref = p_ref
       AND politica.huella_sha256 = p_huella
       AND politica.retirar_en = p_retirada_esperada
       AND pg_catalog.clock_timestamp() < politica.retirar_en;
EXCEPTION WHEN data_exception OR read_only_sql_transaction THEN
    RETURN false;
END $coincide$;
REVOKE ALL ON FUNCTION
    vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(text,text,timestamptz)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(text,text,timestamptz)
    TO vec_identidad_sesiones_v1_revalidador;

-- Conservar las firmas/ACL originales: se sustituye solamente el cuerpo.

CREATE OR REPLACE FUNCTION vec_identidad_sesiones_v1.registrar_sesion_v1(
    p_operacion_ref text,
    p_esquema_hmac text,
    p_dominio_hmac_ref text,
    p_clave_hmac_id text,
    p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea,
    p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea,
    p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea,
    p_cuenta_privilegiada boolean,
    p_superficie text,
    p_metodo_observado text,
    p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz,
    p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text,
    p_politica_garantia_huella_sha256 text
)
RETURNS TABLE(
    autenticacion_ref text,
    asercion_ref text,
    sesion_ref text,
    control_sesion_ref text,
    control_sesion_revision_texto text,
    control_sesion_estado text,
    control_sesion_huella_sha256 text,
    cuenta_ref text,
    cuenta_ordinaria_ref text,
    sesion_revalidada_en timestamptz,
    sesion_valida_hasta timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    cuenta_base record;
    cuenta_ordinaria record;
    cuenta_ordinaria_resuelta_ref text;
    cuenta_base_revision numeric;
    cuenta_ordinaria_revision numeric;
    estado_bloqueado record;
    cuentas_activas integer := 0;
    autenticacion_nueva_ref text;
    asercion_nueva_ref text;
    sesion_nueva_ref text;
    control_nuevo_ref text;
    control_huella text;
    ahora timestamptz(6);
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(
           p_operacion_ref, 'opr_'
       ) IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
           p_esquema_hmac, p_dominio_hmac_ref,
           p_clave_hmac_id, p_clave_hmac_version
       ) IS NOT TRUE
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               p_asercion_id_hmac, p_sesion_id_hmac, p_sujeto_id_hmac,
               p_cuenta_id_hmac
           ]) AS huella
           WHERE vec_identidad_sesiones_v1.huella_hmac_valida(
               huella
           ) IS NOT TRUE
       )
       OR p_asercion_id_hmac IN (
           p_sesion_id_hmac, p_sujeto_id_hmac, p_cuenta_id_hmac
       )
       OR p_sesion_id_hmac IN (p_sujeto_id_hmac, p_cuenta_id_hmac)
       OR p_sujeto_id_hmac = p_cuenta_id_hmac
       OR p_cuenta_privilegiada IS NULL
       OR (p_cuenta_privilegiada AND (
           vec_identidad_sesiones_v1.huella_hmac_valida(
               p_cuenta_ordinaria_id_hmac
           ) IS NOT TRUE
           OR p_cuenta_ordinaria_id_hmac IN (
               p_asercion_id_hmac, p_sesion_id_hmac,
               p_sujeto_id_hmac, p_cuenta_id_hmac
           )
       ))
       OR (NOT p_cuenta_privilegiada
           AND p_cuenta_ordinaria_id_hmac IS NOT NULL)
       OR p_superficie IS NULL OR p_superficie NOT IN (
           'externa_personal', 'interna_corporativa',
           'administracion_privilegiada'
       )
       OR p_metodo_observado IS NULL OR p_metodo_observado NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
       )
       OR p_garantia_observada IS NULL
       OR p_garantia_observada NOT IN ('bajo', 'sustancial', 'alto')
       OR (p_superficie = 'externa_personal'
           AND p_garantia_observada = 'bajo')
       OR (p_superficie = 'administracion_privilegiada'
           AND p_garantia_observada <> 'alto')
       OR (p_superficie = 'interna_corporativa'
           AND p_garantia_observada <> 'alto'
           AND NOT (p_metodo_observado = 'certificado'
               AND p_garantia_observada = 'sustancial'
               AND vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
                   p_politica_garantia_ref,
                   p_politica_garantia_huella_sha256,
                   p_autenticacion_verificada_en)))
       OR p_autenticacion_huella_sha256 IS NULL
       OR p_autenticacion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autenticacion_huella_sha256 = repeat('0', 64)
       OR p_politica_garantia_ref IS NULL
       OR p_politica_garantia_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_politica_garantia_huella_sha256 IS NULL
       OR p_politica_garantia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_politica_garantia_huella_sha256 = repeat('0', 64)
       OR p_autenticacion_verificada_en IS NULL
       OR p_sesion_emitida_en IS NULL OR p_asercion_expira_en IS NULL
       OR p_autenticacion_verificada_en > p_sesion_emitida_en
       OR p_asercion_expira_en <= p_sesion_emitida_en
       OR p_asercion_expira_en - p_sesion_emitida_en >
           interval '5 minutes'
       OR (p_cuenta_privilegiada AND
           p_superficie <> 'administracion_privilegiada')
       OR (NOT p_cuenta_privilegiada AND
           p_superficie = 'administracion_privilegiada')
       OR EXISTS (
           SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion
            WHERE operacion_ref = p_operacion_ref
       ) THEN
        RETURN;
    END IF;

    -- Una sesion IdP puede reemitir aserciones, pero nunca mantiene dos
    -- sesiones VEC simultaneas dentro de una coordenada HMAC. El bloqueo no
    -- contiene datos fuente y una colision solo sobre-serializa de forma segura.
    PERFORM pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_sesiones_v1:sesion:' || p_dominio_hmac_ref || ':' ||
        p_clave_hmac_id || ':' || p_clave_hmac_version::text || ':' ||
        encode(p_sesion_id_hmac, 'hex'),
        0
    ));

    SELECT cuenta.cuenta_ref, cuenta.cuenta_privilegiada,
           cuenta.cuenta_ordinaria_ref
      INTO cuenta_base
      FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
      JOIN vec_identidad_sesiones_v1.cuenta AS cuenta
        ON cuenta.cuenta_ref = alias.cuenta_ref
     WHERE alias.esquema_hmac = p_esquema_hmac
       AND alias.dominio_hmac_ref = p_dominio_hmac_ref
       AND alias.clave_hmac_id = p_clave_hmac_id
       AND alias.clave_hmac_version = p_clave_hmac_version
       AND alias.cuenta_id_hmac = p_cuenta_id_hmac
       AND alias.sujeto_id_hmac = p_sujeto_id_hmac;
    IF NOT FOUND OR cuenta_base.cuenta_privilegiada IS DISTINCT FROM
           p_cuenta_privilegiada THEN
        RETURN;
    END IF;

    IF p_cuenta_privilegiada THEN
        SELECT cuenta.cuenta_ref, cuenta.cuenta_privilegiada
          INTO cuenta_ordinaria
          FROM vec_identidad_sesiones_v1.alias_hmac_cuenta AS alias
          JOIN vec_identidad_sesiones_v1.cuenta AS cuenta
            ON cuenta.cuenta_ref = alias.cuenta_ref
         WHERE alias.esquema_hmac = p_esquema_hmac
           AND alias.dominio_hmac_ref = p_dominio_hmac_ref
           AND alias.clave_hmac_id = p_clave_hmac_id
           AND alias.clave_hmac_version = p_clave_hmac_version
           AND alias.cuenta_id_hmac = p_cuenta_ordinaria_id_hmac
           AND alias.sujeto_id_hmac = p_sujeto_id_hmac;
        IF NOT FOUND OR cuenta_ordinaria.cuenta_privilegiada
           OR cuenta_base.cuenta_ordinaria_ref IS DISTINCT FROM
               cuenta_ordinaria.cuenta_ref THEN
            RETURN;
        END IF;
        cuenta_ordinaria_resuelta_ref := cuenta_ordinaria.cuenta_ref;
    ELSE
        cuenta_ordinaria_resuelta_ref := cuenta_base.cuenta_ref;
    END IF;

    -- El orden global evita interbloqueos entre altas privilegiadas y cambios
    -- de estado. Toda mutacion de cuenta bloquea el mismo puntero estable.
    FOR estado_bloqueado IN
        SELECT actual.cuenta_ref, actual.revision, estado.estado
          FROM vec_identidad_sesiones_v1.estado_cuenta_actual AS actual
          JOIN vec_identidad_sesiones_v1.estado_cuenta AS estado
            ON estado.cuenta_ref = actual.cuenta_ref
           AND estado.revision = actual.revision
         WHERE actual.cuenta_ref IN (
             cuenta_base.cuenta_ref, cuenta_ordinaria_resuelta_ref
         )
         ORDER BY actual.cuenta_ref COLLATE "C"
         FOR UPDATE OF actual
    LOOP
        IF estado_bloqueado.estado = 'activa' THEN
            cuentas_activas := cuentas_activas + 1;
            IF estado_bloqueado.cuenta_ref = cuenta_base.cuenta_ref THEN
                cuenta_base_revision := estado_bloqueado.revision;
            END IF;
            IF estado_bloqueado.cuenta_ref =
                   cuenta_ordinaria_resuelta_ref THEN
                cuenta_ordinaria_revision := estado_bloqueado.revision;
            END IF;
        END IF;
    END LOOP;
    IF cuentas_activas <>
           (CASE WHEN p_cuenta_privilegiada THEN 2 ELSE 1 END) THEN
        RETURN;
    END IF;

    ahora := clock_timestamp();
    IF ahora < p_sesion_emitida_en OR ahora >= p_asercion_expira_en
       OR ahora >= p_autenticacion_verificada_en + (
           CASE p_superficie
               WHEN 'externa_personal' THEN interval '12 hours'
               WHEN 'interna_corporativa' THEN interval '15 minutes'
               WHEN 'administracion_privilegiada' THEN interval '5 minutes'
           END
       )
       OR EXISTS (
           SELECT 1
             FROM vec_identidad_sesiones_v1.consumo_asercion AS consumo
             JOIN vec_autorizacion.control_sesion_v1 AS control
               ON control.sesion_ref = consumo.sesion_ref
              AND control.control_sesion_ref = consumo.control_sesion_ref
              AND control.revision = consumo.control_sesion_revision
            WHERE consumo.esquema_hmac = p_esquema_hmac
              AND consumo.dominio_hmac_ref = p_dominio_hmac_ref
              AND consumo.clave_hmac_id = p_clave_hmac_id
              AND consumo.clave_hmac_version = p_clave_hmac_version
              AND consumo.sesion_id_hmac = p_sesion_id_hmac
              AND control.sesion_valida_hasta > ahora
       ) THEN
        RETURN;
    END IF;
    autenticacion_nueva_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('aut_');
    asercion_nueva_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('ase_');
    sesion_nueva_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('ses_');
    control_nuevo_ref :=
        vec_identidad_sesiones_v1.nueva_referencia('cse_');
    control_huella := vec_identidad_sesiones_v1.huella_control_sesion_v1(
        control_nuevo_ref, 1, sesion_nueva_ref, 'activa', ahora,
        p_asercion_expira_en, p_operacion_ref
    );

    INSERT INTO vec_autorizacion.sesion_autenticacion_v1 (
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        asercion_ref, cuenta_ref, cuenta_ordinaria_ref,
        cuenta_privilegiada, superficie, metodo_observado,
        garantia_observada, politica_garantia_ref,
        politica_garantia_huella_sha256, autenticacion_verificada_en,
        sesion_emitida_en
    ) VALUES (
        sesion_nueva_ref, autenticacion_nueva_ref,
        p_autenticacion_huella_sha256, asercion_nueva_ref,
        cuenta_base.cuenta_ref, cuenta_ordinaria_resuelta_ref,
        p_cuenta_privilegiada, p_superficie, p_metodo_observado,
        p_garantia_observada, p_politica_garantia_ref,
        p_politica_garantia_huella_sha256,
        p_autenticacion_verificada_en, p_sesion_emitida_en
    );
    INSERT INTO vec_autorizacion.control_sesion_v1 (
        control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
        sesion_revalidada_en, sesion_valida_hasta
    ) VALUES (
        control_nuevo_ref, 1, sesion_nueva_ref, 'activa', control_huella,
        ahora, p_asercion_expira_en
    );
    INSERT INTO vec_autorizacion.control_sesion_actual_v1 (
        sesion_ref, control_sesion_ref, revision, actualizada_en, acto_ref
    ) VALUES (
        sesion_nueva_ref, control_nuevo_ref, 1, ahora, p_operacion_ref
    );
    INSERT INTO vec_identidad_sesiones_v1.consumo_asercion (
        operacion_ref, esquema_hmac, dominio_hmac_ref, clave_hmac_id,
        clave_hmac_version, asercion_id_hmac, sesion_id_hmac,
        sujeto_id_hmac, cuenta_id_hmac, cuenta_ordinaria_id_hmac,
        autenticacion_ref, autenticacion_huella_sha256,
        asercion_ref, sesion_ref, control_sesion_ref,
        control_sesion_revision, cuenta_ref, cuenta_revision,
        cuenta_ordinaria_ref, cuenta_ordinaria_revision, consumida_en
    ) VALUES (
        p_operacion_ref, p_esquema_hmac, p_dominio_hmac_ref,
        p_clave_hmac_id, p_clave_hmac_version, p_asercion_id_hmac,
        p_sesion_id_hmac, p_sujeto_id_hmac, p_cuenta_id_hmac,
        p_cuenta_ordinaria_id_hmac, autenticacion_nueva_ref,
        p_autenticacion_huella_sha256, asercion_nueva_ref,
        sesion_nueva_ref, control_nuevo_ref, 1,
        cuenta_base.cuenta_ref, cuenta_base_revision,
        cuenta_ordinaria_resuelta_ref, cuenta_ordinaria_revision, ahora
    );

    RETURN QUERY SELECT
        autenticacion_nueva_ref, asercion_nueva_ref, sesion_nueva_ref,
        control_nuevo_ref, '1'::text, 'activa'::text, control_huella,
        cuenta_base.cuenta_ref, cuenta_ordinaria_resuelta_ref,
        ahora, p_asercion_expira_en;
EXCEPTION
    WHEN data_exception OR invalid_parameter_value OR unique_violation
        OR foreign_key_violation OR check_violation OR cardinality_violation THEN
        RETURN;
END
$funcion$;

CREATE OR REPLACE FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
    p_autenticacion_ref text,
    p_autenticacion_huella_sha256 text,
    p_asercion_ref text,
    p_sesion_ref text,
    p_cuenta_ref text,
    p_cuenta_ordinaria_ref text,
    p_cuenta_privilegiada boolean,
    p_superficie text,
    p_metodo_observado text,
    p_garantia_observada text,
    p_politica_garantia_ref text,
    p_politica_garantia_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz,
    p_control_sesion_ref text,
    p_control_sesion_revision_texto text,
    p_control_sesion_estado text,
    p_control_sesion_huella_sha256 text,
    p_sesion_revalidada_en timestamptz,
    p_sesion_valida_hasta timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    revision_esperada numeric;
    sesion record;
    estado_bloqueado record;
    cuentas_activas integer := 0;
    ahora timestamptz(6);
BEGIN
    IF p_control_sesion_revision_texto !~ '^[1-9][0-9]{0,19}$'
       OR p_control_sesion_estado <> 'activa' THEN
        RETURN false;
    END IF;
    revision_esperada := p_control_sesion_revision_texto::numeric;
    IF revision_esperada > 18446744073709551615 THEN
        RETURN false;
    END IF;

    SELECT base.sesion_ref, base.cuenta_ref, base.cuenta_ordinaria_ref,
           base.cuenta_privilegiada, base.superficie,
           base.metodo_observado, base.garantia_observada,
           base.politica_garantia_ref,
           base.politica_garantia_huella_sha256,
           base.autenticacion_verificada_en, control.sesion_valida_hasta,
           consumo.cuenta_revision, consumo.cuenta_ordinaria_revision
      INTO sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_identidad_sesiones_v1.consumo_asercion AS consumo
        ON consumo.sesion_ref = base.sesion_ref
       AND consumo.autenticacion_ref = base.autenticacion_ref
       AND consumo.autenticacion_huella_sha256 =
           base.autenticacion_huella_sha256
       AND consumo.cuenta_ref = base.cuenta_ref
       AND consumo.cuenta_ordinaria_ref = base.cuenta_ordinaria_ref
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = base.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = actual.sesion_ref
       AND control.control_sesion_ref = actual.control_sesion_ref
       AND control.revision = actual.revision
     WHERE base.autenticacion_ref = p_autenticacion_ref
       AND base.autenticacion_huella_sha256 =
           p_autenticacion_huella_sha256
       AND base.asercion_ref = p_asercion_ref
       AND base.sesion_ref = p_sesion_ref
       AND base.cuenta_ref = p_cuenta_ref
       AND base.cuenta_ordinaria_ref = p_cuenta_ordinaria_ref
       AND base.cuenta_privilegiada = p_cuenta_privilegiada
       AND base.superficie = p_superficie
       AND base.metodo_observado = p_metodo_observado
       AND base.garantia_observada = p_garantia_observada
       AND base.politica_garantia_ref = p_politica_garantia_ref
       AND base.politica_garantia_huella_sha256 =
           p_politica_garantia_huella_sha256
       AND base.autenticacion_verificada_en =
           p_autenticacion_verificada_en
       AND base.sesion_emitida_en = p_sesion_emitida_en
       AND control.control_sesion_ref = p_control_sesion_ref
       AND control.revision = revision_esperada
       AND control.estado = p_control_sesion_estado
       AND control.huella_sha256 = p_control_sesion_huella_sha256
       AND control.sesion_revalidada_en = p_sesion_revalidada_en
       AND control.sesion_valida_hasta = p_sesion_valida_hasta
       AND consumo.control_sesion_ref = control.control_sesion_ref
       AND consumo.control_sesion_revision = control.revision
     FOR UPDATE OF actual;
    IF NOT FOUND THEN
        RETURN false;
    END IF;

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
        IF estado_bloqueado.estado = 'activa'
           AND (
               (estado_bloqueado.cuenta_ref = sesion.cuenta_ref
                AND estado_bloqueado.revision = sesion.cuenta_revision)
               OR
               (estado_bloqueado.cuenta_ref = sesion.cuenta_ordinaria_ref
                AND estado_bloqueado.revision =
                    sesion.cuenta_ordinaria_revision)
           ) THEN
            cuentas_activas := cuentas_activas + 1;
        END IF;
    END LOOP;
    IF cuentas_activas <>
           (CASE WHEN sesion.cuenta_privilegiada THEN 2 ELSE 1 END) THEN
        RETURN false;
    END IF;

    ahora := clock_timestamp();
    RETURN ahora >= p_sesion_revalidada_en
       AND (CASE sesion.superficie
            WHEN 'externa_personal' THEN sesion.garantia_observada <> 'bajo'
            WHEN 'administracion_privilegiada' THEN
                sesion.garantia_observada = 'alto'
            WHEN 'interna_corporativa' THEN
                sesion.garantia_observada = 'alto'
                OR (sesion.garantia_observada = 'sustancial'
                    AND sesion.metodo_observado = 'certificado'
                    AND vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
                        sesion.politica_garantia_ref,
                        sesion.politica_garantia_huella_sha256,
                        sesion.autenticacion_verificada_en))
            ELSE false END)
       AND ahora < sesion.sesion_valida_hasta
       AND ahora < sesion.autenticacion_verificada_en + (
           CASE sesion.superficie
               WHEN 'externa_personal' THEN interval '12 hours'
               WHEN 'interna_corporativa' THEN interval '15 minutes'
               WHEN 'administracion_privilegiada' THEN interval '5 minutes'
               ELSE interval '0 seconds'
           END
       );
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR cardinality_violation THEN
        RETURN false;
END
$funcion$;

CREATE OR REPLACE FUNCTION vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
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
       OR (sesion.superficie = 'administracion_privilegiada'
           AND sesion.garantia_observada <> 'alto')
       OR (sesion.superficie = 'interna_corporativa'
           AND sesion.garantia_observada <> 'alto'
           AND NOT (sesion.metodo_observado = 'certificado'
               AND sesion.garantia_observada = 'sustancial'
               AND vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
                   sesion.politica_garantia_ref,
                   sesion.politica_garantia_huella_sha256,
                   sesion.autenticacion_verificada_en)))
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

CREATE OR REPLACE FUNCTION vec_identidad_sesiones_v1.leer_autenticacion_original_v1(
    p_autenticacion_ref text, p_sesion_ref text, p_autenticacion_sha256 text
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
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog
SET row_security=on
SET statement_timeout='5s'
SET lock_timeout='2s'
AS $historia$
DECLARE

    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;

    sesion record; estado record; instante timestamptz;
    cuentas integer:=0; esperadas integer;
    inicio timestamptz:=clock_timestamp();
BEGIN
    IF current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'on'
       OR current_setting('TimeZone')<>'UTC' THEN
        RAISE EXCEPTION 'lector historico requiere SERIALIZABLE READ ONLY UTC' USING ERRCODE='25000';
    END IF;

    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_identidad_sesiones_v1_lector_historico';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_identidad_sesiones_v1';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)')];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 1 OR array_position(funciones,NULL) IS NOT NULL
       OR login.rolcanlogin IS NOT TRUE
       OR login.rolsuper OR NOT login.rolinherit OR login.rolcreaterole OR login.rolcreatedb
       OR login.rolreplication OR login.rolbypassrls OR login.rolconfig IS NOT NULL
       OR grupo.rolcanlogin OR grupo.rolsuper OR grupo.rolinherit
       OR grupo.rolcreaterole OR grupo.rolcreatedb OR grupo.rolreplication
       OR grupo.rolbypassrls OR grupo.rolconfig IS NOT NULL
       OR current_setting('role') <> 'none' OR membresias <> 1
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>login_oid
              AND pg_catalog.pg_has_role(login_oid,r.oid,'MEMBER')
              AND r.oid<>runtime_oid
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>runtime_oid
              AND pg_catalog.pg_has_role(runtime_oid,r.oid,'MEMBER')
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = runtime_oid
              AND admin_option IS FALSE AND inherit_option IS TRUE AND set_option IS FALSE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting s
            WHERE s.setrole IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl d
           LEFT JOIN LATERAL pg_catalog.aclexplode(
             coalesce(d.defaclacl,'{}'::aclitem[])
           ) a ON true
            WHERE d.defaclrole IN (login_oid,runtime_oid)
               OR a.grantee IN (login_oid,runtime_oid)
               OR a.grantor IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy p
            WHERE login_oid=ANY(p.polroles) OR runtime_oid=ANY(p.polroles)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=login_oid
       )
       OR NOT COALESCE((
           SELECT count(*)=3 AND bool_and(
             d.deptype='a' AND d.objsubid=0 AND (
               (d.classid='pg_catalog.pg_database'::regclass AND d.objid=base_oid) OR
               (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=esquema_oid) OR
               (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(funciones))
             ))
             FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
             FROM pg_catalog.pg_database b
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
             ) a
            WHERE b.oid=base_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
             ) a
            WHERE n.oid=esquema_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND count(DISTINCT p.oid)=1
                  AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))
            ) a
            WHERE p.oid=ANY(funciones) AND a.grantee=runtime_oid
       ),false)
       OR (pg_catalog.has_database_privilege(login_oid,base_oid,'CONNECT')
       AND NOT pg_catalog.has_database_privilege(login_oid,base_oid,'CREATE')
       AND NOT pg_catalog.has_database_privilege(login_oid,base_oid,'TEMPORARY')
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_namespace n
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND ((n.oid <> esquema_oid
                  AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE'))
                 OR pg_catalog.has_schema_privilege(login_oid,n.oid,'CREATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind IN ('r','p','v','m','f') AND (
              pg_catalog.has_table_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'INSERT') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'DELETE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'TRUNCATE') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'REFERENCES') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'TRIGGER') OR
              pg_catalog.has_table_privilege(login_oid,c.oid,'MAINTAIN') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'INSERT') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_any_column_privilege(login_oid,c.oid,'REFERENCES'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind='S' AND (
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'USAGE') OR
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'SELECT') OR
              pg_catalog.has_sequence_privilege(login_oid,c.oid,'UPDATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_proc p
         JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
            AND p.oid <> ALL(funciones)
            AND pg_catalog.has_function_privilege(login_oid,p.oid,'EXECUTE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_type t
         JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND t.typtype IN ('c','d','e','m','r')
            AND pg_catalog.has_type_privilege(login_oid,t.oid,'USAGE')
            -- PostgreSQL atribuye USAGE implícito al tipo fila de una tabla.
            -- Sin ACL propia ni acceso a su esquema, no concede acceso a datos.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
              AND NOT pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
              AND EXISTS (
                SELECT 1 FROM pg_catalog.pg_class relacion
                 WHERE relacion.oid=t.typrelid
                   AND relacion.relkind IN ('r','p','v','m','f')
              )
            )
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_largeobject_metadata l
          WHERE pg_catalog.has_largeobject_privilege(login_oid,l.oid,'SELECT')
             OR pg_catalog.has_largeobject_privilege(login_oid,l.oid,'UPDATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_data_wrapper f
          WHERE pg_catalog.has_foreign_data_wrapper_privilege(login_oid,f.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_server s
          WHERE pg_catalog.has_server_privilege(login_oid,s.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_language l
          WHERE l.oid >= 16384
            AND pg_catalog.has_language_privilege(login_oid,l.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_tablespace t
          WHERE t.oid >= 16384
            AND pg_catalog.has_tablespace_privilege(login_oid,t.oid,'CREATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_parameter_acl a
          WHERE pg_catalog.has_parameter_privilege(login_oid,a.parname,'SET')
             OR pg_catalog.has_parameter_privilege(login_oid,a.parname,'ALTER SYSTEM')
       )) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN lector historico no acreditado';
    END IF;

    IF vec_identidad_sesiones_v1.referencia_valida(p_autenticacion_ref,'aut_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(p_sesion_ref,'ses_') IS NOT TRUE
       OR p_autenticacion_sha256 IS NULL OR p_autenticacion_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autenticacion_sha256=repeat('0',64) THEN
        RAISE EXCEPTION 'selector historico invalido' USING ERRCODE='22023';
    END IF;
    PERFORM pg_advisory_xact_lock_shared(hashtextextended(
      'vec_identidad_sesiones_v1:migracion:lectura_historica:v1',0));
    -- Selección explícita: nunca leer ni devolver columnas HMAC de consumo.
    -- La revisión es la fijada por el consumo original; no punteros actuales.
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
           consumo.cuenta_ordinaria_revision,
           consumo.operacion_ref, consumo.consumida_en
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
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = consumo.sesion_ref
       AND control.control_sesion_ref = consumo.control_sesion_ref
       AND control.revision = consumo.control_sesion_revision
     WHERE base.autenticacion_ref = p_autenticacion_ref
       AND base.sesion_ref = p_sesion_ref
       AND consumo.control_sesion_ref = control.control_sesion_ref
       AND consumo.control_sesion_revision = control.revision
       AND consumo.autenticacion_huella_sha256 = p_autenticacion_sha256;


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
       OR (sesion.superficie = 'administracion_privilegiada'
           AND sesion.garantia_observada <> 'alto')
       OR (sesion.superficie = 'interna_corporativa'
           AND sesion.garantia_observada <> 'alto'
           AND NOT (sesion.metodo_observado = 'certificado'
               AND sesion.garantia_observada = 'sustancial'
               AND vec_identidad_sesiones_v1.existio_politica_certificado_personal_desarrollo_v1(
                   sesion.politica_garantia_ref,
                   sesion.politica_garantia_huella_sha256,
                   sesion.autenticacion_verificada_en)))
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
        RAISE EXCEPTION 'autenticacion historica incoherente' USING ERRCODE='22023';
    END IF;


    IF sesion.autenticacion_ref IS DISTINCT FROM p_autenticacion_ref
       OR sesion.sesion_ref IS DISTINCT FROM p_sesion_ref
       OR sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_autenticacion_sha256
       OR sesion.consumida_en IS DISTINCT FROM sesion.sesion_revalidada_en
       OR sesion.control_sesion_huella_sha256 IS DISTINCT FROM
         vec_identidad_sesiones_v1.huella_control_sesion_v1(
           sesion.control_sesion_ref,sesion.control_sesion_revision,sesion.sesion_ref,
           sesion.control_sesion_estado,sesion.sesion_revalidada_en,sesion.sesion_valida_hasta,
           sesion.operacion_ref)
       OR sesion.sesion_valida_hasta > sesion.sesion_emitida_en + interval '5 minutes'
       OR sesion.sesion_revalidada_en >= sesion.autenticacion_verificada_en + (CASE sesion.superficie
           WHEN 'externa_personal' THEN interval '12 hours'
           WHEN 'interna_corporativa' THEN interval '15 minutes'
           WHEN 'administracion_privilegiada' THEN interval '5 minutes'
           ELSE interval '0 seconds' END) THEN
        RAISE EXCEPTION 'origen historico divergente' USING ERRCODE='22023';
    END IF;
    FOREACH instante IN ARRAY ARRAY[sesion.autenticacion_verificada_en,sesion.sesion_emitida_en,
       sesion.sesion_valida_hasta,sesion.sesion_revalidada_en,sesion.consumida_en] LOOP
        IF instante IS NULL OR NOT isfinite(instante)
           OR instante < timestamptz '0001-01-01 00:00:00+00'
           OR instante >= timestamptz '10000-01-01 00:00:00+00' THEN
            RAISE EXCEPTION 'fecha historica invalida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    esperadas:=CASE WHEN sesion.cuenta_privilegiada THEN 2 ELSE 1 END;
    FOR estado IN
      SELECT e.cuenta_ref,e.revision,e.estado,e.registrada_en
      FROM vec_identidad_sesiones_v1.estado_cuenta e
      WHERE (e.cuenta_ref=sesion.cuenta_ref AND e.revision=sesion.cuenta_revision)
         OR (e.cuenta_ref=sesion.cuenta_ordinaria_ref AND e.revision=sesion.cuenta_ordinaria_revision)
    LOOP
        cuentas:=cuentas+1;
        IF estado.estado IS DISTINCT FROM 'activa'
           OR NOT isfinite(estado.registrada_en)
           OR estado.registrada_en > sesion.consumida_en
           OR (estado.cuenta_ref=sesion.cuenta_ref AND estado.revision<>sesion.cuenta_revision)
           OR (estado.cuenta_ref=sesion.cuenta_ordinaria_ref AND estado.revision<>sesion.cuenta_ordinaria_revision) THEN
            RAISE EXCEPTION 'cuenta historica divergente' USING ERRCODE='22023';
        END IF;
    END LOOP;
    IF cuentas<>esperadas THEN
        RAISE EXCEPTION 'cuenta historica ausente' USING ERRCODE='P0002';
    END IF;
    -- El reloj actual sólo limita la ejecución. No exige permiso actual.
    IF clock_timestamp()>inicio+interval '5 seconds' THEN
        RAISE EXCEPTION 'lectura historica fuera de ventana' USING ERRCODE='57014';
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
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'autenticacion historica no disponible' USING ERRCODE='P0002';
    WHEN data_exception THEN
        RAISE EXCEPTION 'autenticacion historica invalida' USING ERRCODE='22023';
END
$historia$;

CREATE OR REPLACE FUNCTION
    vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(
        p_autenticacion_ref text,
        p_sesion_ref text
    )
RETURNS TABLE (
    cuenta_ref text,
    metodo_observado text,
    garantia_observada text,
    identidad_valida_hasta timestamptz
)
LANGUAGE SQL
VOLATILE
CALLED ON NULL INPUT
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET lock_timeout = '1s'
BEGIN ATOMIC
    WITH guarda AS MATERIALIZED (
        SELECT pg_catalog.pg_advisory_xact_lock_shared(
            pg_catalog.hashtextextended(
                'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1',
                0
            )
        )
    ), identidades AS MATERIALIZED (
        SELECT login.oid AS login_oid, selector.oid AS selector_oid,
               propia.oid AS propia_oid, base.oid AS base_oid,
               identidad.oid AS identidad_oid,
               consumidor.oid AS consumidor_oid,
               revalidador.oid AS revalidador_oid
          FROM guarda
          CROSS JOIN pg_catalog.pg_roles AS login
          CROSS JOIN pg_catalog.pg_roles AS selector
          CROSS JOIN pg_catalog.pg_roles AS identidad
          CROSS JOIN pg_catalog.pg_roles AS consumidor
          CROSS JOIN pg_catalog.pg_roles AS revalidador
          CROSS JOIN pg_catalog.pg_proc AS propia
          CROSS JOIN pg_catalog.pg_proc AS base
         WHERE login.rolname = session_user
           AND selector.rolname =
               'vec_contexto_actor_corporativo_rrhh_selector'
           AND identidad.rolname =
               'vec_identidad_sesiones_v1_propietario'
           AND consumidor.rolname =
               'vec_contexto_actor_v1_propietario'
           AND revalidador.rolname =
               'vec_identidad_sesiones_v1_revalidador'
           AND propia.oid = pg_catalog.to_regprocedure(
               'vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(text,text)'
           )
           AND base.oid = pg_catalog.to_regprocedure(
               'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'
           )
           AND current_user =
               'vec_identidad_sesiones_v1_propietario'
           AND pg_catalog.current_setting('role') = 'none'
           AND pg_catalog.current_setting('transaction_isolation') =
               'serializable'
           AND pg_catalog.current_setting('transaction_read_only') = 'off'
           AND pg_catalog.pg_is_in_recovery() IS FALSE
           AND login.rolcanlogin AND login.rolinherit
           AND NOT login.rolsuper AND NOT login.rolcreatedb
           AND NOT login.rolcreaterole AND NOT login.rolreplication
           AND NOT login.rolbypassrls AND login.rolconfig IS NULL
           AND (
               login.rolvaliduntil IS NULL
               OR pg_catalog.clock_timestamp() < login.rolvaliduntil
           )
           AND NOT selector.rolcanlogin AND NOT selector.rolsuper
           AND NOT selector.rolcreatedb AND NOT selector.rolcreaterole
           AND NOT selector.rolinherit AND NOT selector.rolreplication
           AND NOT selector.rolbypassrls AND selector.rolconnlimit = -1
           AND selector.rolvaliduntil IS NULL
           AND selector.rolconfig IS NULL
           AND pg_catalog.shobj_description(
                   selector.oid, 'pg_authid'
               ) =
               'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1'
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_db_role_setting AS ajuste
                WHERE ajuste.setrole IN (login.oid, selector.oid)
           )
           AND (
               SELECT pg_catalog.count(*) = 1
                  AND pg_catalog.bool_and(
                      m.roleid = selector.oid AND m.member = login.oid
                      AND NOT m.admin_option AND m.inherit_option
                      AND NOT m.set_option
                      AND otorgante.rolsuper
                  )
                 FROM pg_catalog.pg_auth_members AS m
                 JOIN pg_catalog.pg_roles AS otorgante
                   ON otorgante.oid = m.grantor
                WHERE m.member IN (login.oid, selector.oid)
                   OR m.roleid IN (login.oid, selector.oid)
                   OR m.grantor IN (login.oid, selector.oid)
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_namespace AS n
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(n.nspacl) AS a
                WHERE n.oid =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_proc AS p
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(p.proacl) AS a
                WHERE p.pronamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_class AS c
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(c.relacl) AS a
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_type AS t
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(t.typacl) AS a
                WHERE t.typnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_default_acl AS d
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(d.defaclacl) AS a
                WHERE d.defaclnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      d.defaclrole IN (login.oid, selector.oid)
                      OR a.grantee IN (login.oid, selector.oid)
                      OR a.grantor IN (login.oid, selector.oid)
                  )
               UNION ALL
               SELECT 1
                 FROM pg_catalog.pg_policy AS politica
                 JOIN pg_catalog.pg_class AS c
                   ON c.oid = politica.polrelid
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      login.oid = ANY (politica.polroles)
                      OR selector.oid = ANY (politica.polroles)
                  )
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_namespace AS n
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(n.nspacl) AS a
                WHERE n.oid =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND a.grantee = 0
           )
    ), manifiestos AS MATERIALIZED (
        SELECT i.*
          FROM identidades AS i
          JOIN pg_catalog.pg_proc AS propia ON propia.oid = i.propia_oid
          JOIN pg_catalog.pg_language AS lenguaje_propio
            ON lenguaje_propio.oid = propia.prolang
          JOIN pg_catalog.pg_proc AS base ON base.oid = i.base_oid
          JOIN pg_catalog.pg_language AS lenguaje_base
            ON lenguaje_base.oid = base.prolang
         WHERE lenguaje_propio.lanname = 'sql'
           AND propia.proowner = i.identidad_oid
           AND propia.prokind = 'f' AND propia.provolatile = 'v'
           AND propia.proparallel = 'u' AND propia.prosecdef
           AND NOT propia.proleakproof AND NOT propia.proisstrict
           AND propia.proretset AND propia.pronargs = 2
           AND propia.pronargdefaults = 0
           AND propia.proargtypes = '25 25'::oidvector
           AND propia.proconfig = ARRAY[
               'search_path=pg_catalog', 'lock_timeout=1s'
           ]::text[]
           AND propia.prosqlbody IS NOT NULL
           AND (
               SELECT pg_catalog.count(*) = 2
                  AND pg_catalog.count(DISTINCT a.grantee) = 2
                  AND pg_catalog.bool_and(
                      a.grantor = i.identidad_oid
                      AND a.grantee = ANY (ARRAY[
                          i.identidad_oid, i.consumidor_oid
                      ])
                      AND a.privilege_type = 'EXECUTE'
                      AND NOT a.is_grantable
                  )
                 FROM pg_catalog.aclexplode(propia.proacl) AS a
           )
           AND lenguaje_base.lanname = 'plpgsql'
           AND base.proowner = i.identidad_oid
           AND base.prokind = 'f' AND base.provolatile = 'v'
           AND base.proparallel = 'u' AND base.prosecdef
           AND NOT base.proleakproof AND NOT base.proisstrict
           AND base.proretset AND base.pronargs = 2
           AND base.pronargdefaults = 0
           AND base.proargtypes = '25 25'::oidvector
           AND base.proconfig =
               ARRAY['search_path=pg_catalog, pg_temp']::text[]
           AND base.prosqlbody IS NULL
           AND pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(base.prosrc, 'UTF8')
           ), 'hex') =
               'e473c7dc0456e97a5865171d1a60cac24ed91fe70d3112c197567b48d46521e3'
           AND pg_catalog.octet_length(base.prosrc) = 8226
           AND (
               SELECT pg_catalog.count(*) = 2
                  AND pg_catalog.count(DISTINCT a.grantee) = 2
                  AND pg_catalog.bool_and(
                      a.grantor = i.identidad_oid
                      AND a.grantee = ANY (ARRAY[
                          i.identidad_oid, i.revalidador_oid
                      ])
                      AND a.privilege_type = 'EXECUTE'
                      AND NOT a.is_grantable
                  )
                 FROM pg_catalog.aclexplode(base.proacl) AS a
           )
           AND (
               SELECT pg_catalog.count(*) = 1
                 FROM pg_catalog.pg_depend AS d
                WHERE d.classid = 'pg_catalog.pg_proc'::regclass
                  AND d.objid = propia.oid AND d.objsubid = 0
                  AND d.refclassid = 'pg_catalog.pg_proc'::regclass
                  AND d.refobjid = base.oid AND d.refobjsubid = 0
                  AND d.deptype = 'n'
           )
           AND EXISTS (
               SELECT 1 FROM pg_catalog.pg_roles AS r
                WHERE r.oid = i.consumidor_oid AND NOT r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreatedb
                  AND NOT r.rolcreaterole AND NOT r.rolinherit
                  AND NOT r.rolreplication AND NOT r.rolbypassrls
                  AND r.rolconnlimit = -1
                  AND r.rolvaliduntil IS NULL AND r.rolconfig IS NULL
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_db_role_setting
                WHERE setrole = i.consumidor_oid
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_auth_members
                WHERE member = i.consumidor_oid
                   OR grantor = i.consumidor_oid
           )
           AND NOT pg_catalog.pg_has_role(
               i.consumidor_oid, i.identidad_oid, 'MEMBER'
           )
           AND pg_catalog.has_schema_privilege(
               i.consumidor_oid,
               'vec_identidad_sesiones_v1', 'USAGE'
           )
           AND NOT pg_catalog.has_schema_privilege(
               i.consumidor_oid,
               'vec_identidad_sesiones_v1', 'CREATE'
           )
           AND (
               SELECT pg_catalog.count(*) = 1
                  AND pg_catalog.bool_and(
                      a.grantor = i.identidad_oid
                      AND a.grantee = i.consumidor_oid
                      AND a.privilege_type = 'USAGE'
                      AND NOT a.is_grantable
                  )
                 FROM pg_catalog.pg_namespace AS n
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(n.nspacl) AS a
                WHERE n.oid =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      a.grantee = i.consumidor_oid
                      OR a.grantor = i.consumidor_oid
                  )
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_proc AS p
                WHERE p.pronamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND p.oid <> i.propia_oid
                  AND pg_catalog.has_function_privilege(
                      i.consumidor_oid, p.oid, 'EXECUTE'
                  )
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_class AS c
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
                  AND (
                      pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'SELECT'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'INSERT'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'UPDATE'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'DELETE'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'TRUNCATE'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'REFERENCES'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'TRIGGER'
                      )
                      OR pg_catalog.has_table_privilege(
                          i.consumidor_oid, c.oid, 'MAINTAIN'
                      )
                      OR pg_catalog.has_any_column_privilege(
                          i.consumidor_oid, c.oid, 'SELECT'
                      )
                      OR pg_catalog.has_any_column_privilege(
                          i.consumidor_oid, c.oid, 'INSERT'
                      )
                      OR pg_catalog.has_any_column_privilege(
                          i.consumidor_oid, c.oid, 'UPDATE'
                      )
                      OR pg_catalog.has_any_column_privilege(
                          i.consumidor_oid, c.oid, 'REFERENCES'
                      )
                  )
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_class AS c
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND c.relkind = 'S'
                  AND (
                      pg_catalog.has_sequence_privilege(
                          i.consumidor_oid, c.oid, 'USAGE'
                      )
                      OR pg_catalog.has_sequence_privilege(
                          i.consumidor_oid, c.oid, 'SELECT'
                      )
                      OR pg_catalog.has_sequence_privilege(
                          i.consumidor_oid, c.oid, 'UPDATE'
                      )
                  )
           )
           AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_type AS t
                WHERE t.typnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND NOT EXISTS (
                      SELECT 1 FROM pg_catalog.pg_type AS e
                       WHERE e.oid = t.typelem AND e.typarray = t.oid
                  )
                  AND pg_catalog.has_type_privilege(
                      i.consumidor_oid, t.oid, 'USAGE'
                  )
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_default_acl AS d
                 CROSS JOIN LATERAL
                      pg_catalog.aclexplode(d.defaclacl) AS a
                WHERE d.defaclnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND (
                      d.defaclrole = i.consumidor_oid
                      OR a.grantor = i.consumidor_oid
                      OR CASE WHEN a.grantee = 0 THEN true ELSE
                           pg_catalog.pg_has_role(
                               i.consumidor_oid, a.grantee, 'MEMBER'
                           )
                         END
                  )
           )
           AND NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_policy AS politica
                 JOIN pg_catalog.pg_class AS c
                   ON c.oid = politica.polrelid
                 CROSS JOIN LATERAL
                      pg_catalog.unnest(politica.polroles) AS r(oid)
                WHERE c.relnamespace =
                      'vec_identidad_sesiones_v1'::regnamespace
                  AND CASE WHEN r.oid = 0 THEN true ELSE
                        pg_catalog.pg_has_role(
                            i.consumidor_oid, r.oid, 'MEMBER'
                        )
                      END
           )
    ), resultado AS MATERIALIZED (
        SELECT r.*, pg_catalog.count(*) OVER () AS total
          FROM manifiestos
          CROSS JOIN LATERAL
               vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
                   p_autenticacion_ref, p_sesion_ref
               ) AS r
    ), ventana AS MATERIALIZED (
        SELECT r.*, CASE
                   WHEN r.autenticacion_verificada_en IS NOT NULL
                    AND pg_catalog.isfinite(
                        r.autenticacion_verificada_en
                    )
                    AND extract(
                        year FROM (
                            r.autenticacion_verificada_en
                            AT TIME ZONE 'UTC'
                        )
                    ) BETWEEN 1 AND 9999
                    AND r.autenticacion_verificada_en <=
                        timestamptz '9999-12-31 23:44:59.999999+00'
                   THEN r.autenticacion_verificada_en +
                        interval '15 minutes'
                   ELSE NULL
               END AS autenticacion_valida_hasta
          FROM resultado AS r
    )
    SELECT r.cuenta_ref, r.metodo_observado, r.garantia_observada,
           LEAST(
               r.sesion_valida_hasta,
               r.autenticacion_valida_hasta
           )
      FROM ventana AS r
     WHERE r.total = 1
       AND r.autenticacion_ref = p_autenticacion_ref
       AND r.sesion_ref = p_sesion_ref
       AND r.cuenta_ref IS NOT NULL
       AND r.metodo_observado IS NOT NULL
       AND (r.garantia_observada = 'alto'
            OR (r.garantia_observada = 'sustancial'
                AND r.metodo_observado = 'certificado'
                AND vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
                    r.politica_garantia_ref,
                    r.politica_garantia_huella_sha256,
                    r.autenticacion_verificada_en)))
       AND r.superficie = 'interna_corporativa'
       AND NOT r.cuenta_privilegiada
       AND r.cuenta_ref = r.cuenta_ordinaria_ref
       AND r.autenticacion_verificada_en IS NOT NULL
       AND r.sesion_valida_hasta IS NOT NULL
       AND r.autenticacion_valida_hasta IS NOT NULL
       AND pg_catalog.isfinite(r.autenticacion_verificada_en)
       AND pg_catalog.isfinite(r.sesion_valida_hasta)
       AND extract(
           year FROM (r.sesion_valida_hasta AT TIME ZONE 'UTC')
       ) BETWEEN 1 AND 9999
       AND pg_catalog.clock_timestamp() < LEAST(
           r.sesion_valida_hasta,
           r.autenticacion_valida_hasta
       );
END;

-- La funcion corporativa original conserva su ACL nominal y verifica el nuevo
-- manifiesto del revalidador. La garantia sustancial solo entra por la fila
-- privada de desarrollo; administracion sigue exigiendo alto.
COMMIT;
