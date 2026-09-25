\set ON_ERROR_STOP on
-- Calendarios 000001: historia de solo adición de calendarios hábiles y
-- laborales, y lectura por funciones SECURITY DEFINER. Otros módulos no leen
-- estas tablas: consultan el puerto de Calendarios.
BEGIN;
SET LOCAL ROLE vec_calendarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_calendarios:migracion:000001:historia:v1',0));
DO $pre$ BEGIN
 IF current_user<>'vec_calendarios_propietario' OR to_regnamespace('vec_calendarios') IS NULL
    OR to_regclass('vec_calendarios.version_calendario') IS NOT NULL OR to_regclass('vec_calendarios.dia_calendario') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_calendarios_lector' AND NOT rolcanlogin AND NOT rolbypassrls AND NOT rolsuper)
 THEN RAISE EXCEPTION 'Calendarios 000001: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Una versión es inmutable. La corrección de un calendario publicado es la
-- versión siguiente del mismo ámbito y año, y enlaza la que sustituye.
-- conocido_desde lo fija el servidor: es el eje de conocimiento.
CREATE TABLE vec_calendarios.version_calendario (
  id text PRIMARY KEY CHECK (id ~ '^[a-z][a-z0-9:_-]{1,95}$'),
  ambito_tipo text NOT NULL CHECK (ambito_tipo IN ('nacional','autonomico','local','centro')),
  ambito_ref text NOT NULL CHECK (ambito_ref ~ '^[a-z][a-z0-9:_-]{1,95}$'),
  anio integer NOT NULL CHECK (anio BETWEEN 2000 AND 2100),
  numero integer NOT NULL CHECK (numero BETWEEN 1 AND 10000),
  sustituye_id text UNIQUE REFERENCES vec_calendarios.version_calendario(id),
  denominacion text NOT NULL CHECK (char_length(denominacion) BETWEEN 1 AND 240 AND denominacion=btrim(denominacion) AND denominacion !~ '[[:cntrl:]]'),
  procedencia_norma text NOT NULL CHECK (char_length(procedencia_norma) BETWEEN 1 AND 400 AND procedencia_norma=btrim(procedencia_norma) AND procedencia_norma !~ '[[:cntrl:]]'),
  procedencia_referencia text NOT NULL CHECK (char_length(procedencia_referencia) BETWEEN 1 AND 400 AND procedencia_referencia=btrim(procedencia_referencia) AND procedencia_referencia !~ '[[:cntrl:]]'),
  procedencia_publicada_en date NOT NULL,
  sintetica boolean NOT NULL,
  comunidad_ref text CHECK (comunidad_ref ~ '^[a-z][a-z0-9:_-]{1,95}$'),
  municipio_ref text CHECK (municipio_ref ~ '^[a-z][a-z0-9:_-]{1,95}$'),
  conocido_desde timestamptz NOT NULL,
  UNIQUE (ambito_tipo, ambito_ref, anio, numero),
  CHECK (ambito_tipo<>'nacional' OR ambito_ref='es'),
  CHECK ((numero=1)=(sustituye_id IS NULL)),
  CHECK ((ambito_tipo IN ('local','centro'))=(comunidad_ref IS NOT NULL)),
  CHECK ((ambito_tipo='centro')=(municipio_ref IS NOT NULL))
);

CREATE TABLE vec_calendarios.dia_calendario (
  version_id text NOT NULL REFERENCES vec_calendarios.version_calendario(id),
  fecha date NOT NULL,
  efecto text NOT NULL CHECK (efecto IN ('festivo','inhabil_administrativo','no_laborable')),
  denominacion text NOT NULL CHECK (char_length(denominacion) BETWEEN 1 AND 240 AND denominacion=btrim(denominacion) AND denominacion !~ '[[:cntrl:]]'),
  PRIMARY KEY (version_id, fecha, efecto)
);

