-- IS18. Primer epoch: el DBA instala una unica configuracion privada.
-- La ausencia, retirada o divergencia de esa fila deniega globalmente.
-- Dependencia estricta: AD221 antes de IS18. No adopta sesiones IS2 previas.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_identidad_sesiones_v1:migracion:000018', 0));
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
        WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_presentador') IS NOT NULL
       OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_revocador_presentacion') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.registrar_sesion_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,text,bytea,text,text,timestamptz,text,text,text,timestamptz)') IS NULL
       OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1') IS NOT NULL THEN
        RAISE EXCEPTION 'IS18: preimagen o AD221 incompatibles' USING ERRCODE='55000';
    END IF;
END $pre$;
CREATE ROLE vec_identidad_sesiones_v1_presentador NOLOGIN NOINHERIT
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_identidad_sesiones_v1_revocador_presentacion NOLOGIN NOINHERIT
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
-- Ambos grupos nuevos necesitan su propia concesion explicita. El LOGIN
-- concreto se acredita por separado; CONNECT heredado de PUBLIC no satisface
-- el manifiesto de capacidades del adaptador.
DO $connect$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_identidad_sesiones_v1_presentador, vec_identidad_sesiones_v1_revocador_presentacion',
        pg_catalog.current_database()
    );
END $connect$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

-- Usar despues de bloquear todos los controles implicados y antes de la
-- raiz. IS6 y registrar_sesion_v1 volveran a tomar estos mismos cerrojos.
CREATE FUNCTION vec_identidad_sesiones_v1.bloquear_cuentas_presentacion_v1(
    p_sesiones text[]
) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
AS $funcion$
DECLARE cuentas text[];cuenta text;sesiones_encontradas integer;
BEGIN
    IF p_sesiones IS NULL OR pg_catalog.cardinality(p_sesiones) NOT BETWEEN 1 AND 2
       OR EXISTS(SELECT 1 FROM pg_catalog.unnest(p_sesiones) AS s(sesion_ref)
           WHERE s.sesion_ref !~ '^ses_[A-Za-z0-9_-]{22,128}$')
       OR (SELECT pg_catalog.count(DISTINCT s.sesion_ref)
             FROM pg_catalog.unnest(p_sesiones) AS s(sesion_ref))
           <> pg_catalog.cardinality(p_sesiones) THEN RETURN false; END IF;
    SELECT pg_catalog.count(*) INTO sesiones_encontradas
      FROM vec_autorizacion.sesion_autenticacion_v1 b
     WHERE b.sesion_ref=ANY(p_sesiones);
    IF sesiones_encontradas<>pg_catalog.cardinality(p_sesiones) THEN RETURN false; END IF;
    SELECT pg_catalog.array_agg(x.cuenta_ref ORDER BY x.cuenta_ref COLLATE "C")
      INTO cuentas FROM (
        SELECT b.cuenta_ref FROM vec_autorizacion.sesion_autenticacion_v1 b
         WHERE b.sesion_ref=ANY(p_sesiones)
        UNION
        SELECT b.cuenta_ordinaria_ref FROM vec_autorizacion.sesion_autenticacion_v1 b
         WHERE b.sesion_ref=ANY(p_sesiones)
      ) x;
    IF cuentas IS NULL THEN RETURN false; END IF;
    FOREACH cuenta IN ARRAY cuentas LOOP
        PERFORM 1 FROM vec_identidad_sesiones_v1.estado_cuenta_actual a
         WHERE a.cuenta_ref=cuenta FOR UPDATE;
        IF NOT FOUND THEN RETURN false; END IF;
    END LOOP;
    RETURN true;
END $funcion$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.bloquear_cuentas_presentacion_v1(text[])
    FROM PUBLIC;

CREATE TABLE vec_identidad_sesiones_v1.gobierno_presentacion_certificado_v1 (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    esquema_hmac text NOT NULL CHECK (esquema_hmac = 'vec.identidad.hmac-sha256.v1'),
    dominio_hmac_ref text NOT NULL CHECK (dominio_hmac_ref ~ '^idh_[A-Za-z0-9_-]{22,128}$'),
    clave_hmac_id text NOT NULL CHECK (octet_length(clave_hmac_id) BETWEEN 1 AND 128
        AND clave_hmac_id ~ '^[!-~]+$'),
    clave_hmac_version bigint NOT NULL CHECK (clave_hmac_version > 0),
    politica_ref text NOT NULL CHECK (politica_ref ~ '^pga_[A-Za-z0-9_-]{22,128}$'),
    politica_huella_sha256 text NOT NULL CHECK (politica_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND politica_huella_sha256 <> pg_catalog.repeat('0',64)),
    registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
    retirar_en timestamptz(6) NOT NULL CHECK (pg_catalog.isfinite(retirar_en)),
    activa boolean NOT NULL DEFAULT true,
    CHECK (retirar_en > registrada_en)
);
CREATE FUNCTION vec_identidad_sesiones_v1.bloquear_cambio_gobierno_presentacion_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $funcion$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.activa AND NOT NEW.activa
       AND ROW(NEW.esquema_hmac,NEW.dominio_hmac_ref,NEW.clave_hmac_id,
               NEW.clave_hmac_version,NEW.politica_ref,
               NEW.politica_huella_sha256,NEW.registrada_en,NEW.retirar_en)
           IS NOT DISTINCT FROM
           ROW(OLD.esquema_hmac,OLD.dominio_hmac_ref,OLD.clave_hmac_id,
               OLD.clave_hmac_version,OLD.politica_ref,
               OLD.politica_huella_sha256,OLD.registrada_en,OLD.retirar_en) THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'IS18: epoch irreversible; transicion requiere migracion aprobada'
        USING ERRCODE='55000';
END $funcion$;
CREATE TRIGGER gobierno_solo_retirada BEFORE UPDATE OR DELETE
    ON vec_identidad_sesiones_v1.gobierno_presentacion_certificado_v1
    FOR EACH ROW EXECUTE FUNCTION
        vec_identidad_sesiones_v1.bloquear_cambio_gobierno_presentacion_v1();
CREATE TRIGGER gobierno_no_truncar BEFORE TRUNCATE
    ON vec_identidad_sesiones_v1.gobierno_presentacion_certificado_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

