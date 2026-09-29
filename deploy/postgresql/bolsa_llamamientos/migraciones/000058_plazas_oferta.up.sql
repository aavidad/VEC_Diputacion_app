\set ON_ERROR_STOP on
-- B58: ofertas con varias plazas y seguimiento por plaza (Petición RRHH
-- 3.06 y 3.07; duda 75). Una oferta declara al publicarse cuántas plazas
-- cubre; el número entra en el material que autoriza la publicación. Tras el
-- plazo de disposición, cada plaza se resuelve con actos de solo adición:
-- adjudicada, aceptada, renuncia, sin_respuesta o llamamiento_directo.
--
-- Cómo se llama (a la vez o de una en una), el plazo de respuesta y qué se
-- hace tras una renuncia salen del apartado «plazas» de la política de
-- ofertas de la bolsa (B47, siempre de ejemplo y versionada). Una oferta
-- publicada con una versión sin ese apartado conserva el comportamiento
-- anterior: una sola plaza y la adjudicación confirmada la cubre.
--
-- Cada acto consume una autorización nueva de emisión de llamamiento (la
-- misma que B28 usa para resolver) y escribe acto, auditoría y outbox en la
-- misma transacción. La resolución única de B28 queda cerrada al ejecutor;
-- sus filas históricas se siguen proyectando.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000058',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.oferta_publicada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.disposicion_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.resolucion_oferta') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.proyectar_oferta_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.exigir_consumo_candidato_v1(text[],text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(text)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_politica_oferta_b47()') IS NULL
    OR position('horas_naturales' in pg_get_functiondef('vec_bolsa_llamamientos.verificar_politica_oferta_b47()'::regprocedure))=0
    OR position('plazas' in pg_get_functiondef('vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))>0
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.plazas_oferta') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.acto_plaza_oferta') IS NOT NULL
 THEN RAISE EXCEPTION 'B58: preimagen incompatible (B28/B29/B47/B54 requeridas)' USING ERRCODE='55000'; END IF;
END $pre$;

-- Número de plazas fijado al publicar, con la versión de política que gobierna
-- el seguimiento. Una oferta anterior sin fila tiene una sola plaza.
CREATE TABLE vec_bolsa_llamamientos.plazas_oferta(
 oferta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 bolsa_ref text NOT NULL,
 numero_plazas integer NOT NULL CHECK (numero_plazas BETWEEN 1 AND 100),
 politica_version bigint NOT NULL CHECK (politica_version>=1),
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY (bolsa_ref,politica_version) REFERENCES vec_bolsa_llamamientos.politica_ofertas_version(bolsa_ref,version)
);

-- Historia de cada plaza: la secuencia es la versión de la plaza y la
-- confirma el comando (control optimista). El último acto fija su estado.
CREATE TABLE vec_bolsa_llamamientos.acto_plaza_oferta(
 oferta_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.oferta_publicada(oferta_ref),
 numero_de_plaza integer NOT NULL CHECK (numero_de_plaza BETWEEN 1 AND 100),
 secuencia integer NOT NULL CHECK (secuencia BETWEEN 1 AND 10000),
 tipo text NOT NULL CHECK (tipo IN ('adjudicada','aceptada','renuncia','sin_respuesta','llamamiento_directo')),
 participacion_ref text CHECK (participacion_ref IS NULL OR (octet_length(participacion_ref) BETWEEN 1 AND 256 AND participacion_ref=btrim(participacion_ref))),
 orden_vigente bigint,
 responder_antes_de timestamptz(6),
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:plaza-oferta:[0-9a-f]{64}$'),
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 8 AND 256 AND clave_idempotencia=btrim(clave_idempotencia)),
 huella_comando_sha256 text NOT NULL CHECK (huella_comando_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL,
 registrado_en timestamptz(6) NOT NULL,
 PRIMARY KEY (oferta_ref,numero_de_plaza,secuencia),
 UNIQUE (oferta_ref,clave_idempotencia),
 CHECK ((tipo='llamamiento_directo' AND participacion_ref IS NULL AND orden_vigente IS NULL AND responder_antes_de IS NULL)
     OR (tipo='adjudicada' AND participacion_ref IS NOT NULL AND orden_vigente IS NOT NULL)
     OR (tipo IN ('aceptada','renuncia','sin_respuesta') AND participacion_ref IS NOT NULL AND orden_vigente IS NOT NULL AND responder_antes_de IS NULL))
);
CREATE INDEX acto_plaza_oferta_participacion_idx
 ON vec_bolsa_llamamientos.acto_plaza_oferta(oferta_ref,participacion_ref) WHERE participacion_ref IS NOT NULL;

-- Evento durable de cada acto, escrito en la misma transacción. Solo lleva
-- referencias opacas; entregada_en lo marca el relé que lo publique.
CREATE TABLE vec_bolsa_llamamientos.acto_plaza_oferta_outbox(
 recibo_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.acto_plaza_oferta(recibo_ref),
 oferta_ref text NOT NULL,
 bolsa_ref text NOT NULL,
 numero_de_plaza integer NOT NULL,
 secuencia integer NOT NULL,
 tipo text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 entregada_en timestamptz(6),
 FOREIGN KEY (oferta_ref,numero_de_plaza,secuencia) REFERENCES vec_bolsa_llamamientos.acto_plaza_oferta(oferta_ref,numero_de_plaza,secuencia)
);

DO $acl$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['plazas_oferta','acto_plaza_oferta','acto_plaza_oferta_outbox'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY', t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY', t);
  EXECUTE format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user = ''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user = ''vec_bolsa_llamamientos_propietario'')', t || '_solo_propietario', t);
  EXECUTE format('REVOKE ALL ON vec_bolsa_llamamientos.%I FROM PUBLIC', t);
 END LOOP;
 FOREACH t IN ARRAY ARRAY['plazas_oferta','acto_plaza_oferta'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()', t || '_inmutable', t);
 END LOOP;
END $acl$;

-- Proyección por plazas en un instante. Parte de la proyección B28 (datos,
-- plazo, disposiciones y resolución histórica) y añade el número de plazas,
-- la política de plazas aplicada y, por plaza, estado, persona que la ocupa,
-- plazo de respuesta, propuesta de VEC e historia. La propuesta se calcula
-- con el orden vigente B6 en ese mismo instante entre quienes manifestaron
-- disposición en plazo y aún no tienen ningún acto en esta oferta.
CREATE FUNCTION vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta text, p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE
 base jsonb; o record; v_n integer; v_config jsonb; v_gestion boolean;
 v_estados text[]:='{}'; v_ultimos jsonb[]:='{}'; ultimo record; v_estado text;
 v_candidatos text[]:='{}'; v_ordenes bigint[]:='{}'; v_siguiente integer:=1;
 v_pendiente boolean; v_propuesta jsonb; v_plazas jsonb:='[]'::jsonb; v_historial jsonb;
 v_cubiertas integer:=0; v_directas integer:=0; v_actos integer:=0; v_primera boolean:=true;
 v_puede boolean; i integer;
BEGIN
 base:=vec_bolsa_llamamientos.proyectar_oferta_v1(p_oferta,p_corte);
 IF base IS NULL THEN RETURN NULL; END IF;
 SELECT * INTO STRICT o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 SELECT coalesce((SELECT numero_plazas FROM vec_bolsa_llamamientos.plazas_oferta WHERE oferta_ref=p_oferta),1) INTO v_n;
 SELECT pv.politica->'plazas' INTO v_config FROM vec_bolsa_llamamientos.politica_ofertas_version pv
  WHERE pv.bolsa_ref=o.bolsa_ref AND pv.version::text=o.plazo->>'politica_version';
 v_gestion:=jsonb_typeof(v_config)='object';
 IF NOT v_gestion THEN v_config:=NULL; END IF;
 -- Resolución única anterior (B28): se muestra como su única plaza, sin actos.
 IF jsonb_typeof(base->'resolucion')='object' THEN
  RETURN base || jsonb_build_object('propuesta',NULL,'numero_plazas',1,'politica_plazas',v_config,
   'plazas',jsonb_build_array(jsonb_build_object('numero_de_plaza',1,
    'estado',CASE WHEN base#>>'{resolucion,tipo}'='adjudicada' THEN 'cubierta' ELSE 'llamamiento_directo' END,
    'secuencia',0,'participacion_ref',base#>'{resolucion,participacion_ref}','orden_vigente',base#>'{resolucion,orden_vigente}',
    'responder_antes_de',NULL,'puede_sin_respuesta',false,'propuesta',NULL,'historial','[]'::jsonb)));
 END IF;
 FOR i IN 1..v_n LOOP
  SELECT a.* INTO ultimo FROM vec_bolsa_llamamientos.acto_plaza_oferta a
   WHERE a.oferta_ref=p_oferta AND a.numero_de_plaza=i AND a.registrado_en<=p_corte
   ORDER BY a.secuencia DESC LIMIT 1;
  IF ultimo.tipo IS NULL THEN v_estado:='vacante';
  ELSIF ultimo.tipo='adjudicada' THEN v_estado:=CASE WHEN v_gestion THEN 'pendiente_respuesta' ELSE 'cubierta' END;
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
  'propuesta',NULL,'numero_plazas',v_n,'politica_plazas',v_config,'plazas',v_plazas);
END $f$;

-- Publicación con número de plazas: reproduce la guarda de material de B54
-- (publicar_oferta_v2) y añade al final el número de plazas, de modo que la
-- autorización no pueda ampliarse a más plazas. Más de una plaza exige que la
-- versión de política usada tenga el apartado «plazas».
CREATE FUNCTION vec_bolsa_llamamientos.publicar_oferta_v3(
 p_oferta text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_datos jsonb,p_plazo jsonb,
 p_publicada timestamptz,p_vence timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_unidad text,p_ambito text,p_numero_plazas integer)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE d jsonb; v_calendarios text; v_material text; v_contexto text; x record; previa record; v_version bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_publicada IS NULL OR p_vence IS NULL
    OR p_bolsa IS NULL OR p_plazo IS NULL
    OR p_numero_plazas IS NULL OR p_numero_plazas NOT BETWEEN 1 AND 100
    OR p_unidad IS NULL OR p_unidad !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_unidad) NOT BETWEEN 1 AND 256
    OR p_ambito IS NULL OR p_ambito !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_ambito) NOT BETWEEN 1 AND 256
    OR jsonb_typeof(p_plazo) IS DISTINCT FROM 'object'
    OR (p_plazo->>'unidad'='horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>12)
    OR (p_plazo->>'unidad'<>'horas_naturales' AND
        (SELECT count(*) FROM jsonb_object_keys(p_plazo))<>10)
    OR NOT (p_plazo ?& ARRAY['regla_ref','huella_catalogo','unidad','cantidad','computo',
                              'ultimo_dia','ejemplo','calendarios','politica_version','municipio_sede'])
    OR jsonb_typeof(p_plazo->'calendarios') IS DISTINCT FROM 'array'
    OR jsonb_array_length(p_plazo->'calendarios') NOT BETWEEN 1 AND 16
    OR EXISTS(SELECT 1 FROM jsonb_array_elements(p_plazo->'calendarios') c
              WHERE jsonb_typeof(c) IS DISTINCT FROM 'string' OR c#>>'{}' !~ '^[A-Za-z0-9:_-]+$')
    OR p_plazo->>'regla_ref' IS NULL OR p_plazo->>'huella_catalogo' IS NULL
    OR p_plazo->>'unidad' IS NULL OR p_plazo->>'cantidad' IS NULL
    OR p_plazo->>'computo' IS NULL OR p_plazo->>'municipio_sede' IS NULL
    OR p_plazo->>'ultimo_dia' IS NULL OR coalesce(p_plazo->>'politica_version','') !~ '^[1-9][0-9]{0,9}$'
    OR (p_plazo->>'unidad'='horas_naturales' AND
        (p_plazo->>'apertura_en' IS NULL OR p_plazo->>'vence_en' IS NULL))
 THEN RAISE EXCEPTION 'B58: material de plazo o plazas incompleto' USING ERRCODE='22023'; END IF;
 SELECT string_agg(c#>>'{}',chr(30) ORDER BY n) INTO v_calendarios
 FROM jsonb_array_elements(p_plazo->'calendarios') WITH ORDINALITY a(c,n);
 v_material:=encode(sha256(convert_to(array_to_string(ARRAY[
  p_bolsa,
  to_char(p_publicada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  to_char(p_vence AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  p_plazo->>'regla_ref',p_plazo->>'huella_catalogo',p_plazo->>'unidad',
  p_plazo->>'cantidad',p_plazo->>'computo',p_plazo->>'municipio_sede',
  p_plazo->>'ultimo_dia',p_plazo->>'politica_version',v_calendarios
 ] || CASE WHEN p_plazo->>'unidad'='horas_naturales'
           THEN ARRAY[p_plazo->>'apertura_en',p_plazo->>'vence_en']
           ELSE ARRAY[]::text[] END || ARRAY[p_numero_plazas::text],chr(31)),'UTF8')),'hex');
 v_contexto:=encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||p_ambito||'","unidad_ref":"'||p_unidad||
  '"},"atributos":{"material_sha256":"'||v_material||'"}}','UTF8')),'hex');
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B58: decisión de oferta inválida' USING ERRCODE='42501'; END;
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto THEN
  RAISE EXCEPTION 'B58: plazo o plazas distintos de lo autorizado' USING ERRCODE='42501';
 END IF;
 v_version:=(p_plazo->>'politica_version')::bigint;
 IF p_numero_plazas>1 AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version pv
     WHERE pv.bolsa_ref=p_bolsa AND pv.version=v_version AND jsonb_typeof(pv.politica->'plazas')='object') THEN
  RAISE EXCEPTION 'B58: la política de la bolsa no admite varias plazas' USING ERRCODE='VBO08';
 END IF;
 SELECT * INTO STRICT x FROM vec_bolsa_llamamientos.publicar_oferta_v1(
  p_oferta,p_recibo,p_bolsa,p_actor,p_clave,p_datos,p_plazo,p_publicada,p_vence,
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_bolsa_llamamientos.plazas_oferta WHERE oferta_ref=x.oferta->>'oferta_ref';
 IF FOUND THEN
  IF previa.numero_plazas<>p_numero_plazas THEN
   RAISE EXCEPTION 'B58: clave de oferta con otro número de plazas' USING ERRCODE='VBO01'; END IF;
 ELSIF x.reutilizada THEN
  -- Oferta anterior publicada sin fila de plazas: solo equivale a una plaza.
  IF p_numero_plazas<>1 THEN
   RAISE EXCEPTION 'B58: clave de oferta con otro número de plazas' USING ERRCODE='VBO01'; END IF;
 ELSE
  INSERT INTO vec_bolsa_llamamientos.plazas_oferta(oferta_ref,bolsa_ref,numero_plazas,politica_version,registrada_en)
  VALUES(x.oferta->>'oferta_ref',p_bolsa,p_numero_plazas,v_version,clock_timestamp());
 END IF;
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(x.oferta->>'oferta_ref',greatest(clock_timestamp(),p_publicada)),x.reutilizada;
END $f$;

-- Registra un acto sobre una plaza. RRHH confirma exactamente lo que VEC
-- propone o registra la respuesta de quien ocupa la plaza; la secuencia
-- esperada evita actuar sobre un estado ya cambiado. Replay por clave.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(
 p_oferta text,p_recibo text,p_bolsa text,p_numero_de_plaza integer,p_tipo text,p_participacion text,
 p_secuencia_esperada integer,p_actor text,p_clave text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(oferta jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE o record; previo record; consumo record; d jsonb; v_ahora timestamptz;
 v_proy jsonb; v_plaza jsonb; v_huella text; v_responder timestamptz; v_orden bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_oferta IS NULL OR p_oferta !~ '^oferta:[0-9a-f]{64}$'
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:plaza-oferta:[0-9a-f]{64}$'
    OR p_bolsa IS NULL OR octet_length(p_bolsa) NOT BETWEEN 1 AND 256 OR p_bolsa<>btrim(p_bolsa)
    OR p_numero_de_plaza IS NULL OR p_numero_de_plaza NOT BETWEEN 1 AND 100
    OR p_tipo IS NULL OR p_tipo NOT IN ('adjudicada','aceptada','renuncia','sin_respuesta','llamamiento_directo')
    OR (p_tipo='llamamiento_directo')<>(p_participacion IS NULL)
    OR (p_participacion IS NOT NULL AND (p_participacion<>btrim(p_participacion) OR octet_length(p_participacion) NOT BETWEEN 1 AND 256))
    OR p_secuencia_esperada IS NULL OR p_secuencia_esperada NOT BETWEEN 0 AND 9999
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256 THEN
  RAISE EXCEPTION 'B58: acto de plaza inválido' USING ERRCODE='22023';
 END IF;
 -- El mismo cerrojo que la disposición y la resolución B28 de esta oferta.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resolucion-oferta:'||p_oferta,0));
 -- El instante se toma ya con el cerrojo: así ve el acto que otra sesión
 -- acaba de confirmar sobre la misma plaza y responde «la plaza ha cambiado».
 v_ahora:=clock_timestamp();
 SELECT * INTO o FROM vec_bolsa_llamamientos.oferta_publicada WHERE oferta_ref=p_oferta;
 IF NOT FOUND OR o.bolsa_ref<>p_bolsa THEN
  RAISE EXCEPTION 'B58: oferta inexistente en la bolsa' USING ERRCODE='23503';
 END IF;
 -- La decisión viva se consume antes de resolver el replay: un reintento no
 -- devuelve el recibo sin una autorización nueva y verificada.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B58: acto de plaza no autorizado' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_llamamientos_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B58: acto de plaza no autorizado' USING ERRCODE='42501';
 END IF;
 v_huella:=encode(sha256(convert_to(array_to_string(ARRAY[p_oferta,p_numero_de_plaza::text,p_tipo,
   coalesce(p_participacion,''),p_secuencia_esperada::text],chr(31)),'UTF8')),'hex');
 SELECT * INTO previo FROM vec_bolsa_llamamientos.acto_plaza_oferta WHERE oferta_ref=p_oferta AND clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella OR previo.recibo_ref<>p_recibo THEN
   RAISE EXCEPTION 'B58: clave reutilizada con otro acto' USING ERRCODE='VBO01';
  END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora),true;
  RETURN;
 END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.resolucion_oferta r WHERE r.oferta_ref=p_oferta) THEN
  RAISE EXCEPTION 'B58: oferta ya resuelta' USING ERRCODE='VBO02';
 END IF;
 IF v_ahora<o.vence_antes_de THEN
  RAISE EXCEPTION 'B58: plazo de disposición abierto' USING ERRCODE='VBO03';
 END IF;
 v_proy:=vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora);
 v_plaza:=v_proy->'plazas'->(p_numero_de_plaza-1);
 IF v_plaza IS NULL OR (v_plaza->>'numero_de_plaza')::int IS DISTINCT FROM p_numero_de_plaza THEN
  RAISE EXCEPTION 'B58: plaza inexistente' USING ERRCODE='22023';
 END IF;
 IF (v_plaza->>'secuencia')::int IS DISTINCT FROM p_secuencia_esperada THEN
  RAISE EXCEPTION 'B58: la plaza ha cambiado' USING ERRCODE='VBO04';
 END IF;
 IF p_tipo IN ('adjudicada','llamamiento_directo') THEN
  IF v_plaza->>'estado' IS DISTINCT FROM 'vacante' OR jsonb_typeof(v_plaza->'propuesta') IS DISTINCT FROM 'object'
     OR v_plaza#>>'{propuesta,tipo}' IS DISTINCT FROM (CASE p_tipo WHEN 'adjudicada' THEN 'adjudicar' ELSE 'llamamiento_directo' END)
     OR v_plaza#>>'{propuesta,participacion_ref}' IS DISTINCT FROM p_participacion THEN
   RAISE EXCEPTION 'B58: la propuesta ha cambiado' USING ERRCODE='VBO04';
  END IF;
  v_orden:=(v_plaza#>>'{propuesta,orden_vigente}')::bigint;
  IF p_tipo='adjudicada' AND jsonb_typeof(v_proy->'politica_plazas')='object' THEN
   v_responder:=v_ahora+((v_proy#>>'{politica_plazas,respuesta_horas}')::int*interval '1 hour');
  END IF;
 ELSE
  IF jsonb_typeof(v_proy->'politica_plazas') IS DISTINCT FROM 'object'
     OR v_plaza->>'estado' IS DISTINCT FROM 'pendiente_respuesta'
     OR v_plaza->>'participacion_ref' IS DISTINCT FROM p_participacion THEN
   RAISE EXCEPTION 'B58: la plaza ha cambiado' USING ERRCODE='VBO04';
  END IF;
  IF p_tipo='sin_respuesta' AND (v_plaza->>'puede_sin_respuesta')::boolean IS NOT TRUE THEN
   RAISE EXCEPTION 'B58: plazo de respuesta abierto' USING ERRCODE='VBO07';
  END IF;
  v_orden:=(v_plaza->>'orden_vigente')::bigint;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.acto_plaza_oferta(oferta_ref,numero_de_plaza,secuencia,tipo,participacion_ref,orden_vigente,
  responder_antes_de,recibo_ref,actor_ref,clave_idempotencia,huella_comando_sha256,decision_ref,auditoria_ref,registrado_en)
 VALUES(p_oferta,p_numero_de_plaza,p_secuencia_esperada+1,p_tipo,p_participacion,v_orden,
  v_responder,p_recibo,p_actor,p_clave,v_huella,consumo.decision_ref,consumo.auditoria_ref,v_ahora);
 INSERT INTO vec_bolsa_llamamientos.acto_plaza_oferta_outbox(recibo_ref,oferta_ref,bolsa_ref,numero_de_plaza,secuencia,tipo,creada_en)
 VALUES(p_recibo,p_oferta,p_bolsa,p_numero_de_plaza,p_secuencia_esperada+1,p_tipo,v_ahora);
 RETURN QUERY SELECT vec_bolsa_llamamientos.proyectar_oferta_v2(p_oferta,v_ahora),false;
END $f$;

-- La consulta de RRHH proyecta ya por plazas (misma firma y ACL).
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(p_bolsa text, p_corte timestamptz, p_limite integer)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
 SELECT coalesce(jsonb_agg(vec_bolsa_llamamientos.proyectar_oferta_v2(x.oferta_ref, p_corte) ORDER BY x.publicada_en DESC, x.oferta_ref), '[]'::jsonb)
   FROM (SELECT o.oferta_ref, o.publicada_en FROM vec_bolsa_llamamientos.oferta_publicada o
          WHERE o.bolsa_ref = p_bolsa AND o.publicada_en <= p_corte AND p_limite BETWEEN 1 AND 100
          ORDER BY o.publicada_en DESC, o.oferta_ref LIMIT greatest(least(p_limite, 100), 1)) x
$f$;

-- «Mi bolsa»: quien recibe una plaza la ve como adjudicada a su nombre; tras
-- su renuncia o falta de respuesta, como resuelta. La oferta aparece resuelta
-- para las demás personas cuando todas sus plazas están cerradas.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.listar_ofertas_candidato_v1(p_candidato_ref text, p_corte timestamptz)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumo_candidato_v1(ARRAY['consulta'], p_candidato_ref);
 RETURN (
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'oferta_ref', o.oferta_ref, 'bolsa', o.bolsa_ref, 'datos', o.datos,
   'publicada_en', to_char(o.publicada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'vence_antes_de', to_char(o.vence_antes_de,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'estado', CASE WHEN r.oferta_ref IS NOT NULL AND r.participacion_ref = p.participacion_ref THEN 'adjudicada_propia'
                  WHEN r.oferta_ref IS NOT NULL THEN 'resuelta'
                  WHEN pa.tipo IN ('adjudicada','aceptada') THEN 'adjudicada_propia'
                  WHEN pa.tipo IN ('renuncia','sin_respuesta') THEN 'resuelta'
                  WHEN p_corte < o.vence_antes_de THEN 'abierta'
                  WHEN vec_bolsa_llamamientos.proyectar_oferta_v2(o.oferta_ref, p_corte)->>'estado'
                       IN ('adjudicada','llamamiento_directo','cerrada') THEN 'resuelta'
                  ELSE 'pendiente_resolucion' END,
   'disposicion', CASE WHEN d.oferta_ref IS NULL THEN NULL ELSE jsonb_build_object('recibo', d.recibo_ref,
      'manifestada_en', to_char(d.manifestada_en,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   ORDER BY o.vence_antes_de, o.oferta_ref), '[]'::jsonb)
 FROM vec_bolsa_llamamientos.participaciones_vigentes_candidato_v1(p_candidato_ref) p
 JOIN vec_bolsa_llamamientos.oferta_publicada o ON o.bolsa_ref = p.bolsa_ref AND o.publicada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.disposicion_oferta d
   ON d.oferta_ref = o.oferta_ref AND d.participacion_ref = p.participacion_ref AND d.manifestada_en <= p_corte
 LEFT JOIN vec_bolsa_llamamientos.resolucion_oferta r ON r.oferta_ref = o.oferta_ref AND r.resuelta_en <= p_corte
 LEFT JOIN LATERAL (SELECT a.tipo FROM vec_bolsa_llamamientos.acto_plaza_oferta a
                     WHERE a.oferta_ref = o.oferta_ref AND a.participacion_ref = p.participacion_ref AND a.registrado_en <= p_corte
                     ORDER BY a.registrado_en DESC, a.secuencia DESC LIMIT 1) pa ON true
 WHERE p_corte IS NOT NULL
   AND ((r.oferta_ref IS NULL AND p_corte < o.vence_antes_de) OR d.oferta_ref IS NOT NULL));
END $f$;

-- La política admite, además, el apartado opcional «plazas» (duda 75). El
-- resto de la validación B54 queda igual; las versiones publicadas no cambian.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
 p_bolsa text,p_version_esperada bigint,p_politica jsonb,p_actor text,p_clave text,p_recibo text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(politica jsonb,reutilizada boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE consumo record; d jsonb; previa record; v_version bigint; v_huella text; v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR p_version_esperada IS NULL OR p_version_esperada<0 OR p_version_esperada>2147483646
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:politica-ofertas:[0-9a-f]{64}$'
    OR jsonb_typeof(p_politica) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica)) NOT IN (3,4)
    OR NOT (p_politica ?& ARRAY['plazo','adjudicacion','no_cubierta'])
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica))=4 AND NOT (p_politica ? 'plazas'))
    -- B58: apartado opcional de plazas. Solo valores que ejecuta la proyección.
    OR (p_politica ? 'plazas' AND (
        jsonb_typeof(p_politica->'plazas') IS DISTINCT FROM 'object'
        OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazas'))<>3
        OR NOT (p_politica->'plazas' ?& ARRAY['llamada','respuesta_horas','tras_renuncia'])
        OR coalesce(p_politica#>>'{plazas,llamada}','') NOT IN ('simultanea','sucesiva')
        OR jsonb_typeof(p_politica#>'{plazas,respuesta_horas}') IS DISTINCT FROM 'number'
        OR coalesce(p_politica#>>'{plazas,respuesta_horas}','') !~ '^([1-9][0-9]?|[1-6][0-9]{2}|7[01][0-9]|720)$'
        OR coalesce(p_politica#>>'{plazas,tras_renuncia}','') NOT IN ('siguiente_en_orden','llamamiento_directo')))
    OR jsonb_typeof(p_politica->'plazo') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo'))<>4
    OR NOT (p_politica->'plazo' ?& ARRAY['unidad','cantidad','computo','municipio_sede'])
    OR coalesce(p_politica#>>'{plazo,unidad}','') NOT IN ('dias_habiles','dias_naturales','horas_naturales')
    OR jsonb_typeof(p_politica#>'{plazo,cantidad}') IS DISTINCT FROM 'number'
    OR coalesce(p_politica#>>'{plazo,cantidad}','') !~ '^[0-9]{1,3}$'
    OR ((p_politica#>>'{plazo,unidad}' IN ('dias_habiles','dias_naturales')
             AND (p_politica#>>'{plazo,cantidad}')::int BETWEEN 1 AND 30
             AND p_politica#>>'{plazo,computo}'='administrativo')
         OR (p_politica#>>'{plazo,unidad}'='horas_naturales'
             AND (p_politica#>>'{plazo,cantidad}')::int BETWEEN 1 AND 720
             AND p_politica#>>'{plazo,computo}'='continuo_utc')) IS NOT TRUE
    OR coalesce(p_politica#>>'{plazo,municipio_sede}','') !~ '^[0-9]{5}$'
    OR jsonb_typeof(p_politica->'adjudicacion') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))<>2
    OR p_politica#>>'{adjudicacion,criterio}' IS DISTINCT FROM 'orden_vigente'
    OR p_politica#>>'{adjudicacion,elegibilidad}' IS DISTINCT FROM 'disposicion_en_plazo'
    OR jsonb_typeof(p_politica->'no_cubierta') IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'no_cubierta'))<>2
    OR p_politica#>>'{no_cubierta,accion}' IS DISTINCT FROM 'llamamiento_directo'
    OR p_politica#>>'{no_cubierta,condicion}' IS DISTINCT FROM 'sin_disposiciones_elegibles'
 THEN RAISE EXCEPTION 'B47: política de ejemplo inválida' USING ERRCODE='22023'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B47: autorización inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'bolsa.politica_ofertas.publicar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
    OR d->>'finalidad' IS DISTINCT FROM 'gobierno_politica_ofertas_bolsa'
    OR d->>'recurso_ref' IS DISTINCT FROM p_bolsa
 THEN RAISE EXCEPTION 'B47: publicación no autorizada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:politica-ofertas:'||p_bolsa,0));
 v_huella:=encode(sha256(convert_to(p_politica::text,'UTF8')),'hex');
 SELECT * INTO previa FROM vec_bolsa_llamamientos.politica_ofertas_version
  WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF FOUND THEN
  IF previa.actor_ref<>p_actor OR previa.version_esperada<>p_version_esperada
     OR previa.huella_sha256<>v_huella OR previa.recibo_ref<>p_recibo
  THEN RAISE EXCEPTION 'B47: clave reutilizada' USING ERRCODE='VBP01'; END IF;
  RETURN QUERY SELECT jsonb_build_object('bolsa_ref',previa.bolsa_ref,'version',previa.version,
   'huella_sha256',previa.huella_sha256,'ejemplo',true,'configurada',true,'politica',previa.politica,
   'recibo_ref',previa.recibo_ref,
   'publicada_en',to_char(previa.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),true;
  RETURN;
 END IF;
 SELECT coalesce(max(version),0) INTO v_version FROM vec_bolsa_llamamientos.politica_ofertas_version WHERE bolsa_ref=p_bolsa;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B47: versión esperada obsoleta' USING ERRCODE='VBP01';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref=p_bolsa) THEN
  RAISE EXCEPTION 'B47: bolsa no constituida' USING ERRCODE='23503'; END IF;
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
  bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,version_esperada,
  recibo_ref,publicada_en,decision_ref,auditoria_ref)
 VALUES(p_bolsa,v_version+1,p_politica,v_huella,true,p_actor,p_clave,p_version_esperada,
  p_recibo,v_ahora,consumo.decision_ref,consumo.auditoria_ref);
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_outbox(recibo_ref,bolsa_ref,version,huella_sha256,creada_en)
 VALUES(p_recibo,p_bolsa,v_version+1,v_huella,v_ahora);
 RETURN QUERY SELECT vec_bolsa_llamamientos.leer_politica_ofertas_v1(p_bolsa),false;
END $f$;

DO $acl$
DECLARE f regprocedure;
 publicas regprocedure[]:=ARRAY[
  'vec_bolsa_llamamientos.publicar_oferta_v3(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer)'::regprocedure,
  'vec_bolsa_llamamientos.registrar_acto_plaza_oferta_v1(text,text,text,integer,text,text,integer,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure];
 internas regprocedure[]:=ARRAY['vec_bolsa_llamamientos.proyectar_oferta_v2(text,timestamptz)'::regprocedure];
 cerradas regprocedure[]:=ARRAY[
  'vec_bolsa_llamamientos.publicar_oferta_v2(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)'::regprocedure,
  'vec_bolsa_llamamientos.resolver_oferta_v1(text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure];
BEGIN
 FOREACH f IN ARRAY publicas||internas LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas LOOP
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor',f::text);
 END LOOP;
 -- Una sola vía para publicar y resolver: la de plazas.
 FOREACH f IN ARRAY cerradas LOOP
  EXECUTE format('REVOKE EXECUTE ON FUNCTION %s FROM vec_bolsa_llamamientos_ejecutor',f::text);
 END LOOP;
 FOREACH f IN ARRAY publicas||internas||cerradas LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
             WHERE p.oid=f AND a.grantee<>p.proowner
               AND (a.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole OR a.is_grantable OR NOT f=ANY(publicas)))
     OR (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_bolsa_llamamientos_propietario'::regrole
     OR NOT (SELECT prosecdef FROM pg_proc WHERE oid=f) THEN
   RAISE EXCEPTION 'B58: ACL de % incorrecta',f USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.acto_plaza_oferta','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.plazas_oferta','SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.acto_plaza_oferta_outbox','SELECT,INSERT,UPDATE,DELETE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.listar_ofertas_bolsa_v1(text,timestamptz,integer)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
  RAISE EXCEPTION 'B58: privilegios de tabla o consulta incorrectos' USING ERRCODE='55000';
 END IF;
END $acl$;
COMMIT;
