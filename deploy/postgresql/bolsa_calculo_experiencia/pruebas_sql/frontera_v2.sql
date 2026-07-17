-- Prueba adversaria de la frontera estrecha sobre la tabla V2 real. Las filas
-- sinteticas se insertan como superusuario tras suspender solo CHECK/FK dentro
-- de esta transaccion; el registro V2 completo conserva sus propias pruebas.
BEGIN;
SET LOCAL timezone = 'UTC';

DO $retirar_checks$
DECLARE
    restriccion record;
BEGIN
    FOR restriccion IN
        SELECT conname FROM pg_catalog.pg_constraint
         WHERE conrelid =
             'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'::regclass
           AND contype = 'c'
    LOOP
        EXECUTE format(
            'ALTER TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2 DROP CONSTRAINT %I',
            restriccion.conname
        );
    END LOOP;
END
$retirar_checks$;
SET LOCAL session_replication_role = replica;

DO $prueba$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    recurso_lectura text := 'fuente:' || repeat('1', 64);
    recurso_escritura text := 'calculo-oficial:' || repeat('3', 64);
    recurso_rectificacion text :=
        'rectificacion-calculo-oficial:' || repeat('4', 64);
    correlacion_lectura text := 'correlacion_11111111111111111111111111111111';
    correlacion_escritura text := 'correlacion_22222222222222222222222222222222';
    correlacion_rectificacion text := 'correlacion_33333333333333333333333333333333';
    contexto_lectura text := repeat('5', 64);
    contexto_escritura text := repeat('6', 64);
    contexto_rectificacion text := repeat('7', 64);
    decision_lectura bytea := convert_to('decision-lectura', 'UTF8');
    decision_escritura bytea := convert_to('decision-escritura', 'UTF8');
    decision_rectificacion bytea := convert_to('decision-rectificacion', 'UTF8');
    vinculo_interno jsonb := jsonb_build_object(
        'metodo_observado', 'kerberos_ad',
        'superficie', 'interna_corporativa',
        'cuenta_privilegiada', false,
        'garantia_observada', 'alto'
    );
