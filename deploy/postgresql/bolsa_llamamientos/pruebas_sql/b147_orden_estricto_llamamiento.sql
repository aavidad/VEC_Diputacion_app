\set ON_ERROR_STOP on
-- B147 (F1-110): la reserva B7 respeta el orden estricto entre disponibles.
-- Ejecutar como superusuario en PostgreSQL 18 desechable tras B147. Todo se
-- deshace. El consumo V3 se sustituye dentro de la transacción por un doble
-- que acepta la decisión que se le pasa: así se prueba la reserva completa
-- (fila del llamamiento y motivo de salto) sin material firmado.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='60s';

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamptz, consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
 SELECT 'decision:b147:'||encode(sha256(p_decision),'hex'),convert_from(p_decision,'UTF8')::jsonb->>'referencia_recurso',
        repeat('a',64),repeat('b',64),'auditoria:b147:'||encode(sha256(p_decision),'hex'),clock_timestamp(),true
$f$;

-- Reserva como el ejecutor de la aplicación. Devuelve el SQLSTATE y DETAIL
-- del rechazo, o «ok»/«replay» si reserva.
CREATE FUNCTION pg_temp.reservar(p_n int,p_participaciones jsonb,p_salto jsonb,p_necesidad text DEFAULT 'RF') RETURNS text LANGUAGE plpgsql AS $f$
DECLARE v_ref text:=encode(sha256(convert_to('b147:'||p_n,'UTF8')),'hex'); r record; estado text; detalle text;
BEGIN
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.reservar_llamamiento_v2('llamamiento:'||v_ref,'recibo:llamamiento:'||v_ref,'bolsa:b147','per_b147PruebaSinteticaAAAAA','clave-b147-'||p_n,
  p_participaciones,jsonb_build_object('referencia',p_necesidad)||'{"descripcion":"DS","categoria":"CT","centro":"CE","modalidad":"MO","fecha_inicio":"2026-10-01","plazo":"24 horas","plantilla_version":"bolsa-llamamiento-v1","asunto":"AS","cuerpo":"CU"}'::jsonb,
  clock_timestamp(),'\x0101010101010101010101010101010101010101010101010101010101010101','\x00',
  convert_to(jsonb_build_object('principal_id','per_b147PruebaSinteticaAAAAA','accion','llamamiento.emitir.v1','tipo_recurso','bolsa_constituida','referencia_recurso','bolsa:b147','n',p_n)::text,'UTF8'),
  '\x00','\x00',1,1,'\x00','\x00','\x00','\x00',p_salto);
 RETURN CASE WHEN r.reutilizada THEN 'replay' ELSE 'ok' END;
EXCEPTION WHEN OTHERS THEN
 GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE, detalle=PG_EXCEPTION_DETAIL;
 RETURN estado||coalesce(' '||nullif(detalle,''),'');
END $f$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida(bolsa_ref,version,huella_bolsa_sha256,bolsa_canonica,categoria_ref,vigente_desde,estado,registrada_en)
VALUES('bolsa:b147',1,encode(sha256(convert_to('{}','UTF8')),'hex'),convert_to('{}','UTF8'),'categoria:b147','2026-01-01T00:00:00Z','vigente','2026-01-01T00:00:00Z');
INSERT INTO vec_bolsa_llamamientos.instantanea_orden_bolsa(instantanea_ref,version,huella_instantanea_sha256,instantanea_canonica,bolsa_ref,version_bolsa,huella_bolsa_sha256,total_participaciones,referida_en,generada_en,registrada_en)
SELECT 'instantanea:b147',1,encode(sha256(convert_to('{}','UTF8')),'hex'),convert_to('{}','UTF8'),'bolsa:b147',1,huella_bolsa_sha256,3,registrada_en,registrada_en,registrada_en
  FROM vec_bolsa_llamamientos.bolsa_constituida WHERE bolsa_ref='bolsa:b147';
