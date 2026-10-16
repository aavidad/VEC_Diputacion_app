\set ON_ERROR_STOP on
-- B130. Prueba el CHECK instalado mediante una copia temporal; no registra
-- denegaciones reales ni altera su historia.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $estructura$
DECLARE tabla regclass:=to_regclass('vec_bolsa_llamamientos.denegacion_frontera_portal_externo');
 definicion text;
BEGIN
 IF tabla IS NULL OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'B130 prueba: exige DBA y tabla instalada'; END IF;
 SELECT pg_get_constraintdef(c.oid) INTO definicion FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname='denegacion_frontera_portal_externo_ruta_check'
  AND c.contype='c' AND c.convalidated;
 IF definicion IS NULL OR
  encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
   '72f6260369587777c08531ac627925026f150a4682805ca19c7ca2c48e0d4ffa'
  OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid=tabla
   AND c.relowner='vec_bolsa_llamamientos_propietario'::regrole
   AND c.relacl IS NULL AND c.relrowsecurity AND c.relforcerowsecurity)
 THEN RAISE EXCEPTION 'B130 prueba: postimagen no instalada'; END IF;
END $estructura$;

CREATE TEMP TABLE b130_rutas (
 LIKE vec_bolsa_llamamientos.denegacion_frontera_portal_externo INCLUDING CONSTRAINTS
) ON COMMIT DROP;
INSERT INTO b130_rutas(evento_id,correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
SELECT row_number() OVER (), 'corr_no_disponible','autenticacion_requerida',
 'api.bolsa.candidato.ruta_exacta',ruta,NULL,clock_timestamp()
FROM unnest(ARRAY[
 '/api/vec/bolsa/mi-bolsa',
 '/api/vec/bolsa/mi-bolsa/historial',
 '/api/vec/bolsa/mi-bolsa/solicitudes',
 '/api/vec/bolsa/mi-bolsa/respuestas',
 '/api/vec/bolsa/mi-bolsa/disposiciones',
 '/api/vec/bolsa/mi-bolsa/contacto',
 '/api/vec/bolsa/mi-bolsa/solicitudes-documentales',
 '/api/vec/bolsa/mi-bolsa/preferencias',
 '/api/vec/bolsa/mi-bolsa/justificantes'
]::text[]) AS r(ruta);

DO $rutas$
BEGIN
 IF (SELECT count(*) FROM pg_temp.b130_rutas)<>9 THEN
  RAISE EXCEPTION 'B130 prueba: ruta admitida perdida'; END IF;
 BEGIN
  INSERT INTO pg_temp.b130_rutas(evento_id,correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
  VALUES(10,'corr_no_disponible','autenticacion_requerida',
   'api.bolsa.candidato.ruta_exacta','/api/vec/bolsa/mi-bolsa/justificantes/otro',NULL,clock_timestamp());
  RAISE EXCEPTION 'B130 prueba: subruta indebida admitida';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO pg_temp.b130_rutas(evento_id,correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
  VALUES(11,'corr_no_disponible','autenticacion_requerida',
   'api.bolsa.candidato.ruta_exacta','/api/vec/bolsa/mi-bolsa/desconocida',NULL,clock_timestamp());
  RAISE EXCEPTION 'B130 prueba: ruta ajena admitida';
 EXCEPTION WHEN check_violation THEN NULL; END;
 IF (SELECT count(*) FROM pg_temp.b130_rutas)<>9 THEN
  RAISE EXCEPTION 'B130 prueba: rechazo escribió fila'; END IF;
END $rutas$;
ROLLBACK;
\echo B130_PRUEBA_RUTAS_AUDITORIA_OK
