\set ON_ERROR_STOP on
-- CT116 sobre la estructura real restaurada, después de la prueba de CT115
-- (reutiliza su rol de ejecución). La fachada AD3-83 se sustituye por un doble
-- explícito: prueba la transacción CT, no la criptografía V3.
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
\set exp_a 'expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_modificacion_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.expediente.modificar_tras_nombramiento' THEN RAISE EXCEPTION 'doble: denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;

CREATE FUNCTION pg_temp.exigir(c boolean, msg text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF c IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', msg; END IF; RETURN 'ok'; END $$;

-- Entrada de confirmación tal como la compone el adaptador Go.
CREATE FUNCTION pg_temp.entrada_modificacion(p_exp text, p_version numeric, p_inicio text, p_fin text, p_jornada int, p_coste bigint, p_fase text, p_sufijo text)
RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb; inst text; m jsonb; act jsonb; amb jsonb; atr jsonb; h text; dec jsonb; amb_hmac text; hue_hmac text; org text; an jsonb; sig jsonb;
BEGIN
 SELECT v.agregado_json INTO STRICT ag FROM vec_contratacion_temporal.expediente_version_integral v WHERE v.expediente_ref=p_exp AND v.version=p_version;
 org:=ag->>'organizacion_ref';
 inst:=to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 amb_hmac:='hmac-sha256:vec.contratacion-temporal.modificacion-nombramiento.ambito/v1:'||encode(sha256(convert_to('a'||p_sufijo,'UTF8')),'hex');
 hue_hmac:='hmac-sha256:vec.contratacion-temporal.modificacion-nombramiento.peticion/v1:'||encode(sha256(convert_to('p'||p_sufijo,'UTF8')),'hex');
 m:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'version_esperada',p_version,'actor_ref','per_ct116_actor','perfil_ref','prf_ct116',
   'motivo_clave','cambio_jornada','periodo_inicio',p_inicio,'periodo_fin',p_fin,'porcentaje_jornada',p_jornada,'coste_centimos',p_coste,
   'fuente_coste_ref','autoridad:ct:desarrollo:calculo-coste','fase_retorno',p_fase,'estado_retorno','en_curso','observaciones','Reducción de jornada pedida por el centro');
 act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',p_version+1,
   'accion_clave','contratacion_temporal.expediente.modificar_tras_nombramiento','actor_ref','per_ct116_actor','unidad_ref',ag#>>'{asignacion,unidad_ref}',
   'recibo_ref','recibo:ct116:'||p_sufijo,'realizada_en',inst,'fase_origen','nombramiento','fase_destino',p_fase,'estado_origen','en_curso',
   'estado_destino','en_curso','observaciones','Reducción de jornada pedida por el centro');
 an:=(ag->'analisis')||jsonb_build_object('periodo',jsonb_build_object('inicio',p_inicio||'T00:00:00Z','fin',p_fin||'T00:00:00Z'),'porcentaje_jornada',p_jornada,
   'coste_previsto',jsonb_build_object('moneda','EUR','centimos',p_coste),'fuente_coste_ref','autoridad:ct:desarrollo:calculo-coste',
   'actuacion_registro',jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',p_version+1,
     'accion_clave','contratacion_temporal.expediente.modificar_tras_nombramiento','fase_destino',p_fase,'recibo_ref','recibo:ct116:'||p_sufijo));
 sig:=(ag-'fiscalizacion')||jsonb_build_object('version',p_version+1,'fase_actual',p_fase,'estado_actual','en_curso','actualizado_en',inst,'analisis',an,
   'actuaciones',(ag->'actuaciones')||jsonb_build_array(act));
 IF p_fase='informe_juridico' THEN sig:=sig-'informe_juridico'; END IF;
 amb:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'fase_previa','nombramiento','estado_previo','en_curso');
 atr:=jsonb_build_object('version_expediente',p_version::text,'motivo_clave','cambio_jornada','periodo_inicio',p_inicio,'periodo_fin',p_fin,
   'porcentaje_jornada',p_jornada::text,'coste_centimos',p_coste::text,'fuente_coste_ref','autoridad:ct:desarrollo:calculo-coste','fase_retorno',p_fase,
   'estado_retorno','en_curso','observaciones_huella_sha256',encode(sha256(convert_to('Reducción de jornada pedida por el centro','UTF8')),'hex'),
   'politica_ref','vec.contratacion_temporal.reglas','politica_version','1','politica_huella_sha256',repeat('d',64),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac);
 h:=vec_contratacion_temporal.huella_contexto_go_ct115(amb,atr);
 dec:=jsonb_build_object('decision_ref','decision:ct116:'||p_sufijo,'accion','contratacion_temporal.expediente.modificar_tras_nombramiento',
   'modulo_id','contratacion_temporal','tipo_recurso','modificacion_contratacion_temporal','finalidad','modificar_expediente_tras_nombramiento',
   'recurso_ref',p_exp,'principal_id','per_ct116_actor','perfil_activo_ref','prf_ct116','contexto_recurso_huella_sha256',h);
 RETURN jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-modificacion-nombramiento.v1','operacion','modificar_tras_nombramiento','material',m,
   'referencias',jsonb_build_object('reserva_ref','reserva:ct116:'||p_sufijo,'recibo_ref','recibo:ct116:'||p_sufijo,'evento_ref','evento:ct116:'||p_sufijo),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac,'expediente_anterior',ag,'expediente_siguiente',sig,'actuacion',act,
   'politica',jsonb_build_object('definicion_ref','vec.contratacion_temporal.reglas','definicion_version',1,'definicion_huella_sha256',repeat('d',64),
     'accion','contratacion_temporal.expediente.modificar_tras_nombramiento','finalidad','modificar_expediente_tras_nombramiento','evaluada_en',inst,
     'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
   'autorizacion',jsonb_build_object('accion','contratacion_temporal.expediente.modificar_tras_nombramiento','finalidad','modificar_expediente_tras_nombramiento',
     'recurso_ref',p_exp,'principal_id','per_ct116_actor','perfil_activo_ref','prf_ct116','decision_canonica_hex',encode(convert_to(dec::text,'UTF8'),'hex'),
     'motivo_canonico_hex',encode('\x6d6f7469766f'::bytea,'hex'),'persona_version',1,'perfil_version',1,
     'decision_huella_sha256',encode(sha256(convert_to(dec::text,'UTF8')),'hex'),'decision_ref','decision:ct116:'||p_sufijo,'contexto_recurso_huella_sha256',h),
   'instante_efecto',inst,'contexto',jsonb_build_object('ambitos',amb,'atributos',atr),'_decision',dec);
