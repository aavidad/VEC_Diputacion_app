\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000042',0));

-- Esta migración necesita el administrador de roles y ambos propietarios.
-- No concede capacidad: los nuevos grupos sólo informan resultados mínimos.
DO $precondiciones$
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regclass('vec_autorizacion_atestada_v3.resultado_ejecucion_v1') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN (
           'vec_resultado_rutas_dietas_registro','vec_resultado_borrador_dietas_registro','vec_resultado_marcaje_cronos_registro')) THEN
        RAISE EXCEPTION 'preimagen incompatible AD3-42' USING ERRCODE='55000';
    END IF;
END
$precondiciones$;
CREATE ROLE vec_resultado_rutas_dietas_registro NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS NOINHERIT;
CREATE ROLE vec_resultado_borrador_dietas_registro NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS NOINHERIT;
CREATE ROLE vec_resultado_marcaje_cronos_registro NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS NOINHERIT;
DO $conexion$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_resultado_rutas_dietas_registro,vec_resultado_borrador_dietas_registro,vec_resultado_marcaje_cronos_registro',current_database());
END
$conexion$;

-- La autoridad positiva conserva sus tablas privadas. El único lector nuevo
-- devuelve un booleano nominal: no exporta decisiones, documentos ni identidad.
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.concesion_resultado_ejecucion_v1(
    p_consumidor text,p_decision text,p_huella_decision text,p_contexto text,
    p_huella_contexto text,p_actor text,p_perfil text,p_correlacion text,
    p_accion text,p_recurso text,p_huella_recurso text
) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog
AS $funcion$
    SELECT EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_concedida_contexto_actor_v3 d
        WHERE d.decision_ref=p_decision AND d.huella_decision_sha256=p_huella_decision
          AND d.registro_contexto_ref=p_contexto AND d.contexto_actor_huella_sha256=p_huella_contexto
          AND d.documento->>'principal_id'=p_actor AND d.documento->>'perfil_activo_ref'=p_perfil
          AND d.documento->>'correlacion_ref'=p_correlacion AND d.documento->>'accion'=p_accion
          AND d.documento->>'recurso_ref'=p_recurso AND d.documento->>'contexto_recurso_huella_sha256'=p_huella_recurso
          AND d.documento->'concedida'='true'::jsonb
          AND d.documento#>>'{vinculo_autenticacion_actor,superficie}'='interna_corporativa'
          AND d.documento->'campos_permitidos'='[]'::jsonb AND d.documento->'obligaciones'='[]'::jsonb
          AND (
            (p_consumidor='dietas_rutas' AND d.documento->>'modulo_id'='dietas'
             AND d.documento->>'finalidad'='consultar_itinerario_dietas'
             AND ((p_accion='dietas.ruta.catalogo.consultar' AND d.documento->>'tipo_recurso'='catalogo_rutas_dietas' AND p_recurso='dietas:rutas:catalogo_rutas_dietas')
               OR (p_accion='dietas.ruta.calculo.solicitar' AND d.documento->>'tipo_recurso'='calculo_rutas_dietas' AND p_recurso='dietas:rutas:calculo_rutas_dietas')))
            OR (p_consumidor='dietas_borrador' AND d.documento->>'modulo_id'='dietas'
             AND d.documento->>'tipo_recurso'='borrador_comision' AND d.documento->>'finalidad'='gestionar_borrador_propio'
             AND ((p_accion IN ('dietas.borrador.crear_propio','dietas.borrador.recuperar_propio') AND p_recurso ~ '^dietas:borrador:[A-Za-z0-9_-]{16,128}$')
               OR (p_accion='dietas.borrador.listar_propios' AND p_recurso='dietas:borradores:propios')))
            OR (p_consumidor='cronos_marcaje' AND d.documento->>'modulo_id'='cronos'
             AND d.documento->>'tipo_recurso'='marcaje_propio' AND d.documento->>'finalidad'='registrar_marcaje_propio'
             AND p_accion='cronos.marcaje.propio.registrar' AND p_recurso ~ '^marcaje:cronos:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
          )
    )
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion.concesion_resultado_ejecucion_v1(text,text,text,text,text,text,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.concesion_resultado_ejecucion_v1(text,text,text,text,text,text,text,text,text,text,text) TO vec_autorizacion_atestada_v3_propietario;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.resultado_ejecucion_v1 (
    informe_ref text PRIMARY KEY CHECK (informe_ref ~ '^inf_ejec_[0-9a-f]{32}$'),
    perfil_consumidor text NOT NULL CHECK (perfil_consumidor IN ('dietas_rutas','dietas_borrador','cronos_marcaje')),
    decision_ref text NOT NULL REFERENCES vec_autorizacion.decision_concedida_contexto_actor_v3(decision_ref),
    decision_huella_sha256 text NOT NULL CHECK (decision_huella_sha256 ~ '^[0-9a-f]{64}$' AND decision_huella_sha256<>repeat('0',64)),
    contexto_ref text NOT NULL CHECK (contexto_ref ~ '^rca_[A-Za-z0-9_-]{22,128}$'),
    contexto_huella_sha256 text NOT NULL CHECK (contexto_huella_sha256 ~ '^[0-9a-f]{64}$' AND contexto_huella_sha256<>repeat('0',64)),
    actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
    perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[A-Za-z0-9_-]{22,128}$'),
    correlacion_ref text NOT NULL CHECK (correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'),
    accion text NOT NULL,
    recurso_ref text NOT NULL,
    recurso_huella_sha256 text NOT NULL CHECK (recurso_huella_sha256 ~ '^[0-9a-f]{64}$' AND recurso_huella_sha256<>repeat('0',64)),
    resultado text NOT NULL CHECK (resultado IN ('fallo_confirmado','resultado_indeterminado')),
    etapa text NOT NULL,
    causa text NOT NULL,
    contenido_huella_sha256 text NOT NULL CHECK (contenido_huella_sha256 ~ '^[0-9a-f]{64}$'),
    recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^rec_ejec_[0-9a-f]{32}$'),
    secuencia numeric(20,0) NOT NULL UNIQUE CHECK (secuencia BETWEEN 1 AND 9007199254740991),
    anterior_sha256 text NOT NULL CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
    huella_sha256 text NOT NULL UNIQUE CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    observado_en timestamptz(6) NOT NULL,
    CHECK (
        (etapa='preparacion' AND resultado='fallo_confirmado' AND causa IN ('material_no_disponible','material_invalido','contexto_cancelado'))
        OR (etapa='transaccion' AND resultado='fallo_confirmado' AND causa IN ('persistencia','conflicto','recibo_invalido','contexto_cancelado','consumo_rechazado'))
        OR (etapa='rollback' AND resultado='resultado_indeterminado' AND causa='rollback_no_confirmado')
        OR (etapa='commit' AND ((resultado='fallo_confirmado' AND causa='commit_revertido') OR (resultado='resultado_indeterminado' AND causa='commit_no_confirmado')))
        OR (etapa='entrega' AND resultado='resultado_indeterminado' AND causa IN ('respuesta_no_confirmada','recibo_invalido'))
    ),
    CHECK (
        (perfil_consumidor='dietas_rutas' AND ((accion='dietas.ruta.catalogo.consultar' AND recurso_ref='dietas:rutas:catalogo_rutas_dietas') OR (accion='dietas.ruta.calculo.solicitar' AND recurso_ref='dietas:rutas:calculo_rutas_dietas')))
        OR (perfil_consumidor='dietas_borrador' AND ((accion IN ('dietas.borrador.crear_propio','dietas.borrador.recuperar_propio') AND recurso_ref ~ '^dietas:borrador:[A-Za-z0-9_-]{16,128}$') OR (accion='dietas.borrador.listar_propios' AND recurso_ref='dietas:borradores:propios')))
        OR (perfil_consumidor='cronos_marcaje' AND accion='cronos.marcaje.propio.registrar' AND recurso_ref ~ '^marcaje:cronos:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$')
    )
);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.resultado_ejecucion_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.resultado_ejecucion_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
ALTER TABLE vec_autorizacion_atestada_v3.resultado_ejecucion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.resultado_ejecucion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion_atestada_v3.resultado_ejecucion_v1 TO vec_autorizacion_atestada_v3_propietario
    USING (current_user='vec_autorizacion_atestada_v3_propietario') WITH CHECK (current_user='vec_autorizacion_atestada_v3_propietario');
REVOKE ALL ON vec_autorizacion_atestada_v3.resultado_ejecucion_v1 FROM PUBLIC,vec_resultado_rutas_dietas_registro,vec_resultado_borrador_dietas_registro,vec_resultado_marcaje_cronos_registro;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_resultado_ejecucion_interno_v1(
    p_consumidor text,p_informe text,p_decision text,p_huella_decision text,p_contexto text,
    p_huella_contexto text,p_actor text,p_perfil text,p_correlacion text,
    p_accion text,p_recurso text,p_huella_recurso text,p_resultado text,p_etapa text,p_causa text
) RETURNS TABLE (informe_ref text,recibo_ref text,huella_sha256 text,observado_en timestamptz,repeticion boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s'
AS $funcion$
DECLARE
    v_rol text;
    v_contenido bytea;
    v_huella text;
    v_existente vec_autorizacion_atestada_v3.resultado_ejecucion_v1%ROWTYPE;
    v_secuencia numeric;
    v_anterior text;
    v_auditoria text;
    v_recibo text;
    v_ahora timestamptz;
BEGIN
    v_rol:=CASE p_consumidor WHEN 'dietas_rutas' THEN 'vec_resultado_rutas_dietas_registro'
        WHEN 'dietas_borrador' THEN 'vec_resultado_borrador_dietas_registro'
        WHEN 'cronos_marcaje' THEN 'vec_resultado_marcaje_cronos_registro' END;
    IF v_rol IS NULL OR current_user<>'vec_autorizacion_atestada_v3_propietario' OR session_user=current_user
       OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=v_rol AND NOT r.rolcanlogin
           AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls AND NOT r.rolinherit)
       OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
           AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR (SELECT count(*) FROM pg_auth_members a WHERE a.member=session_user::regrole)<>1
       OR NOT EXISTS (SELECT 1 FROM pg_auth_members a WHERE a.member=session_user::regrole
           AND a.roleid=to_regrole(v_rol) AND a.inherit_option AND NOT a.set_option AND NOT a.admin_option)
       OR EXISTS (SELECT 1 FROM pg_roles r WHERE r.oid<>session_user::regrole AND r.oid<>to_regrole(v_rol)
           AND pg_has_role(session_user,r.oid,'MEMBER'))
       OR EXISTS (SELECT 1 FROM pg_auth_members a WHERE a.member=to_regrole(v_rol)) THEN
        RAISE EXCEPTION 'registro resultado no permitido' USING ERRCODE='42501';
    END IF;
    IF p_informe IS NULL OR p_informe !~ '^inf_ejec_[0-9a-f]{32}$'
       OR p_decision IS NULL OR p_decision !~ '^(decision:[0-9a-f]{32}|dec_[A-Za-z0-9_-]{16,128})$'
       OR p_contexto IS NULL OR p_contexto !~ '^rca_[A-Za-z0-9_-]{22,128}$'
       OR p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
       OR p_perfil IS NULL OR p_perfil !~ '^prf_[A-Za-z0-9_-]{22,128}$'
       OR p_correlacion IS NULL OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$'
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(p_huella_decision) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(p_huella_contexto) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(p_huella_recurso) IS NOT TRUE
       OR NOT (coalesce(p_etapa='preparacion' AND p_resultado='fallo_confirmado' AND p_causa IN ('material_no_disponible','material_invalido','contexto_cancelado'),false)
         OR coalesce(p_etapa='transaccion' AND p_resultado='fallo_confirmado' AND p_causa IN ('persistencia','conflicto','recibo_invalido','contexto_cancelado','consumo_rechazado'),false)
         OR coalesce(p_etapa='rollback' AND p_resultado='resultado_indeterminado' AND p_causa='rollback_no_confirmado',false)
         OR coalesce(p_etapa='commit' AND ((p_resultado='fallo_confirmado' AND p_causa='commit_revertido') OR (p_resultado='resultado_indeterminado' AND p_causa='commit_no_confirmado')),false)
         OR coalesce(p_etapa='entrega' AND p_resultado='resultado_indeterminado' AND p_causa IN ('respuesta_no_confirmada','recibo_invalido'),false))
       OR vec_autorizacion.concesion_resultado_ejecucion_v1(p_consumidor,p_decision,p_huella_decision,p_contexto,p_huella_contexto,p_actor,p_perfil,p_correlacion,p_accion,p_recurso,p_huella_recurso) IS NOT TRUE THEN
        RAISE EXCEPTION 'informe resultado invalido' USING ERRCODE='42501';
    END IF;
    -- Validamos el enlace histórico positivo, no su vigencia actual: caducidad o
    -- revocación pueden ser precisamente el motivo del fallo observado.
    v_contenido:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.resultado_ejecucion.v1')
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_informe)||vec_autorizacion_atestada_v3.encuadrar_mac(p_consumidor)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_decision)||vec_autorizacion_atestada_v3.encuadrar_mac(p_huella_decision)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_contexto)||vec_autorizacion_atestada_v3.encuadrar_mac(p_huella_contexto)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_actor)||vec_autorizacion_atestada_v3.encuadrar_mac(p_perfil)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_correlacion)||vec_autorizacion_atestada_v3.encuadrar_mac(p_accion)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_recurso)||vec_autorizacion_atestada_v3.encuadrar_mac(p_huella_recurso)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_resultado)||vec_autorizacion_atestada_v3.encuadrar_mac(p_etapa)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(p_causa);
    v_huella:=encode(sha256(v_contenido),'hex');
    SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
        FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
    SELECT r.* INTO v_existente FROM vec_autorizacion_atestada_v3.resultado_ejecucion_v1 r WHERE r.informe_ref=p_informe;
    IF FOUND THEN
        IF v_existente.contenido_huella_sha256 IS DISTINCT FROM v_huella THEN
            RAISE EXCEPTION 'referencia informe en conflicto' USING ERRCODE='P4201';
        END IF;
        RETURN QUERY SELECT v_existente.informe_ref,v_existente.recibo_ref,v_existente.huella_sha256,v_existente.observado_en,true;
        RETURN;
    END IF;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'limite auditoria AD3' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    v_ahora:=clock_timestamp();
    v_recibo:='rec_ejec_'||substr(v_huella,1,32);
    v_auditoria:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.resultado_ejecucion.auditoria.v1')
        ||vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(v_huella)||vec_autorizacion_atestada_v3.encuadrar_mac(v_recibo)
        ||vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
    INSERT INTO vec_autorizacion_atestada_v3.resultado_ejecucion_v1 VALUES (
        p_informe,p_consumidor,p_decision,p_huella_decision,p_contexto,p_huella_contexto,p_actor,p_perfil,p_correlacion,
        p_accion,p_recurso,p_huella_recurso,p_resultado,p_etapa,p_causa,v_huella,v_recibo,v_secuencia,v_anterior,v_auditoria,v_ahora);
    UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
        SET secuencia=v_secuencia,cabeza_sha256=v_auditoria,actualizada_en=v_ahora WHERE control_id;
    RETURN QUERY SELECT p_informe,v_recibo,v_auditoria,v_ahora,false;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_resultado_ejecucion_interno_v1(text,text,text,text,text,text,text,text,text,text,text,text,text,text,text)
    FROM PUBLIC,vec_resultado_rutas_dietas_registro,vec_resultado_borrador_dietas_registro,vec_resultado_marcaje_cronos_registro;

