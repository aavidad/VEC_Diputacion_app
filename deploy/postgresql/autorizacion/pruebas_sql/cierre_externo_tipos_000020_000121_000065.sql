\set ON_ERROR_STOP on
-- Solo en el clon PostgreSQL 18 desechable. Nunca instala SQL ni deja dominios.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
CREATE ROLE prueba_portales_tipos LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_registro_externo TO prueba_portales_tipos WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
DO $temp$ BEGIN EXECUTE pg_catalog.format('GRANT TEMP ON DATABASE %I TO prueba_portales_tipos',pg_catalog.current_database()); END $temp$;
SET LOCAL SESSION AUTHORIZATION prueba_portales_tipos;
CREATE TEMP TABLE sonda_tipos(owner_actual pg_catalog.text);
CREATE FUNCTION pg_temp.sonda_tipo() RETURNS pg_catalog.bool LANGUAGE plpgsql AS $sonda$
BEGIN
 IF current_user IN('vec_autorizacion_propietario','vec_autorizacion_atestada_v3_propietario','vec_bolsa_llamamientos_propietario') THEN
  INSERT INTO pg_temp.sonda_tipos VALUES(current_user);
 END IF;
 RETURN true;
END $sonda$;
CREATE DOMAIN pg_temp.oid AS pg_catalog.oid CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.regrole AS pg_catalog.regrole CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.jsonb AS pg_catalog.jsonb CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.text AS pg_catalog.text CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.bytea AS pg_catalog.bytea CHECK(pg_temp.sonda_tipo());
CREATE DOMAIN pg_temp.numeric AS pg_catalog.numeric CHECK(pg_temp.sonda_tipo());
GRANT INSERT,SELECT ON TABLE pg_temp.sonda_tipos TO vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario;
REVOKE ALL ON FUNCTION pg_temp.sonda_tipo() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION pg_temp.sonda_tipo() TO vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario;
RESET SESSION AUTHORIZATION;
DO $retirar_temp$ BEGIN EXECUTE pg_catalog.format('REVOKE TEMP ON DATABASE %I FROM prueba_portales_tipos',pg_catalog.current_database()); END $retirar_temp$;
-- La sonda demuestra el fallo anterior y obliga a invalidar el plan tipado al
-- corregir la misma función. La concesión de ensayo se retira antes del ALTER.
GRANT EXECUTE ON FUNCTION vec_autorizacion.login_candidato_externo_v1(text,text) TO prueba_portales_tipos;
SET LOCAL SESSION AUTHORIZATION prueba_portales_tipos;
SELECT vec_autorizacion.login_candidato_externo_v1('prueba_portales_tipos','vec_autorizacion_registro_externo');
DO $antes$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_temp.sonda_tipos WHERE owner_actual='vec_autorizacion_propietario') THEN
  RAISE EXCEPTION 'PORTALES-TIPOS: la sonda no reprodujo la preimagen vulnerable';
 END IF;
 TRUNCATE pg_temp.sonda_tipos;
END $antes$;
RESET SESSION AUTHORIZATION;
REVOKE EXECUTE ON FUNCTION vec_autorizacion.login_candidato_externo_v1(text,text) FROM prueba_portales_tipos;
CREATE TEMP TABLE conservacion_tipos(relacion pg_catalog.oid,huella pg_catalog.text);
DO $guardar_historia$
DECLARE tabla record; huella pg_catalog.text;
BEGIN
 FOR tabla IN SELECT c.oid,c.oid::pg_catalog.regclass AS ref FROM pg_catalog.pg_class c
  JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname IN('vec_autorizacion','vec_autorizacion_atestada_v3','vec_bolsa_llamamientos') AND c.relkind='r' LOOP
  EXECUTE pg_catalog.format('SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(r) ORDER BY pg_catalog.to_jsonb(r)::pg_catalog.text),''[]''::pg_catalog.jsonb)::pg_catalog.text,''UTF8'')),''hex'') FROM %s r',tabla.ref) INTO huella;
  INSERT INTO pg_temp.conservacion_tipos VALUES(tabla.oid,huella);
 END LOOP;
END $guardar_historia$;
CREATE TEMP TABLE canon_tipos(valor pg_catalog.text);
INSERT INTO pg_temp.canon_tipos VALUES(
 vec_autorizacion.texto_json_go_v3('dato sintetico')||
 vec_autorizacion.lista_textos_v3_canonica('["A","B"]'::pg_catalog.jsonb)||
 vec_autorizacion_atestada_v3.texto_json_go('dato sintetico')||
 vec_autorizacion_atestada_v3.encuadrar_mac('dato sintetico'));
