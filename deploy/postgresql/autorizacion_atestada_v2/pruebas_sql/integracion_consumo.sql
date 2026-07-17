-- Prueba mecanica de la puerta. La autoridad nominal se sustituye solo dentro
-- de esta transaccion revertida; su contrato productivo tiene pruebas propias.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $retirar_checks_nominales$
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
$retirar_checks_nominales$;

SET LOCAL session_replication_role = replica;
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
) VALUES (
    'decision:atestada:v2:prueba:1', repeat('1', 64),
    convert_to('nominal-sintetica', 'UTF8'), '{}'::jsonb, '{}'::jsonb,
    'principal:prueba', 'perfil:prueba',
    'bolsa.calculo_experiencia.oficial.confirmar',
    'calculo-oficial:' || repeat('2', 64), 'bolsa',
    'calculo_experiencia_oficial', repeat('3', 64),
    'confirmacion_calculo_oficial_experiencia',
    'correlacion_11111111111111111111111111111111', repeat('4', 64),
    repeat('5', 64), convert_to('motivo-sintetico', 'UTF8'),
    'motivos.prueba', 1, repeat('6', 64),
    'motivo_11111111111111111111111111111111', 'asignacion:prueba',
    'rol:prueba', 'rol:prueba', 1,
    clock_timestamp() - interval '1 minute',
    clock_timestamp() + interval '10 minutes', clock_timestamp()
);
SET LOCAL session_replication_role = origin;

CREATE OR REPLACE FUNCTION
vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
    p_decision_canonica bytea,
    p_motivo_canonico bytea
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $sustituto$
DECLARE
    documento jsonb;
BEGIN
    documento := convert_from(p_decision_canonica, 'UTF8')::jsonb;
    RETURN p_motivo_canonico IS NOT NULL AND EXISTS (
        SELECT 1
          FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
         WHERE decision_ref = documento ->> 'decision_ref'
    );
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$sustituto$;

CREATE FUNCTION pg_temp.ejecutar_prueba_consumo_atestado_v2()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $prueba$
DECLARE
    ahora timestamptz(6) := date_trunc('microseconds', clock_timestamp());
    verificada timestamptz(6) := ahora - interval '200 milliseconds';
    emitida timestamptz(6) := ahora - interval '100 milliseconds';
    expira timestamptz(6) := ahora + interval '4 seconds';
    decision_hasta timestamptz(6) := ahora + interval '5 minutes';
    configuracion record;
    raiz record;
    clave record;
    decision jsonb;
    capacidad jsonb;
    alterada jsonb;
    decision_bytes bytea;
    motivo bytea := convert_to('motivo-atestado-v2-prueba', 'UTF8');
    payload bytea := convert_to('payload-vec-ad-2-prueba', 'UTF8');
    sobre bytea := convert_to('cose-sign1-prueba', 'UTF8');
    evidencia bytea := convert_to('evidencia-verificacion-prueba', 'UTF8');
    resultado record;
    numero bigint;
