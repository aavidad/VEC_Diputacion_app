\set ON_ERROR_STOP on
-- B87: registro real del intento telefónico. Conserva B13/B31/B72 y su
-- histórico; sólo la función nueva escribe instante_servidor=true.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000087',0));

DO $pre$
DECLARE t oid:=pg_catalog.to_regclass('vec_bolsa_llamamientos.contacto_participacion');
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR t IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_participacion_v2(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[])') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_contactos_participacion_v2(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1(text,text,text,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,integer,integer,boolean,text[],text,integer,integer,boolean,text,date,boolean)') IS NOT NULL
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid=t AND attname='instante_servidor' AND NOT attisdropped)
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c WHERE c.conrelid=t
      AND c.conname='contacto_participacion_anotacion_check' AND c.contype='c'
      AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,true),'1000')>0
      AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,true),'anotacion')>0)
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c WHERE c.conrelid=t
      AND c.conname='contacto_participacion_resultado_check' AND c.contype='c'
      AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,true),'''entrega_declarada''')>0
      AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,true),'''comunica''')=0)
 THEN RAISE EXCEPTION 'B87: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

LOCK TABLE vec_bolsa_llamamientos.contacto_participacion IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 ADD COLUMN instante_servidor boolean NOT NULL DEFAULT false;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 DROP CONSTRAINT contacto_participacion_anotacion_check,
 DROP CONSTRAINT contacto_participacion_resultado_check;
ALTER TABLE vec_bolsa_llamamientos.contacto_participacion
 ADD CONSTRAINT contacto_participacion_anotacion_check CHECK(
  anotacion=pg_catalog.btrim(anotacion)
  AND pg_catalog.octet_length(anotacion)<=1000
  AND (pg_catalog.octet_length(anotacion)>=1 OR
       (instante_servidor AND canal='telefono' AND llamamiento_ref IS NOT NULL AND oferta_ref IS NULL))
  AND anotacion !~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email|teléfono)([^[:alpha:]]|$)'
  AND pg_catalog.strpos(anotacion,'@')=0),
 ADD CONSTRAINT contacto_participacion_resultado_check CHECK(resultado IN(
  'contactado','comunica','no_contesta','buzon','acepta','rechaza','aplazado','otro',
  'enviado','no_enviado','numero_erroneo','no_entregado','entrega_declarada')),
 ADD CONSTRAINT contacto_participacion_origen_servidor_check CHECK(
  NOT instante_servidor OR
  (canal='telefono' AND llamamiento_ref IS NOT NULL AND oferta_ref IS NULL)),
 ADD CONSTRAINT contacto_participacion_comunica_check CHECK(
  resultado<>'comunica' OR
  (instante_servidor AND canal='telefono' AND llamamiento_ref IS NOT NULL AND oferta_ref IS NULL));

CREATE FUNCTION vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1(
 p_contacto_ref text,p_bolsa_ref text,p_participacion_ref text,p_llamamiento_ref text,
 p_actor text,p_resultado text,p_anotacion text,p_clave text,p_recibo text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea,p_maximo_intentos integer,p_separacion_segundos integer,
 p_impedir_separacion boolean,p_resultados_sin_contacto text[],p_zona text,
 p_desde_minuto integer,p_hasta_minuto integer,p_solo_dias_habiles boolean,
 p_control_franja text,p_fecha_habil date,p_dia_habil boolean)
RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text,instante timestamptz,
 previos_sin_contacto integer,previo_contactado boolean,previo_ultimo timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE consumo record; d jsonb; anterior vec_bolsa_llamamientos.contacto_participacion%ROWTYPE;
 v_instante timestamptz; v_sin integer:=0; v_contactado boolean:=false; v_ultimo timestamptz;
 v_control boolean; v_local timestamp; v_minuto integer;
 v_resultados constant text[]:=ARRAY['contactado','comunica','no_contesta','buzon','acepta','rechaza','aplazado','numero_erroneo'];
 v_catalogo constant text[]:=ARRAY['contactado','comunica','no_contesta','buzon','acepta','rechaza','aplazado','otro','numero_erroneo','no_entregado'];
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_contacto_ref IS NULL OR p_contacto_ref !~ '^contacto:[0-9a-f]{64}$'
    OR p_bolsa_ref IS NULL OR p_bolsa_ref='' OR p_participacion_ref IS NULL
    OR p_participacion_ref='' OR p_llamamiento_ref IS NULL OR p_llamamiento_ref=''
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_resultado IS NULL OR NOT p_resultado=ANY(v_resultados)
    OR p_anotacion IS NULL OR p_anotacion<>pg_catalog.btrim(p_anotacion)
    OR pg_catalog.octet_length(p_anotacion)>1000
    OR p_clave IS NULL OR p_clave<>pg_catalog.btrim(p_clave)
    OR pg_catalog.octet_length(p_clave) NOT BETWEEN 1 AND 256
    OR p_recibo IS DISTINCT FROM 'recibo:'||p_contacto_ref THEN
   RAISE EXCEPTION 'B87: contacto inválido' USING ERRCODE='22023';
 END IF;
 v_control:=p_maximo_intentos IS NOT NULL;
 IF (v_control AND (p_maximo_intentos NOT BETWEEN 1 AND 100
      OR p_separacion_segundos IS NULL OR p_separacion_segundos NOT BETWEEN 0 AND 604800
      OR p_impedir_separacion IS NULL OR p_resultados_sin_contacto IS NULL
      OR pg_catalog.cardinality(p_resultados_sin_contacto) NOT BETWEEN 1 AND 16
      OR pg_catalog.array_position(p_resultados_sin_contacto,NULL) IS NOT NULL
      OR NOT p_resultados_sin_contacto<@v_catalogo))
    OR (NOT v_control AND (p_separacion_segundos IS NOT NULL OR p_impedir_separacion IS NOT NULL
      OR p_resultados_sin_contacto IS NOT NULL OR p_zona IS NOT NULL OR p_desde_minuto IS NOT NULL
      OR p_hasta_minuto IS NOT NULL OR p_solo_dias_habiles IS NOT NULL OR p_control_franja IS NOT NULL
      OR p_fecha_habil IS NOT NULL OR p_dia_habil IS NOT NULL))
    OR (p_zona IS NULL AND (p_desde_minuto IS NOT NULL OR p_hasta_minuto IS NOT NULL
      OR p_solo_dias_habiles IS NOT NULL OR p_control_franja IS NOT NULL
      OR p_fecha_habil IS NOT NULL OR p_dia_habil IS NOT NULL))
    OR (p_zona IS NOT NULL AND (NOT v_control OR p_desde_minuto IS NULL OR p_hasta_minuto IS NULL
      OR p_desde_minuto NOT BETWEEN 0 AND 1439 OR p_hasta_minuto NOT BETWEEN 1 AND 1440
      OR p_desde_minuto>=p_hasta_minuto OR p_solo_dias_habiles IS NULL
      OR p_control_franja IS NULL OR p_control_franja NOT IN('impedir','advertir')
      OR (p_solo_dias_habiles AND (p_fecha_habil IS NULL OR p_dia_habil IS NULL))
      OR (NOT p_solo_dias_habiles AND (p_fecha_habil IS NOT NULL OR p_dia_habil IS NOT NULL))
      OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_timezone_names z WHERE z.name=p_zona))) THEN
   RAISE EXCEPTION 'B87: control inválido' USING ERRCODE='22023';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e
   JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea)
   WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN
   RAISE EXCEPTION 'B87: participación ajena' USING ERRCODE='23503';
 END IF;
 -- La llamada sigue a un llamamiento de la propia bolsa en el que está la
 -- persona: el de integración (persona seleccionada) o el emitido por el
 -- asistente, siempre que su aviso por correo ya conste (canales
 -- complementarios: el teléfono no sustituye ni adelanta la emisión).
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
   JOIN vec_bolsa_llamamientos.integracion_desarrollo o USING(operacion_ref)
   WHERE l.llamamiento_ref=p_llamamiento_ref AND l.bolsa_ref=p_bolsa_ref
     AND pg_catalog.convert_from(o.registro_canonico,'UTF8')::jsonb#>>'{propuesta,participacion_seleccionada_ref}'=p_participacion_ref)
  AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_emitido l
   CROSS JOIN LATERAL pg_catalog.jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality)
   JOIN vec_bolsa_llamamientos.contacto_participacion c
     ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref AND c.canal='correo'
    AND c.clave_idempotencia=l.clave_idempotencia||':correo:'||x.ordinality
   WHERE l.llamamiento_ref=p_llamamiento_ref AND l.bolsa_ref=p_bolsa_ref AND x.ref=p_participacion_ref) THEN
   RAISE EXCEPTION 'B87: llamamiento ajeno' USING ERRCODE='23503';
 END IF;

 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:contacto:'||p_participacion_ref||pg_catalog.chr(31)||p_clave,0));
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:intentos:'||p_participacion_ref||pg_catalog.chr(31)||p_llamamiento_ref,0));
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B87: no autorizado' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion'
    OR d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
   RAISE EXCEPTION 'B87: no autorizado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.contacto_participacion c
  WHERE c.participacion_ref=p_participacion_ref AND c.clave_idempotencia=p_clave FOR SHARE;
 IF FOUND THEN
  IF NOT anterior.instante_servidor
     OR anterior.contacto_ref IS DISTINCT FROM p_contacto_ref
     OR anterior.recibo_ref IS DISTINCT FROM p_recibo
     OR anterior.bolsa_ref<>p_bolsa_ref
     OR anterior.llamamiento_ref IS DISTINCT FROM p_llamamiento_ref
     OR anterior.canal<>'telefono' OR anterior.actor<>p_actor
     OR anterior.resultado<>p_resultado OR anterior.anotacion<>p_anotacion
     OR anterior.oferta_ref IS NOT NULL THEN
   RAISE EXCEPTION 'B87: clave idempotente divergente' USING ERRCODE='VBC01';
  END IF;
 ELSE
  v_instante:=pg_catalog.clock_timestamp();
  IF p_zona IS NOT NULL THEN
   v_local:=v_instante AT TIME ZONE p_zona;
   v_minuto:=extract(hour FROM v_local)::integer*60
            +extract(minute FROM v_local)::integer;
   IF p_solo_dias_habiles AND v_local::date<>p_fecha_habil THEN
    RAISE EXCEPTION 'B87: fecha de calendario obsoleta' USING ERRCODE='VBC04';
   END IF;
   IF p_control_franja='impedir' AND
      (v_minuto<p_desde_minuto OR v_minuto>=p_hasta_minuto
       OR (p_solo_dias_habiles AND NOT p_dia_habil)) THEN
    RAISE EXCEPTION 'B87: intento fuera de franja' USING ERRCODE='VBC05';
   END IF;
  END IF;
 END IF;
 IF v_control THEN
  SELECT pg_catalog.count(*) FILTER (WHERE c.resultado=ANY(p_resultados_sin_contacto))::integer,
    pg_catalog.coalesce(pg_catalog.bool_or(NOT c.resultado=ANY(p_resultados_sin_contacto)),false),
    pg_catalog.max(c.instante)
   INTO v_sin,v_contactado,v_ultimo
   FROM vec_bolsa_llamamientos.contacto_participacion c
   WHERE c.participacion_ref=p_participacion_ref AND c.llamamiento_ref=p_llamamiento_ref
    AND c.canal='telefono' AND c.contacto_ref<>p_contacto_ref;
 END IF;
 IF anterior.contacto_ref IS NOT NULL THEN
  RETURN QUERY SELECT true,anterior.recibo_ref,anterior.contacto_ref,anterior.instante,
   v_sin,v_contactado,v_ultimo;
  RETURN;
 END IF;
 IF v_control AND NOT v_contactado THEN
  IF v_sin>=p_maximo_intentos THEN
   RAISE EXCEPTION 'B87: intentos agotados' USING ERRCODE='VBC03';
  END IF;
  IF p_impedir_separacion AND v_ultimo IS NOT NULL
     AND pg_catalog.abs(extract(epoch FROM v_instante-v_ultimo))<p_separacion_segundos THEN
   RAISE EXCEPTION 'B87: antes de separación' USING ERRCODE='VBC02';
  END IF;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(
  contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,
  anotacion,clave_idempotencia,recibo_ref,instante_servidor)
 VALUES(p_contacto_ref,p_bolsa_ref,p_participacion_ref,p_llamamiento_ref,'telefono',
  v_instante,p_actor,p_resultado,p_anotacion,p_clave,p_recibo,true)
 RETURNING contacto_participacion.instante INTO v_instante;
 RETURN QUERY SELECT false,p_recibo,p_contacto_ref,v_instante,v_sin,v_contactado,v_ultimo;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1(
 text,text,text,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,
 bytea,bytea,bytea,bytea,integer,integer,boolean,text[],text,integer,integer,boolean,text,date,boolean) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_telefonico_actual_v1(
 text,text,text,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,
 bytea,bytea,bytea,bytea,integer,integer,boolean,text[],text,integer,integer,boolean,text,date,boolean)
 TO vec_bolsa_llamamientos_ejecutor;

