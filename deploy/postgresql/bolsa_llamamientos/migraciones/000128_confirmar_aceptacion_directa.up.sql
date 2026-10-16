\set ON_ERROR_STOP on
-- B128: confirmación RRHH de una aceptación directa B30/B17. La fila de
-- resolución conserva un evento pendiente; no acredita entrega externa.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000128',0));
DO $pre$ BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.trazar_situacion_participacion_v1()') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='vec_bolsa_llamamientos.situacion_participacion'::regclass
  AND tgname='situacion_participacion_traza_valores' AND tgenabled='O')
 OR to_regclass('vec_bolsa_llamamientos.politica_transiciones_gobierno') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.confirmacion_aceptacion_directa') IS NOT NULL THEN
  RAISE EXCEPTION 'B128: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.confirmacion_aceptacion_directa (
 respuesta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.respuesta_portal_llamamiento(respuesta_ref),
 resolucion_ref text NOT NULL UNIQUE CHECK(resolucion_ref~'^resolucion:respuesta-revision:[a-f0-9]{64}$'),
 bolsa_ref text NOT NULL, participacion_ref text NOT NULL, llamamiento_ref text NOT NULL,
 actor_ref text NOT NULL, motivo text NOT NULL CHECK(octet_length(motivo) BETWEEN 1 AND 1000 AND motivo=btrim(motivo)),
 clave_idempotencia text NOT NULL UNIQUE CHECK(octet_length(clave_idempotencia) BETWEEN 8 AND 256
  AND clave_idempotencia~'^[A-Za-z0-9:_-]+$'),
 canon_original text NOT NULL CHECK(octet_length(canon_original) BETWEEN 2 AND 16384),
 canon_sha256 text NOT NULL CHECK(canon_sha256~'^[a-f0-9]{64}$'),
 ambito_sha256 text NOT NULL CHECK(ambito_sha256~'^[a-f0-9]{64}$'),
 situacion_desde_esperada timestamptz(6) NOT NULL,
 situacion_desde_nueva timestamptz(6) NOT NULL,
 politica_version bigint NOT NULL REFERENCES vec_bolsa_llamamientos.politica_transiciones_situacion(version),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref~'^recibo:respuesta-revision:[a-f0-9]{64}$'),
 decision_ref text NOT NULL UNIQUE, auditoria_ref text NOT NULL UNIQUE,
 evento_ref text NOT NULL UNIQUE CHECK(evento_ref~'^evento:respuesta-revision:[a-f0-9]{64}$'),
 evento_canonico bytea NOT NULL CHECK(octet_length(evento_canonico) BETWEEN 2 AND 4096),
 evento_sha256 text NOT NULL CHECK(evento_sha256~'^[a-f0-9]{64}$' AND evento_sha256=encode(sha256(evento_canonico),'hex')),
 resuelta_en timestamptz(6) NOT NULL CHECK(isfinite(resuelta_en)),
 FOREIGN KEY(participacion_ref,situacion_desde_nueva)
  REFERENCES vec_bolsa_llamamientos.situacion_participacion(participacion_ref,desde)
);
ALTER TABLE vec_bolsa_llamamientos.confirmacion_aceptacion_directa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.confirmacion_aceptacion_directa FORCE ROW LEVEL SECURITY;
CREATE POLICY confirmacion_aceptacion_directa_propietario ON vec_bolsa_llamamientos.confirmacion_aceptacion_directa
 TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.confirmacion_aceptacion_directa FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.confirmacion_aceptacion_directa FROM PUBLIC;
CREATE TRIGGER confirmacion_aceptacion_directa_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.confirmacion_aceptacion_directa FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

