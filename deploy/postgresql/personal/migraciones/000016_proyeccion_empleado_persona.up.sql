\set ON_ERROR_STOP on
-- B2. Proyección gobernada persona→empleado, propiedad de Personal.
--
-- Personal es la única autoridad del empleado. Este corte publica el hecho
-- «la persona canónica per_ tiene el empleado canónico emp_» con procedencia,
-- versión, vigencia y revocación, en historia de solo adición. No crea otra
-- identidad: per_ procede del núcleo y aquí solo se referencia de forma opaca.
-- No lee ni escribe tablas de ContextoActor, Bolsa ni Dietas.
--
-- La lectura gobernada devuelve siempre una sola fila con la clase del
-- resultado (sin_empleado, empleado, ambiguo) y nunca elige un empleado entre
-- varios. Una versión no activa o revocada deja de ser efectiva, pero no anula
-- nada: equivale a «sin empleado» para quien consulte.
--
-- Todavía sin consumidor: ninguna función se concede a un rol runtime. El
-- resolutor de contexto de actor la recibirá en su propia migración nominal
-- cuando Dirección fije cómo incorpora la proyección (véase el informe del
-- corte). La publicación queda reservada al propietario de Personal.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000016:proyeccion-empleado-persona:v1',0));
DO $pre$
BEGIN
 IF current_user <> 'vec_personal_propietario'
    OR to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NOT NULL
    OR to_regprocedure('vec_personal.validar_version_proyeccion_empleado_v1()') IS NOT NULL
    OR to_regprocedure('vec_personal.rechazar_mutacion_proyeccion_empleado_v1()') IS NOT NULL
    OR to_regprocedure('vec_personal.publicar_proyeccion_empleado_persona_v1(text,bigint,text,text,text,timestamptz,timestamptz,text,text,bigint,text)') IS NOT NULL
    OR to_regprocedure('vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_ejecutor'
                     AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_personal'
                     AND nspowner='vec_personal_propietario'::regrole) THEN
   RAISE EXCEPTION 'Personal 000016: dependencias o preimagen incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE FUNCTION vec_personal.rechazar_mutacion_proyeccion_empleado_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $fn$
BEGIN
 RAISE EXCEPTION 'historia de proyección persona-empleado inmutable' USING ERRCODE='55000';
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.rechazar_mutacion_proyeccion_empleado_v1() FROM PUBLIC,vec_personal_ejecutor;

