\set ON_ERROR_STOP on
-- D6/D8: circuito interno de la comisión. La cola acredita trabajo pendiente;
-- no acredita notificación, firma, fiscalización legal ni pago.
BEGIN;
-- El inventario debe hacerlo un DBA antes de asumir el propietario: 000006
-- fuerza RLS contextual y el propietario sin persona_ref no ve las filas.
RESET ROLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_dietas:migracion:000007:circuito:v1',0));
DO $preimagen_dba$
BEGIN
 IF current_user IS DISTINCT FROM session_user
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_dietas.comision_revision') IS NULL
 THEN RAISE EXCEPTION 'Dietas 000007: inventario requiere DBA y 000006' USING ERRCODE='55000'; END IF;
END $preimagen_dba$;
-- Bloquea envíos concurrentes hasta el COMMIT y evita instalar una cola vacía
-- sobre una revisión enviada mientras se comprueba la preimagen.
LOCK TABLE vec_dietas.comision_revision IN SHARE ROW EXCLUSIVE MODE;
DO $historia_dba$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_dietas.comision_revision
            WHERE estado NOT IN ('borrador','eliminado'))
 THEN RAISE EXCEPTION 'Dietas 000007: hay circuito previo sin cola' USING ERRCODE='55000'; END IF;
END $historia_dba$;
SET LOCAL ROLE vec_dietas_propietario;
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario'
    OR to_regclass('vec_dietas.comision_revision') IS NULL
    OR to_regclass('vec_dietas.recibo_operacion_comision') IS NULL
    OR to_regclass('vec_dietas.historia_operacion_comision') IS NULL
    OR to_regclass('vec_dietas.outbox_comision') IS NULL
    OR to_regclass('vec_dietas.cola_circuito_comision') IS NOT NULL
    OR to_regprocedure('vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text)') IS NULL
    OR to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_circuito_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_bandeja_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_prelectura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_dietas_propietario','vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_dietas.consultar_regla_devengo_dietas_v1(text,date,text,text)','EXECUTE')
    OR (SELECT count(*) FROM pg_attribute a
         WHERE a.attrelid='vec_dietas.recibo_operacion_comision'::regclass
           AND a.attname IN ('regla_ref','regla_huella_sha256')
           AND a.attnotnull AND NOT a.attisdropped AND a.atttypid='text'::regtype)<>2
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_circuito_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_bandeja_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_dietas_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_prelectura_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'Dietas 000007: faltan 000006, Personal-11 o AD3-52, o ya hay circuito sin cola' USING ERRCODE='55000'; END IF;
END $pre$;

