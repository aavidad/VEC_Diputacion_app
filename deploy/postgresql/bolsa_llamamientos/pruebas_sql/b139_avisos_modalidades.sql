\set ON_ERROR_STOP on
-- Ensayo focal en copia desechable, tras B138/B139. Filas sintéticas y todo ROLLBACK.
-- Desactiva solo las dos guardas de publicación V3 para aislar el trigger B129;
-- no acredita autorización, firma ni publicación nominal del producto.
BEGIN ISOLATION LEVEL SERIALIZABLE;
CREATE TEMP TABLE prueba_b139_pares(
 bolsa_ref text,participacion_a text,candidato_a text,participacion_b text,candidato_b text,
 oferta_antes text,oferta_despues text,message_id_antes text
) ON COMMIT DROP;
CREATE TEMP TABLE prueba_b139_elegibles(bolsa_ref text,participacion_ref text,candidato_ref text) ON COMMIT DROP;
CREATE TEMP TABLE prueba_b139_instantanea(participacion_ref text,message_id text) ON COMMIT DROP;
CREATE TEMP TABLE prueba_b139_reclamo(participacion_ref text,message_id text,intento integer) ON COMMIT DROP;
GRANT SELECT,INSERT,UPDATE ON prueba_b139_pares TO vec_bolsa_llamamientos_propietario;
GRANT SELECT,INSERT ON prueba_b139_elegibles,prueba_b139_instantanea,prueba_b139_reclamo
 TO vec_bolsa_llamamientos_propietario;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';

DO $pre$
BEGIN
 IF to_regclass('vec_bolsa_llamamientos.preferencia_modalidades_version') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.crear_aviso_oferta_b129()') IS NULL
 THEN RAISE EXCEPTION 'B139 no está instalada en esta copia' USING ERRCODE='55000'; END IF;
END $pre$;
ALTER TABLE vec_bolsa_llamamientos.oferta_publicada DISABLE TRIGGER oferta_politica_b47;
ALTER TABLE vec_bolsa_llamamientos.oferta_publicada DISABLE TRIGGER verificar_aceptacion_previa_oferta_b75;
DO $focal$
DECLARE r record;v_bolsa text;v_a record;v_b record;v_oferta text;v_i integer;
 v_candidato text;v_modalidades text[];v_version bigint;v_corte timestamptz:=clock_timestamp();
