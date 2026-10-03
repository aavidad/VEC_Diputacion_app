\set ON_ERROR_STOP on
-- B72: contactos asociados a un ofrecimiento publicado. «enviado» registra
-- la declaración de envío; «no_entregado», el rebote anotado. Ninguno prueba
-- entrega. «entrega_declarada» conserva la referencia y huella aportadas por
-- RRHH; no consulta SMTP ni acredita custodia o notificación legal.
-- Amplía la historia B13/B31 sin modificar filas anteriores. Cierra el
-- replay de contactos por oferta a través de las funciones antiguas.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000072',0));

DO $precondicion$
DECLARE r text; t regclass;
BEGIN
 t:=to_regclass('vec_bolsa_llamamientos.contacto_participacion');
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR t IS NULL
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[])') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_contactos_participacion_v1(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v3(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,text)') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_contactos_participacion_v2(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)') IS NOT NULL THEN
  RAISE EXCEPTION 'B72: preimagen incompatible (B13/B28/B31/B58 requeridas)' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO r FROM pg_constraint c
  WHERE c.conrelid=t AND c.conname='contacto_participacion_resultado_check' AND c.contype='c';
 IF r IS NULL OR strpos(r,'''enviado''')=0 OR strpos(r,'''no_enviado''')=0
    OR strpos(r,'''numero_erroneo''')=0 OR strpos(r,'''no_entregado''')=0 OR strpos(r,'''entrega_declarada''')<>0
    OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=t AND NOT attisdropped
              AND attname IN('oferta_ref','evidencia_ref','evidencia_huella'))
    OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=t AND relowner='vec_bolsa_llamamientos_propietario'::regrole
                   AND relrowsecurity AND relforcerowsecurity)
    OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
    OR NOT EXISTS(SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='contacto_participacion_solo_propietario'
                   AND polroles=ARRAY['vec_bolsa_llamamientos_propietario'::regrole::oid]
                   AND polcmd='*')
    OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid=t AND tgname='contacto_participacion_inmutable'
                   AND NOT tgisinternal AND tgenabled='O'
                   AND tgfoid='vec_bolsa_llamamientos.constitucion_rechazar_mutacion()'::regprocedure) THEN
  RAISE EXCEPTION 'B72: estructura de contactos incompatible' USING ERRCODE='55000';
 END IF;
END $precondicion$;
LOCK TABLE vec_bolsa_llamamientos.contacto_participacion IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 ADD COLUMN oferta_ref text REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 ADD COLUMN evidencia_ref text,
 ADD COLUMN evidencia_huella text;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN(
  'contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro',
  'enviado','no_enviado','numero_erroneo','no_entregado','entrega_declarada')),
 ADD CONSTRAINT contacto_participacion_oferta_check CHECK(oferta_ref IS NULL OR (
  oferta_ref ~ '^oferta:[0-9a-f]{64}$' AND llamamiento_ref IS NULL AND canal='correo'
  AND resultado IN('enviado','no_enviado','no_entregado','entrega_declarada'))),
 ADD CONSTRAINT contacto_participacion_evidencia_check CHECK(
  (resultado='entrega_declarada' AND oferta_ref IS NOT NULL
   AND evidencia_ref IS NOT NULL AND evidencia_huella IS NOT NULL
   AND octet_length(evidencia_ref) BETWEEN 1 AND 256
   AND evidencia_ref ~ '^[A-Za-z][A-Za-z0-9:_-]*$'
   AND strpos(evidencia_ref,'@')=0
   AND evidencia_ref !~* '(^|[:_-])(dni|nie|nif|pasaporte|passport)([:_-]|$)'
   AND evidencia_ref !~* '(([0-9][:_-]?){8}|[XYZ][:_-]?([0-9][:_-]?){7})[A-Z]'
   AND evidencia_huella ~ '^[0-9a-f]{64}$')
  OR (resultado<>'entrega_declarada' AND evidencia_ref IS NULL AND evidencia_huella IS NULL));
CREATE INDEX contacto_participacion_oferta_fecha
 ON vec_bolsa_llamamientos.contacto_participacion(bolsa_ref,oferta_ref,instante DESC,contacto_ref DESC)
 WHERE oferta_ref IS NOT NULL;
CREATE INDEX contacto_participacion_oferta_participacion_fecha
 ON vec_bolsa_llamamientos.contacto_participacion(bolsa_ref,oferta_ref,participacion_ref,instante DESC,contacto_ref DESC)
 WHERE oferta_ref IS NOT NULL;

