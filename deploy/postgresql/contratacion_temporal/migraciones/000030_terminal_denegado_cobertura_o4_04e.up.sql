-- O4-04E/7: cierre terminal común y rama VEC denegada.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 9
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 9
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_cerrar_terminal_v1(jsonb,jsonb,jsonb,timestamp with time zone)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para cierre O4-04E';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.o404e_iguales_constante_v1(
    p_izquierda text,
    p_derecha text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_i integer;
    v_diferencia integer :=
        pg_catalog.octet_length(p_izquierda)
        # pg_catalog.octet_length(p_derecha);
    v_izquierda bytea := pg_catalog.convert_to(p_izquierda, 'UTF8');
    v_derecha bytea := pg_catalog.convert_to(p_derecha, 'UTF8');
    v_maximo integer := greatest(
        pg_catalog.octet_length(v_izquierda),
        pg_catalog.octet_length(v_derecha)
    );
BEGIN
    IF v_maximo > 1024 THEN
        RETURN false;
    END IF;
    FOR v_i IN 0..v_maximo - 1 LOOP
        v_diferencia := v_diferencia |
            (CASE WHEN v_i < pg_catalog.octet_length(v_izquierda)
                  THEN pg_catalog.get_byte(v_izquierda, v_i) ELSE 0 END #
             CASE WHEN v_i < pg_catalog.octet_length(v_derecha)
                  THEN pg_catalog.get_byte(v_derecha, v_i) ELSE 0 END);
    END LOOP;
    RETURN v_diferencia = 0;
END
$funcion$;

-- Proyección terminal cerrada. Devuelve NULL ante cualquier unión incompleta.
CREATE FUNCTION
vec_contratacion_temporal.o404e_leer_terminal_interno_v1(
    p_ambito text,
    p_huella_orden text
)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v record;
    v_recibo jsonb;
    v_evento jsonb;
    v_positivos boolean;
    v_auditoria bytea;
    v_campo text;
