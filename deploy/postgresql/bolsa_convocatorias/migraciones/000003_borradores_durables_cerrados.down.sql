-- Reversion conservadora. La historia durable exige confirmacion explicita.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
DECLARE
    nombre text;
BEGIN
    FOREACH nombre IN ARRAY ARRAY[
        'atestacion_pdp_borrador','material_borrador',
        'diario_borrador_version','diario_borrador_actual',
        'identidad_alias_borrador','prueba_desenlace_borrador',
        'uso_decision_borrador','sellado_motivo_borrador',
        'borrador_convocatoria_version','borrador_convocatoria_actual',
        'auditoria_borrador','auditoria_borrador_actual','outbox_borrador',
        'uso_decision_lectura_borrador','auditoria_lectura_borrador',
        'cursor_listado_borrador'
    ] LOOP
        IF to_regclass('vec_bolsa_convocatorias.' || nombre) IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: inventario de tablas 000003 incompleto',
                DETAIL = nombre;
        END IF;
    END LOOP;
    FOREACH nombre IN ARRAY ARRAY[
        'vec_bolsa_convocatorias.reservar_decision_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.confirmar_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.confirmar_borrador_interna_v1(jsonb,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(bigint,bigint,jsonb,jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(bigint,bigint,jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(jsonb,text,bigint,bigint,timestamp with time zone)',
        'vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(jsonb,text,bigint,bigint,timestamp with time zone)',
        'vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb)',
        'vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(jsonb)',
        'vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(jsonb)',
        'vec_bolsa_convocatorias.listar_borradores_v1(jsonb,jsonb,jsonb,bytea,bytea)',
        'vec_bolsa_convocatorias.listar_borradores_interna_v1(jsonb,jsonb)',
        'vec_bolsa_convocatorias.obtener_borrador_v1(text,jsonb,jsonb,bytea,bytea)',
        'vec_bolsa_convocatorias.obtener_borrador_interna_v1(text,jsonb)',
        'vec_bolsa_convocatorias.registrar_lectura_borrador_interna_v1(jsonb,bytea)',
        'vec_bolsa_convocatorias.validar_reserva_borrador_interna_v1(jsonb,bytea,bytea,bytea,bytea,timestamp with time zone)',
        'vec_bolsa_convocatorias.identidades_operacion_borrador_validas(jsonb)',
        'vec_bolsa_convocatorias.identidad_operacion_borrador_valida(jsonb)',
        'vec_bolsa_convocatorias.identidad_runtime_borrador_valida(text,boolean)',
        'vec_bolsa_convocatorias.contexto_lectura_borrador_valido(bytea,jsonb,text,boolean)'
    ] LOOP
        IF to_regprocedure(nombre) IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: inventario de funciones 000003 incompleto',
                DETAIL = nombre;
        END IF;
    END LOOP;
    FOREACH nombre IN ARRAY ARRAY[
        'referencia_clave_hmac_valida','hmac_sha256_valido',
        'objeto_json_exacto','lista_texto_canonica',
        'referencia_estado_valida','atestacion_pdp_borrador_vigente',
        'validar_avance_puntero_borrador',
        'validar_avance_diario_actual',
        'validar_avance_auditoria_borrador',
        'identidad_runtime_borrador_valida',
        'contexto_lectura_borrador_valido',
        'registrar_lectura_borrador_interna_v1',
        'listar_borradores_interna_v1','listar_borradores_v1',
        'obtener_borrador_interna_v1','obtener_borrador_v1',
        'reclamar_reserva_borrador_interna_v1',
        'reclamar_reserva_borrador_v1',
        'reconciliar_operacion_borrador_interna_v1',
        'reconciliar_operacion_borrador_v1',
        'confirmar_borrador_interna_v1','confirmar_borrador_v1',
        'identidad_operacion_borrador_valida',
        'identidades_operacion_borrador_validas',
        'validar_reserva_borrador_interna_v1',
        'consultar_diario_borrador_interna_v1',
        'consultar_identidades_borrador_interna_v1',
        'consultar_identidades_borrador_v1',
        'reservar_decision_borrador_interna_v1',
        'reservar_decision_borrador_v1'
    ] LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_proc AS p
              JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
               AND p.proname = nombre
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: inventario nominal 000003 incompleto',
                DETAIL = nombre;
        END IF;
    END LOOP;
    IF (SELECT count(*)
          FROM pg_catalog.pg_trigger AS g
          JOIN pg_catalog.pg_class AS t ON t.oid = g.tgrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND g.tgname IN (
             'borrador_actual_avance','diario_borrador_actual_avance',
             'auditoria_borrador_actual_avance'
         ) AND NOT g.tgisinternal) <> 3 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: inventario de disparadores 000003 incompleto';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS c
          JOIN pg_catalog.pg_class AS t ON t.oid = c.conrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND t.relname = 'atestacion_autorizacion_version'
           AND c.conname = 'atestacion_borrador_vinculo_exacto'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta vinculo de atestacion 000003';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS c
          JOIN pg_catalog.pg_class AS t ON t.oid = c.conrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND t.relname = 'identidad_alias_borrador'
           AND c.conname = 'identidad_alias_generacion_primaria_unica'
           AND c.contype = 'u'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta unicidad generacional 000003';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_bolsa_convocatorias.cursor_listado_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.auditoria_lectura_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.uso_decision_lectura_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.outbox_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.auditoria_borrador_actual
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.auditoria_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.sellado_motivo_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.uso_decision_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.diario_borrador_actual
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.identidad_alias_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.prueba_desenlace_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.diario_borrador_version
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.material_borrador
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.borrador_convocatoria_actual
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.borrador_convocatoria_version
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.atestacion_pdp_borrador
    IN ACCESS EXCLUSIVE MODE;

