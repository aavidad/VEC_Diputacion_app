\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_administracion_migrador;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_administracion:configuracion_correo:000001', 0));
SET LOCAL ROLE vec_administracion_propietario;

-- Dependencia deliberada: AD3-33 ha de instalar primero el consumidor V3
-- nominal. El actor incluido en la auditoría no es una autoridad suficiente.
DO $dependencia$
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'falta consumidor V3 nominal de configuracion de correo' USING ERRCODE = '55000';
    END IF;
END
$dependencia$;

CREATE SCHEMA vec_administracion AUTHORIZATION vec_administracion_propietario;
REVOKE ALL ON SCHEMA vec_administracion FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_administracion TO vec_administracion_ejecutor;

CREATE TABLE vec_administracion.configuracion_correo (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    version bigint NOT NULL CHECK (version > 0),
    host text NOT NULL CHECK (length(host) BETWEEN 1 AND 253),
    puerto integer NOT NULL CHECK (puerto BETWEEN 1 AND 65535),
    nombre_servidor text NOT NULL CHECK (length(nombre_servidor) BETWEEN 1 AND 253),
    referencia_ca text NOT NULL CHECK (length(referencia_ca) BETWEEN 1 AND 512),
    remitente_fijo text NOT NULL CHECK (length(remitente_fijo) BETWEEN 3 AND 320),
    usuario text NOT NULL,
    modo_tls text NOT NULL CHECK (modo_tls IN ('tls_implicito','starttls_obligatorio')),
    modo_autenticacion text NOT NULL CHECK (modo_autenticacion IN ('ninguna','plain','xoauth2')),
    tiempo_maximo_ms bigint NOT NULL CHECK (tiempo_maximo_ms BETWEEN 1 AND 3600000),
    version_secreto bigint CHECK (version_secreto > 0),
    actualizada_en timestamptz NOT NULL DEFAULT clock_timestamp()
);
ALTER TABLE vec_administracion.configuracion_correo OWNER TO vec_administracion_propietario;

CREATE TABLE vec_administracion.sobre_configuracion_correo (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    version_secreto bigint NOT NULL CHECK (version_secreto > 0),
    clave_ref text NOT NULL CHECK (length(clave_ref) BETWEEN 1 AND 512),
    nonce bytea NOT NULL CHECK (octet_length(nonce) BETWEEN 12 AND 64),
    secreto_cifrado bytea NOT NULL CHECK (octet_length(secreto_cifrado) BETWEEN 16 AND 32768),
    huella_aad_sha256 text NOT NULL CHECK (huella_aad_sha256 ~ '^[0-9a-f]{64}$'),
    creada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (singleton) REFERENCES vec_administracion.configuracion_correo(singleton) DEFERRABLE INITIALLY DEFERRED
);
ALTER TABLE vec_administracion.sobre_configuracion_correo OWNER TO vec_administracion_propietario;

CREATE TABLE vec_administracion.auditoria_configuracion_correo (
    secuencia bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version_configuracion bigint NOT NULL CHECK (version_configuracion > 0),
    consumo_ref text NOT NULL UNIQUE,
    auditoria jsonb NOT NULL,
    registrada_en timestamptz NOT NULL DEFAULT clock_timestamp()
);
ALTER TABLE vec_administracion.auditoria_configuracion_correo OWNER TO vec_administracion_propietario;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_administracion FROM PUBLIC, vec_administracion_ejecutor, vec_administracion_migrador;

CREATE FUNCTION vec_administracion.leer_configuracion_correo_v1()
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT coalesce((SELECT jsonb_build_object(
        'configurada', true, 'host', c.host, 'puerto', c.puerto,
        'server_name', c.nombre_servidor, 'referencia_ca', c.referencia_ca,
        'remitente_fijo', c.remitente_fijo, 'usuario', c.usuario,
        'modo_tls', c.modo_tls, 'modo_autenticacion', c.modo_autenticacion,
        'tiempo_maximo_ms', c.tiempo_maximo_ms, 'secreto_configurado', (c.modo_autenticacion <> 'ninguna' AND c.version_secreto IS NOT NULL),
        'version', c.version
    ) FROM vec_administracion.configuracion_correo c WHERE c.singleton), '{"configurada":false}'::jsonb)
$funcion$;
ALTER FUNCTION vec_administracion.leer_configuracion_correo_v1() OWNER TO vec_administracion_propietario;
REVOKE ALL ON FUNCTION vec_administracion.leer_configuracion_correo_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion.leer_configuracion_correo_v1() TO vec_administracion_ejecutor;

