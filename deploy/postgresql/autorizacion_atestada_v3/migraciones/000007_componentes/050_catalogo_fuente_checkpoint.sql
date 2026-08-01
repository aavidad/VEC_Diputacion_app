-- Catalogo gobernado de fuentes corporativas para ContextoActor V1.
-- Este componente no publica CRUD ni implementa el consumo de capacidades.

ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;

ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
    ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (
        audiencia_consumo IN (
            'vec_contratacion_temporal.confirmar_alta_atestada.v1',
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
            'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
            'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
            'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
            'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
        )
    );

CREATE TABLE
vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 (
    fuente_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    fuente_version pg_catalog.numeric(20, 0) NOT NULL,
    audiencia_consumo pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    accion pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    tipo_efecto pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    clave_id pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    clave_version pg_catalog.numeric(20, 0) NOT NULL,
    revision_gobierno pg_catalog.numeric(20, 0) NOT NULL,
    huella_gobierno_sha256 pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    emisor_id pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    configuracion_revision pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    configuracion_secuencia pg_catalog.numeric(20, 0) NOT NULL,
    huella_configuracion_sha256 pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    raiz_clave_id pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    raiz_version pg_catalog.numeric(20, 0) NOT NULL,
    huella_raiz_spki_sha256 pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    audiencia_despliegue pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    suite pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    valida_desde pg_catalog.timestamptz(6) NOT NULL,
    valida_hasta pg_catalog.timestamptz(6) NOT NULL,
    acto_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    registrada_en pg_catalog.timestamptz(6) NOT NULL
        DEFAULT pg_catalog.clock_timestamp(),
    CONSTRAINT f0_fuente_pk PRIMARY KEY (
        fuente_ref, fuente_version, audiencia_consumo
    ),
    CONSTRAINT f0_fuente_clave_fk FOREIGN KEY (clave_id, clave_version)
        REFERENCES vec_autorizacion_atestada_v3.clave_capacidad_version (
            clave_id, version
        ) MATCH FULL,
    CONSTRAINT f0_fuente_config_fk FOREIGN KEY (configuracion_revision)
        REFERENCES
        vec_autorizacion_atestada_v3.configuracion_confianza_version (
            revision
        ) MATCH FULL,
    CONSTRAINT f0_fuente_raiz_fk FOREIGN KEY (raiz_clave_id, raiz_version)
        REFERENCES vec_autorizacion_atestada_v3.raiz_confianza_version (
            clave_id, version
        ) MATCH FULL,
    CONSTRAINT f0_fuente_config_raiz_fk FOREIGN KEY (
        configuracion_revision, raiz_clave_id, raiz_version
    ) REFERENCES vec_autorizacion_atestada_v3.configuracion_raiz (
        configuracion_revision, raiz_clave_id, raiz_version
    ) MATCH FULL,
    CONSTRAINT f0_fuente_ref_ck CHECK (
        vec_autorizacion_atestada_v3
            .referencia_opaca_fuente_corporativa_valida(fuente_ref)
    ),
    CONSTRAINT f0_fuente_version_ck CHECK (
        fuente_version BETWEEN 1 AND 9007199254740991::pg_catalog.numeric
    ),
    CONSTRAINT f0_fuente_cruce_ck CHECK (
        (audiencia_consumo, accion, tipo_efecto) IN (
            (
                'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
                'contexto_actor.organizacion_corporativa.publicar',
                'organizacion_corporativa.alta'
            ),
            (
                'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
                'contexto_actor.organizacion_corporativa.revocar',
                'organizacion_corporativa.revocacion'
            ),
            (
                'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
                'contexto_actor.vinculo_corporativo.publicar',
                'vinculo_corporativo.alta'
            ),
            (
                'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1',
                'contexto_actor.vinculo_corporativo.revocar',
                'vinculo_corporativo.revocacion'
            )
        )
    ),
    CONSTRAINT f0_fuente_clave_id_ck CHECK (
        vec_autorizacion_atestada_v3.texto_tecnico_valido(clave_id, 512)
    ),
    CONSTRAINT f0_fuente_clave_version_ck CHECK (
        clave_version BETWEEN 1 AND 9007199254740991::pg_catalog.numeric
    ),
    CONSTRAINT f0_fuente_revision_ck CHECK (
        revision_gobierno BETWEEN 1 AND
            9007199254740991::pg_catalog.numeric
    ),
    CONSTRAINT f0_fuente_huella_gobierno_ck CHECK (
        vec_autorizacion_atestada_v3
            .huella_sha256_valida(huella_gobierno_sha256)
    ),
    CONSTRAINT f0_fuente_emisor_ck CHECK (
        vec_autorizacion_atestada_v3.texto_tecnico_valido(emisor_id, 512)
    ),
    CONSTRAINT f0_fuente_config_revision_ck CHECK (
        vec_autorizacion_atestada_v3.texto_tecnico_valido(
            configuracion_revision, 512
        )
    ),
    CONSTRAINT f0_fuente_config_secuencia_ck CHECK (
        configuracion_secuencia BETWEEN 1 AND
            9007199254740991::pg_catalog.numeric
    ),
    CONSTRAINT f0_fuente_huella_config_ck CHECK (
        vec_autorizacion_atestada_v3
            .huella_sha256_valida(huella_configuracion_sha256)
    ),
    CONSTRAINT f0_fuente_raiz_id_ck CHECK (
        vec_autorizacion_atestada_v3
            .texto_tecnico_valido(raiz_clave_id, 512)
    ),
    CONSTRAINT f0_fuente_raiz_version_ck CHECK (
        raiz_version BETWEEN 1 AND 9007199254740991::pg_catalog.numeric
    ),
    CONSTRAINT f0_fuente_huella_raiz_ck CHECK (
        vec_autorizacion_atestada_v3
            .huella_sha256_valida(huella_raiz_spki_sha256)
    ),
    CONSTRAINT f0_fuente_despliegue_ck CHECK (
        vec_autorizacion_atestada_v3
            .texto_tecnico_valido(audiencia_despliegue, 512)
    ),
    CONSTRAINT f0_fuente_suite_ck CHECK (
        suite = 'VEC-AD-3-COSE-EDDSA-1'
    ),
    CONSTRAINT f0_fuente_desde_ck CHECK (
        vec_autorizacion_atestada_v3
            .instante_fuente_finito_valido(valida_desde)
    ),
    CONSTRAINT f0_fuente_hasta_ck CHECK (
        vec_autorizacion_atestada_v3
            .instante_fuente_finito_valido(valida_hasta)
    ),
    CONSTRAINT f0_fuente_ventana_ck CHECK (valida_hasta > valida_desde),
    CONSTRAINT f0_fuente_acto_ck CHECK (
        vec_autorizacion_atestada_v3.texto_tecnico_valido(acto_ref, 512)
    ),
    CONSTRAINT f0_fuente_registrada_ck CHECK (
        vec_autorizacion_atestada_v3
            .instante_fuente_finito_valido(registrada_en)
    )
);

