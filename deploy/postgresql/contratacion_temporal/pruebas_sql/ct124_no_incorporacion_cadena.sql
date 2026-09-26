\set ON_ERROR_STOP on
-- Cadena «aceptación → no incorporación → baja en Bolsa → siguiente
-- llamamiento» sobre la estructura real restaurada (CT124 y Bolsa 000042
-- instaladas; fixture ct124_no_incorporacion_fixture.sql). El expediente A
-- tiene su aceptación confirmada por RRHH, su propuesta de nombramiento
-- (versión 7) y ninguna incorporación. Requiere antes ct124_utilidades.sql
-- en la misma sesión. Dobles AD3: se prueban las transacciones, no la
-- criptografía V3. Datos sintéticos; base desechable.
\set exp_a 'expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
\set clave_cont '44444444-4444-4444-8444-000000000042'

SELECT agregado_json->>'organizacion_ref' AS org FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version=7 \gset
SELECT r.resolucion_ref AS acept_a, r.llamamiento_ref AS llam_a
  FROM vec_contratacion_temporal.propuesta_formalizacion pf JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r USING (resolucion_ref)
 WHERE pf.expediente_ref=:'exp_a' \gset
SELECT l.operacion_ref AS apertura_a, i.operacion_ref AS terminal_a
  FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
  JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.apertura_operacion_ref=l.operacion_ref AND i.tipo='aceptacion_rrhh'
 WHERE l.llamamiento_ref=:'llam_a' \gset

-- Entrada de la no incorporación, igual que el adaptador Go.
CREATE FUNCTION pg_temp.entrada_no_inc(p_exp text, p_version numeric, p_sufijo text, p_resuelta text, p_acept text,
  p_segunda boolean DEFAULT true, p_fecha text DEFAULT '2026-09-20') RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE e jsonb; m jsonb:=jsonb_build_object('motivo_clave','no_presentado','consecuencia_clave','b24.sancion.baja_llamamiento_directo',
  'resolucion_ref','resolucion:rrhh:2026/0142','resolucion_sha256',repeat('b',64),'resuelta_por',p_resuelta,'segunda_persona',p_segunda,
  'fecha_notificacion',p_fecha,'observaciones','');
BEGIN
 e:=pg_temp.entrada('registrar_no_incorporacion','no-incorporacion','contratacion_temporal.incorporacion.no_incorporacion',
   'no_incorporacion_contratacion_temporal','registrar_no_incorporacion_contratacion_temporal',p_exp,p_version,m,
   (m-'segunda_persona'-'observaciones')||jsonb_build_object('segunda_persona',CASE WHEN p_segunda THEN 'si' ELSE 'no' END,
     'observaciones_huella_sha256',encode(sha256(convert_to('','UTF8')),'hex'),'aceptacion_ref',p_acept),
   jsonb_build_object('fase_destino','fiscalizacion','documentos_ref',jsonb_build_array('resolucion:rrhh:2026/0142')),'en_curso',p_sufijo);
 RETURN jsonb_set(e,'{expediente_siguiente,fase_actual}','"fiscalizacion"')
   ||jsonb_build_object('esquema','vec.contratacion-temporal.confirmar-no-incorporacion.v1');
END $$;
-- Continuación (CT119): material en el orden de json.Marshal de Go.
CREATE FUNCTION pg_temp.solicitud_cont(p_clave text, p_org text, p_exp text, p_res text, p_int text) RETURNS text LANGUAGE sql AS $$
 SELECT format('{"ClaveIdempotencia":"%s","OrganizacionRef":"%s","ExpedienteRef":"%s","ResolucionRef":"%s","IntencionRef":"%s"}',
   p_clave,p_org,p_exp,p_res,p_int) $$;
CREATE FUNCTION pg_temp.continuar(p_material text) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb; d bytea:=convert_to(jsonb_build_object('accion','contratacion_temporal.llamamiento.siguiente.continuar',
   'modulo_id','contratacion_temporal','tipo_recurso','continuacion_llamamiento_ct','finalidad','gestionar_contratacion_temporal',
   'recurso_ref',(p_material::jsonb)->'Solicitud'->>'ExpedienteRef',
   'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||((p_material::jsonb)->'Solicitud'->>'OrganizacionRef')||
     '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
   'principal_id','per_ct124_actor','perfil_activo_ref','prf_ct124')::text,'UTF8');
