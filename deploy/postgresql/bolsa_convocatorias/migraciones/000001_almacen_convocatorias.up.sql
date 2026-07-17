-- Almacen durable y cerrado por defecto para versiones gobernadas de
-- convocatoria, atestaciones, consumos y auditoria encadenada.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_convocatorias') IS NOT NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar el almacen de convocatorias';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_bolsa_convocatorias
    AUTHORIZATION vec_bolsa_convocatorias_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_convocatorias FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    IN SCHEMA vec_bolsa_convocatorias REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    IN SCHEMA vec_bolsa_convocatorias REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_bolsa_convocatorias.texto_opaco_valido(
    p_valor text, p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor = btrim(p_valor)
       AND p_valor !~ '[[:space:][:cntrl:]]'
       AND strpos(p_valor, '*') = 0
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.huella_sha256_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.instante_utc_microsegundo_valido(
    p_valor text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    convertido timestamptz;
BEGIN
    IF p_valor IS NULL OR p_valor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$' THEN
        RETURN false;
    END IF;
    convertido := p_valor::timestamptz;
    RETURN to_char(convertido AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') = p_valor;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'objeto inmutable';
END
$funcion$;

CREATE TABLE vec_bolsa_convocatorias.version_convocatoria (
    convocatoria_id text NOT NULL,
    secuencia bigint NOT NULL,
    referencia text NOT NULL,
    estado text NOT NULL,
    version_canonica bytea NOT NULL,
    huella_version_sha256 text NOT NULL,
    instancia_flujo_ref text,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (convocatoria_id, secuencia),
    UNIQUE (referencia),
    CONSTRAINT version_secuencia_positiva CHECK (secuencia > 0),
    CONSTRAINT version_identidad_exacta CHECK (
        vec_bolsa_convocatorias.texto_opaco_valido(convocatoria_id, 480)
        AND strpos(convocatoria_id, '#') = 0
        AND referencia = convocatoria_id || '#' || secuencia::text
        AND vec_bolsa_convocatorias.texto_opaco_valido(referencia, 512)
    ),
    CONSTRAINT version_estado_cerrado CHECK (
        estado IN ('borrador', 'publicada', 'sustituida', 'retirada')
    ),
    CONSTRAINT version_bytes_huella CHECK (
        octet_length(version_canonica) BETWEEN 2 AND 33554432
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_version_sha256
        )
        AND encode(sha256(version_canonica), 'hex') =
            huella_version_sha256
    ),
    CONSTRAINT version_flujo_ref_valida CHECK (
        instancia_flujo_ref IS NULL OR
        vec_bolsa_convocatorias.texto_opaco_valido(instancia_flujo_ref, 512)
    )
);

CREATE TABLE vec_bolsa_convocatorias.instancia_flujo_version (
    convocatoria_id text NOT NULL,
    secuencia bigint NOT NULL,
    instancia_flujo_ref text NOT NULL,
    instancia_flujo_canonica bytea NOT NULL,
    huella_instancia_flujo_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (convocatoria_id, secuencia),
    FOREIGN KEY (convocatoria_id, secuencia)
        REFERENCES vec_bolsa_convocatorias.version_convocatoria(
            convocatoria_id, secuencia
        ),
    CONSTRAINT instancia_flujo_identidad_valida CHECK (
        vec_bolsa_convocatorias.texto_opaco_valido(
            instancia_flujo_ref, 512
        )
    ),
    CONSTRAINT instancia_flujo_bytes_huella CHECK (
        octet_length(instancia_flujo_canonica) BETWEEN 2 AND 8388608
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_instancia_flujo_sha256
        )
        AND encode(sha256(instancia_flujo_canonica), 'hex') =
            huella_instancia_flujo_sha256
    )
);

CREATE TABLE vec_bolsa_convocatorias.atestacion_autorizacion_version (
    decision_ref text NOT NULL,
    atestacion_ref text NOT NULL,
    version bigint NOT NULL,
    estado text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    evidencia_canonica bytea NOT NULL,
    huella_evidencia_sha256 text NOT NULL,
    sobre_cose_sign1 bytea NOT NULL,
    huella_sobre_sha256 text NOT NULL,
    clave_id text NOT NULL,
    revision_confianza text NOT NULL,
    verificada_en timestamptz(6) NOT NULL,
    valida_desde timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version),
    UNIQUE (decision_ref, atestacion_ref, version, estado),
    CONSTRAINT atestacion_version_positiva CHECK (version > 0),
    CONSTRAINT atestacion_estado_cerrado CHECK (
        estado IN ('activa', 'revocada')
    ),
    CONSTRAINT atestacion_referencias_validas CHECK (
        vec_bolsa_convocatorias.texto_opaco_valido(decision_ref, 512)
        AND vec_bolsa_convocatorias.texto_opaco_valido(atestacion_ref, 512)
        AND vec_bolsa_convocatorias.texto_opaco_valido(clave_id, 512)
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            revision_confianza, 128
        )
    ),
    CONSTRAINT atestacion_huellas_validas CHECK (
        vec_bolsa_convocatorias.huella_sha256_valida(
            huella_decision_sha256
        )
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_evidencia_sha256
        )
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_sobre_sha256
        )
        AND encode(sha256(evidencia_canonica), 'hex') =
            huella_evidencia_sha256
        AND encode(sha256(sobre_cose_sign1), 'hex') = huella_sobre_sha256
    ),
    CONSTRAINT atestacion_tamanos_acotados CHECK (
        octet_length(evidencia_canonica) BETWEEN 1 AND 2097152
        AND octet_length(sobre_cose_sign1) BETWEEN 16 AND 528384
    ),
    CONSTRAINT atestacion_ventana_valida CHECK (
        valida_desde <= verificada_en
        AND verificada_en < valida_hasta
        AND registrada_en >= verificada_en
    )
);

