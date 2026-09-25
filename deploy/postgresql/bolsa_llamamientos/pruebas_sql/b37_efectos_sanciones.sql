\set ON_ERROR_STOP on
-- Prueba de 000037 (efectos de las sanciones) en una base DESECHABLE con
-- 000026, 000032 y 000037 aplicadas. Se ejecuta como superusuario y termina en
-- ROLLBACK. TEST-ONLY: como b24, sustituye dentro de la transacción el
-- consumidor V3 de la situación por un doble y siembra una bolsa sintética
-- sin disparadores de integridad. Nada persiste.
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
VALUES('acta:b37','bolsa:b37',1,repeat('1',64),'instantanea:b37',1,repeat('2',64),'categoria:b37','sistema:prueba','2026-01-01','2026-01-01');
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
SELECT 'instantanea:b37',1,n,'participacion:b37:' || n,n FROM generate_series(1,5) n;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT 'participacion:b37:' || n,'disponible','2026-01-01','Constitución de bolsa','sistema:constitucion','2026-01-01','constitucion:participacion:b37:' || n,'recibo:situacion:constitucion:participacion:b37:' || n
  FROM generate_series(1,5) n;
INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,vigente_hasta,registrada_en)
VALUES('politica:b37','bolsa:b37',1,'puntuacion_desc_acta','rotatoria','misma_posicion',false,'Prueba B37','sistema:prueba','2026-01-01',NULL,'2026-01-01');
RESET ROLE;
SET LOCAL session_replication_role = origin;

CREATE FUNCTION pg_temp.decision(p text, actor text) RETURNS bytea LANGUAGE sql AS $f$
 SELECT convert_to(jsonb_build_object('principal_id',actor,'accion','bolsa.situacion_participacion.cambiar','modulo_id','bolsa','tipo_recurso','participacion_bolsa',
  'finalidad','gestion_situacion_participacion','recurso_ref',p,'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',repeat('3',64))::text,'UTF8')
$f$;
CREATE FUNCTION pg_temp.ref(p text, clave text) RETURNS text LANGUAGE sql AS $f$
 SELECT encode(sha256(convert_to(p || chr(31) || clave,'UTF8')),'hex')
$f$;
CREATE FUNCTION pg_temp.sancionar(p text, clave text, efecto text, hasta date, al_final boolean, fin boolean, desde timestamptz, regla text DEFAULT 'x')
RETURNS TABLE(reutilizada boolean, sancion_ref text, recibo_ref text, situacion text, desde_out timestamptz, orden_final boolean) LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.registrar_sancion_participacion_v2(
  'bolsa:b37', p, 'sancion:' || pg_temp.ref(p, clave), 'b24.sancion.' || efecto, 'Consecuencia ' || efecto,
  efecto, 'Causa ' || clave, DATE '2026-09-20', 'registro:2026/000123', repeat('a',64), 'persona:jefatura',
  'vec.bolsa.reglas:1:b24.sancion.' || regla, repeat('b',64), hasta, DATE '2026-10-20',
  'vec.bolsa.reglas:1:b24.consecuencias', repeat('b',64),
  CASE WHEN efecto = 'ninguna' THEN NULL ELSE desde END, 'per_actor', clave,
  CASE WHEN efecto = 'ninguna' THEN 'recibo:sancion:' ELSE 'recibo:situacion:' END || pg_temp.ref(p, clave), desde,
  al_final, fin, NULL, pg_temp.decision(p, 'per_actor'), NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)
