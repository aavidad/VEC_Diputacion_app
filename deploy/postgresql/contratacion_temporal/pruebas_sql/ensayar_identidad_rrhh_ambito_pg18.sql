-- Contrato estructural de Identidad2; se ejecuta antes y después de los casos.
DO $prueba$
DECLARE
    v_oid oid := pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)');
    v_hash text;
    v_acl jsonb;
BEGIN
    SELECT pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(p.prosrc, 'UTF8')), 'hex'),
           pg_catalog.to_jsonb(p.proacl)
      INTO v_hash, v_acl FROM pg_catalog.pg_proc p WHERE p.oid = v_oid;
    IF v_oid IS NULL
       OR v_hash IS DISTINCT FROM
          '3c6d7bd086f80dfeab8a94a3d6f4b4400b544a9ce793444b0303669f323444e4'
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',v_oid,'EXECUTE')
       OR pg_catalog.has_function_privilege('public',v_oid,'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh',v_oid,'EXECUTE')
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_consultor_rrhh_ambito',v_oid,'EXECUTE')
       OR pg_catalog.has_schema_privilege(
           'vec_contratacion_temporal_consultor_rrhh_ambito',
           'vec_identidad_sesiones_v1','USAGE')
       OR EXISTS (
           SELECT 1 FROM pg_catalog.aclexplode(
               (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=v_oid)) a
            WHERE a.privilege_type <> 'EXECUTE'
               OR a.grantee NOT IN (
                   'vec_identidad_sesiones_v1_propietario'::regrole,
                   'vec_contratacion_temporal_propietario'::regrole))
    THEN
        RAISE EXCEPTION 'Identidad2: contrato, hash o ACL incorrecta';
    END IF;
END
$prueba$;
