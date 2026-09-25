\set ON_ERROR_STOP on
-- CT120 sobre la estructura real restaurada, después de las pruebas de CT115
-- y CT116 (reutiliza su rol de ejecución y el expediente B, que CT116 dejó en
-- fiscalización con el análisis modificado, v8). La fachada AD3-10 de la
-- fiscalización se sustituye por un doble explícito: prueba la transacción
-- CT, no la criptografía V3.
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(
 p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,p_contexto_actor_canonico bytea,p_persona_version numeric,p_perfil_version numeric,p_payload_vec_ad_3 bytea,p_sobre_cose_sign1 bytea,p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision_canonica,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.fiscalizacion.registrar' THEN RAISE EXCEPTION 'doble: fiscalización denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad_canonica||p_decision_canonica),'hex'),
   'aud_v3_'||md5(p_capacidad_canonica||p_decision_canonica), clock_timestamp(), true;
END $f$;

CREATE FUNCTION pg_temp.exigir(c boolean, msg text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF c IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', msg; END IF; RETURN 'ok'; END $$;

-- Entradas de preparación y confirmación tal como las compone el adaptador Go.
CREATE FUNCTION pg_temp.entrada_fiscalizacion(p_exp text, p_resultado text, p_obs text, p_sufijo text, p_fase_siguiente text DEFAULT NULL)
RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb; v numeric; inst text; org text; amb_hmac text; hue_hmac text; refs jsonb; unidad text := 'unidad:intervencion:desarrollo';
 fase text; estado text; act jsonb; vin jsonb; fis jsonb; sig jsonb; ctx bytea; h text; dec jsonb; recibo_mod text; retorno jsonb;
BEGIN
 SELECT v2.agregado_json, v2.version INTO STRICT ag, v FROM vec_contratacion_temporal.expediente_integral_actual a
   JOIN vec_contratacion_temporal.expediente_version_integral v2 USING (expediente_ref,version) WHERE a.expediente_ref=p_exp;
 org:=ag->>'organizacion_ref';
 inst:=to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 amb_hmac:='hmac-sha256:vec.contratacion-temporal.fiscalizacion.ambito/v1:'||encode(sha256(convert_to('a'||p_sufijo,'UTF8')),'hex');
 hue_hmac:='hmac-sha256:vec.contratacion-temporal.fiscalizacion.peticion/v1:'||encode(sha256(convert_to('p'||p_sufijo,'UTF8')),'hex');
 refs:=jsonb_build_object('reserva_ref','reserva:ct120:'||p_sufijo,'fiscalizacion_ref','fiscalizacion:ct120:'||p_sufijo,
   'recibo_ref','recibo:ct120:'||p_sufijo,'evento_ref','evento:ct120:'||p_sufijo,
   'retorno_ref',CASE WHEN p_resultado='desfavorable' THEN 'retorno:ct120:'||p_sufijo ELSE '' END);
 recibo_mod:=(ag->'actuaciones')->-1->>'recibo_ref';
 IF p_resultado='desfavorable' THEN fase:='subsanacion_unidad'; estado:='incidencia'; ELSE fase:='nombramiento'; estado:='en_curso'; END IF;
 fase:=coalesce(p_fase_siguiente,fase);
 act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',v+1,
   'accion_clave','contratacion_temporal.fiscalizacion.registrar','actor_ref','per_ct120_interventor','unidad_ref',unidad,
   'recibo_ref',refs->>'recibo_ref','realizada_en',inst,'fase_origen','fiscalizacion','fase_destino',fase,'estado_origen','en_curso',
   'estado_destino',estado,'observaciones',p_obs,'documentos_ref',jsonb_build_array(ag#>>'{informe_juridico,documento_ref}'));
 IF p_obs='' THEN act:=act-'observaciones'; END IF;
 vin:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',v+1,
   'accion_clave','contratacion_temporal.fiscalizacion.registrar','fase_destino',fase,'estado_destino',estado,'recibo_ref',refs->>'recibo_ref',
   'fiscalizacion_ref',refs->>'fiscalizacion_ref','resultado',p_resultado,'unidad_fiscalizadora_ref',unidad,
   'informe_juridico_ref',ag#>>'{informe_juridico,informe_ref}','documento_informe_ref',ag#>>'{informe_juridico,documento_ref}');
 IF p_resultado='desfavorable' THEN
  vin:=vin||jsonb_build_object('retorno_ref',refs->>'retorno_ref','unidad_retorno_ref',ag#>>'{asignacion,unidad_ref}','responsable_retorno_ref',ag#>>'{asignacion,responsable_ref}');
  retorno:=jsonb_build_object('retorno_ref',refs->>'retorno_ref','unidad_ref',ag#>>'{asignacion,unidad_ref}','responsable_ref',ag#>>'{asignacion,responsable_ref}',
    'estado','pendiente','creado_en',inst);
 END IF;
 fis:=jsonb_build_object('fiscalizacion_ref',refs->>'fiscalizacion_ref','resultado',p_resultado,'unidad_fiscalizadora_ref',unidad,
   'informe_juridico_ref',ag#>>'{informe_juridico,informe_ref}','documento_informe_ref',ag#>>'{informe_juridico,documento_ref}',
   'observaciones',p_obs,'fiscalizada_en',inst,'actuacion_registro',vin);
 IF p_obs='' THEN fis:=fis-'observaciones'; END IF;
 IF retorno IS NOT NULL THEN fis:=fis||jsonb_build_object('retorno',retorno); END IF;
 sig:=ag||jsonb_build_object('version',v+1,'fase_actual',fase,'estado_actual',estado,'actualizado_en',inst,
   'actuaciones',(ag->'actuaciones')||jsonb_build_array(act),'fiscalizacion',fis);
 ctx:=convert_to('{"ambitos":{"estado_previo":"en_curso","expediente_ref":"'||p_exp||'","fase_previa":"fiscalizacion","organizacion_ref":"'||org
   ||'"},"atributos":{"ambito_idempotencia_hmac":"'||amb_hmac||'","documento_informe_ref":"'||(ag#>>'{informe_juridico,documento_ref}')
   ||'","huella_peticion_hmac":"'||hue_hmac||'","informe_juridico_ref":"'||(ag#>>'{informe_juridico,informe_ref}')
   ||'","modificacion_recibo_ref":"'||recibo_mod||'","observaciones_huella_sha256":"'||encode(sha256(convert_to(p_obs,'UTF8')),'hex')
   ||'","politica_huella_sha256":"'||repeat('f',64)||'","politica_ref":"politica:ct:fiscalizacion:desarrollo","politica_version":"1"'
   ||',"responsable_asignado_ref":"'||(ag#>>'{asignacion,responsable_ref}')||'","resultado":"'||p_resultado
   ||'","unidad_asignada_ref":"'||(ag#>>'{asignacion,unidad_ref}')||'","unidad_fiscalizadora_ref":"'||unidad
   ||'","version_expediente":"'||v::text||'"}}','UTF8');
 h:=encode(sha256(ctx),'hex');
 dec:=jsonb_build_object('decision_ref','decision:ct120:'||p_sufijo,'accion','contratacion_temporal.fiscalizacion.registrar',
   'modulo_id','contratacion_temporal','tipo_recurso','fiscalizacion_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
   'recurso_ref',p_exp,'principal_id','per_ct120_interventor','perfil_activo_ref','prf_ct120','contexto_recurso_huella_sha256',h);
 RETURN jsonb_build_object(
  'preparar',jsonb_build_object('esquema','vec.contratacion-temporal.preparar-fiscalizacion.v1','operacion','registrar_resultado',
    'sellos_hmac',jsonb_build_object('activo',jsonb_build_object('ambito_hmac',amb_hmac,'generacion',1,'huella_peticion_hmac',hue_hmac),'retenidos','[]'::jsonb),
    'organizacion_ref',org,'expediente_ref',p_exp,'version_expediente',v,'actor_ref','per_ct120_interventor','perfil_ref','prf_ct120',
    'resultado',p_resultado,'observaciones',p_obs,'referencias_candidatas',refs),
  'confirmar',jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-fiscalizacion.v1','operacion','registrar_resultado',
    'reserva_ref',refs->>'reserva_ref','referencias',refs,'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac,
    'organizacion_ref',org,'expediente_ref',p_exp,'version_anterior',v,'actor_ref','per_ct120_interventor','perfil_ref','prf_ct120',
    'resultado',p_resultado,'observaciones',p_obs,'expediente_anterior',ag,'expediente_siguiente',sig,'actuacion',act,
    'politica',jsonb_build_object('definicion_ref','politica:ct:fiscalizacion:desarrollo','definicion_version',1,'definicion_huella_sha256',repeat('f',64),
      'accion','contratacion_temporal.fiscalizacion.registrar','finalidad','gestionar_contratacion_temporal','unidad_fiscalizadora_ref',unidad,
      'evaluada_en',inst,'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
    'autorizacion',jsonb_build_object('accion','contratacion_temporal.fiscalizacion.registrar','finalidad','gestionar_contratacion_temporal',
      'recurso_ref',p_exp,'principal_id','per_ct120_interventor','perfil_activo_ref','prf_ct120',
      'decision_canonica_hex',encode(convert_to(dec::text,'UTF8'),'hex'),'motivo_canonico_hex',encode('\x6d6f7469766f'::bytea,'hex'),
      'persona_version',1,'perfil_version',1,'decision_huella_sha256',encode(sha256(convert_to(dec::text,'UTF8')),'hex'),
      'decision_ref','decision:ct120:'||p_sufijo,'contexto_recurso_huella_sha256',h),
    'instante_efecto',inst),
  '_decision',dec);
END $$;
CREATE FUNCTION pg_temp.preparar(e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.preparar_fiscalizacion_v2(e->'preparar');
 RETURN jsonb_build_object('resultado',r.resultado,'estado',r.estado,'recibo',r.recibo_json,'fase',(r.expediente_json::jsonb)->>'fase_actual');
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
CREATE FUNCTION pg_temp.confirmar(e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb;
BEGIN
 SELECT recibo_json INTO STRICT r FROM vec_contratacion_temporal.confirmar_fiscalizacion_v2(e->'confirmar','\x6361706163696461640a',
   convert_to((e->'_decision')::text,'UTF8'),'\x6d6f7469766f','\x01',1,1,'\x01','\x01','\x01','\x01');
 RETURN jsonb_build_object('recibo',r);
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct115_runtime',pg_my_temp_schema()::regnamespace); END $$;

SELECT pg_temp.entrada_fiscalizacion(:'exp_b','favorable','','fav')::text AS fav \gset
SELECT pg_temp.entrada_fiscalizacion(:'exp_b','desfavorable','Coste no justificado','des')::text AS des \gset
SELECT pg_temp.entrada_fiscalizacion(:'exp_b','favorable','','fase_mala','fiscalizacion')::text AS fase_mala \gset
SELECT pg_temp.exigir((SELECT fase_clave||'/'||estado FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=8)='fiscalizacion/en_curso','antecedente: modificación de CT116 en fiscalización');

SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar(:'fav'::jsonb)::text AS prep \gset
COMMIT;
SELECT pg_temp.exigir((:'prep'::jsonb)->>'resultado'='preparada' AND (:'prep'::jsonb)->>'fase'='fiscalizacion','preparación por la fachada v2 '||:'prep');
-- Un expediente que no es una modificación sigue yendo a CT93, que lo rechaza
-- con su propio mensaje (aquí, en subsanación antes de subsanar, v6).
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar(jsonb_set(:'fav'::jsonb,'{preparar,expediente_ref}','"expediente:ct:9511d16dce57e0ebe1baa849a749f3eed2a29ff7722dc1499132d1d536d66253"'))::text AS ajeno \gset
COMMIT;
SELECT pg_temp.exigir((:'ajeno'::jsonb)->>'error' IS NOT NULL AND (:'ajeno'::jsonb)->>'mensaje' NOT LIKE '%modificación%','otro origen va a CT93 '||:'ajeno');

-- Negativos: la proyección de CT93 (quedarse en fiscalización) no vale para
-- una modificación, el contexto autorizado debe citar la modificación y la
-- sesión de lectura no confirma.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar(:'fase_mala'::jsonb)->>'error'='22023','proyección que no vuelve al nombramiento');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'fav'::jsonb,'{confirmar,autorizacion,contexto_recurso_huella_sha256}',to_jsonb(repeat('0',64))))->>'error'='42501','contexto sin el recibo de la modificación');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'fav'::jsonb,'{confirmar,expediente_anterior,version}','7'))->>'error' IN ('40001','22023','42501'),'antecedente alterado');
ROLLBACK;

-- Desfavorable (se revierte): vuelve a la unidad gestora con su retorno.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar(:'des'::jsonb)::text AS conf_des \gset
SELECT pg_temp.exigir((:'conf_des'::jsonb) ? 'recibo','desfavorable confirmado '||:'conf_des');
-- El antecedente restaura el contexto de lectura de CT116 al salir: no
-- queda ligado a este expediente el resto de la transacción.
SELECT pg_temp.exigir(coalesce(current_setting('vec.ct115.organizacion_ref',true),'')=''
    AND coalesce(current_setting('vec.ct115.expediente_ref',true),'')='','contexto de CT116 restaurado');
ROLLBACK;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir(NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=9),'desfavorable revertido sin rastro');
SET SESSION AUTHORIZATION vec_ct115_runtime;

-- Favorable: vuelve al nombramiento con el análisis nuevo.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar(:'fav'::jsonb)::text AS conf \gset
COMMIT;
SELECT pg_temp.exigir((:'conf'::jsonb) ? 'recibo','favorable confirmado '||:'conf');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar(:'fav'::jsonb)::text AS recup \gset
COMMIT;
SELECT pg_temp.exigir((:'recup'::jsonb)->>'resultado'='confirmada' AND (:'recup'::jsonb)->'recibo'=(:'conf'::jsonb)->'recibo','recuperación con el mismo recibo');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar(:'fav'::jsonb)->>'error'='40001','la confirmación repetida exige preparar de nuevo');
ROLLBACK;
RESET SESSION AUTHORIZATION;

