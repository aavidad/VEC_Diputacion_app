-- Frontera V2 estrecha para decisiones del calculo oficial. El registro V2
-- existente conserva la autoridad y revalida identidad, RBAC/ABAC, motivo,
-- sesion y vigencia antes de insertar cada decision nueva.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regclass(
           'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_bolsa_calculo_experiencia_propietario'
              AND NOT rolcanlogin
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_bolsa_calculo_experiencia_aplicacion'
              AND NOT rolcanlogin
       )
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_calculo_experiencia_v1(text,text,text,text,text,text,text,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar la frontera V2 del calculo';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
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
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    decision vec_autorizacion.decision_autorizacion_solicitud_ligada_v2%ROWTYPE;
    accion_esperada text;
    tipo_recurso_esperado text;
    finalidad_esperada text;
    campos_esperados jsonb;
    vinculo jsonb;
    instante timestamptz(6);
BEGIN
    IF p_decision_ref IS NULL OR octet_length(p_decision_ref) NOT BETWEEN 1 AND 512
       OR p_decision_ref ~ '[[:space:][:cntrl:]]'
       OR p_huella_decision_sha256 !~ '^[0-9a-f]{64}$'
       OR p_correlacion_ref !~ '^correlacion_[0-9a-f]{32}$'
       OR p_recurso_ref IS NULL OR octet_length(p_recurso_ref) NOT BETWEEN 1 AND 512
       OR p_recurso_ref ~ '[[:space:][:cntrl:]]'
       OR p_contexto_recurso_huella_sha256 !~ '^[0-9a-f]{64}$' THEN
        RETURN false;
    END IF;

    CASE p_tipo
    WHEN 'lectura_fuentes' THEN
        accion_esperada := 'bolsa.calculo_experiencia.fuente.leer';
        tipo_recurso_esperado := 'fuente_calculo_experiencia';
        finalidad_esperada := 'calculo_oficial_experiencia';
        campos_esperados :=
            '["fuente_reglas","instantanea_experiencia","prueba_procedencia"]'::jsonb;
    WHEN 'escritura_resultado' THEN
        IF p_tipo_efecto = 'calculo_inicial' THEN
            accion_esperada := 'bolsa.calculo_experiencia.oficial.confirmar';
            tipo_recurso_esperado := 'calculo_experiencia_oficial';
            finalidad_esperada := 'confirmacion_calculo_oficial_experiencia';
        ELSIF p_tipo_efecto = 'rectificacion' THEN
            accion_esperada := 'bolsa.calculo_experiencia.oficial.rectificar';
            tipo_recurso_esperado :=
                'rectificacion_calculo_experiencia_oficial';
            finalidad_esperada :=
                'rectificacion_calculo_oficial_experiencia';
        ELSE
            RETURN false;
        END IF;
        campos_esperados :=
            '["auditoria","resultado_canonico","salida_eventos"]'::jsonb;
    ELSE
        RETURN false;
    END CASE;
    IF p_perfil_proteccion NOT IN ('externo_ordinario', 'interno_alto')
       OR p_tipo_efecto NOT IN ('calculo_inicial', 'rectificacion')
       OR (p_tipo_efecto = 'rectificacion'
           AND p_perfil_proteccion <> 'interno_alto')
       OR (p_tipo = 'lectura_fuentes'
           AND p_recurso_ref !~ '^fuente:[0-9a-f]{64}$')
       OR (p_tipo = 'escritura_resultado'
           AND p_tipo_efecto = 'calculo_inicial'
           AND p_recurso_ref !~ '^calculo-oficial:[0-9a-f]{64}$') THEN
        RETURN false;
    END IF;
    IF p_tipo = 'escritura_resultado'
       AND p_tipo_efecto = 'rectificacion'
       AND p_recurso_ref !~
           '^rectificacion-calculo-oficial:[0-9a-f]{64}$' THEN
        RETURN false;
    END IF;

    SELECT * INTO decision
      FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
     WHERE decision_ref = p_decision_ref
     FOR SHARE;
    instante := clock_timestamp();
    vinculo := decision.documento_v2 -> 'vinculo_autenticacion_actor';
    IF NOT FOUND
       OR decision.huella_decision_sha256 IS DISTINCT FROM
          p_huella_decision_sha256
       OR encode(sha256(decision.decision_canonica), 'hex') IS DISTINCT FROM
          p_huella_decision_sha256
       OR decision.accion IS DISTINCT FROM accion_esperada
       OR decision.modulo_id IS DISTINCT FROM 'bolsa'
       OR decision.tipo_recurso IS DISTINCT FROM tipo_recurso_esperado
       OR decision.finalidad IS DISTINCT FROM finalidad_esperada
       OR decision.correlacion_ref IS DISTINCT FROM p_correlacion_ref
       OR decision.recurso_ref IS DISTINCT FROM p_recurso_ref
       OR decision.contexto_recurso_huella_sha256 IS DISTINCT FROM
          p_contexto_recurso_huella_sha256
       OR decision.documento_v2 -> 'campos_permitidos' IS DISTINCT FROM
          campos_esperados
       OR decision.documento_v2 -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR decision.documento_v2 -> 'concedida' IS DISTINCT FROM 'true'::jsonb
       OR jsonb_typeof(vinculo) IS DISTINCT FROM 'object'
       OR vinculo ->> 'metodo_observado' IS NULL
       OR vinculo ->> 'metodo_observado' = 'demo'
       OR (CASE p_perfil_proteccion
          WHEN 'externo_ordinario' THEN NOT COALESCE(
              decision.documento_v2 ->> 'garantia_minima'
                  IN ('sustancial', 'alto')
              AND vinculo ->> 'garantia_observada'
                  IN ('sustancial', 'alto')
              AND (
                  decision.documento_v2 ->> 'garantia_minima' = 'sustancial'
                  OR vinculo ->> 'garantia_observada' = 'alto'
              )
              AND vinculo ->> 'superficie' = 'externa_personal'
              AND vinculo -> 'cuenta_privilegiada' = 'false'::jsonb,
              false
          )
          WHEN 'interno_alto' THEN NOT COALESCE(
              decision.documento_v2 ->> 'garantia_minima' = 'alto'
              AND vinculo ->> 'garantia_observada' = 'alto'
              AND (
                  (vinculo ->> 'superficie' = 'interna_corporativa'
                   AND vinculo -> 'cuenta_privilegiada' = 'false'::jsonb)
                  OR
                  (vinculo ->> 'superficie' = 'administracion_privilegiada'
                   AND vinculo -> 'cuenta_privilegiada' = 'true'::jsonb)
              ), false
          )
          ELSE true
          END)
       OR instante < decision.emitida_en
       OR instante >= decision.valida_hasta
       OR decision.registrada_en > instante
       OR (p_tipo = 'escritura_resultado'
           AND instante - decision.registrada_en > interval '30 seconds') THEN
        RETURN false;
    END IF;
    RETURN true;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
        text, text, text, text, text, text, text, text
    ) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO
    vec_bolsa_calculo_experiencia_propietario,
    vec_bolsa_calculo_experiencia_aplicacion;
GRANT REFERENCES (decision_ref) ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    TO vec_bolsa_calculo_experiencia_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
        text, text, text, text, text, text, text, text
    ) TO vec_bolsa_calculo_experiencia_aplicacion;

COMMENT ON FUNCTION
    vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
        text, text, text, text, text, text, text, text
    ) IS
    'Coteja una decision V2 con la accion, recurso, campos y vigencia del calculo oficial. Este paquete no concede al runtime el registrador V2: la decision debe proceder de la futura puerta atestada VEC-AD-2.';
COMMIT;
