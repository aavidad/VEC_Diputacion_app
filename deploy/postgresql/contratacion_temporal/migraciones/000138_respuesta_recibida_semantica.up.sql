\set ON_ERROR_STOP on
-- CT138: ruta aditiva. CT56/63/121/124 y sus recibos siguen intactos.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000138',0));

DO $instalar$
DECLARE
    v_original record;
    v_definicion text;
    v_cambio record;
BEGIN
    SELECT p.oid, pg_get_functiondef(p.oid) AS definicion
      INTO v_original FROM pg_proc p
     WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND
       OR to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR strpos(v_original.definicion,$ct124$r.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')$ct124$)=0 THEN
        RAISE EXCEPTION 'CT138 requiere respuesta CT124 y v2 ausente' USING ERRCODE='55000';
    END IF;
    v_definicion:=v_original.definicion;
    FOR v_cambio IN SELECT * FROM (VALUES
        ($a$registrar_respuesta_recibida_rrhh_v1($a$,$n$registrar_respuesta_recibida_rrhh_v2($n$),
        ($a$    v_previa vec_contratacion_temporal.respuesta_recibida_rrhh%ROWTYPE;$a$,
         $n$    v_previa vec_contratacion_temporal.respuesta_recibida_rrhh%ROWTYPE;
    v_previa_total integer;
    v_seleccion_ref text;
    v_huella_semantica text;
    v_clave_servidor uuid;$n$),
        ($a$       OR (s ->> 'ClaveIdempotencia') !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'$a$,
         $n$       OR ((s ->> 'ClaveIdempotencia') <> '' AND
           (s ->> 'ClaveIdempotencia') !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')$n$),
        ($a$    SELECT * INTO v_previa FROM vec_contratacion_temporal.respuesta_recibida_rrhh
     WHERE organizacion_ref = s ->> 'OrganizacionRef'
       AND clave_idempotencia = (s ->> 'ClaveIdempotencia')::uuid;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM d ->> 'principal_id'
           OR v_previa.perfil_ref IS DISTINCT FROM d ->> 'perfil_activo_ref' THEN
            RAISE EXCEPTION 'replay de respuesta denegado' USING ERRCODE = 'P0563';
        END IF;
        IF v_previa.material IS DISTINCT FROM p_material OR v_previa.material_json IS DISTINCT FROM s THEN
            RAISE EXCEPTION 'clave de respuesta divergente' USING ERRCODE = 'P0561';
        END IF;
        RETURN v_previa.recibo_json || jsonb_build_object('Estado','replay_registrada_por_rrhh');
    END IF;$a$,
         $n$    -- CT138 coteja la respuesta después de verificar el antecedente CT.$n$),
        ($a$       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh
           WHERE organizacion_ref = s ->> 'OrganizacionRef' AND comunicacion_ref = s ->> 'ComunicacionRef') THEN$a$,
         $n$ THEN$n$),
        ($a$    v_ahora := date_trunc('microseconds',clock_timestamp());$a$,
         $n$    -- seleccion_ref ya está seudonimizada en el recibo confirmado CT46.
    -- Nunca procede del POST ni de tablas de Bolsa.
    v_seleccion_ref:=v_seleccion.recibo_json->>'seleccion_ref';
    IF v_seleccion_ref IS NULL
       OR v_seleccion_ref !~ '^hmac-sha256:vec[.]contratacion-temporal[.]seleccion/v[1-9][0-9]*:[0-9a-f]{64}$'
       OR right(v_seleccion_ref,64)=repeat('0',64) THEN
        RAISE EXCEPTION 'selección sin seudónimo verificado' USING ERRCODE='P0562';
    END IF;
    -- Un cerrojo por candidato y llamamiento ordena operaciones equivalentes.
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'ct138:'||(s->>'OrganizacionRef')||':'||(s->>'LlamamientoRef')||':'||v_seleccion_ref,0));
    SELECT count(*) INTO v_previa_total
      FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
      JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
        ON e.clave_idempotencia=r.seleccion_clave
     WHERE r.organizacion_ref=s->>'OrganizacionRef'
       AND r.llamamiento_ref=s->>'LlamamientoRef'
       AND e.recibo_json->>'seleccion_ref'=v_seleccion_ref;
    IF v_previa_total>1 THEN
        RAISE EXCEPTION 'respuestas previas incompatibles' USING ERRCODE='P0561';
    END IF;
    IF v_previa_total=1 THEN
        SELECT r.* INTO STRICT v_previa
          FROM vec_contratacion_temporal.respuesta_recibida_rrhh r
          JOIN vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
            ON e.clave_idempotencia=r.seleccion_clave
         WHERE r.organizacion_ref=s->>'OrganizacionRef'
           AND r.llamamiento_ref=s->>'LlamamientoRef'
           AND e.recibo_json->>'seleccion_ref'=v_seleccion_ref;
        IF v_previa.material_json-'ClaveIdempotencia' IS DISTINCT FROM s-'ClaveIdempotencia' THEN
            RAISE EXCEPTION 'contenido de respuesta divergente' USING ERRCODE='P0561';
        END IF;
        -- El recibo CT56 se devuelve tal cual: no se reescriben su clave,
        -- referencia, auditoría ni fecha. Solo cambia el estado de la vista.
        RETURN v_previa.recibo_json || jsonb_build_object('Estado','replay_registrada_por_rrhh');
    END IF;
    -- Clave de operación derivada por servidor: llamamiento, candidato
    -- seudonimizado, tipo y huella de toda la declaración salvo clave cliente.
    v_huella_semantica:=encode(sha256(convert_to(
        'ct-respuesta-v2:'||(s->>'LlamamientoRef')||':'||v_seleccion_ref||':'||
        (s->>'Respuesta')||':'||(s-'ClaveIdempotencia')::text,'UTF8')),'hex');
    v_clave_servidor:=(substr(v_huella_semantica,1,8)||'-'||
        substr(v_huella_semantica,9,4)||'-4'||substr(v_huella_semantica,14,3)||
        '-8'||substr(v_huella_semantica,18,3)||'-'||substr(v_huella_semantica,21,12))::uuid;
    s:=jsonb_set(s,'{ClaveIdempotencia}',to_jsonb(v_clave_servidor::text));
    p_material:=s::text;
    v_material_huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
    v_ahora := date_trunc('microseconds',clock_timestamp());$n$)
    ) AS cambios(anterior,nuevo) LOOP
        IF length(v_definicion)-length(replace(v_definicion,v_cambio.anterior,''))
           <>length(v_cambio.anterior) THEN
            RAISE EXCEPTION 'preimagen CT138 incompatible: %', left(v_cambio.anterior,48) USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_cambio.anterior,v_cambio.nuevo);
    END LOOP;
    -- Solo una serialización abortada es reintentable con otra autorización.
    -- Un bloqueo o fallo de red permanece incierto y no se reintenta a ciegas.
    IF length(v_definicion)-length(replace(v_definicion,
       $a$WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN$a$,''))
       <>2*length($a$WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN$a$) THEN
        RAISE EXCEPTION 'preimagen de excepciones CT138 incompatible' USING ERRCODE='55000';
    END IF;
    v_definicion:=replace(v_definicion,
       $a$WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN$a$,
       $n$WHEN serialization_failure THEN
            RAISE EXCEPTION 'serialización de respuesta' USING ERRCODE='40001';
        WHEN deadlock_detected OR lock_not_available THEN$n$);
    EXECUTE v_definicion;
    IF to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'CT138 no creó v2' USING ERRCODE='55000';
    END IF;
END
$instalar$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
COMMIT;
