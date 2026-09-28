\set ON_ERROR_STOP on
-- Datos sintéticos preexistentes: CT115 confirmado, incorporación CT75 y
-- raíz de seguimiento coherentes. La falta de antecedente detiene el ensayo.
CREATE ROLE vec_rrhh_reincorp_ct LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
CREATE ROLE vec_rrhh_reincorp_bolsa LOGIN INHERIT IN ROLE vec_bolsa_llamamientos_ejecutor;
GRANT CONNECT ON DATABASE postgres TO vec_rrhh_reincorp_ct,vec_rrhh_reincorp_bolsa;

-- Dobles explícitos sólo de consumo V3. No son evidencia de firma ni de 403 HTTP.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_reincorporacion_titular_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.seguimiento.registrar_reincorporacion_titular' THEN
  RAISE EXCEPTION 'doble V3: acción denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
  encode(sha256(p_capacidad||p_decision),'hex'),'aud_v3_'||md5(p_capacidad||p_decision),clock_timestamp(),true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:bolsa:ensayo',convert_from(p_payload,'UTF8'),convert_from(p_sobre,'UTF8'),
  repeat('e',64),'aud_v3_'||repeat('e',32),clock_timestamp(),true
$f$;

CREATE FUNCTION pg_temp.exigir(ok boolean,detalle text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF ok IS NOT TRUE THEN RAISE EXCEPTION 'ensayo reincorporación: %',detalle; END IF; RETURN 'ok'; END $f$;
CREATE FUNCTION pg_temp.lectura_ajena(participacion text) RETURNS boolean LANGUAGE plpgsql AS $f$
DECLARE d jsonb;
BEGIN
 d:=jsonb_build_object('principal_id','per_bolsa_ensayo','accion','bolsa.situacion_participacion.cambiar',
  'modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
  'recurso_ref',participacion,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',repeat('a',64));
 PERFORM * FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(
  participacion,'per_bolsa_ajeno','\x01',convert_to(d::text,'UTF8'),'\x01','\x01',1,1,
  convert_to(participacion,'UTF8'),convert_to(repeat('a',64),'UTF8'),'\x01','\x01');
 RETURN false;
EXCEPTION WHEN insufficient_privilege THEN RETURN true;
END $f$;
CREATE FUNCTION pg_temp.entrada() RETURNS jsonb LANGUAGE plpgsql AS $f$
DECLARE c record; a jsonb; m jsonb; act jsonb; amb jsonb; atr jsonb; pol jsonb; aut jsonb; dec jsonb;
 inst text; ah text; hh text; h text; refs jsonb;
BEGIN
 SELECT ce.*,r.relacion_ref,ac.version AS version_actual,v.agregado_json AS ag,
  convert_from(i_b.registro_canonico,'UTF8')::jsonb#>>'{propuesta,participacion_seleccionada_ref}' AS participacion_ref
 INTO c FROM vec_contratacion_temporal.cese_nombramiento_v1 ce
 JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=ce.incorporacion_ref
 JOIN vec_contratacion_temporal.seguimiento_raiz_v2 r ON r.seguimiento_ref=i.seguimiento_ref
  AND r.expediente_ref=ce.expediente_ref AND r.organizacion_ref=ce.organizacion_ref
 JOIN vec_contratacion_temporal.expediente_integral_actual ac ON ac.expediente_ref=ce.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral v ON v.expediente_ref=ac.expediente_ref AND v.version=ac.version
 JOIN vec_bolsa_llamamientos.restriccion_cese_bolsa bc ON bc.origen_ref=ce.evento_ref
  AND bc.relacion_ref=r.relacion_ref AND bc.recibo_ct_ref=ce.recibo_ref AND bc.fecha_efecto=ce.fecha_efecto
 JOIN vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l ON l.llamamiento_ref=bc.llamamiento_ref
 JOIN vec_bolsa_llamamientos.integracion_desarrollo i_b ON i_b.operacion_ref=l.operacion_ref AND i_b.tipo='propuesta'
 WHERE ce.causa_clave='fin_sustitucion' AND ce.justificante_tipo='comunicacion_reincorporacion'
  AND ce.estado='confirmada' AND ac.version=ce.version_esperada+1
  AND i.recibo_json->>'EjercicioSintetico'='true' AND i.recibo_json->>'FirmaOficial'='false'
  AND i.recibo_json->>'EficaciaAdministrativa'='false'
  AND v.agregado_json->>'fase_actual'='nombramiento' AND v.agregado_json->>'estado_actual'='en_curso'
  AND i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'=r.relacion_ref
  AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.reincorporacion_titular_v1 x WHERE x.expediente_ref=ce.expediente_ref)
 ORDER BY ce.expediente_ref LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'falta cese CT115 sintético con raíz CT75 y expediente listo'; END IF;
 a:=c.ag;
 inst:=to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 ah:='hmac-sha256:vec.contratacion-temporal.reincorporacion-titular.ambito/v1:'||repeat('a',64);
 hh:='hmac-sha256:vec.contratacion-temporal.reincorporacion-titular.peticion/v1:'||repeat('b',64);
 refs:=jsonb_build_object('reserva_ref','reserva:ct130:ensayo','recibo_ref','recibo:ct130:ensayo','evento_ref','evento:ct130:ensayo');
 m:=jsonb_build_object('organizacion_ref',c.organizacion_ref,'expediente_ref',c.expediente_ref,
  'relacion_ref',c.relacion_ref,'version_esperada',c.version_actual,'actor_ref','per_ct130_ensayo',
  'perfil_ref','prf_ct130_ensayo','fecha_efectiva',to_char(c.fecha_efecto,'YYYY-MM-DD'),
  'documento_ref',c.justificante_ref,'documento_sha256',c.justificante_sha256);
 act:=jsonb_build_object('secuencia',jsonb_array_length(a->'actuaciones')+1,'version_expediente',c.version_actual+1,
  'accion_clave','contratacion_temporal.seguimiento.registrar_reincorporacion_titular','actor_ref','per_ct130_ensayo',
  'unidad_ref',a#>>'{asignacion,unidad_ref}','recibo_ref','recibo:ct130:ensayo','realizada_en',inst,
  'fase_origen','nombramiento','fase_destino','nombramiento','estado_origen','en_curso','estado_destino','en_curso',
  'documentos_ref',jsonb_build_array(c.justificante_ref));
 amb:=jsonb_build_object('organizacion_ref',c.organizacion_ref,'expediente_ref',c.expediente_ref,
  'fase_previa','nombramiento','estado_previo','en_curso');
 atr:=jsonb_build_object('version_expediente',c.version_actual::text,'relacion_ref',c.relacion_ref,
  'fecha_efectiva',to_char(c.fecha_efecto,'YYYY-MM-DD'),'documento_ref',c.justificante_ref,
  'documento_sha256',c.justificante_sha256,'cese_evento_ref',c.evento_ref,'cese_recibo_ref',c.recibo_ref,
  'ambito_idempotencia_hmac',ah,'huella_peticion_hmac',hh,'politica_ref','causas_cese_contratacion_temporal',
  'politica_version','1','politica_huella_sha256',repeat('c',64));
 h:=vec_contratacion_temporal.huella_contexto_go_ct115(amb,atr);
 dec:=jsonb_build_object('decision_ref','decision:ct130:ensayo','accion','contratacion_temporal.seguimiento.registrar_reincorporacion_titular',
  'modulo_id','contratacion_temporal','tipo_recurso','reincorporacion_titular_contratacion_temporal',
  'finalidad','registrar_reincorporacion_titular','recurso_ref',c.expediente_ref,'principal_id','per_ct130_ensayo',
  'perfil_activo_ref','prf_ct130_ensayo','contexto_recurso_huella_sha256',h);
 aut:=jsonb_build_object('accion','contratacion_temporal.seguimiento.registrar_reincorporacion_titular',
  'finalidad','registrar_reincorporacion_titular','recurso_ref',c.expediente_ref,'principal_id','per_ct130_ensayo',
  'perfil_activo_ref','prf_ct130_ensayo','decision_canonica_hex',encode(convert_to(dec::text,'UTF8'),'hex'),
  'motivo_canonico_hex',encode(convert_to('motivo','UTF8'),'hex'),'persona_version',1,'perfil_version',1,
  'decision_huella_sha256',encode(sha256(convert_to(dec::text,'UTF8')),'hex'),
  'decision_ref','decision:ct130:ensayo','contexto_recurso_huella_sha256',h);
 pol:=jsonb_build_object('definicion_ref','causas_cese_contratacion_temporal','definicion_version',1,
  'definicion_huella_sha256',repeat('c',64),'accion','contratacion_temporal.seguimiento.registrar_reincorporacion_titular',
  'finalidad','registrar_reincorporacion_titular','evaluada_en',inst,
  'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 RETURN jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-reincorporacion-titular.v1',
  'operacion','registrar_reincorporacion_titular','material',m,'referencias',refs,
  'ambito_idempotencia_hmac',ah,'huella_peticion_hmac',hh,'expediente_anterior',a,
  'expediente_siguiente',a||jsonb_build_object('version',c.version_actual+1,'actualizado_en',inst,
   'actuaciones',a->'actuaciones'||jsonb_build_array(act)),
  'actuacion',act,'politica',pol,'autorizacion',aut,'instante_efecto',inst,
  'contexto',jsonb_build_object('ambitos',amb,'atributos',atr),'_decision',dec,
  '_participacion_ref',c.participacion_ref);
END $f$;
DO $f$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_rrhh_reincorp_ct,vec_rrhh_reincorp_bolsa',pg_my_temp_schema()::regnamespace); END $f$;
SELECT pg_temp.entrada()::text AS entrada \gset
SELECT pg_temp.exigir((:'entrada'::jsonb)#>>'{material,fecha_efectiva}' IS NOT NULL,'cese antecedente');

SET SESSION AUTHORIZATION vec_rrhh_reincorp_ct;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb_build_object(
 'esquema','vec.contratacion-temporal.preparar-reincorporacion-titular.v1',
 'operacion','registrar_reincorporacion_titular','material',:'entrada'::jsonb->'material',
 'sellos_hmac',jsonb_build_object('activo',jsonb_build_object('ambito_hmac',:'entrada'::jsonb->>'ambito_idempotencia_hmac',
  'generacion',1,'huella_peticion_hmac',:'entrada'::jsonb->>'huella_peticion_hmac'),'retenidos','[]'::jsonb),
 'referencias_candidatas',:'entrada'::jsonb->'referencias'))::text AS preparada \gset
COMMIT;
SELECT pg_temp.exigir((:'preparada'::jsonb)->>'resultado'='preparada','preparación CT130');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(:'entrada'::jsonb-'_decision',
 convert_to('capacidad','UTF8'),convert_to((:'entrada'::jsonb->'_decision')::text,'UTF8'),
 convert_to('motivo','UTF8'),'\x01',1,1,'\x01','\x01','\x01','\x01')::text AS confirmada \gset
COMMIT;
SELECT pg_temp.exigir((:'confirmada'::jsonb)->>'resultado'='confirmada' AND
 (:'confirmada'::jsonb)#>>'{recibo,recibo_ref}'='recibo:ct130:ensayo','acto CT130');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(:'entrada'::jsonb-'_decision',
 convert_to('capacidad','UTF8'),convert_to((:'entrada'::jsonb->'_decision')::text,'UTF8'),
 convert_to('motivo','UTF8'),'\x01',1,1,'\x01','\x01','\x01','\x01')::text AS replay \gset
COMMIT;
SELECT pg_temp.exigir((:'replay'::jsonb)->'recibo'=(:'confirmada'::jsonb)->'recibo','recibo idempotente');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb_build_object(
 'esquema','vec.contratacion-temporal.preparar-reincorporacion-titular.v1',
 'operacion','registrar_reincorporacion_titular',
 'material',jsonb_set(:'entrada'::jsonb->'material','{documento_sha256}',to_jsonb(repeat('d',64))),
 'sellos_hmac',jsonb_build_object('activo',jsonb_build_object('ambito_hmac',:'entrada'::jsonb->>'ambito_idempotencia_hmac',
  'generacion',1,'huella_peticion_hmac',:'entrada'::jsonb->>'huella_peticion_hmac'),'retenidos','[]'::jsonb),
 'referencias_candidatas',:'entrada'::jsonb->'referencias'))::text AS clave_reutilizada \gset