CREATE TABLE vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1 (
    presentacion_ref text PRIMARY KEY CHECK (presentacion_ref ~ '^prs_[A-Za-z0-9_-]{22,128}$'),
    apertura_operacion_ref text NOT NULL UNIQUE CHECK (
        apertura_operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    apertura_preimagen_sha256 text NOT NULL CHECK (apertura_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
    esquema_hmac text NOT NULL,
    dominio_hmac_ref text NOT NULL,
    clave_hmac_id text NOT NULL,
    clave_hmac_version bigint NOT NULL,
    certificado_der_hmac bytea NOT NULL CHECK (vec_identidad_sesiones_v1.huella_hmac_valida(certificado_der_hmac)),
    ca_hmac bytea NOT NULL CHECK (vec_identidad_sesiones_v1.huella_hmac_valida(ca_hmac)),
    sujeto_id_hmac bytea NOT NULL CHECK (vec_identidad_sesiones_v1.huella_hmac_valida(sujeto_id_hmac)),
    cuenta_id_hmac bytea NOT NULL CHECK (vec_identidad_sesiones_v1.huella_hmac_valida(cuenta_id_hmac)),
    acr_original text NOT NULL CHECK (
        acr_original='urn:vec:acr:certificado-desarrollo-protegido'),
    autenticacion_ref text NOT NULL UNIQUE CHECK (autenticacion_ref ~ '^aut_[A-Za-z0-9_-]{22,128}$'),
    sesion_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.sesion_autenticacion_v1(sesion_ref),
    sesion_generacion bigint NOT NULL DEFAULT 1 CHECK (sesion_generacion > 0),
    cuenta_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
    superficie text NOT NULL CHECK (superficie IN
        ('externa_personal','interna_corporativa','administracion_privilegiada')),
    creada_en timestamptz(6) NOT NULL,
    tumba_en timestamptz(6),
    tumba_operacion_ref text UNIQUE,
    tumba_recibo_sha256 text,
    tumba_preimagen_sha256 text,
    CHECK ((tumba_en IS NULL AND tumba_operacion_ref IS NULL
        AND tumba_recibo_sha256 IS NULL AND tumba_preimagen_sha256 IS NULL)
       OR (tumba_en IS NOT NULL AND tumba_operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'
           AND tumba_recibo_sha256 ~ '^[0-9a-f]{64}$'
           AND tumba_preimagen_sha256 ~ '^[0-9a-f]{64}$')),
    CHECK (certificado_der_hmac <> ca_hmac
        AND certificado_der_hmac <> sujeto_id_hmac
        AND certificado_der_hmac <> cuenta_id_hmac
        AND ca_hmac <> sujeto_id_hmac AND ca_hmac <> cuenta_id_hmac),
    UNIQUE (certificado_der_hmac)
);
CREATE INDEX vinculo_presentacion_certificado_cuenta_idx
    ON vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1(cuenta_ref);
CREATE TABLE vec_identidad_sesiones_v1.historia_sesion_presentacion_v1 (
    presentacion_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1(presentacion_ref),
    sesion_generacion bigint NOT NULL CHECK (sesion_generacion > 0),
    operacion_ref text NOT NULL UNIQUE CHECK (operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    preimagen_sha256 text NOT NULL CHECK (preimagen_sha256 ~ '^[0-9a-f]{64}$'),
    autenticacion_ref text NOT NULL UNIQUE CHECK (autenticacion_ref ~ '^aut_[A-Za-z0-9_-]{22,128}$'),
    sesion_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.sesion_autenticacion_v1(sesion_ref),
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (presentacion_ref,sesion_generacion)
);
CREATE TRIGGER historia_sesion_presentacion_inmutable BEFORE UPDATE OR DELETE
    ON vec_identidad_sesiones_v1.historia_sesion_presentacion_v1
    FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE TRIGGER historia_sesion_presentacion_no_truncar BEFORE TRUNCATE
    ON vec_identidad_sesiones_v1.historia_sesion_presentacion_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE FUNCTION vec_identidad_sesiones_v1.solo_tumba_presentacion_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $funcion$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.tumba_en IS NULL AND NEW.tumba_en IS NOT NULL
       AND pg_catalog.isfinite(NEW.tumba_en)
       AND NEW.tumba_operacion_ref IS NOT NULL
       AND NEW.tumba_recibo_sha256 IS NOT NULL
       AND NEW.tumba_preimagen_sha256 IS NOT NULL
       AND (to_jsonb(NEW) - 'tumba_en' - 'tumba_operacion_ref' - 'tumba_recibo_sha256' - 'tumba_preimagen_sha256')
           = (to_jsonb(OLD) - 'tumba_en' - 'tumba_operacion_ref' - 'tumba_recibo_sha256' - 'tumba_preimagen_sha256') THEN
        RETURN NEW;
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.tumba_en IS NULL AND NEW.tumba_en IS NULL
       AND NEW.sesion_generacion = OLD.sesion_generacion + 1
       AND NEW.autenticacion_ref IS DISTINCT FROM OLD.autenticacion_ref
       AND NEW.sesion_ref IS DISTINCT FROM OLD.sesion_ref
       AND (to_jsonb(NEW) - 'sesion_generacion' - 'autenticacion_ref' - 'sesion_ref')
           = (to_jsonb(OLD) - 'sesion_generacion' - 'autenticacion_ref' - 'sesion_ref')
       AND EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.historia_sesion_presentacion_v1 h
           WHERE h.presentacion_ref=NEW.presentacion_ref
             AND h.sesion_generacion=NEW.sesion_generacion
             AND h.autenticacion_ref=NEW.autenticacion_ref
             AND h.sesion_ref=NEW.sesion_ref) THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'IS18: vinculo de certificado inmutable' USING ERRCODE='55000';
END $funcion$;
CREATE TRIGGER vinculo_solo_tumba BEFORE UPDATE OR DELETE
    ON vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
    FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.solo_tumba_presentacion_v1();
CREATE TRIGGER vinculo_no_truncar BEFORE TRUNCATE
    ON vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

CREATE TABLE vec_identidad_sesiones_v1.consumo_presentacion_certificado_v1 (
    operacion_ref text PRIMARY KEY CHECK (operacion_ref ~ '^opr_[A-Za-z0-9_-]{22,128}$'),
    presentacion_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1(presentacion_ref),
    generacion bigint NOT NULL CHECK (generacion > 0),
    sesion_generacion bigint NOT NULL CHECK (sesion_generacion > 0),
    nonce_hmac bytea NOT NULL CHECK (vec_identidad_sesiones_v1.huella_hmac_valida(nonce_hmac)),
    preimagen_sha256 text NOT NULL CHECK (preimagen_sha256 ~ '^[0-9a-f]{64}$'),
    recibo_sha256 text NOT NULL CHECK (recibo_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    canal_sha256 text NOT NULL CHECK (canal_sha256 ~ '^[0-9a-f]{64}$'),
    asercion_actual_sha256 text NOT NULL CHECK (asercion_actual_sha256 ~ '^[0-9a-f]{64}$'),
    UNIQUE (presentacion_ref, generacion),
    UNIQUE (presentacion_ref, nonce_hmac),
    FOREIGN KEY (presentacion_ref,sesion_generacion)
       REFERENCES vec_identidad_sesiones_v1.historia_sesion_presentacion_v1(presentacion_ref,sesion_generacion),
    CHECK (valida_hasta > registrada_en)
);
CREATE TRIGGER consumo_presentacion_inmutable BEFORE UPDATE OR DELETE
    ON vec_identidad_sesiones_v1.consumo_presentacion_certificado_v1
    FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE TRIGGER consumo_presentacion_no_truncar BEFORE TRUNCATE
    ON vec_identidad_sesiones_v1.consumo_presentacion_certificado_v1
    FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

DO $cierre$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY['gobierno_presentacion_certificado_v1',
        'vinculo_presentacion_certificado_v1','historia_sesion_presentacion_v1',
        'consumo_presentacion_certificado_v1'] LOOP
        EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_sesiones_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
        EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_sesiones_v1.%I FORCE ROW LEVEL SECURITY',tabla);
        EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.%I FOR ALL TO vec_identidad_sesiones_v1_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',tabla,
            'vec_identidad_sesiones_v1_propietario','vec_identidad_sesiones_v1_propietario');
        EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_identidad_sesiones_v1.%I FROM PUBLIC',tabla);
        EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_identidad_sesiones_v1.%I FROM PUBLIC',tabla);
    END LOOP;
END $cierre$;

-- Enganche a la revocacion IS2: el UPDATE del puntero ya retiene su lock;
-- este trigger toma despues la raiz. La tumba confirma en su misma TX.
CREATE FUNCTION vec_identidad_sesiones_v1.tumbar_presentacion_por_control_v1()
RETURNS trigger LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
AS $funcion$
DECLARE estado text; v record; politica_original record;
    instante timestamptz(6); op text; recibo text;
BEGIN
    SELECT c.estado INTO estado FROM vec_autorizacion.control_sesion_v1 c
     WHERE c.sesion_ref=NEW.sesion_ref
       AND c.control_sesion_ref=NEW.control_sesion_ref
       AND c.revision=NEW.revision;
    IF estado IS DISTINCT FROM 'revocada' THEN RETURN NEW; END IF;
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
     WHERE sesion_ref=NEW.sesion_ref FOR UPDATE;
    IF FOUND AND v.tumba_en IS NULL THEN
        IF vec_identidad_sesiones_v1.referencia_valida(NEW.acto_ref,'opr_') IS NOT TRUE THEN
            RAISE EXCEPTION 'IS18: origen de revocacion IS2 invalido' USING ERRCODE='42501';
        END IF;
        instante:=pg_catalog.clock_timestamp();
        op:=NEW.acto_ref;
        recibo:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            pg_catalog.jsonb_build_array('vec.presentacion.tumba.control.v1',
                v.presentacion_ref,NEW.sesion_ref,NEW.control_sesion_ref,
                NEW.revision,NEW.acto_ref,op,instante)::text,'UTF8')),'hex');
        UPDATE vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1 x
           SET tumba_en=instante,tumba_operacion_ref=op,
               tumba_recibo_sha256=recibo,
               tumba_preimagen_sha256=pg_catalog.encode(pg_catalog.sha256(
                   pg_catalog.convert_to(pg_catalog.jsonb_build_array(
                       NEW.sesion_ref,NEW.control_sesion_ref,NEW.revision,
                       NEW.acto_ref)::text,'UTF8')),'hex')
         WHERE x.presentacion_ref=v.presentacion_ref AND x.tumba_en IS NULL;
        SELECT b.politica_garantia_ref,b.politica_garantia_huella_sha256
          INTO politica_original FROM vec_autorizacion.sesion_autenticacion_v1 b
         WHERE b.sesion_ref=v.sesion_ref
           AND b.autenticacion_ref=v.autenticacion_ref;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'IS18: origen de politica IS2 ausente' USING ERRCODE='42501';
        END IF;
        -- IS2 no transmite actor humano. El asiento comun dice exactamente
        -- eso: LOGIN técnico, cuenta/sujeto objetivo, control y acto de origen.
        PERFORM vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(
            op,v.presentacion_ref,v.autenticacion_ref,v.sesion_ref,
            v.cuenta_ref,NULL,NULL,NULL,v.superficie,
            'revocacion_control_is2_sin_actor_humano_nominal',recibo,
            v.acr_original,NULL,politica_original.politica_garantia_ref,
            politica_original.politica_garantia_huella_sha256,NULL,NULL,
            v.sujeto_id_hmac,NULL,NULL,NULL,
            NEW.control_sesion_ref,NEW.revision::text,NEW.acto_ref,instante);
    END IF;
    RETURN NEW;
END $funcion$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.tumbar_presentacion_por_control_v1()
    FROM PUBLIC;
RESET ROLE;
CREATE TRIGGER control_sesion_actual_tumba_presentacion_v1
    AFTER UPDATE ON vec_autorizacion.control_sesion_actual_v1
    FOR EACH ROW EXECUTE FUNCTION
        vec_identidad_sesiones_v1.tumbar_presentacion_por_control_v1();
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

