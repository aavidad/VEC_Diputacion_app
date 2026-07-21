-- Contrato estructural y ACL de la Fase A v2. Los valores temporales de una
-- preparacion valida se comprueban dentro de la propia funcion contra la fila
-- inmutable creada por v1.
DO $contrato$
DECLARE
    v2 oid := to_regprocedure(
        'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v2(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'
    );
    v1 oid := to_regprocedure(
        'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'
    );
    nombres text[];
    tipos oid[];
    modos "char"[];
    ultimo integer;
BEGIN
    IF v1 IS NULL OR v2 IS NULL THEN
        RAISE EXCEPTION 'fases A KMS v1/v2 incompletas';
    END IF;
    SELECT p.proargnames, p.proallargtypes, p.proargmodes
      INTO STRICT nombres, tipos, modos
      FROM pg_catalog.pg_proc p
     WHERE p.oid = v2 AND p.prosecdef
       AND p.proconfig @> ARRAY[
           'search_path=pg_catalog, pg_temp', 'TimeZone=UTC'
       ]::text[];
    ultimo := array_upper(nombres, 1);
    IF ultimo IS NULL OR nombres[ultimo] IS DISTINCT FROM 'preparada_en'
       OR tipos[array_upper(tipos, 1)] IS DISTINCT FROM
          'timestamp with time zone'::regtype::oid
       OR modos[array_upper(modos, 1)] IS DISTINCT FROM 't'::"char" THEN
        RAISE EXCEPTION 'Fase A v2 no termina en preparada_en timestamptz';
    END IF;
    IF has_function_privilege(
           'vec_bolsa_convocatorias_proyector_gobierno', v2, 'EXECUTE'
       ) IS NOT TRUE
       OR has_function_privilege(
           'vec_bolsa_convocatorias_proyector_gobierno', v1, 'EXECUTE'
       )
       OR has_function_privilege('public', v2, 'EXECUTE')
       OR has_function_privilege(
           'vec_bolsa_convocatorias_ejecutor_consulta', v2, 'EXECUTE'
       )
       OR has_function_privilege(
           'vec_bolsa_convocatorias_registrador_atestacion', v2, 'EXECUTE'
       )
       OR has_function_privilege(
           'vec_bolsa_convocatorias_verificador_recibo', v2, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'ACL de Fase A KMS v2 no es exclusiva';
    END IF;
END
$contrato$;
