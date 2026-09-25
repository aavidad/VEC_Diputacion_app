\set ON_ERROR_STOP on
-- AD3-81: USAGE del esquema AD3 para el ejecutor de Dietas.
-- AD3-50 concede a vec_dietas_ejecutor EXECUTE sobre la fachada
-- registrar_y_consumir_acceso_rutas_dietas_v3_atestada, cuya guarda exige que
-- la sesión sea miembro exacto de ese grupo, pero no le concede USAGE del
-- esquema: la llamada cualificada desde el adaptador Go de rutas
-- (internal/modules/dietas/adapters/postgres/acceso_rutas.go, pool
-- `dsn_consumo`) falla con 42501 «permission denied for schema». Solo se
-- añade USAGE: ninguna otra función, tabla, columna ni tipo del esquema queda
-- al alcance del ejecutor, porque todas revocan PUBLIC y no le conceden nada;
-- la precondición lo exige y la postcondición lo comprueba sobre el estado
-- efectivo (has_*_privilege) antes del COMMIT.
-- No reaplicar: se detiene si el USAGE directo ya existe.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000081',0));

DO $pre$
DECLARE
 f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 e oid:=to_regrole('vec_dietas_ejecutor');
 s oid:=to_regnamespace('vec_autorizacion_atestada_v3');
BEGIN
 -- Existencia primero: sin rol, esquema o fachada, 55000 y no 42704.
 IF e IS NULL OR s IS NULL OR f IS NULL THEN
  RAISE EXCEPTION 'AD3-81: falta el ejecutor Dietas, el esquema AD3 o la fachada de rutas AD3-50' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
                WHERE p.oid=f AND a.grantee=e
                  AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    -- PUBLIC en funciones del esquema, explícito o por omisión.
    OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.pronamespace=s AND a.grantee=0)
    -- El ejecutor, en cualquier otra función del esquema.
    OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
               WHERE p.pronamespace=s AND a.grantee=e AND p.oid<>f)
    -- Relaciones y columnas del esquema a favor del ejecutor o PUBLIC.
    OR EXISTS (SELECT 1 FROM pg_class c, aclexplode(c.relacl) a
               WHERE c.relnamespace=s AND a.grantee IN (0,e))
    OR EXISTS (SELECT 1 FROM pg_attribute x JOIN pg_class c ON c.oid=x.attrelid,
                    aclexplode(x.attacl) a
               WHERE c.relnamespace=s AND a.grantee IN (0,e))
    -- Tipos con ACL explícita a favor del ejecutor o PUBLIC.
    OR EXISTS (SELECT 1 FROM pg_type t, aclexplode(t.typacl) a
               WHERE t.typnamespace=s AND a.grantee IN (0,e))
    -- Privilegios por defecto que darían al ejecutor o PUBLIC objetos futuros
    -- del esquema (en el esquema o globales a favor del ejecutor).
    OR EXISTS (SELECT 1 FROM pg_default_acl d, aclexplode(d.defaclacl) a
               WHERE ((d.defaclnamespace=s AND a.grantee IN (0,e))
                   OR (d.defaclnamespace=0 AND a.grantee=e)))
    -- Esquema: ni el ejecutor ni PUBLIC tienen ya privilegio alguno.
    OR EXISTS (SELECT 1 FROM pg_namespace n, aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
               WHERE n.oid=s AND a.grantee IN (0,e)) THEN
  RAISE EXCEPTION 'AD3-81: preimagen AD3-50 incompatible o ya instalada' USING ERRCODE='55000';
 END IF;
END $pre$;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_dietas_ejecutor;

-- Postcondición sobre el estado efectivo (incluye PUBLIC y herencia).
DO $post$
DECLARE
 f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 e oid:=to_regrole('vec_dietas_ejecutor');
 s oid:=to_regnamespace('vec_autorizacion_atestada_v3');
BEGIN
 IF NOT has_schema_privilege(e,s,'USAGE')
    OR has_schema_privilege(e,s,'USAGE WITH GRANT OPTION')
    OR has_schema_privilege(e,s,'CREATE')
    OR (SELECT count(*) FROM pg_namespace n, aclexplode(n.nspacl) a
        WHERE n.oid=s AND a.grantee=e)<>1
    OR EXISTS (SELECT 1 FROM pg_namespace n, aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
               WHERE n.oid=s AND a.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.pronamespace=s AND a.grantee=0)
    OR (SELECT array_agg(p.oid) FROM pg_proc p
        WHERE p.pronamespace=s AND has_function_privilege(e,p.oid,'EXECUTE'))
       IS DISTINCT FROM ARRAY[f::oid]
    OR has_function_privilege(e,f,'EXECUTE WITH GRANT OPTION')
    -- CASE fija el orden: has_sequence_privilege solo admite secuencias.
    OR EXISTS (SELECT 1 FROM pg_class c
               WHERE c.relnamespace=s AND CASE
                 WHEN c.relkind='S' THEN has_sequence_privilege(e,c.oid,'USAGE,SELECT,UPDATE')
                 WHEN c.relkind IN ('r','p','v','m','f') THEN
                   has_table_privilege(e,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
                   OR has_any_column_privilege(e,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')
                 ELSE false END) THEN
  RAISE EXCEPTION 'AD3-81: postcondición de privilegios efectivos del ejecutor Dietas fallida' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
