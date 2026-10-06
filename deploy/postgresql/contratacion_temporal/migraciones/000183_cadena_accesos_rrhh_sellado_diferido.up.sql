\set ON_ERROR_STOP on
-- CT183: la cadena de accesos RRHH de Contratación temporal pasa al sellado
-- diferido de AD207 (mismo patrón, tablas y sellador propios de CT).
--
-- Antes, cada consulta RRHH bloqueaba con FOR UPDATE la única fila de
-- control_cadena_accesos_rrhh hasta el COMMIT (y con lock_timeout=1s): las
-- consultas simultáneas se esperaban unas a otras y chocaban en SERIALIZABLE.
--
-- Ahora registrar_acceso_rrhh_interno_v1/v2:
-- 1. toman el número de una secuencia y guardan 64 «f» como anterior; la
--    huella sigue siendo sha256(anterior || prueba_canonica), así que liga el
--    número y todo el contenido;
-- 2. un disparador encola el acceso y rechaza el que no lleve el marcador o
--    llegue con el sellado detenido más allá del plazo (10 s);
-- 3. sellar_cadena_accesos_rrhh_v1 (ejecutable solo por vec_auditoria_encadenador,
--    el sellador de AD207) da posición contigua por lotes y escribe:
--      eslabon(p) = sha256(F('vec.ct.acceso_rrhh.eslabon.v1') F(p) F(eslabon(p-1))
--                          F(secuencia) F(acceso_ref) F(huella_sha256)
--                          F(registrada_en) F(sellado_en))
--    con F = encuadrar_texto_v1 y fechas UTC con microsegundos;
-- 4. control_cadena_accesos_rrhh queda congelada como corte de la cadena
--    anterior, que no cambia; eslabon(N) del corte es su cabeza.
-- La comprobación de la base del registrador v2 se conserva; su fila es
-- inmutable, así que deja de bloquearse con FOR SHARE.
-- Depende de AD207 (grupo vec_auditoria_encadenador).
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='300s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000183',0));
DO $pre$
DECLARE nombre text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regrole('vec_contratacion_temporal_propietario') IS NULL
 OR to_regrole('vec_auditoria_encadenador') IS NULL
 THEN RAISE EXCEPTION 'CT183: PARO clave=base esperado=PG18_y_AD207 actual=incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['control_cadena_accesos_rrhh','registro_acceso_rrhh','control_registrador_acceso_rrhh_v2'] LOOP
  IF to_regclass('vec_contratacion_temporal.'||nombre) IS NULL
  THEN RAISE EXCEPTION 'CT183: PARO clave=tabla_% esperado=presente actual=ausente',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['secuencia_acceso_rrhh_v1','pendiente_sellado_acceso_rrhh_v1','eslabon_acceso_rrhh_v1','sellado_acceso_rrhh_v1'] LOOP
  IF to_regclass('vec_contratacion_temporal.'||nombre) IS NOT NULL
  THEN RAISE EXCEPTION 'CT183: PARO clave=objeto_% esperado=ausente actual=presente',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
LOCK TABLE vec_contratacion_temporal.control_cadena_accesos_rrhh,
 vec_contratacion_temporal.registro_acceso_rrhh IN ACCESS EXCLUSIVE MODE;

