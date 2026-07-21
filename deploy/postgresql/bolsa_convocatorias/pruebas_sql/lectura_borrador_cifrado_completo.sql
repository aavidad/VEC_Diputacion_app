-- Contrato estático de 000005. Se ejecuta después de 000004 y 000005, bajo
-- el propietario del esquema; no inserta ni revela datos de borradores.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
    firma regprocedure :=
        'vec_bolsa_convocatorias.obtener_borrador_v1(text,jsonb,jsonb,bytea,bytea)'::regprocedure;
    resultado text;
    permitido boolean;
BEGIN
    SELECT pg_get_function_result(firma) INTO resultado;
    IF resultado NOT LIKE '%aad_canonica bytea%'
       OR resultado NOT LIKE '%material_clave_envuelto bytea%'
       OR resultado NOT LIKE '%contenido_cifrado bytea%'
       OR resultado NOT LIKE '%atestacion_kms jsonb%'
       OR resultado NOT LIKE '%procedencia jsonb%' THEN
        RAISE EXCEPTION 'contrato de paquete criptografico incompleto: %', resultado;
    END IF;
    SELECT has_function_privilege(
        'vec_bolsa_convocatorias_ejecutor_consulta', firma, 'EXECUTE'
    ) INTO permitido;
    IF permitido IS NOT TRUE OR has_function_privilege(
        'vec_bolsa_convocatorias_proyector_gobierno', firma, 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'ACL de lectura de borrador no nominal';
    END IF;
END
$prueba$;

RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $frescura_exacta$
DECLARE
    verificada timestamptz := '2026-07-21T08:00:00.000000Z';
BEGIN
    IF vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
           verificada, verificada + interval '29.999999 seconds'
       ) IS NOT TRUE
       OR vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
              verificada, verificada + interval '30 seconds'
          ) IS NOT FALSE
       OR vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
              verificada, verificada - interval '0.000001 seconds'
          ) IS NOT FALSE THEN
        RAISE EXCEPTION
            'frescura no aplica [verificada_en,verificada_en+30s)';
    END IF;
END
$frescura_exacta$;
ROLLBACK;
