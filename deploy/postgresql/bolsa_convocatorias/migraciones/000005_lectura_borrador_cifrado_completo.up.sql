-- La lectura de un borrador nunca reconstruye una DEK, AAD o procedencia a
-- partir de la proyeccion ligera de 000003. 000004 ya conserva el paquete
-- criptografico inmutable y ligado por FK; esta migracion solo lo relee bajo
-- la misma decision PDP consumida y auditada por el wrapper de lectura.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $version_unicode$
BEGIN
    IF unicode_version() <> '16.0' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000005 requiere revisar perfil Unicode: PostgreSQL no usa Unicode 16.0';
    END IF;
END
$version_unicode$;

-- Paridad con SelectorListaBorradores.Validar de Go. x/text v0.40 usa
-- Unicode 15 hasta Go 1.26 y Unicode 17 desde Go 1.27; PostgreSQL 18 usa
-- Unicode 16. El perfil comun NFC V1 excluye las 82 runas cuyas propiedades
-- NFC cambian entre esas ediciones y mantiene estable el contrato
-- ante cambios de toolchain. PostgreSQL garantiza UTF-8 valido; aqui tambien
-- se cierran NFC, presupuesto por runas/bytes, TrimSpace, Cc, Cf y U+FFFD.
CREATE FUNCTION
vec_bolsa_convocatorias.texto_selector_borradores_valido_v1(p_texto text)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    indice integer;
    codigo integer;
    primero integer;
    ultimo integer;
BEGIN
    -- Guard deliberado: un upgrade de PostgreSQL debe revisar y versionar el
    -- perfil aun cuando sus exclusiones ya anticipen Unicode 17.
    IF unicode_version() <> '16.0' THEN
        RETURN false;
    END IF;
    IF p_texto IS NULL THEN
        RETURN false;
    END IF;
    IF p_texto = '' THEN
        RETURN true;
    END IF;
    IF octet_length(p_texto) > 720 OR char_length(p_texto) > 180
       OR normalize(p_texto, NFC) IS DISTINCT FROM p_texto THEN
        RETURN false;
    END IF;
    primero := ascii(substr(p_texto, 1, 1));
    ultimo := ascii(substr(p_texto, char_length(p_texto), 1));
    IF primero IN (32,133,160,5760,8232,8233,8239,8287,12288)
       OR primero BETWEEN 8192 AND 8202
       OR ultimo IN (32,133,160,5760,8232,8233,8239,8287,12288)
       OR ultimo BETWEEN 8192 AND 8202 THEN
        RETURN false;
    END IF;
    FOR indice IN 1..char_length(p_texto) LOOP
        codigo := ascii(substr(p_texto, indice, 1));
        IF codigo BETWEEN 0 AND 31 OR codigo BETWEEN 127 AND 159
           OR codigo = 65533
           -- Perfil NFC comun Unicode 15/16/17. La lista sale
           -- de UnicodeData/DerivedNormalizationProps 16.0 y de las tablas
           -- data15/data17 de x/text v0.40; los numeros se corresponden con
           -- los rangos U+ documentados en DEC-097.
           OR codigo = 2199                         -- U+0897
           OR codigo BETWEEN 6863 AND 6877          -- U+1ACF..U+1ADD
           OR codigo BETWEEN 6880 AND 6891          -- U+1AE0..U+1AEB
           OR codigo IN (67017,67026,67034,67044)  -- U+105C9/D2/DA/E4
           OR codigo BETWEEN 68969 AND 68973        -- U+10D69..U+10D6D
           OR codigo BETWEEN 69370 AND 69371        -- U+10EFA..U+10EFB
           OR codigo BETWEEN 70530 AND 70533        -- U+11382..U+11385
           OR codigo IN (70539,70542)               -- U+1138B/U+1138E
           OR codigo BETWEEN 70544 AND 70545        -- U+11390..U+11391
           OR codigo IN (70584,70587,70594,70597)  -- U+113B8/BB/C2/C5
           OR codigo BETWEEN 70599 AND 70601        -- U+113C7..U+113C9
           OR codigo BETWEEN 70606 AND 70608        -- U+113CE..U+113D0
           OR codigo BETWEEN 90398 AND 90409        -- U+1611E..U+16129
           OR codigo = 90415                        -- U+1612F
           OR codigo = 93539                        -- U+16D63
           OR codigo BETWEEN 93543 AND 93546        -- U+16D67..U+16D6A
           OR codigo BETWEEN 124398 AND 124399      -- U+1E5EE..U+1E5EF
           OR codigo IN (124643,124646,124661)     -- U+1E6E3/E6/F5
           OR codigo BETWEEN 124654 AND 124655      -- U+1E6EE..U+1E6EF
           OR codigo IN (
               173,1536,1564,1757,1807,2192,2193,2274,6158,8203,
               65279,65529,69821,69837,917505,917536
           )
           OR codigo BETWEEN 1537 AND 1541
           OR codigo BETWEEN 8204 AND 8207
           OR codigo BETWEEN 8234 AND 8238
           OR codigo BETWEEN 8288 AND 8292
           OR codigo BETWEEN 8294 AND 8303
           OR codigo BETWEEN 65530 AND 65531
           OR codigo BETWEEN 78896 AND 78911
           OR codigo BETWEEN 113824 AND 113827
           OR codigo BETWEEN 119155 AND 119162
           OR codigo BETWEEN 917537 AND 917631 THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_bolsa_convocatorias.selector_lista_borradores_valido_v1(p_selector jsonb)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT jsonb_typeof(p_selector) = 'object'
       AND vec_bolsa_convocatorias.objeto_json_exacto(
               p_selector, ARRAY['categoria','cursor','limite','texto']
           ) IS TRUE
       AND jsonb_typeof(p_selector -> 'limite') = 'number'
       AND (p_selector ->> 'limite') ~ '^[1-9][0-9]?$'
       AND (p_selector ->> 'limite')::integer BETWEEN 1 AND 50
       AND jsonb_typeof(p_selector -> 'cursor') = 'string'
       AND (p_selector ->> 'cursor' = '' OR
            p_selector ->> 'cursor' ~ '^cursor-borrador-[0-9a-f]{64}$')
       AND jsonb_typeof(p_selector -> 'texto') = 'string'
       AND vec_bolsa_convocatorias.texto_selector_borradores_valido_v1(
               p_selector ->> 'texto'
           ) IS TRUE
       AND jsonb_typeof(p_selector -> 'categoria') = 'string'
       AND (p_selector ->> 'categoria' = '' OR
            p_selector ->> 'categoria' ~
                '^[a-z0-9][a-z0-9._-]{0,79}$')