-- Un registro de cola por estado de trabajo. Las versiones anteriores quedan
-- conservadas, y la consulta toma sólo la versión más reciente del documento.
CREATE TABLE vec_dietas.cola_circuito_comision (
 comision_ref text NOT NULL,
 version bigint NOT NULL,
 etapa text NOT NULL CHECK(etapa IN ('revision','autorizacion','liquidacion','fiscalizacion')),
 persona_ref text NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 unidad_ref text NOT NULL CHECK(length(unidad_ref) BETWEEN 1 AND 256),
 asignacion_ref text NOT NULL CHECK(asignacion_ref~'^ads_[A-Za-z0-9_-]{22,128}$'),
 asignacion_version bigint NOT NULL CHECK(asignacion_version>0),
 destinatario_persona_ref text CHECK(destinatario_persona_ref IS NULL OR destinatario_persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 fecha_inicio date NOT NULL,
 fecha_fin date NOT NULL CHECK(fecha_fin>=fecha_inicio),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(comision_ref,version),
 FOREIGN KEY(comision_ref,version) REFERENCES vec_dietas.comision_revision(comision_ref,version),
 CHECK((etapa IN ('revision','autorizacion') AND destinatario_persona_ref IS NOT NULL)
    OR (etapa IN ('liquidacion','fiscalizacion') AND destinatario_persona_ref IS NULL))
);
CREATE INDEX cola_circuito_etapa_unidad_fecha_idx
 ON vec_dietas.cola_circuito_comision(etapa,unidad_ref,fecha_inicio,comision_ref);
ALTER TABLE vec_dietas.cola_circuito_comision ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.cola_circuito_comision FORCE ROW LEVEL SECURITY;
CREATE POLICY insertar_cola_titular ON vec_dietas.cola_circuito_comision
 FOR INSERT TO vec_dietas_propietario
 WITH CHECK (persona_ref=current_setting('vec.dietas.persona_ref',true));
CREATE POLICY leer_cola_acotada ON vec_dietas.cola_circuito_comision
 FOR SELECT TO vec_dietas_propietario
 USING (etapa=current_setting('vec.dietas.circuito_etapa',true)
   AND unidad_ref=current_setting('vec.dietas.circuito_unidad_ref',true));
CREATE TRIGGER cola_circuito_inmutable BEFORE UPDATE OR DELETE ON vec_dietas.cola_circuito_comision
 FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
CREATE TRIGGER cola_circuito_no_truncar BEFORE TRUNCATE ON vec_dietas.cola_circuito_comision
 FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_borrador_v1();
REVOKE ALL ON TABLE vec_dietas.cola_circuito_comision FROM PUBLIC,vec_dietas_ejecutor;

-- 000006 publica la versión enviada. El trigger abre la cola en la misma
-- transacción; no crea un segundo outbox ni pretende que el aviso se entregó.
CREATE FUNCTION vec_dietas.abrir_cola_revision_comision_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $funcion$
#variable_conflict use_variable
DECLARE titular text; unidad text;
BEGIN
 IF current_user<>'vec_dietas_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_migrador','MEMBER')
    OR NEW.estado<>'enviado_pendiente_revision'
    OR NEW.asignacion_ref IS NULL OR NEW.asignacion_version IS NULL
    OR NEW.administrativo_persona_ref IS NULL THEN
   RAISE EXCEPTION 'apertura de cola Dietas inválida' USING ERRCODE='42501';
 END IF;
 SELECT b.persona_ref,b.unidad_ref INTO STRICT titular,unidad
 FROM vec_dietas.borrador_comision b WHERE b.referencia=NEW.comision_ref;
 IF current_setting('vec.dietas.persona_ref',true) IS DISTINCT FROM titular THEN
   RAISE EXCEPTION 'titular de cola Dietas incompatible' USING ERRCODE='42501';
 END IF;
 INSERT INTO vec_dietas.cola_circuito_comision
 (comision_ref,version,etapa,persona_ref,unidad_ref,asignacion_ref,asignacion_version,
  destinatario_persona_ref,fecha_inicio,fecha_fin,registrada_en)
 VALUES (NEW.comision_ref,NEW.version,'revision',titular,unidad,
  NEW.asignacion_ref,NEW.asignacion_version,NEW.administrativo_persona_ref,
  NEW.fecha_inicio,NEW.fecha_fin,NEW.registrada_en);
 RETURN NEW;
END $funcion$;
ALTER FUNCTION vec_dietas.abrir_cola_revision_comision_v1() OWNER TO vec_dietas_propietario;
CREATE TRIGGER abrir_cola_revision AFTER INSERT ON vec_dietas.comision_revision
 FOR EACH ROW WHEN (NEW.estado='enviado_pendiente_revision')
 EXECUTE FUNCTION vec_dietas.abrir_cola_revision_comision_v1();
REVOKE ALL ON FUNCTION vec_dietas.abrir_cola_revision_comision_v1() FROM PUBLIC,vec_dietas_ejecutor;

