\set ON_ERROR_STOP on
\if :{?o404e_solo_casos}
\else
-- Requiere los fixtures sintéticos ContextoActor/VEC de la suite O4-04D.
CREATE SCHEMA vec_o404e_prueba;
REVOKE ALL ON SCHEMA vec_o404e_prueba FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_o404e_prueba TO vec_o404e_tcb;

CREATE TABLE vec_o404e_prueba.carga_denegada(
  carga jsonb NOT NULL,recibo jsonb
);
REVOKE ALL ON TABLE vec_o404e_prueba.carga_denegada FROM PUBLIC;

CREATE FUNCTION vec_o404e_prueba.carga()
RETURNS jsonb
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$
  SELECT carga FROM vec_o404e_prueba.carga_denegada
$$;
REVOKE ALL ON FUNCTION vec_o404e_prueba.carga() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_o404e_prueba.carga() TO vec_o404e_tcb;
CREATE FUNCTION vec_o404e_prueba.recibo()
RETURNS jsonb
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$ SELECT recibo FROM vec_o404e_prueba.carga_denegada $$;
CREATE FUNCTION vec_o404e_prueba.guardar_recibo(p_recibo jsonb)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$
BEGIN
  IF session_user<>'vec_o404e_tcb' OR p_recibo IS NULL THEN
    RAISE EXCEPTION 'recibo denegado fuera de contrato';
  END IF;
  UPDATE vec_o404e_prueba.carga_denegada SET recibo=p_recibo
   WHERE recibo IS NULL;
  RETURN FOUND;
