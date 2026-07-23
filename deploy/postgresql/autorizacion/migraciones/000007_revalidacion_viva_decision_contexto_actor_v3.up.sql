-- Revalidación viva aditiva para consumidores de efectos. La función V3
-- histórica conserva su semántica de reconciliación y no se modifica.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:revalidacion-viva-v3:000007', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para revalidación viva V3';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric
)
RETURNS timestamptz
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    d jsonb;
    v jsonb;
    motivo jsonb;
    rm jsonb;
    asignacion record;
    rol record;
    catalogo record;
    manifiesto jsonb;
    ahora timestamptz(6);
    acreditada_en timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_decision_canonica IS NULL OR p_motivo_canonico IS NULL
       OR p_persona_version IS NULL OR p_perfil_version IS NULL
       OR pg_catalog.scale(p_persona_version) <> 0
       OR pg_catalog.scale(p_perfil_version) <> 0
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric THEN
        RETURN NULL;
    END IF;
    BEGIN
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
        motivo := pg_catalog.convert_from(p_motivo_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RETURN NULL;
    END;
    IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
       OR vec_autorizacion.decision_contexto_actor_v3_canonica(d)
          IS DISTINCT FROM p_decision_canonica
       OR vec_autorizacion.motivo_contexto_actor_v3_canonico(motivo)
          IS DISTINCT FROM p_motivo_canonico
       OR pg_catalog.encode(
           pg_catalog.sha256(p_motivo_canonico), 'hex'
       ) IS DISTINCT FROM d ->> 'motivo_huella_sha256'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.decision_concedida_contexto_actor_v3 e
            WHERE e.decision_ref = d ->> 'decision_ref'
              AND e.decision_canonica = p_decision_canonica
              AND e.motivo_canonico = p_motivo_canonico
              AND e.persona_version = p_persona_version
              AND e.perfil_version = p_perfil_version
       ) THEN
        RETURN NULL;
    END IF;
    v := d -> 'vinculo_autenticacion_actor';
    rm := motivo -> 'referencia';

    -- Misma barrera y mismo orden de autoridades que el registrador nominal.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
        )
    );
    acreditada_en :=
      vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        v ->> 'registro_contexto_ref', v ->> 'contexto_actor_esquema',
        v ->> 'contexto_actor_huella_sha256',
        v ->> 'manifiesto_procedencia_huella_sha256',
        v ->> 'autoridad_efectiva', v ->> 'cuenta_ref',
        (v ->> 'contexto_actor_cuenta_version')::numeric,
        v ->> 'principal_id', p_persona_version,
        v ->> 'perfil_activo_ref', p_perfil_version,
        v ->> 'contexto_actor_ref',
        (v ->> 'contexto_actor_version')::numeric,
        v ->> 'metodo_observado', v ->> 'garantia_observada',
        (d ->> 'emitida_en')::timestamptz,
        (d ->> 'valida_hasta')::timestamptz
      );
    IF acreditada_en IS NULL THEN
        RETURN NULL;
    END IF;

    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR SHARE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;
    SELECT a.asignacion_ref, a.principal_id, a.version_rol_ref,
           a.huella_sha256, a.documento
      INTO asignacion
      FROM vec_autorizacion.asignacion_perfil_actual actual
      JOIN vec_autorizacion.asignacion_perfil a
        ON a.perfil_activo_ref = actual.perfil_activo_ref
       AND a.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = d ->> 'perfil_activo_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion.asignacion_ref <> d ->> 'asignacion_ref'
       OR asignacion.principal_id <> d ->> 'principal_id'
       OR asignacion.version_rol_ref <> d ->> 'version_rol_ref'
       OR asignacion.huella_sha256 <> d ->> 'asignacion_huella_sha256'
       OR asignacion.documento ->> 'estado' <> 'activa' THEN
        RETURN NULL;
    END IF;

    SELECT r.huella_sha256, r.publicada_en, r.documento,
           c.version_rol_ref, c.revision, c.estado,
           c.huella_sha256 AS control_huella
      INTO rol
      FROM vec_autorizacion.version_rol r
      JOIN vec_autorizacion.control_vigencia_version_rol_actual actual
        ON actual.version_rol_ref = r.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol c
        ON c.version_rol_ref = actual.version_rol_ref
       AND c.revision = actual.revision
     WHERE r.version_rol_ref = asignacion.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR rol.huella_sha256 <> d ->> 'version_rol_huella_sha256'
       OR rol.version_rol_ref <>
          d ->> 'control_vigencia_version_rol_ref'
       OR rol.revision <>
          (d ->> 'control_vigencia_version_rol_revision')::numeric
       OR rol.control_huella <>
          d ->> 'control_vigencia_version_rol_huella_sha256'
       OR rol.documento ->> 'estado' <> 'publicada'
       OR rol.estado <> 'habilitada' THEN
        RETURN NULL;
    END IF;

    SELECT revision, huella_sha256
      INTO catalogo
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;
    SELECT coalesce(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_object(
                   'referencia', p.politica_ref,
                   'huella_sha256', p.huella_sha256
               ) ORDER BY p.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO manifiesto
      FROM vec_autorizacion.politica_restrictiva_actual actual
      JOIN vec_autorizacion.politica_restrictiva p
        ON p.politica_id = actual.politica_id
       AND p.politica_ref = actual.politica_ref;
    IF catalogo.revision <>
          (d ->> 'revision_catalogo_politicas')::numeric
       OR catalogo.huella_sha256 <>
          d ->> 'catalogo_politicas_huella_sha256'
       OR manifiesto IS DISTINCT FROM d -> 'politicas_evaluadas' THEN
        RETURN NULL;
    END IF;

    ahora := pg_catalog.clock_timestamp();
    IF ahora < (d ->> 'emitida_en')::timestamptz
       OR ahora >= (d ->> 'valida_hasta')::timestamptz
       OR ahora < (asignacion.documento ->> 'vigente_desde')::timestamptz
       OR ahora >= (asignacion.documento ->> 'vigente_hasta')::timestamptz
       OR rol.publicada_en > ahora
       OR vec_autorizacion.revalidar_sesion_vinculo_v2(
              v, (d ->> 'emitida_en')::timestamptz,
              (d ->> 'valida_hasta')::timestamptz, ahora
          ) IS NOT TRUE
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.motivo_v2_catalogo_publicado mc
             JOIN vec_autorizacion.motivo_v2_entrada me
               ON me.catalogo_id = mc.catalogo_id
              AND me.catalogo_version = mc.catalogo_version
            WHERE mc.catalogo_id = rm ->> 'catalogo_id'
              AND mc.catalogo_version =
                  (rm ->> 'catalogo_version')::integer
              AND mc.catalogo_huella_publicada_sha256 =
                  rm ->> 'catalogo_huella_sha256'
              AND me.entrada_clave = rm ->> 'entrada_clave'
              AND mc.publicado_en <= ahora
              AND me.vigente_desde <= ahora
              AND (me.vigente_hasta IS NULL OR ahora < me.vigente_hasta)
              AND NOT EXISTS (
                  SELECT 1
                    FROM vec_autorizacion.motivo_v2_retirada retirada
                   WHERE retirada.catalogo_id = mc.catalogo_id
                     AND retirada.catalogo_version = mc.catalogo_version
              )
       ) THEN
        RETURN NULL;
    END IF;
    RETURN ahora;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric
)
RETURNS TABLE (
    concedida boolean,
    codigo text,
    decision_huella_sha256 text,
    registrada_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    registro record;
    viva_en timestamptz(6);
BEGIN
    SELECT * INTO registro
      FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version
      );
    IF NOT FOUND OR registro.concedida IS NOT TRUE THEN
        RETURN;
    END IF;
    viva_en :=
      vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
          p_decision_canonica, p_motivo_canonico,
          p_persona_version, p_perfil_version
      );
    IF viva_en IS NULL THEN
        RETURN;
    END IF;
    RETURN QUERY SELECT
        registro.concedida, registro.codigo,
        registro.decision_huella_sha256,
        registro.registrada_en, viva_en;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
        bytea, bytea, numeric, numeric
    ),
    vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_v3(
        bytea, bytea, numeric, numeric
    )
    FROM PUBLIC, vec_autorizacion_registro,
         vec_autorizacion_fuente,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;

COMMIT;
