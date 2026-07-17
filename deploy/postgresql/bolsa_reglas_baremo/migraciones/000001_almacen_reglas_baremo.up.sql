-- Almacen autoritativo append-only de reglas gobernadas. PostgreSQL conserva
-- bytes canonicos y huellas; la restauracion semantica pertenece al dominio Go.
BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:migracion:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_reglas_baremo') IS NOT NULL
       OR to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_reglas_baremo_v1(jsonb,bytea,bytea,text,text,text,text,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar almacen de reglas de baremo';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_bolsa_reglas_baremo
    AUTHORIZATION vec_bolsa_reglas_baremo_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_reglas_baremo FROM PUBLIC;

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_reglas_baremo_propietario
    IN SCHEMA vec_bolsa_reglas_baremo
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_reglas_baremo_propietario
    IN SCHEMA vec_bolsa_reglas_baremo
    REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_reglas_baremo_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_reglas_baremo_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_bolsa_reglas_baremo.referencia_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND octet_length(p_valor) BETWEEN 1 AND 512
       AND (p_valor COLLATE "C") ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$'
$funcion$;

CREATE FUNCTION vec_bolsa_reglas_baremo.huella_sha256_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_reglas_baremo.version_valida(p_valor numeric)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_valor BETWEEN 1 AND 1000000000
       AND trunc(p_valor) = p_valor
$funcion$;

CREATE FUNCTION vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'historia inmutable de reglas de baremo';
END
$funcion$;

CREATE FUNCTION vec_bolsa_reglas_baremo.rechazar_borrado_o_truncado()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'control durable de reglas de baremo no eliminable';
END
$funcion$;

CREATE TABLE vec_bolsa_reglas_baremo.configuracion_tenant (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    tenant_id text NOT NULL UNIQUE,
    instalada_en timestamptz(6) NOT NULL,
    CONSTRAINT configuracion_tenant_valida CHECK (
        tenant_id = 'diputacion_granada' AND isfinite(instalada_en)
    )
);
INSERT INTO vec_bolsa_reglas_baremo.configuracion_tenant(
    control_id, tenant_id, instalada_en
) VALUES (true, 'diputacion_granada', statement_timestamp());

CREATE TABLE vec_bolsa_reglas_baremo.contenido_reglas_baremo (
    tenant_id text NOT NULL,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    huella_contenido_sha256 text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (tenant_id, contenido_ref, contenido_version),
    UNIQUE (
        tenant_id, contenido_ref, contenido_version,
        huella_contenido_sha256
    ),
    FOREIGN KEY (tenant_id)
        REFERENCES vec_bolsa_reglas_baremo.configuracion_tenant(tenant_id),
    CONSTRAINT contenido_identidad_valida CHECK (
        vec_bolsa_reglas_baremo.referencia_valida(contenido_ref)
        AND vec_bolsa_reglas_baremo.version_valida(contenido_version)
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_contenido_sha256
        )
        AND isfinite(creada_en)
    )
);

CREATE TABLE vec_bolsa_reglas_baremo.version_reglas_baremo (
    tenant_id text NOT NULL,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    huella_contenido_sha256 text NOT NULL,
    revision numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    version_canonica bytea NOT NULL,
    huella_estado_sha256 text NOT NULL,
    operacion_origen text NOT NULL,
    intencion_ref text NOT NULL,
    intencion_version numeric(20, 0) NOT NULL,
    intencion_huella_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (
        tenant_id, contenido_ref, contenido_version, revision
    ),
    UNIQUE (
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ),
    UNIQUE (tenant_id, huella_estado_sha256),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version,
        huella_contenido_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.contenido_reglas_baremo(
        tenant_id, contenido_ref, contenido_version,
        huella_contenido_sha256
    ),
    CONSTRAINT version_proyecciones_validas CHECK (
        vec_bolsa_reglas_baremo.version_valida(revision)
        AND vec_bolsa_reglas_baremo.referencia_valida(intencion_ref)
        AND vec_bolsa_reglas_baremo.version_valida(intencion_version)
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            intencion_huella_sha256
        )
        AND isfinite(registrada_en)
    ),
    CONSTRAINT version_bytes_huella CHECK (
        octet_length(version_canonica) BETWEEN 2 AND 5242880
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_estado_sha256
        )
        AND encode(sha256(version_canonica), 'hex') =
            huella_estado_sha256
    ),
    CONSTRAINT version_transicion_cerrada CHECK (
        (operacion_origen = 'alta_borrador'
         AND estado = 'borrador' AND revision = 1)
        OR (operacion_origen = 'publicar'
            AND estado = 'publicada' AND revision = 2)
        OR (operacion_origen = 'activar'
            AND estado = 'activa' AND revision = 3)
        OR (operacion_origen = 'sustituir'
            AND estado = 'sustituida' AND revision = 4)
        OR (operacion_origen = 'retirar'
            AND estado = 'retirada' AND revision = 4)
        OR (operacion_origen = 'descartar'
            AND estado = 'descartada' AND revision = 2)
    )
);

