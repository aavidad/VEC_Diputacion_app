\set ON_ERROR_STOP on
-- Ensayo de proyección sobre el clon PostgreSQL 18. El propietario prepara
-- datos privados de ensayo; no sustituye un acto firmado. ROLLBACK conserva
-- la base y los recibos anteriores.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='20s';
DO $prueba$
DECLARE e record; cand text; j jsonb; respuesta jsonb;
 inicio timestamptz;
 base timestamptz:=clock_timestamp()+interval '1 day';
 recibo text:='recibo:respuesta-portal:'||repeat('a',64);
BEGIN
 SELECT l.bolsa_ref,l.llamamiento_ref,c.acta_ref,c.instantanea_ref,c.version_instantanea,en.participacion_ref
 INTO STRICT e FROM vec_bolsa_llamamientos.llamamiento_emitido l
 JOIN vec_bolsa_llamamientos.constitucion c USING(bolsa_ref)
 JOIN vec_bolsa_llamamientos.constitucion_entrada en USING(instantanea_ref,version_instantanea)
 WHERE NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento r
  WHERE r.llamamiento_ref=l.llamamiento_ref AND r.participacion_ref=en.participacion_ref)
 ORDER BY en.orden LIMIT 1;
 SELECT candidato_ref INTO cand FROM vec_bolsa_llamamientos.vinculo_candidato WHERE participacion_ref=e.participacion_ref;
 IF cand IS NULL THEN
  cand:='can_carmenMolinaB101AAAAAAA';
  INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(participacion_ref,candidato_ref,acta_ref,instantanea_ref,version_instantanea,registrada_en)
   VALUES(e.participacion_ref,cand,e.acta_ref,e.instantanea_ref,e.version_instantanea,base-interval '1 day');
 END IF;
 INSERT INTO vec_bolsa_llamamientos.respuesta_portal_llamamiento(respuesta_ref,recibo_ref,bolsa_ref,participacion_ref,
 candidato_ref,llamamiento_ref,respuesta,modo,contacto_en,vence_antes_de,regla_ref,clave_idempotencia,decision_ref,respondida_en)
 VALUES('respuesta-portal:'||repeat('a',64),recibo,e.bolsa_ref,e.participacion_ref,cand,e.llamamiento_ref,
 'acepta','firme',base-interval '1 hour',base+interval '2 days','b29.portal_candidato','b101:carmen:acepta','decision:b101:carmen',base);
 BEGIN
  PERFORM vec_bolsa_llamamientos.leer_portal_candidato_v1(cand,base,ARRAY['contactado']);
  RAISE EXCEPTION 'B101 clave=lectura_sin_consumo actual=admitida esperado=42501';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 PERFORM vec_bolsa_llamamientos.anotar_consumo_candidato_v1('consulta',cand,'mi-bolsa');
 inicio:=clock_timestamp();
 j:=vec_bolsa_llamamientos.leer_portal_candidato_v1(cand,base,ARRAY['contactado']);
 RAISE NOTICE 'B101 lectura_ms=%',extract(epoch FROM clock_timestamp()-inicio)*1000;
 SELECT x->'ultima_respuesta' INTO STRICT respuesta FROM jsonb_array_elements(j) x WHERE x->>'bolsa'=e.bolsa_ref;
 IF respuesta->>'recibo' IS DISTINCT FROM recibo OR (respuesta->>'en_revision')::boolean IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'B101 clave=respuesta_pendiente actual=% esperado=recibo_propio/en_revision_true',respuesta;
 END IF;
 IF position(e.participacion_ref IN j::text)>0 THEN
  RAISE EXCEPTION 'B101 clave=minimizacion actual=participacion_visible esperado=referencia_oculta';
 END IF;
 j:=vec_bolsa_llamamientos.leer_portal_candidato_v1(cand,base-interval '1 second',ARRAY['contactado']);
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(j) x WHERE x->'ultima_respuesta'->>'recibo'=recibo) THEN
  RAISE EXCEPTION 'B101 clave=corte_historico actual=respuesta_futura_visible esperado=oculta';
 END IF;
 -- La confirmación tiene «desde» anterior a la respuesta, pero se registra
 -- después. Hasta ese registro la respuesta continúa pendiente.
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(e.participacion_ref,'pendiente_incorporacion',base-interval '30 minutes','Incorporación confirmada',
 'per_carmenMolinaB101AAAAAAAA',base+interval '1 hour','b101:carmen:confirma','recibo:b101:confirma');
 j:=vec_bolsa_llamamientos.leer_portal_candidato_v1(cand,base,ARRAY['contactado']);
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(j) x WHERE x->>'bolsa'=e.bolsa_ref
 AND x->'ultima_respuesta'->>'en_revision'='true') THEN
  RAISE EXCEPTION 'B101 clave=confirmacion_futura actual=sin_aviso esperado=revision_pendiente';
 END IF;
 j:=vec_bolsa_llamamientos.leer_portal_candidato_v1(cand,base+interval '2 hours',ARRAY['contactado']);
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(j) x WHERE x->>'bolsa'=e.bolsa_ref
 AND x->'ultima_respuesta'->>'en_revision'='false' AND x->'ultima_respuesta'->>'recibo'=recibo) THEN
  RAISE EXCEPTION 'B101 clave=confirmacion_registrada actual=% esperado=recibo_igual/en_revision_false',j;
 END IF;
END $prueba$;
SELECT 'B101-PRUEBA-OK';
ROLLBACK;