-- Tres envoltorios nominales, de firma idéntica. No existe selector de módulo
-- en la API SQL pública del runtime; el grupo propio tampoco ejecuta los otros.
DO $envoltorios$
DECLARE x record;
BEGIN
    FOR x IN SELECT * FROM (VALUES
        ('rutas_dietas','dietas_rutas','vec_resultado_rutas_dietas_registro'),
        ('borrador_dietas','dietas_borrador','vec_resultado_borrador_dietas_registro'),
        ('marcaje_cronos','cronos_marcaje','vec_resultado_marcaje_cronos_registro')
    ) v(nombre,perfil,rol) LOOP
        EXECUTE format($ddl$
            CREATE FUNCTION vec_autorizacion_atestada_v3.%I(
                p_informe text,p_decision text,p_huella_decision text,p_contexto text,p_huella_contexto text,
                p_actor text,p_perfil text,p_correlacion text,p_accion text,p_recurso text,p_huella_recurso text,
                p_resultado text,p_etapa text,p_causa text
            ) RETURNS TABLE (informe_ref text,recibo_ref text,huella_sha256 text,observado_en timestamptz,repeticion boolean)
            LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
            AS $envoltura$SELECT * FROM vec_autorizacion_atestada_v3.registrar_resultado_ejecucion_interno_v1(%L,p_informe,p_decision,p_huella_decision,p_contexto,p_huella_contexto,p_actor,p_perfil,p_correlacion,p_accion,p_recurso,p_huella_recurso,p_resultado,p_etapa,p_causa)$envoltura$
        $ddl$,'registrar_resultado_'||x.nombre||'_v1',x.perfil);
        EXECUTE format('REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.%I(text,text,text,text,text,text,text,text,text,text,text,text,text,text) FROM PUBLIC','registrar_resultado_'||x.nombre||'_v1');
        EXECUTE format('GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.%I(text,text,text,text,text,text,text,text,text,text,text,text,text,text) TO %I','registrar_resultado_'||x.nombre||'_v1',x.rol);
        EXECUTE format('GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO %I',x.rol);
    END LOOP;
END
$envoltorios$;
COMMIT;
