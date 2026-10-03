\set ON_ERROR_STOP on
-- Usuarios 000011: estado, historia y recibos de imagen externos en tablas propias.
-- Instalar tras Documentos 000010 y Usuarios 000009/000010, con servicio detenido.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='120s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000011',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regclass('vec_usuarios.imagen_actual') IS NULL
    OR to_regclass('vec_usuarios.imagen_historia') IS NULL
    OR to_regclass('vec_usuarios.imagen_recibo') IS NULL
    OR to_regclass('vec_usuarios.imagen_actual_externa') IS NOT NULL
    OR to_regclass('vec_documentos.imagen_personal_externa') IS NULL
 THEN RAISE EXCEPTION 'Usuarios 000011: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_usuarios.imagen_actual,vec_usuarios.imagen_historia,vec_usuarios.imagen_recibo,
 vec_usuarios.imagen_contexto IN ACCESS EXCLUSIVE MODE;
DO $vivo$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto)
 THEN RAISE EXCEPTION 'Usuarios 000011: contexto de imagen abierto' USING ERRCODE='55000'; END IF;
END $vivo$;
CREATE TEMP TABLE u11_antes ON COMMIT DROP AS
 SELECT 'actual'::text tipo,persona_ref,superficie,version,NULL::text clave,to_jsonb(t) fila
 FROM vec_usuarios.imagen_actual t WHERE superficie='externa_personal'
 UNION ALL SELECT 'historia',persona_ref,superficie,version,NULL::text,to_jsonb(t)
 FROM vec_usuarios.imagen_historia t WHERE superficie='externa_personal'
 UNION ALL SELECT 'recibo',persona_ref,superficie,version,clave_operacion,to_jsonb(t)
 FROM vec_usuarios.imagen_recibo t WHERE superficie='externa_personal';
CREATE TABLE vec_usuarios.imagen_actual_externa (LIKE vec_usuarios.imagen_actual INCLUDING ALL);
CREATE TABLE vec_usuarios.imagen_historia_externa (LIKE vec_usuarios.imagen_historia INCLUDING ALL);
CREATE TABLE vec_usuarios.imagen_recibo_externa (LIKE vec_usuarios.imagen_recibo INCLUDING ALL);
ALTER TABLE vec_usuarios.imagen_actual_externa ADD CONSTRAINT imagen_actual_externa_superficie CHECK(superficie='externa_personal');
ALTER TABLE vec_usuarios.imagen_historia_externa ADD CONSTRAINT imagen_historia_externa_superficie CHECK(superficie='externa_personal');
ALTER TABLE vec_usuarios.imagen_recibo_externa ADD CONSTRAINT imagen_recibo_externa_superficie CHECK(superficie='externa_personal');
ALTER TABLE vec_usuarios.imagen_actual_externa ADD CONSTRAINT imagen_actual_externa_catalogo_fkey FOREIGN KEY(catalogo_version_ref) REFERENCES vec_usuarios.catalogo_imagen(version_ref);
ALTER TABLE vec_usuarios.imagen_historia_externa ADD CONSTRAINT imagen_historia_externa_catalogo_fkey FOREIGN KEY(catalogo_version_ref) REFERENCES vec_usuarios.catalogo_imagen(version_ref);
ALTER TABLE vec_usuarios.imagen_recibo_externa ADD CONSTRAINT imagen_recibo_externa_catalogo_fkey FOREIGN KEY(catalogo_version_ref) REFERENCES vec_usuarios.catalogo_imagen(version_ref);
ALTER TABLE vec_usuarios.imagen_historia_externa ADD CONSTRAINT imagen_historia_externa_actual_fkey FOREIGN KEY(persona_ref,superficie)
 REFERENCES vec_usuarios.imagen_actual_externa(persona_ref,superficie) DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_usuarios.imagen_recibo_externa ADD CONSTRAINT imagen_recibo_externa_historia_fkey FOREIGN KEY(persona_ref,superficie,version)
 REFERENCES vec_usuarios.imagen_historia_externa(persona_ref,superficie,version) DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_usuarios.imagen_recibo DISABLE TRIGGER imagen_recibo_inmutable;
