\set ON_ERROR_STOP on
-- B10: captura privada e inmutable de la foto Bolsa ligada a una fase B49.
-- El documento enmascarado, los metadatos públicos y V2 proceden de un
-- proveedor gobernado externo. Aquí solo se congelan referencias internas,
-- orden y estado B6/B45; el material público se inserta antes de publicarlo.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000052',0));

DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.publicacion_cese_b10') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.captura_cese_b10') IS NOT NULL THEN
  RAISE EXCEPTION 'Bolsa 000052: dependencias incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.captura_cese_b10 (
 evento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.restriccion_cese_bolsa(evento_ref),
 fase text NOT NULL CHECK (fase IN ('cese','vencimiento')),
 origen_ref text NOT NULL,
 origen_posicion bigint NOT NULL CHECK (origen_posicion>=0),
 corte timestamptz(6) NOT NULL CHECK (isfinite(corte)),
 -- Lista plana: bolsa, participación, fila de acta, orden y estado efectivos.
 -- No contiene candidato, nombre, documento, contacto, relación ni causa.
 filas jsonb NOT NULL CHECK (jsonb_typeof(filas)='array' AND jsonb_array_length(filas) BETWEEN 1 AND 1000000),
 filas_sha256 text NOT NULL CHECK (filas_sha256=encode(sha256(convert_to(filas::text,'UTF8')),'hex')),
 capturada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(evento_ref,fase),
 UNIQUE(origen_ref,fase)
);
CREATE INDEX captura_cese_b10_cursor ON vec_bolsa_llamamientos.captura_cese_b10(origen_posicion,origen_ref,fase);

CREATE TABLE vec_bolsa_llamamientos.material_cese_b10 (
 evento_ref text NOT NULL,
 fase text NOT NULL,
 anterior_actualizada_en timestamptz(6),
 proyeccion_v2 bytea NOT NULL CHECK (octet_length(proyeccion_v2) BETWEEN 2 AND 268435456),
 manifiesto_v2 bytea NOT NULL CHECK (octet_length(manifiesto_v2) BETWEEN 2 AND 268435456),
 bolsas_v1 bytea NOT NULL CHECK (octet_length(bolsas_v1) BETWEEN 2 AND 67108864),
 material_sha256 text NOT NULL CHECK (material_sha256=encode(sha256(proyeccion_v2||manifiesto_v2||bolsas_v1),'hex')),
 guardado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(evento_ref,fase),
 FOREIGN KEY(evento_ref,fase) REFERENCES vec_bolsa_llamamientos.captura_cese_b10(evento_ref,fase),
 CHECK (anterior_actualizada_en IS NULL OR isfinite(anterior_actualizada_en))
);

DO $proteccion$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['captura_cese_b10','material_cese_b10'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY %I ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING (current_user=''vec_bolsa_llamamientos_propietario'') WITH CHECK (current_user=''vec_bolsa_llamamientos_propietario'')',t||'_solo_propietario',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM PUBLIC',t);
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t||'_inmutable',t);
 END LOOP;
END $proteccion$;

CREATE FUNCTION vec_bolsa_llamamientos.capturar_cese_b10_v1(
 p_origen_posicion bigint,p_origen_ref text,p_evento_ref text,p_bolsa_ref text,p_fase text)
