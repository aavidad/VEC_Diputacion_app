\set ON_ERROR_STOP on
-- D6/D8: revisión del documento dentro del circuito y decisión recuperable.
-- 1) consultar_documento_circuito_v1: quien revisa en su etapa lee el
--    documento, sus líneas y justificantes, consumiendo AD3-80.
-- 2) decidir_comision_v2: la asignación Personal se toma de la versión ya
--    enviada, no del cliente, y la repetición con la misma clave devuelve el
--    mismo recibo antes de volver a revalidar Personal; así una respuesta
--    perdida se recupera también tras un reinicio.
-- Solo adición: no altera tablas ni filas. Retira al ejecutor la decisión y
-- la prelectura v1, que la v2 sustituye. La competencia de cada revisor la
-- sigue acreditando la composición; aquí se repiten las comprobaciones de
-- separación y de asignación sobre la versión bloqueada.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000008:revision-circuito:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.cola_circuito_comision') IS NULL
    OR to_regclass('vec_dietas.numero_documento_comision') IS NULL
    OR to_regprocedure('vec_dietas.cotejar_efecto_circuito_v1(text,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_circuito_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)','EXECUTE')
 THEN RAISE EXCEPTION 'Dietas 000008: faltan 000007 o AD3-80, o ya instalada' USING ERRCODE='55000'; END IF;
END $pre$;

-- Reconstruye el recurso autorizado por Go desde los bytes exactos del
-- material v2. Ni la decisión ni la lectura llevan asignación del cliente.
CREATE FUNCTION vec_dietas.cotejar_efecto_circuito_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_contexto bytea
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; c jsonb; d jsonb; x jsonb;
 etapa text; unidad text; accion text; finalidad text; audiencia text;
 campos jsonb; huella text; amb text; atr text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL
 THEN RETURN false; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RETURN false; END;
 i:=m->'identidad'; unidad:=m->>'unidad_ref';
 IF jsonb_typeof(m)<>'object' OR m->>'esquema'<>'vec.dietas.circuito-operacion.v2'
    OR m ? 'asignacion' OR m ? 'consulta'
    OR unidad !~ '^[A-Za-z0-9:_-]{3,128}$'
    OR m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR i->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR i->>'actor_ref' IS NULL OR i->>'perfil_ref' IS NULL
    OR i->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR i->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR i->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR i->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR i->>'persona_ref' IS DISTINCT FROM x->>'persona_ref'
    OR i->>'persona_ref' IS DISTINCT FROM x->>'principal_ref'
    OR i->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR i->>'persona_version' IS DISTINCT FROM x->>'persona_version'
    OR i->>'perfil_version' IS DISTINCT FROM x->>'perfil_version'
    OR x->>'estado'<>'activo'
    OR c->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex')
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->>'concedida'<>'true' OR d->>'modulo_id'<>'dietas'
    OR d->>'tipo_recurso'<>'documento_dietas'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR c->>'efecto_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
 THEN RETURN false; END IF;
 IF m->>'operacion'='decidir' THEN
  etapa:=m->'comando'->>'etapa';
  accion:=CASE etapa WHEN 'revision' THEN 'dietas.documento.revisar'
   WHEN 'autorizacion' THEN 'dietas.documento.autorizar'
   WHEN 'liquidacion' THEN 'dietas.documento.liquidar'
   WHEN 'fiscalizacion' THEN 'dietas.documento.fiscalizar' END;
  finalidad:=CASE etapa WHEN 'revision' THEN 'revisar_documento_dietas'
   WHEN 'autorizacion' THEN 'autorizar_documento_dietas'
   WHEN 'liquidacion' THEN 'liquidar_documento_dietas'
   WHEN 'fiscalizacion' THEN 'fiscalizar_documento_dietas' END;
  audiencia:='vec_dietas.documento.'||CASE etapa WHEN 'revision' THEN 'revisar'
   WHEN 'autorizacion' THEN 'autorizar' WHEN 'liquidacion' THEN 'liquidar'
   WHEN 'fiscalizacion' THEN 'fiscalizar' END||'.v1';
  campos:='["comision.estado","comision.referencia","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb;
  IF jsonb_typeof(m->'comando')<>'object' OR NOT m ? 'huella_semantica' OR m ? 'etapa'
     OR m->'comando'->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
     OR m->'comando'->>'unidad_ref' IS DISTINCT FROM unidad THEN RETURN false; END IF;
 ELSIF m->>'operacion'='consultar_documento' THEN
  etapa:=m->>'etapa';
  accion:='dietas.circuito.documento.consultar';
  finalidad:='revisar_documento_circuito_dietas';
  audiencia:='vec_dietas.circuito.documento.consultar.v1';
  campos:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'::jsonb;
  IF m ? 'comando' OR m ? 'huella_semantica' THEN RETURN false; END IF;
 ELSE RETURN false; END IF;
 IF etapa IS NULL OR etapa NOT IN ('revision','autorizacion','liquidacion','fiscalizacion')
    OR d->>'accion' IS DISTINCT FROM accion
    OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR c->>'operacion' IS DISTINCT FROM accion
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
 THEN RETURN false; END IF;
 -- RecursoAutorizable.HuellaContextoAutorizacionSHA256: claves ordenadas y
 -- cadenas con el escape de encoding/json usado en Go.
 amb:='{"persona_ref":'||vec_dietas.cadena_json_go_v1(i->>'persona_ref')
   ||',"unidad_ref":'||vec_dietas.cadena_json_go_v1(unidad)||'}';
 atr:='"contexto_actor_ref":'||vec_dietas.cadena_json_go_v1(i->>'contexto_actor_ref')
  ||',"contexto_version":'||vec_dietas.cadena_json_go_v1(i->>'contexto_version')
  ||',"cuenta_ref":'||vec_dietas.cadena_json_go_v1(i->>'cuenta_ref')
  ||',"cuenta_version":'||vec_dietas.cadena_json_go_v1(i->>'cuenta_version')
  ||CASE WHEN m->>'operacion'='consultar_documento' THEN ',"etapa":'||vec_dietas.cadena_json_go_v1(etapa) ELSE '' END
  ||',"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"'
  ||',"operacion":'||vec_dietas.cadena_json_go_v1(m->>'operacion')
  ||',"perfil_version":'||vec_dietas.cadena_json_go_v1(i->>'perfil_version')
  ||',"persona_version":'||vec_dietas.cadena_json_go_v1(i->>'persona_version')
  ||',"recurso_ref":'||vec_dietas.cadena_json_go_v1(m->>'recurso_ref');
 huella:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":{'||atr||'}}','UTF8')),'hex');
 RETURN d->>'contexto_recurso_huella_sha256'=huella
    AND c->>'huella_efecto_sha256'=huella;
