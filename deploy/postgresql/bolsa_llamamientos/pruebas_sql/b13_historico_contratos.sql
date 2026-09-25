\set ON_ERROR_STOP on
-- B13/CT113 en PG18.4 desechable con la estructura real restaurada (roles,
-- CT75, CT61 y la integración Bolsa 000003-000006 con sus llamamientos) y
-- con CT 000113 y Bolsa 000024 ya instaladas. Todo termina en ROLLBACK.
-- La incorporación se inserta como fixture con session_replication_role
-- (solo superusuario): prueba la proyección, el inbox y las ACL; no acredita
-- la cadena de autorización V3 de la incorporación, que tiene sus pruebas.
BEGIN;
SET LOCAL timezone = 'UTC';
DO $pre$ BEGIN
 IF current_setting('server_version_num') <> '180004'
    OR to_regprocedure('vec_contratacion_temporal.leer_contratos_bolsa_v1(bigint,text,integer)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.contrato_participacion') IS NULL
    OR NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion) THEN
  RAISE EXCEPTION 'B13: requiere PG18.4 con CT113, B13 y al menos una propuesta CT61';
 END IF;
END $pre$;

CREATE TEMP TABLE r (nombre text PRIMARY KEY, valor text);
GRANT ALL ON r TO PUBLIC;

SET LOCAL session_replication_role = replica;
DO $fixture$
DECLARE p record; n integer := 0; ref text; hx text;
BEGIN
 FOR p IN SELECT * FROM vec_contratacion_temporal.propuesta_formalizacion ORDER BY expediente_ref LOOP
  n := n + 1;
  hx := encode(sha256(convert_to('b13:' || p.expediente_ref, 'UTF8')), 'hex');
  ref := 'ref:' || hx;
  INSERT INTO vec_contratacion_temporal.incorporacion_outbox_v2(outbox_ref, recibo_ref, evento_json, estado_sha256, creada_en)
  VALUES ('ref:outbox:' || hx, ref, jsonb_build_object('recibo_ref', ref), repeat('a', 64),
          timestamptz '2027-01-02 09:00:00+00' + n * interval '1 second');
  INSERT INTO vec_contratacion_temporal.incorporacion_registro_v2(
    recibo_ref, seguimiento_ref, organizacion_ref, idempotencia_ref, solicitud_ref, expediente_ref,
    version_expediente, version_anterior, version_resultante, estado_anterior_sha256, estado_resultante_sha256,
    material_json, material_canonico, material_sha256, intencion_canonica, intencion_sha256, exportacion_ct,
    persona_version, perfil_version, recibo_json, auditoria_ref, outbox_ref, registrada_en, evidencia_orden_json)
  VALUES (ref, 'seguimiento:b13:' || n, p.organizacion_ref, 'idem:b13:' || n, 'solicitud:b13:' || n, p.expediente_ref,
    p.version_resultante, 0, 1, repeat('b', 64), repeat('c', 64),
    jsonb_build_object('Confirmacion', jsonb_build_object('PeriodoIncorporacion',
       jsonb_build_object('desde', '2027-01-04T00:00:00Z', 'hasta', '2027-03-31T00:00:00Z'))),
    '\x01'::bytea, encode(sha256('\x01'::bytea), 'hex'), '\x02'::bytea, encode(sha256('\x02'::bytea), 'hex'),
    ARRAY['\x01','\x02','\x03','\x04','\x05','\x06','\x07','\x08']::bytea[], 1, 1,
    jsonb_build_object('MaterialOriginalSHA256', encode(sha256('\x01'::bytea), 'hex'),
      'IntencionSHA256', encode(sha256('\x02'::bytea), 'hex'), 'SeguimientoRef', 'seguimiento:b13:' || n,
      'AuditoriaCTRef', 'ref:auditoria:' || hx, 'OutboxCTRef', 'ref:outbox:' || hx,
      'Transicion', jsonb_build_object('recibo_ref', ref), 'EjercicioSintetico', true,
      'FirmaOficial', false, 'EficaciaAdministrativa', false),
    'ref:auditoria:' || hx, 'ref:outbox:' || hx, timestamptz '2027-01-02 09:00:00+00' + n * interval '1 second', '{}'::jsonb);
 END LOOP;
 INSERT INTO pg_temp.r VALUES ('fixtures', n::text);
END $fixture$;
SET LOCAL session_replication_role = origin;

