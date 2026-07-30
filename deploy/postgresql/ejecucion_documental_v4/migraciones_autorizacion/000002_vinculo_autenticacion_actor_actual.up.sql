-- Evolucion compatible del contrato de decision y fuente durable minima para
-- revalidar sesion/control y ContextoActor. No modifica 000001 ni convierte
-- el documento historico de treinta claves: lo conserva como lectura valida,
-- pero toda decision nueva debe aportar el bloque actual de veinticinco datos.
--
-- Estas tablas son la fuente transaccional local del nucleo de autorizacion.
-- Su alimentacion desde el IdP, el gestor de sesiones y el maestro de personas
-- debe realizarla una frontera autoritativa separada; esta migracion no simula
-- que PostgreSQL pueda consultar o bloquear por si solo un directorio externo.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regprocedure(
        'vec_autorizacion.documento_decision_estructura_valida(jsonb)'
    ) IS NULL
       OR to_regclass('vec_autorizacion.decision_autorizacion') IS NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname = 'vec_autorizacion'
              AND p.proname IN (
                  'documento_decision_estructura_valida_v1_legacy',
                  'vinculo_autenticacion_actor_v1_estructura_valida',
                  'revalidar_vinculo_autenticacion_actor_actual_v1'
              )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para migracion de vinculo actual';
    END IF;
END
$prevalidacion$;

-- El OID y el cuerpo originales quedan intactos con nombre explicito. Permite
-- validar filas historicas y restaurar el contrato anterior en el down.
ALTER FUNCTION vec_autorizacion.documento_decision_estructura_valida(jsonb)
    RENAME TO documento_decision_estructura_valida_v1_legacy;

CREATE FUNCTION vec_autorizacion.vinculo_autenticacion_actor_v1_estructura_valida(
    p_vinculo jsonb
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    clave text;
    autenticacion_verificada timestamptz;
    sesion_emitida timestamptz;
    sesion_revalidada timestamptz;
    sesion_valida_hasta timestamptz;
BEGIN
    IF p_vinculo IS NULL OR jsonb_typeof(p_vinculo) <> 'object'
       OR pg_column_size(p_vinculo) > 65536
       OR (SELECT count(*) FROM jsonb_object_keys(p_vinculo)) <> 25
       OR NOT (p_vinculo ?& ARRAY[
           'bloque_version', 'autenticacion_ref',
           'autenticacion_huella_sha256', 'asercion_ref', 'sesion_ref',
           'control_sesion_ref', 'control_sesion_revision',
           'control_sesion_huella_sha256', 'cuenta_ref',
           'cuenta_ordinaria_ref', 'principal_id', 'perfil_activo_ref',
           'cuenta_privilegiada', 'superficie', 'metodo_observado',
           'garantia_observada', 'politica_garantia_ref',
           'politica_garantia_huella_sha256',
           'autenticacion_verificada_en', 'sesion_emitida_en',
           'sesion_valida_hasta', 'sesion_revalidada_en',
           'contexto_actor_ref', 'contexto_actor_version',
           'contexto_actor_huella_sha256'
       ])
       OR jsonb_typeof(p_vinculo -> 'bloque_version') <> 'number'
       OR (p_vinculo ->> 'bloque_version') <> '1'
       OR vec_autorizacion.entero_uint64_json_valido(
           p_vinculo -> 'control_sesion_revision'
       ) IS NOT TRUE
       OR vec_autorizacion.entero_uint64_json_valido(
           p_vinculo -> 'contexto_actor_version'
       ) IS NOT TRUE
       OR jsonb_typeof(p_vinculo -> 'cuenta_privilegiada') <> 'boolean' THEN
        RETURN false;
    END IF;

    FOREACH clave IN ARRAY ARRAY[
        'autenticacion_huella_sha256', 'control_sesion_huella_sha256',
        'politica_garantia_huella_sha256', 'contexto_actor_huella_sha256'
    ] LOOP
        IF jsonb_typeof(p_vinculo -> clave) <> 'string'
           OR (p_vinculo ->> clave) !~ '^[0-9a-f]{64}$' THEN
            RETURN false;
        END IF;
    END LOOP;

    IF (p_vinculo ->> 'autenticacion_ref') !~ '^aut_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'asercion_ref') !~ '^ase_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'sesion_ref') !~ '^ses_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'control_sesion_ref') !~ '^cse_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'cuenta_ref') !~ '^cta_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'cuenta_ordinaria_ref') !~ '^cta_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'principal_id') !~ '^per_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'perfil_activo_ref') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'politica_garantia_ref') !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'contexto_actor_ref') !~ '^vca_[A-Za-z0-9_-]{22,128}$'
       OR (p_vinculo ->> 'superficie') NOT IN (
           'externa_personal', 'interna_corporativa',
           'administracion_privilegiada'
       )
       OR (p_vinculo ->> 'metodo_observado') NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
       )
       OR (p_vinculo ->> 'garantia_observada') NOT IN (
           'bajo', 'sustancial', 'alto'
       ) THEN
        RETURN false;
    END IF;

    IF (p_vinculo ->> 'cuenta_privilegiada')::boolean THEN
        IF p_vinculo ->> 'superficie' <> 'administracion_privilegiada'
           OR p_vinculo ->> 'cuenta_ref' = p_vinculo ->> 'cuenta_ordinaria_ref' THEN
            RETURN false;
        END IF;
    ELSIF p_vinculo ->> 'superficie' = 'administracion_privilegiada'
       OR p_vinculo ->> 'cuenta_ref' <> p_vinculo ->> 'cuenta_ordinaria_ref' THEN
        RETURN false;
    END IF;

    FOREACH clave IN ARRAY ARRAY[
        'autenticacion_verificada_en', 'sesion_emitida_en',
        'sesion_valida_hasta', 'sesion_revalidada_en'
    ] LOOP
        IF jsonb_typeof(p_vinculo -> clave) <> 'string'
           OR vec_autorizacion.instante_utc_microsegundo_valido(
               p_vinculo ->> clave
           ) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END LOOP;

    autenticacion_verificada :=
        (p_vinculo ->> 'autenticacion_verificada_en')::timestamptz;
    sesion_emitida := (p_vinculo ->> 'sesion_emitida_en')::timestamptz;
    sesion_revalidada := (p_vinculo ->> 'sesion_revalidada_en')::timestamptz;
    sesion_valida_hasta := (p_vinculo ->> 'sesion_valida_hasta')::timestamptz;
    RETURN autenticacion_verificada <= sesion_emitida
       AND sesion_revalidada >= autenticacion_verificada
       AND sesion_revalidada >= sesion_emitida
       AND sesion_valida_hasta > sesion_revalidada;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow THEN
        RETURN false;