-- Apertura y reanudacion usan la misma operacion atomica de consumo. La
-- preimagen del consumo incluye TODOS los argumentos de esta fachada.
CREATE FUNCTION vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(
    p_operacion_ref text, p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint, p_superficie text,
    p_certificado_der_hmac bytea, p_ca_hmac bytea,
    p_sujeto_id_hmac bytea, p_cuenta_id_hmac bytea, p_nonce_hmac bytea,
    p_canal_sha256 text, p_asercion_actual_sha256 text,
    p_asercion_actual_emitida_en timestamptz,
    p_asercion_actual_expira_en timestamptz,
    p_certificado_valido_hasta timestamptz, p_ca_valida_hasta timestamptz,
    p_crl_siguiente_actualizacion timestamptz,
    p_revocacion_verificada_en timestamptz,
    p_politica_presentacion_ref text,
    p_politica_presentacion_huella_sha256 text,
    p_metodo_actual text, p_garantia_actual text,
    p_acr_actual text
)
RETURNS TABLE (
    autenticacion_ref text, autenticacion_huella_sha256 text,
    asercion_ref text, sesion_ref text, control_sesion_ref text,
    control_sesion_revision text, control_sesion_huella_sha256 text,
    cuenta_ref text, cuenta_ordinaria_ref text, cuenta_privilegiada boolean,
    superficie text, metodo_observado text, garantia_observada text,
    politica_garantia_ref text, politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz, sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz, sesion_revalidada_en timestamptz,
    presentacion_ref text, presentacion_generacion bigint,
    presentacion_recibo_sha256 text, presentacion_registrada_en timestamptz,
    presentacion_valida_hasta timestamptz, canal_sha256 text,
    asercion_actual_sha256 text, operacion_ref text,
    sesion_generacion bigint, modo_inicio text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
    g record; v record; s record; previo record; control record;
    preimagen text; recibo text; instante timestamptz(6); limite timestamptz(6);
    nueva_generacion bigint;
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.pg_is_in_recovery()
       OR vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
            p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,p_clave_hmac_version) IS NOT TRUE
       OR p_superficie NOT IN
            ('externa_personal','interna_corporativa','administracion_privilegiada')
       OR EXISTS (SELECT 1 FROM pg_catalog.unnest(ARRAY[
            p_certificado_der_hmac,p_ca_hmac,p_sujeto_id_hmac,
            p_cuenta_id_hmac,p_nonce_hmac]) h
            WHERE vec_identidad_sesiones_v1.huella_hmac_valida(h) IS NOT TRUE)
       OR p_certificado_der_hmac IN (p_ca_hmac,p_sujeto_id_hmac,p_cuenta_id_hmac,p_nonce_hmac)
       OR p_ca_hmac IN (p_sujeto_id_hmac,p_cuenta_id_hmac,p_nonce_hmac)
       OR p_sujeto_id_hmac IN (p_cuenta_id_hmac,p_nonce_hmac)
       OR p_cuenta_id_hmac=p_nonce_hmac
       OR p_canal_sha256 !~ '^[0-9a-f]{64}$' OR p_asercion_actual_sha256 !~ '^[0-9a-f]{64}$'
       OR p_canal_sha256=pg_catalog.repeat('0',64)
       OR p_asercion_actual_sha256=pg_catalog.repeat('0',64)
       OR p_politica_presentacion_ref !~ '^pga_[A-Za-z0-9_-]{22,128}$'
       OR p_politica_presentacion_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_politica_presentacion_huella_sha256=pg_catalog.repeat('0',64)
       OR p_metodo_actual NOT IN ('certificado','dnie')
       OR p_garantia_actual NOT IN ('sustancial','alto')
       OR p_acr_actual IS DISTINCT FROM
            'urn:vec:acr:certificado-desarrollo-protegido'
       OR EXISTS (SELECT 1 FROM pg_catalog.unnest(ARRAY[
            p_asercion_actual_emitida_en,p_asercion_actual_expira_en,
            p_certificado_valido_hasta,p_ca_valida_hasta,
            p_crl_siguiente_actualizacion,p_revocacion_verificada_en]) t
            WHERE t IS NULL OR NOT pg_catalog.isfinite(t)) THEN
        RETURN;
    END IF;
    -- El gobierno se bloquea en cada consumo. Una retirada que confirme
    -- primero impide el acceso; el consumo que bloquee primero puede concluir.
    SELECT * INTO g FROM vec_identidad_sesiones_v1.gobierno_presentacion_certificado_v1
        WHERE singleton FOR SHARE;
    instante := pg_catalog.clock_timestamp();
    IF NOT FOUND OR NOT g.activa OR instante >= g.retirar_en
       OR p_esquema_hmac IS DISTINCT FROM g.esquema_hmac
       OR p_dominio_hmac_ref IS DISTINCT FROM g.dominio_hmac_ref
       OR p_clave_hmac_id IS DISTINCT FROM g.clave_hmac_id
       OR p_clave_hmac_version IS DISTINCT FROM g.clave_hmac_version
       OR p_politica_presentacion_ref IS DISTINCT FROM g.politica_ref
       OR p_politica_presentacion_huella_sha256 IS DISTINCT FROM g.politica_huella_sha256
       OR p_asercion_actual_emitida_en > instante
       OR p_revocacion_verificada_en > instante
       OR p_revocacion_verificada_en < p_asercion_actual_emitida_en
       OR p_asercion_actual_expira_en <= instante
       OR p_certificado_valido_hasta <= instante
       OR p_ca_valida_hasta <= instante
       OR p_crl_siguiente_actualizacion <= instante THEN
        RETURN;
    END IF;
    -- Lookup sin bloqueo; revocar_sesion_v1 bloquea primero el puntero y su
    -- trigger despues la raiz. Todas las operaciones siguen ese mismo orden.
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
      WHERE certificado_der_hmac=p_certificado_der_hmac;
    IF NOT FOUND OR v.tumba_en IS NOT NULL
       OR v.esquema_hmac IS DISTINCT FROM p_esquema_hmac
       OR v.dominio_hmac_ref IS DISTINCT FROM p_dominio_hmac_ref
       OR v.clave_hmac_id IS DISTINCT FROM p_clave_hmac_id
       OR v.clave_hmac_version IS DISTINCT FROM p_clave_hmac_version
       OR v.superficie IS DISTINCT FROM p_superficie
       OR v.ca_hmac IS DISTINCT FROM p_ca_hmac
       OR v.sujeto_id_hmac IS DISTINCT FROM p_sujeto_id_hmac
       OR v.cuenta_id_hmac IS DISTINCT FROM p_cuenta_id_hmac
       OR v.acr_original IS DISTINCT FROM p_acr_actual
       OR NOT EXISTS (
           SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion c
            WHERE c.sesion_ref=v.sesion_ref
              AND c.autenticacion_ref=v.autenticacion_ref
              AND c.cuenta_ref=v.cuenta_ref
              AND c.esquema_hmac=v.esquema_hmac
              AND c.dominio_hmac_ref=v.dominio_hmac_ref
              AND c.clave_hmac_id=v.clave_hmac_id
              AND c.clave_hmac_version=v.clave_hmac_version
              AND c.sujeto_id_hmac=p_sujeto_id_hmac
              AND c.cuenta_id_hmac=p_cuenta_id_hmac) THEN
        RETURN;
    END IF;
    SELECT actual.control_sesion_ref,actual.revision,ctl.estado
      INTO control FROM vec_autorizacion.control_sesion_actual_v1 actual
      JOIN vec_autorizacion.control_sesion_v1 ctl
        ON ctl.sesion_ref=actual.sesion_ref
       AND ctl.control_sesion_ref=actual.control_sesion_ref
       AND ctl.revision=actual.revision
     WHERE actual.sesion_ref=v.sesion_ref FOR UPDATE OF actual;
    IF NOT FOUND THEN RETURN; END IF;
    IF vec_identidad_sesiones_v1.bloquear_cuentas_presentacion_v1(
        ARRAY[v.sesion_ref]) IS DISTINCT FROM true THEN RETURN; END IF;
    -- Releer la raiz despues de control y cuentas. Un cambio de generacion
    -- durante la espera obliga a reintentar desde INICIO.
    PERFORM 1 FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1 x
      WHERE x.presentacion_ref=v.presentacion_ref FOR UPDATE;
    IF NOT FOUND THEN RETURN; END IF;
    IF NOT EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1 x
        WHERE x.presentacion_ref=v.presentacion_ref
          AND x.sesion_generacion=v.sesion_generacion
          AND x.sesion_ref=v.sesion_ref
          AND x.autenticacion_ref=v.autenticacion_ref
          AND x.tumba_en IS NULL) THEN RETURN; END IF;
    IF control.estado IS DISTINCT FROM 'activa' OR
       NOT EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion c
          WHERE c.sesion_ref=v.sesion_ref
            AND c.control_sesion_ref=control.control_sesion_ref
            AND c.control_sesion_revision=control.revision) THEN
        RETURN;
    END IF;
    SELECT * INTO s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
        v.autenticacion_ref,v.sesion_ref);
    IF NOT FOUND OR s.cuenta_ref IS DISTINCT FROM v.cuenta_ref
       OR s.superficie IS DISTINCT FROM v.superficie
       OR s.superficie IS DISTINCT FROM p_superficie
       OR s.metodo_observado IS DISTINCT FROM p_metodo_actual
       OR s.garantia_observada IS DISTINCT FROM p_garantia_actual
       OR s.metodo_observado IS DISTINCT FROM 'certificado'
       OR s.garantia_observada IS DISTINCT FROM 'sustancial'
       OR s.politica_garantia_ref IS DISTINCT FROM p_politica_presentacion_ref
       OR s.politica_garantia_huella_sha256 IS DISTINCT FROM
           p_politica_presentacion_huella_sha256 THEN
        RETURN;
    END IF;
    IF p_metodo_actual NOT IN ('certificado','dnie')
       OR (s.superficie='administracion_privilegiada' AND p_garantia_actual<>'alto')
       OR p_asercion_actual_emitida_en < s.autenticacion_verificada_en
       OR p_asercion_actual_expira_en <= p_asercion_actual_emitida_en THEN
        RETURN;
    END IF;
    limite := LEAST(s.sesion_valida_hasta,
        p_asercion_actual_expira_en,p_certificado_valido_hasta,
        p_ca_valida_hasta,p_crl_siguiente_actualizacion,g.retirar_en,
        instante + interval '5 minutes');
    IF limite <= instante THEN RETURN; END IF;
    preimagen:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.jsonb_build_array(p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,
            p_clave_hmac_id,p_clave_hmac_version,p_superficie,p_certificado_der_hmac,p_ca_hmac,
            p_sujeto_id_hmac,p_cuenta_id_hmac,p_nonce_hmac,p_canal_sha256,
            p_asercion_actual_sha256,p_asercion_actual_emitida_en,
            p_asercion_actual_expira_en,p_certificado_valido_hasta,p_ca_valida_hasta,
            p_crl_siguiente_actualizacion,p_revocacion_verificada_en,
            p_politica_presentacion_ref,p_politica_presentacion_huella_sha256,
            p_metodo_actual,p_garantia_actual,p_acr_actual)::text,'UTF8')),'hex');
    SELECT * INTO previo FROM vec_identidad_sesiones_v1.consumo_presentacion_certificado_v1 c
        WHERE c.operacion_ref=p_operacion_ref;
    IF FOUND THEN
        IF previo.presentacion_ref IS DISTINCT FROM v.presentacion_ref
           OR previo.sesion_generacion IS DISTINCT FROM v.sesion_generacion
           OR previo.preimagen_sha256 IS DISTINCT FROM preimagen
           OR previo.nonce_hmac IS DISTINCT FROM p_nonce_hmac
           OR instante >= previo.valida_hasta THEN RETURN; END IF;
        RETURN QUERY SELECT s.autenticacion_ref,s.autenticacion_huella_sha256,
            s.asercion_ref,s.sesion_ref,s.control_sesion_ref,s.control_sesion_revision,
            s.control_sesion_huella_sha256,s.cuenta_ref,s.cuenta_ordinaria_ref,
            s.cuenta_privilegiada,s.superficie,s.metodo_observado,s.garantia_observada,
            s.politica_garantia_ref,s.politica_garantia_huella_sha256,
            s.autenticacion_verificada_en,s.sesion_emitida_en,s.sesion_valida_hasta,
            s.sesion_revalidada_en,v.presentacion_ref,previo.generacion,
            previo.recibo_sha256,previo.registrada_en,previo.valida_hasta,
            previo.canal_sha256,previo.asercion_actual_sha256,
            p_operacion_ref,v.sesion_generacion,'reanudada'::text;
        RETURN;
    END IF;
    IF EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.consumo_presentacion_certificado_v1 c
        WHERE c.presentacion_ref=v.presentacion_ref AND c.nonce_hmac=p_nonce_hmac) THEN RETURN; END IF;
    SELECT COALESCE(MAX(c.generacion),0)+1 INTO nueva_generacion
      FROM vec_identidad_sesiones_v1.consumo_presentacion_certificado_v1 c
     WHERE c.presentacion_ref=v.presentacion_ref;
    recibo:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.jsonb_build_array('vec.presentacion.certificado.v1',v.presentacion_ref,
            nueva_generacion,p_operacion_ref,preimagen,s.autenticacion_ref,s.sesion_ref,
            s.cuenta_ref,p_canal_sha256,p_asercion_actual_sha256,instante,limite)::text,
            'UTF8')),'hex');
    INSERT INTO vec_identidad_sesiones_v1.consumo_presentacion_certificado_v1 (
        operacion_ref,presentacion_ref,generacion,sesion_generacion,nonce_hmac,preimagen_sha256,
        recibo_sha256,registrada_en,valida_hasta,canal_sha256,asercion_actual_sha256)
    VALUES (p_operacion_ref,v.presentacion_ref,nueva_generacion,v.sesion_generacion,p_nonce_hmac,
        preimagen,recibo,instante,limite,p_canal_sha256,p_asercion_actual_sha256);
    PERFORM vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(
        p_operacion_ref,v.presentacion_ref,s.autenticacion_ref,s.sesion_ref,
        s.cuenta_ref,s.cuenta_ref,s.autenticacion_ref,s.sesion_ref,s.superficie,
        CASE WHEN p_operacion_ref=v.apertura_operacion_ref THEN 'apertura'
             WHEN EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.historia_sesion_presentacion_v1 h
                 WHERE h.operacion_ref=p_operacion_ref AND h.sesion_generacion>1)
             THEN 'renovacion' ELSE 'reanudacion' END,recibo,
        v.acr_original,p_acr_actual,s.politica_garantia_ref,
        s.politica_garantia_huella_sha256,p_politica_presentacion_ref,
        p_politica_presentacion_huella_sha256,p_sujeto_id_hmac,
        p_canal_sha256,p_asercion_actual_sha256,limite,
        NULL,NULL,NULL,instante);
    RETURN QUERY SELECT s.autenticacion_ref,s.autenticacion_huella_sha256,
        s.asercion_ref,s.sesion_ref,s.control_sesion_ref,s.control_sesion_revision,
        s.control_sesion_huella_sha256,s.cuenta_ref,s.cuenta_ordinaria_ref,
        s.cuenta_privilegiada,s.superficie,s.metodo_observado,s.garantia_observada,
        s.politica_garantia_ref,s.politica_garantia_huella_sha256,
        s.autenticacion_verificada_en,s.sesion_emitida_en,s.sesion_valida_hasta,
        s.sesion_revalidada_en,v.presentacion_ref,nueva_generacion,recibo,
        instante,limite,p_canal_sha256,p_asercion_actual_sha256,
        p_operacion_ref,v.sesion_generacion,'reanudada'::text;