-- Las versiones anteriores no reciben oferta/evidencia. Su replay debe
-- rechazar filas B72, conservando el resto de su definición viva y sus ACL.
DO $cerrar_replay_legacy$
DECLARE f regprocedure; original text; cuerpo text; esperada text; actual text; metadata jsonb;
 funciones regprocedure[]:=ARRAY[
  'vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[])'::regprocedure];
 marca text:=$marca$ IF FOUND THEN
  IF anterior.bolsa_ref<>p_bolsa_ref$marca$;
 guarda text:=$guarda$ IF FOUND THEN
  IF anterior.oferta_ref IS NOT NULL THEN
   RAISE EXCEPTION 'B72: contacto de oferta requiere versión tres' USING ERRCODE='VBC01';
  END IF;
  IF anterior.bolsa_ref<>p_bolsa_ref$guarda$;
BEGIN
 FOREACH f IN ARRAY funciones LOOP
  SELECT pg_get_functiondef(p.oid),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,cuerpo,metadata
   FROM pg_proc p WHERE p.oid=f;
  IF (length(cuerpo)-length(replace(cuerpo,marca,'')))<>length(marca)
     OR strpos(cuerpo,'anterior.oferta_ref')<>0
     OR (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_bolsa_llamamientos_propietario'::regrole
     OR NOT (SELECT prosecdef FROM pg_proc WHERE oid=f) THEN
   RAISE EXCEPTION 'B72: preimagen de replay incompatible' USING ERRCODE='55000';
  END IF;
  esperada:=replace(original,marca,guarda);
  -- pg_get_functiondef emite CREATE OR REPLACE sobre la definición instalada.
  EXECUTE esperada;
  SELECT pg_get_functiondef(p.oid) INTO STRICT actual FROM pg_proc p WHERE p.oid=f;
  IF actual IS DISTINCT FROM esperada OR replace(actual,guarda,marca) IS DISTINCT FROM original
     OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata THEN
   RAISE EXCEPTION 'B72: cambió metadata o cuerpo fuera de la guarda de replay' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $cerrar_replay_legacy$;

-- Contrato dedicado sin parámetros de control telefónico. Los 21 argumentos
-- B13 se conservan en su orden; oferta y evidencia se añaden al final.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v3(
 p_contacto_ref text,p_bolsa_ref text,p_participacion_ref text,p_llamamiento_ref text,
 p_canal text,p_instante timestamptz,p_actor text,p_resultado text,p_anotacion text,p_clave text,p_recibo text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_oferta_ref text,p_evidencia_ref text,p_evidencia_huella text)
RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE consumo record; d jsonb; anterior vec_bolsa_llamamientos.contacto_participacion%ROWTYPE;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_contacto_ref IS NULL OR p_contacto_ref<>btrim(p_contacto_ref) OR octet_length(p_contacto_ref) NOT BETWEEN 1 AND 256
    OR p_bolsa_ref IS NULL OR p_bolsa_ref<>btrim(p_bolsa_ref) OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 256
    OR p_participacion_ref IS NULL OR p_participacion_ref<>btrim(p_participacion_ref) OR octet_length(p_participacion_ref) NOT BETWEEN 1 AND 256
    OR p_canal IS NULL OR p_canal NOT IN('telefono','correo','sms','presencial','otro')
    OR p_instante IS NULL OR NOT isfinite(p_instante)
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_resultado IS NULL OR p_resultado NOT IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro','enviado','no_enviado','numero_erroneo','no_entregado','entrega_declarada')
    OR p_anotacion IS NULL OR p_anotacion<>btrim(p_anotacion) OR octet_length(p_anotacion) NOT BETWEEN 1 AND 1000
    OR p_anotacion ~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email|teléfono)([^[:alpha:]]|$)' OR strpos(p_anotacion,'@')<>0
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 1 AND 256
    OR p_recibo IS NULL OR p_recibo<>btrim(p_recibo) OR octet_length(p_recibo) NOT BETWEEN 1 AND 256
    OR (p_llamamiento_ref IS NOT NULL AND (p_llamamiento_ref<>btrim(p_llamamiento_ref) OR octet_length(p_llamamiento_ref) NOT BETWEEN 1 AND 256))
    OR (p_oferta_ref IS NOT NULL AND (p_oferta_ref !~ '^oferta:[0-9a-f]{64}$' OR p_llamamiento_ref IS NOT NULL
       OR p_canal<>'correo' OR p_resultado NOT IN('enviado','no_enviado','no_entregado','entrega_declarada')))
    OR (p_resultado='entrega_declarada' AND (p_oferta_ref IS NULL OR p_evidencia_ref IS NULL OR p_evidencia_huella IS NULL
       OR octet_length(p_evidencia_ref) NOT BETWEEN 1 AND 256 OR p_evidencia_ref !~ '^[A-Za-z][A-Za-z0-9:_-]*$'
       OR strpos(p_evidencia_ref,'@')<>0
       OR p_evidencia_ref ~* '(^|[:_-])(dni|nie|nif|pasaporte|passport)([:_-]|$)'
       OR p_evidencia_ref ~* '(([0-9][:_-]?){8}|[XYZ][:_-]?([0-9][:_-]?){7})[A-Z]'
       OR p_evidencia_huella !~ '^[0-9a-f]{64}$'))
    OR (p_resultado<>'entrega_declarada' AND (p_evidencia_ref IS NOT NULL OR p_evidencia_huella IS NOT NULL)) THEN
  RAISE EXCEPTION 'B72: contacto inválido' USING ERRCODE='22023';
 END IF;
 -- Autorizar también el replay: la clave o el recibo históricos no dan acceso.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B72: contacto no autorizado' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion'
    OR d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'B72: contacto no autorizado' USING ERRCODE='42501';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
               JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
               WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B72: participación ajena a la bolsa' USING ERRCODE='23503';
 END IF;
 IF p_oferta_ref IS NOT NULL AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.oferta_publicada o
                                           WHERE o.oferta_ref=p_oferta_ref AND o.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B72: oferta ajena a la bolsa' USING ERRCODE='23503';
 END IF;
 IF p_llamamiento_ref IS NOT NULL AND NOT EXISTS(
  SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
  JOIN vec_bolsa_llamamientos.integracion_desarrollo o USING(operacion_ref)
  WHERE l.llamamiento_ref=p_llamamiento_ref AND l.bolsa_ref=p_bolsa_ref
   AND convert_from(o.registro_canonico,'UTF8')::jsonb#>>'{propuesta,participacion_seleccionada_ref}'=p_participacion_ref) THEN
  RAISE EXCEPTION 'B72: llamamiento ajeno' USING ERRCODE='23503';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_bolsa_llamamientos:contacto:'||p_participacion_ref||chr(31)||p_clave,0));
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.contacto_participacion c
  WHERE c.participacion_ref=p_participacion_ref AND c.clave_idempotencia=p_clave FOR SHARE;
 IF FOUND THEN
  IF anterior.contacto_ref IS DISTINCT FROM p_contacto_ref OR anterior.recibo_ref IS DISTINCT FROM p_recibo
     OR anterior.bolsa_ref IS DISTINCT FROM p_bolsa_ref OR anterior.llamamiento_ref IS DISTINCT FROM p_llamamiento_ref
     OR anterior.canal IS DISTINCT FROM p_canal OR anterior.instante IS DISTINCT FROM p_instante
     OR anterior.actor IS DISTINCT FROM p_actor OR anterior.resultado IS DISTINCT FROM p_resultado
     OR anterior.anotacion IS DISTINCT FROM p_anotacion OR anterior.oferta_ref IS DISTINCT FROM p_oferta_ref
     OR anterior.evidencia_ref IS DISTINCT FROM p_evidencia_ref OR anterior.evidencia_huella IS DISTINCT FROM p_evidencia_huella THEN
   RAISE EXCEPTION 'B72: clave idempotente divergente' USING ERRCODE='VBC01';
  END IF;
  RETURN QUERY SELECT true,anterior.recibo_ref,anterior.contacto_ref; RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
  contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref,
  oferta_ref,evidencia_ref,evidencia_huella)
 VALUES(p_contacto_ref,p_bolsa_ref,p_participacion_ref,p_llamamiento_ref,p_canal,p_instante,p_actor,p_resultado,p_anotacion,p_clave,p_recibo,
  p_oferta_ref,p_evidencia_ref,p_evidencia_huella);
 RETURN QUERY SELECT false,p_recibo,p_contacto_ref;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v2(
 p_bolsa_ref text,p_participacion_ref text,p_cursor text,p_limite integer,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_oferta_ref text)
