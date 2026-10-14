\set ON_ERROR_STOP on
-- CRN21: foto gobernada y consultas C10/C11. Requiere CRN19 y las siete
-- fachadas nominales de AD273. No carga miembros ni política de exportación:
-- la copia recibe datos sintéticos por un acto SQL de su propietario.
BEGIN;
CREATE ROLE vec_cronos_v1_lector_equipo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
 NOINHERIT NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:migracion:000021',0));

DO $pre$
DECLARE n text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regclass('vec_cronos_v1.marcaje_original') IS NULL
  OR to_regprocedure('vec_cronos_v1.marcajes_efectivos_v1(text)') IS NULL
  OR to_regclass('vec_cronos_v1.equipo_foto_c10') IS NOT NULL
  OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_ejecutor'
     AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CRN21: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY[
  'consumir_cronos_presencia_equipo_jefatura_v3_atestada',
  'consumir_cronos_presencia_equipo_rrhh_v3_atestada',
  'consumir_cronos_incidencias_equipo_jefatura_v3_atestada',
  'consumir_cronos_incidencias_equipo_rrhh_v3_atestada',
  'consumir_cronos_informe_propio_exportar_v3_atestada',
  'consumir_cronos_informe_equipo_jefatura_exportar_v3_atestada',
  'consumir_cronos_informe_equipo_rrhh_exportar_v3_atestada'] LOOP
  IF to_regprocedure('vec_autorizacion_atestada_v3.'||n||
      '(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
   OR NOT has_function_privilege('vec_cronos_v1_propietario',
      'vec_autorizacion_atestada_v3.'||n||
      '(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  THEN RAISE EXCEPTION 'CRN21: falta fachada %',n USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- Cada fila conserva la procedencia que debe aportar Personal/Organización.
-- La existencia de una fila no sustituye la comprobación de ese recibo en el
-- guion de provisión. Ningún ejecutor puede insertar ni leer estas tablas.
CREATE TABLE vec_cronos_v1.equipo_foto_c10 (
 version_ref text PRIMARY KEY CHECK(version_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 equipo_ref text NOT NULL CHECK(equipo_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 organismo_ref text NOT NULL CHECK(organismo_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 recibo_fuente_ref text NOT NULL CHECK(recibo_fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 desde date NOT NULL CHECK(isfinite(desde)),
 hasta_exclusivo date NOT NULL CHECK(isfinite(hasta_exclusivo) AND hasta_exclusivo>desde),
 publicada_en timestamptz(6) NOT NULL CHECK(isfinite(publicada_en)),
 UNIQUE(equipo_ref,version_ref)
);
CREATE INDEX equipo_foto_c10_vigencia_idx ON vec_cronos_v1.equipo_foto_c10
 (equipo_ref,desde,hasta_exclusivo);
CREATE TABLE vec_cronos_v1.equipo_miembro_c10 (
 version_ref text NOT NULL REFERENCES vec_cronos_v1.equipo_foto_c10,
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 empleado_etiqueta text NOT NULL CHECK(char_length(empleado_etiqueta) BETWEEN 1 AND 120
  AND empleado_etiqueta=btrim(empleado_etiqueta)
  AND empleado_etiqueta !~ '[[:cntrl:]]'),
 desde date NOT NULL CHECK(isfinite(desde)),
 hasta_exclusivo date NOT NULL CHECK(isfinite(hasta_exclusivo) AND hasta_exclusivo>desde),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 recibo_fuente_ref text NOT NULL CHECK(recibo_fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 PRIMARY KEY(version_ref,empleado_ref)
);
CREATE INDEX equipo_miembro_c10_pagina_idx ON vec_cronos_v1.equipo_miembro_c10
 (version_ref,empleado_ref,desde,hasta_exclusivo);
CREATE TABLE vec_cronos_v1.equipo_jefatura_c10 (
 version_ref text NOT NULL REFERENCES vec_cronos_v1.equipo_foto_c10,
 actor_ref text NOT NULL CHECK(actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 desde date NOT NULL CHECK(isfinite(desde)),
 hasta_exclusivo date NOT NULL CHECK(isfinite(hasta_exclusivo) AND hasta_exclusivo>desde),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 recibo_fuente_ref text NOT NULL CHECK(recibo_fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 PRIMARY KEY(version_ref,actor_ref,desde)
);
CREATE TABLE vec_cronos_v1.equipo_cobertura_c10 (
 version_ref text NOT NULL,
 empleado_ref text NOT NULL,
 fecha date NOT NULL CHECK(isfinite(fecha)),
 cobertura_completa boolean NOT NULL,
 antecedentes_completos boolean NOT NULL,
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 PRIMARY KEY(version_ref,empleado_ref,fecha),
 FOREIGN KEY(version_ref) REFERENCES vec_cronos_v1.equipo_foto_c10(version_ref)
);
CREATE TABLE vec_cronos_v1.equipo_consulta_acceso_c10 (
 decision_ref text PRIMARY KEY,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 equipo_ref text NOT NULL,
 version_ref text NOT NULL REFERENCES vec_cronos_v1.equipo_foto_c10,
 vista text NOT NULL CHECK(vista IN ('presencia','incidencias')),
 alcance text NOT NULL CHECK(alcance IN ('jefatura','rrhh')),
 desde date NOT NULL,
 hasta date NOT NULL CHECK(hasta>=desde),
 cursor text NOT NULL,
 zona_horaria text NOT NULL CHECK(zona_horaria IN ('Europe/Madrid','Atlantic/Canary')),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 consultada_en timestamptz(6) NOT NULL,
 transaccion_id bigint NOT NULL,
 backend_pid integer NOT NULL
);
CREATE TABLE vec_cronos_v1.politica_exportacion_c12 (
 politica_ref text NOT NULL CHECK(politica_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 version bigint NOT NULL CHECK(version>0),
 politica_sha256 text NOT NULL CHECK(politica_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 acto_publicacion_ref text NOT NULL CHECK(acto_publicacion_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 estado text NOT NULL CHECK(estado IN ('provisional','aprobada')),
 audiencia text NOT NULL CHECK(audiencia IN ('empleado','jefatura','rrhh')),
 tipo_informe text NOT NULL CHECK(tipo_informe IN ('saldo','movimientos','permisos')),
 formatos jsonb NOT NULL CHECK(jsonb_typeof(formatos)='array' AND jsonb_array_length(formatos)>0),
 campos jsonb NOT NULL CHECK(jsonb_typeof(campos)='array' AND jsonb_array_length(campos)>0),
 tipos_permiso jsonb NOT NULL CHECK(jsonb_typeof(tipos_permiso)='array'),
 desde timestamptz(6) NOT NULL CHECK(isfinite(desde)),
 hasta timestamptz(6) NOT NULL CHECK(isfinite(hasta) AND hasta>desde),
 publicada_en timestamptz(6) NOT NULL CHECK(isfinite(publicada_en)),
 PRIMARY KEY(politica_ref,version)
);
CREATE TABLE vec_cronos_v1.informe_preparacion_c12 (
 clave_operacion text PRIMARY KEY CHECK(clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
 preparacion_ref text NOT NULL UNIQUE CHECK(preparacion_ref ~ '^informe:preparacion:[0-9a-f-]{36}$'),
 actor_ref text NOT NULL,
 perfil_ref text NOT NULL,
 empleado_ref text NOT NULL,
 equipo_ref text,
 idioma text NOT NULL CHECK(idioma ~ '^[a-z]{2,8}(-[A-Za-z0-9]{2,8})?$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 politica_ref text NOT NULL,
 politica_version bigint NOT NULL,
 politica_sha256 text NOT NULL,
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 campos jsonb NOT NULL CHECK(jsonb_typeof(campos)='array'),
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='array' AND octet_length(datos::text)<=2097152),
 preparada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(politica_ref,politica_version)
  REFERENCES vec_cronos_v1.politica_exportacion_c12(politica_ref,version)
);
CREATE TABLE vec_cronos_v1.informe_confirmacion_c12 (
 clave_operacion text PRIMARY KEY REFERENCES vec_cronos_v1.informe_preparacion_c12,
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^informe:recibo:[0-9a-f-]{36}$'),
 preparacion_ref text NOT NULL,
 idioma text NOT NULL CHECK(idioma ~ '^[a-z]{2,8}(-[A-Za-z0-9]{2,8})?$'),
 documento_sha256 text NOT NULL CHECK(documento_sha256 ~ '^[0-9a-f]{64}$'),
 tamano bigint NOT NULL CHECK(tamano>0 AND tamano<=10485760),
 auditoria_ref text NOT NULL,
 decision_ref text NOT NULL,
 confirmada_utc timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.informe_acceso_c12 (
 decision_ref text PRIMARY KEY,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 clave_operacion text NOT NULL,
 fase text NOT NULL CHECK(fase IN ('preparacion','confirmacion')),
 actor_ref text NOT NULL,
 material_sha256 text NOT NULL,
 accedida_en timestamptz(6) NOT NULL
);

ALTER TABLE vec_cronos_v1.equipo_foto_c10 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.equipo_foto_c10 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.equipo_foto_c10 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.equipo_foto_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.equipo_foto_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.equipo_foto_c10
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.equipo_foto_c10
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.equipo_miembro_c10 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.equipo_miembro_c10 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.equipo_miembro_c10 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.equipo_miembro_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.equipo_miembro_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.equipo_miembro_c10
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.equipo_miembro_c10
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.equipo_jefatura_c10 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.equipo_jefatura_c10 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.equipo_jefatura_c10 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.equipo_jefatura_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.equipo_jefatura_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.equipo_jefatura_c10
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.equipo_jefatura_c10
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.equipo_cobertura_c10 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.equipo_cobertura_c10 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.equipo_cobertura_c10 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.equipo_cobertura_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.equipo_cobertura_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.equipo_cobertura_c10
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.equipo_cobertura_c10
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.equipo_consulta_acceso_c10 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.equipo_consulta_acceso_c10 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.equipo_consulta_acceso_c10 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.equipo_consulta_acceso_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.equipo_consulta_acceso_c10
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.equipo_consulta_acceso_c10
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.equipo_consulta_acceso_c10
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.politica_exportacion_c12 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.politica_exportacion_c12 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.politica_exportacion_c12 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.politica_exportacion_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.politica_exportacion_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.politica_exportacion_c12
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.politica_exportacion_c12
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.informe_preparacion_c12 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.informe_preparacion_c12 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.informe_preparacion_c12 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.informe_preparacion_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.informe_preparacion_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.informe_preparacion_c12
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.informe_preparacion_c12
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.informe_confirmacion_c12 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.informe_confirmacion_c12 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.informe_confirmacion_c12 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.informe_confirmacion_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.informe_confirmacion_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.informe_confirmacion_c12
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.informe_confirmacion_c12
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
ALTER TABLE vec_cronos_v1.informe_acceso_c12 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.informe_acceso_c12 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_cronos_v1.informe_acceso_c12 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.informe_acceso_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.informe_acceso_c12
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_propietario ON vec_cronos_v1.informe_acceso_c12
 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY adicion_propietario ON vec_cronos_v1.informe_acceso_c12
 FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true);
-- Lector de lote sin LOGIN ni miembros. Sus políticas no se suman a las de
-- propietario usadas por los lectores históricos. Solo la fachada sellada
-- siguiente puede asumir este rol y exige un consumo V3 de la TX actual.
GRANT USAGE ON SCHEMA vec_cronos_v1 TO vec_cronos_v1_lector_equipo;
GRANT SELECT ON vec_cronos_v1.equipo_consulta_acceso_c10,
 vec_cronos_v1.equipo_miembro_c10,vec_cronos_v1.equipo_cobertura_c10,
 vec_cronos_v1.marcaje_original,vec_cronos_v1.correccion_asiento
 TO vec_cronos_v1_lector_equipo;
CREATE POLICY lectura_lote_c10 ON vec_cronos_v1.equipo_consulta_acceso_c10
 FOR SELECT TO vec_cronos_v1_lector_equipo USING (true);
CREATE POLICY lectura_lote_c10 ON vec_cronos_v1.equipo_miembro_c10
 FOR SELECT TO vec_cronos_v1_lector_equipo USING (true);
CREATE POLICY lectura_lote_c10 ON vec_cronos_v1.equipo_cobertura_c10
 FOR SELECT TO vec_cronos_v1_lector_equipo USING (true);
CREATE POLICY lectura_lote_c10 ON vec_cronos_v1.marcaje_original
 FOR SELECT TO vec_cronos_v1_lector_equipo USING (true);
CREATE POLICY lectura_lote_c10 ON vec_cronos_v1.correccion_asiento
 FOR SELECT TO vec_cronos_v1_lector_equipo USING (true);

GRANT CREATE ON SCHEMA vec_cronos_v1 TO vec_cronos_v1_lector_equipo;
SET LOCAL ROLE vec_cronos_v1_lector_equipo;
CREATE FUNCTION vec_cronos_v1.leer_lote_equipo_c10_v1(p_decision_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
 SET statement_timeout='15s' AS $f$
DECLARE a record; total integer; ultimo text; filas jsonb;
BEGIN
 IF current_user<>'vec_cronos_v1_lector_equipo' OR session_user=current_user
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'lote de equipo denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT a FROM vec_cronos_v1.equipo_consulta_acceso_c10
  WHERE decision_ref=p_decision_ref AND transaccion_id=txid_current()
   AND backend_pid=pg_backend_pid();
 WITH pagina AS MATERIALIZED (
  SELECT DISTINCT e.empleado_ref,e.empleado_etiqueta,g.fecha::date AS fecha
   FROM vec_cronos_v1.equipo_miembro_c10 e
   CROSS JOIN LATERAL generate_series(greatest(e.desde,a.desde),
      least(e.hasta_exclusivo-1,a.hasta),'1 day'::interval) g(fecha)
   WHERE e.version_ref=a.version_ref
    AND (a.cursor='' OR (e.empleado_ref,g.fecha::date)>
      (split_part(a.cursor,'|',1),split_part(a.cursor,'|',2)::date))
   ORDER BY e.empleado_ref,g.fecha::date LIMIT 101
 ), muestra AS MATERIALIZED (
  SELECT * FROM pagina ORDER BY empleado_ref,fecha LIMIT 100
 ), hechos AS MATERIALIZED (
  SELECT p.empleado_ref,p.fecha,o.marcaje_ref AS ref,o.movimiento,o.instante_utc
   FROM muestra p JOIN vec_cronos_v1.marcaje_original o
    ON o.empleado_ref=p.empleado_ref
    AND (o.instante_utc AT TIME ZONE a.zona_horaria)::date=p.fecha
    AND o.instante_utc<=a.consultada_en
   WHERE NOT EXISTS(SELECT 1 FROM vec_cronos_v1.correccion_asiento z
     WHERE z.empleado_ref=o.empleado_ref AND z.tipo='anulacion'
      AND z.marcaje_original_ref=o.marcaje_ref)
  UNION ALL
  SELECT p.empleado_ref,p.fecha,z.asiento_ref,z.movimiento,z.instante_utc
   FROM muestra p JOIN vec_cronos_v1.correccion_asiento z
    ON z.empleado_ref=p.empleado_ref AND z.tipo='incorporacion'
    AND (z.instante_utc AT TIME ZONE a.zona_horaria)::date=p.fecha
    AND z.instante_utc<=a.consultada_en
 ), por_dia AS (
  SELECT p.empleado_ref,p.empleado_etiqueta,p.fecha,
   coalesce(jsonb_agg(jsonb_build_object('ref',h.ref,'version',1,
    'movimiento',h.movimiento,'instante_utc',h.instante_utc)
    ORDER BY h.instante_utc,h.ref) FILTER (WHERE h.ref IS NOT NULL),'[]'::jsonb) marcas,
   coalesce(b.cobertura_completa,false) cobertura_completa,
   coalesce(b.antecedentes_completos,false) antecedentes_completos
  FROM muestra p LEFT JOIN hechos h ON h.empleado_ref=p.empleado_ref AND h.fecha=p.fecha
   LEFT JOIN vec_cronos_v1.equipo_cobertura_c10 b
    ON b.version_ref=a.version_ref AND b.empleado_ref=p.empleado_ref AND b.fecha=p.fecha
  GROUP BY p.empleado_ref,p.empleado_etiqueta,p.fecha,
   b.cobertura_completa,b.antecedentes_completos
 )
 SELECT (SELECT count(*) FROM pagina),
  (SELECT p.empleado_ref||'|'||p.fecha::text FROM muestra p
   ORDER BY p.empleado_ref DESC,p.fecha DESC LIMIT 1),
  coalesce(jsonb_agg(jsonb_build_object('empleado_ref',d.empleado_ref,
   'empleado_etiqueta',d.empleado_etiqueta,
   'fecha',d.fecha::text,'cobertura_completa',d.cobertura_completa,
   'antecedentes_completos',d.antecedentes_completos,'marcajes',d.marcas)
   ORDER BY d.empleado_ref,d.fecha),'[]'::jsonb)
 INTO total,ultimo,filas FROM por_dia d;
 RETURN jsonb_build_object('filas',filas,
  'siguiente_cursor',CASE WHEN total>100 THEN ultimo ELSE '' END);
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'lote sin consumo vigente' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_cronos_v1.leer_lote_equipo_c10_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.leer_lote_equipo_c10_v1(text)
 TO vec_cronos_v1_propietario;
SET LOCAL ROLE vec_cronos_v1_propietario;
REVOKE CREATE ON SCHEMA vec_cronos_v1 FROM vec_cronos_v1_lector_equipo;

CREATE FUNCTION vec_cronos_v1.acreditar_login_equipo_c10_v1()
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF current_user<>'vec_cronos_v1_propietario' OR session_user=current_user
  OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
   AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole
   AND NOT r.rolreplication AND NOT r.rolbypassrls AND r.rolconfig IS NULL)
  OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
   AND m.roleid='vec_cronos_v1_ejecutor'::regrole AND m.inherit_option
   AND NOT m.set_option AND NOT m.admin_option)
  OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
  OR has_table_privilege(session_user,'vec_cronos_v1.equipo_foto_c10','SELECT,INSERT,UPDATE,DELETE')
 THEN RAISE EXCEPTION 'consulta de equipo denegada' USING ERRCODE='42501'; END IF;
END $f$;

-- Primer paso: solo metadatos de una foto actual para formar el recurso V3.
-- No devuelve miembros, jefatura ni marcajes.
CREATE FUNCTION vec_cronos_v1.foto_equipo_c10_v1(p_equipo_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
 SET row_security=on SET timezone='UTC' AS $f$
DECLARE f record; n integer;
BEGIN
 PERFORM vec_cronos_v1.acreditar_login_equipo_c10_v1();
 IF coalesce(p_equipo_ref,'') !~ '^[-A-Za-z0-9_.:/#]{1,128}$' THEN
  RAISE EXCEPTION 'foto de equipo inválida' USING ERRCODE='22023'; END IF;
 SELECT count(*) INTO n FROM vec_cronos_v1.equipo_foto_c10 x
  WHERE x.equipo_ref=p_equipo_ref
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=x.desde
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<x.hasta_exclusivo;
 IF n<>1 THEN RAISE EXCEPTION 'foto de equipo no acreditada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT f FROM vec_cronos_v1.equipo_foto_c10 x
  WHERE x.equipo_ref=p_equipo_ref
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=x.desde
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<x.hasta_exclusivo;
 RETURN jsonb_build_object('equipo_ref',f.equipo_ref,'unidad_ref',f.unidad_ref,
  'organismo_ref',f.organismo_ref,'version_ref',f.version_ref,
  'recibo_fuente_ref',f.recibo_fuente_ref,'fuente_ref',f.fuente_ref,
  'desde',f.desde::text,'hasta_exclusivo',f.hasta_exclusivo::text);
END $f$;

-- Solo identifica el catálogo vigente. El servidor forma el material V3 con
-- esta respuesta; los campos concretos se vuelven a cotejar al consumir.
CREATE FUNCTION vec_cronos_v1.politica_informe_c12_v1(
 p_audiencia text,p_tipo text,p_formato text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
 SET row_security=on SET timezone='UTC' AS $f$
DECLARE p record; n integer;
BEGIN
 PERFORM vec_cronos_v1.acreditar_login_equipo_c10_v1();
 IF p_audiencia NOT IN ('empleado','jefatura','rrhh')
  OR p_tipo NOT IN ('saldo','movimientos','permisos')
  OR p_formato NOT IN ('pdf','csv') THEN
  RAISE EXCEPTION 'informe no disponible' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO n FROM vec_cronos_v1.politica_exportacion_c12 x
  WHERE x.audiencia=p_audiencia AND x.tipo_informe=p_tipo
   AND x.formatos ? p_formato
   AND clock_timestamp()>=x.desde AND clock_timestamp()<x.hasta;
 IF n<>1 THEN RAISE EXCEPTION 'política de informe no acreditada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_cronos_v1.politica_exportacion_c12 x
  WHERE x.audiencia=p_audiencia AND x.tipo_informe=p_tipo
   AND x.formatos ? p_formato
   AND clock_timestamp()>=x.desde AND clock_timestamp()<x.hasta;
 RETURN jsonb_build_object('politica_ref',p.politica_ref,
  'politica_version',p.version,'politica_sha256',p.politica_sha256);
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_equipo_periodo_c10_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
 SET statement_timeout='15s' AS $f$
DECLARE
 m jsonb; c jsonb; d jsonb; x jsonb; v jsonb; foto record;
 consumo record; filas jsonb:='[]'::jsonb; lote jsonb; corte timestamptz(6);
 material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 accion text; finalidad text; audiencia text; tipo text; fachada text;
 campos jsonb; proximo text:=''; fotos_vigentes integer;
BEGIN
 PERFORM vec_cronos_v1.acreditar_login_equipo_c10_v1();
 IF current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL
  OR nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') IS NOT NULL
  OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
  OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL
  OR p_persona_version IS NULL OR p_perfil_version IS NULL
  OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'consulta de equipo denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de equipo inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
  OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'alcance','actor_ref','cursor','desde','empleado_ref','equipo_ref','equipo_version_ref',
   'fuente_ref','hasta','organismo_ref','perfil_ref','recibo_fuente_ref','unidad_ref','vista','zona_horaria']
  OR EXISTS(SELECT 1 FROM jsonb_each(m) e WHERE jsonb_typeof(e.value)<>'string')
  OR m->>'vista' NOT IN ('presencia','incidencias')
  OR m->>'alcance' NOT IN ('jefatura','rrhh')
  OR m->>'zona_horaria' NOT IN ('Europe/Madrid','Atlantic/Canary')
  OR coalesce(m->>'actor_ref','') !~ '^per_[-A-Za-z0-9_]{22,128}$'
  OR coalesce(m->>'perfil_ref','') !~ '^prf_[-A-Za-z0-9_]{22,128}$'
  OR coalesce(m->>'empleado_ref','') !~ '^emp_[-A-Za-z0-9_]{22,128}$'
  OR coalesce(m->>'cursor','') !~ '^(|emp_[-A-Za-z0-9_]{22,128}\|[0-9]{4}-[0-9]{2}-[0-9]{2})$'
 THEN RAISE EXCEPTION 'material de equipo incompatible' USING ERRCODE='42501'; END IF;
 PERFORM vec_cronos_v1.fecha_material_v1(m->>'desde');
 PERFORM vec_cronos_v1.fecha_material_v1(m->>'hasta');
 IF (m->>'hasta')::date<(m->>'desde')::date
  OR (m->>'hasta')::date-(m->>'desde')::date>31
  OR (m->>'cursor'<>''
   AND ((split_part(m->>'cursor','|',2))::date<(m->>'desde')::date
     OR (split_part(m->>'cursor','|',2))::date>(m->>'hasta')::date))
 THEN RAISE EXCEPTION 'periodo de equipo inválido' USING ERRCODE='22023'; END IF;
 material_canon:='{';
 SELECT material_canon||string_agg(to_jsonb(k)::text||':'||to_jsonb(m->>k)::text,',' ORDER BY k)||'}'
  INTO material_canon FROM jsonb_object_keys(m) k;
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material de equipo no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"equipo_ref":'||to_jsonb(m->>'equipo_ref')::text||
  ',"organismo_ref":'||to_jsonb(m->>'organismo_ref')::text||
  ',"unidad_ref":'||to_jsonb(m->>'unidad_ref')::text||
  '},"atributos":{"alcance":'||to_jsonb(m->>'alcance')::text||
  ',"cursor":'||to_jsonb(CASE WHEN m->>'cursor'='' THEN 'inicio' ELSE m->>'cursor' END)::text||
  ',"desde":'||to_jsonb(m->>'desde')::text||
  ',"equipo_version_ref":'||to_jsonb(m->>'equipo_version_ref')::text||
  ',"hasta":'||to_jsonb(m->>'hasta')::text||
  ',"material_sha256":"'||material_sha||'","vista":'||to_jsonb(m->>'vista')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 tipo:=(m->>'vista')||'_equipo';
 accion:='cronos.'||(m->>'vista')||'.equipo.consultar';
 finalidad:='consultar_'||(m->>'vista')||'_equipo';
 audiencia:='vec_cronos_v1.'||(m->>'vista')||'_equipo.consultar.v1';
 campos:=CASE m->>'vista' WHEN 'presencia'
  THEN '["agregado","causa","cobertura","empleado_etiqueta","estado","fecha","persona_ref"]'::jsonb
  ELSE '["agregado","cobertura","dias","empleado_etiqueta","periodo","persona_ref"]'::jsonb END;
 fachada:='consumir_cronos_'||(m->>'vista')||'_equipo_'||(m->>'alcance')||'_v3_atestada';
 IF d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'cronos'
  OR d->>'accion' IS DISTINCT FROM accion OR d->>'tipo_recurso' IS DISTINCT FROM tipo
  OR d->>'finalidad' IS DISTINCT FROM finalidad OR d->>'recurso_ref' IS DISTINCT FROM m->>'equipo_ref'
  OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
  OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
  OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR c->>'operacion' IS DISTINCT FROM accion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
  OR c->>'efecto_ref' IS DISTINCT FROM m->>'equipo_ref'
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha
 THEN RAISE EXCEPTION 'decisión de equipo divergente' USING ERRCODE='42501'; END IF;
 corte:=date_trunc('microseconds',clock_timestamp());
 v:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',corte);
 IF x->>'persona_version' IS DISTINCT FROM p_persona_version::text
  OR x->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 THEN RAISE EXCEPTION 'actor de equipo divergente' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO fotos_vigentes FROM vec_cronos_v1.equipo_foto_c10 f
  WHERE f.equipo_ref=m->>'equipo_ref'
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=f.desde
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<f.hasta_exclusivo;
 IF fotos_vigentes<>1 THEN RAISE EXCEPTION 'foto de equipo ambigua' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT foto FROM vec_cronos_v1.equipo_foto_c10 f
  WHERE f.version_ref=m->>'equipo_version_ref' AND f.equipo_ref=m->>'equipo_ref'
   AND f.unidad_ref=m->>'unidad_ref' AND f.organismo_ref=m->>'organismo_ref'
   AND f.fuente_ref=m->>'fuente_ref' AND f.recibo_fuente_ref=m->>'recibo_fuente_ref'
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=f.desde
   AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<f.hasta_exclusivo
   AND (m->>'desde')::date>=f.desde AND (m->>'hasta')::date<f.hasta_exclusivo;
 IF m->>'alcance'='jefatura' AND NOT EXISTS (
  SELECT 1 FROM vec_cronos_v1.equipo_jefatura_c10 j
   WHERE j.version_ref=foto.version_ref AND j.actor_ref=m->>'actor_ref'
    AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=j.desde
    AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<j.hasta_exclusivo
 ) THEN RAISE EXCEPTION 'jefatura ajena al equipo' USING ERRCODE='42501'; END IF;
 EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',fachada)
  INTO STRICT consumo USING p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
   p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM foto.equipo_ref
  OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
  OR consumo.auditoria_ref IS NULL OR consumo.consumo_huella_sha256 IS NULL
  OR clock_timestamp()>=vec_cronos_v1.vence_autorizacion_v1(c,d,x,v)
 THEN RAISE EXCEPTION 'consumo de equipo divergente' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_cronos_v1.equipo_consulta_acceso_c10
  (decision_ref,auditoria_ref,consumo_huella_sha256,actor_ref,perfil_ref,equipo_ref,
   version_ref,vista,alcance,desde,hasta,cursor,zona_horaria,material_sha256,
   consultada_en,transaccion_id,backend_pid)
 VALUES(consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
  m->>'actor_ref',m->>'perfil_ref',foto.equipo_ref,foto.version_ref,m->>'vista',
  m->>'alcance',(m->>'desde')::date,(m->>'hasta')::date,m->>'cursor',
  m->>'zona_horaria',material_sha,corte,txid_current(),pg_backend_pid());
 SELECT vec_cronos_v1.leer_lote_equipo_c10_v1(consumo.decision_ref) INTO STRICT lote;
 filas:=lote->'filas'; proximo:=lote->>'siguiente_cursor';
 IF clock_timestamp()>=vec_cronos_v1.vence_autorizacion_v1(c,d,x,v)
  OR (SELECT count(*) FROM vec_cronos_v1.equipo_foto_c10 f
    WHERE f.equipo_ref=foto.equipo_ref
     AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=f.desde
     AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<f.hasta_exclusivo)<>1
  OR NOT EXISTS(SELECT 1 FROM vec_cronos_v1.equipo_foto_c10 f
    WHERE f.version_ref=foto.version_ref
     AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=f.desde
     AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<f.hasta_exclusivo)
  OR (m->>'alcance'='jefatura' AND NOT EXISTS(
    SELECT 1 FROM vec_cronos_v1.equipo_jefatura_c10 j
    WHERE j.version_ref=foto.version_ref AND j.actor_ref=m->>'actor_ref'
      AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=j.desde
      AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<j.hasta_exclusivo))
 THEN RAISE EXCEPTION 'consulta de equipo caducada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('equipo_ref',foto.equipo_ref,'version_ref',foto.version_ref,
  'desde',m->>'desde','hasta',m->>'hasta','vista',m->>'vista',
  'alcance',m->>'alcance',
  'zona_horaria',m->>'zona_horaria','corte_utc',corte,
  'filas',filas,'siguiente_cursor',proximo);
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'foto de equipo no acreditada' USING ERRCODE='42501';
END $f$;

-- Proyección neutral para el renderer común. La política ordena las columnas;
-- no se proyectan tipo de permiso, motivo, justificante ni fuente clínica.
CREATE FUNCTION vec_cronos_v1.datos_informe_c12_v1(m jsonb,p jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' SET statement_timeout='15s' AS $f$
DECLARE empleados text[]; empleado text; fuente jsonb; datos jsonb:='[]'::jsonb;
 evidencias jsonb:='[]'::jsonb; obj jsonb; fila jsonb; j jsonb; h record;
 solicitudes record; minutos bigint; previstos bigint; estado text;
 inicio date; fin date; zona text; material_base jsonb; etiqueta text;
BEGIN
 inicio:=(m->>'desde')::date; fin:=(m->>'hasta')::date;
 zona:=m->>'zona_horaria';
 material_base:=jsonb_set(m-ARRAY['preparacion_ref','fuente_sha256',
  'documento_sha256','tamano'],'{fase}','"preparacion"'::jsonb);
 IF m->>'alcance'='empleado' THEN empleados:=ARRAY[m->>'empleado_ref'];
 ELSE
  SELECT array_agg(DISTINCT x.empleado_ref ORDER BY x.empleado_ref) INTO empleados
   FROM vec_cronos_v1.equipo_miembro_c10 x
   WHERE x.version_ref=m->>'equipo_version_ref'
    AND x.desde<=fin AND x.hasta_exclusivo>inicio;
  IF empleados IS NULL
   OR (SELECT count(DISTINCT x.empleado_ref) FROM vec_cronos_v1.equipo_miembro_c10 x
      WHERE x.version_ref=m->>'equipo_version_ref'
       AND x.desde<=inicio AND x.hasta_exclusivo>fin)<>cardinality(empleados)
  THEN RAISE EXCEPTION 'adscripción de informe incompleta' USING ERRCODE='42501'; END IF;
 END IF;
 IF cardinality(empleados) NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'informe fuera de cupo' USING ERRCODE='54000'; END IF;
 FOREACH empleado IN ARRAY empleados LOOP
  etiqueta:='';
  IF m->>'alcance'<>'empleado' THEN
   SELECT x.empleado_etiqueta INTO STRICT etiqueta
    FROM vec_cronos_v1.equipo_miembro_c10 x
    WHERE x.version_ref=m->>'equipo_version_ref' AND x.empleado_ref=empleado
     AND x.desde<=inicio AND x.hasta_exclusivo>fin;
  END IF;
  PERFORM set_config('vec.cronos.empleado_ref',empleado,true);
  IF m->>'tipo_informe'='saldo' THEN
   fuente:=vec_cronos_v1.consultar_libro_saldo_interno_v1(empleado,inicio,fin,zona);
   evidencias:=evidencias||jsonb_build_array(
    encode(sha256(convert_to(fuente::text,'UTF8')),'hex'));
   FOR j IN SELECT value FROM jsonb_array_elements(fuente->'jornadas') LOOP
    SELECT sum((z.value->>'delta_microsegundos')::bigint) INTO minutos
     FROM jsonb_array_elements(fuente->'movimientos_saldo') z
     WHERE z.value->>'tipo'='trabajado' AND z.value->>'fecha'=j->>'fecha';
    previstos:=(j->>'minutos_previstos')::bigint;
    estado:=CASE WHEN fuente->>'completo'='true' AND minutos IS NOT NULL
      AND minutos%60000000=0 THEN 'determinado' ELSE 'no_determinable' END;
    obj:=jsonb_build_object('empleado_etiqueta',etiqueta,'fecha',j->>'fecha',
     'previstos_minutos',previstos,
     'trabajados_minutos',CASE WHEN estado='determinado' THEN minutos/60000000 ELSE NULL END,
     'saldo_minutos',CASE WHEN estado='determinado' THEN minutos/60000000-previstos ELSE NULL END,
     'estado',estado);
    SELECT coalesce(jsonb_agg(obj->k.value ORDER BY k.ordinality),'[]'::jsonb) INTO fila
     FROM jsonb_array_elements_text(p->'campos') WITH ORDINALITY k(value,ordinality);
    datos:=datos||jsonb_build_array(fila);
   END LOOP;
  ELSIF m->>'tipo_informe'='movimientos' THEN
   FOR h IN SELECT z.marcaje_ref,z.movimiento,z.instante_utc
     FROM vec_cronos_v1.marcajes_efectivos_v1(empleado) z
     WHERE (z.instante_utc AT TIME ZONE zona)::date BETWEEN inicio AND fin
     ORDER BY z.instante_utc,z.marcaje_ref LOOP
    obj:=jsonb_build_object('empleado_etiqueta',etiqueta,
     'fecha',(h.instante_utc AT TIME ZONE zona)::date::text,
     'movimiento',h.movimiento,'instante_utc',h.instante_utc);
    evidencias:=evidencias||jsonb_build_array(h.marcaje_ref);
    SELECT coalesce(jsonb_agg(obj->k.value ORDER BY k.ordinality),'[]'::jsonb) INTO fila
     FROM jsonb_array_elements_text(p->'campos') WITH ORDINALITY k(value,ordinality);
    datos:=datos||jsonb_build_array(fila);
   END LOOP;
  ELSE
   FOR solicitudes IN
    SELECT s.solicitud_ref,s.desde,s.hasta,e.estado_ref,e.estado
     FROM vec_cronos_v1.permiso_solicitud s
     JOIN LATERAL (SELECT z.estado_ref,z.estado FROM vec_cronos_v1.permiso_estado z
      WHERE z.solicitud_ref=s.solicitud_ref AND z.empleado_ref=s.empleado_ref
      ORDER BY z.version DESC LIMIT 1) e ON true
     WHERE s.empleado_ref=empleado AND s.desde<=fin AND s.hasta>=inicio
     ORDER BY s.desde,s.solicitud_ref LOOP
    obj:=jsonb_build_object('empleado_etiqueta',etiqueta,'desde',solicitudes.desde::text,
     'hasta',solicitudes.hasta::text,'estado',solicitudes.estado);
    evidencias:=evidencias||jsonb_build_array(
     solicitudes.solicitud_ref||'/'||solicitudes.estado_ref);
    SELECT coalesce(jsonb_agg(obj->k.value ORDER BY k.ordinality),'[]'::jsonb) INTO fila
     FROM jsonb_array_elements_text(p->'campos') WITH ORDINALITY k(value,ordinality);
    datos:=datos||jsonb_build_array(fila);
   END LOOP;
  END IF;
  PERFORM set_config('vec.cronos.empleado_ref','',true);
  IF jsonb_array_length(datos)>5000 THEN RAISE EXCEPTION 'informe fuera de cupo' USING ERRCODE='54000'; END IF;
 END LOOP;
 RETURN jsonb_build_object('datos',datos,
  'fuente_sha256',encode(sha256(convert_to(jsonb_build_object(
   'material',material_base,'politica_sha256',p->>'politica_sha256',
   'datos',datos,'evidencias',evidencias)::text,'UTF8')),'hex'));
END $f$;

CREATE FUNCTION vec_cronos_v1.consumir_informe_c12_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
 SET statement_timeout='15s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; v jsonb; p record; f record;
 canon text; sha text; contexto text; huella text; alcance text; tipo text;
 accion text; audiencia text; fachada text; recurso text; consumo record;
 n integer; campo text; permitidos text[]; ambitos jsonb; atributos jsonb;
BEGIN
 PERFORM vec_cronos_v1.acreditar_login_equipo_c10_v1();
 IF current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL
  OR nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') IS NOT NULL
  OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 8192
  OR p_capacidad IS NULL OR p_decision IS NULL OR p_contexto IS NULL
  OR p_persona_version IS NULL OR p_perfil_version IS NULL
  OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'informe denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de informe inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m)) NOT IN (15,19,21,25)
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('actor_ref','alcance','clave_operacion','desde','empleado_ref','formato','hasta',
    'perfil_ref','politica_ref','politica_sha256','politica_version','tipo_informe',
    'equipo_ref','unidad_ref','organismo_ref','equipo_version_ref','recibo_fuente_ref','fuente_ref',
    'zona_horaria','idioma','fase','preparacion_ref','fuente_sha256','documento_sha256','tamano'))
  OR m->>'alcance' NOT IN ('empleado','jefatura','rrhh')
  OR m->>'tipo_informe' NOT IN ('saldo','movimientos','permisos')
  OR m->>'formato' NOT IN ('pdf','csv')
  OR m->>'zona_horaria' NOT IN ('Europe/Madrid','Atlantic/Canary')
  OR coalesce(m->>'idioma','') !~ '^[a-z]{2,8}(-[A-Za-z0-9]{2,8})?$'
  OR m->>'fase' NOT IN ('preparacion','confirmacion')
  OR coalesce(m->>'actor_ref','') !~ '^per_[-A-Za-z0-9_]{22,128}$'
  OR coalesce(m->>'perfil_ref','') !~ '^prf_[-A-Za-z0-9_]{22,128}$'
  OR coalesce(m->>'empleado_ref','') !~ '^emp_[-A-Za-z0-9_]{22,128}$'
  OR coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
  OR coalesce(m->>'politica_sha256','') !~ '^[0-9a-f]{64}$'
  OR coalesce(m->>'politica_version','') !~ '^[1-9][0-9]{0,18}$'
  OR (m->>'fase'='preparacion' AND m->>'alcance'='empleado'
      AND (SELECT count(*) FROM jsonb_object_keys(m))<>15)
  OR (m->>'fase'='preparacion' AND m->>'alcance'<>'empleado'
      AND (SELECT count(*) FROM jsonb_object_keys(m))<>21)
  OR (m->>'fase'='confirmacion' AND m->>'alcance'='empleado'
      AND (SELECT count(*) FROM jsonb_object_keys(m))<>19)
  OR (m->>'fase'='confirmacion' AND m->>'alcance'<>'empleado'
      AND (SELECT count(*) FROM jsonb_object_keys(m))<>25)
  OR (m->>'fase'='confirmacion' AND
      (coalesce(m->>'preparacion_ref','') !~ '^informe:preparacion:[0-9a-f-]{36}$'
       OR coalesce(m->>'fuente_sha256','') !~ '^[0-9a-f]{64}$'
       OR coalesce(m->>'documento_sha256','') !~ '^[0-9a-f]{64}$'
       OR coalesce(m->>'tamano','') !~ '^[1-9][0-9]{0,7}$'))
 THEN RAISE EXCEPTION 'contrato de informe incompatible' USING ERRCODE='42501'; END IF;
 PERFORM vec_cronos_v1.fecha_material_v1(m->>'desde');
 PERFORM vec_cronos_v1.fecha_material_v1(m->>'hasta');
 IF (m->>'hasta')::date<(m->>'desde')::date
  OR (m->>'hasta')::date-(m->>'desde')::date>31
 THEN RAISE EXCEPTION 'periodo de informe inválido' USING ERRCODE='22023'; END IF;
 SELECT '{'||string_agg(to_jsonb(k)::text||':'||(m->k)::text,',' ORDER BY k)||'}'
  INTO canon FROM jsonb_object_keys(m) k;
 IF p_material IS DISTINCT FROM canon THEN
  RAISE EXCEPTION 'material de informe no canónico' USING ERRCODE='22023'; END IF;
 sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 alcance:=m->>'alcance'; tipo:=CASE WHEN alcance='empleado' THEN 'informe_propio' ELSE 'informe_equipo' END;
 accion:=CASE WHEN alcance='empleado' THEN 'cronos.informe.propio.exportar'
  ELSE 'cronos.informe.equipo.exportar' END;
 audiencia:=CASE WHEN alcance='empleado' THEN 'vec_cronos_v1.informe_propio.exportar.v1'
  ELSE 'vec_cronos_v1.informe_equipo.exportar.v1' END;
 fachada:=CASE alcance WHEN 'empleado' THEN 'consumir_cronos_informe_propio_exportar_v3_atestada'
  WHEN 'jefatura' THEN 'consumir_cronos_informe_equipo_jefatura_exportar_v3_atestada'
  ELSE 'consumir_cronos_informe_equipo_rrhh_exportar_v3_atestada' END;
 recurso:=CASE WHEN alcance='empleado' THEN m->>'empleado_ref' ELSE m->>'equipo_ref' END;
 ambitos:=CASE WHEN alcance='empleado'
  THEN jsonb_build_object('empleado_ref',recurso)
  ELSE jsonb_build_object('equipo_ref',recurso,
    'unidad_ref',m->>'unidad_ref','organismo_ref',m->>'organismo_ref') END;
 atributos:=jsonb_build_object('alcance',alcance,'desde',m->>'desde',
  'fase',m->>'fase','formato',m->>'formato','hasta',m->>'hasta',
  'idioma',m->>'idioma',
  'material_sha256',sha,'politica_sha256',m->>'politica_sha256',
  'tipo_informe',m->>'tipo_informe','zona_horaria',m->>'zona_horaria');
 IF alcance<>'empleado' THEN
  atributos:=atributos||jsonb_build_object('equipo_version_ref',m->>'equipo_version_ref');
 END IF;
 IF m->>'fase'='confirmacion' THEN
  atributos:=atributos||jsonb_build_object(
   'preparacion_ref',m->>'preparacion_ref','fuente_sha256',m->>'fuente_sha256',
   'documento_sha256',m->>'documento_sha256','tamano',m->>'tamano');
 END IF;
 contexto:='{"ambitos":{'||
  (SELECT string_agg(to_jsonb(e.k)::text||':'||e.v::text,',' ORDER BY e.k)
   FROM jsonb_each(ambitos) e(k,v))||'},"atributos":{'||
  (SELECT string_agg(to_jsonb(e.k)::text||':'||e.v::text,',' ORDER BY e.k)
   FROM jsonb_each(atributos) e(k,v))||'}}';
 huella:=encode(sha256(convert_to(contexto,'UTF8')),'hex');
 IF d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'cronos'
  OR d->>'accion' IS DISTINCT FROM accion OR d->>'tipo_recurso' IS DISTINCT FROM tipo
  OR d->>'finalidad' IS DISTINCT FROM
   (CASE WHEN alcance='empleado' THEN 'exportar_informe_propio' ELSE 'exportar_informe_equipo' END)
  OR d->>'recurso_ref' IS DISTINCT FROM recurso
  OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
  OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
  OR d->'campos_permitidos' IS DISTINCT FROM '["archivo","recibo"]'::jsonb
  OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR c->>'operacion' IS DISTINCT FROM accion
  OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
  OR c->>'efecto_ref' IS DISTINCT FROM recurso
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'decisión de informe divergente' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO n FROM vec_cronos_v1.politica_exportacion_c12 z
  WHERE z.audiencia=alcance AND z.tipo_informe=m->>'tipo_informe'
   AND z.formatos ? (m->>'formato')
   AND clock_timestamp()>=z.desde AND clock_timestamp()<z.hasta;
 IF n<>1 THEN RAISE EXCEPTION 'política de informe ambigua' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO n FROM vec_cronos_v1.politica_exportacion_c12 z
  WHERE z.politica_ref=m->>'politica_ref' AND z.version=(m->>'politica_version')::bigint
   AND z.politica_sha256=m->>'politica_sha256' AND z.audiencia=alcance
   AND z.tipo_informe=m->>'tipo_informe' AND z.formatos ? (m->>'formato')
   AND clock_timestamp()>=z.desde AND clock_timestamp()<z.hasta;
 IF n<>1 THEN RAISE EXCEPTION 'política de informe caducada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_cronos_v1.politica_exportacion_c12 z
  WHERE z.politica_ref=m->>'politica_ref' AND z.version=(m->>'politica_version')::bigint;
 permitidos:=CASE m->>'tipo_informe'
  WHEN 'saldo' THEN ARRAY['empleado_etiqueta','fecha','previstos_minutos','trabajados_minutos','saldo_minutos','estado']
  WHEN 'movimientos' THEN ARRAY['empleado_etiqueta','fecha','movimiento','instante_utc']
  ELSE ARRAY['empleado_etiqueta','desde','hasta','estado'] END;
 IF jsonb_array_length(p.campos)>cardinality(permitidos)
  OR (alcance='empleado' AND p.campos ? 'empleado_etiqueta')
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(p.campos) e
    WHERE jsonb_typeof(e.value)<>'string'
      OR NOT ((e.value#>>'{}')=ANY(permitidos)))
  OR (SELECT count(*) FROM jsonb_array_elements_text(p.campos)) <>
     (SELECT count(DISTINCT value) FROM jsonb_array_elements_text(p.campos))
  OR (p.formatos <@ '["pdf","csv"]'::jsonb) IS NOT TRUE
  OR p.tipos_permiso<>'[]'::jsonb
 THEN RAISE EXCEPTION 'campos de informe no admitidos' USING ERRCODE='42501'; END IF;
 v:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',
  m->>'empleado_ref',clock_timestamp());
 IF x->>'persona_version' IS DISTINCT FROM p_persona_version::text
  OR x->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 THEN RAISE EXCEPTION 'actor de informe divergente' USING ERRCODE='42501'; END IF;
 IF alcance<>'empleado' THEN
  SELECT count(*) INTO n FROM vec_cronos_v1.equipo_foto_c10 z
   WHERE z.equipo_ref=recurso
    AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=z.desde
    AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<z.hasta_exclusivo;
  IF n<>1 THEN RAISE EXCEPTION 'foto de informe ambigua' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT f FROM vec_cronos_v1.equipo_foto_c10 z
   WHERE z.equipo_ref=recurso AND z.version_ref=m->>'equipo_version_ref'
    AND z.unidad_ref=m->>'unidad_ref' AND z.organismo_ref=m->>'organismo_ref'
    AND z.fuente_ref=m->>'fuente_ref' AND z.recibo_fuente_ref=m->>'recibo_fuente_ref'
    AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=z.desde
    AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<z.hasta_exclusivo
    AND (m->>'desde')::date>=z.desde AND (m->>'hasta')::date<z.hasta_exclusivo;
  IF alcance='jefatura' AND NOT EXISTS(
   SELECT 1 FROM vec_cronos_v1.equipo_jefatura_c10 j
    WHERE j.version_ref=f.version_ref AND j.actor_ref=m->>'actor_ref'
     AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date>=j.desde
     AND (clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date<j.hasta_exclusivo)
  THEN RAISE EXCEPTION 'informe ajeno a jefatura' USING ERRCODE='42501'; END IF;
 END IF;
 EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',fachada)
  INTO STRICT consumo USING p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
   p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM recurso
  OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella
  OR clock_timestamp()>=vec_cronos_v1.vence_autorizacion_v1(c,d,x,v)
  OR clock_timestamp()>=p.hasta
 THEN RAISE EXCEPTION 'consumo de informe caducado' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('material',m,'politica',to_jsonb(p),
  'material_sha256',sha,'decision_ref',consumo.decision_ref,
  'auditoria_ref',consumo.auditoria_ref,
  'consumo_huella_sha256',consumo.consumo_huella_sha256,
  'vence_en',vec_cronos_v1.vence_autorizacion_v1(c,d,x,v));
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'informe sin fuente acreditada' USING ERRCODE='42501';
END $f$;

CREATE FUNCTION vec_cronos_v1.comprobar_fuentes_informe_c12_v1(m jsonb)
RETURNS void LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog
 SET row_security=on SET timezone='UTC' AS $f$
DECLARE n integer; fecha date;
BEGIN
 fecha:=(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date;
 SELECT count(*) INTO n FROM vec_cronos_v1.politica_exportacion_c12 p
  WHERE p.audiencia=m->>'alcance' AND p.tipo_informe=m->>'tipo_informe'
   AND p.formatos ? (m->>'formato')
   AND clock_timestamp()>=p.desde AND clock_timestamp()<p.hasta;
 IF n<>1 OR NOT EXISTS(
  SELECT 1 FROM vec_cronos_v1.politica_exportacion_c12 p
  WHERE p.politica_ref=m->>'politica_ref'
   AND p.version=(m->>'politica_version')::bigint
   AND p.politica_sha256=m->>'politica_sha256'
   AND clock_timestamp()>=p.desde AND clock_timestamp()<p.hasta)
 THEN RAISE EXCEPTION 'política de informe cambiada' USING ERRCODE='42501'; END IF;
 IF m->>'alcance'<>'empleado' THEN
  SELECT count(*) INTO n FROM vec_cronos_v1.equipo_foto_c10 f
   WHERE f.equipo_ref=m->>'equipo_ref'
    AND fecha>=f.desde AND fecha<f.hasta_exclusivo;
  IF n<>1 OR NOT EXISTS(
   SELECT 1 FROM vec_cronos_v1.equipo_foto_c10 f
    WHERE f.version_ref=m->>'equipo_version_ref' AND f.equipo_ref=m->>'equipo_ref'
     AND fecha>=f.desde AND fecha<f.hasta_exclusivo)
   OR (m->>'alcance'='jefatura' AND NOT EXISTS(
    SELECT 1 FROM vec_cronos_v1.equipo_jefatura_c10 j
     WHERE j.version_ref=m->>'equipo_version_ref' AND j.actor_ref=m->>'actor_ref'
      AND fecha>=j.desde AND fecha<j.hasta_exclusivo))
  THEN RAISE EXCEPTION 'ámbito de informe cambiado' USING ERRCODE='42501'; END IF;
 END IF;
END $f$;

CREATE FUNCTION vec_cronos_v1.preparar_informe_c12_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
 SET statement_timeout='15s' AS $f$
DECLARE a jsonb; m jsonb; p jsonb; salida jsonb; previa record; ahora timestamptz(6);
BEGIN
 a:=vec_cronos_v1.consumir_informe_c12_v1(p_material,p_capacidad,p_decision,
  p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 m:=a->'material'; p:=a->'politica';
 IF m->>'fase' IS DISTINCT FROM 'preparacion' THEN
  RAISE EXCEPTION 'fase de informe inválida' USING ERRCODE='42501'; END IF;
 PERFORM vec_cronos_v1.comprobar_fuentes_informe_c12_v1(m);
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF ahora>=(a->>'vence_en')::timestamptz OR ahora>=(p->>'hasta')::timestamptz
 THEN RAISE EXCEPTION 'preparación caducada' USING ERRCODE='42501'; END IF;
 SELECT * INTO previa FROM vec_cronos_v1.informe_preparacion_c12 z
  WHERE z.clave_operacion=m->>'clave_operacion';
 IF NOT FOUND THEN
  salida:=vec_cronos_v1.datos_informe_c12_v1(m,p);
  PERFORM vec_cronos_v1.comprobar_fuentes_informe_c12_v1(m);
  ahora:=date_trunc('microseconds',clock_timestamp());
  IF ahora>=(a->>'vence_en')::timestamptz OR ahora>=(p->>'hasta')::timestamptz
  THEN RAISE EXCEPTION 'preparación caducada' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_cronos_v1.informe_preparacion_c12(
   clave_operacion,preparacion_ref,actor_ref,perfil_ref,empleado_ref,equipo_ref,idioma,
   material_sha256,politica_ref,politica_version,politica_sha256,fuente_sha256,
   campos,datos,preparada_en)
  VALUES(m->>'clave_operacion','informe:preparacion:'||gen_random_uuid()::text,
   m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',m->>'equipo_ref',m->>'idioma',
   a->>'material_sha256',m->>'politica_ref',(m->>'politica_version')::bigint,
   m->>'politica_sha256',salida->>'fuente_sha256',p->'campos',salida->'datos',ahora)
  ON CONFLICT (clave_operacion) DO NOTHING;
 END IF;
 SELECT * INTO STRICT previa FROM vec_cronos_v1.informe_preparacion_c12 z
  WHERE z.clave_operacion=m->>'clave_operacion';
 IF previa.actor_ref IS DISTINCT FROM m->>'actor_ref'
  OR previa.perfil_ref IS DISTINCT FROM m->>'perfil_ref'
  OR previa.empleado_ref IS DISTINCT FROM m->>'empleado_ref'
  OR previa.equipo_ref IS DISTINCT FROM m->>'equipo_ref'
  OR previa.idioma IS DISTINCT FROM m->>'idioma'
  OR previa.material_sha256 IS DISTINCT FROM a->>'material_sha256'
  OR previa.politica_ref IS DISTINCT FROM m->>'politica_ref'
  OR previa.politica_version IS DISTINCT FROM (m->>'politica_version')::bigint
  OR previa.politica_sha256 IS DISTINCT FROM m->>'politica_sha256'
  OR previa.campos IS DISTINCT FROM p->'campos'
 THEN RAISE EXCEPTION 'clave de informe usada con otro material' USING ERRCODE='23505'; END IF;
 INSERT INTO vec_cronos_v1.informe_acceso_c12(
  decision_ref,auditoria_ref,consumo_huella_sha256,clave_operacion,fase,
  actor_ref,material_sha256,accedida_en)
 VALUES(a->>'decision_ref',a->>'auditoria_ref',a->>'consumo_huella_sha256',
  m->>'clave_operacion','preparacion',m->>'actor_ref',a->>'material_sha256',ahora);
 RETURN jsonb_build_object('politica_ref',m->>'politica_ref',
  'politica_version',(m->>'politica_version')::bigint,
  'politica_sha256',m->>'politica_sha256',
  'tipo_informe',m->>'tipo_informe','formato',m->>'formato',
  'idioma',previa.idioma,
  'empleado_ref',m->>'empleado_ref','equipo_ref',m->>'equipo_ref',
  'desde',m->>'desde','hasta',m->>'hasta',
  'campos',previa.campos,'datos',previa.datos,
  'fuente_sha256',previa.fuente_sha256,
  'preparacion_ref',previa.preparacion_ref);
END $f$;

CREATE FUNCTION vec_cronos_v1.confirmar_informe_c12_v1(
 p_material text,p_documento_sha256 text,p_tamano bigint,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp
 SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
 SET statement_timeout='15s' AS $f$
DECLARE a jsonb; m jsonb; p jsonb; salida jsonb; previa record; confirmado record;
 ahora timestamptz(6); base jsonb; base_canon text; base_sha text;
BEGIN
 IF coalesce(p_documento_sha256,'') !~ '^[0-9a-f]{64}$'
  OR p_tamano NOT BETWEEN 1 AND 10485760 THEN
  RAISE EXCEPTION 'documento de informe inválido' USING ERRCODE='22023'; END IF;
 a:=vec_cronos_v1.consumir_informe_c12_v1(p_material,p_capacidad,p_decision,
  p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 m:=a->'material'; p:=a->'politica';
 IF m->>'fase' IS DISTINCT FROM 'confirmacion'
  OR m->>'documento_sha256' IS DISTINCT FROM p_documento_sha256
  OR (m->>'tamano')::bigint IS DISTINCT FROM p_tamano
 THEN RAISE EXCEPTION 'documento no ligado a autorización' USING ERRCODE='42501'; END IF;
 base:=jsonb_set(m-ARRAY['preparacion_ref','fuente_sha256',
  'documento_sha256','tamano'],'{fase}','"preparacion"'::jsonb);
 SELECT '{'||string_agg(to_jsonb(k)::text||':'||(base->k)::text,',' ORDER BY k)||'}'
  INTO base_canon FROM jsonb_object_keys(base) k;
 base_sha:=encode(sha256(convert_to(base_canon,'UTF8')),'hex');
 SELECT * INTO STRICT previa FROM vec_cronos_v1.informe_preparacion_c12 z
  WHERE z.clave_operacion=m->>'clave_operacion';
 IF previa.material_sha256 IS DISTINCT FROM base_sha
  OR previa.preparacion_ref IS DISTINCT FROM m->>'preparacion_ref'
  OR previa.idioma IS DISTINCT FROM m->>'idioma'
  OR previa.fuente_sha256 IS DISTINCT FROM m->>'fuente_sha256'
  OR previa.politica_ref IS DISTINCT FROM m->>'politica_ref'
  OR previa.politica_version IS DISTINCT FROM (m->>'politica_version')::bigint
  OR previa.politica_sha256 IS DISTINCT FROM m->>'politica_sha256'
 THEN RAISE EXCEPTION 'fuente o permiso de informe cambiado' USING ERRCODE='42501'; END IF;
 SELECT * INTO confirmado FROM vec_cronos_v1.informe_confirmacion_c12 z
  WHERE z.clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
  IF confirmado.preparacion_ref IS DISTINCT FROM previa.preparacion_ref
   OR confirmado.idioma IS DISTINCT FROM m->>'idioma'
   OR confirmado.documento_sha256 IS DISTINCT FROM p_documento_sha256
   OR confirmado.tamano IS DISTINCT FROM p_tamano
  THEN RAISE EXCEPTION 'clave de descarga usada con otro documento' USING ERRCODE='23505'; END IF;
  PERFORM vec_cronos_v1.comprobar_fuentes_informe_c12_v1(m);
  ahora:=date_trunc('microseconds',clock_timestamp());
  IF ahora>=(a->>'vence_en')::timestamptz OR ahora>=(p->>'hasta')::timestamptz
  THEN RAISE EXCEPTION 'recuperación de informe caducada' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_cronos_v1.informe_acceso_c12(
   decision_ref,auditoria_ref,consumo_huella_sha256,clave_operacion,fase,
   actor_ref,material_sha256,accedida_en)
  VALUES(a->>'decision_ref',a->>'auditoria_ref',a->>'consumo_huella_sha256',
   m->>'clave_operacion','confirmacion',m->>'actor_ref',a->>'material_sha256',ahora);
  RETURN jsonb_build_object('recibo_ref',confirmado.recibo_ref,
   'auditoria_ref',confirmado.auditoria_ref,
   'preparacion_ref',confirmado.preparacion_ref,
   'documento_sha256',confirmado.documento_sha256,
   'tamano',confirmado.tamano,'confirmada_utc',confirmado.confirmada_utc);
 END IF;
 salida:=vec_cronos_v1.datos_informe_c12_v1(m,p);
 PERFORM vec_cronos_v1.comprobar_fuentes_informe_c12_v1(m);
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF previa.fuente_sha256 IS DISTINCT FROM salida->>'fuente_sha256'
  OR ahora>=(a->>'vence_en')::timestamptz OR ahora>=(p->>'hasta')::timestamptz
 THEN RAISE EXCEPTION 'fuente o permiso de informe cambiado' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_cronos_v1.informe_acceso_c12(
  decision_ref,auditoria_ref,consumo_huella_sha256,clave_operacion,fase,
  actor_ref,material_sha256,accedida_en)
 VALUES(a->>'decision_ref',a->>'auditoria_ref',a->>'consumo_huella_sha256',
  m->>'clave_operacion','confirmacion',m->>'actor_ref',a->>'material_sha256',ahora);
 INSERT INTO vec_cronos_v1.informe_confirmacion_c12(
  clave_operacion,recibo_ref,preparacion_ref,idioma,documento_sha256,tamano,
  auditoria_ref,decision_ref,confirmada_utc)
 VALUES(m->>'clave_operacion','informe:recibo:'||gen_random_uuid()::text,
  previa.preparacion_ref,m->>'idioma',p_documento_sha256,p_tamano,
  a->>'auditoria_ref',a->>'decision_ref',ahora)
 ON CONFLICT (clave_operacion) DO NOTHING;
 SELECT * INTO STRICT confirmado FROM vec_cronos_v1.informe_confirmacion_c12 z
  WHERE z.clave_operacion=m->>'clave_operacion';
 IF confirmado.preparacion_ref IS DISTINCT FROM previa.preparacion_ref
  OR confirmado.idioma IS DISTINCT FROM m->>'idioma'
  OR confirmado.documento_sha256 IS DISTINCT FROM p_documento_sha256
  OR confirmado.tamano IS DISTINCT FROM p_tamano
 THEN RAISE EXCEPTION 'clave de descarga usada con otro documento' USING ERRCODE='23505'; END IF;
 RETURN jsonb_build_object('recibo_ref',confirmado.recibo_ref,
  'auditoria_ref',confirmado.auditoria_ref,
  'preparacion_ref',confirmado.preparacion_ref,
  'documento_sha256',confirmado.documento_sha256,
  'tamano',confirmado.tamano,'confirmada_utc',confirmado.confirmada_utc);
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'preparación de informe ausente' USING ERRCODE='42501';
END $f$;

REVOKE ALL ON FUNCTION vec_cronos_v1.acreditar_login_equipo_c10_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.foto_equipo_c10_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.politica_informe_c12_v1(text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.consultar_equipo_periodo_c10_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.datos_informe_c12_v1(jsonb,jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.comprobar_fuentes_informe_c12_v1(jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.consumir_informe_c12_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.preparar_informe_c12_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_cronos_v1.confirmar_informe_c12_v1(
 text,text,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.foto_equipo_c10_v1(text) TO vec_cronos_v1_ejecutor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.politica_informe_c12_v1(text,text,text)
 TO vec_cronos_v1_ejecutor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.consultar_equipo_periodo_c10_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_ejecutor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.preparar_informe_c12_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_cronos_v1_ejecutor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.confirmar_informe_c12_v1(
 text,text,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_cronos_v1_ejecutor;
COMMIT;
