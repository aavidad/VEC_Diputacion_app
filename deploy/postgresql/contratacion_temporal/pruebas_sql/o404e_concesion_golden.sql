\set ON_ERROR_STOP on
-- Golden concedido O4-04E: gobierno real, VEC vivo, C1 y transición.
CREATE SCHEMA vec_o404e_concedida;
REVOKE ALL ON SCHEMA vec_o404e_concedida FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_o404e_concedida TO vec_o404e_tcb,vec_o404e_gob;

CREATE TABLE vec_o404e_concedida.cargas(
  caso text PRIMARY KEY,cantidad integer NOT NULL,carga jsonb NOT NULL,
  recibo jsonb
);
REVOKE ALL ON TABLE vec_o404e_concedida.cargas FROM PUBLIC;
CREATE TABLE vec_o404e_concedida.retiro_gobierno(
  carga jsonb NOT NULL,resultado jsonb
);
REVOKE ALL ON TABLE vec_o404e_concedida.retiro_gobierno FROM PUBLIC;
CREATE TABLE vec_o404e_concedida.vectores(linea text NOT NULL);
\copy vec_o404e_concedida.vectores(linea) FROM '/repo/contratacion_temporal/pruebas_sql/o404e_expedientes_validos.jsonl'
CREATE TABLE vec_o404e_concedida.expedientes(
  caso text PRIMARY KEY,agregado jsonb NOT NULL
);
INSERT INTO vec_o404e_concedida.expedientes(caso,agregado)
SELECT linea::jsonb->>'caso',linea::jsonb->'agregado'
  FROM vec_o404e_concedida.vectores;
DROP TABLE vec_o404e_concedida.vectores;
REVOKE ALL ON TABLE vec_o404e_concedida.expedientes FROM PUBLIC;

CREATE FUNCTION vec_o404e_concedida.publicar_gobierno()
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
SET timezone='UTC'
AS $$
DECLARE
  v_ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
  v_desde text:=vec_contratacion_temporal.texto_instante_utc_go_v2(
    (v_ahora-interval '1 minute')::text);
  v_hasta text:=vec_contratacion_temporal.texto_instante_utc_go_v2(
    (v_ahora+interval '2 hours')::text);
  v_publicada text:=vec_contratacion_temporal.texto_instante_utc_go_v2(
    (v_ahora-interval '2 minutes')::text);
  v_catalogo jsonb;
  v_politica jsonb;
  v_actuacion jsonb;
  v_publicacion jsonb;
  v_secuencia bigint;
BEGIN
  IF session_user<>'vec_o404e_gob' THEN
    RAISE EXCEPTION 'publicación de gobierno fuera del fixture';
  END IF;
  v_catalogo:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.catalogo-vias-cobertura',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','catalogo:o404e:golden','version',1,
    'huella_sha256',repeat('f',64),'publicado_en',v_publicada,
    'vigencia',jsonb_build_object('desde',v_desde,'hasta',v_hasta),
    'procedencia_ref','procedencia:o404e:golden',
    'vias',jsonb_build_array(jsonb_build_object(
      'clave','via_o404e','orden',1,
      'comprobaciones',jsonb_build_array(jsonb_build_object(
        'clave','comprobacion_o404e','orden',1,'obligatoria',true,
        'procedencia',jsonb_build_object(
          'clave','fuente_o404e',
          'definicion_fuente_ref','fuente:o404e:golden'))))));
  v_catalogo:=jsonb_set(v_catalogo,'{huella_sha256}',to_jsonb(encode(
    sha256(vec_contratacion_temporal.gobi_o404b_material_catalogo(v_catalogo)),
    'hex')));
  v_politica:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.politica-decision-cobertura',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','politica:o404e:golden','version',1,
    'huella_sha256',repeat('f',64),
    'catalogo',jsonb_build_object(
      'referencia',v_catalogo->>'referencia','version',v_catalogo->'version',
      'huella_sha256',v_catalogo->>'huella_sha256'),
    'organizacion_ref','organizacion:o404e:golden',
    'finalidad_clave','gestionar_cobertura_temporal',
    'finalidad_ref','finalidad:o404e:golden',
    'publicada_en',v_publicada,
    'vigencia',jsonb_build_object('desde',v_desde,'hasta',v_hasta),
    'procedencia_ref','procedencia:o404e:golden',
    'vias',jsonb_build_array(jsonb_build_object(
      'via_clave','via_o404e','prioridad',1,
      'comprobaciones',jsonb_build_array(jsonb_build_object(
        'clave','comprobacion_o404e',
        'resultados_habilitantes',jsonb_build_array('afirmativa'),
        'tratamiento_ausencia','bloquea')))));
  v_politica:=jsonb_set(v_politica,'{huella_sha256}',to_jsonb(encode(
    sha256(vec_contratacion_temporal.gobi_o404b_material_politica(v_politica)),
    'hex')));
  v_actuacion:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.politica-actuacion-cobertura',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','actuacion:o404e:gobierno:golden','version',1,
    'huella_sha256',repeat('f',64),
    'organizacion_ref','organizacion:o404e:golden',
    'accion','contratacion_temporal.cobertura.decidir',
    'catalogo',v_politica->'catalogo',
    'politica',jsonb_build_object(
      'referencia',v_politica->>'referencia','version',v_politica->'version',
      'huella_sha256',v_politica->>'huella_sha256'),
    'finalidad_contratacion_clave','gestionar_cobertura_temporal',
    'finalidad_contratacion_ref','finalidad:o404e:golden',
    'finalidad_autorizacion_vec','gestion',
    'unidad_ejecutora_ref','unidad:o404e:golden',
    'fase_destino','fase_cobertura','estado_destino','en_curso',
    'motivo_autorizacion_decidir',jsonb_build_object(
      'catalogo_id','motivos_v3','catalogo_version',1,
      'catalogo_huella_sha256',repeat('9',64),
      'entrada_clave','motivo_33333333333333333333333333333333'),
    'motivo_autorizacion_rectificar',jsonb_build_object(
      'catalogo_id','motivos_v3','catalogo_version',1,
      'catalogo_huella_sha256',repeat('9',64),
      'entrada_clave','motivo_44444444444444444444444444444444'),
    'publicada_en',v_publicada,
    'vigencia',jsonb_build_object('desde',v_desde,'hasta',v_hasta));
  v_actuacion:=jsonb_set(v_actuacion,'{huella_sha256}',to_jsonb(encode(
    sha256(vec_contratacion_temporal.gobi_o404b_material_actuacion(v_actuacion)),
    'hex')));
  SELECT ultima_secuencia+1 INTO STRICT v_secuencia
    FROM vec_contratacion_temporal.gobi_o404b_checkpoint WHERE control;
  v_publicacion:=jsonb_build_object(
    'esquema','vec.contratacion-temporal.gobierno-cobertura.o4-04b.v1',
    'secuencia',v_secuencia,'evento_ref','evento_gobi_o404b_'||
      lpad(to_hex(v_secuencia),32,'e'),
    'catalogo',v_catalogo,'politica',v_politica,'actuacion',v_actuacion);
  PERFORM * FROM vec_contratacion_temporal.gobi_o404b_publicar(v_publicacion);
  RETURN true;