RETURNS TABLE(contacto_ref text,bolsa_ref text,participacion_ref text,llamamiento_ref text,canal text,
 instante timestamptz,actor text,resultado text,anotacion text,oferta_ref text,evidencia_ref text,evidencia_huella text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE consumo record; d jsonb; v_cursor_instante timestamptz;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa_ref IS NULL OR p_bolsa_ref<>btrim(p_bolsa_ref) OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 256
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
    OR (p_participacion_ref IS NOT NULL AND (p_participacion_ref<>btrim(p_participacion_ref) OR octet_length(p_participacion_ref) NOT BETWEEN 1 AND 256))
    OR (p_cursor IS NOT NULL AND (p_cursor<>btrim(p_cursor) OR octet_length(p_cursor) NOT BETWEEN 1 AND 256))
    OR (p_oferta_ref IS NOT NULL AND p_oferta_ref !~ '^oferta:[0-9a-f]{64}$') THEN
  RAISE EXCEPTION 'B72: consulta contacto inválida' USING ERRCODE='22023';
 END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B72: consulta contacto no autorizada' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM coalesce(p_participacion_ref,p_bolsa_ref) OR consumo.consumo_nuevo IS NOT TRUE
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
    OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.consultar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'consulta_contactos_participacion'
    OR d->>'recurso_ref' IS DISTINCT FROM coalesce(p_participacion_ref,p_bolsa_ref)
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'B72: consulta contacto no autorizada' USING ERRCODE='42501';
 END IF;
 IF p_oferta_ref IS NOT NULL AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.oferta_publicada o
                                           WHERE o.oferta_ref=p_oferta_ref AND o.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B72: oferta ajena a la bolsa' USING ERRCODE='23503';
 END IF;
 IF p_participacion_ref IS NOT NULL AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
       JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
       WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN
  RAISE EXCEPTION 'B72: participación ajena a la bolsa' USING ERRCODE='23503';
 END IF;
 -- El cursor debe pertenecer al conjunto autorizado y a los mismos filtros.
 -- Nunca usa un contacto ajeno para fijar el punto de corte de la consulta.
 IF p_cursor IS NOT NULL THEN
  SELECT c.instante INTO v_cursor_instante FROM vec_bolsa_llamamientos.contacto_participacion c
   WHERE c.contacto_ref=p_cursor AND c.bolsa_ref=p_bolsa_ref
     AND (p_participacion_ref IS NULL OR c.participacion_ref=p_participacion_ref)
     AND (p_oferta_ref IS NULL OR c.oferta_ref=p_oferta_ref);
  IF NOT FOUND THEN RAISE EXCEPTION 'B72: cursor ajeno a la consulta' USING ERRCODE='22023'; END IF;
 END IF;
 RETURN QUERY SELECT c.contacto_ref,c.bolsa_ref,c.participacion_ref,c.llamamiento_ref,c.canal,c.instante,c.actor,
  c.resultado,c.anotacion,c.oferta_ref,c.evidencia_ref,c.evidencia_huella
 FROM vec_bolsa_llamamientos.contacto_participacion c
 WHERE c.bolsa_ref=p_bolsa_ref AND (p_participacion_ref IS NULL OR c.participacion_ref=p_participacion_ref)
   AND (p_oferta_ref IS NULL OR c.oferta_ref=p_oferta_ref)
   AND (p_cursor IS NULL OR (c.instante,c.contacto_ref)<(v_cursor_instante,p_cursor))
 ORDER BY c.instante DESC,c.contacto_ref DESC LIMIT p_limite;
END $f$;

DO $acl$
DECLARE f regprocedure; t regclass:='vec_bolsa_llamamientos.contacto_participacion'::regclass;
 funciones regprocedure[]:=ARRAY[
  'vec_bolsa_llamamientos.registrar_contacto_participacion_v3(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,text)'::regprocedure,
  'vec_bolsa_llamamientos.listar_contactos_participacion_v2(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)'::regprocedure];
BEGIN
 FOREACH f IN ARRAY funciones LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor',f::text);
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
             WHERE p.oid=f AND a.grantee<>p.proowner
               AND (a.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole OR a.is_grantable OR a.privilege_type<>'EXECUTE'))
     OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_bolsa_llamamientos_propietario'::regrole
                   AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog','lock_timeout=2s','statement_timeout=15s']) THEN
   RAISE EXCEPTION 'B72: ACL de función incorrecta' USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
            WHERE c.oid=t AND a.grantee<>c.relowner)
    OR EXISTS(SELECT 1 FROM pg_attribute p CROSS JOIN LATERAL aclexplode(p.attacl) a
              WHERE p.attrelid=t AND NOT p.attisdropped AND a.grantee<>'vec_bolsa_llamamientos_propietario'::regrole)
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
  RAISE EXCEPTION 'B72: privilegios de tabla incorrectos' USING ERRCODE='55000';
 END IF;
END $acl$;
COMMIT;
