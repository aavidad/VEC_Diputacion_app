\set ON_ERROR_STOP on
-- CT130: constancia del retorno de la persona titular después del cese CT115.
-- No altera disponibilidad, orden ni llamamientos de Bolsa. AD3-92 y CT115
-- son dependencias; la relación procede de CT75 y su raíz de seguimiento.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000130',0));

DO $pre$
DECLARE t text; f oid; v_origen text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' THEN RAISE EXCEPTION 'CT130: propietario incompatible' USING ERRCODE='55000'; END IF;
 FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','incorporacion_registro_v2','seguimiento_raiz_v2',
  'expediente_integral_actual','expediente_version_integral','actuacion_expediente_integral','outbox_expediente_integral',
  'control_cadenas_expediente_integral'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_contratacion_temporal.'||t)
      AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity) THEN
   RAISE EXCEPTION 'CT130: fuente % incompatible',t USING ERRCODE='55000'; END IF;
 END LOOP;
 IF to_regclass('vec_contratacion_temporal.reincorporacion_titular_v1') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'CT130: objeto preexistente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_reincorporacion_titular_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
    OR to_regprocedure('vec_contratacion_temporal.posicion_contrato_bolsa_v1(xid8)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin)
    OR to_regprocedure('vec_contratacion_temporal.validar_preparacion_ct115(jsonb,text,text,text)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.validar_confirmacion_ct115(jsonb,text,text,text,text,text,bytea,bytea,numeric,numeric)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.huella_contexto_go_ct115(jsonb,jsonb)') IS NULL THEN
  RAISE EXCEPTION 'CT130: CT115 o AD3-92 incompatibles' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
  WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
    AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
 IF right(v_origen,4)<>'])))' OR strpos(v_origen,'reincorporacion_titular_ct130')<>0
    OR strpos(v_origen,'''cese_nombramiento_ct115''::text')=0 THEN
  RAISE EXCEPTION 'CT130: origen de versión incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $origen$
DECLARE v text;
BEGIN
 SELECT pg_get_constraintdef(oid) INTO STRICT v FROM pg_constraint
  WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
    AND conname='expediente_version_integral_origen_version_check';
 ALTER TABLE vec_contratacion_temporal.expediente_version_integral DROP CONSTRAINT expediente_version_integral_origen_version_check;
 EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
  ||left(v,length(v)-4)||', ''reincorporacion_titular_ct130''::text])))';
END $origen$;

CREATE TABLE vec_contratacion_temporal.reincorporacion_titular_v1 (
 ambito_hmac text PRIMARY KEY CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]reincorporacion-titular[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
 huella_peticion_hmac text NOT NULL CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]reincorporacion-titular[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
 organizacion_ref text NOT NULL,
 expediente_ref text NOT NULL UNIQUE,
 relacion_ref text NOT NULL,
 version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 fecha_efectiva date NOT NULL CHECK (isfinite(fecha_efectiva)),
 documento_ref text NOT NULL,
 documento_sha256 text NOT NULL CHECK (documento_sha256 ~ '^[0-9a-f]{64}$' AND documento_sha256<>repeat('0',64)),
 cese_evento_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.cese_nombramiento_v1(evento_ref),
 cese_recibo_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.cese_nombramiento_v1(recibo_ref),
 incorporacion_recibo_ref text NOT NULL REFERENCES vec_contratacion_temporal.incorporacion_registro_v2(recibo_ref),
 reserva_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 evento_ref text NOT NULL UNIQUE,
 expediente_anterior_json jsonb NOT NULL CHECK (jsonb_typeof(expediente_anterior_json)='object'),
 expediente_siguiente_json jsonb NOT NULL CHECK (jsonb_typeof(expediente_siguiente_json)='object'),
 recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
 decision_ref text NOT NULL UNIQUE,
 decision_huella_sha256 text NOT NULL CHECK (decision_huella_sha256 ~ '^[0-9a-f]{64}$'),
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$'),
 politica_ref text NOT NULL,
 politica_version numeric(20,0) NOT NULL CHECK (politica_version BETWEEN 1 AND 9007199254740991),
 politica_huella_sha256 text NOT NULL CHECK (politica_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
 confirmada_en timestamptz(6) NOT NULL CHECK (confirmada_en>=registrada_en),
 transaccion_publicacion xid8 NOT NULL DEFAULT pg_current_xact_id(),
 FOREIGN KEY (expediente_ref,version_esperada) REFERENCES vec_contratacion_temporal.expediente_version_integral,
 CHECK (vec_contratacion_temporal.referencia_valida_ct115(organizacion_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(expediente_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(relacion_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(documento_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(actor_ref)
    AND vec_contratacion_temporal.referencia_valida_ct115(perfil_ref))
);
CREATE INDEX reincorporacion_titular_v1_publicacion_bolsa
 ON vec_contratacion_temporal.reincorporacion_titular_v1(transaccion_publicacion,evento_ref);
ALTER TABLE vec_contratacion_temporal.reincorporacion_titular_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.reincorporacion_titular_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_ct130 ON vec_contratacion_temporal.reincorporacion_titular_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
 organizacion_ref=current_setting('vec.ct130.organizacion_ref',true)
 AND expediente_ref=current_setting('vec.ct130.expediente_ref',true)
 AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE POLICY publicacion_ct130 ON vec_contratacion_temporal.reincorporacion_titular_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
 current_setting('vec.ct130.publicacion_bolsa',true)='activa'
 AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE POLICY insercion_ct130 ON vec_contratacion_temporal.reincorporacion_titular_v1 FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
 organizacion_ref=current_setting('vec.ct130.organizacion_ref',true)
 AND expediente_ref=current_setting('vec.ct130.expediente_ref',true)
 AND actor_ref=current_setting('vec.ct130.actor_ref',true)
 AND perfil_ref=current_setting('vec.ct130.perfil_ref',true)
 AND ambito_hmac=current_setting('vec.ct130.ambito_hmac',true)
 AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE POLICY verificador_bolsa_ct130 ON vec_contratacion_temporal.reincorporacion_titular_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
 evento_ref=current_setting('vec.ct130.verificacion_evento_ref',true)
 AND pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
 AND NOT pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
 AND NOT pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER'));
CREATE TRIGGER reincorporacion_titular_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.reincorporacion_titular_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.validar_material_reincorporacion_ct130(m jsonb)
RETURNS date LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE k text; f date;
BEGIN
 IF m IS NULL OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY[
  'organizacion_ref','expediente_ref','relacion_ref','version_esperada','actor_ref','perfil_ref',
  'fecha_efectiva','documento_ref','documento_sha256']) IS NOT TRUE
  OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='version_esperada' THEN 'number' ELSE 'string' END)
  OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version_esperada')::numeric>9007199254740990
  OR m->>'documento_sha256' !~ '^[0-9a-f]{64}$' OR m->>'documento_sha256'=repeat('0',64)
  OR m->>'fecha_efectiva' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
  RAISE EXCEPTION 'CT130: material inválido' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','relacion_ref','actor_ref','perfil_ref','documento_ref'] LOOP
  IF NOT vec_contratacion_temporal.referencia_valida_ct115(m->>k) THEN
   RAISE EXCEPTION 'CT130: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 f:=(m->>'fecha_efectiva')::date;
 IF to_char(f,'YYYY-MM-DD')<>m->>'fecha_efectiva' THEN RAISE EXCEPTION 'CT130: fecha inválida' USING ERRCODE='22023'; END IF;
 RETURN f;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
 RAISE EXCEPTION 'CT130: fecha inválida' USING ERRCODE='22023';
END $f$;

CREATE FUNCTION vec_contratacion_temporal.resultado_reincorporacion_ct130(r vec_contratacion_temporal.reincorporacion_titular_v1)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT jsonb_build_object('esquema','vec.contratacion-temporal.resultado-reincorporacion-titular.v1','resultado','confirmada',
  'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,'expediente_ref',r.expediente_ref,'relacion_ref',r.relacion_ref,
   'version_esperada',r.version_esperada,'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,
   'fecha_efectiva',to_char(r.fecha_efectiva,'YYYY-MM-DD'),'documento_ref',r.documento_ref,'documento_sha256',r.documento_sha256),
  'expediente',r.expediente_siguiente_json,
  'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
  'cese_evento_ref',r.cese_evento_ref,'cese_recibo_ref',r.cese_recibo_ref,'relacion_ref',r.relacion_ref,
  'ambito_idempotencia_hmac',r.ambito_hmac,'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
$$;

CREATE FUNCTION vec_contratacion_temporal.origen_reincorporacion_ct130(m jsonb)
RETURNS TABLE(relacion_ref text, incorporacion_recibo_ref text, cese_evento_ref text, cese_recibo_ref text, estado text)
LANGUAGE plpgsql STABLE SET search_path=pg_catalog SET row_security='on' AS $f$
DECLARE c vec_contratacion_temporal.cese_nombramiento_v1%ROWTYPE; i vec_contratacion_temporal.incorporacion_registro_v2%ROWTYPE;
 r vec_contratacion_temporal.seguimiento_raiz_v2%ROWTYPE;
BEGIN
 SELECT * INTO c FROM vec_contratacion_temporal.cese_nombramiento_v1
  WHERE organizacion_ref=m->>'organizacion_ref' AND expediente_ref=m->>'expediente_ref';
 IF NOT FOUND THEN estado:='sin_cese'; RETURN NEXT; RETURN; END IF;
 IF c.causa_clave<>'fin_sustitucion' OR c.justificante_tipo<>'comunicacion_reincorporacion'
    OR c.fecha_efecto IS DISTINCT FROM (m->>'fecha_efectiva')::date
    OR c.justificante_ref IS DISTINCT FROM m->>'documento_ref'
    OR c.justificante_sha256 IS DISTINCT FROM m->>'documento_sha256'
    OR c.estado<>'confirmada' THEN
  estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral o
   WHERE o.evento_ref=c.evento_ref AND o.expediente_ref=c.expediente_ref
     AND o.operacion_ref=c.reserva_ref AND o.version_expediente=c.version_esperada+1
     AND o.tipo_evento='ct.cese.v1') THEN
  estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 SELECT * INTO i FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE recibo_ref=c.incorporacion_ref;
 IF NOT FOUND THEN estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 SELECT * INTO r FROM vec_contratacion_temporal.seguimiento_raiz_v2 WHERE seguimiento_ref=i.seguimiento_ref;
 IF NOT FOUND OR i.organizacion_ref IS DISTINCT FROM c.organizacion_ref OR i.expediente_ref IS DISTINCT FROM c.expediente_ref
    OR r.organizacion_ref IS DISTINCT FROM c.organizacion_ref OR r.expediente_ref IS DISTINCT FROM c.expediente_ref
    OR i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}' IS DISTINCT FROM r.relacion_ref
    OR m->>'relacion_ref' IS DISTINCT FROM r.relacion_ref
    OR r.relacion_ref IS NULL THEN
  estado:='cese_no_coincide'; RETURN NEXT; RETURN; END IF;
 relacion_ref:=r.relacion_ref; incorporacion_recibo_ref:=i.recibo_ref;
 cese_evento_ref:=c.evento_ref; cese_recibo_ref:=c.recibo_ref; estado:='coincide'; RETURN NEXT;
END $f$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(
 p_operacion jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; aut jsonb:=p_operacion->'autorizacion';
 e text:='vec.contratacion-temporal.resultado-reincorporacion-titular.v1';
 r vec_contratacion_temporal.reincorporacion_titular_v1%ROWTYPE;
 v_fecha date; v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_origen record;
 v_decision jsonb; v_consumo record; v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text;
 v_secuencia_actuacion numeric; v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric;
 v_recibo jsonb; v_restriccion text; v_tabla text; v_esquema text;
BEGIN
 PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);
 PERFORM vec_contratacion_temporal.validar_confirmacion_ct115(p_operacion,
  'vec.contratacion-temporal.confirmar-reincorporacion-titular.v1','registrar_reincorporacion_titular','reincorporacion-titular',
  'contratacion_temporal.seguimiento.registrar_reincorporacion_titular','registrar_reincorporacion_titular',
  p_decision,p_motivo,p_persona_version,p_perfil_version);
 v_fecha:=vec_contratacion_temporal.validar_material_reincorporacion_ct130(m);
 IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'CT130: autorización incompleta' USING ERRCODE='42501'; END IF;
 v_version:=(m->>'version_esperada')::numeric;
 v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
 v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 PERFORM set_config('vec.ct130.organizacion_ref',m->>'organizacion_ref',true);
 PERFORM set_config('vec.ct130.expediente_ref',m->>'expediente_ref',true);
 PERFORM set_config('vec.ct130.actor_ref',m->>'actor_ref',true);
 PERFORM set_config('vec.ct130.perfil_ref',m->>'perfil_ref',true);
 PERFORM set_config('vec.ct130.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
 PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
 PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
 SELECT * INTO r FROM vec_contratacion_temporal.reincorporacion_titular_v1
  WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
 IF FOUND THEN
  IF r.huella_peticion_hmac IS DISTINCT FROM p_operacion->>'huella_peticion_hmac'
    OR r.organizacion_ref IS DISTINCT FROM m->>'organizacion_ref' OR r.expediente_ref IS DISTINCT FROM m->>'expediente_ref'
    OR r.relacion_ref IS DISTINCT FROM m->>'relacion_ref' OR r.version_esperada IS DISTINCT FROM v_version
    OR r.actor_ref IS DISTINCT FROM m->>'actor_ref' OR r.perfil_ref IS DISTINCT FROM m->>'perfil_ref'
    OR r.fecha_efectiva IS DISTINCT FROM v_fecha OR r.documento_ref IS DISTINCT FROM m->>'documento_ref'
    OR r.documento_sha256 IS DISTINCT FROM m->>'documento_sha256' THEN
   RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada'); END IF;
  IF r.decision_ref IS DISTINCT FROM aut->>'decision_ref' THEN
   RAISE EXCEPTION 'CT130: evidencia de repetición divergente' USING ERRCODE='42501'; END IF;
  RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
 END IF;
 SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
  JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
  WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
 IF NOT FOUND OR v_actual.version IS DISTINCT FROM v_version
    OR v_actual.agregado_json IS DISTINCT FROM p_operacion->'expediente_anterior'
    OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
    OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
    OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
    OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
    OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
  RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto'); END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.reincorporacion_titular_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
  RETURN jsonb_build_object('esquema',e,'resultado','reincorporacion_existente'); END IF;
 SELECT * INTO v_origen FROM vec_contratacion_temporal.origen_reincorporacion_ct130(m);
 IF v_origen.estado<>'coincide' THEN RETURN jsonb_build_object('esquema',e,'resultado',v_origen.estado); END IF;
 v_secuencia_actuacion:=jsonb_array_length(v_actual.agregado_json->'actuaciones')+1;
 v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
  'accion_clave','contratacion_temporal.seguimiento.registrar_reincorporacion_titular','actor_ref',m->>'actor_ref',
  'unidad_ref',v_actual.agregado_json#>>'{asignacion,unidad_ref}','recibo_ref',refs->>'recibo_ref',
  'realizada_en',p_operacion->'instante_efecto','fase_origen','nombramiento','fase_destino','nombramiento',
  'estado_origen','en_curso','estado_destino','en_curso','documentos_ref',jsonb_build_array(m->>'documento_ref'));
 v_siguiente:=v_actual.agregado_json||jsonb_build_object('version',v_version+1,'actualizado_en',p_operacion->'instante_efecto',
  'actuaciones',v_actual.agregado_json->'actuaciones'||jsonb_build_array(v_actuacion));
 IF v_instante<(v_actual.agregado_json->>'actualizado_en')::timestamptz
    OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
    OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
  RAISE EXCEPTION 'CT130: proyección divergente' USING ERRCODE='22023'; END IF;
 v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
  'fase_previa','nombramiento','estado_previo','en_curso');
 v_atributos:=jsonb_build_object('version_expediente',v_version::text,'relacion_ref',v_origen.relacion_ref,
  'fecha_efectiva',m->>'fecha_efectiva','documento_ref',m->>'documento_ref','documento_sha256',m->>'documento_sha256',
  'cese_evento_ref',v_origen.cese_evento_ref,'cese_recibo_ref',v_origen.cese_recibo_ref,
  'ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac','huella_peticion_hmac',p_operacion->>'huella_peticion_hmac',
  'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version',
  'politica_huella_sha256',pol->>'definicion_huella_sha256');
 IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos
    OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
  RAISE EXCEPTION 'CT130: contexto divergente' USING ERRCODE='42501'; END IF;
 v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct115(v_ambitos,v_atributos);
 IF aut->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
    OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
    OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
    OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
    OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
    OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.seguimiento.registrar_reincorporacion_titular'
    OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'reincorporacion_titular_contratacion_temporal'
    OR v_decision->>'finalidad' IS DISTINCT FROM 'registrar_reincorporacion_titular'
    OR v_decision->>'decision_ref' IS DISTINCT FROM aut->>'decision_ref' THEN
  RAISE EXCEPTION 'CT130: decisión divergente' USING ERRCODE='42501'; END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'CT130: vigencia agotada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_reincorporacion_titular_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.decision_ref IS DISTINCT FROM aut->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
    OR v_consumo.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'CT130: consumo divergente' USING ERRCODE='42501'; END IF;
 v_ahora:=date_trunc('microseconds',clock_timestamp());
 IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
  RAISE EXCEPTION 'CT130: vigencia final agotada' USING ERRCODE='42501'; END IF;
 v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
 v_prueba:=convert_to('VEC-CT-EXPEDIENTE-REINCORPORACION-CT130'||chr(10)||(m->>'expediente_ref')||chr(10)
  ||(v_version+1)::text||chr(10)||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)
  ||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.expediente_version_integral(
  expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
  flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
 VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
  v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'nombramiento','en_curso',
  'reincorporacion_titular_ct130',refs->>'reserva_ref',v_ahora);
 UPDATE vec_contratacion_temporal.expediente_integral_actual
  SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
  WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT130: CAS perdido' USING ERRCODE='40001'; END IF;
 v_prueba:=convert_to('VEC-CT-ACTUACION-REINCORPORACION-CT130'||chr(10)
  ||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
  expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
  actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
 VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
  encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
 SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
  FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT130: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;
 v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.reincorporacion-titular.v1',
  'evento_ref',refs->>'evento_ref','expediente_ref',m->>'expediente_ref','relacion_ref',v_origen.relacion_ref,
  'fecha_efectiva',m->>'fecha_efectiva','recibo_ct_ref',refs->>'recibo_ref',
  'cese_evento_ref',v_origen.cese_evento_ref,'cese_recibo_ref',v_origen.cese_recibo_ref)::text,'UTF8');
 INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
  evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
  payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
 VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,
  'ct.reincorporacion_titular.v1',v_payload,encode(sha256(v_payload),'hex'),v_anterior,
  encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
 UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
  SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
  WHERE control_id;
 v_recibo:=jsonb_build_object('operacion','registrar_reincorporacion_titular','organizacion_ref',m->>'organizacion_ref',
  'expediente_ref',m->>'expediente_ref','relacion_ref',v_origen.relacion_ref,'version_anterior',v_version,
  'version_resultante',v_version+1,'fase_resultante','nombramiento','estado_resultante','en_curso',
  'fecha_efectiva',m->>'fecha_efectiva','documento_ref',m->>'documento_ref',
  'documento_sha256',m->>'documento_sha256','recibo_ref',refs->>'recibo_ref',
  'recibo_ct_ref',refs->>'recibo_ref',
  'auditoria_ref',v_consumo.auditoria_ref,'evento_ref',refs->>'evento_ref',
  'cese_evento_ref',v_origen.cese_evento_ref,'cese_recibo_ref',v_origen.cese_recibo_ref,
  'actor_ref',m->>'actor_ref','registrada_en',p_operacion->'instante_efecto');
 INSERT INTO vec_contratacion_temporal.reincorporacion_titular_v1(
  ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,relacion_ref,version_esperada,actor_ref,perfil_ref,
  fecha_efectiva,documento_ref,documento_sha256,cese_evento_ref,cese_recibo_ref,incorporacion_recibo_ref,
  reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,recibo_json,
  decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,
  politica_ref,politica_version,politica_huella_sha256,registrada_en,confirmada_en)
 VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',
  m->>'expediente_ref',v_origen.relacion_ref,v_version,m->>'actor_ref',m->>'perfil_ref',v_fecha,
  m->>'documento_ref',m->>'documento_sha256',v_origen.cese_evento_ref,v_origen.cese_recibo_ref,v_origen.incorporacion_recibo_ref,
  refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',v_actual.agregado_json,v_siguiente,v_recibo,
  v_consumo.decision_ref,aut->>'decision_huella_sha256',v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,
  pol->>'definicion_ref',(pol->>'definicion_version')::numeric,pol->>'definicion_huella_sha256',v_instante,v_ahora);
 RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION WHEN unique_violation THEN
 GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
 IF v_esquema='vec_contratacion_temporal' AND v_tabla='reincorporacion_titular_v1' THEN
  RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='reincorporacion_titular_v1_pkey'
   THEN 'idempotencia_reutilizada' ELSE 'reincorporacion_existente' END); END IF;
 RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
 RAISE EXCEPTION 'CT130: entrada inválida' USING ERRCODE='22023';
END $f$;

CREATE FUNCTION vec_contratacion_temporal.preparar_reincorporacion_titular_v1(p_operacion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb:=p_operacion->'material'; pares jsonb; par jsonb; f date; r vec_contratacion_temporal.reincorporacion_titular_v1%ROWTYPE;
 v record; o record; e text:='vec.contratacion-temporal.resultado-reincorporacion-titular.v1';
BEGIN
 PERFORM vec_contratacion_temporal.exigir_sesion_ct115(true);
 pares:=vec_contratacion_temporal.validar_preparacion_ct115(p_operacion,
  'vec.contratacion-temporal.preparar-reincorporacion-titular.v1','registrar_reincorporacion_titular','reincorporacion-titular');
 f:=vec_contratacion_temporal.validar_material_reincorporacion_ct130(m);
 PERFORM set_config('vec.ct130.organizacion_ref',m->>'organizacion_ref',true);
 PERFORM set_config('vec.ct130.expediente_ref',m->>'expediente_ref',true);
 PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
 PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
 FOR par IN SELECT value FROM jsonb_array_elements(pares) LOOP
  SELECT * INTO r FROM vec_contratacion_temporal.reincorporacion_titular_v1 WHERE ambito_hmac=par->>'ambito_hmac';
  IF FOUND THEN
   IF r.huella_peticion_hmac IS DISTINCT FROM par->>'huella_peticion_hmac' OR r.organizacion_ref IS DISTINCT FROM m->>'organizacion_ref'
      OR r.expediente_ref IS DISTINCT FROM m->>'expediente_ref' OR r.relacion_ref IS DISTINCT FROM m->>'relacion_ref'
      OR r.version_esperada IS DISTINCT FROM (m->>'version_esperada')::numeric OR r.actor_ref IS DISTINCT FROM m->>'actor_ref'
      OR r.perfil_ref IS DISTINCT FROM m->>'perfil_ref' OR r.fecha_efectiva IS DISTINCT FROM f
      OR r.documento_ref IS DISTINCT FROM m->>'documento_ref' OR r.documento_sha256 IS DISTINCT FROM m->>'documento_sha256' THEN
    RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada'); END IF;
   RETURN vec_contratacion_temporal.resultado_reincorporacion_ct130(r);
  END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.reincorporacion_titular_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
  RETURN jsonb_build_object('esquema',e,'resultado','reincorporacion_existente'); END IF;
 SELECT a.version,v.agregado_json INTO v FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
  WHERE a.expediente_ref=m->>'expediente_ref';
 IF NOT FOUND OR v.version IS DISTINCT FROM (m->>'version_esperada')::numeric
    OR v.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
    OR v.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
    OR v.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
    OR NOT vec_contratacion_temporal.referencia_valida_ct115(v.agregado_json#>>'{asignacion,unidad_ref}') THEN
  RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto'); END IF;
 SELECT * INTO o FROM vec_contratacion_temporal.origen_reincorporacion_ct130(m);
 IF o.estado<>'coincide' THEN RETURN jsonb_build_object('esquema',e,'resultado',o.estado); END IF;
 RETURN jsonb_build_object('esquema',e,'resultado','preparada','material',m,'expediente',v.agregado_json,
  'referencias',p_operacion->'referencias_candidatas','relacion_ref',o.relacion_ref,
  'cese_evento_ref',o.cese_evento_ref,'cese_recibo_ref',o.cese_recibo_ref,
  'ambito_idempotencia_hmac',p_operacion#>>'{sellos_hmac,activo,ambito_hmac}',
  'huella_peticion_hmac',p_operacion#>>'{sellos_hmac,activo,huella_peticion_hmac}');
EXCEPTION WHEN invalid_text_representation OR numeric_value_out_of_range OR character_not_in_repertoire THEN
 RAISE EXCEPTION 'CT130: entrada inválida' USING ERRCODE='22023';
END $f$;

-- Feed mínimo CT→Bolsa. El payload es el del outbox, sin proyección de
-- disponibilidad. La posición de publicación es la de la transacción CT.
CREATE FUNCTION vec_contratacion_temporal.leer_reincorporaciones_bolsa_v1(p_desde_posicion bigint,p_desde_ref text,p_limite integer)
RETURNS TABLE(evento_ref text,evento jsonb,huella_sha256 text,origen_ref text,origen_posicion bigint,origen_creada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' AS $f$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
  RAISE EXCEPTION 'CT130: lectura no autorizada' USING ERRCODE='42501'; END IF;
 IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 OR (p_desde_posicion IS NULL)<>(p_desde_ref IS NULL)
    OR p_desde_posicion<0 OR octet_length(p_desde_ref)>512 THEN
  RAISE EXCEPTION 'CT130: cursor inválido' USING ERRCODE='22023'; END IF;
 PERFORM set_config('vec.ct130.publicacion_bolsa','activa',true);
 RETURN QUERY
 WITH base AS (
  SELECT r.evento_ref AS origen, r.confirmada_en AS creada,
   vec_contratacion_temporal.posicion_contrato_bolsa_v1(r.transaccion_publicacion) AS posicion,
   convert_from(o.payload_canonico,'UTF8')::jsonb AS cuerpo,o.payload_huella_sha256 AS huella
   FROM vec_contratacion_temporal.reincorporacion_titular_v1 r
   JOIN vec_contratacion_temporal.outbox_expediente_integral o ON o.evento_ref=r.evento_ref
    AND o.expediente_ref=r.expediente_ref AND o.tipo_evento='ct.reincorporacion_titular.v1'
  WHERE (r.transaccion_publicacion<pg_snapshot_xmin(pg_current_snapshot())
    OR r.transaccion_publicacion=pg_current_xact_id_if_assigned())
    AND (p_desde_posicion IS NULL OR
     (vec_contratacion_temporal.posicion_contrato_bolsa_v1(r.transaccion_publicacion),r.evento_ref)>(p_desde_posicion,p_desde_ref))
  ORDER BY 3,r.evento_ref LIMIT p_limite)
 SELECT b.origen,b.cuerpo,b.huella,b.origen,b.posicion,b.creada FROM base b ORDER BY b.posicion,b.origen;
 PERFORM set_config('vec.ct130.publicacion_bolsa','',true);
END $f$;

-- Verificación cruzada: solo Bolsa propietaria ve la fila solicitada. Se
-- cotejan origen, huella del payload y posición de transacción, además de
-- todos los campos inmutables del evento conservado en el outbox.
CREATE FUNCTION vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(
 p_origen_ref text,p_huella_sha256 text,p_posicion bigint)
RETURNS TABLE(evento_ref text,expediente_ref text,relacion_ref text,fecha_efectiva date,recibo_ct_ref text,cese_evento_ref text,cese_recibo_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' AS $f$
DECLARE r vec_contratacion_temporal.reincorporacion_titular_v1%ROWTYPE; o record; v_payload jsonb; v_esperado jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_origen_ref IS NULL OR octet_length(p_origen_ref) NOT BETWEEN 3 AND 512
    OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$' OR p_posicion IS NULL OR p_posicion<0 THEN
  RETURN; END IF;
 PERFORM set_config('vec.ct130.verificacion_evento_ref',p_origen_ref,true);
 SELECT t.* INTO r FROM vec_contratacion_temporal.reincorporacion_titular_v1 t WHERE t.evento_ref=p_origen_ref;
 IF NOT FOUND THEN PERFORM set_config('vec.ct130.verificacion_evento_ref','',true); RETURN; END IF;
 SELECT x.evento_ref,x.operacion_ref,x.expediente_ref,x.version_expediente,x.tipo_evento,x.payload_canonico,x.payload_huella_sha256
  INTO o FROM vec_contratacion_temporal.outbox_expediente_integral x WHERE x.evento_ref=p_origen_ref;
 PERFORM set_config('vec.ct130.verificacion_evento_ref','',true);
 IF NOT FOUND OR o.operacion_ref IS DISTINCT FROM r.reserva_ref
    OR o.expediente_ref IS DISTINCT FROM r.expediente_ref OR o.version_expediente IS DISTINCT FROM r.version_esperada+1
    OR o.tipo_evento IS DISTINCT FROM 'ct.reincorporacion_titular.v1'
    OR o.payload_huella_sha256 IS DISTINCT FROM p_huella_sha256
    OR o.payload_huella_sha256 IS DISTINCT FROM encode(sha256(o.payload_canonico),'hex')
    OR vec_contratacion_temporal.posicion_contrato_bolsa_v1(r.transaccion_publicacion) IS DISTINCT FROM p_posicion THEN
  RETURN; END IF;
 v_payload:=convert_from(o.payload_canonico,'UTF8')::jsonb;
 v_esperado:=jsonb_build_object('esquema','vec.contratacion-temporal.reincorporacion-titular.v1',
  'evento_ref',r.evento_ref,'expediente_ref',r.expediente_ref,'relacion_ref',r.relacion_ref,
  'fecha_efectiva',to_char(r.fecha_efectiva,'YYYY-MM-DD'),'recibo_ct_ref',r.recibo_ref,
  'cese_evento_ref',r.cese_evento_ref,'cese_recibo_ref',r.cese_recibo_ref);
 IF v_payload IS DISTINCT FROM v_esperado OR convert_to(v_payload::text,'UTF8') IS DISTINCT FROM o.payload_canonico THEN RETURN; END IF;
 evento_ref:=r.evento_ref; expediente_ref:=r.expediente_ref; relacion_ref:=r.relacion_ref;
 fecha_efectiva:=r.fecha_efectiva; recibo_ct_ref:=r.recibo_ref;
 cese_evento_ref:=r.cese_evento_ref; cese_recibo_ref:=r.cese_recibo_ref;
 RETURN NEXT;
END $f$;

-- Las únicas fachadas ejecutables por CT son las operaciones nominales. El
-- verificador tiene un único concesionario adicional: Bolsa propietaria.
DO $acl$
DECLARE f regprocedure; x record; destinatario text;
 fichas regprocedure[]:=ARRAY[
  'vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)'::regprocedure,
  'vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.leer_reincorporaciones_bolsa_v1(bigint,text,integer)'::regprocedure];
 auxiliares regprocedure[]:=ARRAY[
  'vec_contratacion_temporal.validar_material_reincorporacion_ct130(jsonb)'::regprocedure,
  'vec_contratacion_temporal.resultado_reincorporacion_ct130(vec_contratacion_temporal.reincorporacion_titular_v1)'::regprocedure,
  'vec_contratacion_temporal.origen_reincorporacion_ct130(jsonb)'::regprocedure];
 verificador regprocedure:='vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint)'::regprocedure;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid='vec_contratacion_temporal.reincorporacion_titular_v1'::regclass AND a.grantee<>c.relowner LOOP
  destinatario:=CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END;
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.reincorporacion_titular_v1 FROM %s',destinatario);
 END LOOP;
 REVOKE ALL ON TABLE vec_contratacion_temporal.reincorporacion_titular_v1 FROM PUBLIC;
 FOREACH f IN ARRAY fichas||auxiliares||ARRAY[verificador] LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND a.grantee<>p.proowner LOOP
   destinatario:=CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END;
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,destinatario);
  END LOOP;
 END LOOP;
 FOREACH f IN ARRAY fichas LOOP
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f::text);
 END LOOP;
 GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_bolsa_llamamientos_propietario;
 GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.verificar_reincorporacion_publicada_bolsa_v1(text,text,bigint)
  TO vec_bolsa_llamamientos_propietario;
 IF has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.reincorporacion_titular_v1',
     'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
    OR EXISTS (SELECT 1 FROM unnest(auxiliares) u WHERE has_function_privilege('vec_contratacion_temporal_ejecutor',u,'EXECUTE'))
    OR EXISTS (SELECT 1 FROM unnest(fichas) u WHERE NOT has_function_privilege('vec_contratacion_temporal_ejecutor',u,'EXECUTE'))
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',verificador,'EXECUTE')
    OR has_function_privilege('public',verificador,'EXECUTE') THEN
  RAISE EXCEPTION 'CT130: ACL efectiva incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
COMMENT ON TABLE vec_contratacion_temporal.reincorporacion_titular_v1 IS
 'CT130: constancia inmutable del retorno titular tras cese fin_sustitucion; no altera disponibilidad Bolsa.';
COMMIT;
