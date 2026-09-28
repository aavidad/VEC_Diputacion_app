\set ON_ERROR_STOP on
-- Usuarios 000001. Requiere roles_up y AD3-106 instalado previamente.
-- No hay salida de correo: los avisos son elecciones de la persona.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000001',0));
DO $pre$ BEGIN
 IF current_user<>'vec_usuarios_propietario'
    OR to_regnamespace('vec_usuarios') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'Usuarios 000001: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE SCHEMA vec_usuarios AUTHORIZATION vec_usuarios_propietario;
REVOKE ALL ON SCHEMA vec_usuarios FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_usuarios_propietario REVOKE ALL ON TABLES FROM PUBLIC,vec_usuarios_ejecutor;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_usuarios_propietario REVOKE ALL ON SEQUENCES FROM PUBLIC,vec_usuarios_ejecutor;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_usuarios_propietario REVOKE ALL ON FUNCTIONS FROM PUBLIC,vec_usuarios_ejecutor;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_usuarios_propietario REVOKE ALL ON TYPES FROM PUBLIC,vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.rechazar_cambio_inmutable()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'historia de Usuarios inmutable' USING ERRCODE='55000'; END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.rechazar_cambio_inmutable() FROM PUBLIC;

CREATE FUNCTION vec_usuarios.valores_validos(v jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
 RETURN jsonb_typeof(v)='object'
  AND ARRAY(SELECT jsonb_object_keys(v) ORDER BY 1) IS NOT DISTINCT FROM
   ARRAY['alto_contraste','aviso_correo_plazos','aviso_correo_tareas','filas','idioma','inicio','tamano_texto','tema']
  AND v->>'idioma' IN ('navegador','es','en')
  AND v->>'tamano_texto' IN ('normal','grande','muy_grande')
  AND jsonb_typeof(v->'alto_contraste')='boolean'
  AND v->>'tema' IN ('sistema','claro','oscuro')
  AND v->>'inicio' IN ('cuadro','peticiones','bolsas')
  AND jsonb_typeof(v->'filas')='number' AND v->>'filas' IN ('20','50','100')
  AND jsonb_typeof(v->'aviso_correo_tareas')='boolean'
  AND jsonb_typeof(v->'aviso_correo_plazos')='boolean';
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.valores_validos(jsonb) FROM PUBLIC;

CREATE TABLE vec_usuarios.catalogo_preferencias (
 version_ref text PRIMARY KEY CHECK(version_ref ~ '^[a-z][A-Za-z0-9:._-]{2,95}$'),
 definicion jsonb NOT NULL CHECK(jsonb_typeof(definicion)='object'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 publicado_en timestamptz(6) NOT NULL,
 CHECK(vec_usuarios.valores_validos(definicion->'predeterminados'))
);
CREATE TABLE vec_usuarios.catalogo_publicacion (
 secuencia bigint PRIMARY KEY CHECK(secuencia>0),
 version_ref text NOT NULL UNIQUE REFERENCES vec_usuarios.catalogo_preferencias(version_ref),
 publicada_en timestamptz(6) NOT NULL
);
-- Las versiones y sus publicaciones son append-only. Una migración posterior
-- añade la siguiente secuencia tras validar la preimagen exacta.
CREATE TRIGGER catalogo_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.catalogo_preferencias
 FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER catalogo_no_truncar BEFORE TRUNCATE ON vec_usuarios.catalogo_preferencias
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER publicacion_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.catalogo_publicacion
 FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER publicacion_no_truncar BEFORE TRUNCATE ON vec_usuarios.catalogo_publicacion
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();

CREATE TABLE vec_usuarios.preferencias_actual (
 persona_ref text PRIMARY KEY CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version>0),
 catalogo_version_ref text NOT NULL REFERENCES vec_usuarios.catalogo_preferencias(version_ref),
 valores jsonb NOT NULL CHECK(vec_usuarios.valores_validos(valores)),
 recibo_ref text NOT NULL,
 actualizado_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_usuarios.preferencias_historia (
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK(version>0),
 catalogo_version_ref text NOT NULL REFERENCES vec_usuarios.catalogo_preferencias(version_ref),
 valores jsonb NOT NULL CHECK(vec_usuarios.valores_validos(valores)),
 recibo_ref text NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,version),
 FOREIGN KEY(persona_ref) REFERENCES vec_usuarios.preferencias_actual(persona_ref) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE vec_usuarios.preferencias_recibo (
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_operacion text NOT NULL CHECK(length(clave_operacion) BETWEEN 16 AND 128 AND clave_operacion ~ '^[A-Za-z0-9:._-]+$'),
 huella_peticion text NOT NULL CHECK(huella_peticion ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^pref_[0-9a-f]{32}$'),
 version bigint NOT NULL CHECK(version>0),
 catalogo_version_ref text NOT NULL REFERENCES vec_usuarios.catalogo_preferencias(version_ref),
 valores jsonb NOT NULL CHECK(vec_usuarios.valores_validos(valores)),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(persona_ref,clave_operacion),
 FOREIGN KEY(persona_ref,version) REFERENCES vec_usuarios.preferencias_historia(persona_ref,version)
  DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.preferencias_historia
 FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_usuarios.preferencias_historia
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER recibo_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios.preferencias_recibo
 FOR EACH ROW EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER recibo_no_truncar BEFORE TRUNCATE ON vec_usuarios.preferencias_recibo
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();

-- Marcador privado y transitorio: se inserta tras un consumo V3 nuevo en esta
-- misma transacción y se borra antes del retorno. Ni PUBLIC ni el ejecutor
-- poseen DML/EXECUTE sobre él. La PK prohíbe acumular contextos en una tx.
CREATE TABLE vec_usuarios.contexto_transaccion (
 xid xid8 NOT NULL,
 backend_pid integer NOT NULL,
 sesion text NOT NULL,
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 modo text NOT NULL CHECK(modo IN ('consultar','recuperar','actualizar')),
 decision_ref text NOT NULL CHECK(length(decision_ref) BETWEEN 1 AND 256),
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(xid,backend_pid,sesion)
);
CREATE TRIGGER contexto_no_truncar BEFORE TRUNCATE ON vec_usuarios.contexto_transaccion
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();

CREATE FUNCTION vec_usuarios.contexto_autorizado(p_persona text,p_modos text[])
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT EXISTS(SELECT 1 FROM vec_usuarios.contexto_transaccion c
  WHERE c.xid=pg_current_xact_id_if_assigned() AND c.backend_pid=pg_backend_pid()
    AND c.sesion=session_user AND c.persona_ref=p_persona AND c.modo=ANY(p_modos))
$f$;
REVOKE ALL ON FUNCTION vec_usuarios.contexto_autorizado(text,text[]) FROM PUBLIC,vec_usuarios_ejecutor;

DO $rls$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['catalogo_preferencias','catalogo_publicacion','preferencias_actual','preferencias_historia','preferencias_recibo','contexto_transaccion'] LOOP
  EXECUTE format('ALTER TABLE vec_usuarios.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_usuarios.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_usuarios.%I FROM PUBLIC,vec_usuarios_ejecutor',t);
 END LOOP;
END $rls$;
-- El catálogo es gobernado por migración; no contiene estado personal.
CREATE POLICY catalogo_lectura ON vec_usuarios.catalogo_preferencias FOR SELECT TO vec_usuarios_propietario USING (true);
CREATE POLICY catalogo_alta ON vec_usuarios.catalogo_preferencias FOR INSERT TO vec_usuarios_propietario WITH CHECK (true);
CREATE POLICY publicacion_lectura ON vec_usuarios.catalogo_publicacion FOR SELECT TO vec_usuarios_propietario USING (true);
CREATE POLICY publicacion_alta ON vec_usuarios.catalogo_publicacion FOR INSERT TO vec_usuarios_propietario WITH CHECK (true);
-- El contexto sólo es accesible al propietario de funciones. Los predicados
-- de estado se atan al xid8 completo, backend, LOGIN, persona y modo.
CREATE POLICY contexto_lectura ON vec_usuarios.contexto_transaccion FOR SELECT TO vec_usuarios_propietario
 USING (current_user='vec_usuarios_propietario');
CREATE POLICY contexto_alta ON vec_usuarios.contexto_transaccion FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (current_user='vec_usuarios_propietario');
CREATE POLICY contexto_retirada ON vec_usuarios.contexto_transaccion FOR DELETE TO vec_usuarios_propietario
 USING (current_user='vec_usuarios_propietario');
CREATE POLICY estado_lectura ON vec_usuarios.preferencias_actual FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado(persona_ref,ARRAY['consultar','actualizar']));
CREATE POLICY estado_alta ON vec_usuarios.preferencias_actual FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado(persona_ref,ARRAY['actualizar']));
CREATE POLICY estado_cambio ON vec_usuarios.preferencias_actual FOR UPDATE TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado(persona_ref,ARRAY['actualizar']))
 WITH CHECK (vec_usuarios.contexto_autorizado(persona_ref,ARRAY['actualizar']));
CREATE POLICY historia_alta ON vec_usuarios.preferencias_historia FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado(persona_ref,ARRAY['actualizar']));
CREATE POLICY recibo_lectura ON vec_usuarios.preferencias_recibo FOR SELECT TO vec_usuarios_propietario
 USING (vec_usuarios.contexto_autorizado(persona_ref,ARRAY['recuperar','actualizar']));