END
$$;
REVOKE ALL ON FUNCTION vec_o404e_concedida.publicar_gobierno() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_o404e_concedida.publicar_gobierno()
  TO vec_o404e_gob;

CREATE FUNCTION vec_o404e_concedida.retirar_gobierno()
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
SET timezone='UTC'
AS $$
DECLARE
  v_carga jsonb;
  v_resultado record;
  v_secuencia bigint;
BEGIN
  IF session_user<>'vec_o404e_gob' THEN
    RAISE EXCEPTION 'retirada O4-04E fuera de contrato';
  END IF;
  SELECT carga INTO v_carga FROM vec_o404e_concedida.retiro_gobierno;
  IF v_carga IS NULL THEN
    SELECT ultima_secuencia+1 INTO STRICT v_secuencia
      FROM vec_contratacion_temporal.gobi_o404b_checkpoint
     WHERE control;
    SELECT jsonb_build_object(
      'esquema',
        'vec.contratacion-temporal.retirar-gobierno-cobertura.o4-04b.v1',
      'secuencia',v_secuencia,
      'evento_ref','evento_gobi_o404b_'||lpad(to_hex(v_secuencia),32,'f'),
      'organizacion_ref',x.organizacion_ref,'accion',x.accion,
      'actuacion_ref',x.actuacion_ref,
      'actuacion_version',x.actuacion_version,
      'actuacion_huella_sha256',x.actuacion_huella_sha256,
      'retirada_en',
        vec_contratacion_temporal.texto_instante_utc_go_v2(
          date_trunc('microseconds',clock_timestamp())::text)
    ) INTO STRICT v_carga
    FROM vec_contratacion_temporal.gobi_o404b_actual x
    WHERE x.organizacion_ref='organizacion:o404e:golden'
      AND x.accion='contratacion_temporal.cobertura.decidir';
    INSERT INTO vec_o404e_concedida.retiro_gobierno VALUES(v_carga,NULL);
  END IF;
  SELECT * INTO STRICT v_resultado
    FROM vec_contratacion_temporal.gobi_o404b_retirar(v_carga);
  UPDATE vec_o404e_concedida.retiro_gobierno
     SET resultado=jsonb_build_object(
       'resultado',v_resultado.resultado,
       'evento_ref',v_resultado.evento_ref,
       'huella_evento_sha256',v_resultado.huella_evento_sha256);
  RETURN (SELECT resultado FROM vec_o404e_concedida.retiro_gobierno);
END
$$;
REVOKE ALL ON FUNCTION vec_o404e_concedida.retirar_gobierno() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_o404e_concedida.retirar_gobierno()
  TO vec_o404e_gob;

SET SESSION AUTHORIZATION vec_o404e_gob;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_concedida.publicar_gobierno();
COMMIT;
RESET SESSION AUTHORIZATION;

BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $autoridad_ct$
DECLARE
  v_ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
  v_publicada timestamptz(6):=v_ahora-interval '1 minute';
  v_hasta timestamptz(6):=v_ahora+interval '30 minutes';
  v_publicada_z text:=to_char(
    v_publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
  v_hasta_z text:=to_char(
    v_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
  v_rol jsonb;
  v_control jsonb;
  v_asignacion jsonb;
BEGIN
  v_rol:=jsonb_build_object(
    'rol_id','o404e_ct','version',1,'nombre','Confirmación CT O4-04E',
    'estado','publicada',
    'concesiones',jsonb_build_array(jsonb_build_object(
      'accion','contratacion_temporal.cobertura.decidir',
      'modulo_id','contratacion_temporal',
      'tipo_recurso','decision_cobertura_gobernada',
      'finalidades',jsonb_build_array('gestion'),
      'garantia_minima','alto',
      'campos_permitidos',jsonb_build_array('estado'),
      'obligaciones',jsonb_build_array('auditar'))),
    'publicada_por','fixture_o404e','publicada_en',v_publicada_z);
  INSERT INTO vec_autorizacion.version_rol(
    version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
  VALUES('rol:o404e_ct:v1','o404e_ct',1,repeat('b',64),v_publicada,v_rol);
  v_control:=jsonb_build_object(
    'version_rol_ref','rol:o404e_ct:v1','revision',1,
    'estado','habilitada','actualizado_por','fixture_o404e',
    'actualizado_en',v_publicada_z);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol(
    version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
  VALUES('rol:o404e_ct:v1',1,'habilitada',repeat('c',64),
    v_publicada,v_control);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
  VALUES('rol:o404e_ct:v1',1,v_ahora,'fixture_o404e',
    'acto:control:o404e');
  v_asignacion:=jsonb_build_object(
    'asignacion_id','registro_v3','version',2,
    'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
    'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'version_rol_ref','rol:o404e_ct:v1','estado','activa',
    'emitida_en',v_publicada_z,'vigente_desde',v_publicada_z,
    'vigente_hasta',v_hasta_z,
    'ambitos',jsonb_build_array(
      jsonb_build_object('clave','organizacion_ref',
        'valores',jsonb_build_array('organizacion:o404e:golden')),
      jsonb_build_object('clave','unidad_ejecutora_ref',
        'valores',jsonb_build_array('unidad:o404e:golden'))));
  INSERT INTO vec_autorizacion.asignacion_perfil(
    asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
    version_rol_ref,huella_sha256,emitida_en,documento)
  VALUES('asignacion:registro_v3:v2','registro_v3',2,
    'prf_sintetico_cccccccccccccccccccccccc',
    'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'rol:o404e_ct:v1',repeat('d',64),v_publicada,v_asignacion);
  UPDATE vec_autorizacion.asignacion_perfil_actual
     SET asignacion_ref='asignacion:registro_v3:v2',
         actualizada_en=v_ahora,actualizada_por='fixture_o404e',
         acto_ref='acto:asignacion:o404e'
   WHERE perfil_activo_ref='prf_sintetico_cccccccccccccccccccccccc'
     AND asignacion_ref='asignacion:registro_v3:v1';
  IF NOT FOUND THEN RAISE EXCEPTION 'asignación CT no avanzó'; END IF;
END
$autoridad_ct$;
COMMIT;

CREATE FUNCTION vec_o404e_concedida.carga(p_caso text)
RETURNS jsonb
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$ SELECT carga FROM vec_o404e_concedida.cargas WHERE caso=p_caso $$;
REVOKE ALL ON FUNCTION vec_o404e_concedida.carga(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_o404e_concedida.carga(text) TO vec_o404e_tcb;

CREATE FUNCTION vec_o404e_concedida.recibo(p_caso text)
RETURNS jsonb
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$ SELECT recibo FROM vec_o404e_concedida.cargas WHERE caso=p_caso $$;
CREATE FUNCTION vec_o404e_concedida.guardar_recibo(
  p_caso text,p_recibo jsonb
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
AS $$
BEGIN
  IF session_user<>'vec_o404e_tcb' OR p_recibo IS NULL THEN
    RAISE EXCEPTION 'recibo golden fuera de contrato';
  END IF;
  UPDATE vec_o404e_concedida.cargas SET recibo=p_recibo
   WHERE caso=p_caso AND recibo IS NULL;
  RETURN FOUND;
END
$$;
REVOKE ALL ON FUNCTION vec_o404e_concedida.recibo(text),
  vec_o404e_concedida.guardar_recibo(text,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_o404e_concedida.recibo(text),
  vec_o404e_concedida.guardar_recibo(text,jsonb) TO vec_o404e_tcb;

CREATE FUNCTION vec_o404e_concedida.preparar_ortogonal(
  p_caso text,p_cantidad integer,p_expediente_caso text,p_evidencia_caso text
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
  v_propiedad timestamptz(6):=v_ahora+interval '5 seconds';
  v_hasta timestamptz(6):=v_ahora+interval '4.8 seconds';
  v_id text:=p_caso;
  v_expediente_id text:=coalesce(p_expediente_caso,p_caso);
  v_evidencia_id text:=coalesce(p_evidencia_caso,p_caso);
  v_vector text:=CASE WHEN p_cantidad=512 THEN 'maximo' ELSE 'uno' END;
  v_ambito text:='hmac-sha256:vec.contratacion-temporal.cobertura-decision.ambito/v1:'||
    encode(sha256(convert_to('ambito:'||p_caso,'UTF8')),'hex');
  v_semantica text:='hmac-sha256:vec.contratacion-temporal.cobertura-decision.semantica/v1:'||
    encode(sha256(convert_to('semantica:'||p_caso,'UTF8')),'hex');
  v_orden text:=encode(sha256(convert_to('orden:'||p_caso,'UTF8')),'hex');
  v_token text:=encode(sha256(convert_to('token:'||p_caso,'UTF8')),'hex');
  v_analisis text:=encode(sha256(convert_to('analisis:'||p_caso,'UTF8')),'hex');
  v_preparacion text:=encode(
    sha256(convert_to('preparacion:'||v_evidencia_id,'UTF8')),'hex');
  v_correlacion text:='correlacion_'||substr(
    encode(sha256(convert_to('correlacion:'||p_caso,'UTF8')),'hex'),1,32);
  v_catalogo jsonb;
  v_politica jsonb;
  v_actuacion_gob jsonb;
  v_gobierno jsonb;
  v_propuesta jsonb;
  v_decision_ct jsonb;
  v_actuacion jsonb;
  v_anterior jsonb;
  v_siguiente jsonb;
  v_motivo_funcional jsonb;
  v_consumos jsonb:='[]'::jsonb;
  v_consumo jsonb;
  v_ambitos jsonb;
  v_atributos jsonb;
  v_contexto_huella text;
  v_motivo bytea;
  v_decision_vec jsonb;
  v_decision_canon bytea;
  v_vinculo jsonb;
  v_contexto record;
  v_sesion record;
  v_cabecera jsonb;
  v_concesion jsonb;
  v_carga jsonb;
  v_lote jsonb;
  v_semantica_propuesta text;
  i integer;
  v_t text;
BEGIN
  IF session_user<>'vec_o404e_tcb'
     OR p_caso!~'^[a-z][a-z0-9_]{1,20}$'
     OR v_expediente_id!~'^[a-z][a-z0-9_]{1,20}$'
     OR v_evidencia_id!~'^[a-z][a-z0-9_]{1,20}$'
     OR p_cantidad NOT BETWEEN 1 AND 512
     OR EXISTS(SELECT 1 FROM vec_o404e_concedida.cargas WHERE caso=p_caso)
  THEN RAISE EXCEPTION 'preparación concedida fuera de contrato'; END IF;
  SELECT ca.publicacion_json,pd.publicacion_json,pa.publicacion_json
    INTO STRICT v_catalogo,v_politica,v_actuacion_gob
    FROM vec_contratacion_temporal.gobi_o404b_actual ga
    JOIN vec_contratacion_temporal.gobi_o404b_actuacion pa
      ON pa.referencia=ga.actuacion_ref AND pa.version=ga.actuacion_version
    JOIN vec_contratacion_temporal.gobi_o404b_politica pd
      ON pd.referencia=pa.politica_ref AND pd.version=pa.politica_version
    JOIN vec_contratacion_temporal.gobi_o404b_catalogo ca
      ON ca.referencia=pa.catalogo_ref AND ca.version=pa.catalogo_version
   WHERE ga.organizacion_ref='organizacion:o404e:golden'
     AND ga.accion='contratacion_temporal.cobertura.decidir';
  v_gobierno:=jsonb_build_object(
    'catalogo',v_catalogo,'politica',v_politica,
    'politica_actuacion',v_actuacion_gob,
    'accion','contratacion_temporal.cobertura.decidir',
    'finalidad_ct_clave','gestionar_cobertura_temporal',
    'finalidad_ct_ref','finalidad:o404e:golden','finalidad_vec','gestion',
    'unidad_ejecutora_ref','unidad:o404e:golden',
    'fase_destino','fase_cobertura','estado_destino','en_curso',
    'motivo_autorizacion',v_actuacion_gob->'motivo_autorizacion_decidir',
    'evaluada_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'valida_hasta',vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text));
  v_propuesta:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.propuesta-decision-cobertura',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','propuesta-cobertura:sha256:'||repeat('f',64),
    'huella_sha256',repeat('f',64),
    'organizacion_ref','organizacion:o404e:golden',
    'expediente_ref','expediente:o404e:'||v_expediente_id,
    'version_expediente',2,
    'analisis_ref','analisis:o404e:'||v_id,'analisis_huella_sha256',v_analisis,
    'preparacion_evidencias_ref','preparacion:o404e:'||v_evidencia_id,
    'preparacion_evidencias_huella_sha256',v_preparacion,
    'catalogo',v_politica->'catalogo',
    'politica',jsonb_build_object(
      'referencia',v_politica->>'referencia','version',v_politica->'version',
      'huella_sha256',v_politica->>'huella_sha256'),
    'finalidad_clave','gestionar_cobertura_temporal',
    'finalidad_ref','finalidad:o404e:golden',
    'categoria_ref','categoria:o404e:golden',
    'periodo',jsonb_build_object(
      'inicio','2026-01-01T00:00:00Z','fin','2026-12-31T00:00:00Z'),
    'generada_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'valida_hasta',vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text),
    'estado','viable','via_propuesta','via_o404e',
    'resultados',jsonb_build_array(jsonb_build_object(
      'clave','comprobacion_o404e','evidencias',jsonb_build_array(
        jsonb_build_object('resultado','afirmativa',
          'fuente_ref','fuente:o404e:golden',
          'recibo_ref','respuesta:o404e:'||v_evidencia_id||':1',
          'evaluada_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text))))),
    'evaluaciones',jsonb_build_array(jsonb_build_object(
      'via_clave','via_o404e','prioridad',1,'estado','viable')));
  v_t:=encode(sha256(vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_propuesta)),'hex');
  v_propuesta:=jsonb_set(jsonb_set(v_propuesta,'{huella_sha256}',to_jsonb(v_t)),
    '{referencia}',to_jsonb('propuesta-cobertura:sha256:'||v_t));
  v_motivo_funcional:=jsonb_build_object(
    'referencia_catalogo',jsonb_build_object(
      'catalogo_id','','catalogo_version',0,
      'catalogo_huella_sha256','','entrada_clave',''),
    'clave_i18n','');
  v_actuacion:=jsonb_build_object(
    'secuencia',3,'version_expediente',3,
    'accion_clave','contratacion_temporal.cobertura.decidir',
    'actor_ref','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'unidad_ref','unidad:o404e:golden',
    'realizada_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'fase_origen','analisis_rrhh','fase_destino','fase_cobertura',
    'estado_origen','en_curso','estado_destino','en_curso',
    'recibo_ref','recibo:o404e:'||v_id);
  v_decision_ct:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.decision-cobertura-gobernada',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','decision-cobertura:sha256:'||repeat('f',64),
    'huella_sha256',repeat('f',64),'tipo','inicial',
    'organizacion_ref','organizacion:o404e:golden',
    'expediente_ref','expediente:o404e:'||v_expediente_id,
    'version_expediente_origen',2,'version_expediente',3,
    'actor_ref','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'perfil_ref','prf_sintetico_cccccccccccccccccccccccc',
    'propuesta_ref',v_propuesta->>'referencia',
    'propuesta_huella_sha256',v_propuesta->>'huella_sha256',
    'preparacion_evidencias_ref','preparacion:o404e:'||v_evidencia_id,
    'preparacion_evidencias_huella_sha256',v_preparacion,
    'analisis_ref','analisis:o404e:'||v_id,
    'analisis_huella_sha256',v_analisis,
    'catalogo',v_politica->'catalogo',
    'politica',jsonb_build_object(
      'referencia',v_politica->>'referencia','version',v_politica->'version',
      'huella_sha256',v_politica->>'huella_sha256'),
    'via_elegida','via_o404e','via_recomendada','via_o404e',
    'motivo',jsonb_build_object(
      'referencia_catalogo',jsonb_build_object(
        'catalogo_id','','catalogo_version',0,
        'catalogo_huella_sha256','','entrada_clave',''),
      'clave_i18n',''),
    'decidida_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'actuacion',v_actuacion);
  v_t:=encode(sha256(vec_contratacion_temporal
    .o404e_material_decision_cobertura_v1(v_decision_ct)),'hex');
  v_decision_ct:=jsonb_set(jsonb_set(v_decision_ct,'{huella_sha256}',to_jsonb(v_t)),
    '{referencia}',to_jsonb('decision-cobertura:sha256:'||v_t));
  SELECT agregado INTO STRICT v_anterior
    FROM vec_o404e_concedida.expedientes WHERE caso=v_vector;
  v_siguiente:=v_anterior||jsonb_build_object(
    'version',3,'actualizado_en',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'fase_actual','fase_cobertura','estado_actual','en_curso',
    'via_cobertura',jsonb_build_object(
      'via_clave','via_o404e','decision_gobernada',v_decision_ct),
    'decisiones_cobertura',jsonb_build_array(v_decision_ct),
    'actuaciones',v_anterior->'actuaciones'||
      jsonb_build_array(v_actuacion));
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       v_anterior,v_siguiente,v_propuesta,v_motivo_funcional,v_ahora)
       IS NOT TRUE THEN RAISE EXCEPTION 'transición golden inválida'; END IF;
  FOR i IN 1..p_cantidad LOOP
    v_consumo:=jsonb_build_object(
      'posicion',i,'total',p_cantidad,
      'peticion_ref','peticion:o404e:'||v_evidencia_id||':'||i,
      'organizacion_ref','organizacion:o404e:golden',
      'expediente_ref','expediente:o404e:'||v_expediente_id,
      'version_expediente',2,
      'catalogo_ref',v_catalogo->>'referencia',
      'catalogo_version',v_catalogo->'version',
      'catalogo_huella_sha256',v_catalogo->>'huella_sha256',
      'via_clave','via_o404e','comprobacion_clave','comprobacion_o404e',
      'comprobacion_resultado','afirmativa',
      'comprobacion_fuente_ref','fuente:o404e:golden',
      'comprobacion_recibo_ref',
        'respuesta:o404e:'||v_evidencia_id||':'||i,
      'comprobacion_evaluada_en',
        vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
      'orden_comprobacion',1,'obligatoria',true,
      'procedencia_clave','fuente_o404e',
      'definicion_fuente_ref','fuente:o404e:golden',
      'categoria_ref','categoria:o404e:golden',
      'periodo',jsonb_build_object(
        'inicio','2026-01-01T00:00:00Z','fin','2026-12-31T00:00:00Z'),
      'solicitada_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
      'emitida_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
      'valida_hasta',vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text),
      'huella_peticion_sha256',encode(
        sha256(convert_to('p'||v_evidencia_id||i,'UTF8')),'hex'),
      'huella_resultado_sha256',encode(
        sha256(convert_to('r'||v_evidencia_id||i,'UTF8')),'hex'),
      'huella_respuesta_sha256',encode(
        sha256(convert_to('s'||v_evidencia_id||i,'UTF8')),'hex'),
      'autoridad_ref','autoridad:o404e:golden','generacion',i,
      'recibo_respuesta_ref','respuesta:o404e:'||v_evidencia_id||':'||i,
      'verificador_ref','verificador:o404e:golden',
      'publicador_catalogo_ref','publicador:o404e:golden',
      'pruebas_canonicas',jsonb_build_object(
        'peticion_hex','01','resultado_hex','02','atestacion_hex','03',
        'confirmacion_tcb_hex','04','catalogo_hex','05',
        'verificador_hex','06','resumen_hex','07'));
    v_consumos:=v_consumos||jsonb_build_array(v_consumo);
  END LOOP;
  v_cabecera:=jsonb_build_object(
    'esquema_sesion','VEC-CT-SESION-TCB-OPERACION-DECISION-COBERTURA-V1',
    'huella_orden_sha256',v_orden,
    'organizacion_ref','organizacion:o404e:golden',
    'expediente_ref','expediente:o404e:'||v_expediente_id,
    'version_expediente',2,
    'reserva_ref','reserva:o404e:'||v_id,'recibo_ref','recibo:o404e:'||v_id,
    'actuacion_ref','actuacion:o404e:'||v_id,
    'auditoria_ref','auditoria:o404e:'||v_id,'evento_ref','evento:o404e:'||v_id,
    'correlacion_vec_ref',v_correlacion,
    'decision_vec_ref','decision-vec:o404e:'||v_id,
    'analisis_ref','analisis:o404e:'||v_id,'analisis_huella_sha256',v_analisis,
    'token_propietario_sha256',v_token,
    'ambito_idempotencia_hmac',v_ambito,'huella_semantica_hmac',v_semantica,
    'revision_cercado_anterior',0,'revision_cercado',1,
    'observada_en_db',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'propiedad_hasta',vec_contratacion_temporal.texto_instante_utc_go_v2(v_propiedad::text),
    'valida_hasta_orden',vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text),
    'preparacion_c1_ref','preparacion:o404e:'||v_evidencia_id,
    'preparacion_c1_huella_sha256',v_preparacion,
    'preparacion_c1_preparada_en',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'preparacion_c1_valida_hasta',
      vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text),
    'numero_consumos_c1',p_cantidad,
    'huella_ordenes_consumo_c1_sha256',
      vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(v_consumos));
  v_concesion:=jsonb_build_object(
    'agregado_anterior',v_anterior,'agregado_siguiente',v_siguiente,
    'propuesta',v_propuesta,'motivo_funcional',v_motivo_funcional,
    'efecto_en',vec_contratacion_temporal.texto_instante_utc_go_v2(v_ahora::text),
    'valida_hasta',vec_contratacion_temporal.texto_instante_utc_go_v2(v_hasta::text));
  v_ambitos:=jsonb_build_object(
    'organizacion_ref','organizacion:o404e:golden',
    'unidad_ejecutora_ref','unidad:o404e:golden');
  v_semantica_propuesta:=
    vec_contratacion_temporal.o404e_semantica_propuesta_v1(v_propuesta);
  v_atributos:=jsonb_build_object(
    'accion','contratacion_temporal.cobertura.decidir',
    'analisis_huella_sha256',v_analisis,'analisis_ref','analisis:o404e:'||v_id,
    'catalogo_huella_sha256',v_catalogo->>'huella_sha256',
    'catalogo_ref',v_catalogo->>'referencia','catalogo_version','1',
    'expediente_ref','expediente:o404e:'||v_expediente_id,
    'politica_actuacion_huella_sha256',v_actuacion_gob->>'huella_sha256',
    'politica_actuacion_ref',v_actuacion_gob->>'referencia',
    'politica_actuacion_version','1',
    'politica_huella_sha256',v_politica->>'huella_sha256',
    'politica_ref',v_politica->>'referencia','politica_version','1',
    'preparacion_evidencias_huella_sha256',v_preparacion,
    'preparacion_evidencias_ref','preparacion:o404e:'||v_evidencia_id,
    'propuesta_huella_sha256',v_propuesta->>'huella_sha256',
    'propuesta_ref',v_propuesta->>'referencia',
    'propuesta_semantica_huella_sha256',v_semantica_propuesta,
    'propuesta_semantica_ref',
      'propuesta-cobertura-semantica:sha256:'||v_semantica_propuesta,
    'reserva_ref','reserva:o404e:'||v_id,'revision_cercado','1',
    'tipo_operacion','inicial','version_expediente_esperada','2',
    'via_elegida','via_o404e');
  v_carga:=jsonb_build_object(
    'esquema',
      'vec.contratacion-temporal.confirmar-operacion-decision-cobertura.o4-04e.v1',
    'rama','concedida','cabecera',v_cabecera,'gobierno',v_gobierno,
    'decision_vec',jsonb_build_object(),'consumos_c1',v_consumos,
    'concesion',v_concesion,'denegacion',NULL);
  v_lote:=vec_contratacion_temporal.o404e_construir_lote_c1_v1(v_carga,v_ambito);
  IF v_lote IS NULL THEN RAISE EXCEPTION 'lote golden inválido'; END IF;
  v_contexto_huella:=encode(sha256(
    vec_contratacion_temporal.o404e_contexto_recurso_concesion_v1(
      jsonb_set(v_carga,'{decision_vec}',jsonb_build_object(
        'accion','contratacion_temporal.cobertura.decidir',
        'ambitos',v_ambitos,'atributos',v_atributos)))),'hex');
  SELECT * INTO STRICT v_contexto
    FROM vec_contexto_actor_v1.registros_contexto
   WHERE registro_contexto_ref='rca_registro_v3_000000000000000000000000';
  SELECT s.autenticacion_verificada_en,s.sesion_emitida_en,
         c.sesion_revalidada_en,c.sesion_valida_hasta
    INTO STRICT v_sesion
    FROM vec_autorizacion.sesion_autenticacion_v1 s
    JOIN vec_autorizacion.control_sesion_actual_v1 a USING(sesion_ref)
    JOIN vec_autorizacion.control_sesion_v1 c
      ON c.sesion_ref=a.sesion_ref AND c.revision=a.revision
   WHERE s.sesion_ref='ses_registro_v3_0000000000000000000000';
  v_motivo:=convert_to(
    '{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada","referencia":{"catalogo_id":"motivos_v3","catalogo_version":1,"catalogo_huella_sha256":"'
    ||repeat('9',64)
    ||'","entrada_clave":"motivo_33333333333333333333333333333333"}}','UTF8');
  v_vinculo:=jsonb_build_object(
    'esquema','vec.autenticacion-actor.vinculo.v2.contexto-registrado',
    'bloque_version',2,
    'autenticacion_ref','aut_registro_v3_0000000000000000000000',
    'autenticacion_huella_sha256',repeat('5',64),
    'asercion_ref','ase_registro_v3_0000000000000000000000',
    'sesion_ref','ses_registro_v3_0000000000000000000000',
    'control_sesion_ref','cse_registro_v3_0000000000000000000000',
    'control_sesion_revision',1,'control_sesion_huella_sha256',repeat('7',64),
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
    'autoridad_efectiva','autoridad_maestra_acreditada');
  v_decision_vec:=jsonb_build_object(
    'esquema','vec.autorizacion.decision.v3.solicitud-ligada.actor-v2',
    'bloque_version',3,'decision_ref','decision-vec:o404e:'||v_id,
    'concedida',true,'codigo','concedida',
    'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
    'accion','contratacion_temporal.cobertura.decidir',
    'recurso_ref','reserva:o404e:'||v_id,
    'modulo_id','contratacion_temporal',
    'tipo_recurso','decision_cobertura_gobernada',
    'contexto_recurso_huella_sha256',v_contexto_huella,'finalidad','gestion',
    'correlacion_ref',v_correlacion,
    'esquema_huella_solicitud',
      'vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2',
    'solicitud_huella_sha256',encode(sha256(convert_to('solicitud:'||v_id,'UTF8')),'hex'),
    'esquema_huella_motivo',
      'vec.autorizacion.motivo.v2.referencia-opaca-catalogada',
    'motivo_huella_sha256',encode(sha256(v_motivo),'hex'),
    'vinculo_autenticacion_actor',v_vinculo,
    'asignacion_ref','asignacion:registro_v3:v2',
    'asignacion_huella_sha256',repeat('d',64),
    'version_rol_ref','rol:o404e_ct:v1',
    'version_rol_huella_sha256',repeat('b',64),
    'control_vigencia_version_rol_ref','rol:o404e_ct:v1',
    'control_vigencia_version_rol_revision',1,
    'control_vigencia_version_rol_huella_sha256',repeat('c',64),
    'revision_catalogo_politicas',1,
    'catalogo_politicas_huella_sha256',
      '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945',
    'politicas_evaluadas',jsonb_build_array(),
    'politicas_aplicables',jsonb_build_array(),'garantia_minima','alto',
    'campos_permitidos',jsonb_build_array('estado'),
    'obligaciones',jsonb_build_array('auditar'),
    'emitida_en',to_char(v_ahora AT TIME ZONE 'UTC',
      'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'valida_hasta',to_char(v_hasta AT TIME ZONE 'UTC',
      'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
  IF v_decision_vec->>'emitida_en'!~'[.][0-9]{6}Z$'
     OR v_decision_vec->>'valida_hasta'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'autenticacion_verificada_en'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'sesion_emitida_en'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'sesion_valida_hasta'!~'[.][0-9]{6}Z$'
     OR v_vinculo->>'sesion_revalidada_en'!~'[.][0-9]{6}Z$' THEN
    RAISE EXCEPTION 'instante VEC concedido sin seis microsegundos';
  END IF;
  v_decision_canon:=vec_autorizacion.decision_contexto_actor_v3_canonica(
    v_decision_vec);
  IF v_decision_canon IS NULL THEN RAISE EXCEPTION 'VEC golden no canónico'; END IF;
  v_carga:=jsonb_set(v_carga,'{decision_vec}',jsonb_build_object(
    'decision_canonica_hex',encode(v_decision_canon,'hex'),
    'motivo_canonico_hex',encode(v_motivo,'hex'),
    'persona_version',2,'perfil_version',2,
    'decision_ref','decision-vec:o404e:'||v_id,
    'decision_huella_sha256',encode(sha256(v_decision_canon),'hex'),
    'codigo_probatorio','concedida','concedida',true,
    'emitida_en',to_char(v_ahora AT TIME ZONE 'UTC',
      'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'valida_hasta',to_char(v_hasta AT TIME ZONE 'UTC',
      'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'principal_id','per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'perfil_activo_ref','prf_sintetico_cccccccccccccccccccccccc',
    'accion','contratacion_temporal.cobertura.decidir',
    'recurso_ref','reserva:o404e:'||v_id,
    'recurso_modulo','contratacion_temporal',
    'recurso_tipo','decision_cobertura_gobernada',
    'ambitos',v_ambitos,'atributos',v_atributos,
    'contexto_recurso_huella_sha256',v_contexto_huella,
    'finalidad','gestion','correlacion_ref',v_correlacion));
  PERFORM set_config('session_replication_role','replica',true);
  IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal
      .expediente_integral_actual
      WHERE expediente_ref='expediente:o404e:'||v_expediente_id) THEN
   INSERT INTO vec_contratacion_temporal.expediente_alta(
    expediente_ref,reserva_ref,numero_visible,organizacion_ref,actor_ref,
    perfil_ref,decision_ref,efecto_ref,huella_efecto_sha256,creada_en,
    confirmacion_ref)
   VALUES('expediente:o404e:'||v_expediente_id,
    'reserva-alta:o404e:'||v_expediente_id,
    '2026/'||v_expediente_id,'organizacion:o404e:golden',
    'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb',
    'prf_sintetico_cccccccccccccccccccccccc',
    'decision-alta:o404e:'||v_expediente_id,
    'efecto-alta:o404e:'||v_expediente_id,repeat('e',64),
    (v_anterior->>'creado_en')::timestamptz,'cnf_ct_'||substr(
      encode(sha256(convert_to('alta:'||v_expediente_id,'UTF8')),'hex'),1,32));
   INSERT INTO vec_contratacion_temporal.expediente_version_integral(
    expediente_ref,version,agregado_json,agregado_json_huella_sha256,
    prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,
    flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,
    registrada_en)
   VALUES('expediente:o404e:'||v_expediente_id,2,v_anterior,
    encode(sha256(convert_to(v_anterior::text,'UTF8')),'hex'),
    convert_to(repeat('prueba-'||v_expediente_id,32),'UTF8'),
    encode(sha256(convert_to(
      repeat('prueba-'||v_expediente_id,32),'UTF8')),'hex'),
    v_anterior#>>'{flujo,definicion_ref}',
    (v_anterior#>>'{flujo,version}')::numeric,
    v_anterior#>>'{flujo,huella_sha256}','analisis_rrhh','en_curso',
    'analisis_o3','operacion:o404e:base:'||v_expediente_id,
    (v_anterior->>'actualizado_en')::timestamptz);
   INSERT INTO vec_contratacion_temporal.expediente_integral_actual
    VALUES('expediente:o404e:'||v_expediente_id,2,
      (v_anterior->>'actualizado_en')::timestamptz,
      'operacion:o404e:base:'||v_expediente_id);
  END IF;
  INSERT INTO vec_contratacion_temporal.reserva_operacion_decision_cobertura
    VALUES(v_ambito,'reserva:o404e:'||v_id,'recibo:o404e:'||v_id,
      'actuacion:o404e:'||v_id,'auditoria:o404e:'||v_id,'evento:o404e:'||v_id,
      v_correlacion,'decision-vec:o404e:'||v_id,
      'organizacion:o404e:golden',
      'expediente:o404e:'||v_expediente_id,2,
      'analisis:o404e:'||v_id,v_analisis,v_semantica,v_ahora);
  INSERT INTO vec_contratacion_temporal.alias_operacion_decision_cobertura
    VALUES(v_ambito,v_ambito,1,v_semantica,v_ahora);
  INSERT INTO vec_contratacion_temporal
    .reserva_operacion_decision_cobertura_version
    VALUES(v_ambito,1,'reservada',1,v_token,v_ahora,v_propiedad,NULL,NULL);
  INSERT INTO vec_contratacion_temporal
    .reserva_operacion_decision_cobertura_actual VALUES(v_ambito,1);
  PERFORM set_config('session_replication_role','origin',true);
  INSERT INTO vec_o404e_concedida.cargas VALUES(p_caso,p_cantidad,v_carga,NULL);
  RETURN v_carga;
END
$$;
REVOKE ALL ON FUNCTION
  vec_o404e_concedida.preparar_ortogonal(text,integer,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
  vec_o404e_concedida.preparar_ortogonal(text,integer,text,text)
  TO vec_o404e_tcb;
