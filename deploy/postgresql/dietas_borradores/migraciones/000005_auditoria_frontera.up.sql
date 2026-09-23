\set ON_ERROR_STOP on
-- Delta posterior a 000001-000004. La identidad LOGIN nominal del registrador
-- se aprovisiona fuera de Git; este rol de grupo no puede iniciar sesión.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_dietas:migracion:000005:auditoria_frontera', 0)
);

DO $preparar_rol$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regnamespace('vec_dietas') IS NULL
      OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_dietas_registrador_frontera'
    ) OR pg_catalog.to_regclass('vec_dietas.auditoria_frontera_comision') IS NOT NULL
      OR pg_catalog.to_regprocedure(
        'vec_dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text)'
      ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Dietas 000005 requiere DBA y preimagen sin auditoria de frontera';
    END IF;

    CREATE ROLE vec_dietas_registrador_frontera
        NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
        NOREPLICATION NOBYPASSRLS;
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_dietas_registrador_frontera',
        pg_catalog.current_database()
    );
END
$preparar_rol$;

SET LOCAL ROLE vec_dietas_propietario;

CREATE TABLE vec_dietas.auditoria_frontera_comision (
    evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    correlacion_ref text NOT NULL,
    motivo text NOT NULL,
    superficie text NOT NULL,
    ruta text NOT NULL,
    accion text NOT NULL,
    actor_ref text,
    registrada_en timestamptz(6) NOT NULL,
    CONSTRAINT auditoria_frontera_correlacion_valida CHECK (
        correlacion_ref = 'corr_no_disponible'
        OR correlacion_ref ~ '^corr_[0-9a-f]{32}$'
    ),
    CONSTRAINT auditoria_frontera_motivo_cerrado CHECK (
        motivo IN (
            'autenticacion_requerida',
            'acceso_denegado',
            'dependencia_no_disponible'
        )
    ),
    CONSTRAINT auditoria_frontera_superficie_cerrada CHECK (
        superficie = 'api.dietas.comisiones'
    ),
    CONSTRAINT auditoria_frontera_ruta_clase_cerrada CHECK (
        ruta IN (
            '/api/vec/dietas/comisiones',
            '/api/vec/dietas/comisiones/detalle'
        )
    ),
    CONSTRAINT auditoria_frontera_accion_cerrada CHECK (
        accion IN ('listar', 'crear', 'consultar_detalle', 'metodo_no_admitido')
    ),
    CONSTRAINT auditoria_frontera_actor_minimo CHECK (
        actor_ref IS NULL OR (
            pg_catalog.length(actor_ref) BETWEEN 1 AND 512
            AND actor_ref ~ '^[A-Za-z0-9:_-]+$'
        )
    ),
    CONSTRAINT auditoria_frontera_sin_actor_anonimo CHECK (
        motivo <> 'autenticacion_requerida' OR actor_ref IS NULL
    )
);

CREATE INDEX auditoria_frontera_comision_correlacion_idx
    ON vec_dietas.auditoria_frontera_comision (correlacion_ref, evento_id);

CREATE FUNCTION vec_dietas.rechazar_mutacion_auditoria_frontera_comision_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '42501',
        MESSAGE = 'auditoria de frontera Dietas inmutable';
END
$funcion$;

CREATE TRIGGER auditoria_frontera_comision_inmutable
BEFORE UPDATE OR DELETE ON vec_dietas.auditoria_frontera_comision
FOR EACH ROW EXECUTE FUNCTION vec_dietas.rechazar_mutacion_auditoria_frontera_comision_v1();

CREATE TRIGGER auditoria_frontera_comision_no_truncar
BEFORE TRUNCATE ON vec_dietas.auditoria_frontera_comision
FOR EACH STATEMENT EXECUTE FUNCTION vec_dietas.rechazar_mutacion_auditoria_frontera_comision_v1();

CREATE FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v1(
    p_correlacion_ref text,
    p_motivo text,
    p_superficie text,
    p_ruta text,
    p_accion text,
    p_actor_ref text
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '2s'
AS $funcion$
BEGIN
    IF current_user <> 'vec_dietas_propietario'
       OR session_user = current_user
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member = session_user::pg_catalog.regrole
              AND membresia.roleid =
                  'vec_dietas_registrador_frontera'::pg_catalog.regrole
              AND membresia.inherit_option
              AND NOT membresia.set_option
              AND NOT membresia.admin_option
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles rol
            WHERE rol.oid <> session_user::pg_catalog.regrole
              AND rol.oid <>
                  'vec_dietas_registrador_frontera'::pg_catalog.regrole
              AND pg_catalog.pg_has_role(session_user, rol.oid, 'MEMBER')
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'registrador de frontera Dietas invalido';
    END IF;

    IF p_correlacion_ref IS NULL
       OR (p_correlacion_ref <> 'corr_no_disponible'
           AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
       OR p_motivo IS NULL
       OR p_motivo NOT IN (
           'autenticacion_requerida',
           'acceso_denegado',
           'dependencia_no_disponible'
       )
       OR p_superficie IS DISTINCT FROM 'api.dietas.comisiones'
       OR p_ruta IS NULL
       OR p_ruta NOT IN (
           '/api/vec/dietas/comisiones',
           '/api/vec/dietas/comisiones/detalle'
       )
       OR p_accion IS NULL
       OR p_accion NOT IN (
           'listar', 'crear', 'consultar_detalle', 'metodo_no_admitido'
       )
       OR (p_motivo = 'autenticacion_requerida' AND p_actor_ref IS NOT NULL)
       OR (p_actor_ref IS NOT NULL AND (
           pg_catalog.length(p_actor_ref) > 512
           OR p_actor_ref !~ '^[A-Za-z0-9:_-]+$'
       )) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'auditoria de frontera Dietas invalida';
    END IF;

    INSERT INTO vec_dietas.auditoria_frontera_comision (
        correlacion_ref, motivo, superficie, ruta, accion, actor_ref, registrada_en
    ) VALUES (
        p_correlacion_ref, p_motivo, p_superficie, p_ruta, p_accion, p_actor_ref,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    );
    RETURN true;
END
$funcion$;

ALTER TABLE vec_dietas.auditoria_frontera_comision ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_dietas.auditoria_frontera_comision FORCE ROW LEVEL SECURITY;
CREATE POLICY auditoria_frontera_comision_propietario
    ON vec_dietas.auditoria_frontera_comision
    TO vec_dietas_propietario
    USING (true) WITH CHECK (true);

REVOKE ALL ON TABLE vec_dietas.auditoria_frontera_comision
    FROM PUBLIC, vec_dietas_ejecutor, vec_dietas_registrador_frontera;
REVOKE ALL ON SEQUENCE vec_dietas.auditoria_frontera_comision_evento_id_seq
    FROM PUBLIC, vec_dietas_ejecutor, vec_dietas_registrador_frontera;
REVOKE ALL ON FUNCTION vec_dietas.rechazar_mutacion_auditoria_frontera_comision_v1()
    FROM PUBLIC, vec_dietas_ejecutor, vec_dietas_registrador_frontera;
REVOKE ALL ON FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text)
    FROM PUBLIC, vec_dietas_ejecutor, vec_dietas_migrador;
GRANT USAGE ON SCHEMA vec_dietas TO vec_dietas_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_dietas.registrar_auditoria_frontera_comision_v1(text,text,text,text,text,text)
    TO vec_dietas_registrador_frontera;

COMMIT;
