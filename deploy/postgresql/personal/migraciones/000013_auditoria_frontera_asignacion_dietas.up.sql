\set ON_ERROR_STOP on
-- D7: auditoría mínima y segregada de rechazos HTTP de Personal. Se instala
-- después de 000012; la identidad LOGIN nominal se aprovisiona fuera de Git.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_personal:migracion:000013:auditoria_frontera_asignacion_dietas', 0)
);

DO $preparar_rol$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) OR pg_catalog.to_regnamespace('vec_personal') IS NULL
      OR pg_catalog.to_regprocedure(
        'vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
      ) IS NULL
      OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_personal_registrador_frontera'
      ) OR pg_catalog.to_regclass('vec_personal.auditoria_frontera_asignacion_dietas') IS NOT NULL
      OR pg_catalog.to_regprocedure(
        'vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)'
      ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'Personal 000013 requiere DBA, 000012 y preimagen sin auditoria de frontera';
    END IF;

    CREATE ROLE vec_personal_registrador_frontera
        NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
        NOREPLICATION NOBYPASSRLS;
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_personal_registrador_frontera',
        pg_catalog.current_database()
    );
END
$preparar_rol$;

SET LOCAL ROLE vec_personal_propietario;

CREATE TABLE vec_personal.auditoria_frontera_asignacion_dietas (
    evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    correlacion_ref text NOT NULL,
    motivo text NOT NULL,
    superficie text NOT NULL,
    ruta text NOT NULL,
    accion text NOT NULL,
    actor_ref text,
    recurso_ref text,
    estado_http smallint NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    CONSTRAINT auditoria_frontera_personal_correlacion CHECK (
        correlacion_ref = 'corr_no_disponible'
        OR correlacion_ref ~ '^corr_[0-9a-f]{32}$'
    ),
    CONSTRAINT auditoria_frontera_personal_resultado CHECK (
        (motivo = 'peticion_invalida' AND estado_http = 400)
        OR (motivo = 'autenticacion_requerida' AND estado_http = 401)
        OR (motivo = 'acceso_denegado' AND estado_http = 403)
        OR (motivo = 'no_encontrada' AND estado_http = 404)
        OR (motivo = 'metodo_no_permitido' AND estado_http = 405)
        OR (motivo = 'representacion_no_admitida' AND estado_http = 406)
        OR (motivo = 'conflicto' AND estado_http = 409)
        OR (motivo = 'dependencia_no_disponible' AND estado_http = 503)
    ),
    CONSTRAINT auditoria_frontera_personal_superficie CHECK (
        superficie = 'api.personal.asignaciones_dietas'
    ),
    CONSTRAINT auditoria_frontera_personal_ruta CHECK (
        ruta IN (
            '/api/vec/personal/asignaciones-dietas',
            '/api/vec/personal/asignaciones-dietas/detalle',
            '/api/vec/personal/asignaciones-dietas/grupo',
            '/api/vec/personal/relaciones-dietas'
        )
    ),
    CONSTRAINT auditoria_frontera_personal_accion CHECK (
        accion IN (
            'consultar', 'registrar_inicial', 'corregir', 'grupo_corregir',
            'consultar_relaciones', 'metodo_no_admitido'
        )
    ),
    CONSTRAINT auditoria_frontera_personal_ruta_accion CHECK (
        accion = 'metodo_no_admitido'
        OR (ruta = '/api/vec/personal/asignaciones-dietas'
            AND accion = 'registrar_inicial')
        OR (ruta = '/api/vec/personal/asignaciones-dietas/detalle'
            AND accion IN ('consultar', 'corregir'))
        OR (ruta = '/api/vec/personal/asignaciones-dietas/grupo'
            AND accion = 'grupo_corregir')
        OR (ruta = '/api/vec/personal/relaciones-dietas'
            AND accion = 'consultar_relaciones')
    ),
    CONSTRAINT auditoria_frontera_personal_actor CHECK (
        actor_ref IS NULL OR (
            pg_catalog.length(actor_ref) BETWEEN 1 AND 512
            AND actor_ref ~ '^[A-Za-z0-9:_-]+$'
        )
    ),
    CONSTRAINT auditoria_frontera_personal_anonimo CHECK (
        motivo <> 'autenticacion_requerida' OR actor_ref IS NULL
    ),
    CONSTRAINT auditoria_frontera_personal_recurso CHECK (
        recurso_ref IS NULL OR
        (ruta = '/api/vec/personal/relaciones-dietas'
            AND recurso_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$') OR
        (ruta <> '/api/vec/personal/relaciones-dietas'
            AND recurso_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$')
    ),
    CONSTRAINT auditoria_frontera_personal_recurso_exigido CHECK (
        recurso_ref IS NOT NULL
        OR accion = 'metodo_no_admitido'
        OR NOT (
            (ruta IN (
                '/api/vec/personal/asignaciones-dietas/detalle',
                '/api/vec/personal/asignaciones-dietas/grupo'
            ) AND estado_http IN (403, 409))
            OR (ruta = '/api/vec/personal/relaciones-dietas'
                AND estado_http = 403)
        )
    )
);

CREATE INDEX auditoria_frontera_personal_correlacion_idx
    ON vec_personal.auditoria_frontera_asignacion_dietas (correlacion_ref, evento_id);

CREATE FUNCTION vec_personal.rechazar_mutacion_auditoria_frontera_asignacion_dietas_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '42501',
        MESSAGE = 'auditoria de frontera Personal inmutable';
END
$funcion$;

CREATE TRIGGER auditoria_frontera_personal_inmutable
BEFORE UPDATE OR DELETE ON vec_personal.auditoria_frontera_asignacion_dietas
FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_auditoria_frontera_asignacion_dietas_v1();

CREATE TRIGGER auditoria_frontera_personal_no_truncar
BEFORE TRUNCATE ON vec_personal.auditoria_frontera_asignacion_dietas
FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_auditoria_frontera_asignacion_dietas_v1();

CREATE FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
    p_correlacion_ref text,
    p_motivo text,
    p_superficie text,
    p_ruta text,
    p_accion text,
    p_actor_ref text,
    p_recurso_ref text,
    p_estado_http smallint
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
    IF current_user <> 'vec_personal_propietario'
       OR session_user = current_user
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members membresia
            WHERE membresia.member = session_user::pg_catalog.regrole
              AND membresia.roleid =
                  'vec_personal_registrador_frontera'::pg_catalog.regrole
              AND membresia.inherit_option
              AND NOT membresia.set_option
              AND NOT membresia.admin_option
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles rol
            WHERE rol.oid <> session_user::pg_catalog.regrole
              AND rol.oid <>
                  'vec_personal_registrador_frontera'::pg_catalog.regrole
              AND pg_catalog.pg_has_role(session_user, rol.oid, 'MEMBER')
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'registrador de frontera Personal invalido';
    END IF;

    IF p_correlacion_ref IS NULL
       OR (p_correlacion_ref <> 'corr_no_disponible'
           AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
       OR p_motivo IS NULL
       OR p_estado_http IS NULL
       OR NOT (
           (p_motivo = 'peticion_invalida' AND p_estado_http = 400)
           OR (p_motivo = 'autenticacion_requerida' AND p_estado_http = 401)
           OR (p_motivo = 'acceso_denegado' AND p_estado_http = 403)
           OR (p_motivo = 'no_encontrada' AND p_estado_http = 404)
           OR (p_motivo = 'metodo_no_permitido' AND p_estado_http = 405)
           OR (p_motivo = 'representacion_no_admitida' AND p_estado_http = 406)
           OR (p_motivo = 'conflicto' AND p_estado_http = 409)
           OR (p_motivo = 'dependencia_no_disponible' AND p_estado_http = 503)
       )
       OR p_superficie IS DISTINCT FROM 'api.personal.asignaciones_dietas'
       OR p_ruta IS NULL
       OR p_ruta NOT IN (
           '/api/vec/personal/asignaciones-dietas',
           '/api/vec/personal/asignaciones-dietas/detalle',
           '/api/vec/personal/asignaciones-dietas/grupo',
           '/api/vec/personal/relaciones-dietas'
       )
       OR p_accion IS NULL
       OR p_accion NOT IN (
           'consultar', 'registrar_inicial', 'corregir', 'grupo_corregir',
           'consultar_relaciones', 'metodo_no_admitido'
       )
       OR NOT (
           p_accion = 'metodo_no_admitido'
           OR (p_ruta = '/api/vec/personal/asignaciones-dietas'
               AND p_accion = 'registrar_inicial')
           OR (p_ruta = '/api/vec/personal/asignaciones-dietas/detalle'
               AND p_accion IN ('consultar', 'corregir'))
           OR (p_ruta = '/api/vec/personal/asignaciones-dietas/grupo'
               AND p_accion = 'grupo_corregir')
           OR (p_ruta = '/api/vec/personal/relaciones-dietas'
               AND p_accion = 'consultar_relaciones')
       )
       OR (p_motivo = 'autenticacion_requerida' AND p_actor_ref IS NOT NULL)
       OR (p_actor_ref IS NOT NULL AND (
           pg_catalog.length(p_actor_ref) NOT BETWEEN 1 AND 512
           OR p_actor_ref !~ '^[A-Za-z0-9:_-]+$'
       ))
       OR (p_recurso_ref IS NOT NULL AND NOT (
           (p_ruta = '/api/vec/personal/relaciones-dietas'
               AND p_recurso_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$')
           OR (p_ruta <> '/api/vec/personal/relaciones-dietas'
               AND p_recurso_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$')
       ))
       OR (p_recurso_ref IS NULL
           AND p_accion <> 'metodo_no_admitido' AND (
           (p_ruta IN (
               '/api/vec/personal/asignaciones-dietas/detalle',
               '/api/vec/personal/asignaciones-dietas/grupo'
           ) AND p_estado_http IN (403, 409))
           OR (p_ruta = '/api/vec/personal/relaciones-dietas'
               AND p_estado_http = 403)
       )) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'auditoria de frontera Personal invalida';
    END IF;

    INSERT INTO vec_personal.auditoria_frontera_asignacion_dietas (
        correlacion_ref, motivo, superficie, ruta, accion, actor_ref,
        recurso_ref, estado_http, registrada_en
    ) VALUES (
        p_correlacion_ref, p_motivo, p_superficie, p_ruta, p_accion, p_actor_ref,
        p_recurso_ref, p_estado_http,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    );
    RETURN true;
END
$funcion$;

ALTER TABLE vec_personal.auditoria_frontera_asignacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.auditoria_frontera_asignacion_dietas FORCE ROW LEVEL SECURITY;
CREATE POLICY auditoria_frontera_personal_insercion_registrador
    ON vec_personal.auditoria_frontera_asignacion_dietas
    FOR INSERT
    TO vec_personal_propietario
    WITH CHECK (
        superficie = 'api.personal.asignaciones_dietas'
        AND current_user = 'vec_personal_propietario'
        AND session_user <> current_user
        AND EXISTS (
            SELECT 1 FROM pg_catalog.pg_auth_members membresia
             WHERE membresia.member = (SELECT oid FROM pg_catalog.pg_roles
                                        WHERE rolname = session_user)
               AND membresia.roleid = (SELECT oid FROM pg_catalog.pg_roles
                                        WHERE rolname = 'vec_personal_registrador_frontera')
               AND membresia.inherit_option
               AND NOT membresia.set_option
               AND NOT membresia.admin_option
        )
        AND NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_roles rol
             WHERE rol.oid <> (SELECT oid FROM pg_catalog.pg_roles
                               WHERE rolname = session_user)
               AND rol.oid <> (SELECT oid FROM pg_catalog.pg_roles
                                WHERE rolname = 'vec_personal_registrador_frontera')
               AND pg_catalog.pg_has_role(session_user, rol.oid, 'MEMBER')
        )
    );

REVOKE ALL ON TABLE vec_personal.auditoria_frontera_asignacion_dietas
    FROM PUBLIC, vec_personal_ejecutor, vec_personal_registrador_frontera;
REVOKE ALL ON SEQUENCE vec_personal.auditoria_frontera_asignacion_dietas_evento_id_seq
    FROM PUBLIC, vec_personal_ejecutor, vec_personal_registrador_frontera;
REVOKE ALL ON FUNCTION vec_personal.rechazar_mutacion_auditoria_frontera_asignacion_dietas_v1()
    FROM PUBLIC, vec_personal_ejecutor, vec_personal_registrador_frontera;
REVOKE ALL ON FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)
    FROM PUBLIC, vec_personal_ejecutor, vec_personal_migrador;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)
    TO vec_personal_registrador_frontera;

COMMIT;
