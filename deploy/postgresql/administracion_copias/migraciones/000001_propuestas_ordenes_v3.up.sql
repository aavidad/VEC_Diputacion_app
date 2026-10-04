\set ON_ERROR_STOP on
-- CS08-000001. AD142/AD143 → ADMIN: propuesta, revisión y orden privadas.
-- El consumidor de emisión depende de la fachada ADMIN en runtime, no al
-- instalar AD143. Los consumos previos de propuesta/revisión no la necesitan.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_administracion_copias:migracion:000001',0));
DO $pre$
DECLARE n text;f oid;
BEGIN
 IF to_regnamespace('vec_administracion_copias') IS NOT NULL THEN
  RAISE EXCEPTION 'copias_esquema_ya_instalado' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY['vec_administracion_copias_propietario','vec_administracion_copias_ejecutor','vec_autorizacion_atestada_v3_propietario'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=n AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls) THEN
   RAISE EXCEPTION 'copias_rol_tecnico_incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH n IN ARRAY ARRAY['consumir_propuesta_restauracion_copias_v3_atestada','consumir_revision_restauracion_copias_v3_atestada','consumir_orden_copias_v3_atestada'] LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||n||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.provolatile='v'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    AND EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE a.grantee='vec_administracion_copias_propietario'::regrole AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE a.grantee NOT IN(p.proowner,'vec_administracion_copias_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
   RAISE EXCEPTION 'copias_consumidor_ad143_incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
CREATE SCHEMA vec_administracion_copias AUTHORIZATION vec_administracion_copias_propietario;
SET LOCAL ROLE vec_administracion_copias_propietario;
REVOKE ALL ON SCHEMA vec_administracion_copias FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_administracion_copias TO vec_administracion_copias_ejecutor,vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_administracion_copias.propuesta(
 orden_ref text PRIMARY KEY,plan_bytes bytea NOT NULL CHECK(octet_length(plan_bytes) BETWEEN 1 AND 16384),
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[a-f0-9]{64}$'),persona_ref text NOT NULL,
 material bytea[] NOT NULL CHECK(array_length(material,1)=8),persona_version numeric(20,0) NOT NULL,perfil_version numeric(20,0) NOT NULL,
 decision_ref text NOT NULL UNIQUE,consumo_sha256 text NOT NULL UNIQUE,auditoria_ad3_ref text NOT NULL,
 recibo text NOT NULL UNIQUE,registrada_en timestamptz(6) NOT NULL);
CREATE TABLE vec_administracion_copias.revision(
 orden_ref text PRIMARY KEY REFERENCES vec_administracion_copias.propuesta,
 plan_sha256 text NOT NULL,persona_ref text NOT NULL,
 material bytea[] NOT NULL CHECK(array_length(material,1)=8),persona_version numeric(20,0) NOT NULL,perfil_version numeric(20,0) NOT NULL,
 decision_ref text NOT NULL UNIQUE,consumo_sha256 text NOT NULL UNIQUE,auditoria_ad3_ref text NOT NULL,
 recibo text NOT NULL UNIQUE,registrada_en timestamptz(6) NOT NULL);
CREATE TABLE vec_administracion_copias.control_destino(
 destino text PRIMARY KEY,epoca text NOT NULL,version numeric(20,0) NOT NULL CHECK(version>=0),fence numeric(20,0) NOT NULL CHECK(fence>=0));
CREATE TABLE vec_administracion_copias.orden(
 orden_ref text PRIMARY KEY REFERENCES vec_administracion_copias.revision,
 operacion_ref text NOT NULL UNIQUE,destino text NOT NULL,consumo_v3 text NOT NULL UNIQUE,
 decision_ref text NOT NULL,decision_sha256 text NOT NULL,
 orden_bytes bytea NOT NULL CHECK(octet_length(orden_bytes) BETWEEN 1 AND 16384),
 orden_sha256 text NOT NULL CHECK(orden_sha256 ~ '^[a-f0-9]{64}$'),
 auditoria_ad3_ref text NOT NULL,comprometida_en timestamptz(6) NOT NULL);
CREATE TABLE vec_administracion_copias.auditoria(
 referencia text PRIMARY KEY,orden_ref text NOT NULL,tipo text NOT NULL CHECK(tipo IN('propuesta','revision','orden')),
 auditoria_ad3_ref text NOT NULL,huella_sha256 text NOT NULL,instante timestamptz(6) NOT NULL,UNIQUE(orden_ref,tipo));