-- CT: el ejecutor lee; nadie más.
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
INSERT INTO pg_temp.r SELECT 'ct_n', count(*)::text FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 100);
INSERT INTO pg_temp.r SELECT 'ev1', evento::text FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 1);
INSERT INTO pg_temp.r SELECT 'hu1', huella_sha256 FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 1);
INSERT INTO pg_temp.r SELECT 'oc1', origen_creada_en::text FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 1);
INSERT INTO pg_temp.r SELECT 'or1', origen_ref FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 1);
INSERT INTO pg_temp.r SELECT 'po1', origen_posicion::text FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 1);
-- Reentrega: misma fila, misma huella.
INSERT INTO pg_temp.r SELECT 'hu1b', huella_sha256 FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 1);
DO $cursor$ DECLARE v_po bigint; v_or text; v_n integer; BEGIN
 SELECT valor::bigint INTO v_po FROM pg_temp.r WHERE nombre = 'po1';
 SELECT valor INTO v_or FROM pg_temp.r WHERE nombre = 'or1';
 SELECT count(*) INTO v_n FROM vec_contratacion_temporal.leer_contratos_bolsa_v1(v_po, v_or, 100);
 INSERT INTO pg_temp.r VALUES ('ct_tras_cursor', v_n::text);
 BEGIN PERFORM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 0);
  RAISE EXCEPTION 'B13: límite 0 aceptado'; EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN PERFORM vec_contratacion_temporal.leer_contratos_bolsa_v1(v_po, NULL, 10);
  RAISE EXCEPTION 'B13: cursor incompleto aceptado'; EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN PERFORM 1 FROM vec_contratacion_temporal.incorporacion_outbox_v2;
  RAISE EXCEPTION 'B13: lectura directa del outbox CT'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $cursor$;
RESET ROLE;

SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl_ct$ BEGIN
 PERFORM vec_contratacion_temporal.leer_contratos_bolsa_v1(NULL, NULL, 1);
 RAISE EXCEPTION 'B13: Bolsa lee CT directamente';
EXCEPTION WHEN insufficient_privilege THEN NULL;
END $acl_ct$;