CREATE INDEX version_reglas_baremo_estado
    ON vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, estado, registrada_en DESC
    );

CREATE TABLE vec_bolsa_reglas_baremo.estado_actual (
    tenant_id text NOT NULL,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    huella_contenido_sha256 text NOT NULL,
    revision numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (tenant_id, contenido_ref, contenido_version),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version,
        huella_contenido_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.contenido_reglas_baremo(
        tenant_id, contenido_ref, contenido_version,
        huella_contenido_sha256
    ),
    CONSTRAINT estado_actual_valido CHECK (
        vec_bolsa_reglas_baremo.version_valida(revision)
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_estado_sha256
        )
        AND isfinite(actualizada_en)
    )
);

CREATE TABLE vec_bolsa_reglas_baremo.uso_decision (
    tenant_id text NOT NULL,
    decision_ref text NOT NULL,
    consumo_autorizacion_ref text NOT NULL,
    consumo_autorizacion_version numeric(20, 0) NOT NULL DEFAULT 1,
    huella_consumo_autorizacion_sha256 text NOT NULL,
    principal_ref text NOT NULL,
    operacion text NOT NULL,
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    correlacion_ref text NOT NULL,
    huella_decision_sha256 text NOT NULL,
    huella_contexto_recurso_sha256 text NOT NULL,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    revision numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    PRIMARY KEY (decision_ref),
    UNIQUE (tenant_id, consumo_autorizacion_ref),
    UNIQUE (
        tenant_id, decision_ref, consumo_autorizacion_ref
    ),
    FOREIGN KEY (decision_ref) REFERENCES
        vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
            decision_ref
        ),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ),
    CONSTRAINT uso_referencias_validas CHECK (
        tenant_id = 'diputacion_granada'
        AND vec_bolsa_reglas_baremo.referencia_valida(decision_ref)
        AND vec_bolsa_reglas_baremo.referencia_valida(
            consumo_autorizacion_ref
        )
        AND consumo_autorizacion_version = 1
        AND vec_bolsa_reglas_baremo.referencia_valida(principal_ref)
        AND vec_bolsa_reglas_baremo.referencia_valida(recurso_ref)
        AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
    ),
    CONSTRAINT uso_operacion_cerrada CHECK (
        operacion IN (
            'alta_borrador', 'publicar', 'activar', 'sustituir',
            'retirar', 'descartar', 'consultar_version_exacta'
        )
    ),
    CONSTRAINT uso_huellas_validas CHECK (
        vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_consumo_autorizacion_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_decision_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_contexto_recurso_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_efecto_sha256
        )
        AND isfinite(consumida_en)
    )
);

CREATE TABLE vec_bolsa_reglas_baremo.uso_prueba_transicion (
    tenant_id text NOT NULL,
    prueba_ref text NOT NULL,
    prueba_version numeric(20, 0) NOT NULL,
    prueba_huella_sha256 text NOT NULL,
    consumo_prueba_ref text NOT NULL,
    consumo_prueba_version numeric(20, 0) NOT NULL DEFAULT 1,
    huella_consumo_prueba_sha256 text NOT NULL,
    intencion_ref text NOT NULL,
    intencion_version numeric(20, 0) NOT NULL,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    revision numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    PRIMARY KEY (
        tenant_id, prueba_ref, prueba_version, prueba_huella_sha256
    ),
    UNIQUE (tenant_id, consumo_prueba_ref),
    UNIQUE (
        tenant_id, consumo_prueba_ref, consumo_prueba_version,
        huella_consumo_prueba_sha256
    ),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ),
    CONSTRAINT uso_prueba_referencias_validas CHECK (
        vec_bolsa_reglas_baremo.referencia_valida(prueba_ref)
        AND vec_bolsa_reglas_baremo.version_valida(prueba_version)
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            prueba_huella_sha256
        )
        AND vec_bolsa_reglas_baremo.referencia_valida(consumo_prueba_ref)
        AND consumo_prueba_version = 1
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_consumo_prueba_sha256
        )
        AND vec_bolsa_reglas_baremo.referencia_valida(intencion_ref)
        AND vec_bolsa_reglas_baremo.version_valida(intencion_version)
        AND isfinite(consumida_en)
    )
);

