\set ON_ERROR_STOP on
-- CT128: propuesta de nombramiento del sucesor tras una no incorporación.
-- CT124 registra que la persona aceptada no se incorpora, devuelve el
-- expediente a la fiscalización en curso y deja la intención de siguiente
-- candidato; CT119/CT121/CT124 llevan al sucesor por el aviso, la respuesta,
-- el justificante y la resolución de su aceptación. Faltaba la propuesta: el
-- expediente solo admitía una (UNIQUE (organizacion_ref,expediente_ref)).
--  * La propuesta anterior no se borra ni se modifica (historia de solo
--    adición): queda sustituida por la no incorporación mediante una fila de
--    `propuesta_sustitucion_v1`, que la propuesta nueva escribe en la misma
--    transacción que la versión, la actuación, el outbox y el consumo de la
--    autorización de la propuesta (la misma audiencia V3 de siempre).
--  * La unicidad pasa a ser «una sola propuesta vigente por expediente»: un
--    disparador de restricción diferido exige, para cada propuesta que no
--    sea la primera, su fila de sustitución, y que nunca haya dos propuestas
--    vigentes. Cada propuesta solo puede sustituirse una vez (cadena lineal).
--  * `registrar_propuesta_formalizacion_v2` (CT96, ya ampliada por CT121) no
--    se reescribe: se sustituyen cuatro fragmentos exactos (antecedente de
--    continuación con aceptación, versión de la selección original, propuesta
--    previa sustituible y escritura de la sustitución), conservando
--    propietario, configuración y ACL. v1 (versión 6) no cambia.
--  * Consumidores que suponían una sola propuesta: la preparación de la no
--    incorporación (CT124) toma la propuesta vigente y la publicación de
--    contratos a Bolsa (CT113) liga cada incorporación a la propuesta vigente
--    en su versión, sin duplicar eventos.
--  * Consulta del panel: `consultar_propuestas_expediente_v1` devuelve la
--    propuesta vigente y la historia (sustituidas y su no incorporación),
--    solo con referencias opacas y claves del catálogo.
-- Requiere CT124 (y con ella CT96, CT113, CT119 y CT121). Sin consumidor AD3
-- nuevo. Detección: to_regclass('vec_contratacion_temporal.propuesta_sustitucion_v1').
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
       OR to_regclass('vec_contratacion_temporal.no_incorporacion_v1') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.registrar_propuesta_formalizacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.leer_contratos_bolsa_v1(bigint,text,integer)') IS NULL THEN
        RAISE EXCEPTION 'CT128 requiere CT124, CT96 y CT113' USING ERRCODE='55000';
    END IF;
    IF to_regclass('vec_contratacion_temporal.propuesta_sustitucion_v1') IS NOT NULL THEN
        RAISE EXCEPTION 'CT128 ya instalada: no se reaplica' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint
                    WHERE conrelid='vec_contratacion_temporal.propuesta_formalizacion'::regclass
                      AND conname='propuesta_formalizacion_organizacion_ref_expediente_ref_key' AND contype='u'
                      AND pg_get_constraintdef(oid)='UNIQUE (organizacion_ref, expediente_ref)') THEN
        RAISE EXCEPTION 'CT128: preimagen de la unicidad de la propuesta incompatible' USING ERRCODE='55000';
    END IF;
END
$pre$;

-- ============================================================ SUSTITUCIÓN
-- Una fila por propuesta que sustituye a la anterior tras su no incorporación.
-- Ronda: 2 para la primera sustitución, 3 para la siguiente, etc.
CREATE TABLE vec_contratacion_temporal.propuesta_sustitucion_v1 (
    propuesta_ref text PRIMARY KEY
        REFERENCES vec_contratacion_temporal.propuesta_formalizacion(propuesta_ref) DEFERRABLE INITIALLY DEFERRED,
    propuesta_anterior_ref text NOT NULL UNIQUE
        REFERENCES vec_contratacion_temporal.propuesta_formalizacion(propuesta_ref),
    no_incorporacion_recibo_ref text NOT NULL UNIQUE
        REFERENCES vec_contratacion_temporal.no_incorporacion_v1(recibo_ref),
    organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    expediente_ref text NOT NULL CHECK (expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    ronda integer NOT NULL CHECK (ronda BETWEEN 2 AND 1000),
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
    CHECK (propuesta_ref<>propuesta_anterior_ref),
    UNIQUE (organizacion_ref,expediente_ref,ronda)
);
ALTER TABLE vec_contratacion_temporal.propuesta_sustitucion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.propuesta_sustitucion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_ct128 ON vec_contratacion_temporal.propuesta_sustitucion_v1
    TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.propuesta_sustitucion_v1
    FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

-- Coherencia de cada sustitución: la anterior es la propuesta de la no
-- incorporación indicada, del mismo expediente, y la nueva parte de una
-- versión posterior a esa no incorporación.
CREATE FUNCTION vec_contratacion_temporal.validar_sustitucion_ct128()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n
                     JOIN vec_contratacion_temporal.propuesta_formalizacion a ON a.propuesta_ref=n.propuesta_ref
                    WHERE n.recibo_ref=NEW.no_incorporacion_recibo_ref AND n.propuesta_ref=NEW.propuesta_anterior_ref
                      AND n.organizacion_ref=NEW.organizacion_ref AND n.expediente_ref=NEW.expediente_ref
                      AND a.organizacion_ref=NEW.organizacion_ref AND a.expediente_ref=NEW.expediente_ref) THEN
        RAISE EXCEPTION 'CT128: sustitución sin su no incorporación' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END
