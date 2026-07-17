-- Extensión estrecha del catálogo: no expone filas ni sustituye su lector.
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
       OR to_regprocedure(
           'vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(text,text,timestamp with time zone,timestamp with time zone,text,numeric,text,timestamp with time zone,timestamp with time zone,text,text,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para cotejo atestado V2';
    END IF;
END
$prevalidacion$;

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
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    puntero_config record;
    configuracion record;
    miembro record;
    puntero_version numeric(20, 0);
    raiz record;
    instante timestamptz(6);
    numero_miembros bigint;
    raices_activas bigint;
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
          ) IS NOT TRUE THEN
        RETURN false;
    END IF;

    -- Es exactamente el candado de gobierno usado por publicaciones,
    -- revocaciones y obtener_confianza_actual(). Se conserva hasta COMMIT.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_confianza_atestacion_v2:gobierno:v1', 0
        )
    );
    instante := clock_timestamp();

    SELECT puntero.orden, puntero.revision INTO puntero_config
      FROM vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
     WHERE puntero.establecida_en <= instante
     ORDER BY puntero.orden DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND OR puntero_config.revision IS DISTINCT FROM p_revision THEN
        RETURN false;
    END IF;
    SELECT version.revision, version.huella_configuracion_sha256,
           version.publicada_en, version.expira_en, version.numero_raices
      INTO STRICT configuracion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version AS version
     WHERE version.revision = p_revision FOR SHARE;
    IF configuracion.huella_configuracion_sha256 IS DISTINCT FROM
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

    -- La configuración completa debe seguir siendo reconstruible; no basta
    -- con que la raíz elegida exista de forma aislada.
    FOR miembro IN
        SELECT enlace.clave_id, enlace.version
          FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
         WHERE enlace.configuracion_revision = configuracion.revision
         ORDER BY enlace.clave_id COLLATE "C" FOR SHARE
    LOOP
        SELECT puntero.version INTO puntero_version
          FROM vec_confianza_atestacion_v2.puntero_raiz_actual AS puntero
         WHERE puntero.clave_id = miembro.clave_id
           AND puntero.establecida_en <= instante
         ORDER BY puntero.orden DESC LIMIT 1 FOR SHARE;
        IF NOT FOUND OR puntero_version <> miembro.version THEN
            RETURN false;
        END IF;
        PERFORM 1
          FROM vec_confianza_atestacion_v2.raiz_confianza_version
         WHERE clave_id = miembro.clave_id AND version = miembro.version
         FOR SHARE;
        IF NOT FOUND THEN
            RETURN false;
        END IF;
    END LOOP;

    SELECT version.clave_id, version.version, version.suite,
           version.audiencia_despliegue,
           version.huella_clave_spki_sha256,
           version.valida_desde, version.valida_hasta
      INTO raiz
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
        ON version.clave_id = enlace.clave_id
       AND version.version = enlace.version
     WHERE enlace.configuracion_revision = configuracion.revision
       AND enlace.clave_id = p_raiz_clave_id
       AND enlace.version = p_raiz_version
     FOR SHARE OF enlace, version;
    IF NOT FOUND OR raiz.huella_clave_spki_sha256 IS DISTINCT FROM
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
       ) THEN
        RETURN false;
    END IF;
    SELECT count(*) INTO raices_activas
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
        ON version.clave_id = enlace.clave_id
       AND version.version = enlace.version
      LEFT JOIN vec_confianza_atestacion_v2.revocacion_raiz AS revocacion
        ON revocacion.clave_id = version.clave_id
       AND revocacion.version = version.version
       AND revocacion.revocada_en <= instante
     WHERE enlace.configuracion_revision = configuracion.revision
       AND revocacion.clave_id IS NULL
       AND instante >= version.valida_desde
       AND instante < version.valida_hasta;
    RETURN raices_activas >= 1;
EXCEPTION
    WHEN data_exception OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    ) FROM PUBLIC, vec_confianza_atestacion_v2_lector_autoridad;
GRANT EXECUTE ON FUNCTION
    vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(
        text, text, timestamptz, timestamptz, text, numeric, text,
        timestamptz, timestamptz, text, text, timestamptz
    ) TO vec_autorizacion_atestada_v2_propietario;
COMMIT;