CREATE TABLE vec_administracion_copias.outbox(
 referencia text PRIMARY KEY,orden_ref text NOT NULL,tipo text NOT NULL CHECK(tipo IN('propuesta','revision','orden')),
 huella_sha256 text NOT NULL,instante timestamptz(6) NOT NULL,UNIQUE(orden_ref,tipo));
CREATE FUNCTION vec_administracion_copias.rechazar_mutacion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
BEGIN RAISE EXCEPTION 'copias_historia_inmutable' USING ERRCODE='42501'; END $f$;
REVOKE ALL ON FUNCTION vec_administracion_copias.rechazar_mutacion() FROM PUBLIC;

-- Validador interno puro. Conserva TODOS los bytes que serán comprometidos
-- por PDP; no reconstruye un plan diferente al aprobado por los actores.
CREATE FUNCTION vec_administracion_copias.plan_v1(p_plan bytea) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r jsonb;m jsonb;k text;
BEGIN
 IF p_plan IS NULL OR octet_length(p_plan) NOT BETWEEN 1 AND 16384 THEN
  RAISE EXCEPTION 'copias_plan_invalido' USING ERRCODE='22023'; END IF;
 BEGIN r:=convert_from(p_plan,'UTF8')::jsonb;m:=r->'datos';
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'copias_plan_invalido' USING ERRCODE='22023'; END;
 IF r->>'esquema' IS DISTINCT FROM 'vec.administracion.plan-orden-copias.v1'
    OR jsonb_typeof(m) IS DISTINCT FROM 'object'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(r) k) IS DISTINCT FROM ARRAY['datos','esquema']
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(m) k) IS DISTINCT FROM
     ARRAY['accion','aprobador_persona','caduca_en','conjunto','destino','emitida_en','epoca','fence','manifiesto_sha256','operacion','orden','politica','politica_sha256','preimagen_sha256','proponente_persona','solicitud_sha256','version_cas'] THEN
  RAISE EXCEPTION 'copias_plan_invalido' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['orden','operacion','conjunto','destino','proponente_persona','aprobador_persona','politica','epoca'] LOOP
  IF m->>k IS NULL OR m->>k !~ '^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,127}$' THEN
   RAISE EXCEPTION 'copias_plan_referencia_invalida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['solicitud_sha256','manifiesto_sha256','preimagen_sha256','politica_sha256'] LOOP
  IF m->>k IS NULL OR m->>k !~ '^[a-f0-9]{64}$' THEN RAISE EXCEPTION 'copias_plan_huella_invalida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF m->>'accion' IS NULL OR m->>'accion' NOT IN('capturar_conjunto','restaurar_conjunto','configurar_copias','aplicar_retencion')
    OR m->>'proponente_persona'=m->>'aprobador_persona'
    OR jsonb_typeof(m->'fence') IS DISTINCT FROM 'number' OR jsonb_typeof(m->'version_cas') IS DISTINCT FROM 'number'
    OR (m->>'fence')::numeric NOT BETWEEN 1 AND 18446744073709551615
    OR (m->>'version_cas')::numeric NOT BETWEEN 1 AND 18446744073709551615
    OR floor((m->>'fence')::numeric)<>(m->>'fence')::numeric OR floor((m->>'version_cas')::numeric)<>(m->>'version_cas')::numeric
    OR jsonb_typeof(m->'emitida_en') IS DISTINCT FROM 'string' OR jsonb_typeof(m->'caduca_en') IS DISTINCT FROM 'string'
    OR (m->>'emitida_en')::timestamptz >= (m->>'caduca_en')::timestamptz THEN
  RAISE EXCEPTION 'copias_plan_ventana_invalida' USING ERRCODE='22023'; END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_administracion_copias.plan_v1(bytea) FROM PUBLIC;