ALTER TABLE vec_usuarios.imagen_historia DISABLE TRIGGER imagen_historia_inmutable;
INSERT INTO vec_usuarios.imagen_actual_externa SELECT * FROM vec_usuarios.imagen_actual WHERE superficie='externa_personal';
INSERT INTO vec_usuarios.imagen_historia_externa SELECT * FROM vec_usuarios.imagen_historia WHERE superficie='externa_personal';
INSERT INTO vec_usuarios.imagen_recibo_externa SELECT * FROM vec_usuarios.imagen_recibo WHERE superficie='externa_personal';
DELETE FROM vec_usuarios.imagen_recibo WHERE superficie='externa_personal';
DELETE FROM vec_usuarios.imagen_historia WHERE superficie='externa_personal';
SET CONSTRAINTS ALL IMMEDIATE;
ALTER TABLE vec_usuarios.imagen_recibo ENABLE TRIGGER imagen_recibo_inmutable;
ALTER TABLE vec_usuarios.imagen_historia ENABLE TRIGGER imagen_historia_inmutable;
DELETE FROM vec_usuarios.imagen_actual WHERE superficie='externa_personal';
DO $control$ BEGIN
 IF EXISTS(SELECT 1 FROM u11_antes a FULL JOIN (
    SELECT 'actual'::text tipo,persona_ref,superficie,version,NULL::text clave,to_jsonb(t) fila FROM vec_usuarios.imagen_actual_externa t
    UNION ALL SELECT 'historia',persona_ref,superficie,version,NULL::text,to_jsonb(t) FROM vec_usuarios.imagen_historia_externa t
    UNION ALL SELECT 'recibo',persona_ref,superficie,version,clave_operacion,to_jsonb(t) FROM vec_usuarios.imagen_recibo_externa t
    ) n ON a.tipo=n.tipo AND a.persona_ref=n.persona_ref AND a.superficie=n.superficie
      AND a.version=n.version AND a.clave IS NOT DISTINCT FROM n.clave WHERE a.fila IS DISTINCT FROM n.fila)
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_actual WHERE superficie<>'interna_corporativa')
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_historia WHERE superficie<>'interna_corporativa')
    OR EXISTS(SELECT 1 FROM vec_usuarios.imagen_recibo WHERE superficie<>'interna_corporativa')
 THEN RAISE EXCEPTION 'Usuarios 000011: copia incompatible' USING ERRCODE='55000'; END IF;
END $control$;
ALTER TABLE vec_usuarios.imagen_actual ADD CONSTRAINT imagen_actual_solo_interna CHECK(superficie='interna_corporativa');
ALTER TABLE vec_usuarios.imagen_historia ADD CONSTRAINT imagen_historia_solo_interna CHECK(superficie='interna_corporativa');
ALTER TABLE vec_usuarios.imagen_recibo ADD CONSTRAINT imagen_recibo_solo_interna CHECK(superficie='interna_corporativa');
CREATE TABLE vec_usuarios.imagen_contexto_externa (LIKE vec_usuarios.imagen_contexto INCLUDING ALL);
ALTER TABLE vec_usuarios.imagen_contexto ADD CONSTRAINT imagen_contexto_solo_interna CHECK(superficie='interna_corporativa');
ALTER TABLE vec_usuarios.imagen_contexto_externa ADD CONSTRAINT imagen_contexto_externa_superficie CHECK(superficie='externa_personal');
ALTER TABLE vec_usuarios.imagen_contexto_externa OWNER TO vec_usuarios_propietario;
REVOKE ALL ON vec_usuarios.imagen_contexto_externa FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
REVOKE ALL ON TYPE vec_usuarios.imagen_contexto_externa FROM PUBLIC;
ALTER TABLE vec_usuarios.imagen_contexto_externa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_usuarios.imagen_contexto_externa FORCE ROW LEVEL SECURITY;
CREATE TRIGGER imagen_contexto_externa_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_contexto_externa
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
DO $owner$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['imagen_actual_externa','imagen_historia_externa','imagen_recibo_externa'] LOOP
  EXECUTE format('ALTER TABLE vec_usuarios.%I OWNER TO vec_usuarios_propietario',t);
  EXECUTE format('REVOKE ALL ON vec_usuarios.%I FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_usuarios.%I FROM PUBLIC',t);
  EXECUTE format('ALTER TABLE vec_usuarios.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_usuarios.%I FORCE ROW LEVEL SECURITY',t);
 END LOOP;
