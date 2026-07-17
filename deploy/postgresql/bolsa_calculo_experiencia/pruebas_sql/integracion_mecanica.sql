-- Prueba la mecanica local en una transaccion revertida. Sustituye solo la
-- autoridad V2 y retira temporalmente sus dos FK; acl_y_modelo.sql comprueba
-- por separado que la instalacion real conserva ambas fronteras.
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;

DO $retirar_fk_autorizacion$
DECLARE
    restriccion record;
BEGIN
    FOR restriccion IN
        SELECT conname FROM pg_catalog.pg_constraint
         WHERE conrelid =
             'vec_bolsa_calculo_experiencia.consumo_autorizaciones'::regclass
           AND confrelid =
             'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'::regclass
           AND contype = 'f'
    LOOP
        EXECUTE format(
            'ALTER TABLE vec_bolsa_calculo_experiencia.consumo_autorizaciones DROP CONSTRAINT %I',
            restriccion.conname
        );
    END LOOP;
END
$retirar_fk_autorizacion$;

CREATE OR REPLACE FUNCTION
vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
    p_decision_ref text,
    p_huella_decision_sha256 text,
    p_tipo text,
    p_perfil_proteccion text,
    p_tipo_efecto text,
    p_correlacion_ref text,
    p_recurso_ref text,
    p_contexto_recurso_huella_sha256 text
)
RETURNS boolean
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $prueba$ SELECT true $prueba$;

SET LOCAL ROLE vec_bolsa_calculo_experiencia_aplicacion;

DO $historia$
DECLARE
    instante_1 constant text := '2026-07-17T10:00:00.000000Z';
    instante_2 constant text := '2026-07-17T10:00:01.000000Z';
    resultado bytea := convert_to(
        '{"esquema":"vec.bolsa.resultado_experiencia.v1","estado":"completado"}',
        'UTF8'
    );
    clave bytea := convert_to(
        '{"esquema":"vec.bolsa.calculo-experiencia-oficial.clave-efecto.v1"}',
        'UTF8'
    );
    intencion bytea := convert_to(
        '{"esquema":"vec.bolsa.calculo-experiencia-oficial.intencion-resultado.v1"}',
        'UTF8'
    );
    selector bytea := convert_to(
        '{"esquema":"vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1"}',
        'UTF8'
    );
    recibo bytea := convert_to(
        '{"esquema":"vec.bolsa.calculo-experiencia-oficial.recibo.v1"}',
        'UTF8'
    );
    auditoria_1 bytea := convert_to('auditoria-intento-1', 'UTF8');
    auditoria_2 bytea := convert_to('auditoria-intento-2', 'UTF8');
    evento bytea := convert_to('resultado-confirmado', 'UTF8');
    h_resultado text := encode(sha256(resultado), 'hex');
    h_clave text := encode(sha256(clave), 'hex');
    h_intencion text := encode(sha256(intencion), 'hex');
    h_selector text := encode(sha256(selector), 'hex');
    h_recibo text := encode(sha256(recibo), 'hex');
    h_auditoria_1 text := encode(sha256(auditoria_1), 'hex');
    h_auditoria_2 text := encode(sha256(auditoria_2), 'hex');
    intento_1 text := 'correlacion_11111111111111111111111111111111';
    intento_2 text := 'correlacion_22222222222222222222222222222222';
    indice text := repeat('1', 64);