BEGIN
 r:=vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(p_material,'\x01'::bytea,d,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 RETURN r;
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
-- Bolsa: capacidad estructural ligada al registro exacto y guardado real.
CREATE FUNCTION pg_temp.guardar_bolsa(p_registro bytea, p_operacion text) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE r jsonb:=convert_from(p_registro,'UTF8')::jsonb; c bytea; g record;
BEGIN
 c:=convert_to(jsonb_build_object('efecto_ref',r->>'operacion_ref',
   'huella_efecto_sha256',encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||to_json(r->>'categoria_ref')::text||
     ',"unidad_ref":'||to_json(r->>'unidad_ref')::text||'},"atributos":{"contenido_sha256":'||
     to_json(encode(sha256(p_registro),'hex'))::text||',"necesidad_ref":'||to_json(r->>'necesidad_ref')::text||'}}','UTF8')),'hex'),
   'operacion',p_operacion,'audiencia_consumo','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','relleno',repeat('x',600))::text,'UTF8');
 SELECT * INTO STRICT g FROM vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(p_registro,c,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
 RETURN jsonb_build_object('recibo_ref',g.recibo_ref,'evento_ref',g.evento_ref,'confirmada_en',g.confirmada_en);
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
-- Siguiente llamamiento Bolsa desde el terminal de aceptación: el mismo
-- orden, fuente e instantánea; evalúa el tramo contiguo posterior.
CREATE FUNCTION pg_temp.siguiente_bolsa(p_apertura text, p_terminal text, p_intencion text, p_op text, p_llam text) RETURNS bytea LANGUAGE plpgsql AS $$
DECLARE r0 jsonb; t bytea; i jsonb; f jsonb; d int; ent jsonb; sit jsonb; regla jsonb; e jsonb; p jsonb;
BEGIN
 SELECT convert_from(registro_canonico,'UTF8')::jsonb INTO STRICT r0 FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref=p_apertura;
 SELECT registro_canonico INTO STRICT t FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref=p_terminal;
 i:=r0->'instantanea'; f:=r0->'fuente'; d:=(r0#>>'{propuesta,orden_seleccionado}')::int; ent:=i->'entradas'->d;
 SELECT s INTO STRICT sit FROM jsonb_array_elements(ent#>'{participacion,situaciones}') s
  WHERE (s->>'desde')::timestamptz<=(i->>'referida_en')::timestamptz AND (s->>'hasta' IS NULL OR (i->>'referida_en')::timestamptz<(s->>'hasta')::timestamptz);
 SELECT g INTO STRICT regla FROM jsonb_array_elements(f->'reglas') g
  WHERE g->>'estado_clave'=sit->>'estado_clave' AND g->>'estado_version'=sit->>'estado_version' AND g->>'huella_estado_sha256'=sit->>'huella_estado_sha256';
 e:=(r0#>'{propuesta,evaluaciones}'->-1)||jsonb_build_object('orden',d+1,'participacion_ref',ent#>>'{participacion,participacion_ref}',
   'sujeto_ref',ent#>>'{participacion,sujeto_ref}','estado_clave',sit->>'estado_clave','estado_version',sit->'estado_version',
   'huella_estado_sha256',sit->>'huella_estado_sha256','resultado',regla->>'resultado','situacion_secuencia',sit->'secuencia');
 p:=(r0->'propuesta')||jsonb_build_object('propuesta_ref','propuesta:prueba:b42',
   'generada_en',to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evaluaciones',jsonb_build_array(e),
   'orden_seleccionado',d+1,'participacion_seleccionada_ref',e->>'participacion_ref','sujeto_seleccionado_ref',e->>'sujeto_ref',
   'continuacion',jsonb_build_object('terminal_operacion_ref',p_terminal,'terminal_sha256',encode(sha256(t),'hex'),
     'propuesta_ref',r0#>>'{propuesta,propuesta_ref}','propuesta_sha256',r0#>>'{propuesta,huella_contenido_sha256}',
     'orden_anterior',d,'intencion_ref',p_intencion));
 RETURN convert_to((r0||jsonb_build_object('operacion_ref',p_op,'propuesta',p,
   'llamamiento',(r0->'llamamiento')||jsonb_build_object('LlamamientoRef',p_llam,'PropuestaRef','propuesta:prueba:b42')))::text,'UTF8');
END $$;
CREATE FUNCTION pg_temp.no_inc_bolsa(p_evento jsonb, p_huella text, p_creada timestamptz, p_pos bigint, p_cons jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE g record;
BEGIN
 SELECT * INTO STRICT g FROM vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(p_evento,p_huella,p_creada,p_pos,p_cons);
 RETURN to_jsonb(g);
EXCEPTION WHEN others THEN RETURN jsonb_build_object('error',SQLSTATE,'mensaje',SQLERRM);
END $$;
CREATE FUNCTION pg_temp.consecuencia(p_efecto text) RETURNS jsonb LANGUAGE sql AS $$
 SELECT jsonb_build_object('clave','b24.sancion.baja_llamamiento_directo','etiqueta','Baja por no incorporarse tras aceptar el llamamiento',
   'efecto',p_efecto,'regla_ref','vec.bolsa.reglas:3:b24.sancion.baja_llamamiento_directo','regla_huella_sha256',repeat('c',64),
   'suspension_hasta',NULL,'recurso_vence','2026-10-20','recurso_regla_ref','vec.bolsa.reglas:3:b24.consecuencias',
   'recurso_regla_huella_sha256',repeat('c',64),'orden_final',false,'fin_automatico',false) $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct115_runtime, vec_b42_runtime',pg_my_temp_schema()::regnamespace); END $$;

SELECT pg_temp.entrada_no_inc(:'exp_a',7,'ni1','per_ct124_segunda',:'acept_a')::text AS ni \gset
SELECT pg_temp.entrada_no_inc(:'exp_a',7,'ni_misma','per_ct124_actor',:'acept_a')::text AS ni_misma \gset
SELECT pg_temp.entrada_no_inc(:'exp_b',7,'ni_b','per_ct124_segunda',:'acept_a')::text AS ni_b \gset
SELECT pg_temp.entrada_no_inc(:'exp_a',7,'ni_futura','per_ct124_segunda',:'acept_a',true,'2099-01-01')::text AS ni_futura \gset
SELECT pg_temp.entrada_no_inc(:'exp_a',7,'ni_otra','per_ct124_segunda',:'acept_a')::text AS ni_otra \gset

-- ---------------------------------------------------------------- Bolsa: sin no incorporación no hay siguiente
SET SESSION AUTHORIZATION vec_b42_runtime;
SET TimeZone='UTC'; SET statement_timeout='15s'; SET idle_in_transaction_session_timeout='20s';
RESET SESSION AUTHORIZATION;
SELECT convert_from(pg_temp.siguiente_bolsa(:'apertura_a',:'terminal_a','intencion-siguiente-bolsa:prueba:b42',
  'operacion-siguiente-rrhh:prueba:b42','llamamiento:prueba:b42'),'UTF8') AS sig_bolsa \gset
SET SESSION AUTHORIZATION vec_b42_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.guardar_bolsa(convert_to(:'sig_bolsa','UTF8'),'bolsa.llamamiento.siguiente.abrir')->>'error'='42501',
  'una aceptación sin no incorporación no abre el siguiente llamamiento');
ROLLBACK;
RESET SESSION AUTHORIZATION;

-- ---------------------------------------------------------------- CT: no incorporación
SET SESSION AUTHORIZATION vec_ct115_runtime;
SELECT pg_temp.exigir(NOT has_table_privilege('vec_contratacion_temporal.no_incorporacion_v1','SELECT')
  AND NOT has_function_privilege('vec_contratacion_temporal.aceptacion_vigente_ct124(text,text)','EXECUTE')
  AND NOT has_function_privilege('vec_contratacion_temporal.antecedente_no_incorporacion_ct124(text,text)','EXECUTE'),
  'tabla y auxiliares de la no incorporación cerrados al ejecutor');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar('preparar_no_incorporacion_v1','vec.contratacion-temporal.preparar-no-incorporacion.v1',:'ni'::jsonb)::text AS prep \gset
SELECT pg_temp.preparar('preparar_no_incorporacion_v1','vec.contratacion-temporal.preparar-no-incorporacion.v1',:'ni_b'::jsonb)->>'resultado' AS prep_b \gset
SELECT pg_temp.preparar('preparar_no_incorporacion_v1','vec.contratacion-temporal.preparar-no-incorporacion.v1',:'ni_misma'::jsonb)->>'error' AS prep_misma \gset
COMMIT;
SELECT pg_temp.exigir((:'prep'::jsonb)->>'resultado'='preparada' AND (:'prep'::jsonb)#>>'{expediente,version}'='7','preparación de la no incorporación');
SELECT pg_temp.exigir(:'prep_b'='incorporacion_existente','con incorporación registrada no hay no incorporación');
SELECT pg_temp.exigir(:'prep_misma'='22023','la segunda persona no puede ser quien registra');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_no_incorporacion_v1',:'ni_futura'::jsonb)->>'resultado'='fecha_no_admitida','notificación futura');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_no_incorporacion_v1',jsonb_set(:'ni'::jsonb,'{contexto,atributos,motivo_clave}','"otro"'))->>'error'='42501','contexto manipulado');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_no_incorporacion_v1',jsonb_set(:'ni'::jsonb,'{expediente_siguiente,fase_actual}','"nombramiento"'))->>'error'='22023','proyección manipulada');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_no_incorporacion_v1',jsonb_set(:'ni'::jsonb,'{_decision,accion}','"contratacion_temporal.ginpix.confirmar"'))->>'error'='42501','acción ajena');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_no_incorporacion_v1',:'ni'::jsonb)::text AS r_ni \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_no_incorporacion_v1',:'ni'::jsonb)::text AS r_ni2 \gset
SELECT pg_temp.confirmar('confirmar_no_incorporacion_v1',jsonb_set(:'ni'::jsonb,'{material,motivo_clave}','"no_aporta_documentacion"'))->>'resultado' AS r_ni3 \gset
SELECT pg_temp.confirmar('confirmar_no_incorporacion_v1',:'ni_otra'::jsonb)->>'resultado' AS r_ni4 \gset
COMMIT;
SELECT pg_temp.exigir((:'r_ni'::jsonb)->>'resultado'='confirmada' AND (:'r_ni'::jsonb)#>>'{recibo,version_resultante}'='8'
  AND (:'r_ni'::jsonb)#>>'{recibo,fase_resultante}'='fiscalizacion' AND (:'r_ni'::jsonb)#>>'{recibo,causa_clave}'='no_presentado',
  'no incorporación: el expediente vuelve a fiscalización en la versión 8 '||:'r_ni');
