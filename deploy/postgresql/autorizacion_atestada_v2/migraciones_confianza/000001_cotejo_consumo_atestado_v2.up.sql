-- Extensión estrecha del catálogo para consumos atestados VEC-AD-2.
BEGIN;
SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
           'vec_confianza_atestacion_v2.obtener_confianza_actual()'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_atestada_v2_propietario'
              AND NOT rolcanlogin AND NOT rolsuper
       )
       OR to_regnamespace(
              'vec_confianza_atestacion_v2_consumo_atestado'
          ) IS NOT NULL
       OR to_regprocedure(
           'vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(text,text,timestamp with time zone,timestamp with time zone,text,numeric,text,timestamp with time zone,timestamp with time zone,text,text,timestamp with time zone)'
       ) IS NOT NULL
       OR to_regprocedure(
           'vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_en_v1(text,text,timestamp with time zone,timestamp with time zone,text,numeric,text,timestamp with time zone,timestamp with time zone,text,text,timestamp with time zone,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para cotejo atestado V2';
    END IF;
END
$prevalidacion$;

-- Estas claves candidatas permiten que las FK atestadas demuestren el valor
-- exacto que se cotejó, no solo la identidad nominal de la fila histórica.
ALTER TABLE vec_confianza_atestacion_v2.configuracion_confianza_version
    ADD CONSTRAINT configuracion_confianza_v2_datos_atestados_unicos
    UNIQUE (
        revision, huella_configuracion_sha256, publicada_en, expira_en
    );
ALTER TABLE vec_confianza_atestacion_v2.raiz_confianza_version
    ADD CONSTRAINT raiz_confianza_v2_datos_atestados_unicos
    UNIQUE (
        clave_id, version, clave_publica_spki,
        huella_clave_spki_sha256, valida_desde, valida_hasta, suite,
        audiencia_despliegue
    );
GRANT REFERENCES (
    revision, huella_configuracion_sha256, publicada_en, expira_en
) ON vec_confianza_atestacion_v2.configuracion_confianza_version
  TO vec_autorizacion_atestada_v2_propietario;
GRANT REFERENCES (
    clave_id, version, clave_publica_spki, huella_clave_spki_sha256,
    valida_desde, valida_hasta, suite, audiencia_despliegue
) ON vec_confianza_atestacion_v2.raiz_confianza_version
  TO vec_autorizacion_atestada_v2_propietario;
GRANT REFERENCES (configuracion_revision, clave_id, version) ON
    vec_confianza_atestacion_v2.configuracion_raiz
    TO vec_autorizacion_atestada_v2_propietario;

-- El checkpoint mutable transforma cualquier gobierno concurrente visible
-- después de un snapshot SERIALIZABLE en un conflicto de fila comprobable.
CREATE SCHEMA vec_confianza_atestacion_v2_consumo_atestado
    AUTHORIZATION vec_confianza_atestacion_v2_propietario;
REVOKE ALL ON SCHEMA
    vec_confianza_atestacion_v2_consumo_atestado FROM PUBLIC;

CREATE TABLE
vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    revision numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CHECK (revision BETWEEN 0 AND 18446744073709551615)
);
INSERT INTO
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno(
        control_id, revision, actualizada_en
    ) VALUES (true, 0, clock_timestamp());

ALTER TABLE
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    FOR ALL TO vec_confianza_atestacion_v2_propietario
    USING (current_user = 'vec_confianza_atestacion_v2_propietario')
    WITH CHECK (current_user = 'vec_confianza_atestacion_v2_propietario');
REVOKE ALL ON TABLE
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_autorizacion_atestada_v2_propietario;
REVOKE ALL ON TYPE
    vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_autorizacion_atestada_v2_propietario;

CREATE FUNCTION
vec_confianza_atestacion_v2_consumo_atestado.sellar_conocimiento_gobierno()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_confianza_atestacion_v2:gobierno:v1', 0
        )
    );
    IF TG_TABLE_NAME = 'acto_gobierno' THEN
        NEW.registrado_en := clock_timestamp();
        RETURN NEW;
    END IF;
    NEW.registrada_en := clock_timestamp();
    IF TG_TABLE_NAME IN (
           'revocacion_configuracion', 'revocacion_raiz'
       ) THEN
        IF NEW.revocada_en < NEW.registrada_en THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'revocacion de confianza retroactiva';
        END IF;
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION
vec_confianza_atestacion_v2_consumo_atestado.avanzar_checkpoint_gobierno()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'operacion de gobierno no admitida por checkpoint';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_confianza_atestacion_v2:gobierno:v1', 0
        )
    );
    UPDATE
        vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
       SET revision = revision + 1,
           actualizada_en = clock_timestamp()
     WHERE control_id = true;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'checkpoint de confianza ausente';
    END IF;
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_confianza_atestacion_v2_consumo_atestado.rechazar_retirada_checkpoint()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'checkpoint de confianza no eliminable';
END
$funcion$;

