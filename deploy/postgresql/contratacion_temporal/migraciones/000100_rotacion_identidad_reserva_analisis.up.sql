\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000100',0));

-- Rotación con raíz retenida y RLS acotada a la reserva de cada llamada.
-- Contexto restaurado al salir; ante error se revierte con la transacción o
-- subtransacción del llamador, igual que las escrituras de la propia operación.
-- No reescribe CT97/98 instaladas ni recupera reservas sin identidad conservada.

DROP POLICY identidad_reserva_operacion_analisis_propietario
ON vec_contratacion_temporal.identidad_reserva_operacion_analisis;
CREATE POLICY identidad_reserva_operacion_analisis_select
ON vec_contratacion_temporal.identidad_reserva_operacion_analisis FOR SELECT
TO vec_contratacion_temporal_propietario
USING (
        current_user='vec_contratacion_temporal_propietario'
        AND session_user<>current_user
        AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
        AND ambito_raiz_hmac=current_setting('vec.contratacion_temporal.reserva_analisis_raiz',true)
);
CREATE POLICY identidad_reserva_operacion_analisis_insert
ON vec_contratacion_temporal.identidad_reserva_operacion_analisis FOR INSERT
TO vec_contratacion_temporal_propietario
WITH CHECK (
        current_user='vec_contratacion_temporal_propietario'
        AND session_user<>current_user
        AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
        AND ambito_raiz_hmac=current_setting('vec.contratacion_temporal.reserva_analisis_raiz',true)
);

