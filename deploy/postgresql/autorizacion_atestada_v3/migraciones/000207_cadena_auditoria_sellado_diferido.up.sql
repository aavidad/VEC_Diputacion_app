\set ON_ERROR_STOP on
-- AD207: cadena de auditoría V3 con sellado diferido.
--
-- Antes: cada asiento bloqueaba la única fila de control_cadena_auditoria
-- (FOR UPDATE), leía la cabeza, calculaba su huella con ella y la actualizaba.
-- Esa fila serializaba todas las operaciones auditadas de VEC y, en
-- SERIALIZABLE, provocaba fallos de serialización en cadena.
--
-- Ahora:
-- 1. El escritor toma su número de orden de una secuencia (sin bloqueo común)
--    y calcula la huella de su asiento con la misma preimagen de siempre, pero
--    con un marcador fijo («f» x 64) en el lugar del anterior: la huella liga
--    todo el contenido y el número, no la posición en la cadena.
-- 2. Un disparador encola el asiento en pendiente_sellado_auditoria_v5.
-- 3. El sellador (sellar_cadena_auditoria_v5, grupo vec_auditoria_encadenador)
--    toma los pendientes por lotes, les da posición contigua y calcula:
--      eslabon(p) = sha256(F('vec.auditoria.eslabon.v5') F(cadena) F(p)
--                          F(eslabon(p-1)) F(secuencia) F(auditoria_ref)
--                          F(tipo_registro) F(huella_sha256)
--                          F(registrada_en) F(sellado_en))
--    con F = encuadrar_mac y fechas UTC con microsegundos. Los eslabones van a
--    eslabon_auditoria_v5, de solo adición. eslabon(N) del corte es la cabeza
--    de la cadena anterior.
-- 4. Plazo máximo de incorporación: el sellador deja su latido en
--    sellado_auditoria_v5 solo si la cola no guarda nada más antiguo que el
--    plazo (10 s). Si el latido caduca, el disparador rechaza asientos nuevos
--    y la operación auditada no se confirma: el sistema falla cerrado.
-- 5. control_cadena_auditoria queda congelada como corte: secuencia N y cabeza
--    de la cadena anterior, que no se toca y sigue verificándose igual. Un
--    escritor antiguo que intente avanzarla falla (disparador), no corrompe.
--
-- Lo mismo para la cadena externa (auditoria_consumo_v3_externa), con su
-- propia secuencia, cola y eslabones. Las dos cadenas siguen separadas.
--
-- Los escritores se reescriben con marca única sobre su definición real
-- (pg_get_functiondef): no se fijan huellas completas del cuerpo, porque
-- AD197/AD199/AD200/AD203 y siguientes también miden el núcleo; da igual el
-- orden en que entren. Cada reescritura comprueba que la marca aparece una vez,
-- que la reversión devuelve exactamente el original y que propietario, ACL,
-- configuración y dependencias no cambian.
--
-- Es el patrón de los registros de transparencia (RFC 9162, Trillian): las
-- entradas se aceptan con una promesa y un secuenciador las incorpora por
-- lotes, con número contiguo, dentro de un plazo máximo; la cabeza se ancla
-- fuera con el sello periódico (AD186). El número de orden del asiento es
-- solo su identificador en la cola: puede tener huecos por ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='300s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000207',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE nombre text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 OR to_regrole('vec_auditoria_encadenador') IS NOT NULL
 THEN RAISE EXCEPTION 'AD207: PARO clave=base esperado=PG18_sin_encadenador actual=incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['control_cadena_auditoria','control_cadena_auditoria_externa','auditoria_consumo_v3','auditoria_consumo_v3_externa'] LOOP
  IF to_regclass('vec_autorizacion_atestada_v3.'||nombre) IS NULL
  THEN RAISE EXCEPTION 'AD207: PARO clave=tabla_% esperado=presente actual=ausente',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['secuencia_auditoria_v5','secuencia_auditoria_externa_v5','pendiente_sellado_auditoria_v5',
  'pendiente_sellado_auditoria_externa_v5','eslabon_auditoria_v5','eslabon_auditoria_externa_v5'] LOOP
  IF to_regclass('vec_autorizacion_atestada_v3.'||nombre) IS NOT NULL
  THEN RAISE EXCEPTION 'AD207: PARO clave=objeto_% esperado=ausente actual=presente',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
CREATE ROLE vec_auditoria_encadenador NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.control_cadena_auditoria,
 vec_autorizacion_atestada_v3.control_cadena_auditoria_externa,
 vec_autorizacion_atestada_v3.auditoria_consumo_v3,
 vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa IN ACCESS EXCLUSIVE MODE;

