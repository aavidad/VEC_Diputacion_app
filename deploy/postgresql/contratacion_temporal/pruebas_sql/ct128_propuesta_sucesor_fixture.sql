\set ON_ERROR_STOP on
-- Fixture de CT128 sobre la estructura real restaurada, después de la cadena
-- de CT124 (ct115_ct116_fixture_pg18.sql, ct124_no_incorporacion_fixture.sql
-- y ct124_no_incorporacion_cadena.sql): el expediente A ya tiene su no
-- incorporación, la baja en Bolsa, el siguiente llamamiento abierto y la
-- continuación confirmada. Dobles explícitos de las fachadas AD3 que consume
-- el circuito del sucesor y el resto del camino (aviso, respuesta,
-- justificante, resolución, propuesta y GINPIX): prueban las transacciones
-- CT, no la criptografía V3. Base desechable: se confirma.
DO $dobles$
DECLARE v_nombre text;
BEGIN
    FOREACH v_nombre IN ARRAY ARRAY[
        'registrar_y_consumir_comunicacion_llamamiento_v3_atestada',
        'registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada',
        'registrar_y_consumir_justificante_respuesta_ct_v3_atestada',
        'registrar_y_consumir_resolucion_manual_ct_v3_atestada',
        'registrar_y_consumir_propuesta_formalizacion_ct_v3_atestada'
    ] LOOP
        IF to_regprocedure('vec_autorizacion_atestada_v3.'||v_nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
            RAISE EXCEPTION 'fachada AD3 ausente: %', v_nombre;
        END IF;
        EXECUTE format($f$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.%I(
            p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
            p_persona_version numeric,p_perfil_version numeric,
            p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
        ) RETURNS TABLE (
            decision_ref text,efecto_ref text,huella_efecto_sha256 text,
            consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
        ) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog
        AS $c$
        DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
            h text := encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex');
        BEGIN
            -- Doble de prueba: eco del efecto de la decisión y consumo único.
            RETURN QUERY SELECT 'decision:'||substr(h,1,32), d->>'recurso_ref',
                d->>'contexto_recurso_huella_sha256', h, 'aud_v3_'||substr(h,33,32), clock_timestamp(), true;
        END
        $c$$f$, v_nombre);
    END LOOP;
END
$dobles$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.ginpix.confirmar' THEN RAISE EXCEPTION 'doble: GINPIX denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;

-- Utilidades persistentes (sobreviven al reinicio) para repetir los pasos
-- con el mismo material: casos, valores y llamadas con la cuenta de ejecución.
CREATE SCHEMA prueba_ct128;
GRANT USAGE ON SCHEMA prueba_ct128 TO vec_ct115_runtime;
CREATE TABLE prueba_ct128.caso (
    caso text PRIMARY KEY, funcion text NOT NULL, accion text NOT NULL, tipo text NOT NULL,
    material text NOT NULL, resultado jsonb NOT NULL);
GRANT SELECT, INSERT ON prueba_ct128.caso TO vec_ct115_runtime;
CREATE TABLE prueba_ct128.valor (clave text PRIMARY KEY, valor text NOT NULL);
GRANT SELECT, INSERT ON prueba_ct128.valor TO vec_ct115_runtime;
CREATE FUNCTION prueba_ct128.v(p text) RETURNS text LANGUAGE sql STABLE SET search_path = pg_catalog AS
$f$ SELECT valor FROM prueba_ct128.valor WHERE clave=p $f$;
CREATE FUNCTION prueba_ct128.r(p_caso text) RETURNS jsonb LANGUAGE sql STABLE SET search_path = pg_catalog AS
$f$ SELECT resultado FROM prueba_ct128.caso WHERE caso=p_caso $f$;
CREATE FUNCTION prueba_ct128.instante(p timestamptz) RETURNS text LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS
$f$ SELECT to_char(p AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') $f$;
-- Decisión del doble AD3 ligada al material exacto, como la emite la frontera.
CREATE FUNCTION prueba_ct128.decision(p_material text, p_accion text, p_tipo text) RETURNS bytea
LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    WITH m AS (SELECT p_material::jsonb AS j)
    SELECT convert_to(jsonb_build_object(
        'accion',p_accion,'modulo_id','contratacion_temporal','tipo_recurso',p_tipo,
        'finalidad','gestionar_contratacion_temporal',
        'recurso_ref',coalesce(j#>>'{Solicitud,ExpedienteRef}',j#>>'{solicitud,ExpedienteRef}',j->>'ExpedienteRef'),
        'contexto_recurso_huella_sha256',encode(sha256(convert_to(
            '{"ambitos":{"organizacion_ref":"'||coalesce(j#>>'{Solicitud,OrganizacionRef}',j#>>'{solicitud,OrganizacionRef}',j->>'OrganizacionRef')||
            '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
        'principal_id','per_ct128_rrhh','perfil_activo_ref','prf_ct128_rrhh')::text,'UTF8')
      FROM m
$f$;
CREATE FUNCTION prueba_ct128.llamar(p_funcion text, p_material text, p_accion text, p_tipo text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v jsonb;
BEGIN
    EXECUTE format('SELECT vec_contratacion_temporal.%I($1,$2,$3,$2,$2,1,1,$2,$2,$2,$2)', p_funcion)
       INTO v USING p_material, '\x01'::bytea, prueba_ct128.decision(p_material,p_accion,p_tipo);
    RETURN v;
END
$f$;
-- Igual que llamar, pero devuelve el SQLSTATE en lugar de fallar.
CREATE FUNCTION prueba_ct128.codigo(p_funcion text, p_material text, p_accion text, p_tipo text) RETURNS text
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
BEGIN
    PERFORM prueba_ct128.llamar(p_funcion,p_material,p_accion,p_tipo);
    RETURN 'ok';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE;
END
$f$;
CREATE FUNCTION prueba_ct128.paso(p_caso text, p_funcion text, p_material text, p_accion text, p_tipo text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v jsonb := prueba_ct128.llamar(p_funcion,p_material,p_accion,p_tipo);
BEGIN
    INSERT INTO prueba_ct128.caso VALUES (p_caso,p_funcion,p_accion,p_tipo,p_material,v);
    RETURN v;
END
$f$;
CREATE FUNCTION prueba_ct128.repetir(p_caso text) RETURNS jsonb
LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $f$
    SELECT prueba_ct128.llamar(funcion,material,accion,tipo) FROM prueba_ct128.caso WHERE caso=p_caso
$f$;
CREATE FUNCTION prueba_ct128.exigir(p_condicion boolean, p_caso text) RETURNS text
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    IF p_condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', p_caso; END IF;
    RETURN 'ok';
END
$f$;
-- Publicaciones de la propuesta, leídas aquí (la cuenta de ejecución no lee la tabla).
INSERT INTO prueba_ct128.valor
SELECT 'publicacion:'||componente, format('{"Referencia":"%s","Version":%s,"HuellaSHA256":"%s"}',referencia,version,huella_sha256)
  FROM vec_contratacion_temporal.publicacion_propuesta_formalizacion_desarrollo;
-- Huella de la historia del expediente A (la calcula el superusuario de la
-- prueba): filas de cada tabla del camino; no debe cambiar al repetir.
CREATE FUNCTION prueba_ct128.conteo(p_exp text) RETURNS text LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    SELECT concat_ws('/',
        (SELECT count(*) FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.actuacion_expediente_integral WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.comunicacion_llamamiento_local WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.respuesta_recibida_rrhh WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.propuesta_formalizacion WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.propuesta_sustitucion_v1 WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_ginpix_v1 WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.cese_nombramiento_v1 WHERE expediente_ref=p_exp),
        (SELECT count(*) FROM vec_contratacion_temporal.cierre_expediente_v1 WHERE expediente_ref=p_exp))
$f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_ct128 TO vec_ct115_runtime;
REVOKE EXECUTE ON FUNCTION prueba_ct128.conteo(text) FROM vec_ct115_runtime;
SELECT 'fixture CT128 OK';
