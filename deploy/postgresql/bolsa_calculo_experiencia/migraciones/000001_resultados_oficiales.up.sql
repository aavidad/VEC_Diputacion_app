-- Resultado oficial, idempotencia, autorizaciones, recibos, auditoria y
-- outbox del calculo de experiencia. No ofrece procedimientos privilegiados:
-- la composicion atomica corresponde al adaptador PostgreSQL.
BEGIN;
SET LOCAL ROLE vec_bolsa_calculo_experiencia_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_calculo_experiencia') IS NOT NULL
       OR to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_calculo_experiencia_v1(text,text,text,text,text,text,text,text)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar resultados oficiales';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_bolsa_calculo_experiencia
    AUTHORIZATION vec_bolsa_calculo_experiencia_propietario;
REVOKE ALL ON SCHEMA vec_bolsa_calculo_experiencia FROM PUBLIC;

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_calculo_experiencia_propietario
    IN SCHEMA vec_bolsa_calculo_experiencia
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_calculo_experiencia_propietario
    IN SCHEMA vec_bolsa_calculo_experiencia
    REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_calculo_experiencia_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_calculo_experiencia_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_bolsa_calculo_experiencia.texto_opaco_valido(
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

CREATE FUNCTION vec_bolsa_calculo_experiencia.huella_sha256_valida(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_calculo_experiencia.indice_hmac_sha256_valido(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^[0-9a-f]{64}$'
$funcion$;

CREATE FUNCTION vec_bolsa_calculo_experiencia.instante_utc_valido(
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
    RETURN to_char(
        convertido AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    ) = p_valor;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE DOMAIN vec_bolsa_calculo_experiencia.instante_utc AS text
    CHECK (vec_bolsa_calculo_experiencia.instante_utc_valido(VALUE));
REVOKE ALL ON TYPE vec_bolsa_calculo_experiencia.instante_utc FROM PUBLIC;

CREATE FUNCTION vec_bolsa_calculo_experiencia.rechazar_mutacion_inmutable()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'objeto inmutable';
END
$funcion$;

-- Fase 1: instalacion de un solo inquilino. No se usa un GUC modificable por
-- la sesion como frontera de aislamiento.
CREATE TABLE vec_bolsa_calculo_experiencia.configuracion_tenant (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    tenant_id text NOT NULL UNIQUE,
    instalada_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    CONSTRAINT configuracion_tenant_valida CHECK (
        vec_bolsa_calculo_experiencia.texto_opaco_valido(tenant_id, 128)
    )
);
INSERT INTO vec_bolsa_calculo_experiencia.configuracion_tenant(
    control_id, tenant_id, instalada_en
) VALUES (
    true, 'diputacion_granada',
    to_char(
        statement_timestamp() AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    )
);
CREATE TRIGGER impedir_mutacion
    BEFORE UPDATE OR DELETE
    ON vec_bolsa_calculo_experiencia.configuracion_tenant
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_calculo_experiencia.rechazar_mutacion_inmutable();
CREATE TRIGGER impedir_truncado
    BEFORE TRUNCATE
    ON vec_bolsa_calculo_experiencia.configuracion_tenant
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_bolsa_calculo_experiencia.rechazar_mutacion_inmutable();

CREATE TABLE vec_bolsa_calculo_experiencia.resultado_oficial (
    tenant_id text NOT NULL,
    resultado_ref text NOT NULL,
    esquema_resultado text NOT NULL,
    resultado_canonico bytea NOT NULL,
    huella_resultado_sha256 text NOT NULL,
    esquema_clave_semantica text NOT NULL,
    clave_semantica_publica bytea NOT NULL,
    huella_clave_semantica_sha256 text NOT NULL,
    generacion_clave_hmac integer NOT NULL,
    indice_efecto_hmac_sha256 text NOT NULL,
    esquema_selector_fuente text NOT NULL,
    selector_fuente_canonico bytea NOT NULL,
    huella_selector_fuente_sha256 text NOT NULL,
    fuente_ref text NOT NULL,
    fuente_version bigint NOT NULL,
    huella_fuente_sha256 text NOT NULL,
    reglas_ref text NOT NULL,
    reglas_version bigint NOT NULL,
    huella_reglas_contenido_sha256 text NOT NULL,
    reglas_revision bigint NOT NULL,
    huella_reglas_estado_sha256 text NOT NULL,
    convocatoria_ref text NOT NULL,
    convocatoria_version bigint NOT NULL,
    huella_convocatoria_sha256 text NOT NULL,
    entrada_ref text NOT NULL,
    entrada_version bigint NOT NULL,
    huella_entrada_sha256 text NOT NULL,
    huella_contenido_entrada_sha256 text NOT NULL,
    sujeto_ref text NOT NULL,
    sujeto_version bigint NOT NULL,
    huella_sujeto_sha256 text NOT NULL,
    tipo_efecto text NOT NULL,
    predecesor_recibo_ref text,
    huella_predecesor_recibo_sha256 text,
    estado text NOT NULL,
    fase text NOT NULL,
    intento_nominal_ref text NOT NULL,
    desenlace_nominal text NOT NULL DEFAULT 'creada',
    recibo_ref text NOT NULL,
    outbox_ref text NOT NULL,
    creada_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    PRIMARY KEY (tenant_id, resultado_ref),
    UNIQUE (tenant_id, generacion_clave_hmac, indice_efecto_hmac_sha256),
    UNIQUE (tenant_id, huella_clave_semantica_sha256),
    UNIQUE (
        tenant_id, resultado_ref, generacion_clave_hmac,
        indice_efecto_hmac_sha256, huella_clave_semantica_sha256,
        huella_resultado_sha256, tipo_efecto, estado, fase
    ),
    UNIQUE (
        tenant_id, resultado_ref,
        sujeto_ref, sujeto_version, huella_sujeto_sha256,
        convocatoria_ref, convocatoria_version, huella_convocatoria_sha256
    ),
    UNIQUE (tenant_id, resultado_ref, huella_resultado_sha256),
    UNIQUE (tenant_id, resultado_ref, huella_selector_fuente_sha256),
    UNIQUE (tenant_id, intento_nominal_ref),
    UNIQUE (tenant_id, recibo_ref),
    UNIQUE (tenant_id, outbox_ref),
    CONSTRAINT resultado_referencias_validas CHECK (
        vec_bolsa_calculo_experiencia.texto_opaco_valido(tenant_id, 128)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            resultado_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(fuente_ref, 512)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(reglas_ref, 512)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            convocatoria_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(entrada_ref, 512)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(sujeto_ref, 512)
        AND (
            predecesor_recibo_ref IS NULL
            OR vec_bolsa_calculo_experiencia.texto_opaco_valido(
                predecesor_recibo_ref, 512
            )
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            intento_nominal_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(recibo_ref, 512)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(outbox_ref, 512)
    ),
    CONSTRAINT resultado_esquemas_cerrados CHECK (
        esquema_resultado = 'vec.bolsa.resultado_experiencia.v1'
        AND esquema_clave_semantica =
            'vec.bolsa.calculo-experiencia-oficial.clave-efecto.v1'
        AND esquema_selector_fuente =
            'vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1'
    ),
    CONSTRAINT resultado_bytes_exactos CHECK (
        octet_length(resultado_canonico) BETWEEN 2 AND 67108864
        AND octet_length(clave_semantica_publica) BETWEEN 2 AND 32768
        AND octet_length(selector_fuente_canonico) BETWEEN 2 AND 32768
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_resultado_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_clave_semantica_sha256
        )
        AND encode(sha256(resultado_canonico), 'hex') =
            huella_resultado_sha256
        AND encode(sha256(clave_semantica_publica), 'hex') =
            huella_clave_semantica_sha256
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_selector_fuente_sha256
        )
        AND encode(sha256(selector_fuente_canonico), 'hex') =
            huella_selector_fuente_sha256
        AND generacion_clave_hmac > 0
        AND vec_bolsa_calculo_experiencia.indice_hmac_sha256_valido(
            indice_efecto_hmac_sha256
        )
    ),
    CONSTRAINT resultado_vinculos_exactos CHECK (
        fuente_version BETWEEN 1 AND 1000000000
        AND reglas_version BETWEEN 1 AND 1000000000
        AND reglas_revision BETWEEN 1 AND 1000000000
        AND convocatoria_version BETWEEN 1 AND 1000000000
        AND entrada_version BETWEEN 1 AND 1000000000
        AND sujeto_version BETWEEN 1 AND 1000000000
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_fuente_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_reglas_contenido_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_reglas_estado_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_convocatoria_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_entrada_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_contenido_entrada_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_sujeto_sha256
        )
    ),
    CONSTRAINT resultado_estado_fase_coherentes CHECK (
        (estado = 'completado' AND fase = 'completado')
        OR (estado = 'bloqueado' AND fase IN (
            'seleccion', 'intervalos', 'puntuacion'
        ))
    ),
    CONSTRAINT resultado_tipo_predecesor_coherente CHECK (
        (tipo_efecto = 'calculo_inicial'
         AND predecesor_recibo_ref IS NULL
         AND huella_predecesor_recibo_sha256 IS NULL)
        OR
        (tipo_efecto = 'rectificacion'
         AND predecesor_recibo_ref IS NOT NULL
         AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
             huella_predecesor_recibo_sha256
         )
         AND predecesor_recibo_ref <> recibo_ref)
    ),
    CONSTRAINT resultado_nominal_creado CHECK (desenlace_nominal = 'creada')
);

CREATE TABLE vec_bolsa_calculo_experiencia.intento (
    tenant_id text NOT NULL,
    intento_ref text NOT NULL,
    resultado_ref text NOT NULL,
    desenlace text NOT NULL,
    esquema_intencion text NOT NULL,
    intencion_canonica bytea NOT NULL,
    huella_intencion_sha256 text NOT NULL,
    generacion_clave_hmac integer NOT NULL,
    indice_efecto_hmac_sha256 text NOT NULL,
    huella_clave_semantica_sha256 text NOT NULL,
    huella_resultado_sha256 text NOT NULL,
    tipo_efecto text NOT NULL,
    estado text NOT NULL,
    fase text NOT NULL,
    consumo_lectura_ref text NOT NULL,
    consumo_escritura_ref text NOT NULL,
    recibo_ref text NOT NULL,
    auditoria_ref text NOT NULL,
    iniciado_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    confirmado_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    PRIMARY KEY (tenant_id, intento_ref),
    UNIQUE (tenant_id, auditoria_ref),
    UNIQUE (tenant_id, consumo_lectura_ref),
    UNIQUE (tenant_id, consumo_escritura_ref),
    UNIQUE (tenant_id, intento_ref, resultado_ref),
    UNIQUE (
        tenant_id, intento_ref, resultado_ref, huella_intencion_sha256
    ),
    UNIQUE (tenant_id, intento_ref, resultado_ref, desenlace),
    UNIQUE (
        tenant_id, intento_ref, resultado_ref, desenlace,
        huella_intencion_sha256
    ),
    UNIQUE (
        tenant_id, intento_ref, resultado_ref,
        consumo_lectura_ref, consumo_escritura_ref
    ),
    UNIQUE (tenant_id, intento_ref, resultado_ref, auditoria_ref),
    CONSTRAINT intento_referencias_validas CHECK (
        vec_bolsa_calculo_experiencia.texto_opaco_valido(tenant_id, 128)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(intento_ref, 512)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            resultado_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            consumo_lectura_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            consumo_escritura_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(recibo_ref, 512)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            auditoria_ref, 512
        )
    ),
    CONSTRAINT intento_desenlace_cerrado CHECK (
        desenlace IN ('creada', 'reutilizada')
    ),
    CONSTRAINT intento_intencion_exacta CHECK (
        esquema_intencion =
            'vec.bolsa.calculo-experiencia-oficial.intencion-resultado.v1'
        AND octet_length(intencion_canonica) BETWEEN 2 AND 32768
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_intencion_sha256
        )
        AND encode(sha256(intencion_canonica), 'hex') =
            huella_intencion_sha256
        AND generacion_clave_hmac > 0
        AND vec_bolsa_calculo_experiencia.indice_hmac_sha256_valido(
            indice_efecto_hmac_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_clave_semantica_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_resultado_sha256
        )
        AND tipo_efecto IN ('calculo_inicial', 'rectificacion')
        AND (
            (estado = 'completado' AND fase = 'completado')
            OR (estado = 'bloqueado' AND fase IN (
                'seleccion', 'intervalos', 'puntuacion'
            ))
        )
    ),
    CONSTRAINT intento_ventana_valida CHECK (iniciado_en <= confirmado_en),
    FOREIGN KEY (
        tenant_id, resultado_ref, generacion_clave_hmac,
        indice_efecto_hmac_sha256, huella_clave_semantica_sha256,
        huella_resultado_sha256, tipo_efecto, estado, fase
    ) REFERENCES vec_bolsa_calculo_experiencia.resultado_oficial(
        tenant_id, resultado_ref, generacion_clave_hmac,
        indice_efecto_hmac_sha256, huella_clave_semantica_sha256,
        huella_resultado_sha256, tipo_efecto, estado, fase
    ) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE vec_bolsa_calculo_experiencia.consumo_autorizaciones (
    tenant_id text NOT NULL,
    intento_ref text NOT NULL,
    resultado_ref text NOT NULL,
    perfil_proteccion text NOT NULL,
    tipo_efecto text NOT NULL,
    consumo_lectura_ref text NOT NULL,
    consumo_lectura_version bigint NOT NULL,
    huella_consumo_lectura_sha256 text NOT NULL,
    consumo_prueba_ref text NOT NULL,
    consumo_prueba_version bigint NOT NULL,
    huella_consumo_prueba_sha256 text NOT NULL,
    decision_lectura_ref text NOT NULL,
    huella_decision_lectura_sha256 text NOT NULL,
    correlacion_lectura_ref text NOT NULL,
    recurso_lectura_ref text NOT NULL,
    consumo_escritura_ref text NOT NULL,
    decision_escritura_ref text NOT NULL,
    huella_decision_escritura_sha256 text NOT NULL,
    correlacion_escritura_ref text NOT NULL,
    recurso_escritura_ref text NOT NULL,
    contexto_recurso_lectura_huella_sha256 text NOT NULL,
    contexto_recurso_escritura_huella_sha256 text NOT NULL,
    huella_selector_fuente_sha256 text NOT NULL,
    huella_intencion_sha256 text NOT NULL,
    huella_efecto_sha256 text NOT NULL,
    lectura_consumida_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    escritura_consumida_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    PRIMARY KEY (tenant_id, intento_ref),
    UNIQUE (tenant_id, consumo_lectura_ref),
    UNIQUE (tenant_id, consumo_escritura_ref),
    UNIQUE (tenant_id, decision_lectura_ref),
    UNIQUE (tenant_id, decision_escritura_ref),
    UNIQUE (tenant_id, consumo_lectura_ref, consumo_lectura_version),
    UNIQUE (tenant_id, consumo_prueba_ref, consumo_prueba_version),
    UNIQUE (
        tenant_id, intento_ref, resultado_ref,
        consumo_lectura_ref, consumo_escritura_ref
    ),
    CONSTRAINT consumo_referencias_validas CHECK (
        perfil_proteccion IN ('externo_ordinario', 'interno_alto')
        AND tipo_efecto IN ('calculo_inicial', 'rectificacion')
        AND NOT (
            tipo_efecto = 'rectificacion'
            AND perfil_proteccion <> 'interno_alto'
        )
        AND consumo_lectura_version BETWEEN 1 AND 1000000000
        AND consumo_prueba_version BETWEEN 1 AND 1000000000
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            consumo_lectura_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            consumo_prueba_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            consumo_escritura_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            decision_lectura_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            decision_escritura_ref, 512
        )
        AND correlacion_lectura_ref ~ '^correlacion_[0-9a-f]{32}$'
        AND correlacion_escritura_ref ~ '^correlacion_[0-9a-f]{32}$'
        AND correlacion_lectura_ref <> correlacion_escritura_ref
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            recurso_lectura_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            recurso_escritura_ref, 512
        )
        AND decision_lectura_ref <> decision_escritura_ref
        AND consumo_lectura_ref <> consumo_escritura_ref
        AND lectura_consumida_en <= escritura_consumida_en
        AND recurso_lectura_ref =
            'fuente:' || huella_selector_fuente_sha256
        AND recurso_escritura_ref = CASE tipo_efecto
            WHEN 'calculo_inicial' THEN
                'calculo-oficial:' || huella_intencion_sha256
            WHEN 'rectificacion' THEN
                'rectificacion-calculo-oficial:' ||
                    huella_intencion_sha256
            ELSE NULL
        END
    ),
    CONSTRAINT consumo_huellas_validas CHECK (
        vec_bolsa_calculo_experiencia.huella_sha256_valida(
            contexto_recurso_lectura_huella_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            contexto_recurso_escritura_huella_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_selector_fuente_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_efecto_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_consumo_lectura_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_consumo_prueba_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_decision_lectura_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_decision_escritura_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_intencion_sha256
        )
    ),
    FOREIGN KEY (
        tenant_id, intento_ref, resultado_ref, huella_intencion_sha256
    )
        REFERENCES vec_bolsa_calculo_experiencia.intento(
            tenant_id, intento_ref, resultado_ref, huella_intencion_sha256
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (decision_lectura_ref)
        REFERENCES vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
            decision_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (decision_escritura_ref)
        REFERENCES vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
            decision_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (tenant_id, resultado_ref, huella_efecto_sha256)
        REFERENCES vec_bolsa_calculo_experiencia.resultado_oficial(
            tenant_id, resultado_ref, huella_resultado_sha256
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (
        tenant_id, resultado_ref, huella_selector_fuente_sha256
    ) REFERENCES vec_bolsa_calculo_experiencia.resultado_oficial(
        tenant_id, resultado_ref, huella_selector_fuente_sha256
        ) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE vec_bolsa_calculo_experiencia.recibo (
    tenant_id text NOT NULL,
    recibo_ref text NOT NULL,
    resultado_ref text NOT NULL,
    intento_nominal_ref text NOT NULL,
    desenlace_nominal text NOT NULL DEFAULT 'creada',
    generacion_clave_hmac integer NOT NULL,
    indice_efecto_hmac_sha256 text NOT NULL,
    huella_clave_semantica_sha256 text NOT NULL,
    huella_intencion_sha256 text NOT NULL,
    huella_resultado_sha256 text NOT NULL,
    tipo_efecto text NOT NULL,
    sujeto_ref text NOT NULL,
    sujeto_version bigint NOT NULL,
    huella_sujeto_sha256 text NOT NULL,
    convocatoria_ref text NOT NULL,
    convocatoria_version bigint NOT NULL,
    huella_convocatoria_sha256 text NOT NULL,
    estado text NOT NULL,
    fase text NOT NULL,
    esquema_recibo text NOT NULL,
    recibo_canonico bytea NOT NULL,
    huella_recibo_sha256 text NOT NULL,
    emitido_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    PRIMARY KEY (tenant_id, recibo_ref),
    UNIQUE (tenant_id, resultado_ref),
    UNIQUE (tenant_id, intento_nominal_ref),
    UNIQUE (tenant_id, recibo_ref, resultado_ref),
    UNIQUE (
        tenant_id, recibo_ref, huella_recibo_sha256,
        sujeto_ref, sujeto_version, huella_sujeto_sha256,
        convocatoria_ref, convocatoria_version, huella_convocatoria_sha256
    ),
    CONSTRAINT recibo_exacto CHECK (
        esquema_recibo =
            'vec.bolsa.calculo-experiencia-oficial.recibo.v1'
        AND octet_length(recibo_canonico) BETWEEN 2 AND 32768
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_recibo_sha256
        )
        AND encode(sha256(recibo_canonico), 'hex') = huella_recibo_sha256
        AND desenlace_nominal = 'creada'
        AND generacion_clave_hmac > 0
        AND vec_bolsa_calculo_experiencia.indice_hmac_sha256_valido(
            indice_efecto_hmac_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_clave_semantica_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_intencion_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_resultado_sha256
        )
        AND tipo_efecto IN ('calculo_inicial', 'rectificacion')
        AND sujeto_version BETWEEN 1 AND 1000000000
        AND convocatoria_version BETWEEN 1 AND 1000000000
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(sujeto_ref, 512)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            convocatoria_ref, 512
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_sujeto_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_convocatoria_sha256
        )
        AND (
            (estado = 'completado' AND fase = 'completado')
            OR (estado = 'bloqueado' AND fase IN (
                'seleccion', 'intervalos', 'puntuacion'
            ))
        )
    ),
    FOREIGN KEY (
        tenant_id, intento_nominal_ref, resultado_ref, desenlace_nominal,
        huella_intencion_sha256
    )
        REFERENCES vec_bolsa_calculo_experiencia.intento(
            tenant_id, intento_ref, resultado_ref, desenlace,
            huella_intencion_sha256
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (
        tenant_id, resultado_ref, generacion_clave_hmac,
        indice_efecto_hmac_sha256, huella_clave_semantica_sha256,
        huella_resultado_sha256, tipo_efecto, estado, fase
    ) REFERENCES vec_bolsa_calculo_experiencia.resultado_oficial(
        tenant_id, resultado_ref, generacion_clave_hmac,
        indice_efecto_hmac_sha256, huella_clave_semantica_sha256,
        huella_resultado_sha256, tipo_efecto, estado, fase
    ) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE vec_bolsa_calculo_experiencia.auditoria (
    tenant_id text NOT NULL,
    auditoria_ref text NOT NULL,
    secuencia bigint NOT NULL,
    intento_ref text NOT NULL,
    resultado_ref text NOT NULL,
    auditoria_anterior_ref text,
    huella_anterior_sha256 text NOT NULL,
    esquema_auditoria text NOT NULL,
    registro_canonico bytea NOT NULL,
    huella_auditoria_sha256 text NOT NULL,
    registrada_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    PRIMARY KEY (tenant_id, auditoria_ref),
    UNIQUE (tenant_id, secuencia),
    UNIQUE (tenant_id, intento_ref),
    UNIQUE (tenant_id, auditoria_ref, intento_ref, resultado_ref),
    UNIQUE (tenant_id, auditoria_ref, huella_auditoria_sha256),
    CONSTRAINT auditoria_identidad_valida CHECK (
        secuencia > 0
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(tenant_id, 128)
        AND vec_bolsa_calculo_experiencia.texto_opaco_valido(
            auditoria_ref, 512
        )
        AND (
            auditoria_anterior_ref IS NULL
            OR vec_bolsa_calculo_experiencia.texto_opaco_valido(
                auditoria_anterior_ref, 512
            )
        )
    ),
    CONSTRAINT auditoria_genesis_o_enlace CHECK (
        (secuencia = 1 AND auditoria_anterior_ref IS NULL
         AND huella_anterior_sha256 = repeat('0', 64))
        OR (secuencia > 1 AND auditoria_anterior_ref IS NOT NULL
            AND huella_anterior_sha256 <> repeat('0', 64))
    ),
    CONSTRAINT auditoria_registro_exacto CHECK (
        esquema_auditoria = 'vec.bolsa.calculo-experiencia.auditoria.v1'
        AND octet_length(registro_canonico) BETWEEN 2 AND 1048576
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_anterior_sha256
        )
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_auditoria_sha256
        )
        AND encode(sha256(registro_canonico), 'hex') =
            huella_auditoria_sha256
    ),
    FOREIGN KEY (tenant_id, intento_ref, resultado_ref)
        REFERENCES vec_bolsa_calculo_experiencia.intento(
            tenant_id, intento_ref, resultado_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (
        tenant_id, auditoria_anterior_ref, huella_anterior_sha256
    ) REFERENCES vec_bolsa_calculo_experiencia.auditoria(
        tenant_id, auditoria_ref, huella_auditoria_sha256
    ) DEFERRABLE INITIALLY DEFERRED
);
CREATE UNIQUE INDEX auditoria_un_sucesor
    ON vec_bolsa_calculo_experiencia.auditoria(
        tenant_id, auditoria_anterior_ref
    ) WHERE auditoria_anterior_ref IS NOT NULL;

CREATE TABLE vec_bolsa_calculo_experiencia.outbox (
    tenant_id text NOT NULL,
    outbox_ref text NOT NULL,
    resultado_ref text NOT NULL,
    ruta text NOT NULL,
    esquema_evento text NOT NULL,
    evento_canonico bytea NOT NULL,
    huella_evento_sha256 text NOT NULL,
    creada_en vec_bolsa_calculo_experiencia.instante_utc NOT NULL,
    PRIMARY KEY (tenant_id, outbox_ref),
    UNIQUE (tenant_id, resultado_ref),
    UNIQUE (tenant_id, outbox_ref, resultado_ref),
    CONSTRAINT outbox_evento_exacto CHECK (
        ruta = 'bolsa.calculo_experiencia.resultado_oficial.v1'
        AND esquema_evento =
            'vec.bolsa.calculo-experiencia.resultado-confirmado.v1'
        AND octet_length(evento_canonico) BETWEEN 2 AND 1048576
        AND vec_bolsa_calculo_experiencia.huella_sha256_valida(
            huella_evento_sha256
        )
        AND encode(sha256(evento_canonico), 'hex') = huella_evento_sha256
    ),
    FOREIGN KEY (tenant_id, resultado_ref)
        REFERENCES vec_bolsa_calculo_experiencia.resultado_oficial(
            tenant_id, resultado_ref
        ) DEFERRABLE INITIALLY DEFERRED
);

ALTER TABLE vec_bolsa_calculo_experiencia.resultado_oficial
    ADD FOREIGN KEY (
        tenant_id, intento_nominal_ref, resultado_ref, desenlace_nominal
    ) REFERENCES vec_bolsa_calculo_experiencia.intento(
        tenant_id, intento_ref, resultado_ref, desenlace
    ) DEFERRABLE INITIALLY DEFERRED,
    ADD FOREIGN KEY (tenant_id, recibo_ref, resultado_ref)
        REFERENCES vec_bolsa_calculo_experiencia.recibo(
            tenant_id, recibo_ref, resultado_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    ADD FOREIGN KEY (
        tenant_id, predecesor_recibo_ref,
        huella_predecesor_recibo_sha256,
        sujeto_ref, sujeto_version, huella_sujeto_sha256,
        convocatoria_ref, convocatoria_version, huella_convocatoria_sha256
    ) REFERENCES vec_bolsa_calculo_experiencia.recibo(
        tenant_id, recibo_ref, huella_recibo_sha256,
        sujeto_ref, sujeto_version, huella_sujeto_sha256,
        convocatoria_ref, convocatoria_version, huella_convocatoria_sha256
    ),
    ADD FOREIGN KEY (tenant_id, outbox_ref, resultado_ref)
        REFERENCES vec_bolsa_calculo_experiencia.outbox(
            tenant_id, outbox_ref, resultado_ref
        ) DEFERRABLE INITIALLY DEFERRED;

CREATE UNIQUE INDEX resultado_un_sucesor_por_predecesor
    ON vec_bolsa_calculo_experiencia.resultado_oficial(
        tenant_id, predecesor_recibo_ref
    ) WHERE predecesor_recibo_ref IS NOT NULL;

ALTER TABLE vec_bolsa_calculo_experiencia.intento
    ADD FOREIGN KEY (
        tenant_id, intento_ref, resultado_ref,
        consumo_lectura_ref, consumo_escritura_ref
    ) REFERENCES vec_bolsa_calculo_experiencia.consumo_autorizaciones(
        tenant_id, intento_ref, resultado_ref,
        consumo_lectura_ref, consumo_escritura_ref
    ) DEFERRABLE INITIALLY DEFERRED,
    ADD FOREIGN KEY (tenant_id, recibo_ref, resultado_ref)
        REFERENCES vec_bolsa_calculo_experiencia.recibo(
            tenant_id, recibo_ref, resultado_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    ADD FOREIGN KEY (tenant_id, auditoria_ref, intento_ref, resultado_ref)
        REFERENCES vec_bolsa_calculo_experiencia.auditoria(
            tenant_id, auditoria_ref, intento_ref, resultado_ref
        ) DEFERRABLE INITIALLY DEFERRED;

CREATE FUNCTION
    vec_bolsa_calculo_experiencia.validar_predecesor_resultado()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.tipo_efecto = 'calculo_inicial' THEN
        RETURN NEW;
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_bolsa_calculo_experiencia.recibo AS recibo
          JOIN vec_bolsa_calculo_experiencia.resultado_oficial AS resultado
            ON resultado.tenant_id = recibo.tenant_id
           AND resultado.resultado_ref = recibo.resultado_ref
         WHERE recibo.tenant_id = NEW.tenant_id
           AND recibo.recibo_ref = NEW.predecesor_recibo_ref
           AND recibo.huella_recibo_sha256 =
               NEW.huella_predecesor_recibo_sha256
           AND resultado.sujeto_ref = NEW.sujeto_ref
           AND resultado.sujeto_version = NEW.sujeto_version
           AND resultado.huella_sujeto_sha256 = NEW.huella_sujeto_sha256
           AND resultado.convocatoria_ref = NEW.convocatoria_ref
           AND resultado.convocatoria_version = NEW.convocatoria_version
           AND resultado.huella_convocatoria_sha256 =
               NEW.huella_convocatoria_sha256
           AND resultado.creada_en < NEW.creada_en
           AND recibo.emitido_en < NEW.creada_en
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'predecesor de rectificacion inexistente o cruzado';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER validar_predecesor
    BEFORE INSERT ON vec_bolsa_calculo_experiencia.resultado_oficial
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_calculo_experiencia.validar_predecesor_resultado();

CREATE FUNCTION
    vec_bolsa_calculo_experiencia.validar_encadenamiento_auditoria()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    anterior vec_bolsa_calculo_experiencia.auditoria%ROWTYPE;
BEGIN
    IF NEW.secuencia = 1 THEN
        RETURN NEW;
    END IF;
    SELECT * INTO anterior
      FROM vec_bolsa_calculo_experiencia.auditoria
     WHERE tenant_id = NEW.tenant_id
       AND auditoria_ref = NEW.auditoria_anterior_ref
       AND huella_auditoria_sha256 = NEW.huella_anterior_sha256;
    IF NOT FOUND OR anterior.secuencia <> NEW.secuencia - 1
       OR anterior.registrada_en > NEW.registrada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'encadenamiento de auditoria invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER validar_encadenamiento
    BEFORE INSERT ON vec_bolsa_calculo_experiencia.auditoria
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_calculo_experiencia.validar_encadenamiento_auditoria();

CREATE FUNCTION
    vec_bolsa_calculo_experiencia.validar_autorizaciones_del_intento()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           NEW.decision_lectura_ref,
           NEW.huella_decision_lectura_sha256,
           'lectura_fuentes',
           NEW.perfil_proteccion,
           NEW.tipo_efecto,
           NEW.correlacion_lectura_ref,
           NEW.recurso_lectura_ref,
           NEW.contexto_recurso_lectura_huella_sha256
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           NEW.decision_escritura_ref,
           NEW.huella_decision_escritura_sha256,
           'escritura_resultado',
           NEW.perfil_proteccion,
           NEW.tipo_efecto,
           NEW.correlacion_escritura_ref,
           NEW.recurso_escritura_ref,
           NEW.contexto_recurso_escritura_huella_sha256
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'autorizaciones V2 del intento no revalidables';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER validar_autorizaciones
    BEFORE INSERT ON vec_bolsa_calculo_experiencia.consumo_autorizaciones
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_calculo_experiencia.validar_autorizaciones_del_intento();

DO $inmutabilidad_y_rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'resultado_oficial', 'intento', 'consumo_autorizaciones',
        'recibo', 'auditoria', 'outbox'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER impedir_mutacion BEFORE UPDATE OR DELETE ON %I.%I '
            'FOR EACH ROW EXECUTE FUNCTION %I.rechazar_mutacion_inmutable()',
            'vec_bolsa_calculo_experiencia', tabla,
            'vec_bolsa_calculo_experiencia'
        );
        EXECUTE format(
            'CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON %I.%I '
            'FOR EACH STATEMENT EXECUTE FUNCTION %I.rechazar_mutacion_inmutable()',
            'vec_bolsa_calculo_experiencia', tabla,
            'vec_bolsa_calculo_experiencia'
        );
        EXECUTE format(
            'ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY',
            'vec_bolsa_calculo_experiencia', tabla
        );
        EXECUTE format(
            'ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY',
            'vec_bolsa_calculo_experiencia', tabla
        );
        EXECUTE format(
            'CREATE POLICY aplicacion_consulta ON %I.%I FOR SELECT TO %I '
            'USING (tenant_id = (SELECT tenant_id FROM %I.configuracion_tenant))',
            'vec_bolsa_calculo_experiencia', tabla,
            'vec_bolsa_calculo_experiencia_aplicacion',
            'vec_bolsa_calculo_experiencia'
        );
        EXECUTE format(
            'CREATE POLICY aplicacion_inserta ON %I.%I FOR INSERT TO %I '
            'WITH CHECK (tenant_id = (SELECT tenant_id FROM %I.configuracion_tenant))',
            'vec_bolsa_calculo_experiencia', tabla,
            'vec_bolsa_calculo_experiencia_aplicacion',
            'vec_bolsa_calculo_experiencia'
        );
        EXECUTE format(
            'CREATE POLICY lectura_operativa ON %I.%I FOR SELECT TO %I '
            'USING (tenant_id = (SELECT tenant_id FROM %I.configuracion_tenant))',
            'vec_bolsa_calculo_experiencia', tabla,
            'vec_bolsa_calculo_experiencia_lector_operativo',
            'vec_bolsa_calculo_experiencia'
        );
    END LOOP;
END
$inmutabilidad_y_rls$;

CREATE POLICY publicador_consulta ON vec_bolsa_calculo_experiencia.outbox
    FOR SELECT TO vec_bolsa_calculo_experiencia_publicador
    USING (
        tenant_id = (
            SELECT tenant_id
              FROM vec_bolsa_calculo_experiencia.configuracion_tenant
        )
    );

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_calculo_experiencia FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_calculo_experiencia FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_calculo_experiencia TO
    vec_bolsa_calculo_experiencia_aplicacion,
    vec_bolsa_calculo_experiencia_lector_operativo,
    vec_bolsa_calculo_experiencia_publicador;
GRANT USAGE ON TYPE vec_bolsa_calculo_experiencia.instante_utc TO
    vec_bolsa_calculo_experiencia_aplicacion,
    vec_bolsa_calculo_experiencia_lector_operativo,
    vec_bolsa_calculo_experiencia_publicador;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_calculo_experiencia.texto_opaco_valido(text, integer),
    vec_bolsa_calculo_experiencia.huella_sha256_valida(text),
    vec_bolsa_calculo_experiencia.indice_hmac_sha256_valido(text),
    vec_bolsa_calculo_experiencia.instante_utc_valido(text),
    vec_bolsa_calculo_experiencia.validar_predecesor_resultado(),
    vec_bolsa_calculo_experiencia.validar_encadenamiento_auditoria(),
    vec_bolsa_calculo_experiencia.validar_autorizaciones_del_intento()
    TO vec_bolsa_calculo_experiencia_aplicacion;
GRANT SELECT (tenant_id) ON
    vec_bolsa_calculo_experiencia.configuracion_tenant TO
    vec_bolsa_calculo_experiencia_aplicacion,
    vec_bolsa_calculo_experiencia_lector_operativo,
    vec_bolsa_calculo_experiencia_publicador;
GRANT SELECT, INSERT ON
    vec_bolsa_calculo_experiencia.resultado_oficial,
    vec_bolsa_calculo_experiencia.intento,
    vec_bolsa_calculo_experiencia.consumo_autorizaciones,
    vec_bolsa_calculo_experiencia.recibo,
    vec_bolsa_calculo_experiencia.auditoria,
    vec_bolsa_calculo_experiencia.outbox
    TO vec_bolsa_calculo_experiencia_aplicacion;

GRANT SELECT (
    tenant_id, resultado_ref, esquema_resultado, huella_resultado_sha256,
    huella_clave_semantica_sha256, generacion_clave_hmac,
    indice_efecto_hmac_sha256,
    esquema_selector_fuente, huella_selector_fuente_sha256,
    fuente_ref, fuente_version, huella_fuente_sha256,
    reglas_ref, reglas_version, huella_reglas_contenido_sha256,
    reglas_revision, huella_reglas_estado_sha256,
    convocatoria_ref, convocatoria_version, huella_convocatoria_sha256,
    entrada_ref, entrada_version, huella_entrada_sha256,
    huella_contenido_entrada_sha256, sujeto_ref, sujeto_version,
    huella_sujeto_sha256, tipo_efecto, predecesor_recibo_ref,
    huella_predecesor_recibo_sha256, estado, fase,
    intento_nominal_ref, recibo_ref, outbox_ref, creada_en
) ON vec_bolsa_calculo_experiencia.resultado_oficial
    TO vec_bolsa_calculo_experiencia_lector_operativo;
GRANT SELECT (
    tenant_id, intento_ref, resultado_ref, desenlace,
    huella_intencion_sha256, generacion_clave_hmac,
    indice_efecto_hmac_sha256, huella_clave_semantica_sha256,
    huella_resultado_sha256, tipo_efecto, estado, fase,
    consumo_lectura_ref, consumo_escritura_ref, recibo_ref, auditoria_ref,
    iniciado_en, confirmado_en
) ON vec_bolsa_calculo_experiencia.intento
    TO vec_bolsa_calculo_experiencia_lector_operativo;
GRANT SELECT (
    tenant_id, intento_ref, resultado_ref, perfil_proteccion, tipo_efecto,
    consumo_lectura_ref, consumo_lectura_version,
    huella_consumo_lectura_sha256, consumo_prueba_ref,
    consumo_prueba_version, huella_consumo_prueba_sha256,
    decision_lectura_ref, huella_decision_lectura_sha256,
    correlacion_lectura_ref, recurso_lectura_ref,
    consumo_escritura_ref, decision_escritura_ref,
    huella_decision_escritura_sha256, correlacion_escritura_ref,
    recurso_escritura_ref, contexto_recurso_lectura_huella_sha256,
    contexto_recurso_escritura_huella_sha256,
    huella_selector_fuente_sha256,
    huella_intencion_sha256, huella_efecto_sha256,
    lectura_consumida_en, escritura_consumida_en
) ON vec_bolsa_calculo_experiencia.consumo_autorizaciones
    TO vec_bolsa_calculo_experiencia_lector_operativo;
GRANT SELECT (
    tenant_id, recibo_ref, resultado_ref, intento_nominal_ref,
    generacion_clave_hmac, indice_efecto_hmac_sha256,
    huella_clave_semantica_sha256, huella_intencion_sha256,
    huella_resultado_sha256, tipo_efecto,
    sujeto_ref, sujeto_version, huella_sujeto_sha256,
    convocatoria_ref, convocatoria_version, huella_convocatoria_sha256,
    estado, fase,
    esquema_recibo, huella_recibo_sha256, emitido_en
) ON vec_bolsa_calculo_experiencia.recibo
    TO vec_bolsa_calculo_experiencia_lector_operativo;
GRANT SELECT (
    tenant_id, auditoria_ref, secuencia, intento_ref, resultado_ref,
    auditoria_anterior_ref, huella_anterior_sha256, esquema_auditoria,
    huella_auditoria_sha256, registrada_en
) ON vec_bolsa_calculo_experiencia.auditoria
    TO vec_bolsa_calculo_experiencia_lector_operativo;
GRANT SELECT (
    tenant_id, outbox_ref, resultado_ref, ruta, esquema_evento,
    huella_evento_sha256, creada_en
) ON vec_bolsa_calculo_experiencia.outbox
    TO vec_bolsa_calculo_experiencia_lector_operativo;
GRANT SELECT ON vec_bolsa_calculo_experiencia.outbox
    TO vec_bolsa_calculo_experiencia_publicador;
COMMIT;