END $$;
CREATE FUNCTION pg_temp.confirmar(e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.confirmar_modificacion_nombramiento_v1(e-'_decision','\x6361706163696461640a',convert_to((e->'_decision')::text,'UTF8'),
   '\x6d6f7469766f','\x01',1,1,'\x01','\x01','\x01','\x01');
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
CREATE FUNCTION pg_temp.preparar(e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.preparar_modificacion_nombramiento_v1(jsonb_build_object('esquema','vec.contratacion-temporal.preparar-modificacion-nombramiento.v1',
  'operacion','modificar_tras_nombramiento','material',e->'material','sellos_hmac',jsonb_build_object('activo',jsonb_build_object('ambito_hmac',
  e->>'ambito_idempotencia_hmac','generacion',1,'huella_peticion_hmac',e->>'huella_peticion_hmac'),'retenidos','[]'::jsonb),'referencias_candidatas',e->'referencias'));
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct115_runtime',pg_my_temp_schema()::regnamespace); END $$;

SELECT pg_temp.entrada_modificacion(:'exp_b',7,'2027-01-01','2027-03-31',5000,2000000,'fiscalizacion','uno')::text AS mod_uno \gset
SELECT pg_temp.entrada_modificacion(:'exp_b',7,'2027-01-01','2027-03-31',10000,2000000,'fiscalizacion','igual')::text AS mod_igual \gset
SELECT pg_temp.entrada_modificacion(:'exp_b',7,'2027-01-01','2027-12-31',10000,9000000,'fiscalizacion','caro')::text AS mod_caro \gset
SELECT pg_temp.entrada_modificacion(:'exp_a',9,'2027-01-01','2027-03-31',5000,2000000,'fiscalizacion','cerrado')::text AS mod_cerrado \gset

SET SESSION AUTHORIZATION vec_ct115_runtime;
SELECT pg_temp.exigir(NOT has_table_privilege('vec_contratacion_temporal.modificacion_nombramiento_v1','SELECT'),'lectura directa de modificaciones');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar(:'mod_uno'::jsonb)->>'resultado' AS prep_uno \gset
SELECT pg_temp.preparar(:'mod_igual'::jsonb)->>'resultado' AS prep_igual \gset
SELECT pg_temp.preparar(:'mod_caro'::jsonb)->>'resultado' AS prep_caro \gset
SELECT pg_temp.preparar(:'mod_cerrado'::jsonb)->>'resultado' AS prep_cerrado \gset
COMMIT;
SELECT pg_temp.exigir(:'prep_uno'='preparada','preparación de la modificación');
SELECT pg_temp.exigir(:'prep_igual'='sin_cambios','modificación sin cambios');
SELECT pg_temp.exigir(:'prep_caro'='credito_insuficiente','coste por encima de la retención de crédito');
SELECT pg_temp.exigir(:'prep_cerrado'='cese_existente','modificación tras el cese');

BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'mod_uno'::jsonb,'{material,fase_retorno}','"nombramiento"'))->>'error'='22023','fase de retorno no admitida');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'mod_uno'::jsonb,'{expediente_siguiente,fiscalizacion}','{}'))->>'error'='22023','fiscalización conservada en la proyección');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'mod_uno'::jsonb,'{contexto,atributos,coste_centimos}','"1"'))->>'error'='42501','coste distinto del autorizado');
SELECT pg_temp.exigir(pg_temp.confirmar(jsonb_set(:'mod_uno'::jsonb,'{material,periodo_fin}','"2026-12-31"'))->>'error'='22023','periodo invertido');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar(:'mod_uno'::jsonb)::text AS conf \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar(:'mod_uno'::jsonb)::text AS replay \gset
COMMIT;
SELECT pg_temp.exigir((:'conf'::jsonb)->>'resultado'='confirmada' AND (:'conf'::jsonb)#>>'{recibo,fase_resultante}'='fiscalizacion'
  AND (:'conf'::jsonb)#>>'{recibo,version_resultante}'='8','confirmación de la modificación '||:'conf');
SELECT pg_temp.exigir((:'replay'::jsonb)->'recibo'=(:'conf'::jsonb)->'recibo','repetición con el mismo recibo');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar(:'mod_uno'::jsonb)::text AS recup \gset
COMMIT;
SELECT pg_temp.exigir((:'recup'::jsonb)->>'resultado'='confirmada' AND (:'recup'::jsonb)->'recibo'=(:'conf'::jsonb)->'recibo','recuperación');
RESET SESSION AUTHORIZATION;

SELECT pg_temp.exigir((SELECT fase_clave||'/'||estado FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=8)='fiscalizacion/en_curso','fase de retorno');
SELECT pg_temp.exigir((SELECT agregado_json#>>'{analisis,porcentaje_jornada}' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=8)='5000'
  AND (SELECT NOT agregado_json ? 'fiscalizacion' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=8)
  AND (SELECT agregado_json ? 'fiscalizacion' FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=7),'análisis nuevo y fiscalización anterior conservada en su versión');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_b' AND tipo_evento='ct.modificacion.v1')=1,'evento de modificación');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh WHERE expediente_ref=:'exp_b' AND version=8)=1,'publicación RRHH de la modificación');
SELECT 'CT116 OK' AS resultado;
