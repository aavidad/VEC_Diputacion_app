\set ON_ERROR_STOP on
-- Identidad2: revalidación RRHH con el grupo nominal de ámbito.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:identidad:consulta_rrhh:v1', 0));

SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
DO $migracion$
DECLARE
    v_funcion oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)');
    v_antes record;
    v_despues record;
    v_dependencias_antes jsonb;
    v_dependencias_despues jsonb;
    v_origen text := $fragmento$g.rolname =
                  'vec_contratacion_temporal_consultor_rrhh'$fragmento$;
    v_destino text := $fragmento$g.rolname IN (
                  'vec_contratacion_temporal_consultor_rrhh',
                  'vec_contratacion_temporal_consultor_rrhh_ambito'
              )$fragmento$;
    v_sql text;
BEGIN
    IF current_user <> 'vec_identidad_sesiones_v1_propietario'
       OR v_funcion IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.rolname = 'vec_contratacion_temporal_consultor_rrhh_ambito'
              AND NOT r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreatedb
              AND NOT r.rolcreaterole AND NOT r.rolreplication
              AND NOT r.rolbypassrls)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.member =
              'vec_contratacion_temporal_consultor_rrhh_ambito'::regrole)
       OR pg_catalog.has_schema_privilege(
           'vec_contratacion_temporal_consultor_rrhh_ambito',
           'vec_identidad_sesiones_v1', 'USAGE')
    THEN
        RAISE EXCEPTION 'Identidad2: rol o función incompatible'
            USING ERRCODE = '55000';
    END IF;
    SELECT p.oid, p.prosrc, p.proacl, p.proowner, p.proconfig,
           p.prosecdef, p.provolatile, p.proparallel,
           pg_catalog.to_jsonb(p) - 'prosrc' AS metadata,
           pg_catalog.pg_get_functiondef(p.oid) AS definition
      INTO v_antes FROM pg_catalog.pg_proc p WHERE p.oid = v_funcion;
    IF v_antes.proowner <>
           'vec_identidad_sesiones_v1_propietario'::regrole
       OR v_antes.prosecdef IS NOT TRUE
       OR v_antes.provolatile <> 'v' OR v_antes.proparallel <> 'u'
       OR v_antes.proconfig IS DISTINCT FROM
          ARRAY['search_path=pg_catalog', 'lock_timeout=1s']::text[]
       OR pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(v_antes.prosrc, 'UTF8')), 'hex')
          IS DISTINCT FROM '69e0c769db76e8eee040334bba764f5374360f30b01c4d1d59db14899fce6c44'
       OR (pg_catalog.length(v_antes.definition) -
           pg_catalog.length(pg_catalog.replace(
               v_antes.definition, v_origen, '')))
          <> 3 * pg_catalog.length(v_origen)
       OR pg_catalog.strpos(v_antes.definition, v_destino) <> 0
       OR NOT pg_catalog.has_schema_privilege(
           'vec_contratacion_temporal_propietario',
           'vec_identidad_sesiones_v1', 'USAGE')
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           v_funcion, 'EXECUTE')
       OR pg_catalog.has_function_privilege('public', v_funcion, 'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh',
           v_funcion, 'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh_ambito',
           v_funcion, 'EXECUTE')
       OR EXISTS (
           SELECT 1 FROM pg_catalog.aclexplode(v_antes.proacl) a
            WHERE a.privilege_type <> 'EXECUTE'
               OR a.grantee NOT IN (
                   'vec_identidad_sesiones_v1_propietario'::regrole,
                   'vec_contratacion_temporal_propietario'::regrole))
    THEN
        RAISE EXCEPTION 'Identidad2: preimagen o ACL incompatible'
            USING ERRCODE = '55000';
    END IF;
    SELECT pg_catalog.coalesce(pg_catalog.jsonb_agg(
        pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
        d.refclassid,d.refobjid,d.refobjsubid,d.deptype), '[]'::jsonb)
      INTO v_dependencias_antes FROM pg_catalog.pg_depend d
     WHERE d.classid = 'pg_catalog.pg_proc'::regclass
       AND d.objid = v_funcion;
    v_sql := pg_catalog.replace(v_antes.definition, v_origen, v_destino);
    EXECUTE v_sql;
    SELECT p.oid, p.prosrc, p.proacl, p.proowner, p.proconfig,
           p.prosecdef, p.provolatile, p.proparallel,
           pg_catalog.to_jsonb(p) - 'prosrc' AS metadata,
           pg_catalog.pg_get_functiondef(p.oid) AS definition
      INTO STRICT v_despues FROM pg_catalog.pg_proc p
     WHERE p.oid = v_funcion;
    SELECT pg_catalog.coalesce(pg_catalog.jsonb_agg(
        pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
        d.refclassid,d.refobjid,d.refobjsubid,d.deptype), '[]'::jsonb)
      INTO v_dependencias_despues FROM pg_catalog.pg_depend d
     WHERE d.classid = 'pg_catalog.pg_proc'::regclass
       AND d.objid = v_funcion;
    IF pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(v_despues.prosrc, 'UTF8')), 'hex')
          IS DISTINCT FROM '3c6d7bd086f80dfeab8a94a3d6f4b4400b544a9ce793444b0303669f323444e4'
       OR v_despues.definition IS DISTINCT FROM v_sql
       OR v_despues.metadata IS DISTINCT FROM v_antes.metadata
       OR v_despues.proacl IS DISTINCT FROM v_antes.proacl
       OR v_despues.proowner IS DISTINCT FROM v_antes.proowner
       OR v_despues.proconfig IS DISTINCT FROM v_antes.proconfig
       OR v_despues.prosecdef IS DISTINCT FROM v_antes.prosecdef
       OR v_despues.provolatile IS DISTINCT FROM v_antes.provolatile
       OR v_despues.proparallel IS DISTINCT FROM v_antes.proparallel
       OR v_dependencias_despues IS DISTINCT FROM v_dependencias_antes
    THEN
        RAISE EXCEPTION 'Identidad2: postimagen o contrato alterado'
            USING ERRCODE = '55000';
    END IF;
END
$migracion$;
COMMIT;
