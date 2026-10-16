\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000129',0));

DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer < 180000
    OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_cabecera') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_destinatario') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_intento') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.situaciones_en_v1(timestamptz)') IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
       'vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz)')),'UTF8')),'hex')
       IS DISTINCT FROM 'b71aa0f3bef157e08f0ab0c93a3c66308d59682d3de4c92b27fd1f4b8e690e21'
    OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
       'vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)')),'UTF8')),'hex')
       IS DISTINCT FROM '3811f4fbb392f0b8c37f2583ae163e0aefafdf71fa34e2f7fc62454ef236d995'
 THEN RAISE EXCEPTION 'B129: PARO preimagen de ofertas incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.aviso_oferta_cabecera (
 oferta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 publicada_en timestamptz(6) NOT NULL,
 huella_token bytea UNIQUE CHECK (huella_token IS NULL OR octet_length(huella_token)=32),
 version_token text CHECK (version_token IS NULL OR version_token ~ '^[A-Za-z0-9._:-]{1,128}$'),
 CHECK ((huella_token IS NULL)=(version_token IS NULL))
);
CREATE TABLE vec_bolsa_llamamientos.aviso_oferta_destinatario (
 oferta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.aviso_oferta_cabecera(oferta_ref),
 participacion_ref text NOT NULL,
 message_id text NOT NULL UNIQUE,
 fuente text NOT NULL DEFAULT 'alta_bolsa' CHECK (fuente='alta_bolsa'),
 estado text NOT NULL DEFAULT 'pendiente'
   CHECK (estado IN ('pendiente','enviado','enviado_sin_confirmacion','no_enviado')),
 intentos integer NOT NULL DEFAULT 0 CHECK (intentos>=0),
 ultimo_resultado text CHECK (ultimo_resultado IS NULL OR ultimo_resultado IN
   ('aceptado','transitorio','permanente','indeterminado','limite_tecnico')),
 ultimo_intento_en timestamptz(6),
 actualizado_en timestamptz(6) NOT NULL,
 PRIMARY KEY (oferta_ref,participacion_ref),
 CHECK ((intentos=0 AND ultimo_intento_en IS NULL AND ultimo_resultado IS NULL)
     OR (intentos>0 AND ultimo_intento_en IS NOT NULL))
);
CREATE INDEX aviso_oferta_pendientes_b129 ON vec_bolsa_llamamientos.aviso_oferta_destinatario(oferta_ref,intentos,participacion_ref)
 WHERE estado='pendiente';
CREATE TABLE vec_bolsa_llamamientos.aviso_oferta_intento (
 oferta_ref text NOT NULL,
 participacion_ref text NOT NULL,
 intento integer NOT NULL CHECK (intento>0),
 reclamado_en timestamptz(6) NOT NULL,
 resultado text CHECK (resultado IS NULL OR resultado IN
   ('aceptado','transitorio','permanente','indeterminado')),
 finalizado_en timestamptz(6),
 PRIMARY KEY (oferta_ref,participacion_ref,intento),
 FOREIGN KEY (oferta_ref,participacion_ref)
   REFERENCES vec_bolsa_llamamientos.aviso_oferta_destinatario(oferta_ref,participacion_ref),
 CHECK ((resultado IS NULL)=(finalizado_en IS NULL))
);
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_cabecera ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_cabecera FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_destinatario ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_destinatario FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_intento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_intento FORCE ROW LEVEL SECURITY;
CREATE POLICY aviso_oferta_cabecera_propietario ON vec_bolsa_llamamientos.aviso_oferta_cabecera
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY aviso_oferta_destinatario_propietario ON vec_bolsa_llamamientos.aviso_oferta_destinatario
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY aviso_oferta_intento_propietario ON vec_bolsa_llamamientos.aviso_oferta_intento
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.aviso_oferta_cabecera,
 vec_bolsa_llamamientos.aviso_oferta_destinatario,
 vec_bolsa_llamamientos.aviso_oferta_intento FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.crear_aviso_oferta_b129() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET row_security=on AS $f$