END $funcion$;

-- La apertura no admite sesiones IS2 preexistentes. registrar_sesion_v1,
-- vinculo, consumo y AD221 confirman o revierten juntos.
CREATE FUNCTION vec_identidad_sesiones_v1.abrir_presentacion_certificado_v1(
    p_operacion_ref text, p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea, p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea, p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea, p_cuenta_privilegiada boolean,
    p_superficie text, p_metodo_observado text, p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz, p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text, p_politica_garantia_huella_sha256 text,
    p_certificado_der_hmac bytea, p_ca_hmac bytea, p_nonce_hmac bytea,
    p_canal_sha256 text, p_asercion_actual_sha256 text,
    p_asercion_actual_emitida_en timestamptz,
    p_asercion_actual_expira_en timestamptz,
    p_certificado_valido_hasta timestamptz, p_ca_valida_hasta timestamptz,
    p_crl_siguiente_actualizacion timestamptz,
    p_revocacion_verificada_en timestamptz,
    p_politica_presentacion_ref text,
    p_politica_presentacion_huella_sha256 text,
    p_acr_actual text
)
RETURNS TABLE (
    autenticacion_ref text, autenticacion_huella_sha256 text,
    asercion_ref text, sesion_ref text, control_sesion_ref text,
    control_sesion_revision text, control_sesion_huella_sha256 text,
    cuenta_ref text, cuenta_ordinaria_ref text, cuenta_privilegiada boolean,
    superficie text, metodo_observado text, garantia_observada text,
    politica_garantia_ref text, politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz, sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz, sesion_revalidada_en timestamptz,
    presentacion_ref text, presentacion_generacion bigint,
    presentacion_recibo_sha256 text, presentacion_registrada_en timestamptz,
    presentacion_valida_hasta timestamptz, canal_sha256 text,
    asercion_actual_sha256 text, operacion_ref text,
    sesion_generacion bigint, modo_inicio text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE g record; v record; alta record; resultado record;
    primera boolean:=false;
    preimagen text; instante timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.pg_is_in_recovery()
       OR vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
            p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,p_clave_hmac_version) IS NOT TRUE
       OR p_metodo_observado IS DISTINCT FROM 'certificado'
       OR p_garantia_observada IS DISTINCT FROM 'sustancial'
       OR p_acr_actual IS DISTINCT FROM
           'urn:vec:acr:certificado-desarrollo-protegido'
       OR p_politica_garantia_ref IS DISTINCT FROM p_politica_presentacion_ref
       OR p_politica_garantia_huella_sha256 IS DISTINCT FROM
           p_politica_presentacion_huella_sha256 THEN RETURN; END IF;
    preimagen:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.jsonb_build_array(p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,
          p_clave_hmac_id,p_clave_hmac_version,p_asercion_id_hmac,p_sesion_id_hmac,
          p_sujeto_id_hmac,p_cuenta_id_hmac,p_cuenta_ordinaria_id_hmac,
          p_cuenta_privilegiada,p_superficie,p_metodo_observado,p_garantia_observada,
          p_autenticacion_huella_sha256,p_autenticacion_verificada_en,
          p_sesion_emitida_en,p_asercion_expira_en,p_politica_garantia_ref,
          p_politica_garantia_huella_sha256,p_certificado_der_hmac,p_ca_hmac,
          p_nonce_hmac,p_canal_sha256,p_asercion_actual_sha256,
          p_asercion_actual_emitida_en,p_asercion_actual_expira_en,
          p_certificado_valido_hasta,p_ca_valida_hasta,p_crl_siguiente_actualizacion,
          p_revocacion_verificada_en,p_politica_presentacion_ref,
          p_politica_presentacion_huella_sha256,p_acr_actual)::text,'UTF8')),'hex');
    SELECT * INTO g FROM vec_identidad_sesiones_v1.gobierno_presentacion_certificado_v1
        WHERE singleton FOR SHARE;
    instante:=pg_catalog.clock_timestamp();
    IF NOT FOUND OR NOT g.activa OR instante >= g.retirar_en
       OR p_esquema_hmac IS DISTINCT FROM g.esquema_hmac
       OR p_dominio_hmac_ref IS DISTINCT FROM g.dominio_hmac_ref
       OR p_clave_hmac_id IS DISTINCT FROM g.clave_hmac_id
       OR p_clave_hmac_version IS DISTINCT FROM g.clave_hmac_version
       OR p_politica_presentacion_ref IS DISTINCT FROM g.politica_ref
       OR p_politica_presentacion_huella_sha256 IS DISTINCT FROM g.politica_huella_sha256
       OR p_certificado_valido_hasta <= instante OR p_ca_valida_hasta <= instante
       OR p_crl_siguiente_actualizacion <= instante
       OR p_revocacion_verificada_en > instante
       OR p_revocacion_verificada_en < p_asercion_actual_emitida_en
       OR p_asercion_actual_emitida_en > instante
       OR p_asercion_actual_expira_en <= instante
       OR p_asercion_actual_expira_en > p_asercion_expira_en THEN RETURN; END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_sesiones_v1:presentacion:cert:' ||
        pg_catalog.encode(p_certificado_der_hmac,'hex'),0));
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
      WHERE certificado_der_hmac=p_certificado_der_hmac;
    IF FOUND THEN
        IF v.apertura_operacion_ref IS DISTINCT FROM p_operacion_ref
           OR v.apertura_preimagen_sha256 IS DISTINCT FROM preimagen
           OR v.tumba_en IS NOT NULL THEN RETURN; END IF;
    ELSE
        primera:=true;
        SELECT * INTO alta FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
            p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
            p_clave_hmac_version,p_asercion_id_hmac,p_sesion_id_hmac,
            p_sujeto_id_hmac,p_cuenta_id_hmac,p_cuenta_ordinaria_id_hmac,
            p_cuenta_privilegiada,p_superficie,p_metodo_observado,
            p_garantia_observada,p_autenticacion_huella_sha256,
            p_autenticacion_verificada_en,p_sesion_emitida_en,
            p_asercion_expira_en,p_politica_garantia_ref,
            p_politica_garantia_huella_sha256);
        IF NOT FOUND THEN RETURN; END IF;
        INSERT INTO vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1(
            presentacion_ref,apertura_operacion_ref,apertura_preimagen_sha256,
            esquema_hmac,dominio_hmac_ref,clave_hmac_id,clave_hmac_version,
            certificado_der_hmac,ca_hmac,sujeto_id_hmac,cuenta_id_hmac,acr_original,
            autenticacion_ref,sesion_ref,cuenta_ref,superficie,creada_en)
        VALUES ('prs_'||pg_catalog.encode(public.gen_random_bytes(18),'hex'),p_operacion_ref,
            preimagen,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
            p_clave_hmac_version,p_certificado_der_hmac,p_ca_hmac,
            p_sujeto_id_hmac,p_cuenta_id_hmac,p_acr_actual,alta.autenticacion_ref,
            alta.sesion_ref,alta.cuenta_ref,p_superficie,instante);
        INSERT INTO vec_identidad_sesiones_v1.historia_sesion_presentacion_v1(
            presentacion_ref,sesion_generacion,operacion_ref,preimagen_sha256,
            autenticacion_ref,sesion_ref,registrada_en)
        SELECT x.presentacion_ref,1,p_operacion_ref,preimagen,
               alta.autenticacion_ref,alta.sesion_ref,instante
          FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1 x
         WHERE x.certificado_der_hmac=p_certificado_der_hmac;
    END IF;
    SELECT * INTO resultado FROM vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(
        p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
        p_clave_hmac_version,p_superficie,p_certificado_der_hmac,p_ca_hmac,
        p_sujeto_id_hmac,p_cuenta_id_hmac,p_nonce_hmac,p_canal_sha256,
        p_asercion_actual_sha256,p_asercion_actual_emitida_en,
        p_asercion_actual_expira_en,p_certificado_valido_hasta,p_ca_valida_hasta,
        p_crl_siguiente_actualizacion,p_revocacion_verificada_en,
        p_politica_presentacion_ref,p_politica_presentacion_huella_sha256,
        p_metodo_observado,p_garantia_observada,p_acr_actual);
    IF NOT FOUND THEN
        RAISE EXCEPTION 'IS18: apertura no confirmada' USING ERRCODE='42501';
    END IF;
    resultado.modo_inicio:=CASE WHEN primera THEN 'abierta' ELSE 'reanudada' END;
    RETURN QUERY SELECT
        resultado.autenticacion_ref, resultado.autenticacion_huella_sha256, resultado.asercion_ref,
        resultado.sesion_ref, resultado.control_sesion_ref, resultado.control_sesion_revision,
        resultado.control_sesion_huella_sha256, resultado.cuenta_ref, resultado.cuenta_ordinaria_ref,
        resultado.cuenta_privilegiada, resultado.superficie, resultado.metodo_observado,
        resultado.garantia_observada, resultado.politica_garantia_ref, resultado.politica_garantia_huella_sha256,
        resultado.autenticacion_verificada_en, resultado.sesion_emitida_en, resultado.sesion_valida_hasta,
        resultado.sesion_revalidada_en, resultado.presentacion_ref, resultado.presentacion_generacion,
        resultado.presentacion_recibo_sha256, resultado.presentacion_registrada_en, resultado.presentacion_valida_hasta,
        resultado.canal_sha256, resultado.asercion_actual_sha256, resultado.operacion_ref,
        resultado.sesion_generacion, resultado.modo_inicio;
