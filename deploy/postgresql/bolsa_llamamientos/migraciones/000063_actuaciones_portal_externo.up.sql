\set ON_ERROR_STOP on
-- B63: actuaciones exteriores propias. No mueve historia ni cambia orden/reglas.
-- Despliegue funcional requiere B64 para consumidores y bloqueo de efectos RRHH.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000063',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.solicitud_portal_candidato_externa') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'B63: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE TABLE vec_bolsa_llamamientos.solicitud_portal_candidato_externa (LIKE vec_bolsa_llamamientos.solicitud_portal_candidato INCLUDING ALL);
CREATE TABLE vec_bolsa_llamamientos.respuesta_portal_llamamiento_externa (LIKE vec_bolsa_llamamientos.respuesta_portal_llamamiento INCLUDING ALL);
CREATE TABLE vec_bolsa_llamamientos.disposicion_oferta_externa (LIKE vec_bolsa_llamamientos.disposicion_oferta INCLUDING ALL);
CREATE TABLE vec_bolsa_llamamientos.disposicion_oferta_candidato_externa (LIKE vec_bolsa_llamamientos.disposicion_oferta_candidato INCLUDING ALL);
CREATE TABLE vec_bolsa_llamamientos.confirmacion_contacto_participacion_externa (LIKE vec_bolsa_llamamientos.confirmacion_contacto_participacion INCLUDING ALL);

ALTER TABLE vec_bolsa_llamamientos.respuesta_portal_llamamiento_externa ADD CONSTRAINT respuesta_externa_llamamiento_fk FOREIGN KEY(llamamiento_ref) REFERENCES vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref);
ALTER TABLE vec_bolsa_llamamientos.disposicion_oferta_externa ADD CONSTRAINT disposicion_externa_oferta_fk FOREIGN KEY(oferta_ref) REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref);
ALTER TABLE vec_bolsa_llamamientos.disposicion_oferta_candidato_externa ADD CONSTRAINT disposicion_candidato_externa_fk FOREIGN KEY(oferta_ref,participacion_ref) REFERENCES vec_bolsa_llamamientos.disposicion_oferta_externa(oferta_ref,participacion_ref);
ALTER TABLE vec_bolsa_llamamientos.confirmacion_contacto_participacion_externa ADD CONSTRAINT confirmacion_externa_contacto_fk FOREIGN KEY(participacion_ref,version) REFERENCES vec_bolsa_llamamientos.datos_contacto_participacion(participacion_ref,version);
-- Proyección gobernada de un vínculo existente; no decide quién es persona.
CREATE TABLE vec_bolsa_llamamientos.contexto_participacion_externa_identidad(
 participacion_ref text PRIMARY KEY,bolsa_ref text NOT NULL,candidato_ref text NOT NULL,
 persona_ref text NOT NULL,perfil_ref text NOT NULL,contexto_actor_ref text NOT NULL,
 UNIQUE(participacion_ref,bolsa_ref,candidato_ref,persona_ref,perfil_ref,contexto_actor_ref)
);
CREATE TABLE vec_bolsa_llamamientos.contexto_participacion_externa_versiones(
 participacion_ref text NOT NULL,version bigint NOT NULL CHECK(version>=1),
 bolsa_ref text NOT NULL,candidato_ref text NOT NULL,persona_ref text NOT NULL,perfil_ref text NOT NULL,contexto_actor_ref text NOT NULL,
 documento jsonb NOT NULL,huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(participacion_ref,version),
 FOREIGN KEY(participacion_ref,bolsa_ref,candidato_ref,persona_ref,perfil_ref,contexto_actor_ref) REFERENCES vec_bolsa_llamamientos.contexto_participacion_externa_identidad(participacion_ref,bolsa_ref,candidato_ref,persona_ref,perfil_ref,contexto_actor_ref)
);
CREATE TABLE vec_bolsa_llamamientos.contexto_participacion_externa_actual(
 participacion_ref text PRIMARY KEY,version bigint NOT NULL,
 FOREIGN KEY(participacion_ref,version) REFERENCES vec_bolsa_llamamientos.contexto_participacion_externa_versiones(participacion_ref,version)
);
CREATE TABLE vec_bolsa_llamamientos.control_contexto_participacion_externa_v1(
 control_id boolean PRIMARY KEY CHECK(control_id),generacion bigint NOT NULL CHECK(generacion>=0)
);
INSERT INTO vec_bolsa_llamamientos.control_contexto_participacion_externa_v1 VALUES(true,0);
DO $proteccion$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['solicitud_portal_candidato_externa','respuesta_portal_llamamiento_externa','disposicion_oferta_externa','disposicion_oferta_candidato_externa','confirmacion_contacto_participacion_externa','contexto_participacion_externa_identidad','contexto_participacion_externa_versiones','contexto_participacion_externa_actual','control_contexto_participacion_externa_v1'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_bolsa_llamamientos.%I FOR ALL TO vec_bolsa_llamamientos_propietario USING(current_user=''vec_bolsa_llamamientos_propietario'') WITH CHECK(current_user=''vec_bolsa_llamamientos_propietario'')',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_bolsa_llamamientos.%I FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t);
  IF t NOT IN('contexto_participacion_externa_actual','control_contexto_participacion_externa_v1') THEN
   EXECUTE pg_catalog.format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t);
  END IF;
 END LOOP;
END $proteccion$;
CREATE FUNCTION vec_bolsa_llamamientos.serializar_contexto_participacion_externa_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:contexto_externo:v1',0));
 UPDATE vec_bolsa_llamamientos.control_contexto_participacion_externa_v1 SET generacion=generacion+1 WHERE control_id=true;
 IF NOT FOUND THEN RAISE EXCEPTION 'generación exterior ausente' USING ERRCODE='55000'; END IF;
 RETURN NULL;
END $f$;
CREATE TRIGGER serializar BEFORE INSERT OR UPDATE OR DELETE ON vec_bolsa_llamamientos.contexto_participacion_externa_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.serializar_contexto_participacion_externa_v1();
CREATE FUNCTION vec_bolsa_llamamientos.huella_contexto_participacion_externa_v1(p_documento jsonb,p_version bigint)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(p_documento,p_version)::text,'UTF8')),'hex')
$f$;
ALTER TABLE vec_bolsa_llamamientos.contexto_participacion_externa_versiones
 ADD CONSTRAINT contexto_exterior_documento_fijo CHECK(
 documento->>'participacion_ref'=participacion_ref AND documento->>'bolsa_ref'=bolsa_ref
 AND documento->>'candidato_ref'=candidato_ref AND documento->>'persona_ref'=persona_ref
 AND documento->>'perfil_ref'=perfil_ref AND documento->>'contexto_actor_ref'=contexto_actor_ref
 AND huella_sha256=vec_bolsa_llamamientos.huella_contexto_participacion_externa_v1(documento,version));
