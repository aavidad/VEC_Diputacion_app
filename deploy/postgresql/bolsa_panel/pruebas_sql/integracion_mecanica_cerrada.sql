-- Esta prueba sustituye la frontera de autorizacion solo dentro de una
-- transaccion revertida. Valida la mecanica del panel, no criptografia ni PDP.
-- El runtime continua sin EXECUTE antes, durante y despues de la prueba.
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;

CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_decision_panel_bolsa_v2(
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_correlacion_ref text,
    p_recurso_ref text,
    p_huella_contexto_sha256 text,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $prueba$ SELECT true $prueba$;

SET LOCAL ROLE vec_bolsa_panel_proyector;
SELECT vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb_build_object(
    'esquema', 'vec.bolsa.panel.proyeccion.v1',
    'selector', jsonb_build_object(
        'clase', 'unidad_gestion',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', 'uni_fedcba9876543210'
    ),
    'revision', 'rev_1111111111111111',
    'actualizada_en', to_char(
        (clock_timestamp() - interval '1 second') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    ),
    'indicadores', jsonb_build_object(
        'convocatorias_borrador', 1,
        'convocatorias_revision', 2,
        'convocatorias_pendientes_firma', 3,
        'convocatorias_publicadas', 4,
        'bolsas_activas', 5,
        'bolsas_suspendidas', 6,
        'bolsas_agotadas', 7,
        'llamamientos_pendientes', 8,
        'llamamientos_en_curso', 9,
        'llamamientos_vencen_hoy', 10,
        'documentos_pendientes_firma', 11,
        'incidencias_abiertas', 12
    ),
    'convocatorias', jsonb_build_array(jsonb_build_object(
        'convocatoria_ref', 'cnv_2222222222222222',
        'categoria_clave', 'auxiliar.administrativo',
        'estado_clave', 'revision.tecnica',
        'plazo_cierra_en', '2026-08-01T10:00:00.000000Z',
        'numero_solicitudes', 25,
        'numero_pendientes', 4
    )),
    'actuaciones_pendientes', jsonb_build_array(jsonb_build_object(
        'actuacion_ref', 'act_3333333333333333',
        'recurso_ref', 'exp_4444444444444444',
        'tipo_clave', 'revision.documental',
        'estado_clave', 'pendiente',
        'prioridad_clave', 'alta',
        'fecha_limite', '2026-07-31T12:00:00.000000Z',
        'numero_elementos', 4
    ))
));

DO $entrada_cerrada$
BEGIN
    BEGIN
        PERFORM vec_bolsa_panel.publicar_proyeccion_panel_v1(
            '{"dni":"00000000T"}'::jsonb
        );
        RAISE EXCEPTION 'el publicador acepto una clave personal ajena';
    EXCEPTION WHEN invalid_parameter_value THEN
        NULL;
    END;
END
$entrada_cerrada$;

RESET ROLE;
SET LOCAL ROLE vec_bolsa_panel_propietario;

DO $mecanica$
DECLARE
    instante timestamptz := date_trunc('microseconds', clock_timestamp());
    decision bytea := convert_to('decision-v2-sintetica-no-autoridad', 'UTF8');
    motivo bytea := convert_to(
        '{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada","referencia":{"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","catalogo_id":"motivos.panel","catalogo_version":1,"entrada_clave":"motivo_0123456789abcdef0123456789abcdef"}}',
        'UTF8'
    );
    huella_decision text;
    operacion jsonb;
    prueba jsonb;
    primera bytea;
    segunda bytea;
    panel jsonb;
BEGIN
    huella_decision := encode(sha256(decision), 'hex');
    operacion := jsonb_build_object(
        'esquema', 'vec.bolsa.panel.interno.consulta-postgresql.v1',
        'clase_ambito', 'unidad_gestion',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', 'uni_fedcba9876543210',
        'accion', 'bolsa.panel_interno.consultar',
        'recurso_ref',
            'panel:org_0123456789abcdef:uni_fedcba9876543210',
        'consultada_en', to_char(instante AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );
    prueba := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
        'decision_ref', 'decision:panel:prueba:1',
        'huella_decision_sha256', huella_decision,
        'verificada_en', to_char(instante AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    );

    INSERT INTO vec_bolsa_panel.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1, huella_sobre_sha256,
        clave_id, revision_confianza, verificada_en, valida_desde,
        valida_hasta, registrada_en
    ) VALUES (
        'decision:panel:prueba:1', 'ate_5555555555555555', 1, 'activa',
        huella_decision, convert_to('evidencia-sintetica', 'UTF8'),
        encode(sha256(convert_to('evidencia-sintetica', 'UTF8')), 'hex'),
        convert_to('sobre-cose-sintetico', 'UTF8'),
        encode(sha256(convert_to('sobre-cose-sintetico', 'UTF8')), 'hex'),
        'clave-prueba-no-productiva', 'confianza-prueba', instante,
        instante - interval '1 second', instante + interval '1 minute',
        instante
    );
    INSERT INTO vec_bolsa_panel.atestacion_autorizacion_actual(
        decision_ref, atestacion_ref, version, estado, actualizada_en
    ) VALUES (
        'decision:panel:prueba:1', 'ate_5555555555555555', 1, 'activa',
        instante
    );

    SELECT resultado.panel_canonico INTO STRICT primera
      FROM vec_bolsa_panel.consultar_panel_interno_v1(
          operacion, prueba, decision, motivo,
          'correlacion_0123456789abcdef0123456789abcdef'
      ) AS resultado;
    SELECT resultado.panel_canonico INTO STRICT segunda
      FROM vec_bolsa_panel.consultar_panel_interno_v1(
          operacion, prueba, decision, motivo,
          'correlacion_0123456789abcdef0123456789abcdef'
      ) AS resultado;
    IF primera IS DISTINCT FROM segunda THEN
        RAISE EXCEPTION 'la repeticion exacta no fue idempotente';
    END IF;
    panel := convert_from(primera, 'UTF8')::jsonb;
    IF panel ->> 'esquema' <> 'vec.bolsa.panel.interno.v1'
       OR panel -> 'origen' ->> 'demostracion' <> 'false'
       OR panel -> 'selector' ->> 'unidad_gestion_ref' <>
          'uni_fedcba9876543210'
       OR jsonb_array_length(panel -> 'convocatorias') <> 1
       OR jsonb_array_length(panel -> 'actuaciones_pendientes') <> 1
       OR panel::text ~* '(dni|nombre|correo|telefono|candidato)'
       OR (SELECT count(*) FROM vec_bolsa_panel.consulta_confirmada) <> 1
       OR (SELECT count(*) FROM vec_bolsa_panel.auditoria) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM vec_bolsa_panel.auditoria
            WHERE encode(sha256(
                decode(huella_anterior_sha256, 'hex') || registro_canonico
            ), 'hex') = huella_auditoria_sha256
       ) THEN
        RAISE EXCEPTION 'panel, minimizacion o auditoria invalidos';
    END IF;

    BEGIN
        PERFORM resultado.panel_canonico
          FROM vec_bolsa_panel.consultar_panel_interno_v1(
              operacion || jsonb_build_object(
                  'recurso_ref', 'panel:org_ffffffffffffffff'
              ),
              prueba, decision, motivo,
              'correlacion_0123456789abcdef0123456789abcdef'
          ) AS resultado;
        RAISE EXCEPTION 'selector cruzado aceptado';
    EXCEPTION WHEN invalid_parameter_value THEN
        NULL;
    END;
END
$mecanica$;

ROLLBACK;
