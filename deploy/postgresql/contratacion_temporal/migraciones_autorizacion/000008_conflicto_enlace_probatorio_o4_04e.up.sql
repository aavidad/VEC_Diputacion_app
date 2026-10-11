-- O4-04E: corregir la colisión PL/pgSQL entre salida decision_ref y árbitro INSERT.
-- Preimagen literal instalada en copia fría principal del 05/10; historia intacta.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:o4_04:migraciones', 0));
DO $preimagen$
DECLARE
    f oid := pg_catalog.to_regprocedure('vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)');
    actual text;
BEGIN
    IF f IS NULL OR pg_catalog.to_regclass('vec_autorizacion.enlace_decision_cobertura_ct_o404e') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint WHERE conrelid='vec_autorizacion.enlace_decision_cobertura_ct_o404e'::pg_catalog.regclass AND conname='enlace_decision_cobertura_ct_o404e_pkey' AND contype='p')
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
            AND p.proowner='vec_autorizacion_propietario'::pg_catalog.regrole
            AND p.prosecdef AND p.provolatile='v'
            AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']
            AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_contratacion_temporal_propietario=X/vec_autorizacion_propietario}')
    THEN RAISE EXCEPTION 'O4-04E8: preimagen estructural incompatible' USING ERRCODE='55000'; END IF;
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex') INTO actual;
    IF actual <> 'adb303af5416013c0f9c6e9612be9dc985c392ac829db3e0776992d6528e60e1'
    THEN RAISE EXCEPTION 'O4-04E8: preimagen definición actual=% esperado=adb303af5416013c0f9c6e9612be9dc985c392ac829db3e0776992d6528e60e1',actual USING ERRCODE='55000'; END IF;
END
$preimagen$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(p_decision_canonica bytea, p_motivo_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_vinculo_efecto jsonb)
 RETURNS TABLE(rama text, concedida boolean, codigo text, decision_ref text, correlacion_ref text, organizacion_ref text, expediente_ref text, version_expediente numeric, reserva_ref text, contexto_recurso_huella_sha256 text, decision_huella_sha256 text, huella_orden_sha256 text, lote_huella_sha256 text, prueba_vinculo_sha256 text, registrada_en timestamp with time zone, revalidada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path = pg_catalog, pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    v record;
    v_vinculada_en timestamptz(6);
BEGIN
    SELECT * INTO v
      FROM vec_autorizacion
           .o404d_registrar_decision_cobertura_sin_enlace_v1(
               p_decision_canonica,
               p_motivo_canonico,
               p_persona_version,
               p_perfil_version,
               p_vinculo_efecto
           );
    IF NOT FOUND THEN
        RETURN;
    END IF;

    v_vinculada_en := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    INSERT INTO vec_autorizacion.enlace_decision_cobertura_ct_o404e (
        decision_ref, rama, concedida, codigo, accion,
        decision_huella_sha256,
        decision_concedida_ref, decision_denegada_ref, correlacion_ref,
        organizacion_ref, expediente_ref, version_expediente, reserva_ref,
        contexto_recurso_huella_sha256, huella_orden_sha256,
        lote_huella_sha256, prueba_vinculo_sha256, registrada_en,
        revalidada_en, vinculada_en
    ) VALUES (
        v.decision_ref, v.rama, v.concedida, v.codigo,
        p_vinculo_efecto ->> 'accion',
        v.decision_huella_sha256,
        CASE WHEN v.rama = 'concedida' THEN v.decision_ref END,
        CASE WHEN v.rama = 'denegada' THEN v.decision_ref END,
        v.correlacion_ref, v.organizacion_ref, v.expediente_ref,
        v.version_expediente, v.reserva_ref,
        v.contexto_recurso_huella_sha256, v.huella_orden_sha256,
        v.lote_huella_sha256, v.prueba_vinculo_sha256,
        v.registrada_en, v.revalidada_en, v_vinculada_en
    ) ON CONFLICT ON CONSTRAINT enlace_decision_cobertura_ct_o404e_pkey DO NOTHING;

    PERFORM 1
      FROM vec_autorizacion.enlace_decision_cobertura_ct_o404e AS e
     WHERE e.decision_ref = v.decision_ref
       AND e.rama = v.rama
       AND e.concedida = v.concedida
       AND e.codigo = v.codigo
       AND e.accion = p_vinculo_efecto ->> 'accion'
       AND e.decision_huella_sha256 = v.decision_huella_sha256
       AND e.correlacion_ref = v.correlacion_ref
       AND e.organizacion_ref = v.organizacion_ref
       AND e.expediente_ref = v.expediente_ref
       AND e.version_expediente = v.version_expediente
       AND e.reserva_ref = v.reserva_ref
       AND e.contexto_recurso_huella_sha256 =
           v.contexto_recurso_huella_sha256
       AND e.huella_orden_sha256 = v.huella_orden_sha256
       AND e.lote_huella_sha256 IS NOT DISTINCT FROM v.lote_huella_sha256
       AND e.prueba_vinculo_sha256 = v.prueba_vinculo_sha256
       AND e.registrada_en = v.registrada_en
       AND e.revalidada_en IS NOT DISTINCT FROM v.revalidada_en;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'colisión de enlace probatorio VEC-CT O4-04E';
    END IF;

    RETURN QUERY SELECT
        v.rama::text, v.concedida::boolean, v.codigo::text,
        v.decision_ref::text, v.correlacion_ref::text,
        v.organizacion_ref::text, v.expediente_ref::text,
        v.version_expediente::numeric, v.reserva_ref::text,
        v.contexto_recurso_huella_sha256::text,
        v.decision_huella_sha256::text, v.huella_orden_sha256::text,
        v.lote_huella_sha256::text, v.prueba_vinculo_sha256::text,
        v.registrada_en::timestamptz, v.revalidada_en::timestamptz;
END
$function$;

DO $postimagen$
DECLARE f oid := pg_catalog.to_regprocedure('vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)');
BEGIN
    IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
       AND p.proowner='vec_autorizacion_propietario'::pg_catalog.regrole
       AND p.prosecdef AND p.provolatile='v'
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
       AND p.proacl::text='{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_contratacion_temporal_propietario=X/vec_autorizacion_propietario}')
       OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(f),'ON CONFLICT ON CONSTRAINT enlace_decision_cobertura_ct_o404e_pkey DO NOTHING')=0
    THEN RAISE EXCEPTION 'O4-04E8: postimagen incompatible' USING ERRCODE='55000'; END IF;
END
$postimagen$;
COMMIT;