CREATE FUNCTION vec_bolsa_llamamientos.preimagen_contexto_participacion_externa_v1(p_participacion_ref text)
RETURNS TABLE(version bigint,huella_sha256 text) LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT v.version,v.huella_sha256 FROM vec_bolsa_llamamientos.contexto_participacion_externa_actual a JOIN vec_bolsa_llamamientos.contexto_participacion_externa_versiones v USING(participacion_ref,version) WHERE a.participacion_ref=p_participacion_ref
$f$;
CREATE FUNCTION vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1(p_documento jsonb,p_version_esperada bigint,p_huella_esperada text,p_huella_aprobada text)
RETURNS TABLE(participacion_ref text,version bigint,huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE d jsonb:=p_documento; anterior record; nueva bigint; h text; ahora timestamptz; k text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(d::text)>16384
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>12
    OR NOT d ?& ARRAY['participacion_ref','bolsa_ref','candidato_ref','persona_ref','perfil_ref','contexto_actor_ref','estado','vigente_desde','vigente_hasta','fuente_ref','fuente_version','fuente_huella_sha256']
    OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 9223372036854775806
    OR p_huella_aprobada IS NULL OR p_huella_aprobada !~ '^[0-9a-f]{64}$'
    OR d->>'candidato_ref' IS NULL OR d->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR d->>'persona_ref' IS NULL OR d->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR d->>'perfil_ref' IS NULL OR d->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR d->>'contexto_actor_ref' IS NULL OR d->>'contexto_actor_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
    OR d->>'fuente_ref' IS NULL OR d->>'fuente_ref' !~ '^prc_[A-Za-z0-9_-]{22,128}$'
    OR pg_catalog.jsonb_typeof(d->'fuente_version') IS DISTINCT FROM 'number'
    OR pg_catalog.scale((d->>'fuente_version')::numeric)<>0
    OR (d->>'fuente_version')::numeric NOT BETWEEN 1 AND 18446744073709551615::numeric
    OR d->>'fuente_huella_sha256' IS NULL OR d->>'fuente_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR (d->>'estado' IN('activo','revocado')) IS NOT TRUE THEN
  RAISE EXCEPTION 'contexto de participación exterior inválido' USING ERRCODE='22023'; END IF;
 FOREACH k IN ARRAY ARRAY['vigente_desde','vigente_hasta'] LOOP
  IF d->>k IS NULL OR d->>k !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$'
     OR (d->>k)::timestamptz IS NULL OR NOT pg_catalog.isfinite((d->>k)::timestamptz)
     OR pg_catalog.to_char((d->>k)::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM d->>k THEN
   RAISE EXCEPTION 'vigencia exterior inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF (d->>'vigente_hasta')::timestamptz<=(d->>'vigente_desde')::timestamptz THEN RAISE EXCEPTION 'vigencia exterior inválida' USING ERRCODE='22023'; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(d->>'candidato_ref') p WHERE p.participacion_ref=d->>'participacion_ref' AND p.bolsa_ref=d->>'bolsa_ref')<>1 THEN
  RAISE EXCEPTION 'vínculo exterior no acreditado por Bolsa' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:contexto_externo:v1',0));
 SELECT v.* INTO anterior FROM vec_bolsa_llamamientos.contexto_participacion_externa_actual a JOIN vec_bolsa_llamamientos.contexto_participacion_externa_versiones v USING(participacion_ref,version) WHERE a.participacion_ref=d->>'participacion_ref' FOR UPDATE OF a;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL)) OR (FOUND AND (anterior.version IS DISTINCT FROM p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada)) THEN
  RAISE EXCEPTION 'contexto exterior CAS divergente' USING ERRCODE='40001'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF d->>'estado'='activo' AND (ahora<(d->>'vigente_desde')::timestamptz OR ahora>=(d->>'vigente_hasta')::timestamptz
       OR (p_version_esperada>0 AND (anterior.documento->>'estado'<>'activo' OR ahora>=(anterior.documento->>'vigente_hasta')::timestamptz))) THEN
  RAISE EXCEPTION 'contexto exterior no reactivable' USING ERRCODE='55000'; END IF;
 nueva:=p_version_esperada+1; h:=vec_bolsa_llamamientos.huella_contexto_participacion_externa_v1(d,nueva);
 IF h IS DISTINCT FROM p_huella_aprobada THEN RAISE EXCEPTION 'huella exterior no aprobada' USING ERRCODE='42501'; END IF;
 IF p_version_esperada=0 THEN INSERT INTO vec_bolsa_llamamientos.contexto_participacion_externa_identidad VALUES(d->>'participacion_ref',d->>'bolsa_ref',d->>'candidato_ref',d->>'persona_ref',d->>'perfil_ref',d->>'contexto_actor_ref'); END IF;
 INSERT INTO vec_bolsa_llamamientos.contexto_participacion_externa_versiones VALUES(d->>'participacion_ref',nueva,d->>'bolsa_ref',d->>'candidato_ref',d->>'persona_ref',d->>'perfil_ref',d->>'contexto_actor_ref',d,h);
 INSERT INTO vec_bolsa_llamamientos.contexto_participacion_externa_actual VALUES(d->>'participacion_ref',nueva) ON CONFLICT ON CONSTRAINT contexto_participacion_externa_actual_pkey DO UPDATE SET version=excluded.version;
 RETURN QUERY SELECT d->>'participacion_ref',nueva,h;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato text,p_participacion text,p_contexto bytea)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v record; x jsonb; ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'contexto exterior requiere serializable' USING ERRCODE='25000'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_bolsa_llamamientos:contexto_externo:v1',0));
 PERFORM 1 FROM vec_bolsa_llamamientos.control_contexto_participacion_externa_v1 WHERE control_id=true FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'generación exterior ausente' USING ERRCODE='42501'; END IF;
 SELECT z.* INTO v FROM vec_bolsa_llamamientos.contexto_participacion_externa_actual a JOIN vec_bolsa_llamamientos.contexto_participacion_externa_versiones z USING(participacion_ref,version) WHERE a.participacion_ref=p_participacion;
 IF NOT FOUND THEN RAISE EXCEPTION 'contexto exterior pendiente de provisión' USING ERRCODE='42501'; END IF;
 x:=pg_catalog.convert_from(p_contexto,'UTF8')::jsonb; ahora:=pg_catalog.clock_timestamp();
 IF v.candidato_ref IS DISTINCT FROM p_candidato OR v.documento->>'estado'<>'activo'
    OR ahora<(v.documento->>'vigente_desde')::timestamptz OR ahora>=(v.documento->>'vigente_hasta')::timestamptz
    OR v.persona_ref IS DISTINCT FROM x->>'persona_ref' OR v.perfil_ref IS DISTINCT FROM x->>'perfil_activo_ref'
    OR v.contexto_actor_ref IS DISTINCT FROM x->>'contexto_actor_ref' THEN RAISE EXCEPTION 'contexto exterior no coincide con el actor' USING ERRCODE='42501'; END IF;
END $f$;
CREATE VIEW vec_bolsa_llamamientos.solicitud_portal_candidato_lectura_portal_externa_v1 WITH(security_barrier=true,security_invoker=true) AS SELECT * FROM vec_bolsa_llamamientos.solicitud_portal_candidato UNION ALL SELECT * FROM vec_bolsa_llamamientos.solicitud_portal_candidato_externa;
REVOKE ALL ON TABLE vec_bolsa_llamamientos.solicitud_portal_candidato_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.solicitud_portal_candidato_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
CREATE VIEW vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 WITH(security_barrier=true,security_invoker=true) AS SELECT * FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento UNION ALL SELECT * FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento_externa;
REVOKE ALL ON TABLE vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
CREATE VIEW vec_bolsa_llamamientos.disposicion_oferta_lectura_portal_externa_v1 WITH(security_barrier=true,security_invoker=true) AS SELECT * FROM vec_bolsa_llamamientos.disposicion_oferta UNION ALL SELECT * FROM vec_bolsa_llamamientos.disposicion_oferta_externa;
REVOKE ALL ON TABLE vec_bolsa_llamamientos.disposicion_oferta_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.disposicion_oferta_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
CREATE VIEW vec_bolsa_llamamientos.disposicion_oferta_candidato_lectura_portal_externa_v1 WITH(security_barrier=true,security_invoker=true) AS SELECT * FROM vec_bolsa_llamamientos.disposicion_oferta_candidato UNION ALL SELECT * FROM vec_bolsa_llamamientos.disposicion_oferta_candidato_externa;
REVOKE ALL ON TABLE vec_bolsa_llamamientos.disposicion_oferta_candidato_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.disposicion_oferta_candidato_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
CREATE VIEW vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 WITH(security_barrier=true,security_invoker=true) AS SELECT * FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion UNION ALL SELECT * FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion_externa;
REVOKE ALL ON TABLE vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;

-- Las claves agregadas son las mismas del carril histórico. Se comprueba
-- la otra historia bajo ese mismo cerrojo, también ante inserción interna.
CREATE FUNCTION vec_bolsa_llamamientos.rechazar_colision_actuacion_externa_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_TABLE_NAME IN('solicitud_portal_candidato_externa','respuesta_portal_llamamiento_externa') THEN
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:portal:'||NEW.participacion_ref,0));
 ELSIF TG_TABLE_NAME='confirmacion_contacto_participacion_externa' THEN
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:datos_contacto:'||NEW.participacion_ref,0));
 ELSE
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:'||NEW.oferta_ref,0));
 END IF;
 IF TG_TABLE_NAME='solicitud_portal_candidato_externa' THEN
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.solicitud_portal_candidato s WHERE s.solicitud_ref=NEW.solicitud_ref OR s.recibo_ref=NEW.recibo_ref OR s.decision_ref=NEW.decision_ref OR (s.participacion_ref=NEW.participacion_ref AND s.clave_idempotencia=NEW.clave_idempotencia)) THEN RAISE EXCEPTION 'clave de actuación exterior ya histórica' USING ERRCODE='23505'; END IF;
 ELSIF TG_TABLE_NAME='respuesta_portal_llamamiento_externa' THEN
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r WHERE r.respuesta_ref=NEW.respuesta_ref OR r.recibo_ref=NEW.recibo_ref OR r.decision_ref=NEW.decision_ref OR (r.participacion_ref=NEW.participacion_ref AND (r.clave_idempotencia=NEW.clave_idempotencia OR r.llamamiento_ref=NEW.llamamiento_ref))) THEN RAISE EXCEPTION 'clave de actuación exterior ya histórica' USING ERRCODE='23505'; END IF;
 ELSIF TG_TABLE_NAME='confirmacion_contacto_participacion_externa' THEN
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion c WHERE c.recibo_ref=NEW.recibo_ref OR c.decision_ref=NEW.decision_ref OR (c.participacion_ref=NEW.participacion_ref AND (c.version=NEW.version OR c.clave_idempotencia=NEW.clave_idempotencia))) THEN RAISE EXCEPTION 'clave de actuación exterior ya histórica' USING ERRCODE='23505'; END IF;
 ELSIF TG_TABLE_NAME='disposicion_oferta_externa' THEN
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.disposicion_oferta d WHERE d.recibo_ref=NEW.recibo_ref OR (d.oferta_ref=NEW.oferta_ref AND d.participacion_ref=NEW.participacion_ref)) THEN RAISE EXCEPTION 'clave de actuación exterior ya histórica' USING ERRCODE='23505'; END IF;
 ELSIF TG_TABLE_NAME='disposicion_oferta_candidato_externa' THEN
  IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.disposicion_oferta_candidato d WHERE d.decision_ref=NEW.decision_ref OR (d.oferta_ref=NEW.oferta_ref AND d.participacion_ref=NEW.participacion_ref)) THEN RAISE EXCEPTION 'clave de actuación exterior ya histórica' USING ERRCODE='23505'; END IF;
 ELSE RAISE EXCEPTION 'tabla exterior desconocida' USING ERRCODE='55000';
 END IF;
 RETURN NEW;