-- La cadena anterior debe estar íntegra en sus enlaces antes de congelarla.
-- La huella de cada asiento la recalcula el verificador por tipo; aquí se
-- comprueban numeración, enlaces y que la cabeza de control es la del último.
DO $corte$
DECLARE v_externa boolean;v_n numeric;v_h text;v_cuenta numeric;v_max numeric;v_rotos bigint;v_ultima text;v_marcados bigint;
BEGIN
 FOREACH v_externa IN ARRAY ARRAY[false,true] LOOP
  EXECUTE format('SELECT secuencia,cabeza_sha256 FROM vec_autorizacion_atestada_v3.%I WHERE control_id',
   CASE WHEN v_externa THEN 'control_cadena_auditoria_externa' ELSE 'control_cadena_auditoria' END) INTO STRICT v_n,v_h;
  EXECUTE format($q$SELECT count(*),coalesce(max(secuencia),0),
    count(*) FILTER (WHERE anterior_sha256 IS DISTINCT FROM coalesce(previa,pg_catalog.repeat('0',64)) OR secuencia<>fila),
    (SELECT a.huella_sha256 FROM vec_autorizacion_atestada_v3.%1$I a ORDER BY a.secuencia DESC LIMIT 1),
    count(*) FILTER (WHERE anterior_sha256=pg_catalog.repeat('f',64))
   FROM (SELECT secuencia,anterior_sha256,lag(huella_sha256) OVER (ORDER BY secuencia) previa,
     row_number() OVER (ORDER BY secuencia) fila FROM vec_autorizacion_atestada_v3.%1$I) x$q$,
   CASE WHEN v_externa THEN 'auditoria_consumo_v3_externa' ELSE 'auditoria_consumo_v3' END)
  INTO STRICT v_cuenta,v_max,v_rotos,v_ultima,v_marcados;
  IF v_cuenta<>v_n OR v_max<>v_n OR v_rotos<>0 OR v_h IS DISTINCT FROM coalesce(v_ultima,pg_catalog.repeat('0',64))
  OR v_marcados<>0
  THEN RAISE EXCEPTION 'AD207: PARO clave=cadena_anterior_% esperado=contigua_enlazada_y_cabeza actual=cuenta_%_max_%_rotos_%',
   CASE WHEN v_externa THEN 'externa' ELSE 'interna' END,v_cuenta,v_max,v_rotos USING ERRCODE='55000'; END IF;
  -- El primer número nuevo sigue al corte.
  EXECUTE format('CREATE SEQUENCE vec_autorizacion_atestada_v3.%I AS bigint MINVALUE 1 MAXVALUE 9007199254740991 START WITH %s NO CYCLE',
   CASE WHEN v_externa THEN 'secuencia_auditoria_externa_v5' ELSE 'secuencia_auditoria_v5' END,v_n+1);
 END LOOP;
END $corte$;

-- Cola de asientos aún sin eslabón. No es evidencia: el sellador borra lo que
-- sella en la misma transacción en que escribe los eslabones.
CREATE TABLE vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 (
 secuencia numeric(20,0) PRIMARY KEY);
CREATE TABLE vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_externa_v5 (
 secuencia numeric(20,0) PRIMARY KEY);

-- Eslabones: solo adición, uno por asiento y posición.
CREATE TABLE vec_autorizacion_atestada_v3.eslabon_auditoria_v5 (
 posicion numeric(20,0) PRIMARY KEY CHECK (posicion BETWEEN 1 AND 9007199254740991),
 secuencia numeric(20,0) NOT NULL UNIQUE REFERENCES vec_autorizacion_atestada_v3.auditoria_consumo_v3(secuencia),
 anterior_sha256 text NOT NULL CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
 eslabon_sha256 text NOT NULL UNIQUE CHECK (eslabon_sha256 ~ '^[0-9a-f]{64}$'),
 sellado_en timestamptz(6) NOT NULL);
CREATE TABLE vec_autorizacion_atestada_v3.eslabon_auditoria_externa_v5 (
 posicion numeric(20,0) PRIMARY KEY CHECK (posicion BETWEEN 1 AND 9007199254740991),
 secuencia numeric(20,0) NOT NULL UNIQUE REFERENCES vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa(secuencia),
 anterior_sha256 text NOT NULL CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
 eslabon_sha256 text NOT NULL UNIQUE CHECK (eslabon_sha256 ~ '^[0-9a-f]{64}$'),
 sellado_en timestamptz(6) NOT NULL);

-- Latido del sellador y plazo máximo de incorporación (una fila). El
-- sellador lo actualiza en READ COMMITTED; los escritores SERIALIZABLE solo lo
-- leen, así que no se crean conflictos de serialización entre ellos.
CREATE TABLE vec_autorizacion_atestada_v3.sellado_auditoria_v5 (
 control boolean PRIMARY KEY DEFAULT true CHECK (control),
 latido timestamptz(6) NOT NULL,
 plazo_maximo_segundos integer NOT NULL CHECK (plazo_maximo_segundos BETWEEN 1 AND 300));
INSERT INTO vec_autorizacion_atestada_v3.sellado_auditoria_v5 VALUES (true,clock_timestamp(),10);
-- Solo el latido cambia; el plazo se cambia con otra migración, que queda en Git.
CREATE FUNCTION vec_autorizacion_atestada_v3.solo_latido_ad207() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp
AS $f$ BEGIN
 IF NEW.plazo_maximo_segundos IS DISTINCT FROM OLD.plazo_maximo_segundos OR NEW.control IS DISTINCT FROM OLD.control THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='plazo VEC-AD-3 inmutable';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.solo_latido_ad207() FROM PUBLIC;
CREATE TRIGGER solo_latido_ad207 BEFORE UPDATE ON vec_autorizacion_atestada_v3.sellado_auditoria_v5
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.solo_latido_ad207();
CREATE TRIGGER no_borrar_ad207 BEFORE DELETE ON vec_autorizacion_atestada_v3.sellado_auditoria_v5
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar_ad207 BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.sellado_auditoria_v5
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();

DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['pendiente_sellado_auditoria_v5','pendiente_sellado_auditoria_externa_v5','eslabon_auditoria_v5','eslabon_auditoria_externa_v5','sellado_auditoria_v5'] LOOP
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.%I FROM PUBLIC',t);
  EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.%I TO vec_autorizacion_atestada_v3_propietario USING (true) WITH CHECK (true)',t);
 END LOOP;
 FOREACH t IN ARRAY ARRAY['eslabon_auditoria_v5','eslabon_auditoria_externa_v5'] LOOP
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()',t);
 END LOOP;
 -- El control queda congelado como corte de la cadena anterior.
 FOREACH t IN ARRAY ARRAY['control_cadena_auditoria','control_cadena_auditoria_externa'] LOOP
  EXECUTE format('CREATE TRIGGER congelada_ad207 BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion()',t);
  EXECUTE format('CREATE TRIGGER no_truncar_ad207 BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado()',t);
 END LOOP;
END $tablas$;

-- Número de orden para un asiento nuevo. Devuelve el número anterior al
-- reservado porque los escritores existentes suman 1 a lo que leían del control.
CREATE FUNCTION vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()
 RETURNS TABLE(secuencia_previa numeric, anterior_sha256 text)
 LANGUAGE sql VOLATILE SET search_path=pg_catalog,pg_temp
AS $f$ SELECT pg_catalog.nextval('vec_autorizacion_atestada_v3.secuencia_auditoria_v5'::pg_catalog.regclass)::numeric-1,pg_catalog.repeat('f',64) $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.reservar_asiento_auditoria_externa_v5()
 RETURNS TABLE(secuencia_previa numeric, anterior_sha256 text)
 LANGUAGE sql VOLATILE SET search_path=pg_catalog,pg_temp
AS $f$ SELECT pg_catalog.nextval('vec_autorizacion_atestada_v3.secuencia_auditoria_externa_v5'::pg_catalog.regclass)::numeric-1,pg_catalog.repeat('f',64) $f$;

-- Todo asiento nuevo entra en la cola. Rechaza los que no siguen al corte o no
-- llevan el marcador: así un escritor antiguo no puede colarse en la cadena.
CREATE FUNCTION vec_autorizacion_atestada_v3.encolar_asiento_auditoria_v5()
 RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp
AS $f$
DECLARE v_corte numeric;v_latido timestamptz;v_plazo integer;v_ahora timestamptz;
BEGIN
 IF TG_TABLE_NAME='auditoria_consumo_v3' THEN
  SELECT c.secuencia INTO STRICT v_corte FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c;
 ELSE
  SELECT c.secuencia INTO STRICT v_corte FROM vec_autorizacion_atestada_v3.control_cadena_auditoria_externa c;
 END IF;
 IF NEW.secuencia<=v_corte OR NEW.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f',64) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='asiento VEC-AD-3 fuera de la cadena con sellado diferido';
 END IF;
 -- Plazo máximo de incorporación: sin sellador al día no se confirma nada.
 SELECT s.latido,s.plazo_maximo_segundos INTO STRICT v_latido,v_plazo FROM vec_autorizacion_atestada_v3.sellado_auditoria_v5 s;
 -- En REPEATABLE READ y SERIALIZABLE el latido se lee con la instantánea de la
 -- transacción: se compara con su inicio (now()) para no rechazar
 -- transacciones largas legítimas, cuya duración acota transaction_timeout en
 -- los LOGIN de la aplicación. En READ COMMITTED se lee el actual.
 v_ahora:=(CASE WHEN current_setting('transaction_isolation') IN ('repeatable read','serializable')
  THEN pg_catalog.now() ELSE pg_catalog.clock_timestamp() END);
 IF v_ahora-v_latido>pg_catalog.make_interval(secs=>v_plazo) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='sellado VEC-AD-3 detenido: asiento rechazado';
 END IF;
 IF TG_TABLE_NAME='auditoria_consumo_v3' THEN
  INSERT INTO vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5(secuencia) VALUES (NEW.secuencia);
 ELSE
  INSERT INTO vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_externa_v5(secuencia) VALUES (NEW.secuencia);
 END IF;
 RETURN NULL;
END $f$;
CREATE TRIGGER encolar_sellado_ad207 AFTER INSERT ON vec_autorizacion_atestada_v3.auditoria_consumo_v3
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.encolar_asiento_auditoria_v5();
CREATE TRIGGER encolar_sellado_ad207 AFTER INSERT ON vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.encolar_asiento_auditoria_v5();

-- Eslabón v5. Mismo cálculo en el sellador, en la verificación SQL y en Go.
CREATE FUNCTION vec_autorizacion_atestada_v3.eslabon_auditoria_v5(p_cadena text,p_posicion numeric,p_anterior text,
 p_secuencia numeric,p_auditoria_ref text,p_tipo text,p_huella text,p_registrada_en timestamptz,p_sellado_en timestamptz)
 RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp
AS $f$ SELECT pg_catalog.encode(pg_catalog.sha256(
 vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.v5')||vec_autorizacion_atestada_v3.encuadrar_mac(p_cadena)||
 vec_autorizacion_atestada_v3.encuadrar_mac(p_posicion::text)||vec_autorizacion_atestada_v3.encuadrar_mac(p_anterior)||
 vec_autorizacion_atestada_v3.encuadrar_mac(p_secuencia::text)||vec_autorizacion_atestada_v3.encuadrar_mac(p_auditoria_ref)||
 vec_autorizacion_atestada_v3.encuadrar_mac(p_tipo)||vec_autorizacion_atestada_v3.encuadrar_mac(p_huella)||
 vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(p_registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))||
 vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(p_sellado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex') $f$;

-- Cabeza sellada de la cadena interna (posición y eslabón). Sin eslabones es
-- el corte congelado.
CREATE FUNCTION vec_autorizacion_atestada_v3.cabeza_sellada_auditoria_v5()
 RETURNS TABLE(posicion numeric, eslabon_sha256 text)
 LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp
AS $f$ SELECT x.posicion,x.eslabon_sha256 FROM (
 (SELECT e.posicion,e.eslabon_sha256 FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5 e ORDER BY e.posicion DESC LIMIT 1)
 UNION ALL
 (SELECT c.secuencia,c.cabeza_sha256 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c)) x
 ORDER BY x.posicion DESC LIMIT 1 $f$;

-- Sella hasta p_max pendientes de una cadena, en orden de número.
CREATE FUNCTION vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(p_externa boolean,p_max integer)
 RETURNS integer LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp
AS $f$
DECLARE
 v_aud text:=CASE WHEN p_externa THEN 'auditoria_consumo_v3_externa' ELSE 'auditoria_consumo_v3' END;
 v_ctl text:=CASE WHEN p_externa THEN 'control_cadena_auditoria_externa' ELSE 'control_cadena_auditoria' END;
 v_cola text:=CASE WHEN p_externa THEN 'pendiente_sellado_auditoria_externa_v5' ELSE 'pendiente_sellado_auditoria_v5' END;
 v_esl text:=CASE WHEN p_externa THEN 'eslabon_auditoria_externa_v5' ELSE 'eslabon_auditoria_v5' END;
 v_cadena text:=CASE WHEN p_externa THEN 'externa' ELSE 'interna' END;
 v_pos numeric;v_ant text;r record;n integer;v_borrados integer;v_sellado timestamptz(6);i integer;
 l_ref text[]:='{}';l_tipo text[]:='{}';l_huella text[]:='{}';l_fecha timestamptz[]:='{}';
 a_pos numeric[]:='{}';a_sec numeric[]:='{}';a_ant text[]:='{}';a_esl text[]:='{}';
BEGIN
 EXECUTE format('SELECT posicion,eslabon_sha256 FROM vec_autorizacion_atestada_v3.%I ORDER BY posicion DESC LIMIT 1',v_esl) INTO v_pos,v_ant;
 IF v_pos IS NULL THEN
  EXECUTE format('SELECT secuencia,cabeza_sha256 FROM vec_autorizacion_atestada_v3.%I WHERE control_id',v_ctl) INTO STRICT v_pos,v_ant;
 END IF;
 -- El lote sale de la cola y cada asiento se busca por su índice (LATERAL):
 -- un merge join recorrería el índice de toda la tabla en cada lote.
 FOR r IN EXECUTE format('SELECT a.secuencia,a.auditoria_ref,%s AS tipo,a.huella_sha256,a.anterior_sha256,a.registrada_en
   FROM (SELECT q.secuencia FROM vec_autorizacion_atestada_v3.%I q ORDER BY q.secuencia LIMIT $1) p
   CROSS JOIN LATERAL (SELECT x.* FROM vec_autorizacion_atestada_v3.%I x WHERE x.secuencia=p.secuencia) a
   ORDER BY p.secuencia',CASE WHEN p_externa THEN '''''::text' ELSE 'a.tipo_registro' END,v_cola,v_aud) USING p_max
 LOOP
  IF r.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f',64) THEN
   RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='asiento VEC-AD-3 pendiente sin marcador';
  END IF;
  a_sec:=a_sec||r.secuencia;l_ref:=l_ref||r.auditoria_ref;l_tipo:=l_tipo||r.tipo;l_huella:=l_huella||r.huella_sha256;l_fecha:=l_fecha||r.registrada_en;
 END LOOP;
 n:=pg_catalog.cardinality(a_sec);
 IF n=0 THEN RETURN 0; END IF;
 -- La hora de sellado se toma después de leer el lote: es posterior a la
 -- confirmación de todos sus asientos, así que sellado_en >= registrada_en.
 v_sellado:=pg_catalog.clock_timestamp();
 FOR i IN 1..n LOOP
  a_ant:=a_ant||v_ant;
  v_pos:=v_pos+1;
  v_ant:=vec_autorizacion_atestada_v3.eslabon_auditoria_v5(v_cadena,v_pos,v_ant,a_sec[i],l_ref[i],l_tipo[i],l_huella[i],l_fecha[i],v_sellado);
  a_pos:=a_pos||v_pos;a_esl:=a_esl||v_ant;
 END LOOP;
 EXECUTE format('INSERT INTO vec_autorizacion_atestada_v3.%I(posicion,secuencia,anterior_sha256,eslabon_sha256,sellado_en)
   SELECT p,s,a,e,$5 FROM unnest($1,$2,$3,$4) AS u(p,s,a,e)',v_esl) USING a_pos,a_sec,a_ant,a_esl,v_sellado;
 EXECUTE format('DELETE FROM vec_autorizacion_atestada_v3.%I WHERE secuencia=ANY($1)',v_cola) USING a_sec;
 GET DIAGNOSTICS v_borrados = ROW_COUNT;
 IF v_borrados<>n THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='sellado VEC-AD-3: cola incoherente';
 END IF;
 RETURN n;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(p_max integer)
 RETURNS TABLE(interna integer, externa integer)
 LANGUAGE plpgsql SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET TimeZone='UTC'
AS $f$
DECLARE v_interna integer;v_externa integer;v_plazo integer;v_antigua timestamptz;
BEGIN
 IF p_max IS NULL OR p_max NOT BETWEEN 1 AND 50000 THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='sellado VEC-AD-3: lote inválido';
 END IF;
 -- En READ COMMITTED cada sentencia ve lo confirmado por el sellado anterior,
 -- que soltó el cerrojo al confirmar. Con otra instantánea el lote podría
 -- partir de una cabeza vieja (lo pararía la clave primaria, pero se evita).
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION USING ERRCODE='25000',MESSAGE='sellado VEC-AD-3: transacción no admitida';
 END IF;
 IF NOT pg_catalog.pg_try_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:sellado_v5',0)) THEN
  RETURN QUERY SELECT 0,0;
  RETURN;
 END IF;
 v_interna:=vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(false,p_max);
 v_externa:=vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(true,p_max);
 -- El latido solo avanza si no queda en cola nada más antiguo que el plazo.
 SELECT s.plazo_maximo_segundos INTO STRICT v_plazo FROM vec_autorizacion_atestada_v3.sellado_auditoria_v5 s;
 SELECT min(x.registrada_en) INTO v_antigua FROM (
  SELECT (SELECT a.registrada_en FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.secuencia=
   (SELECT min(p.secuencia) FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 p)) registrada_en
  UNION ALL
  SELECT (SELECT a.registrada_en FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa a WHERE a.secuencia=
   (SELECT min(p.secuencia) FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_externa_v5 p))) x;
 IF v_antigua IS NULL OR pg_catalog.clock_timestamp()-v_antigua<=pg_catalog.make_interval(secs=>v_plazo) THEN
  UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=pg_catalog.clock_timestamp() WHERE control;
 END IF;
 RETURN QUERY SELECT v_interna,v_externa;
END $f$;

-- Verificación de enlaces de extremo a extremo (propietario o DBA). Recalcula cada
-- eslabón desde el asiento y comprueba la cadena anterior hasta el corte. La
-- huella de cada asiento según su tipo la recalcula el verificador Go.
CREATE FUNCTION vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(p_externa boolean)
 RETURNS jsonb LANGUAGE plpgsql
 SET search_path=pg_catalog,pg_temp SET row_security=on SET TimeZone='UTC'
AS $f$
DECLARE
 v_aud text:=CASE WHEN p_externa THEN 'auditoria_consumo_v3_externa' ELSE 'auditoria_consumo_v3' END;
 v_ctl text:=CASE WHEN p_externa THEN 'control_cadena_auditoria_externa' ELSE 'control_cadena_auditoria' END;
 v_cola text:=CASE WHEN p_externa THEN 'pendiente_sellado_auditoria_externa_v5' ELSE 'pendiente_sellado_auditoria_v5' END;
 v_esl text:=CASE WHEN p_externa THEN 'eslabon_auditoria_externa_v5' ELSE 'eslabon_auditoria_v5' END;
 v_cadena text:=CASE WHEN p_externa THEN 'externa' ELSE 'interna' END;
 v_corte numeric;v_cabeza text;v_cuenta numeric;v_max numeric;v_rotos bigint;v_ultima text;
 v_pos numeric;v_ant text;v_calc text;r record;v_sellados bigint:=0;v_pend bigint;v_huerf bigint;v_doble bigint;
 v_antigua timestamptz;v_plazo integer;v_numeros numeric;v_asientos bigint;resultado jsonb;v_plazo_iv interval;v_tarde bigint:=0;
BEGIN
 SELECT pg_catalog.make_interval(secs=>s.plazo_maximo_segundos) INTO STRICT v_plazo_iv FROM vec_autorizacion_atestada_v3.sellado_auditoria_v5 s;
 EXECUTE format('SELECT secuencia,cabeza_sha256 FROM vec_autorizacion_atestada_v3.%I WHERE control_id',v_ctl) INTO STRICT v_corte,v_cabeza;
 resultado:=jsonb_build_object('cadena',v_cadena,'corte_secuencia',v_corte,'corte_cabeza_sha256',v_cabeza);
 EXECUTE format($q$SELECT count(*),coalesce(max(secuencia),0),
   count(*) FILTER (WHERE anterior_sha256 IS DISTINCT FROM coalesce(previa,pg_catalog.repeat('0',64)) OR secuencia<>fila),
   (SELECT a.huella_sha256 FROM vec_autorizacion_atestada_v3.%1$I a WHERE a.secuencia<=$1 ORDER BY a.secuencia DESC LIMIT 1)
  FROM (SELECT secuencia,anterior_sha256,lag(huella_sha256) OVER (ORDER BY secuencia) previa,
    row_number() OVER (ORDER BY secuencia) fila FROM vec_autorizacion_atestada_v3.%1$I WHERE secuencia<=$1) x$q$,v_aud)
  USING v_corte INTO STRICT v_cuenta,v_max,v_rotos,v_ultima;
 IF v_cuenta<>v_corte OR v_max<>v_corte OR v_rotos<>0 OR v_cabeza IS DISTINCT FROM coalesce(v_ultima,pg_catalog.repeat('0',64)) THEN
  RETURN resultado||jsonb_build_object('estado','rechazada','fallo','cadena_anterior_al_corte');
 END IF;
 v_pos:=v_corte;v_ant:=v_cabeza;
 FOR r IN EXECUTE format('SELECT e.posicion,e.secuencia,e.anterior_sha256 e_anterior,e.eslabon_sha256,e.sellado_en,a.auditoria_ref,%s AS tipo,
   a.huella_sha256,a.anterior_sha256,a.registrada_en FROM vec_autorizacion_atestada_v3.%I e
   LEFT JOIN vec_autorizacion_atestada_v3.%I a USING (secuencia) ORDER BY e.posicion',
   CASE WHEN p_externa THEN '''''::text' ELSE 'a.tipo_registro' END,v_esl,v_aud)
 LOOP
  v_pos:=v_pos+1;
  IF r.posicion<>v_pos THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','posicion','posicion',v_pos); END IF;
  IF r.auditoria_ref IS NULL OR r.secuencia<=v_corte OR r.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f',64) THEN
   RETURN resultado||jsonb_build_object('estado','rechazada','fallo','asiento','posicion',v_pos);
  END IF;
  IF r.e_anterior IS DISTINCT FROM v_ant THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','enlace','posicion',v_pos); END IF;
  IF r.sellado_en<r.registrada_en THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','sellado_antes_de_registro','posicion',v_pos); END IF;
  IF r.sellado_en-r.registrada_en>v_plazo_iv THEN v_tarde:=v_tarde+1; END IF;
  v_calc:=vec_autorizacion_atestada_v3.eslabon_auditoria_v5(v_cadena,v_pos,v_ant,r.secuencia,r.auditoria_ref,r.tipo,r.huella_sha256,r.registrada_en,r.sellado_en);
  IF v_calc IS DISTINCT FROM r.eslabon_sha256 THEN RETURN resultado||jsonb_build_object('estado','rechazada','fallo','eslabon','posicion',v_pos); END IF;
  v_ant:=v_calc;v_sellados:=v_sellados+1;
 END LOOP;
 EXECUTE format('SELECT count(*),min(a.registrada_en) FROM vec_autorizacion_atestada_v3.%I p JOIN vec_autorizacion_atestada_v3.%I a USING (secuencia)',v_cola,v_aud)
  INTO STRICT v_pend,v_antigua;
 EXECUTE format('SELECT count(*) FROM vec_autorizacion_atestada_v3.%I a WHERE a.secuencia>$1
   AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.%I e WHERE e.secuencia=a.secuencia)
   AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.%I p WHERE p.secuencia=a.secuencia)',v_aud,v_esl,v_cola) USING v_corte INTO STRICT v_huerf;
 EXECUTE format('SELECT count(*) FROM vec_autorizacion_atestada_v3.%I p WHERE EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.%I e WHERE e.secuencia=p.secuencia)
   OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.%I a WHERE a.secuencia=p.secuencia)',v_cola,v_esl,v_aud) INTO STRICT v_doble;
 EXECUTE format('SELECT coalesce(max(secuencia),$1)-$1,count(*) FROM vec_autorizacion_atestada_v3.%I WHERE secuencia>$1',v_aud) USING v_corte INTO STRICT v_numeros,v_asientos;
 SELECT s.plazo_maximo_segundos INTO STRICT v_plazo FROM vec_autorizacion_atestada_v3.sellado_auditoria_v5 s;
 resultado:=resultado||jsonb_build_object('sellados',v_sellados,'cabeza_posicion',v_pos,'cabeza_eslabon_sha256',v_ant,
  'pendientes',v_pend,'pendiente_mas_antigua',v_antigua,'plazo_maximo_segundos',v_plazo,
  'numeros_sin_asiento',v_numeros-v_asientos,'sellados_fuera_de_plazo',v_tarde,'sin_sellar_fuera_de_cola',v_huerf,'cola_incoherente',v_doble);
 IF v_huerf<>0 OR v_doble<>0 THEN
  RETURN resultado||jsonb_build_object('estado','rechazada','fallo','cola');
 END IF;
 IF v_antigua IS NOT NULL AND pg_catalog.clock_timestamp()-v_antigua>pg_catalog.make_interval(secs=>v_plazo) THEN
  RETURN resultado||jsonb_build_object('estado','rechazada','fallo','pendiente_fuera_de_plazo');
 END IF;
 RETURN resultado||jsonb_build_object('estado','verificada');
END $f$;

-- Reescritura de los escritores: el SELECT ... FOR UPDATE del control pasa a
-- reservar un número; la actualización del control desaparece. Nada más cambia.
DO $escritores$
DECLARE
 re_sel text:='SELECT\s+(\w+\.)?secuencia\s*,\s*(\w+\.)?cabeza_sha256\s+INTO\s+STRICT\s+(\w+)\s*,\s*(\w+)\s+FROM\s+(?:vec_autorizacion_atestada_v3\.)?control_cadena_auditoria(_externa)?(\s+\w+)?\s+WHERE\s+(\w+\.)?control_id\s+FOR\s+UPDATE\s*;';
 re_upd text:='UPDATE\s+(?:vec_autorizacion_atestada_v3\.)?control_cadena_auditoria(_externa)?\s+SET\s[^;]*WHERE\s+control_id\s*;';
 conocidos text[]:=ARRAY['consumir_consulta_rrhh_v3_interna','consumir_decision_mutacion_v3_interna',
  'consumir_decision_mutacion_v3_externa_interna','consumir_decision_mutacion_v3_usuarios_externa_interna',
  'registrar_contexto_admin_pre_v2_interna_v1','registrar_evento_admin_preperfil_v1','registrar_frontera_admin_tecnica_v1',
  'registrar_gobierno_usuarios_admin_v1','registrar_intento_bootstrap_central_admin_v1','registrar_intento_fuentes_iniciales_admin_v1',
  'registrar_intento_gobierno_usuarios_admin_v1','registrar_intento_mantenimiento_perfil_fijo_admin_v1','registrar_intento_nominal_v1',
  'registrar_intento_perfiles_asignables_admin_v1','registrar_intento_unidad_inicial_personal_v1','registrar_mantenimiento_perfil_fijo_admin_v1',
  'registrar_operacion_periodica_v1','registrar_operacion_preservacion_v1','registrar_perfiles_asignables_admin_v1',
  'registrar_provision_fuentes_iniciales_admin_v1','registrar_unidad_inicial_personal_v1'];
 f record;original text;nueva text;actual text;revertida text;meta jsonb;deps jsonb;
 m text[];viejo_sel text;nuevo_sel text;viejo_upd text;nuevo_upd text;hechos text[]:='{}';v_var text;
BEGIN
 FOR f IN SELECT p.oid,n.nspname,p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE ((n.nspname='vec_autorizacion_atestada_v3' AND p.prosrc ~ '\mcontrol_cadena_auditoria')
    OR p.prosrc ~ 'vec_autorizacion_atestada_v3\.control_cadena_auditoria')
   AND p.proname <> ALL(ARRAY['capturar_sello_periodico_v1','encolar_asiento_auditoria_v5','cabeza_sellada_auditoria_v5',
    'sellar_tramo_auditoria_v5','verificar_cadena_auditoria_v5'])
  ORDER BY p.oid LOOP
  IF f.nspname<>'vec_autorizacion_atestada_v3' THEN
   RAISE EXCEPTION 'AD207: PARO clave=escritor_fuera_esquema actual=%.%',f.nspname,f.proname USING ERRCODE='55000';
  END IF;
  SELECT pg_get_functiondef(f.oid),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f.oid;
  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f.oid;
  IF (SELECT count(*) FROM regexp_matches(original,re_sel,'g'))<>1 OR (SELECT count(*) FROM regexp_matches(original,re_upd,'g'))<>1 THEN
   RAISE EXCEPTION 'AD207: PARO clave=marca_escritor_% esperado=1_SELECT_1_UPDATE',f.proname USING ERRCODE='55000';
  END IF;
  m:=regexp_match(original,'('||re_sel||')');
  viejo_sel:=m[1];v_var:=m[4];
  nuevo_sel:='SELECT asiento_ad207.secuencia_previa, asiento_ad207.anterior_sha256 INTO STRICT '||m[4]||', '||m[5]||
   ' FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria'||coalesce(m[6],'')||'_v5() asiento_ad207;';
  m:=regexp_match(original,'('||re_upd||')');
  viejo_upd:=m[1];
  IF coalesce(m[2],'') IS DISTINCT FROM coalesce(substring(viejo_sel from 'control_cadena_auditoria(_externa)'),'') THEN
   RAISE EXCEPTION 'AD207: PARO clave=cadena_mezclada_% esperado=mismo_control',f.proname USING ERRCODE='55000';
  END IF;
  nuevo_upd:='NULL; /* AD207: la cabeza la avanza el sellado diferido */';
  -- El escritor suma 1 al número leído justo después; si no, la reserva no vale.
  IF original !~ ('\m'||v_var||'\s*:=\s*'||v_var||'\s*\+\s*1\M') THEN
   RAISE EXCEPTION 'AD207: PARO clave=incremento_% esperado=variable_mas_1',f.proname USING ERRCODE='55000';
  END IF;
  IF length(original)-length(replace(original,viejo_sel,''))<>length(viejo_sel)
  OR length(original)-length(replace(original,viejo_upd,''))<>length(viejo_upd)
  OR strpos(original,'asiento_ad207')<>0 THEN
   RAISE EXCEPTION 'AD207: PARO clave=marca_literal_% esperado=1',f.proname USING ERRCODE='55000';
  END IF;
  nueva:=replace(replace(original,viejo_sel,nuevo_sel),viejo_upd,nuevo_upd);
  EXECUTE nueva;
  SELECT pg_get_functiondef(f.oid) INTO STRICT actual;
  revertida:=replace(replace(actual,nuevo_upd,viejo_upd),nuevo_sel,viejo_sel);
  IF actual IS DISTINCT FROM nueva OR revertida IS DISTINCT FROM original
  OR (SELECT prosrc FROM pg_proc WHERE oid=f.oid) ~ '\mcontrol_cadena_auditoria'
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f.oid) IS DISTINCT FROM meta
  OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f.oid) IS DISTINCT FROM deps
  THEN RAISE EXCEPTION 'AD207: PARO clave=delta_% esperado=reversion_y_metadatos_exactos',f.proname USING ERRCODE='55000'; END IF;
  hechos:=hechos||f.proname::text;
 END LOOP;
 IF NOT (hechos @> conocidos) THEN
  RAISE EXCEPTION 'AD207: PARO clave=escritores esperado=los_% conocidos actual=%',cardinality(conocidos),hechos USING ERRCODE='55000';
 END IF;
 RAISE NOTICE 'AD207: % escritores reescritos',cardinality(hechos);