$funcion$;

DROP FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    jsonb,jsonb,jsonb,bytea,bytea
);

CREATE FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    p_selector jsonb, p_lectura jsonb, p_prueba jsonb,
    p_decision_canonica bytea, p_contexto_recurso_canonico bytea
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.selector_lista_borradores_valido_v1(
           p_selector
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'selector de borradores invalido';
    END IF;
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_ejecutor_consulta', true
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
           p_contexto_recurso_canonico, p_lectura,
           'borradores:' || (p_lectura ->> 'organizacion_ref'), true
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           'bolsa.convocatoria.borrador.listar',
           'coleccion_versiones_convocatoria_gobernada',
           p_lectura ->> 'recurso_ref',
           'consulta_interna_convocatorias',
           '["version_convocatoria"]'::jsonb
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'listado de borradores no revalidado con V2';
    END IF;
    RETURN vec_bolsa_convocatorias.listar_borradores_interna_v1(
        p_selector, p_lectura
    );
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    jsonb,jsonb,jsonb,bytea,bytea
) FROM PUBLIC,
       vec_bolsa_convocatorias_proyector_gobierno,
       vec_bolsa_convocatorias_registrador_atestacion,
       vec_bolsa_convocatorias_verificador_recibo;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.listar_borradores_v1(
    jsonb,jsonb,jsonb,bytea,bytea
) TO vec_bolsa_convocatorias_ejecutor_consulta;

DROP FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text,jsonb,jsonb,bytea,bytea
);

