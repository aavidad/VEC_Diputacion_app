-- Identidad externa: sin filas nuevas en cuenta, alias, estado o consumo comunes.
-- El operador interno aprueba una huella exacta en aprobacion_alta; el LOGIN
-- externo solo registra/revalida una cuenta externa previamente provisionada.
BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $pre$
BEGIN
    IF pg_catalog.to_regnamespace('vec_identidad_externa_v1') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n
                       WHERE n.nspname='vec_identidad_externa_v1'
                         AND n.nspowner='vec_identidad_sesiones_v1_propietario'::regrole)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n
                    ON n.oid=c.relnamespace WHERE n.nspname='vec_identidad_externa_v1')
       OR pg_catalog.to_regrole('vec_identidad_externa_v1_provisionador') IS NULL
       OR pg_catalog.to_regrole('vec_identidad_externa_v1_registrador') IS NULL
       OR pg_catalog.to_regrole('vec_identidad_externa_v1_revalidador') IS NULL
       OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.nueva_referencia(text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.coordenadas_hmac_validas(text,text,text,bigint)') IS NULL THEN
        RAISE EXCEPTION 'identidad externa: precondiciones incumplidas' USING ERRCODE='55000';
    END IF;
END $pre$;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    IN SCHEMA vec_identidad_externa_v1 REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    IN SCHEMA vec_identidad_externa_v1 REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    IN SCHEMA vec_identidad_externa_v1 REVOKE ALL ON TYPES FROM PUBLIC;

-- Solo la autoridad de aprovisionamiento interno inserta esta aprobación,
-- fuera de las capacidades de cualquier LOGIN del portal. Es inmutable.
CREATE TABLE vec_identidad_externa_v1.aprobacion_alta (
    aprobacion_ref text PRIMARY KEY,
    huella_material_sha256 text NOT NULL,
    huella_preimagen_sha256 text NOT NULL,
    revision_esperada bigint NOT NULL CHECK (revision_esperada = 0),
    aprobada_por text NOT NULL DEFAULT session_user,
    aprobada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    CHECK (aprobacion_ref ~ '^apr_[A-Za-z0-9_-]{22,128}$'),
    CHECK (huella_material_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (pg_catalog.octet_length(aprobada_por) BETWEEN 1 AND 128)
);

CREATE TABLE vec_identidad_externa_v1.cuenta (
    cuenta_ref text PRIMARY KEY,
    dominio_hmac_ref text NOT NULL,
    huella_material_sha256 text NOT NULL,
    aprobacion_ref text NOT NULL UNIQUE REFERENCES vec_identidad_externa_v1.aprobacion_alta(aprobacion_ref),
    operacion_ref text NOT NULL UNIQUE,
    provisionada_en timestamptz(6) NOT NULL,
    CHECK (vec_identidad_sesiones_v1.referencia_valida(cuenta_ref,'cta_') IS TRUE),
    CHECK (vec_identidad_sesiones_v1.referencia_valida(dominio_hmac_ref, 'idh_') IS TRUE),
    CHECK (huella_material_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (vec_identidad_sesiones_v1.referencia_valida(operacion_ref, 'opr_') IS TRUE)
);

CREATE TABLE vec_identidad_externa_v1.alias_cuenta (
    alias_ref text PRIMARY KEY,
    cuenta_ref text NOT NULL REFERENCES vec_identidad_externa_v1.cuenta(cuenta_ref),
    esquema_hmac text NOT NULL,
    dominio_hmac_ref text NOT NULL,
    clave_hmac_id text NOT NULL,
    clave_hmac_version bigint NOT NULL,
    cuenta_id_hmac bytea NOT NULL,
    sujeto_id_hmac bytea NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    operacion_ref text NOT NULL UNIQUE,
    CHECK (vec_identidad_sesiones_v1.referencia_valida(alias_ref, 'ali_') IS TRUE),
    CHECK (vec_identidad_sesiones_v1.coordenadas_hmac_validas(
        esquema_hmac, dominio_hmac_ref, clave_hmac_id, clave_hmac_version) IS TRUE),
    CHECK (vec_identidad_sesiones_v1.huella_hmac_valida(cuenta_id_hmac) IS TRUE
       AND vec_identidad_sesiones_v1.huella_hmac_valida(sujeto_id_hmac) IS TRUE
       AND cuenta_id_hmac <> sujeto_id_hmac),
    UNIQUE (esquema_hmac, dominio_hmac_ref, clave_hmac_id,
            clave_hmac_version, cuenta_id_hmac)
);

CREATE TABLE vec_identidad_externa_v1.estado_cuenta (
    cuenta_ref text NOT NULL REFERENCES vec_identidad_externa_v1.cuenta(cuenta_ref),
    revision bigint NOT NULL CHECK (revision > 0),
    estado text NOT NULL CHECK (estado IN ('activa', 'inactiva')),
    registrada_en timestamptz(6) NOT NULL,
    operacion_ref text NOT NULL UNIQUE,
    PRIMARY KEY (cuenta_ref, revision)
);
CREATE TABLE vec_identidad_externa_v1.estado_actual (
    cuenta_ref text PRIMARY KEY,
    revision bigint NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    operacion_ref text NOT NULL,
    FOREIGN KEY (cuenta_ref, revision)
        REFERENCES vec_identidad_externa_v1.estado_cuenta(cuenta_ref, revision)
);

CREATE TABLE vec_identidad_externa_v1.consumo_aprobacion (
    aprobacion_ref text PRIMARY KEY REFERENCES vec_identidad_externa_v1.aprobacion_alta(aprobacion_ref),
    cuenta_ref text NOT NULL UNIQUE REFERENCES vec_identidad_externa_v1.cuenta(cuenta_ref),
    operacion_ref text NOT NULL UNIQUE,
    huella_material_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL
);

CREATE TABLE vec_identidad_externa_v1.sesion (
    sesion_ref text PRIMARY KEY,
    autenticacion_ref text NOT NULL UNIQUE,
    asercion_ref text NOT NULL UNIQUE,
    operacion_ref text NOT NULL UNIQUE,
    cuenta_ref text NOT NULL REFERENCES vec_identidad_externa_v1.cuenta(cuenta_ref),
    cuenta_revision bigint NOT NULL,
    esquema_hmac text NOT NULL,
    dominio_hmac_ref text NOT NULL,
    clave_hmac_id text NOT NULL,
    clave_hmac_version bigint NOT NULL,
    asercion_id_hmac bytea NOT NULL,
    sesion_id_hmac bytea NOT NULL,
    sujeto_id_hmac bytea NOT NULL,
    cuenta_id_hmac bytea NOT NULL,
    metodo_observado text NOT NULL,
    garantia_observada text NOT NULL,
    autenticacion_huella_sha256 text NOT NULL,
    autenticacion_verificada_en timestamptz(6) NOT NULL,
    sesion_emitida_en timestamptz(6) NOT NULL,
    politica_garantia_ref text NOT NULL,
    politica_garantia_huella_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (cuenta_ref, cuenta_revision)
        REFERENCES vec_identidad_externa_v1.estado_cuenta(cuenta_ref, revision),
    UNIQUE (esquema_hmac, dominio_hmac_ref, clave_hmac_id,
            clave_hmac_version, asercion_id_hmac),
    UNIQUE (dominio_hmac_ref, autenticacion_huella_sha256),
    CHECK (metodo_observado IN ('certificado', 'dnie', 'sso', 'clave', 'kerberos_ad')),
    CHECK (garantia_observada IN ('sustancial', 'alto')),
    CHECK (autenticacion_huella_sha256 ~ '^[0-9a-f]{64}$'
       AND autenticacion_huella_sha256 <> repeat('0',64)),
    CHECK (politica_garantia_ref ~ '^pga_[A-Za-z0-9_-]{22,128}$'),
    CHECK (politica_garantia_huella_sha256 ~ '^[0-9a-f]{64}$'
       AND politica_garantia_huella_sha256 <> repeat('0',64))
);
CREATE INDEX sesion_id_externa_v1_idx ON vec_identidad_externa_v1.sesion
    (esquema_hmac, dominio_hmac_ref, clave_hmac_id,
     clave_hmac_version, sesion_id_hmac);

CREATE TABLE vec_identidad_externa_v1.control_sesion (
    control_sesion_ref text PRIMARY KEY,
    sesion_ref text NOT NULL UNIQUE REFERENCES vec_identidad_externa_v1.sesion(sesion_ref),
    revision bigint NOT NULL CHECK (revision = 1),
    estado text NOT NULL CHECK (estado = 'activa'),
    huella_sha256 text NOT NULL,
    sesion_revalidada_en timestamptz(6) NOT NULL,
    sesion_valida_hasta timestamptz(6) NOT NULL,
    CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (sesion_valida_hasta > sesion_revalidada_en)
);

CREATE TABLE vec_identidad_externa_v1.evento (
    evento_ref text PRIMARY KEY,
    operacion_ref text NOT NULL UNIQUE,
    actor_ref text NOT NULL,
    accion text NOT NULL CHECK (accion IN ('provisionar', 'registrar_sesion', 'cambiar_estado')),
    recurso_ref text NOT NULL,
    resultado text NOT NULL CHECK (resultado = 'confirmado'),
    registrado_en timestamptz(6) NOT NULL
);

CREATE FUNCTION vec_identidad_externa_v1.huella_material_alta_v1(
    p_cuenta_ref text, p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint,
    p_cuenta_id_hmac bytea, p_sujeto_id_hmac bytea
) RETURNS text LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog, pg_temp AS $f$
    SELECT pg_catalog.encode(public.digest(
        pg_catalog.convert_to(
            pg_catalog.length(p_cuenta_ref)::text || ':' || p_cuenta_ref ||
            pg_catalog.length(p_esquema_hmac)::text || ':' || p_esquema_hmac ||
            pg_catalog.length(p_dominio_hmac_ref)::text || ':' || p_dominio_hmac_ref ||
            pg_catalog.length(p_clave_hmac_id)::text || ':' || p_clave_hmac_id ||
            p_clave_hmac_version::text || ':' ||
            pg_catalog.encode(p_cuenta_id_hmac, 'hex') || ':' ||
            pg_catalog.encode(p_sujeto_id_hmac, 'hex'), 'UTF8'),
        'sha256'), 'hex')
$f$;

-- CAS de ausencia: el marcador de preimagen se computa para la cuenta exacta.
CREATE FUNCTION vec_identidad_externa_v1.huella_ausencia_v1(p_cuenta_ref text)
RETURNS text LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog, pg_temp AS $f$
    SELECT pg_catalog.encode(public.digest(
        pg_catalog.convert_to('vec.identidad.externa.ausente.v1:' ||
            pg_catalog.length(p_cuenta_ref)::text || ':' || p_cuenta_ref,
            'UTF8'), 'sha256'), 'hex')
$f$;

CREATE FUNCTION vec_identidad_externa_v1.confirmar_alta_v1(
    p_operacion_ref text, p_aprobacion_ref text, p_cuenta_ref text,
    p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint,
    p_cuenta_id_hmac bytea, p_sujeto_id_hmac bytea,
    p_huella_aprobada_sha256 text, p_huella_preimagen_sha256 text,
    p_revision_esperada bigint
) RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
DECLARE a record; existente record; ahora timestamptz(6); material text;
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE
       OR p_aprobacion_ref !~ '^apr_[A-Za-z0-9_-]{22,128}$'
       OR vec_identidad_sesiones_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
           p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,p_clave_hmac_version) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_cuenta_id_hmac) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_sujeto_id_hmac) IS NOT TRUE
       OR p_cuenta_id_hmac = p_sujeto_id_hmac OR p_revision_esperada IS DISTINCT FROM 0 THEN
        RETURN NULL;
    END IF;
    material := vec_identidad_externa_v1.huella_material_alta_v1(
        p_cuenta_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
        p_clave_hmac_version,p_cuenta_id_hmac,p_sujeto_id_hmac);
    IF material IS DISTINCT FROM p_huella_aprobada_sha256
       OR vec_identidad_externa_v1.huella_ausencia_v1(p_cuenta_ref)
          IS DISTINCT FROM p_huella_preimagen_sha256 THEN RETURN NULL; END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_externa_v1:cuenta:' || p_cuenta_ref, 0));
    SELECT * INTO a FROM vec_identidad_externa_v1.aprobacion_alta
     WHERE aprobacion_ref=p_aprobacion_ref FOR UPDATE;
    IF NOT FOUND OR a.huella_material_sha256 IS DISTINCT FROM material
       OR a.huella_preimagen_sha256 IS DISTINCT FROM p_huella_preimagen_sha256
       OR a.revision_esperada IS DISTINCT FROM p_revision_esperada THEN RETURN NULL; END IF;
    SELECT c.cuenta_ref,c.operacion_ref,c.aprobacion_ref,c.huella_material_sha256,
           x.esquema_hmac,x.dominio_hmac_ref,x.clave_hmac_id,x.clave_hmac_version,
           x.cuenta_id_hmac,x.sujeto_id_hmac
      INTO existente FROM vec_identidad_externa_v1.cuenta c
      JOIN vec_identidad_externa_v1.alias_cuenta x ON x.cuenta_ref=c.cuenta_ref
     WHERE c.cuenta_ref=p_cuenta_ref;
    IF FOUND THEN
        IF existente.operacion_ref=p_operacion_ref
           AND existente.aprobacion_ref=p_aprobacion_ref
           AND existente.huella_material_sha256=material
           AND existente.esquema_hmac=p_esquema_hmac
           AND existente.dominio_hmac_ref=p_dominio_hmac_ref
           AND existente.clave_hmac_id=p_clave_hmac_id
           AND existente.clave_hmac_version=p_clave_hmac_version
           AND existente.cuenta_id_hmac=p_cuenta_id_hmac
           AND existente.sujeto_id_hmac=p_sujeto_id_hmac THEN RETURN p_cuenta_ref; END IF;
        RETURN NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM vec_identidad_externa_v1.consumo_aprobacion
               WHERE aprobacion_ref=p_aprobacion_ref OR operacion_ref=p_operacion_ref) THEN
        RETURN NULL;
    END IF;
    ahora:=pg_catalog.clock_timestamp();
    INSERT INTO vec_identidad_externa_v1.cuenta VALUES
        (p_cuenta_ref,p_dominio_hmac_ref,material,p_aprobacion_ref,p_operacion_ref,ahora);
    INSERT INTO vec_identidad_externa_v1.alias_cuenta VALUES
        (vec_identidad_sesiones_v1.nueva_referencia('ali_'),p_cuenta_ref,
         p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,p_clave_hmac_version,
         p_cuenta_id_hmac,p_sujeto_id_hmac,ahora,p_operacion_ref);
    INSERT INTO vec_identidad_externa_v1.estado_cuenta VALUES
        (p_cuenta_ref,1,'activa',ahora,p_operacion_ref);
    INSERT INTO vec_identidad_externa_v1.estado_actual VALUES
        (p_cuenta_ref,1,ahora,p_operacion_ref);
    INSERT INTO vec_identidad_externa_v1.consumo_aprobacion VALUES
        (p_aprobacion_ref,p_cuenta_ref,p_operacion_ref,material,ahora);
    INSERT INTO vec_identidad_externa_v1.evento VALUES
        (p_operacion_ref,p_operacion_ref,
         session_user,'provisionar',p_cuenta_ref,'confirmado',ahora);
    RETURN p_cuenta_ref;
