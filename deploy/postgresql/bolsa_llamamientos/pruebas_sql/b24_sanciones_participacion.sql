\set ON_ERROR_STOP on
-- Prueba de 000026 (sanciones de Bolsa) en una base DESECHABLE con 000019 y
-- 000026 aplicadas. Se ejecuta como superusuario y termina en ROLLBACK.
-- TEST-ONLY: dentro de la transacción sustituye el consumidor V3 de la
-- situación por un doble que acepta cualquier decisión bien formada, y siembra
-- una constitución sintética sin disparadores de integridad. Nada persiste.
BEGIN;
SET LOCAL timezone = 'UTC';

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamptz, consumo_nuevo boolean)
LANGUAGE plpgsql AS $f$
DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT 'decision:doble', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', repeat('0',64), 'auditoria:doble', clock_timestamp(), true;
END $f$;

SET LOCAL session_replication_role = replica;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.constitucion(acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
VALUES('acta:b24','bolsa:b24',1,repeat('1',64),'instantanea:b24',1,repeat('2',64),'categoria:b24','sistema:prueba','2026-01-01','2026-01-01');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
VALUES('instantanea:b24',1,1,'participacion:b24:uno',1),('instantanea:b24',1,2,'participacion:b24:dos',2);
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:b24:uno','disponible','2026-01-01','Constitución de bolsa','sistema:constitucion','2026-01-01','constitucion:participacion:b24:uno','recibo:situacion:constitucion:participacion:b24:uno'),
      ('participacion:b24:dos','disponible','2026-01-01','Constitución de bolsa','sistema:constitucion','2026-01-01','constitucion:participacion:b24:dos','recibo:situacion:constitucion:participacion:b24:dos');
RESET ROLE;
SET LOCAL session_replication_role = origin;

CREATE FUNCTION pg_temp.decision(p text, actor text, accion text DEFAULT 'bolsa.situacion_participacion.cambiar') RETURNS bytea LANGUAGE sql AS $f$
 SELECT convert_to(jsonb_build_object('principal_id',actor,'accion',accion,'modulo_id','bolsa','tipo_recurso','participacion_bolsa',
  'finalidad','gestion_situacion_participacion','recurso_ref',p,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',repeat('3',64))::text,'UTF8')
$f$;
CREATE FUNCTION pg_temp.ref(p text, clave text) RETURNS text LANGUAGE sql AS $f$
 SELECT encode(sha256(convert_to(p || chr(31) || clave,'UTF8')),'hex')
$f$;
CREATE FUNCTION pg_temp.sancionar(p text, clave text, efecto text, causa text, resuelta_por text, desde timestamptz, hasta date, decis bytea DEFAULT NULL)
RETURNS TABLE(reutilizada boolean, sancion_ref text, recibo_ref text, situacion text, desde_out timestamptz) LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.registrar_sancion_participacion_v1(
  'bolsa:b24', p, 'sancion:' || pg_temp.ref(p, clave), 'b24.sancion.' || efecto, 'Consecuencia ' || efecto,
  efecto, causa, DATE '2026-09-20', 'registro:2026/000123', repeat('a',64), resuelta_por,
  'vec.bolsa.reglas:1:b24.sancion.' || efecto, repeat('b',64), hasta, DATE '2026-10-20',
  'vec.bolsa.reglas:1:b24.consecuencias', repeat('b',64),
  CASE WHEN efecto = 'ninguna' THEN NULL ELSE desde END, 'per_actor', clave,
  CASE WHEN efecto = 'ninguna' THEN 'recibo:sancion:' ELSE 'recibo:situacion:' END || pg_temp.ref(p, clave), desde,
  NULL, coalesce(decis, pg_temp.decision(p, 'per_actor')), NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)
$f$;

DO $prueba$
DECLARE r record; uno text := 'participacion:b24:uno'; dos text := 'participacion:b24:dos'; filas bigint;
 t0 timestamptz := '2026-09-25 09:00:00+00'; ref_susp text; ref_final text;
BEGIN
 -- Suspensión: aplica la operación B8 «pausar» en la misma transacción.
 SELECT * INTO STRICT r FROM pg_temp.sancionar(uno, 'k-susp', 'pausar', 'No se presentó', 'persona:jefatura', t0, DATE '2027-03-20');
 IF r.reutilizada OR r.situacion <> 'no_disponible' OR r.desde_out <> t0 THEN RAISE EXCEPTION 'B24: suspensión no aplicada %', r; END IF;
 ref_susp := r.sancion_ref;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=uno AND desde=t0 AND operacion='pausar' AND justificante_tipo='resolucion' AND validador='persona:jefatura') THEN
  RAISE EXCEPTION 'B24: falta la operación B8 de la suspensión';
 END IF;
 -- Reintento idéntico: mismo recibo, sin filas nuevas.
 SELECT * INTO STRICT r FROM pg_temp.sancionar(uno, 'k-susp', 'pausar', 'No se presentó', 'persona:jefatura', t0 + interval '1 minute', DATE '2027-03-20');
 IF NOT r.reutilizada OR r.sancion_ref <> ref_susp OR r.situacion <> 'no_disponible' THEN RAISE EXCEPTION 'B24: replay incorrecto %', r; END IF;
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=uno;
 IF filas <> 2 THEN RAISE EXCEPTION 'B24: el replay duplicó la situación (%)', filas; END IF;
 -- Misma clave con otra causa: rechazo.
 BEGIN
  PERFORM pg_temp.sancionar(uno, 'k-susp', 'pausar', 'Otra causa', 'persona:jefatura', t0 + interval '2 minute', DATE '2027-03-20');
  RAISE EXCEPTION 'B24: clave reutilizada aceptada';
 EXCEPTION WHEN SQLSTATE 'VBS01' THEN NULL; END;
 -- Baja resuelta por quien la registra: rechazo antes de consumir.
 BEGIN
  PERFORM pg_temp.sancionar(uno, 'k-baja-mal', 'excluir', 'Renuncia', 'per_actor', t0 + interval '3 minute', NULL);
  RAISE EXCEPTION 'B24: baja autoresuelta aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 -- Fin de suspensión en una baja: rechazo.
 BEGIN
  PERFORM pg_temp.sancionar(uno, 'k-baja-mal2', 'excluir', 'Renuncia', 'persona:jefatura', t0 + interval '3 minute', DATE '2027-03-20');
  RAISE EXCEPTION 'B24: baja con suspensión aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 -- Autorización de otra acción: rechazo.
 BEGIN
  PERFORM pg_temp.sancionar(dos, 'k-otra-accion', 'ninguna', 'Pasa al final', 'persona:jefatura', t0, NULL, pg_temp.decision(dos, 'per_actor', 'bolsa.otra.accion'));
  RAISE EXCEPTION 'B24: autorización ajena aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- Participación de otra bolsa o inexistente: rechazo.
 BEGIN
  PERFORM pg_temp.sancionar('participacion:ajena', 'k-ajena', 'ninguna', 'Pasa al final', 'persona:jefatura', t0, NULL);
  RAISE EXCEPTION 'B24: participación ajena aceptada';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;
 -- Pasar al final: sin cambio de situación.
 SELECT * INTO STRICT r FROM pg_temp.sancionar(dos, 'k-final', 'ninguna', 'Renuncia tardía', 'persona:jefatura', t0, NULL);
 IF r.reutilizada OR r.situacion IS NOT NULL OR r.desde_out IS NOT NULL THEN RAISE EXCEPTION 'B24: pasar al final cambió la situación %', r; END IF;
 ref_final := r.sancion_ref;
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=dos;
 IF filas <> 1 THEN RAISE EXCEPTION 'B24: pasar al final añadió situación'; END IF;
 SELECT * INTO STRICT r FROM pg_temp.sancionar(dos, 'k-final', 'ninguna', 'Renuncia tardía', 'persona:jefatura', t0, NULL);
 IF NOT r.reutilizada OR r.sancion_ref <> ref_final THEN RAISE EXCEPTION 'B24: replay sin efecto incorrecto'; END IF;
 -- Baja tras la suspensión (no_disponible → excluido, transición B2 existente).
 SELECT * INTO STRICT r FROM pg_temp.sancionar(uno, 'k-baja', 'excluir', 'Renuncia a nombramiento', 'persona:jefatura', t0 + interval '5 minute', NULL);
 IF r.situacion <> 'excluido' THEN RAISE EXCEPTION 'B24: baja no aplicada'; END IF;
 -- Una segunda baja no tiene transición: la decide B2.
 BEGIN
  PERFORM pg_temp.sancionar(uno, 'k-baja-2', 'excluir', 'Otra', 'persona:jefatura', t0 + interval '6 minute', NULL);
  RAISE EXCEPTION 'B24: transición imposible aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;

 -- Recurso de reposición.
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(uno, ref_susp, 'interpuesto', DATE '2026-09-24', 'registro:2026/000200', repeat('d',64), 'per_actor', 'r-1', t0 + interval '10 minute', NULL, pg_temp.decision(uno,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 IF r.reutilizada OR r.estado <> 'interpuesto' THEN RAISE EXCEPTION 'B24: recurso no registrado'; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(uno, ref_susp, 'interpuesto', DATE '2026-09-24', 'registro:2026/000200', repeat('d',64), 'per_actor', 'r-1', t0 + interval '11 minute', NULL, pg_temp.decision(uno,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 IF NOT r.reutilizada OR r.registrada_en <> t0 + interval '10 minute' THEN RAISE EXCEPTION 'B24: replay de recurso incorrecto'; END IF;
 PERFORM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(uno, ref_susp, 'desestimado', DATE '2026-09-25', NULL, NULL, 'per_actor', 'r-2', t0 + interval '12 minute', NULL, pg_temp.decision(uno,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(dos, ref_susp, 'interpuesto', DATE '2026-09-24', NULL, NULL, 'per_actor', 'r-3', t0 + interval '13 minute', NULL, pg_temp.decision(dos,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B24: recurso de sanción ajena aceptado';
 EXCEPTION WHEN foreign_key_violation THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(uno, ref_susp, 'interpuesto', DATE '2026-09-01', NULL, NULL, 'per_actor', 'r-4', t0 + interval '13 minute', NULL, pg_temp.decision(uno,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B24: recurso anterior a la notificación aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v1(uno, ref_susp, 'interpuesto', DATE '2026-09-24', 'registro:x', NULL, 'per_actor', 'r-5', t0 + interval '13 minute', NULL, pg_temp.decision(uno,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B24: documento sin huella aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;

 -- Histórico: dos sanciones de «uno», la más reciente primero, con recursos.
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v1(uno,'per_actor',NULL,pg_temp.decision(uno,'per_actor'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 IF filas <> 2 THEN RAISE EXCEPTION 'B24: histórico incompleto (%)', filas; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v1(uno,'per_actor',NULL,pg_temp.decision(uno,'per_actor'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL) WHERE sancion_ref = ref_susp;
 IF jsonb_array_length(r.recursos) <> 2 OR r.recursos->1->>'estado' <> 'desestimado' OR r.suspension_hasta <> DATE '2027-03-20' THEN
  RAISE EXCEPTION 'B24: recursos del histórico incorrectos %', r.recursos;
 END IF;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v1(uno,'per_otro',NULL,pg_temp.decision(uno,'per_actor'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B24: consulta con decisión de otra persona aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;

 -- Solo adición.
 BEGIN
  SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
  UPDATE vec_bolsa_llamamientos.sancion_participacion SET causa = 'x' WHERE sancion_ref = ref_susp;
  RAISE EXCEPTION 'B24: sanción modificada';
 EXCEPTION WHEN others THEN
  IF SQLERRM = 'B24: sanción modificada' THEN RAISE; END IF;
 END;
 RESET ROLE;
END $prueba$;

SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl$ BEGIN
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.sancion_participacion;
  RAISE EXCEPTION 'B24: lectura directa de sanciones permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.recurso_sancion_participacion;
  RAISE EXCEPTION 'B24: lectura directa de recursos permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.consumir_autorizacion_sancion_v1('p','a',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B24: consumo interno invocable';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl$;
RESET ROLE;
DO $acl2$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.listar_sanciones_participacion_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('public','vec_bolsa_llamamientos.registrar_sancion_participacion_v1(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.sancion_participacion'::regclass)
    OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.recurso_sancion_participacion'::regclass) THEN
  RAISE EXCEPTION 'B24: ACL o RLS inesperadas';
 END IF;
END $acl2$;
SELECT 'OK B24 sanciones' AS resultado;
ROLLBACK;