$$;
CREATE TRIGGER sustitucion_coherente BEFORE INSERT ON vec_contratacion_temporal.propuesta_sustitucion_v1
    FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.validar_sustitucion_ct128();

-- Una sola propuesta vigente por expediente (sustituye a la UNIQUE retirada).
-- Diferido: la propuesta y su sustitución se escriben en la misma transacción.
CREATE FUNCTION vec_contratacion_temporal.propuesta_vigente_unica_ct128()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion o
                WHERE o.organizacion_ref=NEW.organizacion_ref AND o.expediente_ref=NEW.expediente_ref
                  AND o.propuesta_ref<>NEW.propuesta_ref)
       AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x
                        JOIN vec_contratacion_temporal.propuesta_formalizacion a ON a.propuesta_ref=x.propuesta_anterior_ref
                       WHERE x.propuesta_ref=NEW.propuesta_ref
                         AND x.organizacion_ref=NEW.organizacion_ref AND x.expediente_ref=NEW.expediente_ref
                         AND a.version_resultante<NEW.version_previa+1 AND a.confirmada_en<=NEW.confirmada_en) THEN
        RAISE EXCEPTION 'CT128: el expediente ya tiene una propuesta de nombramiento' USING ERRCODE='23505';
    END IF;
    IF (SELECT count(*) FROM vec_contratacion_temporal.propuesta_formalizacion o
         WHERE o.organizacion_ref=NEW.organizacion_ref AND o.expediente_ref=NEW.expediente_ref
           AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x
                            WHERE x.propuesta_anterior_ref=o.propuesta_ref))<>1 THEN
        RAISE EXCEPTION 'CT128: más de una propuesta vigente' USING ERRCODE='23505';
    END IF;
    RETURN NULL;
END
$$;
ALTER TABLE vec_contratacion_temporal.propuesta_formalizacion
    DROP CONSTRAINT propuesta_formalizacion_organizacion_ref_expediente_ref_key;
CREATE INDEX propuesta_formalizacion_expediente_ct128
    ON vec_contratacion_temporal.propuesta_formalizacion (organizacion_ref, expediente_ref);
CREATE CONSTRAINT TRIGGER propuesta_vigente_unica_ct128 AFTER INSERT ON vec_contratacion_temporal.propuesta_formalizacion
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.propuesta_vigente_unica_ct128();

-- Propuesta que la nueva sustituye: la de la no incorporación cuya aceptación
-- originó la continuación (recibo de CT119) y que es la vigente del
-- expediente. Vacía si la continuación no procede de una no incorporación.
CREATE FUNCTION vec_contratacion_temporal.propuesta_sustituible_ct128(p_continuacion jsonb, p_organizacion text, p_expediente text, p_version numeric)
RETURNS TABLE(propuesta_ref text, no_incorporacion_recibo_ref text, ronda integer)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT n.propuesta_ref, n.recibo_ref,
           (SELECT count(*)::integer+2 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x
             WHERE x.organizacion_ref=p_organizacion AND x.expediente_ref=p_expediente)
      FROM vec_contratacion_temporal.no_incorporacion_v1 n
     WHERE p_continuacion IS NOT NULL
       AND n.aceptacion_resolucion_ref=p_continuacion->'Solicitud'->>'ResolucionRef'
       AND n.intencion_ref=p_continuacion->'Solicitud'->>'IntencionRef'
       AND n.organizacion_ref=p_organizacion AND n.expediente_ref=p_expediente
       AND n.version_esperada+1<=p_version
       AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_formalizacion o
                        WHERE o.organizacion_ref=p_organizacion AND o.expediente_ref=p_expediente
                          AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x
                                           WHERE x.propuesta_anterior_ref=o.propuesta_ref)
                          AND o.propuesta_ref<>n.propuesta_ref)
       AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.propuesta_sustitucion_v1 x
                        WHERE x.propuesta_anterior_ref=n.propuesta_ref)
$$;

-- ============================================================ FRAGMENTOS
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
    FOR i IN 1..array_length(v_firmas,1) LOOP
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO v_antes FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_firmas[i])
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'CT128: función ausente: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
        v_definicion:=v_antes.definicion;
        IF length(v_definicion)-length(replace(v_definicion,v_viejos[i],''))<>length(v_viejos[i])
           OR strpos(v_definicion,'CT128')<>0 AND i IN (1,5,6) THEN
            RAISE EXCEPTION 'CT128: preimagen incompatible: % (%)',v_firmas[i],i USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_viejos[i],v_nuevos[i]);
        EXECUTE v_definicion;
        SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
        IF v_despues.definicion IS DISTINCT FROM v_definicion
           OR replace(v_despues.definicion,v_nuevos[i],v_viejos[i]) IS DISTINCT FROM v_antes.definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'CT128: definición o permisos alterados: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$fragmentos$;