CREATE FUNCTION vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(
 p_canon text,p_contexto_recurso bytea,
 p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(recibo jsonb,reutilizada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE q jsonb; x jsonb; d jsonb; c jsonb; consumo record; previo record; r record; l record; s record; politica record;
 h text; hc text; ha text; refhash text; recibo_ref text; resolucion_ref text; evento_ref text;
 esperado timestamptz; ahora timestamptz:=date_trunc('microseconds',clock_timestamp()); evento bytea;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR (SELECT setting::bigint FROM pg_settings WHERE name='statement_timeout') NOT BETWEEN 1 AND 15000
 OR (SELECT setting::bigint FROM pg_settings WHERE name='idle_in_transaction_session_timeout') NOT BETWEEN 1 AND 20000
 OR p_canon IS NULL OR octet_length(p_canon) NOT BETWEEN 2 AND 16384
 OR p_contexto_recurso IS NULL OR octet_length(p_contexto_recurso) NOT BETWEEN 2 AND 2048 THEN
  RAISE EXCEPTION 'B128: acto denegado' USING ERRCODE='42501'; END IF;
 BEGIN q:=p_canon::jsonb; x:=convert_from(p_contexto_recurso,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  esperado:=(q->>'situacion_desde_esperada')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B128: material inválido' USING ERRCODE='22023'; END;
 IF q->>'esquema' IS DISTINCT FROM 'vec.bolsa.respuesta-revision.confirmar-aceptacion.v1'
 OR q->>'bolsa_ref' IS NULL OR octet_length(q->>'bolsa_ref') NOT BETWEEN 3 AND 256
 OR q->>'participacion_ref' IS NULL OR octet_length(q->>'participacion_ref') NOT BETWEEN 3 AND 512
 OR q->>'llamamiento_ref' IS NULL OR octet_length(q->>'llamamiento_ref') NOT BETWEEN 3 AND 512
 OR q->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR q->>'destino' IS DISTINCT FROM 'pendiente_incorporacion'
 OR q->>'motivo' IS NULL OR octet_length(q->>'motivo') NOT BETWEEN 1 AND 1000 OR q->>'motivo' IS DISTINCT FROM btrim(q->>'motivo')
 OR octet_length(q->>'clave_idempotencia') NOT BETWEEN 8 AND 256
 OR q->>'clave_idempotencia' !~ '^[A-Za-z0-9:_-]+$'
 OR esperado IS NULL OR NOT isfinite(esperado)
 OR q->>'situacion_desde_esperada' IS DISTINCT FROM to_char(esperado AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
 OR (SELECT count(*) FROM jsonb_object_keys(q))<>9 THEN
  RAISE EXCEPTION 'B128: comando inválido' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to(p_canon,'UTF8')),'hex');
 IF jsonb_typeof(x#>'{ambitos}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(x#>'{atributos}') IS DISTINCT FROM 'object'
 OR coalesce(x#>>'{ambitos,unidad_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$'
 OR coalesce(x#>>'{ambitos,ambito_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$'
 OR x IS DISTINCT FROM jsonb_build_object('ambitos',jsonb_build_object(
  'ambito_ref',x#>>'{ambitos,ambito_ref}','unidad_ref',x#>>'{ambitos,unidad_ref}'),
  'atributos',jsonb_build_object('confirmacion_aceptacion_sha256',h))
 OR convert_from(p_contexto_recurso,'UTF8') IS DISTINCT FROM format(
  '{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{"confirmacion_aceptacion_sha256":%s}}',
  to_json(x#>>'{ambitos,ambito_ref}'),to_json(x#>>'{ambitos,unidad_ref}'),to_json(h)) THEN
  RAISE EXCEPTION 'B128: ámbito no canónico' USING ERRCODE='42501'; END IF;
 hc:=encode(sha256(p_contexto_recurso),'hex');
 ha:=encode(sha256(convert_to(format('{"ambito_ref":%s,"unidad_ref":%s}',
  to_json(x#>>'{ambitos,ambito_ref}'),to_json(x#>>'{ambitos,unidad_ref}')),'UTF8')),'hex');
 IF d->>'principal_id' IS DISTINCT FROM q->>'actor_ref' OR d->>'accion' IS DISTINCT FROM 'bolsa.respuesta_revision.resolver'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'respuesta_revision_bolsa'
 OR d->>'finalidad' IS DISTINCT FROM 'resolver_respuestas_bolsa_rrhh'
 OR d->>'recurso_ref' IS DISTINCT FROM q->>'participacion_ref'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM hc
 OR c->>'efecto_ref' IS DISTINCT FROM q->>'participacion_ref' THEN
  RAISE EXCEPTION 'B128: decisión no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM q->>'participacion_ref'
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM hc THEN
  RAISE EXCEPTION 'B128: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:confirmacion:'||(q->>'participacion_ref'),0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.confirmacion_aceptacion_directa
  WHERE clave_idempotencia=q->>'clave_idempotencia';
 IF FOUND THEN
  IF previo.participacion_ref IS DISTINCT FROM q->>'participacion_ref' OR previo.bolsa_ref IS DISTINCT FROM q->>'bolsa_ref'
  OR previo.llamamiento_ref IS DISTINCT FROM q->>'llamamiento_ref' OR previo.actor_ref IS DISTINCT FROM q->>'actor_ref'
  OR previo.canon_sha256 IS DISTINCT FROM h OR previo.canon_original IS DISTINCT FROM p_canon
  OR previo.ambito_sha256 IS DISTINCT FROM ha OR previo.situacion_desde_esperada IS DISTINCT FROM esperado
  OR previo.evento_sha256 IS DISTINCT FROM encode(sha256(previo.evento_canonico),'hex')
  OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion z
   WHERE z.participacion_ref=previo.participacion_ref AND z.desde=previo.situacion_desde_nueva
    AND z.recibo_ref=previo.recibo_ref AND z.situacion='pendiente_incorporacion') THEN
   RAISE EXCEPTION 'B128: clave reutilizada' USING ERRCODE='VBS01'; END IF;
  RETURN QUERY SELECT jsonb_build_object('bolsa_ref',previo.bolsa_ref,'participacion_ref',previo.participacion_ref,
   'llamamiento_ref',previo.llamamiento_ref,'situacion_desde_esperada',to_char(previo.situacion_desde_esperada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'motivo',previo.motivo,'recibo_ref',previo.recibo_ref,'situacion','pendiente_incorporacion',
   'desde',to_char(previo.situacion_desde_nueva AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'fecha_disponible',NULL,'resuelta_en',to_char(previo.resuelta_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'respuesta_ref',previo.respuesta_ref,'resolucion_ref',previo.resolucion_ref,'canon_sha256',previo.canon_sha256),true;
  RETURN;
 END IF;
 SELECT rp.* INTO r FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento rp
  WHERE rp.llamamiento_ref=q->>'llamamiento_ref' AND rp.participacion_ref=q->>'participacion_ref' FOR SHARE;
 IF NOT FOUND OR r.bolsa_ref IS DISTINCT FROM q->>'bolsa_ref' OR r.respuesta IS DISTINCT FROM 'acepta'
 OR r.respondida_en>ahora
 OR r.causa IS NOT NULL OR r.justificante_ref IS NOT NULL OR r.justificante_sha256 IS NOT NULL
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_aceptacion_directa y WHERE y.respuesta_ref=r.respuesta_ref) THEN
  RAISE EXCEPTION 'B128: respuesta incompatible o resuelta' USING ERRCODE='VBS01'; END IF;
 SELECT z.* INTO l FROM vec_bolsa_llamamientos.llamamiento_emitido z WHERE z.llamamiento_ref=r.llamamiento_ref FOR SHARE;
 IF NOT FOUND OR l.bolsa_ref IS DISTINCT FROM r.bolsa_ref OR NOT(l.participaciones ? r.participacion_ref)
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion c0
  JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
  WHERE c0.bolsa_ref=r.bolsa_ref AND e.participacion_ref=r.participacion_ref) THEN
  RAISE EXCEPTION 'B128: llamamiento o bolsa ajenos' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:situacion:'||r.participacion_ref,0));
 SELECT z.* INTO s FROM vec_bolsa_llamamientos.situacion_participacion z
  WHERE z.participacion_ref=r.participacion_ref ORDER BY z.desde DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND OR s.desde IS DISTINCT FROM esperado OR s.desde>ahora
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion z
  WHERE z.participacion_ref=r.participacion_ref AND z.registrada_en>=r.respondida_en AND z.desde<=ahora) THEN
  RAISE EXCEPTION 'B128: situación cambiada' USING ERRCODE='VBS02'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion',0));
 SELECT p.*,g.ambito_sha256 INTO politica FROM
  (SELECT * FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1) p
 JOIN vec_bolsa_llamamientos.politica_transiciones_gobierno g USING(version);
 IF NOT FOUND OR politica.ambito_sha256 IS DISTINCT FROM ha
 OR NOT((s.situacion||'>pendiente_incorporacion')=ANY(politica.transiciones)) THEN
  RAISE EXCEPTION 'B128: política cerrada o ámbito ajeno' USING ERRCODE='55000'; END IF;
 IF ahora<=s.desde THEN ahora:=s.desde+interval '1 microsecond'; END IF;
 refhash:=encode(sha256(convert_to(r.respuesta_ref||chr(31)||(q->>'clave_idempotencia'),'UTF8')),'hex');
 recibo_ref:='recibo:respuesta-revision:'||encode(sha256(convert_to(r.participacion_ref||chr(31)||(q->>'clave_idempotencia'),'UTF8')),'hex');
 resolucion_ref:='resolucion:respuesta-revision:'||refhash; evento_ref:='evento:respuesta-revision:'||refhash;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion
  (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version)
 VALUES(r.participacion_ref,'pendiente_incorporacion',ahora,NULL,NULL,q->>'motivo',q->>'actor_ref',ahora,
  'confirmacion:'||refhash,recibo_ref,politica.version);
 evento:=convert_to(jsonb_build_object('esquema','vec.bolsa.respuesta-revision.confirmada.v1',
  'evento_ref',evento_ref,'respuesta_ref',r.respuesta_ref,'resolucion_ref',resolucion_ref,
  'participacion_ref',r.participacion_ref,'situacion','pendiente_incorporacion','recibo_ref',recibo_ref)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.confirmacion_aceptacion_directa
  (respuesta_ref,resolucion_ref,bolsa_ref,participacion_ref,llamamiento_ref,actor_ref,motivo,clave_idempotencia,
   canon_original,canon_sha256,ambito_sha256,situacion_desde_esperada,situacion_desde_nueva,politica_version,
   recibo_ref,decision_ref,auditoria_ref,evento_ref,evento_canonico,evento_sha256,resuelta_en)
 VALUES(r.respuesta_ref,resolucion_ref,r.bolsa_ref,r.participacion_ref,r.llamamiento_ref,q->>'actor_ref',q->>'motivo',
  q->>'clave_idempotencia',p_canon,h,ha,esperado,ahora,politica.version,recibo_ref,consumo.decision_ref,
  consumo.auditoria_ref,evento_ref,evento,encode(sha256(evento),'hex'),ahora);
 RETURN QUERY SELECT jsonb_build_object('bolsa_ref',r.bolsa_ref,'participacion_ref',r.participacion_ref,
  'llamamiento_ref',r.llamamiento_ref,'situacion_desde_esperada',to_char(esperado AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'motivo',q->>'motivo','recibo_ref',recibo_ref,'situacion','pendiente_incorporacion',
  'desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'fecha_disponible',NULL,'resuelta_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'respuesta_ref',r.respuesta_ref,'resolucion_ref',resolucion_ref,'canon_sha256',h),false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
DO $post$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_bolsa_llamamientos.confirmacion_aceptacion_directa'::regclass
  AND relrowsecurity AND relforcerowsecurity)
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
  AND 'search_path=pg_catalog, pg_temp'=ANY(p.proconfig))
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
  WHERE p.oid='vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND a.grantee=0)
 OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE') THEN
  RAISE EXCEPTION 'B128: RLS o ACL incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
