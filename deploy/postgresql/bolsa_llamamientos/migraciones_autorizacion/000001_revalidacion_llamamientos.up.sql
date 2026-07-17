-- Frontera estrecha instalada junto al registro de autorizacion. Revalida una
-- decision V1 exacta y su configuracion vigente; no sustituye la atestacion
-- criptografica de procedencia, que se exige adicionalmente en el repositorio.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regclass('vec_autorizacion.decision_autorizacion') IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(jsonb,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_bolsa_llamamientos_propietario'
              AND NOT rolcanlogin
       )
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar la frontera de llamamientos';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea,
    p_accion text,
    p_clase_recurso text,
    p_recurso_ref text,
    p_campos_exactos jsonb,
    p_instante timestamptz
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
    decision_canonica_esperada jsonb;
    recurso_canonico jsonb;
    asignacion_actual record;
    rol_actual record;
    catalogo_actual record;
    manifiesto_actual jsonb;
    referencias_actuales jsonb;
    politicas_evaluadas jsonb;
    politicas_aplicables jsonb;
    campos_ordenados jsonb;
    campos_exigidos jsonb;
    obligaciones_ordenadas jsonb;
    concesiones_coincidentes integer;
    verificada_en timestamptz;
    vinculo jsonb;
BEGIN
    IF p_prueba IS NULL OR jsonb_typeof(p_prueba) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_prueba)) <> 5
       OR NOT (p_prueba ?& ARRAY[
           'esquema_huella', 'decision_ref', 'huella_decision_sha256',
           'verificada_en', 'principal_ref'
       ])
       OR p_prueba ->> 'esquema_huella' IS DISTINCT FROM
          'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
       OR vec_autorizacion.texto_positivo_valido(
           p_prueba ->> 'decision_ref', 512
       ) IS NOT TRUE
       OR (p_prueba ->> 'huella_decision_sha256') !~ '^[0-9a-f]{64}$'
       OR vec_autorizacion.instante_utc_microsegundo_valido(
           p_prueba ->> 'verificada_en'
       ) IS NOT TRUE
       OR vec_autorizacion.texto_positivo_valido(
           p_prueba ->> 'principal_ref', 512
       ) IS NOT TRUE
       OR p_decision_canonica IS NULL
       OR p_recurso_canonico IS NULL
       OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 1048576
       OR octet_length(p_recurso_canonico) NOT BETWEEN 1 AND 65536
       OR encode(sha256(p_decision_canonica), 'hex') IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256'
       OR p_accion <> 'bolsa.llamamiento.proponer'
       OR p_clase_recurso <> 'necesidad_cobertura'
       OR vec_autorizacion.texto_positivo_valido(
           p_recurso_ref, 512
       ) IS NOT TRUE
       OR jsonb_typeof(p_campos_exactos) <> 'array'
       OR vec_autorizacion.lista_positiva_valida(
           p_campos_exactos, false
       ) IS NOT TRUE
       OR p_instante IS NULL THEN
        RETURN false;
    END IF;

    campos_exigidos := '[]'::jsonb;
    IF (SELECT COALESCE(jsonb_agg(valor ORDER BY valor), '[]'::jsonb)
          FROM jsonb_array_elements_text(p_campos_exactos) AS valor)
       IS DISTINCT FROM campos_exigidos THEN
        RETURN false;
    END IF;

    BEGIN
        decision_canonica := convert_from(p_decision_canonica, 'UTF8')::jsonb;
        recurso_canonico := convert_from(p_recurso_canonico, 'UTF8')::jsonb;
        verificada_en := (p_prueba ->> 'verificada_en')::timestamptz;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception THEN
            RETURN false;
    END;

    IF jsonb_typeof(decision_canonica) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(decision_canonica)) <> 30
       OR NOT (decision_canonica ?& ARRAY[
           'esquema', 'decision_ref', 'concedida', 'codigo', 'principal_id',
           'perfil_activo_ref', 'accion', 'recurso_ref', 'modulo_id',
           'tipo_recurso', 'contexto_recurso_huella_sha256', 'finalidad',
           'correlacion_ref', 'vinculo_autenticacion_actor', 'asignacion_ref',
           'asignacion_huella_sha256', 'version_rol_ref',
           'version_rol_huella_sha256',
           'control_vigencia_version_rol_ref',
           'control_vigencia_version_rol_revision',
           'control_vigencia_version_rol_huella_sha256',
           'revision_catalogo_politicas',
           'catalogo_politicas_huella_sha256', 'politicas_evaluadas',
           'politicas_aplicables', 'garantia_minima', 'campos_permitidos',
           'obligaciones', 'emitida_en', 'valida_hasta'
       ])
       OR decision_canonica ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
       OR jsonb_typeof(recurso_canonico) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(recurso_canonico)) <> 2
       OR NOT (recurso_canonico ?& ARRAY['ambitos', 'atributos'])
       OR jsonb_typeof(recurso_canonico -> 'ambitos') <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(
              recurso_canonico -> 'ambitos'
          )) <> 2
       OR NOT ((recurso_canonico -> 'ambitos') ?& ARRAY[
              'categoria_ref', 'unidad_ref'
          ])
       OR vec_autorizacion.texto_positivo_valido(
              recurso_canonico -> 'ambitos' ->> 'categoria_ref', 512
          ) IS NOT TRUE
       OR vec_autorizacion.texto_positivo_valido(
              recurso_canonico -> 'ambitos' ->> 'unidad_ref', 512
          ) IS NOT TRUE
       OR recurso_canonico -> 'atributos' IS DISTINCT FROM '{}'::jsonb THEN
        RETURN false;
    END IF;

    SELECT * INTO decision
      FROM vec_autorizacion.decision_autorizacion
     WHERE decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE;
    IF NOT FOUND OR decision.concedida IS NOT TRUE
       OR decision.codigo <> 'concedida'
       OR decision.principal_id IS DISTINCT FROM
          p_prueba ->> 'principal_ref'
       OR decision.accion IS DISTINCT FROM p_accion
       OR decision.modulo_id IS DISTINCT FROM 'bolsa'
       OR decision.tipo_recurso IS DISTINCT FROM p_clase_recurso
       OR decision.recurso_ref IS DISTINCT FROM p_recurso_ref
       OR decision.finalidad IS DISTINCT FROM
          'gestion_propuestas_llamamiento'
       OR encode(sha256(p_recurso_canonico), 'hex') IS DISTINCT FROM
          decision.contexto_recurso_huella_sha256
       OR decision.documento -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR decision.documento ->> 'garantia_minima' IS DISTINCT FROM 'alto'
       OR decision.emitida_en > verificada_en
       OR verificada_en >= decision.valida_hasta
       OR p_instante < verificada_en
       OR p_instante - verificada_en > interval '30 seconds'
       OR p_instante < decision.emitida_en
       OR p_instante >= decision.valida_hasta THEN
        RETURN false;
    END IF;

    vinculo := decision.documento -> 'vinculo_autenticacion_actor';
    IF jsonb_typeof(vinculo) IS DISTINCT FROM 'object'
       OR jsonb_typeof(vinculo -> 'cuenta_privilegiada')
          IS DISTINCT FROM 'boolean'
       OR vinculo ->> 'garantia_observada' IS DISTINCT FROM 'alto'
       OR vinculo ->> 'metodo_observado' IS NULL
       OR vinculo ->> 'metodo_observado' = 'demo'
       OR NOT COALESCE((
           (vinculo ->> 'superficie' = 'interna_corporativa'
            AND vinculo -> 'cuenta_privilegiada' = 'false'::jsonb)
           OR
           (vinculo ->> 'superficie' = 'administracion_privilegiada'
            AND vinculo -> 'cuenta_privilegiada' = 'true'::jsonb)
       ), false) THEN
        RETURN false;
    END IF;

    SELECT COALESCE(jsonb_agg(valor ORDER BY valor), '[]'::jsonb)
      INTO campos_ordenados
      FROM jsonb_array_elements_text(
          decision.documento -> 'campos_permitidos'
      ) AS valor;
    SELECT COALESCE(jsonb_agg(valor ORDER BY valor), '[]'::jsonb)
      INTO obligaciones_ordenadas
      FROM jsonb_array_elements_text(
          decision.documento -> 'obligaciones'
      ) AS valor;
    SELECT COALESCE(jsonb_agg(
               jsonb_build_object(
                   'referencia', referencia,
                   'huella_sha256', decision.documento
                       -> 'politicas_evaluadas_huellas_sha256' ->> referencia
               ) ORDER BY referencia
           ), '[]'::jsonb)
      INTO politicas_evaluadas
      FROM jsonb_array_elements_text(
          decision.documento -> 'politicas_evaluadas_refs'
      ) AS referencia;
    SELECT COALESCE(jsonb_agg(
               jsonb_build_object(
                   'referencia', referencia,
                   'huella_sha256', decision.documento
                       -> 'politicas_huellas_sha256' ->> referencia
               ) ORDER BY referencia
           ), '[]'::jsonb)
      INTO politicas_aplicables
      FROM jsonb_array_elements_text(
          decision.documento -> 'politicas_refs'
      ) AS referencia;

    decision_canonica_esperada := jsonb_build_object(
        'esquema',
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        'decision_ref', decision.documento -> 'decision_ref',
        'concedida', decision.documento -> 'concedida',
        'codigo', decision.documento -> 'codigo',
        'principal_id', decision.documento -> 'principal_id',
        'perfil_activo_ref', decision.documento -> 'perfil_activo_ref',
        'accion', decision.documento -> 'accion',
        'recurso_ref', decision.documento -> 'recurso_ref',
        'modulo_id', decision.documento -> 'modulo_id',
        'tipo_recurso', decision.documento -> 'tipo_recurso',
        'contexto_recurso_huella_sha256',
            decision.documento -> 'contexto_recurso_huella_sha256',
        'finalidad', decision.documento -> 'finalidad',
        'correlacion_ref', decision.documento -> 'correlacion_ref',
        'vinculo_autenticacion_actor', vinculo,
        'asignacion_ref', decision.documento -> 'asignacion_ref',
        'asignacion_huella_sha256',
            decision.documento -> 'asignacion_huella_sha256',
        'version_rol_ref', decision.documento -> 'version_rol_ref',
        'version_rol_huella_sha256',
            decision.documento -> 'version_rol_huella_sha256',
        'control_vigencia_version_rol_ref',
            decision.documento -> 'control_vigencia_version_rol_ref',
        'control_vigencia_version_rol_revision',
            decision.documento -> 'control_vigencia_version_rol_revision',
        'control_vigencia_version_rol_huella_sha256',
            decision.documento
                -> 'control_vigencia_version_rol_huella_sha256',
        'revision_catalogo_politicas',
            decision.documento -> 'revision_catalogo_politicas',
        'catalogo_politicas_huella_sha256',
            decision.documento -> 'catalogo_politicas_huella_sha256',
        'politicas_evaluadas', politicas_evaluadas,
        'politicas_aplicables', politicas_aplicables,
        'garantia_minima', decision.documento -> 'garantia_minima',
        'campos_permitidos', campos_ordenados,
        'obligaciones', obligaciones_ordenadas,
        'emitida_en', decision.documento -> 'emitida_en',
        'valida_hasta', decision.documento -> 'valida_hasta'
    );
    IF decision_canonica IS DISTINCT FROM decision_canonica_esperada
       OR campos_ordenados IS DISTINCT FROM campos_exigidos THEN
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
       OR asignacion_actual.asignacion_ref IS DISTINCT FROM
          decision.asignacion_ref
       OR asignacion_actual.principal_id IS DISTINCT FROM
          decision.principal_id
       OR asignacion_actual.version_rol_ref IS DISTINCT FROM
          decision.version_rol_ref
       OR asignacion_actual.huella_sha256 IS DISTINCT FROM
          decision.asignacion_huella_sha256 THEN
        RETURN false;
    END IF;

    SELECT rol.huella_sha256, rol.documento,
           control.version_rol_ref, control.revision, control.estado,
           control.huella_sha256 AS huella_control
      INTO rol_actual
      FROM vec_autorizacion.version_rol AS rol
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = rol.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS control
        ON control.version_rol_ref = actual.version_rol_ref
       AND control.revision = actual.revision
     WHERE rol.version_rol_ref = decision.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR rol_actual.huella_sha256 IS DISTINCT FROM
          decision.version_rol_huella_sha256
       OR rol_actual.version_rol_ref IS DISTINCT FROM
          decision.control_vigencia_version_rol_ref
       OR rol_actual.revision IS DISTINCT FROM
          decision.control_vigencia_version_rol_revision
       OR rol_actual.estado IS DISTINCT FROM 'habilitada'
       OR rol_actual.huella_control IS DISTINCT FROM
          decision.control_vigencia_version_rol_huella_sha256 THEN
        RETURN false;
    END IF;

    SELECT count(*) INTO concesiones_coincidentes
      FROM jsonb_array_elements(rol_actual.documento -> 'concesiones') AS c
     WHERE c ->> 'accion' = p_accion
       AND c ->> 'modulo_id' = 'bolsa'
       AND c ->> 'tipo_recurso' = p_clase_recurso
       AND COALESCE(c -> 'obligaciones', '[]'::jsonb)
           IS NOT DISTINCT FROM '[]'::jsonb
       AND (SELECT COALESCE(jsonb_agg(valor ORDER BY valor), '[]'::jsonb)
              FROM jsonb_array_elements_text(
                  COALESCE(c -> 'campos_permitidos', '[]'::jsonb)
              ) AS valor) IS NOT DISTINCT FROM campos_exigidos
       AND EXISTS (
           SELECT 1
             FROM jsonb_array_elements_text(c -> 'finalidades') AS finalidad
            WHERE finalidad = 'gestion_propuestas_llamamiento'
       );
    IF concesiones_coincidentes <> 1 THEN
        RETURN false;
    END IF;

    SELECT revision, huella_sha256
      INTO catalogo_actual
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND
       OR catalogo_actual.revision IS DISTINCT FROM
          decision.revision_catalogo_politicas
       OR catalogo_actual.huella_sha256 IS DISTINCT FROM
          decision.catalogo_politicas_huella_sha256 THEN
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
    IF manifiesto_actual IS DISTINCT FROM
           decision.politicas_evaluadas_manifesto
       OR referencias_actuales IS DISTINCT FROM (
           SELECT COALESCE(
               jsonb_agg(referencia ORDER BY referencia), '[]'::jsonb
           )
             FROM jsonb_array_elements_text(
                 decision.documento -> 'politicas_evaluadas_refs'
             ) AS referencia
       )
       OR vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
           vinculo,
           decision.principal_id,
           decision.perfil_activo_ref,
           vinculo ->> 'contexto_actor_huella_sha256',
           decision.emitida_en,
           decision.valida_hasta,
           p_instante
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(
        jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
    ) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(
        jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
    ) TO vec_bolsa_llamamientos_propietario;

COMMENT ON FUNCTION
    vec_autorizacion.revalidar_decision_bolsa_llamamientos_v1(
        jsonb, bytea, bytea, text, text, text, jsonb, timestamptz
    ) IS
    'Revalida decision exacta para proponer llamamiento, RBAC/ABAC, garantia alta, superficie interna, sesion y ContextoActor. No acredita procedencia COSE.';
COMMIT;