CREATE TABLE vec_bolsa_convocatorias.atestacion_autorizacion_actual (
    decision_ref text PRIMARY KEY,
    atestacion_ref text NOT NULL,
    version bigint NOT NULL,
    estado text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref, atestacion_ref, version, estado)
        REFERENCES vec_bolsa_convocatorias.atestacion_autorizacion_version(
            decision_ref, atestacion_ref, version, estado
        ),
    CONSTRAINT atestacion_actual_estado CHECK (
        estado IN ('activa', 'revocada')
    )
);

CREATE TABLE vec_bolsa_convocatorias.uso_decision_consulta (
    decision_ref text PRIMARY KEY,
    consumo_autorizacion_ref text NOT NULL UNIQUE,
    principal_ref text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    convocatoria_id text NOT NULL,
    secuencia bigint NOT NULL,
    huella_decision_sha256 text NOT NULL,
    huella_recurso_sha256 text NOT NULL,
    atestacion_ref text NOT NULL,
    atestacion_version bigint NOT NULL,
    huella_atestacion_sha256 text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    FOREIGN KEY (convocatoria_id, secuencia)
        REFERENCES vec_bolsa_convocatorias.version_convocatoria(
            convocatoria_id, secuencia
        ),
    FOREIGN KEY (decision_ref, atestacion_ref, atestacion_version)
        REFERENCES vec_bolsa_convocatorias.atestacion_autorizacion_version(
            decision_ref, atestacion_ref, version
        ),
    CONSTRAINT uso_referencias_validas CHECK (
        vec_bolsa_convocatorias.texto_opaco_valido(decision_ref, 512)
        AND vec_bolsa_convocatorias.texto_opaco_valido(
            consumo_autorizacion_ref, 512
        )
        AND vec_bolsa_convocatorias.texto_opaco_valido(principal_ref, 512)
        AND recurso_ref = convocatoria_id || '#' || secuencia::text
    ),
    CONSTRAINT uso_accion_cerrada CHECK (
        accion IN (
            'bolsa.convocatoria.version.consultar',
            'bolsa.convocatoria.version_con_flujo.consultar'
        )
    ),
    CONSTRAINT uso_huellas_validas CHECK (
        vec_bolsa_convocatorias.huella_sha256_valida(
            huella_decision_sha256
        )
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_recurso_sha256
        )
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_atestacion_sha256
        )
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_efecto_sha256
        )
    )
);