END $owner$;
CREATE TRIGGER imagen_actual_externa_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_actual_externa FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_historia_externa_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.imagen_historia_externa FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_historia_externa_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_historia_externa FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_recibo_externa_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.imagen_recibo_externa FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER imagen_recibo_externa_no_truncar BEFORE TRUNCATE ON vec_usuarios.imagen_recibo_externa FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
SET LOCAL ROLE vec_usuarios_propietario;
CREATE POLICY imagen_contexto_externa_lectura ON vec_usuarios.imagen_contexto_externa FOR SELECT TO vec_usuarios_propietario
 USING(backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY imagen_contexto_externa_alta ON vec_usuarios.imagen_contexto_externa FOR INSERT TO vec_usuarios_propietario
 WITH CHECK(xid=pg_current_xact_id_if_assigned() AND backend_pid=pg_backend_pid() AND sesion=session_user
  AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE POLICY imagen_contexto_externa_baja ON vec_usuarios.imagen_contexto_externa FOR DELETE TO vec_usuarios_propietario
 USING(backend_pid=pg_backend_pid() AND sesion=session_user AND superficie=vec_usuarios.superficie_sesion_correos());
CREATE OR REPLACE FUNCTION vec_usuarios.contexto_autorizado_imagen_superficie(p_persona text,p_superficie text,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT CASE WHEN p_superficie='externa_personal' THEN EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto_externa c
   WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid() AND c.sesion=session_user
   AND c.superficie=vec_usuarios.superficie_sesion_correos() AND c.persona_ref=p_persona
   AND c.superficie=p_superficie AND c.modo=ANY(p_modos))
  WHEN p_superficie='interna_corporativa' THEN EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto c
   WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid() AND c.sesion=session_user
   AND c.superficie=vec_usuarios.superficie_sesion_correos() AND c.persona_ref=p_persona
   AND c.superficie=p_superficie AND c.modo=ANY(p_modos))
  ELSE false END
$f$;
CREATE POLICY imagen_actual_externa_lectura ON vec_usuarios.imagen_actual_externa FOR SELECT TO vec_usuarios_propietario
 USING(vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['consultar','actualizar']));