CREATE FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    p_referencia text, p_lectura jsonb, p_prueba jsonb,
    p_decision_canonica bytea, p_contexto_recurso_canonico bytea
)
RETURNS TABLE (
    metadatos jsonb,
    aad_canonica bytea, huella_aad_sha256 text,
    perfil jsonb,
    esquema_envoltura text, clave_maestra_ref text,
    version_clave bigint, material_clave_envuelto bytea,
    huella_envoltura_sha256 text,
    esquema_sobre text, nonce bytea, contenido_cifrado bytea,
    huella_sobre_sha256 text,
    atestacion_kms jsonb, procedencia jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
BEGIN
    IF vec_bolsa_convocatorias.identidad_runtime_borrador_valida(
           'vec_bolsa_convocatorias_ejecutor_consulta', true
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.contexto_lectura_borrador_valido(
           p_contexto_recurso_canonico, p_lectura, p_referencia, false
       ) IS NOT TRUE
       OR vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
           p_prueba, p_decision_canonica, p_contexto_recurso_canonico,
           'bolsa.convocatoria.borrador.consultar',
           'version_convocatoria_gobernada', p_referencia,
           'consulta_interna_convocatorias',
           '["version_convocatoria"]'::jsonb
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'detalle de borrador no revalidado';
    END IF;

    RETURN QUERY
    SELECT d.metadatos, d.aad_canonica, d.huella_aad_sha256,
           d.perfil, d.esquema_envoltura, d.clave_maestra_ref,
           d.version_clave, d.material_clave_envuelto,
           d.huella_envoltura_sha256, d.esquema_sobre, d.nonce,
           d.contenido_cifrado, d.huella_sobre_sha256,
           d.atestacion_kms, d.procedencia
      FROM (
        SELECT r.metadatos, c.aad_canonica, c.huella_aad_sha256,
               jsonb_build_object(
                   'referencia', c.perfil_ref,
                   'version', c.perfil_version,
                   'huella_contenido_sha256', c.huella_perfil_sha256,
                   'algoritmo_aead', c.algoritmo_aead,
                   'algoritmo_envoltura_clave', c.algoritmo_envoltura_clave
               ) AS perfil,
               'bolsa.convocatoria.borrador.clave-envuelta.v1'::text
                   AS esquema_envoltura,
               c.atestacion_kms ->> 'clave_maestra_ref' AS clave_maestra_ref,
               (c.atestacion_kms ->> 'version_clave')::bigint AS version_clave,
               c.material_clave_envuelto, c.huella_envoltura_sha256,
               'bolsa.convocatoria.borrador.sobre-aead.v1'::text AS esquema_sobre,
               c.nonce, c.contenido_cifrado, c.huella_sobre_sha256,
               c.atestacion_kms, c.procedencia
          FROM vec_bolsa_convocatorias.obtener_borrador_interna_v1(
                   p_referencia, p_lectura
               ) AS r
          JOIN vec_bolsa_convocatorias.borrador_convocatoria_version AS v
            ON v.referencia = p_referencia
           AND v.huella_sobre_cifrado_sha256 = r.huella_sobre_cifrado_sha256
          JOIN vec_bolsa_convocatorias.cifrado_kms_borrador AS c
            ON c.convocatoria_id = v.convocatoria_id
           AND c.secuencia = v.secuencia AND c.revision = v.revision
         WHERE c.contenido_cifrado = r.sobre_cifrado
           AND c.contenido_cifrado = v.sobre_cifrado
           AND c.huella_contenido_cifrado_sha256 =
               v.huella_sobre_cifrado_sha256
           AND c.nonce = r.nonce
           AND c.atestacion_kms ->> 'atestacion_ref' = r.atestacion_cifrado_ref
           AND c.atestacion_kms -> 'firma' ->> 'huella_preimagen_sha256' =
               r.huella_atestacion_cifrado_sha256
      ) AS d;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'paquete criptografico durable de borrador inconsistente';
    END IF;
END
$funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text,jsonb,jsonb,bytea,bytea
) FROM PUBLIC,
       vec_bolsa_convocatorias_proyector_gobierno,
       vec_bolsa_convocatorias_registrador_atestacion,
       vec_bolsa_convocatorias_verificador_recibo;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text,jsonb,jsonb,bytea,bytea
) TO vec_bolsa_convocatorias_ejecutor_consulta;

REVOKE ALL ON FUNCTION
    vec_bolsa_convocatorias.selector_lista_borradores_valido_v1(jsonb)
    FROM PUBLIC,
         vec_bolsa_convocatorias_ejecutor_consulta,
         vec_bolsa_convocatorias_proyector_gobierno,
         vec_bolsa_convocatorias_registrador_atestacion,
         vec_bolsa_convocatorias_verificador_recibo;
REVOKE ALL ON FUNCTION
    vec_bolsa_convocatorias.texto_selector_borradores_valido_v1(text)
    FROM PUBLIC,
         vec_bolsa_convocatorias_ejecutor_consulta,
         vec_bolsa_convocatorias_proyector_gobierno,
         vec_bolsa_convocatorias_registrador_atestacion,
         vec_bolsa_convocatorias_verificador_recibo;

COMMENT ON FUNCTION vec_bolsa_convocatorias.obtener_borrador_v1(
    text,jsonb,jsonb,bytea,bytea
) IS 'Lectura interna autorizada: consume decision PDP y auditoria antes de devolver el paquete KMS completo ya persistido por 000004. No fabrica AAD, DEK, perfil ni procedencia.';
COMMIT;
