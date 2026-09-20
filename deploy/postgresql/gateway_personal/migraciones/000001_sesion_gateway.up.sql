BEGIN;
SET LOCAL ROLE vec_gateway_personal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_gateway_personal:migracion:000001:v1', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regnamespace('vec_gateway_personal') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'migracion de sesion gateway incompatible: esquema preexistente';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_gateway_personal
    AUTHORIZATION vec_gateway_personal_propietario;
REVOKE ALL ON SCHEMA vec_gateway_personal FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_gateway_personal
    TO vec_gateway_personal_migrador, vec_gateway_personal_ejecutor;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_gateway_personal_propietario
    IN SCHEMA vec_gateway_personal REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_gateway_personal_propietario
    IN SCHEMA vec_gateway_personal REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_gateway_personal_propietario
    IN SCHEMA vec_gateway_personal REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_gateway_personal_propietario
    IN SCHEMA vec_gateway_personal REVOKE USAGE ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_gateway_personal.hash_hex_64_valido(p_valor text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$' AND p_valor <> pg_catalog.repeat('0', 64)
$funcion$;

CREATE FUNCTION vec_gateway_personal.cuenta_ref_valida(p_valor text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor ~ '^cta_[A-Za-z0-9_-]{22,128}$'
$funcion$;

CREATE FUNCTION vec_gateway_personal.rechazar_historia()
RETURNS trigger LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'historia de sesion gateway inmutable';
END
$funcion$;

CREATE TABLE vec_gateway_personal.sesion (
    hash_sesion text PRIMARY KEY,
    cuenta_ref text NOT NULL,
    huella_cert text NOT NULL,
    autenticado_en timestamptz(6) NOT NULL,
    expira_en timestamptz(6) NOT NULL,
    revocada_en timestamptz(6),
    CONSTRAINT sesion_hash_valido CHECK (
        vec_gateway_personal.hash_hex_64_valido(hash_sesion) IS TRUE
    ),
    CONSTRAINT sesion_cuenta_valida CHECK (
        vec_gateway_personal.cuenta_ref_valida(cuenta_ref) IS TRUE
    ),
    CONSTRAINT sesion_huella_valida CHECK (
        vec_gateway_personal.hash_hex_64_valido(huella_cert) IS TRUE
        AND huella_cert <> hash_sesion
    ),
    CONSTRAINT sesion_intervalo_valido CHECK (expira_en > autenticado_en),
    CONSTRAINT sesion_revocacion_valida CHECK (
        revocada_en IS NULL OR revocada_en >= autenticado_en
    )
);
CREATE UNIQUE INDEX sesion_una_vigente_por_cuenta
    ON vec_gateway_personal.sesion (cuenta_ref)
    WHERE revocada_en IS NULL;

CREATE TABLE vec_gateway_personal.evento_sesion (
    evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hash_sesion text NOT NULL REFERENCES vec_gateway_personal.sesion(hash_sesion),
    cuenta_ref text NOT NULL,
    tipo text NOT NULL,
    ocurrido_en timestamptz(6) NOT NULL,
    CONSTRAINT evento_cuenta_valida CHECK (
        vec_gateway_personal.cuenta_ref_valida(cuenta_ref) IS TRUE
    ),
    CONSTRAINT evento_tipo_valido CHECK (tipo IN ('apertura', 'cierre'))
);
CREATE INDEX evento_sesion_hash_orden
    ON vec_gateway_personal.evento_sesion(hash_sesion, evento_id);

CREATE TRIGGER evento_sesion_inmutable
    BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_gateway_personal.evento_sesion
    FOR EACH STATEMENT EXECUTE FUNCTION vec_gateway_personal.rechazar_historia();
CREATE TRIGGER sesion_no_eliminar
    BEFORE DELETE OR TRUNCATE ON vec_gateway_personal.sesion
    FOR EACH STATEMENT EXECUTE FUNCTION vec_gateway_personal.rechazar_historia();

ALTER TABLE vec_gateway_personal.sesion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_gateway_personal.sesion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_gateway_personal.evento_sesion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_gateway_personal.evento_sesion FORCE ROW LEVEL SECURITY;
CREATE POLICY sesion_solo_propietario ON vec_gateway_personal.sesion
    FOR ALL TO vec_gateway_personal_propietario
    USING (true) WITH CHECK (true);
CREATE POLICY evento_solo_propietario ON vec_gateway_personal.evento_sesion
    FOR ALL TO vec_gateway_personal_propietario
    USING (true) WITH CHECK (true);

CREATE FUNCTION vec_gateway_personal.estado_sesion_v1(
    p_sesion vec_gateway_personal.sesion,
    p_ahora timestamptz
)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT p_sesion.revocada_en IS NULL AND p_sesion.expira_en > p_ahora
$funcion$;

CREATE FUNCTION vec_gateway_personal.abrir_sesion_v1(
    hash_sesion text,
    cuenta_ref text,
    huella_cert text,
    autenticado_en timestamptz,
    expira_en timestamptz
)
RETURNS TABLE(
    autenticada boolean,
    expira_hasta timestamptz,
    cuenta_actual_ref text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    existente vec_gateway_personal.sesion%ROWTYPE;
    anterior vec_gateway_personal.sesion%ROWTYPE;
BEGIN
    IF vec_gateway_personal.hash_hex_64_valido(hash_sesion) IS NOT TRUE
       OR vec_gateway_personal.cuenta_ref_valida(cuenta_ref) IS NOT TRUE
       OR vec_gateway_personal.hash_hex_64_valido(huella_cert) IS NOT TRUE
       OR hash_sesion = huella_cert
       OR autenticado_en IS NULL OR expira_en IS NULL
       OR expira_en <= autenticado_en
       OR autenticado_en < pg_catalog.statement_timestamp() - interval '5 minutes'
       OR autenticado_en > pg_catalog.statement_timestamp() + interval '5 minutes'
       OR expira_en > autenticado_en + interval '30 minutes' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'apertura de sesion gateway invalida';
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_gateway_personal:cuenta:' || cuenta_ref, 0)
    );
    SELECT * INTO existente FROM vec_gateway_personal.sesion
     WHERE sesion.hash_sesion = abrir_sesion_v1.hash_sesion;
    IF FOUND THEN
        IF existente.cuenta_ref IS DISTINCT FROM abrir_sesion_v1.cuenta_ref
           OR existente.huella_cert IS DISTINCT FROM abrir_sesion_v1.huella_cert
           OR existente.autenticado_en IS DISTINCT FROM abrir_sesion_v1.autenticado_en
           OR existente.expira_en IS DISTINCT FROM abrir_sesion_v1.expira_en THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'reintento de apertura de sesion incompatible';
        END IF;
        RETURN QUERY SELECT vec_gateway_personal.estado_sesion_v1(existente, autenticado_en), existente.expira_en, existente.cuenta_ref;
        RETURN;
    END IF;

    FOR anterior IN
        SELECT * FROM vec_gateway_personal.sesion
         WHERE sesion.cuenta_ref = abrir_sesion_v1.cuenta_ref
           AND sesion.revocada_en IS NULL
         ORDER BY sesion.autenticado_en, sesion.hash_sesion
         FOR UPDATE
    LOOP
        UPDATE vec_gateway_personal.sesion
           SET revocada_en = GREATEST(abrir_sesion_v1.autenticado_en, anterior.autenticado_en)
         WHERE sesion.hash_sesion = anterior.hash_sesion;
        INSERT INTO vec_gateway_personal.evento_sesion(
            hash_sesion, cuenta_ref, tipo, ocurrido_en
        ) VALUES (
            anterior.hash_sesion, anterior.cuenta_ref, 'cierre',
            GREATEST(abrir_sesion_v1.autenticado_en, anterior.autenticado_en)
        );
    END LOOP;

    INSERT INTO vec_gateway_personal.sesion(
        hash_sesion, cuenta_ref, huella_cert, autenticado_en, expira_en
    ) VALUES (
        abrir_sesion_v1.hash_sesion, abrir_sesion_v1.cuenta_ref,
        abrir_sesion_v1.huella_cert, abrir_sesion_v1.autenticado_en,
        abrir_sesion_v1.expira_en
    );
    INSERT INTO vec_gateway_personal.evento_sesion(
        hash_sesion, cuenta_ref, tipo, ocurrido_en
    ) VALUES (
        abrir_sesion_v1.hash_sesion, abrir_sesion_v1.cuenta_ref,
        'apertura', abrir_sesion_v1.autenticado_en
    );
    RETURN QUERY SELECT true, abrir_sesion_v1.expira_en, abrir_sesion_v1.cuenta_ref;
END
$funcion$;

CREATE FUNCTION vec_gateway_personal.consultar_sesion_v1(
    hash_sesion text,
    ahora timestamptz
)
RETURNS TABLE(
    autenticada boolean,
    expira_hasta timestamptz,
    cuenta_actual_ref text
)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    existente vec_gateway_personal.sesion%ROWTYPE;
BEGIN
    IF vec_gateway_personal.hash_hex_64_valido(hash_sesion) IS NOT TRUE
       OR ahora IS NULL
       OR ahora < pg_catalog.statement_timestamp() - interval '5 minutes'
       OR ahora > pg_catalog.statement_timestamp() + interval '5 minutes' THEN
        RETURN QUERY SELECT false, NULL::timestamptz, NULL::text;
        RETURN;
    END IF;
    SELECT * INTO existente FROM vec_gateway_personal.sesion
     WHERE sesion.hash_sesion = consultar_sesion_v1.hash_sesion;
    IF NOT FOUND THEN
        RETURN QUERY SELECT false, NULL::timestamptz, NULL::text;
        RETURN;
    END IF;
    RETURN QUERY SELECT vec_gateway_personal.estado_sesion_v1(existente, ahora), existente.expira_en, existente.cuenta_ref;
END
$funcion$;

CREATE FUNCTION vec_gateway_personal.cerrar_sesion_v1(
    hash_sesion text,
    ahora timestamptz
)
RETURNS TABLE(
    autenticada boolean,
    expira_hasta timestamptz,
    cuenta_actual_ref text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    existente vec_gateway_personal.sesion%ROWTYPE;
BEGIN
    IF vec_gateway_personal.hash_hex_64_valido(hash_sesion) IS NOT TRUE
       OR ahora IS NULL
       OR ahora < pg_catalog.statement_timestamp() - interval '5 minutes'
       OR ahora > pg_catalog.statement_timestamp() + interval '5 minutes' THEN
        RETURN QUERY SELECT false, NULL::timestamptz, NULL::text;
        RETURN;
    END IF;
    SELECT * INTO existente FROM vec_gateway_personal.sesion
     WHERE sesion.hash_sesion = cerrar_sesion_v1.hash_sesion
     FOR UPDATE;
    IF NOT FOUND THEN
        RETURN QUERY SELECT false, NULL::timestamptz, NULL::text;
        RETURN;
    END IF;
    IF existente.revocada_en IS NULL THEN
        UPDATE vec_gateway_personal.sesion
           SET revocada_en = GREATEST(ahora, existente.autenticado_en)
         WHERE sesion.hash_sesion = existente.hash_sesion;
        INSERT INTO vec_gateway_personal.evento_sesion(
            hash_sesion, cuenta_ref, tipo, ocurrido_en
        ) VALUES (
            existente.hash_sesion, existente.cuenta_ref, 'cierre',
            GREATEST(ahora, existente.autenticado_en)
        );
    END IF;
    RETURN QUERY SELECT false, existente.expira_en, existente.cuenta_ref;
END
$funcion$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_gateway_personal FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_gateway_personal FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_gateway_personal FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_gateway_personal.abrir_sesion_v1(
    text, text, text, timestamptz, timestamptz
) FROM vec_gateway_personal_migrador;
REVOKE ALL ON FUNCTION vec_gateway_personal.consultar_sesion_v1(text, timestamptz)
    FROM vec_gateway_personal_migrador;
REVOKE ALL ON FUNCTION vec_gateway_personal.cerrar_sesion_v1(text, timestamptz)
    FROM vec_gateway_personal_migrador;
GRANT EXECUTE ON FUNCTION vec_gateway_personal.abrir_sesion_v1(
    text, text, text, timestamptz, timestamptz
) TO vec_gateway_personal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_gateway_personal.consultar_sesion_v1(text, timestamptz)
    TO vec_gateway_personal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_gateway_personal.cerrar_sesion_v1(text, timestamptz)
    TO vec_gateway_personal_ejecutor;
COMMIT;