$f$;
CREATE FUNCTION pg_temp.recurso(p text, sancion text, estado text, clave text, revierte boolean, resuelta text, en timestamptz)
RETURNS TABLE(reutilizada boolean, sancion_ref text, estado text, registrada_en timestamptz, recibo_ref text, situacion text, desde timestamptz) LANGUAGE sql AS $f$
 SELECT * FROM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(
  p, sancion, estado, DATE '2026-09-24', 'registro:2026/000300', repeat('e',64), 'per_actor', clave, en,
  revierte, CASE WHEN revierte THEN resuelta END, CASE WHEN revierte THEN 'vec.bolsa.reglas:1:b24.recurso_revierte' END,
  CASE WHEN revierte THEN repeat('c',64) END, CASE WHEN revierte THEN 'Recurso de reposición estimado' END,
  CASE WHEN revierte THEN 'recibo:readmision:' || encode(sha256(convert_to(sancion || chr(31) || clave,'UTF8')),'hex') END,
  NULL, pg_temp.decision(p,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL)
$f$;
CREATE FUNCTION pg_temp.orden(en timestamptz) RETURNS text LANGUAGE sql AS $f$
 SELECT string_agg(substr(participacion_ref, 19) || ':' || coalesce(orden_vigente::text,'-') || ':' || razon, ',' ORDER BY orden_acta)
   FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:b37', en)
$f$;

DO $prueba$
DECLARE r record; t0 timestamptz := clock_timestamp() + interval '1 minute'; filas bigint; o text;
 p1 text := 'participacion:b37:1'; p2 text := 'participacion:b37:2'; p3 text := 'participacion:b37:3'; p4 text := 'participacion:b37:4';
 ref_final text; ref_final2 text; ref_susp text; ref_baja text; hasta date; fin timestamptz;
BEGIN
 -- Orden de partida: el del acta.
 o := pg_temp.orden(t0);
 IF o <> '1:1:orden_acta,2:2:orden_acta,3:3:orden_acta,4:4:orden_acta,5:5:orden_acta' THEN RAISE EXCEPTION 'B37: orden inicial %', o; END IF;

 -- 1. Pasar al final: 2 y luego 1 quedan tras los no sancionados y conservan su orden relativo.
 SELECT * INTO STRICT r FROM pg_temp.sancionar(p2, 'k-final-2', 'ninguna', NULL, true, false, t0, 'pasar_al_final');
 IF r.reutilizada OR NOT r.orden_final OR r.situacion IS NOT NULL THEN RAISE EXCEPTION 'B37: penalización no aplicada %', r; END IF;
 ref_final := r.sancion_ref;
 o := pg_temp.orden(t0 + interval '1 second');
 IF o <> '1:1:orden_acta,2:5:sancion_al_final,3:2:adelanta_por_sancion,4:3:adelanta_por_sancion,5:4:adelanta_por_sancion' THEN RAISE EXCEPTION 'B37: pasar al final no movió el orden %', o; END IF;
 -- Antes de la sanción, el orden no cambia (historia reproducible).
 IF pg_temp.orden(t0 - interval '1 second') <> '1:1:orden_acta,2:2:orden_acta,3:3:orden_acta,4:4:orden_acta,5:5:orden_acta' THEN RAISE EXCEPTION 'B37: la penalización altera el pasado'; END IF;
 SELECT * INTO STRICT r FROM pg_temp.sancionar(p1, 'k-final-1', 'ninguna', NULL, true, false, t0 + interval '2 second', 'pasar_al_final');
 ref_final2 := r.sancion_ref;
 o := pg_temp.orden(t0 + interval '3 second');
 IF o <> '1:4:sancion_al_final,2:5:sancion_al_final,3:1:adelanta_por_sancion,4:2:adelanta_por_sancion,5:3:adelanta_por_sancion' THEN RAISE EXCEPTION 'B37: orden relativo de penalizados %', o; END IF;
 -- Replay idéntico y replay con otro efecto sobre el orden.
 SELECT * INTO STRICT r FROM pg_temp.sancionar(p2, 'k-final-2', 'ninguna', NULL, true, false, t0, 'pasar_al_final');
 IF NOT r.reutilizada OR r.sancion_ref <> ref_final THEN RAISE EXCEPTION 'B37: replay de penalización %', r; END IF;
 BEGIN
  PERFORM pg_temp.sancionar(p2, 'k-final-2', 'ninguna', NULL, false, false, t0, 'pasar_al_final');
  RAISE EXCEPTION 'B37: replay sin penalización aceptado';
 EXCEPTION WHEN SQLSTATE 'VBS01' THEN NULL; END;
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.penalizacion_orden_sancion;
 IF filas <> 2 THEN RAISE EXCEPTION 'B37: penalizaciones duplicadas (%)', filas; END IF;
 -- Sin penalización (catálogo que no la declara): conducta de 000026.
 SELECT * INTO STRICT r FROM pg_temp.sancionar(p3, 'k-solo-historia', 'ninguna', NULL, false, false, t0 + interval '4 second');
 IF r.orden_final OR pg_temp.orden(t0 + interval '5 second') <> o THEN RAISE EXCEPTION 'B37: sanción sin penalización movió el orden'; END IF;
 -- Una baja no se combina con pasar al final.
 BEGIN
  PERFORM pg_temp.sancionar(p3, 'k-baja-final', 'excluir', NULL, true, false, t0 + interval '5 second');
  RAISE EXCEPTION 'B37: baja con penalización aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;

 -- 2. Suspensión con fin automático: disponible_desde con el día siguiente al último.
 hasta := (t0 AT TIME ZONE 'Europe/Madrid')::date + 180;
 fin := ((hasta + 1)::timestamp AT TIME ZONE 'Europe/Madrid');
 SELECT * INTO STRICT r FROM pg_temp.sancionar(p4, 'k-susp', 'pausar', hasta, false, true, t0 + interval '6 second');
 IF r.reutilizada OR r.situacion <> 'disponible_desde' THEN RAISE EXCEPTION 'B37: suspensión con fin no aplicada %', r; END IF;
 ref_susp := r.sancion_ref;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p4 AND situacion='disponible_desde' AND fecha_disponible=fin)
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p4 AND operacion='pausar' AND justificante_tipo='resolucion' AND validador='persona:jefatura')
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.sancion_participacion WHERE sancion_ref=ref_susp AND efecto='pausar' AND suspension_hasta=hasta) THEN
  RAISE EXCEPTION 'B37: filas de la suspensión incompletas';
 END IF;
 -- 000032: el cambio pasa por la política vigente y guarda su versión.
 IF (SELECT politica_transiciones_version FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p4 AND situacion='disponible_desde') IS DISTINCT FROM 1 THEN
  RAISE EXCEPTION 'B37: la suspensión no anota la versión de política';
 END IF;
 IF (SELECT orden_vigente FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:b37', t0 + interval '7 second') WHERE participacion_ref=p4) IS NOT NULL THEN
  RAISE EXCEPTION 'B37: la suspendida conserva turno';
 END IF;
 -- Al llegar la fecha vuelve sola al turno, sin otra operación.
 IF (SELECT orden_vigente FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:b37', fin) WHERE participacion_ref=p4) IS NULL
    OR (SELECT orden_vigente FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:b37', fin - interval '1 microsecond') WHERE participacion_ref=p4) IS NOT NULL THEN
  RAISE EXCEPTION 'B37: la suspensión no termina sola en su fecha';
 END IF;
 SELECT * INTO STRICT r FROM pg_temp.sancionar(p4, 'k-susp', 'pausar', hasta, false, true, t0 + interval '8 second');
 IF NOT r.reutilizada OR r.sancion_ref <> ref_susp OR r.situacion <> 'disponible_desde' THEN RAISE EXCEPTION 'B37: replay de suspensión %', r; END IF;
 BEGIN
  PERFORM pg_temp.sancionar(p4, 'k-susp', 'pausar', hasta, false, false, t0 + interval '8 second');
  RAISE EXCEPTION 'B37: replay por la vía indefinida aceptado';
 EXCEPTION WHEN SQLSTATE 'VBS01' THEN NULL; END;
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p4;
 IF filas <> 2 THEN RAISE EXCEPTION 'B37: el replay duplicó la situación (%)', filas; END IF;
 -- Fin automático sin plazo, en una baja, ya cumplido o desde una situación no disponible: rechazo.
 BEGIN
  PERFORM pg_temp.sancionar(p3, 'k-susp-sin', 'pausar', NULL, false, true, t0 + interval '9 second');
  RAISE EXCEPTION 'B37: fin automático sin fecha aceptado';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM pg_temp.sancionar(p3, 'k-susp-vieja', 'pausar', DATE '2026-09-21', false, true, t0 + interval '9 second');
  RAISE EXCEPTION 'B37: suspensión cumplida aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM pg_temp.sancionar(p4, 'k-susp-2', 'pausar', hasta, false, true, t0 + interval '9 second');
  RAISE EXCEPTION 'B37: suspensión sobre no disponible aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;

 -- 3. Baja y readmisión por recurso estimado.
 SELECT * INTO STRICT r FROM pg_temp.sancionar(p3, 'k-baja', 'excluir', NULL, false, false, t0 + interval '10 second');
 IF r.situacion <> 'excluido' THEN RAISE EXCEPTION 'B37: baja no aplicada'; END IF;
 ref_baja := r.sancion_ref;
 -- Un estado no revocatorio solo se anota.
 SELECT * INTO STRICT r FROM pg_temp.recurso(p3, ref_baja, 'interpuesto', 'r-int', false, NULL, t0 + interval '11 second');
 IF r.reutilizada OR r.situacion IS NOT NULL THEN RAISE EXCEPTION 'B37: interposición con efectos %', r; END IF;
 -- Readmisión autoresuelta: rechazo antes de efectos.
 BEGIN
  PERFORM pg_temp.recurso(p3, ref_baja, 'estimado', 'r-est-mal', true, 'per_actor', t0 + interval '12 second');
  RAISE EXCEPTION 'B37: readmisión autoresuelta aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 SELECT * INTO STRICT r FROM pg_temp.recurso(p3, ref_baja, 'estimado', 'r-est', true, 'persona:jefatura', t0 + interval '12 second');
 IF r.reutilizada OR r.situacion <> 'disponible' OR r.desde <> t0 + interval '12 second' OR r.recibo_ref !~ '^recibo:readmision:' THEN RAISE EXCEPTION 'B37: readmisión no aplicada %', r; END IF;
 IF (SELECT situacion FROM vec_bolsa_llamamientos.leer_situacion_participacion_v1(p3)) <> 'disponible' THEN RAISE EXCEPTION 'B37: la situación vigente no es la anterior a la baja'; END IF;
 SELECT * INTO STRICT r FROM pg_temp.recurso(p3, ref_baja, 'estimado', 'r-est', true, 'persona:jefatura', t0 + interval '13 second');
 IF NOT r.reutilizada OR r.situacion <> 'disponible' THEN RAISE EXCEPTION 'B37: replay de readmisión %', r; END IF;
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p3;
 IF filas <> 3 THEN RAISE EXCEPTION 'B37: la readmisión duplicó situaciones (%)', filas; END IF;
 -- La readmisión guarda la versión de política y deja en B8 «reactivar»
 -- justificada por la resolución y validada por quien resuelve.
 IF (SELECT politica_transiciones_version FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p3 AND desde=t0 + interval '12 second') IS DISTINCT FROM 1
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p3 AND desde=t0 + interval '12 second'
                    AND operacion='reactivar' AND justificante_tipo='resolucion' AND justificante_ref='registro:2026/000300' AND validador='persona:jefatura' AND actor='per_actor') THEN
  RAISE EXCEPTION 'B37: la readmisión no anota versión ni operación B8';
 END IF;
 -- Una sanción se revierte una sola vez; la clave usada sin reversión no se adopta.
 BEGIN
  PERFORM pg_temp.recurso(p3, ref_baja, 'estimado', 'r-est-2', true, 'persona:jefatura', t0 + interval '14 second');
  RAISE EXCEPTION 'B37: doble reversión aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM pg_temp.recurso(p3, ref_baja, 'interpuesto', 'r-int', true, 'persona:jefatura', t0 + interval '14 second');
  RAISE EXCEPTION 'B37: reversión con clave de otro recurso aceptada';
 EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'VBS01' THEN NULL; END;
 BEGIN
  PERFORM pg_temp.recurso(p3, ref_baja, 'estimado', 'r-est', false, NULL, t0 + interval '14 second');
  RAISE EXCEPTION 'B37: replay sin reversión aceptado';
 EXCEPTION WHEN SQLSTATE 'VBS01' THEN NULL; END;

 -- Recurso estimado contra «pasar al final»: la penalización deja de aplicarse.
 SELECT * INTO STRICT r FROM pg_temp.recurso(p2, ref_final, 'estimado', 'r-final', true, 'persona:jefatura', t0 + interval '15 second');
 IF r.situacion IS NOT NULL THEN RAISE EXCEPTION 'B37: reversión de penalización cambió la situación'; END IF;
 IF (SELECT razon FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:b37', t0 + interval '16 second') WHERE participacion_ref=p2) = 'sancion_al_final'
    OR (SELECT razon FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1('bolsa:b37', t0 + interval '14 second') WHERE participacion_ref=p2) <> 'sancion_al_final' THEN
  RAISE EXCEPTION 'B37: la reversión no levanta la penalización desde su registro';
 END IF;

 -- Recurso estimado contra la suspensión vigente: vuelve a disponible.
 SELECT * INTO STRICT r FROM pg_temp.recurso(p4, ref_susp, 'estimado', 'r-susp', true, 'persona:jefatura', t0 + interval '17 second');
 IF r.situacion <> 'disponible' THEN RAISE EXCEPTION 'B37: la suspensión estimada no se levantó %', r; END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p4 AND operacion='reactivar' AND desde=t0 + interval '17 second') THEN
  RAISE EXCEPTION 'B37: levantar la suspensión no deja «reactivar»';
 END IF;
 -- Revocar exige la huella de la resolución que estima el recurso.
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(
   p1, ref_final2, 'estimado', DATE '2026-09-24', 'registro:2026/000301', NULL, 'per_actor', 'r-sin-huella', t0 + interval '18 second',
   true, 'persona:jefatura', 'vec.bolsa.reglas:1:b24.recurso_revierte', repeat('c',64), 'Recurso de reposición estimado',
   'recibo:readmision:' || encode(sha256(convert_to(ref_final2 || chr(31) || 'r-sin-huella','UTF8')),'hex'),
   NULL, pg_temp.decision(p1,'per_actor'), NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B37: revocación sin huella aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;

 -- Histórico v2 con efecto y reversión.
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v2(p3,'per_actor',NULL,pg_temp.decision(p3,'per_actor'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL) WHERE sancion_ref = ref_baja;
 IF r.situacion_aplicada <> 'excluido' OR r.orden_final OR r.reversion->>'situacion_restaurada' <> 'disponible' OR r.reversion->>'resuelta_por' <> 'persona:jefatura' THEN
  RAISE EXCEPTION 'B37: histórico de la baja %', r;
 END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v2(p4,'per_actor',NULL,pg_temp.decision(p4,'per_actor'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL) WHERE sancion_ref = ref_susp;
 IF r.situacion_aplicada <> 'disponible_desde' OR r.fecha_disponible <> fin THEN RAISE EXCEPTION 'B37: histórico de la suspensión %', r; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v2(p1,'per_actor',NULL,pg_temp.decision(p1,'per_actor'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL) WHERE sancion_ref = ref_final2;
 IF NOT r.orden_final OR r.reversion IS NOT NULL OR r.situacion_aplicada IS NOT NULL THEN RAISE EXCEPTION 'B37: histórico de la penalización %', r; END IF;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.listar_sanciones_participacion_v2(p1,'per_otro',NULL,pg_temp.decision(p1,'per_actor'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B37: consulta con decisión de otra persona aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;

 -- Solo adición.
 BEGIN
  UPDATE vec_bolsa_llamamientos.penalizacion_orden_sancion SET posicion = 'final';
  RAISE EXCEPTION 'B37: penalización modificada';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'B37:%' THEN RAISE; END IF; END;
 BEGIN
  DELETE FROM vec_bolsa_llamamientos.reversion_sancion_participacion;
  RAISE EXCEPTION 'B37: reversión borrada';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'B37:%' THEN RAISE; END IF; END;
END $prueba$;

-- La readmisión de 000032 no es invocable fuera de la función de recurso,
-- ni siquiera por el propietario: exige la marca de transacción que solo pone
-- la función de recurso, con la sanción y la clave exactas. La comprobación no
-- depende del idioma de los mensajes (el ensayo se repite con lc_messages en
-- castellano).
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $readmision$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1('participacion:b37:3','x','r','Motivo','per_actor','readmision:x','recibo:x',clock_timestamp());
  RAISE EXCEPTION 'B37: readmisión directa aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM set_config('vec_bolsa_llamamientos.readmision_recurso', 'otra' || chr(31) || 'r', true);
  PERFORM vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1('participacion:b37:3','x','r','Motivo','per_actor','readmision:x','recibo:x',clock_timestamp());
  RAISE EXCEPTION 'B37: readmisión con marca de otra sanción aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $readmision$;
-- Sin «disponible>disponible_desde» en la política vigente, la suspensión con
-- fin se rechaza: la política de 000032 decide.
SELECT 1 FROM vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1('prueba:b37', repeat('9',64), ARRAY[
   'disponible>no_disponible','disponible>pendiente_incorporacion','disponible>renuncia','disponible>excluido',
   'no_disponible>disponible','no_disponible>excluido',
   'pendiente_incorporacion>trabajando','pendiente_incorporacion>disponible','pendiente_incorporacion>renuncia','pendiente_incorporacion>excluido',
   'trabajando>disponible','trabajando>disponible_desde','trabajando>excluido',
   'disponible_desde>disponible','disponible_desde>excluido','renuncia>disponible','renuncia>excluido']);
RESET ROLE;
DO $politica$ BEGIN
 BEGIN
  PERFORM pg_temp.sancionar('participacion:b37:5', 'k-susp-sin-politica', 'pausar', (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date + 30, false, true, clock_timestamp() + interval '2 minute');
  RAISE EXCEPTION 'B37: suspensión fuera de la política aceptada';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $politica$;

SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl$ BEGIN
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.penalizacion_orden_sancion;
  RAISE EXCEPTION 'B37: lectura directa de penalizaciones permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.reversion_sancion_participacion;
  RAISE EXCEPTION 'B37: lectura directa de reversiones permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.registrar_suspension_con_fin_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B37: suspensión interna invocable';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B37: readmisión invocable por el ejecutor';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- Escribir la marca no basta: el ejecutor no alcanza la función.
 BEGIN
  PERFORM set_config('vec_bolsa_llamamientos.readmision_recurso', 'x' || chr(31) || 'r', true);
  PERFORM 1 FROM vec_bolsa_llamamientos.readmitir_participacion_por_recurso_v1('participacion:b37:3','x','r','Motivo','per_actor','readmision:x','recibo:x',clock_timestamp());
  RAISE EXCEPTION 'B37: readmisión invocable por el ejecutor con marca';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl$;
RESET ROLE;
DO $acl2$ BEGIN
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.registrar_sancion_participacion_v2(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,boolean,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.registrar_recurso_sancion_participacion_v2(text,text,text,date,text,text,text,text,timestamptz,boolean,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.listar_sanciones_participacion_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)','EXECUTE')
    OR has_function_privilege('public','vec_bolsa_llamamientos.registrar_sancion_participacion_v2(text,text,text,text,text,text,text,date,text,text,text,text,text,date,date,text,text,timestamptz,text,text,text,timestamptz,boolean,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_function_privilege('public','vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)','EXECUTE')
    OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.penalizacion_orden_sancion'::regclass)
    OR NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid='vec_bolsa_llamamientos.reversion_sancion_participacion'::regclass) THEN
  RAISE EXCEPTION 'B37: ACL o RLS inesperadas';
 END IF;
END $acl2$;
SELECT 'OK B37 efectos de sanciones' AS resultado;
ROLLBACK;