EXCEPTION WHEN unique_violation OR foreign_key_violation OR check_violation
          OR data_exception THEN RETURN NULL;
END $f$;

CREATE FUNCTION vec_identidad_externa_v1.registrar_sesion_v1(
    p_operacion_ref text, p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea, p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea, p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea, p_cuenta_privilegiada boolean,
    p_superficie text, p_metodo_observado text, p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz, p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text, p_politica_garantia_huella_sha256 text
) RETURNS TABLE (
    autenticacion_ref text, asercion_ref text, sesion_ref text,
    control_sesion_ref text, control_sesion_revision_texto text,
    control_sesion_estado text, control_sesion_huella_sha256 text,
    cuenta_ref text, cuenta_ordinaria_ref text,
    sesion_revalidada_en timestamptz, sesion_valida_hasta timestamptz
) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
DECLARE c record; ahora timestamptz(6); aut text; ase text; ses text;
        ctl text; huella text;
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
           p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,p_clave_hmac_version) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_asercion_id_hmac) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_sesion_id_hmac) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_sujeto_id_hmac) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_cuenta_id_hmac) IS NOT TRUE
       OR p_asercion_id_hmac IN (p_sesion_id_hmac,p_sujeto_id_hmac,p_cuenta_id_hmac)
       OR p_sesion_id_hmac IN (p_sujeto_id_hmac,p_cuenta_id_hmac)
       OR p_sujeto_id_hmac=p_cuenta_id_hmac
       OR p_cuenta_ordinaria_id_hmac IS NOT NULL OR p_cuenta_privilegiada IS DISTINCT FROM false
       OR p_superficie IS DISTINCT FROM 'externa_personal'
       OR p_metodo_observado NOT IN ('certificado','dnie','sso','clave','kerberos_ad')
       OR p_garantia_observada NOT IN ('sustancial','alto')
       OR p_autenticacion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autenticacion_huella_sha256=repeat('0',64)
       OR p_politica_garantia_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_politica_garantia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_politica_garantia_huella_sha256=repeat('0',64)
       OR p_autenticacion_verificada_en IS NULL OR p_sesion_emitida_en IS NULL
       OR p_asercion_expira_en IS NULL
       OR p_autenticacion_verificada_en > p_sesion_emitida_en
       OR p_asercion_expira_en <= p_sesion_emitida_en
       OR p_asercion_expira_en-p_sesion_emitida_en > interval '5 minutes' THEN
        RETURN;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_externa_v1:sesion:' || p_dominio_hmac_ref || ':' ||
        p_clave_hmac_id || ':' || p_clave_hmac_version::text || ':' ||
        pg_catalog.encode(p_sesion_id_hmac,'hex'),0));
    SELECT cuenta.cuenta_ref,actual.revision,estado.estado
      INTO c FROM vec_identidad_externa_v1.alias_cuenta alias
      JOIN vec_identidad_externa_v1.cuenta cuenta
        ON cuenta.cuenta_ref=alias.cuenta_ref
       AND cuenta.dominio_hmac_ref=alias.dominio_hmac_ref
      JOIN vec_identidad_externa_v1.estado_actual actual
        ON actual.cuenta_ref=cuenta.cuenta_ref
      JOIN vec_identidad_externa_v1.estado_cuenta estado
        ON estado.cuenta_ref=actual.cuenta_ref AND estado.revision=actual.revision
     WHERE alias.esquema_hmac=p_esquema_hmac
       AND alias.dominio_hmac_ref=p_dominio_hmac_ref
       AND alias.clave_hmac_id=p_clave_hmac_id
       AND alias.clave_hmac_version=p_clave_hmac_version
       AND alias.cuenta_id_hmac=p_cuenta_id_hmac
       AND alias.sujeto_id_hmac=p_sujeto_id_hmac
     FOR UPDATE OF actual;
    IF NOT FOUND OR c.estado <> 'activa' THEN RETURN; END IF;
    ahora:=pg_catalog.clock_timestamp();
    IF ahora<p_sesion_emitida_en OR ahora>=p_asercion_expira_en
       OR ahora>=p_autenticacion_verificada_en+interval '12 hours'
       OR EXISTS (SELECT 1 FROM vec_identidad_externa_v1.sesion s
                  JOIN vec_identidad_externa_v1.control_sesion x
                    ON x.sesion_ref=s.sesion_ref
                 WHERE s.esquema_hmac=p_esquema_hmac
                   AND s.dominio_hmac_ref=p_dominio_hmac_ref
                   AND s.clave_hmac_id=p_clave_hmac_id
                   AND s.clave_hmac_version=p_clave_hmac_version
                   AND s.sesion_id_hmac=p_sesion_id_hmac
                   AND x.sesion_valida_hasta>ahora) THEN RETURN; END IF;
    aut:=vec_identidad_sesiones_v1.nueva_referencia('aut_');
    ase:=vec_identidad_sesiones_v1.nueva_referencia('ase_');
    ses:=vec_identidad_sesiones_v1.nueva_referencia('ses_');
    ctl:=vec_identidad_sesiones_v1.nueva_referencia('cse_');
    huella:=vec_identidad_sesiones_v1.huella_control_sesion_v1(
        ctl,1,ses,'activa',ahora,p_asercion_expira_en,p_operacion_ref);
    INSERT INTO vec_identidad_externa_v1.sesion VALUES
        (ses,aut,ase,p_operacion_ref,c.cuenta_ref,c.revision,
         p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,p_clave_hmac_version,
         p_asercion_id_hmac,p_sesion_id_hmac,p_sujeto_id_hmac,p_cuenta_id_hmac,
         p_metodo_observado,p_garantia_observada,p_autenticacion_huella_sha256,
         p_autenticacion_verificada_en,p_sesion_emitida_en,
         p_politica_garantia_ref,p_politica_garantia_huella_sha256,ahora);
    INSERT INTO vec_identidad_externa_v1.control_sesion VALUES
        (ctl,ses,1,'activa',huella,ahora,p_asercion_expira_en);
    INSERT INTO vec_identidad_externa_v1.evento VALUES
        (p_operacion_ref,p_operacion_ref,
         c.cuenta_ref,'registrar_sesion',ses,'confirmado',ahora);
    RETURN QUERY SELECT aut,ase,ses,ctl,'1'::text,'activa'::text,huella,
                        c.cuenta_ref,c.cuenta_ref,ahora,p_asercion_expira_en;