END $f$;
DO $colisiones$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['solicitud_portal_candidato_externa','respuesta_portal_llamamiento_externa','disposicion_oferta_externa','disposicion_oferta_candidato_externa','confirmacion_contacto_participacion_externa'] LOOP
  EXECUTE pg_catalog.format('CREATE TRIGGER colision_historica BEFORE INSERT ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.rechazar_colision_actuacion_externa_v1()',t);
 END LOOP;
END $colisiones$;
CREATE FUNCTION vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF session_user<>'vec_externo_bolsa_desarrollo' OR current_setting('role')<>'none'
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles l WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls AND l.rolconfig IS NULL)
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles l WHERE l.rolname='vec_bolsa_llamamientos_portal_externo' AND NOT l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls AND l.rolconfig IS NULL)
    OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole)<>1
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole AND roleid='vec_bolsa_llamamientos_portal_externo'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>session_user::regrole AND r.oid<>'vec_bolsa_llamamientos_portal_externo'::regrole AND pg_catalog.pg_has_role(session_user::regrole,r.oid,'MEMBER'))
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>'vec_bolsa_llamamientos_portal_externo'::regrole AND pg_catalog.pg_has_role('vec_bolsa_llamamientos_portal_externo'::regrole,r.oid,'MEMBER'))
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(session_user::regrole,'vec_bolsa_llamamientos_portal_externo'::regrole)) THEN
  RAISE EXCEPTION 'runtime exterior de Bolsa no acreditado' USING ERRCODE='42501'; END IF;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.anotar_consumo_candidato_externo_v1(p_ambito text,p_candidato_ref text,p_dato text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE firma text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 IF (p_ambito IN('consulta','responder')) IS NOT TRUE OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_dato IS NULL OR pg_catalog.octet_length(p_dato) NOT BETWEEN 1 AND 512 OR pg_catalog.strpos(p_dato,pg_catalog.chr(31))<>0 THEN RAISE EXCEPTION 'marca exterior inválida' USING ERRCODE='22023'; END IF;
 firma:=vec_bolsa_llamamientos.firma_marca_consumo_v1(pg_catalog.pg_current_xact_id(),'portal_externo:'||p_ambito,p_candidato_ref,session_user||pg_catalog.chr(31)||p_dato);
 IF firma IS NULL THEN RAISE EXCEPTION 'firma de marca exterior ausente' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.set_config('vec_bolsa_llamamientos.marca_consumo_portal_externo',pg_catalog.concat_ws(pg_catalog.chr(31),p_ambito,p_candidato_ref,p_dato,firma),true);
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.exigir_consumo_candidato_externo_v1(p_ambitos text[],p_candidato_ref text)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE partes text[]; xid pg_catalog.xid8; esperado text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 partes:=pg_catalog.string_to_array(pg_catalog.current_setting('vec_bolsa_llamamientos.marca_consumo_portal_externo',true),pg_catalog.chr(31));
 xid:=pg_catalog.pg_current_xact_id_if_assigned();
 IF partes IS NULL OR pg_catalog.cardinality(partes)<>4 OR xid IS NULL OR partes[2] IS DISTINCT FROM p_candidato_ref OR (partes[1]=ANY(p_ambitos)) IS NOT TRUE THEN
  RAISE EXCEPTION 'lectura exterior sin consumo propio' USING ERRCODE='42501'; END IF;
 esperado:=vec_bolsa_llamamientos.firma_marca_consumo_v1(xid,'portal_externo:'||partes[1],p_candidato_ref,session_user||pg_catalog.chr(31)||partes[3]);
 IF esperado IS NULL OR esperado IS DISTINCT FROM partes[4] THEN RAISE EXCEPTION 'marca exterior no acreditada' USING ERRCODE='42501'; END IF;
 RETURN partes[3];
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato text)
RETURNS void LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.solicitud_portal_candidato_lectura_portal_externa_v1 s WHERE s.candidato_ref=p_candidato GROUP BY s.participacion_ref,s.clave_idempotencia HAVING count(*)>1)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 s WHERE s.candidato_ref=p_candidato GROUP BY s.participacion_ref,s.clave_idempotencia HAVING count(*)>1)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 s WHERE s.candidato_ref=p_candidato GROUP BY s.participacion_ref,s.llamamiento_ref HAVING count(*)>1)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 s WHERE s.candidato_ref=p_candidato GROUP BY s.participacion_ref,s.clave_idempotencia HAVING count(*)>1)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 s WHERE s.candidato_ref=p_candidato GROUP BY s.participacion_ref,s.version HAVING count(*)>1)
    OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.disposicion_oferta_lectura_portal_externa_v1 s JOIN vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato) p USING(participacion_ref) GROUP BY s.oferta_ref,s.participacion_ref HAVING count(*)>1) THEN
  RAISE EXCEPTION 'historia personal exterior ambigua' USING ERRCODE='23505'; END IF;
END $f$;

