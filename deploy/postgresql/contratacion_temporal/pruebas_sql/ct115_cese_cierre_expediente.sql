\set ON_ERROR_STOP on
-- CT115 sobre la estructura real restaurada (volcado de la principal) con
-- AD3-82 y CT115 instaladas. Base desechable: los datos se confirman.
-- La fachada AD3-82 se sustituye por un doble explícito que devuelve el
-- consumo ligado a la decisión: prueba la transacción CT (versión, actuación,
-- outbox, recibo, idempotencia, RLS y negativos), no la criptografía V3.
\set exp_a 'expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'

-- Requiere antes ct115_ct116_fixture_pg18.sql en la misma sesión.

-- Construye la entrada de confirmación del cese igual que el adaptador Go.
CREATE FUNCTION pg_temp.entrada_cese(p_exp text, p_version numeric, p_causa text, p_fecha text, p_sufijo text, p_accion text DEFAULT 'contratacion_temporal.seguimiento.cesar')
RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb; inst text; m jsonb; act jsonb; amb jsonb; atr jsonb; h text; dec jsonb; amb_hmac text; hue_hmac text; org text; inc text;
BEGIN
 SELECT v.agregado_json INTO STRICT ag FROM vec_contratacion_temporal.expediente_version_integral v WHERE v.expediente_ref=p_exp AND v.version=p_version;
 org:=ag->>'organizacion_ref';
 SELECT recibo_ref INTO inc FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE expediente_ref=p_exp;
 inst:=to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 amb_hmac:='hmac-sha256:vec.contratacion-temporal.cese.ambito/v1:'||encode(sha256(convert_to('a'||p_sufijo,'UTF8')),'hex');
 hue_hmac:='hmac-sha256:vec.contratacion-temporal.cese.peticion/v1:'||encode(sha256(convert_to('p'||p_sufijo||p_causa,'UTF8')),'hex');
 m:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'version_esperada',p_version,'actor_ref','per_ct115_actor','perfil_ref','prf_ct115',
   'causa_clave',p_causa,'fecha_efecto',p_fecha,'justificante_tipo','comunicacion_reincorporacion','justificante_ref','documento:ct115:justificante:'||p_sufijo,
   'justificante_sha256',encode(sha256(convert_to('justificante'||p_sufijo,'UTF8')),'hex'),'observaciones','Reincorporación de la titular');
 act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',p_version+1,'accion_clave','contratacion_temporal.seguimiento.cesar',
   'actor_ref','per_ct115_actor','unidad_ref',ag#>>'{asignacion,unidad_ref}','recibo_ref','recibo:ct115:'||p_sufijo,'realizada_en',inst,
   'fase_origen','nombramiento','fase_destino','nombramiento','estado_origen','en_curso','estado_destino','en_curso',
   'documentos_ref',jsonb_build_array('documento:ct115:justificante:'||p_sufijo),'observaciones','Reincorporación de la titular');
 amb:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'fase_previa','nombramiento','estado_previo','en_curso');
 atr:=jsonb_build_object('version_expediente',p_version::text,'causa_clave',p_causa,'fecha_efecto',p_fecha,'justificante_tipo','comunicacion_reincorporacion',
   'justificante_ref','documento:ct115:justificante:'||p_sufijo,'justificante_sha256',encode(sha256(convert_to('justificante'||p_sufijo,'UTF8')),'hex'),
   'observaciones_huella_sha256',encode(sha256(convert_to('Reincorporación de la titular','UTF8')),'hex'),'incorporacion_ref',coalesce(inc,'ref:ninguna'),
   'politica_ref','causas_cese_contratacion_temporal','politica_version','1','politica_huella_sha256',repeat('c',64),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac);
 h:=vec_contratacion_temporal.huella_contexto_go_ct115(amb,atr);
 dec:=jsonb_build_object('decision_ref','decision:ct115:'||p_sufijo,'accion',p_accion,'modulo_id','contratacion_temporal','tipo_recurso','cese_contratacion_temporal',
   'finalidad','registrar_cese_contratacion_temporal','recurso_ref',p_exp,'principal_id','per_ct115_actor','perfil_activo_ref','prf_ct115','contexto_recurso_huella_sha256',h);
 RETURN jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-cese.v1','operacion','registrar_cese','material',m,
   'referencias',jsonb_build_object('reserva_ref','reserva:ct115:'||p_sufijo,'recibo_ref','recibo:ct115:'||p_sufijo,'evento_ref','evento:ct115:'||p_sufijo),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac,'expediente_anterior',ag,
   'expediente_siguiente',ag||jsonb_build_object('version',p_version+1,'actualizado_en',inst,'actuaciones',(ag->'actuaciones')||jsonb_build_array(act)),
   'actuacion',act,'politica',jsonb_build_object('definicion_ref','causas_cese_contratacion_temporal','definicion_version',1,'definicion_huella_sha256',repeat('c',64),
     'accion','contratacion_temporal.seguimiento.cesar','finalidad','registrar_cese_contratacion_temporal','evaluada_en',inst,
     'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
   'autorizacion',jsonb_build_object('accion',p_accion,'finalidad','registrar_cese_contratacion_temporal','recurso_ref',p_exp,'principal_id','per_ct115_actor',
     'perfil_activo_ref','prf_ct115','decision_canonica_hex',encode(convert_to(dec::text,'UTF8'),'hex'),'motivo_canonico_hex',encode('\x6d6f7469766f'::bytea,'hex'),
     'persona_version',1,'perfil_version',1,'decision_huella_sha256',encode(sha256(convert_to(dec::text,'UTF8')),'hex'),'decision_ref','decision:ct115:'||p_sufijo,
     'contexto_recurso_huella_sha256',h),
   'instante_efecto',inst,'contexto',jsonb_build_object('ambitos',amb,'atributos',atr),
   '_decision',dec);