CREATE INDEX f0_fuente_clave_idx ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
    USING btree (clave_id, clave_version);
CREATE INDEX f0_fuente_config_raiz_idx ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
    USING btree (configuracion_revision, raiz_clave_id, raiz_version);
CREATE INDEX f0_fuente_raiz_idx ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
    USING btree (raiz_clave_id, raiz_version);

CREATE TABLE
vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1 (
    fuente_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    fuente_version pg_catalog.numeric(20, 0) NOT NULL,
    audiencia_consumo pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    revocada_en pg_catalog.timestamptz(6) NOT NULL,
    motivo_catalogado_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    acto_ref pg_catalog.text COLLATE pg_catalog."C" NOT NULL,
    registrada_en pg_catalog.timestamptz(6) NOT NULL
        DEFAULT pg_catalog.clock_timestamp(),
    CONSTRAINT f0_revocacion_fuente_pk PRIMARY KEY (
        fuente_ref, fuente_version, audiencia_consumo
    ),
    CONSTRAINT f0_revocacion_fuente_fk FOREIGN KEY (
        fuente_ref, fuente_version, audiencia_consumo
    ) REFERENCES
        vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1 (
            fuente_ref, fuente_version, audiencia_consumo
        ) MATCH FULL,
    CONSTRAINT f0_revocacion_fuente_ref_ck CHECK (
        vec_autorizacion_atestada_v3
            .referencia_opaca_fuente_corporativa_valida(fuente_ref)
    ),
    CONSTRAINT f0_revocacion_fuente_version_ck CHECK (
        fuente_version BETWEEN 1 AND 9007199254740991::pg_catalog.numeric
    ),
    CONSTRAINT f0_revocacion_fuente_audiencia_ck CHECK (
        audiencia_consumo IN (
            'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
            'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
            'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
            'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1'
        )
    ),
    CONSTRAINT f0_revocacion_fuente_instante_ck CHECK (
        vec_autorizacion_atestada_v3
            .instante_fuente_finito_valido(revocada_en)
    ),
    CONSTRAINT f0_revocacion_fuente_motivo_ck CHECK (
        vec_autorizacion_atestada_v3
            .texto_tecnico_valido(motivo_catalogado_ref, 512)
    ),
    CONSTRAINT f0_revocacion_fuente_acto_ck CHECK (
        vec_autorizacion_atestada_v3.texto_tecnico_valido(acto_ref, 512)
    ),
    CONSTRAINT f0_revocacion_fuente_registrada_ck CHECK (
        vec_autorizacion_atestada_v3
            .instante_fuente_finito_valido(registrada_en)
    )
);

