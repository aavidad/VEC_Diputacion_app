-- Frontera exterior V3. 000005 instala contrato, tablas y helpers.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
    )
);

-- Frontera de bytes: conserva exactamente el orden y escapes de encoding/json
-- del canon congelado en 61f3a6e. Se instala junto a la funcion exterior para
-- mantener 000005 por debajo del limite de tamano de migraciones nuevas.
CREATE OR REPLACE FUNCTION vec_autorizacion.decision_contexto_actor_v3_canonica(
    p_documento jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
      '{"esquema":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'esquema') ||
      ',"bloque_version":' || (p_documento -> 'bloque_version')::text ||
      ',"decision_ref":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'decision_ref') ||
      ',"concedida":' || (p_documento -> 'concedida')::text ||
      ',"codigo":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'codigo') ||
      ',"principal_id":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'principal_id') ||
      ',"perfil_activo_ref":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'perfil_activo_ref') ||
      ',"accion":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'accion') ||
      ',"recurso_ref":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'recurso_ref') ||
      ',"modulo_id":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'modulo_id') ||
      ',"tipo_recurso":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'tipo_recurso') ||
      ',"contexto_recurso_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'contexto_recurso_huella_sha256') ||
      ',"finalidad":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'finalidad') ||
      ',"correlacion_ref":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'correlacion_ref') ||
      ',"esquema_huella_solicitud":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'esquema_huella_solicitud') ||
      ',"solicitud_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'solicitud_huella_sha256') ||
      ',"esquema_huella_motivo":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'esquema_huella_motivo') ||
      ',"motivo_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'motivo_huella_sha256') ||
      ',"vinculo_autenticacion_actor":' || vec_autorizacion.vinculo_contexto_actor_v2_canonico(p_documento -> 'vinculo_autenticacion_actor') ||
      ',"asignacion_ref":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'asignacion_ref') ||
      ',"asignacion_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'asignacion_huella_sha256') ||
      ',"version_rol_ref":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'version_rol_ref') ||
      ',"version_rol_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'version_rol_huella_sha256') ||
      ',"control_vigencia_version_rol_ref":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'control_vigencia_version_rol_ref') ||
      ',"control_vigencia_version_rol_revision":' || (p_documento -> 'control_vigencia_version_rol_revision')::text ||
      ',"control_vigencia_version_rol_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'control_vigencia_version_rol_huella_sha256') ||
      ',"revision_catalogo_politicas":' || (p_documento -> 'revision_catalogo_politicas')::text ||
      ',"catalogo_politicas_huella_sha256":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'catalogo_politicas_huella_sha256') ||
      ',"politicas_evaluadas":' || vec_autorizacion.manifiesto_politicas_v3_canonico(p_documento -> 'politicas_evaluadas') ||
      ',"politicas_aplicables":' || vec_autorizacion.manifiesto_politicas_v3_canonico(p_documento -> 'politicas_aplicables') ||
      ',"garantia_minima":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'garantia_minima') ||
      ',"campos_permitidos":' || vec_autorizacion.lista_textos_v3_canonica(p_documento -> 'campos_permitidos') ||
      ',"obligaciones":' || vec_autorizacion.lista_textos_v3_canonica(p_documento -> 'obligaciones') ||
      ',"emitida_en":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'emitida_en') ||
      ',"valida_hasta":' || vec_autorizacion.texto_json_go_v3(p_documento ->> 'valida_hasta') || '}',
      'UTF8'
    )
$funcion$;

CREATE OR REPLACE FUNCTION vec_autorizacion.motivo_contexto_actor_v3_canonico(
    p_motivo jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
      '{"esquema":' || vec_autorizacion.texto_json_go_v3(p_motivo ->> 'esquema') ||
      ',"referencia":{"catalogo_id":' ||
        vec_autorizacion.texto_json_go_v3(p_motivo -> 'referencia' ->> 'catalogo_id') ||
      ',"catalogo_version":' || (p_motivo -> 'referencia' -> 'catalogo_version')::text ||
      ',"catalogo_huella_sha256":' ||
        vec_autorizacion.texto_json_go_v3(p_motivo -> 'referencia' ->> 'catalogo_huella_sha256') ||
      ',"entrada_clave":' ||
        vec_autorizacion.texto_json_go_v3(p_motivo -> 'referencia' ->> 'entrada_clave') || '}}',
      'UTF8'
    )
$funcion$;

DO $restricciones_canonicas$
DECLARE
    tabla regclass;
    nombre text;