DROP POLICY revision_reserva_operacion_analisis_propietario
ON vec_contratacion_temporal.revision_reserva_operacion_analisis;
CREATE POLICY revision_reserva_operacion_analisis_select
ON vec_contratacion_temporal.revision_reserva_operacion_analisis FOR SELECT
TO vec_contratacion_temporal_propietario
USING (
        current_user='vec_contratacion_temporal_propietario'
        AND session_user<>current_user
        AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
        AND ambito_raiz_hmac=current_setting('vec.contratacion_temporal.reserva_analisis_raiz',true)
);
CREATE POLICY revision_reserva_operacion_analisis_insert
ON vec_contratacion_temporal.revision_reserva_operacion_analisis FOR INSERT
TO vec_contratacion_temporal_propietario
WITH CHECK (
        current_user='vec_contratacion_temporal_propietario'
        AND session_user<>current_user
        AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
        AND ambito_raiz_hmac=current_setting('vec.contratacion_temporal.reserva_analisis_raiz',true)
);

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.preparar_operacion_analisis_v2(p_operacion jsonb)
RETURNS TABLE (
    resultado text, expediente_json text, recibo_json text,
    reserva_ref text, recibo_ref text, operacion text,
    organizacion_ref text, expediente_ref text,
    version_expediente bigint, actor_ref text, perfil_ref text,
    artefacto_ref text, artefacto_huella_sha256 text,
    ambito_hmac text, huella_semantica_hmac text, estado text
)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_contexto_anterior text := current_setting('vec.contratacion_temporal.reserva_analisis_raiz',true);
    v_fila record;
    v_consulta jsonb;
    v_pares jsonb;
    v_artefactos jsonb;
    v_par jsonb;
    v_base jsonb;
    v_raiz text;
    v_raices text[];
    v_continua boolean;
    v_i integer;
    v_generacion integer;
    v_reserva vec_contratacion_temporal.reserva_operacion_analisis%ROWTYPE;
    v_expediente jsonb;
    v_semantica text;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'identidad de ejecución no autorizada' USING ERRCODE='42501';
    END IF;
    v_consulta := p_operacion -> 'sellos_consulta';
    IF jsonb_typeof(v_consulta) IS DISTINCT FROM 'object'
       OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(v_consulta) AS x(k))
          IS DISTINCT FROM ARRAY['activo','retenidos']::text[]
       OR jsonb_typeof(v_consulta -> 'activo') IS DISTINCT FROM 'object'
       OR jsonb_typeof(v_consulta -> 'retenidos') IS DISTINCT FROM 'array'
       OR jsonb_array_length(v_consulta -> 'retenidos') > 3 THEN
        RAISE EXCEPTION 'identidad funcional de reserva inválida' USING ERRCODE='22023';
    END IF;
    -- CT7 valida el resto del contrato, actor, versiones, límites y sellos.
    SELECT * INTO STRICT v_fila
    FROM vec_contratacion_temporal.preparar_operacion_analisis_v1(p_operacion - 'sellos_consulta');
    v_pares := jsonb_build_array(v_consulta -> 'activo') || (v_consulta -> 'retenidos');
    v_artefactos := jsonb_build_array(p_operacion #> '{sellos_hmac,activo}')
                   || (p_operacion #> '{sellos_hmac,retenidos}');
    IF jsonb_array_length(v_pares) <> jsonb_array_length(v_artefactos) THEN
        RAISE EXCEPTION 'generaciones de reserva no alineadas' USING ERRCODE='22023';
    END IF;
    FOR v_i IN 0..jsonb_array_length(v_pares)-1 LOOP
        v_par := v_pares -> v_i;
        v_base := v_artefactos -> v_i;
        IF jsonb_typeof(v_par) IS DISTINCT FROM 'object'
           OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(v_par) AS x(k))
              IS DISTINCT FROM ARRAY['ambito_hmac','generacion','huella_peticion_hmac']::text[]
           OR (v_par -> 'generacion') IS DISTINCT FROM (v_base -> 'generacion')
           OR (v_par -> 'ambito_hmac') IS DISTINCT FROM (v_base -> 'ambito_hmac')
           OR coalesce(v_par ->> 'huella_peticion_hmac','') !~
              '^hmac-sha256:vec[.]contratacion-temporal[.]analisis[.]huella-semantica/v[1-9][0-9]{0,8}:[a-f0-9]{64}$' THEN
            RAISE EXCEPTION 'sello funcional de reserva inválido' USING ERRCODE='22023';
        END IF;
        v_generacion := (v_par ->> 'generacion')::integer;
        IF substring(v_par ->> 'huella_peticion_hmac' FROM '/v([1-9][0-9]{0,8}):')::integer <> v_generacion THEN
            RAISE EXCEPTION 'generación funcional no alineada' USING ERRCODE='22023';
        END IF;
    END LOOP;
    IF v_fila.resultado IN ('reservada','reutilizada','idempotencia_reutilizada') THEN
        -- CT7 puede encontrar una generación retenida sin crear aún el alias activo.
        -- Sus bloqueos de ámbitos y reserva siguen vigentes en esta transacción.
        SELECT array_agg(DISTINCT a.ambito_raiz_hmac) INTO v_raices
        FROM vec_contratacion_temporal.alias_operacion_analisis a
        WHERE a.alias_ambito_hmac IN (
            SELECT p.par ->> 'ambito_hmac'
            FROM jsonb_array_elements(v_artefactos) AS p(par)
        );
        IF cardinality(v_raices) IS DISTINCT FROM 1 THEN
            RAISE EXCEPTION 'raíz de reserva no unívoca' USING ERRCODE='23505';
        END IF;
        v_raiz := v_raices[1];
        SELECT r.* INTO STRICT v_reserva
        FROM vec_contratacion_temporal.reserva_operacion_analisis r
        WHERE r.ambito_raiz_hmac=v_raiz FOR UPDATE;
        PERFORM set_config('vec.contratacion_temporal.reserva_analisis_raiz',v_raiz,true);
        -- Una generación nueva requiere continuidad acreditada con una conservada.
        -- Ningún par ya conocido puede discrepar; no se atribuye identidad al legado.
        v_continua := EXISTS (
            SELECT 1 FROM jsonb_array_elements(v_pares) AS p(par)
            JOIN vec_contratacion_temporal.identidad_reserva_operacion_analisis i
              ON i.ambito_raiz_hmac=v_raiz
             AND i.generacion=(p.par ->> 'generacion')::integer
             AND i.ambito_consulta_hmac=p.par ->> 'ambito_hmac'
             AND i.huella_consulta_hmac=p.par ->> 'huella_peticion_hmac'
        ) AND NOT EXISTS (
            SELECT 1 FROM jsonb_array_elements(v_pares) AS p(par)
            JOIN vec_contratacion_temporal.identidad_reserva_operacion_analisis i
              ON i.ambito_raiz_hmac=v_raiz
             AND (i.generacion=(p.par ->> 'generacion')::integer
                  OR i.ambito_consulta_hmac=p.par ->> 'ambito_hmac')
            WHERE i.generacion<>(p.par ->> 'generacion')::integer
               OR i.ambito_consulta_hmac<>p.par ->> 'ambito_hmac'
               OR i.huella_consulta_hmac<>p.par ->> 'huella_peticion_hmac'
        );
    END IF;
    IF v_fila.resultado = 'idempotencia_reutilizada' THEN
        -- Únicamente la misma intención previamente sellada puede renovar pruebas.
        -- El legado sin identidad funcional conservada continúa en conflicto.
        IF v_reserva.operacion=p_operacion ->> 'operacion'
           AND v_reserva.organizacion_ref=p_operacion ->> 'organizacion_ref'
           AND v_reserva.expediente_ref=p_operacion ->> 'expediente_ref'
           AND v_reserva.version_expediente=(p_operacion ->> 'version_expediente')::numeric
           AND v_reserva.actor_ref=p_operacion ->> 'actor_ref'
           AND v_reserva.perfil_ref=p_operacion ->> 'perfil_ref'
           AND v_reserva.artefacto_ref=p_operacion ->> 'artefacto_ref'
           AND EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.reserva_operacion_analisis_actual a
               JOIN vec_contratacion_temporal.reserva_operacion_analisis_version v
                 USING (ambito_raiz_hmac,revision)
               WHERE a.ambito_raiz_hmac=v_reserva.ambito_raiz_hmac
                 AND a.revision=1 AND v.estado='reservada'
           )
           AND v_continua
           -- La raíz debe seguir retenida: no se cambia el anclaje de la reserva.
           AND EXISTS (
               SELECT 1 FROM jsonb_array_elements(v_artefactos) AS p(par)
               WHERE p.par ->> 'ambito_hmac'=v_raiz
           ) THEN
            SELECT v.agregado_json INTO v_expediente
            FROM vec_contratacion_temporal.expediente_integral_actual a
            JOIN vec_contratacion_temporal.expediente_version_integral v
              ON v.expediente_ref=a.expediente_ref AND v.version=a.version
            WHERE a.expediente_ref=v_reserva.expediente_ref
              AND a.version=v_reserva.version_expediente
            FOR SHARE OF a,v;
            SELECT p.par ->> 'huella_peticion_hmac' INTO STRICT v_semantica
            FROM jsonb_array_elements(v_artefactos) AS p(par)
            WHERE p.par ->> 'ambito_hmac'=v_reserva.ambito_raiz_hmac;
            IF v_expediente IS NOT NULL THEN
                INSERT INTO vec_contratacion_temporal.revision_reserva_operacion_analisis
                    (ambito_raiz_hmac,revision,artefacto_huella_sha256,huella_semantica_hmac)
                SELECT v_reserva.ambito_raiz_hmac,coalesce(max(r.revision),1)+1,
                       p_operacion ->> 'artefacto_huella_sha256',v_semantica
                FROM vec_contratacion_temporal.revision_reserva_operacion_analisis r
                WHERE r.ambito_raiz_hmac=v_reserva.ambito_raiz_hmac
                ON CONFLICT ON CONSTRAINT revision_reserva_analisis_artefacto_unico DO NOTHING;
                v_fila.resultado := 'reutilizada';
                v_fila.expediente_json := v_expediente::text;
                v_fila.ambito_hmac := v_reserva.ambito_raiz_hmac;
                v_fila.huella_semantica_hmac := v_semantica;
                -- CT7 ya devuelve las mismas coordenadas/recibos y el artefacto solicitado.
            END IF;
        END IF;
    END IF;
    IF v_fila.resultado IN ('reservada','reutilizada') THEN
        IF v_fila.resultado = 'reutilizada' AND NOT v_continua THEN
            RAISE EXCEPTION 'reserva sin identidad funcional coincidente' USING ERRCODE='23505';
        END IF;
        FOR v_i IN 0..jsonb_array_length(v_pares)-1 LOOP
            v_par := v_pares -> v_i;
            v_generacion := (v_par ->> 'generacion')::integer;
            INSERT INTO vec_contratacion_temporal.identidad_reserva_operacion_analisis
                (ambito_raiz_hmac,generacion,ambito_consulta_hmac,huella_consulta_hmac)
            VALUES (v_raiz,v_generacion,v_par ->> 'ambito_hmac',v_par ->> 'huella_peticion_hmac')
            ON CONFLICT DO NOTHING;
            IF NOT EXISTS (
                SELECT 1 FROM vec_contratacion_temporal.identidad_reserva_operacion_analisis i
                WHERE i.ambito_raiz_hmac=v_raiz AND i.generacion=v_generacion
                  AND i.ambito_consulta_hmac=v_par ->> 'ambito_hmac'
                  AND i.huella_consulta_hmac=v_par ->> 'huella_peticion_hmac'
            ) THEN
                RAISE EXCEPTION 'reserva sin identidad funcional coincidente' USING ERRCODE='23505';
            END IF;
        END LOOP;
    END IF;
    PERFORM set_config('vec.contratacion_temporal.reserva_analisis_raiz',coalesce(v_contexto_anterior,''),true);
    RETURN QUERY SELECT v_fila.resultado,v_fila.expediente_json,v_fila.recibo_json,
        v_fila.reserva_ref,v_fila.recibo_ref,v_fila.operacion,v_fila.organizacion_ref,
        v_fila.expediente_ref,v_fila.version_expediente,v_fila.actor_ref,v_fila.perfil_ref,
        v_fila.artefacto_ref,v_fila.artefacto_huella_sha256,v_fila.ambito_hmac,
        v_fila.huella_semantica_hmac,v_fila.estado;