COMMIT;
SELECT pg_temp.exigir((:'clave_reutilizada'::jsonb)->>'resultado'='idempotencia_reutilizada',
 'misma clave con documento distinto');
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT evento_ref AS evento,huella_sha256 AS huella,origen_posicion AS posicion
 FROM vec_contratacion_temporal.leer_reincorporaciones_bolsa_v1(NULL,NULL,100)
 WHERE evento_ref='evento:ct130:ensayo' \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir(:'evento'='evento:ct130:ensayo' AND length(:'huella')=64,'feed CT130');

-- B46 consume el feed CT130; su verificador no es un doble.
SET SESSION AUTHORIZATION vec_rrhh_reincorp_bolsa;
SELECT reutilizada AS primera,estado AS estado_primero FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
 :'evento',:'huella',:'posicion'::bigint) \gset
SELECT reutilizada AS segunda,estado AS estado_segundo FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
 :'evento',:'huella',:'posicion'::bigint) \gset
SELECT pg_temp.exigir(:'primera'='f' AND :'segunda'='t' AND :'estado_primero'='cese_aplicado'
 AND :'estado_segundo'='cese_aplicado','recepción idempotente B46');
SELECT (:'entrada'::jsonb->>'_participacion_ref') AS participacion \gset
SELECT pg_temp.exigir(:'participacion'<>'','participación propia resuelta');
SELECT count(*) AS visibles FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(
 :'participacion','per_bolsa_ensayo','\x01',convert_to(jsonb_build_object(
 'principal_id','per_bolsa_ensayo','accion','bolsa.situacion_participacion.cambiar','modulo_id','bolsa',
 'tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
 'recurso_ref',:'participacion','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
 'contexto_recurso_huella_sha256',repeat('a',64))::text,'UTF8'),'\x01','\x01',1,1,
 convert_to(:'participacion','UTF8'),convert_to(repeat('a',64),'UTF8'),'\x01','\x01') \gset
SELECT pg_temp.exigir(:'visibles'=1,'ficha de participación propia');
SELECT pg_temp.exigir(pg_temp.lectura_ajena(:'participacion'),'lectura ajena 42501');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.reincorporacion_titular_v1 WHERE evento_ref=:'evento')=1
 AND (SELECT count(*) FROM vec_bolsa_llamamientos.reincorporacion_titular_ct WHERE evento_ref=:'evento')=1,
 'sin duplicado CT130/B46');
SELECT 'RRHH_REINCORP_CADENA_OK';
