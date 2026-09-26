\set ON_ERROR_STOP on
-- CT123 sobre la estructura real restaurada (volcado sintético): el
-- expediente C está en subsanación tras una fiscalización desfavorable (v6).
-- Recorrido: subsanación (CT92) → informe nuevo (CT123) → nueva fiscalización
-- favorable (CT93) ligada al informe nuevo → sigue en fiscalización. Las
-- fachadas AD3-9/10 y la de subsanación se sustituyen por dobles explícitos:
-- se prueba la transacción CT, no la criptografía V3.
\set exp_c 'expediente:ct:9511d16dce57e0ebe1baa849a749f3eed2a29ff7722dc1499132d1d536d66253'

CREATE ROLE vec_ct123_runtime LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
GRANT CONNECT ON DATABASE postgres TO vec_ct123_runtime;
-- Inicio de sesión de gobierno: publica la política al arrancar.
CREATE ROLE vec_ct123_gobierno LOGIN INHERIT IN ROLE vec_contratacion_temporal_gobernador;
DO $dobles$
DECLARE f text; accion text; o oid;
BEGIN
 FOR f, accion IN VALUES
   ('registrar_y_consumir_subsanacion_reparos_v3_atestada','contratacion_temporal.subsanacion_reparos.registrar'),
   ('registrar_y_consumir_informe_juridico_v3_atestada','contratacion_temporal.informe_juridico.generar'),
   ('registrar_y_consumir_fiscalizacion_v3_atestada','contratacion_temporal.fiscalizacion.registrar') LOOP
  o:=to_regprocedure('vec_autorizacion_atestada_v3.'||f||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  EXECUTE format($f$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.%I(%s) RETURNS %s
   LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $c$
   DECLARE d jsonb:=convert_from($2,'UTF8')::jsonb;
   BEGIN
    IF d->>'accion'<>%L THEN RAISE EXCEPTION 'doble: denegado' USING ERRCODE='42501'; END IF;
    RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256($1||$2),'hex'),
      'aud_v3_'||md5($1||$2), clock_timestamp(), true;
   END $c$$f$, f, pg_get_function_arguments(o), pg_get_function_result(o), accion);
 END LOOP;
END $dobles$;

