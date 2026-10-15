\set ON_ERROR_STOP on
-- B118: política versionada y prelación contextual de una modalidad ofrecida.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000118',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int<180000
 OR current_user<>'vec_bolsa_llamamientos_propietario'
 OR to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.origen_llamamiento_ct_v1') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(text,text,text,bigint)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(jsonb)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.motivo_relacion_oferta_v1(text,text,boolean,text,text,timestamptz,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.eventos_relacion_oferta_v1(text,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(text,text,text,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1(text,text,bigint,int,bytea,bytea,jsonb)') IS NOT NULL
 OR encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.publicar_politica_ofertas_v1(text,bigint,jsonb,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')<>'84890160e52cd550870dada07b2e15890655c93faa5b6c3164fc8cada94bb283'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure),'UTF8')),'hex')<>'470e557538452978dddfc35c431c8293408b6701efa09ee9a5b313cbe49dc37d'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(text)'::regprocedure),'UTF8')),'hex')<>'88f1a5947dc7ed9526065bed6b1d8c09165c89a066d556813643ab608563f555'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb)'::regprocedure),'UTF8')),'hex')<>'26cfbe7c5d0f9691d6a597c0c3a914848b02a14e38e985725dc2994fbafebeab'
 OR encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_llamamientos.registrar_contrato_participacion_v1(jsonb,text,timestamptz,bigint)'::regprocedure),'UTF8')),'hex')<>'d65972b827be3a86c5959f8a683a965aa95cb5726adeb3511ea26242b457d0f7'
 THEN RAISE EXCEPTION 'B118: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Validación de la parte nueva; las claves de modalidad y causa son de catálogo.
CREATE FUNCTION vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(c jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb;q jsonb;l jsonb; n int;
BEGIN
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR octet_length(c::text)>16384
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(c) k WHERE k NOT IN
   ('fuente_normativa','mejora'))
 THEN RETURN false; END IF;
 IF c='{}'::jsonb THEN RETURN true; END IF;
 IF c->>'fuente_normativa' IS NULL
 OR octet_length(c->>'fuente_normativa') NOT BETWEEN 1 AND 2000
 OR c->>'fuente_normativa'<>btrim(c->>'fuente_normativa')
 THEN RETURN false; END IF;
 IF NOT (c ? 'mejora') THEN RETURN true; END IF;
 m:=c->'mejora';
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(m)) NOT BETWEEN 6 AND 7
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('habilitada','modalidades_destino','condiciones_actuales','maximo_vacante_meses',
    'reduccion_por_modalidades_previas','causas_destino','supuestos_reduccion_vacante'))
 OR m ?& ARRAY['habilitada','modalidades_destino','condiciones_actuales',
   'maximo_vacante_meses','reduccion_por_modalidades_previas','causas_destino'] IS NOT TRUE
 OR jsonb_typeof(m->'habilitada') IS DISTINCT FROM 'boolean'
 OR jsonb_typeof(m->'modalidades_destino') IS DISTINCT FROM 'array'
 OR jsonb_array_length(m->'modalidades_destino') NOT BETWEEN 1 AND 100
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(m->'modalidades_destino') x
  WHERE jsonb_typeof(x) IS DISTINCT FROM 'string' OR x#>>'{}' !~ '^[a-z][a-z0-9._-]{1,79}$')
 OR (SELECT count(*) FROM jsonb_array_elements_text(m->'modalidades_destino'))
  <> (SELECT count(DISTINCT x) FROM jsonb_array_elements_text(m->'modalidades_destino') x)
 OR jsonb_typeof(m->'causas_destino') IS DISTINCT FROM 'array'
 OR jsonb_array_length(m->'causas_destino') NOT BETWEEN 1 AND 100
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(m->'causas_destino') x
  WHERE jsonb_typeof(x) IS DISTINCT FROM 'string' OR x#>>'{}' !~ '^[a-z][a-z0-9._-]{1,79}$')
 OR (SELECT count(*) FROM jsonb_array_elements_text(m->'causas_destino'))
  <> (SELECT count(DISTINCT x) FROM jsonb_array_elements_text(m->'causas_destino') x)
 OR jsonb_typeof(m->'maximo_vacante_meses') IS DISTINCT FROM 'number'
 OR m->>'maximo_vacante_meses' !~ '^[1-9][0-9]{0,3}$'
 OR (m->>'maximo_vacante_meses')::int>1200
 OR jsonb_typeof(m->'reduccion_por_modalidades_previas') IS DISTINCT FROM 'array'
 OR jsonb_array_length(m->'reduccion_por_modalidades_previas')>100
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(m->'reduccion_por_modalidades_previas') x
  WHERE jsonb_typeof(x) IS DISTINCT FROM 'string' OR x#>>'{}' !~ '^[a-z][a-z0-9._-]{1,79}$')
 OR (SELECT count(*) FROM jsonb_array_elements_text(m->'reduccion_por_modalidades_previas'))
  <> (SELECT count(DISTINCT x) FROM jsonb_array_elements_text(m->'reduccion_por_modalidades_previas') x)
 OR jsonb_typeof(m->'condiciones_actuales') IS DISTINCT FROM 'array'
 OR jsonb_array_length(m->'condiciones_actuales') NOT BETWEEN 1 AND 100
 THEN RETURN false; END IF;
 IF m ? 'supuestos_reduccion_vacante' THEN
  l:=m->'supuestos_reduccion_vacante';
  IF jsonb_typeof(l) IS DISTINCT FROM 'array'
   OR jsonb_array_length(l) NOT BETWEEN 1 AND 100
   OR EXISTS(SELECT 1 FROM jsonb_array_elements(l) x
    WHERE jsonb_typeof(x) IS DISTINCT FROM 'string' OR x#>>'{}' !~ '^[a-z][a-z0-9._-]{1,79}$')
   OR (SELECT count(*) FROM jsonb_array_elements_text(l))
    <> (SELECT count(DISTINCT x) FROM jsonb_array_elements_text(l) x)
   OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(l) x(v)
    WHERE NOT EXISTS(SELECT 1 FROM jsonb_array_elements(m->'condiciones_actuales') q(valor)
     WHERE q.valor->>'modalidad'=x.v))
  THEN RETURN false; END IF;
 END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements_text(m->'reduccion_por_modalidades_previas') x(v)
   WHERE NOT EXISTS(SELECT 1 FROM jsonb_array_elements(m->'condiciones_actuales') q(valor)
     WHERE q.valor->>'modalidad'=x.v)) THEN RETURN false; END IF;
 FOR q IN SELECT value FROM jsonb_array_elements(m->'condiciones_actuales') LOOP
  IF jsonb_typeof(q) IS DISTINCT FROM 'object'
  OR q->>'modalidad' IS NULL OR q->>'modalidad' !~ '^[a-z][a-z0-9._-]{1,79}$'
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(q) k WHERE k NOT IN
    ('modalidad','causa','requiere_baja_comprobada','meses_transcurridos','meses_duracion_inicial'))
  OR (q ? 'causa' AND (q->>'causa' IS NULL OR q->>'causa' !~ '^[a-z][a-z0-9._-]{1,79}$'))
  OR (q ? 'requiere_baja_comprobada' AND jsonb_typeof(q->'requiere_baja_comprobada') IS DISTINCT FROM 'boolean')
  THEN RETURN false; END IF;
  FOREACH n IN ARRAY ARRAY[1,2] LOOP
   l:=CASE n WHEN 1 THEN q->'meses_transcurridos' ELSE q->'meses_duracion_inicial' END;
   IF l IS NOT NULL AND (jsonb_typeof(l) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(l))<>2
    OR l ?& ARRAY['meses','incluye_limite'] IS NOT TRUE
    OR jsonb_typeof(l->'meses') IS DISTINCT FROM 'number'
    OR l->>'meses' !~ '^[1-9][0-9]{0,3}$'
    OR (l->>'meses')::int>1200
    OR jsonb_typeof(l->'incluye_limite') IS DISTINCT FROM 'boolean')
   THEN RETURN false; END IF;
  END LOOP;
 END LOOP;
 RETURN true;
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(jsonb) FROM PUBLIC;