BEGIN
    INSERT INTO vec_bolsa_calculo_experiencia.resultado_oficial(
        tenant_id, resultado_ref, esquema_resultado, resultado_canonico,
        huella_resultado_sha256, esquema_clave_semantica,
        clave_semantica_publica, huella_clave_semantica_sha256,
        generacion_clave_hmac, indice_efecto_hmac_sha256,
        esquema_selector_fuente, selector_fuente_canonico,
        huella_selector_fuente_sha256,
        fuente_ref, fuente_version, huella_fuente_sha256,
        reglas_ref, reglas_version, huella_reglas_contenido_sha256,
        reglas_revision, huella_reglas_estado_sha256,
        convocatoria_ref, convocatoria_version, huella_convocatoria_sha256,
        entrada_ref, entrada_version, huella_entrada_sha256,
        huella_contenido_entrada_sha256,
        sujeto_ref, sujeto_version, huella_sujeto_sha256, tipo_efecto,
        estado, fase, intento_nominal_ref, recibo_ref, outbox_ref, creada_en
    ) VALUES (
        'diputacion_granada', 'resultado:experiencia:1',
        'vec.bolsa.resultado_experiencia.v1', resultado, h_resultado,
        'vec.bolsa.calculo-experiencia-oficial.clave-efecto.v1',
        clave, h_clave, 1, indice,
        'vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1',
        selector, h_selector,
        'evidencia:fuente:1', 1, repeat('2', 64),
        'reglas:experiencia:1', 1, repeat('3', 64), 1, repeat('4', 64),
        'convocatoria:1', 1, repeat('5', 64),
        'entrada:experiencia:1', 1, repeat('6', 64), repeat('7', 64),
        'sujeto:pseudonimizado:1', 1, repeat('8', 64), 'calculo_inicial',
        'completado', 'completado', intento_1,
        'recibo:experiencia:1', 'outbox:experiencia:1', instante_1
    );

    INSERT INTO vec_bolsa_calculo_experiencia.intento(
        tenant_id, intento_ref, resultado_ref, desenlace,
        esquema_intencion, intencion_canonica, huella_intencion_sha256,
        generacion_clave_hmac, indice_efecto_hmac_sha256,
        huella_clave_semantica_sha256, huella_resultado_sha256,
        tipo_efecto, estado, fase, consumo_lectura_ref, consumo_escritura_ref,
        recibo_ref, auditoria_ref, iniciado_en, confirmado_en
    ) VALUES (
        'diputacion_granada', intento_1, 'resultado:experiencia:1', 'creada',
        'vec.bolsa.calculo-experiencia-oficial.intencion-resultado.v1',
        intencion, h_intencion, 1, indice, h_clave, h_resultado,
        'calculo_inicial',
        'completado', 'completado', 'consumo:lectura:1',
        'consumo:escritura:1', 'recibo:experiencia:1',
        'auditoria:experiencia:1', instante_1, instante_1
    );

    INSERT INTO vec_bolsa_calculo_experiencia.consumo_autorizaciones(
        tenant_id, intento_ref, resultado_ref, perfil_proteccion, tipo_efecto,
        consumo_lectura_ref, consumo_lectura_version,
        huella_consumo_lectura_sha256,
        consumo_prueba_ref, consumo_prueba_version,
        huella_consumo_prueba_sha256,
        decision_lectura_ref, huella_decision_lectura_sha256,
        correlacion_lectura_ref, recurso_lectura_ref,
        consumo_escritura_ref, decision_escritura_ref,
        huella_decision_escritura_sha256,
        correlacion_escritura_ref, recurso_escritura_ref,
        contexto_recurso_lectura_huella_sha256,
        contexto_recurso_escritura_huella_sha256,
        huella_selector_fuente_sha256, huella_intencion_sha256,
        huella_efecto_sha256,
        lectura_consumida_en, escritura_consumida_en
    ) VALUES (
        'diputacion_granada', intento_1, 'resultado:experiencia:1',
        'interno_alto', 'calculo_inicial',
        'consumo:lectura:1', 1, repeat('9', 64),
        'consumo:prueba:1', 1, repeat('a', 64),
        'decision:lectura:1', repeat('b', 64), intento_1,
        'fuente:' || h_selector, 'consumo:escritura:1',
        'decision:escritura:1', repeat('c', 64),
        'correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        'calculo-oficial:' || h_intencion,
        repeat('d', 64), repeat('e', 64),
        h_selector, h_intencion, h_resultado, instante_1, instante_1
    );

    INSERT INTO vec_bolsa_calculo_experiencia.recibo(
        tenant_id, recibo_ref, resultado_ref, intento_nominal_ref,
        generacion_clave_hmac, indice_efecto_hmac_sha256,
        huella_clave_semantica_sha256, huella_intencion_sha256,
        huella_resultado_sha256, tipo_efecto,
        sujeto_ref, sujeto_version, huella_sujeto_sha256,
        convocatoria_ref, convocatoria_version, huella_convocatoria_sha256,
        estado, fase,
        esquema_recibo, recibo_canonico, huella_recibo_sha256, emitido_en
    ) VALUES (
        'diputacion_granada', 'recibo:experiencia:1',
        'resultado:experiencia:1', intento_1, 1, indice, h_clave,
        h_intencion, h_resultado, 'calculo_inicial',
        'sujeto:pseudonimizado:1', 1, repeat('8', 64),
        'convocatoria:1', 1, repeat('5', 64),
        'completado', 'completado',
        'vec.bolsa.calculo-experiencia-oficial.recibo.v1',
        recibo, h_recibo, instante_1
    );

    INSERT INTO vec_bolsa_calculo_experiencia.auditoria(
        tenant_id, auditoria_ref, secuencia, intento_ref, resultado_ref,
        auditoria_anterior_ref, huella_anterior_sha256,
        esquema_auditoria, registro_canonico,
        huella_auditoria_sha256, registrada_en
    ) VALUES (
        'diputacion_granada', 'auditoria:experiencia:1', 1, intento_1,
        'resultado:experiencia:1', NULL, repeat('0', 64),
        'vec.bolsa.calculo-experiencia.auditoria.v1',
        auditoria_1, h_auditoria_1, instante_1
    );
    INSERT INTO vec_bolsa_calculo_experiencia.outbox(
        tenant_id, outbox_ref, resultado_ref, ruta, esquema_evento,
        evento_canonico, huella_evento_sha256, creada_en
    ) VALUES (
        'diputacion_granada', 'outbox:experiencia:1',
        'resultado:experiencia:1',
        'bolsa.calculo_experiencia.resultado_oficial.v1',
        'vec.bolsa.calculo-experiencia.resultado-confirmado.v1',
        evento, encode(sha256(evento), 'hex'), instante_1
    );
    SET CONSTRAINTS ALL IMMEDIATE;
    SET CONSTRAINTS ALL DEFERRED;

    INSERT INTO vec_bolsa_calculo_experiencia.intento(
        tenant_id, intento_ref, resultado_ref, desenlace,
        esquema_intencion, intencion_canonica, huella_intencion_sha256,
        generacion_clave_hmac, indice_efecto_hmac_sha256,
        huella_clave_semantica_sha256, huella_resultado_sha256,
        tipo_efecto, estado, fase, consumo_lectura_ref, consumo_escritura_ref,
        recibo_ref, auditoria_ref, iniciado_en, confirmado_en
    ) VALUES (
        'diputacion_granada', intento_2, 'resultado:experiencia:1',
        'reutilizada',
        'vec.bolsa.calculo-experiencia-oficial.intencion-resultado.v1',
        intencion, h_intencion, 1, indice, h_clave, h_resultado,
        'calculo_inicial',
        'completado', 'completado', 'consumo:lectura:2',
        'consumo:escritura:2', 'recibo:experiencia:1',
        'auditoria:experiencia:2', instante_2, instante_2
    );
    INSERT INTO vec_bolsa_calculo_experiencia.consumo_autorizaciones(
        tenant_id, intento_ref, resultado_ref, perfil_proteccion, tipo_efecto,
        consumo_lectura_ref, consumo_lectura_version,
        huella_consumo_lectura_sha256,
        consumo_prueba_ref, consumo_prueba_version,
        huella_consumo_prueba_sha256,
        decision_lectura_ref, huella_decision_lectura_sha256,
        correlacion_lectura_ref, recurso_lectura_ref,
        consumo_escritura_ref, decision_escritura_ref,
        huella_decision_escritura_sha256,
        correlacion_escritura_ref, recurso_escritura_ref,
        contexto_recurso_lectura_huella_sha256,
        contexto_recurso_escritura_huella_sha256,
        huella_selector_fuente_sha256, huella_intencion_sha256,
        huella_efecto_sha256,
        lectura_consumida_en, escritura_consumida_en
    ) VALUES (
        'diputacion_granada', intento_2, 'resultado:experiencia:1',
        'interno_alto', 'calculo_inicial',
        'consumo:lectura:2', 1, repeat('f', 64),
        'consumo:prueba:2', 1, repeat('1', 64),
        'decision:lectura:2', repeat('2', 64), intento_2,
        'fuente:' || h_selector, 'consumo:escritura:2',
        'decision:escritura:2', repeat('3', 64),
        'correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
        'calculo-oficial:' || h_intencion,
        repeat('4', 64), repeat('5', 64),
        h_selector, h_intencion, h_resultado, instante_2, instante_2
    );
    INSERT INTO vec_bolsa_calculo_experiencia.auditoria(
        tenant_id, auditoria_ref, secuencia, intento_ref, resultado_ref,
        auditoria_anterior_ref, huella_anterior_sha256,
        esquema_auditoria, registro_canonico,
        huella_auditoria_sha256, registrada_en
    ) VALUES (
        'diputacion_granada', 'auditoria:experiencia:2', 2, intento_2,
        'resultado:experiencia:1', 'auditoria:experiencia:1',
        h_auditoria_1, 'vec.bolsa.calculo-experiencia.auditoria.v1',
        auditoria_2, h_auditoria_2, instante_2
    );
    SET CONSTRAINTS ALL IMMEDIATE;

    IF (SELECT count(*) FROM vec_bolsa_calculo_experiencia.resultado_oficial) <> 1
       OR (SELECT count(*) FROM vec_bolsa_calculo_experiencia.recibo) <> 1
       OR (SELECT count(*) FROM vec_bolsa_calculo_experiencia.outbox) <> 1
       OR (SELECT count(*) FROM vec_bolsa_calculo_experiencia.intento) <> 2
       OR (SELECT count(*) FROM vec_bolsa_calculo_experiencia.consumo_autorizaciones) <> 2
       OR (SELECT count(*) FROM vec_bolsa_calculo_experiencia.auditoria) <> 2
       OR NOT EXISTS (
           SELECT 1 FROM vec_bolsa_calculo_experiencia.intento AS i
           JOIN vec_bolsa_calculo_experiencia.recibo AS r
             ON r.tenant_id = i.tenant_id AND r.recibo_ref = i.recibo_ref
          WHERE i.intento_ref = intento_2
            AND r.huella_recibo_sha256 = h_recibo
       ) THEN
        RAISE EXCEPTION 'repeticion o reconciliacion incorrecta';
    END IF;

    BEGIN
        INSERT INTO vec_bolsa_calculo_experiencia.resultado_oficial(
            tenant_id, resultado_ref, esquema_resultado, resultado_canonico,
            huella_resultado_sha256, esquema_clave_semantica,
            clave_semantica_publica, huella_clave_semantica_sha256,
            generacion_clave_hmac, indice_efecto_hmac_sha256,
            esquema_selector_fuente, selector_fuente_canonico,
            huella_selector_fuente_sha256,
            fuente_ref, fuente_version, huella_fuente_sha256,
            reglas_ref, reglas_version, huella_reglas_contenido_sha256,
            reglas_revision, huella_reglas_estado_sha256,
            convocatoria_ref, convocatoria_version, huella_convocatoria_sha256,
            entrada_ref, entrada_version, huella_entrada_sha256,
            huella_contenido_entrada_sha256,
            sujeto_ref, sujeto_version, huella_sujeto_sha256, tipo_efecto,
            estado, fase, intento_nominal_ref, recibo_ref, outbox_ref, creada_en
        ) VALUES (
            'diputacion_granada', 'resultado:colision',
            'vec.bolsa.resultado_experiencia.v1', resultado, h_resultado,
            'vec.bolsa.calculo-experiencia-oficial.clave-efecto.v1',
            convert_to('{"colision":true}', 'UTF8'),
            encode(sha256(convert_to('{"colision":true}', 'UTF8')), 'hex'),
            1, indice,
            'vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1',
            selector, h_selector,
            'evidencia:otra', 1, repeat('2', 64),
            'reglas:otra', 1, repeat('3', 64), 1, repeat('4', 64),
            'convocatoria:otra', 1, repeat('5', 64),
            'entrada:otra', 1, repeat('6', 64), repeat('7', 64),
            'sujeto:otro', 1, repeat('8', 64), 'calculo_inicial',
            'completado', 'completado', 'intento:otro',
            'recibo:otro', 'outbox:otro', instante_2
        );
        RAISE EXCEPTION 'se acepto una colision del indice HMAC';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;

    BEGIN
        UPDATE vec_bolsa_calculo_experiencia.resultado_oficial
           SET fase = 'puntuacion'
         WHERE resultado_ref = 'resultado:experiencia:1';
        RAISE EXCEPTION 'la aplicacion pudo mutar un resultado';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;
