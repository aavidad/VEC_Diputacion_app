\set ON_ERROR_STOP on
-- Documentos 000008 (Fase 1, paso 3 del estudio de datos personales): la foto
-- personal deja de compartirse entre el portal interno (RRHH) y el externo
-- (Área personal). Cada foto viva pertenece a la superficie cuyo LOGIN técnico
-- la custodió; como mucho hay una viva por persona y superficie, y ninguna
-- función devuelve, retira ni cuenta la foto de la otra superficie.
--
-- Las firmas de las tres funciones no cambian: la superficie sale del LOGIN
-- de la sesión (miembro exclusivo de vec_usuarios_ejecutor_interno o
-- vec_usuarios_ejecutor_externo), nunca de un parámetro.
--
-- Filas existentes: su superficie se deduce del LOGIN que anotó la historia
-- «custodiada». Si ese LOGIN ya no existe o no pertenece a un único ejecutor,
-- la migración se detiene (55000) sin cambiar nada. Una sola transacción:
-- cualquier fallo la revierte entera. Aplicar como DBA, después de Documentos
-- 000007 y antes de Usuarios 000008.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000008',0));
DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_documentos.imagen_personal') IS NULL
    OR to_regclass('vec_documentos.imagen_personal_historia') IS NULL
    OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='vec_documentos.imagen_personal'::regclass
      AND attname='superficie' AND NOT attisdropped)
    OR to_regprocedure('vec_documentos.superficie_sesion_imagen_v1()') IS NOT NULL
    OR to_regprocedure('vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text)') IS NULL
    OR to_regprocedure('vec_documentos.retirar_imagen_personal_v1(text,text,text)') IS NULL
    OR to_regprocedure('vec_documentos.abrir_imagen_personal_v1(text,text)') IS NULL
    OR to_regclass('vec_documentos.imagen_personal_una_activa') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_interno' AND NOT rolcanlogin)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_externo' AND NOT rolcanlogin)
 THEN RAISE EXCEPTION 'documentos 000008: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_documentos.imagen_personal, vec_documentos.imagen_personal_historia IN ACCESS EXCLUSIVE MODE;

-- Superficie de cada foto según el LOGIN que la custodió.
CREATE TEMP TABLE documentos_000008_superficie ON COMMIT DROP AS
SELECT i.imagen_ref,
 CASE WHEN r.oid IS NULL THEN NULL
  WHEN pg_has_role(r.oid,'vec_usuarios_ejecutor_interno','MEMBER') AND NOT pg_has_role(r.oid,'vec_usuarios_ejecutor_externo','MEMBER') THEN 'interna_corporativa'
  WHEN pg_has_role(r.oid,'vec_usuarios_ejecutor_externo','MEMBER') AND NOT pg_has_role(r.oid,'vec_usuarios_ejecutor_interno','MEMBER') THEN 'externa_personal'
 END AS superficie
FROM vec_documentos.imagen_personal i
LEFT JOIN vec_documentos.imagen_personal_historia h ON h.imagen_ref=i.imagen_ref AND h.evento='custodiada'
LEFT JOIN pg_roles r ON r.rolname=h.sesion;
DO $mapa$ BEGIN
 IF EXISTS (SELECT 1 FROM documentos_000008_superficie WHERE superficie IS NULL)
    OR (SELECT count(*) FROM documentos_000008_superficie)<>(SELECT count(*) FROM vec_documentos.imagen_personal)
 THEN RAISE EXCEPTION 'documentos 000008: foto sin superficie deducible; revisar antes de migrar' USING ERRCODE='55000'; END IF;
END $mapa$;

-- Copia de control: solo cambia la columna nueva.
CREATE TEMP TABLE documentos_000008_antes ON COMMIT DROP AS
SELECT imagen_ref,to_jsonb(i) AS fila FROM vec_documentos.imagen_personal i;

ALTER TABLE vec_documentos.imagen_personal ADD COLUMN superficie text;
ALTER TABLE vec_documentos.imagen_personal DISABLE TRIGGER imagen_personal_transicion;
UPDATE vec_documentos.imagen_personal i SET superficie=m.superficie
FROM documentos_000008_superficie m WHERE m.imagen_ref=i.imagen_ref;
ALTER TABLE vec_documentos.imagen_personal ENABLE TRIGGER imagen_personal_transicion;
ALTER TABLE vec_documentos.imagen_personal ALTER COLUMN superficie SET NOT NULL;
ALTER TABLE vec_documentos.imagen_personal ADD CONSTRAINT imagen_personal_superficie_check
 CHECK(superficie IN ('interna_corporativa','externa_personal'));
DO $control$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_documentos.imagen_personal i FULL JOIN documentos_000008_antes a USING(imagen_ref)
     WHERE a.fila IS DISTINCT FROM (to_jsonb(i)-'superficie'))
 THEN RAISE EXCEPTION 'documentos 000008: la migración alteraría una foto' USING ERRCODE='55000'; END IF;
END $control$;
-- Minimización: como mucho una foto viva por persona y superficie.
DROP INDEX vec_documentos.imagen_personal_una_activa;
CREATE UNIQUE INDEX imagen_personal_una_activa_superficie ON vec_documentos.imagen_personal(persona_ref,superficie) WHERE estado='activa';