CREATE POLICY imagen_actual_externa_alta ON vec_usuarios.imagen_actual_externa FOR INSERT TO vec_usuarios_propietario
 WITH CHECK(vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY imagen_actual_externa_cambio ON vec_usuarios.imagen_actual_externa FOR UPDATE TO vec_usuarios_propietario
 USING(vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']))
 WITH CHECK(vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY imagen_historia_externa_alta ON vec_usuarios.imagen_historia_externa FOR INSERT TO vec_usuarios_propietario
 WITH CHECK(vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));
CREATE POLICY imagen_recibo_externa_lectura ON vec_usuarios.imagen_recibo_externa FOR SELECT TO vec_usuarios_propietario
 USING(vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['recuperar','actualizar']));
CREATE POLICY imagen_recibo_externa_alta ON vec_usuarios.imagen_recibo_externa FOR INSERT TO vec_usuarios_propietario
 WITH CHECK(vec_usuarios.contexto_autorizado_imagen_superficie(persona_ref,superficie,ARRAY['actualizar']));


CREATE OR REPLACE FUNCTION vec_usuarios.consumir_contexto_imagen(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_modo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE m jsonb; x record; xid_actual xid8;
BEGIN
 m:=vec_usuarios.validar_material_imagen(p_material,p_capacidad,p_decision,p_persona_version,p_perfil_version);
 IF p_modo NOT IN ('consultar','recuperar','actualizar')
    OR (p_modo='consultar')<>(m->>'accion'='vec.imagen.consultar')
 THEN RAISE EXCEPTION 'Usuarios: modo de imagen inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(
  m->>'accion',m->>'superficie',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM m->>'persona_ref'
    OR x.decision_ref IS DISTINCT FROM convert_from(p_decision,'UTF8')::jsonb->>'decision_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM vec_usuarios.huella_contexto_imagen(p_material)
    OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'Usuarios: consumo de imagen divergente' USING ERRCODE='42501'; END IF;
 xid_actual:=pg_current_xact_id();
 IF m->>'superficie'='externa_personal' THEN
  DELETE FROM vec_usuarios.imagen_contexto_externa WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
  IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto_externa WHERE xid=xid_actual AND backend_pid=pg_backend_pid() AND sesion=session_user)
  THEN RAISE EXCEPTION 'Usuarios: contexto de imagen ya activo' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_usuarios.imagen_contexto_externa(xid,backend_pid,sesion,superficie,persona_ref,modo,decision_ref,auditoria_ref,consumo_huella_sha256)
  VALUES(xid_actual,pg_backend_pid(),session_user,m->>'superficie',m->>'persona_ref',p_modo,x.decision_ref,x.auditoria_ref,x.consumo_huella_sha256);
 ELSIF m->>'superficie'='interna_corporativa' THEN
  DELETE FROM vec_usuarios.imagen_contexto WHERE backend_pid=pg_backend_pid() AND xid<>xid_actual;
  IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_contexto WHERE xid=xid_actual AND backend_pid=pg_backend_pid() AND sesion=session_user)
  THEN RAISE EXCEPTION 'Usuarios: contexto de imagen ya activo' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_usuarios.imagen_contexto(xid,backend_pid,sesion,superficie,persona_ref,modo,decision_ref,auditoria_ref,consumo_huella_sha256)
  VALUES(xid_actual,pg_backend_pid(),session_user,m->>'superficie',m->>'persona_ref',p_modo,x.decision_ref,x.auditoria_ref,x.consumo_huella_sha256);
 ELSE RAISE EXCEPTION 'Usuarios: superficie de imagen denegada' USING ERRCODE='42501'; END IF;
 RETURN m;
END $f$;