END
$historia$;

SET LOCAL vec.tenant_id = 'otro_tenant';
DO $aislamiento_tenant$
BEGIN
    IF (SELECT count(*)
          FROM vec_bolsa_calculo_experiencia.resultado_oficial) <> 1 THEN
        RAISE EXCEPTION 'un GUC controlado altero el tenant fijo';
    END IF;
    BEGIN
        INSERT INTO vec_bolsa_calculo_experiencia.outbox(
            tenant_id, outbox_ref, resultado_ref, ruta, esquema_evento,
            evento_canonico, huella_evento_sha256, creada_en
        ) VALUES (
            'otro_tenant', 'outbox:cruzado', 'resultado:experiencia:1',
            'bolsa.calculo_experiencia.resultado_oficial.v1',
            'vec.bolsa.calculo-experiencia.resultado-confirmado.v1',
            convert_to('xx', 'UTF8'),
            encode(sha256(convert_to('xx', 'UTF8')), 'hex'),
            '2026-07-17T10:00:02.000000Z'
        );
        RAISE EXCEPTION 'RLS acepto otro tenant';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;
END
$aislamiento_tenant$;

RESET ROLE;
SET LOCAL ROLE vec_bolsa_calculo_experiencia_propietario;
-- Fuerza una comprobacion del disparador aun frente al propietario: se retira
-- RLS solo dentro de esta transaccion, que se revierte al terminar.
ALTER TABLE vec_bolsa_calculo_experiencia.resultado_oficial
    NO FORCE ROW LEVEL SECURITY;
DO $inmutabilidad_propietario$
BEGIN
    BEGIN
        UPDATE vec_bolsa_calculo_experiencia.resultado_oficial
           SET fase = 'puntuacion'
         WHERE resultado_ref = 'resultado:experiencia:1';
        RAISE EXCEPTION 'el disparador permitio mutar historia';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN
        NULL;
    END;
    BEGIN
        TRUNCATE TABLE
            vec_bolsa_calculo_experiencia.resultado_oficial,
            vec_bolsa_calculo_experiencia.intento,
            vec_bolsa_calculo_experiencia.consumo_autorizaciones,
            vec_bolsa_calculo_experiencia.recibo,
            vec_bolsa_calculo_experiencia.auditoria,
            vec_bolsa_calculo_experiencia.outbox;
        RAISE EXCEPTION 'el disparador permitio truncar historia';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN
        NULL;
    END;
    BEGIN
        TRUNCATE TABLE vec_bolsa_calculo_experiencia.configuracion_tenant;
        RAISE EXCEPTION 'el disparador permitio truncar el tenant';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN
        NULL;
    END;
END
$inmutabilidad_propietario$;
RESET ROLE;

ROLLBACK;