-- Encadenamiento: numero 1 abre el ámbito y año; numero n sustituye
-- exactamente a la n-1, que debe ser la última. El reloj es del servidor.
CREATE FUNCTION vec_calendarios.antes_de_insertar_version() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
DECLARE previa vec_calendarios.version_calendario%ROWTYPE;
BEGIN
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_calendarios:ambito:'||NEW.ambito_tipo||'|'||NEW.ambito_ref||'|'||NEW.anio,0));
  IF NEW.numero=1 THEN
    IF EXISTS (SELECT 1 FROM vec_calendarios.version_calendario v WHERE v.ambito_tipo=NEW.ambito_tipo AND v.ambito_ref=NEW.ambito_ref AND v.anio=NEW.anio) THEN
      RAISE EXCEPTION 'Calendarios: el ámbito y año ya tienen versión' USING ERRCODE='23505';
    END IF;
  ELSE
    SELECT * INTO previa FROM vec_calendarios.version_calendario v WHERE v.id=NEW.sustituye_id;
    IF NOT FOUND OR previa.ambito_tipo<>NEW.ambito_tipo OR previa.ambito_ref<>NEW.ambito_ref OR previa.anio<>NEW.anio OR previa.numero<>NEW.numero-1
       OR previa.comunidad_ref IS DISTINCT FROM NEW.comunidad_ref THEN
      RAISE EXCEPTION 'Calendarios: sustitución incoherente' USING ERRCODE='23514';
    END IF;
  END IF;
  NEW.conocido_desde := date_trunc('microseconds', clock_timestamp());
  -- Marca local de la transacción: solo aquí se admiten sus días.
  PERFORM set_config('vec_calendarios.versiones_abiertas',
    btrim(COALESCE(current_setting('vec_calendarios.versiones_abiertas', true), '')||' '||NEW.id), true);
  RETURN NEW;
END $f$;

-- Los días solo se declaran en la misma transacción que crea su versión:
-- una versión confirmada no admite días añadidos después.
CREATE FUNCTION vec_calendarios.antes_de_insertar_dia() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v vec_calendarios.version_calendario%ROWTYPE;
BEGIN
  SELECT * INTO v FROM vec_calendarios.version_calendario x WHERE x.id=NEW.version_id;
  IF NOT FOUND OR NOT (NEW.version_id = ANY (string_to_array(COALESCE(current_setting('vec_calendarios.versiones_abiertas', true), ''), ' '))) THEN
    RAISE EXCEPTION 'Calendarios: la versión ya está publicada' USING ERRCODE='55000';
  END IF;
  IF extract(year FROM NEW.fecha)::integer<>v.anio OR (v.ambito_tipo='centro')<>(NEW.efecto='no_laborable') THEN
    RAISE EXCEPTION 'Calendarios: día incoherente con su versión' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END $f$;

CREATE FUNCTION vec_calendarios.historia_inmutable() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
  RAISE EXCEPTION 'Calendarios: la historia es de solo adición' USING ERRCODE='55000';
END $f$;

CREATE TRIGGER version_antes_de_insertar BEFORE INSERT ON vec_calendarios.version_calendario
  FOR EACH ROW EXECUTE FUNCTION vec_calendarios.antes_de_insertar_version();
CREATE TRIGGER dia_antes_de_insertar BEFORE INSERT ON vec_calendarios.dia_calendario
  FOR EACH ROW EXECUTE FUNCTION vec_calendarios.antes_de_insertar_dia();
-- Por sentencia: la RLS dejaría sin filas un UPDATE o DELETE y la guarda por
-- fila nunca se ejecutaría; así cualquier intento falla de forma explícita.
CREATE TRIGGER version_inmutable BEFORE UPDATE OR DELETE ON vec_calendarios.version_calendario
  FOR EACH STATEMENT EXECUTE FUNCTION vec_calendarios.historia_inmutable();
CREATE TRIGGER dia_inmutable BEFORE UPDATE OR DELETE ON vec_calendarios.dia_calendario
  FOR EACH STATEMENT EXECUTE FUNCTION vec_calendarios.historia_inmutable();
CREATE TRIGGER version_sin_vaciado BEFORE TRUNCATE ON vec_calendarios.version_calendario
  FOR EACH STATEMENT EXECUTE FUNCTION vec_calendarios.historia_inmutable();
CREATE TRIGGER dia_sin_vaciado BEFORE TRUNCATE ON vec_calendarios.dia_calendario
  FOR EACH STATEMENT EXECUTE FUNCTION vec_calendarios.historia_inmutable();

ALTER TABLE vec_calendarios.version_calendario ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_calendarios.version_calendario FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_calendarios.dia_calendario ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_calendarios.dia_calendario FORCE ROW LEVEL SECURITY;
-- Solo el propietario lee (a través de las funciones) y añade (migraciones o
-- el futuro acto gobernado de RRHH). El lector no tiene privilegio en tablas.
CREATE POLICY lectura_propietario ON vec_calendarios.version_calendario FOR SELECT TO vec_calendarios_propietario USING (true);
CREATE POLICY alta_propietario ON vec_calendarios.version_calendario FOR INSERT TO vec_calendarios_propietario WITH CHECK (true);
CREATE POLICY lectura_propietario ON vec_calendarios.dia_calendario FOR SELECT TO vec_calendarios_propietario USING (true);
CREATE POLICY alta_propietario ON vec_calendarios.dia_calendario FOR INSERT TO vec_calendarios_propietario WITH CHECK (true);
REVOKE ALL ON vec_calendarios.version_calendario, vec_calendarios.dia_calendario FROM PUBLIC;