DO $triggers_checkpoint$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'acto_gobierno', 'configuracion_confianza_version',
        'raiz_confianza_version',
        'configuracion_raiz', 'revocacion_configuracion',
        'revocacion_raiz', 'puntero_raiz_actual',
        'puntero_configuracion_actual'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER a05_sellar_conocimiento_consumo_atestado BEFORE INSERT ON vec_confianza_atestacion_v2.%I FOR EACH ROW EXECUTE FUNCTION vec_confianza_atestacion_v2_consumo_atestado.sellar_conocimiento_gobierno()',
            tabla
        );
        EXECUTE format(
            'CREATE TRIGGER z90_avanzar_checkpoint_consumo_atestado AFTER INSERT ON vec_confianza_atestacion_v2.%I FOR EACH ROW EXECUTE FUNCTION vec_confianza_atestacion_v2_consumo_atestado.avanzar_checkpoint_gobierno()',
            tabla
        );
    END LOOP;
END
$triggers_checkpoint$;
CREATE TRIGGER checkpoint_gobierno_no_eliminar
    BEFORE DELETE ON
        vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    FOR EACH ROW EXECUTE FUNCTION
        vec_confianza_atestacion_v2_consumo_atestado.rechazar_retirada_checkpoint();
CREATE TRIGGER checkpoint_gobierno_no_truncar
    BEFORE TRUNCATE ON
        vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_confianza_atestacion_v2_consumo_atestado.rechazar_retirada_checkpoint();

REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2_consumo_atestado.sellar_conocimiento_gobierno()
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_autorizacion_atestada_v2_propietario;
REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2_consumo_atestado.avanzar_checkpoint_gobierno()
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_autorizacion_atestada_v2_propietario;
REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2_consumo_atestado.rechazar_retirada_checkpoint()
    FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad,
         vec_autorizacion_atestada_v2_propietario;

