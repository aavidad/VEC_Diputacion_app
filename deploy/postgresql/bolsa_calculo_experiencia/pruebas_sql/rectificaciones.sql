-- Prueba una rectificacion completa y los rechazos de rama, cruce y retroceso
-- temporal. La autoridad se sustituye solo dentro de esta transaccion.
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

DO $rectificaciones$
DECLARE
    t_0 constant text := '2026-07-17T11:00:00.000000Z';
    t_1 constant text := '2026-07-17T11:00:01.000000Z';
    resultado_0 bytea := convert_to('resultado-inicial', 'UTF8');
    clave_0 bytea := convert_to('clave-inicial', 'UTF8');
    intencion_0 bytea := convert_to('intencion-inicial', 'UTF8');
    recibo_0 bytea := convert_to('recibo-inicial', 'UTF8');
    selector bytea := convert_to(
        '{"esquema":"vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1"}',
        'UTF8'
    );
    auditoria_0 bytea := convert_to('auditoria-inicial', 'UTF8');
    evento_0 bytea := convert_to('evento-inicial', 'UTF8');
    h_resultado_0 text := encode(sha256(resultado_0), 'hex');
    h_clave_0 text := encode(sha256(clave_0), 'hex');
    h_intencion_0 text := encode(sha256(intencion_0), 'hex');
    h_recibo_0 text := encode(sha256(recibo_0), 'hex');
    h_selector text := encode(sha256(selector), 'hex');
    h_auditoria_0 text := encode(sha256(auditoria_0), 'hex');
    intento_0 text := 'correlacion_10101010101010101010101010101010';

    resultado_1 bytea := convert_to('resultado-rectificado', 'UTF8');
    clave_1 bytea := convert_to('clave-rectificacion', 'UTF8');
    intencion_1 bytea := convert_to('intencion-rectificacion', 'UTF8');
    recibo_1 bytea := convert_to('recibo-rectificacion', 'UTF8');
    auditoria_1 bytea := convert_to('auditoria-rectificacion', 'UTF8');
    evento_1 bytea := convert_to('evento-rectificacion', 'UTF8');
    h_resultado_1 text := encode(sha256(resultado_1), 'hex');
    h_clave_1 text := encode(sha256(clave_1), 'hex');
    h_intencion_1 text := encode(sha256(intencion_1), 'hex');
    h_recibo_1 text := encode(sha256(recibo_1), 'hex');
    h_auditoria_1 text := encode(sha256(auditoria_1), 'hex');
    intento_1 text := 'correlacion_20202020202020202020202020202020';

    fila_resultado vec_bolsa_calculo_experiencia.resultado_oficial%ROWTYPE;
    fila_intento vec_bolsa_calculo_experiencia.intento%ROWTYPE;
    fila_consumo vec_bolsa_calculo_experiencia.consumo_autorizaciones%ROWTYPE;
    fila_recibo vec_bolsa_calculo_experiencia.recibo%ROWTYPE;
    fila_auditoria vec_bolsa_calculo_experiencia.auditoria%ROWTYPE;
    fila_outbox vec_bolsa_calculo_experiencia.outbox%ROWTYPE;
    candidato vec_bolsa_calculo_experiencia.resultado_oficial%ROWTYPE;
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
        sujeto_ref, sujeto_version, huella_sujeto_sha256,
        tipo_efecto, estado, fase, intento_nominal_ref,
        recibo_ref, outbox_ref, creada_en
    ) VALUES (
        'diputacion_granada', 'resultado:inicial',
        'vec.bolsa.resultado_experiencia.v1', resultado_0, h_resultado_0,
        'vec.bolsa.calculo-experiencia-oficial.clave-efecto.v1',
        clave_0, h_clave_0, 1, repeat('1', 64),
        'vec.bolsa.calculo-experiencia.selector-fuente-exacta.v1',
        selector, h_selector,
        'fuente:exacta:inicial', 1, repeat('2', 64),
        'reglas:exactas', 1, repeat('3', 64), 1, repeat('4', 64),
        'convocatoria:misma', 1, repeat('5', 64),
        'entrada:inicial', 1, repeat('6', 64), repeat('7', 64),
        'hmac-sha256:personas:' || repeat('8', 64),
        1, repeat('8', 64),
        'calculo_inicial', 'completado', 'completado', intento_0,
        'recibo:inicial', 'outbox:inicial', t_0
    );
    INSERT INTO vec_bolsa_calculo_experiencia.intento VALUES (
        'diputacion_granada', intento_0, 'resultado:inicial', 'creada',
        'vec.bolsa.calculo-experiencia-oficial.intencion-resultado.v1',
        intencion_0, h_intencion_0, 1, repeat('1', 64), h_clave_0,
        h_resultado_0, 'calculo_inicial', 'completado', 'completado',
        'consumo:lectura:inicial', 'consumo:escritura:inicial',
        'recibo:inicial', 'auditoria:inicial', t_0, t_0
    );
    INSERT INTO vec_bolsa_calculo_experiencia.consumo_autorizaciones VALUES (
        'diputacion_granada', intento_0, 'resultado:inicial',
        'interno_alto', 'calculo_inicial',
        'consumo:lectura:inicial', 1, repeat('9', 64),
        'consumo:prueba:inicial', 1, repeat('a', 64),
        'decision:lectura:inicial', repeat('b', 64), intento_0,
        'fuente:' || h_selector,
        'consumo:escritura:inicial', 'decision:escritura:inicial',
        repeat('c', 64), 'correlacion_30303030303030303030303030303030',
        'calculo-oficial:' || h_intencion_0,
        repeat('d', 64), repeat('e', 64), h_selector, h_intencion_0,
        h_resultado_0, t_0, t_0
    );
    INSERT INTO vec_bolsa_calculo_experiencia.recibo VALUES (
        'diputacion_granada', 'recibo:inicial', 'resultado:inicial',
        intento_0, 'creada', 1, repeat('1', 64), h_clave_0,
        h_intencion_0, h_resultado_0, 'calculo_inicial',
        'hmac-sha256:personas:' || repeat('8', 64),
        1, repeat('8', 64),
        'convocatoria:misma', 1, repeat('5', 64),
        'completado', 'completado',
        'vec.bolsa.calculo-experiencia-oficial.recibo.v1',
        recibo_0, h_recibo_0, t_0
    );
    INSERT INTO vec_bolsa_calculo_experiencia.auditoria VALUES (
        'diputacion_granada', 'auditoria:inicial', 1, intento_0,
        'resultado:inicial', NULL, repeat('0', 64),
        'vec.bolsa.calculo-experiencia.auditoria.v1',
        auditoria_0, h_auditoria_0, t_0
    );
    INSERT INTO vec_bolsa_calculo_experiencia.outbox VALUES (
        'diputacion_granada', 'outbox:inicial', 'resultado:inicial',
        'bolsa.calculo_experiencia.resultado_oficial.v1',
        'vec.bolsa.calculo-experiencia.resultado-confirmado.v1',
        evento_0, encode(sha256(evento_0), 'hex'), t_0
    );
    SET CONSTRAINTS ALL IMMEDIATE;
    SET CONSTRAINTS ALL DEFERRED;

    SELECT * INTO fila_resultado
      FROM vec_bolsa_calculo_experiencia.resultado_oficial
     WHERE resultado_ref = 'resultado:inicial';
    fila_resultado.resultado_ref := 'resultado:rectificacion';
    fila_resultado.resultado_canonico := resultado_1;
    fila_resultado.huella_resultado_sha256 := h_resultado_1;
    fila_resultado.clave_semantica_publica := clave_1;
    fila_resultado.huella_clave_semantica_sha256 := h_clave_1;
    fila_resultado.generacion_clave_hmac := 2;
    fila_resultado.indice_efecto_hmac_sha256 := repeat('2', 64);
    fila_resultado.tipo_efecto := 'rectificacion';
    fila_resultado.predecesor_recibo_ref := 'recibo:inicial';
    fila_resultado.huella_predecesor_recibo_sha256 := h_recibo_0;
    fila_resultado.intento_nominal_ref := intento_1;
    fila_resultado.recibo_ref := 'recibo:rectificacion';
    fila_resultado.outbox_ref := 'outbox:rectificacion';
    fila_resultado.creada_en := t_1;
    INSERT INTO vec_bolsa_calculo_experiencia.resultado_oficial
        SELECT fila_resultado.*;

    SELECT * INTO fila_intento
      FROM vec_bolsa_calculo_experiencia.intento
     WHERE intento_ref = intento_0;
    fila_intento.intento_ref := intento_1;
    fila_intento.resultado_ref := 'resultado:rectificacion';
    fila_intento.intencion_canonica := intencion_1;
    fila_intento.huella_intencion_sha256 := h_intencion_1;
    fila_intento.generacion_clave_hmac := 2;
    fila_intento.indice_efecto_hmac_sha256 := repeat('2', 64);
    fila_intento.huella_clave_semantica_sha256 := h_clave_1;
    fila_intento.huella_resultado_sha256 := h_resultado_1;
    fila_intento.tipo_efecto := 'rectificacion';
    fila_intento.consumo_lectura_ref := 'consumo:lectura:rectificacion';
    fila_intento.consumo_escritura_ref := 'consumo:escritura:rectificacion';
    fila_intento.recibo_ref := 'recibo:rectificacion';
    fila_intento.auditoria_ref := 'auditoria:rectificacion';
    fila_intento.iniciado_en := t_1;
    fila_intento.confirmado_en := t_1;
    INSERT INTO vec_bolsa_calculo_experiencia.intento SELECT fila_intento.*;

    SELECT * INTO fila_consumo
      FROM vec_bolsa_calculo_experiencia.consumo_autorizaciones
     WHERE intento_ref = intento_0;
    fila_consumo.intento_ref := intento_1;
    fila_consumo.resultado_ref := 'resultado:rectificacion';
    fila_consumo.tipo_efecto := 'rectificacion';
    fila_consumo.consumo_lectura_ref := 'consumo:lectura:rectificacion';
    fila_consumo.consumo_prueba_ref := 'consumo:prueba:rectificacion';
    fila_consumo.decision_lectura_ref := 'decision:lectura:rectificacion';
    fila_consumo.correlacion_lectura_ref := intento_1;
    fila_consumo.consumo_escritura_ref := 'consumo:escritura:rectificacion';
    fila_consumo.decision_escritura_ref := 'decision:escritura:rectificacion';
    fila_consumo.correlacion_escritura_ref :=
        'correlacion_40404040404040404040404040404040';
    fila_consumo.recurso_escritura_ref :=
        'rectificacion-calculo-oficial:' || h_intencion_1;
    fila_consumo.huella_intencion_sha256 := h_intencion_1;
    fila_consumo.huella_efecto_sha256 := h_resultado_1;
    fila_consumo.lectura_consumida_en := t_1;
    fila_consumo.escritura_consumida_en := t_1;
    INSERT INTO vec_bolsa_calculo_experiencia.consumo_autorizaciones
        SELECT fila_consumo.*;

    SELECT * INTO fila_recibo
      FROM vec_bolsa_calculo_experiencia.recibo
     WHERE recibo_ref = 'recibo:inicial';
    fila_recibo.recibo_ref := 'recibo:rectificacion';
    fila_recibo.resultado_ref := 'resultado:rectificacion';
    fila_recibo.intento_nominal_ref := intento_1;
    fila_recibo.generacion_clave_hmac := 2;
    fila_recibo.indice_efecto_hmac_sha256 := repeat('2', 64);
    fila_recibo.huella_clave_semantica_sha256 := h_clave_1;
    fila_recibo.huella_intencion_sha256 := h_intencion_1;
    fila_recibo.huella_resultado_sha256 := h_resultado_1;
    fila_recibo.tipo_efecto := 'rectificacion';
    fila_recibo.recibo_canonico := recibo_1;
    fila_recibo.huella_recibo_sha256 := h_recibo_1;
    fila_recibo.emitido_en := t_1;
    INSERT INTO vec_bolsa_calculo_experiencia.recibo SELECT fila_recibo.*;

    SELECT * INTO fila_auditoria
      FROM vec_bolsa_calculo_experiencia.auditoria
     WHERE auditoria_ref = 'auditoria:inicial';
    fila_auditoria.auditoria_ref := 'auditoria:rectificacion';
    fila_auditoria.secuencia := 2;
    fila_auditoria.intento_ref := intento_1;
    fila_auditoria.resultado_ref := 'resultado:rectificacion';
    fila_auditoria.auditoria_anterior_ref := 'auditoria:inicial';
    fila_auditoria.huella_anterior_sha256 := h_auditoria_0;
    fila_auditoria.registro_canonico := auditoria_1;
    fila_auditoria.huella_auditoria_sha256 := h_auditoria_1;
    fila_auditoria.registrada_en := t_1;
    INSERT INTO vec_bolsa_calculo_experiencia.auditoria
        SELECT fila_auditoria.*;

    SELECT * INTO fila_outbox
      FROM vec_bolsa_calculo_experiencia.outbox
     WHERE outbox_ref = 'outbox:inicial';
    fila_outbox.outbox_ref := 'outbox:rectificacion';
    fila_outbox.resultado_ref := 'resultado:rectificacion';
    fila_outbox.evento_canonico := evento_1;
    fila_outbox.huella_evento_sha256 := encode(sha256(evento_1), 'hex');
    fila_outbox.creada_en := t_1;
    INSERT INTO vec_bolsa_calculo_experiencia.outbox SELECT fila_outbox.*;
    SET CONSTRAINTS ALL IMMEDIATE;
    SET CONSTRAINTS ALL DEFERRED;

    IF (SELECT count(*)
          FROM vec_bolsa_calculo_experiencia.resultado_oficial) <> 2
       OR (SELECT count(*) FROM vec_bolsa_calculo_experiencia.recibo) <> 2
       OR NOT EXISTS (
           SELECT 1
             FROM vec_bolsa_calculo_experiencia.resultado_oficial
            WHERE resultado_ref = 'resultado:rectificacion'
              AND predecesor_recibo_ref = 'recibo:inicial'
              AND huella_predecesor_recibo_sha256 = h_recibo_0
       ) THEN
        RAISE EXCEPTION 'la rectificacion completa no quedo ligada';
    END IF;

    SELECT * INTO candidato
      FROM vec_bolsa_calculo_experiencia.resultado_oficial
     WHERE resultado_ref = 'resultado:rectificacion';
    candidato.resultado_ref := 'resultado:rama';
    candidato.clave_semantica_publica := convert_to('clave-rama', 'UTF8');
    candidato.huella_clave_semantica_sha256 :=
        encode(sha256(candidato.clave_semantica_publica), 'hex');
    candidato.generacion_clave_hmac := 3;
    candidato.indice_efecto_hmac_sha256 := repeat('3', 64);
    candidato.intento_nominal_ref := 'intento:rama';
    candidato.recibo_ref := 'recibo:rama';
    candidato.outbox_ref := 'outbox:rama';
    candidato.creada_en := '2026-07-17T11:00:02.000000Z';
    BEGIN
        INSERT INTO vec_bolsa_calculo_experiencia.resultado_oficial
            SELECT candidato.*;
        RAISE EXCEPTION 'se acepto una segunda rama';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;

    candidato.resultado_ref := 'resultado:cruzado';
    candidato.clave_semantica_publica := convert_to('clave-cruzada', 'UTF8');
    candidato.huella_clave_semantica_sha256 :=
        encode(sha256(candidato.clave_semantica_publica), 'hex');
    candidato.generacion_clave_hmac := 4;
    candidato.indice_efecto_hmac_sha256 := repeat('4', 64);
    candidato.sujeto_ref :=
        'hmac-sha256:personas:' || repeat('9', 64);
    candidato.intento_nominal_ref := 'intento:cruzado';
    candidato.recibo_ref := 'recibo:cruzado';
    candidato.outbox_ref := 'outbox:cruzado';
    BEGIN
        INSERT INTO vec_bolsa_calculo_experiencia.resultado_oficial
            SELECT candidato.*;
        RAISE EXCEPTION 'se acepto un predecesor de otro sujeto';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;

    SELECT * INTO candidato
      FROM vec_bolsa_calculo_experiencia.resultado_oficial
     WHERE resultado_ref = 'resultado:rectificacion';
    candidato.resultado_ref := 'resultado:retroceso';
    candidato.clave_semantica_publica := convert_to('clave-retroceso', 'UTF8');
    candidato.huella_clave_semantica_sha256 :=
        encode(sha256(candidato.clave_semantica_publica), 'hex');
    candidato.generacion_clave_hmac := 5;
    candidato.indice_efecto_hmac_sha256 := repeat('5', 64);
    candidato.predecesor_recibo_ref := 'recibo:rectificacion';
    candidato.huella_predecesor_recibo_sha256 := h_recibo_1;
    candidato.intento_nominal_ref := 'intento:retroceso';
    candidato.recibo_ref := 'recibo:retroceso';
    candidato.outbox_ref := 'outbox:retroceso';
    candidato.creada_en := t_0;
    BEGIN
        INSERT INTO vec_bolsa_calculo_experiencia.resultado_oficial
            SELECT candidato.*;
        RAISE EXCEPTION 'se acepto un enlace temporal hacia atras';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;

    SELECT * INTO candidato
      FROM vec_bolsa_calculo_experiencia.resultado_oficial
     WHERE resultado_ref = 'resultado:inicial';
    candidato.resultado_ref := 'resultado:selector-alterado';
    candidato.clave_semantica_publica := convert_to('clave-selector', 'UTF8');
    candidato.huella_clave_semantica_sha256 :=
        encode(sha256(candidato.clave_semantica_publica), 'hex');
    candidato.generacion_clave_hmac := 6;
    candidato.indice_efecto_hmac_sha256 := repeat('6', 64);
    candidato.selector_fuente_canonico :=
        convert_to('selector-distinto', 'UTF8');
    candidato.intento_nominal_ref := 'intento:selector-alterado';
    candidato.recibo_ref := 'recibo:selector-alterado';
    candidato.outbox_ref := 'outbox:selector-alterado';
    BEGIN
        INSERT INTO vec_bolsa_calculo_experiencia.resultado_oficial
            SELECT candidato.*;
        RAISE EXCEPTION 'se acepto un selector con huella ajena';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END
$rectificaciones$;

ROLLBACK;