-- Última versión de cada ámbito pedido conocida hasta p_conocido_en.
CREATE FUNCTION vec_calendarios.versiones_vigentes_v1(p_anio integer, p_tipos text[], p_refs text[], p_conocido_en timestamptz)
RETURNS TABLE (id text, ambito_tipo text, ambito_ref text, anio integer, numero integer, sustituye_id text, denominacion text,
               procedencia_norma text, procedencia_referencia text, procedencia_publicada_en date, sintetica boolean,
               comunidad_ref text, municipio_ref text, conocido_desde timestamptz, dias jsonb)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
#variable_conflict use_column
BEGIN
  IF p_anio IS NULL OR p_anio NOT BETWEEN 2000 AND 2100 OR p_tipos IS NULL OR p_refs IS NULL
     OR cardinality(p_tipos) NOT BETWEEN 1 AND 16 OR cardinality(p_tipos)<>cardinality(p_refs)
     OR array_position(p_tipos, NULL) IS NOT NULL OR array_position(p_refs, NULL) IS NOT NULL
     OR p_conocido_en IS NULL OR p_conocido_en>statement_timestamp() THEN
    RAISE EXCEPTION 'Calendarios: consulta inválida' USING ERRCODE='22023';
  END IF;
  RETURN QUERY
  SELECT v.id, v.ambito_tipo, v.ambito_ref, v.anio, v.numero, v.sustituye_id, v.denominacion,
         v.procedencia_norma, v.procedencia_referencia, v.procedencia_publicada_en, v.sintetica,
         v.comunidad_ref, v.municipio_ref, v.conocido_desde,
         COALESCE((SELECT jsonb_agg(jsonb_build_object('fecha', d.fecha::text, 'efecto', d.efecto, 'denominacion', d.denominacion)
                                    ORDER BY d.fecha, d.efecto)
                     FROM vec_calendarios.dia_calendario d WHERE d.version_id=v.id), '[]'::jsonb)
    FROM (SELECT DISTINCT ON (x.ambito_tipo, x.ambito_ref) x.*
            FROM vec_calendarios.version_calendario x
            JOIN unnest(p_tipos, p_refs) AS pedido(tipo, ref) ON pedido.tipo=x.ambito_tipo AND pedido.ref=x.ambito_ref
           WHERE x.anio=p_anio AND x.conocido_desde<=p_conocido_en
           ORDER BY x.ambito_tipo, x.ambito_ref, x.numero DESC) v
   ORDER BY v.ambito_tipo, v.ambito_ref;
END $f$;

-- Calendarios de centro con versión conocida para el año, sin días.
CREATE FUNCTION vec_calendarios.centros_con_calendario_v1(p_anio integer, p_conocido_en timestamptz)
RETURNS TABLE (id text, ambito_tipo text, ambito_ref text, anio integer, numero integer, sustituye_id text, denominacion text,
               procedencia_norma text, procedencia_referencia text, procedencia_publicada_en date, sintetica boolean,
               comunidad_ref text, municipio_ref text, conocido_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
#variable_conflict use_column
BEGIN
  IF p_anio IS NULL OR p_anio NOT BETWEEN 2000 AND 2100 OR p_conocido_en IS NULL OR p_conocido_en>statement_timestamp() THEN
    RAISE EXCEPTION 'Calendarios: consulta inválida' USING ERRCODE='22023';
  END IF;
  RETURN QUERY
  SELECT v.id, v.ambito_tipo, v.ambito_ref, v.anio, v.numero, v.sustituye_id, v.denominacion,
         v.procedencia_norma, v.procedencia_referencia, v.procedencia_publicada_en, v.sintetica,
         v.comunidad_ref, v.municipio_ref, v.conocido_desde
    FROM (SELECT DISTINCT ON (x.ambito_ref) x.*
            FROM vec_calendarios.version_calendario x
           WHERE x.ambito_tipo='centro' AND x.anio=p_anio AND x.conocido_desde<=p_conocido_en
           ORDER BY x.ambito_ref, x.numero DESC
           LIMIT 1000) v
   ORDER BY v.ambito_ref;
END $f$;

REVOKE ALL ON FUNCTION vec_calendarios.antes_de_insertar_version(), vec_calendarios.antes_de_insertar_dia(),
  vec_calendarios.historia_inmutable(), vec_calendarios.versiones_vigentes_v1(integer,text[],text[],timestamptz),
  vec_calendarios.centros_con_calendario_v1(integer,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_calendarios.versiones_vigentes_v1(integer,text[],text[],timestamptz),
  vec_calendarios.centros_con_calendario_v1(integer,timestamptz) TO vec_calendarios_lector;
COMMIT;
