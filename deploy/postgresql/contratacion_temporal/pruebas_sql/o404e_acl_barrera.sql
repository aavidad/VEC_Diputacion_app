\set ON_ERROR_STOP on
DO $prueba$
DECLARE
  v_tabla text;
  v_funcion text;
BEGIN
  IF current_setting('server_version_num')::integer/10000<>18
     OR pg_catalog.pg_is_in_recovery() THEN
    RAISE EXCEPTION 'O4-04E exige primario PostgreSQL 18';
  END IF;
  IF (SELECT version_esquema
        FROM vec_contratacion_temporal.control_migracion_cobertura_o4
       WHERE control)<>14 THEN
    RAISE EXCEPTION 'barrera O4-04E distinta de 14';
  END IF;
  IF NOT EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname='vec_contratacion_temporal_confirmador_cobertura'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole
       AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls
  ) THEN
    RAISE EXCEPTION 'rol confirmador O4-04E inseguro';
  END IF;
  IF EXISTS(
    SELECT 1 FROM pg_catalog.pg_roles
     WHERE rolname IN(
       'vec_contratacion_temporal_propietario',
       'vec_autorizacion_propietario')
       AND (rolcanlogin OR rolsuper OR rolbypassrls)
  ) OR (SELECT count(*) FROM pg_catalog.pg_roles
         WHERE rolname IN(
           'vec_contratacion_temporal_propietario',
           'vec_autorizacion_propietario'))<>2 THEN
    RAISE EXCEPTION 'propietarios O4-04E inseguros';
  END IF;
  IF NOT pg_catalog.has_schema_privilege(
       'vec_contratacion_temporal_confirmador_cobertura',
       'vec_contratacion_temporal','USAGE')
     OR pg_catalog.has_schema_privilege(
       'vec_contratacion_temporal_confirmador_cobertura',
       'vec_contratacion_temporal','CREATE') THEN
    RAISE EXCEPTION 'ACL de esquema confirmador divergente';
  END IF;
  IF pg_catalog.has_schema_privilege(
       'vec_contratacion_temporal_confirmador_cobertura',
       'vec_autorizacion','USAGE,CREATE') THEN
    RAISE EXCEPTION 'ACL cross-schema VEC publicada al confirmador';
  END IF;
  IF EXISTS(
    SELECT 1 FROM pg_catalog.pg_proc p
    JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
    JOIN pg_catalog.pg_roles r ON r.oid=p.proowner
    WHERE p.oid=ANY(ARRAY[
      'vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(jsonb)'::regprocedure,
      'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(jsonb)'::regprocedure,
      'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)'::regprocedure
    ])
      AND (NOT p.prosecdef
        OR p.proparallel<>'u'
        OR p.provolatile<>CASE
          WHEN p.oid=
            'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(jsonb)'::regprocedure
          THEN 's'::"char" ELSE 'v'::"char" END
        OR r.rolname<>CASE n.nspname
          WHEN 'vec_autorizacion' THEN 'vec_autorizacion_propietario'
          ELSE 'vec_contratacion_temporal_propietario' END
        OR (SELECT array_agg(x ORDER BY x)
              FROM unnest(coalesce(p.proconfig,ARRAY[]::text[])) x)
           IS DISTINCT FROM ARRAY[
             'TimeZone=UTC','lock_timeout=2s','row_security=on',
             'search_path=pg_catalog']::text[])
  ) THEN
    RAISE EXCEPTION
      'owner/SECURITY DEFINER/volatilidad/proconfig O4-04E divergente';
  END IF;
  IF NOT pg_catalog.has_function_privilege(
       'vec_contratacion_temporal_propietario',
       'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)',
       'EXECUTE')
     OR pg_catalog.has_function_privilege(
       'vec_contratacion_temporal_confirmador_cobertura',
       'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)',
       'EXECUTE')
     OR pg_catalog.has_function_privilege(
       'vec_autorizacion_registro',
       'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)',
       'EXECUTE')
     OR pg_catalog.has_function_privilege(
       'vec_autorizacion_motivos_evaluador',
       'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)',
       'EXECUTE')
     OR pg_catalog.has_function_privilege(
       'public',
       'vec_autorizacion.registrar_decision_cobertura_contexto_exacto_o404e_v1(bytea,bytea,numeric,numeric,jsonb)',
       'EXECUTE') THEN
    RAISE EXCEPTION 'ACL cross-schema del wrapper O4-04E divergente';
  END IF;
  FOREACH v_funcion IN ARRAY ARRAY[
    'vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(jsonb)',
    'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(jsonb)'
  ]::text[] LOOP
    IF NOT pg_catalog.has_function_privilege(
      'vec_contratacion_temporal_confirmador_cobertura',
      v_funcion,'EXECUTE'
    ) OR pg_catalog.has_function_privilege('public',v_funcion,'EXECUTE')
       OR pg_catalog.has_function_privilege(
         'vec_contratacion_temporal_ejecutor',v_funcion,'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
         'vec_contratacion_temporal_migrador',v_funcion,'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
         'vec_contratacion_temporal_gobernador',v_funcion,'EXECUTE'
       ) THEN
      RAISE EXCEPTION 'ACL exterior divergente: %',v_funcion;
    END IF;
  END LOOP;
  IF EXISTS(
    SELECT 1 FROM pg_catalog.pg_proc p
    JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname IN(
      'vec_contratacion_temporal','vec_autorizacion')
      AND (p.proname LIKE '%o404e%'
        OR p.proname='registrar_decision_cobertura_contexto_exacto_o404e_v1')
      AND p.oid<>ALL(ARRAY[
        'vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(jsonb)'::regprocedure,
        'vec_contratacion_temporal.leer_terminal_primario_decision_cobertura_o404e_v1(jsonb)'::regprocedure
      ])
      AND (pg_catalog.has_function_privilege(
             'vec_contratacion_temporal_confirmador_cobertura',
             p.oid,'EXECUTE')
        OR pg_catalog.has_function_privilege('public',p.oid,'EXECUTE'))
  ) THEN
    RAISE EXCEPTION 'catálogo privado O4-04E publicó una primitiva';
  END IF;
  FOREACH v_funcion IN ARRAY ARRAY[
    'vec_contratacion_temporal.o404e_construir_lote_c1_v1(jsonb,text)',
    'vec_contratacion_temporal.o404e_confirmar_concesion_v1(jsonb,text,timestamptz)',
    'vec_contratacion_temporal.o404e_confirmar_denegacion_v1(jsonb,timestamptz)',
    'vec_contratacion_temporal.o404e_cerrar_terminal_v1(jsonb,jsonb,jsonb,timestamptz)',
    'vec_contratacion_temporal.o404e_leer_terminal_interno_v1(text,text)'
  ]::text[] LOOP
    IF pg_catalog.has_function_privilege(
      'vec_contratacion_temporal_confirmador_cobertura',
      v_funcion,'EXECUTE'
    ) OR pg_catalog.has_function_privilege('public',v_funcion,'EXECUTE') THEN
      RAISE EXCEPTION 'primitiva O4-04E publicada: %',v_funcion;
    END IF;
  END LOOP;
  FOREACH v_tabla IN ARRAY ARRAY[
    'alias_operacion_decision_cobertura',
    'acreditacion_gobierno_decision_cobertura',
    'auditoria_decision_cobertura',
    'confirmacion_operacion_decision_cobertura',
    'consumo_cobertura_evidencia',
    'consumo_cobertura_lote',
    'control_cadenas_expediente_integral',
    'control_migracion_cobertura_o4',
    'decision_cobertura_gobernada_durable',
    'expediente_integral_actual',
    'expediente_version_integral',
    'actuacion_expediente_integral',
    'outbox_expediente_integral',
    'gobi_o404b_actual',
    'gobi_o404b_actuacion',
    'gobi_o404b_catalogo',
    'gobi_o404b_checkpoint',
    'gobi_o404b_evento',
    'gobi_o404b_politica',
    'gobi_o404b_retirada',
    'prueba_denegacion_decision_cobertura',
    'reserva_operacion_decision_cobertura',
    'reserva_operacion_decision_cobertura_actual',
    'reserva_operacion_decision_cobertura_version',
    'terminal_operacion_decision_cobertura'
  ]::text[] LOOP
    IF NOT EXISTS(
      SELECT 1 FROM pg_catalog.pg_class c
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      JOIN pg_catalog.pg_roles r ON r.oid=c.relowner
       WHERE n.nspname='vec_contratacion_temporal'
         AND c.relname=v_tabla
         AND r.rolname='vec_contratacion_temporal_propietario'
         AND c.relrowsecurity AND c.relforcerowsecurity
    ) OR (SELECT count(*) FROM pg_catalog.pg_policy p
           JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
           JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname='vec_contratacion_temporal'
            AND c.relname=v_tabla
            AND p.polcmd='*'
            AND p.polroles=ARRAY[
              'vec_contratacion_temporal_propietario'::regrole::oid]
            AND pg_catalog.pg_get_expr(p.polqual,p.polrelid)='true'
            AND pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid)='true'
        )<>1
      OR (SELECT count(*) FROM pg_catalog.pg_policy p
           JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
           JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname='vec_contratacion_temporal'
            AND c.relname=v_tabla)<>1
      OR pg_catalog.has_table_privilege(
      'vec_contratacion_temporal_confirmador_cobertura',
      'vec_contratacion_temporal.'||v_tabla,
      'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN'
    ) THEN
      RAISE EXCEPTION 'RLS/ACL de tabla divergente: %',v_tabla;
    END IF;
  END LOOP;
  FOREACH v_tabla IN ARRAY ARRAY[
    'decision_concedida_contexto_actor_v3',
    'decision_denegada_contexto_actor_v3',
    'enlace_decision_cobertura_ct_o404e'
  ]::text[] LOOP
    IF NOT EXISTS(
      SELECT 1 FROM pg_catalog.pg_class c
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      JOIN pg_catalog.pg_roles r ON r.oid=c.relowner
      WHERE n.nspname='vec_autorizacion' AND c.relname=v_tabla
        AND r.rolname='vec_autorizacion_propietario'
        AND c.relrowsecurity AND c.relforcerowsecurity
    ) OR (SELECT count(*) FROM pg_catalog.pg_policy p
           JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
           JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname='vec_autorizacion' AND c.relname=v_tabla
            AND p.polname=CASE v_tabla
              WHEN 'enlace_decision_cobertura_ct_o404e'
                THEN 'propietario_total'
              ELSE 'acceso_propietario_exacto' END
            AND p.polcmd='*'
            AND p.polroles=ARRAY[
              'vec_autorizacion_propietario'::regrole::oid]
            AND pg_catalog.pg_get_expr(p.polqual,p.polrelid) IN(
              'true',
              '(CURRENT_USER = ''vec_autorizacion_propietario''::name)')
            AND pg_catalog.pg_get_expr(p.polwithcheck,p.polrelid) IN(
              'true',
              '(CURRENT_USER = ''vec_autorizacion_propietario''::name)')
        )<>1
      OR (SELECT count(*) FROM pg_catalog.pg_policy p
           JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
           JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname='vec_autorizacion' AND c.relname=v_tabla)<>1
      OR pg_catalog.has_table_privilege(
      'vec_contratacion_temporal_confirmador_cobertura',
      'vec_autorizacion.'||v_tabla,
      'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN'
    ) THEN
      RAISE EXCEPTION 'RLS/ACL cross-schema divergente: %',v_tabla;
    END IF;
  END LOOP;
END
$prueba$;

DO $truncado$
BEGIN
  BEGIN
    SET LOCAL ROLE vec_contratacion_temporal_propietario;
    TRUNCATE vec_contratacion_temporal
      .prueba_denegacion_decision_cobertura;
    RAISE EXCEPTION 'TRUNCATE del propietario no fue bloqueado';
  EXCEPTION WHEN SQLSTATE '55000' THEN
    NULL;
  END;
END
$truncado$;
