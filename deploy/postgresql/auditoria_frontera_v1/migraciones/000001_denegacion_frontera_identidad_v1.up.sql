\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_auditoria_frontera_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_auditoria_frontera_v1:migracion:000001', 0)
);
DO $precondicion$
BEGIN
    IF pg_catalog.to_regnamespace('vec_auditoria_frontera_v1') IS NOT NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_auditoria_frontera_v1_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
              AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication AND NOT rolbypassrls
       ) OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_auditoria_frontera_identidad_v1_registrador'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
              AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication AND NOT rolbypassrls
       ) OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_default_acl AS d
             CROSS JOIN LATERAL pg_catalog.aclexplode(d.defaclacl) AS a
            WHERE d.defaclrole = 'vec_auditoria_frontera_v1_propietario'::regrole
              AND (a.grantee = 0 OR a.grantee <> d.defaclrole)
              AND (
                  d.defaclobjtype = 'n'
                  OR (d.defaclobjtype = 'f' AND a.privilege_type = 'EXECUTE')
                  OR d.defaclobjtype IN ('r', 'S', 'T')
              )
       ) OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members AS m
            WHERE m.roleid = 'vec_auditoria_frontera_v1_propietario'::regrole
               OR m.member = 'vec_auditoria_frontera_v1_propietario'::regrole
               OR m.member = 'vec_auditoria_frontera_identidad_v1_registrador'::regrole
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'precondicion auditoria frontera v1 incompatible';
    END IF;
END
$precondicion$;

CREATE SCHEMA vec_auditoria_frontera_v1 AUTHORIZATION vec_auditoria_frontera_v1_propietario;
RESET ROLE;
DO $cerrar_creacion$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE CREATE ON DATABASE %I FROM vec_auditoria_frontera_v1_propietario',
        pg_catalog.current_database()
    );
END
$cerrar_creacion$;
SET LOCAL ROLE vec_auditoria_frontera_v1_propietario;
SET LOCAL search_path = pg_catalog;
CREATE TABLE vec_auditoria_frontera_v1.denegacion_identidad (
    evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    correlacion_ref text NOT NULL UNIQUE,
    superficie text NOT NULL,
    ruta_exacta text NOT NULL,
    accion text NOT NULL,
    motivo text NOT NULL,
    canal_ref text,
    actor_ref text,
    registrada_en timestamptz(6) NOT NULL,
    CHECK (correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'),
    CHECK (superficie IN ('externa_personal', 'interna_corporativa', 'administracion_privilegiada')),
    CHECK (ruta_exacta = '/api/vec/bolsa/mi-bolsa'),
    CHECK (accion = 'bolsa.participaciones_propias.consultar'),
    CHECK (motivo IN ('autenticacion_requerida', 'acceso_denegado')),
    CHECK (canal_ref IS NULL OR canal_ref ~ '^tls-exportador:sha256:[0-9a-f]{64}$'),
    CHECK (actor_ref IS NULL),
    CHECK (registrada_en = pg_catalog.date_trunc('microseconds', registrada_en))
);

CREATE FUNCTION vec_auditoria_frontera_v1.rechazar_mutacion_denegacion_identidad_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'denegacion de frontera inmutable';
END
$funcion$;
CREATE TRIGGER denegacion_identidad_inmutable
BEFORE UPDATE OR DELETE ON vec_auditoria_frontera_v1.denegacion_identidad
FOR EACH ROW EXECUTE FUNCTION vec_auditoria_frontera_v1.rechazar_mutacion_denegacion_identidad_v1();
CREATE TRIGGER denegacion_identidad_no_truncar
BEFORE TRUNCATE ON vec_auditoria_frontera_v1.denegacion_identidad
FOR EACH STATEMENT EXECUTE FUNCTION vec_auditoria_frontera_v1.rechazar_mutacion_denegacion_identidad_v1();

CREATE FUNCTION vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(
    p_correlacion_ref text, p_superficie text, p_ruta_exacta text, p_accion text,
    p_motivo text, p_canal_ref text, p_actor_ref text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = 'on' SET timezone = 'UTC'
SET lock_timeout = '1s' SET statement_timeout = '2s'
AS $funcion$
DECLARE v_existente vec_auditoria_frontera_v1.denegacion_identidad%ROWTYPE;
BEGIN
    IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^correlacion_[0-9a-f]{32}$'
       OR p_superficie NOT IN ('externa_personal', 'interna_corporativa', 'administracion_privilegiada')
       OR p_ruta_exacta <> '/api/vec/bolsa/mi-bolsa'
       OR p_accion <> 'bolsa.participaciones_propias.consultar'
       OR p_motivo NOT IN ('autenticacion_requerida', 'acceso_denegado')
       OR (p_canal_ref IS NOT NULL AND p_canal_ref !~ '^tls-exportador:sha256:[0-9a-f]{64}$')
       OR p_actor_ref IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'denegacion de frontera invalida';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_auditoria_frontera_v1:correlacion:' || p_correlacion_ref, 0));
    INSERT INTO vec_auditoria_frontera_v1.denegacion_identidad (
        correlacion_ref, superficie, ruta_exacta, accion, motivo, canal_ref, actor_ref, registrada_en
    ) VALUES (
        p_correlacion_ref, p_superficie, p_ruta_exacta, p_accion, p_motivo,
        p_canal_ref, p_actor_ref, pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    ) ON CONFLICT (correlacion_ref) DO NOTHING;
    SELECT * INTO v_existente FROM vec_auditoria_frontera_v1.denegacion_identidad
     WHERE correlacion_ref = p_correlacion_ref;
    IF v_existente.superficie IS DISTINCT FROM p_superficie
       OR v_existente.ruta_exacta IS DISTINCT FROM p_ruta_exacta
       OR v_existente.accion IS DISTINCT FROM p_accion
       OR v_existente.motivo IS DISTINCT FROM p_motivo
       OR v_existente.canal_ref IS DISTINCT FROM p_canal_ref
       OR v_existente.actor_ref IS DISTINCT FROM p_actor_ref THEN
        RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'correlacion de denegacion divergente';
    END IF;
    RETURN true;
END
$funcion$;

ALTER TABLE vec_auditoria_frontera_v1.denegacion_identidad ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_auditoria_frontera_v1.denegacion_identidad FORCE ROW LEVEL SECURITY;
CREATE POLICY denegacion_identidad_propietario_lectura ON vec_auditoria_frontera_v1.denegacion_identidad
    FOR SELECT TO vec_auditoria_frontera_v1_propietario USING (true);
CREATE POLICY denegacion_identidad_propietario_alta ON vec_auditoria_frontera_v1.denegacion_identidad
    FOR INSERT TO vec_auditoria_frontera_v1_propietario WITH CHECK (true);
REVOKE ALL ON SCHEMA vec_auditoria_frontera_v1 FROM PUBLIC;
REVOKE ALL ON TABLE vec_auditoria_frontera_v1.denegacion_identidad FROM PUBLIC;
REVOKE ALL ON SEQUENCE vec_auditoria_frontera_v1.denegacion_identidad_evento_id_seq FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_auditoria_frontera_v1.rechazar_mutacion_denegacion_identidad_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(text,text,text,text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_auditoria_frontera_v1 TO vec_auditoria_frontera_identidad_v1_registrador;
GRANT EXECUTE ON FUNCTION vec_auditoria_frontera_v1.registrar_denegacion_frontera_identidad_v1(text,text,text,text,text,text,text)
    TO vec_auditoria_frontera_identidad_v1_registrador;
COMMIT;