SELECT pg_temp.exigir((:'r_ni2'::jsonb)->'recibo'=(:'r_ni'::jsonb)->'recibo','repetición con el mismo recibo');
SELECT pg_temp.exigir(:'r_ni3'='idempotencia_reutilizada','clave reutilizada con otro motivo');
SELECT pg_temp.exigir(:'r_ni4'='version_en_conflicto','segunda no incorporación con otra clave');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(:'org',:'exp_a')::text AS consulta_a \gset
COMMIT;
SELECT pg_temp.exigir((:'consulta_a'::jsonb)#>>'{no_incorporacion,motivo_clave}'='no_presentado'
  AND (:'consulta_a'::jsonb)#>>'{no_incorporacion,resuelta_por}'='per_ct124_segunda','consulta del detalle con la no incorporación');
-- Publicación a Bolsa.
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT to_jsonb(x)::text AS pub FROM vec_contratacion_temporal.leer_no_incorporaciones_bolsa_v1(NULL,NULL,10) x \gset
SELECT count(*) AS pub_n FROM vec_contratacion_temporal.leer_no_incorporaciones_bolsa_v1(NULL,NULL,10) \gset
SELECT count(*) AS pub_tras FROM vec_contratacion_temporal.leer_no_incorporaciones_bolsa_v1((:'pub'::jsonb->>'origen_posicion')::bigint,:'pub'::jsonb->>'origen_ref',10) \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir(:'pub_n'='1' AND :'pub_tras'='0' AND (:'pub'::jsonb)#>>'{evento,llamamiento_ref}'=:'llam_a'
  AND (:'pub'::jsonb)#>>'{evento,resuelta_por}'='per_ct124_segunda'
  AND (:'pub'::jsonb)->>'huella_sha256'=encode(sha256(convert_to(((:'pub'::jsonb)->'evento')::text,'UTF8')),'hex'),'publicación a Bolsa con marca de agua');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE tipo_evento='ct.no-incorporacion.v1')=1
  AND (SELECT fase_clave FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version=8)='fiscalizacion',
  'versión, actuación y evento en el outbox');

