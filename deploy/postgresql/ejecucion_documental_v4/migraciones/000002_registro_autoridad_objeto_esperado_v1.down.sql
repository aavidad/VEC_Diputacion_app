BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_ejecucion_documental_v4:registro_autoridad_objeto_esperado_v1:down',
    0
));
LOCK TABLE
    vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1,
    vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1,
    vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1,
    vec_ejecucion_documental_v4.control_cadena_auditoria
    IN ACCESS EXCLUSIVE MODE;

DO $proteger_historia$
DECLARE
    cantidad numeric;
    primera numeric;
    ultima numeric;
    huella_previa text;
    control record;
BEGIN
    SELECT count(*), min(secuencia), max(secuencia)
      INTO cantidad, primera, ultima
      FROM vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1;
    IF cantidad = 0 THEN
        RETURN;
    END IF;
    IF current_setting(
           'vec.limpiar_registro_autoridad_objeto_esperado_v1_prueba', true
       ) IS DISTINCT FROM
           'LIMPIAR_REGISTRO_AUTORIDAD_OBJETO_ESPERADO_V1_PRUEBA' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe historia de autoridad de objeto',
            HINT = 'solo el runner desechable puede activar su limpieza explicita';
    END IF;
    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT control
      FROM vec_ejecucion_documental_v4.control_cadena_auditoria
     WHERE control_id = true;
    IF cantidad <> ultima - primera + 1
       OR control.ultima_secuencia <> ultima
       OR control.ultima_huella_sha256 IS DISTINCT FROM (
           SELECT huella_registro_sha256
             FROM vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1
            WHERE secuencia = ultima
       ) OR EXISTS (
           SELECT 1 FROM vec_ejecucion_documental_v4.auditoria
            WHERE secuencia >= primera
       ) OR cantidad <> (
           SELECT count(*)
             FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1
       ) OR cantidad <> (
           SELECT count(*)
             FROM vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1
       ) OR EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1 AS actual
             LEFT JOIN vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1 AS previa
               ON previa.secuencia = actual.secuencia - 1
            WHERE actual.secuencia > primera
              AND actual.huella_anterior_sha256 IS DISTINCT FROM
                  previa.huella_registro_sha256
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la historia no es una cola propia limpiable';
    END IF;
    SELECT huella_anterior_sha256 INTO STRICT huella_previa
      FROM vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1
     WHERE secuencia = primera;
    IF (primera = 1 AND huella_previa <> repeat('0', 64))
       OR (primera > 1 AND NOT EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.auditoria AS anterior
            WHERE anterior.secuencia = primera - 1
              AND anterior.huella_registro_sha256 = huella_previa
       )) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el origen de la cola no es verificable';
    END IF;
    UPDATE vec_ejecucion_documental_v4.control_cadena_auditoria
       SET ultima_secuencia = primera - 1,
           ultima_huella_sha256 = huella_previa
     WHERE control_id = true
       AND ultima_secuencia = ultima
       AND ultima_huella_sha256 = control.ultima_huella_sha256;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'la cola de auditoria cambio durante la limpieza';
    END IF;
END
$proteger_historia$;

DROP FUNCTION
    vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
        numeric, bytea
    ) RESTRICT;
DROP TABLE
    vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1,
    vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1,
    vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1
    RESTRICT;
DROP FUNCTION
    vec_ejecucion_documental_v4.recibo_material_v2_coteja_autoridad_objeto_v1(
        bytea, text, text, text, text, text
    ) RESTRICT;
DROP FUNCTION
    vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(bytea)
    RESTRICT;
DROP FUNCTION
    vec_ejecucion_documental_v4.referencia_opaca_autoridad_objeto_v1(
        text, integer
    ) RESTRICT;
COMMIT;
