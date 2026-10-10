\set ON_ERROR_STOP on
-- Ejecutar tras B100 en PostgreSQL 18 desechable con al menos un llamamiento
-- emitido en una bolsa de dos o más personas en turno. Todo se deshace.
-- Comprueba que una respuesta de Mi Bolsa sin reflejar saca a la persona del
-- turno (orden vigente nulo, razón respuesta_portal_pendiente), que la marca
-- «en revisión» distingue aceptación y renuncia, que el corte histórico
-- anterior a la respuesta no cambia y que, cuando RRHH registra una situación
-- después de la respuesta (aunque la feche antes), la marca desaparece.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='20s';
SET ROLE vec_bolsa_llamamientos_propietario;

SELECT quote_literal(l.bolsa_ref) AS bolsa_sql, quote_literal(l.llamamiento_ref) AS llamamiento_sql,
       quote_literal((clock_timestamp() - interval '2 hours')::text) AS t0_sql
  FROM vec_bolsa_llamamientos.llamamiento_emitido l
 WHERE (SELECT count(*) FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(l.bolsa_ref, clock_timestamp()) o
         WHERE o.orden_vigente IS NOT NULL) >= 2
 ORDER BY l.emitido_en DESC LIMIT 1 \gset

-- Las dos primeras personas en turno antes de responder.
SELECT quote_literal(max(o.participacion_ref) FILTER (WHERE o.orden_vigente = 1)) AS primera_sql,
       quote_literal(max(o.participacion_ref) FILTER (WHERE o.orden_vigente = 2)) AS segunda_sql
  FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(:bolsa_sql, clock_timestamp()) o \gset

-- La primera acepta en firme y la segunda renuncia desde Mi Bolsa.
INSERT INTO vec_bolsa_llamamientos.respuesta_portal_llamamiento(respuesta_ref, recibo_ref, bolsa_ref, participacion_ref,
  candidato_ref, llamamiento_ref, respuesta, modo, contacto_en, vence_antes_de, regla_ref, clave_idempotencia, decision_ref, respondida_en)
SELECT 'respuesta-portal:'||encode(sha256(convert_to('b100'||chr(31)||x.p,'UTF8')),'hex'),
       'recibo:respuesta-portal:'||encode(sha256(convert_to('b100:recibo'||chr(31)||x.p,'UTF8')),'hex'),
       :bolsa_sql, x.p, 'can_b100PruebaSinteticaAAAAAAA', :llamamiento_sql, x.r, 'firme',
       :t0_sql::timestamptz - interval '1 hour', :t0_sql::timestamptz + interval '2 days',
       'b29.portal_candidato', 'b100:prueba:'||x.r, 'decision:b100:'||x.r, :t0_sql::timestamptz
  FROM (VALUES (:primera_sql, 'acepta'), (:segunda_sql, 'renuncia')) AS x(p, r);

SELECT set_config('b100.bolsa', :bolsa_sql, true), set_config('b100.primera', :primera_sql, true),
       set_config('b100.segunda', :segunda_sql, true), set_config('b100.t0', :t0_sql, true) \g /dev/null

DO $fuera$
DECLARE b text := current_setting('b100.bolsa'); p1 text := current_setting('b100.primera');
 p2 text := current_setting('b100.segunda'); t0 timestamptz := current_setting('b100.t0')::timestamptz;
 v record; n_turno int; marca1 text; marca2 text;
BEGIN
 FOR v IN SELECT * FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b, clock_timestamp()) o WHERE o.participacion_ref IN (p1, p2) LOOP
  IF v.orden_vigente IS NOT NULL OR v.razon <> 'respuesta_portal_pendiente' OR v.situacion <> 'disponible' THEN
   RAISE EXCEPTION 'B100: con respuesta pendiente sigue en turno: % % % %', v.participacion_ref, v.orden_vigente, v.razon, v.situacion;
  END IF;
 END LOOP;
 SELECT count(*) INTO n_turno FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b, clock_timestamp()) o
  WHERE o.orden_vigente = 1 AND o.participacion_ref NOT IN (p1, p2);
 IF n_turno <> 1 THEN RAISE EXCEPTION 'B100: el primero en turno no avanza'; END IF;
 -- Corte anterior a la respuesta: el orden histórico no cambia.
 SELECT count(*) INTO n_turno FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b, t0 - interval '1 second') o
  WHERE o.participacion_ref = p1 AND o.orden_vigente = 1;
 IF n_turno <> 1 THEN RAISE EXCEPTION 'B100: el corte anterior a la respuesta cambió'; END IF;
 SELECT m.en_revision INTO marca1 FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(b, clock_timestamp()) m WHERE m.participacion_ref = p1;
 SELECT m.en_revision INTO marca2 FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(b, clock_timestamp()) m WHERE m.participacion_ref = p2;
 IF marca1 IS DISTINCT FROM 'aceptacion_pendiente' OR marca2 IS DISTINCT FROM 'renuncia_pendiente' THEN
  RAISE EXCEPTION 'B100: marcas inesperadas % %', marca1, marca2;
 END IF;
END $fuera$;

-- RRHH refleja la aceptación (pendiente de incorporación) con «desde» anterior
-- a la respuesta y la renuncia con «desde» posterior: ambas cuentan porque se
-- registran después.
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref, situacion, desde, motivo, actor, registrada_en, clave_idempotencia, recibo_ref)
SELECT x.p, x.s, x.d, 'Prueba B100', 'per_b100PruebaSinteticaAAAAAAAA', clock_timestamp(), 'b100:refleja:'||x.s, 'recibo:b100:'||x.s
  FROM (VALUES (:primera_sql, 'pendiente_incorporacion', :t0_sql::timestamptz - interval '30 minutes'),
               (:segunda_sql, 'renuncia', :t0_sql::timestamptz + interval '30 minutes')) AS x(p, s, d);

DO $reflejada$
DECLARE b text := current_setting('b100.bolsa'); p1 text := current_setting('b100.primera');
 p2 text := current_setting('b100.segunda'); n int;
BEGIN
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b, clock_timestamp()) o
  WHERE o.participacion_ref IN (p1, p2) AND (o.razon = 'respuesta_portal_pendiente' OR o.orden_vigente IS NOT NULL);
 IF n <> 0 THEN RAISE EXCEPTION 'B100: tras reflejar sigue la respuesta pendiente'; END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.consultar_marcas_participaciones_v1(b, clock_timestamp()) m
  WHERE m.participacion_ref IN (p1, p2) AND m.en_revision IS NOT NULL;
 IF n <> 0 THEN RAISE EXCEPTION 'B100: tras reflejar sigue la marca en revisión'; END IF;
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor',
     'vec_bolsa_llamamientos.respuestas_portal_sin_reflejar_v1(text,timestamptz)', 'EXECUTE') THEN
  RAISE EXCEPTION 'B100: la función auxiliar no debe estar concedida';
 END IF;
END $reflejada$;

SELECT 'B100-PRUEBA-OK';
ROLLBACK;