-- ---------------------------------------------------------------- Bolsa: baja por la bandeja
SELECT (:'pub'::jsonb)->'evento' AS ev, (:'pub'::jsonb)->>'huella_sha256' AS ev_h, (:'pub'::jsonb)->>'origen_creada_en' AS ev_c,
  (:'pub'::jsonb)->>'origen_posicion' AS ev_p \gset
SELECT (:'ev'::jsonb)||jsonb_build_object('motivo_clave','no_aporta_documentacion') AS ev_div \gset
SET SESSION AUTHORIZATION vec_b42_runtime;
SELECT pg_temp.exigir(NOT has_table_privilege('vec_bolsa_llamamientos.no_incorporacion_bolsa','SELECT')
  AND NOT has_function_privilege('vec_bolsa_llamamientos.no_incorporacion_registrada_b42(text)','EXECUTE'),'bandeja cerrada al ejecutor de Bolsa');
SELECT pg_temp.exigir(pg_temp.no_inc_bolsa((:'ev'::jsonb)||'{"resuelta_por":"per_ct124_actor"}',:'ev_h',:'ev_c'::timestamptz,:'ev_p'::bigint,
  pg_temp.consecuencia('excluir'))->>'error'='22023','huella distinta del cuerpo');
SELECT pg_temp.no_inc_bolsa(:'ev'::jsonb,:'ev_h',:'ev_c'::timestamptz,:'ev_p'::bigint,pg_temp.consecuencia('excluir'))::text AS b1 \gset
SELECT pg_temp.no_inc_bolsa(:'ev'::jsonb,:'ev_h',:'ev_c'::timestamptz,:'ev_p'::bigint,pg_temp.consecuencia('ninguna'))::text AS b2 \gset
SELECT pg_temp.no_inc_bolsa(:'ev_div'::jsonb,encode(sha256(convert_to((:'ev_div'::jsonb)::text,'UTF8')),'hex'),:'ev_c'::timestamptz,:'ev_p'::bigint,
  pg_temp.consecuencia('excluir'))::text AS b3 \gset
