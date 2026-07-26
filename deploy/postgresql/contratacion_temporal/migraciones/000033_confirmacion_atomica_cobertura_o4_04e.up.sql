-- O4-04E/10: concesión y única función exterior atómica.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
  pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_04:migraciones',0
  )
);
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema=12
 FOR UPDATE;
DO $prevalidacion$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema=12
  ) OR pg_catalog.to_regprocedure(
    'vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(jsonb)'
  ) IS NOT NULL THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='estado incompatible para confirmación O4-04E';
  END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_confirmar_concesion_v1(
  p_carga jsonb,
  p_ambito_raiz text,
  p_confirmada_en timestamptz
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
STRICT
SET search_path=pg_catalog
AS $funcion$
DECLARE
  c jsonb:=p_carga->'cabecera';
  g jsonb:=p_carga->'gobierno';
  x jsonb:=p_carga->'decision_vec';
  z jsonb:=p_carga->'concesion';
  v_anterior record;
  v_actual record;
  v_gobierno_actual record;
  v_checkpoint record;
  v_prevalidado record;
  v_vec record;
  v_lote jsonb;
  v_vec_json jsonb;
  v_decision jsonb;
  v_actuacion jsonb;
  v_propuesta jsonb:=z->'propuesta';
  v_version numeric;
  v_agregado_huella text;
  v_prueba bytea;
  v_huella text;
  v_gobierno_ref text;
  v_gobierno_huella text;
BEGIN
  IF p_carga->>'rama'<>'concedida'
     OR p_carga->'denegacion'<>'null'::jsonb
     OR pg_catalog.jsonb_typeof(g)<>'object'
     OR pg_catalog.jsonb_typeof(z)<>'object'
     OR pg_catalog.jsonb_array_length(p_carga->'consumos_c1')
        NOT BETWEEN 1 AND 512
     OR (c->>'numero_consumos_c1')::integer <>
        pg_catalog.jsonb_array_length(p_carga->'consumos_c1')
     OR (z->>'efecto_en')::timestamptz>p_confirmada_en
     OR p_confirmada_en>=(z->>'valida_hasta')::timestamptz
     OR p_confirmada_en>=(x->>'valida_hasta')::timestamptz
     OR vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
          z->'agregado_anterior',z->'agregado_siguiente',
          v_propuesta,z->'motivo_funcional',
          (z->>'efecto_en')::timestamptz
        ) IS NOT TRUE THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='concesión O4-04E inválida';
  END IF;
  SELECT a.* INTO STRICT v_actual
    FROM vec_contratacion_temporal.expediente_integral_actual a
   WHERE a.expediente_ref=c->>'expediente_ref'
   FOR UPDATE;
  SELECT e.* INTO STRICT v_anterior
    FROM vec_contratacion_temporal.expediente_version_integral e
   WHERE e.expediente_ref=v_actual.expediente_ref
     AND e.version=v_actual.version
   FOR UPDATE;
  IF v_actual.version<>(c->>'version_expediente')::numeric
     OR v_anterior.agregado_json<>z->'agregado_anterior'
     OR v_anterior.agregado_json_huella_sha256<>
        pg_catalog.encode(pg_catalog.sha256(
          pg_catalog.convert_to((z->'agregado_anterior')::text,'UTF8')
        ),'hex')
     OR v_anterior.agregado_json#>>'{organizacion_ref}'<>
        c->>'organizacion_ref'
     OR v_anterior.operacion_ref=c->>'actuacion_ref' THEN
    RAISE EXCEPTION USING ERRCODE='40001',
      MESSAGE='CAS del agregado O4-04E perdido';
  END IF;
  IF vec_contratacion_temporal.o404e_revalidar_gobierno_actual_v1(
       c->>'organizacion_ref',g->>'accion',
       g#>>'{catalogo,referencia}',(g#>>'{catalogo,version}')::numeric,
       g#>>'{catalogo,huella_sha256}',
       g#>>'{politica,referencia}',(g#>>'{politica,version}')::numeric,
       g#>>'{politica,huella_sha256}',
       g#>>'{politica_actuacion,referencia}',
       (g#>>'{politica_actuacion,version}')::numeric,
       g#>>'{politica_actuacion,huella_sha256}'
     ) IS NOT TRUE
     OR g->>'accion'<>x->>'accion'
     OR g->>'finalidad_vec'<>x->>'finalidad'
     OR g->>'finalidad_ct_ref'<>v_propuesta->>'finalidad_ref'
     OR g->>'finalidad_ct_clave'<>v_propuesta->>'finalidad_clave'
     OR g#>>'{catalogo,referencia}'<>v_propuesta#>>'{catalogo,referencia}'
     OR g#>>'{catalogo,version}'<>v_propuesta#>>'{catalogo,version}'
     OR g#>>'{catalogo,huella_sha256}'<>
        v_propuesta#>>'{catalogo,huella_sha256}'
     OR g#>>'{politica,referencia}'<>v_propuesta#>>'{politica,referencia}'
     OR g#>>'{politica,version}'<>v_propuesta#>>'{politica,version}'
     OR g#>>'{politica,huella_sha256}'<>
        v_propuesta#>>'{politica,huella_sha256}'
     OR p_confirmada_en>=(g->>'valida_hasta')::timestamptz THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='gobierno O4-04E no vigente';
  END IF;
  SELECT ca.publicacion_json AS catalogo,
         pd.publicacion_json AS politica,
         pa.publicacion_json AS actuacion
    INTO v_gobierno_actual
    FROM vec_contratacion_temporal.gobi_o404b_actual ga
    JOIN vec_contratacion_temporal.gobi_o404b_actuacion pa
      ON pa.referencia=ga.actuacion_ref
     AND pa.version=ga.actuacion_version
     AND pa.huella_sha256=ga.actuacion_huella_sha256
    JOIN vec_contratacion_temporal.gobi_o404b_politica pd
      ON pd.referencia=pa.politica_ref
     AND pd.version=pa.politica_version
     AND pd.huella_sha256=pa.politica_huella_sha256
    JOIN vec_contratacion_temporal.gobi_o404b_catalogo ca
      ON ca.referencia=pa.catalogo_ref
     AND ca.version=pa.catalogo_version
     AND ca.huella_sha256=pa.catalogo_huella_sha256
   WHERE ga.organizacion_ref=c->>'organizacion_ref'
     AND ga.accion=g->>'accion'
   FOR SHARE OF ga,pa,pd,ca;
  IF NOT FOUND
     OR g->'catalogo'<>v_gobierno_actual.catalogo
     OR g->'politica'<>v_gobierno_actual.politica
     OR g->'politica_actuacion'<>v_gobierno_actual.actuacion
     OR g->>'finalidad_ct_clave'<>
        v_gobierno_actual.actuacion->>'finalidad_contratacion_clave'
     OR g->>'finalidad_ct_ref'<>
        v_gobierno_actual.actuacion->>'finalidad_contratacion_ref'
     OR g->>'finalidad_vec'<>
        v_gobierno_actual.actuacion->>'finalidad_autorizacion_vec'
     OR g->>'unidad_ejecutora_ref'<>
        v_gobierno_actual.actuacion->>'unidad_ejecutora_ref'
     OR g->>'fase_destino'<>
        v_gobierno_actual.actuacion->>'fase_destino'
     OR g->>'estado_destino'<>
        v_gobierno_actual.actuacion->>'estado_destino'
     OR g->'motivo_autorizacion'<>
        v_gobierno_actual.actuacion->(CASE g->>'accion'
          WHEN 'contratacion_temporal.cobertura.decidir'
            THEN 'motivo_autorizacion_decidir'
          ELSE 'motivo_autorizacion_rectificar'
        END)
     OR (g->>'evaluada_en')::timestamptz<
        (c->>'observada_en_db')::timestamptz
     OR (g->>'evaluada_en')::timestamptz>p_confirmada_en
     OR (g->>'valida_hasta')::timestamptz<=
        (g->>'evaluada_en')::timestamptz THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='publicaciones de gobierno O4-04E divergentes';
  END IF;
  IF z->'motivo_funcional'<>pg_catalog.jsonb_build_object(
       'referencia_catalogo',pg_catalog.jsonb_build_object(
         'catalogo_id','','catalogo_version',0,
         'catalogo_huella_sha256','','entrada_clave',''),
       'clave_i18n','')
     AND vec_autorizacion.resolver_motivo_cobertura_actual_v1(
       z#>>'{motivo_funcional,referencia_catalogo,catalogo_id}',
       (z#>>'{motivo_funcional,referencia_catalogo,catalogo_version}')::integer,
       z#>>'{motivo_funcional,referencia_catalogo,catalogo_huella_sha256}',
       z#>>'{motivo_funcional,referencia_catalogo,entrada_clave}',
       z#>>'{motivo_funcional,clave_i18n}') IS NOT TRUE THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='motivo funcional O4-04E no vigente';
  END IF;
  v_lote:=vec_contratacion_temporal.o404e_construir_lote_c1_v1(
    p_carga,p_ambito_raiz
  );
  IF v_lote IS NULL
     OR vec_contratacion_temporal.o404e_concesion_ligada_v1(
          p_carga,v_lote) IS NOT TRUE
     OR pg_catalog.encode(pg_catalog.sha256(
          vec_contratacion_temporal
            .o404e_contexto_recurso_concesion_v1(p_carga)),'hex')
        IS DISTINCT FROM x->>'contexto_recurso_huella_sha256'
     OR v_lote->>'preparacion_c1_ref'<>c->>'preparacion_c1_ref'
     OR v_lote->>'preparacion_c1_huella_sha256'<>
        c->>'preparacion_c1_huella_sha256' THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='lote C1 O4-04E divergente';
  END IF;
  SELECT * INTO v_prevalidado
    FROM vec_contratacion_temporal
      .prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(v_lote);
  IF NOT FOUND OR v_prevalidado.estado<>'nueva' THEN
    RAISE EXCEPTION USING ERRCODE='23505',
      MESSAGE='lote C1 O4-04E ya utilizado';
  END IF;
  SELECT * INTO v_vec
    FROM vec_autorizacion
      .registrar_decision_cobertura_contexto_exacto_o404e_v1(
        pg_catalog.decode(x->>'decision_canonica_hex','hex'),
        pg_catalog.decode(x->>'motivo_canonico_hex','hex'),
        (x->>'persona_version')::numeric,(x->>'perfil_version')::numeric,
        pg_catalog.jsonb_build_object(
          'rama','concedida','accion',x->>'accion',
          'organizacion_ref',c->>'organizacion_ref',
          'expediente_ref',c->>'expediente_ref',
          'version_expediente',(c->>'version_expediente')::numeric,
          'reserva_ref',c->>'reserva_ref',
          'decision_ref',c->>'decision_vec_ref',
          'correlacion_ref',c->>'correlacion_vec_ref',
          'finalidad',x->>'finalidad',
          'contexto_recurso_huella_sha256',
            x->>'contexto_recurso_huella_sha256',
          'recurso_modulo',x->>'recurso_modulo',
          'recurso_ref',x->>'recurso_ref',
          'recurso_tipo',x->>'recurso_tipo',
          'ambitos',x->'ambitos','atributos',x->'atributos',
          'huella_orden_sha256',c->>'huella_orden_sha256',
          'lote_huella_sha256',v_lote->>'lote_huella_sha256'
        )
      );
  IF NOT FOUND OR v_vec.rama<>'concedida' OR NOT v_vec.concedida
     OR v_vec.codigo<>'concedida'
     OR v_vec.decision_ref<>c->>'decision_vec_ref'
     OR v_vec.decision_ref<>x->>'decision_ref'
     OR v_vec.correlacion_ref<>c->>'correlacion_vec_ref'
     OR v_vec.decision_huella_sha256<>x->>'decision_huella_sha256'
     OR v_vec.contexto_recurso_huella_sha256<>
        x->>'contexto_recurso_huella_sha256'
     OR v_vec.huella_orden_sha256<>c->>'huella_orden_sha256'
     OR v_vec.lote_huella_sha256<>v_lote->>'lote_huella_sha256'
     OR v_vec.registrada_en IS NULL
     OR v_vec.revalidada_en IS NULL
     OR v_vec.registrada_en<(x->>'emitida_en')::timestamptz
     OR v_vec.revalidada_en<v_vec.registrada_en
     OR v_vec.revalidada_en>=(x->>'valida_hasta')::timestamptz THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='resultado VEC concedido O4-04E divergente';
  END IF;
  v_vec_json:=pg_catalog.jsonb_build_object(
    'rama',v_vec.rama,'concedida',v_vec.concedida,'codigo',v_vec.codigo,
    'decision_ref',v_vec.decision_ref,
    'correlacion_ref',v_vec.correlacion_ref,
    'organizacion_ref',v_vec.organizacion_ref,
    'expediente_ref',v_vec.expediente_ref,
    'version_expediente',v_vec.version_expediente,
    'reserva_ref',v_vec.reserva_ref,
    'contexto_recurso_huella_sha256',
      v_vec.contexto_recurso_huella_sha256,
    'decision_huella_sha256',v_vec.decision_huella_sha256,
    'huella_orden_sha256',v_vec.huella_orden_sha256,
    'lote_huella_sha256',v_vec.lote_huella_sha256,
    'prueba_vinculo_sha256',v_vec.prueba_vinculo_sha256,
    'registrada_en',vec_contratacion_temporal.texto_instante_utc_go_v2(
      v_vec.registrada_en::text),
    'revalidada_en',vec_contratacion_temporal.texto_instante_utc_go_v2(
      v_vec.revalidada_en::text)
  );
  PERFORM * FROM vec_contratacion_temporal
    .persistir_lote_consumo_c1_cobertura_o404d_v1(v_lote,v_vec_json);
  p_confirmada_en:=pg_catalog.date_trunc(
    'microseconds',pg_catalog.clock_timestamp());
  IF p_confirmada_en<v_vec.revalidada_en
     OR p_confirmada_en>=(c->>'propiedad_hasta')::timestamptz
     OR p_confirmada_en>=(c->>'valida_hasta_orden')::timestamptz
     OR p_confirmada_en>=(c->>'preparacion_c1_valida_hasta')::timestamptz
     OR p_confirmada_en>=(z->>'valida_hasta')::timestamptz
     OR p_confirmada_en>=(g->>'valida_hasta')::timestamptz
     OR p_confirmada_en>=(x->>'valida_hasta')::timestamptz THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='vigencia final concedida O4-04E agotada';
  END IF;

  SELECT * INTO STRICT v_checkpoint
    FROM vec_contratacion_temporal.gobi_o404b_checkpoint
   WHERE control FOR SHARE;
  v_prueba:=vec_contratacion_temporal.o404e_texto_v1(
    ''::bytea,'VEC-CT-ACREDITACION-GOBIERNO-COBERTURA-O4-04E-V1');
  FOREACH v_huella IN ARRAY ARRAY[
    c->>'reserva_ref',c->>'recibo_ref',c->>'actuacion_ref',
    c->>'auditoria_ref',c->>'organizacion_ref',c->>'expediente_ref',
    c->>'version_expediente',g->>'accion',
    g#>>'{catalogo,referencia}',g#>>'{catalogo,version}',
    g#>>'{catalogo,huella_sha256}',g#>>'{politica,referencia}',
    g#>>'{politica,version}',g#>>'{politica,huella_sha256}',
    g#>>'{politica_actuacion,referencia}',
    g#>>'{politica_actuacion,version}',
    g#>>'{politica_actuacion,huella_sha256}',
    v_checkpoint.ultima_secuencia::text,
    v_checkpoint.ultimo_evento_ref,
    v_checkpoint.ultima_huella_evento_sha256,v_vec.decision_ref
  ]::text[] LOOP
    v_prueba:=vec_contratacion_temporal.o404e_texto_v1(v_prueba,v_huella);
  END LOOP;
  v_gobierno_huella:=pg_catalog.encode(pg_catalog.sha256(v_prueba),'hex');
  v_gobierno_ref:='gobierno-cobertura:sha256:'||v_gobierno_huella;
  INSERT INTO vec_contratacion_temporal
    .acreditacion_gobierno_decision_cobertura(
    acreditacion_ref,gobierno_huella_sha256,prueba_canonica,
    reserva_ref,recibo_ref,actuacion_ref,auditoria_ref,
    organizacion_ref,expediente_ref,version_expediente,accion,
    catalogo_ref,catalogo_version,catalogo_huella_sha256,
    politica_ref,politica_version,politica_huella_sha256,
    actuacion_gobierno_ref,actuacion_gobierno_version,
    actuacion_gobierno_huella_sha256,checkpoint_secuencia,
    checkpoint_evento_ref,checkpoint_evento_huella_sha256,
    decision_vec_ref,decision_vec_huella_sha256,evaluada_en,acreditada_en
  ) VALUES (
    v_gobierno_ref,v_gobierno_huella,v_prueba,c->>'reserva_ref',
    c->>'recibo_ref',c->>'actuacion_ref',c->>'auditoria_ref',
    c->>'organizacion_ref',c->>'expediente_ref',
    (c->>'version_expediente')::numeric,g->>'accion',
    g#>>'{catalogo,referencia}',(g#>>'{catalogo,version}')::numeric,
    g#>>'{catalogo,huella_sha256}',g#>>'{politica,referencia}',
    (g#>>'{politica,version}')::numeric,g#>>'{politica,huella_sha256}',
    g#>>'{politica_actuacion,referencia}',
    (g#>>'{politica_actuacion,version}')::numeric,
    g#>>'{politica_actuacion,huella_sha256}',
    v_checkpoint.ultima_secuencia,v_checkpoint.ultimo_evento_ref,
    v_checkpoint.ultima_huella_evento_sha256,v_vec.decision_ref,
    v_vec.decision_huella_sha256,(g->>'evaluada_en')::timestamptz,
    p_confirmada_en
  );
  v_version:=(c->>'version_expediente')::numeric+1;
  v_agregado_huella:=pg_catalog.encode(pg_catalog.sha256(
    pg_catalog.convert_to((z->'agregado_siguiente')::text,'UTF8')
  ),'hex');
  v_prueba:=pg_catalog.convert_to(
    'VEC-CT-EXPEDIENTE-INTEGRAL-V1'||chr(10),'UTF8')
    ||vec_contratacion_temporal.encuadrar_texto_v1(c->>'expediente_ref')
    ||vec_contratacion_temporal.encuadrar_texto_v1(v_version::text)
    ||vec_contratacion_temporal.encuadrar_texto_v1(v_agregado_huella)
    ||vec_contratacion_temporal.encuadrar_texto_v1(v_anterior.flujo_ref)
    ||vec_contratacion_temporal.encuadrar_texto_v1(
      v_anterior.flujo_version::text)
    ||vec_contratacion_temporal.encuadrar_texto_v1(
      v_anterior.flujo_huella_sha256)
    ||vec_contratacion_temporal.encuadrar_texto_v1(
      z#>>'{agregado_siguiente,fase_actual}')
    ||vec_contratacion_temporal.encuadrar_texto_v1(
      z#>>'{agregado_siguiente,estado_actual}')
    ||vec_contratacion_temporal.encuadrar_texto_v1('cobertura_o4')
    ||vec_contratacion_temporal.encuadrar_texto_v1(c->>'actuacion_ref')
    ||vec_contratacion_temporal.encuadrar_texto_v1(
      vec_contratacion_temporal.instante_utc_v1(p_confirmada_en));
  INSERT INTO vec_contratacion_temporal.expediente_version_integral(
    expediente_ref,version,agregado_json,agregado_json_huella_sha256,
    prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,
    flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,
    registrada_en
  ) VALUES (
    c->>'expediente_ref',v_version,z->'agregado_siguiente',
    v_agregado_huella,v_prueba,
    pg_catalog.encode(pg_catalog.sha256(v_prueba),'hex'),
    v_anterior.flujo_ref,v_anterior.flujo_version,
    v_anterior.flujo_huella_sha256,z#>>'{agregado_siguiente,fase_actual}',
    z#>>'{agregado_siguiente,estado_actual}','cobertura_o4',
    c->>'actuacion_ref',p_confirmada_en
  );
  UPDATE vec_contratacion_temporal.expediente_integral_actual
     SET version=v_version,actualizada_en=p_confirmada_en,
         operacion_ref=c->>'actuacion_ref'
   WHERE expediente_ref=c->>'expediente_ref'
     AND version=(c->>'version_expediente')::numeric;
  IF NOT FOUND THEN
    RAISE EXCEPTION USING ERRCODE='40001',
      MESSAGE='CAS final del agregado O4-04E perdido';
  END IF;
  v_decision:=z#>ARRAY[
    'agregado_siguiente','decisiones_cobertura',
    (pg_catalog.jsonb_array_length(
      z#>'{agregado_siguiente,decisiones_cobertura}')-1)::text
  ];
  v_actuacion:=z#>ARRAY[
    'agregado_siguiente','actuaciones',
    (pg_catalog.jsonb_array_length(
      z#>'{agregado_siguiente,actuaciones}')-1)::text
  ];
  IF v_decision#>>'{actuacion,unidad_ref}'<>g->>'unidad_ejecutora_ref'
     OR v_decision#>>'{actuacion,fase_destino}'<>g->>'fase_destino'
     OR v_decision#>>'{actuacion,estado_destino}'<>g->>'estado_destino'
     OR v_decision#>>'{actuacion,accion_clave}'<>g->>'accion'
     OR v_decision#>>'{actuacion,recibo_ref}'<>c->>'recibo_ref'
     OR v_decision->>'organizacion_ref'<>c->>'organizacion_ref'
     OR v_decision->>'expediente_ref'<>c->>'expediente_ref'
     OR v_decision->>'actor_ref' IS DISTINCT FROM
        v_decision#>>'{actuacion,actor_ref}'
     OR v_decision->>'actor_ref' IS DISTINCT FROM x->>'principal_id'
     OR v_decision#>>'{actuacion,actor_ref}' IS DISTINCT FROM
        x->>'principal_id'
     OR v_decision->>'perfil_ref' IS DISTINCT FROM
        x->>'perfil_activo_ref' THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='decisión y actuación O4-04E divergentes';
  END IF;
  v_prueba:=vec_contratacion_temporal.o404e_texto_v1(
    ''::bytea,'VEC-CT-ACTUACION-COBERTURA-O4-04E-V1');
  FOREACH v_huella IN ARRAY ARRAY[
    c->>'expediente_ref',v_version::text,c->>'actuacion_ref',
    c->>'recibo_ref',pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(v_actuacion::text,'UTF8')),'hex')
  ]::text[] LOOP
    v_prueba:=vec_contratacion_temporal.o404e_texto_v1(v_prueba,v_huella);
  END LOOP;
  INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
    expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,
    actuacion_json,actuacion_json_huella_sha256,prueba_canonica,
    prueba_huella_sha256,registrada_en
  ) VALUES (
    c->>'expediente_ref',v_version,v_version,c->>'actuacion_ref',
    c->>'recibo_ref',v_actuacion,pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(v_actuacion::text,'UTF8')),'hex'),
    v_prueba,pg_catalog.encode(pg_catalog.sha256(v_prueba),'hex'),
    p_confirmada_en
  );
  INSERT INTO vec_contratacion_temporal
    .decision_cobertura_gobernada_durable(
    decision_ref,decision_huella_sha256,decision_json,decision_canon,
    propuesta_ref,propuesta_huella_sha256,propuesta_json,propuesta_canon,
    tipo,organizacion_ref,expediente_ref,version_expediente_origen,
    version_expediente,reserva_ref,recibo_ref,actuacion_ref,auditoria_ref,
    accion,actor_ref,perfil_ref,via_elegida,via_recomendada,
    predecesora_ref,predecesora_huella_sha256,
    acreditacion_gobierno_ref,gobierno_huella_sha256,
    consumo_c1_lote_ref,consumo_c1_lote_huella_sha256,
    decision_vec_ref,decision_vec_huella_sha256,decidida_en,persistida_en
  ) VALUES (
    v_decision->>'referencia',v_decision->>'huella_sha256',v_decision,
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(v_decision),
    v_propuesta->>'referencia',v_propuesta->>'huella_sha256',v_propuesta,
    vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(v_propuesta),
    v_decision->>'tipo',c->>'organizacion_ref',c->>'expediente_ref',
    (c->>'version_expediente')::numeric,v_version,c->>'reserva_ref',
    c->>'recibo_ref',c->>'actuacion_ref',c->>'auditoria_ref',g->>'accion',
    v_decision->>'actor_ref',v_decision->>'perfil_ref',
    v_decision->>'via_elegida',v_decision->>'via_recomendada',
    v_decision->>'predecesora_ref',v_decision->>'predecesora_huella_sha256',
    v_gobierno_ref,v_gobierno_huella,v_lote->>'lote_ref',
    v_lote->>'lote_huella_sha256',v_vec.decision_ref,
    v_vec.decision_huella_sha256,
    (v_decision->>'decidida_en')::timestamptz,p_confirmada_en
  );
  RETURN vec_contratacion_temporal.o404e_cerrar_terminal_v1(
    p_carga,v_vec_json,pg_catalog.jsonb_build_object(
      'gobierno_ref',v_gobierno_ref,
      'gobierno_huella_sha256',v_gobierno_huella,
      'lote_ref',v_lote->>'lote_ref',
      'lote_huella_sha256',v_lote->>'lote_huella_sha256',
      'decision_ref',v_decision->>'referencia',
      'decision_huella_sha256',v_decision->>'huella_sha256',
      'version_resultante',v_version
    ),p_confirmada_en
  );
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(
  p_carga jsonb
)
RETURNS TABLE(recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
SET row_security='on'
SET timezone='UTC'
SET lock_timeout='2s'
AS $funcion$
DECLARE
  c jsonb:=p_carga->'cabecera';
  v_base record;
  v_alias record;
  v_actual record;
  v_reserva record;
  v_recibo jsonb;
  v_ahora timestamptz(6);
BEGIN
  IF current_user<>'vec_contratacion_temporal_propietario'
     OR session_user=current_user
     OR NOT pg_catalog.pg_has_role(
       session_user,
       'vec_contratacion_temporal_confirmador_cobertura','MEMBER')
     OR (SELECT pg_catalog.count(*)
           FROM pg_catalog.pg_auth_members m
           JOIN pg_catalog.pg_roles u ON u.oid=m.member
          WHERE u.rolname=session_user)<>1
     OR NOT EXISTS(
       SELECT 1 FROM pg_catalog.pg_auth_members m
       JOIN pg_catalog.pg_roles u ON u.oid=m.member
       JOIN pg_catalog.pg_roles r ON r.oid=m.roleid
       WHERE u.rolname=session_user
         AND r.rolname='vec_contratacion_temporal_confirmador_cobertura'
         AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_propietario','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_migrador','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
     OR pg_catalog.pg_has_role(
       session_user,'vec_contratacion_temporal_gobernador','MEMBER')
     OR vec_contratacion_temporal.gobi_o404b_entorno_valido(false)
        IS NOT TRUE
     OR pg_catalog.pg_is_in_recovery()
     OR p_carga IS NULL OR pg_catalog.jsonb_typeof(p_carga)<>'object'
     OR pg_catalog.pg_column_size(p_carga)>25165824
     OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
       p_carga,ARRAY['cabecera','concesion','consumos_c1','decision_vec',
       'denegacion','esquema','gobierno','rama']::text[])
     OR p_carga->>'esquema'<>
       'vec.contratacion-temporal.confirmar-operacion-decision-cobertura.o4-04e.v1'
     OR p_carga->>'rama' NOT IN ('concedida','denegada')
     OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
       p_carga->'decision_vec',ARRAY[
       'accion','ambitos','atributos','codigo_probatorio','concedida',
       'contexto_recurso_huella_sha256','correlacion_ref',
       'decision_canonica_hex','decision_huella_sha256','decision_ref',
       'emitida_en','finalidad','motivo_canonico_hex',
       'perfil_activo_ref','perfil_version','persona_version',
       'principal_id','recurso_modulo','recurso_ref','recurso_tipo',
       'valida_hasta']::text[])
     OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
       c,ARRAY[
       'actuacion_ref','ambito_idempotencia_hmac','analisis_huella_sha256',
       'analisis_ref','auditoria_ref','correlacion_vec_ref',
       'decision_vec_ref','esquema_sesion','evento_ref','expediente_ref',
       'huella_orden_sha256','huella_ordenes_consumo_c1_sha256',
       'huella_semantica_hmac','numero_consumos_c1','observada_en_db',
       'organizacion_ref','preparacion_c1_huella_sha256',
       'preparacion_c1_preparada_en','preparacion_c1_ref',
       'preparacion_c1_valida_hasta','propiedad_hasta','recibo_ref',
       'reserva_ref','revision_cercado','revision_cercado_anterior',
       'token_propietario_sha256','valida_hasta_orden',
      'version_expediente']::text[])
     OR pg_catalog.jsonb_typeof(p_carga->'decision_vec'->'ambitos')<>'object'
     OR pg_catalog.jsonb_typeof(p_carga->'decision_vec'->'atributos')<>'object'
     OR pg_catalog.jsonb_typeof(
          p_carga->'decision_vec'->'concedida')<>'boolean'
     OR pg_catalog.jsonb_typeof(
          p_carga->'decision_vec'->'persona_version')<>'number'
     OR pg_catalog.jsonb_typeof(
          p_carga->'decision_vec'->'perfil_version')<>'number'
     OR vec_contratacion_temporal.numero_entero_json_canonico_v2(
          p_carga->'decision_vec'->'persona_version',
          1,9007199254740991::numeric) IS NOT TRUE
     OR vec_contratacion_temporal.numero_entero_json_canonico_v2(
          p_carga->'decision_vec'->'perfil_version',
          1,9007199254740991::numeric) IS NOT TRUE
     OR EXISTS(
       SELECT 1 FROM pg_catalog.jsonb_each(p_carga->'decision_vec') e
        WHERE e.key=ANY(ARRAY[
          'accion','codigo_probatorio','contexto_recurso_huella_sha256',
          'correlacion_ref','decision_canonica_hex',
          'decision_huella_sha256','decision_ref','emitida_en',
          'finalidad','motivo_canonico_hex','perfil_activo_ref',
          'principal_id','recurso_modulo','recurso_ref','recurso_tipo',
          'valida_hasta']::text[])
          AND pg_catalog.jsonb_typeof(e.value)<>'string')
     OR (p_carga->>'rama'='concedida' AND
          p_carga#>>'{decision_vec,concedida}' IS DISTINCT FROM 'true')
     OR (p_carga->>'rama'='denegada' AND
          p_carga#>>'{decision_vec,concedida}' IS DISTINCT FROM 'false')
     OR p_carga#>>'{decision_vec,recurso_modulo}' IS DISTINCT FROM
          'contratacion_temporal'
     OR p_carga#>>'{decision_vec,recurso_tipo}' IS DISTINCT FROM
          'decision_cobertura_gobernada'
     OR p_carga#>>'{decision_vec,recurso_ref}' IS DISTINCT FROM
          c->>'reserva_ref'
     OR p_carga#>>'{decision_vec,decision_ref}' IS DISTINCT FROM
          c->>'decision_vec_ref'
     OR p_carga#>>'{decision_vec,correlacion_ref}' IS DISTINCT FROM
          c->>'correlacion_vec_ref'
     OR p_carga#>>'{decision_vec,finalidad}' !~
          '^[a-z][a-z0-9._-]{1,79}$'
     OR c->>'esquema_sesion'<>
       'VEC-CT-SESION-TCB-OPERACION-DECISION-COBERTURA-V1'
     OR (c->>'huella_orden_sha256')!~'^[a-f0-9]{64}$' THEN
    RAISE EXCEPTION USING ERRCODE='42501',
      MESSAGE='confirmación O4-04E no autorizada';
  END IF;
  IF p_carga->>'rama'='concedida'
     AND (
       NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
         p_carga->'gobierno',ARRAY[
          'accion','catalogo','estado_destino','evaluada_en',
          'fase_destino','finalidad_ct_clave','finalidad_ct_ref',
          'finalidad_vec','motivo_autorizacion','politica',
          'politica_actuacion','unidad_ejecutora_ref',
          'valida_hasta']::text[])
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
         p_carga->'concesion',ARRAY[
          'agregado_anterior','agregado_siguiente','efecto_en',
          'motivo_funcional','propuesta','valida_hasta']::text[])
     ) THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='carga concedida O4-04E no cerrada';
  END IF;
  PERFORM pg_catalog.pg_advisory_xact_lock_shared(
    pg_catalog.hashtextextended(
      'vec_contratacion_temporal:o4_04:migraciones',0));
  PERFORM 1 FROM vec_contratacion_temporal.control_migracion_cobertura_o4
   WHERE control AND version_esquema=14 FOR SHARE;
  IF NOT FOUND THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='barrera O4-04E no disponible';
  END IF;
  SELECT b.* INTO STRICT v_base
    FROM vec_contratacion_temporal.alias_operacion_decision_cobertura a
    JOIN vec_contratacion_temporal.reserva_operacion_decision_cobertura b
      USING(ambito_raiz_hmac)
   WHERE a.alias_ambito_hmac=c->>'ambito_idempotencia_hmac'
     AND a.alias_huella_semantica_hmac=c->>'huella_semantica_hmac'
   FOR UPDATE OF b;
  SELECT a.* INTO STRICT v_alias
    FROM vec_contratacion_temporal.alias_operacion_decision_cobertura a
   WHERE a.alias_ambito_hmac=c->>'ambito_idempotencia_hmac'
     AND a.ambito_raiz_hmac=v_base.ambito_raiz_hmac;
  SELECT a.* INTO STRICT v_actual
    FROM vec_contratacion_temporal
      .reserva_operacion_decision_cobertura_actual a
   WHERE a.ambito_raiz_hmac=v_base.ambito_raiz_hmac
   FOR UPDATE;
  SELECT r.* INTO STRICT v_reserva
    FROM vec_contratacion_temporal
      .reserva_operacion_decision_cobertura_version r
   WHERE r.ambito_raiz_hmac=v_actual.ambito_raiz_hmac
     AND r.secuencia=v_actual.secuencia
   FOR UPDATE;
  IF v_base.organizacion_ref IS DISTINCT FROM c->>'organizacion_ref'
     OR v_base.expediente_ref IS DISTINCT FROM c->>'expediente_ref'
     OR v_base.version_expediente IS DISTINCT FROM
        (c->>'version_expediente')::numeric
     OR v_base.reserva_ref IS DISTINCT FROM c->>'reserva_ref'
     OR v_base.recibo_ref IS DISTINCT FROM c->>'recibo_ref'
     OR v_base.actuacion_ref IS DISTINCT FROM c->>'actuacion_ref'
     OR v_base.auditoria_ref IS DISTINCT FROM c->>'auditoria_ref'
     OR v_base.evento_ref IS DISTINCT FROM c->>'evento_ref'
     OR v_base.correlacion_vec_ref IS DISTINCT FROM
        c->>'correlacion_vec_ref'
     OR v_base.decision_vec_ref IS DISTINCT FROM c->>'decision_vec_ref'
     OR (
       p_carga->>'rama'='concedida'
       AND (
         v_base.analisis_ref IS DISTINCT FROM c->>'analisis_ref'
         OR v_base.analisis_huella_sha256 IS DISTINCT FROM
            c->>'analisis_huella_sha256'
       )
     )
     OR v_reserva.revision_cercado IS DISTINCT FROM
        (c->>'revision_cercado')::numeric
     OR (c->>'revision_cercado_anterior')::numeric IS DISTINCT FROM
        (c->>'revision_cercado')::numeric-1 THEN
    RAISE EXCEPTION USING ERRCODE='22023',
      MESSAGE='identidad de reserva O4-04E divergente';
  END IF;
  IF v_reserva.estado IN ('aplicada','denegada_vec') THEN
    v_recibo:=vec_contratacion_temporal.o404e_leer_terminal_interno_v1(
      v_base.ambito_raiz_hmac,c->>'huella_orden_sha256');
    IF v_recibo IS NULL
       OR NOT vec_contratacion_temporal.o404e_iguales_constante_v1(
         v_recibo->>'ambito_idempotencia_hmac',
         c->>'ambito_idempotencia_hmac')
       OR NOT vec_contratacion_temporal.o404e_iguales_constante_v1(
         v_recibo->>'huella_semantica_hmac',
         c->>'huella_semantica_hmac')
       OR v_recibo->>'recibo_ref'<>c->>'recibo_ref'
       OR v_recibo->>'reserva_ref'<>c->>'reserva_ref'
       OR v_recibo->>'decision_vec_ref'<>c->>'decision_vec_ref'
       OR v_recibo->>'correlacion_vec_ref'<>c->>'correlacion_vec_ref'
       OR (v_recibo->>'revision_cercado')::numeric<>
          (c->>'revision_cercado')::numeric
       OR NOT EXISTS(
         SELECT 1
           FROM vec_contratacion_temporal
             .confirmacion_operacion_decision_cobertura cf
          WHERE cf.ambito_raiz_hmac=v_base.ambito_raiz_hmac
            AND cf.carga_huella_sha256=pg_catalog.encode(
              pg_catalog.sha256(
                pg_catalog.convert_to(p_carga::text,'UTF8')
              ),'hex')
       )
       OR (p_carga->>'rama'='concedida' AND
           v_recibo->>'aplicada'<>'true')
       OR (p_carga->>'rama'='denegada' AND (
           v_recibo->>'denegada_vec'<>'true'
           OR NOT EXISTS(
             SELECT 1
               FROM vec_contratacion_temporal
                 .prueba_denegacion_decision_cobertura pd
              WHERE pd.ambito_raiz_hmac=v_base.ambito_raiz_hmac
                AND pd.decision_vec_ref=c->>'decision_vec_ref'
                AND pd.prueba_huella_sha256=pg_catalog.encode(
                  pg_catalog.sha256(
                    vec_contratacion_temporal
                      .o404e_material_prueba_denegacion_v1(p_carga)
                  ),'hex')
           )
       )) THEN
      RAISE EXCEPTION USING ERRCODE='55000',
        MESSAGE='replay terminal O4-04E no acreditado';
    END IF;
    RETURN QUERY SELECT v_recibo;
    RETURN;
  END IF;
  v_ahora:=pg_catalog.date_trunc(
    'microseconds',pg_catalog.clock_timestamp());
  IF v_reserva.estado<>'reservada'
     OR NOT vec_contratacion_temporal.o404e_iguales_constante_v1(
       v_alias.alias_huella_semantica_hmac,c->>'huella_semantica_hmac')
     OR NOT vec_contratacion_temporal.o404e_iguales_constante_v1(
       v_reserva.token_propietario_sha256,c->>'token_propietario_sha256')
     OR v_reserva.observada_en<>(c->>'observada_en_db')::timestamptz
     OR v_reserva.propiedad_hasta<>(c->>'propiedad_hasta')::timestamptz
     OR v_ahora>=v_reserva.propiedad_hasta
     OR v_ahora>=(c->>'valida_hasta_orden')::timestamptz
     OR (
       p_carga->>'rama'='concedida'
       AND v_ahora>=(c->>'preparacion_c1_valida_hasta')::timestamptz
     ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='propiedad O4-04E no vigente';
  END IF;
  IF p_carga->>'rama'='concedida' THEN
    v_recibo:=vec_contratacion_temporal.o404e_confirmar_concesion_v1(
      p_carga,v_base.ambito_raiz_hmac,v_ahora);
  ELSE
    v_recibo:=vec_contratacion_temporal.o404e_confirmar_denegacion_v1(
      p_carga,v_ahora);
  END IF;
  RETURN QUERY SELECT v_recibo;
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema=13,
       actualizada_en=
         pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
 WHERE control AND version_esquema=12;
REVOKE ALL ON FUNCTION
 vec_contratacion_temporal.o404e_confirmar_concesion_v1(
   jsonb,text,timestamptz),
 vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(
   jsonb)
FROM PUBLIC,vec_contratacion_temporal_ejecutor,
 vec_contratacion_temporal_confirmador_cobertura,
 vec_contratacion_temporal_migrador,vec_contratacion_temporal_gobernador;
COMMIT;