-- Propuesta/revisión: consumo exacto ACTUAL antes de toda escritura y replay.
CREATE FUNCTION vec_administracion_copias.registrar_control_v1(
 p_tipo text,p_plan bytea,p_version numeric,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb;h text;c jsonb;x record;xp record;prior record;propuesta record;material bytea[];n text;t timestamptz(6);r text;b bytea;
BEGIN
 IF current_user<>'vec_administracion_copias_propietario' OR session_user=current_user OR p_tipo NOT IN('propuesta','revision') THEN
  RAISE EXCEPTION 'copias_control_denegado' USING ERRCODE='42501'; END IF;
 m:=vec_administracion_copias.plan_v1(p_plan);h:=encode(sha256(p_plan),'hex');
 material:=ARRAY[p_capacidad,p_decision,p_motivo,p_contexto,p_payload,p_sobre,p_evidencia,p_raiz];
 FOREACH b IN ARRAY material LOOP
  IF b IS NULL OR octet_length(b) NOT BETWEEN 1 AND 1048576 THEN RAISE EXCEPTION 'copias_material_invalido' USING ERRCODE='22023'; END IF;
 END LOOP;
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF c->>'efecto_ref' IS DISTINCT FROM m->>'orden' OR c->>'huella_efecto_sha256' IS DISTINCT FROM h THEN
  RAISE EXCEPTION 'copias_control_plan_distinto' USING ERRCODE='42501'; END IF;
 IF p_tipo='propuesta' THEN
  IF p_version IS DISTINCT FROM 0::numeric THEN RAISE EXCEPTION 'copias_control_version_distinta' USING ERRCODE='40001'; END IF;
  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_propuesta_restauracion_copias_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  IF p_version IS DISTINCT FROM 1::numeric THEN RAISE EXCEPTION 'copias_control_version_distinta' USING ERRCODE='40001'; END IF;
  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_revision_restauracion_copias_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF x.efecto_ref IS DISTINCT FROM m->>'orden' OR x.huella_efecto_sha256 IS DISTINCT FROM h
    OR x.consumo_nuevo IS NULL OR x.persona_ref IS NULL OR x.persona_ref IS DISTINCT FROM
     (CASE WHEN p_tipo='propuesta' THEN m->>'proponente_persona' ELSE m->>'aprobador_persona' END) THEN
  RAISE EXCEPTION 'copias_control_persona_distinta' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_administracion_copias:orden:'||(m->>'orden'),0));
 t:=date_trunc('microseconds',clock_timestamp());
 IF t<(m->>'emitida_en')::timestamptz OR t>=(m->>'caduca_en')::timestamptz THEN
  RAISE EXCEPTION 'copias_control_caducado' USING ERRCODE='42501'; END IF;
 IF p_tipo='propuesta' THEN
  SELECT * INTO prior FROM vec_administracion_copias.propuesta WHERE orden_ref=m->>'orden';
 ELSE
  SELECT * INTO STRICT propuesta FROM vec_administracion_copias.propuesta WHERE orden_ref=m->>'orden' FOR SHARE;
  IF propuesta.plan_bytes IS DISTINCT FROM p_plan OR propuesta.persona_ref=x.persona_ref THEN
   RAISE EXCEPTION 'copias_revision_propuesta_distinta' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT xp FROM vec_autorizacion_atestada_v3.consumir_propuesta_restauracion_copias_v3_atestada(
   propuesta.material[1],propuesta.material[2],propuesta.material[3],propuesta.material[4],propuesta.persona_version,propuesta.perfil_version,
   propuesta.material[5],propuesta.material[6],propuesta.material[7],propuesta.material[8]);
  IF xp.consumo_nuevo IS DISTINCT FROM false OR xp.persona_ref IS DISTINCT FROM propuesta.persona_ref
     OR xp.decision_ref IS DISTINCT FROM propuesta.decision_ref OR xp.consumo_huella_sha256 IS DISTINCT FROM propuesta.consumo_sha256 THEN
   RAISE EXCEPTION 'copias_proponente_no_vigente' USING ERRCODE='42501'; END IF;
  SELECT * INTO prior FROM vec_administracion_copias.revision WHERE orden_ref=m->>'orden';
 END IF;
 IF FOUND THEN
  IF prior.material IS DISTINCT FROM material OR prior.plan_sha256 IS DISTINCT FROM h
     OR prior.persona_version IS DISTINCT FROM p_persona_version OR prior.perfil_version IS DISTINCT FROM p_perfil_version
     OR prior.persona_ref IS DISTINCT FROM x.persona_ref OR prior.decision_ref IS DISTINCT FROM x.decision_ref
     OR prior.consumo_sha256 IS DISTINCT FROM x.consumo_huella_sha256 OR x.consumo_nuevo THEN
   RAISE EXCEPTION 'copias_control_replay_distinto' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('orden',prior.orden_ref,'plan_sha256',h,'persona_ref',prior.persona_ref,
   'version',CASE WHEN p_tipo='propuesta' THEN 1 ELSE 2 END,'recibo',prior.recibo,'registrada_en',prior.registrada_en,'replay',true);
 END IF;
 IF NOT x.consumo_nuevo THEN RAISE EXCEPTION 'copias_control_replay_sin_registro' USING ERRCODE='42501'; END IF;
 r:='recibo:'||encode(sha256(convert_to(p_tipo||':'||h||':'||x.consumo_huella_sha256,'UTF8')),'hex');
 IF p_tipo='propuesta' THEN
  INSERT INTO vec_administracion_copias.propuesta VALUES(m->>'orden',p_plan,h,x.persona_ref,material,p_persona_version,p_perfil_version,
   x.decision_ref,x.consumo_huella_sha256,x.auditoria_ref,r,t);
 ELSE
  INSERT INTO vec_administracion_copias.revision VALUES(m->>'orden',h,x.persona_ref,material,p_persona_version,p_perfil_version,
   x.decision_ref,x.consumo_huella_sha256,x.auditoria_ref,r,t);
 END IF;
 INSERT INTO vec_administracion_copias.auditoria VALUES('auditoria:'||substr(r,8),m->>'orden',p_tipo,x.auditoria_ref,h,t);
 INSERT INTO vec_administracion_copias.outbox VALUES('outbox:'||substr(r,8),m->>'orden',p_tipo,h,t);
 RETURN jsonb_build_object('orden',m->>'orden','plan_sha256',h,'persona_ref',x.persona_ref,
  'version',CASE WHEN p_tipo='propuesta' THEN 1 ELSE 2 END,'recibo',r,'registrada_en',t,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_administracion_copias.registrar_control_v1(text,bytea,numeric,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
CREATE FUNCTION vec_administracion_copias.registrar_propuesta_v1(
 p_plan bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
RETURN vec_administracion_copias.registrar_control_v1('propuesta',p_plan,0,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
CREATE FUNCTION vec_administracion_copias.registrar_revision_v1(
 p_plan bytea,p_version numeric,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
RETURN vec_administracion_copias.registrar_control_v1('revision',p_plan,p_version,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
REVOKE ALL ON FUNCTION vec_administracion_copias.registrar_propuesta_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_administracion_copias.registrar_revision_v1(bytea,numeric,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion_copias.registrar_propuesta_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_administracion_copias_ejecutor;
GRANT EXECUTE ON FUNCTION vec_administracion_copias.registrar_revision_v1(bytea,numeric,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_administracion_copias_ejecutor;

-- Fachada privada propiedad ADMIN. AD143 nunca lee sus tablas. Revalida los
-- dos consumos históricos exactos por sus puertos centrales, bajo el mismo TX.
CREATE FUNCTION vec_administracion_copias.acreditar_doble_control_orden_v1(
 p_orden text,p_sha text,p_emisor text,p_emitida timestamptz,p_caduca timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE p record;r record;m jsonb;x record;y record;t timestamptz(6);
BEGIN
 IF current_user<>'vec_administracion_copias_propietario' OR session_user=current_user
    OR p_emisor IS NULL OR p_emisor='' OR p_sha IS NULL OR p_sha !~ '^[a-f0-9]{64}$'
    OR p_emitida IS NULL OR p_caduca IS NULL OR p_emitida>=p_caduca THEN
  RAISE EXCEPTION 'copias_doble_control_denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_administracion_copias.propuesta WHERE orden_ref=p_orden FOR SHARE;
 SELECT * INTO STRICT r FROM vec_administracion_copias.revision WHERE orden_ref=p_orden FOR SHARE;
 m:=vec_administracion_copias.plan_v1(p.plan_bytes);t:=date_trunc('microseconds',clock_timestamp());
 IF p.plan_sha256 IS DISTINCT FROM p_sha OR r.plan_sha256 IS DISTINCT FROM p_sha
    OR encode(sha256(p.plan_bytes),'hex') IS DISTINCT FROM p_sha OR m->>'orden' IS DISTINCT FROM p_orden
    OR p.persona_ref IS NOT DISTINCT FROM r.persona_ref
    OR p.persona_ref IS DISTINCT FROM m->>'proponente_persona' OR r.persona_ref IS DISTINCT FROM m->>'aprobador_persona'
    OR p_emitida<(m->>'emitida_en')::timestamptz OR p_caduca>(m->>'caduca_en')::timestamptz
    OR t<(m->>'emitida_en')::timestamptz OR t>=(m->>'caduca_en')::timestamptz THEN
  RAISE EXCEPTION 'copias_doble_control_plan_distinto' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_propuesta_restauracion_copias_v3_atestada(
  p.material[1],p.material[2],p.material[3],p.material[4],p.persona_version,p.perfil_version,p.material[5],p.material[6],p.material[7],p.material[8]);
 SELECT * INTO STRICT y FROM vec_autorizacion_atestada_v3.consumir_revision_restauracion_copias_v3_atestada(
  r.material[1],r.material[2],r.material[3],r.material[4],r.persona_version,r.perfil_version,r.material[5],r.material[6],r.material[7],r.material[8]);
 IF x.consumo_nuevo IS DISTINCT FROM false OR y.consumo_nuevo IS DISTINCT FROM false
    OR x.persona_ref IS DISTINCT FROM p.persona_ref OR y.persona_ref IS DISTINCT FROM r.persona_ref
    OR x.decision_ref IS DISTINCT FROM p.decision_ref OR y.decision_ref IS DISTINCT FROM r.decision_ref
    OR x.consumo_huella_sha256 IS DISTINCT FROM p.consumo_sha256 OR y.consumo_huella_sha256 IS DISTINCT FROM r.consumo_sha256
    OR x.efecto_ref IS DISTINCT FROM p_orden OR y.efecto_ref IS DISTINCT FROM p_orden
    OR x.huella_efecto_sha256 IS DISTINCT FROM p_sha OR y.huella_efecto_sha256 IS DISTINCT FROM p_sha THEN
  RAISE EXCEPTION 'copias_doble_control_autoridad_distinta' USING ERRCODE='42501'; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_administracion_copias.acreditar_doble_control_orden_v1(text,text,text,timestamptz,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion_copias.acreditar_doble_control_orden_v1(text,text,text,timestamptz,timestamptz) TO vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_administracion_copias.comprometer_orden_v1(
 p_orden bytea,p_plan bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE root jsonb;plan jsonb;m jsonb;c jsonb;d jsonb;x record;existing record;control record;
 h text;ah timestamptz(6);expected_keys text[];k text;approved record;
BEGIN
 IF current_user<>'vec_administracion_copias_propietario' OR session_user=current_user
    OR octet_length(p_orden) NOT BETWEEN 1 AND 16384 THEN
  RAISE EXCEPTION 'CS08: material inválido' USING ERRCODE='42501'; END IF;
 BEGIN root:=convert_from(p_orden,'UTF8')::jsonb;m:=root->'datos';plan:=convert_from(p_plan,'UTF8')::jsonb;
 PERFORM vec_administracion_copias.plan_v1(p_plan);
 c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CS08: material inválido' USING ERRCODE='22023'; END;
 expected_keys:=ARRAY['accion','aprobador_persona','auditoria','caduca_en','conjunto','consumo_v3','decision_sha256','decision_v3','destino','emitida_en','epoca','fence','manifiesto_sha256','operacion','orden','outbox','politica','politica_sha256','preimagen_sha256','proponente_persona','solicitud_sha256','version_cas'];
 h:=encode(sha256(p_orden),'hex');ah:=date_trunc('microseconds',clock_timestamp());
 IF root->>'esquema' IS DISTINCT FROM 'vec.administracion.orden-copias.v1'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(root) k) IS DISTINCT FROM ARRAY['datos','esquema']
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(m) k) IS DISTINCT FROM expected_keys
    OR m->>'accion' NOT IN ('capturar_conjunto','restaurar_conjunto','configurar_copias','aplicar_retencion')
    OR m->>'proponente_persona' IS NOT DISTINCT FROM m->>'aprobador_persona'
    OR m->>'decision_v3' IS DISTINCT FROM d->>'decision_ref'
    OR m->>'decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR c->>'efecto_ref' IS DISTINCT FROM m->>'orden'
    OR plan->>'esquema' IS DISTINCT FROM 'vec.administracion.plan-orden-copias.v1'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(plan) k) IS DISTINCT FROM ARRAY['datos','esquema']
    OR plan->'datos' IS DISTINCT FROM (m-ARRAY['decision_v3','decision_sha256','consumo_v3','auditoria','outbox'])
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM encode(sha256(p_plan),'hex')
    OR d->>'accion' IS DISTINCT FROM 'administracion.copias.orden.emitir'
    OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'orden_copia'
    OR d->>'finalidad' IS DISTINCT FROM 'emitir_orden_copia'
    OR d->'campos_permitidos' IS DISTINCT FROM '["orden","recibo"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR (m->>'fence')::numeric NOT BETWEEN 1 AND 18446744073709551615
    OR (m->>'version_cas')::numeric NOT BETWEEN 1 AND 18446744073709551615
    OR ah < (m->>'emitida_en')::timestamptz OR ah >= (m->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'CS08: orden no ligada' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY ARRAY['orden','operacion','conjunto','destino','proponente_persona','aprobador_persona','politica','decision_v3','auditoria','outbox','epoca'] LOOP
  IF m->>k IS NULL OR m->>k !~ '^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,127}$' THEN
   RAISE EXCEPTION 'CS08: referencia inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['solicitud_sha256','manifiesto_sha256','preimagen_sha256','politica_sha256','decision_sha256','consumo_v3'] LOOP
  IF m->>k IS NULL OR m->>k !~ '^[a-f0-9]{64}$' THEN
   RAISE EXCEPTION 'CS08: huella inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF jsonb_typeof(m->'fence') IS DISTINCT FROM 'number' OR jsonb_typeof(m->'version_cas') IS DISTINCT FROM 'number'
    OR floor((m->>'fence')::numeric)<>(m->>'fence')::numeric
    OR floor((m->>'version_cas')::numeric)<>(m->>'version_cas')::numeric
    OR m->>'accion' IS NULL OR m->>'emitida_en' IS NULL OR m->>'caduca_en' IS NULL
    OR (m->>'emitida_en')::timestamptz >= (m->>'caduca_en')::timestamptz THEN
  RAISE EXCEPTION 'CS08: ventana o secuencia inválida' USING ERRCODE='22023'; END IF;
 -- AD143 DEBE revalidar gobierno/session/personas/política y doble aprobación
 -- canónica exacta, incluyendo manifiesto/preimagen/destino/caducidad/epoch.
 -- No basta un candidato o una decisión histórica. Ausencia deniega en $pre$.
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_orden_copias_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.decision_ref IS DISTINCT FROM m->>'decision_v3' OR x.efecto_ref IS DISTINCT FROM m->>'orden'
    OR x.huella_efecto_sha256 IS DISTINCT FROM encode(sha256(p_plan),'hex') OR x.consumo_huella_sha256 IS DISTINCT FROM m->>'consumo_v3'
    OR x.auditoria_ref IS NULL OR x.consumo_nuevo IS NULL THEN
  RAISE EXCEPTION 'CS08: consumo incoherente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT approved FROM vec_administracion_copias.propuesta WHERE orden_ref=m->>'orden' FOR SHARE;
 IF approved.plan_bytes IS DISTINCT FROM p_plan THEN RAISE EXCEPTION 'copias_orden_plan_distinto' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_administracion_copias:destino:'||(m->>'destino'),0));
 SELECT * INTO existing FROM vec_administracion_copias.orden WHERE orden_ref=m->>'orden' OR operacion_ref=m->>'operacion';
 IF FOUND THEN
  IF existing.orden_bytes IS DISTINCT FROM p_orden OR existing.consumo_v3 IS DISTINCT FROM x.consumo_huella_sha256 THEN
   RAISE EXCEPTION 'CS08: idempotencia distinta' USING ERRCODE='23505'; END IF;
  RETURN existing.orden_bytes;
 END IF;
 IF NOT x.consumo_nuevo THEN RAISE EXCEPTION 'CS08: replay sin orden' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_administracion_copias.control_destino VALUES(m->>'destino',m->>'epoca',0,0) ON CONFLICT DO NOTHING;
 SELECT * INTO STRICT control FROM vec_administracion_copias.control_destino WHERE destino=m->>'destino' FOR UPDATE;
 IF control.epoca IS DISTINCT FROM m->>'epoca' OR control.version+1<>(m->>'version_cas')::numeric
    OR control.fence>=(m->>'fence')::numeric THEN RAISE EXCEPTION 'CS08: versión o cercado obsoleto' USING ERRCODE='40001'; END IF;
 UPDATE vec_administracion_copias.control_destino SET version=control.version+1,fence=(m->>'fence')::numeric WHERE destino=m->>'destino';
 INSERT INTO vec_administracion_copias.orden VALUES(m->>'orden',m->>'operacion',m->>'destino',m->>'consumo_v3',m->>'decision_v3',m->>'decision_sha256',p_orden,h,x.auditoria_ref,ah);
 INSERT INTO vec_administracion_copias.auditoria VALUES(m->>'auditoria',m->>'orden','orden',x.auditoria_ref,h,ah);
 INSERT INTO vec_administracion_copias.outbox VALUES(m->>'outbox',m->>'orden','orden',h,ah);
 RETURN p_orden;
END $f$;
REVOKE ALL ON FUNCTION vec_administracion_copias.comprometer_orden_v1(bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion_copias.comprometer_orden_v1(bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_administracion_copias_ejecutor;
-- Trusted publisher's committed reread only. Public HTTP never mounts this
-- infrastructure role or function; current external anchor is mandatory after it.
CREATE FUNCTION vec_administracion_copias.leer_orden_comprometida_v1(p_ref text)
 RETURNS bytea LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 RETURN (SELECT o.orden_bytes FROM vec_administracion_copias.orden o WHERE o.orden_ref=p_ref);
REVOKE ALL ON FUNCTION vec_administracion_copias.leer_orden_comprometida_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion_copias.leer_orden_comprometida_v1(text) TO vec_administracion_copias_ejecutor;

DO $acl$
DECLARE t text;a record;p record;
BEGIN
 FOREACH t IN ARRAY ARRAY['propuesta','revision','control_destino','orden','auditoria','outbox'] LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_administracion_copias.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_administracion_copias.%I FROM PUBLIC',t);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
   WHERE c.oid=to_regclass('vec_administracion_copias.'||t) AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_administracion_copias.%I FROM %I',t,pg_get_userbyid(a.grantee));
  END LOOP;
  FOR a IN SELECT DISTINCT x.grantee FROM pg_type y,LATERAL aclexplode(coalesce(y.typacl,acldefault('T',y.typowner))) x
   WHERE y.oid=to_regtype('vec_administracion_copias.'||t) AND x.grantee<>0 AND x.grantee<>y.typowner LOOP
   EXECUTE format('REVOKE ALL ON TYPE vec_administracion_copias.%I FROM %I',t,pg_get_userbyid(a.grantee));
  END LOOP;
  EXECUTE format('ALTER TABLE vec_administracion_copias.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_administracion_copias.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario ON vec_administracion_copias.%I TO vec_administracion_copias_propietario USING (true) WITH CHECK (true)',t);
  IF t<>'control_destino' THEN
   EXECUTE format('CREATE TRIGGER no_modificar BEFORE UPDATE OR DELETE ON vec_administracion_copias.%I FOR EACH ROW EXECUTE FUNCTION vec_administracion_copias.rechazar_mutacion()',t);
   EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_administracion_copias.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_administracion_copias.rechazar_mutacion()',t);
  END IF;
 END LOOP;
 FOR p IN SELECT p.oid,p.proowner,p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_administracion_copias' LOOP
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc f,LATERAL aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) x
   WHERE f.oid=p.oid AND x.grantee<>0 AND x.grantee<>p.proowner
   AND NOT(x.grantee='vec_administracion_copias_ejecutor'::regrole AND p.proname IN('registrar_propuesta_v1','registrar_revision_v1','comprometer_orden_v1','leer_orden_comprometida_v1'))
   AND NOT(x.grantee='vec_autorizacion_atestada_v3_propietario'::regrole AND p.proname='acreditar_doble_control_orden_v1') LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',p.oid::regprocedure,pg_get_userbyid(a.grantee));
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