INSERT INTO vec_bolsa_llamamientos.constitucion(acta_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,instantanea_ref,version_instantanea,huella_instantanea_sha256,categoria_ref,actor_ref,confirmada_en,registrada_en)
SELECT 'acta:b147','bolsa:b147',1,b.huella_bolsa_sha256,'instantanea:b147',1,i.huella_instantanea_sha256,'categoria:b147','actor:b147',b.registrada_en,b.registrada_en
  FROM vec_bolsa_llamamientos.bolsa_constituida b JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa i USING(bolsa_ref) WHERE b.bolsa_ref='bolsa:b147';
INSERT INTO vec_bolsa_llamamientos.constitucion_entrada(instantanea_ref,version_instantanea,orden,participacion_ref,fila_numero)
SELECT 'instantanea:b147',1,n,'participacion:b147:'||n,n+1 FROM generate_series(1,3) n;
-- La constitución deja a las tres «disponible» (iniciar_situacion_participacion_v1).

SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;

DO $prueba$
DECLARE r text;
 salto jsonb:='{"clave":"b01.salto.prueba","regla_ref":"vec.bolsa.reglas:9:b01.salto.prueba","huella_catalogo":"'||repeat('c',64)||'"}';
BEGIN
 -- Orden 3 con el 2 (y el 1) disponibles y sin llamar: rechazo sin efectos.
 r:=pg_temp.reservar(1,'["participacion:b147:3"]',NULL);
 IF r<>'VBL02 {"orden_vigente": 1, "participacion_ref": "participacion:b147:1"}' THEN RAISE EXCEPTION 'B147: salto aceptado o mal explicado: %',r; END IF;
 -- Orden 1: correcto.
 r:=pg_temp.reservar(2,'["participacion:b147:1"]',NULL);
 IF r<>'ok' THEN RAISE EXCEPTION 'B147: el primero en turno no se reserva: %',r; END IF;
 -- El 1 ya está llamado: el 3 con el 2 disponible sigue fuera de orden.
 r:=pg_temp.reservar(3,'["participacion:b147:3"]',NULL);
 IF r<>'VBL02 {"orden_vigente": 2, "participacion_ref": "participacion:b147:2"}' THEN RAISE EXCEPTION 'B147: orden 3 con el 2 disponible: %',r; END IF;
 -- Varias personas con hueco interno: 1 y 3 sin el 2.
 r:=pg_temp.reservar(4,'["participacion:b147:1","participacion:b147:3"]',NULL);
 IF r NOT LIKE 'VBL02 %participacion:b147:2%' THEN RAISE EXCEPTION 'B147: hueco interno admitido: %',r; END IF;
 -- Un motivo sin salto se rechaza; uno mal formado también.
 r:=pg_temp.reservar(5,'["participacion:b147:2"]',salto);
 IF r<>'22023' THEN RAISE EXCEPTION 'B147: motivo sin salto admitido: %',r; END IF;
 r:=pg_temp.reservar(5,'["participacion:b147:3"]',jsonb_set(salto,'{clave}','"b24.sancion.x"'));
 IF r<>'22023' THEN RAISE EXCEPTION 'B147: motivo ajeno admitido: %',r; END IF;
END $prueba$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $efectos$
DECLARE filas bigint;
BEGIN
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref='bolsa:b147';
 IF filas<>1 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.salto_orden_llamamiento) THEN
  RAISE EXCEPTION 'B147: un rechazo dejó efectos (% llamamientos)',filas; END IF;
END $efectos$;

-- El 2 deja de estar disponible: el 3 pasa a ser el siguiente.
SAVEPOINT no_disponible;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:b147:2','no_disponible',clock_timestamp(),'Pausa B147','actor:b147',clock_timestamp(),'b147:pausa:2','recibo:b147:pausa:2');
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $prueba$
DECLARE r text;
BEGIN
 r:=pg_temp.reservar(7,'["participacion:b147:3"]',NULL);
 IF r<>'ok' THEN RAISE EXCEPTION 'B147: orden 3 con el 2 no disponible: %',r; END IF;
