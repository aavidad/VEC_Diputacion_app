\set ON_ERROR_STOP on
-- PG18, clon desechable tras dependencias: una transacción y ROLLBACK.
-- Dominios exclusivos de pg_temp y secuencia no reversible: el contador
-- conserva el efecto del CHECK aunque la llamada termine en una excepción.
-- No sombrear timestamptz: sus declaraciones con precisión no aceptan un dominio.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='60s';
CREATE ROLE prueba_usuarios_clausura LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_usuarios_ejecutor_externo TO prueba_usuarios_clausura WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
DO $temp$ BEGIN EXECUTE pg_catalog.format('GRANT TEMP ON DATABASE %I TO prueba_usuarios_clausura',pg_catalog.current_database()); END $temp$;
SET LOCAL SESSION AUTHORIZATION prueba_usuarios_clausura;
CREATE TEMP SEQUENCE contador_clausura MINVALUE 0 START 1;
SELECT pg_catalog.setval('pg_temp.contador_clausura'::pg_catalog.regclass,0,true);
CREATE FUNCTION pg_temp.sonda_clausura() RETURNS pg_catalog.bool LANGUAGE plpgsql AS $sonda$
BEGIN
 IF current_user=ANY(ARRAY['postgres','vec_autorizacion_atestada_v3_propietario','vec_documentos_propietario','vec_usuarios_correos_externo_propietario','vec_usuarios_correos_preflight_externo','vec_usuarios_propietario']) THEN
  PERFORM pg_catalog.nextval('pg_temp.contador_clausura'::pg_catalog.regclass);
 END IF;
 RETURN true;
END $sonda$;
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.bytea AS pg_catalog.bytea CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.numeric AS pg_catalog.numeric CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.oid AS pg_catalog.oid CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.regrole AS pg_catalog.regrole CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.regprocedure AS pg_catalog.regprocedure CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.bool AS pg_catalog.bool CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.int4 AS pg_catalog.int4 CHECK(pg_temp.sonda_clausura());
CREATE DOMAIN pg_temp.int8 AS pg_catalog.int8 CHECK(pg_temp.sonda_clausura());
RESET SESSION AUTHORIZATION;
GRANT USAGE,SELECT ON SEQUENCE pg_temp.contador_clausura TO postgres,vec_autorizacion_atestada_v3_propietario,vec_documentos_propietario,vec_usuarios_correos_externo_propietario,vec_usuarios_correos_preflight_externo,vec_usuarios_propietario;
REVOKE ALL ON FUNCTION pg_temp.sonda_clausura() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION pg_temp.sonda_clausura() TO postgres,vec_autorizacion_atestada_v3_propietario,vec_documentos_propietario,vec_usuarios_correos_externo_propietario,vec_usuarios_correos_preflight_externo,vec_usuarios_propietario;
CREATE TEMP TABLE controles_clausura(firma pg_catalog.text,consulta pg_catalog.text);
INSERT INTO pg_temp.controles_clausura VALUES
  ('vec_usuarios.consultar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_usuarios.consultar_imagen_propia_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.consultar_preferencias_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_usuarios.consultar_preferencias_propias_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios_correos_externo.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_usuarios_correos_externo.consultar_correos_propios_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_documentos.abrir_imagen_personal_v1(text,text)','SELECT vec_documentos.abrir_imagen_personal_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)');
GRANT SELECT ON pg_temp.controles_clausura TO prueba_usuarios_clausura;
DO $grants$ DECLARE f pg_catalog.text;
BEGIN
 FOR f IN SELECT firma FROM pg_temp.controles_clausura LOOP
  EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO prueba_usuarios_clausura',f);
 END LOOP;