EXCEPTION WHEN unique_violation OR foreign_key_violation OR check_violation
          OR data_exception THEN RETURN;
END $f$;

CREATE FUNCTION vec_identidad_externa_v1.reconciliar_registro_sesion_v1(
    p_operacion_ref text, p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea, p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea, p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea, p_cuenta_privilegiada boolean,
    p_superficie text, p_metodo_observado text, p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz, p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text, p_politica_garantia_huella_sha256 text
) RETURNS TABLE (
    autenticacion_ref text, asercion_ref text, sesion_ref text,
    control_sesion_ref text, control_sesion_revision_texto text,
    control_sesion_estado text, control_sesion_huella_sha256 text,
    cuenta_ref text, cuenta_ordinaria_ref text,
    sesion_revalidada_en timestamptz, sesion_valida_hasta timestamptz
) LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
    WITH bloqueada AS MATERIALIZED (
    SELECT s.autenticacion_ref,s.asercion_ref,s.sesion_ref,
           x.control_sesion_ref,x.revision,x.estado,x.huella_sha256,
           s.cuenta_ref,x.sesion_revalidada_en,x.sesion_valida_hasta,
           s.autenticacion_verificada_en
      FROM vec_identidad_externa_v1.sesion s
      JOIN vec_identidad_externa_v1.control_sesion x ON x.sesion_ref=s.sesion_ref
      JOIN vec_identidad_externa_v1.cuenta c ON c.cuenta_ref=s.cuenta_ref
      JOIN vec_identidad_externa_v1.estado_actual a ON a.cuenta_ref=c.cuenta_ref
      JOIN vec_identidad_externa_v1.estado_cuenta e
        ON e.cuenta_ref=a.cuenta_ref AND e.revision=a.revision
     WHERE s.operacion_ref=p_operacion_ref
       AND s.esquema_hmac=p_esquema_hmac
       AND s.dominio_hmac_ref=p_dominio_hmac_ref
       AND c.dominio_hmac_ref=p_dominio_hmac_ref
       AND a.revision=s.cuenta_revision AND e.estado='activa'
       AND s.clave_hmac_id=p_clave_hmac_id
       AND s.clave_hmac_version=p_clave_hmac_version
       AND s.asercion_id_hmac=p_asercion_id_hmac
       AND s.sesion_id_hmac=p_sesion_id_hmac
       AND s.sujeto_id_hmac=p_sujeto_id_hmac
       AND s.cuenta_id_hmac=p_cuenta_id_hmac
       AND p_cuenta_ordinaria_id_hmac IS NULL
       AND p_cuenta_privilegiada=false AND p_superficie='externa_personal'
       AND s.metodo_observado=p_metodo_observado
       AND s.garantia_observada=p_garantia_observada
       AND s.autenticacion_huella_sha256=p_autenticacion_huella_sha256
       AND s.autenticacion_verificada_en=p_autenticacion_verificada_en
       AND s.sesion_emitida_en=p_sesion_emitida_en
       AND x.sesion_valida_hasta=p_asercion_expira_en
       AND s.politica_garantia_ref=p_politica_garantia_ref
       AND s.politica_garantia_huella_sha256=p_politica_garantia_huella_sha256
     FOR UPDATE OF a
    )
    SELECT b.autenticacion_ref,b.asercion_ref,b.sesion_ref,
           b.control_sesion_ref,b.revision::text,b.estado,b.huella_sha256,
           b.cuenta_ref,b.cuenta_ref,b.sesion_revalidada_en,b.sesion_valida_hasta
      FROM bloqueada b
     WHERE pg_catalog.clock_timestamp()>=b.sesion_revalidada_en
       AND pg_catalog.clock_timestamp()<b.sesion_valida_hasta
       AND pg_catalog.clock_timestamp()<b.autenticacion_verificada_en+interval '12 hours'