END $funcion$;

-- Renovacion explicita desde INICIO: solo la sesion actual activa y expirada
-- puede dar paso a una nueva sesion IS2. El CAS evita un alta concurrente.
CREATE FUNCTION vec_identidad_sesiones_v1.renovar_sesion_certificado_v1(
    p_operacion_ref text, p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea, p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea, p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea, p_cuenta_privilegiada boolean,
    p_superficie text, p_metodo_observado text, p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz, p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text, p_politica_garantia_huella_sha256 text,
    p_certificado_der_hmac bytea, p_ca_hmac bytea, p_nonce_hmac bytea,
    p_canal_sha256 text, p_asercion_actual_sha256 text,
    p_asercion_actual_emitida_en timestamptz,
    p_asercion_actual_expira_en timestamptz,
    p_certificado_valido_hasta timestamptz, p_ca_valida_hasta timestamptz,
    p_crl_siguiente_actualizacion timestamptz,
    p_revocacion_verificada_en timestamptz,
    p_politica_presentacion_ref text,
    p_politica_presentacion_huella_sha256 text,
    p_acr_actual text,
    p_generacion_esperada bigint
)
RETURNS TABLE (
    autenticacion_ref text, autenticacion_huella_sha256 text,
    asercion_ref text, sesion_ref text, control_sesion_ref text,
    control_sesion_revision text, control_sesion_huella_sha256 text,
    cuenta_ref text, cuenta_ordinaria_ref text, cuenta_privilegiada boolean,
    superficie text, metodo_observado text, garantia_observada text,
    politica_garantia_ref text, politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz, sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz, sesion_revalidada_en timestamptz,
    presentacion_ref text, presentacion_generacion bigint,
    presentacion_recibo_sha256 text, presentacion_registrada_en timestamptz,
    presentacion_valida_hasta timestamptz, canal_sha256 text,
    asercion_actual_sha256 text, operacion_ref text,
    sesion_generacion bigint, modo_inicio text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE g record; v record; ctl record; alta record; resultado record; h record;
    original record; cuenta_alias_ref text;
    nueva boolean:=false;
    preimagen text; instante timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.pg_is_in_recovery()
       OR vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(
            p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,p_clave_hmac_version) IS NOT TRUE
       OR p_metodo_observado IS DISTINCT FROM 'certificado'
       OR p_garantia_observada IS DISTINCT FROM 'sustancial'
       OR p_acr_actual IS DISTINCT FROM
           'urn:vec:acr:certificado-desarrollo-protegido'
       OR p_politica_garantia_ref IS DISTINCT FROM p_politica_presentacion_ref
       OR p_politica_garantia_huella_sha256 IS DISTINCT FROM
           p_politica_presentacion_huella_sha256
       OR p_generacion_esperada IS NULL OR p_generacion_esperada<1
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_sesion_id_hmac) IS NOT TRUE
       THEN RETURN; END IF;
    -- registrar_sesion_v1 bloquea primero este HMAC de sesion IdP y despues
    -- cuentas. La renovacion conserva ese orden antes de tomar controles.
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_sesiones_v1:sesion:'||p_dominio_hmac_ref||':'||
        p_clave_hmac_id||':'||p_clave_hmac_version::text||':'||
        pg_catalog.encode(p_sesion_id_hmac,'hex'),0));
    preimagen:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.jsonb_build_array(p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,
          p_clave_hmac_id,p_clave_hmac_version,p_asercion_id_hmac,p_sesion_id_hmac,
          p_sujeto_id_hmac,p_cuenta_id_hmac,p_cuenta_ordinaria_id_hmac,
          p_cuenta_privilegiada,p_superficie,p_metodo_observado,p_garantia_observada,
          p_autenticacion_huella_sha256,p_autenticacion_verificada_en,
          p_sesion_emitida_en,p_asercion_expira_en,p_politica_garantia_ref,
          p_politica_garantia_huella_sha256,p_certificado_der_hmac,p_ca_hmac,
          p_nonce_hmac,p_canal_sha256,p_asercion_actual_sha256,
          p_asercion_actual_emitida_en,p_asercion_actual_expira_en,
          p_certificado_valido_hasta,p_ca_valida_hasta,p_crl_siguiente_actualizacion,
          p_revocacion_verificada_en,p_politica_presentacion_ref,
          p_politica_presentacion_huella_sha256,p_acr_actual,
          p_generacion_esperada)::text,
          'UTF8')),'hex');
    SELECT * INTO g FROM vec_identidad_sesiones_v1.gobierno_presentacion_certificado_v1
        WHERE singleton FOR SHARE;
    instante:=pg_catalog.clock_timestamp();
    IF NOT FOUND OR NOT g.activa OR instante>=g.retirar_en
       OR p_esquema_hmac IS DISTINCT FROM g.esquema_hmac
       OR p_dominio_hmac_ref IS DISTINCT FROM g.dominio_hmac_ref
       OR p_clave_hmac_id IS DISTINCT FROM g.clave_hmac_id
       OR p_clave_hmac_version IS DISTINCT FROM g.clave_hmac_version
       OR p_politica_presentacion_ref IS DISTINCT FROM g.politica_ref
       OR p_politica_presentacion_huella_sha256 IS DISTINCT FROM g.politica_huella_sha256
       OR p_certificado_valido_hasta<=instante OR p_ca_valida_hasta<=instante
       OR p_crl_siguiente_actualizacion<=instante
       OR p_revocacion_verificada_en>instante
       OR p_revocacion_verificada_en<p_asercion_actual_emitida_en
       OR p_asercion_actual_emitida_en>instante
       OR p_asercion_actual_expira_en<=instante
       OR p_asercion_actual_expira_en>p_asercion_expira_en THEN RETURN; END IF;
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
     WHERE certificado_der_hmac=p_certificado_der_hmac;
    IF NOT FOUND OR v.tumba_en IS NOT NULL
       OR v.ca_hmac IS DISTINCT FROM p_ca_hmac
       OR v.sujeto_id_hmac IS DISTINCT FROM p_sujeto_id_hmac
       OR v.cuenta_id_hmac IS DISTINCT FROM p_cuenta_id_hmac
       OR v.acr_original IS DISTINCT FROM p_acr_actual
       OR v.superficie IS DISTINCT FROM p_superficie
       OR v.dominio_hmac_ref IS DISTINCT FROM p_dominio_hmac_ref THEN RETURN; END IF;
    -- Orden global: control -> cuentas -> raiz. registrar_sesion_v1 solo
    -- volvera a bloquear la cuenta ya retenida por esta transaccion.
    SELECT c.estado,c.sesion_valida_hasta INTO ctl
      FROM vec_autorizacion.control_sesion_actual_v1 a
      JOIN vec_autorizacion.control_sesion_v1 c
        ON c.sesion_ref=a.sesion_ref AND c.control_sesion_ref=a.control_sesion_ref
       AND c.revision=a.revision
     WHERE a.sesion_ref=v.sesion_ref FOR UPDATE OF a;
    IF NOT FOUND OR ctl.estado IS DISTINCT FROM 'activa' THEN RETURN; END IF;
    IF vec_identidad_sesiones_v1.bloquear_cuentas_presentacion_v1(
        ARRAY[v.sesion_ref]) IS DISTINCT FROM true THEN RETURN; END IF;
    SELECT base.metodo_observado,base.garantia_observada,
           base.cuenta_ref,base.cuenta_ordinaria_ref,base.cuenta_privilegiada,
           base.politica_garantia_ref,base.politica_garantia_huella_sha256
      INTO original FROM vec_autorizacion.sesion_autenticacion_v1 base
     WHERE base.sesion_ref=v.sesion_ref
       AND base.autenticacion_ref=v.autenticacion_ref;
    IF NOT FOUND OR original.metodo_observado IS DISTINCT FROM 'certificado'
       OR original.garantia_observada IS DISTINCT FROM 'sustancial'
       OR original.politica_garantia_ref IS DISTINCT FROM p_politica_presentacion_ref
       OR original.politica_garantia_huella_sha256 IS DISTINCT FROM
           p_politica_presentacion_huella_sha256 THEN RETURN; END IF;
    -- Este corte DEV sustancial es de cuenta ordinaria. Evitar que la nueva
    -- asercion resuelva otra cuenta y tome un cerrojo no prebloqueado.
    IF original.cuenta_ref IS DISTINCT FROM v.cuenta_ref
       OR original.cuenta_ordinaria_ref IS DISTINCT FROM v.cuenta_ref
       OR original.cuenta_privilegiada
       OR p_cuenta_privilegiada IS DISTINCT FROM false
       OR p_cuenta_ordinaria_id_hmac IS NOT NULL THEN RETURN; END IF;
    SELECT a.cuenta_ref INTO cuenta_alias_ref
      FROM vec_identidad_sesiones_v1.alias_hmac_cuenta a
     WHERE a.esquema_hmac=p_esquema_hmac
       AND a.dominio_hmac_ref=p_dominio_hmac_ref
       AND a.clave_hmac_id=p_clave_hmac_id
       AND a.clave_hmac_version=p_clave_hmac_version
       AND a.sujeto_id_hmac=p_sujeto_id_hmac
       AND a.cuenta_id_hmac=p_cuenta_id_hmac;
    IF NOT FOUND OR cuenta_alias_ref IS DISTINCT FROM v.cuenta_ref THEN RETURN; END IF;
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
     WHERE certificado_der_hmac=p_certificado_der_hmac FOR UPDATE;
    IF NOT FOUND OR v.tumba_en IS NOT NULL
       OR v.sesion_generacion NOT IN (p_generacion_esperada,p_generacion_esperada+1)
       OR v.ca_hmac IS DISTINCT FROM p_ca_hmac
       OR v.sujeto_id_hmac IS DISTINCT FROM p_sujeto_id_hmac
       OR v.cuenta_id_hmac IS DISTINCT FROM p_cuenta_id_hmac
       OR v.acr_original IS DISTINCT FROM p_acr_actual
       OR v.superficie IS DISTINCT FROM p_superficie THEN RETURN; END IF;
    -- Repeticion exacta de la renovacion ya confirmada: revalidar su sesion
    -- presente y su consumo; no crear otra. Una generacion posterior deniega.
    IF v.sesion_generacion=p_generacion_esperada+1 THEN
        SELECT * INTO h FROM vec_identidad_sesiones_v1.historia_sesion_presentacion_v1 x
         WHERE x.presentacion_ref=v.presentacion_ref
           AND x.sesion_generacion=v.sesion_generacion;
        IF NOT FOUND OR h.operacion_ref IS DISTINCT FROM p_operacion_ref
           OR h.preimagen_sha256 IS DISTINCT FROM preimagen THEN RETURN; END IF;
    ELSE
        nueva:=true;
        IF instante<ctl.sesion_valida_hasta
           OR NOT EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion c
                WHERE c.sesion_ref=v.sesion_ref AND c.cuenta_ref=v.cuenta_ref
                  AND c.sujeto_id_hmac=p_sujeto_id_hmac
                  AND c.cuenta_id_hmac=p_cuenta_id_hmac) THEN RETURN; END IF;
        SELECT * INTO alta FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
            p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
            p_clave_hmac_version,p_asercion_id_hmac,p_sesion_id_hmac,
            p_sujeto_id_hmac,p_cuenta_id_hmac,p_cuenta_ordinaria_id_hmac,
            p_cuenta_privilegiada,p_superficie,p_metodo_observado,
            p_garantia_observada,p_autenticacion_huella_sha256,
            p_autenticacion_verificada_en,p_sesion_emitida_en,
            p_asercion_expira_en,p_politica_garantia_ref,
            p_politica_garantia_huella_sha256);
        IF NOT FOUND OR alta.cuenta_ref IS DISTINCT FROM v.cuenta_ref THEN
            RAISE EXCEPTION 'IS18: renovacion no confirmada' USING ERRCODE='42501';
        END IF;
        INSERT INTO vec_identidad_sesiones_v1.historia_sesion_presentacion_v1(
            presentacion_ref,sesion_generacion,operacion_ref,preimagen_sha256,
            autenticacion_ref,sesion_ref,registrada_en)
        VALUES (v.presentacion_ref,v.sesion_generacion+1,p_operacion_ref,
            preimagen,alta.autenticacion_ref,alta.sesion_ref,instante);
        UPDATE vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1 x
           SET sesion_generacion=v.sesion_generacion+1,
               autenticacion_ref=alta.autenticacion_ref,
               sesion_ref=alta.sesion_ref
         WHERE x.presentacion_ref=v.presentacion_ref;
    END IF;
    SELECT * INTO resultado FROM vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(
        p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
        p_clave_hmac_version,p_superficie,p_certificado_der_hmac,p_ca_hmac,
        p_sujeto_id_hmac,p_cuenta_id_hmac,p_nonce_hmac,p_canal_sha256,
        p_asercion_actual_sha256,p_asercion_actual_emitida_en,
        p_asercion_actual_expira_en,p_certificado_valido_hasta,p_ca_valida_hasta,
        p_crl_siguiente_actualizacion,p_revocacion_verificada_en,
        p_politica_presentacion_ref,p_politica_presentacion_huella_sha256,
        p_metodo_observado,p_garantia_observada,p_acr_actual);
    IF NOT FOUND THEN
        RAISE EXCEPTION 'IS18: renovacion sin consumo' USING ERRCODE='42501';
    END IF;
    resultado.modo_inicio:=CASE WHEN nueva THEN 'renovada' ELSE 'reanudada' END;
    RETURN QUERY SELECT
        resultado.autenticacion_ref, resultado.autenticacion_huella_sha256, resultado.asercion_ref,
        resultado.sesion_ref, resultado.control_sesion_ref, resultado.control_sesion_revision,
        resultado.control_sesion_huella_sha256, resultado.cuenta_ref, resultado.cuenta_ordinaria_ref,
        resultado.cuenta_privilegiada, resultado.superficie, resultado.metodo_observado,
        resultado.garantia_observada, resultado.politica_garantia_ref, resultado.politica_garantia_huella_sha256,
        resultado.autenticacion_verificada_en, resultado.sesion_emitida_en, resultado.sesion_valida_hasta,
        resultado.sesion_revalidada_en, resultado.presentacion_ref, resultado.presentacion_generacion,
        resultado.presentacion_recibo_sha256, resultado.presentacion_registrada_en, resultado.presentacion_valida_hasta,
        resultado.canal_sha256, resultado.asercion_actual_sha256, resultado.operacion_ref,
        resultado.sesion_generacion, resultado.modo_inicio;
