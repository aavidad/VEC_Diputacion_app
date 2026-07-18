-- Se ejecuta inmediatamente despues de 000003 down y antes de 000002 down.
DO $down_limpio$
DECLARE
    nombre text;
BEGIN
    IF to_regclass(
           'vec_bolsa_convocatorias.atestacion_autorizacion_version'
       ) IS NULL THEN
        RAISE EXCEPTION '000003 down elimino una tabla de 000001';
    END IF;
    FOREACH nombre IN ARRAY ARRAY[
        'atestacion_pdp_borrador','material_borrador',
        'diario_borrador_version','diario_borrador_actual',
        'identidad_alias_borrador',
        'prueba_desenlace_borrador','uso_decision_borrador',
        'sellado_motivo_borrador',
        'borrador_convocatoria_version','borrador_convocatoria_actual',
        'auditoria_borrador','auditoria_borrador_actual','outbox_borrador',
        'uso_decision_lectura_borrador','auditoria_lectura_borrador',
        'cursor_listado_borrador'
    ] LOOP
        IF to_regclass('vec_bolsa_convocatorias.' || nombre) IS NOT NULL THEN
            RAISE EXCEPTION '000003 down dejo la tabla %', nombre;
        END IF;
    END LOOP;
    FOREACH nombre IN ARRAY ARRAY[
        'vec_bolsa_convocatorias.reservar_decision_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.confirmar_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.confirmar_borrador_interna_v1(jsonb,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(bigint,bigint,jsonb,jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(bigint,bigint,jsonb,bytea,bytea,bytea,bytea)',
        'vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(jsonb,text,bigint,bigint,timestamp with time zone)',
        'vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(jsonb,text,bigint,bigint,timestamp with time zone)',
        'vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(jsonb)',
        'vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb)',
        'vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(jsonb)',
        'vec_bolsa_convocatorias.identidades_operacion_borrador_validas(jsonb)',
        'vec_bolsa_convocatorias.listar_borradores_v1(jsonb,jsonb,jsonb,bytea,bytea)',
        'vec_bolsa_convocatorias.obtener_borrador_v1(text,jsonb,jsonb,bytea,bytea)'
    ] LOOP
        IF to_regprocedure(nombre) IS NOT NULL THEN
            RAISE EXCEPTION '000003 down dejo la funcion %', nombre;
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
        IF EXISTS (
            SELECT 1
              FROM pg_catalog.pg_proc AS p
              JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
               AND p.proname = nombre
        ) THEN
            RAISE EXCEPTION '000003 down dejo objetos con nombre %', nombre;
        END IF;
    END LOOP;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_trigger AS g
          JOIN pg_catalog.pg_class AS t ON t.oid = g.tgrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND g.tgname IN (
             'borrador_actual_avance','diario_borrador_actual_avance',
             'auditoria_borrador_actual_avance'
         ) AND NOT g.tgisinternal
    ) THEN
        RAISE EXCEPTION '000003 down dejo disparadores';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS c
          JOIN pg_catalog.pg_class AS t ON t.oid = c.conrelid
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND t.relname = 'atestacion_autorizacion_version'
           AND c.conname = 'atestacion_borrador_vinculo_exacto'
    ) THEN
        RAISE EXCEPTION '000003 down dejo la UNIQUE de atestacion';
    END IF;
END
$down_limpio$;
