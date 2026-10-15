\set ON_ERROR_STOP on
-- Solo en clon H tras instalar CT219 una vez. Nunca sobre la principal.
-- Sin material real, acredita definición, ACL y plan publicado; no afirma firma.
-- Para el negativo de punta a punta, dirección puede pasar
-- -v ct219_material_real_sql=/ruta/privada/negativos.sql. Ese archivo se
-- prepara desde una firma real y decisiones nuevas del flujo normal, entra
-- con el LOGIN CT auténtico, prueba v4 (no invoca v3 sin permiso), coteja
-- denegación y ausencia de firmas/recibos nuevos, y no lleva COMMIT ni roles
-- de imitación. Esta prueba envuelve el archivo privado en ROLLBACK.
\if :{?ct219_material_real_sql}
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $runtime$
BEGIN
 IF current_setting('role')<>'none' OR session_user<>current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'CT219: requiere LOGIN runtime CT real sin SET ROLE' USING ERRCODE='42501';
 END IF;
END $runtime$;
\i :ct219_material_real_sql
ROLLBACK;
\else
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
DO $estructura$
DECLARE
 v3 oid:=to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)');
 v2 oid:=to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)');
 v4 oid:=to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 total integer; roles_ok boolean; cargos_ok boolean; distintos boolean;
BEGIN
 IF v3 IS NULL OR v2 IS NULL OR v4 IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef(v3),'UTF8')),'hex')
       IS DISTINCT FROM '8f58548d20ef55314fa78e631cec10d0dd3a918e46e45de5b5b527758b6afb1a'
    OR encode(sha256(convert_to(pg_get_functiondef(v2),'UTF8')),'hex')
       IS DISTINCT FROM '49cf7bf45eb803baf87abee9556fe1417c61de4cf6daa8b0865f208e25ee6bd0'
    OR encode(sha256(convert_to(pg_get_functiondef(v4),'UTF8')),'hex')
       IS DISTINCT FROM '44b50bb77d26457a715eabbb101c9e65a69daa71ab2910a035ecafe8a9498a85'
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=v3
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole
       AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
                             'TimeZone=UTC','lock_timeout=2s']::text[]
       AND p.proacl=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[])
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',v3,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',v4,'EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'CT219: definición histórica, propietario o ACL divergente' USING ERRCODE='55000'; END IF;
 SELECT count(*),
        bool_and((e->'atributos'->>'rol_id' ~ '^ct_cargo_[a-z0-9_]{2,80}$') IS TRUE),
        bool_and((e->'atributos'->>'cargo_ref' ~ '^car_[A-Za-z0-9_-]{22,128}$') IS TRUE),
        bool_and(e->'atributos'->>'rol_id' IS DISTINCT FROM e->'atributos'->>'cargo_ref')
   INTO total,roles_ok,cargos_ok,distintos
   FROM (SELECT canonico_exacto FROM vec_catalogos_configurables.plan_firma_publicacion
          WHERE catalogo_id='ct.plan.firma.hz39.operativa' AND estado='publicado'
          ORDER BY version DESC LIMIT 1) p
   CROSS JOIN LATERAL jsonb_array_elements(convert_from(p.canonico_exacto,'UTF8')::jsonb->'entradas') e;
 IF total<>5 OR roles_ok IS NOT TRUE OR cargos_ok IS NOT TRUE OR distintos IS NOT TRUE THEN
  RAISE EXCEPTION 'CT219: plan operativo no distingue cinco roles y cargos' USING ERRCODE='55000';
 END IF;
END $estructura$;
ROLLBACK;
\echo 'CT219: definición/ACL/plan comprobados; negativo de ejecución pendiente de material real privado'
\endif