-- Reconstruye el recurso autorizado por Go a partir de los bytes exactos del
-- material. Una capacidad para otra unidad, etapa o comando no se reutiliza.
CREATE FUNCTION vec_dietas.cotejar_efecto_circuito_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_contexto bytea
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; c jsonb; d jsonb; x jsonb; a jsonb; v jsonb;
 etapa text; unidad text; accion text; finalidad text; audiencia text; recurso_tipo text;
 campos jsonb; huella text; amb text; atr text; j text;
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
 i:=m->'identidad'; a:=m->'asignacion'; unidad:=m->>'unidad_ref';
 IF m->>'esquema' NOT IN ('vec.dietas.circuito-operacion.v1','vec.dietas.circuito-prelectura.v1')
    OR unidad !~ '^[A-Za-z0-9:_-]{3,128}$'
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
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR c->>'efecto_ref' IS DISTINCT FROM m->>'recurso_ref'
    OR d->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
 THEN RETURN false; END IF;
 IF m->>'operacion'='preleer' AND m->>'esquema'='vec.dietas.circuito-prelectura.v1' THEN
  etapa:=m->>'etapa';
  accion:='dietas.circuito.preleer';
  finalidad:='preleer_competencia_circuito_dietas';
  audiencia:='vec_dietas.circuito.preleer.v1';
  recurso_tipo:='documento_dietas';
  campos:='["comision.administrativo_persona_ref","comision.asignacion_ref","comision.asignacion_version","comision.centro_ref","comision.estado","comision.grupo_dieta","comision.referencia","comision.relacion_ref","comision.responsable_persona_ref","comision.unidad_ref","comision.version","resultado"]'::jsonb;
  IF m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
     OR a IS NOT NULL OR m ? 'comando' OR m ? 'consulta' OR m ? 'huella_semantica'
  THEN RETURN false; END IF;
 ELSIF m->>'operacion'='decidir' AND m->>'esquema'='vec.dietas.circuito-operacion.v1' THEN
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
  recurso_tipo:='documento_dietas';
  campos:='["comision.estado","comision.referencia","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb;
  IF m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
     OR m->'comando'->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
     OR m->'comando'->>'unidad_ref' IS DISTINCT FROM unidad
     OR a IS NULL OR a->>'unidad_ref' IS DISTINCT FROM unidad
     OR a->>'asignacion_ref' !~ '^ads_[A-Za-z0-9_-]{22,128}$'
     OR coalesce((a->>'version')::bigint,0)<=0 THEN RETURN false; END IF;
 ELSIF m->>'operacion'='listar_bandeja' AND m->>'esquema'='vec.dietas.circuito-operacion.v1' THEN
  etapa:=m->'consulta'->>'etapa';
  accion:='dietas.bandeja.'||etapa||'.consultar';
  finalidad:='consultar_bandeja_'||etapa||'_dietas';
  audiencia:='vec_dietas.bandeja.'||etapa||'.consultar.v1';
  recurso_tipo:='bandeja_dietas';
  campos:='["items.estado","items.fecha_fin","items.fecha_inicio","items.referencia","items.version","siguiente_cursor"]'::jsonb;
  IF m->>'recurso_ref' IS DISTINCT FROM 'dietas:bandeja:'||etapa
     OR m->'consulta'->>'unidad_ref' IS DISTINCT FROM unidad
     OR a IS NOT NULL OR m ? 'comando' OR m ? 'huella_semantica' THEN RETURN false; END IF;
 ELSE RETURN false; END IF;
 IF etapa NOT IN ('revision','autorizacion','liquidacion','fiscalizacion')
    OR d->>'accion' IS DISTINCT FROM accion
    OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'tipo_recurso' IS DISTINCT FROM recurso_tipo
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR c->>'operacion' IS DISTINCT FROM accion
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
 THEN RETURN false; END IF;
 -- RecursoAutorizable.HuellaContextoAutorizacionSHA256: campos y claves
 -- ordenados, cadenas con el escape de encoding/json usado en Go.
 j:='"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"';
 amb:='{"persona_ref":'||vec_dietas.cadena_json_go_v1(i->>'persona_ref')
   ||',"unidad_ref":'||vec_dietas.cadena_json_go_v1(unidad)||'}';
 atr:='';
 IF a IS NOT NULL THEN
  atr:='"asignacion_ref":'||vec_dietas.cadena_json_go_v1(a->>'asignacion_ref')
   ||',"asignacion_version":'||vec_dietas.cadena_json_go_v1(a->>'version')||',';
 END IF;
 atr:=atr||'"contexto_actor_ref":'||vec_dietas.cadena_json_go_v1(i->>'contexto_actor_ref')
  ||',"contexto_version":'||vec_dietas.cadena_json_go_v1(i->>'contexto_version')
  ||',"cuenta_ref":'||vec_dietas.cadena_json_go_v1(i->>'cuenta_ref')
  ||',"cuenta_version":'||vec_dietas.cadena_json_go_v1(i->>'cuenta_version')
  ||CASE WHEN m->>'operacion'='preleer' THEN ',"etapa":'||vec_dietas.cadena_json_go_v1(etapa) ELSE '' END
  ||',"material_sha256":'||j
  ||',"operacion":'||vec_dietas.cadena_json_go_v1(m->>'operacion')
  ||',"perfil_version":'||vec_dietas.cadena_json_go_v1(i->>'perfil_version')
  ||',"persona_version":'||vec_dietas.cadena_json_go_v1(i->>'persona_version')
  ||',"recurso_ref":'||vec_dietas.cadena_json_go_v1(m->>'recurso_ref');
 huella:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":{'||atr||'}}','UTF8')),'hex');
 RETURN d->>'contexto_recurso_huella_sha256'=huella
    AND c->>'huella_efecto_sha256'=huella
    AND c->>'huella_contexto_sha256'=encode(sha256(p_contexto),'hex');
