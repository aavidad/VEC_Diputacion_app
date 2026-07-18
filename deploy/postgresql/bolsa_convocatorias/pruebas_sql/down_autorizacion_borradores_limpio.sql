-- Se ejecuta tras retirar exclusivamente la migracion nominal V2 y antes de
-- retirar la autorizacion V1 de consulta exacta.
DO $down_autorizacion_limpio$
BEGIN
    IF to_regprocedure(
           'vec_autorizacion.revalidar_decision_borrador_convocatorias_v2(jsonb,bytea,bytea,text,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION '000002 autorizacion down dejo la frontera V2';
    END IF;
    IF to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_convocatorias_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION '000002 autorizacion down elimino la frontera V1';
    END IF;
END
$down_autorizacion_limpio$;