-- La cadena anterior debe estar íntegra: numeración, enlaces, huella de cada
-- acceso recalculada desde su prueba y cabeza del control.
DO $corte$
DECLARE v_n numeric;v_h text;v_cuenta numeric;v_max numeric;v_rotos bigint;v_ultima text;
BEGIN
 SELECT ultima_secuencia,cabeza_sha256 INTO STRICT v_n,v_h FROM vec_contratacion_temporal.control_cadena_accesos_rrhh WHERE control;
 SELECT count(*),coalesce(max(secuencia),0),
  count(*) FILTER (WHERE anterior_sha256 IS DISTINCT FROM coalesce(previa,pg_catalog.repeat('0',64)) OR secuencia<>fila
   OR huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(anterior_sha256,'hex')||prueba_canonica),'hex')
   OR anterior_sha256=pg_catalog.repeat('f',64)),
  (SELECT a.huella_sha256 FROM vec_contratacion_temporal.registro_acceso_rrhh a ORDER BY a.secuencia DESC LIMIT 1)
 INTO STRICT v_cuenta,v_max,v_rotos,v_ultima
 FROM (SELECT secuencia,anterior_sha256,huella_sha256,prueba_canonica,lag(huella_sha256) OVER (ORDER BY secuencia) previa,
   row_number() OVER (ORDER BY secuencia) fila FROM vec_contratacion_temporal.registro_acceso_rrhh) x;
 IF v_cuenta<>v_n OR v_max<>v_n OR v_rotos<>0 OR v_h IS DISTINCT FROM coalesce(v_ultima,pg_catalog.repeat('0',64))
 THEN RAISE EXCEPTION 'CT183: PARO clave=cadena_anterior esperado=contigua_enlazada_y_cabeza actual=cuenta_%_max_%_rotos_%',v_cuenta,v_max,v_rotos USING ERRCODE='55000'; END IF;
 EXECUTE format('CREATE SEQUENCE vec_contratacion_temporal.secuencia_acceso_rrhh_v1 AS bigint MINVALUE 1 MAXVALUE 9007199254740991 START WITH %s NO CYCLE',v_n+1);
END $corte$;

CREATE TABLE vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 (
 secuencia numeric(20,0) PRIMARY KEY);
CREATE TABLE vec_contratacion_temporal.eslabon_acceso_rrhh_v1 (
 posicion numeric(20,0) PRIMARY KEY CHECK (posicion BETWEEN 1 AND 9007199254740991),
 secuencia numeric(20,0) NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.registro_acceso_rrhh(secuencia),
 anterior_sha256 text NOT NULL CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
 eslabon_sha256 text NOT NULL UNIQUE CHECK (eslabon_sha256 ~ '^[0-9a-f]{64}$'),
 sellado_en timestamptz(6) NOT NULL);
CREATE TABLE vec_contratacion_temporal.sellado_acceso_rrhh_v1 (
 control boolean PRIMARY KEY DEFAULT true CHECK (control),
 latido timestamptz(6) NOT NULL,
 plazo_maximo_segundos integer NOT NULL CHECK (plazo_maximo_segundos BETWEEN 1 AND 300));
INSERT INTO vec_contratacion_temporal.sellado_acceso_rrhh_v1 VALUES (true,clock_timestamp(),10);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['pendiente_sellado_acceso_rrhh_v1','eslabon_acceso_rrhh_v1','sellado_acceso_rrhh_v1'] LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',t);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_total ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',t);
 END LOOP;
END $tablas$;
CREATE TRIGGER eslabon_acceso_rrhh_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.eslabon_acceso_rrhh_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER eslabon_acceso_rrhh_v1_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.eslabon_acceso_rrhh_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
-- Solo el latido cambia; el plazo se cambia con otra migración.
CREATE FUNCTION vec_contratacion_temporal.solo_latido_acceso_rrhh_v1() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp
AS $f$ BEGIN
 IF NEW.plazo_maximo_segundos IS DISTINCT FROM OLD.plazo_maximo_segundos OR NEW.control IS DISTINCT FROM OLD.control THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='plazo de accesos RRHH inmutable';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.solo_latido_acceso_rrhh_v1() FROM PUBLIC;
CREATE TRIGGER sellado_acceso_rrhh_v1_solo_latido BEFORE UPDATE ON vec_contratacion_temporal.sellado_acceso_rrhh_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.solo_latido_acceso_rrhh_v1();
CREATE TRIGGER sellado_acceso_rrhh_v1_no_borrar BEFORE DELETE ON vec_contratacion_temporal.sellado_acceso_rrhh_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER sellado_acceso_rrhh_v1_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.sellado_acceso_rrhh_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
-- El control queda congelado como corte de la cadena anterior.
CREATE TRIGGER control_cadena_accesos_rrhh_congelada_ct183 BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.control_cadena_accesos_rrhh
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER control_cadena_accesos_rrhh_no_truncar_ct183 BEFORE TRUNCATE ON vec_contratacion_temporal.control_cadena_accesos_rrhh
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.reservar_acceso_rrhh_v1()
 RETURNS TABLE(secuencia numeric, anterior_sha256 text)
 LANGUAGE sql VOLATILE SET search_path=pg_catalog,pg_temp
