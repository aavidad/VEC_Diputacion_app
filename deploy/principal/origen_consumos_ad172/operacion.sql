\set ON_ERROR_STOP on
-- Filas de origen AD172 de vec-server. Solo añade configuración técnica a
-- configuracion_origen_consumos_v1: no concede acciones, perfiles ni
-- membresías. Variables psql:
--   ternas     filas de ternas.tsv ya filtradas por bloque (ejecutar.sh)
--   finalizar  ROLLBACK (ensayo) o COMMIT (aplicar)
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:origen_consumos:ad172:20261006', 0));
-- Compartido con las migraciones que reconstruyen el núcleo mientras se coteja.
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_autorizacion_atestada_v3:nucleo', 0));

CREATE TEMP TABLE origen_esperado (
  bloque text, login_nombre name, grupo name, audiencia_consumo text,
  operacion text, canal_permitido text, proceso text, perfil text, campos integer) ON COMMIT DROP;
INSERT INTO origen_esperado
SELECT c[1], c[2], c[3], c[4], c[5], c[6], c[7], c[8], cardinality(c)
  FROM regexp_split_to_table(:'ternas', E'\n') AS l(linea),
       LATERAL string_to_array(l.linea, E'\t') AS c
 WHERE l.linea <> '' AND left(l.linea, 1) <> '#';

DO $pre$
DECLARE
  nucleo text;
  audiencias text;
  t record;