SELECT to_jsonb(x)::text AS cursor_b FROM vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1() x \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((:'b1'::jsonb)->>'estado'='aplicada' AND (:'b1'::jsonb)->>'reutilizado'='false','baja aplicada '||:'b1');
SELECT pg_temp.exigir((:'b2'::jsonb)->>'reutilizado'='true' AND (:'b2'::jsonb)->>'estado'='aplicada','reentrega idéntica sin efecto nuevo');
SELECT pg_temp.exigir((:'b3'::jsonb)->>'en_cuarentena'='true','entrega divergente en cuarentena');
SELECT pg_temp.exigir((:'cursor_b'::jsonb)->>'origen_ref'=(:'pub'::jsonb)->>'origen_ref','cursor de la bandeja');
SELECT pg_temp.exigir((SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion s
    WHERE s.participacion_ref=(:'b1'::jsonb)->>'participacion_ref' ORDER BY desde DESC LIMIT 1)='excluido'
  AND (SELECT count(*) FROM vec_bolsa_llamamientos.sancion_participacion WHERE clave_idempotencia=(:'ev'::jsonb)->>'evento_ref'
       AND efecto='excluir' AND resuelta_por='per_ct124_segunda' AND actor='per_ct124_actor' AND resolucion_ref='resolucion:rrhh:2026/0142')=1
  AND (SELECT count(*) FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE clave_idempotencia=(:'ev'::jsonb)->>'evento_ref'
       AND operacion='excluir' AND validador='per_ct124_segunda' AND justificante_tipo='resolucion')=1
  AND (SELECT count(*) FROM vec_bolsa_llamamientos.no_incorporacion_bolsa_cuarentena)=1,
  'baja en Bolsa: situación excluida, sanción y operación con la resolución y la segunda persona');