AS $f$ SELECT pg_catalog.nextval('vec_contratacion_temporal.secuencia_acceso_rrhh_v1'::pg_catalog.regclass)::numeric,pg_catalog.repeat('f',64) $f$;

CREATE FUNCTION vec_contratacion_temporal.encolar_acceso_rrhh_v1()
 RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp
AS $f$
DECLARE v_corte numeric;v_latido timestamptz;v_plazo integer;v_ahora timestamptz;
BEGIN
 SELECT c.ultima_secuencia INTO STRICT v_corte FROM vec_contratacion_temporal.control_cadena_accesos_rrhh c WHERE c.control;
 IF NEW.secuencia<=v_corte OR NEW.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f',64) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='acceso RRHH fuera de la cadena con sellado diferido';
 END IF;
 SELECT s.latido,s.plazo_maximo_segundos INTO STRICT v_latido,v_plazo FROM vec_contratacion_temporal.sellado_acceso_rrhh_v1 s;
 -- En REPEATABLE READ y SERIALIZABLE el latido se lee con la instantánea de la
 -- transacción: se compara con su inicio (now()) para no rechazar
 -- transacciones largas legítimas, cuya duración acota transaction_timeout en
 -- los LOGIN de la aplicación. En READ COMMITTED se lee el actual.
 v_ahora:=(CASE WHEN current_setting('transaction_isolation') IN ('repeatable read','serializable')
  THEN pg_catalog.now() ELSE pg_catalog.clock_timestamp() END);
 IF v_ahora-v_latido>pg_catalog.make_interval(secs=>v_plazo) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='sellado de accesos RRHH detenido: acceso rechazado';
 END IF;
 INSERT INTO vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1(secuencia) VALUES (NEW.secuencia);
 RETURN NULL;
END $f$;
CREATE TRIGGER encolar_sellado_ct183 AFTER INSERT ON vec_contratacion_temporal.registro_acceso_rrhh
 FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.encolar_acceso_rrhh_v1();

CREATE FUNCTION vec_contratacion_temporal.eslabon_acceso_rrhh_v1(p_posicion numeric,p_anterior text,p_secuencia numeric,
 p_acceso_ref text,p_huella text,p_registrada_en timestamptz,p_sellado_en timestamptz)
 RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp
