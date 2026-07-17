BEGIN;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    oid_recibo oid :=
        'vec_bolsa_reglas_baremo.recibo_cambio_atestado_v2'::regclass;
    oid_confirmar oid :=
        'vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'::regprocedure;
    oid_reconciliar oid :=
        'vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(text,text,text,text,text,text)'::regprocedure;
    oid_central oid :=
        'vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'::regprocedure;
    oid_central_reconciliar oid :=
        'vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(text,text,text,text,text)'::regprocedure;
    oid_lector_fresco oid :=
        'vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(text,text,text,text,text,text,text)'::regprocedure;
    rol text;
BEGIN
    IF encode(sha256(convert_to(
           '{"ambitos":{"convocatoria_ref":"con_11111111111111111111111111111111","expediente_ref":"exp_22222222222222222222222222222222"},"atributos":{}}',
           'UTF8'
       )), 'hex') <>
       '5a2626eed065e5d317177a1ea312b393e523690c24c852f6de8c2abbc2bca2fb' THEN
        RAISE EXCEPTION 'vector Go-SQL del contexto de recurso divergente';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_class
         WHERE oid = oid_recibo AND relrowsecurity AND relforcerowsecurity
    ) OR (SELECT count(*) FROM pg_catalog.pg_policy
           WHERE polrelid = oid_recibo) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_trigger
            WHERE tgrelid = oid_recibo AND NOT tgisinternal
              AND tgname = 'impedir_mutacion'
       ) OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_trigger
            WHERE tgrelid = oid_recibo AND NOT tgisinternal
              AND tgname = 'impedir_truncado'
       ) THEN
        RAISE EXCEPTION 'recibo V2 sin RLS o inmutabilidad completa';
    END IF;

    FOREACH rol IN ARRAY ARRAY[
        'public',
        'vec_bolsa_reglas_baremo_ejecutor_gobierno',
        'vec_bolsa_reglas_baremo_ejecutor_consulta',
        'vec_bolsa_reglas_baremo_publicador_outbox',
        'vec_autorizacion_atestada_v2_consumidor'
    ] LOOP
        IF has_table_privilege(rol, oid_recibo, 'SELECT,INSERT,UPDATE,DELETE')
           OR has_function_privilege(rol, oid_confirmar, 'EXECUTE')
           OR has_function_privilege(rol, oid_reconciliar, 'EXECUTE') THEN
            RAISE EXCEPTION 'superficie V2 abierta a %', rol;
        END IF;
    END LOOP;

    IF has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_central, 'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor',
           oid_central_reconciliar, 'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor',
           oid_lector_fresco, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'el grupo generico conserva una puerta VEC cruda';
    END IF;
    IF NOT has_function_privilege(
           'vec_bolsa_reglas_baremo_propietario', oid_central, 'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           oid_central_reconciliar, 'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           oid_lector_fresco, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'el propietario modular no tiene el puente estrecho';
    END IF;
    IF has_table_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           'vec_autorizacion_atestada_v2.atestacion_decision_v2',
           'SELECT,INSERT,UPDATE,DELETE'
       ) OR has_table_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           'vec_autorizacion_atestada_v2.consumo_decision_v2',
           'SELECT,INSERT,UPDATE,DELETE'
       ) OR has_table_privilege(
           'vec_bolsa_reglas_baremo_propietario',
           'vec_autorizacion_atestada_v2.auditoria_consumo_v2',
           'SELECT,INSERT,UPDATE,DELETE'
       ) THEN
        RAISE EXCEPTION 'el propietario modular obtuvo DML central';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid = oid_recibo AND contype = 'f'
           AND confrelid =
               'vec_autorizacion_atestada_v2.atestacion_decision_v2'::regclass
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid = oid_recibo AND contype = 'f'
           AND confrelid =
               'vec_autorizacion_atestada_v2.consumo_decision_v2'::regclass
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid = oid_recibo AND contype = 'f'
           AND confrelid =
               'vec_autorizacion_atestada_v2.auditoria_consumo_v2'::regclass
    ) THEN
        RAISE EXCEPTION 'recibo V2 sin vinculos FK centrales';
    END IF;

    IF has_function_privilege(
           'vec_bolsa_reglas_baremo_ejecutor_gobierno',
           'vec_bolsa_reglas_baremo.confirmar_cambio_v1(jsonb,jsonb,bytea,bytea,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_reglas_baremo_ejecutor_consulta',
           'vec_bolsa_reglas_baremo.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'V1 fue abierta al instalar V2';
    END IF;
END
$prueba$;

ROLLBACK;