RETURNS TABLE(corte timestamptz,filas jsonb,anterior_actualizada_en timestamptz,
 proyeccion_v2 bytea,manifiesto_v2 bytea,bolsas_v1 bytea)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_captura vec_bolsa_llamamientos.captura_cese_b10;
 v_material vec_bolsa_llamamientos.material_cese_b10;
 v_cese vec_bolsa_llamamientos.restriccion_cese_bolsa;
 v_pendiente record; v_bolsa_origen text; v_filas jsonb; v_corte timestamptz;
 v_conteo bigint; v_esperado bigint; v_duplicados bigint; v_incompletos bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_publicador_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'captura B10 denegada' USING ERRCODE='42501';
 END IF;
 IF current_setting('transaction_isolation') NOT IN ('repeatable read','serializable')
    OR current_setting('transaction_read_only')<>'off'
    OR p_origen_posicion IS NULL OR p_origen_posicion<0
    OR p_origen_ref IS NULL OR octet_length(p_origen_ref) NOT BETWEEN 1 AND 512
    OR p_evento_ref IS NULL OR p_evento_ref !~ '^evento:ct:contrato-bolsa:[a-f0-9]{64}$'
    OR p_bolsa_ref IS NULL OR octet_length(p_bolsa_ref) NOT BETWEEN 1 AND 512
    OR p_fase IS NULL OR p_fase NOT IN ('cese','vencimiento') THEN
  RAISE EXCEPTION 'captura B10 inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:captura-cese-b10',0));
 SELECT * INTO v_cese FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
  WHERE r.evento_ref=p_evento_ref AND r.origen_ref=p_origen_ref
    AND r.origen_posicion=p_origen_posicion;
 IF NOT FOUND THEN RAISE EXCEPTION 'evento B45 ausente' USING ERRCODE='23503'; END IF;
 SELECT cp.bolsa_ref INTO v_bolsa_origen
  FROM vec_bolsa_llamamientos.contrato_participacion cp
  JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=cp.participacion_ref
  WHERE cp.evento_ref=p_evento_ref AND cp.origen_ref=p_origen_ref
    AND cp.origen_posicion=p_origen_posicion AND cp.tipo='cese'
    AND cp.huella_sha256=v_cese.origen_huella_sha256
    AND cp.llamamiento_ref=v_cese.llamamiento_ref AND vc.candidato_ref=v_cese.candidato_ref
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena q
                    WHERE q.evento_ref=cp.evento_ref);
 IF NOT FOUND OR v_bolsa_origen IS DISTINCT FROM p_bolsa_ref THEN
  RAISE EXCEPTION 'vínculo B45/B49 divergente' USING ERRCODE='23503';
 END IF;
 SELECT * INTO v_captura FROM vec_bolsa_llamamientos.captura_cese_b10 c
  WHERE c.evento_ref=p_evento_ref AND c.fase=p_fase;
 IF FOUND THEN
  IF v_captura.origen_ref<>p_origen_ref OR v_captura.origen_posicion<>p_origen_posicion
     OR v_captura.filas_sha256<>encode(sha256(convert_to(v_captura.filas::text,'UTF8')),'hex') THEN
   RAISE EXCEPTION 'captura B10 divergente' USING ERRCODE='VBC10';
  END IF;
  SELECT * INTO v_material FROM vec_bolsa_llamamientos.material_cese_b10 m
   WHERE m.evento_ref=p_evento_ref AND m.fase=p_fase;
  RETURN QUERY SELECT v_captura.corte,v_captura.filas,v_material.anterior_actualizada_en,
    v_material.proyeccion_v2,v_material.manifiesto_v2,v_material.bolsas_v1;
  RETURN;
 END IF;
 SELECT * INTO v_pendiente FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF NOT FOUND OR v_pendiente.origen_posicion<>p_origen_posicion
    OR v_pendiente.origen_ref<>p_origen_ref OR v_pendiente.evento_ref<>p_evento_ref
    OR v_pendiente.bolsa_ref<>p_bolsa_ref OR v_pendiente.fase<>p_fase THEN
  RAISE EXCEPTION 'captura B10 fuera de orden' USING ERRCODE='55000';
 END IF;
 IF p_fase='vencimiento' AND NOT EXISTS (
   SELECT 1 FROM vec_bolsa_llamamientos.captura_cese_b10 c
   WHERE c.evento_ref=p_evento_ref AND c.fase='cese') THEN
  RAISE EXCEPTION 'vencimiento B10 sin cese capturado' USING ERRCODE='55000';
 END IF;
 v_corte:=date_trunc('microseconds',transaction_timestamp());
 IF v_cese.recibida_en>v_corte THEN RAISE EXCEPTION 'corte B10 anterior al cese' USING ERRCODE='55000'; END IF;

 WITH ultimas AS (
  SELECT DISTINCT ON (c.bolsa_ref) c.bolsa_ref,c.categoria_ref,c.acta_ref,
    c.instantanea_ref,c.version_instantanea,i.total_participaciones,
    b.vigente_desde,b.vigente_hasta,b.estado,c.confirmada_en,c.registrada_en
  FROM vec_bolsa_llamamientos.constitucion c
  JOIN vec_bolsa_llamamientos.bolsa_constituida b
    ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
   AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
  JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i
    ON i.instantanea_ref=c.instantanea_ref AND i.version=c.version_instantanea
   AND i.huella_instantanea_sha256=c.huella_instantanea_sha256
  WHERE c.confirmada_en<=v_corte
  ORDER BY c.bolsa_ref,c.confirmada_en DESC,c.registrada_en DESC
 ), activas AS (
  SELECT * FROM ultimas WHERE estado='vigente' AND vigente_desde<=v_corte
   AND (vigente_hasta IS NULL OR vigente_hasta>v_corte)
 ), base AS (
  SELECT a.bolsa_ref,a.categoria_ref,a.acta_ref,a.vigente_desde,a.vigente_hasta,
    a.total_participaciones,e.participacion_ref,e.fila_numero,e.orden AS orden_acta,
    o.orden_vigente,o.situacion,o.tipo_lista,sp.fecha_disponible,
    cs.en_restriccion,cs.trabajo_cesado,cs.disponible_desde,
    row_number() OVER (PARTITION BY a.bolsa_ref
      ORDER BY o.orden_vigente NULLS LAST,e.orden,e.participacion_ref) AS orden_publico
  FROM activas a
  JOIN vec_bolsa_llamamientos.constitucion_entrada e
    ON e.instantanea_ref=a.instantanea_ref AND e.version_instantanea=a.version_instantanea
  LEFT JOIN LATERAL vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(a.bolsa_ref,v_corte) o
    ON o.participacion_ref=e.participacion_ref
  LEFT JOIN LATERAL (SELECT s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s
    WHERE s.participacion_ref=e.participacion_ref AND s.desde<=v_corte
    ORDER BY s.desde DESC LIMIT 1) sp ON true
  LEFT JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_v1(e.participacion_ref,v_corte) cs ON true
 ), validacion AS (
  SELECT count(*) conteo,coalesce(sum(total_participaciones) FILTER (WHERE orden_acta=1),0) esperado,
    count(*)-count(DISTINCT participacion_ref) duplicados,
    count(*) FILTER (WHERE situacion IS NULL OR tipo_lista IS NULL
       OR (en_restriccion IS TRUE AND disponible_desde IS NULL)) incompletos
  FROM base
 )
 SELECT coalesce(jsonb_agg(jsonb_build_object(
    'bolsa_ref',b.bolsa_ref,'categoria_ref',b.categoria_ref,'acta_ref',b.acta_ref,
    'vigente_desde',b.vigente_desde,'vigente_hasta',b.vigente_hasta,
    'total',b.total_participaciones,'tipo_lista',b.tipo_lista,
    'participacion_ref',b.participacion_ref,'fila_numero',b.fila_numero,
    'orden',b.orden_publico,'estado_efectivo',b.situacion,
    'es_origen',b.participacion_ref=(SELECT cp.participacion_ref
      FROM vec_bolsa_llamamientos.contrato_participacion cp WHERE cp.evento_ref=p_evento_ref),
    'fecha_disponible',CASE WHEN b.en_restriccion IS TRUE THEN
      b.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid'
      ELSE b.fecha_disponible END)
    ORDER BY b.bolsa_ref,b.orden_publico),'[]'::jsonb),
    v.conteo,v.esperado,v.duplicados,v.incompletos
 INTO v_filas,v_conteo,v_esperado,v_duplicados,v_incompletos
 FROM validacion v LEFT JOIN base b ON true GROUP BY v.conteo,v.esperado,v.duplicados,v.incompletos;
 IF v_conteo<1 OR v_conteo>1000000 OR v_conteo<>v_esperado
    OR v_duplicados<>0 OR v_incompletos<>0
    OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(v_filas) x
      WHERE x->>'bolsa_ref'=p_bolsa_ref
        AND x->>'participacion_ref'=(SELECT cp.participacion_ref
          FROM vec_bolsa_llamamientos.contrato_participacion cp WHERE cp.evento_ref=p_evento_ref)) THEN
  RAISE EXCEPTION 'captura B10 incompleta: filas %, esperadas %, duplicadas %, incompletas %',
   v_conteo,v_esperado,v_duplicados,v_incompletos USING ERRCODE='55000';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.captura_cese_b10(
  evento_ref,fase,origen_ref,origen_posicion,corte,filas,filas_sha256,capturada_en)
 VALUES(p_evento_ref,p_fase,p_origen_ref,p_origen_posicion,v_corte,v_filas,
  encode(sha256(convert_to(v_filas::text,'UTF8')),'hex'),v_corte);
 RETURN QUERY SELECT v_corte,v_filas,NULL::timestamptz,NULL::bytea,NULL::bytea,NULL::bytea;
