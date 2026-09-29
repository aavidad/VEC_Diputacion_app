\set ON_ERROR_STOP on
-- Documentos 000007 (Usuarios 5.08c): custodia de la foto con la que una
-- persona se identifica en la cabecera de los portales.
--
-- Documentos guarda el JPEG de 256 px ya recodificado por la aplicación (sin
-- EXIF, GPS ni ningún otro metadato; nunca el original) y devuelve a Usuarios
-- una referencia opaca. Usuarios no ve estas tablas: solo llama a las tres
-- funciones de abajo, y únicamente desde sus propias funciones SECURITY
-- DEFINER, después de consumir una autorización V3 fresca del titular en la
-- misma transacción. Al retirar o sustituir una foto sus bytes se borran en
-- el acto y queda solo la historia (referencia, huella, fechas y operación).
--
-- Solo añade objetos; no depende de las tablas de 000001–000006. Aplicar con
-- el migrador documental, después de AD3-108 y antes de Usuarios 000006.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000007',0));
DO $pre$ BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regnamespace('vec_documentos') IS NULL
    OR (SELECT nspowner FROM pg_namespace WHERE nspname='vec_documentos') IS DISTINCT FROM 'vec_documentos_propietario'::regrole
    OR to_regclass('vec_documentos.imagen_personal') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_interno' AND NOT rolcanlogin)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_externo' AND NOT rolcanlogin)
 THEN RAISE EXCEPTION 'documentos 000007: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_documentos.imagen_personal (
 imagen_ref text PRIMARY KEY CHECK(imagen_ref ~ '^docimg_[0-9a-f]{32}$'),
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 mime text NOT NULL CHECK(mime='image/jpeg'),
 tamano integer NOT NULL CHECK(tamano BETWEEN 4 AND 262144),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 -- NULL en cuanto se retira: los bytes no sobreviven a su uso.
 contenido bytea,
 estado text NOT NULL CHECK(estado IN ('activa','retirada')),
 operacion_alta_ref text NOT NULL CHECK(operacion_alta_ref ~ '^img_[0-9a-f]{32}$'),
 operacion_retirada_ref text CHECK(operacion_retirada_ref ~ '^img_[0-9a-f]{32}$'),
 custodiada_en timestamptz(6) NOT NULL,
 retirada_en timestamptz(6),
 CHECK((estado='activa' AND contenido IS NOT NULL AND octet_length(contenido)=tamano
        AND retirada_en IS NULL AND operacion_retirada_ref IS NULL)
    OR (estado='retirada' AND contenido IS NULL AND retirada_en IS NOT NULL
        AND operacion_retirada_ref IS NOT NULL AND retirada_en>=custodiada_en))
);
-- Minimización: como mucho una foto viva por persona.
CREATE UNIQUE INDEX imagen_personal_una_activa ON vec_documentos.imagen_personal(persona_ref) WHERE estado='activa';
ALTER TABLE vec_documentos.imagen_personal ALTER COLUMN contenido SET STORAGE EXTERNAL;

-- Historia de solo adición: qué pasó con cada foto, sin sus bytes.
CREATE TABLE vec_documentos.imagen_personal_historia (
 imagen_ref text NOT NULL REFERENCES vec_documentos.imagen_personal(imagen_ref),
 evento text NOT NULL CHECK(evento IN ('custodiada','retirada')),
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 tamano integer NOT NULL,
 operacion_ref text NOT NULL CHECK(operacion_ref ~ '^img_[0-9a-f]{32}$'),
 sesion text NOT NULL,
 registrado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(imagen_ref,evento)
);

-- Fila viva: la única transición es activa→retirada, borrando el contenido.
CREATE FUNCTION vec_documentos.imagen_personal_transicion_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_OP<>'UPDATE' THEN RAISE EXCEPTION 'documentos: imagen inmutable' USING ERRCODE='55000'; END IF;
 IF OLD.estado<>'activa' OR NEW.estado<>'retirada' OR NEW.contenido IS NOT NULL
    OR (to_jsonb(NEW)-'estado'-'contenido'-'retirada_en'-'operacion_retirada_ref')
       IS DISTINCT FROM (to_jsonb(OLD)-'estado'-'contenido'-'retirada_en'-'operacion_retirada_ref')
 THEN RAISE EXCEPTION 'documentos: transición de imagen inválida' USING ERRCODE='55000'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.imagen_personal_transicion_v1() FROM PUBLIC;
