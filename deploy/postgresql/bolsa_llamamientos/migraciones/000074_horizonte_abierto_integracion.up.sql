-- La necesidad de sustitución o vacante puede terminar por una causa gobernada,
-- sin fecha conocida al abrir el llamamiento. Se conserva íntegro el canon
-- fechado y toda la historia de operaciones, auditoría y recibos existentes.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_llamamientos.necesidad_vigente_integracion_v2(
    p_necesidad jsonb, p_ahora timestamptz
) RETURNS boolean
LANGUAGE plpgsql STABLE SET search_path=pg_catalog AS $f$
DECLARE v_fin jsonb;
BEGIN
    IF jsonb_typeof(p_necesidad) IS DISTINCT FROM 'object' OR p_ahora IS NULL THEN
        RETURN false;
    END IF;
    v_fin := p_necesidad->'fin_previsto';
    IF jsonb_typeof(v_fin) = 'null' THEN
        RETURN coalesce(jsonb_typeof(p_necesidad->'causa_fin_clave') = 'string'
           AND p_necesidad->>'causa_fin_clave' ~ '^[a-z0-9][a-z0-9._-]{0,127}$', false);
    END IF;
    IF jsonb_typeof(v_fin) IS DISTINCT FROM 'string' OR p_necesidad ? 'causa_fin_clave' THEN
        RETURN false;
    END IF;
    RETURN p_ahora < (p_necesidad->>'fin_previsto')::timestamptz;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
    RETURN false;
END
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.necesidad_vigente_integracion_v2(jsonb,timestamptz) FROM PUBLIC;

-- Las ampliaciones anteriores del contrato incorporaron aceptación, continuidad
-- y expiración. Cambiar sólo las dos guardas de horizonte en su definición viva
-- preserva las demás ramas, su ACL y la historia instalada.
DO $cambio$
DECLARE v_def text; v_acl aclitem[]; v_antes text; v_despues text;
BEGIN
    SELECT pg_get_functiondef(p.oid), p.proacl INTO STRICT v_def, v_acl
      FROM pg_proc p
     WHERE p.oid = 'vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner = 'vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
    v_antes := $antes$v_ahora<(f#>>'{datos,Necesidad,fin_previsto}')::timestamptz$antes$;
    v_despues := $despues$vec_bolsa_llamamientos.necesidad_vigente_integracion_v2(f#>'{datos,Necesidad}',v_ahora)$despues$;
    IF (length(v_def)-length(replace(v_def,v_antes,'')))/length(v_antes) <> 2 THEN
        RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='PARO bolsa_guardas_fin: esperado=2, observado distinto';
    END IF;
    v_def := replace(v_def,v_antes,v_despues);
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='PARO bolsa_acl_integracion: esperado=preimagen, observado distinto';
    END IF;
END
$cambio$;
COMMIT;
