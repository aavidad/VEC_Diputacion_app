\set ON_ERROR_STOP on
-- Ejecución en clon sintético tras BIC6, AD213 y B84. No crea datos.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
DO $verificar$
DECLARE
 resumen regprocedure:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1(text,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 candidatos regprocedure:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_candidatos_rrhh_nominal_v1(text,text,text,text,text,text,integer,text,text[],boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 finalizar regprocedure:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.finalizar_barrido_rrhh_nominal_v1(text,text,text,text[])');
 importacion regprocedure:=pg_catalog.to_regprocedure('vec_bolsa_importacion_convoca.recuperar_filas_bolsa_rrhh_v1(text,text,integer[])');
 consumo regprocedure:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF resumen IS NULL OR candidatos IS NULL OR finalizar IS NULL OR importacion IS NULL OR consumo IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',resumen,'EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',candidatos,'EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',finalizar,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',importacion,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',consumo,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
         'vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)','EXECUTE')
    OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
         'vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',importacion,'EXECUTE')
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',consumo,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
         WHERE n.nspname='vec_bolsa_importacion_convoca'
           AND pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',p.oid,'EXECUTE'))<>1
 THEN RAISE EXCEPTION 'B84: ACL o funciones incompatibles' USING ERRCODE='42501'; END IF;
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
            WHERE p.oid IN (resumen,candidatos,finalizar,importacion,consumo) AND a.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                 WHERE p.oid IN (resumen,candidatos,finalizar,importacion,consumo)
                   AND (NOT p.prosecdef OR p.proconfig IS NULL OR p.proowner IS NULL
                     OR NOT EXISTS (SELECT 1 FROM pg_catalog.unnest(p.proconfig) AS x(valor)
                        WHERE pg_catalog.regexp_replace(x.valor,'[[:space:]]','','g')='search_path=pg_catalog,pg_temp')))
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.barrido_rrhh_tx)
 THEN RAISE EXCEPTION 'B84: función abierta o barrido persistido' USING ERRCODE='42501'; END IF;
END $verificar$;
ROLLBACK;