AS $f$ SELECT pg_catalog.encode(pg_catalog.sha256(
 vec_contratacion_temporal.encuadrar_texto_v1('vec.ct.acceso_rrhh.eslabon.v1')||
 vec_contratacion_temporal.encuadrar_texto_v1(p_posicion::text)||vec_contratacion_temporal.encuadrar_texto_v1(p_anterior)||
 vec_contratacion_temporal.encuadrar_texto_v1(p_secuencia::text)||vec_contratacion_temporal.encuadrar_texto_v1(p_acceso_ref)||
 vec_contratacion_temporal.encuadrar_texto_v1(p_huella)||
 vec_contratacion_temporal.encuadrar_texto_v1(pg_catalog.to_char(p_registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))||
 vec_contratacion_temporal.encuadrar_texto_v1(pg_catalog.to_char(p_sellado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex') $f$;

-- Sella hasta p_max accesos pendientes en orden de número; READ COMMITTED y
-- un solo sellador a la vez, como en AD207.
CREATE FUNCTION vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1(p_max integer)
 RETURNS integer LANGUAGE plpgsql SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET TimeZone='UTC'
AS $f$
DECLARE v_pos numeric;v_ant text;r record;n integer;v_borrados integer;v_plazo integer;v_antigua timestamptz;
 v_sellado timestamptz(6);i integer;
 l_ref text[]:='{}';l_huella text[]:='{}';l_fecha timestamptz[]:='{}';
 a_pos numeric[]:='{}';a_sec numeric[]:='{}';a_ant text[]:='{}';a_esl text[]:='{}';
BEGIN
 IF p_max IS NULL OR p_max NOT BETWEEN 1 AND 50000 THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='sellado de accesos RRHH: lote inválido';
 END IF;
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION USING ERRCODE='25000',MESSAGE='sellado de accesos RRHH: transacción no admitida';
 END IF;
 IF NOT pg_catalog.pg_try_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:sellado_accesos_rrhh',0)) THEN
  RETURN 0;
 END IF;
 SELECT e.posicion,e.eslabon_sha256 INTO v_pos,v_ant FROM vec_contratacion_temporal.eslabon_acceso_rrhh_v1 e ORDER BY e.posicion DESC LIMIT 1;
 IF v_pos IS NULL THEN
  SELECT c.ultima_secuencia,c.cabeza_sha256 INTO STRICT v_pos,v_ant FROM vec_contratacion_temporal.control_cadena_accesos_rrhh c WHERE c.control;
 END IF;
 FOR r IN SELECT a.secuencia,a.acceso_ref,a.huella_sha256,a.anterior_sha256,a.registrada_en
   FROM (SELECT q.secuencia FROM vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 q ORDER BY q.secuencia LIMIT p_max) p
   CROSS JOIN LATERAL (SELECT x.* FROM vec_contratacion_temporal.registro_acceso_rrhh x WHERE x.secuencia=p.secuencia) a
   ORDER BY p.secuencia
 LOOP
  IF r.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f',64) THEN
   RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='acceso RRHH pendiente sin marcador';
  END IF;
  a_sec:=a_sec||r.secuencia;l_ref:=l_ref||r.acceso_ref;l_huella:=l_huella||r.huella_sha256;l_fecha:=l_fecha||r.registrada_en;
 END LOOP;
 n:=pg_catalog.cardinality(a_sec);
 IF n>0 THEN
  -- Hora de sellado tomada tras leer el lote: sellado_en >= registrada_en.
  v_sellado:=pg_catalog.clock_timestamp();
  FOR i IN 1..n LOOP
   a_ant:=a_ant||v_ant;
   v_pos:=v_pos+1;
   v_ant:=vec_contratacion_temporal.eslabon_acceso_rrhh_v1(v_pos,v_ant,a_sec[i],l_ref[i],l_huella[i],l_fecha[i],v_sellado);
   a_pos:=a_pos||v_pos;a_esl:=a_esl||v_ant;
  END LOOP;
  INSERT INTO vec_contratacion_temporal.eslabon_acceso_rrhh_v1(posicion,secuencia,anterior_sha256,eslabon_sha256,sellado_en)
   SELECT u.p,u.s,u.a,u.e,v_sellado FROM unnest(a_pos,a_sec,a_ant,a_esl) AS u(p,s,a,e);
  DELETE FROM vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 q WHERE q.secuencia=ANY(a_sec);
  GET DIAGNOSTICS v_borrados = ROW_COUNT;
  IF v_borrados<>n THEN
   RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='sellado de accesos RRHH: cola incoherente';
  END IF;
 END IF;
 -- El latido solo avanza si no queda en cola nada más antiguo que el plazo.
 SELECT s.plazo_maximo_segundos INTO STRICT v_plazo FROM vec_contratacion_temporal.sellado_acceso_rrhh_v1 s;
 SELECT a.registrada_en INTO v_antigua FROM vec_contratacion_temporal.registro_acceso_rrhh a
  WHERE a.secuencia=(SELECT min(q.secuencia) FROM vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 q);
 IF v_antigua IS NULL OR pg_catalog.clock_timestamp()-v_antigua<=pg_catalog.make_interval(secs=>v_plazo) THEN
  UPDATE vec_contratacion_temporal.sellado_acceso_rrhh_v1 SET latido=pg_catalog.clock_timestamp() WHERE control;
 END IF;
 RETURN n;
END $f$;

-- Verificación de extremo a extremo (propietario o DBA): recalcula la huella
-- de cada acceso desde su prueba, los enlaces hasta el corte y cada eslabón.
CREATE FUNCTION vec_contratacion_temporal.verificar_cadena_accesos_rrhh_v1()
 RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp SET row_security=on SET TimeZone='UTC'
AS $f$
DECLARE v_corte numeric;v_cabeza text;v_cuenta numeric;v_max numeric;v_rotos bigint;v_ultima text;
 v_pos numeric;v_ant text;v_calc text;r record;v_sellados bigint:=0;v_pend bigint;v_huerf bigint;v_doble bigint;
 v_antigua timestamptz;v_plazo integer;v_numeros numeric;v_accesos bigint;resultado jsonb;v_plazo_iv interval;v_tarde bigint:=0;
BEGIN
 SELECT pg_catalog.make_interval(secs=>s.plazo_maximo_segundos) INTO STRICT v_plazo_iv FROM vec_contratacion_temporal.sellado_acceso_rrhh_v1 s;
 SELECT ultima_secuencia,cabeza_sha256 INTO STRICT v_corte,v_cabeza FROM vec_contratacion_temporal.control_cadena_accesos_rrhh WHERE control;
 resultado:=jsonb_build_object('cadena','accesos_rrhh_ct','corte_secuencia',v_corte,'corte_cabeza_sha256',v_cabeza);
 SELECT count(*),coalesce(max(secuencia),0),
  count(*) FILTER (WHERE anterior_sha256 IS DISTINCT FROM coalesce(previa,pg_catalog.repeat('0',64)) OR secuencia<>fila),
  (SELECT a.huella_sha256 FROM vec_contratacion_temporal.registro_acceso_rrhh a WHERE a.secuencia<=v_corte ORDER BY a.secuencia DESC LIMIT 1)
 INTO STRICT v_cuenta,v_max,v_rotos,v_ultima
 FROM (SELECT secuencia,anterior_sha256,lag(huella_sha256) OVER (ORDER BY secuencia) previa,
   row_number() OVER (ORDER BY secuencia) fila FROM vec_contratacion_temporal.registro_acceso_rrhh WHERE secuencia<=v_corte) x;
 IF v_cuenta<>v_corte OR v_max<>v_corte OR v_rotos<>0 OR v_cabeza IS DISTINCT FROM coalesce(v_ultima,pg_catalog.repeat('0',64)) THEN
  RETURN resultado||jsonb_build_object('estado','rechazada','fallo','cadena_anterior_al_corte');
 END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.registro_acceso_rrhh a
   WHERE a.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(a.anterior_sha256,'hex')||a.prueba_canonica),'hex')) THEN
  RETURN resultado||jsonb_build_object('estado','rechazada','fallo','huella_acceso');
 END IF;
 v_pos:=v_corte;v_ant:=v_cabeza;
 FOR r IN SELECT e.posicion,e.secuencia,e.anterior_sha256 e_anterior,e.eslabon_sha256,e.sellado_en,a.acceso_ref,a.huella_sha256,
   a.anterior_sha256,a.registrada_en FROM vec_contratacion_temporal.eslabon_acceso_rrhh_v1 e
   LEFT JOIN vec_contratacion_temporal.registro_acceso_rrhh a USING (secuencia) ORDER BY e.posicion
 LOOP
  v_pos:=v_pos+1;
  IF r.posicion<>v_pos THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','posicion','posicion',v_pos); END IF;
  IF r.acceso_ref IS NULL OR r.secuencia<=v_corte OR r.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f',64) THEN
   RETURN resultado||jsonb_build_object('estado','rechazada','fallo','acceso','posicion',v_pos);
  END IF;
  IF r.e_anterior IS DISTINCT FROM v_ant THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','enlace','posicion',v_pos); END IF;
  IF r.sellado_en<r.registrada_en THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','sellado_antes_de_registro','posicion',v_pos); END IF;
  IF r.sellado_en-r.registrada_en>v_plazo_iv THEN v_tarde:=v_tarde+1; END IF;
  v_calc:=vec_contratacion_temporal.eslabon_acceso_rrhh_v1(v_pos,v_ant,r.secuencia,r.acceso_ref,r.huella_sha256,r.registrada_en,r.sellado_en);
  IF v_calc IS DISTINCT FROM r.eslabon_sha256 THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','eslabon','posicion',v_pos); END IF;
  v_ant:=v_calc;v_sellados:=v_sellados+1;
 END LOOP;
 SELECT count(*),min(a.registrada_en) INTO STRICT v_pend,v_antigua FROM vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 p
  JOIN vec_contratacion_temporal.registro_acceso_rrhh a USING (secuencia);
 SELECT count(*) INTO STRICT v_huerf FROM vec_contratacion_temporal.registro_acceso_rrhh a WHERE a.secuencia>v_corte
  AND NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.eslabon_acceso_rrhh_v1 e WHERE e.secuencia=a.secuencia)
  AND NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 p WHERE p.secuencia=a.secuencia);
 SELECT count(*) INTO STRICT v_doble FROM vec_contratacion_temporal.pendiente_sellado_acceso_rrhh_v1 p
  WHERE EXISTS(SELECT 1 FROM vec_contratacion_temporal.eslabon_acceso_rrhh_v1 e WHERE e.secuencia=p.secuencia)
   OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.registro_acceso_rrhh a WHERE a.secuencia=p.secuencia);
 SELECT coalesce(max(secuencia),v_corte)-v_corte,count(*) INTO STRICT v_numeros,v_accesos FROM vec_contratacion_temporal.registro_acceso_rrhh WHERE secuencia>v_corte;
 SELECT s.plazo_maximo_segundos INTO STRICT v_plazo FROM vec_contratacion_temporal.sellado_acceso_rrhh_v1 s;
 resultado:=resultado||jsonb_build_object('sellados',v_sellados,'cabeza_posicion',v_pos,'cabeza_eslabon_sha256',v_ant,
  'pendientes',v_pend,'pendiente_mas_antigua',v_antigua,'plazo_maximo_segundos',v_plazo,
  'numeros_sin_acceso',v_numeros-v_accesos,'sellados_fuera_de_plazo',v_tarde,'sin_sellar_fuera_de_cola',v_huerf,'cola_incoherente',v_doble);
 IF v_huerf<>0 OR v_doble<>0 THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','cola'); END IF;
 IF v_antigua IS NOT NULL AND pg_catalog.clock_timestamp()-v_antigua>pg_catalog.make_interval(secs=>v_plazo) THEN
  RETURN resultado||jsonb_build_object('estado','rechazada','fallo','pendiente_fuera_de_plazo');
 END IF;
 RETURN resultado||jsonb_build_object('estado','verificada');