END $escritores$;

-- Prueba autoritativa de una consulta RRHH: tras el corte, el asiento lleva el
-- marcador y su enlace lo da el eslabón, no el asiento anterior; debe estar
-- sellado o esperando en la cola.
DO $revalidar$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;nueva text;actual text;meta jsonb;
 viejo text:=$a$       OR (
           v_prueba.secuencia > 1
           AND NOT EXISTS ($a$;
 nuevo text:=$b$       OR (
           v_prueba.secuencia > (SELECT k.secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria k)
           AND (v_prueba.anterior_sha256 IS DISTINCT FROM pg_catalog.repeat('f', 64)
                OR (NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5 e
                                 WHERE e.secuencia = v_prueba.secuencia)
                    AND NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 q
                                     WHERE q.secuencia = v_prueba.secuencia)))
       )
       OR (
           v_prueba.secuencia > 1
           AND v_prueba.secuencia <= (SELECT k.secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria k)
           AND NOT EXISTS ($b$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD207: PARO clave=revalidar esperado=presente actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
 IF length(original)-length(replace(original,viejo,''))<>length(viejo) THEN
  RAISE EXCEPTION 'AD207: PARO clave=marca_revalidar esperado=1' USING ERRCODE='55000';
 END IF;
 nueva:=replace(original,viejo,nuevo);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva OR replace(actual,nuevo,viejo) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 THEN RAISE EXCEPTION 'AD207: PARO clave=delta_revalidar esperado=reversion_y_metadatos_exactos' USING ERRCODE='55000'; END IF;
END $revalidar$;

-- Sello periódico (AD186): el checkpoint cubre la cabeza sellada al capturar.
-- El asiento de la propia captura se sella después y entra en el siguiente.
DO $captura$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.capturar_sello_periodico_v1(text,numeric)');
 original text;nueva text;actual text;revertida text;meta jsonb;i integer;
 viejos text[]:=ARRAY[
  $a0$SELECT secuencia,cabeza_sha256 INTO STRICT v_n,v_h FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id FOR UPDATE;$a0$,
  $a1$OR v_n-v_desde+2>p_max_registros$a1$,
  $a3$ v_prev:=coalesce(v_ultimo.checkpoint->'cobertura'->>'cabeza_sha256',repeat('0',64));$a3$,
  $a2$'ultima_secuencia',v_acuse->'secuencia','registros',(v_acuse->>'secuencia')::numeric-v_desde+1,'anterior_sha256',v_prev,'cabeza_sha256',v_acuse->>'huella_sha256'$a2$];
 nuevos text[]:=ARRAY[
  $b0$SELECT c.posicion,c.eslabon_sha256 INTO STRICT v_n,v_h FROM vec_autorizacion_atestada_v3.cabeza_sellada_auditoria_v5() c;$b0$,
  $b1$OR v_n-v_desde+1>p_max_registros$b1$,
  $b3$ v_prev:=coalesce(v_ultimo.checkpoint->'cobertura'->>'cabeza_sha256',repeat('0',64));
 IF v_n<v_desde THEN RAISE EXCEPTION 'periodica_sin_sellado_nuevo' USING ERRCODE='55000'; END IF;$b3$,
  $b2$'ultima_secuencia',v_n,'registros',v_n-v_desde+1,'anterior_sha256',v_prev,'cabeza_sha256',v_h$b2$];
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD207: PARO clave=captura esperado=presente actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE p.oid=f;
 nueva:=original;
 FOR i IN 1..array_length(viejos,1) LOOP
  IF length(nueva)-length(replace(nueva,viejos[i],''))<>length(viejos[i]) THEN
   RAISE EXCEPTION 'AD207: PARO clave=marca_captura_% esperado=1',i USING ERRCODE='55000';
  END IF;
  nueva:=replace(nueva,viejos[i],nuevos[i]);
 END LOOP;
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 revertida:=actual;
 FOR i IN REVERSE array_length(viejos,1)..1 LOOP revertida:=replace(revertida,nuevos[i],viejos[i]); END LOOP;
 IF actual IS DISTINCT FROM nueva OR revertida IS DISTINCT FROM original
 OR (SELECT prosrc FROM pg_proc WHERE oid=f) ~ 'vec_autorizacion_atestada_v3\.control_cadena_auditoria'
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 THEN RAISE EXCEPTION 'AD207: PARO clave=delta_captura esperado=reversion_y_metadatos_exactos' USING ERRCODE='55000'; END IF;
END $captura$;

-- Permisos: solo el propietario usa las piezas internas; el encadenador solo
-- puede sellar. La verificación la ejecuta el DBA como propietario.
REVOKE ALL ON SEQUENCE vec_autorizacion_atestada_v3.secuencia_auditoria_v5, vec_autorizacion_atestada_v3.secuencia_auditoria_externa_v5 FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.reservar_asiento_auditoria_externa_v5() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.encolar_asiento_auditoria_v5() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.eslabon_auditoria_v5(text,numeric,text,numeric,text,text,text,timestamptz,timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cabeza_sellada_auditoria_v5() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(boolean,integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(integer) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(boolean) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_auditoria_encadenador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(integer) TO vec_auditoria_encadenador;

DO $post$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   WHERE (n.nspname='vec_autorizacion_atestada_v3' AND (p.prosrc ~ '\mcontrol_cadena_auditoria(_externa)?(\s+\w+)?\s+WHERE\s+(\w+\.)?control_id\s+FOR\s+UPDATE'
     OR p.prosrc ~ 'UPDATE\s+(vec_autorizacion_atestada_v3\.)?control_cadena_auditoria'))
   OR p.prosrc ~ 'vec_autorizacion_atestada_v3\.control_cadena_auditoria(_externa)?(\s+\w+)?\s+WHERE\s+(\w+\.)?control_id\s+FOR\s+UPDATE'
   OR p.prosrc ~ 'UPDATE\s+vec_autorizacion_atestada_v3\.control_cadena_auditoria')
 OR (SELECT count(*) FROM pg_proc p WHERE p.prosrc ~ 'reservar_asiento_auditoria(_externa)?_v5\(\) asiento_ad207')<21
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5)
 OR (vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(false)->>'estado') IS DISTINCT FROM 'verificada'
 OR (vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(true)->>'estado') IS DISTINCT FROM 'verificada'
 OR (SELECT count(*) FROM aclexplode((SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(integer)'::regprocedure)) a
     WHERE a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_auditoria_encadenador'::regrole))<>0
 THEN RAISE EXCEPTION 'AD207: PARO clave=postcondicion esperado=sin_bloqueo_comun_cadenas_verificadas_ACL_exacta' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