-- p_consumir_en NULL significa: tomar el reloj después de todos los locks.
-- La variante exacta solo la usa la puerta después de un primer cotejo que
-- conserva esos locks hasta COMMIT; así comparte un único instante final con
-- la revalidación de clave y con las inserciones atestadas.
CREATE FUNCTION
vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_en_v1(
    p_revision text,
    p_huella_configuracion_sha256 text,
    p_configuracion_publicada_en timestamptz(6),
    p_configuracion_expira_en timestamptz(6),
    p_raiz_clave_id text,
    p_raiz_version numeric(20, 0),
    p_huella_raiz_spki_sha256 text,
    p_raiz_valida_desde timestamptz(6),
    p_raiz_valida_hasta timestamptz(6),
    p_suite text,
    p_audiencia_despliegue text,
    p_verificada_en timestamptz(6),
    p_consumir_en timestamptz(6)
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    checkpoint_revision numeric(20, 0);
    puntero_config record;
    configuracion record;
    miembro record;
    puntero_version numeric(20, 0);
    raiz_miembro record;
    raiz record;
    instante timestamptz(6);
    numero_miembros bigint;
    raices_activas bigint := 0;
BEGIN
    IF current_user <> 'vec_confianza_atestacion_v2_propietario'
       OR vec_confianza_atestacion_v2.texto_tecnico_valido(
              p_revision, 128
          ) IS NOT TRUE
       OR vec_confianza_atestacion_v2.huella_sha256_valida(
              p_huella_configuracion_sha256
          ) IS NOT TRUE
       OR vec_confianza_atestacion_v2.texto_tecnico_valido(
              p_raiz_clave_id, 512
          ) IS NOT TRUE
       OR p_raiz_version NOT BETWEEN 1 AND 18446744073709551615
       OR vec_confianza_atestacion_v2.huella_sha256_valida(
              p_huella_raiz_spki_sha256
          ) IS NOT TRUE
       OR p_suite <> 'VEC-AD-2-COSE-EDDSA-1'
       OR vec_confianza_atestacion_v2.audiencia_despliegue_valida(
              p_audiencia_despliegue
          ) IS NOT TRUE
       OR vec_confianza_atestacion_v2.instante_go_valido(
              p_configuracion_publicada_en
          ) IS NOT TRUE
       OR vec_confianza_atestacion_v2.instante_go_valido(
              p_configuracion_expira_en
          ) IS NOT TRUE
       OR vec_confianza_atestacion_v2.instante_go_valido(
              p_raiz_valida_desde
          ) IS NOT TRUE
       OR vec_confianza_atestacion_v2.instante_go_valido(
              p_raiz_valida_hasta
          ) IS NOT TRUE
       OR vec_confianza_atestacion_v2.instante_go_valido(
              p_verificada_en
          ) IS NOT TRUE
       OR (p_consumir_en IS NOT NULL AND
           vec_confianza_atestacion_v2.instante_go_valido(
               p_consumir_en
           ) IS NOT TRUE) THEN
        RETURN false;
    END IF;

    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_confianza_atestacion_v2:gobierno:v1', 0
        )
    );
    -- FOR SHARE es deliberado: un snapshot SERIALIZABLE anterior a una
    -- actualización ya comprometida del checkpoint aborta con 40001.
    SELECT revision INTO STRICT checkpoint_revision
      FROM vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
     WHERE control_id = true
     FOR SHARE;

    -- Primero se toman todos los locks potencialmente bloqueantes sin usar
    -- aún el reloj de consumo. El advisory lock cubre también filas ausentes.
    SELECT puntero.orden, puntero.revision, puntero.establecida_en
      INTO puntero_config
      FROM vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
     ORDER BY puntero.orden DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    SELECT version.revision, version.huella_configuracion_sha256,
           version.publicada_en, version.expira_en, version.numero_raices
      INTO configuracion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version AS version
     WHERE version.revision = p_revision FOR SHARE;
    IF NOT FOUND THEN
        RETURN false;
    END IF;
    PERFORM 1
      FROM vec_confianza_atestacion_v2.revocacion_configuracion
     WHERE revision = p_revision FOR SHARE;
    FOR miembro IN
        SELECT enlace.clave_id, enlace.version
          FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
         WHERE enlace.configuracion_revision = p_revision
         ORDER BY enlace.clave_id COLLATE "C" FOR SHARE
    LOOP
        PERFORM 1
          FROM vec_confianza_atestacion_v2.puntero_raiz_actual AS puntero
         WHERE puntero.clave_id = miembro.clave_id
         ORDER BY puntero.orden DESC LIMIT 1 FOR SHARE;
        PERFORM 1
          FROM vec_confianza_atestacion_v2.raiz_confianza_version
         WHERE clave_id = miembro.clave_id AND version = miembro.version
         FOR SHARE;
        PERFORM 1
          FROM vec_confianza_atestacion_v2.revocacion_raiz
         WHERE clave_id = miembro.clave_id AND version = miembro.version
         FOR SHARE;
    END LOOP;

    instante := COALESCE(p_consumir_en, clock_timestamp());
    IF p_consumir_en IS NOT NULL AND p_consumir_en > clock_timestamp() THEN
        RETURN false;
    END IF;
    SELECT puntero.orden, puntero.revision, puntero.establecida_en
      INTO puntero_config
      FROM vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
     WHERE puntero.establecida_en <= instante
       AND puntero.registrada_en <= instante
     ORDER BY puntero.orden DESC LIMIT 1;
    IF NOT FOUND OR puntero_config.revision IS DISTINCT FROM p_revision
       OR configuracion.huella_configuracion_sha256 IS DISTINCT FROM
          p_huella_configuracion_sha256
       OR configuracion.publicada_en IS DISTINCT FROM
          p_configuracion_publicada_en
       OR configuracion.expira_en IS DISTINCT FROM p_configuracion_expira_en
       OR instante < configuracion.publicada_en
       OR instante >= configuracion.expira_en
       OR p_verificada_en < configuracion.publicada_en
       OR p_verificada_en >= configuracion.expira_en
       OR EXISTS (
           SELECT 1
             FROM vec_confianza_atestacion_v2.revocacion_configuracion
            WHERE revision = configuracion.revision
              AND revocada_en <= instante
              AND registrada_en <= instante
       ) THEN
        RETURN false;
    END IF;
    SELECT count(*) INTO numero_miembros
      FROM vec_confianza_atestacion_v2.configuracion_raiz
     WHERE configuracion_revision = configuracion.revision;
    IF numero_miembros <> configuracion.numero_raices
       OR numero_miembros NOT BETWEEN 1 AND 64
       OR vec_confianza_atestacion_v2.calcular_huella_configuracion(
              configuracion.revision
          ) IS DISTINCT FROM configuracion.huella_configuracion_sha256 THEN
        RETURN false;
    END IF;

    FOR miembro IN
        SELECT enlace.clave_id, enlace.version
          FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
         WHERE enlace.configuracion_revision = configuracion.revision
         ORDER BY enlace.clave_id COLLATE "C"
    LOOP
        SELECT puntero.version INTO puntero_version
          FROM vec_confianza_atestacion_v2.puntero_raiz_actual AS puntero
         WHERE puntero.clave_id = miembro.clave_id
           AND puntero.establecida_en <= instante
           AND puntero.registrada_en <= instante
         ORDER BY puntero.orden DESC LIMIT 1;
        IF NOT FOUND OR puntero_version <> miembro.version THEN
            RETURN false;
        END IF;
        SELECT version.clave_id, version.version, version.suite,
               version.audiencia_despliegue,
               version.huella_clave_spki_sha256,
               version.valida_desde, version.valida_hasta
          INTO raiz_miembro
          FROM vec_confianza_atestacion_v2.raiz_confianza_version AS version
         WHERE version.clave_id = miembro.clave_id
           AND version.version = miembro.version;
        IF NOT FOUND THEN
            RETURN false;
        END IF;
        IF instante >= raiz_miembro.valida_desde
           AND instante < raiz_miembro.valida_hasta
           AND NOT EXISTS (
               SELECT 1 FROM vec_confianza_atestacion_v2.revocacion_raiz
                WHERE clave_id = raiz_miembro.clave_id
                  AND version = raiz_miembro.version
                  AND revocada_en <= instante
                  AND registrada_en <= instante
           ) THEN
            raices_activas := raices_activas + 1;
        END IF;
        IF raiz_miembro.clave_id = p_raiz_clave_id
           AND raiz_miembro.version = p_raiz_version THEN
            raiz := raiz_miembro;
        END IF;
    END LOOP;

    IF raiz IS NULL OR raiz.huella_clave_spki_sha256 IS DISTINCT FROM
           p_huella_raiz_spki_sha256
       OR raiz.suite IS DISTINCT FROM p_suite
       OR raiz.audiencia_despliegue IS DISTINCT FROM p_audiencia_despliegue
       OR raiz.valida_desde IS DISTINCT FROM p_raiz_valida_desde
       OR raiz.valida_hasta IS DISTINCT FROM p_raiz_valida_hasta
       OR instante < raiz.valida_desde OR instante >= raiz.valida_hasta
       OR p_verificada_en < raiz.valida_desde
       OR p_verificada_en >= raiz.valida_hasta
       OR EXISTS (
           SELECT 1 FROM vec_confianza_atestacion_v2.revocacion_raiz
            WHERE clave_id = raiz.clave_id AND version = raiz.version
              AND revocada_en <= instante
              AND registrada_en <= instante
       ) THEN
        RETURN false;
    END IF;
    RETURN raices_activas >= 1;
