\set ON_ERROR_STOP on
-- Prueba funcional de CT-148 (ajustes de reglas) sobre una base con AD3-114 y
-- CT-148 instaladas. Se ejecuta como superusuario en UNA transacción que
-- termina en ROLLBACK: sustituye la fachada AD3 por un doble que devuelve un
-- consumo sintético (la criptografía V3 la prueban Go y AD3), crea un rol de
-- prueba miembro del ejecutor y no deja nada escrito.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';

-- Ejercita la fachada AD3 real antes de sustituirla: una decisión de consulta
-- aplicada a material de ajuste debe denegarse antes de consumir la decisión.
DO $fachada_real$
DECLARE p jsonb:='{"operacion":"ajustar","organizacion_ref":"organizacion:desarrollo:dipgra"}'::jsonb;
 h text; contexto_h text; c jsonb; d jsonb; mensaje text;
BEGIN
 h:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');
 c:=jsonb_build_object('operacion','contratacion_temporal.reglas.consultar_ajustes',
   'audiencia_consumo','vec_contratacion_temporal.ajustes_reglas.v1',
   'efecto_ref','vec.contratacion_temporal.reglas','huella_efecto_sha256',contexto_h);
 d:=jsonb_build_object('accion','contratacion_temporal.reglas.consultar_ajustes',
   'modulo_id','contratacion_temporal','tipo_recurso','catalogo_reglas',
   'finalidad','gobierno_reglas_contratacion_temporal',
   'recurso_ref','vec.contratacion_temporal.reglas',
   'contexto_recurso_huella_sha256',contexto_h,
   'campos_permitidos',jsonb_build_array('historial','vigente'),
   'obligaciones','[]'::jsonb);
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_ct_v3_atestada(
    p,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),
    '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
  RAISE EXCEPTION 'PRUEBA FALLIDA: fachada AD3 aceptó consulta sobre ajuste';
 EXCEPTION WHEN insufficient_privilege THEN
  GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
  IF mensaje<>'AD3-114: ajuste de reglas denegado'
  THEN RAISE EXCEPTION 'PRUEBA FALLIDA: rechazo ajeno a la operación AD3: %',mensaje; END IF;
 END;
END $fachada_real$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_ct_v3_atestada(
 p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:prueba:'||gen_random_uuid()::text,'vec.contratacion_temporal.reglas',repeat('a',64),
  encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),'auditoria:prueba:'||gen_random_uuid()::text,clock_timestamp(),true
$f$;

CREATE ROLE prueba_ct148 NOLOGIN;
GRANT vec_contratacion_temporal_ejecutor TO prueba_ct148;
CREATE ROLE prueba_ct148_ajeno NOLOGIN;

