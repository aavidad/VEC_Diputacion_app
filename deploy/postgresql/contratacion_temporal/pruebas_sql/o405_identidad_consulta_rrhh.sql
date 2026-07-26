DO $prueba$
DECLARE
    v_funcion oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.'
        || 'revalidar_consulta_rrhh_v1(text,text)'
    );
    v_salidas text[];
    v_tabla text;
BEGIN
    SELECT p.proargnames[
               p.pronargs + 1:
               pg_catalog.array_length(p.proargnames, 1)
           ]
      INTO v_salidas
      FROM pg_catalog.pg_proc p
     WHERE p.oid = v_funcion;

    IF v_funcion IS NULL
       OR v_salidas IS DISTINCT FROM ARRAY[
           'autenticacion_ref',
           'autenticacion_huella_sha256',
           'asercion_ref',
           'sesion_ref',
           'control_sesion_ref',
           'control_sesion_revision',
           'control_sesion_huella_sha256',
           'cuenta_ref',
           'cuenta_ordinaria_ref',
           'cuenta_privilegiada',
           'superficie',
           'metodo_observado',
           'garantia_observada',
           'politica_garantia_ref',
           'politica_garantia_huella_sha256',
           'autenticacion_verificada_en',
           'sesion_emitida_en',
           'sesion_valida_hasta',
           'sesion_revalidada_en',
           'login_tecnico'
       ]::text[]
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc p
             JOIN pg_catalog.pg_roles r ON r.oid = p.proowner
            WHERE p.oid = v_funcion
              AND r.rolname =
                  'vec_identidad_sesiones_v1_propietario'
              AND p.prosecdef
              AND p.provolatile = 'v'
              AND p.proparallel = 'u'
              AND p.proconfig @> ARRAY[
                  'search_path=pg_catalog',
                  'lock_timeout=1s'
              ]::text[]
       ) THEN
        RAISE EXCEPTION 'contrato de función nominal incorrecto';
    END IF;

    IF pg_catalog.has_function_privilege(
           'public', v_funcion, 'EXECUTE'
       )
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           v_funcion, 'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh',
           v_funcion, 'EXECUTE'
       )
       OR pg_catalog.has_schema_privilege(
           'vec_contratacion_temporal_consultor_rrhh',
           'vec_identidad_sesiones_v1', 'USAGE'
       ) THEN
        RAISE EXCEPTION 'ACL nominal de identidad incorrecta';
    END IF;

    FOREACH v_tabla IN ARRAY ARRAY[
        'cuenta', 'alias_hmac_cuenta', 'estado_cuenta',
        'estado_cuenta_actual', 'consumo_asercion'
    ] LOOP
        IF pg_catalog.has_table_privilege(
               'vec_contratacion_temporal_propietario',
               'vec_identidad_sesiones_v1.' || v_tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
           )
           OR pg_catalog.has_table_privilege(
               'vec_contratacion_temporal_consultor_rrhh',
               'vec_identidad_sesiones_v1.' || v_tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
           ) THEN
            RAISE EXCEPTION 'privilegio directo sobre %', v_tabla;
        END IF;
    END LOOP;
END
$prueba$;
