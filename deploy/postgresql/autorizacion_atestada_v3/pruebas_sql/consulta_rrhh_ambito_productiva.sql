\set ON_ERROR_STOP on
-- Solo en PostgreSQL 18 efímero, después de AD3-51 UP y del rol CT nuevo.
DO $prueba$
DECLARE
    f record;
    v_cantidad integer := 0;
BEGIN
    IF current_setting('server_version_num')::integer < 180000
       OR NOT EXISTS (
           SELECT 1 FROM pg_roles r
            WHERE r.rolname = 'vec_contratacion_temporal_consultor_rrhh_ambito'
              AND NOT r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreatedb
              AND NOT r.rolcreaterole AND NOT r.rolreplication
              AND NOT r.rolbypassrls)
       OR EXISTS (
           SELECT 1 FROM pg_auth_members m
            WHERE m.member = 'vec_contratacion_temporal_consultor_rrhh_ambito'::regrole)
    THEN RAISE EXCEPTION 'AD3-51: rol técnico inválido'; END IF;

    FOR f IN SELECT p.oid,p.proname,p.prosrc,p.proacl,p.proowner,p.prosecdef,p.proconfig
      FROM pg_proc p
     WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
       AND p.proname IN ('consumir_consulta_rrhh_v3_interna',
                        'revalidar_consumo_consulta_rrhh_v3_interna')
    LOOP
        v_cantidad := v_cantidad + 1;
        IF encode(sha256(convert_to(f.prosrc,'UTF8')),'hex') IS DISTINCT FROM
           (CASE f.proname
             WHEN 'consumir_consulta_rrhh_v3_interna' THEN
               '6baef6127627ce9d6e6146c9d5425d7463f1a89b10411aac70fa70ed1944fd98'
             ELSE '3530828669500274e9a46838c0d890aa0a974c2779e3fa38b5b2054c3919706d'
           END)
           OR f.proowner <> 'vec_autorizacion_atestada_v3_propietario'::regrole
           OR NOT f.prosecdef
           OR f.proacl IS DISTINCT FROM
              ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
           OR f.proconfig IS DISTINCT FROM
              (CASE WHEN f.proname='consumir_consulta_rrhh_v3_interna'
                   THEN ARRAY['search_path=pg_catalog','lock_timeout=2s']
                   ELSE ARRAY['search_path=pg_catalog','lock_timeout=1s'] END)
        THEN RAISE EXCEPTION 'AD3-51: contrato de función inválido: %',f.proname; END IF;
    END LOOP;
    IF v_cantidad <> 2 THEN RAISE EXCEPTION 'AD3-51: faltan funciones internas'; END IF;

    v_cantidad := 0;
    FOR f IN SELECT p.oid,p.proname FROM pg_proc p
     WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
       AND p.proname IN (
          'registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada',
          'registrar_y_consumir_consulta_detalle_rrhh_v3_atestada',
          'revalidar_consumo_consulta_cuadro_rrhh_v3_atestada',
          'revalidar_consumo_consulta_detalle_rrhh_v3_atestada')
    LOOP
        v_cantidad := v_cantidad + 1;
        IF has_function_privilege('vec_contratacion_temporal_consultor_rrhh_ambito',f.oid,'EXECUTE')
        THEN RAISE EXCEPTION 'AD3-51: EXECUTE legacy inesperado: %',f.proname; END IF;
    END LOOP;
    IF v_cantidad <> 4 THEN RAISE EXCEPTION 'AD3-51: faltan fachadas'; END IF;
END $prueba$;