-- Llama a la operación con una decisión sintética del actor dado.
CREATE FUNCTION pg_temp.operar(p jsonb,p_actor text DEFAULT 'per_prueba_rrhh_000000000000000000')
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT vec_contratacion_temporal.operar_ajustes_reglas_v1(p,'\x7b7d'::bytea,
  convert_to(jsonb_build_object('principal_id',p_actor,'accion',CASE WHEN p->>'operacion'='consultar'
    THEN 'contratacion_temporal.reglas.consultar_ajustes' ELSE 'contratacion_temporal.reglas.ajustar' END)::text,'UTF8'),
  '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea)
$f$;
CREATE FUNCTION pg_temp.material(p_clave uuid,p_esperada bigint,p_canonico text,p_cambios jsonb)
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('operacion','ajustar','organizacion_ref','organizacion:desarrollo:dipgra',
  'catalogo_id','vec.contratacion_temporal.reglas.ajustes','clave_idempotencia',p_clave::text,
  'version_esperada',p_esperada,'base_version',1,'base_huella_sha256',repeat('b',64),
  'ajustes_canonico',p_canonico,'ajustes_huella_sha256',encode(sha256(convert_to(p_canonico,'UTF8')),'hex'),
  'cambios',p_cambios,'motivo_clave','acuerdo_rrhh','referencia','Acuerdo 12/2026')
$f$;
-- Espera un SQLSTATE concreto.
CREATE FUNCTION pg_temp.falla(p jsonb,p_estado text,p_nombre text,p_actor text DEFAULT 'per_prueba_rrhh_000000000000000000')
RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 PERFORM pg_temp.operar(p,p_actor);
 RAISE EXCEPTION 'PRUEBA FALLIDA (%): admitido',p_nombre;
EXCEPTION WHEN others THEN
 IF SQLSTATE<>p_estado THEN RAISE EXCEPTION 'PRUEBA FALLIDA (%): % %',p_nombre,SQLSTATE,SQLERRM; END IF;
END $f$;

SET SESSION AUTHORIZATION prueba_ct148_ajeno;
DO $ajeno$
BEGIN
 PERFORM vec_contratacion_temporal.leer_ajustes_reglas_en_v1('vec.contratacion_temporal.reglas.ajustes',now());
 RAISE EXCEPTION 'PRUEBA FALLIDA: lectura sin ser ejecutor';
EXCEPTION WHEN insufficient_privilege THEN NULL;
END $ajeno$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION prueba_ct148;
DO $pruebas$
DECLARE r jsonb; r2 jsonb; k1 uuid:=gen_random_uuid(); k2 uuid:=gen_random_uuid(); t1 timestamptz; n integer;
 c1 text:='{"c03.plazo_fiscalizacion":{"cantidad":"7"}}';
 c2 text:='{"c03.plazo_fiscalizacion":{"cantidad":"6"}}';
 cambio1 jsonb:='[{"regla_clave":"c03.plazo_fiscalizacion","campo":"cantidad","anterior":"10","nuevo":"7"}]';
BEGIN
 SELECT count(*) INTO n FROM vec_contratacion_temporal.leer_ajustes_reglas_en_v1('vec.contratacion_temporal.reglas.ajustes',now());
 IF n<>0 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: sin ajustes debe leer 0 filas'; END IF;

 -- Material inválido: huella, forma, cambio incoherente, versión.
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||jsonb_build_object('ajustes_huella_sha256',repeat('0',64)),'22023','huella');
 -- Con ajustes {} y huella coherente, la guarda de coherencia no detecta la
 -- ausencia: sin esta validación se escribiría una v1 sin fila de cambio.
 PERFORM pg_temp.falla(pg_temp.material(k1,0,'{}',cambio1)-'cambios','22023','cambios ausentes');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,'{}',cambio1)||'{"cambios":null}','22023','cambios nulos');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,'{"c03":{"fases":"x"}}','[{"regla_clave":"c03","campo":"fases","anterior":"a","nuevo":"x"}]'),'22023','campo estructural');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,'[{"regla_clave":"c03.plazo_fiscalizacion","campo":"cantidad","anterior":"10","nuevo":"8"}]'),'22023','cambio incoherente');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,'{"c03.plazo_fiscalizacion":{"cantidad":"7"},"c04.plazo_subsanacion":{"cantidad":"5"}}',cambio1),'22023','ajuste sin cambio');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1||cambio1),'22023','cambio repetido');
 PERFORM pg_temp.falla(pg_temp.material(k1,1,c1,cambio1),'40001','versión esperada adelantada');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||jsonb_build_object('nota','con'||chr(10)||'control'),'22023','nota con control');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||'{"referencia":" Acuerdo"}','22023','referencia con espacios');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||jsonb_build_object('referencia',repeat('a',121)),'22023','referencia larga');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||'{"nota":""}','22023','nota vacía');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||'{"extra":1}','22023','clave desconocida');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||'{"organizacion_ref":"otra"}','22023','otra organización');

 -- Primera versión.
 r:=pg_temp.operar(pg_temp.material(k1,0,c1,cambio1));
 IF (r->>'replay')::boolean OR (r#>>'{recibo,version}')::int<>1 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: v1 %',r; END IF;
 t1:=(r#>>'{recibo,vigente_desde}')::timestamptz;
 -- La misma solicitud conserva version_esperada=0. El material derivado
 -- puede cambiar al recalcular el anterior o al releer el catálogo base.
 r2:=pg_temp.operar(pg_temp.material(k1,0,c1,cambio1));
 IF NOT (r2->>'replay')::boolean OR r2#>>'{recibo,recibo_ref}'<>r#>>'{recibo,recibo_ref}' THEN RAISE EXCEPTION 'PRUEBA FALLIDA: replay %',r2; END IF;
 r2:=pg_temp.operar(pg_temp.material(k1,0,c1,
   '[{"regla_clave":"c03.plazo_fiscalizacion","campo":"cantidad","anterior":"7","nuevo":"7"}]')
   ||jsonb_build_object('base_version',2,'base_huella_sha256',repeat('c',64)));
 IF NOT (r2->>'replay')::boolean OR r2#>>'{recibo,recibo_ref}'<>r#>>'{recibo,recibo_ref}'
    OR r2#>>'{recibo,version}'<>'1'
 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: replay con material recalculado %',r2; END IF;
 -- La misma clave con un valor solicitado distinto no es un replay.
 PERFORM pg_temp.falla(pg_temp.material(k1,0,'{"c03.plazo_fiscalizacion":{"cantidad":"8"}}',
   '[{"regla_clave":"c03.plazo_fiscalizacion","campo":"cantidad","anterior":"10","nuevo":"8"}]'),
   '23505','misma clave, nuevo valor distinto');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1)||'{"referencia":"Otra"}','23505','clave reutilizada');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,cambio1),'23505','misma clave, otro actor','per_otra_persona_000000000000000000');
 -- Otra clave con la versión ya superada.
 PERFORM pg_temp.falla(pg_temp.material(k2,0,c1,cambio1),'40001','versión desfasada');
 -- El anterior de un campo ya ajustado debe coincidir.
 PERFORM pg_temp.falla(pg_temp.material(k2,1,c2,'[{"regla_clave":"c03.plazo_fiscalizacion","campo":"cantidad","anterior":"10","nuevo":"6"}]'),'22023','anterior falso');
 -- Un ajuste previo no puede desaparecer.
 PERFORM pg_temp.falla(pg_temp.material(k2,1,'{"c04.plazo_subsanacion":{"cantidad":"5"}}','[{"regla_clave":"c04.plazo_subsanacion","campo":"cantidad","anterior":"10","nuevo":"5"}]'),'22023','ajuste desaparecido');
 r2:=pg_temp.operar(pg_temp.material(k2,1,c2,'[{"regla_clave":"c03.plazo_fiscalizacion","campo":"cantidad","anterior":"7","nuevo":"6"}]'));
 IF (r2#>>'{recibo,version}')::int<>2 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: v2 %',r2; END IF;

 -- Lectura por instante: antes de v1 nada; entre v1 y v2, v1; ahora, v2.
 SELECT count(*) INTO n FROM vec_contratacion_temporal.leer_ajustes_reglas_en_v1('vec.contratacion_temporal.reglas.ajustes',t1-interval '1 microsecond');
 IF n<>0 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: lectura anterior a v1'; END IF;
 SELECT version INTO n FROM vec_contratacion_temporal.leer_ajustes_reglas_en_v1('vec.contratacion_temporal.reglas.ajustes',t1);
 IF n<>1 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: lectura en v1 da %',n; END IF;
 SELECT version INTO n FROM vec_contratacion_temporal.leer_ajustes_reglas_en_v1('vec.contratacion_temporal.reglas.ajustes',clock_timestamp());
 IF n<>2 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: lectura actual da %',n; END IF;

 -- Historial con quién, antes/después y motivo, paginado.
 r:=pg_temp.operar('{"operacion":"consultar","organizacion_ref":"organizacion:desarrollo:dipgra","catalogo_id":"vec.contratacion_temporal.reglas.ajustes","limite":1}');
 IF jsonb_array_length(r->'historial')<>1 OR NOT (r->>'hay_mas')::boolean
    OR r#>>'{historial,0,version}'<>'2' OR r#>>'{historial,0,cambios,0,anterior}'<>'7'
    OR r#>>'{historial,0,actor_ref}'<>'per_prueba_rrhh_000000000000000000'
    OR r#>>'{vigente,version}'<>'2' OR r#>>'{vigente,ajustes,c03.plazo_fiscalizacion,cantidad}'<>'6'
 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: historial %',r; END IF;
 r:=pg_temp.operar('{"operacion":"consultar","organizacion_ref":"organizacion:desarrollo:dipgra","catalogo_id":"vec.contratacion_temporal.reglas.ajustes","limite":50,"antes_de_version":2}');
 IF jsonb_array_length(r->'historial')<>1 OR (r->>'hay_mas')::boolean OR r#>>'{historial,0,referencia}'<>'Acuerdo 12/2026'
 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: segunda página %',r; END IF;
 PERFORM pg_temp.falla('{"operacion":"consultar","organizacion_ref":"organizacion:desarrollo:dipgra","catalogo_id":"vec.contratacion_temporal.reglas.ajustes","limite":51}','22023','límite');
 PERFORM pg_temp.falla('{"operacion":"consultar","organizacion_ref":"organizacion:desarrollo:dipgra","catalogo_id":"vec.contratacion_temporal.reglas.ajustes","limite":"1"}','22023','límite textual');
END $pruebas$;
RESET SESSION AUTHORIZATION;

-- Historia de solo adición, también para el propietario.
DO $inmutable$
BEGIN
 SET LOCAL ROLE vec_contratacion_temporal_propietario;
 BEGIN
  UPDATE vec_contratacion_temporal.regla_ajuste_version_v1 SET nota='x';
  RAISE EXCEPTION 'PRUEBA FALLIDA: UPDATE admitido';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'PRUEBA FALLIDA%' THEN RAISE; END IF; END;
 BEGIN
  DELETE FROM vec_contratacion_temporal.regla_ajuste_cambio_v1;
  RAISE EXCEPTION 'PRUEBA FALLIDA: DELETE admitido';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'PRUEBA FALLIDA%' THEN RAISE; END IF; END;
 RESET ROLE;
 IF (SELECT count(*) FROM vec_contratacion_temporal.regla_ajuste_outbox_v1)<>2
 THEN RAISE EXCEPTION 'PRUEBA FALLIDA: outbox'; END IF;
END $inmutable$;

SELECT 'CT148-PRUEBAS-OK' AS resultado;
ROLLBACK;