END $funcion$;

-- Unica entrada de INICIO. No acepta un estado leído en una petición anterior:
-- elige bajo los locks de IS2 y vuelve a comprobar toda la prueba actual.
CREATE FUNCTION vec_identidad_sesiones_v1.iniciar_y_consumir_presentacion_certificado_v1(
    p_operacion_ref text, p_esquema_hmac text, p_dominio_hmac_ref text,
    p_clave_hmac_id text, p_clave_hmac_version bigint,
    p_asercion_id_hmac bytea, p_sesion_id_hmac bytea,
    p_sujeto_id_hmac bytea, p_cuenta_id_hmac bytea,
    p_cuenta_ordinaria_id_hmac bytea, p_cuenta_privilegiada boolean,
    p_superficie text, p_metodo_observado text, p_garantia_observada text,
    p_autenticacion_huella_sha256 text,
    p_autenticacion_verificada_en timestamptz,
    p_sesion_emitida_en timestamptz, p_asercion_expira_en timestamptz,
    p_politica_garantia_ref text, p_politica_garantia_huella_sha256 text,
    p_certificado_der_hmac bytea, p_ca_hmac bytea, p_nonce_hmac bytea,
    p_canal_sha256 text, p_asercion_actual_sha256 text,
    p_asercion_actual_emitida_en timestamptz,
    p_asercion_actual_expira_en timestamptz,
    p_certificado_valido_hasta timestamptz, p_ca_valida_hasta timestamptz,
    p_crl_siguiente_actualizacion timestamptz,
    p_revocacion_verificada_en timestamptz,
    p_politica_presentacion_ref text,
    p_politica_presentacion_huella_sha256 text,
    p_acr_actual text
)
RETURNS TABLE (
    autenticacion_ref text, autenticacion_huella_sha256 text,
    asercion_ref text, sesion_ref text, control_sesion_ref text,
    control_sesion_revision text, control_sesion_huella_sha256 text,
    cuenta_ref text, cuenta_ordinaria_ref text, cuenta_privilegiada boolean,
    superficie text, metodo_observado text, garantia_observada text,
    politica_garantia_ref text, politica_garantia_huella_sha256 text,
    autenticacion_verificada_en timestamptz, sesion_emitida_en timestamptz,
    sesion_valida_hasta timestamptz, sesion_revalidada_en timestamptz,
    presentacion_ref text, presentacion_generacion bigint,
    presentacion_recibo_sha256 text, presentacion_registrada_en timestamptz,
    presentacion_valida_hasta timestamptz, canal_sha256 text,
    asercion_actual_sha256 text, operacion_ref text,
    sesion_generacion bigint, modo_inicio text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE v record; ctl record; r record; generacion_inicial bigint;
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.pg_is_in_recovery()
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_certificado_der_hmac) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_sesion_id_hmac) IS NOT TRUE
       OR p_acr_actual IS DISTINCT FROM
           'urn:vec:acr:certificado-desarrollo-protegido'
       OR vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE THEN
        RETURN;
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_sesiones_v1:presentacion:cert:'||
        pg_catalog.encode(p_certificado_der_hmac,'hex'),0));
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_identidad_sesiones_v1:sesion:'||p_dominio_hmac_ref||':'||
        p_clave_hmac_id||':'||p_clave_hmac_version::text||':'||
        pg_catalog.encode(p_sesion_id_hmac,'hex'),0));
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
     WHERE certificado_der_hmac=p_certificado_der_hmac;
    IF NOT FOUND THEN
        SELECT * INTO r FROM vec_identidad_sesiones_v1.abrir_presentacion_certificado_v1(
            p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
            p_clave_hmac_version,p_asercion_id_hmac,p_sesion_id_hmac,
            p_sujeto_id_hmac,p_cuenta_id_hmac,p_cuenta_ordinaria_id_hmac,
            p_cuenta_privilegiada,p_superficie,p_metodo_observado,
            p_garantia_observada,p_autenticacion_huella_sha256,
            p_autenticacion_verificada_en,p_sesion_emitida_en,
            p_asercion_expira_en,p_politica_garantia_ref,
            p_politica_garantia_huella_sha256,p_certificado_der_hmac,
            p_ca_hmac,p_nonce_hmac,p_canal_sha256,p_asercion_actual_sha256,
            p_asercion_actual_emitida_en,p_asercion_actual_expira_en,
            p_certificado_valido_hasta,p_ca_valida_hasta,
            p_crl_siguiente_actualizacion,p_revocacion_verificada_en,
            p_politica_presentacion_ref,p_politica_presentacion_huella_sha256,
            p_acr_actual);
        IF FOUND THEN
        RETURN QUERY SELECT
            r.autenticacion_ref, r.autenticacion_huella_sha256, r.asercion_ref,
            r.sesion_ref, r.control_sesion_ref, r.control_sesion_revision,
            r.control_sesion_huella_sha256, r.cuenta_ref, r.cuenta_ordinaria_ref,
            r.cuenta_privilegiada, r.superficie, r.metodo_observado,
            r.garantia_observada, r.politica_garantia_ref, r.politica_garantia_huella_sha256,
            r.autenticacion_verificada_en, r.sesion_emitida_en, r.sesion_valida_hasta,
            r.sesion_revalidada_en, r.presentacion_ref, r.presentacion_generacion,
            r.presentacion_recibo_sha256, r.presentacion_registrada_en, r.presentacion_valida_hasta,
            r.canal_sha256, r.asercion_actual_sha256, r.operacion_ref,
            r.sesion_generacion, r.modo_inicio;
    END IF;
        RETURN;
    END IF;
    IF v.tumba_en IS NOT NULL THEN RETURN; END IF;
    generacion_inicial:=v.sesion_generacion;
    -- Este es el orden de revocar_sesion_v1 + su trigger: control -> raiz.
    SELECT c.estado,c.sesion_valida_hasta INTO ctl
      FROM vec_autorizacion.control_sesion_actual_v1 a
      JOIN vec_autorizacion.control_sesion_v1 c
        ON c.sesion_ref=a.sesion_ref AND c.control_sesion_ref=a.control_sesion_ref
       AND c.revision=a.revision
     WHERE a.sesion_ref=v.sesion_ref FOR UPDATE OF a;
    IF NOT FOUND OR ctl.estado IS DISTINCT FROM 'activa' THEN RETURN; END IF;
    IF vec_identidad_sesiones_v1.bloquear_cuentas_presentacion_v1(
        ARRAY[v.sesion_ref]) IS DISTINCT FROM true THEN RETURN; END IF;
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
     WHERE certificado_der_hmac=p_certificado_der_hmac FOR UPDATE;
    IF NOT FOUND OR v.tumba_en IS NOT NULL
       OR v.sesion_generacion IS DISTINCT FROM generacion_inicial THEN RETURN; END IF;
    IF pg_catalog.clock_timestamp() < ctl.sesion_valida_hasta THEN
        SELECT * INTO r FROM vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(
            p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
            p_clave_hmac_version,p_superficie,p_certificado_der_hmac,p_ca_hmac,
            p_sujeto_id_hmac,p_cuenta_id_hmac,p_nonce_hmac,p_canal_sha256,
            p_asercion_actual_sha256,p_asercion_actual_emitida_en,
            p_asercion_actual_expira_en,p_certificado_valido_hasta,p_ca_valida_hasta,
            p_crl_siguiente_actualizacion,p_revocacion_verificada_en,
            p_politica_presentacion_ref,p_politica_presentacion_huella_sha256,
            p_metodo_observado,p_garantia_observada,p_acr_actual);
    ELSE
        SELECT * INTO r FROM vec_identidad_sesiones_v1.renovar_sesion_certificado_v1(
            p_operacion_ref,p_esquema_hmac,p_dominio_hmac_ref,p_clave_hmac_id,
            p_clave_hmac_version,p_asercion_id_hmac,p_sesion_id_hmac,
            p_sujeto_id_hmac,p_cuenta_id_hmac,p_cuenta_ordinaria_id_hmac,
            p_cuenta_privilegiada,p_superficie,p_metodo_observado,
            p_garantia_observada,p_autenticacion_huella_sha256,
            p_autenticacion_verificada_en,p_sesion_emitida_en,
            p_asercion_expira_en,p_politica_garantia_ref,
            p_politica_garantia_huella_sha256,p_certificado_der_hmac,
            p_ca_hmac,p_nonce_hmac,p_canal_sha256,p_asercion_actual_sha256,
            p_asercion_actual_emitida_en,p_asercion_actual_expira_en,
            p_certificado_valido_hasta,p_ca_valida_hasta,
            p_crl_siguiente_actualizacion,p_revocacion_verificada_en,
            p_politica_presentacion_ref,p_politica_presentacion_huella_sha256,
            p_acr_actual,
            generacion_inicial);
    END IF;
    IF FOUND THEN
        RETURN QUERY SELECT
            r.autenticacion_ref, r.autenticacion_huella_sha256, r.asercion_ref,
            r.sesion_ref, r.control_sesion_ref, r.control_sesion_revision,
            r.control_sesion_huella_sha256, r.cuenta_ref, r.cuenta_ordinaria_ref,
            r.cuenta_privilegiada, r.superficie, r.metodo_observado,
            r.garantia_observada, r.politica_garantia_ref, r.politica_garantia_huella_sha256,
            r.autenticacion_verificada_en, r.sesion_emitida_en, r.sesion_valida_hasta,
            r.sesion_revalidada_en, r.presentacion_ref, r.presentacion_generacion,
            r.presentacion_recibo_sha256, r.presentacion_registrada_en, r.presentacion_valida_hasta,
            r.canal_sha256, r.asercion_actual_sha256, r.operacion_ref,
            r.sesion_generacion, r.modo_inicio;
    END IF;