CREATE FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()
RETURNS pg_catalog.trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_configuracion_secuencia pg_catalog.numeric(20, 0);
    v_raiz_version pg_catalog.numeric(20, 0);
BEGIN
    IF TG_OP <> 'INSERT' OR TG_WHEN <> 'BEFORE' OR TG_LEVEL <> 'ROW'
       OR TG_TABLE_SCHEMA <> 'vec_autorizacion_atestada_v3'
       OR TG_TABLE_NAME NOT IN (
           'fuente_corporativa_contexto_actor_v1',
           'revocacion_fuente_corporativa_contexto_actor_v1'
       ) OR TG_NARGS <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'avance de checkpoint de fuente rechazado';
    END IF;

    -- El checkpoint es siempre la primera fila causal. El consumidor toma
    -- esta misma fila antes de catalogo, clave, configuracion y raiz.
    PERFORM 1
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno AS cp
     WHERE cp.control_id
       AND cp.revision < 9007199254740991::pg_catalog.numeric
       FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'checkpoint de fuente no disponible';
    END IF;

    IF TG_TABLE_NAME = 'fuente_corporativa_contexto_actor_v1' THEN
        v_configuracion_secuencia := NEW.configuracion_secuencia;
        v_raiz_version := NEW.raiz_version;
    ELSE
        SELECT f.configuracion_secuencia, f.raiz_version
          INTO v_configuracion_secuencia, v_raiz_version
          FROM vec_autorizacion_atestada_v3
                   .fuente_corporativa_contexto_actor_v1 AS f
         WHERE f.fuente_ref = NEW.fuente_ref
           AND f.fuente_version = NEW.fuente_version
           AND f.audiencia_consumo = NEW.audiencia_consumo;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '23503',
                MESSAGE = 'fuente corporativa ausente';
        END IF;
    END IF;

    UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
       SET revision = revision + 1,
           configuracion_secuencia_minima = GREATEST(
               configuracion_secuencia_minima, v_configuracion_secuencia
           ),
           raiz_version_minima = GREATEST(
               raiz_version_minima, v_raiz_version
           ),
           actualizada_en = pg_catalog.clock_timestamp()
     WHERE control_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'checkpoint de fuente no disponible';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER f0_checkpoint_antes
BEFORE INSERT ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_fuente_corporativa_contexto_actor_v1();
CREATE TRIGGER f0_checkpoint_antes
BEFORE INSERT ON
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_fuente_corporativa_contexto_actor_v1();

CREATE TRIGGER f0_historia_inmutable
BEFORE UPDATE OR DELETE ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER f0_historia_no_truncable
BEFORE TRUNCATE ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_truncado();
CREATE TRIGGER f0_historia_inmutable
BEFORE UPDATE OR DELETE ON
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER f0_historia_no_truncable
BEFORE TRUNCATE ON
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.rechazar_truncado();

ALTER TABLE
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1
    AS PERMISSIVE FOR ALL TO vec_autorizacion_atestada_v3_propietario
    USING (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario')
    WITH CHECK (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario');

ALTER TABLE
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
    AS PERMISSIVE FOR ALL TO vec_autorizacion_atestada_v3_propietario
    USING (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario')
    WITH CHECK (CURRENT_USER = 'vec_autorizacion_atestada_v3_propietario');

REVOKE ALL ON TABLE
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;
REVOKE ALL ON TYPE
    vec_autorizacion_atestada_v3.fuente_corporativa_contexto_actor_v1,
    vec_autorizacion_atestada_v3.revocacion_fuente_corporativa_contexto_actor_v1
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3
    .avanzar_checkpoint_fuente_corporativa_contexto_actor_v1()
FROM PUBLIC, vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario,
    vec_contexto_actor_v1_propietario;
