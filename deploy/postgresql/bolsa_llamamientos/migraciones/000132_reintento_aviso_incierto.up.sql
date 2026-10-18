\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000132',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::integer<180000
  OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_cabecera') IS NULL
  OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_destinatario') IS NULL
  OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_intento') IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.aviso_oferta_cabecera'::regclass
     AND c.relowner='vec_bolsa_llamamientos_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
  OR to_regclass('vec_bolsa_llamamientos.reintento_aviso_oferta_v1') IS NOT NULL
  OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_bolsa_llamamientos.aviso_oferta_cabecera'::regclass
      AND attname='revision' AND NOT attisdropped)
  OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
    'vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz)')),'UTF8')),'hex')
    IS DISTINCT FROM '9f0a3f09165887932148aac841af3d73796355b0a3735228a12d2be220ced3c7'
  OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'B132: PARO preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La columna nace a cero por defecto sin actualizar cabeceras históricas.
ALTER TABLE vec_bolsa_llamamientos.aviso_oferta_cabecera
 ADD COLUMN revision bigint NOT NULL DEFAULT 0 CHECK(revision BETWEEN 0 AND 9007199254740991);
CREATE TABLE vec_bolsa_llamamientos.reintento_aviso_oferta_v1 (
 oferta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 clave_idempotencia text NOT NULL,
 actor_ref text NOT NULL,
 revision_esperada bigint NOT NULL CHECK(revision_esperada BETWEEN 0 AND 9007199254740991),
 revision_resultante bigint NOT NULL CHECK(revision_resultante BETWEEN 1 AND 9007199254740991),
 huella_comando_sha256 text NOT NULL CHECK(huella_comando_sha256 ~ '^[0-9a-f]{64}$'),
 confirmacion boolean NOT NULL CHECK(confirmacion),
 recibo_ref text NOT NULL UNIQUE,
 registrado_en timestamptz(6) NOT NULL,
 rearmados integer NOT NULL CHECK(rearmados BETWEEN 1 AND 1000000),
 decision_ref text NOT NULL UNIQUE,
 PRIMARY KEY(oferta_ref,clave_idempotencia),
 CHECK(octet_length(clave_idempotencia) BETWEEN 8 AND 256),
 CHECK(revision_resultante>revision_esperada),
 CHECK(actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 CHECK(recibo_ref ~ '^recibo:aviso-oferta:[0-9a-f]{64}$')
);
CREATE INDEX reintento_aviso_oferta_ultimo_b132 ON vec_bolsa_llamamientos.reintento_aviso_oferta_v1
 (oferta_ref,revision_resultante DESC);
ALTER TABLE vec_bolsa_llamamientos.reintento_aviso_oferta_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.reintento_aviso_oferta_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY reintento_aviso_oferta_propietario ON vec_bolsa_llamamientos.reintento_aviso_oferta_v1
 TO vec_bolsa_llamamientos_propietario
 USING(current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.reintento_aviso_oferta_v1 FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.avanzar_revision_aviso_b132() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF NEW IS DISTINCT FROM OLD THEN
  UPDATE vec_bolsa_llamamientos.aviso_oferta_cabecera h
   SET revision=h.revision+1 WHERE h.oferta_ref=NEW.oferta_ref
    AND h.revision<9007199254740991;
  IF NOT FOUND THEN RAISE EXCEPTION 'B132: revisión agotada' USING ERRCODE='55000'; END IF;
 END IF;
 RETURN NULL;
END $f$;
CREATE TRIGGER avanzar_revision_aviso_b132 AFTER UPDATE ON vec_bolsa_llamamientos.aviso_oferta_destinatario
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.avanzar_revision_aviso_b132();
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.avanzar_revision_aviso_b132() FROM PUBLIC;

CREATE FUNCTION vec_bolsa_llamamientos.reintentar_avisos_sin_confirmacion_v1(
 p_oferta text,p_bolsa text,p_actor text,p_clave text,p_revision bigint,p_confirmar boolean,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea,p_unidad text,p_ambito text)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE v_huella text;v_contexto text;v_decision jsonb;v_consumo record;
 v_oferta record;v_cabecera record;v_previo record;v_fila record;
 v_refs text[]:='{}';v_ahora timestamptz:=clock_timestamp();v_recibo text;v_proyeccion jsonb;v_revision_resultante bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
   OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
   OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
   OR current_setting('transaction_isolation')<>'serializable'
   OR current_setting('transaction_read_only')<>'off'
   OR current_setting('statement_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '15 seconds'
   OR current_setting('idle_in_transaction_session_timeout')::interval NOT BETWEEN interval '1 millisecond' AND interval '20 seconds'
   OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
   OR p_bolsa IS NULL OR octet_length(p_bolsa) NOT BETWEEN 1 AND 256
   OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
   OR p_clave IS NULL OR octet_length(p_clave) NOT BETWEEN 8 AND 256 OR p_clave<>btrim(p_clave)
   OR p_revision IS NULL OR p_revision NOT BETWEEN 0 AND 9007199254740991
   OR p_confirmar IS DISTINCT FROM true
   OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$'
   OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$'
 THEN RAISE EXCEPTION 'B132: petición inválida' USING ERRCODE='22023'; END IF;
 v_huella:=encode(sha256(convert_to(
  octet_length('vec.bolsa.oferta.avisos.reintentar.v1')::text||':vec.bolsa.oferta.avisos.reintentar.v1'||
  octet_length(p_bolsa)::text||':'||p_bolsa||octet_length(p_oferta)::text||':'||p_oferta||
  octet_length(p_actor)::text||':'||p_actor||octet_length(p_revision::text)::text||':'||p_revision::text||
  octet_length(p_clave)::text||':'||p_clave||'4:true','UTF8')),'hex');
 v_contexto:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||v_huella||'"}}','UTF8')),'hex');
 BEGIN v_decision:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'B132: decisión ilegible' USING ERRCODE='42501'; END;
 IF v_decision->>'principal_id' IS DISTINCT FROM p_actor
   OR v_decision->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
   OR v_decision->>'modulo_id' IS DISTINCT FROM 'bolsa'
   OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
   OR v_decision->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
   OR v_decision->>'recurso_ref' IS DISTINCT FROM p_bolsa
   OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
 THEN RAISE EXCEPTION 'B132: decisión no ligada al reintento' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:reintento-aviso:'||p_oferta||':'||p_clave,0));
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.efecto_ref IS DISTINCT FROM p_bolsa
   OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto
   OR v_consumo.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'B132: consumo divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_oferta FROM vec_bolsa_llamamientos.oferta_publicada o
  WHERE o.oferta_ref=p_oferta AND o.bolsa_ref=p_bolsa;
 IF NOT FOUND THEN RAISE EXCEPTION 'B132: oferta ajena a la bolsa' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_previo FROM vec_bolsa_llamamientos.reintento_aviso_oferta_v1 j
  WHERE j.oferta_ref=p_oferta AND j.clave_idempotencia=p_clave;
 IF FOUND THEN
  IF v_previo.actor_ref IS DISTINCT FROM p_actor OR v_previo.revision_esperada IS DISTINCT FROM p_revision
    OR v_previo.huella_comando_sha256 IS DISTINCT FROM v_huella OR v_previo.confirmacion IS DISTINCT FROM true
  THEN RAISE EXCEPTION 'B132: clave divergente' USING ERRCODE='VBO01'; END IF;
  v_proyeccion:=vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp());
  oferta:=v_proyeccion||jsonb_build_object('reintento_aviso',jsonb_build_object(
   'recibo_ref',v_previo.recibo_ref,'registrado_en',to_char(v_previo.registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'rearmados',v_previo.rearmados));reutilizada:=true;RETURN NEXT;RETURN;
 END IF;
 -- El orden fila→cabecera coincide con reclamación y finalización automáticas.
 FOR v_fila IN SELECT d.* FROM vec_bolsa_llamamientos.aviso_oferta_destinatario d
   WHERE d.oferta_ref=p_oferta AND d.estado='enviado_sin_confirmacion'
   ORDER BY d.participacion_ref FOR UPDATE LOOP
  IF v_fila.ultimo_resultado IS NULL AND v_fila.ultimo_intento_en>v_ahora-interval '15 seconds' THEN
   RAISE EXCEPTION 'B132: hay un envío en curso' USING ERRCODE='VBO11';
  END IF;
  v_refs:=array_append(v_refs,v_fila.participacion_ref);
 END LOOP;
 SELECT * INTO v_cabecera FROM vec_bolsa_llamamientos.aviso_oferta_cabecera h
   WHERE h.oferta_ref=p_oferta FOR UPDATE;
 IF NOT FOUND OR v_cabecera.huella_token IS NULL OR v_cabecera.version_token IS NULL THEN
  RAISE EXCEPTION 'B132: aviso sin reserva' USING ERRCODE='55000'; END IF;
 IF v_cabecera.revision IS DISTINCT FROM p_revision THEN
  RAISE EXCEPTION 'B132: revisión de avisos cambiada' USING ERRCODE='VBO10'; END IF;
 IF cardinality(v_refs)=0 THEN RAISE EXCEPTION 'B132: no hay avisos inciertos' USING ERRCODE='VBO10'; END IF;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_intento i
  SET resultado='indeterminado',finalizado_en=v_ahora
  WHERE i.oferta_ref=p_oferta AND i.participacion_ref=ANY(v_refs)
    AND i.resultado IS NULL;
 UPDATE vec_bolsa_llamamientos.aviso_oferta_destinatario d
  SET estado='pendiente',ultimo_resultado='indeterminado',actualizado_en=v_ahora
  WHERE d.oferta_ref=p_oferta AND d.participacion_ref=ANY(v_refs)
    AND d.estado='enviado_sin_confirmacion';
 IF NOT FOUND THEN RAISE EXCEPTION 'B132: no se rearmó el aviso' USING ERRCODE='55000'; END IF;
 SELECT h.revision INTO STRICT v_revision_resultante
  FROM vec_bolsa_llamamientos.aviso_oferta_cabecera h WHERE h.oferta_ref=p_oferta;
 v_recibo:='recibo:aviso-oferta:'||encode(sha256(convert_to(p_oferta||chr(31)||p_clave,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.reintento_aviso_oferta_v1(
  oferta_ref,clave_idempotencia,actor_ref,revision_esperada,revision_resultante,huella_comando_sha256,
  confirmacion,recibo_ref,registrado_en,rearmados,decision_ref)
 VALUES(p_oferta,p_clave,p_actor,p_revision,v_revision_resultante,v_huella,true,v_recibo,v_ahora,cardinality(v_refs),v_consumo.decision_ref);
 v_proyeccion:=vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,clock_timestamp());
 oferta:=v_proyeccion||jsonb_build_object('reintento_aviso',jsonb_build_object(
  'recibo_ref',v_recibo,'registrado_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'rearmados',cardinality(v_refs)));
 reutilizada:=false;RETURN NEXT;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.reintentar_avisos_sin_confirmacion_v1(
 text,text,text,text,bigint,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.reintentar_avisos_sin_confirmacion_v1(
 text,text,text,text,bigint,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)
 TO vec_bolsa_llamamientos_ejecutor;

-- Proyección literal derivada del cuerpo instalado, preimagen comprobada arriba.
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
   'fallidos',count(*) FILTER (WHERE d.estado='no_enviado'),'revision',h.revision),
   'aviso_version_token',h.version_token) INTO v_extra
  FROM vec_bolsa_llamamientos.aviso_oferta_cabecera h
  LEFT JOIN vec_bolsa_llamamientos.aviso_oferta_destinatario d ON d.oferta_ref=h.oferta_ref
  WHERE h.oferta_ref=p_oferta GROUP BY h.oferta_ref,h.version_token;
 v_extra:=coalesce(v_extra,'{}'::jsonb) || coalesce((
   SELECT jsonb_build_object('reintento_aviso',jsonb_build_object(
     'recibo_ref',j.recibo_ref,'registrado_en',to_char(j.registrado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
     'rearmados',j.rearmados))
   FROM vec_bolsa_llamamientos.reintento_aviso_oferta_v1 j
   WHERE j.oferta_ref=p_oferta ORDER BY j.revision_resultante DESC LIMIT 1),'{}'::jsonb);
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

DO $post$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.reintento_aviso_oferta_v1'::regclass
   AND c.relowner='vec_bolsa_llamamientos_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
   OR NOT EXISTS(SELECT 1 FROM pg_attribute a JOIN pg_attrdef d
      ON d.adrelid=a.attrelid AND d.adnum=a.attnum
      WHERE a.attrelid='vec_bolsa_llamamientos.aviso_oferta_cabecera'::regclass
        AND a.attname='revision' AND NOT a.attisdropped AND a.attnotnull
        AND a.atttypid='bigint'::regtype AND pg_get_expr(d.adbin,d.adrelid)='0')
   OR NOT EXISTS(SELECT 1 FROM pg_trigger t
      WHERE t.tgrelid='vec_bolsa_llamamientos.aviso_oferta_destinatario'::regclass
        AND t.tgname='avanzar_revision_aviso_b132' AND t.tgenabled='O' AND NOT t.tgisinternal)
   OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_bolsa_llamamientos.reintentar_avisos_sin_confirmacion_v1(text,text,text,text,bigint,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)'::regprocedure
    AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef
    AND EXISTS(SELECT 1 FROM unnest(p.proconfig) x WHERE x IN('search_path=pg_catalog, pg_temp','search_path=pg_catalog,pg_temp'))
    AND NOT EXISTS(SELECT 1 FROM aclexplode(p.proacl) acl WHERE acl.grantee=0 OR acl.grantee NOT IN
      ('vec_bolsa_llamamientos_propietario'::regrole,'vec_bolsa_llamamientos_ejecutor'::regrole)))
   OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.reintento_aviso_oferta_v1','SELECT')
 THEN RAISE EXCEPTION 'B132: PARO postimagen de roles incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