BEGIN
  IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
     OR NOT (SELECT rolsuper FROM pg_roles WHERE rolname = session_user)
     OR session_user <> current_user
     OR current_database() <> 'postgres' THEN
    RAISE EXCEPTION 'ORIGEN-AD172: exige PostgreSQL 18, DBA y base postgres' USING ERRCODE = '42501';
  END IF;

  -- Forma de cada fila: ocho campos, valores admitidos por la tabla y sin
  -- duplicar la clave LOGIN/audiencia/operación.
  IF NOT EXISTS (SELECT 1 FROM origen_esperado)
     OR EXISTS (SELECT 1 FROM origen_esperado
       WHERE campos <> 8 OR bloque !~ '^[a-z]+$'
          OR login_nombre::text !~ '^vec_[a-z0-9_]{1,59}$' OR grupo::text !~ '^vec_[a-z0-9_]{1,59}$'
          OR audiencia_consumo !~ '^[a-z][a-z0-9._:-]*$' OR length(audiencia_consumo) > 512
          OR operacion !~ '^[a-z][a-z0-9._:-]{0,159}$'
          OR proceso !~ '^[a-z][a-z0-9._-]{1,79}$'
          OR canal_permitido NOT IN ('interna_corporativa', 'externa_personal')
          OR perfil !~ '^[a-z][a-z0-9_]{1,79}$')
     OR (SELECT count(*) FROM origen_esperado)
        <> (SELECT count(DISTINCT (login_nombre, audiencia_consumo, operacion)) FROM origen_esperado) THEN
    RAISE EXCEPTION 'ORIGEN-AD172: lista de ternas mal formada' USING ERRCODE = '22023';
  END IF;

  -- AD172 instalada con su tabla protegida y su resolutor.
  IF to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
     OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
     OR NOT EXISTS (SELECT 1 FROM pg_class c
       WHERE c.oid = to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1')
         AND c.relrowsecurity AND c.relforcerowsecurity
         AND c.relowner = to_regrole('vec_autorizacion_atestada_v3_propietario'))
     OR (SELECT count(*) FROM pg_trigger g
          WHERE g.tgrelid = to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1')
            AND NOT g.tgisinternal AND g.tgname IN ('inmutable', 'no_truncar')) <> 2
     OR EXISTS (SELECT 1 FROM pg_class c, aclexplode(coalesce(c.relacl, acldefault('r', c.relowner))) a
       WHERE c.oid = to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1')
         AND a.grantee <> c.relowner) THEN
    RAISE EXCEPTION 'ORIGEN-AD172: AD172 ausente o con permisos distintos de los instalados' USING ERRCODE = '55000';
  END IF;

  -- El núcleo vivo exige el origen y nombra cada perfil, audiencia y
  -- operación; cada audiencia es admisible para una clave de capacidad.
  SELECT p.prosrc INTO nucleo FROM pg_proc p
   WHERE p.oid = to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  SELECT pg_get_constraintdef(c.oid) INTO audiencias FROM pg_constraint c
   WHERE c.conrelid = to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version')
     AND c.conname = 'clave_capacidad_version_audiencia_consumo_check' AND c.contype = 'c';
  IF nucleo IS NULL OR audiencias IS NULL OR strpos(nucleo, 'resolver_origen_consumo_v1') = 0 THEN
    RAISE EXCEPTION 'ORIGEN-AD172: núcleo sin AD172 o sin catálogo de audiencias' USING ERRCODE = '55000';
  END IF;
  FOR t IN SELECT * FROM origen_esperado LOOP
    IF strpos(nucleo, quote_literal(t.perfil)) = 0
       OR strpos(nucleo, quote_literal(t.audiencia_consumo)) = 0
       OR strpos(nucleo, quote_literal(t.operacion)) = 0
       OR strpos(audiencias, quote_literal(t.audiencia_consumo)) = 0 THEN
      RAISE EXCEPTION 'ORIGEN-AD172: terna no reconocida por el núcleo: % % %', t.perfil, t.audiencia_consumo, t.operacion
        USING ERRCODE = '55000';
    END IF;
  END LOOP;

  -- Cada LOGIN cumple lo que pide el resolutor y su única membresía es el
  -- grupo ejecutor que el núcleo exige a su perfil: heredada, sin SET ni ADMIN.
  IF EXISTS (SELECT 1 FROM origen_esperado GROUP BY login_nombre HAVING count(DISTINCT grupo) <> 1) THEN
    RAISE EXCEPTION 'ORIGEN-AD172: un LOGIN aparece con dos grupos' USING ERRCODE = '22023';
  END IF;
  FOR t IN SELECT DISTINCT login_nombre, grupo FROM origen_esperado LOOP
    IF to_regrole(t.grupo) IS NULL OR to_regrole(t.login_nombre) IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname = t.login_nombre
         AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreaterole
         AND NOT r.rolcreatedb AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR (SELECT count(*) FROM pg_auth_members m WHERE m.member = to_regrole(t.login_nombre)) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_auth_members m
         WHERE m.member = to_regrole(t.login_nombre) AND m.roleid = to_regrole(t.grupo)
           AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
       OR EXISTS (SELECT 1 FROM pg_roles g WHERE g.rolname = t.grupo AND g.rolcanlogin) THEN
      RAISE EXCEPTION 'ORIGEN-AD172: LOGIN o grupo ejecutor incompatible: %', t.login_nombre USING ERRCODE = '55000';
    END IF;
  END LOOP;

  -- La tabla es inmutable: una terna ya configurada con otro proceso o canal
  -- no se corrige aquí; requiere decisión del DBA.
  IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 c
       JOIN origen_esperado e USING (login_nombre, audiencia_consumo, operacion)
      WHERE c.proceso IS DISTINCT FROM e.proceso OR c.canal_permitido IS DISTINCT FROM e.canal_permitido) THEN
    RAISE EXCEPTION 'ORIGEN-AD172: terna ya configurada con otro proceso o canal' USING ERRCODE = '55000';
  END IF;
END $pre$;

-- La política de la tabla solo admite al propietario como usuario efectivo.
-- La lista temporal se borra al terminar la transacción.
GRANT SELECT ON origen_esperado TO vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
WITH nuevas AS (
  INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
    (login_nombre, audiencia_consumo, operacion, proceso, canal_permitido)
  SELECT e.login_nombre, e.audiencia_consumo, e.operacion, e.proceso, e.canal_permitido
    FROM origen_esperado e
   WHERE NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 c
     WHERE c.login_nombre = e.login_nombre AND c.audiencia_consumo = e.audiencia_consumo
       AND c.operacion = e.operacion)
  RETURNING 1)
SELECT 'ternas_nuevas=' || count(*) FROM nuevas;
RESET ROLE;

DO $post$
BEGIN
  IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 c
        JOIN origen_esperado e USING (login_nombre, audiencia_consumo, operacion)
       WHERE c.proceso = e.proceso AND c.canal_permitido = e.canal_permitido)
     <> (SELECT count(*) FROM origen_esperado) THEN
    RAISE EXCEPTION 'ORIGEN-AD172: postcondición fallida' USING ERRCODE = '55000';
  END IF;
END $post$;

:finalizar;