CREATE FUNCTION pg_temp.exigir(c boolean, msg text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF c IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', msg; END IF; RETURN 'ok'; END $$;
CREATE FUNCTION pg_temp.ahora() RETURNS text LANGUAGE sql AS $$
 SELECT to_char(date_trunc('microseconds',clock_timestamp()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') $$;
-- Lectura de apoyo de la prueba (superusuario); el rol de ejecución no lee tablas.
CREATE FUNCTION pg_temp.actual(p_exp text) RETURNS jsonb LANGUAGE sql SECURITY DEFINER AS $$
 SELECT v.agregado_json FROM vec_contratacion_temporal.expediente_integral_actual a
   JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version) WHERE a.expediente_ref=p_exp $$;
CREATE FUNCTION pg_temp.sello(p_dominio text, p_sufijo text) RETURNS text LANGUAGE sql AS $$
 SELECT 'hmac-sha256:vec.contratacion-temporal.'||p_dominio||'/v1:'||encode(sha256(convert_to(p_dominio||p_sufijo,'UTF8')),'hex') $$;
CREATE FUNCTION pg_temp.decision(p_ref text, p_accion text, p_tipo text, p_exp text, p_actor text, p_ctx text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT jsonb_build_object('decision_ref',p_ref,'accion',p_accion,'modulo_id','contratacion_temporal','tipo_recurso',p_tipo,
   'finalidad','gestionar_contratacion_temporal','recurso_ref',p_exp,'principal_id',p_actor,'perfil_activo_ref','prf_ct123',
   'contexto_recurso_huella_sha256',p_ctx) $$;
CREATE FUNCTION pg_temp.autorizacion(decis jsonb) RETURNS jsonb LANGUAGE sql AS $$
 SELECT jsonb_build_object('accion',decis->>'accion','finalidad','gestionar_contratacion_temporal','recurso_ref',decis->>'recurso_ref',
   'principal_id',decis->>'principal_id','perfil_activo_ref','prf_ct123','decision_canonica_hex',encode(convert_to(decis::text,'UTF8'),'hex'),
   'motivo_canonico_hex',encode('\x6d6f7469766f'::bytea,'hex'),'persona_version',1,'perfil_version',1,
   'decision_huella_sha256',encode(sha256(convert_to(decis::text,'UTF8')),'hex'),'decision_ref',decis->>'decision_ref',
   'contexto_recurso_huella_sha256',decis->>'contexto_recurso_huella_sha256') $$;

-- Subsanación del reparo vigente, como la compone el adaptador Go (CT92).
CREATE FUNCTION pg_temp.entrada_subsanacion(p_exp text, p_sufijo text) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb:=pg_temp.actual(p_exp); v numeric:=(ag->>'version')::numeric; inst text:=pg_temp.ahora(); obs text:='Coste corregido con el presupuesto nuevo.';
 amb text:=pg_temp.sello('subsanacion-reparos.ambito',p_sufijo); hue text:=pg_temp.sello('subsanacion-reparos.peticion',p_sufijo);
 ret text:=ag#>>'{fiscalizacion,retorno,retorno_ref}'; refs jsonb; act jsonb; sig jsonb; h text; decis jsonb;
BEGIN
 refs:=jsonb_build_object('reserva_ref','reserva:ct123:'||p_sufijo,'recibo_ref','recibo:ct123:'||p_sufijo,'evento_ref','evento:ct123:'||p_sufijo);
 act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',v+1,
   'accion_clave','contratacion_temporal.subsanacion_reparos.registrar','actor_ref','per_ct123_unidad','unidad_ref',ag#>>'{asignacion,unidad_ref}',
   'recibo_ref',refs->>'recibo_ref','realizada_en',inst,'fase_origen','subsanacion_unidad','fase_destino','subsanacion_unidad',
   'estado_origen','incidencia','estado_destino','incidencia','observaciones',obs,'retorno_ref',ret);
 sig:=ag||jsonb_build_object('version',v+1,'actualizado_en',inst,'actuaciones',(ag->'actuaciones')||jsonb_build_array(act));
 h:=encode(sha256(convert_to('{"ambitos":{"estado_previo":"incidencia","expediente_ref":"'||p_exp||'","fase_previa":"subsanacion_unidad","organizacion_ref":"'
   ||(ag->>'organizacion_ref')||'"},"atributos":{"ambito_idempotencia_hmac":"'||amb||'","huella_peticion_hmac":"'||hue
   ||'","observaciones_huella_sha256":"'||encode(sha256(convert_to(obs,'UTF8')),'hex')||'","politica_huella_sha256":"'||repeat('e',64)
   ||'","politica_ref":"politica:ct:subsanacion:desarrollo","politica_version":"1","responsable_asignado_ref":"'||(ag#>>'{asignacion,responsable_ref}')
   ||'","retorno_ref":"'||ret||'","unidad_asignada_ref":"'||(ag#>>'{asignacion,unidad_ref}')||'","version_expediente":"'||v::text||'"}}','UTF8')),'hex');
 decis:=pg_temp.decision('decision:ct123:'||p_sufijo,'contratacion_temporal.subsanacion_reparos.registrar','subsanacion_reparo_contratacion_temporal',p_exp,'per_ct123_unidad',h);
 RETURN jsonb_build_object('confirmar',jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-subsanacion-reparos.v1','operacion','registrar_subsanacion',
   'material',jsonb_build_object('organizacion_ref',ag->>'organizacion_ref','expediente_ref',p_exp,'version_esperada',v,'actor_ref','per_ct123_unidad','perfil_ref','prf_ct123','observaciones',obs),
   'referencias',refs,'ambito_idempotencia_hmac',amb,'huella_peticion_hmac',hue,'retorno_ref',ret,'expediente_anterior',ag,'expediente_siguiente',sig,'actuacion',act,
   'politica',jsonb_build_object('definicion_ref','politica:ct:subsanacion:desarrollo','definicion_version',1,'definicion_huella_sha256',repeat('e',64),
     'accion','contratacion_temporal.subsanacion_reparos.registrar','finalidad','gestionar_contratacion_temporal','evaluada_en',inst,
     'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
   'autorizacion',pg_temp.autorizacion(decis),'instante_efecto',inst),'_decision',decis);
END $$;

-- Huella canónica del borrador y documento de desarrollo, como CT51.
CREATE FUNCTION pg_temp.huella_borrador(b jsonb) RETURNS text LANGUAGE plpgsql AS $$
DECLARE m bytea; e record; anexos jsonb:=coalesce(nullif(b->'anexos','null'::jsonb),'[]'::jsonb);
BEGIN
 m:=int4send(octet_length(convert_to(b#>>'{canon,esquema}','UTF8')))||convert_to(b#>>'{canon,esquema}','UTF8')||int2send((b#>>'{canon,version_esquema}')::smallint)
  ||int4send(octet_length(convert_to(b->>'expediente_ref','UTF8')))||convert_to(b->>'expediente_ref','UTF8')||int8send((b->>'version_esperada_expediente')::bigint)
  ||int4send(octet_length(convert_to(b#>>'{plantilla,plantilla_ref}','UTF8')))||convert_to(b#>>'{plantilla,plantilla_ref}','UTF8')||int8send((b#>>'{plantilla,version}')::bigint)
  ||int4send(octet_length(convert_to(b#>>'{plantilla,huella_sha256}','UTF8')))||convert_to(b#>>'{plantilla,huella_sha256}','UTF8')
  ||int2send(jsonb_array_length(b->'referencias_normativas')::smallint);
 FOR e IN SELECT n.valor FROM jsonb_array_elements(b->'referencias_normativas') WITH ORDINALITY n(valor,p) ORDER BY n.p LOOP
  m:=m||int4send(octet_length(convert_to(e.valor->>'norma_ref','UTF8')))||convert_to(e.valor->>'norma_ref','UTF8')||int8send((e.valor->>'version')::bigint)
   ||int4send(octet_length(convert_to(e.valor->>'huella_sha256','UTF8')))||convert_to(e.valor->>'huella_sha256','UTF8');
 END LOOP;
 m:=m||int2send(jsonb_array_length(anexos)::smallint);
 RETURN encode(sha256(m),'hex');
END $$;

CREATE FUNCTION pg_temp.entrada_informe(p_exp text, p_sufijo text) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb:=pg_temp.actual(p_exp); v numeric:=(ag->>'version')::numeric; inst text:=pg_temp.ahora();
 amb text:=pg_temp.sello('informe-juridico.ambito',p_sufijo); hue text:=pg_temp.sello('informe-juridico.peticion',p_sufijo);
 ret text:=ag#>>'{fiscalizacion,retorno,retorno_ref}'; refs jsonb; b jsonb; hb text; paquete text; contenido text; doc jsonb;
 act jsonb; inf jsonb; sig jsonb; conf jsonb; h text; decis jsonb; normas text;
BEGIN
 refs:=jsonb_build_object('reserva_ref','reserva:ct123:'||p_sufijo,'informe_ref','informe:ct123:'||p_sufijo,'documento_ref','documento:ct123:'||p_sufijo,
   'recibo_ref','recibo:ct123:'||p_sufijo,'auditoria_ref','auditoria:ct123:'||p_sufijo,'evento_ref','evento:ct123:'||p_sufijo);
 b:=jsonb_set((ag#>'{informe_juridico,borrador}')-'huella_sha256','{version_esperada_expediente}',to_jsonb(v));
 hb:=pg_temp.huella_borrador(b); b:=b||jsonb_build_object('huella_sha256',hb);
 SELECT string_agg('{"norma_ref":"'||(n.x->>'norma_ref')||'","version":'||(n.x->>'version')||',"huella_sha256":"'||(n.x->>'huella_sha256')||'"}',',' ORDER BY n.p)
   INTO normas FROM jsonb_array_elements(b->'referencias_normativas') WITH ORDINALITY n(x,p);
 paquete:='{"esquema":"vec.dipgra.contratacion-temporal.informe-juridico.paquete-datos","version_esquema":1,"expediente_ref":"'||p_exp
   ||'","version_expediente":'||v::text||',"plantilla":{"plantilla_ref":"'||(b#>>'{plantilla,plantilla_ref}')||'","version":'||(b#>>'{plantilla,version}')
   ||',"huella_sha256":"'||(b#>>'{plantilla,huella_sha256}')||'"},"referencias_normativas":['||normas||'],"anexos":[],"huella_borrador_sha256":"'||hb||'"}';
 contenido:='DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA'||chr(10)||chr(10)||'INFORME JURIDICO PROVISIONAL'||chr(10)
   ||'Pendiente de revision y firma.'||chr(10)||chr(10)||'Datos juridicos canónicos:'||chr(10)||paquete||chr(10);
 doc:=jsonb_build_object('documento_ref',refs->>'documento_ref','version_documento',1,'formato','text/plain; charset=utf-8',
   'nombre','informe-juridico-desarrollo.txt','huella_documento_sha256',encode(sha256(convert_to(contenido,'UTF8')),'hex'),
   'huella_paquete_sha256',encode(sha256(convert_to(paquete,'UTF8')),'hex'),'contenido_desarrollo',contenido);
 act:=jsonb_build_object('secuencia',v+1,'version_expediente',v+1,'accion_clave','contratacion_temporal.informe_juridico.generar',
   'actor_ref','per_ct123_juridico','unidad_ref',ag#>>'{asignacion,unidad_ref}','recibo_ref',refs->>'recibo_ref','realizada_en',inst,
   'fase_origen','subsanacion_unidad','fase_destino','subsanacion_unidad','estado_origen','incidencia','estado_destino','incidencia',
   'documentos_ref',jsonb_build_array(refs->>'documento_ref'),'retorno_ref',ret);
 inf:=jsonb_build_object('borrador',b,'informe_ref',refs->>'informe_ref','documento_ref',refs->>'documento_ref','version_documento',1,
   'huella_documento_sha256',doc->>'huella_documento_sha256','emitido_en',inst,
   'sustituye',jsonb_build_object('informe_ref',ag#>>'{informe_juridico,informe_ref}','documento_ref',ag#>>'{informe_juridico,documento_ref}','retorno_ref',ret),
   'actuacion_registro',jsonb_build_object('secuencia',v+1,'version_expediente',v+1,'accion_clave','contratacion_temporal.informe_juridico.generar',
     'fase_destino','subsanacion_unidad','recibo_ref',refs->>'recibo_ref','informe_ref',refs->>'informe_ref','documento_ref',refs->>'documento_ref',
     'version_documento',1,'huella_documento_sha256',doc->>'huella_documento_sha256','huella_borrador_sha256',hb));
 sig:=ag||jsonb_build_object('version',v+1,'fase_actual','subsanacion_unidad','estado_actual','incidencia','actualizado_en',inst,
   'actuaciones',(ag->'actuaciones')||jsonb_build_array(act),'informe_juridico',inf);
 conf:=jsonb_build_object('plantilla',b->'plantilla','referencias_normativas',b->'referencias_normativas','anexos',b->'anexos',
   'definicion_ref','configuracion:ct:desarrollo:informe-juridico:v1','definicion_version',1,'definicion_huella_sha256',repeat('d',64),
   'accion','contratacion_temporal.informe_juridico.generar','finalidad','gestionar_contratacion_temporal','unidad_ejecutora_ref',ag#>>'{asignacion,unidad_ref}',
   'evaluada_en',inst,'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 h:=encode(sha256(convert_to('{"ambitos":{"estado_previo":"incidencia","expediente_ref":"'||p_exp||'","fase_previa":"subsanacion_unidad","organizacion_ref":"'
   ||(ag->>'organizacion_ref')||'"},"atributos":{"ambito_idempotencia_hmac":"'||amb||'","borrador_huella_sha256":"'||hb
   ||'","configuracion_huella_sha256":"'||repeat('d',64)||'","configuracion_ref":"configuracion:ct:desarrollo:informe-juridico:v1","configuracion_version":"1"'
   ||',"huella_peticion_hmac":"'||hue||'","plantilla_huella_sha256":"'||(b#>>'{plantilla,huella_sha256}')||'","plantilla_ref":"'||(b#>>'{plantilla,plantilla_ref}')
   ||'","plantilla_version":"'||(b#>>'{plantilla,version}')||'","version_expediente":"'||v::text||'"}}','UTF8')),'hex');
 decis:=pg_temp.decision('decision:ct123:'||p_sufijo,'contratacion_temporal.informe_juridico.generar','informe_juridico_contratacion_temporal',p_exp,'per_ct123_juridico',h);
 RETURN jsonb_build_object(
  'preparar',jsonb_build_object('esquema','vec.contratacion-temporal.preparar-informe-juridico.v1','operacion','preparar',
    'sellos_hmac',jsonb_build_object('activo',jsonb_build_object('ambito_hmac',amb,'generacion',1,'huella_peticion_hmac',hue),'retenidos','[]'::jsonb),
    'organizacion_ref',ag->>'organizacion_ref','expediente_ref',p_exp,'version_expediente',v,'actor_ref','per_ct123_juridico','perfil_ref','prf_ct123',
    'referencias_candidatas',refs),
  'confirmar',jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-informe-juridico.v1','operacion','preparar','reserva_ref',refs->>'reserva_ref',
    'referencias',refs,'ambito_idempotencia_hmac',amb,'huella_peticion_hmac',hue,'organizacion_ref',ag->>'organizacion_ref','expediente_ref',p_exp,
    'version_anterior',v,'actor_ref','per_ct123_juridico','perfil_ref','prf_ct123','expediente_anterior',ag,'expediente_siguiente',sig,'actuacion',act,
    'configuracion',conf,'borrador',b,'documento',doc,'autorizacion',pg_temp.autorizacion(decis),'instante_efecto',inst),
  '_decision',decis);
END $$;

-- Nueva fiscalización tras subsanar (CT93), con el informe vigente.
CREATE FUNCTION pg_temp.entrada_fiscalizacion(p_exp text, p_resultado text, p_obs text, p_sufijo text) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE ag jsonb:=pg_temp.actual(p_exp); v numeric:=(ag->>'version')::numeric; inst text:=pg_temp.ahora(); unidad text:='unidad:intervencion:desarrollo';
 amb text:=pg_temp.sello('fiscalizacion.ambito',p_sufijo); hue text:=pg_temp.sello('fiscalizacion.peticion',p_sufijo);
 ret text:=ag#>>'{fiscalizacion,retorno,retorno_ref}'; sub text; refs jsonb; fase text:='fiscalizacion'; estado text:='en_curso';
 act jsonb; vin jsonb; fis jsonb; sig jsonb; h text; decis jsonb;
BEGIN
 SELECT x->>'recibo_ref' INTO STRICT sub FROM jsonb_array_elements(ag->'actuaciones') x
  WHERE x->>'accion_clave'='contratacion_temporal.subsanacion_reparos.registrar' AND x->>'retorno_ref'=ret;
 refs:=jsonb_build_object('reserva_ref','reserva:ct123:'||p_sufijo,'fiscalizacion_ref','fiscalizacion:ct123:'||p_sufijo,
   'recibo_ref','recibo:ct123:'||p_sufijo,'evento_ref','evento:ct123:'||p_sufijo,'retorno_ref','');
 act:=jsonb_build_object('secuencia',v+1,'version_expediente',v+1,'accion_clave','contratacion_temporal.fiscalizacion.registrar',
   'actor_ref','per_ct123_interventor','unidad_ref',unidad,'recibo_ref',refs->>'recibo_ref','realizada_en',inst,
   'fase_origen','subsanacion_unidad','fase_destino',fase,'estado_origen','incidencia','estado_destino',estado,
   'documentos_ref',jsonb_build_array(ag#>>'{informe_juridico,documento_ref}'),'retorno_ref',ret);
 vin:=jsonb_build_object('secuencia',v+1,'version_expediente',v+1,'accion_clave','contratacion_temporal.fiscalizacion.registrar',
   'fase_destino',fase,'estado_destino',estado,'recibo_ref',refs->>'recibo_ref','fiscalizacion_ref',refs->>'fiscalizacion_ref','resultado',p_resultado,
   'unidad_fiscalizadora_ref',unidad,'informe_juridico_ref',ag#>>'{informe_juridico,informe_ref}','documento_informe_ref',ag#>>'{informe_juridico,documento_ref}');
 fis:=jsonb_build_object('fiscalizacion_ref',refs->>'fiscalizacion_ref','resultado',p_resultado,'unidad_fiscalizadora_ref',unidad,
   'informe_juridico_ref',ag#>>'{informe_juridico,informe_ref}','documento_informe_ref',ag#>>'{informe_juridico,documento_ref}',
   'fiscalizada_en',inst,'actuacion_registro',vin);
 sig:=ag||jsonb_build_object('version',v+1,'fase_actual',fase,'estado_actual',estado,'actualizado_en',inst,
   'actuaciones',(ag->'actuaciones')||jsonb_build_array(act),'fiscalizacion',fis);
 h:=encode(sha256(convert_to('{"ambitos":{"estado_previo":"incidencia","expediente_ref":"'||p_exp||'","fase_previa":"subsanacion_unidad","organizacion_ref":"'
   ||(ag->>'organizacion_ref')||'"},"atributos":{"ambito_idempotencia_hmac":"'||amb||'","documento_informe_ref":"'||(ag#>>'{informe_juridico,documento_ref}')
   ||'","huella_peticion_hmac":"'||hue||'","informe_juridico_ref":"'||(ag#>>'{informe_juridico,informe_ref}')
   ||'","observaciones_huella_sha256":"'||encode(sha256(convert_to(p_obs,'UTF8')),'hex')||'","politica_huella_sha256":"'||repeat('f',64)
   ||'","politica_ref":"politica:ct:fiscalizacion:desarrollo","politica_version":"1","responsable_asignado_ref":"'||(ag#>>'{asignacion,responsable_ref}')
   ||'","resultado":"'||p_resultado||'","retorno_previo_ref":"'||ret||'","subsanacion_recibo_ref":"'||sub
   ||'","unidad_asignada_ref":"'||(ag#>>'{asignacion,unidad_ref}')||'","unidad_fiscalizadora_ref":"'||unidad||'","version_expediente":"'||v::text||'"}}','UTF8')),'hex');
 decis:=pg_temp.decision('decision:ct123:'||p_sufijo,'contratacion_temporal.fiscalizacion.registrar','fiscalizacion_contratacion_temporal',p_exp,'per_ct123_interventor',h);
 RETURN jsonb_build_object('confirmar',jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-fiscalizacion.v1','operacion','registrar_resultado',
    'reserva_ref',refs->>'reserva_ref','referencias',refs,'ambito_idempotencia_hmac',amb,'huella_peticion_hmac',hue,
    'organizacion_ref',ag->>'organizacion_ref','expediente_ref',p_exp,'version_anterior',v,'actor_ref','per_ct123_interventor','perfil_ref','prf_ct123',
    'resultado',p_resultado,'observaciones',p_obs,'expediente_anterior',ag,'expediente_siguiente',sig,'actuacion',act,
    'politica',jsonb_build_object('definicion_ref','politica:ct:fiscalizacion:desarrollo','definicion_version',1,'definicion_huella_sha256',repeat('f',64),
      'accion','contratacion_temporal.fiscalizacion.registrar','finalidad','gestionar_contratacion_temporal','unidad_fiscalizadora_ref',unidad,
      'evaluada_en',inst,'valida_hasta',to_char((clock_timestamp()+interval '5 minutes') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
    'autorizacion',pg_temp.autorizacion(decis),'instante_efecto',inst),'_decision',decis);
END $$;

CREATE FUNCTION pg_temp.llamar(p_funcion text, e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb;
BEGIN
 EXECUTE format('SELECT to_jsonb(x) FROM vec_contratacion_temporal.%I($1,%L,$2,%L,%L,1,1,%L,%L,%L,%L) x', p_funcion,
   '\x6361706163696461640a','\x6d6f7469766f','\x01','\x01','\x01','\x01','\x01')
   INTO STRICT r USING e->'confirmar', convert_to((e->'_decision')::text,'UTF8');
 RETURN r;
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
CREATE FUNCTION pg_temp.preparar_informe(e jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(e->'preparar');
 RETURN jsonb_build_object('resultado',r.resultado,'estado',r.estado,'recibo',r.recibo_json,'version',r.version_expediente);
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
CREATE FUNCTION pg_temp.ronda(p_org text, p_exp text) RETURNS jsonb LANGUAGE plpgsql AS $$
BEGIN RETURN to_jsonb(vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(p_org,p_exp));
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE); END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct123_runtime, vec_ct123_gobierno',pg_my_temp_schema()::regnamespace); END $$;

SELECT pg_temp.actual(:'exp_c')->>'organizacion_ref' AS org \gset
SELECT pg_temp.exigir((pg_temp.actual(:'exp_c')->>'version')='6' AND pg_temp.actual(:'exp_c')->>'fase_actual'='subsanacion_unidad','antecedente: reparo desfavorable en v6');
SELECT pg_temp.actual(:'exp_c')#>>'{informe_juridico,informe_ref}' AS informe_inicial \gset

SET SESSION AUTHORIZATION vec_ct123_runtime;
-- Antes de subsanar no cabe informe nuevo (la preparación exige v7 o más).
SELECT pg_temp.entrada_informe(:'exp_c','antes')::text AS antes \gset
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.exigir(pg_temp.preparar_informe(:'antes'::jsonb)->>'error'='42501','informe nuevo antes de subsanar');
COMMIT;
SELECT pg_temp.exigir(pg_temp.ronda(:'org',:'exp_c')='0'::jsonb,'sin informe nuevo, ronda única');
SELECT pg_temp.exigir(pg_temp.ronda('organizacion:ajena:01',:'exp_c')->>'error'='42501','ronda de otra organización denegada');

-- Subsanación (CT92) → v7.
SELECT pg_temp.entrada_subsanacion(:'exp_c','sub')::text AS sub \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.llamar('confirmar_subsanacion_reparos_v1',:'sub'::jsonb)::text AS conf_sub \gset
COMMIT;
SELECT pg_temp.exigir((:'conf_sub'::jsonb)->>'resultado'='confirmada','subsanación confirmada '||:'conf_sub');

-- Sin catálogo, la nueva fiscalización con el mismo informe sigue cabiendo
-- (conducta de siempre; la exigencia del informe nuevo vive en la aplicación).
SELECT pg_temp.entrada_fiscalizacion(:'exp_c','favorable','','mismo')::text AS mismo \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.llamar('confirmar_fiscalizacion_tras_subsanacion_v1',:'mismo'::jsonb)::text AS conf_mismo \gset
SELECT pg_temp.exigir((:'conf_mismo'::jsonb) ? 'recibo_ref','nueva fiscalización con el mismo informe (sin catálogo) '||:'conf_mismo');
ROLLBACK;

-- Política publicada por el gobierno al arrancar (desde la regla c19): no
-- exigir sin historia no escribe; exigir publica la versión 1 y repetirla no
-- añade otra. Nadie más la publica ni la lee.
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION vec_ct123_gobierno;
SELECT pg_temp.exigir(r.resultado='vigente' AND r.version=0,'no exigir sin historia no escribe')
  FROM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(false,'vec.contratacion_temporal.reglas:1:c19.fiscalizacion_resultados') r;
SELECT pg_temp.exigir(r.resultado='publicada' AND r.version=1,'política que exige informe nuevo publicada')
  FROM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(true,'vec.contratacion_temporal.reglas:1:c19.fiscalizacion_resultados') r;
SELECT pg_temp.exigir(r.resultado='vigente' AND r.version=1,'segundo arranque no republica')
  FROM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(true,'vec.contratacion_temporal.reglas:2:c19.fiscalizacion_resultados') r;
DO $$ BEGIN
 BEGIN PERFORM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(true,'x');
  RAISE EXCEPTION 'FALLO fuente no válida aceptada'; EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN PERFORM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(NULL,'configuracion:ct:prueba');
  RAISE EXCEPTION 'FALLO política nula aceptada'; EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN PERFORM 1 FROM vec_contratacion_temporal.politica_informe_tras_subsanacion;
  RAISE EXCEPTION 'FALLO el gobernador lee la política'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_contratacion_temporal.informe_nuevo_exigido_ct123();
  RAISE EXCEPTION 'FALLO el gobernador consulta la exigencia'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $$;
RESET SESSION AUTHORIZATION;
DO $$ BEGIN
 BEGIN UPDATE vec_contratacion_temporal.politica_informe_tras_subsanacion SET exige_informe_nuevo=false;
  RAISE EXCEPTION 'FALLO' USING ERRCODE='P0099'; EXCEPTION WHEN SQLSTATE 'P0099' THEN RAISE EXCEPTION 'FALLO política modificable'; WHEN others THEN NULL; END;
 BEGIN DELETE FROM vec_contratacion_temporal.politica_informe_tras_subsanacion;
  RAISE EXCEPTION 'FALLO' USING ERRCODE='P0099'; EXCEPTION WHEN SQLSTATE 'P0099' THEN RAISE EXCEPTION 'FALLO política borrable'; WHEN others THEN NULL; END;
END $$;
SET SESSION AUTHORIZATION vec_ct123_runtime;
DO $$ BEGIN
 BEGIN PERFORM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(false,'configuracion:ct:prueba');
  RAISE EXCEPTION 'FALLO el ejecutor publica'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $$;
-- Con esa política, la base rechaza fiscalizar de nuevo con el mismo informe
-- (directo y por la fachada v2 de CT120), sin efecto alguno.
SELECT pg_temp.entrada_fiscalizacion(:'exp_c','favorable','','exigida')::text AS exigida \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.llamar('confirmar_fiscalizacion_tras_subsanacion_v1',:'exigida'::jsonb)::text AS conf_exigida \gset
SELECT pg_temp.exigir((:'conf_exigida'::jsonb)->>'error'='55000'
   AND (:'conf_exigida'::jsonb)->>'mensaje'='informe jurídico nuevo tras subsanación pendiente','sin informe nuevo la base no fiscaliza '||:'conf_exigida');
SELECT pg_temp.llamar('confirmar_fiscalizacion_v2',:'exigida'::jsonb)::text AS conf_exigida_v2 \gset
SELECT pg_temp.exigir(to_regprocedure('vec_contratacion_temporal.confirmar_fiscalizacion_v2(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
   OR (:'conf_exigida_v2'::jsonb)->>'error'='55000','tampoco por la fachada v2 (si CT120 está) '||:'conf_exigida_v2');
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir(pg_temp.actual(:'exp_c')->>'version'='7'
   AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.reserva_fiscalizacion WHERE reserva_ref='reserva:ct123:exigida'),'rechazo sin efectos');
SET SESSION AUTHORIZATION vec_ct123_runtime;

-- Informe nuevo (CT123): preparación de lectura, negativos y confirmación → v8.
SELECT pg_temp.entrada_informe(:'exp_c','nuevo')::text AS inf \gset
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar_informe(:'inf'::jsonb)::text AS prep \gset
COMMIT;
SELECT pg_temp.exigir((:'prep'::jsonb)->>'resultado'='reservada' AND (:'prep'::jsonb)->>'version'='7','preparación del informe nuevo '||:'prep');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.preparar_informe(:'inf'::jsonb)->>'error'='42501','preparación fuera de solo lectura');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.llamar('confirmar_informe_juridico_tras_subsanacion_v1',
  jsonb_set(:'inf'::jsonb,'{confirmar,actuacion,fase_destino}','"informe_juridico"'))->>'error'='22023','proyección que sale de la subsanación');
SELECT pg_temp.exigir(pg_temp.llamar('confirmar_informe_juridico_tras_subsanacion_v1',
  (:'inf'::jsonb) #- '{confirmar,expediente_siguiente,informe_juridico,sustituye}')->>'error'='22023','informe sin la sustitución');
SELECT pg_temp.exigir(pg_temp.llamar('confirmar_informe_juridico_tras_subsanacion_v1',
  jsonb_set(:'inf'::jsonb,'{confirmar,autorizacion,contexto_recurso_huella_sha256}',to_jsonb(repeat('0',64))))->>'error' IN ('22023','42501'),'contexto autorizado ajeno');
SELECT pg_temp.exigir(pg_temp.llamar('confirmar_informe_juridico_v1',:'inf'::jsonb)->>'error' IS NOT NULL,'CT51 sigue sin admitir el informe nuevo');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.llamar('confirmar_informe_juridico_tras_subsanacion_v1',:'inf'::jsonb)::text AS conf_inf \gset
COMMIT;
SELECT pg_temp.exigir((:'conf_inf'::jsonb) ? 'recibo_ref','informe nuevo confirmado '||:'conf_inf');
-- Idempotencia: la preparación recupera el mismo recibo y la confirmación
-- repetida lo devuelve sin otro efecto.
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar_informe(:'inf'::jsonb)::text AS recup \gset
COMMIT;
SELECT pg_temp.exigir((:'recup'::jsonb)->>'resultado'='confirmada' AND (:'recup'::jsonb)->'recibo'=(:'conf_inf'::jsonb),'recuperación con el mismo recibo');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.llamar('confirmar_informe_juridico_tras_subsanacion_v1',:'inf'::jsonb)=(:'conf_inf'::jsonb),'confirmación repetida devuelve el mismo recibo');
COMMIT;
-- Un segundo informe nuevo para el mismo reparo se rechaza.
SELECT pg_temp.entrada_informe(:'exp_c','otro')::text AS otro \gset
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.exigir(pg_temp.preparar_informe(:'otro'::jsonb)->>'error'='40001','segundo informe nuevo para el mismo reparo');
COMMIT;
SELECT pg_temp.exigir(pg_temp.ronda(:'org',:'exp_c')='8'::jsonb,'la ronda de firma empieza en v8');
RESET SESSION AUTHORIZATION;

SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_c' AND version=8
   AND fase_clave='subsanacion_unidad' AND estado='incidencia')=1,'v8 sigue en subsanación');
SELECT pg_temp.exigir(pg_temp.actual(:'exp_c')#>>'{informe_juridico,sustituye,informe_ref}'=:'informe_inicial'
   AND pg_temp.actual(:'exp_c')#>>'{fiscalizacion,informe_juridico_ref}'=:'informe_inicial'
   AND pg_temp.actual(:'exp_c')#>>'{informe_juridico,informe_ref}'='informe:ct123:nuevo','informe nuevo sustituye al fiscalizado');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_c' AND version_expediente=8
   AND tipo_evento='contratacion_temporal.informe_juridico_emitido')=1,'evento del informe nuevo');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.documento_informe_juridico_desarrollo WHERE informe_ref='informe:ct123:nuevo')=1,'documento del informe nuevo');

-- Nueva fiscalización favorable (CT93) con el informe nuevo → v9 y sigue.
SET SESSION AUTHORIZATION vec_ct123_runtime;
SELECT pg_temp.entrada_fiscalizacion(:'exp_c','favorable','','fav')::text AS fav \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.llamar('confirmar_fiscalizacion_tras_subsanacion_v1',:'fav'::jsonb)::text AS conf_fav \gset
COMMIT;
SELECT pg_temp.exigir((:'conf_fav'::jsonb) ? 'recibo_ref','fiscalización favorable '||:'conf_fav');
SELECT pg_temp.entrada_informe(:'exp_c','tarde')::text AS tarde \gset
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.exigir(pg_temp.preparar_informe(:'tarde'::jsonb)->>'error'='40001','informe nuevo tras la fiscalización favorable');
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir(pg_temp.actual(:'exp_c')->>'version'='9' AND pg_temp.actual(:'exp_c')->>'fase_actual'='fiscalizacion'
   AND pg_temp.actual(:'exp_c')#>>'{fiscalizacion,resultado}'='favorable'
   AND pg_temp.actual(:'exp_c')#>>'{fiscalizacion,informe_juridico_ref}'='informe:ct123:nuevo'
   AND pg_temp.actual(:'exp_c')#>>'{fiscalizacion,documento_informe_ref}'='documento:ct123:nuevo','favorable ligada al informe nuevo');
-- Volver a no exigir añade la versión 2 (la historia se conserva) y deja la
-- conducta de siempre para las pruebas que siguen en la misma base.
SET SESSION AUTHORIZATION vec_ct123_gobierno;
SELECT pg_temp.exigir(r.resultado='publicada' AND r.version=2,'vuelta a no exigir publicada')
  FROM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(false,'configuracion:ct:informe-tras-subsanacion:predeterminada') r;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.politica_informe_tras_subsanacion)=2
   AND NOT vec_contratacion_temporal.informe_nuevo_exigido_ct123(),'historia de la política');
SELECT 'CT123 OK' AS resultado;