-- ============================================================ CONSULTA
-- Propuestas del expediente para el panel: la vigente y la historia, con la
-- no incorporación que sustituyó a cada una. Como la consulta de CT124, solo
-- tras acreditar la lectura V3 del mismo detalle; sin datos de la persona.
CREATE FUNCTION vec_contratacion_temporal.consultar_propuestas_expediente_v1(p_organizacion text, p_expediente text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $funcion$
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT128: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    IF NOT vec_contratacion_temporal.referencia_valida_ct115(p_organizacion) OR NOT vec_contratacion_temporal.referencia_valida_ct115(p_expediente) THEN
        RAISE EXCEPTION 'CT128: consulta inválida' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
                 JOIN vec_contratacion_temporal.expediente_version_integral v
                   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
                WHERE a.expediente_ref=p_expediente
                  AND v.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion) THEN
        RAISE EXCEPTION 'CT128: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.propuestas-expediente.v1','expediente_ref',p_expediente,
        'propuestas',coalesce((
            SELECT jsonb_agg(jsonb_build_object(
                'orden',row_number,'version_resultante',x.version_resultante,
                'confirmada_en',to_char(x.confirmada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
                'recibo_ref',x.recibo_ref,'vigente',x.sustituida_por IS NULL,
                'sustitucion',CASE WHEN x.sustituida_por IS NULL THEN NULL ELSE jsonb_build_object(
                    'no_incorporacion_recibo_ref',x.no_incorporacion_recibo_ref,'motivo_clave',x.motivo_clave,
                    'registrada_en',to_char(x.no_incorporacion_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
                ORDER BY x.version_resultante)
              FROM (SELECT pf.version_resultante, pf.confirmada_en, pf.recibo_ref, s.propuesta_ref AS sustituida_por,
                           s.no_incorporacion_recibo_ref, n.motivo_clave, n.registrada_en AS no_incorporacion_en,
                           row_number() OVER (ORDER BY pf.version_resultante)
                      FROM vec_contratacion_temporal.propuesta_formalizacion pf
                      LEFT JOIN vec_contratacion_temporal.propuesta_sustitucion_v1 s ON s.propuesta_anterior_ref=pf.propuesta_ref
                      LEFT JOIN vec_contratacion_temporal.no_incorporacion_v1 n ON n.recibo_ref=s.no_incorporacion_recibo_ref
                     WHERE pf.organizacion_ref=p_organizacion AND pf.expediente_ref=p_expediente) x),'[]'::jsonb));
END
$funcion$;

-- ACL: solo el ejecutor de CT invoca la consulta; tabla y auxiliares cerrados.
DO $acl$
DECLARE v record; f regprocedure; destinatario text;
    fachada regprocedure:='vec_contratacion_temporal.consultar_propuestas_expediente_v1(text,text)'::regprocedure;
    auxiliares regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.validar_sustitucion_ct128()'::regprocedure,
      'vec_contratacion_temporal.propuesta_vigente_unica_ct128()'::regprocedure,
      'vec_contratacion_temporal.propuesta_sustituible_ct128(jsonb,text,text,numeric)'::regprocedure];
BEGIN
    FOR v IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
              WHERE c.oid='vec_contratacion_temporal.propuesta_sustitucion_v1'::regclass AND x.grantee<>c.relowner LOOP
        destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.propuesta_sustitucion_v1 FROM %s',destinatario);
    END LOOP;
    REVOKE ALL ON TABLE vec_contratacion_temporal.propuesta_sustitucion_v1 FROM PUBLIC;
    FOREACH f IN ARRAY auxiliares||fachada LOOP
        FOR v IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                  WHERE p.oid=f AND x.grantee<>p.proowner LOOP
            destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,destinatario);
        END LOOP;
    END LOOP;
    EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',fachada::text);
    IF EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
                WHERE c.oid='vec_contratacion_temporal.propuesta_sustitucion_v1'::regclass
                  AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole OR x.grantee<>c.relowner))
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.propuesta_sustitucion_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR EXISTS (SELECT 1 FROM unnest(auxiliares) x WHERE has_function_privilege('vec_contratacion_temporal_ejecutor',x,'EXECUTE'))
       OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',fachada,'EXECUTE')
       OR has_function_privilege('public',fachada,'EXECUTE')
       OR (SELECT NOT prosecdef FROM pg_proc WHERE oid=fachada) THEN
        RAISE EXCEPTION 'CT128: ACL efectiva incompatible' USING ERRCODE='42501';
    END IF;
END
$acl$;
COMMENT ON TABLE vec_contratacion_temporal.propuesta_sustitucion_v1 IS
    'CT128: propuesta de nombramiento que sustituye a la anterior tras su no incorporación (CT124); historia de solo adición, una sola propuesta vigente por expediente.';
COMMIT;