-- Cada proyección (pep_) une una persona con un empleado para siempre; sus
-- versiones solo cambian estado, vigencia y procedencia. Revocada es terminal.
-- Hasta es exclusivo y finito, igual que las vigencias de ContextoActor.
CREATE TABLE vec_personal.proyeccion_empleado_persona_historia (
 proyeccion_ref text NOT NULL CHECK (proyeccion_ref ~ '^pep_[A-Za-z0-9_-]{22,128}$'),
 version bigint NOT NULL CHECK (version > 0),
 persona_ref text NOT NULL CHECK (persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 estado text NOT NULL CHECK (estado IN ('activa','no_activa','revocada')),
 motivo text CHECK (motivo IN ('baja','suspension','error_material','rectificacion')),
 vigente_desde timestamptz(6) NOT NULL CHECK (isfinite(vigente_desde)),
 vigente_hasta timestamptz(6) NOT NULL CHECK (isfinite(vigente_hasta)),
 procedencia_acto_ref text NOT NULL CHECK (procedencia_acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 procedencia_ref text NOT NULL CHECK (procedencia_ref ~ '^prc_[A-Za-z0-9_-]{22,128}$'),
 procedencia_version bigint NOT NULL CHECK (procedencia_version > 0),
 procedencia_huella_sha256 text NOT NULL CHECK (procedencia_huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (proyeccion_ref, version),
 CHECK (vigente_hasta > vigente_desde),
 CHECK ((estado = 'activa') = (motivo IS NULL))
);
CREATE INDEX proyeccion_empleado_persona_idx
 ON vec_personal.proyeccion_empleado_persona_historia (persona_ref, proyeccion_ref, version DESC);
CREATE INDEX proyeccion_empleado_empleado_idx
 ON vec_personal.proyeccion_empleado_persona_historia (empleado_ref, proyeccion_ref);

-- Serializa por persona y por empleado (orden fijo: persona, empleado) y exige
-- continuidad: versión anterior +1, misma pareja, nada después de revocada y
-- un emp_ nunca proyectado a dos personas ni por dos proyecciones.
CREATE FUNCTION vec_personal.validar_version_proyeccion_empleado_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $fn$
DECLARE previa record;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:proyeccion-empleado:persona:'||NEW.persona_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:proyeccion-empleado:empleado:'||NEW.empleado_ref,0));
 NEW.registrada_en := clock_timestamp();
 SELECT h.version, h.persona_ref, h.empleado_ref, h.estado INTO previa
   FROM vec_personal.proyeccion_empleado_persona_historia h
  WHERE h.proyeccion_ref = NEW.proyeccion_ref
  ORDER BY h.version DESC LIMIT 1;
 IF NOT FOUND THEN
   IF NEW.version <> 1 OR EXISTS (
       SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia h
        WHERE h.empleado_ref = NEW.empleado_ref) THEN
     RAISE EXCEPTION 'proyección persona-empleado no admisible' USING ERRCODE='23505';
   END IF;
 ELSIF NEW.version <> previa.version + 1 OR previa.estado = 'revocada'
       OR NEW.persona_ref <> previa.persona_ref OR NEW.empleado_ref <> previa.empleado_ref THEN
   RAISE EXCEPTION 'versión de proyección persona-empleado no admisible' USING ERRCODE='23505';
 END IF;
 RETURN NEW;
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.validar_version_proyeccion_empleado_v1() FROM PUBLIC,vec_personal_ejecutor;

CREATE TRIGGER version_continua BEFORE INSERT ON vec_personal.proyeccion_empleado_persona_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.validar_version_proyeccion_empleado_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.proyeccion_empleado_persona_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_proyeccion_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.proyeccion_empleado_persona_historia
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_proyeccion_empleado_v1();
ALTER TABLE vec_personal.proyeccion_empleado_persona_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.proyeccion_empleado_persona_historia FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.proyeccion_empleado_persona_historia
 FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.proyeccion_empleado_persona_historia FROM PUBLIC,vec_personal_ejecutor;

-- Publicación idempotente: la misma versión con idéntico contenido devuelve la
-- fila existente; con contenido distinto se rechaza sin tocar la historia.
-- Sin EXECUTE para nadie salvo su propietario: la usa la carga gobernada de
-- Personal (migrador con SET ROLE) hasta que exista el acto RRHH que la invoque.
CREATE FUNCTION vec_personal.publicar_proyeccion_empleado_persona_v1(
 p_proyeccion_ref text, p_version bigint, p_persona_ref text, p_empleado_ref text,
 p_estado text, p_vigente_desde timestamptz, p_vigente_hasta timestamptz,
 p_motivo text, p_procedencia_ref text, p_procedencia_version bigint,
 p_procedencia_huella_sha256 text
) RETURNS TABLE (proyeccion_ref text, version bigint, registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $fn$
DECLARE existente vec_personal.proyeccion_empleado_persona_historia%ROWTYPE;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:proyeccion-empleado:persona:'||coalesce(p_persona_ref,''),0));
 SELECT * INTO existente FROM vec_personal.proyeccion_empleado_persona_historia h
  WHERE h.proyeccion_ref = p_proyeccion_ref AND h.version = p_version;
 IF FOUND THEN
   IF existente.persona_ref IS DISTINCT FROM p_persona_ref
      OR existente.empleado_ref IS DISTINCT FROM p_empleado_ref
      OR existente.estado IS DISTINCT FROM p_estado
      OR existente.motivo IS DISTINCT FROM p_motivo
      OR existente.vigente_desde IS DISTINCT FROM p_vigente_desde
      OR existente.vigente_hasta IS DISTINCT FROM p_vigente_hasta
      OR existente.procedencia_ref IS DISTINCT FROM p_procedencia_ref
      OR existente.procedencia_version IS DISTINCT FROM p_procedencia_version
      OR existente.procedencia_huella_sha256 IS DISTINCT FROM p_procedencia_huella_sha256 THEN
     RAISE EXCEPTION 'colisión de versión de proyección persona-empleado' USING ERRCODE='23505';
   END IF;
   RETURN QUERY SELECT existente.proyeccion_ref, existente.version, existente.registrada_en;
   RETURN;
 END IF;
 RETURN QUERY
 INSERT INTO vec_personal.proyeccion_empleado_persona_historia AS h (
   proyeccion_ref, version, persona_ref, empleado_ref, estado, motivo,
   vigente_desde, vigente_hasta, procedencia_acto_ref, procedencia_ref,
   procedencia_version, procedencia_huella_sha256, registrada_en
 ) VALUES (
   p_proyeccion_ref, p_version, p_persona_ref, p_empleado_ref, p_estado, p_motivo,
   p_vigente_desde, p_vigente_hasta, 'personal:proyeccion-empleado:publicacion',
   p_procedencia_ref, p_procedencia_version, p_procedencia_huella_sha256, clock_timestamp()
 ) RETURNING h.proyeccion_ref, h.version, h.registrada_en;
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.publicar_proyeccion_empleado_persona_v1(text,bigint,text,text,text,timestamptz,timestamptz,text,text,bigint,text) FROM PUBLIC,vec_personal_ejecutor;

-- Lectura gobernada. Considera la última versión de cada proyección conocida
-- en p_instante y solo las activas cuya vigencia [desde,hasta) lo cubre.
-- 0 efectivas: sin_empleado; 1 empleado distinto: empleado; más: ambiguo, sin
-- devolver ninguno. Nunca falla por el estado de una proyección concreta.
CREATE FUNCTION vec_personal.resolver_empleado_canonico_persona_v1(
 p_persona_ref text, p_instante timestamptz
) RETURNS TABLE (
 resultado text, persona_ref text, empleado_ref text, proyeccion_ref text, version bigint,
 estado text, vigente_desde timestamptz, vigente_hasta timestamptz,
 procedencia_ref text, procedencia_version bigint, procedencia_huella_sha256 text,
 efectivas integer
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $fn$
DECLARE n integer := 0; f record; unica record;
BEGIN
 IF p_persona_ref IS NULL OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_instante IS NULL OR NOT isfinite(p_instante) THEN
   RAISE EXCEPTION 'solicitud de proyección persona-empleado inválida' USING ERRCODE='22023';
 END IF;
 -- Orden con publicaciones concurrentes de la misma persona, también cuando
 -- el publicador no usa SERIALIZABLE.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_personal:proyeccion-empleado:persona:'||p_persona_ref,0));
 FOR f IN
   SELECT u.* FROM (
     SELECT DISTINCT ON (h.proyeccion_ref) h.*
       FROM vec_personal.proyeccion_empleado_persona_historia h
      WHERE h.persona_ref = p_persona_ref AND h.registrada_en <= p_instante
      ORDER BY h.proyeccion_ref, h.version DESC
   ) u
   WHERE u.estado = 'activa' AND u.vigente_desde <= p_instante AND p_instante < u.vigente_hasta
   ORDER BY u.proyeccion_ref
 LOOP
   n := n + 1;
   unica := f;
 END LOOP;
 -- Un emp_ solo pertenece a una proyección, así que dos efectivas son siempre
 -- dos empleados distintos: se deniega sin elegir.
 IF n = 0 THEN
   RETURN QUERY SELECT 'sin_empleado'::text, p_persona_ref, NULL::text, NULL::text, NULL::bigint,
     NULL::text, NULL::timestamptz, NULL::timestamptz, NULL::text, NULL::bigint, NULL::text, 0;
 ELSIF n = 1 THEN
   RETURN QUERY SELECT 'empleado'::text, unica.persona_ref, unica.empleado_ref, unica.proyeccion_ref,
     unica.version, unica.estado, unica.vigente_desde, unica.vigente_hasta, unica.procedencia_ref,
     unica.procedencia_version, unica.procedencia_huella_sha256, 1;
 ELSE
   RETURN QUERY SELECT 'ambiguo'::text, p_persona_ref, NULL::text, NULL::text, NULL::bigint,
     NULL::text, NULL::timestamptz, NULL::timestamptz, NULL::text, NULL::bigint, NULL::text, n;
 END IF;
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz) FROM PUBLIC,vec_personal_ejecutor;
COMMENT ON FUNCTION vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz) IS
 'Proyección gobernada persona->empleado de Personal: sin_empleado, empleado o ambiguo; nunca elige. Sin consumidor concedido todavía.';
COMMIT;
