-- Doble exclusivo de la base efímera. La prueba de falsificación se ejecuta
-- antes, contra la frontera real. Este reemplazo solo permite aislar filtros,
-- paginación y consumo único del almacén T13.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
CREATE OR REPLACE FUNCTION
vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(
    p_prueba jsonb, p_decision_canonica bytea, p_recurso_canonico bytea,
    p_correlacion_ref text, p_recurso_ref text, p_finalidad text,
    p_campos_exactos jsonb, p_actor_seudonimo text, p_auth_method text,
    p_auth_assurance text
)
RETURNS boolean
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT (
           (
               p_prueba ->> 'decision_ref' = 'decision_mecanica_t13'
               AND p_correlacion_ref = 'correlacion_' || repeat('1', 32)
               AND p_recurso_ref =
                   'consulta-accesos:sha256:' || repeat('2', 64)
           )
           OR
           (
               p_prueba ->> 'decision_ref' = 'decision_mecanica_t13_2'
               AND p_correlacion_ref = 'correlacion_' || repeat('3', 32)
               AND p_recurso_ref =
                   'consulta-accesos:sha256:' || repeat('4', 64)
           )
           OR
           (
               p_prueba ->> 'decision_ref' = 'decision_mecanica_t13_3'
               AND p_correlacion_ref = 'correlacion_' || repeat('6', 32)
               AND p_recurso_ref =
                   'consulta-accesos:sha256:' || repeat('7', 64)
           )
           OR
           (
               p_prueba ->> 'decision_ref' = 'decision:go:t13:exito'
               AND p_correlacion_ref =
                   'correlacion_' || repeat('9', 32)
               AND p_recurso_ref ~
                   '^consulta-accesos:sha256:[0-9a-f]{64}$'
           )
           OR
           (
               p_prueba ->> 'decision_ref' =
                   'decision:go:t13:cancelacion_commit'
               AND p_correlacion_ref =
                   'correlacion_' || repeat('7', 32)
               AND p_recurso_ref ~
                   '^consulta-accesos:sha256:[0-9a-f]{64}$'
           )
           OR
           (
               p_prueba ->> 'decision_ref' LIKE
                   'decision:go:t13:mutacion:%'
               AND p_correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
               AND p_recurso_ref ~
                   '^consulta-accesos:sha256:[0-9a-f]{64}$'
           )
       )
       AND p_prueba ->> 'huella_decision_sha256' =
           encode(sha256(p_decision_canonica), 'hex')
       AND CASE
           WHEN p_prueba ->> 'decision_ref' LIKE 'decision:go:t13:%'
               THEN p_prueba ->> 'principal_ref' =
                    'per_0123456789abcdefghijkl'
           ELSE p_prueba ->> 'principal_ref' = 'principal_mecanica_t13'
       END
       AND p_finalidad = 'control-interno'
       AND p_campos_exactos = '[
           "accion", "actor_seudonimizado", "expediente_ref", "finalidad",
           "modulo_id", "ocurrido_en", "recurso_ref", "resultado",
           "version_esquema", "version_objeto"
       ]'::jsonb
       AND p_actor_seudonimo =
           'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64)
       AND p_auth_method = 'sso' AND p_auth_assurance = 'alto'
       AND convert_from(p_recurso_canonico, 'UTF8')::jsonb ->
           'atributos' ->> 'actor_operador_seudonimizado' =
           p_actor_seudonimo
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(
        jsonb,bytea,bytea,text,text,text,jsonb,text,text,text
    ) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(
        jsonb,bytea,bytea,text,text,text,jsonb,text,text,text
    ) TO vec_bolsa_accesos_propietario;
COMMIT;