$f$;

CREATE FUNCTION vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(
    p_autenticacion_ref text, p_autenticacion_huella_sha256 text,
    p_asercion_ref text, p_sesion_ref text, p_cuenta_ref text,
    p_cuenta_ordinaria_ref text, p_cuenta_privilegiada boolean,
    p_superficie text, p_metodo_observado text, p_garantia_observada text,
    p_politica_garantia_ref text, p_politica_garantia_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz, p_sesion_emitida_en timestamptz,
    p_control_sesion_ref text, p_control_sesion_revision_texto text,
    p_control_sesion_estado text, p_control_sesion_huella_sha256 text,
    p_sesion_revalidada_en timestamptz, p_sesion_valida_hasta timestamptz
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
DECLARE s record; ahora timestamptz(6);
BEGIN
    IF p_cuenta_privilegiada IS DISTINCT FROM false
       OR p_superficie IS DISTINCT FROM 'externa_personal'
       OR p_cuenta_ref IS DISTINCT FROM p_cuenta_ordinaria_ref
       OR p_control_sesion_revision_texto IS DISTINCT FROM '1'
       OR p_control_sesion_estado IS DISTINCT FROM 'activa' THEN RETURN false; END IF;
    SELECT ses.cuenta_ref,ses.cuenta_revision,ses.autenticacion_verificada_en,
           ctl.sesion_revalidada_en,ctl.sesion_valida_hasta,actual.revision,
           estado.estado
      INTO s FROM vec_identidad_externa_v1.sesion ses
      JOIN vec_identidad_externa_v1.control_sesion ctl ON ctl.sesion_ref=ses.sesion_ref
      JOIN vec_identidad_externa_v1.cuenta cuenta ON cuenta.cuenta_ref=ses.cuenta_ref
      JOIN vec_identidad_externa_v1.estado_actual actual ON actual.cuenta_ref=cuenta.cuenta_ref
      JOIN vec_identidad_externa_v1.estado_cuenta estado
        ON estado.cuenta_ref=actual.cuenta_ref AND estado.revision=actual.revision
     WHERE ses.autenticacion_ref=p_autenticacion_ref
       AND ses.autenticacion_huella_sha256=p_autenticacion_huella_sha256
       AND ses.asercion_ref=p_asercion_ref AND ses.sesion_ref=p_sesion_ref
       AND ses.cuenta_ref=p_cuenta_ref
       AND ses.metodo_observado=p_metodo_observado
       AND ses.garantia_observada=p_garantia_observada
       AND ses.politica_garantia_ref=p_politica_garantia_ref
       AND ses.politica_garantia_huella_sha256=p_politica_garantia_huella_sha256
       AND ses.autenticacion_verificada_en=p_autenticacion_verificada_en
       AND ses.sesion_emitida_en=p_sesion_emitida_en
       AND cuenta.dominio_hmac_ref=ses.dominio_hmac_ref
       AND ctl.control_sesion_ref=p_control_sesion_ref
       AND ctl.revision=1 AND ctl.estado='activa'
       AND ctl.huella_sha256=p_control_sesion_huella_sha256
       AND ctl.sesion_revalidada_en=p_sesion_revalidada_en
       AND ctl.sesion_valida_hasta=p_sesion_valida_hasta
     FOR UPDATE OF actual;
    IF NOT FOUND OR s.estado<>'activa' OR s.revision<>s.cuenta_revision THEN
        RETURN false;
    END IF;
    ahora:=pg_catalog.clock_timestamp();
    RETURN ahora>=s.sesion_revalidada_en
       AND ahora<s.sesion_valida_hasta
       AND ahora<s.autenticacion_verificada_en+interval '12 hours';
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;

CREATE FUNCTION vec_identidad_externa_v1.revalidar_autenticacion_actor_v1(
    p_autenticacion_ref text, p_sesion_ref text
) RETURNS TABLE (
    autenticacion_ref text, autenticacion_huella_sha256 text,
    asercion_ref text, sesion_ref text, control_sesion_ref text,
    control_sesion_revision text, control_sesion_huella_sha256 text,
    cuenta_ref text, cuenta_ordinaria_ref text, cuenta_privilegiada boolean,
    superficie text, metodo_observado text, garantia_observada text,
    politica_garantia_ref text, politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz, sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz, sesion_revalidada_en timestamptz
) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
DECLARE s record; ahora timestamptz(6);
BEGIN
    IF vec_identidad_sesiones_v1.referencia_valida(p_autenticacion_ref,'aut_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(p_sesion_ref,'ses_') IS NOT TRUE THEN
        RETURN;
    END IF;
    SELECT ses.*,ctl.control_sesion_ref,ctl.revision AS control_revision,
           ctl.estado AS control_estado,ctl.huella_sha256 AS control_huella,
           ctl.sesion_valida_hasta,ctl.sesion_revalidada_en,
           cuenta.dominio_hmac_ref AS dominio_cuenta,
           actual.revision AS revision_actual,estado.estado AS estado_actual
      INTO s FROM vec_identidad_externa_v1.sesion ses
      JOIN vec_identidad_externa_v1.control_sesion ctl ON ctl.sesion_ref=ses.sesion_ref
      JOIN vec_identidad_externa_v1.cuenta cuenta ON cuenta.cuenta_ref=ses.cuenta_ref
      JOIN vec_identidad_externa_v1.estado_actual actual ON actual.cuenta_ref=cuenta.cuenta_ref
      JOIN vec_identidad_externa_v1.estado_cuenta estado
        ON estado.cuenta_ref=actual.cuenta_ref AND estado.revision=actual.revision
     WHERE ses.autenticacion_ref=p_autenticacion_ref
       AND ses.sesion_ref=p_sesion_ref FOR UPDATE OF actual;
    IF NOT FOUND OR s.estado_actual<>'activa'
       OR s.revision_actual<>s.cuenta_revision
       OR s.dominio_cuenta<>s.dominio_hmac_ref
       OR s.control_estado<>'activa' OR s.control_revision<>1 THEN RETURN; END IF;
    ahora:=pg_catalog.clock_timestamp();
    IF ahora<s.sesion_revalidada_en OR ahora>=s.sesion_valida_hasta
       OR ahora>=s.autenticacion_verificada_en+interval '12 hours' THEN RETURN; END IF;
    RETURN QUERY SELECT s.autenticacion_ref,s.autenticacion_huella_sha256,
        s.asercion_ref,s.sesion_ref,s.control_sesion_ref,s.control_revision::text,
        s.control_huella,s.cuenta_ref,s.cuenta_ref,false,'externa_personal'::text,
        s.metodo_observado,s.garantia_observada,s.politica_garantia_ref,
        s.politica_garantia_huella_sha256,s.autenticacion_verificada_en,
        s.sesion_emitida_en,s.sesion_valida_hasta,s.sesion_revalidada_en;
EXCEPTION WHEN data_exception THEN RETURN;
END $f$;

CREATE FUNCTION vec_identidad_externa_v1.cambiar_estado_cuenta_v1(
    p_cuenta_ref text, p_revision_esperada bigint,
    p_estado_nuevo text, p_operacion_ref text
) RETURNS bigint LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $f$
DECLARE actual record; ahora timestamptz(6);
BEGIN
    IF p_cuenta_ref IS NULL OR p_revision_esperada IS NULL
       OR p_estado_nuevo NOT IN ('activa','inactiva')
       OR vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE
       OR p_revision_esperada=9223372036854775807 THEN RETURN NULL; END IF;
    SELECT puntero.revision,estado.estado INTO actual
      FROM vec_identidad_externa_v1.estado_actual puntero
      JOIN vec_identidad_externa_v1.estado_cuenta estado
        ON estado.cuenta_ref=puntero.cuenta_ref AND estado.revision=puntero.revision
     WHERE puntero.cuenta_ref=p_cuenta_ref FOR UPDATE OF puntero;
    IF NOT FOUND OR actual.revision<>p_revision_esperada
       OR actual.estado=p_estado_nuevo THEN RETURN NULL; END IF;
    ahora:=pg_catalog.clock_timestamp();
    INSERT INTO vec_identidad_externa_v1.estado_cuenta VALUES
        (p_cuenta_ref,p_revision_esperada+1,p_estado_nuevo,ahora,p_operacion_ref);
    UPDATE vec_identidad_externa_v1.estado_actual SET
        revision=p_revision_esperada+1,actualizada_en=ahora,operacion_ref=p_operacion_ref
     WHERE cuenta_ref=p_cuenta_ref AND revision=p_revision_esperada;
    IF NOT FOUND THEN RETURN NULL; END IF;
    INSERT INTO vec_identidad_externa_v1.evento VALUES
        (p_operacion_ref,p_operacion_ref,session_user,'cambiar_estado',
         p_cuenta_ref,'confirmado',ahora);
    RETURN p_revision_esperada+1;
EXCEPTION WHEN unique_violation OR foreign_key_violation OR check_violation
          OR data_exception THEN RETURN NULL;
END $f$;

CREATE FUNCTION vec_identidad_externa_v1.rechazar_mutacion_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $f$
BEGIN RAISE EXCEPTION 'historia externa inmutable' USING ERRCODE='55000'; END $f$;

DO $proteger$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['aprobacion_alta','cuenta','alias_cuenta','estado_cuenta',
                            'consumo_aprobacion','sesion','control_sesion','evento'] LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_externa_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_externa_v1.rechazar_mutacion_v1()',
            t,t);
    END LOOP;
    FOREACH t IN ARRAY ARRAY['aprobacion_alta','cuenta','alias_cuenta','estado_cuenta',
                            'estado_actual','consumo_aprobacion','sesion','control_sesion','evento'] LOOP
        EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_externa_v1.%I ENABLE ROW LEVEL SECURITY',t);
        EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_externa_v1.%I FORCE ROW LEVEL SECURITY',t);
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_exacto ON vec_identidad_externa_v1.%I FOR ALL TO vec_identidad_sesiones_v1_propietario USING (current_user=%L) WITH CHECK (current_user=%L)',
            t,'vec_identidad_sesiones_v1_propietario','vec_identidad_sesiones_v1_propietario');
    END LOOP;
