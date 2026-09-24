-- Comprobación focal estructural. Ejecutar después de Identidad 000006 y CT 000002.
DO $prueba$
DECLARE
    v_consulta oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)'
    );
    v_politica oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)'
    );
BEGIN
    IF v_consulta IS NULL OR v_politica IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc p
           JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
           WHERE p.oid = v_consulta
             AND r.rolname = 'vec_identidad_sesiones_v1_propietario'
             AND p.prosecdef AND p.provolatile = 'v'
             AND p.proparallel = 'u'
             AND p.proconfig = ARRAY[
                 'search_path=pg_catalog', 'lock_timeout=1s'
             ]::text[]
             AND pg_catalog.octet_length(p.prosrc) = 4039
             AND pg_catalog.encode(pg_catalog.sha256(
                 pg_catalog.convert_to(p.prosrc, 'UTF8')
             ), 'hex') =
                 '81b25aadd776dbd69fa2ab115c756b8e3c06cefc3b1fe2b52169b1e6f8c0f714'
             AND (
                 SELECT pg_catalog.count(*) = 2
                    AND pg_catalog.bool_and(
                        a.grantor = p.proowner
                        AND a.grantee IN (
                            p.proowner,
                            'vec_contratacion_temporal_propietario'::regrole
                        )
                        AND a.privilege_type = 'EXECUTE'
                        AND NOT a.is_grantable
                    )
                   FROM pg_catalog.aclexplode(p.proacl) a
             )
       )
       OR pg_catalog.has_function_privilege('public', v_consulta, 'EXECUTE')
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario', v_consulta, 'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh', v_consulta, 'EXECUTE'
       )
       OR pg_catalog.has_function_privilege('public', v_politica, 'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario', v_politica, 'EXECUTE'
       )
       OR vec_identidad_sesiones_v1.
          admite_politica_certificado_personal_desarrollo_v1(
              'pga_aaaaaaaaaaaaaaaaaaaaaaaa',
              pg_catalog.repeat('b',64), pg_catalog.clock_timestamp()
          ) IS TRUE THEN
        RAISE EXCEPTION 'contrato de revalidación CT incorrecto'
            USING ERRCODE='55000';
    END IF;
END $prueba$;
