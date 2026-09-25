\set ON_ERROR_STOP on
-- AD3-81 DOWN: retira solo el USAGE del esquema AD3 al ejecutor de Dietas.
-- No toca decisiones, consumos ni auditoría; la fachada de rutas deja de ser
-- alcanzable (42501) hasta reinstalar. Se detiene si el ejecutor tiene EXECUTE
-- sobre otra función del esquema (el DOWN la dejaría inalcanzable sin que
-- ninguna migración lo declare) y comprueba que el resto de la ACL del
-- esquema queda idéntica.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000081',0));
DO $pre$
DECLARE
 f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 e oid:=to_regrole('vec_dietas_ejecutor');
 s oid:=to_regnamespace('vec_autorizacion_atestada_v3');
BEGIN
 IF e IS NULL OR s IS NULL OR NOT EXISTS (
      SELECT 1 FROM pg_namespace n, aclexplode(n.nspacl) a
      WHERE n.oid=s AND a.grantee=e AND a.privilege_type='USAGE') THEN
  RAISE EXCEPTION 'AD3-81: no instalada' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p
            WHERE p.pronamespace=s AND p.oid IS DISTINCT FROM f::oid
              AND has_function_privilege(e,p.oid,'EXECUTE')) THEN
  RAISE EXCEPTION 'AD3-81: el ejecutor Dietas tiene EXECUTE sobre otra función del esquema AD3; el DOWN la dejaría inalcanzable' USING ERRCODE='55000';
 END IF;
 -- Preimagen de la ACL del esquema, local a la transacción.
 PERFORM set_config('vec.ad3_81_nspacl',
   (SELECT coalesce(n.nspacl,acldefault('n',n.nspowner))::text FROM pg_namespace n WHERE n.oid=s),true);
END $pre$;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_dietas_ejecutor;
DO $post$
DECLARE
 e oid:=to_regrole('vec_dietas_ejecutor');
 s oid:=to_regnamespace('vec_autorizacion_atestada_v3');
 previa aclitem[]:=current_setting('vec.ad3_81_nspacl')::aclitem[];
 actual aclitem[];
BEGIN
 SELECT coalesce(n.nspacl,acldefault('n',n.nspowner)) INTO STRICT actual FROM pg_namespace n WHERE n.oid=s;
 IF has_schema_privilege(e,s,'USAGE')
    OR EXISTS (SELECT 1 FROM aclexplode(actual) a WHERE a.grantee=e)
    OR EXISTS ((SELECT a.grantor,a.grantee,a.privilege_type,a.is_grantable
                  FROM aclexplode(previa) a WHERE a.grantee<>e)
               EXCEPT
               (SELECT a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM aclexplode(actual) a))
    OR EXISTS ((SELECT a.grantor,a.grantee,a.privilege_type,a.is_grantable FROM aclexplode(actual) a)
               EXCEPT
               (SELECT a.grantor,a.grantee,a.privilege_type,a.is_grantable
                  FROM aclexplode(previa) a WHERE a.grantee<>e)) THEN
  RAISE EXCEPTION 'AD3-81: la retirada alteró otras entradas de la ACL del esquema AD3' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
