\set ON_ERROR_STOP on
-- Documentos 000010: custodia física externa; instalar después de 000008.
-- Una sola transacción, con servicio detenido. No ejecutar DOWN sobre historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='120s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000010',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_documentos.imagen_personal') IS NULL
    OR to_regclass('vec_documentos.imagen_personal_historia') IS NULL
    OR to_regclass('vec_documentos.imagen_personal_externa') IS NOT NULL
    OR to_regprocedure('vec_documentos.superficie_sesion_imagen_v1()') IS NULL
 THEN RAISE EXCEPTION 'Documentos 000010: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_documentos.imagen_personal,vec_documentos.imagen_personal_historia IN ACCESS EXCLUSIVE MODE;
CREATE TEMP TABLE d10_antes ON COMMIT DROP AS
 SELECT imagen_ref,to_jsonb(i) fila FROM vec_documentos.imagen_personal i WHERE superficie='externa_personal';
CREATE TEMP TABLE d10_historia ON COMMIT DROP AS
 SELECT imagen_ref,evento,to_jsonb(h) fila FROM vec_documentos.imagen_personal_historia h
 WHERE imagen_ref IN (SELECT imagen_ref FROM d10_antes);
CREATE TABLE vec_documentos.imagen_personal_externa (LIKE vec_documentos.imagen_personal INCLUDING ALL);
CREATE TABLE vec_documentos.imagen_personal_historia_externa (LIKE vec_documentos.imagen_personal_historia INCLUDING ALL);
ALTER TABLE vec_documentos.imagen_personal_historia_externa ADD CONSTRAINT imagen_personal_historia_externa_imagen_fkey
 FOREIGN KEY(imagen_ref) REFERENCES vec_documentos.imagen_personal_externa(imagen_ref);
ALTER TABLE vec_documentos.imagen_personal_externa ADD CONSTRAINT imagen_personal_externa_superficie
 CHECK(superficie='externa_personal');
INSERT INTO vec_documentos.imagen_personal_externa SELECT * FROM vec_documentos.imagen_personal WHERE superficie='externa_personal';
INSERT INTO vec_documentos.imagen_personal_historia_externa SELECT h.* FROM vec_documentos.imagen_personal_historia h
 JOIN d10_antes a USING(imagen_ref);
ALTER TABLE vec_documentos.imagen_personal_historia DISABLE TRIGGER imagen_personal_historia_inmutable;
DELETE FROM vec_documentos.imagen_personal_historia WHERE imagen_ref IN (SELECT imagen_ref FROM d10_antes);
ALTER TABLE vec_documentos.imagen_personal_historia ENABLE TRIGGER imagen_personal_historia_inmutable;
ALTER TABLE vec_documentos.imagen_personal DISABLE TRIGGER imagen_personal_transicion;
DELETE FROM vec_documentos.imagen_personal WHERE superficie='externa_personal';
ALTER TABLE vec_documentos.imagen_personal ENABLE TRIGGER imagen_personal_transicion;
DO $control$ BEGIN
 IF EXISTS(SELECT 1 FROM d10_antes a FULL JOIN vec_documentos.imagen_personal_externa e USING(imagen_ref)
     WHERE a.fila IS DISTINCT FROM to_jsonb(e))
    OR EXISTS(SELECT 1 FROM d10_historia a FULL JOIN vec_documentos.imagen_personal_historia_externa e USING(imagen_ref,evento)
     WHERE a.fila IS DISTINCT FROM to_jsonb(e))
    OR EXISTS(SELECT 1 FROM vec_documentos.imagen_personal WHERE superficie<>'interna_corporativa')
 THEN RAISE EXCEPTION 'Documentos 000010: copia incompatible' USING ERRCODE='55000'; END IF;
END $control$;
ALTER TABLE vec_documentos.imagen_personal ADD CONSTRAINT imagen_personal_solo_interna CHECK(superficie='interna_corporativa');
ALTER TABLE vec_documentos.imagen_personal_externa ALTER COLUMN contenido SET STORAGE EXTERNAL;
ALTER TABLE vec_documentos.imagen_personal_externa OWNER TO vec_documentos_propietario;
ALTER TABLE vec_documentos.imagen_personal_historia_externa OWNER TO vec_documentos_propietario;
REVOKE ALL ON vec_documentos.imagen_personal_externa,vec_documentos.imagen_personal_historia_externa
 FROM PUBLIC,vec_usuarios_propietario,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
