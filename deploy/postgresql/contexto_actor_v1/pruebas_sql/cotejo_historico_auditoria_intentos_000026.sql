-- Ejecutar solo en clon, despues de CA26. No modifica historia.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL TIME ZONE 'UTC';
DO $test$
DECLARE f regprocedure := 'vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(text,text,text,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) <> 'vec_contexto_actor_v1_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
        WHERE p.oid=f AND a.grantee='vec_autorizacion_atestada_v3_propietario'::regrole
          AND a.privilege_type='EXECUTE' AND NOT a.is_grantable) <> 1
    OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
        WHERE p.oid=f AND a.grantee=0) <> 0
    OR has_function_privilege('vec_contexto_actor_v1_runtime',f,'EXECUTE') THEN
   RAISE EXCEPTION 'CA26 ACL o definidor inesperado';
 END IF;
 BEGIN
  PERFORM vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
    'rca_no-existe','0000000000000000000000000000000000000000000000000000000000000000',
    '0000000000000000000000000000000000000000000000000000000000000000',
    decode('7b7d','hex'));
  RAISE EXCEPTION 'CA26 acepto selector invalido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $test$;
ROLLBACK;
