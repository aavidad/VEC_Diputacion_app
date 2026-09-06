\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000062',0));
-- Solo reversión controlada sin peticiones en curso ni avisos de sucesor.
-- No borra historia. Tras registrar un aviso nuevo, esta bajada está bloqueada.
LOCK TABLE vec_contratacion_temporal.comunicacion_llamamiento_local IN ACCESS EXCLUSIVE MODE;
DO $comunicacion_sucesor_reversa$
DECLARE
    v_antes record; v_despues record;
    v_cuerpo text; v_definicion text;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local
        WHERE (material_json->'solicitud') ? 'TipoAntecedente') THEN
        RAISE EXCEPTION 'se conserva historia de comunicación del sucesor' USING ERRCODE='55000';
    END IF;
    SELECT p.oid,p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,
           p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
      INTO v_antes FROM pg_proc p
     WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'fce60866f45b43caa8e51c8d6ae4064663577933cf1986268702f9354708ba7d' THEN
        RAISE EXCEPTION 'definición CT62 incompatible, no se sobrescribe' USING ERRCODE='55000';
    END IF;
    -- Se quitan solo los bloques añadidos: la rama v1 no se duplicó ni editó.
    v_cuerpo:=regexp_replace(v_antes.cuerpo,
        '(?s)\n    -- CT62 tipo:.*?    -- CT62 fin tipo\.', '');
    v_cuerpo:=regexp_replace(v_cuerpo,
        '(?s)    -- CT62 sucesor:.*?    -- CT62 fin prefijo sucesor\.\n', '');
    v_cuerpo:=replace(v_cuerpo,E'    END IF; -- CT62 fin rama antecedente.\n','');
    v_cuerpo:=replace(v_cuerpo,
        $nuevo$ARRAY['ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef','VersionEsperada','PruebaEntregaRef'] || CASE WHEN s ? 'TipoAntecedente' THEN ARRAY['TipoAntecedente'] ELSE ARRAY[]::text[] END)$nuevo$,
        $anterior$ARRAY['ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef','VersionEsperada','PruebaEntregaRef'])$anterior$);
    v_cuerpo:=replace(v_cuerpo,
        $nuevo$CASE WHEN s ? 'TipoAntecedente' THEN 'recibo_continuacion_ref' ELSE 'recibo_seleccion_ref' END, s ->> 'PruebaEntregaRef'$nuevo$,
        $anterior$'recibo_seleccion_ref', s ->> 'PruebaEntregaRef'$anterior$);
    IF encode(sha256(convert_to(v_cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'aeebedee4d9bbb252a475fb7498ad4370d64c591939c3fb133f6e55fca262cde' THEN
        RAISE EXCEPTION 'reversión CT54 no exacta' USING ERRCODE='55000';
    END IF;
    v_definicion:=replace(v_antes.definicion,v_antes.cuerpo,v_cuerpo);
    EXECUTE v_definicion;
    SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
           p.proconfig AS configuracion,p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
    IF v_despues.definicion IS DISTINCT FROM v_definicion
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'reversión CT62 alteró definición o permisos' USING ERRCODE='55000';
    END IF;
END
$comunicacion_sucesor_reversa$;
COMMIT;