-- ---------------------------------------------------------------- CT: antecedente de la continuación
SELECT intencion_ref AS int_a FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE expediente_ref=:'exp_a' \gset
SELECT pg_temp.solicitud_cont(:'clave_cont',:'org',:'exp_a',:'acept_a',:'int_a') AS sol \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.continuar(format('{"Etapa":"consulta","Solicitud":%s}',:'sol'))::text AS ante \gset
SELECT pg_temp.continuar(format('{"Etapa":"consulta","Solicitud":%s}',replace(:'sol',:'int_a','intencion:ajena')))->>'error' AS ante_ajena \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((:'ante'::jsonb)#>>'{Resolucion,Solicitud,Respuesta}'='aceptacion'
  AND (:'ante'::jsonb)#>>'{ComandoSiguiente,intencion_ref}'=:'int_a'
  AND (:'ante'::jsonb)#>>'{ComandoSiguiente,llamamiento_ref}'=:'llam_a'
  AND (:'ante'::jsonb)#>>'{NoIncorporacion,VersionResultante}'='8'
  AND (:'ante'::jsonb)#>>'{NoIncorporacion,ComandoRef}'=(:'ante'::jsonb)->>'ComandoSiguienteRef','antecedente: aceptación con su no incorporación '||:'ante');
SELECT pg_temp.exigir(:'ante_ajena'='P0602','intención ajena rechazada');

-- ---------------------------------------------------------------- Bolsa: siguiente llamamiento
SET SESSION AUTHORIZATION vec_b42_runtime;
SET TimeZone='UTC'; SET statement_timeout='15s'; SET idle_in_transaction_session_timeout='20s';
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.guardar_bolsa(convert_to(:'sig_bolsa','UTF8'),'bolsa.llamamiento.siguiente.abrir')::text AS g_sig \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.guardar_bolsa(convert_to(:'sig_bolsa','UTF8'),'bolsa.llamamiento.siguiente.abrir')::text AS g_sig2 \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((:'g_sig'::jsonb)->>'recibo_ref' IS NOT NULL AND (:'g_sig2'::jsonb)->>'recibo_ref'=(:'g_sig'::jsonb)->>'recibo_ref',
  'siguiente llamamiento abierto tras la no incorporación (y repetición idéntica) '||:'g_sig');
SELECT pg_temp.exigir((SELECT terminal_anterior_ref FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref='operacion-siguiente-rrhh:prueba:b42')=:'terminal_a'
  AND (SELECT estado FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo WHERE llamamiento_ref='llamamiento:prueba:b42')='abierto'
  AND (SELECT convert_from(registro_canonico,'UTF8')::jsonb#>>'{propuesta,orden_seleccionado}' FROM vec_bolsa_llamamientos.integracion_desarrollo
        WHERE operacion_ref='operacion-siguiente-rrhh:prueba:b42')='3','el siguiente continúa desde la aceptación con la posición 3');

-- ---------------------------------------------------------------- CT: confirmación de la continuación
SELECT format('{"Etapa":"confirmacion","Solicitud":%s,"ReciboBolsa":{"IntencionRef":"%s","TerminalOperacionRef":"%s","OperacionRef":"operacion-siguiente-rrhh:prueba:b42","LlamamientoRef":"llamamiento:prueba:b42","PropuestaRef":"propuesta:prueba:b42","ReciboRef":"%s","AuditoriaRef":"auditoria:prueba:b42","EventoRef":"%s","RegistroSHA256":"%s","ConfirmadaEn":"%s"}}',
  :'sol',:'int_a',:'terminal_a',(:'g_sig'::jsonb)->>'recibo_ref',(:'g_sig'::jsonb)->>'evento_ref',
  (SELECT registro_huella_sha256 FROM vec_bolsa_llamamientos.integracion_desarrollo WHERE operacion_ref='operacion-siguiente-rrhh:prueba:b42'),
  to_char(((:'g_sig'::jsonb)->>'confirmada_en')::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) AS mat_conf \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.continuar(:'mat_conf')::text AS cont \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.continuar(:'mat_conf')::text AS cont2 \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT count(*) AS aviso FROM vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(:'org',:'exp_a','llamamiento:prueba:b42') \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((:'cont'::jsonb)->>'Estado'='confirmado' AND (:'cont'::jsonb)->>'LlamamientoAnteriorRef'=:'llam_a'
  AND (:'cont2'::jsonb)->>'Estado'='replay_confirmado' AND ((:'cont2'::jsonb)-'Estado')=((:'cont'::jsonb)-'Estado'),
  'continuación confirmada y repetición con el mismo recibo '||:'cont');
SELECT pg_temp.exigir(:'aviso'='1','el circuito del sucesor reconoce el nuevo llamamiento');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    WHERE resolucion_ref=:'acept_a' AND continuacion_clave=:'clave_cont'::uuid)=1,'la aceptación queda con su continuación');

CREATE TABLE public.prueba_ct124_ni AS SELECT :'ni'::jsonb AS ni, :'r_ni'::jsonb AS r_ni, :'pub'::jsonb AS pub, :'b1'::jsonb AS b1,
  :'sig_bolsa'::text AS sig_bolsa, :'g_sig'::jsonb AS g_sig, :'mat_conf'::text AS mat_conf, :'cont'::jsonb AS cont;
GRANT SELECT ON public.prueba_ct124_ni TO vec_ct115_runtime, vec_b42_runtime;
SELECT 'cadena no incorporación OK';
