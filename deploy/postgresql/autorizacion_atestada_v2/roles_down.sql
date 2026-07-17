-- Retirada de roles despues de retirar la migracion. Requiere superusuario.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:roles_up:v1', 0
    )
);

DO $prevalidacion$
DECLARE
    rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de roles atestados V2 requiere superusuario';
    END IF;
    IF pg_catalog.to_regnamespace('vec_autorizacion_atestada_v2') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retire primero el esquema atestado V2';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_autorizacion_atestada_v2_propietario',
        'vec_autorizacion_atestada_v2_migrador',
        'vec_autorizacion_atestada_v2_emisor_capacidad',
        'vec_autorizacion_atestada_v2_consumidor'
    ] LOOP
        IF EXISTS (
            SELECT 1
              FROM pg_catalog.pg_auth_members AS membresia
              JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
              JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
             WHERE (grupo.rolname = rol OR miembro.rolname = rol)
               AND NOT (
                   grupo.rolname = 'vec_autorizacion_atestada_v2_propietario'
                   AND miembro.rolname = 'vec_autorizacion_atestada_v2_migrador'
               )
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'un rol atestado V2 conserva membresias externas',
                DETAIL = rol;
        END IF;
    END LOOP;
END
$prevalidacion$;

DROP EVENT TRIGGER IF EXISTS
    vec_autorizacion_atestada_v2_cerrar_acl_tipos;
DROP FUNCTION IF EXISTS
    vec_autorizacion_atestada_v2_guardia.cerrar_acl_tipos();
DROP SCHEMA IF EXISTS vec_autorizacion_atestada_v2_guardia;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
        bytea, bytea
    ) FROM vec_autorizacion_atestada_v2_propietario;
REVOKE REFERENCES (decision_ref) ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FROM vec_autorizacion_atestada_v2_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_autorizacion_atestada_v2_propietario;
REVOKE REFERENCES (configuracion_revision, clave_id, version) ON
    vec_confianza_atestacion_v2.configuracion_raiz
    FROM vec_autorizacion_atestada_v2_propietario;
REVOKE USAGE ON SCHEMA vec_confianza_atestacion_v2
    FROM vec_autorizacion_atestada_v2_propietario;
REVOKE EXECUTE ON FUNCTION public.hmac(bytea, bytea, text)
    FROM vec_autorizacion_atestada_v2_propietario;
REVOKE USAGE ON SCHEMA public
    FROM vec_autorizacion_atestada_v2_propietario;

DO $retirar_privilegios_base$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_atestada_v2_propietario, vec_autorizacion_atestada_v2_migrador, vec_autorizacion_atestada_v2_emisor_capacidad, vec_autorizacion_atestada_v2_consumidor',
        current_database()
    );
END
$retirar_privilegios_base$;

REVOKE vec_autorizacion_atestada_v2_propietario
    FROM vec_autorizacion_atestada_v2_migrador;
DROP ROLE vec_autorizacion_atestada_v2_consumidor;
DROP ROLE vec_autorizacion_atestada_v2_emisor_capacidad;
DROP ROLE vec_autorizacion_atestada_v2_migrador;
DROP ROLE vec_autorizacion_atestada_v2_propietario;

COMMIT;