END $f$;

-- El adaptador ya cotejó el proveedor externo y el contrato canónico. El
-- insert inmutable precede al COMMIT público, para que un 503 tras ese COMMIT
-- recupere exactamente los mismos bytes y la misma fecha.
CREATE FUNCTION vec_bolsa_llamamientos.guardar_material_cese_b10_v1(
 p_evento_ref text,p_fase text,p_corte timestamptz,p_anterior_actualizada_en timestamptz,
 p_proyeccion_v2 bytea,p_manifiesto_v2 bytea,p_bolsas_v1 bytea)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_captura vec_bolsa_llamamientos.captura_cese_b10;
 v_previo vec_bolsa_llamamientos.material_cese_b10;
 v_bolsas jsonb; v_proyeccion jsonb; v_huella text; v_mapa jsonb; v_total bigint;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_publicador_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'material B10 denegado' USING ERRCODE='42501';
 END IF;
 IF p_evento_ref IS NULL OR p_fase NOT IN ('cese','vencimiento')
    OR p_corte IS NULL OR NOT isfinite(p_corte)
    OR (p_anterior_actualizada_en IS NOT NULL AND
        (NOT isfinite(p_anterior_actualizada_en) OR p_anterior_actualizada_en>=p_corte))
    OR octet_length(p_proyeccion_v2) NOT BETWEEN 2 AND 268435456
    OR octet_length(p_manifiesto_v2) NOT BETWEEN 2 AND 268435456
    OR octet_length(p_bolsas_v1) NOT BETWEEN 2 AND 67108864 THEN
  RAISE EXCEPTION 'material B10 inválido' USING ERRCODE='22023';
 END IF;
 v_bolsas:=convert_from(p_bolsas_v1,'UTF8')::jsonb;
 v_proyeccion:=convert_from(p_proyeccion_v2,'UTF8')::jsonb;
 IF jsonb_typeof(v_bolsas)<>'object' OR jsonb_typeof(v_bolsas->'bolsas')<>'array'
    OR jsonb_array_length(v_bolsas->'bolsas')>128
    OR jsonb_typeof(v_proyeccion)<>'object'
    OR v_bolsas->>'generado_en' IS DISTINCT FROM
       to_char(p_corte AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    OR v_proyeccion#>>'{fuente,actualizada_en}' IS DISTINCT FROM v_bolsas->>'generado_en'
    OR EXISTS (SELECT 1 FROM jsonb_object_keys(v_bolsas) k WHERE k NOT IN ('generado_en','bolsas'))
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(v_bolsas->'bolsas') b,
                jsonb_object_keys(b) k WHERE k NOT IN
         ('bolsa_ref','categoria','categoria_clave','grupos','tipo_lista','vigente_desde','vigente_hasta','total','posiciones'))
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(v_bolsas->'bolsas') b,
                jsonb_array_elements(b->'posiciones') p,
                jsonb_object_keys(p) k WHERE k NOT IN ('orden','documento_enmascarado','estado_clave')) THEN
  RAISE EXCEPTION 'material B10 no minimizado o corte divergente' USING ERRCODE='22023';
 END IF;
 v_huella:=encode(sha256(p_proyeccion_v2||p_manifiesto_v2||p_bolsas_v1),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:captura-cese-b10',0));
 SELECT * INTO v_captura FROM vec_bolsa_llamamientos.captura_cese_b10 c
  WHERE c.evento_ref=p_evento_ref AND c.fase=p_fase AND c.corte=p_corte;
 IF NOT FOUND THEN RAISE EXCEPTION 'captura B10 ausente' USING ERRCODE='23503'; END IF;
 SELECT coalesce(jsonb_object_agg(b->>'bolsa_ref',b),'{}'::jsonb),
        coalesce(sum(jsonb_array_length(b->'posiciones')),0)
 INTO v_mapa,v_total FROM jsonb_array_elements(v_bolsas->'bolsas') b;
 IF v_total<>jsonb_array_length(v_captura.filas)
    OR jsonb_array_length(v_bolsas->'bolsas')<>(SELECT count(*) FROM jsonb_object_keys(v_mapa))
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(v_bolsas->'bolsas') b,
                jsonb_array_elements(b->'posiciones') p
      WHERE p->>'documento_enmascarado' !~ '^\*{3}[0-9]{4}\*{2}$'
         OR p->>'estado_clave' NOT IN ('disponible','ocupado','no_disponible','excluido','renuncia_pendiente'))
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(v_captura.filas) f
      WHERE v_mapa->(f->>'bolsa_ref') IS NULL
        OR (v_mapa->(f->>'bolsa_ref'))->>'tipo_lista' IS DISTINCT FROM f->>'tipo_lista'
        OR ((v_mapa->(f->>'bolsa_ref'))->>'total')::bigint IS DISTINCT FROM (f->>'total')::bigint
        OR ((v_mapa->(f->>'bolsa_ref'))->'posiciones'->((f->>'orden')::integer-1))->>'estado_clave'
           IS DISTINCT FROM CASE f->>'estado_efectivo'
             WHEN 'disponible' THEN 'disponible'
             WHEN 'trabajando' THEN 'ocupado'
             WHEN 'pendiente_incorporacion' THEN 'ocupado'
             WHEN 'no_disponible' THEN 'no_disponible'
             WHEN 'disponible_desde' THEN 'no_disponible'
             WHEN 'excluido' THEN 'excluido'
             WHEN 'renuncia' THEN 'renuncia_pendiente' ELSE NULL END
        OR ((v_mapa->(f->>'bolsa_ref'))->'posiciones'->((f->>'orden')::integer-1))->>'orden'
           IS DISTINCT FROM f->>'orden') THEN
  RAISE EXCEPTION 'material B10 no corresponde a la captura' USING ERRCODE='22023';
 END IF;
 SELECT * INTO v_previo FROM vec_bolsa_llamamientos.material_cese_b10 m
  WHERE m.evento_ref=p_evento_ref AND m.fase=p_fase;
 IF FOUND THEN
  IF v_previo.material_sha256<>v_huella OR v_previo.anterior_actualizada_en IS DISTINCT FROM p_anterior_actualizada_en
     OR v_previo.proyeccion_v2<>p_proyeccion_v2 OR v_previo.manifiesto_v2<>p_manifiesto_v2
     OR v_previo.bolsas_v1<>p_bolsas_v1 THEN
   RAISE EXCEPTION 'replay B10 divergente' USING ERRCODE='VBC10';
  END IF;
  RETURN true;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.material_cese_b10(
  evento_ref,fase,anterior_actualizada_en,proyeccion_v2,manifiesto_v2,bolsas_v1,material_sha256,guardado_en)
 VALUES(p_evento_ref,p_fase,p_anterior_actualizada_en,p_proyeccion_v2,p_manifiesto_v2,p_bolsas_v1,
  v_huella,date_trunc('microseconds',clock_timestamp()));
 RETURN false;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.capturar_cese_b10_v1(bigint,text,text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.guardar_material_cese_b10_v1(text,text,timestamptz,timestamptz,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.capturar_cese_b10_v1(bigint,text,text,text,text)
 TO vec_bolsa_llamamientos_publicador_cese;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.guardar_material_cese_b10_v1(text,text,timestamptz,timestamptz,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_publicador_cese;
COMMIT;