CREATE POLICY recibo_alta ON vec_usuarios.preferencias_recibo FOR INSERT TO vec_usuarios_propietario
 WITH CHECK (vec_usuarios.contexto_autorizado(persona_ref,ARRAY['actualizar']));
INSERT INTO vec_usuarios.catalogo_preferencias(version_ref,definicion,huella_sha256,publicado_en)
SELECT 'usuarios-preferencias-v1',c,encode(sha256(convert_to(c::text,'UTF8')),'hex'),
       date_trunc('microseconds',clock_timestamp())
FROM (SELECT '{
 "version_ref":"usuarios-preferencias-v1",
 "idiomas":[{"codigo":"navegador","nombre_key":"ui.usuarios.preferencias.idioma.navegador"},{"codigo":"es","nombre_key":"ui.usuarios.preferencias.idioma.es"},{"codigo":"en","nombre_key":"ui.usuarios.preferencias.idioma.en"}],
 "tamanos_texto":[{"codigo":"normal","nombre_key":"ui.usuarios.preferencias.tamano_texto.normal"},{"codigo":"grande","nombre_key":"ui.usuarios.preferencias.tamano_texto.grande"},{"codigo":"muy_grande","nombre_key":"ui.usuarios.preferencias.tamano_texto.muy_grande"}],
 "temas":[{"codigo":"sistema","nombre_key":"ui.usuarios.preferencias.tema.sistema"},{"codigo":"claro","nombre_key":"ui.usuarios.preferencias.tema.claro"},{"codigo":"oscuro","nombre_key":"ui.usuarios.preferencias.tema.oscuro"}],
 "inicios":[{"codigo":"cuadro","nombre_key":"ui.usuarios.preferencias.inicio.cuadro"},{"codigo":"peticiones","nombre_key":"ui.usuarios.preferencias.inicio.peticiones"},{"codigo":"bolsas","nombre_key":"ui.usuarios.preferencias.inicio.bolsas"}],
 "filas":[20,50,100],
 "predeterminados":{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}
}'::jsonb AS c) q;
INSERT INTO vec_usuarios.catalogo_publicacion(secuencia,version_ref,publicada_en)
VALUES(1,'usuarios-preferencias-v1',date_trunc('microseconds',clock_timestamp()));
COMMIT;
