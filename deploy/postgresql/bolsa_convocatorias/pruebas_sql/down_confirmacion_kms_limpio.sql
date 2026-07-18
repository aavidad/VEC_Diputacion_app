-- Se ejecuta tras 000004 down y antes de retirar 000003.
DO $down_confirmacion_kms_limpio$
DECLARE
    nombre text;
BEGIN
    IF to_regclass(
           'vec_bolsa_convocatorias.borrador_convocatoria_version'
       ) IS NULL THEN
        RAISE EXCEPTION '000004 down elimino una tabla de 000003';
    END IF;
    FOREACH nombre IN ARRAY ARRAY[
        'preparacion_confirmacion_kms_borrador',
        'cifrado_kms_borrador','acreditacion_kms_borrador'
    ] LOOP
        IF to_regclass('vec_bolsa_convocatorias.' || nombre) IS NOT NULL THEN
            RAISE EXCEPTION '000004 down dejo la tabla %', nombre;
        END IF;
    END LOOP;
    FOREACH nombre IN ARRAY ARRAY[
        'preparar_confirmacion_borrador_v1',
        'verificar_recibo_borrador_v1',
        'identidad_runtime_verificador_recibo_valida',
        'presupuesto_confirmacion_kms_borrador_v1',
        'base64url_sin_relleno_valido','procedencia_borrador_valida',
        'perfil_cifrado_borrador_valido','firma_evidencia_borrador_valida',
        'huella_envoltura_clave_borrador_v1',
        'huella_sobre_aead_borrador_v1','aad_canonica_borrador_v1',
        'instante_rfc3339nano_borrador_v1',
        'cuerpo_recibo_canonico_borrador_v1',
        'acreditacion_kms_canonica_borrador_v1',
        'atestacion_kms_preimagen_borrador_v1',
        'firma_base64url_borrador_v1',
        'revalidacion_kms_preimagen_borrador_v1',
        'evidencia_cifrado_kms_borrador_valida',
        'validar_actualizacion_acreditacion_diario_v1',
        'validar_consumo_preparacion_kms_v1',
        'exigir_cierre_preparacion_kms_v1',
        'exigir_acreditacion_kms_durable_v1'
    ] LOOP
        IF EXISTS (
            SELECT 1
              FROM pg_catalog.pg_proc AS p
              JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
               AND p.proname = nombre
        ) THEN
            RAISE EXCEPTION '000004 down dejo la funcion %', nombre;
        END IF;
    END LOOP;
    IF to_regprocedure(
           'vec_bolsa_convocatorias.confirmar_borrador_v1(text,jsonb,bytea)'
       ) IS NOT NULL
       OR to_regprocedure(
           'vec_bolsa_convocatorias.confirmar_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL THEN
        RAISE EXCEPTION '000004 down no restauro el stub NO-GO de 000003';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger AS g
          JOIN pg_catalog.pg_class AS t ON t.oid = g.tgrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND g.tgname IN (
               'borrador_version_exige_acreditacion_kms',
               'auditoria_borrador_exige_acreditacion_kms',
               'outbox_borrador_exige_acreditacion_kms',
               'diario_borrador_exige_acreditacion_kms',
               'preparacion_confirmacion_kms_consumo',
               'preparacion_confirmacion_kms_debe_cerrarse',
               'cifrado_kms_borrador_inmutable',
               'cifrado_kms_borrador_no_truncar',
               'acreditacion_kms_borrador_inmutable',
               'acreditacion_kms_borrador_no_truncar'
           )
           AND NOT g.tgisinternal
    ) THEN
        RAISE EXCEPTION '000004 down dejo disparadores KMS';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger AS g
          JOIN pg_catalog.pg_class AS t ON t.oid = g.tgrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
          JOIN pg_catalog.pg_proc AS p ON p.oid = g.tgfoid
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND t.relname = 'diario_borrador_version'
           AND g.tgname = 'diario_borrador_version_inmutable'
           AND p.proname = 'rechazar_mutacion_inmutable'
           AND NOT g.tgisinternal
    ) THEN
        RAISE EXCEPTION '000004 down no restauro la inmutabilidad de 000003';
    END IF;
END
$down_confirmacion_kms_limpio$;