END $funcion$;

-- Revocacion propia tardia. El actor debe conservar OTRA sesion IS2 vigente
-- de la misma cuenta; una sesion vencida no acredita su propia revocacion.
-- El rol técnico no tiene LOGIN ni membresías sembradas por esta migración.
CREATE FUNCTION vec_identidad_sesiones_v1.revocar_vinculo_certificado_v1(
    p_operacion_ref text,p_certificado_der_hmac bytea,p_nonce_hmac bytea,
    p_actor_autenticacion_ref text,p_actor_sesion_ref text
)
RETURNS TABLE(recibo_sha256 text,tumba_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE actor record; v record; control_bloqueado record; politica_original record;
    preimagen text; recibo text; sesion_objetivo text; control_ref text;
    generacion_objetivo bigint;
    controles integer:=0; estado_objetivo text; hasta_objetivo timestamptz(6);
    instante timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.pg_is_in_recovery()
       OR vec_identidad_sesiones_v1.referencia_valida(p_operacion_ref,'opr_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(p_actor_autenticacion_ref,'aut_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.referencia_valida(p_actor_sesion_ref,'ses_') IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_certificado_der_hmac) IS NOT TRUE
       OR vec_identidad_sesiones_v1.huella_hmac_valida(p_nonce_hmac) IS NOT TRUE
       OR p_certificado_der_hmac=p_nonce_hmac THEN RETURN; END IF;
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
     WHERE certificado_der_hmac=p_certificado_der_hmac;
    IF NOT FOUND OR v.sesion_ref=p_actor_sesion_ref THEN RETURN; END IF;
    sesion_objetivo:=v.sesion_ref;
    generacion_objetivo:=v.sesion_generacion;
    -- Las dos sesiones se bloquean por referencia estable; el revocador IS2
    -- legacy solo toma uno de estos controles antes de su trigger de tumba.
    FOR control_ref IN
        SELECT x.sesion_ref FROM pg_catalog.unnest(
            ARRAY[p_actor_sesion_ref,sesion_objetivo]) x(sesion_ref)
         ORDER BY x.sesion_ref COLLATE "C"
    LOOP
        SELECT a.sesion_ref,c.estado,c.sesion_valida_hasta
          INTO control_bloqueado
          FROM vec_autorizacion.control_sesion_actual_v1 a
          JOIN vec_autorizacion.control_sesion_v1 c
            ON c.sesion_ref=a.sesion_ref
           AND c.control_sesion_ref=a.control_sesion_ref
           AND c.revision=a.revision
         WHERE a.sesion_ref=control_ref FOR UPDATE OF a;
        IF NOT FOUND THEN RETURN; END IF;
        controles:=controles+1;
        IF control_bloqueado.sesion_ref=sesion_objetivo THEN
            estado_objetivo:=control_bloqueado.estado;
            hasta_objetivo:=control_bloqueado.sesion_valida_hasta;
        END IF;
    END LOOP;
    IF controles<>2 OR estado_objetivo IS DISTINCT FROM 'activa' THEN RETURN; END IF;
    IF vec_identidad_sesiones_v1.bloquear_cuentas_presentacion_v1(
        ARRAY[p_actor_sesion_ref,sesion_objetivo]) IS DISTINCT FROM true THEN RETURN; END IF;
    SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
     WHERE certificado_der_hmac=p_certificado_der_hmac FOR UPDATE;
    IF NOT FOUND OR v.sesion_ref IS DISTINCT FROM sesion_objetivo
       OR v.sesion_generacion IS DISTINCT FROM generacion_objetivo THEN RETURN; END IF;
    -- IS6 vuelve a adquirir el control y las cuentas ya retenidos. La
    -- comprobacion nominal ocurre despues de fijar puntero y raiz actuales.
    SELECT * INTO actor FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
        p_actor_autenticacion_ref,p_actor_sesion_ref);
    IF NOT FOUND OR actor.cuenta_ref IS DISTINCT FROM v.cuenta_ref
       OR actor.sesion_ref IS DISTINCT FROM p_actor_sesion_ref THEN RETURN; END IF;
    instante:=pg_catalog.clock_timestamp();
    preimagen:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.jsonb_build_array(p_operacion_ref,p_certificado_der_hmac,
            p_nonce_hmac,p_actor_autenticacion_ref,p_actor_sesion_ref,
            v.presentacion_ref,v.sesion_generacion)::text,'UTF8')),'hex');
    IF v.tumba_en IS NOT NULL THEN
        IF v.tumba_operacion_ref=p_operacion_ref
           AND v.tumba_preimagen_sha256=preimagen THEN
            RETURN QUERY SELECT v.tumba_recibo_sha256,v.tumba_en;
        END IF;
        RETURN;
    END IF;
    -- La sesion actual aun vigente se revoca por el revocador IS2 ordinario.
    IF instante<hasta_objetivo THEN RETURN; END IF;
    SELECT b.politica_garantia_ref,b.politica_garantia_huella_sha256
      INTO politica_original FROM vec_autorizacion.sesion_autenticacion_v1 b
     WHERE b.sesion_ref=v.sesion_ref
       AND b.autenticacion_ref=v.autenticacion_ref;
    IF NOT FOUND THEN RETURN; END IF;
    recibo:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.jsonb_build_array('vec.presentacion.revocacion-propia.v1',
            p_operacion_ref,v.presentacion_ref,v.sesion_ref,actor.cuenta_ref,
            actor.autenticacion_ref,actor.sesion_ref,preimagen,instante)::text,
            'UTF8')),'hex');
    UPDATE vec_identidad_sesiones_v1.vinculo_presentacion_certificado_v1
       SET tumba_en=instante,tumba_operacion_ref=p_operacion_ref,
           tumba_recibo_sha256=recibo,tumba_preimagen_sha256=preimagen
     WHERE presentacion_ref=v.presentacion_ref;
    PERFORM vec_autorizacion_atestada_v3.registrar_auditoria_presentacion_certificado_v1(
        p_operacion_ref,v.presentacion_ref,v.autenticacion_ref,v.sesion_ref,
        v.cuenta_ref,actor.cuenta_ref,actor.autenticacion_ref,actor.sesion_ref,
        v.superficie,'revocacion',recibo,v.acr_original,NULL,
        politica_original.politica_garantia_ref,
        politica_original.politica_garantia_huella_sha256,NULL,NULL,
        v.sujeto_id_hmac,NULL,NULL,NULL,NULL,NULL,NULL,instante);
    RETURN QUERY SELECT recibo,instante;