CREATE TRIGGER imagen_personal_transicion BEFORE UPDATE OR DELETE ON vec_documentos.imagen_personal
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.imagen_personal_transicion_v1();
CREATE FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'documentos: historia de imagen inmutable' USING ERRCODE='55000'; END $f$;
REVOKE ALL ON FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1() FROM PUBLIC;
CREATE TRIGGER imagen_personal_historia_inmutable BEFORE UPDATE OR DELETE ON vec_documentos.imagen_personal_historia
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1();
CREATE TRIGGER imagen_personal_no_truncar BEFORE TRUNCATE ON vec_documentos.imagen_personal
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1();
CREATE TRIGGER imagen_personal_historia_no_truncar BEFORE TRUNCATE ON vec_documentos.imagen_personal_historia
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.imagen_personal_historia_inmutable_v1();

ALTER TABLE vec_documentos.imagen_personal ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.imagen_personal FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.imagen_personal_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.imagen_personal_historia FORCE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE vec_documentos.imagen_personal, vec_documentos.imagen_personal_historia FROM PUBLIC;
-- Solo las funciones de esta migración (propietario) tocan las filas, y solo
-- cuando las invoca una fachada de Usuarios: el LOGIN de la sesión debe ser
-- un ejecutor técnico de Usuarios.
CREATE FUNCTION vec_documentos.sesion_usuarios_imagen_v1()
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT current_user='vec_documentos_propietario' AND session_user<>current_user
  AND (pg_has_role(session_user,'vec_usuarios_ejecutor_interno','MEMBER')
    OR pg_has_role(session_user,'vec_usuarios_ejecutor_externo','MEMBER'))
  AND NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND (rolsuper OR rolbypassrls))
$f$;
REVOKE ALL ON FUNCTION vec_documentos.sesion_usuarios_imagen_v1() FROM PUBLIC;
CREATE POLICY imagen_lectura ON vec_documentos.imagen_personal FOR SELECT TO vec_documentos_propietario
 USING (vec_documentos.sesion_usuarios_imagen_v1());
CREATE POLICY imagen_alta ON vec_documentos.imagen_personal FOR INSERT TO vec_documentos_propietario
 WITH CHECK (vec_documentos.sesion_usuarios_imagen_v1());
CREATE POLICY imagen_retirada ON vec_documentos.imagen_personal FOR UPDATE TO vec_documentos_propietario
 USING (vec_documentos.sesion_usuarios_imagen_v1()) WITH CHECK (vec_documentos.sesion_usuarios_imagen_v1());
CREATE POLICY imagen_historia_alta ON vec_documentos.imagen_personal_historia FOR INSERT TO vec_documentos_propietario
 WITH CHECK (vec_documentos.sesion_usuarios_imagen_v1());

