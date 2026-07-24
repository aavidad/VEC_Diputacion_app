SET search_path = pg_catalog;

DO $acl$
DECLARE
    funcion regprocedure :=
      'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'::regprocedure;
BEGIN
    IF pg_catalog.has_function_privilege('public', funcion, 'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_contexto_actor_v1_runtime', funcion, 'EXECUTE'
       )
       OR NOT pg_catalog.has_function_privilege(
           'vec_contexto_actor_v1_propietario', funcion, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'ACL inesperado en acreditacion de uso V2';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_propietario'
    ) THEN
        RAISE EXCEPTION 'la prueba aislada no debe crear roles de autorizacion';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
         WHERE p.oid = funcion
           AND (p.proowner <> 'vec_contexto_actor_v1_propietario'::regrole
                OR p.prosecdef IS NOT TRUE
                OR p.provolatile <> 'v'
                OR p.proconfig IS DISTINCT FROM
                   ARRAY['search_path=pg_catalog']::text[])
    ) THEN
        RAISE EXCEPTION 'propiedad o cierre SECURITY DEFINER inesperados';
    END IF;
    IF pg_catalog.has_table_privilege(
           'public',
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2',
           'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN'
       )
       OR pg_catalog.has_table_privilege(
           'vec_contexto_actor_v1_runtime',
           'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2',
           'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN'
       )
       OR pg_catalog.has_function_privilege(
           'public',
           'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'public',
           'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contexto_actor_v1_runtime',
           'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contexto_actor_v1_runtime',
           'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'cerrojo MVCC visible para PUBLIC o runtime';
    END IF;
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger AS t
         WHERE t.tgrelid = ANY (ARRAY[
             'vec_contexto_actor_v1.proyeccion_cuenta_actual'::regclass,
             'vec_contexto_actor_v1.persona_actual'::regclass,
             'vec_contexto_actor_v1.perfil_actual'::regclass,
             'vec_contexto_actor_v1.vinculo_contexto_actual'::regclass,
             'vec_contexto_actor_v1.vinculo_referencia_actual'::regclass
         ])
           AND t.tgname IN (
             'puntero_actual_no_truncable_v2',
             'serializar_mutacion_punteros_actuales_v2',
             'avanzar_generacion_punteros_actuales_v2'
           )
           AND t.tgenabled = 'O'
           AND NOT t.tgisinternal
    ) <> 15 THEN
        RAISE EXCEPTION 'los cinco punteros no tienen sus tres triggers activos';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
         WHERE control_id = true AND generacion > 0
    ) THEN
        RAISE EXCEPTION 'la generacion no avanzo con los fixtures actuales';
    END IF;
END
$acl$;

DO $truncados_prohibidos$
DECLARE
    tabla text;
    generacion_anterior numeric;
    generacion_posterior numeric;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'proyeccion_cuenta_actual', 'persona_actual', 'perfil_actual',
        'vinculo_contexto_actual', 'vinculo_referencia_actual'
    ] LOOP
        SELECT generacion
          INTO STRICT generacion_anterior
          FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
         WHERE control_id = true;
        BEGIN
            EXECUTE pg_catalog.format(
                'TRUNCATE TABLE vec_contexto_actor_v1.%I', tabla
            );
            RAISE EXCEPTION 'TRUNCATE aceptado en %', tabla;
        EXCEPTION
            WHEN SQLSTATE '55000' THEN NULL;
        END;
        SELECT generacion
          INTO STRICT generacion_posterior
          FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
         WHERE control_id = true;
        IF generacion_posterior IS DISTINCT FROM generacion_anterior THEN
            RAISE EXCEPTION 'TRUNCATE rechazado altero generacion en %', tabla;
        END IF;
    END LOOP;
END
$truncados_prohibidos$;

BEGIN TRANSACTION ISOLATION LEVEL READ COMMITTED;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $aislamiento$
DECLARE r record;
BEGIN
    SELECT * INTO STRICT r
      FROM vec_contexto_actor_v1.registros_contexto
     WHERE registro_contexto_ref =
           'rca_acredita_000000000000000000000000';
    BEGIN
        PERFORM vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
          r.registro_contexto_ref, 'vec.contexto-actor.vinculado.v2',
          r.huella_sha256, r.manifiesto_procedencia_huella_sha256,
          r.autoridad_efectiva,
          'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
          'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
          'prf_sintetico_cccccccccccccccccccccccc', 2,
          'vca_sintetico_dddddddddddddddddddddddd', 2,
          'certificado', 'alto', pg_catalog.clock_timestamp(),
          pg_catalog.clock_timestamp() + interval '10 minutes'
        );
        RAISE EXCEPTION 'READ COMMITTED fue aceptado';
    EXCEPTION WHEN SQLSTATE '25000' THEN NULL;
    END;
END
$aislamiento$;
COMMIT;

BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ ONLY;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $solo_lectura$
BEGIN
    BEGIN
        PERFORM vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
          'rca_acredita_000000000000000000000000',
          'vec.contexto-actor.vinculado.v2', repeat('a',64), repeat('b',64),
          'autoridad_maestra_acreditada',
          'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
          'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
          'prf_sintetico_cccccccccccccccccccccccc', 2,
          'vca_sintetico_dddddddddddddddddddddddd', 2,
          'certificado', 'alto', pg_catalog.clock_timestamp(),
          pg_catalog.clock_timestamp() + interval '10 minutes'
        );
        RAISE EXCEPTION 'SERIALIZABLE READ ONLY fue aceptado';
    EXCEPTION WHEN SQLSTATE '25000' THEN NULL;
    END;
END
$solo_lectura$;
COMMIT;

BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $contrato$
DECLARE
    r record;
    emitida timestamptz;
    valida_hasta timestamptz;
    acreditada_en timestamptz;
    huella_forjada text;
BEGIN
    SELECT * INTO STRICT r
      FROM vec_contexto_actor_v1.registros_contexto
     WHERE registro_contexto_ref =
           'rca_acredita_000000000000000000000000';
    emitida := pg_catalog.clock_timestamp();
    valida_hasta := emitida + interval '10 minutes';

    acreditada_en :=
      vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        r.registro_contexto_ref, 'vec.contexto-actor.vinculado.v2',
        r.huella_sha256, r.manifiesto_procedencia_huella_sha256,
        r.autoridad_efectiva,
        'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
        'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
        'prf_sintetico_cccccccccccccccccccccccc', 2,
        'vca_sintetico_dddddddddddddddddddddddd', 2,
        'certificado', 'alto', emitida, valida_hasta
      );
    IF acreditada_en IS NULL OR acreditada_en < emitida
       OR acreditada_en >= valida_hasta THEN
        RAISE EXCEPTION 'rca_ V2 exacto no fue acreditado';
    END IF;

    IF vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
         r.registro_contexto_ref, 'vec.contexto-actor.vinculado.v1',
         r.huella_sha256, r.manifiesto_procedencia_huella_sha256,
         r.autoridad_efectiva,
         'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
         'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
         'prf_sintetico_cccccccccccccccccccccccc', 2,
         'vca_sintetico_dddddddddddddddddddddddd', 2,
         'certificado', 'alto', emitida, valida_hasta
       ) IS NOT NULL
       OR vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
         r.registro_contexto_ref, 'vec.contexto-actor.vinculado.v2',
         repeat('0',64), r.manifiesto_procedencia_huella_sha256,
         r.autoridad_efectiva,
         'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
         'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
         'prf_sintetico_cccccccccccccccccccccccc', 2,
         'vca_sintetico_dddddddddddddddddddddddd', 2,
         'certificado', 'alto', emitida, valida_hasta
       ) IS NOT NULL
       OR vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
         r.registro_contexto_ref, 'vec.contexto-actor.vinculado.v2',
         r.huella_sha256, r.manifiesto_procedencia_huella_sha256,
         'no_autoritativa',
         'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 3,
         'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
         'prf_sintetico_cccccccccccccccccccccccc', 2,
         'vca_sintetico_dddddddddddddddddddddddd', 2,
         'certificado', 'alto', emitida, valida_hasta
       ) IS NOT NULL
       OR vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
         'rca_inexistente_000000000000000000000',
         'vec.contexto-actor.vinculado.v2',
         r.huella_sha256, r.manifiesto_procedencia_huella_sha256,
         r.autoridad_efectiva,
         'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
         'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
         'prf_sintetico_cccccccccccccccccccccccc', 2,
         'vca_sintetico_dddddddddddddddddddddddd', 2,
         'certificado', 'alto', emitida, valida_hasta
       ) IS NOT NULL THEN
        RAISE EXCEPTION 'una adulteracion de esquema, huella, autoridad, version o rca_ fue aceptada';
    END IF;

    huella_forjada := pg_catalog.encode(
        pg_catalog.sha256(pg_catalog.convert_to('{}', 'UTF8')), 'hex'
    );
    INSERT INTO vec_contexto_actor_v1.registros_contexto(
      operacion_ref, registro_contexto_ref, cuenta_ref, perfil_ref,
      metodo, garantia, solicitado_en, resuelto_en,
      representacion_canonica, huella_sha256,
      manifiesto_procedencia_canonico,
      manifiesto_procedencia_huella_sha256, autoridad_efectiva
    ) VALUES (
      'oca_forjada_000000000000000000000000',
      'rca_forjada_000000000000000000000000',
      r.cuenta_ref, r.perfil_ref, r.metodo, r.garantia,
      emitida, emitida, pg_catalog.convert_to('{}','UTF8'), huella_forjada,
      r.manifiesto_procedencia_canonico,
      r.manifiesto_procedencia_huella_sha256, r.autoridad_efectiva
    );
    IF vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
         'rca_forjada_000000000000000000000000',
         'vec.contexto-actor.vinculado.v2', huella_forjada,
         r.manifiesto_procedencia_huella_sha256, r.autoridad_efectiva,
         'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa', 2,
         'per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb', 2,
         'prf_sintetico_cccccccccccccccccccccccc', 2,
         'vca_sintetico_dddddddddddddddddddddddd', 2,
         'certificado', 'alto', emitida, valida_hasta
       ) IS NOT NULL THEN
        RAISE EXCEPTION 'bytes autoconsistentes pero no canonicos fueron aceptados';
    END IF;
END
$contrato$;
COMMIT;