EXCEPTION WHEN others THEN RETURN false;
END $funcion$;
ALTER FUNCTION vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea) FROM PUBLIC,vec_dietas_ejecutor;

-- Lectura del documento por quien lo tiene pendiente en su etapa. Aplica las
-- mismas reglas que la bandeja: la persona titular nunca revisa lo suyo, quien
-- ya actuó en otra etapa no actúa en esta y Personal revalida la asignación.
CREATE FUNCTION vec_dietas.consultar_documento_circuito_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; cap jsonb; d jsonb; ctx jsonb; v record;
 q vec_dietas.cola_circuito_comision%ROWTYPE;
 b vec_dietas.borrador_comision%ROWTYPE; r vec_dietas.comision_revision%ROWTYPE;
 numero vec_dietas.numero_documento_comision%ROWTYPE;
 etapa text; estado_esperado text; comision jsonb;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
 THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN
  m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 i:=m->'identidad'; etapa:=m->>'etapa';
 estado_esperado:=CASE etapa WHEN 'revision' THEN 'enviado_pendiente_revision'
  WHEN 'autorizacion' THEN 'pendiente_autorizacion'
  WHEN 'liquidacion' THEN 'pendiente_liquidacion'
  WHEN 'fiscalizacion' THEN 'pendiente_fiscalizacion' END;
 IF m->>'operacion' IS DISTINCT FROM 'consultar_documento' OR estado_esperado IS NULL
    OR vec_dietas.cotejar_efecto_circuito_v2(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric
 THEN RAISE EXCEPTION 'lectura de revisión Dietas incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM cap->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'consumo AD3 de revisión incompatible' USING ERRCODE='PD003'; END IF;
 PERFORM set_config('vec.dietas.circuito_etapa',etapa,true);
 PERFORM set_config('vec.dietas.circuito_unidad_ref',m->>'unidad_ref',true);
 SELECT * INTO q FROM vec_dietas.cola_circuito_comision cola
  WHERE cola.comision_ref=m->>'recurso_ref' AND cola.etapa=etapa
    AND cola.unidad_ref=m->>'unidad_ref'
  ORDER BY cola.version DESC LIMIT 1;
 IF NOT FOUND THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 PERFORM set_config('vec.dietas.persona_ref',q.persona_ref,true);
 SELECT * INTO b FROM vec_dietas.borrador_comision WHERE referencia=q.comision_ref;
 SELECT * INTO r FROM vec_dietas.comision_revision
  WHERE comision_ref=q.comision_ref ORDER BY version DESC LIMIT 1;
 IF NOT FOUND OR b.referencia IS NULL OR b.unidad_ref IS DISTINCT FROM q.unidad_ref
    OR r.version IS DISTINCT FROM q.version OR r.estado IS DISTINCT FROM estado_esperado
    OR r.asignacion_ref IS DISTINCT FROM q.asignacion_ref
    OR r.asignacion_version IS DISTINCT FROM q.asignacion_version
    OR i->>'persona_ref'=q.persona_ref
    OR (q.destinatario_persona_ref IS NOT NULL
      AND q.destinatario_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR (etapa='revision' AND r.administrativo_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR (etapa='autorizacion' AND r.responsable_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision h
      WHERE h.comision_ref=q.comision_ref AND h.actor_ref=i->>'actor_ref'
        AND h.tipo IS DISTINCT FROM ('circuito_'||etapa))
 THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 BEGIN
  IF vec_personal.revalidar_asignacion_dietas_v1(b.relacion_ref,q.persona_ref,
    b.unidad_ref,q.asignacion_ref,q.asignacion_version,r.grupo_dieta,r.centro_ref,
    r.administrativo_persona_ref,r.responsable_persona_ref,current_date) IS NOT TRUE
  THEN RETURN jsonb_build_object('resultado','no_encontrado'); END IF;
 EXCEPTION WHEN SQLSTATE 'P7201' THEN
  RETURN jsonb_build_object('resultado','no_encontrado');
 END;
 SELECT * INTO STRICT numero FROM vec_dietas.numero_documento_comision WHERE comision_ref=q.comision_ref;
 comision:=jsonb_build_object('referencia',q.comision_ref,'numero_documento',numero.numero_documento,
  'fecha_apertura',to_char(numero.fecha_apertura AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'estado',r.estado,'version',r.version,
  'fecha_inicio',r.fecha_inicio::text,'fecha_fin',r.fecha_fin::text,
  'hora_inicio',r.hora_inicio,'hora_fin',r.hora_fin,'motivo',r.motivo,
  'codigos_ruta',r.codigos_ruta,'calculo',r.calculo,'documento',r.documento);
 IF r.vehiculo_propio IS NOT NULL THEN
  comision:=comision||jsonb_build_object('vehiculo_propio',r.vehiculo_propio,'rutas',r.rutas);
 END IF;
 RETURN jsonb_build_object('resultado','concedido','comision',comision);
END $funcion$;
ALTER FUNCTION vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_dietas.decidir_comision_v2(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; c jsonb; cap jsonb; d jsonb; ctx jsonb;
 v record; q vec_dietas.cola_circuito_comision%ROWTYPE;
 b vec_dietas.borrador_comision%ROWTYPE; anterior vec_dietas.comision_revision%ROWTYPE;
 recibo vec_dietas.recibo_operacion_comision%ROWTYPE;
 regla_catalogo jsonb; regla_fin jsonb; regla_huella text;
 etapa text; estado_previo text; estado_nuevo text; etapa_nueva text; destino text;
 huella text; operacion text; ahora timestamptz(6); version_nueva bigint;
 recibo_ref text; evento_ref text; outbox_ref text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
 THEN RAISE EXCEPTION 'ejecutor Dietas inválido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'transacción Dietas incompatible' USING ERRCODE='25000'; END IF;
 BEGIN
  m:=p_material::jsonb; cap:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material Dietas inválido' USING ERRCODE='22023'; END;
 i:=m->'identidad'; c:=m->'comando'; etapa:=c->>'etapa';
 estado_previo:=CASE etapa WHEN 'revision' THEN 'enviado_pendiente_revision'
  WHEN 'autorizacion' THEN 'pendiente_autorizacion'
  WHEN 'liquidacion' THEN 'pendiente_liquidacion'
  WHEN 'fiscalizacion' THEN 'pendiente_fiscalizacion' END;
 etapa_nueva:=CASE etapa WHEN 'revision' THEN 'autorizacion'
  WHEN 'autorizacion' THEN 'liquidacion'
  WHEN 'liquidacion' THEN 'fiscalizacion' END;
 estado_nuevo:=CASE WHEN c->>'decision'='devolver' THEN 'devuelta'
  ELSE CASE etapa WHEN 'revision' THEN 'pendiente_autorizacion'
   WHEN 'autorizacion' THEN 'pendiente_liquidacion'
   WHEN 'liquidacion' THEN 'pendiente_fiscalizacion'
   WHEN 'fiscalizacion' THEN 'fiscalizada' END END;
 IF estado_previo IS NULL OR estado_nuevo IS NULL OR m->>'operacion' IS DISTINCT FROM 'decidir'
    OR c->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'unidad_ref' IS DISTINCT FROM m->>'unidad_ref'
    OR c->>'decision' NOT IN ('aprobar','devolver')
    OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR c->>'version_esperada' !~ '^[1-9][0-9]{0,17}$'
    OR c->>'motivo' IS NULL OR length(c->>'motivo')>600
    OR c->>'motivo' IS DISTINCT FROM btrim(c->>'motivo')
    OR c->>'motivo' ~ '[[:cntrl:]]'
    OR (c->>'decision'='devolver' AND length(c->>'motivo')<3)
 THEN RAISE EXCEPTION 'decisión Dietas inválida' USING ERRCODE='22023'; END IF;
 huella:=encode(sha256(convert_to(concat_ws(chr(31),i->>'persona_ref',m->>'recurso_ref',
  c->>'unidad_ref',etapa,c->>'decision',c->>'motivo',c->>'clave_idempotencia',
  c->>'version_esperada'),'UTF8')),'hex');
 IF m->>'huella_semantica' IS DISTINCT FROM huella
    OR vec_dietas.cotejar_efecto_circuito_v2(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric
 THEN RAISE EXCEPTION 'sello Dietas incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_circuito_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM cap->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'consumo AD3 Dietas incompatible' USING ERRCODE='PD003'; END IF;
 -- Solo tras el consumo nominal se abre la cola de la etapa y unidad exactas.
 PERFORM set_config('vec.dietas.circuito_etapa',etapa,true);
 PERFORM set_config('vec.dietas.circuito_unidad_ref',m->>'unidad_ref',true);
 SELECT * INTO q FROM vec_dietas.cola_circuito_comision cola
 WHERE cola.comision_ref=m->>'recurso_ref' AND cola.etapa=etapa
    AND cola.unidad_ref=m->>'unidad_ref'
    AND cola.version=(c->>'version_esperada')::bigint;
 IF NOT FOUND THEN RAISE EXCEPTION 'documento Dietas no disponible' USING ERRCODE='PD004'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',q.persona_ref,true);
 SELECT * INTO STRICT b FROM vec_dietas.borrador_comision WHERE referencia=q.comision_ref;
 SELECT * INTO STRICT anterior FROM vec_dietas.comision_revision
  WHERE comision_ref=q.comision_ref AND version=q.version FOR UPDATE;
 IF b.unidad_ref IS DISTINCT FROM q.unidad_ref
    OR anterior.estado IS DISTINCT FROM estado_previo
    OR anterior.asignacion_ref IS DISTINCT FROM q.asignacion_ref
    OR anterior.asignacion_version IS DISTINCT FROM q.asignacion_version
    OR anterior.administrativo_persona_ref IS NULL OR anterior.responsable_persona_ref IS NULL
 THEN RAISE EXCEPTION 'asignación Dietas incompatible' USING ERRCODE='PD003'; END IF;
 operacion:='circuito_'||etapa;
 -- La misma clave del mismo actor devuelve el recibo original aunque la
 -- asignación haya cambiado después: el efecto ya se registró y no se repite.
 SELECT * INTO recibo FROM vec_dietas.recibo_operacion_comision r
  WHERE r.persona_ref=q.persona_ref AND r.operacion=operacion
    AND r.clave_idempotencia=c->>'clave_idempotencia';
 IF FOUND THEN
  IF recibo.comision_ref IS DISTINCT FROM q.comision_ref
     OR recibo.huella_semantica_sha256 IS DISTINCT FROM huella
     OR recibo.actor_ref IS DISTINCT FROM i->>'actor_ref'
  THEN RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
  SELECT estado INTO STRICT estado_nuevo FROM vec_dietas.comision_revision
   WHERE comision_ref=q.comision_ref AND version=recibo.version;
  RETURN jsonb_build_object('comision',jsonb_build_object('referencia',q.comision_ref,
   'estado',estado_nuevo,'version',recibo.version),'recibo',jsonb_build_object(
   'referencia',recibo.referencia,'version',recibo.version,
   'registrado_en',to_char(recibo.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'repeticion',true));
 END IF;
 IF i->>'persona_ref'=q.persona_ref
    OR (etapa='revision' AND i->>'persona_ref' IS DISTINCT FROM anterior.administrativo_persona_ref)
    OR (etapa='autorizacion' AND i->>'persona_ref' IS DISTINCT FROM anterior.responsable_persona_ref)
    OR (q.destinatario_persona_ref IS NOT NULL
      AND q.destinatario_persona_ref IS DISTINCT FROM i->>'persona_ref')
    OR EXISTS (SELECT 1 FROM vec_dietas.historia_operacion_comision h
      WHERE h.comision_ref=q.comision_ref AND h.actor_ref=i->>'actor_ref'
        AND h.tipo<>operacion)
 THEN RAISE EXCEPTION 'separación de funciones Dietas' USING ERRCODE='42501'; END IF;
 SELECT max(version) INTO version_nueva FROM vec_dietas.comision_revision
  WHERE comision_ref=q.comision_ref;
 IF version_nueva IS DISTINCT FROM q.version THEN
  RAISE EXCEPTION 'versión Dietas desactualizada' USING ERRCODE='PD005';
 END IF;
 regla_catalogo:=vec_dietas.consultar_regla_devengo_dietas_v1(
  anterior.calculo->>'version_tarifa',anterior.fecha_inicio,'ES','nacional_ordinaria');
 regla_fin:=vec_dietas.consultar_regla_devengo_dietas_v1(
  anterior.calculo->>'version_tarifa',anterior.fecha_fin,'ES','nacional_ordinaria');
 regla_huella:=regla_catalogo->>'huella_sha256';
 IF anterior.regla_ref IS NULL
    OR anterior.regla_ref IS DISTINCT FROM anterior.calculo->>'regla_ref'
    OR anterior.regla_ref IS DISTINCT FROM regla_catalogo->>'regla_ref'
    OR anterior.regla_ref IS DISTINCT FROM regla_fin->>'regla_ref'
    OR regla_huella !~ '^[0-9a-f]{64}$'
    OR regla_huella IS DISTINCT FROM anterior.calculo->>'regla_huella_sha256'
    OR regla_huella IS DISTINCT FROM regla_fin->>'huella_sha256'
    OR regla_catalogo->>'version_tarifa_ref' IS DISTINCT FROM anterior.calculo->>'version_tarifa'
    OR regla_fin->>'version_tarifa_ref' IS DISTINCT FROM anterior.calculo->>'version_tarifa'
 THEN RAISE EXCEPTION 'regla Dietas incompatible' USING ERRCODE='PD003'; END IF;
 IF vec_personal.revalidar_asignacion_dietas_v1(b.relacion_ref,q.persona_ref,
   b.unidad_ref,q.asignacion_ref,q.asignacion_version,anterior.grupo_dieta,
   anterior.centro_ref,anterior.administrativo_persona_ref,
   anterior.responsable_persona_ref,current_date) IS NOT TRUE
 THEN RAISE EXCEPTION 'asignación Personal no vigente' USING ERRCODE='P7201'; END IF;
 version_nueva:=q.version+1;
 ahora:=date_trunc('microseconds',clock_timestamp());
 recibo_ref:='rcd_'||gen_random_uuid()::text;
 evento_ref:='hdi_'||md5(q.comision_ref||version_nueva::text||recibo_ref);
 outbox_ref:='odi_'||md5(q.comision_ref||version_nueva::text||recibo_ref);
 INSERT INTO vec_dietas.comision_revision
 (comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,
  codigos_ruta,vehiculo_propio,rutas,calculo,documento,regla_ref,asignacion_ref,asignacion_version,
  grupo_dieta,centro_ref,administrativo_persona_ref,responsable_persona_ref,registrada_en)
 VALUES (q.comision_ref,version_nueva,estado_nuevo,anterior.fecha_inicio,anterior.fecha_fin,
  anterior.hora_inicio,anterior.hora_fin,anterior.motivo,anterior.codigos_ruta,
  anterior.vehiculo_propio,anterior.rutas,anterior.calculo,anterior.documento,
  anterior.regla_ref,anterior.asignacion_ref,
  anterior.asignacion_version,anterior.grupo_dieta,anterior.centro_ref,
  anterior.administrativo_persona_ref,anterior.responsable_persona_ref,ahora);
 INSERT INTO vec_dietas.recibo_operacion_comision
 (referencia,comision_ref,version,operacion,clave_idempotencia,comando,
  huella_semantica_sha256,decision_ref,consumo_huella_sha256,auditoria_ad3_ref,
  actor_ref,persona_ref,regla_ref,regla_huella_sha256,registrada_en)
 VALUES (recibo_ref,q.comision_ref,version_nueva,operacion,c->>'clave_idempotencia',c,
  huella,v.decision_ref,v.consumo_huella_sha256,v.auditoria_ref,
  i->>'actor_ref',q.persona_ref,anterior.regla_ref,regla_huella,ahora);
 INSERT INTO vec_dietas.historia_operacion_comision
 (evento_ref,comision_ref,version,recibo_ref,estado_anterior,estado_nuevo,tipo,motivo,actor_ref,registrada_en)
 VALUES(evento_ref,q.comision_ref,version_nueva,recibo_ref,estado_previo,estado_nuevo,
  operacion,NULLIF(c->>'motivo',''),i->>'actor_ref',ahora);
 IF estado_nuevo IN ('devuelta','fiscalizada') THEN
  destino:=q.persona_ref;
 ELSIF estado_nuevo='pendiente_autorizacion' THEN
  destino:=anterior.responsable_persona_ref;
 ELSE destino:=NULL; END IF;
 INSERT INTO vec_dietas.outbox_comision
 (evento_ref,comision_ref,version,tipo,persona_ref,destinatario_persona_ref,
  destinatario_unidad_ref,destinatario_etapa,estado,registrada_en)
 VALUES(outbox_ref,q.comision_ref,version_nueva,'circuito_'||estado_nuevo,q.persona_ref,
  destino,CASE WHEN destino IS NULL THEN q.unidad_ref ELSE NULL END,
  CASE WHEN destino IS NULL THEN etapa_nueva ELSE NULL END,'pendiente',ahora);
 IF etapa_nueva IS NOT NULL AND estado_nuevo<>'devuelta' THEN
  INSERT INTO vec_dietas.cola_circuito_comision
  (comision_ref,version,etapa,persona_ref,unidad_ref,asignacion_ref,
   asignacion_version,destinatario_persona_ref,fecha_inicio,fecha_fin,registrada_en)
  VALUES(q.comision_ref,version_nueva,etapa_nueva,q.persona_ref,q.unidad_ref,
   q.asignacion_ref,q.asignacion_version,destino,q.fecha_inicio,q.fecha_fin,ahora);
 END IF;
 RETURN jsonb_build_object('comision',jsonb_build_object('referencia',q.comision_ref,
  'estado',estado_nuevo,'version',version_nueva),'recibo',jsonb_build_object(
  'referencia',recibo_ref,'version',version_nueva,
  'registrado_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'repeticion',false));
END $funcion$;
ALTER FUNCTION vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

GRANT EXECUTE ON FUNCTION
 vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_dietas_ejecutor;
-- La v1 exigía un sello Personal enviado por el cliente y no recuperaba el
-- recibo después de decidir; deja de ser invocable por el ejecutor.
REVOKE EXECUTE ON FUNCTION
 vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_dietas_ejecutor;
COMMIT;