-- Custodia un JPEG recodificado y devuelve su referencia opaca. Comprueba
-- huella, tamaño y firma JPEG; rechaza una segunda foto viva de la persona.
CREATE FUNCTION vec_documentos.custodiar_imagen_personal_v1(
 p_persona text,p_contenido bytea,p_sha256 text,p_operacion text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE ref text; ahora timestamptz(6); n integer;
BEGIN
 IF NOT vec_documentos.sesion_usuarios_imagen_v1()
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
 IF EXISTS(SELECT 1 FROM vec_documentos.imagen_personal WHERE persona_ref=p_persona AND estado='activa')
 THEN RAISE EXCEPTION 'documentos: la persona ya tiene una imagen activa' USING ERRCODE='P1409'; END IF;
 ref:='docimg_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 INSERT INTO vec_documentos.imagen_personal(imagen_ref,persona_ref,mime,tamano,huella_sha256,contenido,estado,operacion_alta_ref,custodiada_en)
 VALUES(ref,p_persona,'image/jpeg',n,p_sha256,p_contenido,'activa',p_operacion,ahora);
 INSERT INTO vec_documentos.imagen_personal_historia(imagen_ref,evento,persona_ref,huella_sha256,tamano,operacion_ref,sesion,registrado_en)
 VALUES(ref,'custodiada',p_persona,p_sha256,n,p_operacion,session_user,ahora);
 RETURN ref;
END $f$;

-- Retira la foto viva de la persona: borra los bytes y anota la historia.
CREATE FUNCTION vec_documentos.retirar_imagen_personal_v1(p_persona text,p_imagen text,p_operacion text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE i record; ahora timestamptz(6);
BEGIN
 IF NOT vec_documentos.sesion_usuarios_imagen_v1()
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: retirada de imagen denegada' USING ERRCODE='42501'; END IF;
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_imagen IS NULL OR p_imagen !~ '^docimg_[0-9a-f]{32}$'
    OR p_operacion IS NULL OR p_operacion !~ '^img_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'documentos: retirada de imagen inválida' USING ERRCODE='22023'; END IF;
 SELECT imagen_ref,huella_sha256,tamano INTO i FROM vec_documentos.imagen_personal
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND estado='activa' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'documentos: imagen ausente' USING ERRCODE='P1409'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 UPDATE vec_documentos.imagen_personal SET estado='retirada',contenido=NULL,retirada_en=ahora,operacion_retirada_ref=p_operacion
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND estado='activa';
 INSERT INTO vec_documentos.imagen_personal_historia(imagen_ref,evento,persona_ref,huella_sha256,tamano,operacion_ref,sesion,registrado_en)
 VALUES(p_imagen,'retirada',p_persona,i.huella_sha256,i.tamano,p_operacion,session_user,ahora);
END $f$;

-- Devuelve los bytes de la foto viva solo a la persona titular (la fachada de
-- Usuarios ya consumió su V3). Una referencia retirada o ajena no devuelve nada.
CREATE FUNCTION vec_documentos.abrir_imagen_personal_v1(p_persona text,p_imagen text)
RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE b bytea; h text;
BEGIN
 IF NOT vec_documentos.sesion_usuarios_imagen_v1()
    OR current_setting('transaction_isolation')<>'serializable'
 THEN RAISE EXCEPTION 'documentos: lectura de imagen denegada' USING ERRCODE='42501'; END IF;
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_imagen IS NULL OR p_imagen !~ '^docimg_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'documentos: lectura de imagen inválida' USING ERRCODE='22023'; END IF;
 SELECT contenido,huella_sha256 INTO b,h FROM vec_documentos.imagen_personal
 WHERE imagen_ref=p_imagen AND persona_ref=p_persona AND estado='activa';
 IF b IS NULL THEN RETURN NULL; END IF;
 IF encode(sha256(b),'hex')<>h THEN RAISE EXCEPTION 'documentos: imagen alterada' USING ERRCODE='55000'; END IF;
 RETURN b;
END $f$;

REVOKE ALL ON FUNCTION vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_documentos.retirar_imagen_personal_v1(text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_documentos.abrir_imagen_personal_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_documentos TO vec_usuarios_propietario;
GRANT EXECUTE ON FUNCTION vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text) TO vec_usuarios_propietario;
GRANT EXECUTE ON FUNCTION vec_documentos.retirar_imagen_personal_v1(text,text,text) TO vec_usuarios_propietario;
GRANT EXECUTE ON FUNCTION vec_documentos.abrir_imagen_personal_v1(text,text) TO vec_usuarios_propietario;

-- Usuarios solo alcanza estas tres fachadas: ninguna tabla, secuencia ni otra
-- función del esquema documental; los ejecutores de Usuarios, nada.
DO $acl$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_documentos'::regnamespace AND c.relkind IN ('r','v','m','S','p')
     AND (has_table_privilege('vec_usuarios_propietario',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR has_table_privilege('vec_usuarios_ejecutor_interno',c.oid,'SELECT,INSERT,UPDATE,DELETE')
       OR has_table_privilege('vec_usuarios_ejecutor_externo',c.oid,'SELECT,INSERT,UPDATE,DELETE')))
    OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_documentos'::regnamespace
     AND has_function_privilege('vec_usuarios_propietario',p.oid,'EXECUTE')
     AND p.proname NOT IN ('custodiar_imagen_personal_v1','retirar_imagen_personal_v1','abrir_imagen_personal_v1'))
    OR EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_documentos'::regnamespace
     AND (has_function_privilege('vec_usuarios_ejecutor_interno',p.oid,'EXECUTE')
       OR has_function_privilege('vec_usuarios_ejecutor_externo',p.oid,'EXECUTE')))
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_documentos' AND tablename LIKE 'imagen_personal%'
     AND roles<>ARRAY['vec_documentos_propietario']::name[])
 THEN RAISE EXCEPTION 'documentos 000007: ACL de imagen incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