END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct115_runtime',pg_my_temp_schema()::regnamespace); END $$;

-- Utilidades de aserción: las variables de psql no se sustituyen dentro de
-- cuerpos entre dólares, así que las comprobaciones reciben los valores.
CREATE FUNCTION pg_temp.exigir(c boolean, msg text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF c IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', msg; END IF; RETURN 'ok'; END $$;
CREATE FUNCTION pg_temp.confirmar_cese(e jsonb, p_decision_extra text DEFAULT '') RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.confirmar_cese_nombramiento_v1(e-'_decision','\x6361706163696461640a',convert_to((e->'_decision')::text||p_decision_extra,'UTF8'),
   '\x6d6f7469766f','\x01',1,1,'\x01','\x01','\x01','\x01');
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
CREATE FUNCTION pg_temp.confirmar_cierre(e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.confirmar_cierre_expediente_v1(e-'_decision','\x6361706163696461640a',convert_to((e->'_decision')::text,'UTF8'),
   '\x6d6f7469766f','\x01',1,1,'\x01','\x01','\x01','\x01');
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
CREATE FUNCTION pg_temp.preparar_cese(e jsonb, p_activo jsonb DEFAULT NULL, p_retenidos jsonb DEFAULT '[]') RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN
 RETURN vec_contratacion_temporal.preparar_cese_nombramiento_v1(jsonb_build_object('esquema','vec.contratacion-temporal.preparar-cese.v1','operacion','registrar_cese',
  'material',e->'material','sellos_hmac',jsonb_build_object('activo',coalesce(p_activo,jsonb_build_object('ambito_hmac',e->>'ambito_idempotencia_hmac',
  'generacion',1,'huella_peticion_hmac',e->>'huella_peticion_hmac')),'retenidos',p_retenidos),'referencias_candidatas',e->'referencias'));
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE);
END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct115_runtime',pg_my_temp_schema()::regnamespace); END $$;

-- Vector de compatibilidad con Go (mismo valor en la prueba Go del contexto).
SELECT pg_temp.exigir(vec_contratacion_temporal.huella_contexto_go_ct115('{"b":"2","a":"x:1"}','{"k":"hmac-sha256:vec.x/v1:ab","c":"cese_registrado,ginpix_confirmado"}')
    =encode(sha256(convert_to('{"ambitos":{"a":"x:1","b":"2"},"atributos":{"c":"cese_registrado,ginpix_confirmado","k":"hmac-sha256:vec.x/v1:ab"}}','UTF8')),'hex'),'vector de contexto');

SELECT pg_temp.entrada_cese(:'exp_a',7,'fin_sustitucion','2027-02-15','uno')::text AS entrada_uno \gset
SELECT pg_temp.entrada_cese(:'exp_a',7,'fin_sustitucion','2026-12-15','anterior')::text AS entrada_anterior \gset
SELECT pg_temp.entrada_cese(:'exp_b',7,'fin_sustitucion','2027-02-15','b')::text AS entrada_b \gset
SELECT pg_temp.entrada_cese(:'exp_a',7,'fin_sustitucion','2027-02-15','accion','contratacion_temporal.expediente.cerrar')::text AS entrada_accion \gset

-- Fuera de una sesión del ejecutor de CT la fachada se niega.
SELECT pg_temp.exigir((pg_temp.preparar_cese(:'entrada_uno'::jsonb))->>'error'='42501','superusuario sin sesión de ejecutor');

SET SESSION AUTHORIZATION vec_ct115_runtime;
SELECT pg_temp.exigir(NOT has_table_privilege('vec_contratacion_temporal.cese_nombramiento_v1','SELECT'),'lectura directa de ceses');
SELECT pg_temp.exigir(NOT has_function_privilege('vec_contratacion_temporal.texto_valido_ct115(text,integer,boolean)','EXECUTE'),'auxiliar ejecutable');

BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar_cese(:'entrada_uno'::jsonb)::text AS prep \gset
SELECT pg_temp.preparar_cese(:'entrada_b'::jsonb)->>'resultado' AS prep_b \gset
COMMIT;
SELECT pg_temp.exigir((:'prep'::jsonb)->>'resultado'='preparada' AND (:'prep'::jsonb)#>>'{incorporacion,inicio}'='2027-01-01'
  AND (:'prep'::jsonb)#>>'{expediente,version}'='7','preparación del cese');
SELECT pg_temp.exigir(:'prep_b'='sin_incorporacion','preparación sin incorporación');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir((pg_temp.preparar_cese(:'entrada_uno'::jsonb))->>'error'='42501','preparación en transacción de escritura');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.exigir((pg_temp.confirmar_cese(:'entrada_uno'::jsonb))->>'error'='42501','confirmación en solo lectura');
ROLLBACK;

-- Negativos de confirmación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar_cese(:'entrada_anterior'::jsonb)->>'resultado'='fecha_anterior_incorporacion','fecha anterior a la incorporación');
SELECT pg_temp.exigir(pg_temp.confirmar_cese(:'entrada_b'::jsonb)->>'resultado'='sin_incorporacion','cese sin incorporación');
SELECT pg_temp.exigir(pg_temp.confirmar_cese(:'entrada_accion'::jsonb)->>'error'='42501','acción ajena');
SELECT pg_temp.exigir(pg_temp.confirmar_cese(jsonb_set(:'entrada_uno'::jsonb,'{expediente_siguiente,estado_actual}','"completado"'))->>'error'='22023','proyección manipulada');
SELECT pg_temp.exigir(pg_temp.confirmar_cese(jsonb_set(:'entrada_uno'::jsonb,'{contexto,atributos,causa_clave}','"renuncia"'))->>'error'='42501','contexto manipulado');
SELECT pg_temp.exigir(pg_temp.confirmar_cese(:'entrada_uno'::jsonb,' ')->>'error'='42501','decisión divergente');
SELECT pg_temp.exigir(pg_temp.confirmar_cese(jsonb_set(:'entrada_uno'::jsonb,'{material,fecha_efecto}','"2027-02-30"'))->>'error'='22023','fecha imposible');
SELECT pg_temp.exigir(pg_temp.confirmar_cese(jsonb_set(:'entrada_uno'::jsonb,'{material,observaciones}','" con espacio"'))->>'error'='22023','texto con espacio de borde');
ROLLBACK;

-- Confirmación, repetición, clave reutilizada y segundo cese.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar_cese(:'entrada_uno'::jsonb)::text AS conf \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar_cese(:'entrada_uno'::jsonb)::text AS replay \gset
SELECT pg_temp.confirmar_cese(jsonb_set(:'entrada_uno'::jsonb,'{material,causa_clave}','"renuncia"'))->>'resultado' AS reutilizada \gset
COMMIT;
SELECT pg_temp.exigir((:'conf'::jsonb)->>'resultado'='confirmada' AND (:'conf'::jsonb)#>>'{recibo,version_resultante}'='8'
  AND (:'conf'::jsonb)#>>'{recibo,causa_clave}'='fin_sustitucion','confirmación del cese');
SELECT pg_temp.exigir((:'replay'::jsonb)->'recibo'=(:'conf'::jsonb)->'recibo','repetición con el mismo recibo');
SELECT pg_temp.exigir(:'reutilizada'='idempotencia_reutilizada','clave reutilizada con otro contenido');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.entrada_cese(:'exp_a',8,'renuncia','2027-02-20','dos')::text AS entrada_dos \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar_cese(:'entrada_dos'::jsonb)->>'resultado'='cese_existente','segundo cese del mismo expediente');
COMMIT;

-- Recuperación por la preparación con una generación HMAC retenida.
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar_cese(:'entrada_uno'::jsonb,
  jsonb_build_object('ambito_hmac','hmac-sha256:vec.contratacion-temporal.cese.ambito/v2:'||repeat('e',64),'generacion',2,
    'huella_peticion_hmac','hmac-sha256:vec.contratacion-temporal.cese.peticion/v2:'||repeat('e',64)),
  jsonb_build_array(jsonb_build_object('ambito_hmac',(:'entrada_uno'::jsonb)->>'ambito_idempotencia_hmac','generacion',1,
    'huella_peticion_hmac',(:'entrada_uno'::jsonb)->>'huella_peticion_hmac')))::text AS recup \gset
COMMIT;
SELECT pg_temp.exigir((:'recup'::jsonb)->>'resultado'='confirmada' AND (:'recup'::jsonb)->'recibo'=(:'conf'::jsonb)->'recibo'
  AND (:'recup'::jsonb)#>>'{referencias,recibo_ref}'='recibo:ct115:uno','recuperación por la preparación');

-- Publicación a Bolsa con el formato común de la incorporación.
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT count(*) AS n_bolsa, min(evento->>'tipo') AS tipo_bolsa, min(evento->>'fin_previsto') AS fin_bolsa, min(evento->>'causa_clave') AS causa_bolsa,
  bool_and(huella_sha256=encode(sha256(convert_to(evento::text,'UTF8')),'hex')) AS huella_bolsa,
  bool_and((SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(evento) k)=ARRAY['categoria_ref','causa_clave','esquema','evento_ref','expediente_ref','fin_previsto','inicio',
             'llamamiento_ref','modalidad_clave','ocurrido_en','organizacion_ref','origen_ref','tipo']) AS claves_bolsa
  FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL,NULL,100) WHERE evento->>'tipo'='cese' \gset
COMMIT;
SELECT pg_temp.exigir(:'n_bolsa'=1 AND :'tipo_bolsa'='cese' AND :'fin_bolsa'='2027-02-15T00:00:00.000000Z' AND :'causa_bolsa'='fin_sustitucion'
  AND :'huella_bolsa' AND :'claves_bolsa','publicación a Bolsa');
RESET SESSION AUTHORIZATION;

-- ---------------------------------------------------------------- cierre
CREATE FUNCTION pg_temp.entrada_cierre(p_exp text, p_version numeric, p_condiciones text[], p_ginpix text, p_fecha text, p_sufijo text)
RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb; inst text; m jsonb; act jsonb; amb jsonb; atr jsonb; h text; dec jsonb; amb_hmac text; hue_hmac text; org text; cese text;
BEGIN
 SELECT v.agregado_json INTO STRICT ag FROM vec_contratacion_temporal.expediente_version_integral v WHERE v.expediente_ref=p_exp AND v.version=p_version;
 org:=ag->>'organizacion_ref';
 PERFORM set_config('vec.ct115.organizacion_ref',org,true); PERFORM set_config('vec.ct115.expediente_ref',p_exp,true);
 cese:='recibo:ct115:uno';
 inst:=to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 amb_hmac:='hmac-sha256:vec.contratacion-temporal.cierre-expediente.ambito/v1:'||encode(sha256(convert_to('a'||p_sufijo,'UTF8')),'hex');
 hue_hmac:='hmac-sha256:vec.contratacion-temporal.cierre-expediente.peticion/v1:'||encode(sha256(convert_to('p'||p_sufijo,'UTF8')),'hex');
 m:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'version_esperada',p_version,'actor_ref','per_ct115_actor','perfil_ref','prf_ct115',
   'condiciones',to_jsonb(p_condiciones),'ginpix_numero',p_ginpix,'ginpix_confirmada_en',p_fecha,'observaciones','');
 act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',p_version+1,'accion_clave','contratacion_temporal.expediente.cerrar',
   'actor_ref','per_ct115_actor','unidad_ref',ag#>>'{asignacion,unidad_ref}','recibo_ref','recibo:ct115c:'||p_sufijo,'realizada_en',inst,
   'fase_origen','nombramiento','fase_destino','nombramiento','estado_origen','en_curso','estado_destino','completado');
 IF p_ginpix<>'' THEN act:=act||jsonb_build_object('documentos_ref',jsonb_build_array('ginpix:'||p_ginpix)); END IF;
 amb:=jsonb_build_object('organizacion_ref',org,'expediente_ref',p_exp,'fase_previa','nombramiento','estado_previo','en_curso');
 atr:=jsonb_build_object('version_expediente',p_version::text,'cese_recibo_ref',cese,'condiciones',array_to_string(p_condiciones,','),'ginpix_numero',p_ginpix,
   'ginpix_confirmada_en',p_fecha,'observaciones_huella_sha256',encode(sha256(convert_to('','UTF8')),'hex'),
   'politica_ref','vec.contratacion_temporal.reglas','politica_version','1','politica_huella_sha256',repeat('d',64),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac);
 h:=vec_contratacion_temporal.huella_contexto_go_ct115(amb,atr);
 dec:=jsonb_build_object('decision_ref','decision:ct115c:'||p_sufijo,'accion','contratacion_temporal.expediente.cerrar','modulo_id','contratacion_temporal',
   'tipo_recurso','cierre_expediente_contratacion_temporal','finalidad','cerrar_expediente_tras_cese','recurso_ref',p_exp,'principal_id','per_ct115_actor',
   'perfil_activo_ref','prf_ct115','contexto_recurso_huella_sha256',h);
 RETURN jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-cierre-expediente.v1','operacion','cerrar_expediente','material',m,
   'referencias',jsonb_build_object('reserva_ref','reserva:ct115c:'||p_sufijo,'recibo_ref','recibo:ct115c:'||p_sufijo,'evento_ref','evento:ct115c:'||p_sufijo),
   'ambito_idempotencia_hmac',amb_hmac,'huella_peticion_hmac',hue_hmac,'expediente_anterior',ag,
   'expediente_siguiente',ag||jsonb_build_object('version',p_version+1,'estado_actual','completado','actualizado_en',inst,'actuaciones',(ag->'actuaciones')||jsonb_build_array(act)),
   'actuacion',act,'politica',jsonb_build_object('definicion_ref','vec.contratacion_temporal.reglas','definicion_version',1,'definicion_huella_sha256',repeat('d',64),
     'accion','contratacion_temporal.expediente.cerrar','finalidad','cerrar_expediente_tras_cese','evaluada_en',inst,
     'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
   'autorizacion',jsonb_build_object('accion','contratacion_temporal.expediente.cerrar','finalidad','cerrar_expediente_tras_cese','recurso_ref',p_exp,'principal_id','per_ct115_actor',
     'perfil_activo_ref','prf_ct115','decision_canonica_hex',encode(convert_to(dec::text,'UTF8'),'hex'),'motivo_canonico_hex',encode('\x6d6f7469766f'::bytea,'hex'),
     'persona_version',1,'perfil_version',1,'decision_huella_sha256',encode(sha256(convert_to(dec::text,'UTF8')),'hex'),'decision_ref','decision:ct115c:'||p_sufijo,
     'contexto_recurso_huella_sha256',h),
   'instante_efecto',inst,'contexto',jsonb_build_object('ambitos',amb,'atributos',atr),'_decision',dec);