CREATE OR REPLACE FUNCTION vec_usuarios.retirar_contexto_imagen(p_persona text,p_modo text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE n integer;
BEGIN
 IF vec_usuarios.superficie_sesion_correos()='externa_personal' THEN
 DELETE FROM vec_usuarios.imagen_contexto_externa WHERE xid=pg_current_xact_id_if_assigned()
 AND backend_pid=pg_backend_pid() AND sesion=session_user AND persona_ref=p_persona AND modo=p_modo;
 ELSE
 DELETE FROM vec_usuarios.imagen_contexto WHERE xid=pg_current_xact_id_if_assigned()
 AND backend_pid=pg_backend_pid() AND sesion=session_user AND persona_ref=p_persona AND modo=p_modo;
 END IF;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION 'Usuarios: contexto de imagen incompleto' USING ERRCODE='42501'; END IF;
END $f$;

CREATE FUNCTION vec_usuarios.recibo_imagen_json(r vec_usuarios.imagen_recibo_externa,p_replay boolean)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('recibo_ref',r.recibo_ref,'persona_ref',r.persona_ref,'version',r.version,
  'catalogo_version_ref',r.catalogo_version_ref,'eleccion',r.eleccion,'foto_nueva',r.foto_nueva,
  'foto_retirada',r.foto_retirada,'fecha_utc',r.registrada_en,'replay',p_replay)
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.recibo_imagen_json(vec_usuarios.imagen_recibo_externa,boolean) FROM PUBLIC;


CREATE OR REPLACE FUNCTION vec_usuarios.consultar_imagen_propia_v1_interna(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(datos jsonb,foto bytea) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; e record; cat jsonb; b bytea;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'consultar');
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 SELECT version,catalogo_version_ref,eleccion,foto_ref INTO e FROM vec_usuarios.imagen_actual
 WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie';
 IF FOUND THEN
  IF e.foto_ref IS NOT NULL THEN b:=vec_documentos.abrir_imagen_personal_v1(m->>'persona_ref',e.foto_ref); END IF;
  datos:=jsonb_build_object('existe',true,'persona_ref',m->>'persona_ref','version',e.version,
   'catalogo_version_ref',e.catalogo_version_ref,'eleccion',e.eleccion);
 ELSE
  datos:=jsonb_build_object('existe',false,'persona_ref',m->>'persona_ref','version',0,
   'catalogo_version_ref',cat->>'version_ref','eleccion',cat->'predeterminada');
 END IF;
 foto:=b;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','consultar');
 RETURN NEXT;
END $f$;

REVOKE ALL ON FUNCTION vec_usuarios.consultar_imagen_propia_v1_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_usuarios.consultar_imagen_propia_v1_externa(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(datos jsonb,foto bytea) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; e record; cat jsonb; b bytea;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'consultar');
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 SELECT version,catalogo_version_ref,eleccion,foto_ref INTO e FROM vec_usuarios.imagen_actual_externa
 WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie';
 IF FOUND THEN
  IF e.foto_ref IS NOT NULL THEN b:=vec_documentos.abrir_imagen_personal_v1(m->>'persona_ref',e.foto_ref); END IF;
  datos:=jsonb_build_object('existe',true,'persona_ref',m->>'persona_ref','version',e.version,
   'catalogo_version_ref',e.catalogo_version_ref,'eleccion',e.eleccion);
 ELSE
  datos:=jsonb_build_object('existe',false,'persona_ref',m->>'persona_ref','version',0,
   'catalogo_version_ref',cat->>'version_ref','eleccion',cat->'predeterminada');
 END IF;
 foto:=b;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','consultar');
 RETURN NEXT;
END $f$;

REVOKE ALL ON FUNCTION vec_usuarios.consultar_imagen_propia_v1_externa(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_usuarios.consultar_imagen_propia_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(datos jsonb,foto bytea) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF vec_usuarios.superficie_sesion_correos()='interna_corporativa' THEN
  RETURN QUERY SELECT * FROM vec_usuarios.consultar_imagen_propia_v1_interna(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSIF vec_usuarios.superficie_sesion_correos()='externa_personal' THEN
  RETURN QUERY SELECT * FROM vec_usuarios.consultar_imagen_propia_v1_externa(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE RAISE EXCEPTION 'Usuarios: superficie de imagen denegada' USING ERRCODE='42501'; END IF;
 RETURN;
END $f$;

CREATE OR REPLACE FUNCTION vec_usuarios.recuperar_imagen_operacion_v1_interna(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; r vec_usuarios.imagen_recibo;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'recuperar');
 SELECT * INTO r FROM vec_usuarios.imagen_recibo WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie'
  AND clave_operacion=m->>'clave_operacion';
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','recuperar');
 IF r.recibo_ref IS NULL THEN RETURN NULL; END IF;
 IF r.huella_peticion IS DISTINCT FROM m->>'huella_peticion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 RETURN vec_usuarios.recibo_imagen_json(r,true);
END $f$;

REVOKE ALL ON FUNCTION vec_usuarios.recuperar_imagen_operacion_v1_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_usuarios.recuperar_imagen_operacion_v1_externa(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; r vec_usuarios.imagen_recibo_externa;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'recuperar');
 SELECT * INTO r FROM vec_usuarios.imagen_recibo_externa WHERE persona_ref=m->>'persona_ref' AND superficie=m->>'superficie'
  AND clave_operacion=m->>'clave_operacion';
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','recuperar');
 IF r.recibo_ref IS NULL THEN RETURN NULL; END IF;
 IF r.huella_peticion IS DISTINCT FROM m->>'huella_peticion'
 THEN RAISE EXCEPTION 'Usuarios: clave reutilizada con otra petición' USING ERRCODE='P1409'; END IF;
 RETURN vec_usuarios.recibo_imagen_json(r,true);
END $f$;

REVOKE ALL ON FUNCTION vec_usuarios.recuperar_imagen_operacion_v1_externa(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_usuarios.recuperar_imagen_operacion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF vec_usuarios.superficie_sesion_correos()='interna_corporativa' THEN
  RETURN vec_usuarios.recuperar_imagen_operacion_v1_interna(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSIF vec_usuarios.superficie_sesion_correos()='externa_personal' THEN
  RETURN vec_usuarios.recuperar_imagen_operacion_v1_externa(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE RAISE EXCEPTION 'Usuarios: superficie de imagen denegada' USING ERRCODE='42501'; END IF;

END $f$;

CREATE OR REPLACE FUNCTION vec_usuarios.guardar_imagen_propia_v1_interna(
 p_material text,p_foto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; cx record; cat jsonb; previo record; v bigint; ref text; ahora timestamptz(6); n integer;
 v_foto_ref text; v_foto_sha text; v_retirada text; r vec_usuarios.imagen_recibo; s text;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'actualizar');
 s:=m->>'superficie';
 IF (m->>'foto_sha256'='')<>(p_foto IS NULL)
    OR (p_foto IS NOT NULL AND encode(sha256(p_foto),'hex') IS DISTINCT FROM m->>'foto_sha256')
 THEN RAISE EXCEPTION 'Usuarios: foto divergente del material' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:imagen:'||s||':'||(m->>'persona_ref'),0));
 IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_recibo WHERE persona_ref=m->>'persona_ref' AND superficie=s AND clave_operacion=m->>'clave_operacion')
 THEN RAISE EXCEPTION 'Usuarios: operación ya registrada; recuperar recibo' USING ERRCODE='40001'; END IF;
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'paletas') q WHERE q->>'codigo'=m#>>'{eleccion,paleta}')
    OR (m#>>'{eleccion,modo}'='icono' AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'iconos') q WHERE q->>'codigo'=m#>>'{eleccion,icono}'))
 THEN RAISE EXCEPTION 'Usuarios: imagen fuera de catálogo' USING ERRCODE='22023'; END IF;
 SELECT version,foto_ref,foto_sha256 INTO previo FROM vec_usuarios.imagen_actual WHERE persona_ref=m->>'persona_ref' AND superficie=s FOR UPDATE;
 IF coalesce(previo.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión de imagen en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(previo.version,0)+1;
 ref:='img_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 v_foto_ref:=previo.foto_ref; v_foto_sha:=previo.foto_sha256;
 IF m#>>'{eleccion,modo}'='foto' AND p_foto IS NULL AND previo.foto_ref IS NULL
 THEN RAISE EXCEPTION 'Usuarios: no hay foto que conservar' USING ERRCODE='P1409'; END IF;
 IF previo.foto_ref IS NOT NULL AND (p_foto IS NOT NULL OR m#>>'{eleccion,modo}'<>'foto') THEN
  PERFORM vec_documentos.retirar_imagen_personal_v1(m->>'persona_ref',previo.foto_ref,ref);
  v_retirada:=previo.foto_ref; v_foto_ref:=NULL; v_foto_sha:=NULL;
 END IF;
 IF p_foto IS NOT NULL THEN
  v_foto_ref:=vec_documentos.custodiar_imagen_personal_v1(m->>'persona_ref',p_foto,m->>'foto_sha256',ref);
  v_foto_sha:=m->>'foto_sha256';
 END IF;
 IF previo.version IS NULL THEN
  INSERT INTO vec_usuarios.imagen_actual(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,recibo_ref,actualizado_en)
  VALUES(m->>'persona_ref',s,v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,ref,ahora);
 ELSE
  UPDATE vec_usuarios.imagen_actual SET version=v,catalogo_version_ref=cat->>'version_ref',eleccion=m->'eleccion',
   foto_ref=v_foto_ref,foto_sha256=v_foto_sha,recibo_ref=ref,actualizado_en=ahora
  WHERE persona_ref=m->>'persona_ref' AND superficie=s AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión de imagen concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 SELECT * INTO STRICT cx FROM vec_usuarios.imagen_contexto WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo='actualizar' AND superficie=s;
 INSERT INTO vec_usuarios.imagen_historia(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,foto_retirada_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(m->>'persona_ref',s,v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,v_retirada,ref,cx.decision_ref,cx.auditoria_ref,ahora);
 INSERT INTO vec_usuarios.imagen_recibo(persona_ref,superficie,clave_operacion,huella_peticion,recibo_ref,version,catalogo_version_ref,eleccion,foto_nueva,foto_retirada,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(m->>'persona_ref',s,m->>'clave_operacion',m->>'huella_peticion',ref,v,cat->>'version_ref',m->'eleccion',p_foto IS NOT NULL,v_retirada IS NOT NULL,cx.decision_ref,cx.auditoria_ref,cx.consumo_huella_sha256,ahora)
 RETURNING * INTO r;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','actualizar');
 RETURN vec_usuarios.recibo_imagen_json(r,false);
END $f$;

REVOKE ALL ON FUNCTION vec_usuarios.guardar_imagen_propia_v1_interna(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_usuarios.guardar_imagen_propia_v1_externa(
 p_material text,p_foto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; cx record; cat jsonb; previo record; v bigint; ref text; ahora timestamptz(6); n integer;
 v_foto_ref text; v_foto_sha text; v_retirada text; r vec_usuarios.imagen_recibo_externa; s text;
BEGIN
 m:=vec_usuarios.consumir_contexto_imagen(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,'actualizar');
 s:=m->>'superficie';
 IF (m->>'foto_sha256'='')<>(p_foto IS NULL)
    OR (p_foto IS NOT NULL AND encode(sha256(p_foto),'hex') IS DISTINCT FROM m->>'foto_sha256')
 THEN RAISE EXCEPTION 'Usuarios: foto divergente del material' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios:imagen:'||s||':'||(m->>'persona_ref'),0));
 IF EXISTS(SELECT 1 FROM vec_usuarios.imagen_recibo_externa WHERE persona_ref=m->>'persona_ref' AND superficie=s AND clave_operacion=m->>'clave_operacion')
 THEN RAISE EXCEPTION 'Usuarios: operación ya registrada; recuperar recibo' USING ERRCODE='40001'; END IF;
 cat:=vec_usuarios.catalogo_imagen_publicado();
 IF m->>'catalogo_version_ref' IS DISTINCT FROM cat->>'version_ref'
 THEN RAISE EXCEPTION 'Usuarios: catálogo de imagen obsoleto' USING ERRCODE='P1409'; END IF;
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'paletas') q WHERE q->>'codigo'=m#>>'{eleccion,paleta}')
    OR (m#>>'{eleccion,modo}'='icono' AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cat->'iconos') q WHERE q->>'codigo'=m#>>'{eleccion,icono}'))
 THEN RAISE EXCEPTION 'Usuarios: imagen fuera de catálogo' USING ERRCODE='22023'; END IF;
 SELECT version,foto_ref,foto_sha256 INTO previo FROM vec_usuarios.imagen_actual_externa WHERE persona_ref=m->>'persona_ref' AND superficie=s FOR UPDATE;
 IF coalesce(previo.version,0)::text IS DISTINCT FROM m->>'version_esperada'
 THEN RAISE EXCEPTION 'Usuarios: versión de imagen en conflicto' USING ERRCODE='P1409'; END IF;
 v:=coalesce(previo.version,0)+1;
 ref:='img_'||replace(gen_random_uuid()::text,'-','');
 ahora:=date_trunc('microseconds',clock_timestamp());
 v_foto_ref:=previo.foto_ref; v_foto_sha:=previo.foto_sha256;
 IF m#>>'{eleccion,modo}'='foto' AND p_foto IS NULL AND previo.foto_ref IS NULL
 THEN RAISE EXCEPTION 'Usuarios: no hay foto que conservar' USING ERRCODE='P1409'; END IF;
 IF previo.foto_ref IS NOT NULL AND (p_foto IS NOT NULL OR m#>>'{eleccion,modo}'<>'foto') THEN
  PERFORM vec_documentos.retirar_imagen_personal_v1(m->>'persona_ref',previo.foto_ref,ref);
  v_retirada:=previo.foto_ref; v_foto_ref:=NULL; v_foto_sha:=NULL;
 END IF;
 IF p_foto IS NOT NULL THEN
  v_foto_ref:=vec_documentos.custodiar_imagen_personal_v1(m->>'persona_ref',p_foto,m->>'foto_sha256',ref);
  v_foto_sha:=m->>'foto_sha256';
 END IF;
 IF previo.version IS NULL THEN
  INSERT INTO vec_usuarios.imagen_actual_externa(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,recibo_ref,actualizado_en)
  VALUES(m->>'persona_ref',s,v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,ref,ahora);
 ELSE
  UPDATE vec_usuarios.imagen_actual_externa SET version=v,catalogo_version_ref=cat->>'version_ref',eleccion=m->'eleccion',
   foto_ref=v_foto_ref,foto_sha256=v_foto_sha,recibo_ref=ref,actualizado_en=ahora
  WHERE persona_ref=m->>'persona_ref' AND superficie=s AND version=v-1;
  GET DIAGNOSTICS n=ROW_COUNT;
  IF n<>1 THEN RAISE EXCEPTION 'Usuarios: versión de imagen concurrente' USING ERRCODE='40001'; END IF;
 END IF;
 SELECT * INTO STRICT cx FROM vec_usuarios.imagen_contexto_externa WHERE xid=pg_current_xact_id_if_assigned()
  AND backend_pid=pg_backend_pid() AND sesion=session_user AND modo='actualizar' AND superficie=s;
 INSERT INTO vec_usuarios.imagen_historia_externa(persona_ref,superficie,version,catalogo_version_ref,eleccion,foto_ref,foto_sha256,foto_retirada_ref,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(m->>'persona_ref',s,v,cat->>'version_ref',m->'eleccion',v_foto_ref,v_foto_sha,v_retirada,ref,cx.decision_ref,cx.auditoria_ref,ahora);
 INSERT INTO vec_usuarios.imagen_recibo_externa(persona_ref,superficie,clave_operacion,huella_peticion,recibo_ref,version,catalogo_version_ref,eleccion,foto_nueva,foto_retirada,decision_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
 VALUES(m->>'persona_ref',s,m->>'clave_operacion',m->>'huella_peticion',ref,v,cat->>'version_ref',m->'eleccion',p_foto IS NOT NULL,v_retirada IS NOT NULL,cx.decision_ref,cx.auditoria_ref,cx.consumo_huella_sha256,ahora)
 RETURNING * INTO r;
 PERFORM vec_usuarios.retirar_contexto_imagen(m->>'persona_ref','actualizar');
 RETURN vec_usuarios.recibo_imagen_json(r,false);
END $f$;

REVOKE ALL ON FUNCTION vec_usuarios.guardar_imagen_propia_v1_externa(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_usuarios.guardar_imagen_propia_v1(
 p_material text,p_foto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF vec_usuarios.superficie_sesion_correos()='interna_corporativa' THEN
  RETURN vec_usuarios.guardar_imagen_propia_v1_interna(p_material,p_foto,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSIF vec_usuarios.superficie_sesion_correos()='externa_personal' THEN
  RETURN vec_usuarios.guardar_imagen_propia_v1_externa(p_material,p_foto,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE RAISE EXCEPTION 'Usuarios: superficie de imagen denegada' USING ERRCODE='42501'; END IF;

END $f$;

RESET ROLE;
COMMIT;