END $f$;

-- Reescritura de los dos registradores con bloques exactos.
DO $escritores$
DECLARE
 fv1 oid:=to_regprocedure('vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)');
 fv2 oid:=to_regprocedure('vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)');
 f oid;original text;nueva text;actual text;revertida text;meta jsonb;deps jsonb;i integer;viejos text[];nuevos text[];
BEGIN
 IF fv1 IS NULL OR fv2 IS NULL THEN RAISE EXCEPTION 'CT183: PARO clave=registradores esperado=v1_y_v2 actual=ausente' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.prosrc ~ '\mcontrol_cadena_accesos_rrhh' AND p.oid NOT IN (fv1,fv2)
   AND p.proname NOT IN ('encolar_acceso_rrhh_v1','sellar_cadena_accesos_rrhh_v1','verificar_cadena_accesos_rrhh_v1'))
 THEN RAISE EXCEPTION 'CT183: PARO clave=otro_escritor esperado=solo_v1_v2' USING ERRCODE='55000'; END IF;
 FOREACH f IN ARRAY ARRAY[fv1,fv2] LOOP
  IF f=fv1 THEN
   viejos:=ARRAY[$a0$    SELECT ultima_secuencia + 1, cabeza_sha256
      INTO STRICT v_secuencia, v_anterior
      FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
     WHERE control
     FOR UPDATE;$a0$,$a1$    UPDATE vec_contratacion_temporal.control_cadena_accesos_rrhh
       SET ultima_secuencia = v_secuencia,
           cabeza_sha256 = v_huella,
           actualizada_en = v_registrada_en
     WHERE control;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'control del registro de accesos RRHH ausente';
    END IF;$a1$];
   nuevos:=ARRAY[$b0$    SELECT asiento_ct183.secuencia, asiento_ct183.anterior_sha256
      INTO STRICT v_secuencia, v_anterior
      FROM vec_contratacion_temporal.reservar_acceso_rrhh_v1() asiento_ct183;$b0$,$b1$    -- CT183: la cabeza la avanza el sellado diferido.$b1$];
  ELSE
   viejos:=ARRAY[$a0$    SELECT cadena.ultima_secuencia + 1, cadena.cabeza_sha256
      INTO STRICT secuencia, anterior
      FROM vec_contratacion_temporal.control_cadena_accesos_rrhh cadena
      JOIN vec_contratacion_temporal.control_registrador_acceso_rrhh_v2 base
        ON base.control = cadena.control$a0$,$a1$     FOR UPDATE OF cadena FOR SHARE OF base;$a1$,$a2$    UPDATE vec_contratacion_temporal.control_cadena_accesos_rrhh
       SET ultima_secuencia = secuencia,
           cabeza_sha256 = huella,
           actualizada_en = registrada_en
     WHERE control;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'control del registro de accesos RRHH ausente';
    END IF;$a2$];
   nuevos:=ARRAY[$b0$    SELECT asiento_ct183.secuencia, asiento_ct183.anterior_sha256
      INTO STRICT secuencia, anterior
      FROM vec_contratacion_temporal.control_cadena_accesos_rrhh cadena
      JOIN vec_contratacion_temporal.control_registrador_acceso_rrhh_v2 base
        ON base.control = cadena.control
     CROSS JOIN vec_contratacion_temporal.reservar_acceso_rrhh_v1() asiento_ct183$b0$,$b1$     ;$b1$,$b2$    -- CT183: la cabeza la avanza el sellado diferido.$b2$];
  END IF;
  SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
  nueva:=original;
  FOR i IN 1..array_length(viejos,1) LOOP
   IF length(nueva)-length(replace(nueva,viejos[i],''))<>length(viejos[i]) THEN
    RAISE EXCEPTION 'CT183: PARO clave=marca_%_% esperado=1',f::regprocedure,i USING ERRCODE='55000';
   END IF;
   nueva:=replace(nueva,viejos[i],nuevos[i]);
  END LOOP;
  EXECUTE nueva;
  SELECT pg_get_functiondef(f) INTO STRICT actual;
  revertida:=actual;
  FOR i IN REVERSE array_length(viejos,1)..1 LOOP revertida:=replace(revertida,nuevos[i],viejos[i]); END LOOP;
  IF actual IS DISTINCT FROM nueva OR revertida IS DISTINCT FROM original
  OR (SELECT prosrc FROM pg_proc WHERE oid=f) ~ 'FOR\s+UPDATE|UPDATE\s+vec_contratacion_temporal\.control_cadena_accesos_rrhh'
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
  THEN RAISE EXCEPTION 'CT183: PARO clave=delta_% esperado=reversion_y_metadatos_exactos',f::regprocedure USING ERRCODE='55000'; END IF;
 END LOOP;
END $escritores$;

REVOKE ALL ON SEQUENCE vec_contratacion_temporal.secuencia_acceso_rrhh_v1 FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.reservar_acceso_rrhh_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.encolar_acceso_rrhh_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.eslabon_acceso_rrhh_v1(numeric,text,numeric,text,text,timestamptz,timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1(integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.verificar_cadena_accesos_rrhh_v1() FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_auditoria_encadenador;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.sellar_cadena_accesos_rrhh_v1(integer) TO vec_auditoria_encadenador;

DO $post$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.prosrc ~ 'UPDATE\s+vec_contratacion_temporal\.control_cadena_accesos_rrhh')
 OR (vec_contratacion_temporal.verificar_cadena_accesos_rrhh_v1()->>'estado') IS DISTINCT FROM 'verificada'
 THEN RAISE EXCEPTION 'CT183: PARO clave=postcondicion esperado=sin_bloqueo_y_cadena_verificada' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
