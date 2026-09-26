\set ON_ERROR_STOP on
-- CT128 DOWN: solo sin historia. Si alguna propuesta sustituye a otra (o un
-- expediente tiene más de una propuesta), la reversión se niega: la historia
-- es de solo adición. Restaura exactamente las tres funciones ampliadas y la
-- unicidad original de la propuesta.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000128',0));
LOCK TABLE vec_contratacion_temporal.propuesta_formalizacion IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regclass('vec_contratacion_temporal.propuesta_sustitucion_v1') IS NULL THEN
        RAISE EXCEPTION 'CT128 DOWN: CT128 no está instalada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion
                   GROUP BY organizacion_ref,expediente_ref HAVING count(*)>1) THEN
        RAISE EXCEPTION 'CT128 DOWN: no admitido con historia de sustituciones' USING ERRCODE='55000';
    END IF;
END
$pre$;

DO $fragmentos$
DECLARE
    v_antes record; v_despues record; v_definicion text; i integer;
    v_firmas text[]:=ARRAY[
        'registrar_propuesta_formalizacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
        'registrar_propuesta_formalizacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
        'registrar_propuesta_formalizacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
        'registrar_propuesta_formalizacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
        'preparar_no_incorporacion_v1(jsonb)',
        'leer_contratos_bolsa_v1(bigint,text,integer)'];
    v_viejos text[]:=ARRAY[
$v1$       AND continuacion.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada')$v1$,
$v2$       AND e.solicitud_json->'version_expediente'=to_jsonb(v_n)
       AND e.recibo_json->'version_expediente'=to_jsonb(v_n)$v2$,
$v3$    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion
        WHERE resolucion_ref=v_resolucion.resolucion_ref OR
            (organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef')) THEN$v3$,
$v4$    v_resultado:=jsonb_build_object('Solicitud',s,'PropuestaRef',v_propuesta,'ReciboLocalRef',v_recibo,$v4$,
$v5$                  AND n.propuesta_ref=(SELECT a.propuesta_ref FROM vec_contratacion_temporal.propuesta_formalizacion a
                                        WHERE a.organizacion_ref=m->>'organizacion_ref' AND a.expediente_ref=m->>'expediente_ref')) THEN$v5$,
$v6$          JOIN vec_contratacion_temporal.propuesta_formalizacion p
            ON p.organizacion_ref = r.organizacion_ref AND p.expediente_ref = r.expediente_ref$v6$];
    v_nuevos text[]:=ARRAY[
$n1$       AND continuacion.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')$n1$,
$n2$       AND ((e.solicitud_json->'version_expediente'=to_jsonb(v_n)
       AND e.recibo_json->'version_expediente'=to_jsonb(v_n))
        -- CT128: tras una no incorporación la selección es la original y el
        -- expediente sigue en una versión posterior a esa no incorporación.
        OR (continuacion.solicitud_json->>'Respuesta'='aceptacion'
            AND e.recibo_json->'version_expediente'=e.solicitud_json->'version_expediente'
            AND EXISTS (SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n128
                         WHERE n128.aceptacion_resolucion_ref=continuacion.resolucion_ref
                           AND n128.intencion_ref=continuacion.continuacion_recibo->'Solicitud'->>'IntencionRef'
                           AND n128.organizacion_ref=j.organizacion_ref AND n128.expediente_ref=j.expediente_ref
                           AND n128.version_esperada+1<=v_n)))$n2$,
$n3$    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion
        WHERE resolucion_ref=v_resolucion.resolucion_ref OR
            (organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef'
             -- CT128: no cuentan las ya sustituidas ni la que sustituye esta propuesta.
             AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x128
                              WHERE x128.propuesta_anterior_ref=propuesta_formalizacion.propuesta_ref)
             AND propuesta_ref IS DISTINCT FROM (SELECT p128.propuesta_ref FROM vec_contratacion_temporal.propuesta_sustituible_ct128(
                 v_continuacion,s->>'OrganizacionRef',s->>'ExpedienteRef',v_n) p128))) THEN$n3$,