END $prueba$;
ROLLBACK TO SAVEPOINT no_disponible;
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $prueba$
DECLARE r text;
BEGIN
 -- Orden 2: correcto, y repetirlo devuelve el mismo llamamiento.
 r:=pg_temp.reservar(6,'["participacion:b147:2"]',NULL);
 IF r<>'ok' THEN RAISE EXCEPTION 'B147: orden 2 rechazado: %',r; END IF;
 r:=pg_temp.reservar(6,'["participacion:b147:2"]',NULL);
 IF r<>'replay' THEN RAISE EXCEPTION 'B147: repetición no reutilizada: %',r; END IF;
 -- Para otra necesidad, el llamamiento anterior sin respuesta no aparta al 1.
 r:=pg_temp.reservar(10,'["participacion:b147:3"]',NULL,'OTRA');
 IF r<>'VBL02 {"orden_vigente": 1, "participacion_ref": "participacion:b147:1"}' THEN RAISE EXCEPTION 'B147: otra necesidad salta al ya llamado: %',r; END IF;
END $prueba$;

-- RRHH registra una situación del 1 después de su llamamiento: vuelve a
-- contar, y un salto solo pasa con motivo, que queda guardado.
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
VALUES('participacion:b147:1','disponible',clock_timestamp(),'Sin respuesta B147','actor:b147',clock_timestamp(),'b147:vuelve:1','recibo:b147:vuelve:1');
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $prueba$
DECLARE r text;
 salto jsonb:='{"clave":"b01.salto.prueba","regla_ref":"vec.bolsa.reglas:9:b01.salto.prueba","huella_catalogo":"'||repeat('c',64)||'"}';
BEGIN
 r:=pg_temp.reservar(8,'["participacion:b147:3"]',NULL);
 IF r<>'VBL02 {"orden_vigente": 1, "participacion_ref": "participacion:b147:1"}' THEN RAISE EXCEPTION 'B147: la situación nueva no devuelve el turno: %',r; END IF;
 r:=pg_temp.reservar(9,'["participacion:b147:3"]',salto);
 IF r<>'ok' THEN RAISE EXCEPTION 'B147: salto con motivo rechazado: %',r; END IF;
 r:=pg_temp.reservar(9,'["participacion:b147:3"]',salto);
 IF r<>'replay' THEN RAISE EXCEPTION 'B147: repetición con motivo: %',r; END IF;
 r:=pg_temp.reservar(9,'["participacion:b147:3"]',NULL);
 IF r<>'VBE01' THEN RAISE EXCEPTION 'B147: repetición sin el motivo original: %',r; END IF;
 -- El ejecutor no lee la tabla ni la función interna.
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.salto_orden_llamamiento; RAISE EXCEPTION 'B147: tabla de saltos visible';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_bolsa_llamamientos.saltos_orden_llamamiento_v1('bolsa:b147','[]',clock_timestamp(),'RF'); RAISE EXCEPTION 'B147: función interna ejecutable';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $prueba$;

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $guardado$
DECLARE s record;
BEGIN
 SELECT * INTO STRICT s FROM vec_bolsa_llamamientos.salto_orden_llamamiento
  WHERE llamamiento_ref='llamamiento:'||encode(sha256(convert_to('b147:9','UTF8')),'hex');
 IF s.motivo_clave<>'b01.salto.prueba' OR s.regla_ref<>'vec.bolsa.reglas:9:b01.salto.prueba'
 OR s.saltadas<>'[{"orden_vigente": 1, "participacion_ref": "participacion:b147:1"}]'::jsonb
 OR s.decision_ref NOT LIKE 'decision:b147:%' OR s.auditoria_ref NOT LIKE 'auditoria:b147:%'
 THEN RAISE EXCEPTION 'B147: motivo mal guardado: %',to_jsonb(s); END IF;
 BEGIN UPDATE vec_bolsa_llamamientos.salto_orden_llamamiento SET motivo_clave='b01.salto.otro'; RAISE EXCEPTION 'B147: motivo mutable';
 EXCEPTION WHEN OTHERS THEN IF SQLERRM LIKE 'B147%' THEN RAISE; END IF; END;
END $guardado$;
ROLLBACK;
