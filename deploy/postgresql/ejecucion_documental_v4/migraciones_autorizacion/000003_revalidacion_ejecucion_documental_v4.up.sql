-- Revalidacion cerrada del perfil documental V4. La decision canonica se
-- recibe como bytea porque la huella reforzada se define sobre esos bytes, no
-- sobre la representacion variable de jsonb. Tras recalcular SHA-256 se
-- interpreta el documento y se cotejan exactamente sus 30 claves, incluido el
-- vinculo de autenticacion de 25 claves, con la decision autoritativa bloqueada.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_autorizacion.huella_lista_documental_v4(
    p_esquema text,
    p_valores jsonb
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    preimagen bytea := ''::bytea;
    valor text;
BEGIN
    IF vec_autorizacion.texto_positivo_valido(p_esquema, 256) IS NOT TRUE
       OR jsonb_typeof(p_valores) <> 'array'
       OR EXISTS (
           SELECT 1
             FROM jsonb_array_elements(p_valores) AS elemento
            WHERE jsonb_typeof(elemento) <> 'string'
               OR vec_autorizacion.texto_positivo_valido(
                   elemento #>> '{}', 512
               ) IS NOT TRUE
       )
       OR (SELECT count(*) FROM jsonb_array_elements(p_valores)) <>
          (SELECT count(DISTINCT elemento #>> '{}')
             FROM jsonb_array_elements(p_valores) AS elemento) THEN
        RETURN NULL;
    END IF;
    FOR valor IN
        SELECT elemento #>> '{}'
          FROM jsonb_array_elements(p_valores) AS elemento
         ORDER BY elemento #>> '{}'
    LOOP
        IF preimagen = ''::bytea THEN
            preimagen := convert_to(
                octet_length(convert_to(p_esquema, 'UTF8'))::text || ':' ||
                p_esquema || E'\n', 'UTF8'
            );
        END IF;
        preimagen := preimagen || convert_to(
            octet_length(convert_to(valor, 'UTF8'))::text || ':' ||
            valor || E'\n', 'UTF8'
        );
    END LOOP;
    IF preimagen = ''::bytea THEN
        preimagen := convert_to(
            octet_length(convert_to(p_esquema, 'UTF8'))::text || ':' ||
            p_esquema || E'\n', 'UTF8'
        );
    END IF;
    RETURN encode(sha256(preimagen), 'hex');
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.decision_canonica_documental_v4_estructura_valida(
    p_decision jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF p_decision IS NULL OR jsonb_typeof(p_decision) <> 'object' THEN
        RETURN false;
    END IF;
    RETURN (SELECT count(*) FROM jsonb_object_keys(p_decision)) = 30
       AND p_decision ?& ARRAY[
           'esquema', 'decision_ref', 'concedida', 'codigo', 'principal_id',
           'perfil_activo_ref', 'accion', 'recurso_ref', 'modulo_id',
           'tipo_recurso', 'contexto_recurso_huella_sha256', 'finalidad',
           'correlacion_ref', 'vinculo_autenticacion_actor', 'asignacion_ref',
           'asignacion_huella_sha256', 'version_rol_ref',
           'version_rol_huella_sha256', 'control_vigencia_version_rol_ref',
           'control_vigencia_version_rol_revision',
           'control_vigencia_version_rol_huella_sha256',
           'revision_catalogo_politicas', 'catalogo_politicas_huella_sha256',
           'politicas_evaluadas', 'politicas_aplicables', 'garantia_minima',
           'campos_permitidos', 'obligaciones', 'emitida_en', 'valida_hasta'
       ];
END
$funcion$;

CREATE FUNCTION vec_autorizacion.revalidar_decision_ejecucion_documental_v4(
    p_aplicacion jsonb,
    p_decision_canonica bytea
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    decision vec_autorizacion.decision_autorizacion%ROWTYPE;
    decision_canonica jsonb;
    decision_esperada jsonb;
    vinculo jsonb;
    asignacion_actual record;
    rol_actual record;
    catalogo_actual record;
    manifiesto_actual jsonb;
    referencias_actuales jsonb;
    politicas_evaluadas jsonb;
    politicas_aplicables jsonb;
    instante timestamptz(6);
BEGIN
    IF p_aplicacion IS NULL OR jsonb_typeof(p_aplicacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_aplicacion)) <> 25
       OR NOT (p_aplicacion ?& ARRAY[
           'esquema', 'decision_ref', 'huella_plan_sha256', 'efecto_ref',
           'esquema_huella_decision', 'huella_decision_sha256',
           'perfil_activo_ref', 'contexto_actor_huella_sha256', 'accion',
           'recurso_ref', 'modulo_id', 'tipo_recurso', 'huella_recurso_sha256',
           'huella_ambitos_sha256', 'finalidad', 'correlacion_ref',
           'huella_campos_permitidos_sha256', 'huella_obligaciones_sha256',
           'huella_cumplimientos_sha256', 'verificada_en', 'vinculada_en',
           'solicitada_en', 'valida_hasta',
           'huella_solicitud_vinculada_sha256',
           'huella_solicitud_aplicacion_sha256'
       ]) OR octet_length(p_decision_canonica) NOT BETWEEN 128 AND 524288 THEN
        RETURN false;
    END IF;
    IF p_aplicacion ->> 'esquema' <>
           'vec.documentos.autorizacion-ejecucion.solicitud-aplicacion.v4'
       OR p_aplicacion ->> 'esquema_huella_decision' <>
           'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
       OR p_aplicacion ->> 'accion' <>
           'vec.documentos.ejecucion.ejecutar_plan_v4'
       OR encode(sha256(p_decision_canonica), 'hex') <>
           p_aplicacion ->> 'huella_decision_sha256'
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'huella_plan_sha256', 'huella_decision_sha256',
               'contexto_actor_huella_sha256', 'huella_recurso_sha256',
               'huella_ambitos_sha256', 'huella_campos_permitidos_sha256',
               'huella_obligaciones_sha256', 'huella_cumplimientos_sha256',
               'huella_solicitud_vinculada_sha256',
               'huella_solicitud_aplicacion_sha256'
           ]) AS clave
           WHERE jsonb_typeof(p_aplicacion -> clave) <> 'string'
              OR (p_aplicacion ->> clave) !~ '^[0-9a-f]{64}$'
       ) OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'verificada_en', 'vinculada_en', 'solicitada_en', 'valida_hasta'
           ]) AS clave
           WHERE vec_autorizacion.instante_utc_microsegundo_valido(
               p_aplicacion ->> clave
           ) IS NOT TRUE
       ) THEN
        RETURN false;
    END IF;

    decision_canonica := convert_from(p_decision_canonica, 'UTF8')::jsonb;
    IF vec_autorizacion.decision_canonica_documental_v4_estructura_valida(
           decision_canonica
       ) IS NOT TRUE OR decision_canonica ->> 'esquema' <>
           'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
       OR decision_canonica ->> 'decision_ref' <>
           p_aplicacion ->> 'decision_ref' THEN
        RETURN false;
    END IF;
    vinculo := decision_canonica -> 'vinculo_autenticacion_actor';
    IF jsonb_typeof(vinculo) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(vinculo)) <> 25
       OR NOT (vinculo ?& ARRAY[
           'bloque_version', 'autenticacion_ref',
           'autenticacion_huella_sha256', 'asercion_ref', 'sesion_ref',
           'control_sesion_ref', 'control_sesion_revision',
           'control_sesion_huella_sha256', 'cuenta_ref',
           'cuenta_ordinaria_ref', 'principal_id', 'perfil_activo_ref',
           'cuenta_privilegiada', 'superficie', 'metodo_observado',
           'garantia_observada', 'politica_garantia_ref',
           'politica_garantia_huella_sha256',
           'autenticacion_verificada_en', 'sesion_emitida_en',
           'sesion_valida_hasta', 'sesion_revalidada_en',
           'contexto_actor_ref', 'contexto_actor_version',
           'contexto_actor_huella_sha256'
       ]) THEN
        RETURN false;
    END IF;

    SELECT * INTO decision
      FROM vec_autorizacion.decision_autorizacion
     WHERE decision_ref = p_aplicacion ->> 'decision_ref'
     FOR SHARE;
    IF NOT FOUND OR decision.concedida IS NOT TRUE OR decision.codigo <> 'concedida'
       OR jsonb_typeof(decision.documento -> 'vinculo_autenticacion_actor') <>
          'object' THEN
        RETURN false;
    END IF;

    SELECT COALESCE(jsonb_agg(
               jsonb_build_object(
                   'referencia', referencia,
                   'huella_sha256', decision.documento ->
                       'politicas_evaluadas_huellas_sha256' ->> referencia
               ) ORDER BY referencia
           ), '[]'::jsonb)
      INTO politicas_evaluadas
      FROM jsonb_array_elements_text(
          decision.documento -> 'politicas_evaluadas_refs'
      ) AS referencia;
    SELECT COALESCE(jsonb_agg(
               jsonb_build_object(
                   'referencia', referencia,
                   'huella_sha256', decision.documento ->
                       'politicas_huellas_sha256' ->> referencia
               ) ORDER BY referencia
           ), '[]'::jsonb)
      INTO politicas_aplicables
      FROM jsonb_array_elements_text(
          decision.documento -> 'politicas_refs'
      ) AS referencia;

    decision_esperada := jsonb_build_object(
        'esquema', 'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        'decision_ref', decision.decision_ref,
        'concedida', decision.concedida,
        'codigo', decision.codigo,
        'principal_id', decision.principal_id,
        'perfil_activo_ref', decision.perfil_activo_ref,
        'accion', decision.accion,
        'recurso_ref', decision.recurso_ref,
        'modulo_id', decision.modulo_id,
        'tipo_recurso', decision.tipo_recurso,
        'contexto_recurso_huella_sha256', decision.contexto_recurso_huella_sha256,
        'finalidad', decision.finalidad,
        'correlacion_ref', decision.correlacion_ref,
        'vinculo_autenticacion_actor', decision.documento -> 'vinculo_autenticacion_actor',
        'asignacion_ref', decision.asignacion_ref,
        'asignacion_huella_sha256', decision.asignacion_huella_sha256,
        'version_rol_ref', decision.version_rol_ref,
        'version_rol_huella_sha256', decision.version_rol_huella_sha256,
        'control_vigencia_version_rol_ref', decision.control_vigencia_version_rol_ref,
        'control_vigencia_version_rol_revision', decision.control_vigencia_version_rol_revision,
        'control_vigencia_version_rol_huella_sha256',
            decision.control_vigencia_version_rol_huella_sha256,
        'revision_catalogo_politicas', decision.revision_catalogo_politicas,
        'catalogo_politicas_huella_sha256', decision.catalogo_politicas_huella_sha256,
        'politicas_evaluadas', politicas_evaluadas,
        'politicas_aplicables', politicas_aplicables,
        'garantia_minima', decision.documento ->> 'garantia_minima',
        'campos_permitidos', (
            SELECT COALESCE(jsonb_agg(valor ORDER BY valor), '[]'::jsonb)
              FROM jsonb_array_elements_text(
                  decision.documento -> 'campos_permitidos'
              ) AS valor
        ),
        'obligaciones', (
            SELECT COALESCE(jsonb_agg(valor ORDER BY valor), '[]'::jsonb)
              FROM jsonb_array_elements_text(
                  decision.documento -> 'obligaciones'
              ) AS valor
        ),
        'emitida_en', to_char(
            decision.emitida_en AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'valida_hasta', to_char(
            decision.valida_hasta AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    );
    IF decision_canonica IS DISTINCT FROM decision_esperada
       OR decision.perfil_activo_ref IS DISTINCT FROM p_aplicacion ->> 'perfil_activo_ref'
       OR decision.accion IS DISTINCT FROM p_aplicacion ->> 'accion'
       OR decision.recurso_ref IS DISTINCT FROM p_aplicacion ->> 'recurso_ref'
       OR decision.modulo_id IS DISTINCT FROM p_aplicacion ->> 'modulo_id'
       OR decision.tipo_recurso IS DISTINCT FROM p_aplicacion ->> 'tipo_recurso'
       OR decision.contexto_recurso_huella_sha256 IS DISTINCT FROM
          p_aplicacion ->> 'huella_recurso_sha256'
       OR decision.finalidad IS DISTINCT FROM p_aplicacion ->> 'finalidad'
       OR decision.correlacion_ref IS DISTINCT FROM p_aplicacion ->> 'correlacion_ref'
       OR decision.valida_hasta IS DISTINCT FROM
          (p_aplicacion ->> 'valida_hasta')::timestamptz
       OR vinculo ->> 'contexto_actor_huella_sha256' IS DISTINCT FROM
          p_aplicacion ->> 'contexto_actor_huella_sha256'
       OR vec_autorizacion.huella_lista_documental_v4(
              'vec.documentos.autorizacion-ejecucion.campos.v4',
              decision_canonica -> 'campos_permitidos'
          ) IS DISTINCT FROM p_aplicacion ->> 'huella_campos_permitidos_sha256'
       -- El perfil V4 permanece cerrado a obligaciones hasta disponer de una
       -- evidencia tipada, durable y revocable de cada cumplimiento.
       OR decision_canonica -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR p_aplicacion ->> 'huella_obligaciones_sha256' IS DISTINCT FROM
          vec_autorizacion.huella_lista_documental_v4(
              'vec.documentos.autorizacion-ejecucion.obligaciones.v4', '[]'::jsonb
          )
       OR p_aplicacion ->> 'huella_cumplimientos_sha256' IS DISTINCT FROM
          vec_autorizacion.huella_lista_documental_v4(
              'vec.documentos.autorizacion-ejecucion.cumplimientos.v4', '[]'::jsonb
          ) THEN
        RETURN false;
    END IF;

    SELECT actual.asignacion_ref, asignacion.principal_id,
           asignacion.version_rol_ref, asignacion.huella_sha256
      INTO asignacion_actual
      FROM vec_autorizacion.asignacion_perfil_actual AS actual
      JOIN vec_autorizacion.asignacion_perfil AS asignacion
        ON asignacion.perfil_activo_ref = actual.perfil_activo_ref
       AND asignacion.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = decision.perfil_activo_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion_actual.asignacion_ref IS DISTINCT FROM decision.asignacion_ref
       OR asignacion_actual.principal_id IS DISTINCT FROM decision.principal_id
       OR asignacion_actual.version_rol_ref IS DISTINCT FROM decision.version_rol_ref
       OR asignacion_actual.huella_sha256 IS DISTINCT FROM decision.asignacion_huella_sha256 THEN
        RETURN false;
    END IF;

    SELECT rol.huella_sha256, control.version_rol_ref, control.revision,
           control.estado, control.huella_sha256 AS huella_control
      INTO rol_actual
      FROM vec_autorizacion.version_rol AS rol
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = actual.version_rol_ref
       AND control.revision = actual.revision
     WHERE rol.version_rol_ref = decision.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND OR rol_actual.huella_sha256 IS DISTINCT FROM decision.version_rol_huella_sha256
       OR rol_actual.version_rol_ref IS DISTINCT FROM decision.control_vigencia_version_rol_ref
       OR rol_actual.revision IS DISTINCT FROM decision.control_vigencia_version_rol_revision
       OR rol_actual.estado IS DISTINCT FROM 'habilitada'
       OR rol_actual.huella_control IS DISTINCT FROM
          decision.control_vigencia_version_rol_huella_sha256 THEN
        RETURN false;
    END IF;

    SELECT revision, huella_sha256 INTO catalogo_actual
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true FOR UPDATE;
    IF NOT FOUND OR catalogo_actual.revision IS DISTINCT FROM decision.revision_catalogo_politicas
       OR catalogo_actual.huella_sha256 IS DISTINCT FROM decision.catalogo_politicas_huella_sha256 THEN
        RETURN false;
    END IF;

    SELECT COALESCE(jsonb_object_agg(
               politica.politica_ref, politica.huella_sha256
               ORDER BY politica.politica_ref
           ), '{}'::jsonb),
           COALESCE(jsonb_agg(
               politica.politica_ref ORDER BY politica.politica_ref
           ), '[]'::jsonb)
      INTO manifiesto_actual, referencias_actuales
      FROM vec_autorizacion.politica_restrictiva_actual AS actual
      JOIN vec_autorizacion.politica_restrictiva AS politica
        ON politica.politica_id = actual.politica_id
       AND politica.politica_ref = actual.politica_ref;
    IF manifiesto_actual IS DISTINCT FROM decision.politicas_evaluadas_manifesto
       OR referencias_actuales IS DISTINCT FROM (
           SELECT COALESCE(jsonb_agg(referencia ORDER BY referencia), '[]'::jsonb)
             FROM jsonb_array_elements_text(
                 decision.documento -> 'politicas_evaluadas_refs'
             ) AS referencia
       ) THEN
        RETURN false;
    END IF;

    instante := clock_timestamp();
    IF instante < decision.emitida_en OR instante >= decision.valida_hasta
       OR vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
           vinculo, decision.principal_id, decision.perfil_activo_ref,
           p_aplicacion ->> 'contexto_actor_huella_sha256',
           decision.emitida_en, decision.valida_hasta, instante
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation OR datetime_field_overflow
        OR character_not_in_repertoire OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion.huella_lista_documental_v4(text, jsonb)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_autorizacion.decision_canonica_documental_v4_estructura_valida(jsonb)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_ejecucion_documental_v4(jsonb, bytea)
    FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_ejecucion_documental_v4_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_ejecucion_documental_v4(jsonb, bytea)
    TO vec_ejecucion_documental_v4_propietario;
COMMIT;
