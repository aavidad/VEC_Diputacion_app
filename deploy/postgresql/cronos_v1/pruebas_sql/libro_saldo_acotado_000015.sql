\set ON_ERROR_STOP on
-- SOLO base nueva vec_crn15_ensayo; aplicar 000001 y 000004 originales antes
-- de este guion. Captura JSON antiguo, instala 000015 y compara sin dobles V3.
DO $guard$
DECLARE f pg_proc%ROWTYPE;
BEGIN
 IF current_database()<>'vec_crn15_ensayo'
    OR to_regprocedure('vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)') IS NULL
 THEN RAISE EXCEPTION 'CRN15 prueba: base desechable o función original ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT f FROM pg_proc
 WHERE oid='vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)'::regprocedure;
 IF strpos(f.prosrc,'WITH seleccion AS (')<>0 THEN
  RAISE EXCEPTION 'CRN15 prueba: la función ya fue sustituida' USING ERRCODE='55000';
 END IF;
END $guard$;
SET timezone='UTC';
INSERT INTO vec_cronos_v1.programacion_jornada
(programacion_ref,empleado_ref,fecha,version,turno_ref,politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
SELECT 'programacion:cronos:base_'||to_char(d,'YYYYMMDD'),
 'emp_aaaaaaaaaaaaaaaaaaaaaa',d,1,'turno:base','politica:jornada:1',
 'fuente:sintetica:crn15','Europe/Madrid',
 CASE WHEN extract(isodow FROM d)<6 THEN 480 ELSE 0 END,
 '2023-12-31T00:00:00Z'::timestamptz
FROM generate_series('2024-01-01'::date,'2024-12-31'::date,interval '1 day') g(d);
INSERT INTO vec_cronos_v1.programacion_jornada
(programacion_ref,empleado_ref,fecha,version,turno_ref,politica_version_ref,fuente_ref,zona_horaria,minutos_previstos,publicada_en)
SELECT 'programacion:cronos:special_'||x.codigo,x.emp,x.fecha,x.version,
 'turno:especial','politica:jornada:1','fuente:sintetica:crn15',x.zona,x.minutos,
 '2025-01-01T00:00:00Z'::timestamptz
FROM (VALUES
 ('night1','emp_nnnnnnnnnnnnnnnnnnnnnn','2025-03-28'::date,1,'Europe/Madrid',120),
 ('night2','emp_nnnnnnnnnnnnnnnnnnnnnn','2025-03-29'::date,1,'Europe/Madrid',360),
 ('dstm','emp_dddddddddddddddddddddd','2025-03-30'::date,1,'Europe/Madrid',120),
 ('dstc','emp_cccccccccccccccccccccc','2025-03-30'::date,1,'Atlantic/Canary',120),
 ('open','emp_oooooooooooooooooooooo','2025-04-01'::date,1,'Europe/Madrid',480),
 ('equal','emp_eeeeeeeeeeeeeeeeeeeeee','2025-04-01'::date,1,'Europe/Madrid',480),
 ('orphan','emp_xxxxxxxxxxxxxxxxxxxxxx','2025-04-02'::date,1,'Europe/Madrid',480),
 ('ancient','emp_hhhhhhhhhhhhhhhhhhhhhh','2025-04-01'::date,1,'Europe/Madrid',480),
 ('ambigv1','emp_vvvvvvvvvvvvvvvvvvvvvv','2025-04-03'::date,1,'Europe/Madrid',480),
 ('ambigv2','emp_vvvvvvvvvvvvvvvvvvvvvv','2025-04-03'::date,2,'Europe/Madrid',450),
 ('antes_dos','emp_pppppppppppppppppppppp','2025-04-01'::date,1,'Europe/Madrid',0),
 ('duracion_24h','emp_bbbbbbbbbbbbbbbbbbbbbb','2025-04-05'::date,1,'Europe/Madrid',720),
 ('sucesor_en_fin','emp_yyyyyyyyyyyyyyyyyyyyyy','2025-04-06'::date,1,'Europe/Madrid',120)
) x(codigo,emp,fecha,version,zona,minutos);
WITH material_src AS (
 SELECT '{"canal":{"politica_version_ref":"politica:canal:1","canal_ref":"canal:terminal:1","origen_ref":"terminal_sintetico","calidad_ref":"calidad:sintetica:1"}}'::text AS material
), filas AS (
 SELECT 'emp_aaaaaaaaaaaaaaaaaaaaaa'::text emp,
        'base_'||to_char(d,'YYYYMMDD')||'_'||t.seq AS clave,
        t.movimiento,(d::timestamp+t.hora) AT TIME ZONE 'Europe/Madrid' AS instante
 FROM generate_series('2024-01-01'::date,'2024-12-31'::date,interval '1 day') g(d)
 CROSS JOIN (VALUES ('1','entrada','08:00'::time),('2','salida','12:00'::time),
                    ('3','entrada','13:00'::time),('4','salida','17:00'::time)) t(seq,movimiento,hora)
 WHERE extract(isodow FROM d)<6
 UNION ALL
 SELECT x.emp,x.clave,x.movimiento,x.hora AT TIME ZONE x.zona
 FROM (VALUES
 ('emp_nnnnnnnnnnnnnnnnnnnnnn','night1','entrada','2025-03-28 22:00'::timestamp,'Europe/Madrid'),
 ('emp_nnnnnnnnnnnnnnnnnnnnnn','night2','salida','2025-03-29 06:00'::timestamp,'Europe/Madrid'),
 ('emp_dddddddddddddddddddddd','dstm1','entrada','2025-03-30 01:30'::timestamp,'Europe/Madrid'),
 ('emp_dddddddddddddddddddddd','dstm2','salida','2025-03-30 04:30'::timestamp,'Europe/Madrid'),
 ('emp_cccccccccccccccccccccc','dstc1','entrada','2025-03-30 00:30'::timestamp,'Atlantic/Canary'),
 ('emp_cccccccccccccccccccccc','dstc2','salida','2025-03-30 03:30'::timestamp,'Atlantic/Canary'),
 ('emp_oooooooooooooooooooooo','open1','entrada','2025-04-01 08:00'::timestamp,'Europe/Madrid'),
 ('emp_eeeeeeeeeeeeeeeeeeeeee','equal1','entrada','2025-04-01 08:00'::timestamp,'Europe/Madrid'),
 ('emp_eeeeeeeeeeeeeeeeeeeeee','equal2','salida','2025-04-01 08:00'::timestamp,'Europe/Madrid'),
 ('emp_xxxxxxxxxxxxxxxxxxxxxx','orphan1','fin_pausa','2025-04-02 08:00'::timestamp,'Europe/Madrid'),
 ('emp_hhhhhhhhhhhhhhhhhhhhhh','ancient1','entrada','2010-01-01 08:00'::timestamp,'Europe/Madrid'),
 ('emp_vvvvvvvvvvvvvvvvvvvvvv','ambig1','entrada','2025-04-03 08:00'::timestamp,'Europe/Madrid'),
 ('emp_vvvvvvvvvvvvvvvvvvvvvv','ambig2','salida','2025-04-03 16:00'::timestamp,'Europe/Madrid'),
 ('emp_pppppppppppppppppppppp','antes1','entrada','2025-03-31 22:00'::timestamp,'Europe/Madrid'),
 ('emp_pppppppppppppppppppppp','antes2','salida','2025-03-31 23:00'::timestamp,'Europe/Madrid'),
 ('emp_bbbbbbbbbbbbbbbbbbbbbb','limite24a','entrada','2025-04-04 12:00'::timestamp,'Europe/Madrid'),
 ('emp_bbbbbbbbbbbbbbbbbbbbbb','limite24b','salida','2025-04-05 12:00'::timestamp,'Europe/Madrid'),
 ('emp_yyyyyyyyyyyyyyyyyyyyyy','sucesor1','entrada','2025-04-06 22:00'::timestamp,'Europe/Madrid'),
 ('emp_yyyyyyyyyyyyyyyyyyyyyy','sucesor2','salida','2025-04-07 00:00'::timestamp,'Europe/Madrid')
 ) x(emp,clave,movimiento,hora,zona)
)
INSERT INTO vec_cronos_v1.marcaje_original
(marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,
 movimiento,instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
SELECT 'marcaje:cronos:'||f.clave,f.emp,f.clave,
 'per_ssssssssssssssssssssss','prf_ssssssssssssssssssssss',m.material,
 encode(sha256(convert_to(m.material,'UTF8')),'hex'),f.movimiento,f.instante,
 'recibo:cronos:'||f.clave,'{}'::jsonb,'auditoria:cronos:'||f.clave,
 'decision:cronos:'||f.clave,encode(sha256(convert_to('consumo:'||f.clave,'UTF8')),'hex'),f.instante
FROM filas f CROSS JOIN material_src m;
ANALYZE vec_cronos_v1.marcaje_original;
ANALYZE vec_cronos_v1.programacion_jornada;
SET ROLE vec_cronos_v1_propietario;
CREATE TEMP TABLE crn15_casos(caso text PRIMARY KEY,empleado text,desde date,hasta date,zona text);
INSERT INTO crn15_casos VALUES
 ('dia','emp_aaaaaaaaaaaaaaaaaaaaaa','2024-06-10','2024-06-10','Europe/Madrid'),
 ('mes','emp_aaaaaaaaaaaaaaaaaaaaaa','2024-06-01','2024-06-30','Europe/Madrid'),
 ('anio','emp_aaaaaaaaaaaaaaaaaaaaaa','2024-01-01','2024-12-31','Europe/Madrid'),
 ('noche','emp_nnnnnnnnnnnnnnnnnnnnnn','2025-03-28','2025-03-29','Europe/Madrid'),
 ('dst_madrid','emp_dddddddddddddddddddddd','2025-03-30','2025-03-30','Europe/Madrid'),
 ('dst_canarias','emp_cccccccccccccccccccccc','2025-03-30','2025-03-30','Atlantic/Canary'),
 ('abierta','emp_oooooooooooooooooooooo','2025-04-01','2025-04-01','Europe/Madrid'),
 ('igual_instante','emp_eeeeeeeeeeeeeeeeeeeeee','2025-04-01','2025-04-01','Europe/Madrid'),
 ('pausa_huerfana','emp_xxxxxxxxxxxxxxxxxxxxxx','2025-04-02','2025-04-02','Europe/Madrid'),
 ('apertura_antigua','emp_hhhhhhhhhhhhhhhhhhhhhh','2025-04-01','2025-04-01','Europe/Madrid'),
 ('programacion_ambigua','emp_vvvvvvvvvvvvvvvvvvvvvv','2025-04-03','2025-04-03','Europe/Madrid'),
 ('dos_anteriores','emp_pppppppppppppppppppppp','2025-04-01','2025-04-01','Europe/Madrid'),
 ('duracion_exacta_24h','emp_bbbbbbbbbbbbbbbbbbbbbb','2025-04-05','2025-04-05','Europe/Madrid'),
 ('sucesor_exacto_fin','emp_yyyyyyyyyyyyyyyyyyyyyy','2025-04-06','2025-04-06','Europe/Madrid');
INSERT INTO crn15_casos
SELECT 'dia_aleatorio_'||to_char(d,'YYYYMMDD'),'emp_aaaaaaaaaaaaaaaaaaaaaa',d,d,'Europe/Madrid'
FROM generate_series('2024-01-03'::date,'2024-12-30'::date,interval '11 day') g(d);
CREATE TEMP TABLE crn15_original(caso text PRIMARY KEY,resultado jsonb NOT NULL);
DO $captura$ DECLARE x record;
BEGIN
 FOR x IN SELECT * FROM crn15_casos ORDER BY caso LOOP
  PERFORM set_config('vec.cronos.empleado_ref',x.empleado,false);
  INSERT INTO crn15_original VALUES(x.caso,
   vec_cronos_v1.consultar_libro_saldo_interno_v1(x.empleado,x.desde,x.hasta,x.zona));
 END LOOP;
END $captura$;
RESET ROLE;
\ir ../migraciones/000015_libro_saldo_acotado.up.sql
SET ROLE vec_cronos_v1_propietario;
DO $comparacion$ DECLARE x record; actual jsonb; contador integer:=0;
BEGIN
 FOR x IN SELECT * FROM crn15_casos ORDER BY caso LOOP
  PERFORM set_config('vec.cronos.empleado_ref',x.empleado,false);
  actual:=vec_cronos_v1.consultar_libro_saldo_interno_v1(x.empleado,x.desde,x.hasta,x.zona);
  IF actual IS DISTINCT FROM (SELECT o.resultado FROM crn15_original o WHERE o.caso=x.caso) THEN
   RAISE EXCEPTION 'CRN15: JSON divergente en caso %',x.caso USING ERRCODE='55000';
  END IF;
  contador:=contador+1;
 END LOOP;
 IF contador<>(SELECT count(*) FROM crn15_original) OR contador<47 THEN
  RAISE EXCEPTION 'CRN15: faltan comparaciones' USING ERRCODE='55000';
 END IF;
END $comparacion$;
SELECT count(*) AS casos_json_iguales FROM crn15_original;
SELECT has_function_privilege('vec_cronos_v1_ejecutor',
 'vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)','EXECUTE') AS runtime_puede_llamar;
DO $negativo$ BEGIN
 PERFORM set_config('vec.cronos.empleado_ref','emp_oooooooooooooooooooooo',false);
 BEGIN
  PERFORM vec_cronos_v1.consultar_libro_saldo_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2024-06-10','2024-06-10','Europe/Madrid');
  RAISE EXCEPTION 'CRN15: empleado ajeno admitido' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE 'PC003' THEN NULL; END;
END $negativo$;
RESET ROLE;
SET ROLE vec_cronos_v1_ejecutor;
DO $runtime$ BEGIN
 BEGIN
  PERFORM vec_cronos_v1.consultar_libro_saldo_interno_v1('emp_aaaaaaaaaaaaaaaaaaaaaa','2024-06-10','2024-06-10','Europe/Madrid');
  RAISE EXCEPTION 'CRN15: EXECUTE runtime directo abierto' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $runtime$;
RESET ROLE;