BEGIN
    SELECT b.*, a.secuencia AS secuencia_actual, rv.estado,
           c.rama, c.huella_orden_sha256,
           c.decision_vec_huella_sha256,
           c.codigo_probatorio_vec, c.revision_cercado,
           c.ambito_idempotencia_hmac, c.huella_semantica_hmac,
           c.confirmada_en, c.decision_cobertura_ref,
           c.decision_cobertura_huella_sha256, c.version_resultante,
           c.evento_ref AS evento_confirmado,
           c.actuacion_ref AS actuacion_confirmada,
           c.recibo_huella_sha256, c.carga_huella_sha256,
           t.outbox_ref, t.gobierno_ref,
           t.gobierno_huella_sha256, t.consumo_c1_lote_ref,
           t.consumo_c1_lote_huella_sha256,
           o.tipo_evento AS outbox_tipo,
           o.payload_canonico AS outbox_payload,
           o.version_expediente AS outbox_version,
           o.registrada_en AS outbox_registrada_en,
           au.prueba_canonica AS auditoria_prueba_canonica,
           au.anterior_sha256 AS auditoria_anterior,
           au.huella_sha256 AS auditoria_huella,
           pd.prueba_canonica AS denegacion_prueba_canonica,
           pd.prueba_huella_sha256 AS denegacion_prueba_huella,
           pd.contexto_recurso_canonico AS denegacion_contexto_canonico,
           pd.contexto_recurso_huella_sha256 AS denegacion_contexto_huella,
           pd.motivo_canonico AS denegacion_motivo_canonico,
           pd.motivo_huella_sha256 AS denegacion_motivo_huella,
           pd.limite_preparacion AS denegacion_limite,
           pd.registrada_en AS denegacion_registrada
      INTO v
      FROM vec_contratacion_temporal
           .reserva_operacion_decision_cobertura b
      JOIN vec_contratacion_temporal
           .reserva_operacion_decision_cobertura_actual a
        USING (ambito_raiz_hmac)
      JOIN vec_contratacion_temporal
           .reserva_operacion_decision_cobertura_version rv
        ON rv.ambito_raiz_hmac = a.ambito_raiz_hmac
       AND rv.secuencia = a.secuencia
      JOIN vec_contratacion_temporal
           .confirmacion_operacion_decision_cobertura c
        ON c.ambito_raiz_hmac = b.ambito_raiz_hmac
      JOIN vec_contratacion_temporal
           .terminal_operacion_decision_cobertura t
        ON t.ambito_raiz_hmac = c.ambito_raiz_hmac
       AND t.secuencia_terminal = rv.secuencia
       AND t.recibo_ref = c.recibo_ref
       AND t.huella_orden_sha256 = c.huella_orden_sha256
       AND t.rama = c.rama
       AND t.decision_vec_ref = c.decision_vec_ref
       AND t.decision_vec_huella_sha256 =
           c.decision_vec_huella_sha256
       AND t.codigo_probatorio_vec = c.codigo_probatorio_vec
       AND t.auditoria_ref = c.auditoria_ref
       AND t.decision_cobertura_ref IS NOT DISTINCT FROM
           c.decision_cobertura_ref
       AND t.decision_cobertura_huella_sha256 IS NOT DISTINCT FROM
           c.decision_cobertura_huella_sha256
       AND t.version_resultante IS NOT DISTINCT FROM c.version_resultante
       AND t.actuacion_ref IS NOT DISTINCT FROM c.actuacion_ref
      JOIN vec_contratacion_temporal.auditoria_decision_cobertura au
        ON au.auditoria_ref = c.auditoria_ref
       AND au.rama = c.rama
       AND au.reserva_ref = b.reserva_ref
       AND au.recibo_ref = b.recibo_ref
       AND au.huella_orden_sha256 = c.huella_orden_sha256
       AND au.decision_vec_ref = c.decision_vec_ref
       AND au.decision_vec_huella_sha256 =
           c.decision_vec_huella_sha256
       AND au.codigo_probatorio_vec = c.codigo_probatorio_vec
       AND au.acreditacion_gobierno_ref IS NOT DISTINCT FROM
           t.gobierno_ref
       AND au.gobierno_huella_sha256 IS NOT DISTINCT FROM
           t.gobierno_huella_sha256
       AND au.consumo_c1_lote_ref IS NOT DISTINCT FROM
           t.consumo_c1_lote_ref
       AND au.consumo_c1_lote_huella_sha256 IS NOT DISTINCT FROM
           t.consumo_c1_lote_huella_sha256
       AND au.decision_cobertura_ref IS NOT DISTINCT FROM
           t.decision_cobertura_ref
       AND au.decision_cobertura_huella_sha256 IS NOT DISTINCT FROM
           t.decision_cobertura_huella_sha256
       AND au.version_resultante IS NOT DISTINCT FROM
           t.version_resultante
       AND au.actuacion_ref IS NOT DISTINCT FROM t.actuacion_ref
      JOIN vec_contratacion_temporal.outbox_expediente_integral o
       ON o.evento_ref = t.outbox_ref
       AND o.operacion_ref = b.actuacion_ref
       AND o.expediente_ref = b.expediente_ref
      LEFT JOIN vec_contratacion_temporal
           .prueba_denegacion_decision_cobertura pd
        ON pd.ambito_raiz_hmac=c.ambito_raiz_hmac
       AND pd.decision_vec_ref=c.decision_vec_ref
     WHERE b.ambito_raiz_hmac = p_ambito
       AND c.huella_orden_sha256 = p_huella_orden
       AND rv.huella_orden_sha256 = c.huella_orden_sha256
       AND rv.revision_cercado = c.revision_cercado
       AND rv.confirmada_en = c.confirmada_en
       AND rv.estado = CASE c.rama
                         WHEN 'concedida' THEN 'aplicada'
                         ELSE 'denegada_vec'
                       END
       AND c.carga_huella_sha256 ~ '^[a-f0-9]{64}$'
       AND c.carga_huella_sha256 <> pg_catalog.repeat('0',64)
       AND ((c.rama='concedida' AND pd.ambito_raiz_hmac IS NULL)
         OR (c.rama='denegada' AND pd.ambito_raiz_hmac IS NOT NULL));
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;
    v_auditoria:=vec_contratacion_temporal.o404e_texto_v1(
      ''::bytea,'VEC-CT-AUDITORIA-DECISION-COBERTURA-O4-04E-V1');
    FOREACH v_campo IN ARRAY ARRAY[
      v.rama,v.auditoria_ref,v.reserva_ref,v.recibo_ref,
      v.huella_orden_sha256,v.organizacion_ref,v.expediente_ref,
      v.version_expediente::text,v.actuacion_ref,v.decision_vec_ref,
      v.decision_vec_huella_sha256,v.codigo_probatorio_vec,
      coalesce(v.gobierno_ref,''),coalesce(v.gobierno_huella_sha256,''),
      coalesce(v.consumo_c1_lote_ref,''),
      coalesce(v.consumo_c1_lote_huella_sha256,''),
      coalesce(v.decision_cobertura_ref,''),
      coalesce(v.decision_cobertura_huella_sha256,''),
      coalesce(v.denegacion_prueba_huella,''),
      coalesce(v.denegacion_contexto_huella,''),
      coalesce(v.denegacion_motivo_huella,'')
    ]::text[] LOOP
      v_auditoria:=vec_contratacion_temporal.o404e_texto_v1(
        v_auditoria,v_campo);
    END LOOP;
    IF v_auditoria IS DISTINCT FROM v.auditoria_prueba_canonica
       OR pg_catalog.encode(pg_catalog.sha256(
            v.auditoria_anterior::bytea||v_auditoria),'hex')
          IS DISTINCT FROM v.auditoria_huella THEN
      RETURN NULL;
    END IF;
    BEGIN
        v_evento := pg_catalog.convert_from(
            v.outbox_payload, 'UTF8'
        )::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RETURN NULL;
    END;
    IF NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           v_evento, ARRAY[
             'auditoria_ref','codigo_probatorio_vec',
             'contexto_recurso_huella_sha256',
             'decision_cobertura_ref','decision_vec_huella_sha256',
             'decision_vec_ref','esquema','motivo_huella_sha256',
             'prueba_denegacion_huella_sha256','rama','recibo_ref',
             'registrada_en','reserva_ref','version_resultante'
           ]::text[]
       )
       OR v_evento->>'esquema' <>
          'vec.contratacion-temporal.evento-decision-cobertura.o4-04e.v1'
       OR v_evento->>'rama' <> v.rama
       OR v_evento->>'reserva_ref' <> v.reserva_ref
       OR v_evento->>'recibo_ref' <> v.recibo_ref
       OR v_evento->>'auditoria_ref' <> v.auditoria_ref
       OR v_evento->>'decision_vec_ref' <> v.decision_vec_ref
       OR v_evento->>'decision_vec_huella_sha256' <>
          v.decision_vec_huella_sha256
       OR v_evento->>'codigo_probatorio_vec' <> v.codigo_probatorio_vec
       OR v_evento->>'prueba_denegacion_huella_sha256' IS DISTINCT FROM
          coalesce(v.denegacion_prueba_huella,'')
       OR v_evento->>'contexto_recurso_huella_sha256' IS DISTINCT FROM
          coalesce(v.denegacion_contexto_huella,'')
       OR v_evento->>'motivo_huella_sha256' IS DISTINCT FROM
          coalesce(v.denegacion_motivo_huella,'')
       OR v_evento->>'decision_cobertura_ref' <>
          coalesce(v.decision_cobertura_ref,'')
       OR (v_evento->>'version_resultante')::numeric <>
          coalesce(v.version_resultante,0)
       OR v.outbox_version <> coalesce(
          v.version_resultante,v.version_expediente)
       OR v.outbox_registrada_en <> v.confirmada_en
       OR v_evento->>'registrada_en' <>
          vec_contratacion_temporal.texto_instante_utc_go_v2(
              v.confirmada_en::text
          )
       OR v.outbox_tipo <> (CASE v.rama
            WHEN 'concedida'
              THEN 'contratacion_temporal.cobertura_aplicada'
            ELSE 'contratacion_temporal.cobertura_denegada_vec'
          END) THEN
        RETURN NULL;
    END IF;

    IF v.rama = 'concedida' THEN
        SELECT
          EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal
                   .acreditacion_gobierno_decision_cobertura g
              JOIN vec_contratacion_temporal.consumo_cobertura_lote l
                ON l.lote_ref = v.consumo_c1_lote_ref
               AND l.lote_huella_sha256 =
                   v.consumo_c1_lote_huella_sha256
              JOIN vec_contratacion_temporal
                   .decision_cobertura_gobernada_durable d
                ON d.decision_ref = v.decision_cobertura_ref
               AND d.decision_huella_sha256 =
                   v.decision_cobertura_huella_sha256
               AND d.acreditacion_gobierno_ref = g.acreditacion_ref
               AND d.consumo_c1_lote_ref = l.lote_ref
              JOIN vec_contratacion_temporal
                   .expediente_version_integral e
                ON e.expediente_ref = v.expediente_ref
               AND e.version = v.version_resultante
              JOIN vec_contratacion_temporal
                   .actuacion_expediente_integral x
                ON x.operacion_ref = v.actuacion_confirmada
               AND x.expediente_ref = e.expediente_ref
               AND x.version_expediente = e.version
             WHERE g.acreditacion_ref = v.gobierno_ref
               AND g.gobierno_huella_sha256 =
                   v.gobierno_huella_sha256
               AND (
                   SELECT pg_catalog.count(*)
                     FROM vec_contratacion_temporal
                          .consumo_cobertura_evidencia ce
                    WHERE ce.lote_ref = l.lote_ref
               ) = l.numero_evidencias
          ) INTO v_positivos;
    ELSE
        SELECT (
          v.denegacion_prueba_huella=
            pg_catalog.encode(pg_catalog.sha256(
              v.denegacion_prueba_canonica),'hex')
          AND v.denegacion_contexto_huella=
            pg_catalog.encode(pg_catalog.sha256(
              v.denegacion_contexto_canonico),'hex')
          AND v.denegacion_motivo_huella=
            pg_catalog.encode(pg_catalog.sha256(
              v.denegacion_motivo_canonico),'hex')
          AND v.denegacion_limite>v.denegacion_registrada
          AND NOT EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal
                   .acreditacion_gobierno_decision_cobertura g
             WHERE g.reserva_ref = v.reserva_ref
            UNION ALL
            SELECT 1
              FROM vec_contratacion_temporal.consumo_cobertura_lote l
             WHERE l.reserva_ref = v.reserva_ref
            UNION ALL
            SELECT 1
              FROM vec_contratacion_temporal
                   .decision_cobertura_gobernada_durable d
             WHERE d.reserva_ref = v.reserva_ref
            UNION ALL
            SELECT 1
             FROM vec_contratacion_temporal
                   .actuacion_expediente_integral x
             WHERE x.operacion_ref = v.actuacion_ref
          )
        ) INTO v_positivos;
    END IF;
    IF v_positivos IS NOT TRUE THEN
        RETURN NULL;
    END IF;

    v_recibo := pg_catalog.jsonb_build_object(
      'esquema',
        'vec.contratacion-temporal.recibo-operacion-decision-cobertura.o4-04e.v1',
      'recibo_ref',v.recibo_ref,'reserva_ref',v.reserva_ref,
      'auditoria_ref',v.auditoria_ref,
      'correlacion_vec_ref',v.correlacion_vec_ref,
      'decision_vec_ref',v.decision_vec_ref,
      'decision_vec_huella_sha256',v.decision_vec_huella_sha256,
      'codigo_probatorio_vec',v.codigo_probatorio_vec,
      'concedida_vec',v.rama = 'concedida',
      'revision_cercado',v.revision_cercado,
      'ambito_idempotencia_hmac',v.ambito_idempotencia_hmac,
      'huella_semantica_hmac',v.huella_semantica_hmac,
      'confirmada_en',
        vec_contratacion_temporal.texto_instante_utc_go_v2(
            v.confirmada_en::text
        ),
      'aplicada',v.rama = 'concedida',
      'denegada_vec',v.rama = 'denegada',
      'decision_cobertura_ref',
        coalesce(v.decision_cobertura_ref, ''),
      'decision_cobertura_huella_sha256',
        coalesce(v.decision_cobertura_huella_sha256, ''),
      'version_resultante',coalesce(v.version_resultante, 0),
      'evento_ref',coalesce(v.evento_confirmado, ''),
      'actuacion_ref',coalesce(v.actuacion_confirmada, '')
    );
    IF pg_catalog.encode(
           pg_catalog.sha256(
             vec_contratacion_temporal.o404e_material_recibo_v1(v_recibo)
           ), 'hex'
       ) <> v.recibo_huella_sha256 THEN
        RETURN NULL;
    END IF;
    RETURN v_recibo;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_cerrar_terminal_v1(
    p_carga jsonb,
    p_vec jsonb,
    p_prueba_rama jsonb,
    p_confirmada_en timestamptz
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    c jsonb := p_carga -> 'cabecera';
    v_rama text := p_carga ->> 'rama';
    v_control record;
    v_prueba bytea;
    v_payload bytea;
    v_auditoria_huella text;
    v_outbox_huella text;
    v_recibo_huella text;
    v_carga_huella text;
    v_recibo jsonb;
    v_secuencia_reserva numeric;
    v_ambito_raiz text;
    v_prueba_denegada bytea;
    v_contexto_denegado bytea;
    v_motivo_denegado bytea;
BEGIN
    v_carga_huella := pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(p_carga::text,'UTF8')
    ),'hex');
    SELECT a.ambito_raiz_hmac INTO STRICT v_ambito_raiz
      FROM vec_contratacion_temporal
           .alias_operacion_decision_cobertura a
     WHERE a.alias_ambito_hmac = c->>'ambito_idempotencia_hmac'
       AND a.alias_huella_semantica_hmac =
           c->>'huella_semantica_hmac';
    SELECT * INTO STRICT v_control
      FROM vec_contratacion_temporal
           .control_cadenas_expediente_integral
     WHERE control_id FOR UPDATE;
    IF p_carga->>'rama'='denegada' THEN
      v_prueba_denegada:=
        vec_contratacion_temporal.o404e_material_prueba_denegacion_v1(
          p_carga);
      v_contexto_denegado:=
        vec_contratacion_temporal.o404e_contexto_recurso_denegacion_v1(
          p_carga);
      v_motivo_denegado:=pg_catalog.decode(
        p_carga#>>'{decision_vec,motivo_canonico_hex}','hex');
      IF v_prueba_denegada IS NULL OR v_contexto_denegado IS NULL
         OR v_motivo_denegado IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='22023',
          MESSAGE='prueba denegada O4-04E no materializable';
      END IF;
      INSERT INTO vec_contratacion_temporal
        .prueba_denegacion_decision_cobertura(
        ambito_raiz_hmac,decision_vec_ref,prueba_canonica,
        prueba_huella_sha256,contexto_recurso_canonico,
        contexto_recurso_huella_sha256,motivo_canonico,
        motivo_huella_sha256,limite_preparacion,registrada_en
      ) VALUES(
        v_ambito_raiz,p_vec->>'decision_ref',v_prueba_denegada,
        p_carga#>>'{denegacion,prueba_huella_sha256}',
        v_contexto_denegado,
        p_carga#>>'{denegacion,recurso_huella_sha256}',
        v_motivo_denegado,
        pg_catalog.encode(pg_catalog.sha256(v_motivo_denegado),'hex'),
        (p_carga#>>'{denegacion,limite_preparacion}')::timestamptz,
        p_confirmada_en
      );
    END IF;
    v_prueba := vec_contratacion_temporal.o404e_texto_v1(
        ''::bytea, 'VEC-CT-AUDITORIA-DECISION-COBERTURA-O4-04E-V1'
    );
    FOREACH v_rama IN ARRAY ARRAY[
      p_carga->>'rama',c->>'auditoria_ref',c->>'reserva_ref',
      c->>'recibo_ref',c->>'huella_orden_sha256',
      c->>'organizacion_ref',c->>'expediente_ref',
      c->>'version_expediente',c->>'actuacion_ref',
      p_vec->>'decision_ref',p_vec->>'decision_huella_sha256',
      p_vec->>'codigo',
      coalesce(p_prueba_rama->>'gobierno_ref',''),
      coalesce(p_prueba_rama->>'gobierno_huella_sha256',''),
      coalesce(p_prueba_rama->>'lote_ref',''),
      coalesce(p_prueba_rama->>'lote_huella_sha256',''),
      coalesce(p_prueba_rama->>'decision_ref',''),
      coalesce(p_prueba_rama->>'decision_huella_sha256',''),
      coalesce(p_carga#>>'{denegacion,prueba_huella_sha256}',''),
      coalesce(p_carga#>>'{denegacion,recurso_huella_sha256}',''),
      CASE WHEN p_carga->>'rama'='denegada'
        THEN pg_catalog.encode(pg_catalog.sha256(v_motivo_denegado),'hex')
        ELSE '' END
    ]::text[] LOOP
        v_prueba := vec_contratacion_temporal.o404e_texto_v1(
            v_prueba, v_rama
        );
    END LOOP;
    v_auditoria_huella := pg_catalog.encode(pg_catalog.sha256(
        v_control.cabeza_auditoria_sha256::bytea || v_prueba
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.auditoria_decision_cobertura (
      auditoria_ref,secuencia,rama,reserva_ref,recibo_ref,
      huella_orden_sha256,organizacion_ref,expediente_ref,
      version_expediente_origen,version_resultante,operacion_ref,accion,
      decision_vec_ref,decision_vec_huella_sha256,codigo_probatorio_vec,
      acreditacion_gobierno_ref,gobierno_huella_sha256,
      consumo_c1_lote_ref,consumo_c1_lote_huella_sha256,
      decision_cobertura_ref,decision_cobertura_huella_sha256,
      actuacion_ref,prueba_canonica,anterior_sha256,huella_sha256,
      registrada_en
    ) VALUES (
      c->>'auditoria_ref',v_control.secuencia_auditoria+1,
      p_carga->>'rama',c->>'reserva_ref',c->>'recibo_ref',
      c->>'huella_orden_sha256',c->>'organizacion_ref',
      c->>'expediente_ref',(c->>'version_expediente')::numeric,
      (p_prueba_rama->>'version_resultante')::numeric,
      c->>'actuacion_ref',
      CASE p_carga->>'rama' WHEN 'concedida'
        THEN p_carga#>>'{gobierno,accion}'
        ELSE p_carga#>>'{denegacion,accion_vec}' END,
      p_vec->>'decision_ref',p_vec->>'decision_huella_sha256',
      p_vec->>'codigo',p_prueba_rama->>'gobierno_ref',
      p_prueba_rama->>'gobierno_huella_sha256',
      p_prueba_rama->>'lote_ref',p_prueba_rama->>'lote_huella_sha256',
      p_prueba_rama->>'decision_ref',
      p_prueba_rama->>'decision_huella_sha256',
      CASE WHEN p_carga->>'rama'='concedida'
           THEN c->>'actuacion_ref' END,
      v_prueba,v_control.cabeza_auditoria_sha256,
      v_auditoria_huella,p_confirmada_en
    );

    v_payload := pg_catalog.convert_to(
      pg_catalog.jsonb_build_object(
        'esquema','vec.contratacion-temporal.evento-decision-cobertura.o4-04e.v1',
        'rama',p_carga->>'rama','reserva_ref',c->>'reserva_ref',
        'recibo_ref',c->>'recibo_ref',
        'auditoria_ref',c->>'auditoria_ref',
        'decision_vec_ref',p_vec->>'decision_ref',
        'decision_vec_huella_sha256',p_vec->>'decision_huella_sha256',
        'codigo_probatorio_vec',p_vec->>'codigo',
        'prueba_denegacion_huella_sha256',
          coalesce(p_carga#>>'{denegacion,prueba_huella_sha256}',''),
        'contexto_recurso_huella_sha256',
          coalesce(p_carga#>>'{denegacion,recurso_huella_sha256}',''),
        'motivo_huella_sha256',
          CASE WHEN p_carga->>'rama'='denegada'
            THEN pg_catalog.encode(
              pg_catalog.sha256(v_motivo_denegado),'hex')
            ELSE '' END,
        'decision_cobertura_ref',
          coalesce(p_prueba_rama->>'decision_ref',''),
        'version_resultante',
          coalesce((p_prueba_rama->>'version_resultante')::numeric,0),
        'registrada_en',
          vec_contratacion_temporal.texto_instante_utc_go_v2(
              p_confirmada_en::text
          )
      )::text, 'UTF8'
    );
    v_outbox_huella := pg_catalog.encode(pg_catalog.sha256(
        v_control.cabeza_outbox_sha256::bytea || v_payload
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
      evento_ref,secuencia,operacion_ref,expediente_ref,
      version_expediente,tipo_evento,payload_canonico,
      payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en
    ) VALUES (
      c->>'evento_ref',v_control.secuencia_outbox+1,c->>'actuacion_ref',
      c->>'expediente_ref',
      coalesce(
        (p_prueba_rama->>'version_resultante')::numeric,
        (c->>'version_expediente')::numeric
      ),
      CASE p_carga->>'rama'
        WHEN 'concedida' THEN 'contratacion_temporal.cobertura_aplicada'
        ELSE 'contratacion_temporal.cobertura_denegada_vec' END,
      v_payload,pg_catalog.encode(pg_catalog.sha256(v_payload),'hex'),
      v_control.cabeza_outbox_sha256,v_outbox_huella,p_confirmada_en
    );
    UPDATE vec_contratacion_temporal
           .control_cadenas_expediente_integral
       SET secuencia_auditoria = v_control.secuencia_auditoria+1,
           cabeza_auditoria_sha256 = v_auditoria_huella,
           secuencia_outbox = v_control.secuencia_outbox+1,
           cabeza_outbox_sha256 = v_outbox_huella,
           actualizada_en = p_confirmada_en
     WHERE control_id
       AND secuencia_auditoria = v_control.secuencia_auditoria
       AND secuencia_outbox = v_control.secuencia_outbox;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE='40001',
            MESSAGE='CAS de cadenas O4-04E perdido';
    END IF;

    v_recibo := pg_catalog.jsonb_build_object(
      'esquema',
        'vec.contratacion-temporal.recibo-operacion-decision-cobertura.o4-04e.v1',
      'recibo_ref',c->>'recibo_ref','reserva_ref',c->>'reserva_ref',
      'auditoria_ref',c->>'auditoria_ref',
      'correlacion_vec_ref',c->>'correlacion_vec_ref',
      'decision_vec_ref',p_vec->>'decision_ref',
      'decision_vec_huella_sha256',p_vec->>'decision_huella_sha256',
      'codigo_probatorio_vec',p_vec->>'codigo',
      'concedida_vec',p_carga->>'rama'='concedida',
      'revision_cercado',(c->>'revision_cercado')::numeric,
      'ambito_idempotencia_hmac',c->>'ambito_idempotencia_hmac',
      'huella_semantica_hmac',c->>'huella_semantica_hmac',
      'confirmada_en',
        vec_contratacion_temporal.texto_instante_utc_go_v2(
            p_confirmada_en::text
        ),
      'aplicada',p_carga->>'rama'='concedida',
      'denegada_vec',p_carga->>'rama'='denegada',
      'decision_cobertura_ref',
        coalesce(p_prueba_rama->>'decision_ref',''),
      'decision_cobertura_huella_sha256',
        coalesce(p_prueba_rama->>'decision_huella_sha256',''),
      'version_resultante',
        coalesce((p_prueba_rama->>'version_resultante')::numeric,0),
      'evento_ref',
        CASE WHEN p_carga->>'rama'='concedida'
             THEN c->>'evento_ref' ELSE '' END,
      'actuacion_ref',
        CASE WHEN p_carga->>'rama'='concedida'
             THEN c->>'actuacion_ref' ELSE '' END
    );
    v_recibo_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.o404e_material_recibo_v1(v_recibo)
    ), 'hex');
    INSERT INTO vec_contratacion_temporal
        .confirmacion_operacion_decision_cobertura (
      ambito_raiz_hmac,recibo_ref,reserva_ref,huella_orden_sha256,rama,
      auditoria_ref,correlacion_vec_ref,decision_vec_ref,
      decision_vec_huella_sha256,codigo_probatorio_vec,revision_cercado,
      ambito_idempotencia_hmac,huella_semantica_hmac,
      decision_cobertura_ref,decision_cobertura_huella_sha256,
      version_resultante,evento_ref,actuacion_ref,confirmada_en,
      recibo_huella_sha256,carga_huella_sha256
    ) VALUES (
      v_ambito_raiz,c->>'recibo_ref',c->>'reserva_ref',
      c->>'huella_orden_sha256',p_carga->>'rama',c->>'auditoria_ref',
      c->>'correlacion_vec_ref',p_vec->>'decision_ref',
      p_vec->>'decision_huella_sha256',p_vec->>'codigo',
      (c->>'revision_cercado')::numeric,c->>'ambito_idempotencia_hmac',
      c->>'huella_semantica_hmac',p_prueba_rama->>'decision_ref',
      p_prueba_rama->>'decision_huella_sha256',
      (p_prueba_rama->>'version_resultante')::numeric,
      CASE WHEN p_carga->>'rama'='concedida' THEN c->>'evento_ref' END,
      CASE WHEN p_carga->>'rama'='concedida' THEN c->>'actuacion_ref' END,
      p_confirmada_en,v_recibo_huella,v_carga_huella
    );
    SELECT a.secuencia + 1 INTO STRICT v_secuencia_reserva
      FROM vec_contratacion_temporal
           .reserva_operacion_decision_cobertura_actual a
     WHERE a.ambito_raiz_hmac = v_ambito_raiz;
    INSERT INTO vec_contratacion_temporal
        .reserva_operacion_decision_cobertura_version (
      ambito_raiz_hmac,secuencia,estado,revision_cercado,
      token_propietario_sha256,observada_en,propiedad_hasta,
      huella_orden_sha256,confirmada_en
    ) VALUES (
      v_ambito_raiz,v_secuencia_reserva,
      CASE p_carga->>'rama'
        WHEN 'concedida' THEN 'aplicada' ELSE 'denegada_vec' END,
      (c->>'revision_cercado')::numeric,c->>'token_propietario_sha256',
      p_confirmada_en,(c->>'propiedad_hasta')::timestamptz,
      c->>'huella_orden_sha256',p_confirmada_en
    );
    INSERT INTO vec_contratacion_temporal
        .terminal_operacion_decision_cobertura (
      ambito_raiz_hmac,secuencia_terminal,recibo_ref,
      huella_orden_sha256,rama,decision_vec_ref,auditoria_ref,outbox_ref,
      gobierno_ref,gobierno_huella_sha256,consumo_c1_lote_ref,
      consumo_c1_lote_huella_sha256,decision_cobertura_ref,
      actuacion_ref,version_resultante,marcada_en,
      decision_vec_huella_sha256,codigo_probatorio_vec,
      decision_cobertura_huella_sha256
    ) VALUES (
      v_ambito_raiz,v_secuencia_reserva,c->>'recibo_ref',
      c->>'huella_orden_sha256',p_carga->>'rama',p_vec->>'decision_ref',
      c->>'auditoria_ref',c->>'evento_ref',
      p_prueba_rama->>'gobierno_ref',
      p_prueba_rama->>'gobierno_huella_sha256',
      p_prueba_rama->>'lote_ref',p_prueba_rama->>'lote_huella_sha256',
      p_prueba_rama->>'decision_ref',
      CASE WHEN p_carga->>'rama'='concedida' THEN c->>'actuacion_ref' END,
      (p_prueba_rama->>'version_resultante')::numeric,p_confirmada_en,
      p_vec->>'decision_huella_sha256',p_vec->>'codigo',
      p_prueba_rama->>'decision_huella_sha256'
    );
    UPDATE vec_contratacion_temporal
           .reserva_operacion_decision_cobertura_actual
       SET secuencia = v_secuencia_reserva
     WHERE ambito_raiz_hmac = v_ambito_raiz
       AND secuencia = v_secuencia_reserva - 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE='40001',
            MESSAGE='CAS de reserva O4-04E perdido';
    END IF;
    RETURN v_recibo;
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 10,
       actualizada_en =
           pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 9;

REVOKE ALL ON FUNCTION
  vec_contratacion_temporal.o404e_iguales_constante_v1(text,text),
  vec_contratacion_temporal.o404e_leer_terminal_interno_v1(text,text),
  vec_contratacion_temporal.o404e_cerrar_terminal_v1(
      jsonb,jsonb,jsonb,timestamptz
  )
FROM PUBLIC,vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_confirmador_cobertura,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;
COMMIT;
