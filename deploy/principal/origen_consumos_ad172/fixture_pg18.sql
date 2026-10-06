\set ON_ERROR_STOP on
-- Fixture estructural sintético para probar el guion en PostgreSQL 18.4
-- desechable, generado desde la misma lista (variable psql `ternas`, la lista
-- completa). La tabla, la política y el resolutor AD172 se cargan después,
-- copiados literalmente de su migración por probar_pg18.sh.
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_adm_lector_sintetico LOGIN;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
CREATE TEMP TABLE lista AS
SELECT c[2] AS login, c[3] AS grupo, c[4] AS aud, c[5] AS op, c[8] AS perfil
  FROM regexp_split_to_table(:'ternas', E'\n') AS l(linea), LATERAL string_to_array(l.linea, E'\t') AS c
 WHERE l.linea <> '' AND left(l.linea, 1) <> '#';
GRANT SELECT ON lista TO vec_autorizacion_atestada_v3_propietario;

DO $r$
DECLARE x record;
BEGIN
  FOR x IN SELECT DISTINCT grupo FROM lista LOOP
    EXECUTE format('CREATE ROLE %I NOLOGIN', x.grupo);
  END LOOP;
  FOR x IN SELECT DISTINCT login, grupo FROM lista LOOP
    EXECUTE format('CREATE ROLE %I LOGIN', x.login);
    EXECUTE format('GRANT %I TO %I WITH INHERIT TRUE, SET FALSE, ADMIN FALSE', x.grupo, x.login);
    EXECUTE format('GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO %I', x.login);
  END LOOP;
END $r$;

SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'mutación rechazada' USING ERRCODE = '55000'; END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'truncado rechazado' USING ERRCODE = '55000'; END $f$;

DO $f$
DECLARE literales text; auds text;
BEGIN
  SELECT string_agg(DISTINCT quote_literal(v), ',') INTO literales
    FROM lista, LATERAL (VALUES (aud), (op), (perfil)) t(v);
  SELECT string_agg(DISTINCT quote_literal(aud), ',') INTO auds FROM lista;
  EXECUTE format($t$CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version (
    audiencia_consumo text CONSTRAINT clave_capacidad_version_audiencia_consumo_check
    CHECK (audiencia_consumo = ANY (ARRAY[%s, 'vec.admin.usuarios.consultar.v1'])))$t$, auds);
  -- Sustituto del núcleo: solo importa su texto (exige el origen y nombra
  -- perfiles, audiencias y operaciones), igual que lo coteja el guion.
  EXECUTE format($t$CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RETURNS void
    LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $c$
    BEGIN
      IF $1 = ANY (ARRAY[%s]) THEN NULL; END IF;
      PERFORM vec_autorizacion_atestada_v3.resolver_origen_consumo_v1('a','b','c');
    END $c$$t$, literales);
END $f$;
RESET ROLE;