BEGIN
 INSERT INTO prueba_b139_elegibles
 SELECT s.bolsa_ref,s.participacion_ref,vc.candidato_ref
 FROM vec_bolsa_llamamientos.situaciones_en_v1(v_corte) s
 JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.participacion_ref=s.participacion_ref
 WHERE s.situacion='disponible' AND EXISTS(
  SELECT 1 FROM vec_bolsa_llamamientos.constitucion c
  JOIN vec_bolsa_llamamientos.bolsa_constituida b
    ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  JOIN vec_bolsa_llamamientos.constitucion_entrada e
    ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea
   AND e.participacion_ref=s.participacion_ref
  WHERE c.bolsa_ref=s.bolsa_ref AND b.estado='vigente'
    AND b.vigente_desde<=v_corte AND (b.vigente_hasta IS NULL OR v_corte<b.vigente_hasta));
 SELECT bolsa_ref INTO v_bolsa FROM prueba_b139_elegibles
 GROUP BY bolsa_ref HAVING count(DISTINCT candidato_ref)>=2
 ORDER BY count(DISTINCT candidato_ref),bolsa_ref LIMIT 1;
 IF v_bolsa IS NULL THEN RAISE EXCEPTION 'B139: faltan dos candidatos elegibles en la semilla' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT v_a FROM prueba_b139_elegibles WHERE bolsa_ref=v_bolsa
 ORDER BY participacion_ref LIMIT 1;
 SELECT * INTO STRICT v_b FROM prueba_b139_elegibles
 WHERE bolsa_ref=v_bolsa AND candidato_ref<>v_a.candidato_ref
 ORDER BY participacion_ref LIMIT 1;
 INSERT INTO prueba_b139_pares VALUES(v_bolsa,v_a.participacion_ref,v_a.candidato_ref,
  v_b.participacion_ref,v_b.candidato_ref,
  'oferta:'||encode(sha256(convert_to('b139-antes:'||v_bolsa,'UTF8')),'hex'),
  'oferta:'||encode(sha256(convert_to('b139-despues:'||v_bolsa,'UTF8')),'hex'),NULL);

 -- Dos personas con última versión opuesta: A acepta vacante, B solo sustitución.
 FOR v_i IN 1..2 LOOP
  v_candidato:=CASE v_i WHEN 1 THEN v_a.candidato_ref ELSE v_b.candidato_ref END;
  v_modalidades:=CASE v_i WHEN 1 THEN ARRAY['vacante'] ELSE ARRAY['sustitucion'] END;
  SELECT coalesce(max(version),0)+1 INTO v_version
  FROM vec_bolsa_llamamientos.preferencia_modalidades_version WHERE candidato_ref=v_candidato;
  INSERT INTO vec_bolsa_llamamientos.preferencia_modalidades_version
   (candidato_ref,version,modalidades,catalogo_snapshot,clave_idempotencia,
    solicitud_huella_sha256,contexto_huella_sha256,actor_ref,perfil_ref,superficie,
    recibo_ref,decision_ref,auditoria_ref,registrada_en)
  VALUES(v_candidato,v_version,v_modalidades,'[]'::jsonb,
   '@'||encode(sha256(convert_to('b139-primera:'||v_candidato,'UTF8')),'hex'),
   encode(sha256(convert_to('solicitud-b139:'||v_candidato,'UTF8')),'hex'),
   encode(sha256(convert_to('contexto-b139:'||v_candidato,'UTF8')),'hex'),
   'per_prueba_b139','prf_prueba_b139','externa_personal',
   'recibo:preferencias-modalidades:'||encode(sha256(convert_to('recibo-b139-primera:'||v_candidato,'UTF8')),'hex'),
   'decision:prueba-b139-primera:'||v_candidato,
   'auditoria:prueba-b139-primera:'||v_candidato,v_corte);
 END LOOP;
 SELECT oferta_antes INTO STRICT v_oferta FROM prueba_b139_pares;
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada
  (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
   publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(v_oferta,'recibo:'||v_oferta,v_bolsa,'per_actoractoractoractoractor',
  'clave-prueba-b139-antes',jsonb_build_object('modalidad_clave','vacante'),
  '{}'::jsonb,v_corte,v_corte+interval '1 day',repeat('a',64),'decision:prueba-b139-antes');
 UPDATE vec_bolsa_llamamientos.aviso_oferta_cabecera SET
  huella_token=sha256(decode(repeat('11',32),'hex')),version_token='b139-prueba'
 WHERE oferta_ref=v_oferta;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
   WHERE oferta_ref=v_oferta AND participacion_ref=v_a.participacion_ref)
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
   WHERE oferta_ref=v_oferta AND participacion_ref=v_b.participacion_ref)
 THEN RAISE EXCEPTION 'B139: la primera publicación no filtró por modalidad' USING ERRCODE='55000'; END IF;
 UPDATE prueba_b139_pares SET message_id_antes=(SELECT message_id
  FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
  WHERE oferta_ref=v_oferta AND participacion_ref=v_a.participacion_ref);
 INSERT INTO prueba_b139_instantanea
 SELECT participacion_ref,message_id FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
 WHERE oferta_ref=v_oferta;

 -- Cambio prospectivo: A ninguna; B vacante. La primera instantánea no se reescribe.
 FOR v_i IN 1..2 LOOP
  v_candidato:=CASE v_i WHEN 1 THEN v_a.candidato_ref ELSE v_b.candidato_ref END;
  v_modalidades:=CASE v_i WHEN 1 THEN ARRAY[]::text[] ELSE ARRAY['vacante'] END;
  SELECT max(version)+1 INTO v_version
  FROM vec_bolsa_llamamientos.preferencia_modalidades_version WHERE candidato_ref=v_candidato;
  INSERT INTO vec_bolsa_llamamientos.preferencia_modalidades_version
   (candidato_ref,version,modalidades,catalogo_snapshot,clave_idempotencia,
    solicitud_huella_sha256,contexto_huella_sha256,actor_ref,perfil_ref,superficie,
    recibo_ref,decision_ref,auditoria_ref,registrada_en)
  VALUES(v_candidato,v_version,v_modalidades,'[]'::jsonb,
   '@'||encode(sha256(convert_to('b139-segunda:'||v_candidato,'UTF8')),'hex'),
   encode(sha256(convert_to('solicitud-b139-2:'||v_candidato,'UTF8')),'hex'),
   encode(sha256(convert_to('contexto-b139-2:'||v_candidato,'UTF8')),'hex'),
   'per_prueba_b139','prf_prueba_b139','externa_personal',
   'recibo:preferencias-modalidades:'||encode(sha256(convert_to('recibo-b139-segunda:'||v_candidato,'UTF8')),'hex'),
   'decision:prueba-b139-segunda:'||v_candidato,
   'auditoria:prueba-b139-segunda:'||v_candidato,v_corte);
 END LOOP;
 SELECT * INTO STRICT r FROM prueba_b139_pares;
 IF (SELECT message_id FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
     WHERE oferta_ref=r.oferta_antes AND participacion_ref=r.participacion_a)
      IS DISTINCT FROM r.message_id_antes
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
     WHERE oferta_ref=r.oferta_antes AND participacion_ref=r.participacion_b)
 THEN RAISE EXCEPTION 'B139: preferencias nuevas alteraron la instantánea anterior' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada
  (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
   publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(r.oferta_despues,'recibo:'||r.oferta_despues,v_bolsa,'per_actoractoractoractoractor',
  'clave-prueba-b139-despues',jsonb_build_object('modalidad_clave','vacante'),
  '{}'::jsonb,v_corte,v_corte+interval '1 day',repeat('b',64),'decision:prueba-b139-despues');
 UPDATE vec_bolsa_llamamientos.aviso_oferta_cabecera SET
  huella_token=sha256(decode(repeat('22',32),'hex')),version_token='b139-prueba'
 WHERE oferta_ref=r.oferta_despues;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
   WHERE oferta_ref=r.oferta_despues AND participacion_ref=r.participacion_a)
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
   WHERE oferta_ref=r.oferta_despues AND participacion_ref=r.participacion_b)
 THEN RAISE EXCEPTION 'B139: la siguiente publicación no tomó la versión nueva' USING ERRCODE='55000'; END IF;
 -- NULL explícito vuelve a todas las modalidades para A en una publicación posterior.
 SELECT max(version)+1 INTO v_version
 FROM vec_bolsa_llamamientos.preferencia_modalidades_version WHERE candidato_ref=r.candidato_a;
 INSERT INTO vec_bolsa_llamamientos.preferencia_modalidades_version
  (candidato_ref,version,modalidades,catalogo_snapshot,clave_idempotencia,
   solicitud_huella_sha256,contexto_huella_sha256,actor_ref,perfil_ref,superficie,
   recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(r.candidato_a,v_version,NULL,'[]'::jsonb,
  '@'||encode(sha256(convert_to('b139-todas:'||r.candidato_a,'UTF8')),'hex'),
  encode(sha256(convert_to('solicitud-b139-todas:'||r.candidato_a,'UTF8')),'hex'),
  encode(sha256(convert_to('contexto-b139-todas:'||r.candidato_a,'UTF8')),'hex'),
  'per_prueba_b139','prf_prueba_b139','externa_personal',
  'recibo:preferencias-modalidades:'||encode(sha256(convert_to('recibo-b139-todas:'||r.candidato_a,'UTF8')),'hex'),
  'decision:prueba-b139-todas:'||r.candidato_a,
  'auditoria:prueba-b139-todas:'||r.candidato_a,v_corte);
 v_oferta:='oferta:'||encode(sha256(convert_to('b139-todas:'||v_bolsa,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada
  (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
   publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(v_oferta,'recibo:'||v_oferta,v_bolsa,'per_actoractoractoractoractor',
  'clave-prueba-b139-todas',jsonb_build_object('modalidad_clave','vacante'),
  '{}'::jsonb,v_corte,v_corte+interval '1 day',repeat('c',64),'decision:prueba-b139-todas');
 UPDATE vec_bolsa_llamamientos.aviso_oferta_cabecera SET
  huella_token=sha256(decode(repeat('33',32),'hex')),version_token='b139-prueba'
 WHERE oferta_ref=v_oferta;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
   WHERE oferta_ref=v_oferta AND participacion_ref=r.participacion_a)
 THEN RAISE EXCEPTION 'B139: preferencia NULL no incluyó a A' USING ERRCODE='55000'; END IF;
END $focal$;
SET CONSTRAINTS vec_bolsa_llamamientos.exigir_token_aviso_oferta_b129 IMMEDIATE;

-- La reclamación usa solo la instantánea existente. El token es local y sintético.
SELECT oferta_antes FROM prueba_b139_pares \gset b139_
RESET ROLE;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_ejecutor;
SELECT participacion_ref,message_id,intento FROM vec_bolsa_llamamientos.reclamar_aviso_oferta_v1(
 :'b139_oferta_antes',decode(repeat('11',32),'hex'),1) \gset reclamo_
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO prueba_b139_reclamo VALUES(:'reclamo_participacion_ref',:'reclamo_message_id',:reclamo_intento);
DO $reclamo$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM prueba_b139_pares;
 IF NOT EXISTS(SELECT 1 FROM prueba_b139_reclamo c
   JOIN prueba_b139_instantanea i USING(participacion_ref,message_id)
   WHERE c.intento=1)
 OR EXISTS(SELECT 1 FROM prueba_b139_reclamo c WHERE c.participacion_ref=r.participacion_b)
 OR (SELECT message_id FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
     WHERE oferta_ref=r.oferta_antes AND participacion_ref=r.participacion_a)
    IS DISTINCT FROM r.message_id_antes
 THEN RAISE EXCEPTION 'B139: la reclamación alteró destinatarios o MessageID' USING ERRCODE='55000'; END IF;
END $reclamo$;
ROLLBACK;