-- Fuente mínima privada; B62 ampliará esta fuente, sin cambiar las fachadas.
CREATE FUNCTION vec_bolsa_llamamientos.ultimo_aviso_portal_externo_v1(p_participacion_ref text,p_bolsa_ref text,p_corte timestamptz)
RETURNS TABLE(llamamiento_ref text,emitido_en timestamptz,resultado text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $aviso$
 SELECT l.llamamiento_ref,l.emitido_en,contacto.resultado
     FROM vec_bolsa_llamamientos.llamamiento_emitido l
     JOIN vec_bolsa_llamamientos.contacto_participacion contacto
       ON contacto.llamamiento_ref=l.llamamiento_ref
      AND contacto.bolsa_ref=l.bolsa_ref
      AND contacto.participacion_ref=p_participacion_ref
      AND contacto.canal='correo'
      AND contacto.resultado IN('enviado','no_enviado')
      AND contacto.instante<=p_corte
    WHERE l.bolsa_ref=p_bolsa_ref
      AND l.participaciones ? p_participacion_ref
      AND l.emitido_en<=p_corte
    ORDER BY l.emitido_en DESC,l.llamamiento_ref DESC,contacto.instante DESC,contacto.contacto_ref DESC
    LIMIT 1
$aviso$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.ultimo_aviso_portal_externo_v1(text,text,timestamptz) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
CREATE FUNCTION vec_bolsa_llamamientos.entradas_llamamiento_portal_externo_v1(p_candidato_ref text,p_corte timestamptz)
RETURNS TABLE(ocurrido_en timestamptz,orden_interno text,item jsonb)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $aviso$
 WITH propias AS MATERIALIZED (
  SELECT participacion_ref,bolsa_ref,categoria_ref FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref)
 )
  SELECT l.emitido_en, 'llamamiento:'||l.llamamiento_ref||':'||ct.contacto_ref,
   jsonb_build_object('clase','llamamiento','bolsa',p.bolsa_ref,
    'categoria',p.categoria_ref,'ocurrido_en',to_char(l.emitido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'canal','correo','resultado',ct.resultado)
   FROM propias p
   JOIN vec_bolsa_llamamientos.llamamiento_emitido l ON l.bolsa_ref=p.bolsa_ref
   JOIN LATERAL jsonb_array_elements_text(l.participaciones) WITH ORDINALITY lp(ref,ordinal) ON lp.ref=p.participacion_ref
   JOIN vec_bolsa_llamamientos.contacto_participacion ct
    ON ct.llamamiento_ref=l.llamamiento_ref AND ct.bolsa_ref=l.bolsa_ref
    AND ct.participacion_ref=p.participacion_ref AND ct.canal='correo'
    AND ct.resultado IN ('enviado','no_enviado')
    AND ct.instante=l.emitido_en
    AND ct.clave_idempotencia=l.clave_idempotencia||':correo:'||lp.ordinal
    AND ct.recibo_ref='recibo:contacto:'||encode(sha256(convert_to(
      l.bolsa_ref||chr(31)||l.clave_idempotencia||chr(31)||p.participacion_ref,'UTF8')),'hex')
   WHERE l.emitido_en<=p_corte AND ct.instante<=p_corte
$aviso$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.entradas_llamamiento_portal_externo_v1(text,timestamptz) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','TimeZone=UTC','lock_timeout=2s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '2e464dc67485d466b2fe964bbfa8cefd868cee4187cfbb820a1756e7ec68f497' THEN
  RAISE EXCEPTION 'B63: fuente confirmar_contacto_propio_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.confirmar_contacto_propio_externo_v1(p_candidato_ref text, p_bolsa_ref text, p_version bigint, p_clave text, p_recibo_ref text, p_confirmada_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, confirmada_en timestamp with time zone) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET TimeZone='UTC' SET lock_timeout='2s' AS $b63$
DECLARE c jsonb; d jsonb; x jsonb; n integer; v_participacion text; v_consumo record;
 v_previa vec_bolsa_llamamientos.confirmacion_contacto_participacion_externa%ROWTYPE;
 v_accion constant text := 'bolsa.participaciones_propias.confirmar_contacto';
 v_ahora timestamptz := clock_timestamp();
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 512 OR p_bolsa_ref <> btrim(p_bolsa_ref)
    OR p_version IS NULL OR p_version < 1
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:confirmacion-contacto:[0-9a-f]{64}$'
    OR p_confirmada_en IS NULL OR p_confirmada_en > v_ahora + interval '1 minute' OR p_confirmada_en < v_ahora - interval '5 minutes' THEN
  RAISE EXCEPTION 'confirmación de contacto inválida' USING ERRCODE='22023';
 END IF;
 BEGIN c := convert_from(p_capacidad,'UTF8')::jsonb; d := convert_from(p_decision,'UTF8')::jsonb; x := convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de confirmación inválido' USING ERRCODE='22023'; END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref OR d->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'operacion' IS DISTINCT FROM v_accion OR d->>'accion' IS DISTINCT FROM v_accion
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'confirmación de contacto denegada' USING ERRCODE='42501';
 END IF;
 SELECT count(*) INTO n FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF n <> 1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
                           WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
  RAISE EXCEPTION 'confirmación de contacto denegada' USING ERRCODE='42501';
 END IF;
 v_participacion := vec_bolsa_llamamientos.participacion_contacto_candidato_v1(p_candidato_ref, p_bolsa_ref);
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 -- Como B2 y el portal (000030), la decisión viva se consume antes de
 -- resolver el replay: un reintento no devuelve el recibo sin una
 -- autorización nueva y verificada.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
  RAISE EXCEPTION 'confirmación de contacto denegada' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:datos_contacto:' || v_participacion, 0));
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 cp
  WHERE cp.participacion_ref = v_participacion AND cp.clave_idempotencia = p_clave;
 IF FOUND THEN
  IF v_previa.version <> p_version OR v_previa.candidato_ref <> p_candidato_ref OR v_previa.bolsa_ref IS DISTINCT FROM p_bolsa_ref THEN
   RAISE EXCEPTION 'clave reutilizada con otra confirmación' USING ERRCODE='VBC01';
  END IF;
  RETURN QUERY SELECT true, v_previa.recibo_ref, v_previa.version, v_previa.confirmada_en;
  RETURN;
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_confirmacion_contacto_externa_interna_v1(
  p_candidato_ref, p_bolsa_ref, v_participacion, p_version, p_clave, p_recibo_ref, p_confirmada_en, v_consumo.decision_ref);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamp with time zone,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s','statement_timeout=15s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '86e484771e7ebe1e9beb8643ac96dd73e0b62c6ee4782dc65072a4f1cbf999c9' THEN
  RAISE EXCEPTION 'B63: fuente consultar_historial_mi_bolsa_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.consultar_historial_mi_bolsa_externo_v1(p_candidato_ref text, p_consultada_en timestamp with time zone, p_pagina integer, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET lock_timeout='2s' SET statement_timeout='15s' AS $b63$
DECLARE c jsonb; d jsonb; x jsonb; v_n integer; v_consumo record;
 v_items jsonb; v_hay_mas boolean;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_consultada_en IS NULL OR NOT isfinite(p_consultada_en)
    OR p_pagina IS NULL OR p_pagina NOT BETWEEN 1 AND 10000
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'consulta de historial inválida' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de historial inválido' USING ERRCODE='22023'; END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.historial_propio.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.mi-bolsa.historial.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'consulta_historial_propio'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["contratos_propios","llamamientos_propios","renuncias_propias"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'consulta de historial denegada' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_n FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_n<>1 OR NOT EXISTS (
   SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
    WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref)
 THEN RAISE EXCEPTION 'consulta de historial denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo
  FROM vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'consulta de historial denegada' USING ERRCODE='42501'; END IF;

 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 WITH propias AS MATERIALIZED (
  SELECT participacion_ref,bolsa_ref,categoria_ref
   FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref)
 ), entradas AS (
  SELECT co.ocurrido_en, 'contrato:'||co.evento_ref AS orden_interno,
   jsonb_build_object('clase','contrato_bolsa','procedencia','evento_ct_recibido','bolsa',p.bolsa_ref,
    'categoria',p.categoria_ref,'ocurrido_en',to_char(co.ocurrido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'tipo',co.tipo,'inicio',co.inicio,'fin_previsto',co.fin_previsto,
    'modalidad_clave',co.modalidad_clave) AS item
   FROM propias p JOIN vec_bolsa_llamamientos.contrato_participacion co
    ON co.participacion_ref=p.participacion_ref AND co.bolsa_ref=p.bolsa_ref
   WHERE co.ocurrido_en<=p_consultada_en
  UNION ALL
  SELECT h.ocurrido_en,h.orden_interno,h.item
   FROM vec_bolsa_llamamientos.entradas_llamamiento_portal_externo_v1(p_candidato_ref,p_consultada_en) h
  UNION ALL
  SELECT r.respondida_en, 'renuncia:'||r.respuesta_ref,
   jsonb_build_object('clase','renuncia','bolsa',p.bolsa_ref,
    'categoria',p.categoria_ref,'ocurrido_en',to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'respuesta',r.respuesta,'modo',r.modo,
    'estado',CASE r.modo WHEN 'firme' THEN 'respuesta_registrada' ELSE 'propuesta_pendiente_rrhh' END)
   FROM propias p JOIN vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 r
    ON r.participacion_ref=p.participacion_ref AND r.bolsa_ref=p.bolsa_ref
    AND r.candidato_ref=p_candidato_ref AND r.respuesta IN ('renuncia','renuncia_justificada')
   WHERE r.respondida_en<=p_consultada_en
 ), pagina AS (
  SELECT e.ocurrido_en,e.orden_interno,e.item FROM entradas e
   ORDER BY e.ocurrido_en DESC,e.orden_interno DESC
   LIMIT 21 OFFSET (p_pagina-1)*20
 ), numerada AS (
  SELECT row_number() OVER (ORDER BY ocurrido_en DESC,orden_interno DESC) AS n,
   ocurrido_en,orden_interno,item FROM pagina
 )
 SELECT count(*)>20,
   coalesce(jsonb_agg(item ORDER BY ocurrido_en DESC,orden_interno DESC) FILTER (WHERE n<=20),'[]'::jsonb)
 INTO v_hay_mas,v_items FROM numerada;
 RETURN jsonb_build_object('consultada_en',to_char(p_consultada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'pagina',p_pagina,'tamano',20,'hay_mas',v_hay_mas,'items',v_items);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'd626cad864b5758498e93ace3788774e205640565d21b1dd257933a04b7c01b1' THEN
  RAISE EXCEPTION 'B63: fuente consultar_mi_bolsa_portal_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_portal_externo_v1(p_candidato_ref text, p_consultada_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET lock_timeout='2s' AS $b63$
DECLARE v jsonb;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 v := vec_bolsa_llamamientos.consultar_mi_bolsa_externo_v1(p_candidato_ref, p_consultada_en, p_capacidad, p_decision, p_motivo, p_contexto,
   p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 PERFORM vec_bolsa_llamamientos.anotar_consumo_candidato_externo_v1('consulta', p_candidato_ref, encode(sha256(p_decision),'hex')||'mi-bolsa');
 RETURN v;
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s','statement_timeout=15s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'ded995d98d206749aae198b3e15d13c38624c6732b426ac24f8fdb07d717d12b' THEN
  RAISE EXCEPTION 'B63: fuente consultar_mi_bolsa_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_externo_v1(p_candidato_ref text, p_consultada_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET lock_timeout='2s' SET statement_timeout='15s' AS $b63$
DECLARE c jsonb; d jsonb; x jsonb; v_candidatos integer;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 IF p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$' OR p_consultada_en IS NULL
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'consulta Mi bolsa inválida' USING ERRCODE='22023';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN
   RAISE EXCEPTION 'consulta Mi bolsa inválida' USING ERRCODE='22023';
 END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR d->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
   RAISE EXCEPTION 'consulta Mi bolsa denegada' USING ERRCODE='42501';
 END IF;
 SELECT count(*) INTO v_candidatos FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_candidatos<>1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
   RAISE EXCEPTION 'consulta Mi bolsa denegada' USING ERRCODE='42501';
 END IF;
 PERFORM 1 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
    'bolsa',participacion.bolsa_ref,'categoria',participacion.categoria_ref,'version',participacion.version_bolsa,'orden_inicial',participacion.orden,'total_instantanea',participacion.total_participaciones,'estado_bolsa',participacion.estado,'vigente_desde',to_char(participacion.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_hasta',CASE WHEN participacion.vigente_hasta IS NULL THEN NULL ELSE to_char(participacion.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
    'situacion_actual',CASE WHEN situacion.participacion_ref IS NULL THEN NULL ELSE jsonb_build_object(
      'estado',CASE WHEN situacion.situacion IN ('disponible','trabajando','disponible_desde')
       AND plazo.disponible_en>p_consultada_en THEN 'disponible_desde'
       WHEN cese.trabajo_cesado OR situacion.situacion='disponible_desde' THEN 'disponible'
       ELSE situacion.situacion END,
      'desde',to_char((CASE
       WHEN cese.en_restriccion AND plazo.disponible_en>p_consultada_en
         AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
         AND (situacion.fecha_disponible IS NULL OR plazo.cese_disponible_en>situacion.fecha_disponible)
         THEN greatest(situacion.desde,cese.fecha_efecto::timestamp AT TIME ZONE 'Europe/Madrid')
       WHEN plazo.disponible_en<=p_consultada_en AND plazo.disponible_en IS NOT NULL
         AND (cese.trabajo_cesado OR situacion.situacion='disponible_desde')
         THEN greatest(situacion.desde,plazo.disponible_en)
       ELSE situacion.desde END)
       AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'hasta',CASE WHEN situacion.hasta IS NULL THEN NULL ELSE to_char(situacion.hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
      'fecha_disponible',CASE
       WHEN situacion.situacion IN ('disponible','trabajando','disponible_desde')
         AND plazo.disponible_en>p_consultada_en
         THEN to_char(plazo.disponible_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       ELSE NULL END
    ) END,
    'ultimo_llamamiento',CASE WHEN ultimo.llamamiento_ref IS NULL THEN NULL ELSE jsonb_build_object(
      'emitido_en',to_char(ultimo.emitido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
      'canal','correo','resultado',ultimo.resultado
    ) END
  ) ORDER BY participacion.confirmada_en DESC,participacion.categoria_ref),'[]'::jsonb) INTO x
 FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1(p_candidato_ref) participacion
 LEFT JOIN LATERAL (
   SELECT s.participacion_ref,s.situacion,s.desde,s.hasta,s.fecha_disponible
     FROM vec_bolsa_llamamientos.situacion_participacion s
    WHERE s.participacion_ref=participacion.participacion_ref AND s.desde<=p_consultada_en
    ORDER BY s.desde DESC LIMIT 1
 ) situacion ON true
 LEFT JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_v1(
   participacion.participacion_ref,p_consultada_en) cese ON true
 LEFT JOIN LATERAL (
   SELECT greatest(
     CASE WHEN situacion.situacion='disponible_desde' THEN situacion.fecha_disponible END,
     CASE WHEN (cese.en_restriccion OR cese.trabajo_cesado)
            AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN cese.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid' END
   ) AS disponible_en,
   cese.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid' AS cese_disponible_en
 ) plazo ON true
 LEFT JOIN LATERAL vec_bolsa_llamamientos.ultimo_aviso_portal_externo_v1(participacion.participacion_ref,participacion.bolsa_ref,p_consultada_en) ultimo ON true;
 RETURN jsonb_build_object('consultada_en',to_char(p_consultada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'participaciones',x);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.exigir_portal_candidato_v1(text,text,text,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'da597bb760b6e2ca54379f5e06e389a1abec8744b24de4c8025fbce3002d8f9a' THEN
  RAISE EXCEPTION 'B63: fuente exigir_portal_candidato_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.exigir_portal_candidato_externo_v1(p_candidato_ref text, p_bolsa_ref text, p_accion text, p_capacidad bytea, p_decision bytea, p_contexto bytea)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path='pg_catalog' AS $b63$
DECLARE c jsonb; d jsonb; x jsonb; n integer;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'portal del candidato no autorizado' USING ERRCODE='42501';
 END IF;
 BEGIN c := convert_from(p_capacidad,'UTF8')::jsonb; d := convert_from(p_decision,'UTF8')::jsonb; x := convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'portal del candidato inválido' USING ERRCODE='22023'; END;
 IF c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref OR d->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR c->>'operacion' IS DISTINCT FROM p_accion OR d->>'accion' IS DISTINCT FROM p_accion
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 SELECT count(*) INTO n FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF n <> 1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 RETURN vec_bolsa_llamamientos.participacion_portal_candidato_v1(p_candidato_ref, p_bolsa_ref);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamp with time zone)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','TimeZone=UTC']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '9deeea99249543b918999cbc4b9526c2d1ff982ca732b12cb86ff75810e80e7a' THEN
  RAISE EXCEPTION 'B63: fuente leer_contacto_candidato_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.leer_contacto_candidato_externo_v1(p_candidato_ref text, p_corte timestamp with time zone)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path='pg_catalog' SET TimeZone='UTC' AS $b63$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_externo_v1(ARRAY['consulta'], p_candidato_ref);
 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 RETURN (
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa', p.bolsa_ref, 'version', v.version,
   'origen', CASE WHEN o.participacion_ref IS NULL THEN NULL ELSE jsonb_build_object('origen', o.origen,
      'vigente_hasta', to_char(o.vigente_hasta,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), 'ultimo_dia', to_char(o.ultimo_dia,'YYYY-MM-DD')) END,
   'confirmada_en', CASE WHEN cf.participacion_ref IS NULL THEN NULL ELSE to_char(cf.confirmada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END)
   ORDER BY p.bolsa_ref), '[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 JOIN LATERAL (SELECT max(dc.version) AS version FROM vec_bolsa_llamamientos.datos_contacto_participacion dc
                WHERE dc.participacion_ref = p.participacion_ref AND dc.registrada_en <= p_corte) v ON v.version IS NOT NULL
 LEFT JOIN vec_bolsa_llamamientos.origen_datos_contacto_participacion o ON o.participacion_ref = p.participacion_ref AND o.version = v.version
 LEFT JOIN vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 cf
   ON cf.participacion_ref = p.participacion_ref AND cf.version = v.version AND cf.confirmada_en <= p_corte
 WHERE p_corte IS NOT NULL);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamp with time zone,text[])');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'd85901d905d46537709708d7a6c96bd9b4d7f3899e3af3e5e2c266ef51bb7755' THEN
  RAISE EXCEPTION 'B63: fuente leer_portal_candidato_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.leer_portal_candidato_externo_v1(p_candidato_ref text, p_corte timestamp with time zone, p_resultados_efectivos text[])
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path='pg_catalog' AS $b63$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_externo_v1(ARRAY['consulta','responder'], p_candidato_ref);
 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 RETURN (
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'bolsa', p.bolsa_ref,
   'llamamiento_abierto', CASE WHEN a.contacto_en IS NULL THEN NULL ELSE jsonb_build_object(
      'contacto_en', to_char(a.contacto_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END,
   'solicitud_pendiente', (SELECT jsonb_build_object('tipo', s.tipo, 'recibo', s.recibo_ref,
        'registrada_en', to_char(s.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'pausa_hasta', CASE WHEN s.pausa_hasta IS NULL THEN NULL ELSE to_char(s.pausa_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END)
      FROM vec_bolsa_llamamientos.solicitud_portal_candidato_lectura_portal_externa_v1 s
     WHERE s.participacion_ref = p.participacion_ref AND s.registrada_en <= p_corte
       AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)
     ORDER BY s.registrada_en DESC LIMIT 1),
   'ultima_respuesta', (SELECT jsonb_build_object('respuesta', r.respuesta, 'modo', r.modo, 'recibo', r.recibo_ref,
        'respondida_en', to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
      FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 r
     WHERE r.participacion_ref = p.participacion_ref AND r.respondida_en <= p_corte
     ORDER BY r.respondida_en DESC LIMIT 1)
 ) ORDER BY p.bolsa_ref), '[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 LEFT JOIN LATERAL vec_bolsa_llamamientos.llamamiento_abierto_portal_externo_v1(p.participacion_ref, p_corte, p_resultados_efectivos) a ON true);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamp with time zone)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','TimeZone=UTC']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'c571a95d1e617819930a5a0ba48011b6f971f484ed7bdc9c10d153ee79f46028' THEN
  RAISE EXCEPTION 'B63: fuente listar_ofertas_candidato_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.listar_ofertas_candidato_externo_v1(p_candidato_ref text, p_corte timestamp with time zone)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path='pg_catalog' SET TimeZone='UTC' AS $b63$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_externo_v1(ARRAY['consulta'], p_candidato_ref);
 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 RETURN (
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'oferta_ref', o.oferta_ref, 'bolsa', o.bolsa_ref, 'datos', o.datos,
   'publicada_en', to_char(o.publicada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'vence_antes_de', to_char(o.vence_antes_de,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado', CASE WHEN r.oferta_ref IS NOT NULL AND r.participacion_ref = p.participacion_ref THEN 'adjudicada_propia'
                  WHEN r.oferta_ref IS NOT NULL THEN 'resuelta'
                  WHEN pa.tipo IN ('adjudicada','aceptada') THEN 'adjudicada_propia'
                  WHEN pa.tipo IN ('renuncia','sin_respuesta') THEN 'resuelta'
                  WHEN p_corte < o.vence_antes_de THEN 'abierta'
                  WHEN vec_bolsa_llamamientos.proyectar_oferta_v2(o.oferta_ref, p_corte)->>'estado'
                       IN ('adjudicada','llamamiento_directo','cerrada') THEN 'resuelta'
                  ELSE 'pendiente_resolucion' END,
   'disposicion', CASE WHEN d.oferta_ref IS NULL THEN NULL ELSE jsonb_build_object('recibo', d.recibo_ref,
      'manifestada_en', to_char(d.manifestada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   ORDER BY o.vence_antes_de, o.oferta_ref), '[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 JOIN vec_bolsa_llamamientos.oferta_publicada o ON o.bolsa_ref = p.bolsa_ref AND o.publicada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.disposicion_oferta_lectura_portal_externa_v1 d
   ON d.oferta_ref = o.oferta_ref AND d.participacion_ref = p.participacion_ref AND d.manifestada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.resolucion_oferta r ON r.oferta_ref = o.oferta_ref AND r.resuelta_en <= p_corte
 LEFT JOIN LATERAL (SELECT a.tipo FROM vec_bolsa_llamamientos.acto_plaza_oferta a
                     WHERE a.oferta_ref = o.oferta_ref AND a.participacion_ref = p.participacion_ref AND a.registrado_en <= p_corte
                     ORDER BY a.registrado_en DESC, a.secuencia DESC LIMIT 1) pa ON true
 WHERE p_corte IS NOT NULL
   AND ((r.oferta_ref IS NULL AND p_corte < o.vence_antes_de) OR d.oferta_ref IS NOT NULL));
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(text,timestamp with time zone,text[])');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'c6c98bf38a5dcfe73557de0c32dcb9ee81ec779983dc421badbd55841ec15f06' THEN
  RAISE EXCEPTION 'B63: fuente llamamiento_abierto_portal_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.llamamiento_abierto_portal_externo_v1(p_participacion_ref text, p_corte timestamp with time zone, p_resultados_efectivos text[])
RETURNS TABLE(llamamiento_ref text, contacto_en timestamp with time zone) LANGUAGE sql STABLE SECURITY DEFINER SET search_path='pg_catalog' AS $b63$
 SELECT l.llamamiento_ref, ct.instante
   FROM (SELECT e.llamamiento_ref, e.emitido_en
           FROM vec_bolsa_llamamientos.llamamiento_emitido e
          WHERE e.participaciones ? p_participacion_ref AND e.emitido_en <= p_corte
          ORDER BY e.emitido_en DESC, e.llamamiento_ref DESC LIMIT 1) l
   CROSS JOIN LATERAL (
     SELECT min(c.instante) AS instante
       FROM vec_bolsa_llamamientos.contacto_participacion c
      WHERE c.participacion_ref = p_participacion_ref AND c.resultado = ANY (p_resultados_efectivos)
        AND c.instante >= l.emitido_en AND c.instante <= p_corte
   ) ct
  WHERE ct.instante IS NOT NULL AND cardinality(p_resultados_efectivos) > 0
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 r
                     WHERE r.llamamiento_ref = l.llamamiento_ref AND r.participacion_ref = p_participacion_ref)
$b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','TimeZone=UTC','lock_timeout=2s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '29998a71df14ee29428ac324ad72aa5389c353b57904462d1e99ccf56334359e' THEN
  RAISE EXCEPTION 'B63: fuente manifestar_disposicion_oferta_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.manifestar_disposicion_oferta_externo_v1(p_oferta_ref text, p_recibo_ref text, p_candidato_ref text, p_clave text, p_manifestada_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, oferta_ref text, manifestada_en timestamp with time zone) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET TimeZone='UTC' SET lock_timeout='2s' AS $b63$
DECLARE c jsonb; d jsonb; x jsonb; n integer; o vec_bolsa_llamamientos.oferta_publicada%ROWTYPE;
 v_participacion text; v_previa vec_bolsa_llamamientos.disposicion_oferta_externa%ROWTYPE; v_consumo record;
 v_accion constant text := 'bolsa.participaciones_propias.manifestar_disposicion';
 v_ahora timestamptz := clock_timestamp();
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR p_oferta_ref IS NULL OR p_oferta_ref !~ '^oferta:[0-9a-f]{64}$'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:disposicion:[0-9a-f]{64}$'
    OR p_manifestada_en IS NULL OR p_manifestada_en > v_ahora + interval '1 minute' OR p_manifestada_en < v_ahora - interval '5 minutes' THEN
  RAISE EXCEPTION 'disposición a oferta inválida' USING ERRCODE='22023';
 END IF;
 BEGIN c := convert_from(p_capacidad,'UTF8')::jsonb; d := convert_from(p_decision,'UTF8')::jsonb; x := convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de disposición inválido' USING ERRCODE='22023'; END;
 -- Acción y recurso exactos: la oferta es el recurso.
 IF c->>'efecto_ref' IS DISTINCT FROM p_oferta_ref OR d->>'recurso_ref' IS DISTINCT FROM p_oferta_ref
    OR c->>'operacion' IS DISTINCT FROM v_accion OR d->>'accion' IS DISTINCT FROM v_accion
    OR d->>'tipo_recurso' IS DISTINCT FROM 'oferta_bolsa'
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'disposición a oferta denegada' USING ERRCODE='42501';
 END IF;
 -- El candidato es el único vínculo candidato activo del contexto atestado.
 SELECT count(*) INTO n FROM jsonb_array_elements(x->'vinculos') e WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF n <> 1 OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
                           WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref) THEN
  RAISE EXCEPTION 'disposición a oferta denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada x2 WHERE x2.oferta_ref = p_oferta_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'oferta inexistente' USING ERRCODE='23503'; END IF;
 v_participacion := vec_bolsa_llamamientos.participacion_oferta_candidato_v1(p_candidato_ref, o.bolsa_ref);
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 -- Como B2 y el portal (000030), la decisión viva se consume antes de
 -- resolver el replay: un reintento no devuelve el recibo sin una
 -- autorización nueva y verificada.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM p_oferta_ref THEN
  RAISE EXCEPTION 'disposición a oferta denegada' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:' || p_oferta_ref, 0));
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.disposicion_oferta_lectura_portal_externa_v1 dp
  WHERE dp.oferta_ref = p_oferta_ref AND dp.participacion_ref = v_participacion;
 IF FOUND THEN
  IF v_previa.clave_idempotencia <> p_clave THEN
   RAISE EXCEPTION 'disposición ya manifestada' USING ERRCODE='VBO05';
  END IF;
  RETURN QUERY SELECT true, v_previa.recibo_ref, v_previa.oferta_ref, v_previa.manifestada_en;
  RETURN;
 END IF;
 IF clock_timestamp() >= o.vence_antes_de THEN
  RAISE EXCEPTION 'la oferta no está abierta' USING ERRCODE='VBO06';
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_disposicion_oferta_externa_interna_v1(
  p_oferta_ref, p_recibo_ref, p_candidato_ref, v_participacion, p_clave, p_manifestada_en, v_consumo.decision_ref);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.preparar_respuesta_portal_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '5b896bbc3be02068a9efea73f9382ca37e5979d5751fdd83b5d91aa3a3fbe586' THEN
  RAISE EXCEPTION 'B63: fuente preparar_respuesta_portal_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.preparar_respuesta_portal_externo_v1(p_candidato_ref text, p_bolsa_ref text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET lock_timeout='2s' AS $b63$
DECLARE v_consumo record;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 PERFORM vec_bolsa_llamamientos.exigir_portal_candidato_externo_v1(p_candidato_ref, p_bolsa_ref,
   'bolsa.participaciones_propias.responder_llamamiento', p_capacidad, p_decision, p_contexto);
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR v_consumo.decision_ref IS NULL OR strpos(v_consumo.decision_ref, chr(31)) <> 0 THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 -- El dato liga la marca a este material exacto: huella de la decisión y su referencia.
 PERFORM vec_bolsa_llamamientos.anotar_consumo_candidato_externo_v1('responder', p_candidato_ref,
   encode(sha256(p_decision), 'hex') || v_consumo.decision_ref);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(text,text,text,bigint,text,text,timestamp with time zone,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'a58d9b9cd6a0c9e20b9fd4d3c4b779093d08b320f545cf6c10140043875d64bf' THEN
  RAISE EXCEPTION 'B63: fuente registrar_confirmacion_contacto_interna_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_confirmacion_contacto_externa_interna_v1(p_candidato_ref text, p_bolsa_ref text, p_participacion_ref text, p_version bigint, p_clave text, p_recibo_ref text, p_confirmada_en timestamp with time zone, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, recibo_ref text, version bigint, confirmada_en timestamp with time zone) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' AS $b63$
DECLARE v_vigente bigint;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_version IS NULL OR p_version < 1 OR p_confirmada_en IS NULL OR p_decision_ref IS NULL
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:confirmacion-contacto:[0-9a-f]{64}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 THEN
  RAISE EXCEPTION 'confirmación de contacto inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:datos_contacto:' || p_participacion_ref, 0));
 SELECT max(x.version) INTO v_vigente FROM vec_bolsa_llamamientos.datos_contacto_participacion x WHERE x.participacion_ref = p_participacion_ref;
 IF v_vigente IS NULL THEN RAISE EXCEPTION 'sin contacto que confirmar' USING ERRCODE='VBC03'; END IF;
 -- Se confirma la versión que la persona vio: otra más reciente la invalida.
 IF v_vigente <> p_version THEN RAISE EXCEPTION 'el contacto ha cambiado' USING ERRCODE='VBC02'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_contacto_participacion_lectura_portal_externa_v1 x
             WHERE x.participacion_ref = p_participacion_ref AND x.version = p_version) THEN
  RAISE EXCEPTION 'contacto ya confirmado' USING ERRCODE='VBC04';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.confirmacion_contacto_participacion_externa(participacion_ref, version, candidato_ref, bolsa_ref, clave_idempotencia,
   recibo_ref, decision_ref, confirmada_en)
 VALUES (p_participacion_ref, p_version, p_candidato_ref, p_bolsa_ref, p_clave, p_recibo_ref, p_decision_ref, p_confirmada_en);
 RETURN QUERY SELECT false, p_recibo_ref, p_version, p_confirmada_en;
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(text,text,text,text,text,timestamp with time zone,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '21b22dac12277a6463c06d77f074aec2a88d715a5592afa88b4570879b1f9d41' THEN
  RAISE EXCEPTION 'B63: fuente registrar_disposicion_oferta_interna_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_disposicion_oferta_externa_interna_v1(p_oferta_ref text, p_recibo_ref text, p_candidato_ref text, p_participacion_ref text, p_clave text, p_manifestada_en timestamp with time zone, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, recibo_ref text, oferta_ref text, manifestada_en timestamp with time zone) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' AS $b63$
DECLARE o vec_bolsa_llamamientos.oferta_publicada%ROWTYPE;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_manifestada_en IS NULL OR p_decision_ref IS NULL
    OR p_recibo_ref IS NULL OR p_recibo_ref !~ '^recibo:disposicion:[0-9a-f]{64}$'
    OR p_clave IS NULL OR p_clave <> btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 THEN
  RAISE EXCEPTION 'disposición a oferta inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:' || p_oferta_ref, 0));
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada x WHERE x.oferta_ref = p_oferta_ref;
 IF NOT FOUND THEN RAISE EXCEPTION 'oferta inexistente' USING ERRCODE='23503'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.disposicion_oferta_lectura_portal_externa_v1 d WHERE d.oferta_ref = p_oferta_ref AND d.participacion_ref = p_participacion_ref) THEN
  RAISE EXCEPTION 'disposición ya manifestada' USING ERRCODE='VBO05';
 END IF;
 -- Abierta: publicada, sin resolución y antes del vencimiento, tanto en el
 -- instante declarado como en el reloj de la base al registrar.
 IF p_manifestada_en < o.publicada_en OR p_manifestada_en >= o.vence_antes_de OR clock_timestamp() >= o.vence_antes_de
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.resolucion_oferta r WHERE r.oferta_ref = p_oferta_ref) THEN
  RAISE EXCEPTION 'la oferta no está abierta' USING ERRCODE='VBO06';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.disposicion_oferta_externa(oferta_ref, participacion_ref, manifestada_en, clave_idempotencia, recibo_ref)
 VALUES (p_oferta_ref, p_participacion_ref, p_manifestada_en, p_clave, p_recibo_ref);
 INSERT INTO vec_bolsa_llamamientos.disposicion_oferta_candidato_externa(oferta_ref, participacion_ref, candidato_ref, decision_ref, registrada_en)
 VALUES (p_oferta_ref, p_participacion_ref, p_candidato_ref, p_decision_ref, p_manifestada_en);
 RETURN QUERY SELECT false, p_recibo_ref, p_oferta_ref, p_manifestada_en;
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '5e675b8ad2662dd3d0a81cde94c30d5013fd1574ebe2ee36aa9f8e71e2f4b102' THEN
  RAISE EXCEPTION 'B63: fuente registrar_respuesta_portal_interna_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_respuesta_portal_externa_interna_v1(p_respuesta_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_participacion_ref text, p_respuesta text, p_causa text, p_justificante_ref text, p_justificante_sha256 text, p_modo text, p_contacto_en timestamp with time zone, p_vence_antes_de timestamp with time zone, p_resultados_efectivos text[], p_regla_ref text, p_clave text, p_respondida_en timestamp with time zone, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, respuesta_ref text, recibo_ref text, respondida_en timestamp with time zone, modo text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' AS $b63$
DECLARE v_abierto record;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_respuesta NOT IN ('acepta','renuncia','renuncia_justificada')
    OR p_modo NOT IN ('firme','propuesta_rrhh') OR p_respondida_en IS NULL OR p_vence_antes_de <= p_contacto_en THEN
  RAISE EXCEPTION 'respuesta del portal inválida' USING ERRCODE='22023';
 END IF;
 -- Sin contacto leído por el servidor no hay llamamiento que responder.
 IF p_contacto_en IS NULL OR p_vence_antes_de IS NULL THEN
  RAISE EXCEPTION 'no hay un llamamiento abierto' USING ERRCODE='VBP04';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:portal:'||p_participacion_ref, 0));
 SELECT * INTO v_abierto FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_externo_v1(p_participacion_ref, p_respondida_en, p_resultados_efectivos);
 IF v_abierto.llamamiento_ref IS NULL OR v_abierto.contacto_en IS DISTINCT FROM p_contacto_en THEN
  RAISE EXCEPTION 'no hay un llamamiento abierto con ese contacto' USING ERRCODE='VBP04';
 END IF;
 -- Fuera de plazo no se admite: queda como «sin respuesta» para RRHH.
 IF p_respondida_en >= p_vence_antes_de THEN
  RAISE EXCEPTION 'respuesta fuera de plazo' USING ERRCODE='VBP05';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.respuesta_portal_llamamiento_externa(respuesta_ref, recibo_ref, bolsa_ref, participacion_ref, candidato_ref,
   llamamiento_ref, respuesta, causa, justificante_ref, justificante_sha256, modo, contacto_en, vence_antes_de, regla_ref,
   clave_idempotencia, decision_ref, respondida_en)
 VALUES (p_respuesta_ref, p_recibo_ref, p_bolsa_ref, p_participacion_ref, p_candidato_ref, v_abierto.llamamiento_ref, p_respuesta,
   p_causa, p_justificante_ref, p_justificante_sha256, p_modo, p_contacto_en, p_vence_antes_de, p_regla_ref, p_clave,
   p_decision_ref, p_respondida_en);
 RETURN QUERY SELECT false, p_respuesta_ref, p_recibo_ref, p_respondida_en, p_modo;
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '93c6793321f509bcf23e34f1c2aec0dfb56b9682e65fa5e89451242a9fa6c476' THEN
  RAISE EXCEPTION 'B63: fuente registrar_solicitud_portal_interna_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_solicitud_portal_externa_interna_v1(p_solicitud_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_participacion_ref text, p_tipo text, p_pausa_hasta timestamp with time zone, p_pausa_maxima timestamp with time zone, p_situaciones_admitidas text[], p_regla_ref text, p_clave text, p_registrada_en timestamp with time zone, p_decision_ref text)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, recibo_ref text, registrada_en timestamp with time zone) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' AS $b63$
DECLARE v_situacion text;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_tipo NOT IN ('pausa','reactivacion') OR p_registrada_en IS NULL
    OR cardinality(p_situaciones_admitidas) IS NULL OR cardinality(p_situaciones_admitidas) = 0
    OR (p_tipo = 'pausa' AND (p_pausa_hasta IS NULL OR p_pausa_maxima IS NULL OR p_pausa_hasta <= p_registrada_en OR p_pausa_hasta > p_pausa_maxima))
    OR (p_tipo = 'reactivacion' AND (p_pausa_hasta IS NOT NULL OR p_pausa_maxima IS NOT NULL)) THEN
  RAISE EXCEPTION 'solicitud del portal inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:portal:'||p_participacion_ref, 0));
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.solicitud_portal_candidato_lectura_portal_externa_v1 s
             WHERE s.participacion_ref = p_participacion_ref AND vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(s.solicitud_ref)) THEN
  RAISE EXCEPTION 'ya hay una solicitud pendiente de RRHH' USING ERRCODE='VBP02';
 END IF;
 SELECT s.situacion INTO v_situacion FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref = p_participacion_ref AND s.desde <= p_registrada_en ORDER BY s.desde DESC LIMIT 1;
 -- Sin hecho B2 la participación está en su situación inicial, disponible.
 v_situacion := coalesce(v_situacion, 'disponible');
 IF NOT v_situacion = ANY (p_situaciones_admitidas) THEN
  RAISE EXCEPTION 'la situación actual no admite esta solicitud' USING ERRCODE='VBP03';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.solicitud_portal_candidato_externa(solicitud_ref, recibo_ref, bolsa_ref, participacion_ref, candidato_ref, tipo,
   pausa_hasta, situacion_previa, regla_ref, clave_idempotencia, decision_ref, registrada_en)
 VALUES (p_solicitud_ref, p_recibo_ref, p_bolsa_ref, p_participacion_ref, p_candidato_ref, p_tipo,
   p_pausa_hasta, v_situacion, p_regla_ref, p_clave, p_decision_ref, p_registrada_en);
 RETURN QUERY SELECT false, p_solicitud_ref, p_recibo_ref, p_registrada_en;
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'b5d47c6df5facc2bb704fc2ecb2fb096825c46f4c23790bd9582ef9a8eb35c03' THEN
  RAISE EXCEPTION 'B63: fuente responder_llamamiento_portal_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.responder_llamamiento_portal_externo_v1(p_respuesta_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_respuesta text, p_causa text, p_justificante_ref text, p_justificante_sha256 text, p_modo text, p_contacto_en timestamp with time zone, p_vence_antes_de timestamp with time zone, p_resultados_efectivos text[], p_regla_ref text, p_clave text, p_respondida_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, respuesta_ref text, recibo_ref text, respondida_en timestamp with time zone, modo text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET lock_timeout='2s' AS $b63$