END
$funcion$;


DO $parche$
DECLARE v_def text; v_cuerpo text;
BEGIN
    SELECT p.prosrc,pg_get_functiondef(p.oid) INTO STRICT v_cuerpo,v_def
    FROM pg_proc p
    WHERE p.oid=to_regprocedure('vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb)')
      AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF encode(sha256(convert_to(v_cuerpo,'UTF8')),'hex') <>
       'bb9de1e4e213cfdeca8aebdea7437a7d0e22eacbe976d6e324829e550614a52f'
       OR position($ancla$    -- Sólo cambia la copia local usada para contrastar esta confirmación.$ancla$ IN v_def)=0 THEN
        RAISE EXCEPTION 'guardado previo de análisis incompatible' USING ERRCODE='55000';
    END IF;
    v_def := replace(v_def,
        $ancla$    -- Sólo cambia la copia local usada para contrastar esta confirmación.$ancla$,
        $nuevo$    DECLARE
        v_contexto_anterior text := current_setting('vec.contratacion_temporal.reserva_analisis_raiz',true);
    BEGIN
    PERFORM set_config('vec.contratacion_temporal.reserva_analisis_raiz',r.ambito_raiz_hmac,true);
    -- Sólo cambia la copia local usada para contrastar esta confirmación.$nuevo$);
    IF position($ancla$    IF r.reserva_ref <> o ->> 'reserva_ref'$ancla$ IN v_def)=0 THEN
        RAISE EXCEPTION 'anclaje de guardado incompatible' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,
        $ancla$    IF r.reserva_ref <> o ->> 'reserva_ref'$ancla$,
        $nuevo$    PERFORM set_config('vec.contratacion_temporal.reserva_analisis_raiz',coalesce(v_contexto_anterior,''),true);
    END;
    IF r.reserva_ref <> o ->> 'reserva_ref'$nuevo$);
END
$parche$;
COMMIT;