END $proteger$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_identidad_externa_v1 FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_identidad_externa_v1 FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_externa_v1 TO vec_identidad_externa_v1_provisionador;
GRANT EXECUTE ON FUNCTION vec_identidad_externa_v1.confirmar_alta_v1(
    text,text,text,text,text,text,bigint,bytea,bytea,text,text,bigint)
    TO vec_identidad_externa_v1_provisionador;
GRANT EXECUTE ON FUNCTION vec_identidad_externa_v1.cambiar_estado_cuenta_v1(
    text,bigint,text,text) TO vec_identidad_externa_v1_provisionador;
GRANT USAGE ON SCHEMA vec_identidad_externa_v1 TO vec_identidad_externa_v1_registrador;
GRANT EXECUTE ON FUNCTION vec_identidad_externa_v1.registrar_sesion_v1(
    text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,
    text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)
    TO vec_identidad_externa_v1_registrador;
GRANT EXECUTE ON FUNCTION vec_identidad_externa_v1.reconciliar_registro_sesion_v1(
    text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,
    text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)
    TO vec_identidad_externa_v1_registrador;
GRANT USAGE ON SCHEMA vec_identidad_externa_v1 TO vec_identidad_externa_v1_revalidador;
GRANT EXECUTE ON FUNCTION vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1(
    text,text,text,text,text,text,boolean,text,text,text,text,text,
    timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)
    TO vec_identidad_externa_v1_revalidador;
GRANT EXECUTE ON FUNCTION vec_identidad_externa_v1.revalidar_autenticacion_actor_v1(text,text)
    TO vec_identidad_externa_v1_revalidador;
COMMIT;