BEGIN
    INSERT INTO vec_autorizacion.decision_autorizacion_solicitud_ligada_v2(
        decision_ref, huella_decision_sha256, decision_canonica,
        documento_v2, documento_comun, principal_id, perfil_activo_ref,
        accion, recurso_ref, modulo_id, tipo_recurso,
        contexto_recurso_huella_sha256, finalidad, correlacion_ref,
        solicitud_huella_sha256, motivo_huella_sha256, motivo_canonico,
        motivo_catalogo_id, motivo_catalogo_version,
        motivo_catalogo_huella_sha256, motivo_entrada_clave,
        asignacion_ref, version_rol_ref, control_vigencia_version_rol_ref,
        control_vigencia_version_rol_revision,
        emitida_en, valida_hasta, registrada_en
    ) VALUES
    (
        'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
        decision_lectura, jsonb_build_object(
            'concedida', true,
            'campos_permitidos',
                '["fuente_reglas","instantanea_experiencia","prueba_procedencia"]'::jsonb,
            'obligaciones', '[]'::jsonb,
            'garantia_minima', 'alto',
            'vinculo_autenticacion_actor', vinculo_interno
        ), '{}'::jsonb, 'principal:prueba', 'perfil:prueba',
        'bolsa.calculo_experiencia.fuente.leer', recurso_lectura, 'bolsa',
        'fuente_calculo_experiencia', contexto_lectura,
        'calculo_oficial_experiencia', correlacion_lectura,
        repeat('8', 64), repeat('9', 64), convert_to('motivo', 'UTF8'),
        'motivos.calculo', 1, repeat('a', 64),
        'motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        'asignacion:prueba', 'rol:prueba', 'rol:prueba', 1,
        ahora - interval '1 second', ahora + interval '1 minute', ahora
    ),
    (
        'decision:calculo:escritura', encode(sha256(decision_escritura), 'hex'),
        decision_escritura, jsonb_build_object(
            'concedida', true,
            'campos_permitidos',
                '["auditoria","resultado_canonico","salida_eventos"]'::jsonb,
            'obligaciones', '[]'::jsonb,
            'garantia_minima', 'alto',
            'vinculo_autenticacion_actor', vinculo_interno
        ), '{}'::jsonb, 'principal:prueba', 'perfil:prueba',
        'bolsa.calculo_experiencia.oficial.confirmar', recurso_escritura,
        'bolsa', 'calculo_experiencia_oficial', contexto_escritura,
        'confirmacion_calculo_oficial_experiencia', correlacion_escritura,
        repeat('b', 64), repeat('c', 64), convert_to('motivo', 'UTF8'),
        'motivos.calculo', 1, repeat('d', 64),
        'motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        'asignacion:prueba', 'rol:prueba', 'rol:prueba', 1,
        ahora - interval '1 second', ahora + interval '1 minute', ahora
    ),
    (
        'decision:calculo:rectificacion',
        encode(sha256(decision_rectificacion), 'hex'),
        decision_rectificacion, jsonb_build_object(
            'concedida', true,
            'campos_permitidos',
                '["auditoria","resultado_canonico","salida_eventos"]'::jsonb,
            'obligaciones', '[]'::jsonb,
            'garantia_minima', 'alto',
            'vinculo_autenticacion_actor', vinculo_interno
        ), '{}'::jsonb, 'principal:prueba', 'perfil:prueba',
        'bolsa.calculo_experiencia.oficial.rectificar',
        recurso_rectificacion, 'bolsa',
        'rectificacion_calculo_experiencia_oficial', contexto_rectificacion,
        'rectificacion_calculo_oficial_experiencia',
        correlacion_rectificacion, repeat('e', 64), repeat('f', 64),
        convert_to('motivo', 'UTF8'), 'motivos.calculo', 1, repeat('1', 64),
        'motivo_cccccccccccccccccccccccccccccccc',
        'asignacion:prueba', 'rol:prueba', 'rol:prueba', 1,
        ahora - interval '1 second', ahora + interval '1 minute', ahora
    );

    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'interno_alto', 'calculo_inicial',
           correlacion_lectura, recurso_lectura, contexto_lectura
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:escritura',
           encode(sha256(decision_escritura), 'hex'),
           'escritura_resultado', 'interno_alto', 'calculo_inicial',
           correlacion_escritura, recurso_escritura, contexto_escritura
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:rectificacion',
           encode(sha256(decision_rectificacion), 'hex'),
           'escritura_resultado', 'interno_alto', 'rectificacion',
           correlacion_rectificacion, recurso_rectificacion,
           contexto_rectificacion
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'la frontera rechazo decisiones validas';
    END IF;

    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:escritura',
           encode(sha256(decision_escritura), 'hex'),
           'escritura_resultado', 'interno_alto', 'rectificacion',
           correlacion_escritura, recurso_escritura, contexto_escritura
       ) IS TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'externo_ordinario', 'calculo_inicial',
           correlacion_lectura, recurso_lectura, contexto_lectura
       ) IS TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'interno_alto', 'calculo_inicial',
           'correlacion_ffffffffffffffffffffffffffffffff',
           recurso_lectura, contexto_lectura
       ) IS TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'interno_alto', 'calculo_inicial',
           correlacion_lectura,
           'fuente:' || repeat('0', 64),
           contexto_lectura
       ) IS TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:rectificacion',
           encode(sha256(decision_rectificacion), 'hex'),
           'escritura_resultado', 'externo_ordinario', 'rectificacion',
           correlacion_rectificacion, recurso_rectificacion,
           contexto_rectificacion
       ) IS TRUE
       OR vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'interno_alto', 'calculo_inicial',
           correlacion_lectura, recurso_lectura, repeat('0', 64)
       ) IS TRUE THEN
        RAISE EXCEPTION 'la frontera acepto un vinculo alterado';
    END IF;

    UPDATE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
       SET documento_v2 = jsonb_set(
           documento_v2,
           '{vinculo_autenticacion_actor,garantia_observada}',
           '"sustancial"'::jsonb
       )
     WHERE decision_ref = 'decision:calculo:escritura';
    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:escritura',
           encode(sha256(decision_escritura), 'hex'),
           'escritura_resultado', 'interno_alto', 'calculo_inicial',
           correlacion_escritura, recurso_escritura, contexto_escritura
       ) IS TRUE THEN
        RAISE EXCEPTION 'la frontera interna acepto garantia no alta';
    END IF;
    UPDATE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
       SET documento_v2 = jsonb_set(
           documento_v2,
           '{vinculo_autenticacion_actor,garantia_observada}',
           '"alto"'::jsonb
       )
     WHERE decision_ref = 'decision:calculo:escritura';

    UPDATE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
       SET documento_v2 = jsonb_set(
           documento_v2,
           '{vinculo_autenticacion_actor,metodo_observado}',
           '"demo"'::jsonb
       )
     WHERE decision_ref = 'decision:calculo:lectura';
    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'interno_alto', 'calculo_inicial',
           correlacion_lectura, recurso_lectura, contexto_lectura
       ) IS TRUE THEN
        RAISE EXCEPTION 'la frontera acepto metodo demo';
    END IF;

    UPDATE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
       SET documento_v2 = jsonb_set(
           jsonb_set(
               jsonb_set(
                   documento_v2,
                   '{garantia_minima}',
                   '"sustancial"'::jsonb
               ),
               '{vinculo_autenticacion_actor,metodo_observado}',
               '"certificado"'::jsonb
           ),
           '{vinculo_autenticacion_actor}',
           jsonb_build_object(
               'metodo_observado', 'certificado',
               'superficie', 'externa_personal',
               'cuenta_privilegiada', false,
               'garantia_observada', 'sustancial'
           )
       )
     WHERE decision_ref = 'decision:calculo:lectura';
    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'externo_ordinario', 'calculo_inicial',
           correlacion_lectura, recurso_lectura, contexto_lectura
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'la frontera rechazo superficie externa valida';
    END IF;

    UPDATE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
       SET documento_v2 = jsonb_set(
           jsonb_set(
               documento_v2,
               '{garantia_minima}',
               '"bajo"'::jsonb
           ),
           '{vinculo_autenticacion_actor,garantia_observada}',
           '"bajo"'::jsonb
       )
     WHERE decision_ref = 'decision:calculo:lectura';
    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'externo_ordinario', 'calculo_inicial',
           correlacion_lectura, recurso_lectura, contexto_lectura
       ) IS TRUE THEN
        RAISE EXCEPTION 'la frontera acepto garantia externa baja';
    END IF;

    UPDATE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
       SET documento_v2 = jsonb_set(
           jsonb_set(
               documento_v2,
               '{garantia_minima}',
               '"sustancial"'::jsonb
           ),
           '{vinculo_autenticacion_actor,garantia_observada}',
           '"sustancial"'::jsonb
       )
     WHERE decision_ref = 'decision:calculo:lectura';

    UPDATE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
       SET valida_hasta = ahora - interval '1 microsecond'
     WHERE decision_ref = 'decision:calculo:lectura';
    IF vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
           'decision:calculo:lectura', encode(sha256(decision_lectura), 'hex'),
           'lectura_fuentes', 'externo_ordinario', 'calculo_inicial',
           correlacion_lectura, recurso_lectura, contexto_lectura
       ) IS TRUE THEN
        RAISE EXCEPTION 'la frontera acepto decision caducada';
    END IF;
END
$prueba$;

ROLLBACK;
