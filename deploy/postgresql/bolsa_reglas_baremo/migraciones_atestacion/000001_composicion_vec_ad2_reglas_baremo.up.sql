-- Puente minimo entre la puerta VEC-AD-2 y el propietario RLS del almacen de
-- reglas. No concede DML ni acceso de lectura a las tablas centrales.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:composicion-vec-ad2:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_reglas_baremo') IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_bolsa_reglas_baremo_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       )
       OR to_regprocedure(
           'vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(text,text,text,text,text)'
       ) IS NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_autorizacion_atestada_v2.consumo_decision_v2'::regclass
              AND conname = 'consumo_decision_v2_vinculo_reglas_unico'
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_autorizacion_atestada_v2.auditoria_consumo_v2'::regclass
              AND conname = 'auditoria_consumo_v2_vinculo_reglas_unico'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para componer reglas con VEC-AD-2';
    END IF;
END
$prevalidacion$;

-- Las claves candidatas permiten que el recibo modular demuestre por FK el
-- registro, consumo, decision, efecto y asiento central exactos.
ALTER TABLE vec_autorizacion_atestada_v2.consumo_decision_v2
    ADD CONSTRAINT consumo_decision_v2_vinculo_reglas_unico UNIQUE (
        consumo_ref, registro_ref, decision_ref, huella_decision_sha256,
        efecto_ref, huella_efecto_sha256
    );
ALTER TABLE vec_autorizacion_atestada_v2.auditoria_consumo_v2
    ADD CONSTRAINT auditoria_consumo_v2_vinculo_reglas_unico UNIQUE (
        auditoria_ref, consumo_ref, registro_ref, decision_ref, efecto_ref,
        huella_registro_sha256
    );

-- Lector de snapshot fresco para el mismo statement PL/pgSQL que acaba de
-- consumir. STABLE no es suficiente aqui: conservaria el snapshot anterior a
-- la llamada VOLATILE. No se concede a ningun LOGIN ni grupo runtime.
CREATE FUNCTION
vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(
    p_registro_ref text,
    p_consumo_ref text,
    p_auditoria_ref text,
    p_decision_ref text,
    p_huella_decision_sha256 text,
    p_efecto_ref text,
    p_huella_efecto_sha256 text
)
RETURNS TABLE (
    registro_ref text,
    consumo_ref text,
    auditoria_ref text,
    consumida_en timestamptz(6),
    huella_auditoria_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF vec_autorizacion_atestada_v2.texto_tecnico_valido(
           p_registro_ref, 512
       ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.texto_tecnico_valido(
              p_consumo_ref, 512
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.texto_tecnico_valido(
              p_auditoria_ref, 512
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.texto_tecnico_valido(
              p_decision_ref, 512
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.texto_tecnico_valido(
              p_efecto_ref, 512
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.huella_sha256_valida(
              p_huella_decision_sha256
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v2.huella_sha256_valida(
              p_huella_efecto_sha256
          ) IS NOT TRUE THEN
        RETURN;
    END IF;
    RETURN QUERY
    SELECT consumo.registro_ref, consumo.consumo_ref,
           auditoria.auditoria_ref, consumo.consumida_en,
           auditoria.huella_registro_sha256
      FROM vec_autorizacion_atestada_v2.consumo_decision_v2 AS consumo
      JOIN vec_autorizacion_atestada_v2.auditoria_consumo_v2 AS auditoria
        ON auditoria.consumo_ref = consumo.consumo_ref
       AND auditoria.registro_ref = consumo.registro_ref
       AND auditoria.decision_ref = consumo.decision_ref
       AND auditoria.efecto_ref = consumo.efecto_ref
     WHERE consumo.registro_ref = p_registro_ref
       AND consumo.consumo_ref = p_consumo_ref
       AND auditoria.auditoria_ref = p_auditoria_ref
       AND consumo.decision_ref = p_decision_ref
       AND consumo.huella_decision_sha256 = p_huella_decision_sha256
       AND consumo.efecto_ref = p_efecto_ref
       AND consumo.huella_efecto_sha256 = p_huella_efecto_sha256;
END
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(
        text, text, text, text, text, text, text
    ) FROM PUBLIC, vec_autorizacion_atestada_v2_consumidor;

GRANT REFERENCES (
    registro_ref, decision_ref, huella_decision_sha256
) ON vec_autorizacion_atestada_v2.atestacion_decision_v2
  TO vec_bolsa_reglas_baremo_propietario;
GRANT REFERENCES (
    consumo_ref, registro_ref, decision_ref, huella_decision_sha256,
    efecto_ref, huella_efecto_sha256
) ON vec_autorizacion_atestada_v2.consumo_decision_v2
  TO vec_bolsa_reglas_baremo_propietario;
GRANT REFERENCES (
    auditoria_ref, consumo_ref, registro_ref, decision_ref, efecto_ref,
    huella_registro_sha256
) ON vec_autorizacion_atestada_v2.auditoria_consumo_v2
  TO vec_bolsa_reglas_baremo_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v2
    TO vec_bolsa_reglas_baremo_propietario;

-- SECURITY DEFINER conserva current_user como propietario del modulo. La
-- puerta central sigue verificando session_user: un LOGIN consumidor valido
-- debe tener una sola membresia directa, la de VEC-AD-2.
-- La puerta cruda se retira del grupo generico: de otro modo ese grupo podria
-- confirmar el consumo central sin materializar el efecto modular.
REVOKE EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) FROM vec_autorizacion_atestada_v2_consumidor;
REVOKE EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
        text, text, text, text, text
    ) FROM vec_autorizacion_atestada_v2_consumidor;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) TO vec_bolsa_reglas_baremo_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.obtener_vinculo_consumo_modular_v2(
        text, text, text, text, text, text, text
    ) TO vec_bolsa_reglas_baremo_propietario;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
        text, text, text, text, text
    ) TO vec_bolsa_reglas_baremo_propietario;

COMMIT;