-- Bolsa: inbox idempotente.
DO $inbox$
DECLARE v_ev jsonb; v_hu text; v_oc timestamptz; v_po bigint; v_res record; v_malo jsonb; v_desconocido jsonb;
BEGIN
 SELECT valor::jsonb INTO v_ev FROM pg_temp.r WHERE nombre = 'ev1';
 SELECT valor INTO v_hu FROM pg_temp.r WHERE nombre = 'hu1';
 SELECT valor::timestamptz INTO v_oc FROM pg_temp.r WHERE nombre = 'oc1';
 SELECT valor::bigint INTO v_po FROM pg_temp.r WHERE nombre = 'po1';
 SELECT * INTO STRICT v_res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_ev, v_hu, v_oc, v_po);
 IF v_res.reutilizado OR v_res.participacion_ref IS NULL THEN RAISE EXCEPTION 'B13: primera entrega sin participación'; END IF;
 INSERT INTO pg_temp.r VALUES ('participacion', v_res.participacion_ref);
 SELECT * INTO STRICT v_res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_ev, v_hu, v_oc, v_po);
 IF NOT v_res.reutilizado THEN RAISE EXCEPTION 'B13: reentrega no reconocida'; END IF;
 -- La misma entrega con otra posición de publicación: divergente. No se
 -- adopta ni detiene el relevo: queda en cuarentena y se señala.
 SELECT * INTO STRICT v_res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_ev, v_hu, v_oc, v_po + 7);
 IF NOT v_res.en_cuarentena OR v_res.reutilizado OR v_res.participacion_ref IS NOT NULL THEN RAISE EXCEPTION 'B13: posición divergente sin cuarentena'; END IF;
 SELECT * INTO STRICT v_res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_ev, v_hu, v_oc, v_po + 7);
 IF NOT v_res.en_cuarentena THEN RAISE EXCEPTION 'B13: reentrega de la divergencia sin señalar'; END IF;
 BEGIN PERFORM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_ev, v_hu, v_oc, -1);
  RAISE EXCEPTION 'B13: posición negativa aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 -- Mismo evento_ref con otro contenido: cuarentena, nunca sobrescritura.
 v_malo := jsonb_set(v_ev, '{causa_clave}', '"otra_causa"');
 SELECT * INTO STRICT v_res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_malo,
   encode(sha256(convert_to(v_malo::text, 'UTF8')), 'hex'), v_oc, v_po);
 IF NOT v_res.en_cuarentena THEN RAISE EXCEPTION 'B13: contenido divergente sin cuarentena'; END IF;
 -- El original sigue intacto y reconocido como reentrega.
 SELECT * INTO STRICT v_res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_ev, v_hu, v_oc, v_po);
 IF NOT v_res.reutilizado OR v_res.en_cuarentena THEN RAISE EXCEPTION 'B13: el original dejó de reconocerse'; END IF;
 -- Huella que no corresponde al contenido.
 BEGIN PERFORM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_ev, repeat('0', 64), v_oc, v_po);
  RAISE EXCEPTION 'B13: huella falsa aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 -- Campo desconocido, esquema ajeno, evento_ref no derivado y fechas invertidas.
 FOR v_malo IN SELECT x FROM (VALUES (v_ev || '{"dni":"x"}'), (jsonb_set(v_ev, '{esquema}', '"otro"')),
   (jsonb_set(v_ev, '{evento_ref}', to_jsonb('evento:ct:contrato-bolsa:' || repeat('1', 64)))),
   (jsonb_set(v_ev, '{fin_previsto}', '"2026-01-01T00:00:00.000000Z"')),
   (jsonb_set(v_ev, '{inicio}', '"2027-01-04"'))) t(x) LOOP
  BEGIN PERFORM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_malo,
    encode(sha256(convert_to(v_malo::text, 'UTF8')), 'hex'), v_oc, v_po);
   RAISE EXCEPTION 'B13: evento inválido aceptado: %', v_malo;
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 -- Llamamiento que Bolsa no conoce: se conserva sin participación.
 v_desconocido := jsonb_set(jsonb_set(v_ev, '{origen_ref}', '"ref:outbox:desconocido"'), '{llamamiento_ref}', '"llamamiento:ajeno"');
 v_desconocido := jsonb_set(v_desconocido, '{evento_ref}', to_jsonb('evento:ct:contrato-bolsa:' ||
   encode(sha256(convert_to('incorporacion' || chr(31) || 'ref:outbox:desconocido', 'UTF8')), 'hex')));
 SELECT * INTO STRICT v_res FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(v_desconocido,
   encode(sha256(convert_to(v_desconocido::text, 'UTF8')), 'hex'), v_oc + interval '1 hour', v_po + 1);
 IF v_res.reutilizado OR v_res.participacion_ref IS NOT NULL THEN RAISE EXCEPTION 'B13: llamamiento ajeno asignado'; END IF;
 -- El cursor devuelve el último origen recibido.
 IF (SELECT origen_ref FROM vec_bolsa_llamamientos.cursor_contratos_participacion_v1()) <> 'ref:outbox:desconocido' THEN
  RAISE EXCEPTION 'B13: cursor del consumidor incorrecto';
 END IF;
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.contrato_participacion;
  RAISE EXCEPTION 'B13: lectura directa del histórico';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.listar_contratos_participacion_v1('participacion:x', 'persona:x',
   NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
  RAISE EXCEPTION 'B13: consulta sin material V3 permitida';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'B13:%' THEN RAISE; END IF; END;
END $inbox$;
RESET ROLE;

DO $final$
DECLARE v_fila record;
BEGIN
 IF (SELECT valor FROM pg_temp.r WHERE nombre = 'hu1') <> (SELECT valor FROM pg_temp.r WHERE nombre = 'hu1b')
    OR (SELECT valor::integer FROM pg_temp.r WHERE nombre = 'ct_n') <> (SELECT valor::integer FROM pg_temp.r WHERE nombre = 'fixtures')
    OR (SELECT valor::integer FROM pg_temp.r WHERE nombre = 'ct_tras_cursor') <> (SELECT valor::integer FROM pg_temp.r WHERE nombre = 'fixtures') - 1 THEN
  RAISE EXCEPTION 'B13: proyección CT no determinista o cursor erróneo';
 END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.contrato_participacion) <> 2 THEN
  RAISE EXCEPTION 'B13: la reentrega duplicó el histórico';
 END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena) <> 2 THEN
  RAISE EXCEPTION 'B13: la cuarentena no conserva las dos divergencias sin duplicar';
 END IF;
 SELECT * INTO STRICT v_fila FROM vec_bolsa_llamamientos.contrato_participacion
  WHERE participacion_ref = (SELECT valor FROM pg_temp.r WHERE nombre = 'participacion');
 IF v_fila.tipo <> 'incorporacion' OR v_fila.inicio <> timestamptz '2027-01-04 00:00:00+00'
    OR v_fila.fin_previsto <> timestamptz '2027-03-31 00:00:00+00' OR v_fila.modalidad_clave IS NULL
    OR v_fila.categoria_ref IS NULL OR v_fila.causa_clave IS NULL OR v_fila.bolsa_ref IS NULL THEN
  RAISE EXCEPTION 'B13: histórico incompleto: %', row_to_json(v_fila);
 END IF;
 BEGIN
  SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
  UPDATE vec_bolsa_llamamientos.contrato_participacion SET causa_clave = 'x';
  RAISE EXCEPTION 'B13: histórico mutable';
 EXCEPTION WHEN others THEN IF SQLERRM = 'B13: histórico mutable' THEN RAISE; END IF; END;
 RAISE NOTICE 'B13: OK (% incorporaciones proyectadas, reentrega idempotente, negativos y ACL)',
  (SELECT valor FROM pg_temp.r WHERE nombre = 'fixtures');
END $final$;
ROLLBACK;