BEGIN
 INSERT INTO vec_bolsa_llamamientos.aviso_oferta_cabecera(oferta_ref,publicada_en)
 VALUES(NEW.oferta_ref,NEW.publicada_en);
 INSERT INTO vec_bolsa_llamamientos.aviso_oferta_destinatario
   (oferta_ref,participacion_ref,message_id,actualizado_en)
 SELECT NEW.oferta_ref,s.participacion_ref,
   '<'||encode(sha256(convert_to(NEW.oferta_ref||chr(31)||s.participacion_ref,'UTF8')),'hex')||'@vec.dipgra.local>',
   NEW.publicada_en
 FROM vec_bolsa_llamamientos.situaciones_en_v1(NEW.publicada_en) s
 WHERE s.bolsa_ref=NEW.bolsa_ref AND s.situacion='disponible'
   AND EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion c
     JOIN vec_bolsa_llamamientos.bolsa_constituida b
       ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
     JOIN vec_bolsa_llamamientos.constitucion_entrada e
       ON e.instantanea_ref=c.instantanea_ref
      AND e.version_instantanea=c.version_instantanea
      AND e.participacion_ref=s.participacion_ref
     WHERE c.bolsa_ref=NEW.bolsa_ref AND b.estado='vigente'
       AND b.vigente_desde<=NEW.publicada_en
       AND (b.vigente_hasta IS NULL OR NEW.publicada_en<b.vigente_hasta));
 RETURN NEW;
END $f$;
CREATE TRIGGER crear_aviso_oferta_b129 AFTER INSERT ON vec_bolsa_llamamientos.oferta_publicada
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.crear_aviso_oferta_b129();

CREATE FUNCTION vec_bolsa_llamamientos.exigir_token_aviso_oferta_b129() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET row_security=on AS $f$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_cabecera h
    WHERE h.oferta_ref=NEW.oferta_ref AND h.huella_token IS NOT NULL
      AND h.version_token IS NOT NULL) THEN
  RAISE EXCEPTION 'B129: publicación sin reserva de despacho' USING ERRCODE='55000';
 END IF;
 RETURN NULL;
END $f$;
CREATE CONSTRAINT TRIGGER exigir_token_aviso_oferta_b129
 AFTER INSERT ON vec_bolsa_llamamientos.aviso_oferta_cabecera
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.exigir_token_aviso_oferta_b129();