-- Las funciones históricas conservan sus cuerpos. Su adaptador consulta esta
-- guarda dentro de la TX antes de confirmar un replay y rechaza una fila B87
-- aunque el cliente acertase el instante original al repetir la clave.
CREATE FUNCTION vec_bolsa_llamamientos.verificar_replay_contacto_legado_v1(
 p_participacion_ref text,p_clave text) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE v_servidor boolean;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_participacion_ref IS NULL
    OR p_clave IS NULL THEN
  RAISE EXCEPTION 'B87: replay inválido' USING ERRCODE='22023';
 END IF;
 SELECT c.instante_servidor INTO v_servidor
 FROM vec_bolsa_llamamientos.contacto_participacion c
 WHERE c.participacion_ref=p_participacion_ref AND c.clave_idempotencia=p_clave;
 IF v_servidor IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'B87: clave idempotente divergente' USING ERRCODE='VBC01';
 END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.verificar_replay_contacto_legado_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.verificar_replay_contacto_legado_v1(text,text)
 TO vec_bolsa_llamamientos_ejecutor;
-- B86 (director, P2 de #879): una llamada sobre un llamamiento emitido ya no
-- recalcula ni serializa la completitud del correo. Cuerpo instalado literal
-- con una sola condición añadida; misma firma, propietario y ACL.
DO $pre86$
BEGIN
 IF pg_catalog.md5((SELECT p.prosrc FROM pg_catalog.pg_proc p
      WHERE p.oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1()')))
    IS DISTINCT FROM '7fec063d31b7cae5026c3fa28db3d402' THEN
  RAISE EXCEPTION 'B87: preimagen B86 incompatible' USING ERRCODE='55000';
 END IF;
END $pre86$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1()
 RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$

DECLARE v_ref text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR TG_OP<>'INSERT' OR TG_LEVEL<>'STATEMENT'
    OR TG_RELID NOT IN ('vec_bolsa_llamamientos.llamamiento_emitido'::pg_catalog.regclass,
      'vec_bolsa_llamamientos.contacto_participacion'::pg_catalog.regclass)
 THEN RAISE EXCEPTION 'B86: origen de proyeccion incompatible' USING ERRCODE='42501'; END IF;

 FOR v_ref IN
   SELECT DISTINCT n.llamamiento_ref FROM nuevas_completitud n
   WHERE n.llamamiento_ref ~ '^llamamiento:[0-9a-f]{64}$'
     -- B87: solo el correo completa la emisión; una llamada no serializa.
     AND (TG_RELID='vec_bolsa_llamamientos.llamamiento_emitido'::pg_catalog.regclass
          OR pg_catalog.to_jsonb(n)->>'canal'='correo')
   ORDER BY n.llamamiento_ref
 LOOP
   -- Una escritura real serializa las carreras: RC relee tras esperar;
   -- RR/SER aborta la transaccion fuente si su instantanea es obsoleta.
   INSERT INTO vec_bolsa_llamamientos.coordinacion_completitud_correo(llamamiento_ref,revision)
     VALUES(v_ref,1)
   ON CONFLICT (llamamiento_ref) DO UPDATE
     SET revision=vec_bolsa_llamamientos.coordinacion_completitud_correo.revision+1;

   -- Sentencia VOLATILE separada del UPSERT: conserva exactamente B17.
   INSERT INTO vec_bolsa_llamamientos.completitud_correo_llamamiento(llamamiento_ref,bolsa_ref)
   SELECT l.llamamiento_ref,l.bolsa_ref
   FROM vec_bolsa_llamamientos.llamamiento_emitido l
   WHERE l.llamamiento_ref=v_ref
     AND (SELECT pg_catalog.count(*)
       FROM pg_catalog.jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality)
       JOIN vec_bolsa_llamamientos.contacto_participacion c
         ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref
        AND c.canal='correo'
        AND c.clave_idempotencia=l.clave_idempotencia||':correo:'||x.ordinality
        AND c.recibo_ref='recibo:contacto:'||pg_catalog.encode(pg_catalog.sha256(
          pg_catalog.convert_to(l.bolsa_ref||pg_catalog.chr(31)||l.clave_idempotencia||
            pg_catalog.chr(31)||x.ref,'UTF8')),'hex'))
       =pg_catalog.jsonb_array_length(l.participaciones)
   ON CONFLICT (llamamiento_ref) DO NOTHING;
 END LOOP;
 RETURN NULL;
END 
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1() FROM PUBLIC;
COMMIT;
