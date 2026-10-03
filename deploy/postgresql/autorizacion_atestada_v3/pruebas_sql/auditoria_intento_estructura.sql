-- Ejecutar después de AD169 en el clon. No escribe ni consume capacidades.
DO $prueba$
DECLARE
    v_consumos bigint;
    v_sin_fk bigint;
    v_control numeric;
    v_max numeric;
BEGIN
    SELECT pg_catalog.count(*), pg_catalog.count(*) FILTER (
        WHERE decision_ref IS NULL OR efecto_ref IS NULL
           OR huella_efecto_sha256 IS NULL)
      INTO v_consumos,v_sin_fk
      FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
     WHERE tipo_registro='consumo_confirmado';
    IF v_sin_fk<>0 THEN
        RAISE EXCEPTION 'consumo histórico sin FK: esperado=0, actual=%',v_sin_fk;
    END IF;
    SELECT secuencia INTO STRICT v_control
      FROM vec_autorizacion_atestada_v3.control_cadena_auditoria
     WHERE control_id;
    SELECT coalesce(pg_catalog.max(secuencia),0) INTO v_max
      FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
    IF v_control<>v_max THEN
        RAISE EXCEPTION 'cabeza de auditoría: esperado=%, actual=%',v_max,v_control;
    END IF;
    IF pg_catalog.has_table_privilege(
           'vec_autorizacion_atestada_v3_registrador_intentos',
           'vec_autorizacion_atestada_v3.auditoria_consumo_v3','INSERT')
       OR pg_catalog.has_table_privilege(
           'vec_autorizacion_atestada_v3_registrador_intentos',
           'vec_autorizacion_atestada_v3.configuracion_runtime_intentos','SELECT')
       OR NOT pg_catalog.has_function_privilege(
           'vec_autorizacion_atestada_v3_registrador_intentos',
           'vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)',
           'EXECUTE') THEN
        RAISE EXCEPTION 'ACL registrador: esperado=sólo fachada, actual=divergente';
    END IF;
    IF vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1(
           'proceso_sintetico','administracion_privilegiada') IS NOT FALSE THEN
        RAISE EXCEPTION 'preflight DBA: esperado=false, actual=true';
    END IF;
    RAISE NOTICE 'estructura AD169 conforme: consumos=%, secuencia=%',
        v_consumos,v_control;
END
$prueba$;
