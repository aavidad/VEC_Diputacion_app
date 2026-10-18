-- Semilla pública F0-29 para la copia desechable HZ37 sin datos.
-- Fuente: AD172 (tabla y RLS), AD233 (audiencias y operaciones),
-- AD278 (orígenes) y contexto_actor_v1/roles_up.sql (migrador de CTX45).
-- Sólo prepara procedencia técnica: no publica permisos de negocio, actos,
-- personas, firmas, recibos ni material criptográfico.
-- Ejecutar una vez como superusuario local tras restaurar esquema_hz37.sql.gz
-- y antes de lista_sql_main_tras_hz37.txt; nunca instalar en producción.
BEGIN;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '2min';
SET LOCAL search_path = pg_catalog;

DO $pre$
DECLARE
  relacion record;
  total integer := 0;
  contiene_filas boolean;
BEGIN
  IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
     OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper)
     OR to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
     OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
     OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
     OR to_regrole('vec_bolsa_llamamientos_ejecutor') IS NULL
     OR to_regrole('vec_contexto_actor_v1_migrador') IS NOT NULL
     OR to_regrole('vec_ct_o207_runtime') IS NOT NULL
     OR to_regrole('vec_bolsa_llamamientos_desarrollo') IS NOT NULL
  THEN RAISE EXCEPTION 'F0-29: esquema o roles de partida incompatibles' USING ERRCODE = '55000';
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_roles
    WHERE rolname IN ('vec_contratacion_temporal_ejecutor',
                      'vec_bolsa_llamamientos_ejecutor',
                      'vec_autorizacion_atestada_v3_propietario')
      AND rolcanlogin
  ) OR NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'vec_autorizacion_atestada_v3'
      AND c.relname = 'configuracion_origen_consumos_v1'
      AND c.relkind = 'r' AND c.relrowsecurity AND c.relforcerowsecurity
      AND c.relowner = 'vec_autorizacion_atestada_v3_propietario'::regrole
  ) THEN RAISE EXCEPTION 'F0-29: propietario o RLS incompatibles' USING ERRCODE = '55000';
  END IF;

  -- Protege frente a una ejecución accidental sobre una base con historia.
  FOR relacion IN
    SELECT n.nspname, c.relname
    FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE (n.nspname = 'public' OR left(n.nspname, 4) = 'vec_')
      AND c.relkind IN ('r', 'm')
      AND NOT EXISTS (
        SELECT 1 FROM pg_depend d
        WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e')
  LOOP
    EXECUTE format('SELECT EXISTS (SELECT 1 FROM %I.%I)',
                   relacion.nspname, relacion.relname) INTO contiene_filas;
    IF contiene_filas THEN
      RAISE EXCEPTION 'F0-29: la base contiene filas; semilla sólo para HZ37 vacío'
        USING ERRCODE = '55000';
    END IF;
    total := total + 1;
  END LOOP;
  IF total <> 662 THEN
    RAISE EXCEPTION 'F0-29: relaciones esperadas 662, encontradas %', total
      USING ERRCODE = '55000';
  END IF;
END $pre$;

-- El extractor de roles del DDL no ve este grupo: CTX45 le concede EXECUTE.
CREATE ROLE vec_contexto_actor_v1_migrador NOLOGIN NOSUPERUSER
  NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_ct_o207_runtime LOGIN INHERIT NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL;
CREATE ROLE vec_bolsa_llamamientos_desarrollo LOGIN INHERIT NOSUPERUSER
  NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL;
DO $caducidad$
DECLARE
  limite text := (clock_timestamp() + INTERVAL '90 days')::text;
BEGIN
  EXECUTE format('ALTER ROLE vec_ct_o207_runtime VALID UNTIL %L', limite);
  EXECUTE format('ALTER ROLE vec_bolsa_llamamientos_desarrollo VALID UNTIL %L', limite);
END $caducidad$;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct_o207_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_bolsa_llamamientos_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
  (login_nombre, audiencia_consumo, operacion, proceso, canal_permitido)
VALUES
  ('vec_ct_o207_runtime',
   'vec_contratacion_temporal.vinculo_emision_bolsa.v1',
   'contratacion_temporal.bolsa.vincular',
   'contratacion_temporal_ci', 'interna_corporativa'),
  ('vec_bolsa_llamamientos_desarrollo',
   'vec_bolsa_llamamientos.llamamiento.emitir.v1',
   'llamamiento.emitir.v1',
   'bolsa_llamamientos_ci', 'interna_corporativa');
RESET ROLE;

DO $post$
DECLARE
  rol record;
BEGIN
  FOR rol IN
    SELECT r.rolname, r.rolcanlogin, r.rolinherit, r.rolsuper,
           r.rolcreatedb, r.rolcreaterole, r.rolreplication,
           r.rolbypassrls, r.rolvaliduntil, r.rolconfig,
           a.rolpassword AS password_hash,
           m.roleid, m.admin_option, m.inherit_option, m.set_option
    FROM pg_roles r
    JOIN pg_authid a ON a.oid = r.oid
    JOIN pg_auth_members m ON m.member = r.oid
    WHERE r.rolname IN ('vec_ct_o207_runtime',
                        'vec_bolsa_llamamientos_desarrollo')
  LOOP
    IF NOT rol.rolcanlogin OR NOT rol.rolinherit
       OR rol.rolsuper OR rol.rolcreatedb OR rol.rolcreaterole
       OR rol.rolreplication OR rol.rolbypassrls
       OR rol.password_hash IS NOT NULL OR rol.rolconfig IS NOT NULL
       OR rol.rolvaliduntil IS NULL
       OR rol.rolvaliduntil < clock_timestamp() + INTERVAL '60 days'
       OR rol.rolvaliduntil > clock_timestamp() + INTERVAL '90 days'
       OR rol.admin_option OR NOT rol.inherit_option OR rol.set_option
       OR rol.roleid <> (CASE rol.rolname
           WHEN 'vec_ct_o207_runtime' THEN 'vec_contratacion_temporal_ejecutor'::regrole
           ELSE 'vec_bolsa_llamamientos_ejecutor'::regrole END)
    THEN RAISE EXCEPTION 'F0-29: rol o membresía fuera del contrato: %', rol.rolname
      USING ERRCODE = '55000';
    END IF;
  END LOOP;
  IF (SELECT count(*) FROM pg_auth_members m JOIN pg_roles r ON r.oid = m.member
      WHERE r.rolname IN ('vec_ct_o207_runtime',
                          'vec_bolsa_llamamientos_desarrollo')) <> 2
     OR (SELECT count(*) FROM pg_roles
         WHERE rolname IN ('vec_ct_o207_runtime',
                           'vec_bolsa_llamamientos_desarrollo')) <> 2
  THEN RAISE EXCEPTION 'F0-29: se esperaba una membresía por LOGIN' USING ERRCODE = '55000';
  END IF;
  IF NOT EXISTS (
       SELECT 1 FROM pg_roles r JOIN pg_authid a ON a.oid = r.oid
       WHERE r.rolname = 'vec_contexto_actor_v1_migrador'
         AND NOT (r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolcreatedb
                  OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
         AND a.rolpassword IS NULL AND r.rolconfig IS NULL
     ) OR EXISTS (
       SELECT 1 FROM pg_auth_members m
       WHERE m.member = 'vec_contexto_actor_v1_migrador'::regrole
          OR m.roleid = 'vec_contexto_actor_v1_migrador'::regrole
     )
  THEN RAISE EXCEPTION 'F0-29: migrador CTX45 fuera del contrato' USING ERRCODE = '55000';
  END IF;
END $post$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $post_filas$
BEGIN
  IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1) <> 2
     OR NOT EXISTS (
       SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
       WHERE login_nombre = 'vec_ct_o207_runtime'
         AND audiencia_consumo = 'vec_contratacion_temporal.vinculo_emision_bolsa.v1'
         AND operacion = 'contratacion_temporal.bolsa.vincular'
         AND proceso = 'contratacion_temporal_ci'
         AND canal_permitido = 'interna_corporativa')
     OR NOT EXISTS (
       SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
       WHERE login_nombre = 'vec_bolsa_llamamientos_desarrollo'
         AND audiencia_consumo = 'vec_bolsa_llamamientos.llamamiento.emitir.v1'
         AND operacion = 'llamamiento.emitir.v1'
         AND proceso = 'bolsa_llamamientos_ci'
         AND canal_permitido = 'interna_corporativa')
  THEN RAISE EXCEPTION 'F0-29: orígenes previos de AD278 incompatibles'
    USING ERRCODE = '55000';
  END IF;
END $post_filas$;
RESET ROLE;
COMMIT;
