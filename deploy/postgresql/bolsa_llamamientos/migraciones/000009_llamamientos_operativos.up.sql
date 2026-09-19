\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000009', 0));

-- C23. Las direcciones no se copian a Bolsa: VEC conserva el correo en su
-- alta. Esta tabla retiene únicamente el índice opaco que permite a la
-- composición preguntar a aquella autoridad antes de un despacho.
CREATE TABLE vec_bolsa_llamamientos.llamamiento_operativo (
    llamamiento_ref text PRIMARY KEY,
    operacion_apertura_ref text NOT NULL UNIQUE,
    participacion_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.vinculo_candidato(participacion_ref),
    actor_apertura_ref text NOT NULL,
    canal text NOT NULL CHECK (canal IN ('correo')),
    comunicado_en timestamptz(6) NOT NULL,
    plazo_respuesta_hasta timestamptz(6) NOT NULL CHECK (comunicado_en < plazo_respuesta_hasta),
    anotacion text NOT NULL DEFAULT '' CHECK (octet_length(anotacion) <= 512 AND anotacion = btrim(anotacion)),
    estado text NOT NULL DEFAULT 'abierto' CHECK (estado IN ('abierto','aceptado','renuncia','sin_respuesta')),
    resultado_operacion_ref text UNIQUE,
    actor_resultado_ref text,
    resultado_registrado_en timestamptz(6),
    recibo_ref text NOT NULL UNIQUE,
    confirmado_en timestamptz(6) NOT NULL,
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(llamamiento_ref,512)),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(operacion_apertura_ref,512)),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(participacion_ref,512)),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(actor_apertura_ref,512)),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(recibo_ref,512)),
    CHECK ((estado='abierto') = (resultado_operacion_ref IS NULL)),
    CHECK ((resultado_operacion_ref IS NULL) = (actor_resultado_ref IS NULL)),
    CHECK ((resultado_operacion_ref IS NULL) = (resultado_registrado_en IS NULL))
);
CREATE TABLE vec_bolsa_llamamientos.situacion_participacion_llamamiento (
    participacion_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.vinculo_candidato(participacion_ref),
    secuencia bigint NOT NULL CHECK (secuencia > 0),
    estado text NOT NULL CHECK (estado IN ('disponible','ocupado','renuncia_pendiente')),
    llamamiento_ref text REFERENCES vec_bolsa_llamamientos.llamamiento_operativo(llamamiento_ref),
    operacion_ref text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (participacion_ref,secuencia),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(operacion_ref,512))
);
CREATE TABLE vec_bolsa_llamamientos.auditoria_llamamiento_operativo (
    auditoria_ref text PRIMARY KEY,
    llamamiento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.llamamiento_operativo(llamamiento_ref),
    operacion_ref text NOT NULL UNIQUE,
    accion text NOT NULL CHECK (accion IN ('apertura','resultado')),
    actor_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(auditoria_ref,512))
);
CREATE TABLE vec_bolsa_llamamientos.outbox_llamamiento_operativo (
    evento_ref text PRIMARY KEY,
    llamamiento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.llamamiento_operativo(llamamiento_ref),
    operacion_ref text NOT NULL UNIQUE,
    tipo text NOT NULL,
    emitido_en timestamptz(6) NOT NULL,
    CHECK (tipo IN ('bolsa.llamamiento.abierto.v1','bolsa.llamamiento.resultado.v1')),
    CHECK (vec_bolsa_llamamientos.constitucion_texto_valido(evento_ref,512))
);

DO $p$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['llamamiento_operativo','situacion_participacion_llamamiento','auditoria_llamamiento_operativo','outbox_llamamiento_operativo'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY solo_propietario_c23 ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user=%L) WITH CHECK (current_user=%L)',t,'vec_bolsa_llamamientos_propietario','vec_bolsa_llamamientos_propietario');
  EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC',t);
 END LOOP;
END $p$;