CREATE TABLE vec_bolsa_convocatorias.auditoria (
    auditoria_ref text PRIMARY KEY,
    secuencia bigint NOT NULL UNIQUE,
    consumo_autorizacion_ref text NOT NULL UNIQUE,
    registro_canonico bytea NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_auditoria_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (consumo_autorizacion_ref)
        REFERENCES vec_bolsa_convocatorias.uso_decision_consulta(
            consumo_autorizacion_ref
        ),
    CONSTRAINT auditoria_secuencia_positiva CHECK (secuencia > 0),
    CONSTRAINT auditoria_referencia_valida CHECK (
        vec_bolsa_convocatorias.texto_opaco_valido(auditoria_ref, 512)
    ),
    CONSTRAINT auditoria_huellas_validas CHECK (
        vec_bolsa_convocatorias.huella_sha256_valida(
            huella_anterior_sha256
        )
        AND vec_bolsa_convocatorias.huella_sha256_valida(
            huella_auditoria_sha256
        )
        AND octet_length(registro_canonico) BETWEEN 2 AND 1048576
        AND encode(sha256(registro_canonico), 'hex') =
            huella_auditoria_sha256
    )
);

CREATE TABLE vec_bolsa_convocatorias.auditoria_actual (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    ultima_secuencia bigint NOT NULL CHECK (ultima_secuencia >= 0),
    ultima_huella_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CONSTRAINT auditoria_actual_huella_valida CHECK (
        vec_bolsa_convocatorias.huella_sha256_valida(
            ultima_huella_sha256
        )
    )
);

INSERT INTO vec_bolsa_convocatorias.auditoria_actual(
    control_id, ultima_secuencia, ultima_huella_sha256, actualizada_en
) VALUES (true, 0, repeat('0', 64), statement_timestamp());

CREATE FUNCTION vec_bolsa_convocatorias.validar_avance_atestacion_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.decision_ref IS DISTINCT FROM OLD.decision_ref
       OR NEW.version <> OLD.version + 1
       OR NEW.actualizada_en <= OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de atestacion invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_bolsa_convocatorias.validar_avance_auditoria_actual()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.control_id IS DISTINCT FROM OLD.control_id
       OR NEW.ultima_secuencia <> OLD.ultima_secuencia + 1
       OR NEW.actualizada_en < OLD.actualizada_en
       OR NOT EXISTS (
           SELECT 1 FROM vec_bolsa_convocatorias.auditoria AS registro
            WHERE registro.secuencia = NEW.ultima_secuencia
              AND registro.huella_anterior_sha256 = OLD.ultima_huella_sha256
              AND registro.huella_auditoria_sha256 = NEW.ultima_huella_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de cadena de auditoria invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER atestacion_actual_avance
    BEFORE UPDATE ON vec_bolsa_convocatorias.atestacion_autorizacion_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.validar_avance_atestacion_actual();
CREATE TRIGGER auditoria_actual_avance
    BEFORE UPDATE ON vec_bolsa_convocatorias.auditoria_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_convocatorias.validar_avance_auditoria_actual();

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'version_convocatoria', 'instancia_flujo_version',
        'atestacion_autorizacion_version', 'uso_decision_consulta',
        'auditoria'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_convocatorias.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_bolsa_convocatorias.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY[
        'atestacion_autorizacion_actual', 'auditoria_actual'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_no_eliminar BEFORE DELETE OR TRUNCATE ON vec_bolsa_convocatorias.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
END
$protecciones$;

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'version_convocatoria', 'instancia_flujo_version',
        'atestacion_autorizacion_version',
        'atestacion_autorizacion_actual', 'uso_decision_consulta',
        'auditoria', 'auditoria_actual'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_convocatorias.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_convocatorias.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_bolsa_convocatorias.%I FOR ALL TO vec_bolsa_convocatorias_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_convocatorias_propietario',
            'vec_bolsa_convocatorias_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_convocatorias FROM PUBLIC;
DO $cerrar_tipos_implicitos$
DECLARE
    tipo record;
BEGIN
    FOR tipo IN
        SELECT espacio.nspname, definicion.typname
          FROM pg_catalog.pg_type AS definicion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = definicion.typnamespace
         WHERE espacio.nspname = 'vec_bolsa_convocatorias'
           AND definicion.typelem = 0
           AND definicion.typisdefined
    LOOP
        EXECUTE format(
            'REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC, %I, %I, %I',
            tipo.nspname, tipo.typname,
            'vec_bolsa_convocatorias_ejecutor_consulta',
            'vec_bolsa_convocatorias_proyector_gobierno',
            'vec_bolsa_convocatorias_registrador_atestacion'
        );
    END LOOP;
END
$cerrar_tipos_implicitos$;
COMMIT;