-- PORTALES-CORRECTIVOS-AQUI
DO $conservar_historia_canon$
DECLARE tabla record; huella pg_catalog.text; canon pg_catalog.text;
BEGIN
 FOR tabla IN SELECT c.relacion,c.relacion::pg_catalog.regclass AS ref,c.huella FROM pg_temp.conservacion_tipos c LOOP
  EXECUTE pg_catalog.format('SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(r) ORDER BY pg_catalog.to_jsonb(r)::pg_catalog.text),''[]''::pg_catalog.jsonb)::pg_catalog.text,''UTF8'')),''hex'') FROM %s r',tabla.ref) INTO huella;
  IF huella IS DISTINCT FROM tabla.huella THEN RAISE EXCEPTION 'PORTALES-TIPOS: historia alterada'; END IF;
 END LOOP;
 canon:=vec_autorizacion.texto_json_go_v3('dato sintetico')||
 vec_autorizacion.lista_textos_v3_canonica('["A","B"]'::pg_catalog.jsonb)||
 vec_autorizacion_atestada_v3.texto_json_go('dato sintetico')||
 vec_autorizacion_atestada_v3.encuadrar_mac('dato sintetico');
 IF canon IS DISTINCT FROM (SELECT valor FROM pg_temp.canon_tipos) THEN RAISE EXCEPTION 'PORTALES-TIPOS: canon alterado'; END IF;
END $conservar_historia_canon$;
-- Todas las firmas de la clausura se prueban con entrada vacía. Ninguna
-- devuelve información personal; no se fabrican decisiones ni recibos.
CREATE TEMP TABLE llamadas_sonda(firma pg_catalog.text,consulta pg_catalog.text);
INSERT INTO pg_temp.llamadas_sonda VALUES
  ('vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb)','SELECT vec_autorizacion.decision_contexto_actor_v3_canonica(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion.decision_contexto_actor_v3_valida(jsonb)','SELECT vec_autorizacion.decision_contexto_actor_v3_valida(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion.lista_textos_v3_canonica(jsonb)','SELECT vec_autorizacion.lista_textos_v3_canonica(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion.login_candidato_externo_v1(text,text)','SELECT vec_autorizacion.login_candidato_externo_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion.manifiesto_politicas_v3_canonico(jsonb)','SELECT vec_autorizacion.manifiesto_politicas_v3_canonico(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb)','SELECT vec_autorizacion.motivo_contexto_actor_v3_canonico(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion.obtener_instantanea_candidato_externo_v1(text,text)','SELECT vec_autorizacion.obtener_instantanea_candidato_externo_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion.publicador_candidato_externo_interno_valido_v1()','SELECT vec_autorizacion.publicador_candidato_externo_interno_valido_v1()'),
  ('vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)','SELECT vec_autorizacion.publicar_asignacion_candidato_externo_v1(NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.int8,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)','SELECT vec_autorizacion.publicar_rol_candidato_externo_v1(NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.text,NULL::pg_catalog.numeric,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.registrar_decision_candidato_externo_v3(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamp with time zone)','SELECT vec_autorizacion.resolver_motivo_candidato_externo_v1(NULL::pg_catalog.text,NULL::pg_catalog.int4,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.revalidar_decision_contexto_actor_externa_v3_viva(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','SELECT vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric)'),
  ('vec_autorizacion.revalidar_sesion_vinculo_externo_v2(jsonb,timestamp with time zone,timestamp with time zone,timestamp with time zone)','SELECT vec_autorizacion.revalidar_sesion_vinculo_externo_v2(NULL::pg_catalog.jsonb,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz)'),
  ('vec_autorizacion.revalidar_sesion_vinculo_v2(jsonb,timestamp with time zone,timestamp with time zone,timestamp with time zone)','SELECT vec_autorizacion.revalidar_sesion_vinculo_v2(NULL::pg_catalog.jsonb,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz)'),
  ('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)','SELECT vec_autorizacion.rol_candidato_externo_acotado_v1(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion.texto_ascii_visible_v3_valido(text,integer)','SELECT vec_autorizacion.texto_ascii_visible_v3_valido(NULL::pg_catalog.text,NULL::pg_catalog.int4)'),
  ('vec_autorizacion.texto_json_go_v3(text)','SELECT vec_autorizacion.texto_json_go_v3(NULL::pg_catalog.text)'),
  ('vec_autorizacion.validar_avance_asignacion_actual_externa()','SELECT vec_autorizacion.validar_avance_asignacion_actual_externa()'),
  ('vec_autorizacion.vinculo_contexto_actor_v2_canonico(jsonb)','SELECT vec_autorizacion.vinculo_contexto_actor_v2_canonico(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion.vinculo_contexto_actor_v2_valido(jsonb)','SELECT vec_autorizacion.vinculo_contexto_actor_v2_valido(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion_atestada_v3.bytea_igual_constante(bytea,bytea)','SELECT vec_autorizacion_atestada_v3.bytea_igual_constante(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.capacidad_canonica(jsonb)','SELECT vec_autorizacion_atestada_v3.capacidad_canonica(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)','SELECT vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.capacidad_tipos_validos(jsonb)','SELECT vec_autorizacion_atestada_v3.capacidad_tipos_validos(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)','SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(NULL::pg_catalog.text,NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.encuadrar_mac(text)','SELECT vec_autorizacion_atestada_v3.encuadrar_mac(NULL::pg_catalog.text)'),
  ('vec_autorizacion_atestada_v3.huella_sha256_valida(text)','SELECT vec_autorizacion_atestada_v3.huella_sha256_valida(NULL::pg_catalog.text)'),
  ('vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)','SELECT vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(NULL::pg_catalog.text,NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion_atestada_v3.preimagen_mac(jsonb)','SELECT vec_autorizacion_atestada_v3.preimagen_mac(NULL::pg_catalog.jsonb)'),
  ('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_historial_propio_bolsa_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_autorizacion_atestada_v3.texto_json_go(text)','SELECT vec_autorizacion_atestada_v3.texto_json_go(NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.anotar_consumo_candidato_v1(text,text,text)','SELECT vec_bolsa_llamamientos.anotar_consumo_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.confirmar_contacto_propio_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.int8,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamp with time zone,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.int4,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.consultar_mi_bolsa_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamp with time zone)','SELECT vec_bolsa_llamamientos.estado_cese_bolsa_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_bolsa_llamamientos.exigir_consumo_candidato_v1(text[],text)','SELECT vec_bolsa_llamamientos.exigir_consumo_candidato_v1(NULL::pg_catalog.text[],NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.exigir_portal_candidato_v1(text,text,text,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.exigir_portal_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.firma_marca_consumo_v1(xid8,text,text,text)','SELECT vec_bolsa_llamamientos.firma_marca_consumo_v1(NULL::pg_catalog.xid8,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamp with time zone)','SELECT vec_bolsa_llamamientos.leer_contacto_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamp with time zone)','SELECT vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamp with time zone,text[])','SELECT vec_bolsa_llamamientos.leer_portal_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text[])'),
  ('vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamp with time zone)','SELECT vec_bolsa_llamamientos.listar_ofertas_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_bolsa_llamamientos.listar_participaciones_candidato_v1(text)','SELECT vec_bolsa_llamamientos.listar_participaciones_candidato_v1(NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(text,timestamp with time zone,text[])','SELECT vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text[])'),
  ('vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.participacion_contacto_candidato_v1(text,text)','SELECT vec_bolsa_llamamientos.participacion_contacto_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(text)','SELECT vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.participacion_oferta_candidato_v1(text,text)','SELECT vec_bolsa_llamamientos.participacion_oferta_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.participacion_portal_candidato_v1(text,text)','SELECT vec_bolsa_llamamientos.participacion_portal_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.preparar_respuesta_portal_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.preparar_respuesta_portal_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.proyectar_oferta_v1(text,timestamp with time zone)','SELECT vec_bolsa_llamamientos.proyectar_oferta_v1(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamp with time zone)','SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(NULL::pg_catalog.text,NULL::pg_catalog.timestamptz)'),
  ('vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(text,text,text,bigint,text,text,timestamp with time zone,text)','SELECT vec_bolsa_llamamientos.registrar_confirmacion_contacto_interna_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.int8,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(text,text,text,text,text,timestamp with time zone,text)','SELECT vec_bolsa_llamamientos.registrar_disposicion_oferta_interna_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(text,text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,text)','SELECT vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text[],NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,text)','SELECT vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text[],NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text)'),
  ('vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.responder_llamamiento_portal_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text[],NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamp with time zone,timestamp with time zone,text[],text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','SELECT vec_bolsa_llamamientos.solicitar_portal_candidato_v1(NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.timestamptz,NULL::pg_catalog.text[],NULL::pg_catalog.text,NULL::pg_catalog.text,NULL::pg_catalog.timestamptz,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.numeric,NULL::pg_catalog.numeric,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea,NULL::pg_catalog.bytea)'),
  ('vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(text)','SELECT vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(NULL::pg_catalog.text)');
