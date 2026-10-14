\set ON_ERROR_STOP on
-- CRN23: metadatos históricos de la programación propia C4.
-- La lectura reutiliza la decisión nominal de saldo, su consumo y auditoría
-- en la misma transacción. La carga exige un acto de gobierno fuera de runtime.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:migracion:000023:programacion-laboral-c4',0));
DO $pre$
BEGIN
 IF current_user<>'vec_cronos_v1_propietario'
  OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regclass('vec_cronos_v1.programacion_jornada') IS NULL
  OR to_regclass('vec_cronos_v1.programacion_contexto_c4') IS NOT NULL
  OR to_regprocedure('vec_cronos_v1.consultar_saldo_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_cronos_v1.consultar_programacion_laboral_c4_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class t WHERE t.oid='vec_cronos_v1.programacion_jornada'::regclass
       AND t.relowner=current_user::regrole AND t.relrowsecurity AND t.relforcerowsecurity)
  OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       WHERE p.oid='vec_cronos_v1.consultar_saldo_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner=current_user::regrole AND p.prosecdef)
 THEN RAISE EXCEPTION 'CRN23: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

ALTER TABLE vec_cronos_v1.programacion_jornada
 ADD CONSTRAINT programacion_jornada_c4_vinculo_unico UNIQUE(programacion_ref,empleado_ref,fecha,version);

CREATE TABLE vec_cronos_v1.programacion_contexto_c4 (
 programacion_ref text PRIMARY KEY REFERENCES vec_cronos_v1.programacion_jornada(programacion_ref),
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 fecha date NOT NULL CHECK(isfinite(fecha)),
 version integer NOT NULL CHECK(version>0),
 relacion_ref text NOT NULL CHECK(relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'),
 centro_ref text NOT NULL CHECK(centro_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 regimen_ref text NOT NULL CHECK(regimen_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 inicio_utc timestamptz(6) NOT NULL CHECK(isfinite(inicio_utc)),
 fin_utc timestamptz(6) NOT NULL CHECK(isfinite(fin_utc)),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[-A-Za-z0-9_.:]{1,255}$'),
 fuente_version integer NOT NULL CHECK(fuente_version>0),
 publicada_en timestamptz(6) NOT NULL CHECK(isfinite(publicada_en)),
 CHECK(fin_utc>inicio_utc AND fin_utc<=inicio_utc+interval '24 hours'),
 UNIQUE(empleado_ref,fecha,version),
 FOREIGN KEY(programacion_ref,empleado_ref,fecha,version)
  REFERENCES vec_cronos_v1.programacion_jornada(programacion_ref,empleado_ref,fecha,version)
);
CREATE INDEX programacion_contexto_c4_empleado_fecha_idx ON vec_cronos_v1.programacion_contexto_c4(empleado_ref,fecha);
ALTER TABLE vec_cronos_v1.programacion_contexto_c4 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.programacion_contexto_c4 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_propia ON vec_cronos_v1.programacion_contexto_c4 FOR SELECT TO vec_cronos_v1_propietario
 USING(empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_propia ON vec_cronos_v1.programacion_contexto_c4 FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK(empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.programacion_contexto_c4
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.programacion_contexto_c4 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
REVOKE ALL ON TYPE vec_cronos_v1.programacion_contexto_c4 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;

CREATE FUNCTION vec_cronos_v1.consultar_programacion_laboral_c4_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
 SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb; saldo jsonb; jornada jsonb; p vec_cronos_v1.programacion_contexto_c4%ROWTYPE;
 fecha_consulta date; empleado text; filas integer;
BEGIN
 -- La función existente acredita actor/empleado, consume V3 una sola vez,
 -- registra el acceso y deja la RLS limitada a ese empleado en esta TX.
 saldo:=vec_cronos_v1.consultar_saldo_propio_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 m:=p_material::jsonb;
 empleado:=m->>'empleado_ref'; fecha_consulta:=(m->>'desde')::date;
 IF m->>'desde' IS DISTINCT FROM m->>'hasta' OR fecha_consulta::text IS DISTINCT FROM m->>'desde'
  OR empleado IS DISTINCT FROM nullif(current_setting('vec.cronos.empleado_ref',true),'')
  OR saldo->>'empleado_ref' IS DISTINCT FROM empleado
  OR saldo->>'desde' IS DISTINCT FROM fecha_consulta::text OR saldo->>'hasta' IS DISTINCT FROM fecha_consulta::text
  OR jsonb_typeof(saldo->'jornadas') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'CRN23: lectura divergente' USING ERRCODE='PC003'; END IF;
 filas:=jsonb_array_length(saldo->'jornadas');
 IF filas<>1 THEN RETURN jsonb_build_object('estado','no_determinable','fecha',fecha_consulta); END IF;
 jornada:=saldo->'jornadas'->0;
 SELECT c.* INTO p FROM vec_cronos_v1.programacion_contexto_c4 c
  WHERE c.programacion_ref=jornada->>'programacion_ref' AND c.empleado_ref=empleado AND c.fecha=fecha_consulta
   AND c.version=(jornada->>'version')::integer AND c.publicada_en<=clock_timestamp()
   AND EXISTS (SELECT 1 FROM vec_cronos_v1.programacion_jornada j
     WHERE j.programacion_ref=c.programacion_ref AND j.empleado_ref=c.empleado_ref
       AND j.fecha=c.fecha AND j.version=c.version AND j.publicada_en<=c.publicada_en
       AND j.fuente_ref=c.fuente_ref);
 IF NOT FOUND OR p.fuente_ref IS DISTINCT FROM jornada->>'fuente_ref'
  OR (p.inicio_utc AT TIME ZONE (jornada->>'zona_horaria'))::date IS DISTINCT FROM fecha_consulta
 THEN RETURN jsonb_build_object('estado','no_determinable','fecha',fecha_consulta); END IF;
 RETURN jsonb_build_object('estado','programado','fecha',fecha_consulta,'empleado_ref',empleado,
  'programacion_ref',p.programacion_ref,'version',p.version,'relacion_ref',p.relacion_ref,
  'centro_ref',p.centro_ref,'regimen_ref',p.regimen_ref,'inicio_utc',p.inicio_utc,'fin_utc',p.fin_utc,
  'turno_ref',jornada->>'turno_ref','politica_version_ref',jornada->>'politica_version_ref',
  'minutos_programados',(jornada->>'minutos_previstos')::integer,'fuente_ref',p.fuente_ref,
  'fuente_version',p.fuente_version,'publicada_en',p.publicada_en);
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.consultar_programacion_laboral_c4_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_cronos_v1_migrador;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.consultar_programacion_laboral_c4_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_cronos_v1_ejecutor;
COMMIT;
