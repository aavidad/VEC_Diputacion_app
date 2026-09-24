\set ON_ERROR_STOP on
-- Ejecutar como DBA sobre la base sintética desechable, después del UP.
DO $prueba$
DECLARE firma text := 'vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vec_personal_registrador_frontera'
                   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
       OR to_regprocedure(firma) IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid = 'vec_personal.auditoria_frontera_asignacion_dietas'::regclass
                       AND relrowsecurity AND relforcerowsecurity)
       OR (SELECT count(*) FROM pg_policy
            WHERE polrelid = 'vec_personal.auditoria_frontera_asignacion_dietas'::regclass) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_policy
            WHERE polrelid = 'vec_personal.auditoria_frontera_asignacion_dietas'::regclass
              AND polcmd = 'a' AND polqual IS NULL AND polwithcheck IS NOT NULL
              AND polroles = ARRAY[(SELECT oid FROM pg_roles
                                     WHERE rolname = 'vec_personal_propietario')])
       OR NOT has_function_privilege('vec_personal_registrador_frontera', firma, 'EXECUTE')
       OR has_function_privilege('vec_personal_ejecutor', firma, 'EXECUTE')
       OR EXISTS (SELECT 1 FROM pg_proc p,
                         LATERAL aclexplode(p.proacl) acl
                   WHERE p.oid = to_regprocedure(firma)
                     AND acl.grantee = 0 AND acl.privilege_type = 'EXECUTE')
       OR has_table_privilege('vec_personal_registrador_frontera', 'vec_personal.auditoria_frontera_asignacion_dietas', 'SELECT,INSERT,UPDATE,DELETE')
       OR has_sequence_privilege('vec_personal_registrador_frontera', 'vec_personal.auditoria_frontera_asignacion_dietas_evento_id_seq', 'USAGE')
       OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid = to_regprocedure(firma)
                       AND p.prosecdef AND p.provolatile = 'v')
    THEN RAISE EXCEPTION 'Personal 000012: estructura o ACL inesperadas'; END IF;
END
$prueba$;

CREATE ROLE vec_personal_auditoria_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_registrador_frontera TO vec_personal_auditoria_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_personal_auditoria_ajena NOLOGIN;
CREATE ROLE vec_personal_auditoria_extra LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_registrador_frontera TO vec_personal_auditoria_extra
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_personal_auditoria_ajena TO vec_personal_auditoria_extra
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