DO $conceder_sonda$
DECLARE f pg_catalog.text;
BEGIN
 FOR f IN SELECT firma FROM pg_temp.llamadas_sonda LOOP
  EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO prueba_portales_tipos',f);
 END LOOP;
END $conceder_sonda$;
GRANT SELECT ON TABLE pg_temp.llamadas_sonda TO prueba_portales_tipos;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- NUMERIC es un nombre de sistema en la gramática SQL de PostgreSQL 18.
-- Se conserva el marcador de copia de AD3-118 y se ensaya contra el dominio.
SELECT '1'::numeric;
RESET ROLE;
SET LOCAL SESSION AUTHORIZATION prueba_portales_tipos;
SELECT vec_autorizacion.login_candidato_externo_v1('prueba_portales_tipos','vec_autorizacion_registro_externo');
DO $despues$
DECLARE consulta pg_catalog.text;
BEGIN
 FOR consulta IN SELECT l.consulta FROM pg_temp.llamadas_sonda l LOOP
  BEGIN EXECUTE consulta; EXCEPTION WHEN OTHERS THEN NULL; END;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_temp.sonda_tipos) THEN
  RAISE EXCEPTION 'PORTALES-TIPOS: un propietario resolvió un dominio temporal';
 END IF;
END $despues$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
\echo PORTALES-TIPOS-CIERRE-OK