END $funcion$;

REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(
    text,text,text,text,bigint,text,bytea,bytea,bytea,bytea,bytea,text,text,
    timestamptz,timestamptz,timestamptz,timestamptz,timestamptz,
    timestamptz,text,text,text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.abrir_presentacion_certificado_v1(
    text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,
    text,text,text,text,timestamptz,timestamptz,timestamptz,text,text,
    bytea,bytea,bytea,text,text,timestamptz,timestamptz,timestamptz,
    timestamptz,timestamptz,timestamptz,text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.renovar_sesion_certificado_v1(
    text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,
    text,text,text,text,timestamptz,timestamptz,timestamptz,text,text,
    bytea,bytea,bytea,text,text,timestamptz,timestamptz,timestamptz,
    timestamptz,timestamptz,timestamptz,text,text,text,bigint) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.iniciar_y_consumir_presentacion_certificado_v1(
    text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,
    text,text,text,text,timestamptz,timestamptz,timestamptz,text,text,
    bytea,bytea,bytea,text,text,timestamptz,timestamptz,timestamptz,
    timestamptz,timestamptz,timestamptz,text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.revocar_vinculo_certificado_v1(
    text,bytea,bytea,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.bloquear_cambio_gobierno_presentacion_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.solo_tumba_presentacion_v1() FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1
    TO vec_identidad_sesiones_v1_presentador,
       vec_identidad_sesiones_v1_revocador_presentacion;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(
    text,text,text,text,bigint,text,bytea,bytea,bytea,bytea,bytea,text,text,
    timestamptz,timestamptz,timestamptz,timestamptz,timestamptz,
    timestamptz,text,text,text,text,text)
    TO vec_identidad_sesiones_v1_presentador;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.iniciar_y_consumir_presentacion_certificado_v1(
    text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,
    text,text,text,text,timestamptz,timestamptz,timestamptz,text,text,
    bytea,bytea,bytea,text,text,timestamptz,timestamptz,timestamptz,
    timestamptz,timestamptz,timestamptz,text,text,text)
    TO vec_identidad_sesiones_v1_presentador;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.revocar_vinculo_certificado_v1(
    text,bytea,bytea,text,text)
    TO vec_identidad_sesiones_v1_revocador_presentacion;
COMMIT;