SELECT pg_temp.exigir((SELECT fase_clave||'/'||estado FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=9)='nombramiento/en_curso','vuelta al nombramiento');
SELECT pg_temp.exigir((SELECT agregado_json#>>'{fiscalizacion,resultado}' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=9)='favorable'
  AND (SELECT agregado_json#>>'{analisis,porcentaje_jornada}' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=9)='5000','fiscalización vigente y análisis modificado');
SELECT pg_temp.exigir((SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=:'exp_b')=9,'cabeza en v9');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_b' AND version_expediente=9 AND tipo_evento='contratacion_temporal.fiscalizacion_registrada')=1,'evento de fiscalización');
SELECT pg_temp.exigir((SELECT estado FROM vec_contratacion_temporal.reserva_fiscalizacion WHERE reserva_ref='reserva:ct120:fav')='confirmada','reserva confirmada');
SELECT pg_temp.exigir(NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.retorno_fiscalizacion_unidad WHERE retorno_ref='retorno:ct120:des'),'sin retorno del desfavorable revertido');

-- Tras volver al nombramiento cabe otra modificación (CT116) sobre v9.
SELECT pg_temp.exigir(vec_contratacion_temporal.proyeccion_modificacion_ct116(
  (SELECT agregado_json FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=9),
  jsonb_build_object('organizacion_ref',(SELECT agregado_json->>'organizacion_ref' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=9),
    'fase_retorno','fiscalizacion','estado_retorno','en_curso','periodo_inicio','2027-01-01','periodo_fin','2027-02-28','porcentaje_jornada',10000,
    'coste_centimos',100,'actor_ref','per_ct120_actor','observaciones','Segunda modificación'),'recibo:ct120:segunda','"2027-01-01T00:00:00Z"',false)->>'resultado'='preparada',
  'nueva modificación admitida tras volver al nombramiento');
SELECT 'CT120 OK' AS resultado;