SET LOCAL ROLE vec_documentos_propietario;
-- La superficie queda congelada como el resto de columnas de identidad.
CREATE OR REPLACE FUNCTION vec_documentos.imagen_personal_transicion_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_OP<>'UPDATE' THEN RAISE EXCEPTION 'documentos: imagen inmutable' USING ERRCODE='55000'; END IF;
 -- Columnas fijas comparadas una a una: no se serializa el contenido.
 IF OLD.estado<>'activa' OR NEW.estado<>'retirada' OR NEW.contenido IS NOT NULL
    OR (NEW.imagen_ref,NEW.persona_ref,NEW.superficie,NEW.mime,NEW.tamano,NEW.huella_sha256,NEW.operacion_alta_ref,NEW.custodiada_en)
       IS DISTINCT FROM (OLD.imagen_ref,OLD.persona_ref,OLD.superficie,OLD.mime,OLD.tamano,OLD.huella_sha256,OLD.operacion_alta_ref,OLD.custodiada_en)
 THEN RAISE EXCEPTION 'documentos: transición de imagen inválida' USING ERRCODE='55000'; END IF;
 RETURN NEW;
END $f$;

-- Superficie del LOGIN técnico: miembro exclusivo de uno de los dos
-- ejecutores de Usuarios. Cualquier otra sesión devuelve NULL.
CREATE FUNCTION vec_documentos.superficie_sesion_imagen_v1()
RETURNS text LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
 SELECT CASE
  WHEN current_user<>'vec_documentos_propietario' OR session_user=current_user
    OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND (rolsuper OR rolbypassrls))
  THEN NULL
  WHEN pg_has_role(session_user,'vec_usuarios_ejecutor_interno','MEMBER')
   AND NOT pg_has_role(session_user,'vec_usuarios_ejecutor_externo','MEMBER') THEN 'interna_corporativa'
  WHEN pg_has_role(session_user,'vec_usuarios_ejecutor_externo','MEMBER')
   AND NOT pg_has_role(session_user,'vec_usuarios_ejecutor_interno','MEMBER') THEN 'externa_personal'
 END
$f$;
REVOKE ALL ON FUNCTION vec_documentos.superficie_sesion_imagen_v1() FROM PUBLIC;

-- Las políticas exigen además que la fila sea de la superficie de la sesión.
DROP POLICY imagen_lectura ON vec_documentos.imagen_personal;
DROP POLICY imagen_alta ON vec_documentos.imagen_personal;
DROP POLICY imagen_retirada ON vec_documentos.imagen_personal;
CREATE POLICY imagen_lectura ON vec_documentos.imagen_personal FOR SELECT TO vec_documentos_propietario
 USING (vec_documentos.sesion_usuarios_imagen_v1() AND superficie=vec_documentos.superficie_sesion_imagen_v1());
CREATE POLICY imagen_alta ON vec_documentos.imagen_personal FOR INSERT TO vec_documentos_propietario
 WITH CHECK (vec_documentos.sesion_usuarios_imagen_v1() AND superficie=vec_documentos.superficie_sesion_imagen_v1());
CREATE POLICY imagen_retirada ON vec_documentos.imagen_personal FOR UPDATE TO vec_documentos_propietario
 USING (vec_documentos.sesion_usuarios_imagen_v1() AND superficie=vec_documentos.superficie_sesion_imagen_v1())
 WITH CHECK (vec_documentos.sesion_usuarios_imagen_v1() AND superficie=vec_documentos.superficie_sesion_imagen_v1());

CREATE OR REPLACE FUNCTION vec_documentos.custodiar_imagen_personal_v1(
 p_persona text,p_contenido bytea,p_sha256 text,p_operacion text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE ref text; ahora timestamptz(6); n integer; s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS NULL
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

CREATE OR REPLACE FUNCTION vec_documentos.retirar_imagen_personal_v1(p_persona text,p_imagen text,p_operacion text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE i record; ahora timestamptz(6); s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS NULL
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

CREATE OR REPLACE FUNCTION vec_documentos.abrir_imagen_personal_v1(p_persona text,p_imagen text)
RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE b bytea; h text; s text;
BEGIN
 s:=vec_documentos.superficie_sesion_imagen_v1();
 IF NOT vec_documentos.sesion_usuarios_imagen_v1() OR s IS NULL
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
RESET ROLE;

-- Las fachadas conservan propietario y ACL; Usuarios sigue sin ver tablas.
DO $acl$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_documentos'::regnamespace
     AND p.proname IN ('custodiar_imagen_personal_v1','retirar_imagen_personal_v1','abrir_imagen_personal_v1','superficie_sesion_imagen_v1','imagen_personal_transicion_v1')
     AND (p.proowner<>'vec_documentos_propietario'::regrole
       OR has_function_privilege('vec_usuarios_ejecutor_interno',p.oid,'EXECUTE')
       OR has_function_privilege('vec_usuarios_ejecutor_externo',p.oid,'EXECUTE')))
    OR has_function_privilege('vec_usuarios_propietario','vec_documentos.superficie_sesion_imagen_v1()','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_documentos.retirar_imagen_personal_v1(text,text,text)','EXECUTE')
    OR NOT has_function_privilege('vec_usuarios_propietario','vec_documentos.abrir_imagen_personal_v1(text,text)','EXECUTE')
    OR has_table_privilege('vec_usuarios_propietario','vec_documentos.imagen_personal','SELECT,INSERT,UPDATE,DELETE')
    OR EXISTS(SELECT 1 FROM pg_policies WHERE schemaname='vec_documentos' AND tablename LIKE 'imagen_personal%'
     AND roles<>ARRAY['vec_documentos_propietario']::name[])
    OR (SELECT count(*) FROM pg_policies WHERE schemaname='vec_documentos' AND tablename='imagen_personal')<>3
 THEN RAISE EXCEPTION 'documentos 000008: ACL de imagen incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