$n4$    -- CT128: la propuesta anterior queda sustituida por su no incorporación.
    INSERT INTO vec_contratacion_temporal.propuesta_sustitucion_v1(
        propuesta_ref,propuesta_anterior_ref,no_incorporacion_recibo_ref,organizacion_ref,expediente_ref,ronda,registrada_en)
    SELECT v_propuesta,p128.propuesta_ref,p128.no_incorporacion_recibo_ref,s->>'OrganizacionRef',s->>'ExpedienteRef',p128.ronda,v_ahora
      FROM vec_contratacion_temporal.propuesta_sustituible_ct128(v_continuacion,s->>'OrganizacionRef',s->>'ExpedienteRef',v_n) p128;
    v_resultado:=jsonb_build_object('Solicitud',s,'PropuestaRef',v_propuesta,'ReciboLocalRef',v_recibo,$n4$,
$n5$                  AND n.propuesta_ref=(SELECT a.propuesta_ref FROM vec_contratacion_temporal.propuesta_formalizacion a
                                        WHERE a.organizacion_ref=m->>'organizacion_ref' AND a.expediente_ref=m->>'expediente_ref'
                                          -- CT128: la propuesta vigente.
                                          AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x128
                                                           WHERE x128.propuesta_anterior_ref=a.propuesta_ref))) THEN$n5$,
$n6$          JOIN vec_contratacion_temporal.propuesta_formalizacion p
            ON p.organizacion_ref = r.organizacion_ref AND p.expediente_ref = r.expediente_ref
           -- CT128: la propuesta vigente en la versión de la incorporación.
           AND p.version_resultante = (SELECT max(p128.version_resultante) FROM vec_contratacion_temporal.propuesta_formalizacion p128
                                        WHERE p128.organizacion_ref = r.organizacion_ref AND p128.expediente_ref = r.expediente_ref
                                          AND p128.version_resultante <= r.version_expediente)$n6$];
BEGIN
    FOR i IN REVERSE array_length(v_firmas,1)..1 LOOP
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO v_antes FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_firmas[i])
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'CT128 DOWN: función ausente: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
        v_definicion:=v_antes.definicion;
        IF length(v_definicion)-length(replace(v_definicion,v_nuevos[i],''))<>length(v_nuevos[i]) THEN
            RAISE EXCEPTION 'CT128 DOWN: ampliación no localizada: % (%)',v_firmas[i],i USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_nuevos[i],v_viejos[i]);
        EXECUTE v_definicion;
        SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
        IF v_despues.definicion IS DISTINCT FROM v_definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'CT128 DOWN: definición o permisos alterados: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                WHERE n.nspname='vec_contratacion_temporal' AND strpos(p.prosrc,'_ct128')<>0
                  AND p.proname NOT IN ('validar_sustitucion_ct128','propuesta_vigente_unica_ct128','propuesta_sustituible_ct128')) THEN
        RAISE EXCEPTION 'CT128 DOWN: quedan funciones ampliadas' USING ERRCODE='55000';
    END IF;
END
$fragmentos$;

DROP FUNCTION vec_contratacion_temporal.consultar_propuestas_expediente_v1(text,text);
DROP TRIGGER propuesta_vigente_unica_ct128 ON vec_contratacion_temporal.propuesta_formalizacion;
DROP INDEX vec_contratacion_temporal.propuesta_formalizacion_expediente_ct128;
ALTER TABLE vec_contratacion_temporal.propuesta_formalizacion
    ADD CONSTRAINT propuesta_formalizacion_organizacion_ref_expediente_ref_key UNIQUE (organizacion_ref,expediente_ref);
DROP TABLE vec_contratacion_temporal.propuesta_sustitucion_v1;
DROP FUNCTION vec_contratacion_temporal.propuesta_sustituible_ct128(jsonb,text,text,numeric);
DROP FUNCTION vec_contratacion_temporal.propuesta_vigente_unica_ct128();
DROP FUNCTION vec_contratacion_temporal.validar_sustitucion_ct128();
COMMIT;