REVOKE ALL ON TYPE vec_documentos.imagen_personal_externa,vec_documentos.imagen_personal_historia_externa FROM PUBLIC;
ALTER TABLE vec_documentos.imagen_personal_externa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.imagen_personal_externa FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.imagen_personal_historia_externa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.imagen_personal_historia_externa FORCE ROW LEVEL SECURITY;
CREATE TRIGGER imagen_personal_externa_transicion BEFORE UPDATE OR DELETE ON vec_documentos.imagen_personal_externa
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.imagen_personal_transicion_v1();
CREATE TRIGGER imagen_personal_externa_no_truncar BEFORE TRUNCATE ON vec_documentos.imagen_personal_externa
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1();
CREATE TRIGGER imagen_personal_historia_externa_inmutable BEFORE UPDATE OR DELETE ON vec_documentos.imagen_personal_historia_externa
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1();
CREATE TRIGGER imagen_personal_historia_externa_no_truncar BEFORE TRUNCATE ON vec_documentos.imagen_personal_historia_externa
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1();
SET LOCAL ROLE vec_documentos_propietario;
CREATE POLICY imagen_externa_lectura ON vec_documentos.imagen_personal_externa FOR SELECT TO vec_documentos_propietario
 USING(vec_documentos.sesion_usuarios_imagen_v1() AND vec_documentos.superficie_sesion_imagen_v1()='externa_personal');
CREATE POLICY imagen_externa_alta ON vec_documentos.imagen_personal_externa FOR INSERT TO vec_documentos_propietario
 WITH CHECK(vec_documentos.sesion_usuarios_imagen_v1() AND vec_documentos.superficie_sesion_imagen_v1()='externa_personal');
CREATE POLICY imagen_externa_retirada ON vec_documentos.imagen_personal_externa FOR UPDATE TO vec_documentos_propietario
 USING(vec_documentos.sesion_usuarios_imagen_v1() AND vec_documentos.superficie_sesion_imagen_v1()='externa_personal')
 WITH CHECK(vec_documentos.sesion_usuarios_imagen_v1() AND vec_documentos.superficie_sesion_imagen_v1()='externa_personal');
CREATE POLICY imagen_externa_historia_alta ON vec_documentos.imagen_personal_historia_externa FOR INSERT TO vec_documentos_propietario
 WITH CHECK(vec_documentos.sesion_usuarios_imagen_v1() AND vec_documentos.superficie_sesion_imagen_v1()='externa_personal');


