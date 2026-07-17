-- Fixture efimero para demostrar que una espera de bloqueo no puede conservar
-- una hora antigua. La frontera permisiva es solo mecanica y esta base se
-- destruye al terminar; ningun rol runtime recibe EXECUTE.
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

CREATE TABLE public.vec_panel_concurrencia_expiracion (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    operacion jsonb NOT NULL,
    prueba jsonb NOT NULL,
    decision_canonica bytea NOT NULL,
    motivo_canonico bytea NOT NULL,
    correlacion_ref text NOT NULL
);
REVOKE ALL ON TABLE public.vec_panel_concurrencia_expiracion FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO vec_bolsa_panel_propietario;
GRANT SELECT, INSERT ON TABLE public.vec_panel_concurrencia_expiracion
    TO vec_bolsa_panel_propietario;

BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_panel.publicar_proyeccion_panel_v1(jsonb_build_object(
    'esquema', 'vec.bolsa.panel.proyeccion.v1',
    'selector', jsonb_build_object(
        'clase', 'organizacion',
        'organizacion_ref', 'org_aaaaaaaaaaaaaaaa'
    ),
    'revision', 'rev_bbbbbbbbbbbbbbbb',
    'actualizada_en', to_char(
        (clock_timestamp() - interval '1 second') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    ),
    'indicadores', jsonb_build_object(
        'convocatorias_borrador', 0,
        'convocatorias_revision', 0,
        'convocatorias_pendientes_firma', 0,
        'convocatorias_publicadas', 1,
        'bolsas_activas', 1,
        'bolsas_suspendidas', 0,
        'bolsas_agotadas', 0,
        'llamamientos_pendientes', 0,
        'llamamientos_en_curso', 0,
        'llamamientos_vencen_hoy', 0,
        'documentos_pendientes_firma', 0,
        'incidencias_abiertas', 0
    ),
    'convocatorias', '[]'::jsonb,
    'actuaciones_pendientes', '[]'::jsonb
));

SET LOCAL ROLE vec_bolsa_panel_propietario;
DO $fixture$
DECLARE
    instante timestamptz := date_trunc('microseconds', clock_timestamp());
    decision bytea := convert_to(
        'decision-v2-concurrencia-no-autoridad', 'UTF8'
    );
    motivo bytea := convert_to(
        '{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada","referencia":{"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","catalogo_id":"motivos.panel","catalogo_version":1,"entrada_clave":"motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}',
        'UTF8'
    );
    huella_decision text := encode(sha256(decision), 'hex');
BEGIN
    INSERT INTO vec_bolsa_panel.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1, huella_sobre_sha256,
        clave_id, revision_confianza, verificada_en, valida_desde,
        valida_hasta, registrada_en
    ) VALUES (
        'decision:panel:concurrencia:1', 'ate_cccccccccccccccc', 1,
        'activa', huella_decision,
        convert_to('evidencia-concurrencia', 'UTF8'),
        encode(sha256(convert_to('evidencia-concurrencia', 'UTF8')), 'hex'),
        convert_to('sobre-cose-concurrencia', 'UTF8'),
        encode(sha256(convert_to('sobre-cose-concurrencia', 'UTF8')), 'hex'),
        'clave-concurrencia-no-productiva', 'confianza-concurrencia',
        instante, instante - interval '1 second',
        instante + interval '10 seconds', instante
    );
    INSERT INTO vec_bolsa_panel.atestacion_autorizacion_actual(
        decision_ref, atestacion_ref, version, estado, actualizada_en
    ) VALUES (
        'decision:panel:concurrencia:1', 'ate_cccccccccccccccc', 1,
        'activa', instante
    );
    INSERT INTO public.vec_panel_concurrencia_expiracion(
        operacion, prueba, decision_canonica, motivo_canonico,
        correlacion_ref
    ) VALUES (
        jsonb_build_object(
            'esquema',
                'vec.bolsa.panel.interno.consulta-postgresql.v1',
            'clase_ambito', 'organizacion',
            'organizacion_ref', 'org_aaaaaaaaaaaaaaaa',
            'accion', 'bolsa.panel_interno.consultar',
            'recurso_ref', 'panel:org_aaaaaaaaaaaaaaaa',
            'consultada_en', to_char(instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ),
        jsonb_build_object(
            'esquema_huella',
                'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
            'decision_ref', 'decision:panel:concurrencia:1',
            'huella_decision_sha256', huella_decision,
            'verificada_en', to_char(instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        ),
        decision, motivo,
        'correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
    );
END
$fixture$;
COMMIT;