END $grants$;
GRANT USAGE ON SCHEMA vec_documentos,vec_usuarios,vec_usuarios_correos_externo,vec_autorizacion_atestada_v3 TO prueba_usuarios_clausura;
SET LOCAL SESSION AUTHORIZATION prueba_usuarios_clausura;
DO $vulnerable$
DECLARE f record; denegada pg_catalog.bool; estado pg_catalog.text;
BEGIN
 FOR f IN SELECT * FROM pg_temp.controles_clausura LOOP
  PERFORM pg_catalog.setval('pg_temp.contador_clausura'::pg_catalog.regclass,0,true);
  denegada:=false;
  BEGIN EXECUTE f.consulta;
  EXCEPTION WHEN OTHERS THEN denegada:=true; GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE; END;
  IF NOT denegada OR estado NOT IN ('42501','22023') OR (SELECT last_value FROM pg_temp.contador_clausura)=0 THEN
   RAISE EXCEPTION 'U15-D11: falta control vulnerable para % (SQLSTATE %, contador %)',f.firma,estado,(SELECT last_value FROM pg_temp.contador_clausura);
  END IF;
 END LOOP;
END $vulnerable$;
RESET SESSION AUTHORIZATION;
-- Reponer ACL exacta antes de validar la preimagen.
DO $revoke$ DECLARE f pg_catalog.text;
BEGIN
 FOR f IN SELECT firma FROM pg_temp.controles_clausura LOOP
  EXECUTE pg_catalog.format('REVOKE EXECUTE ON FUNCTION %s FROM prueba_usuarios_clausura',f);
 END LOOP;
END $revoke$;
-- U15-D11-CORRECTIVOS-AQUI
-- Repetir actor, login, material y llamadas; comprobar denegación y contador.
DO $grants$ DECLARE f pg_catalog.text;
BEGIN
 FOR f IN SELECT firma FROM pg_temp.controles_clausura LOOP
  EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO prueba_usuarios_clausura',f);
 END LOOP;
END $grants$;
SET LOCAL SESSION AUTHORIZATION prueba_usuarios_clausura;
DO $cerrada$
DECLARE f record; denegada pg_catalog.bool; estado pg_catalog.text;
BEGIN
 FOR f IN SELECT * FROM pg_temp.controles_clausura LOOP
  PERFORM pg_catalog.setval('pg_temp.contador_clausura'::pg_catalog.regclass,0,true);
  denegada:=false;
  BEGIN EXECUTE f.consulta;
  EXCEPTION WHEN OTHERS THEN denegada:=true; GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE; END;
  IF NOT denegada OR estado NOT IN ('42501','22023') OR (SELECT last_value FROM pg_temp.contador_clausura)<>0 THEN
   RAISE EXCEPTION 'U15-D11: clausura incorrecta para % (SQLSTATE %, contador %)',f.firma,estado,(SELECT last_value FROM pg_temp.contador_clausura);
  END IF;
 END LOOP;