CREATE FUNCTION vec_administracion.sobre_configuracion_correo_actual_v1()
RETURNS TABLE(version_secreto bigint, clave_ref text, nonce bytea, secreto_cifrado bytea)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog
AS $funcion$
    SELECT version_secreto, clave_ref, nonce, secreto_cifrado
      FROM vec_administracion.sobre_configuracion_correo
     WHERE singleton
$funcion$;
ALTER FUNCTION vec_administracion.sobre_configuracion_correo_actual_v1() OWNER TO vec_administracion_propietario;
REVOKE ALL ON FUNCTION vec_administracion.sobre_configuracion_correo_actual_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion.sobre_configuracion_correo_actual_v1() TO vec_administracion_ejecutor;

CREATE FUNCTION vec_administracion.guardar_configuracion_correo_v1(
    p_negocio bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre_cose bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s' SET row_security = on
AS $funcion$
DECLARE v_consumo record; v_actual vec_administracion.configuracion_correo%ROWTYPE;
        v_version_nueva bigint; v_secreto_version bigint; v_huella_aad text;
        v jsonb; c jsonb; s jsonb; a jsonb; p_host text; p_puerto integer; p_nombre_servidor text; p_referencia_ca text; p_remitente_fijo text; p_usuario text; p_modo_tls text; p_modo_autenticacion text; p_tiempo_maximo_ms bigint; p_version_esperada bigint;
BEGIN
    v := convert_from(p_negocio,'UTF8')::jsonb; c := v->'configuracion'; s := NULLIF(v->'sobre_secreto','null'::jsonb); a := v->'auditoria';
    p_host:=c->>'host'; p_puerto:=(c->>'puerto')::integer; p_nombre_servidor:=c->>'server_name'; p_referencia_ca:=c->>'referencia_ca'; p_remitente_fijo:=c->>'remitente_fijo'; p_usuario:=c->>'usuario'; p_modo_tls:=c->>'modo_tls'; p_modo_autenticacion:=c->>'modo_autenticacion'; p_tiempo_maximo_ms:=(c->>'tiempo_maximo_ms')::bigint; p_version_esperada:=(c->>'version_esperada')::bigint;
    IF jsonb_typeof(v) IS DISTINCT FROM 'object' OR v->>'esquema' IS DISTINCT FROM 'vec.administracion.configuracion-correo.persistencia.v1' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(a) IS DISTINCT FROM 'object' OR p_host IS NULL OR p_puerto NOT BETWEEN 1 AND 65535 OR p_nombre_servidor IS NULL OR p_referencia_ca IS NULL OR p_remitente_fijo IS NULL OR p_usuario IS NULL OR p_modo_tls NOT IN ('tls_implicito','starttls_obligatorio') OR p_modo_autenticacion NOT IN ('ninguna','plain','xoauth2') OR p_tiempo_maximo_ms NOT BETWEEN 1 AND 3600000 OR p_version_esperada IS NULL OR p_version_esperada < 0 OR a->>'subject_ref' IS DISTINCT FROM 'configuracion:smtp:diputacion' OR a->>'action' IS DISTINCT FROM 'administracion.configuracion_correo.actualizar' OR a->>'module_id' IS DISTINCT FROM 'vec.module.administracion' OR a->>'result' IS DISTINCT FROM 'accepted' OR nullif(a->>'actor_id','') IS NULL OR nullif(a->>'occurred_at','') IS NULL THEN
        RAISE EXCEPTION 'configuracion administrativa invalida' USING ERRCODE = '22023';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre_cose,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'consumo no nuevo' USING ERRCODE='P0409'; END IF;
    IF v_consumo.efecto_ref IS DISTINCT FROM 'configuracion:smtp:diputacion'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM encode(sha256(convert_to(
          '{"ambitos":{},"atributos":{"material_sha256":"'||encode(sha256(p_negocio),'hex')||'"}}','UTF8')),'hex') THEN
        RAISE EXCEPTION 'contexto autorizado no ligado al payload' USING ERRCODE='42501';
    END IF;
    SELECT * INTO v_actual FROM vec_administracion.configuracion_correo WHERE singleton FOR UPDATE;
    IF NOT FOUND THEN
        -- El contrato actual usa la primera versión positiva como alta inicial.
        IF p_version_esperada <> 0 OR (p_modo_autenticacion <> 'ninguna' AND s IS NULL) THEN RAISE EXCEPTION 'version inicial en conflicto' USING ERRCODE='P0409'; END IF;
        v_version_nueva := 1; v_secreto_version := CASE WHEN s IS NULL THEN NULL ELSE (s->>'version')::bigint END;
    ELSE
        IF v_actual.version <> p_version_esperada THEN RAISE EXCEPTION 'version en conflicto' USING ERRCODE='P0409'; END IF;
        v_version_nueva := v_actual.version + 1; v_secreto_version := coalesce((s->>'version')::bigint, v_actual.version_secreto);
    END IF;
    IF p_modo_autenticacion <> 'ninguna' AND v_secreto_version IS NULL THEN RAISE EXCEPTION 'falta secreto protegido' USING ERRCODE='22023'; END IF;
    IF s IS NOT NULL AND (jsonb_typeof(s) <> 'object' OR (s->>'version')::bigint <> v_secreto_version OR v_secreto_version <> p_version_esperada + 1 OR s->>'clave_ref' IS NULL OR s->>'nonce' IS NULL OR s->>'cifrado' IS NULL OR s->>'huella_aad_sha256' !~ '^[0-9a-f]{64}$') THEN RAISE EXCEPTION 'sobre de secreto invalido' USING ERRCODE='22023'; END IF;
    v_huella_aad := CASE WHEN s IS NULL THEN NULL ELSE s->>'huella_aad_sha256' END;
    IF s IS NOT NULL AND v_huella_aad IS DISTINCT FROM encode(sha256(convert_to('{"Esquema":"vec.administracion.configuracion-correo.secreto.v1","Referencia":"configuracion:smtp:diputacion","Version":'||v_secreto_version||'}','UTF8')),'hex') THEN RAISE EXCEPTION 'AAD de secreto no ligada a versión' USING ERRCODE='22023'; END IF;
    INSERT INTO vec_administracion.configuracion_correo(singleton,version,host,puerto,nombre_servidor,referencia_ca,remitente_fijo,usuario,modo_tls,modo_autenticacion,tiempo_maximo_ms,version_secreto) VALUES(true,v_version_nueva,p_host,p_puerto,p_nombre_servidor,p_referencia_ca,p_remitente_fijo,p_usuario,p_modo_tls,p_modo_autenticacion,p_tiempo_maximo_ms,v_secreto_version) ON CONFLICT(singleton) DO UPDATE SET version=excluded.version,host=excluded.host,puerto=excluded.puerto,nombre_servidor=excluded.nombre_servidor,referencia_ca=excluded.referencia_ca,remitente_fijo=excluded.remitente_fijo,usuario=excluded.usuario,modo_tls=excluded.modo_tls,modo_autenticacion=excluded.modo_autenticacion,tiempo_maximo_ms=excluded.tiempo_maximo_ms,version_secreto=excluded.version_secreto,actualizada_en=clock_timestamp();
    IF s IS NOT NULL THEN INSERT INTO vec_administracion.sobre_configuracion_correo(singleton,version_secreto,clave_ref,nonce,secreto_cifrado,huella_aad_sha256) VALUES(true,v_secreto_version,s->>'clave_ref',decode(s->>'nonce','base64'),decode(s->>'cifrado','base64'),v_huella_aad) ON CONFLICT(singleton) DO UPDATE SET version_secreto=excluded.version_secreto,clave_ref=excluded.clave_ref,nonce=excluded.nonce,secreto_cifrado=excluded.secreto_cifrado,huella_aad_sha256=excluded.huella_aad_sha256,creada_en=clock_timestamp(); END IF;
    INSERT INTO vec_administracion.auditoria_configuracion_correo(version_configuracion,consumo_ref,auditoria) VALUES(v_version_nueva,v_consumo.auditoria_ref,a);
    RETURN jsonb_build_object('configurada',true,'host',p_host,'puerto',p_puerto,'server_name',p_nombre_servidor,'referencia_ca',p_referencia_ca,'remitente_fijo',p_remitente_fijo,'usuario',p_usuario,'modo_tls',p_modo_tls,'modo_autenticacion',p_modo_autenticacion,'tiempo_maximo_ms',p_tiempo_maximo_ms,'secreto_configurado',(p_modo_autenticacion <> 'ninguna' AND v_secreto_version IS NOT NULL),'version',v_version_nueva);
END
$funcion$;
ALTER FUNCTION vec_administracion.guardar_configuracion_correo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_administracion_propietario;
REVOKE ALL ON FUNCTION vec_administracion.guardar_configuracion_correo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion.guardar_configuracion_correo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_administracion_ejecutor;
COMMIT;