CREATE TABLE vec_bolsa_reglas_baremo.auditoria (
    tenant_id text NOT NULL,
    secuencia bigint NOT NULL,
    auditoria_ref text NOT NULL,
    auditoria_version numeric(20, 0) NOT NULL DEFAULT 1,
    decision_ref text NOT NULL,
    consumo_autorizacion_ref text NOT NULL,
    operacion text NOT NULL,
    registro_canonico bytea NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_auditoria_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (tenant_id, secuencia),
    UNIQUE (tenant_id, auditoria_ref),
    UNIQUE (
        tenant_id, auditoria_ref, auditoria_version,
        huella_auditoria_sha256
    ),
    FOREIGN KEY (
        tenant_id, decision_ref, consumo_autorizacion_ref
    ) REFERENCES vec_bolsa_reglas_baremo.uso_decision(
        tenant_id, decision_ref, consumo_autorizacion_ref
    ),
    CONSTRAINT auditoria_valida CHECK (
        secuencia > 0
        AND vec_bolsa_reglas_baremo.referencia_valida(auditoria_ref)
        AND auditoria_version = 1
        AND operacion IN (
            'alta_borrador', 'publicar', 'activar', 'sustituir',
            'retirar', 'descartar', 'consultar_version_exacta'
        )
        AND octet_length(registro_canonico) BETWEEN 2 AND 1048576
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_anterior_sha256
        )
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_auditoria_sha256
        )
        AND encode(sha256(
            decode(huella_anterior_sha256, 'hex') || registro_canonico
        ), 'hex') = huella_auditoria_sha256
        AND isfinite(registrada_en)
    )
);

CREATE TABLE vec_bolsa_reglas_baremo.auditoria_actual (
    tenant_id text PRIMARY KEY,
    ultima_secuencia bigint NOT NULL,
    ultima_huella_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (tenant_id)
        REFERENCES vec_bolsa_reglas_baremo.configuracion_tenant(tenant_id),
    CONSTRAINT auditoria_actual_valida CHECK (
        ultima_secuencia >= 0
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            ultima_huella_sha256
        )
        AND isfinite(actualizada_en)
    )
);
INSERT INTO vec_bolsa_reglas_baremo.auditoria_actual(
    tenant_id, ultima_secuencia, ultima_huella_sha256, actualizada_en
) VALUES (
    'diputacion_granada', 0, repeat('0', 64), statement_timestamp()
);

CREATE TABLE vec_bolsa_reglas_baremo.outbox (
    tenant_id text NOT NULL,
    outbox_ref text NOT NULL,
    outbox_version numeric(20, 0) NOT NULL DEFAULT 1,
    ruta text NOT NULL,
    esquema_evento text NOT NULL,
    evento_canonico bytea NOT NULL,
    huella_evento_sha256 text NOT NULL,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    revision numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (tenant_id, outbox_ref),
    UNIQUE (
        tenant_id, outbox_ref, outbox_version, huella_evento_sha256
    ),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ),
    CONSTRAINT outbox_valida CHECK (
        vec_bolsa_reglas_baremo.referencia_valida(outbox_ref)
        AND outbox_version = 1
        AND ruta = 'bolsa.reglas_baremo.estado_confirmado.v1'
        AND esquema_evento =
            'vec.bolsa.reglas-baremo.estado-confirmado.v1'
        AND octet_length(evento_canonico) BETWEEN 2 AND 1048576
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_evento_sha256
        )
        AND encode(sha256(evento_canonico), 'hex') =
            huella_evento_sha256
        AND isfinite(creada_en)
    )
);