END
$funcion$;

-- Admite la forma historica de 30 claves para no invalidar evidencia ya
-- almacenada. La forma nueva exige exactamente una clave adicional valida.
CREATE FUNCTION vec_autorizacion.documento_decision_estructura_valida(
    p_documento jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT CASE
        WHEN vec_autorizacion.documento_decision_estructura_valida_v1_legacy(
            p_documento
        ) IS TRUE THEN true
        WHEN jsonb_typeof(p_documento) = 'object'
         AND (SELECT count(*) FROM jsonb_object_keys(p_documento)) = 31
         AND p_documento ? 'vinculo_autenticacion_actor'
         AND vec_autorizacion.documento_decision_estructura_valida_v1_legacy(
             p_documento - 'vinculo_autenticacion_actor'
         ) IS TRUE
         AND vec_autorizacion.vinculo_autenticacion_actor_v1_estructura_valida(
             p_documento -> 'vinculo_autenticacion_actor'
         ) IS TRUE
        THEN true
        ELSE false
    END
$funcion$;

ALTER TABLE vec_autorizacion.decision_autorizacion
    DROP CONSTRAINT decision_documentos_tipo;
ALTER TABLE vec_autorizacion.decision_autorizacion
    ADD CONSTRAINT decision_documentos_tipo_v2 CHECK (
        vec_autorizacion.documento_decision_estructura_valida(documento) IS TRUE
    );

CREATE TABLE vec_autorizacion.sesion_autenticacion_v1 (
    sesion_ref text PRIMARY KEY,
    autenticacion_ref text NOT NULL,
    autenticacion_huella_sha256 text NOT NULL,
    asercion_ref text NOT NULL,
    cuenta_ref text NOT NULL,
    cuenta_ordinaria_ref text NOT NULL,
    cuenta_privilegiada boolean NOT NULL,
    superficie text NOT NULL,
    metodo_observado text NOT NULL,
    garantia_observada text NOT NULL,
    politica_garantia_ref text NOT NULL,
    politica_garantia_huella_sha256 text NOT NULL,
    autenticacion_verificada_en timestamptz(6) NOT NULL,
    sesion_emitida_en timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT sesion_autenticacion_referencias CHECK (
        autenticacion_ref ~ '^aut_[A-Za-z0-9_-]{22,128}$'
        AND asercion_ref ~ '^ase_[A-Za-z0-9_-]{22,128}$'
        AND sesion_ref ~ '^ses_[A-Za-z0-9_-]{22,128}$'
        AND cuenta_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$'
        AND cuenta_ordinaria_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$'
        AND politica_garantia_ref ~ '^pga_[A-Za-z0-9_-]{22,128}$'
    ),
    CONSTRAINT sesion_autenticacion_huellas CHECK (
        autenticacion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND politica_garantia_huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT sesion_autenticacion_perfil_cerrado CHECK (
        superficie IN (
            'externa_personal', 'interna_corporativa',
            'administracion_privilegiada'
        )
        AND metodo_observado IN (
            'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad'
        )
        AND garantia_observada IN ('bajo', 'sustancial', 'alto')
        AND (
            (cuenta_privilegiada
             AND superficie = 'administracion_privilegiada'
             AND cuenta_ref <> cuenta_ordinaria_ref)
            OR
            (NOT cuenta_privilegiada
             AND superficie <> 'administracion_privilegiada'
             AND cuenta_ref = cuenta_ordinaria_ref)
        )
    ),
    CONSTRAINT sesion_autenticacion_cronologia CHECK (
        autenticacion_verificada_en <= sesion_emitida_en
    ),
    UNIQUE (autenticacion_ref, sesion_ref)
);

CREATE TABLE vec_autorizacion.control_sesion_v1 (
    control_sesion_ref text NOT NULL,
    revision numeric(20, 0) NOT NULL,
    sesion_ref text NOT NULL
        REFERENCES vec_autorizacion.sesion_autenticacion_v1(sesion_ref),
    estado text NOT NULL,
    huella_sha256 text NOT NULL,
    sesion_revalidada_en timestamptz(6) NOT NULL,
    sesion_valida_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (control_sesion_ref, revision),
    CONSTRAINT control_sesion_referencia CHECK (
        control_sesion_ref ~ '^cse_[A-Za-z0-9_-]{22,128}$'
    ),
    CONSTRAINT control_sesion_revision CHECK (
        revision BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT control_sesion_estado CHECK (estado IN ('activa', 'revocada')),
    CONSTRAINT control_sesion_huella CHECK (
        huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT control_sesion_cronologia CHECK (
        sesion_valida_hasta > sesion_revalidada_en
    ),
    UNIQUE (sesion_ref, control_sesion_ref, revision)
);

CREATE TABLE vec_autorizacion.control_sesion_actual_v1 (
    sesion_ref text PRIMARY KEY,
    control_sesion_ref text NOT NULL,
    revision numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    CONSTRAINT control_sesion_actual_acto CHECK (
        vec_autorizacion.texto_positivo_valido(acto_ref, 512) IS TRUE
    ),
    FOREIGN KEY (sesion_ref, control_sesion_ref, revision)
        REFERENCES vec_autorizacion.control_sesion_v1(
            sesion_ref, control_sesion_ref, revision
        )
);

CREATE TABLE vec_autorizacion.contexto_actor_v1 (
    contexto_actor_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    cuenta_ref text NOT NULL,
    principal_id text NOT NULL,
    perfil_activo_ref text NOT NULL,
    estado text NOT NULL,
    huella_sha256 text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6) NOT NULL,
    registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (contexto_actor_ref, version),
    CONSTRAINT contexto_actor_referencias CHECK (
        contexto_actor_ref ~ '^vca_[A-Za-z0-9_-]{22,128}$'
        AND cuenta_ref ~ '^cta_[A-Za-z0-9_-]{22,128}$'
        AND principal_id ~ '^per_[A-Za-z0-9_-]{22,128}$'
        AND perfil_activo_ref ~ '^prf_[A-Za-z0-9_-]{22,128}$'
    ),
    CONSTRAINT contexto_actor_version CHECK (
        version BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT contexto_actor_estado CHECK (estado IN ('activo', 'revocado')),
    CONSTRAINT contexto_actor_huella CHECK (
        huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT contexto_actor_cronologia CHECK (vigente_hasta > vigente_desde),
    UNIQUE (
        cuenta_ref, perfil_activo_ref, contexto_actor_ref, version
    )
);

CREATE TABLE vec_autorizacion.contexto_actor_actual_v1 (
    cuenta_ref text NOT NULL,
    perfil_activo_ref text NOT NULL,
    contexto_actor_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    PRIMARY KEY (cuenta_ref, perfil_activo_ref),
    CONSTRAINT contexto_actor_actual_acto CHECK (
        vec_autorizacion.texto_positivo_valido(acto_ref, 512) IS TRUE
    ),
    FOREIGN KEY (
        cuenta_ref, perfil_activo_ref, contexto_actor_ref, version
    ) REFERENCES vec_autorizacion.contexto_actor_v1(
        cuenta_ref, perfil_activo_ref, contexto_actor_ref, version
    )
);

CREATE FUNCTION vec_autorizacion.validar_avance_control_sesion_actual_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    estado_anterior text;
BEGIN
    SELECT estado INTO STRICT estado_anterior
      FROM vec_autorizacion.control_sesion_v1
     WHERE control_sesion_ref = OLD.control_sesion_ref
       AND revision = OLD.revision;
    IF NEW.sesion_ref IS DISTINCT FROM OLD.sesion_ref
       OR NEW.control_sesion_ref IS DISTINCT FROM OLD.control_sesion_ref
       OR NEW.revision IS DISTINCT FROM OLD.revision + 1
       OR NEW.actualizada_en <= OLD.actualizada_en
       OR estado_anterior = 'revocada' THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de control de sesion invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.validar_avance_contexto_actor_actual_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    estado_anterior text;
BEGIN
    SELECT estado INTO STRICT estado_anterior
      FROM vec_autorizacion.contexto_actor_v1
     WHERE contexto_actor_ref = OLD.contexto_actor_ref
       AND version = OLD.version;
    IF NEW.cuenta_ref IS DISTINCT FROM OLD.cuenta_ref
       OR NEW.perfil_activo_ref IS DISTINCT FROM OLD.perfil_activo_ref
       OR NEW.contexto_actor_ref IS DISTINCT FROM OLD.contexto_actor_ref
       OR NEW.version IS DISTINCT FROM OLD.version + 1
       OR NEW.actualizada_en <= OLD.actualizada_en
       OR estado_anterior = 'revocado' THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de ContextoActor invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'sesion_autenticacion_v1', 'control_sesion_v1', 'contexto_actor_v1'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
    END LOOP;
    FOREACH tabla IN ARRAY ARRAY[
        'control_sesion_actual_v1', 'contexto_actor_actual_v1'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_no_eliminar BEFORE DELETE OR TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada()',
            tabla, tabla
        );
    END LOOP;
END
$protecciones$;

CREATE TRIGGER control_sesion_actual_v1_avance
    BEFORE UPDATE ON vec_autorizacion.control_sesion_actual_v1
    FOR EACH ROW EXECUTE FUNCTION
        vec_autorizacion.validar_avance_control_sesion_actual_v1();
CREATE TRIGGER contexto_actor_actual_v1_avance
    BEFORE UPDATE ON vec_autorizacion.contexto_actor_actual_v1
    FOR EACH ROW EXECUTE FUNCTION
        vec_autorizacion.validar_avance_contexto_actor_actual_v1();

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'sesion_autenticacion_v1', 'control_sesion_v1',
        'control_sesion_actual_v1', 'contexto_actor_v1',
        'contexto_actor_actual_v1'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY', tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY', tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_autorizacion_propietario',
            'vec_autorizacion_propietario'
        );
    END LOOP;
END
$rls$;

-- Bloquea los dos punteros actuales. Cualquier revocacion o avance concurrente
-- necesita actualizar esos mismos registros y queda serializado con el uso.
CREATE FUNCTION vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
    p_vinculo jsonb,
    p_principal_id text,
    p_perfil_activo_ref text,
    p_contexto_actor_huella_sha256 text,
    p_emitida_en timestamptz,
    p_valida_hasta timestamptz,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    sesion record;
    actor record;
BEGIN
    IF vec_autorizacion.vinculo_autenticacion_actor_v1_estructura_valida(
           p_vinculo
       ) IS NOT TRUE
       OR p_principal_id IS DISTINCT FROM p_vinculo ->> 'principal_id'
       OR p_perfil_activo_ref IS DISTINCT FROM p_vinculo ->> 'perfil_activo_ref'
       OR p_contexto_actor_huella_sha256 IS DISTINCT FROM
           p_vinculo ->> 'contexto_actor_huella_sha256'
       OR p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL
       OR p_valida_hasta <= p_emitida_en THEN
        RETURN false;
    END IF;

    SELECT base.autenticacion_ref, base.autenticacion_huella_sha256,
           base.asercion_ref, base.sesion_ref, base.cuenta_ref,
           base.cuenta_ordinaria_ref, base.cuenta_privilegiada,
           base.superficie, base.metodo_observado, base.garantia_observada,
           base.politica_garantia_ref,
           base.politica_garantia_huella_sha256,
           base.autenticacion_verificada_en, base.sesion_emitida_en,
           control.control_sesion_ref, control.revision,
           control.estado, control.huella_sha256,
           control.sesion_revalidada_en, control.sesion_valida_hasta
      INTO sesion
      FROM vec_autorizacion.sesion_autenticacion_v1 AS base
      JOIN vec_autorizacion.control_sesion_actual_v1 AS actual
        ON actual.sesion_ref = base.sesion_ref
      JOIN vec_autorizacion.control_sesion_v1 AS control
        ON control.sesion_ref = actual.sesion_ref
       AND control.control_sesion_ref = actual.control_sesion_ref
       AND control.revision = actual.revision
     WHERE base.sesion_ref = p_vinculo ->> 'sesion_ref'
       AND base.autenticacion_ref = p_vinculo ->> 'autenticacion_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND OR sesion.estado <> 'activa'
       OR sesion.autenticacion_huella_sha256 IS DISTINCT FROM
           p_vinculo ->> 'autenticacion_huella_sha256'
       OR sesion.asercion_ref IS DISTINCT FROM p_vinculo ->> 'asercion_ref'
       OR sesion.control_sesion_ref IS DISTINCT FROM
           p_vinculo ->> 'control_sesion_ref'
       OR sesion.revision IS DISTINCT FROM
           (p_vinculo ->> 'control_sesion_revision')::numeric
       OR sesion.huella_sha256 IS DISTINCT FROM
           p_vinculo ->> 'control_sesion_huella_sha256'
       OR sesion.cuenta_ref IS DISTINCT FROM p_vinculo ->> 'cuenta_ref'
       OR sesion.cuenta_ordinaria_ref IS DISTINCT FROM
           p_vinculo ->> 'cuenta_ordinaria_ref'
       OR sesion.cuenta_privilegiada IS DISTINCT FROM
           (p_vinculo ->> 'cuenta_privilegiada')::boolean
       OR sesion.superficie IS DISTINCT FROM p_vinculo ->> 'superficie'
       OR sesion.metodo_observado IS DISTINCT FROM
           p_vinculo ->> 'metodo_observado'
       OR sesion.garantia_observada IS DISTINCT FROM
           p_vinculo ->> 'garantia_observada'
       OR sesion.politica_garantia_ref IS DISTINCT FROM
           p_vinculo ->> 'politica_garantia_ref'
       OR sesion.politica_garantia_huella_sha256 IS DISTINCT FROM
           p_vinculo ->> 'politica_garantia_huella_sha256'
       OR sesion.autenticacion_verificada_en IS DISTINCT FROM
           (p_vinculo ->> 'autenticacion_verificada_en')::timestamptz
       OR sesion.sesion_emitida_en IS DISTINCT FROM
           (p_vinculo ->> 'sesion_emitida_en')::timestamptz
       OR sesion.sesion_revalidada_en IS DISTINCT FROM
           (p_vinculo ->> 'sesion_revalidada_en')::timestamptz
       OR sesion.sesion_valida_hasta IS DISTINCT FROM
           (p_vinculo ->> 'sesion_valida_hasta')::timestamptz THEN
        RETURN false;
    END IF;

    SELECT contexto.contexto_actor_ref, contexto.version,
           contexto.cuenta_ref, contexto.principal_id,
           contexto.perfil_activo_ref, contexto.estado,
           contexto.huella_sha256, contexto.vigente_desde,
           contexto.vigente_hasta
      INTO actor
      FROM vec_autorizacion.contexto_actor_actual_v1 AS actual
      JOIN vec_autorizacion.contexto_actor_v1 AS contexto
        ON contexto.cuenta_ref = actual.cuenta_ref
       AND contexto.perfil_activo_ref = actual.perfil_activo_ref
       AND contexto.contexto_actor_ref = actual.contexto_actor_ref
       AND contexto.version = actual.version
     WHERE actual.cuenta_ref = p_vinculo ->> 'cuenta_ref'
       AND actual.perfil_activo_ref = p_vinculo ->> 'perfil_activo_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND OR actor.estado <> 'activo'
       OR actor.contexto_actor_ref IS DISTINCT FROM
           p_vinculo ->> 'contexto_actor_ref'
       OR actor.version IS DISTINCT FROM
           (p_vinculo ->> 'contexto_actor_version')::numeric
       OR actor.cuenta_ref IS DISTINCT FROM p_vinculo ->> 'cuenta_ref'
       OR actor.principal_id IS DISTINCT FROM p_vinculo ->> 'principal_id'
       OR actor.perfil_activo_ref IS DISTINCT FROM
           p_vinculo ->> 'perfil_activo_ref'
       OR actor.huella_sha256 IS DISTINCT FROM
           p_vinculo ->> 'contexto_actor_huella_sha256' THEN
        RETURN false;
    END IF;

    RETURN p_emitida_en >= sesion.sesion_revalidada_en
       AND p_valida_hasta <= sesion.sesion_valida_hasta
       AND p_instante >= sesion.sesion_revalidada_en
       AND p_instante < sesion.sesion_valida_hasta
       AND p_emitida_en >= actor.vigente_desde
       AND p_valida_hasta <= actor.vigente_hasta
       AND p_instante >= actor.vigente_desde
       AND p_instante < actor.vigente_hasta;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION vec_autorizacion.exigir_vinculo_actual_decision_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6);
BEGIN
    instante := clock_timestamp();
    IF vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
        NEW.documento -> 'vinculo_autenticacion_actor',
        NEW.principal_id,
        NEW.perfil_activo_ref,
        NEW.documento -> 'vinculo_autenticacion_actor'
            ->> 'contexto_actor_huella_sha256',
        NEW.emitida_en,
        NEW.valida_hasta,
        instante
    ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'estado de identidad obsoleto';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER decision_exige_vinculo_actual_v1
    BEFORE INSERT ON vec_autorizacion.decision_autorizacion
    FOR EACH ROW EXECUTE FUNCTION
        vec_autorizacion.exigir_vinculo_actual_decision_v1();

REVOKE ALL ON ALL TABLES IN SCHEMA vec_autorizacion FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_autorizacion FROM PUBLIC;
-- PostgreSQL no aplica ALTER DEFAULT PRIVILEGES ON TYPES a los tipos
-- compuestos implícitos creados junto con una tabla.
REVOKE ALL ON TYPE vec_autorizacion.sesion_autenticacion_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.control_sesion_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.control_sesion_actual_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.contexto_actor_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.contexto_actor_actual_v1 FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
        jsonb, text, text, text, timestamptz, timestamptz, timestamptz
    ) FROM PUBLIC;

COMMENT ON FUNCTION
    vec_autorizacion.revalidar_vinculo_autenticacion_actor_actual_v1(
        jsonb, text, text, text, timestamptz, timestamptz, timestamptz
    ) IS
    'Revalida y bloquea los punteros locales actuales de sesion y ContextoActor; no consulta un IdP remoto.';

COMMIT;