CREATE FUNCTION vec_bolsa_llamamientos.vincular_despacho_aviso_v1(
 p_oferta text,p_actor text,p_clave text,p_token_sha bytea,p_version text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE o record;h record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('statement_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '15 seconds'
    OR current_setting('idle_in_transaction_session_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '20 seconds'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR octet_length(p_token_sha) IS DISTINCT FROM 32
    OR p_version IS NULL OR p_version !~ '^[A-Za-z0-9._:-]{1,128}$'
 THEN RAISE EXCEPTION 'B129: reserva de aviso inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada
  WHERE oferta_ref=p_oferta AND actor_ref=p_actor AND clave_idempotencia=p_clave
    AND xmin::text=pg_current_xact_id_if_assigned()::text;
 IF NOT FOUND THEN RAISE EXCEPTION 'B129: reserva fuera de publicación' USING ERRCODE='42501'; END IF;
 SELECT * INTO h FROM vec_bolsa_llamamientos.aviso_oferta_cabecera
  WHERE oferta_ref=p_oferta AND xmin::text=pg_current_xact_id_if_assigned()::text FOR UPDATE;
 IF NOT FOUND OR h.huella_token IS NOT NULL OR h.version_token IS NOT NULL THEN
  RAISE EXCEPTION 'B129: reserva repetida o ajena' USING ERRCODE='42501';
 END IF;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_cabecera
   SET huella_token=p_token_sha,version_token=p_version WHERE oferta_ref=p_oferta;
 RETURN vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp());
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.reclamar_aviso_oferta_v1(
 p_oferta text,p_token bytea,p_limite integer)
RETURNS TABLE(participacion_ref text,message_id text,intento integer)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE v record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('statement_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '15 seconds'
    OR current_setting('idle_in_transaction_session_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '20 seconds'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$' OR octet_length(p_token) IS DISTINCT FROM 32
    OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 10
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_cabecera h
       WHERE h.oferta_ref=p_oferta AND h.huella_token=sha256(p_token))
 THEN RAISE EXCEPTION 'B129: despacho no autorizado' USING ERRCODE='42501'; END IF;
 -- El contador entero no puede desbordarse dejando una fila pendiente eterna.
 UPDATE vec_bolsa_llamamientos.aviso_oferta_destinatario
  SET estado='no_enviado',ultimo_resultado='limite_tecnico',actualizado_en=clock_timestamp()
  WHERE oferta_ref=p_oferta AND estado='pendiente' AND intentos=2147483647;
 FOR v IN WITH elegidos AS (
   SELECT d.oferta_ref,d.participacion_ref
   FROM vec_bolsa_llamamientos.aviso_oferta_destinatario d
   WHERE d.oferta_ref=p_oferta AND d.estado='pendiente' AND d.intentos<2147483647
   ORDER BY d.intentos,d.participacion_ref LIMIT p_limite FOR UPDATE SKIP LOCKED
  )
  UPDATE vec_bolsa_llamamientos.aviso_oferta_destinatario d
   SET estado='enviado_sin_confirmacion',intentos=d.intentos+1,
       ultimo_intento_en=clock_timestamp(),ultimo_resultado=NULL,actualizado_en=clock_timestamp()
  FROM elegidos e WHERE d.oferta_ref=e.oferta_ref AND d.participacion_ref=e.participacion_ref
  RETURNING d.participacion_ref,d.message_id,d.intentos,d.ultimo_intento_en
 LOOP
  INSERT INTO vec_bolsa_llamamientos.aviso_oferta_intento
   (oferta_ref,participacion_ref,intento,reclamado_en)
  VALUES(p_oferta,v.participacion_ref,v.intentos,v.ultimo_intento_en);
  participacion_ref:=v.participacion_ref;message_id:=v.message_id;intento:=v.intentos;
  RETURN NEXT;
 END LOOP;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.finalizar_aviso_oferta_v1(
 p_oferta text,p_participacion text,p_token bytea,p_intento integer,p_resultado text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE d record;v_estado text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('statement_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '15 seconds'
    OR current_setting('idle_in_transaction_session_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '20 seconds'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$' OR octet_length(p_token) IS DISTINCT FROM 32
    OR p_intento IS NULL OR p_intento<1
    OR p_resultado IS NULL OR p_resultado NOT IN ('aceptado','transitorio','permanente','indeterminado')
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_cabecera h
       WHERE h.oferta_ref=p_oferta AND h.huella_token=sha256(p_token))
 THEN RAISE EXCEPTION 'B129: resultado no autorizado' USING ERRCODE='42501'; END IF;
 SELECT * INTO d FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
  WHERE oferta_ref=p_oferta AND participacion_ref=p_participacion FOR UPDATE;
 IF NOT FOUND OR d.intentos<>p_intento THEN
  RAISE EXCEPTION 'B129: intento no vigente' USING ERRCODE='55000'; END IF;
 v_estado:=CASE p_resultado WHEN 'aceptado' THEN 'enviado'
   WHEN 'transitorio' THEN 'pendiente' WHEN 'permanente' THEN 'no_enviado'
   ELSE 'enviado_sin_confirmacion' END;
 IF d.ultimo_resultado=p_resultado AND d.estado=v_estado THEN
  RETURN vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp());
 END IF;
 IF d.estado<>'enviado_sin_confirmacion' OR d.ultimo_resultado IS NOT NULL THEN
  RAISE EXCEPTION 'B129: resultado contradictorio' USING ERRCODE='55000'; END IF;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_destinatario
  SET estado=v_estado,ultimo_resultado=p_resultado,actualizado_en=clock_timestamp()
  WHERE oferta_ref=p_oferta AND participacion_ref=p_participacion AND intentos=p_intento;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_intento
   SET resultado=p_resultado,finalizado_en=clock_timestamp()
   WHERE oferta_ref=p_oferta AND participacion_ref=p_participacion
     AND intento=p_intento AND resultado IS NULL;
 IF NOT FOUND THEN RAISE EXCEPTION 'B129: historia de intento ausente' USING ERRCODE='55000'; END IF;
 RETURN vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp());
END $f$;

-- Definición literal obtenida de pg_get_functiondef en la copia de la principal.
-- Preimagen SHA-256: b71aa0f3bef157e08f0ab0c93a3c66308d59682d3de4c92b27fd1f4b8e690e21.
-- Solo añade el resumen del aviso en las dos salidas y fija pg_temp en search_path.
-- CREATE OR REPLACE conserva la firma, el OID, el propietario y las ACL instaladas.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta text, p_corte timestamp with time zone)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
AS $function$
DECLARE
 base jsonb; v_extra jsonb; o record; v_n integer; v_config jsonb; v_gestion boolean;
 v_estados text[]:='{}'; v_ultimos jsonb[]:='{}'; ultimo record; v_estado text;
 v_candidatos text[]:='{}'; v_ordenes bigint[]:='{}'; v_siguiente integer:=1;
 v_pendiente boolean; v_propuesta jsonb; v_plazas jsonb:='[]'::jsonb; v_historial jsonb;
 v_cubiertas integer:=0; v_directas integer:=0; v_actos integer:=0; v_primera boolean:=true;
 v_puede boolean; i integer; v_confirmacion text;
BEGIN
 base:=vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta,p_corte);
 IF base IS NULL THEN RETURN NULL; END IF;
 SELECT * INTO STRICT o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 SELECT jsonb_build_object('aviso',jsonb_build_object(
   'total',count(d.participacion_ref),
   'pendientes',count(*) FILTER (WHERE d.estado='pendiente'),
   'enviados',count(*) FILTER (WHERE d.estado='enviado'),
   'sin_confirmacion',count(*) FILTER (WHERE d.estado='enviado_sin_confirmacion'),
   'fallidos',count(*) FILTER (WHERE d.estado='no_enviado')),
   'aviso_version_token',h.version_token) INTO v_extra
  FROM vec_bolsa_llamamientos.aviso_oferta_cabecera h
  LEFT JOIN vec_bolsa_llamamientos.aviso_oferta_destinatario d ON d.oferta_ref=h.oferta_ref
  WHERE h.oferta_ref=p_oferta GROUP BY h.oferta_ref,h.version_token;
 v_extra:=coalesce(v_extra,'{}'::jsonb);
 SELECT coalesce((SELECT numero_plazas FROM vec_bolsa_llamamientos.plazas_oferta WHERE oferta_ref=p_oferta),1) INTO v_n;
 SELECT pv.politica->'plazas', pv.politica#>>'{adjudicacion,confirmacion}' INTO v_config,v_confirmacion FROM vec_bolsa_llamamientos.politica_ofertas_version pv
  WHERE pv.bolsa_ref=o.bolsa_ref AND pv.version::text=o.plazo->>'politica_version';
 v_gestion:=jsonb_typeof(v_config)='object';
 IF NOT v_gestion THEN v_config:=NULL; END IF;
 -- Resolución única anterior (B28): se muestra como su única plaza, sin actos.
 IF jsonb_typeof(base->'resolucion')='object' THEN
  RETURN base || jsonb_build_object('propuesta',NULL,'numero_plazas',1,'politica_plazas',v_config,'confirmacion_adjudicacion',v_confirmacion,
   'plazas',jsonb_build_array(jsonb_build_object('numero_de_plaza',1,
    'estado',CASE WHEN base#>>'{resolucion,tipo}'='adjudicada' THEN 'cubierta' ELSE 'llamamiento_directo' END,
    'secuencia',0,'participacion_ref',base#>'{resolucion,participacion_ref}','orden_vigente',base#>'{resolucion,orden_vigente}',
    'responder_antes_de',NULL,'puede_sin_respuesta',false,'propuesta',NULL,'historial','[]'::jsonb))) || v_extra;
 END IF;
 FOR i IN 1..v_n LOOP
  SELECT a.* INTO ultimo FROM vec_bolsa_llamamientos.acto_plaza_oferta a
   WHERE a.oferta_ref=p_oferta AND a.numero_de_plaza=i AND a.registrado_en<=p_corte
   ORDER BY a.secuencia DESC LIMIT 1;
  IF ultimo.tipo IS NULL THEN v_estado:='vacante';
  ELSIF ultimo.tipo='adjudicada' THEN v_estado:=CASE WHEN v_gestion AND v_confirmacion IS DISTINCT FROM 'aceptacion_previa' THEN 'pendiente_respuesta' ELSE 'cubierta' END;
  ELSIF ultimo.tipo='aceptada' THEN v_estado:='cubierta';
  ELSIF ultimo.tipo='llamamiento_directo' THEN v_estado:='llamamiento_directo';
  ELSE v_estado:='vacante';
  END IF;
  v_estados:=v_estados||v_estado;
  v_ultimos:=v_ultimos||CASE WHEN ultimo.tipo IS NULL THEN 'null'::jsonb ELSE jsonb_build_object(
   'secuencia',ultimo.secuencia,'tipo',ultimo.tipo,'participacion_ref',ultimo.participacion_ref,
   'orden_vigente',ultimo.orden_vigente,
   'responder_antes_de',to_char(ultimo.responder_antes_de AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END;
  IF v_estado='cubierta' THEN v_cubiertas:=v_cubiertas+1; END IF;
  IF v_estado='llamamiento_directo' THEN v_directas:=v_directas+1; END IF;
  IF ultimo.tipo IS NOT NULL THEN v_actos:=v_actos+1; END IF;
 END LOOP;
 v_pendiente:='pendiente_respuesta'=ANY(v_estados);
 IF p_corte>=o.vence_antes_de THEN
  SELECT coalesce(array_agg(x.participacion_ref ORDER BY x.orden_vigente,x.participacion_ref),'{}'),
         coalesce(array_agg(x.orden_vigente ORDER BY x.orden_vigente,x.participacion_ref),'{}')
    INTO v_candidatos,v_ordenes
    FROM (SELECT d.participacion_ref,ov.orden_vigente
            FROM vec_bolsa_llamamientos.disposicion_oferta d
            JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(o.bolsa_ref,p_corte) ov ON ov.participacion_ref=d.participacion_ref
           WHERE d.oferta_ref=p_oferta AND d.manifestada_en<o.vence_antes_de AND ov.orden_vigente IS NOT NULL
             AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.acto_plaza_oferta a
                              WHERE a.oferta_ref=p_oferta AND a.participacion_ref=d.participacion_ref AND a.registrado_en<=p_corte)) x;
 END IF;
 FOR i IN 1..v_n LOOP
  v_propuesta:=NULL;
  IF p_corte>=o.vence_antes_de AND v_estados[i]='vacante' THEN
   -- Sin apartado de plazas o llamada sucesiva: una propuesta cada vez, para
   -- la primera plaza libre y sin nadie pendiente de responder.
   IF (v_gestion AND v_config->>'llamada'='simultanea') OR (v_primera AND NOT v_pendiente) THEN
    IF v_gestion AND v_ultimos[i]->>'tipo' IN ('renuncia','sin_respuesta') AND v_config->>'tras_renuncia'='llamamiento_directo' THEN
     v_propuesta:=jsonb_build_object('tipo','llamamiento_directo');
    ELSIF v_siguiente<=coalesce(array_length(v_candidatos,1),0) THEN
     v_propuesta:=jsonb_build_object('tipo','adjudicar','participacion_ref',v_candidatos[v_siguiente],'orden_vigente',v_ordenes[v_siguiente]);
     v_siguiente:=v_siguiente+1;
    ELSE
     v_propuesta:=jsonb_build_object('tipo','llamamiento_directo');
    END IF;
   END IF;
   v_primera:=false;
  END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('secuencia',a.secuencia,'tipo',a.tipo,'orden_vigente',a.orden_vigente,
          'registrado_en',to_char(a.registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'recibo_ref',a.recibo_ref)
          ORDER BY a.secuencia),'[]'::jsonb)
    INTO v_historial FROM vec_bolsa_llamamientos.acto_plaza_oferta a
   WHERE a.oferta_ref=p_oferta AND a.numero_de_plaza=i AND a.registrado_en<=p_corte;
  v_puede:=v_estados[i]='pendiente_respuesta'
   AND (v_ultimos[i]->>'responder_antes_de' IS NULL OR p_corte>=(v_ultimos[i]->>'responder_antes_de')::timestamptz);
  v_plazas:=v_plazas||jsonb_build_array(jsonb_build_object(
   'numero_de_plaza',i,'estado',v_estados[i],
   'secuencia',coalesce((v_ultimos[i]->>'secuencia')::int,0),
   'participacion_ref',CASE WHEN v_estados[i] IN ('pendiente_respuesta','cubierta') THEN v_ultimos[i]->'participacion_ref' ELSE NULL END,
   'orden_vigente',CASE WHEN v_estados[i] IN ('pendiente_respuesta','cubierta') THEN v_ultimos[i]->'orden_vigente' ELSE NULL END,
   'responder_antes_de',CASE WHEN v_estados[i]='pendiente_respuesta' THEN v_ultimos[i]->'responder_antes_de' ELSE NULL END,
   'puede_sin_respuesta',v_puede,'propuesta',v_propuesta,'historial',v_historial));
 END LOOP;
 RETURN base || jsonb_build_object(
  'estado',CASE WHEN p_corte<o.vence_antes_de THEN 'abierta'
                WHEN v_cubiertas=v_n THEN 'adjudicada'
                WHEN v_directas=v_n THEN 'llamamiento_directo'
                WHEN v_cubiertas+v_directas=v_n THEN 'cerrada'
                WHEN v_actos>0 THEN 'en_curso'
                ELSE 'pendiente_resolucion' END,
  'propuesta',NULL,'numero_plazas',v_n,'politica_plazas',v_config,'confirmacion_adjudicacion',v_confirmacion,'plazas',v_plazas) || v_extra;
END $function$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.crear_aviso_oferta_b129(),
 vec_bolsa_llamamientos.exigir_token_aviso_oferta_b129(),
 vec_bolsa_llamamientos.vincular_despacho_aviso_v1(text,text,text,bytea,text),
 vec_bolsa_llamamientos.reclamar_aviso_oferta_v1(text,bytea,integer),
 vec_bolsa_llamamientos.finalizar_aviso_oferta_v1(text,text,bytea,integer,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
 vec_bolsa_llamamientos.vincular_despacho_aviso_v1(text,text,text,bytea,text),
 vec_bolsa_llamamientos.reclamar_aviso_oferta_v1(text,bytea,integer),
 vec_bolsa_llamamientos.finalizar_aviso_oferta_v1(text,text,bytea,integer,text)
 TO vec_bolsa_llamamientos_ejecutor;

DO $post$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname IN
   ('crear_aviso_oferta_b129','exigir_token_aviso_oferta_b129','vincular_despacho_aviso_v1',
    'reclamar_aviso_oferta_v1','finalizar_aviso_oferta_v1')
   AND (p.proowner<>'vec_bolsa_llamamientos_propietario'::regrole
     OR NOT p.prosecdef OR p.proconfig IS NULL
     OR NOT EXISTS(SELECT 1 FROM unnest(p.proconfig) x WHERE x IN
       ('search_path=pg_catalog, pg_temp','search_path=pg_catalog,pg_temp'))
     OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
       WHERE acl.grantee=0 OR acl.grantee NOT IN
         ('vec_bolsa_llamamientos_propietario'::regrole,
          'vec_bolsa_llamamientos_ejecutor'::regrole))))
    OR EXISTS (SELECT 1 FROM pg_class c
     WHERE c.oid IN ('vec_bolsa_llamamientos.aviso_oferta_cabecera'::regclass,
       'vec_bolsa_llamamientos.aviso_oferta_destinatario'::regclass,
       'vec_bolsa_llamamientos.aviso_oferta_intento'::regclass)
       AND (c.relowner<>'vec_bolsa_llamamientos_propietario'::regrole
         OR NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
 THEN RAISE EXCEPTION 'B129: PARO funciones sin cierre de privilegios' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