CREATE TABLE vec_bolsa_reglas_baremo.intencion_confirmada (
    tenant_id text NOT NULL,
    intencion_ref text NOT NULL,
    intencion_version numeric(20, 0) NOT NULL,
    intencion_huella_sha256 text NOT NULL,
    operacion text NOT NULL,
    esperado_revision numeric(20, 0),
    esperado_huella_estado_sha256 text,
    contenido_ref text NOT NULL,
    contenido_version numeric(20, 0) NOT NULL,
    huella_contenido_sha256 text NOT NULL,
    resultado_revision numeric(20, 0) NOT NULL,
    resultado_estado text NOT NULL,
    resultado_huella_estado_sha256 text NOT NULL,
    transaccion_ref text NOT NULL,
    transaccion_version numeric(20, 0) NOT NULL DEFAULT 1,
    huella_transaccion_sha256 text NOT NULL,
    decision_ref text NOT NULL,
    consumo_autorizacion_ref text NOT NULL,
    prueba_consumo_ref text,
    prueba_consumo_version numeric(20, 0),
    prueba_consumo_huella_sha256 text,
    auditoria_ref text NOT NULL,
    outbox_ref text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (tenant_id, intencion_ref, intencion_version),
    UNIQUE (
        tenant_id, transaccion_ref, transaccion_version,
        huella_transaccion_sha256
    ),
    FOREIGN KEY (
        tenant_id, contenido_ref, contenido_version, resultado_revision,
        resultado_huella_estado_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version, revision,
        huella_estado_sha256
    ),
    FOREIGN KEY (
        tenant_id, decision_ref, consumo_autorizacion_ref
    ) REFERENCES vec_bolsa_reglas_baremo.uso_decision(
        tenant_id, decision_ref, consumo_autorizacion_ref
    ),
    FOREIGN KEY (tenant_id, auditoria_ref)
        REFERENCES vec_bolsa_reglas_baremo.auditoria(
            tenant_id, auditoria_ref
        ),
    FOREIGN KEY (tenant_id, outbox_ref)
        REFERENCES vec_bolsa_reglas_baremo.outbox(tenant_id, outbox_ref),
    FOREIGN KEY (
        tenant_id, prueba_consumo_ref, prueba_consumo_version,
        prueba_consumo_huella_sha256
    ) REFERENCES vec_bolsa_reglas_baremo.uso_prueba_transicion(
        tenant_id, consumo_prueba_ref, consumo_prueba_version,
        huella_consumo_prueba_sha256
    ),
    CONSTRAINT intencion_valida CHECK (
        vec_bolsa_reglas_baremo.referencia_valida(intencion_ref)
        AND vec_bolsa_reglas_baremo.version_valida(intencion_version)
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            intencion_huella_sha256
        )
        AND vec_bolsa_reglas_baremo.version_valida(resultado_revision)
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_contenido_sha256
        )
        AND vec_bolsa_reglas_baremo.referencia_valida(transaccion_ref)
        AND transaccion_version = 1
        AND vec_bolsa_reglas_baremo.huella_sha256_valida(
            huella_transaccion_sha256
        )
        AND isfinite(confirmada_en)
    ),
    CONSTRAINT intencion_esperado_completo CHECK (
        (esperado_revision IS NULL
         AND esperado_huella_estado_sha256 IS NULL)
        OR
        (vec_bolsa_reglas_baremo.version_valida(esperado_revision)
         AND vec_bolsa_reglas_baremo.huella_sha256_valida(
             esperado_huella_estado_sha256
         ))
    ),
    CONSTRAINT intencion_prueba_completa CHECK (
        (prueba_consumo_ref IS NULL
         AND prueba_consumo_version IS NULL
         AND prueba_consumo_huella_sha256 IS NULL)
        OR
        (vec_bolsa_reglas_baremo.referencia_valida(prueba_consumo_ref)
         AND prueba_consumo_version = 1
         AND vec_bolsa_reglas_baremo.huella_sha256_valida(
             prueba_consumo_huella_sha256
         ))
    )
);

DO $inmutabilidad$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'configuracion_tenant', 'contenido_reglas_baremo',
        'version_reglas_baremo', 'uso_decision',
        'uso_prueba_transicion', 'auditoria', 'outbox',
        'intencion_confirmada'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER impedir_mutacion BEFORE UPDATE OR DELETE ON vec_bolsa_reglas_baremo.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable()',
            tabla
        );
        EXECUTE format(
            'CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON vec_bolsa_reglas_baremo.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable()',
            tabla
        );
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY['estado_actual', 'auditoria_actual'] LOOP
        EXECUTE format(
            'CREATE TRIGGER impedir_borrado BEFORE DELETE ON vec_bolsa_reglas_baremo.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_reglas_baremo.rechazar_borrado_o_truncado()',
            tabla
        );
        EXECUTE format(
            'CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON vec_bolsa_reglas_baremo.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_reglas_baremo.rechazar_borrado_o_truncado()',
            tabla
        );
    END LOOP;
END
$inmutabilidad$;

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'configuracion_tenant', 'contenido_reglas_baremo',
        'version_reglas_baremo', 'estado_actual', 'uso_decision',
        'uso_prueba_transicion', 'auditoria', 'auditoria_actual',
        'outbox', 'intencion_confirmada'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_reglas_baremo.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_reglas_baremo.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_bolsa_reglas_baremo.%I FOR ALL TO vec_bolsa_reglas_baremo_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla,
            'vec_bolsa_reglas_baremo_propietario',
            'vec_bolsa_reglas_baremo_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_reglas_baremo FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_bolsa_reglas_baremo FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_reglas_baremo FROM PUBLIC;
COMMIT;
