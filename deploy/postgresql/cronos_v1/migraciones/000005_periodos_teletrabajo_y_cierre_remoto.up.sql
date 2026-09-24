\set ON_ERROR_STOP on
-- Cronos bloque 7. Conserva periodos y revocaciones inmutables. No abre
-- marcaje remoto: requiere consumidor nominal V3 y contrato transaccional
-- compatible con base exclusiva de Cronos, aún sin resolver.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000005',0));
CREATE TABLE vec_cronos_v1.teletrabajo_autorizacion (
  autorizacion_ref text PRIMARY KEY CHECK (autorizacion_ref ~ '^teletrabajo:cronos:[-A-Za-z0-9_]{1,128}$'),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  periodo tstzrange NOT NULL CHECK (NOT isempty(periodo) AND lower_inc(periodo) AND NOT upper_inc(periodo)
    AND NOT lower_inf(periodo) AND NOT upper_inf(periodo) AND lower(periodo)<upper(periodo)),
  resolucion_ref text NOT NULL UNIQUE CHECK (resolucion_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
  politica_version_ref text NOT NULL CHECK (politica_version_ref ~ '^[-A-Za-z0-9_.:]{1,128}$'),
  actor_resolutor_ref text NOT NULL CHECK (actor_resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
  registrada_en timestamptz(6) NOT NULL
);
CREATE INDEX teletrabajo_autorizacion_persona_periodo_idx
 ON vec_cronos_v1.teletrabajo_autorizacion USING gist(periodo);
CREATE INDEX teletrabajo_autorizacion_persona_idx
 ON vec_cronos_v1.teletrabajo_autorizacion(empleado_ref);
CREATE FUNCTION vec_cronos_v1.comprobar_periodo_teletrabajo_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF current_setting('transaction_isolation') IS DISTINCT FROM 'read committed' THEN
   RAISE EXCEPTION 'aislamiento Cronos no admitido' USING ERRCODE='PC003';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:teletrabajo:'||NEW.empleado_ref,0));
 IF EXISTS (SELECT 1 FROM vec_cronos_v1.teletrabajo_autorizacion a
     LEFT JOIN vec_cronos_v1.teletrabajo_revocacion r
       ON r.autorizacion_ref=a.autorizacion_ref
     WHERE a.empleado_ref=NEW.empleado_ref
       AND tstzrange(lower(a.periodo),least(upper(a.periodo),coalesce(r.efectiva_en,upper(a.periodo))),'[)') && NEW.periodo) THEN
   RAISE EXCEPTION 'periodo teletrabajo solapado' USING ERRCODE='PC002';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.comprobar_periodo_teletrabajo_v1() FROM PUBLIC;
CREATE TRIGGER periodo_teletrabajo_unico BEFORE INSERT ON vec_cronos_v1.teletrabajo_autorizacion
 FOR EACH ROW EXECUTE FUNCTION vec_cronos_v1.comprobar_periodo_teletrabajo_v1();
CREATE TABLE vec_cronos_v1.teletrabajo_revocacion (
  revocacion_ref text PRIMARY KEY CHECK (revocacion_ref ~ '^revocacion:cronos:[-A-Za-z0-9_]{1,128}$'),
  autorizacion_ref text NOT NULL UNIQUE REFERENCES vec_cronos_v1.teletrabajo_autorizacion,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  efectiva_en timestamptz(6) NOT NULL,
  resolucion_ref text NOT NULL UNIQUE CHECK (resolucion_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
  actor_resolutor_ref text NOT NULL CHECK (actor_resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
  registrada_en timestamptz(6) NOT NULL
);
CREATE FUNCTION vec_cronos_v1.comprobar_revocacion_teletrabajo_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE autorizada vec_cronos_v1.teletrabajo_autorizacion%ROWTYPE;
BEGIN
 IF current_setting('transaction_isolation') IS DISTINCT FROM 'read committed' THEN
   RAISE EXCEPTION 'aislamiento Cronos no admitido' USING ERRCODE='PC003';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:teletrabajo:'||NEW.empleado_ref,0));
 SELECT * INTO autorizada FROM vec_cronos_v1.teletrabajo_autorizacion
 WHERE autorizacion_ref=NEW.autorizacion_ref;
 IF NOT FOUND OR autorizada.empleado_ref IS DISTINCT FROM NEW.empleado_ref
    OR NEW.efectiva_en<lower(autorizada.periodo)
    OR NEW.efectiva_en>=upper(autorizada.periodo) THEN
   RAISE EXCEPTION 'revocacion teletrabajo divergente' USING ERRCODE='PC003';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.comprobar_revocacion_teletrabajo_v1() FROM PUBLIC;
CREATE TRIGGER revocacion_teletrabajo_coherente BEFORE INSERT ON vec_cronos_v1.teletrabajo_revocacion
 FOR EACH ROW EXECUTE FUNCTION vec_cronos_v1.comprobar_revocacion_teletrabajo_v1();
DO $seguridad$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['teletrabajo_autorizacion','teletrabajo_revocacion'] LOOP
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY lectura_propia ON vec_cronos_v1.%I FOR SELECT TO vec_cronos_v1_propietario USING (empleado_ref=nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
  EXECUTE format('CREATE POLICY adicion_propia ON vec_cronos_v1.%I FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (empleado_ref=nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador',tabla);
 END LOOP;
END $seguridad$;
CREATE FUNCTION vec_cronos_v1.teletrabajo_vigente_interno_v1(p_empleado_ref text,p_instante timestamptz)
RETURNS text LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE autorizacion text;
BEGIN
 IF p_empleado_ref IS NULL OR p_empleado_ref !~ '^emp_[-A-Za-z0-9_]{22,128}$' OR p_instante IS NULL
    OR p_empleado_ref IS DISTINCT FROM nullif(current_setting('vec.cronos.empleado_ref',true),'') THEN
   RAISE EXCEPTION 'consulta Cronos inválida' USING ERRCODE='PC003';
 END IF;
 SELECT a.autorizacion_ref INTO autorizacion FROM vec_cronos_v1.teletrabajo_autorizacion a
 WHERE a.empleado_ref=p_empleado_ref AND a.periodo@>p_instante
   AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.teletrabajo_revocacion r
     WHERE r.autorizacion_ref=a.autorizacion_ref AND r.efectiva_en<=p_instante)
 LIMIT 2;
 IF (SELECT count(*) FROM vec_cronos_v1.teletrabajo_autorizacion a WHERE a.empleado_ref=p_empleado_ref
     AND a.periodo@>p_instante AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.teletrabajo_revocacion r
       WHERE r.autorizacion_ref=a.autorizacion_ref AND r.efectiva_en<=p_instante))<>1 THEN
   RETURN NULL;
 END IF;
 RETURN autorizacion;
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.teletrabajo_vigente_interno_v1(text,timestamptz)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
-- El consumidor antiguo de marcaje propio no acredita una autorización de
-- teletrabajo. Impedir que un material remoto se cuele por esa vía.
CREATE FUNCTION vec_cronos_v1.bloquear_marcaje_remoto_sin_consumidor_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF NEW.tipo_origen IS DISTINCT FROM 'terminal'
    OR NOT EXISTS (SELECT 1 FROM vec_cronos_v1.clasificacion_canal c
      WHERE c.tipo_origen='terminal'
        AND c.politica_version_ref=NEW.material::jsonb #>> '{canal,politica_version_ref}'
        AND c.canal_ref=NEW.material::jsonb #>> '{canal,canal_ref}'
        AND c.origen_ref=NEW.material::jsonb #>> '{canal,origen_ref}'
        AND c.calidad_ref=NEW.material::jsonb #>> '{canal,calidad_ref}') THEN
   RAISE EXCEPTION 'canal Cronos no disponible' USING ERRCODE='PC003';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.bloquear_marcaje_remoto_sin_consumidor_v1() FROM PUBLIC;
CREATE TRIGGER bloquear_remoto_sin_consumidor BEFORE INSERT ON vec_cronos_v1.marcaje_original
 FOR EACH ROW EXECUTE FUNCTION vec_cronos_v1.bloquear_marcaje_remoto_sin_consumidor_v1();
COMMIT;