DECLARE v_participacion text; v_previa vec_bolsa_llamamientos.respuesta_portal_llamamiento_externa%ROWTYPE; v_consumo record;
 v_dato text; v_decision_ref text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 v_participacion := vec_bolsa_llamamientos.exigir_portal_candidato_externo_v1(p_candidato_ref, p_bolsa_ref,
   'bolsa.participaciones_propias.responder_llamamiento', p_capacidad, p_decision, p_contexto);
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 -- Como B2, la decisión viva se consume antes de resolver el replay: un
 -- reintento no devuelve el recibo sin una autorización nueva y verificada.
 -- Si preparar_respuesta_portal_v1 ya la consumió en esta transacción con
 -- este mismo material, se usa esa (una sola vez); si no, se consume aquí.
 BEGIN
  v_dato := vec_bolsa_llamamientos.exigir_consumo_candidato_externo_v1(ARRAY['responder'], p_candidato_ref);
 EXCEPTION WHEN insufficient_privilege THEN v_dato := NULL;
 END;
 IF v_dato IS NOT NULL AND left(v_dato, 64) = encode(sha256(p_decision), 'hex') AND octet_length(v_dato) > 64 THEN
  v_decision_ref := substr(v_dato, 65);
  PERFORM set_config('vec_bolsa_llamamientos.marca_consumo_portal_externo', '', true);
 ELSE
  SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
   p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
  IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
   RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
  END IF;
  v_decision_ref := v_consumo.decision_ref;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:portal:'||v_participacion,0));
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento_lectura_portal_externa_v1 r
  WHERE r.participacion_ref = v_participacion AND r.clave_idempotencia = p_clave;
 IF FOUND THEN
  IF v_previa.respuesta <> p_respuesta OR v_previa.causa IS DISTINCT FROM p_causa OR v_previa.justificante_ref IS DISTINCT FROM p_justificante_ref
     OR v_previa.justificante_sha256 IS DISTINCT FROM p_justificante_sha256 OR v_previa.candidato_ref <> p_candidato_ref
     OR v_previa.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR v_previa.modo IS DISTINCT FROM p_modo
     OR v_previa.contacto_en IS DISTINCT FROM p_contacto_en OR v_previa.vence_antes_de IS DISTINCT FROM p_vence_antes_de
     OR v_previa.regla_ref IS DISTINCT FROM p_regla_ref THEN
   RAISE EXCEPTION 'clave idempotente reutilizada con otra respuesta' USING ERRCODE='VBP01';
  END IF;
  RETURN QUERY SELECT true, v_previa.respuesta_ref, v_previa.recibo_ref, v_previa.respondida_en, v_previa.modo;
  RETURN;
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_respuesta_portal_externa_interna_v1(
  p_respuesta_ref, p_recibo_ref, p_candidato_ref, p_bolsa_ref, v_participacion, p_respuesta, p_causa, p_justificante_ref,
  p_justificante_sha256, p_modo, p_contacto_en, p_vence_antes_de, p_resultados_efectivos, p_regla_ref, p_clave,
  p_respondida_en, v_decision_ref);
