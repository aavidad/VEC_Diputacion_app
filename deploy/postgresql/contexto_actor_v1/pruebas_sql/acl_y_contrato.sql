SET search_path = pg_catalog;

DO $prueba$
DECLARE tablas_visibles integer; funciones_ejecutables integer;
BEGIN
    SELECT count(*) INTO tablas_visibles
     FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
     WHERE n.nspname='vec_contexto_actor_v1' AND c.relkind IN ('r','p','v','m')
       AND has_table_privilege(
             'vec_contexto_actor_v1_runtime',c.oid,
             'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN'
           );
    IF tablas_visibles <> 0 THEN
        RAISE EXCEPTION 'runtime conserva acceso a tablas';
    END IF;
    SELECT count(*) INTO funciones_ejecutables
      FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_contexto_actor_v1'
       AND has_function_privilege('vec_contexto_actor_v1_runtime',p.oid,'EXECUTE');
    IF funciones_ejecutables <> 3 THEN
        RAISE EXCEPTION 'superficie ejecutable inesperada: %',funciones_ejecutables;
    END IF;
    IF EXISTS (
       SELECT 1 FROM pg_catalog.pg_parameter_acl a
        WHERE pg_catalog.has_parameter_privilege(
                  'vec_contexto_actor_v1_runtime',a.parname,'SET'
              )
           OR pg_catalog.has_parameter_privilege(
                  'vec_contexto_actor_v1_runtime',a.parname,'ALTER SYSTEM'
              )
    ) THEN
       RAISE EXCEPTION 'runtime conserva ACL efectiva sobre parametros';
    END IF;
    IF EXISTS (
       SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_contexto_actor_v1' AND p.prosecdef
          AND p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    ) THEN
       RAISE EXCEPTION 'SECURITY DEFINER sin search_path cerrado';
    END IF;
END
$prueba$;
