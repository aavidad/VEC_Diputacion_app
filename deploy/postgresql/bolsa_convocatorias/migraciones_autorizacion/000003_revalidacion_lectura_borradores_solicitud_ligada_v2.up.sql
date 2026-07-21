-- 000002 conserva un contrato historico V1 pese a su nombre. Esta frontera
-- nueva no lo reemplaza: acepta exclusivamente decisiones V2 registradas,
-- ligadas a solicitud y motivo opaco publicado.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:lectura_borradores_v2:000003', 0
));

DO $prevalidacion$
BEGIN
    IF to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.materializar_documento_comun_decision_v2(jsonb)'
       ) IS NULL
       OR to_regclass(
           'vec_autorizacion.motivo_v2_checkpoint_origen'
       ) IS NULL
       OR to_regclass(
           'vec_autorizacion.motivo_v2_catalogo_publicado'
       ) IS NULL
       OR to_regclass('vec_autorizacion.motivo_v2_entrada') IS NULL
       OR to_regclass('vec_autorizacion.motivo_v2_retirada') IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(jsonb,text,text,text,timestamp with time zone,timestamp with time zone,timestamp with time zone)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(timestamp with time zone,timestamp with time zone)'
       ) IS NOT NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_bolsa_convocatorias_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       )
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(jsonb,bytea,bytea,text,text,text,text,jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar lectura V2 de borradores';
    END IF;
END
$prevalidacion$;

-- Predicado privado y determinista para que el limite de frescura pueda
-- probarse exactamente. La frontera productiva solo le entrega el reloj
-- autoritativo obtenido despues de todos los locks; [verificada,+30s).
CREATE FUNCTION
vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
    p_verificada_en timestamptz,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_verificada_en IS NOT NULL
       AND p_instante IS NOT NULL
       AND p_instante >= p_verificada_en
       AND NOT (
           p_instante - p_verificada_en >= interval '30 seconds'
       )
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
        timestamptz,timestamptz
    ) FROM PUBLIC;

CREATE FUNCTION
vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea,
    p_accion text,
    p_clase_recurso text,
    p_recurso_ref text,
    p_finalidad text,
    p_campos_exactos jsonb
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    decision record;
    documento_v2 jsonb;
    documento_comun jsonb;
    recurso jsonb;
    ambitos jsonb;
    motivo jsonb;
    referencia_motivo jsonb;
    vinculo jsonb;
    verificada_en timestamptz;
    asignacion_actual record;
    rol_actual record;
    catalogo_actual record;
    manifiesto_actual jsonb;
    referencias_actuales jsonb;
    referencias_decision jsonb;
    concesiones_coincidentes integer;
    instante timestamptz(6);