CREATE FUNCTION vec_bolsa_llamamientos.contactos_participacion_v1(p_participacion_ref text)
RETURNS TABLE(candidato_ref text, canal text, disponible boolean)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT v.candidato_ref, 'correo'::text, false
 FROM vec_bolsa_llamamientos.vinculo_candidato v
 WHERE v.participacion_ref=p_participacion_ref
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.contactos_participacion_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.abrir_llamamiento_operativo_v1(p_operacion text,p_llamamiento text,p_participacion text,p_actor text,p_canal text,p_comunicado timestamptz,p_plazo timestamptz,p_anotacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE e vec_bolsa_llamamientos.llamamiento_operativo%ROWTYPE; v_ahora timestamptz(6):=clock_timestamp(); v_recibo text:='recibo:bolsa:'||p_operacion; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_canal<>'correo' OR p_comunicado IS NULL OR p_plazo IS NULL OR p_comunicado>=p_plazo OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_operacion,512) OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_llamamiento,512) OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_participacion,512) OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_actor,512) OR p_anotacion IS NULL OR octet_length(p_anotacion)>512 OR p_anotacion<>btrim(p_anotacion) THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='llamamiento invalido'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:llamamiento:'||p_operacion,0));
 SELECT * INTO e FROM vec_bolsa_llamamientos.llamamiento_operativo WHERE operacion_apertura_ref=p_operacion;
 IF FOUND THEN
  IF e.llamamiento_ref<>p_llamamiento OR e.participacion_ref<>p_participacion OR e.actor_apertura_ref<>p_actor OR e.canal<>p_canal OR e.comunicado_en<>p_comunicado OR e.plazo_respuesta_hasta<>p_plazo OR e.anotacion<>p_anotacion THEN RAISE EXCEPTION USING ERRCODE='PBL23',MESSAGE='operacion distinta'; END IF;
  RETURN jsonb_build_object('reutilizado',true,'llamamiento_ref',e.llamamiento_ref,'participacion_ref',e.participacion_ref,'estado',e.estado,'recibo_ref',e.recibo_ref,'confirmado_en',e.confirmado_en);
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato WHERE participacion_ref=p_participacion) THEN RAISE EXCEPTION USING ERRCODE='23503',MESSAGE='participacion inexistente'; END IF;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_operativo(llamamiento_ref,operacion_apertura_ref,participacion_ref,actor_apertura_ref,canal,comunicado_en,plazo_respuesta_hasta,anotacion,recibo_ref,confirmado_en) VALUES(p_llamamiento,p_operacion,p_participacion,p_actor,p_canal,p_comunicado,p_plazo,p_anotacion,v_recibo,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.auditoria_llamamiento_operativo VALUES('auditoria:bolsa:'||p_operacion,p_llamamiento,p_operacion,'apertura',p_actor,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.outbox_llamamiento_operativo VALUES('evento:bolsa:'||p_operacion,p_llamamiento,p_operacion,'bolsa.llamamiento.abierto.v1',v_ahora);
 RETURN jsonb_build_object('reutilizado',false,'llamamiento_ref',p_llamamiento,'participacion_ref',p_participacion,'estado','abierto','recibo_ref',v_recibo,'confirmado_en',v_ahora);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.abrir_llamamiento_operativo_v1(text,text,text,text,text,timestamptz,timestamptz,text) FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_resultado_llamamiento_operativo_v1(p_operacion text,p_llamamiento text,p_actor text,p_resultado text,p_registrado timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE e vec_bolsa_llamamientos.llamamiento_operativo%ROWTYPE; v_ahora timestamptz(6):=clock_timestamp(); v_estado text; v_situacion text; BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_resultado NOT IN ('aceptado','renuncia','sin_respuesta') OR p_registrado IS NULL OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_operacion,512) OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_llamamiento,512) OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(p_actor,512) THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='resultado invalido'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resultado:'||p_llamamiento,0)); SELECT * INTO e FROM vec_bolsa_llamamientos.llamamiento_operativo WHERE llamamiento_ref=p_llamamiento FOR UPDATE; IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503',MESSAGE='llamamiento inexistente'; END IF;
 IF e.resultado_operacion_ref IS NOT NULL THEN IF e.resultado_operacion_ref<>p_operacion OR e.actor_resultado_ref<>p_actor OR e.estado<>p_resultado THEN RAISE EXCEPTION USING ERRCODE='PBL23',MESSAGE='resultado distinto'; END IF; RETURN jsonb_build_object('reutilizado',true,'llamamiento_ref',e.llamamiento_ref,'participacion_ref',e.participacion_ref,'estado',e.estado,'recibo_ref',e.recibo_ref,'confirmado_en',e.resultado_registrado_en); END IF;
 v_situacion:=CASE p_resultado WHEN 'aceptado' THEN 'ocupado' WHEN 'renuncia' THEN 'renuncia_pendiente' ELSE 'disponible' END;
 UPDATE vec_bolsa_llamamientos.llamamiento_operativo SET estado=p_resultado,resultado_operacion_ref=p_operacion,actor_resultado_ref=p_actor,resultado_registrado_en=v_ahora WHERE llamamiento_ref=p_llamamiento;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion_llamamiento(participacion_ref,secuencia,estado,llamamiento_ref,operacion_ref,registrada_en) SELECT e.participacion_ref,coalesce(max(s.secuencia),0)+1,v_situacion,p_llamamiento,p_operacion,v_ahora FROM vec_bolsa_llamamientos.situacion_participacion_llamamiento s WHERE s.participacion_ref=e.participacion_ref;
 INSERT INTO vec_bolsa_llamamientos.auditoria_llamamiento_operativo VALUES('auditoria:bolsa:'||p_operacion,p_llamamiento,p_operacion,'resultado',p_actor,v_ahora); INSERT INTO vec_bolsa_llamamientos.outbox_llamamiento_operativo VALUES('evento:bolsa:'||p_operacion,p_llamamiento,p_operacion,'bolsa.llamamiento.resultado.v1',v_ahora);
 RETURN jsonb_build_object('reutilizado',false,'llamamiento_ref',e.llamamiento_ref,'participacion_ref',e.participacion_ref,'estado',p_resultado,'recibo_ref',e.recibo_ref,'confirmado_en',v_ahora);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_resultado_llamamiento_operativo_v1(text,text,text,text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.contactos_participacion_v1(text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.abrir_llamamiento_operativo_v1(text,text,text,text,text,timestamptz,timestamptz,text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_resultado_llamamiento_operativo_v1(text,text,text,text,timestamptz) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
