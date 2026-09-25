\set ON_ERROR_STOP on
-- Solo para una base PostgreSQL 18 desechable tras UP 000001, 000003, 000004, 000005, 000006.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
INSERT INTO vec_cronos_v1.programacion_jornada
(programacion_ref,empleado_ref,fecha,version,turno_ref,politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
VALUES
('programacion:cronos:uno','emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24',1,'turno:largo','politica:turno:1','fuente:sintetica:1','Europe/Madrid',500,'2026-09-23T00:00:00Z'),
('programacion:cronos:dos','emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-25',1,'turno:descanso','politica:turno:1','fuente:sintetica:1','Europe/Madrid',0,'2026-09-23T00:00:00Z');
INSERT INTO vec_cronos_v1.clasificacion_canal
(politica_version_ref,canal_ref,origen_ref,calidad_ref,tipo_origen,fuente_ref,publicada_en)
VALUES ('politica:canal:1','canal:terminal:1','terminal_sintetico','calidad:sintetica:1',
 'terminal','fuente:sintetica:canal','2026-09-23T00:00:00Z');
INSERT INTO vec_cronos_v1.marcaje_original
(marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,movimiento,
 instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
SELECT 'marcaje:cronos:'||s.clave,'emp_aaaaaaaaaaaaaaaaaaaaaa',s.clave,'per_aaaaaaaaaaaaaaaaaaaaaa',
 'prf_aaaaaaaaaaaaaaaaaaaaaa',s.material,
 encode(sha256(convert_to(s.material,'UTF8')),'hex'),s.movimiento,s.instante,
 'recibo:cronos:'||s.clave,'{}'::jsonb,'auditoria:cronos:'||s.clave,'decision:cronos:'||s.clave,
 encode(sha256(convert_to('consumo:'||s.clave,'UTF8')),'hex'),'2026-09-24T16:00:00Z'
FROM (VALUES
 ('entrada001','entrada','2026-09-24T06:00:00Z'::timestamptz,'{"canal":{"politica_version_ref":"politica:canal:1","canal_ref":"canal:terminal:1","origen_ref":"terminal_sintetico","calidad_ref":"calidad:sintetica:1"}}'),
 ('salida0001','salida','2026-09-24T14:00:00Z'::timestamptz,'{"canal":{"politica_version_ref":"politica:canal:1","canal_ref":"canal:terminal:1","origen_ref":"terminal_sintetico","calidad_ref":"calidad:sintetica:1"}}')
) AS s(clave,movimiento,instante,material);
DO $assert$
DECLARE resultado jsonb;
BEGIN
 resultado:=vec_cronos_v1.consultar_libro_saldo_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24','2026-09-25','Europe/Madrid');
 IF resultado->>'completo'<>'true' OR jsonb_array_length(resultado->'jornadas')<>2
  OR (SELECT sum((e->>'delta_microsegundos')::bigint) FROM jsonb_array_elements(resultado->'movimientos_saldo') e)
     <> -1200000000 THEN
   RAISE EXCEPTION 'libro derivado inesperado: %',resultado;
 END IF;
 IF resultado #>> '{marcajes,0,tipo_origen}' IS DISTINCT FROM 'terminal'
    OR resultado #>> '{marcajes,1,tipo_origen}' IS DISTINCT FROM 'terminal' THEN
   RAISE EXCEPTION 'clasificacion origen inesperada: %',resultado->'marcajes';
 END IF;
 IF has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)','EXECUTE')
    OR has_function_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.teletrabajo_vigente_interno_v1(text,timestamptz)','EXECUTE')
    OR has_table_privilege('vec_cronos_v1_ejecutor','vec_cronos_v1.programacion_jornada','SELECT') THEN
   RAISE EXCEPTION 'ACL Cronos abierta';
 END IF;
END $assert$;
-- Una versión posterior no cambia retroactivamente el saldo ni el recibo.
INSERT INTO vec_cronos_v1.programacion_jornada
(programacion_ref,empleado_ref,fecha,version,turno_ref,politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
VALUES ('programacion:cronos:tres','emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24',2,
 'turno:rectificado','politica:turno:2','fuente:sintetica:rectificacion','Europe/Madrid',450,'2026-09-24T17:00:00Z');
DO $assert$
DECLARE resultado jsonb;
BEGIN
 resultado:=vec_cronos_v1.consultar_libro_saldo_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24','2026-09-25','Europe/Madrid');
 IF resultado->>'completo' IS DISTINCT FROM 'false'
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(resultado->'movimientos_saldo') e
       WHERE e->>'fecha'='2026-09-24' AND e->>'tipo'='previsto')
    OR (SELECT recibo_ref FROM vec_cronos_v1.marcaje_original WHERE marcaje_ref='marcaje:cronos:entrada001')
       IS DISTINCT FROM 'recibo:cronos:entrada001' THEN
   RAISE EXCEPTION 'version posterior altero saldo o recibo: %',resultado;
 END IF;
END $assert$;
ROLLBACK;
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
(autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
VALUES ('teletrabajo:cronos:uno','emp_aaaaaaaaaaaaaaaaaaaaaa',
 tstzrange('2026-09-24T06:00:00Z','2026-09-24T15:00:00Z','[)'),
 'resolucion:sintetica:1','politica:teletrabajo:1','per_bbbbbbbbbbbbbbbbbbbbbb',
 'auditoria:sintetica:1','2026-09-23T00:00:00Z');
DO $assert$
BEGIN
 IF vec_cronos_v1.teletrabajo_vigente_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24T10:00:00Z')
    IS DISTINCT FROM 'teletrabajo:cronos:uno'
    OR vec_cronos_v1.teletrabajo_vigente_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24T15:00:00Z') IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia teletrabajo incorrecta';
 END IF;
END $assert$;
COMMIT;
-- La base de ensayo debe conservar exactamente una autorización tras COMMIT.
SELECT count(*) AS autorizaciones_confirmadas FROM vec_cronos_v1.teletrabajo_autorizacion;
-- 000006: ejercicio de la fachada auditora bajo RLS FORCE, sin DML directo.
RESET ROLE;
CREATE ROLE cronos_ensayo_auditor LOGIN;
GRANT vec_cronos_v1_auditor TO cronos_ensayo_auditor WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SET SESSION AUTHORIZATION cronos_ensayo_auditor;
SELECT vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1(
 'decision:sintetica:2','contexto:sintetico:2','per_aaaaaaaaaaaaaaaaaaaaaa',
 'prf_aaaaaaaaaaaaaaaaaaaaaa','cronos.marcaje.propio.registrar',
 'marcaje:cronos:sintetico','fallo_confirmado','persistencia','2026-09-24T10:00:00Z');
RESET SESSION AUTHORIZATION;
DO $assert$
BEGIN
 IF (SELECT count(*) FROM vec_cronos_v1.resultado_ejecucion_marcaje)<>1
    OR has_table_privilege('cronos_ensayo_auditor','vec_cronos_v1.resultado_ejecucion_marcaje','INSERT') THEN
   RAISE EXCEPTION 'resultado auditor/RLS incorrecto';
 END IF;
END $assert$;
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
DO $assert$
BEGIN
 BEGIN
  INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
   (autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
  VALUES ('teletrabajo:cronos:solapado','emp_aaaaaaaaaaaaaaaaaaaaaa',
   tstzrange('2026-09-24T10:00:00Z','2026-09-24T17:00:00Z','[)'),
   'resolucion:sintetica:2','politica:teletrabajo:1','per_bbbbbbbbbbbbbbbbbbbbbb',
   'auditoria:sintetica:2','2026-09-23T00:00:00Z');
  RAISE EXCEPTION 'solape indebidamente admitido';
 EXCEPTION WHEN SQLSTATE 'PC002' THEN NULL;
 END;
 INSERT INTO vec_cronos_v1.teletrabajo_revocacion
  (revocacion_ref,autorizacion_ref,empleado_ref,efectiva_en,resolucion_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
 VALUES ('revocacion:cronos:uno','teletrabajo:cronos:uno','emp_aaaaaaaaaaaaaaaaaaaaaa',
  '2026-09-24T12:00:00Z','resolucion:sintetica:3','per_bbbbbbbbbbbbbbbbbbbbbb',
  'auditoria:sintetica:3','2026-09-24T12:00:00Z');
 IF vec_cronos_v1.teletrabajo_vigente_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24T12:00:00Z') IS NOT NULL THEN
  RAISE EXCEPTION 'revocacion teletrabajo no aplicada';
 END IF;
 INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
  (autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
 VALUES ('teletrabajo:cronos:sucesora','emp_aaaaaaaaaaaaaaaaaaaaaa',
  tstzrange('2026-09-24T12:00:00Z','2026-09-24T18:00:00Z','[)'),
  'resolucion:sintetica:4','politica:teletrabajo:2','per_bbbbbbbbbbbbbbbbbbbbbb',
  'auditoria:sintetica:4','2026-09-24T12:00:00Z');
 IF vec_cronos_v1.teletrabajo_vigente_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24T12:00:00Z')
    IS DISTINCT FROM 'teletrabajo:cronos:sucesora' THEN
  RAISE EXCEPTION 'periodo sucesor no vigente';
 END IF;
END $assert$;
ROLLBACK;
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
DO $assert$
DECLARE material text:='{"canal":{"origen_ref":"remoto"}}';
BEGIN
 BEGIN
  INSERT INTO vec_cronos_v1.marcaje_original
  (marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,movimiento,
   instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
  VALUES ('marcaje:cronos:remoto_test','emp_aaaaaaaaaaaaaaaaaaaaaa','remoto_test',
   'per_aaaaaaaaaaaaaaaaaaaaaa','prf_aaaaaaaaaaaaaaaaaaaaaa',material,
   encode(sha256(convert_to(material,'UTF8')),'hex'),'entrada','2026-09-24T10:00:00Z',
   'recibo:cronos:remoto_test','{}'::jsonb,'auditoria:cronos:remoto_test',
   'decision:cronos:remoto_test',encode(sha256(convert_to('remoto_test','UTF8')),'hex'),
   '2026-09-24T10:00:00Z');
  RAISE EXCEPTION 'remoto indebidamente admitido';
 EXCEPTION WHEN SQLSTATE 'PC003' THEN NULL;
 END;
END $assert$;
ROLLBACK;
-- Un tramo de más de 24 h iniciado dos días antes del rango no puede
-- desaparecer del libro y dejar `completo=true`.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
INSERT INTO vec_cronos_v1.programacion_jornada
(programacion_ref,empleado_ref,fecha,version,turno_ref,politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
VALUES ('programacion:cronos:larga','emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24',1,
 'turno:largo','politica:turno:1','fuente:sintetica:1','Europe/Madrid',480,'2026-09-23T00:00:00Z');
INSERT INTO vec_cronos_v1.clasificacion_canal
(politica_version_ref,canal_ref,origen_ref,calidad_ref,tipo_origen,fuente_ref,publicada_en)
VALUES ('politica:canal:1','canal:terminal:1','terminal_sintetico','calidad:sintetica:1',
 'terminal','fuente:sintetica:canal','2026-09-23T00:00:00Z');
INSERT INTO vec_cronos_v1.marcaje_original
(marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,movimiento,
 instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
SELECT 'marcaje:cronos:'||s.clave,'emp_aaaaaaaaaaaaaaaaaaaaaa',s.clave,
 'per_aaaaaaaaaaaaaaaaaaaaaa','prf_aaaaaaaaaaaaaaaaaaaaaa',s.material,
 encode(sha256(convert_to(s.material,'UTF8')),'hex'),s.movimiento,s.instante,
 'recibo:cronos:'||s.clave,'{}'::jsonb,'auditoria:cronos:'||s.clave,'decision:cronos:'||s.clave,
 encode(sha256(convert_to('consumo:'||s.clave,'UTF8')),'hex'),clock_timestamp()
FROM (VALUES
 ('largaent1','entrada','2026-09-22T00:00:00Z'::timestamptz,'{"canal":{"politica_version_ref":"politica:canal:1","canal_ref":"canal:terminal:1","origen_ref":"terminal_sintetico","calidad_ref":"calidad:sintetica:1"}}'),
 ('largasal1','salida','2026-09-24T12:00:00Z'::timestamptz,'{"canal":{"politica_version_ref":"politica:canal:1","canal_ref":"canal:terminal:1","origen_ref":"terminal_sintetico","calidad_ref":"calidad:sintetica:1"}}')
) s(clave,movimiento,instante,material);
DO $assert$
DECLARE r jsonb;
BEGIN
 r:=vec_cronos_v1.consultar_libro_saldo_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24','2026-09-24','Europe/Madrid');
 IF r->>'completo' IS DISTINCT FROM 'false'
    OR EXISTS (SELECT 1 FROM jsonb_array_elements(r->'movimientos_saldo') e WHERE e->>'tipo'='trabajado') THEN
  RAISE EXCEPTION 'tramo largo oculto: %',r;
 END IF;
END $assert$;
ROLLBACK;
-- Origen opaco clasificado remoto: tampoco entra por la fachada histórica.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
INSERT INTO vec_cronos_v1.clasificacion_canal
(politica_version_ref,canal_ref,origen_ref,calidad_ref,tipo_origen,fuente_ref,publicada_en)
VALUES ('politica:canal:remota','canal:opaco:7','origen:opaco:7','calidad:remota',
 'remoto','fuente:sintetica:remota','2026-09-23T00:00:00Z');
DO $assert$
DECLARE material text:='{"canal":{"politica_version_ref":"politica:canal:remota","canal_ref":"canal:opaco:7","origen_ref":"origen:opaco:7","calidad_ref":"calidad:remota"}}';
BEGIN
 BEGIN
  INSERT INTO vec_cronos_v1.marcaje_original
  (marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,movimiento,
   instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
  VALUES ('marcaje:cronos:remoto_opaco','emp_aaaaaaaaaaaaaaaaaaaaaa','remoto_opaco',
   'per_aaaaaaaaaaaaaaaaaaaaaa','prf_aaaaaaaaaaaaaaaaaaaaaa',material,
   encode(sha256(convert_to(material,'UTF8')),'hex'),'entrada','2026-09-24T10:00:00Z',
   'recibo:cronos:remoto_opaco','{}'::jsonb,'auditoria:cronos:remoto_opaco',
   'decision:cronos:remoto_opaco',encode(sha256(convert_to('remoto_opaco','UTF8')),'hex'),
   clock_timestamp());
  RAISE EXCEPTION 'remoto opaco indebidamente admitido';
 EXCEPTION WHEN SQLSTATE 'PC003' THEN NULL;
 END;
END $assert$;
ROLLBACK;
-- El bloqueo consultivo solo garantiza la observación tras espera con
-- READ COMMITTED. REPEATABLE READ y SERIALIZABLE se rechazan antes de escribir.
BEGIN ISOLATION LEVEL REPEATABLE READ;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
DO $assert$
BEGIN
 BEGIN
  INSERT INTO vec_cronos_v1.teletrabajo_autorizacion
  (autorizacion_ref,empleado_ref,periodo,resolucion_ref,politica_version_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
  VALUES ('teletrabajo:cronos:rr','emp_aaaaaaaaaaaaaaaaaaaaaa',
   tstzrange('2026-09-25T06:00:00Z','2026-09-25T15:00:00Z','[)'),
   'resolucion:sintetica:rr','politica:teletrabajo:1','per_bbbbbbbbbbbbbbbbbbbbbb',
   'auditoria:sintetica:rr',clock_timestamp());
  RAISE EXCEPTION 'REPEATABLE READ admitió autorización';
 EXCEPTION WHEN SQLSTATE 'PC003' THEN NULL;
 END;
 BEGIN
  INSERT INTO vec_cronos_v1.teletrabajo_revocacion
  (revocacion_ref,autorizacion_ref,empleado_ref,efectiva_en,resolucion_ref,actor_resolutor_ref,auditoria_ref,registrada_en)
  VALUES ('revocacion:cronos:rr','teletrabajo:cronos:uno','emp_aaaaaaaaaaaaaaaaaaaaaa',
   '2026-09-24T12:00:00Z','resolucion:sintetica:revrr','per_bbbbbbbbbbbbbbbbbbbbbb',
   'auditoria:sintetica:revrr',clock_timestamp());
  RAISE EXCEPTION 'REPEATABLE READ admitió revocación';
 EXCEPTION WHEN SQLSTATE 'PC003' THEN NULL;
 END;
END $assert$;
ROLLBACK;
-- Una entrada abierta en A cuatro días antes de B se conserva como hecho
-- relevante y obliga a `completo=false`, aun fuera del margen ordinario.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
INSERT INTO vec_cronos_v1.programacion_jornada
(programacion_ref,empleado_ref,fecha,version,turno_ref,politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
VALUES ('programacion:cronos:periodo_b','emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24',1,
 'turno:ordinario','politica:turno:1','fuente:sintetica:1','Europe/Madrid',480,'2026-09-23T00:00:00Z');
INSERT INTO vec_cronos_v1.clasificacion_canal
(politica_version_ref,canal_ref,origen_ref,calidad_ref,tipo_origen,fuente_ref,publicada_en)
VALUES ('politica:canal:1','canal:terminal:1','terminal_sintetico','calidad:sintetica:1',
 'terminal','fuente:sintetica:canal','2026-09-19T00:00:00Z');
INSERT INTO vec_cronos_v1.marcaje_original
(marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,movimiento,
 instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
SELECT 'marcaje:cronos:entrada_antigua','emp_aaaaaaaaaaaaaaaaaaaaaa','entrada_antigua',
 'per_aaaaaaaaaaaaaaaaaaaaaa','prf_aaaaaaaaaaaaaaaaaaaaaa',m.material,
 encode(sha256(convert_to(m.material,'UTF8')),'hex'),'entrada','2026-09-20T06:00:00Z',
 'recibo:cronos:entrada_antigua','{}'::jsonb,'auditoria:cronos:entrada_antigua',
 'decision:cronos:entrada_antigua',encode(sha256(convert_to('entrada_antigua','UTF8')),'hex'),clock_timestamp()
FROM (VALUES ('{"canal":{"politica_version_ref":"politica:canal:1","canal_ref":"canal:terminal:1","origen_ref":"terminal_sintetico","calidad_ref":"calidad:sintetica:1"}}')) m(material);
DO $assert$
DECLARE r jsonb;
BEGIN
 r:=vec_cronos_v1.consultar_libro_saldo_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2026-09-24','2026-09-24','Europe/Madrid');
 IF r->>'completo' IS DISTINCT FROM 'false'
    OR jsonb_array_length(r->'marcajes')<>1
    OR r #>> '{marcajes,0,marcaje_ref}' IS DISTINCT FROM 'marcaje:cronos:entrada_antigua' THEN
  RAISE EXCEPTION 'entrada antigua abierta quedó oculta: %',r;
 END IF;
END $assert$;
ROLLBACK;