DO $confirmar_historia$
BEGIN
    IF (EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.atestacion_pdp_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.material_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.diario_borrador_version
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.identidad_alias_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.prueba_desenlace_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.borrador_convocatoria_version
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.uso_decision_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.sellado_motivo_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.auditoria_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.outbox_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.uso_decision_lectura_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.auditoria_lectura_borrador
        ) OR EXISTS (
            SELECT 1 FROM vec_bolsa_convocatorias.cursor_listado_borrador
        ))
       AND current_setting(
           'vec.confirmar_destruccion_borradores_convocatorias', true
       ) IS DISTINCT FROM
           'DESTRUIR_HISTORIA_BORRADORES_CONVOCATORIAS_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe historia durable de borradores',
            HINT = 'requiere procedimiento formal y confirmacion explicita';
    END IF;
END
$confirmar_historia$;

DROP FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text, jsonb, jsonb, bytea, bytea
);
DROP FUNCTION vec_bolsa_convocatorias.obtener_borrador_interna_v1(
    text, jsonb
);
DROP FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    jsonb, jsonb, jsonb, bytea, bytea
);
DROP FUNCTION vec_bolsa_convocatorias.listar_borradores_interna_v1(
    jsonb, jsonb
);
DROP FUNCTION
    vec_bolsa_convocatorias.registrar_lectura_borrador_interna_v1(
        jsonb, bytea
    );
DROP FUNCTION vec_bolsa_convocatorias.confirmar_borrador_v1(
    jsonb, jsonb, bytea, bytea, bytea, bytea, bytea
);
DROP FUNCTION vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
    jsonb, bytea, bytea, bytea
);
DROP FUNCTION
    vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(
        jsonb, text, bigint, bigint, timestamptz
    );
DROP FUNCTION vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(
    bigint, bigint, jsonb, jsonb, bytea, bytea, bytea, bytea
);
DROP FUNCTION
    vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(
        bigint, bigint, jsonb, bytea, bytea, bytea, bytea
    );
DROP FUNCTION
    vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
        jsonb, text, bigint, bigint, timestamptz
    );
DROP FUNCTION vec_bolsa_convocatorias.reservar_decision_borrador_v1(
    jsonb, jsonb, bytea, bytea, bytea, bytea
);
DROP FUNCTION
    vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
        jsonb, bytea, bytea, bytea, bytea
    );
DROP FUNCTION
    vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.validar_reserva_borrador_interna_v1(
        jsonb, bytea, bytea, bytea, bytea, timestamptz
    );
DROP FUNCTION
    vec_bolsa_convocatorias.identidades_operacion_borrador_validas(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.identidad_operacion_borrador_valida(jsonb);
DROP FUNCTION
    vec_bolsa_convocatorias.atestacion_pdp_borrador_vigente(
        text, text, bigint, text, text, text, text, timestamptz,
        timestamptz
    );
DROP FUNCTION
    vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
        bytea, jsonb, text, boolean
    );
DROP FUNCTION
    vec_bolsa_convocatorias.identidad_runtime_borrador_valida(text, boolean);

DROP TABLE vec_bolsa_convocatorias.cursor_listado_borrador;
DROP TABLE vec_bolsa_convocatorias.auditoria_lectura_borrador;
DROP TABLE vec_bolsa_convocatorias.uso_decision_lectura_borrador;
DROP TABLE vec_bolsa_convocatorias.outbox_borrador;
DROP TABLE vec_bolsa_convocatorias.auditoria_borrador_actual;
DROP TABLE vec_bolsa_convocatorias.auditoria_borrador;
DROP TABLE vec_bolsa_convocatorias.sellado_motivo_borrador;
DROP TABLE vec_bolsa_convocatorias.uso_decision_borrador;
DROP TABLE vec_bolsa_convocatorias.identidad_alias_borrador;
DROP TABLE vec_bolsa_convocatorias.diario_borrador_actual;
ALTER TABLE vec_bolsa_convocatorias.diario_borrador_version
    DROP CONSTRAINT diario_borrador_prueba_desenlace_fk;
DROP TABLE vec_bolsa_convocatorias.prueba_desenlace_borrador;
DROP TABLE vec_bolsa_convocatorias.diario_borrador_version;
DROP TABLE vec_bolsa_convocatorias.material_borrador;
DROP TABLE vec_bolsa_convocatorias.borrador_convocatoria_actual;
DROP TABLE vec_bolsa_convocatorias.borrador_convocatoria_version;
DROP TABLE vec_bolsa_convocatorias.atestacion_pdp_borrador;

ALTER TABLE vec_bolsa_convocatorias.atestacion_autorizacion_version
    DROP CONSTRAINT atestacion_borrador_vinculo_exacto;

DROP FUNCTION
    vec_bolsa_convocatorias.validar_avance_auditoria_borrador();
DROP FUNCTION vec_bolsa_convocatorias.validar_avance_diario_actual();
DROP FUNCTION vec_bolsa_convocatorias.validar_avance_puntero_borrador();
DROP FUNCTION vec_bolsa_convocatorias.referencia_estado_valida(jsonb);
DROP FUNCTION vec_bolsa_convocatorias.lista_texto_canonica(text[]);
DROP FUNCTION vec_bolsa_convocatorias.objeto_json_exacto(jsonb, text[]);
DROP FUNCTION vec_bolsa_convocatorias.hmac_sha256_valido(bytea);
DROP FUNCTION vec_bolsa_convocatorias.referencia_clave_hmac_valida(
    text, text
);
COMMIT;
