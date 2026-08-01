-- Historias minimizadas de atestacion y consumo de fuente corporativa V1.
-- C1/C2 acreditan el material y gobiernan las inserciones; este componente
-- solo fija persistencia, unicidades, inmutabilidad y denegacion por defecto.

CREATE TABLE
vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1 (
    capacidad_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    fuente_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    fuente_version pg_catalog.numeric(20, 0) NOT NULL,
    evento_fuente_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    CONSTRAINT f0_atestacion_fuente_pk PRIMARY KEY (capacidad_ref),
    CONSTRAINT f0_atestacion_fuente_evento_uq UNIQUE (
        fuente_ref, evento_fuente_ref
    ),
    CONSTRAINT f0_atestacion_capacidad_ref_ck CHECK (
        pg_catalog.octet_length(capacidad_ref) = 68
        AND (capacidad_ref COLLATE pg_catalog."C") ~
            '^cfc_[0-9a-f]{64}$'
        AND pg_catalog.substr(capacidad_ref, 5) <>
            pg_catalog.repeat('0', 64)
    ),
    CONSTRAINT f0_atestacion_fuente_ref_ck CHECK (
        vec_autorizacion_atestada_v3
            .referencia_opaca_fuente_corporativa_valida(fuente_ref)
    ),
    CONSTRAINT f0_atestacion_fuente_version_ck CHECK (
        fuente_version BETWEEN 1 AND
            9007199254740991::pg_catalog.numeric
    ),
    CONSTRAINT f0_atestacion_evento_ref_ck CHECK (
        vec_autorizacion_atestada_v3
            .referencia_opaca_fuente_corporativa_valida(evento_fuente_ref)
    )
);

CREATE TABLE
vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1 (
    capacidad_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    nonce pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    operacion_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    consumo_canonico pg_catalog.bytea NOT NULL,
    consumo_huella_sha256 pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    consumida_en pg_catalog.timestamptz(6) NOT NULL,
    CONSTRAINT f0_consumo_fuente_pk PRIMARY KEY (capacidad_ref),
    CONSTRAINT f0_consumo_fuente_atestacion_fk FOREIGN KEY (capacidad_ref)
        REFERENCES vec_autorizacion_atestada_v3
            .atestacion_fuente_corporativa_contexto_actor_v1 (
                capacidad_ref
            ) MATCH FULL,
    CONSTRAINT f0_consumo_fuente_nonce_uq UNIQUE (nonce),
    CONSTRAINT f0_consumo_fuente_operacion_uq UNIQUE (operacion_ref),
    CONSTRAINT f0_consumo_nonce_ck CHECK (
        vec_autorizacion_atestada_v3.huella_sha256_valida(nonce)
    ),
    CONSTRAINT f0_consumo_operacion_ref_ck CHECK (
        vec_autorizacion_atestada_v3
            .operacion_ref_fuente_corporativa_valida(operacion_ref)
    ),
    CONSTRAINT f0_consumo_canon_ck CHECK (
        pg_catalog.octet_length(consumo_canonico) BETWEEN 512 AND 32768
    ),
    CONSTRAINT f0_consumo_huella_ck CHECK (
        vec_autorizacion_atestada_v3
            .huella_sha256_valida(consumo_huella_sha256)
    ),
    CONSTRAINT f0_consumo_instante_ck CHECK (
        vec_autorizacion_atestada_v3
            .instante_fuente_finito_valido(consumida_en)
    ),
    CONSTRAINT f0_consumo_canon_huella_ck CHECK (
        consumo_huella_sha256 = pg_catalog.encode(
            pg_catalog.sha256(consumo_canonico), 'hex'
        )
    )
);

CREATE TRIGGER f0_historia_inmutable
BEFORE UPDATE OR DELETE ON
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER f0_historia_no_truncable
BEFORE TRUNCATE ON
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_truncado();
CREATE TRIGGER f0_historia_inmutable
BEFORE UPDATE OR DELETE ON
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER f0_historia_no_truncable
BEFORE TRUNCATE ON
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_truncado();

ALTER TABLE
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1
    AS PERMISSIVE FOR ALL TO vec_autorizacion_atestada_v3_propietario
    USING (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario')
    WITH CHECK (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario');

ALTER TABLE
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
    AS PERMISSIVE FOR ALL TO vec_autorizacion_atestada_v3_propietario
    USING (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario')
    WITH CHECK (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario');

REVOKE ALL ON TABLE
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;
REVOKE ALL ON TYPE
    vec_autorizacion_atestada_v3.atestacion_fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.consumo_fuente_corporativa_contexto_actor_v1
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;