BEGIN
    SELECT version.revision, version.huella_configuracion_sha256,
           version.publicada_en, version.expira_en
      INTO STRICT configuracion
      FROM vec_confianza_atestacion_v2.configuracion_confianza_version AS version
      JOIN vec_confianza_atestacion_v2.puntero_configuracion_actual AS puntero
        ON puntero.revision = version.revision
     ORDER BY puntero.orden DESC LIMIT 1;
    SELECT version.clave_id, version.version, version.clave_publica_spki,
           version.huella_clave_spki_sha256, version.valida_desde,
           version.valida_hasta, version.suite, version.audiencia_despliegue
      INTO STRICT raiz
      FROM vec_confianza_atestacion_v2.configuracion_raiz AS enlace
      JOIN vec_confianza_atestacion_v2.raiz_confianza_version AS version
        ON version.clave_id = enlace.clave_id
       AND version.version = enlace.version
     WHERE enlace.configuracion_revision = configuracion.revision
     ORDER BY version.clave_id COLLATE "C" LIMIT 1;
    SELECT version.* INTO STRICT clave
      FROM vec_autorizacion_atestada_v2.clave_capacidad_version AS version
      JOIN vec_autorizacion_atestada_v2.puntero_clave_capacidad AS puntero
        ON puntero.clave_id = version.clave_id
       AND puntero.version = version.version
     ORDER BY puntero.orden DESC LIMIT 1;

    decision := jsonb_build_object(
        'decision_ref', 'decision:atestada:v2:prueba:1',
        'motivo_huella_sha256', encode(sha256(motivo), 'hex'),
        'principal_id', 'principal:prueba',
        'accion', 'bolsa.calculo_experiencia.oficial.confirmar',
        'finalidad', 'confirmacion_calculo_oficial_experiencia',
        'recurso_ref', 'calculo-oficial:' || repeat('2', 64),
        'contexto_recurso_huella_sha256', repeat('3', 64),
        'correlacion_ref',
            'correlacion_11111111111111111111111111111111',
        'emitida_en', to_char(
            ahora - interval '1 second',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'valida_hasta', to_char(
            decision_hasta, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    );
    decision_bytes := convert_to(decision::text, 'UTF8');
    capacidad := jsonb_build_object(
        'esquema',
            'vec.autorizacion.capacidad-registro-consumo-atestado.v2',
        'clave_id', clave.clave_id, 'clave_version', clave.version,
        'emisor_id', clave.emisor_id, 'audiencia', clave.audiencia,
        'nonce', repeat('7', 64),
        'emitida_en', to_char(emitida, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'expira_en', to_char(expira, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'registro_ref', 'registro:atestacion:v2:prueba:1',
        'consumo_ref', 'consumo:atestacion:v2:prueba:1',
        'decision_ref', decision ->> 'decision_ref',
        'huella_decision_sha256', encode(sha256(decision_bytes), 'hex'),
        'huella_motivo_sha256', encode(sha256(motivo), 'hex'),
        'huella_payload_vec_ad_2_sha256', encode(sha256(payload), 'hex'),
        'huella_sobre_cose_sign1_sha256', encode(sha256(sobre), 'hex'),
        'huella_evidencia_verificacion_sha256',
            encode(sha256(evidencia), 'hex'),
        'principal_id', decision ->> 'principal_id',
        'accion', decision ->> 'accion',
        'finalidad', decision ->> 'finalidad',
        'sujeto_ref', 'hmac-sha256:personas:' || repeat('8', 64),
        'recurso_ref', decision ->> 'recurso_ref',
        'contexto_recurso_huella_sha256',
            decision ->> 'contexto_recurso_huella_sha256',
        'correlacion_ref', decision ->> 'correlacion_ref',
        'decision_valida_hasta', decision ->> 'valida_hasta',
        'efecto_ref', 'efecto:calculo:prueba:1',
        'huella_efecto_sha256', repeat('9', 64),
        'verificada_en',
            to_char(verificada, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        'revision_confianza', configuracion.revision,
        'huella_configuracion_sha256',
            configuracion.huella_configuracion_sha256,
        'configuracion_publicada_en', to_char(
            configuracion.publicada_en,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'configuracion_expira_en', to_char(
            configuracion.expira_en,
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_clave_id', raiz.clave_id, 'raiz_version', raiz.version,
        'huella_raiz_spki_sha256', raiz.huella_clave_spki_sha256,
        'raiz_valida_desde', to_char(
            raiz.valida_desde, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'raiz_valida_hasta', to_char(
            raiz.valida_hasta, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'suite', raiz.suite,
        'audiencia_despliegue', raiz.audiencia_despliegue
    );
    capacidad := capacidad || jsonb_build_object(
        'mac_sha256', encode(public.hmac(
            vec_autorizacion_atestada_v2.preimagen_capacidad(capacidad),
            clave.secreto_hmac, 'sha256'
        ), 'hex')
    );
    IF (SELECT count(*) FROM jsonb_object_keys(capacidad)) <> 39 THEN
        RAISE EXCEPTION 'vector de capacidad no tiene 39 campos';
    END IF;

    SELECT * INTO resultado
      FROM vec_autorizacion_atestada_v2.
        registrar_y_consumir_decision_v2_atestada(
            decision_bytes, motivo, payload, sobre, evidencia,
            raiz.clave_publica_spki, capacidad
        );
    IF resultado.registro_ref IS DISTINCT FROM
           'registro:atestacion:v2:prueba:1'
       OR resultado.consumo_ref IS DISTINCT FROM
          'consumo:atestacion:v2:prueba:1'
       OR resultado.auditoria_ref IS NULL THEN
        RAISE EXCEPTION 'la puerta rechazo el vector valido';
    END IF;
    IF (SELECT count(*)
          FROM vec_autorizacion_atestada_v2.atestacion_decision_v2) <> 1
       OR (SELECT count(*)
             FROM vec_autorizacion_atestada_v2.consumo_capacidad_v2) <> 1
       OR (SELECT count(*)
             FROM vec_autorizacion_atestada_v2.consumo_decision_v2) <> 1
       OR (SELECT count(*)
             FROM vec_autorizacion_atestada_v2.auditoria_consumo_v2) <> 1 THEN
        RAISE EXCEPTION 'el grafo de consumo no es completo';
    END IF;
    SELECT count(*) INTO numero
      FROM vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
        capacidad ->> 'decision_ref',
        capacidad ->> 'huella_decision_sha256',
        capacidad ->> 'efecto_ref', capacidad ->> 'huella_efecto_sha256',
        capacidad ->> 'nonce'
      );
    IF numero <> 1 THEN
        RAISE EXCEPTION 'la reconciliacion exacta fallo';
    END IF;

    alterada := jsonb_set(
        capacidad, '{recurso_ref}', '"calculo-oficial:alterado"'
    );
    SELECT count(*) INTO numero
      FROM vec_autorizacion_atestada_v2.
        registrar_y_consumir_decision_v2_atestada(
            decision_bytes, motivo, payload, sobre, evidencia,
            raiz.clave_publica_spki, alterada
        );
    IF numero <> 0 THEN
        RAISE EXCEPTION 'se acepto una capacidad con MAC alterado';
    END IF;

    alterada := capacidad || jsonb_build_object(
        'huella_configuracion_sha256', repeat('a', 64),
        'nonce', repeat('b', 64), 'registro_ref', 'registro:alterado',
        'consumo_ref', 'consumo:alterado', 'decision_ref', 'decision:alterada',
        'efecto_ref', 'efecto:alterado', 'huella_efecto_sha256', repeat('c', 64)
    );
    alterada := jsonb_set(
        alterada, '{mac_sha256}', to_jsonb(encode(public.hmac(
            vec_autorizacion_atestada_v2.preimagen_capacidad(alterada),
            clave.secreto_hmac, 'sha256'
        ), 'hex'))
    );
    SELECT count(*) INTO numero
      FROM vec_autorizacion_atestada_v2.
        registrar_y_consumir_decision_v2_atestada(
            decision_bytes, motivo, payload, sobre, evidencia,
            raiz.clave_publica_spki, alterada
        );
    IF numero <> 0 THEN
        RAISE EXCEPTION 'el catalogo alterado supero el cotejo vivo';
    END IF;

    alterada := capacidad || jsonb_build_object(
        'sujeto_ref', '12345678Z', 'nonce', repeat('d', 64),
        'registro_ref', 'registro:dni', 'consumo_ref', 'consumo:dni',
        'decision_ref', 'decision:dni', 'efecto_ref', 'efecto:dni',
        'huella_efecto_sha256', repeat('d', 64)
    );
    alterada := jsonb_set(
        alterada, '{mac_sha256}', to_jsonb(encode(public.hmac(
            vec_autorizacion_atestada_v2.preimagen_capacidad(alterada),
            clave.secreto_hmac, 'sha256'
        ), 'hex'))
    );
    SELECT count(*) INTO numero
      FROM vec_autorizacion_atestada_v2.
        registrar_y_consumir_decision_v2_atestada(
            decision_bytes, motivo, payload, sobre, evidencia,
            raiz.clave_publica_spki, alterada
        );
    IF numero <> 0 THEN
        RAISE EXCEPTION 'se acepto DNI directo como sujeto';
    END IF;

    BEGIN
        PERFORM * FROM vec_autorizacion_atestada_v2.
            registrar_y_consumir_decision_v2_atestada(
                decision_bytes, motivo, payload, sobre, evidencia,
                raiz.clave_publica_spki, capacidad
            );
        RAISE EXCEPTION 'el nonce y la decision se consumieron dos veces';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;
END
$prueba$;

SET SESSION AUTHORIZATION vec_ad2_consumidor_prueba;
SELECT pg_temp.ejecutar_prueba_consumo_atestado_v2();
RESET SESSION AUTHORIZATION;
ROLLBACK;