-- B13 enlazaba únicamente llamamientos del puente de desarrollo. Los eventos
-- de contratos nacidos del origen B110 tienen que conservar su participación.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_contrato_participacion_v1(p_evento jsonb, p_huella_sha256 text, p_origen_creada_en timestamp with time zone, p_origen_posicion bigint)
 RETURNS TABLE(reutilizado boolean, participacion_ref text, en_cuarentena boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE v_previo record; v_llamamiento record; v_origen record; v_ref text; v_clave text := '^[a-z][a-z0-9._-]{1,79}$';
 v_opaca text := '^[A-Za-z0-9][A-Za-z0-9:._/-]*$';
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='registro de contrato no autorizado';
 END IF;
 IF p_evento IS NULL OR jsonb_typeof(p_evento) <> 'object' OR octet_length(p_evento::text) > 16384
    OR EXISTS (SELECT 1 FROM jsonb_each(p_evento) e WHERE jsonb_typeof(e.value) = 'string' AND octet_length(e.value #>> '{}') > 512)
    OR p_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(p_evento::text, 'UTF8')), 'hex')
    OR p_origen_creada_en IS NULL OR NOT isfinite(p_origen_creada_en)
    OR p_origen_posicion IS NULL OR p_origen_posicion < 0
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_evento) k) IS DISTINCT FROM
       ARRAY['categoria_ref','causa_clave','esquema','evento_ref','expediente_ref','fin_previsto','inicio',
             'llamamiento_ref','modalidad_clave','ocurrido_en','organizacion_ref','origen_ref','tipo']
    OR p_evento->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.contrato-bolsa.v1'
    OR jsonb_typeof(p_evento->'tipo') IS DISTINCT FROM 'string' OR p_evento->>'tipo' !~ '^[a-z][a-z0-9_]{1,39}$'
    OR jsonb_typeof(p_evento->'origen_ref') IS DISTINCT FROM 'string' OR p_evento->>'origen_ref' !~ v_opaca
    OR jsonb_typeof(p_evento->'organizacion_ref') IS DISTINCT FROM 'string' OR p_evento->>'organizacion_ref' !~ v_opaca
    OR jsonb_typeof(p_evento->'expediente_ref') IS DISTINCT FROM 'string' OR p_evento->>'expediente_ref' !~ v_opaca
    OR jsonb_typeof(p_evento->'llamamiento_ref') IS DISTINCT FROM 'string' OR p_evento->>'llamamiento_ref' !~ v_opaca
    OR p_evento->>'evento_ref' IS DISTINCT FROM 'evento:ct:contrato-bolsa:' || encode(sha256(convert_to(
         (p_evento->>'tipo') || chr(31) || (p_evento->>'origen_ref'), 'UTF8')), 'hex')
    OR NOT (jsonb_typeof(p_evento->'modalidad_clave') = 'null' OR (jsonb_typeof(p_evento->'modalidad_clave') = 'string' AND p_evento->>'modalidad_clave' ~ v_clave))
    OR NOT (jsonb_typeof(p_evento->'causa_clave') = 'null' OR (jsonb_typeof(p_evento->'causa_clave') = 'string' AND p_evento->>'causa_clave' ~ v_clave))
    OR NOT (jsonb_typeof(p_evento->'categoria_ref') = 'null' OR (jsonb_typeof(p_evento->'categoria_ref') = 'string' AND p_evento->>'categoria_ref' ~ v_opaca))
    OR NOT vec_bolsa_llamamientos.instante_contrato_valido(p_evento->'inicio', true)
    OR NOT vec_bolsa_llamamientos.instante_contrato_valido(p_evento->'fin_previsto', true)
    OR NOT vec_bolsa_llamamientos.instante_contrato_valido(p_evento->'ocurrido_en', false)
    OR (p_evento->>'fin_previsto')::timestamptz < (p_evento->>'inicio')::timestamptz THEN
  RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='evento de contrato inválido';
 END IF;
 v_ref := p_evento->>'evento_ref';
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:contrato-participacion:' || v_ref, 0));
 SELECT c.huella_sha256, c.participacion_ref, c.origen_posicion INTO v_previo
   FROM vec_bolsa_llamamientos.contrato_participacion c WHERE c.evento_ref = v_ref;
 IF FOUND THEN
  IF v_previo.huella_sha256 <> p_huella_sha256 OR v_previo.origen_posicion <> p_origen_posicion THEN
   INSERT INTO vec_bolsa_llamamientos.contrato_participacion_cuarentena(evento_ref, huella_sha256, evento, origen_ref,
     origen_creada_en, origen_posicion, recibido_en)
   VALUES (v_ref, p_huella_sha256, p_evento, p_evento->>'origen_ref', date_trunc('microseconds', p_origen_creada_en),
     p_origen_posicion, date_trunc('microseconds', clock_timestamp()))
   ON CONFLICT DO NOTHING;
   RETURN QUERY SELECT false, NULL::text, true;
   RETURN;
  END IF;
  RETURN QUERY SELECT true, v_previo.participacion_ref, false;
  RETURN;
 END IF;
 SELECT o.bolsa_ref,o.participacion_ref,o.expediente_ref,
                 o.organizacion_ref,o.categoria_ref
   INTO v_origen
   FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o
  WHERE o.llamamiento_ref=p_evento->>'llamamiento_ref';
 IF FOUND THEN
  IF v_origen.expediente_ref IS DISTINCT FROM p_evento->>'expediente_ref'
     OR v_origen.organizacion_ref IS DISTINCT FROM p_evento->>'organizacion_ref'
     OR v_origen.categoria_ref IS DISTINCT FROM p_evento->>'categoria_ref' THEN
   RAISE EXCEPTION 'B118: contrato CT y origen Bolsa divergentes' USING ERRCODE='23503';
  END IF;
  SELECT v_origen.bolsa_ref,v_origen.participacion_ref AS participacion
    INTO v_llamamiento;
 ELSE
  SELECT l.bolsa_ref, convert_from(i.registro_canonico, 'UTF8')::jsonb #>> '{propuesta,participacion_seleccionada_ref}' AS participacion
    INTO v_llamamiento
    FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
    JOIN vec_bolsa_llamamientos.integracion_desarrollo i ON i.operacion_ref = l.operacion_ref
   WHERE l.llamamiento_ref = p_evento->>'llamamiento_ref';
  IF v_llamamiento.participacion IS NULL THEN
   v_llamamiento.bolsa_ref := NULL;
  END IF;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
   evento_ref, huella_sha256, evento, origen_ref, origen_creada_en, origen_posicion, tipo, organizacion_ref, expediente_ref,
   llamamiento_ref, participacion_ref, bolsa_ref, inicio, fin_previsto, modalidad_clave, categoria_ref,
   causa_clave, ocurrido_en, recibido_en)
 VALUES (v_ref, p_huella_sha256, p_evento, p_evento->>'origen_ref', date_trunc('microseconds', p_origen_creada_en), p_origen_posicion,
   p_evento->>'tipo', p_evento->>'organizacion_ref', p_evento->>'expediente_ref', p_evento->>'llamamiento_ref',
   v_llamamiento.participacion, v_llamamiento.bolsa_ref, (p_evento->>'inicio')::timestamptz,
   (p_evento->>'fin_previsto')::timestamptz, p_evento->>'modalidad_clave', p_evento->>'categoria_ref',
   p_evento->>'causa_clave', (p_evento->>'ocurrido_en')::timestamptz, date_trunc('microseconds', clock_timestamp()));
 RETURN QUERY SELECT false, v_llamamiento.participacion, false;
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION USING ERRCODE='VBC01', MESSAGE='origen de contrato ya registrado con otro evento';
END $function$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_contrato_participacion_v1(jsonb,text,timestamptz,bigint) FROM PUBLIC;

-- Decide solo la excepción por relación. Las demás exclusiones de B6 se
-- aplican después, en el orden contextual. B13 no acredita una baja causal:
-- una sustitución con esa exigencia queda pendiente de comprobación por RRHH.
CREATE FUNCTION vec_bolsa_llamamientos.motivo_relacion_oferta_v1(
 p_participacion text,p_situacion text,p_trabajo_cesado boolean,
 p_modalidad text,p_causa text,p_corte timestamptz,p_criterios jsonb)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET statement_timeout='10s' AS $f$