BEGIN
    FOREACH nombre IN ARRAY ARRAY[
        'decision_concedida_contexto_actor_v3',
        'decision_denegada_contexto_actor_v3'
    ] LOOP
        tabla := pg_catalog.to_regclass('vec_autorizacion.' || nombre);
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_constraint
             WHERE conrelid = tabla AND conname = 'decision_v3_bytes_canonicos'
        ) THEN
            EXECUTE pg_catalog.format(
              'ALTER TABLE %s ADD CONSTRAINT decision_v3_bytes_canonicos CHECK (decision_canonica = vec_autorizacion.decision_contexto_actor_v3_canonica(documento) AND motivo_canonico = vec_autorizacion.motivo_contexto_actor_v3_canonico(pg_catalog.convert_from(motivo_canonico, ''UTF8'')::jsonb) AND pg_catalog.encode(pg_catalog.sha256(motivo_canonico), ''hex'') = documento ->> ''motivo_huella_sha256'')',
              tabla
            );
        END IF;
    END LOOP;
END
$restricciones_canonicas$;

CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3(
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric
)
RETURNS TABLE (
    concedida boolean,
    codigo text,
    decision_huella_sha256 text,
    registrada_en timestamptz
)
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
    referencias jsonb;
    instante timestamptz;
    primera_acreditacion timestamptz;
    huella text;
    coincidencias integer;
    replay record;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'registro V3 requiere SERIALIZABLE de escritura';
    END IF;
    -- Toda invocacion participa en la barrera de retirada. El down toma este
    -- mismo advisory en modo exclusivo antes de comprobar o alterar objetos.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
        )
    );
    IF p_decision_canonica IS NULL OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR pg_catalog.octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
       OR p_persona_version IS NULL OR pg_catalog.scale(p_persona_version) <> 0
       OR p_persona_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_perfil_version IS NULL OR pg_catalog.scale(p_perfil_version) <> 0
       OR p_perfil_version NOT BETWEEN 1 AND 18446744073709551615::numeric THEN
        RETURN;
    END IF;
    BEGIN
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
        motivo := pg_catalog.convert_from(p_motivo_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception THEN
            RETURN;
    END;
    IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
       OR vec_autorizacion.decision_contexto_actor_v3_canonica(d)
          IS DISTINCT FROM p_decision_canonica
       OR pg_catalog.jsonb_typeof(motivo) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(motivo)) <> 2
       OR NOT (motivo ?& ARRAY['esquema', 'referencia'])
       OR motivo ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR pg_catalog.jsonb_typeof(motivo -> 'referencia') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(motivo -> 'referencia')) <> 4
       OR pg_catalog.encode(pg_catalog.sha256(p_motivo_canonico), 'hex')
          IS DISTINCT FROM d ->> 'motivo_huella_sha256' THEN
        RETURN;
    END IF;
    rm := motivo -> 'referencia';
    IF NOT (rm ?& ARRAY[
           'catalogo_id', 'catalogo_version',
           'catalogo_huella_sha256', 'entrada_clave'
       ])
       OR pg_catalog.jsonb_typeof(rm -> 'catalogo_id') <> 'string'
       OR pg_catalog.jsonb_typeof(rm -> 'catalogo_version') <> 'number'
       OR pg_catalog.jsonb_typeof(rm -> 'catalogo_huella_sha256') <> 'string'
       OR pg_catalog.jsonb_typeof(rm -> 'entrada_clave') <> 'string'
       OR rm ->> 'catalogo_id' !~ '^[a-z][a-z0-9._-]{0,127}$'
       OR rm ->> 'catalogo_version' !~ '^[1-9][0-9]{0,9}$'
       OR (rm ->> 'catalogo_version')::numeric NOT BETWEEN 1 AND 2147483647
       OR rm ->> 'catalogo_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR rm ->> 'catalogo_huella_sha256' = pg_catalog.repeat('0', 64)
       OR rm ->> 'entrada_clave' !~ '^motivo_[0-9a-f]{32}$'
       OR vec_autorizacion.motivo_contexto_actor_v3_canonico(motivo)
          IS DISTINCT FROM p_motivo_canonico THEN
        RETURN;
    END IF;
    v := d -> 'vinculo_autenticacion_actor';
    huella := pg_catalog.encode(pg_catalog.sha256(p_decision_canonica), 'hex');

    -- Reconciliacion de un COMMIT incierto: una evidencia ya durable se
    -- devuelve incluso si despues caduco o fue revocada. No crea una capacidad
    -- nueva; reproduce el mismo resultado sellado y exige tambien las dos
    -- versiones que no viajan en el vinculo minimizado.
    SELECT pg_catalog.count(*) INTO coincidencias
      FROM (
        SELECT decision_ref, huella_decision_sha256
          FROM vec_autorizacion.decision_concedida_contexto_actor_v3
        UNION ALL
        SELECT decision_ref, huella_decision_sha256
          FROM vec_autorizacion.decision_denegada_contexto_actor_v3
      ) AS existente
     WHERE existente.decision_ref = d ->> 'decision_ref'
        OR existente.huella_decision_sha256 = huella;
    IF coincidencias > 0 THEN
        SELECT existente.concedida, existente.codigo, existente.registrada_en
          INTO replay
          FROM (
            SELECT true AS concedida, 'concedida'::text AS codigo,
                   c.decision_ref, c.huella_decision_sha256,
                   c.decision_canonica, c.motivo_canonico,
                   c.persona_version, c.perfil_version, c.registrada_en
              FROM vec_autorizacion.decision_concedida_contexto_actor_v3 AS c
            UNION ALL
            SELECT false, n.codigo, n.decision_ref,
                   n.huella_decision_sha256, n.decision_canonica,
                   n.motivo_canonico, n.persona_version,
                   n.perfil_version, n.registrada_en
              FROM vec_autorizacion.decision_denegada_contexto_actor_v3 AS n
          ) AS existente
         WHERE existente.decision_ref = d ->> 'decision_ref'
           AND existente.huella_decision_sha256 = huella
           AND existente.decision_canonica = p_decision_canonica
           AND existente.motivo_canonico = p_motivo_canonico
           AND existente.persona_version = p_persona_version
           AND existente.perfil_version = p_perfil_version;
        IF NOT FOUND OR coincidencias <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'colision de identidad de decision V3';
        END IF;
        RETURN QUERY SELECT replay.concedida, replay.codigo, huella,
                            replay.registrada_en;
        RETURN;
    END IF;

    -- Primera acreditacion: antes de cualquier lock propio de autorizacion.
    primera_acreditacion :=
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
    IF primera_acreditacion IS NULL THEN RETURN; END IF;

    -- Serializa replay/colision y retirada de la migracion tras acreditar.
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_autorizacion:decision-v3:' || (d ->> 'decision_ref'), 0
        )
    );
    LOCK TABLE vec_autorizacion.decision_concedida_contexto_actor_v3,
               vec_autorizacion.decision_denegada_contexto_actor_v3
        IN ROW EXCLUSIVE MODE;

    -- Orden propio unico: motivo, asignacion, rol, catalogo/politicas, sesion.
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR SHARE;
    IF NOT FOUND THEN RETURN; END IF;

    SELECT a.asignacion_ref, a.principal_id, a.version_rol_ref,
           a.huella_sha256, a.documento
      INTO asignacion
      FROM vec_autorizacion.asignacion_perfil_actual AS actual
      JOIN vec_autorizacion.asignacion_perfil AS a
        ON a.perfil_activo_ref = actual.perfil_activo_ref
       AND a.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = d ->> 'perfil_activo_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion.asignacion_ref IS DISTINCT FROM d ->> 'asignacion_ref'
       OR asignacion.principal_id IS DISTINCT FROM d ->> 'principal_id'
       OR asignacion.version_rol_ref IS DISTINCT FROM d ->> 'version_rol_ref'
       OR asignacion.huella_sha256 IS DISTINCT FROM
          d ->> 'asignacion_huella_sha256' THEN
        RETURN;
    END IF;

    SELECT r.huella_sha256, r.publicada_en, r.documento,
           c.version_rol_ref, c.revision, c.estado,
           c.huella_sha256 AS control_huella
      INTO rol
      FROM vec_autorizacion.version_rol AS r
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = r.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS c
        ON c.version_rol_ref = actual.version_rol_ref
       AND c.revision = actual.revision
     WHERE r.version_rol_ref = asignacion.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND OR rol.huella_sha256 IS DISTINCT FROM
          d ->> 'version_rol_huella_sha256'
       OR rol.version_rol_ref IS DISTINCT FROM
          d ->> 'control_vigencia_version_rol_ref'
       OR rol.revision IS DISTINCT FROM
          (d ->> 'control_vigencia_version_rol_revision')::numeric
       OR rol.control_huella IS DISTINCT FROM
          d ->> 'control_vigencia_version_rol_huella_sha256' THEN
        RETURN;
    END IF;

    SELECT revision, huella_sha256
      INTO catalogo
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND OR catalogo.revision IS DISTINCT FROM
          (d ->> 'revision_catalogo_politicas')::numeric
       OR catalogo.huella_sha256 IS DISTINCT FROM
          d ->> 'catalogo_politicas_huella_sha256' THEN
        RETURN;
    END IF;
    SELECT coalesce(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_object(
                   'referencia', p.politica_ref,
                   'huella_sha256', p.huella_sha256
               ) ORDER BY p.politica_ref COLLATE "C"
           ), '[]'::jsonb),
           coalesce(pg_catalog.jsonb_agg(
               p.politica_ref ORDER BY p.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO manifiesto, referencias
      FROM vec_autorizacion.politica_restrictiva_actual AS actual
      JOIN vec_autorizacion.politica_restrictiva AS p
        ON p.politica_id = actual.politica_id
       AND p.politica_ref = actual.politica_ref;
    IF manifiesto IS DISTINCT FROM d -> 'politicas_evaluadas' THEN
        RETURN;
    END IF;

    IF vec_autorizacion.revalidar_sesion_vinculo_v2(
           v, (d ->> 'emitida_en')::timestamptz,
           (d ->> 'valida_hasta')::timestamptz,
           (d ->> 'emitida_en')::timestamptz
       ) IS NOT TRUE THEN
        RETURN;
    END IF;

    -- Segunda acreditacion: mismos 17 valores, despues de todos los locks.
    instante :=
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
    IF instante IS NULL OR instante < primera_acreditacion
       OR vec_autorizacion.revalidar_sesion_vinculo_v2(
              v, (d ->> 'emitida_en')::timestamptz,
              (d ->> 'valida_hasta')::timestamptz, instante
          ) IS NOT TRUE THEN
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_v2_catalogo_publicado AS mc
          JOIN vec_autorizacion.motivo_v2_entrada AS me
            ON me.catalogo_id = mc.catalogo_id
           AND me.catalogo_version = mc.catalogo_version
         WHERE mc.catalogo_id = rm ->> 'catalogo_id'
           AND mc.catalogo_version = (rm ->> 'catalogo_version')::integer
           AND mc.catalogo_huella_publicada_sha256 =
               rm ->> 'catalogo_huella_sha256'
           AND me.entrada_clave = rm ->> 'entrada_clave'
           AND mc.publicado_en <= instante
           AND me.vigente_desde <= instante
           AND (me.vigente_hasta IS NULL OR instante < me.vigente_hasta)
           AND NOT EXISTS (
               SELECT 1 FROM vec_autorizacion.motivo_v2_retirada AS mr
                WHERE mr.catalogo_id = mc.catalogo_id
                  AND mr.catalogo_version = mc.catalogo_version
           )
    ) THEN RETURN; END IF;

    IF instante < (d ->> 'emitida_en')::timestamptz
       OR instante >= (d ->> 'valida_hasta')::timestamptz THEN
        RETURN;
    END IF;

    -- Para concesion, SQL vuelve a exigir estado/vigencia ejecutables. Una
    -- denegacion conserva la instantanea exacta, pero nunca recibe cda_.
    IF (d ->> 'concedida')::boolean AND (
        asignacion.documento ->> 'estado' IS DISTINCT FROM 'activa'
        OR instante < (asignacion.documento ->> 'vigente_desde')::timestamptz
        OR instante >= (asignacion.documento ->> 'vigente_hasta')::timestamptz
        OR rol.documento ->> 'estado' IS DISTINCT FROM 'publicada'
        OR rol.publicada_en > instante
        OR rol.estado IS DISTINCT FROM 'habilitada'
    ) THEN RETURN; END IF;

    -- Replay exacto. Colision de referencia o huella no se degrada a exito.
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_concedida_contexto_actor_v3 x
         WHERE x.decision_ref = d ->> 'decision_ref'
            OR x.huella_decision_sha256 = huella
    ) THEN
        SELECT x.registrada_en
          INTO STRICT instante
          FROM vec_autorizacion.decision_concedida_contexto_actor_v3 x
         WHERE x.decision_ref = d ->> 'decision_ref'
           AND x.huella_decision_sha256 = huella
           AND x.decision_canonica = p_decision_canonica
           AND x.motivo_canonico = p_motivo_canonico
           AND x.persona_version = p_persona_version
           AND x.perfil_version = p_perfil_version;
        RETURN QUERY SELECT true, 'concedida'::text, huella, instante;
        RETURN;
    ELSIF EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_denegada_contexto_actor_v3 x
         WHERE x.decision_ref = d ->> 'decision_ref'
            OR x.huella_decision_sha256 = huella
    ) THEN
        SELECT x.codigo, x.registrada_en
          INTO STRICT codigo, instante
          FROM vec_autorizacion.decision_denegada_contexto_actor_v3 x
         WHERE x.decision_ref = d ->> 'decision_ref'
           AND x.huella_decision_sha256 = huella
           AND x.decision_canonica = p_decision_canonica
           AND x.motivo_canonico = p_motivo_canonico
           AND x.persona_version = p_persona_version
           AND x.perfil_version = p_perfil_version;
        RETURN QUERY SELECT false, codigo, huella, instante;
        RETURN;
    END IF;

    IF (d ->> 'concedida')::boolean THEN
        INSERT INTO vec_autorizacion.decision_concedida_contexto_actor_v3(
            decision_ref, huella_decision_sha256, decision_canonica,
            documento, motivo_canonico, motivo_catalogo_id,
            motivo_catalogo_version, motivo_entrada_clave,
            registro_contexto_ref, contexto_actor_huella_sha256,
            manifiesto_procedencia_huella_sha256, persona_version,
            perfil_version, asignacion_ref,
            version_rol_ref, control_vigencia_version_rol_revision,
            revision_catalogo_politicas, emitida_en,
            valida_hasta, registrada_en
        ) VALUES (
            d ->> 'decision_ref', huella, p_decision_canonica, d,
            p_motivo_canonico, rm ->> 'catalogo_id',
            (rm ->> 'catalogo_version')::integer,
            rm ->> 'entrada_clave', v ->> 'registro_contexto_ref',
            v ->> 'contexto_actor_huella_sha256',
            v ->> 'manifiesto_procedencia_huella_sha256',
            p_persona_version, p_perfil_version,
            d ->> 'asignacion_ref', d ->> 'version_rol_ref',
            (d ->> 'control_vigencia_version_rol_revision')::numeric,
            (d ->> 'revision_catalogo_politicas')::numeric,
            (d ->> 'emitida_en')::timestamptz,
            (d ->> 'valida_hasta')::timestamptz, instante
        );
        RETURN QUERY SELECT true, 'concedida'::text, huella, instante;
    ELSE
        INSERT INTO vec_autorizacion.decision_denegada_contexto_actor_v3(
            decision_ref, huella_decision_sha256, decision_canonica,
            documento, motivo_canonico, motivo_catalogo_id,
            motivo_catalogo_version, motivo_entrada_clave,
            registro_contexto_ref, contexto_actor_huella_sha256,
            manifiesto_procedencia_huella_sha256, persona_version,
            perfil_version, asignacion_ref,
            version_rol_ref, control_vigencia_version_rol_revision,
            revision_catalogo_politicas, codigo, emitida_en, valida_hasta,
            registrada_en
        ) VALUES (
            d ->> 'decision_ref', huella, p_decision_canonica, d,
            p_motivo_canonico, rm ->> 'catalogo_id',
            (rm ->> 'catalogo_version')::integer,
            rm ->> 'entrada_clave', v ->> 'registro_contexto_ref',
            v ->> 'contexto_actor_huella_sha256',
            v ->> 'manifiesto_procedencia_huella_sha256',
            p_persona_version, p_perfil_version,
            d ->> 'asignacion_ref', d ->> 'version_rol_ref',
            (d ->> 'control_vigencia_version_rol_revision')::numeric,
            (d ->> 'revision_catalogo_politicas')::numeric,
            d ->> 'codigo', (d ->> 'emitida_en')::timestamptz,
            (d ->> 'valida_hasta')::timestamptz, instante
        );
        RETURN QUERY SELECT false, d ->> 'codigo', huella, instante;
    END IF;
END
$funcion$;

REVOKE ALL ON TABLE
    vec_autorizacion.decision_concedida_contexto_actor_v3,
    vec_autorizacion.decision_denegada_contexto_actor_v3
    FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;
REVOKE ALL ON FUNCTION
    vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb),
    vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb),
    vec_autorizacion.vinculo_contexto_actor_v2_valido(jsonb)
    FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador;
REVOKE ALL ON FUNCTION
    vec_autorizacion.decision_contexto_actor_v3_valida(jsonb)
    FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente;
REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_sesion_vinculo_v2(
        jsonb, timestamptz, timestamptz, timestamptz
    ) FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente;
REVOKE ALL ON FUNCTION
    vec_autorizacion.registrar_decision_contexto_actor_v3(
        bytea, bytea, numeric, numeric
    ) FROM PUBLIC, vec_autorizacion_fuente,
           vec_autorizacion_motivos_proyector,
           vec_autorizacion_motivos_evaluador;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.registrar_decision_contexto_actor_v3(
        bytea, bytea, numeric, numeric
    ) TO vec_autorizacion_registro;

COMMIT;