END $$;
SELECT pg_temp.entrada_cierre(:'exp_a',8,ARRAY['cese_registrado','ginpix_confirmado'],'GX-2027-0042','2027-02-16','uno')::text AS cierre_uno \gset
SELECT pg_temp.entrada_cierre(:'exp_b',7,ARRAY['cese_registrado'],'','','b')::text AS cierre_b \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar_cierre(jsonb_set(:'cierre_uno'::jsonb,'{material,ginpix_numero}','""'))->>'error'='22023','GINPIX exigido por la regla');
SELECT pg_temp.exigir(pg_temp.confirmar_cierre(jsonb_set(:'cierre_uno'::jsonb,'{material,condiciones}','["ginpix_confirmado"]'))->>'error'='22023','cierre sin la condición de cese');
SELECT pg_temp.exigir(pg_temp.confirmar_cierre(:'cierre_b'::jsonb)->>'resultado'='sin_cese','cierre sin cese');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar_cierre(:'cierre_uno'::jsonb)::text AS cierre \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.consultar_cese_cierre_expediente_v1((:'cierre_uno'::jsonb)#>>'{material,organizacion_ref}',:'exp_a')::text AS consulta \gset
COMMIT;
SELECT pg_temp.exigir((:'cierre'::jsonb)->>'resultado'='confirmada' AND (:'cierre'::jsonb)#>>'{recibo,estado_resultante}'='completado','cierre del expediente '||:'cierre');
SELECT pg_temp.exigir((:'consulta'::jsonb)#>>'{cese,causa_clave}'='fin_sustitucion' AND (:'consulta'::jsonb)#>>'{cierre,ginpix_numero}'='GX-2027-0042'
  AND (:'consulta'::jsonb)#>>'{incorporacion,inicio}'='2027-01-01','consulta del detalle');
-- La lectura sin decisión propia no da texto libre ni quién actuó, y no
-- sirve para otra organización.
SELECT pg_temp.exigir(NOT ((:'consulta'::jsonb)->'cese' ? 'observaciones') AND NOT ((:'consulta'::jsonb)->'cierre' ? 'observaciones')
  AND (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys((:'consulta'::jsonb)#>'{cese,recibo}') k)=ARRAY['recibo_ref','registrada_en']
  AND (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys((:'consulta'::jsonb)#>'{cierre,recibo}') k)=ARRAY['recibo_ref','registrada_en'],
  'consulta sin texto libre ni actor');
CREATE FUNCTION pg_temp.codigo_consulta_ct115(o text, e text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN PERFORM vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(o, e); RETURN 'ok';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE; END $f$;
SELECT pg_temp.exigir(pg_temp.codigo_consulta_ct115('organizacion:otra:ct115', :'exp_a')='42501','consulta con otra organización denegada');
RESET SESSION AUTHORIZATION;

-- ---------------------------------------------------------------- efectos durables e historia
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version IN (8,9))=2,'dos versiones nuevas');
SELECT pg_temp.exigir((SELECT estado FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version=9)='completado'
  AND (SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=:'exp_a')=9,'proyección actual completada');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_a'
  AND tipo_evento IN ('ct.cese.v1','ct.cierre_expediente.v1'))=2,'dos eventos en el outbox');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.actuacion_expediente_integral WHERE expediente_ref=:'exp_a' AND secuencia IN (8,9))=2,'dos actuaciones');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.publicacion_version_rrhh WHERE expediente_ref=:'exp_a' AND version IN (8,9))=2,'publicación RRHH');
SET ROLE vec_contratacion_temporal_propietario;
DO $$ BEGIN
 BEGIN UPDATE vec_contratacion_temporal.expediente_version_integral SET estado='en_curso' WHERE version=9 AND origen_version='cierre_expediente_ct115';
  RAISE EXCEPTION 'FALLO reescritura de versión';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF; END;
END $$;
RESET ROLE;
SELECT 'CT115 OK' AS resultado;