DECLARE v_candidato text;v_candidatos int;v_activos int:=0;
        r record;c jsonb;v_encaja boolean;v_incompleto boolean;
        limite jsonb;v_meses int;v_borde timestamptz;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR p_participacion IS NULL OR p_corte IS NULL OR NOT isfinite(p_corte)
 OR p_modalidad IS NULL OR p_modalidad !~ '^[a-z][a-z0-9._-]{1,79}$'
 THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
 -- La ausencia de criterios conserva exactamente el orden general B6.
 IF p_criterios IS NULL OR p_criterios='{}'::jsonb THEN RETURN 'libre'; END IF;
 IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(p_criterios) IS NOT TRUE
 THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
 SELECT count(DISTINCT vc.candidato_ref),min(vc.candidato_ref)
 INTO v_candidatos,v_candidato
 FROM vec_bolsa_llamamientos.vinculo_candidato vc
 WHERE vc.participacion_ref=p_participacion;
 IF v_candidatos<>1 THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion cp
   LEFT JOIN vec_bolsa_llamamientos.origen_llamamiento_ct_v1 origen
     ON origen.llamamiento_ref=cp.llamamiento_ref
    AND origen.expediente_ref=cp.expediente_ref
    AND origen.organizacion_ref=cp.organizacion_ref
    AND origen.categoria_ref=cp.categoria_ref
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc
     ON vc.participacion_ref=coalesce(cp.participacion_ref,origen.participacion_ref)
    AND vc.candidato_ref=v_candidato
   JOIN vec_bolsa_llamamientos.contrato_participacion_cuarentena q
     ON q.evento_ref=cp.evento_ref AND q.recibido_en<=p_corte
   WHERE cp.ocurrido_en<=p_corte AND cp.recibido_en<=p_corte)
 THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
 -- Una incorporación sigue abierta hasta que conste su cese. Se revisan
 -- todas las participaciones vinculadas a la persona, no solo esta bolsa.
 FOR r IN SELECT DISTINCT cp.evento_ref,cp.modalidad_clave,cp.causa_clave,
                 cp.inicio,cp.fin_previsto
 FROM vec_bolsa_llamamientos.contrato_participacion cp
 LEFT JOIN vec_bolsa_llamamientos.origen_llamamiento_ct_v1 origen
   ON origen.llamamiento_ref=cp.llamamiento_ref
  AND origen.expediente_ref=cp.expediente_ref
  AND origen.organizacion_ref=cp.organizacion_ref
  AND origen.categoria_ref=cp.categoria_ref
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=coalesce(cp.participacion_ref,origen.participacion_ref)
  AND vc.candidato_ref=v_candidato
 WHERE cp.tipo='incorporacion' AND (cp.inicio IS NULL OR cp.inicio<=p_corte)
   AND cp.ocurrido_en<=p_corte AND cp.recibido_en<=p_corte
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena q
     WHERE q.evento_ref=cp.evento_ref AND q.recibido_en<=p_corte)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion fin
     WHERE fin.tipo='cese' AND fin.llamamiento_ref=cp.llamamiento_ref
       AND fin.ocurrido_en<=p_corte AND fin.recibido_en<=p_corte AND NOT EXISTS(
        SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena q
        WHERE q.evento_ref=fin.evento_ref AND q.recibido_en<=p_corte))
 LOOP
  v_activos:=v_activos+1;
  IF v_activos>1 THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
  IF r.modalidad_clave IS NULL OR r.inicio IS NULL OR r.inicio>p_corte
  THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
  IF p_criterios#>>'{mejora,habilitada}' IS DISTINCT FROM 'true'
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements_text(
       coalesce(p_criterios#>'{mejora,modalidades_destino}','[]'::jsonb)) d(v)
       WHERE d.v=p_modalidad)
  THEN RETURN 'trabajando'; END IF;
  IF p_causa IS NULL OR p_causa !~ '^[a-z][a-z0-9._-]{1,79}$'
  THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
  IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements_text(
       p_criterios#>'{mejora,causas_destino}') d(v) WHERE d.v=p_causa)
  THEN RETURN 'trabajando'; END IF;
  v_encaja:=false;v_incompleto:=false;
  FOR c IN SELECT x.value FROM jsonb_array_elements(
    p_criterios#>'{mejora,condiciones_actuales}') x LOOP
   IF c->>'modalidad' IS DISTINCT FROM r.modalidad_clave THEN CONTINUE; END IF;
   IF c ? 'causa' AND r.causa_clave IS NULL THEN
    v_incompleto:=true;CONTINUE; END IF;
   IF c ? 'causa' AND c->>'causa' IS DISTINCT FROM r.causa_clave
   THEN CONTINUE; END IF;
   -- No se deduce la baja comprobada del nombre de una causa.
   IF c->>'requiere_baja_comprobada'='true' THEN
    v_incompleto:=true;CONTINUE; END IF;
   limite:=c->'meses_transcurridos';
   IF limite IS NOT NULL THEN
    v_meses:=(limite->>'meses')::int;
    v_borde:=r.inicio+make_interval(months=>v_meses);
    IF p_corte>v_borde OR (p_corte=v_borde AND limite->>'incluye_limite'='false')
    THEN CONTINUE; END IF;
   END IF;
   limite:=c->'meses_duracion_inicial';
   IF limite IS NOT NULL THEN
    IF r.fin_previsto IS NULL OR r.fin_previsto<=r.inicio THEN
     v_incompleto:=true;CONTINUE; END IF;
    v_meses:=(limite->>'meses')::int;
    v_borde:=r.inicio+make_interval(months=>v_meses);
    IF r.fin_previsto>v_borde OR (r.fin_previsto=v_borde AND limite->>'incluye_limite'='false')
    THEN CONTINUE; END IF;
   END IF;
   v_encaja:=true;EXIT;
  END LOOP;
  IF NOT v_encaja THEN
   IF v_incompleto THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
   RETURN 'trabajando';
 END IF;
 END LOOP;
 IF v_activos=0 THEN
  IF p_situacion='trabajando' AND p_trabajo_cesado IS NOT TRUE
  THEN RETURN 'pendiente_comprobacion_rrhh'; END IF;
  RETURN 'libre';
 END IF;
 -- R1a diagnostica la excepción. El efecto requiere un acto posterior que
 -- compruebe la reducción real del plazo máximo de la vacante.
 RETURN 'pendiente_comprobacion_rrhh';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.motivo_relacion_oferta_v1(
 text,text,boolean,text,text,timestamptz,jsonb) FROM PUBLIC;

-- Referencias de los eventos recibidos antes del corte que sustentan la
-- evaluación. La lectura conserva el conjunto exacto en el origen B110.
CREATE FUNCTION vec_bolsa_llamamientos.eventos_relacion_oferta_v1(
 p_participacion text,p_corte timestamptz) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT coalesce(jsonb_agg(x.evento_ref ORDER BY x.ocurrido_en,x.evento_ref),'[]'::jsonb)
 FROM (SELECT DISTINCT cp.evento_ref,cp.ocurrido_en
  FROM vec_bolsa_llamamientos.vinculo_candidato vc
  JOIN vec_bolsa_llamamientos.contrato_participacion cp ON true
  LEFT JOIN vec_bolsa_llamamientos.origen_llamamiento_ct_v1 origen
    ON origen.llamamiento_ref=cp.llamamiento_ref
   AND origen.expediente_ref=cp.expediente_ref
   AND origen.organizacion_ref=cp.organizacion_ref
   AND origen.categoria_ref=cp.categoria_ref
  JOIN vec_bolsa_llamamientos.vinculo_candidato vc2
    ON vc2.candidato_ref=vc.candidato_ref
   AND vc2.participacion_ref=coalesce(cp.participacion_ref,origen.participacion_ref)
   WHERE vc.participacion_ref=p_participacion AND cp.tipo IN ('incorporacion','cese')
    AND cp.ocurrido_en<=p_corte
    AND cp.recibido_en<=p_corte
    AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena q
      WHERE q.evento_ref=cp.evento_ref AND q.recibido_en<=p_corte)) x
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.eventos_relacion_oferta_v1(text,timestamptz) FROM PUBLIC;

-- B6 se conserva; esta variante usa su prelación para una oferta concreta.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(p_bolsa_ref text, p_modalidad text, p_causa text, p_en timestamp with time zone)
 RETURNS TABLE(politica_ref text, version_politica bigint, criterio text, tipo_lista text, reposicion text, provisional boolean, rotulo text, actor text, vigente_desde timestamp with time zone, participacion_ref text, orden_acta bigint, orden_vigente bigint, situacion text, razon text, eventos_relacion jsonb)
 LANGUAGE sql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
 WITH criterio_oferta AS (
  SELECT po.politica->'criterios_ofrecimiento' AS criterios
  FROM vec_bolsa_llamamientos.politica_ofertas_version po
  WHERE po.bolsa_ref=p_bolsa_ref AND po.publicada_en<=p_en
  ORDER BY po.version DESC LIMIT 1
 ), politica AS (
  SELECT p.* FROM vec_bolsa_llamamientos.politica_orden_bolsa p
   WHERE p.bolsa_ref=p_bolsa_ref AND p.vigente_desde<=p_en AND (p.vigente_hasta IS NULL OR p.vigente_hasta>p_en)
   ORDER BY p.version DESC LIMIT 1
 ), ceses AS MATERIALIZED (
  SELECT ec.* FROM vec_bolsa_llamamientos.estado_cese_bolsa_lote_interno_v2(
   coalesce((SELECT array_agg(e.participacion_ref) FROM vec_bolsa_llamamientos.constitucion c
     JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    WHERE c.bolsa_ref=p_bolsa_ref),ARRAY[]::text[]),p_en) ec
 ), pendientes AS MATERIALIZED (
  SELECT rp.participacion_ref FROM vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(p_bolsa_ref,p_en) rp
 ), base AS (
  SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         coalesce(rc.en_restriccion,false) AS cese_restringido,
         coalesce(rc.cese_pendiente,false) AS cese_pendiente,
         coalesce(rc.trabajo_cesado,false) AS trabajo_cesado,
         (pp.participacion_ref IS NOT NULL) AS respuesta_pendiente,
         em.motivo_relacion,
         vec_bolsa_llamamientos.eventos_relacion_oferta_v1(e.participacion_ref,p_en) AS eventos_relacion,
         ((s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)
           OR coalesce(rc.trabajo_cesado,false))
          AND em.motivo_relacion='libre' AND NOT coalesce(rc.en_restriccion,false)
          AND NOT coalesce(rc.cese_pendiente,false)) AS turno_sin_respuesta,
         ((s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)
           OR coalesce(rc.trabajo_cesado,false))
          AND em.motivo_relacion='libre' AND NOT coalesce(rc.en_restriccion,false)
          AND NOT coalesce(rc.cese_pendiente,false) AND pp.participacion_ref IS NULL) AS ocupa_turno,
         r.aplicada_en AS repuesta_en,
         EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.penalizacion_orden_sancion pe
                  WHERE pe.bolsa_ref=c.bolsa_ref AND pe.participacion_ref=e.participacion_ref AND pe.aplicada_en<=p_en
                    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion rv
                                     WHERE rv.sancion_ref=pe.sancion_ref AND rv.registrada_en<=p_en)) AS penalizada
    FROM vec_bolsa_llamamientos.constitucion c
    LEFT JOIN criterio_oferta co ON true
    JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
    JOIN LATERAL (SELECT sp.situacion,sp.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion sp WHERE sp.participacion_ref=e.participacion_ref AND sp.desde<=p_en ORDER BY sp.desde DESC LIMIT 1) s ON true
    LEFT JOIN ceses rc ON rc.participacion_ref=e.participacion_ref
    LEFT JOIN LATERAL (SELECT vec_bolsa_llamamientos.motivo_relacion_oferta_v1(
       e.participacion_ref,s.situacion,coalesce(rc.trabajo_cesado,false),
       p_modalidad,p_causa,p_en,co.criterios) AS motivo_relacion) em ON true
    LEFT JOIN pendientes pp ON pp.participacion_ref=e.participacion_ref
    LEFT JOIN LATERAL (SELECT ro.aplicada_en FROM vec_bolsa_llamamientos.reposicion_orden_bolsa ro WHERE ro.bolsa_ref=c.bolsa_ref AND ro.participacion_ref=e.participacion_ref AND ro.aplicada_en<=p_en ORDER BY ro.aplicada_en DESC LIMIT 1) r ON true
   WHERE c.bolsa_ref=p_bolsa_ref
 ), elegibles AS (
  SELECT b.participacion_ref,row_number() OVER(ORDER BY
    CASE WHEN b.penalizada THEN 1 ELSE 0 END,
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN 1 ELSE 0 END,
    CASE WHEN p.reposicion='fin_lista' AND b.repuesta_en IS NOT NULL THEN NULL ELSE b.orden_acta END,
    b.repuesta_en,b.orden_acta,b.participacion_ref)::bigint AS orden_vigente
   FROM base b CROSS JOIN politica p WHERE b.ocupa_turno
     AND NOT EXISTS(SELECT 1 FROM base anterior
       WHERE anterior.orden_acta<b.orden_acta
         AND anterior.motivo_relacion='pendiente_comprobacion_rrhh')
 )
 SELECT p.politica_ref,p.version,p.criterio,p.tipo_lista,p.reposicion,p.provisional,p.rotulo,p.actor,p.vigente_desde,
        b.participacion_ref,b.orden_acta,e.orden_vigente,
        CASE WHEN b.cese_pendiente AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'no_disponible'
             WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'disponible_desde' WHEN b.trabajo_cesado THEN 'disponible'
             ELSE b.situacion END AS situacion,
        CASE WHEN b.cese_pendiente AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'cese_pendiente'
             WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde') THEN 'restriccion_cese'
             WHEN b.respuesta_pendiente AND b.turno_sin_respuesta THEN 'respuesta_portal_pendiente'
             WHEN b.motivo_relacion<>'libre' AND NOT b.ocupa_turno
                  AND (b.situacion IN ('disponible','trabajando')
                       OR (b.situacion='disponible_desde' AND b.fecha_disponible<=p_en))
             THEN b.motivo_relacion
             WHEN b.ocupa_turno AND e.orden_vigente IS NULL THEN 'pendiente_comprobacion_previa'
             WHEN b.trabajo_cesado AND b.ocupa_turno THEN 'retorno_tras_cese'
             WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'
             WHEN NOT b.ocupa_turno AND b.situacion='trabajando' THEN 'trabajando'
             WHEN NOT b.ocupa_turno THEN 'sin_turno'
             WHEN b.penalizada THEN 'sancion_al_final'
             WHEN b.repuesta_en IS NOT NULL AND e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'reposicion_tras_contrato'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta
                  AND EXISTS (SELECT 1 FROM base o WHERE o.penalizada AND o.ocupa_turno AND o.orden_acta < b.orden_acta) THEN 'adelanta_por_sancion'
             WHEN e.orden_vigente IS DISTINCT FROM b.orden_acta THEN 'pausa'
             ELSE 'orden_acta' END,
        b.eventos_relacion
   FROM base b CROSS JOIN politica p LEFT JOIN elegibles e USING(participacion_ref)
  ORDER BY e.orden_vigente NULLS LAST,b.orden_acta,b.participacion_ref
 $function$;


REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(text,text,text,timestamptz) FROM PUBLIC;

-- Lectura con la misma acta y el orden contextual de esta modalidad.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(p_categoria text, p_modalidad text, p_causa text)
 RETURNS TABLE(acta_ref text, categoria_ref text, bolsa_ref text, version_bolsa bigint, huella_bolsa_sha256 text, bolsa_canonica bytea, instantanea_ref text, version_instantanea bigint, huella_instantanea_sha256 text, instantanea_canonica bytea, politica_ref text, version_politica bigint, politica_canonica bytea, politica_provisional boolean, corte timestamp with time zone, bolsa_vigente_hasta timestamp with time zone, politica_vigente_hasta timestamp with time zone, entradas jsonb, criterios_ofrecimiento jsonb)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 v_corte timestamptz(6):=date_trunc('microseconds',statement_timestamp());
 v_const record;
 v_politica vec_bolsa_llamamientos.politica_orden_bolsa%ROWTYPE;
 v_entradas jsonb;
 v_total bigint;
 v_orden bigint;
 v_sin_vinculo bigint;
 v_total_posiciones bigint;
 v_politica_divergente boolean;
 v_oferta record;
 v_criterios jsonb;
BEGIN
 IF p_categoria IS NULL OR octet_length(p_categoria) NOT BETWEEN 3 AND 512
    OR p_categoria<>btrim(p_categoria)
    OR p_modalidad IS NULL OR p_modalidad !~ '^[a-z][a-z0-9._-]{1,79}$'
    OR p_causa IS NULL OR (p_causa<>'' AND p_causa !~ '^[a-z][a-z0-9._-]{1,79}$') THEN
  RAISE EXCEPTION 'B110: categoría inválida' USING ERRCODE='22023'; END IF;
 SELECT count(*) INTO v_total
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 WHERE c.categoria_ref=p_categoria AND b.estado='vigente'
   AND b.vigente_desde<=v_corte AND (b.vigente_hasta IS NULL OR v_corte<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF v_total<>1 THEN
  RAISE EXCEPTION 'B110: constitución vigente ausente o ambigua' USING ERRCODE='P1101'; END IF;
 SELECT c.acta_ref,c.categoria_ref,c.bolsa_ref,c.version_bolsa,c.huella_bolsa_sha256,
        b.bolsa_canonica,c.instantanea_ref,c.version_instantanea,
        c.huella_instantanea_sha256,i.instantanea_canonica,i.total_participaciones,
        b.vigente_hasta AS bolsa_vigente_hasta
 INTO STRICT v_const
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
   ON i.instantanea_ref=c.instantanea_ref AND i.version=c.version_instantanea
  AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
 WHERE c.categoria_ref=p_categoria AND b.estado='vigente'
   AND b.vigente_desde<=v_corte AND (b.vigente_hasta IS NULL OR v_corte<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF encode(sha256(v_const.bolsa_canonica),'hex')<>v_const.huella_bolsa_sha256
    OR encode(sha256(v_const.instantanea_canonica),'hex')<>v_const.huella_instantanea_sha256
 THEN RAISE EXCEPTION 'B110: canónico divergente' USING ERRCODE='P1101'; END IF;
 SELECT * INTO v_politica FROM vec_bolsa_llamamientos.politica_orden_bolsa p
 WHERE p.bolsa_ref=v_const.bolsa_ref AND p.vigente_desde<=v_corte
   AND (p.vigente_hasta IS NULL OR v_corte<p.vigente_hasta)
 ORDER BY p.version DESC LIMIT 1;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'B110: política de orden publicada ausente' USING ERRCODE='P1101'; END IF;
 SELECT po.version,po.huella_sha256,po.politica->'criterios_ofrecimiento' AS criterios
 INTO v_oferta FROM vec_bolsa_llamamientos.politica_ofertas_version po
 WHERE po.bolsa_ref=v_const.bolsa_ref AND po.publicada_en<=v_corte
 ORDER BY po.version DESC LIMIT 1;
 IF FOUND AND v_oferta.criterios IS NOT NULL AND v_oferta.criterios<>'{}'::jsonb THEN
  IF vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(v_oferta.criterios) IS NOT TRUE
  THEN RAISE EXCEPTION 'B118: criterios publicados inválidos' USING ERRCODE='55000'; END IF;
  v_criterios:=jsonb_build_object('version_politica_ofertas',v_oferta.version,
   'huella_politica_ofertas_sha256',v_oferta.huella_sha256,
   'modalidad_ofrecida',p_modalidad,'causa_ofrecida',p_causa,
   'configuracion',v_oferta.criterios);
 END IF;
 WITH posiciones AS MATERIALIZED (
   SELECT o.* FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(v_const.bolsa_ref,p_modalidad,p_causa,v_corte) o
 ), control_posiciones AS (
   SELECT count(*) AS total,
     coalesce(bool_or(o.politica_ref IS DISTINCT FROM v_politica.politica_ref
       OR o.version_politica IS DISTINCT FROM v_politica.version
       OR o.provisional IS DISTINCT FROM v_politica.provisional),false) AS divergente
   FROM posiciones o
 )
 SELECT count(*),count(p.participacion_ref),
        count(*) FILTER(WHERE p.orden_vigente IS NOT NULL AND v.candidato_ref IS NULL),
        jsonb_agg(jsonb_build_object('orden_acta',e.orden,'orden_vigente',p.orden_vigente,
          'participacion_ref',e.participacion_ref,'candidato_ref',v.candidato_ref,
          'situacion',p.situacion,'razon',p.razon,
          'eventos_relacion',coalesce(p.eventos_relacion,'[]'::jsonb)) ORDER BY e.orden),
        max(control_posiciones.total),bool_or(control_posiciones.divergente)
 INTO v_total,v_orden,v_sin_vinculo,v_entradas,
      v_total_posiciones,v_politica_divergente
 FROM vec_bolsa_llamamientos.constitucion_entrada e
 CROSS JOIN control_posiciones
 LEFT JOIN posiciones p ON p.participacion_ref=e.participacion_ref AND p.orden_acta=e.orden
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato v
   ON v.participacion_ref=e.participacion_ref AND v.acta_ref=v_const.acta_ref
 WHERE e.instantanea_ref=v_const.instantanea_ref
   AND e.version_instantanea=v_const.version_instantanea;
 IF v_total<>v_const.total_participaciones OR v_orden<>v_total OR v_sin_vinculo<>0
    OR v_entradas IS NULL
    OR v_total_posiciones IS DISTINCT FROM v_total
    OR v_politica_divergente IS NOT FALSE
 THEN RAISE EXCEPTION 'B110: orden o vínculo incompleto' USING ERRCODE='P1101'; END IF;
 RETURN QUERY SELECT v_const.acta_ref,v_const.categoria_ref,v_const.bolsa_ref,v_const.version_bolsa,
   v_const.huella_bolsa_sha256,v_const.bolsa_canonica,v_const.instantanea_ref,
   v_const.version_instantanea,v_const.huella_instantanea_sha256,v_const.instantanea_canonica,
   v_politica.politica_ref,v_politica.version,convert_to(to_jsonb(v_politica)::text,'UTF8'),
   v_politica.provisional,v_corte,v_const.bolsa_vigente_hasta,v_politica.vigente_hasta,v_entradas,v_criterios;
END $function$;


REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text) TO vec_autorizacion_atestada_v3_propietario;

-- La publicación V3 y su replay histórico se conservan; se valida la extensión.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.publicar_politica_ofertas_v1(p_bolsa text, p_version_esperada bigint, p_politica jsonb, p_actor text, p_clave text, p_recibo text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(politica jsonb, reutilizada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE consumo record; d jsonb; previa record; v_version bigint; v_huella text; v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_bolsa IS NULL OR p_bolsa !~ '^bolsa:[A-Za-z0-9:_-]{1,250}$'
    OR p_version_esperada IS NULL OR p_version_esperada<0 OR p_version_esperada>2147483646
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9:_-]+$' OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_recibo IS NULL OR p_recibo !~ '^recibo:politica-ofertas:[0-9a-f]{64}$'
    OR jsonb_typeof(p_politica) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica)) NOT BETWEEN 3 AND 5
    OR NOT (p_politica ?& ARRAY['plazo','adjudicacion','no_cubierta'])
    OR EXISTS(SELECT 1 FROM jsonb_object_keys(p_politica) k
       WHERE k NOT IN ('plazo','adjudicacion','no_cubierta','plazas','criterios_ofrecimiento'))
    OR (p_politica ? 'criterios_ofrecimiento' AND
       vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
        p_politica->'criterios_ofrecimiento') IS NOT TRUE)
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
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo')) NOT IN (4,5)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica->'plazo'))=5
        AND (p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion' AND p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'publicacion'))
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
    OR (SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion')) NOT IN (2,3)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_politica->'adjudicacion'))=3
        AND p_politica#>>'{adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa')
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
 -- Las versiones anteriores pueden repetirse, pero toda política nueva declara
 -- en su propia versión que el plazo empieza en la notificación.
 IF (p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'notificacion' AND p_politica#>>'{plazo,inicio}' IS DISTINCT FROM 'publicacion') THEN
  RAISE EXCEPTION 'B71: inicio de plazo no configurado' USING ERRCODE='22023'; END IF;
 -- El replay histórico ya devolvió arriba el contenido y recibo originales.
 -- Toda versión nueva requiere la aceptación telemática previa.
 IF p_politica#>>'{adjudicacion,confirmacion}' IS DISTINCT FROM 'aceptacion_previa' THEN
  RAISE EXCEPTION 'B70: confirmación telemática requerida para la versión nueva' USING ERRCODE='22023';
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
END $function$;



-- Reserva original con revalidación contextual antes del consumo V3.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_ct_v1(p_llamamiento text, p_recibo text, p_bolsa text, p_actor text, p_clave text, p_participaciones jsonb, p_configuracion jsonb, p_emitido timestamp with time zone, p_huella_finalizacion bytea, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea, p_origen jsonb)
 RETURNS TABLE(emision jsonb, reutilizada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE
 previo vec_bolsa_llamamientos.llamamiento_emitido%ROWTYPE;
 origen_previo vec_bolsa_llamamientos.origen_llamamiento_ct_v1%ROWTYPE;
 consumo record;
 d jsonb;
 v_huella text;
 v_origen_huella text;
 v_evento text;
 v_canon_politica bytea;
 v_confirmacion jsonb;
 v_cuantos bigint;
 v_lectura jsonb;
 v_entrada jsonb;
 v_ct record;
 v_politica_oferta record;
 v_criterios jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR p_llamamiento !~ '^llamamiento:[0-9a-f]{64}$'
    OR p_recibo !~ '^recibo:llamamiento:[0-9a-f]{64}$'
    OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 8 AND 256
    OR p_emitido IS NULL OR octet_length(p_huella_finalizacion) IS DISTINCT FROM 32
    OR jsonb_typeof(p_participaciones)<>'array' OR jsonb_array_length(p_participaciones)<>1
    OR jsonb_typeof(p_participaciones->0)<>'string'
    OR jsonb_typeof(p_configuracion)<>'object' OR
       p_configuracion ?& ARRAY['referencia','descripcion','categoria','centro','modalidad',
       'fecha_inicio','plazo','plantilla_version','asunto','cuerpo'] IS NOT TRUE
    OR (SELECT count(*)<>10 OR bool_or(jsonb_typeof(value)<>'string'
      OR octet_length(value#>>'{}') NOT BETWEEN 2 AND 4000
      OR value#>>'{}'<>btrim(value#>>'{}')) FROM jsonb_each(p_configuracion))
    OR p_configuracion->>'plantilla_version' !~ '^bolsa-llamamiento-v[1-9][0-9]{0,3}$'
    OR jsonb_typeof(p_origen)<>'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_origen)) NOT IN (22,23)
    OR ((SELECT count(*) FROM jsonb_object_keys(p_origen))=23
        AND NOT (p_origen ? 'criterios_ofrecimiento'))
    OR p_origen ?& ARRAY['organizacion_ref','unidad_ref','expediente_ref','version_expediente','clave_ct',
      'necesidad_ref','acta_ref','categoria_ref','bolsa_ref','version_bolsa','huella_bolsa_sha256',
      'instantanea_ref','version_instantanea','huella_instantanea_sha256','politica_ref',
      'version_politica','huella_politica_sha256','participacion_ref','candidato_ref','orden_seleccionado','lectura','version_token'] IS NOT TRUE
    OR p_origen->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR p_origen->>'participacion_ref' IS DISTINCT FROM p_participaciones->>0
    OR p_origen->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_origen->>'clave_ct' IS NULL OR p_origen->>'expediente_ref' IS NULL
    OR p_origen->>'organizacion_ref' IS NULL OR p_origen->>'necesidad_ref' IS NULL
    OR p_origen->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'
    OR p_origen->>'version_token' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 THEN RAISE EXCEPTION 'B110: reserva CT inválida' USING ERRCODE='22023'; END IF;
 v_lectura:=p_origen->'lectura';
 v_criterios:=p_origen->'criterios_ofrecimiento';
 IF jsonb_typeof(v_lectura)<>'object'
    OR v_lectura->>'acta_ref' IS DISTINCT FROM p_origen->>'acta_ref'
    OR v_lectura->>'categoria_ref' IS DISTINCT FROM p_origen->>'categoria_ref'
    OR v_lectura->>'bolsa_ref' IS DISTINCT FROM p_bolsa
    OR v_lectura->>'version_bolsa' IS DISTINCT FROM p_origen->>'version_bolsa'
    OR v_lectura->>'huella_bolsa_sha256' IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR v_lectura->>'instantanea_ref' IS DISTINCT FROM p_origen->>'instantanea_ref'
    OR v_lectura->>'version_instantanea' IS DISTINCT FROM p_origen->>'version_instantanea'
    OR v_lectura->>'huella_instantanea_sha256' IS DISTINCT FROM p_origen->>'huella_instantanea_sha256'
    OR v_lectura->>'politica_ref' IS DISTINCT FROM p_origen->>'politica_ref'
    OR v_lectura->>'version_politica' IS DISTINCT FROM p_origen->>'version_politica'
    OR v_lectura->>'huella_politica_sha256' IS DISTINCT FROM p_origen->>'huella_politica_sha256'
    OR jsonb_typeof(v_lectura->'politica_provisional') IS DISTINCT FROM 'boolean'
    OR jsonb_typeof(v_lectura->'entradas')<>'array'
    OR v_lectura->'criterios_ofrecimiento' IS DISTINCT FROM v_criterios
    OR (v_criterios IS NOT NULL AND (jsonb_typeof(v_criterios) IS DISTINCT FROM 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(v_criterios))<>5
       OR v_criterios ?& ARRAY['version_politica_ofertas',
          'huella_politica_ofertas_sha256','modalidad_ofrecida',
          'causa_ofrecida','configuracion'] IS NOT TRUE
       OR v_criterios->>'huella_politica_ofertas_sha256' !~ '^[0-9a-f]{64}$'
       OR jsonb_typeof(v_criterios->'version_politica_ofertas') IS DISTINCT FROM 'number'
       OR v_criterios->>'version_politica_ofertas' !~ '^[1-9][0-9]{0,17}$'
       OR jsonb_typeof(v_criterios->'causa_ofrecida') IS DISTINCT FROM 'string'
       OR (v_criterios->>'causa_ofrecida'<>''
         AND v_criterios->>'causa_ofrecida' !~ '^[a-z][a-z0-9._-]{1,79}$')
       OR vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(
          v_criterios->'configuracion') IS NOT TRUE))
    OR (v_lectura->>'corte')::timestamptz>p_emitido
    OR encode(sha256(decode(v_lectura->>'bolsa_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_bolsa_sha256'
    OR encode(sha256(decode(v_lectura->>'instantanea_canonica','base64')),'hex')
       IS DISTINCT FROM p_origen->>'huella_instantanea_sha256'
 THEN RAISE EXCEPTION 'B110: lectura CT incompatible' USING ERRCODE='22023'; END IF;
 SELECT x.value INTO v_entrada FROM jsonb_array_elements(v_lectura->'entradas') x(value)
 WHERE x.value->>'participacion_ref'=p_origen->>'participacion_ref'
   AND x.value->>'candidato_ref'=p_origen->>'candidato_ref'
   AND x.value->>'orden_acta'=p_origen->>'orden_seleccionado'
   AND x.value->>'elegible'='true' AND x.value->>'orden_vigente'='1';
 IF NOT FOUND THEN RAISE EXCEPTION 'B110: selección no pertenece a lectura' USING ERRCODE='22023'; END IF;
 -- La validación de la lista y del orden se hace con la política efectiva B6.
 v_huella:=encode(sha256(convert_to(p_bolsa||chr(31)||p_participaciones::text||chr(31)||p_configuracion::text,'UTF8')),'hex');
 v_origen_huella:=encode(sha256(convert_to(p_origen::text,'UTF8')),'hex');
 v_evento:='evento:bolsa:ct:'||encode(sha256(convert_to(p_origen->>'expediente_ref'||chr(31)||p_origen->>'clave_ct','UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:emision:'||p_bolsa||':'||p_clave,0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.llamamiento_emitido
  WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave FOR UPDATE;
 IF FOUND THEN
  SELECT * INTO origen_previo FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1
   WHERE expediente_ref=p_origen->>'expediente_ref' AND clave_ct=p_origen->>'clave_ct';
  IF NOT FOUND OR previo.actor_ref<>p_actor OR previo.huella_comando_sha256<>v_huella
     OR previo.llamamiento_ref<>p_llamamiento OR previo.recibo_ref<>p_recibo
     OR origen_previo.origen_huella_sha256<>v_origen_huella
     OR origen_previo.llamamiento_ref<>previo.llamamiento_ref THEN
   RAISE EXCEPTION 'B110: clave CT divergente' USING ERRCODE='VBE01'; END IF;
  RETURN QUERY SELECT vec_bolsa_llamamientos.recuperar_llamamiento_emitido_v1(p_bolsa,p_clave)
    || jsonb_build_object('origen_ct',vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
      p_origen->>'expediente_ref',p_origen->>'clave_ct')),true;
  RETURN;
 END IF;
 -- Solo una reserva nueva revalida modalidad y versión del expediente. El
 -- replay anterior ya conservó su origen y vuelve sin reinterpretar hechos.
 SELECT * INTO STRICT v_ct FROM vec_contratacion_temporal.leer_modalidad_origen_bolsa_ct_v1(
  p_origen->>'expediente_ref',p_origen->>'categoria_ref',
  p_origen->>'organizacion_ref',(p_origen->>'version_expediente')::bigint);
 IF v_ct.modalidad_clave IS DISTINCT FROM p_configuracion->>'modalidad'
 THEN RAISE EXCEPTION 'B118: modalidad del expediente divergente' USING ERRCODE='23503'; END IF;
 IF v_criterios IS NOT NULL AND v_criterios->>'causa_ofrecida' IS DISTINCT FROM v_ct.causa_clave
 THEN RAISE EXCEPTION 'B118: causa del expediente divergente' USING ERRCODE='23503'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
  'vec_bolsa_llamamientos:politica-ofertas:'||p_bolsa,0));
 SELECT po.version,po.huella_sha256,po.politica->'criterios_ofrecimiento' AS criterios
 INTO v_politica_oferta FROM vec_bolsa_llamamientos.politica_ofertas_version po
 WHERE po.bolsa_ref=p_bolsa ORDER BY po.version DESC LIMIT 1;
 IF FOUND AND v_politica_oferta.criterios IS NOT NULL
    AND v_politica_oferta.criterios<>'{}'::jsonb THEN
  IF v_criterios IS NULL
   OR v_criterios->>'version_politica_ofertas' IS DISTINCT FROM v_politica_oferta.version::text
   OR v_criterios->>'huella_politica_ofertas_sha256' IS DISTINCT FROM v_politica_oferta.huella_sha256
   OR v_criterios->'configuracion' IS DISTINCT FROM v_politica_oferta.criterios
   OR v_criterios->>'modalidad_ofrecida' IS DISTINCT FROM v_ct.modalidad_clave
   OR v_criterios->>'causa_ofrecida' IS DISTINCT FROM v_ct.causa_clave
  THEN RAISE EXCEPTION 'B118: criterio de oferta divergente' USING ERRCODE='23503'; END IF;
 ELSIF v_criterios IS NOT NULL THEN
  RAISE EXCEPTION 'B118: criterio de oferta ausente' USING ERRCODE='23503';
 END IF;
 SELECT count(*) INTO v_cuantos
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=c.bolsa_ref
   AND b.version=c.version_bolsa AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i ON i.instantanea_ref=c.instantanea_ref
   AND i.version=c.version_instantanea AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
 JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref
   AND e.version_instantanea=c.version_instantanea
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=e.participacion_ref
   AND vc.acta_ref=c.acta_ref AND vc.instantanea_ref=c.instantanea_ref
   AND vc.version_instantanea=c.version_instantanea
 JOIN vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(p_bolsa,v_ct.modalidad_clave,v_ct.causa_clave,p_emitido) o
   ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 JOIN vec_bolsa_llamamientos.politica_orden_bolsa pol
   ON pol.politica_ref=o.politica_ref AND pol.version=o.version_politica
 WHERE c.acta_ref=p_origen->>'acta_ref' AND c.categoria_ref=p_origen->>'categoria_ref'
   AND c.bolsa_ref=p_bolsa AND c.version_bolsa=(p_origen->>'version_bolsa')::bigint
   AND c.huella_bolsa_sha256=p_origen->>'huella_bolsa_sha256'
   AND c.instantanea_ref=p_origen->>'instantanea_ref'
   AND c.version_instantanea=(p_origen->>'version_instantanea')::bigint
   AND c.huella_instantanea_sha256=p_origen->>'huella_instantanea_sha256'
   AND e.participacion_ref=p_origen->>'participacion_ref'
   AND e.orden=(p_origen->>'orden_seleccionado')::bigint
   AND vc.candidato_ref=p_origen->>'candidato_ref'
   AND o.orden_vigente=1
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND pol.provisional=(v_lectura->>'politica_provisional')::boolean
   AND encode(sha256(convert_to(to_jsonb(pol)::text,'UTF8')),'hex')=p_origen->>'huella_politica_sha256'
   AND b.estado='vigente' AND b.vigente_desde<=p_emitido
   AND (b.vigente_hasta IS NULL OR p_emitido<b.vigente_hasta)
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
     WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
       AND s.huella_sustituida_sha256=c.huella_bolsa_sha256);
 IF v_cuantos<>1 THEN RAISE EXCEPTION 'B110: origen CT no vigente o divergente' USING ERRCODE='23503'; END IF;
 -- La decisión completa que se conservará para replay se coteja con el
 -- orden B6 al corte original; ninguna entrada puede sustituirse en tránsito.
 SELECT count(*) INTO v_cuantos
 FROM jsonb_array_elements(v_lectura->'entradas') WITH ORDINALITY x(valor,n)
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
   ON e.instantanea_ref=p_origen->>'instantanea_ref'
  AND e.version_instantanea=(p_origen->>'version_instantanea')::bigint
  AND e.orden=x.n AND e.participacion_ref=x.valor->>'participacion_ref'
 JOIN vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(
   p_bolsa,v_ct.modalidad_clave,v_ct.causa_clave,(v_lectura->>'corte')::timestamptz) o
   ON o.participacion_ref=e.participacion_ref AND o.orden_acta=e.orden
 LEFT JOIN vec_bolsa_llamamientos.vinculo_candidato vc
   ON vc.participacion_ref=e.participacion_ref AND vc.acta_ref=p_origen->>'acta_ref'
 WHERE x.valor->>'orden_acta'=e.orden::text
   AND x.valor->>'orden_vigente' IS NOT DISTINCT FROM o.orden_vigente::text
   AND x.valor->>'situacion' IS NOT DISTINCT FROM o.situacion
   AND x.valor->>'razon' IS NOT DISTINCT FROM o.razon
   AND x.valor->'eventos_relacion' IS NOT DISTINCT FROM o.eventos_relacion
   AND x.valor->>'candidato_ref' IS NOT DISTINCT FROM coalesce(vc.candidato_ref,'')
   AND x.valor->>'elegible'=(o.orden_vigente IS NOT NULL)::text
   AND o.politica_ref=p_origen->>'politica_ref'
   AND o.version_politica=(p_origen->>'version_politica')::bigint
   AND o.provisional=(v_lectura->>'politica_provisional')::boolean;
 IF v_cuantos<>jsonb_array_length(v_lectura->'entradas') THEN
  RAISE EXCEPTION 'B110: orden completo divergente' USING ERRCODE='23503'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B110: decisión inválida' USING ERRCODE='42501'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM p_actor
    OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida' THEN
  RAISE EXCEPTION 'B110: emisión no autorizada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(
  llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,
  participaciones,configuracion,huella_comando_sha256,huella_finalizacion,
  estado,emitido_en,decision_ref)
 VALUES(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,
  p_configuracion,v_huella,p_huella_finalizacion,'emision_reservada',p_emitido,consumo.decision_ref);
 INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_v1(
  organizacion_ref,unidad_ref,expediente_ref,version_expediente,clave_ct,necesidad_ref,
  acta_ref,categoria_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,
  version_instantanea,huella_instantanea_sha256,politica_ref,version_politica,
  huella_politica_sha256,participacion_ref,candidato_ref,orden_seleccionado,version_token,
  llamamiento_ref,recibo_ref,decision_ref,auditoria_ref,evento_ref,emitido_en,
  origen_canonico,origen_huella_sha256)
 VALUES(p_origen->>'organizacion_ref',p_origen->>'unidad_ref',p_origen->>'expediente_ref',
  (p_origen->>'version_expediente')::bigint,p_origen->>'clave_ct',p_origen->>'necesidad_ref',
  p_origen->>'acta_ref',p_origen->>'categoria_ref',p_bolsa,(p_origen->>'version_bolsa')::bigint,
  p_origen->>'huella_bolsa_sha256',p_origen->>'instantanea_ref',
  (p_origen->>'version_instantanea')::bigint,p_origen->>'huella_instantanea_sha256',
  p_origen->>'politica_ref',(p_origen->>'version_politica')::bigint,
  p_origen->>'huella_politica_sha256',p_origen->>'participacion_ref',
  p_origen->>'candidato_ref',(p_origen->>'orden_seleccionado')::bigint,p_origen->>'version_token',
  p_llamamiento,p_recibo,consumo.decision_ref,consumo.auditoria_ref,v_evento,
  p_emitido,p_origen,v_origen_huella);
 INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1
   (evento_ref,llamamiento_ref,tipo,estado,creada_en)
 VALUES(v_evento,p_llamamiento,'bolsa.llamamiento.ct.origen_confirmado','pendiente',p_emitido);
 v_confirmacion:=vec_bolsa_llamamientos.recuperar_origen_llamamiento_ct_v1(
  p_origen->>'expediente_ref',p_origen->>'clave_ct') || jsonb_build_object('reutilizada',false);
 RETURN QUERY SELECT jsonb_build_object('llamamiento_ref',p_llamamiento,'recibo_ref',p_recibo,
  'bolsa_ref',p_bolsa,'estado','emision_reservada_resultado_pendiente',
  'participaciones',p_participaciones,'configuracion',p_configuracion,
  'emitido_en',p_emitido,'contactos','[]'::jsonb,'origen_ct',v_confirmacion),false;
END $function$;


-- Lectura RRHH de exclusiones del origen ya reservado, por posición de acta.
-- El acuse nominal nuevo de AD276 se asienta en esta misma transacción.
CREATE FUNCTION vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1(
 p_bolsa_ref text,p_llamamiento_ref text,p_cursor_orden bigint,p_limite int,
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_lectura jsonb;v_entradas jsonb;v_exclusiones jsonb:='[]'::jsonb;
        v_total bigint;v_n int:=0;v_hay_mas boolean:=false;v_siguiente bigint;
        v_hechos jsonb;v_hechos_n int;
        x record;a record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user=current_user
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
 OR p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 512
 OR p_llamamiento_ref IS NULL OR p_llamamiento_ref !~ '^llamamiento:[0-9a-f]{64}$'
 OR p_cursor_orden IS NULL OR p_cursor_orden<0
 OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 1000
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'accion' IS DISTINCT FROM 'bolsa.rrhh.exclusiones_llamamiento.consultar'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM p_llamamiento_ref
 OR p_orden->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'seguimiento_contacto_rrhh'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM 'lectura_exclusiones_llamamiento_correcta'
 THEN RAISE EXCEPTION 'B118: exclusiones no autorizadas' USING ERRCODE='42501'; END IF;
 SELECT o.origen_canonico->'lectura' INTO v_lectura
 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o
 JOIN vec_bolsa_llamamientos.llamamiento_emitido e
   ON e.llamamiento_ref=o.llamamiento_ref AND e.bolsa_ref=o.bolsa_ref
 WHERE o.llamamiento_ref=p_llamamiento_ref AND o.bolsa_ref=p_bolsa_ref;
 v_entradas:=v_lectura->'entradas';
 IF jsonb_typeof(v_entradas) IS DISTINCT FROM 'array'
 OR jsonb_array_length(v_entradas)>100000
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(v_entradas) j(valor)
  WHERE jsonb_typeof(j.valor) IS DISTINCT FROM 'object'
   OR j.valor->>'orden_acta' !~ '^[1-9][0-9]{0,17}$'
   OR j.valor->>'participacion_ref' IS NULL
   OR jsonb_typeof(j.valor->'elegible') IS DISTINCT FROM 'boolean')
 THEN RAISE EXCEPTION 'B118: origen histórico no disponible' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_total FROM jsonb_array_elements(v_entradas) j(valor)
 WHERE j.valor->>'elegible'='false';
 FOR x IN SELECT (j.valor->>'orden_acta')::bigint AS orden_acta,
                 j.valor->>'participacion_ref' AS participacion_ref,
                 j.valor->>'situacion' AS situacion,
                 j.valor->>'razon' AS razon,
                 coalesce(j.valor->'eventos_relacion','[]'::jsonb) AS eventos_relacion
   FROM jsonb_array_elements(v_entradas) j(valor)
  WHERE j.valor->>'elegible'='false'
    AND (j.valor->>'orden_acta')::bigint>p_cursor_orden
  ORDER BY (j.valor->>'orden_acta')::bigint LIMIT p_limite+1
 LOOP
  IF v_n=p_limite THEN v_hay_mas:=true;EXIT; END IF;
  IF jsonb_typeof(x.eventos_relacion) IS DISTINCT FROM 'array'
     OR jsonb_array_length(x.eventos_relacion)>100
     OR EXISTS(SELECT 1 FROM jsonb_array_elements(x.eventos_relacion) e(valor)
       WHERE jsonb_typeof(e.valor) IS DISTINCT FROM 'string'
          OR e.valor#>>'{}' !~ '^evento:ct:contrato-bolsa:[0-9a-f]{64}$')
     OR (SELECT count(*) FROM jsonb_array_elements_text(x.eventos_relacion))
        <> (SELECT count(DISTINCT e.ref) FROM jsonb_array_elements_text(x.eventos_relacion) e(ref))
  THEN RAISE EXCEPTION 'B118: hechos históricos no disponibles' USING ERRCODE='42501'; END IF;
  SELECT count(*),coalesce(jsonb_agg(jsonb_build_object(
    'evento_ref',cp.evento_ref,'tipo',cp.tipo,
    'ocurrido_en',to_char(cp.ocurrido_en AT TIME ZONE 'UTC',
      'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) ORDER BY e.n),'[]'::jsonb)
  INTO v_hechos_n,v_hechos
  FROM jsonb_array_elements_text(x.eventos_relacion) WITH ORDINALITY e(ref,n)
  JOIN vec_bolsa_llamamientos.contrato_participacion cp ON cp.evento_ref=e.ref
  LEFT JOIN vec_bolsa_llamamientos.origen_llamamiento_ct_v1 origen
    ON origen.llamamiento_ref=cp.llamamiento_ref
   AND origen.expediente_ref=cp.expediente_ref
   AND origen.organizacion_ref=cp.organizacion_ref
   AND origen.categoria_ref=cp.categoria_ref
  WHERE cp.tipo IN ('incorporacion','cese')
    AND cp.ocurrido_en<=(v_lectura->>'corte')::timestamptz
    AND cp.recibido_en<=(v_lectura->>'corte')::timestamptz
    AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena q
      WHERE q.evento_ref=cp.evento_ref
        AND q.recibido_en<=(v_lectura->>'corte')::timestamptz)
    AND EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato vc
      JOIN vec_bolsa_llamamientos.vinculo_candidato vc2
        ON vc2.candidato_ref=vc.candidato_ref
      WHERE vc.participacion_ref=x.participacion_ref
        AND vc2.participacion_ref=coalesce(cp.participacion_ref,origen.participacion_ref));
  IF v_hechos_n<>jsonb_array_length(x.eventos_relacion) THEN
   RAISE EXCEPTION 'B118: hechos históricos divergentes' USING ERRCODE='42501'; END IF;
  v_n:=v_n+1;v_siguiente:=x.orden_acta;
  v_exclusiones:=v_exclusiones||jsonb_build_array(jsonb_build_object(
   'participacion_ref',x.participacion_ref,'orden_acta',x.orden_acta,
   'situacion_en_origen',x.situacion,'motivo_clave',x.razon,
   'eventos_relacion',x.eventos_relacion,'hechos_relacion',v_hechos));
 END LOOP;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_lectura_exclusiones_llamamiento_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF a.auditoria_ref IS NULL OR a.secuencia IS NULL
 OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
 OR a.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
 OR a.registrada_en IS NULL
 THEN RAISE EXCEPTION 'B118: acuse de lectura incompatible' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('esquema','vec.bolsa.rrhh.exclusiones-ofrecimiento.v1',
  'bolsa_ref',p_bolsa_ref,'llamamiento_ref',p_llamamiento_ref,
  'criterios_ofrecimiento',v_lectura->'criterios_ofrecimiento',
  'exclusiones',v_exclusiones,'total_exclusiones',v_total,'hay_mas',v_hay_mas,
  'siguiente_orden_acta',CASE WHEN v_hay_mas THEN v_siguiente ELSE NULL END,
  'auditoria_ref',a.auditoria_ref,'correlacion_ref',a.correlacion_ref,
  'consultada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1(
 text,text,bigint,int,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1(
 text,text,bigint,int,bytea,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;
DO $post$ DECLARE firmas regprocedure[]:=ARRAY[
 'vec_bolsa_llamamientos.validar_criterios_ofrecimiento_v1(jsonb)'::regprocedure,
 'vec_bolsa_llamamientos.motivo_relacion_oferta_v1(text,text,boolean,text,text,timestamptz,jsonb)'::regprocedure,
 'vec_bolsa_llamamientos.eventos_relacion_oferta_v1(text,timestamptz)'::regprocedure,
 'vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(text,text,text,timestamptz)'::regprocedure,
 'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(text,text,text)'::regprocedure,
 'vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1(text,text,bigint,int,bytea,bytea,jsonb)'::regprocedure];
BEGIN
 IF (SELECT count(*)=6 AND bool_and(p.prosecdef
       AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole
       AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[])
     FROM pg_proc p WHERE p.oid=ANY(firmas)) IS NOT TRUE
 OR EXISTS(SELECT 1 FROM pg_proc p,
      aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
      WHERE p.oid=ANY(firmas) AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
 THEN RAISE EXCEPTION 'B118: funciones nuevas sin cierre de permisos'
  USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