EXCEPTION
    WHEN data_exception OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
    p_revision text,
    p_huella_configuracion_sha256 text,
    p_configuracion_publicada_en timestamptz(6),
    p_configuracion_expira_en timestamptz(6),
    p_raiz_clave_id text,
    p_raiz_version numeric(20, 0),
    p_huella_raiz_spki_sha256 text,
    p_raiz_valida_desde timestamptz(6),
    p_raiz_valida_hasta timestamptz(6),
    p_suite text,
    p_audiencia_despliegue text,
    p_verificada_en timestamptz(6)
)
RETURNS boolean
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT vec_confianza_atestacion_v2.
        cotejar_confianza_consumo_atestado_en_v1(
            p_revision, p_huella_configuracion_sha256,
            p_configuracion_publicada_en, p_configuracion_expira_en,
            p_raiz_clave_id, p_raiz_version, p_huella_raiz_spki_sha256,
            p_raiz_valida_desde, p_raiz_valida_hasta, p_suite,
            p_audiencia_despliegue, p_verificada_en, NULL
        )
$funcion$;

REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    ) FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad;
REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_en_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz, timestamptz
    ) FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad;
GRANT EXECUTE ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    ) TO vec_autorizacion_atestada_v2_propietario;
GRANT EXECUTE ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_en_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz, timestamptz
    ) TO vec_autorizacion_atestada_v2_propietario;
COMMIT;
