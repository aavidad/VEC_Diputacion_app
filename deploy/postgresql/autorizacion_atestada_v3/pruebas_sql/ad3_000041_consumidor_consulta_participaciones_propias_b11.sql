\set ON_ERROR_STOP on
-- Regresión post-instalación B11; no emite material criptográfico ni datos.
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $prueba$
DECLARE
  propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
  bolsa oid := 'vec_bolsa_llamamientos_propietario'::regrole;
  f_consumo oid := 'vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
  f_revalidacion oid := 'vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
  f_interno oid := 'vec_autorizacion_atestada_v3.consumir_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
  f_reval_interno oid := 'vec_autorizacion_atestada_v3.revalidar_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF pg_catalog.length('registrar_consumo_participaciones_propias_b11_v3') >= 63
    OR pg_catalog.length('revalidar_consumo_participaciones_propias_b11_v3') >= 63 THEN
   RAISE EXCEPTION 'AD3-000041: identificador B11 excede NAMEDATALEN' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=ANY(ARRAY[f_consumo,f_revalidacion])
            AND (p.proowner<>propietario OR NOT p.prosecdef OR p.provolatile<>'v'))
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=ANY(ARRAY[f_interno,f_reval_interno])
               AND pg_get_functiondef(p.oid) NOT LIKE '%vec.bolsa.mi-bolsa.v1%')
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=ANY(ARRAY[f_interno,f_reval_interno])
               AND pg_get_functiondef(p.oid) NOT LIKE '%bolsa.candidato.participaciones.consultar%')
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=ANY(ARRAY[f_interno,f_reval_interno])
               AND pg_get_functiondef(p.oid) NOT LIKE '%consulta_posicion_propia_bolsa%')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.oid=ANY(ARRAY[f_consumo,f_revalidacion])
                 AND (a.privilege_type<>'EXECUTE' OR a.grantee NOT IN (propietario,bolsa) OR a.is_grantable))
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE p.oid=ANY(ARRAY[f_consumo,f_revalidacion]) AND a.privilege_type='EXECUTE')<>4 THEN
   RAISE EXCEPTION 'AD3-000041: firma, ligadura o ACL B11 divergente' USING ERRCODE='55000';
 END IF;
END $prueba$;
ROLLBACK;
