\set ON_ERROR_STOP on
-- Sólo fixture privado sintético: LOGIN registrador nominal. No provisiona
-- permisos ni imprime material V3. La variante esperar_rol_denegado prueba un
-- login con membresía adicional que dirección haya creado para el ensayo.
\if :{?decision_ref}
\else
 \echo AUT29 requiere decision_ref de una concesion sintetica real
 \quit 2
\endif
\if :{?correlacion_ref}
\else
 \echo AUT29 requiere correlacion_ref de esa concesion
 \quit 2
\endif
\if :{?esperar_rol_denegado}
\else
 \set esperar_rol_denegado false
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC'; SET LOCAL search_path=pg_catalog;
SELECT set_config('vec_prueba.decision_ref',:'decision_ref',true),set_config('vec_prueba.correlacion_ref',:'correlacion_ref',true);
\if :esperar_rol_denegado
DO $mixto$ BEGIN
 BEGIN
  PERFORM vec_meritos.registrar_intento_consulta_propia_v1(current_setting('vec_prueba.decision_ref'),current_setting('vec_prueba.correlacion_ref'),'denegada');
  RAISE EXCEPTION 'AUT29: rol mezclado aceptado' USING ERRCODE='XX000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $mixto$;
\else
DO $runtime$ DECLARE d text:=current_setting('vec_prueba.decision_ref'); c text:=current_setting('vec_prueba.correlacion_ref');
 a jsonb; b jsonb; ajena text; BEGIN
 a:=vec_meritos.registrar_intento_consulta_propia_v1(d,c,'denegada');
 b:=vec_meritos.registrar_intento_consulta_propia_v1(d,c,'denegada');
 IF a IS DISTINCT FROM b OR a#>>'{metadata,pdp_resultado}' IS DISTINCT FROM 'concedida'
 OR a#>>'{metadata,origen_resultado}' IS DISTINCT FROM 'observacion_registrador'
 OR a->>'authorization_ref' IS DISTINCT FROM d OR a->>'correlation_ref' IS DISTINCT FROM c
 THEN RAISE EXCEPTION 'AUT29: replay o semántica divergente'; END IF;
 BEGIN
  PERFORM vec_meritos.registrar_intento_consulta_propia_v1(d,c,'no_confirmado');
  RAISE EXCEPTION 'AUT29: replay divergente aceptado' USING ERRCODE='XX000';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 ajena:='corr_'||repeat(CASE WHEN right(c,1)='0' THEN '1' ELSE '0' END,32);
 BEGIN
  PERFORM vec_meritos.registrar_intento_consulta_propia_v1(d,ajena,'denegada');
  RAISE EXCEPTION 'AUT29: cruce de correlación aceptado' USING ERRCODE='XX000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_meritos.registrar_intento_consulta_propia_v1('decision:rum04:inexistente',c,'denegada');
  RAISE EXCEPTION 'AUT29: decisión inexistente aceptada' USING ERRCODE='XX000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_meritos.registrar_intento_consulta_propia_v1(d,c,'concedida');
  RAISE EXCEPTION 'AUT29: resultado libre aceptado' USING ERRCODE='XX000';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_meritos.auditoria_operacion LIMIT 1;
  RAISE EXCEPTION 'AUT29: registrador lee auditoría' USING ERRCODE='XX000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $runtime$;
\endif
-- Este guion no acredita durabilidad o reinicio: conserva la base del ensayo.
ROLLBACK;