BEGIN
    IF p_prueba IS NULL OR jsonb_typeof(p_prueba) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_prueba)) <> 5
       OR NOT (p_prueba ?& ARRAY[
           'esquema_huella','decision_ref','huella_decision_sha256',
           'verificada_en','principal_ref'
       ])
       OR p_prueba ->> 'esquema_huella' IS DISTINCT FROM
          'vec.autorizacion.decision.reforzada.v2.solicitud-ligada'
       OR jsonb_typeof(p_prueba -> 'decision_ref') <> 'string'
       OR octet_length(p_prueba ->> 'decision_ref') NOT BETWEEN 1 AND 512
       OR (p_prueba ->> 'decision_ref') !~ '^[^*[:space:][:cntrl:]]+$'
       OR jsonb_typeof(p_prueba -> 'principal_ref') <> 'string'
       OR octet_length(p_prueba ->> 'principal_ref') NOT BETWEEN 1 AND 512
       OR (p_prueba ->> 'principal_ref') !~ '^[^*[:space:][:cntrl:]]+$'
       OR (p_prueba ->> 'huella_decision_sha256') !~ '^[0-9a-f]{64}$'
       OR (p_prueba ->> 'verificada_en') !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR p_decision_canonica IS NULL
       OR octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR p_recurso_canonico IS NULL
       OR octet_length(p_recurso_canonico) NOT BETWEEN 1 AND 65536
       OR encode(sha256(p_decision_canonica), 'hex') IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256'
       OR p_finalidad IS DISTINCT FROM 'consulta_interna_convocatorias'
       OR p_campos_exactos IS DISTINCT FROM
          '["version_convocatoria"]'::jsonb THEN
        RETURN false;
    END IF;

    IF p_accion = 'bolsa.convocatoria.borrador.listar' THEN
        IF p_clase_recurso IS DISTINCT FROM
              'coleccion_versiones_convocatoria_gobernada'
           OR p_recurso_ref !~ '^borradores:org_[a-z0-9]{16,80}$' THEN
            RETURN false;
        END IF;
    ELSIF p_accion = 'bolsa.convocatoria.borrador.consultar' THEN
        IF p_clase_recurso IS DISTINCT FROM
              'version_convocatoria_gobernada'
           OR p_recurso_ref !~ '^.+#[1-9][0-9]{0,18}$' THEN
            RETURN false;
        END IF;
    ELSE
        RETURN false;
    END IF;

    BEGIN
        documento_v2 := convert_from(p_decision_canonica, 'UTF8')::jsonb;
        documento_comun :=
            vec_autorizacion.materializar_documento_comun_decision_v2(
                documento_v2
            );
        recurso := convert_from(p_recurso_canonico, 'UTF8')::jsonb;
        verificada_en := (p_prueba ->> 'verificada_en')::timestamptz;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception
            OR datetime_field_overflow THEN
            RETURN false;
    END;
    IF documento_comun IS NULL
       OR to_char(verificada_en AT TIME ZONE 'UTC',
                  'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') <>
          p_prueba ->> 'verificada_en'
       OR documento_v2 ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.decision.reforzada.v2.solicitud-ligada'
       OR documento_v2 ->> 'esquema_huella_solicitud' IS DISTINCT FROM
          'vec.autorizacion.solicitud.v2.efectiva-minimizada'
       OR (documento_v2 ->> 'solicitud_huella_sha256') !~ '^[0-9a-f]{64}$'
       OR documento_v2 ->> 'solicitud_huella_sha256' = repeat('0', 64)
       OR documento_v2 ->> 'esquema_huella_motivo' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR (documento_v2 ->> 'motivo_huella_sha256') !~ '^[0-9a-f]{64}$'
       OR documento_v2 ->> 'motivo_huella_sha256' = repeat('0', 64)
       OR jsonb_typeof(recurso) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(recurso)) <> 2
       OR NOT (recurso ?& ARRAY['ambitos','atributos'])
       OR recurso -> 'atributos' IS DISTINCT FROM '{}'::jsonb
       OR jsonb_typeof(recurso -> 'ambitos') <> 'object' THEN
        RETURN false;
    END IF;
    ambitos := recurso -> 'ambitos';
    IF (SELECT count(*) FROM jsonb_object_keys(ambitos)) NOT BETWEEN 1 AND 2
       OR ambitos ->> 'organizacion_ref' !~ '^org_[a-z0-9]{16,80}$'
       OR (ambitos ? 'unidad_gestion_ref' AND
           ambitos ->> 'unidad_gestion_ref' !~ '^uni_[a-z0-9]{16,80}$')
       OR (p_accion = 'bolsa.convocatoria.borrador.listar' AND
           p_recurso_ref IS DISTINCT FROM
           'borradores:' || (ambitos ->> 'organizacion_ref')) THEN
        RETURN false;
    END IF;

    -- Orden unico compartido con autorizacion/000004: motivo, decision,
    -- asignacion, rol, catalogo de politicas, sesion y ContextoActor. El
    -- checkpoint impide retirar el motivo mientras se autoriza el efecto.
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR SHARE;
    IF NOT FOUND THEN
        RETURN false;
    END IF;

    SELECT registro.* INTO decision
      FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
           AS registro
     WHERE registro.decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE OF registro;
    IF NOT FOUND
       OR decision.huella_decision_sha256 IS DISTINCT FROM
          p_prueba ->> 'huella_decision_sha256'
       OR decision.decision_canonica IS DISTINCT FROM p_decision_canonica
       OR decision.documento_v2 IS DISTINCT FROM documento_v2
       OR decision.documento_comun IS DISTINCT FROM documento_comun
       OR decision.principal_id IS DISTINCT FROM
          p_prueba ->> 'principal_ref'
       OR decision.accion IS DISTINCT FROM p_accion
       OR decision.modulo_id IS DISTINCT FROM 'bolsa'
       OR decision.tipo_recurso IS DISTINCT FROM p_clase_recurso
       OR decision.recurso_ref IS DISTINCT FROM p_recurso_ref
       OR decision.contexto_recurso_huella_sha256 IS DISTINCT FROM
          encode(sha256(p_recurso_canonico), 'hex')
       OR decision.finalidad IS DISTINCT FROM p_finalidad
       OR decision.correlacion_ref !~ '^correlacion_[0-9a-f]{32}$'
       OR decision.solicitud_huella_sha256 IS DISTINCT FROM
          documento_v2 ->> 'solicitud_huella_sha256'
       OR decision.motivo_huella_sha256 IS DISTINCT FROM
          documento_v2 ->> 'motivo_huella_sha256'
       OR documento_v2 ->> 'concedida' <> 'true'
       OR documento_v2 ->> 'codigo' <> 'concedida'
       OR documento_v2 ->> 'garantia_minima' <> 'alto'
       OR documento_v2 -> 'campos_permitidos' IS DISTINCT FROM
          p_campos_exactos
       OR documento_v2 -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR decision.emitida_en > verificada_en
       OR verificada_en >= decision.valida_hasta THEN
        RETURN false;
    END IF;

    BEGIN
        motivo := convert_from(decision.motivo_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception THEN
            RETURN false;
    END;
    IF encode(sha256(decision.motivo_canonico), 'hex') IS DISTINCT FROM
          decision.motivo_huella_sha256
       OR jsonb_typeof(motivo) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(motivo)) <> 2
       OR NOT (motivo ?& ARRAY['esquema','referencia'])
       OR motivo ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR jsonb_typeof(motivo -> 'referencia') <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(
              motivo -> 'referencia'
          )) <> 4 THEN
        RETURN false;
    END IF;
    referencia_motivo := motivo -> 'referencia';
    IF decision.motivo_catalogo_id IS DISTINCT FROM
          referencia_motivo ->> 'catalogo_id'
       OR decision.motivo_catalogo_version IS DISTINCT FROM
          (referencia_motivo ->> 'catalogo_version')::integer
       OR decision.motivo_catalogo_huella_sha256 IS DISTINCT FROM
          referencia_motivo ->> 'catalogo_huella_sha256'
       OR decision.motivo_entrada_clave IS DISTINCT FROM
          referencia_motivo ->> 'entrada_clave' THEN
        RETURN false;
    END IF;

    vinculo := documento_v2 -> 'vinculo_autenticacion_actor';
    IF jsonb_typeof(vinculo) <> 'object'
       OR jsonb_typeof(vinculo -> 'cuenta_privilegiada') <> 'boolean'
       OR vinculo ->> 'garantia_observada' <> 'alto'
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
          documento_v2 ->> 'asignacion_huella_sha256' THEN
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
          documento_v2 ->> 'version_rol_huella_sha256'
       OR rol_actual.version_rol_ref IS DISTINCT FROM
          decision.control_vigencia_version_rol_ref
       OR rol_actual.revision IS DISTINCT FROM
          decision.control_vigencia_version_rol_revision
       OR rol_actual.estado IS DISTINCT FROM 'habilitada'
       OR rol_actual.huella_control IS DISTINCT FROM
          documento_v2 ->> 'control_vigencia_version_rol_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT count(*) INTO concesiones_coincidentes
      FROM jsonb_array_elements(rol_actual.documento -> 'concesiones') AS c
     WHERE c ->> 'accion' = p_accion
       AND c ->> 'modulo_id' = 'bolsa'
       AND c ->> 'tipo_recurso' = p_clase_recurso
       AND COALESCE(c -> 'obligaciones', '[]'::jsonb) = '[]'::jsonb
       AND (SELECT COALESCE(jsonb_agg(
                  valor ORDER BY valor COLLATE "C"
              ), '[]'::jsonb)
              FROM jsonb_array_elements_text(
                  COALESCE(c -> 'campos_permitidos', '[]'::jsonb)
              ) AS valor) = p_campos_exactos
       AND EXISTS (
           SELECT 1 FROM jsonb_array_elements_text(c -> 'finalidades') AS f
            WHERE f = p_finalidad
       );
    IF concesiones_coincidentes <> 1 THEN
        RETURN false;
    END IF;

    SELECT revision, huella_sha256 INTO catalogo_actual
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND
       OR catalogo_actual.revision IS DISTINCT FROM
          (decision.documento_comun ->>
           'revision_catalogo_politicas')::numeric
       OR catalogo_actual.huella_sha256 IS DISTINCT FROM
          decision.documento_comun ->> 'catalogo_politicas_huella_sha256' THEN
        RETURN false;
    END IF;

    SELECT COALESCE(jsonb_object_agg(
               politica.politica_ref, politica.huella_sha256
               ORDER BY politica.politica_ref COLLATE "C"
           ), '{}'::jsonb),
           COALESCE(jsonb_agg(
               politica.politica_ref
               ORDER BY politica.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO manifiesto_actual, referencias_actuales
      FROM vec_autorizacion.politica_restrictiva_actual AS actual
      JOIN vec_autorizacion.politica_restrictiva AS politica
        ON politica.politica_id = actual.politica_id
       AND politica.politica_ref = actual.politica_ref;
    SELECT COALESCE(jsonb_agg(
               referencia ORDER BY referencia COLLATE "C"
           ), '[]'::jsonb)
      INTO referencias_decision
      FROM jsonb_array_elements_text(
          decision.documento_comun -> 'politicas_evaluadas_refs'
      ) AS referencia;
    IF manifiesto_actual IS DISTINCT FROM
          decision.documento_comun ->
              'politicas_evaluadas_huellas_sha256'
       OR referencias_actuales IS DISTINCT FROM referencias_decision
       OR EXISTS (
           SELECT 1
             FROM jsonb_each(
                 decision.documento_comun -> 'politicas_huellas_sha256'
             ) AS aplicada
            WHERE manifiesto_actual -> aplicada.key IS DISTINCT FROM
                  aplicada.value
       ) THEN
        RETURN false;
    END IF;

    -- Primera pasada: adquiere, en el ultimo tramo del orden comun, los
    -- punteros actuales de sesion y ContextoActor. La fecha de emision es
    -- solo evidencia historica y no se usa como reloj de autorizacion.
    IF vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
           vinculo, decision.principal_id, decision.perfil_activo_ref,
           vinculo ->> 'contexto_actor_huella_sha256', decision.emitida_en,
           decision.valida_hasta, decision.emitida_en
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;

    -- Unico reloj autoritativo, obtenido despues de todos los locks. La
    -- segunda pasada no puede esperar y aplica ventanas half-open [desde,
    -- hasta) a decision, prueba, motivo, sesion y ContextoActor.
    instante := clock_timestamp();
    IF vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
           verificada_en, instante
       ) IS NOT TRUE
       OR instante < decision.emitida_en
       OR instante >= decision.valida_hasta
       OR vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
              vinculo, decision.principal_id, decision.perfil_activo_ref,
              vinculo ->> 'contexto_actor_huella_sha256',
              decision.emitida_en, decision.valida_hasta, instante
          ) IS NOT TRUE
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.motivo_v2_catalogo_publicado AS catalogo
             JOIN vec_autorizacion.motivo_v2_entrada AS entrada
               ON entrada.catalogo_id = catalogo.catalogo_id
              AND entrada.catalogo_version = catalogo.catalogo_version
            WHERE catalogo.catalogo_id = decision.motivo_catalogo_id
              AND catalogo.catalogo_version =
                  decision.motivo_catalogo_version
              AND catalogo.catalogo_huella_publicada_sha256 =
                  decision.motivo_catalogo_huella_sha256
              AND entrada.entrada_clave = decision.motivo_entrada_clave
              AND catalogo.publicado_en <= instante
              AND entrada.vigente_desde <= instante
              AND (entrada.vigente_hasta IS NULL
                   OR instante < entrada.vigente_hasta)
              AND NOT EXISTS (
                  SELECT 1
                    FROM vec_autorizacion.motivo_v2_retirada AS retirada
                   WHERE retirada.catalogo_id = catalogo.catalogo_id
                     AND retirada.catalogo_version =
                         catalogo.catalogo_version
              )
       ) THEN
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
    vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
        jsonb,bytea,bytea,text,text,text,text,jsonb
    ) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_bolsa_convocatorias_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
        jsonb,bytea,bytea,text,text,text,text,jsonb
    ) TO vec_bolsa_convocatorias_propietario;

COMMENT ON FUNCTION
    vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
        timestamptz,timestamptz
    ) IS
    'Predicado privado de frescura half-open [verificada_en, verificada_en+30s). La frontera de lectura aporta exclusivamente su reloj post-lock.';

COMMENT ON FUNCTION
    vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
        jsonb,bytea,bytea,text,text,text,text,jsonb
    ) IS
    'Revalida lectura de borradores con decision V2 registrada, solicitud ligada, motivo opaco publicado, RBAC/ABAC, garantia alta y sesion interna. 000002 conserva solo el contrato historico V1.';
COMMIT;