EXCEPTION WHEN others THEN RETURN false;
END $funcion$;
ALTER FUNCTION vec_dietas.cotejar_efecto_circuito_v1(text,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.cotejar_efecto_circuito_v1(text,bytea,bytea,bytea) FROM PUBLIC,vec_dietas_ejecutor;

-- Lectura mínima para resolver dco → relación/unidad/sello D7. La competencia
-- por unidad ya está acreditada en el recurso AD3; Personal se revalida luego
-- para la decisión y aquí para no revelar una asignación ya inválida.
CREATE FUNCTION vec_dietas.preleer_circuito_comision_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; cap jsonb; d jsonb; ctx jsonb; v record;
 q vec_dietas.cola_circuito_comision%ROWTYPE;
 b vec_dietas.borrador_comision%ROWTYPE; r vec_dietas.comision_revision%ROWTYPE;
 etapa text; estado_esperado text;
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
 IF m->>'esquema'<>'vec.dietas.circuito-prelectura.v1'
    OR m->>'operacion'<>'preleer' OR estado_esperado IS NULL
    OR m->>'recurso_ref' !~ '^dco_[A-Za-z0-9_-]{22,128}$'
    OR m->>'unidad_ref' !~ '^[A-Za-z0-9:_-]{3,128}$'
    OR vec_dietas.cotejar_efecto_circuito_v1(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric
 THEN RAISE EXCEPTION 'prelectura Dietas incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_prelectura_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM cap->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'consumo AD3 prelectura incompatible' USING ERRCODE='PD003'; END IF;
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
 RETURN jsonb_build_object('resultado','concedido','comision',jsonb_build_object(
  'referencia',q.comision_ref,'relacion_ref',b.relacion_ref,'unidad_ref',b.unidad_ref,
  'version',r.version,'estado',r.estado,'asignacion_ref',r.asignacion_ref,
  'asignacion_version',r.asignacion_version,'grupo_dieta',r.grupo_dieta::text,
  'centro_ref',r.centro_ref,'administrativo_persona_ref',r.administrativo_persona_ref,
  'responsable_persona_ref',r.responsable_persona_ref));
END $funcion$;
ALTER FUNCTION vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_dietas.decidir_comision_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; c jsonb; a jsonb; cap jsonb; d jsonb; ctx jsonb;
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
 i:=m->'identidad'; c:=m->'comando'; a:=m->'asignacion'; etapa:=c->>'etapa';
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
 IF estado_previo IS NULL OR estado_nuevo IS NULL OR m->>'operacion'<>'decidir'
    OR c->>'referencia' IS DISTINCT FROM m->>'recurso_ref'
    OR c->>'unidad_ref' IS DISTINCT FROM m->>'unidad_ref'
    OR c->>'decision' NOT IN ('aprobar','devolver')
    OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9_-]{16,128}$'
    OR c->>'version_esperada' !~ '^[1-9][0-9]{0,17}$'
    OR c->>'motivo' IS NULL OR length(c->>'motivo')>600
    OR c->>'motivo' IS DISTINCT FROM btrim(c->>'motivo')
    OR c->>'motivo' ~ '[[:cntrl:]]'
    OR (c->>'decision'='devolver' AND length(c->>'motivo')<3)
    OR a->>'asignacion_ref' !~ '^ads_[A-Za-z0-9_-]{22,128}$'
    OR a->>'version' !~ '^[1-9][0-9]{0,17}$'
    OR a->>'unidad_ref' IS DISTINCT FROM m->>'unidad_ref'
 THEN RAISE EXCEPTION 'decisión Dietas inválida' USING ERRCODE='22023'; END IF;
 huella:=encode(sha256(convert_to(concat_ws(chr(31),i->>'persona_ref',m->>'recurso_ref',
  c->>'unidad_ref',etapa,c->>'decision',c->>'motivo',c->>'clave_idempotencia',
  c->>'version_esperada'),'UTF8')),'hex');
 IF m->>'huella_semantica' IS DISTINCT FROM huella
    OR vec_dietas.cotejar_efecto_circuito_v1(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
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
 -- Solo después del consumo nominal se abre la cola para la etapa y unidad
 -- exactas de la capacidad. No hay SELECT directo para el ejecutor.
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
    OR a->>'asignacion_ref' IS DISTINCT FROM q.asignacion_ref
    OR (a->>'version')::bigint IS DISTINCT FROM q.asignacion_version
    OR a->>'relacion_ref' IS DISTINCT FROM b.relacion_ref
    OR a->>'persona_ref' IS DISTINCT FROM q.persona_ref
    OR a->>'unidad_ref' IS DISTINCT FROM q.unidad_ref
    OR a->>'centro_ref' IS DISTINCT FROM anterior.centro_ref
    OR a->>'administrativo_persona_ref' IS DISTINCT FROM anterior.administrativo_persona_ref
    OR a->>'responsable_persona_ref' IS DISTINCT FROM anterior.responsable_persona_ref
 THEN RAISE EXCEPTION 'asignación Dietas incompatible' USING ERRCODE='PD003'; END IF;
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
 operacion:='circuito_'||etapa;
 SELECT * INTO recibo FROM vec_dietas.recibo_operacion_comision r
  WHERE r.persona_ref=q.persona_ref AND r.operacion=operacion
    AND r.clave_idempotencia=c->>'clave_idempotencia';
 IF FOUND THEN
  IF recibo.comision_ref IS DISTINCT FROM q.comision_ref
     OR recibo.huella_semantica_sha256 IS DISTINCT FROM huella
     OR recibo.actor_ref IS DISTINCT FROM i->>'actor_ref'
  THEN RAISE EXCEPTION 'conflicto de idempotencia Dietas' USING ERRCODE='PD002'; END IF;
  IF recibo.regla_ref IS DISTINCT FROM anterior.regla_ref
     OR recibo.regla_huella_sha256 IS DISTINCT FROM regla_huella
  THEN RAISE EXCEPTION 'recibo de regla Dietas incompatible' USING ERRCODE='PD003'; END IF;
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
  i->>'actor_ref',q.persona_ref,anterior.regla_ref,regla_huella,ahora)
 RETURNING * INTO recibo;
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
ALTER FUNCTION vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_dietas.listar_bandeja_comisiones_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $funcion$
#variable_conflict use_variable
DECLARE m jsonb; i jsonb; consulta jsonb; cap jsonb; d jsonb; ctx jsonb;
 v record; q vec_dietas.cola_circuito_comision%ROWTYPE;
 b vec_dietas.borrador_comision%ROWTYPE; r vec_dietas.comision_revision%ROWTYPE;
 etapa text; estado_esperado text; desde date; hasta date; cursor text;
 limite int; leidos int:=0; visibles int:=0; siguiente text:='';
 items jsonb:='[]'::jsonb;
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
 i:=m->'identidad'; consulta:=m->'consulta'; etapa:=consulta->>'etapa';
 estado_esperado:=CASE etapa WHEN 'revision' THEN 'enviado_pendiente_revision'
  WHEN 'autorizacion' THEN 'pendiente_autorizacion'
  WHEN 'liquidacion' THEN 'pendiente_liquidacion'
  WHEN 'fiscalizacion' THEN 'pendiente_fiscalizacion' END;
 IF m->>'operacion'<>'listar_bandeja' OR estado_esperado IS NULL
    OR consulta->>'unidad_ref' IS DISTINCT FROM m->>'unidad_ref'
    OR consulta->>'limit' !~ '^([1-9]|[1-4][0-9]|50)$'
    OR (coalesce(consulta->>'cursor','')<>'' AND consulta->>'cursor' !~ '^dco_[A-Za-z0-9_-]{22,128}$')
    OR (coalesce(consulta->>'fecha_desde','')<>'' AND consulta->>'fecha_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$')
    OR (coalesce(consulta->>'fecha_hasta','')<>'' AND consulta->>'fecha_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$')
 THEN RAISE EXCEPTION 'consulta de bandeja Dietas inválida' USING ERRCODE='22023'; END IF;
 BEGIN
  desde:=NULLIF(consulta->>'fecha_desde','')::date;
  hasta:=NULLIF(consulta->>'fecha_hasta','')::date;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'fechas de bandeja Dietas inválidas' USING ERRCODE='22023'; END;
 IF desde IS NOT NULL AND hasta IS NOT NULL AND desde>hasta THEN
  RAISE EXCEPTION 'intervalo de bandeja Dietas inválido' USING ERRCODE='22023';
 END IF;
 limite:=(consulta->>'limit')::int; cursor:=coalesce(consulta->>'cursor','');
 IF vec_dietas.cotejar_efecto_circuito_v1(p_material,p_capacidad,p_decision,p_contexto) IS NOT TRUE
    OR p_persona_version IS DISTINCT FROM (i->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (i->>'perfil_version')::numeric
    OR p_persona_version IS DISTINCT FROM (ctx->>'persona_version')::numeric
    OR p_perfil_version IS DISTINCT FROM (ctx->>'perfil_version')::numeric
 THEN RAISE EXCEPTION 'sello de bandeja Dietas incompatible' USING ERRCODE='PD003'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_bandeja_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR v.efecto_ref IS DISTINCT FROM m->>'recurso_ref'
    OR v.huella_efecto_sha256 IS DISTINCT FROM cap->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'consumo AD3 bandeja incompatible' USING ERRCODE='PD003'; END IF;
 PERFORM set_config('vec.dietas.circuito_etapa',etapa,true);
 PERFORM set_config('vec.dietas.circuito_unidad_ref',m->>'unidad_ref',true);
 -- El máximo de candidatos limita lecturas de historia obsoleta en una página.
 -- El cursor avanza por el último candidato examinado, incluso si se omite.
 FOR q IN SELECT cola.* FROM vec_dietas.cola_circuito_comision cola
  WHERE cola.etapa=etapa AND cola.unidad_ref=m->>'unidad_ref'
    AND (cursor='' OR cola.comision_ref>cursor)
    AND (desde IS NULL OR cola.fecha_inicio>=desde)
    AND (hasta IS NULL OR cola.fecha_inicio<=hasta)
  ORDER BY cola.comision_ref,cola.version DESC LIMIT 500
 LOOP
  leidos:=leidos+1; siguiente:=q.comision_ref;
  PERFORM set_config('vec.dietas.persona_ref',q.persona_ref,true);
  SELECT * INTO b FROM vec_dietas.borrador_comision WHERE referencia=q.comision_ref;
  IF NOT FOUND OR b.unidad_ref IS DISTINCT FROM q.unidad_ref THEN CONTINUE; END IF;
  SELECT * INTO r FROM vec_dietas.comision_revision
   WHERE comision_ref=q.comision_ref ORDER BY version DESC LIMIT 1;
  IF NOT FOUND OR r.version IS DISTINCT FROM q.version
     OR r.estado IS DISTINCT FROM estado_esperado
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
  THEN CONTINUE; END IF;
  BEGIN
   IF vec_personal.revalidar_asignacion_dietas_v1(b.relacion_ref,q.persona_ref,
     b.unidad_ref,q.asignacion_ref,q.asignacion_version,r.grupo_dieta,r.centro_ref,
     r.administrativo_persona_ref,r.responsable_persona_ref,current_date) IS NOT TRUE
   THEN CONTINUE; END IF;
  EXCEPTION WHEN SQLSTATE 'P7201' THEN CONTINUE;
  END;
  visibles:=visibles+1;
  IF visibles>limite THEN EXIT; END IF;
  items:=items||jsonb_build_array(jsonb_build_object('referencia',q.comision_ref,
   'estado',r.estado,'version',r.version,'fecha_inicio',r.fecha_inicio::text,
   'fecha_fin',r.fecha_fin::text));
 END LOOP;
 IF visibles<=limite AND leidos<500 THEN siguiente:=''; END IF;
 RETURN jsonb_build_object('items',items,'siguiente_cursor',siguiente);
END $funcion$;
ALTER FUNCTION vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
 vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_dietas_ejecutor;
COMMIT;