END
$$;
REVOKE ALL ON FUNCTION vec_o404e_prueba.recibo(),
  vec_o404e_prueba.guardar_recibo(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_o404e_prueba.recibo(),
  vec_o404e_prueba.guardar_recibo(jsonb) TO vec_o404e_tcb;

CREATE FUNCTION vec_o404e_prueba.preparar_denegacion(
  p_identidad text DEFAULT 'eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee'
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
SET timezone='UTC'
AS $$
DECLARE
  -- Fuerza un cero final: la regresión prueba que VEC recibe siempre seis cifras.
  v_ahora timestamptz(6):=date_bin(
    interval '10 microseconds',clock_timestamp(),
    timestamptz '2000-01-01 00:00:00+00');
  v_propiedad timestamptz(6):=v_ahora+interval '4.8 seconds';
  v_hasta timestamptz(6):=v_ahora+interval '4.7 seconds';
  v_ambito text:='hmac-sha256:vec.contratacion-temporal.cobertura-decision.ambito/v1:'||repeat('1',64);
  v_semantica text:='hmac-sha256:vec.contratacion-temporal.cobertura-decision.semantica/v1:'||repeat('2',64);
  v_orden text:=repeat('d',64);
  v_token text:=repeat('e',64);
  v_prueba bytea:=decode(repeat('ab',128),'hex');
  v_correlacion text;
  v_decision_ref text;
  v_agregado jsonb:='{}'::jsonb;
  v_recurso_material bytea:=''::bytea;
  v_recurso_huella text;
  v_ambitos jsonb;
  v_atributos jsonb;
  v_contexto record;
  v_sesion record;
  v_motivo bytea;
  v_motivo_ref jsonb;
  v_vinculo jsonb;
  v_decision jsonb;
  v_decision_canon bytea;
  v_cabecera jsonb;
  v_denegacion jsonb;
  v_carga jsonb;
BEGIN
  IF session_user<>'vec_o404e_tcb'
     OR p_identidad!~'^[a-f0-9]{32}$'
     OR (SELECT count(*) FROM vec_o404e_prueba.carga_denegada)<>0 THEN
    RAISE EXCEPTION 'preparación O4-04E fuera de contrato';
  END IF;
  v_correlacion:='correlacion_'||p_identidad;
  v_decision_ref:=CASE
    WHEN p_identidad='eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee'
      THEN 'decision:o404e:denegada'
    ELSE 'decision:o404e:denegada:'||p_identidad END;
  SELECT * INTO STRICT v_contexto
    FROM vec_contexto_actor_v1.registros_contexto
   WHERE registro_contexto_ref=
         'rca_registro_v3_000000000000000000000000';
  SELECT s.autenticacion_verificada_en,s.sesion_emitida_en,
         c.sesion_revalidada_en,c.sesion_valida_hasta
    INTO STRICT v_sesion
    FROM vec_autorizacion.sesion_autenticacion_v1 s
    JOIN vec_autorizacion.control_sesion_v1 c USING(sesion_ref)
   WHERE s.sesion_ref='ses_registro_v3_0000000000000000000000';

  PERFORM set_config('session_replication_role','replica',true);
  INSERT INTO vec_contratacion_temporal.expediente_version_integral(
    expediente_ref,version,agregado_json,agregado_json_huella_sha256,
    prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,
    flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,
    registrada_en
  ) VALUES (
    'expediente:o404e:denegado',2,v_agregado,
    encode(sha256(convert_to(v_agregado::text,'UTF8')),'hex'),
    v_prueba,encode(sha256(v_prueba),'hex'),'flujo:o404e',1,repeat('3',64),
    'fase_o404e','en_curso','analisis_o3','operacion:o404e:base',v_ahora
  );
  INSERT INTO vec_contratacion_temporal
    .reserva_operacion_decision_cobertura VALUES (
    v_ambito,'reserva:o404e:denegada','recibo:o404e:denegada',
    'actuacion:o404e:denegada','auditoria:o404e:denegada',
    'evento:o404e:denegada',v_correlacion,
    v_decision_ref,'organizacion:o404e',
    'expediente:o404e:denegado',2,'analisis:o404e',repeat('4',64),
    v_semantica,v_ahora
  );
  INSERT INTO vec_contratacion_temporal
    .alias_operacion_decision_cobertura VALUES (
    v_ambito,v_ambito,1,v_semantica,v_ahora
  );
  INSERT INTO vec_contratacion_temporal
    .reserva_operacion_decision_cobertura_version VALUES (
    v_ambito,1,'reservada',1,v_token,v_ahora,v_propiedad,NULL,NULL
  );
  INSERT INTO vec_contratacion_temporal
    .reserva_operacion_decision_cobertura_actual VALUES(v_ambito,1);
  PERFORM set_config('session_replication_role','origin',true);

  v_ambitos:=jsonb_build_object(
    'organizacion_ref','organizacion:o404e',
    'unidad_ejecutora_ref','unidad:o404e');
  v_atributos:=jsonb_build_object(
    'accion','contratacion_temporal.cobertura.decidir',
    'analisis_huella_sha256',repeat('4',64),
    'analisis_ref','analisis:o404e',
    'catalogo_huella_sha256',repeat('5',64),
    'catalogo_ref','catalogo:o404e','catalogo_version','1',
    'expediente_ref','expediente:o404e:denegado',
    'politica_actuacion_huella_sha256',repeat('6',64),
    'politica_actuacion_ref','politica-actuacion:o404e',
    'politica_actuacion_version','1',
    'politica_huella_sha256',repeat('7',64),
    'politica_ref','politica:o404e','politica_version','1',
    'preparacion_evidencias_huella_sha256',repeat('8',64),
    'preparacion_evidencias_ref','preparacion:o404e',
    'propuesta_huella_sha256',repeat('9',64),
    'propuesta_ref','propuesta:o404e',
    'propuesta_semantica_huella_sha256',repeat('a',64),
    'propuesta_semantica_ref',
      'propuesta-cobertura-semantica:sha256:'||repeat('a',64),
    'reserva_ref','reserva:o404e:denegada','revision_cercado','1',
    'tipo_operacion','inicial','version_expediente_esperada','2',
    'via_elegida','via_o404e');
  v_recurso_material:=convert_to(
    '{"ambitos":'||
      vec_contratacion_temporal.o404e_mapa_json_go_v1(v_ambitos)||
    ',"atributos":'||
      vec_contratacion_temporal.o404e_mapa_json_go_v1(v_atributos)||'}',
    'UTF8');
  v_recurso_huella:=encode(sha256(v_recurso_material),'hex');

  v_motivo:=convert_to(
    '{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada","referencia":{"catalogo_id":"motivos_v3","catalogo_version":1,"catalogo_huella_sha256":"'
    ||repeat('9',64)
    ||'","entrada_clave":"motivo_33333333333333333333333333333333"}}',
    'UTF8');
  v_motivo_ref:=jsonb_build_object(
    'catalogo_id','motivos_v3','catalogo_version',1,
    'catalogo_huella_sha256',repeat('9',64),
    'entrada_clave','motivo_33333333333333333333333333333333');
  v_vinculo:=jsonb_build_object(
    'esquema','vec.autenticacion-actor.vinculo.v2.contexto-registrado',
    'bloque_version',2,
    'autenticacion_ref','aut_registro_v3_0000000000000000000000',
    'autenticacion_huella_sha256',repeat('5',64),
    'asercion_ref','ase_registro_v3_0000000000000000000000',
    'sesion_ref','ses_registro_v3_0000000000000000000000',
    'control_sesion_ref','cse_registro_v3_0000000000000000000000',
    'control_sesion_revision',1,
    'control_sesion_huella_sha256',repeat('7',64),
    'cuenta_ref','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    'cuenta_ordinaria_ref','cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
    'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
    'cuenta_privilegiada',false,'superficie','interna_corporativa',
    'metodo_observado','certificado','garantia_observada','alto',
    'politica_garantia_ref','pga_registro_v3_0000000000000000000000',
    'politica_garantia_huella_sha256',repeat('6',64),
    'autenticacion_verificada_en',
      to_char(v_sesion.autenticacion_verificada_en AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'sesion_emitida_en',
      to_char(v_sesion.sesion_emitida_en AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'sesion_valida_hasta',
      to_char(v_sesion.sesion_valida_hasta AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'sesion_revalidada_en',
      to_char(v_sesion.sesion_revalidada_en AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'registro_contexto_ref',v_contexto.registro_contexto_ref,
    'contexto_actor_esquema','vec.contexto-actor.vinculado.v2',
    'contexto_actor_ref','vca_sintetico_dddddddddddddddddddddddd',
    'contexto_actor_version',2,'contexto_actor_cuenta_version',2,
    'contexto_actor_huella_sha256',v_contexto.huella_sha256,
    'manifiesto_procedencia_huella_sha256',
      v_contexto.manifiesto_procedencia_huella_sha256,
    'autoridad_efectiva','autoridad_maestra_acreditada'
  );
  v_decision:=jsonb_build_object(
    'esquema','vec.autorizacion.decision.v3.solicitud-ligada.actor-v2',
    'bloque_version',3,'decision_ref',v_decision_ref,
    'concedida',false,'codigo','accion_no_concedida',
    'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
    'accion','contratacion_temporal.cobertura.decidir',
    'recurso_ref','reserva:o404e:denegada',
    'modulo_id','contratacion_temporal',
    'tipo_recurso','decision_cobertura_gobernada',
    'contexto_recurso_huella_sha256',v_recurso_huella,
    'finalidad','gestion',
    'correlacion_ref',v_correlacion,
    'esquema_huella_solicitud',
      'vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2',
    'solicitud_huella_sha256',repeat('c',64),
    'esquema_huella_motivo',
      'vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
    'motivo_huella_sha256',encode(sha256(v_motivo),'hex'),
    'vinculo_autenticacion_actor',v_vinculo,
    'asignacion_ref','asignacion:registro_v3:v1',
    'asignacion_huella_sha256',repeat('4',64),
    'version_rol_ref','rol:registro_v3:v1',
    'version_rol_huella_sha256',repeat('2',64),
    'control_vigencia_version_rol_ref','rol:registro_v3:v1',
    'control_vigencia_version_rol_revision',1,
    'control_vigencia_version_rol_huella_sha256',repeat('3',64),
    'revision_catalogo_politicas',1,
    'catalogo_politicas_huella_sha256',
      '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
    'politicas_evaluadas','[]'::jsonb,'politicas_aplicables','[]'::jsonb,
    'garantia_minima','alto',
    'campos_permitidos',jsonb_build_array('estado'),
    'obligaciones',jsonb_build_array('auditar'),
    'emitida_en',
      to_char(v_ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'valida_hasta',
      to_char(v_hasta AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
  );
  IF v_decision->>'emitida_en'!~'[.][0-9]{6}Z$'
     OR v_decision->>'valida_hasta'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'autenticacion_verificada_en'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'sesion_emitida_en'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'sesion_valida_hasta'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'sesion_revalidada_en'!~'[.][0-9]{6}Z$' THEN
    RAISE EXCEPTION 'instante VEC denegado sin seis microsegundos';
  END IF;
  v_decision_canon:=
    vec_autorizacion.decision_contexto_actor_v3_canonica(v_decision);
  IF v_decision_canon IS NULL THEN
    RAISE EXCEPTION 'decisión VEC de prueba no canónica';
  END IF;
  v_cabecera:=jsonb_build_object(
    'esquema_sesion','VEC-CT-SESION-TCB-OPERACION-DECISION-COBERTURA-V1',
    'huella_orden_sha256',v_orden,'organizacion_ref','organizacion:o404e',
    'expediente_ref','expediente:o404e:denegado','version_expediente',2,
    'reserva_ref','reserva:o404e:denegada',
    'recibo_ref','recibo:o404e:denegada',
    'actuacion_ref','actuacion:o404e:denegada',
    'auditoria_ref','auditoria:o404e:denegada',
    'evento_ref','evento:o404e:denegada',
    'correlacion_vec_ref',v_correlacion,
    'decision_vec_ref',v_decision_ref,
    'analisis_ref','','analisis_huella_sha256','',
    'token_propietario_sha256',v_token,
    'ambito_idempotencia_hmac',v_ambito,
    'huella_semantica_hmac',
      'hmac-sha256:vec.contratacion-temporal.cobertura-decision.semantica/v1:'
      ||repeat('2',64),
    'revision_cercado_anterior',0,'revision_cercado',1,
    'observada_en_db',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'propiedad_hasta',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_propiedad::text),
    'valida_hasta_orden',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text),
    'preparacion_c1_ref','','preparacion_c1_huella_sha256','',
    'preparacion_c1_preparada_en','0001-01-01T00:00:00Z',
    'preparacion_c1_valida_hasta','0001-01-01T00:00:00Z',
    'numero_consumos_c1',0,'huella_ordenes_consumo_c1_sha256',''
  );
  v_denegacion:=jsonb_build_object(
    'organizacion_ref','organizacion:o404e',
    'expediente_ref','expediente:o404e:denegado','version_expediente',2,
    'reserva_ref','reserva:o404e:denegada',
    'recibo_ref','recibo:o404e:denegada',
    'auditoria_ref','auditoria:o404e:denegada',
    'correlacion_vec_ref',v_correlacion,
    'decision_vec_ref',v_decision_ref,'revision_cercado',1,
    'recurso_ref','reserva:o404e:denegada',
    'recurso_modulo','contratacion_temporal',
    'recurso_tipo','decision_cobertura_gobernada',
    'ambitos',v_ambitos,'atributos',v_atributos,
    'recurso_huella_sha256',v_recurso_huella,
    'actor_ref','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'perfil_ref','prf_sintetico_cccccccccccccccccccccccc',
    'accion_vec','contratacion_temporal.cobertura.decidir',
    'finalidad_vec','gestion','motivo_vec',v_motivo_ref,
    'limite_preparacion',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text),
    'valida_hasta',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text),
    'prueba_huella_sha256',''
  );
  v_carga:=jsonb_build_object(
    'esquema',
      'vec.contratacion-temporal.confirmar-operacion-decision-cobertura.o4-04e.v1',
    'rama','denegada','cabecera',v_cabecera,'gobierno',NULL,
    'decision_vec',jsonb_build_object(
      'decision_canonica_hex',encode(v_decision_canon,'hex'),
      'motivo_canonico_hex',encode(v_motivo,'hex'),
      'persona_version',2,'perfil_version',2,
      'decision_ref',v_decision_ref,
      'decision_huella_sha256',encode(sha256(v_decision_canon),'hex'),
      'codigo_probatorio','accion_no_concedida','concedida',false,
      'emitida_en',
        to_char(v_ahora AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'valida_hasta',
        to_char(v_hasta AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
      'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
      'accion','contratacion_temporal.cobertura.decidir',
      'recurso_ref','reserva:o404e:denegada',
      'recurso_modulo','contratacion_temporal',
      'recurso_tipo','decision_cobertura_gobernada',
      'ambitos',v_ambitos,'atributos',v_atributos,
      'contexto_recurso_huella_sha256',v_recurso_huella,
      'finalidad','gestion',
      'correlacion_ref',v_correlacion
    ),
    'consumos_c1','[]'::jsonb,'concesion',NULL,'denegacion',v_denegacion
  );
  v_denegacion:=jsonb_set(
    v_denegacion,'{prueba_huella_sha256}',
    to_jsonb(encode(sha256(
      vec_contratacion_temporal.o404e_material_prueba_denegacion_v1(v_carga)
    ),'hex'))
  );
  v_carga:=jsonb_set(v_carga,'{denegacion}',v_denegacion);
  INSERT INTO vec_o404e_prueba.carga_denegada VALUES(v_carga);
  RETURN v_carga;
END
$$;
REVOKE ALL ON FUNCTION vec_o404e_prueba.preparar_denegacion(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
  vec_o404e_prueba.preparar_denegacion(text) TO vec_o404e_tcb;
\endif

\if :{?o404e_solo_preparar}
\quit
\endif

SET SESSION AUTHORIZATION vec_o404e_tcb;
\if :{?o404e_saltar_primera}
SELECT vec_o404e_prueba.recibo() AS recibo \gset
\else
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_prueba.preparar_denegacion() AS carga \gset
SELECT recibo_json AS recibo
  FROM vec_contratacion_temporal
    .confirmar_operacion_decision_cobertura_o404e_v1(:'carga'::jsonb)
\gset
SELECT vec_o404e_prueba.guardar_recibo(:'recibo'::jsonb)
  AS recibo_guardado \gset
SELECT (:'recibo'::jsonb->>'denegada_vec')::boolean AS denegada \gset
\if :recibo_guardado
\else
  SELECT 1/0;
\endif
\if :denegada
\else
  SELECT 1/0;
\endif
COMMIT;
\endif

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT recibo_json=:'recibo'::jsonb AS replay_exacto
  FROM vec_contratacion_temporal
    .confirmar_operacion_decision_cobertura_o404e_v1(
      vec_o404e_prueba.carga()
    )
\gset
\if :replay_exacto
\else
  SELECT 1/0;
\endif
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $colision$
BEGIN
  PERFORM *
    FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        jsonb_set(
          vec_o404e_prueba.carga(),'{cabecera,huella_orden_sha256}',
          to_jsonb(repeat('b',64))
        )
      );
  RAISE EXCEPTION 'replay divergente aceptado';
EXCEPTION
  WHEN sqlstate '55000' THEN NULL;
END
$colision$;
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $mutante$
BEGIN
  PERFORM *
    FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        jsonb_set(
          vec_o404e_prueba.carga(),
          '{denegacion,atributos,via_elegida}',
          to_jsonb('via_mutada'::text)
        )
      );
  RAISE EXCEPTION 'replay mutado aceptado';
EXCEPTION
  WHEN sqlstate '55000' THEN NULL;
END
$mutante$;
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT resultado_json AS lectura
FROM vec_contratacion_temporal
 .leer_terminal_primario_decision_cobertura_o404e_v1(
  jsonb_build_object(
   'esquema',
    'vec.contratacion-temporal.consulta-primaria-decision-cobertura.o4-04e.v1',
   'organizacion_ref','organizacion:o404e',
   'expediente_ref','expediente:o404e:denegado','version_expediente',2,
   'reserva_ref','reserva:o404e:denegada',
   'recibo_ref','recibo:o404e:denegada',
   'correlacion_vec_ref','correlacion_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee',
   'decision_vec_ref','decision:o404e:denegada','revision_cercado',1,
   'huella_orden_sha256',repeat('d',64)
  )
)
\gset
SELECT (:'lectura'::jsonb->>'encontrado')::boolean AS encontrado \gset
\if :encontrado
\else
  \quit 1
\endif
COMMIT;
RESET SESSION AUTHORIZATION;

DO $invariantes$
BEGIN
  IF (SELECT count(*) FROM vec_contratacion_temporal
       .auditoria_decision_cobertura)<>1
     OR NOT EXISTS(
       SELECT 1 FROM vec_contratacion_temporal.auditoria_decision_cobertura
        WHERE secuencia=1 AND rama='denegada'
          AND anterior_sha256=repeat('0',64))
     OR NOT EXISTS(
       SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
        WHERE secuencia=1
          AND tipo_evento='contratacion_temporal.cobertura_denegada_vec'
          AND anterior_sha256=repeat('0',64))
     OR EXISTS(
       SELECT 1 FROM vec_contratacion_temporal
         .acreditacion_gobierno_decision_cobertura
       UNION ALL
       SELECT 1 FROM vec_contratacion_temporal.consumo_cobertura_lote
       UNION ALL
       SELECT 1 FROM vec_contratacion_temporal
         .decision_cobertura_gobernada_durable
       UNION ALL
       SELECT 1 FROM vec_contratacion_temporal.actuacion_expediente_integral
        WHERE operacion_ref='actuacion:o404e:denegada'
     ) THEN
    RAISE EXCEPTION 'invariantes terminales denegados O4-04E divergentes';
  END IF;
END
$invariantes$;
