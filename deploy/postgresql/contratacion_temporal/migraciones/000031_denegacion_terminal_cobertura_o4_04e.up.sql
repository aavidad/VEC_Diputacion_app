-- O4-04E/8: rama terminal VEC denegada.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=10 FOR UPDATE;
DO $prevalidacion$
BEGIN
  IF NOT EXISTS(
    SELECT 1 FROM vec_contratacion_temporal
      .control_migracion_cobertura_o4
     WHERE control AND version_esquema=10
  ) OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.o404e_confirmar_denegacion_v1(jsonb,timestamp with time zone)'
  ) IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estado incompatible para rama denegada O4-04E';
  END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_confirmar_denegacion_v1(
  p_carga jsonb,p_confirmada_en timestamptz
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
STRICT
SET search_path=pg_catalog
AS $funcion$
DECLARE
  c jsonb:=p_carga->'cabecera';
  d jsonb:=p_carga->'denegacion';
  x jsonb:=p_carga->'decision_vec';
  v_vec record;
  v_vec_observada_en timestamptz;
  v_resultado jsonb;
BEGIN
  IF p_carga->>'rama'<>'denegada'
     OR p_carga->'gobierno'<>'null'::jsonb
     OR p_carga->'concesion'<>'null'::jsonb
     OR p_carga->'consumos_c1'<>'[]'::jsonb
     OR pg_catalog.encode(pg_catalog.sha256(
          vec_contratacion_temporal
            .o404e_material_prueba_denegacion_v1(p_carga)
        ),'hex') IS DISTINCT FROM d->>'prueba_huella_sha256'
     OR d->>'organizacion_ref' IS DISTINCT FROM c->>'organizacion_ref'
     OR d->>'expediente_ref' IS DISTINCT FROM c->>'expediente_ref'
     OR d->>'version_expediente' IS DISTINCT FROM c->>'version_expediente'
     OR d->>'reserva_ref' IS DISTINCT FROM c->>'reserva_ref'
     OR d->>'recibo_ref' IS DISTINCT FROM c->>'recibo_ref'
     OR d->>'auditoria_ref' IS DISTINCT FROM c->>'auditoria_ref'
     OR d->>'correlacion_vec_ref' IS DISTINCT FROM
        c->>'correlacion_vec_ref'
     OR d->>'decision_vec_ref' IS DISTINCT FROM c->>'decision_vec_ref'
     OR d->>'revision_cercado' IS DISTINCT FROM c->>'revision_cercado'
     OR d->>'recurso_ref' IS DISTINCT FROM c->>'reserva_ref'
     OR d->>'recurso_ref' IS DISTINCT FROM x->>'recurso_ref'
     OR d->>'recurso_modulo' IS DISTINCT FROM x->>'recurso_modulo'
     OR d->>'recurso_tipo' IS DISTINCT FROM x->>'recurso_tipo'
     OR d->'ambitos' IS DISTINCT FROM x->'ambitos'
     OR d->'atributos' IS DISTINCT FROM x->'atributos'
     OR d->>'actor_ref' IS DISTINCT FROM x->>'principal_id'
     OR d->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
     OR d->>'accion_vec' IS DISTINCT FROM x->>'accion'
     OR d->>'finalidad_vec' IS DISTINCT FROM x->>'finalidad'
     OR d->>'recurso_huella_sha256' IS DISTINCT FROM
        x->>'contexto_recurso_huella_sha256'
     OR pg_catalog.encode(pg_catalog.sha256(
          vec_contratacion_temporal
            .o404e_contexto_recurso_denegacion_v1(p_carga)),'hex')
        IS DISTINCT FROM d->>'recurso_huella_sha256'
     OR vec_contratacion_temporal.o404e_motivo_denegacion_canon_v1(
          d->'motivo_vec') IS DISTINCT FROM
        pg_catalog.decode(x->>'motivo_canonico_hex','hex')
     OR (d->>'limite_preparacion')::timestamptz<=
        (c->>'observada_en_db')::timestamptz
     OR (d->>'limite_preparacion')::timestamptz>
        (c->>'propiedad_hasta')::timestamptz
     OR (d->>'valida_hasta')::timestamptz>
        (d->>'limite_preparacion')::timestamptz
     OR (d->>'valida_hasta')::timestamptz<>
        (c->>'valida_hasta_orden')::timestamptz
     OR p_confirmada_en>=(d->>'valida_hasta')::timestamptz
     OR p_confirmada_en>=(x->>'valida_hasta')::timestamptz THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='denegación O4-04E divergente';
  END IF;
  SELECT * INTO v_vec
    FROM vec_autorizacion
      .registrar_decision_cobertura_contexto_exacto_o404e_v1(
       pg_catalog.decode(x->>'decision_canonica_hex','hex'),
       pg_catalog.decode(x->>'motivo_canonico_hex','hex'),
       (x->>'persona_version')::numeric,
       (x->>'perfil_version')::numeric,
       pg_catalog.jsonb_build_object(
        'rama','denegada','accion',x->>'accion',
        'organizacion_ref',c->>'organizacion_ref',
        'expediente_ref',c->>'expediente_ref',
        'version_expediente',(c->>'version_expediente')::numeric,
        'reserva_ref',c->>'reserva_ref','recurso_ref',x->>'recurso_ref',
        'decision_ref',c->>'decision_vec_ref',
        'correlacion_ref',c->>'correlacion_vec_ref',
        'finalidad',x->>'finalidad',
        'contexto_recurso_huella_sha256',
          x->>'contexto_recurso_huella_sha256',
        'recurso_modulo',x->>'recurso_modulo',
        'recurso_tipo',x->>'recurso_tipo',
       'ambitos',x->'ambitos','atributos',x->'atributos',
       'huella_orden_sha256',c->>'huella_orden_sha256'));
  v_vec_observada_en:=pg_catalog.date_trunc(
    'microseconds',pg_catalog.clock_timestamp());
  IF NOT FOUND OR v_vec.rama IS DISTINCT FROM 'denegada'
     OR v_vec.concedida IS DISTINCT FROM false
     OR v_vec.codigo IS DISTINCT FROM x->>'codigo_probatorio'
     OR v_vec.decision_ref IS DISTINCT FROM c->>'decision_vec_ref'
     OR v_vec.decision_ref IS DISTINCT FROM x->>'decision_ref'
     OR v_vec.correlacion_ref IS DISTINCT FROM c->>'correlacion_vec_ref'
     OR v_vec.decision_huella_sha256 IS DISTINCT FROM
        x->>'decision_huella_sha256'
     OR v_vec.contexto_recurso_huella_sha256 IS DISTINCT FROM
        x->>'contexto_recurso_huella_sha256'
     OR v_vec.huella_orden_sha256 IS DISTINCT FROM c->>'huella_orden_sha256'
     OR v_vec.lote_huella_sha256 IS NOT NULL
     OR v_vec.revalidada_en IS NOT NULL
     OR v_vec.registrada_en IS NULL
     OR v_vec.registrada_en<(x->>'emitida_en')::timestamptz
     OR v_vec.registrada_en>=(x->>'valida_hasta')::timestamptz
     OR v_vec.registrada_en>v_vec_observada_en THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='resultado VEC denegado O4-04E divergente';
  END IF;
  p_confirmada_en:=pg_catalog.date_trunc(
    'microseconds',pg_catalog.clock_timestamp());
  IF p_confirmada_en>=(c->>'propiedad_hasta')::timestamptz
     OR p_confirmada_en>=(c->>'valida_hasta_orden')::timestamptz
     OR p_confirmada_en>=(d->>'valida_hasta')::timestamptz
     OR p_confirmada_en>=(x->>'valida_hasta')::timestamptz THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='vigencia final denegada O4-04E agotada';
  END IF;
  v_resultado:=pg_catalog.to_jsonb(v_vec);
  RETURN vec_contratacion_temporal.o404e_cerrar_terminal_v1(
    p_carga,v_resultado,pg_catalog.jsonb_build_object(
      'gobierno_ref',NULL,'gobierno_huella_sha256',NULL,
      'lote_ref',NULL,'lote_huella_sha256',NULL,
      'decision_ref',NULL,'decision_huella_sha256',NULL,
      'version_resultante',NULL),p_confirmada_en);
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema=11,
       actualizada_en=pg_catalog.date_trunc(
         'microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=10;
REVOKE ALL ON FUNCTION
  vec_contratacion_temporal.o404e_confirmar_denegacion_v1(
    jsonb,timestamptz)
FROM PUBLIC,vec_contratacion_temporal_ejecutor,
  vec_contratacion_temporal_confirmador_cobertura,
  vec_contratacion_temporal_migrador,
  vec_contratacion_temporal_gobernador;
COMMIT;