CREATE OR REPLACE FUNCTION vec_documentos.custodiar_imagen_personal_v1_interna(
 p_persona text,p_contenido bytea,p_sha256 text,p_operacion text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE ref text; ahora timestamptz(6); n integer; s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS DISTINCT FROM 'interna_corporativa'
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: custodia de imagen denegada' USING ERRCODE='42501'; END IF;
 n:=octet_length(p_contenido);
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_operacion IS NULL OR p_operacion !~ '^img_[0-9a-f]{32}$'
    OR p_sha256 IS NULL OR p_sha256 !~ '^[0-9a-f]{64}$'
    OR n IS NULL OR n NOT BETWEEN 4 AND 262144
    OR substring(p_contenido FROM 1 FOR 3)<>'\xffd8ff'::bytea
    OR substring(p_contenido FROM n-1 FOR 2)<>'\xffd9'::bytea
    OR encode(sha256(p_contenido),'hex')<>p_sha256
 THEN RAISE EXCEPTION 'documentos: imagen no admitida' USING ERRCODE='22023'; END IF;
 IF EXISTS(SELECT 1 FROM vec_documentos.imagen_personal WHERE persona_ref=p_persona AND superficie=s AND estado='activa')
 THEN RAISE EXCEPTION 'documentos: la persona ya tiene una imagen activa' USING ERRCODE='P1409'; END IF;
 ref:='docimg_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_documentos.imagen_personal(imagen_ref,persona_ref,superficie,mime,tamano,huella_sha256,contenido,estado,operacion_alta_ref,custodiada_en)
 VALUES(ref,p_persona,s,'image/jpeg',n,p_sha256,p_contenido,'activa',p_operacion,ahora);
 INSERT INTO vec_documentos.imagen_personal_historia(imagen_ref,evento,persona_ref,huella_sha256,tamano,operacion_ref,sesion,registrado_en)
 VALUES(ref,'custodiada',p_persona,p_sha256,n,p_operacion,session_user,ahora);
 RETURN ref;
END $f$;

REVOKE ALL ON FUNCTION vec_documentos.custodiar_imagen_personal_v1_interna(text,bytea,text,text) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.custodiar_imagen_personal_v1_externa(
 p_persona text,p_contenido bytea,p_sha256 text,p_operacion text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE ref text; ahora timestamptz(6); n integer; s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS DISTINCT FROM 'externa_personal'
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: custodia de imagen denegada' USING ERRCODE='42501'; END IF;
 n:=octet_length(p_contenido);
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_operacion IS NULL OR p_operacion !~ '^img_[0-9a-f]{32}$'
    OR p_sha256 IS NULL OR p_sha256 !~ '^[0-9a-f]{64}$'
    OR n IS NULL OR n NOT BETWEEN 4 AND 262144
    OR substring(p_contenido FROM 1 FOR 3)<>'\xffd8ff'::bytea
    OR substring(p_contenido FROM n-1 FOR 2)<>'\xffd9'::bytea
    OR encode(sha256(p_contenido),'hex')<>p_sha256
 THEN RAISE EXCEPTION 'documentos: imagen no admitida' USING ERRCODE='22023'; END IF;
 IF EXISTS(SELECT 1 FROM vec_documentos.imagen_personal_externa WHERE persona_ref=p_persona AND superficie=s AND estado='activa')
 THEN RAISE EXCEPTION 'documentos: la persona ya tiene una imagen activa' USING ERRCODE='P1409'; END IF;
 ref:='docimg_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_documentos.imagen_personal_externa(imagen_ref,persona_ref,superficie,mime,tamano,huella_sha256,contenido,estado,operacion_alta_ref,custodiada_en)
 VALUES(ref,p_persona,s,'image/jpeg',n,p_sha256,p_contenido,'activa',p_operacion,ahora);
 INSERT INTO vec_documentos.imagen_personal_historia_externa(imagen_ref,evento,persona_ref,huella_sha256,tamano,operacion_ref,sesion,registrado_en)
 VALUES(ref,'custodiada',p_persona,p_sha256,n,p_operacion,session_user,ahora);
 RETURN ref;
END $f$;

REVOKE ALL ON FUNCTION vec_documentos.custodiar_imagen_personal_v1_externa(text,bytea,text,text) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.custodiar_imagen_personal_v1(
 p_persona text,p_contenido bytea,p_sha256 text,p_operacion text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF vec_documentos.superficie_sesion_imagen_v1()='interna_corporativa' THEN
  RETURN vec_documentos.custodiar_imagen_personal_v1_interna(p_persona,p_contenido,p_sha256,p_operacion);
 ELSIF vec_documentos.superficie_sesion_imagen_v1()='externa_personal' THEN
  RETURN vec_documentos.custodiar_imagen_personal_v1_externa(p_persona,p_contenido,p_sha256,p_operacion);
 ELSE RAISE EXCEPTION 'documentos: superficie de imagen denegada' USING ERRCODE='42501'; END IF;

END $f$;

CREATE OR REPLACE FUNCTION vec_documentos.retirar_imagen_personal_v1_interna(p_persona text,p_imagen text,p_operacion text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE i record; ahora timestamptz(6); s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS DISTINCT FROM 'interna_corporativa'
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: retirada de imagen denegada' USING ERRCODE='42501'; END IF;
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_imagen IS NULL OR p_imagen !~ '^docimg_[0-9a-f]{32}$'
    OR p_operacion IS NULL OR p_operacion !~ '^img_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'documentos: retirada de imagen inválida' USING ERRCODE='22023'; END IF;
 SELECT imagen_ref,huella_sha256,tamano INTO i FROM vec_documentos.imagen_personal
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND superficie=s AND estado='activa' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'documentos: imagen ausente' USING ERRCODE='P1409'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 UPDATE vec_documentos.imagen_personal SET estado='retirada',contenido=NULL,retirada_en=ahora,operacion_retirada_ref=p_operacion
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND superficie=s AND estado='activa';
 INSERT INTO vec_documentos.imagen_personal_historia(imagen_ref,evento,persona_ref,huella_sha256,tamano,operacion_ref,sesion,registrado_en)
 VALUES(p_imagen,'retirada',p_persona,i.huella_sha256,i.tamano,p_operacion,session_user,ahora);
END $f$;

REVOKE ALL ON FUNCTION vec_documentos.retirar_imagen_personal_v1_interna(text,text,text) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.retirar_imagen_personal_v1_externa(p_persona text,p_imagen text,p_operacion text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE i record; ahora timestamptz(6); s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS DISTINCT FROM 'externa_personal'
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: retirada de imagen denegada' USING ERRCODE='42501'; END IF;
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_imagen IS NULL OR p_imagen !~ '^docimg_[0-9a-f]{32}$'
    OR p_operacion IS NULL OR p_operacion !~ '^img_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'documentos: retirada de imagen inválida' USING ERRCODE='22023'; END IF;
 SELECT imagen_ref,huella_sha256,tamano INTO i FROM vec_documentos.imagen_personal_externa
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND superficie=s AND estado='activa' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'documentos: imagen ausente' USING ERRCODE='P1409'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 UPDATE vec_documentos.imagen_personal_externa SET estado='retirada',contenido=NULL,retirada_en=ahora,operacion_retirada_ref=p_operacion
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND superficie=s AND estado='activa';
 INSERT INTO vec_documentos.imagen_personal_historia_externa(imagen_ref,evento,persona_ref,huella_sha256,tamano,operacion_ref,sesion,registrado_en)
 VALUES(p_imagen,'retirada',p_persona,i.huella_sha256,i.tamano,p_operacion,session_user,ahora);
END $f$;

REVOKE ALL ON FUNCTION vec_documentos.retirar_imagen_personal_v1_externa(text,text,text) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.retirar_imagen_personal_v1(p_persona text,p_imagen text,p_operacion text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF vec_documentos.superficie_sesion_imagen_v1()='interna_corporativa' THEN
  PERFORM vec_documentos.retirar_imagen_personal_v1_interna(p_persona,p_imagen,p_operacion);
 ELSIF vec_documentos.superficie_sesion_imagen_v1()='externa_personal' THEN
  PERFORM vec_documentos.retirar_imagen_personal_v1_externa(p_persona,p_imagen,p_operacion);
 ELSE RAISE EXCEPTION 'documentos: superficie de imagen denegada' USING ERRCODE='42501'; END IF;
 RETURN;
END $f$;

CREATE OR REPLACE FUNCTION vec_documentos.abrir_imagen_personal_v1_interna(p_persona text,p_imagen text)
RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE b bytea; h text; s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS DISTINCT FROM 'interna_corporativa'
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: lectura de imagen denegada' USING ERRCODE='42501'; END IF;
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_imagen IS NULL OR p_imagen !~ '^docimg_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'documentos: lectura de imagen inválida' USING ERRCODE='22023'; END IF;
 SELECT contenido,huella_sha256 INTO b,h FROM vec_documentos.imagen_personal
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND superficie=s AND estado='activa';
 IF b IS NULL THEN RETURN NULL; END IF;
 IF encode(sha256(b),'hex')<>h THEN RAISE EXCEPTION 'documentos: imagen alterada' USING ERRCODE='55000'; END IF;
 RETURN b;
END $f$;

REVOKE ALL ON FUNCTION vec_documentos.abrir_imagen_personal_v1_interna(text,text) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.abrir_imagen_personal_v1_externa(p_persona text,p_imagen text)
RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE b bytea; h text; s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS DISTINCT FROM 'externa_personal'
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: lectura de imagen denegada' USING ERRCODE='42501'; END IF;
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_imagen IS NULL OR p_imagen !~ '^docimg_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'documentos: lectura de imagen inválida' USING ERRCODE='22023'; END IF;
 SELECT contenido,huella_sha256 INTO b,h FROM vec_documentos.imagen_personal_externa
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND superficie=s AND estado='activa';
 IF b IS NULL THEN RETURN NULL; END IF;
 IF encode(sha256(b),'hex')<>h THEN RAISE EXCEPTION 'documentos: imagen alterada' USING ERRCODE='55000'; END IF;
 RETURN b;
END $f$;

REVOKE ALL ON FUNCTION vec_documentos.abrir_imagen_personal_v1_externa(text,text) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.abrir_imagen_personal_v1(p_persona text,p_imagen text)
RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF vec_documentos.superficie_sesion_imagen_v1()='interna_corporativa' THEN
  RETURN vec_documentos.abrir_imagen_personal_v1_interna(p_persona,p_imagen);
 ELSIF vec_documentos.superficie_sesion_imagen_v1()='externa_personal' THEN
  RETURN vec_documentos.abrir_imagen_personal_v1_externa(p_persona,p_imagen);
 ELSE RAISE EXCEPTION 'documentos: superficie de imagen denegada' USING ERRCODE='42501'; END IF;

END $f$;

RESET ROLE;
COMMIT;