END $b63$;

DO $fuente$
DECLARE p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']::text[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'a1d8bf50f4eebfda08514a7d12c1c98da477ad42258fda5bdb745b61fe09e2ab' THEN
  RAISE EXCEPTION 'B63: fuente solicitar_portal_candidato_v1 divergente' USING ERRCODE='55000'; END IF;
END $fuente$;
CREATE FUNCTION vec_bolsa_llamamientos.solicitar_portal_candidato_externo_v1(p_solicitud_ref text, p_recibo_ref text, p_candidato_ref text, p_bolsa_ref text, p_tipo text, p_pausa_hasta timestamp with time zone, p_pausa_maxima timestamp with time zone, p_situaciones_admitidas text[], p_regla_ref text, p_clave text, p_registrada_en timestamp with time zone, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(reutilizada boolean, solicitud_ref text, recibo_ref text, registrada_en timestamp with time zone) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path='pg_catalog' SET lock_timeout='2s' AS $b63$
DECLARE v_participacion text; v_previa vec_bolsa_llamamientos.solicitud_portal_candidato_externa%ROWTYPE; v_consumo record; v_accion text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1();
 v_accion := CASE p_tipo WHEN 'pausa' THEN 'bolsa.participaciones_propias.solicitar_pausa'
                         WHEN 'reactivacion' THEN 'bolsa.participaciones_propias.solicitar_reactivacion' END;
 IF v_accion IS NULL THEN RAISE EXCEPTION 'solicitud del portal inválida' USING ERRCODE='22023'; END IF;
 v_participacion := vec_bolsa_llamamientos.exigir_portal_candidato_externo_v1(p_candidato_ref, p_bolsa_ref, v_accion, p_capacidad, p_decision, p_contexto);
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 -- Como B2, la decisión viva se consume antes de resolver el replay: un
 -- reintento no devuelve el recibo sin una autorización nueva y verificada.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
  p_capacidad, p_decision, p_motivo, p_contexto, p_persona_version, p_perfil_version, p_payload, p_sobre, p_evidencia, p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref THEN
  RAISE EXCEPTION 'portal del candidato denegado' USING ERRCODE='42501';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:portal:'||v_participacion,0));
 PERFORM vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(p_candidato_ref,v_participacion,p_contexto);
 PERFORM vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(p_candidato_ref);
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.solicitud_portal_candidato_lectura_portal_externa_v1 s
  WHERE s.participacion_ref = v_participacion AND s.clave_idempotencia = p_clave;
 IF FOUND THEN
  IF v_previa.tipo <> p_tipo OR v_previa.pausa_hasta IS DISTINCT FROM p_pausa_hasta OR v_previa.candidato_ref <> p_candidato_ref
     OR v_previa.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR v_previa.regla_ref IS DISTINCT FROM p_regla_ref THEN
   RAISE EXCEPTION 'clave idempotente reutilizada con otra solicitud' USING ERRCODE='VBP01';
  END IF;
  RETURN QUERY SELECT true, v_previa.solicitud_ref, v_previa.recibo_ref, v_previa.registrada_en;
  RETURN;
 END IF;
 RETURN QUERY SELECT * FROM vec_bolsa_llamamientos.registrar_solicitud_portal_externa_interna_v1(
  p_solicitud_ref, p_recibo_ref, p_candidato_ref, p_bolsa_ref, v_participacion, p_tipo, p_pausa_hasta, p_pausa_maxima,
  p_situaciones_admitidas, p_regla_ref, p_clave, p_registrada_en, v_consumo.decision_ref);
END $b63$;

DO $acl$ DECLARE f regprocedure; a record; BEGIN
 f:='vec_bolsa_llamamientos.confirmar_contacto_propio_externo_v1(text,text,bigint,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.confirmar_contacto_propio_externo_v1(text,text,bigint,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.consultar_historial_mi_bolsa_externo_v1(text,timestamp with time zone,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamp with time zone,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_historial_mi_bolsa_externo_v1(text,timestamp with time zone,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.consultar_mi_bolsa_portal_externo_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_portal_externo_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.consultar_mi_bolsa_externo_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_mi_bolsa_externo_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.exigir_portal_candidato_externo_v1(text,text,text,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 f:='vec_bolsa_llamamientos.leer_contacto_candidato_externo_v1(text,timestamp with time zone)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamp with time zone) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_contacto_candidato_externo_v1(text,timestamp with time zone) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.leer_portal_candidato_externo_v1(text,timestamp with time zone,text[])'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamp with time zone,text[]) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_portal_candidato_externo_v1(text,timestamp with time zone,text[]) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.listar_ofertas_candidato_externo_v1(text,timestamp with time zone)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamp with time zone) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_ofertas_candidato_externo_v1(text,timestamp with time zone) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.llamamiento_abierto_portal_externo_v1(text,timestamp with time zone,text[])'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 f:='vec_bolsa_llamamientos.manifestar_disposicion_oferta_externo_v1(text,text,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.manifestar_disposicion_oferta_externo_v1(text,text,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.preparar_respuesta_portal_externo_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.preparar_respuesta_portal_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.preparar_respuesta_portal_externo_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.registrar_confirmacion_contacto_externa_interna_v1(text,text,text,bigint,text,text,timestamp with time zone,text)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 f:='vec_bolsa_llamamientos.registrar_disposicion_oferta_externa_interna_v1(text,text,text,text,text,timestamp with time zone,text)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 f:='vec_bolsa_llamamientos.registrar_respuesta_portal_externa_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,text)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 f:='vec_bolsa_llamamientos.registrar_solicitud_portal_externa_interna_v1(text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,text)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 f:='vec_bolsa_llamamientos.responder_llamamiento_portal_externo_v1(text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.responder_llamamiento_portal_externo_v1(text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
 f:='vec_bolsa_llamamientos.solicitar_portal_candidato_externo_v1(text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 FOR a IN SELECT x.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) x WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f,CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END);
 END LOOP;
 REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_portal_externo;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.solicitar_portal_candidato_externo_v1(text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_portal_externo;
END $acl$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.serializar_contexto_participacion_externa_v1() FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.huella_contexto_participacion_externa_v1(jsonb,bigint) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.preimagen_contexto_participacion_externa_v1(text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1(jsonb,bigint,text,text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_contexto_participacion_externa_v1(text,text,bytea) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_runtime_bolsa_portal_externo_v1() FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.anotar_consumo_candidato_externo_v1(text,text,text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_consumo_candidato_externo_v1(text[],text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_integridad_portal_externo_v1(text) FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.rechazar_colision_actuacion_externa_v1() FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo;
RESET ROLE;
DO $postimagen$
DECLARE grupo oid:='vec_bolsa_llamamientos_portal_externo'::regrole; n integer;
BEGIN
 SELECT count(*) INTO n FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.pronamespace='vec_bolsa_llamamientos'::regnamespace AND a.grantee=grupo;
 IF n<>11 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.pronamespace='vec_bolsa_llamamientos'::regnamespace AND a.grantee=grupo AND (p.proname !~ '_externo_v1$' OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'B63: ACL nominal exterior incompatible' USING ERRCODE='55000'; END IF;
END $postimagen$;
COMMIT;