END $cerrada$;
RESET SESSION AUTHORIZATION;
CREATE TEMP TABLE llamadas_clausura(firma pg_catalog.text,propietario pg_catalog.text,consulta pg_catalog.text);
INSERT INTO pg_temp.llamadas_clausura VALUES
  ('vec_usuarios.activar_contexto_preferencias(text,text,text,text,text)','vec_usuarios_propietario','SELECT vec_usuarios.activar_contexto_preferencias(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_usuarios.catalogo_imagen_publicado()','vec_usuarios_propietario','SELECT vec_usuarios.catalogo_imagen_publicado()'),
  ('vec_usuarios.catalogo_vigente_imagen_v1(text)','vec_usuarios_propietario','SELECT vec_usuarios.catalogo_vigente_imagen_v1(NULL::pg_catalog.text)'),
  ('vec_usuarios.catalogo_vigente_preferencias_v1(text)','vec_usuarios_propietario','SELECT vec_usuarios.catalogo_vigente_preferencias_v1(NULL::pg_catalog.text)'),
  ('vec_usuarios.consultar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.consultar_imagen_propia_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.consultar_imagen_propia_v1_externa(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.consultar_imagen_propia_v1_externa(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.consultar_imagen_propia_v1_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.consultar_imagen_propia_v1_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.consultar_preferencias_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.consultar_preferencias_propias_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.consumir_contexto_imagen(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)','vec_usuarios_propietario','SELECT vec_usuarios.consumir_contexto_imagen(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.text)'),
  ('vec_usuarios.contexto_autorizado_imagen_superficie(text,text,text[])','vec_usuarios_propietario','SELECT vec_usuarios.contexto_autorizado_imagen_superficie(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text[])'),
  ('vec_usuarios.contexto_autorizado_superficie(text,text,text[])','vec_usuarios_propietario','SELECT vec_usuarios.contexto_autorizado_superficie(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text[])'),
  ('vec_usuarios.eleccion_imagen_valida(jsonb)','vec_usuarios_propietario','SELECT vec_usuarios.eleccion_imagen_valida(NULL::pg_catalog.jsonb)'),
  ('vec_usuarios.exigir_ejecutor_preferencias()','vec_usuarios_propietario','SELECT vec_usuarios.exigir_ejecutor_preferencias()'),
  ('vec_usuarios.guardar_imagen_propia_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.guardar_imagen_propia_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.guardar_imagen_propia_v1_externa(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.guardar_imagen_propia_v1_externa(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.guardar_imagen_propia_v1_interna(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.guardar_imagen_propia_v1_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.guardar_preferencias_propias_v1(NULL::pg_catalog.text,NULL::pg_catalog.jsonb,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.huella_contexto_imagen(text)','vec_usuarios_propietario','SELECT vec_usuarios.huella_contexto_imagen(NULL::pg_catalog.text)'),
  ('vec_usuarios.huella_contexto_preferencias(text)','vec_usuarios_propietario','SELECT vec_usuarios.huella_contexto_preferencias(NULL::pg_catalog.text)'),
  ('vec_usuarios.huella_peticion_imagen(jsonb)','vec_usuarios_propietario','SELECT vec_usuarios.huella_peticion_imagen(NULL::pg_catalog.jsonb)'),
  ('vec_usuarios.huella_semantica_preferencias(text,bigint,text,jsonb)','vec_usuarios_propietario','SELECT vec_usuarios.huella_semantica_preferencias(NULL::pg_catalog.text,NULL::pg_catalog.int8,NULL::pg_catalog.text,NULL::pg_catalog.jsonb)'),
  ('vec_usuarios.recibo_imagen_json(vec_usuarios.imagen_recibo,boolean)','vec_usuarios_propietario','SELECT vec_usuarios.recibo_imagen_json(NULL::vec_usuarios.imagen_recibo,NULL::pg_catalog.bool)'),
  ('vec_usuarios.recibo_imagen_json(vec_usuarios.imagen_recibo_externa,boolean)','vec_usuarios_propietario','SELECT vec_usuarios.recibo_imagen_json(NULL::vec_usuarios.imagen_recibo_externa,NULL::pg_catalog.bool)'),
  ('vec_usuarios.recuperar_imagen_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.recuperar_imagen_operacion_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.recuperar_imagen_operacion_v1_externa(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.recuperar_imagen_operacion_v1_externa(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.recuperar_imagen_operacion_v1_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.recuperar_imagen_operacion_v1_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.recuperar_preferencias_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_propietario','SELECT vec_usuarios.recuperar_preferencias_operacion_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)','vec_usuarios_propietario','SELECT vec_usuarios.registrar_denegacion_preferencias_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_usuarios.retirar_contexto_imagen(text,text)','vec_usuarios_propietario','SELECT vec_usuarios.retirar_contexto_imagen(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_usuarios.retirar_contexto_preferencias(text,text,text)','vec_usuarios_propietario','SELECT vec_usuarios.retirar_contexto_preferencias(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_usuarios.sesion_lectora_catalogo()','vec_usuarios_propietario','SELECT vec_usuarios.sesion_lectora_catalogo()'),
  ('vec_usuarios.sesion_lectora_catalogo_imagen()','vec_usuarios_propietario','SELECT vec_usuarios.sesion_lectora_catalogo_imagen()'),
  ('vec_usuarios.sesion_migradora_catalogo()','vec_usuarios_propietario','SELECT vec_usuarios.sesion_migradora_catalogo()'),
  ('vec_usuarios.sesion_superficie_valida(text)','vec_usuarios_propietario','SELECT vec_usuarios.sesion_superficie_valida(NULL::pg_catalog.text)'),
  ('vec_usuarios.superficie_sesion_correos()','vec_usuarios_propietario','SELECT vec_usuarios.superficie_sesion_correos()'),
  ('vec_usuarios.validar_material_imagen(text,bytea,bytea,numeric,numeric)','vec_usuarios_propietario','SELECT vec_usuarios.validar_material_imagen(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_usuarios.validar_material_preferencias(text,text,bytea,bytea,numeric,numeric)','vec_usuarios_propietario','SELECT vec_usuarios.validar_material_preferencias(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_usuarios.valores_validos(jsonb)','vec_usuarios_propietario','SELECT vec_usuarios.valores_validos(NULL::pg_catalog.jsonb)'),
  ('vec_usuarios_correos_externo.activar_contexto_correos(jsonb,text,jsonb)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.activar_contexto_correos(NULL::pg_catalog.jsonb,NULL::pg_catalog.text,NULL::pg_catalog.jsonb)'),
  ('vec_usuarios_correos_externo.aplicar_correos_propios_v1(text,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.aplicar_correos_propios_v1(NULL::pg_catalog.text,NULL::pg_catalog.jsonb,NULL::pg_catalog.jsonb,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios_correos_externo.cerrar_verificacion_correo_v1(text,text,boolean)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.cerrar_verificacion_correo_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bool)'),
  ('vec_usuarios_correos_externo.confirmar_envio_correo_v1(text,text,text,boolean)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.confirmar_envio_correo_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bool)'),
  ('vec_usuarios_correos_externo.consultar_correos_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.consultar_correos_propios_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios_correos_externo.consumir_contexto_correos(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.consumir_contexto_correos(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.text)'),
  ('vec_usuarios_correos_externo.contexto_autorizado_correos(text,text[])','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.contexto_autorizado_correos(NULL::pg_catalog.text,NULL::pg_catalog.text[])'),
  ('vec_usuarios_correos_externo.contexto_avisos_autorizado(text)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.contexto_avisos_autorizado(NULL::pg_catalog.text)'),
  ('vec_usuarios_correos_externo.huella_contexto_correos(text)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.huella_contexto_correos(NULL::pg_catalog.text)'),
  ('vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()','vec_usuarios_correos_preflight_externo','SELECT vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()'),
  ('vec_usuarios_correos_externo.preparar_verificacion_correo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.preparar_verificacion_correo_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios_correos_externo.recuperar_correos_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.recuperar_correos_operacion_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_usuarios_correos_externo.registrar_recibo_correos(text,boolean,text,text,bigint,timestamp with time zone)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.registrar_recibo_correos(NULL::pg_catalog.text,NULL::pg_catalog.bool,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.int8,NULL::pg_catalog.timestamptz)'),
  ('vec_usuarios_correos_externo.replay_correos_autorizado(jsonb)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.replay_correos_autorizado(NULL::pg_catalog.jsonb)'),
  ('vec_usuarios_correos_externo.reservar_envio_correos(text,text,text,text,text,text,timestamp with time zone)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.reservar_envio_correos(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_usuarios_correos_externo.retirar_contexto_correos(text,text)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.retirar_contexto_correos(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_usuarios_correos_externo.sesion_avisos_interna()','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.sesion_avisos_interna()'),
  ('vec_usuarios_correos_externo.sobre_correo_json(vec_usuarios_correos_externo.correos_direccion)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.sobre_correo_json(NULL::vec_usuarios_correos_externo.correos_direccion)'),
  ('vec_usuarios_correos_externo.superficie_sesion()','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.superficie_sesion()'),
  ('vec_usuarios_correos_externo.validar_material_correos(text,bytea,bytea,numeric,numeric)','vec_usuarios_correos_externo_propietario','SELECT vec_usuarios_correos_externo.validar_material_correos(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_usuarios_correos_reclaveado.fila_sellada_v1(text,jsonb)','vec_usuarios_correos_preflight_externo','SELECT vec_usuarios_correos_reclaveado.fila_sellada_v1(NULL::pg_catalog.text,NULL::pg_catalog.jsonb)'),
  ('vec_usuarios_correos_reclaveado.pk_fila_v1(text,jsonb)','postgres','SELECT vec_usuarios_correos_reclaveado.pk_fila_v1(NULL::pg_catalog.text,NULL::pg_catalog.jsonb)'),
  ('vec_documentos.abrir_imagen_personal_v1(text,text)','vec_documentos_propietario','SELECT vec_documentos.abrir_imagen_personal_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.abrir_imagen_personal_v1_externa(text,text)','vec_documentos_propietario','SELECT vec_documentos.abrir_imagen_personal_v1_externa(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.abrir_imagen_personal_v1_interna(text,text)','vec_documentos_propietario','SELECT vec_documentos.abrir_imagen_personal_v1_interna(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.custodiar_imagen_personal_v1(text,bytea,text,text)','vec_documentos_propietario','SELECT vec_documentos.custodiar_imagen_personal_v1(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.custodiar_imagen_personal_v1_externa(text,bytea,text,text)','vec_documentos_propietario','SELECT vec_documentos.custodiar_imagen_personal_v1_externa(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.custodiar_imagen_personal_v1_interna(text,bytea,text,text)','vec_documentos_propietario','SELECT vec_documentos.custodiar_imagen_personal_v1_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.retirar_imagen_personal_v1(text,text,text)','vec_documentos_propietario','SELECT vec_documentos.retirar_imagen_personal_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.retirar_imagen_personal_v1_externa(text,text,text)','vec_documentos_propietario','SELECT vec_documentos.retirar_imagen_personal_v1_externa(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.retirar_imagen_personal_v1_interna(text,text,text)','vec_documentos_propietario','SELECT vec_documentos.retirar_imagen_personal_v1_interna(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_documentos.sesion_usuarios_imagen_v1()','vec_documentos_propietario','SELECT vec_documentos.sesion_usuarios_imagen_v1()'),
  ('vec_documentos.superficie_sesion_imagen_v1()','vec_documentos_propietario','SELECT vec_documentos.superficie_sesion_imagen_v1()'),
  ('vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','SELECT vec_autorizacion_atestada_v3.consumir_correos_v3_atestada(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','SELECT vec_autorizacion_atestada_v3.consumir_imagen_v3_atestada(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','SELECT vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_autorizacion_atestada_v3_propietario','SELECT vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)');
-- SI también se ejecuta desde una autoridad SD: probar bajo propietario,
-- sin conceder privilegios productivos a la identidad de prueba.
DO $callees$
DECLARE f record; estado pg_catalog.text;
BEGIN
 FOR f IN SELECT * FROM pg_temp.llamadas_clausura LOOP
  PERFORM pg_catalog.setval('pg_temp.contador_clausura'::pg_catalog.regclass,0,true);
  EXECUTE pg_catalog.format('SET LOCAL ROLE %I',f.propietario);
  BEGIN EXECUTE f.consulta;
  EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE; END;
  RESET ROLE;
  IF (SELECT last_value FROM pg_temp.contador_clausura)<>0 THEN
   RAISE EXCEPTION 'U15-D11: callee temporal alcanzado %',f.firma;
  END IF;
 END LOOP;
END $callees$;
ROLLBACK;
\echo U15-D11-TIPOS-CIERRE-OK
